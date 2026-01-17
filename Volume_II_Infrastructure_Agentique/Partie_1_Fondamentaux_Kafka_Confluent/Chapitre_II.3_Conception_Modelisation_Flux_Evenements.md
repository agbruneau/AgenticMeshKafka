# Chapitre II.3 — Conception et Modélisation du Flux d'Événements

## De la Vision Métier à l'Architecture Technique

---

La puissance d'une architecture orientée événements ne réside pas dans la sophistication de son infrastructure technique, mais dans la qualité de sa modélisation. Un backbone Kafka parfaitement configuré devient inutile si les événements qu'il transporte ne reflètent pas fidèlement les réalités métier de l'organisation. Ce chapitre explore les méthodologies et les pratiques qui transforment la connaissance du domaine en flux d'événements bien conçus, documentés et évolutifs — fondation indispensable pour les agents cognitifs qui consommeront ces flux.

L'entreprise agentique exige une rigueur particulière dans la conception des événements. Les agents cognitifs, contrairement aux applications traditionnelles, interprètent sémantiquement les données qu'ils reçoivent. Un événement mal nommé, une structure ambiguë, ou une stratégie de partitionnement inappropriée ne causent pas simplement des erreurs techniques — ils induisent des interprétations erronées et des décisions incorrectes de la part des agents. La modélisation des flux d'événements devient ainsi un acte de conception cognitive autant que technique.

Ce chapitre structure cette discipline en cinq dimensions complémentaires : la découverte collaborative du domaine par Event Storming, la classification rigoureuse des types d'événements, la conception technique des topics Kafka, les stratégies d'évolution des schémas, et la documentation formelle avec AsyncAPI. Ensemble, ces pratiques établissent le pont entre l'intention métier et l'implémentation technique.

---

## II.3.1 Modélisation des Domaines Métier (Event Storming)

### L'Atelier Collaboratif comme Point de Départ

Event Storming, créé par Alberto Brandolini, s'est imposé comme la méthodologie de référence pour explorer les domaines métier complexes avant de concevoir des architectures événementielles. Cette technique d'atelier collaboratif réunit les parties prenantes techniques et métier autour d'un mur couvert de post-its colorés, chaque couleur représentant un concept spécifique du domaine.

L'approche se distingue par sa rapidité et son inclusivité. Ce qui prenait traditionnellement des semaines de spécifications formelles s'accomplit en quelques heures d'atelier intensif. Les experts métier, développeurs, architectes et analystes participent sur un pied d'égalité, chacun apportant sa perspective unique. Cette diversité génère des insights que les approches cloisonnées ne peuvent révéler.

> **Définition formelle**  
> Event Storming est une technique de modélisation collaborative où les participants identifient les événements de domaine (ce qui se passe dans le métier), les commandes qui les déclenchent, les acteurs qui initient ces commandes, les agrégats qui encapsulent la logique métier, et les politiques qui orchestrent les réactions aux événements.

### Les Éléments Constitutifs de l'Atelier

La grammaire visuelle d'Event Storming utilise des post-its de couleurs distinctes pour représenter les différents concepts du domaine.

**Événements de domaine (orange)** : Les occurrences significatives dans le métier, formulées au passé. « CommandePassée », « PaiementValidé », « StockÉpuisé ». Ces événements constituent le point de départ et le cœur de l'exploration. Ils représentent les faits métier que le système doit capturer et propager.

**Commandes (bleu)** : Les intentions ou requêtes qui déclenchent les événements. « PasserCommande », « ValiderPaiement », « RéapprovisionnerStock ». Les commandes établissent le lien entre l'intention d'un acteur et le fait métier résultant.

**Acteurs (jaune)** : Les personnes, systèmes ou rôles qui émettent les commandes. « Client », « Agent de support », « Système de facturation ». L'identification des acteurs clarifie les responsabilités et les points d'entrée du système.

**Agrégats (jaune pâle)** : Les entités métier qui encapsulent la logique de traitement des commandes et produisent les événements. « Commande », « CompteBancaire », « InventaireProduit ». Les agrégats définissent les frontières de cohérence transactionnelle.

