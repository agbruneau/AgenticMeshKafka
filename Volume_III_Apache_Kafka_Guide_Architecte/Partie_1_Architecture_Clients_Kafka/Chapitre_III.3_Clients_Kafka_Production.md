# Chapitre III.3

## CLIENTS KAFKA ET PRODUCTION DE MESSAGES

---

> *« La qualité d'un système distribué se mesure à la robustesse de ses producteurs autant qu'à la fiabilité de son infrastructure. »*
>
> — Jay Kreps, Co-créateur d'Apache Kafka

---

Les chapitres précédents ont établi les fondations architecturales de Kafka : la perspective stratégique de l'architecte et les mécanismes internes du cluster. Ce chapitre se concentre sur le côté client de l'équation — comment les applications publient des messages vers Kafka de manière efficace, fiable et performante.

Le producteur Kafka (producer) est souvent perçu comme un composant simple : on envoie un message, Kafka le stocke. Cette simplicité apparente masque une complexité considérable. Le producteur est responsable de la sérialisation des données, du partitionnement, du batching, de la compression, de la gestion des erreurs et des retry, de la confirmation des écritures, et de nombreuses autres préoccupations qui déterminent la fiabilité et la performance du système global.

L'architecte qui comprend en profondeur le fonctionnement du producteur sera capable de concevoir des applications robustes, de diagnostiquer les problèmes de performance, et de faire des choix éclairés sur les compromis entre latence, débit et durabilité. Ce chapitre fournit cette compréhension, en combinant la théorie nécessaire avec des recommandations pratiques issues de l'expérience terrain.

Nous explorerons successivement l'anatomie interne du producteur, les garanties de livraison, les stratégies de partitionnement, la sérialisation et la gestion des schémas, l'optimisation des performances, et les bonnes pratiques architecturales. Chaque section est conçue pour fournir à l'architecte les connaissances nécessaires pour prendre des décisions éclairées.

---

## III.3.1 L'Anatomie d'un Producer Kafka

### Architecture Interne du Producteur

Le producteur Kafka n'est pas un simple client qui envoie des requêtes au serveur. C'est un composant sophistiqué avec sa propre architecture interne, optimisée pour le débit et la fiabilité. Comprendre cette architecture est essentiel pour configurer correctement le producteur et diagnostiquer les problèmes.

Le producteur Kafka se compose de plusieurs composants internes qui travaillent ensemble pour transformer un appel `send()` en messages persistés sur le cluster.

**Le thread principal (application thread).** C'est le thread de l'application qui appelle la méthode `send()`. Ce thread est responsable de la sérialisation du message (clé et valeur), du calcul de la partition cible, et de l'ajout du message au buffer d'accumulation. Le thread principal n'attend pas que le message soit effectivement envoyé au broker — il retourne immédiatement (ou après accumulation selon la configuration).

**L'accumulateur de records (RecordAccumulator).** Les messages ne sont pas envoyés individuellement au broker. Ils sont accumulés dans des buffers par partition, formant des batches. L'accumulateur maintient une file de batches pour chaque partition du cluster. Cette structure permet le batching efficace et l'envoi groupé.

**Le thread sender.** Un thread séparé, appelé « sender », est responsable de l'envoi effectif des batches vers les brokers. Ce thread surveille les batches prêts (qui ont atteint leur taille maximale ou leur temps d'attente maximal), établit les connexions avec les brokers leaders appropriés, envoie les requêtes de production, et gère les réponses et les retry en cas d'erreur.

**Le pool de buffers (BufferPool).** Pour éviter les allocations mémoire répétées, le producteur maintient un pool de buffers réutilisables. La taille totale de ce pool est contrôlée par `buffer.memory` (défaut 32 Mo). Si le pool est épuisé (trop de messages en attente d'envoi), les appels `send()` peuvent bloquer ou échouer selon la configuration.

> **Définition formelle**
>
> Le **RecordAccumulator** est le composant central du producteur Kafka responsable du batching. Il maintient une structure de données `Map<TopicPartition, Deque<ProducerBatch>>` où chaque partition possède une file de batches en cours d'accumulation. Le premier batch de chaque file est le batch « courant » qui reçoit les nouveaux messages ; les batches suivants sont en attente d'envoi.

### Cycle de Vie d'un Message

Pour comprendre le comportement du producteur, suivons le parcours d'un message depuis l'appel `send()` jusqu'à la confirmation de persistance. Cette compréhension détaillée permet de diagnostiquer les problèmes et d'optimiser les performances.

**Étape 1 : Sérialisation.** L'application fournit une clé et une valeur sous forme d'objets Java (ou du langage utilisé). Le producteur utilise les sérialiseurs configurés (`key.serializer` et `value.serializer`) pour convertir ces objets en tableaux d'octets. Si la sérialisation échoue, une exception est levée immédiatement.

La sérialisation est synchrone et s'exécute dans le thread appelant. Un sérialiseur lent ou une sérialisation de gros objets peut impacter la latence perçue par l'application. Pour les objets complexes, considérer la mise en cache des résultats de sérialisation si le même objet est envoyé plusieurs fois.

**Étape 2 : Partitionnement.** Le producteur détermine la partition cible. Si une partition est explicitement spécifiée dans l'appel `send()`, elle est utilisée. Sinon, si une clé est fournie, le hash de la clé détermine la partition. Si aucune clé n'est fournie, le partitionneur par défaut utilise un algorithme round-robin avec « sticky partitioning » pour optimiser le batching.

Le calcul de partition inclut une vérification des métadonnées du cluster. Si les métadonnées sont périmées (leader changé, partition indisponible), une requête de rafraîchissement est déclenchée. Ce rafraîchissement peut ajouter de la latence lors des premiers envois ou après des changements de topologie.

**Étape 3 : Validation et interception.** Avant l'accumulation, le message peut passer par des intercepteurs configurés (`interceptor.classes`). Les intercepteurs permettent d'ajouter des métadonnées (headers), de modifier le message, ou de journaliser. Ils s'exécutent dans le thread appelant et doivent être rapides.

La taille du message est également validée contre `max.request.size`. Un message trop grand est rejeté immédiatement avec une `RecordTooLargeException`.

**Étape 4 : Accumulation.** Le message sérialisé est ajouté au batch courant de la partition cible dans le RecordAccumulator. Si le batch courant est plein, un nouveau batch est créé. Si la mémoire du buffer pool est épuisée, l'appel peut bloquer (jusqu'à `max.block.ms`) ou échouer.

L'accumulation retourne un `Future<RecordMetadata>` et un `FutureRecordMetadata` interne qui sera complété quand la réponse du broker arrivera. Le callback fourni à `send()` est attaché à ce future.

**Étape 5 : Envoi.** Le thread sender surveille les batches prêts à être envoyés. Un batch est considéré prêt si sa taille atteint `batch.size`, ou si son temps d'attente dépasse `linger.ms`, ou si la mémoire est sous pression. Le sender regroupe les batches par broker leader et envoie des requêtes de production.

Le sender maintient des connexions persistantes vers les brokers. La configuration `connections.max.idle.ms` contrôle la fermeture des connexions inactives. Des connexions qui se ferment et se rouvrent fréquemment peuvent indiquer un problème de configuration ou de réseau.

**Étape 6 : Confirmation et callback.** Le broker traite la requête, écrit les messages sur disque, et répond selon la configuration `acks`. Le sender reçoit la réponse et invoque les callbacks associés aux messages (succès ou erreur). En cas d'erreur retriable, le batch peut être renvoyé.

Les callbacks sont exécutés dans le thread sender. Un callback lent bloque le traitement des autres réponses. Pour un traitement asynchrone lourd, le callback devrait déléguer à un thread pool séparé.

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Application Thread                           │
│  ┌──────────┐    ┌──────────────┐    ┌─────────────────────────┐   │
│  │  send()  │───▶│ Serializers  │───▶│     Partitioner         │   │
│  └──────────┘    └──────────────┘    └───────────┬─────────────┘   │
└──────────────────────────────────────────────────┼──────────────────┘
                                                   │
                                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       RecordAccumulator                             │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  Partition 0: [Batch 1] [Batch 2] ...                       │   │
│  │  Partition 1: [Batch 1] ...                                 │   │
│  │  Partition 2: [Batch 1] [Batch 2] [Batch 3] ...             │   │
│  └─────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────┬──────────────────┘
                                                   │
                                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Sender Thread                               │
│  ┌──────────────┐    ┌─────────────┐    ┌────────────────────┐     │
│  │ Batch Ready? │───▶│ Group by    │───▶│  Send to Brokers   │     │
│  │              │    │ Broker      │    │                    │     │
│  └──────────────┘    └─────────────┘    └─────────┬──────────┘     │
└───────────────────────────────────────────────────┼─────────────────┘
                                                    │
                                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Kafka Brokers                               │
