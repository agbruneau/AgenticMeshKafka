# AGENT.MD - Agent de Phase EDA-Lab

Ce fichier configure un agent spécialisé pour implémenter les phases du projet EDA-Lab selon le PRD.MD.

---

## 1. Terminologie

> **Important**: Ce projet utilise deux systèmes de numérotation:
>
> | Terme               | Document | Description                                                  |
> | ------------------- | -------- | ------------------------------------------------------------ |
> | **Itération**       | PRD.MD   | Patron EDA à implémenter (1=Pub/Sub, 2=Event Sourcing, etc.) |
> | **Phase technique** | PLAN.MD  | Étape de construction (0=Infra, 1=Go, 2=Avro, etc.)          |
>
> L'**Itération 1 (MVP Pub/Sub)** est construite via les **Phases techniques 0-8** du PLAN.MD.

---

## 2. Rôle de l'Agent

L'agent est responsable de l'implémentation complète d'une **itération spécifique** du projet EDA-Lab. Il doit:

1. Lire et comprendre le PRD.MD pour l'itération demandée
2. Suivre le PLAN.MD pour les phases techniques (si Itération 1/MVP)
3. Créer la structure de fichiers nécessaire
4. Implémenter le code Go pour les services
5. Créer les schémas Avro
6. Configurer l'infrastructure Docker
7. Écrire les tests
8. Valider que tout fonctionne

---

## 3. Itérations Disponibles (Patrons EDA)

| Itération   | Patron EDA         | Services concernés                    | Référence            |
| ----------- | ------------------ | ------------------------------------- | -------------------- |
| **1 - MVP** | Pub/Sub            | bancaire, simulator, gateway          | PLAN.MD (Phases 0-8) |
| **2**       | Event Sourcing     | assurance-personne, assurance-dommage | PRD.MD §3            |
| **3**       | CQRS               | client-360                            | PRD.MD §3            |
| **4**       | Saga Choreography  | orchestrator                          | PRD.MD §3            |
| **5**       | Saga Orchestration | orchestrator (v2)                     | PRD.MD §3            |
| **6**       | Event Streaming    | tous                                  | PRD.MD §3            |
| **7**       | Dead Letter Queue  | tous                                  | PRD.MD §3            |
| **8**       | Outbox Pattern     | tous                                  | PRD.MD §3            |

---

## 4. Instructions par Itération

### Itération 1 - MVP (Pub/Sub)

**Objectif**: Infrastructure fonctionnelle avec domaine bancaire de base

**Livrables**:

```
eda-lab/
├── infra/
│   ├── docker-compose.yml          # Kafka KRaft + Schema Registry + PostgreSQL
│   └── prometheus/prometheus.yml   # Configuration métriques
├── services/
│   ├── bancaire/
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── domain/             # Entités métier
│   │   │   ├── events/             # Producteurs/Consommateurs Kafka
│   │   │   └── handlers/           # Handlers d'événements
│   │   ├── Dockerfile
│   │   └── go.mod
│   ├── simulator/                  # Générateur d'événements
│   └── gateway/                    # API REST + WebSocket
├── schemas/
│   └── bancaire/
│       ├── compte-ouvert.avsc
│       ├── depot-effectue.avsc
│       └── virement-emis.avsc
└── web-ui/                         # React minimal (Start/Stop)
```

**Événements à implémenter**:

- `CompteOuvert` → topic: `bancaire.compte.ouvert`
- `DepotEffectue` → topic: `bancaire.depot.effectue`
- `RetraitEffectue` → topic: `bancaire.retrait.effectue`
- `VirementEmis` → topic: `bancaire.virement.emis`

**Critères de validation**:

- [ ] `docker-compose up` démarre sans erreur
- [ ] Le simulator produit des événements
- [ ] Le service bancaire consomme les événements
- [ ] Les métriques sont visibles dans Prometheus
- [ ] Tests unitaires passent (`go test ./...`)

---

### Itération 2 - Event Sourcing + Domaines Assurance

**Objectif**: Ajouter Event Sourcing et les domaines d'assurance

**Livrables additionnels**:

- Services `assurance-personne` et `assurance-dommage`
- Tables d'Event Store dans PostgreSQL
- Reconstruction d'état depuis les événements
- Jaeger pour le traçage distribué
- Loki pour les logs

---

### Itération 3 - CQRS + Client 360

**Objectif**: Séparation lecture/écriture avec vue client unifiée

**Livrables additionnels**:

- Service `client-360` avec projections de lecture
- Modèle de lecture dénormalisé
- Synchronisation via événements

---

## 5. Processus d'Implémentation

L'agent doit suivre ce processus pour chaque phase:

