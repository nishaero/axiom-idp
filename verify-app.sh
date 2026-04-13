#!/bin/bash
# Axiom IDP - Application Verification Script
# Tests that all components are working correctly

set -e

echo "=========================================="
echo "Axiom IDP - Application Verification"
echo "=========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to test
test_endpoint() {
    local name=$1
    local method=$2
    local url=$3
    local expected=$4
    
    echo -n "Testing $name... "
    result=$(curl -s -X $method "$url" 2>&1 || echo "FAILED")
    
    if echo "$result" | grep -q "$expected"; then
        echo -e "${GREEN}PASSED${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}FAILED${NC}"
        echo "  Expected: $expected"
        echo "  Got: $result"
        ((TESTS_FAILED++))
    fi
}

# Test Backend
echo "Testing Backend API (Port 8080)..."
echo ""

test_endpoint "Health Check" "GET" "http://localhost:8080/health" "status"
test_endpoint "Server is running" "GET" "http://localhost:8080/health" "ok"

echo ""
echo "Testing Frontend (Port 3000)..."
echo ""

test_endpoint "Frontend Loading" "GET" "http://localhost:3000/" "Axiom IDP"
test_endpoint "Frontend Assets" "GET" "http://localhost:3000/assets/index" "script"

echo ""
echo "=========================================="
echo "Test Results:"
echo "=========================================="
echo -e "Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Failed: ${RED}${TESTS_FAILED}${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo ""
    echo "Application is running successfully!"
    echo ""
    echo "Access the application:"
    echo "  Frontend: http://localhost:3000"
    echo "  Backend API: http://localhost:8080"
    echo "  Health Check: http://localhost:8080/health"
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    exit 1
fi