**Politiques (violet)** : Les règles métier qui réagissent aux événements et déclenchent d'autres commandes. « Quand PaiementValidé, alors ExpédierCommande ». Les politiques révèlent les chaînes de causalité métier.

**Systèmes externes (rose)** : Les services tiers ou legacy avec lesquels le domaine interagit. « Passerelle de paiement », « ERP », « Service de notification ». Ces systèmes définissent les frontières du domaine modélisé.

### Déroulement d'une Session Event Storming

Une session Event Storming typique se déroule en phases progressives, chacune enrichissant le modèle.

**Phase 1 — Chaotic Exploration** : Les participants génèrent librement tous les événements de domaine qui leur viennent à l'esprit, sans ordre ni structure. Cette phase exploite l'intelligence collective et fait émerger des événements que personne n'aurait identifiés seul. L'objectif est la quantité, pas la qualité.

**Phase 2 — Timeline Ordering** : Les événements sont ordonnés chronologiquement sur le mur, révélant les flux métier naturels. Cette organisation expose les dépendances temporelles et les séquences causales. Les incohérences et les trous dans la compréhension deviennent visibles.

**Phase 3 — Pain Points et Questions** : Les zones d'incertitude, les conflits de compréhension et les problèmes connus sont marqués avec des post-its spécifiques (souvent roses ou rouges). Ces marqueurs identifient les priorités d'investigation et les risques architecturaux.

**Phase 4 — Commands et Actors** : L'ajout des commandes et des acteurs explicite qui fait quoi et pourquoi. Cette phase transforme une liste d'événements en un modèle comportemental complet.

**Phase 5 — Aggregates et Policies** : L'identification des agrégats et des politiques structure le modèle en composants implémentables. Les frontières des bounded contexts commencent à émerger naturellement.

> **Bonnes pratiques**  
> Ne traitez pas Event Storming comme un exercice ponctuel. C'est un processus itératif qui évolue avec la compréhension du domaine. Planifiez des sessions de raffinement régulières, particulièrement après les retours de production ou l'identification de nouveaux cas d'usage.

### Event Storming pour les Systèmes Multi-Agents

Une publication récente (2025) démontre l'application d'Event Storming à la conception de systèmes multi-agents (MAS). La méthodologie s'adapte naturellement à ce contexte en traitant les agents comme des acteurs spécialisés et leurs interactions comme des événements de domaine.

Dans ce contexte étendu, chaque agent cognitif peut être modélisé comme un acteur qui émet des commandes et réagit aux événements selon ses politiques internes. Les bounded contexts identifiés par Event Storming correspondent souvent aux domaines de responsabilité des agents individuels. Cette correspondance facilite la définition des frontières d'autonomie et des protocoles de collaboration inter-agents.

L'étude de cas d'une chaîne d'approvisionnement illustre cette approche : les événements « CommandeReçue », « StockVérifié », « ExpéditionPlanifiée » définissent le flux métier, tandis que les agents « Agent de validation », « Agent d'inventaire », « Agent logistique » se répartissent les responsabilités selon les bounded contexts identifiés.

---

## II.3.2 Typologie des Événements

### Classification Fonctionnelle des Événements

Tous les événements ne sont pas équivalents. Une classification rigoureuse guide les décisions de conception — granularité, rétention, partitionnement — et clarifie les contrats entre producteurs et consommateurs.

**Événements de domaine (Domain Events)** : Les faits métier significatifs qui représentent un changement d'état dans le domaine. « ClientEnregistré », « CommandeExpédiée », « FacturePayée ». Ces événements constituent le langage ubiquitaire du domaine et portent une sémantique métier riche. Ils sont la matière première des agents cognitifs qui doivent comprendre le contexte métier.

**Événements d'intégration (Integration Events)** : Les événements conçus spécifiquement pour la communication entre bounded contexts ou systèmes. Ils peuvent être des versions simplifiées ou transformées des événements de domaine, adaptées aux besoins des consommateurs externes. La distinction est importante : un événement de domaine « ArticleAjoutéAuPanier » peut générer un événement d'intégration « InventaireRéservé » pour le système de stock.

