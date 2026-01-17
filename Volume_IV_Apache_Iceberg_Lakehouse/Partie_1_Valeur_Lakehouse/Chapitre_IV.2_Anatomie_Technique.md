# Chapitre IV.2 — Anatomie Technique d'Apache Iceberg

## Introduction

La compréhension profonde de l'architecture interne d'Apache Iceberg constitue le fondement de toute implémentation réussie du Data Lakehouse. Alors que le chapitre précédent présentait la valeur stratégique et le positionnement d'Iceberg dans l'écosystème moderne des données, ce chapitre plonge dans les mécanismes techniques qui permettent au format de table de délivrer ses promesses de performance, de fiabilité et d'évolutivité.

Apache Iceberg se distingue des formats de table traditionnels par son architecture métadonnées sophistiquée qui découple complètement la représentation logique des tables de leur disposition physique. Cette séparation fondamentale permet des fonctionnalités qui étaient auparavant impossibles dans les lacs de données : transactions ACID véritables, évolution de schéma sans réécriture, partitionnement masqué et voyage dans le temps. Pour l'architecte de données et l'ingénieur, maîtriser ces mécanismes internes est essentiel pour concevoir des systèmes optimaux et résoudre les problèmes en production.

Dans les architectures de données traditionnelles basées sur Apache Hive, la structure des répertoires servait de métadonnées implicites. Pour déterminer quels fichiers appartiennent à une table, le moteur de requête devait parcourir l'arborescence de fichiers, listant potentiellement des milliers de répertoires de partition et des millions de fichiers de données. Cette approche, bien que simple à comprendre, présente des limitations sévères à grande échelle : temps de planification des requêtes croissant linéairement avec le nombre de partitions, absence de garanties transactionnelles lors des écritures concurrentes, et impossibilité de modifier le schéma de partitionnement sans migration complète des données.

Iceberg résout ces limitations fondamentales en introduisant une couche de métadonnées explicites qui maintient un inventaire complet et versionné de tous les fichiers appartenant à une table. Cette approche de type "manifest" transforme les opérations de planification de requête de complexité O(n) — où n représente le nombre de fichiers ou partitions — en opérations de complexité O(1) en termes d'appels au système de fichiers. Le gain de performance devient spectaculaire à l'échelle : une table de millions de fichiers peut être planifiée aussi rapidement qu'une table de quelques fichiers.

Au-delà de la performance, l'architecture de métadonnées d'Iceberg permet des garanties impossibles avec les approches basées sur le système de fichiers. Les transactions ACID sont implémentées via un protocole de commit optimiste qui garantit l'atomicité des modifications. L'isolation snapshot assure que les lecteurs voient toujours un état cohérent de la table, même pendant les écritures concurrentes. L'historique des snapshots permet le voyage dans le temps et le rollback vers des états antérieurs.

Ce chapitre décortique l'anatomie complète d'une table Iceberg, depuis la couche de catalogue jusqu'aux fichiers de données, en passant par la hiérarchie de métadonnées qui constitue le cœur de l'innovation. Nous explorerons les structures de données détaillées, les algorithmes de commit, les stratégies de mise à jour au niveau des lignes et les mécanismes d'optimisation des requêtes. L'objectif est de fournir aux praticiens une compréhension suffisamment profonde pour prendre des décisions architecturales éclairées et diagnostiquer efficacement les problèmes de performance.

La maîtrise de ces concepts est particulièrement pertinente dans le contexte canadien où les réglementations comme la Loi 25 au Québec et la LPRPDE fédérale imposent des exigences strictes de traçabilité et de gouvernance des données. Les fonctionnalités natives d'Iceberg — voyage dans le temps pour l'audit, suppression au niveau des lignes pour le droit à l'oubli, lignage via les métadonnées — répondent directement à ces exigences réglementaires.

## L'Architecture en Trois Couches

Apache Iceberg organise une table selon une architecture en trois couches distinctes, chacune servant un rôle spécifique dans la gestion et l'accès aux données. Cette stratification permet la séparation des préoccupations et offre la flexibilité nécessaire pour supporter différents moteurs de calcul et systèmes de stockage.

### Vue d'Ensemble de l'Architecture

La première couche, le catalogue, agit comme point d'entrée et source de vérité pour localiser les tables. La deuxième couche, les métadonnées, contient toute l'information nécessaire pour comprendre l'état d'une table à n'importe quel moment. La troisième couche, les données, stocke les enregistrements réels dans des formats de fichiers optimisés.

Cette architecture se différencie fondamentalement de l'approche traditionnelle de type Hive où la structure des répertoires servait de métadonnées implicites. Dans le modèle Hive, pour déterminer quels fichiers appartiennent à une table, le moteur de requête devait effectuer des opérations coûteuses de listage de répertoires, parcourant potentiellement des milliers de partitions et des millions de fichiers. Iceberg remplace cette approche par un système de manifestes explicites qui énumère précisément chaque fichier appartenant à la table.

### Flux de Lecture d'une Table

Lorsqu'un moteur de requête accède à une table Iceberg, le flux d'opérations suit une séquence bien définie. Premièrement, le moteur interroge le catalogue pour obtenir l'emplacement du fichier de métadonnées courant. Deuxièmement, il lit ce fichier metadata.json pour identifier le snapshot actif. Troisièmement, il accède à la liste de manifestes de ce snapshot. Quatrièmement, il consulte les fichiers manifestes pour déterminer quels fichiers de données doivent être lus. Finalement, il lit uniquement les fichiers de données nécessaires.

Cette approche transforme une opération de complexité O(n) — où n représente le nombre de fichiers ou partitions — en une opération de complexité O(1) en termes d'appels distants pour la planification des requêtes. Le gain de performance devient spectaculaire à grande échelle : une table de millions de fichiers peut être planifiée aussi rapidement qu'une table de quelques fichiers.

## La Couche de Catalogue

Le catalogue Iceberg remplit deux fonctions essentielles : le suivi des tables par nom et la gestion atomique des pointeurs vers les métadonnées courantes. Sans catalogue, chaque moteur de requête devrait connaître l'emplacement exact des fichiers de métadonnées, rendant impossible la coordination entre différents systèmes.

### Rôle et Responsabilités

Le catalogue maintient une correspondance entre les noms de tables et les emplacements des fichiers de métadonnées. Lorsqu'une table est mise à jour, le catalogue doit atomiquement basculer le pointeur vers le nouveau fichier de métadonnées. Cette atomicité est cruciale : elle garantit que tous les lecteurs voient soit l'ancien état complet, soit le nouveau état complet, jamais un état intermédiaire incohérent.

Les responsabilités du catalogue incluent la création et la suppression de tables, le renommage de tables entre espaces de noms, la gestion des espaces de noms eux-mêmes, et le support des opérations de commit avec détection de conflits. Le catalogue ne stocke pas les métadonnées de la table elle-même — celles-ci résident dans les fichiers metadata.json — mais uniquement le pointeur vers ces métadonnées.

### Types de Catalogues

Iceberg supporte plusieurs implémentations de catalogues, classées en deux catégories : les catalogues basés sur fichiers et les catalogues basés sur services.

Les catalogues basés sur fichiers utilisent un fichier version-hint.txt pour pointer vers le fichier metadata.json courant. Le catalogue Hadoop en est l'exemple canonique. Cette approche est simple mais présente des limitations pour les environnements à haute concurrence, car les systèmes de fichiers distribués ne garantissent pas tous l'atomicité des opérations de renommage nécessaires aux commits.

Les catalogues basés sur services maintiennent les références dans une base de données ou un service externe. Le Hive Metastore, AWS Glue Data Catalog, et les implémentations REST en sont des exemples. Ces catalogues offrent de meilleures garanties de concurrence et des fonctionnalités avancées comme le contrôle d'accès et l'audit.

### La Spécification REST Catalog

La communauté Iceberg a développé une spécification REST Catalog pour standardiser les interactions avec n'importe quel catalogue. Cette spécification OpenAPI définit les points de terminaison requis pour les opérations sur les tables et les espaces de noms.

Les avantages du REST Catalog sont considérables. Un client unique peut interagir avec n'importe quel catalogue implémentant la spécification, éliminant le besoin d'implémentations spécifiques pour chaque langage et chaque catalogue. Cette interopérabilité simplifie l'adoption multi-moteurs et facilite les migrations entre fournisseurs.

Les points de terminaison principaux de la spécification incluent GET /v1/config pour la configuration du catalogue, POST /v1/namespaces pour la création d'espaces de noms, POST /v1/namespaces/{namespace}/tables pour la création de tables, et POST /v1/namespaces/{namespace}/tables/{table} pour les commits de métadonnées.

