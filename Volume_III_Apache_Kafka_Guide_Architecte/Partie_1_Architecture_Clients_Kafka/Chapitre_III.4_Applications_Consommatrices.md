# Chapitre III.4

## CRÉATION D'APPLICATIONS CONSOMMATRICES

---

> *« Un système de messagerie n'est aussi bon que sa capacité à délivrer les messages de manière fiable et efficace aux applications qui les attendent. »*
>
> — Neha Narkhede, Co-créatrice d'Apache Kafka

---

Le chapitre précédent a exploré en détail la production de messages vers Kafka. Ce chapitre examine l'autre côté de l'équation : comment les applications consomment et traitent les messages depuis Kafka. Si la production détermine comment les données entrent dans le système, la consommation détermine comment elles en sortent et créent de la valeur.

Le consommateur Kafka (consumer) est souvent perçu comme le symétrique du producteur — là où l'un envoie, l'autre reçoit. Cette vision simpliste masque une complexité considérable. Le consommateur doit gérer le parallélisme via les groupes de consommateurs, coordonner le rééquilibrage lors des changements de topologie, suivre sa progression via les offsets, gérer les erreurs de traitement, et maintenir des performances optimales sous charge variable.

L'architecte qui maîtrise les concepts développés dans ce chapitre sera capable de concevoir des applications de consommation robustes, scalables et performantes. Il comprendra les compromis entre les différentes stratégies de commit d'offset, saura dimensionner les groupes de consommateurs, et pourra diagnostiquer les problèmes de lag ou de rééquilibrage.

Nous explorerons successivement l'architecture du consommateur, les groupes de consommateurs et le parallélisme, le rééquilibrage, les modèles de conception, les stratégies avancées, l'optimisation des performances, et la construction de consommateurs résilients.

---

## III.4.1 Consommateur Kafka : Architecture et Principes Fondamentaux

### Architecture Interne du Consommateur

Le consommateur Kafka est un client qui lit des messages depuis un ou plusieurs topics. Contrairement à certains systèmes de messagerie où le broker « pousse » les messages vers les consommateurs, Kafka utilise un modèle « pull » où le consommateur demande activement les messages au broker.

**Le modèle pull.** Le consommateur envoie des requêtes « fetch » aux brokers pour récupérer les messages. Ce modèle présente plusieurs avantages : le consommateur contrôle son rythme de consommation (backpressure naturel), il peut relire des messages en repositionnant son offset, et le broker n'a pas besoin de maintenir l'état de chaque consommateur.

**Composants internes.** Le consommateur Kafka se compose de plusieurs sous-systèmes qui collaborent pour récupérer et traiter les messages.

Le **Fetcher** est responsable de l'envoi des requêtes fetch aux brokers et de la réception des réponses. Il maintient des buffers de messages pré-fetchés pour réduire la latence perçue par l'application.

Le **ConsumerCoordinator** gère l'appartenance au groupe de consommateurs, participe au protocole de rééquilibrage, et communique avec le coordinateur de groupe sur le broker.

Le **SubscriptionState** maintient l'état des abonnements (topics, partitions assignées) et la position courante (offset) pour chaque partition.

Le **ConsumerNetworkClient** gère les connexions réseau vers les brokers, le multiplexage des requêtes, et les timeouts.

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Application Thread                           │
│  ┌──────────┐    ┌──────────────┐    ┌─────────────────────────┐   │
│  │  poll()  │───▶│ Fetcher      │───▶│   Deserializers         │   │
│  └──────────┘    └──────────────┘    └───────────┬─────────────┘   │
│                         │                        │                  │
│                         ▼                        ▼                  │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │              ConsumerRecords<K, V>                           │  │
│  └──────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    ConsumerCoordinator                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────────┐ │
│  │ Group       │  │ Heartbeat   │  │ Rebalance                   │ │
│  │ Membership  │  │ Thread      │  │ Protocol                    │ │
│  └─────────────┘  └─────────────┘  └─────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Kafka Brokers                                  │
│              (Group Coordinator + Partition Leaders)                │
└─────────────────────────────────────────────────────────────────────┘
```

> **Définition formelle**
>
> Un **consommateur Kafka** est un client qui s'abonne à un ou plusieurs topics et récupère les messages via des requêtes fetch. Le consommateur maintient un **offset** par partition, représentant la position du prochain message à lire. L'offset est un entier 64 bits monotone croissant, unique par partition.

### La Boucle de Consommation

Le pattern fondamental d'utilisation du consommateur est la boucle de consommation (poll loop). L'application appelle répétitivement `poll()` pour récupérer des lots de messages.

```java
Properties props = new Properties();
props.put("bootstrap.servers", "localhost:9092");
props.put("group.id", "my-consumer-group");
props.put("key.deserializer", "org.apache.kafka.common.serialization.StringDeserializer");
props.put("value.deserializer", "org.apache.kafka.common.serialization.StringDeserializer");

KafkaConsumer<String, String> consumer = new KafkaConsumer<>(props);
consumer.subscribe(Arrays.asList("my-topic"));

try {
    while (true) {
        ConsumerRecords<String, String> records = consumer.poll(Duration.ofMillis(100));
        for (ConsumerRecord<String, String> record : records) {
            processRecord(record);
        }
    }
} finally {
    consumer.close();
}
```

**Comportement de `poll()`.** L'appel `poll(Duration timeout)` effectue plusieurs opérations critiques simultanément. Il envoie des heartbeats au coordinateur de groupe pour maintenir l'appartenance au groupe. Il rafraîchit les métadonnées du cluster si nécessaire (changement de leader, nouvelles partitions). Il fetch de nouveaux messages si le buffer local est insuffisant. Finalement, il retourne les messages disponibles dans un objet `ConsumerRecords`.

Le timeout spécifie combien de temps attendre si aucun message n'est disponible. Un timeout de 0 retourne immédiatement (avec ou sans messages). Un timeout long bloque jusqu'à ce que des messages soient disponibles ou que le timeout expire. Le choix du timeout impacte la réactivité de l'application et la fréquence des heartbeats.

**Thread safety.** Le consommateur Kafka n'est **pas thread-safe**. Tous les appels doivent être effectués depuis le même thread. Cette contrainte de conception est volontaire : elle simplifie l'implémentation et évite les problèmes de synchronisation coûteux. L'exception est `wakeup()` qui peut être appelé depuis un autre thread pour interrompre un `poll()` bloquant, utile pour l'arrêt gracieux.

**Structure de ConsumerRecords.** L'objet retourné par `poll()` contient les messages groupés par partition. Cela permet un traitement optimisé :

```java
ConsumerRecords<String, String> records = consumer.poll(Duration.ofMillis(100));

// Itération sur tous les records
for (ConsumerRecord<String, String> record : records) {
    processRecord(record);
}

// Ou itération par partition (utile pour le commit par partition)
for (TopicPartition partition : records.partitions()) {
    List<ConsumerRecord<String, String>> partitionRecords = records.records(partition);
    for (ConsumerRecord<String, String> record : partitionRecords) {
        processRecord(record);
    }
    // Commit pour cette partition spécifiquement
    long lastOffset = partitionRecords.get(partitionRecords.size() - 1).offset();
    consumer.commitSync(Collections.singletonMap(partition, 
        new OffsetAndMetadata(lastOffset + 1)));
}
```

> **Note de terrain**
>
> *Contexte* : Application de traitement de logs avec exigence de monitoring précis par source.
>
> *Problème initial* : Traitement de tous les records en vrac, impossible de savoir quelle partition/source pose problème.
>
> *Solution* : Itération par partition avec métriques séparées. Chaque partition correspond à une source de logs distincte. Le monitoring par partition permet d'identifier rapidement les sources problématiques.
>
> *Bonus* : Le commit par partition permet une granularité fine — si une partition a des erreurs, les autres peuvent continuer à progresser.

### Gestion des Offsets

L'**offset** est le mécanisme par lequel le consommateur suit sa progression dans chaque partition. Comprendre la gestion des offsets est crucial pour garantir le traitement correct des messages.

**Offset courant vs. offset commité.** Le consommateur maintient deux notions d'offset par partition :
- L'**offset courant** (position) est l'offset du prochain message à lire. Il avance automatiquement après chaque `poll()`.
- L'**offset commité** est le dernier offset persisté, indiquant jusqu'où le traitement est confirmé.

La différence entre ces deux offsets est importante lors des redémarrages. Si le consommateur crashe, il reprendra depuis le dernier offset commité, pas depuis l'offset courant. Les messages entre l'offset commité et l'offset courant seront retraités.

**Commit automatique.** Par défaut (`enable.auto.commit=true`), le consommateur commite automatiquement les offsets périodiquement (`auto.commit.interval.ms`, défaut 5 secondes). Ce mode est simple mais peut causer des pertes ou des duplicatas :
- Si le consommateur crashe après avoir traité des messages mais avant le commit automatique, ces messages seront retraités (duplicatas).
- Si le commit automatique se produit avant que le traitement ne soit terminé et que le consommateur crashe, des messages peuvent être perdus.

**Commit manuel.** Pour un contrôle précis, désactiver le commit automatique (`enable.auto.commit=false`) et commiter explicitement :

```java
// Commit synchrone - bloque jusqu'à confirmation
consumer.commitSync();

// Commit asynchrone - retourne immédiatement
consumer.commitAsync((offsets, exception) -> {
    if (exception != null) {
        log.error("Commit failed", exception);
    }
});

