
# Chapitre IV.6 - Architecture de la Couche d'Ingestion

## Introduction

La couche d'ingestion constitue l'interface vitale entre vos systèmes sources et votre Data Lakehouse. Elle détermine la fraîcheur des données disponibles pour l'analyse, la fiabilité des pipelines de données et, ultimement, la capacité de votre organisation à prendre des décisions éclairées en temps opportun. Une architecture d'ingestion bien conçue transforme le chaos des données brutes provenant de dizaines — voire de centaines — de systèmes hétérogènes en un flux ordonné alimentant vos tables Apache Iceberg.

Dans l'écosystème du Lakehouse moderne, l'ingestion ne se limite plus au traditionnel chargement batch nocturne. Les organisations exigent désormais une palette complète de latences : du temps réel pour les tableaux de bord opérationnels, au micro-batch pour les analyses tactiques, jusqu'au batch classique pour les consolidations historiques. Apache Iceberg, grâce à ses capacités transactionnelles ACID et son support natif des écritures concurrentes, permet d'implémenter cette diversité de patterns au sein d'une architecture unifiée.

Ce chapitre vous guide dans la conception et l'implémentation d'une couche d'ingestion robuste pour votre Lakehouse Iceberg. Nous examinerons les patterns fondamentaux d'ingestion, comparerons les technologies disponibles — Apache Spark, Apache Flink, Kafka Connect et leurs alternatives —, détaillerons les stratégies de chargement adaptées à chaque cas d'usage et illustrerons nos recommandations par des architectures de référence et des études de cas concrètes.

L'intégration avec Apache Kafka, documentée en profondeur dans le Volume III de cette monographie, occupe une place centrale dans notre exploration. Le Streaming Lakehouse — fusion du traitement événementiel temps réel et de l'analytique sur données historiques — représente l'aboutissement architectural vers lequel convergent les organisations les plus matures en matière de données.

---

## Patterns Fondamentaux d'Ingestion

### Taxonomie des Modes d'Ingestion

L'ingestion de données dans un Lakehouse Iceberg s'articule autour de trois patterns fondamentaux, chacun répondant à des exigences distinctes de latence, de volume et de complexité opérationnelle.

**Ingestion Batch** : Le pattern historique demeure pertinent pour les scénarios où la latence de plusieurs heures est acceptable. Les chargements batch traitent des volumes massifs de données en une seule opération, optimisant l'utilisation des ressources et simplifiant la gestion des erreurs. Ce mode convient aux extractions quotidiennes de systèmes ERP, aux consolidations mensuelles et aux migrations de données historiques.

**Ingestion Micro-Batch** : Compromis entre latence et efficacité, le micro-batch traite les données en petits lots à intervalles réguliers — typiquement toutes les 5 à 15 minutes. Ce pattern offre une fraîcheur suffisante pour la plupart des cas d'usage analytiques tout en préservant les optimisations batch comme le dimensionnement des fichiers et la compaction efficace.

**Ingestion Streaming** : Le traitement événement par événement — ou en micro-lots de quelques secondes — répond aux exigences de temps réel. Les tableaux de bord opérationnels, la détection de fraude et les systèmes de recommandation en temps réel nécessitent cette latence minimale. Iceberg supporte désormais nativement ce mode grâce à ses commits incrémentaux et sa gestion des conflits.

| Pattern     | Latence typique | Volume par opération | Complexité | Cas d'usage              |
| ----------- | --------------- | --------------------- | ----------- | ------------------------ |
| Batch       | Heures          | Go à To              | Faible      | ETL nocturne, migrations |
| Micro-batch | Minutes         | Mo à Go              | Moyenne     | Rapports tactiques       |
| Streaming   | Secondes        | Ko à Mo              | Élevée    | Temps réel, alertes     |

### Évolution vers le Streaming Lakehouse

L'architecture Streaming Lakehouse, introduite conceptuellement dans le Volume III avec Apache Kafka, unifie les traitements batch et streaming au sein d'une plateforme cohérente. Apache Iceberg joue un rôle central dans cette unification en fournissant :

* **Commits atomiques** permettant des écritures concurrentes depuis plusieurs pipelines
* **Snapshots immuables** garantissant des lectures cohérentes pendant les écritures
* **Time Travel** offrant un accès aux états historiques sans duplication
* **Schema Evolution** accommodant les changements de structure sans interruption

Cette convergence élimine la traditionnelle « architecture Lambda » avec ses chemins batch et streaming séparés, réduisant la complexité opérationnelle et les risques d'incohérence entre couches.

```
Architecture Lambda traditionnelle:
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Sources   │────▶│ Couche Batch│────▶│   Vue Batch │
└─────────────┘     └─────────────┘     └──────┬──────┘
       │                                        │
       │            ┌─────────────┐     ┌──────▼──────┐
       └───────────▶│Couche Stream│────▶│ Vue Fusionnée│
                    └─────────────┘     └─────────────┘

Architecture Streaming Lakehouse:
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Sources   │────▶│    Kafka    │────▶│   Iceberg   │
└─────────────┘     └─────────────┘     │  (unifié)   │
                                        └─────────────┘
```

### Sémantiques de Livraison

La fiabilité de l'ingestion repose sur la sémantique de livraison garantie par votre architecture. Trois niveaux de garantie existent :

**At-most-once** (au plus une fois) : Les données peuvent être perdues mais ne seront jamais dupliquées. Acceptable uniquement pour les métriques non critiques où une perte occasionnelle est tolérable.

**At-least-once** (au moins une fois) : Les données ne seront jamais perdues mais peuvent être dupliquées. Nécessite une logique de déduplication en aval ou des opérations idempotentes.

**Exactly-once** (exactement une fois) : Chaque enregistrement est traité exactement une fois, sans perte ni duplication. Cette sémantique exige une coordination entre le système source, le pipeline de traitement et le système cible.

Apache Iceberg, combiné à Kafka et un moteur de traitement adéquat (Flink, Spark Structured Streaming), permet d'atteindre la sémantique exactly-once grâce à :

* La gestion transactionnelle des commits Iceberg
* Les offsets Kafka comme marqueurs de progression
* Le checkpointing des moteurs de traitement

---

## Architecture de Référence

### Vue d'Ensemble

Une architecture d'ingestion complète pour un Lakehouse Iceberg comprend plusieurs composantes orchestrées pour garantir fiabilité, performance et observabilité.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        SOURCES DE DONNÉES                               │
├───────────┬───────────┬───────────┬───────────┬───────────┬────────────┤
│   SGBD    │    API    │  Fichiers │    IoT    │    SaaS   │   Logs     │
│ (CDC)     │  (REST)   │  (S3/SFTP)│ (MQTT)    │ (Webhook) │ (Syslog)   │
└─────┬─────┴─────┬─────┴─────┬─────┴─────┬─────┴─────┬─────┴──────┬─────┘
      │           │           │           │           │            │
      ▼           ▼           ▼           ▼           ▼            ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      COUCHE DE COLLECTE                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │   Debezium  │  │   Airbyte   │  │    NiFi     │  │  Connecteurs│    │
│  │   (CDC)     │  │  (ELT)      │  │  (Routage)  │  │   Kafka     │    │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘    │
└─────────┼────────────────┼────────────────┼────────────────┼───────────┘
          │                │                │                │
          ▼                ▼                ▼                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      APACHE KAFKA (Bus Événementiel)                   │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Topics: cdc.*, api.*, files.*, iot.*, saas.*, logs.*           │   │
│  │  Schema Registry: Avro/Protobuf schemas                          │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    COUCHE DE TRAITEMENT                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐         │
│  │  Apache Flink   │  │  Spark Streaming│  │   Kafka Connect │         │
│  │  (Temps réel)   │  │  (Micro-batch)  │  │   (Direct sink) │         │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘         │
└───────────┼────────────────────┼────────────────────┼───────────────────┘
            │                    │                    │
            ▼                    ▼                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      LAKEHOUSE ICEBERG                                  │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Bronze: raw.*, staging.*                                        │   │