Des implémentations de catalogues REST comme Apache Polaris (en incubation), Nessie, et les services gérés d'AWS Glue et Snowflake adoptent cette spécification. La convergence vers ce standard renforce l'interopérabilité de l'écosystème Iceberg.

> **Performance**  
> L'utilisation d'un catalogue REST avec mise en cache côté serveur peut réduire la latence de planification des requêtes de 40 à 60 % par rapport aux catalogues basés sur fichiers, particulièrement pour les tables à haute fréquence d'accès.

## La Couche de Métadonnées

La couche de métadonnées constitue le cœur de l'innovation d'Iceberg. Elle transforme une collection de fichiers en une table cohérente avec des propriétés ACID, un historique de versions et des statistiques pour l'optimisation des requêtes. Cette couche comprend quatre types de fichiers : le fichier de métadonnées principal (metadata.json), les listes de manifestes, les fichiers manifestes et les fichiers Puffin pour les statistiques avancées.

### Le Fichier metadata.json

Le fichier metadata.json est le point d'entrée pour comprendre l'état d'une table. Il contient toute l'information nécessaire pour reconstruire la table à un moment donné, incluant l'historique des schémas, les spécifications de partitionnement, l'historique des snapshots et les propriétés de la table.

La structure d'un fichier metadata.json typique comprend les champs suivants :

```json
{
  "format-version": 2,
  "table-uuid": "f7m3-a812-4a5c-96b6-8a3a",
  "location": "s3://warehouse/db/events",
  "last-updated-ms": 1664472000000,
  "last-column-id": 5,
  "schemas": [...],
  "current-schema-id": 0,
  "partition-specs": [...],
  "default-spec-id": 0,
  "sort-orders": [...],
  "default-sort-order-id": 0,
  "snapshots": [...],
  "current-snapshot-id": 8235603094578364387,
  "snapshot-log": [...],
  "metadata-log": [...]
}
```

Le champ format-version indique la version de la spécification Iceberg. La version 1 supporte les tables analytiques de base, la version 2 ajoute les opérations au niveau des lignes avec les fichiers de suppression, et la version 3 (adoptée en 2024-2025) introduit les vecteurs de suppression, les nouveaux types de données comme variant et les types géospatiaux.

Le champ table-uuid identifie uniquement la table à travers les opérations. Si un client détecte un UUID différent après rafraîchissement, cela indique une corruption ou un conflit nécessitant une investigation.

Le champ location spécifie l'emplacement de base où sont stockés les fichiers de données, les manifestes et les métadonnées. Les écrivains utilisent cet emplacement pour déterminer où placer les nouveaux fichiers.

L'historique des schémas dans le tableau schemas préserve chaque version de schéma avec un identifiant unique. Cette préservation permet aux lecteurs d'interpréter correctement les fichiers de données écrits avec d'anciens schémas.

### Les Snapshots

Un snapshot représente l'état complet d'une table à un moment précis. Chaque opération d'écriture — insertion, mise à jour, suppression — crée un nouveau snapshot. Cette immuabilité des snapshots est fondamentale pour les garanties ACID et le voyage dans le temps.

La structure d'un snapshot comprend :

```json
{
  "snapshot-id": 8235603094578364387,
  "parent-snapshot-id": 3051729675574644887,
  "timestamp-ms": 1664472000000,
  "summary": {
    "operation": "append",
    "added-data-files": "10",
    "added-records": "50000",
    "total-data-files": "150",
    "total-records": "750000"
  },
  "manifest-list": "s3://warehouse/db/events/metadata/snap-8235603.avro",
  "schema-id": 0
}
```

Le champ parent-snapshot-id établit la lignée des snapshots, permettant de reconstituer l'historique complet des modifications. Le champ summary fournit des statistiques sur l'opération effectuée, utiles pour le monitoring et le débogage.

Chaque snapshot référence une liste de manifestes qui décrit l'ensemble exact des fichiers appartenant à la table à ce moment. Cette référence est le mécanisme qui permet l'isolation des lectures : un lecteur utilise le snapshot qui était courant au moment où il a chargé les métadonnées et n'est pas affecté par les écritures concurrentes.

### Les Listes de Manifestes

La liste de manifestes (manifest list) est un fichier Avro qui énumère tous les fichiers manifestes composant un snapshot. Elle contient également des statistiques agrégées sur chaque manifeste, permettant un premier niveau d'élagage lors de la planification des requêtes.

Chaque entrée de la liste de manifestes inclut :

```json
{
  "manifest_path": "s3://warehouse/db/events/metadata/d2f5ebe2.avro",
  "manifest_length": 7292,
  "partition_spec_id": 0,
  "added_snapshot_id": 8235603094578364387,
  "added_data_files_count": 10,
  "existing_data_files_count": 140,
  "deleted_data_files_count": 0,
  "added_rows_count": 50000,
  "existing_rows_count": 700000,
  "deleted_rows_count": 0,
  "partitions": [
    {
      "contains_null": false,
      "contains_nan": false,
      "lower_bound": "2024-01-01",
      "upper_bound": "2024-01-31"
    }
  ]
}
```

Les statistiques de partition (partitions) permettent d'éliminer des manifestes entiers sans les lire. Si une requête filtre sur une date hors de la plage d'un manifeste, ce manifeste peut être ignoré complètement. Ce premier niveau d'élagage réduit considérablement le volume de métadonnées à lire pour les requêtes sélectives.

### Les Fichiers Manifestes

Les fichiers manifestes sont le niveau le plus granulaire de la hiérarchie de métadonnées. Chaque manifeste est un fichier Avro qui liste un sous-ensemble des fichiers de données de la table, avec des statistiques détaillées pour chaque fichier.

La structure d'une entrée de manifeste comprend :

```json
{
  "status": 1,
  "snapshot_id": 8235603094578364387,
  "data_file": {
    "file_path": "s3://warehouse/db/events/data/part-00001.parquet",
    "file_format": "PARQUET",
    "partition": {"event_date": "2024-01-15"},
    "record_count": 5000,
    "file_size_in_bytes": 15728640,
    "column_sizes": {1: 1048576, 2: 524288, 3: 2097152},
    "value_counts": {1: 5000, 2: 5000, 3: 4850},
    "null_value_counts": {1: 0, 2: 0, 3: 150},
    "nan_value_counts": {},
    "lower_bounds": {1: "event_001", 2: 100, 3: 1.5},
    "upper_bounds": {1: "event_999", 2: 9999, 3: 99.9}
  }
}
```

Le champ status indique l'état du fichier dans ce manifeste : 0 pour existant, 1 pour ajouté, 2 pour supprimé. Cette information est cruciale pour le suivi des modifications entre snapshots.

Les statistiques au niveau des colonnes — lower_bounds et upper_bounds — permettent l'élagage de fichiers (data skipping). Si une requête recherche des événements avec un identifiant supérieur à "event_999", ce fichier peut être ignoré car sa borne supérieure est exactement "event_999".

Les comptages de valeurs null et NaN aident également l'optimiseur. Une colonne avec 100 % de valeurs null n'a pas besoin d'être lue si la requête filtre sur des valeurs non-null.

> **Performance**  
> Les statistiques de manifeste permettent typiquement d'éliminer 70 à 95 % des fichiers pour les requêtes sélectives, transformant des scans de téraoctets en lectures de gigaoctets.

## La Couche de Données

La couche de données stocke les enregistrements réels dans des formats de fichiers optimisés pour l'analytique. Iceberg est agnostique au format de fichier et supporte Apache Parquet, Apache ORC et Apache Avro.

### Formats de Fichiers Supportés

Apache Parquet est le format par défaut et le plus largement utilisé. Son organisation en colonnes, sa compression efficace et ses statistiques de pied de page en font un choix optimal pour les charges analytiques. Parquet supporte des schémas complexes avec des structures imbriquées, des listes et des maps.

La structure interne de Parquet est particulièrement bien adaptée à Iceberg. Chaque fichier Parquet est organisé en groupes de lignes (row groups), typiquement de 128 Mo, qui peuvent être lus indépendamment. Chaque groupe de lignes contient des métadonnées incluant les statistiques min/max par colonne, permettant un élagage supplémentaire au-delà des statistiques de manifeste Iceberg.

Les colonnes dans Parquet sont stockées consécutivement, permettant de ne lire que les colonnes nécessaires à une requête (projection pushdown). La compression est appliquée par colonne avec des codecs adaptés aux types de données : dictionnaire pour les chaînes répétitives, delta encoding pour les entiers séquentiels, run-length encoding pour les valeurs répétées.

