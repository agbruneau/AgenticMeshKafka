# Chapitre IV.11 - OPÉRATIONNALISER APACHE ICEBERG

*De la conception à l'exploitation : transformer votre lakehouse en plateforme de production*

---

## Introduction

Le passage d'un environnement de développement à une plateforme de production représente l'une des transitions les plus critiques dans le cycle de vie d'un Data Lakehouse. Concevoir une architecture Apache Iceberg élégante constitue un accomplissement technique, mais l'opérationnaliser de manière fiable, observable et résiliente représente un défi d'une tout autre nature. Cette distinction entre « ça fonctionne » et « ça fonctionne en production » sépare les projets expérimentaux des plateformes de données d'entreprise.

Les organisations qui adoptent Apache Iceberg découvrent rapidement que la valeur du format de table ne se réalise pleinement qu'avec une discipline opérationnelle rigoureuse. Sans orchestration appropriée, les pipelines s'exécutent de manière chaotique, les dépendances se brisent silencieusement et les opérations de maintenance s'accumulent. Sans audit, les exigences réglementaires deviennent ingérables et le diagnostic des problèmes tourne au cauchemar. Sans stratégie de récupération, un incident mineur peut se transformer en catastrophe.

Dans le chapitre précédent, nous avons exploré les mécanismes de maintenance d'un lakehouse Iceberg en production, notamment la compaction, l'expiration des snapshots et l'exploration des tables de métadonnées. Ce chapitre prolonge cette réflexion en abordant les dimensions opérationnelles plus larges : l'orchestration des pipelines, l'audit et la traçabilité des opérations, ainsi que la récupération après sinistre.

L'opérationnalisation d'Apache Iceberg s'inscrit dans la discipline émergente du DataOps, qui applique les principes DevOps au domaine des données. Cette approche reconnaît que les pipelines de données partagent les mêmes exigences que les applications logicielles en termes de fiabilité, d'observabilité et de récupération. Toutefois, les données présentent des caractéristiques uniques — leur volume, leur état et leur historique — qui nécessitent des stratégies adaptées.

Le contexte réglementaire canadien ajoute une couche de complexité supplémentaire. La Loi 25 au Québec, entrée pleinement en vigueur en 2024, impose des obligations strictes en matière de traçabilité et de protection des données personnelles. Les organisations assujetties doivent pouvoir démontrer où sont stockées les données, qui y accède, et comment elles sont transformées. Un lakehouse bien opérationnalisé, avec ses capacités d'audit et de *time travel*, répond naturellement à ces exigences lorsqu'il est correctement configuré.

Ce chapitre s'adresse aux architectes data et aux ingénieurs plateforme responsables de l'exploitation quotidienne d'un lakehouse Iceberg. Nous examinerons les outils d'orchestration modernes, les pratiques d'audit conformes aux exigences réglementaires et les stratégies de récupération après sinistre qui exploitent les capacités natives d'Iceberg. L'objectif est de fournir un cadre pratique permettant de transformer une architecture de données en une plateforme opérationnelle robuste.

---

## IV.11.1 Orchestration du Lakehouse

L'orchestration constitue le système nerveux central de tout lakehouse en production. Elle coordonne l'exécution des pipelines de données, gère les dépendances entre tâches et assure la résilience face aux défaillances. Dans le contexte d'Apache Iceberg, l'orchestration doit également gérer les opérations de maintenance spécifiques au format de table, telles que la compaction et l'expiration des snapshots.

### Le rôle de l'orchestration dans un lakehouse moderne

Un lakehouse Iceberg en production comprend typiquement des dizaines, voire des centaines de pipelines interconnectés. Ces pipelines ingèrent des données depuis diverses sources, appliquent des transformations, exécutent des opérations de maintenance et alimentent les consommateurs en aval. Sans orchestration centralisée, la coordination de ces processus devient rapidement ingérable.

L'orchestration remplit plusieurs fonctions essentielles dans ce contexte. Premièrement, elle assure la planification temporelle des tâches selon des calendriers définis ou des événements déclencheurs. Deuxièmement, elle gère les dépendances entre pipelines, garantissant qu'une transformation ne s'exécute qu'après l'ingestion réussie des données sources. Troisièmement, elle orchestre les opérations de reprise automatique en cas d'échec, avec des stratégies de backoff exponentielles pour éviter la surcharge des systèmes. Quatrièmement, elle fournit une vue unifiée de l'état de tous les pipelines, facilitant le diagnostic des problèmes.

L'orchestration moderne va au-delà de la simple planification de tâches. Elle intègre des concepts comme l'orchestration orientée données (*data-aware orchestration*), où les décisions d'exécution sont basées sur l'état des données plutôt que sur des horaires fixes. Cette approche s'aligne naturellement avec les capacités d'Iceberg, où les snapshots et les métadonnées fournissent une visibilité complète sur l'état des tables.

### Panorama des orchestrateurs pour les lakehouses

Le paysage des outils d'orchestration a considérablement évolué ces dernières années. Selon l'analyse de l'écosystème des orchestrateurs en 2025 (PracData, 2025), Apache Airflow demeure le standard dominant avec plus de 320 millions de téléchargements annuels, suivi de Prefect et Dagster. Chaque outil présente des forces distinctes pour l'orchestration d'un lakehouse Iceberg.

**Apache Airflow** représente le choix éprouvé pour les organisations recherchant maturité et écosystème étendu. Son modèle basé sur les DAG (Directed Acyclic Graphs) définit les flux de travail comme du code Python, offrant flexibilité et versionnement. Airflow excelle dans les environnements où les pipelines sont relativement stables et où l'intégration avec des systèmes existants est primordiale. Ses milliers d'opérateurs prêts à l'emploi facilitent l'intégration avec Apache Spark, Trino et les services infonuagiques. Les versions gérées comme Amazon MWAA, Google Cloud Composer et Astronomer simplifient l'exploitation en production.

Toutefois, Airflow présente des limitations pour les lakehouses modernes. Son modèle centré sur les tâches plutôt que sur les données complique la gestion de la lignée et l'observabilité des actifs de données. Les performances peuvent se dégrader avec des déploiements à très grande échelle, nécessitant une optimisation attentive de l'ordonnanceur.

**Dagster** adopte une approche fondamentalement différente, centrée sur les actifs de données (*software-defined assets*). Plutôt que de définir des tâches, les ingénieurs définissent les données qu'ils souhaitent produire, et Dagster gère automatiquement les dépendances et l'exécution. Cette philosophie s'aligne remarquablement bien avec les lakehouses, où les tables Iceberg constituent les actifs principaux.

Dagster offre plusieurs avantages pour l'orchestration Iceberg. Son système de partitionnement natif gère élégamment les tables partitionnées temporellement, courantes dans les lakehouses. Les *sensors* permettent de déclencher des pipelines en réponse à l'arrivée de nouvelles données, plutôt que selon des horaires fixes. Les vérifications de qualité des données (*asset checks*) s'intègrent directement dans la définition des actifs, facilitant la validation avant publication.

L'article de Dagster (2025) démontre comment construire un lakehouse complet avec MinIO, Trino et Iceberg en utilisant une approche orientée actifs. Cette architecture permet une observabilité native où chaque table Iceberg est visible avec son historique de matérialisation, ses dépendances et ses métriques de santé.

**Prefect** se positionne comme une alternative pythonique avec un modèle d'exécution hybride. La version 3.0 (2025) embrasse l'orchestration sans serveur (*serverless*), où les ressources de calcul sont provisionnées à la demande. Cette approche peut réduire significativement les coûts pour les pipelines à exécution intermittente.

**Temporal** mérite également une mention pour les cas d'usage impliquant des processus de longue durée ou des agents d'intelligence artificielle. Selon Datum Labs (2025), Temporal excelle dans la gestion d'états complexes et la récupération après des défaillances prolongées, ce qui peut être pertinent pour les pipelines d'apprentissage automatique alimentés par un lakehouse.

### Comparaison des orchestrateurs pour les lakehouses

Le choix d'un orchestrateur dépend de multiples facteurs : maturité de l'équipe, complexité des pipelines, exigences de latence et intégration avec l'écosystème existant. Le tableau suivant synthétise les caractéristiques clés des principaux orchestrateurs pour les lakehouses Iceberg.

| Critère | Apache Airflow | Dagster | Prefect | Temporal |
|---------|----------------|---------|---------|----------|
| **Philosophie** | Orienté tâches (DAG) | Orienté actifs (Assets) | Flux de données | Workflows durables |
| **Maturité** | Très mature (2014) | Mature (2019) | Mature (2018) | Mature (2019) |
| **Communauté** | Très large | Large | Moyenne | Moyenne |
| **Courbe d'apprentissage** | Moyenne | Moyenne-haute | Faible | Haute |
| **Observabilité native** | Moyenne | Excellente | Bonne | Bonne |
| **Lignée des données** | Limitée | Native | Limitée | N/A |
| **Gestion des partitions** | Manuelle | Native | Manuelle | N/A |
| **Tests locaux** | Limités | Excellents | Bons | Bons |
| **Intégration infonuagique** | Excellente (MWAA, Composer) | Dagster Cloud | Prefect Cloud | Temporal Cloud |
| **Coût (auto-hébergé)** | Moyen | Moyen | Faible | Moyen |
| **Cas d'usage optimal** | Pipelines batch stables | Lakehouses, ML | Pipelines agiles | Workflows IA |

### Critères de sélection d'un orchestrateur

La sélection d'un orchestrateur pour un lakehouse Iceberg doit considérer plusieurs dimensions.

**Maturité de l'équipe data** : Les équipes familières avec Airflow peuvent bénéficier de sa large communauté et de ses ressources de formation abondantes. Les équipes plus modernes ou construisant un nouveau lakehouse peuvent préférer Dagster pour son alignement avec les architectures orientées actifs.

**Exigences de lignée** : Si la traçabilité des données est critique (conformité réglementaire, debug de pipelines complexes), Dagster offre une lignée native supérieure. Airflow nécessite des outils complémentaires comme Apache Atlas ou des solutions commerciales.

**Complexité des pipelines** : Les pipelines simples avec peu de dépendances fonctionnent bien avec n'importe quel orchestrateur. Les architectures complexes avec des centaines de tables interconnectées bénéficient de l'approche orientée actifs de Dagster.

**Budget et ressources** : Airflow en mode auto-hébergé peut devenir coûteux en ressources d'administration. Les versions gérées (MWAA, Composer) simplifient l'exploitation mais ajoutent des coûts directs. Dagster Cloud et Prefect Cloud offrent des modèles de tarification basés sur l'usage.

