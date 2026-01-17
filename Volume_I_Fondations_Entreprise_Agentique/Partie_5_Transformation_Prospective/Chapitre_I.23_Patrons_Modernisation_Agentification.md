# Chapitre I.23 — Patrons de Modernisation et d'Agentification

---

## I.23.0 Introduction

Le chapitre précédent a présenté l'APM cognitif comme outil de
pilotage, identifiant pour chaque application une stratégie d'action
parmi six options : retrait stratégique, encapsulation agentique,
enrichissement cognitif, modernisation préparatoire, remplacement
agentique et fédération. Ce chapitre détaille les patrons de mise en
oeuvre de ces stratégies — les approches concrètes pour transformer les
applications legacy en composantes du maillage agentique.

La modernisation des systèmes legacy représente un défi majeur pour les
organisations. Selon les estimations récentes, la dette technique
constitue 20 à 40 % de la valeur totale du patrimoine technologique, et
70 % des budgets IT sont consacrés à la maintenance d'infrastructures
obsolètes. Les approches traditionnelles de réécriture complète (« Big
Bang ») échouent fréquemment, tandis que les méthodes incrémentales
comme le patron Strangler Fig ont émergé comme standards architecturaux
pour la modernisation à risque maîtrisé.

Ce chapitre présente quatre patrons principaux d'agentification, chacun
adapté à un contexte spécifique identifié par l'évaluation APM
cognitive.

## I.23.1 Stratégies de Transformation Applicative (Les 6 R)

Les stratégies de transformation applicative, communément appelées les «
6 R », constituent un cadre de référence pour les décisions de
modernisation. Popularisées par Gartner puis enrichies par AWS, ces
stratégies couvrent le spectre complet des options de migration et de
transformation.

**Tableau I.23.1 — Les 6 R de la transformation applicative**

| **Stratégie** | **Description** | **Effort** | **Cas d'usage** |
|---------------|-----------------|------------|-----------------|
| **Rehost (Lift & Shift)** | Migration sans modification du code | Faible | Gains rapides, migration infonuagique |
| **Replatform** | Modifications mineures pour le nuage | Moyen-faible | Optimisation sans refonte |
| **Refactor/Rearchitect** | Restructuration pour le cloud-native | Élevé | Modernisation complète |
| **Repurchase** | Remplacement par SaaS | Moyen | Applications commoditisées |
| **Retire** | Décommissionnement | Faible | Applications obsolètes |
| **Retain** | Maintien en place | Minimal | Dépendances, contraintes légales |

Pour l'agentification, nous enrichissons ce cadre avec des stratégies
spécifiques qui intègrent la dimension cognitive. Les patrons présentés
dans ce chapitre s'appuient sur ces fondations tout en les adaptant aux
exigences du maillage agentique.

> **Définition formelle**
>
> Patron d'agentification : Approche architecturale éprouvée pour
> transformer une application existante en composante du maillage
> agentique. Chaque patron définit les conditions d'application, les
> étapes de mise en oeuvre, les interfaces à créer et les métriques de
> succès.

## I.23.2 Patron 1 : Le Retrait Stratégique

Le retrait stratégique s'applique aux applications identifiées comme
candidates à l'élimination dans l'évaluation TIME, avec un faible
potentiel d'agentification. Ce patron ne signifie pas un simple
décommissionnement, mais une transition planifiée qui préserve la valeur
métier tout en éliminant la charge technique.

### I.23.2.1 Conditions d'Application

Le retrait stratégique est approprié lorsque l'application présente une
faible valeur métier et une faible qualité technique, qu'elle n'offre
aucun potentiel réaliste d'agentification, que ses fonctionnalités sont
soit obsolètes soit disponibles ailleurs, et que le coût de maintenance
dépasse la valeur générée.

### I.23.2.2 Étapes de Mise en Oeuvre

La première étape consiste à inventorier les données et les processus
métier encore actifs dans l'application. Certaines données peuvent
avoir une valeur historique ou réglementaire même si l'application
elle-même est obsolète.

Ensuite, les données à préserver doivent être migrées vers des systèmes
appropriés — soit des applications existantes du portefeuille, soit
des solutions d'archivage. Les utilisateurs restants sont redirigés
vers des alternatives, accompagnés d'une formation si nécessaire.

Enfin, un plan de décommissionnement progressif est établi, incluant une
période de fonctionnement parallèle pour permettre le retour en arrière
si des dépendances inattendues sont découvertes.

