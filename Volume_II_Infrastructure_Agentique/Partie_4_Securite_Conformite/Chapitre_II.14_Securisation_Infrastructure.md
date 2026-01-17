# Chapitre II.14 — Sécurisation de l'Infrastructure

## Introduction

Le chapitre précédent a dressé un panorama alarmant des menaces ciblant les systèmes agentiques. Face à cette réalité, la sécurisation de l'infrastructure sous-jacente devient un impératif stratégique non négociable. Une architecture agentique repose sur deux piliers technologiques majeurs : le backbone événementiel Kafka qui orchestre les flux de données en temps réel, et la couche cognitive hébergée sur Google Cloud qui opérationnalise l'intelligence artificielle. Sécuriser ces fondations exige une approche holistique qui transcende les mesures ponctuelles pour établir une posture de défense en profondeur.

Ce chapitre détaille les mécanismes de sécurisation à chaque niveau de l'infrastructure. Nous explorerons d'abord les contrôles natifs de Confluent Platform pour protéger le backbone Kafka, puis examinerons la gestion des identités dans Google Cloud avec ses mécanismes modernes comme Workload Identity Federation. La sécurité réseau sera analysée à travers le prisme des VPC Service Controls et de la connectivité privée. Nous découvrirons ensuite les capacités de Security Command Center pour la protection des charges de travail IA. Enfin, nous établirons les fondations d'une traçabilité exhaustive via les journaux d'audit, condition sine qua non de la conformité réglementaire et de la réponse aux incidents.

---

## II.14.1 Sécurité du Backbone Kafka

### L'Impératif de Sécurisation du Système Nerveux Numérique

Apache Kafka constitue le système nerveux numérique de l'entreprise agentique, transportant des événements métier critiques entre systèmes et agents cognitifs. Par défaut, Kafka opère en mode permissif, autorisant un accès non restreint entre brokers et services externes. Cette configuration, acceptable en développement, représente un risque majeur en production. La sécurisation du backbone événementiel s'articule autour de trois piliers fondamentaux : l'authentification, l'autorisation et le chiffrement.

### Authentification : Vérifier l'Identité des Acteurs

L'authentification établit l'identité des clients et des brokers avant toute interaction avec le cluster. Kafka supporte plusieurs mécanismes, chacun adapté à des contextes spécifiques.

**TLS/SSL Client Authentication (mTLS)** constitue la méthode privilégiée pour les environnements de production. Le protocole mTLS assure une authentification bidirectionnelle : les clients vérifient l'identité des brokers via leurs certificats, et réciproquement. Cette approche élimine la nécessité de gérer des mots de passe tout en offrant une sécurité cryptographique robuste.

> **Note technique**  
> La configuration mTLS requiert la génération de keystores et truststores pour chaque composant. Utilisez des certificats signés par une autorité de certification interne plutôt que des certificats auto-signés pour faciliter la gestion à l'échelle.

**SASL (Simple Authentication and Security Layer)** offre une flexibilité accrue via plusieurs mécanismes :

| Mécanisme | Usage Recommandé | Considérations |
|-----------|------------------|----------------|
| SASL/GSSAPI (Kerberos) | Environnements entreprise avec infrastructure Kerberos existante | Intégration native avec Active Directory |
| SASL/SCRAM-SHA-512 | Clusters sans Kerberos, authentification par mot de passe | Stockage sécurisé des credentials dans ZooKeeper/KRaft |
| SASL/OAUTHBEARER | Intégration avec fournisseurs d'identité modernes (Okta, Entra ID) | Recommandé pour les architectures cloud-native |
| SASL/PLAIN | Développement uniquement | Jamais en production sans TLS |

Confluent Platform enrichit ces mécanismes natifs avec l'intégration LDAP via le Metadata Service (MDS), permettant une authentification centralisée alignée sur l'annuaire d'entreprise.

### Autorisation : Contrôler les Actions Permises

Une fois l'identité établie, l'autorisation détermine les opérations permises sur les ressources Kafka. Deux approches coexistent et peuvent être combinées.

**Access Control Lists (ACLs)** offrent un contrôle granulaire au niveau des ressources individuelles. Chaque ACL spécifie un principal (utilisateur ou groupe), une ressource (topic, groupe de consommateurs, cluster), une opération (READ, WRITE, CREATE, DELETE) et une décision (ALLOW ou DENY).

```bash
# Exemple : Autoriser l'agent de recommandation à consommer le topic events.customer
kafka-acls --bootstrap-server kafka:9092 \
  --add --allow-principal User:recommendation-agent \
  --operation READ --topic events.customer
```

> **Bonnes pratiques**  
> Adoptez une politique « deny by default » en production. Aucun accès n'est autorisé sans ACL explicite. Configurez les super-utilisateurs uniquement pour l'administration du cluster.

