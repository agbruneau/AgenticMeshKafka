# 15.1 Orchestration vs Choreography

## Résumé

La coordination des systèmes distribués peut suivre deux philosophies fondamentalement différentes : l'**orchestration** centralisée ou la **chorégraphie** décentralisée. Ce choix architectural impacte profondément la maintenabilité, la scalabilité et la résilience de votre écosystème.

## Points clés

- **Orchestration** : Un chef d'orchestre central coordonne tous les participants
- **Chorégraphie** : Chaque participant connaît sa partition et réagit aux événements
- Le choix dépend du contexte métier, pas d'une règle universelle
- Les deux approches peuvent coexister dans un même écosystème

---

## Orchestration Centralisée

### Principe

Un composant central (orchestrateur) contrôle explicitement le flux d'exécution, appelant chaque service dans l'ordre requis et gérant les résultats.

```
┌─────────────────────────────────────────────────────────────┐
│                    ORCHESTRATION                             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│                    ┌─────────────┐                           │
│                    │ Orchestrator│                           │
│                    │   (Chef)    │                           │
│                    └──────┬──────┘                           │
│                           │                                  │
│         ┌─────────────────┼─────────────────┐               │
│         │                 │                 │               │
│         ▼                 ▼                 ▼               │
│    ┌─────────┐      ┌─────────┐      ┌─────────┐           │
│    │Service A│      │Service B│      │Service C│           │
│    │ (Quote) │      │(Policy) │      │(Billing)│           │
│    └─────────┘      └─────────┘      └─────────┘           │
│                                                              │
│    L'orchestrateur appelle chaque service séquentiellement  │
│    et gère le flux complet de bout en bout                  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Cas d'usage en assurance

**Processus de souscription orchestré :**

```
ORCHESTRATEUR: SouscriptionWorkflow

Étape 1: Valider le devis
    → Appeler QuoteEngine.validate(quote_id)
    ← Résultat: devis valide

Étape 2: Vérifier le client
    → Appeler CustomerHub.checkEligibility(customer_id)
    ← Résultat: client éligible

Étape 3: Créer la police
    → Appeler PolicyAdmin.create(policy_data)
    ← Résultat: police POL-2024-001

Étape 4: Générer la facture
    → Appeler Billing.createInvoice(policy_id, amount)
    ← Résultat: facture générée

Étape 5: Notifier le client
    → Appeler Notifications.send(customer_id, "POLICY_CREATED")
    ← Résultat: notification envoyée

FIN: Souscription complète
```

### Avantages

| Avantage | Description |
|----------|-------------|
| **Visibilité** | Flux clairement défini et traçable |
| **Contrôle** | Gestion centralisée des erreurs et compensations |
| **Debugging** | Point unique pour diagnostiquer les problèmes |
| **Transactions** | Plus facile d'implémenter des transactions distribuées |

### Inconvénients

| Inconvénient | Description |
|--------------|-------------|
| **Point de défaillance unique** | L'orchestrateur devient critique |
| **Couplage** | L'orchestrateur connaît tous les services |
| **Bottleneck** | Peut devenir un goulot d'étranglement |
| **Évolution** | Modifications centralisées requises |

---

## Chorégraphie Décentralisée

### Principe

Chaque service réagit aux événements qui le concernent et publie ses propres événements. Il n'y a pas de coordinateur central - les services "dansent" ensemble.

```
┌─────────────────────────────────────────────────────────────┐
│                    CHORÉGRAPHIE                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│                    ┌─────────────┐                           │
│    PolicyCreated   │  Event Bus  │   InvoiceGenerated       │
│    ─────────────▶  │   (Topics)  │  ◀─────────────          │
│                    └──────┬──────┘                           │
│                           │                                  │
│         ┌─────────────────┼─────────────────┐               │
│         │                 │                 │               │
│         ▼                 ▼                 ▼               │
│    ┌─────────┐      ┌─────────┐      ┌─────────┐           │
│    │ Billing │      │  Notif  │      │  Audit  │           │
│    │ Service │      │ Service │      │ Service │           │
│    └─────────┘      └─────────┘      └─────────┘           │
│                                                              │
│    Chaque service écoute les événements qui le concernent   │
│    et réagit de manière autonome                            │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Cas d'usage en assurance

