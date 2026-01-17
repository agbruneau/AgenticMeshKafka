# Chapitre II.9 — Patrons Architecturaux Avancés pour l'AEM

---

## Introduction

L'Agentic Event Mesh (AEM) représente l'infrastructure fondamentale sur laquelle s'appuient les systèmes multi-agents modernes. Cependant, la complexité intrinsèque des architectures distribuées et la nature non déterministe des agents cognitifs exigent des patrons architecturaux sophistiqués pour garantir la cohérence, la résilience et la traçabilité des opérations. Ce chapitre explore en profondeur les patrons avancés qui permettent de construire des systèmes agentiques robustes et maintenables.

Les systèmes agentiques présentent des défis uniques que les architectures traditionnelles peinent à résoudre. Un agent peut initier une chaîne de traitements impliquant plusieurs services, bases de données et autres agents, le tout de manière asynchrone et potentiellement non déterministe. Comment garantir la cohérence transactionnelle dans un tel contexte ? Comment assurer que le système peut récupérer d'une défaillance partielle sans perdre l'état ni corrompre les données ?

Les patrons présentés dans ce chapitre — Saga Chorégraphiée, CQRS, Event Sourcing et Outbox Transactionnel — constituent les briques fondamentales pour répondre à ces questions. Ils ne sont pas mutuellement exclusifs mais se combinent naturellement pour former une architecture cohérente où chaque patron adresse un aspect spécifique de la problématique globale.

Le patron Saga Chorégraphiée orchestre les transactions distribuées à travers une séquence d'événements, permettant aux agents de coordonner leurs actions sans couplage fort. CQRS sépare les flux de lecture et d'écriture, optimisant les performances tout en permettant des vues spécialisées pour les différents consommateurs. Event Sourcing capture l'intégralité de l'historique des changements d'état, offrant une traçabilité complète et la possibilité de reconstituer l'état à n'importe quel moment. L'Outbox Transactionnel garantit la cohérence entre les modifications de base de données et la publication d'événements, éliminant les risques de perte ou de duplication.

Ce chapitre détaille chacun de ces patrons avec des implémentations concrètes utilisant l'écosystème Confluent et Google Cloud, en montrant comment ils s'intègrent naturellement dans l'architecture AEM présentée dans les chapitres précédents.

---

## II.9.1 Patron Saga Chorégraphiée

### Fondements et Motivation

Dans les systèmes distribués traditionnels, les transactions ACID garantissent l'atomicité des opérations au sein d'une base de données unique. Cependant, les architectures de microservices et les systèmes agentiques impliquent souvent des opérations qui traversent plusieurs services et sources de données. Le patron de Saga, introduit par Hector Garcia-Molina et Kenneth Salem en 1987, répond à ce défi en décomposant une transaction longue en une séquence de transactions locales, chacune publiée via des événements.

La variante chorégraphiée de la Saga se distingue de l'approche orchestrée par l'absence d'un coordinateur central. Chaque participant écoute les événements pertinents et réagit en exécutant sa transaction locale puis en publiant l'événement suivant. Cette approche distribue la logique de coordination et élimine un point unique de défaillance, au prix d'une complexité accrue dans la compréhension du flux global.

Dans le contexte agentique, la Saga Chorégraphiée prend une dimension particulière. Un agent peut initier un processus métier complexe — comme le traitement d'une demande de prêt — qui implique des vérifications de crédit, des validations de documents, des approbations hiérarchiques et des notifications. Chaque étape peut être gérée par un agent spécialisé ou un service dédié, communiquant exclusivement via le backbone événementiel.

### Architecture de la Saga Chorégraphiée

L'architecture d'une Saga Chorégraphiée repose sur plusieurs éléments fondamentaux. Les événements de domaine capturent les faits métier significatifs. Les participants réagissent à ces événements et produisent de nouveaux événements. Les événements de compensation permettent d'annuler les effets d'une transaction locale en cas d'échec ultérieur.

| Type d'événement | Rôle | Exemple |
|------------------|------|---------|
| Commande | Initie une action | ProcessLoanRequest |
| Succès | Confirme l'exécution | CreditCheckPassed |
| Échec | Signale un problème | CreditCheckFailed |
| Compensation | Annule une action | ReservationCancelled |

```python
# saga/events.py
from dataclasses import dataclass, field
from typing import Optional, List, Dict, Any
from datetime import datetime
from enum import Enum
import uuid

class SagaStatus(Enum):
    STARTED = "started"
    IN_PROGRESS = "in_progress"
    COMPLETED = "completed"
    COMPENSATING = "compensating"
    COMPENSATED = "compensated"
    FAILED = "failed"

@dataclass
class SagaEvent:
    """Événement de base pour une Saga"""
    event_id: str = field(default_factory=lambda: str(uuid.uuid4()))
    saga_id: str = ""
    correlation_id: str = ""
    timestamp: datetime = field(default_factory=datetime.utcnow)
    event_type: str = ""
    payload: Dict[str, Any] = field(default_factory=dict)
    metadata: Dict[str, Any] = field(default_factory=dict)
    
    def to_kafka_headers(self) -> List[tuple]:
        """Convertit les métadonnées en headers Kafka"""
        return [
            ("saga_id", self.saga_id.encode()),
            ("correlation_id", self.correlation_id.encode()),
            ("event_type", self.event_type.encode()),
            ("timestamp", self.timestamp.isoformat().encode())
        ]

@dataclass
class LoanApplicationStarted(SagaEvent):
    """Événement de démarrage d'une demande de prêt"""
    event_type: str = "loan.application.started"
    applicant_id: str = ""
    loan_amount: float = 0.0
    loan_purpose: str = ""

@dataclass
class CreditCheckRequested(SagaEvent):
    """Demande de vérification de crédit"""
    event_type: str = "credit.check.requested"
    applicant_id: str = ""
    requested_amount: float = 0.0

@dataclass
class CreditCheckCompleted(SagaEvent):
    """Résultat de la vérification de crédit"""
    event_type: str = "credit.check.completed"
    applicant_id: str = ""
    credit_score: int = 0
    approved: bool = False
    max_approved_amount: Optional[float] = None
    rejection_reason: Optional[str] = None

@dataclass
class DocumentVerificationRequested(SagaEvent):
    """Demande de vérification des documents"""
    event_type: str = "document.verification.requested"
    applicant_id: str = ""
    document_ids: List[str] = field(default_factory=list)

@dataclass
class DocumentVerificationCompleted(SagaEvent):
    """Résultat de la vérification des documents"""
    event_type: str = "document.verification.completed"
    applicant_id: str = ""
    verified: bool = False
    issues: List[str] = field(default_factory=list)

@dataclass
class LoanApproved(SagaEvent):
    """Prêt approuvé"""
    event_type: str = "loan.approved"
    applicant_id: str = ""
    approved_amount: float = 0.0
    interest_rate: float = 0.0
    term_months: int = 0

@dataclass
class LoanRejected(SagaEvent):
    """Prêt rejeté"""
    event_type: str = "loan.rejected"
    applicant_id: str = ""
    rejection_reasons: List[str] = field(default_factory=list)

@dataclass
class CompensationEvent(SagaEvent):
    """Événement de compensation générique"""
    event_type: str = "compensation"
    original_event_type: str = ""
    compensation_reason: str = ""
```

La définition des événements constitue la fondation de la Saga. Chaque événement capture un fait métier significatif avec toutes les informations nécessaires pour que les participants puissent réagir de manière autonome. Le saga_id permet de corréler tous les événements appartenant à une même transaction distribuée, tandis que le correlation_id facilite le traçage à travers les systèmes.

### Implémentation des Participants

Chaque participant de la Saga implémente une logique de réaction aux événements. Le participant exécute sa transaction locale, publie le résultat, et maintient suffisamment d'état pour pouvoir compenser si nécessaire.

```python
# saga/participants/credit_check.py
from typing import Optional
from confluent_kafka import Consumer, Producer
import json

class CreditCheckParticipant:
    """Participant responsable de la vérification de crédit"""
    
    def __init__(self, consumer: Consumer, producer: Producer, credit_service):
        self.consumer = consumer
        self.producer = producer
        self.credit_service = credit_service
        self.pending_checks: dict = {}  # Pour la compensation
        
        # Souscription au topic de requêtes
        self.consumer.subscribe(['credit.check.requests'])
    
    async def process_events(self):
        """Boucle principale de traitement des événements"""
        while True:
            msg = self.consumer.poll(1.0)
            if msg is None:
                continue
            if msg.error():
                self._handle_error(msg.error())
                continue
            
            event = json.loads(msg.value().decode())
            event_type = event.get('event_type')
            
            if event_type == 'credit.check.requested':
                await self._handle_credit_check_request(event)
            elif event_type == 'compensation.credit.check':
                await self._handle_compensation(event)
    
    async def _handle_credit_check_request(self, event: dict):
        """Traite une demande de vérification de crédit"""
        saga_id = event['saga_id']
        applicant_id = event['applicant_id']
        requested_amount = event['requested_amount']
        
        try:
            # Exécution de la vérification
            result = await self.credit_service.check_credit(
                applicant_id=applicant_id,
                amount=requested_amount
            )
            
            # Stockage pour compensation potentielle
            self.pending_checks[saga_id] = {
                'applicant_id': applicant_id,
                'check_id': result.check_id,
                'timestamp': datetime.utcnow().isoformat()
            }
            
            # Publication du résultat
            response_event = {
                'event_id': str(uuid.uuid4()),
                'saga_id': saga_id,
                'correlation_id': event.get('correlation_id'),
                'event_type': 'credit.check.completed',
                'applicant_id': applicant_id,
                'credit_score': result.score,
                'approved': result.approved,
                'max_approved_amount': result.max_amount if result.approved else None,
                'rejection_reason': result.reason if not result.approved else None,
                'timestamp': datetime.utcnow().isoformat()
            }
            
            self._publish_event('credit.check.results', response_event)
            
        except Exception as e:
            # Publication d'un événement d'échec
            error_event = {
                'event_id': str(uuid.uuid4()),
                'saga_id': saga_id,
                'event_type': 'credit.check.failed',
                'error': str(e),
                'timestamp': datetime.utcnow().isoformat()
            }
            self._publish_event('credit.check.results', error_event)
    
    async def _handle_compensation(self, event: dict):
        """Compense une vérification de crédit précédente"""
        saga_id = event['saga_id']
        
        if saga_id in self.pending_checks:
            check_info = self.pending_checks[saga_id]
            
            # Annulation de la vérification (marquer comme annulée)
            await self.credit_service.cancel_check(check_info['check_id'])
            
            # Nettoyage
            del self.pending_checks[saga_id]
            
            # Confirmation de la compensation
            self._publish_event('compensation.results', {
                'saga_id': saga_id,
                'event_type': 'credit.check.compensated',
                'timestamp': datetime.utcnow().isoformat()
            })
    
    def _publish_event(self, topic: str, event: dict):
        """Publie un événement sur Kafka"""
        self.producer.produce(
            topic=topic,
            key=event['saga_id'].encode(),
            value=json.dumps(event).encode(),
            headers=[
                ('saga_id', event['saga_id'].encode()),
                ('event_type', event['event_type'].encode())
            ]
        )
        self.producer.flush()
```

### Coordinateur de Saga Chorégraphiée

Bien que la Saga Chorégraphiée n'ait pas de coordinateur central au sens strict, il est utile de maintenir un composant qui observe l'état global de la saga et peut déclencher les compensations si nécessaire. Ce coordinateur n'intervient pas dans le flux normal mais surveille les timeouts et les situations anormales.

```python
# saga/coordinator.py
from dataclasses import dataclass
from typing import Dict, List, Optional
from datetime import datetime, timedelta
from enum import Enum
import asyncio

@dataclass
class SagaState:
    """État d'une Saga en cours"""
    saga_id: str
    status: SagaStatus
    started_at: datetime
    current_step: str
    completed_steps: List[str]
    failed_step: Optional[str] = None
    compensation_started: bool = False
    events: List[dict] = None
    
    def __post_init__(self):
        if self.events is None:
            self.events = []

class SagaCoordinator:
    """Coordinateur observateur pour les Sagas Chorégraphiées"""
    
    def __init__(self, consumer, producer, state_store, config: dict):
        self.consumer = consumer
        self.producer = producer
        self.state_store = state_store
        self.config = config
        
        # Configuration des timeouts
        self.step_timeout = timedelta(seconds=config.get('step_timeout', 300))
        self.saga_timeout = timedelta(seconds=config.get('saga_timeout', 3600))
        
        # Définition du flux de la saga
        self.saga_flow = config.get('saga_flow', [
            'credit.check.completed',
            'document.verification.completed',
            'loan.decision.made'
        ])
        
        # Mapping des compensations
        self.compensation_map = {
            'credit.check.completed': 'compensation.credit.check',
            'document.verification.completed': 'compensation.document.verification',
            'loan.decision.made': 'compensation.loan.decision'
        }
    
    async def monitor_sagas(self):
        """Surveillance continue des sagas en cours"""
        # Souscription à tous les topics de résultats
        self.consumer.subscribe([
            'loan.application.events',
            'credit.check.results',
            'document.verification.results',
            'loan.decision.results'
        ])
        
        while True:
            msg = self.consumer.poll(1.0)
            if msg is not None and not msg.error():
                await self._process_event(json.loads(msg.value().decode()))
            
            # Vérification périodique des timeouts
            await self._check_timeouts()
    
    async def _process_event(self, event: dict):
        """Traite un événement et met à jour l'état de la saga"""
        saga_id = event.get('saga_id')
        event_type = event.get('event_type')
        
        if not saga_id:
            return
        
        # Récupération ou création de l'état
        state = await self.state_store.get(saga_id)
        if state is None:
            if event_type == 'loan.application.started':
                state = SagaState(
                    saga_id=saga_id,
                    status=SagaStatus.STARTED,
                    started_at=datetime.utcnow(),
                    current_step='started',
                    completed_steps=[]
                )
            else:
                return  # Événement orphelin
        
        # Mise à jour de l'état
        state.events.append(event)
        
        if event_type.endswith('.completed') and 'error' not in event:
            state.completed_steps.append(event_type)
            state.current_step = event_type
            
            # Vérification de la complétion
            if self._is_saga_complete(state):
                state.status = SagaStatus.COMPLETED
        
        elif event_type.endswith('.failed') or 'error' in event:
            state.status = SagaStatus.COMPENSATING
            state.failed_step = event_type
            await self._initiate_compensation(state)
        
        await self.state_store.save(state)
    
    def _is_saga_complete(self, state: SagaState) -> bool:
        """Vérifie si la saga est terminée avec succès"""
        return all(step in state.completed_steps for step in self.saga_flow)
    
    async def _initiate_compensation(self, state: SagaState):
        """Déclenche la compensation pour les étapes complétées"""
        if state.compensation_started:
            return
        
        state.compensation_started = True
        
        # Compensation en ordre inverse
        for step in reversed(state.completed_steps):
            compensation_event_type = self.compensation_map.get(step)
            if compensation_event_type:
                compensation_event = {
                    'event_id': str(uuid.uuid4()),
                    'saga_id': state.saga_id,
                    'event_type': compensation_event_type,
                    'original_step': step,
                    'reason': f'Compensation due to failure at {state.failed_step}',
                    'timestamp': datetime.utcnow().isoformat()
                }
                
                self._publish_compensation(compensation_event)
    
    async def _check_timeouts(self):
        """Vérifie les sagas en timeout"""
        active_sagas = await self.state_store.get_active()
        now = datetime.utcnow()
        
        for saga in active_sagas:
            # Timeout global de la saga
            if now - saga.started_at > self.saga_timeout:
                saga.status = SagaStatus.FAILED
                saga.failed_step = 'timeout'
                await self._initiate_compensation(saga)
                await self.state_store.save(saga)
    
    def _publish_compensation(self, event: dict):
        """Publie un événement de compensation"""
        topic = f"compensation.{event['original_step'].split('.')[0]}"
        self.producer.produce(
            topic=topic,
            key=event['saga_id'].encode(),
            value=json.dumps(event).encode()
        )
        self.producer.flush()
```

> **Bonnes pratiques**  
> Conservez un historique complet des événements de chaque saga pour faciliter le débogage et l'audit. Implémentez des mécanismes de retry avec backoff exponentiel pour les compensations qui échouent. Utilisez des identifiants idempotents pour éviter les doubles traitements.

---

## II.9.2 CQRS dans un Contexte Agentique

### Principes Fondamentaux de CQRS

Command Query Responsibility Segregation (CQRS) est un patron architectural qui sépare les opérations de lecture (queries) des opérations d'écriture (commands) en utilisant des modèles distincts. Cette séparation permet d'optimiser chaque côté indépendamment : le modèle de commande peut être normalisé pour garantir la cohérence, tandis que le modèle de lecture peut être dénormalisé pour maximiser les performances des requêtes.