Apache ORC offre des caractéristiques similaires avec une compression souvent supérieure et des index intégrés plus sophistiqués. ORC maintient des index de bloom filter et des index de position qui peuvent accélérer les recherches ponctuelles. Il est particulièrement populaire dans l'écosystème Hive et pour les charges avec des patterns d'accès prédictibles.

Apache Avro, bien que principalement orienté lignes, est utilisé pour les fichiers de métadonnées (manifestes et listes de manifestes) et peut servir pour les données dans certains cas d'usage nécessitant une évolution de schéma flexible ou une sérialisation compacte pour le streaming.

### Configuration des Formats de Fichiers

Les propriétés de table permettent de configurer finement le comportement d'écriture :

```sql
-- Configuration Parquet avancée
ALTER TABLE events SET TBLPROPERTIES (
  'write.format.default' = 'parquet',
  'write.parquet.compression-codec' = 'zstd',
  'write.parquet.compression-level' = '3',
  'write.parquet.dict-size-bytes' = '2097152',
  'write.parquet.page-size-bytes' = '1048576',
  'write.parquet.row-group-size-bytes' = '134217728',
  'write.parquet.bloom-filter-enabled.column.user_id' = 'true'
);

-- Configuration ORC
ALTER TABLE events SET TBLPROPERTIES (
  'write.format.default' = 'orc',
  'write.orc.compression-codec' = 'zstd',
  'write.orc.stripe-size-bytes' = '67108864',
  'write.orc.bloom.filter.columns' = 'user_id,product_id'
);
```

### Immutabilité des Fichiers

Un principe fondamental d'Iceberg est l'immutabilité des fichiers. Une fois écrit, un fichier de données n'est jamais modifié. Les mises à jour et suppressions sont gérées soit par réécriture complète du fichier (Copy-on-Write), soit par des fichiers de suppression séparés (Merge-on-Read).

Cette immutabilité simplifie considérablement la gestion de la concurrence et la récupération après erreur. Il n'y a jamais de fichiers partiellement écrits ou corrompus par des modifications concurrentes. Les lecteurs peuvent accéder aux fichiers en toute sécurité sans verrous.

L'immutabilité facilite également le caching. Un fichier une fois lu peut être mis en cache indéfiniment car son contenu ne changera jamais. Les systèmes de stockage peuvent optimiser la réplication et la distribution en sachant que les fichiers sont stables.

### Organisation Physique

Contrairement aux tables Hive traditionnelles, Iceberg ne dépend pas de la structure des répertoires pour organiser les données. Les fichiers d'une même partition peuvent résider dans différents répertoires ou préfixes. Cette flexibilité permet d'éviter les goulots d'étranglement liés à la limitation de débit des systèmes de stockage objet qui peuvent throttler les requêtes sur un préfixe unique.

En pratique, de nombreuses implémentations organisent tout de même les fichiers par partition pour faciliter la navigation manuelle et le débogage, mais cette organisation n'est pas requise par la spécification. Certaines architectures utilisent intentionnellement une distribution aléatoire des fichiers pour maximiser le parallélisme de lecture sur les systèmes de stockage objet.

### Tri des Données

Iceberg supporte le tri des données au sein des fichiers pour améliorer l'efficacité des requêtes. Un ordre de tri (sort order) peut être défini sur la table :

```sql
-- Définition d'un ordre de tri
ALTER TABLE events WRITE ORDERED BY event_date DESC, event_type ASC;

-- Réécriture avec tri pour optimiser les lectures
CALL catalog.system.rewrite_data_files(
  table => 'db.events',
  strategy => 'sort',
  sort_order => 'event_date DESC, event_type ASC'
);
```

Les données triées améliorent l'efficacité de la compression (valeurs similaires groupées) et permettent un élagage plus agressif via les statistiques min/max. Pour les requêtes qui filtrent sur les colonnes de tri, le bénéfice peut être substantiel.

## Gestion des Schémas

L'évolution de schéma est l'une des fonctionnalités les plus puissantes d'Iceberg. Elle permet de modifier la structure d'une table — ajouter, supprimer, renommer ou réordonner des colonnes — sans réécrire les données existantes.

### Le Système d'Identifiants de Colonnes

Le mécanisme clé qui permet l'évolution de schéma sans effets de bord est l'utilisation d'identifiants uniques pour chaque colonne. Contrairement aux systèmes qui identifient les colonnes par leur position ou leur nom, Iceberg assigne un entier unique permanent à chaque champ lors de sa création.

Considérons un fichier écrit avec le schéma suivant : 1: a int, 2: b string, 3: c double. Si le schéma évolue pour devenir 3: measurement double, 2: name string, 4: d boolean, la projection de lecture reste correcte. La colonne c (ID 3) est maintenant appelée measurement, la colonne b (ID 2) est renommée name, et la nouvelle colonne d (ID 4) retourne null pour les anciens fichiers.

Ce système garantit que :
- L'ajout d'une colonne ne lit jamais les valeurs d'une autre colonne existante
- La suppression d'une colonne ne modifie pas les valeurs des autres colonnes
- Le renommage d'une colonne n'affecte pas les données sous-jacentes
- La réorganisation des colonnes ne change pas les valeurs associées

### Opérations d'Évolution de Schéma

Iceberg supporte les opérations suivantes sur les schémas :

**Add** — Ajoute une nouvelle colonne à la table ou à une structure imbriquée. La colonne reçoit un nouvel identifiant unique. Les fichiers existants retournent null ou la valeur par défaut pour cette colonne.

**Drop** — Supprime une colonne existante. Les fichiers existants conservent les données de cette colonne mais elles ne sont plus accessibles. La suppression est une opération de métadonnées uniquement.

**Rename** — Renomme une colonne ou un champ imbriqué. L'identifiant reste le même, seul le nom change dans le schéma.

**Reorder** — Modifie l'ordre des colonnes. Comme les colonnes sont identifiées par leur ID, l'ordre n'affecte pas la lecture des fichiers existants.

**Update** — Élargit le type d'une colonne. Les promotions de type supportées incluent :
- int vers long
- float vers double
- decimal vers decimal avec précision plus large

```sql
-- Exemples d'évolution de schéma en SQL
ALTER TABLE events ADD COLUMN user_agent STRING;
ALTER TABLE events DROP COLUMN legacy_field;
ALTER TABLE events RENAME COLUMN old_name TO new_name;
ALTER TABLE events ALTER COLUMN amount TYPE decimal(12,2);
```

### Valeurs par Défaut

La spécification Iceberg v3 introduit un support amélioré pour les valeurs par défaut. Deux types de défauts existent : initial-default et write-default.

Le initial-default est utilisé lors de la lecture de fichiers écrits avant l'ajout de la colonne. Une fois défini, il ne peut pas être modifié.

Le write-default est utilisé lors de l'écriture si aucune valeur n'est fournie. Il peut être modifié au fil du temps.

Cette distinction permet des scénarios où une colonne a une valeur par défaut différente pour les données historiques et les nouvelles données. Par exemple, une colonne status pourrait avoir une valeur par défaut 'UNKNOWN' pour les données historiques (initial-default) mais 'PENDING' pour les nouvelles insertions (write-default).

### Promotions de Type Sûres

Iceberg supporte l'élargissement sécurisé de certains types sans réécriture de données. Les promotions autorisées sont conçues pour être toujours sans perte :

| Type Source | Type Cible | Notes |
|-------------|------------|-------|
| int | long | Élargissement entier standard |
| float | double | Élargissement flottant standard |
| decimal(P1, S) | decimal(P2, S) | P2 > P1, même échelle |

Les promotions non supportées incluent les conversions string vers numeric, les réductions de précision, et les changements de type incompatibles. Ces restrictions garantissent qu'aucune donnée existante ne sera mal interprétée après l'évolution du schéma.

```sql
-- Promotion de type sécurisée
ALTER TABLE events ALTER COLUMN quantity TYPE BIGINT;  -- int -> long

-- Promotion de decimal
ALTER TABLE events ALTER COLUMN amount TYPE DECIMAL(12, 2);  -- de DECIMAL(10,2)
```

### Schémas Imbriqués

L'évolution de schéma dans Iceberg s'applique également aux structures imbriquées. Les structs, arrays et maps peuvent voir leurs champs internes modifiés avec les mêmes garanties de sécurité que les colonnes de premier niveau.

```sql
-- Ajout d'un champ à un struct imbriqué
ALTER TABLE events ALTER COLUMN address ADD COLUMN country STRING;

-- Renommage d'un champ de struct
ALTER TABLE events ALTER COLUMN address.city RENAME TO city_name;
```

Chaque champ dans une structure imbriquée possède son propre identifiant unique, permettant une évolution indépendante à n'importe quel niveau de la hiérarchie.

