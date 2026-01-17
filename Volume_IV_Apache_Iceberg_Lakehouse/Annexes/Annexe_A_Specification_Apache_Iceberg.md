# Annexe A - La Spécification Apache Iceberg

## Introduction

Cette annexe constitue un guide de référence technique approfondi sur la spécification Apache Iceberg. Elle s'adresse aux architectes données, ingénieurs plateforme et développeurs qui souhaitent comprendre les fondements techniques du format de table Iceberg pour concevoir, opérer et optimiser leurs environnements Lakehouse.

Apache Iceberg est un format de table ouvert conçu pour gérer de vastes collections de fichiers de données dans des systèmes de fichiers distribués ou des magasins objets. La spécification définit précisément comment les métadonnées sont structurées, comment les opérations transactionnelles sont garanties et comment les différents moteurs de requête peuvent interopérer de manière cohérente sur les mêmes tables.

La compréhension de cette spécification est essentielle pour quiconque souhaite tirer pleinement parti des capacités d'Iceberg, que ce soit pour l'évolution de schéma, le partitionnement masqué, le voyage dans le temps ou l'optimisation des performances de requête.

---

## A.1 Comprendre la spécification Iceberg

### A.1.1 Philosophie et principes fondamentaux

Apache Iceberg a émergé en 2017 chez Netflix pour résoudre un problème fondamental : les lacs de données construits sur des formats de fichiers comme Apache Parquet et ORC manquaient de garanties transactionnelles, d'évolution de schéma fiable et de visibilité sur l'état des tables. Le format Hive, largement utilisé à l'époque, montrait ses limites, particulièrement à grande échelle et lors de la migration vers des magasins objets infonuagiques comme Amazon S3.

La spécification Iceberg repose sur plusieurs principes directeurs :

**Séparation logique et physique.** La définition logique d'une table est découplée de sa disposition physique en stockage. Cette séparation permet à plusieurs moteurs de calcul d'opérer sur le même jeu de données avec des garanties transactionnelles complètes, sans dépendre de conventions de répertoires ou de systèmes de fichiers spécifiques.

**Immutabilité des fichiers.** Les fichiers de données ne sont jamais modifiés après leur écriture. Les mises à jour et suppressions créent de nouveaux fichiers ou des fichiers de suppression, tandis que les snapshots maintiennent l'historique des états de la table. Cette immutabilité simplifie considérablement la gestion de la concurrence et permet des fonctionnalités comme le voyage dans le temps.

**Métadonnées riches.** Iceberg maintient des statistiques détaillées au niveau des colonnes, des partitions et des fichiers. Ces métadonnées permettent aux moteurs de requête d'élaguer efficacement les fichiers et partitions non pertinents, réduisant drastiquement les données analysées.

**Transactions ACID.** Chaque modification de table est atomique. Les lecteurs voient toujours un état cohérent de la table, même pendant les écritures concurrentes. Les conflits sont détectés et résolus au moment du commit grâce au contrôle optimiste de la concurrence.

### A.1.2 Architecture en couches

La spécification définit une architecture en trois couches distinctes :

**Couche catalogue.** Le catalogue maintient une référence vers le fichier de métadonnées courant pour chaque table. Cette référence est le point d'entrée unique pour accéder à une table Iceberg. Le catalogue peut être implémenté de diverses manières : Hive Metastore, catalogue REST, JDBC, AWS Glue, ou fichiers dans un système de stockage.

**Couche métadonnées.** Cette couche comprend le fichier de métadonnées JSON (metadata.json), les listes de manifestes (manifest lists) et les fichiers de manifestes (manifest files). Le fichier de métadonnées contient le schéma de la table, les spécifications de partitionnement, les propriétés et la liste des snapshots. Chaque snapshot référence une liste de manifestes qui elle-même pointe vers les fichiers de manifestes décrivant les fichiers de données.

**Couche données.** Les fichiers de données contiennent les enregistrements réels dans des formats ouverts comme Parquet, ORC ou Avro. Les fichiers de suppression (delete files) et vecteurs de suppression (deletion vectors) marquent les lignes supprimées sans réécrire les fichiers de données originaux.

### A.1.3 Structure des métadonnées

Le fichier de métadonnées JSON constitue le cœur de la définition d'une table Iceberg. Il contient :

