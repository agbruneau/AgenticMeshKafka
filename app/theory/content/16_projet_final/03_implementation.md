# 16.3 Implémentation Guidée

## Résumé

Cette section vous guide pas à pas dans l'implémentation de l'architecture conçue. Chaque étape est réalisable dans le sandbox avec des instructions détaillées.

## Points clés

- Implémentation progressive, flux par flux
- Chaque étape est testable indépendamment
- Les erreurs sont attendues et pédagogiques
- Le sandbox simule l'environnement complet

---

## Étape 1 : Configuration de la Gateway

### Objectif

Configurer l'API Gateway comme point d'entrée unique avec :
- Routing vers les services backend
- Authentification JWT
- Rate limiting
- Circuit breaker

### Instructions Sandbox

```
1. Accéder au panneau Gateway (Sandbox → Gateway)

2. Configurer les routes :
   /quotes/*     → Quote Engine (port 8001)
   /policies/*   → Policy Admin (port 8002)
   /claims/*     → Claims Management (port 8003)
   /customers/*  → Customer Hub (port 8004)

3. Activer l'authentification :
   Type: JWT
   Secret: sandbox-secret-key
   Header: Authorization: Bearer <token>

4. Configurer le rate limiting :
   Par client: 100 req/min
   Par IP: 1000 req/min
   Burst: 20 req

5. Activer le circuit breaker :
   Failure threshold: 5
   Reset timeout: 30s
   Half-open requests: 3
```

### Validation

```bash
# Test routing
curl http://localhost:8000/gateway/quotes/Q001
# Expected: 200 OK avec détails du devis

# Test auth (sans token)
curl http://localhost:8000/gateway/policies
# Expected: 401 Unauthorized

# Test rate limit (boucle rapide)
for i in {1..150}; do curl -s http://localhost:8000/gateway/quotes; done
# Expected: 429 Too Many Requests après 100 requêtes
```

---

## Étape 2 : Implémentation du BFF Mobile

### Objectif

Créer un Backend for Frontend optimisé pour l'application mobile avec :
- Payloads réduits (champs essentiels uniquement)
- Agrégation de données
- Cache intelligent

### Instructions Sandbox

```
1. Accéder au panneau BFF (Sandbox → BFF → Mobile)

2. Définir l'endpoint client simplifié :
   GET /bff/mobile/customer/{id}

3. Mapper les champs :
   Source (Customer Hub)     → Mobile
   ═══════════════════════════════════
   full_name                 → name
   email                     → email
   phone                     → phone
   (policies omises)
   (claims omises)

4. Configurer le cache :
   TTL: 5 minutes
   Invalidation: sur événement CustomerUpdated

5. Activer la compression :
   Format: gzip
   Min size: 1KB
```

### Comparaison BFF Mobile vs Broker

```json
// BFF Mobile - Réponse optimisée (500 bytes)
{
  "name": "Jean Dupont",
  "email": "jean.dupont@email.com",
  "phone": "0612345678",
  "active_policies_count": 2,
  "pending_claims_count": 1
}

// BFF Broker - Réponse complète (5KB)
{
  "customer": {
    "id": "C001",
    "full_name": "Jean Dupont",
    "email": "jean.dupont@email.com",
    "phone": "0612345678",
    "address": { ... },
    "risk_profile": { ... }
  },
  "policies": [
    { "number": "POL-001", "type": "AUTO", "premium": 850, ... },
    { "number": "POL-002", "type": "HOME", "premium": 420, ... }
  ],
  "claims": [
    { "number": "CLM-001", "status": "OPEN", "amount": 2500, ... }
  ],
  "invoices": [ ... ],
  "documents": [ ... ]
}
```

---

## Étape 3 : Saga de Souscription

### Objectif

Implémenter le workflow de souscription avec :
- Orchestration des étapes
- Compensation en cas d'échec
- État persisté

### Instructions Sandbox

```
1. Accéder au panneau Saga (Sandbox → Events → Saga)

2. Définir les étapes :

   ÉTAPE 1: ReserveQuote
   ├── Action: QuoteEngine.reserve(quote_id)
   ├── Compensation: QuoteEngine.release(quote_id)
   └── Timeout: 5s

   ÉTAPE 2: VerifyCustomer
   ├── Action: CustomerHub.verify(customer_id)
   ├── Compensation: None (idempotent)
   └── Timeout: 3s

   ÉTAPE 3: CreatePolicy
   ├── Action: PolicyAdmin.create(policy_data)
   ├── Compensation: PolicyAdmin.cancel(policy_id)
   └── Timeout: 10s

   ÉTAPE 4: GenerateInvoice
   ├── Action: Billing.createInvoice(policy_id, amount)
   ├── Compensation: Billing.voidInvoice(invoice_id)
   └── Timeout: 5s

3. Configurer la persistance :
   Store: SQLite (sandbox)
   Table: saga_state

4. Activer le logging :
   Level: DEBUG
   Format: JSON
```

### Test de Compensation

```
1. Lancer une souscription normale :
   Sandbox → Execute Saga → subscription_saga
   Data: { quote_id: "Q001", customer_id: "C001" }
   Expected: SUCCESS - toutes les étapes complétées

2. Simuler une panne à l'étape 3 :
   Sandbox → Inject Failure → PolicyAdmin
   Failure: "Connection timeout" pour 30s

3. Relancer la souscription :
   Expected: COMPENSATED
   - GenerateInvoice: non exécuté
   - CreatePolicy: échec → compensation
   - VerifyCustomer: compensation (no-op)
   - ReserveQuote: compensation (quote relâché)

4. Vérifier l'état :
   Quote Q001: status = "AVAILABLE" (pas réservé)
```

