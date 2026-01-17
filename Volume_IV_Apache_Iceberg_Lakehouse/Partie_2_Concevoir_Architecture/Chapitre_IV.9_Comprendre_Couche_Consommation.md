
# Chapitre IV.9 - Comprendre la Couche de Consommation

## Introduction

La couche de consommation représente l'aboutissement de votre architecture Data Lakehouse — le moment où les données méticuleusement collectées, transformées et gouvernées deviennent valeur tangible pour l'organisation. Elle définit comment les différents profils d'utilisateurs et d'applications accèdent aux données, les interrogent et les exploitent pour prendre des décisions éclairées. Une couche de consommation bien conçue démocratise l'accès aux données tout en préservant la sécurité et la performance ; mal conçue, elle crée frustration, goulots d'étranglement et risques de gouvernance.

Dans l'écosystème Apache Iceberg, la consommation des données bénéficie de caractéristiques uniques qui transforment l'expérience utilisateur. Le Time Travel permet aux analystes d'explorer les états historiques des données sans intervention technique. L'évolution de schéma garantit que les requêtes existantes continuent de fonctionner malgré les modifications structurelles. Le partitionnement masqué libère les utilisateurs de la complexité technique du stockage. Ces fonctionnalités, invisibles pour la plupart des consommateurs, élèvent considérablement la qualité de l'expérience analytique.

Ce chapitre explore en profondeur les patterns de consommation adaptés à chaque profil d'utilisateur : analystes d'affaires explorant les données via Power BI ou Tableau, scientifiques de données entraînant des modèles depuis leurs notebooks Jupyter, applications métier interrogeant le Lakehouse via API, et ingénieurs de données construisant des pipelines automatisés. Pour chaque profil, nous examinerons les outils appropriés, les configurations optimales et les bonnes pratiques garantissant performance et gouvernance.

L'enjeu dépasse la simple connectivité technique. La couche de consommation façonne la culture de données de votre organisation. Elle détermine qui peut accéder à quelles informations, avec quelle facilité et quelle rapidité. Une démocratisation réussie des données — où chaque collaborateur dispose des informations nécessaires à ses décisions — repose sur une architecture de consommation inclusive, performante et sécurisée.

---

## Taxonomie des Consommateurs de Données

### Profils d'Utilisateurs

La diversité des consommateurs de données nécessite une compréhension fine de leurs besoins, compétences et patterns d'utilisation distincts. Cette taxonomie guide les choix architecturaux et les priorités d'optimisation.

**Analystes d'affaires** : Utilisateurs orientés métier, ils explorent les données pour répondre à des questions business. Leur expertise réside dans la compréhension du domaine plutôt que dans les technologies de données. Ils privilégient les interfaces visuelles, les requêtes ad-hoc et les tableaux de bord interactifs.

**Analystes de données** : À la frontière entre métier et technique, ils maîtrisent SQL et les outils de visualisation avancés. Ils construisent des rapports complexes, effectuent des analyses exploratoires et préparent des datasets pour d'autres consommateurs.

**Scientifiques de données** : Experts en statistiques et apprentissage automatique, ils accèdent aux données via notebooks et frameworks ML. Leurs besoins incluent l'accès à de grands volumes de données brutes, la reproductibilité des expériences et l'intégration avec les pipelines MLOps.

**Ingénieurs de données** : Constructeurs des pipelines de données, ils accèdent programmatiquement au Lakehouse pour l'ingestion, la transformation et l'orchestration. Leur focus est la fiabilité, la performance et l'automatisation.

**Applications et services** : Systèmes automatisés consommant les données via API pour alimenter des fonctionnalités métier — personnalisation, détection de fraude, tableaux de bord embarqués.

**Dirigeants et décideurs** : Consommateurs de synthèses et indicateurs clés, ils accèdent aux données via des rapports consolidés et des alertes automatisées plutôt que par exploration directe.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    PYRAMIDE DES CONSOMMATEURS                           │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│                         ┌───────────────┐                               │
│                         │  Dirigeants   │  Volume: Faible               │
│                         │  (Synthèses)  │  Fréquence: Quotidienne      │
│                         └───────┬───────┘                               │
│                                 │                                       │
│                    ┌────────────┴────────────┐                          │
│                    │   Analystes d'affaires  │  Volume: Moyen           │
│                    │   (Tableaux de bord)    │  Fréquence: Continue     │
│                    └────────────┬────────────┘                          │
│                                 │                                       │
│               ┌─────────────────┴─────────────────┐                     │
│               │      Analystes de données         │  Volume: Élevé      │
│               │      (Requêtes ad-hoc)            │  Fréquence: Continue│
│               └─────────────────┬─────────────────┘                     │
│                                 │                                       │
│          ┌──────────────────────┴──────────────────────┐                │
│          │         Data Scientists / ML Engineers       │  Volume: Très │
│          │         (Notebooks, Feature Stores)          │  élevé        │
│          └──────────────────────┬──────────────────────┘                │
│                                 │                                       │
│     ┌───────────────────────────┴───────────────────────────┐           │
│     │              Applications / Services / API             │  Volume:  │
│     │              (Accès programmatique)                    │  Massif   │
│     └───────────────────────────────────────────────────────┘           │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Patterns d'Accès Caractéristiques

Chaque profil d'utilisateur exhibe des patterns d'accès distincts qui influencent les choix d'optimisation.

| Profil             | Pattern dominant                 | Volume typique | Latence attendue | Fréquence  |
| ------------------ | -------------------------------- | -------------- | ---------------- | ----------- |
| Dirigeants         | Lecture KPI agrégés            | Ko-Mo          | < 1s             | Quotidienne |
| Analystes affaires | Requêtes filtrees, drill-down   | Mo-Go          | < 5s             | Continue    |
| Analystes données | Scans larges, jointures          | Go-To          | < 30s            | Continue    |
| Data Scientists    | Scans complets, échantillonnage | To             | Minutes          | Par projet  |
| Applications       | Lookups par clé, agrégats      | Ko-Mo          | < 100ms          | Temps réel |
| Pipelines          | Lectures/écritures massives     | To             | Minutes-Heures   | Planifiée  |

### Exigences par Profil

**Exigences fonctionnelles** :

| Profil             | Interface préférée   | Fonctionnalités clés               |
| ------------------ | ----------------------- | ------------------------------------ |
| Analystes affaires | GUI (Power BI, Tableau) | Visualisation, filtres, export       |
| Analystes données | SQL + GUI               | Requêtes complexes, joins, CTEs     |
| Data Scientists    | Notebooks + DataFrames  | Accès brut, échantillonnage, ML    |
| Applications       | API REST/gRPC           | Faible latence, haute disponibilité |
| Pipelines          | SDK programmatique      | Transactions, streaming, batch       |

**Exigences non-fonctionnelles** :

| Profil             | Priorité performance | Priorité disponibilité | Tolérance complexité |
| ------------------ | --------------------- | ------------------------ | ---------------------- |
| Dirigeants         | Élevée              | Élevée                 | Aucune                 |
| Analystes affaires | Élevée              | Moyenne                  | Faible                 |
| Analystes données | Moyenne               | Moyenne                  | Moyenne                |
| Data Scientists    | Moyenne               | Moyenne                  | Élevée               |
| Applications       | Critique              | Critique                 | Élevée               |
| Pipelines          | Variable              | Élevée                 | Élevée               |

---

## Consommation pour l'Intelligence d'Affaires

### Intégration Power BI

Microsoft Power BI représente l'outil de BI dominant dans de nombreuses organisations canadiennes, particulièrement celles investies dans l'écosystème Microsoft. L'intégration avec un Lakehouse Iceberg offre plusieurs chemins, chacun avec ses compromis.

**Mode Import** : Les données sont copiées dans le modèle Power BI.

```
Lakehouse Iceberg → Requête → Import Power BI → Modèle en mémoire
```

Avantages :

* Performance de requête optimale (données en mémoire)
* Fonctionnement hors ligne
* Compression et optimisation automatiques

Limitations :

* Données non temps réel (rafraîchissement planifié)
* Limite de taille du modèle (selon licence)
* Duplication des données

**Mode DirectQuery** : Les requêtes sont exécutées directement sur le Lakehouse.

