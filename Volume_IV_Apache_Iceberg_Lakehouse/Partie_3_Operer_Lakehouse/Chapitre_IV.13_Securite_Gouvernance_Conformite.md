# Chapitre IV.13 - SÉCURITÉ, GOUVERNANCE ET CONFORMITÉ DU LAKEHOUSE

---

## Introduction

La démocratisation des données constitue l'une des promesses les plus séduisantes du Data Lakehouse. En centralisant les données organisationnelles dans un format ouvert et accessible par une multitude de moteurs de requête, l'architecture Iceberg permet à chaque analyste, scientifique de données et application d'accéder aux informations dont ils ont besoin. Cependant, cette accessibilité accrue s'accompagne d'une responsabilité proportionnelle : protéger les données sensibles, contrôler les accès, et démontrer la conformité aux cadres réglementaires de plus en plus stricts.

Pour les organisations canadiennes, ces enjeux revêtent une importance particulière. La Loi 25 au Québec, entrée pleinement en vigueur en septembre 2024, impose des exigences strictes en matière de consentement explicite, de droit à l'effacement et de portabilité des données. La LPRPDE (Loi sur la protection des renseignements personnels et les documents électroniques) au niveau fédéral, actuellement en cours de modernisation, établit les principes fondamentaux de protection de la vie privée. Dans les secteurs réglementés tels que les services financiers, les directives du BSIF ajoutent des exigences supplémentaires de résilience opérationnelle et de gestion des risques technologiques.

Ce chapitre présente une approche holistique de la sécurité et de la gouvernance du Lakehouse. Nous examinerons d'abord l'architecture de sécurité multicouche qui protège les données à chaque niveau, du stockage objet au catalogue. Puis nous explorerons les mécanismes de contrôle d'accès, des permissions grossières au niveau des tables jusqu'au masquage dynamique au niveau des colonnes. Ensuite, nous aborderons la gouvernance des données à travers le lignage, la qualité et l'observabilité. Enfin, nous présenterons les stratégies de conformité spécifiques au contexte canadien, illustrées par des études de cas d'organisations ayant réussi cette transformation.

---

## IV.13.1 Architecture de Sécurité Multicouche

### Le Défi de la Sécurité Distribuée

L'architecture du Data Lakehouse introduit des défis de sécurité uniques par rapport aux systèmes monolithiques traditionnels. Dans un entrepôt de données classique, un seul système contrôle l'accès aux données, simplifiant la gestion des permissions. Dans un Lakehouse, les données résident sur un stockage objet (S3, ADLS, GCS), les métadonnées sont gérées par un catalogue (REST Catalog, Glue, Polaris), et de multiples moteurs de requête (Spark, Flink, Trino, Dremio) peuvent accéder aux mêmes tables. Cette distribution crée des vecteurs d'attaque potentiels à chaque couche.

La spécification Apache Iceberg elle-même ne définit pas de mécanismes de gouvernance ou de contrôle d'accès. Ce choix architectural délibéré permet la flexibilité et l'interopérabilité, mais transfère la responsabilité de la sécurité aux couches adjacentes de l'écosystème. Une stratégie de sécurité efficace doit donc couvrir simultanément le stockage, le catalogue et les moteurs de calcul.

```
┌─────────────────────────────────────────────────────────────────────────┐
│             ARCHITECTURE DE SÉCURITÉ MULTICOUCHE DU LAKEHOUSE           │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  COUCHE 1 : STOCKAGE OBJET                                              │
│  ┌───────────────────────────────────────────────────────────────────┐ │
│  │  • Chiffrement au repos (SSE-S3, SSE-KMS, SSE-C)                   │ │
│  │  • Chiffrement en transit (TLS 1.2+)                               │ │
│  │  • Politiques de bucket (IAM, S3 ACL)                              │ │
│  │  • Versioning et Object Lock                                        │ │
│  │  • VPC Endpoints / Private Link                                    │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                                    │                                    │
│                                    ▼                                    │
│  COUCHE 2 : FORMAT DE TABLE (ICEBERG)                                   │
│  ┌───────────────────────────────────────────────────────────────────┐ │
│  │  • Chiffrement natif des fichiers Parquet                          │ │
│  │  • Isolation des métadonnées par table                             │ │
│  │  • Time Travel et audit par snapshots                              │ │
│  │  • Chemins de fichiers isolés par namespace                        │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                                    │                                    │
│                                    ▼                                    │
│  COUCHE 3 : CATALOGUE                                                   │
│  ┌───────────────────────────────────────────────────────────────────┐ │
│  │  • RBAC (Role-Based Access Control)                                │ │
│  │  • Credential Vending (jetons temporaires)                         │ │
│  │  • Authentification OAuth2/OIDC                                    │ │
│  │  • Audit logging des opérations                                    │ │
│  │  • Isolation multi-tenant                                          │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                                    │                                    │
│                                    ▼                                    │
│  COUCHE 4 : MOTEURS DE CALCUL                                           │
│  ┌───────────────────────────────────────────────────────────────────┐ │
│  │  • FGAC (Fine-Grained Access Control)                              │ │
│  │  • Row-Level Security / Column Masking                             │ │
│  │  • Query Audit et Logging                                          │ │
│  │  • Vues dynamiques sécurisées                                      │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Sécurité du Stockage Objet

La première ligne de défense réside dans le stockage objet qui héberge les fichiers de données et de métadonnées Iceberg. Cette couche doit assurer la confidentialité, l'intégrité et la disponibilité des données au repos.

**Le chiffrement au repos** constitue la protection fondamentale. AWS S3 offre trois options de chiffrement côté serveur. SSE-S3 utilise des clés gérées par AWS, offrant simplicité et absence de coût supplémentaire. SSE-KMS utilise AWS Key Management Service, permettant le contrôle des clés, l'audit de leur utilisation et la rotation automatique. SSE-C (clés fournies par le client) offre le contrôle maximal mais nécessite une gestion des clés par l'organisation.

Pour les organisations canadiennes soumises à des exigences réglementaires strictes, SSE-KMS avec des clés CMK (Customer Master Key) hébergées dans une région canadienne représente généralement le meilleur compromis entre sécurité et opérabilité.

```python
# Configuration Terraform pour un bucket S3 sécurisé hébergeant Iceberg
# Région canadienne avec chiffrement KMS et versioning

resource "aws_kms_key" "lakehouse_key" {
  description             = "Clé KMS pour le chiffrement du Lakehouse"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "Enable IAM User Permissions"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Action   = "kms:*"
        Resource = "*"
      },
      {
        Sid    = "Allow Lakehouse Services"
        Effect = "Allow"
        Principal = {
          Service = ["s3.amazonaws.com", "glue.amazonaws.com"]
        }
        Action = [
          "kms:Encrypt",
          "kms:Decrypt",
          "kms:GenerateDataKey*"
        ]
        Resource = "*"
      }
    ]
  })
  
  tags = {
    Environment = "production"
    DataClass   = "confidential"
    Compliance  = "loi25-lprpde"
  }
}

resource "aws_s3_bucket" "lakehouse" {
  bucket = "entreprise-lakehouse-ca-central-1"
  
  tags = {
    Environment = "production"
    DataClass   = "confidential"
  }
}

resource "aws_s3_bucket_versioning" "lakehouse" {
  bucket = aws_s3_bucket.lakehouse.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "lakehouse" {
  bucket = aws_s3_bucket.lakehouse.id

  rule {
    apply_server_side_encryption_by_default {
      kms_master_key_id = aws_kms_key.lakehouse_key.arn
      sse_algorithm     = "aws:kms"
    }
    bucket_key_enabled = true  # Réduit les coûts KMS
  }
}

resource "aws_s3_bucket_public_access_block" "lakehouse" {
  bucket = aws_s3_bucket.lakehouse.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Politique de bucket restrictive
resource "aws_s3_bucket_policy" "lakehouse" {
  bucket = aws_s3_bucket.lakehouse.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "EnforceHTTPS"
        Effect    = "Deny"
        Principal = "*"
        Action    = "s3:*"
        Resource = [
          aws_s3_bucket.lakehouse.arn,
          "${aws_s3_bucket.lakehouse.arn}/*"
        ]
        Condition = {
          Bool = {
            "aws:SecureTransport" = "false"
          }
        }
      },
      {
        Sid       = "RestrictToVPCEndpoint"
        Effect    = "Deny"
        Principal = "*"
        Action    = "s3:*"
        Resource = [
          aws_s3_bucket.lakehouse.arn,
          "${aws_s3_bucket.lakehouse.arn}/*"
        ]
        Condition = {
          StringNotEquals = {
            "aws:SourceVpce" = var.vpc_endpoint_id
          }
        }
      }
    ]
  })
}
```

**L'isolation réseau** via VPC Endpoints (AWS) ou Private Link (Azure, GCP) garantit que le trafic vers le stockage ne transite jamais par l'internet public. Cette configuration est particulièrement importante pour les données sensibles et constitue souvent une exigence explicite des politiques de sécurité d'entreprise.

### Le Chiffrement Natif Iceberg

Au-delà du chiffrement au niveau du stockage, Apache Iceberg supporte le chiffrement natif des fichiers Parquet. Ce mécanisme permet un contrôle plus granulaire, où différentes colonnes peuvent être chiffrées avec différentes clés.

Le chiffrement Iceberg utilise une architecture à deux niveaux de clés. Les clés de chiffrement de données (DEK - Data Encryption Keys) chiffrent les données elles-mêmes. Ces DEK sont ensuite chiffrées par des clés maîtresses (KEK - Key Encryption Keys) stockées dans un système de gestion de clés externe comme AWS KMS ou HashiCorp Vault.

```java
// Configuration du chiffrement Iceberg avec Spark
SparkConf conf = new SparkConf()
    .setAppName("Lakehouse-Encrypted")
    // Configuration du gestionnaire de clés
    .set("spark.sql.catalog.lakehouse.io-impl", 
         "org.apache.iceberg.aws.s3.S3FileIO")
    .set("spark.sql.catalog.lakehouse.encryption.manager", 
         "org.apache.iceberg.encryption.StandardEncryptionManager")
    .set("spark.sql.catalog.lakehouse.encryption.kms", 
         "org.apache.iceberg.aws.kms.AwsKmsClient")
    .set("spark.sql.catalog.lakehouse.encryption.kms.key-id", 
         "arn:aws:kms:ca-central-1:123456789:key/abcd-1234");

// Création d'une table avec chiffrement par colonne
spark.sql("""
    CREATE TABLE lakehouse.clients.donnees_personnelles (
        client_id STRING,
        nom STRING,
        prenom STRING,
        nas STRING,           -- Numéro d'assurance sociale (chiffré)
        date_naissance DATE,  -- (chiffré)
        courriel STRING,
        telephone STRING
    )
    USING iceberg
    TBLPROPERTIES (
        'encryption.column.nas' = 'key1',
        'encryption.column.date_naissance' = 'key1',
        'encryption.key.key1' = 'arn:aws:kms:ca-central-1:123456789:key/nas-key'
    )
""");
```

Cette approche offre plusieurs avantages pour la conformité réglementaire. Les colonnes contenant des données sensibles (numéro d'assurance sociale, informations médicales, données financières) peuvent être protégées indépendamment. L'accès aux clés de déchiffrement peut être contrôlé séparément de l'accès aux tables, permettant une séparation des responsabilités.

### Sécurité au Niveau du Catalogue

Le catalogue représente le point de contrôle centralisé pour la gouvernance du Lakehouse. Les catalogues modernes comme Apache Polaris, Unity Catalog ou Nessie offrent des capacités de sécurité avancées qui vont bien au-delà du simple stockage de métadonnées.

**Apache Polaris**, en incubation à la fondation Apache, émerge comme le standard de facto pour les catalogues Iceberg. Lancé en 2024 par Snowflake et donné à la communauté open source, Polaris implémente intégralement l'API REST Iceberg et ajoute des capacités de gouvernance essentielles.

Le modèle RBAC de Polaris distingue deux types de rôles. Les **Principal Roles** sont attribués aux utilisateurs ou aux applications (service principals) et définissent leur identité dans le système. Les **Catalog Roles** définissent les permissions sur les objets (catalogues, namespaces, tables, vues) et sont attribués aux Principal Roles.

```python
# Configuration d'Apache Polaris avec RBAC
# Création des rôles et attribution des permissions

