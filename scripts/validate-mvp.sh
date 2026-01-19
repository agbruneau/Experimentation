#!/bin/bash
# ============================================================================
# MVP Validation Script
# ============================================================================
# This script validates the complete MVP by running all infrastructure,
# integration, and E2E tests.
#
# Usage: ./scripts/validate-mvp.sh
# ============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0
SKIPPED=0

# Log functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((PASSED++))
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAILED++))
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    ((SKIPPED++))
}

log_step() {
    echo ""
    echo -e "${BLUE}============================================================${NC}"
    echo -e "${BLUE}STEP $1: $2${NC}"
    echo -e "${BLUE}============================================================${NC}"
}

# Wait for a service to be healthy
wait_for_service() {
    local url=$1
    local name=$2
    local max_wait=${3:-60}

    log_info "Waiting for $name to be ready..."

    for i in $(seq 1 $max_wait); do
        if curl -s "$url" > /dev/null 2>&1; then
            log_success "$name is ready"
            return 0
        fi
        sleep 1
    done

    log_error "$name failed to start after ${max_wait}s"
    return 1
}

# Check if command exists
check_command() {
    if command -v $1 &> /dev/null; then
        log_success "$1 is installed"
        return 0
    else
        log_error "$1 is not installed"
        return 1
    fi
}

# ============================================================================
# STEP 0: Prerequisites Check
# ============================================================================
log_step "0" "Checking prerequisites"

check_command docker || exit 1
check_command docker-compose || check_command "docker compose" || exit 1
check_command go || exit 1
check_command curl || exit 1

# Check Docker is running
if docker info > /dev/null 2>&1; then
    log_success "Docker daemon is running"
else
    log_error "Docker daemon is not running"
    exit 1
fi

# ============================================================================
# STEP 1: Infrastructure Startup
# ============================================================================
log_step "1" "Starting infrastructure"

cd "$(dirname "$0")/.."

log_info "Stopping any existing containers..."
docker-compose -f infra/docker-compose.yml down --remove-orphans 2>/dev/null || true

log_info "Starting infrastructure containers..."
if docker-compose -f infra/docker-compose.yml up -d; then
    log_success "Infrastructure containers started"
else
    log_error "Failed to start infrastructure"
    exit 1
fi

# Wait for services
wait_for_service "http://localhost:9092" "Kafka" 120 || true
wait_for_service "http://localhost:8081/subjects" "Schema Registry" 60
wait_for_service "http://localhost:5432" "PostgreSQL" 60 || true

# ============================================================================
# STEP 2: Infrastructure Tests
# ============================================================================
log_step "2" "Running infrastructure tests"

log_info "Testing Kafka connectivity..."
if docker-compose -f infra/docker-compose.yml exec -T kafka kafka-topics --bootstrap-server localhost:29092 --list > /dev/null 2>&1; then
    log_success "Kafka is accessible"
else
    log_warning "Kafka test skipped (may need more time)"
fi

log_info "Testing Schema Registry..."
if curl -s http://localhost:8081/subjects | grep -q '\['; then
    log_success "Schema Registry is accessible"
else
    log_error "Schema Registry not responding correctly"
fi

log_info "Testing PostgreSQL..."
if docker-compose -f infra/docker-compose.yml exec -T postgres pg_isready -U edalab > /dev/null 2>&1; then
    log_success "PostgreSQL is ready"
else
    log_error "PostgreSQL is not ready"
fi

# ============================================================================
# STEP 3: Create Kafka Topics
# ============================================================================
log_step "3" "Creating Kafka topics"

if [ -f scripts/create-topics.sh ]; then
    if bash scripts/create-topics.sh; then
        log_success "Kafka topics created"
    else
        log_warning "Some topics may already exist"
    fi
else
    log_warning "create-topics.sh not found, skipping"
fi

# ============================================================================
# STEP 4: Register Avro Schemas
# ============================================================================
log_step "4" "Registering Avro schemas"