**Intégration avec l'IA/ML** : Pour les organisations intégrant fortement l'apprentissage automatique, Dagster offre des intégrations natives avec MLflow, Weights & Biases et les frameworks de feature store. Temporal convient aux workflows d'agents IA nécessitant une gestion d'état durable.

> **Étude de cas : Recommandation d'orchestrateur**
> *Contexte* : Une entreprise de commerce électronique canadienne avec 50 tables Iceberg, 20 data engineers et des exigences de conformité Loi 25.
> *Analyse* : L'équipe utilise actuellement des scripts cron et souhaite professionnaliser l'orchestration. La lignée des données est importante pour la conformité. Le budget est limité.
> *Recommandation* : Dagster en mode auto-hébergé sur Kubernetes. La lignée native répond aux exigences de conformité. L'approche orientée actifs s'aligne avec l'architecture Iceberg. Les coûts restent maîtrisés en évitant les solutions SaaS.
> *Alternative* : Airflow sur Amazon MWAA si l'équipe est déjà familière avec Airflow et préfère une exploitation simplifiée.

### Modèles d'orchestration pour Apache Iceberg

L'orchestration d'un lakehouse Iceberg requiert des modèles spécifiques qui tiennent compte des caractéristiques du format de table. Nous identifions trois modèles principaux : l'orchestration par horaire, l'orchestration événementielle et l'orchestration hybride.

**L'orchestration par horaire** demeure le modèle le plus répandu, où les pipelines s'exécutent selon des calendriers prédéfinis. Un pipeline d'ingestion quotidien pourrait s'exécuter à minuit, suivi des transformations à 2h00 et des tâches de maintenance à 4h00. Ce modèle offre prévisibilité et simplicité, mais peut entraîner des latences importantes et une utilisation inefficace des ressources.

```python
# Exemple Airflow : Pipeline Iceberg quotidien
from airflow import DAG
from airflow.operators.python import PythonOperator
from datetime import datetime, timedelta

default_args = {
    'owner': 'data-engineering',
    'depends_on_past': False,
    'retries': 3,
    'retry_delay': timedelta(minutes=5),
}

with DAG(
    'lakehouse_daily_pipeline',
    default_args=default_args,
    schedule_interval='0 0 * * *',
    start_date=datetime(2025, 1, 1),
    catchup=False
) as dag:
    
    ingest_task = PythonOperator(
        task_id='ingest_raw_data',
        python_callable=ingest_to_iceberg,
    )
    
    transform_task = PythonOperator(
        task_id='transform_silver_layer',
        python_callable=transform_silver,
    )
    
    compact_task = PythonOperator(
        task_id='compact_iceberg_tables',
        python_callable=run_compaction,
    )
    
    expire_snapshots_task = PythonOperator(
        task_id='expire_old_snapshots',
        python_callable=expire_snapshots,
    )
    
    ingest_task >> transform_task >> compact_task >> expire_snapshots_task
```

**L'orchestration événementielle** déclenche les pipelines en réponse à des événements, tels que l'arrivée de nouveaux fichiers dans le stockage objet ou la publication d'un message Kafka. Ce modèle réduit la latence et optimise l'utilisation des ressources, mais augmente la complexité opérationnelle.

L'intégration avec Apache Kafka, détaillée dans le Volume III de cette monographie, permet de créer des architectures de *Streaming Lakehouse* où les données s'écoulent continuellement vers les tables Iceberg. L'orchestrateur surveille alors les commits Iceberg ou les offsets Kafka pour déclencher les transformations en aval.

```python
# Exemple Dagster : Sensor surveillant MinIO pour nouveaux fichiers
from dagster import sensor, RunRequest, SensorEvaluationContext
from datetime import datetime, timedelta

@sensor(
    minimum_interval_seconds=60,
    job_name="incremental_ingestion_job"
)
def minio_new_files_sensor(context: SensorEvaluationContext):
    """Surveille MinIO pour détecter les nouveaux fichiers et déclencher l'ingestion."""
    minio_client = context.resources.minio
    
    # Vérifier les fichiers des 24 dernières heures
    for days_back in range(1, 8):
        date = datetime.now() - timedelta(days=days_back)
        date_str = date.strftime("%Y-%m-%d")
        prefix = f"raw/transactions/dt={date_str}/"
        
        new_files = minio_client.list_new_objects(prefix)
        
        if new_files:
            yield RunRequest(
                partition_key=date_str,
                tags={"trigger": "new_data_detected", "source": "minio_sensor"}
            )
```

**L'orchestration hybride** combine les deux approches, utilisant des horaires pour les traitements batch réguliers et des événements pour les besoins de faible latence. Ce modèle représente l'état de l'art pour les lakehouses d'entreprise qui doivent supporter à la fois des rapports quotidiens et des tableaux de bord quasi temps réel.

### Orchestrer les opérations de maintenance Iceberg

Les opérations de maintenance constituent un aspect critique de l'orchestration d'un lakehouse. La compaction, l'expiration des snapshots et le nettoyage des fichiers orphelins doivent être exécutés régulièrement pour maintenir des performances optimales.

L'orchestration de la compaction requiert une attention particulière. Exécuter la compaction trop fréquemment consomme des ressources inutilement ; l'exécuter trop rarement dégrade les performances de lecture. Une stratégie efficace consiste à surveiller les métriques des tables (nombre de fichiers, taille moyenne des fichiers) et à déclencher la compaction lorsque des seuils sont dépassés.

```python
# Fonction de compaction conditionnelle
def conditional_compaction(table_name: str, spark_session):
    """Exécute la compaction uniquement si nécessaire."""
    from pyspark.sql.functions import count, avg
    
    # Récupérer les métriques des fichiers
    files_df = spark_session.read.format("iceberg").load(f"{table_name}.files")
    
    metrics = files_df.agg(
        count("*").alias("file_count"),
        avg("file_size_in_bytes").alias("avg_file_size")
    ).collect()[0]
    
    file_count = metrics["file_count"]
    avg_file_size = metrics["avg_file_size"]
    
    # Seuils de compaction
    MAX_FILES = 1000
    MIN_AVG_SIZE = 64 * 1024 * 1024  # 64 MB
    
    if file_count > MAX_FILES or avg_file_size < MIN_AVG_SIZE:
        spark_session.sql(f"""
            CALL iceberg.system.rewrite_data_files(
                table => '{table_name}',
                options => map('target-file-size-bytes', '134217728')
            )
        """)
        return {"compacted": True, "file_count": file_count}
    
    return {"compacted": False, "file_count": file_count}
```

L'expiration des snapshots doit être coordonnée avec les politiques de rétention de l'organisation et les exigences de *time travel*. Une pratique courante consiste à conserver les snapshots des 7 derniers jours pour permettre les analyses ad hoc, tout en maintenant des snapshots mensuels pour les audits.

```sql
-- Expiration des snapshots avec rétention de 7 jours
CALL iceberg.system.expire_snapshots(
    table => 'lakehouse.gold.customer_360',
    older_than => TIMESTAMP '2025-01-09 00:00:00',
    retain_last => 10
);

-- Nettoyage des fichiers orphelins (exécuter mensuellement)
CALL iceberg.system.remove_orphan_files(
    table => 'lakehouse.gold.customer_360',
    older_than => TIMESTAMP '2025-01-01 00:00:00'
);
```

### Infrastructure as Code pour l'orchestration

L'approche *Infrastructure as Code* (IaC) s'applique naturellement à l'orchestration d'un lakehouse. Les définitions de pipelines, les configurations des tâches et les paramètres d'exécution doivent être versionnés dans un système de contrôle de version comme Git.

Cette approche offre plusieurs avantages. La traçabilité permet de comprendre l'évolution des pipelines et d'identifier les changements responsables de régressions. La reproductibilité garantit que le même code produit le même comportement dans tous les environnements. La collaboration facilite la revue par les pairs et l'amélioration continue des pipelines.

Les outils modernes comme Dagster et Prefect sont conçus pour cette approche, avec des définitions de pipelines en Python pur qui s'intègrent naturellement avec les pratiques de développement logiciel.

```python
# Structure de projet recommandée pour un lakehouse orchestré
# lakehouse-orchestration/
# ├── dags/                    # Définitions Airflow
# │   ├── ingestion/
# │   ├── transformation/
# │   └── maintenance/
# ├── assets/                  # Définitions Dagster
# │   ├── bronze/
# │   ├── silver/
# │   └── gold/
# ├── configs/
# │   ├── dev.yaml
# │   ├── staging.yaml
# │   └── prod.yaml
# ├── tests/
# │   ├── unit/
# │   └── integration/
# └── requirements.txt
```

Le déploiement des pipelines doit suivre un flux CI/CD (Intégration Continue / Déploiement Continu) avec des tests automatisés. Les modifications passent d'abord par un environnement de développement, puis de staging, avant d'être promues en production.

```yaml
# Exemple GitHub Actions pour déploiement Airflow
name: Deploy DAGs

on:
  push:
    branches: [main]
    paths: ['dags/**']

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Test DAGs
        run: |
          pip install apache-airflow pytest
          pytest tests/unit/
      
  deploy:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - name: Sync DAGs to S3
        run: |
          aws s3 sync dags/ s3://airflow-dags-bucket/dags/
```

### Gestion des environnements multiples

Un lakehouse d'entreprise comprend typiquement plusieurs environnements : développement, test, staging et production. L'orchestration doit gérer ces environnements de manière cohérente tout en permettant des différences de configuration.

Les pratiques recommandées incluent la paramétrisation des pipelines pour externaliser les configurations spécifiques à l'environnement (chemins de stockage, tailles de cluster, politiques de rétention). La configuration par environnement utilise des fichiers YAML ou des variables d'environnement pour injecter les valeurs appropriées. L'isolation des données garantit que les environnements de développement utilisent des sous-ensembles ou des données anonymisées.

