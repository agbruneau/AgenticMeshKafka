# Chapitre IV.12 - L'ÉVOLUTION VERS LE STREAMING LAKEHOUSE

---

## Introduction

L'histoire des architectures de données est jalonnée de compromis entre latence et complétude, entre fraîcheur et fiabilité. Pendant plus d'une décennie, les organisations ont jonglé avec des systèmes parallèles : d'un côté, les pipelines de traitement par lots (batch) offrant une vue complète mais décalée des données; de l'autre, les flux en temps réel capturant l'immédiateté au prix de la complexité. L'architecture Lambda, proposée par Nathan Marz en 2011, a tenté de réconcilier ces deux mondes en les faisant coexister. Mais cette cohabitation forcée a engendré une dette technique considérable : duplication de code, divergence des résultats, et coûts opérationnels exponentiels.

Le Streaming Lakehouse représente l'aboutissement d'une quête de plus de quinze ans vers l'unification des données en mouvement et au repos. En combinant Apache Kafka comme colonne vertébrale événementielle avec Apache Iceberg comme format de table ouvert, les organisations peuvent enfin construire une architecture où chaque événement capturé devient instantanément disponible pour l'analyse, l'intelligence artificielle et les applications opérationnelles, sans duplication ni transformation redondante.

Ce chapitre retrace l'évolution architecturale qui a conduit au Streaming Lakehouse moderne. Nous examinerons d'abord les limites des architectures Lambda et Kappa qui ont dominé la dernière décennie. Puis nous explorerons le rôle central de Confluent et Apache Kafka dans cette convergence, avec une attention particulière portée à Tableflow, la fonctionnalité qui matérialise les topics Kafka directement en tables Iceberg. Enfin, nous présenterons les patterns d'implémentation et les considérations pratiques pour les organisations canadiennes qui entreprennent cette transformation.

---

## IV.12.1 De l'Architecture Lambda au Streaming Lakehouse

### Les Origines : Le Défi du Big Data

Au tournant des années 2010, l'explosion des volumes de données a confronté les architectes à un dilemme fondamental. Les systèmes traditionnels de traitement par lots, hérités de l'ère MapReduce et Hadoop, excellaient dans le traitement exhaustif de grands volumes historiques. Ils offraient des garanties de complétude et de cohérence que les systèmes transactionnels ne pouvaient égaler à grande échelle. Cependant, leur latence intrinsèque, mesurée en heures voire en jours, les rendait inadaptés aux exigences croissantes de réactivité.

Parallèlement, les premiers systèmes de traitement de flux comme Storm (développé chez Twitter) et S4 (Yahoo) démontraient qu'il était possible de traiter des événements avec des latences de l'ordre de la milliseconde. Mais ces systèmes sacrifiaient souvent les garanties de cohérence et la capacité de retraitement historique qui caractérisaient le batch.

### L'Architecture Lambda : Une Réconciliation Imparfaite

Face à ce dilemme, Nathan Marz a proposé en 2011 l'architecture Lambda, une approche hybride qui fait coexister deux chemins de traitement parallèles :

**La couche batch (Batch Layer)** traite l'intégralité des données historiques selon des cycles réguliers (typiquement journaliers ou horaires). Elle utilise des technologies comme Hadoop MapReduce ou Apache Spark pour produire des « vues batch » exhaustives et cohérentes. Cette couche constitue la source de vérité, capable de corriger toute erreur introduite par la couche temps réel.

**La couche vitesse (Speed Layer)** traite les données en temps réel à mesure de leur arrivée. Elle produit des « vues temps réel » qui compensent la latence de la couche batch. Ces vues sont approximatives et temporaires, destinées à être remplacées par les résultats batch une fois disponibles.

**La couche de service (Serving Layer)** fusionne les vues batch et temps réel pour présenter une vue unifiée aux consommateurs. Elle gère la complexité de la réconciliation entre des résultats potentiellement divergents.

```
┌─────────────────────────────────────────────────────────────────┐
│                    ARCHITECTURE LAMBDA                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────┐                                               │
│   │   Sources   │                                               │
│   │  de Données │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌──────────────┐                                              │
│   │  Master      │                                              │
│   │  Dataset     │                                              │
│   └──────┬───────┘                                              │
│          │                                                      │
│    ┌─────┴─────┐                                                │
│    │           │                                                │
│    ▼           ▼                                                │
│ ┌──────────┐ ┌──────────┐                                       │
│ │  Batch   │ │  Speed   │                                       │
│ │  Layer   │ │  Layer   │                                       │
│ │(Hadoop/  │ │(Storm/   │                                       │
│ │ Spark)   │ │ Samza)   │                                       │
│ └────┬─────┘ └────┬─────┘                                       │
│      │            │                                             │
│      ▼            ▼                                             │
│ ┌──────────┐ ┌──────────┐                                       │
│ │  Batch   │ │Real-time │                                       │
│ │  Views   │ │  Views   │                                       │
│ └────┬─────┘ └────┬─────┘                                       │
│      │            │                                             │
│      └─────┬──────┘                                             │
│            ▼                                                    │
│     ┌─────────────┐                                             │
│     │   Serving   │                                             │
│     │    Layer    │                                             │
│     └─────────────┘                                             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Les Limites Fondamentales de Lambda

Malgré son élégance conceptuelle, l'architecture Lambda souffre de limitations profondes qui ont progressivement conduit à son abandon par les organisations les plus avancées.

**La duplication du code** constitue le problème le plus immédiat. La même logique métier doit être implémentée deux fois : une version batch (souvent en Scala/Spark) et une version streaming (Storm, Samza, ou autre). Ces implémentations divergent inévitablement au fil du temps, introduisant des incohérences subtiles entre les résultats batch et temps réel.

**La divergence des résultats** découle de cette duplication. Les vues batch et temps réel peuvent produire des résultats différents pour les mêmes données, créant des situations où les utilisateurs observent des « sauts » dans les métriques lors de la réconciliation.

**La complexité opérationnelle** est considérable. L'équipe doit maîtriser et opérer deux écosystèmes technologiques distincts, chacun avec ses propres modes de défaillance, patterns de déploiement et exigences de monitoring.

**Les coûts d'infrastructure** sont doublés. Les données sont stockées et traitées deux fois, multipliant les besoins en calcul et en stockage.

> **Migration**  
> *De* : Architecture Lambda avec Hadoop + Storm  
> *Vers* : Streaming Lakehouse unifié  
> *Stratégie* : Migration incrémentale en commençant par les cas d'usage les plus critiques en latence. Maintenir Lambda en parallèle pendant la transition, puis décommissionner progressivement la couche batch.

### L'Architecture Kappa : La Simplification par le Flux

En 2014, Jay Kreps, cofondateur d'Apache Kafka, a proposé l'architecture Kappa comme alternative radicale à Lambda. Son principe fondateur est simple : traiter toutes les données comme un flux, éliminant ainsi la distinction entre batch et temps réel.

Dans Kappa, le journal d'événements immuable (typiquement un cluster Kafka) devient la source de vérité unique. Toutes les transformations sont exprimées comme des applications de traitement de flux. Lorsqu'une correction ou une évolution de la logique est nécessaire, l'application est simplement rejouée depuis le début du journal pour reconstruire l'état correct.

```
┌─────────────────────────────────────────────────────────────────┐
│                    ARCHITECTURE KAPPA                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────┐                                               │
│   │   Sources   │                                               │
│   │  de Données │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌──────────────────────────────────────────┐                  │
│   │         Journal Immuable (Kafka)         │                  │
│   │    Source de Vérité Unique               │                  │
│   └──────────────────┬───────────────────────┘                  │
│                      │                                          │
│                      ▼                                          │
│   ┌──────────────────────────────────────────┐                  │
│   │      Traitement de Flux Unifié           │                  │
│   │      (Kafka Streams / Flink)             │                  │
│   └──────────────────┬───────────────────────┘                  │
│                      │                                          │
│                      ▼                                          │
│   ┌──────────────────────────────────────────┐                  │
│   │         Vues Matérialisées               │                  │
│   │    (Tables, Caches, Index)               │                  │
│   └──────────────────────────────────────────┘                  │
│                                                                 │
│   Retraitement = Rejouer depuis le début du journal             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

L'architecture Kappa offre des avantages significatifs par rapport à Lambda. L'élimination de la duplication de code simplifie drastiquement le développement et la maintenance. La source de vérité unique garantit la cohérence des résultats. Les coûts opérationnels sont réduits par l'utilisation d'une seule pile technologique.

Cependant, Kappa présente ses propres limitations. Le retraitement depuis le début du journal peut s'avérer coûteux en temps et en ressources pour des historiques volumineux. Les résultats intermédiaires stockés dans Kafka ne sont pas directement interrogeables par SQL, limitant l'accès aux analystes et aux outils d'intelligence d'affaires. Enfin, le stockage prolongé dans Kafka peut devenir onéreux comparé aux solutions de stockage objet.

### L'Émergence du Data Lakehouse

Le concept de Data Lakehouse, introduit vers 2017, a apporté une réponse partielle à ces limitations. En combinant la scalabilité économique des lacs de données avec les garanties transactionnelles des entrepôts de données, les formats de table ouverts comme Apache Iceberg, Delta Lake et Apache Hudi ont transformé les lacs de données en systèmes fiables et interrogeables.

Ces formats introduisent des capacités essentielles pour la gestion de données à l'échelle de l'entreprise. Les transactions ACID garantissent la cohérence des lectures et écritures concurrentes. L'évolution de schéma permet de modifier la structure des tables sans réécriture complète. Le partitionnement masqué (hidden partitioning) optimise les performances de requête sans exposer la complexité aux utilisateurs. Le time travel permet d'interroger les données telles qu'elles existaient à un instant donné.

Cependant, les Lakehouses traditionnels restent fondamentalement orientés batch. Leurs optimisations ciblent les requêtes analytiques sur de grands volumes statiques plutôt que l'ingestion continue à faible latence. L'intégration avec les systèmes de streaming nécessite des pipelines ETL additionnels, réintroduisant la complexité que Kappa cherchait à éliminer.

### Le Streaming Lakehouse : La Convergence

Le Streaming Lakehouse représente la synthèse de ces évolutions architecturales. Il unifie les forces de Kappa (traitement unifié, source de vérité événementielle) avec celles du Lakehouse (stockage économique, interrogation SQL, gouvernance des données) en une architecture cohérente.

