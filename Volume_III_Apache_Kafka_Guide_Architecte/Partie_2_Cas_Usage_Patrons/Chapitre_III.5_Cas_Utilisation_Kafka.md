# Chapitre III.5

## CAS D'UTILISATION KAFKA

---

> *« Kafka n'est pas la solution à tous les problèmes de données. Mais pour les problèmes qu'il résout, il les résout exceptionnellement bien. »*
>
> — Jay Kreps, Co-créateur d'Apache Kafka

---

Les chapitres précédents ont établi les fondements techniques de Kafka : architecture du cluster, production et consommation de messages. Cette maîtrise technique est nécessaire mais insuffisante. L'architecte doit également savoir **quand** utiliser Kafka, **comment** naviguer les défis d'implémentation en contexte réel, et **quelles alternatives** considérer selon les cas d'usage.

Ce chapitre adopte une perspective pragmatique. Kafka est un outil puissant, mais comme tout outil, il excelle dans certains contextes et s'avère inadapté dans d'autres. Une adoption non critique de Kafka peut introduire une complexité injustifiée, tandis qu'un refus dogmatique peut priver l'organisation d'une plateforme transformationnelle.

Nous explorerons les critères de décision pour choisir Kafka, les défis d'implémentation en contexte réel avec des retours d'expérience concrets, les différences fondamentales avec les autres plateformes de messagerie, et les alternatives à considérer selon les cas d'usage spécifiques.

---

## III.5.1 Quand Choisir Kafka — et Quand Ne Pas le Faire

### Les Forces Fondamentales de Kafka

Apache Kafka excelle dans des scénarios spécifiques où ses caractéristiques architecturales apportent une valeur distinctive. Comprendre ces forces permet d'identifier les cas d'usage optimaux.

**Durabilité et rétention des messages.** Contrairement aux systèmes de messagerie traditionnels qui suppriment les messages après consommation, Kafka persiste les messages sur disque pour une durée configurable (heures, jours, ou indéfiniment). Cette durabilité permet le replay, la reconstruction de systèmes, et la consommation par plusieurs applications à des rythmes différents.

**Débit massif.** L'architecture de Kafka, basée sur l'écriture séquentielle et le zero-copy, permet des débits de millions de messages par seconde sur un cluster correctement dimensionné. Cette capacité est essentielle pour les pipelines de données à haute vélocité.

**Ordre garanti par partition.** Les messages dans une partition sont strictement ordonnés. Cette garantie est fondamentale pour les cas d'usage où l'ordre des événements est sémantiquement important (transactions financières, audit, Event Sourcing).

**Scalabilité horizontale.** L'ajout de brokers et de partitions permet de scaler linéairement la capacité du système. Cette élasticité supporte la croissance des volumes de données sans refonte architecturale.

**Découplage producteur-consommateur.** Les producteurs et consommateurs sont complètement découplés. Un producteur peut écrire sans se soucier de qui consomme, et un consommateur peut lire à son propre rythme. Ce découplage facilite l'évolution indépendante des systèmes.

**Écosystème riche.** Kafka Connect pour l'intégration, Kafka Streams pour le traitement, Schema Registry pour la gouvernance des schémas, et des centaines de connecteurs pré-construits forment un écosystème complet.

### Cas d'Usage Optimaux pour Kafka

Certains patterns architecturaux bénéficient particulièrement des caractéristiques de Kafka. L'architecte doit reconnaître ces patterns dans les besoins métier pour recommander Kafka de manière appropriée.

**Event Sourcing et CQRS.** L'Event Sourcing persiste l'état d'une entité comme une séquence d'événements plutôt que comme un état final. Chaque modification génère un événement immuable qui est ajouté au journal. L'état actuel est reconstruit en rejouant les événements depuis le début. Kafka est idéal pour ce pattern : sa rétention durable permet de reconstruire l'état à partir des événements, son ordre par partition garantit la cohérence temporelle, et sa capacité multi-consommateur permet de dériver plusieurs vues (CQRS) depuis le même flux d'événements.

Le pattern CQRS (Command Query Responsibility Segregation) sépare les modèles de lecture et d'écriture. Les commandes modifient l'état et génèrent des événements. Les requêtes lisent des modèles optimisés pour la lecture, construits en consommant les événements. Kafka permet à plusieurs projections (modèles de lecture) de consommer le même flux d'événements indépendamment, chacune optimisée pour un cas d'usage spécifique.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        Event Sourcing avec Kafka                        │
│                                                                         │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────────────┐ │
│  │  Commande   │───▶│   Kafka     │───▶│   Projection Read Model     │ │
│  │  Service    │    │   Topic     │    │   (Base de données)         │ │
│  └─────────────┘    │  (Events)   │    └─────────────────────────────┘ │
│                     │             │                                     │
│                     │             │    ┌─────────────────────────────┐ │
│                     │             │───▶│   Projection Analytics      │ │
│                     │             │    │   (Data Warehouse)          │ │
│                     │             │    └─────────────────────────────┘ │
│                     │             │                                     │
│                     │             │    ┌─────────────────────────────┐ │
│                     │             │───▶│   Projection Audit          │ │
│                     │             │    │   (Stockage long terme)     │ │
│                     └─────────────┘    └─────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

> **Exemple concret**
>
> *Scénario* : Système bancaire où chaque compte est modélisé par ses événements (dépôt, retrait, transfert).
>
> *Implémentation* : Topic `account-events` partitionné par `account-id`. Chaque événement contient le type, le montant, le timestamp, et les métadonnées.
>
> *Bénéfices* :
> - Le solde actuel est calculé en rejouant les événements
> - L'historique complet est disponible pour l'audit réglementaire
> - Plusieurs vues (mobile, web, rapports) dérivent du même flux
> - En cas de bug dans le calcul du solde, correction et reconstruction possible
> - Conformité réglementaire facilitée par l'immutabilité des événements

**Intégration de données en temps réel (CDC).** Le Change Data Capture est un pattern qui capture les modifications des bases de données et les propage vers d'autres systèmes. Kafka Connect avec des connecteurs comme Debezium permet de capturer les changements au niveau du journal de transactions de la base source (binlog MySQL, WAL PostgreSQL), garantissant qu'aucune modification n'est manquée.

Ce pattern remplace avantageusement les ETL batch traditionnels qui introduisent une latence de plusieurs heures entre les systèmes. Avec le CDC vers Kafka, les changements sont propagés en secondes, permettant des architectures véritablement temps réel.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        CDC avec Kafka Connect                           │
│                                                                         │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌──────────┐ │
│  │  MySQL      │───▶│  Debezium   │───▶│   Kafka     │───▶│ Elastic  │ │
│  │  (Source)   │    │  Connector  │    │   Topic     │    │ Search   │ │
│  └─────────────┘    └─────────────┘    │             │    └──────────┘ │
│                                        │             │                  │
│  ┌─────────────┐    ┌─────────────┐    │             │    ┌──────────┐ │
│  │ PostgreSQL  │───▶│  Debezium   │───▶│             │───▶│ Data     │ │
│  │  (Source)   │    │  Connector  │    │             │    │ Lake     │ │
│  └─────────────┘    └─────────────┘    └─────────────┘    └──────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

**Pipelines de données streaming.** Pour les flux de données continus nécessitant transformation, enrichissement, et agrégation, Kafka offre un écosystème complet. Kafka Streams permet le traitement stateful avec des tables et des jointures. ksqlDB offre une interface SQL pour le streaming. Apache Flink peut se connecter nativement à Kafka pour les traitements les plus complexes.

Les cas d'usage incluent : enrichissement d'événements avec des données de référence, agrégations en temps réel (compteurs, moyennes mobiles), détection de patterns complexes (CEP), et transformation de format entre systèmes.

**Communication inter-services à grande échelle.** Dans les architectures microservices avec des dizaines ou centaines de services, Kafka fournit un backbone de communication asynchrone découplé. Comparé aux appels REST synchrones point-à-point, Kafka offre plusieurs avantages :

*Découplage temporel* : Le producteur n'attend pas que le consommateur soit disponible. Si un service est temporairement indisponible, les messages s'accumulent dans Kafka et seront traités à son retour.

*Découplage de connaissances* : Le producteur n'a pas besoin de connaître les consommateurs. De nouveaux services peuvent s'abonner aux événements existants sans modifier le producteur.

*Résilience* : La persistance de Kafka garantit qu'aucun message n'est perdu même si des services crashent.

*Historique* : La rétention permet le replay pour le débogage, la reconstruction, ou l'ajout de nouveaux consommateurs.

**Collecte de métriques et logs.** La capacité de Kafka à ingérer des millions d'événements par seconde le rend idéal pour la collecte centralisée de métriques applicatives, de logs, et de traces de monitoring. Les producteurs (agents sur chaque serveur) publient continuellement vers Kafka. Les consommateurs (systèmes d'analyse, stockage) traitent ces flux à leur rythme.