│  │  Silver: cleaned.*, enriched.*                                   │   │
│  │  Gold: analytics.*, marts.*                                      │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│  Stockage: S3 / ADLS Gen2 / MinIO                                      │
│  Catalogue: REST Catalog / AWS Glue / Hive Metastore                   │
└─────────────────────────────────────────────────────────────────────────┘
```

### Composantes Clés

**Couche de Collecte** : Responsable de l'extraction des données depuis les systèmes sources. Cette couche doit gérer la diversité des protocoles (JDBC, REST, SFTP, MQTT), la transformation initiale des formats et la gestion des erreurs de connexion.

**Bus Événementiel (Kafka)** : Point central de découplage entre les producteurs et les consommateurs. Kafka assure la persistance temporaire, le replay en cas d'erreur et la distribution vers de multiples consommateurs. Le Schema Registry garantit la compatibilité des schémas.

**Couche de Traitement** : Transforme et charge les données dans Iceberg. Le choix du moteur dépend des exigences de latence : Flink pour le temps réel, Spark Structured Streaming pour le micro-batch, Kafka Connect pour les chargements directs simples.

**Lakehouse Iceberg** : Destination finale structurée en couches médaillon (Bronze/Silver/Gold) pour organiser les données selon leur niveau de raffinement.

### Dimensionnement Initial

Le dimensionnement de l'architecture d'ingestion dépend de plusieurs facteurs interdépendants :

| Facteur            | Impact                     | Recommandation initiale                     |
| ------------------ | -------------------------- | ------------------------------------------- |
| Volume quotidien   | Capacité Kafka, stockage  | 3× le volume quotidien en rétention Kafka |
| Pic de débit      | Partitions Kafka, workers  | Prévoir 2× le débit moyen                |
| Nombre de sources  | Connecteurs, parallélisme | 1 topic par source majeure                  |
| Latence cible      | Choix du moteur            | Flink < 1s, Spark 1-15min                   |
| Fenêtre de replay | Rétention Kafka           | 7 jours minimum                             |

---

## Ingestion Batch avec Apache Spark

### Configuration Spark-Iceberg

Apache Spark demeure le moteur de référence pour l'ingestion batch dans Iceberg. Sa maturité, son écosystème riche et son intégration native avec Iceberg en font le choix par défaut pour les chargements volumineux.

**Configuration de base** :

```python
from pyspark.sql import SparkSession

spark = SparkSession.builder \
    .appName("IngestionBatchIceberg") \
    .config("spark.sql.extensions", "org.apache.iceberg.spark.extensions.IcebergSparkSessionExtensions") \
    .config("spark.sql.catalog.lakehouse", "org.apache.iceberg.spark.SparkCatalog") \
    .config("spark.sql.catalog.lakehouse.type", "rest") \
    .config("spark.sql.catalog.lakehouse.uri", "http://iceberg-rest-catalog:8181") \
    .config("spark.sql.catalog.lakehouse.warehouse", "s3://entreprise-lakehouse/warehouse") \
    .config("spark.sql.catalog.lakehouse.io-impl", "org.apache.iceberg.aws.s3.S3FileIO") \
    .config("spark.hadoop.fs.s3a.endpoint", "s3.ca-central-1.amazonaws.com") \
    .config("spark.hadoop.fs.s3a.aws.credentials.provider", 
            "com.amazonaws.auth.DefaultAWSCredentialsProviderChain") \
    .getOrCreate()
```

**Paramètres de performance critiques** :

```python
# Optimisations pour l'écriture batch
spark.conf.set("spark.sql.shuffle.partitions", "200")
spark.conf.set("spark.sql.adaptive.enabled", "true")
spark.conf.set("spark.sql.adaptive.coalescePartitions.enabled", "true")

# Configuration Iceberg spécifique
spark.conf.set("spark.sql.catalog.lakehouse.write.target-file-size-bytes", "536870912")  # 512 Mo
spark.conf.set("spark.sql.catalog.lakehouse.write.distribution-mode", "hash")
```

### Patterns d'Écriture Batch

**Append simple** : Le pattern le plus performant pour les données nouvelles sans mise à jour.

```python
# Lecture depuis la source
df_nouvelles_transactions = spark.read \
    .format("jdbc") \
    .option("url", "jdbc:postgresql://source-db:5432/production") \
    .option("dbtable", "(SELECT * FROM transactions WHERE date_creation >= '2024-01-15') AS t") \
    .option("user", "reader") \
    .option("password", "${DB_PASSWORD}") \
    .option("fetchsize", "10000") \
    .option("numPartitions", "16") \
    .option("partitionColumn", "id") \
    .option("lowerBound", "1") \
    .option("upperBound", "10000000") \
    .load()

# Écriture en mode append
df_nouvelles_transactions.writeTo("lakehouse.bronze.transactions") \
    .option("write.distribution-mode", "hash") \
    .option("write.target-file-size-bytes", "536870912") \
    .append()
```

**Chargement initial complet** : Pour les migrations ou les premières charges.

```python
# Création de la table avec partitionnement
spark.sql("""
    CREATE TABLE IF NOT EXISTS lakehouse.bronze.clients (
        client_id BIGINT,
        nom STRING,
        courriel STRING,
        date_inscription DATE,
        province STRING,
        segment STRING
    )
    USING iceberg
    PARTITIONED BY (province, months(date_inscription))
    TBLPROPERTIES (
        'write.target-file-size-bytes' = '536870912',
        'write.distribution-mode' = 'hash',
        'format-version' = '2'
    )
""")

# Chargement avec repartitionnement optimal
df_clients = spark.read.format("jdbc") \
    .option("url", "jdbc:oracle:thin:@legacy-crm:1521/PROD") \
    .option("dbtable", "CLIENTS") \
    .load()

df_clients \
    .repartition(100, "province") \
    .sortWithinPartitions("date_inscription") \
    .writeTo("lakehouse.bronze.clients") \
    .createOrReplace()
```

### Optimisation des Extractions JDBC

L'extraction depuis les bases de données relationnelles constitue souvent le goulot d'étranglement de l'ingestion batch. Plusieurs stratégies permettent d'optimiser ce processus.

**Parallélisation de l'extraction** :

```python
# Extraction parallèle avec partitionnement numérique
df = spark.read.format("jdbc") \
    .option("url", jdbc_url) \
    .option("dbtable", "commandes") \
    .option("numPartitions", 32) \
    .option("partitionColumn", "commande_id") \
    .option("lowerBound", 1) \
    .option("upperBound", max_id) \
    .option("fetchsize", 10000) \
    .load()

# Alternative: partitionnement par prédicat pour colonnes non numériques
predicates = [
    "province = 'QC'",
    "province = 'ON'",
    "province = 'BC'",
    "province = 'AB'",
    "province NOT IN ('QC', 'ON', 'BC', 'AB')"
]

df = spark.read.jdbc(
    url=jdbc_url,
    table="commandes",
    predicates=predicates,
    properties={"fetchsize": "10000"}
)
```

**Extraction incrémentale avec watermark** :

```python
from datetime import datetime, timedelta

# Récupération du dernier watermark
dernier_watermark = spark.sql("""
    SELECT MAX(date_modification) as watermark 
    FROM lakehouse.bronze.commandes
""").collect()[0]["watermark"]

# Extraction des nouvelles données uniquement
df_incremental = spark.read.format("jdbc") \
    .option("url", jdbc_url) \
    .option("dbtable", f"""
        (SELECT * FROM commandes 
         WHERE date_modification > '{dernier_watermark}'
         AND date_modification <= '{datetime.now()}') AS incr
    """) \
    .load()