**Role-Based Access Control (RBAC)** simplifie la gestion à l'échelle via des rôles prédéfinis. Confluent Platform implémente RBAC via le Metadata Service, offrant plusieurs avantages :

- **Centralisation** : Un point unique pour gérer les autorisations de tous les clusters
- **Granularité** : Rôles applicables aux clusters, topics, groupes de consommateurs, connecteurs et sujets Schema Registry
- **Intégration LDAP** : Synchronisation automatique avec les groupes d'entreprise

Les rôles prédéfinis incluent :

| Rôle | Portée | Permissions |
|------|--------|-------------|
| ClusterAdmin | Cluster | Administration complète du cluster |
| Operator | Cluster | Opérations de maintenance sans modification des ACLs |
| ResourceOwner | Ressource | Contrôle total sur une ressource spécifique |
| DeveloperRead | Ressource | Lecture seule sur les topics et schémas |
| DeveloperWrite | Ressource | Lecture et écriture sur les topics |

### Chiffrement : Protéger les Données en Transit et au Repos

Le chiffrement assure la confidentialité des données à chaque étape de leur cycle de vie.

**Chiffrement en transit** via TLS protège les communications entre clients et brokers, entre brokers, et avec les composants écosystème (Schema Registry, Connect, ksqlDB). La configuration requiert la définition de listeners sécurisés et de protocoles inter-broker.

```properties
# Configuration broker pour TLS
listeners=SSL://:9093,SASL_SSL://:9094
security.inter.broker.protocol=SSL
ssl.keystore.location=/var/kafka/ssl/kafka.keystore.jks
ssl.keystore.password=${KEYSTORE_PASSWORD}
ssl.truststore.location=/var/kafka/ssl/kafka.truststore.jks
ssl.truststore.password=${TRUSTSTORE_PASSWORD}
ssl.client.auth=required
```

> **Attention**  
> Le chiffrement TLS impacte les performances (10-30% selon les configurations). Dimensionnez vos clusters en conséquence et utilisez des suites cryptographiques modernes (TLS 1.3) pour minimiser l'overhead.

**Chiffrement au repos** protège les données stockées sur les brokers. Confluent Cloud intègre le chiffrement transparent avec gestion des clés via AWS KMS, Azure Key Vault ou Google Cloud KMS. Pour les déploiements on-premises, le chiffrement au niveau du système de fichiers (LUKS, BitLocker) ou l'utilisation de solutions tierces s'impose.

### Sécurisation de l'Écosystème Confluent

La sécurité du backbone s'étend à tous les composants de la plateforme.

**Kafka Connect** nécessite une attention particulière car il interface des systèmes externes. Le Secret Registry de Confluent Platform permet de stocker les credentials des connecteurs de manière chiffrée, évitant leur exposition dans les configurations :

```properties
# Configuration Connect avec Secret Registry
config.providers=secret
config.providers.secret.class=io.confluent.connect.secretregistry.rbac.config.provider.InternalSecretConfigProvider
config.providers.secret.param.master.encryption.key=${MASTER_KEY}
```

**Schema Registry** requiert une protection équivalente car il centralise les contrats de données. L'intégration RBAC permet de contrôler qui peut enregistrer, modifier ou supprimer des schémas, préservant ainsi l'intégrité des contrats.

### Évolution vers KRaft : Implications Sécuritaires

L'abandon de ZooKeeper au profit du consensus KRaft (Kafka Raft) simplifie l'architecture mais modifie les considérations de sécurité. Historiquement, ZooKeeper stockait les métadonnées du cluster et représentait une cible critique — sa compromission permettait la manipulation des configurations, l'ajout de brokers malveillants ou la corruption des offsets.

KRaft intègre la gestion des métadonnées directement dans les brokers, éliminant ce composant externe. Cette consolidation offre plusieurs avantages sécuritaires :

- **Surface d'attaque réduite** : Un composant de moins à sécuriser et patcher
- **Authentification unifiée** : Plus de configuration séparée pour ZooKeeper
- **Contrôle d'accès simplifié** : Les ACLs s'appliquent uniformément via Kafka

Cependant, les contrôleurs KRaft deviennent désormais les gardiens des métadonnées. Leur isolation réseau et leur protection contre les accès non autorisés demeurent critiques.

### Architecture de Sécurité de Référence

Une implémentation de production combine ces éléments en une architecture cohérente :

