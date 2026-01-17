
# Chapitre IV.7 - Implémentation de la Couche de Catalogue

## Introduction

Le catalogue constitue le cerveau de votre Data Lakehouse Apache Iceberg. Tandis que la couche de stockage héberge physiquement les données et que la couche d'ingestion alimente le système, le catalogue orchestre l'ensemble en maintenant la cartographie complète de vos actifs de données : où se trouvent les tables, comment elles sont structurées, quelles versions existent et qui peut y accéder. Sans catalogue robuste, votre Lakehouse n'est qu'une collection de fichiers Parquet dispersés dans le stockage objet — techniquement accessibles, mais pratiquement inexploitables à l'échelle de l'entreprise.

Apache Iceberg introduit une architecture de catalogue distinctive qui sépare clairement les responsabilités. Le catalogue Iceberg ne stocke pas les métadonnées détaillées des tables ; il maintient plutôt un registre de pointeurs vers les fichiers de métadonnées résidant dans le stockage. Cette indirection élégante permet une flexibilité remarquable : plusieurs moteurs de requête peuvent accéder simultanément aux mêmes tables, les métadonnées évoluent de manière atomique et la migration entre catalogues devient une opération de reconfiguration plutôt qu'un déplacement massif de données.

Ce chapitre vous guide dans la sélection, l'implémentation et l'opération d'une couche de catalogue adaptée à votre contexte. Nous examinerons les différentes implémentations disponibles — du traditionnel Hive Metastore au moderne REST Catalog, en passant par les solutions gérées des fournisseurs infonuagiques —, détaillerons les configurations de production et explorerons les stratégies de fédération pour les architectures multi-environnements.

L'enjeu dépasse la simple gestion technique des métadonnées. Le catalogue devient le point de contrôle central pour la gouvernance des données, la sécurité d'accès et la découverte des actifs. Une implémentation réussie transforme votre Lakehouse d'une infrastructure technique en une plateforme de données gouvernée, où chaque table est documentée, chaque accès est contrôlé et chaque évolution est tracée.

---

## Fondamentaux du Catalogue Iceberg

### Rôle et Responsabilités

Le catalogue Iceberg assume des responsabilités précises et délimitées dans l'architecture globale du Lakehouse. Comprendre cette délimitation est essentiel pour concevoir une implémentation efficace.

**Registre des tables** : Le catalogue maintient la correspondance entre les noms logiques des tables (par exemple, `lakehouse.ventes.transactions`) et l'emplacement physique de leurs métadonnées dans le stockage. Cette indirection permet de renommer ou déplacer des tables sans modifier les fichiers sous-jacents.

**Gestion des espaces de noms** : Les catalogues Iceberg supportent une hiérarchie d'espaces de noms (namespaces) permettant d'organiser logiquement les tables. Cette structure reflète typiquement l'organisation métier : domaines, départements ou environnements.

**Contrôle de concurrence** : Lors des commits de modifications, le catalogue garantit l'atomicité des mises à jour du pointeur de métadonnées. Ce mécanisme prévient les conditions de course entre écrivains concurrents.

**Découverte des métadonnées** : Le catalogue expose les informations nécessaires aux moteurs de requête pour localiser et interroger les tables : schéma, partitionnement, statistiques et emplacement des fichiers de manifeste.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         CATALOGUE ICEBERG                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Registre des Tables                                             │   │
│  │  ┌─────────────────┬──────────────────────────────────────────┐ │   │
│  │  │ Nom logique     │ Pointeur métadonnées                     │ │   │
│  │  ├─────────────────┼──────────────────────────────────────────┤ │   │
│  │  │ ventes.trans    │ s3://lake/ventes/trans/metadata/v42.json │ │   │
│  │  │ ventes.clients  │ s3://lake/ventes/clients/metadata/v15.json│ │   │
│  │  │ rh.employes     │ s3://lake/rh/employes/metadata/v8.json   │ │   │
│  │  └─────────────────┴──────────────────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Espaces de Noms (Namespaces)                                    │   │
│  │  lakehouse/                                                      │   │
│  │  ├── ventes/                                                     │   │
│  │  │   ├── transactions                                            │   │
│  │  │   └── clients                                                 │   │
│  │  ├── rh/                                                         │   │
│  │  │   └── employes                                                │   │
│  │  └── finance/                                                    │   │
│  │      └── budget                                                  │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      STOCKAGE (S3/ADLS/MinIO)                           │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Fichiers de Métadonnées Iceberg                                 │   │
│  │  v42.metadata.json → manifest-list → manifests → data files     │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

### Architecture Découplée

L'architecture Iceberg découple intentionnellement le catalogue des métadonnées détaillées. Le catalogue ne contient qu'un pointeur vers le fichier `metadata.json` courant ; toute la richesse des métadonnées — schéma, partitionnement, snapshots, manifestes — réside dans les fichiers du stockage objet.

Cette séparation offre plusieurs avantages :

**Portabilité** : Les métadonnées complètes voyagent avec les données. Migrer vers un nouveau catalogue nécessite uniquement de réenregistrer les tables avec leurs pointeurs existants.

**Scalabilité** : Le catalogue reste léger puisqu'il ne stocke que des références. Les métadonnées volumineuses (manifestes pour des millions de fichiers) résident dans le stockage objet élastique.

**Cohérence** : Les mises à jour atomiques du pointeur garantissent que les lecteurs voient toujours un état cohérent, même pendant les écritures concurrentes.

**Interopérabilité** : Plusieurs moteurs de requête accèdent aux mêmes métadonnées sans dépendre d'un format de catalogue propriétaire.

### Interface Catalog d'Iceberg

Apache Iceberg définit une interface Java standardisée que toutes les implémentations de catalogue doivent respecter :

```java
public interface Catalog {
    // Gestion des espaces de noms
    List<Namespace> listNamespaces();
    List<Namespace> listNamespaces(Namespace namespace);
    Map<String, String> loadNamespaceMetadata(Namespace namespace);
    boolean createNamespace(Namespace namespace, Map<String, String> metadata);
    boolean dropNamespace(Namespace namespace);
  
    // Gestion des tables
    List<TableIdentifier> listTables(Namespace namespace);
    Table loadTable(TableIdentifier identifier);
    Table createTable(TableIdentifier identifier, Schema schema, 
                      PartitionSpec spec, Map<String, String> properties);
    boolean dropTable(TableIdentifier identifier, boolean purge);
    void renameTable(TableIdentifier from, TableIdentifier to);
  
    // Transactions
    Transaction newCreateTableTransaction(TableIdentifier identifier, 
                                          Schema schema, PartitionSpec spec);
    Transaction newReplaceTableTransaction(TableIdentifier identifier,
                                           Schema schema, PartitionSpec spec);
}
```

Cette interface standardisée garantit que le code applicatif reste indépendant de l'implémentation de catalogue choisie. Vous pouvez développer avec un catalogue de test local et déployer en production avec un catalogue d'entreprise sans modification de code.

---

## Panorama des Implémentations de Catalogue

### Hive Metastore

Le Hive Metastore (HMS) représente l'implémentation historique, héritée de l'écosystème Hadoop. Malgré son ancienneté, il demeure largement déployé et offre une compatibilité étendue avec les outils existants.

**Architecture** :

```
┌─────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Clients   │────▶│  Hive Metastore │────▶│  Base de données│
│ (Spark,     │     │    Service      │     │  (MySQL/Postgres)│
│  Trino...)  │     │  (Thrift API)   │     │                 │
└─────────────┘     └─────────────────┘     └─────────────────┘
```

**Avantages** :

* Compatibilité avec l'écosystème Hadoop existant
* Maturité et stabilité éprouvées
* Support natif dans Spark, Trino, Hive
* Documentation et expertise abondantes

**Limitations** :

* Performance limitée pour les opérations à haute fréquence
* Modèle de sécurité basique
* Complexité opérationnelle (service + base de données)
* Pas de support natif pour les fonctionnalités Iceberg avancées

**Configuration Spark avec Hive Metastore** :