import requests
from typing import Dict, Any

class PolarisAdminClient:
    def __init__(self, base_url: str, admin_token: str):
        self.base_url = base_url
        self.headers = {
            "Authorization": f"Bearer {admin_token}",
            "Content-Type": "application/json"
        }
    
    def create_catalog_role(self, catalog: str, role_name: str, 
                            privileges: list) -> Dict[str, Any]:
        """Crée un rôle de catalogue avec des privilèges spécifiques."""
        payload = {
            "name": role_name,
            "properties": {}
        }
        
        response = requests.post(
            f"{self.base_url}/api/v1/catalogs/{catalog}/catalog-roles",
            headers=self.headers,
            json=payload
        )
        response.raise_for_status()
        
        # Attribution des privilèges au rôle
        for privilege in privileges:
            self._grant_privilege(catalog, role_name, privilege)
        
        return response.json()
    
    def _grant_privilege(self, catalog: str, role_name: str, 
                         privilege: Dict) -> None:
        """Attribue un privilège à un rôle de catalogue."""
        response = requests.put(
            f"{self.base_url}/api/v1/catalogs/{catalog}/catalog-roles/"
            f"{role_name}/grants",
            headers=self.headers,
            json=privilege
        )
        response.raise_for_status()
    
    def create_principal_role(self, role_name: str) -> Dict[str, Any]:
        """Crée un rôle principal."""
        payload = {
            "name": role_name,
            "properties": {}
        }
        
        response = requests.post(
            f"{self.base_url}/api/v1/principal-roles",
            headers=self.headers,
            json=payload
        )
        response.raise_for_status()
        return response.json()
    
    def assign_catalog_role_to_principal(self, principal_role: str,
                                         catalog: str, 
                                         catalog_role: str) -> None:
        """Attribue un rôle de catalogue à un rôle principal."""
        response = requests.put(
            f"{self.base_url}/api/v1/principal-roles/{principal_role}/"
            f"catalog-roles/{catalog}/{catalog_role}",
            headers=self.headers
        )
        response.raise_for_status()


# Exemple d'utilisation : Configuration RBAC pour une équipe analytique
admin = PolarisAdminClient(
    base_url="https://polaris.entreprise.ca",
    admin_token="admin_token_secret"
)

# Créer les rôles de catalogue avec différents niveaux d'accès
admin.create_catalog_role(
    catalog="production",
    role_name="analyst_readonly",
    privileges=[
        {
            "type": "TABLE_READ",
            "namespace": "ventes",
            "table": "*"
        },
        {
            "type": "TABLE_READ", 
            "namespace": "marketing",
            "table": "*"
        }
    ]
)

admin.create_catalog_role(
    catalog="production",
    role_name="data_engineer",
    privileges=[
        {
            "type": "TABLE_FULL",
            "namespace": "*",
            "table": "*"
        },
        {
            "type": "NAMESPACE_CREATE",
            "namespace": "*"
        }
    ]
)

# Restreindre l'accès aux données sensibles
admin.create_catalog_role(
    catalog="production", 
    role_name="pii_access",
    privileges=[
        {
            "type": "TABLE_READ",
            "namespace": "clients",
            "table": "donnees_personnelles"
        }
    ]
)

# Créer les rôles principaux et les attributions
admin.create_principal_role("equipe_analytique")
admin.create_principal_role("equipe_ingenierie")
admin.create_principal_role("equipe_conformite")

# Attribuer les rôles de catalogue aux rôles principaux
admin.assign_catalog_role_to_principal(
    principal_role="equipe_analytique",
    catalog="production",
    catalog_role="analyst_readonly"
)

admin.assign_catalog_role_to_principal(
    principal_role="equipe_conformite",
    catalog="production", 
    catalog_role="pii_access"
)
```

**Le Credential Vending** représente une innovation majeure des catalogues modernes. Plutôt que de configurer des identifiants statiques (access keys) sur chaque moteur de calcul, le catalogue émet des jetons temporaires (STS tokens) à la demande. Ces jetons ont une durée de vie limitée (typiquement 1 heure) et sont scopés aux tables spécifiques que l'utilisateur est autorisé à accéder.

Ce mécanisme offre plusieurs avantages de sécurité. Il élimine la nécessité de distribuer et gérer des identifiants permanents. En cas de compromission, les jetons expirent automatiquement. L'audit devient plus précis car chaque accès est associé à un jeton spécifique. Les politiques d'accès sont appliquées de manière cohérente quel que soit le moteur de calcul utilisé.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      FLUX DE CREDENTIAL VENDING                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────┐     1. Authentification      ┌────────────────────┐      │
│  │  Client  │ ─────────────────────────────▶│  Identity Provider │      │
│  │  (Spark) │                               │  (Okta, Azure AD)  │      │
│  └────┬─────┘                               └─────────┬──────────┘      │
│       │                                               │                 │
│       │ 2. Requête table                              │ 1b. Token JWT   │
│       │    + JWT token                                ▼                 │
│       │                                     ┌────────────────────┐      │
│       └────────────────────────────────────▶│   Apache Polaris   │      │
│                                             │     Catalog        │      │
│                                             └─────────┬──────────┘      │
│                                                       │                 │
│       ┌───────────────────────────────────────────────┘                 │
│       │                                                                 │
│       │ 3. Vérification permissions RBAC                                │
│       │ 4. Génération STS credentials (scopés à la table)               │
│       │ 5. Retour métadonnées + credentials temporaires                 │
│       ▼                                                                 │
│  ┌──────────┐                               ┌────────────────────┐      │
│  │  Client  │     6. Accès direct S3        │   AWS S3           │      │
│  │  (Spark) │ ─────────────────────────────▶│   (avec STS)       │      │
│  └──────────┘     avec credentials temp.    └────────────────────┘      │
│                                                                         │
│  Avantages :                                                            │
│  • Pas de credentials permanents côté client                            │
│  • Jetons temporaires (1h par défaut)                                   │
│  • Scope limité aux tables autorisées                                   │
│  • Audit précis de chaque accès                                         │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Project Nessie : Git pour les Données

**Project Nessie** offre une approche alternative à la gouvernance du catalogue, inspirée des concepts de contrôle de version Git. Développé par Dremio et donné à la communauté open source, Nessie permet de gérer les tables Iceberg avec des branches, des tags et des commits atomiques multi-tables.

Cette approche apporte des avantages uniques pour la gouvernance. Les **branches** permettent d'isoler les environnements (développement, test, production) tout en partageant le même stockage physique. Les **tags** marquent des états stables (version auditée, point de conformité). Les **commits atomiques** garantissent la cohérence lors des modifications multi-tables.

```python
# Configuration de Nessie avec contrôles d'accès basés sur les branches
from pynessie import init
from pynessie.auth import NessieAuthConfig

# Connexion au serveur Nessie avec authentification
nessie = init(
    "http://nessie.entreprise.ca:19120/api/v1",
    auth=NessieAuthConfig(
        type="bearer",
        token="${NESSIE_AUTH_TOKEN}"
    )
)

# Créer une branche pour l'équipe analytique
# Cette branche est en lecture seule pour la production
branch_analytique = nessie.create_branch(
    name="analytique-q1-2025",
    ref="main"
)

# Créer une branche pour les développements
branch_dev = nessie.create_branch(
    name="dev-feature-xyz",
    ref="main"
)

# Les permissions Nessie contrôlent :
# - Qui peut lire chaque branche
# - Qui peut committer sur chaque branche
# - Qui peut créer/supprimer des branches

# Configuration des règles d'accès (via fichier de configuration)
nessie_access_rules = """
authorization:
  rules:
    # Production : lecture seule sauf pour les data engineers
    - pattern: "main"
      actions:
        - allow: READ
          principals: ["group:analystes", "group:data_engineers"]
        - allow: COMMIT
          principals: ["group:data_engineers"]
        - allow: CREATE_REFERENCE
          principals: ["group:data_engineers"]
          
    # Branches analytiques : lecture seule
    - pattern: "analytique-*"
      actions:
        - allow: READ
          principals: ["group:analystes"]
        - allow: COMMIT
          principals: []  # Personne ne peut modifier
          
    # Branches de développement : accès complet pour les devs
    - pattern: "dev-*"
      actions:
        - allow: READ
          principals: ["group:developpeurs", "group:data_engineers"]
        - allow: COMMIT
          principals: ["group:developpeurs", "group:data_engineers"]
        - allow: DELETE_REFERENCE
          principals: ["group:data_engineers"]
"""
```

La gouvernance basée sur les branches offre un contrôle temporel unique. Un auditeur peut examiner l'état exact des données à un moment donné en créant un tag, sans risque que ces données soient modifiées. Les analystes peuvent travailler sur une branche stable pendant que les ingénieurs préparent des modifications sur une autre branche.

### Gestion des Clés et Rotation

La gestion du cycle de vie des clés de chiffrement constitue un aspect critique souvent négligé. Une stratégie robuste doit couvrir la création, la distribution, la rotation et la révocation des clés.

**La rotation automatique des clés** limite l'impact d'une compromission potentielle. AWS KMS permet de configurer une rotation annuelle automatique des CMK. Cependant, cette rotation ne re-chiffre pas automatiquement les données existantes; les anciennes versions des clés sont conservées pour déchiffrer les données historiques.

```python
# Stratégie de rotation des clés pour le Lakehouse
import boto3
from datetime import datetime, timedelta

class GestionnaireClésLakehouse:
    """
    Gère le cycle de vie des clés KMS pour le Lakehouse.
    
    Stratégie recommandée :
    - Rotation automatique annuelle via KMS
    - Re-chiffrement périodique des données sensibles
    - Séparation des clés par classification de données
    """
    
    def __init__(self, region: str = 'ca-central-1'):
        self.kms = boto3.client('kms', region_name=region)
        self.s3 = boto3.client('s3', region_name=region)
        
    def creer_cle_pour_classification(self, 
                                       classification: str,
                                       description: str) -> str:
        """
        Crée une clé KMS dédiée à une classification de données.
        
        Classifications suggérées :
        - RESTREINT : Données hautement sensibles (NAS, financières)
        - CONFIDENTIEL : Données personnelles (PII)
        - INTERNE : Données métier non publiques
        - PUBLIC : Données sans restriction
        """
        response = self.kms.create_key(
            Description=f"Lakehouse - {description}",
            KeyUsage='ENCRYPT_DECRYPT',
            KeySpec='SYMMETRIC_DEFAULT',
            Origin='AWS_KMS',
            Tags=[
                {'TagKey': 'Environment', 'TagValue': 'production'},
                {'TagKey': 'Classification', 'TagValue': classification},
                {'TagKey': 'Application', 'TagValue': 'lakehouse'},
                {'TagKey': 'ManagedBy', 'TagValue': 'data-platform-team'}
            ]
        )
        
        key_id = response['KeyMetadata']['KeyId']
        
        # Activer la rotation automatique
        self.kms.enable_key_rotation(KeyId=key_id)
        
        # Créer un alias pour faciliter l'utilisation
        alias = f"alias/lakehouse-{classification.lower()}"
        self.kms.create_alias(
            AliasName=alias,
            TargetKeyId=key_id
        )
        
        return key_id
    
    def planifier_rechiffrement(self, 
                                table_path: str,
                                spark_session) -> None:
        """
        Planifie le re-chiffrement d'une table Iceberg.
        
        Le re-chiffrement est nécessaire pour :
        - Révoquer l'accès aux anciennes versions de clés
        - Migrer vers des algorithmes plus forts
        - Conformité aux politiques de rotation
        """
        # Réécrire les fichiers de données avec la nouvelle version de clé
        spark_session.sql(f"""
            CALL system.rewrite_data_files(
                table => '{table_path}',
                options => map(
                    'rewrite-all', 'true',
                    'target-file-size-bytes', '268435456'
                )
            )
        """)
        
        # Expirer les anciens snapshots (chiffrés avec l'ancienne clé)
        spark_session.sql(f"""
            CALL system.expire_snapshots(
                table => '{table_path}',
                older_than => TIMESTAMP '{datetime.utcnow().isoformat()}',
                retain_last => 1
            )
        """)
        
        # Nettoyer les fichiers orphelins
        spark_session.sql(f"""
            CALL system.remove_orphan_files(
                table => '{table_path}'
            )
        """)
    
    def auditer_utilisation_cles(self) -> dict:
        """
        Audite l'utilisation des clés KMS pour identifier :
        - Clés non utilisées (candidates à la suppression)
        - Clés sur-utilisées (risque de performance)
        - Patterns d'accès anormaux
        """
        # Récupérer toutes les clés du Lakehouse
        paginator = self.kms.get_paginator('list_keys')
        
        rapport = {
            'cles_actives': [],
            'cles_desactivees': [],
            'cles_sans_rotation': [],
            'alertes': []
        }
        
        for page in paginator.paginate():
            for key in page['Keys']:
                key_id = key['KeyId']
                
                # Récupérer les métadonnées
                metadata = self.kms.describe_key(KeyId=key_id)['KeyMetadata']
                
                # Vérifier si c'est une clé Lakehouse
                tags = self.kms.list_resource_tags(KeyId=key_id).get('Tags', [])
                if not any(t['TagKey'] == 'Application' and 
                          t['TagValue'] == 'lakehouse' for t in tags):
                    continue
                
                # Vérifier l'état
                if metadata['KeyState'] == 'Enabled':
                    rapport['cles_actives'].append(key_id)
                else:
                    rapport['cles_desactivees'].append(key_id)
                
                # Vérifier la rotation
                rotation_status = self.kms.get_key_rotation_status(KeyId=key_id)
                if not rotation_status['KeyRotationEnabled']:
                    rapport['cles_sans_rotation'].append(key_id)
                    rapport['alertes'].append(
                        f"ALERTE: Clé {key_id} n'a pas la rotation activée"
                    )
        
        return rapport
