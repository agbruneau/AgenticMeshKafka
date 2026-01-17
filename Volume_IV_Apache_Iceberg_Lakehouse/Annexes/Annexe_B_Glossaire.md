# Annexe B - Glossaire

## Introduction

Ce glossaire rassemble les termes techniques essentiels utilisés dans le Volume IV et plus largement dans l'écosystème Apache Iceberg et Data Lakehouse. Il constitue un outil de référence normalisé pour assurer une compréhension cohérente de la terminologie à travers l'ensemble de la monographie.

Les définitions sont organisées en quatre sections thématiques : terminologie Apache Iceberg, terminologie Lakehouse et Data Engineering, terminologie Streaming et Kafka, et acronymes et abréviations. Pour chaque terme, nous fournissons une définition concise, des exemples d'utilisation lorsque pertinent, et des références croisées vers les termes connexes.

---

## B.1 Terminologie Apache Iceberg

### Catalog (Catalogue)

Service ou composant maintenant les références vers les fichiers de métadonnées courants des tables Iceberg. Le catalogue sert de point d'entrée unique pour accéder aux tables et garantit l'atomicité des commits via des mécanismes de verrouillage ou de compare-and-swap.

*Exemples* : REST Catalog, Hive Metastore Catalog, AWS Glue Catalog, JDBC Catalog, Apache Polaris

*Voir aussi* : REST Catalog, Metastore

### Compaction

Processus de maintenance consolidant plusieurs petits fichiers de données en fichiers plus grands et optimisés. La compaction améliore les performances de lecture en réduisant le nombre de fichiers à scanner et en optimisant la disposition des données selon les patterns d'accès.

*Types* :
- **Bin-packing** : Regroupement de fichiers sans réorganisation du contenu
- **Sort compaction** : Réorganisation et tri des données selon les colonnes fréquemment utilisées

*Voir aussi* : Maintenance, Rewrite Data Files

### Copy-on-Write (CoW)

Mode d'écriture où les modifications de données entraînent la réécriture complète des fichiers affectés. Une mise à jour d'une seule ligne nécessite la réécriture du fichier entier contenant cette ligne, créant une nouvelle version avec la modification appliquée.

*Avantages* : Lecture optimale sans réconciliation
*Inconvénients* : Amplification d'écriture significative

*Voir aussi* : Merge-on-Read, Write Amplification

### Data File (Fichier de données)

Fichier contenant les enregistrements réels d'une table Iceberg, stocké dans un format ouvert comme Parquet, ORC ou Avro. Les fichiers de données sont immuables après écriture et référencés par les fichiers de manifestes.

*Voir aussi* : Manifest File, Delete File

### Delete File (Fichier de suppression)

Fichier marquant des lignes comme supprimées sans réécrire les fichiers de données originaux. Utilisé en mode Merge-on-Read pour des suppressions et mises à jour efficaces.

*Types* :
- **Position delete file** : Identifie les lignes par chemin de fichier et position (V2, déprécié en V3)
- **Equality delete file** : Identifie les lignes par valeurs de colonnes

*Voir aussi* : Deletion Vector, Merge-on-Read

### Deletion Vector (Vecteur de suppression)

Structure binaire compacte introduite en V3 identifiant les lignes supprimées dans un fichier de données par leur position, stockée sous forme de bitmap Roaring dans un fichier Puffin. Plus efficace que les fichiers de suppression par position de V2.

*Caractéristiques* :
- Un seul vecteur par fichier de données dans un snapshot
- Format bitmap pour représentation compacte
- Stocké dans des fichiers Puffin

*Voir aussi* : Delete File, Puffin, Roaring Bitmap

### Equality Delete (Suppression par égalité)

Méthode de suppression identifiant les lignes à supprimer par les valeurs d'une ou plusieurs colonnes clés, plutôt que par leur position dans un fichier. Particulièrement utile pour les pipelines CDC où la clé primaire identifie les enregistrements.

*Voir aussi* : Delete File, Position Delete

### Hidden Partitioning (Partitionnement masqué)