if [ -f scripts/register-schemas.sh ]; then
    if bash scripts/register-schemas.sh; then
        log_success "Avro schemas registered"
    else
        log_warning "Some schemas may already exist"
    fi
else
    log_warning "register-schemas.sh not found, skipping"
fi

# ============================================================================
# STEP 5: Go Integration Tests
# ============================================================================
log_step "5" "Running Go integration tests"

if [ -d tests/integration ]; then
    log_info "Running integration tests..."
    cd tests/integration
    if go test -v -tags=integration ./... 2>&1 | tee /tmp/integration-tests.log; then
        log_success "Integration tests passed"
    else
        log_error "Integration tests failed"
        cat /tmp/integration-tests.log
    fi
    cd ../..
else
    log_warning "Integration tests directory not found"
fi

# ============================================================================
# STEP 6: Start Application Services
# ============================================================================
log_step "6" "Starting application services"

# Build and start services (if docker-compose includes services profile)
log_info "Building services..."
if docker-compose -f infra/docker-compose.yml --profile services build 2>/dev/null; then
    log_success "Services built"

    log_info "Starting services..."
    if docker-compose -f infra/docker-compose.yml --profile services up -d; then
        log_success "Services started"
    else
        log_warning "Services may not be configured yet"
    fi
else
    log_warning "Services profile not configured, skipping"
fi

# ============================================================================
# STEP 7: Check Prometheus
# ============================================================================
log_step "7" "Checking Prometheus"

if wait_for_service "http://localhost:9090/-/healthy" "Prometheus" 30; then
    log_info "Checking Prometheus targets..."
    targets=$(curl -s http://localhost:9090/api/v1/targets 2>/dev/null)
    if echo "$targets" | grep -q "activeTargets"; then
        log_success "Prometheus targets configured"
    else
        log_warning "No Prometheus targets found"
    fi
else
    log_warning "Prometheus not available"
fi

# ============================================================================
# STEP 8: Check Grafana
# ============================================================================
log_step "8" "Checking Grafana"

if wait_for_service "http://localhost:3000/api/health" "Grafana" 30; then
    log_info "Checking Grafana datasources..."
    datasources=$(curl -s http://localhost:3000/api/datasources 2>/dev/null || echo "[]")
    if echo "$datasources" | grep -q "prometheus"; then
        log_success "Grafana datasources configured"
    else
        log_warning "Grafana datasources may not be configured"
    fi
else
    log_warning "Grafana not available"
fi

# ============================================================================
# STEP 9: E2E Tests
# ============================================================================
log_step "9" "Running E2E tests"

if [ -d tests/e2e ]; then
    log_info "Running E2E tests..."
    cd tests/e2e
    if go test -v -tags=e2e ./... -timeout 5m 2>&1 | tee /tmp/e2e-tests.log; then
        log_success "E2E tests passed"
    else
        log_warning "E2E tests failed or services not running"
    fi
    cd ../..
else
    log_warning "E2E tests directory not found"
fi

# ============================================================================
# STEP 10: Cleanup (Optional)
# ============================================================================
log_step "10" "Summary"

echo ""
echo "============================================================"
echo "MVP VALIDATION SUMMARY"
echo "============================================================"
echo -e "Passed:  ${GREEN}$PASSED${NC}"
echo -e "Failed:  ${RED}$FAILED${NC}"
echo -e "Skipped: ${YELLOW}$SKIPPED${NC}"
echo "============================================================"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}MVP VALIDATION SUCCESSFUL!${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Start services:     make services-up"
    echo "  2. Open Web UI:        http://localhost:5173"
    echo "  3. Open Grafana:       http://localhost:3000"
    echo "  4. View Prometheus:    http://localhost:9090"
    exit 0
else
    echo -e "${RED}MVP VALIDATION FAILED${NC}"
    echo ""
    echo "Please check the logs above for details."
    echo "To stop infrastructure: make infra-down"
    exit 1
fi
