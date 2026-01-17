# Chapitre IV.10 - Maintenir un Lakehouse Iceberg en Production

## Introduction

La mise en production d'un Data Lakehouse Apache Iceberg ne constitue pas une destination mais le début d'un voyage opérationnel continu. Les tables Iceberg, contrairement aux systèmes de fichiers statiques, sont des organismes vivants qui accumulent des fichiers, génèrent des métadonnées et évoluent au rythme des écritures et modifications. Sans maintenance proactive, cette croissance organique dégrade progressivement les performances, gonfle les coûts de stockage et compromet la fiabilité des requêtes. La différence entre un Lakehouse performant et un système léthargique réside souvent dans la qualité des pratiques opérationnelles plutôt que dans l'architecture initiale.

Apache Iceberg intègre des mécanismes sophistiqués de gestion du cycle de vie des données — compaction, expiration des snapshots, réécriture des fichiers de manifeste — mais ces fonctionnalités ne s'activent pas automatiquement. Elles requièrent une orchestration délibérée, un paramétrage adapté à votre contexte et une surveillance continue. Les organisations qui excellent dans l'exploitation de leur Lakehouse ont développé des pratiques opérationnelles matures, combinant automatisation intelligente, monitoring proactif et procédures de réponse aux incidents bien rodées.

Ce chapitre vous guide dans l'établissement de pratiques de maintenance robustes pour votre Lakehouse Iceberg en production. Nous examinerons en détail les opérations essentielles — compaction, expiration, optimisation —, leurs paramètres critiques et leurs stratégies d'orchestration. Nous aborderons le monitoring et l'observabilité, la gestion des incidents et les procédures de récupération. Les études de cas canadiennes illustreront comment des organisations de différents secteurs ont structuré leurs opérations pour maintenir performance et fiabilité dans la durée.

L'enjeu dépasse la simple exécution de tâches techniques. La maintenance d'un Lakehouse en production façonne l'expérience utilisateur quotidienne, influence les coûts d'exploitation et détermine la confiance que l'organisation accorde à sa plateforme de données. Un Lakehouse bien maintenu devient un actif stratégique ; mal maintenu, il devient une source de frustration et de risques.

---

## Anatomie de la Maintenance Iceberg

### Comprendre l'Accumulation des Fichiers

Chaque opération d'écriture dans une table Iceberg génère de nouveaux fichiers sans modifier les existants — c'est le principe fondamental de l'immutabilité qui garantit les propriétés ACID. Cette approche, bien que puissante, crée une accumulation naturelle de fichiers qui nécessite une gestion active.

**Sources d'accumulation** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    ACCUMULATION DES FICHIERS ICEBERG                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ÉCRITURES STREAMING                                                    │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Commit toutes les minutes → 1440 fichiers/jour/partition       │   │
│  │  Fichiers typiquement sous-dimensionnés (1-10 Mo)               │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ÉCRITURES BATCH FRÉQUENTES                                            │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Chargements horaires → 24 fichiers/jour/partition              │   │
│  │  Fichiers de taille variable selon volume                       │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  OPÉRATIONS UPDATE/DELETE                                              │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Copy-on-Write: nouveaux fichiers complets                      │   │
│  │  Merge-on-Read: fichiers de position deletes                    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  SNAPSHOTS ET MÉTADONNÉES                                              │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Chaque commit → nouveau snapshot + manifests                   │   │
│  │  Historique conservé pour Time Travel                           │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

**Impact de l'accumulation non gérée** :

| Symptôme                    | Cause                   | Impact                    |
| ---------------------------- | ----------------------- | ------------------------- |
| Ralentissement des requêtes | Trop de petits fichiers | Overhead de planification |
| Augmentation des coûts S3   | Fichiers orphelins      | Stockage inutile          |
| Lenteur du listage de tables | Manifests volumineux    | Latence metadata          |
| Échecs de commit            | Conflits de concurrence | Fiabilité dégradée     |
| Time Travel défaillant      | Snapshots corrompus     | Perte de fonctionnalité  |

### Cycle de Vie des Objets Iceberg

Comprendre le cycle de vie des différents objets Iceberg est essentiel pour une maintenance efficace.

**Fichiers de données** :

* Créés lors des écritures (INSERT, UPDATE via Copy-on-Write)
* Référencés par les manifests
* Deviennent orphelins après compaction ou expiration de snapshot
* Supprimables uniquement après déréférencement complet

**Fichiers de manifeste (Manifest Files)** :

* Listent les fichiers de données avec leurs statistiques
* Créés lors des commits
* Peuvent être réécrits pour optimisation
* Orphelins après réécriture ou expiration

**Listes de manifestes (Manifest Lists)** :

* Pointent vers les manifests pour un snapshot
* Un fichier par snapshot
* Supprimables avec le snapshot associé

**Fichiers de métadonnées (Metadata Files)** :

* Décrivent l'état complet de la table
* Un nouveau fichier par commit
* Contiennent l'historique des snapshots
* Le catalogue pointe vers le fichier courant

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    HIÉRARCHIE DES FICHIERS ICEBERG                     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  Catalogue                                                              │
│      │                                                                  │
│      └──▶ v00042.metadata.json (fichier métadonnées courant)           │
│               │                                                         │
│               ├──▶ snap-1234.avro (manifest list snapshot 1234)        │
│               │       │                                                 │
│               │       ├──▶ manifest-abc.avro                           │
│               │       │       ├──▶ data-001.parquet                    │
│               │       │       └──▶ data-002.parquet                    │
│               │       │                                                 │
│               │       └──▶ manifest-def.avro                           │
│               │               └──▶ data-003.parquet                    │
│               │                                                         │
│               └──▶ snap-1235.avro (manifest list snapshot 1235)        │
│                       │                                                 │
│                       └──▶ manifest-ghi.avro                           │
│                               ├──▶ data-001.parquet (réutilisé)        │
│                               └──▶ data-004.parquet (nouveau)          │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Opérations de Maintenance Essentielles

Les opérations de maintenance Iceberg se répartissent en plusieurs catégories selon leur fréquence et leur impact.

| Opération             | Fréquence         | Impact Performance      | Ressources    | Priorité |
| ---------------------- | ------------------ | ----------------------- | ------------- | --------- |
| Compaction             | Quotidienne        | Élevé (amélioration) | Moyen-Élevé | Critique  |
| Expiration snapshots   | Quotidienne        | Moyen (stockage)        | Faible        | Haute     |
| Nettoyage orphelins    | Hebdomadaire       | Faible (stockage)       | Faible        | Moyenne   |
| Réécriture manifests | Mensuelle          | Moyen (metadata)        | Faible        | Moyenne   |
| Optimisation tri       | Selon besoin       | Élevé (requêtes)     | Élevé       | Variable  |
| Collecte statistiques  | Après changements | Moyen (optimiseur)      | Moyen         | Haute     |

---

## Compaction des Fichiers de Données

### Principes et Stratégies

La compaction constitue l'opération de maintenance la plus critique pour les performances d'un Lakehouse Iceberg. Elle consolide les petits fichiers en fichiers plus volumineux, réduisant l'overhead de planification des requêtes et optimisant les lectures.

**Problème des petits fichiers** :

```
Avant compaction (streaming toutes les minutes pendant 24h):
├── data-00001.parquet (5 Mo)
├── data-00002.parquet (3 Mo)
├── data-00003.parquet (8 Mo)
├── ... (1437 fichiers)
└── data-01440.parquet (4 Mo)

Total: 1440 fichiers, ~6 Go, taille moyenne 4 Mo

Après compaction:
├── data-compacted-001.parquet (512 Mo)
├── data-compacted-002.parquet (512 Mo)
├── ... (10 fichiers)
└── data-compacted-012.parquet (480 Mo)

Total: 12 fichiers, ~6 Go, taille moyenne 500 Mo
```

**Impact sur les requêtes** :

```
Requête: SELECT SUM(montant) FROM transactions WHERE date = '2024-01-15'

Avant compaction:
- Planification: lecture de 1440 entrées de manifest
- Exécution: ouverture de 1440 fichiers
- Temps total: 45 secondes

Après compaction:
- Planification: lecture de 12 entrées de manifest
- Exécution: ouverture de 12 fichiers
- Temps total: 3 secondes

Amélioration: 15x
```

### Procédure de Compaction avec Spark

**Compaction de base (bin-packing)** :