```

**La séparation des clés par classification** limite l'impact d'une compromission. Une clé compromise pour les données « INTERNE » ne permet pas d'accéder aux données « RESTREINT ». Cette approche complexifie la gestion mais renforce significativement la posture de sécurité.

---

## IV.13.2 Contrôles d'Accès Granulaires

### Les Niveaux de Contrôle d'Accès

La sécurité d'un Lakehouse mature repose sur plusieurs niveaux de contrôle d'accès complémentaires. Chaque niveau adresse des besoins différents et offre des compromis distincts entre granularité, performance et complexité de gestion.

**Le Role-Based Access Control (RBAC)** constitue le niveau fondamental. Il contrôle l'accès aux objets (catalogues, namespaces, tables, vues) en fonction des rôles attribués aux utilisateurs. RBAC répond à la question : « Cet utilisateur peut-il accéder à cette table ? ». Ce niveau est typiquement implémenté au niveau du catalogue et s'applique de manière cohérente à tous les moteurs de calcul.

**Le Fine-Grained Access Control (FGAC)** affine le contrôle au niveau des lignes et des colonnes. Il répond à des questions plus précises : « Cet utilisateur peut-il voir les lignes correspondant à sa région ? » ou « Cette colonne doit-elle être masquée pour cet utilisateur ? ». FGAC est généralement implémenté au niveau des moteurs de calcul car il nécessite l'évaluation des données elles-mêmes.

**L'Attribute-Based Access Control (ABAC)** généralise ces contrôles en définissant des politiques basées sur des attributs (tags, classifications) plutôt que sur des objets individuels. Une politique ABAC pourrait stipuler : « Toutes les colonnes taguées PII doivent être masquées pour les utilisateurs sans le rôle compliance ». Cette approche simplifie la gestion à grande échelle.

### Row-Level Security (RLS)

La sécurité au niveau des lignes permet de filtrer automatiquement les données qu'un utilisateur peut voir en fonction de son identité ou de ses attributs. Cette capacité est essentielle pour les organisations où différentes équipes ou régions doivent accéder aux mêmes tables mais avec des périmètres de données différents.

Unity Catalog de Databricks offre une implémentation mature de RLS via les filtres de lignes. Ces filtres sont définis comme des fonctions SQL qui retournent TRUE pour les lignes accessibles à l'utilisateur courant.

```sql
-- Création d'une fonction de filtre pour la sécurité régionale
-- Les gestionnaires de compte ne voient que les clients de leur région
CREATE OR REPLACE FUNCTION production.securite.filtre_region_gestionnaire()
RETURNS BOOLEAN
RETURN 
    is_account_group_member('administrateurs_donnees')  -- Accès complet
    OR current_user() IN (
        SELECT courriel_gestionnaire 
        FROM production.reference.gestionnaires_regions gr
        WHERE gr.region_id = production.clients.transactions.region_id
    );

-- Application du filtre à la table des transactions
ALTER TABLE production.clients.transactions
SET ROW FILTER production.securite.filtre_region_gestionnaire ON ();

-- Vérification : Un gestionnaire de la région Québec
-- ne verra que les transactions de sa région
SELECT COUNT(*) FROM production.clients.transactions;
-- Retourne 45,230 (transactions Québec seulement)

-- Pour un administrateur de données
-- Retourne 1,234,567 (toutes les transactions)
```

AWS Lake Formation offre une approche similaire via les Data Filters. Ces filtres peuvent être définis de manière déclarative et appliqués lors de l'attribution des permissions.

```python
# Configuration de Row-Level Security avec AWS Lake Formation
import boto3

lakeformation = boto3.client('lakeformation', region_name='ca-central-1')

# Créer un filtre de données pour les transactions
lakeformation.create_data_cells_filter(
    TableData={
        'DatabaseName': 'lakehouse',
        'Name': 'filtre_region_quebec',
        'TableCatalogId': '123456789012',
        'TableName': 'transactions',
        'RowFilter': {
            'FilterExpression': "region = 'QC'"
        }
    }
)

# Créer un filtre pour l'Ontario
lakeformation.create_data_cells_filter(
    TableData={
        'DatabaseName': 'lakehouse',
        'Name': 'filtre_region_ontario',
        'TableCatalogId': '123456789012',
        'TableName': 'transactions',
        'RowFilter': {
            'FilterExpression': "region = 'ON'"
        }
    }
)

# Attribuer les permissions avec le filtre
lakeformation.grant_permissions(
    Principal={
        'DataLakePrincipalIdentifier': 'arn:aws:iam::123456789012:role/AnalysteQuebec'
    },
    Resource={
        'DataCellsFilter': {
            'DatabaseName': 'lakehouse',
            'TableName': 'transactions',
            'TableCatalogId': '123456789012',
            'Name': 'filtre_region_quebec'
        }
    },
    Permissions=['SELECT']
)
```

### Column-Level Security et Masquage Dynamique

Le masquage de colonnes permet de protéger les données sensibles sans créer de copies multiples des tables. Les utilisateurs autorisés voient les valeurs réelles, tandis que les autres voient des valeurs masquées ou nulles.

```sql
-- Fonction de masquage pour le numéro d'assurance sociale
CREATE OR REPLACE FUNCTION production.securite.masque_nas(nas STRING)
RETURNS STRING
RETURN 
    CASE 
        WHEN is_account_group_member('acces_nas_complet') THEN nas
        WHEN is_account_group_member('acces_nas_partiel') THEN 
            CONCAT('***-***-', RIGHT(nas, 3))  -- Affiche seulement les 3 derniers chiffres
        ELSE '***-***-***'  -- Masquage complet
    END;

-- Fonction de masquage pour les montants (arrondi pour certains utilisateurs)
CREATE OR REPLACE FUNCTION production.securite.masque_montant(montant DECIMAL(18,2))
RETURNS DECIMAL(18,2)
RETURN
    CASE
        WHEN is_account_group_member('finance_detail') THEN montant
        WHEN is_account_group_member('finance_apercu') THEN 
            ROUND(montant, -3)  -- Arrondi au millier
        ELSE NULL
    END;

-- Application des masques à la table
ALTER TABLE production.clients.donnees_personnelles
ALTER COLUMN nas SET MASK production.securite.masque_nas;

ALTER TABLE production.finance.transactions
ALTER COLUMN montant SET MASK production.securite.masque_montant;

-- Résultat pour un analyste sans accès NAS complet
SELECT client_id, nom, nas, date_naissance 
FROM production.clients.donnees_personnelles
LIMIT 3;

-- client_id | nom          | nas         | date_naissance
-- C001      | Tremblay     | ***-***-234 | 1985-03-15
-- C002      | Gagnon       | ***-***-891 | 1990-07-22
-- C003      | Roy          | ***-***-456 | 1978-11-08
```

### Vues Dynamiques Sécurisées

Les vues dynamiques offrent une approche alternative au masquage au niveau des tables. Elles permettent de définir des transformations de sécurité complexes et de les appliquer de manière transparente aux consommateurs.

```sql
-- Vue sécurisée qui combine RLS et masquage de colonnes
CREATE OR REPLACE VIEW production.vues_securisees.v_clients_securise AS
SELECT
    client_id,
    nom,
    prenom,
    -- Masquage du courriel basé sur le groupe
    CASE 
        WHEN is_account_group_member('marketing_complet') THEN courriel
        ELSE CONCAT(LEFT(courriel, 3), '***@', SPLIT_PART(courriel, '@', 2))
    END AS courriel,
    -- Masquage du téléphone
    CASE
        WHEN is_account_group_member('service_client') THEN telephone
        ELSE CONCAT('(***) ***-', RIGHT(telephone, 4))
    END AS telephone,
    -- NAS complètement masqué sauf pour compliance
    CASE
        WHEN is_account_group_member('compliance_pii') THEN nas
        ELSE NULL
    END AS nas,
    region,
    date_inscription,
    -- Segment client visible pour le marketing
    segment_client
FROM production.clients.donnees_personnelles
-- RLS : Filtre régional
WHERE 
    is_account_group_member('acces_national')
    OR region = (
        SELECT region 
        FROM production.reference.utilisateurs_regions 
        WHERE courriel = current_user()
    );

-- Attribution des permissions sur la vue uniquement
GRANT SELECT ON production.vues_securisees.v_clients_securise 
TO role_analyste_marketing;

-- Révocation de l'accès direct à la table source
REVOKE ALL ON production.clients.donnees_personnelles 
FROM role_analyste_marketing;
```

### Patterns de Masquage Avancés

Au-delà du masquage simple par remplacement de caractères, plusieurs patterns avancés permettent de préserver l'utilité analytique des données tout en protégeant la confidentialité.

**Le masquage préservant le format** maintient la structure des données (longueur, format) tout en remplaçant les valeurs. Cette approche est utile pour les tests et le développement où le réalisme des données est important.

```sql
-- Fonction de masquage préservant le format pour les numéros de carte
CREATE OR REPLACE FUNCTION production.securite.masque_carte_credit(
    numero_carte STRING
) RETURNS STRING
RETURN
    CASE
        WHEN is_account_group_member('paiements_complet') THEN numero_carte
        ELSE CONCAT(
            '****-****-****-',
            RIGHT(numero_carte, 4)  -- Préserve les 4 derniers chiffres
        )
    END;

-- Masquage préservant le format pour les codes postaux
CREATE OR REPLACE FUNCTION production.securite.masque_code_postal(
    code_postal STRING
) RETURNS STRING
RETURN
    CASE
        WHEN is_account_group_member('geo_complet') THEN code_postal
        -- Préserve la région (3 premiers caractères), masque le reste
        ELSE CONCAT(LEFT(code_postal, 3), ' ***')
    END;

-- Masquage avec bruit différentiel pour les montants
-- Permet l'analyse statistique tout en protégeant les valeurs individuelles
CREATE OR REPLACE FUNCTION production.securite.masque_montant_differentiel(
    montant DECIMAL(18,2),
    epsilon DOUBLE DEFAULT 0.1
) RETURNS DECIMAL(18,2)
RETURN
    CASE
        WHEN is_account_group_member('finance_detail') THEN montant
        ELSE ROUND(
            montant * (1 + (RAND() - 0.5) * epsilon),
            2
        )
    END;
