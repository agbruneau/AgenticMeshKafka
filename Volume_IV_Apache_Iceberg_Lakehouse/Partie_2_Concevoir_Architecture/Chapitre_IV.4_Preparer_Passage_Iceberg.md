# Chapitre IV.4 — Préparer Votre Passage à Apache Iceberg

---

La Partie 1 de ce volume vous a présenté la valeur du Data Lakehouse, l'anatomie technique d'Apache Iceberg et une expérience pratique avec les outils de l'écosystème. Vous disposez désormais d'une compréhension solide des concepts fondamentaux et d'une intuition concrète sur le fonctionnement d'Iceberg. La Partie 2 marque une transition cruciale : nous passons de l'exploration technologique à la conception architecturale pour votre organisation.

Toute migration vers une nouvelle architecture de données représente un investissement significatif en temps, en ressources et en capital humain. Les échecs de projets de modernisation des données sont malheureusement fréquents, et leurs causes sont rarement techniques. Dans la grande majorité des cas, ces échecs résultent d'une préparation insuffisante : une compréhension incomplète de l'existant, des exigences mal définies, ou une communication déficiente avec les parties prenantes.

Ce chapitre établit les fondations méthodologiques pour votre passage à Apache Iceberg. Nous aborderons d'abord la réalisation d'un audit exhaustif de votre plateforme de données actuelle, puis nous illustrerons cette démarche à travers l'étude de cas détaillée de la Banque Hamerliwa, une institution financière fictive mais représentative des défis rencontrés par les organisations canadiennes. Ensuite, nous transformerons les constats de l'audit en exigences architecturales structurées, et finalement, nous explorerons les stratégies de communication et de présentation pour obtenir l'adhésion des décideurs.

L'approche préconisée privilégie la rigueur méthodologique sans sacrifier la pragmatisme. Un audit trop superficiel manquera des éléments critiques qui resurgiront pendant l'implémentation ; un audit excessivement détaillé paralysera le projet avant même qu'il ne commence. L'objectif est de trouver le juste équilibre entre exhaustivité et vélocité, entre analyse et action.

---

## IV.4.1 Réalisation de l'audit de votre plateforme

L'audit de la plateforme de données constitue la première étape indispensable de tout projet de migration vers Apache Iceberg. Cette démarche systématique vise à établir une cartographie complète de l'existant, à identifier les forces et les faiblesses de l'architecture actuelle, et à révéler les opportunités d'amélioration que le Lakehouse peut adresser. Sans cette compréhension approfondie, les décisions architecturales reposeront sur des hypothèses plutôt que sur des faits, augmentant considérablement les risques d'échec.

### Objectifs de l'audit

Un audit de plateforme de données bien conduit poursuit plusieurs objectifs complémentaires qui, ensemble, fournissent la base informationnelle nécessaire aux décisions architecturales.

Le premier objectif consiste à dresser l'inventaire complet des actifs de données. Cet inventaire couvre les sources de données (systèmes opérationnels, applications, flux externes), les zones de stockage (bases de données, entrepôts, lacs de données), les pipelines de transformation (processus ETL, traitements par lots, flux temps réel) et les outils de consommation (tableaux de bord, rapports, modèles analytiques). Cette cartographie révèle souvent des surprises : des sources de données oubliées, des pipelines redondants, ou des dépendances non documentées.

Le deuxième objectif vise à évaluer la santé technique de la plateforme. Cette évaluation examine les performances actuelles (temps de traitement, latence des requêtes, disponibilité), la qualité des données (complétude, exactitude, cohérence), la dette technique accumulée (code obsolète, dépendances désuètes, documentation manquante) et les risques opérationnels (points de défaillance unique, procédures de reprise non testées).

Le troisième objectif cherche à comprendre les besoins métier non satisfaits. Les limitations de la plateforme actuelle créent des frustrations chez les utilisateurs : rapports trop lents, données périmées, impossibilité de croiser certaines sources, coûts prohibitifs pour de nouvelles analyses. Ces besoins non comblés représentent la justification métier de la migration vers le Lakehouse.

Le quatrième objectif établit la base de référence pour mesurer le succès de la migration. Sans métriques initiales documentées, il sera impossible de démontrer objectivement les améliorations apportées par la nouvelle architecture.

### Les six dimensions de l'audit

Pour structurer la collecte d'information, nous proposons un cadre d'audit organisé en six dimensions complémentaires. Chaque dimension explore un aspect spécifique de la plateforme et contribue à la vision d'ensemble.

**Dimension 1 : Architecture et infrastructure**

Cette dimension cartographie les composants techniques de la plateforme actuelle. L'audit documente les systèmes de stockage utilisés (bases relationnelles, systèmes de fichiers distribués, stockage objet), les moteurs de traitement déployés (Spark, Hive, moteurs propriétaires), les outils d'orchestration (Airflow, Control-M, solutions maison) et l'infrastructure sous-jacente (sur site, infonuagique, hybride).

Questions clés à explorer :
- Quels sont les principaux systèmes de stockage et leurs volumes respectifs ?
- Quels moteurs de calcul sont utilisés pour quels types de traitements ?
- Comment l'infrastructure est-elle dimensionnée et quels sont les goulots d'étranglement ?
- Quelle est la répartition entre ressources sur site et infonuagiques ?
- Quels sont les accords de niveau de service (SLA) actuels et sont-ils respectés ?

**Dimension 2 : Données et modélisation**

Cette dimension examine la structure et l'organisation des données. L'audit inventorie les schémas de données, les modèles dimensionnels, les zones de données (brutes, raffinées, agrégées) et les relations entre les ensembles de données. Une attention particulière est portée aux formats de fichiers utilisés, aux stratégies de partitionnement en place et aux mécanismes de gestion des schémas.

Questions clés à explorer :
- Quels formats de fichiers sont utilisés (Parquet, ORC, Avro, CSV, JSON) ?
- Comment les données sont-elles partitionnées et cette stratégie est-elle optimale ?
- Existe-t-il un catalogue de métadonnées centralisé ?
- Comment l'évolution des schémas est-elle gérée aujourd'hui ?
- Quelle est la volumétrie par zone de données et son taux de croissance ?

**Dimension 3 : Flux et pipelines**

Cette dimension analyse les processus de mouvement et de transformation des données. L'audit recense les pipelines d'ingestion (par lots et en continu), les transformations ETL/ELT, les dépendances entre pipelines et les fenêtres de traitement. L'objectif est de comprendre comment les données circulent à travers la plateforme, du point d'origine jusqu'à la consommation.

Questions clés à explorer :
- Combien de pipelines sont en production et quelle est leur criticité ?
- Quels sont les temps de traitement moyens et leurs variations ?
- Existe-t-il des pipelines en temps réel et quelles technologies utilisent-ils ?
- Comment les erreurs et les reprises sont-elles gérées ?
- Quelle est la fréquence de rafraîchissement des différentes zones de données ?

**Dimension 4 : Consommation et utilisateurs**

Cette dimension évalue comment les données sont utilisées par les différentes communautés d'utilisateurs. L'audit identifie les outils de consommation (outils BI, notebooks, applications), les profils d'utilisateurs (analystes, scientifiques de données, développeurs, dirigeants), les cas d'usage principaux et les volumes de requêtes.

Questions clés à explorer :
- Quels outils BI sont utilisés et par combien d'utilisateurs ?
- Quelles sont les requêtes les plus fréquentes et les plus coûteuses ?
- Existe-t-il des besoins d'analyse ad hoc non satisfaits ?
- Comment les utilisateurs accèdent-ils aux données (SQL, API, fichiers) ?
- Quels sont les temps de réponse actuels et les attentes des utilisateurs ?

**Dimension 5 : Gouvernance et sécurité**

Cette dimension examine les mécanismes de contrôle et de protection des données. L'audit évalue les politiques d'accès, les processus de catalogage, la gestion du lignage, la conformité réglementaire et les pratiques de qualité des données.

Questions clés à explorer :
- Comment les accès aux données sont-ils contrôlés ?
- Existe-t-il un catalogue de données et est-il à jour ?
- Comment le lignage des données est-il tracé ?
- Quelles réglementations s'appliquent (Loi 25, RGPD, réglementations sectorielles) ?
- Quels processus de qualité des données sont en place ?

**Dimension 6 : Coûts et opérations**

Cette dimension quantifie les aspects économiques et opérationnels de la plateforme. L'audit documente les coûts directs (infrastructure, licences, personnel), les coûts indirects (temps perdu, opportunités manquées), la charge opérationnelle et les incidents récurrents.

