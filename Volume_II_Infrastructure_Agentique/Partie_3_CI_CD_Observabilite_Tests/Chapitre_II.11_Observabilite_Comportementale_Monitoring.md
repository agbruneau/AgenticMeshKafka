# Chapitre II.11 — Observabilité Comportementale et Monitoring

---

## Introduction

L'observabilité des systèmes agentiques constitue un défi fondamentalement différent de la supervision traditionnelle des applications logicielles. Là où les métriques classiques se concentrent sur la disponibilité, les temps de réponse et les taux d'erreur, l'observabilité d'un agent cognitif doit capturer la qualité du raisonnement, la pertinence des décisions et l'alignement comportemental avec les objectifs définis. Cette dimension cognitive transforme la discipline de l'observabilité en une pratique hybride, à la croisée de l'ingénierie logicielle et de l'évaluation de l'intelligence artificielle.

Les systèmes multi-agents introduisent une complexité supplémentaire par la nature émergente de leurs comportements collectifs. Un agent peut fonctionner parfaitement en isolation tout en contribuant à des dysfonctionnements systémiques lorsqu'il interagit avec d'autres agents. Cette propriété émergente exige des mécanismes d'observation capables de corréler les comportements individuels avec les résultats globaux du système.

Ce chapitre explore les fondements de l'observabilité comportementale pour les architectures agentiques. Nous examinerons d'abord les défis spécifiques qui distinguent cette discipline de l'observabilité traditionnelle. L'implémentation du traçage distribué via OpenTelemetry sera détaillée, avec une attention particulière à l'instrumentation des chaînes de raisonnement. Nous aborderons ensuite les métriques de performance cognitive, les Key Agent Indicators (KAIs), qui complètent les indicateurs techniques classiques. La détection de dérive comportementale, essentielle pour maintenir l'alignement des agents au fil du temps, sera analysée en profondeur. Enfin, nous présenterons l'architecture d'un cockpit de supervision intégré permettant aux équipes opérationnelles de piloter efficacement leurs systèmes agentiques.

L'objectif de ce chapitre est d'équiper les équipes d'ingénierie des outils conceptuels et pratiques nécessaires pour maintenir une visibilité complète sur le comportement de leurs agents en production, condition sine qua non de la confiance opérationnelle dans les systèmes cognitifs autonomes.

---

## II.11.1 Défis de l'Observabilité des Systèmes Agentiques

### La Rupture avec l'Observabilité Traditionnelle

L'observabilité traditionnelle, formalisée autour des trois piliers que sont les métriques, les logs et les traces, a été conçue pour des systèmes déterministes. Une requête HTTP donnée, avec les mêmes paramètres et le même état système, produira toujours le même résultat. Cette prédictibilité permet de définir des seuils d'alerte clairs et des signatures d'anomalie reproductibles.

Les agents cognitifs rompent fondamentalement avec ce paradigme. Deux requêtes identiques peuvent générer des réponses différentes selon le contexte conversationnel accumulé, les variations stochastiques du modèle de langage, ou même l'ordre de traitement des informations. Cette non-reproductibilité intrinsèque invalide les approches de monitoring basées sur la comparaison avec un comportement de référence fixe.

De plus, la notion même d'« erreur » devient floue dans un contexte agentique. Une réponse techniquement correcte peut être pragmatiquement inadéquate : l'agent a peut-être fourni une information exacte mais non pertinente pour le contexte de l'utilisateur, ou adopté un ton inapproprié malgré un contenu factuel irréprochable. Ces dimensions qualitatives échappent aux métriques binaires de succès/échec.

### Les Dimensions de l'Observabilité Agentique

L'observabilité d'un système agentique doit couvrir plusieurs dimensions complémentaires qui forment ensemble une vision holistique du comportement du système.

La **dimension technique** englobe les métriques traditionnelles d'infrastructure : latence des appels API, consommation de tokens, disponibilité des services, taux d'erreur réseau. Ces indicateurs restent essentiels mais ne suffisent plus à caractériser la santé du système.

La **dimension cognitive** capture la qualité du raisonnement : pertinence des outils sélectionnés, cohérence des chaînes de pensée, adéquation entre la requête et la réponse. Cette dimension nécessite des métriques sémantiques qui évaluent le sens plutôt que la forme.

La **dimension comportementale** observe l'alignement de l'agent avec ses objectifs définis et ses contraintes éthiques. Un agent peut produire des réponses techniquement et cognitivement correctes tout en dérivant progressivement de son périmètre de responsabilité ou en adoptant des comportements non prévus.

La **dimension systémique** analyse les interactions entre agents et leurs effets émergents. La performance d'un agent individuel peut masquer des dysfonctionnements au niveau du système global, comme des boucles de rétroaction négatives ou des conflits de ressources.

### Le Défi de la Causalité dans les Systèmes Non-Déterministes

L'établissement de liens de causalité entre un changement et ses effets représente un défi majeur de l'observabilité agentique. Dans un système déterministe, une modification de code suivie d'une dégradation des métriques permet d'identifier rapidement la cause. Dans un système agentique, la variance naturelle des comportements brouille ces corrélations.

Une dégradation observée de la satisfaction utilisateur peut résulter d'un changement de prompt, d'une évolution du modèle de langage sous-jacent, d'une modification de la distribution des requêtes, ou simplement de fluctuations statistiques normales. Distinguer ces causes nécessite des approches statistiques sophistiquées et des fenêtres d'observation suffisamment longues pour établir des conclusions significatives.

> **Note technique**  
> L'analyse causale dans les systèmes agentiques s'appuie généralement sur des tests A/B prolongés et des méthodes d'inférence causale (comme les différences-en-différences) plutôt que sur de simples corrélations temporelles.

### Volume et Coût des Données d'Observabilité

L'instrumentation complète d'un agent génère un volume de données considérable. Chaque interaction peut produire des dizaines de spans de trace, des logs structurés détaillant chaque étape du raisonnement, et des métriques multidimensionnelles. Le stockage et l'analyse de ces données représentent un coût significatif qui doit être équilibré avec la granularité d'observation souhaitée.

Les stratégies d'échantillonnage intelligent deviennent essentielles pour maintenir une observabilité économiquement viable. L'échantillonnage basé sur la tête (head-based sampling) capture un pourcentage fixe des requêtes, tandis que l'échantillonnage basé sur la queue (tail-based sampling) préserve préférentiellement les traces présentant des anomalies ou des latences élevées.

---

## II.11.2 Traçage Distribué (OpenTelemetry)

### OpenTelemetry comme Standard d'Instrumentation

OpenTelemetry (OTel) s'est imposé comme le standard d'instrumentation pour les systèmes distribués. Ce projet, né de la fusion d'OpenTracing et OpenCensus sous l'égide de la Cloud Native Computing Foundation, fournit un ensemble unifié d'API, de SDK et d'outils pour la collecte de traces, métriques et logs.

Pour les systèmes agentiques, OpenTelemetry offre plusieurs avantages décisifs. Son modèle de données flexible permet d'enrichir les spans avec des attributs sémantiques spécifiques aux agents. Sa compatibilité avec de multiples backends (Jaeger, Zipkin, Google Cloud Trace, Datadog) évite le verrouillage technologique. Son approche vendor-agnostic garantit la portabilité de l'instrumentation entre environnements.

