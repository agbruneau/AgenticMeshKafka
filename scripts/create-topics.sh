#!/bin/bash
# Create all MVP Kafka topics

set -e

KAFKA_CONTAINER="${KAFKA_CONTAINER:-edalab-kafka}"
BOOTSTRAP_SERVER="localhost:9092"
PARTITIONS="${PARTITIONS:-3}"
REPLICATION_FACTOR="${REPLICATION_FACTOR:-1}"

echo "Creating Kafka topics..."

# MVP Topics
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
        --partitions $PARTITIONS \
        --replication-factor $REPLICATION_FACTOR \
        --if-not-exists
done

echo ""
echo "All topics created. Listing topics:"
docker exec $KAFKA_CONTAINER kafka-topics --bootstrap-server $BOOTSTRAP_SERVER --list
