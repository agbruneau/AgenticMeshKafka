
# Chapitre IV.8 - Conception de la Couche de Fédération

## Introduction

La couche de fédération représente l'interface stratégique entre votre Data Lakehouse Apache Iceberg et les consommateurs de données — analystes, scientifiques de données, applications et tableaux de bord. Elle unifie l'accès à des sources hétérogènes, masque la complexité technique sous-jacente et optimise l'exécution des requêtes pour offrir des performances analytiques de classe entreprise. Sans cette couche, votre Lakehouse demeure une infrastructure technique ; avec elle, il devient une plateforme de données accessible et exploitable par l'ensemble de l'organisation.

Dans l'architecture traditionnelle des entrepôts de données, la fédération se limitait souvent à connecter quelques sources via des liens de base de données. L'écosystème moderne du Lakehouse élargit considérablement cette vision : les moteurs de requête fédérée comme Trino, Dremio ou Spark SQL permettent d'interroger simultanément des tables Iceberg, des bases de données relationnelles, des API REST, des fichiers CSV et des services infonuagiques — le tout via une syntaxe SQL unifiée et des optimisations transparentes.

Ce chapitre vous guide dans la conception et l'implémentation d'une couche de fédération performante pour votre Lakehouse Iceberg. Nous examinerons les moteurs de requête disponibles, leurs forces respectives et leurs cas d'usage optimaux. Nous détaillerons les configurations de production, les stratégies d'optimisation des performances et les patterns architecturaux éprouvés. Les études de cas canadiennes illustreront comment des organisations de divers secteurs ont déployé ces technologies pour transformer leurs capacités analytiques.

L'enjeu dépasse la simple exécution de requêtes SQL. La couche de fédération définit l'expérience utilisateur de votre plateforme de données, influence les coûts d'exploitation et détermine la capacité de votre organisation à démocratiser l'accès aux données tout en maintenant une gouvernance rigoureuse.

---

## Fondamentaux de la Fédération de Données

### Définition et Objectifs

La fédération de données désigne la capacité d'accéder à des données distribuées sur plusieurs systèmes comme si elles résidaient dans une source unique. Cette abstraction offre plusieurs bénéfices stratégiques :

**Accès unifié** : Les utilisateurs interrogent les données via une interface SQL standard, indépendamment de leur localisation physique ou de leur format de stockage. Un analyste peut joindre une table Iceberg avec une base PostgreSQL et un fichier CSV sans connaître les détails techniques de chaque source.

**Réduction de la duplication** : Plutôt que de copier les données vers un entrepôt central, la fédération permet de les interroger in situ. Cette approche réduit les coûts de stockage, élimine les problèmes de synchronisation et garantit l'accès aux données les plus récentes.

**Agilité analytique** : Les nouvelles sources de données deviennent accessibles rapidement, sans les délais traditionnels d'intégration ETL. Cette agilité accélère les cycles d'analyse et permet de répondre plus rapidement aux besoins métier.

**Gouvernance centralisée** : La couche de fédération devient le point de contrôle unique pour l'accès aux données, simplifiant l'application des politiques de sécurité et d'audit.

### Architecture de Fédération pour Iceberg

L'architecture de fédération pour un Lakehouse Iceberg s'articule autour de plusieurs composantes interdépendantes :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      CONSOMMATEURS                                      │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐      │
│  │ Power BI│  │ Tableau │  │ Jupyter │  │  DBT    │  │   API   │      │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘      │
└───────┼────────────┼────────────┼────────────┼────────────┼───────────┘
        │            │            │            │            │
        └────────────┴────────────┴─────┬──────┴────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    COUCHE DE FÉDÉRATION                                 │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │              MOTEUR DE REQUÊTE FÉDÉRÉE                          │   │
│  │         (Trino / Dremio / Spark SQL / Starburst)                │   │
│  │  ┌─────────────────────────────────────────────────────────┐    │   │
│  │  │  • Planification distribuée des requêtes                │    │   │
│  │  │  • Optimisation (predicate pushdown, projection)        │    │   │
│  │  │  • Gestion du cache et des résultats intermédiaires     │    │   │
│  │  │  • Contrôle d'accès et audit                            │    │   │
│  │  └─────────────────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
        ┌────────────────────────┼────────────────────────────┐
        │                        │                            │
        ▼                        ▼                            ▼
┌───────────────┐        ┌───────────────┐        ┌───────────────┐
│   CATALOGUE   │        │   CATALOGUE   │        │   SOURCES     │
│   ICEBERG     │        │   ICEBERG     │        │   EXTERNES    │
│   (Bronze)    │        │ (Silver/Gold) │        │               │
└───────┬───────┘        └───────┬───────┘        └───────┬───────┘
        │                        │                        │
        ▼                        ▼                        ▼
┌───────────────┐        ┌───────────────┐        ┌───────────────┐
│    STOCKAGE   │        │    STOCKAGE   │        │  PostgreSQL   │
│  S3 / ADLS    │        │  S3 / ADLS    │        │  MongoDB      │
│               │        │               │        │  API REST     │
└───────────────┘        └───────────────┘        └───────────────┘
```

### Patterns de Requêtes Fédérées

La fédération de données supporte plusieurs patterns d'accès, chacun avec ses caractéristiques de performance et ses cas d'usage :

**Pattern 1 : Requête locale Iceberg**
La requête cible exclusivement des tables Iceberg du même catalogue. C'est le scénario le plus performant, bénéficiant pleinement des optimisations Iceberg.

```sql
-- Requête locale optimisée
SELECT 
    date_transaction,
    SUM(montant) as total_ventes
FROM lakehouse.gold.transactions
WHERE date_transaction >= DATE '2024-01-01'
GROUP BY date_transaction;
```

**Pattern 2 : Jointure inter-catalogue**
La requête joint des tables de différents catalogues Iceberg, potentiellement sur différents systèmes de stockage.

```sql
-- Jointure entre catalogues
SELECT 
    t.transaction_id,
    c.nom_client,
    t.montant
FROM lakehouse_ventes.gold.transactions t
JOIN lakehouse_crm.silver.clients c 
    ON t.client_id = c.client_id
WHERE t.date_transaction >= DATE '2024-01-01';
```

**Pattern 3 : Fédération hybride**
La requête combine des données Iceberg avec des sources externes (SGBD, fichiers, API).

```sql
-- Fédération avec source externe
SELECT 
    i.transaction_id,
    i.montant,
    p.nom_produit,
    inv.quantite_stock
FROM lakehouse.gold.transactions i
JOIN postgresql.erp.produits p 
    ON i.produit_id = p.produit_id
JOIN mongodb.inventaire.stock inv 
    ON p.sku = inv.sku
WHERE i.date_transaction = CURRENT_DATE;
```

**Pattern 4 : Virtualisation de données**
Création de vues virtuelles combinant plusieurs sources, exposées comme une source unique aux consommateurs.

```sql
-- Vue virtualisée
CREATE VIEW unified.analytics.vue_360_client AS
SELECT 
    c.client_id,
    c.nom,
    c.segment,
    COALESCE(t.total_achats, 0) as total_achats,
    COALESCE(s.score_satisfaction, 0) as satisfaction,
    COALESCE(w.derniere_visite, DATE '1900-01-01') as derniere_visite_web
FROM lakehouse.gold.clients c
LEFT JOIN lakehouse.gold.transactions_agregees t 
    ON c.client_id = t.client_id
LEFT JOIN salesforce.survey.scores s 
    ON c.courriel = s.email
LEFT JOIN snowplow.analytics.sessions w 
    ON c.client_id = w.user_id;
