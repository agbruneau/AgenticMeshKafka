# Chapitre III.10 - Organisation d'un Projet Kafka

---

## Introduction

La réussite d'un projet Kafka ne repose pas uniquement sur la maîtrise technique de la plateforme. Elle dépend tout autant de la rigueur organisationnelle avec laquelle l'équipe définit ses exigences, structure son infrastructure et valide ses développements. Trop souvent, les organisations abordent Kafka comme un simple composant technique à déployer, négligeant les dimensions méthodologiques qui conditionnent le succès à long terme.

Ce chapitre adopte la perspective de l'architecte responsable de l'organisation globale d'un projet Kafka. Il couvre trois dimensions fondamentales : la définition rigoureuse des exigences qui guideront les décisions architecturales, l'adoption de pratiques GitOps pour maintenir l'infrastructure comme code, et l'établissement d'une stratégie de tests adaptée aux spécificités des systèmes de streaming événementiel.

L'enjeu est considérable. Un projet Kafka mal organisé accumule rapidement une dette technique qui compromet sa capacité à évoluer. Les topics prolifèrent sans gouvernance, les configurations divergent entre environnements, et les régressions passent inaperçues jusqu'à la production. À l'inverse, une organisation méthodique transforme Kafka en actif stratégique durable, capable d'absorber la croissance et de s'adapter aux nouveaux besoins métier.

---

## III.10.1 Définition des Exigences d'un Projet Kafka

### III.10.1.1 La Taxonomie des Exigences Kafka

La définition des exigences d'un projet Kafka requiert une approche structurée qui distingue plusieurs catégories interdépendantes. Contrairement à un projet applicatif traditionnel, les exigences Kafka concernent simultanément les flux de données, les garanties de livraison, les contraintes de performance et les impératifs de gouvernance.

**Exigences fonctionnelles de flux**

Les exigences fonctionnelles décrivent les flux de données que le système doit supporter. Elles répondent aux questions fondamentales : quelles données circulent, entre quels systèmes, et selon quelle logique métier ?

| Dimension | Questions clés | Exemple |
|-----------|---------------|---------|
| Sources | Quels systèmes produisent des événements ? | ERP, CRM, IoT, applications web |
| Destinations | Quels systèmes consomment ces événements ? | Data warehouse, microservices, alerting |
| Transformation | Quelle logique de traitement intermédiaire ? | Enrichissement, agrégation, filtrage |
| Temporalité | Événements temps réel ou batch ? | Streaming continu vs micro-batch horaire |

> **Décision architecturale**  
> *Contexte* : Un projet d'intégration bancaire hésite entre modéliser les transactions comme événements atomiques ou comme agrégats par compte.  
> *Options* : (1) Événements granulaires par transaction, (2) Événements agrégés par compte/période.  
> *Décision* : Événements granulaires avec agrégation côté consommateur. Cette approche préserve la flexibilité et permet des cas d'usage non anticipés, au prix d'un volume plus élevé compensé par la compression Kafka.

**Exigences non fonctionnelles**

Les exigences non fonctionnelles définissent les caractéristiques qualitatives du système. Pour Kafka, elles se déclinent en plusieurs axes critiques :

*Volumétrie et débit* : Le dimensionnement du cluster dépend directement des volumes anticipés. L'architecte doit quantifier :
- Le débit moyen en messages par seconde
- Les pics de charge et leur fréquence
- La taille moyenne des messages
- Le taux de croissance annuel prévu

*Latence* : Les contraintes de latence varient considérablement selon les cas d'usage. Un système de détection de fraude exige une latence de bout en bout inférieure à 100 millisecondes, tandis qu'une synchronisation de référentiels tolère plusieurs secondes.

*Disponibilité* : Le niveau de disponibilité requis influence directement l'architecture de réplication et les stratégies de basculement. Un SLA de 99,99 % impose une architecture multi-datacenter avec réplication synchrone.

*Rétention* : La durée de conservation des événements répond à des besoins techniques (rejeu, reprise) et réglementaires (audit, conformité). Cette exigence impacte directement les coûts de stockage.

```
Exemple de spécification non fonctionnelle :

NFR-001: Débit
- Débit nominal : 50 000 messages/seconde
- Pic maximal : 200 000 messages/seconde (Black Friday)
- Durée des pics : 4 heures maximum

NFR-002: Latence
- P50 : < 10 ms
- P99 : < 100 ms
- P99.9 : < 500 ms

NFR-003: Disponibilité
- SLA cible : 99.95%
- RTO : 15 minutes
- RPO : 0 (aucune perte de données)

NFR-004: Rétention
- Topics opérationnels : 7 jours
- Topics d'audit : 2 ans (tiered storage)
```

**Exigences de gouvernance**

Les exigences de gouvernance encadrent l'utilisation de la plateforme à l'échelle de l'organisation. Elles concernent :

*Nomenclature* : Les conventions de nommage des topics, des groupes de consommateurs et des connecteurs. Une nomenclature rigoureuse facilite l'opération et la compréhension du système.

*Ownership* : L'attribution claire des responsabilités pour chaque topic. Qui peut créer, modifier, supprimer ? Qui est responsable de la qualité des données ?

*Évolution des schémas* : Les règles de compatibilité et le processus d'approbation pour les modifications de schémas.

*Accès et sécurité* : Les politiques d'authentification, d'autorisation et de chiffrement.

### III.10.1.2 Le Processus de Collecte des Exigences

La collecte des exigences Kafka mobilise plusieurs parties prenantes aux perspectives complémentaires. L'architecte orchestre ce processus en facilitant le dialogue entre domaines métier, équipes techniques et opérations.

**Phase 1 : Découverte des flux métier**

La première phase identifie les flux de données du point de vue métier. Les techniques d'Event Storming, décrites au chapitre précédent, s'avèrent particulièrement efficaces. L'objectif est de cartographier :

- Les événements métier significatifs
- Les acteurs qui les produisent et les consomment
- Les dépendances temporelles et causales
- Les invariants métier à respecter

> **Note de terrain**  
> *Contexte* : Projet de refonte du système de commandes d'un détaillant québécois.  
> *Défi* : Les équipes métier et techniques utilisaient des vocabulaires incompatibles. "Commande" désignait tantôt l'intention d'achat, tantôt la transaction confirmée.  
> *Solution* : Atelier d'Event Storming de deux jours avec glossaire partagé. Distinction formelle entre OrderPlaced, OrderConfirmed, OrderShipped.  
> *Leçon* : Investir dans l'alignement sémantique avant de modéliser les topics évite des refactorisations coûteuses.

**Phase 2 : Quantification technique**

La deuxième phase traduit les flux métier en métriques techniques. Cette quantification requiert une collaboration étroite avec les équipes applicatives :

| Flux métier | Volume estimé | Taille message | Pic/nominal | Latence requise |
|-------------|---------------|----------------|-------------|-----------------|
| Transactions POS | 10 000/min | 2 Ko | 5x | < 50 ms |
| Mises à jour inventaire | 1 000/min | 500 octets | 3x | < 1 s |
| Événements navigation web | 100 000/min | 1 Ko | 10x | < 5 s |
| Alertes fraude | 100/min | 5 Ko | 2x | < 100 ms |

