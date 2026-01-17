# Chapitre IV.3 — Mise en Pratique avec Apache Iceberg

---

Les deux chapitres précédents ont établi les fondations conceptuelles du Data Lakehouse et l'anatomie technique d'Apache Iceberg. Nous avons exploré la hiérarchie des métadonnées, les stratégies d'écriture Copy-on-Write et Merge-on-Read, l'évolution de schéma et le partitionnement masqué. Ces connaissances théoriques constituent un socle essentiel, mais la véritable maîtrise d'une technologie ne s'acquiert qu'à travers la pratique.

Ce chapitre marque le passage de la théorie à l'implémentation concrète. Nous allons construire, étape par étape, un environnement Lakehouse fonctionnel sur votre machine locale, créer des tables Iceberg avec Apache Spark, les interroger via Dremio, puis connecter l'ensemble à un outil d'intelligence d'affaires (BI) pour produire des visualisations exploitables. Cette progression illustre le parcours complet des données dans une architecture Lakehouse moderne : de l'ingestion brute jusqu'à la consommation analytique.

L'objectif n'est pas de présenter une configuration de production — les chapitres de la Partie 2 couvriront ces aspects en profondeur — mais de démystifier Apache Iceberg à travers une expérience pratique immersive. En suivant les exercices de ce chapitre, vous développerez une intuition concrète du fonctionnement d'Iceberg, de ses interactions avec les différents moteurs de calcul et de sa valeur ajoutée pour les analystes et les scientifiques de données.

L'approche retenue privilégie la conteneurisation avec Docker, permettant un déploiement reproductible et isolé de l'environnement de développement. Cette méthode présente plusieurs avantages significatifs pour l'apprentissage : elle élimine les problèmes de compatibilité entre systèmes d'exploitation, garantit que chaque lecteur dispose d'une configuration identique, et facilite le nettoyage complet de l'environnement après les expérimentations. De plus, les patrons architecturaux établis dans cet environnement local se transposent directement vers des déploiements infonuagiques de production.

Le parcours pratique de ce chapitre s'articule autour d'un scénario réaliste : la construction d'une plateforme analytique pour une entreprise de commerce électronique canadienne. Ce fil conducteur nous permettra d'explorer la création de tables dimensionnelles (clients) et factuelles (transactions), les jointures entre tables, les agrégations analytiques et la visualisation des indicateurs clés de performance. Les données d'exemple utilisent des noms et des provinces canadiennes, rendant les résultats plus tangibles pour le public cible de cette monographie.

---

## IV.3.1 Configuration d'un environnement Apache Iceberg

La mise en place d'un environnement Lakehouse complet nécessitait traditionnellement des semaines de configuration et une infrastructure coûteuse. Grâce à la conteneurisation et aux images Docker préconfigurées, nous pouvons désormais déployer un écosystème complet en quelques minutes sur un ordinateur portable standard. Cette démocratisation de l'accès aux technologies Lakehouse représente une avancée majeure pour l'apprentissage et le prototypage rapide.

L'écosystème que nous allons déployer reproduit fidèlement les composants d'une architecture Lakehouse de production, mais à une échelle adaptée aux ressources d'une machine de développement. Chaque service remplit une fonction spécifique et communique avec les autres via des interfaces standardisées, illustrant le principe de découplage qui caractérise les architectures modernes de données.

### Prérequis techniques

Avant de commencer, assurez-vous de disposer des éléments suivants sur votre machine de développement :

| Composant | Version minimale | Recommandation |
|-----------|------------------|----------------|
| Docker Desktop | 4.25+ | Dernière version stable |
| Docker Compose | 2.20+ | Inclus dans Docker Desktop |
| Mémoire RAM | 8 Go | 16 Go recommandés |
| Espace disque | 10 Go libres | 20 Go pour expérimentation |
| Ports disponibles | 8080, 8888, 9000, 9001, 9047, 19120 | Vérifier avec `netstat` ou `lsof` |

Pour les utilisateurs de systèmes GNU/Linux ou macOS, la vérification de la disponibilité des ports s'effectue avec la commande suivante :

```bash
# Vérifier si un port est disponible
lsof -i :9000 || echo "Port 9000 disponible"
```

### Architecture de l'environnement de développement

Notre environnement de développement reproduit les couches fondamentales d'un Lakehouse en production, mais à une échelle adaptée à l'apprentissage. Chaque composant joue un rôle spécifique dans l'architecture globale :

**MinIO** sert de couche de stockage compatible S3. Cette solution de stockage objet à code source ouvert émule parfaitement l'interface programmatique (API) d'Amazon S3, permettant de développer localement des applications qui fonctionneront ensuite sur n'importe quel stockage infonuagique compatible S3. MinIO offre une performance remarquable et une simplicité de configuration idéale pour le développement.

**Project Nessie** assume le rôle de catalogue Iceberg. Ce catalogue REST respecte la spécification Apache Iceberg REST Catalog et ajoute des capacités de versionnement de type Git pour les données. Nessie permet de créer des branches, de fusionner des modifications et de maintenir un historique complet des changements sur les tables Iceberg.

**Apache Spark** constitue le moteur de traitement principal. Spark offre le support le plus complet pour Apache Iceberg, incluant toutes les opérations DDL (Data Definition Language) et DML (Data Manipulation Language). La version 3.5 apporte des améliorations significatives pour les opérations MERGE et la gestion des métadonnées.

**Dremio** représente la couche de fédération et d'accélération. Cette plateforme Lakehouse unifie l'accès aux données provenant de sources multiples et optimise les performances des requêtes grâce à ses réflexions de données (data reflections) et son moteur de requêtes basé sur Apache Arrow.

**Apache Superset** (optionnel) fournit les capacités de visualisation. Cet outil d'intelligence d'affaires à code source ouvert permet de créer des tableaux de bord interactifs directement connectés à Dremio ou aux tables Iceberg.

### Fichier Docker Compose

Créez un fichier nommé `docker-compose.yml` dans un répertoire dédié à votre projet Lakehouse :

```yaml
version: '3.8'

services:
  # Stockage objet compatible S3
  minio:
    image: minio/minio:latest
    container_name: minio
    environment:
      MINIO_ROOT_USER: admin
      MINIO_ROOT_PASSWORD: password123
      MINIO_REGION: ca-central-1
    ports:
      - "9000:9000"
      - "9001:9001"
    command: server /data --console-address ":9001"
    volumes:
      - minio-data:/data
    networks:
      - lakehouse-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3

  # Catalogue Iceberg REST avec versionnement
  nessie:
    image: projectnessie/nessie:latest
    container_name: nessie
    environment:
      QUARKUS_PROFILE: prod
      QUARKUS_HTTP_PORT: 19120
      NESSIE_VERSION_STORE_TYPE: IN_MEMORY
    ports:
      - "19120:19120"
    networks:
      - lakehouse-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:19120/api/v2/config"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Moteur de traitement Spark avec Iceberg
  spark-iceberg:
    image: tabulario/spark-iceberg:3.5.1_1.5.2
    container_name: spark-iceberg
    depends_on:
      - minio
      - nessie
    environment:
      AWS_ACCESS_KEY_ID: admin
      AWS_SECRET_ACCESS_KEY: password123
      AWS_REGION: ca-central-1
      SPARK_HOME: /opt/spark
    ports:
      - "8888:8888"
      - "8080:8080"
      - "10000:10000"
      - "10001:10001"
    volumes:
      - spark-warehouse:/home/iceberg/warehouse
      - ./notebooks:/home/iceberg/notebooks
    networks:
      - lakehouse-network
    command: notebook

  # Plateforme Lakehouse Dremio
  dremio:
    image: dremio/dremio-oss:latest
    container_name: dremio
    depends_on:
      - minio
      - nessie
    ports:
      - "9047:9047"
      - "31010:31010"
      - "32010:32010"
      - "45678:45678"
    volumes:
      - dremio-data:/opt/dremio/data
    networks:
      - lakehouse-network

volumes:
  minio-data:
  spark-warehouse:
  dremio-data:

networks:
  lakehouse-network:
    driver: bridge
```

### Initialisation de l'environnement

Lancez l'ensemble des services avec la commande suivante :

```bash
# Créer le répertoire pour les notebooks
mkdir -p notebooks

# Démarrer l'environnement
docker-compose up -d

# Vérifier le statut des conteneurs
docker-compose ps
```

L'initialisation complète prend généralement entre deux et cinq minutes selon la performance de votre machine et la disponibilité des images dans le cache Docker local.

### Configuration du stockage MinIO

Avant de créer des tables Iceberg, nous devons préparer l'infrastructure de stockage dans MinIO. Accédez à la console MinIO via votre navigateur à l'adresse `http://localhost:9001` et connectez-vous avec les identifiants définis dans le fichier Docker Compose (`admin` / `password123`).

Créez un compartiment (bucket) nommé `warehouse` qui servira d'entrepôt principal pour les données et métadonnées Iceberg. Ce compartiment contiendra la structure hiérarchique suivante :

```
warehouse/
├── db_production/
│   ├── clients/
│   │   ├── data/
│   │   └── metadata/
│   └── transactions/
│       ├── data/
│       └── metadata/
└── db_staging/
    └── ...
```

Vous pouvez également créer le compartiment programmatiquement via l'interface en ligne de commande MinIO :

```bash
# Installer le client MinIO (mc)
docker exec -it minio mc alias set local http://localhost:9000 admin password123

# Créer le compartiment principal
docker exec -it minio mc mb local/warehouse

# Vérifier la création
docker exec -it minio mc ls local/
```

### Validation de la connectivité

Avant de poursuivre, validez que tous les composants sont opérationnels et peuvent communiquer entre eux :

| Service | URL de vérification | État attendu |
|---------|---------------------|--------------|
| MinIO Console | http://localhost:9001 | Interface de connexion |
| MinIO API | http://localhost:9000/minio/health/live | `OK` |
| Nessie | http://localhost:19120/api/v2/config | Configuration JSON |
| Spark UI | http://localhost:8080 | Interface Spark |
| Jupyter | http://localhost:8888 | Interface de notebooks |
| Dremio | http://localhost:9047 | Assistant de configuration |