### Modélisation des Traces pour Agents Cognitifs

La structure des traces pour un agent cognitif diffère significativement des traces d'applications traditionnelles. Une trace agentique doit capturer non seulement les appels de service, mais aussi les étapes du raisonnement interne, les décisions prises et les alternatives considérées.

```python
# instrumentation/agent_tracing.py
from opentelemetry import trace
from opentelemetry.trace import Status, StatusCode
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from functools import wraps

resource = Resource.create({
    "service.name": "customer-service-agent",
    "service.version": "2.3.1",
    "agent.type": "cognitive",
    "deployment.environment": "production"
})

provider = TracerProvider(resource=resource)
processor = BatchSpanProcessor(OTLPSpanExporter(
    endpoint="otel-collector.monitoring.svc:4317"
))
provider.add_span_processor(processor)
trace.set_tracer_provider(provider)
tracer = trace.get_tracer("agent.cognitive", "1.0.0")

class AgentTracer:
    """Instrumentation OpenTelemetry pour agents cognitifs"""
    
    @staticmethod
    def trace_reasoning_step(step_name: str):
        """Décorateur pour tracer une étape de raisonnement"""
        def decorator(func):
            @wraps(func)
            def wrapper(*args, **kwargs):
                with tracer.start_as_current_span(
                    f"reasoning.{step_name}",
                    kind=trace.SpanKind.INTERNAL
                ) as span:
                    span.set_attribute("agent.reasoning.step", step_name)
                    
                    try:
                        result = func(*args, **kwargs)
                        
                        # Enrichissement avec les métadonnées de raisonnement
                        if hasattr(result, 'confidence'):
                            span.set_attribute("agent.confidence", result.confidence)
                        if hasattr(result, 'alternatives_considered'):
                            span.set_attribute(
                                "agent.alternatives_count",
                                len(result.alternatives_considered)
                            )
                        
                        span.set_status(Status(StatusCode.OK))
                        return result
                        
                    except Exception as e:
                        span.set_status(Status(StatusCode.ERROR, str(e)))
                        span.record_exception(e)
                        raise
            return wrapper
        return decorator
    
    @staticmethod
    def trace_llm_call(model_name: str):
        """Décorateur pour tracer un appel au modèle de langage"""
        def decorator(func):
            @wraps(func)
            def wrapper(*args, **kwargs):
                with tracer.start_as_current_span(
                    "llm.completion",
                    kind=trace.SpanKind.CLIENT
                ) as span:
                    span.set_attribute("llm.model", model_name)
                    span.set_attribute("llm.provider", "anthropic")
                    
                    # Capture des paramètres d'inférence
                    if 'temperature' in kwargs:
                        span.set_attribute("llm.temperature", kwargs['temperature'])
                    if 'max_tokens' in kwargs:
                        span.set_attribute("llm.max_tokens", kwargs['max_tokens'])
                    
                    result = func(*args, **kwargs)
                    
                    # Métriques de consommation
                    if hasattr(result, 'usage'):
                        span.set_attribute("llm.tokens.input", result.usage.input_tokens)
                        span.set_attribute("llm.tokens.output", result.usage.output_tokens)
                        span.set_attribute("llm.tokens.total", 
                            result.usage.input_tokens + result.usage.output_tokens)
                    
                    return result
            return wrapper
        return decorator
    
    @staticmethod
    def trace_tool_execution(tool_name: str):
        """Décorateur pour tracer l'exécution d'un outil"""
        def decorator(func):
            @wraps(func)
            def wrapper(*args, **kwargs):
                with tracer.start_as_current_span(
                    f"tool.{tool_name}",
                    kind=trace.SpanKind.CLIENT
                ) as span:
                    span.set_attribute("tool.name", tool_name)
                    span.set_attribute("tool.parameters", json.dumps(kwargs))
                    
                    result = func(*args, **kwargs)
                    span.set_attribute("tool.success", result.get("success", True))
                    
                    return result
            return wrapper
        return decorator
```

### Propagation du Contexte dans le Backbone Événementiel

L'intégration d'OpenTelemetry avec le backbone Kafka nécessite une attention particulière à la propagation du contexte de trace. Chaque événement publié doit transporter les identifiants de trace et de span parent pour permettre la reconstruction de la chaîne complète d'exécution.

```python
# instrumentation/kafka_propagation.py
from opentelemetry import trace, context
from opentelemetry.propagate import inject, extract
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator
from confluent_kafka import Producer, Consumer
import json

propagator = TraceContextTextMapPropagator()

class TracedKafkaProducer:
    """Producer Kafka avec propagation de contexte OpenTelemetry"""
    
    def __init__(self, config: dict):
        self.producer = Producer(config)
        self.tracer = trace.get_tracer("kafka.producer")
    
    def produce(self, topic: str, key: str, value: dict, headers: dict = None):
        """Produit un message avec contexte de trace injecté"""
        
        with self.tracer.start_as_current_span(
            f"kafka.produce.{topic}",
            kind=trace.SpanKind.PRODUCER
        ) as span:
            span.set_attribute("messaging.system", "kafka")
            span.set_attribute("messaging.destination", topic)
            span.set_attribute("messaging.destination_kind", "topic")
            
            # Injection du contexte dans les headers
            carrier = headers or {}
            propagator.inject(carrier)
            
            # Conversion des headers pour Kafka
            kafka_headers = [(k, v.encode() if isinstance(v, str) else v) 
                           for k, v in carrier.items()]
            
            self.producer.produce(
                topic=topic,
                key=key.encode(),
                value=json.dumps(value).encode(),
                headers=kafka_headers,
                callback=self._delivery_callback(span)
            )
            self.producer.flush()
    
    def _delivery_callback(self, span):
        def callback(err, msg):
            if err:
                span.set_status(Status(StatusCode.ERROR, str(err)))
            else:
                span.set_attribute("messaging.kafka.partition", msg.partition())
                span.set_attribute("messaging.kafka.offset", msg.offset())
        return callback


class TracedKafkaConsumer:
    """Consumer Kafka avec extraction de contexte OpenTelemetry"""
    
    def __init__(self, config: dict):
        self.consumer = Consumer(config)
        self.tracer = trace.get_tracer("kafka.consumer")
    
    def poll_with_context(self, timeout: float = 1.0):
        """Consomme un message et restaure le contexte de trace"""
        
        msg = self.consumer.poll(timeout)
        if msg is None or msg.error():
            return None, None
        
        # Extraction du contexte depuis les headers
        headers_dict = {k: v.decode() if isinstance(v, bytes) else v 
                       for k, v in (msg.headers() or [])}
        ctx = propagator.extract(headers_dict)
        
        # Création d'un span consommateur lié au producteur
        with self.tracer.start_as_current_span(
            f"kafka.consume.{msg.topic()}",
            context=ctx,
            kind=trace.SpanKind.CONSUMER
        ) as span:
            span.set_attribute("messaging.system", "kafka")
            span.set_attribute("messaging.destination", msg.topic())
            span.set_attribute("messaging.kafka.partition", msg.partition())
            span.set_attribute("messaging.kafka.offset", msg.offset())
            
            value = json.loads(msg.value().decode())
            return value, trace.get_current_span().get_span_context()
```