1. **Listeners dédiés** : Séparer les listeners clients (SASL_SSL), inter-broker (SSL), contrôleur KRaft (SSL restreint) et administration (SASL_SSL avec restrictions IP)
2. **Authentification hybride** : mTLS pour les services, OAUTHBEARER pour les applications cloud-native
3. **RBAC avec LDAP** : Rôles alignés sur l'organisation, groupes synchronisés automatiquement
4. **Secrets externalisés** : HashiCorp Vault ou gestionnaire de secrets cloud pour toutes les credentials
5. **Monitoring sécurisé** : Métriques Confluent exportées via endpoints authentifiés
6. **Isolation des contrôleurs** : Sous-réseau dédié pour les nœuds contrôleur KRaft

---

## II.14.2 Gestion des Identités dans Google Cloud

### Le Défi de l'Identité dans les Architectures Agentiques

Les architectures agentiques introduisent une complexité identitaire sans précédent. Au-delà des utilisateurs humains traditionnels, le système doit authentifier et autoriser des agents cognitifs autonomes, des pipelines CI/CD, des services multi-cloud et des workloads éphémères. Google Cloud propose un modèle d'identité sophistiqué qui répond à ces exigences via une combinaison de comptes de service, de fédération d'identité et de contrôles d'accès granulaires.

### Comptes de Service : Identités pour les Workloads

Les comptes de service constituent le mécanisme fondamental pour attribuer une identité aux applications et workloads. Contrairement aux comptes utilisateur, ils sont conçus pour l'authentification programmatique et ne possèdent pas de mot de passe interactif.

Google Cloud distingue trois types de comptes de service :

| Type | Gestion | Usage |
|------|---------|-------|
| User-managed | Créés et gérés par l'organisation | Workloads applicatifs, agents cognitifs |
| Default | Créés automatiquement par certains services | À éviter en production (permissions trop larges) |
| Service agents | Gérés par Google | Actions internes des services Google Cloud |

> **Bonnes pratiques**  
> Créez un compte de service dédié par agent ou workload. Évitez les comptes de service partagés qui violent le principe de moindre privilège et compliquent l'audit. Désactivez les comptes de service par défaut dans vos projets.

L'attachement de comptes de service aux ressources (Compute Engine, Cloud Run, GKE) reste la méthode privilégiée car elle élimine la gestion de credentials. Le workload hérite automatiquement de l'identité du compte attaché via le serveur de métadonnées.

### Workload Identity Federation : Éliminer les Clés de Service

Les clés de compte de service représentent un risque de sécurité majeur : leur compromission accorde un accès persistant jusqu'à révocation explicite. Workload Identity Federation élimine ce risque en permettant aux workloads externes d'échanger leurs credentials natifs contre des tokens Google Cloud éphémères.

Le mécanisme repose sur trois composants :

1. **Workload Identity Pool** : Conteneur logique représentant un environnement externe (AWS, Azure, GitHub, pipeline CI/CD)
2. **Workload Identity Provider** : Configuration de confiance avec le fournisseur d'identité externe
3. **Attribute Mapping** : Règles de transformation des attributs du token externe vers les attributs Google Cloud

```yaml
# Exemple : Configuration pour GitHub Actions
workload_identity_pool: "github-pool"
provider: "github-provider"
attribute_mapping:
  google.subject: "assertion.sub"
  attribute.actor: "assertion.actor"
  attribute.repository: "assertion.repository"
attribute_condition: |
  assertion.repository == "my-org/my-repo"
```

Cette architecture offre plusieurs avantages décisifs :

- **Élimination des secrets statiques** : Plus de clés à stocker, rotationner ou risquer de fuiter
- **Credentials éphémères** : Tokens de courte durée limitant la fenêtre d'exploitation
- **Audit amélioré** : Traçabilité complète de l'identité externe dans Cloud Audit Logs
- **Multi-cloud natif** : Support AWS, Azure, OIDC, SAML 2.0

> **Attention**  
> Configurez des conditions d'attributs strictes pour éviter les usurpations d'identité. Une condition trop permissive pourrait autoriser des workloads non autorisés à obtenir des tokens.

### Workload Identity Federation for GKE

Pour les clusters GKE, Workload Identity Federation permet d'associer des identités IAM aux pods Kubernetes sans credentials statiques. Chaque ServiceAccount Kubernetes peut être lié à un compte de service IAM, permettant aux pods d'accéder aux ressources Google Cloud de manière sécurisée.

```yaml
# Annotation du ServiceAccount Kubernetes
apiVersion: v1
kind: ServiceAccount
metadata:
  name: recommendation-agent
  namespace: agents
  annotations:
    iam.gke.io/gcp-service-account: recommendation-agent@project.iam.gserviceaccount.com
```

L'accès direct aux ressources (Direct Resource Access) constitue l'évolution récente permettant d'éviter l'impersonation de compte de service en accordant les rôles IAM directement à l'identité Kubernetes.

### Agent Identities : Identités pour l'IA Agentique

