#!/bin/bash
# Test PostgreSQL functionality

set -e

POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-edalab}"
POSTGRES_USER="${POSTGRES_USER:-edalab}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-edalab_password}"
MAX_RETRIES="${MAX_RETRIES:-30}"
RETRY_INTERVAL="${RETRY_INTERVAL:-2}"

echo "Testing PostgreSQL at $POSTGRES_HOST:$POSTGRES_PORT..."

# Wait for PostgreSQL to be ready
echo "Step 1: Waiting for PostgreSQL..."
retry_count=0
while [ $retry_count -lt $MAX_RETRIES ]; do
    if docker exec edalab-postgres pg_isready -U $POSTGRES_USER -d $POSTGRES_DB > /dev/null 2>&1; then
        echo "PostgreSQL is ready!"
        break
    fi

    retry_count=$((retry_count + 1))
    echo "Attempt $retry_count/$MAX_RETRIES - PostgreSQL not ready, waiting ${RETRY_INTERVAL}s..."
    sleep $RETRY_INTERVAL
done

if [ $retry_count -eq $MAX_RETRIES ]; then
    echo "ERROR: PostgreSQL failed to become ready"
    exit 1
fi

# Test query
echo "Step 2: Running test query..."
RESULT=$(docker exec edalab-postgres psql -U $POSTGRES_USER -d $POSTGRES_DB -t -c "SELECT status FROM bancaire.health_check LIMIT 1;" 2>&1)

if echo "$RESULT" | grep -q "healthy"; then
    echo "SUCCESS: PostgreSQL test passed!"
    echo "Result: $RESULT"
    exit 0
else
    echo "ERROR: Unexpected result from health check query"
    echo "Result: $RESULT"
    exit 1
fi
