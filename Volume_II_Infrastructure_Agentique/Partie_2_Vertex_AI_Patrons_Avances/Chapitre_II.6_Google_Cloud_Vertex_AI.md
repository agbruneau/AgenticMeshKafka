# Chapitre II.6 — Google Cloud Vertex AI comme Environnement d'Exploitation Agentique

---

## Introduction

La transformation vers l'entreprise agentique exige une infrastructure capable de supporter le cycle de vie complet des agents cognitifs : de leur conception initiale jusqu'à leur déploiement en production, en passant par leur gouvernance et leur observabilité. Google Cloud Vertex AI s'impose aujourd'hui comme l'une des plateformes les plus complètes pour répondre à ces exigences, offrant un environnement intégré où les équipes d'ingénierie peuvent construire, mettre à l'échelle et gouverner des systèmes multi-agents de niveau entreprise.

L'évolution de Vertex AI illustre parfaitement le passage d'une plateforme d'apprentissage automatique traditionnelle vers un véritable environnement d'exploitation agentique. Lancée initialement en mai 2021 comme plateforme unifiée pour le développement et le déploiement de modèles d'IA, Vertex AI a progressivement intégré les capacités nécessaires à l'ère agentique : l'Agent Builder pour la construction d'agents, l'Agent Engine pour leur exécution managée, le RAG Engine pour l'ancrage contextuel, et une suite complète d'outils d'observabilité et de gouvernance.

Ce chapitre explore en profondeur l'architecture et les capacités de Vertex AI en tant qu'environnement d'exploitation agentique. Nous examinerons d'abord la vision d'ensemble de la plateforme et son positionnement dans l'écosystème Google Cloud. Nous détaillerons ensuite le Model Garden, véritable catalogue de plus de 200 modèles fondamentaux, avant d'approfondir l'Agent Builder et ses composantes. Nous aborderons également les patrons de développement d'agents personnalisés avec l'Agent Development Kit (ADK), pour conclure sur les environnements d'exécution et les capacités de mise en production.

L'objectif est de fournir aux architectes et aux ingénieurs les connaissances nécessaires pour concevoir et opérer des systèmes agentiques robustes sur Vertex AI, en tirant parti de l'intégration native avec l'écosystème Google Cloud et du backbone événementiel Confluent présenté dans les chapitres précédents.

---

## II.6.1 Vue d'Ensemble de la Plateforme Vertex AI

### Architecture Fondamentale

Vertex AI constitue la plateforme unifiée de Google Cloud pour l'intelligence artificielle et l'apprentissage automatique. Son architecture repose sur trois piliers fondamentaux qui structurent l'ensemble des capacités offertes : **Construire** (Build), **Mettre à l'échelle** (Scale) et **Gouverner** (Govern).

Le pilier **Construire** englobe les outils de développement, depuis le Vertex AI Studio pour le prototypage rapide jusqu'à l'Agent Development Kit (ADK) pour le développement code-first d'agents sophistiqués. Ce pilier inclut également l'Agent Garden, une bibliothèque d'agents et d'outils préconçus permettant d'accélérer le développement.

Le pilier **Mettre à l'échelle** s'articule autour de l'Agent Engine, l'environnement d'exécution managé qui prend en charge le déploiement, la gestion des sessions, la mémoire à long terme et l'exécution de code dans des environnements isolés. Ce runtime permet de passer du prototype à la production sans restructuration majeure du code.

Le pilier **Gouverner** adresse les impératifs de sécurité et de conformité entreprise : identité des agents via IAM, détection des menaces avec Security Command Center, contrôles d'accès aux outils via le Cloud API Registry, et traçabilité complète des opérations.

### Intégration dans l'Écosystème Google Cloud

L'une des forces majeures de Vertex AI réside dans son intégration native avec l'ensemble des services Google Cloud. Cette intégration se manifeste à plusieurs niveaux :

**Données et analytique** : Les agents peuvent interroger directement BigQuery pour l'analyse de données structurées, accéder à Cloud Storage et Google Drive pour les documents, et exploiter Vertex AI Search pour la recherche sémantique sur les corpus d'entreprise.