│                    (Receive, Write, Respond)                        │
└─────────────────────────────────────────────────────────────────────┘
```

### Configuration Fondamentale

Les paramètres de configuration du producteur contrôlent chaque aspect de son comportement. L'architecte doit comprendre les paramètres clés et leurs interactions.

**`bootstrap.servers`** : Liste des brokers pour la découverte initiale du cluster. Le producteur contacte ces brokers pour obtenir les métadonnées du cluster (liste complète des brokers, leaders des partitions). Il n'est pas nécessaire de lister tous les brokers — quelques-uns suffisent pour la découverte.

**`key.serializer` et `value.serializer`** : Classes responsables de la conversion des objets en octets. Les sérialiseurs standards incluent `StringSerializer`, `ByteArraySerializer`, `IntegerSerializer`. Pour les formats complexes (Avro, Protobuf), des sérialiseurs spécifiques sont nécessaires.

**`acks`** : Niveau de confirmation attendu du broker. Cette configuration est cruciale pour le compromis durabilité/performance et sera détaillée dans la section suivante.

**`buffer.memory`** (défaut 32 Mo) : Mémoire totale disponible pour l'accumulation des messages. Si cette mémoire est épuisée, les appels `send()` bloquent.

**`batch.size`** (défaut 16 Ko) : Taille cible d'un batch en octets. Des batches plus grands améliorent le débit mais augmentent la latence.

**`linger.ms`** (défaut 0) : Temps d'attente avant d'envoyer un batch incomplet. Une valeur de 0 signifie envoi immédiat ; une valeur plus haute permet plus de batching.

**`compression.type`** (défaut none) : Algorithme de compression (none, gzip, snappy, lz4, zstd).

**`retries`** (défaut 2147483647 en Kafka 2.1+) : Nombre de tentatives en cas d'erreur retriable.

**`max.in.flight.requests.per.connection`** (défaut 5) : Nombre maximal de requêtes non confirmées par connexion. Impacte l'ordre des messages en cas de retry.

> **Note de terrain**
>
> *Contexte* : Déploiement initial d'un producteur Kafka avec configuration par défaut.
>
> *Observation* : Latence de production très variable — certains messages sont confirmés en 2ms, d'autres en 200ms.
>
> *Diagnostic* : Avec `linger.ms=0` (défaut), chaque message est envoyé dès qu'il est prêt, sans attendre de former un batch. Les messages arrivant seuls subissent le coût complet d'un aller-retour réseau. Les messages arrivant en rafale bénéficient du batching naturel.
>
> *Solution* : Configuration `linger.ms=5` pour permettre l'accumulation de messages pendant 5ms avant envoi. La latence maximale augmente de 5ms mais devient prévisible, et le débit global s'améliore significativement grâce au batching.
>
> *Leçon* : Les valeurs par défaut sont conservatrices. L'ajustement de `linger.ms` et `batch.size` est souvent le premier levier d'optimisation.

### Gestion Asynchrone et Callbacks

L'API de production Kafka est fondamentalement asynchrone. L'appel `send()` retourne immédiatement un `Future<RecordMetadata>` sans attendre la confirmation du broker. Cette conception permet un débit élevé mais requiert une gestion appropriée des résultats.

**Mode fire-and-forget.** L'application appelle `send()` et ignore le résultat. Simple mais dangereux — les erreurs passent inaperçues. Ce mode n'est acceptable que pour les données non critiques (métriques, logs) où la perte occasionnelle est tolérable.

**Mode synchrone.** L'application appelle `send().get()` pour bloquer jusqu'à la confirmation. Garantit la détection des erreurs mais limite sévèrement le débit (une seule requête en vol à la fois). Utile pour les tests ou les cas très critiques où le débit n'est pas une préoccupation.

**Mode asynchrone avec callback.** L'application fournit un callback qui sera invoqué à la réception de la réponse. C'est le mode recommandé — il combine le débit élevé de l'asynchrone avec la gestion des erreurs.

```java
producer.send(record, (metadata, exception) -> {
    if (exception != null) {
        // Gestion de l'erreur : log, retry applicatif, alerte
        logger.error("Échec de production", exception);
        errorHandler.handle(record, exception);
    } else {
        // Succès : le message est persisté
        logger.debug("Message envoyé à partition {} offset {}", 
                     metadata.partition(), metadata.offset());
    }
});
```

**Considérations sur les callbacks.** Les callbacks sont exécutés dans le thread sender, pas dans le thread applicatif. Un callback lent bloque l'envoi des autres messages. Les callbacks doivent être rapides et non-bloquants — si un traitement lourd est nécessaire, le déléguer à un autre thread ou une file.

### Intercepteurs de Production

Les **intercepteurs** permettent d'injecter une logique transversale dans le cycle de production sans modifier le code applicatif. Ils sont configurés via `interceptor.classes` et implémentent l'interface `ProducerInterceptor`.

**Cas d'usage des intercepteurs :**

*Traçage distribué.* Ajouter automatiquement des headers de corrélation (trace ID, span ID) à chaque message pour permettre le suivi de bout en bout dans un système distribué.

```java
public class TracingInterceptor implements ProducerInterceptor<String, String> {
    @Override
    public ProducerRecord<String, String> onSend(ProducerRecord<String, String> record) {
        String traceId = TraceContext.current().traceId();
        record.headers().add("X-Trace-Id", traceId.getBytes());
        return record;
    }
    
