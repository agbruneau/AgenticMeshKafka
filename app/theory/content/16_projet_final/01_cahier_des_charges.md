# 16.1 Cahier des Charges - Projet Final

## Résumé

Le projet final vous met dans la peau d'un architecte d'entreprise devant concevoir et implémenter une solution d'intégration complète pour un assureur. Vous devrez mobiliser les trois piliers d'intégration pour créer un écosystème cohérent.

## Points clés

- Scénario réaliste d'intégration en assurance
- Combinaison des trois piliers (Applications, Événements, Données)
- Décisions d'architecture documentées
- Implémentation guidée dans le sandbox

---

## Contexte du Projet

### L'Entreprise

**AssurPlus** est un assureur dommage (Auto + Habitation) qui modernise son système d'information. L'entreprise dispose de plusieurs systèmes legacy qui doivent être intégrés dans un écosystème moderne.

```
┌─────────────────────────────────────────────────────────────────┐
│                    ASSURPLUS - SITUATION ACTUELLE               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  SYSTÈMES EXISTANTS (Legacy)                                    │
│  ════════════════════════════                                   │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   Quote     │  │   Policy    │  │   Claims    │             │
│  │  Engine     │  │   Admin     │  │    Mgmt     │             │
│  │  (REST)     │  │  (COBOL)    │  │  (Oracle)   │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │  Billing    │  │  Customer   │  │   Rating    │             │
│  │  System     │  │    Hub      │  │    API      │             │
│  │  (SAP)      │  │  (CRM)      │  │  (Externe)  │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│                                                                 │
│  PROBLÈMES ACTUELS                                              │
│  ═════════════════                                              │
│  ❌ Intégrations point-à-point (spaghetti)                     │
│  ❌ Pas de vision client unifiée                               │
│  ❌ Latence élevée sur les parcours                            │
│  ❌ Données incohérentes entre systèmes                        │
│  ❌ Pas d'audit trail complet                                  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Objectifs du Projet

### Objectifs Métier

| # | Objectif | Mesure de succès |
|---|----------|------------------|
| 1 | **Parcours client unifié** | Un seul point d'entrée, vue 360° |
| 2 | **Souscription en ligne** | Devis → Police en < 5 minutes |
| 3 | **Notification temps réel** | Client informé en < 1 minute |
| 4 | **Reporting actuariat** | Données consolidées quotidiennes |
| 5 | **Traçabilité complète** | Audit trail de toute modification |

### Objectifs Techniques

| # | Objectif | Mesure de succès |
|---|----------|------------------|
| 1 | **API unifiée** | Gateway avec documentation OpenAPI |
| 2 | **Découplage** | Services indépendamment déployables |
| 3 | **Résilience** | Circuit breaker, retry, fallback |
| 4 | **Observabilité** | Logs, métriques, traces corrélés |
| 5 | **Scalabilité** | Horizontal scaling possible |

---

## Périmètre Fonctionnel

### Flux à Implémenter

#### 1. Parcours de Souscription (Critique)

```
┌─────────────────────────────────────────────────────────────────┐
│              FLUX 1: SOUSCRIPTION EN LIGNE                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. Client demande un devis (portail web)                       │
│     └── 🔗 API Gateway → Quote Engine                          │
│         └── 🔗 Appel Rating API (tarif externe)                │
│                                                                 │
│  2. Client valide le devis                                      │
│     └── 🔗 API Gateway → Policy Admin                          │
│         └── ⚡ Saga: Reserve → Verify → Create → Bill          │
│                                                                 │
│  3. Police créée                                                │
│     └── ⚡ Event: PolicyCreated                                 │
│         ├── Billing: Génère facture                            │
│         ├── Notifications: Email + SMS                         │
│         ├── Document: Génère contrat PDF                       │
│         └── Audit: Log création                                │
│                                                                 │
│  4. Synchronisation reporting                                   │
│     └── 📊 CDC: Policy → Data Warehouse                        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 2. Déclaration de Sinistre