### Instrumentation des Chaînes de Raisonnement

Les chaînes de raisonnement (Chain-of-Thought) des agents constituent un artefact précieux pour le débogage et l'amélioration continue. L'instrumentation doit capturer non seulement le résultat final, mais aussi les étapes intermédiaires du raisonnement.

```python
# instrumentation/reasoning_trace.py
from dataclasses import dataclass, field
from typing import List, Optional, Any
from datetime import datetime

@dataclass
class ReasoningStep:
    """Représentation d'une étape de raisonnement"""
    step_type: str  # "observation", "thought", "action", "reflection"
    content: str
    timestamp: datetime = field(default_factory=datetime.utcnow)
    confidence: Optional[float] = None
    metadata: dict = field(default_factory=dict)

@dataclass  
class ReasoningTrace:
    """Trace complète d'un cycle de raisonnement"""
    trace_id: str
    agent_id: str
    session_id: str
    steps: List[ReasoningStep] = field(default_factory=list)
    final_action: Optional[str] = None
    total_tokens: int = 0
    
    def add_step(self, step: ReasoningStep):
        self.steps.append(step)
        
    def to_otel_events(self, span) -> None:
        """Convertit la trace en événements OpenTelemetry"""
        for i, step in enumerate(self.steps):
            span.add_event(
                name=f"reasoning.{step.step_type}",
                attributes={
                    "step.index": i,
                    "step.content_length": len(step.content),
                    "step.confidence": step.confidence or 0.0,
                    "step.timestamp": step.timestamp.isoformat()
                }
            )


class ReasoningInstrumentor:
    """Instrumenteur pour les cycles de raisonnement ReAct"""
    
    def __init__(self, tracer):
        self.tracer = tracer
        self.current_trace: Optional[ReasoningTrace] = None
    
    def start_reasoning_cycle(self, agent_id: str, session_id: str) -> str:
        """Démarre un nouveau cycle de raisonnement"""
        trace_id = f"reason-{datetime.utcnow().timestamp()}"
        self.current_trace = ReasoningTrace(
            trace_id=trace_id,
            agent_id=agent_id,
            session_id=session_id
        )
        return trace_id
    
    def record_observation(self, content: str, source: str):
        """Enregistre une observation"""
        if self.current_trace:
            self.current_trace.add_step(ReasoningStep(
                step_type="observation",
                content=content,
                metadata={"source": source}
            ))
    
    def record_thought(self, content: str, confidence: float = None):
        """Enregistre une pensée/réflexion"""
        if self.current_trace:
            self.current_trace.add_step(ReasoningStep(
                step_type="thought",
                content=content,
                confidence=confidence
            ))
    
    def record_action(self, action: str, tool: str, parameters: dict):
        """Enregistre une action décidée"""
        if self.current_trace:
            self.current_trace.add_step(ReasoningStep(
                step_type="action",
                content=action,
                metadata={"tool": tool, "parameters": parameters}
            ))
    
    def finalize(self, span) -> ReasoningTrace:
        """Finalise et exporte la trace de raisonnement"""
        if self.current_trace:
            self.current_trace.to_otel_events(span)
            
            # Attributs de synthèse
            span.set_attribute("reasoning.steps_count", len(self.current_trace.steps))
            span.set_attribute("reasoning.has_action", 
                              self.current_trace.final_action is not None)
            
            trace = self.current_trace
            self.current_trace = None
            return trace
        return None
```

> **Bonnes pratiques**  
> Pour les environnements de production à fort volume, considérez l'échantillonnage des traces de raisonnement détaillées. Capturez systématiquement les métadonnées de synthèse (nombre d'étapes, confiance finale) mais limitez l'enregistrement du contenu complet aux cas d'erreur ou aux échantillons aléatoires.

---

## II.11.3 Monitoring de la Performance Cognitive

### Key Agent Indicators (KAIs) : Une Nouvelle Classe de Métriques

Les Key Agent Indicators (KAIs) complètent les Key Performance Indicators (KPIs) traditionnels en capturant les dimensions spécifiquement cognitives de la performance agentique. Ces métriques évaluent la qualité du raisonnement et des décisions plutôt que les seuls aspects techniques.

Les KAIs se répartissent en plusieurs catégories complémentaires qui, ensemble, fournissent une image complète de la santé cognitive d'un agent.

| Catégorie | Indicateur | Description |
|-----------|------------|-------------|
| Efficacité | Task Completion Rate | Taux de tâches accomplies avec succès |
| Efficacité | Tool Selection Accuracy | Précision dans le choix des outils |
| Qualité | Groundedness Score | Ancrage des réponses dans les faits |
| Qualité | Hallucination Rate | Fréquence des informations fabriquées |
| Alignement | Guardrail Compliance | Respect des contraintes définies |
| Alignement | Escalation Appropriateness | Pertinence des escalades humaines |
| Efficience | Reasoning Efficiency | Tokens consommés par tâche réussie |

### Implémentation d'un Collecteur de KAIs

La collecte des KAIs nécessite une infrastructure dédiée capable d'évaluer la qualité sémantique des interactions. Cette évaluation peut s'effectuer en temps réel pour certaines métriques ou de manière asynchrone pour les évaluations plus coûteuses.

