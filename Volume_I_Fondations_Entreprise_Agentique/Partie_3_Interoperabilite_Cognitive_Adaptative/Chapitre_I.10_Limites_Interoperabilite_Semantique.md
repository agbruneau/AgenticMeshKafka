# Chapitre I.10 — Limites de l'Interopérabilité Sémantique Traditionnelle

---

## I.10.0 Introduction

La Partie 2 a établi les fondations techniques de l'architecture réactive : API, événements, contrats de données, infrastructure observable. Ces composantes constituent le substrat technique de l'interopérabilité --- la capacité des systèmes à communiquer. Mais communiquer ne signifie pas comprendre. Cette Partie 3 franchit un nouveau seuil : celui de l'interopérabilité sémantique et cognitive, où la question n'est plus « les systèmes peuvent-ils échanger des données? » mais « peuvent-ils se comprendre? ».

Ce chapitre inaugure cette exploration en examinant les approches traditionnelles de l'interopérabilité sémantique --- ontologies formelles, gestion des données de référence, modèles canoniques --- et leurs limites fondamentales. Ces approches, développées sur plusieurs décennies, ont apporté des contributions précieuses. Mais elles se heurtent à des obstacles qui deviennent insurmontables à mesure que les systèmes gagnent en complexité, en dynamisme et en autonomie.

Comprendre ces limites n'est pas un exercice académique. C'est le préalable nécessaire à l'adoption d'approches nouvelles. L'entreprise agentique, où des agents cognitifs autonomes doivent interpréter des contextes ambigus et prendre des décisions en situation d'incertitude, ne peut pas s'appuyer sur des mécanismes sémantiques qui supposent un monde stable, exhaustivement modélisé et exempt d'ambiguïté. Les chapitres suivants proposeront des alternatives; celui-ci établit pourquoi ces alternatives sont nécessaires.

## I.10.1 Le Rôle et les Limites des Ontologies Formelles (RDF, OWL)

Les ontologies formelles représentent l'effort le plus ambitieux pour résoudre le problème de l'interopérabilité sémantique. Héritières de la tradition logiciste en intelligence artificielle, elles proposent de modéliser explicitement les concepts, les relations et les règles d'un domaine dans un langage formel manipulable par les machines.

> **Définition formelle**
>
> *Ontologie (en informatique) : Spécification formelle et explicite d'une conceptualisation partagée. Une ontologie définit les types d'entités qui existent dans un domaine, leurs propriétés, les relations entre elles et les contraintes qui s'appliquent. Elle fournit un vocabulaire commun et des axiomes permettant le raisonnement automatique.*

Le **Web sémantique**, vision proposée par Tim Berners-Lee au début des années 2000, a donné naissance aux standards qui structurent le domaine. **RDF (Resource Description Framework)** fournit le modèle de données de base : des triplets sujet-prédicat-objet qui expriment des faits atomiques. **RDFS (RDF Schema)** ajoute les concepts de classe et de hiérarchie. **OWL (Web Ontology Language)** étend ces capacités avec des constructeurs logiques permettant d'exprimer des axiomes complexes : équivalences, disjonctions, restrictions de cardinalité, propriétés transitives.

La promesse des ontologies était séduisante : en définissant formellement la signification des termes utilisés dans les échanges, les systèmes pourraient automatiquement traduire entre vocabulaires différents, détecter les incohérences, inférer de nouvelles connaissances. Un agent logiciel découvrant un nouveau service pourrait, en consultant son ontologie, comprendre les données qu'il manipule et comment les utiliser.

> **Exemple concret**
>
> *L'ontologie FIBO (Financial Industry Business Ontology), développée par l'EDM Council, tente de standardiser les concepts du domaine financier : instruments, parties, contrats, événements. Elle compte des milliers de classes et de propriétés, organisées en modules (titres, dérivés, prêts, etc.). Son objectif : permettre aux institutions financières de partager des données avec une sémantique non ambiguë, facilitant la conformité réglementaire et l'intégration inter-organisationnelle.*