```python
spark = SparkSession.builder \
    .appName("IcebergHiveMetastore") \
    .config("spark.sql.catalog.lakehouse", "org.apache.iceberg.spark.SparkCatalog") \
    .config("spark.sql.catalog.lakehouse.type", "hive") \
    .config("spark.sql.catalog.lakehouse.uri", "thrift://hive-metastore:9083") \
    .config("spark.sql.catalog.lakehouse.warehouse", "s3://entreprise-lakehouse/warehouse") \
    .config("spark.hadoop.hive.metastore.uris", "thrift://hive-metastore:9083") \
    .enableHiveSupport() \
    .getOrCreate()
```

### REST Catalog

Le REST Catalog représente l'évolution moderne de l'architecture de catalogue Iceberg. Défini par la spécification OpenAPI d'Iceberg, il standardise les interactions via une API HTTP/JSON, facilitant l'interopérabilité et les déploiements distribués.

**Architecture** :

```
┌─────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Clients   │────▶│   REST Catalog  │────▶│  Backend        │
│ (Spark,     │ HTTP│    Service      │     │  (JDBC, DynamoDB,│
│  Flink...)  │     │  (OpenAPI)      │     │   Custom...)    │
└─────────────┘     └─────────────────┘     └─────────────────┘
```

**Avantages** :

* Standard ouvert et bien documenté
* Indépendant du langage et de la plateforme
* Support natif des fonctionnalités Iceberg (views, transactions)
* Flexibilité du backend de stockage
* Facilité d'intégration avec les systèmes d'authentification modernes

**Limitations** :

* Écosystème moins mature que Hive Metastore
* Nécessite le déploiement d'un service dédié
* Moins d'implémentations de production disponibles

**Spécification REST Catalog** :

L'API REST Catalog définit des endpoints standardisés :

| Endpoint                                       | Méthode    | Description                  |
| ---------------------------------------------- | ----------- | ---------------------------- |
| `/v1/config`                                 | GET         | Configuration du catalogue   |
| `/v1/namespaces`                             | GET, POST   | Gestion des espaces de noms  |
| `/v1/namespaces/{ns}`                        | GET, DELETE | Opérations sur un namespace |
| `/v1/namespaces/{ns}/tables`                 | GET, POST   | Liste et création de tables |
| `/v1/namespaces/{ns}/tables/{table}`         | GET, DELETE | Opérations sur une table    |
| `/v1/namespaces/{ns}/tables/{table}/metrics` | POST        | Métriques de scan           |

### AWS Glue Data Catalog

AWS Glue Data Catalog offre une solution entièrement gérée pour les déploiements AWS. Il élimine la charge opérationnelle du Hive Metastore tout en maintenant la compatibilité avec l'écosystème.

**Architecture** :

```
┌─────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Clients   │────▶│   AWS Glue      │────▶│   Stockage      │
│ (EMR, Athena│ SDK │   Data Catalog  │     │   Interne AWS   │
│  Spark...)  │     │   (Géré)        │     │                 │
└─────────────┘     └─────────────────┘     └─────────────────┘
```

**Avantages** :

* Aucune infrastructure à gérer
* Intégration native avec les services AWS (Athena, EMR, Redshift Spectrum)
* Haute disponibilité garantie par AWS
* Intégration IAM pour la sécurité
* Découverte automatique via Glue Crawlers

**Limitations** :

* Verrouillage fournisseur AWS
* Coûts potentiellement élevés à grande échelle
* Limitations sur le nombre d'objets et les requêtes par seconde
* Régions canadiennes limitées (ca-central-1, ca-west-1)

**Configuration Spark avec Glue Catalog** :

```python
spark = SparkSession.builder \
    .appName("IcebergGlueCatalog") \
    .config("spark.sql.catalog.glue_catalog", "org.apache.iceberg.spark.SparkCatalog") \
    .config("spark.sql.catalog.glue_catalog.catalog-impl", 
            "org.apache.iceberg.aws.glue.GlueCatalog") \
    .config("spark.sql.catalog.glue_catalog.warehouse", 
            "s3://entreprise-lakehouse/warehouse") \
    .config("spark.sql.catalog.glue_catalog.io-impl", 
            "org.apache.iceberg.aws.s3.S3FileIO") \
    .config("spark.sql.catalog.glue_catalog.glue.skip-archive", "true") \
    .getOrCreate()
```

### Azure Purview et Unity Catalog

Pour les environnements Azure, plusieurs options de catalogue coexistent, avec des positionnements distincts.

**Azure Purview** : Solution de gouvernance des données offrant découverte, classification et lignage. Purview peut indexer les tables Iceberg mais n'agit pas comme catalogue transactionnel Iceberg natif.

**Databricks Unity Catalog** : Catalogue unifié pour les environnements Databricks, supportant Iceberg via le format Delta Lake avec interopérabilité UniForm. Unity Catalog offre gouvernance fine, lignage et partage de données.

**Configuration avec Unity Catalog** :

```python
# Dans un environnement Databricks avec Unity Catalog
spark.sql("""
    CREATE CATALOG IF NOT EXISTS lakehouse_iceberg
    USING iceberg
""")

spark.sql("""
    CREATE SCHEMA IF NOT EXISTS lakehouse_iceberg.bronze
    LOCATION 's3://entreprise-lakehouse/bronze'
""")
```

### Catalogues Commerciaux

Plusieurs éditeurs proposent des catalogues Iceberg de classe entreprise avec des fonctionnalités avancées.

**Tabular** (acquis par Databricks) : Catalogue SaaS fondé par les créateurs d'Iceberg, offrant gouvernance, collaboration et optimisation automatique.

**Dremio Arctic** : Catalogue géré intégré à la plateforme Dremio, avec support Git-like pour le versionnement des tables et branches de données.

**Starburst Galaxy** : Catalogue cloud de Starburst (contributeur majeur à Trino), unifiant l'accès aux données avec gouvernance intégrée.

| Solution         | Modèle      | Forces                                 | Considérations                 |
| ---------------- | ------------ | -------------------------------------- | ------------------------------- |
| Tabular          | SaaS         | Expertise Iceberg, optimisation        | Acquisition Databricks récente |
| Dremio Arctic    | SaaS/Hybride | Versionnement Git, intégration Dremio | Écosystème Dremio             |
| Starburst Galaxy | SaaS         | Fédération, Trino natif              | Coût, dépendance              |
| Nessie           | Open Source  | Git-like, flexible                     | Maturité, support              |

### Nessie : Le Catalogue Git-like

Project Nessie propose une approche innovante inspirée de Git pour la gestion des catalogues Iceberg. Il permet de créer des branches, de commiter des modifications et de fusionner des changements, apportant les pratiques DevOps au monde des données.

**Concepts clés** :

* **Branches** : Versions parallèles du catalogue pour développement isolé
* **Commits** : Modifications atomiques avec historique complet
* **Tags** : Points de référence immuables pour les releases
* **Merge** : Fusion de branches avec détection de conflits

```
main ─────●─────●─────●─────●─────●─────────●───────
           \                       /
feature ────●─────●─────●─────●────
```

**Cas d'usage** :

* Environnements de développement isolés
* Tests de migrations de schéma avant déploiement
* Rollback transactionnel multi-tables
* Audit et conformité avec historique complet

---

## Implémentation du REST Catalog

### Architecture de Déploiement

Le REST Catalog nécessite le déploiement d'un service HTTP exposant l'API standardisée Iceberg. Plusieurs implémentations de référence existent.

**Architecture de production** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         CLIENTS                                         │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐      │
│  │  Spark  │  │  Trino  │  │  Flink  │  │ Dremio  │  │   API   │      │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘      │
└───────┼────────────┼────────────┼────────────┼────────────┼───────────┘
        │            │            │            │            │
        └────────────┴────────────┴─────┬──────┴────────────┘
                                        │ HTTPS
                                        ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      LOAD BALANCER                                      │
│                   (ALB / Nginx / Traefik)                               │
└────────────────────────────┬────────────────────────────────────────────┘
                             │
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│  REST Catalog   │ │  REST Catalog   │ │  REST Catalog   │
│   Instance 1    │ │   Instance 2    │ │   Instance 3    │
└────────┬────────┘ └────────┬────────┘ └────────┬────────┘
         │                   │                   │
         └───────────────────┼───────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    BACKEND DE PERSISTANCE                               │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐         │
