# Chapitre III.6

## CONTRATS DE DONNÉES

---

> *« Un schéma est un contrat. Un contrat brisé détruit la confiance. La confiance est le fondement de tout système distribué. »*
>
> — Martin Kleppmann, Designing Data-Intensive Applications

---

Le chapitre précédent a exploré quand utiliser Kafka et les alternatives disponibles. Cette analyse stratégique présuppose une question fondamentale : comment les producteurs et consommateurs s'accordent-ils sur la structure des données échangées ? Sans réponse rigoureuse à cette question, même l'architecture la plus élégante s'effondre sous le poids des incompatibilités et des erreurs de désérialisation.

Les contrats de données sont le fondement silencieux de toute architecture événementielle réussie. Ils définissent la structure, le format, et les règles d'évolution des messages. Sans eux, chaque modification de schéma devient une opération risquée nécessitant la coordination synchrone de tous les systèmes — exactement le couplage que Kafka est censé éliminer.

Ce chapitre explore la conception, l'implémentation, et la gouvernance des contrats de données dans l'écosystème Kafka. Nous verrons comment traduire les besoins métier en schémas techniques, comment Kafka gère (ou ne gère pas) la structure des événements, et comment le Schema Registry apporte les garanties nécessaires pour une évolution contrôlée.

---

## III.6.1 Traduire les Produits d'Affaires en Schémas

### Du Domaine Métier à la Représentation Technique

La conception d'un schéma d'événement commence par la compréhension du domaine métier, pas par les considérations techniques. L'erreur la plus fréquente est de concevoir les schémas en fonction des structures de bases de données existantes plutôt que des événements métier réels.

**L'approche Domain-Driven Design (DDD).** Les événements Kafka doivent représenter des faits métier significatifs — des choses qui se sont produites dans le domaine. Un événement `OrderCreated` capture le fait qu'une commande a été créée, avec toutes les informations pertinentes au moment de la création.

*Événement vs. État* : Un événement capture un changement à un instant donné. Il est immuable par nature — ce qui s'est passé ne peut pas être modifié. L'état, en revanche, est une projection cumulative des événements. Cette distinction est fondamentale pour la conception des schémas.

```
Événement (fait immuable)              État (projection mutable)
─────────────────────────              ─────────────────────────
OrderCreated {                         Order {
  order_id: "123"                        order_id: "123"
  customer_id: "456"                     customer_id: "456"
  items: [...]                           items: [...]
  total: 150.00                          total: 150.00
  created_at: "2024-01-15T10:30:00Z"     status: "shipped"      ← modifié
}                                        shipped_at: "2024-01-16T..." ← ajouté
                                       }
OrderShipped {
  order_id: "123"
  shipped_at: "2024-01-16T14:00:00Z"
  tracking_number: "XYZ789"
}
```

**Event Storming comme outil de découverte.** L'Event Storming est un atelier collaboratif qui réunit experts métier et développeurs pour identifier les événements du domaine. Le résultat est une cartographie des événements qui guide directement la conception des topics et des schémas.

*Processus* :
1. Identifier les événements métier (post-its orange) : « Commande créée », « Paiement reçu », « Article expédié »
2. Identifier les commandes qui déclenchent ces événements (post-its bleu) : « Passer commande », « Confirmer paiement »
3. Identifier les agrégats (entités) concernés (post-its jaune) : « Commande », « Paiement », « Expédition »
4. Regrouper par bounded context : « Ventes », « Paiements », « Logistique »

Chaque événement identifié devient potentiellement un type de message Kafka avec son propre schéma.

### Principes de Conception des Schémas

**Principe 1 : Autonomie de l'événement (Self-Contained Events).**

Un événement doit contenir toutes les informations nécessaires pour être compris et traité indépendamment. Le consommateur ne devrait pas avoir besoin de faire des lookups dans d'autres systèmes pour comprendre l'événement.

```json
// ❌ Mauvais : Événement incomplet nécessitant des lookups
{
  "event_type": "OrderCreated",
  "order_id": "123",
  "customer_id": "456"  // Le consommateur doit chercher les détails du client ailleurs
}

// ✅ Bon : Événement autonome avec les informations pertinentes
{
  "event_type": "OrderCreated",
  "order_id": "123",
  "customer": {
    "id": "456",
    "name": "Jean Dupont",
    "email": "jean@example.com",
    "segment": "premium"
  },
  "items": [...],
  "total": 150.00,
  "currency": "CAD",
  "created_at": "2024-01-15T10:30:00Z"
}
```

*Trade-off* : Les événements autonomes sont plus volumineux mais réduisent le couplage. Les consommateurs sont indépendants des bases de données des producteurs.

**Principe 2 : Nommage explicite et cohérent.**

Les noms des événements et des champs doivent être explicites, non ambigus, et cohérents à travers tous les schémas de l'organisation.