## Gestion du Partitionnement

Le partitionnement dans Iceberg représente une innovation majeure par rapport aux systèmes traditionnels. Le concept de partitionnement masqué (hidden partitioning) libère les utilisateurs de la nécessité de comprendre et de gérer explicitement la disposition physique des données.

### Partitionnement Masqué

Dans les systèmes traditionnels comme Hive, le partitionnement est explicite. Les partitions apparaissent comme des colonnes dans le schéma et doivent être spécifiées lors des écritures et des requêtes. Cette approche génère plusieurs problèmes : les producteurs de données doivent calculer correctement les valeurs de partition, les consommateurs doivent connaître le schéma de partitionnement pour écrire des requêtes efficaces, et les erreurs de partitionnement corrompent silencieusement les données.

Iceberg résout ces problèmes en gérant le partitionnement comme une transformation de métadonnées. Les valeurs de partition sont dérivées automatiquement des colonnes source selon des transformations configurées. Les producteurs écrivent simplement les données, Iceberg calcule les partitions. Les consommateurs écrivent des requêtes naturelles, Iceberg applique automatiquement l'élagage de partition.

### Transformations de Partition

Iceberg supporte plusieurs transformations pour dériver les valeurs de partition :

**Identity** — Utilise la valeur de la colonne directement. Approprié pour les colonnes catégoriques à faible cardinalité.

**Year, Month, Day, Hour** — Extrait les composantes temporelles d'une colonne timestamp ou date. Permet de partitionner par granularité temporelle sans colonnes dérivées explicites.

**Bucket(N)** — Distribue les valeurs en N seaux par hachage. Utile pour les colonnes à haute cardinalité comme les identifiants, assurant une distribution uniforme.

**Truncate(W)** — Tronque les valeurs à une largeur fixe. Pour les chaînes, conserve les W premiers caractères. Pour les nombres, arrondit aux bins de taille W.

```sql
-- Création d'une table avec partitionnement masqué
CREATE TABLE events (
  event_id STRING,
  event_time TIMESTAMP,
  user_id BIGINT,
  event_type STRING
)
USING iceberg
PARTITIONED BY (
  day(event_time),
  bucket(16, user_id)
);

-- Requête naturelle - l'élagage s'applique automatiquement
SELECT * FROM events 
WHERE event_time BETWEEN '2024-01-01' AND '2024-01-15'
  AND user_id = 12345;
```

Dans cet exemple, le moteur de requête comprend que le filtre sur event_time implique les partitions de jours du 1er au 15 janvier, et que user_id = 12345 correspond à un seau spécifique. Les fichiers des autres partitions sont ignorés sans que l'utilisateur n'ait à spécifier ces filtres explicitement.

### Évolution du Partitionnement

L'évolution du partitionnement permet de modifier le schéma de partitionnement d'une table existante sans réécrire les données. Lorsque la spécification de partition change, les anciennes données conservent leur partitionnement original tandis que les nouvelles données utilisent le nouveau schéma.

Chaque spécification de partition reçoit un identifiant unique, et les fichiers manifestes enregistrent quelle spécification était active lors de l'écriture de chaque fichier. Lors de la planification des requêtes, Iceberg dérive les filtres appropriés pour chaque layout de partition.

```sql
-- Migration du partitionnement journalier vers horaire
ALTER TABLE events ADD PARTITION FIELD hour(event_time);

-- Les deux schémas coexistent
-- Données 2023: partitionnées par jour
-- Données 2024: partitionnées par jour ET par heure
```

Cette flexibilité est cruciale pour les tables à longue durée de vie dont les patterns d'accès évoluent. Une table initialement partitionnée par mois peut être migrée vers un partitionnement journalier quand le volume augmente, sans migration de données coûteuse.

La planification des requêtes avec plusieurs spécifications de partition fonctionne comme suit. Supposons une table initialement partitionnée par mois(event_time), puis modifiée pour être partitionnée par jour(event_time). Une requête filtrant sur event_time BETWEEN '2024-01-15' AND '2024-01-20' génère deux ensembles de filtres : pour les fichiers anciens avec la partition mois, le filtre devient event_month = '2024-01'; pour les fichiers nouveaux avec la partition jour, le filtre devient event_day IN ('2024-01-15', '2024-01-16', ..., '2024-01-20'). Chaque ensemble est appliqué aux manifestes correspondants.

### Bonnes Pratiques de Partitionnement

Le choix du schéma de partitionnement impacte significativement les performances. Plusieurs principes guident ce choix.

**Analyser les patterns de requête** — Le partitionnement doit correspondre aux filtres les plus fréquents. Si 90 % des requêtes filtrent par date, le partitionnement temporel est approprié. Si les requêtes filtrent principalement par région, le partitionnement par région est préférable.

**Équilibrer la granularité** — Trop peu de partitions (par exemple, par année pour des données quotidiennes) ne permet pas un élagage efficace. Trop de partitions (par exemple, par seconde) crée une explosion de métadonnées. Une bonne règle est de viser des partitions contenant entre 100 Mo et 1 Go de données.

**Éviter les colonnes à haute cardinalité en identity** — Partitionner par identity sur une colonne avec des millions de valeurs uniques crée autant de partitions. Utiliser plutôt bucket() ou truncate() pour regrouper les valeurs.

**Considérer les jointures** — Si deux tables sont fréquemment jointes, un partitionnement compatible peut permettre des jointures partition-à-partition plus efficaces.

> **Migration**  
> *De* : Partitionnement Hive rigide avec colonnes explicites  
> *Vers* : Partitionnement masqué Iceberg avec transformations  
> *Stratégie* : Utiliser les métadonnées de migration Iceberg pour convertir les tables Hive existantes. Les partitions Hive sont préservées et Iceberg applique automatiquement l'élagage basé sur les filtres de requête.

## Opérations au Niveau des Lignes

La version 2 de la spécification Iceberg introduit le support des opérations au niveau des lignes — UPDATE et DELETE — pour les tables analytiques composées de fichiers immuables. Deux stratégies sont disponibles : Copy-on-Write (COW) et Merge-on-Read (MOR), avec différents compromis entre performance d'écriture et de lecture.

### Copy-on-Write (COW)

Dans la stratégie Copy-on-Write, toute modification d'une ligne provoque la réécriture complète du fichier de données contenant cette ligne. Si un fichier de 1 Go contient une ligne à mettre à jour, un nouveau fichier de 1 Go est créé avec toutes les lignes sauf celle modifiée, plus la ligne mise à jour.

**Avantages de COW :**
- Performance de lecture optimale — aucune réconciliation nécessaire lors des requêtes
- Pas de fichiers de suppression à gérer
- Modèle mental simple

**Inconvénients de COW :**
- Amplification d'écriture significative pour les modifications ponctuelles
- Coût élevé pour les tables à haute fréquence de mise à jour
- Latence d'écriture proportionnelle à la taille des fichiers modifiés

COW est recommandé pour les charges avec lectures fréquentes et mises à jour peu fréquentes, ou pour les mises à jour par lots massifs où la réécriture est inévitable.

### Merge-on-Read (MOR)

La stratégie Merge-on-Read évite la réécriture immédiate en créant des fichiers de suppression qui enregistrent les lignes invalidées. Lors de la lecture, ces fichiers de suppression sont appliqués pour filtrer les lignes obsolètes des fichiers de données originaux.

Iceberg supporte deux types de fichiers de suppression :

**Position Deletes** — Enregistrent le chemin du fichier de données et la position ordinale de la ligne supprimée. La structure est simple : (file_path, row_position). Pour supprimer la ligne 42 du fichier part-00001.parquet, le fichier de position delete contient ("part-00001.parquet", 42).

```
+--------------------------------+----------+
| file_path                      | pos      |
+--------------------------------+----------+
| .../part-00001.parquet         | 42       |
| .../part-00001.parquet         | 157      |
| .../part-00003.parquet         | 891      |
+--------------------------------+----------+
```

**Equality Deletes** — Identifient les lignes à supprimer par les valeurs de certaines colonnes plutôt que par position. Par exemple, "supprimer où order_id = 12345". Cette approche est plus rapide à écrire car elle ne nécessite pas de lire les fichiers existants pour déterminer les positions.

```
+-------------------+
| order_id          |
+-------------------+
| 12345             |
| 67890             |
+-------------------+
```

Les equality deletes utilisent des numéros de séquence pour garantir que seules les lignes écrites avant la suppression sont affectées. Une ligne avec order_id = 12345 écrite après le delete equality n'est pas supprimée.