```

---

## Moteurs de Requête Fédérée

### Trino : Le Standard Open Source

Trino (anciennement Presto SQL) s'est établi comme le moteur de requête fédérée de référence dans l'écosystème open source. Conçu pour les requêtes analytiques interactives à grande échelle, il offre une architecture distribuée optimisée pour la latence faible et le débit élevé.

**Architecture Trino** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         CLUSTER TRINO                                   │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                      COORDINATOR                                 │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │   │
│  │  │   Parser    │  │  Analyzer   │  │  Planner    │              │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘              │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │   │
│  │  │  Optimizer  │  │  Scheduler  │  │  Metadata   │              │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                │                                        │
│                                ▼                                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │
│  │   Worker 1  │  │   Worker 2  │  │   Worker 3  │  │   Worker N  │   │
│  │  ┌───────┐  │  │  ┌───────┐  │  │  ┌───────┐  │  │  ┌───────┐  │   │
│  │  │ Tasks │  │  │  │ Tasks │  │  │  │ Tasks │  │  │  │ Tasks │  │   │
│  │  └───────┘  │  │  └───────┘  │  │  └───────┘  │  │  └───────┘  │   │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

**Forces de Trino** :

* Architecture MPP (Massively Parallel Processing) pour performances élevées
* Écosystème de connecteurs riche (40+ sources supportées)
* Communauté active et développement continu
* Coût nul de licence (open source Apache 2.0)
* Intégration native avec Iceberg de haute qualité

**Limitations** :

* Pas de persistance des résultats intermédiaires (tout en mémoire)
* Configuration et tuning requièrent une expertise
* Moins adapté aux requêtes très longues (heures)

**Configuration Trino pour Iceberg** :

```properties
# etc/catalog/lakehouse.properties
connector.name=iceberg

# Configuration du catalogue
iceberg.catalog.type=rest
iceberg.rest-catalog.uri=https://catalog.lakehouse.entreprise.ca
iceberg.rest-catalog.warehouse=s3://entreprise-lakehouse/warehouse

# Authentification
iceberg.rest-catalog.security=OAUTH2
iceberg.rest-catalog.oauth2.credential=file:///etc/trino/secrets/oauth-credential

# Configuration S3
iceberg.file-system-type=S3
s3.region=ca-central-1
s3.endpoint=s3.ca-central-1.amazonaws.com
s3.path-style-access=false

# Optimisations de lecture
iceberg.split-manager-threads=32
iceberg.max-partitions-per-writer=1000
iceberg.target-max-file-size=512MB

# Cache de métadonnées
iceberg.metadata.cache-ttl=5m
iceberg.metadata.cache-size=1000

# Optimisations de performance
iceberg.parquet.use-column-index=true
iceberg.parquet.use-bloom-filter=true
```

**Déploiement Kubernetes** :

```yaml
# trino-deployment.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: trino-config
  namespace: lakehouse
data:
  config.properties: |
    coordinator=true
    node-scheduler.include-coordinator=false
    http-server.http.port=8080
    discovery.uri=http://trino-coordinator:8080
    query.max-memory=50GB
    query.max-memory-per-node=8GB
    query.max-total-memory-per-node=10GB
    memory.heap-headroom-per-node=2GB
  
  jvm.config: |
    -server
    -Xmx16G
    -XX:+UseG1GC
    -XX:G1HeapRegionSize=32M
    -XX:+ExplicitGCInvokesConcurrent
    -XX:+HeapDumpOnOutOfMemoryError
    -XX:HeapDumpPath=/var/trino/data
    -Djdk.attach.allowAttachSelf=true
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: trino-coordinator
  namespace: lakehouse
spec:
  replicas: 1
  selector:
    matchLabels:
      app: trino
      role: coordinator
  template:
    metadata:
      labels:
        app: trino
        role: coordinator
    spec:
      containers:
        - name: trino
          image: trinodb/trino:435
          ports:
            - containerPort: 8080
          resources:
            requests:
              memory: "20Gi"
              cpu: "4"
            limits:
              memory: "24Gi"
              cpu: "8"
          volumeMounts:
            - name: config
              mountPath: /etc/trino
            - name: catalog
              mountPath: /etc/trino/catalog
      volumes:
        - name: config
          configMap:
            name: trino-config
        - name: catalog
          configMap:
            name: trino-catalogs
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: trino-worker
  namespace: lakehouse
spec:
  replicas: 10
  selector:
    matchLabels:
      app: trino
      role: worker
  template:
    metadata:
      labels:
        app: trino
        role: worker
    spec:
      containers:
        - name: trino
          image: trinodb/trino:435
          resources:
            requests:
              memory: "32Gi"
              cpu: "8"
            limits:
              memory: "40Gi"
              cpu: "16"
```

### Dremio : La Plateforme Lakehouse Unifiée

Dremio se positionne comme une plateforme Lakehouse complète, combinant moteur de requête, couche sémantique et accélération des requêtes. Son intégration native avec Iceberg et ses fonctionnalités d'entreprise en font une option attractive pour les organisations recherchant une solution clé en main.

**Architecture Dremio** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         DREMIO CLOUD / ON-PREM                          │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    COUCHE SÉMANTIQUE                            │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │   │
│  │  │   Espaces   │  │    Vues     │  │   Wikis     │              │   │
│  │  │   Virtuels  │  │  Virtuelles │  │   & Tags    │              │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    REFLECTIONS (Accélération)                   │   │
│  │  ┌─────────────────────────────────────────────────────────┐    │   │
│  │  │  Matérialisations automatiques pour accélération        │    │   │
│  │  │  • Raw Reflections (copie avec format optimisé)         │    │   │
│  │  │  • Aggregation Reflections (pré-agrégations)            │    │   │
│  │  └─────────────────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    MOTEUR D'EXÉCUTION                           │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │   │
│  │  │ Coordinator │  │  Executors  │  │   Cache     │              │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

**Forces de Dremio** :

* Reflections : accélération automatique des requêtes via matérialisation
* Couche sémantique intégrée pour gouvernance et collaboration
* Interface utilisateur intuitive pour exploration des données
* Support natif d'Iceberg avec optimisations spécifiques
* Mode SaaS (Dremio Cloud) pour déploiement simplifié

**Limitations** :

* Coût de licence significatif pour les fonctionnalités entreprise
* Complexité de dimensionnement des Reflections
* Moins de connecteurs que Trino pour sources exotiques

**Configuration des sources Iceberg dans Dremio** :

```sql
-- Ajout d'une source Iceberg Nessie
ALTER SOURCE lakehouse_bronze SET (
    TYPE = 'NESSIE',
    NESSIE_ENDPOINT = 'http://nessie:19120/api/v2',
    NESSIE_AUTH_TYPE = 'BEARER',
    NESSIE_AUTH_TOKEN = '${NESSIE_TOKEN}',
    NESSIE_DEFAULT_BRANCH = 'main',
    AWS_ROOT_PATH = 's3://entreprise-lakehouse/bronze',
    AWS_REGION = 'ca-central-1',
    ENABLE_ASYNC_ACCESS = true
);

-- Ajout d'une source Iceberg AWS Glue
ALTER SOURCE lakehouse_gold SET (
    TYPE = 'AWSGLUE',
    GLUE_CATALOG_ID = '123456789012',
    GLUE_AWS_REGION = 'ca-central-1',
    GLUE_DATABASE_FILTER = 'lakehouse_gold.*',
    AWS_ROOT_PATH = 's3://entreprise-lakehouse/gold'
);
```

**Configuration des Reflections** :

```sql
-- Création d'une Raw Reflection pour accélération de lecture
ALTER TABLE lakehouse_gold.transactions 
CREATE RAW REFLECTION reflection_transactions_raw
USING DISPLAY (
    transaction_id,
    client_id,
    produit_id,
    montant,
    date_transaction,
    region
)
PARTITION BY (TRUNCATE(date_transaction, MONTH), region)
LOCALSORT BY (client_id);

