#!/bin/bash
# Create Kafka topics for EDA-Lab

set -e

KAFKA_CONTAINER="edalab-kafka"
BOOTSTRAP_SERVER="localhost:9092"

echo "Creating Kafka topics..."

# Bancaire domain topics
TOPICS=(
    "bancaire.compte.ouvert"
    "bancaire.compte.ferme"
    "bancaire.depot.effectue"
    "bancaire.retrait.effectue"
    "bancaire.virement.emis"
    "bancaire.virement.recu"
    "bancaire.paiement-prime.effectue"
    "system.dlq"
)

for topic in "${TOPICS[@]}"; do
    echo "Creating topic: $topic"
    docker exec $KAFKA_CONTAINER kafka-topics \
        --bootstrap-server $BOOTSTRAP_SERVER \
        --create \
        --topic "$topic" \
        --partitions 3 \
        --replication-factor 1 \
        --if-not-exists
done

echo ""
echo "Topics created successfully!"
echo ""
echo "Listing all topics:"
docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $BOOTSTRAP_SERVER --list
