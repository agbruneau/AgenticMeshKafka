.PHONY: help infra-up infra-down infra-logs infra-clean test-infra kafka-topics kafka-create-topic clean

# Default target
.DEFAULT_GOAL := help

# Infrastructure
infra-up: ## Start all infrastructure containers
	docker-compose -f infra/docker-compose.yml up -d
	@echo "Waiting for services to be healthy..."
	@sleep 5
	@./scripts/wait-for-kafka.sh

infra-down: ## Stop all infrastructure containers
	docker-compose -f infra/docker-compose.yml down

infra-logs: ## View logs from all infrastructure containers
	docker-compose -f infra/docker-compose.yml logs -f

infra-clean: ## Remove volumes and restart fresh
	docker-compose -f infra/docker-compose.yml down -v
	@echo "Volumes removed. Run 'make infra-up' to restart."

# Tests infrastructure
test-infra: ## Run all infrastructure tests
	@echo "Testing Kafka..."
	@./scripts/wait-for-kafka.sh
	@echo "Testing Schema Registry..."
	@./scripts/test-schema-registry.sh
	@echo "Testing PostgreSQL..."
	@./scripts/test-postgres.sh
	@echo ""
	@echo "All infrastructure tests passed!"

# Kafka
kafka-topics: ## List all Kafka topics
	docker exec edalab-kafka kafka-topics --bootstrap-server localhost:9092 --list

kafka-create-topic: ## Create a Kafka topic (usage: make kafka-create-topic TOPIC=name)
	docker exec edalab-kafka kafka-topics --bootstrap-server localhost:9092 --create --topic $(TOPIC) --partitions 3 --replication-factor 1 --if-not-exists

kafka-create-all-topics: ## Create all MVP topics
	@./scripts/create-topics.sh

# Go services
test-unit: ## Run unit tests
	go test ./pkg/... ./services/...

test-integration: ## Run integration tests (requires infra-up)
	cd tests/integration && go test -tags=integration -v ./...

test-e2e: ## Run end-to-end tests (requires infra-up)
	cd tests/e2e && go test -tags=e2e -v ./...

# Utilities
clean: ## Clean build artifacts
	go clean ./...
	rm -rf bin/ dist/ coverage.out

help: ## Display this help message
	@echo "EDA-Lab Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
