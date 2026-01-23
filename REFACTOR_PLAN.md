# Plan de Refactorisation et Optimisation - FibCalc

## Résumé

Ce plan couvre la refactorisation et l'optimisation du dépôt FibCalc, un calculateur Fibonacci haute performance en Go. Les changements sont organisés en 7 phases par ordre de priorité et de risque.

**Statut: ✅ COMPLÉTÉ** - Toutes les phases (1-7) implémentées.

---

## Phase 1: Couverture de Tests Critique (Risque: FAIBLE)

**Objectif**: Établir un filet de sécurité avant les modifications structurelles.

### 1.1 Tests pour `internal/logging/logger.go` (CRITIQUE - 20 fonctions non testées)
- Créer `internal/logging/logger_test.go`
- Tester: `NewDefaultLogger()`, `NewLogger()`, adaptateurs Zerolog/StdLog
- Tester: helpers Field (`String()`, `Int()`, `Uint64()`, `Float64()`, `Err()`)

### 1.2 Tests pour `internal/server/security.go` et `metrics.go`
- Créer `internal/server/security_test.go` - headers CORS, SecurityMiddleware
- Créer `internal/server/metrics_test.go` - Prometheus, IncrementActiveRequests

### 1.3 Tests pour `cmd/generate-golden/main.go` (CRITIQUE)
- Créer `cmd/generate-golden/main_test.go`
- Valider `fibBig()` pour n=0,1,2,92,93,94

### 1.4 Tests pour `internal/ui/colors.go` et `internal/calibration/runner.go`
- Créer fichiers de tests correspondants

---

## Phase 2: Élimination du Code Dupliqué (Risque: MOYEN)

### 2.1 Supprimer duplication des fonctions couleur
**Fichiers**:
- `internal/cli/ui.go` (lignes 59-84) - SUPPRIMER les 9 wrappers
- Garder `internal/ui/colors.go` comme source canonique
- Mettre à jour imports dans `cli/repl.go`, `cli/output.go`

### 2.2 Extraire la logique de progression dans `cli/repl.go`
**Fichier**: `internal/cli/repl.go`
- Créer fonction `runWithProgress(ctx, calc, n, opts)`
- Remplacer code dupliqué dans `calculate()` (203-262) et `cmdCompare()` (284-352)

### 2.3 Supprimer code mort dans `fibonacci/fastdoubling.go`
**Fichier**: `internal/fibonacci/fastdoubling.go`
- Supprimer `acquireState()`/`releaseState()` (lignes 264-280)
- Utiliser `AcquireState()`/`ReleaseState()` directement

---

## Phase 3: Réduction de la Complexité Cyclomatique (Risque: MOYEN) ✅ COMPLÉTÉE

### 3.1 Refactoriser `DisplayResult` dans `cli/ui.go` (CC≈6-7) ✅
**Fichier**: `internal/cli/ui.go`
- ✅ Extrait `displayResultHeader()` - affiche la taille binaire
- ✅ Extrait `displayDetailedAnalysis()` - affiche les métriques détaillées
- ✅ Extrait `displayCalculatedValue()` - affiche la valeur calculée
- ✅ Refactorisé `DisplayResult()` pour utiliser les nouvelles fonctions
- ✅ Corrigé `emptyStringTest` dans `formatNumberString()`

### 3.2 Considérer registre de commandes pour `processCommand` (CC≈10-12) - DIFFÉRÉ
**Fichier**: `internal/cli/repl.go` (lignes 144-184)
- Option: Convertir switch en `map[string]commandHandler`
- **Décision**: Structure actuelle conservée - le switch est clair et lisible
- Une refactorisation ajouterait de la complexité pour un gain minimal

---

## Phase 4: Optimisations de Performance (Risque: ÉLEVÉ) ✅ COMPLÉTÉE

### 4.1 Optimiser zeroing manuel avec `clear()` (Go 1.21+) ✅
**Fichiers**:
- `internal/bigfft/pool.go` - ✅ Remplacé toutes les boucles `for i := range slice { slice[i] = 0 }` par `clear(slice)`
- `internal/bigfft/bump.go` - ✅ Remplacé la boucle de zeroing dans `Alloc()` par `clear(slice)`
```go
// Avant: for i := range slice { slice[i] = 0 }
// Après: clear(slice)
```