```
┌─────────────────────────────────────────────────────────────────┐
│                   STREAMING LAKEHOUSE                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│   │Applications │  │     CDC     │  │    IoT      │            │
│   │   & APIs    │  │  Debezium   │  │  Capteurs   │            │
│   └──────┬──────┘  └──────┬──────┘  └──────┬──────┘            │
│          │                │                │                    │
│          └────────────────┼────────────────┘                    │
│                           ▼                                     │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │              Apache Kafka (Dorsale Événementielle)       │  │
│   │         Source de Vérité pour les Données en Mouvement   │  │
│   │              + Schema Registry + Gouvernance             │  │
│   └───────────────────────┬─────────────────────────────────┘  │
│                           │                                     │
│              ┌────────────┼────────────┐                       │
│              │            │            │                        │
│              ▼            ▼            ▼                        │
│   ┌──────────────┐ ┌───────────┐ ┌───────────────┐             │
│   │Apache Flink  │ │Tableflow  │ │ Kafka Streams │             │
│   │(Traitement   │ │(Matéria-  │ │(Microservices)│             │
│   │ Complexe)    │ │ lisation) │ │               │             │
│   └──────┬───────┘ └─────┬─────┘ └───────────────┘             │
│          │               │                                      │
│          └───────┬───────┘                                      │
│                  ▼                                              │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │            Apache Iceberg (Format de Table Ouvert)       │  │
│   │         Source de Vérité pour les Données au Repos       │  │
│   │        Stockage Objet (S3/ADLS/GCS) + REST Catalog       │  │
│   └───────────────────────┬─────────────────────────────────┘  │
│                           │                                     │
│              ┌────────────┼────────────┐                       │
│              │            │            │                        │
│              ▼            ▼            ▼                        │
│   ┌──────────────┐ ┌───────────┐ ┌───────────────┐             │
│   │   Dremio/    │ │ Power BI  │ │   ML/AI       │             │
│   │   Trino      │ │ Tableau   │ │   Platforms   │             │
│   └──────────────┘ └───────────┘ └───────────────┘             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

Les caractéristiques distinctives du Streaming Lakehouse incluent :

**L'ingestion unifiée** : Les données transitent par Kafka quelle que soit leur source (applications, CDC, IoT, fichiers batch). Cette normalisation simplifie la gouvernance et assure la traçabilité complète.

**La matérialisation automatique** : Les topics Kafka sont automatiquement matérialisés en tables Iceberg, éliminant les pipelines ETL manuels. Des solutions comme Tableflow (Confluent) ou le Flink Dynamic Iceberg Sink gèrent cette conversion de manière transparente.

**Le stockage unique** : Les données sont stockées une seule fois dans un format ouvert (Parquet sur stockage objet), accessible par tous les moteurs de consommation sans duplication.

**L'interrogation universelle** : Les mêmes tables sont accessibles via des moteurs de streaming (Flink), des moteurs SQL analytiques (Trino, Dremio), des outils d'intelligence d'affaires (Power BI, Tableau), et des plateformes ML/AI.

### Comparaison des Architectures

Le tableau suivant synthétise les caractéristiques des trois architectures évoquées :

| Critère | Lambda | Kappa | Streaming Lakehouse |
|---------|--------|-------|---------------------|
| Duplication de code | Élevée (2 implémentations) | Aucune | Aucune |
| Latence | Mixte (temps réel + batch) | Temps réel | Temps réel + batch unifié |
| Interrogation SQL | Via couche de service | Limitée | Native sur Iceberg |
| Coût de stockage | Élevé (duplication) | Moyen (Kafka long terme) | Optimal (stockage objet) |
| Retraitement | Via couche batch | Rejeu du journal | Rejeu + time travel |
| Gouvernance | Fragmentée | Sur Kafka | Unifiée (Kafka + Iceberg) |
| Complexité opérationnelle | Élevée | Moyenne | Moyenne |
| Maturité | Mature (2011) | Mature (2014) | Émergente (2023+) |

> **Performance**  
> Les organisations qui ont migré de Lambda vers un Streaming Lakehouse rapportent typiquement une réduction de 40 à 60 % des coûts d'infrastructure, principalement grâce à l'élimination de la duplication des données et du traitement. La latence de bout en bout pour les cas d'usage analytiques passe de T+1 jour à quelques minutes.

### Le Pattern « Shift Left »

L'architecture Shift Left, décrite par Kai Waehner de Confluent, représente une évolution philosophique importante. Plutôt que de considérer la gouvernance et la qualité des données comme des préoccupations en aval (dans l'entrepôt ou le lakehouse), cette approche les déplace « vers la gauche », c'est-à-dire plus près de la source.

Dans un Streaming Lakehouse adoptant Shift Left, les transformations de données, les validations de qualité et l'enrichissement sémantique s'effectuent dans la couche de streaming (Kafka + Flink) plutôt que dans des processus ETL tardifs. Les données arrivent dans le Lakehouse déjà structurées, validées et conformes aux contrats de schéma définis dans le Schema Registry.

Ce pattern offre plusieurs avantages pour les organisations canadiennes soumises à des réglementations strictes (Loi 25, LPRPDE) :

- La traçabilité de chaque transformation est garantie par le journal Kafka
- Les règles de qualité sont appliquées en temps réel, détectant les anomalies immédiatement
- La gouvernance des schémas via le Schema Registry prévient les évolutions incompatibles
- Les métadonnées de lignage sont capturées automatiquement pour l'audit

L'implémentation du Shift Left avec Apache Flink illustre cette approche :

```sql
-- Flink SQL : Pipeline Shift Left avec validation et enrichissement
-- Les données sont nettoyées et enrichies AVANT d'atteindre le Lakehouse

-- Table source brute depuis Kafka
CREATE TABLE raw_transactions (
    transaction_id STRING,
    account_id STRING,
    amount DECIMAL(18, 2),
    currency STRING,
    merchant_name STRING,
    merchant_category STRING,
    transaction_time TIMESTAMP(3),
    WATERMARK FOR transaction_time AS transaction_time - INTERVAL '30' SECOND
) WITH (
    'connector' = 'kafka',
    'topic' = 'raw.transactions',
    'properties.bootstrap.servers' = 'kafka:9092',
    'format' = 'avro-confluent',
    'avro-confluent.url' = 'http://schema-registry:8081'
);

-- Table de référence pour enrichissement (lookup join)
CREATE TABLE merchant_categories (
    merchant_category STRING,
    category_group STRING,
    risk_level STRING,
    PRIMARY KEY (merchant_category) NOT ENFORCED
) WITH (
    'connector' = 'jdbc',
    'url' = 'jdbc:postgresql://postgres:5432/reference',
    'table-name' = 'merchant_categories',
    'lookup.cache.max-rows' = '10000',
    'lookup.cache.ttl' = '1h'
);

-- Table Iceberg cible avec données enrichies et validées
CREATE TABLE lakehouse_transactions (
    transaction_id STRING,
    account_id STRING,
    amount_cad DECIMAL(18, 2),
    original_amount DECIMAL(18, 2),
    original_currency STRING,
    merchant_name STRING,
    category_group STRING,
    risk_level STRING,
    transaction_time TIMESTAMP(3),
    processing_time TIMESTAMP(3),
    data_quality_score INT
) WITH (
    'connector' = 'iceberg',
    'catalog-name' = 'lakehouse',
    'database-name' = 'financial',
    'table-name' = 'transactions'
);

-- Pipeline de traitement avec validation et enrichissement
INSERT INTO lakehouse_transactions
SELECT 
    t.transaction_id,
    t.account_id,
    CASE t.currency 
        WHEN 'CAD' THEN t.amount
        WHEN 'USD' THEN t.amount * 1.36
        WHEN 'EUR' THEN t.amount * 1.47
        ELSE t.amount
    END AS amount_cad,
    t.amount AS original_amount,
    t.currency AS original_currency,
    COALESCE(TRIM(t.merchant_name), 'UNKNOWN') AS merchant_name,
    COALESCE(mc.category_group, 'OTHER') AS category_group,
    COALESCE(mc.risk_level, 'MEDIUM') AS risk_level,
    t.transaction_time,
    CURRENT_TIMESTAMP AS processing_time,
    CASE 
        WHEN t.amount > 0 AND t.merchant_name IS NOT NULL THEN 100
        WHEN t.amount > 0 THEN 75
        ELSE 50
    END AS data_quality_score
FROM raw_transactions t
LEFT JOIN merchant_categories FOR SYSTEM_TIME AS OF t.transaction_time AS mc
    ON t.merchant_category = mc.merchant_category
WHERE t.amount IS NOT NULL 
  AND t.transaction_id IS NOT NULL
  AND t.amount BETWEEN -1000000 AND 1000000;
```

Ce pipeline illustre les principes Shift Left :

1. **Validation à la source** : Les transactions invalides sont filtrées avant d'atteindre le Lakehouse
2. **Enrichissement en vol** : Les données de référence sont jointes en temps réel via lookup join
3. **Normalisation** : Les montants sont convertis en dollars canadiens pour uniformité
4. **Score de qualité** : Chaque enregistrement reçoit un indicateur exploitable en aval
5. **Traçabilité** : Le timestamp de traitement permet de suivre la latence du pipeline

### Couches Bronze, Silver, Gold dans le Streaming Lakehouse

Le modèle Medallion (Bronze-Silver-Gold), popularisé par Databricks, s'adapte naturellement au Streaming Lakehouse. Chaque couche représente un niveau de raffinement des données :

**Couche Bronze** : Données brutes, telles qu'ingérées depuis les sources. Stockées avec un minimum de transformation pour préserver la fidélité. Utiles pour le retraitement et l'audit.

**Couche Silver** : Données nettoyées, validées et enrichies. Les duplications sont éliminées, les schémas sont normalisés, les références sont résolues.

**Couche Gold** : Données agrégées et modélisées pour la consommation métier. Métriques précalculées, dimensions denormalisées, vues optimisées pour les cas d'usage analytiques.

```
┌─────────────────────────────────────────────────────────────────┐
│                 MODÈLE MEDALLION STREAMING                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌────────────────┐                                            │
│   │ Sources Kafka  │                                            │
│   │ (Topics Bruts) │                                            │
│   └───────┬────────┘                                            │
│           │                                                     │
│           ▼                                                     │
│   ┌────────────────────────────────────────┐                    │
│   │            COUCHE BRONZE               │                    │
│   │  • Tables Iceberg Raw                  │                    │
│   │  • Données brutes sans transformation  │                    │
│   │  • Partitionnement par heure           │                    │
│   │  • Rétention longue (années)           │                    │
│   └───────────────┬────────────────────────┘                    │
│                   │ Flink: Nettoyage + Validation               │
│                   ▼                                             │
│   ┌────────────────────────────────────────┐                    │
│   │            COUCHE SILVER               │                    │
│   │  • Tables Iceberg Curated              │                    │
│   │  • Données validées et enrichies       │                    │
│   │  • Schémas normalisés                  │                    │
│   │  • Rétention moyenne (mois)            │                    │
│   └───────────────┬────────────────────────┘                    │
│                   │ Flink: Agrégation + Modélisation            │
│                   ▼                                             │
│   ┌────────────────────────────────────────┐                    │
│   │             COUCHE GOLD                │                    │
│   │  • Tables Iceberg Analytics            │                    │
│   │  • Métriques précalculées              │                    │
│   │  • Vues métier optimisées              │                    │
│   │  • Accès BI / ML direct                │                    │
│   └────────────────────────────────────────┘                    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

L'avantage du streaming dans ce modèle est la mise à jour continue de chaque couche. Contrairement à l'approche batch où les couches Silver et Gold sont recalculées périodiquement, le streaming permet une propagation quasi instantanée des changements à travers toutes les couches.

### Gestion du Change Data Capture (CDC)

Le Change Data Capture représente un pattern fondamental pour alimenter le Streaming Lakehouse depuis les bases de données transactionnelles. Debezium, le standard open source pour le CDC, capture les modifications (INSERT, UPDATE, DELETE) depuis les journaux de transaction des bases de données et les publie comme événements Kafka.