> **Perspective stratégique**
>
> Le retrait stratégique libère des ressources pour l'innovation. Si
> une organisation consacre 70 % de son budget IT à la maintenance de
> systèmes legacy, chaque application retirée représente une capacité
> libérée pour la transformation agentique. Les organisations doivent
> considérer le retrait non comme un échec mais comme une décision
> stratégique créatrice de valeur.

## I.23.3 Patron 2 : L'Encapsulation Agentique

L'encapsulation agentique s'inspire du patron Strangler Fig, l'une
des approches les plus efficaces et les moins perturbatrices de
modernisation. Au lieu de modifier le coeur legacy, on construit une
couche d'interfaces modernes autour de celui-ci, exposant ses
fonctionnalités de manière standardisée pour que les agents cognitifs
puissent les consommer.

> **Définition formelle**
>
> Patron Strangler Fig : Stratégie architecturale de migration
> graduelle où un nouveau système « étrangle » progressivement
> l'ancien en le remplaçant fonction par fonction. Une façade (proxy)
> intercepte les requêtes et les route soit vers le système legacy soit
> vers les nouveaux services, permettant une transition sans
> interruption de service.

### I.23.3.1 Conditions d'Application

L'encapsulation agentique convient aux applications à haute qualité
technique mais faible valeur stratégique (quadrant « Tolérer » du TIME),
qui possèdent un potentiel d'agentification moyen à élevé grâce à des
données accessibles ou des processus automatisables, et où une
modernisation complète n'est pas justifiée économiquement.

### I.23.3.2 Architecture de l'Encapsulation

L'encapsulation repose sur trois composantes principales. Premièrement,
une passerelle API (API Gateway) agit comme couche de routage
intelligente, dirigeant le trafic entre le monolithe legacy et les
services modernes selon des règles métier et la progression de la
migration.

Deuxièmement, des wrappers API encapsulent les fonctionnalités legacy
dans des interfaces REST, GraphQL ou gRPC standardisées. Ces wrappers
traduisent les protocoles propriétaires en formats modernes consommables
par les agents.

Troisièmement, des adaptateurs événementiels permettent au système
legacy d'émettre et de consommer des événements sur le backbone Kafka,
l'intégrant ainsi au flux d'information du maillage agentique sans
modification de son code source.

**Tableau I.23.2 — Composantes de l'encapsulation agentique**

| **Composante** | **Fonction** | **Technologies** |
|----------------|--------------|------------------|
| **Passerelle API** | Routage, authentification, limitation de débit | Kong, Apigee, AWS API Gateway |
| **Wrappers API** | Traduction protocoles, exposition standardisée | REST, GraphQL, gRPC |
| **Adaptateurs événementiels** | Émission/consommation événements | Kafka Connect, Debezium (CDC) |
| **Couche anti-corruption** | Isolation domaine, traduction modèles | Hexagonal Architecture patterns |
| **Cache intelligent** | Réduction latence, découplage temporel | Redis, Hazelcast |

> **Exemple concret**
>
> Allianz, l'un des plus grands assureurs mondiaux, a modernisé ses
> systèmes d'assurance centraux en utilisant le patron Strangler Fig
> combiné au streaming de données. En implémentant Kafka comme backbone
> événementiel, Allianz a pu migrer graduellement de mainframes legacy
> vers une architecture cloud moderne et évolutive. Les nouveaux
> microservices traitent les données de réclamations en temps réel,
> améliorant vitesse et efficacité sans perturber les opérations
> existantes.

## I.23.4 Patron 3 : L'Enrichissement Cognitif

L'enrichissement cognitif s'applique aux applications à haute valeur
métier et haute qualité technique (quadrant « Investir » du TIME).
Plutôt que de les remplacer ou de simplement les encapsuler, ce patron
les augmente avec des capacités agentiques qui amplifient leur valeur.

### I.23.4.1 Conditions d'Application

L'enrichissement cognitif convient aux applications stratégiques qui
fonctionnent bien mais pourraient bénéficier d'automatisation
intelligente, de capacités prédictives, d'interfaces conversationnelles
ou d'aide à la décision augmentée. L'application doit disposer de
données exploitables pour l'enrichissement contextuel (RAG) et
d'interfaces permettant l'intégration d'agents.

### I.23.4.2 Modes d'Enrichissement

