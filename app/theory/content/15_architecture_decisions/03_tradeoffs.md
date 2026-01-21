# 15.3 Trade-offs et Compromis

## Résumé

Chaque décision d'architecture implique des **compromis**. Il n'existe pas de solution parfaite - seulement des solutions adaptées à un contexte donné. Cette section vous aide à identifier et évaluer consciemment les trade-offs de vos choix.

## Points clés

- Tout avantage a un coût associé
- Expliciter les compromis évite les mauvaises surprises
- Le contexte détermine quels compromis sont acceptables
- Documenter les décisions avec leurs trade-offs

---

## Les Trade-offs Fondamentaux

### Théorème CAP (Distributed Systems)

```
┌─────────────────────────────────────────────────────────────────┐
│                      THÉORÈME CAP                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Dans un système distribué, vous ne pouvez garantir que        │
│  DEUX des trois propriétés suivantes :                         │
│                                                                 │
│                    Consistency                                  │
│                    (Cohérence)                                  │
│                        ▲                                        │
│                       / \                                       │
│                      /   \                                      │
│                     /     \                                     │
│                    /       \                                    │
│                   /  CHOIX  \                                   │
│                  /     !!    \                                  │
│                 /             \                                 │
│                ▼               ▼                                │
│         Availability ◀──────▶ Partition                        │
│        (Disponibilité)      Tolerance                          │
│                            (Tolérance aux                       │
│                             partitions)                         │
│                                                                 │
│  CP : Cohérent mais peut être indisponible (ex: SGBD trad.)   │
│  AP : Disponible mais peut être incohérent (ex: DNS, cache)   │
│  CA : N'existe pas en distribué réel                          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Application en Assurance

| Système | Choix CAP | Justification |
|---------|-----------|---------------|
| **PolicyAdmin** | CP | Intégrité des polices critique |
| **Cache tarifs** | AP | Tarifs anciens OK temporairement |
| **Notifications** | AP | Mieux vaut notifier en retard que pas du tout |
| **Facturation** | CP | Montants doivent être exacts |

---

## Trade-offs par Pilier

### 🔗 Applications : Synchrone vs Asynchrone

```
┌─────────────────────────────────────────────────────────────────┐
│            SYNCHRONE vs ASYNCHRONE                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  SYNCHRONE (REST/gRPC)                                          │
│  ═══════════════════                                            │
│  ✅ Réponse immédiate                                           │
│  ✅ Gestion d'erreur simple                                     │
│  ✅ Debugging facile                                            │
│  ❌ Couplage temporel fort                                      │
│  ❌ Cascade de pannes possible                                  │
│  ❌ Scalabilité limitée                                         │
│                                                                 │
│  ASYNCHRONE (Message Queue)                                     │
│  ══════════════════════════                                     │
│  ✅ Découplage temporel                                         │
│  ✅ Résilience aux pannes                                       │
│  ✅ Meilleure scalabilité                                       │
│  ❌ Complexité accrue                                           │
│  ❌ Debugging difficile                                         │
│  ❌ Consistance éventuelle                                      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Exemple Assurance :**
- Calcul devis → **Synchrone** (client attend la réponse)
- Envoi documents → **Asynchrone** (peut attendre quelques secondes)

---

### ⚡ Événements : Event Notification vs Event-Carried State