**Événements de notification** : Les signaux légers indiquant qu'un changement s'est produit, sans porter toutes les données du changement. Le consommateur doit interroger la source pour obtenir les détails. Cette approche réduit la taille des messages mais crée un couplage temporel avec la source.

**Événements de transfert d'état (Event-Carried State Transfer)** : Les événements qui transportent l'état complet ou partiel de l'entité concernée, permettant aux consommateurs de maintenir des projections locales sans interroger la source. Cette approche favorise l'autonomie des consommateurs au prix d'une redondance de données.

> **Perspective stratégique**  
> Pour les systèmes agentiques, privilégiez les événements de transfert d'état. Les agents cognitifs fonctionnent mieux avec un contexte riche immédiatement disponible plutôt qu'avec des références nécessitant des requêtes supplémentaires. Le surcoût en volume de données est compensé par la réduction de latence et la simplification du raisonnement de l'agent.

### Granularité des Événements

La granularité — le niveau de détail capturé par chaque événement — influence profondément l'architecture et les capacités analytiques du système.

**Événements fins (fine-grained)** : Chaque micro-changement génère un événement distinct. « PrixModifié », « QuantitéAjustée », « AdresseCorrigée ». Cette approche maximise la traçabilité et permet une reconstruction précise de l'historique, mais génère un volume élevé d'événements et peut fragmenter la compréhension du contexte.

**Événements agrégés (coarse-grained)** : Les changements sont regroupés en événements significatifs. « CommandeMiseÀJour » avec tous les champs modifiés. Cette approche réduit le volume et simplifie le traitement, mais perd le détail des changements individuels.

**Approche hybride** : La pratique recommandée combine les deux niveaux. Les événements fins alimentent les besoins d'audit et d'event sourcing, tandis que des événements agrégés sont dérivés pour les consommateurs qui n'ont pas besoin du détail.

### Conventions de Nommage des Événements

Le nommage des événements constitue un acte de conception du langage ubiquitaire. Des conventions cohérentes facilitent la compréhension, la découverte et la maintenance.

**Structure recommandée** : `<Entité><Action><Qualificateur optionnel>`
- Entité : Le concept métier concerné (Commande, Client, Paiement)
- Action : Ce qui s'est passé, au participe passé (Créée, Validé, Annulée)
- Qualificateur : Précision contextuelle si nécessaire (ParClient, PourFraude)

**Exemples** :
- `CommandeCréée` — événement de création simple
- `PaiementValidé` — événement de transition d'état
- `CommandeAnnuléePourFraude` — événement qualifié avec contexte

**Anti-patterns à éviter** :
- Noms techniques : `INSERT_ORDER` ou `order.created.v2`
- Noms génériques : `DataChanged` ou `EntityUpdated`
- Noms ambigus : `OrderEvent` sans indication de l'action

---

## II.3.3 Conception des Topics et Stratégies de Partitionnement

### Principes de Conception des Topics Kafka

La conception des topics Kafka traduit le modèle de domaine en structure technique. Chaque décision — nombre de topics, granularité, nommage — impacte les performances, la scalabilité et la maintenabilité du système.

**Un topic par type d'événement** : L'approche la plus simple et la plus courante. Chaque type d'événement dispose de son topic dédié : `orders.created`, `orders.shipped`, `payments.validated`. Cette stratégie facilite le filtrage par les consommateurs et permet des politiques de rétention différenciées.

**Un topic par agrégat** : Tous les événements d'un même agrégat cohabitent dans un topic unique : `orders` contient `OrderCreated`, `OrderShipped`, `OrderCancelled`. Cette approche simplifie l'event sourcing et garantit l'ordre des événements d'une même entité, mais complique le filtrage pour les consommateurs intéressés par un seul type.

**Topics par bounded context** : Les événements sont regroupés par domaine fonctionnel : `sales-events`, `inventory-events`, `shipping-events`. Cette organisation reflète la structure organisationnelle et facilite la gouvernance par équipe.

> **Bonnes pratiques**  
> Commencez par un topic par type d'événement pour les nouveaux projets. Cette granularité offre la meilleure flexibilité pour l'évolution future. Consolidez en topics par agrégat uniquement si l'event sourcing est un besoin explicite et que la garantie d'ordre intra-agrégat est critique.