// Commit d'offsets spécifiques
Map<TopicPartition, OffsetAndMetadata> offsets = new HashMap<>();
offsets.put(new TopicPartition("topic", 0), new OffsetAndMetadata(lastOffset + 1));
consumer.commitSync(offsets);
```

> **Exemple concret**
>
> *Scénario* : Un consommateur traite des commandes. Il lit 10 messages (offsets 100-109), traite les 5 premiers avec succès, puis crashe.
>
> *Avec commit automatique (défaut)* : Si le commit automatique s'est produit après avoir lu les 10 messages, l'offset commité est 110. Au redémarrage, le consommateur reprend à 110 — les messages 105-109 non traités sont perdus.
>
> *Avec commit manuel après chaque message* : L'offset commité est 105 (dernier message traité + 1). Au redémarrage, le consommateur reprend à 105 — les messages 105-109 seront retraités.
>
> *Leçon* : Le commit automatique est dangereux pour les traitements critiques. Préférer le commit manuel après traitement réussi.

### Configuration Fondamentale

Les paramètres de configuration du consommateur contrôlent son comportement. L'architecte doit comprendre les paramètres clés.

**`bootstrap.servers`** : Liste des brokers pour la découverte initiale du cluster.

**`group.id`** : Identifiant du groupe de consommateurs. Obligatoire pour la consommation avec groupes. Les consommateurs avec le même `group.id` partagent les partitions.

**`key.deserializer` et `value.deserializer`** : Classes de désérialisation pour convertir les octets en objets.

**`enable.auto.commit`** (défaut true) : Active le commit automatique des offsets.

**`auto.commit.interval.ms`** (défaut 5000) : Intervalle entre les commits automatiques.

**`auto.offset.reset`** (défaut latest) : Comportement quand aucun offset commité n'existe ou que l'offset est invalide. Les valeurs sont `earliest` (début de la partition), `latest` (fin de la partition), ou `none` (exception).

**`max.poll.records`** (défaut 500) : Nombre maximal de messages retournés par `poll()`.

**`max.poll.interval.ms`** (défaut 300000) : Intervalle maximal entre deux appels `poll()` avant que le consommateur soit considéré comme mort.

**`session.timeout.ms`** (défaut 45000) : Timeout de session avec le coordinateur. Si aucun heartbeat n'est reçu dans ce délai, le consommateur est éjecté du groupe.

**`fetch.min.bytes`** (défaut 1) : Taille minimale de données à retourner. Le broker attend d'avoir au moins cette quantité avant de répondre.

**`fetch.max.wait.ms`** (défaut 500) : Temps maximal d'attente du broker si `fetch.min.bytes` n'est pas atteint.

---

## III.4.2 Atteindre le Parallélisme : Groupes de Consommateurs

### Le Concept de Groupe de Consommateurs

Un **groupe de consommateurs** (consumer group) est un ensemble de consommateurs qui collaborent pour consommer un topic. Les partitions du topic sont distribuées entre les membres du groupe, permettant un traitement parallèle.

**Principe fondamental.** Chaque partition est assignée à exactement un consommateur du groupe à un instant donné. Un consommateur peut être assigné à plusieurs partitions, mais une partition ne peut avoir qu'un seul consommateur dans un groupe donné.

Cette règle garantit que les messages d'une partition sont traités dans l'ordre par un seul consommateur, préservant les garanties d'ordre de Kafka. C'est une propriété fondamentale qui distingue Kafka des systèmes de messagerie traditionnels où les messages peuvent être distribués à n'importe quel worker disponible.

**Parallélisme maximal.** Le nombre maximal de consommateurs actifs dans un groupe est égal au nombre de partitions du topic. Si un groupe a plus de consommateurs que de partitions, les consommateurs excédentaires restent inactifs (idle), attendant qu'une partition se libère.

Cette limite implique que le parallélisme doit être planifié dès la création du topic. Un topic avec 4 partitions ne peut pas avoir plus de 4 consommateurs actifs simultanément, quelle que soit la puissance des machines ou le nombre d'instances déployées.

```
Topic avec 4 partitions :

Groupe A (2 consommateurs) :
┌─────────────┐     ┌─────────────┐
│ Consumer A1 │     │ Consumer A2 │
│ P0, P1      │     │ P2, P3      │
└─────────────┘     └─────────────┘

Groupe B (4 consommateurs) :
┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
│ Consumer B1 │ │ Consumer B2 │ │ Consumer B3 │ │ Consumer B4 │
│ P0          │ │ P1          │ │ P2          │ │ P3          │
└─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘

Groupe C (6 consommateurs) :
┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
│ Consumer C1 │ │ Consumer C2 │ │ Consumer C3 │ │ Consumer C4 │ │ Consumer C5 │ │ Consumer C6 │
│ P0          │ │ P1          │ │ P2          │ │ P3          │ │ (idle)      │ │ (idle)      │
└─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘
```

> **Définition formelle**
>
> Un **groupe de consommateurs** est identifié par un `group.id` unique. Tous les consommateurs partageant le même `group.id` forment un groupe et se partagent les partitions des topics auxquels ils sont abonnés. Le **coordinateur de groupe** (group coordinator) est un broker responsable de la gestion des membres du groupe et de l'orchestration des rééquilibrages.

### Indépendance des Groupes

Les groupes de consommateurs sont complètement indépendants les uns des autres. Chaque groupe maintient ses propres offsets et sa propre progression dans les topics. Cette indépendance est fondamentale pour l'architecture événementielle.

**Implications de l'indépendance :**

*Pas d'interférence.* Un groupe qui prend du retard n'impacte pas les autres groupes. Chaque groupe avance à son propre rythme.

*Offsets séparés.* Les offsets sont stockés par groupe dans le topic `__consumer_offsets`. Le groupe A peut être à l'offset 1000 tandis que le groupe B est à l'offset 5000.

*Consommation multiple.* Le même message peut être consommé par plusieurs groupes. Kafka ne supprime pas les messages après consommation (contrairement à une file traditionnelle).

Cette indépendance permet plusieurs patterns architecturaux puissants :

**Multiples applications consommant le même topic.** Chaque application a son propre `group.id` et consomme tous les messages indépendamment. C'est le pattern classique pour diffuser des événements à plusieurs systèmes — le système de facturation, le système de notification, et le système d'analytique peuvent tous consommer les événements de commande.

**Scaling horizontal d'une application.** Toutes les instances d'une même application partagent le même `group.id` et se répartissent les partitions. C'est le pattern pour augmenter le débit de traitement sans modifier le code applicatif.

**Environnements de test.** Un groupe de test peut consommer le même topic qu'un groupe de production sans interférence. Utile pour valider des changements avec des données réelles.

**Replay et retraitement.** Un nouveau groupe peut consommer un topic depuis le début (`auto.offset.reset=earliest`) pour reconstruire un système downstream ou effectuer une analyse historique.

> **Exemple concret**
>
> *Scénario* : Un topic `orders.created` reçoit tous les événements de création de commande.
>
> *Groupe 1 : fulfillment-service* : Déclenche la préparation des commandes. Besoin de traitement rapide.
>
> *Groupe 2 : analytics-pipeline* : Alimente un data warehouse. Peut tolérer quelques minutes de retard.
>
> *Groupe 3 : fraud-detection* : Analyse en temps réel pour détecter les fraudes. Priorité maximale sur la latence.
>
> *Groupe 4 : audit-service* : Archive toutes les commandes pour conformité. Consomme depuis le début du topic.
>
> Ces quatre groupes consomment le même topic simultanément, chacun avec ses propres exigences et sa propre progression.

### Le Coordinateur de Groupe

Le **coordinateur de groupe** (group coordinator) est un broker désigné pour gérer un groupe de consommateurs spécifique. Le choix du coordinateur est déterministe basé sur le hash du `group.id`.

**Responsabilités du coordinateur :**
- Maintenir la liste des membres actifs du groupe
- Détecter les défaillances via les heartbeats
- Orchestrer les rééquilibrages
- Stocker les offsets commités dans le topic `__consumer_offsets`

**Élection du leader du groupe.** Parmi les membres du groupe, un est élu « leader ». Le leader est responsable de calculer l'assignation des partitions lors d'un rééquilibrage. Le coordinateur exécute l'assignation calculée par le leader.

### Stratégies d'Assignation

La stratégie d'assignation détermine comment les partitions sont distribuées entre les consommateurs. Plusieurs stratégies sont disponibles via `partition.assignment.strategy`.

**RangeAssignor (défaut historique).** Assigne des plages contiguës de partitions à chaque consommateur. Peut créer des déséquilibres si le nombre de partitions n'est pas divisible par le nombre de consommateurs.

**RoundRobinAssignor.** Distribue les partitions en round-robin entre les consommateurs. Plus équilibré que Range mais peut séparer les partitions d'un même topic entre plusieurs consommateurs.

**StickyAssignor.** Tente de préserver les assignations existantes lors des rééquilibrages tout en maintenant l'équilibre. Réduit le nombre de partitions qui changent de propriétaire.

**CooperativeStickyAssignor (recommandé).** Combinaison de StickyAssignor avec le protocole de rééquilibrage coopératif. Minimise les interruptions lors des rééquilibrages.

| Stratégie | Équilibre | Stabilité | Interruption |
|-----------|-----------|-----------|--------------|
| RangeAssignor | Moyen | Faible | Totale |
| RoundRobinAssignor | Bon | Faible | Totale |
| StickyAssignor | Bon | Bonne | Totale |
| CooperativeStickyAssignor | Bon | Bonne | Minimale |

> **Décision architecturale**
>
> *Contexte* : Choix de la stratégie d'assignation pour un groupe de consommateurs traitant des commandes critiques.
>
> *Exigences* : Minimiser les interruptions lors des déploiements, maintenir un équilibre de charge.
>
> *Options* :
> 1. RangeAssignor (défaut) : Simple mais rééquilibrages disruptifs.
> 2. StickyAssignor : Moins de mouvements de partitions mais rééquilibrages bloquants.
> 3. CooperativeStickyAssignor : Rééquilibrages non-bloquants, stabilité.
>
> *Décision* : CooperativeStickyAssignor — le rééquilibrage coopératif permet aux consommateurs non affectés de continuer à traiter pendant le rééquilibrage.
>
> *Configuration* :
> ```properties
> partition.assignment.strategy=org.apache.kafka.clients.consumer.CooperativeStickyAssignor
> ```

### Dimensionnement des Groupes

Le dimensionnement du groupe de consommateurs est une décision architecturale importante qui impacte le débit, la latence, et la résilience.

**Facteurs à considérer :**

*Débit requis.* Si un seul consommateur ne peut pas traiter le débit du topic, plus de consommateurs sont nécessaires. Mesurer le débit de traitement d'un consommateur et comparer au débit de production.

*Nombre de partitions.* Le parallélisme maximal est limité par le nombre de partitions. Avoir plus de consommateurs que de partitions est inutile.

*Latence de traitement.* Plus de consommateurs parallèles peut réduire la latence globale en distribuant la charge.

*Coût et ressources.* Chaque consommateur consomme des ressources (mémoire, connexions réseau, CPU). Équilibrer le besoin de performance avec le coût.

*Résilience.* Avoir des consommateurs en excès (standby) permet une reprise rapide en cas de défaillance.

**Règle empirique.** Commencer avec un nombre de consommateurs égal au nombre de partitions divisé par 2, puis ajuster basé sur les métriques de lag et de débit.

---

## III.4.3 Maîtriser le Rééquilibrage des Consommateurs

### Anatomie d'un Rééquilibrage

Le **rééquilibrage** (rebalance) est le processus par lequel les partitions sont redistribuées entre les consommateurs d'un groupe. Il se produit lors de changements de membership (nouveau consommateur, départ, crash) ou de changements de souscription (ajout/suppression de topics).

**Déclencheurs de rééquilibrage :**
- Un nouveau consommateur rejoint le groupe
- Un consommateur quitte le groupe (fermeture gracieuse)
- Un consommateur est considéré mort (timeout de session ou de poll)
- Le nombre de partitions d'un topic souscrit change
- Un consommateur change sa souscription

**Protocole de rééquilibrage (avant le rééquilibrage coopératif) :**

1. **JoinGroup** : Les consommateurs envoient une requête JoinGroup au coordinateur avec leurs souscriptions et les stratégies d'assignation supportées.

2. **Synchronisation** : Le coordinateur choisit un leader parmi les membres. Le leader calcule l'assignation.

3. **SyncGroup** : Le leader envoie l'assignation au coordinateur. Tous les membres récupèrent leur assignation via SyncGroup.

4. **Reprise** : Les consommateurs commencent à consommer leurs partitions assignées.

**Le problème du « stop-the-world ».** Dans le protocole classique (Eager), tous les consommateurs arrêtent de consommer pendant le rééquilibrage. Même les consommateurs dont les partitions ne changent pas sont interrompus. Pour les grands groupes, ce « stop-the-world » peut durer plusieurs secondes.

### Rééquilibrage Coopératif (Incrémental)

Le **rééquilibrage coopératif** (Kafka 2.4+) améliore significativement l'expérience en permettant aux consommateurs de continuer à traiter leurs partitions non affectées pendant le rééquilibrage.

**Principe.** Au lieu de révoquer toutes les partitions au début du rééquilibrage, seules les partitions qui doivent changer de propriétaire sont révoquées. Les autres consommateurs continuent normalement.

**Déroulement :**

1. **Premier rééquilibrage** : L'assignation cible est calculée. Les partitions à transférer sont identifiées et révoquées de leurs propriétaires actuels.

2. **Deuxième rééquilibrage** : Les partitions révoquées sont assignées à leurs nouveaux propriétaires.

3. **Continuation** : Les consommateurs non affectés n'ont jamais arrêté de traiter.

**Activation.** Utiliser `CooperativeStickyAssignor` comme stratégie d'assignation :

```java
props.put("partition.assignment.strategy", 
          "org.apache.kafka.clients.consumer.CooperativeStickyAssignor");
