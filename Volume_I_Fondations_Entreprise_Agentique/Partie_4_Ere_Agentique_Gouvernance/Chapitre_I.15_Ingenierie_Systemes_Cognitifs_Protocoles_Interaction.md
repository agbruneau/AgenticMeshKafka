# Chapitre I.15 — Ingénierie des Systèmes Cognitifs et Protocoles d'Interaction

---

## I.15.0 Introduction

Le chapitre précédent a établi l'architecture du maillage agentique comme infrastructure de collaboration entre agents cognitifs. Mais cette architecture ne peut fonctionner sans les mécanismes concrets qui permettent aux agents de raisonner efficacement, d'accéder aux connaissances pertinentes et de communiquer entre eux selon des protocoles standardisés.

Ce chapitre explore les disciplines d'ingénierie qui donnent vie aux systèmes cognitifs. L'ingénierie du contexte définit comment les agents accèdent aux informations dont ils ont besoin pour accomplir leurs tâches. La modélisation des workflows cognitifs structure les processus de raisonnement complexes. Les protocoles d'interopérabilité --- A2A et MCP --- établissent les standards de communication qui permettent aux agents de collaborer au sein du maillage agentique. Enfin, l'écosystème des cadriciels agentiques offre aux développeurs les outils nécessaires pour construire ces systèmes.

La maîtrise de ces techniques constitue un prérequis pour toute organisation aspirant à déployer des systèmes agentiques en production. Au-delà de la simple compréhension des concepts, ce chapitre vise à fournir les clés d'une mise en œuvre réussie.

## I.15.1 L'Ingénierie du Contexte : Prompt Engineering et RAG Avancé

L'efficacité d'un agent cognitif dépend fondamentalement de sa capacité à accéder aux informations pertinentes au moment opportun. Cette capacité repose sur deux disciplines complémentaires : l'ingénierie des prompts, qui optimise la formulation des instructions, et la génération augmentée par récupération (RAG), qui ancre les agents dans les données de l'entreprise.

### I.15.1.1 Les Fondamentaux du Prompt Engineering

Le prompt engineering désigne l'art et la science de formuler des instructions efficaces pour les grands modèles de langage. Loin d'être une simple rédaction de consignes, cette discipline exige une compréhension fine des mécanismes cognitifs des LLM et des patrons qui maximisent la qualité de leurs réponses.

Les techniques fondamentales incluent le **few-shot prompting**, où quelques exemples concrets guident le modèle vers le comportement attendu, et le **chain-of-thought**, qui encourage le raisonnement étape par étape plutôt que les réponses directes. Ces techniques ont démontré des améliorations significatives sur les tâches de raisonnement complexe.

Pour les agents cognitifs, le prompt engineering prend une dimension systémique. Le prompt système définit l'identité, les compétences et les contraintes de l'agent. Les prompts dynamiques s'adaptent au contexte de chaque interaction. Les métaprompts orchestrent le comportement de l'agent face à différentes situations.

> **Définition formelle**
>
> Prompt Engineering : Discipline d'ingénierie visant à optimiser les instructions fournies aux grands modèles de langage pour maximiser la qualité, la pertinence et la fiabilité de leurs réponses dans un contexte applicatif donné.

### I.15.1.2 La Génération Augmentée par Récupération (RAG)

Les grands modèles de langage, aussi puissants soient-ils, souffrent de limitations fondamentales : leurs connaissances sont figées à la date de leur entraînement et ils n'ont pas accès aux données propriétaires de l'entreprise. La génération augmentée par récupération (Retrieval-Augmented Generation, RAG) adresse ces limitations en injectant dynamiquement des informations externes dans le contexte du modèle.

Le processus RAG classique se déroule en trois étapes. L'indexation préalable transforme les documents de l'entreprise en représentations vectorielles (embeddings) stockées dans une base de données vectorielle. Lors d'une requête, la récupération identifie les documents les plus pertinents par similarité sémantique. Enfin, la génération utilise ces documents comme contexte pour produire une réponse fondée sur les données de l'entreprise.

Cette approche a révolutionné le déploiement des LLM en entreprise. Elle permet de réduire les hallucinations en ancrant les réponses dans des sources vérifiables, d'actualiser les connaissances sans réentraîner le modèle et de garantir la confidentialité en gardant les données sensibles dans l'infrastructure de l'entreprise.