Questions clés à explorer :
- Quel est le coût total de possession (TCO) de la plateforme actuelle ?
- Comment les coûts sont-ils répartis entre stockage, calcul et licences ?
- Quelle est la charge de travail de l'équipe de données ?
- Quels sont les incidents les plus fréquents et leur impact ?
- Existe-t-il des inefficacités connues (données dupliquées, traitements redondants) ?

### Méthodes de collecte d'information

La collecte d'information pour l'audit combine plusieurs méthodes complémentaires, chacune apportant une perspective différente sur la plateforme.

**Analyse documentaire** : L'examen de la documentation existante (diagrammes d'architecture, spécifications techniques, manuels opérationnels) fournit une première vue de la plateforme. Toutefois, la documentation est souvent incomplète ou obsolète, ce qui nécessite une validation par d'autres méthodes. Les documents à collecter incluent les schémas de données, les dictionnaires de données, les procédures opérationnelles, les rapports d'incidents passés, les contrats avec les fournisseurs et les évaluations de sécurité antérieures.

**Entretiens structurés** : Les conversations avec les différentes parties prenantes révèlent des informations que les documents ne capturent pas : les frustrations quotidiennes, les solutions de contournement, les besoins non exprimés. Les entretiens doivent couvrir les équipes techniques (administrateurs, développeurs de pipelines, analystes de données) et les utilisateurs métier (analystes financiers, équipes marketing, dirigeants). Un guide d'entretien standardisé assure la cohérence des questions tout en laissant place aux explorations spontanées. Prévoyez 60 à 90 minutes par entretien et documentez les réponses immédiatement pour éviter les pertes d'information.

**Analyse technique** : L'examen direct des systèmes fournit des données objectives sur la performance, les volumes et les configurations. Cette analyse inclut l'inspection des métadonnées, le profilage des requêtes, l'analyse des journaux système et le recensement des composants installés. Pour les systèmes de données, portez une attention particulière aux statistiques de tables (cardinalité, distribution des valeurs, taux de nullité), aux plans d'exécution des requêtes fréquentes, aux métriques de latence et de débit, ainsi qu'aux alertes et incidents récents dans les outils de supervision.

**Ateliers collaboratifs** : Les sessions de travail en groupe permettent de valider les constats, de résoudre les ambiguïtés et de construire une compréhension partagée de la situation actuelle. Ces ateliers rassemblent des représentants des différentes équipes concernées. Structurez ces ateliers autour de questions ouvertes comme « Quels sont les trois problèmes de données qui vous font perdre le plus de temps ? » ou « Si vous pouviez changer une chose dans la plateforme actuelle, ce serait quoi ? ». Utilisez des techniques de facilitation comme le vote par points pour prioriser les enjeux identifiés.

**Observation directe** : Passez du temps avec les équipes pendant leurs activités quotidiennes. Observez comment les analystes accèdent aux données, quelles manipulations manuelles ils effectuent, combien de temps ils attendent les résultats de leurs requêtes. Cette méthode révèle des inefficacités que les personnes concernées ne mentionnent plus parce qu'elles les considèrent comme normales.

> **Étude de cas : Banque TD — L'audit qui a révélé les angles morts**  
> *Secteur* : Services financiers  
> *Défi* : Un audit initial basé uniquement sur la documentation existante avait conclu que la plateforme de données était en bon état. Les projets de modernisation basés sur ces conclusions ont échoué.  
> *Solution* : Un second audit incorporant des entretiens approfondis et une observation directe des équipes a révélé que 40 % des traitements critiques reposaient sur des scripts non documentés maintenus par des individus spécifiques.  
> *Résultats* : La cartographie complète des dépendances cachées a permis de planifier une migration réaliste, complétée avec 3 mois d'avance sur le calendrier révisé.

### Livrables de l'audit

L'audit produit plusieurs livrables qui serviront de référence tout au long du projet de migration. La qualité et la complétude de ces livrables déterminent directement la qualité des décisions architecturales qui en découleront.

Le **rapport d'audit** synthétise les constats organisés par dimension, accompagnés d'une évaluation de la maturité de la plateforme et d'une identification des risques et opportunités. Ce document de 30 à 50 pages constitue la référence principale pour les décideurs. Il doit être rédigé dans un langage accessible aux non-techniciens tout en conservant la rigueur nécessaire pour les experts. Une version exécutive de 5 pages accompagne le rapport complet pour les audiences pressées.

La **cartographie des actifs** présente visuellement l'architecture actuelle : systèmes, flux de données, dépendances et zones de responsabilité. Cette cartographie utilise idéalement une notation standardisée (comme ArchiMate ou C4) pour faciliter la communication et la maintenance. Elle doit pouvoir être mise à jour au fil du projet pour refléter l'évolution de l'architecture.

L'**inventaire des données** liste les ensembles de données avec leurs caractéristiques : volume, format, propriétaire, niveau de sensibilité, fréquence d'utilisation, qualité observée. Pour les grandes organisations, cet inventaire peut contenir des milliers d'entrées. Une priorisation basée sur la criticité métier et la complexité technique aide à focaliser les efforts de migration.

Le **catalogue des pipelines** documente les processus de données : sources, transformations, destinations, planification, propriétaires, SLA associés. Ce catalogue identifie également les dépendances entre pipelines, information critique pour planifier l'ordre de migration.