**Avantages de MOR :**
- Écritures rapides — seuls les changements sont écrits
- Faible latence pour les mises à jour ponctuelles
- Adapté aux charges à haute fréquence de modification

**Inconvénients de MOR :**
- Overhead de lecture pour appliquer les suppressions
- Nécessite une compaction régulière pour maintenir les performances
- Complexité accrue de la gestion des fichiers

### Vecteurs de Suppression (Version 3)

La version 3 de la spécification introduit les vecteurs de suppression (deletion vectors), une représentation plus efficace des suppressions par position. Au lieu de fichiers séparés, les suppressions sont encodées dans des bitmaps binaires stockés dans des fichiers Puffin.

Les vecteurs de suppression offrent plusieurs avantages sur les fichiers de position delete :
- Un seul vecteur par fichier de données maximum, simplifiant la gestion
- Représentation bitmap compacte pour les suppressions denses
- Lecture plus efficace car le vecteur peut être mis en cache

La migration vers les vecteurs de suppression est recommandée pour les tables v3 avec des charges de mise à jour significatives.

### Configuration des Stratégies

Iceberg permet de configurer indépendamment la stratégie pour les opérations DELETE, UPDATE et MERGE :

```sql
-- Configuration de table pour MOR
ALTER TABLE events SET TBLPROPERTIES (
  'write.delete.mode' = 'merge-on-read',
  'write.update.mode' = 'merge-on-read',
  'write.merge.mode' = 'copy-on-write'
);
```

Cette flexibilité permet d'optimiser selon les patterns d'utilisation. Par exemple, utiliser MOR pour les suppressions fréquentes de conformité RGPD et COW pour les mises à jour par lots massifs.

> **Étude de cas : Institution Financière Canadienne**  
> *Secteur* : Services financiers  
> *Défi* : Conformité aux demandes de suppression LPRPDE avec tables de 50 To  
> *Solution* : Configuration MOR pour les suppressions, compaction nocturne  
> *Résultats* : Temps de suppression réduit de 4 heures (COW) à 5 minutes (MOR), avec impact négligeable sur les performances de lecture après compaction quotidienne

## Statistiques et Optimisation des Requêtes

Apache Iceberg collecte et maintient plusieurs niveaux de statistiques qui permettent aux moteurs de requête d'optimiser l'exécution. Ces statistiques vont des métriques simples dans les manifestes aux structures probabilistes sophistiquées dans les fichiers Puffin.

### Statistiques de Manifeste

Chaque entrée de manifeste contient des statistiques au niveau des colonnes pour le fichier de données référencé :

**Comptages** — record_count indique le nombre total d'enregistrements, value_counts le nombre de valeurs non-null par colonne, null_value_counts le nombre de null par colonne, et nan_value_counts le nombre de NaN pour les colonnes flottantes.

**Bornes** — lower_bounds et upper_bounds stockent les valeurs minimales et maximales pour chaque colonne. Ces bornes sont sérialisées en binaire selon le type de données.

**Tailles** — column_sizes indique la taille en octets de chaque colonne dans le fichier, et file_size_in_bytes la taille totale du fichier.

Ces statistiques permettent l'élagage de fichiers (data skipping). Pour une requête avec WHERE amount > 1000, les fichiers dont la borne supérieure de la colonne amount est inférieure ou égale à 1000 peuvent être ignorés sans être lus.

### Filtres de Bloom

Les filtres de Bloom sont des structures probabilistes qui permettent de tester rapidement si une valeur appartient à un ensemble. Ils sont particulièrement utiles pour les colonnes à haute cardinalité où les bornes min/max sont peu sélectives.

Pour activer les filtres de Bloom lors de l'écriture :

```sql
-- Activation des filtres de Bloom pour une colonne
ALTER TABLE events SET TBLPROPERTIES (
  'write.parquet.bloom-filter-enabled.column.user_id' = 'true',
  'write.parquet.bloom-filter-max-bytes' = '1048576'
);
```

Un filtre de Bloom peut répondre à la question "cette valeur est-elle possiblement dans ce fichier ?" avec deux réponses possibles : "certainement non" ou "peut-être oui". Cette propriété permet d'éliminer des fichiers avec certitude sans faux négatifs, au prix de quelques faux positifs qui nécessiteront une lecture du fichier.

### Fichiers Puffin et Statistiques Avancées

Le format Puffin est conçu pour stocker des statistiques et index additionnels qui ne peuvent pas être directement intégrés dans les manifestes. Son cas d'usage principal est le stockage des estimations du nombre de valeurs distinctes (NDV) utilisant l'algorithme Theta Sketch.

La structure d'un fichier Puffin comprend :
- Un en-tête magique identifiant le format
- Des blobs contenant les statistiques sérialisées
- Un pied de page avec les métadonnées des blobs

Le blob type apache-datasketches-theta-v1 stocke un sketch Theta qui permet d'estimer le NDV d'une colonne avec une précision configurable. Cette information est cruciale pour l'optimisation des jointures : connaître qu'une colonne a 10 valeurs distinctes versus 10 millions permet au planificateur de choisir la bonne stratégie de jointure.

```python
# Génération des statistiques Puffin avec Spark
spark.sql("CALL catalog.system.compute_table_stats('events')")
```

Les statistiques Puffin sont liées à un snapshot spécifique et doivent être régénérées après des modifications significatives de la table. AWS Glue Data Catalog et d'autres services offrent une génération automatisée de ces statistiques.

### Élagage Multi-Niveau

Lors de la planification d'une requête, Iceberg applique l'élagage à plusieurs niveaux :

1. **Élagage de partition** — Utilise les statistiques de la liste de manifestes pour éliminer des manifestes entiers dont les plages de partition ne correspondent pas aux filtres de requête.

2. **Élagage de fichier** — Pour les manifestes restants, utilise les bornes de colonnes pour éliminer les fichiers dont les valeurs sont hors de la plage recherchée.

3. **Élagage de groupe de lignes** — Au sein des fichiers Parquet sélectionnés, utilise les statistiques de pied de page pour éliminer des groupes de lignes (row groups).

4. **Élagage par Bloom** — Pour les colonnes avec filtres de Bloom activés, vérifie si les valeurs recherchées sont possiblement présentes.

Cette cascade d'élagage peut réduire le volume de données lu de plusieurs ordres de grandeur pour les requêtes sélectives.

## Transactions et Concurrence

Iceberg fournit des garanties ACID complètes pour les opérations sur les tables, incluant l'atomicité, la cohérence, l'isolation et la durabilité. Ces garanties sont essentielles pour les environnements de production où plusieurs processus accèdent simultanément aux mêmes tables.

### Isolation des Lectures

Les lectures dans Iceberg sont toujours isolées des écritures concurrentes. Lorsqu'un lecteur charge les métadonnées d'une table, il obtient une référence au snapshot courant. Toutes les opérations de ce lecteur utilisent ce snapshot, même si de nouveaux snapshots sont créés pendant l'exécution de la requête.

Cette isolation snapshot (snapshot isolation) garantit que les lecteurs voient toujours un état cohérent de la table. Il n'y a jamais de lectures fantômes où des lignes apparaissent ou disparaissent au milieu d'une requête.

### Concurrence Optimiste

Les écritures utilisent un protocole de concurrence optimiste. Plutôt que d'acquérir des verrous avant d'écrire, chaque écrivain suppose qu'il n'y aura pas de conflit et vérifie cette hypothèse au moment du commit.

Le processus de commit d'une écriture suit ces étapes :

1. **Lecture des métadonnées** — L'écrivain charge le fichier metadata.json courant et note le snapshot actif.

2. **Exécution de l'écriture** — L'écrivain crée les nouveaux fichiers de données, les fichiers manifestes et la liste de manifestes pour un nouveau snapshot.

3. **Création des métadonnées** — Un nouveau fichier metadata.json est préparé, référençant le nouveau snapshot.

4. **Validation du commit** — L'écrivain vérifie que le pointeur du catalogue n'a pas changé depuis la lecture initiale.

5. **Commit atomique** — Si le pointeur n'a pas changé, l'écrivain demande au catalogue de basculer atomiquement vers le nouveau fichier de métadonnées.

6. **Résolution de conflit** — Si le pointeur a changé (un autre écrivain a commité entre-temps), l'écrivain peut soit rejouer son opération sur le nouveau snapshot, soit abandonner.

Ce protocole évite les verrous coûteux tout en garantissant la cohérence. Les conflits sont détectés au moment du commit plutôt que prévenus par verrouillage préemptif.

### Types de Conflits

Iceberg distingue plusieurs types de conflits avec des sémantiques de résolution différentes :