    @Override
    public void onAcknowledgement(RecordMetadata metadata, Exception exception) {
        // Log ou métriques
    }
}
```

*Métriques applicatives.* Collecter des métriques personnalisées sur les messages produits : compteurs par type de message, histogrammes de taille, etc.

*Validation.* Valider les messages avant envoi (bien que la validation soit généralement mieux placée au niveau applicatif).

*Chiffrement.* Chiffrer le contenu des messages au niveau applicatif avant envoi (pour un chiffrement de bout en bout, au-delà du TLS transport).

**Chaîne d'intercepteurs.** Plusieurs intercepteurs peuvent être configurés et s'exécutent dans l'ordre de configuration. Chaque intercepteur reçoit le record potentiellement modifié par l'intercepteur précédent.

**Précautions.** Les intercepteurs s'exécutent dans le chemin critique de la production. Un intercepteur lent ou qui lève une exception impacte tous les messages. Les intercepteurs doivent être robustes, rapides, et gérer leurs propres erreurs sans les propager.

### Utilisation des Headers

Les **headers** Kafka permettent d'attacher des métadonnées aux messages sans modifier la valeur du message. Introduits dans Kafka 0.11, ils sont devenus essentiels pour les patterns modernes.

**Cas d'usage des headers :**

*Métadonnées de routage.* Indiquer le type de message, la version du schéma, la source du message.

*Traçage.* Propager les identifiants de corrélation pour le traçage distribué.

*Timestamps applicatifs.* Ajouter des timestamps métier distincts du timestamp Kafka.

*Informations de sécurité.* Propager l'identité de l'émetteur, les claims JWT, les signatures.

```java
ProducerRecord<String, String> record = new ProducerRecord<>("orders", key, value);
record.headers().add("X-Message-Type", "OrderCreated".getBytes());
record.headers().add("X-Schema-Version", "2".getBytes());
record.headers().add("X-Source-System", "checkout-service".getBytes());
record.headers().add("X-Correlation-Id", correlationId.getBytes());
producer.send(record);
```

**Bonnes pratiques pour les headers :**

- Préfixer les headers personnalisés (ex: `X-` ou un namespace d'organisation) pour éviter les conflits.
- Garder les headers légers — ils sont inclus dans chaque message et consomment de l'espace.
- Documenter les headers attendus dans le contrat du topic.
- Les headers ne participent pas au partitionnement — seule la clé détermine la partition.

---

## III.3.2 Garanties Fondamentales de Production

### Le Spectre des Garanties de Livraison

Les systèmes de messagerie offrent traditionnellement trois niveaux de garanties de livraison. Comprendre ces garanties et comment Kafka les implémente est essentiel pour concevoir des systèmes fiables.

**At-most-once (au plus une fois).** Chaque message est livré zéro ou une fois. La perte de messages est possible, mais les duplications sont impossibles. C'est la garantie la plus faible, obtenue quand le producteur n'attend pas de confirmation (`acks=0`) ou quand il ne réessaie pas après une erreur.

**At-least-once (au moins une fois).** Chaque message est livré une ou plusieurs fois. La perte de messages est évitée, mais des duplications sont possibles. C'est la garantie par défaut de Kafka avec `acks=1` ou `acks=all` et les retries activés. Si un message est envoyé mais que la confirmation est perdue, le retry créera un duplicata.

**Exactly-once (exactement une fois).** Chaque message est livré exactement une fois. Ni perte ni duplication. C'est la garantie la plus forte, disponible dans Kafka via les producteurs idempotents et les transactions.

> **Définition formelle**
>
> La garantie de livraison **exactly-once** dans Kafka est implémentée par deux mécanismes complémentaires :
> 1. **L'idempotence du producteur** : Le broker détecte et élimine les duplicatas causés par les retries du producteur.
> 2. **Les transactions** : Un ensemble de messages peut être écrit atomiquement sur plusieurs partitions, avec garantie de tout-ou-rien.

### Configuration `acks` : Le Compromis Fondamental

Le paramètre `acks` du producteur détermine le niveau de confirmation attendu avant de considérer un message comme « envoyé avec succès ». Ce paramètre est le levier principal du compromis durabilité/latence/débit.

**`acks=0` : Fire-and-forget total.** Le producteur n'attend aucune confirmation. Dès que le message est envoyé sur le réseau, il est considéré comme réussi. Le producteur ne sait pas si le broker a reçu le message, encore moins s'il l'a persisté.

*Avantages* : Latence minimale, débit maximal.
*Inconvénients* : Perte de messages invisible, aucune garantie de durabilité.
*Cas d'usage* : Métriques haute fréquence où la perte occasionnelle est acceptable.

**`acks=1` : Confirmation du leader.** Le producteur attend que le leader de la partition confirme l'écriture. Le message est persisté sur le disque du leader (ou au moins dans son buffer d'écriture).

*Avantages* : Bon compromis latence/durabilité pour la majorité des cas.
*Inconvénients* : Si le leader crashe avant réplication, les messages confirmés peuvent être perdus.
*Cas d'usage* : La plupart des applications de production.

**`acks=all` (ou `acks=-1`) : Confirmation de tous les ISR.** Le producteur attend que tous les réplicas in-sync (ISR) confirment l'écriture. Le message est répliqué sur tous les réplicas synchronisés avant confirmation.

*Avantages* : Durabilité maximale, survie à la perte du leader.
*Inconvénients* : Latence plus élevée (attente de la réplication), débit potentiellement réduit.
*Cas d'usage* : Données critiques (transactions financières, événements d'audit).

| Configuration | Durabilité | Latence | Débit | Cas d'usage |
|---------------|------------|---------|-------|-------------|
| `acks=0` | Aucune | ~0ms | Maximal | Métriques non critiques |
| `acks=1` | Leader seul | ~2-10ms | Élevé | Majorité des cas |
| `acks=all` | Tous les ISR | ~10-50ms | Modéré | Données critiques |

### Idempotence du Producteur

L'**idempotence** garantit que les retries du producteur ne créent pas de duplicatas. Un producteur idempotent peut réessayer l'envoi d'un message sans risquer de le dupliquer côté broker.

**Mécanisme.** Chaque producteur idempotent reçoit un identifiant unique (Producer ID ou PID) lors de son initialisation. Chaque message envoyé inclut un numéro de séquence par partition. Le broker maintient un état par PID+partition et rejette les messages avec un numéro de séquence déjà vu.

**Activation.** L'idempotence est activée par `enable.idempotence=true`. À partir de Kafka 3.0, elle est activée par défaut. L'idempotence requiert `acks=all`, `retries > 0`, et `max.in.flight.requests.per.connection ≤ 5`. Si ces conditions ne sont pas remplies et que l'idempotence est explicitement demandée, une exception est levée.

**Portée.** L'idempotence est garantie par session producteur. Si le producteur redémarre (nouveau PID), les garanties ne s'appliquent pas aux messages de la session précédente. Pour une idempotence cross-session, les transactions sont nécessaires.

> **Exemple concret**
>
> *Scénario* : Un producteur envoie le message M1 avec séquence 42 vers la partition P0. Le broker reçoit et persiste M1. La confirmation est perdue sur le réseau.
>
> *Sans idempotence* : Le producteur réessaie. Le broker reçoit à nouveau M1 et le persiste — M1 existe maintenant en double dans P0.
>
> *Avec idempotence* : Le producteur réessaie avec le même PID et séquence 42. Le broker détecte que la séquence 42 a déjà été traitée pour ce PID et cette partition. Il retourne une confirmation sans dupliquer le message.

### Transactions Kafka

Les **transactions** étendent l'idempotence pour permettre l'écriture atomique de messages sur plusieurs partitions. Un ensemble de messages est soit entièrement visible, soit entièrement invisible pour les consommateurs.

**Cas d'usage principal : exactly-once stream processing.** Dans un pipeline de traitement de flux, une application lit des messages d'un topic source, les transforme, et écrit les résultats vers un topic destination. Avec les transactions, la lecture, la transformation et l'écriture peuvent être atomiques — si l'application crashe au milieu, soit tout est validé, soit rien ne l'est.

**Architecture transactionnelle.** Le producteur transactionnel interagit avec un coordinateur de transactions hébergé sur l'un des brokers. Ce coordinateur maintient l'état des transactions dans un topic interne (`__transaction_state`). Le flux typique est :

1. Le producteur s'initialise avec `initTransactions()`, récupérant ou créant un epoch pour son `transactional.id`.
2. `beginTransaction()` démarre une nouvelle transaction.
3. Les appels `send()` envoient les messages mais ils ne sont pas encore visibles pour les consommateurs `read_committed`.
4. `sendOffsetsToTransaction()` enregistre les offsets consommés comme partie de la transaction.
5. `commitTransaction()` finalise la transaction — tous les messages deviennent visibles atomiquement.
6. En cas d'erreur, `abortTransaction()` annule tous les messages de la transaction.

**Activation.** Les transactions sont activées par `transactional.id`, un identifiant stable qui survit aux redémarrages du producteur. Le producteur doit appeler `initTransactions()` au démarrage, puis utiliser `beginTransaction()`, `send()`, `sendOffsetsToTransaction()` (pour commiter les offsets consommés), et `commitTransaction()` ou `abortTransaction()`.

```java
Properties props = new Properties();
props.put("transactional.id", "order-processor-1");
props.put("enable.idempotence", "true");
// ... autres configs

KafkaProducer<String, String> producer = new KafkaProducer<>(props);
producer.initTransactions();

try {
    producer.beginTransaction();
    
    // Envoyer plusieurs messages atomiquement
    producer.send(new ProducerRecord<>("orders-processed", key1, value1));
    producer.send(new ProducerRecord<>("notifications", key2, value2));
    
    // Commiter les offsets consommés dans la même transaction
    producer.sendOffsetsToTransaction(offsets, consumerGroupId);
    
    producer.commitTransaction();
} catch (Exception e) {
    producer.abortTransaction();
    throw e;
}
```

**Isolation côté consommateur.** Par défaut (`isolation.level=read_uncommitted`), les consommateurs voient tous les messages, y compris ceux de transactions non encore commitées. Avec `isolation.level=read_committed`, les consommateurs ne voient que les messages de transactions commitées. Les messages de transactions en cours ou abandonnées sont filtrés.

**Gestion des epochs.** Chaque `transactional.id` a un epoch (compteur) qui s'incrémente à chaque appel `initTransactions()`. Si deux producteurs utilisent le même `transactional.id`, le second « fence » le premier — les messages du premier sont rejetés. Ce mécanisme évite les duplicatas en cas de redémarrage.

**Overhead et limites.** Les transactions ajoutent une latence (coordination avec le transaction coordinator, écriture dans `__transaction_state`) et une complexité. Le timeout de transaction (`transaction.timeout.ms`, défaut 60 secondes) limite la durée d'une transaction. Les transactions très longues (nombreux messages) peuvent impacter les performances.

**Bonnes pratiques transactionnelles :**

- Utiliser des `transactional.id` stables et uniques par instance logique de producteur.
- Garder les transactions courtes — quelques secondes maximum.
- Grouper les messages liés dans une même transaction plutôt que de faire une transaction par message.
- Monitorer les métriques transactionnelles : durée des transactions, taux d'abort, lag du coordinateur.

> **Décision architecturale**
>
> *Contexte* : Application de traitement d'événements de paiement. Chaque paiement doit être traité exactement une fois.
>
> *Options* :
> 1. Producteur idempotent + consommateurs idempotents : Simple, suffit si les consommateurs gèrent leurs propres duplicatas.
> 2. Transactions Kafka : Garantie exactly-once de bout en bout, mais complexité accrue.
>
> *Analyse* : Les paiements sont critiques. Un duplicata pourrait signifier un double prélèvement. Les consommateurs en aval sont multiples et difficiles à coordonner pour l'idempotence.
>
> *Décision* : Transactions Kafka avec `isolation.level=read_committed` côté consommateurs. L'overhead de latence (~20-50ms par transaction) est acceptable pour des paiements.
>
> *Alternative considérée* : Pour les cas où la latence transactionnelle est prohibitive, implémenter l'idempotence applicative (stockage des IDs traités, déduplication) peut être préférable.

### Gestion des Erreurs et Retry

Le producteur Kafka distingue deux catégories d'erreurs : les erreurs retriables et les erreurs non retriables.

**Erreurs retriables.** Ces erreurs sont potentiellement temporaires et justifient un retry automatique. Exemples : `NetworkException` (problème réseau temporaire), `NotLeaderForPartitionException` (le leader a changé, les métadonnées doivent être rafraîchies), `RequestTimedOutException` (timeout, le broker est peut-être surchargé).

**Erreurs non retriables.** Ces erreurs indiquent un problème fondamental que le retry ne résoudra pas. Exemples : `SerializationException` (le message ne peut pas être sérialisé), `RecordTooLargeException` (le message dépasse la limite de taille), `InvalidTopicException` (le topic n'existe pas ou est invalide).

**Configuration des retries.** Le paramètre `retries` définit le nombre maximal de tentatives (défaut : MAX_INT en Kafka 2.1+, essentiellement infini). Le paramètre `retry.backoff.ms` (défaut 100ms) définit le délai entre les tentatives. Le paramètre `delivery.timeout.ms` (défaut 120000ms = 2 minutes) définit le temps total maximum pour la livraison d'un message, incluant les retries.

**Impact sur l'ordre.** Les retries peuvent impacter l'ordre des messages. Si le batch B1 échoue et est réessayé pendant que B2 est envoyé avec succès, B2 peut être persisté avant B1. Pour préserver l'ordre strict avec retries, configurer `max.in.flight.requests.per.connection=1` (une seule requête en vol à la fois). L'idempotence (Kafka 2.4+) préserve l'ordre même avec jusqu'à 5 requêtes en vol.

> **Anti-patron**
>
> *« Nous désactivons les retries pour garantir l'ordre des messages. »* Cette approche sacrifie la fiabilité (perte de messages sur erreur temporaire) pour préserver l'ordre. Elle est rarement justifiée.
>
> *Meilleure approche* : Activer l'idempotence (`enable.idempotence=true`) qui garantit à la fois l'absence de duplicatas et la préservation de l'ordre avec jusqu'à 5 requêtes en vol. Si l'idempotence n'est pas disponible, `max.in.flight.requests.per.connection=1` préserve l'ordre avec un impact sur le débit.

---

## III.3.3 Stratégies de Partitionnement et Ordonnancement

### Le Partitionnement comme Décision Architecturale

Le choix de la stratégie de partitionnement est l'une des décisions architecturales les plus importantes lors de la conception d'un système Kafka. Ce choix détermine les garanties d'ordre, la distribution de charge, et le parallélisme de traitement possible.

Le partitionnement répond à deux objectifs potentiellement contradictoires. D'un côté, la **localité** : les messages qui doivent être traités ensemble ou dans un ordre spécifique doivent aller vers la même partition. De l'autre, la **distribution** : la charge doit être répartie équitablement entre les partitions pour éviter les goulots d'étranglement.

### Stratégies de Partitionnement Intégrées

**Partitionnement par clé (DefaultPartitioner avec clé).** Quand une clé non nulle est fournie, Kafka calcule `murmur2(key) % numPartitions`. Tous les messages avec la même clé vont vers la même partition, garantissant leur ordre relatif.

Cette stratégie est appropriée quand il existe une clé métier naturelle (ID client, ID commande, ID compte) et que l'ordre des messages pour cette clé est important. La distribution dépend de la distribution des clés — des clés uniformément distribuées donnent une charge équilibrée.

**Sticky Partitioning (DefaultPartitioner sans clé, Kafka 2.4+).** Sans clé, le partitionneur « colle » à une partition pendant un certain temps ou jusqu'à ce qu'un batch soit complet, puis passe à une autre partition. Cette approche améliore le batching par rapport au round-robin pur.

Cette stratégie est appropriée quand l'ordre entre messages n'a pas d'importance et que le débit est la priorité. Les messages sont distribués équitablement à long terme.

**Round-Robin (RoundRobinPartitioner).** Distribue les messages strictement en round-robin entre les partitions. Chaque message va vers la partition suivante dans la séquence.

Cette stratégie garantit une distribution parfaitement équitable mais peut réduire l'efficacité du batching car les messages consécutifs vont vers des partitions différentes.

**Partition explicite.** L'application peut spécifier explicitement la partition dans l'appel `send()`. Cette approche donne un contrôle total mais couple l'application à la topologie du topic.

### Implémentation d'un Partitionneur Personnalisé

Pour des besoins spécifiques, une application peut implémenter l'interface `Partitioner`. Un partitionneur personnalisé peut implémenter une logique métier arbitraire.

```java
public class GeoPartitioner implements Partitioner {
    @Override
    public int partition(String topic, Object key, byte[] keyBytes,
                        Object value, byte[] valueBytes, Cluster cluster) {
        // Extraction de la région depuis la clé (ex: "EU-customer123")
        String region = extractRegion(key.toString());
        
        // Mapping région -> plage de partitions
        int numPartitions = cluster.partitionCountForTopic(topic);
        return regionToPartitionRange(region, numPartitions);
    }
    