```python
# monitoring/kai_collector.py
from dataclasses import dataclass
from typing import Dict, List, Optional
from datetime import datetime
from prometheus_client import Gauge, Counter, Histogram
import asyncio

# Métriques Prometheus pour les KAIs
task_completion_rate = Gauge(
    'agent_task_completion_rate',
    'Taux de complétion des tâches',
    ['agent_id', 'task_type']
)

tool_selection_accuracy = Gauge(
    'agent_tool_selection_accuracy', 
    'Précision de sélection des outils',
    ['agent_id']
)

groundedness_score = Histogram(
    'agent_groundedness_score',
    'Distribution du score d\'ancrage factuel',
    ['agent_id'],
    buckets=[0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0]
)

hallucination_events = Counter(
    'agent_hallucination_total',
    'Nombre total d\'hallucinations détectées',
    ['agent_id', 'severity']
)

guardrail_violations = Counter(
    'agent_guardrail_violations_total',
    'Violations des garde-fous',
    ['agent_id', 'guardrail_type']
)

@dataclass
class InteractionRecord:
    """Enregistrement d'une interaction pour évaluation"""
    interaction_id: str
    agent_id: str
    timestamp: datetime
    user_query: str
    agent_response: str
    tools_used: List[str]
    context_documents: List[str]
    task_completed: bool
    escalated: bool


class KAICollector:
    """Collecteur de Key Agent Indicators"""
    
    def __init__(self, evaluator_client, config: dict):
        self.evaluator = evaluator_client
        self.config = config
        self.pending_evaluations: List[InteractionRecord] = []
    
    async def record_interaction(self, record: InteractionRecord):
        """Enregistre une interaction pour évaluation"""
        
        # Métriques immédiates (sans évaluation LLM)
        self._update_completion_metrics(record)
        self._check_guardrails(record)
        
        # File d'attente pour évaluation asynchrone
        self.pending_evaluations.append(record)
        
        # Déclenchement de l'évaluation si seuil atteint
        if len(self.pending_evaluations) >= self.config['batch_size']:
            await self._process_evaluation_batch()
    
    def _update_completion_metrics(self, record: InteractionRecord):
        """Met à jour les métriques de complétion"""
        
        # Détermination du type de tâche
        task_type = self._classify_task(record.user_query)
        
        # Mise à jour du taux de complétion (moyenne mobile)
        current_rate = task_completion_rate.labels(
            agent_id=record.agent_id,
            task_type=task_type
        )._value.get() or 0.5
        
        new_value = 1.0 if record.task_completed else 0.0
        smoothed_rate = 0.95 * current_rate + 0.05 * new_value
        
        task_completion_rate.labels(
            agent_id=record.agent_id,
            task_type=task_type
        ).set(smoothed_rate)
    
    def _check_guardrails(self, record: InteractionRecord):
        """Vérifie le respect des garde-fous"""
        
        violations = self._detect_violations(record)
        for violation in violations:
            guardrail_violations.labels(
                agent_id=record.agent_id,
                guardrail_type=violation['type']
            ).inc()
    
    async def _process_evaluation_batch(self):
        """Traite un lot d'évaluations sémantiques"""
        
        batch = self.pending_evaluations[:self.config['batch_size']]
        self.pending_evaluations = self.pending_evaluations[self.config['batch_size']:]
        
        for record in batch:
            # Évaluation du groundedness
            grounding_result = await self.evaluator.evaluate_groundedness(
                response=record.agent_response,
                sources=record.context_documents
            )
            
            groundedness_score.labels(agent_id=record.agent_id).observe(
                grounding_result.score
            )
            
            # Détection d'hallucinations
            if grounding_result.hallucinations:
                for h in grounding_result.hallucinations:
                    hallucination_events.labels(
                        agent_id=record.agent_id,
                        severity=h.severity
                    ).inc()
            
            # Évaluation de la sélection d'outils
            if record.tools_used:
                tool_eval = await self.evaluator.evaluate_tool_selection(
                    query=record.user_query,
                    tools_selected=record.tools_used
                )
                
                tool_selection_accuracy.labels(
                    agent_id=record.agent_id
                ).set(tool_eval.accuracy)
    
    def _classify_task(self, query: str) -> str:
        """Classifie le type de tâche à partir de la requête"""
        if any(word in query.lower() for word in ['commande', 'livraison', 'suivi']):
            return 'order_tracking'
        elif any(word in query.lower() for word in ['remboursement', 'retour']):
            return 'refund_request'
        elif any(word in query.lower() for word in ['problème', 'erreur', 'bug']):
            return 'issue_resolution'
        return 'general_inquiry'
    
    def _detect_violations(self, record: InteractionRecord) -> List[dict]:
        """Détecte les violations de garde-fous"""
        violations = []
        
        forbidden_patterns = self.config.get('forbidden_patterns', [])
        for pattern in forbidden_patterns:
            if pattern['regex'].search(record.agent_response):
                violations.append({
                    'type': pattern['name'],
                    'severity': pattern['severity']
                })
        
        return violations
```

### Tableaux de Bord de Performance Cognitive

La visualisation des KAIs nécessite des tableaux de bord adaptés qui présentent les dimensions cognitives de manière intuitive. Les dashboards traditionnels focalisés sur les métriques techniques doivent être enrichis avec des visualisations spécifiques aux agents.

Un tableau de bord de performance cognitive efficace inclut plusieurs vues complémentaires : une vue de santé globale avec les KAIs agrégés, une vue de tendance montrant l'évolution temporelle des métriques, une vue comparative entre agents ou versions, et une vue d'investigation pour l'analyse des cas individuels.

```json
// Configuration Grafana pour dashboard KAI
// grafana/dashboards/agent-cognitive-performance.json
{
  "title": "Performance Cognitive des Agents",
  "panels": [
    {
      "title": "Taux de Complétion par Type de Tâche",
      "type": "gauge",
      "targets": [{
        "expr": "agent_task_completion_rate{agent_id=~\"$agent\"}",
        "legendFormat": "{{task_type}}"
      }],
      "options": {
        "reduceOptions": { "calcs": ["lastNotNull"] },
        "thresholds": {
          "steps": [
            { "color": "red", "value": 0 },
            { "color": "yellow", "value": 0.7 },
            { "color": "green", "value": 0.85 }
          ]
        }
      }
    },
    {
      "title": "Distribution Groundedness (24h)",
      "type": "histogram",
      "targets": [{
        "expr": "histogram_quantile(0.5, rate(agent_groundedness_score_bucket{agent_id=~\"$agent\"}[24h]))",
        "legendFormat": "p50"
      }]
    },
    {
      "title": "Hallucinations par Heure",
      "type": "timeseries",
      "targets": [{
        "expr": "rate(agent_hallucination_total{agent_id=~\"$agent\"}[1h]) * 3600",
        "legendFormat": "{{severity}}"
      }]
    },
    {
      "title": "Violations de Garde-fous",
      "type": "stat",
      "targets": [{
        "expr": "sum(increase(agent_guardrail_violations_total{agent_id=~\"$agent\"}[24h]))",
        "legendFormat": "Total 24h"
      }],
      "options": {
        "colorMode": "background",
        "thresholds": {
          "steps": [
            { "color": "green", "value": 0 },
            { "color": "yellow", "value": 1 },
            { "color": "red", "value": 5 }
          ]
        }
      }
    }
  ]
}
```

---

## II.11.4 Détection de Dérive Comportementale

### Nature et Causes de la Dérive Agentique

La dérive comportementale désigne l'évolution progressive du comportement d'un agent par rapport à ses spécifications initiales. Contrairement aux défaillances franches, la dérive se manifeste par des changements subtils qui peuvent passer inaperçus individuellement mais qui, cumulés, altèrent significativement les performances du système.

Les causes de dérive sont multiples et souvent interdépendantes. Les modifications non annoncées des modèles de langage sous-jacents peuvent modifier subtilement les réponses. L'évolution de la distribution des requêtes expose l'agent à des cas qu'il gère moins bien. L'accumulation de contexte dans la mémoire peut créer des biais non prévus. Enfin, les changements dans les données de référence pour le RAG peuvent introduire des incohérences.

### Mécanismes de Détection de Dérive

La détection de dérive s'appuie sur la comparaison continue des comportements observés avec des référentiels établis. Cette comparaison peut s'effectuer à différents niveaux de granularité.

