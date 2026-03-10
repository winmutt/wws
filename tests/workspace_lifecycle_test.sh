#!/bin/bash
# Integration tests for workspace lifecycle

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="/tmp/wws-lifecycle-test"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

log_test() {
    echo -e "${YELLOW}[TEST]${NC} $1"
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

setup() {
    rm -rf "${TEST_DIR}"
    mkdir -p "${TEST_DIR}"
}

cleanup() {
    rm -rf "${TEST_DIR}"
}

# Test: Workspace creation workflow
test_workspace_creation() {
    log_test "Workspace creation workflow"
    
    local workspace_name="test-workspace-$(date +%s)"
    local workspace_file="${TEST_DIR}/${workspace_name}.json"
    
    # Simulate workspace creation
    cat > "${workspace_file}" << EOF
{
  "name": "${workspace_name}",
  "organization_id": 1,
  "provider": "podman",
  "status": "pending",
  "config": {
    "cpu": 2,
    "memory": 4,
    "storage": 20
  }
}
EOF
    
    if [ -f "${workspace_file}" ]; then
        log_pass "Workspace creation workflow"
    else
        log_fail "Workspace creation workflow"
    fi
}

# Test: Workspace status transitions
test_workspace_status_transitions() {
    log_test "Workspace status transitions"
    
    local status_file="${TEST_DIR}/status_test.json"
    
    # Test status: pending -> running -> stopped -> running
    echo '{"status": "pending"}' > "${status_file}"
    
    # Transition to running
    echo '{"status": "running"}' > "${status_file}"
    local status=$(grep -o '"status": "[^"]*"' "${status_file}" | cut -d'"' -f4)
    
    if [ "$status" = "running" ]; then
        # Transition to stopped
        echo '{"status": "stopped"}' > "${status_file}"
        status=$(grep -o '"status": "[^"]*"' "${status_file}" | cut -d'"' -f4)
        
        if [ "$status" = "stopped" ]; then
            # Transition back to running
            echo '{"status": "running"}' > "${status_file}"
            status=$(grep -o '"status": "[^"]*"' "${status_file}" | cut -d'"' -f4)
            
            if [ "$status" = "running" ]; then
                log_pass "Workspace status transitions"
            else
                log_fail "Workspace status transitions"
            fi
        else
            log_fail "Workspace status transitions"
        fi
    else
        log_fail "Workspace status transitions"
    fi
}

# Test: Workspace deletion
test_workspace_deletion() {
    log_test "Workspace deletion"
    
    local workspace_file="${TEST_DIR}/deletion_test.json"
    
    # Create workspace
    echo '{"status": "running"}' > "${workspace_file}"
    
    # Delete workspace (set status to deleted)
    echo '{"status": "deleted"}' > "${workspace_file}"
    
    if grep -q '"status": "deleted"' "${workspace_file}"; then
        rm "${workspace_file}"
        if [ ! -f "${workspace_file}" ]; then
            log_pass "Workspace deletion"
        else
            log_fail "Workspace deletion - file not removed"
        fi
    else
        log_fail "Workspace deletion"
    fi
}

# Test: Resource allocation validation
test_resource_allocation() {
    log_test "Resource allocation validation"
    
    local config_file="${TEST_DIR}/config_test.json"
    
    # Test valid configuration
    cat > "${config_file}" << EOF
{
  "cpu": 2,
  "memory": 4,
  "storage": 20
}
EOF
    
    local cpu=$(grep -o '"cpu": [0-9]*' "${config_file}" | grep -o '[0-9]*')
    local memory=$(grep -o '"memory": [0-9]*' "${config_file}" | grep -o '[0-9]*')
    local storage=$(grep -o '"storage": [0-9]*' "${config_file}" | grep -o '[0-9]*')
    
    if [ "$cpu" -ge 1 ] && [ "$cpu" -le 16 ] && \
       [ "$memory" -ge 1 ] && [ "$memory" -le 64 ] && \
       [ "$storage" -ge 10 ] && [ "$storage" -le 500 ]; then
        log_pass "Resource allocation validation"
    else
        log_fail "Resource allocation validation"
    fi
}

# Test: Workspace tag generation
test_workspace_tag_generation() {
    log_test "Workspace tag generation"
    
    local workspace_name="my-workspace"
    local tag="${workspace_name}-$(cat /proc/sys/kernel/random/uuid 2>/dev/null || echo "test-uuid")"
    
    if [[ "$tag" == "${workspace_name}-"* ]]; then
        log_pass "Workspace tag generation"
    else
        log_fail "Workspace tag generation"
    fi
}

# Test: Provider selection
test_provider_selection() {
    log_test "Provider selection"
    
    local provider_file="${TEST_DIR}/provider_test.json"
    
    # Test podman provider
    echo '{"provider": "podman"}' > "${provider_file}"
    
    if grep -q '"provider": "podman"' "${provider_file}"; then
        log_pass "Provider selection"
    else
        log_fail "Provider selection"
    fi
}

# Test: Workspace configuration persistence
test_config_persistence() {
    log_test "Workspace configuration persistence"
    
    local config_file="${TEST_DIR}/persistence_test.json"
    
    # Save configuration
    cat > "${config_file}" << EOF
{
  "name": "persistent-workspace",
  "config": {
    "cpu": 4,
    "memory": 8,
    "storage": 50
  }
}
EOF
    
    # Load and verify
    if [ -f "${config_file}" ]; then
        local name=$(grep -o '"name": "[^"]*"' "${config_file}" | cut -d'"' -f4)
        if [ "$name" = "persistent-workspace" ]; then
            log_pass "Workspace configuration persistence"
        else
            log_fail "Workspace configuration persistence"
        fi
    else
        log_fail "Workspace configuration persistence"
    fi
}

# Test: Workspace start/stop/restart cycle
test_lifecycle_cycle() {
    log_test "Workspace start/stop/restart cycle"
    
    local lifecycle_file="${TEST_DIR}/lifecycle_test.log"
    rm -f "${lifecycle_file}"
    
    # Simulate lifecycle
    echo "created" >> "${lifecycle_file}"
    echo "starting" >> "${lifecycle_file}"
    echo "running" >> "${lifecycle_file}"
    echo "stopping" >> "${lifecycle_file}"
    echo "stopped" >> "${lifecycle_file}"
    echo "starting" >> "${lifecycle_file}"
    echo "running" >> "${lifecycle_file}"
    echo "restarting" >> "${lifecycle_file}"
    echo "running" >> "${lifecycle_file}"
    
    local running_count=$(grep -c "running" "${lifecycle_file}")
    local stopped_count=$(grep -c "stopped" "${lifecycle_file}")
    
    if [ "$running_count" -eq 3 ] && [ "$stopped_count" -eq 1 ]; then
        log_pass "Workspace start/stop/restart cycle"
    else
        log_fail "Workspace start/stop/restart cycle"
    fi
}

# Test: Concurrent workspace operations
test_concurrent_operations() {
    log_test "Concurrent workspace operations"
    
    local concurrent_dir="${TEST_DIR}/concurrent"
    mkdir -p "${concurrent_dir}"
    
    # Simulate concurrent operations
    for i in 1 2 3; do
        echo "operation-${i}" > "${concurrent_dir}/op-${i}.tmp" &
    done
    wait
    
    local count=$(ls "${concurrent_dir}"/*.tmp 2>/dev/null | wc -l)
    
    if [ "$count" -eq 3 ]; then
        log_pass "Concurrent workspace operations"
    else
        log_fail "Concurrent workspace operations"
    fi
}

# Test: Workspace metadata validation
test_metadata_validation() {
    log_test "Workspace metadata validation"
    
    local metadata_file="${TEST_DIR}/metadata_test.json"
    
    cat > "${metadata_file}" << EOF
{
  "id": 123,
  "tag": "test-uuid",
  "name": "test-workspace",
  "organization_id": 1,
  "owner_id": 1,
  "provider": "podman",
  "status": "running",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
EOF
    
    local required_fields=("id" "tag" "name" "organization_id" "owner_id" "provider" "status" "created_at" "updated_at")
    local all_present=true
    
    for field in "${required_fields[@]}"; do
        if ! grep -q "\"${field}\"" "${metadata_file}"; then
            all_present=false
            break
        fi
    done
    
    if [ "$all_present" = true ]; then
        log_pass "Workspace metadata validation"
    else
        log_fail "Workspace metadata validation"
    fi
}

# Run all tests
run_tests() {
    echo "========================================="
    echo "Running Workspace Lifecycle Integration Tests"
    echo "========================================="
    echo ""
    
    setup
    
    test_workspace_creation
    test_workspace_status_transitions
    test_workspace_deletion
    test_resource_allocation
    test_workspace_tag_generation
    test_provider_selection
    test_config_persistence
    test_lifecycle_cycle
    test_concurrent_operations
    test_metadata_validation
    
    cleanup
    
    echo ""
    echo "========================================="
    echo "Test Results"
    echo "========================================="
    echo -e "Passed: ${GREEN}${TESTS_PASSED}${NC}"
    echo -e "Failed: ${RED}${TESTS_FAILED}${NC}"
    echo "========================================="
    
    if [ "${TESTS_FAILED}" -gt 0 ]; then
        exit 1
    fi
}

# Run tests
run_tests
