# Chapitre III.7

## PATRONS D'INTERACTION KAFKA

---

> *« Les patterns ne sont pas des solutions ; ce sont des vocabulaires partagés qui permettent aux équipes de communiquer efficacement sur des problèmes complexes. »*
>
> — Gregor Hohpe, Enterprise Integration Patterns

---

Le chapitre précédent a établi les fondations des contrats de données — les accords qui définissent la structure et la sémantique des messages échangés. Mais un contrat sans contexte d'utilisation reste abstrait. Comment ces messages circulent-ils ? Quels patterns gouvernent les interactions entre producteurs et consommateurs ? Comment garantir la fiabilité dans un système distribué ?

Ce chapitre explore les patrons d'interaction Kafka : les modèles architecturaux éprouvés qui structurent la communication événementielle à l'échelle de l'entreprise. Nous commencerons par des cas problématiques réels qui illustrent pourquoi ces patterns sont nécessaires, puis nous examinerons l'implémentation d'un maillage de données (data mesh), l'utilisation de Kafka Connect pour l'intégration, et les mécanismes qui garantissent la livraison fiable des messages.

---

## III.7.1 Notes de Terrain : Cas Problématiques

Avant d'explorer les solutions, examinons les problèmes. Les cas suivants, tirés de projets réels, illustrent les défis que les patrons d'interaction Kafka sont conçus pour résoudre.

### Cas 1 : Le Système de Commandes Incohérent

**Contexte** : Une plateforme e-commerce avec trois services — Commandes, Inventaire, et Paiements — communiquant via Kafka.

**Architecture initiale** :

```
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│   Service       │      │   Service       │      │   Service       │
│   Commandes     │─────▶│   Inventaire    │─────▶│   Paiements     │
│                 │      │                 │      │                 │
│  [OrderCreated] │      │ [InventoryRes.] │      │ [PaymentProc.]  │
└─────────────────┘      └─────────────────┘      └─────────────────┘
        │                        │                        │
        ▼                        ▼                        ▼
   ┌─────────┐             ┌─────────┐             ┌─────────┐
   │ DB      │             │ DB      │             │ DB      │
   │ Orders  │             │ Inventory│            │ Payments│
   └─────────┘             └─────────┘             └─────────┘
```

**Le problème** : Le service Commandes créait une commande dans sa base de données, puis publiait un événement `OrderCreated`. Mais entre l'écriture en base et la publication, plusieurs défaillances pouvaient survenir :

1. L'écriture réussit, mais le service crashe avant de publier → Commande créée mais jamais traitée
2. L'écriture échoue, mais l'événement est publié → Inventaire réservé pour une commande inexistante
3. L'événement est publié deux fois (retry après timeout) → Double réservation d'inventaire

**Impact business** : En 3 mois de production, 0.3% des commandes présentaient des incohérences. Sur 100 000 commandes/jour, cela représentait 300 cas problématiques quotidiens nécessitant une intervention manuelle.

**Symptômes observés** :
- Clients facturés pour des commandes jamais expédiées
- Stock négatif dans le système d'inventaire
- Réconciliations comptables impossibles

> **Note de terrain**
>
> *Diagnostic* : L'équipe a passé 2 semaines à déboguer ce qu'ils pensaient être un « bug aléatoire ». Le vrai problème était architectural : l'absence de garantie transactionnelle entre l'écriture en base et la publication de l'événement.
>
> *Solution appliquée* : Pattern Outbox Transactionnel (détaillé plus loin dans ce chapitre).
>
> *Résultat* : Incohérences réduites à 0%, temps de réconciliation divisé par 10.

### Cas 2 : La Tempête de Retry

**Contexte** : Un service de notification qui envoie des emails suite aux événements de commande.

**Le problème** : Le service de notification consommait les événements et appelait un service SMTP externe. En cas d'échec SMTP (timeout, service indisponible), le message n'était pas commité, causant un retry.

```
Consumer                    SMTP Service               Kafka
   │                            │                        │
   │◀────────── OrderShipped ───────────────────────────│
   │                            │                        │
   │────── Send Email ─────────▶│                        │
   │                            │ (timeout 30s)          │
   │◀───── Timeout Error ───────│                        │
   │                            │                        │
   │ (no commit, rebalance)     │                        │
   │                            │                        │
   │◀────────── OrderShipped (replay) ──────────────────│
   │                            │                        │
   │────── Send Email ─────────▶│                        │
   │                            │ (timeout 30s)          │
   │◀───── Timeout Error ───────│                        │
   │                            │                        │
   └─── (boucle infinie) ───────┴────────────────────────┘
```

**Impact** : Quand le service SMTP est devenu lent (pas indisponible, juste lent), le consumer a commencé à boucler. Chaque retry augmentait la charge sur le SMTP, aggravant la lenteur. En 30 minutes :
- 50 000 tentatives d'envoi pour 500 emails
- Service SMTP saturé
- Alertes en cascade sur tous les systèmes dépendants

**Symptômes** :
- Lag Kafka explosant (de 0 à 100 000 messages)
- CPU du consumer à 100%
- Timeouts en cascade

> **Anti-patron**
>
> *Erreur fondamentale* : Traiter un appel externe (SMTP) comme une opération synchrone bloquante dans un consumer Kafka.
>
> *Règle violée* : Ne jamais bloquer un consumer Kafka sur une opération externe non bornée en temps.
>
> *Pattern correctif* : Dead Letter Queue + Circuit Breaker + Traitement asynchrone avec backoff exponentiel.

### Cas 3 : Le Consommateur Lent qui Bloque Tout

**Contexte** : Un topic partagé par 5 services consommateurs avec des besoins de traitement très différents.

**Architecture** :

```
                    Topic: order-events
                           │
       ┌───────────────────┼───────────────────┐
       │                   │                   │
       ▼                   ▼                   ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Analytics   │     │ Notification│     │ ML Training │
│ (rapide)    │     │ (moyen)     │     │ (très lent) │
│ ~1ms/msg    │     │ ~50ms/msg   │     │ ~2s/msg     │
└─────────────┘     └─────────────┘     └─────────────┘
```

**Le problème** : Tous les services étaient dans le même consumer group pour « simplifier la gestion ». Quand le service ML (qui faisait du feature engineering complexe sur chaque événement) prenait du retard :

1. Le rebalancing Kafka redistribuait ses partitions aux autres consumers
2. Les autres services recevaient des messages qu'ils ne savaient pas traiter
3. Erreurs en cascade, puis crash du groupe entier

**Impact** : Indisponibilité de tous les services de notification pendant 4 heures le jour du Black Friday.

> **Décision architecturale**
>
> *Problème* : Comment permettre à des consommateurs avec des vitesses de traitement très différentes de consommer le même topic ?
>
> *Solution* : Consumer groups séparés par service. Chaque service a son propre groupe et progresse à son rythme.
>
> ```
>                     Topic: order-events
>                            │
>        ┌───────────────────┼───────────────────┐
>        │                   │                   │
>        ▼                   ▼                   ▼
>   Group: analytics    Group: notif       Group: ml-training
>   (offset: 1000)     (offset: 950)       (offset: 500)
> ```
>
> *Trade-off* : Plus de ressources (chaque groupe lit toutes les partitions), mais isolation complète.

### Cas 4 : Le Schéma Poison

**Contexte** : Un producteur a déployé une nouvelle version de schéma avec un bug — un champ obligatoire était mal formaté.

**Le problème** : Les 50 000 messages produits pendant 2 heures étaient tous invalides. Les consumers ne pouvaient pas les désérialiser.

```
Producer (buggy)           Kafka              Consumer
     │                       │                    │
     │── [Invalid Avro] ────▶│                    │
     │── [Invalid Avro] ────▶│                    │
     │── [Invalid Avro] ────▶│                    │
     │        ...            │                    │
     │                       │                    │
     │                       │◀── Poll ──────────│
     │                       │                    │
     │                       │── [Invalid] ──────▶│
     │                       │               CRASH│
     │                       │                    │
     │                       │◀── Poll (retry) ──│
     │                       │── [Invalid] ──────▶│
     │                       │               CRASH│
```

**Impact** : 
- Tous les consumers en boucle de crash
- Impossible de « sauter » les messages invalides sans intervention manuelle
- 6 heures d'indisponibilité totale du pipeline

> **Note de terrain**
>
> *Cause racine* : Validation de schéma insuffisante côté producteur. Le Schema Registry validait la compatibilité structurelle, mais pas la validité des données.
>
> *Solutions implémentées* :
> 1. Validation applicative avant publication
> 2. Dead Letter Queue pour les messages non désérialisables
> 3. Consumer avec gestion d'erreur gracieuse (log + skip)
> 4. Alertes sur le taux d'erreur de désérialisation

### Cas 5 : La Duplication Invisible

**Contexte** : Un système de facturation qui crée des factures à partir des événements de commande.

