# 16.4 Tests et Validation

## Résumé

Une architecture n'est bonne que si elle fonctionne sous contrainte. Cette section vous guide dans la validation de votre implémentation avec des scénarios de test réalistes incluant des pannes simulées.

## Points clés

- Tester le chemin nominal (happy path)
- Tester les cas d'erreur (failure scenarios)
- Valider la résilience (chaos testing)
- Vérifier l'observabilité (monitoring)

---

## Scénario 1 : Parcours de Souscription Complet

### Objectif

Valider le flux complet de souscription du devis à la police avec toutes les intégrations.

### Étapes du Test

```
TEST: Souscription nominale
══════════════════════════

PRÉPARATION
───────────
□ Tous les services démarrés (status: 🟢)
□ Event Bus connecté
□ CDC actif
□ DWH accessible

EXÉCUTION
─────────
1. Demander un devis
   POST /gateway/quotes
   {
     "customer_id": "C001",
     "product": "AUTO",
     "risk_data": {
       "vehicle_type": "sedan",
       "driver_age": 35,
       "bonus_malus": 0.8
     }
   }

   ✓ Expected: 201 Created
   ✓ Response: { "quote_id": "Q-XXX", "premium": 850 }
   ✓ Latency: < 3s

2. Valider le devis (créer la police)
   POST /gateway/quotes/Q-XXX/accept

   ✓ Expected: 201 Created
   ✓ Response: { "policy_number": "POL-XXX" }
   ✓ Latency: < 10s

3. Vérifier les événements propagés
   GET /api/events/recent?policy_id=POL-XXX

   ✓ PolicyCreated published
   ✓ Billing: InvoiceGenerated
   ✓ Notifications: EmailSent
   ✓ Documents: ContractGenerated
   ✓ Audit: EventLogged

4. Vérifier le reporting
   (Attendre 1 minute pour le cycle CDC)
   GET /api/dwh/policies/POL-XXX

   ✓ Record présent dans DWH
   ✓ Customer enrichment OK
   ✓ Calculated fields présents

CRITÈRES DE SUCCÈS
──────────────────
□ Devis en < 3s
□ Police créée en < 10s
□ 4 consommateurs notifiés en < 30s
□ DWH synchronisé en < 2 minutes
□ Trace corrélé de bout en bout
```

---

## Scénario 2 : Résilience aux Pannes

### Objectif

Valider que le système se comporte gracieusement en cas de panne d'un service.

### Test 2.1 : Panne du Rating API

```
TEST: Circuit Breaker sur Rating API
════════════════════════════════════

PRÉPARATION
───────────
□ Injecter une panne sur Rating API
   Sandbox → Inject Failure → External Rating
   Type: Connection Refused
   Duration: 60s

EXÉCUTION
─────────
1. Demander un devis
   POST /gateway/quotes
   { "customer_id": "C001", "product": "AUTO", ... }

   ✓ Expected: 200 OK (pas 500!)
   ✓ Response: { "quote_id": "Q-XXX", "premium": 900 }
   ✓ Note: Premium from fallback (default rates)

2. Vérifier l'état du circuit breaker
   GET /api/gateway/circuits

   ✓ external_rating: OPEN
   ✓ last_failure: "Connection refused"
   ✓ failures_count: 5

3. Observer le fallback
   Logs: "Rating API unavailable, using cached rates"

4. Attendre la récupération (après 60s)
   GET /api/gateway/circuits

   ✓ external_rating: HALF_OPEN
   ✓ Tentative de reconnexion

5. Refaire un devis
   POST /gateway/quotes
   { ... }

   ✓ external_rating: CLOSED
   ✓ Real-time rates restored

CRITÈRES DE SUCCÈS
──────────────────
□ Service client non bloqué
□ Fallback automatique
□ Circuit breaker fonctionnel
□ Récupération automatique
```

### Test 2.2 : Panne de Policy Admin (Compensation Saga)

```
TEST: Compensation de Saga
══════════════════════════

PRÉPARATION
───────────
□ Créer un devis valide Q001
□ Injecter une panne sur Policy Admin
   Type: Timeout après 3s
   Trigger: Sur appel create_policy

EXÉCUTION
─────────
1. Accepter le devis
   POST /gateway/quotes/Q001/accept

   (Le système va tenter de créer la police)

2. Observer la saga
   GET /api/saga/Q001/status

   ✓ Step 1 (ReserveQuote): COMPLETED
   ✓ Step 2 (VerifyCustomer): COMPLETED
   ✓ Step 3 (CreatePolicy): FAILED (timeout)
   ✓ Saga status: COMPENSATING

3. Observer les compensations
   GET /api/saga/Q001/events

   ✓ "Compensating VerifyCustomer" (no-op)
   ✓ "Compensating ReserveQuote" (releasing Q001)

4. Vérifier l'état final
   GET /gateway/quotes/Q001

   ✓ status: AVAILABLE (pas RESERVED)
   ✓ Message: "Subscription failed, please retry"

5. Après récupération, retenter
   POST /gateway/quotes/Q001/accept

   ✓ Saga complète cette fois
   ✓ Policy created: POL-XXX

CRITÈRES DE SUCCÈS
──────────────────
□ Saga détecte l'échec
□ Compensations exécutées dans l'ordre inverse
□ État cohérent après compensation
□ Retry possible après récupération
```