**Sécurité et identité** : L'intégration avec Identity and Access Management (IAM) permet de gérer les permissions des agents comme celles de tout autre service Google Cloud. Les VPC Service Controls assurent l'isolation du trafic réseau, tandis que les journaux d'audit Cloud Logging garantissent la traçabilité.

**Opérations et observabilité** : Cloud Trace (avec support OpenTelemetry), Cloud Monitoring et Cloud Logging forment la colonne vertébrale de l'observabilité, permettant de suivre les performances, détecter les anomalies et déboguer les comportements des agents.

> **Perspective stratégique**  
> L'intégration native de Vertex AI avec l'écosystème Google Cloud représente un avantage compétitif significatif pour les organisations déjà présentes sur cette plateforme. Elle permet de construire des systèmes agentiques qui héritent automatiquement des politiques de sécurité, des contrôles d'accès et des capacités d'audit déjà en place, réduisant considérablement le temps de mise en production.

### Modèle de Tarification et Considérations Opérationnelles

Depuis mars 2025, la tarification de l'Agent Engine repose sur la consommation de ressources calculées en heures vCPU et GiB-heures. Les services Sessions, Memory Bank et Code Execution ont atteint la disponibilité générale (GA) en décembre 2025, avec une facturation effective débutant le 28 janvier 2026.

| Service | Statut | Facturation |
|---------|--------|-------------|
| Agent Engine Runtime | GA | Active depuis novembre 2025 |
| Sessions | GA | À partir de janvier 2026 |
| Memory Bank | GA | À partir de janvier 2026 |
| Code Execution | Preview | À partir de janvier 2026 |

Un mode Express permet aux développeurs de démarrer sans compte Google Cloud complet, avec une période d'essai gratuite de 90 jours et des quotas limités. Cette option facilite l'expérimentation avant l'engagement sur des charges de travail de production.

---

## II.6.2 Vertex AI Model Garden

### Catalogue de Modèles Fondamentaux

Le Model Garden constitue le point d'entrée vers plus de 200 modèles fondamentaux, organisés en trois catégories principales : les modèles propriétaires Google (première partie), les modèles partenaires (troisième partie) et les modèles open source.

**Modèles Google (Première partie)** : La famille Gemini représente le cœur de l'offre Google. Gemini 3 Pro, le modèle le plus récent, est optimisé pour les flux de travail agentiques complexes et le raisonnement avancé, avec une fenêtre de contexte d'un million de tokens. Gemini 2.5 Flash offre un équilibre entre intelligence et latence, tandis que Gemini 2.5 Flash-Lite est conçu pour les tâches à haut débit nécessitant une optimisation des coûts.

**Modèles Partenaires** : Le Model Garden intègre des modèles de partenaires stratégiques, notamment Claude 3.7 Sonnet et Claude Haiku 4.5 d'Anthropic, Llama 4 de Meta, et les modèles Mistral. Cette diversité permet aux architectes de sélectionner le modèle le mieux adapté à chaque cas d'usage spécifique.

**Modèles Open Source** : Une large sélection de modèles open source est disponible, incluant Gemma (la version open source de Gemini), les modèles Llama, Qwen, et de nombreux autres. Ces modèles peuvent être déployés sur l'infrastructure managée de Vertex AI ou personnalisés avec des données propriétaires.

### Sélection et Déploiement des Modèles

Le Model Garden offre plusieurs modes de consommation adaptés aux différents besoins :

**Model-as-a-Service (MaaS)** : Les modèles sont accessibles via API sans gestion d'infrastructure. Ce mode convient aux charges de travail variables et aux expérimentations.

**Déploiement auto-hébergé** : Pour les exigences de latence strictes ou de souveraineté des données, les modèles peuvent être déployés sur une infrastructure dédiée avec contrôle total des ressources.

**Personnalisation et affinage** : Les modèles peuvent être affinés (fine-tuning) avec des données propriétaires pour améliorer les performances sur des tâches spécifiques. Gemini 2.5 Flash-Lite et Gemini 2.5 Pro supportent notamment l'affinage supervisé.