    private String extractRegion(String key) {
        return key.split("-")[0]; // Simplifié
    }
    
    private int regionToPartitionRange(String region, int numPartitions) {
        // EU -> partitions 0-3, US -> partitions 4-7, APAC -> partitions 8-11
        switch(region) {
            case "EU": return ThreadLocalRandom.current().nextInt(0, 4);
            case "US": return ThreadLocalRandom.current().nextInt(4, 8);
            case "APAC": return ThreadLocalRandom.current().nextInt(8, 12);
            default: return ThreadLocalRandom.current().nextInt(numPartitions);
        }
    }
}
```

**Cas d'usage des partitionneurs personnalisés :**

*Routage géographique* : Diriger les messages vers des partitions proches des consommateurs concernés pour réduire la latence.

*Équilibrage de charge intelligent* : Prendre en compte la charge actuelle des partitions pour éviter les hot spots.

*Isolation par tenant* : Dans un système multi-tenant, isoler les données de chaque tenant dans des partitions spécifiques.

*Priorités* : Kafka ne supporte pas les priorités natives, mais un partitionneur peut diriger les messages haute priorité vers des partitions dédiées avec des consommateurs plus nombreux.

> **Note de terrain**
>
> *Contexte* : Système de e-commerce avec un topic `orders` partitionné par `customer_id`. Pendant le Black Friday, quelques gros clients génèrent 30% du trafic, créant des « hot partitions ».
>
> *Problème* : Le consumer lag sur ces partitions explose pendant que d'autres partitions sont sous-utilisées.
>
> *Solution initiale envisagée* : Partitionneur personnalisé qui distribue les commandes des « gros clients » sur plusieurs partitions.
>
> *Problème de la solution* : Perte de l'ordre des commandes par client, ce qui peut causer des incohérences (annulation avant création).
>
> *Solution retenue* : Accepter le déséquilibre et dimensionner les consommateurs pour le pire cas. Ajouter du monitoring pour alerter sur les hot partitions. Documenter la contrainte métier qui justifie ce choix.
>
> *Leçon* : Le partitionnement est souvent un compromis entre l'ordre (invariants métier) et la distribution (performance). Les invariants métier doivent généralement primer.

### Préservation de l'Ordre

L'ordre des messages est garanti uniquement au sein d'une partition. Pour les applications où l'ordre global ou inter-entité est critique, plusieurs stratégies sont possibles.

**Partition unique.** Le cas le plus simple : tous les messages vont vers une seule partition. L'ordre global est garanti, mais le parallélisme est impossible (un seul consommateur actif). Cette approche convient uniquement aux très faibles volumes.

**Clé de partitionnement basée sur l'entité.** Tous les messages concernant la même entité (client, commande, compte) partagent la même clé et donc la même partition. L'ordre est garanti par entité, et le parallélisme est possible entre entités. C'est l'approche la plus courante et recommandée.

**Numéro de séquence applicatif.** L'application inclut un numéro de séquence dans chaque message. Les consommateurs peuvent réordonner les messages si nécessaire. Cette approche est complexe et rarement recommandée car elle déplace la complexité vers les consommateurs.

**Timestamps et fenêtrage.** Pour certains cas d'usage (analytique, agrégation), l'ordre strict n'est pas nécessaire si les messages arrivent dans une fenêtre temporelle acceptable. Le stream processing avec fenêtrage peut gérer le désordre borné.

**Garanties d'ordre avec retries.** Les retries peuvent perturber l'ordre si `max.in.flight.requests.per.connection` > 1. Par exemple, si le batch B1 échoue et est réessayé pendant que B2 est envoyé avec succès, B2 peut être persisté avant B1.

Pour préserver l'ordre strict avec retries :
- Activer l'idempotence (`enable.idempotence=true`) — préserve l'ordre avec jusqu'à 5 requêtes en vol.
- Ou configurer `max.in.flight.requests.per.connection=1` — une seule requête en vol, ordre garanti mais débit réduit.

> **Note de terrain**
>
> *Contexte* : Application de gestion de comptes bancaires où l'ordre des opérations est critique (le solde doit être cohérent).
>
> *Exigence* : Les opérations sur un même compte doivent être traitées dans l'ordre de soumission.
>
> *Solution* : Clé de partitionnement = numéro de compte. Idempotence activée. `max.in.flight.requests.per.connection=5` (acceptable avec idempotence).
>
> *Validation* : Tests de charge avec simulation de pannes réseau pour vérifier que l'ordre est préservé même avec retries.
>
> *Résultat* : L'ordre par compte est garanti. Le parallélisme entre comptes permet un débit suffisant.

### Impact du Nombre de Partitions sur le Partitionnement

Le nombre de partitions d'un topic influence le comportement du partitionnement et doit être considéré lors de la conception.

**Stabilité du mapping.** Le mapping clé → partition dépend du nombre de partitions (`hash(key) % numPartitions`). Si le nombre de partitions change, le mapping change pour certaines clés. Les messages avec ces clés iront vers de nouvelles partitions, potentiellement en désordre avec les messages précédents.

Par exemple, avec 10 partitions, la clé « customer-123 » pourrait aller vers la partition 7. Si on augmente à 15 partitions, la même clé pourrait maintenant aller vers la partition 12. Les nouveaux messages de ce client seront dans la partition 12, mais les anciens messages sont toujours dans la partition 7.

**Granularité de la distribution.** Avec peu de partitions, la distribution peut être déséquilibrée si certaines clés sont plus fréquentes. Plus de partitions permettent une distribution plus fine, mais au prix d'une complexité opérationnelle accrue.

**Recommandation.** Dimensionner le nombre de partitions dès le départ pour la charge maximale anticipée, avec une marge confortable. Éviter d'augmenter le nombre de partitions sur un topic avec des données ordonnées en production. Si une augmentation est nécessaire, planifier une période de transition où le désordre temporaire est acceptable.

---

## III.3.4 Sérialisation des Données et Gestion des Schémas

### Le Rôle Critique de la Sérialisation

La sérialisation convertit les objets applicatifs en séquences d'octets pour le transport et le stockage. La désérialisation effectue l'opération inverse côté consommateur. Ces opérations sont critiques pour l'interopérabilité entre producteurs et consommateurs, potentiellement développés par des équipes différentes, dans des langages différents, et évoluant à des rythmes différents.

Kafka est agnostique au format des données — il transporte des tableaux d'octets sans interprétation. Cette flexibilité laisse le choix du format à l'architecte, mais ce choix a des implications majeures sur l'évolutivité, la performance, et la gouvernance.

### Formats de Sérialisation Courants

**JSON.** Le format le plus accessible : lisible par les humains, supporté universellement, sans schéma explicite. Cependant, JSON est verbeux (noms de champs répétés dans chaque message), lent à parser, et sans validation de schéma native. JSON convient au prototypage et aux cas où la lisibilité prime sur la performance, mais est déconseillé pour les systèmes à haute volumétrie.

**Apache Avro.** Format binaire compact avec schéma évolutif. Le schéma est séparé des données et peut être stocké dans un Schema Registry. Avro supporte l'évolution des schémas (ajout/suppression de champs) avec des règles de compatibilité. C'est le format recommandé pour la majorité des déploiements Kafka entreprise.

**Protocol Buffers (Protobuf).** Format binaire développé par Google, très performant et compact. Protobuf utilise des fichiers `.proto` pour définir les schémas et génère du code dans de nombreux langages. Excellent pour la performance mais l'évolution des schémas est plus contrainte qu'Avro.

**Apache Thrift.** Similar à Protobuf, développé par Facebook. Moins courant dans l'écosystème Kafka.

| Format | Taille | Vitesse | Lisibilité | Évolution schéma | Adoption Kafka |
|--------|--------|---------|------------|------------------|----------------|
| JSON | Grande | Lente | Excellente | Ad hoc | Prototypage |
| Avro | Compacte | Rapide | Binaire | Excellente | Recommandé |
| Protobuf | Très compacte | Très rapide | Binaire | Bonne | Performance critique |

### Schema Registry

Le **Schema Registry** est un composant central pour la gestion des schémas dans un écosystème Kafka. Il stocke les schémas, assigne des identifiants uniques, et valide la compatibilité des évolutions.

**Fonctionnement.** Le producteur enregistre le schéma lors du premier envoi (ou vérifie qu'il existe). Le Schema Registry retourne un ID de schéma. Le producteur inclut cet ID dans chaque message (typiquement les 5 premiers octets). Le consommateur extrait l'ID, récupère le schéma depuis le Registry, et désérialise le message.

**Règles de compatibilité.** Le Schema Registry applique des règles de compatibilité lors de l'enregistrement de nouveaux schémas :

- **BACKWARD** : Les nouveaux schémas peuvent lire les données écrites avec des anciens schémas. Permet d'ajouter des champs optionnels ou de supprimer des champs.
- **FORWARD** : Les anciens schémas peuvent lire les données écrites avec des nouveaux schémas. Permet d'ajouter des champs ou de supprimer des champs optionnels.
- **FULL** : Combinaison de BACKWARD et FORWARD. Les schémas sont compatibles dans les deux sens.
- **NONE** : Pas de validation de compatibilité (déconseillé en production).

**Implémentations.** Confluent Schema Registry est l'implémentation de référence, intégrée à Confluent Platform et Confluent Cloud. Des alternatives open source existent : Apicurio (Red Hat), Karapace (Aiven).

> **Exemple concret**
>
> *Scénario* : Évolution d'un schéma d'événement `OrderCreated`.
>
> *Version 1* :
> ```json
> {
>   "type": "record",
>   "name": "OrderCreated",
>   "fields": [
>     {"name": "orderId", "type": "string"},
>     {"name": "customerId", "type": "string"},
>     {"name": "amount", "type": "double"}
>   ]
> }
> ```
>
> *Version 2* : Ajout d'un champ `currency` avec valeur par défaut.
> ```json
> {
>   "type": "record",
>   "name": "OrderCreated",
>   "fields": [
>     {"name": "orderId", "type": "string"},
>     {"name": "customerId", "type": "string"},
>     {"name": "amount", "type": "double"},
>     {"name": "currency", "type": "string", "default": "USD"}
>   ]
> }
> ```
>
> *Avec compatibilité BACKWARD* : Cette évolution est acceptée. Les nouveaux consommateurs (V2) peuvent lire les anciens messages (V1) en utilisant la valeur par défaut pour `currency`.
>
> *Version 3 problématique* : Suppression du champ `customerId` sans valeur par défaut.
>
> *Avec compatibilité BACKWARD* : Cette évolution est rejetée. Les nouveaux consommateurs ne pourraient pas lire les anciens messages qui contiennent `customerId`.

### Bonnes Pratiques de Gestion des Schémas

**Définir la compatibilité au niveau du sujet.** Chaque topic (ou sujet dans le Schema Registry) devrait avoir une règle de compatibilité définie. BACKWARD est recommandé pour la plupart des cas car elle permet aux consommateurs de se mettre à jour avant les producteurs.

La compatibilité BACKWARD signifie qu'un nouveau schéma peut lire les données écrites avec l'ancien schéma. Cela permet un déploiement progressif : déployer d'abord les nouveaux consommateurs (qui comprennent les deux formats), puis déployer les nouveaux producteurs.

**Versionner explicitement les schémas.** Inclure un champ de version ou utiliser les namespaces pour distinguer les versions majeures incompatibles. Pour les changements incompatibles, créer un nouveau topic plutôt que de casser la compatibilité.

```json
{
  "type": "record",
  "name": "OrderCreated",
  "namespace": "com.example.orders.v2",
  "fields": [...]
}
```

**Documenter les schémas.** Utiliser les champs `doc` d'Avro pour documenter la sémantique de chaque champ. Cette documentation devient la spécification du contrat entre producteurs et consommateurs.

```json
{
  "type": "record",
  "name": "OrderCreated",
  "doc": "Événement émis quand une nouvelle commande est créée dans le système.",
  "fields": [
    {
      "name": "orderId",
      "type": "string",
      "doc": "Identifiant unique de la commande, format UUID."
    },
    {
      "name": "customerId",
      "type": "string",
      "doc": "Identifiant du client ayant passé la commande."
    },
    {
      "name": "totalAmount",
      "type": {
        "type": "bytes",
        "logicalType": "decimal",
        "precision": 10,
        "scale": 2
      },
      "doc": "Montant total de la commande en devise locale."
    }
  ]
}
```

**Tester les évolutions.** Avant de déployer une nouvelle version de schéma en production, valider la compatibilité avec le Schema Registry et tester avec des données réelles.

```bash
# Validation de compatibilité avant enregistrement
curl -X POST \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  --data '{"schema": "{...}"}' \
  http://schema-registry:8081/compatibility/subjects/orders-value/versions/latest