L'intégration CDC-Iceberg requiert une attention particulière au format des opérations. Les événements Debezium contiennent non seulement la nouvelle valeur mais aussi l'ancienne valeur et le type d'opération. Apache Iceberg supporte nativement les opérations d'upsert via le mode Merge-on-Read, permettant de matérialiser fidèlement l'état des tables source.

```yaml
# Configuration Debezium pour PostgreSQL vers Kafka
# debezium-postgres-config.yaml
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaConnector
metadata:
  name: postgres-cdc-connector
spec:
  class: io.debezium.connector.postgresql.PostgresConnector
  tasksMax: 1
  config:
    database.hostname: postgres-primary.internal
    database.port: 5432
    database.user: debezium
    database.password: ${secrets:cdc-password}
    database.dbname: production
    database.server.name: prod-db
    
    # Tables à capturer
    table.include.list: public.customers,public.orders,public.products
    
    # Configuration du format Avro
    key.converter: io.confluent.connect.avro.AvroConverter
    key.converter.schema.registry.url: http://schema-registry:8081
    value.converter: io.confluent.connect.avro.AvroConverter
    value.converter.schema.registry.url: http://schema-registry:8081
    
    # Options CDC
    publication.autocreate.mode: filtered
    slot.name: debezium_prod
    plugin.name: pgoutput
    
    # Transformations pour extraction des valeurs
    transforms: unwrap
    transforms.unwrap.type: io.debezium.transforms.ExtractNewRecordState
    transforms.unwrap.drop.tombstones: false
    transforms.unwrap.delete.handling.mode: rewrite
```

La matérialisation des flux CDC en tables Iceberg avec support des opérations DELETE utilise le mode upsert de Flink :

```sql
-- Table Iceberg configurée pour CDC avec upsert
CREATE TABLE iceberg_customers (
    customer_id STRING,
    email STRING,
    full_name STRING,
    created_at TIMESTAMP(3),
    updated_at TIMESTAMP(3),
    is_deleted BOOLEAN,
    PRIMARY KEY (customer_id) NOT ENFORCED
) WITH (
    'connector' = 'iceberg',
    'catalog-name' = 'lakehouse',
    'database-name' = 'crm',
    'table-name' = 'customers',
    'write.upsert.enabled' = 'true',
    'write.format.default' = 'parquet',
    'write.merge-mode' = 'merge-on-read'
);

-- Ingestion CDC avec gestion des suppressions logiques
INSERT INTO iceberg_customers
SELECT 
    customer_id,
    email,
    full_name,
    created_at,
    CURRENT_TIMESTAMP AS updated_at,
    __deleted AS is_deleted
FROM cdc_customers_kafka
WHERE customer_id IS NOT NULL;
```

> **Migration**  
> *De* : Réplication batch nocturne depuis PostgreSQL  
> *Vers* : CDC temps réel avec Debezium vers Streaming Lakehouse  
> *Stratégie* : 
> 1. Déployer Debezium en mode snapshot pour la capture initiale
> 2. Basculer en mode streaming après synchronisation complète
> 3. Maintenir la réplication batch en parallèle pendant validation (2-4 semaines)
> 4. Décommissionner le batch après période de stabilisation

### Architecture Delta et Streamhouse

Au-delà du Streaming Lakehouse, d'autres évolutions architecturales méritent attention. L'architecture Delta, portée par Databricks, utilise des techniques de micro-batching pour unifier batch et streaming dans un modèle de flux continu. Elle organise les données en couches Bronze (données brutes), Silver (données nettoyées) et Gold (données agrégées), chacune représentée par des tables Delta Lake.

Plus récemment, le concept de Streamhouse émerge comme une évolution du Lakehouse vers le streaming natif. Porté notamment par Ververica avec Apache Paimon, ce modèle place le traitement de flux au cœur de l'architecture plutôt qu'en périphérie. Paimon, développé par Alibaba et incubé chez Apache, offre un format de table optimisé pour l'ingestion streaming à haute fréquence avec support natif du CDC.

Ces évolutions confirment la tendance vers l'unification streaming-lakehouse, chaque approche offrant des compromis différents entre latence, coût et complexité.

---

## IV.12.2 Le Rôle de Confluent et Kafka

### Apache Kafka comme Système Nerveux Central

Apache Kafka s'est imposé comme la plateforme de streaming de facto pour l'entreprise moderne. Son modèle de journal distribué immuable offre des caractéristiques uniques qui en font le fondement idéal d'une architecture de Streaming Lakehouse.

**Le découplage producteurs-consommateurs** permet à chaque système de fonctionner à son propre rythme. Les producteurs publient des événements sans se soucier de la disponibilité ou de la capacité des consommateurs. Les consommateurs lisent à leur vitesse, reprenant là où ils s'étaient arrêtés en cas d'interruption.

**La persistance durable** garantit que chaque événement est stocké de manière fiable, avec une rétention configurable pouvant s'étendre sur des mois ou des années. Cette persistance permet le rejeu historique pour le retraitement ou la reconstruction d'état.

**L'ordonnancement garanti** au niveau de la partition assure que les événements sont traités dans l'ordre où ils ont été produits, propriété essentielle pour maintenir la cohérence des états dérivés.

**La scalabilité horizontale** permet d'absorber des volumes croissants en ajoutant simplement des brokers et des partitions, sans interruption de service.

### L'Écosystème Confluent

Confluent, fondée par les créateurs de Kafka, a construit un écosystème complet autour du noyau open source. Pour le Streaming Lakehouse, plusieurs composants sont particulièrement pertinents.

**Confluent Cloud** offre Kafka en tant que service géré avec une séparation calcul-stockage, éliminant la complexité opérationnelle des clusters auto-gérés. Son architecture serverless permet une scalabilité élastique et un modèle de coût basé sur l'utilisation réelle.

**Schema Registry** gère les contrats de données entre producteurs et consommateurs. Il supporte les formats Avro, Protobuf et JSON Schema avec des règles de compatibilité configurables (backward, forward, full). Cette gouvernance des schémas est essentielle pour l'évolution contrôlée des structures de données.

**Apache Flink (intégré)** fournit un moteur de traitement de flux de niveau entreprise directement dans Confluent Cloud. Flink excelle dans les transformations complexes avec état, les jointures de flux, et les fenêtres temporelles.

**Tableflow** représente l'innovation la plus significative pour le Streaming Lakehouse. Cette fonctionnalité, disponible en disponibilité générale depuis mars 2025, permet de matérialiser automatiquement les topics Kafka en tables Apache Iceberg ou Delta Lake.

### Tableflow : La Passerelle vers le Lakehouse

Tableflow élimine la complexité traditionnellement associée à l'alimentation d'un Lakehouse depuis Kafka. Avant Tableflow, les organisations devaient déployer et maintenir des pipelines Kafka Connect, des jobs Spark Streaming, ou des applications Flink dédiées pour convertir les événements Kafka en fichiers Parquet compatibles Iceberg.

L'architecture de Tableflow exploite des innovations dans la couche de stockage Kora de Confluent Cloud. Les segments Kafka sont directement convertis en fichiers Parquet, sans copie intermédiaire ni duplication des données. Un service de matérialisation de métadonnées génère automatiquement les métadonnées Iceberg et les journaux de transactions Delta Lake en s'appuyant sur le Schema Registry.

```python
# Configuration Tableflow via API Confluent Cloud
# Exemple d'activation sur un topic existant

"""
Étape 1: Définir la configuration Tableflow
"""
tableflow_config = {
    "topic_name": "orders.events",
    "table_format": "ICEBERG",
    "catalog_type": "REST",
    "storage": {
        "type": "S3",
        "bucket": "s3://lakehouse-streaming/iceberg/",
        "region": "ca-central-1"
    },
    "schema_evolution": True,
    "compaction": {
        "enabled": True,
        "interval_minutes": 60,
        "target_file_size_mb": 256
    }
}

"""
Étape 2: Activer Tableflow via CLI Confluent
"""
# confluent tableflow enable orders.events \
#   --format iceberg \
#   --catalog-type rest \
#   --storage s3://lakehouse-streaming/iceberg/ \
#   --region ca-central-1

"""
Étape 3: Configuration Trino pour interroger les tables Iceberg
"""
# catalog.properties (trino/etc/catalog/iceberg_confluent.properties)
# connector.name=iceberg
# iceberg.catalog.type=rest
# iceberg.rest-catalog.uri=https://tableflow.confluent.cloud/v1
# iceberg.rest-catalog.security=OAUTH2
# iceberg.rest-catalog.oauth2.credential=<api-key>:<api-secret>
```

Les capacités clés de Tableflow incluent :

**La conversion automatique des schémas** : Les schémas Avro, Protobuf ou JSON enregistrés dans le Schema Registry sont automatiquement convertis en schémas Iceberg compatibles, avec gestion des types et des conversions nécessaires.

**L'évolution de schéma transparente** : Lorsqu'un producteur émet un événement avec un nouveau champ, Tableflow détecte le changement et fait évoluer le schéma de la table Iceberg de manière compatible.

**La compaction automatique** : Les petits fichiers générés par le streaming continu sont régulièrement compactés en fichiers plus grands, optimisés pour les requêtes analytiques.

**L'intégration avec les catalogues** : Tableflow supporte son propre REST Catalog ainsi que l'intégration avec AWS Glue, Snowflake Open Catalog, et Apache Polaris.

### Patterns d'Intégration Kafka-Iceberg

Au-delà de Tableflow, plusieurs patterns permettent d'alimenter un Lakehouse Iceberg depuis Kafka, chacun avec ses compromis.

**Pattern 1 : Kafka Connect avec Iceberg Sink**

Kafka Connect offre une approche mature et flexible pour l'ingestion vers Iceberg. Le connecteur Iceberg Sink, disponible en open source, consomme les messages Kafka et les écrit en tables Iceberg.

```properties
# Configuration Kafka Connect Iceberg Sink
name=iceberg-sink-orders
connector.class=org.apache.iceberg.connect.IcebergSinkConnector
tasks.max=4

# Configuration Kafka
topics=orders.events
key.converter=io.confluent.connect.avro.AvroConverter
value.converter=io.confluent.connect.avro.AvroConverter
key.converter.schema.registry.url=http://schema-registry:8081
value.converter.schema.registry.url=http://schema-registry:8081

# Configuration Iceberg
iceberg.catalog.type=rest
iceberg.catalog.uri=http://nessie:19120/api/v1
iceberg.catalog.warehouse=s3://lakehouse/warehouse
iceberg.tables=default.orders
iceberg.tables.auto-create-enabled=true

# Configuration de commit
iceberg.control.commit.interval-ms=60000
iceberg.control.commit.records=10000
```

Ce pattern convient aux organisations qui disposent déjà d'une expertise Kafka Connect et souhaitent un contrôle fin sur la configuration. Il nécessite cependant la gestion d'un cluster Connect dédié.

**Pattern 2 : Apache Flink avec Iceberg Sink**

Apache Flink offre le plus haut niveau de flexibilité pour les transformations complexes avant l'écriture vers Iceberg. Le Flink Dynamic Iceberg Sink, introduit fin 2025, permet même de router dynamiquement vers plusieurs tables et de gérer l'évolution de schéma sans redémarrage.