│  │   PostgreSQL    │  │    DynamoDB     │  │      JDBC       │         │
│  │   (Recommandé)  │  │   (AWS natif)   │  │   (Générique)   │         │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Déploiement avec Polaris (Incubateur Apache)

Apache Polaris (anciennement Snowflake Polaris Catalog, donné à la fondation Apache) représente l'implémentation REST Catalog de référence open source.

**Déploiement Kubernetes** :

```yaml
# polaris-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: polaris-catalog
  namespace: lakehouse
spec:
  replicas: 3
  selector:
    matchLabels:
      app: polaris-catalog
  template:
    metadata:
      labels:
        app: polaris-catalog
    spec:
      containers:
        - name: polaris
          image: apache/polaris:latest
          ports:
            - containerPort: 8181
              name: http
            - containerPort: 8182
              name: management
          env:
            - name: POLARIS_PERSISTENCE_TYPE
              value: "eclipse-link"
            - name: POLARIS_PERSISTENCE_ECLIPSELINK_URL
              valueFrom:
                secretKeyRef:
                  name: polaris-db-credentials
                  key: jdbc-url
            - name: POLARIS_PERSISTENCE_ECLIPSELINK_USER
              valueFrom:
                secretKeyRef:
                  name: polaris-db-credentials
                  key: username
            - name: POLARIS_PERSISTENCE_ECLIPSELINK_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: polaris-db-credentials
                  key: password
            - name: POLARIS_DEFAULT_REALM
              value: "entreprise"
            - name: AWS_REGION
              value: "ca-central-1"
          resources:
            requests:
              memory: "2Gi"
              cpu: "1000m"
            limits:
              memory: "4Gi"
              cpu: "2000m"
          livenessProbe:
            httpGet:
              path: /healthcheck
              port: 8182
            initialDelaySeconds: 30
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /healthcheck
              port: 8182
            initialDelaySeconds: 10
            periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: polaris-catalog
  namespace: lakehouse
spec:
  selector:
    app: polaris-catalog
  ports:
    - name: http
      port: 8181
      targetPort: 8181
    - name: management
      port: 8182
      targetPort: 8182
  type: ClusterIP
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: polaris-catalog-ingress
  namespace: lakehouse
  annotations:
    kubernetes.io/ingress.class: nginx
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  tls:
    - hosts:
        - catalog.lakehouse.entreprise.ca
      secretName: polaris-tls
  rules:
    - host: catalog.lakehouse.entreprise.ca
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: polaris-catalog
                port:
                  number: 8181
```

**Configuration de la base de données PostgreSQL** :

```yaml
# postgresql-polaris.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: polaris-db-init
  namespace: lakehouse
data:
  init.sql: |
    CREATE DATABASE polaris_catalog;
    CREATE USER polaris WITH ENCRYPTED PASSWORD '${POLARIS_DB_PASSWORD}';
    GRANT ALL PRIVILEGES ON DATABASE polaris_catalog TO polaris;
  
    \c polaris_catalog
  
    -- Schéma de persistance Polaris
    CREATE SCHEMA IF NOT EXISTS polaris;
    GRANT ALL ON SCHEMA polaris TO polaris;
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: polaris-postgresql
  namespace: lakehouse
spec:
  serviceName: polaris-postgresql
  replicas: 1
  selector:
    matchLabels:
      app: polaris-postgresql
  template:
    metadata:
      labels:
        app: polaris-postgresql
    spec:
      containers:
        - name: postgresql
          image: postgres:15
          ports:
            - containerPort: 5432
          env:
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: polaris-db-credentials
                  key: postgres-password
            - name: PGDATA
              value: /var/lib/postgresql/data/pgdata
          volumeMounts:
            - name: data
              mountPath: /var/lib/postgresql/data
            - name: init-scripts
              mountPath: /docker-entrypoint-initdb.d
          resources:
            requests:
              memory: "1Gi"
              cpu: "500m"
      volumes:
        - name: init-scripts
          configMap:
            name: polaris-db-init
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: ["ReadWriteOnce"]
        storageClassName: gp3
        resources:
          requests:
            storage: 100Gi
```

### Configuration Client

**Configuration Spark** :

```python
from pyspark.sql import SparkSession

spark = SparkSession.builder \
    .appName("IcebergRESTCatalog") \
    .config("spark.sql.extensions", 
            "org.apache.iceberg.spark.extensions.IcebergSparkSessionExtensions") \
    .config("spark.sql.catalog.lakehouse", 
            "org.apache.iceberg.spark.SparkCatalog") \
    .config("spark.sql.catalog.lakehouse.type", "rest") \
    .config("spark.sql.catalog.lakehouse.uri", 
            "https://catalog.lakehouse.entreprise.ca") \
    .config("spark.sql.catalog.lakehouse.warehouse", 
            "s3://entreprise-lakehouse/warehouse") \
    .config("spark.sql.catalog.lakehouse.credential", 
            "${CATALOG_TOKEN}") \
    .config("spark.sql.catalog.lakehouse.token", 
            "${CATALOG_OAUTH_TOKEN}") \
    .config("spark.sql.catalog.lakehouse.io-impl", 
            "org.apache.iceberg.aws.s3.S3FileIO") \
    .config("spark.sql.catalog.lakehouse.s3.endpoint", 
            "s3.ca-central-1.amazonaws.com") \
    .getOrCreate()
```

**Configuration Trino** :

```properties
# etc/catalog/lakehouse.properties
connector.name=iceberg
iceberg.catalog.type=rest
iceberg.rest-catalog.uri=https://catalog.lakehouse.entreprise.ca
iceberg.rest-catalog.warehouse=s3://entreprise-lakehouse/warehouse

# Authentification OAuth2
iceberg.rest-catalog.security=OAUTH2
iceberg.rest-catalog.oauth2.token=file:///etc/trino/secrets/oauth-token

# Configuration S3
iceberg.file-system-type=S3
s3.region=ca-central-1
s3.endpoint=s3.ca-central-1.amazonaws.com
```

**Configuration Flink** :

```sql
CREATE CATALOG lakehouse WITH (
    'type' = 'iceberg',
    'catalog-type' = 'rest',
    'uri' = 'https://catalog.lakehouse.entreprise.ca',
    'warehouse' = 's3://entreprise-lakehouse/warehouse',
    'io-impl' = 'org.apache.iceberg.aws.s3.S3FileIO'
);

USE CATALOG lakehouse;
```

---

## Implémentation AWS Glue Catalog

### Configuration et Déploiement

AWS Glue Data Catalog ne nécessite aucun déploiement d'infrastructure — le service est entièrement géré. La configuration se concentre sur les permissions IAM et les paramètres de connexion.

**Politique IAM pour accès Glue Catalog** :

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "GlueCatalogAccess",
            "Effect": "Allow",
            "Action": [
                "glue:GetDatabase",
                "glue:GetDatabases",
                "glue:CreateDatabase",
                "glue:UpdateDatabase",
                "glue:DeleteDatabase",
                "glue:GetTable",
                "glue:GetTables",
                "glue:CreateTable",
                "glue:UpdateTable",
                "glue:DeleteTable",
                "glue:GetPartition",
                "glue:GetPartitions",
                "glue:BatchGetPartition",
                "glue:CreatePartition",
                "glue:BatchCreatePartition",
                "glue:UpdatePartition",
                "glue:DeletePartition",
                "glue:BatchDeletePartition"
            ],
            "Resource": [
                "arn:aws:glue:ca-central-1:${AWS_ACCOUNT_ID}:catalog",
                "arn:aws:glue:ca-central-1:${AWS_ACCOUNT_ID}:database/lakehouse_*",
                "arn:aws:glue:ca-central-1:${AWS_ACCOUNT_ID}:table/lakehouse_*/*"
            ]
        },
        {
            "Sid": "S3DataAccess",
            "Effect": "Allow",
            "Action": [
                "s3:GetObject",
                "s3:PutObject",
                "s3:DeleteObject",
                "s3:ListBucket"
            ],
            "Resource": [
                "arn:aws:s3:::entreprise-lakehouse",
                "arn:aws:s3:::entreprise-lakehouse/*"
            ]
        }
    ]
}
```

**Configuration Terraform** :

```hcl
# glue-catalog.tf

