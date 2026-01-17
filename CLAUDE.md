# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AgentMeshKafka is an **Enterprise Agentic Mesh** implementation using Apache Kafka and Anthropic Claude. It demonstrates a decentralized architecture where autonomous agents collaborate asynchronously for bank loan processing.

**Scenario:** Three agents process loan applications through validation → risk analysis → decision.

## Commands

### Setup
```bash
pip install -r phase4/requirements.txt
export ANTHROPIC_API_KEY=sk-ant-...
```

### Testing (Evaluation Diamond)
```bash
# L1 - Unit tests (deterministic Python)
pytest tests/unit/ -v

# L2 - Cognitive evaluation (LLM-Judge)
pytest tests/evaluation/ -v

# L3 - Adversarial testing (prompt injection, red team)
pytest tests/adversarial/ -v

# All tests with coverage
pytest --cov=phase4/src --cov-report=html

# Single test
pytest tests/unit/test_telemetry.py -v -k "test_pattern"
```

### Code Quality
```bash
black phase4/
ruff check phase4/
mypy phase4/src/
```

### Kafka (Phase 1+)
```bash
docker-compose up -d
python scripts/init_kafka.py
```

## Architecture

### Three-Pillar Design

1. **Nervous System (Communication):** Apache Kafka (KRaft mode) with Event Sourcing, CQRS
2. **Brain (Cognition):** ReAct-pattern agents powered by Claude (Haiku/Sonnet/Opus 4.5)
3. **Immune System (Security):** AgentSec with 6-layer validation, Zero Trust

### Agent Pipeline

```
Request JSON → [Intake Agent] → Kafka → [Risk Agent] → Kafka → [Decision Agent] → Result
                Claude Haiku     topic    Claude Sonnet   topic   Claude Sonnet
                T=0.0           .v1      T=0.2           .v1     T=0.1
```

| Agent | Model | Temperature | Role |
|-------|-------|-------------|------|
| Intake Specialist | Claude 3.5 Haiku | 0.0 | Validation & normalization |
| Risk Analyst | Claude Sonnet 4 / Opus 4.5 | 0.2 | Risk evaluation + RAG |
| Loan Officer | Claude 3.5 Sonnet | 0.1 | Final decision |

### Core Modules (phase4/src/shared/)

- **`instrumented_agent.py`** - Base class `InstrumentedAgent[T]` with telemetry, Kafka integration
- **`telemetry.py`** - `AgentTelemetry` singleton for OpenTelemetry distributed tracing
- **`logging.py`** - Structured logging with structlog, automatic trace_id injection

### Data Flow

Agents communicate **only via Kafka topics** (Zero Trust):
- `finance.loan.application.v1` - Validated applications (Avro)
- `risk.scoring.result.v1` - Risk assessments (Avro)
- `finance.loan.decision.v1` - Final decisions (Avro)

## Constitution Rules (docs/07-Constitution.md)

### Three Laws
1. **Contract Integrity:** Events must validate against Avro schemas. Invalid = Dead Letter Queue.
2. **Cognitive Transparency:** Every decision includes Chain of Thought justification.
3. **Security & Privacy:** Prompts protected, PII sanitized, injections blocked.

### Development Principles
- **Schema First:** Define Avro schema → Generate Pydantic → Implement → Test
- **Event-Driven Only:** No direct agent-to-agent calls, Kafka only
- **Fail Fast, Fail Loud:** No silent failures, immediate error signals

## Test Coverage Targets

| Level | Target | Minimum |
|-------|--------|---------|
| L1 Unit | 80% | 70% |
| L2 Cognitive | All agents score ≥ 7/10 | - |
| L3 Adversarial | 95% attack blocking | 90% |

## Environment Variables

```bash
ANTHROPIC_API_KEY         # Required
KAFKA_BOOTSTRAP_SERVERS   # Default: localhost:9092
OTEL_SERVICE_NAME         # Default: agent-mesh-kafka
OTEL_CONSOLE_EXPORT       # Default: true
OTEL_OTLP_EXPORT          # Default: false
ENVIRONMENT               # development/staging/production
```

## Phase Progression

| Phase | Focus | Infrastructure |
|-------|-------|----------------|
| 0 | MVP sequential scripts | None |
| 1 | Event-driven Kafka | Kafka |
| 2 | RAG with ChromaDB | Kafka + ChromaDB |
| 3 | Evaluation Diamond | Kafka + ChromaDB |
| 4 | Production governance | Full stack + OpenTelemetry |