```
┌─────────────────────────────────────────────────────────────────┐
│     EVENT NOTIFICATION vs EVENT-CARRIED STATE TRANSFER          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  EVENT NOTIFICATION                                             │
│  ══════════════════                                             │
│  { "type": "PolicyCreated", "policy_id": "POL-001" }           │
│                                                                 │
│  ✅ Messages légers (petite taille)                             │
│  ✅ Source de vérité unique (l'appelant fetch les détails)     │
│  ❌ Couplage : consommateur doit appeler l'API                 │
│  ❌ Plus de latence (aller-retour supplémentaire)              │
│                                                                 │
│  ──────────────────────────────────────────────────────────────│
│                                                                 │
│  EVENT-CARRIED STATE TRANSFER                                   │
│  ════════════════════════════                                   │
│  { "type": "PolicyCreated",                                     │
│    "policy_id": "POL-001",                                     │
│    "customer": { "id": "C001", "name": "Dupont" },             │
│    "premium": 850,                                             │
│    "coverages": ["RC", "VOL"] }                                │
│                                                                 │
│  ✅ Consommateur autonome (pas d'appel API)                    │
│  ✅ Moins de latence                                           │
│  ❌ Messages volumineux                                        │
│  ❌ Données potentiellement obsolètes                          │
│  ❌ Duplication de données                                      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Recommandation Assurance :**
- Audit/Compliance → **Event-Carried** (historique complet)
- Notifications → **Notification** (juste besoin de l'ID)

---

### 📊 Données : Batch vs Streaming

```
┌─────────────────────────────────────────────────────────────────┐
│                    BATCH vs STREAMING                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  BATCH (ETL traditionnel)                                       │
│  ════════════════════════                                       │
│  ✅ Simple à implémenter                                        │
│  ✅ Coût infrastructure moindre                                 │
│  ✅ Facile à debugger (données statiques)                      │
│  ❌ Latence élevée (heures/jours)                              │
│  ❌ Données obsolètes entre exécutions                          │
│  ❌ Reprise coûteuse si erreur                                 │
│                                                                 │
│  ──────────────────────────────────────────────────────────────│
│                                                                 │
│  STREAMING (CDC, temps réel)                                    │
│  ═══════════════════════════                                    │
│  ✅ Données fraîches (secondes/minutes)                        │
│  ✅ Détection anomalies en temps réel                          │
│  ✅ Reprise incrémentale facile                                │
│  ❌ Complexité opérationnelle                                  │
│  ❌ Coût infrastructure plus élevé                              │
│  ❌ Debugging plus complexe                                    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Choix Assurance :**
- Reporting actuariat mensuel → **Batch** (latence acceptable)
- Dashboard fraude → **Streaming** (détection temps réel critique)

---

## Matrice des Compromis

### Simplicité vs Flexibilité

| Approche | Simplicité | Flexibilité | Cas d'usage |
|----------|------------|-------------|-------------|
| Monolithe | ⭐⭐⭐⭐⭐ | ⭐⭐ | MVP, équipe réduite |
| Microservices | ⭐⭐ | ⭐⭐⭐⭐⭐ | Scale-up, équipes multiples |
| Modular Monolith | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | Évolution progressive |

### Consistance vs Disponibilité

| Approche | Consistance | Disponibilité | Cas d'usage |
|----------|-------------|---------------|-------------|
| Transaction ACID | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | Paiements, comptabilité |
| Eventual Consistency | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | Social, notifications |
| Saga avec compensation | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | Workflows distribués |

### Performance vs Maintenabilité

| Approche | Performance | Maintenabilité | Cas d'usage |
|----------|-------------|----------------|-------------|
| Code optimisé | ⭐⭐⭐⭐⭐ | ⭐⭐ | Systèmes critiques |
| Code lisible | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | Applications métier |
| Framework standard | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | Équilibre commun |

---

## Documenter les Trade-offs

### Format ADR (Architecture Decision Record)

```markdown
# ADR-003: Choix Pub/Sub pour les événements métier

## Contexte
Notre écosystème doit propager les événements métier (PolicyCreated,
ClaimSubmitted) vers plusieurs systèmes consommateurs.

## Décision
Nous utilisons le pattern Pub/Sub avec un broker de messages centralisé.

## Conséquences

### Acceptées (trade-offs conscients)
- ❌ Complexité opérationnelle accrue (broker à maintenir)
- ❌ Consistance éventuelle (délai de propagation)
- ❌ Debugging plus difficile (flux asynchrone)

### Bénéfices attendus
- ✅ Découplage total entre producteurs et consommateurs
- ✅ Scalabilité horizontale des consommateurs
- ✅ Ajout facile de nouveaux consommateurs
- ✅ Résilience aux pannes (retry automatique)

## Alternatives considérées
1. Appels REST directs : Rejeté car couplage trop fort
2. Webhook : Rejeté car gestion des erreurs complexe
3. Polling : Rejeté car inefficace et latence élevée

## Date: 2024-01-15
## Auteur: Équipe Architecture
```

---

## Questions à se Poser

Avant chaque décision, posez-vous :

```
1. Quel problème cette solution résout-elle vraiment ?
   → Évite les solutions à la recherche d'un problème

2. Quels compromis acceptons-nous consciemment ?
   → Les documenter explicitement

3. Comment ce choix évoluera-t-il dans 2-3 ans ?
   → Anticiper la dette technique

4. L'équipe a-t-elle les compétences pour opérer cette solution ?
   → Capacité organisationnelle

5. Pouvons-nous revenir en arrière si ce choix s'avère mauvais ?
   → Réversibilité de la décision
```

---

## Sandbox : Expérimenter

Le scénario **CROSS-04** vous confrontera à des choix où vous devrez consciemment arbitrer entre :
- Cohérence forte vs disponibilité
- Simplicité vs scalabilité
- Performance vs maintenabilité

Vous documenterez vos décisions avec leurs trade-offs explicites.