**Phase 3 : Analyse des contraintes**

La troisième phase identifie les contraintes qui limitent l'espace des solutions :

*Contraintes techniques* : Infrastructure existante, compétences disponibles, intégrations obligatoires avec des systèmes legacy.

*Contraintes organisationnelles* : Structure des équipes, processus de déploiement, cycles de release.

*Contraintes réglementaires* : Localisation des données, durées de rétention légales, exigences d'audit.

*Contraintes budgétaires* : Enveloppe disponible pour l'infrastructure, les licences et la formation.

**Phase 4 : Priorisation et arbitrage**

La quatrième phase arbitre entre exigences potentiellement contradictoires. L'architecte utilise des matrices de priorisation pour expliciter les compromis :

```
Matrice de priorisation (exemple) :

                    Importance haute    Importance basse
Urgence haute       P1 - Critique       P2 - Important
                    (Latence fraude)    (Rétention audit)
                    
Urgence basse       P3 - Planifié       P4 - Nice-to-have
                    (Multi-DC)          (Compression Zstd)
```

### III.10.1.3 La Documentation des Exigences

La documentation des exigences Kafka adopte un format structuré qui facilite la traçabilité et la validation. Le document d'exigences Kafka (Kafka Requirements Document, KRD) constitue l'artefact central du projet.

**Structure recommandée du KRD**

```markdown
# Kafka Requirements Document - [Nom du projet]

## 1. Contexte et objectifs
- Énoncé du problème
- Objectifs métier
- Périmètre du projet

## 2. Parties prenantes
- Sponsors
- Équipes contributrices
- Utilisateurs finaux

## 3. Exigences fonctionnelles
### 3.1 Catalogue des topics
| Topic | Description | Producteur | Consommateurs | Schéma |
|-------|-------------|------------|---------------|--------|

### 3.2 Flux de données
[Diagrammes de flux]

### 3.3 Règles métier
[Invariants, validations, transformations]

## 4. Exigences non fonctionnelles
### 4.1 Performance
### 4.2 Disponibilité
### 4.3 Sécurité
### 4.4 Rétention

## 5. Contraintes
### 5.1 Techniques
### 5.2 Organisationnelles
### 5.3 Réglementaires

## 6. Hypothèses et risques

## 7. Critères d'acceptation

## 8. Glossaire
```

> **Anti-patron**  
> Documenter les exigences dans des courriels ou des conversations Slack. Cette approche disperse l'information, rend la traçabilité impossible et garantit des incompréhensions lors des transitions d'équipe. Toute exigence validée doit être consignée dans le KRD versionné.

### III.10.1.4 La Validation des Exigences

La validation des exigences s'effectue selon plusieurs dimensions :

*Complétude* : Toutes les questions pertinentes ont-elles reçu une réponse ? Les cas limites sont-ils couverts ?

*Cohérence* : Les exigences sont-elles compatibles entre elles ? Une latence de 10 ms est-elle réaliste avec une rétention de 2 ans sur stockage économique ?

*Faisabilité* : Les exigences sont-elles techniquement réalisables avec les ressources disponibles ?

*Testabilité* : Chaque exigence peut-elle être vérifiée par un test objectif ?

*Traçabilité* : Chaque exigence est-elle reliée à un besoin métier identifié ?

La revue des exigences implique les parties prenantes techniques et métier. L'architecte anime cette revue en s'assurant que chaque exigence est comprise, acceptée et réalisable.

---

## III.10.2 Maintenir la Structure du Cluster : Outils et GitOps

### III.10.2.1 L'Impératif de l'Infrastructure comme Code

La gestion manuelle d'un cluster Kafka devient rapidement intenable à mesure que le système croît. La multiplication des topics, des ACL, des quotas et des configurations crée une complexité qui dépasse les capacités de gestion ad hoc. L'approche Infrastructure as Code (IaC) répond à ce défi en traitant la configuration Kafka comme du code source versionné.

**Les bénéfices de l'IaC pour Kafka**

| Bénéfice | Description | Impact |
|----------|-------------|--------|
| Reproductibilité | Environnements identiques dev/staging/prod | Réduction des bugs "ça marche sur ma machine" |
| Auditabilité | Historique complet des modifications | Conformité et analyse des incidents |
| Revue par les pairs | Pull requests pour les changements | Qualité et partage des connaissances |
| Automatisation | Déploiements sans intervention manuelle | Vélocité et réduction des erreurs humaines |
| Rollback | Retour à un état antérieur facilité | Résilience face aux erreurs |

**Ce qui doit être versionné**

L'ensemble de la configuration Kafka doit être géré comme code :

```yaml
# Structure recommandée du dépôt GitOps Kafka

kafka-gitops/
├── README.md
├── environments/
│   ├── dev/
│   │   ├── cluster.yaml
│   │   ├── topics/
│   │   ├── acls/
│   │   └── quotas/
│   ├── staging/
│   │   └── ...
│   └── prod/
│       └── ...
├── schemas/
│   ├── events/
│   │   ├── order-placed.avsc
│   │   └── payment-processed.avsc
│   └── commands/
├── connectors/
│   ├── source/
│   └── sink/
├── pipelines/
│   └── ci-cd.yaml
└── docs/
    └── runbooks/
```

### III.10.2.2 Outils de Gestion GitOps pour Kafka

Plusieurs outils permettent d'implémenter l'approche GitOps pour Kafka. Le choix dépend du contexte organisationnel et des fonctionnalités requises.

**Julie Kafka GitOps (anciennement Kafka Topology Builder)**

Julie est l'outil open source de référence pour la gestion déclarative de Kafka. Développé par Purbon (Pere Urbón), il permet de définir la topologie complète du cluster dans des fichiers YAML.

```yaml
# Exemple de topologie Julie

context: "production"
company: "acme-corp"

projects:
  - name: "orders"
    consumers:
      - principal: "User:order-processor"
        group: "order-processing-group"
        topics:
          - "orders.created"
          - "orders.confirmed"
    producers:
      - principal: "User:order-service"
        topics:
          - "orders.created"
    topics:
      - name: "orders.created"
        config:
          retention.ms: "604800000"  # 7 jours
          partitions: 12
          replication.factor: 3
      - name: "orders.confirmed"
        config:
          retention.ms: "2592000000"  # 30 jours
          partitions: 12
          replication.factor: 3

  - name: "payments"
    consumers:
      - principal: "User:payment-processor"
        group: "payment-processing-group"
        topics:
          - "payments.initiated"
    producers:
      - principal: "User:payment-gateway"
        topics:
          - "payments.initiated"
    topics:
      - name: "payments.initiated"
        config:
          retention.ms: "2592000000"
          partitions: 6
          replication.factor: 3
```

Julie génère automatiquement les ACL correspondant à la topologie déclarée, garantissant la cohérence entre les permissions et les usages déclarés.