Dans le contexte des systèmes agentiques, CQRS prend une dimension particulière. Les agents cognitifs consomment souvent des informations agrégées provenant de multiples sources pour prendre leurs décisions, tandis que leurs actions génèrent des événements qui modifient l'état du système. La séparation lecture/écriture permet de construire des vues optimisées pour chaque agent sans compromettre l'intégrité du modèle d'écriture.

Le modèle de commande, ou write model, capture l'état autoritatif du système. C'est la source de vérité qui applique les règles métier et garantit la cohérence des données. Le modèle de lecture, ou read model, est une projection optimisée de cet état, mise à jour de manière asynchrone via les événements de domaine. Cette projection peut prendre de multiples formes selon les besoins des consommateurs.

| Aspect | Write Model | Read Model |
|--------|-------------|------------|
| Objectif | Cohérence, règles métier | Performance de lecture |
| Structure | Normalisée (3NF) | Dénormalisée |
| Mise à jour | Synchrone, transactionnelle | Asynchrone, éventuelle |
| Stockage typique | SGBD relationnel | NoSQL, cache, search |
| Scalabilité | Verticale principalement | Horizontale |

### Architecture CQRS pour Systèmes Agentiques

L'implémentation de CQRS dans un système agentique s'articule autour du backbone événementiel Kafka. Les commandes sont traitées par des handlers dédiés qui appliquent les règles métier et publient des événements de domaine. Ces événements sont consommés par des projecteurs qui maintiennent les différentes vues de lecture. Les agents interrogent ces vues pour obtenir l'information nécessaire à leurs décisions.

```python
# cqrs/commands.py
from dataclasses import dataclass, field
from typing import Any, Dict, Optional
from datetime import datetime
from abc import ABC, abstractmethod
import uuid

@dataclass
class Command(ABC):
    """Classe de base pour les commandes"""
    command_id: str = field(default_factory=lambda: str(uuid.uuid4()))
    correlation_id: str = ""
    timestamp: datetime = field(default_factory=datetime.utcnow)
    metadata: Dict[str, Any] = field(default_factory=dict)
    
    @abstractmethod
    def validate(self) -> bool:
        """Valide la commande"""
        pass

@dataclass
class CreateCustomerCommand(Command):
    """Commande de création d'un client"""
    customer_id: str = ""
    name: str = ""
    email: str = ""
    phone: Optional[str] = None
    address: Optional[Dict[str, str]] = None
    
    def validate(self) -> bool:
        if not self.name or len(self.name) < 2:
            raise ValueError("Le nom doit contenir au moins 2 caractères")
        if not self.email or '@' not in self.email:
            raise ValueError("Email invalide")
        return True

@dataclass
class UpdateCustomerCommand(Command):
    """Commande de mise à jour d'un client"""
    customer_id: str = ""
    updates: Dict[str, Any] = field(default_factory=dict)
    
    def validate(self) -> bool:
        if not self.customer_id:
            raise ValueError("customer_id requis")
        if 'email' in self.updates and '@' not in self.updates['email']:
            raise ValueError("Email invalide")
        return True

@dataclass
class ProcessInteractionCommand(Command):
    """Commande de traitement d'une interaction agent"""
    customer_id: str = ""
    agent_id: str = ""
    interaction_type: str = ""
    content: Dict[str, Any] = field(default_factory=dict)
    resolution: Optional[str] = None
    sentiment_score: Optional[float] = None
    
    def validate(self) -> bool:
        if not all([self.customer_id, self.agent_id, self.interaction_type]):
            raise ValueError("customer_id, agent_id et interaction_type requis")
        return True
```

```python
# cqrs/command_handlers.py
from typing import List
from abc import ABC, abstractmethod

class CommandHandler(ABC):
    """Interface pour les handlers de commandes"""
    
    @abstractmethod
    async def handle(self, command: Command) -> List[DomainEvent]:
        """Traite une commande et retourne les événements générés"""
        pass

class CustomerCommandHandler(CommandHandler):
    """Handler pour les commandes relatives aux clients"""
    
    def __init__(self, repository, event_publisher, validator):
        self.repository = repository
        self.publisher = event_publisher
        self.validator = validator
    
    async def handle(self, command: Command) -> List[DomainEvent]:
        """Dispatch vers le handler approprié"""
        handlers = {
            CreateCustomerCommand: self._handle_create,
            UpdateCustomerCommand: self._handle_update,
            ProcessInteractionCommand: self._handle_interaction
        }
        
        handler = handlers.get(type(command))
        if not handler:
            raise ValueError(f"Handler non trouvé pour {type(command)}")
        
        # Validation
        command.validate()
        
        # Exécution et publication des événements
        events = await handler(command)
        for event in events:
            await self.publisher.publish(event)
        
        return events
    
    async def _handle_create(self, cmd: CreateCustomerCommand) -> List[DomainEvent]:
        """Traite la création d'un client"""
        
        # Vérification de l'unicité
        existing = await self.repository.find_by_email(cmd.email)
        if existing:
            raise ValueError(f"Un client existe déjà avec l'email {cmd.email}")
        
        # Création de l'agrégat
        customer = Customer(
            customer_id=cmd.customer_id or str(uuid.uuid4()),
            name=cmd.name,
            email=cmd.email,
            phone=cmd.phone,
            address=cmd.address
        )
        
        # Persistance
        await self.repository.save(customer)
        
        # Génération de l'événement
        return [CustomerCreatedEvent(
            event_id=str(uuid.uuid4()),
            aggregate_id=customer.customer_id,
            correlation_id=cmd.correlation_id,
            customer_id=customer.customer_id,
            name=customer.name,
            email=customer.email,
            timestamp=datetime.utcnow()
        )]
    
    async def _handle_update(self, cmd: UpdateCustomerCommand) -> List[DomainEvent]:
        """Traite la mise à jour d'un client"""
        
        customer = await self.repository.get(cmd.customer_id)
        if not customer:
            raise ValueError(f"Client {cmd.customer_id} non trouvé")
        
        # Application des modifications
        old_values = {}
        for key, value in cmd.updates.items():
            if hasattr(customer, key):
                old_values[key] = getattr(customer, key)
                setattr(customer, key, value)
        
        # Persistance
        await self.repository.save(customer)
        
        # Génération de l'événement
        return [CustomerUpdatedEvent(
            event_id=str(uuid.uuid4()),
            aggregate_id=customer.customer_id,
            correlation_id=cmd.correlation_id,
            customer_id=customer.customer_id,
            changes=cmd.updates,
            previous_values=old_values,
            timestamp=datetime.utcnow()
        )]
    
    async def _handle_interaction(self, cmd: ProcessInteractionCommand) -> List[DomainEvent]:
        """Traite une interaction avec un agent"""
        
        customer = await self.repository.get(cmd.customer_id)
        if not customer:
            raise ValueError(f"Client {cmd.customer_id} non trouvé")
        
        # Enregistrement de l'interaction
        interaction = Interaction(
            interaction_id=str(uuid.uuid4()),
            customer_id=cmd.customer_id,
            agent_id=cmd.agent_id,
            interaction_type=cmd.interaction_type,
            content=cmd.content,
            resolution=cmd.resolution,
            sentiment_score=cmd.sentiment_score,
            timestamp=datetime.utcnow()
        )
        
        customer.interactions.append(interaction)
        await self.repository.save(customer)
        
        # Génération de l'événement
        return [InteractionRecordedEvent(
            event_id=str(uuid.uuid4()),
            aggregate_id=customer.customer_id,
            correlation_id=cmd.correlation_id,
            customer_id=cmd.customer_id,
            agent_id=cmd.agent_id,
            interaction_id=interaction.interaction_id,
            interaction_type=cmd.interaction_type,
            sentiment_score=cmd.sentiment_score,
            timestamp=datetime.utcnow()
        )]
```

### Projecteurs et Modèles de Lecture

Les projecteurs sont responsables de la transformation des événements de domaine en vues de lecture optimisées. Chaque projecteur maintient une ou plusieurs vues spécialisées, mises à jour de manière idempotente à partir du flux d'événements. L'idempotence est cruciale car un événement peut être traité plusieurs fois en cas de redémarrage ou de rééquilibrage des consommateurs.

```python
# cqrs/projectors.py
from abc import ABC, abstractmethod
from typing import Dict, Any, List
from datetime import datetime

class Projector(ABC):
    """Classe de base pour les projecteurs"""
    
    @abstractmethod
    async def project(self, event: DomainEvent) -> None:
        """Projette un événement vers le modèle de lecture"""
        pass
    
    @abstractmethod
    def handles(self) -> List[str]:
        """Retourne la liste des types d'événements gérés"""
        pass

class CustomerProfileProjector(Projector):
    """Projecteur pour la vue profil client"""
    
    def __init__(self, read_store):
        self.store = read_store
    
    def handles(self) -> List[str]:
        return [
            'customer.created',
            'customer.updated',
            'interaction.recorded'
        ]
    
    async def project(self, event: DomainEvent) -> None:
        """Projette un événement vers la vue profil client"""
        
        handlers = {
            'customer.created': self._project_created,
            'customer.updated': self._project_updated,
            'interaction.recorded': self._project_interaction
        }
        
        handler = handlers.get(event.event_type)
        if handler:
            await handler(event)
    
    async def _project_created(self, event: CustomerCreatedEvent):
        """Projette la création d'un client"""
        profile = {
            'customer_id': event.customer_id,
            'name': event.name,
            'email': event.email,
            'created_at': event.timestamp.isoformat(),
            'total_interactions': 0,
            'last_interaction': None,
            'sentiment_trend': [],
            'preferred_channels': [],
            'version': 1
        }
        await self.store.upsert('customer_profiles', event.customer_id, profile)
    
    async def _project_updated(self, event: CustomerUpdatedEvent):
        """Projette la mise à jour d'un client"""
        profile = await self.store.get('customer_profiles', event.customer_id)
        if profile:
            for key, value in event.changes.items():
                if key in profile:
                    profile[key] = value
            profile['version'] += 1
            await self.store.upsert('customer_profiles', event.customer_id, profile)
    
    async def _project_interaction(self, event: InteractionRecordedEvent):
        """Projette une interaction"""
        profile = await self.store.get('customer_profiles', event.customer_id)
        if profile:
            profile['total_interactions'] += 1
            profile['last_interaction'] = event.timestamp.isoformat()
            
            # Mise à jour de la tendance de sentiment
            if event.sentiment_score is not None:
                profile['sentiment_trend'].append({
                    'timestamp': event.timestamp.isoformat(),
                    'score': event.sentiment_score
                })
                # Garder les 10 derniers
                profile['sentiment_trend'] = profile['sentiment_trend'][-10:]
            
            profile['version'] += 1
            await self.store.upsert('customer_profiles', event.customer_id, profile)


class Agent360ViewProjector(Projector):
    """Projecteur pour la vue 360° utilisée par les agents"""
    
    def __init__(self, read_store, enrichment_service):
        self.store = read_store
        self.enrichment = enrichment_service
    
    def handles(self) -> List[str]:
        return [
            'customer.created',
            'customer.updated',
            'interaction.recorded',
            'order.placed',
            'order.completed',
            'support.ticket.created',
            'support.ticket.resolved'
        ]
    
    async def project(self, event: DomainEvent) -> None:
        """Projette vers la vue 360°"""
        
        customer_id = event.aggregate_id
        view = await self.store.get('customer_360', customer_id)
        
        if view is None:
            view = self._create_empty_view(customer_id)
        
        # Mise à jour selon le type d'événement
        if event.event_type == 'customer.created':
            view['profile'] = {
                'name': event.name,
                'email': event.email,
                'created_at': event.timestamp.isoformat()
            }
        
        elif event.event_type == 'interaction.recorded':
            view['interactions']['total'] += 1
            view['interactions']['last_at'] = event.timestamp.isoformat()
            view['interactions']['by_type'][event.interaction_type] = \
                view['interactions']['by_type'].get(event.interaction_type, 0) + 1
            
            if event.sentiment_score is not None:
                self._update_sentiment_metrics(view, event.sentiment_score)
        
        elif event.event_type == 'order.placed':
            view['orders']['total'] += 1
            view['orders']['total_value'] += event.order_value
            view['orders']['last_at'] = event.timestamp.isoformat()
        
        elif event.event_type == 'support.ticket.created':
            view['support']['open_tickets'] += 1
            view['support']['total_tickets'] += 1
        
        elif event.event_type == 'support.ticket.resolved':
            view['support']['open_tickets'] = max(0, view['support']['open_tickets'] - 1)
        
        # Enrichissement contextuel pour les agents
        view['context'] = await self.enrichment.enrich(view)
        view['updated_at'] = datetime.utcnow().isoformat()
        view['version'] += 1
        
        await self.store.upsert('customer_360', customer_id, view)
    
    def _create_empty_view(self, customer_id: str) -> Dict[str, Any]:
        """Crée une vue 360° vide"""
        return {
            'customer_id': customer_id,
            'profile': {},
            'interactions': {
                'total': 0,
                'last_at': None,
                'by_type': {},
                'sentiment_avg': None,
                'sentiment_trend': 'neutral'
            },
            'orders': {
                'total': 0,
                'total_value': 0,
                'last_at': None
            },
            'support': {
                'open_tickets': 0,
                'total_tickets': 0,
                'avg_resolution_time': None
            },
            'context': {},
            'version': 0,
            'updated_at': None
        }
    
    def _update_sentiment_metrics(self, view: dict, score: float):
        """Met à jour les métriques de sentiment"""
        interactions = view['interactions']
        
        if interactions['sentiment_avg'] is None:
            interactions['sentiment_avg'] = score
        else:
            # Moyenne mobile exponentielle
            alpha = 0.3
            interactions['sentiment_avg'] = (
                alpha * score + (1 - alpha) * interactions['sentiment_avg']
            )
        
        # Détermination de la tendance
        avg = interactions['sentiment_avg']
        if avg > 0.6:
            interactions['sentiment_trend'] = 'positive'
        elif avg < 0.4:
            interactions['sentiment_trend'] = 'negative'
        else:
            interactions['sentiment_trend'] = 'neutral'
```

> **Note technique**  
> Les projecteurs doivent être idempotents : le traitement multiple d'un même événement doit produire le même résultat. Utilisez le numéro de version ou l'offset Kafka pour détecter et ignorer les événements déjà traités.

---

## II.9.3 Event Sourcing

### Philosophie de l'Event Sourcing

L'Event Sourcing représente un changement de paradigme fondamental dans la persistance des données. Au lieu de stocker l'état courant d'une entité, on stocke la séquence complète des événements qui ont conduit à cet état. L'état actuel est alors une fonction déterministe de cette séquence d'événements : State(t) = fold(apply, InitialState, Events[0..t]).

Cette approche offre plusieurs avantages majeurs pour les systèmes agentiques. La traçabilité complète permet de comprendre exactement comment et pourquoi le système est arrivé à son état actuel — information précieuse pour le débogage des comportements des agents. La capacité de reconstituer l'état à n'importe quel moment dans le temps facilite l'analyse rétrospective et la correction des erreurs. La possibilité de rejouer les événements permet de créer de nouvelles projections sans modifier les données sources.

L'Event Sourcing s'intègre naturellement avec CQRS : les événements constituent le mécanisme de synchronisation entre le modèle d'écriture et les modèles de lecture. Combiné avec le patron Saga, il fournit également une piste d'audit complète des transactions distribuées.

### Implémentation de l'Event Store

L'Event Store est le composant central de l'architecture Event Sourcing. Il doit garantir l'ordonnancement des événements au sein d'un agrégat, l'atomicité des écritures, et la possibilité de lire efficacement l'historique complet ou partiel d'un agrégat.