Fonctionnalité d'Iceberg où les colonnes de partition ne sont pas exposées dans le schéma de la table. Les transformations de partition sont définies séparément et appliquées automatiquement, évitant les erreurs de requête liées aux colonnes de partition.

*Transformations supportées* : identity, bucket, truncate, year, month, day, hour

*Voir aussi* : Partition Evolution, Partition Spec

### Manifest File (Fichier de manifeste)

Fichier Avro contenant les métadonnées détaillées des fichiers de données : chemins, tailles, statistiques de colonnes (min, max, nulls, NaN), informations de partition et métadonnées de tri. Les manifestes permettent l'élagage efficace sans lister le stockage.

*Voir aussi* : Manifest List, Data File

### Manifest List (Liste de manifestes)

Fichier Avro listant les fichiers de manifestes d'un snapshot avec des métadonnées résumées : plages de partition, statistiques agrégées et compteurs de fichiers. Permet l'élagage rapide au niveau des manifestes.

*Voir aussi* : Manifest File, Snapshot

### Merge-on-Read (MoR)

Mode d'écriture où les modifications créent des fichiers de suppression réconciliés avec les données originales au moment de la lecture. Les mises à jour et suppressions sont appliquées pendant l'exécution des requêtes.

*Avantages* : Écriture rapide sans amplification
*Inconvénients* : Surcoût de lecture pour la réconciliation

*Voir aussi* : Copy-on-Write, Delete File

### Metadata File (Fichier de métadonnées)

Fichier JSON contenant la définition complète d'une table Iceberg : schéma, spécifications de partition, propriétés, snapshots, historique et statistiques. Le catalogue pointe vers le fichier de métadonnées courant.

*Contenu* : format-version, table-uuid, location, schemas, partition-specs, sort-orders, current-snapshot-id, snapshots, snapshot-log

*Voir aussi* : Snapshot, Schema Evolution

### Partition Evolution (Évolution de partition)

Capacité de modifier le schéma de partitionnement sans réécrire les données existantes. Les nouvelles données utilisent le nouveau schéma tandis que les anciennes conservent leur partitionnement original, avec réconciliation automatique lors des requêtes.

*Voir aussi* : Hidden Partitioning, Schema Evolution

### Partition Pruning (Élagage de partition)

Optimisation excluant les partitions non pertinentes de la planification de scan basée sur les prédicats de requête. L'élagage utilise les informations de partition stockées dans les manifestes sans accéder aux fichiers de données.

*Voir aussi* : File Pruning, Statistics

### Position Delete (Suppression par position)

Méthode de suppression identifiant les lignes par leur chemin de fichier de données et leur position de ligne (offset) dans ce fichier. Dépréciée en V3 au profit des vecteurs de suppression.

*Voir aussi* : Delete File, Deletion Vector

### Puffin

Format de fichier conteneur pour stocker des informations auxiliaires sur les données Iceberg : statistiques, index et vecteurs de suppression. Structure simple avec magic bytes, blobs binaires et footer JSON.

*Cas d'usage* :
- Esquisses Theta pour statistiques NDV
- Vecteurs de suppression (V3)
- Index secondaires (filtres Bloom)

*Voir aussi* : Deletion Vector, Statistics

### REST Catalog

Spécification d'API OpenAPI définissant une interface REST standardisée pour interagir avec les catalogues Iceberg. Permet l'interopérabilité entre clients et serveurs indépendamment de leur implémentation.

*Implémentations* : Apache Polaris, Project Nessie, AWS Glue, Google BigLake

*Voir aussi* : Catalog, OAuth 2.0

### Row Lineage (Lignage des lignes)

Métadonnées au niveau des lignes introduites en V3 pour le traitement incrémental. Chaque ligne reçoit un identifiant unique (`_row_id`) et un numéro de séquence de dernière mise à jour (`_last_updated_sequence_number`).

*Applications* : CDC, vues matérialisées, traitement différentiel

*Voir aussi* : Sequence Number, Snapshot

### Schema Evolution (Évolution de schéma)

Capacité de modifier le schéma d'une table (ajout, suppression, renommage de colonnes) sans réécrire les données existantes. Les identifiants de champs stables assurent la compatibilité entre versions de schéma.