# Base de données Glue pour le Lakehouse
resource "aws_glue_catalog_database" "lakehouse_bronze" {
  name        = "lakehouse_bronze"
  description = "Couche Bronze du Lakehouse - Données brutes"
  
  location_uri = "s3://entreprise-lakehouse/bronze/"
  
  create_table_default_permission {
    permissions = ["ALL"]
  
    principal {
      data_lake_principal_identifier = "IAM_ALLOWED_PRINCIPALS"
    }
  }
}

resource "aws_glue_catalog_database" "lakehouse_silver" {
  name        = "lakehouse_silver"
  description = "Couche Silver du Lakehouse - Données nettoyées"
  
  location_uri = "s3://entreprise-lakehouse/silver/"
}

resource "aws_glue_catalog_database" "lakehouse_gold" {
  name        = "lakehouse_gold"
  description = "Couche Gold du Lakehouse - Données agrégées"
  
  location_uri = "s3://entreprise-lakehouse/gold/"
}

# Politique de ressources pour le catalogue
resource "aws_glue_resource_policy" "lakehouse_policy" {
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "CrossAccountAccess"
        Effect    = "Allow"
        Principal = {
          AWS = [
            "arn:aws:iam::${var.analytics_account_id}:root"
          ]
        }
        Action = [
          "glue:GetDatabase",
          "glue:GetDatabases",
          "glue:GetTable",
          "glue:GetTables",
          "glue:GetPartition",
          "glue:GetPartitions"
        ]
        Resource = [
          "arn:aws:glue:ca-central-1:${data.aws_caller_identity.current.account_id}:catalog",
          "arn:aws:glue:ca-central-1:${data.aws_caller_identity.current.account_id}:database/lakehouse_*",
          "arn:aws:glue:ca-central-1:${data.aws_caller_identity.current.account_id}:table/lakehouse_*/*"
        ]
      }
    ]
  })
}

# Configuration Lake Formation pour gouvernance avancée
resource "aws_lakeformation_data_lake_settings" "lakehouse" {
  admins = [
    data.aws_iam_role.data_platform_admin.arn
  ]
  
  create_database_default_permissions {
    permissions = ["ALL"]
    principal   = "IAM_ALLOWED_PRINCIPALS"
  }
  
  create_table_default_permissions {
    permissions = ["ALL"]
    principal   = "IAM_ALLOWED_PRINCIPALS"
  }
}
```

### Intégration avec Lake Formation

AWS Lake Formation ajoute une couche de gouvernance fine au-dessus de Glue Catalog, permettant un contrôle d'accès au niveau des colonnes et des lignes.

**Configuration des permissions Lake Formation** :

```hcl
# Permissions au niveau table
resource "aws_lakeformation_permissions" "analysts_read" {
  principal = aws_iam_role.data_analysts.arn
  
  permissions = ["SELECT"]
  
  table {
    database_name = aws_glue_catalog_database.lakehouse_gold.name
    name          = "ventes_agregees"
  }
}

# Permissions au niveau colonne
resource "aws_lakeformation_permissions" "analysts_partial" {
  principal = aws_iam_role.data_analysts.arn
  
  permissions = ["SELECT"]
  
  table_with_columns {
    database_name = aws_glue_catalog_database.lakehouse_silver.name
    name          = "clients"
  
    # Exclusion des colonnes sensibles
    excluded_column_names = [
      "numero_assurance_sociale",
      "date_naissance",
      "adresse_complete"
    ]
  }
}

# Filtrage au niveau ligne
resource "aws_lakeformation_data_cells_filter" "region_quebec" {
  table_data {
    database_name = aws_glue_catalog_database.lakehouse_silver.name
    name          = "transactions"
  
    table_catalog_id = data.aws_caller_identity.current.account_id
  }
  
  name = "filter_quebec_only"
  
  row_filter {
    filter_expression = "province = 'QC'"
  }
}
```

> **Performance**
>
> AWS Glue Catalog impose des limites de débit : 10 requêtes GetTable par seconde par compte. Pour les charges de travail intensives, activez le cache de métadonnées côté client et envisagez de regrouper les opérations. Les requêtes ListTables avec filtrage côté client sont plus efficaces que de multiples GetTable individuels.

---

## Implémentation Nessie

### Concepts et Architecture

Nessie apporte les concepts de contrôle de version Git au monde des catalogues de données. Cette approche révolutionne la gestion des environnements de développement et de test.

**Concepts fondamentaux** :

| Concept Git | Équivalent Nessie | Usage Lakehouse                        |
| ----------- | ------------------ | -------------------------------------- |
| Branch      | Branch             | Environnement isolé (dev, test, prod) |
| Commit      | Commit             | Modification atomique multi-tables     |
| Tag         | Tag                | Version release, point de restauration |
| Merge       | Merge              | Promotion dev → prod                  |
| Cherry-pick | Transplant         | Migration sélective de tables         |

**Architecture Nessie** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         NESSIE SERVER                                   │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Version Store                                                   │   │
│  │  ┌─────────────────────────────────────────────────────────┐    │   │
│  │  │  main ────●────●────●────●────●────●                    │    │   │
│  │  │            \              /                              │    │   │
│  │  │  dev ───────●────●────●──                               │    │   │
│  │  │                    \                                     │    │   │
│  │  │  feature-x ─────────●────●                              │    │   │
│  │  └─────────────────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Content Store (Pointeurs vers métadonnées Iceberg)             │   │
│  │  main:     transactions → s3://.../metadata/v42.json            │   │
│  │  dev:      transactions → s3://.../metadata/v45.json            │   │
│  │  feature-x: transactions → s3://.../metadata/v43.json           │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

### Déploiement Nessie

**Déploiement Kubernetes avec backend JDBC** :

```yaml
# nessie-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nessie-server
  namespace: lakehouse
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nessie
  template:
    metadata:
      labels:
        app: nessie
    spec:
      containers:
        - name: nessie
          image: ghcr.io/projectnessie/nessie:latest
          ports:
            - containerPort: 19120
              name: http
            - containerPort: 9000
              name: management
          env:
            - name: NESSIE_VERSION_STORE_TYPE
              value: "JDBC"
            - name: QUARKUS_DATASOURCE_JDBC_URL
              valueFrom:
                secretKeyRef:
                  name: nessie-db-credentials
                  key: jdbc-url
            - name: QUARKUS_DATASOURCE_USERNAME
              valueFrom:
                secretKeyRef:
                  name: nessie-db-credentials
                  key: username
            - name: QUARKUS_DATASOURCE_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: nessie-db-credentials
                  key: password
            - name: NESSIE_SERVER_DEFAULT_BRANCH
              value: "main"
            - name: NESSIE_SERVER_SEND_STACKTRACE_TO_CLIENT
              value: "false"
          resources:
            requests:
              memory: "1Gi"
              cpu: "500m"
            limits:
              memory: "2Gi"
              cpu: "1000m"
          livenessProbe:
            httpGet:
              path: /q/health/live
              port: 9000
            initialDelaySeconds: 30
          readinessProbe:
            httpGet:
              path: /q/health/ready
              port: 9000
            initialDelaySeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: nessie
  namespace: lakehouse
spec:
  selector:
    app: nessie
  ports:
    - name: http
      port: 19120
      targetPort: 19120
```

### Flux de Travail Git-like

**Configuration Spark avec Nessie** :

```python
spark = SparkSession.builder \
    .appName("IcebergNessie") \
    .config("spark.sql.catalog.nessie", 
            "org.apache.iceberg.spark.SparkCatalog") \
    .config("spark.sql.catalog.nessie.catalog-impl", 
            "org.apache.iceberg.nessie.NessieCatalog") \
    .config("spark.sql.catalog.nessie.uri", 
            "http://nessie:19120/api/v2") \
    .config("spark.sql.catalog.nessie.ref", "main") \
    .config("spark.sql.catalog.nessie.warehouse", 
            "s3://entreprise-lakehouse/warehouse") \
    .config("spark.sql.catalog.nessie.authentication.type", "BEARER") \
    .config("spark.sql.catalog.nessie.authentication.token", 
            "${NESSIE_TOKEN}") \
    .getOrCreate()