```python
# eventsourcing/store.py
from dataclasses import dataclass, field
from typing import List, Optional, Iterator, Callable
from datetime import datetime
import json
import hashlib

@dataclass
class StoredEvent:
    """Événement persisté dans l'Event Store"""
    event_id: str
    aggregate_type: str
    aggregate_id: str
    sequence_number: int
    event_type: str
    event_data: dict
    metadata: dict
    timestamp: datetime
    checksum: str = ""
    
    def __post_init__(self):
        if not self.checksum:
            self.checksum = self._compute_checksum()
    
    def _compute_checksum(self) -> str:
        """Calcule un checksum pour l'intégrité"""
        content = f"{self.event_id}{self.aggregate_id}{self.sequence_number}{json.dumps(self.event_data, sort_keys=True)}"
        return hashlib.sha256(content.encode()).hexdigest()[:16]


class EventStore:
    """Event Store basé sur Kafka et PostgreSQL"""
    
    def __init__(self, db_pool, kafka_producer, config: dict):
        self.db = db_pool
        self.producer = kafka_producer
        self.config = config
        self.topic_prefix = config.get('topic_prefix', 'events')
    
    async def append(
        self,
        aggregate_type: str,
        aggregate_id: str,
        events: List[DomainEvent],
        expected_version: int = -1
    ) -> List[StoredEvent]:
        """Ajoute des événements à l'historique d'un agrégat"""
        
        async with self.db.acquire() as conn:
            async with conn.transaction():
                # Verrouillage optimiste
                current_version = await self._get_current_version(
                    conn, aggregate_type, aggregate_id
                )
                
                if expected_version >= 0 and current_version != expected_version:
                    raise ConcurrencyError(
                        f"Version attendue {expected_version}, "
                        f"version actuelle {current_version}"
                    )
                
                stored_events = []
                next_sequence = current_version + 1
                
                for event in events:
                    stored_event = StoredEvent(
                        event_id=event.event_id,
                        aggregate_type=aggregate_type,
                        aggregate_id=aggregate_id,
                        sequence_number=next_sequence,
                        event_type=event.event_type,
                        event_data=event.to_dict(),
                        metadata={
                            'correlation_id': event.correlation_id,
                            'causation_id': getattr(event, 'causation_id', None)
                        },
                        timestamp=event.timestamp
                    )
                    
                    # Persistance dans PostgreSQL
                    await self._insert_event(conn, stored_event)
                    
                    stored_events.append(stored_event)
                    next_sequence += 1
                
                # Publication sur Kafka après commit
                for stored_event in stored_events:
                    await self._publish_to_kafka(stored_event)
                
                return stored_events
    
    async def get_events(
        self,
        aggregate_type: str,
        aggregate_id: str,
        from_version: int = 0,
        to_version: Optional[int] = None
    ) -> List[StoredEvent]:
        """Récupère les événements d'un agrégat"""
        
        query = """
            SELECT event_id, aggregate_type, aggregate_id, sequence_number,
                   event_type, event_data, metadata, timestamp, checksum
            FROM events
            WHERE aggregate_type = $1 AND aggregate_id = $2
                  AND sequence_number >= $3
        """
        params = [aggregate_type, aggregate_id, from_version]
        
        if to_version is not None:
            query += " AND sequence_number <= $4"
            params.append(to_version)
        
        query += " ORDER BY sequence_number ASC"
        
        async with self.db.acquire() as conn:
            rows = await conn.fetch(query, *params)
            return [self._row_to_event(row) for row in rows]
    
    async def get_all_events(
        self,
        from_position: int = 0,
        batch_size: int = 1000
    ) -> Iterator[StoredEvent]:
        """Récupère tous les événements (pour replay global)"""
        
        query = """
            SELECT event_id, aggregate_type, aggregate_id, sequence_number,
                   event_type, event_data, metadata, timestamp, checksum
            FROM events
            WHERE global_position > $1
            ORDER BY global_position ASC
            LIMIT $2
        """
        
        position = from_position
        
        while True:
            async with self.db.acquire() as conn:
                rows = await conn.fetch(query, position, batch_size)
            
            if not rows:
                break
            
            for row in rows:
                event = self._row_to_event(row)
                yield event
                position = row['global_position']
    
    async def _get_current_version(
        self,
        conn,
        aggregate_type: str,
        aggregate_id: str
    ) -> int:
        """Récupère la version actuelle d'un agrégat"""
        
        result = await conn.fetchval("""
            SELECT COALESCE(MAX(sequence_number), -1)
            FROM events
            WHERE aggregate_type = $1 AND aggregate_id = $2
        """, aggregate_type, aggregate_id)
        
        return result
    
    async def _insert_event(self, conn, event: StoredEvent):
        """Insère un événement dans la base"""
        
        await conn.execute("""
            INSERT INTO events (
                event_id, aggregate_type, aggregate_id, sequence_number,
                event_type, event_data, metadata, timestamp, checksum
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        """,
            event.event_id,
            event.aggregate_type,
            event.aggregate_id,
            event.sequence_number,
            event.event_type,
            json.dumps(event.event_data),
            json.dumps(event.metadata),
            event.timestamp,
            event.checksum
        )
    
    async def _publish_to_kafka(self, event: StoredEvent):
        """Publie l'événement sur Kafka"""
        
        topic = f"{self.topic_prefix}.{event.aggregate_type}"
        
        self.producer.produce(
            topic=topic,
            key=event.aggregate_id.encode(),
            value=json.dumps({
                'event_id': event.event_id,
                'aggregate_id': event.aggregate_id,
                'sequence_number': event.sequence_number,
                'event_type': event.event_type,
                'event_data': event.event_data,
                'metadata': event.metadata,
                'timestamp': event.timestamp.isoformat()
            }).encode(),
            headers=[
                ('event_type', event.event_type.encode()),
                ('aggregate_type', event.aggregate_type.encode()),
                ('sequence', str(event.sequence_number).encode())
            ]
        )
        self.producer.flush()
    
    def _row_to_event(self, row) -> StoredEvent:
        """Convertit une ligne de base de données en événement"""
        return StoredEvent(
            event_id=row['event_id'],
            aggregate_type=row['aggregate_type'],
            aggregate_id=row['aggregate_id'],
            sequence_number=row['sequence_number'],
            event_type=row['event_type'],
            event_data=json.loads(row['event_data']),
            metadata=json.loads(row['metadata']),
            timestamp=row['timestamp'],
            checksum=row['checksum']
        )
```

### Agrégats et Reconstruction d'État

Un agrégat en Event Sourcing est une entité dont l'état est reconstruit à partir de ses événements. L'agrégat définit les règles métier et génère de nouveaux événements lorsque des commandes sont traitées. La méthode apply permet de reconstituer l'état à partir de l'historique.

```python
# eventsourcing/aggregate.py
from abc import ABC, abstractmethod
from typing import List, Optional, Type
from dataclasses import dataclass, field

class AggregateRoot(ABC):
    """Classe de base pour les agrégats Event-Sourced"""
    
    def __init__(self, aggregate_id: str):
        self._id = aggregate_id
        self._version = -1
        self._pending_events: List[DomainEvent] = []
    
    @property
    def id(self) -> str:
        return self._id
    
    @property
    def version(self) -> int:
        return self._version
    
    @property
    def pending_events(self) -> List[DomainEvent]:
        return self._pending_events.copy()
    
    def clear_pending_events(self):
        self._pending_events.clear()
    
    def load_from_history(self, events: List[StoredEvent]):
        """Reconstruit l'état à partir de l'historique"""
        for event in events:
            self._apply_event(event.event_data, event.event_type)
            self._version = event.sequence_number
    
    def _raise_event(self, event: DomainEvent):
        """Enregistre un nouvel événement"""
        self._apply_event(event.to_dict(), event.event_type)
        self._pending_events.append(event)
    
    def _apply_event(self, event_data: dict, event_type: str):
        """Applique un événement à l'état"""
        method_name = f"_apply_{event_type.replace('.', '_')}"
        method = getattr(self, method_name, None)
        
        if method:
            method(event_data)
        else:
            # Log warning pour événement non géré
            pass
    
    @abstractmethod
    def _get_aggregate_type(self) -> str:
        """Retourne le type de l'agrégat"""
        pass


@dataclass
class CustomerAggregate(AggregateRoot):
    """Agrégat Client avec Event Sourcing"""
    
    name: str = ""
    email: str = ""
    phone: Optional[str] = None
    status: str = "active"
    interactions: List[dict] = field(default_factory=list)
    preferences: dict = field(default_factory=dict)
    
    def __init__(self, customer_id: str):
        super().__init__(customer_id)
    
    def _get_aggregate_type(self) -> str:
        return "customer"
    
    # Commandes
    
    @classmethod
    def create(cls, customer_id: str, name: str, email: str, 
               phone: Optional[str] = None) -> 'CustomerAggregate':
        """Factory method pour créer un nouveau client"""
        customer = cls(customer_id)
        
        customer._raise_event(CustomerCreatedEvent(
            event_id=str(uuid.uuid4()),
            aggregate_id=customer_id,
            customer_id=customer_id,
            name=name,
            email=email,
            phone=phone,
            timestamp=datetime.utcnow()
        ))
        
        return customer
    
    def update_contact(self, email: Optional[str] = None, 
                       phone: Optional[str] = None):
        """Met à jour les informations de contact"""
        changes = {}
        if email and email != self.email:
            changes['email'] = email
        if phone and phone != self.phone:
            changes['phone'] = phone
        
        if changes:
            self._raise_event(CustomerContactUpdatedEvent(
                event_id=str(uuid.uuid4()),
                aggregate_id=self._id,
                customer_id=self._id,
                changes=changes,
                timestamp=datetime.utcnow()
            ))
    
    def record_interaction(self, agent_id: str, interaction_type: str,
                          content: dict, sentiment_score: Optional[float] = None):
        """Enregistre une interaction avec un agent"""
        interaction_id = str(uuid.uuid4())
        
        self._raise_event(InteractionRecordedEvent(
            event_id=str(uuid.uuid4()),
            aggregate_id=self._id,
            customer_id=self._id,
            agent_id=agent_id,
            interaction_id=interaction_id,
            interaction_type=interaction_type,
            content=content,
            sentiment_score=sentiment_score,
            timestamp=datetime.utcnow()
        ))
    
    def deactivate(self, reason: str):
        """Désactive le compte client"""
        if self.status != 'active':
            raise ValueError("Le client n'est pas actif")
        
        self._raise_event(CustomerDeactivatedEvent(
            event_id=str(uuid.uuid4()),
            aggregate_id=self._id,
            customer_id=self._id,
            reason=reason,
            timestamp=datetime.utcnow()
        ))
    
    # Applicateurs d'événements
    
    def _apply_customer_created(self, event_data: dict):
        self.name = event_data['name']
        self.email = event_data['email']
        self.phone = event_data.get('phone')
        self.status = 'active'
    
    def _apply_customer_contact_updated(self, event_data: dict):
        changes = event_data.get('changes', {})
        if 'email' in changes:
            self.email = changes['email']
        if 'phone' in changes:
            self.phone = changes['phone']
    
    def _apply_interaction_recorded(self, event_data: dict):
        self.interactions.append({
            'interaction_id': event_data['interaction_id'],
            'agent_id': event_data['agent_id'],
            'type': event_data['interaction_type'],
            'timestamp': event_data['timestamp']
        })
    
    def _apply_customer_deactivated(self, event_data: dict):
        self.status = 'inactive'


class AggregateRepository:
    """Repository pour les agrégats Event-Sourced"""
    
    def __init__(self, event_store: EventStore, aggregate_class: Type[AggregateRoot]):
        self.event_store = event_store
        self.aggregate_class = aggregate_class
    
    async def get(self, aggregate_id: str) -> Optional[AggregateRoot]:
        """Charge un agrégat depuis l'Event Store"""
        aggregate = self.aggregate_class(aggregate_id)
        
        events = await self.event_store.get_events(
            aggregate._get_aggregate_type(),
            aggregate_id
        )
        
        if not events:
            return None
        
        aggregate.load_from_history(events)
        return aggregate
    
    async def save(self, aggregate: AggregateRoot):
        """Sauvegarde les nouveaux événements d'un agrégat"""
        pending = aggregate.pending_events
        
        if not pending:
            return
        
        await self.event_store.append(
            aggregate_type=aggregate._get_aggregate_type(),
            aggregate_id=aggregate.id,
            events=pending,
            expected_version=aggregate.version
        )
        
        aggregate.clear_pending_events()
```

> **Attention**  
> L'Event Sourcing génère un volume important de données. Implémentez des mécanismes de snapshot pour les agrégats avec un long historique, et définissez des politiques de rétention adaptées à vos besoins de conformité.

### Snapshots et Optimisation de Performance

À mesure que le nombre d'événements d'un agrégat croît, le temps de reconstruction de l'état augmente linéairement. Pour un agrégat avec des milliers d'événements, cette reconstruction peut devenir prohibitive. Les snapshots résolvent ce problème en capturant périodiquement l'état complet de l'agrégat, permettant de ne rejouer que les événements postérieurs au dernier snapshot.

```python
# eventsourcing/snapshots.py
from dataclasses import dataclass
from typing import Optional, Dict, Any
from datetime import datetime
import json

@dataclass
class Snapshot:
    """Capture de l'état d'un agrégat à un instant donné"""
    aggregate_type: str
    aggregate_id: str
    version: int
    state: Dict[str, Any]
    timestamp: datetime
    checksum: str = ""
    
    def to_json(self) -> str:
        return json.dumps({
            'aggregate_type': self.aggregate_type,
            'aggregate_id': self.aggregate_id,
            'version': self.version,
            'state': self.state,
            'timestamp': self.timestamp.isoformat()
        })


class SnapshotStore:
    """Gestionnaire de snapshots pour les agrégats"""
    
    def __init__(self, db_pool, config: dict):
        self.db = db_pool
        self.snapshot_interval = config.get('snapshot_interval', 100)
        self.max_snapshots_per_aggregate = config.get('max_snapshots', 3)
    
    async def save_snapshot(self, aggregate: AggregateRoot):
        """Sauvegarde un snapshot de l'agrégat"""
        
        snapshot = Snapshot(
            aggregate_type=aggregate._get_aggregate_type(),
            aggregate_id=aggregate.id,
            version=aggregate.version,
            state=aggregate.to_dict(),
            timestamp=datetime.utcnow()
        )
        
        async with self.db.acquire() as conn:
            async with conn.transaction():
                # Insertion du nouveau snapshot
                await conn.execute("""
                    INSERT INTO snapshots 
                    (aggregate_type, aggregate_id, version, state, timestamp)
                    VALUES ($1, $2, $3, $4, $5)
                """,
                    snapshot.aggregate_type,
                    snapshot.aggregate_id,
                    snapshot.version,
                    snapshot.to_json(),
                    snapshot.timestamp
                )
                
                # Nettoyage des anciens snapshots
                await self._cleanup_old_snapshots(
                    conn,
                    snapshot.aggregate_type,
                    snapshot.aggregate_id
                )
    
    async def get_latest_snapshot(
        self,
        aggregate_type: str,
        aggregate_id: str
    ) -> Optional[Snapshot]:
        """Récupère le dernier snapshot d'un agrégat"""
        
        async with self.db.acquire() as conn:
            row = await conn.fetchrow("""
                SELECT aggregate_type, aggregate_id, version, state, timestamp
                FROM snapshots
                WHERE aggregate_type = $1 AND aggregate_id = $2
                ORDER BY version DESC
                LIMIT 1
            """, aggregate_type, aggregate_id)
            
            if row:
                state_data = json.loads(row['state'])
                return Snapshot(
                    aggregate_type=row['aggregate_type'],
                    aggregate_id=row['aggregate_id'],
                    version=row['version'],
                    state=state_data['state'],
                    timestamp=row['timestamp']
                )
            
            return None
    
    async def should_snapshot(self, aggregate: AggregateRoot) -> bool:
        """Détermine si un snapshot doit être créé"""
        
        if aggregate.version < self.snapshot_interval:
            return False
        
        latest = await self.get_latest_snapshot(
            aggregate._get_aggregate_type(),
            aggregate.id
        )
        
        if latest is None:
            return True
        
        events_since_snapshot = aggregate.version - latest.version
        return events_since_snapshot >= self.snapshot_interval
    
    async def _cleanup_old_snapshots(
        self,
        conn,
        aggregate_type: str,
        aggregate_id: str
    ):
        """Supprime les snapshots excédentaires"""
        
        await conn.execute("""
            DELETE FROM snapshots
            WHERE aggregate_type = $1 
              AND aggregate_id = $2
              AND version NOT IN (
                  SELECT version FROM snapshots
                  WHERE aggregate_type = $1 AND aggregate_id = $2
                  ORDER BY version DESC
                  LIMIT $3
              )
        """, aggregate_type, aggregate_id, self.max_snapshots_per_aggregate)


class OptimizedAggregateRepository:
    """Repository avec support des snapshots"""
    
    def __init__(
        self,
        event_store: EventStore,
        snapshot_store: SnapshotStore,
        aggregate_class: Type[AggregateRoot]
    ):
        self.event_store = event_store
        self.snapshot_store = snapshot_store
        self.aggregate_class = aggregate_class
    
    async def get(self, aggregate_id: str) -> Optional[AggregateRoot]:
        """Charge un agrégat en utilisant les snapshots si disponibles"""
        
        aggregate = self.aggregate_class(aggregate_id)
        aggregate_type = aggregate._get_aggregate_type()
        
        # Tentative de chargement depuis snapshot
        snapshot = await self.snapshot_store.get_latest_snapshot(
            aggregate_type, aggregate_id
        )
        
        from_version = 0
        
        if snapshot:
            # Restauration depuis le snapshot
            aggregate.restore_from_snapshot(snapshot.state)
            aggregate._version = snapshot.version
            from_version = snapshot.version + 1
        
        # Chargement des événements manquants
        events = await self.event_store.get_events(
            aggregate_type,
            aggregate_id,
            from_version=from_version
        )
        
        if not events and not snapshot:
            return None
        
        # Application des événements récents
        for event in events:
            aggregate._apply_event(event.event_data, event.event_type)
            aggregate._version = event.sequence_number
        
        return aggregate
    
    async def save(self, aggregate: AggregateRoot):
        """Sauvegarde l'agrégat avec création optionnelle de snapshot"""
        
        pending = aggregate.pending_events
        
        if not pending:
            return
        
        # Sauvegarde des événements
        await self.event_store.append(
            aggregate_type=aggregate._get_aggregate_type(),
            aggregate_id=aggregate.id,
            events=pending,
            expected_version=aggregate.version - len(pending)
        )
        
        aggregate.clear_pending_events()
        
        # Vérification si snapshot nécessaire
        if await self.snapshot_store.should_snapshot(aggregate):
            await self.snapshot_store.save_snapshot(aggregate)
```