```

**La tokenisation** remplace les valeurs sensibles par des jetons non réversibles, permettant les jointures et agrégations sans exposer les valeurs réelles.

```sql
-- Table de correspondance pour la tokenisation (accès très restreint)
CREATE TABLE production.securite.tokens_clients (
    token STRING,
    valeur_reelle STRING,  -- Chiffré avec clé séparée
    type_donnee STRING,    -- EMAIL, TELEPHONE, NAS
    date_creation TIMESTAMP,
    date_expiration TIMESTAMP
) USING iceberg;

-- Fonction de tokenisation
CREATE OR REPLACE FUNCTION production.securite.tokeniser(
    valeur STRING,
    type_donnee STRING
) RETURNS STRING
LANGUAGE PYTHON AS $$
    import hashlib
    import hmac
    
    # Clé secrète (en production, récupérée depuis un gestionnaire de secrets)
    secret_key = get_secret('tokenization_key')
    
    # Génération du token via HMAC
    token = hmac.new(
        secret_key.encode(),
        (type_donnee + ':' + valeur).encode(),
        hashlib.sha256
    ).hexdigest()[:16]
    
    return f"TKN_{type_donnee}_{token}"
$$;

-- Exemple d'utilisation : Vue tokenisée pour l'analytique
CREATE OR REPLACE VIEW production.analytique.v_transactions_tokenisees AS
SELECT
    transaction_id,
    production.securite.tokeniser(client_id, 'CLIENT') AS client_token,
    montant,
    date_transaction,
    categorie_produit,
    region
FROM production.gold.transactions;
```

**La généralisation** réduit la précision des données pour protéger les individus tout en préservant les tendances statistiques.

```sql
-- Généralisation de l'âge en tranches
CREATE OR REPLACE FUNCTION production.securite.generaliser_age(
    date_naissance DATE
) RETURNS STRING
RETURN
    CASE
        WHEN is_account_group_member('demographie_detail') THEN 
            CAST(YEAR(current_date()) - YEAR(date_naissance) AS STRING)
        ELSE
            CASE
                WHEN YEAR(current_date()) - YEAR(date_naissance) < 18 THEN 'Moins de 18'
                WHEN YEAR(current_date()) - YEAR(date_naissance) < 25 THEN '18-24'
                WHEN YEAR(current_date()) - YEAR(date_naissance) < 35 THEN '25-34'
                WHEN YEAR(current_date()) - YEAR(date_naissance) < 45 THEN '35-44'
                WHEN YEAR(current_date()) - YEAR(date_naissance) < 55 THEN '45-54'
                WHEN YEAR(current_date()) - YEAR(date_naissance) < 65 THEN '55-64'
                ELSE '65 et plus'
            END
    END;

-- Généralisation géographique
CREATE OR REPLACE FUNCTION production.securite.generaliser_localisation(
    code_postal STRING
) RETURNS STRING
RETURN
    CASE
        WHEN is_account_group_member('geo_detail') THEN code_postal
        -- Retourne seulement la région de tri d'acheminement (RTA)
        ELSE LEFT(code_postal, 3)
    END;
```

> **Performance**  
> Les contrôles d'accès granulaires (RLS, masquage) ajoutent une surcharge de traitement. Pour les tables très volumineuses, considérez les stratégies suivantes :
> - Utiliser des filtres basés sur des colonnes de partition pour bénéficier du pruning
> - Pré-calculer les tables de mapping des permissions pour éviter les sous-requêtes coûteuses
> - Évaluer l'utilisation de vues matérialisées pour les cas d'usage récurrents
> - Monitorer les temps de requête et ajuster les politiques si nécessaire

### Tableaux de Bord de Gouvernance

La visibilité sur l'état de la gouvernance est essentielle pour maintenir une posture de sécurité adéquate. Un tableau de bord centralisé permet aux équipes de sécurité et de conformité de surveiller les métriques clés.

```sql
-- Métriques de gouvernance calculées quotidiennement
CREATE OR REPLACE VIEW production.gouvernance.v_metriques_quotidiennes AS
WITH metriques_acces AS (
    SELECT
        date_partition,
        COUNT(DISTINCT principal) AS utilisateurs_actifs,
        COUNT(*) AS total_requetes,
        SUM(CASE WHEN code_erreur = 'ACCESS_DENIED' THEN 1 ELSE 0 END) AS acces_refuses,
        COUNT(DISTINCT ressource_table) AS tables_accedees
    FROM lakehouse.audit.evenements_unifies
    WHERE date_partition >= current_date() - INTERVAL 30 DAYS
    GROUP BY date_partition
),

metriques_pii AS (
    SELECT
        date_partition,
        COUNT(*) AS acces_pii,
        COUNT(DISTINCT principal) AS utilisateurs_pii
    FROM lakehouse.audit.evenements_unifies
    WHERE ressource_table IN (
        SELECT chemin_technique 
        FROM lakehouse.gouvernance.catalogue_metier
        WHERE contient_pii = true
    )
    AND date_partition >= current_date() - INTERVAL 30 DAYS
    GROUP BY date_partition
),

metriques_conformite AS (
    SELECT
        COUNT(*) AS demandes_effacement_en_cours,
        SUM(CASE WHEN statut = 'COMPLETE' THEN 1 ELSE 0 END) AS demandes_completees,
        AVG(delai_traitement_jours) AS delai_moyen_jours
    FROM lakehouse.conformite.demandes_effacement
    WHERE date_demande >= current_date() - INTERVAL 30 DAYS
)

SELECT
    ma.date_partition,
    ma.utilisateurs_actifs,
    ma.total_requetes,
    ma.acces_refuses,
    ROUND(100.0 * ma.acces_refuses / NULLIF(ma.total_requetes, 0), 2) AS taux_refus_pct,
    ma.tables_accedees,
    COALESCE(mp.acces_pii, 0) AS acces_pii,
    COALESCE(mp.utilisateurs_pii, 0) AS utilisateurs_pii,
    mc.demandes_effacement_en_cours,
    mc.demandes_completees,
    mc.delai_moyen_jours
FROM metriques_acces ma
LEFT JOIN metriques_pii mp ON ma.date_partition = mp.date_partition
CROSS JOIN metriques_conformite mc
ORDER BY ma.date_partition DESC;

-- Vue des anomalies de sécurité détectées
CREATE OR REPLACE VIEW production.gouvernance.v_anomalies_securite AS
SELECT
    timestamp,
    principal,
    type_anomalie,
    description,
    severite,
    statut_investigation
FROM (
    -- Accès en dehors des heures normales
    SELECT
        timestamp,
        principal,
        'ACCES_HORS_HEURES' AS type_anomalie,
        CONCAT('Accès à ', ressource_table, ' à ', 
               DATE_FORMAT(timestamp, 'HH:mm')) AS description,
        'MOYEN' AS severite,
        'EN_ATTENTE' AS statut_investigation
    FROM lakehouse.audit.evenements_unifies
    WHERE HOUR(timestamp) < 6 OR HOUR(timestamp) > 22
    AND principal NOT IN (SELECT courriel FROM production.reference.comptes_service)
    
    UNION ALL
    
    -- Volume de données anormalement élevé
    SELECT
        MAX(timestamp) AS timestamp,
        principal,
        'VOLUME_ANORMAL' AS type_anomalie,
        CONCAT('Volume: ', SUM(lignes_affectees), ' lignes en 1 heure') AS description,
        CASE 
            WHEN SUM(lignes_affectees) > 10000000 THEN 'CRITIQUE'
            WHEN SUM(lignes_affectees) > 1000000 THEN 'ELEVE'
            ELSE 'MOYEN'
        END AS severite,
        'EN_ATTENTE' AS statut_investigation
    FROM lakehouse.audit.evenements_unifies
    WHERE timestamp >= current_timestamp() - INTERVAL 1 HOUR
    GROUP BY principal
    HAVING SUM(lignes_affectees) > 100000
    
    UNION ALL
    
    -- Tentatives d'accès répétées refusées
    SELECT
        MAX(timestamp) AS timestamp,
        principal,
        'TENTATIVES_REFUSEES' AS type_anomalie,
        CONCAT(COUNT(*), ' tentatives refusées en 10 minutes') AS description,
        CASE 
            WHEN COUNT(*) > 50 THEN 'CRITIQUE'
            WHEN COUNT(*) > 20 THEN 'ELEVE'
            ELSE 'MOYEN'
        END AS severite,
        'EN_ATTENTE' AS statut_investigation
    FROM lakehouse.audit.evenements_unifies
    WHERE code_erreur = 'ACCESS_DENIED'
    AND timestamp >= current_timestamp() - INTERVAL 10 MINUTES
    GROUP BY principal
    HAVING COUNT(*) > 10
) anomalies
WHERE timestamp >= current_date() - INTERVAL 7 DAYS
ORDER BY 
    CASE severite 
        WHEN 'CRITIQUE' THEN 1 
        WHEN 'ELEVE' THEN 2 
        ELSE 3 
    END,
    timestamp DESC;

---

## IV.13.3 Gouvernance des Données

### Le Lignage des Données avec OpenLineage

Le lignage des données répond à des questions fondamentales pour la gouvernance : D'où viennent ces données ? Comment ont-elles été transformées ? Qui y a accédé ? Ces informations sont essentielles pour l'audit, l'analyse d'impact et le débogage des problèmes de qualité.

**OpenLineage** émerge comme le standard ouvert pour la collecte du lignage. Cette spécification, hébergée par la Linux Foundation AI & Data, définit un modèle de données et un protocole pour capturer les événements de lignage à travers différents outils et plateformes. Apache Spark, Apache Flink, Apache Airflow, dbt et de nombreux autres outils supportent désormais l'émission d'événements OpenLineage.

Le modèle OpenLineage capture trois entités principales. Les **Jobs** représentent les processus de transformation (un job Spark, un DAG Airflow, un modèle dbt). Les **Runs** représentent les exécutions individuelles de ces jobs. Les **Datasets** représentent les entrées et sorties (tables Iceberg, fichiers, topics Kafka).

```python
# Configuration OpenLineage pour Spark avec tables Iceberg
from pyspark.sql import SparkSession

spark = SparkSession.builder \
    .appName("Pipeline-ETL-Clients") \
    .config("spark.jars.packages", 
            "io.openlineage:openlineage-spark_2.12:1.23.0") \
    .config("spark.extraListeners", 
            "io.openlineage.spark.agent.OpenLineageSparkListener") \
    .config("spark.openlineage.transport.type", "http") \
    .config("spark.openlineage.transport.url", 
            "http://marquez.entreprise.ca:5000") \
    .config("spark.openlineage.namespace", "lakehouse-production") \
    .config("spark.openlineage.parentJobNamespace", "orchestration") \
    .config("spark.openlineage.parentJobName", "pipeline-clients-quotidien") \
    .getOrCreate()

# Exécution d'un pipeline ETL
# OpenLineage capture automatiquement les entrées, sorties et transformations

# Lecture depuis les sources
df_transactions = spark.read \
    .format("iceberg") \
    .load("lakehouse.bronze.transactions_brutes")

df_clients = spark.read \
    .format("iceberg") \
    .load("lakehouse.bronze.clients_bruts")

# Transformations
df_enrichi = df_transactions \
    .join(df_clients, "client_id") \
    .withColumn("montant_cad", 
                F.when(F.col("devise") == "USD", 
                       F.col("montant") * 1.36)
                 .otherwise(F.col("montant"))) \
    .withColumn("date_traitement", F.current_timestamp())

# Écriture vers la couche Silver
df_enrichi.writeTo("lakehouse.silver.transactions_enrichies") \
    .using("iceberg") \
    .partitionedBy(F.years("date_transaction")) \
    .createOrReplace()

# OpenLineage émet automatiquement un événement incluant :
# - Input datasets : transactions_brutes, clients_bruts
# - Output dataset : transactions_enrichies
# - Schema de chaque dataset
# - Statistiques d'exécution (lignes lues/écrites, durée)
# - Lignage au niveau des colonnes (column-level lineage)
```