> **Note de terrain**  
> *Contexte* : Migration d'une configuration Kafka manuelle vers Julie dans une institution financière.  
> *Défi* : Plus de 200 topics existants avec des configurations hétérogènes et des ACL incohérentes.  
> *Solution* : Export de la configuration existante, normalisation progressive sur 3 sprints, validation en staging avant application en production.  
> *Leçon* : La migration vers GitOps est un projet en soi. Prévoir du temps pour l'archéologie de configuration et la normalisation.

**Confluent for Kubernetes (CFK)**

Pour les déploiements Kubernetes, Confluent for Kubernetes propose une approche native via des Custom Resource Definitions (CRD). Cette solution s'intègre naturellement aux pratiques GitOps Kubernetes existantes.

```yaml
# Exemple de Topic CRD pour CFK

apiVersion: platform.confluent.io/v1beta1
kind: KafkaTopic
metadata:
  name: orders-created
  namespace: confluent
spec:
  replicas: 3
  partitionCount: 12
  configs:
    retention.ms: "604800000"
    cleanup.policy: "delete"
    min.insync.replicas: "2"
```

**Terraform avec le provider Kafka**

Terraform offre une approche unifiée pour gérer Kafka aux côtés d'autres ressources cloud. Le provider Kafka permet de déclarer topics, ACL et configurations.

```hcl
# Exemple Terraform pour Kafka

provider "kafka" {
  bootstrap_servers = ["kafka1:9092", "kafka2:9092", "kafka3:9092"]
  tls_enabled       = true
  sasl_mechanism    = "SCRAM-SHA-512"
  sasl_username     = var.kafka_username
  sasl_password     = var.kafka_password
}

resource "kafka_topic" "orders_created" {
  name               = "orders.created"
  partitions         = 12
  replication_factor = 3

  config = {
    "retention.ms"        = "604800000"
    "cleanup.policy"      = "delete"
    "min.insync.replicas" = "2"
    "compression.type"    = "zstd"
  }
}

resource "kafka_acl" "order_service_producer" {
  resource_name       = "orders.created"
  resource_type       = "Topic"
  acl_principal       = "User:order-service"
  acl_host            = "*"
  acl_operation       = "Write"
  acl_permission_type = "Allow"
}
```

**Comparaison des outils**

| Critère | Julie | CFK | Terraform |
|---------|-------|-----|-----------|
| Courbe d'apprentissage | Moyenne | Élevée (K8s requis) | Faible si Terraform connu |
| Intégration CI/CD | Excellente | Native K8s | Excellente |
| Gestion des ACL | Automatique | Via CRD | Manuelle |
| Dry-run | Oui | Oui | Plan |
| Écosystème | Kafka pur | Confluent complet | Multi-cloud |
| Licence | Apache 2.0 | Confluent | MPL 2.0 |

> **Décision architecturale**  
> *Contexte* : Choix d'outil GitOps pour un déploiement Kafka on-premise.  
> *Options* : (1) Julie pour sa simplicité, (2) Terraform pour l'unification avec l'IaC existante.  
> *Décision* : Julie comme outil principal pour Kafka, intégré dans un pipeline Jenkins existant. Terraform réservé à l'infrastructure sous-jacente (VMs, réseau). Cette séparation des responsabilités simplifie la maintenance et permet aux équipes Kafka de travailler indépendamment.

### III.10.2.3 Pipeline CI/CD pour la Configuration Kafka

L'automatisation du déploiement de configuration Kafka suit un pipeline structuré qui garantit la qualité et la traçabilité.

**Étapes du pipeline**

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Commit    │───▶│  Validation │───▶│   Staging   │───▶│ Production  │
│   & Push    │    │   & Tests   │    │   Deploy    │    │   Deploy    │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
      │                   │                  │                  │
      ▼                   ▼                  ▼                  ▼
  Pull Request      Lint YAML          Apply config       Apply config
  créée            Validate schemas    Smoke tests        Monitoring
                   Dry-run Julie       Approbation        Alerting
```

**Exemple de pipeline GitLab CI**

```yaml
# .gitlab-ci.yml pour GitOps Kafka

stages:
  - validate
  - test
  - deploy-staging
  - deploy-production

variables:
  JULIE_VERSION: "4.0.0"

validate-yaml:
  stage: validate
  script:
    - yamllint environments/
    - python scripts/validate_naming_conventions.py
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'

validate-schemas:
  stage: validate
  script:
    - |
      for schema in schemas/**/*.avsc; do
        java -jar avro-tools.jar compile schema $schema /tmp/
      done
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'

dry-run:
  stage: test
  script:
    - |
      java -jar julie-ops.jar \
        --brokers $STAGING_BROKERS \
        --topology environments/staging/topology.yaml \
        --dry-run
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'

deploy-staging:
  stage: deploy-staging
  script:
    - |
      java -jar julie-ops.jar \
        --brokers $STAGING_BROKERS \
        --topology environments/staging/topology.yaml
    - ./scripts/smoke_tests.sh staging
  environment:
    name: staging
  rules:
    - if: '$CI_COMMIT_BRANCH == "main"'

deploy-production:
  stage: deploy-production
  script:
    - |
      java -jar julie-ops.jar \
        --brokers $PROD_BROKERS \
        --topology environments/prod/topology.yaml
  environment:
    name: production
  rules:
    - if: '$CI_COMMIT_BRANCH == "main"'
  when: manual
  allow_failure: false
```

### III.10.2.4 Gestion des Schémas avec GitOps

La gestion des schémas Avro, Protobuf ou JSON Schema s'intègre naturellement dans l'approche GitOps. Le Schema Registry devient un consommateur des schémas versionnés dans Git.

**Stratégie de versionnement des schémas**

Les schémas suivent une stratégie de versionnement sémantique adaptée :

- **Patch** (1.0.x) : Ajout de champs optionnels avec valeur par défaut
- **Minor** (1.x.0) : Ajout de champs optionnels sans défaut (compatible en lecture)
- **Major** (x.0.0) : Changements incompatibles (nouveau topic requis)

```avro
// schemas/events/order-placed-v2.avsc
{
  "type": "record",
  "name": "OrderPlaced",
  "namespace": "com.acme.orders.events",
  "doc": "Événement émis lors de la création d'une commande",
  "fields": [
    {
      "name": "orderId",
      "type": "string",
      "doc": "Identifiant unique de la commande"
    },
    {
      "name": "customerId",
      "type": "string",
      "doc": "Identifiant du client"
    },
    {
      "name": "orderDate",
      "type": {
        "type": "long",
        "logicalType": "timestamp-millis"
      },
      "doc": "Date de création de la commande"
    },
    {
      "name": "totalAmount",
      "type": {
        "type": "bytes",
        "logicalType": "decimal",
        "precision": 10,
        "scale": 2
      },
      "doc": "Montant total de la commande"
    },
    {
      "name": "currency",
      "type": "string",
      "default": "CAD",
      "doc": "Devise (ajouté en v2)"
    },
    {
      "name": "metadata",
      "type": ["null", {
        "type": "map",
        "values": "string"
      }],
      "default": null,
      "doc": "Métadonnées additionnelles (ajouté en v2)"
    }
  ]
}
```

**Automatisation de l'enregistrement des schémas**

```bash
#!/bin/bash
# scripts/register_schemas.sh

