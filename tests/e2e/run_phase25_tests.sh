#!/bin/bash
# Phase 2.5 Complete Test Suite
# Runs all Phase 2.5 testing phase tests

set -e

echo "=============================================="
echo "  Phase 2.5: Testing Phase 2 - Complete Suite"
echo "=============================================="
echo ""

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

pass() { echo -e "${GREEN}✓ PASS${NC}: $1"; }
fail() { echo -e "${RED}✗ FAIL${NC}: $1"; }
section() { echo -e "\n${BLUE}=== $1 ===${NC}\n"; }

# Track results
PASSED=0
FAILED=0

run_test() {
    local test_name="$1"
    local test_file="$2"
    
    section "$test_name"
    
    if bash "$test_file"; then
        ((PASSED++))
    else
        ((FAILED++))
        echo -e "${RED}Test $test_name failed!${NC}"
    fi
}

# Run all tests
run_test "2.5.3: Security Features E2E Tests" "$SCRIPT_DIR/security_e2e_test.sh"
run_test "2.5.4: Resource Monitoring Tests" "$SCRIPT_DIR/resource_monitoring_test.sh"
run_test "2.5.5: Backup/Restore Tests" "$SCRIPT_DIR/backup_restore_test.sh"
run_test "2.5.6: Idle Shutdown Tests" "$SCRIPT_DIR/idle_shutdown_test.sh"

# Summary
echo ""
echo "=============================================="
echo "  Test Summary"
echo "=============================================="
echo -e "${GREEN}Passed: $PASSED${NC}"
if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}Failed: 0${NC}"
fi

echo ""
echo "=============================================="
echo -e "${GREEN}  Phase 2.5: ALL TESTS PASSED${NC}"
echo "=============================================="
echo ""
echo "Phase 2 Complete: ✅"
echo ""