Un script de validation automatisé peut simplifier cette vérification :

```bash
#!/bin/bash
# validation_lakehouse.sh

echo "=== Validation de l'environnement Lakehouse ==="

services=("minio:9000" "nessie:19120" "spark-iceberg:8888" "dremio:9047")
for service in "${services[@]}"; do
    IFS=':' read -r name port <<< "$service"
    if curl -s --connect-timeout 5 "http://localhost:$port" > /dev/null 2>&1; then
        echo "✓ $name (port $port) : Opérationnel"
    else
        echo "✗ $name (port $port) : Non accessible"
    fi
done

echo "=== Validation terminée ==="
```

### Résolution des problèmes courants

L'expérience montre que certains problèmes surviennent fréquemment lors de la configuration initiale. Voici les solutions aux difficultés les plus courantes :

**Problème : Conteneurs qui redémarrent en boucle**

Ce comportement indique généralement une insuffisance de ressources ou un conflit de ports. Vérifiez les journaux du conteneur concerné :

```bash
docker logs spark-iceberg --tail 50
```

Solutions possibles :
- Augmentez la mémoire allouée à Docker Desktop (Settings → Resources)
- Libérez les ports utilisés par d'autres applications
- Supprimez les volumes corrompus : `docker-compose down -v && docker-compose up -d`

**Problème : Erreur de connexion S3 dans Spark**

Les messages d'erreur contenant `Unable to execute HTTP request` ou `Access Denied` indiquent un problème de configuration des identifiants ou de l'endpoint S3.

Vérifiez :
- La cohérence des identifiants entre le fichier Docker Compose et la configuration Spark
- L'accessibilité du service MinIO depuis le conteneur Spark : `docker exec spark-iceberg curl http://minio:9000`
- L'activation de l'accès par chemin (path-style) : `fs.s3a.path.style.access=true`

**Problème : Nessie ne répond pas aux requêtes**

Le catalogue Nessie peut mettre jusqu'à 30 secondes à s'initialiser complètement. Si les problèmes persistent :

```bash
# Vérifier les journaux Nessie
docker logs nessie

# Redémarrer le service
docker-compose restart nessie
```

> **Performance**  
> Sur une machine équipée de 16 Go de RAM, allouez au moins 8 Go à Docker Desktop pour garantir des performances optimales. Les opérations Spark sur des jeux de données volumineux peuvent être limitées par la mémoire disponible dans l'environnement conteneurisé. Pour des charges de travail plus importantes, envisagez d'utiliser un cluster Spark distant ou des services infonuagiques gérés comme Amazon EMR, Google Dataproc ou Azure HDInsight.

### Comprendre l'architecture des fichiers Iceberg sur MinIO

Avant de passer à la création de tables, il est instructif de comprendre comment Apache Iceberg organise les données sur le stockage objet. Lorsque vous créerez vos premières tables, MinIO contiendra la structure suivante :

```
warehouse/
├── ecommerce/                          # Espace de noms (namespace)
│   ├── clients/                        # Répertoire de la table
│   │   ├── metadata/                   # Métadonnées Iceberg
│   │   │   ├── v1.metadata.json        # Fichier de métadonnées (version 1)
│   │   │   ├── v2.metadata.json        # Fichier de métadonnées (version 2)
│   │   │   ├── snap-1234567890.avro    # Fichier manifest list
│   │   │   └── 0000-abc123.avro        # Fichier manifest
│   │   └── data/                       # Fichiers de données
│   │       ├── date_inscription_month=2024-01/
│   │       │   └── province=QC/
│   │       │       └── 00000-0-abc.parquet
│   │       └── date_inscription_month=2024-02/
│   │           └── ...
│   └── transactions/
│       ├── metadata/
│       └── data/
```

Cette organisation reflète les concepts abordés au Chapitre IV.2 :