```python
# Configuration multi-environnement avec Dagster
from dagster import ConfigurableResource
from pydantic import Field

class IcebergConfig(ConfigurableResource):
    """Configuration paramétrable par environnement."""
    catalog_uri: str = Field(description="URI du catalogue Iceberg")
    warehouse_path: str = Field(description="Chemin du warehouse")
    default_database: str = Field(description="Base de données par défaut")
    retention_days: int = Field(default=7, description="Rétention des snapshots")
    compaction_threshold_files: int = Field(default=500, description="Seuil de compaction")

# Configurations par environnement
configs = {
    "dev": IcebergConfig(
        catalog_uri="http://nessie-dev:19120/api/v1",
        warehouse_path="s3a://lakehouse-dev/warehouse",
        default_database="dev_lakehouse",
        retention_days=3,
        compaction_threshold_files=100
    ),
    "prod": IcebergConfig(
        catalog_uri="http://nessie-prod:19120/api/v1",
        warehouse_path="s3a://lakehouse-prod/warehouse",
        default_database="prod_lakehouse",
        retention_days=30,
        compaction_threshold_files=500
    )
}
```

### Bonnes pratiques d'orchestration

L'expérience opérationnelle révèle plusieurs bonnes pratiques pour l'orchestration d'un lakehouse Iceberg.

**Idempotence des pipelines** : Chaque tâche doit être idempotente, c'est-à-dire que son exécution répétée avec les mêmes entrées produit le même résultat. Cette propriété est essentielle pour les reprises automatiques. Iceberg facilite l'idempotence grâce aux opérations atomiques et au *time travel*, permettant de rejouer des transformations à partir d'un snapshot spécifique.

**Observabilité native** : L'orchestrateur doit émettre des métriques détaillées sur l'exécution des pipelines : durée, volume de données traitées, nombre de fichiers créés. Ces métriques alimentent les tableaux de bord opérationnels et les alertes.

**Gestion des échecs** : Les stratégies de reprise doivent être adaptées au type de défaillance. Une erreur de connexion réseau justifie une reprise immédiate avec backoff exponentiel ; une erreur de données nécessite une intervention humaine. Les alertes doivent distinguer ces cas pour éviter la fatigue des équipes.

**Tests des pipelines** : Les DAG et les définitions d'actifs doivent être testés comme du code applicatif. Dagster offre des facilités de test local particulièrement développées, permettant de valider les pipelines avant déploiement en production.

> **Étude de cas : Financière Sun Life**
> *Secteur* : Services financiers
> *Défi* : Orchestrer 150+ pipelines de données alimentant les analyses actuarielles et la conformité réglementaire, avec des fenêtres de traitement nocturnes de 4 heures maximum.
> *Solution* : Migration d'Airflow vers Dagster pour bénéficier de l'orchestration orientée actifs. Chaque table Iceberg (bronze, silver, gold) est définie comme un actif avec ses dépendances explicites. Les sensors surveillent les commits Iceberg pour déclencher les transformations en cascade.
> *Résultats* : Réduction de 40 % du temps de traitement global grâce à une meilleure parallélisation. Amélioration de la visibilité avec lignée automatique des données. Temps moyen de résolution des incidents réduit de 2 heures à 25 minutes.

---

## IV.11.2 Audit du Lakehouse

L'audit constitue une dimension incontournable de l'opérationnalisation d'un lakehouse, particulièrement dans les secteurs réglementés comme la finance, la santé et les télécommunications. Au-delà des exigences de conformité, l'audit fournit une traçabilité essentielle pour le diagnostic des problèmes et l'analyse forensique des données.

### Exigences d'audit dans un contexte réglementaire

Les réglementations modernes imposent des exigences strictes en matière de traçabilité des données. Le Règlement général sur la protection des données (RGPD) en Europe et la Loi 25 au Québec exigent que les organisations puissent démontrer comment les données personnelles sont collectées, transformées et utilisées. Les normes sectorielles comme SOX (secteur financier) ou HIPAA (santé) ajoutent des contraintes supplémentaires.

Ces exigences se traduisent en capacités d'audit spécifiques pour un lakehouse. La traçabilité des accès enregistre qui a accédé à quelles données, quand et depuis quelle application. La traçabilité des modifications documente chaque changement apporté aux données, incluant l'auteur, le moment et la nature de la modification. La lignée des données trace le parcours des données depuis leur source jusqu'à leur consommation, permettant de comprendre comment un rapport est produit à partir des données brutes.

Apache Iceberg offre des fondations solides pour répondre à ces exigences grâce à son architecture basée sur les snapshots. Chaque modification d'une table Iceberg crée un nouveau snapshot, préservant l'historique complet des changements. Cette immutabilité native constitue un avantage significatif par rapport aux systèmes traditionnels où les données sont modifiées en place.

### Architecture d'audit pour un lakehouse Iceberg

Une architecture d'audit complète pour un lakehouse Iceberg comprend plusieurs composants : les journaux d'accès, les journaux de modification, la lignée des données et les rapports de conformité.

**Journaux d'accès** : Les moteurs de requête comme Dremio et Trino génèrent des journaux détaillant chaque requête exécutée. Ces journaux capturent l'identité de l'utilisateur, la requête SQL, les tables accédées, les données retournées (ou un résumé) et les performances. Ces journaux doivent être centralisés dans un système d'observabilité comme Elasticsearch/Kibana (ELK) ou une solution spécialisée.

```sql
-- Exemple de structure pour une table de journalisation des accès
CREATE TABLE audit.query_logs (
    query_id STRING,
    user_id STRING,
    user_email STRING,
    client_ip STRING,
    query_text STRING,
    tables_accessed ARRAY<STRING>,
    columns_accessed ARRAY<STRING>,
    rows_returned BIGINT,
    execution_time_ms BIGINT,
    query_status STRING,
    error_message STRING,
    timestamp TIMESTAMP
) USING iceberg
PARTITIONED BY (days(timestamp));
```

**Journaux de modification** : Iceberg maintient nativement l'historique des modifications à travers les snapshots. La table de métadonnées `snapshots` fournit un journal de toutes les opérations effectuées sur une table.

```sql
-- Consulter l'historique des modifications d'une table
SELECT 
    snapshot_id,
    committed_at,
    operation,
    summary['added-data-files'] as files_added,
    summary['deleted-data-files'] as files_deleted,
    summary['added-records'] as records_added,
    summary['deleted-records'] as records_deleted
FROM lakehouse.gold.customer_360.snapshots
ORDER BY committed_at DESC
LIMIT 100;
```

Pour enrichir ces journaux avec des informations contextuelles (utilisateur, application source, raison de la modification), une pratique efficace consiste à utiliser les propriétés de snapshot. Lors de chaque écriture, l'application peut définir des propriétés personnalisées qui seront préservées dans les métadonnées.

```python
# Écriture avec métadonnées d'audit
df.writeTo("lakehouse.gold.customer_360") \
    .option("snapshot-property.audit.user", current_user) \
    .option("snapshot-property.audit.application", "etl-pipeline-v2") \
    .option("snapshot-property.audit.reason", "Daily refresh from CRM") \
    .option("snapshot-property.audit.ticket", "JIRA-12345") \
    .append()
```

**Lignée des données** : La lignée documente les relations entre les tables, permettant de tracer le parcours des données. Cette information est cruciale pour comprendre l'impact d'un changement en amont sur les rapports en aval, et pour répondre aux demandes de conformité concernant la provenance des données.

Les outils de catalogues comme Apache Polaris, Atlan ou Alation peuvent extraire automatiquement la lignée en analysant les requêtes SQL et les définitions de pipelines. Cette lignée peut être enrichie par les orchestrateurs orientés actifs comme Dagster, qui maintiennent une lignée native basée sur les définitions de dépendances.

```python
# Exemple Dagster : Définition d'actifs avec lignée explicite
from dagster import asset

@asset(
    description="Table agrégée des ventes par région",
    metadata={
        "owner": "equipe-analytics",
        "data_quality_score": 0.95,
        "pii_classification": "none"
    }
)
def ventes_par_region(transactions_nettoyees, reference_regions):
    """Agrège les transactions par région géographique."""
    # La lignée est automatiquement déduite des dépendances
    return transactions_nettoyees.join(reference_regions).groupBy("region").agg(...)
```

### Intégration de la qualité des données dans l'audit

La qualité des données fait partie intégrante de l'audit d'un lakehouse. Les vérifications de qualité documentent l'état des données à chaque étape du pipeline, fournissant une piste d'audit de la fiabilité des données.

Les cadres de qualité des données comme Great Expectations, Soda ou dbt tests permettent de définir des attentes sur les données et de les valider automatiquement. Ces validations génèrent des rapports qui alimentent la piste d'audit.

```python
# Exemple Great Expectations pour validation avec audit
import great_expectations as gx

context = gx.get_context()

# Définir les attentes pour la table customer_360
suite = context.add_expectation_suite("customer_360_quality")

# Attentes de complétude
suite.add_expectation(
    gx.expectations.ExpectColumnValuesToNotBeNull(
        column="customer_id",
        meta={"audit_category": "completeness", "severity": "critical"}
    )
)

# Attentes de validité
suite.add_expectation(
    gx.expectations.ExpectColumnValuesToBeBetween(
        column="age",
        min_value=0,
        max_value=150,
        meta={"audit_category": "validity", "severity": "warning"}
    )
)

# Attentes de cohérence
suite.add_expectation(
    gx.expectations.ExpectColumnPairValuesAToBeGreaterThanB(
        column_A="total_purchases",
        column_B="total_returns",
        or_equal=True,
        meta={"audit_category": "consistency", "severity": "warning"}
    )
)

# Exécuter la validation et stocker les résultats pour audit
checkpoint = context.add_checkpoint(
    name="customer_360_daily_validation",
    validations=[{
        "batch_request": {"datasource_name": "iceberg_lakehouse", "data_asset_name": "customer_360"},
        "expectation_suite_name": "customer_360_quality"
    }]
)

results = checkpoint.run()

# Les résultats sont automatiquement stockés pour audit
print(f"Validation réussie : {results.success}")
print(f"Statistiques : {results.statistics}")
```

Les résultats de validation doivent être persistés dans une table d'audit dédiée, permettant l'analyse historique de la qualité des données.

```sql
-- Table d'audit de qualité des données
CREATE TABLE audit.data_quality_results (
    validation_id STRING,
    table_name STRING,
    validation_time TIMESTAMP,
    expectation_type STRING,
    column_name STRING,
    success BOOLEAN,
    observed_value STRING,
    expected_value STRING,
    severity STRING,
    audit_category STRING,
    snapshot_id BIGINT
) USING iceberg
PARTITIONED BY (days(validation_time));
```

### Architecture de collecte des journaux d'audit

Une architecture robuste de collecte des journaux d'audit doit gérer le volume, la fiabilité et la rétention des données d'audit.

**Collecte des journaux** : Les journaux proviennent de multiples sources : moteurs de requête (Dremio, Trino, Spark), orchestrateurs (Airflow, Dagster), applications métier et services infonuagiques. Un agent de collecte comme Fluentd ou Vector agrège ces journaux et les achemine vers un système centralisé.