Google Cloud introduit les **Agent Identities** (en Preview), des identités gérées spécifiquement conçues pour les workloads agentiques. Ces identités attestées sont liées au cycle de vie des agents déployés sur Vertex AI Agent Engine, offrant :

- **Attestation forte** : Vérification cryptographique de l'origine de l'agent
- **Gestion automatique** : Création et rotation automatiques des credentials
- **Intégration native** : Support transparent dans l'écosystème Vertex AI
- **Traçabilité complète** : Attribution claire des actions dans les journaux d'audit

Cette fonctionnalité répond directement au défi ASI03 (Agent Identity and Authorization Abuse) identifié par l'OWASP, en établissant une chaîne de confiance vérifiable pour les agents autonomes.

L'annonce d'**Agentic IAM** lors du Security Summit 2025 signale l'évolution vers un système d'identité conçu nativement pour les agents. Cette fonctionnalité, prévue pour fin 2025, permettra aux organisations de définir des identités d'agents avec des propriétés spécifiques :

- **Scope limité** : Restrictions explicites sur les ressources accessibles
- **Durée de vie contrôlée** : Expiration automatique des credentials selon le cycle de vie de l'agent
- **Héritage de contexte** : Propagation des attributs de sécurité dans les chaînes d'orchestration multi-agents
- **Révocation instantanée** : Invalidation immédiate en cas de détection d'anomalie comportementale

### Principe de Moindre Privilège et Rôles Personnalisés

IAM Google Cloud implémente le principe de moindre privilège via une hiérarchie de rôles :

| Niveau | Exemples | Usage |
|--------|----------|-------|
| Rôles primitifs | Owner, Editor, Viewer | À proscrire sauf cas exceptionnels |
| Rôles prédéfinis | roles/aiplatform.user, roles/pubsub.publisher | Point de départ recommandé |
| Rôles personnalisés | Combinaison précise de permissions | Production avec exigences strictes |

> **Bonnes pratiques**  
> Auditez régulièrement les permissions accordées via Policy Analyzer. Identifiez les permissions non utilisées et réduisez les rôles au strict nécessaire. Utilisez les recommandations IAM pour identifier les réductions possibles.

---

## II.14.3 Sécurité Réseau

### Architecture Zéro Confiance pour les Systèmes Agentiques

Le modèle zéro confiance abandonne la distinction traditionnelle entre réseau interne « de confiance » et réseau externe « hostile ». Chaque requête est vérifiée indépendamment de son origine, établissant une posture de sécurité adaptée aux architectures distribuées modernes. Google Cloud propose un arsenal complet pour implémenter cette philosophie.

### VPC Service Controls : Périmètres de Sécurité pour les Données

VPC Service Controls crée des périmètres de sécurité autour des ressources Google Cloud, empêchant l'exfiltration de données même en cas de compromission de credentials. Cette couche de défense complète IAM en ajoutant des contrôles contextuels.