```
Utilisateur → Power BI → DirectQuery → Trino/Dremio → Iceberg
```

Avantages :

* Données toujours à jour
* Pas de limite de volume
* Pas de duplication

Limitations :

* Performance dépendante du Lakehouse
* Fonctionnalités DAX limitées
* Charge sur l'infrastructure

**Mode Direct Lake (Microsoft Fabric)** : Lecture directe des fichiers Parquet/Iceberg sans import.

```
Utilisateur → Power BI → Direct Lake → OneLake → Iceberg (via shortcut)
```

Avantages :

* Performance proche de l'import
* Données quasi temps réel
* Pas de duplication

Limitations :

* Requiert Microsoft Fabric
* Configuration OneLake nécessaire

**Configuration Power BI avec Trino** :

```
# Chaîne de connexion ODBC
Driver={Trino ODBC Driver};
Host=trino.lakehouse.entreprise.ca;
Port=443;
Catalog=lakehouse;
Schema=gold;
SSL=1;
AuthenticationType=OAuth2;
OAuth2ClientId=<CLIENT_ID>;
OAuth2ClientSecret=<CLIENT_SECRET>;
OAuth2TokenEndpoint=https://login.microsoftonline.com/<TENANT>/oauth2/v2.0/token;
```

**Optimisation des rapports Power BI** :

```sql
-- Vue optimisée pour tableau de bord exécutif
CREATE VIEW lakehouse.semantic.kpi_ventes_quotidien AS
SELECT 
    date_transaction,
    region,
    categorie_produit,
    SUM(montant) as chiffre_affaires,
    COUNT(DISTINCT client_id) as clients_uniques,
    COUNT(*) as nombre_transactions,
    AVG(montant) as panier_moyen
FROM lakehouse.gold.transactions
WHERE date_transaction >= CURRENT_DATE - INTERVAL '365' DAY
GROUP BY date_transaction, region, categorie_produit;

-- Statistiques pour optimisation des requêtes
ANALYZE lakehouse.semantic.kpi_ventes_quotidien;
```

### Intégration Tableau

Tableau offre une expérience d'exploration visuelle puissante avec plusieurs options de connexion au Lakehouse Iceberg.

**Connexion Tableau via JDBC** :

```xml
<!-- tableau-connection.tdc -->
<connection-customization class='genericjdbc' enabled='true'>
    <vendor name='trino' />
    <driver name='trino' />
    <customizations>
        <customization name='CAP_CREATE_TEMP_TABLES' value='yes' />
        <customization name='CAP_QUERY_BOOLEXPR_TO_INTEXPR' value='no' />
        <customization name='CAP_QUERY_GROUP_BY_ALIAS' value='yes' />
        <customization name='CAP_QUERY_SUBQUERIES' value='yes' />
        <customization name='CAP_QUERY_SUBQUERIES_WITH_TOP' value='yes' />
        <customization name='SQL_AGGREGATE_FUNCTION_STDDEV' value='STDDEV_SAMP' />
        <customization name='SQL_AGGREGATE_FUNCTION_VAR' value='VAR_SAMP' />
    </customizations>
</connection-customization>
```

**Extraits Tableau optimisés** :

Pour les analyses répétitives, les extraits Tableau (.hyper) offrent des performances supérieures :

```sql
-- Requête source pour extrait
SELECT 
    t.transaction_id,
    t.date_transaction,
    t.montant,
    t.quantite,
    c.segment_client,
    c.region,
    p.categorie,
    p.sous_categorie
FROM lakehouse.gold.transactions t
JOIN lakehouse.gold.dim_clients c ON t.client_id = c.client_id
JOIN lakehouse.gold.dim_produits p ON t.produit_id = p.produit_id
WHERE t.date_transaction >= DATE_ADD('year', -2, CURRENT_DATE)
```

**Tableau Server avec connexions publiées** :

```yaml
# Configuration datasource Tableau Server
datasource:
  name: "Lakehouse - Ventes Gold"
  connection:
    class: "genericjdbc"
    dbname: "lakehouse"
    schema: "gold"
    server: "trino.lakehouse.entreprise.ca"
    port: "443"
    authentication: "oauth"
    oauth-config:
      client-id: "${TABLEAU_CLIENT_ID}"
      auth-uri: "https://login.microsoftonline.com/${TENANT}/oauth2/v2.0/authorize"
      token-uri: "https://login.microsoftonline.com/${TENANT}/oauth2/v2.0/token"
  refresh-schedule:
    frequency: "daily"
    time: "06:00"
    timezone: "America/Toronto"
```

### Self-Service Analytics

L'analytique en libre-service permet aux utilisateurs métier d'explorer les données sans dépendance constante envers les équipes techniques. Cette démocratisation repose sur une préparation soignée de la couche sémantique.

**Principes de conception self-service** :

1. **Tables larges dénormalisées** : Réduire les jointures pour les utilisateurs non techniques
2. **Nommage métier** : Utiliser des noms compréhensibles (`chiffre_affaires` plutôt que `amt_ttc`)
3. **Documentation intégrée** : Descriptions et exemples dans les métadonnées
4. **Métriques pré-calculées** : Éviter les calculs complexes côté utilisateur
5. **Filtres par défaut** : Limiter automatiquement aux données pertinentes

**Création d'une couche sémantique self-service** :

```sql
-- Table analytique optimisée pour self-service
CREATE TABLE lakehouse.semantic.analyse_ventes (
    -- Dimensions temporelles
    date_transaction DATE COMMENT 'Date de la transaction',
    annee INT COMMENT 'Année (ex: 2024)',
    trimestre VARCHAR COMMENT 'Trimestre (ex: T1 2024)',
    mois VARCHAR COMMENT 'Mois (ex: Janvier 2024)',
    semaine_annee INT COMMENT 'Numéro de semaine dans l''année',
    jour_semaine VARCHAR COMMENT 'Jour de la semaine (ex: Lundi)',
    est_fin_semaine BOOLEAN COMMENT 'Vrai si samedi ou dimanche',
  
    -- Dimensions client
    client_id BIGINT COMMENT 'Identifiant unique du client',
    segment_client VARCHAR COMMENT 'Segment marketing (Premium, Standard, Économique)',
    anciennete_client VARCHAR COMMENT 'Ancienneté (Nouveau, 1-2 ans, 3-5 ans, 5+ ans)',
    province_client VARCHAR COMMENT 'Province de résidence',
    ville_client VARCHAR COMMENT 'Ville de résidence',
  
    -- Dimensions produit
    categorie_produit VARCHAR COMMENT 'Catégorie principale du produit',
    sous_categorie VARCHAR COMMENT 'Sous-catégorie du produit',
    marque VARCHAR COMMENT 'Marque du produit',
  
    -- Dimensions géographiques
    region_vente VARCHAR COMMENT 'Région de vente',
    magasin VARCHAR COMMENT 'Nom du magasin ou "En ligne"',
    canal VARCHAR COMMENT 'Canal de vente (Magasin, Web, Mobile)',
  
    -- Métriques
    chiffre_affaires DECIMAL(15,2) COMMENT 'Montant total de la vente en CAD',
    quantite INT COMMENT 'Nombre d''unités vendues',
    marge_brute DECIMAL(15,2) COMMENT 'Marge brute en CAD',
    taux_marge DECIMAL(5,2) COMMENT 'Taux de marge en pourcentage',
    cout_acquisition DECIMAL(10,2) COMMENT 'Coût d''acquisition attribué'
)
USING iceberg
PARTITIONED BY (annee, trimestre)
COMMENT 'Table analytique des ventes pour exploration self-service. 
Mise à jour quotidienne à 6h00. Données disponibles depuis 2020.';

-- Ajout de propriétés pour documentation
ALTER TABLE lakehouse.semantic.analyse_ventes SET TBLPROPERTIES (
    'owner' = 'equipe-analytics@entreprise.ca',
    'data-steward' = 'marie.tremblay@entreprise.ca',
    'refresh-frequency' = 'daily',
    'last-refresh' = '2024-01-15T06:00:00Z',
    'row-count' = '45000000',
    'documentation-url' = 'https://wiki.entreprise.ca/data/ventes'
);
```