```

> **Note de terrain**
>
> *Contexte* : Application de traitement de commandes avec 20 consommateurs et 100 partitions. Déploiements fréquents (plusieurs fois par jour).
>
> *Problème avec le protocole Eager* : Chaque déploiement (rolling restart) déclenche ~20 rééquilibrages, chacun causant une interruption de 5-10 secondes. Au total, ~2 minutes d'indisponibilité par déploiement.
>
> *Solution* : Migration vers CooperativeStickyAssignor.
>
> *Résultat* : Les rééquilibrages sont quasi-transparents. Seules les partitions du consommateur redémarré sont temporairement non traitées (~2 secondes). Les 19 autres consommateurs continuent sans interruption.
>
> *Leçon* : Le rééquilibrage coopératif est essentiel pour les groupes de consommateurs de production. Migrer dès que possible.

### Optimisation du Rééquilibrage

Même avec le rééquilibrage coopératif, certaines optimisations réduisent l'impact des rééquilibrages.

**Assignation statique (`group.instance.id`).** En assignant un identifiant d'instance statique à chaque consommateur, Kafka peut reconnaître un consommateur qui redémarre et lui réassigner les mêmes partitions sans rééquilibrage complet.

```java
props.put("group.instance.id", "consumer-instance-1");
```

Avec un `group.instance.id`, le consommateur a un délai de grâce (`session.timeout.ms`) pour redémarrer avant qu'un rééquilibrage ne soit déclenché.

**Réduction des timeouts.** Des timeouts plus courts permettent une détection plus rapide des consommateurs morts, mais augmentent le risque de faux positifs (consommateur temporairement lent considéré comme mort).

**Heartbeats fréquents.** Configurer `heartbeat.interval.ms` à environ 1/3 de `session.timeout.ms` pour une détection fiable.

**Traitement rapide dans `poll()`.** Si le traitement entre deux `poll()` dépasse `max.poll.interval.ms`, le consommateur est éjecté. Soit traiter plus rapidement, soit réduire `max.poll.records`.

### Gestion des Callbacks de Rééquilibrage

Le consommateur peut être notifié des rééquilibrages via un `ConsumerRebalanceListener`. Ces callbacks permettent d'effectuer des actions avant et après le rééquilibrage.

```java
consumer.subscribe(Arrays.asList("topic"), new ConsumerRebalanceListener() {
    @Override
    public void onPartitionsRevoked(Collection<TopicPartition> partitions) {
        // Appelé avant que les partitions soient révoquées
        // Commiter les offsets, fermer les ressources
        log.info("Partitions révoquées: {}", partitions);
        consumer.commitSync();
    }
    
    @Override
    public void onPartitionsAssigned(Collection<TopicPartition> partitions) {
        // Appelé après que les nouvelles partitions sont assignées
        // Initialiser les ressources, éventuellement seek
        log.info("Partitions assignées: {}", partitions);
    }
    
    @Override
    public void onPartitionsLost(Collection<TopicPartition> partitions) {
        // Appelé quand les partitions sont perdues (pas de chance de commit)
        // Avec rééquilibrage coopératif uniquement
        log.warn("Partitions perdues: {}", partitions);
    }
});
```

**Cas d'usage des callbacks :**

*Commit avant révocation.* Commiter les offsets des messages traités avant de perdre les partitions pour éviter le retraitement.

*Nettoyage de ressources.* Fermer les connexions, fichiers, ou caches associés aux partitions révoquées.

*Initialisation de ressources.* Ouvrir des connexions ou charger des caches pour les nouvelles partitions.

*Positionnement personnalisé.* Après assignation, utiliser `seek()` pour repositionner l'offset si nécessaire (ex: reprise depuis un checkpoint externe).

---

## III.4.4 Modèles de Conception Fondamentaux

### Pattern : Un Thread par Consommateur

Le pattern le plus simple et le plus courant : chaque consommateur s'exécute dans son propre thread.

```java
public class SingleThreadConsumer implements Runnable {
    private final KafkaConsumer<String, String> consumer;
    private final AtomicBoolean running = new AtomicBoolean(true);
    
    public SingleThreadConsumer(Properties props) {
        this.consumer = new KafkaConsumer<>(props);
        consumer.subscribe(Arrays.asList("topic"));
    }
    
    @Override
    public void run() {
        try {
            while (running.get()) {
                ConsumerRecords<String, String> records = consumer.poll(Duration.ofMillis(100));
                for (ConsumerRecord<String, String> record : records) {
                    process(record);
                }
                consumer.commitSync();
            }
        } finally {
            consumer.close();
        }
    }
    
    public void shutdown() {
        running.set(false);
        consumer.wakeup();
    }
}

// Utilisation
ExecutorService executor = Executors.newFixedThreadPool(numConsumers);
for (int i = 0; i < numConsumers; i++) {
    executor.submit(new SingleThreadConsumer(createProps()));
}
```

**Avantages :**
- Simple à implémenter et comprendre
- Respect naturel du modèle single-threaded du consommateur Kafka
- Ordre de traitement préservé par partition

**Inconvénients :**
- Un seul thread de traitement par consommateur
- Si le traitement est lent, le consommateur ne peut pas suivre
- Pas de parallélisation du traitement au sein d'un consommateur

### Pattern : Découplage Fetch et Traitement

Pour les traitements lourds, découpler le fetch des messages de leur traitement permet de paralléliser le traitement.

```java
public class DecoupledConsumer {
    private final KafkaConsumer<String, String> consumer;
    private final ExecutorService processingPool;
    private final BlockingQueue<ConsumerRecord<String, String>> queue;
    private final AtomicBoolean running = new AtomicBoolean(true);
    
    public DecoupledConsumer(Properties props, int numProcessors) {
        this.consumer = new KafkaConsumer<>(props);
        this.processingPool = Executors.newFixedThreadPool(numProcessors);
        this.queue = new LinkedBlockingQueue<>(1000);
        consumer.subscribe(Arrays.asList("topic"));
        
        // Démarrer les workers de traitement
        for (int i = 0; i < numProcessors; i++) {
            processingPool.submit(this::processRecords);
        }
    }
    
    public void fetchLoop() {
        try {
            while (running.get()) {
                ConsumerRecords<String, String> records = consumer.poll(Duration.ofMillis(100));
                for (ConsumerRecord<String, String> record : records) {
                    queue.put(record); // Peut bloquer si la queue est pleine
                }
            }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        } finally {
            consumer.close();
        }
    }
    