# Écriture avec mise à jour du watermark
df_incremental.writeTo("lakehouse.bronze.commandes").append()
```

> **Performance**
>
> L'extraction JDBC parallèle peut améliorer les temps de chargement de 5× à 20× selon la configuration. Cependant, attention à ne pas surcharger le système source : limitez le nombre de connexions parallèles et planifiez les extractions durant les périodes de faible activité.

---

## Ingestion Streaming avec Apache Kafka

### Architecture Kafka-Iceberg

L'intégration Kafka-Iceberg forme le cœur du Streaming Lakehouse. Deux approches principales permettent de connecter ces technologies :

**Approche 1 : Kafka Connect avec Iceberg Sink**

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Sources   │────▶│    Kafka    │────▶│Kafka Connect│────▶ Iceberg
│   (CDC)     │     │   Topics    │     │ Iceberg Sink│
└─────────────┘     └─────────────┘     └─────────────┘
```

**Approche 2 : Moteur de Traitement Streaming**

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Sources   │────▶│    Kafka    │────▶│ Flink/Spark │────▶ Iceberg
│   (CDC)     │     │   Topics    │     │  Streaming  │
└─────────────┘     └─────────────┘     └─────────────┘
```

Le choix entre ces approches dépend de la complexité des transformations requises. Kafka Connect convient aux chargements directs avec transformations minimales, tandis qu'un moteur de traitement s'impose pour les agrégations, jointures ou logiques métier complexes.

### Configuration Kafka Connect Iceberg Sink

Le connecteur Iceberg Sink pour Kafka Connect, développé par Tabular (maintenant partie de Databricks), offre une solution clé en main pour l'ingestion streaming.

```json
{
  "name": "iceberg-sink-transactions",
  "config": {
    "connector.class": "io.tabular.iceberg.connect.IcebergSinkConnector",
    "tasks.max": "4",
    "topics": "cdc.production.transactions",
  
    "iceberg.tables": "lakehouse.bronze.transactions",
    "iceberg.catalog.type": "rest",
    "iceberg.catalog.uri": "http://iceberg-rest-catalog:8181",
    "iceberg.catalog.warehouse": "s3://entreprise-lakehouse/warehouse",
    "iceberg.catalog.s3.endpoint": "s3.ca-central-1.amazonaws.com",
  
    "iceberg.table.auto-create": "true",
    "iceberg.table.evolve-schema-enabled": "true",
  
    "iceberg.control.commit.interval-ms": "60000",
    "iceberg.control.commit.timeout-ms": "300000",
  
    "key.converter": "io.confluent.connect.avro.AvroConverter",
    "key.converter.schema.registry.url": "http://schema-registry:8081",
    "value.converter": "io.confluent.connect.avro.AvroConverter",
    "value.converter.schema.registry.url": "http://schema-registry:8081",
  
    "transforms": "unwrap",
    "transforms.unwrap.type": "io.debezium.transforms.ExtractNewRecordState",
    "transforms.unwrap.drop.tombstones": "false",
    "transforms.unwrap.delete.handling.mode": "rewrite"
  }
}
```

**Paramètres critiques** :

| Paramètre                | Description                    | Recommandation              |
| ------------------------- | ------------------------------ | --------------------------- |
| `tasks.max`             | Parallélisme du connecteur    | 1 par partition Kafka (max) |
| `commit.interval-ms`    | Fréquence des commits Iceberg | 60-300s selon latence cible |
| `commit.timeout-ms`     | Timeout pour commit            | 3-5× l'intervalle          |
| `evolve-schema-enabled` | Évolution automatique         | `true`pour CDC            |

### Apache Flink pour le Streaming Temps Réel

Apache Flink excelle pour les scénarios nécessitant une latence sub-seconde ou des transformations complexes en streaming. Son intégration native avec Iceberg via le connecteur Flink-Iceberg offre des garanties exactly-once robustes.

**Configuration Flink-Iceberg** :

```java
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.table.api.bridge.java.StreamTableEnvironment;

StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
env.enableCheckpointing(60000); // Checkpoint toutes les 60 secondes

StreamTableEnvironment tableEnv = StreamTableEnvironment.create(env);

// Configuration du catalogue Iceberg
tableEnv.executeSql("""
    CREATE CATALOG lakehouse WITH (
        'type' = 'iceberg',
        'catalog-type' = 'rest',
        'uri' = 'http://iceberg-rest-catalog:8181',
        'warehouse' = 's3://entreprise-lakehouse/warehouse',
        'io-impl' = 'org.apache.iceberg.aws.s3.S3FileIO'
    )
""");

// Définition de la source Kafka
tableEnv.executeSql("""
    CREATE TABLE kafka_transactions (
        transaction_id BIGINT,
        client_id BIGINT,
        montant DECIMAL(10,2),
        devise STRING,
        timestamp_evenement TIMESTAMP(3),
        WATERMARK FOR timestamp_evenement AS timestamp_evenement - INTERVAL '5' SECOND
    ) WITH (
        'connector' = 'kafka',
        'topic' = 'cdc.production.transactions',
        'properties.bootstrap.servers' = 'kafka:9092',
        'properties.group.id' = 'flink-iceberg-ingestion',
        'scan.startup.mode' = 'earliest-offset',
        'format' = 'avro-confluent',
        'avro-confluent.url' = 'http://schema-registry:8081'
    )
""");

// Ingestion continue vers Iceberg
tableEnv.executeSql("""
    INSERT INTO lakehouse.bronze.transactions
    SELECT 
        transaction_id,
        client_id,
        montant,
        devise,
        timestamp_evenement,
        CURRENT_TIMESTAMP as timestamp_ingestion
    FROM kafka_transactions
""");
```

**Agrégations en temps réel** :

```sql
-- Table d'agrégation par fenêtre temporelle
CREATE TABLE lakehouse.silver.metriques_temps_reel (
    fenetre_debut TIMESTAMP,
    fenetre_fin TIMESTAMP,
    total_transactions BIGINT,
    montant_total DECIMAL(15,2),
    montant_moyen DECIMAL(10,2)
) WITH (
    'format-version' = '2',
    'write.upsert.enabled' = 'true'
);

-- Pipeline d'agrégation
INSERT INTO lakehouse.silver.metriques_temps_reel
SELECT 
    window_start as fenetre_debut,
    window_end as fenetre_fin,
    COUNT(*) as total_transactions,
    SUM(montant) as montant_total,
    AVG(montant) as montant_moyen
FROM TABLE(
    TUMBLE(TABLE kafka_transactions, DESCRIPTOR(timestamp_evenement), INTERVAL '1' MINUTE)
)
GROUP BY window_start, window_end;
```

### Spark Structured Streaming

Pour les organisations déjà investies dans l'écosystème Spark, Structured Streaming offre une alternative mature à Flink avec une courbe d'apprentissage réduite.

```python
from pyspark.sql import SparkSession
from pyspark.sql.functions import from_json, col, current_timestamp
from pyspark.sql.types import StructType, StructField, LongType, DecimalType, StringType, TimestampType

# Schéma des événements
schema_transaction = StructType([
    StructField("transaction_id", LongType()),
    StructField("client_id", LongType()),
    StructField("montant", DecimalType(10, 2)),
    StructField("devise", StringType()),
    StructField("timestamp_evenement", TimestampType())
])

# Lecture streaming depuis Kafka
df_stream = spark.readStream \
    .format("kafka") \
    .option("kafka.bootstrap.servers", "kafka:9092") \
    .option("subscribe", "cdc.production.transactions") \
    .option("startingOffsets", "earliest") \
    .option("maxOffsetsPerTrigger", 100000) \
    .load()

