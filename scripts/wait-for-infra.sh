#!/bin/bash
# Wait for infrastructure services to be healthy

set -e

echo "Waiting for Kafka..."
timeout=120
elapsed=0
while ! docker exec edalab-kafka kafka-broker-api-versions --bootstrap-server localhost:9092 >/dev/null 2>&1; do
    if [ $elapsed -ge $timeout ]; then
        echo "ERROR: Kafka did not become healthy within $timeout seconds"
        exit 1
    fi
    sleep 2
    elapsed=$((elapsed + 2))
    echo "  Waiting for Kafka... ($elapsed/$timeout seconds)"
done
echo "Kafka is ready!"

echo "Waiting for Schema Registry..."
elapsed=0
while ! curl -s http://localhost:8081/ >/dev/null 2>&1; do
    if [ $elapsed -ge $timeout ]; then
        echo "ERROR: Schema Registry did not become healthy within $timeout seconds"
        exit 1
    fi
    sleep 2
    elapsed=$((elapsed + 2))
    echo "  Waiting for Schema Registry... ($elapsed/$timeout seconds)"
done
echo "Schema Registry is ready!"

echo "Waiting for PostgreSQL..."
elapsed=0
while ! docker exec edalab-postgres pg_isready -U edalab -d edalab >/dev/null 2>&1; do
    if [ $elapsed -ge $timeout ]; then
        echo "ERROR: PostgreSQL did not become healthy within $timeout seconds"
        exit 1
    fi
    sleep 2
    elapsed=$((elapsed + 2))
    echo "  Waiting for PostgreSQL... ($elapsed/$timeout seconds)"
done
echo "PostgreSQL is ready!"

echo "Waiting for Prometheus..."
elapsed=0
while ! curl -s http://localhost:9090/-/healthy >/dev/null 2>&1; do
    if [ $elapsed -ge $timeout ]; then
        echo "ERROR: Prometheus did not become healthy within $timeout seconds"
        exit 1
    fi
    sleep 2
    elapsed=$((elapsed + 2))
    echo "  Waiting for Prometheus... ($elapsed/$timeout seconds)"
done
echo "Prometheus is ready!"

echo ""
echo "All infrastructure services are healthy!"