```

**Opérations de branchement** :

```python
from pynessie import init as nessie_init
from pynessie.model import Branch, Tag

# Initialisation du client Nessie
client = nessie_init(endpoint="http://nessie:19120/api/v2")

# Création d'une branche de développement
dev_branch = client.create_reference(
    Branch(name="dev-feature-123", hash_=client.get_default_branch().hash_)
)
print(f"Branche créée: {dev_branch.name} à {dev_branch.hash_}")

# Basculement vers la branche de développement dans Spark
spark.conf.set("spark.sql.catalog.nessie.ref", "dev-feature-123")

# Modifications sur la branche dev
spark.sql("""
    ALTER TABLE nessie.bronze.transactions 
    ADD COLUMN nouvelle_colonne STRING
""")

# Validation des modifications
client.commit(
    branch="dev-feature-123",
    message="Ajout colonne nouvelle_colonne à transactions",
    author="data-engineer@entreprise.ca"
)

# Merge vers main après validation
client.merge(
    from_ref="dev-feature-123",
    to_ref="main",
    message="Merge feature-123: nouvelle colonne transactions"
)

# Création d'un tag pour la release
client.create_reference(
    Tag(name="release-2024-01-15", hash_=client.get_reference("main").hash_)
)
```

**Requêtes sur différentes branches** :

```sql
-- Requête sur la branche principale
SELECT * FROM nessie.bronze.transactions VERSION AS OF 'main';

-- Requête sur une branche de développement
SELECT * FROM nessie.bronze.transactions VERSION AS OF 'dev-feature-123';

-- Requête sur un tag spécifique (point dans le temps)
SELECT * FROM nessie.bronze.transactions VERSION AS OF 'release-2024-01-15';

-- Time travel combiné avec branching
SELECT * FROM nessie.bronze.transactions 
VERSION AS OF 'main' 
FOR SYSTEM_TIME AS OF '2024-01-10 10:00:00';
```

> **Migration**
>
> *De* : Environnements dev/test/prod avec copies de données
>
> *Vers* : Branches Nessie partageant les mêmes données sous-jacentes
>
> *Stratégie* : Les branches Nessie ne dupliquent pas les données — elles créent des vues isolées sur les mêmes fichiers. Seules les modifications divergentes créent de nouveaux fichiers. Cette approche réduit drastiquement les coûts de stockage et accélère la création d'environnements.

---

## Fédération Multi-Catalogue

### Architecture de Fédération

Les grandes organisations opèrent souvent plusieurs catalogues : un par département, par région ou par niveau de sensibilité des données. La fédération permet d'unifier l'accès tout en préservant l'autonomie de chaque domaine.

**Architecture fédérée** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      COUCHE DE FÉDÉRATION                               │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Dremio / Trino / Starburst                    │   │
│  │  ┌─────────────────────────────────────────────────────────┐    │   │
│  │  │  Namespace unifié: entreprise.*                          │    │   │
│  │  │    ├── ventes.*     (→ Catalogue Ventes)                │    │   │
│  │  │    ├── rh.*         (→ Catalogue RH)                    │    │   │
│  │  │    ├── finance.*    (→ Catalogue Finance)               │    │   │
│  │  │    └── externe.*    (→ Catalogue Partenaires)           │    │   │
│  │  └─────────────────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└───────────┬───────────────┬───────────────┬───────────────┬─────────────┘
            │               │               │               │
            ▼               ▼               ▼               ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│ Catalogue     │  │ Catalogue     │  │ Catalogue     │  │ Catalogue     │
│ Ventes        │  │ RH            │  │ Finance       │  │ Partenaires   │
│ (Glue)        │  │ (REST)        │  │ (Nessie)      │  │ (Hive)        │
└───────┬───────┘  └───────┬───────┘  └───────┬───────┘  └───────┬───────┘
        │                  │                  │                  │
        ▼                  ▼                  ▼                  ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│ S3 Bucket     │  │ ADLS Gen2     │  │ S3 Bucket     │  │ MinIO         │
│ ca-central-1  │  │ Canada East   │  │ ca-central-1  │  │ On-premise    │
└───────────────┘  └───────────────┘  └───────────────┘  └───────────────┘
```

### Configuration Trino Multi-Catalogue

```properties
# etc/catalog/ventes.properties
connector.name=iceberg
iceberg.catalog.type=glue
iceberg.glue.region=ca-central-1
iceberg.file-system-type=S3

# etc/catalog/rh.properties
connector.name=iceberg
iceberg.catalog.type=rest
iceberg.rest-catalog.uri=https://catalog-rh.entreprise.ca
iceberg.file-system-type=AZURE

# etc/catalog/finance.properties
connector.name=iceberg
iceberg.catalog.type=nessie
iceberg.nessie-catalog.uri=http://nessie-finance:19120/api/v2
iceberg.nessie-catalog.ref=main
iceberg.file-system-type=S3

# etc/catalog/partenaires.properties
connector.name=iceberg
iceberg.catalog.type=hive_metastore
hive.metastore.uri=thrift://hive-partenaires:9083
iceberg.file-system-type=S3
```

**Requêtes fédérées** :

```sql
-- Jointure entre catalogues
SELECT 
    v.transaction_id,
    v.montant,
    c.nom_client,
    e.nom_employe as vendeur
FROM ventes.gold.transactions v
JOIN rh.silver.employes e ON v.employe_id = e.employe_id
JOIN partenaires.bronze.clients c ON v.client_id = c.client_id
WHERE v.date_transaction >= DATE '2024-01-01';

-- Agrégation cross-catalogue
SELECT 
    f.centre_cout,
    SUM(v.montant) as ventes_totales,
    COUNT(DISTINCT v.employe_id) as nb_vendeurs
FROM ventes.gold.transactions v
JOIN finance.silver.centres_cout f ON v.departement_id = f.departement_id
GROUP BY f.centre_cout;
```

### Dremio comme Couche de Fédération

Dremio excelle dans la fédération de sources hétérogènes avec une couche sémantique unifiée.

**Configuration des sources Dremio** :

```sql
-- Source Iceberg sur S3 avec Glue
ALTER SOURCE "Lakehouse Ventes" ADD
LOCATION 's3://entreprise-ventes/warehouse'
FORMAT iceberg
CATALOG glue
REGION 'ca-central-1';

-- Source Iceberg sur ADLS avec REST Catalog
ALTER SOURCE "Lakehouse RH" ADD
LOCATION 'abfss://lakehouse@entrepriserh.dfs.core.windows.net/warehouse'
FORMAT iceberg
CATALOG rest
URI 'https://catalog-rh.entreprise.ca';

-- Création d'un espace virtuel unifié
CREATE VDS "Espace Unifié"."Ventes avec Vendeurs" AS
SELECT 
    t.*,
    e.nom as vendeur_nom,
    e.departement
FROM "Lakehouse Ventes".gold.transactions t
LEFT JOIN "Lakehouse RH".silver.employes e 
    ON t.employe_id = e.employe_id;
```

---

## Sécurité et Gouvernance du Catalogue

### Modèles d'Authentification

**OAuth 2.0 / OIDC** :

L'authentification moderne repose sur OAuth 2.0 et OpenID Connect, permettant l'intégration avec les fournisseurs d'identité d'entreprise (Azure AD, Okta, Keycloak).

```yaml
# Configuration Polaris avec OAuth2
polaris:
  authentication:
    type: oauth2
    oauth2:
      issuer-url: https://login.microsoftonline.com/${TENANT_ID}/v2.0
      client-id: ${POLARIS_CLIENT_ID}
      client-secret: ${POLARIS_CLIENT_SECRET}
      scopes:
        - openid
        - profile
        - api://polaris/.default
      audience: api://polaris
```

**Configuration client Spark avec OAuth** :