> **Performance**
>
> Les tables self-service dénormalisées augmentent le volume de stockage (redondance) mais réduisent drastiquement la complexité des requêtes utilisateur et améliorent les performances. Pour une table de 50 millions de transactions, la dénormalisation typique augmente le stockage de 30-50% mais réduit le temps de requête moyen de 60-80% en éliminant les jointures.

---

## Consommation pour la Science des Données

### Accès Notebook et DataFrame

Les scientifiques de données privilégient l'accès programmatique via notebooks (Jupyter, Databricks, Google Colab) et frameworks DataFrame (pandas, Polars, PySpark). L'intégration avec Iceberg doit supporter ces workflows tout en garantissant la reproductibilité.

**Configuration PyIceberg pour notebooks** :

```python
# Installation
# pip install pyiceberg[s3,pyarrow]

from pyiceberg.catalog import load_catalog
import pyarrow as pa

# Configuration du catalogue
catalog = load_catalog(
    "lakehouse",
    **{
        "type": "rest",
        "uri": "https://catalog.lakehouse.entreprise.ca",
        "credential": os.environ["ICEBERG_TOKEN"],
        "warehouse": "s3://entreprise-lakehouse/warehouse",
        "s3.region": "ca-central-1"
    }
)

# Chargement d'une table
table = catalog.load_table("gold.transactions")

# Conversion en DataFrame Arrow (efficace pour grandes tables)
arrow_table = table.scan(
    row_filter="date_transaction >= '2024-01-01'",
    selected_fields=["transaction_id", "client_id", "montant", "date_transaction"]
).to_arrow()

# Conversion en pandas
df = arrow_table.to_pandas()

# Ou directement en Polars (plus performant pour grandes tables)
import polars as pl
df_polars = pl.from_arrow(arrow_table)
```

**Accès PySpark pour volumes importants** :

```python
from pyspark.sql import SparkSession

spark = SparkSession.builder \
    .appName("DataScienceWorkbench") \
    .config("spark.sql.extensions", 
            "org.apache.iceberg.spark.extensions.IcebergSparkSessionExtensions") \
    .config("spark.sql.catalog.lakehouse", 
            "org.apache.iceberg.spark.SparkCatalog") \
    .config("spark.sql.catalog.lakehouse.type", "rest") \
    .config("spark.sql.catalog.lakehouse.uri", 
            "https://catalog.lakehouse.entreprise.ca") \
    .config("spark.sql.catalog.lakehouse.warehouse", 
            "s3://entreprise-lakehouse/warehouse") \
    .getOrCreate()

# Lecture avec filtrage pushdown
df_train = spark.read.table("lakehouse.gold.transactions") \
    .filter("date_transaction >= '2023-01-01'") \
    .filter("date_transaction < '2024-01-01'")

# Échantillonnage pour exploration
df_sample = df_train.sample(fraction=0.01, seed=42)

# Conversion en pandas pour ML local
pdf = df_sample.toPandas()
```

**Reproductibilité avec Time Travel** :

```python
# Lecture d'un snapshot spécifique pour reproductibilité
snapshot_id = 1234567890123456789

# Via PyIceberg
table = catalog.load_table("gold.transactions")
arrow_table = table.scan(snapshot_id=snapshot_id).to_arrow()

# Via Spark
df_reproducible = spark.read \
    .option("snapshot-id", snapshot_id) \
    .table("lakehouse.gold.transactions")

# Ou via timestamp
df_at_time = spark.read \
    .option("as-of-timestamp", "2024-01-01T00:00:00Z") \
    .table("lakehouse.gold.transactions")
```

### Feature Stores et Iceberg

Les Feature Stores centralisent la gestion des features ML, garantissant cohérence entre entraînement et inférence. Apache Iceberg s'intègre naturellement comme backend de stockage pour ces systèmes.

**Architecture Feature Store avec Iceberg** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        FEATURE STORE                                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  REGISTRE DES FEATURES                                          │   │
│  │  ┌─────────────────┬─────────────────┬─────────────────┐        │   │
│  │  │ Feature Group   │ Entity          │ Description     │        │   │
│  │  ├─────────────────┼─────────────────┼─────────────────┤        │   │
│  │  │ client_profil   │ client_id       │ Profil client   │        │   │
│  │  │ client_activite │ client_id       │ Métriques       │        │   │
│  │  │ produit_stats   │ produit_id      │ Stats produit   │        │   │
│  │  └─────────────────┴─────────────────┴─────────────────┘        │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                │                                        │
│                                ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  STOCKAGE ICEBERG                                               │   │
│  │  lakehouse.features.client_profil (partitionné par date)        │   │
│  │  lakehouse.features.client_activite (partitionné par date)      │   │
│  │  lakehouse.features.produit_stats (partitionné par date)        │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                │                                        │
│              ┌─────────────────┼─────────────────┐                     │
│              ▼                 ▼                 ▼                      │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐               │
│  │ Entraînement  │  │  Inférence    │  │  Inférence    │               │
│  │    Batch      │  │    Batch      │  │  Temps Réel   │               │
│  │  (Time Travel)│  │  (Dernière)   │  │   (Cache)     │               │
│  └───────────────┘  └───────────────┘  └───────────────┘               │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

**Implémentation avec Feast** :

```python
# feature_store.yaml
project: lakehouse_ml
registry: s3://entreprise-lakehouse/feast/registry.db
provider: local
offline_store:
  type: spark
  spark_conf:
    spark.sql.catalog.lakehouse: org.apache.iceberg.spark.SparkCatalog
    spark.sql.catalog.lakehouse.type: rest
    spark.sql.catalog.lakehouse.uri: https://catalog.lakehouse.entreprise.ca
online_store:
  type: redis
  connection_string: redis://redis-features:6379
```

```python
# features.py
from feast import Entity, Feature, FeatureView, FileSource
from feast.types import Float32, Int64
from datetime import timedelta

# Entité client
client = Entity(
    name="client_id",
    description="Identifiant unique du client"
)

# Source Iceberg pour features client
client_features_source = FileSource(
    name="client_features_source",
    path="iceberg://lakehouse.features.client_profil",
    timestamp_field="date_feature"
)

# Feature View
client_features = FeatureView(
    name="client_features",
    entities=[client],
    ttl=timedelta(days=1),
    schema=[
        Feature(name="total_achats_30j", dtype=Float32),
        Feature(name="nb_transactions_30j", dtype=Int64),
        Feature(name="panier_moyen_30j", dtype=Float32),
        Feature(name="jours_depuis_dernier_achat", dtype=Int64),
        Feature(name="score_fidelite", dtype=Float32),
    ],
    source=client_features_source,
    online=True
)
```

**Lecture des features pour entraînement** :

```python
from feast import FeatureStore
import pandas as pd

store = FeatureStore(repo_path=".")

# Entités pour lesquelles récupérer les features
entity_df = pd.DataFrame({
    "client_id": [1001, 1002, 1003, 1004, 1005],
    "event_timestamp": pd.to_datetime(["2024-01-15"] * 5)
})

# Récupération des features (utilise Time Travel Iceberg)
training_df = store.get_historical_features(
    entity_df=entity_df,
    features=[
        "client_features:total_achats_30j",
        "client_features:nb_transactions_30j",
        "client_features:panier_moyen_30j",
        "client_features:score_fidelite"
    ]
).to_df()
```

### MLOps et Intégration Continue

L'intégration d'Iceberg dans les pipelines MLOps garantit la reproductibilité, la traçabilité et l'automatisation des workflows de machine learning.

**Pipeline MLOps avec versionnement Iceberg** :

```python
import mlflow
from mlflow.tracking import MlflowClient

# Configuration MLflow
mlflow.set_tracking_uri("https://mlflow.lakehouse.entreprise.ca")
mlflow.set_experiment("prediction_churn")

def train_model(snapshot_id: int):
    """Entraînement reproductible avec snapshot Iceberg."""
  
    with mlflow.start_run():
        # Logging du snapshot utilisé pour reproductibilité
        mlflow.log_param("iceberg_snapshot_id", snapshot_id)
        mlflow.log_param("iceberg_table", "lakehouse.gold.client_features")
      
        # Chargement des données depuis snapshot spécifique
        df_train = spark.read \
            .option("snapshot-id", snapshot_id) \
            .table("lakehouse.gold.client_features") \
            .toPandas()
      
        mlflow.log_param("training_rows", len(df_train))
      
        # Préparation des features
        X = df_train[feature_columns]
        y = df_train["churned"]
      
        # Entraînement
        model = XGBClassifier(**hyperparams)
        model.fit(X, y)
      
        # Évaluation
        predictions = model.predict(X_test)
        accuracy = accuracy_score(y_test, predictions)
        mlflow.log_metric("accuracy", accuracy)
      
        # Sauvegarde du modèle
        mlflow.xgboost.log_model(model, "model")
      
        return model

# Récupération du snapshot courant pour reproductibilité
table = catalog.load_table("gold.client_features")
current_snapshot = table.current_snapshot().snapshot_id

# Entraînement
model = train_model(current_snapshot)
```