```
┌─────────────────────────────────────────────────────────────┐
│ ÉTAPE 1: ANALYSE                                            │
│ - Lire PRD.MD section pertinente                            │
│ - Identifier les composants à créer                         │
│ - Lister les dépendances                                    │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ ÉTAPE 2: PROTOCOLE TDD (Stricte)                            │
│ 1. Écrire le test (RED)                                     │
│ 2. Vérifier que le test échoue (Compilation ou Assert)      │
│ 3. Écrire l'implémentation minimale (GREEN)                 │
│ 4. Vérifier que le test passe                               │
│ 5. Refactoriser si nécessaire                               │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ ÉTAPE 3: VALIDATION & ARRÊT D'URGENCE                       │
│ - Si une étape échoue : STOP IMMÉDIAT                       │
│ - Ne jamais passer à la phase suivante sur des bases rouges │
│ - Fixer le problème AVANT de continuer                      │
└─────────────────────────────────────────────────────────────┘
```

### Protocoles Spécifiques

#### Protocole TDD Strict

Pour chaque fonctionnalité Go :

1. **Créer le fichier de test** (`_test.go`) avec les cas de tests définis.
2. **Exécuter le test** : Il DOIT échouer (ou ne pas compiler).
3. **Créer le code** (`.go`) pour satisfaire le test uniquement.
4. **Exécuter le test** : Il DOIT passer.

#### Protocole d'Arrêt d'Urgence

Si l'infrastructure (Phase 0) ne passe pas `make test-infra`, **INTERDICTION** de procéder à la Phase 1. L'agent doit s'arrêter et demander une intervention ou tenter de corriger l'infrastructure.

---

## 6. Conventions de Code

### Go

```go
// Structure d'un service
services/<nom>/
├── cmd/
│   └── main.go           // Point d'entrée
├── internal/
│   ├── config/           // Configuration
│   ├── domain/           // Entités et value objects
│   ├── events/           // Kafka producer/consumer
│   ├── handlers/         // Business logic
│   └── repository/       // Persistence
├── Dockerfile
└── go.mod
```

### Avro Schema

```json
{
  "type": "record",
  "name": "CompteOuvert",
  "namespace": "com.edalab.bancaire",
  "fields": [
    { "name": "event_id", "type": "string" },
    { "name": "timestamp", "type": "long", "logicalType": "timestamp-millis" },
    { "name": "compte_id", "type": "string" },
    { "name": "client_id", "type": "string" },
    {
      "name": "type_compte",
      "type": {
        "type": "enum",
        "name": "TypeCompte",
        "symbols": ["COURANT", "EPARGNE"]
      }
    },
    {
      "name": "solde_initial",
      "type": {
        "type": "bytes",
        "logicalType": "decimal",
        "precision": 18,
        "scale": 2
      }
    }
  ]
}
```

### Kafka Topics

- Format: `<domaine>.<entite>.<action>`
- Exemple: `bancaire.compte.ouvert`

---

## 7. Commande d'Invocation

Pour invoquer l'agent et lui demander d'implémenter une itération:

```
Implémente l'Itération [N] du projet EDA-Lab selon le PRD.MD et AGENT.MD
```

**Exemples**:

- `Implémente l'Itération 1 (MVP Pub/Sub) du projet EDA-Lab selon le PRD.MD et AGENT.MD`
- `Implémente l'Itération 2 (Event Sourcing) du projet EDA-Lab selon le PRD.MD et AGENT.MD`

**Options avancées**:

- `Implémente uniquement l'infrastructure de l'Itération 1` - Pour une sous-partie
- `Valide l'Itération 1 avant de passer à l'Itération 2` - Pour vérification
- `Continue l'Itération 1 à partir de la Phase technique 4` - Pour reprendre le MVP

---

## 8. Checklist de Validation par Itération

### Itération 1 - MVP (Pub/Sub)

- [ ] Infrastructure Docker fonctionnelle (`make infra-up`)
- [ ] Kafka avec Schema Registry opérationnel (`make test-infra`)
- [ ] Service bancaire produit/consomme des événements
- [ ] Simulator génère des données fictives
- [ ] Gateway expose l'API REST
- [ ] Prometheus collecte les métriques
- [ ] Web UI affiche Start/Stop
- [ ] Tests unitaires passent (`make test-unit`)
- [ ] Tests d'intégration passent (`make test-integration`)
- [ ] Script validate-mvp.sh passe (`./scripts/validate-mvp.sh`)

### Itérations 2+

- [ ] Tous les critères Itération 1
- [ ] Critères spécifiques à l'itération (voir PRD.MD §3 et §15)

---

## 9. Gestion des Erreurs

Si l'agent rencontre un problème:

1. **Dépendance manquante**: Installer via `go get` ou `npm install`
2. **Conflit de schéma**: Vérifier la compatibilité Avro
3. **Erreur Docker**: Vérifier les logs avec `docker-compose logs`
4. **Test en échec**: Corriger avant de continuer

---

**Document créé le 2026-01-20**
**Compatible avec: PRD.MD v1.0**