# Réponse : {"is_compatible": true}
```

**Gouverner les changements.** Établir un processus de revue pour les modifications de schémas, similaire aux revues de code. Les changements de schémas impactent potentiellement tous les consommateurs d'un topic.

Un processus typique de gouvernance des schémas inclut :
1. Proposition de changement avec justification métier.
2. Validation de la compatibilité avec le Schema Registry.
3. Revue par l'équipe architecture ou les owners du topic.
4. Tests d'intégration avec les consommateurs connus.
5. Déploiement progressif avec monitoring.

**Stratégies de migration pour changements incompatibles :**

Quand un changement incompatible est nécessaire (changement de type d'un champ, renommage d'un champ obligatoire), plusieurs stratégies existent :

*Double-écriture temporaire.* Produire vers l'ancien et le nouveau topic simultanément pendant une période de migration. Les consommateurs migrent progressivement vers le nouveau topic.

*Topic versionné.* Créer un nouveau topic avec le nouveau schéma (ex: `orders-v2`). Migrer les consommateurs un par un, puis arrêter la production vers l'ancien topic.

*Événement de migration.* Publier un événement spécial qui signale le changement de format, permettant aux consommateurs de s'adapter dynamiquement.

> **Anti-patron**
>
> *« Nous utilisons JSON sans schéma pour rester flexibles. »* Cette approche, séduisante au début, conduit invariablement à des problèmes :
> - Consommateurs qui cassent silencieusement quand un champ change de nom ou de type.
> - Impossibilité de savoir quels champs sont obligatoires ou optionnels.
> - Documentation qui diverge de la réalité.
> - Débogage difficile des erreurs de parsing.
>
> *Meilleure approche* : Utiliser Avro ou Protobuf avec Schema Registry dès le début. Le « coût » de la définition de schémas est largement compensé par la fiabilité et la maintenabilité à long terme.

---

## III.3.5 Optimisation des Performances

### Métriques de Performance du Producteur

Avant d'optimiser, il faut mesurer. Les métriques clés du producteur permettent d'identifier les goulots d'étranglement et de valider les optimisations.

**Débit (throughput).** Nombre de messages ou volume de données produits par seconde. Mesuré par `record-send-rate` et `byte-rate` dans les métriques JMX du producteur.

**Latence.** Temps entre l'appel `send()` et la confirmation du broker. Mesuré par `request-latency-avg` et `request-latency-max`. La latence perçue par l'application inclut aussi le temps d'accumulation dans le buffer.

**Taux d'erreur.** Pourcentage de messages qui échouent après tous les retries. Mesuré par `record-error-rate`. Un taux non nul indique des problèmes à investiguer.

**Utilisation des buffers.** Pourcentage de la mémoire buffer utilisée. Mesuré par `buffer-available-bytes` comparé à `buffer.memory`. Un buffer constamment plein indique que le producteur ne peut pas suivre le rythme de l'application.

**Batching efficiency.** Taille moyenne des batches envoyés, comparée à `batch.size`. Des batches petits indiquent que `linger.ms` est trop bas ou que le débit est faible.

### Leviers d'Optimisation du Débit

**Augmenter `batch.size`.** Des batches plus grands amortissent le coût de chaque requête réseau sur plus de messages. La valeur par défaut (16 Ko) est conservatrice. Pour les charges élevées, des valeurs de 64 Ko à 256 Ko peuvent significativement améliorer le débit.

**Augmenter `linger.ms`.** Un temps d'attente plus long permet d'accumuler plus de messages par batch. Une valeur de 5-20 ms est souvent un bon compromis entre latence et débit. Pour les applications où la latence n'est pas critique, des valeurs plus élevées (50-100 ms) maximisent le batching.

**Activer la compression.** La compression réduit le volume de données à transmettre et à stocker. LZ4 ou Zstd offrent d'excellents ratios avec un overhead CPU modéré. Pour les données textuelles (JSON, logs), des ratios de compression de 60-80% sont courants.

**Augmenter `buffer.memory`.** Plus de mémoire permet d'absorber les pics de production et de maintenir un pipeline de batches prêts à envoyer. La valeur par défaut (32 Mo) peut être insuffisante pour les charges élevées.

**Paralléliser les producteurs.** Pour les très hauts débits, plusieurs instances de producteur (dans des threads ou processus séparés) peuvent être nécessaires. Chaque producteur a son propre buffer et sender thread.

### Leviers d'Optimisation de la Latence

**Réduire `linger.ms`.** Une valeur de 0 envoie immédiatement chaque message (ou petit batch). Cela minimise la latence mais peut réduire le débit si les messages arrivent en flux continu.

**Utiliser `acks=1` plutôt que `acks=all`.** La confirmation du leader seul est plus rapide que l'attente de tous les ISR. Ce choix sacrifie de la durabilité pour de la latence.

**Optimiser la sérialisation.** Des sérialiseurs efficaces (Avro, Protobuf) sont plus rapides que JSON. La génération de code (Avro SpecificRecord, Protobuf) est plus rapide que la réflexion (Avro GenericRecord).

Pour les applications où chaque milliseconde compte, profiler la sérialisation pour identifier les goulots d'étranglement. La création d'objets, la réflexion, et la conversion de types peuvent être coûteuses.

**Proximité réseau.** La latence réseau entre le producteur et les brokers est incompressible. Déployer les producteurs dans la même région/zone que les brokers. Pour les déploiements multi-région, chaque région devrait avoir ses propres brokers locaux.

**Tuning TCP.** Pour les cas de latence critique, ajuster les paramètres TCP :
- `socket.send.buffer.bytes` : Taille du buffer d'envoi socket.
- `socket.receive.buffer.bytes` : Taille du buffer de réception.
- Désactiver Nagle's algorithm au niveau OS si nécessaire.

**Pré-chauffage des connexions.** Au démarrage, le producteur doit établir les connexions et récupérer les métadonnées. Les premiers messages peuvent avoir une latence plus élevée. Pour les applications sensibles à la latence de démarrage, envoyer quelques messages de « warm-up » avant le trafic réel.

### Optimisation de la Compression

La compression peut significativement améliorer le débit en réduisant le volume de données transférées, mais elle consomme du CPU. Le choix de l'algorithme et des paramètres dépend du profil de l'application.

**Comparaison des algorithmes :**

| Algorithme | Ratio | Vitesse compression | Vitesse décompression | CPU |
|------------|-------|---------------------|----------------------|-----|
| None | 1.0x | - | - | Minimal |
| Snappy | 1.5-2x | Très rapide | Très rapide | Faible |
| LZ4 | 2-3x | Très rapide | Très rapide | Faible |
| Zstd | 3-5x | Rapide | Rapide | Modéré |
| GZIP | 4-6x | Lente | Modérée | Élevé |

**Recommandations par cas d'usage :**

- *Latence critique* : Snappy ou LZ4 (overhead minimal).
- *Débit maximal* : LZ4 ou Zstd niveau bas (bon ratio, rapide).
- *Coût stockage/réseau critique* : Zstd niveau 3-5 ou GZIP (meilleur ratio).
- *Messages déjà compressés* : None (double compression inefficace).

**Compression par batch.** La compression s'applique au batch entier, pas aux messages individuels. Des batches plus grands donnent de meilleurs ratios de compression car l'algorithme peut exploiter les patterns répétitifs sur plus de données.

**Impact sur les consommateurs.** La décompression s'effectue côté consommateur. Un algorithme lent à compresser mais rapide à décompresser (comme GZIP) peut être acceptable si les consommateurs sont nombreux et les producteurs peu nombreux.

### Compromis Latence vs. Débit

Les optimisations de latence et de débit sont souvent en tension. L'architecte doit comprendre ces compromis pour faire des choix éclairés.

| Paramètre | Pour le débit | Pour la latence | Impact |
|-----------|---------------|-----------------|--------|
| `linger.ms` | Augmenter (5-100ms) | Réduire (0-5ms) | Batching vs. réactivité |
| `batch.size` | Augmenter (64-256 Ko) | Réduire (16 Ko) | Efficacité vs. attente |
| `acks` | `1` ou `0` | `1` | Durabilité vs. vitesse |
| `compression` | Activer | Dépend du CPU | Réseau vs. CPU |
| `buffer.memory` | Augmenter | Moins important | Capacité d'absorption |
| `max.in.flight.requests` | Augmenter (5) | Maintenir (5) | Pipeline vs. ordre |

> **Note de terrain**
>
> *Contexte* : Optimisation d'un producteur pour un système de trading où la latence est critique (< 10ms p99).
>
> *Configuration initiale* : Défauts Kafka (`linger.ms=0`, `batch.size=16Ko`, `acks=all`).
>
> *Mesures initiales* : Latence p99 = 45ms, principalement due à l'attente des ISR.
>
> *Optimisations appliquées* :
> 1. `acks=1` : Latence p99 → 15ms (suppression de l'attente des followers).
> 2. Brokers sur SSD avec cache suffisant : Latence p99 → 8ms.
> 3. Compression désactivée (messages petits, CPU est le goulot) : Latence p99 → 6ms.
>
> *Compromis accepté* : `acks=1` réduit la durabilité (perte possible si le leader crashe immédiatement après confirmation). Mitigation : réplication synchrone avec `min.insync.replicas=2` sur les brokers, de sorte que le leader a probablement déjà répliqué quand il confirme.
>
> *Leçon* : L'optimisation de latence requiert de comprendre où le temps est passé (profiling) et d'accepter des compromis explicites.

### Monitoring et Alerting

Un producteur en production doit être monitoré pour détecter les problèmes avant qu'ils n'impactent les utilisateurs. Les métriques du producteur sont exposées via JMX et peuvent être collectées par Prometheus, Datadog, ou d'autres systèmes de monitoring.

**Métriques à surveiller :**

*Métriques de débit :*
- `record-send-rate` : Nombre de messages envoyés par seconde.
- `byte-rate` : Volume de données envoyées par seconde.
- `record-send-total` : Total cumulé de messages envoyés.

*Métriques de latence :*
- `request-latency-avg` : Latence moyenne des requêtes vers les brokers.
- `request-latency-max` : Latence maximale observée.
- `record-queue-time-avg` : Temps moyen passé dans le buffer avant envoi.

*Métriques d'erreur :*
- `record-error-rate` : Taux de messages en erreur par seconde.
- `record-error-total` : Total cumulé de messages en erreur.
- `record-retry-rate` : Taux de retry par seconde.

*Métriques de ressources :*
- `buffer-available-bytes` : Mémoire buffer disponible.
- `buffer-total-bytes` : Mémoire buffer totale (égale à `buffer.memory`).
- `bufferpool-wait-time` : Temps d'attente pour obtenir de la mémoire buffer.
- `waiting-threads` : Nombre de threads bloqués en attente de mémoire.

*Métriques de batching :*
- `batch-size-avg` : Taille moyenne des batches envoyés.
- `batch-size-max` : Taille maximale des batches.
- `records-per-request-avg` : Nombre moyen de messages par requête.
- `compression-rate-avg` : Ratio de compression moyen.

*Métriques de connexion :*
- `connection-count` : Nombre de connexions actives vers les brokers.
- `connection-creation-rate` : Taux de création de nouvelles connexions.
- `connection-close-rate` : Taux de fermeture de connexions.

**Configuration de l'export des métriques :**

Pour Prometheus avec JMX Exporter :

```yaml
# jmx_exporter_config.yml
rules:
  - pattern: kafka.producer<type=producer-metrics, client-id=(.+)><>(.+)
    name: kafka_producer_$2
    labels:
      client_id: "$1"
  - pattern: kafka.producer<type=producer-topic-metrics, client-id=(.+), topic=(.+)><>(.+)
    name: kafka_producer_topic_$3
    labels:
      client_id: "$1"
      topic: "$2"