```sql
-- Flink SQL : Création d'une table source Kafka
CREATE TABLE kafka_orders (
    order_id STRING,
    customer_id STRING,
    amount DECIMAL(10, 2),
    currency STRING,
    order_time TIMESTAMP(3),
    WATERMARK FOR order_time AS order_time - INTERVAL '5' SECOND
) WITH (
    'connector' = 'kafka',
    'topic' = 'orders.events',
    'properties.bootstrap.servers' = 'kafka:9092',
    'properties.group.id' = 'flink-iceberg-writer',
    'scan.startup.mode' = 'earliest-offset',
    'format' = 'avro-confluent',
    'avro-confluent.url' = 'http://schema-registry:8081'
);

-- Création de la table Iceberg cible
CREATE TABLE iceberg_orders (
    order_id STRING,
    customer_id STRING,
    amount DECIMAL(10, 2),
    currency STRING,
    order_time TIMESTAMP(3),
    processing_time TIMESTAMP(3)
) WITH (
    'connector' = 'iceberg',
    'catalog-name' = 'lakehouse',
    'catalog-type' = 'rest',
    'uri' = 'http://nessie:19120/api/v1',
    'warehouse' = 's3://lakehouse/warehouse',
    'database-name' = 'sales',
    'table-name' = 'orders'
);

-- Pipeline d'ingestion avec enrichissement
INSERT INTO iceberg_orders
SELECT 
    order_id,
    customer_id,
    amount,
    currency,
    order_time,
    CURRENT_TIMESTAMP AS processing_time
FROM kafka_orders
WHERE amount > 0;
```

**Pattern 3 : Tableflow (Zero-ETL)**

Tableflow représente l'approche la plus simple, éliminant tout code ou configuration de pipeline. Les données Kafka sont automatiquement disponibles en format Iceberg sans intervention.

| Pattern | Complexité | Flexibilité | Latence | Cas d'usage |
|---------|------------|-------------|---------|-------------|
| Tableflow | Minimale | Moyenne | ~Minutes | Matérialisation simple |
| Kafka Connect | Moyenne | Élevée | ~Minutes | Intégration existante |
| Flink | Élevée | Maximale | Configurable | Transformations complexes |

### Le Flink Dynamic Iceberg Sink

Une avancée significative de 2025 est le Flink Dynamic Iceberg Sink, décrit en détail dans la documentation Apache Flink. Ce pattern résout un problème fondamental des pipelines d'ingestion traditionnels : leur incapacité à s'adapter aux changements sans redémarrage.

Dans une architecture traditionnelle, chaque topic Kafka nécessite un graphe de traitement (DAG) dédié avec un schéma figé. L'ajout d'un nouveau topic ou la modification d'un schéma requiert la modification du code et le redémarrage du job.

Le Dynamic Iceberg Sink permet au contraire un pipeline unique capable de :

- Écrire vers plusieurs tables Iceberg dynamiquement
- Créer automatiquement de nouvelles tables selon les instructions contenues dans les messages
- Évoluer le schéma des tables existantes en temps réel
- S'adapter aux changements de partitionnement sans interruption

```java
// Configuration du Dynamic Iceberg Sink (exemple simplifié)
DataStream<GenericRecord> kafkaStream = env
    .fromSource(kafkaSource, WatermarkStrategy.noWatermarks(), "Kafka Source");

// Le DynamicIcebergSink route vers les tables selon les métadonnées du message
DynamicIcebergSink<GenericRecord> sink = DynamicIcebergSink.<GenericRecord>builder()
    .catalog(catalogName)
    .tableNameExtractor(record -> extractTableName(record))
    .schemaProvider(record -> extractSchema(record))
    .tableCreationBehavior(TableCreationBehavior.CREATE_IF_NOT_EXISTS)
    .schemaEvolutionBehavior(SchemaEvolutionBehavior.AUTO_EVOLVE)
    .build();

kafkaStream.sinkTo(sink);
```

### Intégration avec le Schema Registry

Le Schema Registry de Confluent joue un rôle central dans le Streaming Lakehouse en assurant la cohérence des contrats de données entre les producteurs Kafka et les tables Iceberg.

Lorsqu'un producteur enregistre un schéma Avro dans le Registry, celui-ci devient la définition canonique de la structure des données. Tableflow et les connecteurs Iceberg utilisent ce schéma pour :

1. **Créer automatiquement** les tables Iceberg avec la structure correspondante
2. **Valider** que les nouveaux messages respectent le contrat
3. **Évoluer** le schéma Iceberg de manière compatible lors des changements
4. **Convertir** les types Avro vers les types Iceberg équivalents

La correspondance des types entre Avro et Iceberg est gérée automatiquement :

| Type Avro | Type Iceberg |
|-----------|--------------|
| string | STRING |
| int | INTEGER |
| long | LONG |
| float | FLOAT |
| double | DOUBLE |
| boolean | BOOLEAN |
| bytes (decimal) | DECIMAL(p, s) |
| long (timestamp-millis) | TIMESTAMP |
| array | LIST |
| map | MAP |
| record | STRUCT |

### Gestion de l'Évolution de Schéma

L'évolution de schéma représente l'un des défis les plus complexes dans un Streaming Lakehouse. Les producteurs évoluent indépendamment des consommateurs, et les tables Iceberg doivent absorber ces changements sans interruption de service.

Le Schema Registry de Confluent offre plusieurs modes de compatibilité qui régissent les évolutions autorisées :

**BACKWARD** (par défaut) : Les nouveaux schémas peuvent lire les données écrites avec les anciens schémas. Autorise l'ajout de champs optionnels et la suppression de champs.

**FORWARD** : Les anciens schémas peuvent lire les données écrites avec les nouveaux schémas. Autorise l'ajout de champs et la suppression de champs optionnels.

**FULL** : Combine BACKWARD et FORWARD. Autorise uniquement l'ajout ou la suppression de champs optionnels.

**NONE** : Aucune vérification de compatibilité. Déconseillé en production.

```python
# Exemple de configuration Schema Registry pour évolution contrôlée
from confluent_kafka.schema_registry import SchemaRegistryClient
from confluent_kafka.schema_registry.avro import AvroSerializer

# Configuration du client Schema Registry
schema_registry_config = {
    'url': 'http://schema-registry:8081',
    'basic.auth.user.info': '<api-key>:<api-secret>'
}

sr_client = SchemaRegistryClient(schema_registry_config)

# Définir la règle de compatibilité pour un sujet
sr_client.set_compatibility(
    subject_name='orders.events-value',
    level='FULL_TRANSITIVE'  # Compatibilité stricte dans les deux sens
)

# Schéma Avro v1
schema_v1 = """
{
  "type": "record",
  "name": "Order",
  "namespace": "com.example.orders",
  "fields": [
    {"name": "order_id", "type": "string"},
    {"name": "customer_id", "type": "string"},
    {"name": "amount", "type": {"type": "bytes", "logicalType": "decimal", "precision": 10, "scale": 2}},
    {"name": "order_time", "type": {"type": "long", "logicalType": "timestamp-millis"}}
  ]
}
"""

# Schéma Avro v2 : Ajout d'un champ optionnel (compatible FULL)
schema_v2 = """
{
  "type": "record",
  "name": "Order",
  "namespace": "com.example.orders",
  "fields": [
    {"name": "order_id", "type": "string"},
    {"name": "customer_id", "type": "string"},
    {"name": "amount", "type": {"type": "bytes", "logicalType": "decimal", "precision": 10, "scale": 2}},
    {"name": "order_time", "type": {"type": "long", "logicalType": "timestamp-millis"}},
    {"name": "currency", "type": ["null", "string"], "default": null}
  ]
}
"""

# Tester la compatibilité avant enregistrement
is_compatible = sr_client.test_compatibility(
    subject_name='orders.events-value',
    schema=schema_v2
)

if is_compatible:
    # Enregistrer le nouveau schéma
    schema_id = sr_client.register_schema(
        subject_name='orders.events-value',
        schema=schema_v2
    )
    print(f"Schéma v2 enregistré avec ID: {schema_id}")
else:
    print("Erreur: Schéma incompatible, évolution rejetée")
```

Lorsqu'un nouveau schéma est enregistré, Tableflow détecte automatiquement le changement et fait évoluer la table Iceberg correspondante. L'opération `ALTER TABLE ... ADD COLUMN` est exécutée de manière atomique, et les nouvelles données sont écrites avec le schéma élargi tandis que les anciennes données restent lisibles avec des valeurs NULL pour les nouveaux champs.

### Intégration avec les Catalogues Externes

Tableflow supporte l'intégration avec plusieurs catalogues pour exposer les tables Iceberg aux moteurs de consommation. Chaque option présente des compromis différents :

**REST Catalog Confluent (intégré)** : Catalogue natif de Tableflow, accessible via l'API REST standard Iceberg. Idéal pour les déploiements centrés sur Confluent Cloud.

**AWS Glue Data Catalog** : Intégration native avec l'écosystème AWS (Athena, EMR, Redshift Spectrum). Recommandé pour les organisations investies dans AWS.

**Snowflake Open Catalog** : Permet l'accès direct depuis Snowflake sans duplication. Idéal pour les utilisateurs Snowflake souhaitant interroger des données streaming.

**Apache Polaris (incubating)** : Catalogue open source avec gouvernance avancée, en cours d'incubation à Apache. Option de neutralité vendeur.

```python
# Configuration multi-catalogue pour Tableflow
# Exemple : Synchronisation vers AWS Glue et Snowflake Open Catalog

tableflow_multi_catalog_config = {
    "topic_name": "orders.events",
    "table_format": "ICEBERG",
    
    # Stockage primaire
    "storage": {
        "type": "S3",
        "bucket": "s3://streaming-lakehouse-ca/iceberg/",
        "region": "ca-central-1"
    },
    
    # Catalogues de synchronisation
    "catalogs": [
        {
            "type": "AWS_GLUE",
            "database": "streaming_lakehouse",
            "table_prefix": "kafka_",
            "aws_region": "ca-central-1",
            "iam_role": "arn:aws:iam::123456789:role/TableflowGlueRole"
        },
        {
            "type": "SNOWFLAKE_OPEN_CATALOG",
            "catalog_name": "streaming_data",
            "schema": "kafka_tables",
            "connection": {
                "account": "org-account.canada-central.azure",
                "warehouse": "STREAMING_WH"
            }
        }
    ],
    
    # Options de synchronisation
    "sync_options": {
        "sync_interval_seconds": 60,
        "sync_on_commit": True,
        "propagate_schema_changes": True
    }
}
```

### Traitement Pré-Lakehouse avec Flink

Avant que les données n'atteignent le Lakehouse via Tableflow ou d'autres mécanismes, Apache Flink permet d'effectuer des transformations sophistiquées. Ce traitement « pré-lakehouse » est particulièrement utile pour :

**L'agrégation en temps réel** : Calculer des métriques sur des fenêtres glissantes avant persistance.

**Le filtrage et routage** : Diriger différents types d'événements vers différentes tables.

**L'enrichissement contextuel** : Joindre avec des données de référence en temps réel.