**Automatisation CI/CD pour features** :

```yaml
# .github/workflows/feature-pipeline.yml
name: Feature Pipeline

on:
  schedule:
    - cron: '0 6 * * *'  # Quotidien à 6h
  workflow_dispatch:

jobs:
  compute-features:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
    
      - name: Configure AWS
        uses: aws-actions/configure-aws-credentials@v2
        with:
          role-to-assume: ${{ secrets.AWS_ROLE_ARN }}
          aws-region: ca-central-1
    
      - name: Run Feature Computation
        run: |
          spark-submit \
            --master k8s://https://k8s-api.lakehouse.entreprise.ca \
            --conf spark.kubernetes.container.image=features:${{ github.sha }} \
            compute_features.py
    
      - name: Validate Features
        run: |
          python validate_features.py \
            --table lakehouse.features.client_profil \
            --checks null_check,range_check,freshness_check
    
      - name: Update Feature Store Registry
        run: |
          feast apply
          feast materialize-incremental $(date +%Y-%m-%dT%H:%M:%S)
```

---

## Consommation Applicative

### APIs de Données

Les applications modernes nécessitent un accès programmatique aux données du Lakehouse via des APIs bien définies. Plusieurs patterns architecturaux répondent à ces besoins.

**Architecture API de données** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      APPLICATIONS CLIENTES                              │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐      │
│  │   Web   │  │  Mobile │  │Microservice│ │  BI    │  │ Partner │      │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘      │
└───────┼────────────┼────────────┼────────────┼────────────┼───────────┘
        │            │            │            │            │
        └────────────┴────────────┴─────┬──────┴────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      API GATEWAY                                        │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  • Authentification (OAuth2 / API Keys)                         │   │
│  │  • Rate Limiting                                                 │   │
│  │  • Caching                                                       │   │
│  │  • Routing                                                       │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              ▼                  ▼                  ▼
┌───────────────────┐  ┌───────────────────┐  ┌───────────────────┐
│   Data API        │  │   Query API       │  │   Streaming API   │
│   (REST/GraphQL)  │  │   (SQL passthrough)│  │   (WebSocket)     │
│                   │  │                   │  │                   │
│   GET /clients/123│  │   POST /query     │  │   WS /events      │
│   GET /kpi/ventes │  │   {sql: "..."}    │  │   subscribe()     │
└─────────┬─────────┘  └─────────┬─────────┘  └─────────┬─────────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      COUCHE DE FÉDÉRATION                               │
│                      (Trino / Dremio)                                   │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      LAKEHOUSE ICEBERG                                  │
└─────────────────────────────────────────────────────────────────────────┘
```

**API REST avec FastAPI** :

```python
from fastapi import FastAPI, Depends, HTTPException, Query
from fastapi.security import OAuth2PasswordBearer
from pydantic import BaseModel
from typing import List, Optional
import trino

app = FastAPI(title="Lakehouse Data API", version="1.0.0")
oauth2_scheme = OAuth2PasswordBearer(tokenUrl="token")

# Pool de connexions Trino
def get_trino_connection():
    return trino.dbapi.connect(
        host="trino.lakehouse.entreprise.ca",
        port=443,
        user="api_service",
        catalog="lakehouse",
        schema="gold",
        http_scheme="https",
        auth=trino.auth.OAuth2Authentication()
    )

# Modèles Pydantic
class ClientSummary(BaseModel):
    client_id: int
    nom: str
    segment: str
    total_achats: float
    derniere_transaction: str

class KPIVentes(BaseModel):
    date: str
    chiffre_affaires: float
    nombre_transactions: int
    panier_moyen: float

# Endpoints
@app.get("/clients/{client_id}", response_model=ClientSummary)
async def get_client(client_id: int, token: str = Depends(oauth2_scheme)):
    """Récupère le résumé d'un client spécifique."""
    conn = get_trino_connection()
    cursor = conn.cursor()
  
    cursor.execute("""
        SELECT client_id, nom, segment, total_achats, 
               CAST(derniere_transaction AS VARCHAR) as derniere_transaction
        FROM lakehouse.gold.vue_client_360
        WHERE client_id = ?
    """, (client_id,))
  
    row = cursor.fetchone()
    if not row:
        raise HTTPException(status_code=404, detail="Client non trouvé")
  
    return ClientSummary(
        client_id=row[0],
        nom=row[1],
        segment=row[2],
        total_achats=row[3],
        derniere_transaction=row[4]
    )

@app.get("/kpi/ventes", response_model=List[KPIVentes])
async def get_kpi_ventes(
    date_debut: str = Query(..., description="Date début (YYYY-MM-DD)"),
    date_fin: str = Query(..., description="Date fin (YYYY-MM-DD)"),
    region: Optional[str] = Query(None, description="Filtrer par région"),
    token: str = Depends(oauth2_scheme)
):
    """Récupère les KPI de ventes pour une période."""
    conn = get_trino_connection()
    cursor = conn.cursor()
  
    query = """
        SELECT 
            CAST(date_transaction AS VARCHAR) as date,
            SUM(montant) as chiffre_affaires,
            COUNT(*) as nombre_transactions,
            AVG(montant) as panier_moyen
        FROM lakehouse.gold.transactions
        WHERE date_transaction BETWEEN ? AND ?
    """
    params = [date_debut, date_fin]
  
    if region:
        query += " AND region = ?"
        params.append(region)
  
    query += " GROUP BY date_transaction ORDER BY date_transaction"
  
    cursor.execute(query, params)
  
    return [
        KPIVentes(
            date=row[0],
            chiffre_affaires=row[1],
            nombre_transactions=row[2],
            panier_moyen=row[3]
        )
        for row in cursor.fetchall()
    ]
```

**API GraphQL pour requêtes flexibles** :

```python
import strawberry
from strawberry.fastapi import GraphQLRouter
from typing import List, Optional

@strawberry.type
class Transaction:
    transaction_id: int
    client_id: int
    montant: float
    date_transaction: str
    produit: str

@strawberry.type
class ClientStats:
    client_id: int
    nom: str
    total_achats: float
    nombre_transactions: int
    transactions: List[Transaction]

@strawberry.type
class Query:
    @strawberry.field
    async def client(self, client_id: int) -> Optional[ClientStats]:
        # Implémentation avec requête Trino
        pass
  
    @strawberry.field
    async def transactions(
        self,
        date_debut: str,
        date_fin: str,
        client_id: Optional[int] = None,
        limit: int = 100
    ) -> List[Transaction]:
        # Implémentation avec requête Trino
        pass

schema = strawberry.Schema(query=Query)
graphql_app = GraphQLRouter(schema)
app.include_router(graphql_app, prefix="/graphql")
```

### Accès Temps Réel et Faible Latence

Certaines applications requièrent des latences sub-seconde incompatibles avec les requêtes directes au Lakehouse. Des patterns de cache et de pré-calcul répondent à ces exigences.

**Architecture cache multi-niveaux** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      APPLICATION                                        │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      CACHE NIVEAU 1 (In-Memory)                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Redis / Memcached                                               │   │
│  │  TTL: 1-5 minutes                                                │   │
│  │  Hit rate cible: 90%+                                            │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │ Cache miss
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      CACHE NIVEAU 2 (Matérialisé)                      │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Tables Iceberg pré-agrégées                                     │   │
│  │  Rafraîchissement: 5-15 minutes                                  │   │
│  │  Dremio Reflections / Vues matérialisées                        │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │ Cache miss
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      LAKEHOUSE (Source de vérité)                      │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Tables Iceberg Gold                                             │   │
│  │  Requêtes ad-hoc                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

**Implémentation avec Redis** :

```python
import redis
import json
from functools import wraps
import hashlib