### Conventions de Nommage des Topics

Un schéma de nommage cohérent est essentiel pour la découvrabilité et la gouvernance à l'échelle de l'entreprise. Confluent recommande une structure hiérarchique.

**Structure recommandée** : `<environnement>.<domaine>.<entité>.<action>.<version>`

| Composant | Description | Exemples |
|-----------|-------------|----------|
| environnement | Contexte de déploiement | `prod`, `staging`, `dev` |
| domaine | Bounded context ou équipe | `sales`, `inventory`, `payments` |
| entité | Agrégat ou concept métier | `orders`, `customers`, `invoices` |
| action | Type d'événement | `created`, `updated`, `shipped` |
| version | Version du schéma (optionnel) | `v1`, `v2` |

**Exemples** :
- `prod.sales.orders.created`
- `staging.inventory.stock.adjusted`
- `dev.payments.transactions.validated.v2`

**Éléments à exclure** : Évitez d'inclure dans le nom du topic les informations dynamiques (producteur, consommateur, timestamp) ou les métadonnées disponibles ailleurs (nombre de partitions, niveau de sécurité).

### Stratégies de Partitionnement

Le partitionnement détermine comment les messages sont distribués entre les partitions d'un topic. Cette décision impacte directement le parallélisme, l'ordonnancement et la scalabilité.

**Partitionnement par clé métier** : La stratégie la plus courante. Les messages avec la même clé (ex: `customerId`, `orderId`) sont garantis d'arriver dans la même partition, préservant l'ordre pour cette entité. Cette approche est idéale quand l'ordre des événements d'une même entité est critique.

```
Clé: order-123 → Partition 2 (tous les événements order-123)
Clé: order-456 → Partition 0 (tous les événements order-456)
Clé: order-789 → Partition 2 (hash collision possible)
```

**Partitionnement round-robin** : Sans clé, Kafka distribue les messages de manière équilibrée entre les partitions. Cette approche maximise le parallélisme mais ne garantit aucun ordre. Appropriée pour les événements sans relation entre eux.

**Partitionnement personnalisé** : Un partitionneur custom permet des stratégies sophistiquées — partitionnement géographique, par priorité, ou par affinité de traitement.

> **Attention**  
> Le choix de la clé de partitionnement est irréversible pour les données existantes. Kafka garantit l'ordre uniquement au sein d'une partition. Si vous choisissez `customerId` comme clé, tous les événements d'un client seront ordonnés, mais pas les événements de commandes différentes du même client si vous avez aussi besoin de l'ordre par commande.

### Dimensionnement des Partitions

Le nombre de partitions d'un topic définit le parallélisme maximal de consommation — un consumer group ne peut avoir plus de consommateurs actifs que de partitions.

**Règles de dimensionnement** :
- Minimum 6 partitions pour tout topic de production (permettant une croissance future)
- Aligner sur le nombre de consommateurs attendus × facteur de croissance
- Considérer le débit : chaque partition peut gérer ~10 MB/s en écriture
- Éviter l'excès : trop de partitions augmente la latence de réplication et la charge du contrôleur

**Formule pratique** :
```
Partitions = max(débit_cible / débit_par_partition, 
                 consommateurs_attendus × 2,
                 6)
```

Pour un topic à 50 MB/s avec 4 consommateurs :
```
Partitions = max(50/10, 4×2, 6) = max(5, 8, 6) = 8 partitions
```

---

## II.3.4 Patrons d'Évolution des Événements (Versioning)

### Le Défi de l'Évolution des Schémas

Dans les systèmes distribués, les schémas d'événements évoluent inévitablement. De nouveaux champs apparaissent, d'anciens deviennent obsolètes, des types changent. Sans stratégie d'évolution, ces changements brisent les consommateurs et paralysent le système.

Le Schema Registry de Confluent adresse ce défi en centralisant la gestion des schémas et en appliquant des règles de compatibilité. Chaque schéma reçoit un identifiant unique et un numéro de version. Les producteurs enregistrent leurs schémas ; les consommateurs les récupèrent pour désérialiser correctement les messages.