**La détection d'anomalies** : Identifier et marquer les données suspectes.

```sql
-- Flink SQL : Agrégation temps réel avant persistance Iceberg
-- Calcul de métriques par fenêtre de 5 minutes

-- Table source streaming
CREATE TABLE raw_clickstream (
    user_id STRING,
    session_id STRING,
    page_url STRING,
    action_type STRING,
    device_type STRING,
    event_time TIMESTAMP(3),
    WATERMARK FOR event_time AS event_time - INTERVAL '10' SECOND
) WITH (
    'connector' = 'kafka',
    'topic' = 'clickstream.events',
    'properties.bootstrap.servers' = 'kafka:9092',
    'format' = 'avro-confluent',
    'avro-confluent.url' = 'http://schema-registry:8081',
    'scan.startup.mode' = 'latest-offset'
);

-- Table Iceberg pour les métriques agrégées
CREATE TABLE clickstream_metrics_5min (
    window_start TIMESTAMP(3),
    window_end TIMESTAMP(3),
    device_type STRING,
    page_views BIGINT,
    unique_users BIGINT,
    unique_sessions BIGINT,
    clicks BIGINT,
    avg_session_depth DOUBLE,
    bounce_rate DOUBLE
) WITH (
    'connector' = 'iceberg',
    'catalog-name' = 'lakehouse',
    'database-name' = 'analytics',
    'table-name' = 'clickstream_metrics_5min'
);

-- Pipeline d'agrégation avec fenêtre tumbling
INSERT INTO clickstream_metrics_5min
SELECT 
    TUMBLE_START(event_time, INTERVAL '5' MINUTE) AS window_start,
    TUMBLE_END(event_time, INTERVAL '5' MINUTE) AS window_end,
    device_type,
    COUNT(*) AS page_views,
    COUNT(DISTINCT user_id) AS unique_users,
    COUNT(DISTINCT session_id) AS unique_sessions,
    COUNT(CASE WHEN action_type = 'click' THEN 1 END) AS clicks,
    AVG(CAST(session_depth AS DOUBLE)) AS avg_session_depth,
    CAST(
        COUNT(CASE WHEN session_depth = 1 THEN 1 END) AS DOUBLE
    ) / NULLIF(COUNT(DISTINCT session_id), 0) AS bounce_rate
FROM (
    SELECT 
        *,
        COUNT(*) OVER (
            PARTITION BY session_id 
            ORDER BY event_time 
            ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
        ) AS session_depth
    FROM raw_clickstream
)
GROUP BY 
    TUMBLE(event_time, INTERVAL '5' MINUTE),
    device_type;

-- Table Iceberg pour les sessions individuelles (grain fin)
CREATE TABLE user_sessions (
    session_id STRING,
    user_id STRING,
    device_type STRING,
    session_start TIMESTAMP(3),
    session_end TIMESTAMP(3),
    page_count INT,
    total_duration_seconds INT,
    conversion_flag BOOLEAN,
    PRIMARY KEY (session_id) NOT ENFORCED
) WITH (
    'connector' = 'iceberg',
    'catalog-name' = 'lakehouse',
    'database-name' = 'analytics',
    'table-name' = 'user_sessions',
    'write.upsert.enabled' = 'true'
);

-- Pipeline de sessionisation avec fenêtre de session
INSERT INTO user_sessions
SELECT 
    session_id,
    FIRST_VALUE(user_id) AS user_id,
    FIRST_VALUE(device_type) AS device_type,
    SESSION_START(event_time, INTERVAL '30' MINUTE) AS session_start,
    SESSION_END(event_time, INTERVAL '30' MINUTE) AS session_end,
    CAST(COUNT(*) AS INT) AS page_count,
    CAST(
        TIMESTAMPDIFF(SECOND, 
            SESSION_START(event_time, INTERVAL '30' MINUTE),
            SESSION_END(event_time, INTERVAL '30' MINUTE)
        ) AS INT
    ) AS total_duration_seconds,
    MAX(CASE WHEN action_type = 'purchase' THEN TRUE ELSE FALSE END) AS conversion_flag
FROM raw_clickstream
GROUP BY 
    SESSION(event_time, INTERVAL '30' MINUTE),
    session_id;
```

### Garanties de Livraison et Exactly-Once

La sémantique de livraison « exactly-once » est cruciale pour un Streaming Lakehouse où la cohérence des données analytiques dépend de l'absence de duplications ou de pertes.

**Kafka** offre des garanties exactly-once via les transactions et l'idempotence des producteurs. Chaque message est écrit exactement une fois, même en cas de retry.

**Flink** maintient l'état de manière cohérente via son mécanisme de checkpointing distribué. En cas de défaillance, le traitement reprend depuis le dernier checkpoint avec une cohérence garantie.

**Iceberg** assure l'atomicité des commits. Chaque commit de fichiers est une opération atomique : soit tous les fichiers sont visibles, soit aucun.

La combinaison de ces garanties permet un pipeline end-to-end exactly-once :

```java
// Configuration Flink pour exactly-once avec Iceberg
StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

// Activer le checkpointing pour exactly-once
env.enableCheckpointing(60000, CheckpointingMode.EXACTLY_ONCE);
env.getCheckpointConfig().setMinPauseBetweenCheckpoints(30000);
env.getCheckpointConfig().setCheckpointTimeout(120000);
env.getCheckpointConfig().setMaxConcurrentCheckpoints(1);
env.getCheckpointConfig().setExternalizedCheckpointCleanup(
    CheckpointConfig.ExternalizedCheckpointCleanup.RETAIN_ON_CANCELLATION
);

// Configurer le state backend pour haute disponibilité
env.setStateBackend(new EmbeddedRocksDBStateBackend());
env.getCheckpointConfig().setCheckpointStorage(
    "s3://checkpoints-bucket/flink-checkpoints/"
);

// Configuration Kafka source avec exactly-once
KafkaSource<GenericRecord> kafkaSource = KafkaSource.<GenericRecord>builder()
    .setBootstrapServers("kafka:9092")
    .setTopics("orders.events")
    .setGroupId("flink-iceberg-exactly-once")
    .setStartingOffsets(OffsetsInitializer.committedOffsets(OffsetResetStrategy.EARLIEST))
    .setDeserializer(KafkaRecordDeserializationSchema.of(
        new ConfluentRegistryAvroDeserializationSchema<>(
            GenericRecord.class,
            schemaRegistryUrl
        )
    ))
    .setProperty("isolation.level", "read_committed")  // Important pour exactly-once
    .build();

// Configuration Iceberg sink
FlinkSink.Builder<RowData> sinkBuilder = FlinkSink.forRowData(rowDataStream)
    .tableLoader(tableLoader)
    .overwrite(false)
    .equalityFieldColumns(Arrays.asList("order_id"))  // Clé pour upsert
    .upsert(true);

// Le commit Iceberg est coordonné avec le checkpoint Flink
// garantissant l'exactly-once end-to-end
```

### Considérations de Coûts

L'optimisation des coûts dans un Streaming Lakehouse requiert une compréhension des différents postes de dépenses et de leurs leviers.

**Coûts Kafka/Confluent** :
- Débit d'ingestion (par Go ingéré)
- Rétention (par Go-heure stocké)
- Connecteurs et Flink (par UCU - Confluent Unit Compute)

**Coûts Stockage Objet (S3/ADLS/GCS)** :
- Stockage (par Go/mois)
- Requêtes (par millier de requêtes PUT/GET)
- Transfert inter-régions (par Go transféré)

**Coûts Calcul (Flink, Trino, etc.)** :
- Instances de traitement (par vCPU-heure)
- Mémoire (par Go-heure)

| Composant | Coût mensuel estimé (100 To, 1M msg/s) |
|-----------|----------------------------------------|
| Confluent Cloud (Kafka + Flink) | 15 000 - 25 000 $ CAD |
| Stockage S3 (ca-central-1) | 2 300 $ CAD |
| Requêtes S3 | 500 - 1 500 $ CAD |
| Trino/Dremio (si utilisé) | 3 000 - 8 000 $ CAD |
| **Total estimé** | **20 800 - 36 800 $ CAD** |

Stratégies d'optimisation :

1. **Compaction agressive** : Réduire le nombre de fichiers diminue les coûts de requêtes S3
2. **Tiering de rétention** : Données récentes dans Kafka, historique dans Iceberg
3. **Compression optimale** : Zstandard offre le meilleur ratio compression/performance
4. **Partitionnement intelligent** : Aligner avec les patterns de requête pour maximiser le pruning

### Considérations de Performance

L'optimisation des performances dans un Streaming Lakehouse requiert une attention particulière à plusieurs paramètres.

**La fréquence de commit** détermine le compromis entre latence et efficacité. Des commits fréquents (toutes les minutes) offrent une fraîcheur maximale mais génèrent de nombreux petits fichiers nécessitant une compaction agressive. Des commits moins fréquents (toutes les heures) produisent des fichiers mieux dimensionnés mais augmentent la latence.

**Le parallélisme d'écriture** influence le débit d'ingestion. Augmenter le nombre de tâches (partitions Kafka, tâches Flink, workers Connect) améliore le débit au prix d'une multiplication des fichiers générés.

**La stratégie de partitionnement** Iceberg doit aligner avec les patterns de requête. Un partitionnement par date (heure ou jour) convient à la majorité des cas d'usage analytiques.

> **Performance**  
> Configuration recommandée pour un Streaming Lakehouse à haut débit :
> - Intervalle de commit : 5-15 minutes pour les tables analytiques, 1 minute pour les cas d'usage temps réel critiques
> - Compaction : Activer la compaction automatique avec un seuil de 10-20 fichiers par partition
> - Taille cible des fichiers : 128-256 Mo pour équilibrer performance de requête et overhead de métadonnées
> - Write mode : Utiliser Merge-on-Read pour les cas CDC/upsert, Copy-on-Write pour l'append-only

### Observabilité du Pipeline Streaming-Lakehouse

La surveillance d'un Streaming Lakehouse nécessite une observabilité de bout en bout, du producteur Kafka jusqu'aux requêtes analytiques sur Iceberg.

```yaml
# Configuration Prometheus pour métriques Streaming Lakehouse
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  # Métriques Kafka (JMX Exporter)
  - job_name: 'kafka-brokers'
    static_configs:
      - targets: ['kafka-1:9090', 'kafka-2:9090', 'kafka-3:9090']
    
  # Métriques Consumer Lag
  - job_name: 'kafka-consumer-lag'
    static_configs:
      - targets: ['kafka-lag-exporter:8080']
    
  # Métriques Flink
  - job_name: 'flink-jobmanager'
    static_configs:
      - targets: ['flink-jobmanager:9249']
      
  - job_name: 'flink-taskmanagers'
    static_configs:
      - targets: ['flink-tm-1:9249', 'flink-tm-2:9249']
    
  # Métriques Iceberg Catalog (Nessie)
  - job_name: 'nessie-catalog'
    static_configs:
      - targets: ['nessie:9091']
```

Les métriques critiques à surveiller incluent :