SCHEMA_REGISTRY_URL="${1:-http://localhost:8081}"
SCHEMAS_DIR="schemas"

for schema_file in $(find $SCHEMAS_DIR -name "*.avsc"); do
    # Extraire le sujet du chemin
    subject=$(echo $schema_file | sed 's/schemas\///' | sed 's/\.avsc$//' | tr '/' '-')
    
    echo "Enregistrement du schéma: $subject"
    
    # Vérifier la compatibilité
    compatibility_result=$(curl -s -X POST \
        -H "Content-Type: application/vnd.schemaregistry.v1+json" \
        --data "{\"schema\": $(cat $schema_file | jq -Rs .)}" \
        "$SCHEMA_REGISTRY_URL/compatibility/subjects/$subject-value/versions/latest")
    
    if echo "$compatibility_result" | grep -q '"is_compatible":false'; then
        echo "ERREUR: Schéma incompatible pour $subject"
        exit 1
    fi
    
    # Enregistrer le schéma
    curl -s -X POST \
        -H "Content-Type: application/vnd.schemaregistry.v1+json" \
        --data "{\"schema\": $(cat $schema_file | jq -Rs .)}" \
        "$SCHEMA_REGISTRY_URL/subjects/$subject-value/versions"
    
    echo ""
done
```

### III.10.2.5 Gouvernance et Conventions

L'approche GitOps requiert des conventions claires pour maintenir la cohérence à l'échelle de l'organisation.

**Convention de nommage des topics**

```
<domaine>.<entité>.<action>.<version>

Exemples:
- orders.order.created.v1
- payments.payment.processed.v1
- inventory.stock.updated.v1
- notifications.email.sent.v1
```

**Convention de nommage des groupes de consommateurs**

```
<application>-<environnement>-<fonction>

Exemples:
- order-processor-prod-main
- analytics-staging-realtime
- fraud-detection-prod-primary
```

**Fichier de configuration des conventions**

```yaml
# conventions.yaml

naming:
  topics:
    pattern: "^[a-z]+\\.[a-z-]+\\.[a-z]+\\.v[0-9]+$"
    segments:
      - name: domain
        allowed: [orders, payments, inventory, users, notifications]
      - name: entity
        pattern: "[a-z-]+"
      - name: action
        allowed: [created, updated, deleted, processed, sent, received]
      - name: version
        pattern: "v[0-9]+"
  
  consumer_groups:
    pattern: "^[a-z-]+-[a-z]+-[a-z]+$"
    
  connectors:
    source_pattern: "^source-[a-z-]+-[a-z]+$"
    sink_pattern: "^sink-[a-z-]+-[a-z]+$"

defaults:
  topics:
    partitions: 6
    replication_factor: 3
    retention_ms: 604800000  # 7 jours
    
  acls:
    default_host: "*"
```

> **Anti-patron**  
> Permettre la création de topics directement via l'API Kafka sans passer par le processus GitOps. Cette pratique crée une dérive entre la configuration déclarée et l'état réel du cluster. Configurer `auto.create.topics.enable=false` et exiger que tout topic soit déclaré dans le dépôt Git.

---

## III.10.3 Tester les Applications Kafka

### III.10.3.1 La Pyramide des Tests pour Kafka

Les applications Kafka requièrent une stratégie de tests adaptée aux spécificités des systèmes distribués et asynchrones. La pyramide des tests traditionnelle s'enrichit de nouvelles catégories.

```
                    ┌───────────────┐
                    │   Tests E2E   │  ← Environnement complet
                    │   (5-10%)     │
                ┌───┴───────────────┴───┐
                │   Tests d'intégration │  ← Kafka embedded/Testcontainers
                │       (20-30%)        │
            ┌───┴───────────────────────┴───┐
            │       Tests de composant      │  ← TopologyTestDriver
            │          (30-40%)             │
        ┌───┴───────────────────────────────┴───┐
        │            Tests unitaires            │  ← Mocks, logique pure
        │              (30-40%)                 │
        └───────────────────────────────────────┘
```

**Tests unitaires**

Les tests unitaires valident la logique métier isolée des dépendances Kafka. Ils concernent :
- Les transformations de données
- Les validations de schémas
- Les règles métier pures
- Les fonctions de sérialisation/désérialisation

```java
// Test unitaire d'une transformation

public class OrderTransformerTest {

    private OrderTransformer transformer;
    
    @BeforeEach
    void setUp() {
        transformer = new OrderTransformer();
    }
    
    @Test
    void shouldCalculateTotalWithTax() {
        // Given
        OrderLine line1 = new OrderLine("PROD-001", 2, new BigDecimal("10.00"));
        OrderLine line2 = new OrderLine("PROD-002", 1, new BigDecimal("25.00"));
        Order order = new Order("ORD-123", Arrays.asList(line1, line2));
        
        // When
        OrderWithTotal result = transformer.calculateTotal(order, new BigDecimal("0.14975"));
        
        // Then
        assertThat(result.getSubtotal()).isEqualByComparingTo("45.00");
        assertThat(result.getTax()).isEqualByComparingTo("6.74");
        assertThat(result.getTotal()).isEqualByComparingTo("51.74");
    }
    
    @Test
    void shouldRejectOrderWithNegativeQuantity() {
        // Given
        OrderLine invalidLine = new OrderLine("PROD-001", -1, new BigDecimal("10.00"));
        Order order = new Order("ORD-123", Collections.singletonList(invalidLine));
        
        // When/Then
        assertThatThrownBy(() -> transformer.validate(order))
            .isInstanceOf(InvalidOrderException.class)
            .hasMessageContaining("quantity must be positive");
    }
}
```

**Tests de composant avec TopologyTestDriver**

Le `TopologyTestDriver` de Kafka Streams permet de tester les topologies sans démarrer de broker. Cette approche offre des tests rapides et déterministes.

```java
// Test de topologie Kafka Streams

public class OrderProcessingTopologyTest {

    private TopologyTestDriver testDriver;
    private TestInputTopic<String, OrderPlaced> inputTopic;
    private TestOutputTopic<String, OrderConfirmed> outputTopic;
    
    @BeforeEach
    void setUp() {
        // Configuration de test
        Properties props = new Properties();
        props.put(StreamsConfig.APPLICATION_ID_CONFIG, "test-app");
        props.put(StreamsConfig.BOOTSTRAP_SERVERS_CONFIG, "dummy:1234");
        props.put(StreamsConfig.DEFAULT_KEY_SERDE_CLASS_CONFIG, Serdes.String().getClass());
        
        // Création de la topologie
        Topology topology = new OrderProcessingTopology().buildTopology();
        testDriver = new TopologyTestDriver(topology, props);
        
        // Configuration des topics de test
        Serde<OrderPlaced> orderPlacedSerde = new SpecificAvroSerde<>();
        Serde<OrderConfirmed> orderConfirmedSerde = new SpecificAvroSerde<>();
        
        inputTopic = testDriver.createInputTopic(
            "orders.placed",
            Serdes.String().serializer(),
            orderPlacedSerde.serializer()
        );
        
        outputTopic = testDriver.createOutputTopic(
            "orders.confirmed",
            Serdes.String().deserializer(),
            orderConfirmedSerde.deserializer()
        );
    }
    