```python
from pyspark.sql import SparkSession

spark = SparkSession.builder \
    .appName("IcebergCompaction") \
    .config("spark.sql.extensions", 
            "org.apache.iceberg.spark.extensions.IcebergSparkSessionExtensions") \
    .config("spark.sql.catalog.lakehouse", 
            "org.apache.iceberg.spark.SparkCatalog") \
    .config("spark.sql.catalog.lakehouse.type", "rest") \
    .config("spark.sql.catalog.lakehouse.uri", 
            "https://catalog.lakehouse.entreprise.ca") \
    .getOrCreate()

# Compaction simple - regroupe les petits fichiers
spark.sql("""
    CALL lakehouse.system.rewrite_data_files(
        table => 'gold.transactions',
        options => map(
            'target-file-size-bytes', '536870912',
            'min-file-size-bytes', '104857600',
            'max-file-size-bytes', '805306368',
            'min-input-files', '5',
            'max-concurrent-file-group-rewrites', '10',
            'partial-progress.enabled', 'true',
            'partial-progress.max-commits', '10'
        )
    )
""")
```

**Paramètres de compaction** :

| Paramètre                             | Description                             | Valeur recommandée |
| -------------------------------------- | --------------------------------------- | ------------------- |
| `target-file-size-bytes`             | Taille cible des fichiers               | 512 Mo (536870912)  |
| `min-file-size-bytes`                | Seuil minimum pour inclusion            | 100 Mo              |
| `max-file-size-bytes`                | Taille maximale autorisée              | 768 Mo              |
| `min-input-files`                    | Nombre min de fichiers pour déclencher | 5                   |
| `max-concurrent-file-group-rewrites` | Parallélisme                           | 10-20               |
| `partial-progress.enabled`           | Commits intermédiaires                 | true                |
| `partial-progress.max-commits`       | Limite commits partiels                 | 10                  |

**Compaction avec filtrage de partitions** :

```python
# Compaction ciblée sur partitions récentes
from datetime import datetime, timedelta

date_limite = (datetime.now() - timedelta(days=7)).strftime('%Y-%m-%d')

spark.sql(f"""
    CALL lakehouse.system.rewrite_data_files(
        table => 'gold.transactions',
        strategy => 'binpack',
        where => 'date_partition >= DATE ''{date_limite}''',
        options => map(
            'target-file-size-bytes', '536870912',
            'min-input-files', '3'
        )
    )
""")
```

### Compaction avec Tri (Sort Compaction)

La compaction avec tri réorganise physiquement les données selon un ordre optimal pour les patterns de requêtes, améliorant significativement l'efficacité du filtrage.

**Compaction avec tri simple** :

```python
# Tri par colonnes fréquemment filtrées
spark.sql("""
    CALL lakehouse.system.rewrite_data_files(
        table => 'gold.transactions',
        strategy => 'sort',
        sort_order => 'date_transaction ASC NULLS LAST, client_id ASC NULLS LAST',
        options => map(
            'target-file-size-bytes', '536870912',
            'rewrite-all', 'true'
        )
    )
""")
```

**Compaction Z-Order pour requêtes multi-colonnes** :

Le Z-Order organise les données pour optimiser les requêtes filtrant sur plusieurs colonnes simultanément — typique des analyses ad-hoc.

```python
# Z-Order sur colonnes de filtrage fréquent
spark.sql("""
    CALL lakehouse.system.rewrite_data_files(
        table => 'gold.transactions',
        strategy => 'sort',
        sort_order => 'zorder(region, categorie_produit, client_id)',
        options => map(
            'target-file-size-bytes', '536870912',
            'rewrite-all', 'true'
        )
    )
""")
```

> **Performance**
>
> Le Z-Order peut améliorer les performances de requêtes filtrant sur 2-3 colonnes de 3× à 10×. Cependant, il dégrade légèrement les requêtes filtrant sur une seule colonne par rapport à un tri simple. Analysez vos patterns de requêtes avant de choisir entre tri simple et Z-Order. Pour les tables avec un pattern de filtrage dominant (ex: toujours par date), le tri simple reste préférable.

### Stratégies de Compaction par Type de Table