redis_client = redis.Redis(
    host='redis-cache.lakehouse.entreprise.ca',
    port=6379,
    db=0,
    decode_responses=True
)

def cache_query(ttl_seconds: int = 300):
    """Décorateur pour mise en cache des requêtes."""
    def decorator(func):
        @wraps(func)
        async def wrapper(*args, **kwargs):
            # Génération de la clé de cache
            cache_key = f"query:{func.__name__}:{hashlib.md5(str(kwargs).encode()).hexdigest()}"
          
            # Vérification du cache
            cached = redis_client.get(cache_key)
            if cached:
                return json.loads(cached)
          
            # Exécution de la requête
            result = await func(*args, **kwargs)
          
            # Mise en cache
            redis_client.setex(cache_key, ttl_seconds, json.dumps(result))
          
            return result
        return wrapper
    return decorator

@cache_query(ttl_seconds=60)
async def get_kpi_temps_reel(region: str = None):
    """KPI avec cache de 60 secondes."""
    conn = get_trino_connection()
    cursor = conn.cursor()
  
    query = """
        SELECT 
            SUM(montant) as ca_jour,
            COUNT(*) as nb_transactions,
            COUNT(DISTINCT client_id) as clients_uniques
        FROM lakehouse.gold.transactions
        WHERE date_transaction = CURRENT_DATE
    """
    if region:
        query += f" AND region = '{region}'"
  
    cursor.execute(query)
    row = cursor.fetchone()
  
    return {
        "ca_jour": float(row[0] or 0),
        "nb_transactions": row[1],
        "clients_uniques": row[2]
    }
```

**Tables pré-agrégées pour performances** :

```sql
-- Table de KPI temps réel rafraîchie toutes les 5 minutes
CREATE TABLE lakehouse.cache.kpi_temps_reel (
    timestamp_calcul TIMESTAMP,
    region VARCHAR,
    ca_jour DECIMAL(15,2),
    nb_transactions BIGINT,
    clients_uniques BIGINT,
    ca_heure_precedente DECIMAL(15,2),
    tendance VARCHAR
)
USING iceberg
PARTITIONED BY (days(timestamp_calcul));

-- Procédure de rafraîchissement
MERGE INTO lakehouse.cache.kpi_temps_reel target
USING (
    SELECT 
        CURRENT_TIMESTAMP as timestamp_calcul,
        region,
        SUM(montant) as ca_jour,
        COUNT(*) as nb_transactions,
        COUNT(DISTINCT client_id) as clients_uniques,
        SUM(CASE WHEN date_transaction >= CURRENT_TIMESTAMP - INTERVAL '1' HOUR 
                 THEN montant ELSE 0 END) as ca_heure_precedente,
        CASE 
            WHEN SUM(montant) > LAG(SUM(montant)) OVER (PARTITION BY region ORDER BY 1) 
            THEN 'hausse'
            ELSE 'baisse'
        END as tendance
    FROM lakehouse.gold.transactions
    WHERE date_transaction = CURRENT_DATE
    GROUP BY region
) source
ON target.region = source.region 
   AND DATE(target.timestamp_calcul) = CURRENT_DATE
WHEN MATCHED THEN UPDATE SET *
WHEN NOT MATCHED THEN INSERT *;
```

### Microservices et Architecture Événementielle

L'intégration du Lakehouse dans une architecture de microservices requiert des patterns spécifiques pour maintenir la cohérence et la performance.

**Pattern CQRS avec Lakehouse** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      COMMANDES (Écriture)                              │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Service Commande → Kafka → Pipeline Ingestion → Iceberg        │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                      REQUÊTES (Lecture)                                │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Service Requête → Cache Redis → Trino → Iceberg                │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

**Service de données événementiel** :

```python
from kafka import KafkaConsumer, KafkaProducer
import json

class DataChangeNotifier:
    """Notifie les services des changements de données."""
  
    def __init__(self):
        self.producer = KafkaProducer(
            bootstrap_servers=['kafka:9092'],
            value_serializer=lambda v: json.dumps(v).encode('utf-8')
        )
  
    def notify_table_update(self, table: str, snapshot_id: int, rows_affected: int):
        """Publie un événement de mise à jour de table."""
        event = {
            "event_type": "TABLE_UPDATED",
            "table": table,
            "snapshot_id": snapshot_id,
            "rows_affected": rows_affected,
            "timestamp": datetime.utcnow().isoformat()
        }
        self.producer.send('lakehouse.events', value=event)

class DataChangeListener:
    """Écoute les changements de données pour invalidation de cache."""
  
    def __init__(self):
        self.consumer = KafkaConsumer(
            'lakehouse.events',
            bootstrap_servers=['kafka:9092'],
            value_deserializer=lambda m: json.loads(m.decode('utf-8'))
        )
        self.redis = redis.Redis(host='redis-cache', port=6379)
  
    def listen(self):
        for message in self.consumer:
            event = message.value
            if event["event_type"] == "TABLE_UPDATED":
                self.invalidate_cache(event["table"])
  
    def invalidate_cache(self, table: str):
        """Invalide le cache pour une table mise à jour."""
        pattern = f"query:*:{table}:*"
        keys = self.redis.keys(pattern)
        if keys:
            self.redis.delete(*keys)
```

---

## Patterns d'Accès Avancés Iceberg

### Time Travel pour Analyse Historique

Le Time Travel Iceberg permet d'interroger les données à n'importe quel point dans le temps, offrant des capacités analytiques puissantes pour l'audit, la conformité et l'analyse de tendances.

**Cas d'usage du Time Travel** :

| Cas d'usage          | Requête                         | Bénéfice  |
| -------------------- | -------------------------------- | ----------- |
| Audit réglementaire | État à date de clôture        | Conformité |
| Debug de pipeline    | État avant transformation       | Diagnostic  |
| Analyse YoY          | Comparaison même date an passé | Tendances   |
| Reproductibilité ML | Snapshot d'entraînement         | Science     |
| Rollback analytique  | Retour état précédent         | Correction  |

**Requêtes Time Travel** :

```sql
-- Requête à un timestamp spécifique
SELECT * FROM lakehouse.gold.transactions
FOR TIMESTAMP AS OF TIMESTAMP '2024-01-01 00:00:00';

-- Requête à un snapshot spécifique
SELECT * FROM lakehouse.gold.transactions
FOR VERSION AS OF 1234567890123456789;

-- Comparaison année sur année
WITH 
ventes_2024 AS (
    SELECT 
        MONTH(date_transaction) as mois,
        SUM(montant) as ca
    FROM lakehouse.gold.transactions
    FOR TIMESTAMP AS OF TIMESTAMP '2024-12-31 23:59:59'
    WHERE YEAR(date_transaction) = 2024
    GROUP BY MONTH(date_transaction)
),
ventes_2023 AS (
    SELECT 
        MONTH(date_transaction) as mois,
        SUM(montant) as ca
    FROM lakehouse.gold.transactions
    FOR TIMESTAMP AS OF TIMESTAMP '2023-12-31 23:59:59'
    WHERE YEAR(date_transaction) = 2023
    GROUP BY MONTH(date_transaction)
)
SELECT 
    v24.mois,
    v24.ca as ca_2024,
    v23.ca as ca_2023,
    (v24.ca - v23.ca) / v23.ca * 100 as croissance_pct