```python
# monitoring/drift_detection.py
from dataclasses import dataclass
from typing import List, Dict, Optional
from datetime import datetime, timedelta
import numpy as np
from scipy import stats
from enum import Enum

class DriftType(Enum):
    PERFORMANCE = "performance"
    DISTRIBUTION = "distribution"
    SEMANTIC = "semantic"
    BEHAVIORAL = "behavioral"

@dataclass
class DriftAlert:
    """Alerte de dérive détectée"""
    drift_type: DriftType
    severity: str  # "low", "medium", "high", "critical"
    metric_name: str
    baseline_value: float
    current_value: float
    deviation_pct: float
    confidence: float
    detected_at: datetime
    description: str


class DriftDetector:
    """Détecteur de dérive comportementale pour agents"""
    
    def __init__(self, config: dict, metrics_store, alert_service):
        self.config = config
        self.metrics = metrics_store
        self.alerts = alert_service
        
        # Fenêtres de comparaison
        self.baseline_window = timedelta(days=config.get('baseline_days', 7))
        self.current_window = timedelta(hours=config.get('current_hours', 24))
        
        # Seuils de détection
        self.thresholds = config.get('thresholds', {
            'performance_deviation': 0.10,  # 10% de déviation
            'distribution_pvalue': 0.05,    # p-value pour test statistique
            'semantic_drift': 0.15          # 15% de dérive sémantique
        })
    
    async def run_detection_cycle(self, agent_id: str) -> List[DriftAlert]:
        """Exécute un cycle complet de détection de dérive"""
        
        alerts = []
        
        # Détection de dérive de performance
        perf_alerts = await self._detect_performance_drift(agent_id)
        alerts.extend(perf_alerts)
        
        # Détection de dérive de distribution
        dist_alerts = await self._detect_distribution_drift(agent_id)
        alerts.extend(dist_alerts)
        
        # Détection de dérive sémantique
        sem_alerts = await self._detect_semantic_drift(agent_id)
        alerts.extend(sem_alerts)
        
        # Notification des alertes
        for alert in alerts:
            if alert.severity in ['high', 'critical']:
                await self.alerts.send_alert(alert)
        
        return alerts
    
    async def _detect_performance_drift(self, agent_id: str) -> List[DriftAlert]:
        """Détecte les dérives de métriques de performance"""
        
        alerts = []
        metrics_to_check = [
            'task_completion_rate',
            'tool_selection_accuracy',
            'groundedness_score_mean',
            'response_latency_p95'
        ]
        
        for metric_name in metrics_to_check:
            baseline = await self.metrics.get_aggregated(
                agent_id=agent_id,
                metric=metric_name,
                window=self.baseline_window
            )
            
            current = await self.metrics.get_aggregated(
                agent_id=agent_id,
                metric=metric_name,
                window=self.current_window
            )
            
            if baseline.value > 0:
                deviation = (current.value - baseline.value) / baseline.value
                
                if abs(deviation) > self.thresholds['performance_deviation']:
                    severity = self._calculate_severity(deviation)
                    
                    alerts.append(DriftAlert(
                        drift_type=DriftType.PERFORMANCE,
                        severity=severity,
                        metric_name=metric_name,
                        baseline_value=baseline.value,
                        current_value=current.value,
                        deviation_pct=deviation * 100,
                        confidence=baseline.confidence,
                        detected_at=datetime.utcnow(),
                        description=f"Dérive de {deviation*100:.1f}% détectée sur {metric_name}"
                    ))
        
        return alerts
    
    async def _detect_distribution_drift(self, agent_id: str) -> List[DriftAlert]:
        """Détecte les changements dans la distribution des données"""
        
        alerts = []
        
        # Récupération des distributions
        baseline_dist = await self.metrics.get_distribution(
            agent_id=agent_id,
            metric='response_tokens',
            window=self.baseline_window
        )
        
        current_dist = await self.metrics.get_distribution(
            agent_id=agent_id,
            metric='response_tokens',
            window=self.current_window
        )
        
        # Test de Kolmogorov-Smirnov
        ks_statistic, p_value = stats.ks_2samp(
            baseline_dist.values,
            current_dist.values
        )
        
        if p_value < self.thresholds['distribution_pvalue']:
            alerts.append(DriftAlert(
                drift_type=DriftType.DISTRIBUTION,
                severity=self._pvalue_to_severity(p_value),
                metric_name='response_tokens_distribution',
                baseline_value=np.mean(baseline_dist.values),
                current_value=np.mean(current_dist.values),
                deviation_pct=ks_statistic * 100,
                confidence=1 - p_value,
                detected_at=datetime.utcnow(),
                description=f"Distribution significativement différente (KS={ks_statistic:.3f}, p={p_value:.4f})"
            ))
        
        return alerts
    
    async def _detect_semantic_drift(self, agent_id: str) -> List[DriftAlert]:
        """Détecte les dérives dans le contenu sémantique des réponses"""
        
        alerts = []
        
        # Échantillonnage des réponses récentes
        recent_responses = await self.metrics.get_recent_responses(
            agent_id=agent_id,
            limit=100,
            window=self.current_window
        )
        
        # Comparaison avec le corpus de référence
        reference_embeddings = await self.metrics.get_reference_embeddings(agent_id)
        
        # Calcul de la distance moyenne au corpus de référence
        current_embeddings = await self._compute_embeddings(recent_responses)
        
        distances = self._compute_centroid_distances(
            current_embeddings,
            reference_embeddings
        )
        
        mean_distance = np.mean(distances)
        baseline_distance = await self.metrics.get_baseline_distance(agent_id)
        
        drift_ratio = (mean_distance - baseline_distance) / baseline_distance
        
        if drift_ratio > self.thresholds['semantic_drift']:
            alerts.append(DriftAlert(
                drift_type=DriftType.SEMANTIC,
                severity=self._calculate_severity(drift_ratio),
                metric_name='semantic_distance',
                baseline_value=baseline_distance,
                current_value=mean_distance,
                deviation_pct=drift_ratio * 100,
                confidence=0.9,
                detected_at=datetime.utcnow(),
                description=f"Dérive sémantique de {drift_ratio*100:.1f}% par rapport au corpus de référence"
            ))
        
        return alerts
    
    def _calculate_severity(self, deviation: float) -> str:
        """Calcule la sévérité en fonction de la déviation"""
        abs_dev = abs(deviation)
        if abs_dev > 0.30:
            return "critical"
        elif abs_dev > 0.20:
            return "high"
        elif abs_dev > 0.15:
            return "medium"
        return "low"
    
    def _pvalue_to_severity(self, p_value: float) -> str:
        """Convertit une p-value en niveau de sévérité"""
        if p_value < 0.001:
            return "critical"
        elif p_value < 0.01:
            return "high"
        elif p_value < 0.05:
            return "medium"
        return "low"
```

### Stratégies de Remédiation Automatique

La détection de dérive doit s'accompagner de mécanismes de remédiation pour maintenir la qualité du service. Ces mécanismes peuvent être automatiques pour les dérives mineures ou déclencher une escalade humaine pour les situations plus complexes.