**Marquez** constitue l'implémentation de référence pour la collecte et la visualisation des événements OpenLineage. Il fournit une API REST pour l'ingestion, une base de données pour le stockage et une interface web pour l'exploration du graphe de lignage.

```yaml
# Configuration Docker Compose pour Marquez avec stockage PostgreSQL
version: '3.8'

services:
  marquez:
    image: marquezproject/marquez:latest
    ports:
      - "5000:5000"  # API
      - "5001:5001"  # Admin
    environment:
      MARQUEZ_CONFIG: /opt/marquez/config/marquez.yml
    volumes:
      - ./marquez-config.yml:/opt/marquez/config/marquez.yml
    depends_on:
      - postgres-marquez

  marquez-web:
    image: marquezproject/marquez-web:latest
    ports:
      - "3000:3000"
    environment:
      MARQUEZ_HOST: marquez
      MARQUEZ_PORT: 5000

  postgres-marquez:
    image: postgres:14
    environment:
      POSTGRES_USER: marquez
      POSTGRES_PASSWORD: ${MARQUEZ_DB_PASSWORD}
      POSTGRES_DB: marquez
    volumes:
      - marquez-data:/var/lib/postgresql/data

volumes:
  marquez-data:
```

Le lignage au niveau des colonnes (Column-Level Lineage) représente une avancée significative d'OpenLineage. Il permet de tracer exactement quelles colonnes source ont contribué à chaque colonne de sortie, incluant les transformations appliquées.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    LIGNAGE AU NIVEAU DES COLONNES                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  transactions_brutes                 transactions_enrichies             │
│  ┌────────────────────┐             ┌────────────────────────┐         │
│  │ transaction_id     │─────────────▶│ transaction_id        │         │
│  │ client_id          │──┐           │ client_id             │         │
│  │ montant            │──┼───────────▶│ montant_cad          │         │
│  │ devise             │──┘           │ date_transaction      │         │
│  │ date_transaction   │─────────────▶│ nom_client           │         │
│  └────────────────────┘              │ region_client         │         │
│                                      │ date_traitement       │         │
│  clients_bruts                       └────────────────────────┘         │
│  ┌────────────────────┐                       ▲                        │
│  │ client_id          │───────────────────────┤                        │
│  │ nom                │───────────────────────┤                        │
│  │ region             │───────────────────────┘                        │
│  └────────────────────┘                                                │
│                                                                         │
│  Transformations capturées :                                            │
│  • montant_cad = CASE WHEN devise='USD' THEN montant*1.36 ELSE montant │
│  • date_traitement = current_timestamp()                                │
│  • JOIN sur client_id                                                   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Qualité des Données

La gouvernance du Lakehouse inclut nécessairement la qualité des données. Les tables Iceberg peuvent contenir des milliards d'enregistrements, et détecter les anomalies de qualité devient un défi à cette échelle.

Plusieurs frameworks permettent d'implémenter des contrôles de qualité automatisés. **Great Expectations** offre une approche déclarative pour définir et valider des « attentes » sur les données. **Deequ** (Amazon) s'intègre nativement à Spark pour des validations performantes à grande échelle. **Soda Core** propose une syntaxe simple et des intégrations avec les orchestrateurs.

```python
# Validation de qualité avec Great Expectations sur une table Iceberg
import great_expectations as gx
from great_expectations.core.batch import RuntimeBatchRequest

# Configuration du contexte
context = gx.get_context()

# Définition de la suite de validations
context.add_expectation_suite("transactions_silver_suite")

# Création des attentes
validator = context.get_validator(
    batch_request=RuntimeBatchRequest(
        datasource_name="spark_iceberg",
        data_connector_name="default_runtime_data_connector",
        data_asset_name="transactions_enrichies",
        runtime_parameters={"path": "lakehouse.silver.transactions_enrichies"},
        batch_identifiers={"batch_id": "daily_2025_01_16"},
    ),
    expectation_suite_name="transactions_silver_suite",
)

# Attentes de complétude
validator.expect_column_values_to_not_be_null("transaction_id")
validator.expect_column_values_to_not_be_null("client_id")
validator.expect_column_values_to_not_be_null("montant_cad")

# Attentes de format
validator.expect_column_values_to_match_regex(
    "client_id", 
    regex=r"^C[0-9]{6}$",
    meta={"description": "Format client_id : C suivi de 6 chiffres"}
)

# Attentes de plage
validator.expect_column_values_to_be_between(
    "montant_cad",
    min_value=0.01,
    max_value=1000000.00,
    meta={"description": "Montant entre 0.01 et 1M CAD"}
)

# Attentes de référentiel
validator.expect_column_values_to_be_in_set(
    "region_client",
    ["QC", "ON", "BC", "AB", "MB", "SK", "NS", "NB", "PE", "NL", "NT", "YT", "NU"]
)

# Attentes statistiques
validator.expect_column_mean_to_be_between(
    "montant_cad",
    min_value=50.0,
    max_value=500.0,
    meta={"description": "Montant moyen attendu entre 50 et 500 CAD"}
)

# Unicité
validator.expect_column_values_to_be_unique("transaction_id")

# Fraîcheur des données
validator.expect_column_max_to_be_between(
    "date_transaction",
    min_value=(datetime.now() - timedelta(days=1)).isoformat(),
    max_value=datetime.now().isoformat(),
    meta={"description": "Données datant de moins de 24h"}
)

# Exécution des validations
results = validator.validate()

# Intégration avec le pipeline : échec si validations critiques échouent
if not results.success:
    critical_failures = [
        r for r in results.results 
        if not r.success and r.expectation_config.meta.get("severity") == "critical"
    ]
    if critical_failures:
        raise DataQualityException(
            f"Échec des validations critiques : {critical_failures}"
        )
```

### Catalogage et Découverte des Données

Un catalogue de données métier complète le catalogue technique (Polaris, Glue) en ajoutant une couche de contexte métier. Il permet aux utilisateurs de découvrir les données disponibles, comprendre leur signification et identifier les propriétaires.

```sql
-- Structure de métadonnées métier dans le Lakehouse
-- Table de documentation des assets de données

CREATE TABLE lakehouse.gouvernance.catalogue_metier (
    asset_id STRING,
    asset_type STRING,  -- TABLE, VIEW, MODEL, DASHBOARD
    chemin_technique STRING,  -- lakehouse.silver.transactions
    nom_metier STRING,
    description STRING,
    domaine_metier STRING,  -- Finance, Marketing, Operations
    proprietaire_courriel STRING,
    intendant_donnees_courriel STRING,
    classification_donnees STRING,  -- PUBLIC, INTERNE, CONFIDENTIEL, RESTREINT
    contient_pii BOOLEAN,
    contient_donnees_financieres BOOLEAN,
    retention_jours INT,
    reglementation_applicable ARRAY<STRING>,  -- ['LOI25', 'LPRPDE', 'SOX']
    termes_glossaire ARRAY<STRING>,
    tags ARRAY<STRING>,
    date_creation TIMESTAMP,
    date_derniere_modification TIMESTAMP,
    frequence_mise_a_jour STRING,  -- TEMPS_REEL, QUOTIDIEN, HEBDOMADAIRE
    documentation_url STRING
) USING iceberg
PARTITIONED BY (domaine_metier);

-- Exemple d'entrée de catalogue
INSERT INTO lakehouse.gouvernance.catalogue_metier VALUES (
    'asset_001',
    'TABLE',
    'lakehouse.silver.transactions_enrichies',
    'Transactions Clients Enrichies',
    'Table consolidée des transactions clients avec enrichissement démographique et conversion de devise. Source de vérité pour les rapports financiers et analyses marketing.',
    'Finance',
    'marie.tremblay@entreprise.ca',
    'jean.gagnon@entreprise.ca',
    'CONFIDENTIEL',
    true,
    true,
    2555,  -- 7 ans pour conformité SOX
    array('LOI25', 'LPRPDE', 'SOX'),
    array('transaction', 'client', 'montant', 'chiffre_affaires'),
    array('finance', 'analytique', 'certifie'),
    current_timestamp(),
    current_timestamp(),
    'QUOTIDIEN',
    'https://wiki.entreprise.ca/donnees/transactions-enrichies'
);
```

---

## IV.13.4 Conformité Réglementaire au Canada

### La Loi 25 du Québec

La Loi 25 (anciennement Projet de loi 64), pleinement en vigueur depuis septembre 2024, constitue le cadre de protection des renseignements personnels le plus strict au Canada. Elle s'aligne sur le RGPD européen et impose des obligations significatives aux organisations qui traitent des données de résidents québécois.

Les principales exigences de la Loi 25 avec impact sur l'architecture Lakehouse incluent :

**Le consentement explicite** pour la collecte et l'utilisation des renseignements personnels. Le Lakehouse doit pouvoir démontrer que le consentement a été obtenu et enregistrer les préférences de chaque individu.

**Le droit à l'effacement** (droit à l'oubli) permet aux individus de demander la suppression de leurs données. Cette exigence entre en tension directe avec l'immutabilité naturelle d'un Lakehouse, où le time travel conserve l'historique des données.

**Le droit à la portabilité** oblige les organisations à fournir les données personnelles dans un format structuré et couramment utilisé. Le Lakehouse doit pouvoir extraire et formater les données d'un individu spécifique.

**L'évaluation des facteurs relatifs à la vie privée (EFVP)** est obligatoire pour tout projet impliquant des renseignements personnels. L'architecture du Lakehouse elle-même doit faire l'objet d'une EFVP.