    @AfterEach
    void tearDown() {
        testDriver.close();
    }
    
    @Test
    void shouldConfirmValidOrder() {
        // Given
        OrderPlaced orderPlaced = OrderPlaced.newBuilder()
            .setOrderId("ORD-123")
            .setCustomerId("CUST-456")
            .setTotalAmount(new BigDecimal("100.00"))
            .setOrderDate(Instant.now())
            .build();
        
        // When
        inputTopic.pipeInput("ORD-123", orderPlaced);
        
        // Then
        assertThat(outputTopic.isEmpty()).isFalse();
        
        KeyValue<String, OrderConfirmed> result = outputTopic.readKeyValue();
        assertThat(result.key).isEqualTo("ORD-123");
        assertThat(result.value.getStatus()).isEqualTo("CONFIRMED");
        assertThat(result.value.getConfirmationDate()).isNotNull();
    }
    
    @Test
    void shouldRejectOrderExceedingLimit() {
        // Given
        OrderPlaced largeOrder = OrderPlaced.newBuilder()
            .setOrderId("ORD-789")
            .setCustomerId("CUST-456")
            .setTotalAmount(new BigDecimal("1000000.00"))
            .setOrderDate(Instant.now())
            .build();
        
        // When
        inputTopic.pipeInput("ORD-789", largeOrder);
        
        // Then
        assertThat(outputTopic.isEmpty()).isTrue();
        
        // Vérifier le topic de rejet
        TestOutputTopic<String, OrderRejected> rejectedTopic = 
            testDriver.createOutputTopic("orders.rejected", ...);
        assertThat(rejectedTopic.isEmpty()).isFalse();
    }
    
    @Test
    void shouldAggregateOrdersByCustomer() {
        // Given - Plusieurs commandes du même client
        inputTopic.pipeInput("ORD-001", createOrder("ORD-001", "CUST-123", "50.00"));
        inputTopic.pipeInput("ORD-002", createOrder("ORD-002", "CUST-123", "75.00"));
        inputTopic.pipeInput("ORD-003", createOrder("ORD-003", "CUST-456", "100.00"));
        
        // When - Lecture du store d'état
        KeyValueStore<String, CustomerStats> statsStore = 
            testDriver.getKeyValueStore("customer-stats-store");
        
        // Then
        CustomerStats customer123Stats = statsStore.get("CUST-123");
        assertThat(customer123Stats.getOrderCount()).isEqualTo(2);
        assertThat(customer123Stats.getTotalSpent()).isEqualByComparingTo("125.00");
        
        CustomerStats customer456Stats = statsStore.get("CUST-456");
        assertThat(customer456Stats.getOrderCount()).isEqualTo(1);
    }
}
```

### III.10.3.2 Tests d'Intégration avec Testcontainers

Les tests d'intégration valident le comportement avec un véritable broker Kafka. Testcontainers simplifie la gestion des conteneurs de test.

```java
// Test d'intégration avec Testcontainers

@Testcontainers
public class OrderServiceIntegrationTest {

    @Container
    static KafkaContainer kafka = new KafkaContainer(
        DockerImageName.parse("confluentinc/cp-kafka:7.5.0")
    );
    
    @Container
    static GenericContainer<?> schemaRegistry = new GenericContainer<>(
        DockerImageName.parse("confluentinc/cp-schema-registry:7.5.0")
    )
    .withExposedPorts(8081)
    .withEnv("SCHEMA_REGISTRY_HOST_NAME", "schema-registry")
    .withEnv("SCHEMA_REGISTRY_KAFKASTORE_BOOTSTRAP_SERVERS", 
        "PLAINTEXT://" + kafka.getNetworkAliases().get(0) + ":9092")
    .dependsOn(kafka)
    .withNetwork(kafka.getNetwork());
    
    private KafkaProducer<String, OrderPlaced> producer;
    private KafkaConsumer<String, OrderConfirmed> consumer;
    
    @BeforeEach
    void setUp() {
        // Configuration du producer
        Properties producerProps = new Properties();
        producerProps.put(ProducerConfig.BOOTSTRAP_SERVERS_CONFIG, 
            kafka.getBootstrapServers());
        producerProps.put(ProducerConfig.KEY_SERIALIZER_CLASS_CONFIG, 
            StringSerializer.class);
        producerProps.put(ProducerConfig.VALUE_SERIALIZER_CLASS_CONFIG, 
            KafkaAvroSerializer.class);
        producerProps.put("schema.registry.url", 
            "http://" + schemaRegistry.getHost() + ":" + schemaRegistry.getMappedPort(8081));
        
        producer = new KafkaProducer<>(producerProps);
        
        // Configuration du consumer
        Properties consumerProps = new Properties();
        consumerProps.put(ConsumerConfig.BOOTSTRAP_SERVERS_CONFIG, 
            kafka.getBootstrapServers());
        consumerProps.put(ConsumerConfig.GROUP_ID_CONFIG, "test-consumer-group");
        consumerProps.put(ConsumerConfig.AUTO_OFFSET_RESET_CONFIG, "earliest");
        consumerProps.put(ConsumerConfig.KEY_DESERIALIZER_CLASS_CONFIG, 
            StringDeserializer.class);
        consumerProps.put(ConsumerConfig.VALUE_DESERIALIZER_CLASS_CONFIG, 
            KafkaAvroDeserializer.class);
        consumerProps.put("schema.registry.url", 
            "http://" + schemaRegistry.getHost() + ":" + schemaRegistry.getMappedPort(8081));
        consumerProps.put("specific.avro.reader", true);
        
        consumer = new KafkaConsumer<>(consumerProps);
        consumer.subscribe(Collections.singletonList("orders.confirmed"));
    }
    
    @AfterEach
    void tearDown() {
        producer.close();
        consumer.close();
    }
    
    @Test
    void shouldProcessOrderEndToEnd() {
        // Given
        OrderPlaced order = createTestOrder("ORD-INT-001", "CUST-001", "250.00");
        
        // When
        producer.send(new ProducerRecord<>("orders.placed", order.getOrderId(), order));
        producer.flush();
        
        // Then - Attendre le résultat avec timeout
        List<OrderConfirmed> results = new ArrayList<>();
        Awaitility.await()
            .atMost(Duration.ofSeconds(30))
            .pollInterval(Duration.ofMillis(500))
            .until(() -> {
                ConsumerRecords<String, OrderConfirmed> records = 
                    consumer.poll(Duration.ofMillis(100));
                records.forEach(record -> results.add(record.value()));
                return results.stream()
                    .anyMatch(r -> r.getOrderId().equals("ORD-INT-001"));
            });
        
        OrderConfirmed confirmed = results.stream()
            .filter(r -> r.getOrderId().equals("ORD-INT-001"))
            .findFirst()
            .orElseThrow();
        
        assertThat(confirmed.getStatus()).isEqualTo("CONFIRMED");
        assertThat(confirmed.getProcessedBy()).isEqualTo("order-processor");
    }
    