-- Création d'une Aggregation Reflection pour accélération d'agrégats
ALTER TABLE lakehouse_gold.transactions
CREATE AGGREGATION REFLECTION reflection_transactions_agg
USING DIMENSIONS (
    date_transaction,
    region,
    categorie_produit
)
MEASURES (
    montant (SUM, COUNT, MIN, MAX),
    quantite (SUM, COUNT)
)
PARTITION BY (TRUNCATE(date_transaction, MONTH));
```

### Apache Spark SQL

Apache Spark SQL offre des capacités de requête fédérée intégrées au framework Spark, particulièrement adaptées aux pipelines de transformation et aux requêtes longues.

**Forces de Spark SQL** :

* Intégration native avec les pipelines Spark existants
* Excellent pour les requêtes longues et les transformations complexes
* Support DataFrame/Dataset pour développement programmatique
* Écosystème MLlib pour analytics avancé

**Limitations** :

* Latence de démarrage (cold start) élevée
* Moins optimisé pour les requêtes ad-hoc interactives
* Overhead de gestion de cluster

**Configuration Spark SQL pour fédération** :

```python
from pyspark.sql import SparkSession

spark = SparkSession.builder \
    .appName("FederatedAnalytics") \
    .config("spark.sql.extensions", 
            "org.apache.iceberg.spark.extensions.IcebergSparkSessionExtensions") \
    \
    # Catalogue Iceberg principal
    .config("spark.sql.catalog.lakehouse", 
            "org.apache.iceberg.spark.SparkCatalog") \
    .config("spark.sql.catalog.lakehouse.type", "rest") \
    .config("spark.sql.catalog.lakehouse.uri", 
            "https://catalog.lakehouse.entreprise.ca") \
    \
    # Source PostgreSQL via JDBC
    .config("spark.sql.catalog.postgres", 
            "org.apache.spark.sql.execution.datasources.v2.jdbc.JDBCTableCatalog") \
    .config("spark.sql.catalog.postgres.url", 
            "jdbc:postgresql://postgres-erp:5432/production") \
    .config("spark.sql.catalog.postgres.driver", 
            "org.postgresql.Driver") \
    \
    # Source MongoDB
    .config("spark.mongodb.read.connection.uri", 
            "mongodb://mongo-cluster:27017") \
    .config("spark.mongodb.read.database", "analytics") \
    \
    .getOrCreate()

# Requête fédérée
df_federated = spark.sql("""
    SELECT 
        i.transaction_id,
        i.montant,
        p.nom_produit,
        m.score_satisfaction
    FROM lakehouse.gold.transactions i
    JOIN postgres.public.produits p 
        ON i.produit_id = p.id
    LEFT JOIN mongo_analytics.feedback m 
        ON i.transaction_id = m.transaction_id
    WHERE i.date_transaction >= '2024-01-01'
""")
```

### Starburst Galaxy et Enterprise

Starburst, l'entreprise derrière le fork commercial de Trino, offre des solutions entreprise avec fonctionnalités avancées de sécurité, gouvernance et support.

**Starburst Galaxy** (SaaS) :

* Déploiement géré dans le nuage
* Intégration native avec les services infonuagiques
* Découverte automatique des données
* Interface de requête intégrée

**Starburst Enterprise** (On-premise/Hybride) :

* Contrôle d'accès basé sur les rôles avancé
* Audit et lignage des données
* Support et SLA entreprise
* Intégration LDAP/AD native

**Configuration Starburst avec BIAC (Built-in Access Control)** :

```properties
# etc/catalog/lakehouse.properties avec contrôle d'accès
connector.name=iceberg
iceberg.catalog.type=rest

# Contrôle d'accès au niveau colonne
iceberg.security=system

# Masquage des données
column.mask.functions.enabled=true
```

```sql
-- Politique de masquage Starburst
CREATE FUNCTION mask_courriel(val VARCHAR)
RETURNS VARCHAR
RETURN CASE 
    WHEN is_member_of('data_admin') THEN val
    ELSE regexp_replace(val, '(.{2}).*@', '$1***@')
END;

-- Application du masquage
ALTER TABLE lakehouse.silver.clients 
ALTER COLUMN courriel SET MASK mask_courriel;
```

### Comparaison des Moteurs

| Critère                         | Trino      | Dremio                          | Spark SQL        | Starburst  |
| -------------------------------- | ---------- | ------------------------------- | ---------------- | ---------- |
| **Latence requêtes**      | Faible     | Très faible (avec Reflections) | Moyenne-Élevée | Faible     |
| **Requêtes longues**      | Limité    | Bon                             | Excellent        | Limité    |
| **Coût**                  | Gratuit    | 
$$
| Gratuit          |
$$

$$       |                  |            |
| **Facilité déploiement** | Moyenne    | Facile (Cloud)                  | Complexe         | Facile     |
| **Accélération**         | Non native | Reflections                     | Non native       | Non native |
| **Couche sémantique**     | Non        | Oui                             | Non              | Limitée   |
| **Connecteurs**            | 40+        | 30+                             | 20+              | 40+        |
| **Support Iceberg**        | Excellent  | Excellent                       | Excellent        | Excellent  |

---

## Configuration Avancée des Connecteurs

### Connecteur Iceberg Optimisé

La configuration du connecteur Iceberg influence directement les performances de vos requêtes. Une configuration soignée peut améliorer les temps de réponse de 2× à 10×.

**Optimisations de lecture** :

```properties
# etc/catalog/lakehouse.properties (Trino)

# Parallélisme de lecture
iceberg.split-manager-threads=32
iceberg.max-splits-per-task=256

# Utilisation des index Parquet
iceberg.parquet.use-column-index=true
iceberg.parquet.use-bloom-filter=true
iceberg.parquet.max-read-block-size=16MB

# Cache de métadonnées agressif
iceberg.metadata.cache-ttl=10m
iceberg.metadata.cache-size=2000
iceberg.manifest-cache-size=5000

# Pushdown de prédicats
iceberg.projection-pushdown-enabled=true
iceberg.statistics-based-optimization-enabled=true

# Lecture optimisée des petits fichiers
iceberg.merge-small-files-on-read=true
iceberg.small-file-threshold=64MB
```

**Configuration pour requêtes de jointure** :

```properties
# Optimisation des jointures distribuées
join-distribution-type=AUTOMATIC
join-reordering-strategy=AUTOMATIC
optimizer.join-max-broadcast-table-size=100MB

# Hash partitioning pour grandes jointures
hash-partition-count=64
```

### Connecteurs Sources Externes

**PostgreSQL avec pushdown optimisé** :

```properties
# etc/catalog/postgres_erp.properties
connector.name=postgresql
connection-url=jdbc:postgresql://postgres-erp.internal:5432/production
connection-user=${POSTGRES_USER}
connection-password=${POSTGRES_PASSWORD}

# Pool de connexions
connection-pool.max-size=30
connection-pool.min-size=5

# Pushdown agressif
postgresql.array-mapping=AS_ARRAY
postgresql.experimental.enable-string-pushdown-with-collate=true
postgresql.include-system-tables=false

# Statistiques pour optimisation
jdbc.statistics-enabled=true
jdbc.aggregation-pushdown.enabled=true
jdbc.join-pushdown.enabled=true
jdbc.topn-pushdown.enabled=true
```

**MongoDB pour données semi-structurées** :

```properties
# etc/catalog/mongo_analytics.properties
connector.name=mongodb
mongodb.connection-url=mongodb://mongo-cluster.internal:27017
mongodb.credentials=${MONGO_CREDENTIALS}

mongodb.schema-collection=_schema
mongodb.case-insensitive-name-matching=true

# Projection et filtre pushdown
mongodb.projection-pushdown-enabled=true
mongodb.filter-pushdown-enabled=true
```

**Elasticsearch pour recherche** :

```properties
# etc/catalog/elastic_logs.properties
connector.name=elasticsearch
elasticsearch.host=elasticsearch.internal
elasticsearch.port=9200
elasticsearch.default-schema-name=logs

elasticsearch.request-timeout=30s
elasticsearch.scroll-size=1000
elasticsearch.scroll-timeout=1m