```yaml
# Règles d'alerte pour Streaming Lakehouse
# alerting_rules.yml
groups:
  - name: streaming_lakehouse_alerts
    rules:
      # Consumer lag critique
      - alert: KafkaConsumerLagCritical
        expr: kafka_consumer_group_lag > 500000
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "Retard consommateur critique"
          description: "Le groupe {{ $labels.group }} a un retard de {{ $value }} messages sur {{ $labels.topic }}"
          
      # Échec de commit Iceberg
      - alert: IcebergCommitFailure
        expr: increase(iceberg_commit_failures_total[5m]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Échec de commit Iceberg"
          description: "Des commits Iceberg échouent, vérifier les logs"
          
      # Prolifération de petits fichiers
      - alert: IcebergSmallFilesProliferation
        expr: iceberg_table_files_count > 500 AND iceberg_table_avg_file_size_mb < 50
        for: 2h
        labels:
          severity: warning
        annotations:
          summary: "Compaction nécessaire"
          description: "La table {{ $labels.table }} contient {{ $value }} fichiers de taille moyenne {{ $labels.avg_size }}MB"
          
      # Fraîcheur des données
      - alert: DataFreshnessLag
        expr: (time() - iceberg_table_last_commit_timestamp) > 3600
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "Données non fraîches"
          description: "La table {{ $labels.table }} n'a pas reçu de commit depuis plus d'une heure"
```

### L'Approche Lakehouse-Native avec Ursa

Une approche émergente mérite mention : les moteurs de streaming « lakehouse-native ». Le projet Ursa, développé par StreamNative et récompensé du Best Industry Paper à VLDB 2025, représente cette nouvelle génération.

Ursa écrit les données directement en format Iceberg ou Delta Lake sur stockage objet, éliminant la distinction entre stockage Kafka et stockage analytique. Chaque topic est simultanément un flux d'événements consommable via l'API Kafka et une table Iceberg interrogeable via SQL. Cette dualité simplifie radicalement l'architecture en éliminant la nécessité de pipelines de synchronisation.

L'architecture d'Ursa utilise un journal d'écriture anticipée (WAL) sur stockage objet pour les données récentes avec latence minimale, tandis qu'un service de compaction en arrière-plan convertit progressivement ces données en fichiers Parquet organisés selon les formats de table ouverts.

Cette approche offre une réduction potentielle des coûts jusqu'à 10× par rapport à un cluster Kafka traditionnel, principalement grâce à l'élimination de la réplication triple et de la duplication streaming/lakehouse.

### Considérations pour le Contexte Canadien

Les organisations canadiennes adoptant le Streaming Lakehouse doivent tenir compte de considérations spécifiques liées à la réglementation, à la géographie et à l'écosystème technologique local.

**La souveraineté des données** requiert que les données sensibles restent dans des régions canadiennes. AWS offre les régions Canada (Montréal) ca-central-1 et Canada (Calgary) ca-west-1, tandis qu'Azure propose Canada Centre (Toronto) et Canada Est (Québec). Google Cloud dispose de la région northamerica-northeast1 (Montréal) et northamerica-northeast2 (Toronto). Confluent Cloud supporte le déploiement dans ces régions, et le stockage Iceberg doit être configuré sur des buckets S3 ou conteneurs Azure situés au Canada.

**La conformité Loi 25** (anciennement Projet de loi 64) impose des exigences strictes sur le consentement, la transparence et le droit à l'effacement des données personnelles des résidents québécois. Un Streaming Lakehouse bien conçu facilite la conformité grâce à :

- La traçabilité complète offerte par le journal Kafka (qui a accédé à quoi, quand)
- Le support natif des opérations DELETE par Iceberg pour le droit à l'effacement
- Les capacités de time travel permettant de démontrer l'état des données à un moment donné
- La gouvernance centralisée des schémas documentant la structure des données personnelles

```python
# Implémentation du droit à l'effacement (Loi 25, Article 27)
# Pipeline de suppression propagée Kafka -> Iceberg

from dataclasses import dataclass
from typing import List
import json

@dataclass
class ErasureRequest:
    request_id: str
    subject_identifier: str  # Identifiant de la personne concernée
    identifier_type: str     # email, customer_id, etc.
    requested_at: str
    tables_to_process: List[str]
    
class ErasurePipeline:
    """
    Pipeline de conformité Loi 25 pour le droit à l'effacement.
    Coordonne la suppression entre Kafka et Iceberg.
    """
    
    def __init__(self, kafka_admin, iceberg_catalog, audit_logger):
        self.kafka_admin = kafka_admin
        self.iceberg_catalog = iceberg_catalog
        self.audit_logger = audit_logger
        
    def process_erasure_request(self, request: ErasureRequest):
        """
        Traite une demande d'effacement conformément à la Loi 25.
        Délai légal : 30 jours.
        """
        # 1. Journaliser la demande pour audit
        self.audit_logger.log_erasure_request(
            request_id=request.request_id,
            subject=request.subject_identifier,
            timestamp=request.requested_at
        )
        
        results = {}
        
        # 2. Supprimer des tables Iceberg
        for table_name in request.tables_to_process:
            table = self.iceberg_catalog.load_table(table_name)
            
            # Identifier la colonne correspondant à l'identifiant
            id_column = self._get_identifier_column(
                table, 
                request.identifier_type
            )
            
            # Exécuter la suppression
            delete_count = self._execute_delete(
                table=table,
                column=id_column,
                value=request.subject_identifier
            )
            
            results[table_name] = {
                'deleted_rows': delete_count,
                'timestamp': datetime.utcnow().isoformat()
            }
            
            # Journaliser pour audit
            self.audit_logger.log_table_erasure(
                request_id=request.request_id,
                table=table_name,
                rows_deleted=delete_count
            )
        
        # 3. Produire un tombstone sur les topics Kafka pertinents
        self._produce_tombstones(
            subject_identifier=request.subject_identifier,
            identifier_type=request.identifier_type
        )
        
        # 4. Générer le certificat de suppression
        certificate = self._generate_erasure_certificate(
            request=request,
            results=results
        )
        
        return certificate
    
    def _execute_delete(self, table, column, value):
        """
        Exécute une suppression SQL sur la table Iceberg.
        Utilise le DELETE natif d'Iceberg (v2).
        """
        # Via Spark SQL ou Flink
        delete_sql = f"""
            DELETE FROM {table.name()}
            WHERE {column} = '{value}'
        """
        return self.spark.sql(delete_sql).count()
    
    def _produce_tombstones(self, subject_identifier, identifier_type):
        """
        Produit des tombstones Kafka pour les topics contenant
        les données de la personne concernée.
        """
        topics = self._identify_affected_topics(identifier_type)
        
        for topic in topics:
            # Le tombstone (valeur null avec la clé) permet
            # la compaction du log Kafka
            self.kafka_producer.send(
                topic=topic,
                key=subject_identifier,
                value=None  # Tombstone
            )
```

**Les coûts de transfert inter-régions** peuvent s'accumuler rapidement dans une architecture distribuée. L'optimisation consiste à colocater les composants Kafka, Flink et le stockage Iceberg dans la même région pour minimiser les frais d'egress. Pour les organisations multi-régionales (ex: Québec et Ontario), une architecture active-passive avec réplication asynchrone peut être plus économique qu'une distribution active-active.

**L'écosystème local** inclut plusieurs fournisseurs de services gérés avec présence canadienne :

| Fournisseur | Service | Régions Canada |
|-------------|---------|----------------|
| Confluent | Confluent Cloud | ca-central-1 (Montréal) |
| Aiven | Aiven for Apache Kafka | Toronto, Montréal |
| Amazon | Amazon MSK | ca-central-1, ca-west-1 |
| Microsoft | Azure Event Hubs | Canada Central, Canada East |
| Starburst | Starburst Galaxy | Canada |

> **Étude de cas : Mouvement Desjardins**  
> *Secteur* : Services financiers coopératifs  
> *Défi* : Unifier les données de 7 millions de membres provenant de multiples systèmes (bancaires, assurances, valeurs mobilières) pour une vue client 360° temps réel, tout en respectant les exigences strictes de l'AMF et de la Loi 25  
> *Solution* : Streaming Lakehouse avec Kafka hébergé au Québec (ca-central-1), Apache Flink pour l'agrégation temps réel des profils clients, et Apache Iceberg pour l'historique analytique. CDC Debezium depuis les systèmes legacy AS/400 et Oracle.  
> *Architecture* :  
> - 47 topics Kafka pour les événements temps réel (transactions, connexions, appels)  
> - 3 clusters Flink pour les différents domaines (bancaire, assurance, investissement)  
> - REST Catalog Nessie avec gouvernance centralisée  
> - Stockage Iceberg sur S3 ca-central-1 avec chiffrement KMS  
> *Résultats* :  
> - Latence de mise à jour du profil client réduite de 24 heures à 3 minutes  
> - Conformité Loi 25 assurée par traçabilité complète et capacité d'effacement  
> - Réduction de 45 % des coûts par élimination des ETL batch nocturnes  
> - Temps de réponse aux demandes d'accès de l'AMF réduit de 5 jours à 4 heures