# Transformation
df_transactions = df_stream \
    .select(from_json(col("value").cast("string"), schema_transaction).alias("data")) \
    .select("data.*") \
    .withColumn("timestamp_ingestion", current_timestamp())

# Écriture vers Iceberg
query = df_transactions.writeStream \
    .format("iceberg") \
    .outputMode("append") \
    .option("path", "lakehouse.bronze.transactions") \
    .option("checkpointLocation", "s3://entreprise-lakehouse/checkpoints/transactions") \
    .trigger(processingTime="1 minute") \
    .start()
```

> **Performance**
>
> Spark Structured Streaming avec Iceberg atteint typiquement des latences de 1 à 5 minutes selon la configuration du trigger. Pour des latences sub-minute, privilégiez Apache Flink ou Kafka Connect avec des intervalles de commit courts.

---

## Change Data Capture (CDC)

### Fondamentaux du CDC

Le Change Data Capture représente la technique privilégiée pour synchroniser les données des systèmes transactionnels vers le Lakehouse. Plutôt que d'extraire périodiquement l'intégralité des tables, le CDC capture uniquement les modifications — insertions, mises à jour, suppressions — réduisant drastiquement le volume de données transférées et la charge sur les systèmes sources.

**Types de CDC** :

| Type            | Mécanisme                           | Latence  | Impact source |
| --------------- | ------------------------------------ | -------- | ------------- |
| Log-based       | Lecture des journaux de transactions | Secondes | Minimal       |
| Trigger-based   | Déclencheurs base de données       | Secondes | Modéré      |
| Query-based     | Requêtes périodiques sur timestamp | Minutes  | Variable      |
| Timestamp-based | Colonne de modification              | Minutes  | Minimal       |

Le CDC basé sur les journaux (log-based) offre le meilleur compromis latence/impact et constitue la recommandation par défaut pour les bases de données le supportant.

### Debezium : La Référence Open Source

Debezium s'est imposé comme la solution CDC de référence dans l'écosystème open source. Fonctionnant comme connecteur Kafka Connect, il capture les modifications depuis les journaux de transaction des principales bases de données.

**Architecture Debezium-Kafka-Iceberg** :

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  PostgreSQL │────▶│  Debezium   │────▶│    Kafka    │────▶│   Iceberg   │
│  (WAL)      │     │  Connector  │     │   Topics    │     │   Tables    │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
```

**Configuration Debezium pour PostgreSQL** :

```json
{
  "name": "postgres-cdc-source",
  "config": {
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "tasks.max": "1",
  
    "database.hostname": "postgres-production.internal",
    "database.port": "5432",
    "database.user": "debezium",
    "database.password": "${POSTGRES_PASSWORD}",
    "database.dbname": "production",
    "database.server.name": "prod",
  
    "plugin.name": "pgoutput",
    "publication.name": "debezium_publication",
    "slot.name": "debezium_slot",
  
    "table.include.list": "public.clients,public.commandes,public.produits",
  
    "key.converter": "io.confluent.connect.avro.AvroConverter",
    "key.converter.schema.registry.url": "http://schema-registry:8081",
    "value.converter": "io.confluent.connect.avro.AvroConverter",
    "value.converter.schema.registry.url": "http://schema-registry:8081",
  
    "transforms": "route",
    "transforms.route.type": "org.apache.kafka.connect.transforms.RegexRouter",
    "transforms.route.regex": "prod\\.public\\.(.*)",
    "transforms.route.replacement": "cdc.production.$1",
  
    "heartbeat.interval.ms": "10000",
    "snapshot.mode": "initial"
  }
}
```

### Traitement des Événements CDC dans Iceberg

Les événements CDC contiennent des métadonnées indiquant le type d'opération (c=create, u=update, d=delete) et l'état avant/après de l'enregistrement. Plusieurs stratégies permettent de matérialiser ces événements dans Iceberg.

**Stratégie 1 : Append-only avec historisation**

Conserve tous les événements bruts, permettant une reconstruction complète de l'historique.

```python
# Lecture des événements CDC
df_cdc = spark.readStream \
    .format("kafka") \
    .option("subscribe", "cdc.production.clients") \
    .load()

# Extraction des champs Debezium
df_parsed = df_cdc.select(
    from_json(col("value").cast("string"), schema_debezium).alias("data")
).select(
    col("data.after.*"),
    col("data.op").alias("operation"),
    col("data.ts_ms").alias("timestamp_cdc"),
    current_timestamp().alias("timestamp_ingestion")
)

# Écriture append-only
df_parsed.writeStream \
    .format("iceberg") \
    .outputMode("append") \
    .option("path", "lakehouse.bronze.clients_cdc") \
    .start()
```

**Stratégie 2 : Upsert avec état courant**

Maintient une vue à jour de l'état courant via les opérations MERGE d'Iceberg.

```python
from pyspark.sql.functions import window, max as spark_max

# Micro-batch avec déduplication
def process_batch(batch_df, batch_id):
    # Garder uniquement le dernier état par clé
    latest = batch_df \
        .groupBy("client_id") \
        .agg(spark_max("timestamp_cdc").alias("max_ts")) \
        .join(batch_df, ["client_id"]) \
        .where(col("timestamp_cdc") == col("max_ts"))
  
    # MERGE INTO pour upsert
    latest.createOrReplaceTempView("updates")
  
    spark.sql("""
        MERGE INTO lakehouse.silver.clients t
        USING updates s
        ON t.client_id = s.client_id
        WHEN MATCHED AND s.operation = 'd' THEN DELETE
        WHEN MATCHED THEN UPDATE SET *
        WHEN NOT MATCHED AND s.operation != 'd' THEN INSERT *
    """)

df_cdc.writeStream \
    .foreachBatch(process_batch) \
    .option("checkpointLocation", checkpoint_path) \
    .trigger(processingTime="5 minutes") \
    .start()
```

> **Migration**
>
> *De* : Réplication batch quotidienne avec comparaison de tables
>
> *Vers* : CDC temps réel avec Debezium-Kafka-Iceberg
>
> *Stratégie* : Déploiement progressif table par table, en commençant par les tables les moins critiques. Validation par comparaison batch/CDC pendant 2 semaines avant basculement complet.

### Gestion des Suppressions

Les suppressions méritent une attention particulière dans les architectures Lakehouse. Trois approches coexistent :

**Suppression physique (hard delete)** : L'enregistrement est réellement supprimé de la table Iceberg. Simple mais irréversible et potentiellement problématique pour l'audit.

```sql
MERGE INTO lakehouse.silver.clients t
USING deletes s
ON t.client_id = s.client_id
WHEN MATCHED THEN DELETE
```

**Suppression logique (soft delete)** : Un indicateur marque l'enregistrement comme supprimé sans le retirer physiquement.

```sql
MERGE INTO lakehouse.silver.clients t
USING deletes s
ON t.client_id = s.client_id
WHEN MATCHED THEN UPDATE SET 
    est_supprime = true,
    date_suppression = current_timestamp()
```

**Historisation complète** : Conserve toutes les versions avec dates d'effet (SCD Type 2).

```sql
-- Fermeture de la version précédente
UPDATE lakehouse.silver.clients_historique
SET date_fin_validite = current_timestamp(), est_courant = false
WHERE client_id IN (SELECT client_id FROM deletes)
  AND est_courant = true;

-- Insertion de la version supprimée
INSERT INTO lakehouse.silver.clients_historique
SELECT *, current_timestamp() as date_debut_validite, 
       null as date_fin_validite, true as est_courant, 'SUPPRIME' as statut
FROM deletes;
```

---

## Ingestion de Fichiers

### Patterns d'Ingestion de Fichiers

L'ingestion de fichiers demeure un pattern incontournable pour les échanges avec partenaires externes, les exports de systèmes legacy et les données semi-structurées. Plusieurs mécanismes permettent d'automatiser ce processus.