L'architecture découplée permet d'ajouter de nouveaux consommateurs (nouveau système d'alerte, nouveau data lake) sans modifier les producteurs déployés sur des milliers de serveurs.

**Machine Learning en temps réel.** Les pipelines de ML modernes nécessitent des données fraîches pour le feature engineering, la détection d'anomalies, et le scoring en temps réel. Kafka permet de construire des pipelines où les événements bruts sont enrichis, transformés en features, et scorés par des modèles ML en quelques millisecondes.

Cas d'usage typiques :
- Détection de fraude sur les transactions en temps réel
- Recommandations personnalisées basées sur le comportement récent
- Pricing dynamique ajusté aux conditions du marché
- Maintenance prédictive basée sur les flux IoT

> **Note de terrain**
>
> *Contexte* : Grande banque canadienne avec 50+ systèmes legacy produisant des événements de transactions.
>
> *Problème initial* : Chaque système avait ses propres intégrations point-à-point. 200+ intégrations à maintenir, latence de propagation de plusieurs heures. L'ajout d'un nouveau consommateur nécessitait des mois de développement.
>
> *Solution* : Kafka comme backbone événementiel central. Chaque système publie ses événements dans Kafka avec un format standardisé. Les consommateurs s'abonnent aux événements pertinents via des groupes de consommateurs.
>
> *Résultats quantifiés* :
> - Intégrations réduites de 200+ à 50+ (systèmes vers Kafka uniquement)
> - Latence de propagation : heures → secondes (réduction de 99%)
> - Temps d'intégration d'un nouveau consommateur : mois → jours
> - Nouveaux cas d'usage (détection de fraude temps réel, Customer 360) rendus possibles
> - Audit complet via rétention des événements pour conformité réglementaire
>
> *Leçon* : Kafka transforme les architectures point-à-point en architecture hub-and-spoke avec des bénéfices multiplicateurs. L'investissement initial est significatif mais le ROI sur 3 ans est substantiel.

### Cas d'Usage Où Kafka N'est Pas Optimal

Kafka n'est pas la solution universelle. L'architecte doit reconnaître les cas d'usage où d'autres technologies sont plus appropriées. Utiliser Kafka là où il n'est pas nécessaire introduit une complexité injustifiée, augmente les coûts opérationnels, et peut dégrader les performances.

**Files de travail (work queues) simples.** Si le besoin est de distribuer des tâches à des workers sans exigence d'ordre ou de replay, des systèmes comme RabbitMQ, Amazon SQS, ou Redis sont plus simples à opérer et suffisants. Le pattern « competing consumers » où plusieurs workers se disputent les messages d'une file est le modèle natif de ces systèmes.

Kafka peut implémenter ce pattern avec un groupe de consommateurs, mais il apporte des complexités non nécessaires : gestion des partitions (le parallélisme est limité par le nombre de partitions, pas par le nombre de workers), rééquilibrage lors de l'ajout/suppression de workers, et persistance des messages après traitement (consommation d'espace disque inutile).

*Exemple* : Traitement asynchrone de fichiers uploadés. L'utilisateur uploade un fichier, un message est envoyé dans une file, un worker traite le fichier. Pas besoin de replay, pas besoin d'ordre, un seul consommateur suffit. RabbitMQ ou SQS sont plus appropriés.

**Communication request-response.** Pour les interactions synchrones où un service attend une réponse immédiate, REST/gRPC sont plus appropriés. Kafka peut implémenter request-response avec des topics de requête et de réponse, des correlation IDs, et des consommateurs temporaires, mais c'est un anti-pattern qui complexifie l'architecture et augmente la latence.

*Symptôme* : Si l'équipe implémente un pattern où le producteur attend une réponse dans un topic dédié avec timeout, c'est probablement un cas où REST/gRPC serait plus simple.

**Messages avec TTL court et consommation unique.** Si les messages doivent expirer rapidement (quelques minutes) et ne seront consommés qu'une fois par un seul consommateur, une file de messages traditionnelle est plus adaptée. Kafka conserve les messages pour la durée de rétention configurée, ce qui est un gaspillage si les messages n'ont plus de valeur après quelques minutes.

**Faibles volumes de données.** Pour quelques centaines de messages par jour, la complexité opérationnelle de Kafka est disproportionnée. Un cluster Kafka minimum viable (3 brokers avec réplication, ZooKeeper/KRaft, monitoring) représente un investissement significatif en infrastructure et en compétences.

*Règle empirique* : En dessous de 1 000 messages par minute, évaluer sérieusement des alternatives plus simples. En dessous de 100 messages par minute, Kafka est presque certainement over-engineering.

**Transactions distribuées ACID.** Bien que Kafka supporte les transactions (depuis la version 0.11), elles sont limitées au scope Kafka. Une transaction Kafka garantit l'atomicité entre la consommation de messages et la production de messages vers d'autres topics Kafka. Elle ne s'étend pas aux bases de données externes.

Pour les transactions impliquant des bases de données avec garanties ACID strictes, des patterns comme Saga (orchestrée ou chorégraphiée) ou Two-Phase Commit sont nécessaires. Ces patterns ajoutent de la complexité et ne sont pas natifs à Kafka.

*Alternative* : Si la transaction doit être ACID avec une base de données, le pattern Outbox (écriture dans une table de la même base dans la même transaction, puis publication vers Kafka par un processus séparé) est souvent plus approprié.

**Requêtes ad-hoc sur les données.** Kafka n'est pas une base de données. Les données dans Kafka sont organisées par partition et offset, pas par clé arbitraire. Interroger l'historique des messages nécessite de les relire séquentiellement depuis le début ou depuis un offset connu.

Pour les requêtes analytiques (« combien de commandes de plus de 100€ cette semaine ? »), les données doivent être déversées dans un système approprié : data warehouse (Snowflake, BigQuery), data lake (Iceberg, Delta Lake), base de données analytique (ClickHouse, Druid).

*Pattern courant* : Kafka comme couche de transport, Iceberg/Delta Lake comme couche de stockage analytique, moteur SQL (Trino, Spark) pour les requêtes.

**Besoins de routage complexe.** Kafka offre un routage simple basé sur les topics : un producteur publie vers un topic, les consommateurs s'abonnent aux topics pertinents. Pour le routage basé sur le contenu des messages (envoyer les commandes de plus de 1000€ vers une file prioritaire), le producteur ou un consommateur intermédiaire doit implémenter cette logique.

RabbitMQ avec ses exchanges (direct, topic, headers, fanout) offre un routage déclaratif plus puissant sans code applicatif.

> **Anti-patron**
>
> *« Nous utilisons Kafka pour tout, même pour les notifications email ponctuelles et les jobs de nettoyage nocturnes. »*
>
> *Problème* : Over-engineering systématique. Kafka pour 100 emails/jour nécessite un cluster, du monitoring, des compétences spécifiques, alors qu'un simple système de file (ou même une table de base de données avec un cron job) suffirait amplement.
>
> *Conséquences observées* :
> - Coût opérationnel élevé (infrastructure, monitoring, astreinte)
> - Complexité de débogage pour les équipes non familières avec Kafka
> - Latence ajoutée pour des cas simples (le message passe par Kafka inutilement)
> - Dépendance critique sur Kafka même pour des fonctionnalités non critiques
>
> *Meilleure approche* : Utiliser Kafka pour les cas à haute valeur (événements métier critiques, streaming temps réel, haute volumétrie) et des solutions plus simples pour les cas triviaux. Définir des critères clairs pour l'utilisation de Kafka vs. alternatives.

> **Perspective stratégique**
>
> L'adoption de Kafka doit être guidée par les besoins réels, pas par la popularité de la technologie ou le désir de « moderniser ». Chaque composant d'infrastructure ajouté augmente la surface de complexité, les compétences requises, et les points de défaillance potentiels.
>
> La question clé n'est pas « pouvons-nous utiliser Kafka ? » mais « avons-nous besoin des caractéristiques uniques de Kafka (durabilité, replay, multi-consommateur, haute vélocité) ? »

### Matrice de Décision

La décision d'utiliser Kafka doit être basée sur une évaluation multicritère. Le tableau suivant guide cette évaluation en fournissant des seuils concrets.

| Critère | Kafka Recommandé | Kafka Déconseillé |
|---------|------------------|-------------------|
| Volume de messages | > 10 000/seconde | < 100/seconde |
| Besoin de replay | Oui, fréquent | Non, jamais |
| Consommateurs multiples | Plusieurs groupes indépendants | Un seul consommateur |
| Rétention des messages | Jours/semaines/indéfini | Minutes/heures |
| Ordre des messages | Critique | Non important |
| Pattern de communication | Pub/Sub, Streaming | Request/Response |
| Durabilité | Critique | Best-effort acceptable |
| Compétences équipe | Disponibles ou à développer | Absentes et non prioritaires |
| Budget opérationnel | Suffisant pour cluster | Limité |

**Utilisation de la matrice :**

Pour chaque critère, évaluer si le cas d'usage penche vers « Kafka Recommandé » ou « Kafka Déconseillé ». Si la majorité des critères penchent vers « Recommandé », Kafka est probablement approprié. Si la majorité penchent vers « Déconseillé », explorer les alternatives.

*Zone grise* : Quand les critères sont partagés (par exemple, volume élevé mais pas de besoin de replay), une analyse plus approfondie est nécessaire. Considérer le coût total de possession (infrastructure, formation, opérations) versus les alternatives.

> **Décision architecturale**
>
> *Contexte* : Startup fintech en phase de croissance. Actuellement 1 000 transactions/jour, projection à 100 000/jour dans 18 mois. Équipe de 8 développeurs, aucune expertise Kafka.
>
> *Évaluation par la matrice* :
> - Volume : Actuellement faible, mais croissance attendue → Partagé
> - Replay : Nécessaire pour audit réglementaire → Kafka
> - Consommateurs : Plusieurs systèmes (notifications, analytics, compliance) → Kafka
> - Rétention : 7 ans pour compliance → Kafka
> - Ordre : Critique pour les transactions → Kafka
> - Compétences : Absentes mais à développer → Partagé
>
> *Question clé* : Investir maintenant ou attendre ?
>
> *Options analysées* :
> 1. Kafka auto-géré maintenant : Prêt pour la croissance, mais complexité opérationnelle élevée pour une petite équipe sans expertise.
> 2. Solution simple (PostgreSQL) maintenant, migration plus tard : Moins de complexité immédiate, mais coût de migration significatif et dette technique.
> 3. Kafka managé (Confluent Cloud) maintenant : Complexité opérationnelle réduite, coût financier plus élevé mais prévisible.
>
> *Décision* : Option 3 — Kafka managé via Confluent Cloud. La croissance est certaine et les exigences réglementaires justifient Kafka. Le service managé réduit la charge opérationnelle pour une équipe en croissance.
>
> *Critères de révision* : Évaluer le passage à un cluster auto-géré si les coûts Confluent Cloud dépassent 50 000$/an et que l'équipe a développé l'expertise nécessaire.

