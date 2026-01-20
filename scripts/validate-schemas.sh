#!/bin/bash
# Validate that all Avro schemas are registered in Schema Registry

set -e

SCHEMA_REGISTRY_URL="${SCHEMA_REGISTRY_URL:-http://localhost:8081}"

echo "Validating schemas in Schema Registry at $SCHEMA_REGISTRY_URL..."
echo ""

# Expected schemas for MVP
EXPECTED_SCHEMAS=(
    "bancaire.compte.ouvert-value"
    "bancaire.depot.effectue-value"
    "bancaire.virement.emis-value"
)

# Get list of registered subjects
registered=$(curl -s "$SCHEMA_REGISTRY_URL/subjects")

if [ -z "$registered" ] || [ "$registered" == "[]" ]; then
    echo "Error: No schemas registered in Schema Registry"
    exit 1
fi

echo "Registered subjects:"
echo "$registered" | jq -r '.[]' | while read subject; do
    echo "  - $subject"
done
echo ""

# Validate each expected schema
all_valid=0
for schema in "${EXPECTED_SCHEMAS[@]}"; do
    if echo "$registered" | jq -e ". | index(\"$schema\")" > /dev/null 2>&1; then
        echo "[OK] $schema is registered"

        # Get schema details
        version=$(curl -s "$SCHEMA_REGISTRY_URL/subjects/$schema/versions/latest" | jq -r '.version')
        schema_id=$(curl -s "$SCHEMA_REGISTRY_URL/subjects/$schema/versions/latest" | jq -r '.id')
        echo "     Version: $version, ID: $schema_id"
    else
        echo "[MISSING] $schema is NOT registered"
        all_valid=1
    fi
done

echo ""
if [ $all_valid -eq 0 ]; then
    echo "All expected schemas are registered."
    exit 0
else
    echo "Some schemas are missing. Please run ./scripts/register-schemas.sh"
    exit 1
fi
