#!/bin/bash
# Register all Avro schemas with Schema Registry

set -e

SCHEMA_REGISTRY_URL="${SCHEMA_REGISTRY_URL:-http://localhost:8081}"
SCHEMAS_DIR="${SCHEMAS_DIR:-schemas}"

echo "Registering Avro schemas with Schema Registry at $SCHEMA_REGISTRY_URL..."

# Function to validate a schema
validate_schema() {
    local schema_file=$1

    if [ ! -f "$schema_file" ]; then
        echo "Error: Schema file not found: $schema_file"
        return 1
    fi

    # Validate JSON syntax
    if ! jq empty "$schema_file" 2>/dev/null; then
        echo "Error: Invalid JSON in $schema_file"
        return 1
    fi

    # Validate required Avro fields
    local schema_type=$(jq -r '.type' "$schema_file")
    local schema_name=$(jq -r '.name' "$schema_file")
    local schema_namespace=$(jq -r '.namespace' "$schema_file")

    if [ "$schema_type" != "record" ]; then
        echo "Error: Schema type must be 'record' in $schema_file"
        return 1
    fi

    if [ -z "$schema_name" ] || [ "$schema_name" == "null" ]; then
        echo "Error: Schema name is required in $schema_file"
        return 1
    fi

    if [ -z "$schema_namespace" ] || [ "$schema_namespace" == "null" ]; then
        echo "Error: Schema namespace is required in $schema_file"
        return 1
    fi

    echo "  Validated: $schema_file"
    return 0
}

# Function to register a schema
register_schema() {
    local subject=$1
    local schema_file=$2

    if [ ! -f "$schema_file" ]; then
        echo "Warning: Schema file not found: $schema_file"
        return 0
    fi

    echo "Registering schema for subject: $subject"

    # Read schema and escape for JSON
    local schema_content=$(cat "$schema_file" | jq -c '.' | jq -Rs '.')

    local response=$(curl -s -X POST \
        -H "Content-Type: application/vnd.schemaregistry.v1+json" \
        --data "{\"schema\": $schema_content}" \
        "$SCHEMA_REGISTRY_URL/subjects/${subject}/versions")

    local schema_id=$(echo $response | jq -r '.id')

    if [ "$schema_id" != "null" ] && [ -n "$schema_id" ]; then
        echo "  Registered with ID: $schema_id"
    else
        echo "  Warning: May already exist or error: $response"
    fi
}

# Register bancaire schemas (when they exist)
if [ -d "$SCHEMAS_DIR/bancaire" ]; then
    echo ""
    echo "=== Validating schemas ==="
    validation_failed=0
    for schema_file in "$SCHEMAS_DIR/bancaire"/*.avsc; do
        if [ -f "$schema_file" ]; then
            if ! validate_schema "$schema_file"; then
                validation_failed=1
            fi
        fi
    done

    if [ $validation_failed -eq 1 ]; then
        echo "Schema validation failed. Aborting registration."
        exit 1
    fi

    echo ""
    echo "=== Registering schemas ==="
    for schema_file in "$SCHEMAS_DIR/bancaire"/*.avsc; do
        if [ -f "$schema_file" ]; then
            filename=$(basename "$schema_file" .avsc)
            # Convert filename to subject name: compte-ouvert -> bancaire.compte.ouvert-value
            subject=$(echo "$filename" | sed 's/-/./g')
            register_schema "bancaire.${subject}-value" "$schema_file"
        fi
    done
else
    echo "No schemas directory found at $SCHEMAS_DIR/bancaire"
    echo "Schemas will be registered in Phase 2"
fi

echo ""
echo "Schema registration complete."
echo "Listing registered subjects:"
curl -s "$SCHEMA_REGISTRY_URL/subjects" | jq '.'