    @Test
    void shouldHandleSchemaEvolution() {
        // Given - Nouveau champ optionnel ajouté
        OrderPlacedV2 orderV2 = OrderPlacedV2.newBuilder()
            .setOrderId("ORD-V2-001")
            .setCustomerId("CUST-001")
            .setTotalAmount(new BigDecimal("100.00"))
            .setOrderDate(Instant.now())
            .setCurrency("USD")  // Nouveau champ
            .build();
        
        // When
        producer.send(new ProducerRecord<>("orders.placed", orderV2.getOrderId(), orderV2));
        producer.flush();
        
        // Then - Le consumer V1 doit ignorer le nouveau champ
        // et continuer à fonctionner normalement
        Awaitility.await()
            .atMost(Duration.ofSeconds(30))
            .until(() -> {
                ConsumerRecords<String, OrderConfirmed> records = 
                    consumer.poll(Duration.ofMillis(100));
                return records.count() > 0;
            });
    }
}
```

### III.10.3.3 Tests de Performance et de Charge

Les tests de performance valident que le système respecte les exigences non fonctionnelles de débit et de latence.

**Framework de test de charge**

```java
// Test de charge avec JMH (Java Microbenchmark Harness)

@BenchmarkMode(Mode.Throughput)
@OutputTimeUnit(TimeUnit.SECONDS)
@State(Scope.Benchmark)
@Fork(value = 1, warmups = 1)
@Warmup(iterations = 3, time = 10)
@Measurement(iterations = 5, time = 30)
public class KafkaProducerBenchmark {

    private KafkaProducer<String, byte[]> producer;
    private byte[] payload;
    
    @Param({"100", "1000", "10000"})
    private int payloadSize;
    
    @Param({"1", "10", "100"})
    private int batchSize;
    
    @Setup
    public void setup() {
        Properties props = new Properties();
        props.put(ProducerConfig.BOOTSTRAP_SERVERS_CONFIG, "kafka:9092");
        props.put(ProducerConfig.KEY_SERIALIZER_CLASS_CONFIG, StringSerializer.class);
        props.put(ProducerConfig.VALUE_SERIALIZER_CLASS_CONFIG, ByteArraySerializer.class);
        props.put(ProducerConfig.BATCH_SIZE_CONFIG, batchSize * 1024);
        props.put(ProducerConfig.LINGER_MS_CONFIG, 5);
        props.put(ProducerConfig.ACKS_CONFIG, "all");
        
        producer = new KafkaProducer<>(props);
        payload = new byte[payloadSize];
        new Random().nextBytes(payload);
    }
    
    @Benchmark
    public void measureThroughput(Blackhole blackhole) {
        RecordMetadata metadata = producer.send(
            new ProducerRecord<>("benchmark-topic", UUID.randomUUID().toString(), payload)
        ).get();
        blackhole.consume(metadata);
    }
    
    @TearDown
    public void tearDown() {
        producer.close();
    }
}
```

**Script de test de charge avec kafka-producer-perf-test**

```bash
#!/bin/bash
# scripts/load_test.sh

BOOTSTRAP_SERVERS="${1:-kafka:9092}"
TOPIC="${2:-load-test-topic}"
NUM_RECORDS="${3:-1000000}"
RECORD_SIZE="${4:-1024}"
THROUGHPUT="${5:--1}"  # -1 = pas de limite

# Création du topic de test
kafka-topics.sh --bootstrap-server $BOOTSTRAP_SERVERS \
    --create --topic $TOPIC \
    --partitions 12 \
    --replication-factor 3 \
    --config retention.ms=3600000

# Test de production
echo "=== Test de production ==="
kafka-producer-perf-test.sh \
    --topic $TOPIC \
    --num-records $NUM_RECORDS \
    --record-size $RECORD_SIZE \
    --throughput $THROUGHPUT \
    --producer-props \
        bootstrap.servers=$BOOTSTRAP_SERVERS \
        acks=all \
        batch.size=16384 \
        linger.ms=5 \
        compression.type=lz4

# Test de consommation
echo "=== Test de consommation ==="
kafka-consumer-perf-test.sh \
    --bootstrap-server $BOOTSTRAP_SERVERS \
    --topic $TOPIC \
    --messages $NUM_RECORDS \
    --threads 4

# Nettoyage
kafka-topics.sh --bootstrap-server $BOOTSTRAP_SERVERS \
    --delete --topic $TOPIC
```

### III.10.3.4 Tests de Résilience et de Chaos

Les tests de résilience valident le comportement du système face aux pannes. L'ingénierie du chaos applique des perturbations contrôlées pour découvrir les faiblesses.

**Scénarios de chaos pour Kafka**

| Scénario | Description | Validation attendue |
|----------|-------------|---------------------|
| Perte d'un broker | Arrêt brutal d'un broker | Réélection du leader, continuité du service |
| Partition réseau | Isolation d'un broker | Shrink ISR, messages non perdus |
| Disque plein | Saturation du stockage | Alerting, rejection propre |
| Latence réseau | Injection de latence | Timeout géré, retry efficace |
| Schema Registry down | Indisponibilité du registry | Cache local, dégradation gracieuse |

```java
// Test de résilience avec Toxiproxy

@Testcontainers
public class KafkaResilienceTest {

    @Container
    static ToxiproxyContainer toxiproxy = new ToxiproxyContainer(
        DockerImageName.parse("ghcr.io/shopify/toxiproxy:2.5.0")
    ).withNetwork(network);
    
    @Container
    static KafkaContainer kafka = new KafkaContainer(
        DockerImageName.parse("confluentinc/cp-kafka:7.5.0")
    ).withNetwork(network);
    
    private ToxiproxyClient toxiproxyClient;
    private Proxy kafkaProxy;
    
    @BeforeEach
    void setUp() throws IOException {
        toxiproxyClient = new ToxiproxyClient(
            toxiproxy.getHost(), 
            toxiproxy.getControlPort()
        );
        
        kafkaProxy = toxiproxyClient.createProxy(
            "kafka",
            "0.0.0.0:9093",
            kafka.getNetworkAliases().get(0) + ":9092"
        );
    }
    
    @Test
    void shouldHandleNetworkLatency() throws Exception {
        // Given - Producer configuré avec timeout
        Properties props = createProducerProps();
        props.put(ProducerConfig.REQUEST_TIMEOUT_MS_CONFIG, 5000);
        props.put(ProducerConfig.DELIVERY_TIMEOUT_MS_CONFIG, 10000);
        
        KafkaProducer<String, String> producer = new KafkaProducer<>(props);
        
        // When - Injection de latence
        kafkaProxy.toxics()
            .latency("latency-toxic", ToxicDirection.DOWNSTREAM, 3000);
        
        // Then - Le producer doit gérer la latence
        long startTime = System.currentTimeMillis();
        RecordMetadata metadata = producer.send(
            new ProducerRecord<>("test-topic", "key", "value")
        ).get(15, TimeUnit.SECONDS);
        long duration = System.currentTimeMillis() - startTime;
        
        assertThat(duration).isGreaterThan(3000);
        assertThat(metadata.offset()).isNotNegative();
        
        // Cleanup
        kafkaProxy.toxics().get("latency-toxic").remove();
    }
    
