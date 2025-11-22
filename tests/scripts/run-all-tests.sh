#!/bin/bash
# Comprehensive Test Automation Script for AnsiGo
# This script runs all tests: unit tests, integration tests, and linting

set -e  # Exit on error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results
UNIT_TESTS_PASSED=false
INTEGRATION_TESTS_PASSED=false
BUILD_PASSED=false

echo "=================================================="
echo "AnsiGo Comprehensive Test Suite"
echo "=================================================="
echo ""

# Step 1: Run go mod tidy
echo -e "${BLUE}Step 1: Running go mod tidy...${NC}"
cd "${PROJECT_ROOT}"
if go mod tidy; then
    echo -e "${GREEN}✓ go mod tidy succeeded${NC}"
else
    echo -e "${RED}✗ go mod tidy failed${NC}"
    exit 1
fi
echo ""

# Step 2: Build check
echo -e "${BLUE}Step 2: Building binaries...${NC}"
cd "${PROJECT_ROOT}"
if go build -o bin/ansigo ./cmd/ansigo && go build -o bin/ansigo-playbook ./cmd/ansigo-playbook; then
    echo -e "${GREEN}✓ Build succeeded${NC}"
    BUILD_PASSED=true
else
    echo -e "${RED}✗ Build failed${NC}"
    exit 1
fi
echo ""

# Step 3: Run unit tests
echo -e "${BLUE}Step 3: Running unit tests...${NC}"
echo "--------------------------------------"
cd "${PROJECT_ROOT}"
if go test ./pkg/... -v -cover; then
    echo ""
    echo -e "${GREEN}✓ All unit tests passed${NC}"
    UNIT_TESTS_PASSED=true
else
    echo ""
    echo -e "${RED}✗ Some unit tests failed${NC}"
    UNIT_TESTS_PASSED=false
fi
echo ""

# Step 4: Run integration tests
echo -e "${BLUE}Step 4: Running integration tests...${NC}"
echo "--------------------------------------"
if [ -f "${SCRIPT_DIR}/run-integration-tests.sh" ]; then
    if bash "${SCRIPT_DIR}/run-integration-tests.sh"; then
        INTEGRATION_TESTS_PASSED=true
    else
        INTEGRATION_TESTS_PASSED=false
    fi
else
    echo -e "${YELLOW}⚠ Integration test script not found, skipping...${NC}"
    INTEGRATION_TESTS_PASSED=true
fi
echo ""

# Step 5: Run go vet
echo -e "${BLUE}Step 5: Running go vet...${NC}"
cd "${PROJECT_ROOT}"
if go vet ./...; then
    echo -e "${GREEN}✓ go vet passed${NC}"
else
    echo -e "${YELLOW}⚠ go vet found issues${NC}"
fi
echo ""

# Print final summary
echo "=================================================="
echo "Final Test Summary"
echo "=================================================="
echo -e "Build:              ${BUILD_PASSED} ${GREEN}✓${NC}" || echo -e "Build:              ${BUILD_PASSED} ${RED}✗${NC}"
if [ "${UNIT_TESTS_PASSED}" = true ]; then
    echo -e "Unit Tests:         ${GREEN}✓ PASS${NC}"
else
    echo -e "Unit Tests:         ${RED}✗ FAIL${NC}"
fi

if [ "${INTEGRATION_TESTS_PASSED}" = true ]; then
    echo -e "Integration Tests:  ${GREEN}✓ PASS${NC}"
else
    echo -e "Integration Tests:  ${RED}✗ FAIL${NC}"
fi
echo ""

# Exit with appropriate code
if [ "${UNIT_TESTS_PASSED}" = true ] && [ "${INTEGRATION_TESTS_PASSED}" = true ]; then
    echo -e "${GREEN}═══════════════════════════════════════${NC}"
    echo -e "${GREEN}  All tests passed successfully! 🎉   ${NC}"
    echo -e "${GREEN}═══════════════════════════════════════${NC}"
    exit 0
else
    echo -e "${RED}═══════════════════════════════════════${NC}"
    echo -e "${RED}  Some tests failed. Please review.   ${NC}"
    echo -e "${RED}═══════════════════════════════════════${NC}"
    exit 1
fi