Un périmètre de service définit :
- **Projets protégés** : Les ressources incluses dans le périmètre
- **Services restreints** : Les APIs Google Cloud soumises aux contrôles (BigQuery, Vertex AI, Cloud Storage, etc.)
- **Niveaux d'accès** : Conditions contextuelles autorisant l'accès (IP, identité de l'appareil, géolocalisation)

```yaml
# Exemple de périmètre pour l'infrastructure agentique
name: "agentique-perimeter"
resources:
  - "projects/12345678901"  # Projet agents
  - "projects/12345678902"  # Projet données
restricted_services:
  - "aiplatform.googleapis.com"
  - "bigquery.googleapis.com"
  - "storage.googleapis.com"
access_levels:
  - "accessPolicies/123456789/accessLevels/corporate-network"
```

> **Perspective stratégique**  
> VPC Service Controls atténue les risques de vol de credentials et de menaces internes. Même si un attaquant obtient des tokens valides, il ne peut pas exfiltrer les données vers des ressources extérieures au périmètre.

Les règles d'entrée (ingress) et de sortie (egress) permettent des échanges contrôlés entre périmètres, essentiels pour les architectures multi-équipes ou les partenariats.

### Connectivité Privée : Private Service Connect et Private Google Access

La connectivité privée élimine l'exposition aux réseaux publics, réduisant drastiquement la surface d'attaque.

**Private Service Connect** crée des endpoints privés dans votre VPC pour accéder aux services Google Cloud. Le trafic reste entièrement sur le réseau Google, sans traverser Internet.

**Private Google Access** permet aux instances sans IP publique d'accéder aux APIs Google Cloud via des plages d'adresses privées :

| Domaine | IP Range | Usage |
|---------|----------|-------|
| private.googleapis.com | 199.36.153.8/30 | Accès privé standard |
| restricted.googleapis.com | 199.36.153.4/30 | Accès compatible VPC Service Controls |

Pour les charges de travail on-premises, Cloud VPN ou Cloud Interconnect étendent cette connectivité privée au datacenter de l'entreprise.

### Segmentation Réseau et Règles de Pare-feu

La segmentation réseau isole les composants de l'architecture agentique selon leur criticité et leurs patterns de communication.

**Shared VPC** permet une gestion centralisée du réseau tout en isolant les workloads dans des projets de service distincts. L'équipe réseau contrôle les sous-réseaux, les règles de pare-feu et les routes, tandis que les équipes applicatives déploient leurs ressources de manière autonome.

**Politiques de pare-feu hiérarchiques** définissent des règles à l'échelle de l'organisation, des dossiers et des projets :

```yaml
# Politique organisation : bloquer par défaut
- priority: 65534
  action: DENY
  direction: INGRESS
  
# Politique dossier production : autoriser le trafic interne
- priority: 1000
  action: ALLOW
  direction: INGRESS
  match:
    srcIpRanges: ["10.0.0.0/8"]
```

> **Bonnes pratiques**  
> Utilisez des tags réseau pour identifier les workloads (agent-tier, data-tier, api-tier) et définissez les règles de pare-feu en fonction de ces tags plutôt que d'adresses IP statiques. Cette approche facilite l'évolution de l'infrastructure.

### Sécurité des Communications Inter-Agents

Les communications entre agents cognitifs nécessitent une attention particulière car elles transportent des contextes d'intention et des décisions potentiellement sensibles.

**Cloud Service Mesh** (basé sur Istio) implémente mTLS automatique entre tous les services du mesh, chiffrant et authentifiant chaque communication sans modification du code applicatif. Les politiques d'autorisation définissent précisément quels services peuvent communiquer :

```yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: agent-communication-policy
spec:
  selector:
    matchLabels:
      app: orchestrator-agent
  rules:
  - from:
    - source:
        principals: ["cluster.local/ns/agents/sa/recommendation-agent"]
    to:
    - operation:
        methods: ["POST"]
        paths: ["/api/v1/decisions"]
```

Cette approche répond directement aux risques ASI07 (Insecure Inter-Agent Communication) en établissant une authentification mutuelle et une autorisation explicite pour chaque canal de communication.

### Cloud NGFW et Protection DDoS

**Cloud Next Generation Firewall (NGFW)** ajoute une couche de protection applicative au-delà des règles de pare-feu traditionnelles. Les fonctionnalités avancées incluent :

- **Intrusion Prevention Service (IPS)** : Détection et blocage des tentatives d'exploitation de vulnérabilités connues
- **Inspection TLS** : Analyse du trafic chiffré pour détecter les menaces dissimulées
- **Tags organisationnels** : Application de politiques cohérentes à l'échelle de l'organisation

Pour les charges de travail haute performance, y compris les workloads IA, Cloud NGFW supporte désormais les réseaux RDMA (Remote Direct Memory Access), permettant l'application de politiques zéro confiance même sur les communications à très faible latence.

**Cloud Armor** protège les endpoints exposés contre les attaques DDoS et les abus applicatifs. Les politiques de sécurité WAF (Web Application Firewall) peuvent bloquer les patterns d'injection de prompts connus lorsqu'ils transitent via des APIs HTTP.

---

## II.14.4 Google Cloud Security Command Center

### Une Plateforme Unifiée de Gestion des Risques

Security Command Center (SCC) constitue la plateforme native de Google Cloud pour la gestion de la posture de sécurité. Loin d'être un simple scanner de vulnérabilités, SCC offre une vision holistique des risques incluant la découverte d'actifs, la détection des menaces, la gestion de la conformité et, désormais, la protection spécifique des charges de travail IA.

### AI Protection : Sécuriser le Cycle de Vie de l'IA

Annoncée en mars 2025 et disponible en général dans le tier Enterprise, **AI Protection** étend les capacités de SCC aux workloads d'intelligence artificielle. Cette fonctionnalité répond à un constat : les applications IA nécessitent des contrôles de sécurité spécialisés que les outils traditionnels ne couvrent pas.

AI Protection s'articule autour de quatre capacités :

1. **Découverte de l'inventaire IA** : Identification automatique des agents, modèles, applications, endpoints et données IA dans l'environnement
2. **Évaluation des risques** : Analyse des vulnérabilités spécifiques aux workloads IA
3. **Contrôles et garde-fous** : Postures de sécurité recommandées pour Vertex AI
4. **Détection et réponse aux menaces** : Identification des attaques ciblant les systèmes IA

> **Perspective stratégique**  
> AI Protection intègre les renseignements de sécurité de Google et Mandiant pour identifier les techniques d'attaque émergentes contre les systèmes IA, incluant le détournement de modèles, l'empoisonnement de données et l'injection de prompts.

Le tableau de bord AI Security offre une vue consolidée de la posture de sécurité IA, incluant :
- Inventaire des actifs IA par type (modèles, datasets, endpoints)
- Résumé des données sensibles dans les datasets Vertex AI
- Statistiques Model Armor (injections détectées, jailbreaks bloqués)
- Recommandations de remédiation priorisées

### Model Armor : Filtrage des Prompts et Réponses

**Model Armor** constitue le composant défensif actif d'AI Protection, filtrant les interactions avec les modèles pour détecter et bloquer les contenus malveillants.

Les capacités de détection incluent :
- **Injection de prompts** : Tentatives de manipulation du comportement du modèle
- **Jailbreak** : Contournement des garde-fous de sécurité
- **Fuite de données sensibles** : Détection via intégration avec Sensitive Data Protection
- **URLs malveillantes** : Blocage des références à des ressources dangereuses
- **Contenu offensant** : Filtrage selon les politiques d'utilisation

Model Armor s'intègre désormais directement avec Vertex AI, appliquant une configuration de sécurité par défaut sur tous les nouveaux endpoints de prédiction. L'intégration avec les serveurs MCP (Model Context Protocol) permet également de filtrer les interactions agent-outil.

```python
# Exemple d'intégration Model Armor via API
from google.cloud import modelarmor_v1

client = modelarmor_v1.ModelArmorServiceClient()
request = modelarmor_v1.SanitizeRequest(
    name="projects/my-project/locations/us-central1",
    content=user_prompt,
    model_armor_settings=modelarmor_v1.ModelArmorSettings(
        prompt_injection_detection=True,
        jailbreak_detection=True,
        sensitive_data_protection=True
    )
)
response = client.sanitize(request)
if response.blocked:
    # Gérer la tentative d'attaque
    log_security_event(response.blocking_reasons)
```

### Event Threat Detection pour Vertex AI

Security Command Center intègre des règles de détection spécifiques aux actifs Vertex AI, identifiant les comportements suspects en temps quasi réel :

| Détecteur | Menace Ciblée | Action Recommandée |
|-----------|---------------|-------------------|
| Vertex AI Notebook Public Access | Exposition d'un notebook via IP publique | Restreindre l'accès immédiatement |
| Vertex AI Workbench File Download | Exfiltration potentielle de données | Investiguer l'utilisateur et le contenu |
| Vertex AI Privilege Escalation | Modification suspecte des droits d'accès | Révoquer et auditer les changements |
| Vertex AI Model Hijacking | Tentative de détournement de modèle | Isoler le modèle et analyser les accès |
| Vertex AI Dataset Anomaly | Accès inhabituel aux données d'entraînement | Vérifier la légitimité de l'opération |

L'Agent Engine Threat Detection (Preview) étend ces capacités aux agents déployés sur Vertex AI Agent Engine Runtime, détectant les attaques spécifiques aux systèmes agentiques. Les détections incluent :

- **Comportement d'agent anormal** : Déviation significative des patterns d'actions habituels
- **Escalade de privilèges via outils** : Tentatives d'accès à des ressources non autorisées via appels de fonctions
- **Communication suspecte inter-agents** : Échanges de données avec des agents non autorisés
- **Injection de contexte malveillant** : Manipulation du contexte de mémoire de l'agent

### Simulation d'Attaques et Scores d'Exposition

SCC propose des capacités avancées d'analyse des risques :

**Attack Path Simulation** modélise les chemins qu'un attaquant pourrait emprunter pour compromettre les actifs IA. Cette simulation identifie les combinaisons toxiques de vulnérabilités et de mauvaises configurations qui, prises isolément, semblent mineures mais constituent collectivement un risque majeur.

**Attack Exposure Score** quantifie le risque associé à chaque actif en fonction de :
- La criticité de l'actif (dataset d'entraînement, modèle de production)
- Le nombre de chemins d'attaque viables
- La facilité d'exploitation des vulnérabilités identifiées

