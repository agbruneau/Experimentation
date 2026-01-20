.PHONY: help infra-up infra-down infra-logs infra-clean test-infra kafka-topics kafka-create-topic services-up services-down test-unit test-integration test-e2e validate-mvp clean

# Default target
help:
	@echo "EDA-Lab Makefile Commands"
	@echo ""
	@echo "Infrastructure:"
	@echo "  make infra-up        - Start all infrastructure containers"
	@echo "  make infra-down      - Stop all infrastructure containers"
	@echo "  make infra-logs      - View infrastructure container logs"
	@echo "  make infra-clean     - Remove volumes and restart fresh"
	@echo "  make test-infra      - Validate infrastructure is healthy"
	@echo ""
	@echo "Kafka:"
	@echo "  make kafka-topics    - List all Kafka topics"
	@echo "  make kafka-create-topic TOPIC=name - Create a specific topic"
	@echo ""
	@echo "Services:"
	@echo "  make services-up     - Start all application services"
	@echo "  make services-down   - Stop all application services"
	@echo ""
	@echo "Testing:"
	@echo "  make test-unit       - Run unit tests"
	@echo "  make test-integration - Run integration tests"
	@echo "  make test-e2e        - Run end-to-end tests"
	@echo ""
	@echo "Validation:"
	@echo "  make validate-mvp    - Run full MVP validation"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean           - Clean build artifacts"

# Infrastructure commands
infra-up:
	@echo "Starting infrastructure..."
	cd infra && docker-compose up -d
	@echo "Waiting for services to be healthy..."
	@./scripts/wait-for-infra.sh
	@echo "Creating Kafka topics..."
	@./scripts/create-topics.sh
	@echo "Infrastructure is ready!"

infra-down:
	@echo "Stopping infrastructure..."
	cd infra && docker-compose down

infra-logs:
	cd infra && docker-compose logs -f

infra-clean:
	@echo "Cleaning infrastructure..."
	cd infra && docker-compose down -v --remove-orphans
	@echo "Infrastructure cleaned!"

test-infra:
	@echo "Testing infrastructure..."
	@./scripts/test-infra.sh

# Kafka commands
kafka-topics:
	docker exec edalab-kafka kafka-topics --bootstrap-server localhost:9092 --list

kafka-create-topic:
ifndef TOPIC
	$(error TOPIC is required. Usage: make kafka-create-topic TOPIC=my-topic)
endif
	docker exec edalab-kafka kafka-topics --bootstrap-server localhost:9092 --create --topic $(TOPIC) --partitions 3 --replication-factor 1 --if-not-exists

# Services commands
services-up:
	@echo "Starting application services..."
	cd infra && docker-compose --profile services up -d
	@echo "Services started!"

services-down:
	@echo "Stopping application services..."
	cd infra && docker-compose --profile services down

# Testing commands
test-unit:
	@echo "Running unit tests..."
	go test ./pkg/... ./services/... -v -short

test-integration:
	@echo "Running integration tests..."
	go test ./tests/integration/... -v -tags=integration

test-e2e:
	@echo "Running end-to-end tests..."
	go test ./tests/e2e/... -v -tags=e2e

# Validation
validate-mvp:
	@echo "Running MVP validation..."
	@./scripts/validate-mvp.sh

# Utilities
clean:
	@echo "Cleaning build artifacts..."
	go clean ./...
	rm -rf bin/ dist/ coverage/
	@echo "Clean complete!"