```

**Alertes recommandées :**

| Métrique | Seuil | Sévérité | Action |
|----------|-------|----------|--------|
| `record-error-rate` | > 0.1% pendant 5 min | Haute | Investiguer les erreurs |
| `buffer-available-bytes` | < 20% pendant 2 min | Haute | Vérifier le débit, augmenter buffer |
| `request-latency-avg` | > 2× baseline pendant 5 min | Moyenne | Vérifier les brokers |
| `batch-size-avg` | < 10% de `batch.size` | Basse | Ajuster `linger.ms` |
| `waiting-threads` | > 0 pendant 1 min | Moyenne | Backpressure, réduire le débit |
| `connection-creation-rate` | > 10/min | Basse | Vérifier la stabilité réseau |

**Dashboards recommandés :**

Un dashboard de monitoring de producteur devrait inclure :

1. *Vue d'ensemble* : Débit global (messages/s, bytes/s), taux d'erreur global.
2. *Latence* : Histogramme de latence, percentiles (p50, p95, p99).
3. *Ressources* : Utilisation du buffer, threads en attente.
4. *Par topic* : Débit et erreurs ventilés par topic.
5. *Santé* : Connexions actives, retries, compression.

---

## III.3.6 Recommandations Architecturales pour les Producers

### Patterns de Conception Recommandés

**Producteur singleton par application.** Créer une instance de producteur au démarrage de l'application et la réutiliser pour tous les envois. Le producteur est thread-safe et conçu pour être partagé. Créer un producteur par envoi est un anti-pattern qui gaspille des ressources et dégrade les performances.

```java
// Anti-pattern : producteur par envoi
public void sendMessage(String message) {
    try (KafkaProducer<String, String> producer = new KafkaProducer<>(props)) {
        producer.send(new ProducerRecord<>("topic", message));
    }
}