    private void processRecords() {
        while (running.get() || !queue.isEmpty()) {
            try {
                ConsumerRecord<String, String> record = queue.poll(100, TimeUnit.MILLISECONDS);
                if (record != null) {
                    process(record);
                }
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                break;
            }
        }
    }
}
```

**Avantages :**
- Traitement parallèle des messages
- Le fetch n'est pas bloqué par le traitement

**Inconvénients :**
- L'ordre de traitement n'est plus garanti
- La gestion des offsets est complexe (quand commiter ?)
- Risque de perte de messages si la queue en mémoire est perdue

> **Anti-patron**
>
> *« Nous traitons les messages dans un thread pool et commitons immédiatement après le fetch. »* Ce pattern est dangereux : si le traitement échoue après le commit, les messages sont perdus. Si l'application crashe avec des messages dans la queue en mémoire, ils sont perdus.
>
> *Meilleure approche* : Si le traitement parallèle est nécessaire et que la perte n'est pas acceptable, utiliser un système de tracking externe (base de données) pour suivre les messages traités, ou accepter les duplicatas avec un traitement idempotent.

### Pattern : Pause et Resume

Le consommateur peut mettre en pause et reprendre la consommation de partitions spécifiques. Ce pattern est utile pour la gestion de backpressure.

```java
public class BackpressureConsumer {
    private final KafkaConsumer<String, String> consumer;
    private final BlockingQueue<ConsumerRecord<String, String>> buffer;
    private static final int HIGH_WATERMARK = 1000;
    private static final int LOW_WATERMARK = 200;
    
    public void run() {
        while (running.get()) {
            ConsumerRecords<String, String> records = consumer.poll(Duration.ofMillis(100));
            
            // Ajouter au buffer
            for (ConsumerRecord<String, String> record : records) {
                buffer.add(record);
            }
            
            // Gestion du backpressure
            if (buffer.size() > HIGH_WATERMARK) {
                consumer.pause(consumer.assignment());
                log.info("Buffer plein, pause de la consommation");
            } else if (buffer.size() < LOW_WATERMARK) {
                consumer.resume(consumer.assignment());
            }
        }
    }
}
```

**Cas d'usage :**
- Limiter la pression sur un système downstream lent
- Gérer des pics de charge temporaires
- Implémenter une consommation à débit contrôlé

### Pattern : Assignation Manuelle

Au lieu de s'abonner à un topic et de laisser Kafka gérer l'assignation, le consommateur peut demander des partitions spécifiques.

```java
// Assignation manuelle
TopicPartition partition0 = new TopicPartition("topic", 0);
TopicPartition partition1 = new TopicPartition("topic", 1);
consumer.assign(Arrays.asList(partition0, partition1));

// Avec seek pour repositionnement
consumer.seek(partition0, 0); // Début de la partition
consumer.seekToEnd(Arrays.asList(partition1)); // Fin de la partition
```

**Cas d'usage :**
- Replay de données spécifiques
- Migration ou réparation de données
- Consommation sans groupe (pas de coordination)
- Tests et débogage

**Attention.** Avec l'assignation manuelle, il n'y a pas de groupe de consommateurs, pas de rééquilibrage automatique, et pas de stockage automatique des offsets dans `__consumer_offsets`. L'application est responsable de tout.

### Pattern : At-Least-Once avec Idempotence

Pour garantir at-least-once sans perdre de messages, commiter les offsets après le traitement réussi et rendre le traitement idempotent pour gérer les duplicatas.

```java
public void processWithIdempotence() {
    while (running.get()) {
        ConsumerRecords<String, String> records = consumer.poll(Duration.ofMillis(100));
        
        for (ConsumerRecord<String, String> record : records) {
            String messageId = extractMessageId(record);
            
            // Vérifier si déjà traité
            if (processedMessages.contains(messageId)) {
                log.debug("Message {} déjà traité, skip", messageId);
                continue;
            }
            
            // Traiter
            process(record);
            
            // Marquer comme traité
            processedMessages.add(messageId);
        }
        
        // Commiter après traitement
        consumer.commitSync();
    }
}
```

**Implémentation de l'idempotence :**
- Stocker les IDs des messages traités dans une base de données ou un cache
- Utiliser des opérations naturellement idempotentes (UPSERT, PUT)
- Inclure un ID unique dans chaque message côté producteur

---

## III.4.5 Stratégies de Consommation Avancées

### Exactly-Once Semantic avec Transactions

Pour atteindre l'exactly-once de bout en bout dans un pipeline Kafka (consommation → transformation → production), utiliser les transactions côté consommateur en conjonction avec un producteur transactionnel. Cette approche garantit que le traitement d'un message et la production de ses résultats sont atomiques.

```java
// Configuration du consommateur
Properties consumerProps = new Properties();
consumerProps.put("bootstrap.servers", "localhost:9092");
consumerProps.put("group.id", "exactly-once-processor");
consumerProps.put("isolation.level", "read_committed");
consumerProps.put("enable.auto.commit", "false");

// Configuration du producteur transactionnel
Properties producerProps = new Properties();
producerProps.put("bootstrap.servers", "localhost:9092");
producerProps.put("transactional.id", "processor-txn-1");
producerProps.put("enable.idempotence", "true");

KafkaConsumer<String, String> consumer = new KafkaConsumer<>(consumerProps);
KafkaProducer<String, String> producer = new KafkaProducer<>(producerProps);

consumer.subscribe(Arrays.asList("input-topic"));
producer.initTransactions();

while (running.get()) {
    ConsumerRecords<String, String> records = consumer.poll(Duration.ofMillis(100));
    
    if (!records.isEmpty()) {
        producer.beginTransaction();
        try {
            for (ConsumerRecord<String, String> record : records) {
                // Transformer et produire
                String transformedValue = transform(record.value());
                ProducerRecord<String, String> output = new ProducerRecord<>(
                    "output-topic", 
                    record.key(), 
                    transformedValue
                );
                producer.send(output);
            }
            
            // Commiter les offsets consommés dans la même transaction
            Map<TopicPartition, OffsetAndMetadata> offsetsToCommit = new HashMap<>();
            for (TopicPartition partition : records.partitions()) {
                List<ConsumerRecord<String, String>> partitionRecords = records.records(partition);
                long lastOffset = partitionRecords.get(partitionRecords.size() - 1).offset();
                offsetsToCommit.put(partition, new OffsetAndMetadata(lastOffset + 1));
            }
            producer.sendOffsetsToTransaction(offsetsToCommit, consumer.groupMetadata());
            
            producer.commitTransaction();
        } catch (ProducerFencedException | OutOfOrderSequenceException e) {
            // Erreur fatale - le producteur doit être recréé
            throw e;
        } catch (KafkaException e) {
            producer.abortTransaction();
        }
    }
}
```

**Points clés de l'exactly-once :**

*`isolation.level=read_committed`* : Le consommateur ne voit que les messages de transactions commitées. Les messages de transactions en cours ou abandonnées sont filtrés automatiquement.

*Les offsets sont commités dans la transaction* : L'appel `sendOffsetsToTransaction()` inclut les offsets consommés dans la transaction en cours. Si la transaction est abandonnée, les offsets ne sont pas commités, et les messages seront relus.

*Atomicité garantie* : Soit tous les messages de sortie sont produits ET les offsets sont commités, soit rien ne se passe. Il n'y a pas d'état intermédiaire visible.

**Limitations et considérations :**

L'exactly-once transactionnel ajoute une latence significative (coordination avec le transaction coordinator). Il est justifié pour les traitements critiques mais peut être excessif pour les pipelines à haute performance où l'at-least-once avec idempotence suffit.

Le `transactional.id` doit être unique par instance de processeur. En cas de scaling, chaque nouvelle instance a besoin de son propre ID.

### Consommation Multi-Topic

Un consommateur peut s'abonner à plusieurs topics simultanément, soit en les listant explicitement, soit via un pattern regex. Cette capacité est puissante pour les architectures où un service doit réagir à plusieurs types d'événements.

```java
// Liste explicite - quand les topics sont connus à l'avance
consumer.subscribe(Arrays.asList("orders", "payments", "shipments"));

// Pattern regex - pour les topics dynamiques
// Consomme tous les topics commençant par "events-"
consumer.subscribe(Pattern.compile("events-.*"));

// Pattern avec région
// Consomme events-eu-*, events-us-*, etc.
consumer.subscribe(Pattern.compile("events-[a-z]{2}-.*"));
```

**Considérations pour le multi-topic :**

*Distribution des partitions* : Les partitions de tous les topics sont distribuées entre les membres du groupe comme s'il s'agissait d'un seul topic. Un consommateur peut recevoir des partitions de différents topics.

*Routing du traitement* : Un seul `poll()` peut retourner des messages de différents topics. Le traitement doit router les messages vers la logique appropriée.

```java
ConsumerRecords<String, String> records = consumer.poll(Duration.ofMillis(100));

for (ConsumerRecord<String, String> record : records) {
    switch (record.topic()) {
        case "orders":
            handleOrder(record);
            break;
        case "payments":
            handlePayment(record);
            break;
        case "shipments":
            handleShipment(record);
            break;
        default:
            // Pour les patterns regex, gérer les topics inconnus
            if (record.topic().startsWith("events-")) {
                handleGenericEvent(record);
            } else {
                log.warn("Topic inattendu: {}", record.topic());
            }
    }
}
```

*Pattern regex et découverte dynamique* : Avec un pattern regex, le consommateur découvre les nouveaux topics périodiquement (`metadata.max.age.ms`, défaut 5 minutes). Un nouveau topic matching le pattern sera automatiquement ajouté à la souscription.

> **Note de terrain**
>
> *Contexte* : Plateforme multi-tenant où chaque tenant a son propre topic `events-{tenant-id}`.
>
> *Approche initiale* : Un consommateur dédié par tenant. Résultat : 500 consommateurs pour 500 tenants, overhead de ressources massif.
>
> *Solution* : Un groupe de consommateurs avec pattern `events-.*`. Tous les topics tenant sont consommés par le même groupe, les partitions distribuées entre ~20 consommateurs.
>
> *Bénéfices* : Réduction de 95% des ressources. Scaling automatique quand de nouveaux tenants sont ajoutés. Monitoring centralisé.
>
> *Attention* : Le routing doit extraire le tenant-id du nom de topic pour appliquer la logique appropriée.

### Seek et Replay

Le consommateur peut repositionner son offset pour relire des messages ou sauter en avant. Cette capacité est fondamentale pour plusieurs cas d'usage opérationnels.

```java
// Repositionner au début - rejouer tout l'historique
consumer.seekToBeginning(consumer.assignment());

// Repositionner à la fin - ignorer l'historique
consumer.seekToEnd(consumer.assignment());

// Repositionner à un offset spécifique
consumer.seek(new TopicPartition("topic", 0), 1000);