# Pushdown de recherche
elasticsearch.pushdown-enabled=true
```

### Configuration Multi-Catalogue

Pour les architectures fédérant plusieurs catalogues Iceberg, la configuration cohérente assure une expérience utilisateur uniforme.

```properties
# Catalogue Bronze (données brutes)
# etc/catalog/bronze.properties
connector.name=iceberg
iceberg.catalog.type=rest
iceberg.rest-catalog.uri=https://catalog.lakehouse.entreprise.ca
iceberg.rest-catalog.prefix=bronze
iceberg.rest-catalog.warehouse=s3://entreprise-lakehouse/bronze

# Catalogue Silver (données nettoyées)
# etc/catalog/silver.properties
connector.name=iceberg
iceberg.catalog.type=rest
iceberg.rest-catalog.uri=https://catalog.lakehouse.entreprise.ca
iceberg.rest-catalog.prefix=silver
iceberg.rest-catalog.warehouse=s3://entreprise-lakehouse/silver

# Catalogue Gold (données agrégées)
# etc/catalog/gold.properties
connector.name=iceberg
iceberg.catalog.type=rest
iceberg.rest-catalog.uri=https://catalog.lakehouse.entreprise.ca
iceberg.rest-catalog.prefix=gold
iceberg.rest-catalog.warehouse=s3://entreprise-lakehouse/gold
```

---

## Optimisation des Performances

### Predicate Pushdown et Projection Pushdown

L'optimisation la plus critique pour les performances de requêtes fédérées est le pushdown — la capacité de pousser les filtres et les projections vers les sources de données plutôt que de les appliquer après lecture.

**Predicate Pushdown** :

```sql
-- Requête avec filtre
SELECT * FROM lakehouse.gold.transactions
WHERE date_transaction >= DATE '2024-01-01'
  AND region = 'QC'
  AND montant > 1000;

-- Sans pushdown: lecture de TOUTES les données, filtrage en mémoire
-- Avec pushdown: Iceberg utilise les statistiques de partition et de fichier
--                pour ne lire que les fichiers pertinents
```

**Vérification du pushdown** (Trino) :

```sql
EXPLAIN (TYPE DISTRIBUTED)
SELECT * FROM lakehouse.gold.transactions
WHERE date_transaction >= DATE '2024-01-01'
  AND region = 'QC';

-- Sortie attendue montrant le pushdown:
-- TableScan[table = lakehouse.gold.transactions, 
--           constraint = (date_transaction >= 2024-01-01, region = 'QC')]
--   Layout: [transaction_id, client_id, montant, ...]
--   Estimates: {rows: 125000, cpu: ?, memory: ?, network: ?}
```

**Projection Pushdown** :

```sql
-- Sélection de colonnes spécifiques
SELECT transaction_id, montant, date_transaction
FROM lakehouse.gold.transactions
WHERE date_transaction >= DATE '2024-01-01';

-- Iceberg lit uniquement les colonnes demandées du fichier Parquet
-- Réduction potentielle de 80-95% du volume de données lues
```

> **Performance**
>
> Sur une table de 10 To avec 50 colonnes, une requête sélectionnant 5 colonnes avec projection pushdown lit environ 1 To au lieu de 10 To. Combiné avec le predicate pushdown utilisant le partitionnement par date, la lecture peut descendre à 50 Go pour une journée de données. L'impact sur les coûts de transfert S3 et les temps de réponse est considérable.

### Stratégies de Jointure

Le choix de la stratégie de jointure influence dramatiquement les performances des requêtes fédérées.

**Broadcast Join** : La petite table est diffusée à tous les workers.

```sql
-- Forcer un broadcast join (table de référence)
SELECT /*+ BROADCAST(r) */ 
    t.transaction_id,
    r.nom_region
FROM lakehouse.gold.transactions t
JOIN lakehouse.gold.regions r ON t.region_id = r.region_id;
```

**Distributed Hash Join** : Les deux tables sont partitionnées par la clé de jointure.

```sql
-- Pour grandes tables, le hash join est automatiquement choisi
SELECT 
    t.transaction_id,
    c.nom_client
FROM lakehouse.gold.transactions t
JOIN lakehouse.gold.clients c ON t.client_id = c.client_id
WHERE t.date_transaction >= DATE '2024-01-01';
```

**Sort Merge Join** : Les tables sont triées puis fusionnées.

```sql
-- Efficace si les données sont déjà triées
-- Configuration Trino
SET SESSION join_distribution_type = 'PARTITIONED';
SET SESSION colocated_join = true;
```

**Matrice de décision jointure** :

| Taille Table A    | Taille Table B    | Stratégie Recommandée |
| ----------------- | ----------------- | ----------------------- |
| Petite (< 100 Mo) | Grande            | Broadcast A             |
| Grande            | Petite (< 100 Mo) | Broadcast B             |
| Grande            | Grande            | Distributed Hash        |
| Grande (triée)   | Grande (triée)   | Sort Merge              |

### Gestion du Cache

Le cache améliore significativement les performances des requêtes répétitives et des dashboards.

**Cache de métadonnées Trino** :

```properties
# Configuration du cache de métadonnées
iceberg.metadata.cache-ttl=10m
iceberg.metadata.cache-size=2000
iceberg.manifest-cache-size=5000

# Cache de statistiques
iceberg.statistics-enabled=true
iceberg.table-statistics-cache-ttl=1h
```

**Cache de données Dremio (Reflections)** :

```sql
-- Configuration du rafraîchissement des Reflections
ALTER TABLE lakehouse.gold.transactions
MODIFY REFLECTION reflection_transactions_raw
REFRESH EVERY 1 HOUR;

-- Rafraîchissement incrémental pour tables Iceberg
ALTER TABLE lakehouse.gold.transactions
MODIFY REFLECTION reflection_transactions_raw
REFRESH INCREMENTAL;
```

**Cache externe avec Alluxio** :

```yaml
# Configuration Alluxio comme cache pour Trino
# alluxio-site.properties
alluxio.master.mount.table.root.ufs=s3://entreprise-lakehouse/warehouse
alluxio.user.file.metadata.sync.interval=5m
alluxio.user.file.passive.cache.enabled=true
alluxio.user.file.cache.partially.read.block=true

# Intégration Trino
# etc/catalog/lakehouse.properties
hive.cache.enabled=true
hive.cache.location=/alluxio
hive.cache.data-transfer-size-threshold=16MB
```

### Dimensionnement et Scaling

**Formule de dimensionnement Trino** :

```
Mémoire totale workers = Volume données actives × Facteur concurrence × 0.3
Nombre workers = Mémoire totale / Mémoire par worker

Exemple:
- Volume données actives: 5 To
- Requêtes concurrentes: 50
- Mémoire par worker: 64 Go

Mémoire totale = 5000 Go × 50 × 0.3 = 75 000 Go
Nombre workers = 75 000 / 64 ≈ 1 170 workers (pic théorique)

En pratique avec pushdown efficace:
- Données réellement lues: 5% = 250 Go
- Mémoire nécessaire: 250 × 50 × 0.3 = 3 750 Go
- Nombre workers: 3 750 / 64 ≈ 60 workers
```

**Auto-scaling Kubernetes** :

```yaml
# trino-hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: trino-worker-hpa
  namespace: lakehouse
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: trino-worker
  minReplicas: 5
  maxReplicas: 50
  metrics:
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 75
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
        - type: Pods
          value: 4
          periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Pods
          value: 2
          periodSeconds: 120