> **Définition formelle**  
> La compatibilité de schéma définit les conditions sous lesquelles un nouveau schéma peut remplacer un ancien sans briser les producteurs ou consommateurs existants. Les modes principaux sont : BACKWARD (nouveaux consommateurs lisent anciens messages), FORWARD (anciens consommateurs lisent nouveaux messages), et FULL (les deux directions).

### Modes de Compatibilité

**BACKWARD** (défaut Confluent) : Les consommateurs avec le nouveau schéma peuvent lire les messages produits avec l'ancien schéma. Permet d'ajouter des champs optionnels avec valeur par défaut, de supprimer des champs. Idéal quand les consommateurs sont mis à jour avant les producteurs.

**FORWARD** : Les consommateurs avec l'ancien schéma peuvent lire les messages produits avec le nouveau schéma. Permet d'ajouter des champs (ignorés par les anciens consommateurs), de supprimer des champs optionnels. Idéal quand les producteurs sont mis à jour avant les consommateurs.

**FULL** : Combinaison de BACKWARD et FORWARD. Les changements autorisés sont plus restrictifs : ajout/suppression de champs optionnels avec valeur par défaut uniquement. Recommandé pour l'event sourcing où les événements historiques doivent être relisibles par toutes les versions.

**BACKWARD_TRANSITIVE / FORWARD_TRANSITIVE / FULL_TRANSITIVE** : La compatibilité est vérifiée non seulement avec la version précédente, mais avec toutes les versions historiques. Critique pour les systèmes de longue durée.

| Mode | Ajout champ optionnel | Suppression champ optionnel | Ajout champ requis | Changement type |
|------|----------------------|-----------------------------|--------------------|-----------------|
| BACKWARD | ✓ (avec défaut) | ✓ | ✗ | ✗ |
| FORWARD | ✓ | ✓ (avec défaut) | ✗ | ✗ |
| FULL | ✓ (avec défaut) | ✓ (avec défaut) | ✗ | ✗ |

### Stratégies d'Évolution par Format

**Apache Avro** : Le format privilégié pour Kafka grâce à son support natif de l'évolution. Les champs avec valeur par défaut peuvent être ajoutés ou supprimés librement en mode BACKWARD. Le schéma est stocké séparément des données, permettant une sérialisation compacte.

```json
{
  "type": "record",
  "name": "OrderCreated",
  "namespace": "com.example.events",
  "fields": [
    {"name": "orderId", "type": "string"},
    {"name": "customerId", "type": "string"},
    {"name": "amount", "type": "double"},
    {"name": "currency", "type": "string", "default": "CAD"},
    {"name": "loyaltyPoints", "type": ["null", "int"], "default": null}
  ]
}
```

**Protocol Buffers (Protobuf)** : Populaire pour sa performance et son support multilangage. Depuis Protobuf 3, tous les champs sont optionnels par défaut, facilitant l'évolution. La recommandation pour Protobuf est BACKWARD_TRANSITIVE car l'ajout de nouveaux types de messages n'est pas forward compatible.

**JSON Schema** : Plus flexible mais moins strict. Les règles de compatibilité sont moins formalisées que pour Avro ou Protobuf. Utile pour les cas où la lisibilité humaine prime sur la performance.

> **Bonnes pratiques**  
> Adoptez FULL_TRANSITIVE pour les événements critiques et les systèmes d'event sourcing. Le coût en flexibilité est compensé par la garantie que tout événement historique reste lisible par toute version du consommateur. Pour les événements éphémères à courte rétention, BACKWARD suffit généralement.

### Gestion des Changements Incompatibles

Malgré les meilleures intentions, des changements incompatibles surviennent — renommage de champ, changement de type, restructuration majeure. Plusieurs stratégies permettent de gérer ces situations.

**Topic versionné** : Créer un nouveau topic (`orders.created.v2`) pour le nouveau schéma. Les consommateurs migrent progressivement. L'ancien topic est maintenu jusqu'à expiration de la rétention ou migration complète.

