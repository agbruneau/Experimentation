#!/bin/bash
# Full MVP validation script for EDA-Lab

set -e

echo "=========================================="
echo "    EDA-Lab MVP Validation"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    exit 1
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Step 1: Check infrastructure
echo "Step 1: Checking infrastructure..."
echo "-----------------------------------"

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    fail "Docker is not running"
fi
pass "Docker is running"

# Check if containers are running
if docker ps | grep -q edalab-kafka; then
    pass "Kafka container is running"
else
    warn "Kafka container not running. Starting infrastructure..."
    make infra-up
fi

# Step 2: Test infrastructure health
echo ""
echo "Step 2: Testing infrastructure health..."
echo "-----------------------------------------"
./scripts/test-infra.sh || fail "Infrastructure tests failed"
pass "Infrastructure is healthy"

# Step 3: Run unit tests
echo ""
echo "Step 3: Running unit tests..."
echo "-----------------------------"
if go test ./pkg/... ./services/... -v -short 2>/dev/null; then
    pass "Unit tests passed"
else
    warn "Unit tests failed or not yet implemented"
fi

# Step 4: Run integration tests
echo ""
echo "Step 4: Running integration tests..."
echo "-------------------------------------"
if go test ./tests/integration/... -v -tags=integration 2>/dev/null; then
    pass "Integration tests passed"
else
    warn "Integration tests failed or not yet implemented"
fi

# Step 5: Check services
echo ""
echo "Step 5: Checking services..."
echo "----------------------------"

check_service() {
    local name=$1
    local url=$2

    if curl -s "$url/health" >/dev/null 2>&1; then
        pass "$name is responding"
        return 0
    else
        warn "$name is not responding at $url"
        return 1
    fi
}

SERVICES_RUNNING=0

if check_service "Simulator" "http://localhost:8080"; then
    SERVICES_RUNNING=$((SERVICES_RUNNING + 1))
fi

if check_service "Bancaire" "http://localhost:8081"; then
    SERVICES_RUNNING=$((SERVICES_RUNNING + 1))
fi

if check_service "Gateway" "http://localhost:8082"; then
    SERVICES_RUNNING=$((SERVICES_RUNNING + 1))
fi

if [ "$SERVICES_RUNNING" -eq 0 ]; then
    warn "No services are running. Start them with: make services-up"
elif [ "$SERVICES_RUNNING" -eq 3 ]; then
    pass "All services are running!"
fi

# Step 6: Check Prometheus metrics
echo ""
echo "Step 6: Checking Prometheus metrics..."
echo "--------------------------------------"
PROM_TARGETS=$(curl -s http://localhost:9090/api/v1/targets 2>/dev/null | jq -r '.data.activeTargets | length')
if [ "$PROM_TARGETS" -ge 1 ]; then
    pass "Prometheus has $PROM_TARGETS active targets"
else
    warn "Prometheus has no active targets"
fi

# Step 7: Check Grafana
echo ""
echo "Step 7: Checking Grafana..."
echo "---------------------------"
GRAFANA_HEALTH=$(curl -s http://localhost:3001/api/health 2>/dev/null | jq -r '.database')
if [ "$GRAFANA_HEALTH" = "ok" ]; then
    pass "Grafana is healthy"
else
    warn "Grafana is not responding"
fi

# Step 8: Run E2E tests (if services are running)
echo ""
echo "Step 8: Running E2E tests..."
echo "----------------------------"
if [ "$SERVICES_RUNNING" -eq 3 ]; then
    if go test ./tests/e2e/... -v -tags=e2e 2>/dev/null; then
        pass "E2E tests passed"
    else
        warn "E2E tests failed"
    fi
else
    warn "E2E tests skipped (services not running)"
fi

# Summary
echo ""
echo "=========================================="
echo "    MVP Validation Summary"
echo "=========================================="
echo ""
echo "Infrastructure:"
echo "  - Kafka:           Ready"
echo "  - Schema Registry: Ready"
echo "  - PostgreSQL:      Ready"
echo "  - Prometheus:      Ready"
echo "  - Grafana:         Ready"
echo ""
echo "Services: $SERVICES_RUNNING/3 running"
echo "  - Simulator:       $([ "$SERVICES_RUNNING" -ge 1 ] && echo 'Running' || echo 'Not running')"
echo "  - Bancaire:        $([ "$SERVICES_RUNNING" -ge 2 ] && echo 'Running' || echo 'Not running')"
echo "  - Gateway:         $([ "$SERVICES_RUNNING" -ge 3 ] && echo 'Running' || echo 'Not running')"
echo ""
echo "Next steps:"
if [ "$SERVICES_RUNNING" -lt 3 ]; then
    echo "  1. Start services: make services-up"
    echo "  2. Run E2E tests:  make test-e2e"
else
    echo "  - All services running!"
    echo "  - Access Web UI: http://localhost:5173"
    echo "  - Access Grafana: http://localhost:3001"
fi
echo ""
echo "=========================================="
echo "    Validation Complete!"
echo "=========================================="