Cette priorisation guide les équipes vers les remédiations à plus fort impact.

### Intégration avec Sensitive Data Protection

**Sensitive Data Protection (SDP)** étend sa découverte automatisée aux datasets Vertex AI, identifiant les types de données sensibles présentes dans les données d'entraînement et de fine-tuning. Cette visibilité est critique pour :

- Identifier les risques de fuite de PII via les réponses du modèle
- Valider la conformité des datasets avec les politiques de l'organisation
- Détecter l'empoisonnement de données par injection de contenu malveillant

Les profils de données générés fournissent une cartographie précise de la sensibilité, permettant d'appliquer des contrôles proportionnés au risque.

---

## II.14.5 Audit et Traçabilité

### L'Impératif de l'Audit Exhaustif

La traçabilité exhaustive constitue le socle de la posture de sécurité. Sans journaux complets et fiables, la détection d'intrusions, l'investigation d'incidents et la démonstration de conformité deviennent impossibles. Pour les systèmes agentiques, cet impératif s'intensifie : les actions autonomes des agents doivent être traçables et attribuables avec la même rigueur que les actions humaines.

### Cloud Audit Logs : Fondation de la Traçabilité

Google Cloud génère automatiquement des journaux d'audit pour toutes les opérations sur ses ressources. Ces journaux répondent à la question fondamentale : « Qui a fait quoi, où et quand ? ».