### I.15.1.3 L'Évolution vers le RAG Agentique

Les systèmes RAG traditionnels suivent un workflow statique et linéaire. Le RAG agentique transcende cette rigidité en intégrant des agents autonomes dans le pipeline de récupération. Ces agents ne se contentent pas de récupérer passivement l'information; ils planifient dynamiquement leurs stratégies de recherche, évaluent la qualité des résultats et s'adaptent en temps réel.

> **Définition formelle**
>
> RAG Agentique (Agentic RAG) : Architecture de génération augmentée par récupération intégrant des agents autonomes capables de réflexion, planification, utilisation d'outils et collaboration multi-agents pour adapter dynamiquement les stratégies de récupération aux exigences de chaque requête.

Les patrons agentiques identifiés au Chapitre I.13 trouvent ici une application directe. La réflexion permet à l'agent d'évaluer la qualité des documents récupérés et de reformuler sa requête si nécessaire. La planification décompose les questions complexes en sous-requêtes ciblées. L'utilisation d'outils permet d'accéder à des sources variées --- bases vectorielles, graphes de connaissances, API externes. La collaboration multi-agents distribue la recherche entre agents spécialisés par domaine.

> **Exemple concret**
>
> Un système RAG agentique pour l'analyse financière reçoit la question : « Quel sera l'impact des nouvelles réglementations européennes sur notre stratégie d'investissement? » L'agent planificateur décompose en trois sous-requêtes : (1) récupérer les réglementations pertinentes, (2) analyser le portefeuille actuel, (3) identifier les secteurs impactés. Trois agents spécialisés exécutent ces recherches en parallèle. Un agent synthétiseur agrège les résultats. Un agent évaluateur vérifie la cohérence avant de produire la réponse finale. Cette orchestration dynamique surpasse largement un RAG linéaire sur des questions multifacettes.

## I.15.2 Modélisation des Workflows Cognitifs (DAG)

Les agents cognitifs accomplissent rarement leurs tâches en une seule étape. La plupart des processus agentiques impliquent des séquences d'actions, des branchements conditionnels, des itérations et des agrégations. La modélisation de ces workflows comme des graphes acycliques dirigés (DAG) fournit un cadre formel pour leur conception et leur exécution.

### I.15.2.1 Les Graphes comme Structure de Contrôle

Un graphe acyclique dirigé représente un workflow où les nœuds correspondent à des actions ou décisions et les arêtes aux transitions entre elles. L'absence de cycles garantit que le workflow progresse vers une terminaison, évitant les boucles infinies. Cette structure offre plusieurs avantages pour les systèmes cognitifs.

La visualisation explicite du flux de contrôle facilite la compréhension et le débogage des comportements complexes. La décomposition en nœuds indépendants permet la parallélisation des tâches non dépendantes. La traçabilité de chaque transition supporte l'audit et l'observabilité requis pour les systèmes en production. Enfin, la modularité autorise la réutilisation de sous-graphes dans différents contextes.

### I.15.2.2 Patrons de Workflows Cognitifs

L'expérience accumulée dans le développement de systèmes agentiques a fait émerger plusieurs patrons récurrents de workflows cognitifs.

**Tableau I.15.1 --- Patrons de workflows cognitifs**

| **Patron** | **Structure** | **Cas d'usage** |
|------------|---------------|-----------------|
| **Séquentiel** | Chaîne linéaire A → B → C | Pipelines de traitement de documents |
| **Parallèle** | Distribution puis agrégation | Recherches multi-sources simultanées |
| **Conditionnel** | Branchement selon le contexte | Routage vers agents spécialisés |
| **Itératif** | Boucle avec condition d'arrêt | Raffinement progressif de réponses |
| **Réflexif** | Évaluation et correction | Auto-amélioration des résultats |
| **Hiérarchique** | Superviseur et sous-agents | Décomposition de tâches complexes |

### I.15.2.3 Gestion de l'État et Persistance

La gestion de l'état constitue un défi majeur dans les workflows cognitifs. Contrairement aux programmes déterministes, les agents peuvent prendre des chemins imprévisibles, produire des résultats variables et nécessiter des interventions humaines. L'architecture doit prévoir la persistance de l'état à chaque étape.