### 4.2 Optimiser cache FFT avec pooling - DIFFÉRÉ
**Fichier**: `internal/bigfft/fft_cache.go`
- **Décision**: Différé - le pooling des copies de cache changerait l'API
- Les copies dans `Get()` sont retournées au caller qui ne sait pas qu'il doit les libérer
- Les copies dans `Put()` doivent persister dans le cache indéfiniment
- Risque de memory leaks ou use-after-free si implémenté

### 4.3 Limiter goroutines Strassen avec sémaphore ✅
**Fichier**: `internal/fibonacci/common.go`
- ✅ Ajouté `taskSemaphore` avec capacité `runtime.NumCPU()*2`
- ✅ Modifié `executeTasks()` pour acquérir/libérer token du sémaphore
- ✅ Modifié `executeMixedTasks()` pour utiliser le même pattern

### 4.4 Ajouter support context dans FFT - DIFFÉRÉ (Optionnel)
**Fichier**: `internal/bigfft/fft_recursion.go`
- **Décision**: Différé - marqué optionnel dans le plan original
- Risque élevé - changement d'API significatif qui affecterait de nombreux fichiers

---

## Phase 5: Cohérence de Nommage (Risque: FAIBLE) ✅ COMPLÉTÉE

### 5.1 Documenter conventions de nommage ✅
**Fichier**: `internal/cli/output.go`
- ✅ Ajouté documentation package expliquant:
  - `Display*` = écrit vers io.Writer
  - `Format*` = retourne string
  - `Write*` = écrit vers fichier

---

## Phase 6: Améliorations Architecture (Risque: ÉLEVÉ) ✅ COMPLÉTÉE

### 6.1 Éliminer la dépendance inverse orchestration → CLI ✅
**Problème**: Le package `orchestration` importait `cli`, violant les principes Clean Architecture (la couche métier ne doit pas dépendre de la présentation).

**Solution implémentée**:
- ✅ Créé `ProgressReporter` interface dans `internal/orchestration/interfaces.go`
- ✅ Créé `ResultPresenter` interface pour découpler la présentation des résultats
- ✅ Créé `NullProgressReporter` pour le mode silencieux
- ✅ Refactorisé `ExecuteCalculations()` pour accepter un `ProgressReporter`
- ✅ Refactorisé `AnalyzeComparisonResults()` pour accepter un `ResultPresenter`
- ✅ Supprimé les imports `cli` et `ui` du package `orchestration`

### 6.2 Implémentations CLI des interfaces ✅
**Fichier**: `internal/cli/presenter.go`
- ✅ `CLIProgressReporter` - Implémente `orchestration.ProgressReporter`
- ✅ `CLIResultPresenter` - Implémente `orchestration.ResultPresenter`
  - `PresentComparisonTable()` - Affiche le tableau de comparaison coloré
  - `PresentResult()` - Affiche le résultat final
  - `FormatDuration()` - Formate les durées
  - `HandleError()` - Gère les erreurs avec codes de sortie

### 6.3 Injection de dépendances dans app layer ✅
**Fichier**: `internal/app/app.go`
- ✅ Injecte `CLIProgressReporter{}` ou `NullProgressReporter{}` selon le mode
- ✅ Injecte `CLIResultPresenter{}` pour l'analyse des résultats

**Bénéfices**:
- Orchestration ne dépend plus de CLI (Clean Architecture respectée)
- Meilleure testabilité (interfaces mockables)
- Séparation claire des responsabilités

---

## Phase 7: Mise à Jour de la Documentation (Risque: FAIBLE) ✅ COMPLÉTÉE

**Objectif**: Assurer que toute la documentation reflète l'état actuel du code après refactorisation.

### 7.1 Mettre à jour le README.md ✅
- ✅ Ajouté optimisations Phase 4 (task semaphore, clear())
- ✅ Mis à jour section "Key Commands" avec commandes complètes
- ✅ Ajouté packages `internal/cli` et `internal/logging` au tableau des composants