> **Bonnes pratiques**  
> La sélection d'un modèle doit s'appuyer sur une évaluation systématique des critères suivants : capacités de raisonnement requises, latence acceptable, coût par token, fenêtre de contexte nécessaire, et exigences multimodales. Le Vertex AI Studio permet de tester rapidement différents modèles avant de s'engager en production.

### Gouvernance et Cycle de Vie

Google assure une gouvernance rigoureuse des modèles disponibles dans le Model Garden :

**Sécurité** : Les conteneurs de service et d'affinage fournis par Google font l'objet de tests approfondis et d'analyses de vulnérabilités. Les modèles partenaires subissent des scans de points de contrôle pour garantir leur authenticité.

**Cycle de vie** : Chaque modèle suit un cycle de vie documenté avec des dates de disponibilité, de dépréciation et de retrait. Les aliases auto-mis à jour (par exemple `gemini-2.5-flash`) pointent automatiquement vers la dernière version stable, facilitant les migrations.

---

## II.6.3 Vertex AI Agent Builder

### Vue d'Ensemble de la Suite Agent Builder

Vertex AI Agent Builder représente la suite complète de produits pour construire, mettre à l'échelle et gouverner des agents IA en production. Elle se compose de plusieurs éléments interdépendants :

**Agent Development Kit (ADK)** : Cadriciel open source pour le développement code-first d'agents sophistiqués. Disponible en Python, Java, Go et TypeScript, l'ADK a été téléchargé plus de 7 millions de fois et alimente les agents des produits Google comme Agentspace et le Customer Engagement Suite.

**Agent Engine** : Ensemble de services managés pour le déploiement, la gestion et la mise à l'échelle des agents. Comprend le runtime, les sessions, la Memory Bank, l'exécution de code et les capacités d'évaluation.

**Agent Garden** : Bibliothèque d'agents et d'outils préconçus accessibles dans la console Google Cloud. Les développeurs y trouvent des solutions prêtes à l'emploi pour des cas d'usage courants ainsi que des composants réutilisables.

**Cloud API Registry** : Registre centralisé permettant aux administrateurs de gouverner les outils disponibles pour les développeurs d'agents. Supporte les serveurs MCP (Model Context Protocol) personnalisés et les outils préconçus pour les services Google.

### Agent Development Kit (ADK) en Détail

L'ADK applique les principes du développement logiciel à la création d'agents IA. Ses caractéristiques principales incluent :

**Développement Code-First** : La logique des agents, les outils et l'orchestration sont définis directement en code Python (ou Java/Go/TypeScript), offrant flexibilité, testabilité et versionnement.

```python
from google.adk.agents import Agent
from google.adk.tools import google_search

root_agent = Agent(
    name="assistant_recherche",
    model="gemini-2.5-flash",
    instruction="Tu es un assistant de recherche. Réponds aux questions en utilisant Google Search si nécessaire.",
    description="Un assistant capable de rechercher sur le web.",
    tools=[google_search]
)
```

**Systèmes Multi-Agents Modulaires** : L'ADK permet de concevoir des hiérarchies d'agents spécialisés qui collaborent pour accomplir des tâches complexes. Un agent coordinateur peut déléguer des tâches à des agents spécialistes selon les besoins.

**Écosystème d'Outils Riche** : Les agents peuvent exploiter des outils préconçus (Google Search, Vertex AI Search, exécution de code), des fonctions personnalisées, des spécifications OpenAPI, des outils MCP, ou même d'autres agents utilisés comme outils.

**Déploiement Flexible** : Les agents développés avec l'ADK peuvent être conteneurisés et déployés sur Cloud Run, ou mis à l'échelle via l'Agent Engine sans modification du code.

### Services de l'Agent Engine

L'Agent Engine fournit l'infrastructure managée pour l'exécution des agents en production :

**Sessions** : Service de gestion de l'historique conversationnel dans le cadre d'une session. Chaque session contient la séquence chronologique des messages et actions (SessionEvents) entre l'utilisateur et l'agent.

