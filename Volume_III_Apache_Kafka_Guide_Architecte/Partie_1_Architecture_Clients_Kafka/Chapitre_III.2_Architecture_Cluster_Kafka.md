# Chapitre III.2

## ARCHITECTURE D'UN CLUSTER KAFKA

---

> *« Pour maîtriser un système, il faut comprendre non seulement ce qu'il fait, mais comment il le fait — car c'est dans le comment que résident les limites et les possibilités. »*
>
> — Werner Vogels

---

Le chapitre précédent a présenté Apache Kafka du point de vue stratégique de l'architecte d'entreprise. Ce chapitre plonge dans les mécanismes internes qui font de Kafka une plateforme de streaming exceptionnelle. Comprendre l'architecture d'un cluster Kafka n'est pas un exercice académique — c'est une nécessité pratique pour dimensionner correctement l'infrastructure, diagnostiquer les problèmes de performance, et concevoir des applications résilientes.

L'architecte qui maîtrise les concepts développés dans ce chapitre sera capable d'expliquer pourquoi certaines configurations fonctionnent mieux que d'autres, de prédire le comportement du système sous différentes charges, et de prendre des décisions éclairées sur le partitionnement, la réplication, et la rétention des données. Cette compréhension profonde distingue l'architecte compétent de celui qui se contente d'appliquer des recettes sans les comprendre.

Nous explorerons successivement l'anatomie d'un message Kafka, l'organisation logique en topics et partitions, la représentation physique sur disque, le modèle de réplication qui assure la durabilité, et la gestion du cycle de vie des données. Chaque section combine la théorie nécessaire à la compréhension avec les implications pratiques pour l'architecte.

---

## III.2.1 L'Unité Fondamentale : Anatomie d'un Message Kafka

### Le Record Kafka : Structure et Composants

L'unité fondamentale de données dans Kafka est le **record** (ou message). Chaque record publié dans un topic Kafka possède une structure précise que l'architecte doit comprendre pour optimiser l'utilisation de la plateforme. Cette structure, bien que simple en apparence, a des implications profondes sur la performance, le partitionnement, et l'évolution des schémas.

Un record Kafka se compose de plusieurs éléments constitutifs qui forment ensemble l'unité atomique de données transitant par la plateforme.

**La clé (Key).** La clé est un tableau d'octets optionnel qui détermine la partition de destination du message. Deux messages avec la même clé seront systématiquement routés vers la même partition, garantissant leur ordre relatif. La clé peut être nulle — dans ce cas, le message est distribué selon un algorithme round-robin entre les partitions disponibles.