---

## Scénario 3 : Test de Charge

### Objectif

Valider que le système maintient ses performances sous charge.

```
TEST: Charge nominale
═════════════════════

CONFIGURATION
─────────────
□ Concurrent users: 50
□ Duration: 5 minutes
□ Request mix:
   - 60% GET /quotes (read)
   - 30% POST /quotes (create)
   - 10% POST /quotes/accept (subscribe)

MÉTRIQUES À SURVEILLER
──────────────────────
□ Throughput: > 100 req/s
□ Latency P50: < 500ms
□ Latency P99: < 3s
□ Error rate: < 1%
□ Circuit breakers: all CLOSED

EXÉCUTION (Sandbox → Load Test)
───────────────────────────────
1. Lancer le test de charge
   Duration: 5 minutes
   Ramp-up: 30 seconds

2. Observer les métriques en temps réel
   Dashboard → Performance

3. Analyser les résultats

RÉSULTATS ATTENDUS
──────────────────
Throughput:    125 req/s ✓
Latency P50:   320ms ✓
Latency P99:   2.1s ✓
Error rate:    0.3% ✓
Memory:        Stable (~500MB)
CPU:           < 70%

CRITÈRES DE SUCCÈS
──────────────────
□ Aucun timeout
□ Aucun circuit ouvert
□ Latences dans les SLA
□ Pas de memory leak
```

---

## Scénario 4 : Validation de l'Observabilité

### Objectif

Vérifier que le système est correctement instrumenté pour le debugging.

```
TEST: Tracing distribué
═══════════════════════

1. Effectuer une souscription complète

2. Récupérer le trace_id de la réponse
   Header: X-Trace-ID: abc123

3. Visualiser le trace complet
   Sandbox → Observability → Traces → abc123

   ✓ Span: Gateway (10ms)
     └── Span: BFF (5ms)
         ├── Span: QuoteEngine (800ms)
         │   └── Span: RatingAPI (500ms)
         └── Span: CustomerHub (200ms)
     └── Span: PolicyAdmin (3s)
         └── Span: Database (50ms)
     └── Span: Event Publish (20ms)

4. Vérifier la corrélation des logs
   Sandbox → Logs → Filter: trace_id=abc123

   ✓ Tous les logs avec le même trace_id
   ✓ Chronologie cohérente
   ✓ Erreurs facilement identifiables

5. Vérifier les métriques
   Sandbox → Metrics → Dashboard

   ✓ Request count: +1
   ✓ Latency histogram: updated
   ✓ Active policies: +1

CRITÈRES DE SUCCÈS
──────────────────
□ Trace complet visible
□ Tous les services présents
□ Logs corrélés
□ Métriques mises à jour
```

---

## Scénario 5 : Validation des Données

### Objectif

Vérifier l'intégrité et la qualité des données synchronisées.

```
TEST: Qualité des données DWH
═════════════════════════════

1. Créer 10 polices de test
   Script: create_test_policies.py

2. Attendre le cycle CDC (2 minutes)

3. Exécuter les contrôles qualité
   Sandbox → Data → Quality Checks

   CONTRÔLE              RÉSULTAT    ATTENDU
   ───────────────────────────────────────
   Completeness          100%        > 99%    ✓
   Referential Integrity PASS        PASS     ✓
   Duplicates            0           0        ✓
   Freshness             < 2min      < 5min   ✓
   Schema Validity       PASS        PASS     ✓

4. Vérifier le lineage
   Sandbox → Data → Lineage → fact_policies

   ✓ Source: PolicyAdmin.policies
   ✓ Transform: customer_enrichment
   ✓ Target: DWH.fact_policies
   ✓ All transformations documented

CRITÈRES DE SUCCÈS
──────────────────
□ Aucune donnée manquante
□ Intégrité référentielle OK
□ Pas de doublons
□ Fraîcheur < 5 minutes
□ Lineage complet
```

---

## Checklist de Validation Finale

```
FONCTIONNALITÉS CORE
════════════════════
□ Devis calculé en temps réel (< 3s)
□ Police créée avec saga (< 10s)
□ Événements propagés (< 30s)
□ Données dans DWH (< 2min)
□ Vue 360° client fonctionnelle

RÉSILIENCE
══════════
□ Circuit breaker sur tous les services externes
□ Fallback configurés et testés
□ Saga compensation fonctionne
□ Retry avec backoff
□ DLQ pour messages échoués

OBSERVABILITÉ
═════════════
□ Logs structurés avec trace_id
□ Métriques exposées
□ Traces distribuées complètes
□ Alertes configurées

DONNÉES
═══════
□ CDC capture tous les changements
□ ETL transforme correctement
□ Qualité validée
□ Lineage documenté

SÉCURITÉ
════════
□ Auth JWT fonctionnel
□ Rate limiting actif
□ Audit trail complet
```

---

## Prochaine Étape

Passez à la section **16.5 Évaluation Finale** pour synthétiser vos apprentissages et identifier les améliorations possibles.
