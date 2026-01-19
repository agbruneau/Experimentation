#!/bin/bash
# ============================================================================
# test-kafka.sh - Test Kafka connectivity and basic operations
# ============================================================================

set -e

echo "=========================================="
echo "Testing Kafka..."
echo "=========================================="

# Wait for Kafka to be ready
echo "1. Checking Kafka broker status..."
if ! docker exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092 > /dev/null 2>&1; then
    echo "✗ Kafka broker is not responding"
    exit 1
fi
echo "✓ Kafka broker is responding"

# Create a test topic
TEST_TOPIC="test-topic-$(date +%s)"
echo ""
echo "2. Creating test topic: $TEST_TOPIC"
docker exec kafka kafka-topics --bootstrap-server localhost:9092 \
    --create \
    --topic "$TEST_TOPIC" \
    --partitions 1 \
    --replication-factor 1 \
    > /dev/null 2>&1
echo "✓ Test topic created"

# List topics
echo ""
echo "3. Listing topics..."
docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list

# Produce a test message
echo ""
echo "4. Producing test message..."
echo "Hello EDA-Lab" | docker exec -i kafka kafka-console-producer \
    --bootstrap-server localhost:9092 \
    --topic "$TEST_TOPIC" \
    > /dev/null 2>&1
echo "✓ Test message produced"

# Consume the test message
echo ""
echo "5. Consuming test message..."
MESSAGE=$(docker exec kafka kafka-console-consumer \
    --bootstrap-server localhost:9092 \
    --topic "$TEST_TOPIC" \
    --from-beginning \
    --max-messages 1 \
    --timeout-ms 10000 \
    2>/dev/null)

if [ "$MESSAGE" = "Hello EDA-Lab" ]; then
    echo "✓ Test message consumed: '$MESSAGE'"
else
    echo "✗ Unexpected message: '$MESSAGE'"
    exit 1
fi

# Delete test topic
echo ""
echo "6. Cleaning up test topic..."
docker exec kafka kafka-topics --bootstrap-server localhost:9092 \
    --delete \
    --topic "$TEST_TOPIC" \
    > /dev/null 2>&1
echo "✓ Test topic deleted"

echo ""
echo "=========================================="
echo "✓ Kafka tests passed!"
echo "=========================================="