```python
spark = SparkSession.builder \
    .config("spark.sql.catalog.lakehouse.credential", 
            "${OAUTH_ACCESS_TOKEN}") \
    .config("spark.sql.catalog.lakehouse.oauth2-server-uri", 
            "https://login.microsoftonline.com/${TENANT_ID}/oauth2/v2.0/token") \
    .config("spark.sql.catalog.lakehouse.oauth2.client-id", 
            "${CLIENT_ID}") \
    .config("spark.sql.catalog.lakehouse.oauth2.client-secret", 
            "${CLIENT_SECRET}") \
    .config("spark.sql.catalog.lakehouse.oauth2.scope", 
            "api://polaris/.default") \
    .getOrCreate()
```

### Contrôle d'Accès Granulaire

**Modèle RBAC (Role-Based Access Control)** :

```sql
-- Création des rôles dans Polaris
CREATE ROLE data_analyst;
CREATE ROLE data_engineer;
CREATE ROLE data_admin;

-- Attribution des privilèges
GRANT USAGE ON CATALOG lakehouse TO ROLE data_analyst;
GRANT SELECT ON NAMESPACE lakehouse.gold TO ROLE data_analyst;

GRANT USAGE ON CATALOG lakehouse TO ROLE data_engineer;
GRANT SELECT, INSERT, UPDATE, DELETE ON NAMESPACE lakehouse.bronze TO ROLE data_engineer;
GRANT SELECT, INSERT, UPDATE, DELETE ON NAMESPACE lakehouse.silver TO ROLE data_engineer;
GRANT SELECT ON NAMESPACE lakehouse.gold TO ROLE data_engineer;

GRANT ALL PRIVILEGES ON CATALOG lakehouse TO ROLE data_admin;

-- Attribution des rôles aux utilisateurs/groupes
GRANT ROLE data_analyst TO GROUP 'Analystes-BI@entreprise.ca';
GRANT ROLE data_engineer TO GROUP 'Data-Engineering@entreprise.ca';
GRANT ROLE data_admin TO USER 'admin@entreprise.ca';
```

**Masquage des données sensibles** :

```sql
-- Politique de masquage pour données PII
CREATE MASKING POLICY mask_courriel AS (val STRING)
RETURNS STRING ->
    CASE
        WHEN CURRENT_ROLE() IN ('data_admin', 'data_engineer') THEN val
        ELSE REGEXP_REPLACE(val, '(.{2}).*@', '$1***@')
    END;

-- Application de la politique
ALTER TABLE lakehouse.silver.clients 
ALTER COLUMN courriel SET MASKING POLICY mask_courriel;

-- Politique de masquage pour NAS
CREATE MASKING POLICY mask_nas AS (val STRING)
RETURNS STRING ->
    CASE
        WHEN CURRENT_ROLE() = 'data_admin' THEN val
        ELSE '***-***-' || RIGHT(val, 3)
    END;
```

### Audit et Traçabilité

**Configuration de l'audit** :

```yaml
# Configuration Polaris pour audit
polaris:
  audit:
    enabled: true
    log-level: INFO
    include-request-body: false
    include-response-body: false
  
    # Export vers CloudWatch (AWS)
    cloudwatch:
      enabled: true
      log-group: /polaris/audit
      region: ca-central-1
  
    # Export vers fichier
    file:
      enabled: true
      path: /var/log/polaris/audit.log
      rotation:
        max-size: 100MB
        max-files: 30
```

**Structure des événements d'audit** :

```json
{
  "timestamp": "2024-01-15T14:30:22.123Z",
  "event_type": "TABLE_READ",
  "catalog": "lakehouse",
  "namespace": "gold",
  "table": "transactions",
  "principal": {
    "type": "USER",
    "name": "analyst@entreprise.ca",
    "roles": ["data_analyst"]
  },
  "action": "SELECT",
  "result": "SUCCESS",
  "client": {
    "application": "Trino",
    "version": "435",
    "ip_address": "10.0.1.45"
  },
  "query_id": "20240115_143022_00042_abcde",
  "rows_returned": 15423,
  "execution_time_ms": 2340
}
```

**Requêtes d'audit** :

```sql
-- Analyse des accès par utilisateur (dernières 24h)
SELECT 
    principal_name,
    COUNT(*) as nb_requetes,
    COUNT(DISTINCT table_name) as tables_accedees,
    SUM(rows_returned) as lignes_totales
FROM audit_logs
WHERE timestamp >= CURRENT_TIMESTAMP - INTERVAL '24' HOUR
GROUP BY principal_name
ORDER BY nb_requetes DESC;

-- Détection d'accès anormaux
SELECT 
    principal_name,
    table_name,
    COUNT(*) as nb_acces,
    AVG(rows_returned) as moy_lignes
FROM audit_logs
WHERE timestamp >= CURRENT_TIMESTAMP - INTERVAL '7' DAY
GROUP BY principal_name, table_name
HAVING COUNT(*) > 100 
   AND AVG(rows_returned) > 100000;
```

---

## Migration entre Catalogues

### Stratégies de Migration

La migration entre catalogues constitue une opération délicate nécessitant une planification rigoureuse. Trois stratégies principales s'offrent à vous :

**Migration par réenregistrement** : Les données restent en place ; seuls les pointeurs sont recréés dans le nouveau catalogue. C'est l'approche la plus rapide et la moins risquée.

**Migration avec déplacement** : Les données sont copiées vers un nouvel emplacement avec enregistrement dans le nouveau catalogue. Nécessaire lors de changement de fournisseur de stockage.

**Migration progressive** : Les tables sont migrées individuellement avec une période de coexistence des deux catalogues.

### Migration Hive Metastore vers REST Catalog

**Phase 1 : Inventaire et préparation**

```python
from pyspark.sql import SparkSession

# Connexion au Hive Metastore existant
spark_hive = SparkSession.builder \
    .config("spark.sql.catalog.hive_catalog", "org.apache.iceberg.spark.SparkCatalog") \
    .config("spark.sql.catalog.hive_catalog.type", "hive") \
    .config("spark.sql.catalog.hive_catalog.uri", "thrift://hive-metastore:9083") \
    .enableHiveSupport() \
    .getOrCreate()

# Inventaire des tables Iceberg
tables_iceberg = spark_hive.sql("""
    SELECT 
        database_name,
        table_name,
        table_location,
        table_properties
    FROM hive_catalog.information_schema.tables
    WHERE table_type = 'ICEBERG'
""").collect()

# Export de l'inventaire
inventaire = []
for table in tables_iceberg:
    metadata_location = spark_hive.sql(f"""
        SELECT current_snapshot_id, metadata_location
        FROM hive_catalog.{table.database_name}.{table.table_name}.metadata_log_entries
        ORDER BY timestamp DESC LIMIT 1
    """).collect()[0]
  
    inventaire.append({
        'database': table.database_name,
        'table': table.table_name,
        'location': table.table_location,
        'metadata_location': metadata_location.metadata_location
    })

print(f"Tables à migrer: {len(inventaire)}")
```

**Phase 2 : Création des namespaces dans le nouveau catalogue**

```python
import requests

REST_CATALOG_URL = "https://catalog.lakehouse.entreprise.ca"
HEADERS = {"Authorization": f"Bearer {OAUTH_TOKEN}", "Content-Type": "application/json"}

# Extraction des namespaces uniques
namespaces = set(t['database'] for t in inventaire)

for ns in namespaces:
    # Création du namespace
    response = requests.post(
        f"{REST_CATALOG_URL}/v1/namespaces",
        headers=HEADERS,
        json={
            "namespace": [ns],
            "properties": {
                "location": f"s3://entreprise-lakehouse/{ns}",
                "migrated_from": "hive_metastore",
                "migration_date": "2024-01-15"
            }
        }
    )
  
    if response.status_code == 200:
        print(f"Namespace {ns} créé")
    elif response.status_code == 409:
        print(f"Namespace {ns} existe déjà")
    else:
        print(f"Erreur création {ns}: {response.text}")
```

**Phase 3 : Enregistrement des tables**