**Même processus de souscription, version chorégraphiée :**

```
1. QuoteEngine publie: QuoteValidated { quote_id, customer_id }
     ↓
2. CustomerHub écoute QuoteValidated
   → Vérifie l'éligibilité
   → Publie: CustomerVerified { customer_id, eligible: true }
     ↓
3. PolicyAdmin écoute CustomerVerified
   → Crée la police
   → Publie: PolicyCreated { policy_id, customer_id, amount }
     ↓
4. Billing écoute PolicyCreated
   → Génère la facture
   → Publie: InvoiceGenerated { invoice_id, policy_id }
     ↓
5. Notifications écoute PolicyCreated
   → Envoie la notification
   → Publie: NotificationSent { customer_id, type: "POLICY" }
```

### Avantages

| Avantage | Description |
|----------|-------------|
| **Découplage** | Services indépendants et autonomes |
| **Scalabilité** | Pas de point central limitant |
| **Résilience** | Pas de single point of failure |
| **Évolution** | Ajout facile de nouveaux consommateurs |

### Inconvénients

| Inconvénient | Description |
|--------------|-------------|
| **Complexité** | Flux difficile à suivre et debugger |
| **Consistance** | Garantir l'ordre et la complétude est complexe |
| **Monitoring** | Nécessite un tracing distribué |
| **Transactions** | Compensation distribuée plus complexe |

---

## Critères de Décision

### Quand choisir l'Orchestration ?

```
✅ ORCHESTRATION si:

□ Processus métier avec ordre strict obligatoire
  → Ex: Souscription avec validation règlementaire séquentielle

□ Besoin de transactions avec rollback
  → Ex: Saga de paiement avec compensation automatique

□ Visibilité métier importante
  → Ex: Dashboard de suivi des dossiers en cours

□ Équipe centralisée gérant le processus
  → Ex: Équipe dédiée "Parcours Client"

□ Nombre limité de services (< 10)
  → La complexité reste gérable
```

### Quand choisir la Chorégraphie ?

```
✅ CHORÉGRAPHIE si:

□ Services très indépendants
  → Ex: Notifications, Audit, Analytics

□ Événements consommés par plusieurs services
  → Ex: PolicyCreated déclenche 5+ actions différentes

□ Équipes autonomes (Domain Teams)
  → Chaque équipe gère son domaine

□ Scalabilité prioritaire
  → Millions d'événements/jour

□ Tolérance aux délais
  → Eventual consistency acceptable
```

---

## Approche Hybride

En pratique, les deux approches coexistent souvent :

```
┌─────────────────────────────────────────────────────────────┐
│              ARCHITECTURE HYBRIDE ASSURANCE                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ORCHESTRÉ (critique, transactionnel)                       │
│  ═══════════════════════════════════                        │
│                                                              │
│  ┌─────────────────────────────────────────┐                │
│  │ Saga Souscription                        │                │
│  │ Quote → Verify → Policy → Billing        │                │
│  │ (avec compensation en cas d'erreur)      │                │
│  └─────────────────────────┬───────────────┘                │
│                            │                                 │
│                            │ publie PolicyCreated            │
│                            ▼                                 │
│  CHORÉGRAPHIÉ (découplé, scalable)                          │
│  ══════════════════════════════════                         │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │                    Event Bus                            │ │
│  └──────┬──────────────┬──────────────┬──────────────────┘ │
│         │              │              │                     │
│         ▼              ▼              ▼                     │
│    Notifications    Analytics     Compliance                │
│    (email, SMS)     (reporting)   (audit trail)            │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Principe :** Orchestrer le "chemin critique" transactionnel, chorégraphier les "effets de bord" découplés.

---

## Sandbox : Expérimenter

Dans le scénario **CROSS-04**, vous implémenterez les deux approches :
- Orchestration de la saga de souscription
- Chorégraphie pour les notifications et l'audit

Cette expérience pratique vous permettra de ressentir les différences concrètes entre les deux philosophies.