Le mécanisme de snapshot s'intègre de manière transparente avec l'Event Store. Le repository optimisé charge d'abord le dernier snapshot disponible, puis applique uniquement les événements survenus depuis. Cette approche réduit considérablement le temps de chargement pour les agrégats avec un long historique tout en préservant la capacité de reconstruction complète si nécessaire.

### Projection Replay et Reconstruction

L'un des avantages majeurs de l'Event Sourcing est la capacité de reconstruire des projections ou d'en créer de nouvelles à partir de l'historique complet des événements. Cette fonctionnalité est essentielle pour corriger des erreurs dans les projecteurs, ajouter de nouvelles vues, ou migrer vers de nouveaux schémas de données.

```python
# eventsourcing/replay.py
from typing import Callable, List, Optional, Dict, Any
from datetime import datetime
import asyncio

class ProjectionRebuilder:
    """Outil de reconstruction des projections"""
    
    def __init__(
        self,
        event_store: EventStore,
        projectors: List[Projector],
        config: dict
    ):
        self.event_store = event_store
        self.projectors = {p.__class__.__name__: p for p in projectors}
        self.batch_size = config.get('batch_size', 1000)
        self.checkpoint_interval = config.get('checkpoint_interval', 10000)
        self.checkpoint_store = config.get('checkpoint_store')
    
    async def rebuild_projection(
        self,
        projector_name: str,
        from_position: int = 0,
        to_position: Optional[int] = None,
        progress_callback: Optional[Callable] = None
    ):
        """Reconstruit une projection depuis l'Event Store"""
        
        projector = self.projectors.get(projector_name)
        if not projector:
            raise ValueError(f"Projecteur {projector_name} non trouvé")
        
        # Nettoyage de la projection existante
        await projector.clear()
        
        position = from_position
        processed = 0
        start_time = datetime.utcnow()
        
        async for event in self.event_store.get_all_events(
            from_position=position,
            batch_size=self.batch_size
        ):
            # Vérification de la limite
            if to_position and event.sequence_number > to_position:
                break
            
            # Projection de l'événement si pertinent
            if event.event_type in projector.handles():
                await projector.project(event)
            
            processed += 1
            position = event.sequence_number
            
            # Checkpoint périodique
            if processed % self.checkpoint_interval == 0:
                await self._save_checkpoint(projector_name, position)
                
                if progress_callback:
                    elapsed = (datetime.utcnow() - start_time).total_seconds()
                    rate = processed / elapsed if elapsed > 0 else 0
                    await progress_callback({
                        'projector': projector_name,
                        'processed': processed,
                        'position': position,
                        'rate': rate,
                        'elapsed_seconds': elapsed
                    })
        
        # Checkpoint final
        await self._save_checkpoint(projector_name, position, completed=True)
        
        return {
            'projector': projector_name,
            'events_processed': processed,
            'final_position': position,
            'duration_seconds': (datetime.utcnow() - start_time).total_seconds()
        }
    
    async def rebuild_all_projections(
        self,
        from_position: int = 0,
        progress_callback: Optional[Callable] = None
    ):
        """Reconstruit toutes les projections en parallèle"""
        
        # Groupement des événements par batch
        position = from_position
        processed = 0
        
        async for event in self.event_store.get_all_events(
            from_position=position,
            batch_size=self.batch_size
        ):
            # Distribution aux projecteurs concernés
            tasks = []
            for projector in self.projectors.values():
                if event.event_type in projector.handles():
                    tasks.append(projector.project(event))
            
            if tasks:
                await asyncio.gather(*tasks)
            
            processed += 1
            position = event.sequence_number
            
            if processed % self.checkpoint_interval == 0:
                for name in self.projectors.keys():
                    await self._save_checkpoint(name, position)
                
                if progress_callback:
                    await progress_callback({
                        'processed': processed,
                        'position': position
                    })
    
    async def _save_checkpoint(
        self,
        projector_name: str,
        position: int,
        completed: bool = False
    ):
        """Sauvegarde un point de reprise"""
        
        if self.checkpoint_store:
            await self.checkpoint_store.save({
                'projector': projector_name,
                'position': position,
                'timestamp': datetime.utcnow().isoformat(),
                'completed': completed
            })


class IncrementalProjectionUpdater:
    """Mise à jour incrémentale des projections depuis Kafka"""
    
    def __init__(
        self,
        consumer,
        projectors: List[Projector],
        config: dict
    ):
        self.consumer = consumer
        self.projectors = projectors
        self.projector_map = self._build_projector_map()
        self.commit_interval = config.get('commit_interval', 100)
    
    def _build_projector_map(self) -> Dict[str, List[Projector]]:
        """Construit un mapping event_type -> projectors"""
        mapping = {}
        for projector in self.projectors:
            for event_type in projector.handles():
                if event_type not in mapping:
                    mapping[event_type] = []
                mapping[event_type].append(projector)
        return mapping
    
    async def run(self):
        """Boucle principale de mise à jour"""
        
        processed = 0
        
        while True:
            msg = self.consumer.poll(1.0)
            
            if msg is None:
                continue
            
            if msg.error():
                continue
            
            event = self._parse_event(msg)
            event_type = event.get('event_type')
            
            # Distribution aux projecteurs concernés
            projectors = self.projector_map.get(event_type, [])
            for projector in projectors:
                await projector.project(event)
            
            processed += 1
            
            # Commit périodique
            if processed % self.commit_interval == 0:
                self.consumer.commit()
    
    def _parse_event(self, msg) -> dict:
        """Parse un message Kafka en événement"""
        import json
        return json.loads(msg.value().decode())
```

La reconstruction des projections est une opération coûteuse qui doit être planifiée avec soin. En production, il est recommandé d'exécuter les reconstructions pendant les périodes de faible charge et de surveiller attentivement les ressources consommées. Le mécanisme de checkpoint permet de reprendre une reconstruction interrompue sans repartir du début.

---

## II.9.4 Patron « Outbox Transactionnel »

### Le Problème de la Double Écriture

Dans une architecture événementielle, chaque modification d'état doit être accompagnée de la publication d'un événement correspondant. Cependant, cette double opération — écriture en base de données et publication sur le broker de messages — pose un problème fondamental de cohérence. Si l'une des deux opérations échoue après que l'autre a réussi, le système se retrouve dans un état incohérent.

Considérons un scénario typique : un agent traite une demande client et doit mettre à jour la base de données puis publier un événement. Si la mise à jour réussit mais que la publication échoue (timeout réseau, broker indisponible), l'événement est perdu et les consommateurs ne seront jamais informés du changement. Inversement, si la publication réussit mais que la transaction de base de données échoue ensuite, un événement a été émis pour une modification qui n'a pas eu lieu.

Le patron Outbox Transactionnel résout ce problème en utilisant la base de données comme intermédiaire fiable. Les événements sont d'abord écrits dans une table outbox au sein de la même transaction que les modifications métier. Un processus séparé lit ensuite cette table et publie les événements sur le broker, garantissant ainsi la cohérence entre l'état de la base de données et les événements publiés.

### Architecture du Patron Outbox

L'architecture du patron Outbox comprend trois composants principaux : la table outbox qui stocke les événements en attente, le service métier qui écrit dans cette table transactionnellement, et le relay qui lit la table et publie les événements.

```sql
-- Schema de la table Outbox
CREATE TABLE outbox (
    id BIGSERIAL PRIMARY KEY,
    aggregate_type VARCHAR(255) NOT NULL,
    aggregate_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    published_at TIMESTAMP WITH TIME ZONE,
    retry_count INTEGER DEFAULT 0,
    last_error TEXT,
    
    -- Index pour le polling efficace
    INDEX idx_outbox_unpublished (published_at) WHERE published_at IS NULL,
    INDEX idx_outbox_aggregate (aggregate_type, aggregate_id)
);

-- Table de tracking pour la publication
CREATE TABLE outbox_position (
    consumer_id VARCHAR(255) PRIMARY KEY,
    last_processed_id BIGINT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

```python
# outbox/service.py
from contextlib import asynccontextmanager
from typing import List, Optional
import json

class OutboxService:
    """Service de gestion de l'outbox transactionnel"""
    
    def __init__(self, db_pool):
        self.db = db_pool
    
    @asynccontextmanager
    async def transaction(self):
        """Gestionnaire de contexte pour transaction avec outbox"""
        async with self.db.acquire() as conn:
            async with conn.transaction():
                yield OutboxTransaction(conn)
    
    async def get_unpublished(
        self,
        batch_size: int = 100,
        consumer_id: str = "default"
    ) -> List[dict]:
        """Récupère les événements non publiés"""
        
        async with self.db.acquire() as conn:
            # Récupération de la dernière position
            last_id = await conn.fetchval("""
                SELECT COALESCE(last_processed_id, 0)
                FROM outbox_position
                WHERE consumer_id = $1
            """, consumer_id) or 0
            
            # Récupération des événements
            rows = await conn.fetch("""
                SELECT id, aggregate_type, aggregate_id, event_type,
                       payload, metadata, created_at
                FROM outbox
                WHERE id > $1 AND published_at IS NULL
                ORDER BY id ASC
                LIMIT $2
            """, last_id, batch_size)
            
            return [dict(row) for row in rows]
    
    async def mark_published(
        self,
        event_ids: List[int],
        consumer_id: str = "default"
    ):
        """Marque des événements comme publiés"""
        
        if not event_ids:
            return
        
        async with self.db.acquire() as conn:
            async with conn.transaction():
                # Mise à jour des événements
                await conn.execute("""
                    UPDATE outbox
                    SET published_at = NOW()
                    WHERE id = ANY($1)
                """, event_ids)
                
                # Mise à jour de la position
                max_id = max(event_ids)
                await conn.execute("""
                    INSERT INTO outbox_position (consumer_id, last_processed_id)
                    VALUES ($1, $2)
                    ON CONFLICT (consumer_id)
                    DO UPDATE SET 
                        last_processed_id = EXCLUDED.last_processed_id,
                        updated_at = NOW()
                """, consumer_id, max_id)
    
    async def mark_failed(
        self,
        event_id: int,
        error: str
    ):
        """Marque un événement comme échoué"""
        
        async with self.db.acquire() as conn:
            await conn.execute("""
                UPDATE outbox
                SET retry_count = retry_count + 1,
                    last_error = $2
                WHERE id = $1
            """, event_id, error)


class OutboxTransaction:
    """Transaction avec support outbox"""
    
    def __init__(self, conn):
        self.conn = conn
        self._pending_events: List[dict] = []
    
    async def execute(self, query: str, *args):
        """Exécute une requête dans la transaction"""
        return await self.conn.execute(query, *args)
    
    async def fetch(self, query: str, *args):
        """Exécute une requête de lecture"""
        return await self.conn.fetch(query, *args)
    
    async def add_event(
        self,
        aggregate_type: str,
        aggregate_id: str,
        event_type: str,
        payload: dict,
        metadata: Optional[dict] = None
    ):
        """Ajoute un événement à l'outbox dans la transaction"""
        
        await self.conn.execute("""
            INSERT INTO outbox (aggregate_type, aggregate_id, event_type, payload, metadata)
            VALUES ($1, $2, $3, $4, $5)
        """,
            aggregate_type,
            aggregate_id,
            event_type,
            json.dumps(payload),
            json.dumps(metadata or {})
        )
```

### Outbox Relay avec Kafka Connect

Le relay peut être implémenté de plusieurs façons. La méthode la plus robuste utilise Kafka Connect avec le connecteur Debezium, qui capture les changements de la table outbox via le Change Data Capture (CDC). Cette approche élimine le polling et garantit une latence minimale.

```json
{
    "name": "outbox-connector",
    "config": {
        "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
        "database.hostname": "postgres",
        "database.port": "5432",
        "database.user": "debezium",
        "database.password": "${file:/secrets/db-password}",
        "database.dbname": "agents",
        "database.server.name": "agents-db",
        
        "table.include.list": "public.outbox",
        
        "transforms": "outbox",
        "transforms.outbox.type": "io.debezium.transforms.outbox.EventRouter",
        "transforms.outbox.table.fields.additional.placement": "aggregate_type:header,aggregate_id:header",
        "transforms.outbox.table.field.event.id": "id",
        "transforms.outbox.table.field.event.key": "aggregate_id",
        "transforms.outbox.table.field.event.type": "event_type",
        "transforms.outbox.table.field.event.payload": "payload",
        "transforms.outbox.table.field.event.timestamp": "created_at",
        "transforms.outbox.route.by.field": "aggregate_type",
        "transforms.outbox.route.topic.replacement": "events.${routedByValue}",
        
        "tombstones.on.delete": false,
        "key.converter": "org.apache.kafka.connect.storage.StringConverter",
        "value.converter": "org.apache.kafka.connect.json.JsonConverter",
        "value.converter.schemas.enable": false
    }
}
```

Pour les environnements où Debezium n'est pas disponible ou adapté, un relay basé sur le polling reste une option viable :

```python
# outbox/relay.py
import asyncio
from typing import Optional

class OutboxRelay:
    """Relay Outbox basé sur le polling"""
    
    def __init__(
        self,
        outbox_service: OutboxService,
        kafka_producer,
        config: dict
    ):
        self.outbox = outbox_service
        self.producer = kafka_producer
        self.config = config
        
        self.consumer_id = config.get('consumer_id', 'relay-1')
        self.batch_size = config.get('batch_size', 100)
        self.poll_interval = config.get('poll_interval', 1.0)
        self.topic_prefix = config.get('topic_prefix', 'events')
        
        self._running = False
    
    async def start(self):
        """Démarre le relay"""
        self._running = True
        
        while self._running:
            try:
                await self._process_batch()
            except Exception as e:
                # Log error, continue
                await asyncio.sleep(self.poll_interval * 2)
            
            await asyncio.sleep(self.poll_interval)
    
    def stop(self):
        """Arrête le relay"""
        self._running = False
    
    async def _process_batch(self):
        """Traite un lot d'événements"""
        
        events = await self.outbox.get_unpublished(
            batch_size=self.batch_size,
            consumer_id=self.consumer_id
        )
        
        if not events:
            return
        
        published_ids = []
        
        for event in events:
            try:
                await self._publish_event(event)
                published_ids.append(event['id'])
            except Exception as e:
                await self.outbox.mark_failed(event['id'], str(e))
        
        if published_ids:
            await self.outbox.mark_published(published_ids, self.consumer_id)
    
    async def _publish_event(self, event: dict):
        """Publie un événement sur Kafka"""
        
        topic = f"{self.topic_prefix}.{event['aggregate_type']}"
        
        # Publication synchrone pour garantir l'ordre
        future = self.producer.produce(
            topic=topic,
            key=event['aggregate_id'].encode(),
            value=json.dumps(event['payload']).encode(),
            headers=[
                ('event_type', event['event_type'].encode()),
                ('aggregate_type', event['aggregate_type'].encode()),
                ('aggregate_id', event['aggregate_id'].encode()),
                ('outbox_id', str(event['id']).encode())
            ]
        )
        
        self.producer.flush()
```

> **Bonnes pratiques**  
> Configurez une politique de rétention pour la table outbox : les événements publiés peuvent être supprimés après un délai configurable. Surveillez la taille de la table et le lag du relay comme indicateurs de santé du système.

---

## II.9.5 Gestion des Erreurs et Résilience

### Taxonomie des Erreurs dans les Systèmes Agentiques

Les systèmes agentiques sont exposés à une variété d'erreurs qui nécessitent des stratégies de traitement différenciées. Une taxonomie claire permet de définir les comportements appropriés pour chaque type d'erreur et d'éviter les réponses inadaptées qui pourraient aggraver la situation.

| Type | Caractéristiques | Exemples | Stratégie |
|------|------------------|----------|-----------|
| Transitoire | Temporaire, auto-résolution | Timeout réseau, surcharge | Retry avec backoff |
| Permanente | Ne se résout pas seule | Données invalides, autorisation | Dead Letter Queue |
| Cognitive | Erreur de l'agent IA | Hallucination, hors-sujet | Escalade humaine |
| Systémique | Défaillance infrastructure | Broker down, DB crash | Circuit breaker |

### Implémentation des Patterns de Résilience

La résilience d'un système agentique repose sur plusieurs patterns complémentaires : le retry avec backoff exponentiel pour les erreurs transitoires, le circuit breaker pour prévenir les cascades de défaillances, la dead letter queue pour isoler les messages problématiques, et le bulkhead pour limiter l'impact des défaillances.

```python
# resilience/patterns.py
from dataclasses import dataclass, field
from typing import Callable, Optional, Any
from datetime import datetime, timedelta
from enum import Enum
import asyncio
import random