*Opérations supportées* :
- Ajout de colonnes (avec valeur par défaut en V3)
- Suppression de colonnes
- Renommage de colonnes
- Réordonnancement de colonnes
- Élargissement de types (int → bigint)

*Voir aussi* : Partition Evolution, Field ID

### Sequence Number (Numéro de séquence)

Entier croissant assigné à chaque opération de commit, permettant d'ordonner les opérations et de déterminer l'applicabilité des fichiers de suppression. Un fichier de suppression s'applique uniquement aux fichiers de données avec un numéro de séquence inférieur ou égal.

*Voir aussi* : Snapshot, Delete File

### Snapshot

État complet et cohérent d'une table Iceberg à un instant donné. Chaque snapshot contient un identifiant unique, une référence au snapshot parent, un horodatage et un pointeur vers sa liste de manifestes.

*Voir aussi* : Time Travel, Snapshot Expiration

### Snapshot Expiration (Expiration des snapshots)

Processus de maintenance supprimant les snapshots anciens et les fichiers de données orphelins pour libérer l'espace de stockage. Configuré selon une politique de rétention (temps ou nombre de snapshots).

*Voir aussi* : Snapshot, Orphan Files

### Snapshot Isolation (Isolation des snapshots)

Niveau d'isolation transactionnelle où les lectures voient un état cohérent de la table correspondant à un snapshot spécifique, indépendamment des écritures concurrentes.

*Voir aussi* : ACID, Optimistic Concurrency

### Statistics (Statistiques)

Métadonnées collectées sur les données pour l'optimisation des requêtes. Iceberg maintient des statistiques au niveau des colonnes (min, max, null count, NaN count) dans les manifestes et des statistiques avancées (NDV, histogrammes) dans les fichiers Puffin.

*Voir aussi* : Puffin, Partition Pruning

### Time Travel (Voyage dans le temps)

Fonctionnalité permettant d'interroger une table dans un état historique, soit par identifiant de snapshot, soit par horodatage. Essentiel pour l'audit, le débogage et la reproductibilité analytique.

*Syntaxe SQL* :
```sql
SELECT * FROM table AS OF TIMESTAMP '2025-01-01';
SELECT * FROM table AS OF VERSION 12345678901234;
```

*Voir aussi* : Snapshot, Rollback

### Variant

Type de données semi-structuré introduit en V3 pour stocker des données JSON-like avec un ensemble élargi de types primitifs (date, timestamp, binary, decimal). Permet l'ingestion de données non typées sans enforcement strict de schéma.

*Voir aussi* : Schema Evolution, V3

---

## B.2 Terminologie Lakehouse et Data Engineering

### ACID

Acronyme pour Atomicité, Cohérence (Consistency), Isolation et Durabilité. Propriétés transactionnelles garantissant l'intégrité des données. Iceberg implémente des transactions ACID via le contrôle optimiste de concurrence et l'immutabilité des fichiers.

*Voir aussi* : Snapshot Isolation, Optimistic Concurrency

### CDC (Change Data Capture)

Processus capturant les modifications de données (insertions, mises à jour, suppressions) depuis une source pour les propager vers des systèmes cibles. Iceberg supporte le CDC via les fichiers de suppression et le lignage des lignes.

*Voir aussi* : Row Lineage, Merge-on-Read

### Column Pruning (Élagage de colonnes)

Optimisation lisant uniquement les colonnes requises par une requête, exploitant les formats de fichiers colonnaires comme Parquet. Réduit significativement les I/O et la mémoire utilisée.

*Voir aussi* : File Pruning, Parquet

### Data Lake

Architecture de stockage centralisée acceptant des données brutes de tous formats (structurées, semi-structurées, non structurées) à grande échelle. Typiquement basé sur du stockage objet comme Amazon S3, Azure ADLS ou Google Cloud Storage.

*Voir aussi* : Data Lakehouse, Object Storage

### Data Lakehouse

Architecture hybride combinant la flexibilité et le coût du Data Lake avec les capacités transactionnelles et la performance du Data Warehouse. Apache Iceberg est un format de table fondamental pour les architectures Lakehouse.