**Conflits d'ajout** — Deux écrivains ajoutent des données simultanément. Dans la plupart des cas, ces opérations sont compatibles et peuvent être fusionnées. L'écrivain qui détecte le conflit peut simplement rebaser son commit sur le nouveau snapshot.

**Conflits de suppression** — Un écrivain supprime des lignes qu'un autre écrivain modifie. Ce conflit nécessite généralement une intervention ou une logique de résolution spécifique à l'application.

**Conflits de schéma** — Un écrivain modifie le schéma pendant qu'un autre écrit des données. La résolution dépend de la compatibilité des changements de schéma avec les données écrites.

Le catalogue peut configurer le nombre de tentatives de résolution automatique avant d'abandonner :

```sql
ALTER TABLE events SET TBLPROPERTIES (
  'commit.retry.num-retries' = '10',
  'commit.retry.min-wait-ms' = '100',
  'commit.retry.max-wait-ms' = '60000'
);
```

### Niveau d'Isolation Sérialisable

Pour les cas nécessitant des garanties plus fortes que l'isolation snapshot, Iceberg supporte l'isolation sérialisable. Ce mode garantit que les transactions s'exécutent comme si elles étaient séquentielles, même en présence de concurrence.

L'isolation sérialisable est obtenue en validant que les fichiers lus pendant la transaction n'ont pas été modifiés par d'autres transactions. Si une ligne lue a été modifiée, la transaction est abandonnée plutôt que de produire un résultat potentiellement incohérent.

```sql
-- Activation de l'isolation sérialisable
ALTER TABLE events SET TBLPROPERTIES (
  'write.wap.enabled' = 'true',
  'commit.retry.isolation-level' = 'serializable'
);
```

## Les Tables de Métadonnées

Iceberg expose des tables virtuelles permettant d'interroger les métadonnées d'une table avec SQL standard. Ces tables de métadonnées sont essentielles pour le monitoring, le débogage et la compréhension de l'état d'une table.

### Tables de Métadonnées Principales

**snapshots** — Liste tous les snapshots de la table avec leur timestamp, opération et statistiques de résumé.

```sql
SELECT snapshot_id, committed_at, operation, summary
FROM catalog.db.events.snapshots
ORDER BY committed_at DESC
LIMIT 10;
```

**history** — Affiche l'historique des snapshots courants au fil du temps, permettant de comprendre l'évolution de la table.

**files** — Énumère tous les fichiers de données du snapshot courant avec leurs statistiques.

```sql
SELECT file_path, record_count, file_size_in_bytes
FROM catalog.db.events.files
WHERE record_count > 1000000;
```

**manifests** — Liste les fichiers manifestes du snapshot courant.

**partitions** — Résume les partitions de la table avec des statistiques agrégées.

```sql
SELECT partition, record_count, file_count
FROM catalog.db.events.partitions
ORDER BY record_count DESC;
```

**metadata_log_entries** — Trace l'évolution des fichiers de métadonnées au fil du temps.

### Utilisation pour le Diagnostic

Les tables de métadonnées sont précieuses pour diagnostiquer les problèmes de performance :

```sql
-- Identifier les petits fichiers nécessitant compaction
SELECT COUNT(*) as file_count, 
       AVG(file_size_in_bytes) as avg_size,
       SUM(record_count) as total_records
FROM catalog.db.events.files
WHERE file_size_in_bytes < 10000000;

-- Vérifier l'accumulation de snapshots
SELECT COUNT(*) as snapshot_count,
       MIN(committed_at) as oldest_snapshot,
       MAX(committed_at) as newest_snapshot
FROM catalog.db.events.snapshots;

-- Analyser la distribution des partitions
SELECT partition, file_count, record_count
FROM catalog.db.events.partitions
WHERE file_count > 100
ORDER BY file_count DESC;
```

Ces requêtes permettent d'identifier les tables nécessitant maintenance avant que les problèmes n'impactent les performances de production.

## Le Processus de Commit en Détail

Le processus de commit est le mécanisme central qui garantit les propriétés ACID des tables Iceberg. Comprendre ce processus en détail est essentiel pour diagnostiquer les problèmes de concurrence et optimiser les performances d'écriture.

### Anatomie d'un Commit

Un commit Iceberg suit une séquence précise d'opérations qui transforme les données brutes en une mise à jour atomique de la table.

**Phase 1 : Planification de l'écriture**

L'écrivain détermine d'abord quelles données doivent être écrites et dans quels fichiers. Pour une insertion simple, cela implique de partitionner les données selon la spécification de partition et de regrouper les enregistrements par partition. Pour une mise à jour ou suppression, l'écrivain doit également identifier les fichiers existants affectés.

La configuration des tailles de fichiers cibles influence cette phase :

```sql
-- Configuration des tailles de fichiers
ALTER TABLE events SET TBLPROPERTIES (
  'write.target-file-size-bytes' = '536870912',  -- 512 Mo
  'write.parquet.row-group-size-bytes' = '134217728'  -- 128 Mo
);
```

**Phase 2 : Écriture des fichiers de données**

Les fichiers de données sont écrits dans le stockage objet. Chaque fichier est créé avec un nom unique (généralement un UUID) pour éviter les collisions. Les fichiers sont écrits complètement avant de passer à la phase suivante — il n'y a jamais de fichiers partiellement écrits référencés par les métadonnées.

Le format de fichier par défaut est Parquet avec compression Snappy, mais cela est configurable :

```sql
ALTER TABLE events SET TBLPROPERTIES (
  'write.format.default' = 'parquet',
  'write.parquet.compression-codec' = 'zstd',
  'write.parquet.compression-level' = '3'
);
```

**Phase 3 : Création des manifestes**

Une fois les fichiers de données écrits, l'écrivain crée les fichiers manifestes. Chaque manifeste est un fichier Avro qui liste les fichiers de données ajoutés (status = 1), supprimés (status = 2) ou inchangés (status = 0). Les statistiques de colonnes sont calculées et sérialisées dans le manifeste.

Pour les opérations d'append simples, un seul nouveau manifeste est généralement créé contenant tous les fichiers ajoutés. Pour les opérations de compaction ou de réécriture, les anciens manifestes peuvent être référencés avec les entrées marquées comme supprimées.

**Phase 4 : Création de la liste de manifestes**

La liste de manifestes est créée, référençant tous les manifestes du nouveau snapshot. Elle inclut les statistiques de partition agrégées pour chaque manifeste, permettant l'élagage au niveau manifeste.

**Phase 5 : Préparation des métadonnées**

Un nouveau fichier metadata.json est préparé. Il contient :
- Le nouveau snapshot avec sa référence à la liste de manifestes
- Le snapshot-log mis à jour
- Potentiellement de nouvelles entrées dans schemas ou partition-specs si le schéma a évolué
- Les propriétés de table mises à jour

**Phase 6 : Commit atomique**

C'est l'étape critique. L'écrivain demande au catalogue de basculer atomiquement le pointeur de métadonnées courantes vers le nouveau fichier. Le catalogue vérifie que le pointeur n'a pas changé depuis que l'écrivain a commencé, puis effectue la mise à jour.

Si le pointeur a changé (un autre écrivain a commité entre-temps), l'opération échoue et l'écrivain doit décider de rejouer ou d'abandonner.

### Retry et Backoff Exponentiel

Lorsqu'un conflit est détecté, Iceberg implémente une stratégie de retry avec backoff exponentiel. À chaque tentative échouée, le temps d'attente augmente exponentiellement jusqu'au maximum configuré. Cette approche évite les tempêtes de retry où de nombreux écrivains réessaient simultanément, créant plus de conflits.

Les paramètres configurables incluent le nombre maximum de tentatives (typiquement 4), le temps d'attente minimum (100 ms), le temps d'attente maximum (60 secondes), et le temps total maximum de retry (30 minutes).

### Gestion des Fichiers Orphelins

Un aspect critique du processus de commit est la gestion des fichiers orphelins. Si un écrivain crée des fichiers de données mais échoue avant le commit, ces fichiers restent dans le stockage sans être référencés par aucun snapshot.

Iceberg fournit des procédures pour nettoyer ces fichiers :

```sql
-- Suppression des fichiers orphelins plus vieux que 3 jours
CALL catalog.system.remove_orphan_files(
  table => 'db.events',
  older_than => TIMESTAMP '2024-01-01 00:00:00',
  dry_run => true
);
```

La procédure compare les fichiers présents dans le stockage avec ceux référencés dans les métadonnées et identifie les orphelins. Le mode dry_run permet de prévisualiser sans supprimer.

## Branches et Tags

La spécification Iceberg inclut le support des branches et tags, apportant des concepts de contrôle de version similaires à Git directement aux tables de données.

### Branches

