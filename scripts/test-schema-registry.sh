#!/bin/bash
# Test Schema Registry functionality

set -e

SCHEMA_REGISTRY_URL="${SCHEMA_REGISTRY_URL:-http://localhost:8081}"
MAX_RETRIES="${MAX_RETRIES:-30}"
RETRY_INTERVAL="${RETRY_INTERVAL:-2}"

echo "Testing Schema Registry at $SCHEMA_REGISTRY_URL..."

# Wait for Schema Registry to be ready
echo "Step 1: Waiting for Schema Registry..."
retry_count=0
while [ $retry_count -lt $MAX_RETRIES ]; do
    if curl -s "$SCHEMA_REGISTRY_URL/subjects" > /dev/null 2>&1; then
        echo "Schema Registry is ready!"
        break
    fi

    retry_count=$((retry_count + 1))
    echo "Attempt $retry_count/$MAX_RETRIES - Schema Registry not ready, waiting ${RETRY_INTERVAL}s..."
    sleep $RETRY_INTERVAL
done

if [ $retry_count -eq $MAX_RETRIES ]; then
    echo "ERROR: Schema Registry failed to become ready"
    exit 1
fi

# Register a test schema
echo "Step 2: Registering test schema..."
TEST_SCHEMA='{
  "type": "record",
  "name": "TestEvent",
  "namespace": "com.edalab.test",
  "fields": [
    {"name": "id", "type": "string"},
    {"name": "timestamp", "type": "long"}
  ]
}'

RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/vnd.schemaregistry.v1+json" \
    --data "{\"schema\": $(echo $TEST_SCHEMA | jq -c '.' | jq -Rs .)}" \
    "$SCHEMA_REGISTRY_URL/subjects/test-topic-value/versions")

SCHEMA_ID=$(echo $RESPONSE | jq -r '.id')

if [ "$SCHEMA_ID" == "null" ] || [ -z "$SCHEMA_ID" ]; then
    echo "ERROR: Failed to register schema"
    echo "Response: $RESPONSE"
    exit 1
fi

echo "Schema registered with ID: $SCHEMA_ID"

# Retrieve the schema
echo "Step 3: Retrieving schema by ID..."
RETRIEVED=$(curl -s "$SCHEMA_REGISTRY_URL/schemas/ids/$SCHEMA_ID")

if [ -z "$RETRIEVED" ]; then
    echo "ERROR: Failed to retrieve schema"
    exit 1
fi

echo "Retrieved schema: $RETRIEVED"

# Verify subjects
echo "Step 4: Listing subjects..."
SUBJECTS=$(curl -s "$SCHEMA_REGISTRY_URL/subjects")
echo "Subjects: $SUBJECTS"

if echo "$SUBJECTS" | grep -q "test-topic-value"; then
    echo "SUCCESS: Schema Registry test passed!"
    exit 0
else
    echo "ERROR: test-topic-value not found in subjects"
    exit 1
fi