**Architecture d'ingestion de fichiers** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         SOURCES DE FICHIERS                             │
├───────────────┬───────────────┬───────────────┬───────────────┬────────┤
│   SFTP/FTP    │   S3 Events   │   Azure Blob  │   Partages    │  API   │
│   Partenaires │   Notifications│   Events     │   Réseau      │ Upload │
└───────┬───────┴───────┬───────┴───────┬───────┴───────┬───────┴────┬───┘
        │               │               │               │            │
        ▼               ▼               ▼               ▼            ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    ORCHESTRATION / DÉTECTION                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │   Apache    │  │   AWS       │  │   Azure     │  │   Airflow   │    │
│  │    NiFi     │  │   Lambda    │  │  Functions  │  │   Sensors   │    │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘    │
└─────────┼────────────────┼────────────────┼────────────────┼───────────┘
          │                │                │                │
          ▼                ▼                ▼                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    ZONE D'ATTERRISSAGE (Landing Zone)                   │
│                    s3://lakehouse/landing/                              │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    TRAITEMENT ET VALIDATION                             │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Spark Job: validation schéma, déduplication, transformation    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         LAKEHOUSE ICEBERG                               │
│                    lakehouse.bronze.fichiers_*                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Détection et Déclenchement

**S3 Event Notifications avec Lambda** :

```python
# Lambda function déclenchée par S3 Event
import boto3
import json

def lambda_handler(event, context):
    # Extraction des informations du fichier
    bucket = event['Records'][0]['s3']['bucket']['name']
    key = event['Records'][0]['s3']['object']['key']
  
    # Publication vers Kafka pour traitement
    kafka_client = boto3.client('kafka')
  
    message = {
        'source': 's3',
        'bucket': bucket,
        'key': key,
        'timestamp': event['Records'][0]['eventTime'],
        'size': event['Records'][0]['s3']['object']['size']
    }
  
    # Déclenchement du pipeline d'ingestion
    # Via Step Functions, Airflow, ou publication Kafka
    trigger_ingestion_pipeline(message)
  
    return {'statusCode': 200}
```

**Apache Airflow avec File Sensor** :

```python
from airflow import DAG
from airflow.providers.amazon.aws.sensors.s3 import S3KeySensor
from airflow.providers.apache.spark.operators.spark_submit import SparkSubmitOperator
from datetime import datetime, timedelta

default_args = {
    'owner': 'data-platform',
    'retries': 3,
    'retry_delay': timedelta(minutes=5)
}

with DAG(
    'ingestion_fichiers_partenaires',
    default_args=default_args,
    schedule_interval='*/15 * * * *',  # Toutes les 15 minutes
    catchup=False
) as dag:
  
    # Détection de nouveaux fichiers
    detect_fichiers = S3KeySensor(
        task_id='detect_nouveaux_fichiers',
        bucket_name='entreprise-landing',
        bucket_key='partenaires/{{ ds }}/*.csv',
        wildcard_match=True,
        timeout=60 * 10,
        poke_interval=60
    )
  
    # Traitement Spark
    traiter_fichiers = SparkSubmitOperator(
        task_id='traiter_fichiers_csv',
        application='s3://entreprise-lakehouse/jobs/ingestion_csv.py',
        conf={
            'spark.sql.catalog.lakehouse': 'org.apache.iceberg.spark.SparkCatalog',
            'spark.sql.catalog.lakehouse.type': 'rest'
        },
        application_args=['--date', '{{ ds }}', '--source', 'partenaires']
    )
  
    detect_fichiers >> traiter_fichiers
```

### Traitement de Formats Variés

**CSV avec inférence et validation de schéma** :

```python
from pyspark.sql.functions import input_file_name, current_timestamp
from pyspark.sql.types import StructType, StructField, StringType, DecimalType, DateType

# Schéma attendu (contrat avec le partenaire)
schema_attendu = StructType([
    StructField("code_produit", StringType(), False),
    StructField("description", StringType(), True),
    StructField("prix_unitaire", DecimalType(10, 2), False),
    StructField("date_effet", DateType(), False)
])

# Lecture avec validation
df_fichiers = spark.read \
    .option("header", "true") \
    .option("delimiter", ";") \
    .option("encoding", "UTF-8") \
    .option("mode", "PERMISSIVE") \
    .option("columnNameOfCorruptRecord", "_corrupt_record") \
    .schema(schema_attendu) \
    .csv("s3://entreprise-landing/partenaires/2024-01-15/*.csv")

# Ajout de métadonnées de traçabilité
df_enrichi = df_fichiers \
    .withColumn("fichier_source", input_file_name()) \
    .withColumn("timestamp_ingestion", current_timestamp())

# Séparation données valides / invalides
df_valides = df_enrichi.filter(col("_corrupt_record").isNull())
df_erreurs = df_enrichi.filter(col("_corrupt_record").isNotNull())

# Écriture des données valides
df_valides.drop("_corrupt_record") \
    .writeTo("lakehouse.bronze.produits_partenaire") \
    .append()

# Archivage des erreurs pour analyse
df_erreurs.writeTo("lakehouse.quarantine.produits_erreurs").append()
```

**JSON semi-structuré avec schéma évolutif** :

```python
from pyspark.sql.functions import explode, get_json_object

# Lecture JSON avec inférence de schéma
df_json = spark.read \
    .option("multiLine", "true") \
    .option("mode", "PERMISSIVE") \
    .json("s3://entreprise-landing/api-exports/2024-01-15/")

# Aplatissement de structures imbriquées
df_aplati = df_json \
    .select(
        col("transaction_id"),
        col("client.id").alias("client_id"),
        col("client.nom").alias("client_nom"),
        explode("lignes_commande").alias("ligne")
    ) \
    .select(
        col("transaction_id"),
        col("client_id"),
        col("client_nom"),
        col("ligne.produit_id"),
        col("ligne.quantite"),
        col("ligne.prix_unitaire")
    )

# Écriture avec évolution de schéma activée
df_aplati.writeTo("lakehouse.bronze.transactions_api") \
    .option("merge-schema", "true") \
    .append()
```

---

## Qualité des Données à l'Ingestion

### Validation en Ligne

L'intégration de contrôles de qualité directement dans les pipelines d'ingestion permet de détecter les anomalies au plus tôt, réduisant les coûts de correction et protégeant l'intégrité du Lakehouse.

**Framework de validation avec Great Expectations** :

```python
import great_expectations as gx
from great_expectations.core.batch import RuntimeBatchRequest

# Configuration du contexte
context = gx.get_context()

# Définition des attentes pour les transactions
expectation_suite = context.create_expectation_suite(
    expectation_suite_name="transactions_ingestion",
    overwrite_existing=True
)

# Attentes de base
expectation_suite.add_expectation(
    gx.core.ExpectationConfiguration(
        expectation_type="expect_column_values_to_not_be_null",
        kwargs={"column": "transaction_id"}
    )
)

expectation_suite.add_expectation(
    gx.core.ExpectationConfiguration(
        expectation_type="expect_column_values_to_be_between",
        kwargs={"column": "montant", "min_value": 0, "max_value": 1000000}
    )
)

expectation_suite.add_expectation(
    gx.core.ExpectationConfiguration(
        expectation_type="expect_column_values_to_match_regex",
        kwargs={"column": "courriel", "regex": r"^[\w\.-]+@[\w\.-]+\.\w+$"}
    )
)

# Validation dans le pipeline Spark
def validate_batch(batch_df, batch_id):
    # Conversion en DataFrame Pandas pour GX
    pandas_df = batch_df.toPandas()
  
    batch_request = RuntimeBatchRequest(
        datasource_name="spark_datasource",
        data_connector_name="runtime_data_connector",
        data_asset_name="transactions_batch",
        runtime_parameters={"batch_data": pandas_df},
        batch_identifiers={"batch_id": str(batch_id)}
    )
  
    validator = context.get_validator(
        batch_request=batch_request,
        expectation_suite_name="transactions_ingestion"
    )
  
    results = validator.validate()
  
    if results.success:
        # Écriture dans la table principale
        batch_df.writeTo("lakehouse.bronze.transactions").append()
    else:
        # Quarantaine des données invalides
        batch_df.writeTo("lakehouse.quarantine.transactions").append()
        # Alerte
        send_quality_alert(results, batch_id)
```