---

## Étape 4 : Configuration Pub/Sub

### Objectif

Mettre en place la propagation d'événements avec :
- Topics par entité métier
- Consommateurs multiples
- Dead Letter Queue

### Instructions Sandbox

```
1. Accéder au panneau Event Bus (Sandbox → Events → Pub/Sub)

2. Créer les topics :
   topic.policies    → PolicyCreated, PolicyCancelled, PolicyRenewed
   topic.claims      → ClaimSubmitted, ClaimApproved, ClaimRejected
   topic.billing     → InvoiceGenerated, PaymentReceived

3. Enregistrer les consommateurs pour topic.policies :

   Consumer: Billing
   ├── Handler: on_policy_created → generate_invoice
   ├── Retry: 3x avec backoff
   └── DLQ: billing.dlq

   Consumer: Notifications
   ├── Handler: on_policy_created → send_welcome_email
   ├── Retry: 5x avec backoff
   └── DLQ: notifications.dlq

   Consumer: Documents
   ├── Handler: on_policy_created → generate_contract_pdf
   ├── Retry: 3x avec backoff
   └── DLQ: documents.dlq

   Consumer: Audit
   ├── Handler: on_any_event → log_to_audit_trail
   ├── Retry: 10x avec backoff
   └── DLQ: audit.dlq

4. Configurer les garanties :
   Delivery: at-least-once
   Ordering: par partition (policy_id)
   Idempotency: consumer-side (dedup key)
```

### Test de Propagation

```
1. Publier un événement PolicyCreated :
   Sandbox → Publish Event → topic.policies
   Event: {
     "type": "PolicyCreated",
     "policy_id": "POL-2024-001",
     "customer_id": "C001",
     "premium": 850,
     "coverages": ["RC", "VOL"]
   }

2. Observer la propagation :
   Expected: 4 consommateurs notifiés
   - Billing: ✅ Invoice INV-001 created
   - Notifications: ✅ Email sent to jean.dupont@email.com
   - Documents: ✅ PDF generated (contract_POL-001.pdf)
   - Audit: ✅ Event logged (audit_id: 12345)

3. Vérifier le timeline :
   t+0ms:    Event published
   t+50ms:   Billing received
   t+100ms:  Notifications received
   t+150ms:  Documents received
   t+200ms:  Audit received
```

---

## Étape 5 : Pipeline CDC → Reporting

### Objectif

Configurer la capture des changements et l'alimentation du Data Warehouse.

### Instructions Sandbox

```
1. Accéder au panneau CDC (Sandbox → Data → CDC)

2. Configurer les sources :

   Source: PolicyAdmin.policies
   ├── Mode: Log-based (timestamp)
   ├── Columns: id, customer_id, premium, status, updated_at
   └── Interval: 10s

   Source: Claims.claims
   ├── Mode: Log-based (timestamp)
   ├── Columns: id, policy_id, amount, status, updated_at
   └── Interval: 10s

   Source: Billing.invoices
   ├── Mode: Log-based (timestamp)
   ├── Columns: id, policy_id, amount, status, updated_at
   └── Interval: 10s

3. Configurer le pipeline de transformation :

   Stage 1: Extract
   ├── Read from CDC streams
   └── Deduplicate by id + timestamp

   Stage 2: Transform
   ├── Join policy + customer (enrichment)
   ├── Calculate derived fields (premium_per_coverage)
   └── Apply business rules (filter test data)

   Stage 3: Load
   ├── Target: DataWarehouse.fact_policies
   ├── Mode: Upsert (on id)
   └── Quality checks: completeness, referential integrity

4. Configurer le scheduling :
   Micro-batch: every 1 minute
   Full refresh: daily at 02:00
```

### Validation du Pipeline

```
1. Créer une nouvelle police (via API) :
   POST /gateway/policies
   { customer_id: "C001", type: "AUTO", premium: 850 }

2. Attendre le cycle CDC (< 30s) :
   Observer: CDC → Captured change for policy POL-XXX

3. Vérifier la transformation :
   Observer: ETL → Transformed record with customer enrichment

4. Vérifier le chargement :
   Query DWH: SELECT * FROM fact_policies WHERE id = 'POL-XXX'
   Expected: Record with all transformed fields

5. Valider la qualité :
   Check: completeness = 100%
   Check: referential_integrity = PASS
```

---

## Étape 6 : Observabilité

### Configuration du Tracing

```
1. Activer le tracing distribué (Sandbox → Observability → Tracing)

2. Instrumenter les services :
   - Gateway: trace_id generation
   - All services: span_id propagation
   - Events: trace_id in message headers

3. Configurer la corrélation :
   Header: X-Trace-ID
   Propagation: W3C Trace Context

4. Tester un parcours complet :
   - Faire une souscription complète
   - Observer le trace dans le visualiseur
   - Vérifier tous les spans (Gateway → BFF → Services → Events)
```

---

## Points de Contrôle

Avant de passer à la validation, vérifiez :

```
□ Gateway routes fonctionnelles
□ BFF Mobile retourne des payloads réduits
□ Saga complète avec compensation testée
□ Pub/Sub propage vers 4+ consommateurs
□ CDC capture les changements en < 30s
□ DWH contient les données transformées
□ Tracing corrèle les requêtes bout en bout
```

---

## Prochaine Étape

Passez à la section **16.4 Tests et Validation** pour valider votre implémentation avec des scénarios réalistes.
