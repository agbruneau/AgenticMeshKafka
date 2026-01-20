#!/bin/bash
# Validate Phase 0 checkpoint

set -e

echo "=================================================="
echo "EDA-Lab Phase 0 Validation"
echo "=================================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Check Docker
echo "Checking Docker..."
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}ERROR: Docker is not running. Please start Docker Desktop.${NC}"
    exit 1
fi
echo -e "${GREEN}Docker is running${NC}"
echo ""

# Navigate to project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"
cd "$PROJECT_ROOT"

# Step 1: Start infrastructure
echo "Step 1: Starting infrastructure..."
docker-compose -f infra/docker-compose.yml up -d
echo "Waiting for services to start..."
sleep 10

# Step 2: Check container health
echo ""
echo "Step 2: Checking container health..."
echo ""

check_container() {
    local name=$1
    local status=$(docker inspect --format='{{.State.Health.Status}}' $name 2>/dev/null || echo "not found")
    if [ "$status" == "healthy" ]; then
        echo -e "  ${GREEN}[OK]${NC} $name is healthy"
        return 0
    elif [ "$status" == "starting" ]; then
        echo -e "  [..] $name is starting..."
        return 1
    else
        echo -e "  ${RED}[FAIL]${NC} $name status: $status"
        return 1
    fi
}

# Wait for containers to be healthy (max 2 minutes)
MAX_WAIT=120
WAIT_INTERVAL=5
elapsed=0

while [ $elapsed -lt $MAX_WAIT ]; do
    all_healthy=true

    check_container "edalab-kafka" || all_healthy=false
    check_container "edalab-schema-registry" || all_healthy=false
    check_container "edalab-postgres" || all_healthy=false

    if $all_healthy; then
        break
    fi

    elapsed=$((elapsed + WAIT_INTERVAL))
    echo ""
    echo "Waiting... ($elapsed/$MAX_WAIT seconds)"
    sleep $WAIT_INTERVAL
done

if ! $all_healthy; then
    echo -e "${RED}ERROR: Not all containers became healthy${NC}"
    echo "Run 'docker-compose -f infra/docker-compose.yml logs' to see errors"
    exit 1
fi

echo ""
echo "All containers are healthy!"
echo ""

# Step 3: Run infrastructure tests
echo "Step 3: Running infrastructure tests..."
echo ""

echo "Testing Kafka..."
./scripts/wait-for-kafka.sh
echo ""

echo "Testing Schema Registry..."
./scripts/test-schema-registry.sh
echo ""

echo "Testing PostgreSQL..."
./scripts/test-postgres.sh
echo ""

# Step 4: Create Kafka topics
echo "Step 4: Creating Kafka topics..."
./scripts/create-topics.sh
echo ""

# Step 5: Run Go integration tests
echo "Step 5: Running Go integration tests..."
cd tests/integration
go test -tags=integration -v ./...
cd "$PROJECT_ROOT"
echo ""

# Summary
echo "=================================================="
echo -e "${GREEN}Phase 0 Validation PASSED!${NC}"
echo "=================================================="
echo ""
echo "Infrastructure is ready:"
echo "  - Kafka:           localhost:9092"
echo "  - Schema Registry: http://localhost:8081"
echo "  - PostgreSQL:      localhost:5432"
echo ""
echo "You can now proceed to Phase 1."