Les cadriciels modernes comme LangGraph implémentent des mécanismes de **checkpointing** qui capturent l'état complet du workflow à chaque transition. Cette approche permet la reprise en cas d'échec, le débogage par rejeu des étapes passées et la mise en pause pour intervention humaine (*human-in-the-loop*). La persistance peut s'appuyer sur des stores en mémoire pour le développement ou sur des bases distribuées pour la production.

## I.15.3 Protocoles d'Interopérabilité Agentique (A2A, MCP)

Le maillage agentique décrit au chapitre précédent repose sur la capacité des agents à communiquer selon des protocoles standardisés. Deux protocoles émergent comme standards de facto : le Model Context Protocol (MCP) pour l'accès aux outils et données, et le protocole Agent-to-Agent (A2A) pour la communication inter-agents. Leur adoption rapide par l'industrie marque une étape décisive vers l'interopérabilité agentique.

### I.15.3.1 Model Context Protocol (MCP) : Le « USB-C » de l'IA

Le Model Context Protocol, introduit par Anthropic en novembre 2024, standardise la manière dont les applications d'IA accèdent aux sources de données et outils externes. Avant MCP, chaque combinaison modèle-outil nécessitait une intégration sur mesure, créant une explosion combinatoire difficile à maintenir. MCP réduit cette complexité à une équation simple : chaque application implémente le protocole client une fois, chaque outil implémente le protocole serveur une fois, et tout fonctionne ensemble.

> **Définition formelle**
>
> Model Context Protocol (MCP) : Standard ouvert permettant aux systèmes d'IA de se connecter de manière sécurisée et bidirectionnelle aux sources de données et outils externes, remplaçant les intégrations fragmentées par un protocole universel comparable à un « port USB-C » pour l'intelligence artificielle.

L'architecture MCP distingue trois composants. Les serveurs MCP exposent des ressources --- données, fonctions, capacités --- selon un schéma standardisé. Les clients MCP, typiquement des applications d'IA ou des agents, consomment ces ressources. Les hôtes orchestrent les connexions et gèrent les autorisations.

L'adoption de MCP a dépassé toutes les attentes. En un an, le protocole est passé d'une expérimentation open source au standard de facto de l'industrie. OpenAI, Google DeepMind et Microsoft ont annoncé leur support. Plus de 5 800 serveurs MCP sont désormais disponibles, couvrant les systèmes d'entreprise les plus répandus : Google Drive, Slack, GitHub, Salesforce, Stripe. En décembre 2025, Anthropic a cédé le standard à l'Agentic AI Foundation sous l'égide de la Linux Foundation, garantissant sa gouvernance neutre et son évolution communautaire.

### I.15.3.2 Agent-to-Agent Protocol (A2A) : La Communication Inter-Agents

Si MCP standardise l'accès aux outils, le protocole Agent-to-Agent (A2A) standardise la communication entre agents eux-mêmes. Introduit par Google en avril 2025, A2A permet à des agents construits sur des plateformes différentes, par des fournisseurs différents, de collaborer sur des tâches complexes.

A2A répond à un besoin croissant. À mesure que les organisations déploient des agents spécialisés, la nécessité de les faire collaborer devient critique. Un agent de recrutement doit communiquer avec un agent de vérification des antécédents. Un agent d'analyse financière doit interroger un agent de conformité réglementaire. Sans protocole commun, chaque paire d'agents nécessite une intégration ad hoc.

> **Définition formelle**
>
> Agent-to-Agent Protocol (A2A) : Standard ouvert de communication inter-agents permettant la découverte mutuelle des capacités, la négociation des modalités d'interaction et la gestion collaborative des tâches, indépendamment des frameworks ou vendeurs sous-jacents.

L'architecture A2A repose sur plusieurs concepts clés. L'Agent Card est un fichier JSON publié par chaque agent décrivant ses capacités, son point d'accès et ses méthodes d'authentification. Les tâches structurent les interactions : un agent client soumet une tâche à un agent serveur, qui peut la traiter immédiatement ou la gérer de manière asynchrone. Le protocole supporte les échanges textuels, fichiers et données structurées.

**Tableau I.15.2 --- Comparaison MCP et A2A**