### Anti-Patterns et Erreurs Courantes

L'expérience collective des implémentations Kafka révèle des erreurs récurrentes à éviter.

**Anti-pattern : Kafka comme base de données.**

*Symptôme* : Requêtes fréquentes sur l'historique des messages, tentatives d'indexation des topics, utilisation de Kafka pour des lookups par clé.

*Problème* : Kafka est optimisé pour l'écriture et la lecture séquentielle, pas pour les requêtes aléatoires. Les performances se dégradent et l'architecture devient fragile.

*Solution* : Utiliser Kafka pour le transport et un système approprié (base de données, data lake) pour le stockage interrogeable. Kafka Connect facilite cette séparation.

**Anti-pattern : Topic fourre-tout.**

*Symptôme* : Un topic contient des événements de types très différents (commandes, utilisateurs, produits, logs), différenciés par un champ `event_type`.

*Problème* : Les consommateurs doivent filtrer les messages non pertinents, les schémas deviennent complexes (union de tous les types), la rétention ne peut pas être configurée par type.

*Solution* : Un topic par type d'événement avec un schéma dédié. Plus de topics à gérer, mais architecture plus claire et performante.

**Anti-pattern : Ignorer le partitionnement.**

*Symptôme* : Utilisation de clés de partition aléatoires ou nulles, messages avec la même clé logique dispersés sur plusieurs partitions.

*Problème* : L'ordre n'est pas garanti pour une même entité, les jointures côté consommateur deviennent complexes ou impossibles.

*Solution* : Choisir une clé de partition alignée avec les besoins métier (ID de l'entité principale). S'assurer que les messages devant être traités dans l'ordre partagent la même clé.

**Anti-pattern : Commit automatique en production critique.**

*Symptôme* : `enable.auto.commit=true` avec un traitement qui peut échouer après le commit.

*Problème* : Les messages sont marqués comme consommés avant d'être réellement traités. En cas d'échec, ils sont perdus.

*Solution* : Commit manuel après traitement réussi, ou utilisation de transactions pour l'exactly-once.

**Anti-pattern : Sous-dimensionnement des partitions.**

*Symptôme* : Un topic avec 3 partitions pour un cas d'usage qui nécessitera 50 consommateurs parallèles.

*Problème* : Le nombre de partitions ne peut pas être réduit et l'augmentation peut causer une redistribution des clés. Le parallélisme est limité à 3.

*Solution* : Planifier le nombre de partitions en fonction du parallélisme maximum anticipé, avec une marge de croissance. Réviser lors des exercices de planification de capacité.

---

## III.5.2 Naviguer dans l'Implémentation en Contexte Réel

### Les Défis Organisationnels

L'adoption de Kafka dépasse la technique. Les défis organisationnels sont souvent plus complexes que les défis techniques.

**Compétences et formation.** Kafka requiert des compétences spécifiques : compréhension du modèle de partitionnement, gestion des offsets, tuning des producteurs/consommateurs, opération du cluster. Sans ces compétences, les équipes produisent des implémentations fragiles.

**Changement de paradigme.** Passer d'une architecture synchrone (appels REST entre services) à une architecture asynchrone (événements via Kafka) est un changement de paradigme. Les développeurs habitués au request-response doivent apprendre à penser en termes d'événements et de réactions.

**Gouvernance des topics.** Sans gouvernance, la prolifération de topics devient ingérable. Qui peut créer un topic ? Quelles conventions de nommage ? Quelle rétention par défaut ? Ces questions doivent être adressées avant l'adoption à grande échelle.

**Responsabilité des schémas.** Qui est responsable de l'évolution des schémas de messages ? Comment coordonner les changements entre producteurs et consommateurs ? Le Schema Registry aide techniquement, mais les processus organisationnels doivent l'accompagner.

> **Note de terrain**
>
> *Contexte* : Entreprise de commerce électronique, 200 développeurs, adoption de Kafka en cours.
>
> *Problème observé* : Chaque équipe créait ses topics avec des conventions différentes, des configurations aléatoires, des schémas non documentés. Après 6 mois, 150 topics existaient, 40% abandonnés ou non maintenus.
>
> *Solution mise en place* :
> 1. **Centre d'excellence Kafka** : Équipe de 3 personnes responsable des standards et de l'accompagnement.
> 2. **Processus de création de topic** : Demande via PR avec justification, revue par le centre d'excellence.
> 3. **Conventions de nommage** : `{domaine}.{sous-domaine}.{entité}.{version}` (ex: `commerce.orders.created.v1`)
> 4. **Schémas obligatoires** : Tout topic doit avoir un schéma Avro enregistré.
> 5. **Propriétaire identifié** : Chaque topic a un propriétaire responsable de sa maintenance.
>
> *Résultats après 1 an* : 80 topics actifs, tous documentés, schémas versionnés, coût opérationnel maîtrisé.
>
> *Leçon* : La gouvernance est aussi importante que la technologie. Sans elle, l'adoption de Kafka crée du chaos.

### Défis Techniques Courants

Certains défis techniques reviennent fréquemment lors des implémentations Kafka.

**Choix du nombre de partitions.** Le nombre de partitions d'un topic ne peut pas être réduit après création. Trop peu de partitions limite le parallélisme futur. Trop de partitions augmente la charge sur le cluster et la latence de rééquilibrage.

*Règle empirique* : Commencer avec un nombre de partitions égal au débit cible divisé par le débit d'un consommateur, avec une marge de 2-3× pour la croissance. Réviser lors de la planification de capacité annuelle.

**Stratégie de partitionnement.** Le choix de la clé de partitionnement détermine la distribution des messages et les garanties d'ordre. Une clé mal choisie peut créer des « hot partitions » (partitions surchargées) ou briser les invariants métier.

*Bonnes pratiques* :
- Choisir une clé avec une cardinalité suffisante (pas seulement 3 valeurs possibles)
- S'assurer que l'ordre par clé correspond aux besoins métier
- Monitorer la distribution des messages entre partitions

**Gestion de la rétention.** La rétention consomme de l'espace disque. Une rétention trop longue peut épuiser le stockage ; une rétention trop courte peut empêcher le replay nécessaire.

*Approche recommandée* :
- Définir la rétention basée sur les exigences métier (replay, audit, conformité)
- Utiliser la compaction pour les topics représentant un état (dernier état par clé)
- Monitorer l'utilisation disque et alerter avant saturation

**Exactly-once vs. at-least-once.** L'exactly-once ajoute de la latence et de la complexité (transactions). Beaucoup de cas d'usage tolèrent l'at-least-once avec un traitement idempotent, plus simple et plus performant.

*Question à poser* : « Que se passe-t-il si ce message est traité deux fois ? » Si la réponse est « rien de grave » ou « le traitement est idempotent », l'at-least-once suffit.

### Patterns d'Implémentation Éprouvés

Certains patterns ont fait leurs preuves dans les implémentations Kafka à grande échelle.

**Pattern : Topic par type d'événement.** Créer un topic par type d'événement métier (`orders.created`, `orders.shipped`, `payments.received`) plutôt qu'un topic fourre-tout (`all-events`).

*Avantages* :
- Les consommateurs s'abonnent uniquement aux événements pertinents
- Le schéma de chaque topic est homogène
- La rétention peut être configurée par type d'événement
- Le monitoring est plus granulaire

*Inconvénient* : Plus de topics à gérer (atténué par une bonne gouvernance).

**Pattern : Envelope avec métadonnées.** Envelopper chaque message avec des métadonnées standardisées.

```json
{
  "metadata": {
    "event_id": "uuid-...",
    "event_type": "OrderCreated",
    "event_time": "2024-01-15T10:30:00Z",
    "source_system": "order-service",
    "correlation_id": "uuid-...",
    "schema_version": "1.2"
  },
  "payload": {
    "order_id": "12345",
    "customer_id": "67890",
    "total_amount": 150.00
  }
}
```

*Avantages* :
- Traçabilité des messages (correlation_id)
- Déduplication possible (event_id)
- Routage basé sur les métadonnées
- Audit facilité

**Pattern : Outbox transactionnel.** Pour garantir la cohérence entre une mise à jour de base de données et la publication d'un événement, utiliser le pattern Outbox.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        Pattern Outbox                                   │
│                                                                         │
│  ┌─────────────┐    ┌─────────────────┐    ┌─────────────────────────┐ │
│  │  Service    │    │   Base de       │    │   Outbox Relay          │ │
│  │  Applicatif │───▶│   Données       │───▶│   (CDC ou Polling)      │ │
│  └─────────────┘    │  ┌───────────┐  │    └───────────┬─────────────┘ │
│                     │  │ Table     │  │                │               │
│       Transaction   │  │ Métier    │  │                │               │
│       atomique      │  ├───────────┤  │                ▼               │
│                     │  │ Table     │  │    ┌─────────────────────────┐ │
│                     │  │ Outbox    │  │    │         Kafka           │ │
│                     │  └───────────┘  │    │         Topic           │ │
│                     └─────────────────┘    └─────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

*Fonctionnement* :
1. La transaction applicative écrit dans la table métier ET dans la table Outbox atomiquement
2. Un processus séparé (CDC avec Debezium, ou polling) lit la table Outbox et publie dans Kafka
3. Après publication confirmée, l'entrée Outbox est marquée comme traitée ou supprimée

*Avantage* : Garantie de cohérence entre l'état de la base et les événements publiés.

**Pattern : Consumer avec état local.** Pour les traitements nécessitant des agrégations ou des jointures, maintenir un état local dans le consommateur (via Kafka Streams ou une base embarquée).

*Avantages* :
- Pas de dépendance à une base externe pour l'état
- L'état est reconstruit automatiquement depuis Kafka en cas de perte
- Performance optimale (état en mémoire ou sur disque local)

*Inconvénient* : Complexité accrue, temps de reconstruction au démarrage.

### Migration vers Kafka

La migration d'un système existant vers Kafka nécessite une stratégie progressive qui minimise les risques et permet un rollback à chaque étape. Les migrations « big bang » sont à éviter absolument.

**Phase 1 : Double-écriture (Dual Write).** Le système existant continue de fonctionner normalement, mais commence également à publier des événements dans Kafka. Cette phase ne modifie pas les consommateurs — ils continuent d'utiliser l'ancien système. L'objectif est de valider que les données arrivent correctement dans Kafka.

*Durée typique* : 1-2 semaines.

*Activités* : Déployer les producteurs Kafka, monitorer les erreurs, vérifier la correspondance entre les données de l'ancien système et Kafka.

**Phase 2 : Consommation shadow.** Les nouveaux consommateurs Kafka sont déployés en mode « shadow » — ils lisent et traitent les messages mais ne produisent pas d'effets de bord (pas d'écriture en base, pas d'envoi d'emails). L'objectif est de valider la logique de traitement.

