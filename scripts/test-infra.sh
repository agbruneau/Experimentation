#!/bin/bash
# Test infrastructure connectivity and health

set -e

echo "=== EDA-Lab Infrastructure Test ==="
echo ""

# Test Kafka
echo "1. Testing Kafka..."
if docker exec edalab-kafka kafka-broker-api-versions --bootstrap-server localhost:9092 >/dev/null 2>&1; then
    echo "   [OK] Kafka is responding"
else
    echo "   [FAIL] Kafka is not responding"
    exit 1
fi

# Test Schema Registry
echo "2. Testing Schema Registry..."
SR_RESPONSE=$(curl -s http://localhost:8081/)
if [ "$SR_RESPONSE" = "{}" ]; then
    echo "   [OK] Schema Registry is responding"
else
    echo "   [FAIL] Schema Registry is not responding correctly"
    exit 1
fi

# Test PostgreSQL
echo "3. Testing PostgreSQL..."
if docker exec edalab-postgres pg_isready -U edalab -d edalab >/dev/null 2>&1; then
    echo "   [OK] PostgreSQL is ready"
else
    echo "   [FAIL] PostgreSQL is not ready"
    exit 1
fi

# Test PostgreSQL tables
echo "4. Testing PostgreSQL tables..."
HEALTH_CHECK=$(docker exec edalab-postgres psql -U edalab -d edalab -t -c "SELECT status FROM bancaire.health_check LIMIT 1;" 2>/dev/null | tr -d ' ')
if [ "$HEALTH_CHECK" = "initialized" ]; then
    echo "   [OK] PostgreSQL tables are initialized"
else
    echo "   [FAIL] PostgreSQL tables are not properly initialized"
    exit 1
fi

# Test Prometheus
echo "5. Testing Prometheus..."
PROM_HEALTH=$(curl -s http://localhost:9090/-/healthy)
if [ "$PROM_HEALTH" = "Prometheus Server is Healthy." ]; then
    echo "   [OK] Prometheus is healthy"
else
    echo "   [FAIL] Prometheus is not healthy"
    exit 1
fi

# Test Kafka topics
echo "6. Testing Kafka topics..."
TOPIC_COUNT=$(docker exec edalab-kafka kafka-topics --bootstrap-server localhost:9092 --list 2>/dev/null | grep -c "bancaire" || echo "0")
if [ "$TOPIC_COUNT" -ge 1 ]; then
    echo "   [OK] Kafka topics exist ($TOPIC_COUNT bancaire topics)"
else
    echo "   [WARN] No bancaire topics found - run 'make infra-up' to create them"
fi

echo ""
echo "=== All infrastructure tests passed! ==="