**Memory Bank** : Service de mémoire à long terme permettant aux agents de stocker, récupérer et gérer des informations pertinentes à travers plusieurs sessions. Basé sur une méthode de recherche thématique développée par Google Cloud AI Research (acceptée à ACL 2025), Memory Bank extrait automatiquement les faits et préférences des conversations.

**Code Execution** : Environnement sandbox isolé permettant aux agents d'exécuter du code généré de manière sécurisée. Essentiel pour les agents devant effectuer des calculs, manipuler des données ou exécuter des scripts.

**Evaluation** : Service d'évaluation permettant de tester la fiabilité des agents en simulant des interactions utilisateur et en mesurant la qualité des réponses.

> **Note technique**  
> L'intégration de Memory Bank avec l'ADK s'effectue via le `VertexAiMemoryBankService`. Les mémoires sont extraites de manière asynchrone en arrière-plan à partir de l'historique conversationnel stocké dans Sessions, sans nécessiter de pipelines d'extraction complexes.

---

## II.6.4 Développement d'Agents Personnalisés

### Patrons Architecturaux Multi-Agents

L'ADK supporte plusieurs patrons de conception pour les systèmes multi-agents, chacun adapté à des besoins spécifiques :

**Pipeline Séquentiel** : Chaîne d'agents où chaque agent complète une tâche avant de passer le relais au suivant. Ce patron, déterministe et facile à déboguer, convient aux pipelines de traitement de données.

```python
from google.adk.agents import SequentialAgent, LlmAgent

parser_agent = LlmAgent(name="parser", model="gemini-2.5-flash", ...)
extractor_agent = LlmAgent(name="extractor", model="gemini-2.5-flash", ...)
summarizer_agent = LlmAgent(name="summarizer", model="gemini-2.5-flash", ...)

pipeline = SequentialAgent(
    name="document_pipeline",
    sub_agents=[parser_agent, extractor_agent, summarizer_agent]
)
```

**Orchestration Hiérarchique** : Un agent coordinateur délègue des tâches à des agents spécialistes selon leurs descriptions. Le mécanisme AutoFlow de l'ADK gère automatiquement le routage basé sur les descriptions fournies.

**Exécution Parallèle** : Plusieurs agents travaillent simultanément sur des sous-tâches indépendantes, les résultats étant agrégés à la fin. Adapté aux tâches décomposables sans dépendances.

**Boucle Itérative** : Un agent exécute une tâche de manière répétée jusqu'à ce qu'une condition de sortie soit satisfaite. Utile pour le raffinement progressif ou la validation.

**Human-in-the-Loop (HITL)** : L'ADK supporte un flux de confirmation d'outils permettant de garder l'humain dans la boucle pour les actions critiques nécessitant une validation explicite.

### Intégration du RAG Engine

Le RAG Engine de Vertex AI permet d'ancrer les agents dans les données d'entreprise via la génération augmentée par récupération (Retrieval-Augmented Generation). Le processus comprend plusieurs étapes :

**Ingestion des données** : Import depuis diverses sources (fichiers locaux, Cloud Storage, Google Drive, Slack, Jira, et plus de 100 connecteurs).

**Transformation** : Découpage du contenu en fragments (chunks) avec configuration de la taille et du chevauchement.

**Indexation** : Création d'un corpus avec génération d'embeddings vectoriels. Le RAG Engine supporte plusieurs bases de données vectorielles, dont Vertex AI Vector Search, Pinecone et Weaviate.

**Récupération et génération** : Lors d'une requête, les fragments les plus pertinents sont récupérés et injectés dans le prompt envoyé au modèle.

```python
from vertexai import rag
from vertexai.generative_models import GenerativeModel, Tool

# Création d'un outil de récupération RAG
rag_retrieval_tool = Tool.from_retrieval(
    retrieval=rag.Retrieval(
        source=rag.VertexRagStore(
            rag_corpora=[rag_corpus.name],
            similarity_top_k=10,
            vector_distance_threshold=0.5,
        ),
    )
)

# Chargement de l'outil dans un modèle Gemini
rag_model = GenerativeModel(
    "gemini-2.5-flash",
    tools=[rag_retrieval_tool],
)
```

