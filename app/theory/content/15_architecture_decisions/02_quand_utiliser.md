# 15.2 Quand Utiliser Chaque Type d'Intégration

## Résumé

Choisir entre intégration **Applications**, **Événements** ou **Données** n'est pas une question de préférence technique, mais de **besoin métier**. Cette section vous guide pour faire le bon choix selon le contexte.

## Points clés

- Chaque pilier résout un problème différent
- Le même flux peut combiner plusieurs piliers
- La latence et le couplage sont les critères principaux
- Il n'y a pas de solution universelle

---

## Arbre de Décision

```
┌──────────────────────────────────────────────────────────────────┐
│                QUEL TYPE D'INTÉGRATION CHOISIR ?                 │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Quel est le BESOIN PRINCIPAL ?                          │    │
│  └─────────────────────────────────────────────────────────┘    │
│         │                                                        │
│         ├──▶ "Je dois APPELER un service et attendre sa réponse"│
│         │         │                                              │
│         │         └──▶ 🔗 INTÉGRATION APPLICATIONS              │
│         │              └── REST API, gRPC, Gateway, BFF         │
│         │                                                        │
│         ├──▶ "Je dois RÉAGIR quand quelque chose se passe"      │
│         │         │                                              │
│         │         └──▶ ⚡ INTÉGRATION ÉVÉNEMENTS                │
│         │              └── Pub/Sub, Event Sourcing, Saga        │
│         │                                                        │
│         └──▶ "Je dois SYNCHRONISER ou ANALYSER des données"     │
│                   │                                              │
│                   └──▶ 📊 INTÉGRATION DONNÉES                   │
│                        └── ETL, CDC, Data Pipeline, MDM         │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

---

## 🔗 Intégration Applications

### Quand l'utiliser

| Situation | Exemple Assurance |
|-----------|-------------------|
| **Réponse immédiate requise** | Calcul de prime en temps réel |
| **Interface utilisateur** | Portail client, App mobile |
| **Requête/Réponse synchrone** | Vérification éligibilité |
| **Agrégation de données live** | Vue 360° client |
| **Partenaires externes (B2B)** | API courtiers |

### Patterns recommandés

```
BESOIN                              PATTERN
──────────────────────────────────────────────────────
Point d'entrée unique            → API Gateway
API par canal (mobile/web)       → BFF (Backend for Frontend)
Agrégation multi-sources         → API Composition
Isolation système legacy         → Anti-Corruption Layer
Migration progressive            → Strangler Fig
```

### Exemple : Devis en temps réel

```
Client App                     Écosystème Assurance
    │                                   │
    │ POST /quotes                      │
    │ { customer, vehicle, coverage }   │
    ├──────────────────────────────────▶│
    │                                   │ Gateway
    │                                   │    │
    │                                   │    ├──▶ QuoteEngine (calcul)
    │                                   │    ├──▶ CustomerHub (historique)
    │                                   │    └──▶ RatingAPI (tarif externe)
    │                                   │
    │ 200 OK                            │
    │ { quote_id, premium: 850€ }       │
    │◀──────────────────────────────────┤
    │                                   │

💡 Synchrone car le client attend la réponse pour continuer
```

---

## ⚡ Intégration Événements

### Quand l'utiliser

| Situation | Exemple Assurance |
|-----------|-------------------|
| **Découplage nécessaire** | Notifications multi-canaux |
| **Plusieurs consommateurs** | PolicyCreated → 5 services |
| **Workflow longue durée** | Processus sinistre (jours/semaines) |
| **Audit trail complet** | Historique modifications police |
| **Résilience aux pannes** | Retry automatique si service down |

### Patterns recommandés

```
BESOIN                              PATTERN
──────────────────────────────────────────────────────
Diffusion multi-consommateurs    → Pub/Sub
Traitement séquentiel garanti    → Message Queue
Historique complet               → Event Sourcing
Transactions distribuées         → Saga Pattern
Fiabilité publication            → Outbox Pattern
```

### Exemple : Création de police

```
PolicyAdmin                    Event Bus                    Consommateurs
    │                              │                              │
    │ Police créée POL-001         │                              │
    │ ─────────────────────────────┤                              │
    │                              │                              │
    │                              │ PolicyCreated                │
    │                              │ { policy_id, customer,       │
    │                              │   premium, coverages }       │
    │                              │ ────────────────────────────▶│
    │                              │                              │
    │                              │         ┌───────────────────┤
    │                              │         │ Billing: Facture  │
    │                              │         ├───────────────────┤
    │                              │         │ Notif: Email      │
    │                              │         ├───────────────────┤
    │                              │         │ Analytics: Stats  │
    │                              │         ├───────────────────┤
    │                              │         │ Audit: Log        │
    │                              │         └───────────────────┤