// Pattern recommandé : producteur singleton
public class MessageSender {
    private final KafkaProducer<String, String> producer;
    
    public MessageSender(Properties props) {
        this.producer = new KafkaProducer<>(props);
    }
    
    public void sendMessage(String message) {
        producer.send(new ProducerRecord<>("topic", message), this::handleResult);
    }
    
    public void close() {
        producer.close();
    }
}
```

**Fermeture gracieuse.** Appeler `producer.close()` lors de l'arrêt de l'application. Cette méthode attend que les messages en buffer soient envoyés et confirmés. Sans fermeture gracieuse, les messages en buffer peuvent être perdus.

**Gestion centralisée des erreurs.** Implémenter un handler d'erreur centralisé invoqué par les callbacks. Ce handler peut logger, alerter, stocker les messages en erreur pour retry ultérieur, ou déclencher un circuit breaker.

**Séparation des topics par criticité.** Utiliser des producteurs séparés (avec des configurations différentes) pour les topics critiques et non critiques. Les messages critiques utilisent `acks=all` et des retries agressifs ; les messages non critiques utilisent `acks=1` et des configurations optimisées pour le débit.

### Résilience et Haute Disponibilité

**Idempotence par défaut.** Activer `enable.idempotence=true` pour tous les producteurs (c'est le défaut en Kafka 3.0+). Il n'y a pas de raison de ne pas l'activer — les gains en fiabilité sont gratuits.

L'idempotence ne garantit pas l'exactly-once de bout en bout (qui nécessite les transactions), mais elle élimine les duplicatas causés par les retries du producteur. C'est une amélioration significative par rapport au comportement at-least-once de base.

**Timeouts appropriés.** Configurer `delivery.timeout.ms` selon les exigences de l'application. La valeur par défaut (2 minutes) est conservatrice. Pour les applications temps réel, une valeur plus courte (30 secondes) permet de détecter les problèmes plus rapidement.

La hiérarchie des timeouts doit être cohérente :
- `delivery.timeout.ms` ≥ `linger.ms` + `request.timeout.ms`
- `request.timeout.ms` (défaut 30s) est le timeout d'une requête individuelle
- `retry.backoff.ms` (défaut 100ms) est le délai entre les retries

**Circuit breaker applicatif.** Si le producteur échoue de manière répétée (broker down, réseau coupé), l'application ne devrait pas continuer à accumuler des messages indéfiniment. Implémenter un circuit breaker qui rejette les nouveaux messages quand le producteur est en échec, permettant à l'application de réagir (mode dégradé, stockage alternatif).

```java
public class ResilientProducer {
    private final KafkaProducer<String, String> producer;
    private final CircuitBreaker circuitBreaker;
    private final AtomicInteger consecutiveFailures = new AtomicInteger(0);
    
    private static final int FAILURE_THRESHOLD = 10;
    private static final Duration RECOVERY_TIMEOUT = Duration.ofSeconds(30);
    
    public void send(ProducerRecord<String, String> record) {
        if (!circuitBreaker.allowRequest()) {
            throw new CircuitOpenException("Producteur en mode dégradé");
        }
        
        producer.send(record, (metadata, exception) -> {
            if (exception != null) {
                if (consecutiveFailures.incrementAndGet() >= FAILURE_THRESHOLD) {
                    circuitBreaker.trip();
                }
                // Gestion de l'erreur
            } else {
                consecutiveFailures.set(0);
            }
        });
    }
}
```

**Métriques de santé.** Exposer les métriques du producteur via JMX ou un endpoint de santé. Intégrer ces métriques au système de monitoring de l'organisation. Un producteur qui ne peut pas envoyer est aussi problématique qu'un broker down.

**Stratégie de backpressure.** Quand le producteur ne peut pas suivre le rythme de l'application (buffer plein), plusieurs stratégies sont possibles :

*Blocage* : L'appel `send()` bloque jusqu'à ce que de la mémoire soit disponible (`max.block.ms`). Simple mais peut propager les problèmes à l'application appelante.

*Rejet rapide* : Configurer `max.block.ms=0` pour échouer immédiatement si le buffer est plein. L'application doit gérer le rejet (file locale, stockage alternatif).

*Throttling* : L'application limite proactivement son débit de production basé sur les métriques du producteur (buffer utilization).

**Gestion des Dead Letter Queues (DLQ).** Bien que Kafka n'ait pas de DLQ native côté producteur, une implémentation applicative est recommandée pour les messages qui échouent de manière répétée :

```java
public void sendWithDLQ(ProducerRecord<String, String> record) {
    producer.send(record, (metadata, exception) -> {
        if (exception != null && isNonRetriable(exception)) {
            // Envoyer vers la DLQ
            ProducerRecord<String, String> dlqRecord = new ProducerRecord<>(
                record.topic() + ".dlq",
                record.key(),
                record.value()
            );
            dlqRecord.headers().add("X-Original-Topic", record.topic().getBytes());
            dlqRecord.headers().add("X-Error", exception.getMessage().getBytes());
            producer.send(dlqRecord);
        }
    });
}
```

### Considérations Multi-Datacenter

**Producteur local, réplication globale.** Le producteur devrait toujours envoyer vers le cluster Kafka local (même région/datacenter). La réplication vers d'autres régions est gérée par MirrorMaker ou Cluster Linking, pas par le producteur.

**Fallback en cas de panne régionale.** Si le cluster local devient indisponible, l'application peut basculer vers un cluster distant. Ce basculement doit être explicite (changement de configuration, redémarrage) plutôt qu'automatique, car il a des implications sur la latence et potentiellement sur l'ordre des messages.

**Gestion des identifiants transactionnels.** Si les transactions Kafka sont utilisées, le `transactional.id` doit être unique par datacenter pour éviter les conflits. Un pattern courant est d'inclure l'identifiant du datacenter dans le transactional.id.

### Tests des Producteurs

**Tests unitaires avec MockProducer.** Kafka fournit un `MockProducer` pour les tests unitaires. Il permet de vérifier les messages envoyés sans cluster Kafka réel.

```java
MockProducer<String, String> mockProducer = new MockProducer<>(
    true, // autocomplete
    new StringSerializer(),
    new StringSerializer()
);

MyService service = new MyService(mockProducer);
service.processOrder(order);

List<ProducerRecord<String, String>> records = mockProducer.history();
assertEquals(1, records.size());
assertEquals("orders", records.get(0).topic());
```

**Tests d'intégration avec Testcontainers.** Pour les tests d'intégration, Testcontainers permet de démarrer un cluster Kafka éphémère dans Docker.

```java
@Testcontainers
class KafkaIntegrationTest {
    @Container
    static KafkaContainer kafka = new KafkaContainer(
        DockerImageName.parse("confluentinc/cp-kafka:7.5.0")
    );
    