### Gouvernance des Outils avec le Cloud API Registry

Le Cloud API Registry, intégré dans la console Agent Builder, permet aux administrateurs de gouverner les outils disponibles pour les développeurs :

**Outils préconçus pour les services Google** : Support MCP pour BigQuery, Google Maps et autres services Google, disponibles directement dans Agent Builder.

**Serveurs MCP personnalisés** : Apigee permet de transformer les APIs existantes en serveurs MCP, rendant l'ensemble du patrimoine API accessible aux agents.

**Gestion simplifiée pour les développeurs** : L'ADK introduit l'objet `ApiRegistry` permettant aux développeurs d'exploiter facilement les outils managés par l'organisation.

> **Attention**  
> La gouvernance des outils est critique pour la sécurité des systèmes agentiques. Un agent disposant d'accès à des outils non contrôlés peut potentiellement exfiltrer des données ou effectuer des actions non autorisées. Le Cloud API Registry permet d'établir une liste blanche d'outils approuvés au niveau organisationnel.

---

## II.6.5 Environnements d'Exécution

### Agent Engine Runtime

L'Agent Engine Runtime constitue l'environnement d'exécution managé pour les agents déployés sur Vertex AI. Ses caractéristiques principales incluent :

**Déploiement simplifié** : Les agents développés avec l'ADK peuvent être déployés vers le runtime avec une seule commande CLI, sans nécessiter de compte Google Cloud complet grâce au mode Express.

**Mise à l'échelle automatique** : L'infrastructure s'adapte automatiquement aux variations de charge, garantissant des performances constantes pendant les pics d'utilisation.

**Isolation et sécurité** : Chaque agent s'exécute dans un environnement isolé avec ses propres ressources, réduisant les risques de contamination croisée.

**Support multi-frameworks** : Outre l'ADK natif, l'Agent Engine supporte les agents construits avec LangGraph, CrewAI et d'autres frameworks populaires.

### Gestion de l'État avec Sessions et Memory Bank

La gestion de l'état constitue un défi majeur pour les systèmes agentiques. Vertex AI propose deux niveaux complémentaires :

**Sessions (mémoire à court terme)** : Gère l'historique conversationnel dans le cadre d'une session unique. Chaque événement (message utilisateur, réponse agent, appel d'outil) est persisté et peut être récupéré pour maintenir le contexte.

```python
from google.adk.sessions import VertexAiSessionService

session_service = VertexAiSessionService(
    project="my-project",
    location="us-central1"
)

# Création d'une session
session = session_service.create_session(
    user_id="user123",
    agent_engine_id="my-agent-engine"
)

# Ajout d'événements
session_service.append_event(session.id, user_message_event)
```

**Memory Bank (mémoire à long terme)** : Extrait et stocke les faits, préférences et informations clés à travers plusieurs sessions. Les agents peuvent ainsi se souvenir des interactions passées et personnaliser leurs réponses.

Le flux typique combine les deux services : Sessions stocke l'historique de la conversation en cours, Memory Bank génère des mémoires à partir de cet historique (de manière asynchrone), et ces mémoires sont récupérées lors des sessions futures pour enrichir le contexte.

### Observabilité et Monitoring

L'observabilité des systèmes agentiques présente des défis spécifiques liés au non-déterminisme des modèles de langage. Vertex AI fournit une suite complète d'outils :

**Traçage distribué** : Support natif d'OpenTelemetry et intégration avec Cloud Trace pour suivre les flux d'exécution à travers les agents, outils et services.

**Tableau de bord de performance** : Visualisation des métriques clés (consommation de tokens, latence, taux d'erreurs, appels d'outils) au fil du temps.

**Journaux structurés** : Intégration avec Cloud Logging pour la capture et l'analyse des événements d'exécution.

**Détection de dérive comportementale** : Capacité à identifier les changements de comportement des agents par rapport à leurs performances de référence.

| Métrique | Description | Seuil Recommandé |
|----------|-------------|------------------|
| Latence P95 | 95e percentile du temps de réponse | < 5 secondes |
| Taux d'erreur | Pourcentage de requêtes en échec | < 1% |
| Tokens/requête | Consommation moyenne de tokens | Variable selon le cas d'usage |
| Taux de succès des outils | Pourcentage d'appels d'outils réussis | > 95% |

### Sécurité et Conformité

La sécurité des systèmes agentiques nécessite une approche multicouche :

**Identité des agents** : L'Agent Engine supporte les identités IAM pour les agents, permettant de gérer les permissions et l'authentification selon les politiques de sécurité organisationnelles.

**Model Armor** : Service de protection contre les attaques par injection de prompt. Model Armor inspecte les prompts envoyés aux modèles et peut bloquer les requêtes malveillantes.

**VPC Service Controls** : Isolation du trafic réseau et blocage de l'accès Internet public, confinant les données aux périmètres autorisés.

**Agent Engine Threat Detection** : Service intégré à Security Command Center permettant de détecter et investiguer les attaques potentielles sur les agents déployés.

**Access Transparency** : Journalisation des accès par le personnel Google aux ressources Agent Engine, garantissant la traçabilité pour les audits de conformité.

> **Bonnes pratiques**  
> Pour les charges de travail sensibles, activez systématiquement VPC Service Controls, configurez Model Armor avec des règles strictes, et implémentez une surveillance proactive via Security Command Center. Les tests adversariaux (red teaming) doivent faire partie intégrante du cycle de développement pour identifier les vulnérabilités avant la mise en production.

---

## II.6.6 Résumé

Ce chapitre a présenté Google Cloud Vertex AI comme environnement d'exploitation agentique complet. Les points essentiels à retenir sont :

**Architecture de la plateforme** : Vertex AI s'organise autour de trois piliers (Construire, Mettre à l'échelle, Gouverner) et s'intègre nativement avec l'écosystème Google Cloud pour la gestion des données, la sécurité et l'observabilité.