class CircuitState(Enum):
    CLOSED = "closed"      # Fonctionnement normal
    OPEN = "open"          # Bloque les appels
    HALF_OPEN = "half_open"  # Test de récupération

@dataclass
class RetryConfig:
    """Configuration des retries"""
    max_attempts: int = 3
    base_delay: float = 1.0
    max_delay: float = 60.0
    exponential_base: float = 2.0
    jitter: bool = True
    retryable_exceptions: tuple = (Exception,)

class RetryHandler:
    """Gestionnaire de retry avec backoff exponentiel"""
    
    def __init__(self, config: RetryConfig):
        self.config = config
    
    async def execute(
        self,
        func: Callable,
        *args,
        **kwargs
    ) -> Any:
        """Exécute une fonction avec retry"""
        
        last_exception = None
        
        for attempt in range(self.config.max_attempts):
            try:
                return await func(*args, **kwargs)
            except self.config.retryable_exceptions as e:
                last_exception = e
                
                if attempt < self.config.max_attempts - 1:
                    delay = self._calculate_delay(attempt)
                    await asyncio.sleep(delay)
        
        raise last_exception
    
    def _calculate_delay(self, attempt: int) -> float:
        """Calcule le délai avec backoff exponentiel et jitter"""
        
        delay = min(
            self.config.base_delay * (self.config.exponential_base ** attempt),
            self.config.max_delay
        )
        
        if self.config.jitter:
            delay = delay * (0.5 + random.random())
        
        return delay


@dataclass
class CircuitBreakerConfig:
    """Configuration du circuit breaker"""
    failure_threshold: int = 5
    success_threshold: int = 3
    timeout: float = 30.0
    half_open_max_calls: int = 3

class CircuitBreaker:
    """Circuit breaker pour protection contre les cascades de défaillances"""
    
    def __init__(self, name: str, config: CircuitBreakerConfig):
        self.name = name
        self.config = config
        
        self._state = CircuitState.CLOSED
        self._failure_count = 0
        self._success_count = 0
        self._last_failure_time: Optional[datetime] = None
        self._half_open_calls = 0
    
    @property
    def state(self) -> CircuitState:
        """État actuel du circuit"""
        if self._state == CircuitState.OPEN:
            if self._should_attempt_reset():
                self._state = CircuitState.HALF_OPEN
                self._half_open_calls = 0
        return self._state
    
    async def execute(
        self,
        func: Callable,
        *args,
        **kwargs
    ) -> Any:
        """Exécute une fonction avec protection circuit breaker"""
        
        state = self.state
        
        if state == CircuitState.OPEN:
            raise CircuitOpenError(f"Circuit {self.name} is open")
        
        if state == CircuitState.HALF_OPEN:
            if self._half_open_calls >= self.config.half_open_max_calls:
                raise CircuitOpenError(f"Circuit {self.name} half-open limit reached")
            self._half_open_calls += 1
        
        try:
            result = await func(*args, **kwargs)
            self._on_success()
            return result
        except Exception as e:
            self._on_failure()
            raise
    
    def _on_success(self):
        """Appelé après un succès"""
        if self._state == CircuitState.HALF_OPEN:
            self._success_count += 1
            if self._success_count >= self.config.success_threshold:
                self._reset()
        else:
            self._failure_count = 0
    
    def _on_failure(self):
        """Appelé après un échec"""
        self._failure_count += 1
        self._last_failure_time = datetime.utcnow()
        self._success_count = 0
        
        if self._failure_count >= self.config.failure_threshold:
            self._state = CircuitState.OPEN
    
    def _should_attempt_reset(self) -> bool:
        """Vérifie si le circuit devrait passer en half-open"""
        if self._last_failure_time is None:
            return True
        
        elapsed = (datetime.utcnow() - self._last_failure_time).total_seconds()
        return elapsed >= self.config.timeout
    
    def _reset(self):
        """Réinitialise le circuit"""
        self._state = CircuitState.CLOSED
        self._failure_count = 0
        self._success_count = 0
        self._half_open_calls = 0


class CircuitOpenError(Exception):
    """Erreur levée quand le circuit est ouvert"""
    pass
```

### Dead Letter Queue et Traitement des Poisons

Les messages qui ne peuvent pas être traités après plusieurs tentatives sont dirigés vers une Dead Letter Queue (DLQ). Cette file d'attente permet d'isoler les messages problématiques sans bloquer le traitement des autres messages, tout en préservant la possibilité d'analyse et de retraitement manuel.

```python
# resilience/dlq.py
from dataclasses import dataclass
from typing import Optional, Dict, Any
from datetime import datetime
import json

@dataclass
class DeadLetter:
    """Message dans la Dead Letter Queue"""
    original_topic: str
    original_key: str
    original_value: bytes
    original_headers: Dict[str, bytes]
    error_type: str
    error_message: str
    stack_trace: Optional[str]
    retry_count: int
    first_failure: datetime
    last_failure: datetime
    consumer_group: str

class DeadLetterHandler:
    """Gestionnaire de Dead Letter Queue"""
    
    def __init__(self, producer, config: dict):
        self.producer = producer
        self.dlq_topic = config.get('dlq_topic', 'dead-letter-queue')
        self.max_retries = config.get('max_retries', 3)
    
    async def send_to_dlq(
        self,
        message,
        error: Exception,
        retry_count: int,
        consumer_group: str
    ):
        """Envoie un message vers la DLQ"""
        
        dead_letter = DeadLetter(
            original_topic=message.topic(),
            original_key=message.key().decode() if message.key() else "",
            original_value=message.value(),
            original_headers={
                k: v for k, v in (message.headers() or [])
            },
            error_type=type(error).__name__,
            error_message=str(error),
            stack_trace=self._get_stack_trace(error),
            retry_count=retry_count,
            first_failure=datetime.utcnow(),
            last_failure=datetime.utcnow(),
            consumer_group=consumer_group
        )
        
        self.producer.produce(
            topic=self.dlq_topic,
            key=f"{dead_letter.original_topic}:{dead_letter.original_key}".encode(),
            value=json.dumps({
                'original_topic': dead_letter.original_topic,
                'original_key': dead_letter.original_key,
                'original_value': dead_letter.original_value.decode('utf-8', errors='replace'),
                'error_type': dead_letter.error_type,
                'error_message': dead_letter.error_message,
                'retry_count': dead_letter.retry_count,
                'first_failure': dead_letter.first_failure.isoformat(),
                'last_failure': dead_letter.last_failure.isoformat(),
                'consumer_group': dead_letter.consumer_group
            }).encode(),
            headers=[
                ('dlq_reason', dead_letter.error_type.encode()),
                ('original_topic', dead_letter.original_topic.encode()),
                ('retry_count', str(dead_letter.retry_count).encode())
            ]
        )
        self.producer.flush()
    
    def _get_stack_trace(self, error: Exception) -> Optional[str]:
        """Extrait la stack trace d'une exception"""
        import traceback
        return ''.join(traceback.format_exception(
            type(error), error, error.__traceback__
        ))


class ResilientConsumer:
    """Consumer Kafka avec gestion complète de la résilience"""
    
    def __init__(
        self,
        consumer,
        handler: Callable,
        retry_handler: RetryHandler,
        circuit_breaker: CircuitBreaker,
        dlq_handler: DeadLetterHandler,
        config: dict
    ):
        self.consumer = consumer
        self.handler = handler
        self.retry = retry_handler
        self.circuit = circuit_breaker
        self.dlq = dlq_handler
        self.config = config
        
        self.consumer_group = config.get('group_id', 'default')
        self._retry_counts: Dict[str, int] = {}
    
    async def process_message(self, message):
        """Traite un message avec toutes les protections de résilience"""
        
        message_id = self._get_message_id(message)
        retry_count = self._retry_counts.get(message_id, 0)
        
        try:
            # Protection circuit breaker
            await self.circuit.execute(
                # Retry avec backoff
                self.retry.execute,
                self.handler,
                message
            )
            
            # Succès - nettoyage
            self._retry_counts.pop(message_id, None)
            
        except CircuitOpenError:
            # Circuit ouvert - ne pas retenter, attendre
            raise
            
        except Exception as e:
            retry_count += 1
            self._retry_counts[message_id] = retry_count
            
            if retry_count >= self.config.get('max_retries', 3):
                # Épuisement des retries - DLQ
                await self.dlq.send_to_dlq(
                    message, e, retry_count, self.consumer_group
                )
                self._retry_counts.pop(message_id, None)
            else:
                # Relancer pour retry ultérieur
                raise
    
    def _get_message_id(self, message) -> str:
        """Génère un identifiant unique pour le message"""
        return f"{message.topic()}:{message.partition()}:{message.offset()}"
```

> **Note technique**  
> Configurez des alertes sur la croissance de la DLQ. Un nombre croissant de messages en DLQ peut indiquer un problème systémique nécessitant une intervention, comme un changement de format incompatible ou une défaillance d'un service dépendant.

---

## II.9.6 Intégration avec les Agents Cognitifs Vertex AI

### Orchestration Agent-Événement

L'intégration des patrons architecturaux avec les agents cognitifs Vertex AI crée une synergie puissante où les agents peuvent participer aux Sagas, consommer des vues CQRS optimisées, et générer des événements traçables. Cette section détaille les patterns d'intégration spécifiques au contexte agentique.

```python
# agents/event_aware_agent.py
from typing import Optional, Dict, Any, List
from dataclasses import dataclass
from datetime import datetime
import uuid

@dataclass
class AgentContext:
    """Contexte enrichi pour un agent cognitif"""
    agent_id: str
    saga_id: Optional[str] = None
    correlation_id: str = ""
    customer_context: Optional[Dict[str, Any]] = None
    conversation_history: List[Dict] = None
    
    def __post_init__(self):
        if not self.correlation_id:
            self.correlation_id = str(uuid.uuid4())
        if self.conversation_history is None:
            self.conversation_history = []


class EventAwareAgent:
    """Agent cognitif intégré avec l'architecture événementielle"""
    
    def __init__(
        self,
        agent_id: str,
        vertex_client,
        event_publisher,
        read_model_client,
        saga_coordinator: Optional[SagaCoordinator] = None
    ):
        self.agent_id = agent_id
        self.vertex = vertex_client
        self.publisher = event_publisher
        self.read_model = read_model_client
        self.saga_coordinator = saga_coordinator
        
        self.model_name = "gemini-1.5-pro"
        self.system_instruction = self._build_system_instruction()
    
    def _build_system_instruction(self) -> str:
        """Construit les instructions système pour l'agent"""
        return """
        Tu es un agent spécialisé dans le traitement des demandes clients.
        Tu dois toujours:
        1. Utiliser le contexte client fourni pour personnaliser tes réponses
        2. Signaler clairement quand tu as besoin d'informations supplémentaires
        3. Proposer des actions concrètes et traçables
        4. Respecter les garde-fous définis dans la constitution
        """
    
    async def process_request(
        self,
        request: str,
        context: AgentContext
    ) -> Dict[str, Any]:
        """Traite une requête avec contexte complet"""
        
        # Enrichissement du contexte depuis les vues CQRS
        enriched_context = await self._enrich_context(context)
        
        # Publication de l'événement de début de traitement
        await self._publish_event(
            event_type="agent.processing.started",
            context=context,
            payload={
                'request': request,
                'customer_id': enriched_context.get('customer_id')
            }
        )
        
        try:
            # Appel au modèle Vertex AI
            response = await self._invoke_model(request, enriched_context)
            
            # Extraction des actions proposées
            actions = self._extract_actions(response)
            
            # Publication de l'événement de succès
            await self._publish_event(
                event_type="agent.processing.completed",
                context=context,
                payload={
                    'response_summary': response.get('summary'),
                    'actions_proposed': len(actions),
                    'confidence_score': response.get('confidence', 0.0)
                }
            )
            
            # Si dans une Saga, progression de l'état
            if context.saga_id and self.saga_coordinator:
                await self._progress_saga(context, actions)
            
            return {
                'response': response,
                'actions': actions,
                'context': enriched_context,
                'trace_id': context.correlation_id
            }
            
        except Exception as e:
            # Publication de l'événement d'échec
            await self._publish_event(
                event_type="agent.processing.failed",
                context=context,
                payload={
                    'error_type': type(e).__name__,
                    'error_message': str(e)
                }
            )
            raise
    
    async def _enrich_context(self, context: AgentContext) -> Dict[str, Any]:
        """Enrichit le contexte avec les données des vues CQRS"""
        
        enriched = {
            'agent_id': self.agent_id,
            'correlation_id': context.correlation_id,
            'timestamp': datetime.utcnow().isoformat()
        }
        
        # Chargement de la vue 360° client si disponible
        if context.customer_context and 'customer_id' in context.customer_context:
            customer_id = context.customer_context['customer_id']
            customer_360 = await self.read_model.get(
                'customer_360',
                customer_id
            )
            
            if customer_360:
                enriched['customer'] = {
                    'profile': customer_360.get('profile', {}),
                    'sentiment_trend': customer_360.get('interactions', {}).get('sentiment_trend'),
                    'open_tickets': customer_360.get('support', {}).get('open_tickets', 0),
                    'total_orders': customer_360.get('orders', {}).get('total', 0),
                    'context_summary': customer_360.get('context', {})
                }
        
        # Contexte de Saga si applicable
        if context.saga_id:
            enriched['saga'] = {
                'saga_id': context.saga_id,
                'in_transaction': True
            }
        
        return enriched
    
    async def _invoke_model(
        self,
        request: str,
        context: Dict[str, Any]
    ) -> Dict[str, Any]:
        """Invoque le modèle Vertex AI avec le contexte"""
        
        # Construction du prompt enrichi
        prompt = self._build_prompt(request, context)
        
        # Appel à Vertex AI
        response = await self.vertex.generate_content(
            model=self.model_name,
            contents=[
                {"role": "user", "parts": [{"text": prompt}]}
            ],
            generation_config={
                "temperature": 0.2,
                "max_output_tokens": 2048
            },
            system_instruction=self.system_instruction
        )
        
        # Parsing de la réponse
        return self._parse_response(response)
    
    def _build_prompt(self, request: str, context: Dict[str, Any]) -> str:
        """Construit le prompt avec le contexte"""
        
        prompt_parts = [f"Requête: {request}\n"]
        
        if 'customer' in context:
            customer = context['customer']
            prompt_parts.append(f"""
Contexte client:
- Tendance de sentiment: {customer.get('sentiment_trend', 'inconnu')}
- Tickets ouverts: {customer.get('open_tickets', 0)}
- Nombre de commandes: {customer.get('total_orders', 0)}
""")
        
        if context.get('saga', {}).get('in_transaction'):
            prompt_parts.append("""
Note: Cette interaction fait partie d'une transaction en cours.
Assure-toi que tes recommandations sont réversibles si nécessaire.
""")
        
        return "\n".join(prompt_parts)
    
    def _extract_actions(self, response: Dict[str, Any]) -> List[Dict]:
        """Extrait les actions proposées de la réponse"""
        
        actions = []
        
        if 'proposed_actions' in response:
            for action in response['proposed_actions']:
                actions.append({
                    'action_id': str(uuid.uuid4()),
                    'type': action.get('type'),
                    'description': action.get('description'),
                    'requires_approval': action.get('requires_approval', False),
                    'reversible': action.get('reversible', True)
                })
        
        return actions
    
    async def _publish_event(
        self,
        event_type: str,
        context: AgentContext,
        payload: Dict[str, Any]
    ):
        """Publie un événement sur le backbone"""
        
        event = {
            'event_id': str(uuid.uuid4()),
            'event_type': event_type,
            'agent_id': self.agent_id,
            'correlation_id': context.correlation_id,
            'saga_id': context.saga_id,
            'payload': payload,
            'timestamp': datetime.utcnow().isoformat()
        }
        
        await self.publisher.publish('agent.events', event)
    
    async def _progress_saga(
        self,
        context: AgentContext,
        actions: List[Dict]
    ):
        """Progresse l'état de la Saga si applicable"""
        
        if not self.saga_coordinator:
            return
        
        # Détermination de l'événement de progression
        if any(a.get('requires_approval') for a in actions):
            event_type = "saga.step.pending_approval"
        else:
            event_type = "saga.step.completed"
        
        await self.saga_coordinator.progress(
            saga_id=context.saga_id,
            step=f"agent.{self.agent_id}",
            event_type=event_type,
            data={'actions': actions}
        )
    
    def _parse_response(self, vertex_response) -> Dict[str, Any]:
        """Parse la réponse Vertex AI"""
        
        text = vertex_response.candidates[0].content.parts[0].text
        
        # Extraction structurée (simplifié)
        return {
            'text': text,
            'summary': text[:200] if len(text) > 200 else text,
            'confidence': 0.85,  # À calculer selon le contexte
            'proposed_actions': []  # À extraire du texte
        }
```

### Agent Participant de Saga

Un agent peut également agir comme participant à part entière dans une Saga, répondant aux événements et émettant ses propres événements de domaine.

```python
# agents/saga_participant_agent.py
from typing import Dict, Any, Optional
from datetime import datetime
import json