*Durée typique* : 2-4 semaines.

*Activités* : Comparer les résultats du système existant et des consommateurs Kafka shadow. Identifier et corriger les divergences (ordres de traitement différents, erreurs de désérialisation, cas limites non gérés).

**Phase 3 : Bascule progressive avec traffic splitting.** Migrer le trafic progressivement de l'ancien système vers Kafka. Commencer avec un pourcentage faible (5-10%) et augmenter graduellement. L'ancien système peut être maintenu en lecture seule comme fallback.

*Durée typique* : 2-6 semaines selon la criticité.

*Exemple de progression* : 5% → 10% → 25% → 50% → 75% → 100%

*Critères de progression* : Aucune erreur pendant 24-48h, métriques de performance dans les limites acceptables, validation business des résultats.

**Phase 4 : Décommissionnement.** Une fois tous les consommateurs migrés et la stabilité confirmée (typiquement 2-4 semaines à 100%), décommissionner l'ancien système. Conserver les logs et la documentation pour référence.

*Activités* : Arrêter les anciens consommateurs, supprimer la double-écriture côté producteur, nettoyer l'infrastructure legacy.

> **Note de terrain**
>
> *Contexte* : Migration d'un système de notification par email basé sur une file RabbitMQ vers Kafka pour une entreprise de 10 000 employés.
>
> *Approche détaillée* :
>
> *Semaine 1-2* : Double-écriture dans RabbitMQ et Kafka. Consommateur Kafka en shadow (compte les messages, ne les traite pas). Validation : 100 000 messages reçus des deux côtés.
>
> *Semaine 3* : Consommateur Kafka traite réellement mais envoie vers une boîte email de test. Comparaison manuelle avec les envois RabbitMQ réels. Découverte d'un bug de formatage HTML corrigé.
>
> *Semaine 4* : 10% du trafic vers le consommateur Kafka (round-robin au niveau du routeur), 90% vers RabbitMQ. Monitoring intensif.
>
> *Semaine 5* : 25% vers Kafka. Un incident mineur (timeout Schema Registry) identifié et corrigé.
>
> *Semaine 6* : 50% vers Kafka. Performance validée sous charge réelle.
>
> *Semaine 7* : 100% vers Kafka. RabbitMQ maintenu en standby.
>
> *Semaine 8* : Arrêt de la double-écriture et du consommateur RabbitMQ. Migration complète.
>
> *Résultat* : Migration sans interruption de service, rollback possible à chaque étape.
>
> *Leçon* : Les migrations progressives réduisent le risque. Résister à la tentation du « big bang » même si la pression business est forte.

**Outils de validation pour la migration :**

*Comparaison de données* : Scripts qui comparent les sorties de l'ancien et du nouveau système (checksums, comptages, échantillonnage).

*Métriques de parité* : Dashboards montrant le lag entre les deux systèmes, les taux d'erreur relatifs, les différences de latence.

*Feature flags* : Permettent de basculer instantanément entre les systèmes sans redéploiement.

*Canary analysis* : Analyse automatisée comparant les métriques du groupe Kafka vs. le groupe legacy.

---

## III.5.3 Différences avec d'Autres Plateformes de Messagerie

Comprendre les différences fondamentales entre Kafka et les autres plateformes de messagerie est essentiel pour faire des choix architecturaux éclairés. Ces différences ne sont pas des détails d'implémentation mais des différences de modèle qui impactent profondément les patterns applicables.

### Kafka vs. Files de Messages Traditionnelles (RabbitMQ, ActiveMQ)

Les files de messages traditionnelles et Kafka résolvent des problèmes différents, bien qu'ils partagent une surface commune (envoi et réception de messages). Les confondre mène à des architectures sous-optimales.

**Modèle de consommation.** C'est la différence la plus fondamentale.

Les files traditionnelles utilisent un modèle « **competing consumers** » : plusieurs consommateurs se disputent les messages d'une file. Quand un consommateur prend un message, ce message est verrouillé. Après traitement et acknowledgment, le message est supprimé de la file. Si le consommateur échoue, le message retourne dans la file pour être pris par un autre consommateur. Ce modèle est idéal pour la distribution de tâches.

Kafka utilise un modèle « **consumer groups** » fondamentalement différent : chaque groupe de consommateurs reçoit tous les messages. Au sein d'un groupe, les partitions (pas les messages individuels) sont distribuées entre les consommateurs. Un message n'est pas « pris » par un consommateur — il reste dans le topic et peut être lu par d'autres groupes. L'avancement est tracké par l'offset, pas par l'acknowledgment individuel.