Une branche est un pointeur nommé vers un snapshot qui peut évoluer indépendamment de la branche principale. Les cas d'usage incluent :

**Développement isolé** — Les ingénieurs peuvent créer une branche pour tester des transformations complexes sans affecter les données de production.

```sql
-- Création d'une branche de développement
ALTER TABLE events CREATE BRANCH dev;

-- Écriture sur la branche
INSERT INTO events.branch_dev SELECT ...;

-- Requête de la branche
SELECT * FROM events VERSION AS OF 'dev';
```

**Pipelines de staging** — Les données peuvent être écrites dans une branche de staging, validées, puis promues en production par fast-forward.

**Expérimentation ML** — Différentes versions des données d'entraînement peuvent être maintenues sur des branches séparées pour la reproductibilité des expériences.

### Tags

Un tag est un pointeur nommé immutable vers un snapshot spécifique. Contrairement aux branches qui peuvent avancer, les tags sont permanents et représentent un moment précis de l'historique.

```sql
-- Création d'un tag pour marquer une release
ALTER TABLE events CREATE TAG end_of_q4_2024 
  AS OF SNAPSHOT 8235603094578364387;

-- Requête du tag
SELECT * FROM events VERSION AS OF 'end_of_q4_2024';
```

Les tags sont utiles pour marquer les données utilisées pour entraîner un modèle en production, identifier les états de table correspondant aux rapports financiers, et créer des points de référence pour la reproduction des analyses.

### Rétention des Branches et Tags

Les branches et tags sont indépendants de l'expiration des snapshots. Un snapshot référencé par un tag ou une branche n'est pas éligible à l'expiration, garantissant que les points de référence nommés restent accessibles.

```sql
-- Configuration de la rétention des branches
ALTER TABLE events CREATE BRANCH audit_2024
  RETAIN 365 DAYS
  WITH SNAPSHOT RETENTION 100 SNAPSHOTS;
```

## Intégration avec les Moteurs de Requête

L'architecture ouverte d'Iceberg permet l'intégration avec de nombreux moteurs de requête. Chaque moteur interagit avec les métadonnées de manière similaire mais peut avoir des optimisations spécifiques.

### Apache Spark

Spark est le moteur le plus mature pour Iceberg, avec un support complet des opérations DML et des procédures de maintenance.

```python
# Configuration Spark pour Iceberg
spark = SparkSession.builder \
    .config("spark.sql.extensions", 
            "org.apache.iceberg.spark.extensions.IcebergSparkSessionExtensions") \
    .config("spark.sql.catalog.catalog", 
            "org.apache.iceberg.spark.SparkCatalog") \
    .config("spark.sql.catalog.catalog.type", "rest") \
    .config("spark.sql.catalog.catalog.uri", "http://catalog:8181") \
    .getOrCreate()
```

Spark exploite les statistiques de manifeste pour l'élagage de partition et les statistiques de fichier pour l'élagage de fichier. Le pushdown des prédicats est automatique pour les filtres sur les colonnes de partition et les colonnes avec statistiques.

### Trino

Trino offre des performances de requête exceptionnelles sur Iceberg grâce à son architecture de mémoire distribuée et son optimiseur sophistiqué.

```sql
-- Création d'un catalogue Iceberg dans Trino
CREATE CATALOG iceberg WITH (
  connector.name = 'iceberg',
  iceberg.catalog.type = 'rest',
  iceberg.rest-catalog.uri = 'http://catalog:8181'
);
```

Trino est particulièrement efficace pour les requêtes ad-hoc et les jointures complexes. Il peut générer et utiliser les statistiques Puffin pour l'optimisation des jointures.

### Apache Flink

Flink excelle pour l'ingestion streaming vers Iceberg, créant des commits réguliers à partir de flux de données continus.

Flink utilise le checkpointing pour coordonner les commits, garantissant la cohérence exactly-once même en cas de défaillance. Cette intégration permet de construire des architectures de Streaming Lakehouse où les données passent de Kafka à Iceberg en temps quasi-réel.

### Dremio

Dremio est optimisé pour les requêtes analytiques sur les Lakehouses, avec des accélérations via la réflection et le caching. Dremio utilise les statistiques Iceberg pour l'optimisation mais maintient également ses propres statistiques supplémentaires via les réflections pour accélérer les requêtes récurrentes.

> **Étude de cas : Plateforme de Commerce Électronique**  
> *Secteur* : Commerce de détail en ligne  
> *Défi* : Unifier l'accès aux données entre Spark (ETL), Trino (analytics ad-hoc) et Flink (temps réel)  
> *Solution* : Lakehouse Iceberg avec REST Catalog centralisé  
> *Résultats* : Trois moteurs accèdent aux mêmes tables avec isolation complète, latence de requête ad-hoc réduite de 75 %, ingestion temps réel à 100 000 événements/seconde

## Considérations Avancées de Performance

La compréhension de l'anatomie technique d'Iceberg permet d'anticiper et de résoudre les problèmes de performance. Plusieurs facteurs influencent les performances des tables Iceberg et méritent une attention particulière.

### Prolifération de Petits Fichiers

Le streaming et les écritures fréquentes peuvent créer de nombreux petits fichiers, dégradant les performances de lecture. Chaque fichier implique un overhead de lecture des métadonnées et d'ouverture de fichier. Une table avec un million de fichiers de 1 Mo sera significativement plus lente à interroger qu'une table avec mille fichiers de 1 Go.

Les causes de prolifération incluent :
- Commits fréquents en streaming sans batching adéquat
- Partitionnement trop fin créant de nombreuses partitions avec peu de données
- Opérations de mise à jour fréquentes en mode COW
- Manque de compaction régulière

**Solutions recommandées :**

```sql
-- Configurer une taille cible de fichier appropriée
ALTER TABLE events SET TBLPROPERTIES (
  'write.target-file-size-bytes' = '536870912'  -- 512 Mo
);

-- Exécuter une compaction pour fusionner les petits fichiers
CALL catalog.system.rewrite_data_files(
  table => 'db.events',
  strategy => 'binpack',
  options => map('target-file-size-bytes', '536870912')
);
```

La compaction binpack fusionne les petits fichiers jusqu'à atteindre la taille cible. Des stratégies plus sophistiquées comme sort permettent également de trier les données pour améliorer la localité.

### Accumulation de Snapshots

Sans expiration, les snapshots s'accumulent indéfiniment, gonflant les métadonnées et les coûts de stockage. Chaque snapshot référence tous les fichiers de la table à ce moment, donc même si les données changent peu, les fichiers de métadonnées grossissent.

**Solutions recommandées :**

```sql
-- Configuration de l'expiration automatique
ALTER TABLE events SET TBLPROPERTIES (
  'history.expire.max-snapshot-age-ms' = '604800000',  -- 7 jours
  'history.expire.min-snapshots-to-keep' = '10'
);

-- Expiration manuelle des anciens snapshots
CALL catalog.system.expire_snapshots(
  table => 'db.events',
  older_than => TIMESTAMP '2024-01-01 00:00:00',
  retain_last => 10
);
```

L'expiration supprime les snapshots obsolètes et les fichiers de données qui ne sont plus référencés par aucun snapshot restant. Cette opération doit être planifiée régulièrement pour maintenir la santé de la table.

### Fichiers de Suppression Non Compactés

En mode Merge-on-Read, les fichiers de suppression s'accumulent et dégradent les performances de lecture car chaque lecture doit appliquer les suppressions. Une table avec des milliers de fichiers de suppression peut voir ses temps de requête multiplier par 10 ou plus.

**Indicateurs de problème :**
- Ratio élevé fichiers de suppression / fichiers de données
- Temps de planification de requête anormalement long
- Performance de lecture dégradée sans augmentation de volume

**Solutions recommandées :**

```sql
-- Compaction majeure pour absorber les suppressions
CALL catalog.system.rewrite_data_files(
  table => 'db.events',
  strategy => 'sort',
  options => map(
    'target-file-size-bytes', '536870912',
    'rewrite-all', 'true'
  )
);

-- Réécriture spécifique des fichiers de position delete
CALL catalog.system.rewrite_position_delete_files(
  table => 'db.events'
);
```

### Statistiques Obsolètes

Des statistiques obsolètes peuvent conduire à des plans de requête sous-optimaux, particulièrement pour les jointures. L'optimiseur basé sur les coûts (CBO) utilise les statistiques NDV pour estimer les cardinalités et choisir les stratégies de jointure.

**Solutions recommandées :**

```sql
-- Régénérer les statistiques après modifications majeures
CALL catalog.system.compute_table_stats(
  table => 'db.events',
  columns => array('user_id', 'product_id', 'event_date')
);
```