*Conventions recommandées* :
- Événements au passé : `OrderCreated`, `PaymentReceived`, `ItemShipped` (pas `CreateOrder`)
- Champs en snake_case ou camelCase (choisir une convention et s'y tenir)
- Types explicites : `customer_id` plutôt que `id`, `order_total_amount` plutôt que `total`
- Unités dans le nom si ambigu : `duration_seconds`, `amount_cents`

**Principe 3 : Versionner dès le début.**

Tout schéma évoluera. Intégrer la notion de version dès la conception initiale, même si une seule version existe.

```json
{
  "schema_version": "1.0",
  "event_type": "OrderCreated",
  "event_id": "uuid-...",
  "event_time": "2024-01-15T10:30:00Z",
  "payload": {
    // Données métier
  }
}
```

**Principe 4 : Séparer métadonnées et payload.**

Les métadonnées techniques (timestamps, IDs de corrélation, source) doivent être séparées des données métier. Cela permet une évolution indépendante et un traitement standardisé des métadonnées.

```json
{
  "metadata": {
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "event_type": "OrderCreated",
    "event_time": "2024-01-15T10:30:00Z",
    "source_system": "checkout-service",
    "correlation_id": "req-abc-123",
    "causation_id": "evt-xyz-789",
    "schema_version": "1.2"
  },
  "payload": {
    "order_id": "ORD-12345",
    "customer_id": "CUST-67890",
    "items": [...],
    "total_amount": 15000,
    "currency": "CAD"
  }
}
```

> **Définition formelle**
>
> **Contrat de données** : Accord formel entre producteurs et consommateurs spécifiant la structure, le format, la sémantique, et les règles d'évolution des messages échangés. Le contrat inclut :
> - Le schéma technique (Avro, Protobuf, JSON Schema)
> - La documentation sémantique (signification des champs, unités, valeurs valides)
> - Les règles de compatibilité (backward, forward, full)
> - Les SLA (latence, disponibilité, fraîcheur des données)
> - Les responsabilités (propriétaire du schéma, processus de modification)
>
> Un contrat de données bien défini est la fondation de la confiance dans une architecture événementielle. Sans lui, chaque interaction entre systèmes devient une source potentielle d'erreurs et d'incompréhensions.

### Granularité des Événements

La question de la granularité est récurrente : faut-il des événements fins (un événement par changement atomique) ou des événements agrégés (un événement résumant plusieurs changements) ? La réponse dépend des besoins des consommateurs et des exigences de traçabilité.

**Événements fins (Fine-Grained Events).**

```
OrderCreated → ItemAddedToOrder → ItemAddedToOrder → OrderSubmitted → PaymentReceived → OrderConfirmed
```

*Avantages* : Flexibilité maximale, possibilité de reconstruire n'importe quel état intermédiaire, audit détaillé complet, support natif de l'Event Sourcing.

*Inconvénients* : Volume élevé de messages, complexité de traitement pour les consommateurs qui ont besoin de l'image complète, nécessité de maintenir un état pour reconstituer les agrégats.

**Événements agrégés (Coarse-Grained Events).**

```
OrderCompleted (contient tous les détails de la commande finalisée avec l'historique résumé)
```

*Avantages* : Simple à consommer, volume réduit, image complète en un seul message, pas besoin de maintenir un état côté consommateur.

*Inconvénients* : Perte d'information sur le processus détaillé, moins de flexibilité pour les cas d'usage non anticipés, difficulté à répondre aux questions « quand exactement X s'est-il produit ? ».

**Approche hybride recommandée.**

Publier les événements fins pour les consommateurs qui en ont besoin (audit, replay détaillé), et des événements agrégés pour les consommateurs qui ont besoin d'une vue synthétique. Les deux types peuvent coexister dans des topics différents.

```
Topic: orders.events (fin)           Topic: orders.completed (agrégé)
├── OrderCreated                     └── OrderCompleted (résumé)
├── ItemAdded
├── ItemAdded
├── PaymentReceived
└── OrderConfirmed
```

> **Note de terrain**
>
> *Contexte* : Plateforme e-commerce avec 50 microservices consommant des événements de commande.
>
> *Problème initial* : Seuls des événements fins étaient publiés. Les services de reporting devaient agréger 5-10 événements pour avoir une vue complète d'une commande. Complexité, latence, et bugs fréquents.
>
> *Solution* : Ajout d'un topic `orders.snapshots` avec des événements agrégés publiés à chaque changement d'état significatif. Les services simples consomment les snapshots ; les services d'audit consomment les événements fins.
>
> *Résultat* : Réduction de 70% du code de consommation pour les services simples, maintien de l'audit détaillé.

---

## III.6.2 Comment Kafka Gère la Structure des Événements

### Kafka est Agnostique au Contenu

Un point fondamental souvent mal compris : **Kafka ne comprend pas le contenu des messages**. Pour Kafka, un message est une séquence d'octets (bytes) avec une clé optionnelle. Kafka ne valide pas, ne parse pas, et ne transforme pas le contenu.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Message Kafka (vue interne)                          │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │ Headers (optionnel)                                              │   │
│  │ ┌─────────────────┬─────────────────┐                           │   │
│  │ │ Key: "trace-id" │ Value: "abc123" │                           │   │
│  │ └─────────────────┴─────────────────┘                           │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │ Key (optionnel) : byte[]                                         │   │
│  │ 0x4F 0x52 0x44 0x2D 0x31 0x32 0x33 0x34 0x35  ("ORD-12345")     │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │ Value : byte[]                                                   │   │
│  │ 0x7B 0x22 0x6F 0x72 0x64 0x65 0x72 ...  (JSON, Avro, Protobuf?) │   │
│  │ Kafka ne sait pas et ne se soucie pas du format                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  Timestamp: 1705312200000                                              │
│  Partition: 3                                                          │
│  Offset: 42                                                            │
└─────────────────────────────────────────────────────────────────────────┘
```

**Conséquences de cette agnosticité :**

*Flexibilité* : Kafka peut transporter n'importe quel format — JSON, Avro, Protobuf, XML, binaire propriétaire, images, fichiers compressés.

*Responsabilité déplacée* : La validation et la compatibilité des schémas sont la responsabilité des producteurs et consommateurs, pas de Kafka.

*Risque* : Sans mécanisme externe, rien n'empêche un producteur de publier des données corrompues ou incompatibles.

### Formats de Sérialisation

Le choix du format de sérialisation impacte les performances, la compatibilité, et l'outillage disponible.

**JSON (JavaScript Object Notation).**

```json
{
  "order_id": "ORD-12345",
  "customer_id": "CUST-67890",
  "total_amount": 15000,
  "currency": "CAD",
  "created_at": "2024-01-15T10:30:00Z"
}
```

*Avantages* :
- Lisible par les humains (débogage facile)
- Supporté universellement (tous les langages)
- Pas de schéma requis pour la lecture de base
- Flexible (champs optionnels naturels)

*Inconvénients* :
- Verbeux (noms de champs répétés dans chaque message)
- Pas de typage fort (le champ `total_amount` est-il un entier ou un flottant ?)
- Pas de validation de schéma native
- Performance de sérialisation/désérialisation modérée

*Cas d'usage* : Prototypage, faibles volumes, intégration avec des systèmes legacy, cas où la lisibilité prime.

**Apache Avro.**

```avro
{
  "type": "record",
  "name": "OrderCreated",
  "namespace": "com.example.orders",
  "fields": [
    {"name": "order_id", "type": "string"},
    {"name": "customer_id", "type": "string"},
    {"name": "total_amount", "type": "long", "doc": "Amount in cents"},
    {"name": "currency", "type": "string", "default": "CAD"},
    {"name": "created_at", "type": "long", "logicalType": "timestamp-millis"}
  ]
}
```

*Avantages* :
- Format binaire compact (pas de noms de champs dans les données)
- Schéma intégré ou référencé (Schema Registry)
- Évolution de schéma native avec règles de compatibilité
- Performance excellente
- Support natif dans l'écosystème Confluent

*Inconvénients* :
- Non lisible par les humains (binaire)
- Nécessite le schéma pour la lecture
- Courbe d'apprentissage pour les développeurs

*Cas d'usage* : Production à grande échelle, intégration Confluent, cas où la compatibilité de schéma est critique.

**Protocol Buffers (Protobuf).**

```protobuf
syntax = "proto3";

package com.example.orders;

message OrderCreated {
  string order_id = 1;
  string customer_id = 2;
  int64 total_amount = 3;  // Amount in cents
  string currency = 4;
  google.protobuf.Timestamp created_at = 5;
}
```

*Avantages* :
- Format binaire très compact
- Performance de sérialisation excellente
- Génération de code dans de nombreux langages
- Évolution de schéma via numéros de champs
- Largement utilisé (Google, gRPC)

*Inconvénients* :
- Nécessite compilation du schéma (.proto → code)
- Non lisible par les humains
- Moins intégré nativement avec l'écosystème Kafka (mais supporté par Schema Registry)

*Cas d'usage* : Environnements polyglotes, intégration avec gRPC, haute performance requise.

**Comparaison des formats :**

| Critère | JSON | Avro | Protobuf |
|---------|------|------|----------|
| Taille message | Grande | Petite | Très petite |
| Lisibilité | Excellente | Nulle | Nulle |
| Performance sérialisation | Modérée | Excellente | Excellente |
| Typage | Faible | Fort | Fort |
| Évolution de schéma | Manuelle | Native | Native |
| Intégration Schema Registry | Via JSON Schema | Native | Supporté |
| Génération de code | Non nécessaire | Optionnelle | Requise |
| Courbe d'apprentissage | Faible | Moyenne | Moyenne |
| Débogage | Facile | Difficile | Difficile |

**Impact sur la taille des messages — exemple concret :**

Pour un événement `OrderCreated` typique avec 10 champs, voici les tailles observées en production :

```
Événement OrderCreated (même contenu, formats différents):
┌────────────────────────────────────────────────────────────────────────┐
│ Format      │ Taille (bytes) │ Ratio vs JSON │ Messages/sec (1 Gbps)  │
├────────────────────────────────────────────────────────────────────────┤
│ JSON        │     450        │    1.0x       │    277,000             │
│ JSON (gzip) │     180        │    0.4x       │    694,000             │
│ Avro        │     120        │    0.27x      │  1,041,000             │
│ Protobuf    │      95        │    0.21x      │  1,315,000             │
└────────────────────────────────────────────────────────────────────────┘
```

*Observation* : Le choix du format peut multiplier par 4-5 le débit théorique sur un même réseau. Pour les architectures à très haut volume (> 100 000 msg/sec), ce choix est critique.

**Considérations pratiques pour le choix :**

*Choisir JSON si* :
- L'équipe n'a pas d'expérience avec les formats binaires
- Le volume est faible (< 1 000 msg/sec)
- Le débogage fréquent est nécessaire (environnements de développement)
- L'intégration avec des systèmes legacy JSON est requise
- La flexibilité prime sur la performance

*Choisir Avro si* :
- L'environnement utilise Confluent Platform
- La compatibilité de schéma est critique
- Le volume justifie un format compact
- L'équipe peut investir dans l'apprentissage
- La génération de code optionnelle est un avantage

*Choisir Protobuf si* :
- L'environnement utilise déjà gRPC
- La performance est la priorité absolue
- L'équipe maîtrise déjà Protobuf
- La compilation de schémas est acceptable dans le workflow

> **Décision architecturale**
>
> *Contexte* : Nouvelle plateforme événementielle pour une banque. 50 microservices, 100+ types d'événements, volumes de 50 000 messages/seconde, exigences réglementaires strictes.
>
> *Options considérées* :
> 1. JSON : Simple, lisible, mais pas de garanties de compatibilité et taille importante
> 2. Avro : Compact, évolution native, intégration Confluent Cloud
> 3. Protobuf : Très compact, très performant, mais compilation requise
>
> *Analyse détaillée* :
> - Volume : 50k msg/sec × 450 bytes (JSON) = 22 MB/sec vs 6 MB/sec (Avro) → économie réseau significative
> - Compatibilité : Critique pour une banque — Avro et Protobuf offrent des garanties, pas JSON
> - Intégration : Confluent Cloud choisi → Avro nativement supporté
> - Équipes : 8 équipes de développement, expertise variable → Avro plus accessible que Protobuf
>
> *Décision* : Avro avec Schema Registry Confluent.
>
> *Justification* :
> - La compatibilité de schéma est critique pour une banque (pas de messages perdus ou corrompus)
> - L'intégration native avec Confluent Cloud simplifie l'opérationnel
> - Les volumes justifient un format compact (économie de 70% sur la bande passante)
> - La génération de code optionnelle réduit la friction pour les équipes
>
> *Concession* : JSON autorisé pour les topics de développement/debug avec rétention courte (24h).
>
> *Métriques de succès* : Aucun incident de compatibilité en 18 mois de production.

---

## III.6.3 Défis dans la Conception d'Événements

### Le Problème de l'Évolution des Schémas

Les schémas évoluent inévitablement. Les besoins métier changent, de nouveaux champs sont nécessaires, d'anciens champs deviennent obsolètes. Le défi est de gérer cette évolution sans briser les consommateurs existants.

**Scénario typique de rupture :**

```
Jour 1: Producteur publie OrderCreated v1
        Consommateur A traite OrderCreated v1 ✓

Jour 30: Producteur modifie le schéma → OrderCreated v2
         - Renomme "total" en "total_amount"
         - Ajoute champ obligatoire "currency"

Jour 30: Consommateur A (non mis à jour) reçoit OrderCreated v2
         - Cherche le champ "total" → absent → ERREUR
         - Ne connaît pas "currency" → ignore (ou erreur)
```

Ce scénario illustre pourquoi l'évolution des schémas est l'un des problèmes les plus critiques dans les architectures événementielles. Un changement apparemment simple peut avoir des conséquences en cascade sur des dizaines de consommateurs.

**Types de changements de schéma :**

*Changements rétrocompatibles (Backward Compatible)* :
- Ajouter un champ avec valeur par défaut
- Supprimer un champ optionnel
- Élargir un type (int → long)
- Ajouter un alias pour un champ existant

*Changements non rétrocompatibles (Breaking Changes)* :
- Renommer un champ
- Supprimer un champ obligatoire
- Changer le type d'un champ (string → int)
- Ajouter un champ obligatoire sans défaut
- Réduire un type (long → int)
- Modifier le nom d'un type enum

### Analyse d'Impact des Changements

Avant toute modification de schéma, une analyse d'impact rigoureuse est nécessaire.

**Checklist d'analyse d'impact :**

1. **Identifier tous les consommateurs** : Quels services, applications, ou équipes consomment ce topic ?

2. **Évaluer la nature du changement** : Est-il backward compatible ? Forward compatible ? Breaking ?

3. **Déterminer l'ordre de déploiement** : Qui doit être mis à jour en premier — producteurs ou consommateurs ?

4. **Planifier la migration** : Période de transition, communication, rollback possible ?

5. **Tester la compatibilité** : Validation automatisée avec les schémas existants.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Matrice d'Impact des Changements                     │
│                                                                         │
│  Type de changement          │ Impact │ Action requise                  │
│  ────────────────────────────┼────────┼───────────────────────────────  │
│  Ajout champ avec défaut     │ Faible │ Déployer producteur d'abord     │
│  Suppression champ optionnel │ Moyen  │ Vérifier aucun consommateur     │
│                              │        │ n'utilise le champ              │
│  Renommage de champ          │ Élevé  │ Migration en plusieurs phases   │
│  Changement de type          │ Élevé  │ Nouveau topic ou versioning     │
│  Ajout champ obligatoire     │ Élevé  │ Ajouter défaut ou nouveau topic │
└─────────────────────────────────────────────────────────────────────────┘
```

> **Note de terrain**
>
> *Contexte* : Équipe produit souhaitant renommer le champ `price` en `unit_price` pour plus de clarté.
>
> *Analyse* : 12 consommateurs utilisent ce champ. Un renommage direct casserait tous les consommateurs.
>
> *Solution mise en place* (migration en 4 phases) :
> 1. **Phase 1** : Ajouter `unit_price` comme alias de `price` (v2 du schéma). Les deux champs contiennent la même valeur.
> 2. **Phase 2** : Mettre à jour tous les consommateurs pour lire `unit_price` (sur 4 semaines).
> 3. **Phase 3** : Modifier le producteur pour ne plus remplir `price` (déprécié).
> 4. **Phase 4** : Supprimer `price` du schéma (v3) après confirmation que plus aucun consommateur ne l'utilise.
>
> *Durée totale* : 8 semaines. Aucune interruption de service.
>
> *Leçon* : Les renommages « simples » sont en réalité des migrations complexes. Planifier en conséquence.

### Stratégies de Compatibilité

Le Schema Registry Confluent définit plusieurs modes de compatibilité qui contrôlent quelles évolutions sont autorisées.

**BACKWARD (par défaut).**

Les nouveaux schémas peuvent lire les données écrites avec les anciens schémas. C'est le mode le plus courant car il protège les consommateurs existants.

*Autorisé* : Supprimer des champs, ajouter des champs optionnels avec défaut.

*Interdit* : Ajouter des champs obligatoires, changer les types de manière restrictive.

*Cas d'usage* : Les consommateurs sont mis à jour avant les producteurs. C'est l'approche recommandée pour la plupart des organisations.

```
Ordre de déploiement BACKWARD:
1. Mettre à jour les consommateurs (peuvent lire v1 et v2)
2. Mettre à jour les producteurs (écrivent v2)
3. Les anciens messages v1 sont toujours lisibles
4. Pas de downtime, pas de perte de messages
```

**FORWARD.**

Les anciens schémas peuvent lire les données écrites avec les nouveaux schémas. Moins courant mais utile quand le contrôle sur les consommateurs est limité.

*Autorisé* : Ajouter des champs (les anciens consommateurs ignorent), supprimer des champs optionnels.

*Interdit* : Supprimer des champs obligatoires, changer les types.

*Cas d'usage* : Les producteurs sont mis à jour avant les consommateurs. Utile quand les consommateurs sont externes ou difficiles à mettre à jour.

```
Ordre de déploiement FORWARD:
1. Mettre à jour les producteurs (écrivent v2)
2. Les anciens consommateurs lisent v2 (ignorent les nouveaux champs)
3. Mettre à jour les consommateurs à leur rythme
4. Flexibilité maximale pour les producteurs
```

**FULL.**

Combinaison de BACKWARD et FORWARD. Les schémas peuvent évoluer dans les deux sens.

*Autorisé* : Ajouter/supprimer des champs optionnels avec défaut uniquement.

*Interdit* : Tout changement de champ obligatoire ou de type.

*Cas d'usage* : Environnements où l'ordre de déploiement n'est pas contrôlable, ou microservices avec déploiements indépendants.

**BACKWARD_TRANSITIVE / FORWARD_TRANSITIVE / FULL_TRANSITIVE.**

Les versions transitives vérifient la compatibilité avec TOUTES les versions précédentes, pas seulement la dernière. Plus strict mais plus sûr pour les topics avec longue rétention.

**NONE.**

Aucune vérification de compatibilité. Le Schema Registry accepte tout schéma.

*Cas d'usage* : Développement uniquement. JAMAIS en production.

> **Perspective stratégique**
>
> Le choix du mode de compatibilité est une décision d'architecture, pas une décision technique ponctuelle. Il détermine :
> - L'ordre de déploiement des services
> - La liberté d'évolution des schémas
> - Le risque de rupture en production
> - La complexité des migrations
>
> Pour la plupart des organisations, **BACKWARD** est le bon choix par défaut. Il permet une évolution contrôlée tout en protégeant contre les ruptures accidentelles. Pour les topics critiques avec longue rétention, **BACKWARD_TRANSITIVE** offre une protection supplémentaire.

### Versioning Explicite vs. Implicite

**Versioning implicite (via Schema Registry).**

Le Schema Registry assigne automatiquement des IDs de version aux schémas. Les messages contiennent l'ID du schéma utilisé. Le consommateur récupère le schéma correspondant pour désérialiser.

```
Message Avro avec versioning implicite:
┌──────────────┬────────────────────────────────────┐
│ Magic Byte   │ 0x00 (Confluent wire format)      │
│ Schema ID    │ 0x00 0x00 0x00 0x05 (ID = 5)      │
│ Data         │ Données binaires Avro              │
└──────────────┴────────────────────────────────────┘

Processus de consommation:
1. Lire le magic byte (validation format)
2. Lire les 4 bytes suivants → Schema ID = 5
3. Requête au Schema Registry: GET /schemas/ids/5
4. Cache le schéma localement
5. Désérialiser les données avec ce schéma
```

*Avantages* : Transparent, géré automatiquement, pas de champ version dans les données, efficace en espace.

*Inconvénients* : Dépendance au Schema Registry (point de défaillance), opaque pour le débogage manuel.

**Versioning explicite (dans les données).**

Un champ version explicite dans chaque message permet un routage et un traitement différencié sans dépendance externe.

```json
{
  "schema_version": "2.1",
  "event_type": "OrderCreated",
  "payload": {...}
}
```

*Avantages* : Visible, permet un routage explicite, fonctionne sans Schema Registry, facilite le débogage.

*Inconvénients* : Redondant si Schema Registry est utilisé, maintenance manuelle du numéro de version.

**Approche recommandée** : Utiliser le Schema Registry pour la validation et la compatibilité (versioning implicite), et optionnellement un champ version explicite dans le header pour la traçabilité et le débogage. Les deux approches ne sont pas mutuellement exclusives.

### Patterns d'Évolution de Schéma

**Pattern 1 : Ajout de champ avec migration progressive.**

```avro
// Version 1
{
  "fields": [
    {"name": "order_id", "type": "string"},
    {"name": "total", "type": "long"}
  ]
}

// Version 2 : Ajout de currency avec défaut
{
  "fields": [
    {"name": "order_id", "type": "string"},
    {"name": "total", "type": "long"},
    {"name": "currency", "type": "string", "default": "CAD"}
  ]
}
```

Déploiement :
1. Déployer les consommateurs mis à jour (lisent v1 et v2)
2. Déployer les producteurs (écrivent v2)
3. Les messages v1 restants sont lus avec currency = "CAD"

**Pattern 2 : Dépréciation de champ.**

```avro
// Version 1
{
  "fields": [
    {"name": "order_id", "type": "string"},
    {"name": "total", "type": "long"},
    {"name": "price", "type": "long", "doc": "DEPRECATED: use total instead"}
  ]
}

// Version 2 : Suppression après migration
{
  "fields": [
    {"name": "order_id", "type": "string"},
    {"name": "total", "type": "long"}
  ]
}
```

**Pattern 3 : Nouveau topic pour breaking changes.**

Quand un changement est véritablement breaking et qu'une migration n'est pas possible, créer un nouveau topic.

```
orders-v1 (ancien schéma) → maintenu pour les anciens consommateurs
orders-v2 (nouveau schéma) → utilisé par les nouveaux producteurs/consommateurs

Service de transition : lit v1, transforme, écrit v2
```

> **Anti-patron**
>
> *« Nous changeons le type du champ `amount` de string à long pour corriger une erreur de conception. »*
>
> *Problème* : Changement de type = breaking change. Tous les consommateurs casseront immédiatement.
>
> *Conséquences* :
> - Erreurs de désérialisation en production
> - Perte potentielle de messages
> - Déploiement d'urgence de tous les consommateurs
>
> *Solution correcte* :
> 1. Ajouter un nouveau champ `amount_cents` (long) avec défaut
> 2. Migrer tous les consommateurs vers le nouveau champ
> 3. Marquer `amount` comme déprécié
> 4. Supprimer `amount` après confirmation que plus personne ne l'utilise

---

## III.6.4 Structure de l'Événement et Mapping

### Anatomie d'un Événement Bien Conçu

Un événement complet comprend plusieurs couches d'information, chacune avec un rôle spécifique.

```json
{
  "header": {
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "event_type": "com.example.orders.OrderCreated",
    "event_version": "1.2",
    "event_time": "2024-01-15T10:30:00.000Z",
    "source": {
      "system": "checkout-service",
      "instance": "checkout-prod-3",
      "version": "2.4.1"
    },
    "correlation": {
      "correlation_id": "req-abc-123",
      "causation_id": "evt-xyz-789",
      "trace_id": "trace-456"
    }
  },
  "payload": {
    "order": {
      "id": "ORD-12345",
      "status": "created",
      "customer": {
        "id": "CUST-67890",
        "name": "Jean Dupont",
        "email": "jean@example.com"
      },
      "items": [
        {
          "product_id": "PROD-111",
          "name": "Widget Pro",
          "quantity": 2,
          "unit_price_cents": 5000
        }
      ],
      "totals": {
        "subtotal_cents": 10000,
        "tax_cents": 1500,
        "total_cents": 11500
      },
      "currency": "CAD"
    }
  },
  "context": {
    "tenant_id": "tenant-acme",
    "region": "ca-central-1",
    "environment": "production"
  }
}
```

**Couche Header (métadonnées techniques) :**

| Champ | Description | Utilité |
|-------|-------------|---------|
| event_id | Identifiant unique de l'événement (UUID) | Déduplication, traçabilité |
| event_type | Type complet de l'événement | Routage, filtrage |
| event_version | Version du schéma | Compatibilité |
| event_time | Timestamp de l'événement métier | Ordre, fenêtrage |
| source | Information sur le producteur | Débogage, audit |
| correlation_id | ID de la requête/transaction origine | Traçage distribué |
| causation_id | ID de l'événement qui a causé celui-ci | Chaîne causale |

**Couche Payload (données métier) :**

Le payload contient les données métier spécifiques à l'événement. Sa structure dépend du domaine et du type d'événement.

**Couche Context (contexte d'exécution) :**

Informations sur le contexte dans lequel l'événement a été produit. Utile pour le multi-tenant, le routage régional, ou la segmentation environnementale.

### Mapping entre Systèmes

Dans une architecture événementielle, les événements traversent des frontières de systèmes. Le mapping entre les représentations est un défi récurrent.

**Mapping de types de données :**

| Concept | Java | Avro | JSON | PostgreSQL |
|---------|------|------|------|------------|
| Entier 64 bits | long | long | number | bigint |
| Date/heure | Instant | long (timestamp-millis) | string (ISO8601) | timestamp |
| Décimal précis | BigDecimal | bytes (decimal) | string | numeric |
| UUID | UUID | string | string | uuid |
| Énumération | enum | enum | string | enum/varchar |

> **Anti-patron**
>
> *« Nous utilisons des float pour les montants monétaires. »*
>
> *Problème* : Les nombres à virgule flottante (float, double) ne peuvent pas représenter précisément les valeurs décimales. 0.1 + 0.2 ≠ 0.3 en virgule flottante.
>
> *Conséquences* : Erreurs d'arrondi dans les calculs financiers, différences entre systèmes, audits qui ne correspondent pas.
>
> *Solution* : Utiliser des entiers en cents (ou millièmes) pour les montants, ou des types décimaux précis (BigDecimal, numeric).
>
> ```json
> // ❌ Mauvais
> { "amount": 15.99 }
>
> // ✅ Bon
> { "amount_cents": 1599 }
> ```

**Mapping d'énumérations :**

Les énumérations évoluent (nouveaux statuts, nouvelles catégories). Le mapping doit anticiper cette évolution.

```avro
// Schéma Avro avec enum
{
  "type": "enum",
  "name": "OrderStatus",
  "symbols": ["CREATED", "CONFIRMED", "SHIPPED", "DELIVERED", "CANCELLED"],
  "default": "CREATED"  // Valeur par défaut pour les valeurs inconnues
}
```

*Problème* : Si le producteur ajoute un nouveau statut `RETURNED` et que le consommateur ne connaît pas cette valeur, que se passe-t-il ?

*Solutions* :
1. Valeur par défaut : Le consommateur utilise la valeur par défaut
2. String au lieu d'enum : Plus flexible mais moins typé
3. Versioning explicite : Le consommateur traite selon la version

### Gestion des Références et Relations

Les événements contiennent souvent des références à d'autres entités. Comment les représenter ?

**Option 1 : Référence par ID uniquement.**

```json
{
  "order_id": "ORD-123",
  "customer_id": "CUST-456"  // Référence seule
}
```

*Avantages* : Compact, pas de duplication.

*Inconvénients* : Le consommateur doit faire un lookup pour obtenir les détails.

**Option 2 : Objet embarqué (dénormalisé).**

```json
{
  "order_id": "ORD-123",
  "customer": {
    "id": "CUST-456",
    "name": "Jean Dupont",
    "email": "jean@example.com"
  }
}
```

*Avantages* : Événement autonome, pas de lookup nécessaire.

*Inconvénients* : Données potentiellement obsolètes si le client change.

**Option 3 : Hybride avec snapshot.**

```json
{
  "order_id": "ORD-123",
  "customer_id": "CUST-456",
  "customer_snapshot": {
    "name": "Jean Dupont",
    "email": "jean@example.com",
    "snapshot_time": "2024-01-15T10:30:00Z"
  }
}
```

*Avantages* : ID pour les jointures futures, snapshot pour le contexte historique.

*Inconvénients* : Plus verbeux, nécessite de maintenir les snapshots.

> **Décision architecturale**
>
> *Contexte* : Système de e-commerce avec événements de commande. Les commandes référencent des clients et des produits.
>
> *Question* : Embarquer les détails ou référencer par ID ?
>
> *Analyse* :
> - Pour le client : Les données client au moment de la commande sont importantes (adresse de livraison, segment). Embarquer.
> - Pour les produits : Les détails produit peuvent changer (prix, description), mais le prix au moment de la commande est fixe. Embarquer le prix et la quantité, référencer le produit par ID.
>
> *Décision* : Hybride — embarquer les données critiques au moment de l'événement, référencer par ID pour les lookups futurs.

---

## III.6.5 Notes de Terrain : Stratégies de Données Customer360

### Le Défi du Customer 360

Le « Customer 360 » est un cas d'usage classique des architectures événementielles : construire une vue unifiée du client à partir de multiples sources de données. Ce cas illustre parfaitement les défis des contrats de données à grande échelle.

**Le contexte typique :**

```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│   CRM       │  │ E-commerce  │  │   Support   │  │  Marketing  │
│ (Salesforce)│  │  (Magento)  │  │  (Zendesk)  │  │  (HubSpot)  │
└──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘
       │                │                │                │
       ▼                ▼                ▼                ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Apache Kafka                            │
│  ┌───────────────┐ ┌───────────────┐ ┌───────────────┐         │
│  │ crm.customers │ │ ecom.orders   │ │ support.tickets│         │
│  └───────────────┘ └───────────────┘ └───────────────┘         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                 ┌─────────────────────────┐
                 │   Customer 360 Service  │
                 │   (Kafka Streams)       │
                 └───────────┬─────────────┘
                             │
                             ▼
                 ┌─────────────────────────┐
                 │   Vue Client Unifiée    │
                 │   (Elasticsearch)       │
                 └─────────────────────────┘
```

**Les défis concrets rencontrés :**

### Défi 1 : Identification du Client

Chaque système a sa propre notion d'identifiant client.

| Système | Identifiant | Format |
|---------|-------------|--------|
| CRM | Account ID | `ACC-12345` |
| E-commerce | Customer ID | `67890` (numérique) |
| Support | User ID | `user_abc123` |
| Marketing | Contact ID | UUID |

*Solution mise en place* : Table de correspondance (Identity Resolution) et identifiant canonique.

```json
// Événement enrichi avec identifiants multiples
{
  "event_type": "CustomerProfileUpdated",
  "identifiers": {
    "canonical_id": "CUST-550e8400-e29b-41d4",
    "crm_id": "ACC-12345",
    "ecom_id": "67890",
    "support_id": "user_abc123",
    "marketing_id": "550e8400-e29b-41d4-a716-446655440000"
  },
  "profile": {...}
}
```

### Défi 2 : Schémas Hétérogènes

Chaque système a sa propre représentation du client avec des champs différents, des types différents, et des sémantiques différentes.

```json
// CRM (Salesforce)
{
  "AccountId": "ACC-12345",
  "Name": "ACME Corp",
  "BillingAddress": {
    "street": "123 Main St",
    "city": "Montreal",
    "state": "QC"
  }
}

// E-commerce (Magento)
{
  "customer_id": 67890,
  "firstname": "Jean",
  "lastname": "Dupont",
  "addresses": [
    {"type": "billing", "street1": "123 Main St", ...}
  ]
}
```

*Solution* : Schéma canonique avec mapping explicite.

```avro
// Schéma canonique Customer360
{
  "type": "record",
  "name": "Customer360",
  "fields": [
    {"name": "canonical_id", "type": "string"},
    {"name": "display_name", "type": "string"},
    {"name": "email", "type": ["null", "string"]},
    {"name": "addresses", "type": {
      "type": "array",
      "items": {
        "type": "record",
        "name": "Address",
        "fields": [
          {"name": "type", "type": {"type": "enum", "symbols": ["BILLING", "SHIPPING", "OTHER"]}},
          {"name": "line1", "type": "string"},
          {"name": "line2", "type": ["null", "string"]},
          {"name": "city", "type": "string"},
          {"name": "region", "type": "string"},
          {"name": "postal_code", "type": "string"},
          {"name": "country", "type": "string"}
        ]
      }
    }},
    {"name": "source_records", "type": {
      "type": "array",
      "items": {
        "type": "record",
        "name": "SourceRecord",
        "fields": [
          {"name": "source_system", "type": "string"},
          {"name": "source_id", "type": "string"},
          {"name": "last_updated", "type": "long"}
        ]
      }
    }}
  ]
}
```

### Défi 3 : Cohérence Temporelle

Les événements arrivent dans le désordre. Un événement CRM peut arriver avant l'événement e-commerce correspondant, ou vice versa.

*Problème* : Comment fusionner les données si elles arrivent dans le désordre ?

*Solution* : Fenêtrage et timestamps explicites.

```java
// Kafka Streams : Jointure avec fenêtre temporelle
KStream<String, CrmEvent> crmStream = ...;
KStream<String, EcomEvent> ecomStream = ...;

// Jointure sur une fenêtre de 5 minutes
KStream<String, Customer360> joined = crmStream.join(
    ecomStream,
    (crm, ecom) -> merge(crm, ecom),
    JoinWindows.of(Duration.ofMinutes(5)),
    StreamJoined.with(Serdes.String(), crmSerde, ecomSerde)
);
```

### Défi 4 : Qualité des Données

Les données sources contiennent des erreurs, des incohérences, et des doublons. Ce problème est systématiquement sous-estimé lors de la planification des projets d'intégration.

*Exemples rencontrés en production* :
- Emails invalides : `"jean@"`, `"test@test.test"`, `"N/A"`
- Dates impossibles : `"2099-13-45"`, `"0000-00-00"`
- Doublons : Même client créé deux fois avec des IDs différents suite à des imports manuels
- Données incohérentes : Adresse de facturation au Canada avec code postal américain
- Champs mal mappés : Numéro de téléphone dans le champ fax
- Encodage incorrect : Caractères accentués corrompus `"JÃ©rÃ´me"` au lieu de `"Jérôme"`

*Solution* : Pipeline de validation et enrichissement avec plusieurs couches.

```java
// Service de validation et nettoyage des données client
public class CustomerDataValidator {
    
    private final EmailValidator emailValidator;
    private final AddressValidator addressValidator;
    private final PhoneNormalizer phoneNormalizer;
    
    public ValidationResult validate(RawCustomerEvent event) {
        Customer360 customer = map(event);
        List<ValidationWarning> warnings = new ArrayList<>();
        List<ValidationError> errors = new ArrayList<>();
        
        // Validation email
        if (customer.getEmail() != null) {
            if (!emailValidator.isValid(customer.getEmail())) {
                customer.setEmail(null);
                warnings.add(new ValidationWarning(
                    "email", 
                    "Invalid email discarded: " + customer.getEmail(),
                    Severity.MEDIUM
                ));
            }
        }
        
        // Validation dates
        if (customer.getBirthDate() != null) {
            if (customer.getBirthDate().isAfter(LocalDate.now())) {
                customer.setBirthDate(null);
                warnings.add(new ValidationWarning(
                    "birth_date",
                    "Future birth date discarded",
                    Severity.LOW
                ));
            }
            if (customer.getBirthDate().isBefore(LocalDate.of(1900, 1, 1))) {
                customer.setBirthDate(null);
                warnings.add(new ValidationWarning(
                    "birth_date",
                    "Implausible birth date discarded",
                    Severity.LOW
                ));
            }
        }
        
        // Normalisation téléphone
        if (customer.getPhone() != null) {
            String normalized = phoneNormalizer.normalize(customer.getPhone());
            if (normalized == null) {
                warnings.add(new ValidationWarning(
                    "phone",
                    "Could not normalize phone: " + customer.getPhone(),
                    Severity.LOW
                ));
            } else {
                customer.setPhone(normalized);
            }
        }
        
        // Validation adresse avec enrichissement
        if (customer.getAddress() != null) {
            AddressValidationResult addrResult = addressValidator.validate(customer.getAddress());
            if (addrResult.isValid()) {
                // Enrichir avec les données normalisées (code postal formaté, etc.)
                customer.setAddress(addrResult.getNormalizedAddress());
            } else {
                warnings.add(new ValidationWarning(
                    "address",
                    "Address validation failed: " + addrResult.getMessage(),
                    Severity.MEDIUM
                ));
            }
        }
        
        // Calcul du score de qualité
        double qualityScore = calculateQualityScore(customer, warnings);
        customer.setDataQualityScore(qualityScore);
        
        return new ValidationResult(customer, warnings, errors, qualityScore);
    }
    
    private double calculateQualityScore(Customer360 customer, List<ValidationWarning> warnings) {
        double score = 100.0;
        
        // Pénalités pour champs manquants
        if (customer.getEmail() == null) score -= 15;
        if (customer.getPhone() == null) score -= 10;
        if (customer.getAddress() == null) score -= 20;
        
        // Pénalités pour warnings
        for (ValidationWarning warning : warnings) {
            switch (warning.getSeverity()) {
                case HIGH: score -= 10; break;
                case MEDIUM: score -= 5; break;
                case LOW: score -= 2; break;
            }
        }
        
        return Math.max(0, score);
    }
}
```

**Métriques de qualité à surveiller :**

| Métrique | Description | Seuil d'alerte |
|----------|-------------|----------------|
| Taux de validation | % d'enregistrements passant la validation | < 85% |
| Score qualité moyen | Score de qualité moyen des enregistrements | < 70 |
| Taux de doublons | % d'enregistrements identifiés comme doublons | > 5% |
| Taux d'enrichissement | % d'enregistrements enrichis avec succès | < 90% |
| Latence de traitement | Temps de traitement par enregistrement | > 100ms |

> **Note de terrain**
>
> *Contexte* : Implémentation Customer 360 pour un détaillant avec 5 millions de clients et 4 systèmes sources.
>
> *Durée du projet* : 8 mois (initial), 18 mois (avec améliorations itératives)
>
> *Statistiques de qualité découvertes* :
> - 15% des enregistrements clients avaient au moins un problème de qualité
> - 8% des emails étaient invalides ou fictifs
> - 3% des clients existaient en double dans au moins deux systèmes
> - 12% des adresses ne pouvaient pas être validées
>
> *Leçons apprises* :
>
> 1. **L'identity resolution est le problème le plus difficile.** 40% de l'effort total a été consacré à la résolution d'identité. Investir dans ce domaine dès le début du projet.
>
> 2. **Les schémas canoniques évoluent.** Prévoir 2-3 itérations majeures du schéma canonique. Utiliser BACKWARD compatibility pour permettre cette évolution.
>
> 3. **La qualité des données est pire qu'attendu.** Toujours. Construire des pipelines de nettoyage robustes et prévoir du temps pour l'analyse des anomalies.
>
> 4. **Le temps réel n'est pas toujours nécessaire.** La vue Customer 360 était initialement temps réel (latence < 1 seconde), mais les consommateurs n'avaient besoin que d'une fraîcheur de 15 minutes. Simplifier l'architecture a réduit les coûts de 60%.
>
> 5. **Documenter les mappings est critique.** Un wiki avec les mappings source → canonique, maintenu à jour, a été essentiel pour l'onboarding des nouveaux développeurs et le débogage en production.
>
> 6. **Prévoir un mécanisme de replay.** La capacité de rejouer les événements depuis une date donnée a sauvé le projet lors de bugs de mapping découverts après plusieurs semaines.

---

## III.6.6 Schema Registry dans l'Écosystème Kafka

### Architecture et Fonctionnement

Le Schema Registry est un composant central de l'écosystème Confluent qui fournit un service de gestion des schémas pour Kafka. Il résout le problème fondamental de l'agnosticité de Kafka envers le contenu des messages en ajoutant une couche de gouvernance des schémas.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Architecture Schema Registry                         │
│                                                                         │
│  ┌─────────────┐                              ┌─────────────────────┐  │
│  │  Producer   │                              │   Schema Registry   │  │
│  │             │  1. Enregistrer schéma       │   (REST API)        │  │
│  │  ┌───────┐  │ ─────────────────────────▶   │                     │  │
│  │  │Avro   │  │                              │  ┌───────────────┐  │  │
│  │  │Serial.│  │  2. Obtenir Schema ID        │  │ Schémas       │  │  │
│  │  └───────┘  │ ◀─────────────────────────   │  │ (Kafka topic) │  │  │
│  └──────┬──────┘                              │  └───────────────┘  │  │
│         │                                      └─────────────────────┘  │
│         │ 3. Publier message                            ▲              │
│         │    [Schema ID | Data]                         │              │
│         ▼                                               │              │
│  ┌─────────────────────────────────────────┐           │              │
│  │              Apache Kafka               │           │              │
│  │  ┌─────────────────────────────────┐   │           │              │
│  │  │ Topic: orders                    │   │           │              │
│  │  │ [5|binary] [5|binary] [5|binary] │   │           │              │
│  │  └─────────────────────────────────┘   │           │              │
│  └──────────────────┬──────────────────────┘           │              │
│                     │                                   │              │
│         ┌───────────┴───────────┐                      │              │
│         ▼                       ▼                      │              │
│  ┌─────────────┐         ┌─────────────┐              │              │
│  │  Consumer A │         │  Consumer B │              │              │
│  │  ┌───────┐  │         │  ┌───────┐  │              │              │
│  │  │Avro   │  │ 4. Récupérer schéma   │              │              │
│  │  │Deser. │──┼─────────┼──│Deser. │──┼──────────────┘              │
│  │  └───────┘  │         │  └───────┘  │                             │
│  └─────────────┘         └─────────────┘                             │
└─────────────────────────────────────────────────────────────────────────┘
```

**Composants du Schema Registry :**

*Service REST* : API HTTP pour l'enregistrement, la récupération, et la validation des schémas. Haute disponibilité via clustering.

*Stockage backend* : Les schémas sont stockés dans un topic Kafka interne (`_schemas`), garantissant la durabilité et la réplication.

*Cache* : Chaque instance maintient un cache en mémoire des schémas pour des performances optimales.

**Flux de fonctionnement détaillé :**

1. **Enregistrement du schéma** : Lors du premier message avec un nouveau schéma (ou schéma modifié), le sérialiseur Avro envoie le schéma au Registry. Le Registry :
   - Vérifie si le schéma existe déjà (par hash)
   - Si nouveau, vérifie la compatibilité avec les versions précédentes
   - Assigne un ID unique global
   - Stocke le schéma dans le topic `_schemas`

2. **Sérialisation** : Le sérialiseur préfixe le message avec :
   - Magic byte (0x00) : Identifie le format Confluent
   - Schema ID (4 bytes big-endian) : Référence au schéma
   - Données binaires Avro

3. **Publication** : Le message complet est publié dans Kafka. Kafka ne voit que des bytes.

4. **Consommation** : Le désérialiseur :
   - Lit le magic byte (validation)
   - Extrait le Schema ID
   - Récupère le schéma depuis le cache local ou le Registry
   - Désérialise les données avec le schéma approprié

### Haute Disponibilité et Déploiement

Le Schema Registry supporte le clustering pour la haute disponibilité.

**Architecture de déploiement recommandée :**

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Cluster Schema Registry (3 nœuds)                    │
│                                                                         │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                 │
│  │  Registry 1 │    │  Registry 2 │    │  Registry 3 │                 │
│  │  (Leader)   │    │  (Follower) │    │  (Follower) │                 │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘                 │
│         │                  │                  │                        │
│         └──────────────────┼──────────────────┘                        │
│                            │                                           │
│                            ▼                                           │
│                 ┌─────────────────────┐                                │
│                 │   Load Balancer     │                                │
│                 └──────────┬──────────┘                                │
│                            │                                           │
│         ┌──────────────────┼──────────────────┐                        │
│         ▼                  ▼                  ▼                        │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                 │
│  │ Producer A  │    │ Consumer B  │    │ Consumer C  │                 │
│  └─────────────┘    └─────────────┘    └─────────────┘                 │
└─────────────────────────────────────────────────────────────────────────┘
```

*Mode Leader-Follower* : Un seul nœud (leader) accepte les écritures. Les followers répondent aux lectures. En cas de défaillance du leader, un follower est élu.

*Facteur de réplication* : Le topic `_schemas` doit avoir un facteur de réplication ≥ 3 pour la durabilité.

### Configuration et Utilisation

**Configuration du producteur Avro :**

```java
Properties props = new Properties();
props.put("bootstrap.servers", "kafka:9092");
props.put("key.serializer", "io.confluent.kafka.serializers.KafkaAvroSerializer");
props.put("value.serializer", "io.confluent.kafka.serializers.KafkaAvroSerializer");
props.put("schema.registry.url", "http://schema-registry:8081");

// Options de comportement
props.put("auto.register.schemas", "true");   // Auto-enregistrement (dev)
props.put("use.latest.version", "false");     // Utiliser le schéma du producteur
props.put("avro.remove.java.properties", "true");  // Nettoyer les propriétés Java

// Options de sécurité (si Schema Registry sécurisé)
props.put("basic.auth.credentials.source", "USER_INFO");
props.put("basic.auth.user.info", "user:password");

// Configuration TLS
props.put("schema.registry.ssl.truststore.location", "/path/to/truststore.jks");
props.put("schema.registry.ssl.truststore.password", "changeit");

KafkaProducer<String, OrderCreated> producer = new KafkaProducer<>(props);
```

**Configuration du consommateur Avro :**

```java
Properties props = new Properties();
props.put("bootstrap.servers", "kafka:9092");
props.put("group.id", "order-processor");
props.put("key.deserializer", "io.confluent.kafka.serializers.KafkaAvroDeserializer");
props.put("value.deserializer", "io.confluent.kafka.serializers.KafkaAvroDeserializer");
props.put("schema.registry.url", "http://schema-registry:8081");

// Options de lecture
props.put("specific.avro.reader", "true");   // Classes générées (vs. GenericRecord)
props.put("avro.use.logical.type.converters", "true");  // Support des types logiques

// Gestion des schémas inconnus
props.put("schema.reflection.fallback", "false");  // Erreur si schéma inconnu

KafkaConsumer<String, OrderCreated> consumer = new KafkaConsumer<>(props);
```

**Différence entre GenericRecord et SpecificRecord :**

```java
// GenericRecord : Pas de classe générée, accès par nom de champ
GenericRecord record = (GenericRecord) consumer.poll(...).value();
String orderId = record.get("order_id").toString();
Long total = (Long) record.get("total_amount");

// SpecificRecord : Classe générée depuis le schéma Avro
OrderCreated order = consumer.poll(...).value();
String orderId = order.getOrderId();    // Typage fort
long total = order.getTotalAmount();    // Pas de cast
```

*Recommandation* : Utiliser SpecificRecord en production pour le typage fort et la détection d'erreurs à la compilation.

### API REST du Schema Registry

Le Schema Registry expose une API REST complète pour la gestion des schémas.

**Opérations de lecture :**

```bash
# Lister tous les sujets (subjects)
curl http://schema-registry:8081/subjects
# Réponse: ["orders-key", "orders-value", "customers-value"]

# Obtenir les versions d'un sujet
curl http://schema-registry:8081/subjects/orders-value/versions
# Réponse: [1, 2, 3]

# Obtenir un schéma par sujet et version
curl http://schema-registry:8081/subjects/orders-value/versions/2
# Réponse: {"subject":"orders-value","version":2,"id":5,"schema":"{...}"}

# Obtenir le dernier schéma
curl http://schema-registry:8081/subjects/orders-value/versions/latest

# Obtenir un schéma par ID global
curl http://schema-registry:8081/schemas/ids/5
# Réponse: {"schema":"{...}"}
```

**Opérations d'écriture :**

```bash
# Enregistrer un nouveau schéma
curl -X POST -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  --data '{"schema": "{\"type\":\"record\",\"name\":\"Order\",\"fields\":[...]}"}' \
  http://schema-registry:8081/subjects/orders-value/versions
# Réponse: {"id":6}

# Vérifier la compatibilité avant enregistrement
curl -X POST -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  --data '{"schema": "{...}"}' \
  http://schema-registry:8081/compatibility/subjects/orders-value/versions/latest
# Réponse: {"is_compatible":true} ou {"is_compatible":false}
```

**Configuration de compatibilité :**

```bash
# Obtenir la configuration globale
curl http://schema-registry:8081/config
# Réponse: {"compatibilityLevel":"BACKWARD"}

# Configurer la compatibilité globale
curl -X PUT -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  --data '{"compatibility": "FULL"}' \
  http://schema-registry:8081/config

# Configurer la compatibilité par sujet
curl -X PUT -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  --data '{"compatibility": "BACKWARD_TRANSITIVE"}' \
  http://schema-registry:8081/config/orders-value

# Obtenir la configuration d'un sujet
curl http://schema-registry:8081/config/orders-value
```

**Opérations de suppression (avec précaution) :**

```bash
# Supprimer une version (soft delete)
curl -X DELETE http://schema-registry:8081/subjects/orders-value/versions/1

# Supprimer un sujet complet (soft delete)
curl -X DELETE http://schema-registry:8081/subjects/orders-value

# Suppression permanente (hard delete) - DANGER
curl -X DELETE http://schema-registry:8081/subjects/orders-value?permanent=true
```

### Conventions de Nommage des Sujets

Le Schema Registry organise les schémas par « sujets » (subjects). La stratégie de nommage détermine la correspondance entre topics Kafka et sujets Schema Registry.

**TopicNameStrategy (par défaut) :**

```
Topic: orders
  → Sujet clé: orders-key
  → Sujet valeur: orders-value
```

*Configuration* : Aucune (comportement par défaut)

*Avantage* : Simple, un schéma par topic. La compatibilité est garantie au niveau du topic.

*Inconvénient* : Un topic ne peut contenir qu'un seul type de message.

**RecordNameStrategy :**

```
Topic: orders (peut contenir plusieurs types)
  → Sujet: com.example.OrderCreated
  → Sujet: com.example.OrderUpdated
  → Sujet: com.example.OrderCancelled
```

*Configuration* :
```java
props.put("value.subject.name.strategy", 
    "io.confluent.kafka.serializers.subject.RecordNameStrategy");
```

*Avantage* : Plusieurs types d'événements dans un même topic, schémas organisés par type.

*Inconvénient* : Pas de garantie de compatibilité inter-types dans le même topic.

**TopicRecordNameStrategy :**

```
Topic: orders
  → Sujet: orders-com.example.OrderCreated
  → Sujet: orders-com.example.OrderUpdated
```

*Configuration* :
```java
props.put("value.subject.name.strategy", 
    "io.confluent.kafka.serializers.subject.TopicRecordNameStrategy");
```

*Avantage* : Combinaison des deux — isolation par topic ET par type.

*Cas d'usage* : Topics multi-types où chaque type évolue indépendamment.

> **Décision architecturale**
>
> *Contexte* : Architecture avec des topics par domaine contenant plusieurs types d'événements.
>
> *Question* : Quelle stratégie de nommage ?
>
> *Analyse* :
> - TopicNameStrategy : Trop restrictif (un type par topic)
> - RecordNameStrategy : Risque de confusion si le même type existe dans plusieurs topics
> - TopicRecordNameStrategy : Meilleur compromis
>
> *Décision* : TopicRecordNameStrategy pour les topics multi-types, TopicNameStrategy pour les topics mono-type.
>
> *Documentation* : Documenter la stratégie dans les guidelines de l'équipe.

---

## III.6.7 Problèmes Courants dans la Gestion des Contrats

L'expérience collective des implémentations Kafka à grande échelle révèle des problèmes récurrents dans la gestion des contrats de données. Cette section documente ces problèmes et leurs solutions éprouvées.

### Problème 1 : Schéma Drift (Dérive de Schéma)

**Symptôme** : Les producteurs et consommateurs utilisent des versions de schéma différentes non compatibles, causant des erreurs de désérialisation en production.

**Manifestations courantes** :
- Exceptions `SerializationException` ou `AvroRuntimeException` dans les logs
- Messages non traités s'accumulant dans les topics
- Consommateurs qui crashent au démarrage après un déploiement

**Causes racines** :
- Déploiements non coordonnés entre équipes
- `auto.register.schemas=true` en production permettant des schémas non validés
- Compatibilité désactivée ou mal configurée
- Absence de validation dans la CI/CD
- Tests insuffisants avant déploiement

**Solutions :**

*Solution 1 : Désactiver l'auto-registration en production*

```java
// Configuration producteur en PRODUCTION
props.put("auto.register.schemas", "false");  // OBLIGATOIRE
props.put("use.latest.version", "true");      // Utiliser le dernier schéma enregistré
```

Les schémas doivent être enregistrés explicitement via la CI/CD, pas automatiquement au runtime.

*Solution 2 : Pipeline CI/CD avec validation*

```yaml
# GitLab CI : Validation et enregistrement de schéma
stages:
  - validate
  - register
  - deploy

validate-schema:
  stage: validate
  script:
    - |
      # Lire le schéma
      SCHEMA=$(cat schemas/order-created.avsc | jq -c '.')
      
      # Vérifier la compatibilité
      RESULT=$(curl -s -X POST \
        -H "Content-Type: application/vnd.schemaregistry.v1+json" \
        -d "{\"schema\": \"$SCHEMA\"}" \
        "$SCHEMA_REGISTRY_URL/compatibility/subjects/orders-value/versions/latest")
      
      IS_COMPATIBLE=$(echo $RESULT | jq -r '.is_compatible')
      
      if [ "$IS_COMPATIBLE" != "true" ]; then
        echo "❌ ERREUR: Schéma incompatible!"
        echo "Détails: $RESULT"
        exit 1
      fi
      
      echo "✅ Schéma compatible"

register-schema:
  stage: register
  only:
    - main
  script:
    - |
      SCHEMA=$(cat schemas/order-created.avsc | jq -c '.')
      
      curl -X POST \
        -H "Content-Type: application/vnd.schemaregistry.v1+json" \
        -d "{\"schema\": \"$SCHEMA\"}" \
        "$SCHEMA_REGISTRY_URL/subjects/orders-value/versions"
      
      echo "✅ Schéma enregistré"
```

*Solution 3 : Tests d'intégration avec schémas réels*

```java
@Test
void shouldDeserializeWithRegisteredSchema() {
    // Utiliser le Schema Registry de test
    Properties props = new Properties();
    props.put("schema.registry.url", testSchemaRegistryUrl);
    
    // Produire un message avec le nouveau schéma
    producer.send(new ProducerRecord<>("orders", key, newOrderEvent));
    
    // Consommer avec l'ancien code
    ConsumerRecord<String, GenericRecord> record = consumer.poll(...);
    
    // Vérifier que la désérialisation fonctionne
    assertNotNull(record.value().get("order_id"));
}
```

### Problème 2 : Schémas Non Documentés

**Symptôme** : Les développeurs ne comprennent pas la signification des champs, les unités utilisées, ou les valeurs attendues. Les questions récurrentes sont : « C'est quoi ce champ `amount` ? C'est en dollars ou en cents ? »

**Causes** :
- Documentation dans le schéma absente ou minimale
- Documentation externe (wiki, confluence) désynchronisée du schéma
- Nommage ambigu des champs (`id`, `value`, `data`)
- Pas de processus de revue des schémas

**Solutions :**

*Solution 1 : Documentation inline obligatoire*

```avro
{
  "type": "record",
  "name": "OrderCreated",
  "namespace": "com.example.orders.events",
  "doc": "Événement émis lors de la création d'une nouvelle commande. Publié dans le topic 'orders-events'. Propriétaire: équipe checkout.",
  "fields": [
    {
      "name": "order_id",
      "type": "string",
      "doc": "Identifiant unique de la commande. Format: ORD-{UUID-v4}. Exemple: ORD-550e8400-e29b-41d4-a716-446655440000"
    },
    {
      "name": "customer_id",
      "type": "string",
      "doc": "Identifiant du client. Référence vers le domaine 'customers'. Format: CUST-{UUID-v4}"
    },
    {
      "name": "total_amount_cents",
      "type": "long",
      "doc": "Montant total de la commande en CENTIMES (pas en dollars). Exemple: 1599 représente 15.99 CAD. Toujours positif."
    },
    {
      "name": "currency",
      "type": "string",
      "default": "CAD",
      "doc": "Code devise ISO 4217. Valeurs supportées: CAD, USD, EUR. Défaut: CAD"
    },
    {
      "name": "created_at",
      "type": {
        "type": "long",
        "logicalType": "timestamp-millis"
      },
      "doc": "Timestamp de création en millisecondes depuis epoch UTC. Représente le moment où la commande a été soumise par le client."
    },
    {
      "name": "items",
      "type": {
        "type": "array",
        "items": "OrderItem"
      },
      "doc": "Liste des articles de la commande. Ne peut pas être vide. Maximum 100 items par commande."
    }
  ]
}
```

*Solution 2 : Conventions de nommage explicites*

| Convention | Exemple | Signification |
|------------|---------|---------------|
| `*_cents` | `amount_cents` | Montant en centimes |
| `*_at` | `created_at` | Timestamp |
| `*_id` | `customer_id` | Identifiant/référence |
| `*_count` | `item_count` | Compteur entier |
| `*_seconds` | `duration_seconds` | Durée en secondes |
| `*_millis` | `latency_millis` | Durée en millisecondes |
| `is_*` | `is_active` | Booléen |
| `has_*` | `has_items` | Booléen |

*Solution 3 : Génération automatique de documentation*

```bash
# Génération de documentation HTML depuis les schémas Avro
# Utilisation de avro-tools ou plugins Maven/Gradle

# Avec avrodoc (outil open source)
avrodoc schemas/ --output docs/schemas/

# Intégration dans la CI pour publication automatique
```

*Solution 4 : Processus de revue obligatoire*

Tout nouveau schéma ou modification doit être revu par :
1. Un membre de l'équipe Data/Architecture
2. Au moins un consommateur du schéma
3. Vérification de la documentation et des conventions

### Problème 3 : Breaking Changes Accidentels

**Symptôme** : Un déploiement « anodin » casse les consommateurs existants. L'équipe découvre le problème en production.

**Exemples de breaking changes subtils** :

```avro
// ❌ Breaking: Changement de type (même si logiquement équivalent)
// Avant: {"name": "status", "type": "string"}
// Après: {"name": "status", "type": {"type": "enum", "symbols": ["CREATED", "SHIPPED"]}}

// ❌ Breaking: Suppression d'un champ utilisé par les consommateurs
// Avant: {"name": "legacy_id", "type": ["null", "string"]}
// Après: (champ supprimé)

// ❌ Breaking: Ajout d'un champ obligatoire
// Avant: (pas de champ shipping_address)
// Après: {"name": "shipping_address", "type": "Address"}  // pas de défaut!

// ❌ Breaking: Réduction de la plage d'un type numérique
// Avant: {"name": "quantity", "type": "long"}
// Après: {"name": "quantity", "type": "int"}  // peut tronquer les valeurs
```

**Solutions :**

*Solution 1 : Mode de compatibilité strict*

```bash
# Configurer BACKWARD_TRANSITIVE pour les topics critiques
curl -X PUT -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  --data '{"compatibility": "BACKWARD_TRANSITIVE"}' \
  http://schema-registry:8081/config/orders-value
```

*Solution 2 : Hook pre-commit pour validation locale*

```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "🔍 Validation des schémas Avro..."

for schema_file in $(git diff --cached --name-only | grep '\.avsc$'); do
    subject=$(basename "$schema_file" .avsc)-value
    
    echo "Vérification de $schema_file → $subject"
    
    # Vérifier la syntaxe Avro
    if ! avro-tools idl2schemata "$schema_file" > /dev/null 2>&1; then
        echo "❌ Erreur de syntaxe dans $schema_file"
        exit 1
    fi
    
    # Vérifier la compatibilité (si le Schema Registry est accessible)
    if curl -s "$SCHEMA_REGISTRY_URL/subjects" > /dev/null 2>&1; then
        SCHEMA=$(cat "$schema_file" | jq -c '.')
        RESULT=$(curl -s -X POST \
            -H "Content-Type: application/vnd.schemaregistry.v1+json" \
            -d "{\"schema\": \"$(echo $SCHEMA | sed 's/"/\\"/g')\"}" \
            "$SCHEMA_REGISTRY_URL/compatibility/subjects/$subject/versions/latest")
        
        IS_COMPATIBLE=$(echo $RESULT | jq -r '.is_compatible // "true"')
        
        if [ "$IS_COMPATIBLE" != "true" ]; then
            echo "❌ $schema_file incompatible avec la version actuelle"
            echo "Détails: $RESULT"
            exit 1
        fi
    fi
done

echo "✅ Tous les schémas sont valides"
```

*Solution 3 : Tests de régression de schéma*

```java
@Test
void schemaEvolutionShouldBeBackwardCompatible() {
    // Charger le schéma actuel depuis le Schema Registry
    Schema currentSchema = schemaRegistry.getLatestSchema("orders-value");
    
    // Charger le nouveau schéma depuis les fichiers
    Schema newSchema = new Schema.Parser().parse(new File("schemas/order.avsc"));
    
    // Vérifier la compatibilité
    SchemaCompatibility.SchemaPairCompatibility compatibility = 
        SchemaCompatibility.checkReaderWriterCompatibility(currentSchema, newSchema);
    
    assertEquals(
        SchemaCompatibility.SchemaCompatibilityType.COMPATIBLE,
        compatibility.getType(),
        "Le nouveau schéma doit être backward compatible"
    );
}
```

### Problème 4 : Prolifération de Schémas

**Symptôme** : Des centaines de schémas avec des doublons, des versions abandonnées, et pas de propriétaire identifié. Personne ne sait quel schéma est utilisé où.

**Indicateurs du problème** :
- > 500 sujets dans le Schema Registry
- Nombreux schémas avec une seule version (jamais mis à jour = probablement abandonnés)
- Conventions de nommage incohérentes
- Questions fréquentes « Qui utilise ce schéma ? »

**Solutions :**

*Solution 1 : Catalogue centralisé avec métadonnées*

```yaml
# catalog/orders-value.yaml
subject: orders-value
topic: orders-events
owner:
  team: checkout
  slack: "#checkout-team"
  oncall: "checkout-oncall@example.com"
description: |
  Événements du cycle de vie des commandes.
  Publié par le service checkout.
consumers:
  - service: inventory-service
    team: logistics
    usage: "Mise à jour du stock"
  - service: notification-service
    team: platform
    usage: "Emails de confirmation"
created: 2023-06-15
last_reviewed: 2024-01-10
retention: 30d
compatibility: BACKWARD
sla:
  latency_p99: 500ms
  availability: 99.9%
```

*Solution 2 : Processus de création avec approbation*

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Processus de Gestion des Schémas                     │
│                                                                         │
│  ┌─────────────┐                                                       │
│  │ Développeur │                                                       │
│  │ crée/modifie│                                                       │
│  │ schéma      │                                                       │
│  └──────┬──────┘                                                       │
│         │                                                              │
│         ▼                                                              │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │ Pull Request                                                     │  │
│  │ - Fichier .avsc dans /schemas                                   │  │
│  │ - Métadonnées dans /catalog                                     │  │
│  │ - Documentation mise à jour                                      │  │
│  └──────────────────────────┬──────────────────────────────────────┘  │
│                             │                                          │
│         ┌───────────────────┼───────────────────┐                     │
│         ▼                   ▼                   ▼                     │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐               │
│  │   CI/CD     │    │  Revue par  │    │  Revue par  │               │
│  │ Validation  │    │  Data Team  │    │ Consommateur│               │
│  │ automatique │    │             │    │  (si modif) │               │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘               │
│         │                  │                  │                       │
│         └──────────────────┼──────────────────┘                       │
│                            │                                          │
│                            ▼                                          │
│                 ┌─────────────────────┐                               │
│                 │ Merge → Déploiement │                               │
│                 │ → Enregistrement    │                               │
│                 └─────────────────────┘                               │
└─────────────────────────────────────────────────────────────────────────┘
```

*Solution 3 : Nettoyage régulier*

```bash
#!/bin/bash
# Script de nettoyage mensuel des schémas obsolètes

echo "📊 Analyse des schémas..."

# Lister tous les sujets
SUBJECTS=$(curl -s $SCHEMA_REGISTRY_URL/subjects)

for subject in $(echo $SUBJECTS | jq -r '.[]'); do
    # Obtenir le nombre de versions
    VERSIONS=$(curl -s "$SCHEMA_REGISTRY_URL/subjects/$subject/versions" | jq '. | length')
    
    # Obtenir la date de dernière modification (approximation via metadata)
    LATEST=$(curl -s "$SCHEMA_REGISTRY_URL/subjects/$subject/versions/latest")
    
    # Vérifier si le topic existe et a du trafic
    # ... (intégration avec monitoring)
    
    # Alerter si le schéma semble abandonné
    if [ "$VERSIONS" -eq 1 ]; then
        echo "⚠️  $subject: Une seule version, potentiellement abandonné"
    fi
done
```

### Problème 5 : Performance du Schema Registry

**Symptôme** : Latence élevée lors de la sérialisation/désérialisation, surtout au démarrage des applications ou lors des pics de trafic.

**Causes** :
- Cache client mal configuré ou désactivé
- Schema Registry sous-dimensionné
- Réseau lent entre les clients et le Registry
- Trop de requêtes au Registry (pas de mise en cache)

**Solutions :**

*Solution 1 : Configuration optimale du cache client*

```java
// Configuration du cache côté client
props.put("schema.registry.cache.capacity", "1000");  // Nombre de schémas en cache

// Le cache est activé par défaut, mais vérifier qu'il n'est pas désactivé
// La première requête pour un schéma va au Registry, les suivantes utilisent le cache
```

*Solution 2 : Pré-chargement au démarrage*

```java
@Component
public class SchemaPreloader {
    
    @Autowired
    private SchemaRegistryClient schemaRegistry;
    
    @PostConstruct
    public void preloadSchemas() {
        log.info("Pré-chargement des schémas...");
        
        List<String> criticalSubjects = Arrays.asList(
            "orders-value",
            "customers-value",
            "payments-value"
        );
        
        for (String subject : criticalSubjects) {
            try {
                schemaRegistry.getLatestSchemaMetadata(subject);
                log.info("Schéma {} chargé", subject);
            } catch (Exception e) {
                log.warn("Impossible de charger {}: {}", subject, e.getMessage());
            }
        }
        
        log.info("Pré-chargement terminé");
    }
}
```

*Solution 3 : Schema Registry en haute disponibilité*

```yaml
# docker-compose.yml pour Schema Registry HA
services:
  schema-registry-1:
    image: confluentinc/cp-schema-registry:7.5.0
    environment:
      SCHEMA_REGISTRY_HOST_NAME: schema-registry-1
      SCHEMA_REGISTRY_KAFKASTORE_BOOTSTRAP_SERVERS: kafka:9092
      SCHEMA_REGISTRY_LEADER_ELIGIBILITY: "true"
      SCHEMA_REGISTRY_HEAP_OPTS: "-Xms1g -Xmx2g"
    
  schema-registry-2:
    image: confluentinc/cp-schema-registry:7.5.0
    environment:
      SCHEMA_REGISTRY_HOST_NAME: schema-registry-2
      SCHEMA_REGISTRY_KAFKASTORE_BOOTSTRAP_SERVERS: kafka:9092
      SCHEMA_REGISTRY_LEADER_ELIGIBILITY: "true"
      SCHEMA_REGISTRY_HEAP_OPTS: "-Xms1g -Xmx2g"
    
  schema-registry-lb:
    image: nginx:alpine
    ports:
      - "8081:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
```

*Solution 4 : Monitoring des performances*

```java
// Métriques à surveiller
// - schema.registry.client.cache.hits : Taux de hit du cache
// - schema.registry.client.cache.misses : Requêtes au Registry
// - schema.registry.client.request.latency : Latence des requêtes

// Alerte si le taux de cache miss est élevé
if (cacheHitRate < 0.95) {
    alert("Schema Registry cache hit rate below 95%");
}
```

---

## III.6.8 Résumé

Ce chapitre a exploré les contrats de données comme fondement de toute architecture événementielle Kafka réussie. Les contrats définissent l'accord entre producteurs et consommateurs sur la structure, le format, la sémantique, et les règles d'évolution des messages échangés. Sans cette fondation, même l'architecture technique la plus élégante s'effondre sous le poids des incompatibilités et des erreurs de communication.

### Traduire les Besoins Métier en Schémas

La conception des schémas commence par la compréhension du domaine métier, pas par les considérations techniques. L'erreur classique de modéliser les événements d'après les structures de bases de données existantes mène à des schémas inadaptés à l'architecture événementielle.

**L'approche Domain-Driven Design (DDD)** et les ateliers **Event Storming** permettent d'identifier les événements métier significatifs — des faits immuables qui se sont produits dans le domaine. Ces événements guident directement la conception des topics et des schémas.

**Principes de conception retenus :**

*Autonomie des événements* : Chaque événement contient toutes les informations nécessaires pour être compris et traité indépendamment. Le consommateur ne doit pas faire de lookups externes pour comprendre l'événement. Ce principe réduit le couplage au prix d'événements plus volumineux.

*Nommage explicite et cohérent* : Les conventions de nommage (événements au passé, champs avec unités explicites, types différenciés) éliminent l'ambiguïté et facilitent la compréhension à travers les équipes.

*Versioning dès le début* : Tout schéma évoluera. Intégrer la notion de version dès la conception initiale, même avec une seule version, prépare l'évolution future.

*Séparation métadonnées/payload* : Les métadonnées techniques (timestamps, IDs de corrélation, source) séparées des données métier permettent une évolution indépendante et un traitement standardisé.

*Granularité appropriée* : L'approche hybride — événements fins pour l'audit et le replay détaillé, événements agrégés pour la consommation simple — offre le meilleur des deux mondes.

### Kafka et la Structure des Événements

**Kafka est agnostique au contenu des messages** — il transporte des bytes sans validation, parsing, ou transformation. Cette flexibilité est à la fois une force (tout format possible) et un risque (aucune protection native contre les schémas incompatibles).

**Choix du format de sérialisation — critères de décision :**

*JSON* : Lisible par les humains, universellement supporté, flexible. Mais verbeux, non typé, et sans validation native. Approprié pour le prototypage, les faibles volumes, et l'intégration avec des systèmes legacy.

*Avro* : Format binaire compact avec schéma intégré ou référencé, évolution native avec règles de compatibilité, performance excellente. Recommandé pour les environnements de production avec Schema Registry.

*Protobuf* : Très compact, performant, génération de code dans de nombreux langages. Approprié pour les environnements polyglotes et l'intégration avec gRPC.

Le choix dépend des exigences spécifiques. Pour la majorité des architectures événementielles d'entreprise, **Avro avec Schema Registry** offre le meilleur équilibre entre performance, sécurité, et gouvernance.

### Défis de l'Évolution des Schémas

Les schémas évoluent inévitablement — nouveaux champs, champs obsolètes, changements de types. Le défi est de gérer cette évolution sans briser les consommateurs existants ni interrompre le service.

**L'analyse d'impact** avant toute modification est critique : identifier les consommateurs, évaluer la nature du changement, déterminer l'ordre de déploiement, planifier la migration, et tester la compatibilité.

**Stratégies de compatibilité — choix et implications :**

*BACKWARD (recommandé par défaut)* : Les nouveaux schémas lisent les anciennes données. Les consommateurs sont mis à jour avant les producteurs. Protège contre les ruptures pour les messages en rétention.

*FORWARD* : Les anciens schémas lisent les nouvelles données. Les producteurs sont mis à jour avant les consommateurs. Utile quand le contrôle sur les consommateurs est limité.

*FULL* : Compatibilité dans les deux sens. Plus restrictif mais plus sûr pour les environnements avec déploiements non coordonnés.

*BACKWARD_TRANSITIVE / FULL_TRANSITIVE* : Vérifie la compatibilité avec TOUTES les versions précédentes. Essentiel pour les topics avec longue rétention.

**Patterns d'évolution éprouvés :**
- Ajout de champ avec valeur par défaut
- Dépréciation progressive avec période de transition
- Nouveau topic pour les breaking changes majeurs
- Versioning explicite dans le header pour la traçabilité

### Structure et Mapping des Événements

Un événement bien conçu comprend des couches distinctes avec des responsabilités claires :

*Header (métadonnées techniques)* : event_id, event_type, event_version, event_time, source, correlation_id, causation_id. Ces champs permettent la traçabilité, la déduplication, et le débogage distribué.

*Payload (données métier)* : Les données spécifiques à l'événement, structurées selon le domaine.

*Context (contexte d'exécution)* : tenant_id, region, environment. Informations sur le contexte de production.

**Le mapping entre systèmes** pose des défis récurrents : types de données (décimaux précis pour les montants, pas de float), énumérations évolutives, références vs. objets embarqués. L'approche hybride (ID pour les jointures futures + snapshot pour le contexte historique) offre souvent le meilleur compromis.

### Le Cas Customer 360

L'étude de cas Customer 360 illustre les défis réels des contrats de données à grande échelle dans un contexte d'intégration multi-sources.

**Défis documentés :**
- *Résolution d'identité* : Chaque système a sa propre notion d'identifiant client. La création d'une table de correspondance et d'un identifiant canonique est essentielle.
- *Schémas hétérogènes* : Les sources ont des représentations différentes. Le schéma canonique avec mapping explicite permet l'unification.
- *Cohérence temporelle* : Les événements arrivent dans le désordre. Le fenêtrage et les timestamps explicites gèrent cette complexité.
- *Qualité des données* : Les données sources contiennent des erreurs. La validation et l'enrichissement dans le pipeline sont nécessaires.

**Leçons clés du terrain :**
1. L'identity resolution consomme 40% de l'effort — investir dès le début
2. Les schémas canoniques évoluent — prévoir 2-3 itérations
3. La qualité des données est pire qu'attendu — construire des pipelines de nettoyage robustes
4. Le temps réel n'est pas toujours nécessaire — simplifier si une latence de minutes est acceptable
5. Documenter les mappings est critique pour l'onboarding et le débogage

### Schema Registry — Gouvernance des Schémas

Le Schema Registry Confluent est le composant central pour la gestion des schémas dans l'écosystème Kafka. Il résout le problème de l'agnosticité de Kafka envers le contenu en ajoutant une couche de gouvernance.

**Fonctionnalités clés :**
- Stockage centralisé et versionné des schémas
- Validation automatique de compatibilité avant enregistrement
- IDs de schéma pour la sérialisation efficace (pas de schéma dans chaque message)
- API REST pour l'intégration CI/CD
- Haute disponibilité via clustering

**Stratégies de nommage des sujets :**
- *TopicNameStrategy* (défaut) : Un schéma par topic, simple mais restrictif
- *RecordNameStrategy* : Plusieurs types par topic, organisés par nom de record
- *TopicRecordNameStrategy* : Combinaison des deux, isolation par topic ET par type

Le choix de la stratégie détermine les possibilités d'organisation des schémas et doit être cohérent à travers l'organisation.

### Problèmes Courants et Solutions Éprouvées

L'expérience collective des implémentations Kafka révèle des problèmes récurrents :

**Schema drift** : Les producteurs et consommateurs utilisent des versions incompatibles. *Solutions* : Désactiver l'auto-registration en production, validation CI/CD obligatoire, tests d'intégration avec le Schema Registry.

**Documentation insuffisante** : Les développeurs ne comprennent pas les schémas. *Solutions* : Documentation inline avec le champ `doc`, conventions de nommage explicites, génération automatique de documentation.

**Breaking changes accidentels** : Des modifications « anodines » cassent les consommateurs. *Solutions* : Compatibilité stricte (BACKWARD ou FULL), hooks pre-commit, tests de régression de schéma.

**Prolifération de schémas** : Des centaines de schémas sans gouvernance. *Solutions* : Catalogue centralisé avec métadonnées et propriétaires, processus de création avec approbation, nettoyage régulier.

**Performance du Schema Registry** : Latence élevée au démarrage ou sous charge. *Solutions* : Configuration optimale du cache client, pré-chargement des schémas, haute disponibilité du Registry.

### Principes Directeurs pour l'Architecte

1. **Les contrats sont des accords, pas des fichiers techniques.** Ils engagent producteurs et consommateurs sur la structure ET la sémantique des données. Un changement de schéma est un changement de contrat qui doit être traité avec la rigueur correspondante.

2. **La compatibilité est non négociable en production.** Les breaking changes doivent être exceptionnels, planifiés, et coordonnés. Le mode BACKWARD avec validation CI/CD est le minimum acceptable.

3. **Documenter est aussi important que définir.** Un schéma sans documentation est une dette technique. Les champs `doc` dans Avro, les conventions de nommage, et les catalogues de schémas sont des investissements essentiels.

4. **La gouvernance précède l'échelle.** Établir les processus, les conventions, et les responsabilités avant la prolifération des schémas. Corriger une gouvernance défaillante après coup est exponentiellement plus difficile.

5. **Automatiser la validation.** Les humains font des erreurs. La CI/CD détecte les incompatibilités. Les hooks pre-commit empêchent les erreurs d'atteindre le repository. L'automatisation est la seule approche scalable.

6. **Le Schema Registry est critique.** Il n'est pas optionnel pour les déploiements production d'entreprise. Sans lui, la gestion des schémas devient manuelle, fragile, et source d'incidents.

---

### Vers le Chapitre Suivant

Les contrats de données définissent la structure des messages échangés. Le chapitre suivant, « Patrons d'Interaction Kafka », explorera les patterns architecturaux qui utilisent ces messages : intégration par événements, data mesh, garanties de livraison, et coordination entre services dans les architectures distribuées.

---

*Volume III : Apache Kafka - Guide de l'Architecte*

*Chapitre III.6 — Contrats de Données*

*Monographie « L'Entreprise Agentique »*