// Repositionner à un timestamp - rejouer depuis une date
Map<TopicPartition, Long> timestampsToSearch = new HashMap<>();
Instant targetTime = Instant.now().minus(Duration.ofHours(1));

for (TopicPartition partition : consumer.assignment()) {
    timestampsToSearch.put(partition, targetTime.toEpochMilli());
}

Map<TopicPartition, OffsetAndTimestamp> offsets = consumer.offsetsForTimes(timestampsToSearch);
for (Map.Entry<TopicPartition, OffsetAndTimestamp> entry : offsets.entrySet()) {
    if (entry.getValue() != null) {
        consumer.seek(entry.getKey(), entry.getValue().offset());
        log.info("Partition {} repositionnée à offset {} (timestamp {})",
            entry.getKey(), entry.getValue().offset(), targetTime);
    }
}
```

**Cas d'usage du seek :**

*Replay après correction de bug* : Un bug a causé un traitement incorrect des messages des dernières 24 heures. Après correction, repositionner les offsets pour retraiter ces messages.

*Reconstruction d'un système downstream* : Une base de données dérivée est corrompue. Repositionner au début du topic pour reconstruire l'état complet.

*Saut de messages corrompus* : Des messages malformés bloquent le traitement. Repositionner après le segment corrompu pour continuer.

*Reprise depuis un checkpoint externe* : L'état de traitement est persisté dans une base externe plutôt que dans Kafka. Au démarrage, repositionner selon le checkpoint.

> **Décision architecturale**
>
> *Contexte* : Système de projection CQRS où l'état est reconstruit depuis les événements.
>
> *Options pour le checkpoint* :
> 1. *Offsets Kafka uniquement* : Simple, mais la reconstruction nécessite de rejouer tout le topic.
> 2. *Snapshots périodiques + offsets* : Snapshot de l'état toutes les heures, position offset sauvegardée avec le snapshot.
>
> *Décision* : Option 2. Au démarrage, charger le dernier snapshot et `seek()` à l'offset correspondant. La reconstruction ne rejoue que les événements depuis le snapshot.
>
> *Bénéfice* : Temps de démarrage réduit de 99% (5 minutes vs 8 heures pour un topic de 1 an).

### Consommation avec Dead Letter Queue

Quand un message ne peut pas être traité après plusieurs tentatives (erreur de désérialisation, erreur métier, dépendance indisponible), le router vers une Dead Letter Queue (DLQ) permet de continuer le traitement des autres messages sans blocage.

```java
public class DLQEnabledConsumer {
    private final KafkaConsumer<String, String> consumer;
    private final KafkaProducer<String, String> dlqProducer;
    private final int maxRetries = 3;
    
    public void consumeWithDLQ() {
        while (running.get()) {
            ConsumerRecords<String, String> records = consumer.poll(Duration.ofMillis(100));
            
            for (ConsumerRecord<String, String> record : records) {
                boolean processed = processWithRetry(record);
                
                if (!processed) {
                    sendToDLQ(record, lastException);
                }
            }
            
            consumer.commitSync();
        }
    }
    
    private boolean processWithRetry(ConsumerRecord<String, String> record) {
        int attempts = 0;
        while (attempts < maxRetries) {
            try {
                process(record);
                return true;
            } catch (RetriableException e) {
                attempts++;
                lastException = e;
                try {
                    Thread.sleep(100 * (long) Math.pow(2, attempts)); // Backoff exponentiel
                } catch (InterruptedException ie) {
                    Thread.currentThread().interrupt();
                    return false;
                }
            } catch (NonRetriableException e) {
                lastException = e;
                return false; // Pas de retry, envoyer directement en DLQ
            }
        }
        return false;
    }
    
    private void sendToDLQ(ConsumerRecord<String, String> record, Exception error) {
        String dlqTopic = record.topic() + ".dlq";
        
        ProducerRecord<String, String> dlqRecord = new ProducerRecord<>(
            dlqTopic,
            record.key(),
            record.value()
        );
        
        // Métadonnées pour le diagnostic et le rejeu
        dlqRecord.headers().add("X-Original-Topic", record.topic().getBytes());
        dlqRecord.headers().add("X-Original-Partition", 
            String.valueOf(record.partition()).getBytes());
        dlqRecord.headers().add("X-Original-Offset", 
            String.valueOf(record.offset()).getBytes());
        dlqRecord.headers().add("X-Original-Timestamp", 
            String.valueOf(record.timestamp()).getBytes());
        dlqRecord.headers().add("X-Error-Class", 
            error.getClass().getName().getBytes());
        dlqRecord.headers().add("X-Error-Message", 
            (error.getMessage() != null ? error.getMessage() : "null").getBytes());
        dlqRecord.headers().add("X-Error-Timestamp", 
            Instant.now().toString().getBytes());
        dlqRecord.headers().add("X-Retry-Count", 
            String.valueOf(maxRetries).getBytes());
        
        dlqProducer.send(dlqRecord, (metadata, exception) -> {
            if (exception != null) {
                log.error("Échec d'envoi vers DLQ pour offset {}", record.offset(), exception);
            } else {
                log.info("Message envoyé vers DLQ: {} -> {} offset {}",
                    record.topic(), dlqTopic, metadata.offset());
            }
        });
    }
}
```

**Bonnes pratiques DLQ :**

*Métadonnées complètes* : Inclure suffisamment d'informations pour diagnostiquer l'erreur et rejouer le message si nécessaire.

*Monitoring de la DLQ* : Configurer des alertes sur le volume de la DLQ. Un pic soudain indique un problème systémique.

*Processus de traitement DLQ* : Définir un processus clair pour examiner et traiter les messages en DLQ — correction manuelle, rejeu automatique après correction du bug, archivage.

*Rétention adaptée* : Configurer une rétention plus longue sur les topics DLQ pour laisser le temps d'investiguer et corriger.

---

## III.4.6 Réglage des Performances

### Métriques de Performance du Consommateur

Avant d'optimiser, mesurer. Les métriques clés permettent d'identifier les goulots d'étranglement et de valider les optimisations. Le consommateur Kafka expose des dizaines de métriques via JMX, mais certaines sont plus critiques que d'autres.

**Consumer lag — La métrique reine.** Le lag est la différence entre le dernier offset produit et l'offset courant du consommateur. Un lag croissant indique que le consommateur ne suit pas le rythme de production.

```
Lag = (Latest Offset) - (Current Consumer Offset)
```

Le lag peut être mesuré de plusieurs façons :

*Métriques JMX du consommateur* : `records-lag` (par partition), `records-lag-avg` (moyenne sur toutes les partitions), `records-lag-max` (maximum). Ces métriques sont disponibles si le consommateur est actif.

*Outils externes* : Burrow (LinkedIn), Kafka Lag Exporter (Lightbend), Conduktor, Confluent Control Center. Ces outils peuvent mesurer le lag même si le consommateur est arrêté.

*Commande CLI* : `kafka-consumer-groups.sh --bootstrap-server localhost:9092 --group my-group --describe`

> **Perspective stratégique**
>
> Le lag est l'indicateur le plus important de la santé d'un pipeline Kafka. Un lag stable (même non nul) est acceptable — le consommateur est simplement en retard d'un montant constant. Un lag croissant est un problème : le consommateur prend du retard progressivement et finira par accumuler des heures voire des jours de retard.
>
> La dérivée du lag (lag growth rate) est souvent plus importante que le lag absolu.

**Throughput.** Nombre de messages ou volume de données consommés par seconde. Les métriques clés sont :
- `records-consumed-rate` : Messages par seconde
- `bytes-consumed-rate` : Octets par seconde
- `fetch-rate` : Requêtes fetch par seconde

**Latence de fetch.** Temps pour récupérer les messages depuis le broker :
- `fetch-latency-avg` : Latence moyenne des requêtes fetch
- `fetch-latency-max` : Latence maximale observée

Une latence fetch élevée peut indiquer des problèmes réseau, un broker surchargé, ou des partitions sur des disques lents.

**Métriques de groupe.** Pour comprendre la santé du groupe de consommateurs :
- `rebalance-total` : Nombre total de rééquilibrages depuis le démarrage
- `rebalance-rate-per-hour` : Fréquence des rééquilibrages
- `last-rebalance-seconds-ago` : Temps depuis le dernier rééquilibrage
- `assigned-partitions` : Nombre de partitions assignées à ce consommateur

**Métriques de commit.** Pour comprendre le comportement des commits :
- `commit-rate` : Nombre de commits par seconde
- `commit-latency-avg` : Latence moyenne des commits
- `committed-time-ns-total` : Temps total passé à commiter

### Configuration JMX et Export

Pour collecter les métriques du consommateur, activer JMX et configurer un exporteur :

```bash
# Activation JMX sur l'application Java
java -Dcom.sun.management.jmxremote \
     -Dcom.sun.management.jmxremote.port=9999 \
     -Dcom.sun.management.jmxremote.authenticate=false \
     -Dcom.sun.management.jmxremote.ssl=false \
     -jar my-consumer-app.jar
```

Pour Prometheus avec JMX Exporter :

```yaml
# jmx_exporter_config.yml
rules:
  # Métriques de lag
  - pattern: kafka.consumer<type=consumer-fetch-manager-metrics, client-id=(.+), topic=(.+), partition=(.+)><>records-lag
    name: kafka_consumer_records_lag
    labels:
      client_id: "$1"
      topic: "$2"
      partition: "$3"
  
  # Métriques de débit
  - pattern: kafka.consumer<type=consumer-fetch-manager-metrics, client-id=(.+)><>records-consumed-rate
    name: kafka_consumer_records_consumed_rate
    labels:
      client_id: "$1"
  
  # Métriques de groupe
  - pattern: kafka.consumer<type=consumer-coordinator-metrics, client-id=(.+)><>rebalance-total
    name: kafka_consumer_rebalance_total
    labels:
      client_id: "$1"