```

---

## Couche Sémantique et Virtualisation

### Concepts de la Couche Sémantique

La couche sémantique ajoute une abstraction métier au-dessus des tables techniques, facilitant la compréhension et l'utilisation des données par les utilisateurs non techniques.

**Composantes de la couche sémantique** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      COUCHE SÉMANTIQUE                                  │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  MODÈLE MÉTIER                                                   │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │   │
│  │  │  Entités    │  │  Métriques  │  │ Dimensions  │              │   │
│  │  │  (Client,   │  │  (CA, Marge,│  │ (Temps,     │              │   │
│  │  │   Produit)  │  │   Volume)   │  │  Région)    │              │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  VUES VIRTUELLES                                                 │   │
│  │  ┌─────────────────────────────────────────────────────────┐    │   │
│  │  │  • vue_360_client (agrégation multi-sources)            │    │   │
│  │  │  • cube_ventes (dimensions pré-calculées)               │    │   │
│  │  │  • rapport_quotidien (filtres et transformations)       │    │   │
│  │  └─────────────────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  GOUVERNANCE                                                     │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │   │
│  │  │ Documentation│  │  Lignage    │  │  Contrôle   │              │   │
│  │  │ & Glossaire │  │  & Impact   │  │  d'accès    │              │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Implémentation avec Dremio

Dremio offre une couche sémantique native intégrée à son moteur de requête.

**Création d'un espace virtuel** :

```sql
-- Espace pour le domaine Ventes
CREATE SPACE "Analytique Ventes";

-- Dossier pour les vues client
CREATE FOLDER "Analytique Ventes"."Clients";

-- Vue 360° client
CREATE VDS "Analytique Ventes"."Clients"."Vue 360" AS
SELECT 
    c.client_id,
    c.nom,
    c.prenom,
    c.courriel,
    c.segment,
    c.date_inscription,
  
    -- Métriques transactionnelles
    COALESCE(t.nombre_transactions, 0) as nombre_achats,
    COALESCE(t.montant_total, 0) as valeur_totale,
    COALESCE(t.montant_moyen, 0) as panier_moyen,
    t.derniere_transaction,
  
    -- Score de fidélité calculé
    CASE 
        WHEN t.montant_total > 10000 AND t.nombre_transactions > 50 THEN 'Platine'
        WHEN t.montant_total > 5000 AND t.nombre_transactions > 20 THEN 'Or'
        WHEN t.montant_total > 1000 AND t.nombre_transactions > 5 THEN 'Argent'
        ELSE 'Bronze'
    END as niveau_fidelite,
  
    -- Données comportementales
    w.sessions_30j,
    w.pages_vues_30j,
  
    -- Satisfaction
    s.score_nps
  
FROM lakehouse_gold.clients c

LEFT JOIN (
    SELECT 
        client_id,
        COUNT(*) as nombre_transactions,
        SUM(montant) as montant_total,
        AVG(montant) as montant_moyen,
        MAX(date_transaction) as derniere_transaction
    FROM lakehouse_gold.transactions
    WHERE date_transaction >= CURRENT_DATE - INTERVAL '365' DAY
    GROUP BY client_id
) t ON c.client_id = t.client_id

LEFT JOIN lakehouse_silver.web_analytics w 
    ON c.client_id = w.user_id

LEFT JOIN salesforce.nps.scores s 
    ON c.courriel = s.email;

-- Documentation de la vue
COMMENT ON VDS "Analytique Ventes"."Clients"."Vue 360" IS 
'Vue consolidée 360° des clients combinant données transactionnelles, 
comportementales et satisfaction. Mise à jour en temps réel.';
```

**Configuration des Reflections pour accélération** :

```sql
-- Reflection pour accélération de la vue 360
ALTER VDS "Analytique Ventes"."Clients"."Vue 360"
CREATE RAW REFLECTION reflection_vue360
USING DISPLAY (
    client_id, nom, segment, niveau_fidelite,
    nombre_achats, valeur_totale, panier_moyen,
    score_nps
)
PARTITION BY (segment)
LOCALSORT BY (valeur_totale DESC);

-- Refresh incrémental quotidien
ALTER VDS "Analytique Ventes"."Clients"."Vue 360"
MODIFY REFLECTION reflection_vue360
REFRESH EVERY 24 HOURS;
```

### Implémentation avec DBT et Trino

Pour les organisations préférant une approche code-first, DBT combiné à Trino offre une couche sémantique déclarative.

**Structure projet DBT** :

```
dbt_lakehouse/
├── dbt_project.yml
├── models/
│   ├── staging/
│   │   ├── stg_transactions.sql
│   │   └── stg_clients.sql
│   ├── intermediate/
│   │   ├── int_transactions_enrichies.sql
│   │   └── int_clients_metriques.sql
│   └── marts/
│       ├── dim_clients.sql
│       ├── dim_produits.sql
│       ├── fct_ventes.sql
│       └── agg_ventes_quotidiennes.sql
├── analyses/
├── macros/
│   └── masquage_pii.sql
└── seeds/
```

**Configuration DBT pour Trino-Iceberg** :

```yaml
# dbt_project.yml
name: 'lakehouse_analytics'
version: '1.0.0'

profile: 'trino_lakehouse'

model-paths: ["models"]
analysis-paths: ["analyses"]
test-paths: ["tests"]
seed-paths: ["seeds"]
macro-paths: ["macros"]

target-path: "target"
clean-targets:
  - "target"
  - "dbt_packages"

vars:
  date_debut_analyse: '2023-01-01'

models:
  lakehouse_analytics:
    staging:
      +materialized: view
      +schema: staging
    intermediate:
      +materialized: ephemeral
    marts:
      +materialized: table
      +schema: gold
      +file_format: iceberg
      +table_properties:
        write.format.default: 'parquet'
        write.parquet.compression-codec: 'zstd'
```

```yaml
# profiles.yml
trino_lakehouse:
  target: prod
  outputs:
    prod:
      type: trino
      method: oauth
      host: trino.lakehouse.entreprise.ca
      port: 443
      user: dbt_service
      catalog: lakehouse
      schema: analytics
      threads: 8
      http_scheme: https
      session_properties:
        query_max_execution_time: 2h
```

**Modèle DBT pour dimension client** :

```sql
-- models/marts/dim_clients.sql
{{
    config(
        materialized='incremental',
        unique_key='client_id',
        incremental_strategy='merge',
        file_format='iceberg',
        partition_by=['segment'],
        table_properties={
            'write.target-file-size-bytes': '536870912'
        }
    )
}}

WITH clients_base AS (
    SELECT * FROM {{ ref('stg_clients') }}
),

metriques_transactions AS (
    SELECT * FROM {{ ref('int_clients_metriques') }}
),

final AS (
    SELECT 
        c.client_id,
        c.nom,
        c.prenom,
        {{ masquer_courriel('c.courriel') }} as courriel,
        c.segment,
        c.province,
        c.date_inscription,
      
        m.nombre_transactions,
        m.montant_total,
        m.panier_moyen,
        m.derniere_transaction,
      
        CASE 
            WHEN m.montant_total > 10000 THEN 'Platine'
            WHEN m.montant_total > 5000 THEN 'Or'
            WHEN m.montant_total > 1000 THEN 'Argent'
            ELSE 'Bronze'
        END as niveau_fidelite,
      
        CURRENT_TIMESTAMP as _dbt_updated_at
      
    FROM clients_base c
    LEFT JOIN metriques_transactions m 
        ON c.client_id = m.client_id
  
    {% if is_incremental() %}
    WHERE c._updated_at > (SELECT MAX(_dbt_updated_at) FROM {{ this }})
       OR m._updated_at > (SELECT MAX(_dbt_updated_at) FROM {{ this }})
    {% endif %}
)

SELECT * FROM final
```

---

## Sécurité et Gouvernance

### Contrôle d'Accès Unifié

La couche de fédération devient le point de contrôle central pour la sécurité d'accès aux données. Une stratégie cohérente doit couvrir l'authentification, l'autorisation et l'audit.

**Architecture de sécurité** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    UTILISATEURS / APPLICATIONS                          │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    AUTHENTIFICATION                                     │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  OAuth 2.0 / OIDC → Azure AD / Okta / Keycloak                  │   │
│  │  Kerberos → Active Directory (legacy)                            │   │
│  │  Certificats mTLS → Service accounts                             │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    AUTORISATION                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Niveau Catalogue: GRANT USAGE ON CATALOG lakehouse              │   │
│  │  Niveau Schema:    GRANT SELECT ON SCHEMA gold                   │   │
│  │  Niveau Table:     GRANT SELECT ON TABLE transactions            │   │
│  │  Niveau Colonne:   Column masking policies                       │   │
│  │  Niveau Ligne:     Row-level security filters                    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    AUDIT                                                │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  • Qui a accédé à quelles données                               │   │
│  │  • Quand et depuis où                                           │   │
│  │  • Quelles requêtes exécutées                                   │   │
│  │  • Combien de données retournées                                │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

**Configuration OAuth2 Trino** :

```properties
# etc/config.properties
http-server.authentication.type=OAUTH2