**Informations de version.** Le numéro de version du format (1, 2 ou 3), l'UUID unique de la table, et l'emplacement du fichier de métadonnées lui-même.

**Schéma.** La définition complète des colonnes avec leurs types, identifiants uniques et informations de documentation. Les identifiants de champs sont stables à travers les évolutions de schéma, permettant la compatibilité ascendante et descendante.

**Spécifications de partition.** Les règles définissant comment les données sont partitionnées, incluant les transformations appliquées aux colonnes sources (identity, bucket, truncate, year, month, day, hour).

**Spécifications de tri.** L'ordre dans lequel les données sont triées au sein des fichiers, optimisant certains types de requêtes.

**Snapshots.** La liste des états historiques de la table, chaque snapshot représentant un état complet et cohérent. Le snapshot courant définit l'état visible par défaut.

**Historique des snapshots.** Le journal des changements de snapshot courant, permettant le voyage dans le temps vers n'importe quel état précédent.

Les fichiers de manifestes sont des fichiers Avro contenant des métadonnées détaillées sur les fichiers de données : chemins, tailles, statistiques de colonnes (min, max, compte de null, compte de valeurs distinctes), et informations de partition.

### A.1.4 Garanties transactionnelles

Iceberg implémente l'isolation des snapshots (snapshot isolation) pour garantir la cohérence des lectures et écritures concurrentes :

**Lectures cohérentes.** Une lecture voit toujours un état complet de la table correspondant à un snapshot spécifique. Les modifications en cours n'affectent pas les lectures existantes.

**Écritures atomiques.** Une écriture crée un nouveau snapshot contenant tous les changements. L'opération réussit entièrement ou échoue entièrement, sans état intermédiaire visible.

**Contrôle optimiste de concurrence.** Les écritures concurrentes sont détectées au moment du commit. En cas de conflit, l'écriture peut être réessayée avec une base plus récente. Les conflits sont résolus au niveau du catalogue, qui garantit qu'un seul commit réussit pour un état donné.

**Sérialisation des modifications.** Les modifications sont sérialisées par le catalogue via des mécanismes de verrouillage ou de comparaison-et-échange (compare-and-swap), assurant la linéarisabilité des opérations.

---

## A.2 Versions du format de table Iceberg

### A.2.1 Version 1 : Les fondations

La version 1, première version stable de la spécification, établit les fondations de la gestion de tables analytiques à grande échelle. Elle définit :

**Gestion des fichiers immuables.** Support des formats Parquet, Avro et ORC pour le stockage des données. Les fichiers sont écrits une fois et jamais modifiés, garantissant l'intégrité des données et simplifiant la réplication.

**Métadonnées structurées.** Architecture en couches avec fichiers de métadonnées JSON, listes de manifestes et manifestes Avro. Cette structure permet une navigation efficace de la table sans lister récursivement les répertoires.

**Évolution de schéma.** Support pour l'ajout, la suppression et le renommage de colonnes sans réécriture des données existantes. Les identifiants de champs stables assurent la compatibilité.

**Évolution de partition.** Possibilité de modifier le schéma de partitionnement sans réécrire les données. Les nouvelles données utilisent le nouveau schéma tandis que les anciennes conservent leur partitionnement original.

**Partitionnement masqué.** Les colonnes de partition ne sont pas exposées dans le schéma de la table. Les transformations de partition sont appliquées automatiquement lors des écritures et des requêtes.

La version 1 est conçue pour des charges de travail principalement en ajout (append-only), typiques des entrepôts de données traditionnels et des lacs de données analytiques.

### A.2.2 Version 2 : Suppressions au niveau des lignes

La version 2, introduite pour supporter des charges de travail plus dynamiques, ajoute des capacités essentielles pour les pipelines de données modernes :

**Fichiers de suppression par position.** Les fichiers de suppression par position identifient les lignes supprimées par leur chemin de fichier de données et leur position de ligne dans ce fichier. Cette approche évite la réécriture des fichiers de données lors de suppressions ponctuelles.

**Fichiers de suppression par égalité.** Les fichiers de suppression par égalité identifient les lignes supprimées par les valeurs d'une ou plusieurs colonnes. Cette approche est particulièrement efficace pour les suppressions basées sur des clés métier.