💡 Asynchrone car PolicyAdmin n'attend pas les consommateurs
```

---

## 📊 Intégration Données

### Quand l'utiliser

| Situation | Exemple Assurance |
|-----------|-------------------|
| **Volumes massifs** | Export 1M sinistres pour actuariat |
| **Analytics/BI** | Dashboard sinistralité mensuel |
| **Synchronisation batch** | Alimentation Data Warehouse |
| **Données de référence** | Client unique (MDM) |
| **Compliance/Archivage** | Rétention légale 10 ans |

### Patterns recommandés

```
BESOIN                              PATTERN
──────────────────────────────────────────────────────
Extraction périodique            → ETL Batch
Synchronisation temps réel       → CDC (Change Data Capture)
Client unique                    → Master Data Management
Traçabilité données              → Data Lineage
Qualité données                  → Data Quality Checks
```

### Exemple : Reporting sinistres

```
Sources                        Data Pipeline                    Cibles
    │                              │                              │
    │ Claims DB                    │                              │
    │ ─────────────────────────────┤ EXTRACT                      │
    │                              │ (nuit, 02:00)                │
    │                              │                              │
    │ Policy DB                    │                              │
    │ ─────────────────────────────┤ TRANSFORM                    │
    │                              │ (enrichissement,             │
    │                              │  calcul KPIs)                │
    │                              │                              │
    │                              │ LOAD                         │
    │                              │ ────────────────────────────▶│
    │                              │                              │ Data Warehouse
    │                              │                              │ ↓
    │                              │                              │ BI Reports
    │                              │                              │ ↓
    │                              │                              │ ML Models

💡 Batch car le reporting n'est pas temps réel
```

---

## Matrice de Comparaison

| Critère | 🔗 Applications | ⚡ Événements | 📊 Données |
|---------|-----------------|---------------|------------|
| **Latence** | Temps réel | Near real-time | Batch à temps réel |
| **Couplage** | Moyen-Fort | Faible | Variable |
| **Volume** | Transactionnel | Transactionnel | Massif |
| **Consistance** | Forte | Éventuelle | Éventuelle |
| **Complexité** | Moyenne | Haute | Haute |
| **Outils** | API Gateway, ESB | Kafka, RabbitMQ | Spark, Talend |

---

## Combinaison des Piliers

Un flux métier complet combine souvent les trois piliers :

```
┌─────────────────────────────────────────────────────────────────┐
│            FLUX COMPLET : SOUSCRIPTION ASSURANCE                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  PHASE 1 - 🔗 APPLICATIONS (temps réel, synchrone)              │
│  ═══════════════════════════════════════════════               │
│  Client ──▶ Gateway ──▶ QuoteEngine ──▶ RatingAPI              │
│  └── Calcul prime en temps réel, réponse immédiate             │
│                                                                 │
│  PHASE 2 - ⚡ ÉVÉNEMENTS (asynchrone, découplé)                 │
│  ═══════════════════════════════════════════════               │
│  PolicyAdmin publie "PolicyCreated"                            │
│  └── Billing, Notifications, Audit réagissent                  │
│                                                                 │
│  PHASE 3 - 📊 DONNÉES (batch, analytics)                        │
│  ═══════════════════════════════════════════════               │
│  CDC capture les changements                                   │
│  └── Data Warehouse, Reporting, ML alimentés                   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Questions de Décision

Posez-vous ces questions pour chaque flux d'intégration :

```
1. L'appelant attend-il une réponse immédiate ?
   OUI → 🔗 Applications
   NON → Continue...

2. Plusieurs systèmes doivent-ils réagir au même événement ?
   OUI → ⚡ Événements
   NON → Continue...

3. S'agit-il de synchroniser de gros volumes de données ?
   OUI → 📊 Données
   NON → Réévalue le besoin

4. Le processus peut-il tolérer une latence de quelques secondes ?
   OUI → ⚡ Événements possible
   NON → 🔗 Applications

5. Avez-vous besoin d'un audit trail complet ?
   OUI → ⚡ Event Sourcing
```

---

## Sandbox : Pratiquer

Le scénario **CROSS-04** vous fera implémenter un flux complet utilisant les trois piliers ensemble, vous permettant de ressentir concrètement quand et comment les combiner.