```yaml
# Configuration Vector pour collecte des journaux d'audit
sources:
  trino_logs:
    type: file
    include: ["/var/log/trino/query*.log"]
    
  spark_logs:
    type: file
    include: ["/var/log/spark/audit*.log"]
    
  airflow_logs:
    type: file
    include: ["/var/log/airflow/scheduler*.log"]

transforms:
  parse_trino:
    type: remap
    inputs: ["trino_logs"]
    source: |
      . = parse_json!(.message)
      .source = "trino"
      .audit_timestamp = now()

sinks:
  iceberg_audit:
    type: http
    inputs: ["parse_trino", "parse_spark", "parse_airflow"]
    uri: "http://spark-rest:8080/audit/ingest"
    encoding:
      codec: json
```

**Rétention et archivage** : Les journaux d'audit doivent être conservés selon les exigences réglementaires, souvent 7 ans pour les données financières. Une stratégie de tiering déplace les journaux anciens vers un stockage économique (S3 Glacier, Azure Archive) tout en maintenant leur accessibilité.

```sql
-- Politique de rétention pour les journaux d'audit
-- Tables actives (< 90 jours) : Stockage standard
-- Tables archivées (90 jours - 7 ans) : Stockage archive

-- Vue unifiée des journaux d'audit
CREATE VIEW audit.unified_query_logs AS
SELECT * FROM audit.query_logs_active
UNION ALL
SELECT * FROM audit.query_logs_archive
WHERE timestamp > CURRENT_DATE - INTERVAL 7 YEAR;
```

### Exploiter les tables de métadonnées pour l'audit

Iceberg expose plusieurs tables de métadonnées qui constituent des sources d'information précieuses pour l'audit. Ces tables sont accessibles via des requêtes SQL standard, facilitant leur intégration dans les processus d'audit existants.

**Table `history`** : Fournit l'historique des snapshots avec les métadonnées associées.

```sql
-- Historique complet des modifications
SELECT 
    made_current_at,
    snapshot_id,
    parent_id,
    is_current_ancestor
FROM lakehouse.gold.customer_360.history
ORDER BY made_current_at DESC;
```

**Table `snapshots`** : Détaille chaque snapshot avec les statistiques d'opération.

**Table `manifests`** : Liste les fichiers manifest pour chaque snapshot, permettant une analyse fine des modifications au niveau des fichiers.