class SagaParticipantAgent:
    """Agent agissant comme participant de Saga"""
    
    def __init__(
        self,
        agent_id: str,
        consumer,
        producer,
        vertex_client,
        config: dict
    ):
        self.agent_id = agent_id
        self.consumer = consumer
        self.producer = producer
        self.vertex = vertex_client
        
        self.input_topic = config.get('input_topic', f'agent.{agent_id}.requests')
        self.output_topic = config.get('output_topic', f'agent.{agent_id}.results')
        self.compensation_topic = config.get('compensation_topic', f'agent.{agent_id}.compensate')
        
        # État local pour compensations
        self._completed_tasks: Dict[str, Dict] = {}
    
    async def run(self):
        """Boucle principale du participant"""
        
        self.consumer.subscribe([
            self.input_topic,
            self.compensation_topic
        ])
        
        while True:
            msg = self.consumer.poll(1.0)
            
            if msg is None:
                continue
            
            if msg.error():
                continue
            
            event = json.loads(msg.value().decode())
            topic = msg.topic()
            
            if topic == self.input_topic:
                await self._handle_request(event)
            elif topic == self.compensation_topic:
                await self._handle_compensation(event)
    
    async def _handle_request(self, event: dict):
        """Traite une requête de Saga"""
        
        saga_id = event.get('saga_id')
        task_id = event.get('task_id', str(uuid.uuid4()))
        
        try:
            # Exécution de la tâche cognitive
            result = await self._execute_cognitive_task(event)
            
            # Stockage pour compensation potentielle
            self._completed_tasks[saga_id] = {
                'task_id': task_id,
                'original_event': event,
                'result': result,
                'timestamp': datetime.utcnow().isoformat()
            }
            
            # Publication du succès
            await self._publish_result(
                saga_id=saga_id,
                task_id=task_id,
                success=True,
                result=result
            )
            
        except Exception as e:
            # Publication de l'échec
            await self._publish_result(
                saga_id=saga_id,
                task_id=task_id,
                success=False,
                error=str(e)
            )
    
    async def _handle_compensation(self, event: dict):
        """Compense une tâche précédente"""
        
        saga_id = event.get('saga_id')
        
        if saga_id not in self._completed_tasks:
            # Rien à compenser
            return
        
        task_info = self._completed_tasks[saga_id]
        
        try:
            # Exécution de la compensation
            await self._execute_compensation(task_info)
            
            # Nettoyage
            del self._completed_tasks[saga_id]
            
            # Confirmation
            self._publish_event(
                topic='saga.compensations',
                event={
                    'saga_id': saga_id,
                    'agent_id': self.agent_id,
                    'event_type': 'agent.task.compensated',
                    'task_id': task_info['task_id'],
                    'timestamp': datetime.utcnow().isoformat()
                }
            )
            
        except Exception as e:
            # Échec de compensation - alerte critique
            self._publish_event(
                topic='saga.compensation.failures',
                event={
                    'saga_id': saga_id,
                    'agent_id': self.agent_id,
                    'error': str(e),
                    'requires_manual_intervention': True
                }
            )
    
    async def _execute_cognitive_task(self, event: dict) -> Dict[str, Any]:
        """Exécute la tâche cognitive demandée"""
        
        task_type = event.get('task_type')
        payload = event.get('payload', {})
        
        # Construction du prompt selon le type de tâche
        prompt = self._build_task_prompt(task_type, payload)
        
        # Appel Vertex AI
        response = await self.vertex.generate_content(
            model="gemini-1.5-pro",
            contents=[{"role": "user", "parts": [{"text": prompt}]}],
            generation_config={"temperature": 0.1}
        )
        
        return {
            'task_type': task_type,
            'output': response.candidates[0].content.parts[0].text,
            'processed_at': datetime.utcnow().isoformat()
        }
    
    async def _execute_compensation(self, task_info: dict):
        """Exécute la logique de compensation"""
        
        task_type = task_info['original_event'].get('task_type')
        
        # Logique de compensation selon le type
        # Dans le cas d'un agent, cela peut signifier:
        # - Annuler une recommandation
        # - Marquer une analyse comme invalide
        # - Notifier qu'une décision précédente est révoquée
        
        compensation_prompt = f"""
        Une tâche précédente doit être compensée/annulée.
        Type de tâche: {task_type}
        Résultat original: {task_info['result']}
        
        Génère un message de notification approprié pour informer
        que cette action est annulée dans le cadre d'une compensation de transaction.
        """
        
        await self.vertex.generate_content(
            model="gemini-1.5-pro",
            contents=[{"role": "user", "parts": [{"text": compensation_prompt}]}]
        )
    
    def _build_task_prompt(self, task_type: str, payload: dict) -> str:
        """Construit le prompt selon le type de tâche"""
        
        prompts = {
            'analyze_request': f"Analyse la demande suivante: {payload.get('request')}",
            'validate_documents': f"Valide les documents: {payload.get('documents')}",
            'generate_recommendation': f"Génère une recommandation pour: {payload.get('context')}",
            'assess_risk': f"Évalue le risque pour: {payload.get('scenario')}"
        }
        
        return prompts.get(task_type, f"Traite la tâche: {task_type} avec {payload}")
    
    async def _publish_result(
        self,
        saga_id: str,
        task_id: str,
        success: bool,
        result: Optional[Dict] = None,
        error: Optional[str] = None
    ):
        """Publie le résultat de la tâche"""
        
        event = {
            'saga_id': saga_id,
            'agent_id': self.agent_id,
            'task_id': task_id,
            'event_type': 'agent.task.completed' if success else 'agent.task.failed',
            'success': success,
            'timestamp': datetime.utcnow().isoformat()
        }
        
        if success:
            event['result'] = result
        else:
            event['error'] = error
        
        self._publish_event(self.output_topic, event)
    
    def _publish_event(self, topic: str, event: dict):
        """Publie un événement sur Kafka"""
        
        self.producer.produce(
            topic=topic,
            key=event.get('saga_id', '').encode(),
            value=json.dumps(event).encode(),
            headers=[
                ('agent_id', self.agent_id.encode()),
                ('event_type', event.get('event_type', '').encode())
            ]
        )
        self.producer.flush()
```

---

## II.9.7 Tests des Patrons Architecturaux

### Stratégies de Test pour l'Event Sourcing

Tester des systèmes basés sur l'Event Sourcing nécessite des approches spécifiques. Les tests doivent vérifier non seulement l'état final mais aussi la séquence d'événements produite. L'approche Given-When-Then est particulièrement adaptée.

```python
# tests/eventsourcing_tests.py
import pytest
from typing import List
from dataclasses import dataclass

@dataclass
class TestScenario:
    """Scénario de test pour Event Sourcing"""
    name: str
    given_events: List[dict]
    when_command: dict
    then_events: List[dict]
    then_state: dict


class EventSourcingTestHarness:
    """Harnais de test pour les agrégats Event-Sourced"""
    
    def __init__(self, aggregate_class, event_store=None):
        self.aggregate_class = aggregate_class
        self.event_store = event_store or InMemoryEventStore()
        self.published_events: List[dict] = []
    
    async def given(self, events: List[dict]):
        """Configure l'état initial avec des événements"""
        
        if not events:
            return
        
        aggregate_id = events[0].get('aggregate_id')
        
        for event in events:
            await self.event_store.append(
                aggregate_type=self.aggregate_class.__name__.lower(),
                aggregate_id=aggregate_id,
                events=[self._dict_to_event(event)],
                expected_version=-1
            )
    
    async def when(self, command: dict) -> 'EventSourcingTestHarness':
        """Exécute une commande"""
        
        aggregate_id = command.get('aggregate_id')
        command_type = command.get('type')
        
        # Chargement de l'agrégat
        aggregate = self.aggregate_class(aggregate_id)
        events = await self.event_store.get_events(
            self.aggregate_class.__name__.lower(),
            aggregate_id
        )
        aggregate.load_from_history(events)
        
        # Exécution de la commande
        method = getattr(aggregate, command_type)
        method(**command.get('params', {}))
        
        # Capture des événements produits
        self.published_events = [
            e.to_dict() for e in aggregate.pending_events
        ]
        
        # Sauvegarde
        await self.event_store.append(
            aggregate_type=self.aggregate_class.__name__.lower(),
            aggregate_id=aggregate_id,
            events=aggregate.pending_events,
            expected_version=aggregate.version - len(aggregate.pending_events)
        )
        
        return self
    
    def then_events_match(self, expected_events: List[dict]):
        """Vérifie les événements produits"""
        
        assert len(self.published_events) == len(expected_events), \
            f"Nombre d'événements: attendu {len(expected_events)}, obtenu {len(self.published_events)}"
        
        for i, (actual, expected) in enumerate(zip(self.published_events, expected_events)):
            assert actual['event_type'] == expected['event_type'], \
                f"Event {i}: type attendu {expected['event_type']}, obtenu {actual['event_type']}"
            
            for key, value in expected.get('payload', {}).items():
                assert actual.get(key) == value, \
                    f"Event {i}: {key} attendu {value}, obtenu {actual.get(key)}"
    
    async def then_state_equals(self, expected_state: dict):
        """Vérifie l'état final de l'agrégat"""
        
        if not self.published_events:
            return
        
        aggregate_id = self.published_events[0].get('aggregate_id')
        
        aggregate = self.aggregate_class(aggregate_id)
        events = await self.event_store.get_events(
            self.aggregate_class.__name__.lower(),
            aggregate_id
        )
        aggregate.load_from_history(events)
        
        for key, value in expected_state.items():
            assert getattr(aggregate, key) == value, \
                f"État: {key} attendu {value}, obtenu {getattr(aggregate, key)}"
    
    def _dict_to_event(self, event_dict: dict):
        """Convertit un dict en événement"""
        # Implémentation selon le mapping des types d'événements
        pass


class InMemoryEventStore:
    """Event Store en mémoire pour les tests"""
    
    def __init__(self):
        self.events: dict = {}  # aggregate_id -> List[events]
    
    async def append(self, aggregate_type, aggregate_id, events, expected_version):
        key = f"{aggregate_type}:{aggregate_id}"
        
        if key not in self.events:
            self.events[key] = []
        
        current_version = len(self.events[key]) - 1
        
        if expected_version >= 0 and current_version != expected_version:
            raise ConcurrencyError()
        
        for event in events:
            self.events[key].append(event)
    
    async def get_events(self, aggregate_type, aggregate_id, from_version=0):
        key = f"{aggregate_type}:{aggregate_id}"
        return self.events.get(key, [])[from_version:]


# Exemple de tests
class TestCustomerAggregate:
    """Tests pour l'agrégat Customer"""
    
    @pytest.fixture
    def harness(self):
        return EventSourcingTestHarness(CustomerAggregate)
    
    @pytest.mark.asyncio
    async def test_create_customer(self, harness):
        """Test de création d'un client"""
        
        await harness.when({
            'type': 'create',
            'aggregate_id': 'cust-123',
            'params': {
                'name': 'Jean Dupont',
                'email': 'jean@example.com'
            }
        })
        
        harness.then_events_match([{
            'event_type': 'customer.created',
            'payload': {
                'name': 'Jean Dupont',
                'email': 'jean@example.com'
            }
        }])
        
        await harness.then_state_equals({
            'name': 'Jean Dupont',
            'email': 'jean@example.com',
            'status': 'active'
        })
    
    @pytest.mark.asyncio
    async def test_update_existing_customer(self, harness):
        """Test de mise à jour d'un client existant"""
        
        await harness.given([{
            'event_type': 'customer.created',
            'aggregate_id': 'cust-123',
            'name': 'Jean Dupont',
            'email': 'jean@example.com'
        }])
        
        await harness.when({
            'type': 'update_contact',
            'aggregate_id': 'cust-123',
            'params': {
                'email': 'jean.dupont@example.com'
            }
        })
        
        harness.then_events_match([{
            'event_type': 'customer.contact.updated',
            'payload': {
                'changes': {'email': 'jean.dupont@example.com'}
            }
        }])
    
    @pytest.mark.asyncio
    async def test_deactivate_inactive_customer_fails(self, harness):
        """Test d'échec de désactivation d'un client déjà inactif"""
        
        await harness.given([
            {
                'event_type': 'customer.created',
                'aggregate_id': 'cust-123',
                'name': 'Jean Dupont',
                'email': 'jean@example.com'
            },
            {
                'event_type': 'customer.deactivated',
                'aggregate_id': 'cust-123',
                'reason': 'Demande client'
            }
        ])
        
        with pytest.raises(ValueError, match="n'est pas actif"):
            await harness.when({
                'type': 'deactivate',
                'aggregate_id': 'cust-123',
                'params': {'reason': 'Autre raison'}
            })
```

### Tests de Saga avec Simulation

Les tests de Saga nécessitent de simuler les interactions entre participants et de vérifier les compensations.

```python
# tests/saga_tests.py
import pytest
from typing import List, Dict
from dataclasses import dataclass, field
from datetime import datetime

@dataclass
class SagaTestContext:
    """Contexte de test pour les Sagas"""
    saga_id: str
    published_events: List[dict] = field(default_factory=list)
    consumed_events: List[dict] = field(default_factory=list)
    compensations_triggered: List[str] = field(default_factory=list)


class SagaTestHarness:
    """Harnais de test pour les Sagas Chorégraphiées"""
    
    def __init__(self):
        self.contexts: Dict[str, SagaTestContext] = {}
        self.participant_handlers: Dict[str, callable] = {}
        self.compensation_handlers: Dict[str, callable] = {}
    
    def register_participant(
        self,
        event_type: str,
        handler: callable,
        compensation_handler: callable = None
    ):
        """Enregistre un participant simulé"""
        
        self.participant_handlers[event_type] = handler
        if compensation_handler:
            self.compensation_handlers[event_type] = compensation_handler
    
    async def start_saga(self, saga_id: str, initial_event: dict) -> SagaTestContext:
        """Démarre une saga de test"""
        
        context = SagaTestContext(saga_id=saga_id)
        self.contexts[saga_id] = context
        
        initial_event['saga_id'] = saga_id
        initial_event['timestamp'] = datetime.utcnow().isoformat()
        
        await self._process_event(context, initial_event)
        
        return context
    
    async def _process_event(self, context: SagaTestContext, event: dict):
        """Traite un événement et propage la chaîne"""
        
        context.consumed_events.append(event)
        event_type = event.get('event_type')
        
        # Recherche du handler
        handler = self.participant_handlers.get(event_type)
        
        if handler:
            try:
                result_events = await handler(event)
                
                for result_event in result_events:
                    result_event['saga_id'] = context.saga_id
                    context.published_events.append(result_event)
                    
                    # Propagation récursive
                    await self._process_event(context, result_event)
                    
            except Exception as e:
                # Déclenchement des compensations
                await self._trigger_compensations(context, event_type, str(e))
    
    async def _trigger_compensations(
        self,
        context: SagaTestContext,
        failed_step: str,
        error: str
    ):
        """Déclenche les compensations en ordre inverse"""
        
        # Identification des étapes complétées
        completed_types = [
            e['event_type'] for e in context.consumed_events
            if e['event_type'].endswith('.completed')
        ]
        
        # Compensation en ordre inverse
        for event_type in reversed(completed_types):
            base_type = event_type.replace('.completed', '')
            compensation_handler = self.compensation_handlers.get(base_type)
            
            if compensation_handler:
                context.compensations_triggered.append(base_type)
                await compensation_handler({'saga_id': context.saga_id})
    
    def assert_saga_completed(self, context: SagaTestContext):
        """Vérifie que la saga s'est terminée avec succès"""
        
        final_events = [
            e for e in context.published_events
            if e['event_type'].endswith('.approved') or 
               e['event_type'].endswith('.completed')
        ]
        
        assert len(final_events) > 0, "Aucun événement de complétion trouvé"
        assert len(context.compensations_triggered) == 0, \
            f"Des compensations ont été déclenchées: {context.compensations_triggered}"
    
    def assert_saga_compensated(
        self,
        context: SagaTestContext,
        expected_compensations: List[str]
    ):
        """Vérifie que la saga a été compensée correctement"""
        
        assert set(context.compensations_triggered) == set(expected_compensations), \
            f"Compensations attendues: {expected_compensations}, obtenues: {context.compensations_triggered}"