### 7.2 Documenter l'API REST ✅
- ✅ Documentation API déjà complète dans `Docs/api/API.md`
- ✅ Date mise à jour (January 2026)

### 7.3 Documentation du code ✅
- ✅ Package `internal/cli/output.go` documenté avec conventions de nommage (Phase 5)
- ✅ Interfaces et fonctions critiques déjà documentées

### 7.4 Mettre à jour CLAUDE.md ✅
- ✅ Ajouté patterns Phase 4 (Task Semaphore, Optimized Zeroing)
- ✅ Ajouté conventions de nommage Phase 5 (Display*/Format*/Write*)
- ✅ Ajouté section Test Coverage avec fichiers créés en Phase 1

### 7.5 Vérifier CONTRIBUTING.md ✅
- ✅ Guide de contribution déjà complet et bien structuré
- ✅ Aucune modification nécessaire

### 7.6 Créer cmd/fibcalc/main.go ✅ (CRITIQUE - était manquant)
- ✅ Point d'entrée de l'application créé
- ✅ Gestion du flag --version ajoutée

---

## Fichiers Critiques à Modifier

| Phase | Fichier | Changement | Statut |
|-------|---------|------------|--------|
| 1 | `internal/logging/logger_test.go` | CRÉER | ✅ |
| 1 | `internal/server/security_test.go` | CRÉER | ✅ |
| 1 | `internal/server/metrics_test.go` | CRÉER | ✅ |
| 2 | `internal/cli/ui.go` | Supprimer lignes 59-84 | ✅ |
| 2 | `internal/cli/repl.go` | Extraire runWithProgress | ✅ |
| 2 | `internal/fibonacci/fastdoubling.go` | Supprimer lignes 264-280 | ✅ |
| 3 | `internal/cli/ui.go` | Refactoriser DisplayResult | ✅ |
| 4 | `internal/bigfft/pool.go` | Utiliser clear() | ✅ |
| 4 | `internal/bigfft/bump.go` | Utiliser clear() | ✅ |
| 4 | `internal/fibonacci/common.go` | Ajouter taskSemaphore | ✅ |
| 5 | `internal/cli/output.go` | Documenter conventions nommage | ✅ |
| 6 | `internal/orchestration/interfaces.go` | CRÉER - Interfaces découplage | ✅ |
| 6 | `internal/orchestration/orchestrator.go` | Utiliser interfaces | ✅ |
| 6 | `internal/cli/presenter.go` | CRÉER - Implémentations CLI | ✅ |
| 6 | `internal/app/app.go` | Injection de dépendances | ✅ |
| 7 | `README.md` | Mise à jour complète | ✅ |
| 7 | `Docs/api/API.md` | Documentation API REST | ✅ |
| 7 | `CLAUDE.md` | Refléter changements Phase 1-5 | ✅ |
| 7 | `CONTRIBUTING.md` | Vérifier (déjà complet) | ✅ |
| 7 | `cmd/fibcalc/main.go` | CRÉER (manquant) | ✅ |

---

## Vérification

Après chaque phase:
```bash
make test              # Tests passent
make lint              # Pas d'erreurs linting
make coverage          # Couverture maintenue/améliorée
go test -race ./...    # Pas de race conditions
make benchmark         # Performance non dégradée (Phase 4)
```

### Tests End-to-End
```bash
# Calcul CLI basique
go run ./cmd/fibcalc -n 1000 -algo fast

# Mode serveur
go run ./cmd/fibcalc --server --port 8080 &
curl "http://localhost:8080/calculate?n=100"
curl "http://localhost:8080/health"

# REPL interactif
go run ./cmd/fibcalc --interactive
# > calc 100
# > exit
```

---

## Ordre d'Implémentation

```
Phase 1 (Tests) ✅
    → Phase 2 (Déduplication) ✅
        → Phase 3 (Complexité) ✅
            → Phase 4 (Performance) ✅
                → Phase 5 (Nommage) ✅
                    → Phase 6 (Architecture) ✅
                        → Phase 7 (Documentation) ✅
```

Chaque phase est indépendamment testable et déployable.

**🎉 Refactorisation complétée!** Toutes les phases ont été implémentées.