Malgré ces promesses et des investissements considérables, les ontologies formelles ont rencontré des obstacles qui limitent leur adoption et leur efficacité. Ces limites ne sont pas des défauts d'implémentation; elles sont inhérentes à l'approche elle-même.

### I.10.1.1 Le Goulot d'Étranglement de l'Acquisition des Connaissances

La construction d'une ontologie de qualité est un processus extraordinairement coûteux. Il requiert l'expertise combinée de spécialistes du domaine (qui comprennent les concepts mais pas la formalisation) et d'ontologistes (qui maîtrisent le formalisme mais pas le domaine). Cette collaboration, difficile à organiser, produit des résultats qui doivent ensuite être validés par la communauté d'utilisateurs.

Le problème s'aggrave avec la taille et la complexité du domaine. Une ontologie couvrant un domaine riche comme la médecine, la finance ou la logistique peut contenir des dizaines de milliers de concepts et de relations. Chaque concept doit être défini avec précision, positionné dans la hiérarchie, relié aux concepts connexes. Chaque axiome doit être vérifié pour éviter les incohérences logiques qui rendraient le raisonnement invalide.

Ce **« goulot d'étranglement de l'acquisition des connaissances »** (knowledge acquisition bottleneck), identifié dès les années 1980 par les chercheurs en systèmes experts, n'a jamais été véritablement résolu. Les outils d'aide à la construction d'ontologies (Protégé, TopBraid) facilitent la tâche mais ne l'éliminent pas. L'effort reste manuel, lent et sujet à erreur.

### I.10.1.2 Le Défi de la Maintenance Continue

Une ontologie n'est pas un artefact statique. Les domaines évoluent : nouveaux produits, nouvelles réglementations, nouvelles pratiques. Chaque évolution peut nécessiter des modifications de l'ontologie : ajout de concepts, révision des définitions, réorganisation des hiérarchies. Ces modifications doivent être propagées à tous les systèmes qui utilisent l'ontologie, sous peine de créer des incohérences.

La maintenance d'une ontologie partagée entre plusieurs organisations est particulièrement problématique. Qui a l'autorité pour décider des modifications? Comment gérer les versions concurrentes? Comment s'assurer que tous les utilisateurs adoptent les mises à jour? Ces questions de gouvernance, au-delà des aspects techniques, ont fait échouer de nombreux projets d'ontologies partagées.

> **Perspective stratégique**
>
> *Le coût de maintenance des ontologies est souvent sous-estimé lors de leur création. Une ontologie qui n'est pas maintenue devient rapidement obsolète, perdant sa valeur comme référentiel partagé. Les organisations qui s'engagent dans cette voie doivent prévoir des ressources permanentes pour la gouvernance et l'évolution de leurs ontologies --- un investissement que beaucoup ne sont pas prêtes à consentir.*

### I.10.1.3 Les Limites Expressives des Formalismes

Les langages ontologiques comme OWL sont fondés sur des logiques de description (description logics), variantes de la logique du premier ordre. Ces formalismes, bien que puissants, ne peuvent pas capturer tous les aspects de la signification. Plusieurs dimensions échappent à leur expressivité.

Le **contexte pragmatique** --- l'usage qui est fait d'un concept dans une situation particulière --- ne peut pas être formalisé. Le mot « client » a une définition différente selon qu'on parle de vente, de support ou de contentieux. Ces nuances contextuelles, évidentes pour un humain, sont opaques à un système de raisonnement ontologique.

Les **connaissances tacites** --- le savoir-faire implicite des experts --- résistent à la formalisation. Un médecin expérimenté « sent » qu'un patient est gravement malade avant de pouvoir expliciter tous les indices qui le conduisent à ce jugement. Ce type de connaissance, crucial pour la prise de décision, ne se laisse pas capturer dans des axiomes logiques.