### Déduplication

La déduplication à l'ingestion prévient l'accumulation de doublons qui dégradent les analyses et augmentent les coûts de stockage.

**Déduplication en streaming** :

```python
from pyspark.sql.functions import window, row_number
from pyspark.sql.window import Window

def deduplicate_stream(df):
    # Fenêtre de déduplication: garde le premier enregistrement par clé
    window_spec = Window \
        .partitionBy("transaction_id") \
        .orderBy(col("timestamp_evenement").asc())
  
    df_dedup = df \
        .withColumn("row_num", row_number().over(window_spec)) \
        .filter(col("row_num") == 1) \
        .drop("row_num")
  
    return df_dedup

# Application dans le pipeline streaming
df_stream \
    .transform(deduplicate_stream) \
    .writeStream \
    .format("iceberg") \
    .option("path", "lakehouse.bronze.transactions") \
    .start()
```

**Déduplication batch avec Iceberg MERGE** :

```sql
-- Déduplication lors de l'insertion
MERGE INTO lakehouse.bronze.transactions t
USING (
    SELECT * FROM (
        SELECT *, ROW_NUMBER() OVER (
            PARTITION BY transaction_id 
            ORDER BY timestamp_evenement DESC
        ) as rn
        FROM staging.nouvelles_transactions
    ) WHERE rn = 1
) s
ON t.transaction_id = s.transaction_id
WHEN NOT MATCHED THEN INSERT *;
```

### Gestion des Erreurs et Quarantaine

Une stratégie robuste de gestion des erreurs distingue les pipelines de production fiables des prototypes fragiles.

**Architecture de quarantaine** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         FLUX D'INGESTION                                │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
                    ┌─────────────────────────┐
                    │   Validation Schéma     │
                    └─────────────┬───────────┘
                          ┌───────┴───────┐
                          ▼               ▼
                       [PASS]          [FAIL]
                          │               │
                          ▼               ▼
            ┌─────────────────────┐  ┌─────────────────────┐
            │ Validation Qualité  │  │  quarantine.schema  │
            └─────────┬───────────┘  └─────────────────────┘
                ┌─────┴─────┐
                ▼           ▼
             [PASS]      [FAIL]
                │           │
                ▼           ▼
    ┌───────────────────┐  ┌─────────────────────┐
    │ bronze.table      │  │ quarantine.quality  │
    └───────────────────┘  └─────────────────────┘
```

**Implémentation de la quarantaine** :

```python
from dataclasses import dataclass
from enum import Enum
from datetime import datetime

class ErreurType(Enum):
    SCHEMA = "schema_mismatch"
    QUALITE = "quality_failure"
    TRANSFORMATION = "transformation_error"
    TIMEOUT = "processing_timeout"

@dataclass
class EnregistrementQuarantaine:
    donnees_originales: str
    type_erreur: ErreurType
    message_erreur: str
    source: str
    timestamp_erreur: datetime
    tentatives: int

def process_with_quarantine(batch_df, batch_id):
    try:
        # Validation schéma
        df_valide_schema = validate_schema(batch_df)
        df_erreurs_schema = get_schema_errors(batch_df)
      
        # Quarantaine des erreurs de schéma
        if df_erreurs_schema.count() > 0:
            quarantine_records(df_erreurs_schema, ErreurType.SCHEMA)
      
        # Validation qualité
        df_valide_qualite = validate_quality(df_valide_schema)
        df_erreurs_qualite = get_quality_errors(df_valide_schema)
      
        # Quarantaine des erreurs de qualité
        if df_erreurs_qualite.count() > 0:
            quarantine_records(df_erreurs_qualite, ErreurType.QUALITE)
      
        # Écriture des données valides
        df_valide_qualite.writeTo("lakehouse.bronze.transactions").append()
      
        # Métriques
        log_ingestion_metrics(batch_id, 
                            total=batch_df.count(),
                            valides=df_valide_qualite.count(),
                            quarantaine=df_erreurs_schema.count() + df_erreurs_qualite.count())
      
    except Exception as e:
        # Quarantaine complète du batch en cas d'erreur inattendue
        quarantine_entire_batch(batch_df, ErreurType.TRANSFORMATION, str(e))
        raise

def quarantine_records(df_erreurs, type_erreur):
    df_quarantaine = df_erreurs \
        .withColumn("type_erreur", lit(type_erreur.value)) \
        .withColumn("timestamp_quarantaine", current_timestamp()) \
        .withColumn("batch_id", lit(batch_id))
  
    df_quarantaine.writeTo("lakehouse.quarantine.transactions").append()
```

---

## Orchestration des Pipelines

### Apache Airflow pour l'Orchestration

Apache Airflow demeure le standard de facto pour l'orchestration des pipelines de données complexes. Son modèle de DAG (Directed Acyclic Graph) permet d'exprimer les dépendances entre tâches et de gérer les reprises sur erreur.

**DAG d'ingestion complète** :

```python
from airflow import DAG
from airflow.operators.python import PythonOperator, BranchPythonOperator
from airflow.providers.apache.spark.operators.spark_submit import SparkSubmitOperator
from airflow.providers.slack.operators.slack_webhook import SlackWebhookOperator
from airflow.utils.task_group import TaskGroup
from datetime import datetime, timedelta

default_args = {
    'owner': 'data-engineering',
    'depends_on_past': False,
    'email_on_failure': True,
    'email': ['data-alerts@entreprise.ca'],
    'retries': 3,
    'retry_delay': timedelta(minutes=5),
    'retry_exponential_backoff': True
}

with DAG(
    'ingestion_quotidienne_ventes',
    default_args=default_args,
    description='Pipeline d\'ingestion des données de ventes',
    schedule_interval='0 6 * * *',  # 6h00 quotidien
    start_date=datetime(2024, 1, 1),
    catchup=False,
    tags=['ingestion', 'ventes', 'production']
) as dag:
  
    # Vérification de la disponibilité des sources
    with TaskGroup('verification_sources') as verification:
        verif_postgres = PythonOperator(
            task_id='verif_postgres',
            python_callable=check_postgres_availability
        )
      
        verif_api = PythonOperator(
            task_id='verif_api',
            python_callable=check_api_availability
        )
  
    # Extraction parallèle
    with TaskGroup('extraction') as extraction:
        extract_transactions = SparkSubmitOperator(
            task_id='extract_transactions',
            application='s3://lakehouse/jobs/extract_transactions.py',
            conf={'spark.executor.instances': '10'},
            application_args=['--date', '{{ ds }}']
        )
      
        extract_clients = SparkSubmitOperator(
            task_id='extract_clients',
            application='s3://lakehouse/jobs/extract_clients.py',
            application_args=['--date', '{{ ds }}']
        )
      
        extract_produits = SparkSubmitOperator(
            task_id='extract_produits',
            application='s3://lakehouse/jobs/extract_produits.py',
            application_args=['--date', '{{ ds }}']
        )
  
    # Validation qualité
    validation = SparkSubmitOperator(
        task_id='validation_qualite',
        application='s3://lakehouse/jobs/validate_ingestion.py',
        application_args=['--date', '{{ ds }}']
    )
  
    # Branchement selon résultats validation
    def decide_next_step(**context):
        validation_result = context['ti'].xcom_pull(task_ids='validation_qualite')
        if validation_result['success_rate'] > 0.99:
            return 'transformation'
        else:
            return 'alerte_qualite'
  
    branch = BranchPythonOperator(
        task_id='branch_validation',
        python_callable=decide_next_step
    )
  
    # Transformation Silver
    transformation = SparkSubmitOperator(
        task_id='transformation',
        application='s3://lakehouse/jobs/transform_silver.py',
        application_args=['--date', '{{ ds }}']
    )
  
    # Alerte en cas de problème
    alerte = SlackWebhookOperator(
        task_id='alerte_qualite',
        slack_webhook_conn_id='slack_data_alerts',
        message='⚠️ Qualité des données insuffisante pour {{ ds }}'
    )
  
    # Dépendances
    verification >> extraction >> validation >> branch
    branch >> [transformation, alerte]