http-server.authentication.oauth2.issuer=https://login.microsoftonline.com/${TENANT_ID}/v2.0
http-server.authentication.oauth2.client-id=${CLIENT_ID}
http-server.authentication.oauth2.client-secret=${CLIENT_SECRET}
http-server.authentication.oauth2.scopes=openid,profile,email
http-server.authentication.oauth2.principal-field=email
http-server.authentication.oauth2.groups-field=groups
```

**Contrôle d'accès basé sur les rôles** :

```sql
-- Création des rôles
CREATE ROLE data_analyst;
CREATE ROLE data_engineer;
CREATE ROLE data_scientist;
CREATE ROLE finance_analyst;

-- Permissions globales
GRANT USAGE ON CATALOG lakehouse TO ROLE data_analyst;
GRANT SELECT ON SCHEMA lakehouse.gold TO ROLE data_analyst;

GRANT USAGE ON CATALOG lakehouse TO ROLE data_engineer;
GRANT ALL PRIVILEGES ON SCHEMA lakehouse.bronze TO ROLE data_engineer;
GRANT ALL PRIVILEGES ON SCHEMA lakehouse.silver TO ROLE data_engineer;
GRANT SELECT ON SCHEMA lakehouse.gold TO ROLE data_engineer;

-- Permissions spécifiques au domaine Finance
GRANT SELECT ON TABLE lakehouse.gold.transactions TO ROLE finance_analyst;
GRANT SELECT ON TABLE lakehouse.gold.revenus_mensuels TO ROLE finance_analyst;

-- Attribution des rôles
GRANT ROLE data_analyst TO USER 'marie.tremblay@entreprise.ca';
GRANT ROLE finance_analyst TO GROUP 'Finance-Equipe@entreprise.ca';
```

### Masquage et Filtrage des Données

**Masquage au niveau colonne** :

```sql
-- Fonction de masquage pour courriel
CREATE FUNCTION masque_courriel(val VARCHAR)
RETURNS VARCHAR
RETURN 
    CASE 
        WHEN current_user IN ('admin@entreprise.ca') THEN val
        WHEN is_role_enabled('data_admin') THEN val
        ELSE regexp_replace(val, '(.{2}).*@(.*)\.(.{2,3})$', '$1***@$2.$3')
    END;

-- Application du masquage
ALTER TABLE lakehouse.silver.clients 
ALTER COLUMN courriel SET MASK masque_courriel;

-- Masquage pour numéro de carte
CREATE FUNCTION masque_carte(val VARCHAR)
RETURNS VARCHAR
RETURN 
    CASE 
        WHEN is_role_enabled('finance_admin') THEN val
        ELSE concat('****-****-****-', right(val, 4))
    END;
```

**Filtrage au niveau ligne (Row-Level Security)** :

```sql
-- Politique de filtrage par région
CREATE ROW ACCESS POLICY filtre_region ON lakehouse.gold.transactions
FOR SELECT
TO ROLE regional_analyst
USING (
    region IN (
        SELECT region_autorisee 
        FROM lakehouse.security.autorisations_regions 
        WHERE utilisateur = current_user
    )
);

-- Politique de filtrage par date (données récentes uniquement)
CREATE ROW ACCESS POLICY filtre_historique ON lakehouse.gold.transactions
FOR SELECT
TO ROLE analyst_junior
USING (
    date_transaction >= CURRENT_DATE - INTERVAL '90' DAY
);
```

### Audit et Traçabilité

**Configuration de l'audit Trino** :

```properties
# etc/event-listener.properties
event-listener.name=http
http-client.request-timeout=30s
http-event-listener.connect-ingest-uri=https://audit.entreprise.ca/ingest
```

```java
// Plugin d'audit personnalisé
public class AuditEventListener implements EventListener {
    @Override
    public void queryCompleted(QueryCompletedEvent event) {
        AuditRecord record = AuditRecord.builder()
            .queryId(event.getMetadata().getQueryId())
            .user(event.getContext().getUser())
            .query(event.getMetadata().getQuery())
            .catalog(event.getContext().getCatalog().orElse(""))
            .schema(event.getContext().getSchema().orElse(""))
            .tablesAccessed(extractTables(event))
            .rowsReturned(event.getStatistics().getOutputRows())
            .executionTimeMs(event.getStatistics().getWallTime().toMillis())
            .status(event.getMetadata().getQueryState())
            .timestamp(Instant.now())
            .build();
      
        auditService.log(record);
    }
}
```

**Requêtes d'analyse d'audit** :

```sql
-- Accès aux données sensibles (dernières 24h)
SELECT 
    user_email,
    table_name,
    COUNT(*) as nb_requetes,
    SUM(rows_returned) as total_lignes
FROM audit.query_logs
WHERE timestamp >= CURRENT_TIMESTAMP - INTERVAL '24' HOUR
  AND table_name LIKE '%pii%' OR table_name LIKE '%sensitive%'
GROUP BY user_email, table_name
ORDER BY total_lignes DESC;

-- Détection d'anomalies (volume inhabituel)
WITH stats_utilisateur AS (
    SELECT 
        user_email,
        AVG(rows_returned) as moy_lignes,
        STDDEV(rows_returned) as stddev_lignes
    FROM audit.query_logs
    WHERE timestamp >= CURRENT_TIMESTAMP - INTERVAL '30' DAY
    GROUP BY user_email
)
SELECT 
    q.user_email,
    q.query_id,
    q.rows_returned,
    s.moy_lignes,
    (q.rows_returned - s.moy_lignes) / NULLIF(s.stddev_lignes, 0) as z_score
FROM audit.query_logs q
JOIN stats_utilisateur s ON q.user_email = s.user_email
WHERE q.timestamp >= CURRENT_TIMESTAMP - INTERVAL '1' HOUR
  AND (q.rows_returned - s.moy_lignes) / NULLIF(s.stddev_lignes, 0) > 3;
```

---

## Intégration avec les Outils BI

### Power BI et Microsoft Fabric

L'intégration de Power BI avec un Lakehouse Iceberg offre plusieurs chemins, chacun avec ses compromis.

**Option 1 : Connexion directe via Trino/Dremio ODBC**

```
Power BI Desktop → ODBC Driver → Trino/Dremio → Iceberg
```

Configuration du DSN ODBC :

```ini
# odbc.ini
[Trino_Lakehouse]
Driver=/opt/trino-odbc/lib/libtrino-odbc.so
Host=trino.lakehouse.entreprise.ca
Port=443
SSL=1
AuthenticationType=OAuth2
OAuth2ClientId=${CLIENT_ID}
OAuth2ClientSecret=${CLIENT_SECRET}
OAuth2TokenEndpoint=https://login.microsoftonline.com/${TENANT_ID}/oauth2/v2.0/token
Catalog=lakehouse
Schema=gold
```

**Option 2 : Microsoft Fabric avec OneLake Shortcuts**

```sql
-- Dans Fabric, création d'un raccourci vers S3
-- (Via interface Fabric ou API)
CREATE SHORTCUT gold_transactions
    LOCATION 's3://entreprise-lakehouse/gold/transactions'
    WITH (
        FORMAT = 'ICEBERG',
        CREDENTIAL = 'aws_credential'
    );