```
Files traditionnelles (RabbitMQ) :
┌─────────────────────────────────────────────────────────────────────────┐
│                         File unique                                     │
│  ┌───┬───┬───┬───┬───┬───┐                                             │
│  │ 6 │ 5 │ 4 │ 3 │ 2 │ 1 │ ◄─── Messages (supprimés après ACK)        │
│  └───┴───┴───┴───┴─┬─┴───┘                                             │
│                    │                                                    │
│          ┌─────────┼─────────┐                                         │
│          ▼         ▼         ▼                                         │
│    ┌──────────┐ ┌──────────┐ ┌──────────┐                              │
│    │ Worker 1 │ │ Worker 2 │ │ Worker 3 │  Competing consumers        │
│    └──────────┘ └──────────┘ └──────────┘                              │
└─────────────────────────────────────────────────────────────────────────┘

Kafka (Consumer Groups) :
┌─────────────────────────────────────────────────────────────────────────┐
│                     Topic (3 partitions)                                │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐                                 │
│  │ P0      │  │ P1      │  │ P2      │  Messages persistés             │
│  │ 1,2,3.. │  │ 1,2,3.. │  │ 1,2,3.. │  (rétention configurable)       │
│  └────┬────┘  └────┬────┘  └────┬────┘                                 │
│       │            │            │                                       │
│  Groupe A:        ▼            │                                       │
│  ┌──────────┐ ┌──────────┐     │                                       │
│  │Consumer1 │ │Consumer2 │◄────┘  (P0→C1, P1→C2, P2→C2)               │
│  │ (P0)     │ │ (P1,P2)  │                                             │
│  └──────────┘ └──────────┘                                             │
│                                                                         │
│  Groupe B (indépendant):                                               │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Tous les messages aussi disponibles pour le Groupe B             │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

**Persistance des messages.** Les files traditionnelles sont conçues pour la livraison transitoire — les messages sont supprimés après acknowledgment réussi. La file est un buffer temporaire entre producteur et consommateur. Kafka persiste les messages pour une durée configurable, permettant le replay, la consommation multiple, et l'Event Sourcing.

**Garantie d'ordre.** Les files traditionnelles ne garantissent généralement pas l'ordre au-delà d'une file unique. Avec plusieurs consommateurs en compétition, l'ordre de traitement est imprévisible. RabbitMQ offre quelques options (single active consumer) mais ce n'est pas son modèle principal.

Kafka garantit l'ordre strict au sein de chaque partition. Les messages avec la même clé vont dans la même partition et sont donc traités dans l'ordre par un seul consommateur.

**Routage des messages.** Les files traditionnelles offrent un routage sophistiqué. RabbitMQ avec ses exchanges (direct, topic, fanout, headers) permet de router les messages vers différentes files basé sur des patterns de routing keys, des en-têtes, ou du broadcast.

Kafka a un routage simple basé sur les topics — un message va dans un topic spécifique, et le routage avancé est la responsabilité du consommateur (filtrer les messages non pertinents) ou du producteur (choisir le bon topic).

**Protocole de communication.** RabbitMQ implémente AMQP (Advanced Message Queuing Protocol), un standard ouvert avec des implémentations multiples. ActiveMQ supporte JMS, AMQP, STOMP, et d'autres protocoles. Kafka utilise son propre protocole binaire, optimisé pour le débit et non standardisé (bien que documenté).

| Caractéristique | Kafka | RabbitMQ/ActiveMQ |
|-----------------|-------|-------------------|
| Modèle | Log distribué, consumer groups | File de messages, competing consumers |
| Persistance | Durable par défaut (jours/semaines) | Transitoire par défaut (jusqu'à ACK) |
| Replay | Natif (seek par offset/timestamp) | Non supporté (message supprimé après ACK) |
| Ordre | Strict par partition | Par file unique (limité avec scaling) |
| Débit | Très élevé (100k+ msg/s par broker) | Modéré (10k-50k msg/s) |
| Routage | Simple (topics) | Avancé (exchanges, bindings) |
| Complexité opérationnelle | Élevée (cluster, réplication) | Modérée |
| Cas d'usage principal | Streaming, Event Sourcing, intégration | Work queues, RPC asynchrone |

> **Décision architecturale**
>
> *Contexte* : Architecture microservices avec deux besoins de communication distincts :
> 1. Distribution de tâches asynchrones à des workers (traitement d'images uploadées)
> 2. Propagation d'événements métier à plusieurs services (commande créée → notification, inventory, shipping, analytics)
>
> *Analyse* :
> - Besoin 1 : Competing consumers naturel, pas besoin de replay (l'image est traitée une fois), ordre non critique (peu importe quelle image est traitée en premier), volume modéré → **File traditionnelle appropriée**
> - Besoin 2 : Multiple consumers indépendants (chaque service a besoin de tous les événements), replay utile (reconstruction d'un service défaillant), ordre par commande important → **Kafka approprié**
>
> *Décision* : Architecture hybride — RabbitMQ pour les work queues de traitement, Kafka pour les événements métier.
>
> *Alternative considérée* : Tout sur Kafka. Rejetée car over-engineering pour le besoin 1, et l'équipe maîtrise déjà RabbitMQ.
>
> *Révision prévue* : Si le volume de traitement d'images dépasse 10 000/heure ou si le besoin de tracking/audit émerge, reconsidérer Kafka.

### Kafka vs. Services de Streaming Cloud (Kinesis, Event Hubs, Pub/Sub)

Les hyperscalers offrent des services de streaming managés qui partagent des concepts avec Kafka mais diffèrent dans l'implémentation, le modèle de facturation, et les compromis.

**Amazon Kinesis Data Streams.** Service AWS de streaming. Les shards sont similaires aux partitions Kafka — unités de parallélisme avec ordre garanti. Différences clés :
- *Scaling* : Par shard, chaque shard a une capacité fixe (1 MB/s en écriture, 2 MB/s en lecture). L'ajout de shards est manuel et peut impliquer un resharding.
- *Rétention* : Maximum 365 jours (vs. illimitée pour Kafka).
- *Écosystème* : Intégration native avec Lambda, Firehose, Analytics. Pas de Kafka Connect.
- *Protocole* : API propriétaire AWS, pas compatible Kafka.
- *Coût* : Par shard-heure + par PUT payload unit. Peut devenir coûteux à grande échelle.

**Azure Event Hubs.** Service Azure compatible avec le protocole Kafka (Event Hubs for Kafka). Cela permet d'utiliser les clients Kafka existants avec Event Hubs comme backend.
- *Compatibilité Kafka* : Les applications Kafka peuvent se connecter à Event Hubs avec changements de configuration minimaux.
- *Scaling* : Par unités de débit (throughput units), plus abstrait que les partitions.
- *Rétention* : Maximum 7 jours (90 jours avec Event Hubs Dedicated).
- *Capture* : Intégration native vers Azure Blob Storage et Data Lake pour archivage automatique.
- *Écosystème* : Azure Functions, Stream Analytics, intégration Fabric.

**Google Cloud Pub/Sub.** Service GCP de messaging avec un modèle conceptuellement différent.
- *Modèle* : Messages individuels avec acknowledgment, pas de partitions ni d'offsets. Plus proche d'une file traditionnelle avec fan-out.
- *Ordre* : Garanti uniquement avec ordering keys (équivalent approximatif des clés de partition).
- *Subscriptions* : Pull (le consommateur demande) ou Push (Pub/Sub envoie vers un endpoint HTTP).
- *Rétention* : Maximum 7 jours.
- *Scaling* : Automatique et transparent, pas de concept de partitions à gérer.
- *Écosystème* : Dataflow (Apache Beam), BigQuery, intégration native GCP.

| Caractéristique | Kafka | Kinesis | Event Hubs | Pub/Sub |
|-----------------|-------|---------|------------|---------|
| Modèle | Partitions/Offsets | Shards/Sequence | Partitions/Offsets | Messages/Acks |
| Rétention max | Illimitée | 365 jours | 7-90 jours | 7 jours |
| Compatibilité Kafka | Native | Non | Oui (protocol) | Non |
| Scaling | Manuel (partitions) | Manuel (shards) | Semi-auto (TUs) | Auto |
| Écosystème | Connect, Streams, ksqlDB | Lambda, Firehose | Functions, Stream Analytics | Dataflow, BigQuery |
| Vendor lock-in | Faible (open source) | AWS | Azure | GCP |
| Opérationnel | Élevé (si auto-géré) | Faible | Faible | Très faible |

**Critères de choix entre Kafka et services cloud :**

*Choisir Kafka (auto-géré ou Confluent Cloud)* quand :
- La portabilité multi-cloud est importante (éviter le lock-in)
- L'écosystème Kafka (Connect avec 200+ connecteurs, Streams, ksqlDB) est nécessaire
- La rétention longue ou illimitée est requise (compliance, Event Sourcing)
- L'équipe a l'expertise Kafka ou souhaite la développer
- Le coût à grande échelle doit être optimisé (Kafka auto-géré peut être moins cher)

*Choisir le service cloud natif* quand :
- L'organisation est mono-cloud et souhaite minimiser l'opérationnel
- L'intégration avec l'écosystème cloud est prioritaire (Lambda + Kinesis, Functions + Event Hubs)
- Le budget favorise l'opex (pay-per-use) vs. le capex (infrastructure)
- L'équipe est petite et ne peut pas se permettre l'expertise Kafka
- Le démarrage rapide est prioritaire sur l'optimisation long terme

### Kafka vs. Systèmes de Streaming (Flink, Spark Streaming)

Apache Flink et Spark Streaming sont des **moteurs de traitement de flux**, pas des systèmes de stockage de messages. Ils sont complémentaires à Kafka, pas concurrents.

**Positionnement.** Kafka est une plateforme de stockage et de transport d'événements. Flink/Spark sont des moteurs de traitement qui peuvent lire depuis Kafka, transformer les données, et écrire vers Kafka ou d'autres destinations.

**Kafka Streams vs. Apache Flink.** La comparaison pertinente est entre Kafka Streams (la bibliothèque de traitement de flux intégrée à Kafka) et Flink.

*Kafka Streams* est une bibliothèque Java légère pour le traitement de flux. Elle n'a pas de cluster séparé — l'application est déployée comme un service Java standard (conteneur, VM). L'état est stocké localement (RocksDB) et sauvegardé dans Kafka pour la récupération. Idéal pour les traitements modérément complexes sans infrastructure supplémentaire.

*Apache Flink* est un moteur de traitement distribué avec son propre cluster (JobManager, TaskManagers). Il offre des capacités plus avancées : CEP (Complex Event Processing), fenêtrage sophistiqué, exactement-once garanti vers des sinks externes, support SQL riche. Plus puissant mais plus complexe à opérer.

| Caractéristique | Kafka Streams | Apache Flink |
|-----------------|---------------|--------------|
| Déploiement | Bibliothèque Java (pas de cluster) | Cluster dédié |
| Complexité opérationnelle | Faible | Élevée |
| Capacités de traitement | Modérées | Avancées (CEP, ML) |
| État | RocksDB local + changelog Kafka | Checkpointing distribué |
| Exactly-once | Vers Kafka uniquement | Vers sinks externes |
| Cas d'usage | Enrichissement, agrégations simples | ETL complexe, CEP, ML temps réel |

**Architecture typique avec Flink :**

```
┌─────────────┐    ┌─────────────────────────────┐    ┌─────────────┐
│   Kafka     │───▶│      Apache Flink           │───▶│   Kafka     │
│   (Source)  │    │  - Transformation           │    │   (Sink)    │
│   topics    │    │  - Enrichissement           │    │   topics    │
└─────────────┘    │  - Agrégation               │    └─────────────┘
                   │  - Détection de patterns    │           │
                   └─────────────────────────────┘           │
                                                             ▼
                                                    ┌─────────────────┐
                                                    │  Data Lake      │
                                                    │  (Iceberg)      │
                                                    └─────────────────┘
```

---

## III.5.4 Alternatives à Kafka

L'architecte doit connaître les alternatives à Kafka pour recommander la technologie appropriée à chaque cas d'usage. Cette section présente les principales alternatives avec leurs forces, limites, et cas d'usage optimaux.

### Pour les Files de Travail : RabbitMQ

RabbitMQ est le leader des files de messages traditionnelles. Mature, bien documenté, et largement adopté, il est souvent le choix par défaut pour la communication asynchrone.

**Quand choisir RabbitMQ plutôt que Kafka :**
- Distribution de tâches à des workers (competing consumers)
- Routage complexe basé sur des attributs de message (routing keys, headers)
- Protocole standard (AMQP) requis pour l'interopérabilité
- Équipe familière avec les files traditionnelles et sans expertise Kafka
- Volume modéré (< 100 000 messages/seconde)
- Besoin de priorités de messages (traitement urgent avant normal)
- TTL par message (expiration automatique)

**Forces de RabbitMQ :**

*Modèle de routage puissant.* Les exchanges (direct, topic, fanout, headers) permettent un routage déclaratif sophistiqué sans code applicatif. Un message peut être routé vers différentes files basé sur des patterns de routing keys.

*Priorités de messages natives.* RabbitMQ supporte les priorités de messages (0-255), permettant aux messages urgents d'être traités avant les messages normaux.

*Dead letter queues intégrées.* Quand un message est rejeté, expire, ou ne peut être routé, il peut être automatiquement envoyé vers une DLQ configurée de manière déclarative.

*TTL par message.* Chaque message peut avoir son propre TTL (time-to-live). Les messages expirés sont automatiquement supprimés ou routés vers la DLQ.

*Plugins riches.* Management UI pour la supervision, federation pour la distribution géographique, shovel pour le déplacement de messages entre clusters, delayed message exchange pour les messages différés.

*Communauté active et mature.* Plus de 15 ans d'existence, documentation extensive, nombreuses ressources d'apprentissage.

**Limites par rapport à Kafka :**

*Pas de replay natif.* Une fois qu'un message est acknowledgé, il est supprimé. Impossible de relire les messages passés.

*Débit inférieur à grande échelle.* RabbitMQ atteint typiquement 10 000-50 000 messages/seconde, vs. 100 000+ pour Kafka.

*Ordre garanti limité.* Avec plusieurs consommateurs en compétition, l'ordre de traitement n'est pas garanti. Le mode « single active consumer » existe mais limite le parallélisme.

*Pas de log persistant.* RabbitMQ n'est pas conçu pour l'Event Sourcing ou le stockage long terme d'événements.

```java
// Exemple RabbitMQ : Publication avec routage sophistiqué
// Exchange topic avec routing keys hiérarchiques
channel.exchangeDeclare("orders-exchange", "topic", true);