**Le problème** : Le consumer utilisait `enable.auto.commit=true` avec un traitement qui pouvait prendre plus de 5 secondes (le délai d'auto-commit). Scénario :

1. Consumer reçoit le message, commence le traitement
2. Après 5 secondes, auto-commit se déclenche (message marqué comme traité)
3. Le traitement continue pendant 10 secondes
4. À la seconde 12, le traitement échoue
5. Le consumer crashe, redémarre
6. Le message est considéré comme traité (déjà commité) → Message perdu

Inversement, si le consumer crashait AVANT l'auto-commit :
1. Consumer reçoit le message, traite en 3 secondes
2. Traitement réussi, facture créée
3. Consumer crashe avant l'auto-commit
4. Au redémarrage, le message est rejoué → Facture en double

**Impact** : 0.5% des factures étaient soit manquantes, soit en double. Pour une entreprise avec 10 000 factures/mois, cela représentait 50 cas de contentieux potentiels.

> **Décision architecturale**
>
> *Règle absolue* : Ne JAMAIS utiliser `enable.auto.commit=true` pour des traitements critiques.
>
> *Pattern* : Commit manuel après traitement réussi, avec idempotence côté consommateur.
>
> ```java
> // Pattern correct
> while (true) {
>     ConsumerRecords<K, V> records = consumer.poll(Duration.ofMillis(100));
>     for (ConsumerRecord<K, V> record : records) {
>         try {
>             // Traitement idempotent
>             processIdempotent(record);
>             // Commit après succès
>             consumer.commitSync(Collections.singletonMap(
>                 new TopicPartition(record.topic(), record.partition()),
>                 new OffsetAndMetadata(record.offset() + 1)
>             ));
>         } catch (Exception e) {
>             // Gestion d'erreur explicite
>             handleError(record, e);
>         }
>     }
> }
> ```

### Synthèse des Cas Problématiques

Ces cas illustrent les défis fondamentaux des architectures événementielles :

| Cas | Problème | Pattern de solution |
|-----|----------|---------------------|
| Commandes incohérentes | Atomicité DB + Event | Outbox Transactionnel |
| Tempête de retry | Appel externe bloquant | Dead Letter Queue, Circuit Breaker |
| Consumer lent | Isolation insuffisante | Consumer groups séparés |
| Schéma poison | Messages non désérialisables | DLQ, validation, skip gracieux |
| Duplication | Auto-commit non fiable | Commit manuel, idempotence |

### Analyse Approfondie : Patterns de Résilience

Les cas présentés convergent vers un ensemble de patterns de résilience qui forment le socle de toute architecture Kafka robuste.

**Circuit Breaker pour les Dépendances Externes**

Quand un consumer dépend d'un service externe (API, base de données, service SMTP), le pattern Circuit Breaker prévient les cascades de défaillances.

```java
public class CircuitBreakerConsumer {
    
    private final CircuitBreaker circuitBreaker;
    private final ExternalService externalService;
    private final DeadLetterQueue dlq;
    
    public CircuitBreakerConsumer() {
        this.circuitBreaker = CircuitBreaker.builder("external-service")
            .failureRateThreshold(50)           // Ouvre après 50% d'échecs
            .waitDurationInOpenState(Duration.ofSeconds(30))
            .slidingWindowSize(10)              // Sur les 10 derniers appels
            .build();
    }
    
    public void process(ConsumerRecord<String, Event> record) {
        try {
            // Le circuit breaker protège l'appel externe
            circuitBreaker.executeSupplier(() -> {
                return externalService.call(record.value());
            });
        } catch (CircuitBreakerOpenException e) {
            // Circuit ouvert : envoyer au DLQ pour retry ultérieur
            log.warn("Circuit breaker open, sending to DLQ");
            dlq.send(record, "Circuit breaker open - external service unavailable");
        } catch (Exception e) {
            // Autre erreur : retry normal ou DLQ selon le type
            handleError(record, e);
        }
    }
}
```

**États du Circuit Breaker** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    États du Circuit Breaker                             │
│                                                                         │
│     CLOSED                   OPEN                    HALF-OPEN          │
│  ┌─────────────┐         ┌─────────────┐         ┌─────────────┐       │
│  │ Appels      │         │ Appels      │         │ Appels      │       │
│  │ autorisés   │────────▶│ rejetés     │────────▶│ limités     │       │
│  │             │ échecs  │ immédiatement│ timeout │ (test)      │       │
│  │             │ > seuil │             │         │             │       │
│  └─────────────┘         └─────────────┘         └──────┬──────┘       │
│        ▲                                                │              │
│        │                                                │              │
│        │                 succès                         │              │
│        └────────────────────────────────────────────────┘              │
│                                                                         │
│  Métriques clés:                                                       │
│  - failure_rate: Taux d'échec courant                                  │
│  - state: CLOSED/OPEN/HALF_OPEN                                        │
│  - calls_not_permitted: Appels rejetés par circuit ouvert             │
└─────────────────────────────────────────────────────────────────────────┘
```

**Backpressure et Rate Limiting**

Quand un consumer ne peut pas suivre le rythme de production, le backpressure permet de contrôler le flux.

```java
public class RateLimitedConsumer {
    
    private final RateLimiter rateLimiter;
    private final Semaphore concurrencyLimiter;
    
    public RateLimitedConsumer(int maxRps, int maxConcurrent) {
        this.rateLimiter = RateLimiter.create(maxRps);
        this.concurrencyLimiter = new Semaphore(maxConcurrent);
    }
    
    public void consumeWithBackpressure(ConsumerRecords<K, V> records) {
        for (ConsumerRecord<K, V> record : records) {
            // Limiter le débit
            rateLimiter.acquire();
            
            // Limiter la concurrence
            concurrencyLimiter.acquire();
            try {
                processAsync(record).whenComplete((result, error) -> {
                    concurrencyLimiter.release();
                    if (error != null) {
                        handleError(record, error);
                    }
                });
            } catch (Exception e) {
                concurrencyLimiter.release();
                throw e;
            }
        }
    }
}
```

**Bulkhead Pattern (Isolation des Ressources)**

Le pattern Bulkhead isole les ressources pour éviter qu'une défaillance dans un domaine n'affecte les autres.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Pattern Bulkhead                                     │
│                                                                         │
│  Consumer avec Bulkheads séparés par type de traitement:               │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Thread Pool Principal                         │   │
│  │                                                                  │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │   │
│  │  │ Bulkhead        │  │ Bulkhead        │  │ Bulkhead        │ │   │
│  │  │ Orders          │  │ Payments        │  │ Notifications   │ │   │
│  │  │ (10 threads)    │  │ (5 threads)     │  │ (3 threads)     │ │   │
│  │  │                 │  │                 │  │                 │ │   │
│  │  │ Si saturé:      │  │ Si saturé:      │  │ Si saturé:      │ │   │
│  │  │ DLQ orders      │  │ DLQ payments    │  │ DLQ notif       │ │   │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  Avantage: Une saturation du bulkhead Payments n'affecte pas           │
│  le traitement des Orders                                              │
└─────────────────────────────────────────────────────────────────────────┘
```

**Timeout Patterns**

Les timeouts sont essentiels pour éviter les blocages indéfinis.

```java
public class TimeoutAwareConsumer {
    
    private static final Duration PROCESSING_TIMEOUT = Duration.ofSeconds(30);
    private static final Duration EXTERNAL_CALL_TIMEOUT = Duration.ofSeconds(5);
    
    private final ExecutorService executor;
    
    public void processWithTimeout(ConsumerRecord<K, V> record) {
        CompletableFuture<Void> future = CompletableFuture.runAsync(() -> {
            // Traitement avec timeout sur chaque appel externe
            try {
                ExternalResult result = callExternalWithTimeout(record.value());
                saveResult(result);
            } catch (TimeoutException e) {
                throw new ProcessingException("External call timeout", e);
            }
        }, executor);
        
        try {
            // Timeout global sur le traitement
            future.get(PROCESSING_TIMEOUT.toMillis(), TimeUnit.MILLISECONDS);
        } catch (TimeoutException e) {
            future.cancel(true);
            sendToDlq(record, "Processing timeout exceeded");
        } catch (ExecutionException e) {
            handleError(record, e.getCause());
        }
    }
    
    private ExternalResult callExternalWithTimeout(Event event) throws TimeoutException {
        return CompletableFuture
            .supplyAsync(() -> externalService.call(event))
            .get(EXTERNAL_CALL_TIMEOUT.toMillis(), TimeUnit.MILLISECONDS);
    }
}

---

## III.7.2 Implémentation d'un Maillage de Données (Data Mesh)

### Du Monolithe de Données au Maillage

Le Data Mesh est un paradigme architectural qui décentralise la propriété et la gouvernance des données. Au lieu d'un lac de données centralisé géré par une équipe Data, chaque domaine métier devient responsable de ses propres « produits de données ».

**Architecture traditionnelle (Data Lake centralisé)** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        Équipe Data Centrale                             │
│                                                                         │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                 │
│  │   ETL       │    │  Data Lake  │    │  Data       │                 │
│  │  Pipelines  │───▶│  (central)  │───▶│  Warehouse  │                 │
│  └─────────────┘    └─────────────┘    └─────────────┘                 │
│         ▲                                     │                        │
│         │                                     ▼                        │
│  ┌──────┴──────────────────────────────────────────┐                  │
│  │          Équipes Métier (sources)               │                  │
│  │  Commandes │ Clients │ Inventaire │ Paiements   │                  │
│  └─────────────────────────────────────────────────┘                  │
└─────────────────────────────────────────────────────────────────────────┘

Problèmes:
- Goulot d'étranglement sur l'équipe Data
- Délais de mise à disposition (semaines/mois)
- Perte de contexte métier
- Qualité des données dégradée
```

**Architecture Data Mesh avec Kafka** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        Plateforme Self-Service                          │
│                         (Kafka + Gouvernance)                           │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Event Backbone (Kafka)                        │   │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐             │   │
│  │  │orders.events │ │customers.    │ │inventory.    │             │   │
│  │  │              │ │events        │ │events        │             │   │
│  │  └──────────────┘ └──────────────┘ └──────────────┘             │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│         ▲                  ▲                  ▲                        │
│         │                  │                  │                        │
│  ┌──────┴──────┐    ┌──────┴──────┐    ┌──────┴──────┐                │
│  │  Domaine    │    │  Domaine    │    │  Domaine    │                │
│  │  Commandes  │    │  Clients    │    │  Inventaire │                │
│  │             │    │             │    │             │                │
│  │ [Produit    │    │ [Produit    │    │ [Produit    │                │
│  │  de données]│    │  de données]│    │  de données]│                │
│  └─────────────┘    └─────────────┘    └─────────────┘                │
│                                                                         │
│  Chaque domaine:                                                       │
│  - Possède ses données                                                 │
│  - Publie des produits de données                                      │
│  - Garantit la qualité et la documentation                             │
└─────────────────────────────────────────────────────────────────────────┘
```

### Les Quatre Principes du Data Mesh

**Principe 1 : Propriété par Domaine (Domain Ownership)**

Chaque domaine métier possède et gère ses données comme un produit. L'équipe Commandes est responsable des événements de commande, de leur qualité, de leur documentation, et de leur évolution.

*Implémentation Kafka* :
- Convention de nommage : `{domaine}.{type}.{version}` (ex: `orders.events.v1`)
- Schémas gérés par l'équipe du domaine
- SLA définis et mesurés par le domaine

```yaml
# Exemple de définition de produit de données
product:
  name: "orders.events"
  domain: "commerce/orders"
  owner:
    team: "team-orders"
    contact: "orders-team@company.com"
  description: "Événements du cycle de vie des commandes"
  
  topics:
    - name: "orders.events.v1"
      schema: "schemas/order-event.avsc"
      partitions: 24
      retention: "30d"
      
  sla:
    availability: "99.9%"
    latency_p99: "500ms"
    freshness: "< 5min"
    
  documentation:
    wiki: "https://wiki.company.com/orders-events"
    schema_docs: "https://schema-registry/orders"
```

**Principe 2 : Données comme Produit (Data as Product)**

Les données ne sont pas un sous-produit des applications ; elles sont des produits à part entière avec des utilisateurs, des SLA, et une roadmap.

*Caractéristiques d'un produit de données* :
- Découvrable : Catalogue centralisé, documentation accessible
- Compréhensible : Schémas documentés, exemples, métadonnées
- Fiable : SLA mesurés, alertes, processus d'incident
- Interopérable : Formats standards, contrats explicites

```
┌─────────────────────────────────────────────────────────────────────────┐
│                   Produit de Données : Orders Events                    │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Découvrabilité                                                    │  │
│  │ - Enregistré dans le catalogue de données                        │  │
│  │ - Tags: commerce, orders, transactional                          │  │
│  │ - Recherchable par domaine, type, propriétaire                   │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Documentation                                                     │  │
│  │ - Schéma Avro avec champs doc                                    │  │
│  │ - Exemples de messages pour chaque type d'événement              │  │
│  │ - Guide d'intégration pour les consommateurs                     │  │
│  │ - Changelog des versions                                         │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Qualité & SLA                                                     │  │
│  │ - Disponibilité: 99.9% (mesuré: 99.95%)                         │  │
│  │ - Latence p99: 500ms (mesuré: 320ms)                            │  │
│  │ - Fraîcheur: < 5min (mesuré: 2min)                              │  │
│  │ - Taux d'erreur: < 0.1% (mesuré: 0.02%)                         │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Interopérabilité                                                  │  │
│  │ - Format: Avro avec Schema Registry                              │  │
│  │ - Compatibilité: BACKWARD                                        │  │
│  │ - Protocole: Kafka (Confluent Cloud)                             │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

**Principe 3 : Plateforme Self-Service**

Une plateforme commune fournit les outils, l'infrastructure, et les abstractions qui permettent aux équipes de domaine de publier et consommer des données sans dépendre d'une équipe centrale.

*Composants de la plateforme* :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      Plateforme Data Mesh                               │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Couche Self-Service                           │   │
│  │  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐    │   │
│  │  │ Catalogue │  │ Portail   │  │ CLI/API   │  │ Templates │    │   │
│  │  │ de données│  │ développeur│ │ provision │  │ de topics │    │   │
│  │  └───────────┘  └───────────┘  └───────────┘  └───────────┘    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Couche Gouvernance                            │   │
│  │  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐    │   │
│  │  │ Schema    │  │ Policies  │  │ Audit &   │  │ Sécurité  │    │   │
│  │  │ Registry  │  │ (naming,  │  │ Lineage   │  │ (ACLs)    │    │   │
│  │  │           │  │  retention)│ │           │  │           │    │   │
│  │  └───────────┘  └───────────┘  └───────────┘  └───────────┘    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Couche Infrastructure                         │   │
│  │  ┌───────────────────────────────────────────────────────────┐  │   │
│  │  │              Kafka (Confluent Cloud / Self-managed)        │  │   │
│  │  └───────────────────────────────────────────────────────────┘  │   │
│  │  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐    │   │
│  │  │ Kafka     │  │ ksqlDB    │  │ Flink     │  │ Monitoring│    │   │
│  │  │ Connect   │  │           │  │           │  │           │    │   │
│  │  └───────────┘  └───────────┘  └───────────┘  └───────────┘    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

**Principe 4 : Gouvernance Fédérée**

La gouvernance n'est pas centralisée mais fédérée : des standards globaux (conventions, sécurité, interopérabilité) combinés avec une autonomie locale (schémas, SLA, évolution).

*Standards globaux (non négociables)* :
- Convention de nommage des topics
- Format de sérialisation (Avro)
- Métadonnées obligatoires dans les événements
- Politique de rétention minimum
- Exigences de sécurité (chiffrement, authentification)

*Autonomie locale (par domaine)* :
- Structure des schémas métier
- SLA spécifiques au produit
- Fréquence de publication
- Stratégie de partitionnement

### Implémentation Technique avec Kafka

**Convention de nommage des topics** :

```
{domaine}.{sous-domaine}.{type}.{version}

Exemples:
- orders.checkout.events.v1
- orders.fulfillment.events.v1
- customers.profile.events.v1
- inventory.warehouse.snapshots.v1
```

**Structure de métadonnées standard** :

```avro
{
  "type": "record",
  "name": "DataMeshEnvelope",
  "namespace": "com.company.datamesh",
  "fields": [
    {
      "name": "header",
      "type": {
        "type": "record",
        "name": "DataMeshHeader",
        "fields": [
          {"name": "event_id", "type": "string"},
          {"name": "event_type", "type": "string"},
          {"name": "event_time", "type": "long", "logicalType": "timestamp-millis"},
          {"name": "domain", "type": "string"},
          {"name": "product", "type": "string"},
          {"name": "version", "type": "string"},
          {"name": "correlation_id", "type": ["null", "string"]},
          {"name": "causation_id", "type": ["null", "string"]},
          {
            "name": "source",
            "type": {
              "type": "record",
              "name": "Source",
              "fields": [
                {"name": "system", "type": "string"},
                {"name": "instance", "type": ["null", "string"]}
              ]
            }
          }
        ]
      }
    },
    {
      "name": "payload",
      "type": "bytes",
      "doc": "Payload spécifique au domaine, sérialisé selon le schéma du produit"
    }
  ]
}
```

**Workflow de création d'un nouveau produit de données** :

```yaml
# data-product.yaml - Définition déclarative
apiVersion: datamesh/v1
kind: DataProduct
metadata:
  name: order-events
  domain: commerce/orders
  owner: team-orders
spec:
  topics:
    - name: orders.events.v1
      partitions: 24
      replication: 3
      retention: 30d
      schema:
        type: avro
        file: schemas/order-event.avsc
        compatibility: BACKWARD
      
  access:
    producers:
      - service: order-service
        environment: [dev, staging, prod]
    consumers:
      - service: analytics-service
        environment: [prod]
      - service: notification-service
        environment: [prod]
        
  monitoring:
    alerts:
      - type: lag
        threshold: 10000
        severity: warning
      - type: error_rate
        threshold: 0.01
        severity: critical
```

```bash
# Provisionnement via CLI
datamesh apply -f data-product.yaml

# Résultat:
# ✓ Topic orders.events.v1 créé
# ✓ Schema enregistré dans Schema Registry
# ✓ ACLs configurés pour les producteurs/consommateurs
# ✓ Dashboards de monitoring créés
# ✓ Produit enregistré dans le catalogue
```

### Gouvernance et Qualité des Données

Dans un Data Mesh, la qualité des données est la responsabilité du domaine producteur. Kafka offre plusieurs mécanismes pour garantir cette qualité.

**Validation à la Source**

```java
public class ValidatingProducer {
    
    private final KafkaProducer<String, OrderEvent> producer;
    private final Validator validator;
    private final MetricRegistry metrics;
    
    public void publishOrder(OrderEvent event) {
        // Validation avant publication
        ValidationResult result = validator.validate(event);
        
        if (!result.isValid()) {
            metrics.counter("events.validation.failed").inc();
            log.error("Event validation failed: {}", result.getErrors());
            throw new ValidationException(result.getErrors());
        }
        
        // Enrichissement des métadonnées
        event.getHeader().setProducedAt(Instant.now());
        event.getHeader().setProducerVersion(getApplicationVersion());
        
        // Publication avec callback de confirmation
        producer.send(
            new ProducerRecord<>("orders.events.v1", event.getOrderId(), event),
            (metadata, exception) -> {
                if (exception != null) {
                    metrics.counter("events.publish.failed").inc();
                    log.error("Failed to publish event", exception);
                } else {
                    metrics.counter("events.publish.success").inc();
                    log.debug("Event published to partition {} offset {}", 
                        metadata.partition(), metadata.offset());
                }
            }
        );
    }
}
```

**Contrats de Qualité (Data Quality SLAs)**

```yaml
# quality-contract.yaml
product: orders.events.v1
quality_rules:
  - name: completeness
    description: "Tous les champs obligatoires sont présents"
    check: "order_id IS NOT NULL AND customer_id IS NOT NULL AND total > 0"
    threshold: 99.9%
    
  - name: freshness
    description: "Événements publiés dans les 5 minutes suivant l'action"
    check: "event_time - action_time < 300000"
    threshold: 95%
    
  - name: accuracy
    description: "Total correspond à la somme des items"
    check: "ABS(total - SUM(items.price * items.quantity)) < 1"
    threshold: 100%
    
  - name: uniqueness
    description: "Pas de doublons sur event_id"
    check: "COUNT(DISTINCT event_id) = COUNT(*)"
    threshold: 100%

monitoring:
  check_interval: 5m
  alert_on_breach: true
  alert_channels:
    - slack: "#orders-team"
    - pagerduty: "orders-oncall"
```

**Observabilité du Data Mesh**

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Dashboard Data Mesh                                  │
│                                                                         │
│  Santé des Produits de Données                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │ Produit              │ SLA Dispo │ SLA Latence │ Qualité │ État │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │ orders.events.v1     │ 99.95%    │ 320ms       │ 99.8%   │ ✓    │   │
│  │ customers.profile.v1 │ 99.90%    │ 450ms       │ 99.5%   │ ✓    │   │
│  │ inventory.stock.v1   │ 98.50%    │ 1200ms      │ 97.2%   │ ⚠    │   │
│  │ payments.txn.v1      │ 99.99%    │ 180ms       │ 100%    │ ✓    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  Flux Inter-domaines                                                   │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                                                                  │   │
│  │  Orders ──────▶ Inventory ──────▶ Shipping                      │   │
│  │    │                │                │                          │   │
│  │    │                ▼                ▼                          │   │
│  │    └──────▶ Payments ──────▶ Notifications                      │   │
│  │                                                                  │   │
│  │  Latence end-to-end: 2.3s (SLA: 5s) ✓                          │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  Alertes Actives: 1                                                    │
│  │ ⚠ inventory.stock.v1: Latence p99 au-dessus du SLA (1200ms)    │   │
└─────────────────────────────────────────────────────────────────────────┘
```

> **Perspective stratégique**
>
> Le Data Mesh n'est pas une technologie mais un changement organisationnel. Kafka est un excellent enabler technique, mais le succès dépend de :
> - L'engagement des équipes de domaine à traiter les données comme un produit
> - L'investissement dans une plateforme self-service mature
> - La culture de collaboration et de standards partagés
> - La capacité à mesurer et améliorer la qualité des données
>
> *Indicateur de maturité* : Quand une nouvelle équipe peut publier un produit de données en moins d'une journée sans intervention de l'équipe plateforme, le Data Mesh fonctionne.
>
> *Anti-pattern à éviter* : Créer un « Data Mesh » qui n'est qu'un Data Lake renommé avec la même équipe centrale qui fait tout le travail. Le Data Mesh requiert une véritable décentralisation.

---

## III.7.3 Utilisation de Kafka Connect

### Le Rôle de Kafka Connect

Kafka Connect est le framework d'intégration de l'écosystème Kafka. Il permet de connecter Kafka à des systèmes externes (bases de données, files, APIs, stockage cloud) sans écrire de code custom.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      Architecture Kafka Connect                         │
│                                                                         │
│  Sources externes              Kafka              Destinations          │
│  ┌─────────────┐         ┌─────────────┐         ┌─────────────┐       │
│  │ PostgreSQL  │─┐       │             │       ┌─│ Elasticsearch│       │
│  └─────────────┘ │       │             │       │ └─────────────┘       │
│  ┌─────────────┐ │       │   Topics    │       │ ┌─────────────┐       │
│  │  MongoDB    │─┼──────▶│             │──────┼─│    S3       │       │
│  └─────────────┘ │       │             │       │ └─────────────┘       │
│  ┌─────────────┐ │       │             │       │ ┌─────────────┐       │
│  │  Salesforce │─┘       │             │       └─│  Snowflake  │       │
│  └─────────────┘         └─────────────┘         └─────────────┘       │
│        │                        │                        │              │
│        │     Source Connectors  │    Sink Connectors     │              │
│        └────────────────────────┼────────────────────────┘              │
│                                 │                                       │
│                    ┌────────────┴────────────┐                         │
│                    │    Kafka Connect        │                         │
│                    │    Workers (cluster)    │                         │
│                    └─────────────────────────┘                         │
└─────────────────────────────────────────────────────────────────────────┘
```

**Avantages de Kafka Connect** :

*Configuration vs. Code* : Les connecteurs sont configurés en JSON/YAML, pas développés. Cela réduit le temps de mise en place et les risques de bugs.

*Scalabilité native* : Les workers Kafka Connect forment un cluster qui distribue automatiquement la charge.

*Tolérance aux pannes* : En cas de défaillance d'un worker, les tâches sont redistribuées aux workers restants.

*Écosystème riche* : Plus de 200 connecteurs disponibles pour les systèmes courants.

### Patterns d'Intégration avec Kafka Connect

**Pattern 1 : Change Data Capture (CDC)**

Le CDC capture les changements dans une base de données et les publie comme événements Kafka. C'est le pattern le plus puissant pour intégrer des systèmes legacy.

```
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│   PostgreSQL    │      │  Debezium       │      │    Kafka        │
│                 │      │  Connector      │      │                 │
│  ┌───────────┐  │      │                 │      │  ┌───────────┐  │
│  │  orders   │──┼─────▶│  (reads WAL)   │─────▶│  │ orders.   │  │
│  │  table    │  │      │                 │      │  │ cdc       │  │
│  └───────────┘  │      │                 │      │  └───────────┘  │
│                 │      │                 │      │                 │
│  Transaction    │      │  Captures:      │      │  Events:        │
│  Log (WAL)      │      │  - INSERT       │      │  - Created      │
│                 │      │  - UPDATE       │      │  - Updated      │
│                 │      │  - DELETE       │      │  - Deleted      │
└─────────────────┘      └─────────────────┘      └─────────────────┘
```

**Configuration Debezium pour PostgreSQL** :

```json
{
  "name": "orders-cdc-connector",
  "config": {
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "database.hostname": "postgres.internal",
    "database.port": "5432",
    "database.user": "debezium",
    "database.password": "${secrets:postgres-password}",
    "database.dbname": "orders_db",
    "database.server.name": "orders",
    "table.include.list": "public.orders,public.order_items",
    "plugin.name": "pgoutput",
    
    "transforms": "route",
    "transforms.route.type": "org.apache.kafka.connect.transforms.RegexRouter",
    "transforms.route.regex": "orders\\.public\\.(.*)",
    "transforms.route.replacement": "orders.cdc.$1",
    
    "key.converter": "io.confluent.connect.avro.AvroConverter",
    "key.converter.schema.registry.url": "http://schema-registry:8081",
    "value.converter": "io.confluent.connect.avro.AvroConverter",
    "value.converter.schema.registry.url": "http://schema-registry:8081"
  }
}
```

**Structure d'un événement CDC** :

```json
{
  "before": {
    "id": 123,
    "status": "pending",
    "total": 5000
  },
  "after": {
    "id": 123,
    "status": "confirmed",
    "total": 5000
  },
  "source": {
    "version": "2.4.0.Final",
    "connector": "postgresql",
    "name": "orders",
    "ts_ms": 1705312200000,
    "snapshot": "false",
    "db": "orders_db",
    "schema": "public",
    "table": "orders",
    "txId": 12345,
    "lsn": 98765432
  },
  "op": "u",
  "ts_ms": 1705312200100
}
```

> **Note de terrain**
>
> *Contexte* : Migration d'un monolithe vers des microservices. Le monolithe utilisait une base PostgreSQL partagée.
>
> *Approche* : Debezium pour capturer les changements de la base legacy et les publier vers Kafka. Les nouveaux microservices consomment les événements CDC.
>
> *Avantages* :
> - Pas de modification du monolithe (non-invasif)
> - Latence faible (millisecondes)
> - Historique complet des changements
>
> *Pièges évités* :
> - Configurer la rétention du WAL suffisante (évite la perte d'événements)
> - Monitorer le lag du connecteur (alerte si > 1 minute)
> - Tester le comportement lors des migrations de schéma BD

**Pattern 2 : Sink vers Data Lake / Data Warehouse**

Kafka Connect peut écrire les événements vers des systèmes analytiques pour le reporting et le machine learning.

```
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│    Kafka        │      │  S3 Sink        │      │    S3 / Iceberg │
│                 │      │  Connector      │      │                 │
│  ┌───────────┐  │      │                 │      │  ┌───────────┐  │
│  │ orders.   │──┼─────▶│  (batches to   │─────▶│  │ /data/    │  │
│  │ events    │  │      │   Parquet)     │      │  │ orders/   │  │
│  └───────────┘  │      │                 │      │  └───────────┘  │
│                 │      │  - Partitioning │      │                 │
│                 │      │  - Compaction   │      │  Format:        │
│                 │      │  - Schema evol. │      │  Parquet/Iceberg│
└─────────────────┘      └─────────────────┘      └─────────────────┘
```

**Configuration S3 Sink avec partitionnement temporel** :

```json
{
  "name": "orders-s3-sink",
  "config": {
    "connector.class": "io.confluent.connect.s3.S3SinkConnector",
    "tasks.max": "4",
    "topics": "orders.events.v1",
    
    "s3.region": "ca-central-1",
    "s3.bucket.name": "company-data-lake",
    "s3.part.size": "52428800",
    
    "storage.class": "io.confluent.connect.s3.storage.S3Storage",
    "format.class": "io.confluent.connect.s3.format.parquet.ParquetFormat",
    "parquet.codec": "snappy",
    
    "partitioner.class": "io.confluent.connect.storage.partitioner.TimeBasedPartitioner",
    "path.format": "'year'=YYYY/'month'=MM/'day'=dd/'hour'=HH",
    "locale": "en-CA",
    "timezone": "America/Toronto",
    "partition.duration.ms": "3600000",
    
    "flush.size": "10000",
    "rotate.interval.ms": "600000",
    
    "key.converter": "io.confluent.connect.avro.AvroConverter",
    "key.converter.schema.registry.url": "http://schema-registry:8081",
    "value.converter": "io.confluent.connect.avro.AvroConverter",
    "value.converter.schema.registry.url": "http://schema-registry:8081"
  }
}
```

**Pattern 3 : Intégration API avec HTTP Sink**

Pour les systèmes sans connecteur natif, le HTTP Sink permet d'appeler des APIs REST.

```json
{
  "name": "webhook-sink",
  "config": {
    "connector.class": "io.confluent.connect.http.HttpSinkConnector",
    "tasks.max": "2",
    "topics": "notifications.events.v1",
    
    "http.api.url": "https://api.external-system.com/events",
    "request.method": "POST",
    "headers": "Content-Type:application/json|Authorization:Bearer ${secrets:api-token}",
    
    "batch.max.size": "100",
    "request.body.format": "json",
    
    "retry.on.status.codes": "500-599",
    "max.retries": "5",
    "retry.backoff.ms": "1000",
    
    "behavior.on.error": "log",
    
    "key.converter": "org.apache.kafka.connect.storage.StringConverter",
    "value.converter": "org.apache.kafka.connect.json.JsonConverter",
    "value.converter.schemas.enable": "false"
  }
}
```

### Gestion des Erreurs dans Kafka Connect

**Dead Letter Queue pour les erreurs** :

```json
{
  "name": "orders-sink-with-dlq",
  "config": {
    "connector.class": "...",
    
    "errors.tolerance": "all",
    "errors.deadletterqueue.topic.name": "orders-sink-dlq",
    "errors.deadletterqueue.topic.replication.factor": "3",
    "errors.deadletterqueue.context.headers.enable": "true",
    
    "errors.log.enable": "true",
    "errors.log.include.messages": "true"
  }
}
```

**Structure des headers DLQ** :

| Header | Description |
|--------|-------------|
| `__connect.errors.topic` | Topic source du message en erreur |
| `__connect.errors.partition` | Partition source |
| `__connect.errors.offset` | Offset du message |
| `__connect.errors.connector.name` | Nom du connecteur |
| `__connect.errors.task.id` | ID de la tâche |
| `__connect.errors.exception.class` | Classe de l'exception |
| `__connect.errors.exception.message` | Message d'erreur |
| `__connect.errors.exception.stacktrace` | Stack trace complète |

### Monitoring de Kafka Connect

**Métriques JMX essentielles** :

```
# Santé des connecteurs
kafka.connect:type=connector-metrics,connector=*
  - connector-status (running/paused/failed)
  - connector-type (source/sink)
  
# Performance des tâches
kafka.connect:type=task-metrics,connector=*,task=*
  - batch-size-avg
  - batch-size-max
  - offset-commit-success-rate
  - offset-commit-failure-rate
  
# Erreurs
kafka.connect:type=task-error-metrics,connector=*,task=*
  - total-errors-logged
  - total-records-failed
  - total-records-skipped
  - deadletterqueue-produce-requests
```

**Dashboard de monitoring recommandé** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Kafka Connect Dashboard                              │
│                                                                         │
│  Connecteurs actifs: 12/12 ✓        Workers: 3/3 ✓                     │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │ Throughput (messages/sec)                                        │   │
│  │ ████████████████████████████████████░░░░░░ 45,000 msg/s         │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │ Error Rate                                                       │   │
│  │ ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 0.02%               │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  Connecteurs avec erreurs:                                             │
│  │ salesforce-source │ 3 erreurs │ Voir DLQ │                         │
│                                                                         │
│  Top connecteurs par volume:                                           │
│  │ orders-cdc        │ 15,000 msg/s │ lag: 0    │                     │
│  │ customers-cdc     │ 8,000 msg/s  │ lag: 120  │                     │
│  │ s3-sink           │ 22,000 msg/s │ lag: 500  │                     │
└─────────────────────────────────────────────────────────────────────────┘
```

### Transformations Single Message Transforms (SMT)

Les SMT permettent de transformer les messages à la volée sans code custom. Elles sont essentielles pour adapter les données sources au format cible.

**Transformations courantes** :

```json
{
  "name": "orders-cdc-with-transforms",
  "config": {
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "database.hostname": "postgres",
    "database.dbname": "orders",
    
    "transforms": "route,unwrap,timestamp,mask",
    
    "transforms.route.type": "org.apache.kafka.connect.transforms.RegexRouter",
    "transforms.route.regex": "orders\\.public\\.(.*)",
    "transforms.route.replacement": "cdc.$1.events",
    
    "transforms.unwrap.type": "io.debezium.transforms.ExtractNewRecordState",
    "transforms.unwrap.drop.tombstones": "false",
    "transforms.unwrap.delete.handling.mode": "rewrite",
    "transforms.unwrap.add.fields": "op,source.ts_ms",
    
    "transforms.timestamp.type": "org.apache.kafka.connect.transforms.InsertField$Value",
    "transforms.timestamp.timestamp.field": "kafka_timestamp",
    
    "transforms.mask.type": "org.apache.kafka.connect.transforms.MaskField$Value",
    "transforms.mask.fields": "credit_card_number,ssn",
    "transforms.mask.replacement": "****"
  }
}
```

**Transformation personnalisée** :

```java
public class EnrichWithEnvironment implements Transformation<SourceRecord> {
    
    private String environment;
    
    @Override
    public void configure(Map<String, ?> configs) {
        this.environment = (String) configs.get("environment");
    }
    
    @Override
    public SourceRecord apply(SourceRecord record) {
        Struct value = (Struct) record.value();
        
        // Créer une nouvelle structure enrichie
        Schema newSchema = SchemaBuilder.struct()
            .field("environment", Schema.STRING_SCHEMA)
            .field("data", record.valueSchema())
            .build();
        
        Struct newValue = new Struct(newSchema)
            .put("environment", environment)
            .put("data", value);
        
        return record.newRecord(
            record.topic(),
            record.kafkaPartition(),
            record.keySchema(),
            record.key(),
            newSchema,
            newValue,
            record.timestamp()
        );
    }
    
    @Override
    public ConfigDef config() {
        return new ConfigDef()
            .define("environment", ConfigDef.Type.STRING, 
                    ConfigDef.Importance.HIGH, "Environment name");
    }
    
    @Override
    public void close() {}
}
```

### Scalabilité et Haute Disponibilité de Kafka Connect

**Architecture distribuée** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                Cluster Kafka Connect Distribué                          │
│                                                                         │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐         │
│  │ Worker 1        │  │ Worker 2        │  │ Worker 3        │         │
│  │                 │  │                 │  │                 │         │
│  │ Connector A     │  │ Connector A     │  │ Connector B     │         │
│  │ Task 0          │  │ Task 1          │  │ Task 0          │         │
│  │                 │  │                 │  │                 │         │
│  │ Connector B     │  │ Connector C     │  │ Connector C     │         │
│  │ Task 1          │  │ Task 0          │  │ Task 1          │         │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘         │
│           │                    │                    │                   │
│           └────────────────────┼────────────────────┘                   │
│                                │                                        │
│                                ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Topics Internes                               │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │   │
│  │  │ connect-    │  │ connect-    │  │ connect-    │              │   │
│  │  │ configs     │  │ offsets     │  │ status      │              │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  En cas de défaillance du Worker 2:                                    │
│  - Les tâches sont redistribuées aux Workers 1 et 3                    │
│  - Les offsets sont préservés (stockés dans Kafka)                     │
│  - Reprise automatique sans perte de données                           │
└─────────────────────────────────────────────────────────────────────────┘
```

**Configuration pour la haute disponibilité** :

```properties
# worker.properties
group.id=connect-cluster-prod

# Stockage distribué
config.storage.topic=connect-configs
config.storage.replication.factor=3

offset.storage.topic=connect-offsets
offset.storage.replication.factor=3
offset.storage.partitions=25

status.storage.topic=connect-status
status.storage.replication.factor=3
status.storage.partitions=5

# Heartbeat et rebalancing
heartbeat.interval.ms=3000
session.timeout.ms=30000
rebalance.timeout.ms=60000

# Nombre de tâches par worker
tasks.max.per.worker=20
```

> **Note de terrain**
>
> *Contexte* : Migration de 50 tables PostgreSQL vers Kafka via Debezium.
>
> *Défis rencontrés* :
> 1. Initial snapshot de tables volumineuses (100M+ lignes) causait des timeouts
> 2. Pics de charge lors des mises à jour batch saturaient les workers
> 3. Changements de schéma BD causaient des erreurs de sérialisation
>
> *Solutions appliquées* :
> 1. Snapshot incrémental avec `snapshot.mode=when_needed` et filtrage par date
> 2. Scaling horizontal à 5 workers avec tasks.max=3 par connecteur
> 3. SMT pour filtrer les colonnes non nécessaires + alertes sur les migrations de schéma
>
> *Résultat* : Latence CDC < 500ms pour 99% des événements, 0 perte de données en 18 mois.

---

## III.7.4 Assurer la Garantie de Livraison

La garantie de livraison est au cœur de toute architecture événementielle fiable. Kafka offre plusieurs niveaux de garantie, chacun avec des compromis entre performance, complexité, et fiabilité. Comprendre ces compromis est essentiel pour choisir la bonne approche pour chaque cas d'usage.

### Les Trois Sémantiques de Livraison

Kafka supporte trois niveaux de garantie de livraison, chacun avec des trade-offs différents. Le choix dépend des exigences métier et de la tolérance aux pertes ou duplications.

**At-Most-Once (Au plus une fois)**

Le message est livré zéro ou une fois. En cas d'erreur, le message peut être perdu. C'est la sémantique la plus simple mais aussi la moins fiable.

```
Producer                    Broker                    Consumer
   │                          │                          │
   │─── Send message ────────▶│                          │
   │                          │                          │
   │    (no ack wait)         │                          │
   │                          │◀─── Poll ────────────────│
   │                          │                          │
   │                          │─── Message ─────────────▶│
   │                          │                          │
   │                          │         (process)        │
   │                          │                          │
   │                          │◀─── Commit ──────────────│
   │                          │         (before success) │
   │                          │                          │
   │                          │         [CRASH]          │
   │                          │                          │
   │                          │    Message perdu         │
```

*Mécanisme* : Le producteur envoie le message sans attendre d'acquittement (`acks=0`). Le consommateur commit l'offset avant ou pendant le traitement. Si le traitement échoue après le commit, le message est perdu.

*Configuration* :
```java
// Producer - Pas d'attente d'acquittement
Properties producerProps = new Properties();
producerProps.put("acks", "0");
producerProps.put("retries", "0");

// Consumer - Auto-commit avant traitement
Properties consumerProps = new Properties();
consumerProps.put("enable.auto.commit", "true");
consumerProps.put("auto.commit.interval.ms", "1000");
```

*Performances* : Latence minimale (~1-2ms), débit maximal.

*Cas d'usage appropriés* :
- Logs applicatifs non critiques
- Métriques de monitoring où la perte occasionnelle est acceptable
- Données de clickstream pour analytics approximatif
- Systèmes de cache warming où la fraîcheur prime sur la complétude

*Cas d'usage inappropriés* :
- Transactions financières
- Données réglementaires
- Événements déclenchant des actions irréversibles

**At-Least-Once (Au moins une fois)**

Le message est livré une ou plusieurs fois. En cas d'erreur, le message peut être dupliqué mais jamais perdu. C'est la sémantique la plus couramment utilisée.

```
Producer                    Broker                    Consumer
   │                          │                          │
   │─── Send message ────────▶│                          │
   │◀── Ack ──────────────────│                          │
   │                          │                          │
   │                          │◀─── Poll ────────────────│
   │                          │─── Message ─────────────▶│
   │                          │                          │
   │                          │         (process OK)     │
   │                          │                          │
   │                          │         [CRASH before    │
   │                          │          commit]         │
   │                          │                          │
   │                          │◀─── Poll (after restart)─│
   │                          │─── Message (replay) ────▶│
   │                          │                          │
   │                          │         Message dupliqué │
```

*Mécanisme* : Le producteur attend l'acquittement (`acks=all`) et retry en cas d'échec. Le consommateur commit l'offset après traitement réussi. Si le consumer crashe après traitement mais avant commit, le message sera rejoué.

*Configuration* :
```java
// Producer - Acquittement complet avec retry
Properties producerProps = new Properties();
producerProps.put("acks", "all");
producerProps.put("retries", Integer.MAX_VALUE);
producerProps.put("retry.backoff.ms", "100");
producerProps.put("delivery.timeout.ms", "120000");
producerProps.put("enable.idempotence", "false");  // Pas d'idempotence producteur

// Consumer - Commit manuel après traitement
Properties consumerProps = new Properties();
consumerProps.put("enable.auto.commit", "false");
consumerProps.put("auto.offset.reset", "earliest");

// Boucle de consommation
while (true) {
    ConsumerRecords<K, V> records = consumer.poll(Duration.ofMillis(100));
    for (ConsumerRecord<K, V> record : records) {
        try {
            process(record);  // Traitement métier
            consumer.commitSync(Collections.singletonMap(
                new TopicPartition(record.topic(), record.partition()),
                new OffsetAndMetadata(record.offset() + 1)
            ));
        } catch (Exception e) {
            // Gestion d'erreur - le message sera rejoué
            handleError(record, e);
        }
    }
}
```

*Performances* : Latence modérée (~5-20ms), débit élevé.

*Cas d'usage appropriés* :
- La plupart des cas d'usage métier
- Systèmes où l'idempotence peut être implémentée côté consommateur
- Pipelines de données avec déduplication en aval
- Notifications (envoyer deux fois vaut mieux que pas du tout)

*Exigence critique* : Le consommateur DOIT être idempotent pour gérer les duplications.

**Exactly-Once (Exactement une fois)**

Le message est livré exactement une fois, même en cas d'erreur. C'est la garantie la plus forte mais aussi la plus coûteuse en termes de performance et de complexité.

```
Producer (transactionnel)   Broker                    Consumer (read_committed)
   │                          │                          │
   │─── beginTransaction() ──▶│                          │
   │─── Send message ────────▶│ (non visible)            │
   │─── commitTransaction() ─▶│                          │
   │                          │ (visible)                │
   │                          │                          │
   │                          │◀─── Poll ────────────────│
   │                          │─── Message ─────────────▶│
   │                          │                          │
   │                          │         (process)        │
   │                          │                          │
   │                          │◀─── sendOffsetsToTx ─────│
   │                          │◀─── commitTransaction ───│
```

*Mécanisme* : Kafka utilise des transactions pour garantir l'atomicité entre la production de messages et le commit des offsets. Les consommateurs en mode `read_committed` ne voient que les messages des transactions commitées.

*Configuration producteur transactionnel* :
```java
Properties producerProps = new Properties();
producerProps.put("acks", "all");
producerProps.put("enable.idempotence", "true");
producerProps.put("transactional.id", "orders-producer-" + instanceId);
producerProps.put("transaction.timeout.ms", "60000");

KafkaProducer<K, V> producer = new KafkaProducer<>(producerProps);
producer.initTransactions();

try {
    producer.beginTransaction();
    
    // Envoyer plusieurs messages dans la même transaction
    producer.send(new ProducerRecord<>("topic-a", key1, value1));
    producer.send(new ProducerRecord<>("topic-b", key2, value2));
    
    // Commit des offsets du consumer dans la transaction
    producer.sendOffsetsToTransaction(offsets, consumerGroupId);
    
    producer.commitTransaction();
} catch (ProducerFencedException | OutOfOrderSequenceException e) {
    // Erreur fatale - le producteur doit être recréé
    producer.close();
    throw e;
} catch (KafkaException e) {
    // Erreur récupérable - abort et retry
    producer.abortTransaction();
    throw e;
}
```

*Configuration consommateur read_committed* :
```java
Properties consumerProps = new Properties();
consumerProps.put("isolation.level", "read_committed");
consumerProps.put("enable.auto.commit", "false");
```

*Performances* : Latence plus élevée (~20-100ms), débit réduit de 20-30%.

*Cas d'usage appropriés* :
- Transactions financières où toute perte ou duplication est inacceptable
- Transfert de fonds entre comptes
- Systèmes de comptabilité et d'audit
- Pipelines Kafka Streams avec state stores

*Limitations* :
- Exactly-once est limité à l'écosystème Kafka
- Les appels à des systèmes externes (BD, API) restent at-least-once
- Coût en performance non négligeable

### Comparaison des Sémantiques

| Critère | At-Most-Once | At-Least-Once | Exactly-Once |
|---------|--------------|---------------|--------------|
| Perte possible | Oui | Non | Non |
| Duplication possible | Non | Oui | Non |
| Latence | Minimale | Modérée | Élevée |
| Débit | Maximum | Élevé | Réduit |
| Complexité | Simple | Moyenne | Élevée |
| Idempotence requise | Non | Oui | Non |
| Cas d'usage | Logs, métriques | Plupart des cas | Financier, critique |

### Pattern Outbox Transactionnel

Le pattern Outbox résout le problème de l'atomicité entre une écriture en base de données et la publication d'un événement Kafka. C'est l'un des patterns les plus importants pour les architectures événementielles.

**Le problème** :

```java
// Code problématique - PAS atomique
@Transactional
public void createOrder(Order order) {
    orderRepository.save(order);      // 1. Écriture BD
    kafkaProducer.send(orderEvent);   // 2. Publication Kafka
    // Que se passe-t-il si le service crashe entre 1 et 2 ?
}
```

**La solution Outbox** :

```
┌─────────────────────────────────────────────────────────────────────────┐
│                       Pattern Outbox Transactionnel                     │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Service Application                           │   │
│  │                                                                  │   │
│  │  @Transactional                                                 │   │
│  │  public void createOrder(Order order) {                         │   │
│  │      orderRepository.save(order);                               │   │
│  │      outboxRepository.save(new OutboxEvent(                     │   │
│  │          "OrderCreated",                                        │   │
│  │          order.getId(),                                         │   │
│  │          serialize(order)                                       │   │
│  │      ));                                                        │   │
│  │      // Les deux écritures sont dans la même transaction DB     │   │
│  │  }                                                              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                              │                                          │
│                              ▼                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Base de Données                               │   │
│  │  ┌───────────────┐    ┌───────────────────────────────────┐     │   │
│  │  │ orders        │    │ outbox                             │     │   │
│  │  │ (table métier)│    │ (table outbox)                    │     │   │
│  │  │               │    │                                    │     │   │
│  │  │ id: 123       │    │ id: 456                           │     │   │
│  │  │ status: new   │    │ event_type: OrderCreated          │     │   │
│  │  │ total: 5000   │    │ aggregate_id: 123                 │     │   │
│  │  │               │    │ payload: {...}                    │     │   │
│  │  │               │    │ created_at: 2024-01-15T10:30:00Z  │     │   │
│  │  │               │    │ published: false                  │     │   │
│  │  └───────────────┘    └───────────────────────────────────┘     │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                              │                                          │
│                              ▼                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │               Debezium CDC Connector (ou polling)                │   │
│  │                                                                  │   │
│  │  - Lit les nouvelles entrées dans la table outbox               │   │
│  │  - Publie vers Kafka                                            │   │
│  │  - Marque comme published (ou supprime)                         │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                              │                                          │
│                              ▼                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                         Kafka                                    │   │
│  │  ┌───────────────────────────────────────────────────────────┐  │   │
│  │  │ orders.events.v1                                           │  │   │
│  │  │ [OrderCreated: {id: 123, ...}]                            │  │   │
│  │  └───────────────────────────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

**Structure de la table Outbox** :

```sql
CREATE TABLE outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(255) NOT NULL,
    aggregate_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    published_at TIMESTAMP WITH TIME ZONE,
    
    -- Index pour le polling ou CDC
    INDEX idx_outbox_unpublished (created_at) WHERE published_at IS NULL
);
```

**Implémentation avec Debezium Outbox Extension** :

```json
{
  "name": "orders-outbox-connector",
  "config": {
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "database.hostname": "postgres",
    "database.port": "5432",
    "database.user": "debezium",
    "database.password": "${secrets:db-password}",
    "database.dbname": "orders",
    "database.server.name": "orders",
    
    "table.include.list": "public.outbox",
    "tombstones.on.delete": "false",
    
    "transforms": "outbox",
    "transforms.outbox.type": "io.debezium.transforms.outbox.EventRouter",
    "transforms.outbox.table.fields.additional.placement": "event_type:header:eventType",
    "transforms.outbox.route.by.field": "aggregate_type",
    "transforms.outbox.route.topic.replacement": "${routedByValue}.events"
  }
}
```

**Implémentation Alternative : Polling**

Pour les environnements où le CDC n'est pas possible, un job de polling peut publier les événements de la table outbox.

```java
@Component
public class OutboxPoller {
    
    private final OutboxRepository outboxRepository;
    private final KafkaTemplate<String, byte[]> kafkaTemplate;
    
    @Scheduled(fixedRate = 100)  // Poll toutes les 100ms
    @Transactional
    public void pollAndPublish() {
        List<OutboxEvent> events = outboxRepository
            .findUnpublishedEvents(100);  // Batch de 100
        
        for (OutboxEvent event : events) {
            try {
                // Publication vers Kafka
                kafkaTemplate.send(
                    event.getAggregateType() + ".events",
                    event.getAggregateId(),
                    event.getPayload()
                ).get(5, TimeUnit.SECONDS);  // Attendre la confirmation
                
                // Marquer comme publié
                event.setPublishedAt(Instant.now());
                outboxRepository.save(event);
                
            } catch (Exception e) {
                log.error("Failed to publish outbox event {}", event.getId(), e);
                // L'événement sera retenté au prochain poll
            }
        }
    }
    
    // Nettoyage des événements publiés (job séparé)
    @Scheduled(cron = "0 0 * * * *")  // Toutes les heures
    @Transactional
    public void cleanupOldEvents() {
        Instant cutoff = Instant.now().minus(Duration.ofDays(7));
        int deleted = outboxRepository.deletePublishedBefore(cutoff);
        log.info("Deleted {} old outbox events", deleted);
    }
}
```

**Comparaison CDC vs. Polling** :

| Aspect | CDC (Debezium) | Polling |
|--------|----------------|---------|
| Latence | ~10-100ms | ~100-1000ms (dépend de l'intervalle) |
| Charge BD | Faible (lecture du WAL) | Modérée (requêtes répétées) |
| Complexité | Plus élevée (infrastructure CDC) | Plus simple |
| Ordre garanti | Oui (ordre du WAL) | Oui (si ORDER BY timestamp) |
| Scalabilité | Excellente | Limitée (contention sur la table) |

### Idempotence Côté Consommateur

Même avec exactly-once côté Kafka, les consommateurs doivent être idempotents car des duplications peuvent survenir au niveau applicatif (retry, reprocessing, replay manuel). L'idempotence est la capacité à traiter le même message plusieurs fois sans effet de bord.

**Pattern 1 : Stockage des IDs traités**

Ce pattern maintient une table des événements déjà traités pour détecter et ignorer les duplications.

```java
@Service
public class IdempotentOrderConsumer {
    
    private final OrderRepository orderRepository;
    private final ProcessedEventRepository processedEventRepository;
    
    @Transactional
    public void processOrder(OrderCreatedEvent event) {
        String eventId = event.getHeader().getEventId();
        
        // Vérifier si déjà traité (dans la même transaction)
        if (processedEventRepository.existsById(eventId)) {
            log.info("Event {} already processed, skipping", eventId);
            metrics.counter("events.duplicates.skipped").inc();
            return;
        }
        
        // Traiter l'événement
        Order order = createOrderFromEvent(event);
        orderRepository.save(order);
        
        // Marquer comme traité (dans la même transaction)
        processedEventRepository.save(new ProcessedEvent(
            eventId,
            "OrderCreated",
            event.getHeader().getEventTime(),
            Instant.now()
        ));
        
        metrics.counter("events.processed.success").inc();
    }
}

// Table des événements traités
@Entity
@Table(name = "processed_events", indexes = {
    @Index(name = "idx_processed_events_type_time", columnList = "event_type, event_time")
})
public class ProcessedEvent {
    @Id
    private String eventId;
    private String eventType;
    private Instant eventTime;
    private Instant processedAt;
    
    // TTL pour nettoyage automatique
    // Conserver les IDs assez longtemps pour couvrir la fenêtre de replay possible
}

// Nettoyage périodique
@Scheduled(cron = "0 0 2 * * *")  // Tous les jours à 2h
@Transactional
public void cleanupOldProcessedEvents() {
    // Conserver 30 jours (doit être > rétention Kafka)
    Instant cutoff = Instant.now().minus(Duration.ofDays(30));
    processedEventRepository.deleteByProcessedAtBefore(cutoff);
}
```

**Pattern 2 : Clé naturelle d'idempotence**

Utiliser une contrainte unique sur une clé métier permet à la base de données de rejeter les duplications.

```java
@Service
public class IdempotentPaymentProcessor {
    
    private final PaymentRepository paymentRepository;
    
    @Transactional
    public PaymentResult processPayment(PaymentRequestEvent event) {
        String orderId = event.getOrderId();
        String paymentId = event.getPaymentId();
        
        // Vérifier si un paiement existe déjà pour cette commande
        Optional<Payment> existingPayment = paymentRepository
            .findByOrderId(orderId);
        
        if (existingPayment.isPresent()) {
            log.info("Payment for order {} already exists: {}", 
                orderId, existingPayment.get().getId());
            return PaymentResult.alreadyProcessed(existingPayment.get());
        }
        
        // Créer le paiement avec contrainte unique
        try {
            Payment payment = Payment.builder()
                .id(paymentId)
                .orderId(orderId)  // UNIQUE constraint
                .amount(event.getAmount())
                .status(PaymentStatus.PENDING)
                .createdAt(Instant.now())
                .build();
            
            paymentRepository.save(payment);
            
            // Traitement du paiement (appel au PSP, etc.)
            PaymentStatus result = paymentGateway.charge(payment);
            payment.setStatus(result);
            paymentRepository.save(payment);
            
            return PaymentResult.success(payment);
            
        } catch (DataIntegrityViolationException e) {
            // Race condition : un autre thread a créé le paiement
            // entre notre check et notre insert
            log.info("Concurrent payment creation for order {}", orderId);
            Payment existing = paymentRepository.findByOrderId(orderId)
                .orElseThrow(() -> new IllegalStateException(
                    "Payment should exist after constraint violation"));
            return PaymentResult.alreadyProcessed(existing);
        }
    }
}

// Contrainte unique sur la table
@Entity
@Table(name = "payments", uniqueConstraints = {
    @UniqueConstraint(name = "uk_payments_order_id", columnNames = "order_id")
})
public class Payment {
    @Id
    private String id;
    
    @Column(name = "order_id", nullable = false)
    private String orderId;
    
    // ...
}
```

**Pattern 3 : Idempotence par versioning optimiste**

Pour les mises à jour, le versioning optimiste garantit qu'une mise à jour n'est appliquée qu'une seule fois.

```java
@Entity
public class Order {
    @Id
    private String id;
    
    @Version
    private Long version;
    
    private OrderStatus status;
    // ...
}

@Service
public class IdempotentOrderUpdater {
    
    @Transactional
    public void updateOrderStatus(OrderStatusChangedEvent event) {
        Order order = orderRepository.findById(event.getOrderId())
            .orElseThrow(() -> new OrderNotFoundException(event.getOrderId()));
        
        // Vérifier si la mise à jour est déjà appliquée
        if (order.getStatus() == event.getNewStatus()) {
            log.info("Order {} already in status {}", 
                event.getOrderId(), event.getNewStatus());
            return;
        }
        
        // Vérifier la version (optimistic locking)
        if (event.getExpectedVersion() != null && 
            !event.getExpectedVersion().equals(order.getVersion())) {
            log.warn("Version mismatch for order {}: expected {}, actual {}", 
                event.getOrderId(), event.getExpectedVersion(), order.getVersion());
            // Décider selon le cas : ignorer, alerter, ou forcer
            return;
        }
        
        order.setStatus(event.getNewStatus());
        orderRepository.save(order);  // Version auto-incrémentée
    }
}
```

### Dead Letter Queue (DLQ) Pattern

Les messages qui échouent de manière répétée doivent être isolés pour ne pas bloquer le traitement des autres messages. Le pattern DLQ est essentiel pour la résilience des systèmes événementiels.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      Pattern Dead Letter Queue                          │
│                                                                         │
│  ┌─────────────────┐      ┌─────────────────┐                          │
│  │ orders.events   │      │   Consumer      │                          │
│  │                 │─────▶│                 │                          │
│  └─────────────────┘      │  ┌───────────┐  │                          │
│                           │  │  Process  │  │                          │
│                           │  │           │  │                          │
│                           │  │  Success? │  │                          │
│                           │  └─────┬─────┘  │                          │
│                           │        │        │                          │
│                           │    ┌───┴───┐    │                          │
│                           │   Yes     No    │                          │
│                           │    │       │    │                          │
│                           │    ▼       ▼    │                          │
│                           │ Commit   Retry  │                          │
│                           │          (3x)   │                          │
│                           │           │     │                          │
│                           │      Still No?  │                          │
│                           │           │     │                          │
│                           │           ▼     │                          │
│                           │      Send to    │                          │
│                           │        DLQ      │                          │
│                           └─────────┬───────┘                          │
│                                     │                                   │
│                                     ▼                                   │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │ orders.events.dlq                                                │   │
│  │                                                                  │   │
│  │ Headers:                                                         │   │
│  │ - original_topic: orders.events                                 │   │
│  │ - original_partition: 3                                         │   │
│  │ - original_offset: 12345                                        │   │
│  │ - failure_reason: "Database connection timeout"                 │   │
│  │ - retry_count: 3                                                │   │
│  │ - failed_at: 2024-01-15T10:30:00Z                              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                     │                                   │
│                                     ▼                                   │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │               DLQ Handler (manuel ou automatisé)                 │   │
│  │                                                                  │   │
│  │ Options:                                                         │   │
│  │ 1. Analyse manuelle et correction                               │   │
│  │ 2. Republication vers le topic original après correction        │   │
│  │ 3. Archivage pour audit                                         │   │
│  │ 4. Alerte et escalade                                           │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

**Implémentation Java** :

```java
public class ResilientConsumer {
    
    private static final int MAX_RETRIES = 3;
    private final KafkaTemplate<String, byte[]> dlqProducer;
    private final MeterRegistry metrics;
    
    public void consume(ConsumerRecord<String, OrderEvent> record) {
        int retryCount = 0;
        Exception lastException = null;
        
        while (retryCount < MAX_RETRIES) {
            try {
                processOrder(record.value());
                metrics.counter("consumer.success").inc();
                return;  // Succès
            } catch (RetriableException e) {
                retryCount++;
                lastException = e;
                log.warn("Retry {}/{} for offset {} - {}", 
                    retryCount, MAX_RETRIES, record.offset(), e.getMessage());
                metrics.counter("consumer.retry").inc();
                sleep(exponentialBackoff(retryCount));
            } catch (NonRetriableException e) {
                // Erreur permanente, envoyer directement au DLQ
                log.error("Non-retriable error for offset {}", record.offset(), e);
                sendToDlq(record, e, 0);
                metrics.counter("consumer.dlq.non_retriable").inc();
                return;
            }
        }
        
        // Max retries atteint
        log.error("Max retries reached for offset {}", record.offset(), lastException);
        sendToDlq(record, lastException, retryCount);
        metrics.counter("consumer.dlq.max_retries").inc();
    }
    
    private void sendToDlq(ConsumerRecord<String, OrderEvent> record, 
                          Exception exception, 
                          int retryCount) {
        String dlqTopic = record.topic() + ".dlq";
        
        ProducerRecord<String, byte[]> dlqRecord = new ProducerRecord<>(
            dlqTopic,
            record.key(),
            serialize(record.value())
        );
        
        // Ajouter des headers de diagnostic
        dlqRecord.headers()
            .add("dlq.original.topic", record.topic().getBytes(StandardCharsets.UTF_8))
            .add("dlq.original.partition", String.valueOf(record.partition()).getBytes())
            .add("dlq.original.offset", String.valueOf(record.offset()).getBytes())
            .add("dlq.original.timestamp", String.valueOf(record.timestamp()).getBytes())
            .add("dlq.failure.reason", exception.getMessage().getBytes(StandardCharsets.UTF_8))
            .add("dlq.failure.exception", exception.getClass().getName().getBytes())
            .add("dlq.retry.count", String.valueOf(retryCount).getBytes())
            .add("dlq.failed.at", Instant.now().toString().getBytes())
            .add("dlq.consumer.group", consumerGroupId.getBytes())
            .add("dlq.consumer.instance", instanceId.getBytes());
        
        // Stack trace complet pour le débogage
        StringWriter sw = new StringWriter();
        exception.printStackTrace(new PrintWriter(sw));
        dlqRecord.headers().add("dlq.stacktrace", sw.toString().getBytes());
        
        try {
            dlqProducer.send(dlqRecord).get(5, TimeUnit.SECONDS);
            log.info("Message sent to DLQ {} after {} retries", dlqTopic, retryCount);
        } catch (Exception e) {
            // Échec critique : impossible d'envoyer au DLQ
            log.error("CRITICAL: Failed to send to DLQ", e);
            metrics.counter("consumer.dlq.send_failed").inc();
            // Alerter immédiatement
            alertService.critical("DLQ send failure", e);
        }
    }
    
    private long exponentialBackoff(int retryCount) {
        // Backoff exponentiel avec jitter
        long baseDelay = 1000L * (long) Math.pow(2, retryCount);  // 2s, 4s, 8s
        long jitter = (long) (baseDelay * 0.2 * Math.random());   // ±20% jitter
        return baseDelay + jitter;
    }
    
    private void sleep(long millis) {
        try {
            Thread.sleep(millis);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new RuntimeException("Consumer interrupted during backoff", e);
        }
    }
}
```

### Traitement des Messages DLQ

Les messages dans le DLQ doivent être analysés et traités. Plusieurs stratégies sont possibles.

**Stratégie 1 : Analyse et correction manuelle**

```java
@RestController
@RequestMapping("/api/dlq")
public class DlqManagementController {
    
    private final KafkaConsumer<String, byte[]> dlqConsumer;
    private final KafkaProducer<String, byte[]> replayProducer;
    
    @GetMapping("/{topic}/messages")
    public List<DlqMessage> getMessages(
            @PathVariable String topic,
            @RequestParam(defaultValue = "100") int limit) {
        
        // Lire les messages du DLQ sans commit
        dlqConsumer.assign(getPartitions(topic + ".dlq"));
        dlqConsumer.seekToBeginning(dlqConsumer.assignment());
        
        List<DlqMessage> messages = new ArrayList<>();
        ConsumerRecords<String, byte[]> records = dlqConsumer.poll(Duration.ofSeconds(10));
        
        for (ConsumerRecord<String, byte[]> record : records) {
            if (messages.size() >= limit) break;
            messages.add(DlqMessage.from(record));
        }
        
        return messages;
    }
    
    @PostMapping("/{topic}/replay/{offset}")
    public ResponseEntity<?> replayMessage(
            @PathVariable String topic,
            @PathVariable long offset) {
        
        // Récupérer le message
        ConsumerRecord<String, byte[]> dlqRecord = findMessage(topic + ".dlq", offset);
        
        // Extraire le topic original des headers
        String originalTopic = new String(
            dlqRecord.headers().lastHeader("dlq.original.topic").value());
        
        // Republier vers le topic original
        ProducerRecord<String, byte[]> replayRecord = new ProducerRecord<>(
            originalTopic,
            dlqRecord.key(),
            dlqRecord.value()
        );
        replayRecord.headers().add("dlq.replayed", "true".getBytes());
        replayRecord.headers().add("dlq.replay.time", Instant.now().toString().getBytes());
        
        replayProducer.send(replayRecord).get();
        
        // Optionnel : supprimer du DLQ après replay réussi
        // (ou marquer comme traité dans une table séparée)
        
        return ResponseEntity.ok().build();
    }
    
    @DeleteMapping("/{topic}/messages/{offset}")
    public ResponseEntity<?> acknowledgeMessage(
            @PathVariable String topic,
            @PathVariable long offset,
            @RequestBody AcknowledgeRequest request) {
        
        // Enregistrer la raison de l'acquittement (ignoré, corrigé manuellement, etc.)
        dlqAuditService.acknowledge(topic, offset, request.getReason(), request.getUser());
        
        return ResponseEntity.ok().build();
    }
}
```

**Stratégie 2 : Retry automatique avec délai**

```java
@Component
public class DlqRetryProcessor {
    
    private final KafkaTemplate<String, byte[]> producer;
    
    @KafkaListener(topics = "orders.events.dlq", groupId = "dlq-retry-processor")
    public void processRetryable(ConsumerRecord<String, byte[]> record) {
        // Vérifier si le message peut être retenté
        Header retryCountHeader = record.headers().lastHeader("dlq.retry.count");
        int previousRetries = Integer.parseInt(
            new String(retryCountHeader.value()));
        
        if (previousRetries >= 10) {
            // Trop de retries, archiver et alerter
            archiveAndAlert(record);
            return;
        }
        
        // Vérifier le délai depuis l'échec
        Header failedAtHeader = record.headers().lastHeader("dlq.failed.at");
        Instant failedAt = Instant.parse(new String(failedAtHeader.value()));
        Duration timeSinceFail = Duration.between(failedAt, Instant.now());
        
        // Attendre un délai croissant avant retry
        Duration requiredDelay = Duration.ofMinutes(previousRetries * 5L);  // 0, 5, 10, 15... minutes
        
        if (timeSinceFail.compareTo(requiredDelay) < 0) {
            // Pas encore temps de retry, remettre dans le DLQ
            republishToDlq(record);
            return;
        }
        
        // Retry vers le topic original
        String originalTopic = new String(
            record.headers().lastHeader("dlq.original.topic").value());
        
        ProducerRecord<String, byte[]> retryRecord = new ProducerRecord<>(
            originalTopic,
            record.key(),
            record.value()
        );
        retryRecord.headers().add("dlq.retry.attempt", 
            String.valueOf(previousRetries + 1).getBytes());
        
        producer.send(retryRecord);
        log.info("Retrying message from DLQ, attempt {}", previousRetries + 1);
    }
}
```

**Stratégie 3 : Alertes et escalade**

```java
@Component
public class DlqAlertProcessor {
    
    private final AlertService alertService;
    private final MetricRegistry metrics;
    
    @Scheduled(fixedRate = 60000)  // Toutes les minutes
    public void checkDlqHealth() {
        for (String dlqTopic : getDlqTopics()) {
            long messageCount = getMessageCount(dlqTopic);
            long oldestMessageAge = getOldestMessageAge(dlqTopic);
            
            metrics.gauge("dlq.message_count", () -> messageCount, 
                Tags.of("topic", dlqTopic));
            metrics.gauge("dlq.oldest_message_age_seconds", () -> oldestMessageAge, 
                Tags.of("topic", dlqTopic));
            
            // Alertes basées sur des seuils
            if (messageCount > 1000) {
                alertService.warning(
                    "DLQ " + dlqTopic + " has " + messageCount + " messages");
            }
            
            if (oldestMessageAge > Duration.ofHours(24).getSeconds()) {
                alertService.critical(
                    "DLQ " + dlqTopic + " has messages older than 24 hours");
            }
        }
    }
}
```

### Patterns Avancés de Communication

Au-delà des patterns de base (publication/abonnement), Kafka supporte des patterns de communication plus sophistiqués nécessaires pour certains cas d'usage complexes.

**Pattern Request-Reply**

Bien que Kafka soit conçu pour la communication asynchrone, certains cas d'usage nécessitent une sémantique requête-réponse. Ce pattern implémente une communication pseudo-synchrone au-dessus de Kafka.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Pattern Request-Reply avec Kafka                     │
│                                                                         │
│  Service A (Requester)                      Service B (Responder)       │
│  ┌─────────────────────┐                   ┌─────────────────────┐     │
│  │  1. Générer         │                   │                     │     │
│  │     correlation_id  │                   │                     │     │
│  │  2. Envoyer requête │                   │                     │     │
│  │     avec reply_topic│                   │                     │     │
│  └──────────┬──────────┘                   └──────────┬──────────┘     │
│             │                                         │                 │
│             ▼                                         │                 │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                      Topic: requests                             │   │
│  │  {correlation_id: "abc", reply_topic: "replies.svc-a", ...}     │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                         │                               │
│                                         ▼                               │
│                              ┌──────────┴──────────┐                   │
│                              │  3. Traiter requête │                   │
│                              │  4. Envoyer réponse │                   │
│                              └──────────┬──────────┘                   │
│                                         │                               │
│                                         ▼                               │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                   Topic: replies.svc-a                           │   │
│  │  {correlation_id: "abc", result: "success", ...}                │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│             │                                                           │
│             ▼                                                           │
│  ┌──────────┴──────────┐                                               │
│  │  5. Corrélation     │                                               │
│  │  6. Retourner       │                                               │
│  │     réponse         │                                               │
│  └─────────────────────┘                                               │
└─────────────────────────────────────────────────────────────────────────┘
```

**Implémentation Request-Reply** :

```java
@Service
public class KafkaRequestReplyService {
    
    private final KafkaTemplate<String, Request> requestTemplate;
    private final Map<String, CompletableFuture<Response>> pendingRequests;
    private final String replyTopic;
    
    public KafkaRequestReplyService(String instanceId) {
        this.pendingRequests = new ConcurrentHashMap<>();
        this.replyTopic = "replies." + instanceId;
    }
    
    public CompletableFuture<Response> sendRequest(Request request, Duration timeout) {
        String correlationId = UUID.randomUUID().toString();
        
        // Créer le future pour la réponse
        CompletableFuture<Response> future = new CompletableFuture<>();
        pendingRequests.put(correlationId, future);
        
        // Configurer le timeout
        future.orTimeout(timeout.toMillis(), TimeUnit.MILLISECONDS)
            .whenComplete((result, error) -> {
                pendingRequests.remove(correlationId);
                if (error instanceof TimeoutException) {
                    log.warn("Request {} timed out after {}", correlationId, timeout);
                }
            });
        
        // Envoyer la requête avec les headers de corrélation
        ProducerRecord<String, Request> record = new ProducerRecord<>(
            "requests",
            request.getKey(),
            request
        );
        record.headers()
            .add("correlation_id", correlationId.getBytes())
            .add("reply_topic", replyTopic.getBytes());
        
        requestTemplate.send(record);
        
        return future;
    }
    
    @KafkaListener(topicPattern = "replies\\..*", groupId = "${instance.id}")
    public void handleReply(ConsumerRecord<String, Response> record) {
        String correlationId = new String(
            record.headers().lastHeader("correlation_id").value());
        
        CompletableFuture<Response> future = pendingRequests.get(correlationId);
        if (future != null) {
            future.complete(record.value());
        } else {
            log.warn("Received reply for unknown correlation_id: {}", correlationId);
        }
    }
}
```

> **Décision architecturale**
>
> *Question* : Quand utiliser Request-Reply avec Kafka plutôt qu'un appel HTTP direct ?
>
> *Utiliser Request-Reply Kafka quand* :
> - Le traitement peut prendre plusieurs secondes/minutes
> - Le responder peut être down temporairement (découplage)
> - Besoin de retry automatique et persistance
> - Audit trail des requêtes/réponses requis
>
> *Utiliser HTTP quand* :
> - Latence < 100ms requise
> - Communication point-à-point simple
> - Pas besoin de persistance des requêtes

**Pattern Saga Chorégraphiée**

Le pattern Saga gère les transactions distribuées à travers plusieurs services via une séquence d'événements. Chaque service publie un événement après avoir terminé sa partie, et les autres services réagissent à ces événements.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Saga Chorégraphiée : Commande                        │
│                                                                         │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐         │
│  │ Orders   │    │ Inventory│    │ Payment  │    │ Shipping │         │
│  └────┬─────┘    └────┬─────┘    └────┬─────┘    └────┬─────┘         │
│       │               │               │               │                │
│       │ OrderCreated  │               │               │                │
│       │──────────────▶│               │               │                │
│       │               │               │               │                │
│       │               │InventoryReserved              │                │
│       │               │──────────────▶│               │                │
│       │               │               │               │                │
│       │               │               │ PaymentProcessed               │
│       │               │               │──────────────▶│                │
│       │               │               │               │                │
│       │               │               │               │ ShipmentCreated│
│       │◀──────────────┼───────────────┼───────────────│                │
│       │               │               │               │                │
│       │ OrderCompleted│               │               │                │
│       │──────────────▶│──────────────▶│──────────────▶│                │
│                                                                         │
│  Compensation (en cas d'échec Payment):                                │
│       │               │PaymentFailed  │               │                │
│       │               │◀──────────────│               │                │
│       │               │InventoryReleased              │                │
│       │◀──────────────│               │               │                │
│       │ OrderCancelled│               │               │                │
└─────────────────────────────────────────────────────────────────────────┘
```

**Implémentation de la Saga (Service Orders)** :

```java
@Service
public class OrderSagaParticipant {
    
    private final OrderRepository orderRepository;
    private final KafkaTemplate<String, Object> kafkaTemplate;
    
    @KafkaListener(topics = "inventory.events", groupId = "orders-saga")
    public void onInventoryEvent(InventoryEvent event) {
        if (event instanceof InventoryReservedEvent reserved) {
            log.info("Inventory reserved for order {}", reserved.getOrderId());
            orderRepository.updateStatus(reserved.getOrderId(), 
                OrderStatus.INVENTORY_RESERVED);
        } else if (event instanceof InventoryReservationFailedEvent failed) {
            log.warn("Inventory reservation failed for order {}", failed.getOrderId());
            cancelOrder(failed.getOrderId(), "Insufficient inventory");
        }
    }
    
    @KafkaListener(topics = "payment.events", groupId = "orders-saga")
    public void onPaymentEvent(PaymentEvent event) {
        if (event instanceof PaymentProcessedEvent processed) {
            log.info("Payment processed for order {}", processed.getOrderId());
            orderRepository.updateStatus(processed.getOrderId(), OrderStatus.PAID);
        } else if (event instanceof PaymentFailedEvent failed) {
            log.warn("Payment failed for order {}", failed.getOrderId());
            compensateOrder(failed.getOrderId(), "Payment failed: " + failed.getReason());
        }
    }
    
    @KafkaListener(topics = "shipping.events", groupId = "orders-saga")
    public void onShippingEvent(ShippingEvent event) {
        if (event instanceof ShipmentCreatedEvent shipped) {
            log.info("Shipment created for order {}", shipped.getOrderId());
            completeOrder(shipped.getOrderId());
        }
    }
    
    private void compensateOrder(String orderId, String reason) {
        Order order = orderRepository.findById(orderId).orElseThrow();
        
        // Publier l'événement de compensation
        OrderCancelledEvent cancelEvent = OrderCancelledEvent.builder()
            .orderId(orderId)
            .reason(reason)
            .cancelledAt(Instant.now())
            .build();
        
        kafkaTemplate.send("orders.events", orderId, cancelEvent);
        orderRepository.updateStatus(orderId, OrderStatus.CANCELLED);
        
        log.info("Order {} cancelled due to: {}", orderId, reason);
    }
}
```

**Implémentation de la Saga (Service Inventory)** :

```java
@Service
public class InventorySagaParticipant {
    
    private final InventoryRepository inventoryRepository;
    private final ReservationRepository reservationRepository;
    private final KafkaTemplate<String, Object> kafkaTemplate;
    
    @KafkaListener(topics = "orders.events", groupId = "inventory-saga")
    public void onOrderEvent(OrderEvent event) {
        if (event instanceof OrderCreatedEvent created) {
            try {
                reserveInventory(created);
            } catch (InsufficientInventoryException e) {
                publishReservationFailed(created.getOrderId(), e.getMessage());
            }
        } else if (event instanceof OrderCancelledEvent cancelled) {
            // Compensation : libérer l'inventaire réservé
            releaseInventory(cancelled.getOrderId());
        }
    }
    
    @Transactional
    private void reserveInventory(OrderCreatedEvent event) {
        List<Reservation> reservations = new ArrayList<>();
        
        for (OrderItem item : event.getItems()) {
            Inventory inventory = inventoryRepository
                .findByProductIdWithLock(item.getProductId())
                .orElseThrow(() -> new ProductNotFoundException(item.getProductId()));
            
            if (inventory.getAvailable() < item.getQuantity()) {
                throw new InsufficientInventoryException(
                    "Product " + item.getProductId() + ": requested " + 
                    item.getQuantity() + ", available " + inventory.getAvailable());
            }
            
            inventory.reserve(item.getQuantity());
            inventoryRepository.save(inventory);
            
            reservations.add(Reservation.builder()
                .orderId(event.getOrderId())
                .productId(item.getProductId())
                .quantity(item.getQuantity())
                .reservedAt(Instant.now())
                .build());
        }
        
        reservationRepository.saveAll(reservations);
        
        // Publier le succès
        kafkaTemplate.send("inventory.events", event.getOrderId(),
            InventoryReservedEvent.builder()
                .orderId(event.getOrderId())
                .items(event.getItems())
                .reservedAt(Instant.now())
                .build());
    }
    
    @Transactional
    private void releaseInventory(String orderId) {
        List<Reservation> reservations = reservationRepository.findByOrderId(orderId);
        
        if (reservations.isEmpty()) {
            log.info("No reservations found for order {}", orderId);
            return;
        }
        
        for (Reservation reservation : reservations) {
            Inventory inventory = inventoryRepository
                .findByProductId(reservation.getProductId())
                .orElseThrow();
            
            inventory.release(reservation.getQuantity());
            inventoryRepository.save(inventory);
        }
        
        reservationRepository.deleteAll(reservations);
        
        kafkaTemplate.send("inventory.events", orderId,
            InventoryReleasedEvent.builder()
                .orderId(orderId)
                .releasedAt(Instant.now())
                .build());
        
        log.info("Released inventory for order {}", orderId);
    }
}
```

### Patterns de Partitionnement et Ordering

Le partitionnement est crucial pour la scalabilité et l'ordre des messages. Le choix de la clé de partitionnement impacte directement les garanties d'ordering et la distribution de charge.

**Stratégies de Partitionnement** :

| Stratégie | Clé de partition | Garantie d'ordre | Distribution | Cas d'usage |
|-----------|------------------|------------------|--------------|-------------|
| Par entité | entity_id | Ordre par entité | Bonne si entités équilibrées | Commandes, utilisateurs |
| Par tenant | tenant_id | Ordre par tenant | Risque de hot partition | SaaS multi-tenant |
| Par région | region_code | Ordre par région | Limitée (peu de régions) | Données géographiques |
| Round-robin | null | Aucune | Optimale | Logs, métriques |
| Par temps | timestamp bucket | Temporel approximatif | Bonne | Time-series |
| Composite | tenant:entity | Ordre par entité dans tenant | Excellente | Multi-tenant avec entités |

**Implémentation du partitionnement custom** :

```java
public class TenantAwarePartitioner implements Partitioner {
    
    @Override
    public int partition(String topic, Object key, byte[] keyBytes,
                        Object value, byte[] valueBytes, Cluster cluster) {
        
        List<PartitionInfo> partitions = cluster.partitionsForTopic(topic);
        int numPartitions = partitions.size();
        
        if (key == null) {
            // Round-robin pour les messages sans clé
            return ThreadLocalRandom.current().nextInt(numPartitions);
        }
        
        // Extraire le tenant_id de la clé composite "tenant_id:entity_id"
        String keyStr = (String) key;
        String tenantId = extractTenantId(keyStr);
        
        // Hash cohérent sur le tenant_id
        // Tous les messages du même tenant vont dans la même partition
        int hash = Math.abs(murmur2(tenantId.getBytes(StandardCharsets.UTF_8)));
        return hash % numPartitions;
    }
    
    private String extractTenantId(String compositeKey) {
        int separatorIndex = compositeKey.indexOf(':');
        return separatorIndex > 0 ? compositeKey.substring(0, separatorIndex) : compositeKey;
    }
    
    // Implémentation Murmur2 (même algo que le partitioner par défaut de Kafka)
    private int murmur2(byte[] data) {
        int length = data.length;
        int seed = 0x9747b28c;
        int m = 0x5bd1e995;
        int r = 24;
        int h = seed ^ length;
        int length4 = length / 4;
        
        for (int i = 0; i < length4; i++) {
            int i4 = i * 4;
            int k = (data[i4] & 0xff) + ((data[i4 + 1] & 0xff) << 8) + 
                    ((data[i4 + 2] & 0xff) << 16) + ((data[i4 + 3] & 0xff) << 24);
            k *= m;
            k ^= k >>> r;
            k *= m;
            h *= m;
            h ^= k;
        }
        
        // Traiter les bytes restants
        switch (length % 4) {
            case 3: h ^= (data[(length & ~3) + 2] & 0xff) << 16;
            case 2: h ^= (data[(length & ~3) + 1] & 0xff) << 8;
            case 1: h ^= data[length & ~3] & 0xff;
                    h *= m;
        }
        
        h ^= h >>> 13;
        h *= m;
        h ^= h >>> 15;
        
        return h;
    }
    
    @Override
    public void close() {}
    
    @Override
    public void configure(Map<String, ?> configs) {}
}
```

**Gestion du Hot Partition** :

Un hot partition survient quand une clé de partitionnement est surreprésentée, causant une distribution inégale de la charge. C'est un problème courant en multi-tenant quand un gros client génère beaucoup plus de trafic que les autres.

```java
@Component
public class AdaptivePartitioner implements Partitioner {
    
    private final Map<String, HotKeyTracker> keyTrackers = new ConcurrentHashMap<>();
    private final long hotKeyThreshold;
    private final int spreadFactor;
    
    public AdaptivePartitioner() {
        this.hotKeyThreshold = 10000;  // Messages par minute déclenchant le spread
        this.spreadFactor = 4;          // Nombre de partitions pour distribuer les hot keys
    }
    
    @Override
    public int partition(String topic, Object key, byte[] keyBytes,
                        Object value, byte[] valueBytes, Cluster cluster) {
        
        int numPartitions = cluster.partitionsForTopic(topic).size();
        
        if (key == null) {
            return ThreadLocalRandom.current().nextInt(numPartitions);
        }
        
        String keyStr = (String) key;
        String trackingKey = topic + ":" + keyStr;
        
        // Tracker pour cette clé
        HotKeyTracker tracker = keyTrackers.computeIfAbsent(trackingKey, 
            k -> new HotKeyTracker());
        tracker.increment();
        
        // Calculer la partition de base
        int basePartition = Math.abs(keyStr.hashCode()) % numPartitions;
        
        if (tracker.isHot(hotKeyThreshold)) {
            // Hot key détectée : distribuer sur plusieurs partitions
            // Le suffix est basé sur un compteur pour maintenir un ordre relatif
            int spreadIndex = (int) (tracker.getCount() % spreadFactor);
            int targetPartition = (basePartition + spreadIndex) % numPartitions;
            
            log.debug("Hot key {} spreading to partition {} (base: {})", 
                keyStr, targetPartition, basePartition);
            
            return targetPartition;
        }
        
        return basePartition;
    }
    
    // Reset des compteurs toutes les minutes
    @Scheduled(fixedRate = 60000)
    public void resetCounters() {
        keyTrackers.values().forEach(HotKeyTracker::reset);
    }
    
    private static class HotKeyTracker {
        private final AtomicLong count = new AtomicLong(0);
        private volatile long lastResetTime = System.currentTimeMillis();
        
        void increment() {
            count.incrementAndGet();
        }
        
        long getCount() {
            return count.get();
        }
        
        boolean isHot(long threshold) {
            long elapsed = System.currentTimeMillis() - lastResetTime;
            double rate = count.get() * 60000.0 / Math.max(elapsed, 1);
            return rate > threshold;
        }
        
        void reset() {
            count.set(0);
            lastResetTime = System.currentTimeMillis();
        }
    }
}
```

> **Anti-patron**
>
> *Erreur courante* : Utiliser une clé avec cardinalité trop faible (ex: `country_code` avec seulement 10 valeurs pour un topic à 100 partitions).
>
> *Conséquence* : 90% des partitions sont vides, 10% sont surchargées.
>
> *Solution* : Utiliser une clé composite `country:customer_id` ou accepter de perdre l'ordering par pays.

---

## III.7.5 Résumé

Ce chapitre a exploré les patrons d'interaction Kafka, des modèles architecturaux éprouvés qui structurent la communication événementielle à l'échelle de l'entreprise. Ces patterns ne sont pas des abstractions théoriques mais des solutions concrètes à des problèmes réels rencontrés en production. Leur maîtrise est essentielle pour tout architecte travaillant avec Kafka.

### Leçons des Cas Problématiques

Les cas de terrain présentés illustrent les défis fondamentaux des architectures événementielles et les conséquences concrètes d'une mauvaise conception :

*Incohérence transactionnelle* : L'absence d'atomicité entre l'écriture en base et la publication d'événement cause des états incohérents qui se manifestent par des commandes fantômes, des factures en double, ou des stocks négatifs. Le pattern Outbox Transactionnel résout ce problème en utilisant la base de données comme intermédiaire fiable, garantissant que l'événement n'est publié que si l'écriture métier a réussi.

*Tempêtes de retry* : Les appels externes bloquants dans les consumers peuvent causer des boucles infinies qui saturent les systèmes. Un service SMTP lent a causé 50 000 tentatives d'envoi pour 500 emails en 30 minutes. Les Dead Letter Queues isolent les messages problématiques, les circuit breakers protègent contre les cascades de défaillances, et le backoff exponentiel évite l'aggravation des problèmes.

*Isolation insuffisante* : Les consumers avec des vitesses de traitement différentes dans le même groupe causent des interférences. Un service ML traitant des événements en 2 secondes a bloqué tous les services de notification pendant le Black Friday. La solution est simple : consumer groups séparés pour chaque service, permettant une progression indépendante.

*Messages poison* : Les messages malformés ou non désérialisables bloquent tout le pipeline si non gérés. Un bug de schéma a causé 6 heures d'indisponibilité totale. Les DLQ avec métadonnées de diagnostic, la validation à la source, et les mécanismes de skip gracieux sont essentiels.

*Duplications invisibles* : L'auto-commit est dangereux pour les traitements critiques car il ne garantit pas que le traitement a réussi. 0.5% des factures étaient soit manquantes, soit en double. Le commit manuel après traitement réussi, combiné avec l'idempotence côté consommateur, est la seule approche fiable.

**Patterns de résilience essentiels** :

| Pattern | Problème résolu | Implémentation |
|---------|-----------------|----------------|
| Circuit Breaker | Cascade de défaillances | Resilience4j, Hystrix |
| Backpressure | Surcharge du consumer | Rate limiting, semaphores |
| Bulkhead | Isolation des ressources | Thread pools séparés |
| Timeout | Blocage indéfini | Timeouts explicites sur chaque opération |

### Data Mesh avec Kafka

Le Data Mesh transforme l'approche de gestion des données en décentralisant la propriété vers les domaines métier. Cette transformation est organisationnelle autant que technique, et Kafka est un enabler naturel de ce paradigme.

*Propriété par domaine* : Chaque équipe métier publie ses données comme des produits avec des conventions de nommage claires (`{domaine}.{type}.{version}`). L'équipe Commandes est responsable des événements de commande, de leur qualité, de leur documentation, et de leur évolution. Cette responsabilité inclut les SLA, les schémas, et le support aux consommateurs.

*Données comme produit* : Les événements sont documentés, versionnés, et accompagnés de SLA mesurables. Un produit de données doit être découvrable (catalogue), compréhensible (documentation), fiable (SLA), et interopérable (formats standards). Le Schema Registry fournit la gouvernance des schémas, et les contrats de qualité définissent les attentes.

*Plateforme self-service* : Un catalogue de données, des templates de provisionnement, et des outils CLI permettent aux équipes de créer des produits de données en autonomie. L'indicateur de maturité : une nouvelle équipe peut publier un produit de données en moins d'une journée sans intervention de l'équipe plateforme.

*Gouvernance fédérée* : Standards globaux (formats, sécurité, métadonnées) combinés avec autonomie locale (schémas métier, SLA spécifiques). Les standards non négociables incluent les conventions de nommage, le format de sérialisation (Avro), les métadonnées obligatoires, et les exigences de sécurité.

**Métriques de santé du Data Mesh** :

| Métrique | Description | Cible |
|----------|-------------|-------|
| Time to first product | Temps pour publier un premier produit | < 1 jour |
| Product discovery rate | % de produits découvrables dans le catalogue | > 95% |
| SLA compliance | % de produits respectant leurs SLA | > 99% |
| Consumer satisfaction | NPS des consommateurs de données | > 50 |

### Kafka Connect pour l'Intégration

Kafka Connect simplifie l'intégration avec les systèmes externes en fournissant un framework configuration-driven plutôt que code-driven :

*Change Data Capture (CDC)* : Debezium capture les changements des bases de données et les publie comme événements. C'est le pattern le plus puissant pour intégrer des systèmes legacy sans les modifier. La latence typique est de 10-100ms, et l'ordre des événements est garanti par la lecture du WAL.

*Sink vers Data Lake* : Les connecteurs S3/Iceberg permettent d'alimenter les systèmes analytiques en temps réel avec partitionnement automatique par date/heure. Le format Parquet avec compression Snappy offre un excellent compromis entre taille et performance de lecture.

*Transformations SMT* : Les Single Message Transforms permettent de modifier les messages à la volée sans code custom — routage, filtrage, masquage de données sensibles, enrichissement avec métadonnées.

*Scalabilité et HA* : Le cluster Kafka Connect distribue automatiquement les tâches entre les workers. En cas de défaillance d'un worker, les tâches sont redistribuées sans perte de données grâce au stockage des offsets dans Kafka.

*Gestion des erreurs* : Les Dead Letter Queues de Kafka Connect isolent les messages problématiques avec des métadonnées de diagnostic complètes (topic original, offset, exception, stack trace).

### Garanties de Livraison

Les trois sémantiques de livraison offrent des trade-offs différents que l'architecte doit comprendre pour choisir la bonne approche :

*At-most-once* : Simple mais risque de perte. Latence minimale (~1-2ms), débit maximal. Approprié pour les logs non critiques et les métriques approximatives. Ne jamais utiliser pour des données métier importantes.

*At-least-once* : Garantit la livraison mais peut dupliquer. Latence modérée (~5-20ms), débit élevé. C'est la solution standard pour la plupart des cas d'usage, combinée avec l'idempotence côté consommateur. L'idempotence peut être implémentée via le stockage des IDs traités, les clés naturelles avec contraintes d'unicité, ou le versioning optimiste.

*Exactly-once* : Garantie maximale via les transactions Kafka. Latence plus élevée (~20-100ms), débit réduit de 20-30%. Nécessaire pour les systèmes financiers critiques où toute perte ou duplication est inacceptable. Limité à l'écosystème Kafka — les appels à des systèmes externes restent at-least-once.

**Le pattern Outbox Transactionnel** garantit l'atomicité entre les opérations de base de données et la publication d'événements. C'est l'un des patterns les plus importants car il résout le problème classique du double-commit qui cause des incohérences entre l'état de la base de données et les événements publiés.

**L'idempotence côté consommateur** est toujours nécessaire, quelle que soit la sémantique de livraison Kafka. Les trois approches principales sont :
1. Stockage des IDs traités dans une table dédiée
2. Clés naturelles avec contraintes d'unicité
3. Versioning optimiste pour les mises à jour

**Le pattern DLQ (Dead Letter Queue)** est essentiel pour la résilience. Les messages en échec sont isolés avec des métadonnées de diagnostic, permettant une analyse et un traitement ultérieur sans bloquer le flux principal.

### Principes Directeurs pour l'Architecte

1. **Concevoir pour l'échec** : Tout composant peut échouer à tout moment. Les patterns DLQ, retry avec backoff exponentiel, et circuit breaker sont des nécessités architecturales, pas des optimisations optionnelles. Le coût de leur absence se mesure en heures d'indisponibilité et en données perdues.

2. **Isoler les consommateurs** : Chaque service avec des besoins différents doit avoir son propre consumer group. L'isolation évite les effets de cascade où un service lent bloque tous les autres. Le coût en ressources (chaque groupe lit toutes les partitions) est largement compensé par la robustesse.

3. **Préférer at-least-once avec idempotence** : C'est le meilleur compromis entre fiabilité et complexité pour la majorité des cas d'usage. L'exactly-once a un coût en performance et en complexité qui n'est justifié que pour les cas véritablement critiques.

4. **Utiliser Kafka Connect plutôt que du code custom** : Pour les intégrations standard (CDC, sinks vers S3/Snowflake/Elasticsearch), les connecteurs sont plus fiables, plus performants, et plus maintenables que le code custom. L'écosystème de 200+ connecteurs couvre la majorité des cas.

5. **Adopter le Data Mesh progressivement** : Commencer par un domaine pilote, prouver la valeur, puis étendre. C'est un changement organisationnel autant que technique, et la résistance au changement est le principal obstacle.

6. **Monitorer proactivement** : Le lag, les erreurs, le throughput, et la taille des DLQ doivent être surveillés avec des alertes. Les problèmes détectés tôt sont exponentiellement plus faciles à résoudre. Un lag qui augmente est souvent le premier signe d'un problème plus grave.

7. **Documenter les décisions** : Chaque choix de pattern (sémantique de livraison, stratégie d'idempotence, configuration DLQ) doit être documenté avec sa justification. Les architectures événementielles sont complexes, et la documentation est essentielle pour l'onboarding et la maintenance.

---

### Vers le Chapitre Suivant

Les patrons d'interaction définissent comment les messages circulent dans l'écosystème Kafka et comment gérer les cas d'erreur. Le chapitre suivant, « Conception d'Application de Traitement de Flux en Continu », explorera Kafka Streams — la bibliothèque qui permet de transformer, agréger, joindre, et enrichir ces flux d'événements en temps réel, ouvrant la porte à des cas d'usage avancés comme les vues matérialisées, les agrégations en fenêtres, et le traitement stateful.

---

*Volume III : Apache Kafka - Guide de l'Architecte*

*Chapitre III.7 — Patrons d'Interaction Kafka*

*Monographie « L'Entreprise Agentique »*
