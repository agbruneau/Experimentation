#!/bin/bash
# ============================================================================
# wait-for-kafka.sh - Wait for Kafka to be ready
# ============================================================================

set -e

KAFKA_BOOTSTRAP_SERVER="${KAFKA_BOOTSTRAP_SERVER:-localhost:9092}"
MAX_ATTEMPTS="${MAX_ATTEMPTS:-30}"
SLEEP_INTERVAL="${SLEEP_INTERVAL:-2}"

echo "Waiting for Kafka at $KAFKA_BOOTSTRAP_SERVER..."

attempt=1
while [ $attempt -le $MAX_ATTEMPTS ]; do
    echo "Attempt $attempt/$MAX_ATTEMPTS..."

    if docker exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092 > /dev/null 2>&1; then
        echo "✓ Kafka is ready!"
        exit 0
    fi

    echo "Kafka not ready yet, waiting ${SLEEP_INTERVAL}s..."
    sleep $SLEEP_INTERVAL
    attempt=$((attempt + 1))
done

echo "✗ Kafka failed to become ready after $MAX_ATTEMPTS attempts"
exit 1