```

### Patterns d'Orchestration Avancés

**Ingestion incrémentale avec gestion d'état** :

```python
from airflow.models import Variable
from airflow.operators.python import PythonOperator

def get_watermark(**context):
    """Récupère le dernier watermark traité"""
    watermark = Variable.get(
        'watermark_transactions', 
        default_var='1970-01-01T00:00:00Z'
    )
    context['ti'].xcom_push(key='watermark', value=watermark)
    return watermark

def update_watermark(**context):
    """Met à jour le watermark après traitement réussi"""
    new_watermark = context['ti'].xcom_pull(
        task_ids='extract_incremental', 
        key='max_timestamp'
    )
    Variable.set('watermark_transactions', new_watermark)

with DAG('ingestion_incrementale') as dag:
  
    get_wm = PythonOperator(
        task_id='get_watermark',
        python_callable=get_watermark
    )
  
    extract = SparkSubmitOperator(
        task_id='extract_incremental',
        application='s3://lakehouse/jobs/extract_incremental.py',
        application_args=[
            '--watermark', '{{ ti.xcom_pull(task_ids="get_watermark", key="watermark") }}'
        ]
    )
  
    update_wm = PythonOperator(
        task_id='update_watermark',
        python_callable=update_watermark,
        trigger_rule='all_success'
    )
  
    get_wm >> extract >> update_wm
```

**Backfill contrôlé** :

```python
from airflow.operators.python import PythonOperator
from datetime import datetime, timedelta

def generate_backfill_dates(start_date, end_date):
    """Génère les dates à retraiter"""
    dates = []
    current = start_date
    while current <= end_date:
        dates.append(current.strftime('%Y-%m-%d'))
        current += timedelta(days=1)
    return dates

with DAG(
    'backfill_transactions',
    schedule_interval=None,  # Déclenché manuellement
    params={
        'start_date': '2024-01-01',
        'end_date': '2024-01-31'
    }
) as dag:
  
    def run_backfill(**context):
        start = datetime.strptime(context['params']['start_date'], '%Y-%m-%d')
        end = datetime.strptime(context['params']['end_date'], '%Y-%m-%d')
        dates = generate_backfill_dates(start, end)
      
        for date in dates:
            # Déclenchement du DAG d'ingestion pour chaque date
            trigger_dag_run('ingestion_quotidienne_ventes', date)
  
    backfill = PythonOperator(
        task_id='run_backfill',
        python_callable=run_backfill
    )
```

---

## Performance et Optimisation

### Dimensionnement des Pipelines

Le dimensionnement correct des pipelines d'ingestion équilibre performance, coûts et fiabilité.

**Facteurs de dimensionnement** :

| Composant        | Métrique clé          | Formule de base                     |
| ---------------- | ----------------------- | ----------------------------------- |
| Partitions Kafka | Parallélisme           | Volume/heure ÷ 10 Mo/s/partition   |
| Executors Spark  | Capacité traitement    | Volume batch ÷ 100 Mo/executor/min |
| Tasks Flink      | Parallélisme streaming | Partitions Kafka × 1-2             |
| Workers Connect  | Connecteurs             | 1 worker par 3-5 connecteurs        |

**Exemple de calcul** :

```
Scénario: 10 To/jour, latence cible 15 minutes

Volume par batch (15 min): 10 To ÷ 96 = ~104 Go
Débit requis: 104 Go ÷ 15 min = ~7 Go/min = ~115 Mo/s

Partitions Kafka: 115 Mo/s ÷ 10 Mo/s = 12 partitions (arrondi à 16)
Executors Spark: 104 Go ÷ (100 Mo × 15) = 70 executors
Mémoire totale: 70 × 8 Go = 560 Go

Configuration recommandée:
- Kafka: 16 partitions, 3 brokers, rétention 7 jours
- Spark: 70 executors × 4 cores × 8 Go RAM
- Iceberg: fichiers 512 Mo, compaction quotidienne
```

### Optimisation des Écritures Iceberg

**Configuration pour haute performance** :

```python
# Propriétés de table optimisées pour l'ingestion
spark.sql("""
    ALTER TABLE lakehouse.bronze.transactions SET TBLPROPERTIES (
        'write.target-file-size-bytes' = '536870912',
        'write.distribution-mode' = 'hash',
        'write.parquet.compression-codec' = 'zstd',
        'write.parquet.compression-level' = '3',
        'write.metadata.compression-codec' = 'gzip',
        'commit.retry.num-retries' = '10',
        'commit.retry.min-wait-ms' = '100',
        'commit.retry.max-wait-ms' = '60000'
    )
""")
```

**Distribution des données** :

```python
# Distribution par hash pour parallélisme optimal
df.writeTo("lakehouse.bronze.transactions") \
    .option("write.distribution-mode", "hash") \
    .option("write.fanout.enabled", "true") \
    .append()

# Distribution par range pour données ordonnées
df.sortWithinPartitions("timestamp_evenement") \
    .writeTo("lakehouse.bronze.evenements") \
    .option("write.distribution-mode", "range") \
    .append()
```

> **Performance**
>
> Le mode `fanout` permet à chaque tâche Spark d'écrire vers toutes les partitions Iceberg simultanément, améliorant le parallélisme au prix d'une utilisation mémoire accrue. Activez-le pour les pipelines avec de nombreuses partitions cibles.

### Monitoring et Observabilité

**Métriques clés à surveiller** :

| Métrique                | Seuil d'alerte | Action corrective               |
| ------------------------ | -------------- | ------------------------------- |
| Lag consommateur Kafka   | > 5 min        | Augmenter parallélisme         |
| Durée de commit Iceberg | > 60s          | Réduire taille des batches     |
| Fichiers par commit      | > 1000         | Activer compaction              |
| Taux d'erreurs           | > 1%           | Investigation qualité données |
| Utilisation mémoire     | > 80%          | Augmenter ressources            |

**Dashboard Grafana pour ingestion** :

```yaml
# Configuration Prometheus pour métriques Spark
- job_name: 'spark-ingestion'
  static_configs:
    - targets: ['spark-driver:4040']
  metrics_path: /metrics/prometheus

# Alertes critiques
groups:
  - name: ingestion-alerts
    rules:
      - alert: IngestionLagCritique
        expr: kafka_consumer_lag > 300000
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Lag d'ingestion critique"
        
      - alert: IcebergCommitLent
        expr: iceberg_commit_duration_seconds > 120
        for: 3m
        labels:
          severity: warning
        annotations:
          summary: "Commits Iceberg anormalement lents"
