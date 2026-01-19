#!/bin/bash
# ============================================================================
# create-topics.sh - Create all Kafka topics for EDA-Lab
# ============================================================================

set -e

KAFKA_CONTAINER="${KAFKA_CONTAINER:-kafka}"
BOOTSTRAP_SERVER="localhost:9092"
PARTITIONS="${PARTITIONS:-3}"
REPLICATION_FACTOR="${REPLICATION_FACTOR:-1}"

echo "=========================================="
echo "Creating Kafka Topics..."
echo "=========================================="

# Define all topics
TOPICS=(
    # Bancaire domain events
    "bancaire.compte.ouvert"
    "bancaire.compte.ferme"
    "bancaire.depot.effectue"
    "bancaire.retrait.effectue"
    "bancaire.virement.emis"
    "bancaire.virement.recu"
    "bancaire.paiement-prime.effectue"

    # System topics
    "system.dlq"
    "system.audit"
)

# Function to create a topic
create_topic() {
    local topic=$1
    echo -n "Creating topic: $topic ... "

    # Check if topic exists
    if docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $BOOTSTRAP_SERVER --list 2>/dev/null | grep -q "^${topic}$"; then
        echo "already exists"
        return 0
    fi

    # Create the topic
    if docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $BOOTSTRAP_SERVER \
        --create \
        --topic "$topic" \
        --partitions $PARTITIONS \
        --replication-factor $REPLICATION_FACTOR \
        > /dev/null 2>&1; then
        echo "✓ created"
    else
        echo "✗ failed"
        return 1
    fi
}

# Wait for Kafka to be ready
echo "Waiting for Kafka to be ready..."
./scripts/wait-for-kafka.sh

echo ""
echo "Creating topics..."
echo ""

# Create all topics
FAILED=0
for topic in "${TOPICS[@]}"; do
    if ! create_topic "$topic"; then
        FAILED=$((FAILED + 1))
    fi
done

echo ""
echo "=========================================="
echo "Listing all topics:"
echo "=========================================="
docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $BOOTSTRAP_SERVER --list

echo ""
if [ $FAILED -eq 0 ]; then
    echo "✓ All topics created successfully!"
else
    echo "✗ $FAILED topic(s) failed to create"
    exit 1
fi