// Publication vers orders.created.europe.france
channel.basicPublish(
    "orders-exchange",           // Exchange
    "orders.created.europe.fr",  // Routing key
    null,                        // Properties
    messageBytes                 // Body
);

// Consommateur avec binding pattern
// Reçoit tous les événements de création en Europe
channel.queueDeclare("eu-orders-queue", true, false, false, null);
channel.queueBind(
    "eu-orders-queue",           // Queue
    "orders-exchange",           // Exchange
    "orders.created.europe.*"    // Binding pattern (wildcard)
);

// Consommation avec acknowledgment manuel
channel.basicConsume("eu-orders-queue", false, (consumerTag, delivery) -> {
    try {
        processOrder(delivery.getBody());
        channel.basicAck(delivery.getEnvelope().getDeliveryTag(), false);
    } catch (Exception e) {
        // Rejet avec requeue=false → message va en DLQ si configurée
        channel.basicReject(delivery.getEnvelope().getDeliveryTag(), false);
    }
}, consumerTag -> {});
```

> **Note de terrain**
>
> *Contexte* : Startup e-commerce avec 10 développeurs. Besoin de traitement asynchrone de commandes (génération de factures, emails, stock).
>
> *Choix initial* : Kafka, car « c'est ce que les grandes entreprises utilisent ».
>
> *Problèmes rencontrés* :
> - Complexité opérationnelle excessive pour l'équipe
> - Temps de débogage élevé (les développeurs ne maîtrisaient pas Kafka)
> - Coût d'infrastructure non justifié pour 1 000 commandes/jour
>
> *Migration vers RabbitMQ* :
> - Setup en 2 heures vs. 2 semaines pour Kafka
> - Équipe productive immédiatement (RabbitMQ plus intuitif)
> - Coût d'infrastructure divisé par 5
> - Le routage par routing keys simplifie l'architecture
>
> *Leçon* : Choisir la technologie appropriée au contexte, pas la plus « hype ».

### Pour les Volumes Faibles : PostgreSQL avec LISTEN/NOTIFY ou Tables de Messages

Pour les faibles volumes de messages, une base de données relationnelle existante peut suffire, évitant l'ajout d'une infrastructure supplémentaire.

**Quand choisir PostgreSQL plutôt que Kafka :**
- Volume très faible (< 1 000 messages/jour)
- Infrastructure existante PostgreSQL
- Pas de besoin de replay ou de rétention longue
- Transactions ACID avec le reste de l'application critiques
- Équipe sans compétences messaging
- Budget infrastructure limité

**Implémentation simple avec table de messages :**

```sql
-- Schéma de la table de messages
CREATE TABLE message_queue (
    id SERIAL PRIMARY KEY,
    topic VARCHAR(100) NOT NULL,
    key VARCHAR(255),
    payload JSONB NOT NULL,
    headers JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,
    status VARCHAR(20) DEFAULT 'pending',
    retry_count INT DEFAULT 0,
    error_message TEXT
);

-- Index pour les requêtes fréquentes
CREATE INDEX idx_queue_pending ON message_queue(topic, status, created_at) 
    WHERE status = 'pending';
CREATE INDEX idx_queue_key ON message_queue(key) WHERE key IS NOT NULL;

-- Publication d'un message
INSERT INTO message_queue (topic, key, payload, headers) 
VALUES (
    'orders.created', 
    '12345',  -- order_id comme clé
    '{"order_id": "12345", "customer": "John", "total": 150.00}',
    '{"source": "checkout-service"}'
);

