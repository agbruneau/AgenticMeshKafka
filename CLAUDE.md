# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**EDA-Lab** is an academic simulation of Event Driven Architecture (EDA) for learning EDA patterns. The MVP implements Pub/Sub with a simulated banking domain (French financial services context).

## Terminology

> **Important**: Two numbering systems are used:
> - **Itérations (1-8)**: EDA patterns from PRD.MD (1=Pub/Sub, 2=Event Sourcing, etc.)
> - **Phases techniques (0-8)**: Technical build steps from PLAN.MD for MVP construction
>
> **Itération 1 (MVP Pub/Sub)** = **Phases techniques 0-8** from PLAN.MD

## Tech Stack

- **Backend**: Go 1.21+ (monorepo with 3 microservices)
- **Message Broker**: Confluent Platform (Kafka KRaft mode, no ZooKeeper)
- **Schema Registry**: Confluent Schema Registry with Avro
- **Database**: PostgreSQL 16
- **Frontend**: React + Vite + React Flow + Tailwind CSS + Zustand
- **Observability**: Prometheus + Grafana
- **Containerization**: Docker Compose (Windows 11 / WSL2)

## Architecture

```
Simulator (produces) → Kafka → Bancaire (consumes/persists)
                         ↓
                      Gateway → WebSocket → web-ui
```

Services:
- `simulator` - Generates fake banking events at configurable rate
- `bancaire` - Consumes events, persists accounts/transactions to PostgreSQL
- `gateway` - REST API proxy + WebSocket hub for real-time UI updates

Kafka topic naming: `<domain>.<entity>.<action>` (e.g., `bancaire.compte.ouvert`)

## Development Commands

```bash
# Infrastructure
make infra-up              # Start Kafka, Schema Registry, PostgreSQL, Prometheus, Grafana
make infra-down            # Stop all containers
make infra-logs            # View container logs
make infra-clean           # Remove volumes and restart fresh
make test-infra            # Validate infrastructure health

# Kafka
make kafka-topics                    # List all topics
make kafka-create-topic TOPIC=name   # Create a specific topic
./scripts/create-topics.sh           # Create all MVP topics
./scripts/register-schemas.sh        # Register all Avro schemas

# Go services
cd services/<service-name>
go build ./cmd/...
go test ./...
go test -race ./...
go test -v -run TestName ./path/to/package  # Run single test

# Frontend
cd web-ui
npm install
npm run dev     # Dev server on :5173
npm run build   # Production build

# Testing & Validation
make test-unit              # Unit tests
make test-integration       # Integration tests (requires infra-up)
make test-e2e               # End-to-end tests
./scripts/validate-mvp.sh   # Full MVP validation
```

## Project Structure

```
services/<name>/
├── cmd/<name>/main.go      # Entry point
├── internal/
│   ├── api/                # HTTP handlers
│   ├── domain/             # Entities
│   ├── handler/            # Kafka event handlers
│   └── repository/         # PostgreSQL persistence
├── Dockerfile
└── go.mod

pkg/                        # Shared packages: config, kafka, database, events, observability
schemas/<domain>/           # Avro schemas (namespace: com.edalab.<domain>.events)
tests/integration/          # Integration tests (//go:build integration)
tests/e2e/                  # E2E tests (//go:build e2e)
```

## Strict TDD Workflow

This project enforces **TDD Protocol**:

1. **RED**: Create test file, run `go test ./path/...` → MUST FAIL
2. **GREEN**: Write minimal implementation → MUST PASS
3. **REFACTOR**: Improve without changing behavior → MUST PASS

**Emergency Stop**: If `make test-infra` fails, STOP. Do not write code until infrastructure is healthy.

## Project Documentation

| File       | Purpose                                              |
|------------|------------------------------------------------------|
| `PRD.MD`   | Product Definition - Itérations (EDA patterns) specs |
| `PLAN.MD`  | Implementation plan - Phases techniques with details |
| `AGENT.MD` | Agent instructions for implementing iterations       |

To implement a new iteration: `Implémente l'Itération [N] du projet EDA-Lab selon le PRD.MD et AGENT.MD`

## Key Patterns (Itérations)

| # | Pattern            | Status  |
|---|--------------------|---------|
| 1 | Pub/Sub (MVP)      | Active  |
| 2 | Event Sourcing     | Planned |
| 3 | CQRS               | Planned |
| 4 | Saga Choreography  | Planned |
| 5 | Saga Orchestration | Planned |
| 6 | Event Streaming    | Planned |
| 7 | Dead Letter Queue  | Planned |
| 8 | Outbox Pattern     | Planned |