```

### Optimisation du Débit

Quand le lag est croissant et que le consommateur ne suit pas, plusieurs leviers permettent d'augmenter le débit.

**Augmenter `fetch.min.bytes`.** Par défaut (1 byte), le broker répond dès qu'il a des données. Augmenter cette valeur force le broker à attendre d'avoir plus de données, réduisant le nombre de requêtes et améliorant l'efficacité.

```properties
fetch.min.bytes=1048576  # 1 MB
```

*Quand utiliser* : Quand le réseau ou le nombre de requêtes est le goulot d'étranglement. Le consommateur fait trop de petites requêtes.

*Effet secondaire* : Augmente la latence minimale — le broker attend d'avoir suffisamment de données.

**Augmenter `fetch.max.wait.ms`.** Temps maximal d'attente si `fetch.min.bytes` n'est pas atteint. Combiné avec `fetch.min.bytes`, permet d'optimiser le batching des fetch.

```properties
fetch.max.wait.ms=500  # 500ms
```

**Augmenter `max.partition.fetch.bytes`.** Taille maximale de données à récupérer par partition par requête. Une valeur plus grande permet de récupérer plus de messages en une seule requête.

```properties
max.partition.fetch.bytes=1048576  # 1 MB par partition
```

*Attention* : Augmente la mémoire utilisée. Avec N partitions assignées, le consommateur peut utiliser jusqu'à N × `max.partition.fetch.bytes` de mémoire pour les buffers.

**Augmenter `max.poll.records`.** Nombre maximal de messages retournés par `poll()`. Plus de messages par poll réduit l'overhead de l'appel poll() mais augmente le temps entre les polls.

```properties
max.poll.records=1000  # ou plus
```

*Attention* : Si le traitement de `max.poll.records` messages prend plus de `max.poll.interval.ms`, le consommateur sera éjecté du groupe.

**Paralléliser les consommateurs.** Si un seul consommateur ne peut pas suivre malgré les optimisations, ajouter des consommateurs au groupe (dans la limite du nombre de partitions).

**Optimiser le traitement.** Souvent, le goulot d'étranglement n'est pas Kafka mais le traitement applicatif. Profiler le code de traitement pour identifier les inefficacités.

> **Note de terrain**
>
> *Contexte* : Application de traitement d'événements avec lag croissant de 500 messages/seconde.
>
> *Diagnostic* : 
> - Débit consommateur : 1000 msg/s
> - Débit producteur : 1500 msg/s
> - Profiling : 60% du temps dans la sérialisation JSON, 30% dans l'appel base de données, 10% dans Kafka
>
> *Actions* :
> 1. Migration JSON → Avro : Sérialisation 5× plus rapide → débit 2000 msg/s
> 2. Batch des écritures DB : Latence DB réduite → débit 3000 msg/s
> 3. Kafka n'était pas le problème !
>
> *Leçon* : Avant d'optimiser Kafka, vérifier que Kafka est le goulot d'étranglement.

### Optimisation de la Latence

Pour les applications temps réel où chaque milliseconde compte, optimiser la latence de bout en bout.

**Réduire `fetch.min.bytes`.** Une valeur basse (1, le défaut) garantit que le broker répond dès qu'il a des données, minimisant la latence.

```properties
fetch.min.bytes=1
```

**Réduire `fetch.max.wait.ms`.** Limite le temps d'attente du broker même si `fetch.min.bytes` n'est pas atteint.

```properties
fetch.max.wait.ms=100  # ou moins
```

**Réduire `max.poll.records`.** Moins de messages par poll signifie un traitement plus rapide et un retour plus rapide à poll().

```properties
max.poll.records=100
```

**Désactiver le commit automatique.** Le commit automatique introduit un délai avant que les offsets ne soient persistés. Avec un commit manuel immédiat après traitement, la progression est plus prévisible.

**Traitement asynchrone avec commit immédiat.** Pour la latence minimale, commiter immédiatement après réception (avant traitement) et traiter de manière asynchrone. Attention : cela convertit la garantie en at-most-once.

### Compromis Débit vs. Latence — Tableau de Référence

| Paramètre | Pour le débit | Pour la latence | Impact |
|-----------|---------------|-----------------|--------|
| `fetch.min.bytes` | 1 MB | 1 byte | Batching vs. réactivité |
| `fetch.max.wait.ms` | 500ms | 50-100ms | Attente vs. réactivité |
| `max.poll.records` | 1000+ | 50-100 | Volume vs. fréquence |
| `max.partition.fetch.bytes` | 1 MB | 256 KB | Efficacité vs. mémoire |
| Nombre de consommateurs | Moins, plus chargés | Plus, moins chargés | Ressources vs. parallélisme |
| Commit | Périodique | Après chaque batch | Efficacité vs. progression |

### Éviter les Problèmes de Timeout

Les timeouts mal configurés sont une source fréquente de problèmes en production. Un timeout trop court cause des éjections intempestives ; un timeout trop long retarde la détection des vraies pannes.

**`session.timeout.ms`** (défaut 45s) : Si le consommateur ne peut pas envoyer de heartbeat dans ce délai, il est éjecté du groupe. 

*Trop court* : Des GC pauses ou des pics de charge peuvent causer des éjections intempestives, déclenchant des rééquilibrages inutiles.

*Trop long* : Un consommateur vraiment mort n'est pas détecté rapidement, laissant ses partitions sans traitement.

**`heartbeat.interval.ms`** (défaut 3s) : Intervalle entre les heartbeats envoyés au coordinateur. Devrait être environ 1/3 de `session.timeout.ms` pour garantir plusieurs heartbeats par session.

```properties
session.timeout.ms=30000
heartbeat.interval.ms=10000
```

**`max.poll.interval.ms`** (défaut 5 min) : Intervalle maximal entre deux appels `poll()`. Si le traitement des messages prend plus de temps que cette valeur, le consommateur est considéré comme mort et éjecté.

C'est le timeout le plus fréquemment mal configuré. Si le traitement d'un batch de messages prend longtemps (accès base de données, appels API externes), augmenter ce timeout ou réduire `max.poll.records`.

```properties
# Pour un traitement lent (batch ML, agrégations complexes)
max.poll.interval.ms=600000  # 10 minutes
max.poll.records=50  # Moins de messages par poll

# Pour un traitement rapide
max.poll.interval.ms=30000  # 30 secondes
max.poll.records=500
```

> **Anti-patron**
>
> *« Notre consommateur est éjecté régulièrement alors qu'il fonctionne. »*
>
> *Diagnostic* : `max.poll.interval.ms` trop court par rapport au temps de traitement réel.
>
> *Symptôme* : Logs montrant `"Member ... has failed to heartbeat"` ou `"Member ... has left the group"` alors que l'application tourne.
>
> *Solution* : Mesurer le temps réel de traitement d'un batch, configurer `max.poll.interval.ms` à 2-3× ce temps, ou réduire `max.poll.records`.

---

## III.4.7 Construire des Consommateurs Résilients

### Gestion des Erreurs de Désérialisation

Les erreurs de désérialisation sont courantes quand les schémas évoluent ou quand des messages corrompus arrivent dans le topic. Par défaut, une erreur de désérialisation fait échouer le `poll()`, bloquant potentiellement tout le consommateur pour un seul message malformé.

**Désérialiseur avec gestion d'erreur :**

```java
public class ErrorHandlingDeserializer<T> implements Deserializer<T> {
    private final Deserializer<T> delegate;
    private final String errorTopic;
    private final KafkaProducer<byte[], byte[]> errorProducer;
    
    @Override
    public T deserialize(String topic, byte[] data) {
        try {
            return delegate.deserialize(topic, data);
        } catch (Exception e) {
            log.error("Erreur de désérialisation pour topic {}: {}", topic, e.getMessage());
            
            // Optionnel : envoyer le message brut vers un topic d'erreur
            if (errorProducer != null) {
                ProducerRecord<byte[], byte[]> errorRecord = new ProducerRecord<>(
                    errorTopic, data);
                errorRecord.headers().add("X-Original-Topic", topic.getBytes());
                errorRecord.headers().add("X-Error", e.getMessage().getBytes());
                errorProducer.send(errorRecord);
            }
            
            return null; // Retourner null permet de filtrer ensuite
        }
    }
}

// Utilisation avec filtrage des nulls
ConsumerRecords<String, MyObject> records = consumer.poll(Duration.ofMillis(100));
for (ConsumerRecord<String, MyObject> record : records) {
    if (record.value() == null) {
        log.warn("Message ignoré (désérialisation échouée) à offset {}", record.offset());
        continue;
    }
    process(record);
}
```

**Spring Kafka ErrorHandlingDeserializer.** Pour les applications Spring Kafka, un `ErrorHandlingDeserializer` encapsule l'erreur dans un header plutôt que de lever une exception :

```java
props.put(ConsumerConfig.VALUE_DESERIALIZER_CLASS_CONFIG, ErrorHandlingDeserializer.class);
props.put(ErrorHandlingDeserializer.VALUE_DESERIALIZER_CLASS, JsonDeserializer.class);
```

L'erreur est accessible via `record.headers().lastHeader(ErrorHandlingDeserializer.VALUE_DESERIALIZER_EXCEPTION_HEADER)`.

**Schémas et compatibilité.** La meilleure prévention des erreurs de désérialisation est une gestion rigoureuse des schémas avec Schema Registry et des règles de compatibilité. Cependant, même avec une bonne gouvernance, des erreurs peuvent survenir (messages legacy, corruption, bugs producteur).

### Stratégies de Retry Avancées

Quand le traitement échoue, plusieurs stratégies de retry permettent de récupérer des erreurs transitoires sans perdre de messages.

**Retry immédiat avec limite.** Réessayer immédiatement un nombre limité de fois. Simple mais peut surcharger un système déjà en difficulté.

```java
public void processWithImmediateRetry(ConsumerRecord<String, String> record) {
    int maxAttempts = 3;
    for (int attempt = 1; attempt <= maxAttempts; attempt++) {
        try {
            process(record);
            return; // Succès
        } catch (RetriableException e) {
            if (attempt == maxAttempts) {
                sendToDLQ(record, e);
            }
            log.warn("Tentative {}/{} échouée pour offset {}", 
                attempt, maxAttempts, record.offset());
        }
    }
}
```

**Retry avec backoff exponentiel.** Augmenter progressivement le délai entre les tentatives. Laisse le temps au système de récupérer.

```java
public void processWithExponentialBackoff(ConsumerRecord<String, String> record) {
    int maxAttempts = 5;
    long baseDelayMs = 100;
    long maxDelayMs = 10000;
    
    for (int attempt = 1; attempt <= maxAttempts; attempt++) {
        try {
            process(record);
            return;
        } catch (RetriableException e) {
            if (attempt == maxAttempts) {
                sendToDLQ(record, e);
                return;
            }
            
            long delay = Math.min(baseDelayMs * (long) Math.pow(2, attempt - 1), maxDelayMs);
            // Ajouter du jitter pour éviter les thundering herds
            delay += ThreadLocalRandom.current().nextLong(delay / 4);
            
            log.warn("Tentative {}/{} échouée, retry dans {}ms", attempt, maxAttempts, delay);
            try {
                Thread.sleep(delay);
            } catch (InterruptedException ie) {
                Thread.currentThread().interrupt();
                throw new RuntimeException(ie);
            }
        }
    }
}
```

**Retry topic avec délai.** Pour les erreurs nécessitant un délai plus long (système externe indisponible), envoyer vers des topics de retry avec des délais progressifs.

```java
// Architecture : topic → retry-1m → retry-5m → retry-15m → dlq
public void processWithRetryTopics(ConsumerRecord<String, String> record) {
    String retryHeader = getHeader(record, "X-Retry-Count");
    int retryCount = retryHeader != null ? Integer.parseInt(retryHeader) : 0;
    
    try {
        process(record);
    } catch (RetriableException e) {
        String nextTopic;
        if (retryCount == 0) {
            nextTopic = record.topic() + ".retry-1m";
        } else if (retryCount == 1) {
            nextTopic = record.topic() + ".retry-5m";
        } else if (retryCount == 2) {
            nextTopic = record.topic() + ".retry-15m";
        } else {
            nextTopic = record.topic() + ".dlq";
        }
        
        sendToTopic(nextTopic, record, retryCount + 1);
    }
}
```

Les topics de retry peuvent être consommés par des consommateurs dédiés qui attendent le délai approprié avant de renvoyer au topic principal.

### Shutdown Gracieux

Un arrêt gracieux permet de terminer le traitement en cours et de commiter les offsets avant de quitter, évitant le retraitement au redémarrage.

```java
public class GracefulConsumer {
    private final KafkaConsumer<String, String> consumer;
    private final AtomicBoolean running = new AtomicBoolean(true);
    private final CountDownLatch shutdownLatch = new CountDownLatch(1);
    