*Caractéristiques* : Transactions ACID, évolution de schéma, performance optimisée, stockage ouvert

*Voir aussi* : Data Lake, Apache Iceberg

### Data Warehouse

Système de stockage et d'analyse de données structurées optimisé pour les requêtes analytiques. Traditionnellement basé sur des architectures propriétaires avec stockage et calcul couplés.

*Voir aussi* : Data Lakehouse, OLAP

### ELT (Extract, Load, Transform)

Pattern d'intégration de données chargeant les données brutes dans la destination avant transformation, exploitant la puissance de calcul du système cible. Prédominant dans les architectures Lakehouse modernes.

*Voir aussi* : ETL, Data Pipeline

### ETL (Extract, Transform, Load)

Pattern d'intégration de données transformant les données dans un système intermédiaire avant chargement dans la destination. Pattern traditionnel des entrepôts de données.

*Voir aussi* : ELT, Data Pipeline

### File Format (Format de fichier)

Spécification définissant comment les données sont sérialisées et stockées dans des fichiers individuels. Iceberg supporte Parquet, ORC et Avro comme formats de fichiers de données.

*Voir aussi* : Table Format, Parquet, ORC, Avro

### File Pruning (Élagage de fichiers)

Optimisation excluant les fichiers de données non pertinents de la lecture basée sur les statistiques stockées dans les manifestes (min/max, null count). Réduit les I/O en évitant la lecture de fichiers entiers.

*Voir aussi* : Partition Pruning, Column Pruning

### Infonuagique (Cloud Computing)

Modèle de fourniture de services informatiques (calcul, stockage, réseau) à la demande via internet. Les architectures Lakehouse modernes s'appuient sur l'infrastructure infonuagique pour la scalabilité et la flexibilité.

*Voir aussi* : Object Storage, Serverless

### Medallion Architecture (Architecture Médaillon)

Pattern d'organisation des données en couches progressives de raffinement : Bronze (données brutes), Silver (données nettoyées et conformées), Gold (données agrégées et optimisées pour la consommation).

*Voir aussi* : Data Lakehouse, ELT

### Object Storage (Stockage objet)

Service de stockage infonuagique gérant les données comme objets avec métadonnées associées. Offre durabilité, scalabilité et coût optimisé pour le stockage de données à grande échelle.

*Exemples* : Amazon S3, Azure Blob Storage, Google Cloud Storage, MinIO

*Voir aussi* : Infonuagique, Data Lake

### OLAP (Online Analytical Processing)

Catégorie de traitement de données optimisée pour les requêtes analytiques complexes sur de grands volumes de données. Les tables Iceberg sont conçues pour les charges de travail OLAP.

*Voir aussi* : Data Warehouse, Analytics

### Optimistic Concurrency (Concurrence optimiste)

Modèle de contrôle de concurrence où les transactions procèdent sans verrouillage, avec détection des conflits au moment du commit. Iceberg utilise la concurrence optimiste pour les écritures parallèles.

*Voir aussi* : ACID, Snapshot Isolation

### Parquet

Format de fichier colonnaire open source optimisé pour les charges de travail analytiques. Format de données par défaut recommandé pour Iceberg, offrant compression efficace et lecture sélective de colonnes.

*Voir aussi* : ORC, Avro, File Format

### Query Engine (Moteur de requête)

Système exécutant des requêtes SQL ou similaires sur les données. Plusieurs moteurs peuvent interroger les mêmes tables Iceberg : Apache Spark, Trino, Dremio, Flink, Presto.

*Voir aussi* : SQL, Apache Spark

### Schema-on-Read

Approche où le schéma est appliqué lors de la lecture plutôt que lors de l'écriture. Permet la flexibilité d'ingestion mais peut causer des problèmes de qualité de données.

*Voir aussi* : Schema-on-Write, Data Lake

### Schema-on-Write

Approche où le schéma est appliqué lors de l'écriture, validant les données avant stockage. Iceberg supporte l'enforcement de schéma à l'écriture tout en permettant l'évolution de schéma.

*Voir aussi* : Schema-on-Read, Schema Evolution

### Table Format (Format de table)