L'**incertitude et la gradualité** sont mal gérées par les ontologies classiques. Le monde réel est plein de cas limites, de situations ambiguës, de jugements nuancés. Les ontologies, par construction, définissent des frontières nettes entre concepts. Un objet est ou n'est pas une instance d'une classe; il n'y a pas de « presque » en logique classique.

## I.10.2 Les Défis de la Gestion des Données de Référence (MDM)

Face aux difficultés des ontologies académiques, le monde de l'entreprise a développé une approche plus pragmatique : la gestion des données de référence (Master Data Management ou MDM). Plutôt que de modéliser formellement tous les concepts d'un domaine, le MDM se concentre sur les entités métier critiques --- clients, produits, fournisseurs, localisations --- et cherche à en établir une version unique et fiable, partagée à travers l'organisation.

> **Définition formelle**
>
> *Master Data Management (MDM) : Discipline et ensemble de pratiques visant à créer et maintenir une source unique et autoritaire (« golden record ») pour les entités métier critiques de l'organisation. Le MDM englobe les processus de gouvernance, les règles de qualité des données et les outils technologiques nécessaires à cette unification.*

L'enjeu du MDM est concret et urgent pour les grandes organisations. Un même client peut apparaître sous des formes légèrement différentes dans le CRM, le système de facturation, l'entrepôt de données marketing et le système de support. Ces variations --- fautes de frappe, abréviations, données obsolètes --- créent des incohérences qui faussent les analyses, dégradent l'expérience client et génèrent des coûts opérationnels considérables.

### I.10.2.1 Les Approches Architecturales du MDM

Plusieurs architectures MDM ont été développées, chacune avec ses compromis entre cohérence et agilité.

L'approche **centralisée (registry style)** crée un référentiel central qui stocke les attributs clés des entités maîtres. Les systèmes sources conservent leurs données mais s'alignent sur les identifiants du référentiel central. Cette approche minimise les perturbations mais ne résout pas les divergences dans les attributs non centralisés.

L'approche **consolidée (repository style)** va plus loin : le référentiel central devient la source de vérité pour tous les attributs des entités maîtres. Les systèmes sources doivent synchroniser leurs données avec ce référentiel. Cette approche maximise la cohérence mais impose une gouvernance lourde et peut créer des goulots d'étranglement.

L'approche **coexistence (hybrid style)** combine les deux précédentes : certains attributs sont maîtrisés centralement, d'autres restent dans les systèmes sources. Cette flexibilité est souvent nécessaire en pratique mais complexifie la gouvernance.

| **Approche** | **Avantages** | **Limites** |
|--------------|---------------|-------------|
| **Centralisée (Registry)** | Déploiement rapide, perturbation minimale | Cohérence partielle, divergences persistantes |
| **Consolidée (Repository)** | Cohérence maximale, source unique de vérité | Gouvernance lourde, rigidité, goulot d'étranglement |
| **Hybride (Coexistence)** | Flexibilité, adaptation au contexte | Complexité de gouvernance, règles multiples |

### I.10.2.2 Les Causes Récurrentes d'Échec des Initiatives MDM

Les projets MDM ont un taux d'échec remarquablement élevé. Selon diverses études de l'industrie, entre 50 % et 80 % des initiatives MDM ne produisent pas les bénéfices attendus. Ces échecs ne sont généralement pas dus à des défaillances technologiques mais à des facteurs organisationnels et conceptuels.

La **sous-estimation de la complexité politique** est une cause fréquente. Le MDM touche à la propriété des données --- un sujet sensible dans toute organisation. Quelle division « possède » la définition du client? Qui a l'autorité pour décider qu'un enregistrement est le « golden record »? Ces questions de pouvoir, déguisées en questions techniques, peuvent paralyser les projets.

