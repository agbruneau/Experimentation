#!/bin/bash
# ============================================================================
# test-schema-registry.sh - Test Schema Registry connectivity
# ============================================================================

set -e

SCHEMA_REGISTRY_URL="${SCHEMA_REGISTRY_URL:-http://localhost:8081}"

echo "=========================================="
echo "Testing Schema Registry..."
echo "=========================================="

# Check Schema Registry is responding
echo "1. Checking Schema Registry status..."
if ! curl -s "${SCHEMA_REGISTRY_URL}/subjects" > /dev/null 2>&1; then
    echo "✗ Schema Registry is not responding"
    exit 1
fi
echo "✓ Schema Registry is responding"

# Register a test schema
TEST_SUBJECT="test-subject-$(date +%s)"
echo ""
echo "2. Registering test schema: $TEST_SUBJECT"

SCHEMA='{
  "type": "record",
  "name": "TestEvent",
  "namespace": "com.edalab.test",
  "fields": [
    {"name": "id", "type": "string"},
    {"name": "timestamp", "type": "long"},
    {"name": "message", "type": "string"}
  ]
}'

# Escape the schema for JSON payload
SCHEMA_ESCAPED=$(echo "$SCHEMA" | jq -c .)
PAYLOAD="{\"schema\": $(echo "$SCHEMA_ESCAPED" | jq -Rs .)}"

RESPONSE=$(curl -s -X POST "${SCHEMA_REGISTRY_URL}/subjects/${TEST_SUBJECT}/versions" \
    -H "Content-Type: application/vnd.schemaregistry.v1+json" \
    -d "$PAYLOAD")

SCHEMA_ID=$(echo "$RESPONSE" | jq -r '.id // empty')

if [ -z "$SCHEMA_ID" ]; then
    echo "✗ Failed to register schema"
    echo "Response: $RESPONSE"
    exit 1
fi
echo "✓ Schema registered with ID: $SCHEMA_ID"

# Retrieve the schema
echo ""
echo "3. Retrieving schema by ID: $SCHEMA_ID"
RETRIEVED=$(curl -s "${SCHEMA_REGISTRY_URL}/schemas/ids/${SCHEMA_ID}")
RETRIEVED_TYPE=$(echo "$RETRIEVED" | jq -r '.schema' | jq -r '.type // empty')

if [ "$RETRIEVED_TYPE" = "record" ]; then
    echo "✓ Schema retrieved successfully"
else
    echo "✗ Failed to retrieve schema"
    echo "Response: $RETRIEVED"
    exit 1
fi

# Check compatibility (should be compatible with itself)
echo ""
echo "4. Checking schema compatibility..."
COMPAT_RESPONSE=$(curl -s -X POST "${SCHEMA_REGISTRY_URL}/compatibility/subjects/${TEST_SUBJECT}/versions/latest" \
    -H "Content-Type: application/vnd.schemaregistry.v1+json" \
    -d "$PAYLOAD")

IS_COMPATIBLE=$(echo "$COMPAT_RESPONSE" | jq -r '.is_compatible // empty')

if [ "$IS_COMPATIBLE" = "true" ]; then
    echo "✓ Schema is compatible"
else
    echo "✗ Schema compatibility check failed"
    echo "Response: $COMPAT_RESPONSE"
fi

# List all subjects
echo ""
echo "5. Listing subjects..."
curl -s "${SCHEMA_REGISTRY_URL}/subjects" | jq .

# Clean up - delete the test subject
echo ""
echo "6. Cleaning up test subject..."
curl -s -X DELETE "${SCHEMA_REGISTRY_URL}/subjects/${TEST_SUBJECT}" > /dev/null 2>&1
curl -s -X DELETE "${SCHEMA_REGISTRY_URL}/subjects/${TEST_SUBJECT}?permanent=true" > /dev/null 2>&1
echo "✓ Test subject deleted"

echo ""
echo "=========================================="
echo "✓ Schema Registry tests passed!"
echo "=========================================="