```

---

## Études de Cas Canadiennes

### Secteur Télécommunications : Opérateur National

> **Étude de cas : Grand opérateur de télécommunications canadien**
>
> *Secteur* : Télécommunications
>
> *Défi* : Ingérer 500 millions d'événements réseau quotidiens (CDR, logs réseau, métriques QoS) avec une latence maximale de 5 minutes pour alimenter les tableaux de bord opérationnels et les systèmes de détection d'anomalies.
>
> *Solution* : Architecture Streaming Lakehouse avec Kafka (80 partitions), Apache Flink pour le traitement temps réel, et Iceberg sur ADLS Gen2. CDC Debezium pour la synchronisation des données clients depuis les systèmes CRM et de facturation.
>
> *Architecture* :
>
> ```
> Sources réseau → Kafka (500M evt/jour) → Flink → Iceberg Bronze
>                                            ↓
>                          Flink (agrégation) → Iceberg Silver
>                                            ↓
>                              Dremio → Tableaux de bord
> ```
>
> *Résultats* :
>
> * Latence médiane de 2 minutes (P99 < 5 min)
> * Réduction de 80% du temps de détection des anomalies réseau
> * Économies de 3M$/an par rapport à la solution propriétaire précédente
> * Conformité maintenue avec les exigences du CRTC

### Secteur Commerce de Détail : Chaîne Nationale

> **Étude de cas : Chaîne de magasins pancanadienne**
>
> *Secteur* : Commerce de détail
>
> *Défi* : Unifier les données de 400 magasins, du commerce électronique et des programmes de fidélité pour créer une vue client 360° alimentant la personnalisation en temps réel.
>
> *Solution* : Ingestion hybride combinant CDC (Debezium) pour les systèmes transactionnels POS, API webhooks pour le commerce électronique, et batch pour les fichiers partenaires. Kafka comme bus central, Spark Structured Streaming pour le micro-batch (5 min), et Iceberg sur S3.
>
> *Résultats* :
>
> * Vue client unifiée avec latence < 10 minutes
> * Augmentation de 15% du taux de conversion des recommandations
> * Réduction de 60% du temps de préparation des campagnes marketing
> * Capacité d'analyse cross-canal impossible auparavant

### Secteur Public : Agence Gouvernementale

> **Étude de cas : Agence fédérale de statistiques**
>
> *Secteur* : Gouvernement fédéral
>
> *Défi* : Moderniser l'infrastructure d'ingestion pour le recensement et les enquêtes continues, avec des exigences strictes de sécurité (Protégé B) et de résidence des données au Canada.
>
> *Solution* : Déploiement sur site avec MinIO pour le stockage, Kafka sur Kubernetes pour le bus événementiel, et Spark pour l'ingestion. Toute l'infrastructure hébergée dans les centres de données gouvernementaux certifiés.
>
> *Particularités* :
>
> * Chiffrement de bout en bout avec clés gérées par l'agence
> * Audit complet de toutes les opérations d'ingestion
> * Isolation réseau stricte entre environnements
>
>   *Résultats* :
> * Conformité Protégé B validée par audit indépendant
> * Réduction de 70% du temps de traitement des enquêtes
> * Capacité d'intégrer de nouvelles sources en semaines plutôt qu'en mois

---

## Matrice de Décision Technologique

### Choix du Moteur d'Ingestion

| Critère                   | Kafka Connect     | Spark Streaming | Apache Flink |
| -------------------------- | ----------------- | --------------- | ------------ |
| Latence                    | 1-5 min           | 1-15 min        | < 1 sec      |
| Complexité transformation | Faible            | Élevée        | Élevée     |
| Exactly-once               | Oui (avec config) | Oui             | Oui          |
| Courbe apprentissage       | Faible            | Moyenne         | Élevée     |
| Opérations                | Simple            | Modéré        | Complexe     |
| Cas d'usage                | CDC direct        | ETL batch/micro | Temps réel  |

### Arbre de Décision

```
                    ┌─────────────────────────────────┐
                    │  Quelle est votre latence cible?│
                    └─────────────────────────────────┘
                                    │
            ┌───────────────────────┼───────────────────────┐
            ▼                       ▼                       ▼
        [< 1 sec]              [1-15 min]               [> 15 min]
            │                       │                       │
            ▼                       ▼                       ▼
    ┌───────────────┐       ┌───────────────┐       ┌───────────────┐
    │ Transformations│       │Transformations│       │ Spark Batch   │
    │ complexes?     │       │ complexes?    │       └───────────────┘
    └───────────────┘       └───────────────┘
            │                       │
      ┌─────┴─────┐           ┌─────┴─────┐
      ▼           ▼           ▼           ▼
    [OUI]       [NON]       [OUI]       [NON]
      │           │           │           │
      ▼           ▼           ▼           ▼
   Flink     Kafka Connect   Spark    Kafka Connect
            + Flink aval   Streaming
```

---

## Conclusion

L'architecture de la couche d'ingestion détermine la capacité de votre Data Lakehouse à transformer des données brutes dispersées en actifs analytiques exploitables. Une conception rigoureuse de cette couche — combinant les bons patterns d'ingestion, les technologies appropriées et des pratiques opérationnelles solides — constitue le fondement sur lequel repose toute votre stratégie de données.

Apache Iceberg transforme la manière dont nous concevons l'ingestion en unifiant les paradigmes batch et streaming au sein d'une architecture cohérente. Les commits atomiques, l'évolution de schéma et le support des écritures concurrentes éliminent les compromis historiques entre fraîcheur des données et fiabilité des pipelines.

L'écosystème technologique disponible — Kafka pour le découplage, Debezium pour le CDC, Spark et Flink pour le traitement, Airflow pour l'orchestration — offre des composantes matures et interopérables pour construire des pipelines de classe production. Le choix entre ces technologies dépend de vos exigences spécifiques de latence, de complexité de transformation et de capacités opérationnelles.

Les études de cas canadiennes présentées démontrent que ces architectures s'appliquent avec succès dans des contextes variés — télécommunications, commerce de détail, secteur public — tout en respectant les contraintes réglementaires locales. La résidence des données au Canada, les certifications Protégé B et les exigences sectorielles spécifiques sont compatibles avec une architecture Lakehouse moderne.

Le chapitre suivant explore la couche de catalogue, composante essentielle qui organise et gouverne les métadonnées de votre Lakehouse, permettant la découverte, l'accès contrôlé et l'évolution des tables Iceberg à l'échelle de l'entreprise.

---

## Résumé

**Patterns d'ingestion** :

* Batch pour les volumes massifs avec latence tolérante (heures)
* Micro-batch pour l'équilibre latence/efficacité (minutes)
* Streaming pour les exigences temps réel (secondes)
* Le Streaming Lakehouse unifie ces patterns avec Iceberg

**Technologies clés** :

* Apache Kafka comme bus événementiel central
* Debezium pour le CDC depuis les bases relationnelles
* Spark Structured Streaming pour le micro-batch
* Apache Flink pour le streaming temps réel
* Kafka Connect pour l'ingestion directe simple

**Stratégies de chargement** :

* Append pour les données nouvelles sans mise à jour
* MERGE INTO pour les upserts CDC
* Gestion des suppressions : physique, logique ou historisée

**Qualité des données** :

* Validation en ligne avec Great Expectations
* Déduplication intégrée aux pipelines
* Architecture de quarantaine pour les données invalides

**Optimisations** :

* Distribution hash pour parallélisme optimal
* Dimensionnement fichiers 256-512 Mo
* Configuration des retries pour commits robustes
* Monitoring Kafka lag et durée commits

**Orchestration** :

* Airflow pour les pipelines complexes
* Gestion d'état pour l'ingestion incrémentale
* Backfill contrôlé pour les reprises historiques

---

*Ce chapitre établit les fondations de l'alimentation de votre Lakehouse. Le chapitre suivant, « Configuration du Catalogue Iceberg », détaille comment organiser, gouverner et exposer les métadonnées de vos tables pour une exploitation à l'échelle de l'entreprise.*