**Période de double publication** : Produire temporairement sur les deux versions du topic, permettant aux consommateurs de migrer à leur rythme. Coûteux en ressources mais offrant une transition douce.

**Transformation en vol** : Un composant intermédiaire (Kafka Streams, Flink) transforme les événements de l'ancien format vers le nouveau, alimentant un topic de destination unifié.

---

## II.3.5 Documentation des Flux Asynchrones avec AsyncAPI

### AsyncAPI : L'OpenAPI des Architectures Événementielles

AsyncAPI est la spécification standard pour documenter les API asynchrones, jouant pour les architectures événementielles le rôle qu'OpenAPI joue pour les API REST. La version 3.0, publiée en 2023 et largement adoptée en 2024-2025, apporte des améliorations majeures : support du pattern request/reply, channels réutilisables, et séparation claire entre canaux et opérations.

> **Définition formelle**  
> AsyncAPI est une spécification ouverte qui décrit les API asynchrones de manière indépendante du protocole. Elle définit les serveurs (brokers), les canaux (topics/queues), les messages (structure et schéma), et les opérations (send/receive), permettant la génération automatique de documentation, de code, et de validateurs.

### Structure d'un Document AsyncAPI 3.0

Un document AsyncAPI se compose de plusieurs sections décrivant exhaustivement l'API asynchrone.

```yaml
asyncapi: 3.0.0
info:
  title: Service de Commandes
  version: 1.0.0
  description: API événementielle pour la gestion des commandes

servers:
  production:
    host: kafka.example.com:9092
    protocol: kafka
    description: Cluster Kafka de production
    security:
      - type: scramSha256

channels:
  orderCreated:
    address: prod.sales.orders.created
    description: Événements de création de commande
    messages:
      orderCreatedMessage:
        $ref: '#/components/messages/OrderCreated'
    bindings:
      kafka:
        partitions: 12
        replicas: 3

operations:
  publishOrderCreated:
    action: send
    channel:
      $ref: '#/channels/orderCreated'
    summary: Publie un événement de création de commande
    
  consumeOrderCreated:
    action: receive
    channel:
      $ref: '#/channels/orderCreated'
    summary: Consomme les événements de création de commande

components:
  messages:
    OrderCreated:
      name: OrderCreated
      contentType: application/json
      payload:
        $ref: '#/components/schemas/OrderCreatedPayload'
        
  schemas:
    OrderCreatedPayload:
      type: object
      required:
        - orderId
        - customerId
        - amount
      properties:
        orderId:
          type: string
          format: uuid
        customerId:
          type: string
        amount:
          type: number
          format: double
        currency:
          type: string
          default: CAD
        createdAt:
          type: string
          format: date-time
```

### Bindings Kafka Spécifiques

AsyncAPI supporte des bindings spécifiques aux protocoles, permettant de documenter les configurations Kafka.

```yaml
channels:
  orderEvents:
    bindings:
      kafka:
        topic: prod.sales.orders
        partitions: 12
        replicas: 3
        topicConfiguration:
          cleanup.policy: ['delete']
          retention.ms: 604800000  # 7 jours
          
operations:
  publishOrder:
    bindings:
      kafka:
        groupId: 
          type: string
          description: ID du consumer group
        clientId:
          type: string
        bindingVersion: '0.5.0'
```

### Écosystème d'Outils AsyncAPI

L'écosystème AsyncAPI fournit des outils qui automatisent la documentation et le développement.

**AsyncAPI Studio** : Éditeur en ligne pour créer, valider et visualiser les spécifications AsyncAPI. Génère automatiquement une documentation interactive.

**AsyncAPI Generator** : Génère du code (Java, Python, TypeScript), de la documentation HTML, ou des configurations à partir de la spécification. Supporte de nombreux templates communautaires.

```bash
# Générer un projet Spring Boot
asyncapi generate fromTemplate spec.yaml @asyncapi/java-spring-template -o ./output

# Générer de la documentation HTML
asyncapi generate fromTemplate spec.yaml @asyncapi/html-template -o ./docs
```

**AsyncAPI CLI** : Outil en ligne de commande pour valider, convertir et manipuler les spécifications.

