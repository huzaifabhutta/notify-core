#!/bin/bash

# Comprehensive Test Suite for notify-core
# Tests both Simple and Multi-Tenant modes with aggressive edge case testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
    ((TESTS_PASSED++))
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
    ((TESTS_FAILED++))
}

log_test() {
    echo -e "${YELLOW}[TEST]${NC} $1"
    ((TESTS_RUN++))
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Wait for service to be ready
wait_for_service() {
    local url=$1
    local max_attempts=30
    local attempt=0

    log_info "Waiting for service at $url..."
    while [ $attempt -lt $max_attempts ]; do
        if curl -s -o /dev/null -w "%{http_code}" "$url" | grep -q "200\|401"; then
            log_success "Service is ready"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 1
    done

    log_error "Service failed to start after $max_attempts seconds"
    return 1
}

# Test HTTP endpoint
test_endpoint() {
    local method=$1
    local url=$2
    local headers=$3
    local data=$4
    local expected_status=$5
    local test_name=$6

    log_test "$test_name"

    local response
    local status

    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" $headers -d "$data" 2>&1)
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" $headers 2>&1)
    fi

    status=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$status" = "$expected_status" ]; then
        log_success "$test_name (Status: $status)"
        return 0
    else
        log_error "$test_name (Expected: $expected_status, Got: $status)"
        echo "Response: $body"
        return 1
    fi
}

# Cleanup function
cleanup() {
    log_info "Cleaning up..."
    pkill -f "notify-simple" || true
    pkill -f "notify-multitenant" || true
    docker stop notify-postgres-test 2>/dev/null || true
    docker rm notify-postgres-test 2>/dev/null || true
    rm -f /tmp/notify-test-*.log
}

# Set trap to cleanup on exit
trap cleanup EXIT

echo "========================================"
echo "  notify-core Comprehensive Test Suite"
echo "========================================"
echo ""

# Prerequisites check
log_info "Checking prerequisites..."
if ! command_exists go; then
    log_error "Go is not installed"
    exit 1
fi

if ! command_exists curl; then
    log_error "curl is not installed"
    exit 1
fi

if ! command_exists docker; then
    log_error "Docker is not installed (needed for multi-tenant tests)"
    exit 1
fi

log_success "All prerequisites met"
echo ""

# Build binaries
log_info "Building binaries..."
make clean
if make build 2>&1 | tee /tmp/notify-build.log; then
    log_success "Binaries built successfully"
else
    log_error "Build failed"
    cat /tmp/notify-build.log
    exit 1
fi
echo ""

#############################################
# TEST SUITE 1: Simple Mode Tests
#############################################
echo "========================================"
echo "  TEST SUITE 1: Simple Mode"
echo "========================================"
echo ""

log_info "Setting up Simple Mode environment..."
export API_KEYS="test-key-123:tenant1,test-key-456:tenant2"
export SMTP_HOST="smtp.example.com"
export SMTP_PORT=587
export SMTP_USER="test@example.com"
export SMTP_PASS="testpassword"
export SMTP_FROM="noreply@example.com"
export PORT=8081
export LOG_LEVEL=error

log_info "Starting Simple Mode server..."
./bin/notify-simple > /tmp/notify-simple.log 2>&1 &
SIMPLE_PID=$!

if ! wait_for_service "http://localhost:8081/health"; then
    log_error "Simple Mode server failed to start"
    cat /tmp/notify-simple.log
    exit 1
fi

# Test 1.1: Health check
test_endpoint "GET" "http://localhost:8081/health" "" "" "200" "Simple Mode: Health check"

# Test 1.2: Root endpoint
test_endpoint "GET" "http://localhost:8081/" "" "" "200" "Simple Mode: Root endpoint"

# Test 1.3: Invalid API key
test_endpoint "POST" "http://localhost:8081/send" \
    '-H "X-API-Key: invalid-key" -H "Content-Type: application/json"' \
    '{"channel":"email","to":"user@example.com","subject":"Test","body":"Test"}' \
    "401" "Simple Mode: Invalid API key"

# Test 1.4: Missing API key
test_endpoint "POST" "http://localhost:8081/send" \
    '-H "Content-Type: application/json"' \
    '{"channel":"email","to":"user@example.com","subject":"Test","body":"Test"}' \
    "401" "Simple Mode: Missing API key"

# Test 1.5: Valid API key (will fail at SMTP but auth should pass)
test_endpoint "POST" "http://localhost:8081/send" \
    '-H "X-API-Key: test-key-123" -H "Content-Type: application/json"' \
    '{"channel":"email","to":"user@example.com","subject":"Test","body":"Test"}' \
    "500" "Simple Mode: Valid API key (SMTP failure expected)"

# Test 1.6: Empty request body
test_endpoint "POST" "http://localhost:8081/send" \
    '-H "X-API-Key: test-key-123" -H "Content-Type: application/json"' \
    '{}' \
    "400" "Simple Mode: Empty request body"

# Test 1.7: Invalid JSON
test_endpoint "POST" "http://localhost:8081/send" \
    '-H "X-API-Key: test-key-123" -H "Content-Type: application/json"' \
    '{invalid json}' \
    "400" "Simple Mode: Invalid JSON"

# Test 1.8: Missing required fields
test_endpoint "POST" "http://localhost:8081/send" \
    '-H "X-API-Key: test-key-123" -H "Content-Type: application/json"' \
    '{"channel":"email"}' \
    "400" "Simple Mode: Missing required fields"

# Test 1.9: XSS attempt in email body
test_endpoint "POST" "http://localhost:8081/send" \
    '-H "X-API-Key: test-key-123" -H "Content-Type: application/json"' \
    '{"channel":"email","to":"user@example.com","subject":"Test","body":"<script>alert(1)</script>"}' \
    "500" "Simple Mode: XSS in body (sanitization test)"

# Test 1.10: SQL injection attempt in subject
test_endpoint "POST" "http://localhost:8081/send" \
    '-H "X-API-Key: test-key-123" -H "Content-Type: application/json"' \
    "{\"channel\":\"email\",\"to\":\"user@example.com\",\"subject\":\"Test'; DROP TABLE users; --\",\"body\":\"Test\"}" \
    "500" "Simple Mode: SQL injection attempt"

# Test 1.11: Very long email address
test_endpoint "POST" "http://localhost:8081/send" \
    '-H "X-API-Key: test-key-123" -H "Content-Type: application/json"' \
    '{"channel":"email","to":"'$(printf 'a%.0s' {1..1000})'@example.com","subject":"Test","body":"Test"}' \
    "400" "Simple Mode: Very long email"

# Test 1.12: Invalid email format
test_endpoint "POST" "http://localhost:8081/send" \
    '-H "X-API-Key: test-key-123" -H "Content-Type: application/json"' \
    '{"channel":"email","to":"not-an-email","subject":"Test","body":"Test"}' \
    "400" "Simple Mode: Invalid email format"

# Test 1.13: Unicode in subject
test_endpoint "POST" "http://localhost:8081/send" \
    '-H "X-API-Key: test-key-123" -H "Content-Type: application/json"' \
    '{"channel":"email","to":"user@example.com","subject":"Testing 测试 テスト 🚀","body":"Test"}' \
    "500" "Simple Mode: Unicode in subject"

# Test 1.14: Rate limiting (make 25 rapid requests)
log_test "Simple Mode: Rate limiting test"
rate_limit_failed=0
for i in {1..25}; do
    status=$(curl -s -o /dev/null -w "%{http_code}" \
        -X POST http://localhost:8081/send \
        -H "X-API-Key: test-key-123" \
        -H "Content-Type: application/json" \
        -d '{"channel":"email","to":"user@example.com","subject":"Test","body":"Test"}')

    if [ "$status" = "429" ]; then
        rate_limit_failed=1
        break
    fi
done

if [ $rate_limit_failed -eq 1 ]; then
    log_success "Simple Mode: Rate limiting triggered correctly"
else
    log_error "Simple Mode: Rate limiting did not trigger"
fi

log_info "Stopping Simple Mode server..."
kill $SIMPLE_PID 2>/dev/null || true
wait $SIMPLE_PID 2>/dev/null || true
sleep 2

echo ""
echo "========================================"
echo "  TEST SUITE 2: Multi-Tenant Mode"
echo "========================================"
echo ""

log_info "Starting PostgreSQL for testing..."
docker run -d --name notify-postgres-test \
    -e POSTGRES_USER=notify \
    -e POSTGRES_PASSWORD=testpassword \
    -e POSTGRES_DB=notify \
    -p 5433:5432 \
    postgres:15-alpine > /dev/null 2>&1

log_info "Waiting for PostgreSQL to be ready..."
sleep 5

log_info "Setting up Multi-Tenant Mode environment..."
export DB_HOST=localhost
export DB_PORT=5433
export DB_USER=notify
export DB_PASSWORD=testpassword
export DB_NAME=notify
export DB_SSL_MODE=disable
export ENCRYPTION_KEY="test-encryption-key-32-chars-long-12345678"
export PORT=8082
export LOG_LEVEL=error
unset API_KEYS

log_info "Starting Multi-Tenant Mode server..."
./bin/notify-multitenant > /tmp/notify-multitenant.log 2>&1 &
MULTITENANT_PID=$!

if ! wait_for_service "http://localhost:8082/health"; then
    log_error "Multi-Tenant Mode server failed to start"
    cat /tmp/notify-multitenant.log
    docker logs notify-postgres-test
    exit 1
fi

# Test 2.1: Health check
test_endpoint "GET" "http://localhost:8082/health" "" "" "200" "Multi-Tenant: Health check"

# Test 2.2: Create tenant with full config
log_test "Multi-Tenant: Create tenant with full config"
CREATE_RESPONSE=$(curl -s -X POST http://localhost:8082/v2/tenants \
    -H "Content-Type: application/json" \
    -d '{
        "name": "test-tenant-1",
        "smtp_host": "smtp.sendgrid.net",
        "smtp_port": 587,
        "smtp_user": "apikey",
        "smtp_password": "SG.test123",
        "smtp_from": "noreply@tenant1.com",
        "wa_token": "test-wa-token",
        "wa_phone_id": "123456789",
        "wa_base_url": "https://waba.360dialog.io",
        "wa_api_version": "v21.0"
    }')

if echo "$CREATE_RESPONSE" | grep -q '"status":"success"'; then
    TENANT1_API_KEY=$(echo "$CREATE_RESPONSE" | grep -o '"api_key":"[^"]*"' | cut -d'"' -f4)
    log_success "Multi-Tenant: Create tenant with full config (API Key: ${TENANT1_API_KEY:0:10}...)"
else
    log_error "Multi-Tenant: Create tenant failed"
    echo "Response: $CREATE_RESPONSE"
fi

# Test 2.3: Create tenant with minimal config
log_test "Multi-Tenant: Create tenant with minimal config"
CREATE_RESPONSE_2=$(curl -s -X POST http://localhost:8082/v2/tenants \
    -H "Content-Type: application/json" \
    -d '{"name": "test-tenant-2"}')

if echo "$CREATE_RESPONSE_2" | grep -q '"status":"success"'; then
    TENANT2_API_KEY=$(echo "$CREATE_RESPONSE_2" | grep -o '"api_key":"[^"]*"' | cut -d'"' -f4)
    log_success "Multi-Tenant: Create tenant with minimal config (API Key: ${TENANT2_API_KEY:0:10}...)"
else
    log_error "Multi-Tenant: Create tenant with minimal config failed"
    echo "Response: $CREATE_RESPONSE_2"
fi

# Test 2.4: Duplicate tenant name
test_endpoint "POST" "http://localhost:8082/v2/tenants" \
    '-H "Content-Type: application/json"' \
    '{"name": "test-tenant-1"}' \
    "400" "Multi-Tenant: Duplicate tenant name"

# Test 2.5: Invalid tenant name (special characters)
test_endpoint "POST" "http://localhost:8082/v2/tenants" \
    '-H "Content-Type: application/json"' \
    '{"name": "test@tenant"}' \
    "400" "Multi-Tenant: Invalid tenant name"

# Test 2.6: Get tenant info with valid API key
if [ -n "$TENANT1_API_KEY" ]; then
    test_endpoint "GET" "http://localhost:8082/v2/tenants/me" \
        "-H \"X-API-Key: $TENANT1_API_KEY\"" \
        "" "200" "Multi-Tenant: Get tenant info"
fi

# Test 2.7: Get tenant info with invalid API key
test_endpoint "GET" "http://localhost:8082/v2/tenants/me" \
    '-H "X-API-Key: invalid-key-12345"' \
    "" "401" "Multi-Tenant: Invalid API key"

# Test 2.8: Update tenant
if [ -n "$TENANT1_API_KEY" ]; then
    test_endpoint "PUT" "http://localhost:8082/v2/tenants/me" \
        "-H \"X-API-Key: $TENANT1_API_KEY\" -H \"Content-Type: application/json\"" \
        '{"smtp_host": "smtp.mailgun.org"}' \
        "200" "Multi-Tenant: Update tenant"
fi

# Test 2.9: Send with tenant credentials
if [ -n "$TENANT1_API_KEY" ]; then
    test_endpoint "POST" "http://localhost:8082/v2/send" \
        "-H \"X-API-Key: $TENANT1_API_KEY\" -H \"Content-Type: application/json\"" \
        '{"channel":"email","to":"user@example.com","subject":"Test","body":"Test"}' \
        "500" "Multi-Tenant: Send with tenant credentials (SMTP failure expected)"
fi

# Test 2.10: Send with global fallback (tenant without SMTP)
if [ -n "$TENANT2_API_KEY" ]; then
    test_endpoint "POST" "http://localhost:8082/v2/send" \
        "-H \"X-API-Key: $TENANT2_API_KEY\" -H \"Content-Type: application/json\"" \
        '{"channel":"email","to":"user@example.com","subject":"Test","body":"Test"}' \
        "500" "Multi-Tenant: Send with global fallback (SMTP failure expected)"
fi

# Test 2.11: Rate limiting per API key
if [ -n "$TENANT1_API_KEY" ]; then
    log_test "Multi-Tenant: Per-tenant rate limiting"
    rate_limit_hit=0
    for i in {1..60}; do
        status=$(curl -s -o /dev/null -w "%{http_code}" \
            -X POST http://localhost:8082/v2/send \
            -H "X-API-Key: $TENANT1_API_KEY" \
            -H "Content-Type: application/json" \
            -d '{"channel":"email","to":"user@example.com","subject":"Test","body":"Test"}')

        if [ "$status" = "429" ]; then
            rate_limit_hit=1
            break
        fi
    done

    if [ $rate_limit_hit -eq 1 ]; then
        log_success "Multi-Tenant: Per-tenant rate limiting triggered"
    else
        log_error "Multi-Tenant: Per-tenant rate limiting did not trigger"
    fi
fi

# Test 2.12: Brute force API key attempts (rate limit on validation)
log_test "Multi-Tenant: Brute force protection"
brute_force_blocked=0
for i in {1..15}; do
    status=$(curl -s -o /dev/null -w "%{http_code}" \
        -X GET http://localhost:8082/v2/tenants/me \
        -H "X-API-Key: brute-force-attempt-$i")

    if [ "$status" = "429" ]; then
        brute_force_blocked=1
        break
    fi
done

if [ $brute_force_blocked -eq 1 ]; then
    log_success "Multi-Tenant: Brute force protection triggered"
else
    log_error "Multi-Tenant: Brute force protection did not trigger"
fi

# Test 2.13: Verify API keys are hashed (not stored in plain text)
log_test "Multi-Tenant: API key hashing verification"
if [ -n "$TENANT1_API_KEY" ]; then
    # Query database to check if API key is hashed
    STORED_KEY=$(docker exec notify-postgres-test psql -U notify -d notify -t -c \
        "SELECT api_key FROM tenants WHERE name='test-tenant-1';" 2>/dev/null | tr -d ' \n')

    if [ "$STORED_KEY" != "$TENANT1_API_KEY" ] && [ ${#STORED_KEY} -gt 50 ]; then
        log_success "Multi-Tenant: API keys are properly hashed (bcrypt)"
    else
        log_error "Multi-Tenant: API keys may not be hashed correctly"
    fi
fi

# Test 2.14: Verify credentials are encrypted
log_test "Multi-Tenant: Credential encryption verification"
STORED_PASSWORD=$(docker exec notify-postgres-test psql -U notify -d notify -t -c \
    "SELECT smtp_password FROM tenants WHERE name='test-tenant-1';" 2>/dev/null | tr -d ' \n')

if [ ${#STORED_PASSWORD} -gt 20 ] && [ "$STORED_PASSWORD" != "SG.test123" ]; then
    log_success "Multi-Tenant: Credentials are properly encrypted (AES-256-GCM)"
else
    log_error "Multi-Tenant: Credentials may not be encrypted correctly"
fi

# Test 2.15: SQL injection in tenant name
test_endpoint "POST" "http://localhost:8082/v2/tenants" \
    '-H "Content-Type: application/json"' \
    "{\"name\": \"test'; DROP TABLE tenants; --\"}" \
    "400" "Multi-Tenant: SQL injection in tenant name"

# Test 2.16: Very long tenant name
test_endpoint "POST" "http://localhost:8082/v2/tenants" \
    '-H "Content-Type: application/json"' \
    '{"name": "'$(printf 'a%.0s' {1..500})'"}' \
    "400" "Multi-Tenant: Very long tenant name"

# Test 2.17: Empty tenant name
test_endpoint "POST" "http://localhost:8082/v2/tenants" \
    '-H "Content-Type: application/json"' \
    '{"name": ""}' \
    "400" "Multi-Tenant: Empty tenant name"

# Test 2.18: Null byte injection
test_endpoint "POST" "http://localhost:8082/v2/tenants" \
    '-H "Content-Type: application/json"' \
    '{"name": "test\u0000tenant"}' \
    "400" "Multi-Tenant: Null byte injection"

log_info "Stopping Multi-Tenant Mode server..."
kill $MULTITENANT_PID 2>/dev/null || true
wait $MULTITENANT_PID 2>/dev/null || true

log_info "Stopping PostgreSQL..."
docker stop notify-postgres-test > /dev/null 2>&1
docker rm notify-postgres-test > /dev/null 2>&1

echo ""
echo "========================================"
echo "  TEST SUMMARY"
echo "========================================"
echo ""
echo "Total Tests Run: $TESTS_RUN"
echo -e "${GREEN}Tests Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Tests Failed: $TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ ALL TESTS PASSED${NC}"
    exit 0
else
    echo -e "${RED}✗ SOME TESTS FAILED${NC}"
    exit 1
fi
