#!/bin/bash
# Script to register Avro schemas in Confluent Schema Registry

set -e

SCHEMA_REGISTRY_URL="${SCHEMA_REGISTRY_URL:-http://localhost:8081}"
SCHEMAS_DIR="${SCHEMAS_DIR:-./schemas}"

echo "=== Registering Avro Schemas in Schema Registry ==="
echo "Schema Registry URL: $SCHEMA_REGISTRY_URL"
echo "Schemas directory: $SCHEMAS_DIR"
echo ""

# Wait for Schema Registry to be available
wait_for_schema_registry() {
    echo "Waiting for Schema Registry to be ready..."
    max_attempts=30
    attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "$SCHEMA_REGISTRY_URL/subjects" > /dev/null 2>&1; then
            echo "Schema Registry is ready!"
            return 0
        fi
        attempt=$((attempt + 1))
        echo "Attempt $attempt/$max_attempts - Schema Registry not ready yet..."
        sleep 2
    done
    echo "ERROR: Schema Registry did not become ready in time"
    exit 1
}

# Register a schema
register_schema() {
    local subject="$1"
    local schema_file="$2"

    if [ ! -f "$schema_file" ]; then
        echo "ERROR: Schema file not found: $schema_file"
        return 1
    fi

    echo "Registering schema: $subject"

    # Read and escape the schema for JSON
    schema_content=$(cat "$schema_file" | jq -c '.')

    # Create the request payload
    payload=$(jq -n --arg schema "$schema_content" '{"schema": $schema}')

    # Register the schema
    response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/vnd.schemaregistry.v1+json" \
        -d "$payload" \
        "$SCHEMA_REGISTRY_URL/subjects/$subject/versions")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" -eq 200 ] || [ "$http_code" -eq 409 ]; then
        schema_id=$(echo "$body" | jq -r '.id // "already exists"')
        echo "  ✓ Registered with ID: $schema_id"
        return 0
    else
        echo "  ✗ Failed to register schema (HTTP $http_code)"
        echo "    Response: $body"
        return 1
    fi
}

# Main execution
wait_for_schema_registry

echo ""
echo "=== Registering Bancaire Domain Schemas ==="
echo ""

# Define schema mappings: subject -> schema file
declare -A SCHEMAS=(
    ["bancaire.compte.ouvert-value"]="$SCHEMAS_DIR/bancaire/compte-ouvert.avsc"
    ["bancaire.compte.ferme-value"]="$SCHEMAS_DIR/bancaire/compte-ferme.avsc"
    ["bancaire.depot.effectue-value"]="$SCHEMAS_DIR/bancaire/depot-effectue.avsc"
    ["bancaire.retrait.effectue-value"]="$SCHEMAS_DIR/bancaire/retrait-effectue.avsc"
    ["bancaire.virement.emis-value"]="$SCHEMAS_DIR/bancaire/virement-emis.avsc"
    ["bancaire.virement.recu-value"]="$SCHEMAS_DIR/bancaire/virement-recu.avsc"
    ["bancaire.paiement-prime.effectue-value"]="$SCHEMAS_DIR/bancaire/paiement-prime-effectue.avsc"
)

success_count=0
error_count=0

for subject in "${!SCHEMAS[@]}"; do
    schema_file="${SCHEMAS[$subject]}"
    if register_schema "$subject" "$schema_file"; then
        success_count=$((success_count + 1))
    else
        error_count=$((error_count + 1))
    fi
done

echo ""
echo "=== Schema Registration Complete ==="
echo "Successful: $success_count"
echo "Failed: $error_count"
echo ""

# List all registered subjects
echo "=== Registered Subjects ==="
curl -s "$SCHEMA_REGISTRY_URL/subjects" | jq -r '.[]' | sort

if [ $error_count -gt 0 ]; then
    exit 1
fi

echo ""
echo "All schemas registered successfully!"