Quatre catégories de journaux coexistent :

| Type | Contenu | Rétention par défaut | Coût |
|------|---------|---------------------|------|
| Admin Activity | Opérations administratives (création, modification, suppression) | 400 jours | Inclus |
| Data Access | Lectures et écritures de données | 30 jours | Facturable |
| System Event | Actions automatiques des services Google | 400 jours | Inclus |
| Policy Denied | Requêtes refusées par IAM ou VPC Service Controls | 30 jours | Inclus |

> **Bonnes pratiques**  
> Activez les journaux Data Access pour tous les services manipulant des données sensibles. Ces journaux permettent de détecter les accès anormaux et sont essentiels pour les investigations d'incidents.

### Configuration et Centralisation

La configuration des journaux d'audit s'effectue au niveau du projet, du dossier ou de l'organisation. Une configuration organisationnelle assure une couverture uniforme et facilite la gouvernance.

```bash
# Activer les Data Access logs pour Vertex AI au niveau organisation
gcloud organizations add-iam-policy-binding $ORG_ID \
  --member="serviceAccount:cloud-logs@system.gserviceaccount.com" \
  --role="roles/logging.logWriter"
  
gcloud logging update --organization=$ORG_ID \
  --audit-log-config="service=aiplatform.googleapis.com,log_type=DATA_READ" \
  --audit-log-config="service=aiplatform.googleapis.com,log_type=DATA_WRITE"
```

**Log Sinks** permettent de router les journaux vers des destinations multiples :

- **Cloud Storage** : Archivage long terme pour conformité
- **BigQuery** : Analyse ad-hoc et investigation
- **Pub/Sub** : Intégration temps réel avec SIEM externes
- **Log Buckets personnalisés** : Contrôle fin de la rétention et du chiffrement

Les sinks agrégés au niveau organisation capturent les journaux de tous les projets enfants, garantissant qu'aucun événement n'échappe à la centralisation.

### Intégration SIEM et Détection des Menaces

Les journaux d'audit alimentent les systèmes de détection pour transformer les données brutes en intelligence actionnable.

**Google SecOps (Chronicle)** ingère nativement les Cloud Audit Logs et applique des règles de détection basées sur les techniques MITRE ATT&CK. Les capacités incluent :

- Corrélation d'événements multi-sources
- Enrichissement contextuel (identité, géolocalisation, réputation)
- Détection comportementale via machine learning
- Timeline d'investigation pour les incidents

> **Exemple concret**  
> Un analyste détecte qu'un compte de service a modifié des politiques IAM (SetIamPolicy) puis accédé à des datasets sensibles en dehors des heures de bureau. Chronicle corrèle ces événements, enrichit avec les données de géolocalisation, et génère une alerte haute priorité pour investigation.

L'intégration avec des SIEM tiers (Splunk, Datadog, Sumo Logic) s'effectue via Pub/Sub ou exportation BigQuery, permettant aux organisations de conserver leurs investissements existants.

### Audit Spécifique aux Systèmes Agentiques

Les agents cognitifs génèrent des patterns d'activité distincts qui nécessitent des approches d'audit adaptées.

**Traçabilité des décisions** : Chaque action d'un agent doit être reliée à la chaîne de raisonnement qui l'a produite. Les journaux doivent capturer :
- Le contexte d'entrée (événements déclencheurs, données de contexte)
- Le raisonnement intermédiaire (étapes de ReAct, appels de fonctions)
- La décision finale et son exécution
- Les effets de bord sur les systèmes externes

**Workload Identity Federation Logging** : Lorsque des workloads externes impersonnent des comptes de service via Workload Identity Federation, les journaux incluent une section `serviceAccountDelegationInfo` identifiant le principal externe. Cette traçabilité est essentielle pour l'audit des pipelines CI/CD et des agents multi-cloud.

### Rétention et Conformité

Les exigences réglementaires dictent souvent des durées de rétention spécifiques :