class TestLoanApplicationSaga:
    """Tests pour la Saga de demande de prêt"""
    
    @pytest.fixture
    def harness(self):
        h = SagaTestHarness()
        
        # Participant: Vérification de crédit
        async def credit_check_handler(event):
            return [{
                'event_type': 'credit.check.completed',
                'credit_score': 750,
                'approved': True
            }]
        
        async def credit_check_compensate(event):
            pass  # Pas de compensation nécessaire
        
        # Participant: Vérification documents
        async def doc_verification_handler(event):
            return [{
                'event_type': 'document.verification.completed',
                'verified': True
            }]
        
        # Participant: Décision finale
        async def decision_handler(event):
            return [{
                'event_type': 'loan.approved',
                'amount': 50000,
                'rate': 5.5
            }]
        
        h.register_participant(
            'loan.application.started',
            credit_check_handler,
            credit_check_compensate
        )
        h.register_participant(
            'credit.check.completed',
            doc_verification_handler
        )
        h.register_participant(
            'document.verification.completed',
            decision_handler
        )
        
        return h
    
    @pytest.mark.asyncio
    async def test_successful_loan_application(self, harness):
        """Test d'une demande de prêt réussie"""
        
        context = await harness.start_saga('saga-001', {
            'event_type': 'loan.application.started',
            'applicant_id': 'app-123',
            'amount': 50000
        })
        
        harness.assert_saga_completed(context)
        
        # Vérification des événements produits
        event_types = [e['event_type'] for e in context.published_events]
        assert 'credit.check.completed' in event_types
        assert 'document.verification.completed' in event_types
        assert 'loan.approved' in event_types
    
    @pytest.mark.asyncio
    async def test_loan_rejected_triggers_compensation(self):
        """Test de rejet avec compensation"""
        
        harness = SagaTestHarness()
        
        # Credit check OK
        async def credit_ok(event):
            return [{'event_type': 'credit.check.completed', 'approved': True}]
        
        # Doc verification échoue
        async def doc_fails(event):
            raise ValueError("Documents invalides")
        
        async def credit_compensate(event):
            pass
        
        harness.register_participant(
            'loan.application.started',
            credit_ok,
            credit_compensate
        )
        harness.register_participant(
            'credit.check.completed',
            doc_fails
        )
        
        context = await harness.start_saga('saga-002', {
            'event_type': 'loan.application.started',
            'applicant_id': 'app-456'
        })
        
        harness.assert_saga_compensated(
            context,
            expected_compensations=['loan.application.started']
        )
```

---

## II.9.8 Métriques et Observabilité des Patrons

### Indicateurs Clés de Performance

Chaque patron architectural génère des métriques spécifiques qui doivent être surveillées pour garantir la santé du système.

```python
# observability/pattern_metrics.py
from prometheus_client import Counter, Histogram, Gauge
from typing import Dict, Any
from datetime import datetime

# Métriques Saga
saga_started_total = Counter(
    'saga_started_total',
    'Nombre total de Sagas démarrées',
    ['saga_type']
)

saga_completed_total = Counter(
    'saga_completed_total',
    'Nombre de Sagas complétées avec succès',
    ['saga_type']
)

saga_compensated_total = Counter(
    'saga_compensated_total',
    'Nombre de Sagas compensées',
    ['saga_type', 'failed_step']
)

saga_duration_seconds = Histogram(
    'saga_duration_seconds',
    'Durée des Sagas en secondes',
    ['saga_type', 'outcome'],
    buckets=[1, 5, 10, 30, 60, 120, 300, 600]
)

saga_active_count = Gauge(
    'saga_active_count',
    'Nombre de Sagas actuellement en cours',
    ['saga_type']
)

# Métriques Event Sourcing
events_appended_total = Counter(
    'events_appended_total',
    'Nombre total d\'événements ajoutés',
    ['aggregate_type', 'event_type']
)

aggregate_load_duration_seconds = Histogram(
    'aggregate_load_duration_seconds',
    'Temps de chargement des agrégats',
    ['aggregate_type', 'from_snapshot'],
    buckets=[0.01, 0.05, 0.1, 0.5, 1, 2, 5]
)

snapshot_created_total = Counter(
    'snapshot_created_total',
    'Nombre de snapshots créés',
    ['aggregate_type']
)

events_since_snapshot = Histogram(
    'events_since_snapshot',
    'Nombre d\'événements rejoués depuis le snapshot',
    ['aggregate_type'],
    buckets=[0, 10, 50, 100, 500, 1000]
)

# Métriques CQRS
projection_lag_seconds = Gauge(
    'projection_lag_seconds',
    'Retard de la projection par rapport au write model',
    ['projector_name']
)

projection_events_processed = Counter(
    'projection_events_processed_total',
    'Événements traités par les projecteurs',
    ['projector_name', 'event_type']
)

read_model_queries_total = Counter(
    'read_model_queries_total',
    'Requêtes sur les modèles de lecture',
    ['view_name']
)

read_model_query_duration = Histogram(
    'read_model_query_duration_seconds',
    'Durée des requêtes read model',
    ['view_name'],
    buckets=[0.001, 0.005, 0.01, 0.05, 0.1, 0.5]
)

# Métriques Outbox
outbox_pending_count = Gauge(
    'outbox_pending_count',
    'Nombre de messages en attente dans l\'outbox'
)

outbox_publish_duration = Histogram(
    'outbox_publish_duration_seconds',
    'Temps de publication depuis l\'outbox',
    buckets=[0.01, 0.05, 0.1, 0.5, 1, 5]
)

outbox_failures_total = Counter(
    'outbox_failures_total',
    'Échecs de publication depuis l\'outbox',
    ['error_type']
)

# Métriques Résilience
circuit_breaker_state = Gauge(
    'circuit_breaker_state',
    'État du circuit breaker (0=closed, 1=half-open, 2=open)',
    ['circuit_name']
)

retry_attempts_total = Counter(
    'retry_attempts_total',
    'Nombre de tentatives de retry',
    ['operation', 'attempt_number']
)

dlq_messages_total = Counter(
    'dlq_messages_total',
    'Messages envoyés en DLQ',
    ['source_topic', 'error_type']
)


class PatternMetricsCollector:
    """Collecteur centralisé des métriques des patrons"""
    
    def __init__(self, registry=None):
        self.registry = registry
        self._saga_start_times: Dict[str, datetime] = {}
    
    # Saga
    
    def saga_started(self, saga_type: str, saga_id: str):
        saga_started_total.labels(saga_type=saga_type).inc()
        saga_active_count.labels(saga_type=saga_type).inc()
        self._saga_start_times[saga_id] = datetime.utcnow()
    
    def saga_completed(self, saga_type: str, saga_id: str):
        saga_completed_total.labels(saga_type=saga_type).inc()
        saga_active_count.labels(saga_type=saga_type).dec()
        self._record_saga_duration(saga_type, saga_id, 'completed')
    
    def saga_compensated(self, saga_type: str, saga_id: str, failed_step: str):
        saga_compensated_total.labels(
            saga_type=saga_type,
            failed_step=failed_step
        ).inc()
        saga_active_count.labels(saga_type=saga_type).dec()
        self._record_saga_duration(saga_type, saga_id, 'compensated')
    
    def _record_saga_duration(self, saga_type: str, saga_id: str, outcome: str):
        if saga_id in self._saga_start_times:
            duration = (datetime.utcnow() - self._saga_start_times[saga_id]).total_seconds()
            saga_duration_seconds.labels(
                saga_type=saga_type,
                outcome=outcome
            ).observe(duration)
            del self._saga_start_times[saga_id]
    
    # Event Sourcing
    
    def event_appended(self, aggregate_type: str, event_type: str):
        events_appended_total.labels(
            aggregate_type=aggregate_type,
            event_type=event_type
        ).inc()
    
    def aggregate_loaded(
        self,
        aggregate_type: str,
        duration: float,
        from_snapshot: bool,
        events_replayed: int
    ):
        aggregate_load_duration_seconds.labels(
            aggregate_type=aggregate_type,
            from_snapshot=str(from_snapshot)
        ).observe(duration)
        
        if from_snapshot:
            events_since_snapshot.labels(
                aggregate_type=aggregate_type
            ).observe(events_replayed)
    
    def snapshot_created(self, aggregate_type: str):
        snapshot_created_total.labels(aggregate_type=aggregate_type).inc()
    
    # CQRS
    
    def projection_event_processed(self, projector_name: str, event_type: str):
        projection_events_processed.labels(
            projector_name=projector_name,
            event_type=event_type
        ).inc()
    
    def projection_lag_updated(self, projector_name: str, lag_seconds: float):
        projection_lag_seconds.labels(projector_name=projector_name).set(lag_seconds)
    
    def read_model_query(self, view_name: str, duration: float):
        read_model_queries_total.labels(view_name=view_name).inc()
        read_model_query_duration.labels(view_name=view_name).observe(duration)
    
    # Outbox
    
    def outbox_pending_updated(self, count: int):
        outbox_pending_count.set(count)
    
    def outbox_published(self, duration: float):
        outbox_publish_duration.observe(duration)
    
    def outbox_failed(self, error_type: str):
        outbox_failures_total.labels(error_type=error_type).inc()
    
    # Résilience
    
    def circuit_state_changed(self, circuit_name: str, state: CircuitState):
        state_value = {'closed': 0, 'half_open': 1, 'open': 2}.get(state.value, 0)
        circuit_breaker_state.labels(circuit_name=circuit_name).set(state_value)
    
    def retry_attempted(self, operation: str, attempt: int):
        retry_attempts_total.labels(
            operation=operation,
            attempt_number=str(attempt)
        ).inc()
    
    def dlq_message_sent(self, source_topic: str, error_type: str):
        dlq_messages_total.labels(
            source_topic=source_topic,
            error_type=error_type
        ).inc()
```

### Dashboard Grafana pour les Patrons

Configuration d'un dashboard Grafana pour visualiser la santé des patrons architecturaux :

```json
{
  "dashboard": {
    "title": "Patrons Architecturaux - Santé du Système",
    "panels": [
      {
        "title": "Sagas Actives",
        "type": "stat",
        "targets": [{
          "expr": "sum(saga_active_count)",
          "legendFormat": "Sagas en cours"
        }]
      },
      {
        "title": "Taux de Succès des Sagas",
        "type": "gauge",
        "targets": [{
          "expr": "sum(rate(saga_completed_total[5m])) / sum(rate(saga_started_total[5m])) * 100"
        }],
        "fieldConfig": {
          "defaults": {
            "thresholds": {
              "steps": [
                {"color": "red", "value": 0},
                {"color": "yellow", "value": 90},
                {"color": "green", "value": 98}
              ]
            },
            "unit": "percent"
          }
        }
      },
      {
        "title": "Durée des Sagas (P95)",
        "type": "timeseries",
        "targets": [{
          "expr": "histogram_quantile(0.95, sum(rate(saga_duration_seconds_bucket[5m])) by (le, saga_type))",
          "legendFormat": "{{saga_type}}"
        }]
      },
      {
        "title": "Lag des Projections",
        "type": "timeseries",
        "targets": [{
          "expr": "projection_lag_seconds",
          "legendFormat": "{{projector_name}}"
        }],
        "fieldConfig": {
          "defaults": {
            "thresholds": {
              "steps": [
                {"color": "green", "value": 0},
                {"color": "yellow", "value": 30},
                {"color": "red", "value": 60}
              ]
            }
          }
        }
      },
      {
        "title": "Temps de Chargement des Agrégats",
        "type": "heatmap",
        "targets": [{
          "expr": "sum(rate(aggregate_load_duration_seconds_bucket[5m])) by (le)",
          "format": "heatmap"
        }]
      },
      {
        "title": "Messages Outbox en Attente",
        "type": "stat",
        "targets": [{
          "expr": "outbox_pending_count"
        }],
        "fieldConfig": {
          "defaults": {
            "thresholds": {
              "steps": [
                {"color": "green", "value": 0},
                {"color": "yellow", "value": 100},
                {"color": "red", "value": 1000}
              ]
            }
          }
        }
      },
      {
        "title": "État des Circuit Breakers",
        "type": "table",
        "targets": [{
          "expr": "circuit_breaker_state",
          "format": "table"
        }],
        "transformations": [{
          "id": "organize",
          "options": {
            "renameByName": {
              "circuit_name": "Circuit",
              "Value": "État"
            }
          }
        }]
      },
      {
        "title": "Messages DLQ (24h)",
        "type": "timeseries",
        "targets": [{
          "expr": "sum(increase(dlq_messages_total[1h])) by (source_topic)",
          "legendFormat": "{{source_topic}}"
        }]
      }
    ]
  }
}
```

---

## II.9.9 Étude de Cas : Système de Gestion de Commandes Agentique

### Contexte et Exigences

Cette étude de cas illustre l'intégration complète des patrons dans un système réel de gestion de commandes où des agents cognitifs participent au traitement. Le système doit gérer des commandes clients impliquant vérification de stock, validation de paiement, préparation logistique, et notifications — le tout avec une traçabilité complète et une capacité de compensation.

```
Architecture de référence :

┌─────────────────────────────────────────────────────────────────┐
│                    COUCHE PRÉSENTATION                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────────────────┐  │
│  │ API REST │  │ WebSocket│  │ Agent Conversationnel        │  │
│  └────┬─────┘  └────┬─────┘  │ (Vertex AI)                  │  │
│       │             │        └──────────────┬───────────────┘  │
└───────┼─────────────┼───────────────────────┼──────────────────┘
        │             │                       │
┌───────▼─────────────▼───────────────────────▼──────────────────┐
│                    COUCHE COMMAND (Write)                       │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                  Command Handlers                         │  │
│  │  ┌────────────┐ ┌────────────┐ ┌─────────────────────┐   │  │
│  │  │ Order Cmd  │ │ Payment Cmd│ │ Inventory Cmd       │   │  │
│  │  └─────┬──────┘ └──────┬─────┘ └──────────┬──────────┘   │  │
│  └────────┼───────────────┼──────────────────┼──────────────┘  │
│           │               │                  │                  │
│  ┌────────▼───────────────▼──────────────────▼──────────────┐  │
│  │                    Event Store + Outbox                   │  │
│  │              (PostgreSQL + Transactional Outbox)          │  │
│  └────────────────────────────┬─────────────────────────────┘  │
└───────────────────────────────┼─────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────┐
│                    BACKBONE ÉVÉNEMENTIEL (Kafka)                │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ Topics: orders.events, payments.events, inventory.events │  │
│  │         saga.events, agent.events, dlq                   │  │
│  └──────────────────────────────────────────────────────────┘  │
└───────────────────────────────┬─────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────┐
│                    COUCHE QUERY (Read)                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │ Order View   │  │ Customer 360 │  │ Analytics View       │  │
│  │ (MongoDB)    │  │ (Redis)      │  │ (ClickHouse)         │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Implémentation Complète