```python
# Implémentation du droit à l'effacement conforme à la Loi 25
from datetime import datetime
from dataclasses import dataclass
from typing import List, Dict, Optional
import logging

@dataclass
class DemandeEffacement:
    """Représente une demande d'effacement Loi 25."""
    demande_id: str
    identifiant_personne: str  # Courriel, NAS, ou autre identifiant
    type_identifiant: str
    date_demande: datetime
    tables_a_traiter: List[str]
    demandeur_courriel: str
    motif: str
    statut: str = "EN_ATTENTE"

class GestionnaireDroitEffacementLoi25:
    """
    Gestionnaire des demandes d'effacement conformes à la Loi 25.
    
    Responsabilités :
    - Traiter les demandes d'effacement dans le délai légal (30 jours)
    - Supprimer les données des tables Iceberg
    - Expirer les snapshots contenant les données supprimées
    - Produire les tombstones Kafka pour la propagation
    - Générer le certificat de suppression pour audit
    """
    
    def __init__(self, spark_session, kafka_producer, 
                 iceberg_catalog, journal_audit):
        self.spark = spark_session
        self.producer = kafka_producer
        self.catalog = iceberg_catalog
        self.audit = journal_audit
        self.logger = logging.getLogger(__name__)
    
    def traiter_demande(self, demande: DemandeEffacement) -> Dict:
        """
        Traite une demande d'effacement de bout en bout.
        
        Étapes :
        1. Validation et journalisation de la demande
        2. Identification des données à supprimer
        3. Suppression des tables Iceberg (DELETE)
        4. Expiration des snapshots historiques
        5. Production des tombstones Kafka
        6. Génération du certificat de conformité
        """
        self.logger.info(f"Traitement demande effacement: {demande.demande_id}")
        
        # 1. Journaliser la demande pour audit
        self.audit.journaliser_demande_effacement(
            demande_id=demande.demande_id,
            identifiant=demande.identifiant_personne,
            type_identifiant=demande.type_identifiant,
            date_demande=demande.date_demande,
            demandeur=demande.demandeur_courriel
        )
        
        resultats = {}
        
        # 2-3. Pour chaque table, supprimer les données
        for table_name in demande.tables_a_traiter:
            try:
                # Identifier la colonne correspondant à l'identifiant
                colonne_id = self._obtenir_colonne_identifiant(
                    table_name, 
                    demande.type_identifiant
                )
                
                # Compter les lignes avant suppression (pour audit)
                count_avant = self._compter_lignes(
                    table_name, colonne_id, demande.identifiant_personne
                )
                
                # Exécuter la suppression via Spark SQL
                self.spark.sql(f"""
                    DELETE FROM {table_name}
                    WHERE {colonne_id} = '{demande.identifiant_personne}'
                """)
                
                # Vérifier la suppression
                count_apres = self._compter_lignes(
                    table_name, colonne_id, demande.identifiant_personne
                )
                
                resultats[table_name] = {
                    'lignes_supprimees': count_avant - count_apres,
                    'timestamp': datetime.utcnow().isoformat(),
                    'statut': 'SUCCES' if count_apres == 0 else 'PARTIEL'
                }
                
                # Journaliser pour audit
                self.audit.journaliser_suppression_table(
                    demande_id=demande.demande_id,
                    table=table_name,
                    lignes_supprimees=count_avant - count_apres
                )
                
            except Exception as e:
                self.logger.error(f"Erreur suppression {table_name}: {e}")
                resultats[table_name] = {
                    'erreur': str(e),
                    'statut': 'ECHEC'
                }
        
        # 4. Expirer les snapshots historiques contenant les données
        self._expirer_snapshots_historiques(demande.tables_a_traiter)
        
        # 5. Produire les tombstones Kafka pour propagation
        self._produire_tombstones(
            identifiant=demande.identifiant_personne,
            type_identifiant=demande.type_identifiant
        )
        
        # 6. Générer le certificat de suppression
        certificat = self._generer_certificat(demande, resultats)
        
        return certificat
    
    def _expirer_snapshots_historiques(self, tables: List[str]) -> None:
        """
        Expire les snapshots Iceberg pour garantir que les données
        supprimées ne sont plus accessibles via time travel.
        
        Note : Cette opération doit être planifiée avec soin car
        elle est irréversible et affecte la capacité de rollback.
        """
        for table_name in tables:
            # Expirer tous les snapshots antérieurs à maintenant
            self.spark.sql(f"""
                CALL system.expire_snapshots(
                    table => '{table_name}',
                    older_than => TIMESTAMP '{datetime.utcnow().isoformat()}',
                    retain_last => 1
                )
            """)
            
            # Nettoyer les fichiers orphelins
            self.spark.sql(f"""
                CALL system.remove_orphan_files(
                    table => '{table_name}'
                )
            """)
    
    def _produire_tombstones(self, identifiant: str, 
                             type_identifiant: str) -> None:
        """
        Produit des tombstones Kafka pour les topics contenant
        potentiellement les données de la personne concernée.
        
        Un tombstone (message avec valeur null) permet la compaction
        du log Kafka pour supprimer les données historiques.
        """
        topics_affectes = self._identifier_topics_affectes(type_identifiant)
        
        for topic in topics_affectes:
            self.producer.send(
                topic=topic,
                key=identifiant.encode('utf-8'),
                value=None  # Tombstone
            )
            self.producer.flush()
    
    def _generer_certificat(self, demande: DemandeEffacement,
                            resultats: Dict) -> Dict:
        """
        Génère un certificat de suppression attestant de l'exécution
        de la demande d'effacement, requis pour audit Loi 25.
        """
        certificat = {
            'certificat_id': f"CERT-{demande.demande_id}",
            'demande_id': demande.demande_id,
            'date_traitement': datetime.utcnow().isoformat(),
            'identifiant_personne_hash': self._hasher_identifiant(
                demande.identifiant_personne
            ),
            'resultats_par_table': resultats,
            'snapshots_expires': True,
            'tombstones_produits': True,
            'conformite_loi25': True,
            'delai_traitement_jours': (
                datetime.utcnow() - demande.date_demande
            ).days
        }
        
        # Vérification du délai légal (30 jours)
        if certificat['delai_traitement_jours'] > 30:
            certificat['alerte_delai'] = "DEPASSEMENT_DELAI_LEGAL"
            self.logger.warning(
                f"Demande {demande.demande_id} traitée hors délai légal"
            )
        
        # Stocker le certificat pour audit
        self.audit.stocker_certificat(certificat)
        
        return certificat
```

### La LPRPDE Fédérale

La Loi sur la protection des renseignements personnels et les documents électroniques (LPRPDE) s'applique aux organisations du secteur privé dans tout le Canada pour les activités commerciales. Bien que moins stricte que la Loi 25, elle établit des principes fondamentaux que le Lakehouse doit respecter.

Les dix principes de la LPRPDE se traduisent en exigences architecturales :

| Principe LPRPDE | Implication pour le Lakehouse |
|-----------------|------------------------------|
| Responsabilité | Désignation d'un responsable, documentation des politiques |
| Détermination des fins | Métadonnées documentant la finalité de chaque dataset |
| Consentement | Enregistrement des consentements, respect des préférences |
| Limitation de la collecte | Contrôles limitant les données collectées au nécessaire |
| Limitation de l'utilisation | Contrôles d'accès basés sur la finalité déclarée |
| Exactitude | Processus de validation et correction des données |
| Mesures de sécurité | Chiffrement, contrôles d'accès, surveillance |
| Transparence | Catalogue accessible documentant les données détenues |
| Accès individuel | Mécanismes d'extraction des données personnelles |
| Possibilité de porter plainte | Processus de gestion des demandes et plaintes |

### Conformité Sectorielle (BSIF, SOX)

Les organisations dans des secteurs réglementés font face à des exigences supplémentaires. Le Bureau du surintendant des institutions financières (BSIF) impose aux banques et assureurs canadiens des directives strictes en matière de gestion des risques technologiques et de résilience opérationnelle.

**La directive B-13 (Gestion du risque lié aux technologies et du cyberrisque)** exige notamment l'inventaire complet des actifs de données, la classification selon leur criticité, et des contrôles d'accès proportionnés aux risques.

**La ligne directrice E-21 (Gestion du risque lié aux tiers)** s'applique lorsque le Lakehouse utilise des services infonuagiques de tiers (AWS, Azure, GCP, Confluent). Elle exige une diligence raisonnable, des clauses contractuelles appropriées et une surveillance continue.

```sql
-- Structure de classification des données pour conformité BSIF
CREATE TABLE lakehouse.gouvernance.classification_bsif (
    table_id STRING,
    chemin_table STRING,
    classification_criticite STRING,  -- CRITIQUE, IMPORTANT, STANDARD
    classification_sensibilite STRING,  -- TRES_ELEVE, ELEVE, MOYEN, FAIBLE
    proprietaire_metier STRING,
    gardien_technique STRING,
    
    -- Exigences de protection basées sur classification
    chiffrement_requis BOOLEAN,
    masquage_requis BOOLEAN,
    retention_minimale_jours INT,
    retention_maximale_jours INT,
    
    -- Contrôles d'accès
    acces_restreint_equipes ARRAY<STRING>,
    approbation_requise BOOLEAN,
    niveau_approbation STRING,  -- GESTIONNAIRE, DIRECTEUR, VP
    
    -- Résilience
    rpo_heures INT,  -- Recovery Point Objective
    rto_heures INT,  -- Recovery Time Objective
    sauvegarde_requise BOOLEAN,
    replication_geo_requise BOOLEAN,
    
    -- Audit
    audit_acces_requis BOOLEAN,
    retention_audit_jours INT,
    
    -- Tiers
    partage_tiers_autorise BOOLEAN,
    tiers_autorises ARRAY<STRING>,
    
    date_derniere_revue DATE,
    prochaine_revue DATE
) USING iceberg;

-- Exemple de classification pour une table critique
INSERT INTO lakehouse.gouvernance.classification_bsif VALUES (
    'tbl_001',
    'lakehouse.gold.positions_portefeuille',
    'CRITIQUE',
    'TRES_ELEVE',
    'vp.gestion.actifs@banque.ca',
    'architecte.donnees@banque.ca',
    true,   -- chiffrement
    true,   -- masquage
    2555,   -- retention min 7 ans
    3650,   -- retention max 10 ans
    array('equipe_gestion_actifs', 'equipe_risques', 'equipe_conformite'),
    true,   -- approbation requise
    'VP',
    4,      -- RPO 4 heures
    24,     -- RTO 24 heures
    true,   -- sauvegarde
    true,   -- réplication géo
    true,   -- audit accès
    2555,   -- retention audit 7 ans
    false,  -- pas de partage tiers
    array(),
    current_date(),
    date_add(current_date(), 365)
);
```

---

## IV.13.5 Audit et Surveillance

### Journalisation des Accès

Une stratégie d'audit complète capture les accès à plusieurs niveaux : le stockage objet, le catalogue et les moteurs de calcul. La corrélation de ces journaux permet de reconstituer l'activité complète d'un utilisateur.

```python
# Pipeline d'agrégation des journaux d'audit multi-sources
from pyspark.sql import SparkSession
from pyspark.sql import functions as F

spark = SparkSession.builder \
    .appName("Audit-Aggregation") \
    .getOrCreate()

# Source 1 : AWS CloudTrail (accès S3)
df_cloudtrail = spark.read \
    .format("json") \
    .load("s3://audit-logs/cloudtrail/") \
    .filter(F.col("eventSource") == "s3.amazonaws.com") \
    .select(
        F.col("eventTime").alias("timestamp"),
        F.lit("S3").alias("source"),
        F.col("userIdentity.arn").alias("principal"),
        F.col("eventName").alias("action"),
        F.col("requestParameters.bucketName").alias("ressource_bucket"),
        F.col("requestParameters.key").alias("ressource_cle"),
        F.col("sourceIPAddress").alias("ip_source"),
        F.col("errorCode").alias("code_erreur")
    )

# Source 2 : Polaris Audit Logs (accès catalogue)
df_polaris = spark.read \
    .format("json") \
    .load("s3://audit-logs/polaris/") \
    .select(
        F.col("timestamp"),
        F.lit("POLARIS").alias("source"),
        F.col("principal"),
        F.col("action"),
        F.col("catalog").alias("ressource_catalogue"),
        F.col("namespace").alias("ressource_namespace"),
        F.col("table").alias("ressource_table"),
        F.col("sourceIp").alias("ip_source"),
        F.col("errorCode").alias("code_erreur")
    )

# Source 3 : Spark Query Logs (requêtes exécutées)
df_spark_audit = spark.read \
    .format("json") \
    .load("s3://audit-logs/spark/") \
    .select(
        F.col("timestamp"),
        F.lit("SPARK").alias("source"),
        F.col("user").alias("principal"),
        F.col("operation").alias("action"),
        F.col("database").alias("ressource_database"),
        F.col("table").alias("ressource_table"),
        F.col("query").alias("requete_sql"),
        F.col("duration_ms").alias("duree_ms"),
        F.col("rows_affected").alias("lignes_affectees")
    )

# Agrégation et enrichissement
df_audit_unifie = df_cloudtrail \
    .unionByName(df_polaris, allowMissingColumns=True) \
    .unionByName(df_spark_audit, allowMissingColumns=True) \
    .withColumn("date_partition", F.to_date("timestamp")) \
    .withColumn("heure", F.hour("timestamp"))

# Écriture vers la table d'audit unifiée
df_audit_unifie.writeTo("lakehouse.audit.evenements_unifies") \
    .using("iceberg") \
    .partitionedBy("date_partition") \
    .append()

# Création de vues analytiques pour les investigations
spark.sql("""
    CREATE OR REPLACE VIEW lakehouse.audit.v_acces_pii AS
    SELECT 
        timestamp,
        source,
        principal,
        action,
        COALESCE(ressource_table, ressource_cle) as ressource,
        ip_source,
        requete_sql
    FROM lakehouse.audit.evenements_unifies
    WHERE 
        ressource_table IN (
            SELECT chemin_technique 
            FROM lakehouse.gouvernance.catalogue_metier
            WHERE contient_pii = true
        )
        OR ressource_cle LIKE '%donnees_personnelles%'
    ORDER BY timestamp DESC
""")
```

### Alertes de Sécurité

La détection proactive des comportements anormaux constitue une couche de défense essentielle. Les patterns suivants méritent une surveillance particulière :

