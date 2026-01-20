#!/bin/bash
# Wait for Kafka to be ready

set -e

KAFKA_BOOTSTRAP_SERVERS="${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}"
MAX_RETRIES="${MAX_RETRIES:-30}"
RETRY_INTERVAL="${RETRY_INTERVAL:-2}"

echo "Waiting for Kafka at $KAFKA_BOOTSTRAP_SERVERS..."

retry_count=0
while [ $retry_count -lt $MAX_RETRIES ]; do
    if docker exec edalab-kafka kafka-broker-api-versions --bootstrap-server localhost:9092 > /dev/null 2>&1; then
        echo "Kafka is ready!"
        exit 0
    fi

    retry_count=$((retry_count + 1))
    echo "Attempt $retry_count/$MAX_RETRIES - Kafka not ready yet, waiting ${RETRY_INTERVAL}s..."
    sleep $RETRY_INTERVAL
done

echo "ERROR: Kafka failed to become ready after $MAX_RETRIES attempts"
exit 1