FROM ventes_2024 v24
JOIN ventes_2023 v23 ON v24.mois = v23.mois
ORDER BY v24.mois;
```

**API Time Travel pour applications** :

```python
class TimeTravel:
    """Service d'accès aux données historiques."""
  
    def __init__(self, catalog):
        self.catalog = catalog
  
    def query_at_timestamp(self, table: str, timestamp: str, sql: str) -> pd.DataFrame:
        """Exécute une requête à un timestamp donné."""
        table_ref = f"{table} FOR TIMESTAMP AS OF TIMESTAMP '{timestamp}'"
        full_sql = sql.replace(f"FROM {table}", f"FROM {table_ref}")
        return self._execute(full_sql)
  
    def query_at_snapshot(self, table: str, snapshot_id: int, sql: str) -> pd.DataFrame:
        """Exécute une requête à un snapshot donné."""
        table_ref = f"{table} FOR VERSION AS OF {snapshot_id}"
        full_sql = sql.replace(f"FROM {table}", f"FROM {table_ref}")
        return self._execute(full_sql)
  
    def get_snapshots(self, table: str) -> List[dict]:
        """Liste les snapshots disponibles pour une table."""
        iceberg_table = self.catalog.load_table(table)
        return [
            {
                "snapshot_id": s.snapshot_id,
                "timestamp": s.timestamp_ms,
                "operation": s.operation,
                "summary": s.summary
            }
            for s in iceberg_table.history()
        ]
  
    def diff_between_snapshots(self, table: str, 
                               snapshot_from: int, 
                               snapshot_to: int) -> pd.DataFrame:
        """Calcule les différences entre deux snapshots."""
        # Utilise les incremental reads Iceberg
        pass
```

### Branches pour Environnements Isolés

Avec des catalogues comme Nessie, les branches permettent de créer des environnements de données isolés sans duplication physique.

**Cas d'usage des branches** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      WORKFLOW DE BRANCHES                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  main ─────●─────●─────●─────●─────●─────●─────●───────────             │
│             \                   /         \                              │
│  dev ────────●─────●─────●─────           ●─────●────────               │
│               \                            │                             │
│  feature-a ────●─────●─────────────────────┘                            │
│                                                                          │
│  Cas d'usage:                                                           │
│  • Développement de nouvelles transformations                           │
│  • Test de migrations de schéma                                         │
│  • Analyse exploratoire sans impact production                          │
│  • Validation de données avant publication                              │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

**Utilisation des branches pour analyse** :

```python
# Configuration pour branche de développement
spark_dev = SparkSession.builder \
    .config("spark.sql.catalog.lakehouse.ref", "dev-experiment") \
    .getOrCreate()

# Les requêtes utilisent l'état de la branche dev
df_dev = spark_dev.sql("""
    SELECT * FROM lakehouse.gold.transactions
    WHERE date_transaction = CURRENT_DATE
""")

# Modifications expérimentales (n'affectent pas main)
spark_dev.sql("""
    ALTER TABLE lakehouse.gold.transactions 
    ADD COLUMN nouvelle_metrique DECIMAL(10,2)
""")

# Après validation, merge vers main
nessie_client.merge(
    from_ref="dev-experiment",
    to_ref="main",
    message="Ajout métrique validée après tests"
)
```

### Lectures Incrémentielles

Les lectures incrémentielles permettent de ne lire que les données modifiées depuis une position donnée, optimisant les pipelines de synchronisation et les flux temps réel.

**Pattern de lecture incrémentielle** :

```python
def incremental_read(table: str, 
                     start_snapshot: int, 
                     end_snapshot: int = None) -> DataFrame:
    """Lit uniquement les modifications entre deux snapshots."""
  
    reader = spark.read.format("iceberg")
  
    if end_snapshot:
        # Modifications entre deux snapshots
        reader = reader \
            .option("start-snapshot-id", start_snapshot) \
            .option("end-snapshot-id", end_snapshot)
    else:
        # Modifications depuis un snapshot jusqu'à maintenant
        reader = reader \
            .option("start-snapshot-id", start_snapshot)
  
    return reader.load(f"lakehouse.{table}")

# Exemple: synchronisation incrémentielle vers système externe
class IncrementalSync:
    def __init__(self, table: str):
        self.table = table
        self.state_store = redis.Redis()
  
    def sync(self):
        # Récupération du dernier snapshot synchronisé
        last_snapshot = self.state_store.get(f"sync:{self.table}:last_snapshot")
        last_snapshot = int(last_snapshot) if last_snapshot else None
      
        # Récupération du snapshot courant
        iceberg_table = catalog.load_table(self.table)
        current_snapshot = iceberg_table.current_snapshot().snapshot_id
      
        if last_snapshot == current_snapshot:
            print("Aucune modification")
            return
      
        # Lecture incrémentielle
        if last_snapshot:
            df_changes = incremental_read(self.table, last_snapshot, current_snapshot)
        else:
            # Premier sync: lecture complète
            df_changes = spark.read.table(f"lakehouse.{self.table}")
      
        # Synchronisation vers système cible
        self.sync_to_target(df_changes)
      
        # Mise à jour de l'état
        self.state_store.set(f"sync:{self.table}:last_snapshot", current_snapshot)
```

**Streaming incrémental avec Spark** :

```python
# Lecture streaming des modifications Iceberg
df_stream = spark.readStream \
    .format("iceberg") \
    .option("stream-from-timestamp", "2024-01-01T00:00:00Z") \
    .load("lakehouse.gold.transactions")

# Traitement des modifications en temps réel
query = df_stream \
    .writeStream \
    .foreachBatch(process_incremental_batch) \
    .option("checkpointLocation", "s3://checkpoints/incremental-sync") \
    .start()
```

---

## Gouvernance de la Consommation

### Quotas et Limites de Ressources

La gestion des ressources prévient les abus et garantit une expérience équitable pour tous les consommateurs.

**Configuration des quotas Trino** :

```properties
# etc/resource-groups.json
{
  "rootGroups": [
    {
      "name": "global",
      "softMemoryLimit": "80%",
      "hardConcurrencyLimit": 100,
      "maxQueued": 1000,
      "subGroups": [
        {
          "name": "adhoc",
          "softMemoryLimit": "30%",
          "hardConcurrencyLimit": 30,
          "maxQueued": 200,
          "schedulingPolicy": "fair",
          "schedulingWeight": 1
        },
        {
          "name": "pipeline",
          "softMemoryLimit": "40%",
          "hardConcurrencyLimit": 20,
          "maxQueued": 100,
          "schedulingPolicy": "fair",
          "schedulingWeight": 2
        },
        {
          "name": "bi_reports",
          "softMemoryLimit": "20%",
          "hardConcurrencyLimit": 40,
          "maxQueued": 500,
          "schedulingPolicy": "fair",
          "schedulingWeight": 1
        },
        {
          "name": "api_service",
          "softMemoryLimit": "10%",
          "hardConcurrencyLimit": 50,
          "maxQueued": 1000,
          "schedulingPolicy": "fair",
          "schedulingWeight": 3
        }
      ]
    }
  ],
  "selectors": [
    {
      "group": "global.pipeline",
      "user": "pipeline_.*"
    },
    {
      "group": "global.bi_reports",
      "source": "tableau|powerbi"
    },
    {
      "group": "global.api_service",
      "user": "api_service"
    },
    {
      "group": "global.adhoc"
    }
  ]
}
```

**Limites par utilisateur** :

```sql
-- Configuration des limites utilisateur
CREATE RESOURCE GROUP user_standard WITH (
    cpu_quota_period = '1m',
    cpu_quota = '500ms',
    memory_limit = '4GB',
    query_max_memory = '2GB',
    query_max_execution_time = '10m',
    query_max_cpu_time = '5m'
);

-- Attribution à un groupe d'utilisateurs
GRANT RESOURCE GROUP user_standard TO ROLE data_analyst;
```

### Contrôle d'Accès Granulaire

Le contrôle d'accès doit refléter les besoins métier tout en minimisant la surface d'exposition des données sensibles.

**Matrice d'accès par profil** :

| Profil            | Bronze            | Silver            | Gold                  | Features          | Cache   |
| ----------------- | ----------------- | ----------------- | --------------------- | ----------------- | ------- |
| Dirigeant         | ✗                | ✗                | Lecture (agrégé)    | ✗                | Lecture |
| Analyste affaires | ✗                | ✗                | Lecture               | ✗                | Lecture |
| Analyste données | ✗                | Lecture           | Lecture               | ✗                | Lecture |
| Data Scientist    | Lecture           | Lecture           | Lecture               | Lecture/Écriture | Lecture |
| Data Engineer     | Lecture/Écriture | Lecture/Écriture | Lecture/Écriture     | Lecture/Écriture | Admin   |
| Application       | ✗                | ✗                | Lecture (spécifique) | Lecture           | Lecture |

**Implémentation des politiques** :

```sql
-- Politique pour analystes d'affaires
CREATE ROLE business_analyst;
GRANT USAGE ON CATALOG lakehouse TO ROLE business_analyst;
GRANT USAGE ON SCHEMA lakehouse.gold TO ROLE business_analyst;
GRANT USAGE ON SCHEMA lakehouse.semantic TO ROLE business_analyst;
GRANT SELECT ON ALL TABLES IN SCHEMA lakehouse.gold TO ROLE business_analyst;
GRANT SELECT ON ALL TABLES IN SCHEMA lakehouse.semantic TO ROLE business_analyst;