    public void run() {
        try {
            consumer.subscribe(Arrays.asList("topic"));
            
            while (running.get()) {
                ConsumerRecords<String, String> records = consumer.poll(Duration.ofMillis(100));
                
                for (ConsumerRecord<String, String> record : records) {
                    if (!running.get()) {
                        // Arrêt demandé pendant le traitement
                        // Commiter les offsets déjà traités
                        break;
                    }
                    process(record);
                }
                
                if (!records.isEmpty()) {
                    consumer.commitSync();
                }
            }
        } catch (WakeupException e) {
            // Exception normale si shutdown appelé pendant poll()
            if (running.get()) {
                throw e; // Inattendu si on n'est pas en shutdown
            }
        } finally {
            try {
                // Commit final des offsets
                consumer.commitSync(Duration.ofSeconds(10));
            } catch (Exception e) {
                log.warn("Échec du commit final", e);
            } finally {
                consumer.close(Duration.ofSeconds(10));
                shutdownLatch.countDown();
            }
        }
    }
    
    public void shutdown() {
        log.info("Arrêt gracieux demandé");
        running.set(false);
        consumer.wakeup(); // Interrompt le poll() en cours
        
        try {
            // Attendre la fin du traitement
            boolean completed = shutdownLatch.await(60, TimeUnit.SECONDS);
            if (!completed) {
                log.warn("Timeout lors de l'arrêt gracieux");
            }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }
    }
}

// Intégration avec le hook de shutdown JVM
public static void main(String[] args) {
    GracefulConsumer consumer = new GracefulConsumer(createProps());
    
    Runtime.getRuntime().addShutdownHook(new Thread(() -> {
        log.info("Signal de shutdown reçu");
        consumer.shutdown();
    }));
    
    consumer.run();
}
```

**Points clés du shutdown gracieux :**

*`wakeup()`* : Seule méthode thread-safe du consommateur, permet d'interrompre un `poll()` bloquant depuis un autre thread.

*Timeout de commit final* : Le commit final peut échouer si le coordinateur est indisponible. Prévoir un timeout raisonnable.

*Timeout de close* : La fermeture du consommateur notifie le coordinateur de son départ. Un timeout évite de bloquer indéfiniment.

*Hook JVM* : Le shutdown hook garantit que le signal SIGTERM (Kubernetes, systemd) déclenche l'arrêt gracieux.

### Idempotence du Traitement

Même avec les meilleures pratiques, des messages peuvent être traités plusieurs fois (rééquilibrage au mauvais moment, crash après traitement mais avant commit). Le traitement doit être idempotent ou gérer explicitement les duplicatas.

**Stratégies d'idempotence :**

*Opérations naturellement idempotentes* : UPSERT, PUT (vs. INSERT, POST). Le résultat est le même que l'opération soit exécutée une ou plusieurs fois.

```java
// Idempotent : écraser la valeur existante
database.upsert(key, newValue);