```python
# order_system/domain/order_aggregate.py
from dataclasses import dataclass, field
from typing import List, Optional, Dict, Any
from datetime import datetime
from enum import Enum
import uuid

class OrderStatus(Enum):
    CREATED = "created"
    VALIDATED = "validated"
    PAYMENT_PENDING = "payment_pending"
    PAYMENT_CONFIRMED = "payment_confirmed"
    PREPARING = "preparing"
    SHIPPED = "shipped"
    DELIVERED = "delivered"
    CANCELLED = "cancelled"

@dataclass
class OrderItem:
    product_id: str
    quantity: int
    unit_price: float
    
    @property
    def total(self) -> float:
        return self.quantity * self.unit_price

@dataclass
class OrderAggregate(AggregateRoot):
    """Agrégat Order avec Event Sourcing complet"""
    
    customer_id: str = ""
    items: List[OrderItem] = field(default_factory=list)
    status: OrderStatus = OrderStatus.CREATED
    shipping_address: Dict[str, str] = field(default_factory=dict)
    payment_id: Optional[str] = None
    tracking_number: Optional[str] = None
    total_amount: float = 0.0
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None
    
    def __init__(self, order_id: str):
        super().__init__(order_id)
    
    def _get_aggregate_type(self) -> str:
        return "order"
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            'order_id': self.id,
            'customer_id': self.customer_id,
            'items': [
                {'product_id': i.product_id, 'quantity': i.quantity, 'unit_price': i.unit_price}
                for i in self.items
            ],
            'status': self.status.value,
            'shipping_address': self.shipping_address,
            'payment_id': self.payment_id,
            'tracking_number': self.tracking_number,
            'total_amount': self.total_amount
        }
    
    def restore_from_snapshot(self, state: Dict[str, Any]):
        self.customer_id = state['customer_id']
        self.items = [
            OrderItem(**item) for item in state.get('items', [])
        ]
        self.status = OrderStatus(state['status'])
        self.shipping_address = state.get('shipping_address', {})
        self.payment_id = state.get('payment_id')
        self.tracking_number = state.get('tracking_number')
        self.total_amount = state.get('total_amount', 0.0)
    
    # Commands
    
    @classmethod
    def create(
        cls,
        order_id: str,
        customer_id: str,
        items: List[Dict],
        shipping_address: Dict[str, str]
    ) -> 'OrderAggregate':
        """Crée une nouvelle commande"""
        
        order = cls(order_id)
        
        order_items = [OrderItem(**item) for item in items]
        total = sum(item.total for item in order_items)
        
        order._raise_event(OrderCreatedEvent(
            event_id=str(uuid.uuid4()),
            aggregate_id=order_id,
            order_id=order_id,
            customer_id=customer_id,
            items=items,
            shipping_address=shipping_address,
            total_amount=total,
            timestamp=datetime.utcnow()
        ))
        
        return order
    
    def validate(self, validation_result: Dict[str, Any]):
        """Valide la commande après vérifications"""
        
        if self.status != OrderStatus.CREATED:
            raise ValueError(f"Impossible de valider une commande en statut {self.status}")
        
        if not validation_result.get('stock_available'):
            self._raise_event(OrderValidationFailedEvent(
                event_id=str(uuid.uuid4()),
                aggregate_id=self.id,
                reason="stock_unavailable",
                details=validation_result.get('unavailable_items', []),
                timestamp=datetime.utcnow()
            ))
            return
        
        self._raise_event(OrderValidatedEvent(
            event_id=str(uuid.uuid4()),
            aggregate_id=self.id,
            order_id=self.id,
            validation_details=validation_result,
            timestamp=datetime.utcnow()
        ))
    
    def confirm_payment(self, payment_id: str, transaction_details: Dict):
        """Confirme le paiement"""
        
        if self.status != OrderStatus.PAYMENT_PENDING:
            raise ValueError(f"Paiement non attendu en statut {self.status}")
        
        self._raise_event(PaymentConfirmedEvent(
            event_id=str(uuid.uuid4()),
            aggregate_id=self.id,
            order_id=self.id,
            payment_id=payment_id,
            transaction_details=transaction_details,
            timestamp=datetime.utcnow()
        ))
    
    def start_preparation(self, warehouse_id: str):
        """Démarre la préparation"""
        
        if self.status != OrderStatus.PAYMENT_CONFIRMED:
            raise ValueError(f"Préparation impossible en statut {self.status}")
        
        self._raise_event(PreparationStartedEvent(
            event_id=str(uuid.uuid4()),
            aggregate_id=self.id,
            order_id=self.id,
            warehouse_id=warehouse_id,
            timestamp=datetime.utcnow()
        ))
    
    def ship(self, carrier: str, tracking_number: str):
        """Expédie la commande"""
        
        if self.status != OrderStatus.PREPARING:
            raise ValueError(f"Expédition impossible en statut {self.status}")
        
        self._raise_event(OrderShippedEvent(
            event_id=str(uuid.uuid4()),
            aggregate_id=self.id,
            order_id=self.id,
            carrier=carrier,
            tracking_number=tracking_number,
            timestamp=datetime.utcnow()
        ))
    
    def cancel(self, reason: str, cancelled_by: str):
        """Annule la commande"""
        
        if self.status in [OrderStatus.SHIPPED, OrderStatus.DELIVERED]:
            raise ValueError("Impossible d'annuler une commande expédiée ou livrée")
        
        self._raise_event(OrderCancelledEvent(
            event_id=str(uuid.uuid4()),
            aggregate_id=self.id,
            order_id=self.id,
            reason=reason,
            cancelled_by=cancelled_by,
            previous_status=self.status.value,
            timestamp=datetime.utcnow()
        ))
    
    # Event Applicators
    
    def _apply_order_created(self, data: dict):
        self.customer_id = data['customer_id']
        self.items = [OrderItem(**item) for item in data['items']]
        self.shipping_address = data['shipping_address']
        self.total_amount = data['total_amount']
        self.status = OrderStatus.CREATED
        self.created_at = datetime.fromisoformat(data['timestamp'])
    
    def _apply_order_validated(self, data: dict):
        self.status = OrderStatus.PAYMENT_PENDING
        self.updated_at = datetime.fromisoformat(data['timestamp'])
    
    def _apply_order_validation_failed(self, data: dict):
        self.status = OrderStatus.CANCELLED
        self.updated_at = datetime.fromisoformat(data['timestamp'])
    
    def _apply_payment_confirmed(self, data: dict):
        self.payment_id = data['payment_id']
        self.status = OrderStatus.PAYMENT_CONFIRMED
        self.updated_at = datetime.fromisoformat(data['timestamp'])
    
    def _apply_preparation_started(self, data: dict):
        self.status = OrderStatus.PREPARING
        self.updated_at = datetime.fromisoformat(data['timestamp'])
    
    def _apply_order_shipped(self, data: dict):
        self.tracking_number = data['tracking_number']
        self.status = OrderStatus.SHIPPED
        self.updated_at = datetime.fromisoformat(data['timestamp'])
    
    def _apply_order_cancelled(self, data: dict):
        self.status = OrderStatus.CANCELLED
        self.updated_at = datetime.fromisoformat(data['timestamp'])


# order_system/saga/order_saga.py
class OrderProcessingSaga:
    """Saga de traitement de commande complète"""
    
    STEPS = [
        'order.created',
        'inventory.reserved',
        'payment.processed',
        'order.validated',
        'preparation.started',
        'order.shipped'
    ]
    
    COMPENSATION_MAP = {
        'inventory.reserved': 'inventory.release',
        'payment.processed': 'payment.refund',
        'preparation.started': 'preparation.cancel'
    }
    
    def __init__(self, event_store, outbox_service, metrics_collector):
        self.event_store = event_store
        self.outbox = outbox_service
        self.metrics = metrics_collector
    
    async def start(self, order_id: str, order_data: dict) -> str:
        """Démarre la saga de traitement"""
        
        saga_id = f"order-saga-{order_id}-{uuid.uuid4().hex[:8]}"
        
        # Création de l'événement initial
        async with self.outbox.transaction() as tx:
            # Sauvegarde de l'état initial de la saga
            await tx.execute("""
                INSERT INTO saga_state (saga_id, saga_type, status, current_step, data)
                VALUES ($1, $2, $3, $4, $5)
            """, saga_id, 'order_processing', 'started', 'order.created', 
                json.dumps(order_data))
            
            # Événement de démarrage
            await tx.add_event(
                aggregate_type='saga',
                aggregate_id=saga_id,
                event_type='order.saga.started',
                payload={
                    'saga_id': saga_id,
                    'order_id': order_id,
                    'order_data': order_data
                }
            )
            
            # Première étape: réservation inventaire
            await tx.add_event(
                aggregate_type='inventory',
                aggregate_id=order_id,
                event_type='inventory.reservation.requested',
                payload={
                    'saga_id': saga_id,
                    'order_id': order_id,
                    'items': order_data['items']
                },
                metadata={'saga_id': saga_id}
            )
        
        self.metrics.saga_started('order_processing', saga_id)
        
        return saga_id
    
    async def handle_step_completed(self, saga_id: str, step: str, result: dict):
        """Gère la complétion d'une étape"""
        
        async with self.outbox.transaction() as tx:
            # Mise à jour de l'état
            await tx.execute("""
                UPDATE saga_state 
                SET current_step = $2, 
                    completed_steps = array_append(completed_steps, $2),
                    updated_at = NOW()
                WHERE saga_id = $1
            """, saga_id, step)
            
            # Détermination de l'étape suivante
            next_step = self._get_next_step(step)
            
            if next_step:
                await self._trigger_next_step(tx, saga_id, next_step, result)
            else:
                # Saga terminée
                await self._complete_saga(tx, saga_id)
    
    async def handle_step_failed(self, saga_id: str, failed_step: str, error: str):
        """Gère l'échec d'une étape et déclenche les compensations"""
        
        async with self.outbox.transaction() as tx:
            # Récupération des étapes complétées
            row = await tx.fetch("""
                SELECT completed_steps, data FROM saga_state WHERE saga_id = $1
            """, saga_id)
            
            completed_steps = row[0]['completed_steps'] or []
            saga_data = json.loads(row[0]['data'])
            
            # Mise à jour du statut
            await tx.execute("""
                UPDATE saga_state 
                SET status = 'compensating', 
                    failed_step = $2,
                    error = $3,
                    updated_at = NOW()
                WHERE saga_id = $1
            """, saga_id, failed_step, error)
            
            # Déclenchement des compensations en ordre inverse
            for step in reversed(completed_steps):
                compensation = self.COMPENSATION_MAP.get(step)
                if compensation:
                    await tx.add_event(
                        aggregate_type='saga',
                        aggregate_id=saga_id,
                        event_type=f'{compensation}.requested',
                        payload={
                            'saga_id': saga_id,
                            'original_step': step,
                            'saga_data': saga_data
                        }
                    )
        
        self.metrics.saga_compensated('order_processing', saga_id, failed_step)
    
    def _get_next_step(self, current_step: str) -> Optional[str]:
        """Détermine l'étape suivante"""
        try:
            idx = self.STEPS.index(current_step)
            if idx < len(self.STEPS) - 1:
                return self.STEPS[idx + 1]
        except ValueError:
            pass
        return None
    
    async def _trigger_next_step(self, tx, saga_id: str, step: str, context: dict):
        """Déclenche l'étape suivante"""
        
        step_events = {
            'inventory.reserved': ('payment', 'payment.process.requested'),
            'payment.processed': ('order', 'order.validation.requested'),
            'order.validated': ('warehouse', 'preparation.start.requested'),
            'preparation.started': ('shipping', 'shipment.create.requested')
        }
        
        if step in step_events:
            aggregate_type, event_type = step_events[step]
            await tx.add_event(
                aggregate_type=aggregate_type,
                aggregate_id=context.get('order_id'),
                event_type=event_type,
                payload={
                    'saga_id': saga_id,
                    **context
                }
            )
    
    async def _complete_saga(self, tx, saga_id: str):
        """Finalise la saga avec succès"""
        
        await tx.execute("""
            UPDATE saga_state 
            SET status = 'completed', 
                completed_at = NOW()
            WHERE saga_id = $1
        """, saga_id)
        
        await tx.add_event(
            aggregate_type='saga',
            aggregate_id=saga_id,
            event_type='order.saga.completed',
            payload={'saga_id': saga_id}
        )
        
        self.metrics.saga_completed('order_processing', saga_id)
```

Cette étude de cas démontre comment les patrons s'intègrent dans un système réel, avec l'Event Sourcing pour la traçabilité des commandes, CQRS pour les vues optimisées, les Sagas pour la coordination des services, et l'Outbox pour la fiabilité des publications.

---

## Conclusion

Les patrons architecturaux présentés dans ce chapitre constituent les fondations techniques sur lesquelles repose la fiabilité des systèmes agentiques. La Saga Chorégraphiée permet d'orchestrer des transactions distribuées complexes tout en maintenant l'autonomie des participants. CQRS optimise les performances en séparant les flux de lecture et d'écriture, permettant de construire des vues adaptées aux besoins spécifiques de chaque agent. L'Event Sourcing offre une traçabilité complète et la capacité de reconstituer l'état du système à tout moment. Le patron Outbox Transactionnel garantit la cohérence entre les modifications de données et la publication des événements.

Ces patrons ne fonctionnent pas isolément mais se combinent naturellement. Une architecture robuste utilise typiquement l'Event Sourcing comme fondation, CQRS pour optimiser les accès, le patron Outbox pour garantir la publication des événements, et les Sagas pour coordonner les processus multi-étapes. Les mécanismes de résilience — retry, circuit breaker, dead letter queue — viennent compléter l'ensemble en assurant que le système peut absorber et récupérer des défaillances inévitables.

L'intégration avec les agents cognitifs Vertex AI ajoute une dimension supplémentaire où les agents peuvent participer aux Sagas comme des participants à part entière, consommer des vues CQRS optimisées pour leurs besoins décisionnels, et générer des événements traçables. Cette synergie entre architecture événementielle et intelligence artificielle constitue le cœur de l'Agentic Event Mesh.

Les stratégies de test présentées — harnais de test Given-When-Then pour l'Event Sourcing, simulation de Sagas avec compensation — permettent de valider le comportement de ces systèmes complexes de manière déterministe malgré leur nature distribuée et asynchrone.

La mise en œuvre de ces patrons exige une compréhension approfondie des compromis impliqués. L'Event Sourcing offre une traçabilité incomparable mais génère un volume de données important, nécessitant des mécanismes de snapshot et des politiques de rétention. CQRS simplifie les requêtes mais introduit une complexité dans la synchronisation des modèles et le monitoring du lag. Les Sagas permettent les transactions distribuées mais nécessitent une gestion rigoureuse des compensations et des timeouts. L'observabilité fine de ces patrons via des métriques Prometheus et des dashboards Grafana est essentielle pour maintenir la santé opérationnelle du système.

Le chapitre suivant explorera les pipelines CI/CD et les stratégies de déploiement qui permettent de mettre ces architectures en production de manière fiable et reproductible, en tenant compte des spécificités des systèmes agentiques.

---

## II.9.10 Résumé

**Saga Chorégraphiée.** Patron de coordination des transactions distribuées sans coordinateur central. Chaque participant réagit aux événements, exécute sa transaction locale et publie le résultat. Les compensations permettent d'annuler les effets en cas d'échec. Idéal pour les processus métier multi-étapes impliquant plusieurs agents ou services.

**Événements de Saga.** Quatre types principaux : commandes (initient une action), succès (confirment l'exécution), échecs (signalent un problème), compensations (annulent une action). Le saga_id corrèle tous les événements d'une même transaction distribuée.

**CQRS (Command Query Responsibility Segregation).** Séparation des opérations de lecture et d'écriture en modèles distincts. Le modèle de commande (write) est normalisé pour la cohérence, le modèle de lecture (read) est dénormalisé pour la performance. Synchronisation asynchrone via événements.

**Projecteurs.** Composants responsables de la transformation des événements en vues de lecture. Doivent être idempotents pour supporter le retraitement. Permettent de créer des vues spécialisées (profil client, vue 360° pour agents) sans modifier le modèle d'écriture.

**Event Sourcing.** Persistance de l'état sous forme de séquence d'événements plutôt que d'état courant. L'état est reconstitué en appliquant les événements dans l'ordre. Offre traçabilité complète, audit naturel, et capacité de time-travel (reconstitution à tout instant).

**Event Store.** Composant central de l'Event Sourcing. Garantit l'ordonnancement des événements par agrégat, l'atomicité des écritures avec verrouillage optimiste, et la lecture efficace de l'historique. Publication sur Kafka après persistance pour diffusion aux projecteurs.

**Snapshots.** Mécanisme d'optimisation pour l'Event Sourcing. Capture périodique de l'état complet d'un agrégat, permettant de ne rejouer que les événements postérieurs. Réduit considérablement le temps de chargement des agrégats avec un long historique.

**Projection Replay.** Capacité de reconstruire des projections à partir de l'historique complet des événements. Essentiel pour corriger des erreurs, ajouter de nouvelles vues, ou migrer vers de nouveaux schémas. Mécanisme de checkpoint pour reprendre les reconstructions interrompues.

**Agrégats Event-Sourced.** Entités dont l'état est reconstruit à partir des événements. Définissent les règles métier, valident les commandes, génèrent de nouveaux événements. La méthode apply reconstruit l'état, la méthode raise_event enregistre les changements.

**Outbox Transactionnel.** Résout le problème de la double écriture (base de données + broker). Les événements sont écrits dans une table outbox dans la même transaction que les modifications métier. Un relay (polling ou CDC/Debezium) publie ensuite sur Kafka.

**Retry avec Backoff Exponentiel.** Stratégie pour les erreurs transitoires. Délai croissant entre les tentatives (base × 2^attempt) avec plafond et jitter pour éviter les thundering herds. Configuration du nombre max de tentatives et des exceptions retryables.

**Circuit Breaker.** Protection contre les cascades de défaillances. Trois états : fermé (normal), ouvert (bloque les appels), semi-ouvert (teste la récupération). Transition basée sur le nombre d'échecs consécutifs et un timeout de récupération.

**Dead Letter Queue (DLQ).** File d'attente pour les messages qui ont épuisé leurs tentatives de retry. Isole les messages problématiques sans bloquer le traitement. Préserve les informations d'erreur et permet l'analyse et le retraitement manuel.

**Intégration Agent-Événement.** Les agents cognitifs peuvent participer aux Sagas comme participants, consommer des vues CQRS optimisées, et émettre des événements traçables. Le contexte enrichi permet aux agents de prendre des décisions informées basées sur l'état global du système.

**Agent Participant de Saga.** Pattern où un agent agit comme participant à part entière dans une transaction distribuée. L'agent réagit aux événements, exécute des tâches cognitives, et peut être compensé si nécessaire. Stockage local de l'état pour permettre la compensation.

**Tests Event Sourcing.** Approche Given-When-Then avec harnais de test spécialisé. Vérification non seulement de l'état final mais aussi de la séquence d'événements produite. Event Store en mémoire pour l'isolation des tests.

**Tests de Saga.** Simulation des interactions entre participants avec vérification des compensations. Enregistrement de handlers simulés pour chaque étape, validation du parcours complet ou de la récupération après échec.

**Métriques des Patrons.** Indicateurs spécifiques : durée et taux de succès des Sagas, temps de chargement des agrégats, lag des projections, taille de l'outbox, état des circuit breakers, messages en DLQ. Dashboard Grafana centralisé pour la supervision.

**Combinaison des patrons.** Architecture typique : Event Sourcing comme fondation avec snapshots pour l'optimisation, CQRS pour les vues spécialisées, Outbox pour la publication fiable, Sagas pour la coordination multi-services, et mécanismes de résilience pour la robustesse. L'étude de cas du système de commandes illustre cette intégration complète.

---

*Chapitre suivant : Chapitre II.10 — Pipelines CI/CD et Déploiement des Agents*
