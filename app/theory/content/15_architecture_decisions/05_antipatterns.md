# 15.5 Anti-patterns à Éviter

## Résumé

Un **anti-pattern** est une solution qui semble bonne mais qui crée plus de problèmes qu'elle n'en résout. Reconnaître ces pièges vous permet de les éviter avant qu'ils ne deviennent coûteux à corriger.

## Points clés

- Les anti-patterns sont des "bonnes idées" qui ont mal tourné
- Ils émergent souvent de décisions prises sous pression
- Les reconnaître tôt évite une dette technique massive
- Chaque anti-pattern a un remède (refactoring pattern)

---

## Anti-patterns d'Intégration

### 1. Le Plat de Spaghetti (Spaghetti Integration)

```
┌─────────────────────────────────────────────────────────────────┐
│              ❌ ANTI-PATTERN: SPAGHETTI                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│    ┌─────┐     ┌─────┐     ┌─────┐     ┌─────┐                │
│    │  A  │◀───▶│  B  │◀───▶│  C  │◀───▶│  D  │                │
│    └──┬──┘     └──┬──┘     └──┬──┘     └──┬──┘                │
│       │           │           │           │                    │
│       └───────────┼───────────┼───────────┘                    │
│                   │           │                                │
│       ┌───────────┘           └───────────┐                    │
│       │                                   │                    │
│       ▼                                   ▼                    │
│    ┌─────┐                             ┌─────┐                │
│    │  E  │◀───────────────────────────▶│  F  │                │
│    └─────┘                             └─────┘                │
│                                                                 │
│    Chaque service appelle directement tous les autres          │
│    → Maintenance impossible, effet papillon garanti            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Symptômes :**
- Modifier un service casse 5 autres
- Personne ne comprend le flux complet
- Tests d'intégration impossibles
- Déploiements coordonnés obligatoires

**Remède :**
- Introduire un Event Bus pour découpler
- Utiliser une API Gateway comme point d'entrée
- Définir des contrats d'interface clairs

---

### 2. Le God Service (Service Omniscient)

```
┌─────────────────────────────────────────────────────────────────┐
│              ❌ ANTI-PATTERN: GOD SERVICE                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│                    ┌─────────────────────┐                      │
│                    │                     │                      │
│                    │    GOD SERVICE      │                      │
│                    │                     │                      │
│                    │  - Gère les clients │                      │
│                    │  - Gère les polices │                      │
│                    │  - Gère les sinistres                      │
│                    │  - Gère la facturation                     │
│                    │  - Gère les notifs  │                      │
│                    │  - Gère les docs    │                      │
│                    │  - Gère l'audit     │                      │
│                    │  - Gère le reporting│                      │
│                    │                     │                      │
│                    └─────────────────────┘                      │
│                                                                 │
│    Un service qui fait TOUT                                    │
│    → Monolithe déguisé, impossible à faire évoluer             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Symptômes :**
- Un service avec 50+ endpoints
- Équipe de 20 personnes sur un seul repo
- Déploiements de 2 heures
- Impossible de scaler une partie sans tout scaler

**Remède :**
- Identifier les bounded contexts (DDD)
- Extraire des microservices par domaine
- Strangler Fig pour migration progressive

---

### 3. Le Distributed Monolith

```
┌─────────────────────────────────────────────────────────────────┐
│              ❌ ANTI-PATTERN: DISTRIBUTED MONOLITH              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  "On a des microservices !"                                    │
│                                                                 │
│    ┌─────┐     ┌─────┐     ┌─────┐                            │
│    │  A  │────▶│  B  │────▶│  C  │                            │
│    └─────┘     └─────┘     └─────┘                            │
│       │           │           │                                │
│       │           │           │                                │
│       └─────┬─────┴─────┬─────┘                                │
│             │           │                                      │
│             ▼           ▼                                      │
│    ┌─────────────────────────────┐                            │
│    │      BASE DE DONNÉES        │                            │
│    │         PARTAGÉE            │                            │
│    └─────────────────────────────┘                            │
│                                                                 │
│    Microservices en apparence, monolithe en pratique           │
│    → Tous les inconvénients, aucun avantage                    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Symptômes :**
- Base de données partagée entre services
- Déploiements synchronisés obligatoires
- Un service down = tout down
- Schéma DB modifié = tous les services impactés

**Remède :**
- Database per Service
- Communication via événements
- API comme seule interface

---

### 4. Le Chatty Service (Bavardage Excessif)

```
┌─────────────────────────────────────────────────────────────────┐
│              ❌ ANTI-PATTERN: CHATTY SERVICE                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│    Client                         Services                     │
│       │                                                        │
│       │──▶ GET /customer/C001                                  │
│       │◀── { name: "Dupont" }                                  │
│       │                                                        │
│       │──▶ GET /customer/C001/address                          │
│       │◀── { city: "Paris" }                                   │
│       │                                                        │
│       │──▶ GET /customer/C001/policies                         │
│       │◀── [{ id: "P001" }]                                    │
│       │                                                        │
│       │──▶ GET /policy/P001                                    │
│       │◀── { premium: 850 }                                    │
│       │                                                        │
│       │──▶ GET /policy/P001/claims                             │
│       │◀── [{ id: "CLM-001" }]                                 │
│       │                                                        │
│       │──▶ GET /claim/CLM-001                                  │
│       │◀── { status: "OPEN" }                                  │
│       │                                                        │
│    6 appels pour afficher une page !                           │
│    → Latence cumulée, réseau saturé                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Symptômes :**
- 10+ appels API pour une page
- Latence utilisateur > 2 secondes
- Bande passante réseau saturée
- Erreurs en cascade (si un appel échoue)

