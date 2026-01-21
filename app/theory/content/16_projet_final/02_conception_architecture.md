# 16.2 Conception de l'Architecture

## Résumé

Avant d'implémenter, il faut concevoir. Cette section vous guide dans la création d'une architecture d'intégration cohérente en utilisant les trois piliers de manière complémentaire.

## Points clés

- Partir des besoins métier, pas des technologies
- Un pilier par type de besoin
- Documenter chaque décision avec un ADR
- L'architecture émerge des contraintes

---

## Méthodologie de Conception

### Étape 1 : Cartographie des Flux

Identifiez tous les flux d'intégration et leurs caractéristiques :

```
┌─────────────────────────────────────────────────────────────────┐
│              CARTOGRAPHIE DES FLUX                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  FLUX              LATENCE    CONSUMERS   VOLUME   PILIER      │
│  ═══════════════════════════════════════════════════════════   │
│                                                                 │
│  Calcul devis      < 3s       1           Trans.   🔗 APP      │
│  Création police   < 10s      1           Trans.   🔗 APP      │
│  Notif création    < 1min     5+          Trans.   ⚡ EVT      │
│  Vue 360° client   < 2s       1           Trans.   🔗 APP      │
│  Audit changes     N/A        1           Stream   ⚡ EVT      │
│  Reporting DWH     Nuit       1           Massif   📊 DATA     │
│  CDC polices       < 30s      3           Stream   📊 DATA     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Étape 2 : Affectation des Piliers

Pour chaque flux, appliquez l'arbre de décision :

```
FLUX: Calcul de devis
├── Réponse immédiate requise ? OUI → 🔗 Applications
├── Pattern: API Gateway + REST
└── Services impliqués: Quote Engine, Rating API

FLUX: Notification création police
├── Réponse immédiate requise ? NON
├── Plusieurs consommateurs ? OUI → ⚡ Événements
├── Pattern: Pub/Sub
└── Consommateurs: Billing, Notifications, Documents, Audit

FLUX: Reporting actuariat
├── Volume massif ? OUI → 📊 Données
├── Temps réel requis ? NON
├── Pattern: ETL batch + CDC
└── Cibles: Data Warehouse, BI Tools
```

---

## Architecture Cible

### Vue d'Ensemble

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    ARCHITECTURE CIBLE ASSURPLUS                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  CANAUX                                                                  │
│  ═══════                                                                 │
│  ┌─────────┐   ┌─────────┐   ┌─────────┐                               │
│  │ Portail │   │  App    │   │Courtiers│                               │
│  │   Web   │   │ Mobile  │   │   B2B   │                               │
│  └────┬────┘   └────┬────┘   └────┬────┘                               │
│       │             │             │                                      │
│       └──────────┬──┴─────────────┘                                      │
│                  ▼                                                       │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                      🔗 API GATEWAY                                │  │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐              │  │
│  │  │ Routing │  │  Auth   │  │  Rate   │  │ Circuit │              │  │
│  │  │         │  │  JWT    │  │  Limit  │  │ Breaker │              │  │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘              │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                  │                                                       │
│       ┌──────────┼──────────┬───────────┐                               │
│       ▼          ▼          ▼           ▼                               │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌───────────┐                     │
│  │   BFF   │ │   BFF   │ │   API   │ │    API    │                     │
│  │ Mobile  │ │ Broker  │ │ Compos. │ │  Direct   │                     │
│  └────┬────┘ └────┬────┘ └────┬────┘ └─────┬─────┘                     │
│       │          │           │             │                            │
│       └──────────┴───────────┴─────────────┘                            │
│                              │                                          │
│  SERVICES MÉTIER             │                                          │
│  ═══════════════             │                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                                                                  │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐            │   │
│  │  │  Quote  │  │ Policy  │  │ Claims  │  │ Billing │            │   │
│  │  │ Engine  │  │  Admin  │  │  Mgmt   │  │ System  │            │   │
│  │  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘            │   │
│  │       │            │            │            │                   │   │
│  └───────┼────────────┼────────────┼────────────┼───────────────────┘   │
│          │            │            │            │                        │
│          └────────────┴────────────┴────────────┘                        │
│                              │                                          │
│                              ▼                                          │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                      ⚡ EVENT BUS                                  │  │
│  │  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐      │  │
│  │  │ topic.policies │  │ topic.claims   │  │ topic.billing  │      │  │
│  │  └───────┬────────┘  └───────┬────────┘  └───────┬────────┘      │  │
│  │          │                   │                   │                │  │
│  │          └───────────────────┴───────────────────┘                │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                              │                                          │
│       ┌──────────────────────┼──────────────────────┐                   │
│       ▼                      ▼                      ▼                   │
│  ┌─────────┐           ┌─────────┐           ┌─────────┐               │
│  │  Notif  │           │  Audit  │           │Documents│               │
│  │ Service │           │  Trail  │           │  Mgmt   │               │
│  └─────────┘           └─────────┘           └─────────┘               │
│                                                                          │
│  📊 DATA PLATFORM                                                        │
│  ═════════════════                                                       │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐              │  │
│  │  │   CDC   │──│   ETL   │──│   DWH   │──│   BI    │              │  │
│  │  │ Capture │  │Pipeline │  │         │  │ Reports │              │  │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘              │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Décisions d'Architecture (ADR)

### ADR-001 : API Gateway Centralisée

```markdown
# ADR-001: API Gateway comme point d'entrée unique

## Statut
Accepté

## Contexte
AssurPlus expose des services à 3 canaux différents (Web, Mobile, B2B)
avec des besoins communs de sécurité, rate limiting et observabilité.