```

**Option 3 : Direct Lake avec Dremio**

Dremio peut exposer ses sources comme endpoints compatibles Power BI Direct Lake :

```sql
-- Configuration Dremio pour Power BI
ALTER SOURCE "Lakehouse Gold" SET (
    ENABLE_POWERBI_DIRECTLAKE = true,
    POWERBI_WORKSPACE_ID = '${WORKSPACE_ID}'
);
```

> **Performance**
>
> Direct Lake offre les meilleures performances pour Power BI car les données restent dans le format Parquet/Iceberg sans import. Les rapports interrogent directement le Lakehouse avec un cache optimisé. Pour les datasets volumineux (> 100 Go), Direct Lake peut réduire les temps de rafraîchissement de 90% par rapport à l'import.

### Tableau

**Connexion Tableau via JDBC** :

```xml
<!-- tableau-trino.tdc -->
<connection-customization class='genericjdbc' enabled='true' version='1.0'>
    <vendor name='trino' />
    <driver name='trino' />
    <customizations>
        <customization name='CAP_QUERY_SUBQUERIES_WITH_TOP' value='no' />
        <customization name='CAP_SELECT_INTO' value='no' />
        <customization name='CAP_SUPPORTS_UNION' value='yes' />
        <customization name='SQL_AGGREGATE_FUNCTION_STDDEV' value='STDDEV_SAMP' />
        <customization name='SQL_AGGREGATE_FUNCTION_VAR' value='VAR_SAMP' />
    </customizations>
</connection-customization>
```

**Live connection optimisée** :

```
# Paramètres de connexion Tableau
Server: trino.lakehouse.entreprise.ca
Port: 443
Catalog: lakehouse
Schema: gold
Authentication: OAuth
Initial SQL: SET SESSION query_max_execution_time = '10m'
```

### Superset et Metabase

**Configuration Superset** :

```python
# superset_config.py
SQLALCHEMY_DATABASE_URI = 'trino://trino.lakehouse.entreprise.ca:443/lakehouse'

DATABASES = {
    'lakehouse': {
        'engine': 'trino',
        'host': 'trino.lakehouse.entreprise.ca',
        'port': 443,
        'catalog': 'lakehouse',
        'schema': 'gold',
        'extra': {
            'http_scheme': 'https',
            'auth': 'oauth2',
            'oauth2_config': {
                'client_id': '${CLIENT_ID}',
                'client_secret': '${CLIENT_SECRET}',
                'token_url': 'https://login.microsoftonline.com/${TENANT_ID}/oauth2/v2.0/token'
            }
        }
    }
}
```

**Configuration Metabase** :

```yaml
# Via API Metabase
POST /api/database
{
  "engine": "presto-jdbc",
  "name": "Lakehouse Analytics",
  "details": {
    "host": "trino.lakehouse.entreprise.ca",
    "port": 443,
    "catalog": "lakehouse",
    "schema": "gold",
    "ssl": true,
    "auth-method": "oauth2",
    "oauth2-client-id": "${CLIENT_ID}",
    "oauth2-client-secret": "${CLIENT_SECRET}"
  }
}
```

---

## Études de Cas Canadiennes

### Secteur Bancaire : Institution Financière Majeure

> **Étude de cas : Grande banque canadienne**
>
> *Secteur* : Services financiers
>
> *Défi* : Unifier l'accès analytique à 15+ silos de données (core banking, CRM, risque, conformité) tout en maintenant une ségrégation stricte entre lignes d'affaires conformément aux exigences du BSIF. Les analystes passaient 60% de leur temps à chercher et préparer les données plutôt qu'à les analyser.
>
> *Solution* : Déploiement de Starburst Enterprise comme couche de fédération, connectant des catalogues Iceberg (données de marché, transactions), des bases Oracle (core banking), et Salesforce (CRM). Implémentation de row-level security basée sur la ligne d'affaires de l'utilisateur.
>
> *Architecture* :
>
> ```
> ┌─────────────────────────────────────────────────────────────────┐
> │                    Starburst Enterprise                         │
> │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐           │
> │  │ Iceberg │  │ Oracle  │  │Salesforce│  │  Kafka  │           │
> │  │ (Marché)│  │ (Core)  │  │  (CRM)  │  │(Streaming)│          │
> │  └─────────┘  └─────────┘  └─────────┘  └─────────┘           │
> │                                                                 │
> │  Row-Level Security: WHERE ligne_affaires = current_user_lob() │
> └─────────────────────────────────────────────────────────────────┘
> ```
>
> *Résultats* :
>
> * Temps de préparation des données réduit de 60% à 15%
> * 500+ analystes avec accès self-service sécurisé
> * Conformité BSIF maintenue avec audit complet
> * ROI de 400% sur 18 mois

### Secteur Commerce : Détaillant Omnicanal

> **Étude de cas : Chaîne de magasins pancanadienne**
>
> *Secteur* : Commerce de détail
>
> *Défi* : Créer une vue unifiée des clients cross-canal (magasin, web, application mobile, marketplace) pour personnalisation en temps réel. Les données étaient dispersées entre 8 systèmes avec des identifiants clients incompatibles.
>
> *Solution* : Déploiement de Dremio Cloud comme plateforme de fédération avec couche sémantique. Création d'un graphe d'identité client via Iceberg, exposé comme vue virtuelle « Client 360 ». Reflections pour accélération des tableaux de bord Power BI temps réel.
>
> *Particularités* :
>
> * Resolution d'identité via algorithme de matching probabiliste
> * Rafraîchissement des Reflections toutes les 15 minutes
> * Intégration avec système de recommandation ML
>
>   *Résultats* :
> * Vue client unifiée couvrant 8M de clients
> * Latence des tableaux de bord < 2 secondes (vs 45 secondes avant)
> * Augmentation de 23% du taux de conversion des recommandations
> * Réduction de 70% des coûts d'infrastructure BI

### Secteur Santé : Réseau Hospitalier Provincial

> **Étude de cas : Réseau de santé provincial**
>
> *Secteur* : Santé publique
>
> *Défi* : Permettre l'analyse populationnelle tout en respectant strictement la confidentialité des données de santé. Les chercheurs avaient besoin d'accéder à des données agrégées sans jamais voir de données individuelles identifiantes.
>
> *Solution* : Architecture Trino avec couche de fédération hautement sécurisée. Implémentation de k-anonymisation automatique pour toutes les requêtes retournant moins de 5 individus. Masquage systématique des identifiants directs et quasi-identifiants.
>
> *Architecture de sécurité* :
>
> ```sql
> -- Politique de k-anonymisation
> CREATE POLICY k_anon ON requete_resultat
> WHEN COUNT(*) < 5 THEN 
>     SUPPRESS_RESULT('Résultat supprimé: k < 5')
> ```
>
> *Résultats* :
>
> * Conformité Loi 25 et LPRPDE validée par commissaire à la vie privée
> * 200+ chercheurs avec accès sécurisé
> * Temps d'accès aux données pour recherche réduit de 6 mois à 2 semaines
> * Aucune brèche de confidentialité depuis le déploiement (3 ans)

### Secteur Énergie : Producteur d'Électricité

> **Étude de cas : Société d'État hydroélectrique**
>
> *Secteur* : Énergie et services publics
>
> *Défi* : Analyser en temps réel les données de 50 000 capteurs IoT répartis sur le réseau de production et distribution, combinées aux données météorologiques, de marché et de maintenance pour optimisation de la production.
>
> *Solution* : Trino déployé sur Kubernetes avec auto-scaling pour gérer les pics de charge. Fédération entre Iceberg (historique capteurs), TimescaleDB (séries temporelles temps réel), et API météo externes. Cache Alluxio pour accélération des lectures répétitives.
>
> *Métriques* :
>
> * 50 milliards de points de données/jour ingérés
> * Requêtes analytiques en < 10 secondes sur 2 ans d'historique
> * 200 requêtes concurrentes en pic
>
>   *Résultats* :
> * Optimisation de la production augmentant les revenus de 3%
> * Détection précoce des anomalies réduisant les pannes de 25%
> * Économies de 15M$/an en maintenance prédictive

---

## Bonnes Pratiques et Recommandations

### Architecture de Référence

**Architecture recommandée pour entreprise moyenne** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      CONSOMMATEURS                                      │
│  Power BI │ Tableau │ Jupyter │ DBT │ Applications │ API REST          │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              COUCHE DE FÉDÉRATION (Trino/Dremio)                       │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Coordinator: 2 instances (HA)                                   │   │
│  │  Workers: 10-50 (auto-scaling)                                   │   │
│  │  Mémoire par worker: 32-64 Go                                    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
        ┌────────────────────────┼────────────────────────────────┐
        │                        │                                │
        ▼                        ▼                                ▼
┌───────────────────┐  ┌───────────────────┐  ┌───────────────────┐
│  ICEBERG          │  │  ICEBERG          │  │  SOURCES          │
│  Bronze/Silver    │  │  Gold             │  │  EXTERNES         │
│                   │  │  (Optimisé BI)    │  │                   │
│  REST Catalog     │  │  REST Catalog     │  │  PostgreSQL       │
│  S3/ADLS          │  │  S3/ADLS          │  │  MongoDB          │
│                   │  │                   │  │  API              │
└───────────────────┘  └───────────────────┘  └───────────────────┘
```