Le premier mode est le copilote intégré. Un agent cognitif est déployé
aux côtés de l'application existante, assistant les utilisateurs dans
leurs tâches sans modifier l'application elle-même. L'agent accède aux
données de l'application via API pour fournir des recommandations
contextuelles, automatiser des tâches répétitives et répondre aux
questions des utilisateurs.

Le deuxième mode est l'automatisation de processus. Des agents sont
déployés pour exécuter automatiquement des workflows qui traversent
l'application, déclenchés par des événements ou des conditions.
L'application devient un outil que les agents manipulent, plutôt qu'un
système que les humains opèrent directement.

Le troisième mode est l'analytique augmentée. Les données de
l'application alimentent des modèles prédictifs et des tableaux de bord
intelligents. Les agents analysent les patterns, détectent les anomalies
et génèrent des insights que l'application seule ne pourrait pas
produire.

**Tableau I.23.3 — Modes d'enrichissement cognitif**

| **Mode** | **Description** | **Cas d'usage** |
|----------|-----------------|-----------------|
| **Copilote intégré** | Agent assistant les utilisateurs | Support client, rédaction, recherche |
| **Automatisation de processus** | Agents exécutant des workflows | Approbations, réconciliations, rapports |
| **Analytique augmentée** | Modèles prédictifs sur les données | Prévisions, détection anomalies, scoring |
| **Interface conversationnelle** | Accès naturel aux fonctionnalités | Requêtes vocales, chatbots métier |
| **Aide à la décision** | Recommandations contextuelles | Pricing, allocation, priorisation |

### I.23.4.3 Intégration RAG pour l'Enrichissement Contextuel

L'enrichissement cognitif repose souvent sur la technique RAG
(Retrieval-Augmented Generation) présentée au Chapitre I.15. Les
documents, manuels et données historiques de l'application sont indexés
dans une base vectorielle, permettant aux agents de répondre aux
questions avec un contexte spécifique à l'application.

Cette approche transforme la documentation passive en connaissance
active. Un agent enrichi par RAG peut expliquer les fonctionnalités de
l'application, guider les utilisateurs dans les processus complexes et
résoudre les problèmes en s'appuyant sur l'historique des incidents
similaires.

## I.23.5 Patron 4 : La Promotion et la Fédération

Le patron de promotion et fédération s'applique aux applications qui,
après évaluation, se révèlent être des candidates idéales pour devenir
des services centraux du maillage agentique. Ces applications sont «
promues » au rang de capacités partagées, exposées à l'ensemble de
l'écosystème d'agents.

### I.23.5.1 Conditions d'Application

La promotion est appropriée pour les applications à haute valeur métier
et haute qualité technique, avec un potentiel d'agentification élevé,
dont les fonctionnalités sont utiles à plusieurs domaines de
l'entreprise. Ces applications possèdent des données ou des capacités
qui pourraient enrichir le contexte d'autres agents du maillage.

### I.23.5.2 Architecture de Fédération

La fédération transforme l'application en service du maillage
agentique, exposé via les protocoles standardisés A2A (Agent-to-Agent)
et MCP (Model Context Protocol). Les autres agents peuvent découvrir et
invoquer ses capacités de manière déclarative, sans intégration
point-à-point.

L'application promue devient un « outil » au sens des cadriciels
agentiques — une capacité que les agents peuvent utiliser pour
accomplir leurs objectifs. Elle publie un manifeste décrivant ses
capacités, ses paramètres d'entrée et ses formats de sortie, permettant
aux agents de l'invoquer de manière autonome.

> **Perspective stratégique**
>
> La tendance 2025 combine le patron Strangler Fig avec le streaming
> événementiel (Apache Kafka). Plutôt que de simplement router des
> requêtes HTTP, les organisations utilisent la Capture de Données de
> Changement (CDC) pour synchroniser les bases legacy avec des magasins
> de données modernes en temps réel, permettant aux opérations de
> lecture d'être déchargées vers la pile moderne avant même que la
> logique d'écriture ne soit touchée.

### I.23.5.3 Rôle de l'IA Générative dans la Modernisation

Les équipes d'entreprise utilisent de plus en plus les outils d'IA
générative pour accélérer la modernisation. Ces outils analysent la
logique du code legacy (par exemple, extraction des règles métier du
COBOL) pour accélérer la création des microservices de remplacement,
rendant l'approche Strangler Fig plus rapide que jamais.