    @Test
    void shouldProduceMessages() {
        Properties props = new Properties();
        props.put("bootstrap.servers", kafka.getBootstrapServers());
        // ... configuration
        
        try (KafkaProducer<String, String> producer = new KafkaProducer<>(props)) {
            producer.send(new ProducerRecord<>("test-topic", "key", "value")).get();
        }
        
        // Vérification avec un consommateur
    }
}
```

**Tests de charge.** Avant la mise en production, valider les performances avec des tests de charge réalistes. L'outil `kafka-producer-perf-test.sh` fourni avec Kafka permet des tests rapides. Pour des tests plus sophistiqués, des outils comme Gatling ou custom JMeter peuvent simuler des patterns de production réalistes.

```bash
# Test de performance intégré
kafka-producer-perf-test.sh \
    --topic test-topic \
    --num-records 1000000 \
    --record-size 1024 \
    --throughput -1 \
    --producer-props bootstrap.servers=localhost:9092 \
                     acks=all \
                     linger.ms=5 \
                     batch.size=65536
```

**Tests de résilience.** Valider le comportement du producteur en cas de panne : broker down, partition leader failover, réseau lent. Ces tests révèlent souvent des problèmes de configuration ou de gestion d'erreurs. Des outils comme Chaos Monkey, Toxiproxy, ou simplement `iptables` peuvent simuler ces conditions.

### Producteurs dans Différents Langages

Bien que ce chapitre utilise Java pour les exemples, Kafka dispose de clients dans de nombreux langages. L'architecte doit comprendre les différences et les limitations.

**librdkafka (C/C++).** Bibliothèque native de haute performance utilisée par de nombreux wrappers dans d'autres langages. Offre d'excellentes performances mais une API de plus bas niveau.

**confluent-kafka-python.** Wrapper Python autour de librdkafka. Populaire pour les applications data science et les scripts. Attention : le GIL Python peut limiter le parallélisme.

```python
from confluent_kafka import Producer

producer = Producer({
    'bootstrap.servers': 'localhost:9092',
    'acks': 'all'
})

def delivery_callback(err, msg):
    if err:
        print(f'Erreur: {err}')
    else:
        print(f'Message envoyé à {msg.topic()} [{msg.partition()}]')

producer.produce('topic', key='key', value='value', callback=delivery_callback)
producer.flush()
```

**confluent-kafka-go.** Client Go natif avec d'excellentes performances. Idiomatique pour les développeurs Go.

**node-rdkafka et kafkajs.** Deux options pour Node.js : node-rdkafka (wrapper librdkafka, plus performant) et kafkajs (JavaScript pur, plus facile à installer).

**.NET (Confluent.Kafka).** Client .NET officiel, wrapper autour de librdkafka. Bien intégré à l'écosystème .NET.

**Considérations cross-langage :**

- Les fonctionnalités avancées (transactions, exactly-once) peuvent ne pas être disponibles dans tous les clients.
- Les performances varient significativement entre les implémentations.
- La sérialisation avec Schema Registry nécessite des bibliothèques spécifiques par langage.
- Les configurations par défaut peuvent différer — toujours vérifier.

> **Perspective stratégique**
>
> Le producteur Kafka est souvent traité comme un « détail d'implémentation », configuré avec les valeurs par défaut et oublié. Cette approche mène à des problèmes en production : perte de messages, latence imprévisible, saturation de mémoire.
>
> L'architecte devrait traiter la configuration du producteur comme une décision architecturale à part entière, documentée, revue, et testée. Les choix de `acks`, de stratégie de partitionnement, de gestion des schémas, et de résilience ont un impact direct sur les garanties du système global.
>
> Un « template » de producteur bien configuré, documenté, et testé devrait être fourni aux équipes de développement comme point de départ. Ce template encode les décisions architecturales de l'organisation et assure une cohérence entre les applications.

### Checklist de Mise en Production

Avant de déployer un producteur en production, valider les points suivants :

**Configuration :**
- [ ] `bootstrap.servers` pointe vers le cluster de production
- [ ] `acks` est configuré selon la criticité des données
- [ ] `enable.idempotence=true` est activé
- [ ] `compression.type` est défini (lz4 ou zstd recommandé)
- [ ] `linger.ms` et `batch.size` sont ajustés pour le cas d'usage
- [ ] `delivery.timeout.ms` est approprié

**Sérialisation :**
- [ ] Les sérialiseurs sont configurés et testés
- [ ] Le schéma est enregistré dans Schema Registry
- [ ] La compatibilité du schéma est vérifiée

**Résilience :**
- [ ] Les callbacks gèrent les erreurs
- [ ] Un circuit breaker est implémenté si approprié
- [ ] La fermeture gracieuse est implémentée

**Monitoring :**
- [ ] Les métriques du producteur sont exposées
- [ ] Les alertes sont configurées
- [ ] Les dashboards sont disponibles

**Tests :**
- [ ] Tests unitaires avec MockProducer
- [ ] Tests d'intégration avec cluster de test
- [ ] Tests de charge validés
- [ ] Tests de résilience effectués

---

## III.3.7 Résumé

Ce chapitre a exploré en profondeur le producteur Kafka, le composant responsable de la publication des messages vers le cluster. Une compréhension approfondie du producteur est essentielle pour concevoir des applications fiables et performantes.

### Architecture Interne

Le producteur Kafka est un composant sophistiqué avec plusieurs sous-systèmes : le thread principal qui sérialise et partitionne, le RecordAccumulator qui batche les messages par partition, le thread sender qui gère l'envoi réseau et les réponses, et le buffer pool qui gère la mémoire.

Le cycle de vie d'un message traverse ces composants : sérialisation → partitionnement → accumulation → envoi → confirmation. Chaque étape est configurable et impacte les performances et les garanties. Les intercepteurs et les headers permettent d'enrichir ce cycle avec des comportements transversaux comme le traçage distribué.

### Garanties de Livraison

Kafka offre trois niveaux de garanties : at-most-once (perte possible), at-least-once (duplicatas possibles), et exactly-once (ni perte ni duplicata). Le niveau est déterminé par la configuration `acks`, l'activation de l'idempotence, et l'utilisation optionnelle des transactions.

L'idempotence (`enable.idempotence=true`) élimine les duplicatas causés par les retries et devrait être activée par défaut. Les transactions permettent l'écriture atomique cross-partition pour les cas d'usage avancés comme le stream processing exactly-once. Le choix du niveau de garantie dépend des exigences métier — les transactions ne sont justifiées que pour les cas nécessitant l'atomicité.

### Stratégies de Partitionnement

Le partitionnement détermine les garanties d'ordre et la distribution de charge. Le partitionnement par clé garantit l'ordre pour une clé donnée. Le partitionnement sans clé (sticky ou round-robin) optimise la distribution. Des partitionneurs personnalisés permettent des logiques métier spécifiques comme le routage géographique ou l'isolation par tenant.

Le choix de la stratégie de partitionnement est une décision architecturale critique qui doit être documentée et ses implications comprises. La tension entre ordre (invariants métier) et distribution (performance) doit être résolue en faveur des invariants métier dans la plupart des cas.

### Sérialisation et Schémas

La sérialisation convertit les objets en octets pour le transport. Avro avec Schema Registry est recommandé pour les déploiements entreprise : format compact, évolution contrôlée des schémas, validation de compatibilité. Protobuf est une alternative performante quand la compatibilité binaire est importante.

La gestion des schémas requiert une gouvernance : règles de compatibilité (BACKWARD, FORWARD, FULL), processus de revue des changements, documentation des champs. Les changements incompatibles nécessitent des stratégies de migration explicites (double-écriture, topics versionnés).

### Optimisation des Performances

Le débit s'optimise par le batching (`linger.ms`, `batch.size`), la compression, et le parallélisme. La latence s'optimise par la réduction du batching et le choix de `acks`. Ces objectifs sont souvent en tension et requièrent des compromis explicites selon les exigences de l'application.

La compression réduit le volume de données au prix du CPU. LZ4 et Zstd offrent d'excellents compromis pour la plupart des cas d'usage. Le monitoring des métriques du producteur est essentiel pour détecter les problèmes et valider les optimisations.

### Bonnes Pratiques

Les producteurs doivent être des singletons réutilisés, avec fermeture gracieuse. La gestion des erreurs doit être centralisée et les erreurs traitées (log, alerte, retry applicatif, DLQ). La configuration doit être adaptée à la criticité des données.

La résilience requiert l'idempotence, des timeouts appropriés, et des stratégies de circuit breaker pour les situations de panne prolongée. Le monitoring des métriques JMX et la mise en place d'alertes pertinentes permettent de détecter les problèmes avant qu'ils n'impactent les utilisateurs.

Les tests (unitaires avec MockProducer, intégration avec Testcontainers, charge, résilience) valident le comportement avant la production. Un template de producteur bien configuré et documenté devrait être fourni aux équipes de développement comme point de départ standardisé.

---

### Vers le Chapitre Suivant

Ce chapitre a couvert la production de messages — comment les applications publient vers Kafka. Le chapitre suivant, « Création d'Applications Consommatrices », explorera l'autre côté de l'équation : comment les applications lisent et traitent les messages depuis Kafka.

La maîtrise des deux côtés — production et consommation — permet à l'architecte de concevoir des systèmes événementiels complets, de bout en bout.

---

*Volume III : Apache Kafka - Guide de l'Architecte*

*Chapitre III.3 — Clients Kafka et Production de Messages*

*Monographie « L'Entreprise Agentique »*