-- Masquage automatique pour analystes
ALTER TABLE lakehouse.gold.clients 
ALTER COLUMN courriel SET MASK 
    CASE WHEN current_role() = 'data_admin' THEN courriel 
         ELSE regexp_replace(courriel, '(.{2}).*@', '$1***@') END;

-- Filtrage ligne pour accès régional
CREATE ROW ACCESS POLICY regional_access ON lakehouse.gold.transactions
FOR SELECT TO ROLE regional_analyst
USING region IN (
    SELECT region_autorisee 
    FROM lakehouse.security.autorisations_regionales 
    WHERE utilisateur = current_user()
);
```

### Monitoring et Observabilité

La surveillance de la consommation permet d'identifier les problèmes de performance, les abus et les opportunités d'optimisation.

**Métriques de consommation** :

```yaml
# prometheus-rules.yaml
groups:
  - name: consumption-metrics
    rules:
      # Requêtes par profil utilisateur
      - record: trino_queries_by_user_group
        expr: sum(rate(trino_query_completed_total[5m])) by (user_group)
    
      # Latence par type de requête
      - record: trino_query_latency_p99
        expr: histogram_quantile(0.99, rate(trino_query_execution_time_bucket[5m]))
    
      # Volume de données scannées
      - record: trino_data_scanned_bytes_total
        expr: sum(rate(trino_physical_input_bytes_total[5m]))
    
      # Alertes
      - alert: HighQueryLatency
        expr: trino_query_latency_p99 > 30
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Latence de requête élevée"
        
      - alert: ExcessiveDataScan
        expr: trino_data_scanned_bytes_total > 1e12
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Scan de données excessif détecté"
```

**Dashboard de consommation** :

```sql
-- Vue pour monitoring de la consommation
CREATE VIEW lakehouse.monitoring.consumption_stats AS
SELECT 
    DATE_TRUNC('hour', query_start_time) as heure,
    user_group,
    COUNT(*) as nb_requetes,
    SUM(CASE WHEN state = 'FINISHED' THEN 1 ELSE 0 END) as succes,
    SUM(CASE WHEN state = 'FAILED' THEN 1 ELSE 0 END) as echecs,
    AVG(execution_time_ms) as latence_moyenne_ms,
    PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY execution_time_ms) as latence_p99_ms,
    SUM(physical_input_bytes) / 1e9 as donnees_scannees_gb,
    SUM(output_rows) as lignes_retournees
FROM system.runtime.queries
WHERE query_start_time >= CURRENT_TIMESTAMP - INTERVAL '7' DAY
GROUP BY DATE_TRUNC('hour', query_start_time), user_group;
```

---

## Études de Cas Canadiennes

### Secteur Manufacturier : Équipementier Automobile

> **Étude de cas : Équipementier automobile ontarien**
>
> *Secteur* : Fabrication automobile
>
> *Défi* : Démocratiser l'accès aux données de production pour 500+ ingénieurs et gestionnaires répartis sur 12 usines au Canada et aux États-Unis. Les données de production, qualité et maintenance étaient silotées dans des systèmes MES (Manufacturing Execution System) propriétaires.
>
> *Solution* : Déploiement d'un Lakehouse Iceberg consolidant les données de tous les sites avec une couche sémantique Dremio exposant des vues métier standardisées. Tableaux de bord Power BI pour les gestionnaires, accès notebook pour les ingénieurs qualité, API pour les systèmes d'alerte.
>
> *Architecture de consommation* :
>
> ```
> ┌─────────────────────────────────────────────────────────────────┐
> │                    CONSOMMATEURS                                │
> │  Gestionnaires (150) │ Ingénieurs (300) │ Systèmes (50)        │
> │  Power BI            │ Jupyter + SQL    │ API REST             │
> └─────────────────────────────────────────────────────────────────┘
>                                 │
>                                 ▼
> ┌─────────────────────────────────────────────────────────────────┐
> │                    COUCHE SÉMANTIQUE (Dremio)                  │
> │  • Vues standardisées par usine                                 │
> │  • KPI qualité temps réel                                       │
> │  • Historique maintenance                                       │
> │  • Reflections pour tableaux de bord                           │
> └─────────────────────────────────────────────────────────────────┘
>                                 │
>                                 ▼
> ┌─────────────────────────────────────────────────────────────────┐
> │                    LAKEHOUSE ICEBERG (S3)                       │
> │  12 usines × 200 tables = 2400 tables                          │
> │  5 ans d'historique = 50 To                                    │
> └─────────────────────────────────────────────────────────────────┘
> ```
>
> *Résultats* :
>
> * Temps d'accès aux données réduit de 2 jours à 5 minutes
> * 85% des rapports désormais en self-service
> * Détection d'anomalies qualité accélérée de 70%
> * Économies de 2M$/an en coûts d'analyse externalisée

### Secteur Assurance : Mutuelle Pancanadienne

> **Étude de cas : Compagnie d'assurance mutuelle**
>
> *Secteur* : Assurance IARD (Incendie, Accidents, Risques Divers)
>
> *Défi* : Permettre aux actuaires et analystes de risques d'accéder aux données de sinistres historiques (20 ans) pour modélisation, tout en respectant les exigences de confidentialité des assurés. Les scientifiques de données avaient besoin d'accès aux données brutes pour l'entraînement de modèles de détection de fraude.
>
> *Solution* : Lakehouse Iceberg avec contrôle d'accès granulaire via Apache Ranger. Données personnelles masquées automatiquement selon le rôle. Feature Store Iceberg pour les modèles ML avec versionnement des features.
>
> *Contrôle d'accès* :
>
> ```sql
> -- Actuaires: accès agrégé uniquement
> SELECT 
>     region, 
>     type_sinistre,
>     annee,
>     SUM(montant_indemnise) as total_indemnites,
>     COUNT(*) as nb_sinistres,
>     AVG(montant_indemnise) as indemnite_moyenne
> FROM lakehouse.gold.sinistres_anonymises
> GROUP BY region, type_sinistre, annee;
>
> -- Data Scientists: accès détail avec masquage PII
> SELECT 
>     sinistre_id,
>     hash(assure_id) as assure_hash,  -- Pseudonymisé
>     '***' as nom_assure,              -- Masqué
>     type_sinistre,
>     montant_reclame,
>     montant_indemnise,
>     features_fraude
> FROM lakehouse.silver.sinistres_ml;
> ```
>
> *Résultats* :
>
> * 200 actuaires avec accès self-service sécurisé
> * Modèle de fraude réduisant les pertes de 15%
> * Conformité aux exigences du BSIF et de l'AMF
> * Temps de développement des modèles réduit de 60%

### Secteur Public : Agence Provinciale de Transport

> **Étude de cas : Société de transport provincial**
>
> *Secteur* : Transport public
>
> *Défi* : Analyser les données de 50 millions de trajets annuels pour optimiser les horaires et capacités. Exposition de données ouvertes au public tout en protégeant les données de validation des titres de transport individuels.
>
> *Solution* : Architecture de consommation à trois niveaux : données ouvertes (agrégées) via API publique, données opérationnelles via Power BI pour gestionnaires, données détaillées via notebooks pour équipe analytique interne.
>
> *Architecture* :
>
> ```
> ┌─────────────────────────────────────────────────────────────────┐
> │  PUBLIC (Données ouvertes)                                      │
> │  • API REST publique                                            │
> │  • Fréquentation par station/heure (agrégé)                    │
> │  • Pas de données individuelles                                 │
> └─────────────────────────────────────────────────────────────────┘
>                                 │
> ┌─────────────────────────────────────────────────────────────────┐
> │  OPÉRATIONNEL (Gestionnaires)                                  │
> │  • Power BI avec Row-Level Security                            │
> │  • Données par ligne/station gérée                             │
> │  • Rafraîchissement horaire                                     │
> └─────────────────────────────────────────────────────────────────┘
>                                 │
> ┌─────────────────────────────────────────────────────────────────┐
> │  ANALYTIQUE (Équipe interne)                                   │
> │  • Jupyter + Spark                                              │
> │  • Données détaillées pseudonymisées                           │
> │  • Modélisation prédictive                                      │
> └─────────────────────────────────────────────────────────────────┘
> ```
>
> *Résultats* :
>
> * 10 000+ téléchargements/mois des données ouvertes
> * Optimisation des horaires réduisant les temps d'attente de 12%
> * Conformité aux lois d'accès à l'information
> * Économies de 5M$/an en optimisation des ressources

### Secteur Détail : Épicier Québécois

> **Étude de cas : Chaîne d'épiceries québécoise**
>
> *Secteur* : Commerce alimentaire de détail
>
> *Défi* : Unifier les données de 400 magasins, du commerce en ligne et du programme de fidélité pour personnalisation des offres. Les gestionnaires de catégories avaient besoin d'analyses en temps réel pendant les négociations fournisseurs.
>
> *Solution* : Lakehouse Iceberg avec couche de consommation multi-niveaux. Dremio avec Reflections pour accélération des tableaux de bord. Feature Store pour le moteur de recommandation personnalisée.
>
> *Métriques de consommation* :
>
> ```
> Consommateurs:
> - 50 gestionnaires catégories: Power BI (temps réel)
> - 20 analystes merchandising: SQL + Tableau
> - 5 data scientists: Notebooks + Feature Store
> - 1 système recommandation: API (1000 req/sec)
>
> Performance:
> - Latence tableaux de bord: < 2 secondes (avec Reflections)
> - Latence API recommandation: < 100ms (avec cache Redis)
> - Fraîcheur données: 15 minutes (streaming lakehouse)
> ```
>
> *Résultats* :
>
> * Personnalisation augmentant le panier moyen de 8%
> * Réduction des ruptures de stock de 25%
> * Temps de génération des rapports catégorie de 4h à 30 secondes
> * ROI de 300% la première année

---

## Bonnes Pratiques et Recommandations

### Architecture de Référence par Profil

**Pour analystes d'affaires** :

```
Recommandations:
- Couche sémantique avec tables dénormalisées
- Nommage métier explicite
- Documentation intégrée
- Métriques pré-calculées
- Accès Power BI/Tableau optimisé
- Reflections pour performance