Les escouades d'agents spécialisés peuvent automatiser l'analyse du
code source, la génération de tests, le refactoring et la documentation.
Cette assistance IA réduit considérablement le temps et l'effort requis
pour la modernisation tout en préservant la logique métier critique
enfouie dans les systèmes legacy.

## I.23.6 Conclusion

Les quatre patrons présentés constituent un catalogue d'actions pour
l'agentification. Chaque patron correspond à un profil d'application
identifié par l'évaluation APM cognitive, assurant que les ressources
de transformation sont investies là où elles créent le plus de valeur.

**Tableau I.23.4 — Catalogue des patrons d'agentification**

| **Patron** | **Profil APM** | **Résultat** |
|------------|----------------|--------------|
| **Retrait stratégique** | Éliminer + Potentiel faible | Libération ressources, réduction dette |
| **Encapsulation agentique** | Tolérer + Potentiel moyen-élevé | Intégration au maillage sans modification |
| **Enrichissement cognitif** | Investir + Potentiel élevé | Augmentation capacités par agents |
| **Promotion et fédération** | Investir + Potentiel élevé | Service partagé du maillage agentique |

L'approche incrémentale est essentielle. Contrairement aux réécritures
« Big Bang » qui convertissent une dépense d'investissement (CAPEX)
risquée en une livraison continue à risque contrôlé, les patrons
présentés permettent de valider l'approche rapidement et d'apprendre
des erreurs tôt dans le processus.

Le chapitre suivant (I.24) aborde l'industrialisation de ces approches
via l'ingénierie de plateforme, montrant comment mettre à l'échelle
les pratiques d'agentification à travers l'organisation.

## I.23.7 Résumé

Ce chapitre a présenté les patrons de modernisation et d'agentification
:

**Les 6 R de transformation :** Rehost (lift & shift), Replatform
(modifications mineures), Refactor/Rearchitect (restructuration
cloud-native), Repurchase (SaaS), Retire (décommissionnement), Retain
(maintien). Cadre enrichi pour l'agentification avec dimension
cognitive.

**Patron 1 — Retrait stratégique :** Applications TIME « Éliminer » +
potentiel faible. Inventaire données/processus, migration vers archivage
ou alternatives, décommissionnement progressif avec période parallèle.
Libère 70 % budget IT maintenance pour innovation.

**Patron 2 — Encapsulation agentique :** Applications TIME « Tolérer
» + potentiel moyen-élevé. Patron Strangler Fig : façade (proxy) routant
vers legacy ou nouveaux services. Composantes : passerelle API, wrappers
API, adaptateurs événementiels, couche anti-corruption, cache
intelligent. Exemple Allianz : migration mainframes vers cloud via Kafka
sans interruption.

**Patron 3 — Enrichissement cognitif :** Applications TIME « Investir
» + potentiel élevé. Modes : copilote intégré (assistant utilisateurs),
automatisation processus (agents exécutant workflows), analytique
augmentée (prédictions, anomalies), interface conversationnelle, aide à
la décision. RAG pour enrichissement contextuel.

**Patron 4 — Promotion et fédération :** Applications TIME « Investir
» + potentiel élevé + utilité multi-domaines. Transformation en service
du maillage via A2A/MCP. Application devient « outil » invocable par
agents. Manifeste déclaratif des capacités. Tendance 2025 : Strangler
Fig + streaming Kafka + CDC.

**Accélération par IA générative :** Outils GenAI analysent logique code
legacy (COBOL), accélèrent création microservices. Escouades d'agents
automatisent analyse, tests, refactoring, documentation. Dette technique
= 20-40 % valeur patrimoine. Approche incrémentale (OPEX) vs Big Bang
(CAPEX) réduit risque à quasi-zéro.

**Tableau I.23.5 — Synthèse des patrons et leurs indicateurs de succès**

| **Patron** | **Indicateur de succès** | **Risque maîtrisé** |
|------------|--------------------------|---------------------|
| **Retrait stratégique** | Coût maintenance réduit | Perte données/fonctions critiques |
| **Encapsulation** | Agents consomment API legacy | Latence, couplage façade |
| **Enrichissement** | Productivité utilisateurs améliorée | Adoption agent, qualité RAG |
| **Promotion/Fédération** | Réutilisation inter-domaines | Gouvernance, disponibilité service |

---

*Le Chapitre I.24 présente l'Industrialisation via l'Ingénierie de
Plateforme — comment mettre à l'échelle les pratiques
d'agentification à travers l'organisation.*