```python
# Connexion au nouveau REST Catalog
spark_rest = SparkSession.builder \
    .config("spark.sql.catalog.lakehouse", "org.apache.iceberg.spark.SparkCatalog") \
    .config("spark.sql.catalog.lakehouse.type", "rest") \
    .config("spark.sql.catalog.lakehouse.uri", REST_CATALOG_URL) \
    .getOrCreate()

for table_info in inventaire:
    db = table_info['database']
    tbl = table_info['table']
    metadata_loc = table_info['metadata_location']
  
    try:
        # Enregistrement via REGISTER TABLE (Iceberg 1.4+)
        spark_rest.sql(f"""
            CALL lakehouse.system.register_table(
                table => '{db}.{tbl}',
                metadata_file => '{metadata_loc}'
            )
        """)
        print(f"Table {db}.{tbl} enregistrée")
      
    except Exception as e:
        print(f"Erreur enregistrement {db}.{tbl}: {e}")
        # Journalisation pour reprise
        log_migration_error(db, tbl, str(e))
```

**Phase 4 : Validation**

```python
# Validation de la migration
def validate_table_migration(db, tbl):
    # Comparaison des métadonnées
    hive_meta = spark_hive.sql(f"""
        SELECT COUNT(*) as cnt, SUM(file_size_in_bytes) as size
        FROM hive_catalog.{db}.{tbl}.files
    """).collect()[0]
  
    rest_meta = spark_rest.sql(f"""
        SELECT COUNT(*) as cnt, SUM(file_size_in_bytes) as size
        FROM lakehouse.{db}.{tbl}.files
    """).collect()[0]
  
    if hive_meta.cnt == rest_meta.cnt and hive_meta.size == rest_meta.size:
        print(f"✓ {db}.{tbl} validée")
        return True
    else:
        print(f"✗ {db}.{tbl} divergente: HMS({hive_meta.cnt}, {hive_meta.size}) vs REST({rest_meta.cnt}, {rest_meta.size})")
        return False

# Validation de toutes les tables
resultats = [validate_table_migration(t['database'], t['table']) for t in inventaire]
print(f"Migration validée: {sum(resultats)}/{len(resultats)} tables")
```

### Migration Glue vers REST Catalog

Pour les organisations souhaitant réduire leur dépendance à AWS, la migration depuis Glue vers un REST Catalog offre plus de flexibilité.

```python
import boto3

# Client Glue
glue = boto3.client('glue', region_name='ca-central-1')

# Récupération des tables Iceberg depuis Glue
def get_glue_iceberg_tables(database_name):
    tables = []
    paginator = glue.get_paginator('get_tables')
  
    for page in paginator.paginate(DatabaseName=database_name):
        for table in page['TableList']:
            # Vérification si c'est une table Iceberg
            params = table.get('Parameters', {})
            if params.get('table_type') == 'ICEBERG':
                tables.append({
                    'database': database_name,
                    'table': table['Name'],
                    'location': table['StorageDescriptor']['Location'],
                    'metadata_location': params.get('metadata_location')
                })
  
    return tables

# Migration
databases = ['lakehouse_bronze', 'lakehouse_silver', 'lakehouse_gold']
all_tables = []

for db in databases:
    all_tables.extend(get_glue_iceberg_tables(db))

print(f"Tables Glue à migrer: {len(all_tables)}")

# Enregistrement dans REST Catalog (même logique que précédemment)
for table_info in all_tables:
    register_in_rest_catalog(table_info)
```

---

## Études de Cas Canadiennes

### Secteur Assurance : Assureur National

> **Étude de cas : Grand assureur pancanadien**
>
> *Secteur* : Assurance vie et santé
>
> *Défi* : Unifier les catalogues de données hérités de 5 acquisitions tout en respectant les exigences de l'AMF (Autorité des marchés financiers) et du BSIF concernant la gouvernance des données. Chaque entité acquise opérait son propre Hive Metastore avec des conventions de nommage incompatibles.
>
> *Solution* : Déploiement d'Apache Polaris comme REST Catalog central dans la région Azure Canada Central. Migration progressive des 5 Hive Metastores sur 8 mois. Implémentation d'une couche de gouvernance avec Azure Purview pour la découverte et le lignage.
>
> *Architecture* :
>
> ```
> ┌─────────────────────────────────────────────────────────────┐
> │                    Polaris (REST Catalog)                   │
> │              catalog.assureur.ca (Canada Central)           │
> └─────────────────────────────┬───────────────────────────────┘
>                               │
>        ┌──────────────────────┼──────────────────────┐
>        ▼                      ▼                      ▼
> ┌─────────────┐        ┌─────────────┐        ┌─────────────┐
> │  Namespace  │        │  Namespace  │        │  Namespace  │
> │  assurance  │        │   sante     │        │  groupe     │
> │   vie       │        │             │        │             │
> └─────────────┘        └─────────────┘        └─────────────┘
> ```
>
> *Résultats* :
>
> * Consolidation de 5 catalogues en 1 avec 2 400 tables migrées
> * Temps de découverte des données réduit de 85%
> * Conformité aux exigences de traçabilité de l'AMF
> * Économies de 400 000$/an en coûts d'infrastructure HMS

### Secteur Minier : Société d'Exploration

> **Étude de cas : Société d'exploration minière québécoise**
>
> *Secteur* : Ressources naturelles et exploitation minière
>
> *Défi* : Gérer les données de 12 sites d'exploration répartis dans des régions éloignées du Nord québécois avec connectivité limitée. Les données géologiques et de forage devaient être consolidées pour l'analyse centrale tout en permettant l'autonomie locale.
>
> *Solution* : Architecture de catalogue distribuée avec Nessie déployé localement sur chaque site (MinIO + Nessie sur serveurs edge) et synchronisation périodique vers le catalogue central hébergé dans le centre de données de Montréal. Les branches Nessie permettent aux géologues de travailler hors ligne puis de synchroniser.
>
> *Particularités* :
>
> * Connectivité satellite intermittente (1-4 Mbps)
> * Synchronisation par batch quotidien via Starlink
> * Résolution de conflits automatisée pour les données de forage
>
>   *Résultats* :
> * Analyse consolidée de 15 ans de données d'exploration
> * Productivité des géologues de terrain augmentée de 40%
> * Résilience totale aux pannes de connectivité

### Secteur Public : Ministère Provincial

> **Étude de cas : Ministère de la Santé provincial**
>
> *Secteur* : Gouvernement provincial
>
> *Défi* : Moderniser l'infrastructure de données tout en maintenant une stricte conformité aux exigences de la Loi 25 (protection des renseignements personnels). Les données de santé ne pouvaient en aucun cas résider hors du Canada ou transiter par des services américains.
>
> *Solution* : Déploiement entièrement sur site avec MinIO pour le stockage et REST Catalog (implémentation maison basée sur la spécification Iceberg) hébergé dans les centres de données gouvernementaux certifiés. Intégration avec le système d'identité gouvernemental (CLIQ) pour l'authentification.
>
> *Architecture de sécurité* :
>
> * Chiffrement AES-256 des données au repos et en transit
> * Audit exhaustif de toutes les opérations catalogue
> * Ségrégation réseau entre environnements
> * Authentification multifacteur obligatoire
>
>   *Résultats* :
> * Conformité Loi 25 validée par audit externe
> * Aucune donnée transitant hors des infrastructures gouvernementales
> * Temps de provisionnement d'accès réduit de 2 semaines à 2 heures

---

## Bonnes Pratiques et Recommandations

### Convention de Nommage

Une convention de nommage cohérente simplifie la découverte et la gouvernance des données à l'échelle de l'entreprise.

**Structure hiérarchique recommandée** :

```
{catalogue}.{couche}.{domaine}_{entité}_{qualificateur}

Exemples:
lakehouse.bronze.ventes_transactions_raw
lakehouse.silver.ventes_transactions_clean
lakehouse.gold.ventes_kpi_quotidien
lakehouse.gold.ventes_cube_regional
```

**Règles de nommage** :

| Élément | Convention                        | Exemples                         |
| --------- | --------------------------------- | -------------------------------- |
| Catalogue | snake_case, environnement         | lakehouse_prod, lakehouse_dev    |
| Namespace | snake_case, couche/domaine        | bronze, silver, gold, ventes, rh |
| Table     | snake_case, entité_qualificateur | transactions_raw, clients_v2     |
| Colonne   | snake_case                        | client_id, date_transaction      |