| **Critère** | **MCP** | **A2A** |
|-------------|---------|---------|
| **Fonction principale** | Accès aux outils et données | Communication inter-agents |
| **Initiateur** | Anthropic (nov. 2024) | Google (avril 2025) |
| **Gouvernance** | Linux Foundation (AAIF) | Linux Foundation |
| **Partenaires** | OpenAI, Google, Microsoft, AWS | 150+ organisations |
| **Cas d'usage** | Agent ↔ Outil/Données | Agent ↔ Agent |
| **Transport** | JSON-RPC sur stdio/HTTP/SSE | JSON-RPC sur HTTPS, gRPC |
| **Complémentarité** | Connecte agents aux ressources | Connecte agents entre eux |

> **Perspective stratégique**
>
> MCP et A2A ne sont pas concurrents mais complémentaires. MCP permet à un agent d'accéder à ses outils et données; A2A lui permet de collaborer avec d'autres agents. Ensemble, ils forment l'infrastructure de communication du maillage agentique. Les entreprises devraient considérer ces protocoles comme des investissements stratégiques : leur adoption précoce facilite l'intégration future dans un écosystème agentique de plus en plus interconnecté.

## I.15.4 Écosystème des Cadriciels Agentiques

La construction de systèmes agentiques s'appuie sur un écosystème florissant de cadriciels (frameworks) qui abstraient la complexité sous-jacente et accélèrent le développement. Ces outils ont considérablement maturé depuis 2023, offrant désormais des capacités de production pour les déploiements d'entreprise.

### I.15.4.1 LangChain et LangGraph

LangChain s'est imposé comme le cadriciel de référence pour le développement d'applications basées sur les LLM. Sa force réside dans l'abstraction des composants récurrents --- chaînes de prompts, intégrations vectorielles, connexions aux modèles --- en modules réutilisables. Le LangChain Expression Language (LCEL) offre une syntaxe déclarative pour composer ces modules en pipelines sophistiqués.

LangGraph étend LangChain en introduisant une architecture de graphes pour les workflows agentiques complexes. Contrairement aux chaînes linéaires, les graphes permettent les branchements conditionnels, les cycles contrôlés et la gestion fine de l'état. Cette flexibilité fait de LangGraph le choix privilégié pour les applications nécessitant un contrôle précis du flux d'exécution et une traçabilité complète.

### I.15.4.2 Cadriciels Multi-Agents

Plusieurs cadriciels se spécialisent dans l'orchestration de systèmes multi-agents, chacun avec sa philosophie et ses forces.

**CrewAI** adopte une métaphore organisationnelle intuitive : les agents sont des membres d'une équipe (crew), assignés à des rôles avec des objectifs définis. Le cadriciel gère automatiquement la coordination, les transferts de contexte et la synthèse des résultats. Son approche « coordinateur-travailleurs » facilite le déploiement rapide de systèmes collaboratifs.

**AutoGen** de Microsoft privilégie les conversations multi-agents comme paradigme de collaboration. Les agents s'engagent dans des dialogues structurés pour résoudre des problèmes complexes. L'intégration avec Semantic Kernel et l'écosystème Azure en fait un choix naturel pour les environnements Microsoft.

**Google Agent Development Kit (ADK)** offre une approche code-first avec support natif des patrons multi-agents. Le SequentialAgent orchestre des chaînes de traitement, le ParallelAgent distribue les tâches, le LoopAgent gère les itérations. L'intégration native avec A2A et les services Google Cloud simplifie les déploiements sur cette plateforme.

**Tableau I.15.3 --- Comparaison des cadriciels agentiques**

| **Cadriciel** | **Architecture** | **Force principale** |
|---------------|------------------|----------------------|
| **LangGraph** | Graphes avec état | Contrôle fin et traçabilité |
| **CrewAI** | Coordinateur-travailleurs | Déploiement rapide |
| **AutoGen** | Conversations multi-agents | Écosystème Microsoft |
| **Google ADK** | Patrons multi-agents | Intégration Google Cloud |
| **Amazon Bedrock** | Agents managés | Services AWS natifs |
| **Vertex AI Agent Builder** | Low-code/No-code | Accessibilité métier |

> **Perspective stratégique**
>
> Le choix d'un cadriciel agentique dépend de plusieurs facteurs : le niveau d'expertise de l'équipe, la complexité des workflows envisagés, l'écosystème cloud existant et les exigences de production. Selon les analyses sectorielles, 72 % des projets d'IA d'entreprise impliquent désormais des architectures multi-agents. L'investissement dans la maîtrise de ces outils constitue un impératif stratégique pour les équipes d'ingénierie.