Le choix de la clé est une décision architecturale critique qui mérite une attention particulière. Une clé bien choisie préserve les invariants métier (tous les événements d'un même client dans la même partition) tout en assurant une distribution équilibrée de la charge. Une clé mal choisie peut créer des « partitions chaudes » surchargées pendant que d'autres restent sous-utilisées, ou pire, perdre les garanties d'ordre nécessaires au traitement correct des événements.

**La valeur (Value).** La valeur est le contenu principal du message — les données métier que l'on souhaite transmettre. C'est également un tableau d'octets, ce qui signifie que Kafka est agnostique au format des données. JSON, Avro, Protobuf, XML, ou même des formats binaires propriétaires peuvent être utilisés. Cette flexibilité est à la fois une force (liberté de choix) et un défi (nécessité de gouvernance des formats).

La taille de la valeur impacte directement les performances. Kafka est optimisé pour des messages de quelques kilo-octets. Les messages volumineux (plusieurs mégaoctets) sont techniquement possibles mais dégradent les performances et compliquent la gestion de la mémoire. Pour les données volumineuses, le pattern recommandé est de stocker les données dans un système externe (S3, GCS) et de publier uniquement une référence dans Kafka.

**Les en-têtes (Headers).** Introduits dans Kafka 0.11, les en-têtes permettent d'ajouter des métadonnées au message sans modifier la valeur. Chaque en-tête est une paire clé-valeur où la clé est une chaîne et la valeur est un tableau d'octets. Les en-têtes sont utiles pour le traçage distribué (correlation IDs, trace IDs), les métadonnées de routage, et les informations de provenance.

Les en-têtes ne participent pas au calcul de la partition — seule la clé du message est utilisée à cette fin. Cette distinction est importante : les en-têtes sont des métadonnées « passives » qui accompagnent le message sans influencer son routage.

**Le timestamp.** Chaque record possède un timestamp qui peut être de deux types selon la configuration du topic. Le type `CreateTime` (par défaut) utilise le timestamp fourni par le producteur au moment de la création du message. Le type `LogAppendTime` utilise le timestamp du broker au moment de l'écriture dans le log. Le choix du type de timestamp a des implications sur le fenêtrage temporel dans le stream processing et sur les requêtes basées sur le temps.

> **Définition formelle**
>
> Un **record Kafka** est un n-uplet composé de : `(key: bytes | null, value: bytes, headers: [(string, bytes)], timestamp: long, offset: long)`. L'offset est attribué par le broker lors de l'écriture et n'est pas fourni par le producteur. Le timestamp peut être fourni par le producteur ou attribué par le broker selon la configuration.

### Format de Sérialisation sur le Fil

Lors de la transmission sur le réseau, les records sont regroupés en **batches** pour optimiser les performances. Un batch contient plusieurs records destinés à la même partition, ce qui permet d'amortir les coûts de réseau et d'I/O disque sur plusieurs messages.

Le format de batch (introduit dans Kafka 0.11 sous le nom « message format v2 ») utilise une structure optimisée comprenant plusieurs éléments. L'en-tête du batch contient les métadonnées communes : premier offset, dernier offset delta, timestamp du premier message, timestamp max, attributs (compression, type de timestamp), et CRC de validation. Les records individuels sont stockés avec des deltas relatifs au premier message du batch, économisant de l'espace.

Cette structure en batch a des implications pratiques pour l'architecte. La compression s'applique au niveau du batch, pas du message individuel, ce qui améliore le ratio de compression. La configuration `linger.ms` contrôle le temps d'attente avant d'envoyer un batch incomplet — un compromis entre latence (valeur basse) et throughput (valeur haute). La configuration `batch.size` définit la taille maximale d'un batch en octets.

### Compression des Messages

Kafka supporte plusieurs algorithmes de compression qui s'appliquent au niveau du batch. Le choix de l'algorithme de compression est une décision architecturale qui impacte l'utilisation du CPU, du réseau, et du stockage.

**Aucune compression (none).** Les messages sont transmis et stockés sans modification. Cette option minimise l'utilisation CPU mais maximise l'utilisation réseau et stockage. Elle convient aux messages déjà compressés (images, vidéos) ou aux environnements où le CPU est le goulot d'étranglement.

**GZIP.** Offre un excellent ratio de compression mais consomme significativement plus de CPU que les alternatives. GZIP convient aux scénarios où la bande passante réseau est le facteur limitant et où le CPU est abondant. Le ratio de compression typique est de 70-80 % pour des données textuelles.

**Snappy.** Développé par Google, Snappy privilégie la vitesse sur le ratio de compression. Il est environ 10 fois plus rapide que GZIP mais avec un ratio de compression inférieur (40-50 %). Snappy est un bon choix par défaut pour la plupart des cas d'usage.

**LZ4.** Similaire à Snappy en termes de compromis vitesse/compression, LZ4 offre généralement de meilleures performances avec un ratio de compression comparable. C'est souvent le meilleur choix pour les déploiements à haute performance.

**Zstandard (zstd).** Introduit dans Kafka 2.1, Zstandard offre un excellent compromis entre ratio de compression et vitesse. Il surpasse généralement GZIP en ratio de compression tout en étant significativement plus rapide. Zstandard supporte également des niveaux de compression configurables, permettant d'ajuster le compromis vitesse/ratio.

> **Décision architecturale**
>
> *Contexte* : Choix de l'algorithme de compression pour un backbone événementiel traitant 100 000 événements JSON par seconde.
>
> *Analyse* : Les événements JSON sont hautement compressibles. Le réseau inter-datacenter est coûteux. Les brokers disposent de CPU moderne en quantité suffisante.
>
> *Options évaluées* : GZIP (ratio excellent, CPU élevé), LZ4 (ratio bon, CPU faible), Zstd (ratio très bon, CPU modéré).
>
> *Décision* : Zstandard niveau 3 — offre 65 % de compression avec un overhead CPU acceptable, réduisant significativement les coûts réseau et stockage.
>
> *Métriques de suivi* : Ratio de compression effectif, utilisation CPU des brokers, latence de production.

### Gestion des Erreurs de Sérialisation

La sérialisation et la désérialisation des messages sont des opérations critiques qui peuvent échouer. L'architecte doit anticiper ces échecs et concevoir des stratégies de gestion appropriées.

**Erreurs côté producteur.** Si la sérialisation échoue (objet incompatible avec le schéma, dépassement de taille), le message ne sera pas envoyé. Le producteur doit implémenter une gestion d'erreur explicite : journalisation de l'erreur, notification à un système de monitoring, éventuellement routage vers une file de messages en erreur.

**Erreurs côté consommateur.** Si la désérialisation échoue (schéma incompatible, données corrompues), le consommateur fait face à un dilemme : ignorer le message et continuer, bloquer jusqu'à résolution, ou router le message vers une Dead Letter Queue (DLQ) pour traitement ultérieur. La stratégie DLQ est généralement recommandée car elle préserve les messages problématiques pour analyse tout en permettant au consommateur de progresser.

**Versionnement des schémas.** L'évolution des schémas au fil du temps est inévitable. Sans gouvernance, les incompatibilités de schémas deviennent une source majeure d'incidents. L'utilisation d'un Schema Registry avec des règles de compatibilité strictes (BACKWARD pour permettre les nouveaux consommateurs de lire les anciens messages, FORWARD pour permettre les anciens consommateurs de lire les nouveaux messages, FULL pour les deux) est fortement recommandée.

> **Note de terrain**
>
> *Contexte* : Un système de facturation consomme des événements de commande. L'équipe produit ajoute un nouveau champ obligatoire au schéma sans coordination.
>
> *Impact* : Les nouveaux messages ne peuvent plus être désérialisés par le consommateur existant. Le consumer lag explose. Les factures ne sont plus générées.
>
> *Résolution immédiate* : Déploiement d'urgence du consommateur avec le nouveau schéma.
>
> *Résolution structurelle* : Mise en place de Schema Registry avec validation de compatibilité BACKWARD. Tout nouveau schéma incompatible est rejeté au moment de la publication, forçant la coordination entre équipes.
>
> *Leçon* : La gouvernance des schémas n'est pas optionnelle pour les systèmes de production. Le coût de sa mise en place est négligeable comparé au coût des incidents d'incompatibilité.

### Implications pour la Conception des Messages

La structure des messages Kafka a des implications directes sur la conception des applications et des schémas de données.

**Taille des messages.** La limite par défaut (`message.max.bytes`) est de 1 Mo par message. Cette limite peut être augmentée mais avec des conséquences sur la gestion mémoire. Les messages volumineux augmentent la pression sur le heap des brokers et des clients, risquent de déclencher des timeouts si le traitement est lent, et compliquent la gestion des erreurs (rejeu d'un message de 10 Mo vs. 10 Ko). La recommandation est de maintenir les messages sous 100 Ko pour la majorité des cas d'usage, et d'utiliser des références externes pour les données volumineuses.

**Conception des clés.** La clé détermine le partitionnement et donc les garanties d'ordre. Une clé efficace possède plusieurs caractéristiques : elle préserve les invariants métier (ordre des événements d'une même entité), elle distribue uniformément la charge (éviter les « hot keys »), elle est stable dans le temps (une clé qui change fréquemment perd son utilité), et elle est compacte (les clés volumineuses gaspillent de l'espace, surtout avec la compaction).

**Évolution des schémas.** Kafka étant agnostique au format, la compatibilité des schémas est une responsabilité applicative. L'utilisation d'un Schema Registry (Confluent, Apicurio) avec des formats évolutifs (Avro, Protobuf) est fortement recommandée pour les déploiements entreprise. Les règles de compatibilité (BACKWARD, FORWARD, FULL) doivent être définies et appliquées dès le départ.

> **Exemple concret**
>
> *Scénario* : Un système de commerce électronique publie des événements de commande.
>
> *Mauvaise conception* : Clé = `null` (round-robin), Valeur = JSON non versionné incluant les détails produits complets (images en base64).
>
> *Problèmes* : Les événements d'une même commande peuvent arriver dans le désordre (pas de clé). Les messages sont volumineux (images incluses). L'évolution du schéma cassera les consommateurs.
>
> *Bonne conception* : Clé = `order_id`, Valeur = Avro avec référence aux produits (IDs, pas les détails), Version du schéma dans le header.
>
> *Avantages* : Ordre garanti par commande. Messages compacts. Évolution contrôlée via Schema Registry.

---

## III.2.2 Organisation Logique : Topics, Partitions et Stratégies

### Le Topic : Unité Logique de Publication

Un **topic** Kafka est un flux logique de messages regroupés par catégorie ou domaine métier. C'est l'abstraction principale avec laquelle les applications interagissent — les producteurs publient vers des topics, les consommateurs s'abonnent à des topics.

Conceptuellement, un topic peut être vu comme une catégorie de messages ou un canal de communication. En pratique, un topic est une collection de partitions distribuées sur les brokers du cluster. Cette distinction entre l'abstraction logique (topic) et l'implémentation physique (partitions) est fondamentale pour comprendre le comportement de Kafka.

Les topics sont identifiés par un nom unique au sein du cluster. Les conventions de nommage varient selon les organisations, mais une structure hiérarchique est généralement recommandée. Le format `<domaine>.<entité>.<événement>` donne par exemple `sales.orders.created` ou `inventory.stock.updated`. Cette convention facilite la gouvernance, le filtrage, et la compréhension du paysage événementiel.

> **Définition formelle**
>
> Un **topic** Kafka est une abstraction logique représentant un flux de messages partageant une sémantique commune. Physiquement, un topic est matérialisé par une ou plusieurs partitions distribuées sur les brokers du cluster. Les propriétés du topic (nombre de partitions, facteur de réplication, politique de rétention) sont configurables indépendamment pour chaque topic.

### La Partition : Unité de Parallélisme et d'Ordre

Chaque topic est divisé en une ou plusieurs **partitions**. La partition est l'unité fondamentale de parallélisme dans Kafka — c'est le niveau auquel l'ordre des messages est garanti et le niveau auquel les consommateurs parallélisent leur traitement.

Une partition est un journal ordonné et immuable de messages. Chaque message dans une partition reçoit un offset séquentiel unique. Les messages sont ajoutés à la fin de la partition (append-only) et ne peuvent être ni modifiés ni supprimés individuellement (seule la rétention globale supprime les anciens messages).

L'ordre des messages est garanti au sein d'une partition mais pas entre partitions. Si un producteur envoie les messages A, B, C vers la même partition, un consommateur les recevra dans cet ordre. Mais si A va vers la partition 0 et B vers la partition 1, l'ordre relatif de A et B n'est pas garanti.

Cette propriété est cruciale pour la conception des applications. Les événements qui doivent être traités dans l'ordre doivent partager la même clé de partitionnement. Les événements sans dépendance d'ordre peuvent être distribués sur plusieurs partitions pour maximiser le parallélisme.

### Stratégies de Partitionnement

Le **partitionnement** détermine quelle partition recevra chaque message. Kafka offre plusieurs stratégies de partitionnement, et le choix de la stratégie a des implications majeures sur les performances et les garanties de l'application.

**Partitionnement par clé (défaut avec clé).** Quand un message possède une clé non nulle, Kafka calcule un hash de la clé et utilise le modulo du nombre de partitions pour déterminer la partition cible. La formule est : `partition = hash(key) % num_partitions`. Cette stratégie garantit que tous les messages avec la même clé arrivent dans la même partition.

L'algorithme de hash par défaut est « murmur2 », choisi pour sa bonne distribution et sa performance. L'architecte doit comprendre que si le nombre de partitions change, le mapping clé-partition change également, ce qui peut perturber l'ordre des messages en cours de traitement.

**Partitionnement round-robin (défaut sans clé).** Quand un message n'a pas de clé (clé nulle), Kafka distribue les messages en round-robin entre les partitions disponibles. Cette stratégie maximise la distribution de charge mais ne fournit aucune garantie d'ordre entre les messages.

Depuis Kafka 2.4, le comportement par défaut pour les messages sans clé a évolué vers un « sticky partitioning » qui envoie plusieurs messages consécutifs vers la même partition avant de changer, améliorant le batching et donc les performances.

**Partitionnement personnalisé.** Les applications peuvent implémenter leur propre logique de partitionnement via l'interface `Partitioner`. Cette approche est utile pour des cas spéciaux comme le routage géographique, l'équilibrage basé sur la charge des partitions, ou des règles métier complexes.

**Partitionnement composite.** Dans certains cas, une clé composite permet de satisfaire des exigences contradictoires. Par exemple, pour un système de commandes où l'on souhaite traiter les commandes d'un client dans l'ordre mais aussi distribuer la charge, une clé `customer_id` préserve l'ordre par client, tandis qu'une clé `customer_id + order_id % N` distribue les commandes d'un même client sur N partitions (perdant l'ordre strict mais gagnant en parallélisme).

**Impact du changement de partitions.** Augmenter le nombre de partitions est une opération courante lors de la croissance. Cependant, l'architecte doit comprendre que cette opération modifie le mapping clé-partition. Les messages avec une clé donnée iront vers une partition différente après l'ajout. Si des traitements sont en cours, cela peut créer des désordres temporaires. Pour les topics où l'ordre strict est critique, planifier les changements de partitions pendant des périodes de faible activité et s'assurer que les consommateurs ont traité tous les messages existants.

**Analyse de la distribution des clés.** Avant de mettre en production, analyser la distribution attendue des clés. Un histogramme des clés par volume permet de détecter les déséquilibres potentiels. Les outils de monitoring Kafka (Confluent Control Center, Conduktor) peuvent visualiser la charge par partition pour détecter les « hot partitions » en production.

> **Note de terrain**
>
> *Contexte* : Système de trading avec un topic `trades` partitionné par symbole d'instrument (`AAPL`, `GOOG`, etc.).
>
> *Problème observé* : La partition contenant `AAPL` (action très tradée) reçoit 40 % du trafic total, créant un déséquilibre majeur. Le consumer de cette partition ne suit pas, créant un lag croissant.
>
> *Analyse* : Le partitionnement par clé unique (symbole) crée des « hot partitions » quand la distribution des clés est non uniforme.
>
> *Solutions envisagées* : (1) Augmenter les partitions — ne résout pas le problème car les trades AAPL restent ensemble. (2) Partitionner par `symbole + trade_id % N` — distribue les trades d'un même symbole mais perd l'ordre. (3) Accepter le déséquilibre et dimensionner pour le pire cas.
>
> *Décision* : Option 3 retenue car l'ordre des trades par symbole est un invariant métier non négociable. Dimensionnement des consumers pour gérer le pic de la partition la plus chargée.
>
> *Leçon* : Le partitionnement est un compromis entre ordre et distribution. Les invariants métier doivent guider le choix.

### Dimensionnement du Nombre de Partitions

Le nombre de partitions d'un topic est une décision architecturale importante qui impacte le parallélisme, la performance, et la complexité opérationnelle.

**Facteurs en faveur d'un grand nombre de partitions :**

Le parallélisme des consommateurs est limité par le nombre de partitions — un consumer group ne peut avoir plus de consommateurs actifs que de partitions. Augmenter les partitions permet plus de consommateurs parallèles. Le débit d'écriture maximal d'une partition est limité (typiquement 10-50 Mo/s selon le matériel). Plus de partitions permettent un débit total plus élevé.

**Facteurs en faveur d'un petit nombre de partitions :**

Chaque partition consomme des ressources sur les brokers : descripteurs de fichiers, mémoire pour les index, threads de réplication. Les clusters avec des millions de partitions deviennent difficiles à gérer. Les élections de leader après un crash de broker prennent un temps proportionnel au nombre de partitions affectées. La latence de bout en bout peut augmenter avec plus de partitions (plus de coordination nécessaire).

**Recommandations pratiques :**

Pour un nouveau topic, commencer avec un nombre modéré de partitions basé sur le débit attendu. Une règle empirique est de viser 10-20 Mo/s par partition et de prévoir le double du parallélisme de consommation nécessaire. Le nombre de partitions peut être augmenté ultérieurement mais jamais réduit (sans recréer le topic).

| Débit cible | Partitions recommandées | Justification |
|-------------|------------------------|---------------|
| < 10 Mo/s | 3-6 | Minimum pour la résilience avec RF=3 |
| 10-100 Mo/s | 6-12 | Bon équilibre performance/complexité |
| 100 Mo/s - 1 Go/s | 12-50 | Parallélisme élevé nécessaire |
| > 1 Go/s | 50-100+ | Cas extrêmes, expertise requise |

> **Anti-patron**
>
> *« Créons 1000 partitions par défaut pour être tranquilles. »* Ce sur-provisionnement crée une charge opérationnelle inutile : temps de récupération après panne allongé, consommation mémoire excessive, complexité de monitoring accrue. Le coût marginal de chaque partition est faible mais s'accumule à l'échelle du cluster.
>
> Mieux vaut commencer conservateur et augmenter si nécessaire. L'augmentation du nombre de partitions est une opération en ligne dans Kafka moderne.

### Assignation des Partitions aux Brokers

Les partitions d'un topic sont distribuées sur les brokers du cluster selon des règles configurables. Cette distribution détermine comment la charge est répartie et comment le système se comporte en cas de panne.

Par défaut, Kafka distribue les partitions de manière à équilibrer la charge entre les brokers. Le **leader** de chaque partition est le broker responsable de toutes les lectures et écritures pour cette partition. Les **followers** répliquent les données du leader et peuvent prendre le relais en cas de panne.

La configuration `broker.rack` permet d'indiquer à Kafka la topologie physique du cluster (racks, zones de disponibilité). Kafka utilisera cette information pour placer les réplicas sur des racks différents, améliorant la tolérance aux pannes physiques.

L'outil `kafka-reassign-partitions` permet de redistribuer manuellement les partitions, utile lors de l'ajout de brokers ou pour corriger un déséquilibre. Cette opération déplace des données et doit être planifiée en période creuse.

---

## III.2.3 Représentation Physique : Segments de Log et Indexation

### Structure du Répertoire de Données

Chaque broker Kafka stocke ses données dans un répertoire configuré par `log.dirs`. La structure de ce répertoire reflète l'organisation logique en topics et partitions.

```
/var/kafka-logs/
├── orders.created-0/           # Topic "orders.created", partition 0
│   ├── 00000000000000000000.log
│   ├── 00000000000000000000.index
│   ├── 00000000000000000000.timeindex
│   ├── 00000000000000523456.log
│   ├── 00000000000000523456.index
│   ├── 00000000000000523456.timeindex
│   ├── leader-epoch-checkpoint
│   └── partition.metadata
├── orders.created-1/           # Topic "orders.created", partition 1
│   └── ...
├── inventory.updated-0/        # Topic "inventory.updated", partition 0
│   └── ...
└── __consumer_offsets-0/       # Topic système pour les offsets
    └── ...
```

Chaque partition est représentée par un répertoire nommé `<topic>-<partition_number>`. Ce répertoire contient les segments de log et leurs fichiers d'index associés.

### Anatomie des Segments

Un **segment** est un fichier contenant une séquence contiguë de messages. Les segments sont l'unité de base pour la gestion du stockage — c'est au niveau du segment que s'appliquent les politiques de rétention et de compaction.

**Le fichier de log (.log).** Contient les messages sérialisés dans le format de batch décrit précédemment. Le nom du fichier correspond à l'offset du premier message qu'il contient. Par exemple, `00000000000000523456.log` commence à l'offset 523456.

Le segment actif (celui qui reçoit les nouvelles écritures) reste ouvert. Quand il atteint sa taille maximale (`log.segment.bytes`, défaut 1 Go) ou son âge maximal (`log.segment.ms`), il est « roulé » : fermé et renommé, et un nouveau segment est créé.

**L'index d'offsets (.index).** Permet de localiser rapidement un message par son offset sans parcourir tout le segment. C'est un index « sparse » — il ne contient pas une entrée pour chaque message mais une entrée tous les N octets (`log.index.interval.bytes`, défaut 4 Ko).

Chaque entrée de l'index est une paire `(offset_relatif, position_physique)` où l'offset relatif est la différence avec l'offset de base du segment, et la position physique est l'offset en octets dans le fichier .log.

Pour trouver un message à l'offset O dans un segment débutant à l'offset B, Kafka effectue une recherche binaire dans l'index pour trouver l'entrée avec le plus grand offset ≤ (O-B), puis scanne linéairement le fichier .log depuis cette position.

**L'index temporel (.timeindex).** Permet de localiser un message par son timestamp. Structure similaire à l'index d'offsets mais avec des entrées `(timestamp, offset)`. Utile pour les consommateurs qui veulent commencer à une date spécifique plutôt qu'à un offset.

### Optimisations d'I/O

Kafka utilise plusieurs techniques d'optimisation des I/O qui expliquent ses performances exceptionnelles. Comprendre ces optimisations permet à l'architecte de dimensionner correctement l'infrastructure et de diagnostiquer les problèmes de performance.

**Écriture séquentielle.** Les messages sont toujours ajoutés à la fin du segment courant — jamais insérés au milieu ou modifiés. Cette approche « append-only » est idéale pour les disques, même les disques rotatifs (HDD), car elle élimine les seeks aléatoires.

Les disques modernes, qu'ils soient HDD ou SSD, offrent des performances d'écriture séquentielle qui dépassent largement celles des écritures aléatoires. Un HDD typique peut atteindre 100-200 Mo/s en écriture séquentielle mais seulement quelques Mo/s en écriture aléatoire. Cette différence explique pourquoi Kafka peut atteindre des débits élevés même sur du matériel standard.

**Zero-copy transfer.** Lors de l'envoi de messages aux consommateurs, Kafka utilise le mécanisme `sendfile()` du système d'exploitation pour transférer les données directement du cache de pages vers le socket réseau, sans copie intermédiaire dans l'espace utilisateur. Cette optimisation réduit drastiquement l'utilisation CPU pour les transferts réseau.

Sans zero-copy, un transfert de données implique quatre copies : du disque vers le buffer du noyau, du buffer du noyau vers le buffer applicatif, du buffer applicatif vers le buffer de socket, et du buffer de socket vers la carte réseau. Avec zero-copy, seules deux copies sont nécessaires (disque vers buffer du noyau, buffer du noyau vers carte réseau), et le CPU n'intervient pas dans le transfert.

**Exploitation du Page Cache.** Kafka s'appuie sur le cache de pages du système d'exploitation plutôt que sur un cache applicatif. Les messages récemment écrits restent en mémoire (page cache) et peuvent être lus par les consommateurs sans accès disque. Cette approche est simple, efficace, et bénéficie des optimisations du noyau.

Le page cache est géré par le système d'exploitation selon des algorithmes LRU (Least Recently Used) sophistiqués. Les données fréquemment accédées restent en cache ; les données anciennes sont évincées quand la mémoire est nécessaire. Pour Kafka, cela signifie que les consommateurs « à jour » (qui lisent les messages récents) bénéficient de lectures depuis le cache, tandis que les consommateurs en retard (qui lisent des messages anciens) accèdent au disque.

**Batching et compression.** Comme décrit précédemment, les messages sont regroupés en batches et compressés, réduisant le nombre d'I/O et le volume de données transférées. Le batching est particulièrement efficace car il amortit le coût fixe de chaque opération I/O sur plusieurs messages.

**Read-ahead et write-behind.** Le système d'exploitation anticipe les lectures séquentielles (read-ahead) en chargeant proactivement les données suivantes en mémoire. De même, les écritures peuvent être bufferisées (write-behind) avant d'être persistées sur disque. Kafka bénéficie de ces optimisations grâce à son pattern d'accès séquentiel prévisible.

### Impact sur le Dimensionnement Matériel

Ces optimisations ont des implications directes sur les choix de matériel pour un cluster Kafka.

**Mémoire.** La majorité de la RAM doit être disponible pour le page cache, pas pour le heap JVM. Un broker avec 64 Go de RAM devrait avoir un heap JVM de 6-8 Go maximum, laissant ~56 Go pour le page cache. Cette configuration permet de servir les lectures récentes depuis la mémoire.

**Stockage.** Les SSD offrent de meilleures performances pour les lectures aléatoires (consommateurs en retard) mais les HDD suffisent pour les charges principalement séquentielles. Le choix dépend du ratio de consommateurs « à jour » vs. « en retard » et des exigences de latence.

**Réseau.** Le réseau devient souvent le goulot d'étranglement avant le disque ou le CPU. Des liens 10 Gbps ou plus sont recommandés pour les clusters à haut débit. Le réseau inter-broker (réplication) et le réseau client-broker partagent la bande passante et doivent être dimensionnés ensemble.

**CPU.** Contrairement à une idée reçue, Kafka n'est pas particulièrement gourmand en CPU dans les configurations par défaut. Cependant, la compression (surtout GZIP) et le chiffrement TLS peuvent devenir CPU-bound. Les processeurs modernes avec support matériel AES-NI (pour TLS) sont recommandés.

> **Perspective stratégique**
>
> Ces optimisations expliquent pourquoi Kafka peut atteindre des débits de plusieurs Go/s par broker avec du matériel standard. Elles expliquent aussi pourquoi les recommandations de configuration mettent l'accent sur la mémoire disponible pour le page cache plutôt que sur le heap JVM — Kafka est fondamentalement un système d'I/O, pas un système de traitement en mémoire.

### Configuration du Stockage

Plusieurs paramètres contrôlent le comportement du stockage et doivent être ajustés selon le cas d'usage.

**`log.segment.bytes`** (défaut : 1 Go). Taille maximale d'un segment avant roulement. Des segments plus petits permettent une granularité plus fine pour la rétention et la compaction, mais augmentent le nombre de fichiers à gérer. Des segments plus grands réduisent le nombre de fichiers mais rendent la rétention moins précise.

**`log.segment.ms`** (défaut : 7 jours). Âge maximal d'un segment avant roulement, même s'il n'a pas atteint sa taille maximale. Important pour les topics à faible débit où un segment pourrait ne jamais atteindre sa taille maximale.

**`log.index.interval.bytes`** (défaut : 4096). Intervalle entre les entrées de l'index. Une valeur plus petite crée des index plus précis mais plus volumineux. Une valeur plus grande économise de l'espace mais augmente le temps de recherche.

**`log.flush.interval.messages`** et **`log.flush.interval.ms`**. Contrôlent la fréquence de `fsync` vers le disque. Les valeurs par défaut (pas de flush explicite) s'appuient sur la réplication pour la durabilité plutôt que sur le flush disque. Dans la plupart des cas, les valeurs par défaut sont appropriées — la réplication est plus efficace que le flush synchrone pour assurer la durabilité.

> **Note de terrain**
>
> *Contexte* : Cluster Kafka avec des topics à débit très variable — certains reçoivent des millions de messages par jour, d'autres quelques centaines.
>
> *Observation* : Les topics à faible débit accumulent des segments qui ne sont jamais roulés, rendant la rétention imprécise (un segment de 7 jours ne peut être supprimé que quand tous ses messages ont expiré).
>
> *Solution* : Configurer `log.segment.ms` à 24h pour les topics à faible débit, assurant un roulement quotidien même avec peu de messages. Cela permet une rétention plus précise et facilite le monitoring.

---

## III.2.4 Durabilité et Haute Disponibilité : Modèle de Réplication

### Principes de la Réplication Kafka

La réplication est le mécanisme qui assure la durabilité des données et la haute disponibilité du service dans Kafka. Chaque partition est répliquée sur plusieurs brokers, et un protocole de consensus détermine quel réplica est le « leader » responsable des lectures et écritures.

Le **facteur de réplication** (`replication.factor`) détermine combien de copies de chaque partition existent dans le cluster. Un facteur de 3 signifie que chaque partition existe sur 3 brokers différents. Le facteur de réplication est configuré par topic et ne peut pas être inférieur au nombre de brokers disponibles.

> **Définition formelle**
>
> Le **facteur de réplication (RF)** est le nombre total de réplicas pour chaque partition d'un topic. Avec RF=N, le système tolère la perte de N-1 brokers sans perte de données. Un facteur de réplication de 3 est le standard de l'industrie pour les environnements de production, offrant un bon équilibre entre durabilité et coût en stockage.

### Leader et Followers

Pour chaque partition, un réplica est désigné comme **leader** et les autres sont des **followers**. Cette distinction est fondamentale pour comprendre le flux de données dans Kafka.

**Le leader** est le seul réplica qui accepte les écritures des producteurs et sert les lectures des consommateurs. Toutes les opérations de données passent par le leader. Cette centralisation simplifie la coordination et garantit un ordre cohérent des messages.

**Les followers** répliquent passivement les données du leader. Ils envoient des requêtes « fetch » au leader pour récupérer les nouveaux messages, de façon similaire à ce que font les consommateurs. Les followers ne servent pas directement les clients (sauf configuration spéciale avec `replica.selector.class`).

La **réplication est asynchrone** par défaut — le leader n'attend pas que tous les followers aient répliqué un message avant de confirmer l'écriture au producteur. Le niveau de synchronisation est contrôlé par le paramètre `acks` du producteur.

### In-Sync Replicas (ISR)

Le concept d'**In-Sync Replicas (ISR)** est central dans le modèle de réplication Kafka. L'ISR est l'ensemble des réplicas considérés comme « synchronisés » avec le leader — ceux qui ont répliqué tous les messages du leader (ou presque).

Un réplica est considéré in-sync s'il a communiqué avec le leader récemment (`replica.lag.time.max.ms`, défaut 30 secondes). Si un follower prend trop de retard, il est retiré de l'ISR. Quand il rattrape son retard, il est réintégré.

L'ISR est crucial pour les garanties de durabilité. Avec `acks=all`, le producteur attend que tous les réplicas de l'ISR aient confirmé l'écriture. Si l'ISR est réduit à un seul réplica (le leader), `acks=all` n'offre pas plus de protection que `acks=1`.

Le paramètre `min.insync.replicas` définit le nombre minimal de réplicas in-sync requis pour accepter les écritures avec `acks=all`. Avec RF=3 et `min.insync.replicas=2`, le système refuse les écritures si l'ISR tombe à 1 réplica, préférant l'indisponibilité à la perte de données potentielle.

> **Exemple concret**
>
> *Configuration* : RF=3, `min.insync.replicas=2`, producteur avec `acks=all`.
>
> *Scénario 1* : Les 3 réplicas sont sains. L'ISR contient 3 réplicas. Les écritures sont acceptées après confirmation de 3 réplicas.
>
> *Scénario 2* : Un broker tombe. L'ISR passe à 2 réplicas. Les écritures sont toujours acceptées (≥ min.insync.replicas).
>
> *Scénario 3* : Un deuxième broker tombe. L'ISR passe à 1 réplica. Les écritures sont refusées (< min.insync.replicas) avec l'erreur `NotEnoughReplicas`.
>
> *Compromis* : Cette configuration privilégie la durabilité (pas de perte de données) sur la disponibilité (écritures bloquées si trop de brokers down).

### Évolution vers KRaft : Élimination de ZooKeeper

Apache Kafka a historiquement utilisé ZooKeeper pour la gestion des métadonnées du cluster et la coordination des brokers. À partir de Kafka 3.0, un nouveau protocole de métadonnées appelé **KRaft** (Kafka Raft) permet de s'affranchir de cette dépendance externe.

**Limitations de ZooKeeper.** ZooKeeper introduisait plusieurs contraintes : une dépendance externe à déployer et opérer séparément, une limite pratique sur le nombre de partitions (~200 000 par cluster) due aux métadonnées stockées dans ZooKeeper, un temps de récupération après panne proportionnel à la taille des métadonnées, et une complexité opérationnelle accrue (deux systèmes à maintenir).

**Architecture KRaft.** Avec KRaft, les métadonnées du cluster sont stockées dans un topic Kafka spécial (`__cluster_metadata`) et répliquées entre des brokers désignés comme « controllers » utilisant le protocole de consensus Raft. Cette architecture élimine ZooKeeper tout en conservant les garanties de cohérence.

Les avantages de KRaft incluent une architecture simplifiée (un seul système), une scalabilité accrue (millions de partitions possibles), un temps de récupération réduit (métadonnées propagées efficacement), et une base pour de futures améliorations (snapshots, réplication cross-datacenter des métadonnées).

**Migration.** Pour les clusters existants, Kafka fournit un chemin de migration de ZooKeeper vers KRaft. La migration peut être réalisée en ligne sans interruption de service. L'architecte planifiant un nouveau déploiement Kafka devrait privilégier KRaft dès le départ.

> **Perspective stratégique**
>
> KRaft représente l'avenir de Kafka. Les nouveaux déploiements devraient utiliser KRaft par défaut. Les clusters existants devraient planifier la migration vers KRaft pour bénéficier des améliorations de scalabilité et de simplicité opérationnelle. ZooKeeper sera éventuellement déprécié dans les versions futures de Kafka.

### Élection du Leader

Quand le leader d'une partition devient indisponible (crash du broker, maintenance), un nouveau leader doit être élu parmi les followers. Ce processus d'**élection du leader** est géré par le contrôleur du cluster.

**Avec ZooKeeper (versions < 3.0).** Un broker est élu « contrôleur » et maintient les métadonnées du cluster dans ZooKeeper. Lors d'une panne de leader, le contrôleur choisit un nouveau leader parmi les réplicas de l'ISR et notifie les brokers concernés.

**Avec KRaft (versions ≥ 3.0).** Le protocole KRaft (Kafka Raft) élimine la dépendance à ZooKeeper. Les métadonnées sont gérées par un quorum de brokers contrôleurs utilisant le protocole Raft pour le consensus. L'élection du leader suit les mêmes principes mais avec une implémentation différente.

Le paramètre `unclean.leader.election.enable` (défaut : false) contrôle si un réplica hors de l'ISR peut être élu leader. Avec `false`, si tous les réplicas de l'ISR sont indisponibles, la partition devient indisponible (pas de leader). Avec `true`, un réplica retardataire peut devenir leader, mais les messages non répliqués sur ce réplica sont perdus. Ce paramètre est un compromis explicite disponibilité vs. durabilité.

> **Décision architecturale**
>
> *Contexte* : Configuration de `unclean.leader.election.enable` pour un cluster de production.
>
> *Analyse* :
> - `false` : Durabilité maximale, mais risque d'indisponibilité si tous les réplicas ISR sont down.
> - `true` : Disponibilité maximale, mais risque de perte de messages en cas de failover vers un réplica retardataire.
>
> *Décision* : `false` pour les topics critiques (transactions financières, audit), `true` acceptable pour les topics non critiques (logs, métriques) où la perte ponctuelle est tolérable.
>
> *Note* : Cette configuration peut être définie par topic via la propriété `unclean.leader.election.enable` du topic.

### Configuration de la Réplication

Plusieurs paramètres contrôlent le comportement de la réplication et doivent être ajustés selon les exigences de durabilité et de performance.

**`replication.factor`** (défaut : 1). Le facteur de réplication par défaut pour les topics créés automatiquement. Pour la production, RF=3 est le standard. RF=2 offre une protection minimale (tolérance à 1 panne). RF=1 n'offre aucune protection et ne devrait être utilisé qu'en développement.

**`min.insync.replicas`** (défaut : 1). Le nombre minimal de réplicas in-sync requis pour les écritures avec `acks=all`. Avec RF=3, `min.insync.replicas=2` est recommandé. Cette configuration assure qu'au moins 2 copies existent avant de confirmer une écriture.

**`replica.lag.time.max.ms`** (défaut : 30000). Le temps maximal de retard avant qu'un follower soit exclu de l'ISR. Une valeur trop basse cause des exclusions fréquentes lors de pics de charge. Une valeur trop haute retarde la détection des followers défaillants.

**`num.replica.fetchers`** (défaut : 1). Le nombre de threads utilisés par chaque broker pour répliquer les données des leaders. Augmenter cette valeur peut améliorer le débit de réplication mais consomme plus de ressources réseau et CPU.

### Topologie Multi-Datacenter et Disaster Recovery

Pour les déploiements critiques, la réplication au sein d'un seul datacenter n'est pas suffisante. L'architecte doit considérer des topologies multi-datacenter pour assurer la continuité de service en cas de sinistre majeur.

**Stretch cluster.** Une approche consiste à déployer un cluster Kafka unique dont les brokers sont répartis sur plusieurs datacenters. Avec `broker.rack` configuré pour identifier les datacenters, Kafka place les réplicas sur des datacenters différents. Cette approche offre une haute disponibilité transparente mais avec des contraintes : la latence réseau inter-datacenter impacte les performances de réplication, le quorum doit considérer la topologie (3 datacenters minimum pour éviter le split-brain), et le coût réseau inter-datacenter peut être significatif.

**MirrorMaker / Cluster Linking.** L'approche alternative est de maintenir des clusters Kafka indépendants par datacenter et de répliquer les données entre eux. MirrorMaker 2 (open source) et Cluster Linking (Confluent) permettent cette réplication asynchrone. Les avantages incluent l'isolation des pannes (un cluster peut continuer même si la réplication échoue), des performances optimales par cluster, et la flexibilité de répliquer sélectivement certains topics. Les inconvénients sont la complexité opérationnelle accrue, le RPO non nul (les données en transit peuvent être perdues), et le besoin de mécanismes de basculement applicatif.

**Active-Active vs Active-Passive.** Dans une configuration Active-Active, les deux datacenters servent simultanément les producteurs et consommateurs, avec réplication bidirectionnelle. La gestion des conflits (messages identiques produits des deux côtés) est complexe. Dans une configuration Active-Passive, un seul datacenter est actif à la fois, l'autre étant un standby qui réplique les données. Le basculement (failover) nécessite une coordination applicative mais évite les conflits.

> **Décision architecturale**
>
> *Contexte* : Architecture de disaster recovery pour un système de paiements avec RPO=0 (aucune perte de données) et RTO<5 minutes (reprise rapide).
>
> *Analyse* :
> - Stretch cluster : RPO=0 possible avec acks=all, mais latence élevée et complexité opérationnelle.
> - MirrorMaker Active-Passive : RPO>0 (données en transit perdues), mais opérationnellement plus simple.
>
> *Décision* : Stretch cluster sur 3 zones de disponibilité (AZ) au sein de la même région cloud. Les zones sont suffisamment proches pour une latence acceptable (<10ms) tout en offrant une isolation physique. Le DR multi-région utilise MirrorMaker avec un RPO accepté de quelques secondes.
>
> *Justification* : Le RPO=0 intra-région protège contre les pannes de zone. Le DR multi-région avec RPO>0 est un compromis acceptable car une panne régionale complète est extrêmement rare.

### Impact de la Configuration `acks`

Le paramètre `acks` du producteur détermine le niveau de confirmation attendu avant de considérer une écriture comme réussie. Ce paramètre a un impact direct sur la durabilité, la latence, et le débit.

| Configuration | Comportement | Durabilité | Latence | Débit |
|---------------|-------------|------------|---------|-------|
| `acks=0` | Pas d'attente de confirmation | Aucune garantie | Minimale | Maximal |
| `acks=1` | Attente de confirmation du leader | Perte possible si crash leader | Modérée | Élevé |
| `acks=all` | Attente de confirmation de tous les ISR | Maximale (selon `min.insync.replicas`) | Plus élevée | Réduit |

**`acks=0`** offre les meilleures performances mais aucune garantie de durabilité. Le producteur envoie le message et considère l'écriture réussie immédiatement, sans attendre de réponse. Utile pour les métriques ou logs où la perte occasionnelle est acceptable.

**`acks=1`** est le compromis par défaut. Le producteur attend la confirmation du leader. Si le leader crashe après avoir confirmé mais avant réplication, les messages peuvent être perdus. Bon compromis pour la majorité des cas d'usage.

**`acks=all`** (ou `acks=-1`) offre la durabilité maximale. Le producteur attend que tous les réplicas de l'ISR aient confirmé. Combiné avec `min.insync.replicas=2` et RF=3, cette configuration garantit qu'au moins 2 copies existent pour chaque message confirmé.

> **Perspective stratégique**
>
> Le choix de `acks` doit être fait en fonction des exigences métier, pas des performances. Les transactions financières, les commandes clients, et les données réglementées justifient `acks=all` malgré le coût en latence. Les logs applicatifs et les métriques peuvent souvent se contenter de `acks=1` voire `acks=0`.
>
> L'architecte doit classifier les topics par niveau de criticité et appliquer la configuration `acks` appropriée à chaque catégorie.

---

## III.2.5 Gestion du Cycle de Vie des Données

### Politiques de Rétention

Les données dans Kafka ne sont pas conservées indéfiniment (sauf configuration explicite). La **politique de rétention** détermine quand les anciens messages sont supprimés. Deux mécanismes principaux existent : la rétention temporelle et la rétention par taille.

**Rétention temporelle (`retention.ms`).** Les messages plus anciens que la durée configurée sont éligibles à la suppression. La valeur par défaut est de 7 jours (604800000 ms). Cette politique est la plus courante car elle fournit une garantie de fraîcheur des données : tout message dans le topic a été publié dans les N derniers jours.

La rétention s'applique au niveau du segment, pas du message individuel. Un segment ne peut être supprimé que si tous ses messages ont expiré. C'est pourquoi un segment peut contenir des messages plus vieux que la rétention configurée si le segment n'a pas été roulé. Pour une rétention précise, il est recommandé de configurer `log.segment.ms` à une valeur inférieure à `retention.ms` (par exemple, `log.segment.ms` = 24h pour une rétention de 7 jours).

**Rétention par taille (`retention.bytes`).** Les segments les plus anciens sont supprimés quand la taille totale de la partition dépasse le seuil configuré. Cette politique est utile pour les environnements à stockage contraint. La valeur par défaut est -1 (pas de limite de taille).

La rétention par taille s'applique par partition, pas par topic. Un topic avec 10 partitions et `retention.bytes=1Go` pourra consommer jusqu'à 10 Go au total. Pour contrôler la taille totale d'un topic, multiplier la valeur souhaitée par le nombre de partitions.

Les deux politiques peuvent être combinées — les segments sont supprimés dès qu'une des conditions est remplie (temps OU taille). Cette combinaison est utile pour avoir une rétention temporelle normale tout en ayant une limite de sécurité sur la taille.

**Rétention infinie.** En configurant `retention.ms=-1` et `retention.bytes=-1`, les données sont conservées indéfiniment. Cette configuration est appropriée pour les topics utilisés comme source de vérité (event sourcing) mais requiert une planification du stockage à long terme et une stratégie d'archivage si les volumes deviennent importants.

**Rétention différenciée par topic.** Kafka permet de configurer la rétention individuellement par topic via la commande `kafka-configs.sh` ou lors de la création du topic. Cette flexibilité permet d'adapter la rétention aux besoins de chaque cas d'usage : logs applicatifs (24-48h), événements métier (7-30 jours), données de conformité (7 ans avec tiered storage).

> **Exemple concret**
>
> *Scénario* : Un topic de logs applicatifs avec un débit de 100 Mo/heure.
>
> *Configuration initiale* : `retention.ms=604800000` (7 jours), `retention.bytes=-1` (pas de limite).
>
> *Volume résultant* : 100 Mo × 24h × 7 jours = ~17 Go par partition.
>
> *Ajustement possible* : Si l'équipe n'a besoin que des logs des dernières 24h pour le debugging, réduire à `retention.ms=86400000` (1 jour) économise ~85 % du stockage.
>
> *Alternative* : Configurer `retention.bytes=5368709120` (5 Go) pour limiter l'espace quelle que soit la durée.

### Compaction des Logs

La **compaction** est une alternative à la suppression basée sur le temps. Avec la compaction, Kafka conserve au minimum le dernier message pour chaque clé unique, supprimant les messages antérieurs avec la même clé.

La compaction transforme le topic en une sorte de « table » où chaque clé a une valeur courante. Un nouveau consommateur peut lire le topic compacté pour obtenir l'état actuel de toutes les clés sans parcourir l'historique complet. Cette caractéristique est fondamentale pour plusieurs patterns architecturaux.

**Configuration de la compaction.** La politique de nettoyage est configurée par le paramètre `cleanup.policy` du topic. Les valeurs possibles sont `delete` (rétention temporelle/taille), `compact` (compaction), ou `compact,delete` (les deux).

**Processus de compaction.** Un thread de compaction en arrière-plan (le « log cleaner ») identifie les segments éligibles et les réécrit en conservant uniquement le dernier message pour chaque clé. Le processus est progressif et n'impacte pas la disponibilité du topic.

Le cleaner maintient un ratio entre les données « dirty » (non compactées) et « clean » (déjà compactées). Le paramètre `min.cleanable.dirty.ratio` (défaut 0.5) détermine quand déclencher la compaction — avec 0.5, la compaction démarre quand 50% ou plus des données sont dirty.

La compaction ne supprime pas immédiatement les anciens messages — elle les marque pour suppression et les élimine lors de la prochaine réécriture de segment. Le paramètre `min.compaction.lag.ms` peut être configuré pour garantir que les messages récents ne soient pas compactés immédiatement, laissant une fenêtre pour les consommateurs qui pourraient avoir besoin de l'historique court terme.

**Tombstones.** Pour supprimer une clé, le producteur publie un message avec cette clé et une valeur nulle. Ce message « tombstone » signale au compacteur de supprimer la clé. Après une période configurable (`delete.retention.ms`), le tombstone lui-même est supprimé.

La gestion des tombstones requiert une attention particulière. Si les tombstones sont supprimés trop rapidement, un nouveau consommateur pourrait ne pas recevoir l'information de suppression et croire que la clé existe toujours. Le paramètre `delete.retention.ms` (défaut 24h) doit être configuré en fonction de la fréquence de lecture complète du topic par les consommateurs.

**Compaction et ordre des messages.** La compaction préserve l'ordre relatif des messages pour une même clé (le dernier message est conservé) mais ne garantit pas l'ordre entre clés différentes. Pour les topics compactés, l'ordre inter-clé n'a généralement pas de signification métier car seul l'état final de chaque clé compte.

**Considérations de performance.** La compaction consomme des ressources CPU et I/O. Pour les topics à très haut débit avec de nombreuses mises à jour par clé, le cleaner peut devenir un goulot d'étranglement. Les paramètres `log.cleaner.threads`, `log.cleaner.io.buffer.size`, et `log.cleaner.dedupe.buffer.size` permettent d'ajuster les ressources allouées au cleaner.

> **Définition formelle**
>
> Une **tombstone** est un message avec une clé non nulle et une valeur nulle. Elle signale l'intention de supprimer cette clé du topic compacté. La tombstone est conservée pendant une durée configurable (`delete.retention.ms`, défaut 24h) pour permettre aux consommateurs de la voir et de réagir, puis elle est supprimée lors de la prochaine compaction.

### Cas d'Usage de la Compaction

La compaction est particulièrement utile pour plusieurs scénarios architecturaux.

**Tables de référence.** Un topic compacté contenant les données de référence (clients, produits, configuration) peut servir de source pour les consommateurs. Chaque mise à jour remplace la version précédente. Un nouveau consommateur lit le topic entier pour charger l'état initial.

**CQRS et projections.** Dans une architecture CQRS, les projections peuvent être reconstruites à partir d'un topic compacté contenant l'état actuel des agrégats.

**Changelog de connecteurs.** Kafka Connect utilise des topics compactés pour stocker les offsets des connecteurs et l'état des tâches.

**Topic `__consumer_offsets`.** Le topic système qui stocke les offsets des consumer groups utilise la compaction — seul le dernier offset commité pour chaque partition de consumer group est conservé.

### Paramètres de Configuration

| Paramètre | Défaut | Description |
|-----------|--------|-------------|
| `cleanup.policy` | delete | Politique de nettoyage (delete, compact, compact,delete) |
| `retention.ms` | 604800000 (7j) | Rétention temporelle pour la politique delete |
| `retention.bytes` | -1 | Rétention par taille pour la politique delete |
| `min.cleanable.dirty.ratio` | 0.5 | Ratio minimum de données « dirty » pour déclencher la compaction |
| `delete.retention.ms` | 86400000 (24h) | Durée de conservation des tombstones |
| `segment.ms` | 604800000 (7j) | Durée maximale d'un segment avant roulement |
| `segment.bytes` | 1073741824 (1Go) | Taille maximale d'un segment avant roulement |

### Tiered Storage

Introduit dans Kafka 3.0 et stabilisé dans les versions récentes, le **Tiered Storage** permet de décharger les segments anciens vers un stockage objet (S3, GCS, Azure Blob) moins coûteux que le stockage local des brokers.

**Principe.** Les segments récents restent sur le stockage local des brokers pour les accès rapides. Les segments anciens sont copiés vers le stockage objet et peuvent être supprimés localement. Les lectures d'anciens messages récupèrent transparentement les données depuis le stockage objet.

**Avantages.** Réduction significative des coûts de stockage pour les longues rétentions. Séparation du compute (brokers) et du storage (objet). Possibilité de conserver des historiques très longs sans impacter les performances des brokers.

**Configuration.** Le tiered storage est configuré au niveau du cluster et peut être activé/désactivé par topic. Les paramètres clés incluent `remote.storage.enable`, les détails de connexion au stockage objet, et les politiques de tiering (quand déplacer les segments).

> **Perspective stratégique**
>
> Le Tiered Storage change l'économie de la rétention Kafka. Auparavant, une rétention de 30 jours vs. 7 jours avait un impact direct sur le coût du cluster (4× plus de stockage). Avec le tiered storage, les données au-delà des premiers jours sont stockées à un coût marginal faible (stockage objet).
>
> Cette évolution permet des architectures où Kafka sert véritablement de « source de vérité » avec des rétentions de mois ou d'années, là où c'était économiquement prohibitif auparavant.

---

## III.2.6 Recommandations Architecturales et Bonnes Pratiques

### Conception des Topics

La conception des topics est une décision architecturale fondamentale qui impacte la gouvernance, les performances, et l'évolutivité du système.

**Granularité des topics.** L'architecte doit trouver l'équilibre entre des topics trop fins (explosion du nombre de topics, complexité de gestion) et des topics trop larges (perte de flexibilité, filtrage coûteux). Une approche recommandée est d'avoir un topic par type d'événement métier significatif (par exemple, `orders.created`, `orders.shipped`, `orders.cancelled` plutôt qu'un seul topic `orders`).

**Convention de nommage.** Établir et documenter une convention de nommage dès le départ. Une structure recommandée est `<domaine>.<entité>.<action>[.<version>]`. Les noms doivent être en minuscules, utiliser des points comme séparateurs (pas des tirets ou underscores dans la structure hiérarchique), et être stables (ne pas inclure d'éléments variables comme des dates).

**Métadonnées des topics.** Kafka ne fournit pas de mécanisme natif pour documenter les topics. L'architecte doit mettre en place un catalogue de topics (Schema Registry, Confluent Data Catalog, outil interne) qui documente le schéma, le producteur, les consommateurs autorisés, et la sémantique de chaque topic.

> **Anti-patron**
>
> *« Nous créons un topic par client : customer-123-events, customer-456-events, etc. »* Cette approche crée une explosion du nombre de topics (potentiellement des millions), chacun avec très peu de messages. Elle complique la gestion, dégrade les performances des métadonnées, et empêche le traitement agrégé.
>
> *Meilleure approche* : Un topic unique `customer.events` avec l'ID client comme clé de partitionnement. Les événements de chaque client sont ordonnés (même partition), et le système reste gérable.

### Stratégie de Partitionnement

Le choix du nombre de partitions et de la clé de partitionnement requiert une analyse des besoins métier et techniques.

**Analyse du parallélisme requis.** Estimer le nombre de consommateurs parallèles nécessaires pour traiter la charge. Le nombre de partitions doit être au moins égal à ce nombre. Prévoir une marge pour la croissance future car augmenter les partitions est possible mais peut perturber l'ordre existant.

**Analyse des clés naturelles.** Identifier les clés métier qui préservent les invariants d'ordre. Pour un système de commandes, la clé naturelle est souvent l'ID de commande ou l'ID client, selon les exigences de traitement.

**Détection des « hot keys ».** Analyser la distribution des clés pour détecter les déséquilibres potentiels. Si 10 % des clés génèrent 90 % du trafic, envisager des stratégies de mitigation (clé composite, partitionnement personnalisé).

**Documentation du contrat de partitionnement.** Documenter explicitement la sémantique du partitionnement : quelle clé est utilisée, quelles garanties d'ordre en découlent, quelles sont les limitations connues.

### Configuration de la Réplication

La configuration de la réplication doit être adaptée aux exigences de durabilité et de disponibilité de chaque catégorie de topics.

**Topics critiques (données transactionnelles, audit).** Configuration recommandée : RF=3, `min.insync.replicas=2`, producteurs avec `acks=all`. Cette configuration garantit qu'au moins 2 copies existent pour chaque message et refuse les écritures si ce n'est pas possible.

**Topics standards (événements métier).** Configuration recommandée : RF=3, `min.insync.replicas=1`, producteurs avec `acks=1`. Cette configuration offre une bonne durabilité avec de meilleures performances que la configuration critique.

**Topics non critiques (logs, métriques).** Configuration acceptable : RF=2, `min.insync.replicas=1`, producteurs avec `acks=1` ou `acks=0`. Cette configuration optimise les coûts et les performances au détriment de la durabilité maximale.

### Stratégie de Rétention

La politique de rétention doit être définie en fonction des besoins métier et des contraintes de coût.

**Analyse des besoins de relecture.** Pour combien de temps les consommateurs peuvent-ils avoir besoin de revenir en arrière ? Les cas d'usage courants sont 24-48h pour le debugging, 7-30 jours pour la reconstruction de projections, et plus pour l'audit ou l'event sourcing.

**Analyse des contraintes de coût.** Estimer le volume de stockage pour différentes durées de rétention. Avec tiered storage, les coûts de longue rétention sont réduits mais pas nuls.

**Classification des topics.** Établir des classes de rétention standard (par exemple : « ephemeral » 24h, « standard » 7 jours, « extended » 30 jours, « permanent » avec compaction) et assigner chaque topic à une classe.

### Monitoring et Alerting

Un cluster Kafka en production requiert un monitoring rigoureux pour détecter les problèmes avant qu'ils n'impactent les applications.

**Métriques essentielles des brokers.** Le nombre de partitions under-replicated (ISR < RF) indique des problèmes de réplication. Le taux d'élection de leaders indique des instabilités. L'utilisation disque, réseau, CPU révèle les goulots d'étranglement. Le nombre de requêtes en queue sur le contrôleur indique une surcharge.

**Métriques essentielles des producteurs.** Le taux d'erreurs de production et les types d'erreurs révèlent les problèmes de communication. La latence de production (temps de réponse) indique la santé du cluster. La taille des batches et le ratio de compression mesurent l'efficacité.

**Métriques essentielles des consommateurs.** Le consumer lag (retard de traitement) est critique — un lag croissant indique que les consommateurs ne suivent pas. Le taux de rééquilibrage indique une instabilité du consumer group. Le throughput de consommation mesure la capacité de traitement.

**Alertes recommandées.** Un lag consumer > seuil pendant > 5 minutes est une alerte haute priorité. Des partitions under-replicated > 0 pendant > 10 minutes nécessitent une investigation. Un disque > 80 % requiert une action préventive. Des erreurs de production en rafale indiquent un problème systémique.

> **Note de terrain**
>
> *Contexte* : Mise en place du monitoring pour un nouveau cluster de production.
>
> *Erreur initiale* : Configuration d'alertes sur toutes les métriques disponibles, résultant en des dizaines d'alertes par jour, la plupart sans impact réel. L'équipe a commencé à ignorer les alertes (« alert fatigue »).
>
> *Correction* : Réduction aux métriques vraiment actionnables : lag consumer (impact direct sur les applications), partitions under-replicated (risque de perte de données), espace disque critique (risque d'arrêt).
>
> *Résultat* : ~2-3 alertes par semaine, toutes nécessitant une action. L'équipe réagit rapidement car les alertes sont significatives.
>
> *Leçon* : Moins d'alertes de meilleure qualité est préférable à une couverture exhaustive qui génère du bruit.

### Sécurité

La sécurité d'un cluster Kafka en entreprise requiert une attention sur plusieurs dimensions.

**Authentification.** Configurer l'authentification pour tous les accès au cluster. SASL/SCRAM est recommandé pour la plupart des cas (bon équilibre sécurité/simplicité). mTLS est préférable pour les environnements à haute sécurité. Éviter SASL/PLAIN qui transmet les mots de passe en clair.

**Autorisation.** Configurer les ACL pour contrôler qui peut lire/écrire quels topics. Le principe du moindre privilège s'applique : donner à chaque application uniquement les permissions nécessaires. Pour les environnements complexes, considérer RBAC (Confluent) ou l'intégration avec un gestionnaire d'identités externe.

**Chiffrement.** Activer TLS pour les communications inter-brokers et client-broker. Le chiffrement au repos dépend des exigences réglementaires et peut être géré au niveau du système de fichiers ou du stockage.

**Gouvernance.** Mettre en place un processus de revue pour la création de topics et les modifications de permissions. Auditer régulièrement les accès et les permissions.

### Tests et Validation

La validation d'une architecture Kafka requiert des tests à plusieurs niveaux.

**Tests de charge.** Avant la mise en production, valider que le cluster supporte la charge prévue avec marge. Utiliser des outils comme `kafka-producer-perf-test` et `kafka-consumer-perf-test` pour mesurer le débit maximal. Tester avec des charges réalistes (distribution de clés, taille de messages, pattern de production).

**Tests de résilience.** Valider le comportement en cas de panne de broker, de partition réseau, de saturation disque. Vérifier que les failovers se produisent comme attendu et que les applications se comportent correctement (reconnexion, gestion des erreurs).

**Tests d'évolution.** Valider les procédures de mise à jour du cluster, d'ajout de brokers, de modification du nombre de partitions. Ces opérations doivent être maîtrisées avant d'être nécessaires en urgence.

**Chaos engineering.** Pour les systèmes critiques, considérer l'injection régulière de pannes contrôlées (chaos engineering) pour valider la résilience en conditions réelles.

---

## III.2.7 Résumé

Ce chapitre a exploré en profondeur l'architecture interne d'un cluster Apache Kafka. Cette compréhension technique est essentielle pour l'architecte qui doit dimensionner, configurer, et opérer la plateforme efficacement.

### Anatomie du Message

Le record Kafka se compose d'une clé (optionnelle, détermine le partitionnement), d'une valeur (les données métier), d'en-têtes (métadonnées), et d'un timestamp. Le choix de la clé est une décision architecturale critique qui détermine les garanties d'ordre.

Les messages sont regroupés en batches pour optimiser les I/O. La compression s'applique au niveau du batch, avec plusieurs algorithmes disponibles offrant différents compromis performance/ratio.

### Organisation Logique

Les topics sont l'abstraction principale de publication/abonnement. Chaque topic est divisé en partitions qui sont l'unité de parallélisme et d'ordre. L'ordre des messages est garanti au sein d'une partition mais pas entre partitions.

Le nombre de partitions détermine le parallélisme maximal des consommateurs et doit être dimensionné selon le débit attendu et les besoins de traitement parallèle. Un sur-provisionnement crée une charge opérationnelle inutile ; un sous-provisionnement limite les performances.

### Représentation Physique

Les partitions sont stockées comme des segments de fichiers sur le disque des brokers. Chaque segment est accompagné d'index permettant la recherche efficace par offset et par timestamp.

Kafka optimise les I/O par l'écriture séquentielle, le zero-copy transfer, et l'exploitation du page cache du système d'exploitation. Ces optimisations expliquent les performances exceptionnelles de la plateforme.

### Modèle de Réplication

La réplication assure la durabilité et la haute disponibilité. Chaque partition a un leader (qui sert les requêtes) et des followers (qui répliquent). L'ISR (In-Sync Replicas) est l'ensemble des réplicas synchronisés.

La configuration `acks` du producteur détermine le niveau de confirmation attendu. Combinée avec `min.insync.replicas`, elle permet d'ajuster le compromis durabilité/disponibilité/performance selon les exigences métier.

L'élection du leader lors d'une panne est gérée par le contrôleur (via ZooKeeper ou KRaft). La configuration `unclean.leader.election.enable` détermine si un réplica non synchronisé peut devenir leader.

### Cycle de Vie des Données

La rétention temporelle (`retention.ms`) et par taille (`retention.bytes`) contrôlent la suppression des anciens messages. La compaction (`cleanup.policy=compact`) préserve le dernier message pour chaque clé, transformant le topic en « table ».

Le Tiered Storage permet de décharger les segments anciens vers un stockage objet moins coûteux, changeant l'économie de la rétention longue.

### Bonnes Pratiques

La conception des topics doit suivre des conventions de nommage cohérentes et documenter la sémantique de chaque topic. La stratégie de partitionnement doit préserver les invariants métier tout en assurant une distribution équilibrée.

La configuration de la réplication doit être adaptée à la criticité de chaque catégorie de topics. Le monitoring doit se concentrer sur les métriques actionnables (lag, under-replicated partitions, espace disque) pour éviter l'alert fatigue.

La sécurité (authentification, autorisation, chiffrement) et les tests (charge, résilience, évolution) complètent les éléments d'une architecture Kafka robuste.

---

### Vers le Chapitre Suivant

Ce chapitre a établi les fondations techniques de l'architecture Kafka. Le chapitre suivant, « Clients Kafka et Production de Messages », explorera en détail le côté client de l'équation : comment les applications interagissent avec le cluster pour publier des messages efficacement et de manière fiable.

L'architecte qui maîtrise à la fois l'architecture du cluster (ce chapitre) et les patterns de production (chapitre suivant) sera équipé pour concevoir des systèmes événementiels performants et résilients.

---

*Volume III : Apache Kafka - Guide de l'Architecte*

*Chapitre III.2 — Architecture d'un Cluster Kafka*

*Monographie « L'Entreprise Agentique »*