### Haute Disponibilité

**Configuration REST Catalog HA** :

```yaml
# Déploiement multi-zone avec réplication
apiVersion: apps/v1
kind: Deployment
metadata:
  name: polaris-catalog
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  template:
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchLabels:
                  app: polaris-catalog
              topologyKey: topology.kubernetes.io/zone
      containers:
        - name: polaris
          # ... configuration ...
        
---
# Base de données PostgreSQL en haute disponibilité
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: polaris-db
spec:
  instances: 3
  primaryUpdateStrategy: unsupervised
  
  storage:
    size: 100Gi
    storageClass: gp3
  
  backup:
    barmanObjectStore:
      destinationPath: s3://entreprise-backups/polaris-db
      s3Credentials:
        accessKeyId:
          name: s3-creds
          key: ACCESS_KEY_ID
        secretAccessKey:
          name: s3-creds
          key: SECRET_ACCESS_KEY
      wal:
        compression: gzip
    retentionPolicy: "30d"
```

### Surveillance et Alertes

**Métriques essentielles** :

| Métrique                   | Seuil d'alerte        | Description                     |
| --------------------------- | --------------------- | ------------------------------- |
| catalog_request_latency_p99 | > 500ms               | Latence des requêtes catalogue |
| catalog_error_rate          | > 1%                  | Taux d'erreurs des opérations  |
| catalog_active_connections  | > 80% capacité       | Connexions actives              |
| catalog_commit_conflicts    | > 10/min              | Conflits de commit              |
| catalog_table_count         | Croissance > 10%/jour | Prolifération des tables       |

**Configuration Prometheus** :

```yaml
# prometheus-rules.yaml
groups:
  - name: catalog-alerts
    rules:
      - alert: CatalogLatencyHigh
        expr: histogram_quantile(0.99, rate(catalog_request_duration_seconds_bucket[5m])) > 0.5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Latence catalogue élevée"
          description: "P99 latence > 500ms depuis 5 minutes"
        
      - alert: CatalogErrorRateHigh
        expr: rate(catalog_requests_total{status="error"}[5m]) / rate(catalog_requests_total[5m]) > 0.01
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Taux d'erreurs catalogue critique"
        
      - alert: CatalogCommitConflicts
        expr: rate(catalog_commit_conflicts_total[5m]) > 10/60
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Conflits de commit fréquents"
          description: "Vérifier la charge d'écriture concurrente"
```

### Matrice de Décision

**Choix du type de catalogue** :

| Critère                     | Hive MS    | REST Catalog | Glue       | Nessie     |
| ---------------------------- | ---------- | ------------ | ---------- | ---------- |
| Écosystème existant Hadoop | ★★★★★ | ★★★☆☆   | ★★☆☆☆ | ★★☆☆☆ |
| Multi-moteur moderne         | ★★★☆☆ | ★★★★★   | ★★★★☆ | ★★★★☆ |
| Complexité opérationnelle  | ★★★☆☆ | ★★☆☆☆   | ★☆☆☆☆ | ★★★☆☆ |
| Gouvernance native           | ★★☆☆☆ | ★★★★☆   | ★★★★★ | ★★★☆☆ |
| Versionnement Git-like       | ☆☆☆☆☆ | ☆☆☆☆☆   | ☆☆☆☆☆ | ★★★★★ |
| Verrouillage fournisseur     | ★☆☆☆☆ | ★☆☆☆☆   | ★★★★★ | ★☆☆☆☆ |
| Coût (grande échelle)      | ★★★☆☆ | ★★★☆☆   | ★★☆☆☆ | ★★★★☆ |

**Arbre de décision** :

```
                    ┌─────────────────────────────────┐
                    │  Écosystème principal?          │
                    └─────────────────────────────────┘
                                    │
            ┌───────────────────────┼───────────────────────┐
            ▼                       ▼                       ▼
         [AWS]                  [Azure]                [Multi/On-prem]
            │                       │                       │
            ▼                       ▼                       ▼
    ┌───────────────┐       ┌───────────────┐       ┌───────────────┐
    │ Simplicité    │       │ Intégration   │       │ Versionnement │
    │ prioritaire?  │       │ Fabric?       │       │ requis?       │
    └───────────────┘       └───────────────┘       └───────────────┘
            │                       │                       │
      ┌─────┴─────┐           ┌─────┴─────┐           ┌─────┴─────┐
      ▼           ▼           ▼           ▼           ▼           ▼
    [OUI]       [NON]       [OUI]       [NON]       [OUI]       [NON]
      │           │           │           │           │           │
      ▼           ▼           ▼           ▼           ▼           ▼
    Glue      REST        Unity       REST       Nessie      REST
              Catalog     Catalog     Catalog                Catalog
```

---

## Conclusion

La couche de catalogue constitue le système nerveux central de votre Data Lakehouse Apache Iceberg. Elle transforme une collection de fichiers Parquet en une plateforme de données gouvernée, découvrable et accessible à l'ensemble de l'organisation. Le choix et l'implémentation de cette couche influencent directement l'expérience utilisateur, la sécurité des données et l'agilité opérationnelle de votre plateforme.

L'écosystème des catalogues Iceberg a considérablement mûri ces dernières années. Du traditionnel Hive Metastore aux solutions modernes comme le REST Catalog et Nessie, les organisations disposent désormais d'options adaptées à chaque contexte : services gérés pour la simplicité opérationnelle, solutions open source pour le contrôle total, catalogues Git-like pour les pratiques DevOps avancées.

Pour les organisations canadiennes, les considérations de résidence des données et de conformité réglementaire ajoutent une dimension critique au choix du catalogue. Les études de cas présentées démontrent que des architectures conformes aux exigences les plus strictes — Loi 25, AMF, BSIF — sont réalisables avec les technologies actuelles, que ce soit via des services infonuagiques dans les régions canadiennes ou des déploiements entièrement sur site.

La migration entre catalogues, longtemps perçue comme risquée, devient une opération de routine grâce à l'architecture découplée d'Iceberg. Les métadonnées détaillées résidant dans le stockage objet, seuls les pointeurs doivent être recréés dans le nouveau catalogue. Cette portabilité libère les organisations du verrouillage technologique et leur permet de faire évoluer leur architecture au rythme de leurs besoins.

Le chapitre suivant explore la couche de fédération et de requête, où nous examinerons comment les moteurs analytiques comme Trino et Dremio consomment les métadonnées du catalogue pour exécuter des requêtes performantes sur votre Lakehouse Iceberg.

---

## Résumé

**Rôle du catalogue Iceberg** :

* Registre des tables avec pointeurs vers les métadonnées
* Gestion des espaces de noms pour l'organisation logique
* Contrôle de concurrence pour les commits atomiques
* Interface standardisée indépendante de l'implémentation

**Principales implémentations** :

* **Hive Metastore** : Maturité et compatibilité Hadoop, mais fonctionnalités limitées
* **REST Catalog** : Standard moderne, interopérabilité maximale, déploiement requis
* **AWS Glue** : Service géré, intégration AWS native, verrouillage fournisseur
* **Nessie** : Versionnement Git-like, branches et merges de données

**Sécurité et gouvernance** :

* Authentification OAuth 2.0/OIDC pour intégration IAM
* Contrôle d'accès RBAC au niveau catalogue, namespace et table
* Masquage des données sensibles via politiques
* Audit exhaustif pour traçabilité et conformité

**Migration** :

* Réenregistrement des tables sans déplacement de données
* Validation par comparaison des métadonnées
* Migration progressive avec période de coexistence

**Recommandations** :

* Convention de nommage cohérente : catalogue.couche.domaine_entité
* Haute disponibilité : déploiement multi-zone, base de données répliquée
* Surveillance : latence, taux d'erreurs, conflits de commit
* Choix selon contexte : Glue pour AWS, REST pour multi-moteur, Nessie pour DevOps

---

*Ce chapitre établit la gouvernance des métadonnées de votre Lakehouse. Le chapitre suivant, « Requêtes Fédérées et Moteurs de Consommation », détaille comment exploiter efficacement ces métadonnées pour l'analytique à l'échelle de l'entreprise.*