```yaml
# Configuration des alertes de sécurité pour le Lakehouse
# alertmanager-rules.yml

groups:
  - name: lakehouse_security_alerts
    rules:
      # Tentatives d'accès non autorisées
      - alert: AccesNonAutoriseDetecte
        expr: |
          sum(rate(polaris_access_denied_total[5m])) by (principal) > 10
        for: 5m
        labels:
          severity: warning
          equipe: securite
        annotations:
          summary: "Tentatives d'accès refusées répétées"
          description: "Le principal {{ $labels.principal }} a eu {{ $value }} tentatives refusées en 5 minutes"
          
      # Accès inhabituel aux données PII
      - alert: AccesPIIAnormal
        expr: |
          sum(rate(spark_query_pii_tables_total[1h])) by (user) 
          > 3 * avg_over_time(spark_query_pii_tables_total[7d])
        for: 15m
        labels:
          severity: high
          equipe: securite
        annotations:
          summary: "Volume d'accès PII anormalement élevé"
          description: "L'utilisateur {{ $labels.user }} accède aux tables PII à un rythme 3x supérieur à la normale"
          
      # Export massif de données
      - alert: ExportMassifDetecte
        expr: |
          sum(rate(iceberg_rows_read_total[10m])) by (user, table) > 10000000
        for: 5m
        labels:
          severity: critical
          equipe: securite
        annotations:
          summary: "Export massif de données détecté"
          description: "{{ $labels.user }} lit >10M lignes de {{ $labels.table }} en 10 minutes"
          
      # Accès depuis IP inhabituelle
      - alert: AccesIPInhabituelle
        expr: |
          count(polaris_access_total{ip_source!~"10\\..*|172\\.16\\..*|192\\.168\\..*"}) by (principal, ip_source) > 0
        for: 1m
        labels:
          severity: high
          equipe: securite
        annotations:
          summary: "Accès depuis IP externe"
          description: "{{ $labels.principal }} accède depuis l'IP {{ $labels.ip_source }}"
          
      # Modification de permissions
      - alert: ModificationPermissions
        expr: |
          increase(polaris_grant_revoke_total[5m]) > 0
        for: 0m
        labels:
          severity: info
          equipe: gouvernance
        annotations:
          summary: "Modification des permissions détectée"
          description: "Des permissions ont été modifiées dans le catalogue"
```

### Réponse aux Incidents de Sécurité

Un plan de réponse aux incidents documenté et testé est essentiel pour minimiser l'impact d'une violation de données. Le Lakehouse, grâce à ses capacités de time travel et son audit détaillé, offre des outils puissants pour l'investigation et la remédiation.

```python
# Framework de réponse aux incidents pour le Lakehouse
from dataclasses import dataclass
from datetime import datetime
from enum import Enum
from typing import List, Optional
import json

class SeveriteIncident(Enum):
    CRITIQUE = "CRITIQUE"  # Données sensibles exposées, action immédiate
    ELEVE = "ELEVE"        # Accès non autorisé détecté
    MOYEN = "MOYEN"        # Anomalie nécessitant investigation
    FAIBLE = "FAIBLE"      # Événement suspect à surveiller

@dataclass
class IncidentSecurite:
    """Représente un incident de sécurité dans le Lakehouse."""
    incident_id: str
    timestamp_detection: datetime
    severite: SeveriteIncident
    type_incident: str  # ACCES_NON_AUTORISE, EXFILTRATION, MODIFICATION
    description: str
    tables_affectees: List[str]
    utilisateur_suspect: Optional[str]
    ip_source: Optional[str]
    statut: str = "OUVERT"

class GestionnaireIncidents:
    """
    Gère le cycle de vie des incidents de sécurité du Lakehouse.
    
    Phases de réponse :
    1. Détection et qualification
    2. Confinement immédiat
    3. Investigation approfondie
    4. Remédiation
    5. Récupération
    6. Leçons apprises
    """
    
    def __init__(self, spark_session, polaris_client, 
                 notification_service):
        self.spark = spark_session
        self.polaris = polaris_client
        self.notifier = notification_service
    
    def phase_1_detection(self, incident: IncidentSecurite) -> dict:
        """
        Phase 1 : Détection et qualification initiale.
        
        Actions :
        - Confirmer l'incident
        - Évaluer la sévérité
        - Notifier les parties prenantes appropriées
        """
        # Journaliser l'incident
        self._journaliser_incident(incident)
        
        # Notification selon la sévérité
        if incident.severite == SeveriteIncident.CRITIQUE:
            self.notifier.alerte_urgente(
                destinataires=["securite@entreprise.ca", 
                              "ciso@entreprise.ca",
                              "juridique@entreprise.ca"],
                sujet=f"[CRITIQUE] Incident sécurité Lakehouse: {incident.incident_id}",
                corps=self._formater_alerte(incident)
            )
        elif incident.severite == SeveriteIncident.ELEVE:
            self.notifier.alerte_urgente(
                destinataires=["securite@entreprise.ca"],
                sujet=f"[ÉLEVÉ] Incident sécurité Lakehouse: {incident.incident_id}",
                corps=self._formater_alerte(incident)
            )
        
        return {"phase": "DETECTION", "statut": "COMPLETE"}
    
    def phase_2_confinement(self, incident: IncidentSecurite) -> dict:
        """
        Phase 2 : Confinement pour limiter l'impact.
        
        Actions :
        - Révoquer les accès suspects
        - Isoler les ressources compromises
        - Préserver les preuves
        """
        actions_prises = []
        
        # Révoquer immédiatement les accès de l'utilisateur suspect
        if incident.utilisateur_suspect:
            self.polaris.revoquer_tous_acces(
                principal=incident.utilisateur_suspect
            )
            actions_prises.append(
                f"Accès révoqués pour {incident.utilisateur_suspect}"
            )
        
        # Bloquer l'IP source si identifiée
        if incident.ip_source:
            self._ajouter_ip_liste_noire(incident.ip_source)
            actions_prises.append(
                f"IP {incident.ip_source} ajoutée à la liste noire"
            )
        
        # Créer un tag Iceberg pour marquer l'état des tables au moment de l'incident
        for table in incident.tables_affectees:
            tag_name = f"incident_{incident.incident_id}_{datetime.utcnow().strftime('%Y%m%d%H%M%S')}"
            self.spark.sql(f"""
                ALTER TABLE {table} 
                CREATE TAG `{tag_name}`
            """)
            actions_prises.append(
                f"Tag de préservation créé pour {table}: {tag_name}"
            )
        
        return {
            "phase": "CONFINEMENT", 
            "statut": "COMPLETE",
            "actions": actions_prises
        }
    
    def phase_3_investigation(self, incident: IncidentSecurite) -> dict:
        """
        Phase 3 : Investigation approfondie.
        
        Utilise les capacités du Lakehouse :
        - Time travel pour examiner l'état avant/après
        - Audit logs pour retracer les actions
        - Lignage pour identifier l'impact en aval
        """
        rapport_investigation = {
            "incident_id": incident.incident_id,
            "timestamp_debut_investigation": datetime.utcnow().isoformat()
        }
        
        # Analyser les accès de l'utilisateur suspect
        if incident.utilisateur_suspect:
            df_acces = self.spark.sql(f"""
                SELECT 
                    timestamp,
                    action,
                    ressource_table,
                    requete_sql,
                    lignes_affectees,
                    ip_source
                FROM lakehouse.audit.evenements_unifies
                WHERE principal = '{incident.utilisateur_suspect}'
                AND timestamp >= '{(incident.timestamp_detection - timedelta(days=30)).isoformat()}'
                ORDER BY timestamp DESC
            """)
            
            rapport_investigation["historique_acces"] = df_acces.collect()
        
        # Examiner les modifications sur les tables affectées
        for table in incident.tables_affectees:
            # Obtenir l'historique des snapshots
            df_snapshots = self.spark.sql(f"""
                SELECT * FROM {table}.history
                ORDER BY made_current_at DESC
            """)
            
            rapport_investigation[f"snapshots_{table}"] = df_snapshots.collect()
            
            # Comparer les données avant/après l'incident (si applicable)
            if incident.type_incident == "MODIFICATION":
                snapshot_avant = self._trouver_snapshot_avant(
                    table, incident.timestamp_detection
                )
                if snapshot_avant:
                    df_diff = self.spark.sql(f"""
                        SELECT 'SUPPRIME' as change_type, * 
                        FROM {table} VERSION AS OF {snapshot_avant}
                        EXCEPT
                        SELECT 'SUPPRIME' as change_type, * 
                        FROM {table}
                        
                        UNION ALL
                        
                        SELECT 'AJOUTE' as change_type, * 
                        FROM {table}
                        EXCEPT
                        SELECT 'AJOUTE' as change_type, * 
                        FROM {table} VERSION AS OF {snapshot_avant}
                    """)
                    rapport_investigation[f"modifications_{table}"] = df_diff.collect()
        
        # Identifier l'impact en aval via le lignage
        for table in incident.tables_affectees:
            tables_dependantes = self._obtenir_dependances_aval(table)
            rapport_investigation[f"impact_aval_{table}"] = tables_dependantes
        
        return rapport_investigation
    
    def phase_4_remediation(self, incident: IncidentSecurite,
                            rapport_investigation: dict) -> dict:
        """
        Phase 4 : Remédiation des dommages.
        
        Actions possibles :
        - Restauration depuis un snapshot antérieur
        - Correction des données modifiées
        - Renforcement des contrôles
        """
        actions_remediation = []
        
        # Si des données ont été modifiées, proposer une restauration
        if incident.type_incident == "MODIFICATION":
            for table in incident.tables_affectees:
                snapshot_propre = self._trouver_dernier_snapshot_propre(
                    table, incident.timestamp_detection
                )
                
                if snapshot_propre:
                    # Restauration (nécessite approbation)
                    self.notifier.demander_approbation(
                        destinataires=["proprietaire_donnees@entreprise.ca"],
                        sujet=f"Approbation restauration {table}",
                        corps=f"""
                        L'incident {incident.incident_id} a causé des modifications 
                        non autorisées sur {table}.
                        
                        Snapshot de restauration proposé: {snapshot_propre}
                        
                        Veuillez approuver ou rejeter la restauration.
                        """
                    )
                    actions_remediation.append(
                        f"Demande de restauration envoyée pour {table}"
                    )
        
        # Renforcer les contrôles d'accès
        for table in incident.tables_affectees:
            # Activer l'audit détaillé si pas déjà actif
            self._activer_audit_detaille(table)
            actions_remediation.append(
                f"Audit détaillé activé pour {table}"
            )
        
        return {
            "phase": "REMEDIATION",
            "statut": "EN_COURS",
            "actions": actions_remediation
        }
    
    def phase_5_recuperation(self, incident: IncidentSecurite,
                             approbation_restauration: bool = False) -> dict:
        """
        Phase 5 : Récupération et retour à la normale.
        
        Actions :
        - Exécuter les restaurations approuvées
        - Rétablir les accès légitimes
        - Valider l'intégrité des données
        """
        actions_recuperation = []
        
        # Exécuter les restaurations si approuvées
        if approbation_restauration:
            for table in incident.tables_affectees:
                snapshot_propre = self._trouver_dernier_snapshot_propre(
                    table, incident.timestamp_detection
                )
                
                if snapshot_propre:
                    self.spark.sql(f"""
                        CALL system.rollback_to_snapshot(
                            table => '{table}',
                            snapshot_id => {snapshot_propre}
                        )
                    """)
                    actions_recuperation.append(
                        f"Restauration exécutée pour {table}"
                    )
        
        # Valider l'intégrité des données restaurées
        for table in incident.tables_affectees:
            validation = self._valider_integrite(table)
            actions_recuperation.append(
                f"Validation intégrité {table}: {validation}"
            )
        
        return {
            "phase": "RECUPERATION",
            "statut": "COMPLETE",
            "actions": actions_recuperation
        }
    
    def phase_6_lecons_apprises(self, incident: IncidentSecurite,
                                 rapport_complet: dict) -> dict:
        """
        Phase 6 : Documentation et amélioration continue.
        
        Produit un rapport final incluant :
        - Chronologie complète
        - Cause racine
        - Actions correctives
        - Recommandations
        """
        rapport_final = {
            "incident_id": incident.incident_id,
            "date_cloture": datetime.utcnow().isoformat(),
            "chronologie": self._construire_chronologie(incident),
            "cause_racine": "",  # À compléter par l'analyste
            "donnees_exposees": [],  # À évaluer
            "actions_correctives_implementees": [],
            "recommandations": [
                "Renforcer la formation des utilisateurs",
                "Réviser les politiques d'accès",
                "Augmenter la fréquence des audits"
            ],
            "indicateurs_surveillance": []
        }
        
        # Stocker le rapport pour référence future
        self._archiver_rapport(rapport_final)
        
        # Notifier les parties prenantes de la clôture
        self.notifier.notification_cloture(
            destinataires=["securite@entreprise.ca", 
                          "gouvernance@entreprise.ca"],
            rapport=rapport_final
        )
        
        return rapport_final
```