// Non idempotent : insère un nouveau record à chaque fois
database.insert(key, newValue);
```

*Tracking des messages traités* : Stocker les IDs des messages traités et vérifier avant traitement.

```java
public void processIdempotent(ConsumerRecord<String, String> record) {
    String messageId = record.topic() + "-" + record.partition() + "-" + record.offset();
    
    if (processedCache.contains(messageId)) {
        log.debug("Message {} déjà traité, skip", messageId);
        return;
    }
    
    process(record);
    processedCache.add(messageId);
}
```

*Clé de déduplication dans le message* : Le producteur inclut un ID unique que le consommateur utilise pour dédupliquer.

### Monitoring et Alerting

**Métriques essentielles à surveiller :**

| Métrique | Seuil d'alerte | Signification |
|----------|----------------|---------------|
| Consumer lag | > 10000 pendant 5 min | Le consommateur ne suit pas |
| Lag growth rate | Positif pendant 10 min | Problème de capacité persistant |
| Rebalance rate | > 1/heure | Instabilité du groupe |
| Poll rate | < attendu | Traitement trop lent ou consommateur bloqué |
| Commit latency | > 1s | Problème avec le coordinateur |
| Error rate | > 0.1% | Erreurs de traitement à investiguer |
| DLQ volume | > 0 | Messages non traités |

**Alertes recommandées en production :**

```yaml
# Prometheus alerting rules
groups:
  - name: kafka-consumer-alerts
    rules:
      - alert: ConsumerLagHigh
        expr: kafka_consumer_records_lag > 10000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Consumer lag élevé pour {{ $labels.group }}"
          
      - alert: ConsumerLagCritical
        expr: kafka_consumer_records_lag > 100000
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Consumer lag critique pour {{ $labels.group }}"
          
      - alert: FrequentRebalances
        expr: rate(kafka_consumer_rebalance_total[1h]) > 2
        labels:
          severity: warning
        annotations:
          summary: "Rééquilibrages fréquents pour {{ $labels.group }}"
          
      - alert: DLQNonEmpty
        expr: kafka_consumer_records_lag{topic=~".*\\.dlq"} > 0
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Messages en DLQ pour {{ $labels.topic }}"
```

**Dashboards recommandés :**

1. *Vue d'ensemble du groupe* : Nombre de membres, partitions assignées, lag total par groupe.
2. *Lag par partition* : Heatmap ou graphique permettant d'identifier les partitions problématiques.
3. *Throughput* : Messages/seconde et bytes/seconde par consommateur et au total.
4. *Santé* : Taux d'erreur, fréquence des rééquilibrages, latence des commits.
5. *Dead Letter Queue* : Volume des DLQ par topic, évolution dans le temps.

### Checklist de Mise en Production

**Configuration :**
- [ ] `group.id` unique et descriptif
- [ ] `client.id` configuré pour identification dans les métriques
- [ ] `enable.auto.commit=false` pour le commit manuel
- [ ] `auto.offset.reset` approprié (earliest ou latest selon le cas)
- [ ] Timeouts ajustés selon le temps de traitement réel
- [ ] `partition.assignment.strategy` = CooperativeStickyAssignor
- [ ] `group.instance.id` pour les déploiements rolling (optionnel)

**Résilience :**
- [ ] Gestion des erreurs de désérialisation
- [ ] Stratégie de retry avec backoff
- [ ] Dead Letter Queue configurée
- [ ] Shutdown gracieux implémenté avec hook JVM
- [ ] Traitement idempotent ou gestion des duplicatas

**Monitoring :**
- [ ] Métriques JMX exposées et collectées
- [ ] Alertes sur le lag configurées
- [ ] Alertes sur les rééquilibrages fréquents
- [ ] Dashboard de monitoring disponible
- [ ] Monitoring de la DLQ

**Tests :**
- [ ] Tests unitaires avec MockConsumer
- [ ] Tests d'intégration avec Testcontainers
- [ ] Tests de charge validés
- [ ] Tests de résilience (kill de consommateurs, rééquilibrages forcés)
- [ ] Tests de replay (seek + retraitement)

---

## III.4.8 Résumé

Ce chapitre a exploré en profondeur le consommateur Kafka, le composant responsable de la lecture et du traitement des messages depuis le cluster. Une maîtrise complète du consommateur est essentielle pour construire des applications réactives, scalables et fiables dans une architecture événementielle.

### Architecture et Principes Fondamentaux

Le consommateur Kafka utilise un modèle **pull** où il demande activement les messages aux brokers via des requêtes fetch. Cette architecture fondamentale offre plusieurs avantages distinctifs par rapport au modèle push des systèmes de messagerie traditionnels. Le consommateur contrôle naturellement son rythme de consommation, créant un backpressure naturel qui empêche la surcharge. Il peut repositionner son offset pour relire des messages, permettant le replay et la reconstruction de systèmes. Le broker n'a pas besoin de maintenir l'état de chaque consommateur, simplifiant son implémentation et améliorant sa scalabilité.

La **boucle de consommation** (poll loop) est le pattern central de toute application consommatrice. L'application appelle répétitivement `poll()` pour récupérer des lots de messages. Chaque appel à `poll()` effectue plusieurs opérations critiques simultanément : envoi de heartbeats au coordinateur pour maintenir l'appartenance au groupe, rafraîchissement des métadonnées du cluster si nécessaire, fetch de nouveaux messages depuis les brokers, et retour des messages disponibles à l'application. Le consommateur n'est pas thread-safe — tous les appels doivent provenir du même thread, ce qui simplifie l'implémentation et évite les problèmes de synchronisation coûteux.

La **gestion des offsets** est le mécanisme central pour le suivi de la progression et les garanties de livraison. L'offset courant avance automatiquement à chaque lecture, mais l'offset commité (persisté dans le topic `__consumer_offsets`) détermine le point de reprise en cas de redémarrage. Cette distinction est cruciale : le commit automatique est pratique mais dangereux pour les traitements critiques car il peut causer des pertes de messages ou des duplicatas. Le commit manuel après traitement réussi est fortement recommandé pour les applications de production où la fiabilité est importante.

### Groupes de Consommateurs et Parallélisme

Les **groupes de consommateurs** sont le mécanisme fondamental pour le parallélisme en Kafka. Un groupe est identifié par un `group.id` unique, et tous les consommateurs partageant ce `group.id` forment un groupe qui se partage les partitions des topics souscrits. La règle fondamentale est que chaque partition est assignée à exactement un consommateur du groupe à tout instant, garantissant que les messages d'une partition sont traités dans l'ordre par un seul consommateur.

Cette architecture implique que le **parallélisme maximal** est égal au nombre de partitions du topic. Avoir plus de consommateurs que de partitions signifie que certains consommateurs resteront inactifs, attendant qu'une partition se libère. Cette limite doit être considérée dès la conception du topic — le nombre de partitions détermine le parallélisme maximal possible pour tout le cycle de vie du topic.

Les groupes de consommateurs sont **complètement indépendants** les uns des autres. Chaque groupe maintient ses propres offsets et sa propre progression. Cette indépendance permet plusieurs patterns architecturaux puissants : diffusion d'événements vers plusieurs systèmes (chaque système a son propre groupe), scaling horizontal d'une application (toutes les instances partagent le même groupe), environnements de test isolés (le groupe de test n'interfère pas avec la production), et replay pour reconstruction (un nouveau groupe peut consommer depuis le début).

Le **dimensionnement du groupe** doit équilibrer plusieurs facteurs : le débit requis (plus de consommateurs pour plus de débit), le nombre de partitions (limite supérieure du parallélisme), les ressources disponibles (chaque consommateur consomme mémoire, CPU, connexions), et la résilience souhaitée (consommateurs en standby pour reprise rapide).

### Rééquilibrage des Consommateurs

Le **rééquilibrage** est le processus par lequel les partitions sont redistribuées entre les membres d'un groupe lors de changements de membership. Un rééquilibrage se produit quand un nouveau consommateur rejoint le groupe, quand un consommateur quitte le groupe (gracieusement ou par crash), quand le nombre de partitions d'un topic souscrit change, ou quand un consommateur change sa souscription.

Le protocole de rééquilibrage **classique (Eager)** interrompt tous les consommateurs du groupe pendant le rééquilibrage. Même les consommateurs dont les partitions ne changent pas doivent arrêter de consommer, créant une interruption « stop-the-world » qui peut durer plusieurs secondes pour les grands groupes. Ce comportement est particulièrement problématique pour les applications nécessitant une haute disponibilité.

Le protocole de rééquilibrage **coopératif** (disponible depuis Kafka 2.4) améliore significativement cette situation en permettant aux consommateurs de continuer à traiter leurs partitions non affectées pendant le rééquilibrage. Seules les partitions qui changent de propriétaire sont révoquées, et le rééquilibrage se déroule en deux phases pour minimiser les interruptions. L'utilisation du `CooperativeStickyAssignor` est fortement recommandée pour tous les nouveaux déploiements.

L'**assignation statique** (`group.instance.id`) permet à un consommateur redémarré de récupérer automatiquement ses partitions précédentes sans déclencher de rééquilibrage complet, pourvu qu'il redémarre avant l'expiration du `session.timeout.ms`. Cette fonctionnalité est particulièrement utile pour les déploiements rolling où les consommateurs redémarrent fréquemment.

### Modèles de Conception

Plusieurs **patterns** structurent les applications consommatrices selon leurs besoins spécifiques :

Le pattern **un thread par consommateur** est le plus simple et le plus courant. Chaque consommateur s'exécute dans son propre thread, préservant naturellement l'ordre de traitement et respectant la contrainte single-threaded du consommateur. Ce pattern est recommandé pour la majorité des cas d'usage.

Le pattern **découplage fetch/traitement** sépare le thread qui appelle `poll()` des threads qui traitent les messages. Cette approche permet le traitement parallèle mais complexifie significativement la gestion des offsets — quand peut-on commiter si le traitement est asynchrone ? Ce pattern nécessite une attention particulière pour éviter les pertes de messages.

Le pattern **pause/resume** utilise les méthodes `pause()` et `resume()` du consommateur pour implémenter le backpressure vers les systèmes downstream. Le consommateur met en pause les partitions quand le système downstream est surchargé et reprend quand la situation s'améliore.

Le pattern **assignation manuelle** utilise `assign()` au lieu de `subscribe()` pour un contrôle total sur les partitions consommées. Ce pattern est utile pour le replay ciblé, la migration de données, ou les cas où la coordination de groupe n'est pas souhaitée.

Le pattern **at-least-once avec idempotence** combine le commit manuel après traitement avec un traitement idempotent pour garantir qu'aucun message n'est perdu tout en gérant correctement les duplicatas inévitables.

### Stratégies Avancées

L'**exactly-once sémantique** de bout en bout est atteinte en combinant les transactions Kafka avec `isolation.level=read_committed` côté consommateur. Les offsets consommés et les messages produits sont commités dans la même transaction atomique, garantissant que le traitement d'un message et la production de ses résultats réussissent ou échouent ensemble.

La **consommation multi-topic** permet à un groupe de traiter plusieurs types d'événements depuis différents topics. Les abonnements peuvent être explicites (liste de topics) ou dynamiques (pattern regex). Le routing du traitement doit gérer les différents types de messages reçus.

Le **seek** permet le repositionnement des offsets pour diverses raisons : replay après correction de bug, reconstruction de systèmes downstream, saut de messages corrompus, ou reprise depuis un checkpoint externe. Le repositionnement par timestamp (`offsetsForTimes()`) est particulièrement utile pour les reprises basées sur une date.

Les **Dead Letter Queues** isolent les messages non traitables après épuisement des retry, permettant au flux principal de continuer tout en préservant les messages problématiques pour analyse et correction ultérieure. Une DLQ bien conçue inclut suffisamment de métadonnées pour diagnostiquer l'erreur et rejouer le message.

### Optimisation des Performances

Le **consumer lag** est la métrique reine pour évaluer la santé d'un consommateur. Le lag représente la différence entre le dernier offset produit et l'offset courant du consommateur. Un lag stable (même non nul) est acceptable — le consommateur est simplement en retard d'un montant constant. Un lag croissant est un problème qui nécessite une intervention : soit augmenter le nombre de consommateurs, soit optimiser le traitement.

L'optimisation du **débit** passe par l'augmentation du batching : `fetch.min.bytes` plus élevé force le broker à attendre plus de données avant de répondre, `max.poll.records` plus élevé permet de traiter plus de messages par appel poll(), et l'ajout de consommateurs parallèles augmente la capacité de traitement globale.

L'optimisation de la **latence** privilégie des fetch rapides et un traitement fréquent : `fetch.min.bytes` bas garantit une réponse rapide du broker, `fetch.max.wait.ms` court limite le temps d'attente, et `max.poll.records` modéré permet un retour rapide à la boucle de consommation.

Les **timeouts** doivent être soigneusement configurés pour éviter les faux positifs (éjection d'un consommateur fonctionnel) tout en détectant rapidement les vraies pannes. `session.timeout.ms` contrôle la détection des pannes, `heartbeat.interval.ms` doit être environ 1/3 du session timeout, et `max.poll.interval.ms` doit accommoder le temps de traitement réel entre les polls.

### Résilience et Opérations

Les consommateurs résilients implémentent plusieurs mécanismes de protection :

La **gestion des erreurs de désérialisation** évite qu'un seul message malformé ne bloque tout le consommateur. Un désérialiseur avec gestion d'erreur peut logger l'erreur, envoyer le message brut vers un topic d'erreur, et retourner null pour permettre au traitement de continuer.

Les **stratégies de retry** avec backoff exponentiel permettent de récupérer des erreurs transitoires sans surcharger les systèmes en difficulté. Le jitter (variation aléatoire) évite les « thundering herds » où tous les retry arrivent simultanément.

Le **shutdown gracieux** termine le traitement en cours et commite les offsets avant de quitter, évitant le retraitement au redémarrage. L'intégration avec le hook de shutdown JVM garantit que les signaux système (SIGTERM) déclenchent l'arrêt gracieux.

L'**idempotence du traitement** garantit que le même message peut être traité plusieurs fois avec le même résultat, gérant correctement les duplicatas inévitables lors des rééquilibrages ou des reprises.

Le **monitoring** du lag, des rééquilibrages, et du taux d'erreur permet de détecter les problèmes avant qu'ils n'impactent les utilisateurs. Les alertes doivent être actionnables — un lag croissant nécessite une action, pas seulement une notification.

---

### Vers le Chapitre Suivant

Ce chapitre a couvert la consommation de messages — comment les applications lisent et traitent depuis Kafka. Le chapitre suivant, « Cas d'Utilisation Kafka », explorera quand utiliser Kafka, comment naviguer les implémentations en contexte réel, et les alternatives à considérer selon les cas d'usage.

La maîtrise de la production (chapitre III.3) et de la consommation (ce chapitre) permet à l'architecte de concevoir des pipelines événementiels complets, de bout en bout, avec les garanties appropriées à chaque cas d'usage métier.

---

*Volume III : Apache Kafka - Guide de l'Architecte*

*Chapitre III.4 — Création d'Applications Consommatrices*

*Monographie « L'Entreprise Agentique »*