### Taille du Fichier de Métadonnées

Le fichier metadata.json peut devenir volumineux si l'historique complet des snapshots et des schémas y est conservé. Un fichier de métadonnées de plusieurs centaines de mégaoctets ralentit chaque accès à la table.

**Solutions recommandées :**

```sql
-- Limiter l'historique conservé dans les métadonnées
ALTER TABLE events SET TBLPROPERTIES (
  'write.metadata.delete-after-commit.enabled' = 'true',
  'write.metadata.previous-versions-max' = '100'
);
```

### Monitoring de la Santé des Tables

Le monitoring proactif de la santé des tables permet d'identifier les problèmes avant qu'ils n'impactent la production.

```sql
-- Tableau de bord de santé via les tables de métadonnées
SELECT 
  'file_count' as metric,
  COUNT(*) as value,
  CASE 
    WHEN COUNT(*) > 10000 THEN 'WARNING: Consider compaction'
    ELSE 'OK'
  END as status
FROM catalog.db.events.files
UNION ALL
SELECT 
  'avg_file_size_mb' as metric,
  AVG(file_size_in_bytes) / 1048576 as value,
  CASE 
    WHEN AVG(file_size_in_bytes) < 10485760 THEN 'WARNING: Small files detected'
    ELSE 'OK'
  END as status
FROM catalog.db.events.files
UNION ALL
SELECT 
  'snapshot_count' as metric,
  COUNT(*) as value,
  CASE 
    WHEN COUNT(*) > 1000 THEN 'WARNING: Consider snapshot expiration'
    ELSE 'OK'
  END as status
FROM catalog.db.events.snapshots;
```

Ce type de monitoring peut être automatisé et intégré aux systèmes d'alerting existants pour maintenir la santé des tables Iceberg en production.

## Résumé

Ce chapitre a exploré en profondeur l'anatomie technique d'Apache Iceberg, révélant les mécanismes qui permettent au format de table de délivrer des performances et une fiabilité de niveau entreprise. La compréhension de ces composants internes est essentielle pour tout architecte ou ingénieur de données travaillant avec le Data Lakehouse moderne.

### Architecture en Trois Couches

L'architecture d'Iceberg sépare clairement les préoccupations entre le catalogue, les métadonnées et les données. Le catalogue maintient les références aux tables et garantit l'atomicité des commits. La couche de métadonnées, organisée hiérarchiquement en fichiers metadata.json, listes de manifestes et manifestes, fournit toute l'information nécessaire pour reconstruire l'état de la table à n'importe quel moment. La couche de données stocke les enregistrements dans des formats optimisés comme Parquet.

Cette séparation permet une flexibilité maximale : les moteurs de requête peuvent être changés sans modifier les données, les catalogues peuvent être migrés, et les stratégies de stockage peuvent évoluer. L'isolation des lectures garantit que les requêtes ne sont jamais affectées par les écritures concurrentes.

### Hiérarchie de Métadonnées

La hiérarchie de métadonnées — metadata.json vers manifest list vers manifest vers data files — forme une structure arborescente permettant un accès efficace à différents niveaux de granularité. Cette architecture permet l'élagage progressif : d'abord au niveau des manifestes via les statistiques de partition, puis au niveau des fichiers via les bornes de colonnes, et finalement au niveau des groupes de lignes dans les fichiers Parquet.

Les statistiques collectées à chaque niveau — comptages, bornes min/max, valeurs null — alimentent l'optimiseur pour éliminer les données non pertinentes avant même de les lire. Pour les requêtes sélectives, cette cascade d'élagage peut réduire le volume lu de plusieurs ordres de grandeur.

### Système d'Identifiants de Colonnes

Le système d'identifiants uniques permanents pour chaque champ est l'innovation qui rend possible l'évolution de schéma sans effets de bord. Contrairement aux systèmes basés sur la position ou le nom, l'identification par ID permet de renommer, réordonner et supprimer des colonnes sans impact sur les fichiers existants.

Les colonnes ajoutées reçoivent automatiquement de nouveaux identifiants, garantissant qu'aucune donnée historique ne sera accidentellement interprétée avec le nouveau schéma. Cette approche permet aux tables de longue durée d'évoluer naturellement avec les besoins métier sans migrations coûteuses.

### Partitionnement Masqué et Évolution

Le partitionnement masqué libère les utilisateurs de la gestion explicite des partitions. Les transformations — year, month, day, hour, bucket, truncate — dérivent automatiquement les valeurs de partition des colonnes source. Les producteurs n'ont pas besoin de connaître le schéma de partitionnement, et les consommateurs écrivent des requêtes naturelles.

L'évolution du partitionnement permet de changer le schéma sans réécrire les données existantes. Les anciennes et nouvelles spécifications coexistent, avec une planification de requête appropriée pour chaque layout. Cette flexibilité est cruciale pour adapter les tables aux volumes croissants et aux patterns d'accès évolutifs.

### Stratégies de Modification au Niveau Lignes

Les stratégies Copy-on-Write et Merge-on-Read offrent des compromis configurables entre performance d'écriture et de lecture. COW réécrit les fichiers affectés pour une lecture optimale, tandis que MOR crée des fichiers de suppression pour une écriture rapide.

Les fichiers de suppression positionnelle et d'égalité permettent différents niveaux de compromis. La version 3 introduit les vecteurs de suppression pour une représentation plus efficace. La possibilité de configurer indépendamment DELETE, UPDATE et MERGE permet une optimisation fine selon les patterns d'utilisation.

### Statistiques Multi-Niveaux

Le système de statistiques d'Iceberg opère à plusieurs niveaux. Les manifestes stockent les comptages et bornes de colonnes. Les filtres de Bloom accélèrent les recherches ponctuelles. Les fichiers Puffin stockent les statistiques avancées comme le NDV via les sketches Theta.

Ces statistiques alimentent l'optimisation à chaque étape : planification des requêtes, choix des stratégies de jointure, allocation des ressources. L'investissement dans la collecte et la maintenance des statistiques se traduit directement en performances de requête améliorées.

### Concurrence Optimiste et Transactions

Le protocole de concurrence optimiste permet une haute concurrence sans verrous coûteux. Les écrivains préparent leurs modifications indépendamment et valident au moment du commit. Les conflits sont détectés et peuvent être résolus automatiquement par retry avec backoff exponentiel.

Les garanties ACID — atomicité, cohérence, isolation, durabilité — sont maintenues par ce protocole. Les lecteurs bénéficient de l'isolation snapshot, voyant toujours un état cohérent même pendant les écritures concurrentes.

### Branches et Tags

Le support des branches et tags apporte le contrôle de version Git aux tables de données. Les branches permettent le développement isolé et les pipelines de staging. Les tags marquent les états importants pour l'audit et la reproductibilité.

Ces fonctionnalités sont particulièrement précieuses pour le Machine Learning (versionnement des données d'entraînement), la conformité (audit trails) et les opérations (rollback facile).

### Implications Pratiques

Pour l'architecte de données, la compréhension de ces mécanismes guide les décisions de conception. Le choix entre COW et MOR dépend du ratio lecture/écriture. Le schéma de partitionnement doit correspondre aux patterns de requête. La configuration de l'expiration des snapshots équilibre le voyage dans le temps et les coûts de stockage.

Pour l'ingénieur de données, cette connaissance permet le diagnostic efficace des problèmes. Les tables de métadonnées révèlent l'accumulation de petits fichiers, les fichiers de suppression non compactés, et les snapshots en croissance. Les procédures de maintenance — compaction, expiration, nettoyage d'orphelins — résolvent ces problèmes.

Pour l'équipe d'opérations, les détails techniques éclairent les procédures. Le monitoring de la santé des tables prévient les dégradations de performance. La planification des maintenances minimise l'impact sur les charges de production. La compréhension des conflits guide la configuration de la concurrence.

### Perspectives Techniques

L'évolution continue de la spécification Iceberg apporte régulièrement de nouvelles capacités. La version 3 ajoute les vecteurs de suppression, les types variant pour les données semi-structurées, et les types géospatiaux. La version 4, en développement, promet de nouvelles optimisations et fonctionnalités.

L'ajout prévu d'un endpoint de scan planning au REST Catalog permettra aux catalogues d'optimiser et de cacher les plans de requête. Les vues interopérables standardiseront la définition des vues entre moteurs. Ces évolutions renforceront le positionnement d'Iceberg comme standard universel du Data Lakehouse.

Le chapitre suivant appliquera ces connaissances techniques à des scénarios concrets d'implémentation, démontrant comment créer, alimenter et interroger des tables Iceberg avec différents moteurs de requête dans un environnement de production.