> **Exemple concret**
>
> *Une grande banque a investi 50 millions de dollars sur trois ans dans un programme MDM pour unifier sa vision du client à travers les divisions retail, corporate et investment banking. Le projet a échoué principalement parce que chaque division avait une définition différente du « client » qui reflétait ses besoins métier spécifiques. Un client retail et un client corporate de la même entreprise devaient-ils être fusionnés? Les règles de confidentialité permettaient-elles de croiser les données? Ces questions, non résolues avant le lancement, ont conduit à une impasse politique.*

La **rigidité face à l'évolution métier** constitue une autre limite. Les référentiels MDM sont conçus autour de modèles de données relativement stables. Mais le métier évolue : nouveaux segments de clientèle, nouveaux types de produits, nouvelles structures organisationnelles. Chaque évolution requiert une modification du modèle central, un processus souvent lent et laborieux qui crée un décalage entre le référentiel et la réalité opérationnelle.

Le **problème de la qualité à la source** est souvent négligé. Le MDM peut nettoyer et consolider les données existantes, mais il ne peut pas empêcher la création de nouvelles données de mauvaise qualité dans les systèmes sources. Sans discipline de saisie à l'origine, le référentiel central se dégrade progressivement, nécessitant des efforts de nettoyage récurrents.

> **Perspective stratégique**
>
> *L'approche Data Mesh, évoquée au Chapitre I.7, propose une alternative au MDM centralisé. Au lieu d'un référentiel unique, chaque domaine métier est responsable de ses données et les expose comme des produits avec des contrats explicites. Cette décentralisation résout certains problèmes de gouvernance mais en crée d'autres, notamment celui de la cohérence transverse. Le Data Mesh n'élimine pas le besoin d'interopérabilité sémantique; il le redistribue.*

## I.10.3 Le Fossé Sémantique : Quand le Contexte Dépasse la Définition

Au-delà des difficultés pratiques des ontologies et du MDM, un problème plus fondamental se pose : la signification des données ne peut pas être entièrement capturée par des définitions statiques. Elle dépend du contexte dans lequel ces données sont utilisées --- contexte qui est souvent implicite, variable et impossible à anticiper exhaustivement.

> **Définition formelle**
>
> *Fossé sémantique (Semantic Gap) : Écart entre la représentation formelle d'un concept (sa définition dans un schéma ou une ontologie) et sa signification effective dans un contexte d'usage particulier. Ce fossé résulte de l'impossibilité de capturer toutes les nuances contextuelles dans une définition statique.*

Le fossé sémantique se manifeste dans de nombreuses situations quotidiennes de l'entreprise. Considérons quelques exemples représentatifs.

### I.10.3.1 La Polysémie des Termes Métier