Spécification définissant comment les données sont organisées, gérées et interrogées à un niveau d'abstraction supérieur aux fichiers individuels. Définit les métadonnées, les transactions et l'évolution de schéma.

*Exemples* : Apache Iceberg, Delta Lake, Apache Hudi

*Voir aussi* : File Format, Apache Iceberg

### Write Amplification (Amplification d'écriture)

Phénomène où une modification logique mineure entraîne une réécriture physique disproportionnée. Le mode Copy-on-Write souffre d'amplification d'écriture; Merge-on-Read la minimise.

*Voir aussi* : Copy-on-Write, Merge-on-Read

---

## B.3 Terminologie Streaming et Kafka

### Apache Kafka

Plateforme de streaming d'événements distribuée pour la construction de pipelines de données temps réel et d'applications de streaming. Kafka peut alimenter des tables Iceberg via des connecteurs dédiés.

*Voir aussi* : Kafka Connect, Streaming Lakehouse

### Batch Processing (Traitement par lots)

Mode de traitement de données collectées et traitées en groupes à intervalles définis. Complémentaire au traitement en flux (streaming) dans les architectures modernes.

*Voir aussi* : Stream Processing, Lambda Architecture

### Broker

Serveur Kafka individuel gérant le stockage et la distribution des messages au sein d'un cluster. Les brokers forment l'épine dorsale de l'infrastructure Kafka.

*Voir aussi* : Apache Kafka, Partition (Kafka)

### Consumer

Application ou service lisant des messages depuis des topics Kafka. Les consumers Iceberg écrivent les données consommées dans des tables Iceberg.

*Voir aussi* : Producer, Consumer Group

### Consumer Group (Groupe de consommateurs)

Ensemble de consumers partageant la charge de lecture d'un topic Kafka. Chaque partition est assignée à un seul consumer du groupe, permettant le parallélisme.

*Voir aussi* : Consumer, Partition (Kafka)

### Event Sourcing

Pattern architectural stockant l'état de l'application comme une séquence d'événements immuables. Les événements peuvent être matérialisés dans des tables Iceberg pour l'analyse.

*Voir aussi* : CDC, Streaming Lakehouse

### Exactly-Once Semantics (Sémantiques exactly-once)

Garantie de traitement où chaque message est traité exactement une fois, sans perte ni duplication. Iceberg supporte les sémantiques exactly-once via les transactions ACID.

*Voir aussi* : At-Least-Once, ACID

### Kafka Connect

Framework pour connecter Kafka à des systèmes externes via des connecteurs standardisés. Le connecteur Iceberg sink écrit les données Kafka dans des tables Iceberg.

*Voir aussi* : Apache Kafka, Sink Connector

### Lambda Architecture

Architecture hybride combinant traitement par lots et traitement en flux pour gérer de grands volumes de données. Les tables Iceberg peuvent servir de couche de service unifiée.

*Voir aussi* : Kappa Architecture, Streaming Lakehouse

### Kappa Architecture

Architecture simplifiée utilisant uniquement le traitement en flux pour tous les cas d'usage, éliminant la complexité du traitement par lots séparé.

*Voir aussi* : Lambda Architecture, Stream Processing

### Offset

Position d'un message dans une partition Kafka, utilisée pour le suivi de la progression de lecture. Les offsets permettent la reprise après échec et le retraitement.

*Voir aussi* : Partition (Kafka), Consumer

### Partition (Kafka)

Unité de parallélisme et d'ordonnancement dans un topic Kafka. Chaque partition est une séquence ordonnée de messages stockée sur un broker.

*Voir aussi* : Topic, Broker

### Producer

Application ou service publiant des messages dans des topics Kafka. Les producers sont les sources de données dans l'architecture de streaming.

*Voir aussi* : Consumer, Topic

### Sink Connector

Connecteur Kafka Connect écrivant des données depuis Kafka vers un système externe. Le sink connector Iceberg écrit dans des tables Iceberg avec gestion des transactions.

*Voir aussi* : Kafka Connect, Source Connector

### Source Connector

Connecteur Kafka Connect lisant des données depuis un système externe vers Kafka. Utilisé pour la capture de données depuis bases de données, fichiers ou APIs.