```python
# monitoring/drift_remediation.py
from enum import Enum
from typing import Optional
import asyncio

class RemediationAction(Enum):
    NONE = "none"
    INCREASE_MONITORING = "increase_monitoring"
    ROLLBACK_PROMPT = "rollback_prompt"
    ROLLBACK_VERSION = "rollback_version"
    DISABLE_AGENT = "disable_agent"
    ESCALATE_HUMAN = "escalate_human"


class DriftRemediator:
    """Gestionnaire de remédiation automatique des dérives"""
    
    def __init__(self, deployment_manager, notification_service, config: dict):
        self.deployment = deployment_manager
        self.notifications = notification_service
        self.config = config
        
        # Mapping sévérité -> action
        self.action_map = {
            ("performance", "low"): RemediationAction.INCREASE_MONITORING,
            ("performance", "medium"): RemediationAction.INCREASE_MONITORING,
            ("performance", "high"): RemediationAction.ESCALATE_HUMAN,
            ("performance", "critical"): RemediationAction.ROLLBACK_VERSION,
            
            ("semantic", "low"): RemediationAction.INCREASE_MONITORING,
            ("semantic", "medium"): RemediationAction.ESCALATE_HUMAN,
            ("semantic", "high"): RemediationAction.ROLLBACK_PROMPT,
            ("semantic", "critical"): RemediationAction.ROLLBACK_VERSION,
            
            ("behavioral", "low"): RemediationAction.INCREASE_MONITORING,
            ("behavioral", "medium"): RemediationAction.ESCALATE_HUMAN,
            ("behavioral", "high"): RemediationAction.DISABLE_AGENT,
            ("behavioral", "critical"): RemediationAction.DISABLE_AGENT,
        }
    
    async def remediate(self, alert: DriftAlert) -> dict:
        """Exécute la remédiation appropriée pour une alerte"""
        
        action = self._determine_action(alert)
        
        result = {
            "alert_id": alert.detected_at.isoformat(),
            "action": action.value,
            "success": False,
            "details": None
        }
        
        if action == RemediationAction.NONE:
            result["success"] = True
            result["details"] = "Aucune action requise"
            
        elif action == RemediationAction.INCREASE_MONITORING:
            await self._increase_monitoring(alert)
            result["success"] = True
            result["details"] = "Fréquence de monitoring augmentée"
            
        elif action == RemediationAction.ROLLBACK_PROMPT:
            rollback_result = await self._rollback_prompt(alert)
            result["success"] = rollback_result["success"]
            result["details"] = rollback_result
            
        elif action == RemediationAction.ROLLBACK_VERSION:
            rollback_result = await self._rollback_version(alert)
            result["success"] = rollback_result["success"]
            result["details"] = rollback_result
            
        elif action == RemediationAction.DISABLE_AGENT:
            disable_result = await self._disable_agent(alert)
            result["success"] = disable_result["success"]
            result["details"] = disable_result
            
        elif action == RemediationAction.ESCALATE_HUMAN:
            await self._escalate_to_human(alert)
            result["success"] = True
            result["details"] = "Escalade envoyée"
        
        # Journalisation de l'action
        await self._log_remediation(alert, action, result)
        
        return result
    
    def _determine_action(self, alert: DriftAlert) -> RemediationAction:
        """Détermine l'action de remédiation appropriée"""
        key = (alert.drift_type.value, alert.severity)
        return self.action_map.get(key, RemediationAction.ESCALATE_HUMAN)
    
    async def _increase_monitoring(self, alert: DriftAlert):
        """Augmente la fréquence de monitoring"""
        await self.deployment.update_monitoring_config(
            agent_id=alert.metric_name.split('.')[0],
            config={"sampling_rate": 1.0, "evaluation_interval": 60}
        )
    
    async def _rollback_prompt(self, alert: DriftAlert) -> dict:
        """Rollback vers la version précédente du prompt"""
        agent_id = self._extract_agent_id(alert)
        
        previous_prompt = await self.deployment.get_previous_prompt_version(agent_id)
        if previous_prompt:
            await self.deployment.deploy_prompt(
                agent_id=agent_id,
                prompt_version=previous_prompt.version
            )
            return {"success": True, "reverted_to": previous_prompt.version}
        
        return {"success": False, "error": "Pas de version précédente disponible"}
    
    async def _rollback_version(self, alert: DriftAlert) -> dict:
        """Rollback vers la version précédente complète de l'agent"""
        agent_id = self._extract_agent_id(alert)
        
        previous_version = await self.deployment.get_previous_stable_version(agent_id)
        if previous_version:
            await self.deployment.deploy_version(
                agent_id=agent_id,
                version=previous_version.version
            )
            
            await self.notifications.send(
                channel="ops-critical",
                message=f"Rollback automatique de {agent_id} vers {previous_version.version} "
                        f"suite à dérive {alert.drift_type.value}"
            )
            
            return {"success": True, "reverted_to": previous_version.version}
        
        return {"success": False, "error": "Pas de version stable précédente"}
    
    async def _disable_agent(self, alert: DriftAlert) -> dict:
        """Désactive l'agent problématique"""
        agent_id = self._extract_agent_id(alert)
        
        await self.deployment.disable_agent(agent_id)
        
        await self.notifications.send_urgent(
            message=f"CRITIQUE: Agent {agent_id} désactivé automatiquement - "
                    f"Dérive {alert.drift_type.value} {alert.severity}"
        )
        
        return {"success": True, "agent_disabled": agent_id}
    
    async def _escalate_to_human(self, alert: DriftAlert):
        """Escalade vers l'équipe humaine"""
        await self.notifications.create_incident(
            title=f"Dérive détectée: {alert.metric_name}",
            severity=alert.severity,
            description=alert.description,
            context={
                "drift_type": alert.drift_type.value,
                "baseline": alert.baseline_value,
                "current": alert.current_value,
                "deviation": f"{alert.deviation_pct:.1f}%"
            }
        )
```

---

## II.11.5 Cockpit de Supervision

### Architecture du Cockpit Cognitif

Le cockpit de supervision centralise l'ensemble des fonctionnalités d'observabilité dans une interface unifiée destinée aux équipes opérationnelles. Cette interface doit permettre une compréhension rapide de l'état du système tout en offrant les capacités d'investigation approfondie nécessaires au diagnostic des problèmes.

L'architecture du cockpit s'organise autour de plusieurs couches fonctionnelles. La **couche de présentation** fournit les tableaux de bord et visualisations. La **couche d'agrégation** consolide les données provenant de multiples sources (métriques, traces, logs, évaluations). La **couche d'analyse** applique les algorithmes de détection d'anomalies et de dérive. La **couche d'action** expose les interfaces de contrôle pour les remédiations manuelles.