-- Consommation avec verrouillage (FOR UPDATE SKIP LOCKED évite les deadlocks)
WITH next_message AS (
    SELECT id FROM message_queue
    WHERE topic = 'orders.created' AND status = 'pending'
    ORDER BY created_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE message_queue
SET status = 'processing', processed_at = NOW()
WHERE id IN (SELECT id FROM next_message)
RETURNING *;

-- Après traitement réussi
UPDATE message_queue SET status = 'completed' WHERE id = ?;

-- Après échec (avec retry)
UPDATE message_queue 
SET status = 'pending', retry_count = retry_count + 1, error_message = ?
WHERE id = ? AND retry_count < 3;

-- Après échecs répétés → dead letter
UPDATE message_queue SET status = 'dead_letter', error_message = ? WHERE id = ?;

-- Nettoyage périodique des messages traités (optionnel)
DELETE FROM message_queue 
WHERE status = 'completed' AND processed_at < NOW() - INTERVAL '7 days';
```

**PostgreSQL LISTEN/NOTIFY pour les notifications temps réel :**

```sql
-- Trigger pour notifier lors de l'insertion
CREATE OR REPLACE FUNCTION notify_new_message()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify('new_message', json_build_object(
        'id', NEW.id,
        'topic', NEW.topic,
        'key', NEW.key
    )::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER message_inserted
AFTER INSERT ON message_queue
FOR EACH ROW EXECUTE FUNCTION notify_new_message();
```

```python
# Consommateur Python avec LISTEN/NOTIFY
import psycopg2
import select

conn = psycopg2.connect("dbname=myapp")
conn.set_isolation_level(psycopg2.extensions.ISOLATION_LEVEL_AUTOCOMMIT)
cursor = conn.cursor()
cursor.execute("LISTEN new_message;")

while True:
    if select.select([conn], [], [], 5) == ([], [], []):
        # Timeout - vérifier quand même s'il y a des messages
        pass
    else:
        conn.poll()
        while conn.notifies:
            notify = conn.notifies.pop(0)
            message_info = json.loads(notify.payload)
            process_message(message_info['id'])
```

**Avantages de l'approche PostgreSQL :**

*Pas d'infrastructure supplémentaire.* La base existe déjà, pas de cluster à gérer.

*Transactions ACID avec les données métier.* L'insertion du message et la mise à jour des données métier peuvent être dans la même transaction.

*SQL familier.* Pas de nouvelle technologie à apprendre pour l'équipe.

*Monitoring et backup existants.* Les outils de supervision et sauvegarde de la base couvrent automatiquement les messages.

*Requêtes ad-hoc.* Possible de requêter l'historique des messages avec SQL standard.

**Limites :**

*Ne scale pas.* PostgreSQL atteint ses limites à quelques milliers de messages/seconde.

*Pas de partitionnement natif.* Le parallélisme dépend du nombre de workers, pas de partitions.

*Polling ou LISTEN/NOTIFY limité.* LISTEN/NOTIFY ne garantit pas la livraison si le listener est temporairement déconnecté.

*Pas d'écosystème.* Pas de connecteurs, pas de stream processing intégré.

### Pour le Cloud Natif : Services Managés (Kinesis, Event Hubs, Pub/Sub)

**Quand choisir un service cloud managé :**
- Organisation mono-cloud avec stratégie cloud-first établie
- Équipe réduite sans capacité d'opérer un cluster Kafka
- Intégration étroite avec l'écosystème cloud souhaitée
- Budget favorisant l'opex (pay-per-use)
- Time-to-market prioritaire sur l'optimisation long terme

**Amazon Kinesis Data Streams — Exemple d'utilisation :**

```python
import boto3
import json

# Configuration
kinesis_client = boto3.client('kinesis', region_name='us-east-1')
stream_name = 'orders-stream'

# Publication d'un événement
def publish_order(order):
    response = kinesis_client.put_record(
        StreamName=stream_name,
        Data=json.dumps(order),
        PartitionKey=str(order['order_id'])  # Équivalent de la clé Kafka
    )
    return response['SequenceNumber']

# Consommation avec Lambda (serverless)
def lambda_handler(event, context):
    for record in event['Records']:
        # Kinesis encode en base64
        payload = json.loads(base64.b64decode(record['kinesis']['data']))
        
        # Traitement
        process_order(payload)
        
        # Pas d'acknowledgment explicite - Lambda gère automatiquement
    
    return {'statusCode': 200}
```

**Azure Event Hubs avec compatibilité Kafka :**

L'avantage d'Event Hubs est la compatibilité avec le protocole Kafka. Une application Kafka existante peut se connecter à Event Hubs avec des changements de configuration minimaux.

```java
// Configuration Kafka vers Event Hubs
Properties props = new Properties();
props.put("bootstrap.servers", "namespace.servicebus.windows.net:9093");
props.put("security.protocol", "SASL_SSL");
props.put("sasl.mechanism", "PLAIN");
props.put("sasl.jaas.config", 
    "org.apache.kafka.common.security.plain.PlainLoginModule required " +
    "username=\"$ConnectionString\" " +
    "password=\"Endpoint=sb://namespace.servicebus.windows.net/;" +
    "SharedAccessKeyName=RootManageSharedAccessKey;" +
    "SharedAccessKey=xxxxx\";");

// Le reste du code Kafka standard fonctionne tel quel
props.put("key.serializer", StringSerializer.class.getName());
props.put("value.serializer", StringSerializer.class.getName());

KafkaProducer<String, String> producer = new KafkaProducer<>(props);
producer.send(new ProducerRecord<>("orders-topic", orderId, orderJson));
```

Cette compatibilité permet une migration progressive : développer avec Kafka localement, déployer sur Event Hubs en production, et migrer vers Kafka auto-géré ou Confluent Cloud si nécessaire plus tard.

**Google Cloud Pub/Sub — Modèle différent :**

Pub/Sub a un modèle conceptuellement différent, plus proche des files traditionnelles avec fan-out.

```python
from google.cloud import pubsub_v1

# Publication
publisher = pubsub_v1.PublisherClient()
topic_path = publisher.topic_path('my-project', 'orders-topic')

def publish_order(order):
    data = json.dumps(order).encode('utf-8')
    future = publisher.publish(
        topic_path, 
        data,
        ordering_key=str(order['order_id']),  # Pour garantir l'ordre par commande
        source='checkout-service'  # Attributs personnalisés
    )
    return future.result()

# Consommation avec pull
subscriber = pubsub_v1.SubscriberClient()
subscription_path = subscriber.subscription_path('my-project', 'orders-subscription')

def callback(message):
    order = json.loads(message.data.decode('utf-8'))
    try:
        process_order(order)
        message.ack()  # Acknowledgment explicite
    except Exception as e:
        message.nack()  # Message sera re-délivré

# Souscription asynchrone
streaming_pull_future = subscriber.subscribe(subscription_path, callback=callback)
```

### Pour le Temps Réel Extrême : Redis Streams

Redis Streams (disponible depuis Redis 5.0) offre des capacités de streaming avec la latence extrêmement faible caractéristique de Redis.

**Quand choisir Redis Streams :**
- Latence sub-milliseconde requise (trading haute fréquence, gaming temps réel)
- Volume modéré (< 1 million messages/seconde)
- Redis déjà présent dans l'architecture
- Simplicité opérationnelle prioritaire
- Rétention courte acceptable (limité par la mémoire)

**Forces de Redis Streams :**

*Latence extrêmement faible.* Redis opère en mémoire, offrant des latences de l'ordre de la microseconde.

*Modèle de consumer groups.* Similar à Kafka — plusieurs consumers dans un groupe se partagent les messages.

*Commandes simples.* XADD pour publier, XREAD/XREADGROUP pour consommer.

*Persistance optionnelle.* AOF (Append-Only File) et RDB (snapshots) pour la durabilité.

*Écosystème Redis.* S'intègre naturellement avec les autres structures Redis (caches, sessions, pub/sub).

```python
import redis

r = redis.Redis(host='localhost', port=6379, decode_responses=True)

# Publication
def publish_event(stream_name, event):
    # '*' = ID auto-généré (timestamp + sequence)
    message_id = r.xadd(stream_name, event, maxlen=100000)  # Limite la taille
    return message_id

# Exemple
publish_event('orders-stream', {
    'order_id': '12345',
    'customer': 'John',
    'total': '150.00',
    'timestamp': '2024-01-15T10:30:00Z'
})

# Création d'un consumer group (une seule fois)
try:
    r.xgroup_create('orders-stream', 'order-processors', id='0', mkstream=True)
except redis.ResponseError:
    pass  # Groupe existe déjà

# Consommation avec consumer group
def consume_events():
    while True:
        # '>' = nouveaux messages uniquement
        messages = r.xreadgroup(
            groupname='order-processors',
            consumername='processor-1',
            streams={'orders-stream': '>'},
            count=10,
            block=5000  # Block 5 secondes si pas de messages
        )
        
        for stream, stream_messages in messages:
            for message_id, fields in stream_messages:
                try:
                    process_order(fields)
                    # Acknowledgment après traitement réussi
                    r.xack('orders-stream', 'order-processors', message_id)
                except Exception as e:
                    # Le message sera réclamé par un autre consumer après timeout
                    log.error(f"Error processing {message_id}: {e}")

# Réclamation des messages non-acquittés (pour les consumers qui ont crashé)
def claim_pending_messages():
    # Récupérer les messages pending depuis plus de 30 secondes
    pending = r.xpending_range('orders-stream', 'order-processors', 
                               min='-', max='+', count=10)
    
    for entry in pending:
        if entry['time_since_delivered'] > 30000:  # 30 secondes
            # Réclamer le message
            claimed = r.xclaim('orders-stream', 'order-processors', 'processor-1',
                              min_idle_time=30000, message_ids=[entry['message_id']])
            for msg_id, fields in claimed:
                process_order(fields)
                r.xack('orders-stream', 'order-processors', msg_id)
```

**Limites de Redis Streams :**

*Scalabilité limitée.* Redis Cluster distribue les streams sur différents nœuds, mais chaque stream est sur un seul nœud. Pas de partitionnement au sein d'un stream.

*Rétention limitée par la mémoire.* Contrairement à Kafka qui utilise le disque, Redis est principalement en mémoire. MAXLEN ou MINID limitent la taille mais peuvent causer des pertes si mal configurés.

*Moins de garanties de durabilité.* La persistance Redis (AOF/RDB) n'est pas aussi robuste que la réplication Kafka. Une panne peut causer la perte des dernières écritures.

*Écosystème moins riche.* Pas d'équivalent à Kafka Connect ou Kafka Streams.

### Pour l'Event Sourcing Léger : EventStoreDB

EventStoreDB est une base de données spécialement conçue pour l'Event Sourcing, offrant des fonctionnalités que Kafka n'a pas nativement.

**Quand choisir EventStoreDB :**
- Event Sourcing est le pattern principal de l'application
- Projections complexes nécessaires (agrégations, transformations côté serveur)
- Volume modéré (< 100 000 événements/seconde)
- Équipe prête à adopter un produit spécialisé
- Besoin de streams par agrégat avec garanties de version

**Forces d'EventStoreDB :**

*Conçu pour l'Event Sourcing.* Les streams par agrégat, la gestion des versions, et l'optimistic concurrency sont natifs.

*Projections intégrées.* Les projections JavaScript permettent de créer des vues dérivées côté serveur, sans consommateur externe.

*Subscriptions catch-up et persistantes.* Consommation des événements avec reprise automatique après déconnexion.

*Global stream.* Vue de tous les événements de tous les streams pour les projections cross-agrégats.

```javascript
// Projection EventStoreDB en JavaScript
// Compte le nombre de commandes par client
fromStream('orders')
    .when({
        $init: function() {
            return { ordersPerCustomer: {} };
        },
        OrderCreated: function(state, event) {
            var customerId = event.data.customerId;
            state.ordersPerCustomer[customerId] = 
                (state.ordersPerCustomer[customerId] || 0) + 1;
        }
    })
    .outputState();
```

**Limites d'EventStoreDB :**

*Écosystème plus petit.* Moins de connecteurs, moins de ressources d'apprentissage que Kafka.

*Moins adapté au streaming généraliste.* Optimisé pour l'Event Sourcing, moins pour le streaming de données ou l'intégration.

*Compétences spécifiques requises.* Modèle mental différent des bases de données traditionnelles ou de Kafka.

### Tableau Comparatif Complet des Alternatives

| Critère | Kafka | RabbitMQ | PostgreSQL | Kinesis | Redis Streams | EventStoreDB |
|---------|-------|----------|------------|---------|---------------|--------------|
| Débit max | Très élevé | Élevé | Faible | Élevé | Très élevé | Modéré |
| Latence typique | 5-50 ms | 1-10 ms | 10-100 ms | 5-50 ms | < 1 ms | 1-10 ms |
| Durabilité | Excellente | Bonne | Excellente | Bonne | Configurable | Excellente |
| Replay natif | Oui | Non | Non (manuel) | Limité | Limité | Oui |
| Complexité opérationnelle | Élevée | Modérée | Faible | Faible | Faible | Modérée |
| Event Sourcing | Bon | Non adapté | Manuel | Possible | Possible | Excellent |
| Écosystème | Très riche | Riche | SQL | AWS | Redis | Spécialisé |
| Coût infrastructure | Élevé | Modéré | Faible | Pay-per-use | Faible | Modéré |
| Courbe d'apprentissage | Raide | Modérée | Faible | Modérée | Faible | Modérée |

> **Décision architecturale**
>
> *Contexte* : Équipe de 5 développeurs construisant une application SaaS B2B avec les besoins suivants :
> - Notifications en temps réel aux utilisateurs (faible volume, ~100/heure)
> - Traitement asynchrone de fichiers uploadés (files de travail, ~500/jour)
> - Historique des actions utilisateur pour audit (Event Sourcing léger, ~10 000/jour)
>
> *Analyse des options* :
> - Kafka : Puissant mais over-engineering massif pour ces volumes et cette équipe
> - RabbitMQ : Bien pour les files de travail, moins pour l'historique permanent
> - PostgreSQL : Simple, unifié avec la base existante, suffisant pour les volumes
>
> *Décision* : PostgreSQL avec tables de messages. Une seule technologie à maîtriser, infrastructure existante, simplicité maximale.
>
> *Critères de révision* : Réévaluer si :
> - Volume > 100 000 messages/jour
> - Temps de traitement des files > 30 secondes
> - Besoin de replay sophistiqué avec filtres
> - Plusieurs équipes avec besoins de messaging indépendants

---

## III.5.5 Résumé

Ce chapitre a adopté une perspective pragmatique sur l'adoption de Kafka, dépassant les considérations purement techniques pour adresser les questions stratégiques que tout architecte doit se poser avant de recommander cette technologie.

### Quand Choisir Kafka — Les Critères Décisifs

Kafka excelle dans des scénarios spécifiques où ses caractéristiques architecturales apportent une valeur distinctive qui justifie sa complexité opérationnelle.

**Cas d'usage optimaux identifiés :**

*Event Sourcing et CQRS* : Le pattern où l'état est reconstruit depuis un journal d'événements immuables bénéficie directement des forces de Kafka — durabilité, ordre par partition, rétention configurable, et consommation multiple. Les projections CQRS peuvent être construites indépendamment depuis le même flux d'événements.

*Intégration de données en temps réel (CDC)* : Kafka Connect avec Debezium capture les changements de bases de données et les propage en secondes plutôt qu'en heures. Ce pattern transforme les architectures batch traditionnelles en architectures temps réel.

*Pipelines de streaming* : Transformation, enrichissement, et agrégation de flux continus avec Kafka Streams ou ksqlDB, permettant un traitement sophistiqué sans infrastructure supplémentaire.

*Communication inter-services à grande échelle* : Dans les architectures microservices complexes, Kafka fournit un backbone découplé temporellement et par connaissances, plus résilient que les appels synchrones point-à-point.

*Collecte haute vélocité* : Métriques, logs, et traces à des millions d'événements par seconde vers des systèmes d'analyse downstream.

*Machine Learning temps réel* : Feature engineering, scoring, et détection d'anomalies avec des latences de l'ordre de la seconde.

**Critères de sélection favorables quantifiés :**
- Volume > 10 000 messages/seconde justifiant l'infrastructure
- Besoin de replay fréquent pour reconstruction ou débogage
- Consommateurs multiples indépendants (> 3 groupes)
- Rétention > 7 jours pour audit, compliance, ou reconstruction
- Ordre des messages critique pour les invariants métier
- Durabilité non négociable (aucune perte acceptable)

**Cas où Kafka n'est pas optimal — Reconnaissance des limites :**

L'architecte doit également reconnaître les cas où Kafka apporte une complexité injustifiée :

*Files de travail simples* : Distribution de tâches sans ordre ni replay — RabbitMQ ou SQS suffisent.

*Request-response synchrone* : REST/gRPC sont plus simples et plus performants.

*Faibles volumes* : En dessous de 1 000 messages/minute, PostgreSQL avec tables de messages est souvent suffisant.

*Transactions ACID cross-système* : Kafka ne résout pas ce problème, des patterns comme Saga sont nécessaires.

La matrice de décision présentée dans ce chapitre guide l'évaluation multicritère nécessaire avant toute adoption, évitant les décisions basées sur la popularité plutôt que sur les besoins réels.

### Naviguer l'Implémentation — Au-delà de la Technique

Les défis d'implémentation dépassent largement les considérations techniques. L'expérience montre que les échecs de projets Kafka sont plus souvent organisationnels que technologiques.

**Défis organisationnels critiques :**

*Compétences et formation* : Kafka requiert un investissement significatif en formation. Les développeurs habitués aux appels REST synchrones doivent apprendre à penser en termes d'événements et de réactions asynchrones. Cette transition cognitive prend plusieurs mois.

*Gouvernance des topics* : Sans conventions claires (nommage, rétention, schémas, propriétaires), la prolifération de topics devient rapidement ingérable. L'établissement d'un centre d'excellence ou d'un processus de revue avant création de topic est fortement recommandé.

*Responsabilité des schémas* : L'évolution des schémas de messages nécessite une coordination entre producteurs et consommateurs. Le Schema Registry aide techniquement, mais les processus organisationnels (qui approuve un changement de schéma ?) doivent l'accompagner.

**Défis techniques récurrents documentés :**

*Choix du nombre de partitions* : Décision irréversible avec impact long terme. Trop peu limite le parallélisme futur ; trop beaucoup augmente la charge cluster et la latence de rééquilibrage. La règle empirique (débit cible / débit par consommateur × 2-3) est un point de départ, mais la révision lors de la planification de capacité est essentielle.

*Stratégie de partitionnement* : Le choix de la clé détermine la distribution et l'ordre. Une clé avec cardinalité insuffisante crée des hot partitions ; une clé mal alignée avec les besoins métier brise les invariants d'ordre.

*Exactly-once vs. at-least-once* : L'exactly-once ajoute latence et complexité. La question « que se passe-t-il si ce message est traité deux fois ? » permet souvent de découvrir que l'at-least-once avec idempotence est suffisant et plus simple.

**Patterns d'implémentation éprouvés :**

*Topic par type d'événement* : Clarté, schémas homogènes, rétention configurable, monitoring granulaire.

*Envelope avec métadonnées* : Traçabilité (correlation_id), déduplication (event_id), routage et audit facilités.

*Outbox transactionnel* : Cohérence garantie entre l'état de la base et les événements publiés.

*Migration progressive* : Double-écriture, validation shadow, bascule progressive, décommissionnement après stabilisation. Les migrations « big bang » sont à éviter.

### Différences Fondamentales avec les Autres Plateformes

La confusion entre Kafka et les files de messages traditionnelles mène à des architectures sous-optimales. Les différences ne sont pas des détails d'implémentation mais des différences de modèle fondamentales.

**Kafka vs. Files traditionnelles (RabbitMQ, ActiveMQ) :**

La différence la plus fondamentale est le modèle de consommation. Les files traditionnelles utilisent le « competing consumers » (messages distribués entre workers, supprimés après acknowledgment). Kafka utilise les « consumer groups » (chaque groupe reçoit tous les messages, partitions distribuées au sein du groupe, messages persistés).

Les conséquences architecturales sont profondes : Kafka permet le replay, la consommation multiple, l'Event Sourcing — impossibles avec les files traditionnelles. Mais Kafka est plus complexe pour la simple distribution de tâches.

**Kafka vs. Services cloud (Kinesis, Event Hubs, Pub/Sub) :**

Le choix dépend de la stratégie cloud de l'organisation. Kafka (auto-géré ou Confluent Cloud) offre la portabilité et l'écosystème le plus riche. Les services cloud offrent l'opérationnel simplifié et l'intégration native avec leur écosystème respectif.

Event Hubs mérite une mention spéciale pour sa compatibilité avec le protocole Kafka, permettant une migration progressive et une portabilité partielle.

**Kafka vs. Moteurs de traitement (Flink, Spark) :**

Ces technologies sont complémentaires, pas concurrentes. Kafka est la plateforme de stockage et transport ; Flink/Spark sont les moteurs de traitement. Kafka Streams offre une alternative plus légère pour les traitements modérément complexes.

### Alternatives à Considérer — Choisir l'Outil Approprié

L'architecte pragmatique connaît les alternatives et recommande la technologie appropriée à chaque contexte.

**RabbitMQ** : Pour les files de travail avec routage sophistiqué. Plus simple que Kafka pour le pattern competing consumers. Mature et bien compris.

**PostgreSQL avec tables de messages** : Pour les faibles volumes. Pas d'infrastructure supplémentaire, transactions ACID avec les données métier, SQL familier. Ne scale pas, mais souvent suffisant.

**Services cloud (Kinesis, Event Hubs, Pub/Sub)** : Pour les organisations mono-cloud souhaitant minimiser l'opérationnel. Pay-per-use, intégration native, mais lock-in et rétention limitée.

**Redis Streams** : Pour la latence sub-milliseconde avec volumes modérés. Redis en mémoire offre des performances extrêmes, mais avec des compromis sur la durabilité et la scalabilité.

**EventStoreDB** : Pour l'Event Sourcing comme pattern principal. Conçu spécifiquement pour ce cas d'usage avec des fonctionnalités natives (projections, streams par agrégat) que Kafka n'offre pas.

### Principes Directeurs pour l'Architecte

À retenir de ce chapitre :

**1. Évaluer avant d'adopter.** Kafka apporte de la valeur significative dans les bons contextes, mais aussi de la complexité significative dans tous les contextes. S'assurer que cette complexité est justifiée par les exigences réelles, mesurables.

**2. La gouvernance est aussi importante que la technologie.** Les projets Kafka échouent plus souvent pour des raisons organisationnelles (manque de gouvernance, compétences insuffisantes, processus absents) que pour des raisons techniques. Investir dans la gouvernance dès le début.

**3. Migration progressive.** Les migrations « big bang » vers Kafka sont risquées. La double-écriture, la validation shadow, et la bascule progressive permettent des transitions sûres avec rollback possible à chaque étape.

**4. Hybride est acceptable et souvent optimal.** Utiliser Kafka pour les cas à haute valeur (événements métier critiques, streaming temps réel, haute vélocité) et des solutions plus simples pour les cas triviaux (notifications ponctuelles, files de travail simples) est une approche pragmatique qui optimise le rapport valeur/complexité.

**5. Réévaluer régulièrement.** Les besoins évoluent, les volumes changent, les équipes grandissent. Une décision appropriée aujourd'hui (PostgreSQL pour 1 000 messages/jour) peut ne plus l'être dans deux ans (10 000 messages/minute). Définir des critères de révision explicites.

**6. Mesurer avant d'optimiser.** Les décisions d'architecture doivent être basées sur des mesures réelles (volume actuel et projeté, latence requise, coût de l'infrastructure) plutôt que sur des intuitions ou la popularité des technologies.

---

### Vers le Chapitre Suivant

Ce chapitre a exploré **quand** et **pourquoi** utiliser Kafka, ainsi que les alternatives disponibles. Le chapitre suivant, « Contrats de Données », approfondira **comment** structurer les messages échangés via Kafka pour garantir l'interopérabilité et l'évolutivité à long terme entre producteurs et consommateurs.

Les contrats de données sont le fondement de la gouvernance Kafka à grande échelle. Sans eux, même la meilleure architecture technique échouera face à la complexité organisationnelle.

---

*Volume III : Apache Kafka - Guide de l'Architecte*

*Chapitre III.5 — Cas d'Utilisation Kafka*

*Monographie « L'Entreprise Agentique »*