> **Étude de cas : Bell Canada**  
> *Secteur* : Télécommunications  
> *Défi* : Traiter en temps réel les données de réseau et d'utilisation de 23 millions d'abonnés pour la détection proactive des pannes, l'optimisation de l'expérience client, et la prévention de la fraude  
> *Solution* : Architecture Kappa évoluée vers Streaming Lakehouse avec :  
> - Ingestion temps réel des métriques réseau (15 millions d'événements/seconde en pointe) via Kafka  
> - Apache Flink pour la détection d'anomalies avec modèles ML embarqués  
> - Persistance Iceberg pour l'analyse historique et l'entraînement des modèles  
> - Intégration Power BI via Dremio pour les tableaux de bord opérationnels  
> *Architecture multi-régions* :  
> - Cluster Kafka primaire : Toronto (Canada Central)  
> - Réplica DR : Montréal (Canada East)  
> - Latence de réplication : < 100ms  
> *Résultats* :  
> - Temps moyen de détection des incidents réseau réduit de 45 minutes à 90 secondes  
> - Capacité d'analyse historique sur 2 ans de données réseau (850 To)  
> - Détection de fraude SIM swap améliorée de 78 %  
> - Économies de 3,2 M$ CAD/an en coûts d'infrastructure

> **Étude de cas : Loblaws Digital**  
> *Secteur* : Commerce de détail alimentaire  
> *Défi* : Personnaliser l'expérience client sur les plateformes numériques (PC Optimum, click & collect) en temps réel basé sur le comportement d'achat et les préférences  
> *Solution* : Streaming Lakehouse alimentant un moteur de recommandation temps réel :  
> - Événements de navigation et d'achat capturés via Kafka  
> - Calcul de features temps réel avec Flink (panier moyen, catégories favorites, sensibilité prix)  
> - Feature Store matérialisé dans Iceberg pour l'entraînement ML  
> - Modèles de recommandation servis via API avec latence < 50ms  
> *Résultats* :  
> - Taux de conversion des recommandations augmenté de 23 %  
> - Valeur moyenne du panier en hausse de 8 %  
> - Temps d'entraînement des modèles réduit de 4 heures à 45 minutes grâce aux features pré-calculées

### Architecture de Référence pour le Canada

L'architecture suivante représente un déploiement de référence pour une organisation canadienne adoptant le Streaming Lakehouse :

```
┌─────────────────────────────────────────────────────────────────────────┐
│              STREAMING LAKEHOUSE - ARCHITECTURE CANADA                  │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  RÉGION PRIMAIRE (ca-central-1 / Montréal)                             │
│  ┌───────────────────────────────────────────────────────────────────┐ │
│  │                                                                   │ │
│  │  ┌─────────────┐   ┌─────────────┐   ┌─────────────────────────┐ │ │
│  │  │  Sources    │──▶│   Kafka     │──▶│      Apache Flink       │ │ │
│  │  │  (CDC, App) │   │  Confluent  │   │   (Traitement Temps    │ │ │
│  │  └─────────────┘   │   Cloud     │   │        Réel)           │ │ │
│  │                    │             │   └───────────┬─────────────┘ │ │
│  │                    │  + Schema   │               │               │ │
│  │                    │   Registry  │               │               │ │
│  │                    └──────┬──────┘               │               │ │
│  │                           │                      │               │ │
│  │                           │ Tableflow            │               │ │
│  │                           ▼                      ▼               │ │
│  │                    ┌─────────────────────────────────────────┐   │ │
│  │                    │           Apache Iceberg                │   │ │
│  │                    │      (S3 ca-central-1, chiffré KMS)     │   │ │
│  │                    │                                         │   │ │
│  │                    │  REST Catalog (Nessie ou Confluent)     │   │ │
│  │                    └─────────────────────────────────────────┘   │ │
│  │                                      │                           │ │
│  │                    ┌─────────────────┼─────────────────┐         │ │
│  │                    │                 │                 │         │ │
│  │                    ▼                 ▼                 ▼         │ │
│  │             ┌───────────┐     ┌───────────┐     ┌───────────┐   │ │
│  │             │  Dremio   │     │ Power BI  │     │ ML/AI     │   │ │
│  │             │  (Query)  │     │(Direct    │     │(SageMaker │   │ │
│  │             │           │     │ Lake)     │     │ Vertex)   │   │ │
│  │             └───────────┘     └───────────┘     └───────────┘   │ │
│  │                                                                   │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                                                                         │
│  RÉGION DR (ca-west-1 / Calgary)                                       │
│  ┌───────────────────────────────────────────────────────────────────┐ │
│  │                                                                   │ │
│  │  ┌─────────────────┐        ┌─────────────────────────────────┐  │ │
│  │  │  Kafka Replica  │◀──────│  S3 Cross-Region Replication   │  │ │
│  │  │  (Cluster Link) │        │  (Iceberg métadonnées + data)  │  │ │
│  │  └─────────────────┘        └─────────────────────────────────┘  │ │
│  │                                                                   │ │
│  │  RTO: 15 minutes | RPO: 5 minutes                                │ │
│  │                                                                   │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                                                                         │
│  GOUVERNANCE ET CONFORMITÉ                                             │
│  ┌───────────────────────────────────────────────────────────────────┐ │
│  │  • Loi 25 : Consentement, Effacement, Portabilité                │ │
│  │  • LPRPDE : Protection des renseignements personnels              │ │
│  │  • SOX (si applicable) : Intégrité et audit financier            │ │
│  │  • BSIF (institutions financières) : Résilience opérationnelle   │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

Cette architecture assure :

- **Souveraineté des données** : Toutes les données restent dans les régions canadiennes
- **Haute disponibilité** : Réplication entre Montréal et Calgary
- **Conformité** : Traçabilité, audit et capacités d'effacement
- **Performance** : Latence minimale grâce à la colocation des composants
- **Évolutivité** : Scaling indépendant de chaque couche

### Feuille de Route pour la Migration

La transition vers un Streaming Lakehouse depuis une architecture Lambda ou des pipelines batch traditionnels requiert une approche méthodique en phases. L'objectif est de minimiser les risques tout en démontrant rapidement la valeur.

**Phase 1 : Fondations (Mois 1-3)**

Cette phase établit l'infrastructure de base sans perturber les systèmes existants.

*Objectifs* :
- Déployer un cluster Kafka ou Confluent Cloud dans une région canadienne
- Établir le Schema Registry avec les premiers schémas Avro
- Configurer le stockage Iceberg sur S3/ADLS avec un REST Catalog
- Former l'équipe aux concepts fondamentaux

*Livrables* :
- Environnement de développement fonctionnel
- Premier topic Kafka alimenté par une source non critique
- Première table Iceberg matérialisée (via Tableflow ou Kafka Connect)
- Documentation des standards et conventions

*Métriques de succès* :
- Latence de bout en bout < 5 minutes
- Zéro perte de données sur un mois de production test

**Phase 2 : Cas d'Usage Pilote (Mois 4-6)**

Cette phase déploie un premier cas d'usage en production parallèlement aux systèmes existants.

*Sélection du cas pilote* : Choisir un cas d'usage présentant les caractéristiques suivantes :
- Haute valeur métier mais risque limité en cas d'échec
- Latence actuelle insatisfaisante (typiquement T+1)
- Volume gérable (< 100 Go/jour, < 50 000 événements/seconde)
- Équipe métier engagée et disponible pour valider

*Exemples de cas pilotes recommandés* :
- Tableau de bord temps réel pour les indicateurs opérationnels
- Détection d'anomalies sur un flux de transactions secondaire
- Alimentation d'un Feature Store pour un modèle ML existant

*Architecture parallèle* : Durant cette phase, le nouveau pipeline Streaming Lakehouse fonctionne en parallèle avec l'existant. Les consommateurs peuvent comparer les résultats des deux systèmes pour valider la cohérence avant basculement.

```
┌───────────────────────────────────────────────────────────────┐
│                  ARCHITECTURE DE MIGRATION PARALLÈLE          │
├───────────────────────────────────────────────────────────────┤
│                                                               │
│  Sources  ───┬───▶ [Pipeline Lambda Existant] ───▶ Résultats │
│  de          │                                       Batch    │
│  Données     │                                                │
│              │                                                │
│              └───▶ [Nouveau Streaming Lakehouse] ──▶ Résultats│
│                         Kafka → Flink → Iceberg    Temps Réel │
│                                                               │
│                              ▼                                │
│                    [Comparaison et Validation]                │
│                                                               │
└───────────────────────────────────────────────────────────────┘
```

*Livrables* :
- Pipeline Streaming Lakehouse complet pour le cas pilote
- Tableaux de bord d'observabilité
- Runbooks opérationnels
- Formation des équipes de support

**Phase 3 : Extension et Consolidation (Mois 7-12)**

Cette phase étend le Streaming Lakehouse à des cas d'usage additionnels et commence le décommissionnement des systèmes legacy.

*Objectifs* :
- Migrer 3 à 5 cas d'usage supplémentaires
- Intégrer les sources CDC pour les systèmes transactionnels
- Établir la gouvernance centralisée (catalogage, lignage, accès)
- Décommissionner les premiers pipelines batch redondants

*Considérations critiques* :
- Maintenir une période de fonctionnement parallèle d'au moins 3 mois avant décommissionnement
- Documenter les différences sémantiques entre ancien et nouveau système
- Planifier la migration des consommateurs en coordination avec les équipes métier

**Phase 4 : Optimisation et Maturité (Année 2)**

Cette phase finalise la transition et optimise les coûts et performances.

*Objectifs* :
- Décommissionner l'architecture Lambda complète
- Implémenter les patterns avancés (Shift Left, CDC complet)
- Optimiser les coûts (compaction, tiering, réservations)
- Établir les pratiques MLOps intégrées

> **Migration**  
> *De* : Architecture Lambda avec ETL batch nocturnes  
> *Vers* : Streaming Lakehouse unifié Kafka-Iceberg  
> *Stratégie* : Migration progressive en 4 phases sur 12-18 mois. Fonctionnement parallèle obligatoire avant tout décommissionnement. Budget de contingence de 20 % pour les imprévus techniques.

### Résolution des Problèmes Courants

L'exploitation d'un Streaming Lakehouse présente des défis opérationnels spécifiques. Cette section documente les problèmes les plus fréquents et leurs solutions.

**Problème : Consumer Lag Croissant**

*Symptômes* : Le retard entre la production et la consommation des messages Kafka augmente progressivement, indiquant que les consommateurs ne parviennent pas à suivre le rythme d'ingestion.

*Causes possibles* :
- Ressources de calcul insuffisantes (tâches Flink sous-dimensionnées)
- Backpressure dans le pipeline Flink dû à des opérations coûteuses
- Commits Iceberg trop fréquents créant des goulots d'étranglement

*Diagnostic* :
```bash
# Vérifier le consumer lag
kafka-consumer-groups.sh --bootstrap-server kafka:9092 \
  --describe --group flink-streaming-lakehouse

# Identifier le backpressure Flink (si disponible)
curl http://flink-jobmanager:8081/jobs/<job-id>/vertices/<vertex-id>/backpressure
```

*Solutions* :
- Augmenter le parallélisme Flink (`SET 'parallelism.default' = '16';`)
- Optimiser les opérations coûteuses (éviter les aggregations non-bornées)
- Réduire la fréquence de checkpoint si elle est excessive
- Augmenter l'intervalle de commit Iceberg (5-15 minutes au lieu de 1 minute)

**Problème : Prolifération de Petits Fichiers**

*Symptômes* : Les performances de requête sur les tables Iceberg se dégradent progressivement. Le nombre de fichiers par partition dépasse plusieurs centaines.

*Causes possibles* :
- Commits trop fréquents générant de nombreux fichiers sous-dimensionnés
- Parallélisme d'écriture élevé combiné à un faible volume
- Compaction désactivée ou mal configurée

*Diagnostic* :
```sql
-- Analyser la distribution des fichiers
SELECT 
    partition,
    COUNT(*) as file_count,
    SUM(file_size_in_bytes) / 1024 / 1024 as total_size_mb,
    AVG(file_size_in_bytes) / 1024 / 1024 as avg_file_size_mb
FROM prod.lakehouse.orders.files
GROUP BY partition
HAVING file_count > 100
ORDER BY file_count DESC;
```

*Solutions* :
- Activer la compaction automatique avec des seuils appropriés
- Augmenter l'intervalle entre les commits
- Réduire le parallélisme d'écriture si le volume est faible
- Exécuter une compaction manuelle pour les tables problématiques

```sql
-- Compaction manuelle (Spark)
CALL prod.system.rewrite_data_files(
    table => 'lakehouse.orders',
    options => map('target-file-size-bytes', '268435456')  -- 256 MB
);
```

**Problème : Échecs de Checkpoint Flink**

*Symptômes* : Les checkpoints Flink échouent régulièrement, provoquant des retours en arrière et des duplications potentielles.

*Causes possibles* :
- Timeout de checkpoint trop court pour le volume d'état
- State backend mal configuré ou saturé
- Problèmes de connectivité avec le stockage des checkpoints

*Solutions* :
- Augmenter le timeout de checkpoint (`execution.checkpointing.timeout: 10min`)
- Activer les checkpoints incrémentaux pour RocksDB
- Vérifier la latence vers le stockage S3/ADLS
- Réduire la taille de l'état en utilisant des TTL sur les états non nécessaires

```java
// Configuration du TTL pour les états Flink
StateTtlConfig ttlConfig = StateTtlConfig
    .newBuilder(Time.hours(24))  // Expiration après 24h d'inactivité
    .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
    .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
    .build();

ValueStateDescriptor<UserState> stateDescriptor = 
    new ValueStateDescriptor<>("user-state", UserState.class);
stateDescriptor.enableTimeToLive(ttlConfig);
```

**Problème : Divergence de Schéma entre Kafka et Iceberg**

*Symptômes* : Les écritures vers Iceberg échouent avec des erreurs de schéma incompatible, bien que le Schema Registry accepte les évolutions.

*Causes possibles* :
- Évolution de schéma dans le Registry non propagée vers Iceberg
- Types incompatibles entre Avro et Iceberg
- Champs obligatoires ajoutés sans valeur par défaut

*Solutions* :
- Vérifier que l'évolution de schéma respecte les règles de compatibilité Iceberg
- Utiliser FULL ou FULL_TRANSITIVE pour une compatibilité stricte
- Toujours définir des valeurs par défaut pour les nouveaux champs
- Recréer le connecteur après évolution majeure si nécessaire

**Problème : Latence de Requête Élevée**

*Symptômes* : Les requêtes analytiques sur les tables Iceberg prennent plusieurs minutes au lieu des secondes attendues.

*Causes possibles* :
- Partitionnement inadapté aux patterns de requête
- Absence de pruning sur les prédicats de filtre
- Trop de fichiers nécessitant un scan complet
- Métriques de colonnes non maintenues

*Solutions* :
- Analyser le plan d'exécution pour identifier les scans complets
- Revoir la stratégie de partitionnement
- Activer les statistiques de colonnes
- Considérer le tri (clustering) sur les colonnes de filtre fréquentes

```sql
-- Ajouter le tri sur les colonnes fréquemment filtrées (Spark)
ALTER TABLE prod.lakehouse.orders 
WRITE ORDERED BY customer_id, order_date;

-- Réécrire avec le nouveau tri
CALL prod.system.rewrite_data_files(
    table => 'lakehouse.orders',
    strategy => 'sort'
);
```

### Gestion des Données Tardives

Dans un système de streaming, les données arrivent parfois avec un retard significatif par rapport à leur timestamp d'événement. Cette situation, fréquente dans les architectures IoT ou mobiles, nécessite une gestion explicite pour maintenir l'intégrité analytique.

Apache Flink utilise le concept de watermark pour délimiter la frontière entre les données attendues et les données tardives. Le watermark représente une assertion que tous les événements avec un timestamp inférieur sont arrivés (ou devraient être considérés comme tels).

```sql
-- Configuration du watermark avec tolérance pour les retards
CREATE TABLE mobile_events (
    event_id STRING,
    user_id STRING,
    event_type STRING,
    event_time TIMESTAMP(3),
    device_info STRING,
    -- Watermark avec tolérance de 5 minutes pour les événements mobiles
    WATERMARK FOR event_time AS event_time - INTERVAL '5' MINUTE
) WITH (
    'connector' = 'kafka',
    'topic' = 'mobile.events',
    'properties.bootstrap.servers' = 'kafka:9092',
    'format' = 'avro-confluent',
    'avro-confluent.url' = 'http://schema-registry:8081'
);

-- Stratégie pour les données extrêmement tardives
-- Option 1 : Table séparée pour les données tardives
CREATE TABLE late_mobile_events (
    event_id STRING,
    user_id STRING,
    event_type STRING,
    event_time TIMESTAMP(3),
    arrival_time TIMESTAMP(3) DEFAULT PROCTIME(),
    PRIMARY KEY (event_id) NOT ENFORCED
) WITH (
    'connector' = 'iceberg',
    'catalog-name' = 'lakehouse',
    'database-name' = 'staging',
    'table-name' = 'late_mobile_events'
);

-- Pipeline avec gestion des données tardives
INSERT INTO late_mobile_events
SELECT 
    event_id,
    user_id,
    event_type,
    event_time,
    PROCTIME() as arrival_time
FROM mobile_events
WHERE event_time < CURRENT_WATERMARK(event_time) - INTERVAL '5' MINUTE;
```

Les données tardives capturées séparément peuvent ensuite être réconciliées via des processus batch périodiques qui mettent à jour les agrégations Iceberg. Le support des opérations MERGE d'Iceberg v2 facilite cette réconciliation.

```sql
-- Réconciliation batch des données tardives (exécuté quotidiennement)
MERGE INTO analytics.daily_metrics AS target
USING (
    SELECT 
        DATE(event_time) as metric_date,
        user_id,
        COUNT(*) as late_event_count,
        SUM(CASE WHEN event_type = 'purchase' THEN 1 ELSE 0 END) as late_purchases
    FROM staging.late_mobile_events
    WHERE arrival_time >= CURRENT_DATE - INTERVAL 1 DAY
    GROUP BY DATE(event_time), user_id
) AS source
ON target.metric_date = source.metric_date 
   AND target.user_id = source.user_id
WHEN MATCHED THEN UPDATE SET
    event_count = target.event_count + source.late_event_count,
    purchase_count = target.purchase_count + source.late_purchases,
    last_updated = CURRENT_TIMESTAMP
WHEN NOT MATCHED THEN INSERT (metric_date, user_id, event_count, purchase_count, last_updated)
VALUES (source.metric_date, source.user_id, source.late_event_count, source.late_purchases, CURRENT_TIMESTAMP);
```

Cette approche hybride combine la faible latence du streaming pour la majorité des données avec la complétude du batch pour les données tardives, offrant le meilleur des deux mondes.

---

## IV.12.3 Résumé

Ce chapitre a retracé l'évolution des architectures de données depuis l'approche Lambda hybride jusqu'au Streaming Lakehouse moderne unifié. Les points essentiels à retenir sont structurés ci-dessous.

### L'Évolution Architecturale

L'architecture Lambda (2011) a posé les fondations en reconnaissant le besoin de combiner traitement batch et temps réel. Cependant, sa dualité intrinsèque engendre une duplication de code, une divergence des résultats et une complexité opérationnelle qui deviennent rapidement insurmontables à l'échelle de l'entreprise.

L'architecture Kappa (2014) a simplifié radicalement l'approche en traitant toutes les données comme un flux continu, avec le journal Kafka comme source de vérité unique. Cette unification élimine la duplication mais limite l'accès SQL et peut s'avérer coûteuse pour le stockage long terme.

Le Data Lakehouse (2017+) a apporté les garanties transactionnelles et l'interrogation SQL aux lacs de données via des formats comme Apache Iceberg, mais reste fondamentalement orienté batch.

Le Streaming Lakehouse (2023+) synthétise ces approches en combinant Kafka pour les données en mouvement et Iceberg pour les données au repos, avec une matérialisation automatique entre les deux couches. Cette architecture unifie enfin streaming et analytique dans une plateforme cohérente.

### Le Rôle Central de Kafka et Confluent

Apache Kafka s'impose comme la colonne vertébrale événementielle du Streaming Lakehouse, offrant le découplage, la persistance et l'ordonnancement nécessaires à une architecture de données fiable.

Tableflow de Confluent révolutionne l'intégration Kafka-Iceberg en éliminant les pipelines ETL traditionnels. La matérialisation automatique des topics en tables Iceberg, avec gestion native de l'évolution de schéma et de la compaction, réduit drastiquement la complexité opérationnelle.

Le Schema Registry assure la gouvernance des contrats de données entre producteurs et consommateurs, garantissant la cohérence structurelle de bout en bout.

### Patterns d'Implémentation

Trois patterns principaux permettent l'alimentation du Lakehouse depuis Kafka :

1. **Tableflow** pour une matérialisation simple sans code
2. **Kafka Connect** avec Iceberg Sink pour une intégration flexible avec contrôle fin
3. **Apache Flink** pour les transformations complexes et le routage dynamique

Le choix du pattern dépend des exigences de transformation, de l'expertise existante et du niveau de contrôle souhaité.

### Considérations Opérationnelles

L'observabilité de bout en bout est essentielle, couvrant les métriques Kafka (consumer lag, throughput), Flink (checkpointing, backpressure) et Iceberg (commits, file count, compaction).

La fréquence de commit constitue le principal levier de compromis entre latence et efficacité du stockage. Une compaction automatique bien configurée maintient les performances de requête.

### Recommandations Stratégiques

Pour les organisations qui entreprennent la transition vers le Streaming Lakehouse, les recommandations suivantes guident l'adoption :

1. **Commencer par les cas d'usage à haute valeur** où la latence actuelle constitue un frein tangible (détection de fraude, pricing dynamique, tableaux de bord temps réel)

2. **Adopter le pattern Shift Left** en déplaçant la gouvernance et la qualité des données vers la couche streaming plutôt que dans des processus ETL tardifs

3. **Standardiser sur Apache Iceberg** comme format de table pour maximiser l'interopérabilité entre les moteurs de consommation

4. **Investir dans l'observabilité** de bout en bout, du producteur Kafka jusqu'aux requêtes analytiques sur Iceberg

5. **Considérer Tableflow** pour les nouveaux cas d'usage afin de minimiser le code et la maintenance opérationnelle

6. **Planifier la migration incrémentale** depuis Lambda ou les architectures batch existantes, en maintenant les systèmes en parallèle pendant la transition

7. **Assurer la souveraineté des données** en déployant tous les composants (Kafka, Flink, stockage Iceberg) dans des régions canadiennes pour la conformité Loi 25 et LPRPDE

Le Streaming Lakehouse représente l'état de l'art pour les architectures de données modernes. En unifiant les données en mouvement et au repos sous une gouvernance commune, il permet aux organisations de répondre aux exigences de temps réel sans sacrifier la complétude historique ni la rigueur analytique. Pour les organisations canadiennes, cette architecture offre en outre un cadre robuste pour la conformité réglementaire grâce à sa traçabilité inhérente et son support natif des opérations d'effacement.

---

## Références

1. Confluent (2025). *Introducing Tableflow: Unifying Streaming and Analytics*. Documentation technique Confluent.

2. Apache Flink (2025). *From Stream to Lakehouse: Kafka Ingestion with the Flink Dynamic Iceberg Sink*. Blog officiel Apache Flink.

3. Waehner, K. (2025). *The Rise of Kappa Architecture in the Era of Agentic AI and Data Streaming*. Blog technique.

4. Waehner, K. (2025). *Data Streaming Meets Lakehouse: Apache Iceberg for Unified Real-Time and Batch Analytics*. Présentation Open Source Data Summit.

5. Ververica (2025). *From Kappa Architecture to Streamhouse: Making the Lakehouse Real-Time*. Documentation technique.

6. StreamNative (2025). *Ursa Wins VLDB 2025 Best Industry Paper: The First Lakehouse-Native Streaming Engine for Kafka*. Communiqué de presse.

7. Apache Iceberg (2025). *Flink Getting Started*. Documentation officielle.

8. Merced, A. (2024). *2025 Guide to Architecting an Iceberg Lakehouse*. Medium.

9. Kreps, J. (2014). *Questioning the Lambda Architecture*. Blog O'Reilly.

10. Marz, N. (2011). *How to beat the CAP theorem*. Blog personnel.