```python
# cockpit/architecture.py
from dataclasses import dataclass
from typing import Dict, List, Optional
from datetime import datetime

@dataclass
class CockpitView:
    """Définition d'une vue du cockpit"""
    view_id: str
    title: str
    layout: str  # "grid", "dashboard", "timeline"
    widgets: List[dict]
    refresh_interval: int  # secondes
    filters: Dict[str, any]


class CockpitController:
    """Contrôleur principal du cockpit de supervision"""
    
    def __init__(
        self,
        metrics_service,
        trace_service,
        alert_service,
        agent_registry,
        action_service
    ):
        self.metrics = metrics_service
        self.traces = trace_service
        self.alerts = alert_service
        self.agents = agent_registry
        self.actions = action_service
        
        # Configuration des vues prédéfinies
        self.views = self._configure_default_views()
    
    def _configure_default_views(self) -> Dict[str, CockpitView]:
        """Configure les vues par défaut du cockpit"""
        
        return {
            "overview": CockpitView(
                view_id="overview",
                title="Vue d'ensemble",
                layout="dashboard",
                widgets=[
                    {"type": "health_matrix", "position": "top-left"},
                    {"type": "active_alerts", "position": "top-right"},
                    {"type": "kpi_trends", "position": "middle"},
                    {"type": "agent_status_grid", "position": "bottom"}
                ],
                refresh_interval=30,
                filters={}
            ),
            
            "agent_detail": CockpitView(
                view_id="agent_detail",
                title="Détail Agent",
                layout="grid",
                widgets=[
                    {"type": "agent_health_gauge", "position": "header"},
                    {"type": "kai_metrics", "position": "left"},
                    {"type": "trace_timeline", "position": "center"},
                    {"type": "conversation_samples", "position": "right"},
                    {"type": "drift_indicators", "position": "footer"}
                ],
                refresh_interval=10,
                filters={"agent_id": None}
            ),
            
            "investigation": CockpitView(
                view_id="investigation",
                title="Investigation",
                layout="timeline",
                widgets=[
                    {"type": "trace_explorer", "position": "main"},
                    {"type": "log_viewer", "position": "side"},
                    {"type": "context_inspector", "position": "bottom"}
                ],
                refresh_interval=0,  # Pas de refresh auto
                filters={"trace_id": None, "time_range": None}
            )
        }
    
    async def get_system_health(self) -> dict:
        """Récupère l'état de santé global du système"""
        
        agents = await self.agents.list_active()
        
        health_data = {
            "timestamp": datetime.utcnow().isoformat(),
            "overall_status": "healthy",
            "agents": {},
            "alerts": {
                "critical": 0,
                "high": 0,
                "medium": 0,
                "low": 0
            }
        }
        
        for agent in agents:
            agent_health = await self._get_agent_health(agent.id)
            health_data["agents"][agent.id] = agent_health
            
            if agent_health["status"] == "critical":
                health_data["overall_status"] = "critical"
            elif agent_health["status"] == "degraded" and health_data["overall_status"] != "critical":
                health_data["overall_status"] = "degraded"
        
        # Agrégation des alertes actives
        active_alerts = await self.alerts.get_active()
        for alert in active_alerts:
            health_data["alerts"][alert.severity] += 1
        
        return health_data
    
    async def _get_agent_health(self, agent_id: str) -> dict:
        """Calcule la santé d'un agent spécifique"""
        
        metrics = await self.metrics.get_latest(agent_id)
        alerts = await self.alerts.get_for_agent(agent_id)
        
        # Calcul du score de santé
        health_score = self._calculate_health_score(metrics, alerts)
        
        return {
            "agent_id": agent_id,
            "status": self._score_to_status(health_score),
            "health_score": health_score,
            "metrics": {
                "task_completion": metrics.get("task_completion_rate", 0),
                "groundedness": metrics.get("groundedness_score", 0),
                "latency_p95": metrics.get("latency_p95_ms", 0)
            },
            "active_alerts": len(alerts),
            "last_activity": metrics.get("last_interaction")
        }
    
    def _calculate_health_score(self, metrics: dict, alerts: list) -> float:
        """Calcule un score de santé composite"""
        
        # Poids des différentes métriques
        weights = {
            "task_completion_rate": 0.30,
            "groundedness_score": 0.25,
            "guardrail_compliance": 0.25,
            "latency_normalized": 0.10,
            "alert_penalty": 0.10
        }
        
        score = 0.0
        
        # Contribution des métriques
        score += metrics.get("task_completion_rate", 0) * weights["task_completion_rate"]
        score += metrics.get("groundedness_score", 0) * weights["groundedness_score"]
        score += metrics.get("guardrail_compliance", 1) * weights["guardrail_compliance"]
        
        # Normalisation de la latence (inversée - basse latence = bon score)
        latency = metrics.get("latency_p95_ms", 1000)
        latency_score = max(0, 1 - (latency / 5000))  # 5s = score 0
        score += latency_score * weights["latency_normalized"]
        
        # Pénalité pour alertes actives
        alert_penalty = sum(
            {"critical": 0.5, "high": 0.3, "medium": 0.1, "low": 0.05}.get(a.severity, 0)
            for a in alerts
        )
        score -= min(alert_penalty, weights["alert_penalty"])
        
        return max(0, min(1, score))
    
    def _score_to_status(self, score: float) -> str:
        """Convertit un score en statut"""
        if score >= 0.9:
            return "healthy"
        elif score >= 0.7:
            return "degraded"
        elif score >= 0.5:
            return "unhealthy"
        return "critical"
```

### Interface de Contrôle et Actions

Le cockpit doit fournir des capacités d'action directe pour permettre aux opérateurs d'intervenir rapidement sur les systèmes. Ces actions incluent la modification des paramètres de déploiement, le déclenchement de rollbacks manuels et l'ajustement des seuils de détection.

```python
# cockpit/action_controller.py
from enum import Enum
from typing import Dict, Any, Optional
from datetime import datetime, timedelta
import asyncio

class ActionType(Enum):
    ADJUST_TRAFFIC = "adjust_traffic"
    ROLLBACK = "rollback"
    DISABLE_FEATURE = "disable_feature"
    FORCE_ESCALATION = "force_escalation"
    UPDATE_THRESHOLD = "update_threshold"
    TRIGGER_EVALUATION = "trigger_evaluation"


class ActionController:
    """Contrôleur des actions opérationnelles"""
    
    def __init__(self, deployment_service, config_service, audit_log):
        self.deployment = deployment_service
        self.config = config_service
        self.audit = audit_log
    
    async def execute_action(
        self,
        action_type: ActionType,
        target_agent: str,
        parameters: Dict[str, Any],
        operator_id: str,
        reason: str
    ) -> dict:
        """Exécute une action opérationnelle avec audit complet"""
        
        # Validation des permissions
        if not await self._validate_permissions(operator_id, action_type):
            return {"success": False, "error": "Permissions insuffisantes"}
        
        # Journalisation avant action
        action_id = await self.audit.log_action_start(
            action_type=action_type.value,
            target=target_agent,
            parameters=parameters,
            operator=operator_id,
            reason=reason
        )
        
        try:
            result = await self._dispatch_action(
                action_type, target_agent, parameters
            )
            
            # Journalisation du succès
            await self.audit.log_action_complete(action_id, result)
            
            return {
                "success": True,
                "action_id": action_id,
                "result": result
            }
            
        except Exception as e:
            # Journalisation de l'échec
            await self.audit.log_action_failed(action_id, str(e))
            
            return {
                "success": False,
                "action_id": action_id,
                "error": str(e)
            }
    
    async def _dispatch_action(
        self,
        action_type: ActionType,
        target_agent: str,
        parameters: Dict[str, Any]
    ) -> dict:
        """Dispatch l'action vers le handler approprié"""
        
        handlers = {
            ActionType.ADJUST_TRAFFIC: self._handle_traffic_adjustment,
            ActionType.ROLLBACK: self._handle_rollback,
            ActionType.DISABLE_FEATURE: self._handle_feature_toggle,
            ActionType.FORCE_ESCALATION: self._handle_force_escalation,
            ActionType.UPDATE_THRESHOLD: self._handle_threshold_update,
            ActionType.TRIGGER_EVALUATION: self._handle_evaluation_trigger
        }
        
        handler = handlers.get(action_type)
        if handler:
            return await handler(target_agent, parameters)
        
        raise ValueError(f"Action non supportée: {action_type}")
    
    async def _handle_traffic_adjustment(
        self,
        agent_id: str,
        params: dict
    ) -> dict:
        """Ajuste le routage du trafic vers un agent"""
        
        new_percentage = params.get("traffic_percentage", 100)
        
        await self.deployment.update_traffic_split(
            agent_id=agent_id,
            percentage=new_percentage
        )
        
        return {
            "agent_id": agent_id,
            "new_traffic_percentage": new_percentage,
            "effective_at": datetime.utcnow().isoformat()
        }
    
    async def _handle_rollback(
        self,
        agent_id: str,
        params: dict
    ) -> dict:
        """Exécute un rollback vers une version spécifique"""
        
        target_version = params.get("target_version")
        if not target_version:
            # Rollback vers la dernière version stable
            target_version = await self.deployment.get_last_stable_version(agent_id)
        
        await self.deployment.deploy_version(
            agent_id=agent_id,
            version=target_version,
            strategy="immediate"
        )
        
        return {
            "agent_id": agent_id,
            "rolled_back_to": target_version,
            "completed_at": datetime.utcnow().isoformat()
        }
    
    async def _handle_feature_toggle(
        self,
        agent_id: str,
        params: dict
    ) -> dict:
        """Active ou désactive une fonctionnalité"""
        
        feature_name = params["feature_name"]
        enabled = params.get("enabled", False)
        
        await self.config.update_feature_flag(
            agent_id=agent_id,
            feature=feature_name,
            enabled=enabled
        )
        
        return {
            "agent_id": agent_id,
            "feature": feature_name,
            "enabled": enabled
        }
    
    async def _handle_force_escalation(
        self,
        agent_id: str,
        params: dict
    ) -> dict:
        """Force l'escalade de toutes les conversations vers un humain"""
        
        duration_minutes = params.get("duration_minutes", 30)
        
        await self.config.set_escalation_mode(
            agent_id=agent_id,
            mode="all",
            duration_minutes=duration_minutes
        )
        
        return {
            "agent_id": agent_id,
            "escalation_mode": "all",
            "expires_at": (
                datetime.utcnow() + timedelta(minutes=duration_minutes)
            ).isoformat()
        }
```

