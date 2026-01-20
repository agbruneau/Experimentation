#!/bin/bash
# Register Avro schemas with Schema Registry

set -e

SCHEMA_REGISTRY_URL="http://localhost:8081"
SCHEMAS_DIR="schemas"

echo "Registering Avro schemas with Schema Registry..."

# Function to register a schema
register_schema() {
    local topic=$1
    local schema_file=$2

    if [ ! -f "$schema_file" ]; then
        echo "   [SKIP] Schema file not found: $schema_file"
        return
    fi

    # Read schema and escape for JSON
    local schema=$(cat "$schema_file" | jq -c '.')
    local payload=$(jq -n --arg schema "$schema" '{"schema": $schema}')

    echo "   Registering schema for: $topic-value"
    response=$(curl -s -X POST \
        -H "Content-Type: application/vnd.schemaregistry.v1+json" \
        -d "$payload" \
        "$SCHEMA_REGISTRY_URL/subjects/${topic}-value/versions")

    if echo "$response" | jq -e '.id' >/dev/null 2>&1; then
        schema_id=$(echo "$response" | jq -r '.id')
        echo "   [OK] Schema registered with ID: $schema_id"
    else
        echo "   [ERROR] Failed to register schema: $response"
    fi
}

# Bancaire domain schemas
echo ""
echo "Registering Bancaire domain schemas..."
register_schema "bancaire.compte.ouvert" "$SCHEMAS_DIR/bancaire/compte-ouvert.avsc"
register_schema "bancaire.depot.effectue" "$SCHEMAS_DIR/bancaire/depot-effectue.avsc"
register_schema "bancaire.retrait.effectue" "$SCHEMAS_DIR/bancaire/retrait-effectue.avsc"
register_schema "bancaire.virement.emis" "$SCHEMAS_DIR/bancaire/virement-emis.avsc"

echo ""
echo "Schema registration complete!"
echo ""
echo "Listing registered subjects:"
curl -s "$SCHEMA_REGISTRY_URL/subjects" | jq '.'