Ce framework de réponse aux incidents exploite les capacités uniques du Lakehouse Iceberg. Le time travel permet d'examiner l'état exact des données à tout moment dans le passé. Les tags préservent des points de référence pour l'investigation. L'audit unifié retrace chaque action effectuée. Le lignage identifie l'impact potentiel en cascade sur les systèmes consommateurs.

---

## IV.13.6 Études de Cas Canadiennes

> **Étude de cas : Banque Nationale du Canada**  
> *Secteur* : Services financiers  
> *Défi* : Moderniser l'architecture de données pour supporter l'analytique avancée et l'IA tout en maintenant la conformité BSIF, Loi 25 et SOX. L'infrastructure legacy Hadoop/Hive ne supportait pas les exigences de gouvernance granulaire.  
> *Solution* : Migration vers un Lakehouse Iceberg sur AWS avec :  
> - Apache Polaris comme catalogue centralisé avec RBAC et credential vending  
> - Unity Catalog (Databricks) pour les contrôles fins (RLS, column masking)  
> - OpenLineage/Marquez pour le lignage de bout en bout  
> - Pipeline automatisé de conformité Loi 25 (effacement, portabilité)  
> *Architecture* :  
> - 3 environnements isolés (dev, staging, prod) avec politiques RBAC distinctes  
> - Classification automatique des données via ML (détection PII)  
> - Chiffrement SSE-KMS avec rotation des clés  
> - Audit unifié CloudTrail + Polaris + Spark dans une table Iceberg dédiée  
> *Résultats* :  
> - Temps de réponse aux demandes d'accès BSIF réduit de 2 semaines à 4 heures  
> - 100 % des demandes Loi 25 traitées dans le délai légal de 30 jours  
> - Réduction de 60 % des coûts de conformité grâce à l'automatisation  
> - Zero incident de sécurité des données depuis la mise en production

> **Étude de cas : Hydro-Québec**  
> *Secteur* : Énergie (société d'État)  
> *Défi* : Centraliser les données de 4,3 millions de clients et des réseaux de distribution tout en respectant les exigences de souveraineté des données et la Loi 25. Les données incluent des informations sensibles sur la consommation énergétique des résidences.  
> *Solution* : Lakehouse souverain déployé exclusivement dans les régions canadiennes :  
> - Stockage S3 dans ca-central-1 (Montréal) avec réplication vers ca-west-1 (Calgary)  
> - Nessie comme catalogue open source avec contrôles d'accès basés sur les branches  
> - Masquage dynamique des adresses et données de consommation pour les analystes  
> - Intégration avec le système de gestion des consentements pour la Loi 25  
> *Architecture* :  
> - Séparation physique des données clients (zone restreinte) et données opérationnelles (zone standard)  
> - Anonymisation automatique des données pour l'analytique générale  
> - Vues sécurisées pour chaque équipe avec périmètre géographique  
> - Audit complet avec rétention 10 ans pour conformité Loi sur les archives  
> *Résultats* :  
> - Conformité Loi 25 certifiée par audit externe  
> - Temps d'extraction pour demandes d'accès réduit de 5 jours à 2 heures  
> - Démocratisation des données : 200+ analystes avec accès sécurisé (vs 15 auparavant)  
> - Économies de 2,1 M$ CAD/an en licences et infrastructure

> **Étude de cas : Shopify**  
> *Secteur* : Commerce électronique (technologie)  
> *Défi* : Gérer les données de millions de marchands et leurs clients à travers le monde, avec conformité multi-juridictionnelle (RGPD, CCPA, Loi 25, LPRPDE). Volume de données massif nécessitant des contrôles d'accès performants.  
> *Solution* : Lakehouse global avec gouvernance fédérée :  
> - Tables Iceberg partitionnées par région géographique pour isolation réglementaire  
> - Attribute-Based Access Control (ABAC) basé sur les tags de classification  
> - Pipeline automatisé de « Right to Be Forgotten » traitant 50 000+ demandes/mois  
> - Système de consentement intégré au pipeline d'ingestion  
> *Architecture* :  
> - Multi-région avec résidence des données (Canada pour marchands canadiens)  
> - Tags automatiques : PII, FINANCIAL, MERCHANT_CONFIDENTIAL  
> - Politiques ABAC appliquées à la découverte des tags  
> - Effacement distribué avec certificats de conformité générés automatiquement  
> *Résultats* :  
> - SLA de 72h pour les demandes d'effacement (vs 30 jours légaux)  
> - Traitement de 50 000+ demandes de droits/mois entièrement automatisé  
> - Audit de conformité RGPD/Loi 25 passé sans recommandation majeure  
> - Réduction de 80 % de l'équipe dédiée à la conformité manuelle

---

## IV.13.7 Résumé

Ce chapitre a présenté une approche holistique de la sécurité, de la gouvernance et de la conformité du Data Lakehouse. Les points essentiels à retenir sont structurés ci-dessous.

### Architecture de Sécurité Multicouche

La sécurité du Lakehouse repose sur quatre couches complémentaires. Le **stockage objet** assure le chiffrement au repos et en transit, l'isolation réseau et les politiques d'accès de base. Le **format de table Iceberg** permet le chiffrement natif des fichiers Parquet avec contrôle par colonne. Le **catalogue** (Polaris, Unity Catalog, Nessie) centralise le RBAC et le credential vending pour une gouvernance cohérente. Les **moteurs de calcul** implémentent les contrôles fins (RLS, masquage) qui nécessitent l'accès aux données.

La séparation des responsabilités entre ces couches permet une défense en profondeur. Une compromission à un niveau est contenue par les contrôles des autres niveaux.

### Contrôles d'Accès Granulaires

Le RBAC au niveau du catalogue constitue le fondement, contrôlant l'accès aux objets (tables, namespaces). Apache Polaris et Unity Catalog offrent des implémentations matures avec credential vending, éliminant le besoin de distribuer des identifiants permanents.

Le Row-Level Security filtre automatiquement les données visibles selon l'identité de l'utilisateur. Cette approche permet de partager une même table entre équipes avec des périmètres de données différents.

Le Column Masking protège les colonnes sensibles en affichant des valeurs masquées aux utilisateurs non autorisés. Plusieurs patterns de masquage (préservation de format, tokenisation, généralisation) permettent de préserver l'utilité analytique tout en protégeant la confidentialité.

L'ABAC à grande échelle permet de définir des politiques basées sur des tags plutôt que sur des objets individuels. Cette approche simplifie considérablement la gestion pour les organisations avec des milliers de tables.

### Gouvernance des Données

OpenLineage s'impose comme le standard pour la collecte du lignage à travers les outils et plateformes. L'intégration avec Spark, Flink et Airflow permet de capturer automatiquement les transformations et leurs dépendances.

La qualité des données doit être validée automatiquement à chaque étape du pipeline. Les frameworks comme Great Expectations permettent de définir des « attentes » déclaratives qui sont vérifiées à chaque exécution.

Le catalogage métier complète le catalogue technique en documentant le contexte, la propriété et la classification des données. Cette documentation est essentielle pour la transparence requise par les réglementations.

### Conformité Canadienne

La Loi 25 du Québec impose des exigences strictes de consentement explicite, droit à l'effacement et portabilité. Ces exigences nécessitent des pipelines automatisés et une gestion rigoureuse des snapshots Iceberg. L'expiration des snapshots après suppression garantit l'effectivité du droit à l'effacement.

La LPRPDE fédérale établit les dix principes fondamentaux que l'architecture doit supporter. Bien que moins stricte que la Loi 25, elle s'applique à toutes les organisations canadiennes dans le secteur privé.

Les directives BSIF (B-13, E-21) ajoutent des exigences de résilience et de gestion des tiers pour les institutions financières. La classification des données selon leur criticité et leur sensibilité guide les contrôles à appliquer.

### Audit et Réponse aux Incidents

La journalisation des accès à plusieurs niveaux (stockage, catalogue, calcul) permet de reconstituer l'activité complète lors d'investigations. L'unification de ces journaux dans une table Iceberg dédiée simplifie l'analyse.

Les alertes proactives détectent les comportements anormaux avant qu'ils ne causent des dommages significatifs. Les patterns à surveiller incluent les accès hors heures normales, les volumes anormaux et les tentatives répétées refusées.

Le framework de réponse aux incidents exploite les capacités uniques du Lakehouse. Le time travel permet d'examiner l'état des données à tout moment. Les tags préservent des points de référence pour l'investigation. La restauration depuis un snapshot antérieur permet de remédier aux modifications non autorisées.

### Recommandations Stratégiques

Pour les organisations qui construisent ou modernisent leur Lakehouse, les recommandations suivantes guident l'approche :

1. **Adopter le credential vending** via un catalogue moderne (Polaris, Unity Catalog) pour éliminer les identifiants permanents distribués aux moteurs de calcul

2. **Implémenter la classification automatique** des données dès l'ingestion pour appliquer les contrôles appropriés sans intervention manuelle

3. **Automatiser les processus de conformité** (effacement, portabilité, audit) pour respecter les délais légaux et réduire les coûts opérationnels

4. **Unifier les journaux d'audit** des différentes couches (stockage, catalogue, calcul) dans une table Iceberg dédiée pour les investigations

5. **Documenter les métadonnées métier** dans un catalogue accessible pour permettre la découverte et démontrer la transparence requise par les réglementations

6. **Expirer proactivement les snapshots** contenant des données supprimées pour garantir l'effectivité du droit à l'effacement

7. **Déployer dans les régions canadiennes** (ca-central-1, ca-west-1 pour AWS; Canada Central, Canada East pour Azure) pour assurer la souveraineté des données et simplifier la conformité

8. **Établir un plan de réponse aux incidents** documenté et testé, exploitant les capacités de time travel et d'audit du Lakehouse

9. **Former les équipes** aux bonnes pratiques de sécurité et aux exigences réglementaires spécifiques à leur rôle

10. **Réviser périodiquement** les politiques d'accès et les classifications de données pour s'assurer qu'elles restent alignées avec les besoins métier et les exigences réglementaires

La sécurité et la gouvernance ne sont pas des projets ponctuels mais des pratiques continues. L'évolution constante des réglementations (modernisation de la LPRPDE, nouvelles directives BSIF) et des menaces de sécurité exige une architecture adaptable et une surveillance vigilante. Le Lakehouse, par sa nature ouverte et sa traçabilité inhérente via les snapshots Iceberg, offre une fondation solide pour cette gouvernance évolutive.

---

## Références

1. Apache Polaris (2025). *Documentation officielle*. polaris.apache.org

2. Atlan (2025). *Apache Iceberg Tables Governance: A Practical Guide*. Documentation technique.

3. Dremio (2025). *Securing Your Apache Iceberg Data Lakehouse*. Blog technique.

4. Dremio (2025). *Credential Vending with Iceberg REST Catalogs*. Blog technique.

5. OpenLineage (2025). *An Open Standard for Data Lineage*. Documentation officielle.

6. Databricks (2025). *Row Filters and Column Masks in Unity Catalog*. Documentation technique.

7. AWS (2025). *Governance and Access Control for Apache Iceberg on AWS*. Prescriptive Guidance.

8. Commission d'accès à l'information du Québec (2024). *Loi 25 - Guide de mise en œuvre*. Publication officielle.

9. Bureau du surintendant des institutions financières (2024). *Ligne directrice B-13 : Gestion du risque lié aux technologies*. Directive réglementaire.

10. Office of the Privacy Commissioner of Canada (2024). *PIPEDA in Brief*. Guide de conformité.