> **Attention**  
> Toutes les actions opérationnelles doivent être journalisées avec l'identité de l'opérateur, la raison de l'action et l'horodatage. Cet audit est essentiel pour la conformité réglementaire et l'analyse post-incident.

---

## Conclusion

L'observabilité comportementale représente une extension fondamentale des pratiques de monitoring pour répondre aux spécificités des systèmes agentiques. Ce chapitre a établi les bases d'une approche holistique combinant les métriques techniques traditionnelles avec des indicateurs cognitifs spécifiques aux agents.

L'instrumentation via OpenTelemetry fournit la visibilité nécessaire sur les chaînes de raisonnement et les interactions distribuées. Les Key Agent Indicators complètent les KPIs traditionnels en capturant les dimensions de qualité, d'efficacité et d'alignement des comportements agentiques. Les mécanismes de détection de dérive permettent d'identifier précocement les évolutions non souhaitées avant qu'elles n'impactent significativement le service.

Le cockpit de supervision centralise ces capacités dans une interface opérationnelle qui permet aux équipes de maintenir une compréhension claire de l'état de leurs systèmes et d'intervenir rapidement lorsque nécessaire. L'automatisation des remédiations pour les cas simples libère l'attention humaine pour les situations complexes nécessitant un jugement expert.

Le chapitre suivant approfondira la dimension des tests et de l'évaluation, complétant ainsi le triptyque CI/CD, observabilité et validation qui constitue le socle de l'excellence opérationnelle AgentOps.

---

## II.11.6 Résumé

**Rupture paradigmatique.** L'observabilité agentique rompt avec les approches traditionnelles : le non-déterminisme des réponses invalide les comparaisons avec un comportement de référence fixe, et la notion d'erreur devient floue car une réponse techniquement correcte peut être pragmatiquement inadéquate.

**Quatre dimensions d'observabilité.** La supervision des agents couvre les dimensions technique (latence, tokens, disponibilité), cognitive (qualité du raisonnement, pertinence des outils), comportementale (alignement avec les objectifs et contraintes éthiques) et systémique (interactions entre agents et effets émergents).

**OpenTelemetry comme standard.** L'instrumentation via OTel permet d'enrichir les spans avec des attributs sémantiques spécifiques aux agents et assure la compatibilité avec de multiples backends. La propagation du contexte de trace dans les headers Kafka maintient la traçabilité à travers le backbone événementiel.

**Instrumentation des chaînes de raisonnement.** Les traces agentiques capturent non seulement les appels de service mais aussi les étapes de raisonnement (observation, pensée, action, réflexion), les décisions prises et les alternatives considérées, avec des événements OTel pour chaque étape.

**Key Agent Indicators (KAIs).** Nouvelle classe de métriques évaluant la performance cognitive : taux de complétion de tâche, précision de sélection d'outils, score d'ancrage factuel (groundedness), taux d'hallucination, conformité aux garde-fous et efficience de raisonnement (tokens par tâche réussie).

**Collecte asynchrone des KAIs.** Les métriques immédiates (complétion, violations) sont collectées en temps réel, tandis que les évaluations sémantiques coûteuses (groundedness, hallucinations) sont traitées par lots de manière asynchrone via un évaluateur LLM.

**Détection de dérive comportementale.** La dérive résulte de modifications des modèles sous-jacents, d'évolutions de la distribution des requêtes, d'accumulation de biais en mémoire ou de changements dans les données de référence RAG. La détection s'appuie sur des tests statistiques (Kolmogorov-Smirnov) et des comparaisons de distances sémantiques.

**Remédiation automatique graduée.** Les actions de remédiation suivent une escalade : augmentation du monitoring pour les dérives légères, rollback de prompt ou de version pour les dérives modérées, désactivation de l'agent et escalade humaine pour les situations critiques.

**Architecture du cockpit.** Le cockpit de supervision s'organise en couches : présentation (tableaux de bord), agrégation (consolidation multi-sources), analyse (détection d'anomalies) et action (interfaces de contrôle). Trois vues principales : vue d'ensemble, détail agent et investigation.

**Score de santé composite.** La santé d'un agent se calcule via une pondération des KAIs (complétion 30%, ancrage 25%, conformité garde-fous 25%, latence 10%) avec pénalité pour alertes actives, produisant un statut global (healthy, degraded, unhealthy, critical).

**Actions opérationnelles auditées.** Le cockpit expose des actions directes (ajustement du trafic, rollback, toggle de fonctionnalités, escalade forcée) avec validation des permissions et journalisation complète de l'opérateur, de la raison et de l'horodatage pour conformité et analyse post-incident.

**Échantillonnage intelligent.** Face au volume de données d'observabilité, l'échantillonnage tail-based préserve préférentiellement les traces présentant des anomalies ou latences élevées, permettant une observabilité économiquement viable sans perte des cas critiques.

---

*Chapitre suivant : Chapitre II.12 — Tests, Évaluation et Simulation des Systèmes Multi-Agents*