## Décision
Implémenter une API Gateway centralisée qui :
- Route vers les services backend
- Applique l'authentification JWT
- Implémente le rate limiting par client
- Intègre un circuit breaker

## Conséquences
✅ Point d'entrée unique et sécurisé
✅ Observabilité centralisée
❌ Single point of failure (mitigé par redondance)
❌ Latence additionnelle (+10ms)
```

### ADR-002 : Pub/Sub pour les Événements Métier

```markdown
# ADR-002: Pub/Sub pour la propagation des événements

## Statut
Accepté

## Contexte
La création d'une police doit déclencher 5+ actions :
facturation, notifications, documents, audit, reporting.
L'approche point-à-point créerait un couplage fort.

## Décision
Utiliser le pattern Pub/Sub avec des topics par entité :
- topic.policies (PolicyCreated, PolicyCancelled, ...)
- topic.claims (ClaimSubmitted, ClaimApproved, ...)
- topic.billing (InvoiceGenerated, PaymentReceived, ...)

## Conséquences
✅ Découplage total entre producteurs et consommateurs
✅ Ajout facile de nouveaux consommateurs
❌ Consistance éventuelle (délai de propagation)
❌ Debugging plus complexe (tracing distribué requis)
```

### ADR-003 : Saga Orchestrée pour la Souscription

```markdown
# ADR-003: Saga orchestrée pour le parcours de souscription

## Statut
Accepté

## Contexte
La souscription implique plusieurs services :
Quote → Customer → Policy → Billing
Chaque étape peut échouer et nécessite compensation.

## Décision
Implémenter une Saga orchestrée avec :
- Orchestrateur central gérant le flux
- Compensation automatique en cas d'échec
- État persisté pour reprise après panne

## Conséquences
✅ Visibilité complète du processus
✅ Compensation automatisée
❌ Orchestrateur = single point of failure
❌ Couplage avec tous les services participants
```

### ADR-004 : CDC pour le Reporting

```markdown
# ADR-004: CDC pour l'alimentation du Data Warehouse

## Statut
Accepté

## Contexte
Le reporting actuariat nécessite des données fraîches
(< 24h) sans impacter les performances des systèmes sources.

## Décision
Implémenter CDC (Change Data Capture) :
- Capture des changements en temps réel
- Pipeline de transformation
- Chargement incrémental dans le DWH

## Conséquences
✅ Données quasi temps réel
✅ Pas d'impact sur les sources
❌ Complexité opérationnelle
❌ Gestion du schéma evolution
```

---

## Stratégie de Résilience

### Patterns par Service

| Service | Circuit Breaker | Retry | Fallback | Timeout |
|---------|----------------|-------|----------|---------|
| **Quote Engine** | ✅ | 3x | Cache tarifs | 3s |
| **Policy Admin** | ✅ | 2x | Mode dégradé | 5s |
| **Rating API** | ✅ | 3x | Tarifs par défaut | 2s |
| **Claims** | ✅ | 3x | Queue async | 5s |
| **Billing** | ✅ | 5x | Retry later | 10s |

### Chaîne de Résilience

```
Client Request
      │
      ▼
┌─────────────────┐
│    Gateway      │
│  ┌───────────┐  │
│  │ Timeout   │  │──▶ 504 Gateway Timeout
│  │   10s     │  │
│  └─────┬─────┘  │
│        │        │
│  ┌─────▼─────┐  │
│  │ Circuit   │  │──▶ 503 Service Unavailable
│  │ Breaker   │  │    (avec fallback si dispo)
│  └─────┬─────┘  │
│        │        │
│  ┌─────▼─────┐  │
│  │   Retry   │  │──▶ Retry avec backoff
│  │   3x      │  │    (exponential)
│  └─────┬─────┘  │
│        │        │
└────────┼────────┘
         │
         ▼
   Service Backend
```

---

## Observabilité

### Les 3 Piliers

```
LOGS (Structured)
═════════════════
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "service": "policy-admin",
  "trace_id": "abc123",
  "span_id": "def456",
  "message": "Policy created",
  "policy_id": "POL-001",
  "customer_id": "C001"
}

METRICS
═══════
• policy_created_total (counter)
• quote_calculation_duration_seconds (histogram)
• circuit_breaker_state (gauge)
• active_subscriptions (gauge)

TRACES (Distributed)
════════════════════
Gateway → BFF → PolicyAdmin → Billing → Notifications
   │        │         │          │           │
   └────────┴─────────┴──────────┴───────────┘
                 trace_id: abc123
```

### Dashboard Principal

```
┌─────────────────────────────────────────────────────────────────┐
│                    DASHBOARD OPÉRATIONS                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  SANTÉ SERVICES                MÉTRIQUES TEMPS RÉEL            │
│  ═══════════════                ════════════════════            │
│                                                                 │
│  Quote Engine    🟢            Devis/min:        125           │
│  Policy Admin    🟢            Polices/heure:    45            │
│  Claims Mgmt     🟡 (degraded) Sinistres/jour:   23            │
│  Billing         🟢                                            │
│  Rating API      🔴 (circuit open)                             │
│                                                                 │
│  ERREURS (24h)                 LATENCE P99                     │
│  ═════════════                 ═══════════                     │
│                                                                 │
│  500 errors:     12            Quote:     1.2s                 │
│  Timeouts:       8             Policy:    3.4s                 │
│  Circuit trips:  3             Claim:     2.1s                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Prochaine Étape

Passez à la section **16.3 Implémentation Guidée** pour mettre en œuvre cette architecture dans le sandbox.