Les termes métier courants sont souvent polysémiques --- ils ont plusieurs significations selon le contexte. Le mot « compte » signifie quelque chose de différent pour un comptable (une ligne dans le plan comptable), un gestionnaire de relation client (une entreprise cliente), un informaticien (un identifiant d'accès) et un community manager (un profil sur un réseau social).

Ces distinctions peuvent sembler évidentes, mais elles créent des problèmes réels d'interopérabilité. Lorsqu'un système de CRM envoie un message concernant un « compte » à un système financier, l'interprétation peut différer. Une ontologie bien construite distinguerait ces concepts par des termes différents (« CustomerAccount », « LedgerAccount », « UserAccount »), mais cette distinction suppose que les concepteurs aient anticipé tous les usages et que tous les systèmes adhèrent à cette taxonomie.

> **Exemple concret**
>
> *Un projet d'intégration dans le secteur de la santé a révélé que le terme « visite » avait 17 significations différentes selon les systèmes : visite de consultation, visite de suivi, visite d'urgence, visite de téléconsultation, visite à domicile, visite de courtoisie, et ainsi de suite. Chaque système avait sa propre définition, reflétant les processus métier de son département d'origine. L'harmonisation a nécessité la création d'un modèle conceptuel complexe avec des sous-types et des attributs discriminants --- modèle que personne n'utilisait dans la pratique quotidienne.*

### I.10.3.2 La Dimension Temporelle du Sens

La signification des données dépend aussi du moment où elles sont considérées. Un « client actif » n'a pas la même définition selon qu'on analyse l'activité du mois dernier, de l'année dernière ou de la relation historique complète. Un « prix » peut être un prix catalogue, un prix négocié, un prix promotionnel, un prix historique --- et la distinction pertinente dépend de l'usage.

Les ontologies et les schémas de données peinent à capturer cette dimension temporelle du sens. Ils peuvent modéliser des horodatages (« date de création », « date de modification »), mais pas les règles contextuelles qui déterminent quelle version d'une donnée est pertinente pour quel usage. Ces règles sont souvent implicites dans les processus métier et varient selon les cas d'usage.

### I.10.3.3 Le Contexte Organisationnel et Culturel

Les organisations développent des cultures et des jargons propres qui influencent la signification des termes. Un « projet stratégique » dans une startup peut signifier un développement de trois semaines; dans une grande entreprise, il peut désigner une initiative pluriannuelle avec des dizaines d'intervenants. Ces nuances culturelles, évidentes pour les initiés, sont invisibles dans les définitions formelles.

Lors des fusions et acquisitions, ce fossé culturel devient un obstacle majeur à l'intégration des systèmes. Deux entreprises qui utilisent apparemment le même terme pour désigner le même concept découvrent, lors de l'intégration, que leurs définitions opérationnelles divergent de manières subtiles mais significatives.

> **Perspective stratégique**
>
> *Le fossé sémantique n'est pas un problème à résoudre définitivement; c'est une réalité à gérer continuellement. Les approches traditionnelles tentent d'éliminer l'ambiguïté par des définitions plus précises et plus complètes. L'approche cognitive accepte l'ambiguïté comme inhérente et développe des mécanismes pour l'interpréter en contexte --- c'est la direction que prendront les agents cognitifs de l'entreprise agentique.*

## I.10.4 La Rigidité des Modèles Canoniques face à la Dynamique Métier

Face aux défis de l'interopérabilité sémantique, une approche pragmatique s'est largement répandue dans les architectures d'intégration : le modèle canonique. L'idée est de définir un format de données commun, intermédiaire entre tous les systèmes, vers lequel et depuis lequel toutes les données sont traduites. Cette approche, intuitive et pratique, comporte cependant des limites structurelles qui deviennent problématiques dans les environnements dynamiques.

> **Définition formelle**
>
> *Modèle canonique (Canonical Data Model) : Représentation standardisée des données d'un domaine, indépendante des systèmes sources et cibles, utilisée comme format pivot dans les architectures d'intégration. Chaque système traduit ses données vers le modèle canonique (mapping entrant) et depuis le modèle canonique (mapping sortant).*

Le modèle canonique réduit théoriquement la complexité des intégrations. Sans modèle canonique, N systèmes nécessitent potentiellement N×(N-1) mappings point à point. Avec un modèle canonique, on a seulement 2×N mappings (un entrant et un sortant par système). Ce gain mathématique explique l'attrait de l'approche.

### I.10.4.1 Le Problème du Plus Petit Dénominateur Commun

Le modèle canonique doit pouvoir représenter toutes les données de tous les systèmes connectés. En pratique, deux stratégies s'opposent. La stratégie du « plus petit dénominateur commun » ne retient que les attributs présents dans tous les systèmes --- perdant ainsi la richesse des données spécifiques. La stratégie de l'« union » inclut tous les attributs de tous les systèmes --- créant un modèle tentaculaire, largement vide pour chaque message particulier.

Ni l'une ni l'autre de ces stratégies n'est satisfaisante. La première perd de l'information; la seconde devient ingérable. Les approches hybrides tentent de trouver un équilibre, distinguant un « noyau commun » d'« extensions optionnelles », mais cette distinction est elle-même sujette à débat et à évolution.

> **Exemple concret**
>
> *Un intégrateur de systèmes de santé a développé un modèle canonique pour le « dossier patient ». Le modèle initial, basé sur les besoins de trois hôpitaux, comptait 200 attributs. Après l'intégration de 10 établissements supplémentaires, le modèle avait gonflé à 1 500 attributs, la plupart vides dans la majorité des messages. Les transformations étaient devenues si complexes que les temps de traitement avaient été multipliés par cinq, et les erreurs de mapping représentaient 30 % des tickets de support.*

### I.10.4.2 L'Inertie du Modèle Canonique

Une fois établi, le modèle canonique devient difficile à modifier. Chaque changement --- ajout d'attribut, modification de type, réorganisation de structure --- impacte potentiellement tous les systèmes connectés. Les équipes d'intégration, conscientes de ce risque, deviennent conservatrices : elles préfèrent « bricoler » des solutions de contournement plutôt que de faire évoluer le modèle.

Cette inertie crée un décalage croissant entre le modèle canonique et la réalité métier. Les nouveaux besoins sont accommodés par des conventions ad hoc, des champs « fourre-tout » et des métadonnées informelles. Le modèle canonique, conçu pour simplifier l'intégration, devient progressivement une source de complexité et de dette technique.

Le phénomène de **« modèle canonique zombie »** décrit les situations où le modèle officiel n'est plus maintenu ni respecté, mais continue d'exister formellement parce que le coût de son remplacement est perçu comme prohibitif. Les équipes développent des pratiques parallèles, des « dialectes » du modèle canonique, qui fragmentent à nouveau l'interopérabilité que le modèle était censé garantir.

### I.10.4.3 L'Incompatibilité avec les Pratiques Agiles

Les modèles canoniques ont été conçus à une époque où les cycles de développement étaient longs et les changements rares. Dans un contexte agile, où les équipes livrent des incréments toutes les deux semaines et où les besoins évoluent continuellement, le processus de modification du modèle canonique devient un goulot d'étranglement.

La gouvernance du modèle canonique implique typiquement des comités de revue, des analyses d'impact, des validations croisées. Ces processus, justifiés par le risque de rupture, rallongent le délai entre l'identification d'un besoin et sa prise en compte. Les équipes agiles, frustrées par ces délais, contournent le modèle canonique par des intégrations directes, recréant la complexité point à point que le modèle était censé éliminer.

> **Perspective stratégique**
>
> *Les contrats de données décentralisés, présentés au Chapitre I.7, offrent une alternative au modèle canonique centralisé. Au lieu d'un schéma unique imposé à tous, chaque producteur définit son propre contrat, et les consommateurs s'adaptent. Cette approche transfère la responsabilité de l'interprétation vers les consommateurs, qui peuvent utiliser des techniques de mapping flexibles --- y compris, à terme, des agents cognitifs capables d'interpréter des structures variées.*

## I.10.5 Conclusion

Ce chapitre a examiné les approches traditionnelles de l'interopérabilité sémantique et leurs limites fondamentales. Les ontologies formelles offrent une rigueur logique mais butent sur les coûts d'acquisition et de maintenance des connaissances, ainsi que sur les limites expressives des formalismes. Le MDM adresse des besoins concrets de cohérence des données mais échoue souvent face aux complexités politiques et à la rigidité des modèles. Les modèles canoniques simplifient théoriquement les intégrations mais créent une inertie incompatible avec l'agilité moderne.

Ces approches partagent une hypothèse commune : que la signification peut être définie a priori, de manière exhaustive et stable. Cette hypothèse est de moins en moins tenable dans un monde où les métiers évoluent rapidement, où les contextes d'usage se multiplient, où l'incertitude et l'ambiguïté sont la norme plutôt que l'exception.

L'entreprise agentique amplifie ces défis. Les **agents cognitifs** doivent interpréter des données provenant de sources multiples, dans des contextes variés, pour prendre des décisions autonomes. Ils ne peuvent pas s'appuyer sur des ontologies exhaustives qui n'existent pas, ni sur des modèles canoniques figés qui ne captent pas les nuances. Ils ont besoin d'une **interopérabilité adaptative** --- capable d'interpréter le sens en contexte, de gérer l'incertitude, de s'adapter aux évolutions.

Le chapitre suivant explorera comment l'intelligence artificielle, et particulièrement les grands modèles de langage, peut transformer l'interopérabilité en lui conférant des capacités d'interprétation contextuelle que les approches formelles ne peuvent pas offrir. Cette évolution --- du formel au cognitif, du statique à l'adaptatif --- constitue le cœur de la transition vers l'Interopérabilité Cognitivo-Adaptative (ICA) que nous définirons au Chapitre I.12.

La reconnaissance des limites des approches traditionnelles n'est pas un constat d'échec. Ces approches ont apporté des contributions précieuses et restent pertinentes dans certains contextes --- domaines stables, vocabulaires bien définis, besoins de rigueur formelle. Mais elles ne peuvent pas, seules, répondre aux exigences de l'entreprise agentique. L'avenir réside dans leur combinaison avec des capacités cognitives nouvelles, dans une architecture hybride qui tire le meilleur de chaque approche.

## I.10.6 Résumé

Ce chapitre a analysé les limites des approches traditionnelles de l'interopérabilité sémantique, préparant le terrain pour les alternatives cognitives :

**Les ontologies formelles (RDF, OWL)** offrent une rigueur logique pour modéliser les concepts et leurs relations. Cependant, elles se heurtent au goulot d'étranglement de l'acquisition des connaissances (coût de construction), au défi de la maintenance continue (évolution des domaines) et aux limites expressives des formalismes (contexte pragmatique, connaissances tacites, incertitude). Le Web sémantique, malgré ses promesses, n'a pas atteint l'adoption espérée.

**La gestion des données de référence (MDM)** vise à créer des sources uniques de vérité pour les entités métier critiques. Les approches centralisée, consolidée et hybride présentent chacune des compromis entre cohérence et agilité. Le taux d'échec élevé des projets MDM (50-80 %) résulte de la sous-estimation de la complexité politique, de la rigidité face à l'évolution métier et du problème de la qualité à la source.

**Le fossé sémantique** désigne l'écart entre les définitions formelles et la signification en contexte. La polysémie des termes métier, la dimension temporelle du sens et le contexte organisationnel créent des ambiguïtés que les schémas statiques ne peuvent pas capturer. Ce fossé est une réalité à gérer plutôt qu'un problème à éliminer.

**Les modèles canoniques** simplifient théoriquement les intégrations mais souffrent du problème du plus petit dénominateur commun, de l'inertie face aux changements et de l'incompatibilité avec les pratiques agiles. Le « modèle canonique zombie » illustre l'échec de cette approche dans les environnements dynamiques.

**Le besoin d'interopérabilité adaptative** émerge de ces limites. L'entreprise agentique requiert des mécanismes capables d'interpréter le sens en contexte, de gérer l'incertitude et de s'adapter aux évolutions. L'IA et les grands modèles de langage ouvrent la voie à cette transformation, exploitée aux chapitres suivants.

**Tableau de synthèse : Limites des approches sémantiques traditionnelles**

| **Approche** | **Promesse** | **Limite fondamentale** |
|--------------|--------------|------------------------|
| **Ontologies formelles** | Modélisation logique exhaustive du domaine | Coût d'acquisition, rigidité, limites expressives |
| **MDM** | Source unique de vérité pour les entités | Complexité politique, rigidité, qualité à la source |
| **Modèle canonique** | Format pivot réduisant la complexité | Inertie, plus petit dénominateur, dette technique |
| **Schémas statiques** | Définition a priori de la structure | Fossé sémantique, contexte non capturé |
| **Vocabulaires partagés** | Accord sur la terminologie | Polysémie, évolution des usages |

---

*Chapitre suivant : Chapitre I.11 — Intelligence Artificielle comme Moteur d'Interopérabilité Adaptative*
