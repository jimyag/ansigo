#!/bin/bash
# Integration Test Script for AnsiGo
# This script runs all playbook tests and verifies the output

set -e  # Exit on error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
TESTS_DIR="${PROJECT_ROOT}/tests"
PLAYBOOKS_DIR="${TESTS_DIR}/playbooks"
INVENTORY_FILE="${TESTS_DIR}/inventory/hosts.ini"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Check if ansigo-playbook is built
if [ ! -f "${PROJECT_ROOT}/bin/ansigo-playbook" ]; then
    echo -e "${YELLOW}Building ansigo-playbook...${NC}"
    cd "${PROJECT_ROOT}"
    go build -o bin/ansigo-playbook ./cmd/ansigo-playbook
fi

ANSIGO_PLAYBOOK="${PROJECT_ROOT}/bin/ansigo-playbook"

echo "=================================================="
echo "AnsiGo Integration Test Suite"
echo "=================================================="
echo ""

# Function to run a single test
run_test() {
    local test_name="$1"
    local playbook_file="$2"
    local expected_success="$3"  # "true" or "false"

    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    echo -n "Running: ${test_name}... "

    # Run the playbook and capture output
    if OUTPUT=$("${ANSIGO_PLAYBOOK}" -i "${INVENTORY_FILE}" "${playbook_file}" 2>&1); then
        if [ "${expected_success}" = "true" ]; then
            echo -e "${GREEN}✓ PASS${NC}"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            return 0
        else
            echo -e "${RED}✗ FAIL${NC} (Expected failure but succeeded)"
            TESTS_FAILED=$((TESTS_FAILED + 1))
            return 1
        fi
    else
        if [ "${expected_success}" = "false" ]; then
            echo -e "${GREEN}✓ PASS${NC} (Expected failure)"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            return 0
        else
            echo -e "${RED}✗ FAIL${NC}"
            echo "${OUTPUT}"
            TESTS_FAILED=$((TESTS_FAILED + 1))
            return 1
        fi
    fi
}

# Test 1: Basic Jinja2 working features
echo "Test Suite 1: Jinja2 Working Features"
echo "--------------------------------------"
run_test "Jinja2 Working Features" "${PLAYBOOKS_DIR}/test-jinja2-working.yml" "true"
echo ""

# Test 2: Jinja2 Filters
echo "Test Suite 2: Jinja2 Filters"
echo "-----------------------------"
run_test "Jinja2 Filters" "${PLAYBOOKS_DIR}/test-jinja2-filters.yml" "true"
echo ""

# Test 3: Jinja2 Loops
echo "Test Suite 3: Jinja2 Loops"
echo "--------------------------"
run_test "Jinja2 Loops" "${PLAYBOOKS_DIR}/test-jinja2-loops.yml" "true"
echo ""

# Test 4: Jinja2 Advanced
echo "Test Suite 4: Jinja2 Advanced"
echo "------------------------------"
run_test "Jinja2 Advanced" "${PLAYBOOKS_DIR}/test-jinja2-advanced.yml" "true"
echo ""

# Test 5: Basic Playbook
echo "Test Suite 5: Basic Playbook Features"
echo "--------------------------------------"
if [ -f "${PLAYBOOKS_DIR}/test-basic.yml" ]; then
    run_test "Basic Features" "${PLAYBOOKS_DIR}/test-basic.yml" "true"
fi
echo ""

# Test 6: Conditionals
echo "Test Suite 6: Conditionals"
echo "--------------------------"
if [ -f "${PLAYBOOKS_DIR}/test-conditionals.yml" ]; then
    run_test "Conditionals" "${PLAYBOOKS_DIR}/test-conditionals.yml" "true"
fi
echo ""

# Test 7: Core Modules
echo "Test Suite 7: Core Modules (ping, command, shell, copy)"
echo "--------------------------------------------------------"
if [ -f "${PLAYBOOKS_DIR}/test-modules.yml" ]; then
    run_test "Core Modules" "${PLAYBOOKS_DIR}/test-modules.yml" "true"
fi
echo ""

# Test 8: Variable Precedence
echo "Test Suite 8: Variable Precedence and Scoping"
echo "----------------------------------------------"
if [ -f "${PLAYBOOKS_DIR}/test-variable-precedence.yml" ]; then
    run_test "Variable Precedence" "${PLAYBOOKS_DIR}/test-variable-precedence.yml" "true"
fi
echo ""

# Test 9: Loops and Iteration
echo "Test Suite 9: Loops and Iteration"
echo "----------------------------------"
if [ -f "${PLAYBOOKS_DIR}/test-loops-iteration.yml" ]; then
    run_test "Loops and Iteration" "${PLAYBOOKS_DIR}/test-loops-iteration.yml" "true"
fi
echo ""

# Test 10: Error Handling
echo "Test Suite 10: Error Handling (ignore_errors)"
echo "----------------------------------------------"
if [ -f "${PLAYBOOKS_DIR}/test-error-handling.yml" ]; then
    run_test "Error Handling" "${PLAYBOOKS_DIR}/test-error-handling.yml" "true"
fi
echo ""

# Test 11: Advanced When Conditions
echo "Test Suite 11: Advanced When Conditions"
echo "----------------------------------------"
if [ -f "${PLAYBOOKS_DIR}/test-when-conditions.yml" ]; then
    run_test "When Conditions" "${PLAYBOOKS_DIR}/test-when-conditions.yml" "true"
fi
echo ""

# Test 12: Multi-Host Execution
echo "Test Suite 12: Multi-Host Concurrent Execution"
echo "-----------------------------------------------"
if [ -f "${PLAYBOOKS_DIR}/test-multi-host.yml" ]; then
    run_test "Multi-Host" "${PLAYBOOKS_DIR}/test-multi-host.yml" "true"
fi
echo ""

# Test 13: Module Arguments and Return Values
echo "Test Suite 13: Module Arguments and Return Values"
echo "--------------------------------------------------"
if [ -f "${PLAYBOOKS_DIR}/test-module-args.yml" ]; then
    run_test "Module Args" "${PLAYBOOKS_DIR}/test-module-args.yml" "true"
fi
echo ""

# Print summary
echo "=================================================="
echo "Test Summary"
echo "=================================================="
echo "Total Tests:  ${TESTS_TOTAL}"
echo -e "Passed:       ${GREEN}${TESTS_PASSED}${NC}"
if [ ${TESTS_FAILED} -gt 0 ]; then
    echo -e "Failed:       ${RED}${TESTS_FAILED}${NC}"
else
    echo -e "Failed:       ${TESTS_FAILED}"
fi
echo ""

# Exit with appropriate code
if [ ${TESTS_FAILED} -gt 0 ]; then
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
