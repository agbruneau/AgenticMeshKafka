#!/bin/bash
# Register all Avro schemas with Schema Registry

set -e

SCHEMA_REGISTRY_URL="${SCHEMA_REGISTRY_URL:-http://localhost:8081}"
SCHEMAS_DIR="${SCHEMAS_DIR:-schemas}"

echo "Registering Avro schemas with Schema Registry at $SCHEMA_REGISTRY_URL..."

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
    for schema_file in "$SCHEMAS_DIR/bancaire"/*.avsc; do
        if [ -f "$schema_file" ]; then
            filename=$(basename "$schema_file" .avsc)
            # Convert filename to subject name: compte_ouvert -> bancaire.compte.ouvert-value
            subject=$(echo "$filename" | sed 's/_/./g')
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
