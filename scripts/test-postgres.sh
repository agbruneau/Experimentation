#!/bin/bash
# ============================================================================
# test-postgres.sh - Test PostgreSQL connectivity
# ============================================================================

set -e

POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-edalab}"
POSTGRES_USER="${POSTGRES_USER:-edalab}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-edalab_password}"

echo "=========================================="
echo "Testing PostgreSQL..."
echo "=========================================="

# Check PostgreSQL is responding
echo "1. Checking PostgreSQL status..."
if ! docker exec postgres pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" > /dev/null 2>&1; then
    echo "✗ PostgreSQL is not responding"
    exit 1
fi
echo "✓ PostgreSQL is responding"

# Test connection and query
echo ""
echo "2. Testing database connection..."
RESULT=$(docker exec postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "SELECT 1" 2>/dev/null | tr -d ' ')
if [ "$RESULT" = "1" ]; then
    echo "✓ Database connection successful"
else
    echo "✗ Database connection failed"
    exit 1
fi

# Check bancaire schema exists
echo ""
echo "3. Checking bancaire schema..."
SCHEMA_EXISTS=$(docker exec postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c \
    "SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = 'bancaire');" \
    2>/dev/null | tr -d ' ')
if [ "$SCHEMA_EXISTS" = "t" ]; then
    echo "✓ Schema 'bancaire' exists"
else
    echo "✗ Schema 'bancaire' not found"
    exit 1
fi

# Check health_check table
echo ""
echo "4. Querying health_check table..."
HEALTH_STATUS=$(docker exec postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c \
    "SELECT status FROM bancaire.health_check LIMIT 1;" \
    2>/dev/null | tr -d ' ')
if [ -n "$HEALTH_STATUS" ]; then
    echo "✓ Health check status: $HEALTH_STATUS"
else
    echo "✗ Failed to query health_check table"
    exit 1
fi

# List all tables in bancaire schema
echo ""
echo "5. Listing tables in bancaire schema..."
docker exec postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c \
    "SELECT table_name FROM information_schema.tables WHERE table_schema = 'bancaire' ORDER BY table_name;"

# Test comptes table structure
echo ""
echo "6. Checking comptes table structure..."
docker exec postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c \
    "SELECT column_name, data_type FROM information_schema.columns WHERE table_schema = 'bancaire' AND table_name = 'comptes' ORDER BY ordinal_position;"

echo ""
echo "=========================================="
echo "✓ PostgreSQL tests passed!"
echo "=========================================="