Anti-patterns à éviter:
- Exposition directe des tables techniques
- Jointures complexes requises
- Nommage technique (id_fk, amt_ttc_ht)
```

**Pour data scientists** :

```
Recommandations:
- Accès direct aux tables Silver pour exploration
- Feature Store centralisé
- Time Travel pour reproductibilité
- Branches pour expérimentation
- Intégration MLflow/Weights&Biases

Anti-patterns à éviter:
- Copies de données manuelles
- Features calculées localement
- Manque de versionnement
```

**Pour applications** :

```
Recommandations:
- Cache multi-niveaux (Redis + matérialisé)
- API avec rate limiting
- Tables pré-agrégées pour requêtes fréquentes
- Circuit breaker pour résilience
- Monitoring latence et disponibilité

Anti-patterns à éviter:
- Requêtes ad-hoc depuis applications
- Absence de cache
- Dépendance directe au Lakehouse pour SLA critique
```

### Checklist de Mise en Production

**Préparation** :

* [ ] Profils de consommateurs identifiés et documentés
* [ ] Exigences de latence définies par profil
* [ ] Volumes et fréquences d'accès estimés
* [ ] Politiques de sécurité définies

**Infrastructure** :

* [ ] Couche sémantique configurée
* [ ] Cache multi-niveaux déployé
* [ ] API documentée et versionnée
* [ ] Quotas et limites configurés

**Sécurité** :

* [ ] Contrôle d'accès par rôle implémenté
* [ ] Masquage des données sensibles actif
* [ ] Row-level security si nécessaire
* [ ] Audit de consommation activé

**Monitoring** :

* [ ] Métriques de latence collectées
* [ ] Alertes configurées
* [ ] Dashboard de consommation disponible
* [ ] Processus d'optimisation établi

### Matrice de Décision Outils

| Besoin             | Solution Recommandée       | Alternative      |
| ------------------ | --------------------------- | ---------------- |
| BI entreprise      | Power BI + Trino            | Tableau + Dremio |
| Self-service       | Dremio (couche sémantique) | Superset         |
| Data Science       | PyIceberg + Notebooks       | PySpark          |
| Feature Store      | Feast + Iceberg             | Tecton           |
| API faible latence | FastAPI + Redis + Trino     | GraphQL + Cache  |
| Temps réel        | Kafka + Flink + Iceberg     | Spark Streaming  |

---

## Conclusion

La couche de consommation constitue l'interface critique entre votre Data Lakehouse Apache Iceberg et les utilisateurs qui en extraient la valeur. Son succès repose sur une compréhension fine des besoins distincts de chaque profil de consommateur — des dirigeants consultant des KPI synthétiques aux applications interrogeant le Lakehouse en temps réel — et sur une architecture adaptée à cette diversité.

Apache Iceberg apporte des capacités différenciantes qui transforment l'expérience de consommation. Le Time Travel permet aux analystes d'explorer l'historique sans intervention technique. L'évolution de schéma garantit la continuité des requêtes malgré les modifications structurelles. Les lectures incrémentielles optimisent les synchronisations. Ces fonctionnalités, souvent invisibles pour les utilisateurs finaux, élèvent considérablement la qualité et la fiabilité de l'accès aux données.

La démocratisation des données — objectif central de nombreuses organisations — ne se résume pas à « donner accès ». Elle requiert une couche sémantique traduisant la complexité technique en concepts métier, des mécanismes de cache garantissant des performances acceptables, et une gouvernance granulaire protégeant les données sensibles sans entraver les usages légitimes. Les études de cas canadiennes présentées démontrent que cet équilibre est atteignable dans des contextes variés.

Les tendances émergentes — feature stores centralisés, APIs de données standardisées, intégration MLOps native — continueront d'enrichir les patterns de consommation. Les organisations investissant dans une couche de consommation bien architecturée se positionnent favorablement pour adopter ces évolutions sans refonte majeure.

Le chapitre suivant aborde la maintenance en production du Lakehouse, où nous examinerons les opérations quotidiennes — compaction, expiration des snapshots, optimisation — garantissant performance et fiabilité dans la durée.

---

## Résumé

**Profils de consommateurs** :

* **Dirigeants** : KPI agrégés, latence < 1s, interfaces visuelles
* **Analystes affaires** : Tableaux de bord, drill-down, self-service
* **Analystes données** : SQL complexe, jointures, exploration
* **Data Scientists** : Notebooks, DataFrames, reproductibilité
* **Applications** : APIs faible latence, haute disponibilité
* **Pipelines** : Accès programmatique, batch/streaming

**Intégration BI** :

* Power BI : Import, DirectQuery ou Direct Lake (Fabric)
* Tableau : JDBC avec extraits pour performance
* Self-service : Tables dénormalisées, nommage métier, documentation

**Science des données** :

* PyIceberg et PySpark pour accès DataFrame
* Time Travel pour reproductibilité des expériences
* Feature Stores avec Iceberg comme backend
* Intégration MLflow pour traçabilité

**Consommation applicative** :

* APIs REST/GraphQL avec FastAPI
* Cache multi-niveaux (Redis + matérialisé)
* Pattern CQRS pour séparation lecture/écriture
* Rate limiting et circuit breakers

**Patterns Iceberg avancés** :

* Time Travel pour analyse historique et audit
* Branches (Nessie) pour environnements isolés
* Lectures incrémentielles pour synchronisation optimisée

**Gouvernance** :

* Quotas et resource groups par profil
* Contrôle d'accès granulaire (table, colonne, ligne)
* Masquage automatique des données sensibles
* Monitoring et alerting de la consommation

---

*Ce chapitre établit les fondations de l'accès aux données de votre Lakehouse. Le chapitre suivant, « Maintenance en Production », détaille les opérations quotidiennes garantissant performance et fiabilité sur le long terme.*
