# ============================================================================
# EDA-Lab Makefile
# ============================================================================

.PHONY: help infra-up infra-down infra-logs infra-clean test-infra \
        services-up services-down kafka-topics kafka-create-topic \
        test-unit test-integration test-e2e clean

# Default target
.DEFAULT_GOAL := help

# ============================================================================
# Variables
# ============================================================================
DOCKER_COMPOSE := docker-compose -f infra/docker-compose.yml
DOCKER_COMPOSE_SERVICES := docker-compose -f infra/docker-compose.yml --profile services

# ============================================================================
# Help
# ============================================================================
help: ## Affiche cette aide
	@echo "EDA-Lab - Commandes disponibles:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

# ============================================================================
# Infrastructure
# ============================================================================
infra-up: ## Démarre l'infrastructure (Kafka, PostgreSQL, Schema Registry)
	$(DOCKER_COMPOSE) up -d
	@echo "Infrastructure démarrée. Attendez ~30s pour que tout soit prêt."
	@echo "Utilisez 'make test-infra' pour vérifier."

infra-down: ## Arrête l'infrastructure
	$(DOCKER_COMPOSE) down

infra-logs: ## Affiche les logs de l'infrastructure
	$(DOCKER_COMPOSE) logs -f

infra-clean: ## Supprime les volumes et repart de zéro
	$(DOCKER_COMPOSE) down -v --remove-orphans
	@echo "Volumes supprimés."

test-infra: ## Valide que l'infrastructure est opérationnelle
	@echo "Test de l'infrastructure..."
	@./scripts/test-kafka.sh || echo "Kafka: FAILED"
	@./scripts/test-schema-registry.sh || echo "Schema Registry: FAILED"
	@./scripts/test-postgres.sh || echo "PostgreSQL: FAILED"
	@echo "Tests terminés."

# ============================================================================
# Services
# ============================================================================
services-up: ## Démarre tous les services applicatifs
	$(DOCKER_COMPOSE_SERVICES) up -d

services-down: ## Arrête les services applicatifs
	$(DOCKER_COMPOSE_SERVICES) down

# ============================================================================
# Kafka
# ============================================================================
kafka-topics: ## Liste les topics Kafka
	@docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list

kafka-create-topic: ## Crée un topic Kafka (usage: make kafka-create-topic TOPIC=nom)
ifndef TOPIC
	$(error TOPIC n'est pas défini. Usage: make kafka-create-topic TOPIC=nom)
endif
	@docker exec kafka kafka-topics --bootstrap-server localhost:9092 --create --topic $(TOPIC) --partitions 3 --replication-factor 1

kafka-create-topics: ## Crée tous les topics par défaut
	@./scripts/create-topics.sh

# ============================================================================
# Tests
# ============================================================================
test-unit: ## Exécute les tests unitaires
	@echo "Exécution des tests unitaires..."
	go test ./pkg/... ./services/... -v -short

test-integration: ## Exécute les tests d'intégration (nécessite infra-up)
	@echo "Exécution des tests d'intégration..."
	go test ./tests/integration/... -v -tags=integration

test-e2e: ## Exécute les tests end-to-end
	@echo "Exécution des tests E2E..."
	go test ./tests/e2e/... -v -tags=e2e

test-all: test-unit test-integration test-e2e ## Exécute tous les tests

# ============================================================================
# Build
# ============================================================================
build: ## Build tous les services
	@echo "Build des services..."
	cd services/simulator && go build -o ../../bin/simulator ./cmd/simulator
	cd services/bancaire && go build -o ../../bin/bancaire ./cmd/bancaire
	cd services/gateway && go build -o ../../bin/gateway ./cmd/gateway
	@echo "Build terminé. Binaires dans ./bin/"

build-docker: ## Build les images Docker
	$(DOCKER_COMPOSE_SERVICES) build

# ============================================================================
# Development
# ============================================================================
dev-ui: ## Démarre le frontend en mode développement
	cd web-ui && npm run dev

dev-simulator: ## Démarre le simulator en mode développement
	cd services/simulator && go run ./cmd/simulator

dev-bancaire: ## Démarre le service bancaire en mode développement
	cd services/bancaire && go run ./cmd/bancaire

dev-gateway: ## Démarre le gateway en mode développement
	cd services/gateway && go run ./cmd/gateway

# ============================================================================
# Utilities
# ============================================================================
clean: ## Nettoie les artefacts de build
	rm -rf bin/
	rm -rf web-ui/dist/
	rm -rf web-ui/node_modules/
	go clean -cache

validate-mvp: ## Exécute la validation complète du MVP
	@./scripts/validate-mvp.sh

# ============================================================================
# Schema Registry
# ============================================================================
register-schemas: ## Enregistre les schémas Avro dans Schema Registry
	@./scripts/register-schemas.sh

generate-avro: ## Génère le code Go depuis les schémas Avro
	@./scripts/generate-avro.sh