    @Test
    void shouldRecoverFromConnectionReset() throws Exception {
        // Given
        KafkaProducer<String, String> producer = new KafkaProducer<>(createProducerProps());
        
        // Envoi initial réussi
        producer.send(new ProducerRecord<>("test-topic", "key1", "value1")).get();
        
        // When - Reset de connexion
        kafkaProxy.toxics()
            .resetPeer("reset-toxic", ToxicDirection.DOWNSTREAM, 0);
        
        // Attendre que le toxic s'applique
        Thread.sleep(100);
        kafkaProxy.toxics().get("reset-toxic").remove();
        
        // Then - Le producer doit se reconnecter
        Awaitility.await()
            .atMost(Duration.ofSeconds(30))
            .until(() -> {
                try {
                    producer.send(new ProducerRecord<>("test-topic", "key2", "value2"))
                        .get(5, TimeUnit.SECONDS);
                    return true;
                } catch (Exception e) {
                    return false;
                }
            });
    }
}
```

### III.10.3.5 Tests de Contrats et de Compatibilité

Les tests de contrats valident que les schémas respectent les règles de compatibilité définies.

```java
// Test de compatibilité de schéma

public class SchemaCompatibilityTest {

    private static final String SCHEMA_REGISTRY_URL = "http://localhost:8081";
    private CachedSchemaRegistryClient schemaRegistry;
    
    @BeforeEach
    void setUp() {
        schemaRegistry = new CachedSchemaRegistryClient(SCHEMA_REGISTRY_URL, 100);
    }
    
    @Test
    void shouldMaintainBackwardCompatibility() throws Exception {
        // Given - Schéma V1 existant
        String subject = "orders.placed-value";
        Schema schemaV1 = new Schema.Parser().parse(
            Files.readString(Path.of("schemas/events/order-placed-v1.avsc"))
        );
        
        // When - Enregistrement de V2
        Schema schemaV2 = new Schema.Parser().parse(
            Files.readString(Path.of("schemas/events/order-placed-v2.avsc"))
        );
        
        // Then - Vérifier la compatibilité
        boolean isCompatible = schemaRegistry.testCompatibility(subject, schemaV2);
        assertThat(isCompatible)
            .as("Le schéma V2 doit être rétrocompatible avec V1")
            .isTrue();
    }
    
    @Test
    void shouldRejectIncompatibleChange() throws Exception {
        // Given - Schéma existant avec champ requis
        String subject = "orders.placed-value";
        
        // When - Tentative de suppression d'un champ requis
        Schema incompatibleSchema = new Schema.Parser().parse("""
            {
              "type": "record",
              "name": "OrderPlaced",
              "fields": [
                {"name": "orderId", "type": "string"}
                // customerId supprimé - INCOMPATIBLE
              ]
            }
            """);
        
        // Then
        assertThatThrownBy(() -> 
            schemaRegistry.register(subject, incompatibleSchema)
        ).isInstanceOf(RestClientException.class)
         .hasMessageContaining("incompatible");
    }
    
    @ParameterizedTest
    @MethodSource("provideSchemaEvolutions")
    void shouldValidateSchemaEvolution(String oldSchemaPath, String newSchemaPath, 
                                        boolean expectedCompatible) throws Exception {
        Schema oldSchema = new Schema.Parser().parse(
            Files.readString(Path.of(oldSchemaPath))
        );
        Schema newSchema = new Schema.Parser().parse(
            Files.readString(Path.of(newSchemaPath))
        );
        
        SchemaCompatibility.SchemaPairCompatibility compatibility = 
            SchemaCompatibility.checkReaderWriterCompatibility(newSchema, oldSchema);
        
        boolean isCompatible = compatibility.getType() == 
            SchemaCompatibility.SchemaCompatibilityType.COMPATIBLE;
        
        assertThat(isCompatible).isEqualTo(expectedCompatible);
    }
    
    static Stream<Arguments> provideSchemaEvolutions() {
        return Stream.of(
            Arguments.of("v1.avsc", "v2_add_optional.avsc", true),
            Arguments.of("v1.avsc", "v2_add_required.avsc", false),
            Arguments.of("v1.avsc", "v2_remove_field.avsc", false),
            Arguments.of("v1.avsc", "v2_rename_field.avsc", false),
            Arguments.of("v1.avsc", "v2_change_type.avsc", false)
        );
    }
}
```

### III.10.3.6 Stratégie de Tests en Environnement

L'organisation des tests s'adapte aux différents environnements du cycle de développement.

**Matrice des tests par environnement**

| Type de test | Local | CI | Staging | Production |
|--------------|-------|-----|---------|------------|
| Unitaires | ✓ | ✓ | - | - |
| Composant (TopologyTestDriver) | ✓ | ✓ | - | - |
| Intégration (Testcontainers) | ✓ | ✓ | - | - |
| Performance (baseline) | - | ✓ | ✓ | - |
| Performance (charge) | - | - | ✓ | - |
| Chaos | - | - | ✓ | ✓ (contrôlé) |
| Smoke tests | - | - | ✓ | ✓ |
| Canary | - | - | - | ✓ |

**Configuration des tests CI**

```yaml
# .github/workflows/kafka-tests.yml

name: Kafka Application Tests

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-java@v4
        with:
          java-version: '21'
          distribution: 'temurin'
      - name: Run unit tests
        run: ./gradlew test -x integrationTest
      - name: Upload coverage
        uses: codecov/codecov-action@v3

  integration-tests:
    runs-on: ubuntu-latest
    needs: unit-tests
    services:
      # Services gérés par Testcontainers
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-java@v4
        with:
          java-version: '21'
          distribution: 'temurin'
      - name: Run integration tests
        run: ./gradlew integrationTest
        env:
          TESTCONTAINERS_RYUK_DISABLED: true

  performance-baseline:
    runs-on: ubuntu-latest
    needs: integration-tests
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      - name: Run performance baseline
        run: ./scripts/perf_baseline.sh
      - name: Compare with previous baseline
        run: ./scripts/compare_perf.sh
      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: perf-results
          path: build/reports/performance/

  contract-tests:
    runs-on: ubuntu-latest
    needs: unit-tests
    steps:
      - uses: actions/checkout@v4
      - name: Validate schema compatibility
        run: ./scripts/validate_schemas.sh