Les **métriques de référence** établissent les indicateurs de performance actuels qui serviront de base de comparaison après la migration. Ces métriques couvrent les dimensions de performance (temps de traitement, latence), de qualité (taux d'erreurs, complétude), de coût (par téraoctet, par requête) et de satisfaction (tickets de support, temps de résolution).

Le **registre des risques** identifie les risques découverts pendant l'audit avec leur probabilité, leur impact et les stratégies de mitigation proposées. Ce registre sera enrichi tout au long du projet.

> **Performance**  
> L'audit lui-même peut révéler des opportunités d'amélioration rapide (« quick wins ») indépendantes de la migration Iceberg. Par exemple, l'identification de requêtes mal optimisées, de partitionnements inadéquats ou de données dupliquées peut justifier des corrections immédiates qui amélioreront la situation sans attendre le déploiement du Lakehouse. Chez Hamerliwa, l'audit a identifié 8 quick wins potentiels représentant une économie annuelle de 180 K$ et une amélioration de 25 % des temps de traitement nocturnes.

### Facteurs de succès de l'audit

Plusieurs facteurs contribuent à la réussite de l'audit et à l'utilité de ses résultats. L'expérience des projets similaires permet d'identifier les conditions qui maximisent les chances de succès.

**Engagement de la direction** : L'audit nécessite l'accès à des ressources (temps des équipes, systèmes de production) qui requiert un soutien explicite de la direction. Sans ce soutien, les équipes percevront l'audit comme une distraction de leurs responsabilités quotidiennes. Le sponsor exécutif doit communiquer clairement l'importance de l'audit et autoriser explicitement les équipes à y consacrer le temps nécessaire.

**Périmètre réaliste** : Un audit trop ambitieux s'enlisera dans les détails ; un audit trop restreint manquera des éléments critiques. La définition du périmètre doit équilibrer exhaustivité et pragmatisme en fonction des ressources disponibles. En règle générale, prévoyez 2 à 3 jours d'effort par dimension pour une organisation de taille moyenne comme Hamerliwa.

**Équipe pluridisciplinaire** : L'audit bénéficie de la diversité des perspectives. Une équipe composée d'architectes, d'ingénieurs de données, d'analystes métier et de représentants des opérations couvrira mieux l'ensemble des dimensions. L'inclusion d'un regard externe (consultant, auditeur) apporte une objectivité que les équipes internes, trop proches des systèmes, peuvent avoir du mal à maintenir.

**Communication transparente** : L'audit n'est pas une mission d'inspection visant à identifier des coupables. Cette distinction doit être clairement communiquée pour obtenir la coopération des équipes et des informations franches sur les problèmes existants. Les participants doivent comprendre que l'objectif est d'améliorer la situation, pas de pointer du doigt les responsables des problèmes passés.

**Calendrier réaliste** : Un audit bâclé en deux semaines manquera des éléments essentiels. À l'inverse, un audit qui s'étire sur six mois perdra son élan et ses conclusions risquent d'être déjà obsolètes. Pour une organisation de la taille d'Hamerliwa, un calendrier de 6 à 8 semaines représente un bon équilibre.

**Accès aux données réelles** : L'audit doit pouvoir examiner les systèmes de production, pas seulement la documentation. Les métriques de performance, les journaux d'erreurs, les statistiques d'utilisation fournissent une image objective que la documentation seule ne peut pas donner. Prévoyez les autorisations nécessaires dès le début du projet.

**Documentation au fil de l'eau** : Documenter les constats immédiatement, pendant que le contexte est frais, améliore considérablement la qualité du rapport final. Attendre la fin de l'audit pour tout documenter entraîne des pertes d'information et des incohérences.

---

## IV.4.2 L'audit de la Banque Hamerliwa en action

Pour illustrer concrètement la démarche d'audit, nous suivrons le cas de la Banque Hamerliwa, une institution financière canadienne de taille moyenne. Bien que fictive, cette banque représente un archétype réaliste des organisations qui envisagent la migration vers un Lakehouse Apache Iceberg. Son histoire, ses défis et ses aspirations reflètent les situations rencontrées dans de nombreuses entreprises canadiennes du secteur financier et au-delà.

### Profil de la Banque Hamerliwa

La Banque Hamerliwa est une banque régionale établie depuis 1952, dont le siège social se situe à Québec. Avec 2 500 employés répartis dans 85 succursales au Québec et en Ontario, elle gère 45 milliards de dollars d'actifs et sert 850 000 clients particuliers et 35 000 entreprises. Son nom, signifiant « banque des gens » en langue algonquine, reflète sa mission historique d'accessibilité financière.

Au cours de la dernière décennie, Hamerliwa a connu plusieurs transformations technologiques successives. En 2015, elle a migré son système bancaire central vers une plateforme moderne. En 2018, elle a lancé une application mobile qui a rapidement gagné en popularité. En 2020, face à la pandémie, elle a accéléré sa transformation numérique. Chacune de ces initiatives a généré de nouveaux besoins en données, aboutissant à une architecture complexe et parfois incohérente.

### Contexte de l'audit

En septembre 2025, le Conseil d'administration d'Hamerliwa approuve un programme de modernisation des données sur trois ans, doté d'un budget de 28 millions de dollars. Ce programme répond à plusieurs pressions convergentes :

**Pression réglementaire** : L'entrée en vigueur complète de la Loi 25 au Québec et les exigences accrues du BSIF (Bureau du surintendant des institutions financières) en matière de gestion des données imposent des améliorations dans la gouvernance et la traçabilité.

**Pression concurrentielle** : Les néobanques et les fintechs offrent des expériences client personnalisées basées sur l'analyse avancée des données. Hamerliwa doit développer des capacités similaires pour maintenir sa position sur le marché.

**Pression opérationnelle** : La plateforme de données actuelle montre des signes de fatigue. Les rapports réglementaires prennent de plus en plus de temps à produire, les équipes d'analyse passent davantage de temps à préparer les données qu'à les analyser, et les coûts d'infrastructure augmentent plus vite que les volumes de données.

L'équipe d'audit est constituée en octobre 2025, sous la direction de Maryse Chen, architecte de données senior. Elle est composée de deux ingénieurs de données, d'une analyste métier, d'un spécialiste de la gouvernance et d'un consultant externe spécialisé dans les architectures Lakehouse. L'audit doit être complété en huit semaines.

### Constats de l'audit — Dimension Architecture

L'analyse de l'architecture révèle une plateforme fragmentée, résultat de décisions prises à différentes époques pour répondre à des besoins spécifiques.

**Systèmes de stockage multiples** : Hamerliwa exploite simultanément un entrepôt de données Oracle Exadata acquis en 2014, un cluster Hadoop Cloudera déployé en 2017 pour le « big data », une instance Snowflake adoptée en 2021 pour les analyses marketing, et plusieurs bases de données Azure SQL pour des applications départementales. Cette multiplicité génère des silos de données, des duplications et des incohérences.

**Infrastructure hybride non optimisée** : L'Exadata reste dans le centre de données principal de Québec, le cluster Hadoop occupe deux armoires vieillissantes, tandis que Snowflake et Azure SQL fonctionnent dans l'infonuagique. Cette répartition résulte davantage de l'historique que d'une stratégie délibérée. Les coûts de transfert de données entre environnements sont significatifs.

**Moteurs de calcul disparates** : Les traitements s'exécutent sur Informatica PowerCenter (ETL traditionnel), Apache Spark sur Hadoop (traitements big data), Snowflake (analyses marketing) et des scripts Python ad hoc. Chaque moteur a ses experts, ses pratiques et sa dette technique.

| Composant | Année | Volumes | Coût annuel |
|-----------|-------|---------|-------------|
| Oracle Exadata | 2014 | 15 To | 1,2 M$ |
| Cloudera Hadoop | 2017 | 180 To | 850 K$ |
| Snowflake | 2021 | 8 To | 420 K$ |
| Azure SQL (divers) | 2019-2023 | 2 To | 180 K$ |

### Constats de l'audit — Dimension Données

L'examen des données révèle des pratiques de gestion hétérogènes et une dette technique substantielle.

**Formats multiples sans standard** : Les données sont stockées en format CSV (systèmes hérités), Parquet (traitements Spark récents), ORC (premiers projets Hadoop), tables relationnelles (Exadata, SQL), et formats propriétaires Snowflake. Cette diversité complique les traitements inter-systèmes.

**Partitionnement incohérent** : Sur Hadoop, certaines tables sont partitionnées par date, d'autres par région, d'autres encore ne sont pas partitionnées du tout. Les choix de partitionnement ne correspondent pas toujours aux patterns d'accès, entraînant des analyses de données complètes (« full scans ») évitables.

**Évolution de schéma problématique** : Les modifications de schéma sur Hadoop nécessitent souvent une réécriture complète des données historiques. Plusieurs tables contiennent des colonnes obsolètes conservées par prudence, tandis que d'autres manquent de colonnes ajoutées récemment dans les systèmes sources.

**Catalogue incomplet** : Un outil de catalogage a été déployé en 2022, mais seulement 35 % des actifs de données y sont documentés. Les métadonnées sont souvent périmées, et le lignage n'est tracé que pour les pipelines les plus critiques.

### Constats de l'audit — Dimension Flux

L'analyse des pipelines met en lumière une complexité accumulée au fil des ans.

**Prolifération des pipelines** : L'inventaire recense 847 pipelines en production, dont 312 considérés comme critiques. Cependant, 23 % des pipelines n'ont pas été modifiés depuis plus de trois ans et leur pertinence est incertaine. Certains pipelines font double emploi, produisant des résultats similaires par des chemins différents.

**Fenêtres de traitement saturées** : Les traitements nocturnes par lots s'étendent désormais de 22h à 7h, ne laissant qu'une marge de 3 heures avant l'ouverture des marchés. Tout retard significatif impacte la disponibilité des rapports matinaux.

**Absence de temps réel** : Malgré l'acquisition d'une licence Kafka en 2023, les flux temps réel restent limités à quelques cas d'usage pilotes. La majorité des données analytiques ont un délai de fraîcheur de T+1 (disponibles le lendemain).

**Gestion des erreurs artisanale** : Les reprises après erreur sont largement manuelles. L'équipe des opérations consacre en moyenne 15 heures par semaine à la correction de pipelines échoués.

> **Étude de cas : Banque Hamerliwa — Le syndrome du pipeline orphelin**  
> *Secteur* : Services financiers  
> *Défi* : Au cours de l'audit, l'équipe découvre 47 pipelines dont les propriétaires ont quitté l'organisation sans transfert de connaissances. Ces pipelines continuent de s'exécuter, consommant des ressources, mais personne ne peut confirmer si leurs résultats sont encore utilisés.  
> *Solution* : Un processus de « quarantaine » est mis en place : les pipelines suspects sont désactivés temporairement et, en l'absence de réclamation après 30 jours, ils sont archivés.  
> *Résultats* : 31 pipelines sont définitivement retirés, libérant 8 % de la capacité de calcul Hadoop et simplifiant la planification nocturne.

### Constats de l'audit — Dimension Consommation

L'évaluation de l'utilisation des données révèle un écart entre les capacités offertes et les besoins réels. Cette dimension est particulièrement importante car elle détermine la valeur perçue de la plateforme de données par les utilisateurs métier.

**Outils BI multiples** : Hamerliwa utilise simultanément Power BI (équipes centrales, 180 utilisateurs), Tableau (équipes marketing, 45 utilisateurs), QlikView (hérité, encore utilisé par la comptabilité, 30 utilisateurs) et des rapports Excel personnalisés (nombre indéterminé mais estimé à plusieurs centaines). Cette diversité complique la gouvernance et génère des résultats parfois contradictoires pour les mêmes indicateurs. L'audit identifie 12 cas où le même KPI est calculé différemment selon l'outil utilisé.

**Performance insuffisante** : Les requêtes analytiques complexes sur l'Exadata peuvent prendre plusieurs heures. L'audit mesure un temps de réponse médian de 47 minutes pour les requêtes impliquant des jointures entre plus de 5 tables. Les analystes contournent cette limitation en extrayant des échantillons de données vers des fichiers locaux, créant des risques de sécurité et de cohérence. On estime que 2,3 To de données « grises » existent sur les postes de travail et les espaces de partage non gouvernés.

**Besoins non satisfaits** : Les entretiens avec les équipes métier révèlent des frustrations récurrentes documentées dans le tableau suivant :

| Équipe | Besoin exprimé | Impact estimé |
|--------|----------------|---------------|
| Gestion des risques | Analyses temps réel pour détection de fraude | 2,1 M$ de pertes évitables/an |
| Marketing | Segmentation avancée avec données comportementales | 15 % d'amélioration potentielle des campagnes |
| Finance | Clôtures accélérées avec réconciliation automatique | 5 jours/mois récupérés |
| Direction | Tableaux de bord consolidés multi-domaines | Décisions plus rapides et mieux informées |
| Conformité | Rapports réglementaires automatisés | Réduction des erreurs de 40 % |

**Compétences limitées** : Seuls 12 employés maîtrisent SQL avancé, créant un goulot d'étranglement pour les demandes d'analyse. Les scientifiques de données (équipe de 8 personnes) passent 60 % de leur temps à préparer les données plutôt qu'à construire des modèles. Le ratio préparation/analyse devrait idéalement être inversé. Cette situation freine l'adoption de l'apprentissage automatique malgré les investissements réalisés dans les outils.

**Libre-service limité** : Les tentatives de déployer des outils de libre-service analytique ont échoué faute de données suffisamment préparées et documentées. Les utilisateurs métier abandonnent rapidement lorsqu'ils ne trouvent pas les données dont ils ont besoin ou ne comprennent pas leur signification.

### Constats de l'audit — Dimension Gouvernance

L'examen de la gouvernance expose des lacunes qui deviennent critiques face aux exigences réglementaires croissantes.

**Contrôle d'accès fragmenté** : Chaque système possède son propre mécanisme de gestion des accès. Un employé peut avoir accès à des données sensibles dans Snowflake sans autorisation équivalente dans l'Exadata, créant des incohérences difficiles à auditer.

**Lignage incomplet** : Le lignage des données n'est documenté que pour 40 % des pipelines critiques. En cas de question réglementaire sur l'origine d'une donnée, la reconstitution manuelle peut prendre plusieurs jours.

**Qualité des données variable** : Des règles de qualité existent pour les données réglementaires, mais la majorité des données analytiques ne font l'objet d'aucun contrôle systématique. Les utilisateurs découvrent les problèmes de qualité lors de l'analyse, générant perte de temps et méfiance.

**Conformité Loi 25** : L'évaluation révèle des écarts par rapport aux exigences de la Loi 25, notamment en matière de consentement, de conservation et de droit à l'effacement. Les données personnelles sont dispersées dans plusieurs systèmes, compliquant la réponse aux demandes d'accès.

### Constats de l'audit — Dimension Coûts

L'analyse financière quantifie le fardeau économique de l'architecture actuelle.

**Coût total de possession** : Les coûts directs de la plateforme de données s'élèvent à 4,8 millions de dollars annuellement, répartis entre infrastructure (2,65 M$), licences logicielles (1,35 M$) et services professionnels (0,8 M$). Ces coûts ont augmenté de 18 % sur les deux dernières années, principalement à cause de Snowflake et des transferts de données inter-environnements.

**Coûts cachés** : Les coûts indirects sont plus difficiles à quantifier mais substantiels. L'équipe estime à 1,2 million de dollars la valeur du temps perdu par les analystes en préparation de données. Les opportunités manquées faute de capacités analytiques adéquates sont incalculables.

**Structure de coûts rigide** : 78 % des coûts sont fixes (contrats pluriannuels, infrastructure sur site). Cette rigidité limite la capacité d'adaptation aux variations de la demande.

| Catégorie | Coût annuel | % du total | Tendance |
|-----------|-------------|------------|----------|
| Infrastructure sur site | 1,45 M$ | 30 % | Stable |
| Infonuagique (Snowflake, Azure) | 1,20 M$ | 25 % | +25 %/an |
| Licences (Oracle, Informatica, BI) | 1,35 M$ | 28 % | +5 %/an |
| Personnel dédié | 0,55 M$ | 12 % | +10 %/an |
| Services professionnels | 0,25 M$ | 5 % | Variable |

### Synthèse de l'audit Hamerliwa

L'audit de la Banque Hamerliwa dresse le portrait d'une plateforme de données fonctionnelle mais fragile, dont la complexité et les coûts augmentent plus vite que la valeur générée. Les constats principaux se résument ainsi :

**Forces identifiées** :
- Équipe technique compétente et motivée
- Données réglementaires de bonne qualité
- Infrastructure Exadata stable pour les traitements critiques
- Expérience initiale avec Kafka et technologies modernes

**Faiblesses majeures** :
- Architecture fragmentée en silos technologiques
- Absence de catalogue de données centralisé et complet
- Incapacité à répondre aux besoins temps réel
- Coûts croissants sans amélioration proportionnelle des capacités
- Gouvernance inadaptée aux exigences réglementaires actuelles

**Opportunités Lakehouse** :
- Consolidation des données sur une plateforme unifiée
- Réduction des coûts par élimination des duplications
- Capacités analytiques avancées (temps réel, ML)
- Gouvernance intégrée avec contrôle d'accès fin
- Évolution de schéma native sans reconstruction

**Risques à mitiger** :
- Résistance au changement des équipes habituées aux outils existants
- Complexité de migration des 847 pipelines
- Dépendance aux compétences externes pendant la transition
- Continuité des services pendant la migration

---

## IV.4.3 De l'audit aux exigences

Les constats de l'audit constituent une mine d'informations, mais ils doivent être transformés en exigences architecturales exploitables pour guider la conception du Lakehouse. Cette traduction représente une étape critique : des exigences mal formulées mèneront à une architecture inadaptée, tandis que des exigences trop vagues laisseront trop de latitude aux interprétations divergentes.

### Structure des exigences

Nous organisons les exigences en quatre catégories complémentaires qui, ensemble, définissent le cadre de la future architecture.

**Exigences fonctionnelles** : Ce que la plateforme doit permettre de faire. Elles décrivent les capacités requises en termes de stockage, traitement, consommation et gouvernance.

**Exigences non fonctionnelles** : Comment la plateforme doit se comporter. Elles spécifient les niveaux de performance, disponibilité, sécurité et évolutivité attendus.

**Exigences de migration** : Comment passer de l'existant à la cible. Elles définissent les contraintes de continuité, les priorités de migration et les critères d'acceptation.

**Exigences de gouvernance** : Comment la plateforme sera encadrée. Elles établissent les politiques de contrôle, les responsabilités et les processus de gestion.

### Exigences fonctionnelles pour Hamerliwa

À partir des constats de l'audit, l'équipe formule les exigences fonctionnelles suivantes pour le futur Lakehouse.

**EF-01 : Stockage unifié des données**  
La plateforme doit permettre le stockage de toutes les données analytiques dans un format ouvert et interopérable. Les données actuellement dispersées entre Exadata (analytique), Hadoop, Snowflake et Azure SQL doivent être consolidables sur une infrastructure commune.

*Justification audit* : Les silos de données actuels génèrent des duplications (estimées à 40 % des volumes), des incohérences et des coûts de transfert élevés.

**EF-02 : Évolution de schéma sans interruption**  
La plateforme doit supporter l'ajout, la suppression et la modification de colonnes sans nécessiter la réécriture des données historiques ni l'interruption des traitements.

*Justification audit* : Les modifications de schéma actuelles sur Hadoop mobilisent en moyenne 3 jours-personnes et créent des fenêtres d'indisponibilité.

**EF-03 : Capacité de requêtes historiques (Time Travel)**  
La plateforme doit permettre d'interroger l'état des données à n'importe quel moment passé, avec une rétention configurable par table.

*Justification audit* : Les demandes réglementaires de reconstitution historique nécessitent actuellement des restaurations de sauvegardes coûteuses et lentes.

**EF-04 : Ingestion temps réel**  
La plateforme doit supporter l'ingestion de données en continu avec un délai maximal de 5 minutes entre l'événement source et la disponibilité pour l'analyse.

*Justification audit* : Les besoins de détection de fraude et de personnalisation en temps réel ne peuvent être satisfaits avec l'architecture T+1 actuelle.

**EF-05 : Accès SQL unifié**  
La plateforme doit exposer toutes les données via une interface SQL standard, accessible depuis les outils BI existants (Power BI, Tableau) sans développement spécifique.

*Justification audit* : La multiplication des interfaces d'accès (JDBC Oracle, connecteurs Snowflake, scripts Python) complique l'adoption et génère des erreurs.

**EF-06 : Catalogue de métadonnées centralisé**  
La plateforme doit inclure un catalogue automatiquement alimenté répertoriant tous les actifs de données avec leurs métadonnées techniques et métier.

*Justification audit* : Le catalogue actuel ne couvre que 35 % des actifs et ses informations sont souvent périmées.

**EF-07 : Lignage de données automatique**  
La plateforme doit capturer automatiquement le lignage des données, de l'ingestion à la consommation, traçant les transformations appliquées.

*Justification audit* : La reconstitution manuelle du lignage pour les audits réglementaires consomme plusieurs jours-personnes par demande.

### Exigences non fonctionnelles pour Hamerliwa

Les exigences non fonctionnelles définissent les niveaux de service attendus de la plateforme. Ces exigences sont souvent négligées au profit des fonctionnalités, mais elles déterminent l'expérience réelle des utilisateurs et la viabilité opérationnelle de la solution.

**ENF-01 : Performance des requêtes analytiques**  
Les requêtes analytiques standards (agrégations sur 30 jours de transactions) doivent s'exécuter en moins de 30 secondes pour les utilisateurs BI. Les requêtes ad hoc complexes doivent s'exécuter en moins de 5 minutes.

*Métrique actuelle* : 45 minutes en moyenne pour les requêtes complexes sur Exadata.

*Critères de mesure* : Percentile 95 des temps de réponse mesuré sur une semaine glissante pour un échantillon représentatif de requêtes.

**ENF-02 : Disponibilité de la plateforme**  
La plateforme doit garantir une disponibilité de 99,9 % pendant les heures ouvrables (6h-22h, heure de l'Est) et de 99,5 % hors heures ouvrables.

*Métrique actuelle* : 99,2 % de disponibilité moyenne, avec des incidents récurrents lors des traitements nocturnes.

*Implications* : 99,9 % correspond à 8,76 heures d'indisponibilité maximale par an pendant les heures ouvrables. Cette cible nécessite une architecture hautement disponible avec basculement automatique.

**ENF-03 : Scalabilité du stockage**  
La plateforme doit supporter une croissance de 50 % des volumes de données annuellement sans dégradation de performance ni intervention architecturale majeure.

*Projection* : Les volumes actuels de 205 To atteindront 500 To d'ici 2028 selon les tendances observées.

*Contrainte technique* : Le stockage objet infonuagique répond nativement à cette exigence, contrairement aux systèmes sur site qui nécessitent des acquisitions matérielles planifiées.

**ENF-04 : Élasticité du calcul**  
La capacité de calcul doit pouvoir augmenter de 300 % en moins de 15 minutes pour absorber les pics de charge (clôtures mensuelles, campagnes marketing).

*Contrainte actuelle* : L'infrastructure fixe actuelle ne permet aucune élasticité.

*Justification* : Les analyses de clôture trimestrielle génèrent des pics de charge prévisibles mais intenses. Sans élasticité, ces périodes créent des goulots d'étranglement et des retards dans les livrables.

**ENF-05 : Temps de récupération après sinistre**  
En cas de sinistre majeur, la plateforme doit être récupérable avec un RPO (Recovery Point Objective) de 1 heure et un RTO (Recovery Time Objective) de 4 heures.

*État actuel* : RPO de 24 heures, RTO non testé estimé à 48 heures.

*Implications architecturales* : Cette exigence nécessite une réplication continue des métadonnées critiques et des snapshots fréquents des tables Iceberg. La stratégie de récupération doit être documentée et testée trimestriellement.

**ENF-06 : Coût cible**  
Le coût total de possession de la nouvelle plateforme ne doit pas dépasser le coût actuel (4,8 M$/an) la première année, et doit diminuer de 15 % d'ici la troisième année malgré l'augmentation des volumes.

*Décomposition cible année 3* :
- Stockage objet : 0,8 M$ (contre 1,2 M$ actuel pour Snowflake + transferts)
- Calcul élastique : 1,2 M$ (variable selon usage)
- Licences (Dremio, outils) : 0,9 M$ (contre 1,35 M$ actuel)
- Personnel et formation : 0,7 M$
- Services professionnels : 0,5 M$
- **Total année 3** : 4,1 M$ (réduction de 15 %)

**ENF-07 : Latence d'ingestion temps réel**  
Pour les flux désignés comme temps réel, le délai entre l'événement source et la disponibilité de la donnée pour interrogation ne doit pas dépasser 5 minutes dans 99 % des cas.

*Cas d'usage* : Détection de fraude, alertes de seuil, tableaux de bord opérationnels.

*Contrainte technique* : Cette exigence oriente vers une architecture Streaming Lakehouse avec ingestion Kafka directe dans les tables Iceberg, plutôt qu'un modèle micro-batch traditionnel.

**ENF-08 : Concurrence utilisateurs**  
La plateforme doit supporter simultanément 200 utilisateurs BI actifs et 50 requêtes analytiques lourdes sans dégradation significative (< 20 % d'augmentation du temps de réponse).

*État actuel* : Au-delà de 80 utilisateurs simultanés, les performances de l'Exadata se dégradent notablement.

### Exigences de migration pour Hamerliwa

Les exigences de migration encadrent la transition de l'architecture actuelle vers le Lakehouse cible.

**EM-01 : Continuité des services critiques**  
Les services réglementaires (rapports BSIF, déclarations fiscales) doivent rester opérationnels sans interruption tout au long de la migration. Une période de fonctionnement parallèle est requise avant toute décommission.

**EM-02 : Migration progressive par domaine**  
La migration doit procéder par domaine de données (risque, finance, marketing, opérations) plutôt que par technologie, permettant une validation métier à chaque étape.

**EM-03 : Réversibilité**  
Chaque phase de migration doit préserver la capacité de retour à l'état précédent pendant 90 jours, avec des procédures documentées et testées.

**EM-04 : Validation des données migrées**  
Des mécanismes de réconciliation automatique doivent confirmer l'intégrité des données migrées. Une tolérance de 0,01 % est acceptée pour les écarts numériques dus aux conversions de formats.

**EM-05 : Transfert de connaissances**  
La migration doit inclure un programme de formation pour les équipes internes, avec l'objectif que 80 % des opérations courantes soient gérées en interne à la fin du projet.

> **Migration**  
> *De* : Architecture fragmentée (Exadata + Hadoop + Snowflake + Azure SQL)  
> *Vers* : Lakehouse unifié Apache Iceberg  
> *Stratégie* : Migration par domaine avec période de fonctionnement parallèle. Priorité 1 : données marketing (moins critiques, bon terrain d'apprentissage). Priorité 2 : données opérationnelles. Priorité 3 : données réglementaires (après validation approfondie). Décommission des anciens systèmes uniquement après 6 mois de stabilité sur la nouvelle plateforme.

### Exigences de gouvernance pour Hamerliwa

Les exigences de gouvernance établissent le cadre de contrôle de la future plateforme.

**EG-01 : Contrôle d'accès fin**  
La plateforme doit supporter des politiques d'accès au niveau des lignes et des colonnes, permettant de restreindre l'accès aux données sensibles selon le profil de l'utilisateur.

*Contrainte réglementaire* : Loi 25, exigences BSIF sur la séparation des fonctions.

**EG-02 : Chiffrement des données**  
Toutes les données doivent être chiffrées au repos (AES-256) et en transit (TLS 1.3). Les clés de chiffrement doivent être gérées par un service dédié avec rotation automatique.

**EG-03 : Conservation et purge**  
La plateforme doit supporter des politiques de conservation configurables par classification de données, avec purge automatique à l'expiration et preuve de destruction.

**EG-04 : Audit des accès**  
Tous les accès aux données sensibles doivent être journalisés avec horodatage, identité de l'utilisateur, action effectuée et données consultées. Ces journaux doivent être conservés 7 ans.

**EG-05 : Propriété des données**  
Chaque ensemble de données doit avoir un propriétaire métier identifié, responsable de la qualité, des accès et du cycle de vie.

### Matrice de traçabilité

Pour assurer que chaque exigence répond à un constat de l'audit et que chaque constat critique est adressé, une matrice de traçabilité est établie :

| Constat audit | Exigences associées | Priorité |
|---------------|---------------------|----------|
| Silos de données multiples | EF-01, EF-05 | Critique |
| Évolution de schéma difficile | EF-02 | Haute |
| Pas de capacité temps réel | EF-04 | Haute |
| Catalogue incomplet | EF-06 | Moyenne |
| Lignage non tracé | EF-07 | Haute |
| Requêtes lentes | ENF-01 | Critique |
| Coûts croissants | ENF-06 | Haute |
| Gouvernance fragmentée | EG-01 à EG-05 | Critique |

Cette matrice sera utilisée tout au long du projet pour vérifier que l'architecture conçue répond effectivement aux besoins identifiés.

---

## IV.4.4 Plan architectural et présentation itinérante

Les exigences définies à la section précédente constituent le cahier des charges de la future architecture. Avant d'entamer la conception détaillée — qui sera l'objet des chapitres IV.5 à IV.9 — il est essentiel d'établir un plan architectural de haut niveau et de le communiquer efficacement aux parties prenantes. Cette section traite de ces deux aspects complémentaires : la vision architecturale et la stratégie de communication.

### Vision architecturale de haut niveau

Le plan architectural traduit les exigences en orientations technologiques et organisationnelles. Pour Hamerliwa, la vision architecturale s'articule autour de cinq piliers.

**Pilier 1 : Apache Iceberg comme format de table universel**

Apache Iceberg sera adopté comme format de table unique pour toutes les données analytiques. Ce choix répond aux exigences EF-01 (stockage unifié), EF-02 (évolution de schéma), EF-03 (Time Travel) et apporte plusieurs bénéfices :

- Format ouvert évitant le verrouillage fournisseur
- Support natif de l'évolution de schéma et du partitionnement masqué
- Interopérabilité avec tous les moteurs de calcul majeurs
- Snapshots pour le Time Travel et la récupération après erreur

**Pilier 2 : Stockage objet infonuagique**

Les données seront stockées sur un service de stockage objet infonuagique (à déterminer : Azure Blob Storage, Amazon S3, ou Google Cloud Storage selon l'analyse du chapitre IV.5). Cette orientation répond aux exigences ENF-03 (scalabilité), ENF-04 (élasticité) et ENF-06 (coûts) :

- Scalabilité pratiquement illimitée
- Modèle de coûts à l'usage aligné sur la consommation réelle
- Durabilité et disponibilité natives
- Séparation stockage/calcul pour optimisation des coûts

**Pilier 3 : Catalogue REST unifié**

Un catalogue conforme à la spécification Apache Iceberg REST Catalog centralisera la gestion des métadonnées. Cette orientation répond aux exigences EF-06 (catalogue centralisé) et EF-07 (lignage) :

- Point d'accès unique pour tous les moteurs de calcul
- Versionnement des métadonnées (capacité Git-like)
- Intégration native avec les outils de gouvernance
- Support du multi-tenancy pour les environnements isolés

**Pilier 4 : Couche de fédération Dremio**

Dremio sera déployé comme couche de fédération et d'accélération, répondant aux exigences EF-05 (accès SQL unifié) et ENF-01 (performance) :

- Interface SQL unifiée pour tous les consommateurs
- Réflexions de données pour l'accélération des requêtes
- Connexion native aux outils BI existants
- Contrôle d'accès fin intégré

**Pilier 5 : Intégration Kafka pour le temps réel**

L'infrastructure Kafka existante sera étendue pour alimenter le Lakehouse en temps réel, répondant à l'exigence EF-04 (ingestion temps réel). Cette intégration exploitera les connecteurs Kafka Connect pour Iceberg, permettant l'écriture directe des flux événementiels dans les tables Iceberg. Le Volume III de cette monographie détaille les patrons d'architecture Kafka applicables.

### Diagramme d'architecture cible

L'architecture cible d'Hamerliwa se structure en couches clairement définies :

```
┌─────────────────────────────────────────────────────────────────────┐
│                    COUCHE CONSOMMATION                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│  │  Power BI   │  │   Tableau   │  │  Notebooks  │  │   API ML    │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘ │
└─────────┼────────────────┼────────────────┼────────────────┼────────┘
          │                │                │                │
          ▼                ▼                ▼                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    COUCHE FÉDÉRATION (Dremio)                       │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │  Moteur SQL │ Réflexions │ Contrôle d'accès │ Virtualisation   ││
│  └─────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    COUCHE CATALOGUE (REST Catalog)                  │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │  Métadonnées │ Versionnement │ Gouvernance │ Lignage            ││
│  └─────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
          ┌───────────────────────┼───────────────────────┐
          │                       │                       │
          ▼                       ▼                       ▼
┌─────────────────┐   ┌─────────────────────┐   ┌─────────────────┐
│   Apache Spark  │   │    Kafka Connect    │   │     Trino       │
│   (Traitement   │   │    (Ingestion       │   │   (Requêtes     │
│    par lots)    │   │    temps réel)      │   │    ad hoc)      │
└────────┬────────┘   └──────────┬──────────┘   └────────┬────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    COUCHE STOCKAGE (Iceberg sur Objet)              │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │  Tables Iceberg │ Parquet │ Métadonnées │ Snapshots            ││
│  └─────────────────────────────────────────────────────────────────┘│
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │              Stockage Objet Infonuagique                        ││
│  └─────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────┘
```

### Feuille de route de migration

La migration vers cette architecture cible s'étendra sur 24 mois, organisée en quatre phases principales.

**Phase 1 : Fondations (Mois 1-6)**
- Déploiement de l'infrastructure de stockage objet
- Installation et configuration du catalogue REST
- Mise en place de Dremio pour la fédération
- Migration pilote du domaine marketing
- Formation des équipes de base

**Phase 2 : Expansion (Mois 7-12)**
- Migration des domaines opérationnels
- Intégration des flux temps réel Kafka
- Déploiement des capacités de gouvernance avancées
- Extension de l'adoption BI

**Phase 3 : Consolidation (Mois 13-18)**
- Migration des domaines réglementaires
- Optimisation des performances
- Décommission progressive de Snowflake
- Développement des cas d'usage ML

**Phase 4 : Optimisation (Mois 19-24)**
- Décommission de l'infrastructure Hadoop
- Réduction de l'empreinte Exadata
- Automatisation avancée des opérations
- Transfert complet aux équipes internes

| Phase | Domaines | Systèmes impactés | Budget |
|-------|----------|-------------------|--------|
| 1 | Marketing | Snowflake, Hadoop | 5 M$ |
| 2 | Opérations | Hadoop, Azure SQL | 8 M$ |
| 3 | Réglementaire | Exadata (partiel), Hadoop | 10 M$ |
| 4 | Optimisation | Tous | 5 M$ |

### La présentation itinérante

Un plan architectural, aussi solide soit-il, ne se réalisera pas sans l'adhésion des parties prenantes. La « présentation itinérante » (*roadshow*) désigne la série de présentations adaptées à chaque audience pour construire cette adhésion. Chaque groupe de parties prenantes a des préoccupations différentes et nécessite un message personnalisé.

**Présentation au Comité exécutif**

L'audience : Le PDG, le CFO, le CIO et les VP des principales lignes d'affaires.

Les préoccupations : Retour sur investissement, risques stratégiques, alignement avec la vision d'entreprise.

Le message clé : Le Lakehouse Apache Iceberg permettra à Hamerliwa de développer des capacités analytiques avancées tout en réduisant les coûts à terme. C'est un investissement défensif (conformité, réduction des risques) et offensif (compétitivité, innovation).

Le contenu :
- Contexte : pressions réglementaires et concurrentielles
- Constats d'audit résumés en indicateurs clés
- Vision architecturale en une diapositive
- Analyse financière : investissement, économies, ROI sur 5 ans
- Risques principaux et stratégies de mitigation
- Demande : approbation du budget et du calendrier

Durée : 30 minutes + 15 minutes de questions.

**Présentation au Comité de direction TI**

L'audience : Le CIO, les directeurs infrastructure, applications, sécurité et données.

Les préoccupations : Faisabilité technique, impacts sur les équipes, intégration avec l'existant.

Le message clé : L'architecture proposée s'appuie sur des technologies matures et des standards ouverts. La migration progressive minimise les risques et permet un apprentissage incrémental.

Le contenu :
- Architecture actuelle et ses limitations techniques
- Architecture cible détaillée avec choix technologiques justifiés
- Stratégie de migration par phases avec dépendances
- Impacts sur les équipes et plan de formation
- Métriques de succès et critères d'acceptation
- Gouvernance du projet et processus de décision

Durée : 60 minutes + 30 minutes de discussion.

**Présentation aux équipes métier**

L'audience : Les directeurs et gestionnaires des domaines marketing, risque, finance, opérations.

Les préoccupations : Continuité des services, amélioration de leurs capacités, efforts de leur part.

Le message clé : Le nouveau Lakehouse vous donnera accès à des données plus fraîches, plus complètes et plus faciles à exploiter. Votre participation à la définition des priorités est essentielle.

Le contenu :
- Frustrations actuelles validées par l'audit
- Capacités nouvelles promises par le Lakehouse
- Calendrier par domaine avec leurs dates clés
- Rôle attendu de leurs équipes
- Engagement à maintenir la continuité des services critiques
- Canal de communication et de remontée des problèmes

Durée : 45 minutes avec atelier interactif.

**Présentation aux équipes techniques**

L'audience : Les architectes, développeurs, administrateurs et analystes de données.

Les préoccupations : Technologies à apprendre, évolution de leurs rôles, qualité de l'implémentation.

Le message clé : Le Lakehouse représente une opportunité d'acquérir des compétences recherchées sur des technologies d'avenir. Votre expertise de l'existant est précieuse pour réussir la migration.

Le contenu :
- Plongée technique dans Apache Iceberg (lien avec Partie 1)
- Architecture détaillée avec composants et interfaces
- Stack technologique complet
- Plan de formation et certifications
- Processus de contribution et revue de code
- Démonstration pratique de l'environnement de développement

Durée : 90 minutes avec démonstration.

> **Étude de cas : Industrielle Alliance — La communication comme facteur de succès**  
> *Secteur* : Assurances et services financiers  
> *Défi* : Migration d'un entrepôt de données vers une architecture Lakehouse avec résistance initiale des équipes historiques.  
> *Solution* : Programme de communication structuré avec 25 sessions adaptées aux différentes audiences, complété par un site intranet dédié au projet et des bulletins hebdomadaires de progression.  
> *Résultats* : Taux d'adoption de 94 % à la fin de la première phase, contre 67 % pour un projet similaire sans programme de communication.

### Gestion des objections

Chaque présentation génèrera des questions et des objections légitimes. L'équipe projet doit anticiper ces objections et préparer des réponses factuelles. Une objection bien traitée renforce la crédibilité du projet ; une objection mal gérée peut compromettre l'adhésion de toute une audience.

**Objection : « Pourquoi ne pas simplement étendre Snowflake ? »**

Réponse : Snowflake est un excellent produit, mais crée une dépendance à un fournisseur unique avec des coûts croissants. Apache Iceberg offre la même flexibilité avec le contrôle des données et l'interopérabilité multi-fournisseur. De plus, Snowflake supporte maintenant Iceberg, donc les deux approches peuvent coexister.

*Données à l'appui* : Les coûts Snowflake d'Hamerliwa ont augmenté de 35 % l'an dernier pour une croissance des données de 20 %. Avec Iceberg sur stockage objet, le coût au téraoctet diminue quand le volume augmente.

**Objection : « L'équipe n'a pas les compétences nécessaires »**

Réponse : Le plan inclut un programme de formation substantiel et un accompagnement externe pendant la phase initiale. Les compétences Iceberg, Spark et Dremio sont transférables et valorisantes pour les carrières. Plusieurs membres de l'équipe ont déjà exprimé leur intérêt pour ces technologies.

*Plan de formation* : 40 heures de formation formelle par membre de l'équipe technique, complétées par un mentorat pendant 6 mois. Budget formation : 350 K$ sur la durée du projet.

**Objection : « C'est trop risqué pour les systèmes réglementaires »**

Réponse : C'est précisément pourquoi la migration des systèmes réglementaires est planifiée en phase 3, après que l'équipe ait acquis de l'expérience sur les domaines moins critiques. Une période de fonctionnement parallèle de 6 mois est prévue avant toute décommission.

*Mécanismes de mitigation* : Tests de non-régression automatisés, réconciliation quotidienne entre ancien et nouveau système, comité de validation réglementaire avant chaque décommission.

**Objection : « Les coûts sont trop élevés »**

Réponse : L'investissement total de 28 M$ sur 3 ans représente 1,9 fois le coût annuel actuel de la plateforme. Cependant, les économies projetées atteignent 1,2 M$/an à partir de l'année 4, générant un retour sur investissement positif en année 5. Sans cette modernisation, les coûts actuels continueront d'augmenter de 15-20 % par an.

*Scénario sans action* : En maintenant l'architecture actuelle, les coûts atteindront 7,2 M$/an en année 5 (projection basée sur les tendances historiques), contre 4,1 M$/an avec le Lakehouse.

**Objection : « Nous avons d'autres priorités »**

Réponse : Les exigences réglementaires (Loi 25, BSIF) ne sont pas optionnelles et leurs échéances approchent. De plus, chaque mois de retard accumule de la dette technique et repousse les bénéfices. Le plan par phases permet de démarrer avec des ressources limitées et d'ajuster selon les résultats.

*Calendrier réglementaire* : Les nouvelles exigences BSIF sur la gestion des données entrent en vigueur en janvier 2027. La phase 3 du projet, qui adresse la conformité réglementaire, doit être complétée 3 mois avant cette date pour permettre les validations nécessaires.

**Objection : « Nos concurrents utilisent une autre approche »**

Réponse : Les approches varient selon les contextes. Toutefois, Apache Iceberg est devenu le standard de facto pour les nouveaux projets Lakehouse. Son adoption par Apple, Netflix, Adobe et de nombreuses institutions financières témoigne de sa maturité. En choisissant un format ouvert, Hamerliwa conserve la flexibilité d'évoluer sans réécrire toute son infrastructure.

**Objection : « Le projet va perturber nos opérations quotidiennes »**

Réponse : Le plan prévoit explicitement une coexistence prolongée entre l'ancien et le nouveau système. Aucune décommission ne se fera avant 6 mois de fonctionnement stable en parallèle. Les équipes opérationnelles seront impliquées dans la définition des critères de bascule et auront un droit de veto sur le calendrier de décommission.

### Stratégie de gestion du changement

Au-delà des présentations formelles, la réussite du projet nécessite une stratégie de gestion du changement structurée.

**Identification des champions** : Dans chaque équipe impactée, identifiez un ou deux individus enthousiastes qui deviendront les ambassadeurs du projet. Ces champions reçoivent une formation avancée et participent aux décisions de conception. Leur rôle est de relayer l'information dans leurs équipes et de remonter les préoccupations.

**Communication régulière** : Établissez un rythme de communication prévisible. Par exemple, un bulletin hebdomadaire par courriel résumant les progrès, les prochaines étapes et les points d'attention. Un site intranet dédié au projet centralise la documentation, les questions fréquentes et les ressources de formation.

**Célébration des succès** : Chaque jalon atteint mérite d'être communiqué et célébré. La migration réussie du premier domaine, la première requête exécutée 10 fois plus vite qu'avant, le premier tableau de bord migré sans modification — ces succès renforcent la confiance et l'élan du projet.

**Gestion des résistances** : Certaines résistances sont inévitables. Plutôt que de les ignorer ou de les combattre frontalement, cherchez à comprendre leurs causes profondes. Souvent, la résistance masque une peur légitime (perte de compétences, changement de rôle, incertitude). Adressez ces peurs directement et impliquez les résistants dans la conception des solutions.

> **Étude de cas : Mouvement Desjardins — La force du réseau de champions**  
> *Secteur* : Services financiers coopératifs  
> *Défi* : Migration de 4 500 utilisateurs vers une nouvelle plateforme analytique dans un contexte de résistance historique au changement.  
> *Solution* : Réseau de 120 champions formés intensivement, répartis dans toutes les régions et lignes d'affaires. Ces champions ont organisé plus de 300 sessions de démonstration locales et ont servi de premier niveau de support.  
> *Résultats* : Taux d'adoption de 91 % après 6 mois, contre une cible de 75 %. Le réseau de champions a été maintenu pour les évolutions futures de la plateforme.

### Documentation et gouvernance du plan

Le plan architectural et les exigences doivent être documentés formellement pour servir de référence tout au long du projet. Cette documentation inclut :

**Document de vision architecturale** : Synthèse exécutive de l'architecture cible, des principes directeurs et des bénéfices attendus. Ce document d'une dizaine de pages sert de référence commune à toutes les parties prenantes.

**Dossier d'exigences** : Liste complète des exigences fonctionnelles, non fonctionnelles, de migration et de gouvernance, avec traçabilité vers les constats d'audit et critères d'acceptation mesurables.

**Plan de projet** : Feuille de route détaillée avec jalons, livrables, ressources et dépendances. Ce plan sera affiné au cours des chapitres suivants qui traiteront chaque couche de l'architecture.

**Registre des décisions architecturales** : Journal des décisions significatives avec contexte, options considérées, décision retenue et justification. Ce registre facilite la compréhension des choix par les nouveaux membres de l'équipe et évite de revisiter des décisions déjà prises.

**Matrice RACI** : Clarification des rôles et responsabilités pour chaque livrable et décision majeure du projet.

---

## IV.4.5 Résumé

Ce chapitre a établi les fondations méthodologiques pour préparer votre passage à Apache Iceberg. À travers l'étude de cas de la Banque Hamerliwa, nous avons illustré concrètement chaque étape de cette préparation critique.

### Concepts clés

**L'audit de plateforme** constitue le point de départ indispensable de tout projet de migration. Organisé en six dimensions (architecture, données, flux, consommation, gouvernance, coûts), il fournit une compréhension factuelle de l'existant qui éclaire toutes les décisions ultérieures.

**La transformation des constats en exigences** traduit les observations qualitatives de l'audit en spécifications exploitables. Les exigences se structurent en quatre catégories : fonctionnelles, non fonctionnelles, de migration et de gouvernance. La matrice de traçabilité assure que chaque constat critique trouve une réponse dans les exigences.

**Le plan architectural de haut niveau** définit les orientations technologiques majeures avant d'entrer dans le détail de chaque composant. Pour Hamerliwa, cinq piliers structurent cette vision : Apache Iceberg comme format universel, stockage objet infonuagique, catalogue REST unifié, couche de fédération Dremio et intégration Kafka.

**La présentation itinérante** reconnaît que l'adhésion des parties prenantes est aussi importante que la qualité technique du plan. Chaque audience (exécutifs, direction TI, métiers, techniques) nécessite un message adapté à ses préoccupations spécifiques.

### Livrables de la phase de préparation

À l'issue de cette phase, les livrables suivants doivent être disponibles :

| Livrable | Contenu | Audience |
|----------|---------|----------|
| Rapport d'audit | Constats par dimension, forces/faiblesses, métriques de référence | Direction, équipe projet |
| Cartographie des actifs | Inventaire des systèmes, flux, données | Équipe technique |
| Dossier d'exigences | Exigences structurées avec traçabilité | Équipe projet, fournisseurs |
| Document de vision | Architecture cible, principes, bénéfices | Toutes parties prenantes |
| Plan de projet | Phases, jalons, ressources, budget | Direction, PMO |
| Supports de présentation | Versions adaptées par audience | Équipe projet |

### Facteurs de succès

L'expérience des projets de migration similaires révèle plusieurs facteurs déterminants pour le succès :

**Engagement visible de la direction** : Sans un sponsor exécutif actif et visible, le projet peinera à obtenir les ressources et l'attention nécessaires. Le sponsor doit comprendre suffisamment le projet pour le défendre devant ses pairs.

**Qualité de l'audit initial** : Les surprises découvertes pendant l'implémentation coûtent plus cher que le temps investi dans un audit approfondi. Mieux vaut repousser le démarrage de quelques semaines pour un audit complet que de subir des retards de plusieurs mois en cours de projet.

**Communication continue** : La présentation itinérante n'est pas un événement ponctuel mais le début d'un processus de communication continu. Les parties prenantes doivent être informées régulièrement de la progression, des succès et des défis.

**Réalisme du calendrier** : Les projets de migration sous-estiment systématiquement les efforts requis. Un calendrier réaliste avec des marges pour les imprévus génère plus de confiance qu'un calendrier agressif qui sera rapidement dépassé.

**Équipe dédiée** : Les migrations réussies s'appuient sur une équipe dédiée au projet, pas sur des ressources partagées entre les opérations courantes et le projet. Le coût apparent de cette dédicace est compensé par l'accélération de la livraison.

### Prochaines étapes

Ce chapitre conclut la préparation stratégique du passage à Apache Iceberg. Les chapitres suivants de la Partie 2 aborderont la conception détaillée de chaque couche de l'architecture :

- **Chapitre IV.5** : Sélection de la couche de stockage — évaluation des options de stockage objet et critères de décision
- **Chapitre IV.6** : Architecture de la couche d'ingestion — patrons d'ingestion par lots et temps réel, intégration Kafka
- **Chapitre IV.7** : Implémentation de la couche de catalogue — options de catalogue REST et critères de sélection
- **Chapitre IV.8** : Conception de la couche de fédération — Dremio, Trino et alternatives
- **Chapitre IV.9** : Comprendre la couche de consommation — outils BI, ML et interfaces d'accès

Chaque chapitre reprendra les exigences définies pour Hamerliwa et les utilisera pour guider les décisions architecturales spécifiques à chaque couche. Cette approche systématique assure la cohérence entre les exigences métier et les choix techniques.

### Commandes et outils essentiels

Bien que ce chapitre soit principalement méthodologique, voici quelques outils utiles pour la phase d'audit :

**Profilage des données avec Apache Spark** :
```python
# Analyse du volume et de la distribution des données
df = spark.read.parquet("s3://bucket/donnees/")
df.printSchema()
df.describe().show()
df.groupBy("partition_col").count().orderBy("count", ascending=False).show()
```

**Inventaire des tables Hive/Iceberg** :
```sql
-- Lister toutes les tables d'un namespace
SHOW TABLES IN database_name;

-- Obtenir les métadonnées d'une table
DESCRIBE EXTENDED database_name.table_name;

-- Analyser les snapshots Iceberg
SELECT * FROM database_name.table_name.snapshots;
```

**Analyse des requêtes (Dremio)** :
```sql
-- Examiner l'historique des requêtes
SELECT query_text, submitted_ts, finish_ts, 
       TIMESTAMPDIFF(SECOND, submitted_ts, finish_ts) as duration_sec
FROM sys.jobs
WHERE finish_ts > CURRENT_TIMESTAMP - INTERVAL '7' DAY
ORDER BY duration_sec DESC
LIMIT 20;
```

> **Performance**  
> Lors de l'audit, évitez d'exécuter des analyses lourdes pendant les heures de pointe des systèmes de production. Planifiez les requêtes de profilage pendant les fenêtres de maintenance ou utilisez des répliques de lecture si disponibles. Pour les très grands volumes, échantillonnez les données plutôt que d'analyser l'ensemble complet.

### Transition vers le chapitre suivant

Avec les exigences clairement définies et le plan architectural approuvé, nous pouvons maintenant entrer dans le détail de la conception. Le Chapitre IV.5 ouvrira cette exploration en abordant la couche fondamentale de toute architecture Lakehouse : le stockage. Nous évaluerons les différentes options de stockage objet disponibles, comparerons leurs caractéristiques de performance et de coûts, et appliquerons les exigences d'Hamerliwa pour guider la sélection.

---

**Références**

- Inmon, W.H. et Linstedt, D. (2024). *Data Architecture: From Zen to Reality*. Technics Publications.
- Kimball, R. et Ross, M. (2023). *The Data Warehouse Toolkit*, 4e édition. Wiley.
- Reis, J. et Housley, M. (2022). *Fundamentals of Data Engineering*. O'Reilly Media.
- Apache Iceberg (2025). *Migration Guide*. https://iceberg.apache.org/docs/latest/migration/
- Dremio (2024). *Lakehouse Migration Best Practices*. https://www.dremio.com/resources/
- BSIF (2025). *Ligne directrice E-21 : Gestion des risques liés aux technologies de l'information*. https://www.osfi-bsif.gc.ca/
- Commission d'accès à l'information du Québec (2024). *Guide d'application de la Loi 25*. https://www.cai.gouv.qc.ca/
- Gartner (2024). *Magic Quadrant for Data Integration Tools*. Gartner Research.
- Forrester (2024). *The State of Data Management*. Forrester Research.
