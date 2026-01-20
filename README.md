# EDA-Lab

Academic simulation of Event Driven Architecture (EDA) for learning EDA patterns.

## Overview

EDA-Lab implements a simulated banking domain (French financial services context) to demonstrate Event Driven Architecture patterns, starting with Pub/Sub and progressively adding Event Sourcing, CQRS, Saga patterns, and more.

## Prerequisites

- **Docker Desktop** with WSL2 backend (Windows) or Docker Engine (Linux/macOS)
- **Go 1.21+**
- **Node.js 20 LTS**
- **Make** (Git Bash or WSL2 on Windows)

## Quick Start

```bash
# Start infrastructure (Kafka, Schema Registry, PostgreSQL)
make infra-up

# Validate infrastructure
make test-infra

# Stop infrastructure
make infra-down
```

## Architecture

```
Simulator (produces) --> Kafka --> Bancaire (consumes/persists)
                           |
                        Gateway --> WebSocket --> web-ui
```

### Services

| Service | Description | Port |
|---------|-------------|------|
| simulator | Generates fake banking events | 8080 |
| bancaire | Consumes events, persists to PostgreSQL | 8081 |
| gateway | REST API proxy + WebSocket hub | 8082 |

### Infrastructure

| Component | Description | Port |
|-----------|-------------|------|
| Kafka | Message broker (KRaft mode) | 9092 |
| Schema Registry | Avro schema management | 8081 |
| PostgreSQL | Database | 5432 |
| Prometheus | Metrics collection | 9090 |
| Grafana | Dashboards | 3000 |

## Documentation

| Document | Purpose |
|----------|---------|
| [PRD.MD](PRD.MD) | Product Requirements - EDA patterns specs |
| [PLAN.MD](PLAN.MD) | Implementation plan - Technical phases |

## Project Structure

```
services/           # Go microservices
  bancaire/         # Event consumer service
  simulator/        # Event generator service
  gateway/          # REST + WebSocket gateway
pkg/                # Shared Go packages
schemas/            # Avro schemas
web-ui/             # React frontend
infra/              # Docker and infrastructure config
scripts/            # Utility scripts
tests/              # Integration and E2E tests
```

## License

Educational project - MIT License