```bash
# Valider une spécification
asyncapi validate spec.yaml

# Convertir de 2.x vers 3.0
asyncapi convert spec-v2.yaml --output=spec-v3.yaml --target-version=3.0.0
```

> **Perspective stratégique**  
> Intégrez la génération de documentation AsyncAPI dans vos pipelines CI/CD. Chaque modification du schéma d'événement doit automatiquement mettre à jour la documentation. Cette approche « documentation as code » garantit que la documentation reste synchronisée avec l'implémentation.

### AsyncAPI pour les Systèmes Agentiques

Dans le contexte de l'entreprise agentique, AsyncAPI joue un rôle crucial pour les agents cognitifs. La spécification sert de contrat formel que les agents peuvent interpréter pour comprendre les événements disponibles, leur structure, et leur sémantique.

Un agent d'orchestration peut parser la spécification AsyncAPI pour découvrir dynamiquement les canaux disponibles et les formats de messages. Cette capacité d'introspection est fondamentale pour les architectures où les agents doivent s'adapter à des écosystèmes évolutifs.

Les descriptions en langage naturel incluses dans AsyncAPI (`description`, `summary`) fournissent le contexte sémantique que les agents LLM peuvent exploiter pour comprendre l'intention métier des événements, au-delà de leur structure technique.

---

## II.3.6 Résumé

Ce chapitre a établi les fondations méthodologiques et techniques pour la conception de flux d'événements de qualité dans l'entreprise agentique.

**Event Storming comme point de départ** : Cette technique collaborative réunit experts métier et techniques pour explorer le domaine à travers ses événements. Les post-its colorés — événements (orange), commandes (bleu), acteurs (jaune), agrégats (jaune pâle), politiques (violet) — construisent un modèle partagé qui se traduit directement en architecture événementielle. L'approche s'étend naturellement aux systèmes multi-agents où chaque agent correspond à un bounded context.

**Typologie rigoureuse des événements** : La classification distingue événements de domaine, d'intégration, de notification, et de transfert d'état. Pour les systèmes agentiques, les événements riches en contexte (transfert d'état) sont privilégiés car ils fournissent aux agents l'information nécessaire au raisonnement sans requêtes supplémentaires. La granularité et les conventions de nommage (`<Entité><Action>`) structurent le langage ubiquitaire du système.

**Conception des topics Kafka** : Le nommage hiérarchique (`<env>.<domaine>.<entité>.<action>`) assure la découvrabilité à l'échelle entreprise. Le choix de la clé de partitionnement — critique et irréversible — détermine les garanties d'ordre. Le dimensionnement des partitions (minimum 6, aligné sur le parallélisme cible) équilibre performance et overhead opérationnel.

**Stratégies d'évolution des schémas** : Le Schema Registry Confluent centralise la gouvernance des schémas avec des modes de compatibilité (BACKWARD, FORWARD, FULL) qui préviennent les ruptures. Avro reste le format privilégié pour sa gestion native de l'évolution. FULL_TRANSITIVE est recommandé pour les événements critiques et l'event sourcing. Les changements incompatibles se gèrent par topics versionnés ou transformation en vol.

**Documentation AsyncAPI 3.0** : La spécification standardise la documentation des API asynchrones avec support des bindings Kafka. L'écosystème d'outils (Studio, Generator, CLI) automatise la génération de documentation et de code. Pour les agents cognitifs, AsyncAPI fournit le contrat formel et le contexte sémantique nécessaires à l'interprétation des événements.

Ces pratiques établissent le pont entre l'intention métier capturée en atelier et l'implémentation technique sur Kafka. Le chapitre suivant (II.4) approfondira la gouvernance sémantique avec le Schema Registry, transformant ces principes de conception en contrats de données exécutoires.

---

*La qualité d'une architecture événementielle ne se mesure pas à la sophistication de son infrastructure, mais à la clarté avec laquelle elle exprime le domaine métier. Les événements sont le langage du système — leur conception mérite la même attention que l'on porterait à la conception d'une API publique ou d'une interface utilisateur.*

*Chapitre suivant : Chapitre II.4 — Contrats de Données et Gouvernance Sémantique (Schema Registry)*