### Checklist de Déploiement

**Infrastructure** :

* [ ] Cluster Trino/Dremio dimensionné selon charge attendue
* [ ] Haute disponibilité du coordinator configurée
* [ ] Auto-scaling des workers activé
* [ ] Monitoring et alerting en place
* [ ] Sauvegarde des configurations

**Sécurité** :

* [ ] Authentification OAuth2/OIDC configurée
* [ ] Rôles et permissions définis
* [ ] Masquage des données sensibles implémenté
* [ ] Row-level security si nécessaire
* [ ] Audit activé et journaux centralisés

**Performance** :

* [ ] Predicate pushdown vérifié pour chaque connecteur
* [ ] Cache de métadonnées configuré
* [ ] Reflections/matérialisations pour requêtes fréquentes
* [ ] Statistiques de table à jour

**Gouvernance** :

* [ ] Documentation des sources et vues
* [ ] Lignage des données tracé
* [ ] Glossaire métier défini
* [ ] Propriétaires de données identifiés

### Matrice de Décision

**Choix du moteur de fédération** :

| Critère                 | Trino      | Dremio     | Spark SQL  | Starburst  |
| ------------------------ | ---------- | ---------- | ---------- | ---------- |
| Budget limité           | ★★★★★ | ★★☆☆☆ | ★★★★★ | ★☆☆☆☆ |
| Requêtes interactives   | ★★★★★ | ★★★★★ | ★★☆☆☆ | ★★★★★ |
| Couche sémantique       | ★★☆☆☆ | ★★★★★ | ★★☆☆☆ | ★★★☆☆ |
| Simplicité déploiement | ★★★☆☆ | ★★★★★ | ★★☆☆☆ | ★★★★☆ |
| Connecteurs sources      | ★★★★★ | ★★★★☆ | ★★★☆☆ | ★★★★★ |
| Support entreprise       | ★★☆☆☆ | ★★★★★ | ★★★☆☆ | ★★★★★ |
| Intégration BI          | ★★★★☆ | ★★★★★ | ★★★☆☆ | ★★★★☆ |

**Arbre de décision** :

```
                    ┌─────────────────────────────────┐
                    │  Quel est votre budget?         │
                    └─────────────────────────────────┘
                                    │
            ┌───────────────────────┼───────────────────────┐
            ▼                       ▼                       ▼
        [Limité]               [Modéré]                [Confortable]
            │                       │                       │
            ▼                       ▼                       ▼
    ┌───────────────┐       ┌───────────────┐       ┌───────────────┐
    │ Expertise     │       │ Couche        │       │ Support       │
    │ interne?      │       │ sémantique    │       │ critique?     │
    │               │       │ requise?      │       │               │
    └───────────────┘       └───────────────┘       └───────────────┘
            │                       │                       │
      ┌─────┴─────┐           ┌─────┴─────┐           ┌─────┴─────┐
      ▼           ▼           ▼           ▼           ▼           ▼
    [OUI]       [NON]       [OUI]       [NON]       [OUI]       [NON]
      │           │           │           │           │           │
      ▼           ▼           ▼           ▼           ▼           ▼
    Trino     Spark SQL    Dremio      Trino    Starburst    Dremio
```

---

## Conclusion

La couche de fédération transforme votre Data Lakehouse Apache Iceberg d'une infrastructure de stockage en une plateforme analytique accessible à l'ensemble de l'organisation. Elle unifie l'accès à des sources hétérogènes, optimise l'exécution des requêtes et centralise la gouvernance des données. Le choix du moteur de fédération — Trino pour sa flexibilité open source, Dremio pour sa couche sémantique intégrée, ou Starburst pour le support entreprise — dépend de votre contexte spécifique en termes de budget, d'expertise et d'exigences fonctionnelles.

Les performances de cette couche reposent sur une configuration soignée : predicate et projection pushdown correctement activés, cache de métadonnées dimensionné, stratégies de jointure optimisées et, pour les cas d'usage répétitifs, matérialisations via Reflections ou vues matérialisées. Le dimensionnement doit anticiper les pics de charge avec des mécanismes d'auto-scaling appropriés.

La sécurité et la gouvernance constituent des piliers incontournables. L'authentification moderne via OAuth2/OIDC, le contrôle d'accès granulaire jusqu'au niveau colonne et ligne, le masquage des données sensibles et l'audit exhaustif permettent de démocratiser l'accès aux données tout en maintenant la conformité réglementaire — exigence particulièrement critique dans le contexte canadien avec la Loi 25 et les régulations sectorielles.

Les études de cas présentées démontrent que des organisations de secteurs variés — finance, commerce, santé, énergie — ont réussi à déployer ces architectures de fédération pour transformer leurs capacités analytiques. Les gains mesurés sont significatifs : réduction drastique du temps de préparation des données, accélération des temps de réponse, et amélioration mesurable des décisions métier.

Le chapitre suivant explore la couche de consommation et les patterns d'accès aux données, où nous examinerons comment les différents profils d'utilisateurs — analystes, scientifiques de données, applications — interagissent avec votre Lakehouse fédéré pour extraire de la valeur des données.

---

## Résumé

**Fondamentaux de la fédération** :

* Accès unifié à des sources hétérogènes via SQL standard
* Réduction de la duplication par requêtes in situ
* Point de contrôle central pour sécurité et gouvernance
* Patterns : local Iceberg, inter-catalogue, hybride, virtualisation

**Moteurs principaux** :

* **Trino** : Standard open source, MPP, 40+ connecteurs, gratuit
* **Dremio** : Couche sémantique, Reflections, interface intuitive
* **Spark SQL** : Intégration pipelines, requêtes longues, ML
* **Starburst** : Trino entreprise, support, gouvernance avancée

**Optimisations de performance** :

* Predicate et projection pushdown essentiels
* Stratégies de jointure : broadcast, hash, merge selon tailles
* Cache métadonnées et Reflections pour requêtes répétitives
* Auto-scaling pour gérer la variabilité de charge

**Sécurité et gouvernance** :

* Authentification OAuth2/OIDC avec IdP entreprise
* RBAC au niveau catalogue, schema, table, colonne, ligne
* Masquage des données sensibles via fonctions
* Audit exhaustif pour traçabilité et conformité

**Intégration BI** :

* Power BI : ODBC, Fabric shortcuts, Direct Lake
* Tableau : JDBC avec customizations
* Superset/Metabase : configurations natives

**Recommandations** :

* Trino pour budget limité et flexibilité
* Dremio pour couche sémantique et simplicité
* Starburst pour support entreprise critique
* Toujours activer pushdown et configurer le cache

---

*Ce chapitre établit l'interface d'accès à votre Lakehouse. Le chapitre suivant, « Conception de la Couche de Consommation », détaille les patterns d'accès pour les différents profils d'utilisateurs et applications.*