*Voir aussi* : Kafka Connect, Sink Connector

### Stream Processing (Traitement en flux)

Mode de traitement des données en continu à mesure de leur arrivée, permettant des réponses temps réel ou quasi-temps réel. Apache Flink est un moteur de stream processing populaire avec Iceberg.

*Voir aussi* : Batch Processing, Apache Flink

### Streaming Lakehouse

Architecture combinant le streaming temps réel (Kafka) avec le stockage analytique (Iceberg) pour des analyses unifiées sur données fraîches et historiques.

*Voir aussi* : Data Lakehouse, Apache Kafka, Apache Iceberg

### Topic

Canal de publication-souscription dans Kafka où les messages sont organisés. Chaque topic est divisé en partitions pour le parallélisme et la scalabilité.

*Voir aussi* : Partition (Kafka), Producer, Consumer

### Watermark

Mécanisme de gestion du temps événementiel dans le traitement de flux, indiquant la progression du temps dans le flux de données. Utilisé pour gérer les données tardives et déclencher les calculs.

*Voir aussi* : Stream Processing, Event Time

---

## B.4 Acronymes et abréviations

| Acronyme | Signification | Description |
|----------|--------------|-------------|
| ACID | Atomicity, Consistency, Isolation, Durability | Propriétés transactionnelles garantissant l'intégrité des données |
| ADLS | Azure Data Lake Storage | Service de stockage objet Microsoft Azure |
| API | Application Programming Interface | Interface de programmation applicative |
| AWS | Amazon Web Services | Services infonuagiques Amazon |
| BI | Business Intelligence | Intelligence d'affaires, analyse décisionnelle |
| CDC | Change Data Capture | Capture des modifications de données |
| CDO | Chief Data Officer | Directeur des données |
| CoW | Copy-on-Write | Mode d'écriture avec copie |
| CPU | Central Processing Unit | Unité centrale de traitement |
| CSV | Comma-Separated Values | Valeurs séparées par des virgules |
| DDL | Data Definition Language | Langage de définition de données |
| DML | Data Manipulation Language | Langage de manipulation de données |
| DV | Deletion Vector | Vecteur de suppression |
| ELT | Extract, Load, Transform | Pattern d'intégration données |
| EMR | Elastic MapReduce | Service AWS de traitement de données |
| ETL | Extract, Transform, Load | Pattern d'intégration données |
| GCP | Google Cloud Platform | Services infonuagiques Google |
| GCS | Google Cloud Storage | Service de stockage objet Google |
| HDFS | Hadoop Distributed File System | Système de fichiers distribué Hadoop |
| HMS | Hive Metastore | Métastore Apache Hive |
| HTTP | Hypertext Transfer Protocol | Protocole de transfert hypertexte |
| I/O | Input/Output | Entrées/Sorties |
| JDBC | Java Database Connectivity | Connectivité base de données Java |
| JSON | JavaScript Object Notation | Notation d'objets JavaScript |
| MB | Megabyte | Mégaoctet |
| ML | Machine Learning | Apprentissage automatique |
| MoR | Merge-on-Read | Mode d'écriture avec fusion à la lecture |
| NDV | Number of Distinct Values | Nombre de valeurs distinctes |
| NaN | Not a Number | Non un nombre (valeur spéciale flottante) |
| OAuth | Open Authorization | Protocole d'autorisation ouvert |
| OLAP | Online Analytical Processing | Traitement analytique en ligne |
| OLTP | Online Transaction Processing | Traitement transactionnel en ligne |
| ORC | Optimized Row Columnar | Format de fichier colonnaire optimisé |
| PB | Petabyte | Pétaoctet |
| REST | Representational State Transfer | Transfert d'état représentationnel |
| RGPD | Règlement Général sur la Protection des Données | Réglementation européenne sur la protection des données |
| S3 | Simple Storage Service | Service de stockage simple (AWS) |
| SDK | Software Development Kit | Kit de développement logiciel |
| SQL | Structured Query Language | Langage de requête structuré |
| SSO | Single Sign-On | Authentification unique |
| TB | Terabyte | Téraoctet |
| TLS | Transport Layer Security | Sécurité de la couche de transport |
| URI | Uniform Resource Identifier | Identifiant uniforme de ressource |
| URL | Uniform Resource Locator | Localisateur uniforme de ressource |
| UTC | Coordinated Universal Time | Temps universel coordonné |
| UUID | Universally Unique Identifier | Identifiant unique universel |
| V1 | Version 1 | Première version du format Iceberg |
| V2 | Version 2 | Deuxième version du format Iceberg |
| V3 | Version 3 | Troisième version du format Iceberg |
| VM | Virtual Machine | Machine virtuelle |
| XML | Extensible Markup Language | Langage de balisage extensible |
| YAML | YAML Ain't Markup Language | Format de sérialisation de données |