**Tables de streaming (haute fréquence d'écriture)** :

```python
# Configuration agressive pour streaming
compaction_config_streaming = {
    'target-file-size-bytes': '268435456',  # 256 Mo (plus petit pour fraîcheur)
    'min-input-files': '3',  # Seuil bas pour compaction fréquente
    'max-concurrent-file-group-rewrites': '20',
    'partial-progress.enabled': 'true',
    'partial-progress.max-commits': '5'
}

# Exécution toutes les heures
spark.sql(f"""
    CALL lakehouse.system.rewrite_data_files(
        table => 'bronze.events_streaming',
        strategy => 'binpack',
        where => 'event_hour >= current_timestamp - INTERVAL 2 HOURS',
        options => map({', '.join(f"'{k}', '{v}'" for k, v in compaction_config_streaming.items())})
    )
""")
```

**Tables analytiques (requêtes complexes)** :

```python
# Configuration optimisée pour analytique
compaction_config_analytics = {
    'target-file-size-bytes': '536870912',  # 512 Mo
    'min-input-files': '5',
    'max-concurrent-file-group-rewrites': '10',
    'rewrite-all': 'false'  # Uniquement fichiers sous-optimaux
}

# Avec tri pour optimisation des requêtes
spark.sql("""
    CALL lakehouse.system.rewrite_data_files(
        table => 'gold.transactions',
        strategy => 'sort',
        sort_order => 'date_transaction DESC, region',
        options => map(
            'target-file-size-bytes', '536870912',
            'min-input-files', '5'
        )
    )
""")
```

**Tables historiques (archivage)** :

```python
# Configuration pour tables peu modifiées
compaction_config_archive = {
    'target-file-size-bytes': '1073741824',  # 1 Go (gros fichiers pour archivage)
    'min-input-files': '10',
    'rewrite-all': 'true'  # Optimisation complète
}
```

---

## Gestion des Snapshots

### Expiration des Snapshots

Les snapshots permettent le Time Travel et garantissent les lectures cohérentes, mais leur accumulation consomme de l'espace et ralentit les opérations de métadonnées. L'expiration contrôlée des anciens snapshots équilibre ces besoins.

**Procédure d'expiration** :

```python
from datetime import datetime, timedelta

# Expiration des snapshots plus vieux que 7 jours
retention_date = datetime.now() - timedelta(days=7)
retention_timestamp = retention_date.strftime('%Y-%m-%d %H:%M:%S')

spark.sql(f"""
    CALL lakehouse.system.expire_snapshots(
        table => 'gold.transactions',
        older_than => TIMESTAMP '{retention_timestamp}',
        retain_last => 10,
        max_concurrent_deletes => 50,
        stream_results => true
    )
""")
```

**Paramètres d'expiration** :

| Paramètre                 | Description                            | Recommandation           |
| -------------------------- | -------------------------------------- | ------------------------ |
| `older_than`             | Timestamp limite de rétention         | 7-30 jours selon besoins |
| `retain_last`            | Nombre minimum de snapshots conservés | 10-100                   |
| `max_concurrent_deletes` | Parallélisme de suppression           | 50-100                   |
| `stream_results`         | Affichage progressif                   | true                     |

**Politique de rétention par type de table** :

| Type de table         | Rétention recommandée | Justification                      |
| --------------------- | ----------------------- | ---------------------------------- |
| Streaming/temps réel | 3-7 jours               | Données fraîches, volume élevé |
| Transactionnelle      | 7-14 jours              | Audit court terme                  |
| Analytique            | 30-90 jours             | Analyses historiques               |
| Réglementée         | 365+ jours ou jamais    | Conformité légale                |
| Archive               | Conservation indéfinie | Historique complet                 |

**Expiration avec conservation de snapshots tagués** :

```python
# Conservation des snapshots de référence (ex: fin de mois)
# D'abord, tagger les snapshots importants
spark.sql("""
    ALTER TABLE lakehouse.gold.transactions
    CREATE TAG `fin_mois_2024_01`
    AS OF VERSION 1234567890
""")

# L'expiration ne supprime pas les snapshots tagués
spark.sql("""
    CALL lakehouse.system.expire_snapshots(
        table => 'gold.transactions',
        older_than => TIMESTAMP '2024-01-01 00:00:00',
        retain_last => 5
    )
    -- Les snapshots tagués sont préservés automatiquement
""")
```

### Nettoyage des Fichiers Orphelins

Les fichiers orphelins sont des fichiers de données qui ne sont plus référencés par aucun snapshot actif. Ils résultent de compactions, d'expirations de snapshots ou d'écritures avortées.

**Identification des fichiers orphelins** :

```python
# Listage des fichiers orphelins (dry run)
orphans_df = spark.sql("""
    CALL lakehouse.system.remove_orphan_files(
        table => 'gold.transactions',
        older_than => TIMESTAMP '2024-01-01 00:00:00',
        dry_run => true
    )
""")

print(f"Fichiers orphelins identifiés: {orphans_df.count()}")
orphans_df.show(truncate=False)
```

**Suppression des fichiers orphelins** :

```python
# Suppression effective (avec précautions)
spark.sql("""
    CALL lakehouse.system.remove_orphan_files(
        table => 'gold.transactions',
        older_than => TIMESTAMP '2024-01-08 00:00:00',
        location => 's3://entreprise-lakehouse/gold/transactions',
        dry_run => false,
        max_concurrent_deletes => 100
    )
""")
```

> **Migration**
>
> *De* : Nettoyage manuel ad-hoc des fichiers orphelins
>
> *Vers* : Procédure automatisée hebdomadaire avec validation
>
> *Stratégie* : Toujours exécuter en dry_run d'abord, valider la liste, puis exécuter la suppression. Conserver un délai de sécurité d'au moins 7 jours entre l'expiration des snapshots et le nettoyage des orphelins pour éviter de supprimer des fichiers encore nécessaires à des requêtes en cours.

**Script de nettoyage sécurisé** :

```python
from datetime import datetime, timedelta
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def safe_orphan_cleanup(spark, table: str, dry_run_first: bool = True):
    """Nettoyage sécurisé des fichiers orphelins."""
  
    # Délai de sécurité: 14 jours après dernier snapshot expiré
    safety_margin = timedelta(days=14)
    cutoff_date = datetime.now() - safety_margin
    cutoff_timestamp = cutoff_date.strftime('%Y-%m-%d %H:%M:%S')
  
    logger.info(f"Analyse des orphelins pour {table} (avant {cutoff_timestamp})")
  
    # Phase 1: Dry run
    orphans = spark.sql(f"""
        CALL lakehouse.system.remove_orphan_files(
            table => '{table}',
            older_than => TIMESTAMP '{cutoff_timestamp}',
            dry_run => true
        )
    """).collect()
  
    orphan_count = len(orphans)
    total_size = sum(row.file_size_in_bytes for row in orphans) / (1024**3)
  
    logger.info(f"Fichiers orphelins identifiés: {orphan_count} ({total_size:.2f} Go)")
  
    if orphan_count == 0:
        logger.info("Aucun fichier orphelin à supprimer")
        return
  
    if dry_run_first:
        logger.info("Mode dry_run activé - pas de suppression")
        return orphans
  
    # Phase 2: Suppression effective
    logger.info("Suppression des fichiers orphelins...")
    spark.sql(f"""
        CALL lakehouse.system.remove_orphan_files(
            table => '{table}',
            older_than => TIMESTAMP '{cutoff_timestamp}',
            dry_run => false,
            max_concurrent_deletes => 100
        )
    """)
  
    logger.info(f"Suppression terminée: {orphan_count} fichiers, {total_size:.2f} Go libérés")
    return orphan_count, total_size
```

### Réécriture des Fichiers de Manifeste

Les fichiers de manifeste peuvent devenir fragmentés ou contenir des références obsolètes. La réécriture optimise leur structure pour accélérer la planification des requêtes.

**Réécriture des manifestes** :

```python
# Réécriture pour consolider les manifestes
spark.sql("""
    CALL lakehouse.system.rewrite_manifests(
        table => 'gold.transactions',
        use_caching => true
    )
""")
```

**Quand réécrire les manifestes** :

* Après une série de compactions majeures
* Lorsque le nombre de manifestes dépasse 100-200
* Si la planification des requêtes devient lente
* Après des opérations de maintenance importantes

```python
# Vérification du nombre de manifestes
manifests_info = spark.sql("""
    SELECT COUNT(*) as manifest_count,
           SUM(added_data_files_count) as total_files,
           AVG(added_data_files_count) as avg_files_per_manifest
    FROM lakehouse.gold.transactions.manifests
""").collect()[0]

if manifests_info.manifest_count > 100:
    logger.info(f"Réécriture recommandée: {manifests_info.manifest_count} manifestes")
    spark.sql("""
        CALL lakehouse.system.rewrite_manifests(
            table => 'gold.transactions'
        )
    """)
```

---

## Monitoring et Observabilité

### Métriques Essentielles

Un monitoring efficace du Lakehouse Iceberg repose sur la collecte et l'analyse de métriques couvrant plusieurs dimensions.

**Métriques de santé des tables** :

```python
def collect_table_metrics(spark, table: str) -> dict:
    """Collecte les métriques de santé d'une table Iceberg."""
  
    # Statistiques des fichiers
    files_stats = spark.sql(f"""
        SELECT 
            COUNT(*) as total_files,
            SUM(file_size_in_bytes) / (1024*1024*1024) as total_size_gb,
            AVG(file_size_in_bytes) / (1024*1024) as avg_file_size_mb,
            MIN(file_size_in_bytes) / (1024*1024) as min_file_size_mb,
            MAX(file_size_in_bytes) / (1024*1024) as max_file_size_mb,
            SUM(CASE WHEN file_size_in_bytes < 104857600 THEN 1 ELSE 0 END) as small_files_count,
            SUM(record_count) as total_records
        FROM {table}.files
    """).collect()[0]
  
    # Statistiques des snapshots
    snapshots_stats = spark.sql(f"""
        SELECT 
            COUNT(*) as snapshot_count,
            MIN(committed_at) as oldest_snapshot,
            MAX(committed_at) as latest_snapshot
        FROM {table}.snapshots
    """).collect()[0]
  
    # Statistiques des manifestes
    manifests_stats = spark.sql(f"""
        SELECT 
            COUNT(*) as manifest_count,
            SUM(added_data_files_count) as total_manifest_entries
        FROM {table}.manifests
    """).collect()[0]
  
    # Statistiques des partitions
    partitions_stats = spark.sql(f"""
        SELECT COUNT(DISTINCT partition) as partition_count
        FROM {table}.files
    """).collect()[0]
  
    return {
        'table': table,
        'total_files': files_stats.total_files,
        'total_size_gb': float(files_stats.total_size_gb or 0),
        'avg_file_size_mb': float(files_stats.avg_file_size_mb or 0),
        'small_files_count': files_stats.small_files_count,
        'small_files_ratio': files_stats.small_files_count / files_stats.total_files if files_stats.total_files > 0 else 0,
        'total_records': files_stats.total_records,
        'snapshot_count': snapshots_stats.snapshot_count,
        'manifest_count': manifests_stats.manifest_count,
        'partition_count': partitions_stats.partition_count
    }
```

**Seuils d'alerte recommandés** :

| Métrique               | Seuil Warning | Seuil Critical | Action             |
| ----------------------- | ------------- | -------------- | ------------------ |
| Ratio petits fichiers   | > 20%         | > 50%          | Compaction urgente |
| Nombre de snapshots     | > 100         | > 500          | Expiration         |
| Nombre de manifestes    | > 100         | > 500          | Réécriture       |
| Taille moyenne fichier  | < 100 Mo      | < 50 Mo        | Compaction         |
| Fichiers orphelins (Go) | > 100         | > 500          | Nettoyage          |

### Dashboard Grafana pour Lakehouse

**Configuration Prometheus pour métriques Iceberg** :

```yaml
# prometheus-iceberg-rules.yaml
groups:
  - name: iceberg-table-health
    interval: 5m
    rules:
      # Métrique personnalisée: ratio de petits fichiers
      - record: iceberg_small_files_ratio
        expr: |
          iceberg_table_small_files_count / iceberg_table_total_files
        labels:
          severity: "{{ if gt .Value 0.5 }}critical{{ else if gt .Value 0.2 }}warning{{ else }}ok{{ end }}"
    
      # Alerte: trop de petits fichiers
      - alert: IcebergSmallFilesHigh
        expr: iceberg_small_files_ratio > 0.2
        for: 30m
        labels:
          severity: warning
        annotations:
          summary: "Table {{ $labels.table }} a {{ $value | humanizePercentage }} de petits fichiers"
          description: "Compaction recommandée pour améliorer les performances"
    
      # Alerte: snapshots accumulés
      - alert: IcebergSnapshotsAccumulated
        expr: iceberg_table_snapshot_count > 100
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "Table {{ $labels.table }} a {{ $value }} snapshots"
          description: "Expiration des snapshots recommandée"
    
      # Alerte: croissance anormale
      - alert: IcebergAbnormalGrowth
        expr: |
          (iceberg_table_size_bytes - iceberg_table_size_bytes offset 1d) 
          / iceberg_table_size_bytes offset 1d > 0.5
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "Croissance anormale de {{ $labels.table }}"
          description: "La table a grandi de plus de 50% en 24h"
```

**Dashboard Grafana** :

```json
{
  "dashboard": {
    "title": "Lakehouse Iceberg - Santé des Tables",
    "panels": [
      {
        "title": "Vue d'ensemble des tables",
        "type": "table",
        "targets": [
          {
            "expr": "iceberg_table_total_size_gb",
            "legendFormat": "{{ table }}"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "thresholds": {
              "mode": "absolute",
              "steps": [
                {"color": "green", "value": null},
                {"color": "yellow", "value": 100},
                {"color": "red", "value": 500}
              ]
            }
          }
        }
      },
      {
        "title": "Ratio de petits fichiers par table",
        "type": "gauge",
        "targets": [
          {
            "expr": "iceberg_small_files_ratio",
            "legendFormat": "{{ table }}"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "max": 1,
            "thresholds": {
              "mode": "percentage",
              "steps": [
                {"color": "green", "value": null},
                {"color": "yellow", "value": 20},
                {"color": "red", "value": 50}
              ]
            }
          }
        }
      },
      {
        "title": "Évolution du nombre de fichiers",
        "type": "timeseries",
        "targets": [
          {
            "expr": "iceberg_table_total_files",
            "legendFormat": "{{ table }}"
          }
        ]
      },
      {
        "title": "Opérations de maintenance",
        "type": "stat",
        "targets": [
          {
            "expr": "sum(rate(iceberg_compaction_completed_total[24h]))",
            "legendFormat": "Compactions/24h"
          },
          {
            "expr": "sum(rate(iceberg_snapshots_expired_total[24h]))",
            "legendFormat": "Expirations/24h"
          }
        ]
      }
    ]
  }
}
```

### Collecte Automatisée des Métriques

**Service de collecte des métriques** :

```python
from prometheus_client import Gauge, start_http_server
import schedule
import time

# Définition des métriques Prometheus
iceberg_total_files = Gauge(
    'iceberg_table_total_files', 
    'Nombre total de fichiers',
    ['catalog', 'database', 'table']
)

iceberg_total_size = Gauge(
    'iceberg_table_total_size_gb',
    'Taille totale en Go',
    ['catalog', 'database', 'table']
)

iceberg_small_files = Gauge(
    'iceberg_table_small_files_count',
    'Nombre de petits fichiers (<100Mo)',
    ['catalog', 'database', 'table']
)

iceberg_snapshot_count = Gauge(
    'iceberg_table_snapshot_count',
    'Nombre de snapshots',
    ['catalog', 'database', 'table']
)

iceberg_avg_file_size = Gauge(
    'iceberg_table_avg_file_size_mb',
    'Taille moyenne des fichiers en Mo',
    ['catalog', 'database', 'table']
)

class IcebergMetricsCollector:
    def __init__(self, spark, catalog: str, tables: list):
        self.spark = spark
        self.catalog = catalog
        self.tables = tables
  
    def collect_all_metrics(self):
        """Collecte les métriques pour toutes les tables configurées."""
        for table_path in self.tables:
            try:
                self._collect_table_metrics(table_path)
            except Exception as e:
                logger.error(f"Erreur collecte métriques {table_path}: {e}")
  
    def _collect_table_metrics(self, table_path: str):
        """Collecte les métriques pour une table."""
        parts = table_path.split('.')
        database = parts[-2] if len(parts) > 1 else 'default'
        table = parts[-1]
      
        metrics = collect_table_metrics(self.spark, table_path)
      
        labels = [self.catalog, database, table]
      
        iceberg_total_files.labels(*labels).set(metrics['total_files'])
        iceberg_total_size.labels(*labels).set(metrics['total_size_gb'])
        iceberg_small_files.labels(*labels).set(metrics['small_files_count'])
        iceberg_snapshot_count.labels(*labels).set(metrics['snapshot_count'])
        iceberg_avg_file_size.labels(*labels).set(metrics['avg_file_size_mb'])

# Démarrage du serveur Prometheus
start_http_server(8000)

# Configuration de la collecte périodique
collector = IcebergMetricsCollector(
    spark=spark,
    catalog='lakehouse',
    tables=[
        'lakehouse.gold.transactions',
        'lakehouse.gold.clients',
        'lakehouse.silver.events',
        'lakehouse.bronze.raw_data'
    ]
)

schedule.every(5).minutes.do(collector.collect_all_metrics)

while True:
    schedule.run_pending()
    time.sleep(1)
```

---

## Automatisation des Opérations

### Orchestration avec Apache Airflow

L'automatisation des opérations de maintenance via Airflow garantit leur exécution régulière et fiable.

**DAG de maintenance quotidienne** :

```python
from airflow import DAG
from airflow.operators.python import PythonOperator
from airflow.providers.apache.spark.operators.spark_submit import SparkSubmitOperator
from airflow.operators.email import EmailOperator
from airflow.utils.task_group import TaskGroup
from datetime import datetime, timedelta

default_args = {
    'owner': 'data-platform',
    'depends_on_past': False,
    'email': ['data-ops@entreprise.ca'],
    'email_on_failure': True,
    'retries': 2,
    'retry_delay': timedelta(minutes=15),
    'retry_exponential_backoff': True
}

# Liste des tables à maintenir
TABLES_CRITIQUES = [
    'lakehouse.gold.transactions',
    'lakehouse.gold.clients',
    'lakehouse.silver.events'
]

TABLES_STANDARD = [
    'lakehouse.bronze.raw_events',
    'lakehouse.bronze.raw_logs'
]

with DAG(
    'lakehouse_maintenance_quotidienne',
    default_args=default_args,
    description='Maintenance quotidienne du Lakehouse Iceberg',
    schedule_interval='0 3 * * *',  # 3h00 quotidien
    start_date=datetime(2024, 1, 1),
    catchup=False,
    tags=['maintenance', 'iceberg', 'production'],
    max_active_runs=1
) as dag:
  
    # Collecte des métriques pré-maintenance
    collect_metrics_pre = SparkSubmitOperator(
        task_id='collect_metrics_pre',
        application='s3://lakehouse/jobs/collect_metrics.py',
        conf={'spark.app.name': 'MetricsPreMaintenance'},
        application_args=['--output', 's3://lakehouse/metrics/pre/{{ ds }}']
    )
  
    # Groupe: Compaction des tables critiques
    with TaskGroup('compaction_critiques') as compaction_critiques:
        for table in TABLES_CRITIQUES:
            table_name = table.split('.')[-1]
            SparkSubmitOperator(
                task_id=f'compact_{table_name}',
                application='s3://lakehouse/jobs/compaction.py',
                conf={
                    'spark.app.name': f'Compaction-{table_name}',
                    'spark.executor.instances': '10',
                    'spark.executor.memory': '8g'
                },
                application_args=[
                    '--table', table,
                    '--strategy', 'binpack',
                    '--target-size', '512MB'
                ]
            )
  
    # Groupe: Compaction des tables standard
    with TaskGroup('compaction_standard') as compaction_standard:
        for table in TABLES_STANDARD:
            table_name = table.split('.')[-1]
            SparkSubmitOperator(
                task_id=f'compact_{table_name}',
                application='s3://lakehouse/jobs/compaction.py',
                conf={
                    'spark.app.name': f'Compaction-{table_name}',
                    'spark.executor.instances': '5'
                },
                application_args=[
                    '--table', table,
                    '--strategy', 'binpack',
                    '--target-size', '512MB',
                    '--partitions-last-days', '7'
                ]
            )
  
    # Expiration des snapshots
    with TaskGroup('expiration_snapshots') as expiration_snapshots:
        for table in TABLES_CRITIQUES + TABLES_STANDARD:
            table_name = table.split('.')[-1]
            SparkSubmitOperator(
                task_id=f'expire_{table_name}',
                application='s3://lakehouse/jobs/expire_snapshots.py',
                application_args=[
                    '--table', table,
                    '--retention-days', '7',
                    '--retain-last', '10'
                ]
            )
  
    # Nettoyage des orphelins (une fois par semaine le dimanche)
    cleanup_orphans = SparkSubmitOperator(
        task_id='cleanup_orphans',
        application='s3://lakehouse/jobs/cleanup_orphans.py',
        application_args=['--older-than-days', '14'],
        trigger_rule='all_done'  # Exécute même si compaction échoue
    )
  
    # Collecte des métriques post-maintenance
    collect_metrics_post = SparkSubmitOperator(
        task_id='collect_metrics_post',
        application='s3://lakehouse/jobs/collect_metrics.py',
        application_args=['--output', 's3://lakehouse/metrics/post/{{ ds }}']
    )
  
    # Génération du rapport
    generate_report = PythonOperator(
        task_id='generate_report',
        python_callable=generate_maintenance_report,
        op_kwargs={'date': '{{ ds }}'}
    )
  
    # Notification
    notify_success = EmailOperator(
        task_id='notify_success',
        to='data-ops@entreprise.ca',
        subject='[Lakehouse] Maintenance quotidienne terminée - {{ ds }}',
        html_content='{{ task_instance.xcom_pull(task_ids="generate_report") }}'
    )
  
    # Dépendances
    collect_metrics_pre >> [compaction_critiques, compaction_standard]
    compaction_critiques >> expiration_snapshots
    compaction_standard >> expiration_snapshots
    expiration_snapshots >> cleanup_orphans >> collect_metrics_post
    collect_metrics_post >> generate_report >> notify_success
```

**Script de compaction réutilisable** :

```python
# jobs/compaction.py
import argparse
from pyspark.sql import SparkSession
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def main():
    parser = argparse.ArgumentParser(description='Compaction Iceberg')
    parser.add_argument('--table', required=True, help='Table à compacter')
    parser.add_argument('--strategy', default='binpack', choices=['binpack', 'sort'])
    parser.add_argument('--target-size', default='512MB', help='Taille cible des fichiers')
    parser.add_argument('--partitions-last-days', type=int, help='Limiter aux N derniers jours')
    parser.add_argument('--sort-order', help='Ordre de tri (pour strategy=sort)')
    args = parser.parse_args()
  
    spark = SparkSession.builder \
        .appName(f"Compaction-{args.table}") \
        .getOrCreate()
  
    # Conversion de la taille cible
    size_bytes = parse_size(args.target_size)
  
    # Construction de la requête
    options = {
        'target-file-size-bytes': str(size_bytes),
        'min-input-files': '3',
        'partial-progress.enabled': 'true'
    }
  
    where_clause = ""
    if args.partitions_last_days:
        where_clause = f"WHERE date_partition >= current_date - INTERVAL '{args.partitions_last_days}' DAY"
  
    sort_order = ""
    if args.strategy == 'sort' and args.sort_order:
        sort_order = f", sort_order => '{args.sort_order}'"
  
    query = f"""
        CALL lakehouse.system.rewrite_data_files(
            table => '{args.table}',
            strategy => '{args.strategy}'
            {sort_order},
            options => map({', '.join(f"'{k}', '{v}'" for k, v in options.items())})
        )
    """
  
    logger.info(f"Exécution compaction: {args.table}")
    logger.info(f"Stratégie: {args.strategy}, Taille cible: {args.target_size}")
  
    result = spark.sql(query).collect()
  
    rewritten_files = sum(r.rewritten_data_files_count for r in result)
    added_files = sum(r.added_data_files_count for r in result)
  
    logger.info(f"Compaction terminée: {rewritten_files} fichiers réécrits → {added_files} fichiers")
  
    spark.stop()

def parse_size(size_str: str) -> int:
    """Convertit une chaîne de taille en bytes."""
    units = {'B': 1, 'KB': 1024, 'MB': 1024**2, 'GB': 1024**3}
    size_str = size_str.upper().strip()
    for unit, multiplier in units.items():
        if size_str.endswith(unit):
            return int(float(size_str[:-len(unit)]) * multiplier)
    return int(size_str)

if __name__ == '__main__':
    main()
```

### Maintenance Événementielle

Au-delà de la maintenance planifiée, certaines opérations doivent se déclencher sur événements.

**Triggers automatiques** :

```python
from airflow.sensors.external_task import ExternalTaskSensor
from airflow.operators.trigger_dagrun import TriggerDagRunOperator

# DAG de compaction déclenchée par accumulation
with DAG(
    'lakehouse_compaction_triggered',
    schedule_interval=None,  # Déclenché par événement
    catchup=False
) as dag:
  
    def check_compaction_needed(**context):
        """Vérifie si une compaction est nécessaire."""
        table = context['dag_run'].conf.get('table')
        metrics = collect_table_metrics(spark, table)
      
        # Critères de déclenchement
        if metrics['small_files_ratio'] > 0.3:
            return True
        if metrics['total_files'] > 10000:
            return True
      
        return False
  
    check_needed = PythonOperator(
        task_id='check_compaction_needed',
        python_callable=check_compaction_needed
    )
  
    run_compaction = SparkSubmitOperator(
        task_id='run_compaction',
        application='s3://lakehouse/jobs/compaction.py',
        application_args=[
            '--table', '{{ dag_run.conf.table }}',
            '--strategy', 'binpack'
        ]
    )
  
    check_needed >> run_compaction

# Fonction pour déclencher depuis un monitoring externe
def trigger_compaction_if_needed(table: str, metrics: dict):
    """Déclenche une compaction si les seuils sont dépassés."""
    from airflow.api.common.experimental.trigger_dag import trigger_dag
  
    if metrics['small_files_ratio'] > 0.3:
        trigger_dag(
            dag_id='lakehouse_compaction_triggered',
            conf={'table': table}
        )
```

---

## Gestion des Incidents

### Procédures de Diagnostic

**Runbook de diagnostic des problèmes de performance** :

```markdown
# Runbook: Diagnostic Performance Lakehouse

## Symptôme: Requêtes lentes

### Étape 1: Vérifier l'état des tables concernées
```sql
-- Statistiques des fichiers
SELECT 
    COUNT(*) as total_files,
    AVG(file_size_in_bytes)/(1024*1024) as avg_size_mb,
    SUM(CASE WHEN file_size_in_bytes < 104857600 THEN 1 ELSE 0 END) as small_files
FROM lakehouse.gold.transactions.files;
```

### Étape 2: Vérifier les statistiques de table

```sql
-- Dernière mise à jour des stats
SELECT * FROM lakehouse.gold.transactions.metadata_log_entries
ORDER BY timestamp DESC LIMIT 5;
```

### Étape 3: Analyser le plan de requête

```sql
EXPLAIN ANALYZE SELECT ... FROM lakehouse.gold.transactions WHERE ...;
```

### Actions correctives:

* Si small_files > 20% du total → Exécuter compaction
* Si statistiques obsolètes → ANALYZE TABLE
* Si plan de requête inefficace → Vérifier partitionnement

```

**Script de diagnostic automatisé** :

```python
class LakehouseDiagnostics:
    def __init__(self, spark):
        self.spark = spark
  
    def diagnose_table(self, table: str) -> dict:
        """Diagnostic complet d'une table."""
        issues = []
        recommendations = []
      
        # 1. Analyse des fichiers
        files_analysis = self._analyze_files(table)
        if files_analysis['small_files_ratio'] > 0.2:
            issues.append(f"Trop de petits fichiers: {files_analysis['small_files_ratio']:.1%}")
            recommendations.append("Exécuter compaction binpack")
      
        # 2. Analyse des snapshots
        snapshots_analysis = self._analyze_snapshots(table)
        if snapshots_analysis['count'] > 100:
            issues.append(f"Accumulation de snapshots: {snapshots_analysis['count']}")
            recommendations.append("Exécuter expiration des snapshots")
      
        # 3. Analyse des manifestes
        manifests_analysis = self._analyze_manifests(table)
        if manifests_analysis['count'] > 100:
            issues.append(f"Fragmentation des manifestes: {manifests_analysis['count']}")
            recommendations.append("Réécrire les manifestes")
      
        # 4. Vérification des statistiques
        stats_analysis = self._analyze_statistics(table)
        if stats_analysis['age_days'] > 7:
            issues.append(f"Statistiques obsolètes: {stats_analysis['age_days']} jours")
            recommendations.append("Mettre à jour les statistiques (ANALYZE TABLE)")
      
        return {
            'table': table,
            'health_score': self._calculate_health_score(files_analysis, snapshots_analysis, manifests_analysis),
            'issues': issues,
            'recommendations': recommendations,
            'details': {
                'files': files_analysis,
                'snapshots': snapshots_analysis,
                'manifests': manifests_analysis,
                'statistics': stats_analysis
            }
        }
  
    def _analyze_files(self, table: str) -> dict:
        result = self.spark.sql(f"""
            SELECT 
                COUNT(*) as total,
                SUM(file_size_in_bytes) as total_size,
                AVG(file_size_in_bytes) as avg_size,
                SUM(CASE WHEN file_size_in_bytes < 104857600 THEN 1 ELSE 0 END) as small_count
            FROM {table}.files
        """).collect()[0]
      
        return {
            'total_files': result.total,
            'total_size_gb': result.total_size / (1024**3) if result.total_size else 0,
            'avg_size_mb': result.avg_size / (1024**2) if result.avg_size else 0,
            'small_files_count': result.small_count,
            'small_files_ratio': result.small_count / result.total if result.total > 0 else 0
        }
  
    def _calculate_health_score(self, files, snapshots, manifests) -> int:
        """Calcule un score de santé de 0 à 100."""
        score = 100
      
        # Pénalités pour petits fichiers
        if files['small_files_ratio'] > 0.5:
            score -= 30
        elif files['small_files_ratio'] > 0.2:
            score -= 15
      
        # Pénalités pour snapshots
        if snapshots['count'] > 500:
            score -= 20
        elif snapshots['count'] > 100:
            score -= 10
      
        # Pénalités pour manifestes
        if manifests['count'] > 500:
            score -= 20
        elif manifests['count'] > 100:
            score -= 10
      
        return max(0, score)
```

### Récupération après Incident

**Procédure de rollback après écriture corrompue** :

```python
def rollback_to_snapshot(spark, table: str, snapshot_id: int = None, 
                         timestamp: str = None):
    """Rollback d'une table vers un état antérieur."""
  
    if snapshot_id:
        # Rollback vers snapshot spécifique
        spark.sql(f"""
            CALL lakehouse.system.rollback_to_snapshot(
                table => '{table}',
                snapshot_id => {snapshot_id}
            )
        """)
        logger.info(f"Rollback vers snapshot {snapshot_id} effectué")
      
    elif timestamp:
        # Rollback vers timestamp
        spark.sql(f"""
            CALL lakehouse.system.rollback_to_timestamp(
                table => '{table}',
                timestamp => TIMESTAMP '{timestamp}'
            )
        """)
        logger.info(f"Rollback vers {timestamp} effectué")
  
    else:
        # Rollback vers snapshot précédent
        snapshots = spark.sql(f"""
            SELECT snapshot_id, committed_at 
            FROM {table}.snapshots 
            ORDER BY committed_at DESC 
            LIMIT 2
        """).collect()
      
        if len(snapshots) < 2:
            raise ValueError("Pas assez de snapshots pour rollback")
      
        previous_snapshot = snapshots[1].snapshot_id
        spark.sql(f"""
            CALL lakehouse.system.rollback_to_snapshot(
                table => '{table}',
                snapshot_id => {previous_snapshot}
            )
        """)
        logger.info(f"Rollback vers snapshot précédent {previous_snapshot}")
```

**Récupération de métadonnées corrompues** :

```python
def recover_from_metadata_corruption(spark, table: str, 
                                     metadata_backup_path: str):
    """Récupération depuis une sauvegarde de métadonnées."""
  
    # 1. Identifier le dernier fichier metadata valide
    valid_metadata = find_valid_metadata(metadata_backup_path)
  
    if not valid_metadata:
        raise ValueError("Aucun fichier metadata valide trouvé")
  
    # 2. Réenregistrer la table avec le metadata valide
    catalog = table.split('.')[0]
    db_table = '.'.join(table.split('.')[1:])
  
    # Supprimer l'entrée catalogue corrompue
    spark.sql(f"DROP TABLE IF EXISTS {table}")
  
    # Réenregistrer avec metadata valide
    spark.sql(f"""
        CALL {catalog}.system.register_table(
            table => '{db_table}',
            metadata_file => '{valid_metadata}'
        )
    """)
  
    logger.info(f"Table {table} récupérée depuis {valid_metadata}")

def find_valid_metadata(backup_path: str) -> str:
    """Trouve le fichier metadata le plus récent valide."""
    import json
    from smart_open import open
  
    # Lister les fichiers metadata dans le backup
    metadata_files = list_files(backup_path, pattern='*.metadata.json')
  
    # Trier par date décroissante
    metadata_files.sort(reverse=True)
  
    for metadata_file in metadata_files:
        try:
            # Vérifier que le fichier est valide
            with open(metadata_file, 'r') as f:
                metadata = json.load(f)
              
            # Vérifications de base
            if 'format-version' in metadata and 'snapshots' in metadata:
                return metadata_file
              
        except Exception as e:
            logger.warning(f"Fichier metadata invalide: {metadata_file} - {e}")
            continue
  
    return None
```

---

## Sauvegarde et Récupération

### Stratégies de Sauvegarde

**Sauvegarde des métadonnées** :

Les fichiers de métadonnées Iceberg sont critiques — leur perte signifie la perte d'accès aux données. Une stratégie de sauvegarde robuste est essentielle.

```python
import boto3
from datetime import datetime

class IcebergBackupManager:
    def __init__(self, source_bucket: str, backup_bucket: str):
        self.s3 = boto3.client('s3')
        self.source_bucket = source_bucket
        self.backup_bucket = backup_bucket
  
    def backup_table_metadata(self, table_path: str):
        """Sauvegarde les métadonnées d'une table."""
      
        metadata_prefix = f"{table_path}/metadata/"
        backup_date = datetime.now().strftime('%Y-%m-%d_%H%M%S')
        backup_prefix = f"backups/{backup_date}/{table_path}/metadata/"
      
        # Lister tous les fichiers metadata
        paginator = self.s3.get_paginator('list_objects_v2')
      
        copied_files = 0
        for page in paginator.paginate(Bucket=self.source_bucket, 
                                        Prefix=metadata_prefix):
            for obj in page.get('Contents', []):
                source_key = obj['Key']
                dest_key = source_key.replace(metadata_prefix, backup_prefix)
              
                # Copie vers bucket de backup
                self.s3.copy_object(
                    Bucket=self.backup_bucket,
                    Key=dest_key,
                    CopySource={'Bucket': self.source_bucket, 'Key': source_key}
                )
                copied_files += 1
      
        logger.info(f"Sauvegarde {table_path}: {copied_files} fichiers")
        return backup_prefix
  
    def backup_all_tables(self, tables: list):
        """Sauvegarde toutes les tables configurées."""
        backup_manifest = {
            'timestamp': datetime.now().isoformat(),
            'tables': {}
        }
      
        for table in tables:
            table_path = table.replace('.', '/')
            backup_path = self.backup_table_metadata(table_path)
            backup_manifest['tables'][table] = backup_path
      
        # Sauvegarder le manifeste
        manifest_key = f"backups/{datetime.now().strftime('%Y-%m-%d_%H%M%S')}/manifest.json"
        self.s3.put_object(
            Bucket=self.backup_bucket,
            Key=manifest_key,
            Body=json.dumps(backup_manifest)
        )
      
        return backup_manifest
```

**Configuration de la réplication S3** :

```hcl
# terraform/s3-replication.tf

resource "aws_s3_bucket_replication_configuration" "lakehouse_replication" {
  bucket = aws_s3_bucket.lakehouse.id
  role   = aws_iam_role.replication.arn

  rule {
    id     = "metadata-replication"
    status = "Enabled"

    filter {
      prefix = "warehouse/"
    }

    destination {
      bucket        = aws_s3_bucket.lakehouse_backup.arn
      storage_class = "STANDARD_IA"
    
      replication_time {
        status = "Enabled"
        time {
          minutes = 15
        }
      }
    
      metrics {
        status = "Enabled"
        event_threshold {
          minutes = 15
        }
      }
    }

    delete_marker_replication {
      status = "Disabled"  # Conserver les données supprimées dans le backup
    }
  }
}
```

### Plan de Reprise après Sinistre

**RTO/RPO par criticité** :

| Niveau   | Tables                | RPO | RTO | Stratégie               |
| -------- | --------------------- | --- | --- | ------------------------ |
| Critique | Transactions, Clients | 1h  | 4h  | Réplication temps réel |
| Élevé  | Événements, Logs    | 4h  | 8h  | Backup horaire           |
| Standard | Référentiels        | 24h | 24h | Backup quotidien         |
| Faible   | Archives              | 7j  | 48h | Backup hebdomadaire      |

**Procédure de reprise** :

```python
class DisasterRecovery:
    def __init__(self, primary_region: str, dr_region: str):
        self.primary_region = primary_region
        self.dr_region = dr_region
        self.s3_primary = boto3.client('s3', region_name=primary_region)
        self.s3_dr = boto3.client('s3', region_name=dr_region)
  
    def initiate_failover(self, tables: list):
        """Initie le basculement vers la région DR."""
      
        failover_report = {
            'initiated_at': datetime.now().isoformat(),
            'tables': []
        }
      
        for table in tables:
            try:
                # 1. Vérifier la disponibilité dans DR
                dr_status = self._check_dr_availability(table)
              
                # 2. Mettre à jour le catalogue pour pointer vers DR
                self._update_catalog_to_dr(table)
              
                # 3. Valider l'accès aux données
                validation = self._validate_table_access(table)
              
                failover_report['tables'].append({
                    'table': table,
                    'status': 'success',
                    'dr_snapshot': dr_status['latest_snapshot'],
                    'data_loss_minutes': dr_status['lag_minutes']
                })
              
            except Exception as e:
                failover_report['tables'].append({
                    'table': table,
                    'status': 'failed',
                    'error': str(e)
                })
      
        return failover_report
  
    def _check_dr_availability(self, table: str) -> dict:
        """Vérifie la disponibilité et la fraîcheur des données DR."""
        # Implémentation spécifique selon l'architecture
        pass
  
    def _update_catalog_to_dr(self, table: str):
        """Met à jour le catalogue pour pointer vers la région DR."""
        # Mise à jour des références dans le catalogue REST
        pass
```

---

## Optimisation Continue

### Analyse des Patterns de Requêtes

L'optimisation continue repose sur l'analyse des patterns d'utilisation réels pour guider les décisions de maintenance.

```python
class QueryPatternAnalyzer:
    def __init__(self, spark):
        self.spark = spark
  
    def analyze_query_patterns(self, table: str, days: int = 30) -> dict:
        """Analyse les patterns de requêtes sur une table."""
      
        # Récupération des logs de requêtes (depuis Trino/système d'audit)
        query_logs = self.spark.sql(f"""
            SELECT 
                query_text,
                execution_time_ms,
                rows_read,
                bytes_read,
                query_timestamp
            FROM audit.query_logs
            WHERE table_name = '{table}'
              AND query_timestamp >= current_date - INTERVAL '{days}' DAY
        """)
      
        # Analyse des colonnes filtrées
        filter_columns = self._extract_filter_columns(query_logs)
      
        # Analyse des colonnes projetées
        projected_columns = self._extract_projected_columns(query_logs)
      
        # Analyse des patterns temporels
        temporal_patterns = self._analyze_temporal_patterns(query_logs)
      
        return {
            'table': table,
            'total_queries': query_logs.count(),
            'most_filtered_columns': filter_columns[:5],
            'most_projected_columns': projected_columns[:10],
            'peak_hours': temporal_patterns['peak_hours'],
            'recommendations': self._generate_recommendations(
                filter_columns, projected_columns
            )
        }
  
    def _generate_recommendations(self, filter_cols, project_cols) -> list:
        """Génère des recommandations d'optimisation."""
        recommendations = []
      
        # Recommandation de tri basée sur les filtres fréquents
        if filter_cols:
            top_filter = filter_cols[0]
            recommendations.append({
                'type': 'sort_order',
                'description': f"Considérer un tri par {top_filter['column']}",
                'impact': 'Amélioration potentielle des scans filtrés',
                'effort': 'moyen'
            })
      
        # Recommandation de Z-order si plusieurs colonnes filtrées
        if len(filter_cols) >= 2:
            cols = [c['column'] for c in filter_cols[:3]]
            recommendations.append({
                'type': 'z_order',
                'description': f"Considérer Z-order sur {', '.join(cols)}",
                'impact': 'Amélioration des requêtes multi-colonnes',
                'effort': 'élevé'
            })
      
        return recommendations
```

### Ajustement Automatique des Paramètres

```python
class AdaptiveMaintenanceScheduler:
    """Ajuste automatiquement les paramètres de maintenance selon les métriques."""
  
    def __init__(self, spark):
        self.spark = spark
        self.default_config = {
            'compaction_threshold': 0.2,  # Ratio petits fichiers
            'snapshot_retention_days': 7,
            'compaction_frequency_hours': 24
        }
  
    def calculate_optimal_config(self, table: str) -> dict:
        """Calcule la configuration optimale basée sur les métriques."""
      
        metrics = collect_table_metrics(self.spark, table)
        query_patterns = QueryPatternAnalyzer(self.spark).analyze_query_patterns(table)
      
        config = self.default_config.copy()
      
        # Ajustement basé sur le taux d'écriture
        write_rate = self._estimate_write_rate(table)
        if write_rate > 1000:  # Plus de 1000 écritures/heure
            config['compaction_frequency_hours'] = 1
            config['compaction_threshold'] = 0.1
        elif write_rate > 100:
            config['compaction_frequency_hours'] = 6
            config['compaction_threshold'] = 0.15
      
        # Ajustement basé sur la taille de la table
        if metrics['total_size_gb'] > 1000:  # Plus de 1 To
            config['snapshot_retention_days'] = 3  # Rétention réduite
      
        # Ajustement basé sur la fréquence des requêtes
        if query_patterns['total_queries'] > 10000:  # Table très consultée
            config['compaction_threshold'] = 0.1  # Compaction plus agressive
      
        return config
  
    def _estimate_write_rate(self, table: str) -> float:
        """Estime le taux d'écriture basé sur l'historique des snapshots."""
        result = self.spark.sql(f"""
            SELECT 
                COUNT(*) as snapshot_count,
                TIMESTAMPDIFF(HOUR, MIN(committed_at), MAX(committed_at)) as hours_span
            FROM {table}.snapshots
            WHERE committed_at >= current_timestamp - INTERVAL '24' HOUR
        """).collect()[0]
      
        if result.hours_span and result.hours_span > 0:
            return result.snapshot_count / result.hours_span
        return 0
```

---

## Études de Cas Canadiennes

### Secteur Financier : Banque d'Investissement

> **Étude de cas : Banque d'investissement canadienne**
>
> *Secteur* : Services financiers - marchés des capitaux
>
> *Défi* : Maintenir un Lakehouse de 500 To contenant 10 ans d'historique de transactions de marché avec des exigences de rétention réglementaire strictes (7 ans minimum) et des SLA de performance pour les requêtes analytiques (< 30 secondes).
>
> *Solution* : Architecture de maintenance à trois niveaux avec orchestration Airflow, monitoring Prometheus/Grafana et procédures de récupération automatisées. Stratégie de tiering automatique déplaçant les données > 1 an vers S3 Glacier.
>
> *Configuration de maintenance* :
>
> ```
> Tables critiques (données < 90 jours):
> - Compaction: toutes les 2 heures
> - Taille cible: 256 Mo
> - Expiration snapshots: 3 jours, retain_last=50
>
> Tables historiques (données 90 jours - 1 an):
> - Compaction: quotidienne
> - Taille cible: 512 Mo  
> - Expiration snapshots: 7 jours, retain_last=10
>
> Tables archives (> 1 an):
> - Compaction: mensuelle
> - Taille cible: 1 Go
> - Expiration snapshots: 30 jours, retain_last=5
> ```
>
> *Résultats* :
>
> * Réduction des coûts de stockage de 45% via tiering
> * 99.9% de disponibilité sur 2 ans
> * Temps de requête P95 < 25 secondes
> * Conformité réglementaire maintenue avec audit complet

### Secteur Télécommunications : Opérateur Mobile

> **Étude de cas : Opérateur mobile pancanadien**
>
> *Secteur* : Télécommunications
>
> *Défi* : Gérer l'ingestion de 50 milliards d'événements réseau par jour avec compaction temps réel pour maintenir des performances de requête acceptables pour le NOC (Network Operations Center).
>
> *Solution* : Architecture de maintenance continue avec compaction streaming intégrée au pipeline Flink, complétée par une compaction batch quotidienne pour optimisation finale.
>
> *Architecture de maintenance* :
>
> ```
> ┌─────────────────────────────────────────────────────────────────┐
> │  COMPACTION TEMPS RÉEL (Flink)                                 │
> │  - Micro-compaction toutes les 5 minutes                       │
> │  - Cible: 64 Mo (compromis fraîcheur/performance)              │
> └─────────────────────────────────────────────────────────────────┘
>                                 │
>                                 ▼
> ┌─────────────────────────────────────────────────────────────────┐
> │  COMPACTION BATCH (Spark - 4h00)                               │
> │  - Consolidation en fichiers 512 Mo                            │
> │  - Tri par timestamp pour requêtes temporelles                 │
> └─────────────────────────────────────────────────────────────────┘
>                                 │
>                                 ▼
> ┌─────────────────────────────────────────────────────────────────┐
> │  MAINTENANCE HEBDOMADAIRE                                       │
> │  - Z-Order sur (cell_id, timestamp)                            │
> │  - Nettoyage orphelins                                         │
> │  - Réécriture manifestes                                        │
> └─────────────────────────────────────────────────────────────────┘
> ```
>
> *Résultats* :
>
> * Latence de requête NOC < 5 secondes malgré 50B événements/jour
> * Ratio petits fichiers maintenu < 5%
> * Coût de maintenance: 3% du coût total d'infrastructure
> * Zéro incident de performance en production

### Secteur Commerce : Détaillant Alimentaire

> **Étude de cas : Chaîne d'épiceries québécoise**
>
> *Secteur* : Commerce de détail alimentaire
>
> *Défi* : Maintenir un Lakehouse de 50 To avec des ressources d'équipe limitées (2 data engineers) tout en supportant 200 utilisateurs analytiques.
>
> *Solution* : Automatisation complète de la maintenance via Airflow avec alertes proactives. Utilisation de Dremio pour la couche de requête avec Reflections réduisant la dépendance à une compaction ultra-optimisée.
>
> *Stratégie low-touch* :
>
> ```python
> # Configuration simplifiée pour équipe réduite
> maintenance_config = {
>     'compaction': {
>         'schedule': 'daily_3am',
>         'strategy': 'binpack',  # Simple, pas de tri complexe
>         'target_size': '512MB',
>         'auto_trigger_threshold': 0.3  # 30% petits fichiers
>     },
>     'expiration': {
>         'schedule': 'daily_4am',
>         'retention_days': 14,
>         'retain_last': 20
>     },
>     'orphan_cleanup': {
>         'schedule': 'weekly_sunday_5am',
>         'safety_margin_days': 21
>     },
>     'alerting': {
>         'small_files_warning': 0.25,
>         'small_files_critical': 0.5,
>         'notify': 'slack://data-alerts'
>     }
> }
> ```
>
> *Résultats* :
>
> * Temps de maintenance: 2h/semaine (vs 15h/semaine avant automatisation)
> * Aucune intervention manuelle requise en 6 mois
> * Performance stable malgré croissance de 30% du volume
> * ROI de l'automatisation: 200% en première année

### Secteur Public : Agence Statistique

> **Étude de cas : Agence fédérale de statistiques**
>
> *Secteur* : Gouvernement fédéral
>
> *Défi* : Maintenir un Lakehouse contenant des données de recensement avec exigences de conservation perpétuelle et capacité de reproduire exactement les analyses publiées des décennies plus tard.
>
> *Solution* : Stratégie de versionnement strict avec tags Iceberg pour chaque publication officielle, jamais d'expiration de snapshots tagués, et archivage géographiquement distribué.
>
> *Politique de rétention* :
>
> ```sql
> -- Aucune expiration automatique pour tables de publication
> ALTER TABLE lakehouse.recensement.donnees_2021 
> SET TBLPROPERTIES ('gc.enabled' = 'false');
>
> -- Tags pour chaque publication officielle
> ALTER TABLE lakehouse.recensement.donnees_2021
> CREATE TAG `publication_officielle_2022_02_09`
> AS OF VERSION 1234567890;
>
> -- Les tags sont JAMAIS supprimés
> -- La compaction préserve les fichiers référencés par les tags
> ```
>
> *Résultats* :
>
> * Reproductibilité parfaite des analyses sur 20+ ans
> * Conformité aux exigences archivistiques fédérales
> * Capacité Time Travel illimitée pour données officielles
> * Coût de stockage maîtrisé via tiering Glacier pour données non-taggées

---

## Bonnes Pratiques et Recommandations

### Checklist de Maintenance

**Quotidien** :

* [ ] Vérification des alertes de monitoring
* [ ] Exécution de la compaction des tables critiques
* [ ] Expiration des snapshots selon politique
* [ ] Vérification des jobs de maintenance planifiés

**Hebdomadaire** :

* [ ] Nettoyage des fichiers orphelins
* [ ] Revue des métriques de santé des tables
* [ ] Analyse des tendances de croissance
* [ ] Vérification des sauvegardes

**Mensuel** :

* [ ] Réécriture des manifestes si nécessaire
* [ ] Analyse des patterns de requêtes
* [ ] Ajustement des paramètres de maintenance
* [ ] Test de récupération après sinistre
* [ ] Revue de la politique de rétention

**Trimestriel** :

* [ ] Audit complet de la santé du Lakehouse
* [ ] Optimisation du tri (Z-order) si applicable
* [ ] Revue de l'architecture de maintenance
* [ ] Mise à jour de la documentation

### Matrice de Décision

**Fréquence de compaction** :

| Taux d'écriture   | Volume table   | Fréquence recommandée |
| ------------------ | -------------- | ----------------------- |
| Streaming continu  | > 100 Go/jour  | Horaire                 |
| Batch fréquent    | 10-100 Go/jour | Toutes les 6h           |
| Batch quotidien    | 1-10 Go/jour   | Quotidienne             |
| Batch hebdomadaire | < 1 Go/jour    | Hebdomadaire            |

**Rétention des snapshots** :

| Type de table    | Fréquence modification | Rétention recommandée  |
| ---------------- | ----------------------- | ------------------------ |
| Temps réel      | Continue                | 3-7 jours                |
| Transactionnelle | Quotidienne             | 7-14 jours               |
| Analytique       | Hebdomadaire            | 30-90 jours              |
| Réglementée    | Variable                | Selon exigences légales |

### Anti-Patterns à Éviter

| Anti-pattern               | Problème                 | Solution                  |
| -------------------------- | ------------------------- | ------------------------- |
| Pas de compaction          | Dégradation performance  | Automatiser la compaction |
| Compaction trop fréquente | Gaspillage de ressources  | Seuils adaptés au volume |
| Expiration agressive       | Perte de Time Travel      | Rétention suffisante     |
| Pas de monitoring          | Problèmes non détectés | Alertes proactives        |
| Maintenance manuelle       | Oublis, incohérences     | Automatisation complète  |
| Même config pour tout     | Sous-optimisation         | Config par type de table  |

---

## Conclusion

La maintenance d'un Lakehouse Apache Iceberg en production constitue une discipline à part entière, distincte de la conception et du développement initial. Les tables Iceberg, par leur nature immutable et versionnée, accumulent naturellement fichiers et métadonnées qui, sans gestion active, dégradent progressivement les performances et gonflent les coûts. La différence entre un Lakehouse performant et un système problématique réside souvent dans la qualité des pratiques opérationnelles plutôt que dans l'architecture initiale.

Les opérations fondamentales — compaction, expiration des snapshots, nettoyage des orphelins — doivent être orchestrées de manière automatisée et adaptée au profil de chaque table. Une table de streaming haute fréquence nécessite une compaction agressive toutes les heures, tandis qu'une table d'archive peut se contenter d'une optimisation mensuelle. Cette différenciation, guidée par les métriques et les patterns d'utilisation réels, optimise l'équilibre entre performance et coût de maintenance.

Le monitoring proactif transforme la maintenance de réactive à préventive. Des métriques bien choisies — ratio de petits fichiers, nombre de snapshots, fragmentation des manifestes — permettent d'anticiper les problèmes avant qu'ils n'impactent les utilisateurs. Les alertes automatisées et les dashboards de santé offrent une visibilité continue sur l'état du Lakehouse.

Les études de cas canadiennes démontrent que des organisations de tailles et secteurs variés ont réussi à établir des pratiques de maintenance robustes. Que ce soit une banque d'investissement avec 500 To de données réglementées, un opérateur télécoms gérant 50 milliards d'événements quotidiens, ou une chaîne d'épiceries avec une équipe réduite, des stratégies adaptées permettent de maintenir performance et fiabilité dans la durée.

Le chapitre suivant aborde l'opérationnalisation complète du Lakehouse, où nous examinerons les aspects organisationnels, les modèles d'exploitation et les pratiques DevOps/DataOps qui complètent les opérations techniques de maintenance.

---

## Résumé

**Opérations essentielles** :

* **Compaction** : Consolide les petits fichiers, améliore les performances de lecture
* **Expiration snapshots** : Libère l'espace, contrôle la croissance des métadonnées
* **Nettoyage orphelins** : Supprime les fichiers déréférencés
* **Réécriture manifestes** : Optimise la structure des métadonnées

**Paramètres clés de compaction** :

* Taille cible : 256-512 Mo pour tables actives, 512 Mo-1 Go pour archives
* Seuil de déclenchement : 20-30% de petits fichiers
* Stratégie : binpack pour simple consolidation, sort/Z-order pour optimisation requêtes

**Monitoring recommandé** :

* Ratio petits fichiers (seuil: < 20%)
* Nombre de snapshots (seuil: < 100)
* Taille moyenne des fichiers (seuil: > 100 Mo)
* Nombre de manifestes (seuil: < 100)

**Automatisation** :

* Orchestration Airflow pour planification fiable
* Triggers événementiels pour réaction aux seuils
* Scripts réutilisables et paramétrables
* Alertes proactives vers Slack/courriel

**Récupération** :

* Rollback via snapshots pour erreurs d'écriture
* Sauvegarde régulière des métadonnées
* Réplication cross-région pour DR
* Procédures testées et documentées

**Fréquences recommandées** :

* Compaction : horaire à quotidienne selon volume
* Expiration : quotidienne
* Orphelins : hebdomadaire
* Manifestes : mensuelle

---

*Ce chapitre établit les pratiques opérationnelles de votre Lakehouse. Le chapitre suivant, « Opérationnalisation et DevOps », explore les aspects organisationnels et les pratiques d'équipe pour une exploitation mature de la plateforme.*