| Réglementation | Durée Typique | Données Concernées |
|----------------|---------------|-------------------|
| RGPD | Minimisation | Données personnelles |
| SOC 2 | 1 an minimum | Tous les journaux de sécurité |
| PCI DSS | 1 an minimum, 3 mois en ligne | Accès aux données de paiement |
| HIPAA | 6 ans | Données de santé |

Cloud Logging permet de configurer des durées de rétention personnalisées (1 à 3650 jours) par bucket de logs. Le chiffrement CMEK (Customer-Managed Encryption Keys) assure le contrôle des clés pour les exigences les plus strictes.

> **Attention**  
> La suppression de journaux avant l'expiration de la période réglementaire peut constituer une obstruction. Configurez des politiques de rétention conservatrices et documentez la justification de toute purge.

### Audit des Systèmes Kafka

La traçabilité du backbone Kafka complète les journaux Google Cloud. Confluent Platform génère des journaux d'audit détaillés capturant :

- **Opérations administratives** : Création de topics, modification des ACLs, changements de configuration
- **Authentification** : Succès et échecs de connexion, par mécanisme et principal
- **Autorisation** : Décisions d'accès (autorisé/refusé) avec contexte complet

L'intégration avec Cloud Logging permet de centraliser les journaux Kafka avec ceux de Google Cloud, offrant une vue unifiée pour l'investigation et la corrélation.

### Immutabilité et Intégrité

La valeur probante des journaux repose sur leur intégrité. Plusieurs mécanismes garantissent l'immutabilité :

- **Bucket locks** : Empêchent la modification ou la suppression des journaux
- **Signed URLs** : Prouvent l'authenticité des exports archivés
- **Contrôles IAM stricts** : Limitent l'accès administratif aux journaux

La séparation des responsabilités impose que les équipes opérationnelles ne puissent pas modifier les journaux qu'elles génèrent. Un modèle courant attribue les droits d'écriture aux workloads (via logging.logWriter) et les droits de lecture/gestion à une équipe sécurité distincte.

---

## II.14.6 Résumé

Ce chapitre a établi les fondations de la sécurisation de l'infrastructure agentique, démontrant que la protection des systèmes autonomes exige une approche multicouche cohérente.

### Principes Clés

| Domaine | Principe Directeur | Implémentation |
|---------|-------------------|----------------|
| Backbone Kafka | Défense en profondeur | mTLS + RBAC + chiffrement au repos |
| Identités Google Cloud | Élimination des secrets statiques | Workload Identity Federation |
| Sécurité réseau | Zéro confiance | VPC Service Controls + Private Service Connect |
| Protection IA | Visibilité et contrôle | AI Protection + Model Armor |
| Audit | Traçabilité exhaustive | Cloud Audit Logs + SIEM |

### Recommandations Opérationnelles

1. **Standardisez l'authentification** : Adoptez mTLS pour les communications inter-services et OAUTHBEARER pour les workloads cloud-native. Éliminez les credentials statiques via Workload Identity Federation.

2. **Implémentez RBAC systématiquement** : Configurez le contrôle d'accès basé sur les rôles pour Kafka (via Confluent MDS) et Google Cloud (via IAM). Alignez les rôles sur l'organisation et automatisez la synchronisation avec l'annuaire d'entreprise.

3. **Établissez des périmètres de sécurité** : Déployez VPC Service Controls autour de toutes les ressources sensibles. Définissez des règles d'entrée/sortie explicites pour les échanges inter-périmètres.

4. **Activez AI Protection** : Configurez Security Command Center avec AI Protection pour obtenir une visibilité complète sur les actifs IA. Déployez Model Armor sur tous les endpoints Vertex AI en production.

5. **Centralisez et protégez les journaux** : Configurez des sinks organisationnels vers BigQuery et Cloud Storage. Activez les Data Access logs pour les services critiques. Intégrez avec votre SIEM pour la détection en temps réel.

### Avertissement Final

> **Attention**  
> La sécurité de l'infrastructure constitue une condition nécessaire mais non suffisante. Les contrôles décrits dans ce chapitre protègent le substrat technologique, mais ne peuvent prévenir les dérives comportementales des agents cognitifs eux-mêmes. La gouvernance constitutionnelle et l'observabilité comportementale, traitées dans les chapitres précédents, complètent cette fondation technique pour établir une posture de sécurité véritablement holistique.

La sécurisation de l'infrastructure agentique représente un investissement significatif, mais l'alternative — opérer des agents autonomes sur une infrastructure vulnérable — expose l'organisation à des risques existentiels. Le chapitre suivant aborde la dimension complémentaire de la conformité réglementaire et de la gestion de la confidentialité, bouclant ainsi la boucle de la sécurité des systèmes agentiques.

---

*Chapitre suivant : Chapitre II.15 — Conformité Réglementaire et Gestion de la Confidentialité*