**Model Garden** : Catalogue de plus de 200 modèles fondamentaux incluant les modèles Gemini, les modèles partenaires (Claude, Llama) et les modèles open source, avec plusieurs modes de consommation et de personnalisation.

**Agent Builder** : Suite complète comprenant l'Agent Development Kit (ADK) pour le développement code-first, l'Agent Engine pour l'exécution managée, l'Agent Garden pour les composants préconçus, et le Cloud API Registry pour la gouvernance des outils.

**Développement d'agents** : L'ADK supporte plusieurs patrons architecturaux (pipeline séquentiel, orchestration hiérarchique, exécution parallèle, HITL) et s'intègre avec le RAG Engine pour l'ancrage contextuel dans les données d'entreprise.

**Environnements d'exécution** : L'Agent Engine Runtime offre déploiement simplifié, mise à l'échelle automatique et isolation sécurisée. Sessions et Memory Bank gèrent respectivement la mémoire à court et long terme. L'observabilité repose sur OpenTelemetry, Cloud Trace et Cloud Monitoring.

**Sécurité** : Approche multicouche avec identités IAM, Model Armor, VPC Service Controls, et détection des menaces via Security Command Center.

---

> **Points clés du chapitre**
>
> - Vertex AI fournit un environnement d'exploitation agentique complet couvrant le cycle de vie entier des agents
> - Le Model Garden offre plus de 200 modèles avec flexibilité de déploiement et de personnalisation
> - L'ADK permet le développement code-first d'agents sophistiqués avec support multi-langage
> - L'Agent Engine assure l'exécution managée avec Sessions (mémoire court terme) et Memory Bank (mémoire long terme)
> - La gouvernance des outils via Cloud API Registry est essentielle pour la sécurité organisationnelle
> - L'observabilité native avec OpenTelemetry et Cloud Trace répond aux défis du non-déterminisme agentique

---

*Ce chapitre prépare le terrain pour le Chapitre II.7, qui approfondira l'ingénierie du contexte et les stratégies RAG avancées pour ancrer les agents dans la réalité des données d'entreprise.*

*Chapitre suivant : Chapitre II.7 — Ingénierie du Contexte et RAG*