**Remède :**
- API Composition (agrégation côté serveur)
- BFF (Backend for Frontend)
- GraphQL pour requêtes flexibles

---

### 5. Le Fire and Forget (Tir et Oubli)

```
┌─────────────────────────────────────────────────────────────────┐
│              ❌ ANTI-PATTERN: FIRE AND FORGET                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│    Service A                          Message Queue            │
│       │                                    │                   │
│       │ publish("important_event")         │                   │
│       │───────────────────────────────────▶│                   │
│       │                                    │                   │
│       │ return "OK, published!"            │                   │
│       │                                    │   ?               │
│       │                                    │   └── Consommé ?  │
│       │                                    │   └── Erreur ?    │
│       │                                    │   └── Perdu ?     │
│                                            │                   │
│    Aucune vérification que le message a été traité             │
│    → Données perdues silencieusement                           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Symptômes :**
- Messages disparus sans trace
- Incohérences entre systèmes
- "Je suis sûr d'avoir envoyé l'événement"
- Debugging impossible

**Remède :**
- Outbox Pattern (atomicité DB + message)
- Dead Letter Queue pour les échecs
- Monitoring des queues
- Idempotence côté consommateur

---

### 6. Le Golden Hammer (Marteau d'Or)

```
┌─────────────────────────────────────────────────────────────────┐
│              ❌ ANTI-PATTERN: GOLDEN HAMMER                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│    "Tout est un clou quand on a un marteau"                    │
│                                                                 │
│    Problème: Calcul temps réel         → "On utilise Kafka !"  │
│    Problème: Stockage fichiers         → "On utilise Kafka !"  │
│    Problème: Cache distribué           → "On utilise Kafka !"  │
│    Problème: Base de données           → "On utilise Kafka !"  │
│    Problème: Envoi d'emails            → "On utilise Kafka !"  │
│                                                                 │
│    Une technologie utilisée pour TOUT                          │
│    → Solution inadaptée à de nombreux cas                      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Symptômes :**
- Une seule techno pour tous les problèmes
- Complexité artificielle pour des cas simples
- "On a toujours fait comme ça"
- Solutions qui ne correspondent pas au besoin

**Remède :**
- Évaluer chaque besoin individuellement
- Polyglot persistence (bonne techno pour chaque usage)
- Accepter d'apprendre de nouvelles solutions

---

## Anti-patterns Spécifiques par Pilier

### 🔗 Applications

| Anti-pattern | Description | Remède |
|--------------|-------------|--------|
| **API Versioning Hell** | 10 versions d'API en production | Deprecation policy, migration forcée |
| **Anemic API** | GET /doSomething, GET /process | RESTful design, ressources |
| **Mega Payload** | Retourner 1000 champs par requête | Pagination, filtering, GraphQL |

### ⚡ Événements

| Anti-pattern | Description | Remède |
|--------------|-------------|--------|
| **Event Storming** | 1000 événements par seconde mal gérés | Batching, backpressure |
| **Temporal Coupling** | Événements dépendants de l'ordre | Idempotence, timestamps |
| **Schema Evolution Nightmare** | Changer un événement casse tout | Event versioning, upcasting |

### 📊 Données

| Anti-pattern | Description | Remède |
|--------------|-------------|--------|
| **ETL Spaghetti** | 500 jobs ETL interdépendants | Data lineage, orchestration |
| **Copy-Paste Data** | Données dupliquées partout | MDM, source of truth |
| **Big Ball of Mud** | Data lake = data swamp | Data governance, cataloging |

---

## Comment Détecter les Anti-patterns ?

### Code Smells d'Intégration

```
🚨 ALERTES À SURVEILLER :

□ "On ne peut pas déployer A sans déployer B"
  → Couplage trop fort

□ "Personne ne sait qui consomme cet événement"
  → Documentation manquante

□ "On attend 30 secondes la réponse"
  → Chatty service ou timeout mal configuré

□ "Le bug n'est reproductible qu'en prod"
  → Manque de tests d'intégration

□ "On ne peut pas scaler ce service seul"
  → Distributed monolith
```

### Questions de Revue d'Architecture

```
1. Si ce service tombe, combien d'autres sont impactés ?
   > 3 → Couplage problématique

2. Combien d'appels réseau pour une opération métier ?
   > 5 → Chatty service probable

3. Peut-on déployer ce service indépendamment ?
   NON → Distributed monolith

4. Un nouveau développeur comprend-il le flux en < 1h ?
   NON → Documentation ou architecture à revoir
```

---

## Sandbox : Identifier les Anti-patterns

Le scénario **CROSS-04** vous présentera délibérément des anti-patterns que vous devrez identifier et corriger :
- Flux spaghetti à démêler
- Service trop bavard à optimiser
- Fire-and-forget à sécuriser

Cette expérience vous entraînera à reconnaître ces pièges dans vos propres projets.