**Table `files`** : Énumère tous les fichiers de données avec leurs statistiques (taille, nombre d'enregistrements, valeurs min/max par colonne).

```sql
-- Analyse des fichiers pour détecter des anomalies
SELECT 
    partition,
    COUNT(*) as file_count,
    SUM(record_count) as total_records,
    AVG(file_size_in_bytes) / 1024 / 1024 as avg_file_size_mb,
    MIN(file_size_in_bytes) / 1024 as min_file_size_kb,
    MAX(file_size_in_bytes) / 1024 / 1024 as max_file_size_mb
FROM lakehouse.gold.customer_360.files
GROUP BY partition
HAVING MIN(file_size_in_bytes) < 1024 * 1024  -- Fichiers < 1 MB
ORDER BY file_count DESC;
```

### Observabilité et alertes

L'observabilité transforme les données d'audit brutes en informations actionnables. Elle permet de détecter proactivement les anomalies, d'alerter les équipes et de faciliter le diagnostic des problèmes.

**Métriques clés à surveiller** :

| Métrique | Description | Seuil d'alerte suggéré |
|----------|-------------|------------------------|
| Nombre de fichiers par partition | Indicateur de fragmentation | > 500 fichiers |
| Taille moyenne des fichiers | Performance de lecture | < 32 MB ou > 512 MB |
| Latence d'ingestion | Fraîcheur des données | > 15 minutes du SLA |
| Taux d'échec des requêtes | Santé du système | > 1 % |
| Durée de compaction | Efficacité de maintenance | > 2x moyenne historique |
| Croissance du stockage | Coûts et capacité | > 20 % /semaine |

**Architecture d'observabilité** : Une architecture typique comprend la collecte des métriques via Prometheus ou DataDog, la visualisation via Grafana, et les alertes configurées pour notifier les équipes via Slack, PagerDuty ou courriel.

L'architecture d'observabilité pour un lakehouse Iceberg doit couvrir plusieurs dimensions : la santé des tables, la performance des requêtes, l'état des pipelines et les métriques d'infrastructure.

```yaml
# Configuration Prometheus pour métriques Iceberg
global:
  scrape_interval: 30s
  evaluation_interval: 30s

rule_files:
  - /etc/prometheus/rules/iceberg_alerts.yml

scrape_configs:
  - job_name: 'spark-metrics'
    static_configs:
      - targets: ['spark-master:4040']
    metrics_path: /metrics/prometheus
    
  - job_name: 'trino-metrics'
    static_configs:
      - targets: ['trino-coordinator:8080']
    metrics_path: /v1/jmx
    
  - job_name: 'nessie-metrics'
    static_configs:
      - targets: ['nessie:19120']
    metrics_path: /api/v2/metrics
```

```yaml
# Règles d'alerte Prometheus pour le lakehouse
groups:
  - name: iceberg_table_health
    rules:
      - alert: IcebergTableFragmented
        expr: iceberg_table_file_count > 500
        for: 30m
        labels:
          severity: warning
          team: data-engineering
        annotations:
          summary: "Table Iceberg fragmentée: {{ $labels.table_name }}"
          description: "La table {{ $labels.table_name }} contient {{ $value }} fichiers. Compaction recommandée."
          runbook_url: "https://runbooks.example.com/iceberg/compaction"
      
      - alert: IcebergIngestionLatency
        expr: iceberg_last_commit_age_seconds > 3600
        for: 15m
        labels:
          severity: critical
          team: data-engineering
        annotations:
          summary: "Retard d'ingestion: {{ $labels.table_name }}"
          description: "La table {{ $labels.table_name }} n'a pas été mise à jour depuis {{ $value | humanizeDuration }}."
          
      - alert: IcebergSnapshotGrowth
        expr: rate(iceberg_snapshot_count[1h]) > 100
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "Croissance rapide des snapshots: {{ $labels.table_name }}"
          description: "La table génère des snapshots à un rythme élevé. Vérifier la fréquence d'écriture."
          
      - alert: IcebergStorageGrowth
        expr: rate(iceberg_table_size_bytes[24h]) / iceberg_table_size_bytes > 0.2
        for: 2h
        labels:
          severity: warning
        annotations:
          summary: "Croissance du stockage > 20%/jour: {{ $labels.table_name }}"
          description: "Croissance inhabituelle du stockage. Vérifier les politiques de rétention."

  - name: iceberg_query_performance
    rules:
      - alert: IcebergSlowQueries
        expr: histogram_quantile(0.95, rate(query_execution_time_seconds_bucket{engine="trino"}[5m])) > 30
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Requêtes lentes détectées"
          description: "Le 95e percentile des requêtes dépasse 30 secondes."
          
      - alert: IcebergQueryFailureRate
        expr: rate(query_failures_total[5m]) / rate(query_total[5m]) > 0.01
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Taux d'échec des requêtes élevé"
          description: "Plus de 1% des requêtes échouent. Investiguer immédiatement."
```

### Intégration avec les plateformes d'observabilité

Les organisations utilisent souvent des plateformes d'observabilité intégrées qui combinent métriques, journaux et traces. L'intégration avec ces plateformes est essentielle pour une vision unifiée.

**Intégration DataDog** : DataDog offre des intégrations natives pour de nombreux composants du lakehouse, incluant Spark, Trino et AWS S3.

```python
# Émission de métriques personnalisées vers DataDog
from datadog import initialize, statsd
from pyspark.sql import SparkSession

initialize(statsd_host='datadog-agent', statsd_port=8125)

def emit_table_metrics(spark: SparkSession, table_name: str):
    """Émet les métriques de santé d'une table Iceberg vers DataDog."""
    
    # Récupérer les statistiques de fichiers
    files_df = spark.sql(f"""
        SELECT 
            COUNT(*) as file_count,
            AVG(file_size_in_bytes) as avg_file_size,
            SUM(file_size_in_bytes) as total_size,
            SUM(record_count) as total_records
        FROM {table_name}.files
    """).collect()[0]
    
    # Récupérer les statistiques de snapshots
    snapshots_df = spark.sql(f"""
        SELECT 
            COUNT(*) as snapshot_count,
            MAX(committed_at) as last_commit
        FROM {table_name}.snapshots
    """).collect()[0]
    
    # Émettre les métriques
    tags = [f"table:{table_name}", "lakehouse:prod"]
    
    statsd.gauge('iceberg.table.file_count', files_df['file_count'], tags=tags)
    statsd.gauge('iceberg.table.avg_file_size_mb', 
                 files_df['avg_file_size'] / 1024 / 1024, tags=tags)
    statsd.gauge('iceberg.table.total_size_gb', 
                 files_df['total_size'] / 1024 / 1024 / 1024, tags=tags)
    statsd.gauge('iceberg.table.total_records', files_df['total_records'], tags=tags)
    statsd.gauge('iceberg.table.snapshot_count', snapshots_df['snapshot_count'], tags=tags)
    
    # Calculer l'âge du dernier commit
    from datetime import datetime
    if snapshots_df['last_commit']:
        last_commit_age = (datetime.utcnow() - snapshots_df['last_commit']).total_seconds()
        statsd.gauge('iceberg.table.last_commit_age_seconds', last_commit_age, tags=tags)
```

**Intégration OpenTelemetry** : OpenTelemetry fournit un standard ouvert pour la collecte de métriques, traces et journaux. Cette approche offre flexibilité et évite le verrouillage fournisseur.

```python
# Configuration OpenTelemetry pour le lakehouse
from opentelemetry import trace, metrics
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter

# Configuration du traceur
trace.set_tracer_provider(TracerProvider())
tracer = trace.get_tracer("lakehouse.iceberg")

otlp_exporter = OTLPSpanExporter(endpoint="otel-collector:4317", insecure=True)
trace.get_tracer_provider().add_span_processor(
    BatchSpanProcessor(otlp_exporter)
)

# Configuration des métriques
metrics.set_meter_provider(MeterProvider())
meter = metrics.get_meter("lakehouse.iceberg")

# Création des instruments de métrique
file_count_gauge = meter.create_observable_gauge(
    "iceberg.table.file_count",
    callbacks=[lambda: get_file_counts()],
    description="Nombre de fichiers par table Iceberg"
)

query_duration_histogram = meter.create_histogram(
    "iceberg.query.duration",
    unit="ms",
    description="Durée des requêtes Iceberg"
)

# Exemple d'instrumentation d'une opération
def instrumented_write(df, table_name: str):
    """Écriture instrumentée avec tracing."""
    with tracer.start_as_current_span("iceberg.write") as span:
        span.set_attribute("table.name", table_name)
        span.set_attribute("records.count", df.count())
        
        start_time = time.time()
        df.writeTo(table_name).append()
        duration_ms = (time.time() - start_time) * 1000
        
        query_duration_histogram.record(duration_ms, {"operation": "write", "table": table_name})
        span.set_attribute("duration.ms", duration_ms)
```

Les plateformes d'observabilité des données comme Monte Carlo, Bigeye ou Great Expectations complètent cette architecture en surveillant la qualité des données elles-mêmes : distribution des valeurs, valeurs nulles inattendues, anomalies de volume.

> **Performance**
> L'exploitation intensive des tables de métadonnées Iceberg pour l'audit peut impacter les performances du catalogue, particulièrement pour les tables avec un historique long. Une bonne pratique consiste à extraire périodiquement ces métadonnées vers des tables d'audit dédiées, réduisant la charge sur le catalogue principal.

### Conformité et rapports réglementaires

Les organisations assujetties à des réglementations doivent produire régulièrement des rapports de conformité démontrant la traçabilité et la protection des données.

**Rapports RGPD/Loi 25** : Ces rapports documentent le traitement des données personnelles, incluant les bases légales du traitement, les durées de rétention et les accès effectués.

```sql
-- Rapport d'accès aux données personnelles (derniers 90 jours)
SELECT 
    q.user_email,
    t.table_name,
    COUNT(*) as access_count,
    MAX(q.timestamp) as last_access,
    t.pii_classification
FROM audit.query_logs q
JOIN metadata.table_classifications t 
    ON ARRAY_CONTAINS(q.tables_accessed, t.table_name)
WHERE q.timestamp > CURRENT_TIMESTAMP - INTERVAL 90 DAY
    AND t.pii_classification IN ('PII', 'Sensitive-PII')
GROUP BY q.user_email, t.table_name, t.pii_classification
ORDER BY access_count DESC;
```

**Rapports SOX** : Pour les entreprises cotées en bourse, les rapports SOX documentent les contrôles sur les données financières, incluant la séparation des tâches et les approbations.

**Demandes de droit d'accès** : Le *time travel* d'Iceberg facilite la réponse aux demandes de droit d'accès des individus en permettant de reconstituer l'état des données personnelles à une date donnée.

```sql
-- Reconstituer l'état des données d'un client à une date spécifique
SELECT *
FROM lakehouse.gold.customer_360 
FOR TIMESTAMP AS OF TIMESTAMP '2024-06-15 00:00:00'
WHERE customer_id = 'CUST-12345';
```

> **Étude de cas : Mouvement Desjardins**
> *Secteur* : Services financiers
> *Défi* : Répondre aux exigences de l'Autorité des marchés financiers (AMF) et de la Loi 25 pour la traçabilité des données des 7 millions de membres.
> *Solution* : Déploiement d'une architecture d'audit basée sur les tables de métadonnées Iceberg, enrichie par des journaux applicatifs centralisés dans Elasticsearch. Chaque modification de données membres est associée à un ticket de changement et un approbateur.
> *Résultats* : Temps de réponse aux demandes de l'AMF réduit de 5 jours à 4 heures. Capacité de produire un rapport complet de lignée pour n'importe quel point de données en moins de 30 minutes. Conformité Loi 25 attestée par audit externe.

---

## IV.11.3 Récupération Après Sinistre

La récupération après sinistre (Disaster Recovery, DR) constitue l'ultime filet de sécurité d'un lakehouse en production. Malgré toutes les précautions, des événements catastrophiques peuvent survenir : défaillance d'une région infonuagique, corruption de données à grande échelle, erreur humaine dévastatrice ou cyberattaque. Une stratégie de DR robuste garantit la continuité des opérations face à ces scénarios.

### Définir les objectifs de récupération

Avant de concevoir une stratégie de DR, l'organisation doit définir deux métriques fondamentales pour chaque classe de données.

**RTO (Recovery Time Objective)** : Le temps maximum acceptable entre la survenue d'un sinistre et la restauration du service. Un RTO de 4 heures signifie que le lakehouse doit être opérationnel dans les 4 heures suivant l'incident.

**RPO (Recovery Point Objective)** : La quantité maximale de données que l'organisation peut se permettre de perdre. Un RPO de 1 heure signifie que les sauvegardes doivent avoir moins d'une heure d'ancienneté.

Ces objectifs varient selon la criticité des données. Les tables alimentant les systèmes transactionnels peuvent exiger un RTO de 15 minutes et un RPO de 5 minutes, tandis que les données d'archivage peuvent tolérer un RTO de 24 heures et un RPO d'une semaine.

| Classe de données | Exemple | RTO cible | RPO cible |
|-------------------|---------|-----------|-----------|
| Critique | Données clients actifs | 15 min | 5 min |
| Important | Rapports financiers | 2 heures | 1 heure |
| Standard | Analyses marketing | 8 heures | 4 heures |
| Archive | Données historiques | 24 heures | 1 semaine |

### Capacités natives d'Iceberg pour la récupération

L'architecture d'Apache Iceberg offre des capacités de récupération supérieures aux formats de données traditionnels, grâce à plusieurs caractéristiques fondamentales.

**Immutabilité des fichiers** : Les fichiers de données Iceberg ne sont jamais modifiés une fois écrits. Chaque modification crée de nouveaux fichiers, préservant l'historique complet. Cette immutabilité signifie qu'une corruption ou suppression accidentelle n'affecte pas les snapshots historiques tant que leurs fichiers sont préservés.

**Snapshots comme points de récupération** : Chaque snapshot représente un état cohérent et complet de la table. La récupération vers un snapshot antérieur est une opération atomique qui ne nécessite aucune restauration de fichiers.

**Séparation métadonnées/données** : Les métadonnées Iceberg (fichiers JSON et manifest) sont distinctes des fichiers de données. Cette séparation permet des stratégies de sauvegarde différenciées et facilite la récupération en cas de corruption partielle.

**Procédures de récupération intégrées** : Iceberg fournit des procédures Spark pour les opérations de récupération courantes, notamment `rollback_to_snapshot`, `rollback_to_timestamp` et `register_table`.

### Scénarios de récupération et solutions

L'article de Dremio (2025) sur la récupération après sinistre pour les tables Iceberg identifie plusieurs scénarios de défaillance et leurs solutions.

**Scénario 1 : Erreur humaine — suppression ou modification accidentelle de données**

Ce scénario représente la cause la plus fréquente de perte de données. Un ingénieur exécute par erreur un `DELETE` sans clause `WHERE`, ou un script de migration corrompt des enregistrements.

*Solution* : Utiliser la procédure `rollback_to_snapshot` ou `rollback_to_timestamp` pour revenir à l'état antérieur à l'erreur.

```sql
-- Identifier le snapshot cible
SELECT snapshot_id, committed_at, operation, summary
FROM lakehouse.gold.customer_360.snapshots
WHERE committed_at < TIMESTAMP '2025-01-15 14:30:00'
ORDER BY committed_at DESC
LIMIT 5;

-- Revenir au snapshot avant l'erreur
CALL iceberg.system.rollback_to_snapshot(
    'lakehouse.gold.customer_360',
    8744736658442914487
);

-- Ou revenir à un horodatage spécifique
CALL iceberg.system.rollback_to_timestamp(
    'lakehouse.gold.customer_360',
    TIMESTAMP '2025-01-15 14:00:00'
);
```

Cette récupération est quasi instantanée car elle ne modifie que le pointeur du snapshot courant, sans copier de données.

**Scénario 2 : Corruption du catalogue**

Le catalogue Iceberg (Hive Metastore, Nessie, REST Catalog) peut être corrompu ou indisponible, rendant les tables inaccessibles même si les fichiers de données sont intacts.

*Solution* : Utiliser la procédure `register_table` pour recréer l'entrée du catalogue en pointant vers le fichier de métadonnées le plus récent.

```sql
-- Ré-enregistrer une table à partir de son fichier metadata.json
CALL spark_catalog.system.register_table(
    table => 'lakehouse.gold.customer_360',
    metadata_file => 's3a://lakehouse-bucket/gold/customer_360/metadata/v17.metadata.json'
);
```

> **Migration**
> *De* : Catalogue corrompu ou inaccessible
> *Vers* : Table réenregistrée dans un nouveau catalogue
> *Stratégie* : Localiser le fichier `metadata.json` le plus récent dans le répertoire `metadata/` de la table, puis utiliser `register_table` pour recréer l'entrée du catalogue. Maintenir un inventaire des chemins metadata.json comme partie de la documentation de DR.

**Scénario 3 : Restauration depuis une sauvegarde vers un nouvel emplacement**

Lors d'une restauration de sauvegarde, les fichiers peuvent être copiés vers un emplacement différent de l'original. Les métadonnées Iceberg utilisent des chemins absolus, qui deviennent alors invalides.

*Solution* : Utiliser la procédure `rewrite_table_path` pour mettre à jour tous les chemins dans les métadonnées.

```sql
-- Réécrire les chemins après restauration vers un nouvel emplacement
CALL spark_catalog.system.rewrite_table_path(
    table => 'lakehouse.gold.customer_360',
    source_prefix => 'hdfs://old-cluster/warehouse/customer_360',
    target_prefix => 's3a://dr-bucket/warehouse/customer_360'
);
```

Cette procédure génère un fichier CSV listant tous les fichiers à copier physiquement vers le nouvel emplacement.

**Scénario 4 : Perte complète d'une région infonuagique**

Ce scénario catastrophique nécessite une réplication préventive vers une région secondaire.

*Solution* : Implémenter une réplication continue des fichiers de données et de métadonnées vers une région DR. Les fournisseurs de stockage objet comme AWS S3 et Azure Blob offrent des fonctionnalités de réplication inter-région. Une approche complémentaire consiste à utiliser des outils comme Apache DistCp pour Hadoop ou des solutions de réplication spécialisées.

### Stratégie de sauvegarde pour un lakehouse Iceberg

Une stratégie de sauvegarde complète pour un lakehouse Iceberg doit couvrir trois composants : les métadonnées du catalogue, les métadonnées des tables et les fichiers de données.

**Sauvegarde des métadonnées du catalogue** : Le catalogue (Hive Metastore, base de données Nessie, etc.) doit être sauvegardé régulièrement. Pour un Hive Metastore MySQL, cela signifie des sauvegardes de base de données. Pour Nessie, l'exportation du journal des commits.

**Sauvegarde des métadonnées Iceberg** : Les fichiers `metadata.json` et les manifests doivent être inclus dans les sauvegardes. Une bonne pratique consiste à maintenir un inventaire des fichiers metadata.json pour chaque table, avec leur horodatage.

```python
# Script de catalogage des fichiers metadata.json pour DR
import boto3
from datetime import datetime

def catalog_metadata_files(bucket_name: str, table_prefix: str):
    """Catalogue les fichiers metadata.json pour faciliter la récupération."""
    s3 = boto3.client('s3')
    
    paginator = s3.get_paginator('list_objects_v2')
    pages = paginator.paginate(
        Bucket=bucket_name,
        Prefix=f"{table_prefix}/metadata/"
    )
    
    metadata_files = []
    for page in pages:
        for obj in page.get('Contents', []):
            if obj['Key'].endswith('.metadata.json'):
                metadata_files.append({
                    'path': f"s3a://{bucket_name}/{obj['Key']}",
                    'last_modified': obj['LastModified'].isoformat(),
                    'size_bytes': obj['Size']
                })
    
    # Trier par date, le plus récent en premier
    metadata_files.sort(key=lambda x: x['last_modified'], reverse=True)
    
    return metadata_files
```

**Sauvegarde des fichiers de données** : Pour les fichiers de données (Parquet, ORC), la stratégie dépend des objectifs RPO. Une réplication synchrone vers une région secondaire offre un RPO proche de zéro mais augmente les coûts. Une réplication asynchrone quotidienne offre un compromis coût/RPO acceptable pour de nombreux cas d'usage.

### Configuration de la réplication S3 Cross-Region

AWS S3 offre une fonctionnalité de réplication inter-région (CRR) qui peut être configurée pour répliquer automatiquement les fichiers du lakehouse vers une région DR. Cette configuration est essentielle pour les organisations déployées sur AWS.

```json
{
  "Role": "arn:aws:iam::123456789012:role/s3-replication-role",
  "Rules": [
    {
      "ID": "ReplicateLakehouseData",
      "Status": "Enabled",
      "Priority": 1,
      "Filter": {
        "Prefix": "warehouse/"
      },
      "Destination": {
        "Bucket": "arn:aws:s3:::lakehouse-dr-bucket",
        "StorageClass": "STANDARD",
        "ReplicationTime": {
          "Status": "Enabled",
          "Time": {
            "Minutes": 15
          }
        },
        "Metrics": {
          "Status": "Enabled",
          "EventThreshold": {
            "Minutes": 15
          }
        }
      },
      "DeleteMarkerReplication": {
        "Status": "Enabled"
      }
    }
  ]
}
```

Pour Azure Blob Storage, la fonctionnalité équivalente s'appelle *Object Replication*. Pour Google Cloud Storage, il s'agit de *Dual-region* ou *Multi-region* buckets combinés avec des politiques de réplication personnalisées.

### Création d'un inventaire DR complet

Un inventaire DR complet facilite la récupération en documentant tous les composants du lakehouse et leurs emplacements de sauvegarde.

```python
# Script complet d'inventaire DR
import boto3
import json
from datetime import datetime
from typing import List, Dict

class DRInventoryManager:
    """Gère l'inventaire de récupération après sinistre pour un lakehouse Iceberg."""
    
    def __init__(self, warehouse_bucket: str, inventory_bucket: str):
        self.warehouse_bucket = warehouse_bucket
        self.inventory_bucket = inventory_bucket
        self.s3 = boto3.client('s3')
    
    def list_all_tables(self) -> List[str]:
        """Découvre toutes les tables Iceberg dans le warehouse."""
        tables = []
        paginator = self.s3.get_paginator('list_objects_v2')
        
        # Rechercher les répertoires metadata/ qui indiquent une table Iceberg
        pages = paginator.paginate(
            Bucket=self.warehouse_bucket,
            Delimiter='/'
        )
        
        for page in pages:
            for prefix in page.get('CommonPrefixes', []):
                table_prefix = prefix['Prefix'].rstrip('/')
                # Vérifier si c'est une table Iceberg (présence de metadata/)
                try:
                    self.s3.head_object(
                        Bucket=self.warehouse_bucket,
                        Key=f"{table_prefix}/metadata/"
                    )
                    tables.append(table_prefix)
                except:
                    pass
        
        return tables
    
    def get_table_metadata_files(self, table_prefix: str) -> List[Dict]:
        """Liste tous les fichiers metadata.json pour une table."""
        metadata_files = []
        paginator = self.s3.get_paginator('list_objects_v2')
        
        pages = paginator.paginate(
            Bucket=self.warehouse_bucket,
            Prefix=f"{table_prefix}/metadata/"
        )
        
        for page in pages:
            for obj in page.get('Contents', []):
                if obj['Key'].endswith('.metadata.json'):
                    metadata_files.append({
                        'path': f"s3a://{self.warehouse_bucket}/{obj['Key']}",
                        'last_modified': obj['LastModified'].isoformat(),
                        'size_bytes': obj['Size'],
                        'version': self._extract_version(obj['Key'])
                    })
        
        metadata_files.sort(key=lambda x: x['last_modified'], reverse=True)
        return metadata_files
    
    def _extract_version(self, key: str) -> int:
        """Extrait le numéro de version du nom de fichier metadata."""
        import re
        match = re.search(r'v(\d+)\.metadata\.json$', key)
        return int(match.group(1)) if match else 0
    
    def create_full_inventory(self) -> Dict:
        """Crée un inventaire complet du lakehouse pour DR."""
        tables = self.list_all_tables()
        
        inventory = {
            'generated_at': datetime.utcnow().isoformat(),
            'warehouse_bucket': self.warehouse_bucket,
            'table_count': len(tables),
            'tables': []
        }
        
        for table_path in tables:
            metadata_files = self.get_table_metadata_files(table_path)
            
            if metadata_files:
                table_info = {
                    'table_path': table_path,
                    'full_name': table_path.replace('/', '.'),
                    'latest_metadata': metadata_files[0]['path'],
                    'latest_version': metadata_files[0]['version'],
                    'metadata_count': len(metadata_files),
                    'last_updated': metadata_files[0]['last_modified'],
                    'recovery_commands': self._generate_recovery_commands(
                        table_path, 
                        metadata_files[0]['path']
                    )
                }
                inventory['tables'].append(table_info)
        
        return inventory
    
    def _generate_recovery_commands(self, table_path: str, metadata_file: str) -> Dict:
        """Génère les commandes de récupération pour une table."""
        table_name = table_path.replace('/', '.')
        return {
            'register_table': f"""
CALL spark_catalog.system.register_table(
    table => '{table_name}',
    metadata_file => '{metadata_file}'
);""",
            'verify_table': f"""
SELECT COUNT(*) as row_count, 
       MAX(updated_at) as last_update
FROM {table_name};""",
            'list_snapshots': f"""
SELECT snapshot_id, committed_at, operation
FROM {table_name}.snapshots
ORDER BY committed_at DESC
LIMIT 10;"""
        }
    
    def save_inventory(self, inventory: Dict) -> str:
        """Sauvegarde l'inventaire dans le bucket de sauvegarde."""
        inventory_key = f"dr-inventory/{datetime.utcnow().strftime('%Y/%m/%d/%H%M%S')}/inventory.json"
        
        self.s3.put_object(
            Bucket=self.inventory_bucket,
            Key=inventory_key,
            Body=json.dumps(inventory, indent=2, default=str),
            ContentType='application/json'
        )
        
        # Créer également un pointeur vers la dernière version
        self.s3.put_object(
            Bucket=self.inventory_bucket,
            Key="dr-inventory/latest.json",
            Body=json.dumps(inventory, indent=2, default=str),
            ContentType='application/json'
        )
        
        return f"s3://{self.inventory_bucket}/{inventory_key}"


# Utilisation
if __name__ == "__main__":
    manager = DRInventoryManager(
        warehouse_bucket="lakehouse-prod",
        inventory_bucket="lakehouse-dr-backups"
    )
    
    inventory = manager.create_full_inventory()
    location = manager.save_inventory(inventory)
    
    print(f"Inventaire DR créé : {location}")
    print(f"Tables documentées : {inventory['table_count']}")
```

### Automatisation des tests de récupération

Les tests de récupération doivent être automatisés et exécutés régulièrement. Un pipeline de test DR valide la capacité à restaurer les tables critiques dans les délais RTO définis.

```python
# Pipeline de test DR automatisé
from datetime import datetime, timedelta
import time
from typing import List, Dict, Optional

class DRTestRunner:
    """Exécute et valide les tests de récupération après sinistre."""
    
    def __init__(self, spark_session, dr_config: Dict):
        self.spark = spark_session
        self.config = dr_config
        self.results: List[Dict] = []
    
    def test_snapshot_rollback(self, table_name: str, max_rto_seconds: int) -> Dict:
        """Teste la récupération par rollback de snapshot."""
        start_time = time.time()
        
        try:
            # Identifier le snapshot cible (24h avant)
            target_time = datetime.utcnow() - timedelta(hours=24)
            
            snapshots = self.spark.sql(f"""
                SELECT snapshot_id, committed_at 
                FROM {table_name}.snapshots
                WHERE committed_at < '{target_time.isoformat()}'
                ORDER BY committed_at DESC
                LIMIT 1
            """).collect()
            
            if not snapshots:
                return self._record_result(table_name, "rollback", False, 
                    "Aucun snapshot disponible", time.time() - start_time)
            
            target_snapshot = snapshots[0]['snapshot_id']
            
            # Créer une branche de test pour ne pas affecter la production
            self.spark.sql(f"""
                ALTER TABLE {table_name}
                CREATE BRANCH dr_test_{datetime.utcnow().strftime('%Y%m%d%H%M%S')} 
                AS OF VERSION {target_snapshot}
            """)
            
            branch_name = f"dr_test_{datetime.utcnow().strftime('%Y%m%d%H%M%S')}"
            
            # Valider la branche
            count = self.spark.sql(f"""
                SELECT COUNT(*) as cnt FROM {table_name}.branch_{branch_name}
            """).collect()[0]['cnt']
            
            elapsed = time.time() - start_time
            
            # Nettoyer
            self.spark.sql(f"ALTER TABLE {table_name} DROP BRANCH {branch_name}")
            
            success = elapsed <= max_rto_seconds and count > 0
            return self._record_result(
                table_name, "rollback", success,
                f"Durée: {elapsed:.2f}s, Enregistrements: {count}, Snapshot: {target_snapshot}",
                elapsed
            )
            
        except Exception as e:
            elapsed = time.time() - start_time
            return self._record_result(table_name, "rollback", False, str(e), elapsed)
    
    def test_register_table(self, table_path: str, metadata_file: str, 
                           max_rto_seconds: int) -> Dict:
        """Teste la récupération par ré-enregistrement de table."""
        start_time = time.time()
        
        try:
            # Enregistrer la table avec un nouveau nom temporaire
            test_table_name = f"dr_test_{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}"
            
            self.spark.sql(f"""
                CALL iceberg.system.register_table(
                    table => 'dr_catalog.{test_table_name}',
                    metadata_file => '{metadata_file}'
                )
            """)
            
            # Valider que la table est accessible
            count = self.spark.sql(f"""
                SELECT COUNT(*) as cnt FROM dr_catalog.{test_table_name}
            """).collect()[0]['cnt']
            
            # Vérifier les métadonnées
            schema_cols = self.spark.sql(f"""
                DESCRIBE dr_catalog.{test_table_name}
            """).count()
            
            elapsed = time.time() - start_time
            
            # Nettoyer
            self.spark.sql(f"DROP TABLE dr_catalog.{test_table_name}")
            
            success = elapsed <= max_rto_seconds and count > 0
            return self._record_result(
                table_path, "register", success,
                f"Durée: {elapsed:.2f}s, Enregistrements: {count}, Colonnes: {schema_cols}",
                elapsed
            )
            
        except Exception as e:
            elapsed = time.time() - start_time
            return self._record_result(table_path, "register", False, str(e), elapsed)
    
    def test_time_travel_query(self, table_name: str, hours_back: int = 48) -> Dict:
        """Teste la capacité de requête time travel."""
        start_time = time.time()
        
        try:
            target_time = datetime.utcnow() - timedelta(hours=hours_back)
            
            # Exécuter une requête time travel
            result = self.spark.sql(f"""
                SELECT COUNT(*) as cnt
                FROM {table_name}
                FOR TIMESTAMP AS OF TIMESTAMP '{target_time.isoformat()}'
            """).collect()[0]['cnt']
            
            elapsed = time.time() - start_time
            
            success = result > 0
            return self._record_result(
                table_name, "time_travel", success,
                f"Durée: {elapsed:.2f}s, Enregistrements à t-{hours_back}h: {result}",
                elapsed
            )
            
        except Exception as e:
            elapsed = time.time() - start_time
            return self._record_result(table_name, "time_travel", False, str(e), elapsed)
    
    def _record_result(self, table: str, test_type: str, success: bool, 
                      message: str, duration: float) -> Dict:
        """Enregistre le résultat d'un test."""
        result = {
            'table': table,
            'test_type': test_type,
            'success': success,
            'message': message,
            'duration_seconds': round(duration, 2),
            'timestamp': datetime.utcnow().isoformat()
        }
        self.results.append(result)
        return result
    
    def run_full_test_suite(self, critical_tables: List[Dict]) -> Dict:
        """Exécute la suite complète de tests DR."""
        print(f"Démarrage des tests DR à {datetime.utcnow().isoformat()}")
        print(f"Tables à tester : {len(critical_tables)}")
        
        for table_config in critical_tables:
            table_name = table_config['name']
            max_rto = table_config.get('max_rto_seconds', 300)
            metadata_file = table_config.get('metadata_file')
            
            print(f"\nTest de {table_name}...")
            
            # Test 1: Rollback de snapshot
            self.test_snapshot_rollback(table_name, max_rto)
            
            # Test 2: Time travel
            self.test_time_travel_query(table_name)
            
            # Test 3: Register table (si metadata_file fourni)
            if metadata_file:
                self.test_register_table(table_name, metadata_file, max_rto)
        
        return self.generate_report()
    
    def generate_report(self) -> Dict:
        """Génère un rapport de test DR complet."""
        total = len(self.results)
        passed = sum(1 for r in self.results if r['success'])
        failed = total - passed
        
        avg_duration = sum(r['duration_seconds'] for r in self.results) / total if total > 0 else 0
        max_duration = max(r['duration_seconds'] for r in self.results) if self.results else 0
        
        report = {
            'summary': {
                'generated_at': datetime.utcnow().isoformat(),
                'total_tests': total,
                'passed': passed,
                'failed': failed,
                'success_rate': f"{(passed/total)*100:.1f}%" if total > 0 else "N/A",
                'average_duration_seconds': round(avg_duration, 2),
                'max_duration_seconds': round(max_duration, 2)
            },
            'by_test_type': {},
            'failures': [r for r in self.results if not r['success']],
            'details': self.results
        }
        
        # Agrégation par type de test
        for test_type in ['rollback', 'register', 'time_travel']:
            type_results = [r for r in self.results if r['test_type'] == test_type]
            if type_results:
                report['by_test_type'][test_type] = {
                    'total': len(type_results),
                    'passed': sum(1 for r in type_results if r['success']),
                    'avg_duration': round(
                        sum(r['duration_seconds'] for r in type_results) / len(type_results), 2
                    )
                }
        
        return report
```

> **Performance**
> La réplication inter-région peut avoir un impact significatif sur les performances d'écriture si elle est synchrone. Pour les tables à haut débit d'écriture, privilégier une réplication asynchrone avec un objectif RPO de 15-60 minutes, combinée à des snapshots horodatés réguliers permettant une récupération à un point précis dans le temps.

### Plan de récupération après sinistre

Un plan de DR documenté et testé est essentiel. Ce plan doit inclure les éléments suivants.

**Inventaire des actifs** : Liste de toutes les tables Iceberg, leur classification de criticité, et les chemins des fichiers metadata.json correspondants.

**Procédures de récupération** : Scripts et runbooks pour chaque scénario de défaillance identifié, avec les commandes exactes à exécuter.

**Rôles et responsabilités** : Identification des personnes autorisées à déclencher une récupération et des équipes responsables de chaque composant.

**Tests réguliers** : Simulation de sinistres et exécution des procédures de récupération dans un environnement de test, au minimum trimestriellement.

```markdown
# Runbook de récupération — Suppression accidentelle de données

## Prérequis
- Accès administrateur au cluster Spark
- Identifiants du catalogue Iceberg
- Connaissance de l'heure approximative de l'incident

## Étapes

1. **Identifier le snapshot cible**
   ```sql
   SELECT snapshot_id, committed_at, operation, summary
   FROM <table>.snapshots
   WHERE committed_at < '<heure_incident>'
   ORDER BY committed_at DESC
   LIMIT 10;
   ```

2. **Valider le snapshot** (vérifier que les données attendues sont présentes)
   ```sql
   SELECT COUNT(*), MAX(updated_at)
   FROM <table> FOR VERSION AS OF <snapshot_id>;
   ```

3. **Exécuter le rollback**
   ```sql
   CALL iceberg.system.rollback_to_snapshot('<table>', <snapshot_id>);
   ```

4. **Valider la récupération**
   ```sql
   SELECT COUNT(*), MAX(updated_at) FROM <table>;
   ```

5. **Notifier les parties prenantes**
   - Informer l'équipe data engineering
   - Créer un ticket d'incident
   - Documenter la cause racine

## Escalade
Si la récupération échoue, contacter le DBA de garde au [numéro].
```

### Réplication multi-région et haute disponibilité

Pour les organisations exigeant une haute disponibilité (HA) avec des RTO de quelques minutes, une architecture multi-région active-passive ou active-active est nécessaire.

**Architecture active-passive** : Une région primaire traite toutes les écritures, avec réplication asynchrone vers une région secondaire. En cas de sinistre, la région secondaire est promue en primaire. Cette architecture offre un bon compromis entre coût et résilience.

**Architecture active-active** : Les deux régions acceptent les écritures, avec résolution des conflits. Cette architecture est complexe à implémenter avec Iceberg en raison des garanties ACID, et nécessite généralement une couche de coordination externe.

Les catalogues modernes comme Apache Polaris et Nessie offrent des fonctionnalités de réplication qui facilitent la synchronisation des métadonnées entre régions. Combinées à la réplication de stockage objet, ces fonctionnalités permettent de construire des architectures HA sophistiquées.

### Conception d'une architecture multi-région

L'architecture multi-région pour un lakehouse Iceberg requiert une planification minutieuse des composants de stockage, de catalogue et de calcul.

**Composant stockage** : Le stockage objet constitue la fondation. AWS S3, Azure Blob Storage et Google Cloud Storage offrent tous des options de réplication inter-région. La configuration dépend des exigences RPO.

- **Réplication S3 Cross-Region** : Latence typique de 15 minutes pour la réplication. Convient pour RPO > 15 minutes.
- **S3 Multi-Region Access Points** : Permet des écritures automatiquement routées vers la région la plus proche. Convient pour les architectures actif-actif.
- **Azure Geo-Redundant Storage (GRS)** : Réplication synchrone vers une région secondaire. RPO proche de zéro mais surcoût significatif.

```bash
# Configuration AWS CLI pour activer la réplication CRR
aws s3api put-bucket-replication \
    --bucket lakehouse-primary \
    --replication-configuration file://replication-config.json

# Vérification du statut de réplication
aws s3api get-bucket-replication-metrics \
    --bucket lakehouse-primary \
    --id ReplicationRule1
```

**Composant catalogue** : Le catalogue doit être répliqué pour permettre la découverte des tables dans la région DR.

- **Nessie** : Supporte la réplication via export/import des commits. La communauté travaille sur une réplication native.
- **Apache Polaris** : Offre des API de synchronisation pour les environnements multi-région.
- **Hive Metastore** : Peut être répliqué via la réplication de la base de données sous-jacente (MySQL, PostgreSQL).

```python
# Script de synchronisation Nessie vers région DR
import requests
from datetime import datetime

class NessieReplicator:
    """Réplique les commits Nessie vers une région DR."""
    
    def __init__(self, source_uri: str, target_uri: str):
        self.source_uri = source_uri
        self.target_uri = target_uri
        self.last_synced_commit = None
    
    def get_commits(self, branch: str = "main", since_commit: str = None) -> list:
        """Récupère les commits depuis la source."""
        url = f"{self.source_uri}/api/v2/trees/{branch}/log"
        params = {}
        if since_commit:
            params['filter'] = f"commit.hash != '{since_commit}'"
        
        response = requests.get(url, params=params)
        return response.json().get('logEntries', [])
    
    def replicate_commits(self, commits: list):
        """Réplique les commits vers la cible."""
        for commit in reversed(commits):  # Du plus ancien au plus récent
            # Récupérer les opérations du commit
            operations = self.get_commit_operations(commit['commitMeta']['hash'])
            
            # Appliquer à la cible
            self.apply_operations(operations, commit['commitMeta'])
            
            self.last_synced_commit = commit['commitMeta']['hash']
    
    def get_commit_operations(self, commit_hash: str) -> list:
        """Récupère les opérations d'un commit."""
        url = f"{self.source_uri}/api/v2/trees/main/diff"
        params = {'from': f'{commit_hash}^', 'to': commit_hash}
        response = requests.get(url, params=params)
        return response.json().get('diffs', [])
    
    def apply_operations(self, operations: list, commit_meta: dict):
        """Applique les opérations à la cible."""
        url = f"{self.target_uri}/api/v2/trees/main/commit"
        payload = {
            'commitMeta': {
                'message': f"[REPLICATED] {commit_meta.get('message', '')}",
                'author': commit_meta.get('author'),
                'properties': {
                    'source_commit': commit_meta.get('hash'),
                    'replicated_at': datetime.utcnow().isoformat()
                }
            },
            'operations': operations
        }
        response = requests.post(url, json=payload)
        response.raise_for_status()
    
    def sync(self, branch: str = "main"):
        """Exécute une synchronisation incrémentale."""
        commits = self.get_commits(branch, self.last_synced_commit)
        if commits:
            print(f"Réplication de {len(commits)} commits...")
            self.replicate_commits(commits)
            print(f"Synchronisation terminée. Dernier commit: {self.last_synced_commit}")
        else:
            print("Aucun nouveau commit à répliquer.")
```

**Composant calcul** : Les clusters Spark, Trino ou Dremio doivent être déployés dans chaque région avec accès au stockage et au catalogue locaux.

### Considérations de coûts pour la DR

La récupération après sinistre engendre des coûts significatifs qu'il convient d'optimiser selon les exigences réelles.

| Composant | Coût mensuel typique (100 TB) | Stratégie d'optimisation |
|-----------|------------------------------|--------------------------|
| Réplication stockage | 500-2000 $ | Répliquer seulement les tables critiques |
| Stockage DR | 1000-3000 $ | Utiliser des classes de stockage économiques (S3 IA) |
| Infrastructure de test DR | 200-500 $ | Clusters éphémères pour les tests |
| Bande passante inter-région | 100-500 $ | Compression des données répliquées |

**Stratégies d'optimisation des coûts** :

1. **Classification des données** : Ne pas répliquer toutes les tables avec la même stratégie. Les données d'archivage peuvent tolérer un RPO plus long et utiliser une réplication asynchrone moins coûteuse.

2. **Réplication sélective** : Répliquer uniquement les métadonnées pour les tables volumineuses mais facilement reconstituables. Les fichiers de données peuvent être régénérés depuis les sources en cas de sinistre.

3. **Compression et déduplication** : Les outils de réplication peuvent compresser les données en transit, réduisant les coûts de bande passante.

4. **Classes de stockage DR** : Utiliser S3 Intelligent-Tiering ou S3 Glacier Instant Retrieval pour le stockage DR, réduisant les coûts tout en maintenant l'accessibilité.

```python
# Exemple de politique de réplication différenciée par criticité
replication_policies = {
    'critical': {
        'tables': ['customer_360', 'transactions', 'orders'],
        'storage_class': 'STANDARD',
        'replication_time_minutes': 15,
        'retain_snapshots_days': 30
    },
    'important': {
        'tables': ['product_catalog', 'inventory', 'pricing'],
        'storage_class': 'STANDARD_IA',
        'replication_time_minutes': 60,
        'retain_snapshots_days': 14
    },
    'standard': {
        'tables': ['analytics_*', 'reporting_*'],
        'storage_class': 'INTELLIGENT_TIERING',
        'replication_time_minutes': 240,
        'retain_snapshots_days': 7
    },
    'archive': {
        'tables': ['historical_*', 'audit_logs_*'],
        'storage_class': 'GLACIER_IR',
        'replication_time_minutes': 1440,  # 24 heures
        'retain_snapshots_days': 90
    }
}
```

> **Étude de cas : Banque Nationale du Canada**
> *Secteur* : Services financiers
> *Défi* : Répondre aux exigences du Bureau du surintendant des institutions financières (BSIF) pour la continuité des activités, avec un RTO de 2 heures et un RPO de 15 minutes pour les données critiques.
> *Solution* : Architecture multi-région active-passive avec lakehouse Iceberg principal à Toronto et DR à Montréal. Réplication S3 Cross-Region Replication pour les fichiers de données. Catalogue Nessie avec réplication des commits vers la région DR. Tests de basculement trimestriels.
> *Résultats* : RTO effectif de 45 minutes lors du dernier test de basculement. RPO de 12 minutes garanti par la fréquence de réplication. Conformité BSIF attestée.

---

## IV.11.4 Résumé

L'opérationnalisation d'Apache Iceberg transforme une architecture de données prometteuse en une plateforme de production fiable et résiliente. Ce chapitre a exploré les trois piliers de cette transformation : l'orchestration, l'audit et la récupération après sinistre.

### Orchestration du lakehouse

L'orchestration constitue le système nerveux central du lakehouse, coordonnant les pipelines d'ingestion, de transformation et de maintenance. Les orchestrateurs modernes se divisent en deux catégories : les outils orientés tâches comme Apache Airflow, et les outils orientés actifs comme Dagster.

Pour les lakehouses Iceberg, l'orchestration orientée actifs présente des avantages significatifs en termes de lignée, d'observabilité et de gestion des partitions. Les modèles d'orchestration incluent l'approche par horaire, l'approche événementielle et l'hybride, chacun adapté à des exigences de latence et de complexité différentes.

Les opérations de maintenance Iceberg (compaction, expiration des snapshots, nettoyage des orphelins) doivent être intégrées dans les pipelines orchestrés avec des stratégies conditionnelles basées sur les métriques des tables.

### Audit du lakehouse

L'audit répond aux exigences de conformité réglementaire tout en fournissant une traçabilité essentielle pour le diagnostic et l'analyse forensique. Les tables de métadonnées Iceberg offrent une base solide pour l'audit, capturant l'historique complet des modifications.

Une architecture d'audit complète comprend les journaux d'accès (générés par les moteurs de requête), les journaux de modification (via les snapshots Iceberg) et la lignée des données (maintenue par les catalogues et les orchestrateurs).

L'observabilité transforme les données d'audit en informations actionnables, avec des métriques clés à surveiller incluant la fragmentation des fichiers, la latence d'ingestion et les taux d'échec.

### Récupération après sinistre

La stratégie de DR doit être alignée sur les objectifs RTO et RPO définis pour chaque classe de données. Iceberg offre des capacités natives de récupération grâce à l'immutabilité des fichiers, les snapshots comme points de récupération et les procédures intégrées (rollback, register_table, rewrite_table_path).

Les scénarios de récupération couverts incluent les erreurs humaines (rollback vers un snapshot antérieur), la corruption du catalogue (ré-enregistrement des tables) et la restauration vers un nouvel emplacement (réécriture des chemins).

Pour les exigences de haute disponibilité, les architectures multi-région active-passive ou active-active peuvent être implémentées en combinant la réplication de stockage objet et la réplication des métadonnées du catalogue.

### Principes directeurs

L'opérationnalisation réussie d'un lakehouse Iceberg repose sur plusieurs principes directeurs :

1. **Automatisation systématique** : Toute opération répétitive doit être automatisée et orchestrée, éliminant les interventions manuelles sources d'erreurs.

2. **Observabilité native** : Les métriques, journaux et traces doivent être collectés et corrélés pour permettre un diagnostic rapide et une détection proactive des anomalies.

3. **Défense en profondeur** : La protection des données repose sur plusieurs couches : snapshots pour la récupération rapide, sauvegardes pour la protection contre la corruption, réplication pour la continuité régionale.

4. **Tests réguliers** : Les procédures de récupération doivent être testées régulièrement dans des conditions réalistes pour valider leur efficacité.

5. **Documentation vivante** : Les runbooks et procédures doivent être maintenus à jour et accessibles à toutes les équipes concernées.

Le chapitre suivant explorera l'évolution vers le *Streaming Lakehouse*, où l'intégration d'Apache Kafka avec Apache Iceberg permet de traiter les données en temps réel tout en maintenant les garanties de cohérence et la gouvernance du lakehouse.

---

### Points clés à retenir

- L'orchestration orientée actifs (Dagster, Prefect) s'aligne naturellement avec l'architecture Iceberg centrée sur les tables et les snapshots

- Les tables de métadonnées Iceberg (`snapshots`, `history`, `files`, `manifests`) constituent des sources d'audit natives nécessitant un enrichissement contextuel

- Le *time travel* Iceberg permet une récupération quasi instantanée des erreurs humaines sans restauration de fichiers

- La procédure `register_table` permet de reconstruire un catalogue corrompu à partir des fichiers de métadonnées

- La réplication multi-région combine la réplication du stockage objet et la synchronisation des métadonnées du catalogue

- Les tests de récupération réguliers sont essentiels pour valider les procédures et former les équipes

---

### Références

- Apache Iceberg Documentation (2025). *Maintenance Operations*. https://iceberg.apache.org/docs/latest/maintenance/
- Dagster (2025). *Building a Better Lakehouse: From Airflow to Dagster*. https://dagster.io/blog/building-a-better-lakehouse
- Dremio (2025). *Disaster Recovery for Apache Iceberg Tables*. https://www.dremio.com/blog/disaster-recovery-for-apache-iceberg-tables
- IOMETE (2025). *Iceberg Disaster Recovery*. https://iomete.com/resources/blog/iceberg-disaster-recovery
- PracData (2025). *State of Open Source Workflow Orchestration Systems 2025*. https://www.pracdata.io/p/state-of-workflow-orchestration-ecosystem-2025
- Atlan (2025). *Apache Iceberg Tables Governance: A Practical Guide*. https://atlan.com/know/iceberg/apache-iceberg-table-governance/