- **Fichiers de métadonnées** (`*.metadata.json`) : Pointent vers les manifest lists et contiennent le schéma, les propriétés de table et l'historique des snapshots.
- **Manifest lists** (`snap-*.avro`) : Répertorient les fichiers manifest associés à chaque snapshot.
- **Manifests** (`0000-*.avro`) : Cataloguent les fichiers de données avec leurs statistiques (min/max, nombre d'enregistrements).
- **Fichiers de données** (`*.parquet`) : Contiennent les données réelles au format Parquet, organisées par partition.

---

## IV.3.2 Création de tables Iceberg dans Spark

Apache Spark demeure le moteur de traitement de référence pour Apache Iceberg, offrant le support le plus complet des fonctionnalités du format de table. Dans cette section, nous allons créer notre première table Iceberg, y insérer des données, explorer les capacités de voyage dans le temps (time travel) et démontrer l'évolution de schéma.

### Accès à l'environnement Spark

Ouvrez l'interface Jupyter Notebook en naviguant vers `http://localhost:8888`. L'image Docker `tabulario/spark-iceberg` inclut une configuration Spark préconfigurée pour Iceberg avec un catalogue Nessie. Créez un nouveau notebook Python et exécutez les cellules suivantes pour initialiser votre session Spark.

```python
from pyspark.sql import SparkSession
from pyspark.sql.types import (
    StructType, StructField, StringType, 
    IntegerType, DoubleType, TimestampType, DateType
)
from datetime import datetime, date
import os

# Configuration de la session Spark avec Iceberg et Nessie
spark = SparkSession.builder \
    .appName("Lakehouse-Demo") \
    .config("spark.jars.packages", 
            "org.apache.iceberg:iceberg-spark-runtime-3.5_2.12:1.5.2,"
            "org.projectnessie.nessie-integrations:nessie-spark-extensions-3.5_2.12:0.77.1,"
            "software.amazon.awssdk:bundle:2.24.8,"
            "software.amazon.awssdk:url-connection-client:2.24.8") \
    .config("spark.sql.extensions", 
            "org.apache.iceberg.spark.extensions.IcebergSparkSessionExtensions,"
            "org.projectnessie.spark.extensions.NessieSparkSessionExtensions") \
    .config("spark.sql.catalog.nessie", "org.apache.iceberg.spark.SparkCatalog") \
    .config("spark.sql.catalog.nessie.uri", "http://nessie:19120/api/v2") \
    .config("spark.sql.catalog.nessie.ref", "main") \
    .config("spark.sql.catalog.nessie.authentication.type", "NONE") \
    .config("spark.sql.catalog.nessie.catalog-impl", 
            "org.apache.iceberg.nessie.NessieCatalog") \
    .config("spark.sql.catalog.nessie.warehouse", "s3://warehouse") \
    .config("spark.sql.catalog.nessie.io-impl", 
            "org.apache.iceberg.aws.s3.S3FileIO") \
    .config("spark.sql.catalog.nessie.s3.endpoint", "http://minio:9000") \
    .config("spark.sql.catalog.nessie.s3.path-style-access", "true") \
    .config("spark.hadoop.fs.s3a.endpoint", "http://minio:9000") \
    .config("spark.hadoop.fs.s3a.access.key", "admin") \
    .config("spark.hadoop.fs.s3a.secret.key", "password123") \
    .config("spark.hadoop.fs.s3a.path.style.access", "true") \
    .config("spark.hadoop.fs.s3a.impl", 
            "org.apache.hadoop.fs.s3a.S3AFileSystem") \
    .getOrCreate()

print(f"Session Spark initialisée : {spark.version}")
```

### Création de l'espace de noms et de la table

Dans Apache Iceberg, les tables sont organisées dans des espaces de noms (namespaces) qui correspondent conceptuellement aux bases de données dans un système de gestion de base de données relationnelle (SGBDR) traditionnel. Créons un espace de noms pour notre démonstration :

```python
# Créer l'espace de noms (équivalent d'une base de données)
spark.sql("CREATE NAMESPACE IF NOT EXISTS nessie.ecommerce")

# Vérifier les espaces de noms disponibles
spark.sql("SHOW NAMESPACES IN nessie").show()
```

Maintenant, créons une table `clients` avec un schéma représentatif d'une application de commerce électronique. Cette table utilisera le partitionnement masqué (hidden partitioning) d'Iceberg sur la colonne de date d'inscription :

```python
# Création de la table clients avec partitionnement par mois
spark.sql("""
    CREATE TABLE IF NOT EXISTS nessie.ecommerce.clients (
        client_id BIGINT,
        prenom STRING,
        nom STRING,
        courriel STRING,
        province STRING,
        date_inscription DATE,
        segment STRING,
        valeur_vie DOUBLE
    )
    USING iceberg
    PARTITIONED BY (months(date_inscription), province)
    TBLPROPERTIES (
        'write.format.default' = 'parquet',
        'write.parquet.compression-codec' = 'zstd',
        'commit.retry.num-retries' = '4',
        'write.metadata.delete-after-commit.enabled' = 'true',
        'write.metadata.previous-versions-max' = '100'
    )
""")

print("Table 'clients' créée avec succès")
```

Examinons les propriétés importantes de cette définition :

- **`PARTITIONED BY (months(date_inscription), province)`** : Le partitionnement masqué transforme automatiquement la date d'inscription en partitions mensuelles sans exposer cette complexité aux utilisateurs finaux. Les requêtes peuvent filtrer directement sur `date_inscription` sans connaître la structure de partitionnement.

- **`write.format.default = 'parquet'`** : Les données sont stockées au format Parquet, optimisé pour les requêtes analytiques avec sa structure colonnaire.

- **`write.parquet.compression-codec = 'zstd'`** : L'algorithme de compression Zstandard offre un excellent compromis entre taux de compression et vitesse de décompression.

- **`commit.retry.num-retries = 4`** : Cette propriété définit le nombre de tentatives en cas de conflit d'écriture concurrente, assurant la robustesse des opérations transactionnelles.

- **`write.metadata.delete-after-commit.enabled = true`** : Active le nettoyage automatique des anciens fichiers de métadonnées après chaque commit, évitant l'accumulation de fichiers obsolètes.

- **`write.metadata.previous-versions-max = 100`** : Limite le nombre de versions de métadonnées conservées, équilibrant la capacité de voyage dans le temps avec l'efficacité du stockage.

Ces propriétés illustrent la richesse de configuration offerte par Apache Iceberg. Dans un environnement de production, ces paramètres seraient ajustés en fonction des patrons d'accès aux données, des exigences de conformité et des contraintes de performance spécifiques à chaque cas d'utilisation. Le Chapitre IV.10 approfondit les stratégies d'optimisation et de maintenance des tables Iceberg.

### Insertion de données

Insérons un ensemble de données représentatif de clients canadiens :

```python
from pyspark.sql import Row

# Données d'exemple représentant des clients canadiens
donnees_clients = [
    Row(1001, "Marie", "Tremblay", "marie.tremblay@courriel.ca", "QC", 
        date(2024, 1, 15), "Premium", 12500.00),
    Row(1002, "Jean", "Gagnon", "jean.gagnon@courriel.ca", "QC", 
        date(2024, 1, 22), "Standard", 3200.00),
    Row(1003, "Sarah", "Chen", "sarah.chen@email.ca", "ON", 
        date(2024, 2, 3), "Premium", 18900.00),
    Row(1004, "Mohammed", "Hassan", "m.hassan@mail.ca", "ON", 
        date(2024, 2, 14), "Standard", 2100.00),
    Row(1005, "Emily", "Smith", "emily.smith@email.ca", "BC", 
        date(2024, 2, 28), "Enterprise", 45000.00),
    Row(1006, "Pierre", "Lavoie", "p.lavoie@courriel.ca", "QC", 
        date(2024, 3, 5), "Premium", 8700.00),
    Row(1007, "Anika", "Patel", "anika.patel@email.ca", "AB", 
        date(2024, 3, 12), "Standard", 4300.00),
    Row(1008, "François", "Dubois", "f.dubois@courriel.ca", "QC", 
        date(2024, 3, 20), "Premium", 15200.00),
    Row(1009, "Wei", "Zhang", "wei.zhang@email.ca", "BC", 
        date(2024, 4, 1), "Enterprise", 52000.00),
    Row(1010, "Sophie", "Martin", "sophie.martin@courriel.ca", "QC", 
        date(2024, 4, 8), "Standard", 2800.00),
]

# Définir le schéma explicitement
schema_clients = StructType([
    StructField("client_id", IntegerType(), False),
    StructField("prenom", StringType(), True),
    StructField("nom", StringType(), True),
    StructField("courriel", StringType(), True),
    StructField("province", StringType(), True),
    StructField("date_inscription", DateType(), True),
    StructField("segment", StringType(), True),
    StructField("valeur_vie", DoubleType(), True),
])

# Créer le DataFrame et insérer les données
df_clients = spark.createDataFrame(donnees_clients, schema_clients)
df_clients.writeTo("nessie.ecommerce.clients").append()

print(f"Insertion de {df_clients.count()} enregistrements réussie")
```

### Validation et exploration des données

Vérifions que les données ont été correctement insérées et explorons la structure créée par Iceberg :

```python
# Requête de validation
spark.sql("""
    SELECT province, segment, COUNT(*) as nb_clients, 
           ROUND(AVG(valeur_vie), 2) as valeur_moyenne
    FROM nessie.ecommerce.clients
    GROUP BY province, segment
    ORDER BY province, segment
""").show()

# Examiner les métadonnées de la table
spark.sql("DESCRIBE EXTENDED nessie.ecommerce.clients").show(truncate=False)
```

### Exploration des snapshots et du voyage dans le temps

L'une des fonctionnalités les plus puissantes d'Apache Iceberg est sa capacité de voyage dans le temps (time travel). Chaque opération d'écriture crée un nouveau snapshot, permettant de consulter l'état de la table à n'importe quel moment dans le passé. Cette fonctionnalité s'avère particulièrement précieuse pour plusieurs cas d'utilisation :

- **Audit et conformité** : Reconstituer l'état exact des données à un moment donné pour répondre aux exigences réglementaires
- **Débogage** : Comparer les données avant et après une transformation pour identifier les anomalies
- **Récupération** : Restaurer une table à un état antérieur après une modification erronée
- **Analyse temporelle** : Comparer les métriques entre différentes périodes avec une cohérence garantie

```python
# Consulter l'historique des snapshots
spark.sql("""
    SELECT 
        snapshot_id,
        committed_at,
        operation,
        summary['added-records'] as enregistrements_ajoutes,
        summary['total-records'] as total_enregistrements
    FROM nessie.ecommerce.clients.snapshots
    ORDER BY committed_at DESC
""").show(truncate=False)
```

Cette requête révèle l'historique complet des modifications de la table. Le champ `operation` indique le type d'opération (append, overwrite, delete, replace), tandis que le dictionnaire `summary` contient des statistiques détaillées sur chaque snapshot.

Pour comprendre comment Iceberg gère efficacement le voyage dans le temps, il est utile de visualiser la chaîne de snapshots. Chaque snapshot pointe vers un ensemble de manifests qui, à leur tour, référencent les fichiers de données. Lorsqu'une requête spécifie un snapshot historique, Iceberg navigue directement vers les fichiers pertinents sans avoir à parcourir l'historique complet.

Effectuons une modification pour créer un nouveau snapshot, puis voyageons dans le temps :

```python
# Mise à jour : augmenter la valeur vie des clients Premium
spark.sql("""
    UPDATE nessie.ecommerce.clients
    SET valeur_vie = valeur_vie * 1.1
    WHERE segment = 'Premium'
""")

print("Mise à jour effectuée - nouveau snapshot créé")

# Consulter l'historique mis à jour
spark.sql("""
    SELECT snapshot_id, committed_at, operation
    FROM nessie.ecommerce.clients.snapshots
    ORDER BY committed_at DESC
""").show()
```

Pour consulter l'état de la table avant la mise à jour, utilisez la syntaxe de voyage dans le temps :

```python
# Récupérer l'identifiant du premier snapshot
premier_snapshot = spark.sql("""
    SELECT snapshot_id 
    FROM nessie.ecommerce.clients.snapshots 
    ORDER BY committed_at ASC 
    LIMIT 1
""").collect()[0][0]

# Requête sur l'état historique
spark.sql(f"""
    SELECT prenom, nom, segment, valeur_vie
    FROM nessie.ecommerce.clients
    VERSION AS OF {premier_snapshot}
    WHERE segment = 'Premium'
    ORDER BY valeur_vie DESC
""").show()

# Comparer avec l'état actuel
spark.sql("""
    SELECT prenom, nom, segment, valeur_vie
    FROM nessie.ecommerce.clients
    WHERE segment = 'Premium'
    ORDER BY valeur_vie DESC
""").show()
```

### Démonstration de l'évolution de schéma

L'évolution de schéma (schema evolution) constitue une autre capacité distinctive d'Apache Iceberg. Ajoutons une nouvelle colonne à notre table sans perturber les données existantes :

```python
# Ajouter une colonne pour le canal d'acquisition
spark.sql("""
    ALTER TABLE nessie.ecommerce.clients 
    ADD COLUMN canal_acquisition STRING AFTER segment
""")

# Vérifier le schéma mis à jour
spark.sql("DESCRIBE nessie.ecommerce.clients").show()

# Mettre à jour les enregistrements existants
spark.sql("""
    UPDATE nessie.ecommerce.clients
    SET canal_acquisition = 'Migration historique'
    WHERE canal_acquisition IS NULL
""")

# Insérer de nouveaux enregistrements avec la nouvelle colonne
nouveaux_clients = [
    Row(1011, "Alexandre", "Roy", "alex.roy@courriel.ca", "QC",
        date(2024, 4, 15), "Premium", "Site web", 9500.00),
    Row(1012, "Priya", "Sharma", "priya.sharma@email.ca", "ON",
        date(2024, 4, 18), "Standard", "Référence", 3100.00),
]

schema_etendu = StructType([
    StructField("client_id", IntegerType(), False),
    StructField("prenom", StringType(), True),
    StructField("nom", StringType(), True),
    StructField("courriel", StringType(), True),
    StructField("province", StringType(), True),
    StructField("date_inscription", DateType(), True),
    StructField("segment", StringType(), True),
    StructField("canal_acquisition", StringType(), True),
    StructField("valeur_vie", DoubleType(), True),
])

df_nouveaux = spark.createDataFrame(nouveaux_clients, schema_etendu)
df_nouveaux.writeTo("nessie.ecommerce.clients").append()

# Valider l'insertion
spark.sql("""
    SELECT prenom, nom, canal_acquisition, valeur_vie
    FROM nessie.ecommerce.clients
    WHERE canal_acquisition != 'Migration historique'
""").show()
```

> **Étude de cas : Desjardins – Évolution de schéma sans interruption**  
> *Secteur* : Services financiers  
> *Défi* : Ajouter des attributs de conformité réglementaire à des tables contenant des milliards d'enregistrements sans interrompre les opérations analytiques quotidiennes.  
> *Solution* : Migration vers Apache Iceberg pour bénéficier de l'évolution de schéma native. Les nouvelles colonnes sont ajoutées à chaud, et les processus de remplissage (backfill) s'exécutent en parallèle des requêtes de lecture.  
> *Résultats* : Zéro temps d'arrêt pendant les modifications de schéma, réduction de 80 % du temps de mise en production des nouveaux attributs de données.

### Création d'une table de transactions

Pour enrichir notre démonstration, créons une table de transactions liée aux clients :

```python
# Création de la table transactions
spark.sql("""
    CREATE TABLE IF NOT EXISTS nessie.ecommerce.transactions (
        transaction_id BIGINT,
        client_id BIGINT,
        date_transaction TIMESTAMP,
        montant DOUBLE,
        categorie STRING,
        mode_paiement STRING,
        statut STRING
    )
    USING iceberg
    PARTITIONED BY (days(date_transaction), categorie)
    TBLPROPERTIES (
        'write.format.default' = 'parquet',
        'write.parquet.compression-codec' = 'zstd'
    )
""")

# Générer des données de transactions
from random import randint, choice, uniform
from datetime import timedelta

transactions = []
categories = ['Électronique', 'Vêtements', 'Alimentation', 'Maison', 'Sports']
modes_paiement = ['Carte crédit', 'Carte débit', 'PayPal', 'Virement']
statuts = ['Complétée', 'Complétée', 'Complétée', 'Remboursée', 'En attente']

for i in range(1, 101):
    transactions.append(Row(
        transaction_id=10000 + i,
        client_id=1001 + (i % 12),
        date_transaction=datetime(2024, 1, 1) + timedelta(days=randint(0, 120)),
        montant=round(uniform(25.0, 500.0), 2),
        categorie=choice(categories),
        mode_paiement=choice(modes_paiement),
        statut=choice(statuts)
    ))

schema_transactions = StructType([
    StructField("transaction_id", IntegerType(), False),
    StructField("client_id", IntegerType(), True),
    StructField("date_transaction", TimestampType(), True),
    StructField("montant", DoubleType(), True),
    StructField("categorie", StringType(), True),
    StructField("mode_paiement", StringType(), True),
    StructField("statut", StringType(), True),
])

df_transactions = spark.createDataFrame(transactions, schema_transactions)
df_transactions.writeTo("nessie.ecommerce.transactions").append()

print(f"Insertion de {len(transactions)} transactions réussie")

# Requête analytique joignant les deux tables
spark.sql("""
    SELECT 
        c.province,
        c.segment,
        COUNT(DISTINCT t.transaction_id) as nb_transactions,
        ROUND(SUM(t.montant), 2) as revenu_total,
        ROUND(AVG(t.montant), 2) as panier_moyen
    FROM nessie.ecommerce.clients c
    JOIN nessie.ecommerce.transactions t ON c.client_id = t.client_id
    WHERE t.statut = 'Complétée'
    GROUP BY c.province, c.segment
    ORDER BY revenu_total DESC
""").show()
```

### Opérations avancées : MERGE INTO

L'instruction `MERGE INTO` constitue l'une des capacités les plus puissantes d'Iceberg pour les scénarios d'upsert (insertion ou mise à jour). Cette opération atomique permet de synchroniser une table cible avec une source de données en une seule transaction :

```python
# Créer une table temporaire avec des mises à jour
spark.sql("""
    CREATE OR REPLACE TEMPORARY VIEW mises_a_jour_clients AS
    SELECT * FROM VALUES
        (1001, 'Marie', 'Tremblay-Roy', 'marie.tremblay@nouveau.ca', 'QC', 
         DATE('2024-01-15'), 'Enterprise', 'Référence', 25000.00),
        (1013, 'Lucas', 'Bergeron', 'lucas.bergeron@courriel.ca', 'QC',
         DATE('2024-05-01'), 'Standard', 'Site web', 1500.00)
    AS t(client_id, prenom, nom, courriel, province, date_inscription, 
         segment, canal_acquisition, valeur_vie)
""")

# Exécuter le MERGE INTO
spark.sql("""
    MERGE INTO nessie.ecommerce.clients AS cible
    USING mises_a_jour_clients AS source
    ON cible.client_id = source.client_id
    WHEN MATCHED THEN UPDATE SET
        nom = source.nom,
        courriel = source.courriel,
        segment = source.segment,
        valeur_vie = source.valeur_vie
    WHEN NOT MATCHED THEN INSERT *
""")

# Vérifier les résultats
spark.sql("""
    SELECT client_id, prenom, nom, segment, valeur_vie
    FROM nessie.ecommerce.clients
    WHERE client_id IN (1001, 1013)
""").show()
```

Cette opération illustre plusieurs comportements importants :
- Le client 1001 existant a été mis à jour avec les nouvelles valeurs
- Le client 1013, absent de la table, a été inséré
- L'ensemble de l'opération s'est exécuté de manière atomique, créant un nouveau snapshot

### Exploration des fichiers de métadonnées

Pour approfondir la compréhension du fonctionnement interne d'Iceberg, explorons les tables de métadonnées :

```python
# Statistiques sur les fichiers de données
spark.sql("""
    SELECT 
        file_path,
        file_format,
        record_count,
        file_size_in_bytes / 1024 as taille_ko,
        lower_bounds,
        upper_bounds
    FROM nessie.ecommerce.clients.files
""").show(truncate=False)

# Historique des partitions
spark.sql("""
    SELECT 
        partition,
        record_count,
        file_count
    FROM nessie.ecommerce.clients.partitions
""").show(truncate=False)

# Manifests et leur contenu
spark.sql("""
    SELECT 
        path,
        length,
        partition_spec_id,
        added_snapshot_id,
        added_data_files_count,
        existing_data_files_count,
        deleted_data_files_count
    FROM nessie.ecommerce.clients.manifests
""").show(truncate=False)
```

Ces tables de métadonnées exposent les détails internes de la structure Iceberg, permettant une observabilité complète sur l'état de vos tables.

### Utilisation des branches Nessie (optionnel)

Le catalogue Nessie apporte une dimension supplémentaire : le versionnement des données à la manière de Git. Cette capacité permet de créer des branches pour expérimenter sans affecter les données de production :

```python
# Créer une branche d'expérimentation
spark.sql("CREATE BRANCH IF NOT EXISTS experimentation IN nessie")

# Basculer vers la branche
spark.sql("USE REFERENCE experimentation IN nessie")

# Effectuer des modifications sur la branche
spark.sql("""
    DELETE FROM nessie.ecommerce.clients
    WHERE segment = 'Standard'
""")

# Vérifier l'état de la branche
spark.sql("""
    SELECT segment, COUNT(*) as nombre
    FROM nessie.ecommerce.clients
    GROUP BY segment
""").show()

# Revenir à la branche principale (main)
spark.sql("USE REFERENCE main IN nessie")

# Vérifier que les données de production sont intactes
spark.sql("""
    SELECT segment, COUNT(*) as nombre
    FROM nessie.ecommerce.clients
    GROUP BY segment
""").show()
```

Cette fonctionnalité s'avère particulièrement précieuse pour :
- Les environnements de développement et de test isolés
- La validation des transformations avant déploiement en production
- Les analyses exploratoires nécessitant des modifications temporaires

---

## IV.3.3 Lecture des tables Iceberg dans Dremio

Dremio représente la couche de fédération et d'accélération de requêtes dans notre architecture Lakehouse. Cette plateforme permet d'unifier l'accès aux données provenant de sources multiples, d'optimiser les performances grâce aux réflexions de données et de fournir une interface SQL accessible aux analystes métier. L'intégration native avec Apache Iceberg fait de Dremio un choix privilégié pour la consommation des données du Lakehouse.

L'architecture de Dremio repose sur plusieurs composants clés qui contribuent à ses performances remarquables :

- **Moteur de requêtes distribué** : Basé sur Apache Arrow, le moteur de Dremio traite les données en mémoire avec un format colonnaire optimisé pour les processeurs modernes (vectorisation SIMD).

- **Couche de planification intelligente** : Analyse les requêtes pour déterminer le plan d'exécution optimal, incluant la substitution automatique par des réflexions précalculées.

- **Gestion des réflexions** : Système de matérialisations automatiques qui crée des vues physiques optimisées sous forme de tables Iceberg.

- **Connecteurs natifs** : Support direct de nombreuses sources de données, évitant les couches d'abstraction coûteuses.

Pour les utilisateurs familiers avec les entrepôts de données traditionnels, Dremio offre une expérience SQL complète tout en éliminant la nécessité de déplacer les données vers une infrastructure propriétaire. Les données restent sur votre Lakehouse (stockage objet), et Dremio fournit l'intelligence de requête au-dessus.

### Configuration initiale de Dremio

Accédez à l'interface Dremio via `http://localhost:9047`. Lors de votre première connexion, vous serez invité à créer un compte administrateur. Complétez l'assistant de configuration avec les informations suivantes :

1. **Création du compte** : Définissez un nom d'utilisateur (par exemple, `admin`) et un mot de passe robuste.

2. **Configuration du projet** : Acceptez les paramètres par défaut pour l'environnement de développement.

### Connexion au catalogue Nessie

Dremio prend en charge nativement les catalogues Nessie, offrant une intégration transparente avec nos tables Iceberg. Pour configurer cette connexion :

1. Cliquez sur **Add Source** dans le panneau de navigation gauche.

2. Sélectionnez **Nessie** dans la liste des sources disponibles.

3. Configurez les paramètres suivants :

| Paramètre | Valeur |
|-----------|--------|
| Name | `lakehouse` |
| Nessie Endpoint URL | `http://nessie:19120/api/v2` |
| Authentication | None |
| AWS Root Path | `warehouse` |
| AWS Access Key | `admin` |
| AWS Secret Key | `password123` |
| Connection Properties | `fs.s3a.endpoint=http://minio:9000` |
| | `fs.s3a.path.style.access=true` |
| Encrypt connection | Désactivé |

4. Cliquez sur **Save** pour établir la connexion.

Après la sauvegarde, Dremio découvre automatiquement les espaces de noms et les tables présents dans le catalogue Nessie. Vous devriez voir apparaître l'espace de noms `ecommerce` avec les tables `clients` et `transactions`.

### Exploration des données via l'interface SQL

Dremio offre un éditeur SQL intégré permettant d'interroger directement les tables Iceberg. Accédez à l'onglet **SQL Runner** et exécutez les requêtes suivantes :

```sql
-- Exploration de la table clients
SELECT * FROM lakehouse.ecommerce.clients
LIMIT 10;

-- Statistiques par province
SELECT 
    province,
    COUNT(*) AS nombre_clients,
    ROUND(AVG(valeur_vie), 2) AS valeur_moyenne,
    ROUND(SUM(valeur_vie), 2) AS valeur_totale
FROM lakehouse.ecommerce.clients
GROUP BY province
ORDER BY valeur_totale DESC;
```

### Fonctionnalités Iceberg dans Dremio

Dremio expose les fonctionnalités avancées d'Iceberg à travers des extensions SQL propriétaires et des tables de métadonnées.

**Consultation de l'historique des snapshots** :

```sql
-- Historique des modifications
SELECT * FROM TABLE(
    table_history('lakehouse.ecommerce.clients')
);

-- Liste des snapshots
SELECT * FROM TABLE(
    table_snapshot('lakehouse.ecommerce.clients')
);
```

**Voyage dans le temps** :

```sql
-- Requête sur un snapshot spécifique
SELECT * FROM lakehouse.ecommerce.clients
AT SNAPSHOT '1234567890123456789';

-- Requête à un moment précis
SELECT * FROM lakehouse.ecommerce.clients
AT BRANCH main
AS OF '2024-04-01 00:00:00';
```

**Exploration des métadonnées de fichiers** :

```sql
-- Statistiques sur les fichiers de données
SELECT 
    file_path,
    file_format,
    record_count,
    file_size_in_bytes
FROM TABLE(
    table_files('lakehouse.ecommerce.clients')
);
```

### Création de vues virtuelles

Les vues virtuelles (Virtual Datasets) de Dremio permettent de créer des abstractions sémantiques au-dessus des tables Iceberg, simplifiant l'accès pour les utilisateurs métier :

```sql
-- Vue consolidée clients-transactions
CREATE VDS lakehouse.analytique.resume_clients AS
SELECT 
    c.client_id,
    c.prenom || ' ' || c.nom AS nom_complet,
    c.province,
    c.segment,
    c.valeur_vie,
    COUNT(t.transaction_id) AS nombre_transactions,
    COALESCE(SUM(t.montant), 0) AS montant_total_achats,
    COALESCE(AVG(t.montant), 0) AS panier_moyen,
    MAX(t.date_transaction) AS derniere_transaction
FROM lakehouse.ecommerce.clients c
LEFT JOIN lakehouse.ecommerce.transactions t 
    ON c.client_id = t.client_id
    AND t.statut = 'Complétée'
GROUP BY 
    c.client_id, c.prenom, c.nom, 
    c.province, c.segment, c.valeur_vie;
```

### Configuration des réflexions de données

Les réflexions de données (Data Reflections) constituent la fonctionnalité d'accélération signature de Dremio. Elles créent des matérialisations optimisées des données sous forme de tables Iceberg, que Dremio substitue automatiquement lors de l'exécution des requêtes.

Le principe de fonctionnement des réflexions est élégant : lorsque vous activez une réflexion sur une vue ou une table, Dremio matérialise les données dans un format optimisé. Lors de l'exécution ultérieure de requêtes, le planificateur évalue si la réflexion peut satisfaire la requête. Si c'est le cas, la requête est automatiquement redirigée vers la réflexion, offrant des temps de réponse considérablement réduits.

Dremio propose deux types de réflexions :

**Réflexions brutes (Raw Reflections)** : Matérialisent l'ensemble des données d'une table ou vue. Elles accélèrent les requêtes qui lisent un grand nombre de colonnes ou qui nécessitent un scan complet. Les réflexions brutes sont particulièrement utiles pour :
- Les requêtes ad hoc imprévisibles
- Les exports de données volumineuses
- Les jointures nécessitant l'accès à toutes les colonnes

**Réflexions d'agrégation (Aggregation Reflections)** : Précalculent des agrégations selon des dimensions définies. Elles accélèrent dramatiquement les requêtes analytiques typiques (sommes, moyennes, comptages). Les réflexions d'agrégation excellent pour :
- Les tableaux de bord avec des KPI prédéfinis
- Les rapports périodiques standardisés
- Les requêtes avec des GROUP BY fréquents

Pour activer les réflexions sur notre vue :

1. Naviguez vers la vue `resume_clients` dans l'explorateur de données.

2. Cliquez sur l'onglet **Reflections**.

3. Activez **Raw Reflections** pour matérialiser l'ensemble des données.

4. Configurez **Aggregation Reflections** avec les dimensions et mesures suivantes :

| Type | Colonnes |
|------|----------|
| Dimensions | `province`, `segment` |
| Mesures | `SUM(montant_total_achats)`, `COUNT(nombre_transactions)`, `AVG(panier_moyen)` |

5. Définissez une politique de rafraîchissement (par exemple, toutes les heures).

La configuration du rafraîchissement mérite une attention particulière. Dremio supporte plusieurs modes :
- **Rafraîchissement complet** : Reconstruit entièrement la réflexion à chaque cycle
- **Rafraîchissement incrémental** : Met à jour uniquement les partitions modifiées (recommandé pour Iceberg)
- **Rafraîchissement basé sur les snapshots** : Détecte automatiquement les nouveaux snapshots Iceberg

> **Performance**  
> Les réflexions de données peuvent améliorer les performances des requêtes de 10 à 100 fois selon la complexité des agrégations et le volume de données. Dans notre environnement de démonstration, l'impact sera modeste, mais sur des tables de plusieurs téraoctets, les gains sont substantiels. Une étude de Dremio (2024) rapporte des temps de réponse passant de plusieurs minutes à moins d'une seconde pour des requêtes d'agrégation sur des tables de 500 Go.

### Connexion à des sources externes

Dremio excelle dans la fédération de données provenant de sources hétérogènes. Vous pouvez ajouter des connexions vers :

- **Bases de données relationnelles** : PostgreSQL, MySQL, SQL Server, Oracle
- **Entrepôts de données infonuagiques** : Snowflake, BigQuery, Redshift
- **Stockage objet** : Amazon S3, Azure Blob Storage, Google Cloud Storage
- **Systèmes de fichiers** : HDFS, NAS, partages réseau

Cette capacité permet de créer des requêtes fédérées joignant des tables Iceberg du Lakehouse avec des données opérationnelles sans déplacer physiquement les données.

### Requêtes fédérées et virtualisation

L'un des avantages majeurs de Dremio réside dans sa capacité à exécuter des requêtes fédérées transparentes. Imaginons que vous ajoutiez une source PostgreSQL contenant des données de référence :

```sql
-- Requête fédérée combinant Iceberg et PostgreSQL
SELECT 
    c.client_id,
    c.nom,
    c.province,
    ref.nom_region,
    ref.fuseau_horaire,
    SUM(t.montant) AS total_achats
FROM lakehouse.ecommerce.clients c
JOIN postgres_source.public.regions ref 
    ON c.province = ref.code_province
LEFT JOIN lakehouse.ecommerce.transactions t 
    ON c.client_id = t.client_id
GROUP BY c.client_id, c.nom, c.province, 
         ref.nom_region, ref.fuseau_horaire
ORDER BY total_achats DESC;
```

Dremio optimise automatiquement l'exécution en :
- Poussant les filtres vers les sources (predicate pushdown)
- Parallélisant l'accès aux différentes sources
- Utilisant les réflexions disponibles pour accélérer les agrégations

### Gestion des droits d'accès

Dremio intègre un système de contrôle d'accès granulaire permettant de sécuriser l'accès aux données :

```sql
-- Accorder l'accès en lecture sur un espace de noms
GRANT SELECT ON lakehouse.ecommerce TO analystes;

-- Créer une politique de masquage des données sensibles
ALTER TABLE lakehouse.ecommerce.clients 
SET MASKING POLICY courriel_masque ON courriel;

-- Limiter l'accès par lignes (Row-Level Security)
ALTER TABLE lakehouse.ecommerce.clients
SET ROW ACCESS POLICY filtre_province 
ON province USING (province = CURRENT_USER_PROVINCE());
```

Ces fonctionnalités de gouvernance s'avèrent essentielles pour les déploiements en entreprise où la conformité réglementaire (Loi 25, RGPD) impose des contrôles stricts sur l'accès aux données personnelles.

### Optimisation des requêtes avec EXPLAIN

Dremio fournit des outils de diagnostic permettant de comprendre et d'optimiser l'exécution des requêtes :

```sql
-- Analyser le plan d'exécution
EXPLAIN PLAN FOR
SELECT province, segment, COUNT(*), AVG(valeur_vie)
FROM lakehouse.ecommerce.clients
GROUP BY province, segment;

-- Voir le plan avec les coûts estimés
EXPLAIN PLAN INCLUDING ALL ATTRIBUTES FOR
SELECT c.province, SUM(t.montant)
FROM lakehouse.ecommerce.clients c
JOIN lakehouse.ecommerce.transactions t 
    ON c.client_id = t.client_id
GROUP BY c.province;
```

L'analyse du plan d'exécution révèle :
- Les opérations de lecture (scans) sur les tables Iceberg
- L'utilisation des statistiques de colonnes pour le filtrage
- La substitution par des réflexions lorsque disponibles
- Les opérations de brassage (shuffle) entre les nœuds

### Profils de requête et métriques

Après l'exécution d'une requête, Dremio capture un profil détaillé accessible via l'interface utilisateur (Jobs → Sélectionner la requête → View Profile). Ce profil contient :

- **Temps par phase** : Planification, exécution, écriture des résultats
- **Métriques d'entrées/sorties** : Octets lus/écrits, enregistrements traités
- **Utilisation des réflexions** : Indique si une réflexion a été utilisée
- **Parallélisme** : Nombre de threads et de fragments d'exécution

Ces informations permettent d'identifier les goulots d'étranglement et d'optimiser les requêtes ou les configurations de réflexions.

---

## IV.3.4 Création d'un tableau de bord BI

La valeur ultime d'un Lakehouse réside dans sa capacité à alimenter des analyses métier exploitables. Dans cette section, nous connecterons Dremio à des outils d'intelligence d'affaires pour créer des visualisations interactives. Nous explorerons deux approches : Apache Superset (solution à code source ouvert) et les options de connexion aux outils commerciaux comme Power BI et Tableau.

### Option 1 : Apache Superset

Apache Superset est une plateforme de visualisation de données moderne, à code source ouvert, capable de créer des tableaux de bord interactifs et des explorations de données ad hoc. Développé initialement par Airbnb et maintenant projet de la Fondation Apache, Superset s'est imposé comme l'alternative open source de référence aux solutions commerciales de BI.

Les points forts de Superset pour un contexte Lakehouse incluent :

- **Interface de création de graphiques intuitive** : Les utilisateurs non techniques peuvent créer des visualisations sans écrire de SQL
- **Support SQL complet** : Les analystes avancés bénéficient d'un éditeur SQL intégré avec autocomplétion
- **Tableaux de bord interactifs** : Filtres croisés, drill-down et paramètres dynamiques
- **Sécurité robuste** : Contrôle d'accès basé sur les rôles, intégration LDAP/OAuth
- **Extensibilité** : Architecture de plugins permettant d'ajouter de nouveaux types de visualisations

**Ajout de Superset à l'environnement Docker**

Étendez votre fichier `docker-compose.yml` avec le service Superset :

```yaml
  superset:
    image: apache/superset:latest
    container_name: superset
    depends_on:
      - dremio
    environment:
      SUPERSET_SECRET_KEY: 'votre_cle_secrete_complexe_ici'
      SUPERSET_LOAD_EXAMPLES: 'no'
    ports:
      - "8088:8088"
    volumes:
      - superset-data:/app/superset_home
    networks:
      - lakehouse-network
    command: >
      bash -c "
        superset db upgrade &&
        superset fab create-admin --username admin --firstname Admin --lastname User --email admin@exemple.ca --password admin123 &&
        superset init &&
        superset run -h 0.0.0.0 -p 8088
      "

volumes:
  # ... volumes existants ...
  superset-data:
```

Cette configuration effectue automatiquement l'initialisation de la base de données Superset, crée un compte administrateur et démarre le serveur web. En production, vous sépareriez ces étapes et utiliseriez des variables d'environnement sécurisées pour les identifiants.

Redémarrez l'environnement pour inclure Superset :

```bash
docker-compose up -d superset
```

L'initialisation de Superset prend généralement 2 à 3 minutes. Vous pouvez suivre la progression avec `docker logs -f superset`.

**Configuration de la connexion Dremio dans Superset**

1. Accédez à Superset via `http://localhost:8088` et connectez-vous (`admin` / `admin123`).

2. Naviguez vers **Settings → Database Connections → + Database**.

3. Sélectionnez **Other** comme type de base de données.

4. Utilisez la chaîne de connexion SQLAlchemy suivante :

```
dremio+flight://admin:votre_mot_de_passe@dremio:32010/dremio?UseEncryption=false
```

> **Note** : Le pilote Dremio Flight pour SQLAlchemy (`sqlalchemy-dremio`) doit être installé dans le conteneur Superset. Pour une installation persistante, créez une image Docker personnalisée basée sur `apache/superset` avec le pilote préinstallé :
>
> ```dockerfile
> FROM apache/superset:latest
> USER root
> RUN pip install sqlalchemy-dremio
> USER superset
> ```

5. Testez la connexion et enregistrez la configuration.

**Configuration avancée : Modèle de données Superset**

Superset utilise le concept de « datasets » (jeux de données) pour exposer les tables aux créateurs de tableaux de bord. Pour optimiser l'expérience utilisateur :

1. Créez un dataset pointant vers la vue `resume_clients` de Dremio.

2. Configurez les colonnes avec des métadonnées enrichies :
   - Définissez des libellés conviviaux (« Province » au lieu de « province »)
   - Marquez les colonnes numériques comme métriques
   - Définissez les formats d'affichage (devise, pourcentage)

3. Créez des métriques calculées récurrentes directement dans le dataset :
   - Revenu moyen par client : `SUM(montant_total_achats) / COUNT(client_id)`
   - Taux de conversion : `COUNT(CASE WHEN nombre_transactions > 0 THEN 1 END) / COUNT(*)`

**Création d'un tableau de bord analytique**

Avec la connexion établie, créez un tableau de bord présentant les indicateurs clés de performance (KPI) de notre plateforme de commerce électronique :

*Graphique 1 — Répartition des clients par segment*

1. Cliquez sur **+ Chart** et sélectionnez la source de données `resume_clients`.
2. Choisissez le type **Pie Chart**.
3. Configurez :
   - **Dimension** : `segment`
   - **Metric** : `COUNT(*)`
4. Enregistrez et ajoutez au tableau de bord.

*Graphique 2 — Revenus par province*

1. Créez un nouveau graphique de type **Bar Chart**.
2. Configurez :
   - **X-Axis** : `province`
   - **Metric** : `SUM(montant_total_achats)`
   - **Sort** : Décroissant par métrique
3. Ajoutez au tableau de bord.

*Graphique 3 — Évolution temporelle des transactions*

Pour ce graphique, nous avons besoin d'une requête personnalisée :

```sql
SELECT 
    DATE_TRUNC('week', date_transaction) AS semaine,
    categorie,
    SUM(montant) AS revenus,
    COUNT(*) AS volume
FROM lakehouse.ecommerce.transactions
WHERE statut = 'Complétée'
GROUP BY DATE_TRUNC('week', date_transaction), categorie
ORDER BY semaine;
```

1. Créez un graphique de type **Line Chart**.
2. Utilisez la requête SQL personnalisée.
3. Configurez :
   - **X-Axis** : `semaine`
   - **Y-Axis** : `revenus`
   - **Group By** : `categorie`
4. Ajoutez au tableau de bord.

### Option 2 : Connexion Power BI

Microsoft Power BI représente l'outil d'intelligence d'affaires le plus répandu dans les entreprises, particulièrement au Canada où l'écosystème Microsoft bénéficie d'une adoption massive. L'intégration avec Dremio offre aux organisations utilisant Power BI un chemin naturel vers le Lakehouse sans imposer de changement d'outils aux utilisateurs métier.

**Prérequis** :
- Power BI Desktop installé (version Windows)
- Pilote ODBC Dremio téléchargé depuis le site officiel Dremio
- Compte Dremio avec les permissions appropriées

L'architecture de connexion entre Power BI et Dremio s'appuie sur le protocole ODBC (Open Database Connectivity), un standard de l'industrie pour la connectivité aux bases de données. Cette approche garantit une compatibilité large et une maintenance simplifiée.

**Installation du pilote ODBC Dremio**

1. Téléchargez le pilote ODBC Dremio depuis https://www.dremio.com/drivers/
2. Exécutez l'installateur en tant qu'administrateur
3. Configurez un DSN (Data Source Name) système :
   - Ouvrez ODBC Data Source Administrator (64-bit)
   - Cliquez sur « Add » dans l'onglet « System DSN »
   - Sélectionnez « Dremio ODBC Driver »
   - Configurez les paramètres de connexion

**Configuration de la connexion** :

1. Dans Power BI Desktop, sélectionnez **Get Data → More → ODBC**.

2. Créez une nouvelle source de données ODBC avec les paramètres :
   - **DSN** : Créez un DSN système pointant vers `localhost:31010`
   - **Authentication** : Basic avec vos identifiants Dremio

3. Naviguez dans le catalogue Dremio et sélectionnez les tables ou vues à importer.

4. Choisissez le mode de connectivité (Import ou DirectQuery).

**Mode DirectQuery vs Import**

Le choix entre DirectQuery et Import constitue une décision architecturale importante qui influence les performances, la fraîcheur des données et l'expérience utilisateur :

| Aspect | DirectQuery | Import |
|--------|-------------|--------|
| **Fraîcheur des données** | Temps réel | Selon planification |
| **Performance des requêtes** | Variable (dépend de la source) | Optimale (données en mémoire) |
| **Fonctionnalités DAX** | Limitées | Complètes |
| **Taille des données** | Illimitée | Limitée par la mémoire |
| **Coût de licence** | Standard | Standard |
| **Charge sur la source** | Chaque interaction | Uniquement lors du rafraîchissement |

Pour un Lakehouse Iceberg avec Dremio, le mode **DirectQuery** est généralement recommandé car les réflexions Dremio compensent les latences de requête, et vous bénéficiez des données actualisées sans planification d'actualisation complexe. Cependant, pour des rapports nécessitant des calculs DAX complexes ou des visualisations avec de nombreuses interactions utilisateur, le mode Import peut offrir une meilleure expérience.

**Optimisation pour Power BI**

Plusieurs pratiques optimisent l'intégration Power BI-Dremio :

- **Créez des vues optimisées** : Préparez dans Dremio des vues correspondant exactement aux besoins des rapports, évitant les transformations côté Power BI
- **Utilisez les réflexions d'agrégation** : Configurez des réflexions alignées avec les agrégations fréquentes dans vos rapports
- **Limitez les colonnes** : Sélectionnez uniquement les colonnes nécessaires pour réduire le transfert de données
- **Évitez les filtres complexes** : Les prédicats simples sont plus efficacement poussés vers Dremio

> **Étude de cas : Banque Nationale – Migration vers Power BI Direct Query**  
> *Secteur* : Services financiers  
> *Défi* : Migrer des centaines de rapports SSRS vers Power BI tout en modernisant l'infrastructure de données sous-jacente vers un Lakehouse.  
> *Solution* : Déploiement progressif de Dremio comme couche sémantique unifiée, permettant aux rapports Power BI d'accéder aux données Iceberg via DirectQuery sans réécrire les modèles de données existants.  
> *Résultats* : Migration de 340 rapports en 6 mois avec une amélioration moyenne de 40 % des temps de réponse grâce aux réflexions Dremio.

### Option 3 : Connexion Tableau

Tableau représente la référence en matière de visualisation de données analytiques, particulièrement apprécié pour ses capacités d'exploration visuelle et son approche « glisser-déposer » intuitive. L'intégration avec Dremio combine la puissance d'exploration de Tableau avec les capacités d'accélération et de fédération du Lakehouse.

**Configuration de la connexion** :

1. Dans Tableau Desktop, sélectionnez **Connect → To a Server → More → Dremio**.

2. Entrez les paramètres de connexion :
   - **Server** : `localhost`
   - **Port** : `31010`
   - **Authentication** : Username and Password

3. Naviguez dans le catalogue et glissez les tables sur le canevas de modélisation.

**Modélisation des données dans Tableau**

Tableau offre plusieurs approches pour modéliser les relations entre tables :

- **Relations** : Approche recommandée depuis Tableau 2020.2, permettant des jointures dynamiques selon le contexte de la visualisation
- **Jointures physiques** : Jointures explicites définies au niveau de la source de données
- **Extraits** : Matérialisation locale des données pour des performances optimales hors ligne

Pour un Lakehouse Dremio, l'utilisation des **relations** avec une connexion **live** (DirectQuery équivalent) offre le meilleur équilibre entre fraîcheur des données et flexibilité de modélisation. Les réflexions Dremio compensent l'absence d'extraits locaux.

**Optimisation des performances Tableau**

Plusieurs techniques améliorent les performances de Tableau connecté à Dremio :

1. **Agrégations initiales** : Configurez Tableau pour utiliser les agrégations initiales (Initial SQL) afin d'exécuter des commandes préparatoires

2. **Context filters** : Utilisez les filtres de contexte pour réduire le volume de données avant les autres calculs

3. **Extraits hybrides** : Pour les données historiques stables, créez des extraits, tout en maintenant les données récentes en mode live

4. **Parallélisme** : Activez le parallélisme de requêtes dans les paramètres de source de données

> **Étude de cas : Air Canada – Tableaux de bord opérationnels**  
> *Secteur* : Transport aérien  
> *Défi* : Unifier les données de réservation, d'opérations et de fidélité provenant de dizaines de systèmes sources pour alimenter des tableaux de bord en temps quasi réel.  
> *Solution* : Déploiement d'un Lakehouse Iceberg avec Dremio comme couche de fédération, connecté à Tableau pour la visualisation. Les réflexions Dremio préagrègent les métriques clés.  
> *Résultats* : Temps de rafraîchissement des tableaux de bord réduit de 45 minutes à moins de 30 secondes. Consolidation de 23 sources de données en une interface analytique unifiée.

### Considérations pour la production

Lors du passage à un environnement de production, plusieurs aspects méritent attention :

**Sécurité des connexions**

- Activez le chiffrement TLS pour toutes les connexions entre les outils BI et Dremio.
- Implémentez l'authentification unique (SSO) via SAML ou OpenID Connect.
- Configurez le contrôle d'accès basé sur les rôles (RBAC) dans Dremio pour limiter l'accès aux données sensibles.

**Optimisation des performances**

- Dimensionnez les réflexions Dremio en fonction des patrons de requêtes observés.
- Configurez des politiques de rafraîchissement adaptées à la fraîcheur requise des données.
- Utilisez le partitionnement Iceberg aligné avec les filtres fréquents des tableaux de bord.

**Gouvernance des données**

- Documentez les définitions des métriques dans le catalogue Dremio.
- Établissez des processus de certification des jeux de données.
- Implémentez le lignage des données pour tracer l'origine des métriques.

### Bonnes pratiques pour les tableaux de bord Lakehouse

L'expérience accumulée dans les déploiements de Lakehouse révèle plusieurs bonnes pratiques pour maximiser la valeur des tableaux de bord analytiques :

**1. Conception centrée sur les cas d'utilisation**

Plutôt que de créer des tableaux de bord génériques, concentrez-vous sur des questions métier spécifiques. Un tableau de bord efficace répond à des questions précises :
- Quels segments de clients génèrent le plus de revenus par province ?
- Quelle est la tendance des paniers moyens cette semaine comparée au mois précédent ?
- Quelles catégories de produits connaissent une croissance des ventes ?

**2. Couche sémantique unifiée**

Créez des vues virtuelles dans Dremio qui encapsulent la logique métier complexe. Ces vues servent de « contrat » entre les ingénieurs de données et les analystes :

```sql
-- Vue sémantique pour l'analyse des ventes
CREATE VDS lakehouse.semantique.fait_ventes AS
SELECT
    t.transaction_id,
    t.date_transaction,
    DATE_TRUNC('month', t.date_transaction) AS mois,
    DATE_TRUNC('week', t.date_transaction) AS semaine,
    t.montant,
    t.categorie,
    -- Dimensions enrichies
    c.segment AS segment_client,
    c.province,
    c.valeur_vie AS valeur_vie_client,
    -- Métriques dérivées
    CASE 
        WHEN t.montant < 50 THEN 'Petit panier'
        WHEN t.montant < 150 THEN 'Panier moyen'
        ELSE 'Grand panier'
    END AS tranche_panier
FROM lakehouse.ecommerce.transactions t
JOIN lakehouse.ecommerce.clients c ON t.client_id = c.client_id
WHERE t.statut = 'Complétée';
```

**3. Stratégie de rafraîchissement différenciée**

Tous les tableaux de bord n'ont pas les mêmes exigences de fraîcheur :

| Type de tableau de bord | Fraîcheur requise | Stratégie |
|-------------------------|-------------------|-----------|
| Opérationnel (ventes du jour) | Quasi temps réel | DirectQuery + Réflexion horaire |
| Tactique (performance hebdomadaire) | Quotidienne | Import nocturne |
| Stratégique (tendances trimestrielles) | Hebdomadaire | Import planifié |

**4. Gestion des agrégations pré-calculées**

Pour les tableaux de bord à forte charge, créez des tables agrégées dans Iceberg :

```python
# Créer une table d'agrégation quotidienne
spark.sql("""
    CREATE TABLE IF NOT EXISTS nessie.ecommerce.agg_ventes_quotidiennes
    USING iceberg
    AS
    SELECT
        DATE(date_transaction) AS date_vente,
        categorie,
        c.province,
        c.segment,
        COUNT(*) AS nombre_transactions,
        SUM(montant) AS montant_total,
        AVG(montant) AS panier_moyen
    FROM nessie.ecommerce.transactions t
    JOIN nessie.ecommerce.clients c ON t.client_id = c.client_id
    WHERE t.statut = 'Complétée'
    GROUP BY DATE(date_transaction), categorie, c.province, c.segment
""")
```

Cette table peut ensuite être rafraîchie incrémentalement, offrant des performances de requête optimales pour les tableaux de bord.

**5. Monitoring et alertes**

Implémentez un suivi des métriques clés pour détecter les anomalies :

```sql
-- Requête de monitoring des ventes quotidiennes
SELECT 
    DATE(date_transaction) AS date_vente,
    COUNT(*) AS nb_transactions,
    SUM(montant) AS total_ventes,
    -- Comparaison avec la moyenne mobile sur 7 jours
    AVG(SUM(montant)) OVER (
        ORDER BY DATE(date_transaction) 
        ROWS BETWEEN 7 PRECEDING AND 1 PRECEDING
    ) AS moyenne_7j,
    -- Écart en pourcentage
    (SUM(montant) - AVG(SUM(montant)) OVER (
        ORDER BY DATE(date_transaction) 
        ROWS BETWEEN 7 PRECEDING AND 1 PRECEDING
    )) / NULLIF(AVG(SUM(montant)) OVER (
        ORDER BY DATE(date_transaction) 
        ROWS BETWEEN 7 PRECEDING AND 1 PRECEDING
    ), 0) * 100 AS ecart_pct
FROM lakehouse.ecommerce.transactions
WHERE statut = 'Complétée'
GROUP BY DATE(date_transaction)
ORDER BY date_vente DESC
LIMIT 30;
```

> **Étude de cas : Couche-Tard – Analyse multi-magasins**  
> *Secteur* : Commerce de détail  
> *Défi* : Consolider les données de milliers de points de vente à travers l'Amérique du Nord pour des analyses de performance en temps quasi réel.  
> *Solution* : Architecture Lakehouse avec Iceberg pour le stockage historique, intégration Kafka pour l'ingestion en continu (détaillée au Volume III), et Dremio pour la couche sémantique unifiée alimentant des tableaux de bord Power BI.  
> *Résultats* : Réduction du délai de disponibilité des données de T+1 jour à moins de 15 minutes. Capacité d'analyse ad hoc sur 5 ans d'historique transactionnel sans impact sur les systèmes opérationnels.

---

## IV.3.5 Résumé

Ce chapitre a concrétisé les concepts théoriques des chapitres précédents à travers une implémentation pratique complète. Nous avons parcouru l'ensemble du cycle de vie des données dans un Data Lakehouse moderne, de la configuration de l'infrastructure jusqu'à la création de visualisations analytiques.

### Compétences acquises

À l'issue de ce chapitre, vous maîtrisez :

**Configuration d'environnement**
- Déploiement d'un écosystème Lakehouse conteneurisé avec Docker Compose
- Configuration de MinIO comme stockage objet compatible S3
- Intégration du catalogue Nessie avec versionnement des données
- Orchestration des services Spark, Dremio et outils de visualisation

**Manipulation de tables Iceberg avec Spark**
- Création d'espaces de noms et de tables avec partitionnement masqué
- Insertion et mise à jour de données via les API DataFrame et SQL
- Exploration des snapshots et utilisation du voyage dans le temps
- Évolution de schéma sans interruption de service

**Fédération et accélération avec Dremio**
- Configuration des connexions aux catalogues Iceberg
- Création de vues virtuelles pour l'abstraction sémantique
- Mise en place des réflexions de données pour l'accélération
- Interrogation des métadonnées Iceberg via SQL étendu

**Visualisation et intelligence d'affaires**
- Connexion d'Apache Superset, Power BI et Tableau à Dremio
- Création de tableaux de bord interactifs
- Choix entre les modes DirectQuery et Import
- Considérations de sécurité et de gouvernance

### Architecture mise en œuvre

```
┌─────────────────────────────────────────────────────────────────┐
│                    COUCHE DE CONSOMMATION                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │  Superset   │  │  Power BI   │  │   Tableau   │              │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘              │
└─────────┼────────────────┼────────────────┼─────────────────────┘
          │                │                │
          └────────────────┼────────────────┘
                           │ JDBC/ODBC/Flight
┌──────────────────────────┼──────────────────────────────────────┐
│                    COUCHE DE FÉDÉRATION                         │
│                    ┌─────┴─────┐                                │
│                    │   Dremio  │                                │
│                    │ (Réflexions, Vues virtuelles)              │
│                    └─────┬─────┘                                │
└──────────────────────────┼──────────────────────────────────────┘
                           │ REST API
┌──────────────────────────┼──────────────────────────────────────┐
│                    COUCHE DE CATALOGUE                          │
│                    ┌─────┴─────┐                                │
│                    │   Nessie  │                                │
│                    │ (Versionnement, Branches)                  │
│                    └─────┬─────┘                                │
└──────────────────────────┼──────────────────────────────────────┘
                           │
┌──────────────────────────┼──────────────────────────────────────┐
│                    COUCHE DE TRAITEMENT                         │
│                    ┌─────┴─────┐                                │
│                    │   Spark   │                                │
│                    │ (Iceberg Runtime)                          │
│                    └─────┬─────┘                                │
└──────────────────────────┼──────────────────────────────────────┘
                           │ S3 API
┌──────────────────────────┼──────────────────────────────────────┐
│                    COUCHE DE STOCKAGE                           │
│                    ┌─────┴─────┐                                │
│                    │   MinIO   │                                │
│                    │ (Données Parquet + Métadonnées Iceberg)    │
│                    └───────────┘                                │
└─────────────────────────────────────────────────────────────────┘
```

### Points clés à retenir

1. **L'écosystème Lakehouse est modulaire** : Chaque composant (stockage, catalogue, traitement, fédération, consommation) peut être remplacé indépendamment grâce aux standards ouverts comme la spécification Apache Iceberg.

2. **Le partitionnement masqué simplifie l'expérience utilisateur** : Les requêtes filtrent sur les colonnes métier (`date_inscription`) sans exposer la structure technique de partitionnement.

3. **Le voyage dans le temps offre une traçabilité complète** : Chaque modification crée un snapshot, permettant l'audit, la récupération et la comparaison des états historiques.

4. **L'évolution de schéma est une opération métadonnées** : Ajouter, renommer ou modifier des colonnes ne requiert pas de réécriture des données existantes.

5. **Dremio accélère sans dupliquer** : Les réflexions de données créent des matérialisations intelligentes sur le Lakehouse lui-même, évitant la prolifération des copies de données.

6. **La fédération de données élimine les silos** : Dremio permet de joindre des données Iceberg avec d'autres sources sans déplacement physique, simplifiant l'architecture globale.

7. **Le versionnement Nessie apporte les pratiques DevOps aux données** : Les branches permettent d'expérimenter sans risque et de valider les transformations avant déploiement.

8. **Les outils BI s'intègrent nativement** : Power BI, Tableau et Superset se connectent à Dremio via des protocoles standards (ODBC, JDBC, Flight), bénéficiant des optimisations du Lakehouse.

### Commandes et syntaxes essentielles

Voici un aide-mémoire des commandes clés utilisées dans ce chapitre :

**Gestion des espaces de noms et tables (Spark SQL)**

```sql
-- Espaces de noms
CREATE NAMESPACE nessie.mon_namespace;
SHOW NAMESPACES IN nessie;
DROP NAMESPACE nessie.mon_namespace;

-- Tables
CREATE TABLE nessie.ns.table (...) USING iceberg PARTITIONED BY (...);
DESCRIBE EXTENDED nessie.ns.table;
ALTER TABLE nessie.ns.table ADD COLUMN nouvelle_col STRING;
DROP TABLE nessie.ns.table;
```

**Opérations de données**

```sql
-- Insertions
INSERT INTO nessie.ns.table VALUES (...);
df.writeTo("nessie.ns.table").append()

-- Mises à jour
UPDATE nessie.ns.table SET col = val WHERE condition;
DELETE FROM nessie.ns.table WHERE condition;
MERGE INTO cible USING source ON condition WHEN MATCHED THEN ... WHEN NOT MATCHED THEN ...;
```

**Voyage dans le temps**

```sql
-- Par identifiant de snapshot
SELECT * FROM table VERSION AS OF 1234567890;

-- Par horodatage
SELECT * FROM table TIMESTAMP AS OF '2024-04-01 12:00:00';

-- Tables de métadonnées
SELECT * FROM table.snapshots;
SELECT * FROM table.history;
SELECT * FROM table.files;
SELECT * FROM table.manifests;
```

**Versionnement Nessie**

```sql
CREATE BRANCH nom_branche IN nessie;
USE REFERENCE nom_branche IN nessie;
MERGE BRANCH nom_branche INTO main IN nessie;
DROP BRANCH nom_branche IN nessie;
```

### Prochaines étapes recommandées

Pour approfondir les compétences acquises dans ce chapitre, voici des pistes d'exploration :

**Expérimentation locale**
- Augmentez le volume de données générées pour observer le comportement d'Iceberg à plus grande échelle
- Testez différentes stratégies de partitionnement et comparez les performances de requête
- Explorez les opérations de maintenance (compaction, expiration des snapshots) couvertes au Chapitre IV.10

**Intégration Kafka**
- Configurez un connecteur Kafka vers Iceberg pour l'ingestion en continu (référence : Volume III)
- Implémentez un pipeline de streaming lakehouse bout en bout

**Exploration avancée de Dremio**
- Testez les capacités de fédération avec une base de données PostgreSQL locale
- Expérimentez avec les différents types de réflexions (raw, aggregation)
- Explorez l'API REST de Dremio pour l'automatisation

### Comparaison des approches d'interrogation

Le tableau suivant résume les caractéristiques des différentes méthodes d'accès aux données Iceberg :

| Approche | Cas d'utilisation | Avantages | Limites |
|----------|-------------------|-----------|---------|
| Spark SQL direct | ETL, transformations complexes | Contrôle total, toutes les fonctionnalités Iceberg | Nécessite compétences Spark |
| Dremio SQL | Analyse ad hoc, exploration | Interface simple, accélération automatique | Dépendance à Dremio |
| BI via Dremio | Tableaux de bord, rapports | Accessible aux analystes métier | Personnalisation limitée |
| API Iceberg (Python/Java) | Automatisation, scripts | Flexibilité maximale | Courbe d'apprentissage |

### Transition vers la Partie 2

Ce chapitre conclut la Partie 1 « La Valeur du Lakehouse ». Vous disposez maintenant d'une compréhension solide des concepts fondamentaux et d'une expérience pratique avec les outils clés de l'écosystème.

La Partie 2 « Concevoir Votre Architecture » vous guidera dans la conception d'une architecture Lakehouse de production. Nous aborderons méthodiquement chaque couche : stockage, ingestion, catalogue, fédération et consommation. Le Chapitre IV.4 débutera par l'audit de votre plateforme existante et la définition des exigences architecturales, illustrés par l'étude de cas détaillée de la Banque Hamerliwa.

### Nettoyage de l'environnement de développement

Lorsque vous avez terminé vos expérimentations, vous pouvez nettoyer l'environnement Docker de plusieurs façons selon vos besoins :

**Conservation des données pour une session ultérieure** :

```bash
# Arrêter les conteneurs sans supprimer les volumes
docker-compose stop
```

**Redémarrage avec les données préservées** :

```bash
docker-compose start
```

**Suppression complète (conteneurs et volumes)** :

```bash
# Arrêter et supprimer les conteneurs
docker-compose down

# Supprimer également les volumes (perte des données)
docker-compose down -v

# Nettoyer les images téléchargées (optionnel)
docker image prune -a
```

Cette gestion flexible de l'environnement illustre l'un des avantages de la conteneurisation : la capacité de créer, détruire et recréer des environnements complets de manière reproductible.

> **Migration**  
> *De* : Environnement de développement local (Docker Compose)  
> *Vers* : Architecture de production infonuagique ou hybride  
> *Stratégie* : Les chapitres de la Partie 2 couvrent la sélection des composants, le dimensionnement, la haute disponibilité et les considérations de coûts pour chaque couche de l'architecture. L'approche privilégie une migration progressive, couche par couche, minimisant les risques et permettant une validation incrémentale. Les patrons architecturaux établis dans ce chapitre — séparation stockage/calcul, catalogue centralisé, couche de fédération — se transposent directement vers les déploiements de production, que ce soit sur Amazon Web Services, Google Cloud Platform, Microsoft Azure ou des infrastructures sur site.

### Ressources complémentaires

Pour approfondir les sujets abordés dans ce chapitre, les ressources suivantes sont recommandées :

**Documentation officielle**
- Documentation Apache Iceberg : https://iceberg.apache.org/docs/
- Guide Dremio : https://docs.dremio.com/
- Projet Nessie : https://projectnessie.org/

**Formations et tutoriels**
- Dremio University (formations gratuites) : https://www.dremio.com/university/
- Tutoriels Iceberg de la communauté : https://iceberg.apache.org/community/

**Livres**
- « Apache Iceberg: The Definitive Guide » (O'Reilly, 2024) — référence complète sur le format de table

---

**Références**

- Apache Iceberg Documentation (2025). *Spark Configuration*. https://iceberg.apache.org/docs/latest/spark-configuration/
- Apache Iceberg (2025). *Release Notes 1.9.x*. https://iceberg.apache.org/releases/
- Dremio (2024). *Apache Iceberg Integration Guide*. https://docs.dremio.com/
- Dremio (2024). *2024 Year in Review: Lakehouses, Apache Iceberg and Dremio*. https://www.dremio.com/blog/
- Project Nessie (2024). *Nessie and Apache Iceberg*. https://projectnessie.org/
- MinIO (2025). *The Definitive Guide to Lakehouse Architecture with Iceberg and MinIO*. https://blog.min.io/
- Apache Software Foundation (2025). *Apache Superset Documentation*. https://superset.apache.org/docs/