**Numéros de séquence.** Chaque opération reçoit un numéro de séquence croissant, permettant d'ordonner les opérations et de déterminer quels fichiers de suppression s'appliquent à quels fichiers de données. Un fichier de suppression ne s'applique qu'aux fichiers de données avec un numéro de séquence inférieur ou égal.

**Modes Copy-on-Write et Merge-on-Read.** Les utilisateurs peuvent choisir entre réécrire les fichiers de données lors des modifications (Copy-on-Write, optimisé pour la lecture) ou créer des fichiers de suppression réconciliés à la lecture (Merge-on-Read, optimisé pour l'écriture).

La version 2 rend Iceberg adapté aux charges de travail dynamiques avec mises à jour fréquentes, aux pipelines de capture de données modifiées (CDC), et aux exigences de conformité RGPD nécessitant des suppressions ciblées.

### A.2.3 Version 3 : Performance et expressivité

La version 3, ratifiée par la communauté Apache Iceberg en 2025, représente un saut significatif en termes de performance et de flexibilité. Les premières fonctionnalités V3 sont apparues dans Iceberg 1.8.0 (février 2025), avec des ajouts dans les versions 1.9.0 et 1.10.0.

**Vecteurs de suppression (Deletion Vectors).** Les vecteurs de suppression remplacent les fichiers de suppression par position avec un format binaire compact stocké dans des fichiers Puffin. Utilisant des bitmaps Roaring, ils offrent une représentation plus efficace des suppressions et éliminent la traduction entre fichiers Parquet et représentations mémoire. Les vecteurs de suppression sont liés à un seul fichier de données, avec au plus un vecteur par fichier de données dans un snapshot.

**Lignage des lignes (Row Lineage).** La version 3 introduit des métadonnées au niveau des lignes pour simplifier le traitement incrémental. Chaque ligne reçoit un identifiant unique (`_row_id`) et un numéro de séquence de dernière mise à jour (`_last_updated_sequence_number`). Ces champs permettent aux moteurs de détecter les changements au niveau des lignes entre commits, optimisant les vues matérialisées et les pipelines CDC.

**Valeurs par défaut des colonnes.** Les colonnes peuvent désormais avoir des valeurs par défaut définies dans les métadonnées. L'ajout d'une colonne avec valeur par défaut est instantané (modification des métadonnées uniquement) sans réécriture des fichiers de données existants. Les moteurs de requête appliquent la valeur par défaut à la volée pour les anciens fichiers.

**Types de données avancés.** La version 3 ajoute plusieurs nouveaux types :
- **Variant** : Type flexible pour données semi-structurées (similaire à JSON) avec support pour date, timestamp, binary et decimal
- **Geometry et Geography** : Types géospatiaux pour analyses de localisation
- **Timestamps nanosecondes** : Précision nanoseconde avec ou sans fuseau horaire

**Transformations multi-arguments.** Support pour des stratégies de partitionnement et de tri plus avancées utilisant plusieurs colonnes ou fonctions composites.

### A.2.4 Compatibilité entre versions

La spécification Iceberg est conçue pour la compatibilité ascendante et descendante :

**Compatibilité ascendante.** Les lecteurs d'une version ultérieure peuvent lire les tables de versions antérieures. Les fichiers de métadonnées V1 sont valides dans les tables V2 et V3, permettant les mises à niveau sans réécriture de l'arbre de métadonnées.

**Mise à niveau atomique.** Une table peut être mise à niveau vers une version ultérieure de manière atomique. Par exemple, la mise à niveau de V2 vers V3 crée un nouveau snapshot de métadonnées, réutilise les fichiers de données Parquet existants, et ajoute les champs de lignage des lignes aux métadonnées de table.

**Mise à niveau irréversible.** Les mises à niveau de version sont unidirectionnelles. Une table mise à niveau de V2 vers V3 ne peut pas être rétrogradée vers V2 via des opérations standard.

**Tolérance des lecteurs.** Les lecteurs doivent être tolérants envers les champs inconnus et les transformations inconnues pour assurer la compatibilité future. Les champs de partition avec transformations inconnues sont ignorés lors de la planification des scans.

---

## A.3 Gestion des snapshots et métadonnées

### A.3.1 Anatomie d'un snapshot

Un snapshot représente un état complet et cohérent d'une table Iceberg à un instant donné. Chaque snapshot contient :

**Identifiant unique.** Un entier long identifiant de manière unique le snapshot au sein de la table.

**ID de snapshot parent.** Référence au snapshot précédent, formant une chaîne de lignage permettant le voyage dans le temps.

**Numéro de séquence.** Numéro croissant assigné lors du commit, utilisé pour ordonner les opérations et déterminer l'applicabilité des fichiers de suppression.

**Horodatage.** Moment de création du snapshot en millisecondes depuis l'époque Unix.

**Emplacement de la liste de manifestes.** Chemin vers le fichier de liste de manifestes décrivant les fichiers de données et de suppression du snapshot.

**Résumé.** Métadonnées résumant les changements : opération effectuée (append, replace, overwrite, delete), statistiques de modifications (fichiers ajoutés/supprimés, enregistrements ajoutés/supprimés).

### A.3.2 Structure des manifestes

La liste de manifestes (manifest list) est un fichier Avro contenant des métadonnées sur les fichiers de manifestes du snapshot :

- Chemin du fichier de manifeste
- Longueur du fichier
- ID de la spécification de partition utilisée
- Plages de valeurs de partition couvertes
- Statistiques agrégées : nombre de fichiers, nombre d'enregistrements, comptes de suppressions

Les fichiers de manifestes (manifest files) sont également des fichiers Avro contenant des métadonnées détaillées sur les fichiers de données :

**Informations de fichier.** Chemin, format (Parquet/ORC/Avro), taille en octets, nombre d'enregistrements.

**Statistiques de colonnes.** Pour chaque colonne : valeurs minimale et maximale, compte de valeurs null, compte de valeurs NaN (pour les flottants).

**Informations de partition.** Valeurs des colonnes de partition pour le fichier.

**Métadonnées de tri.** Ordre de tri des données dans le fichier, permettant une fusion efficace lors des lectures.

### A.3.3 Opérations sur les snapshots

Les opérations de base créant de nouveaux snapshots incluent :

**Append.** Ajout de nouveaux fichiers de données sans modifier les fichiers existants. L'opération la plus courante pour les charges de travail d'ingestion.

**Overwrite.** Remplacement de fichiers de données correspondant à un prédicat de filtre. Utilisé pour les réécritures partielles de table.

**Replace.** Remplacement complet des fichiers de la table. Utilisé pour les reconstructions complètes ou les opérations de compaction.

**Delete.** Ajout de fichiers de suppression marquant des lignes comme supprimées. Utilisé en mode Merge-on-Read.

**Row Delta.** Combinaison d'ajouts de fichiers de données et de fichiers de suppression dans une même opération atomique.

### A.3.4 Voyage dans le temps

Le voyage dans le temps (Time Travel) permet d'interroger la table à un état historique spécifique :

**Par identifiant de snapshot.** Interrogation d'un snapshot spécifique par son ID unique.

**Par horodatage.** Interrogation de l'état de la table à un moment donné. Iceberg sélectionne le snapshot le plus récent dont l'horodatage est antérieur ou égal à la valeur spécifiée.

**Syntaxe SQL.** La plupart des moteurs supportent la syntaxe `AS OF` pour le voyage dans le temps :
```sql
SELECT * FROM table AS OF TIMESTAMP '2025-01-01 00:00:00';
SELECT * FROM table AS OF VERSION 12345678901234;
```

### A.3.5 Expiration des snapshots

L'expiration des snapshots libère l'espace de stockage en supprimant les métadonnées et fichiers de données obsolètes :

**Politique de rétention.** Les snapshots plus anciens qu'un seuil configurable sont candidats à l'expiration. La politique par défaut conserve généralement quelques jours ou semaines d'historique.

**Fichiers orphelins.** L'expiration supprime les fichiers de données qui ne sont plus référencés par aucun snapshot retenu.

**Opération de maintenance.** L'expiration doit être exécutée régulièrement pour éviter l'accumulation de fichiers obsolètes et la croissance incontrôlée du stockage.

### A.3.6 Rollback

Le rollback permet de revenir à un état antérieur de la table :

**Rollback à un snapshot.** Définit un snapshot historique comme le snapshot courant de la table.

**Préservation de l'historique.** Le rollback ne supprime pas les snapshots intermédiaires. Les nouvelles écritures après rollback créent des snapshots basés sur l'état restauré.

**Branches et tags.** Iceberg supporte des branches et tags nommés pour référencer des snapshots spécifiques, facilitant les workflows de gestion de versions des données.

---

## A.4 La spécification REST Catalog

### A.4.1 Motivation et objectifs

La spécification REST Catalog a été développée pour résoudre les problèmes de compatibilité et d'interopérabilité entre catalogues. Avant son introduction, chaque implémentation de catalogue nécessitait des clients spécifiques dans chaque langage supporté par Iceberg (Java, Python, Rust, Go), créant des incohérences et des barrières à l'adoption.

Les objectifs principaux de la spécification REST Catalog sont :

**Compatibilité universelle.** Une seule implémentation client peut interagir avec n'importe quel catalogue conforme à la spécification, indépendamment de l'implémentation serveur.

**Flexibilité d'implémentation.** Les fournisseurs de catalogues peuvent implémenter le serveur dans n'importe quel langage et utiliser n'importe quelle technologie de stockage, tant que l'API REST est respectée.

**Centralisation des opérations.** Les opérations sont gérées côté serveur plutôt que côté client, permettant un contrôle centralisé et des mises à jour simplifiées.

**Sécurité intégrée.** Support natif pour OAuth 2.0 et les mécanismes d'authentification standard.

### A.4.2 Structure de l'API

La spécification REST Catalog définit une API OpenAPI couvrant les opérations suivantes :

**Gestion des espaces de noms.** Création, listage, mise à jour et suppression d'espaces de noms (équivalents aux bases de données ou schémas).

```
GET    /v1/{prefix}/namespaces
POST   /v1/{prefix}/namespaces
GET    /v1/{prefix}/namespaces/{namespace}
DELETE /v1/{prefix}/namespaces/{namespace}
POST   /v1/{prefix}/namespaces/{namespace}/properties
```

**Gestion des tables.** Création, listage, chargement, mise à jour et suppression de tables.

```
GET    /v1/{prefix}/namespaces/{namespace}/tables
POST   /v1/{prefix}/namespaces/{namespace}/tables
GET    /v1/{prefix}/namespaces/{namespace}/tables/{table}
POST   /v1/{prefix}/namespaces/{namespace}/tables/{table}
DELETE /v1/{prefix}/namespaces/{namespace}/tables/{table}
HEAD   /v1/{prefix}/namespaces/{namespace}/tables/{table}
POST   /v1/{prefix}/namespaces/{namespace}/tables/{table}/metrics
```

**Gestion des vues.** Support des vues Iceberg avec opérations similaires aux tables.

**Commit des transactions.** Endpoint pour soumettre les changements de métadonnées avec résolution des conflits côté serveur.

### A.4.3 Authentification et sécurité

La spécification REST Catalog intègre OAuth 2.0 pour l'authentification :

**Flux client credentials.** Échange de credentials (client ID et secret) contre un jeton d'accès.

**Échange de jetons.** Échange d'un jeton utilisateur (jeton sujet) contre un jeton d'accès plus spécifique, utilisant le jeton d'accès du catalogue comme jeton acteur.

**Jetons bearer.** Les requêtes aux tables et vues peuvent inclure des jetons d'autorisation bearer si la sécurité OAuth2 est activée.

**Signature S3 distante.** Pour le stockage S3, la spécification permet la signature distante des requêtes, centralisant les credentials côté serveur.

### A.4.4 Configuration et credentials de stockage

Le REST Catalog gère la configuration d'accès au stockage :

**Storage credentials.** Le serveur retourne des credentials temporaires pour accéder aux fichiers de données. Les clients vérifient d'abord le champ `storage-credentials` avant de consulter le champ `config`.

**Configuration FileIO.** Le serveur peut spécifier une implémentation FileIO spécifique pour la table selon son stockage sous-jacent.

**Préfixes de credentials.** Support pour plusieurs préfixes de credentials de stockage, permettant l'accès à des fichiers dans différentes régions ou comptes.

### A.4.5 Implémentations notables

Plusieurs implémentations de la spécification REST Catalog sont disponibles :

**Apache Polaris (Incubating).** Catalogue open source développé initialement par Snowflake, offrant un serveur REST Catalog complet avec contrôle d'accès granulaire.

**Project Nessie.** Catalogue Git-like avec versioning des métadonnées, branches et merges, exposant une API REST Catalog compatible.

**AWS Glue Data Catalog.** Service AWS géré offrant une API REST Catalog native pour l'intégration avec les services AWS et Iceberg.

**Google BigLake REST Catalog.** Implémentation Google Cloud entièrement gérée, intégrée avec BigQuery pour la gouvernance et la sécurité.

**Dremio Arctic / Hybrid Catalog.** Catalogue géré Dremio intégrant maintenance automatique et fédération multi-emplacements.

### A.4.6 Évolutions futures

La spécification REST Catalog continue d'évoluer pour supporter de nouveaux cas d'usage :

**Endpoint de planification de scan.** Un endpoint permettant de déléguer la planification des scans au serveur est en discussion. Cela permettrait aux moteurs de requête de bénéficier d'optimisations centralisées et de compatibilité multi-formats.

**Service de maintenance.** Un système de notification pour les requêtes de maintenance permettrait aux outils souscrits d'optimiser automatiquement les tables selon des conditions configurables.

**Fédération de catalogues.** Support pour l'interrogation de tables à travers plusieurs catalogues, simplifiant la gestion d'écosystèmes de données distribués.

---

## A.5 Spécification du format de fichier Puffin

### A.5.1 Objectif et cas d'usage

Puffin est un format de fichier conteneur conçu pour stocker des informations auxiliaires sur les données gérées par Iceberg qui ne peuvent pas être stockées directement dans les manifestes. Le format a été adopté par la communauté Apache Iceberg en 2022 et est devenu central avec l'introduction des vecteurs de suppression en V3.

Les cas d'usage principaux de Puffin incluent :

**Statistiques de table.** Stockage d'esquisses (sketches) pour estimer le nombre de valeurs distinctes (NDV) d'une colonne. Ces statistiques sont essentielles pour l'optimisation des plans de requête, notamment le réordonnancement des jointures.

**Vecteurs de suppression.** En V3, les vecteurs de suppression sont stockés dans des fichiers Puffin comme bitmaps binaires compacts, remplaçant les fichiers de suppression par position de V2.

**Index secondaires.** Le format est conçu pour accueillir des index comme les filtres Bloom ou autres structures d'accès accéléré.

### A.5.2 Structure du fichier

Un fichier Puffin suit une structure simple et efficace :

```
Magic (4 octets) | Blob₁ | Blob₂ | ... | Blobₙ | Footer
```

**Magic.** Quatre octets identifiant le format : `0x50, 0x46, 0x41, 0x31` (P, F, A, 1 en ASCII).

**Blobs.** Séquence de blobs binaires contenant les données réelles (statistiques, vecteurs de suppression, index).

**Footer.** Métadonnées JSON décrivant les blobs contenus dans le fichier.

### A.5.3 Métadonnées de fichier

Le footer contient un objet JSON `FileMetadata` avec :

**Blobs.** Liste des métadonnées de chaque blob :
- `type` : Type du blob (ex. `apache-datasketches-theta-v1`, `deletion-vector-v1`)
- `fields` : Liste des IDs de champs concernés
- `snapshot-id` : ID du snapshot source
- `sequence-number` : Numéro de séquence du snapshot
- `offset` : Position du blob dans le fichier
- `length` : Taille du blob en octets
- `compression-codec` : Codec de compression utilisé (optionnel)

**Properties.** Propriétés générales du fichier :
- `created-by` : Identification de l'application ayant créé le fichier et sa version

### A.5.4 Types de blobs définis

La spécification définit plusieurs types de blobs :

**apache-datasketches-theta-v1.** Esquisse Theta sérialisée produite par la bibliothèque Apache DataSketches. L'esquisse est construite en utilisant la famille Alpha avec la graine par défaut, alimentée avec les valeurs distinctes converties en octets selon la sérialisation single-value d'Iceberg.

Les métadonnées du blob peuvent inclure :
- `ndv` : Estimation du nombre de valeurs distinctes dérivée de l'esquisse

**deletion-vector-v1.** Vecteur de suppression sérialisé représentant les positions des lignes supprimées dans un fichier de données. Un bit à 1 à la position P indique que la ligne à la position P est supprimée.

Le format utilise des bitmaps Roaring pour une représentation compacte. Les métadonnées requises incluent :
- `referenced-data-file` : Emplacement du fichier de données référencé
- `cardinality` : Nombre de lignes supprimées (bits à 1) dans le vecteur

Les vecteurs de suppression ne sont pas compressés (le champ `compression-codec` est omis).

### A.5.5 Compression

Les blobs peuvent être compressés avec les codecs suivants pour une interopérabilité maximale :

| Codec | Description |
|-------|-------------|
| `lz4` | Frame de compression LZ4 unique avec taille de contenu présente |
| `zstd` | Frame de compression Zstandard unique avec taille de contenu présente |

Le footer JSON lui-même peut être compressé en LZ4. Les données peuvent également être non compressées si le codec n'est pas spécifié.

### A.5.6 Intégration avec Iceberg

Les fichiers Puffin sont référencés dans les métadonnées de table Iceberg :

**Fichiers de statistiques.** Les fichiers Puffin contenant des statistiques sont enregistrés dans le champ `statistics` des métadonnées de table. Chaque entrée contient :
- `snapshot-id` : ID du snapshot associé
- `statistics-path` : Chemin du fichier Puffin
- `file-size-in-bytes` : Taille du fichier
- `file-footer-size-in-bytes` : Taille du footer
- `blob-metadata` : Liste des métadonnées de blobs

**Vecteurs de suppression.** Les vecteurs de suppression sont suivis individuellement dans les manifestes de suppression par emplacement de fichier, offset et longueur dans le fichier Puffin conteneur.

### A.5.7 Considérations de performance

L'utilisation de fichiers Puffin offre plusieurs avantages de performance :

**Planification de requêtes optimisée.** Les statistiques NDV permettent aux optimiseurs de choisir les meilleures stratégies de jointure et d'ordonnancement des opérations.

**Prédiction d'élagage améliorée.** Les statistiques détaillées permettent un pushdown de prédicats plus efficace, réduisant les données scannées.

**Suppressions efficaces.** Les vecteurs de suppression évitent l'amplification d'écriture des mises à jour par lots et des suppressions de conformité, réduisant significativement le surcoût de maintenance des données fraîches.

**Traitement incrémental.** Les esquisses peuvent être mises à jour de manière incrémentale sans relire les données complètes de la table.

---

## A.6 Compatibilité et migration

### A.6.1 Stratégies de migration vers Iceberg

La migration vers Apache Iceberg peut suivre plusieurs approches selon le contexte :

**Migration in-place.** Conversion d'une table existante (Hive, par exemple) vers Iceberg en place. Cette approche modifie les métadonnées de la table originale pour la convertir en table Iceberg, préservant l'emplacement des fichiers de données.

```sql
CALL spark_catalog.system.migrate('hive_db.hive_table')
```

**Migration par snapshot.** Création d'une copie Iceberg d'une table existante tout en préservant la table source. Cette approche permet de valider la migration avant de basculer les applications.

```sql
CALL spark_catalog.system.snapshot(
  source_table => 'hive_db.hive_table',
  table => 'iceberg_db.iceberg_table'
)
```

**Migration avec reconfiguration.** Migration vers Iceberg avec modification simultanée des propriétés de table (version de format, modes de suppression, etc.).

```sql
CALL spark_catalog.system.snapshot(
  source_table => 'hive_db.hive_table',
  table => 'iceberg_db.iceberg_table',
  properties => map(
    'format-version', '3',
    'write.delete.mode', 'merge-on-read'
  )
)
```

### A.6.2 Mise à niveau des versions de format

La mise à niveau entre versions de format Iceberg suit des règles spécifiques :

**De V1 vers V2.** La mise à niveau active le support des suppressions au niveau des lignes. Elle est effectuée en modifiant la propriété de table :

```sql
ALTER TABLE my_table SET TBLPROPERTIES('format-version' = '2')
```

**De V2 vers V3.** La mise à niveau active les vecteurs de suppression et le lignage des lignes. Elle est atomique et ne nécessite pas de réécriture des données :

```sql
ALTER TABLE my_table SET TBLPROPERTIES('format-version' = '3')
```

Après mise à niveau vers V3 :
- Les champs de lignage des lignes sont ajoutés aux métadonnées de table
- La prochaine compaction supprimera les anciens fichiers de suppression V2
- Les nouvelles modifications utiliseront les fichiers de vecteurs de suppression V3
- La mise à niveau ne remplit pas rétroactivement les enregistrements de suivi de lignage

### A.6.3 Compatibilité des moteurs

Avant de migrer ou mettre à niveau, il est essentiel de vérifier la compatibilité des moteurs :

| Moteur | V1 | V2 | V3 |
|--------|----|----|-----|
| Apache Spark 3.5+ | ✓ | ✓ | ✓ (avec Iceberg 1.8+) |
| Apache Flink 2.0+ | ✓ | ✓ | ✓ (avec Iceberg 1.10+) |
| Trino | ✓ | ✓ | Partiel |
| Dremio | ✓ | ✓ | ✓ |
| AWS Athena | ✓ | ✓ | En cours |
| Snowflake | ✓ | ✓ | Annoncé |

La vérification de compatibilité doit inclure :
- Support des types de données utilisés
- Support des modes de suppression requis
- Performance acceptable pour les cas d'usage cibles

### A.6.4 Considérations de rétrocompatibilité

Lors de la planification d'une migration ou mise à niveau :

**Tables multi-moteurs.** Si plusieurs moteurs accèdent à la même table, tous doivent supporter la version cible avant la mise à niveau.

**Applications héritées.** Les applications non compatibles avec les nouvelles versions nécessitent soit une mise à jour, soit le maintien de tables dans l'ancienne version.

**Période de transition.** Prévoir une période où les deux versions coexistent pour valider le comportement avant de compléter la migration.

### A.6.5 Rollback de migration

En cas de problème après migration :

**Rollback de données.** Utiliser le voyage dans le temps Iceberg pour revenir à un état antérieur si les données sont corrompues.

**Rollback de version.** La rétrogradation de version (V3 vers V2) n'est pas supportée nativement. La solution consiste à recréer la table en V2 à partir d'un export ou d'un snapshot antérieur.

**Tables de sauvegarde.** Conserver les tables sources pendant la période de validation pour permettre un retour arrière complet si nécessaire.

### A.6.6 Meilleures pratiques de migration

Pour une migration réussie, les pratiques suivantes sont recommandées :

**Environnement de test.** Valider la migration dans un environnement de développement avant la production.

**Migration incrémentale.** Commencer par les tables les moins critiques pour acquérir de l'expérience avant de migrer les tables stratégiques.

**Monitoring post-migration.** Surveiller les performances, la taille des fichiers et les métriques de requête après migration pour détecter les problèmes.

**Documentation.** Documenter les configurations, les décisions et les procédures de migration pour faciliter les futures évolutions.

---

## Résumé

Cette annexe a présenté les fondements techniques de la spécification Apache Iceberg, couvrant :

**Principes fondamentaux.** L'architecture en couches d'Iceberg sépare la définition logique des tables de leur stockage physique, permettant l'interopérabilité entre moteurs et les garanties transactionnelles ACID sur des systèmes de fichiers distribués et des magasins objets.

**Versions du format.** La spécification a évolué à travers trois versions majeures : V1 établissant les fondations pour les charges de travail analytiques, V2 ajoutant les suppressions au niveau des lignes pour les pipelines dynamiques, et V3 introduisant les vecteurs de suppression, le lignage des lignes et de nouveaux types de données pour une performance et une expressivité accrues.

**Gestion des snapshots.** Le système de snapshots immutables d'Iceberg permet le voyage dans le temps, le rollback et la concurrence optimiste, tout en nécessitant une maintenance régulière via l'expiration des snapshots pour contrôler la croissance du stockage.

**REST Catalog.** La spécification REST Catalog standardise l'interaction avec les catalogues Iceberg, permettant l'interopérabilité universelle entre clients et serveurs indépendamment de leur implémentation.

**Format Puffin.** Le format de fichier Puffin stocke les statistiques avancées et les vecteurs de suppression, complétant les métadonnées des manifestes pour des optimisations de requête et des suppressions plus efficaces.

**Migration et compatibilité.** Les stratégies de migration permettent l'adoption progressive d'Iceberg et la mise à niveau entre versions de format, tout en maintenant la compatibilité avec les écosystèmes existants.

La maîtrise de cette spécification est essentielle pour concevoir des architectures Lakehouse robustes, optimiser les performances des requêtes et opérer efficacement des environnements de données à grande échelle.

---

*Dernière mise à jour : Janvier 2026*
*Basé sur la spécification Apache Iceberg v3 et les versions 1.8.0 à 1.10.x*