---

## Index des termes par thème

### Architecture et stockage
- Data Lake, Data Lakehouse, Data Warehouse
- Object Storage, HDFS, Infonuagique
- Medallion Architecture, Table Format, File Format

### Apache Iceberg - Core
- Catalog, Metadata File, Manifest File, Manifest List
- Snapshot, Time Travel, Rollback
- Schema Evolution, Partition Evolution, Hidden Partitioning

### Apache Iceberg - Opérations
- Copy-on-Write, Merge-on-Read
- Delete File, Deletion Vector, Position Delete, Equality Delete
- Compaction, Snapshot Expiration

### Apache Iceberg - Avancé
- REST Catalog, Puffin, Row Lineage
- Statistics, Sequence Number, Variant

### Performance et optimisation
- Partition Pruning, File Pruning, Column Pruning
- Write Amplification, ACID, Optimistic Concurrency

### Streaming et Kafka
- Apache Kafka, Topic, Partition (Kafka)
- Producer, Consumer, Consumer Group
- Kafka Connect, Sink Connector, Source Connector
- Stream Processing, Streaming Lakehouse

### Intégration de données
- CDC, ETL, ELT
- Batch Processing, Stream Processing
- Event Sourcing, Exactly-Once Semantics

---

## Références croisées vers les chapitres

| Terme | Chapitres de référence |
|-------|------------------------|
| Apache Iceberg | Chapitres 1, 2, 3 |
| Catalog | Chapitres 5, 6 |
| Compaction | Chapitre 10 |
| Copy-on-Write / Merge-on-Read | Chapitres 3, 8 |
| Deletion Vector | Chapitres 3, 10 |
| Hidden Partitioning | Chapitres 2, 4 |
| Manifest / Metadata | Chapitre 2 |
| REST Catalog | Chapitres 5, 6 |
| Schema Evolution | Chapitres 2, 4 |
| Snapshot / Time Travel | Chapitres 2, 3 |
| Statistics / Puffin | Chapitres 3, 10 |
| Streaming Lakehouse | Chapitre 12 |

---

## Résumé

Ce glossaire fournit les définitions normalisées de la terminologie utilisée dans le Volume IV et l'écosystème Lakehouse plus largement. Les termes sont organisés en quatre catégories principales :

**Terminologie Apache Iceberg.** Les concepts fondamentaux du format de table Iceberg, incluant la structure des métadonnées (catalog, snapshot, manifest), les mécanismes de mise à jour (Copy-on-Write, Merge-on-Read, delete files), et les fonctionnalités avancées (schema evolution, hidden partitioning, time travel).

**Terminologie Lakehouse et Data Engineering.** Les concepts architecturaux et les patterns de l'ingénierie des données moderne, des formats de fichiers aux architectures de référence en passant par les optimisations de performance.

**Terminologie Streaming et Kafka.** Les concepts du traitement de données en flux et de l'écosystème Apache Kafka, essentiels pour comprendre le Streaming Lakehouse présenté au Chapitre 12.

**Acronymes et abréviations.** Référence rapide des acronymes techniques utilisés dans l'ouvrage.

Ce glossaire doit être consulté comme référence complémentaire lors de la lecture des chapitres techniques, et comme outil de normalisation pour les équipes adoptant Apache Iceberg.

---

*Dernière mise à jour : Janvier 2026*