## I.15.5 Conclusion

L'ingénierie des systèmes cognitifs représente une discipline émergente à l'intersection du génie logiciel, de l'intelligence artificielle et de l'architecture d'entreprise. Ce chapitre a exploré les techniques fondamentales qui permettent de donner vie aux agents cognitifs au sein du maillage agentique.

L'ingénierie du contexte --- prompt engineering et RAG avancé --- définit comment les agents accèdent aux informations pertinentes. Le passage du RAG classique au RAG agentique illustre l'évolution vers des systèmes capables de planifier dynamiquement leurs stratégies de récupération plutôt que de suivre des workflows figés.

La modélisation des workflows cognitifs comme graphes acycliques dirigés fournit un cadre formel pour structurer les processus de raisonnement complexes. Les patrons récurrents --- séquentiels, parallèles, conditionnels, itératifs --- constituent une bibliothèque de solutions éprouvées.

Les protocoles d'interopérabilité MCP et A2A établissent les fondations de la communication standardisée. Leur adoption rapide par l'industrie --- y compris par des concurrents directs --- témoigne du besoin impérieux de standards ouverts pour le maillage agentique.

L'écosystème des cadriciels agentiques offre aux développeurs les outils nécessaires pour traduire ces concepts en implémentations concrètes. La diversité des approches --- de LangGraph à CrewAI, d'AutoGen à Google ADK --- reflète la richesse des cas d'usage et des philosophies de conception.

Ces compétences techniques ne constituent cependant qu'une partie de l'équation. Le chapitre suivant abordera le modèle opérationnel et la symbiose humain-agent, examinant comment les organisations s'adaptent à cette nouvelle réalité où les agents cognitifs deviennent des collaborateurs à part entière.

## I.15.6 Résumé

Ce chapitre a présenté les disciplines d'ingénierie essentielles à la construction de systèmes cognitifs :

**L'ingénierie du contexte :** Le prompt engineering optimise les instructions aux LLM via des techniques comme le few-shot prompting et le chain-of-thought. Le RAG ancre les agents dans les données de l'entreprise. Le RAG agentique transcende les workflows statiques en intégrant des agents autonomes capables de réflexion, planification et collaboration pour des stratégies de récupération dynamiques.

**Les workflows cognitifs :** La modélisation comme graphes acycliques dirigés (DAG) structure les processus de raisonnement complexes. Six patrons récurrents --- séquentiel, parallèle, conditionnel, itératif, réflexif, hiérarchique --- constituent une bibliothèque de solutions. La gestion de l'état via checkpointing permet la reprise, le débogage et l'intervention humaine.

**Les protocoles d'interopérabilité :** MCP (Anthropic, 2024) standardise l'accès aux outils et données --- le « USB-C » de l'IA. A2A (Google, 2025) standardise la communication inter-agents. Ces protocoles complémentaires, désormais sous gouvernance Linux Foundation, forment l'infrastructure de communication du maillage agentique.

**L'écosystème des cadriciels :** LangGraph offre un contrôle fin via les graphes avec état. CrewAI facilite le déploiement rapide avec son modèle coordinateur-travailleurs. AutoGen et Google ADK intègrent leurs écosystèmes respectifs. 72 % des projets d'IA d'entreprise impliquent désormais des architectures multi-agents.

**Tableau I.15.4 --- Synthèse des techniques d'ingénierie cognitive**

| **Discipline** | **Techniques clés** | **Évolution récente** |
|----------------|---------------------|----------------------|
| **Prompt Engineering** | Few-shot, Chain-of-thought | Métaprompts systémiques |
| **RAG** | Indexation, Récupération, Génération | RAG Agentique multi-agents |
| **Workflows** | DAG, Patrons récurrents | Checkpointing, Human-in-the-loop |
| **Protocoles** | MCP (outils), A2A (agents) | Gouvernance Linux Foundation |
| **Cadriciels** | LangGraph, CrewAI, AutoGen | Support natif multi-agents |

---

*Chapitre suivant : Chapitre I.16 — Modèle Opérationnel et la Symbiose Humain-Agent*