```

> **Note de terrain**  
> *Contexte* : Projet de migration vers Kafka dans une entreprise de télécommunications.  
> *Défi* : Les tests d'intégration avec Testcontainers prenaient plus de 15 minutes en CI, ralentissant le feedback.  
> *Solution* : Parallélisation des tests par domaine métier, réutilisation des conteneurs entre tests du même groupe, et extraction des tests de charge vers un job séparé nocturne.  
> *Leçon* : La vitesse du feedback CI est critique pour l'adoption. Investir dans l'optimisation des tests paie rapidement.

### III.10.3.7 Monitoring des Tests en Production

Le déploiement en production ne marque pas la fin des tests. Le monitoring continu valide le comportement réel du système.

**Canary deployments**

```yaml
# Configuration de déploiement canary

apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: order-processor
spec:
  replicas: 10
  strategy:
    canary:
      steps:
        - setWeight: 10
        - pause: {duration: 5m}
        - analysis:
            templates:
              - templateName: kafka-success-rate
            args:
              - name: service-name
                value: order-processor
        - setWeight: 30
        - pause: {duration: 10m}
        - analysis:
            templates:
              - templateName: kafka-latency-check
        - setWeight: 60
        - pause: {duration: 15m}
        - setWeight: 100

---
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: kafka-success-rate
spec:
  metrics:
    - name: success-rate
      interval: 1m
      successCondition: result[0] >= 0.99
      provider:
        prometheus:
          address: http://prometheus:9090
          query: |
            sum(rate(kafka_consumer_records_consumed_total{
              service="{{args.service-name}}",
              status="success"
            }[5m])) /
            sum(rate(kafka_consumer_records_consumed_total{
              service="{{args.service-name}}"
            }[5m]))
```

**Smoke tests post-déploiement**

```java
// Smoke test exécuté après chaque déploiement

@SpringBootTest
@ActiveProfiles("smoke")
public class ProductionSmokeTest {

    @Autowired
    private KafkaTemplate<String, String> kafkaTemplate;
    
    @Autowired
    private KafkaConsumer<String, String> testConsumer;
    
    @Value("${smoke.test.topic}")
    private String smokeTestTopic;
    
    @Test
    @Timeout(60)
    void shouldProduceAndConsumeMessage() {
        // Given
        String testMessage = "smoke-test-" + UUID.randomUUID();
        
        // When
        kafkaTemplate.send(smokeTestTopic, testMessage).get();
        
        // Then
        testConsumer.subscribe(Collections.singletonList(smokeTestTopic));
        
        boolean messageReceived = false;
        Instant deadline = Instant.now().plusSeconds(30);
        
        while (Instant.now().isBefore(deadline) && !messageReceived) {
            ConsumerRecords<String, String> records = 
                testConsumer.poll(Duration.ofSeconds(1));
            
            for (ConsumerRecord<String, String> record : records) {
                if (record.value().equals(testMessage)) {
                    messageReceived = true;
                    break;
                }
            }
        }
        
        assertThat(messageReceived)
            .as("Le message de smoke test doit être reçu dans les 30 secondes")
            .isTrue();
    }
    
    @Test
    void shouldConnectToSchemaRegistry() {
        // Validation de la connectivité Schema Registry
        RestTemplate restTemplate = new RestTemplate();
        ResponseEntity<String> response = restTemplate.getForEntity(
            schemaRegistryUrl + "/subjects",
            String.class
        );
        
        assertThat(response.getStatusCode()).isEqualTo(HttpStatus.OK);
    }
}
```

---

## III.10.4 Résumé

Ce chapitre a exploré les dimensions organisationnelles essentielles à la réussite d'un projet Kafka, complétant les aspects techniques abordés dans les chapitres précédents.

### Points clés à retenir

**Définition des exigences**

La rigueur dans la collecte et la documentation des exigences conditionne le succès du projet. Les exigences Kafka se déclinent en trois catégories interdépendantes : fonctionnelles (flux de données), non fonctionnelles (performance, disponibilité) et de gouvernance (conventions, ownership). Le Kafka Requirements Document (KRD) constitue l'artefact central qui guide les décisions architecturales et permet la traçabilité des choix.

| Catégorie | Éléments clés | Impact |
|-----------|--------------|--------|
| Fonctionnelles | Sources, destinations, transformations, temporalité | Design des topics et topologies |
| Non fonctionnelles | Débit, latence, disponibilité, rétention | Dimensionnement et architecture |
| Gouvernance | Nomenclature, ownership, évolution, sécurité | Opérabilité long terme |

**Infrastructure comme code et GitOps**

L'approche GitOps transforme la gestion du cluster Kafka en processus auditable et reproductible. Les outils comme Julie, Confluent for Kubernetes ou Terraform permettent de déclarer l'état souhaité du cluster dans des fichiers versionnés. Le pipeline CI/CD automatise la validation et le déploiement des configurations, éliminant les dérives entre environnements.

Les conventions de nommage et les politiques de gouvernance codifiées garantissent la cohérence à l'échelle de l'organisation. La désactivation de la création automatique de topics (`auto.create.topics.enable=false`) force le passage par le processus GitOps pour toute modification.

**Stratégie de tests**

Les applications Kafka requièrent une pyramide de tests enrichie qui combine :
- Tests unitaires pour la logique métier pure
- Tests de composant avec TopologyTestDriver pour les topologies Kafka Streams
- Tests d'intégration avec Testcontainers pour les interactions broker
- Tests de performance pour valider les exigences non fonctionnelles
- Tests de résilience et chaos engineering pour découvrir les faiblesses
- Tests de contrats pour garantir la compatibilité des schémas

La matrice des tests par environnement guide l'exécution : tests rapides en local et CI, tests de charge en staging, monitoring continu et canary deployments en production.

### Recommandations pratiques

1. **Investir dans la phase d'exigences** : Un KRD complet évite les refactorisations coûteuses. Prévoir des ateliers d'Event Storming avec les parties prenantes métier et techniques.

2. **Adopter GitOps dès le premier topic** : La migration vers GitOps sur un cluster existant est laborieuse. Démarrer avec cette approche simplifie la gouvernance future.

3. **Automatiser les tests de compatibilité** : Intégrer la validation des schémas dans le pipeline CI prévient les régressions de contrat qui impactent les consommateurs.

4. **Mesurer la vitesse du feedback** : Le temps entre un commit et le résultat des tests influence directement la productivité. Optimiser le pipeline CI est un investissement rentable.

5. **Planifier les tests de résilience** : Les pannes en production sont inévitables. Le chaos engineering contrôlé révèle les faiblesses avant qu'elles ne causent des incidents.

### Perspectives

L'organisation d'un projet Kafka évolue avec la maturité de l'équipe et la croissance du système. Les pratiques décrites dans ce chapitre constituent une fondation solide pour les déploiements initiaux. À mesure que le cluster s'étend, des considérations supplémentaires émergent : fédération de clusters, gestion multi-tenant, automatisation avancée avec des opérateurs Kubernetes.

Le chapitre suivant aborde les aspects opérationnels de Kafka en production : évolution du cluster, mobilité des données, surveillance et reprise après sinistre. Ces préoccupations opérationnelles complètent les pratiques organisationnelles établies ici pour assurer le succès durable de la plateforme événementielle.

---

*Fin du Chapitre III.10*