```
┌─────────────────────────────────────────────────────────────────┐
│              FLUX 2: DÉCLARATION SINISTRE                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. Client déclare un sinistre (app mobile)                     │
│     └── 🔗 BFF Mobile → Claims Management                      │
│                                                                 │
│  2. Sinistre enregistré                                         │
│     └── ⚡ Event: ClaimSubmitted                                │
│         ├── Queue: Traitement expert                           │
│         ├── Notifications: Accusé réception                    │
│         └── Document: Upload photos                            │
│                                                                 │
│  3. Traitement du sinistre                                      │
│     └── ⚡ Event Sourcing: États successifs                    │
│         ClaimSubmitted → ClaimAssessed → ClaimApproved         │
│                                                                 │
│  4. Indemnisation                                               │
│     └── ⚡ Saga: Approve → Pay → Close                         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 3. Vue 360° Client

```
┌─────────────────────────────────────────────────────────────────┐
│              FLUX 3: VUE 360° CLIENT                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. Agent consulte un client (portail interne)                  │
│     └── 🔗 API Composition                                     │
│         ├── Customer Hub: Profil                               │
│         ├── Policy Admin: Polices actives                      │
│         ├── Claims: Sinistres en cours                         │
│         ├── Billing: Factures impayées                         │
│         └── Documents: Pièces jointes                          │
│                                                                 │
│  2. Agrégation avec gestion d'erreurs                          │
│     └── 🔗 Partial composition si service down                 │
│         └── Circuit Breaker + Fallback                         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 4. Reporting Actuariat

```
┌─────────────────────────────────────────────────────────────────┐
│              FLUX 4: REPORTING ACTUARIAT                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. Capture des changements (temps réel)                        │
│     └── 📊 CDC sur Policy + Claims + Billing                   │
│                                                                 │
│  2. Pipeline de transformation                                  │
│     └── 📊 ETL: Nettoyage, enrichissement, agrégation          │
│                                                                 │
│  3. Chargement Data Warehouse                                   │
│     └── 📊 Load avec contrôles qualité                         │
│                                                                 │
│  4. Dashboards BI                                               │
│     └── Sinistralité, primes, rétention                        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Contraintes

### Contraintes Techniques

| Contrainte | Description |
|------------|-------------|
| **Latence devis** | < 3 secondes |
| **Latence création police** | < 10 secondes |
| **Disponibilité** | 99.5% pour les flux critiques |
| **Cohérence** | Éventuelle acceptable (< 30s) |
| **Volume** | 1000 devis/jour, 200 polices/jour |

### Contraintes Organisationnelles

| Contrainte | Description |
|------------|-------------|
| **Équipes** | 3 équipes domain (Quote, Policy, Claims) |
| **Déploiement** | Indépendant par service |
| **Legacy** | Policy Admin COBOL reste en place |
| **Budget** | Pas de nouvelle infrastructure (cloud OK) |

---

## Livrables Attendus

### 1. Architecture Documentée

```
□ Diagramme d'architecture globale
□ 3+ ADR (Architecture Decision Records)
□ Mapping piliers → flux métier
□ Stratégie de résilience
```

### 2. Implémentation Sandbox

```
□ Configuration Gateway + BFF
□ Saga de souscription fonctionnelle
□ Events PolicyCreated propagés
□ Pipeline CDC → Reporting
□ Observabilité configurée
```

### 3. Tests de Validation

```
□ Parcours souscription complet
□ Gestion de panne (circuit breaker)
□ Replay événements (audit)
□ Données consolidées (reporting)
```

---

## Critères d'Évaluation

| Critère | Pondération | Description |
|---------|-------------|-------------|
| **Cohérence architecture** | 30% | Les choix sont justifiés et cohérents |
| **Utilisation des 3 piliers** | 25% | Chaque pilier est utilisé à bon escient |
| **Résilience** | 20% | Le système gère les pannes gracieusement |
| **Documentation** | 15% | ADR complets, diagrammes clairs |
| **Tests** | 10% | Scénarios validés dans le sandbox |

---

## Planning Suggéré

```
ÉTAPE 1 - CONCEPTION (Module 16.2)
════════════════════════════════════
□ Analyser les besoins
□ Choisir les patterns par flux
□ Dessiner l'architecture cible
□ Documenter les décisions (ADR)

ÉTAPE 2 - IMPLÉMENTATION (Module 16.3)
══════════════════════════════════════
□ Configurer la Gateway
□ Implémenter la Saga souscription
□ Configurer le Pub/Sub
□ Mettre en place le CDC

ÉTAPE 3 - VALIDATION (Module 16.4)
══════════════════════════════════
□ Tester le parcours complet
□ Injecter des pannes
□ Vérifier le reporting
□ Valider l'observabilité

ÉTAPE 4 - SYNTHÈSE (Module 16.5)
════════════════════════════════
□ Documenter les leçons apprises
□ Identifier les améliorations
□ Évaluation finale
```

---

## Sandbox : Démarrer

Le scénario **CROSS-04** vous guide pas à pas dans la réalisation de ce projet. Commencez par la section 16.2 pour concevoir votre architecture avant de passer à l'implémentation.
