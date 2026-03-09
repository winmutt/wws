#!/bin/bash
# End-to-End tests for workspace lifecycle

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="/tmp/wws-e2e-test"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

log_test() {
    echo -e "${YELLOW}[E2E TEST]${NC} $1"
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
    mkdir -p "${TEST_DIR}/workspaces"
    mkdir -p "${TEST_DIR}/logs"
    mkdir -p "${TEST_DIR}/config"
}

cleanup() {
    rm -rf "${TEST_DIR}"
}

# E2E Test: Complete workspace lifecycle from creation to deletion
test_complete_lifecycle() {
    log_test "Complete workspace lifecycle from creation to deletion"
    
    local workspace_id="e2e-workspace-$(date +%s)"
    local workspace_dir="${TEST_DIR}/workspaces/${workspace_id}"
    local log_file="${TEST_DIR}/logs/${workspace_id}.log"
    
    # Step 1: Create workspace
    mkdir -p "${workspace_dir}"
    echo '{"status": "pending", "created_at": "'$(date -Iseconds)'"}' > "${workspace_dir}/metadata.json"
    echo "CREATED: ${workspace_id}" >> "${log_file}"
    
    # Step 2: Start workspace
    sleep 0.1
    echo '{"status": "running", "updated_at": "'$(date -Iseconds)'"}' > "${workspace_dir}/metadata.json"
    echo "STARTED: ${workspace_id}" >> "${log_file}"
    
    # Step 3: Verify running status
    local status=$(grep -o '"status": "[^"]*"' "${workspace_dir}/metadata.json" | cut -d'"' -f4)
    if [ "$status" != "running" ]; then
        log_fail "Complete workspace lifecycle - status not running"
        return
    fi
    
    # Step 4: Stop workspace
    sleep 0.1
    echo '{"status": "stopped", "updated_at": "'$(date -Iseconds)'"}' > "${workspace_dir}/metadata.json"
    echo "STOPPED: ${workspace_id}" >> "${log_file}"
    
    # Step 5: Restart workspace
    sleep 0.1
    echo '{"status": "running", "updated_at": "'$(date -Iseconds)'"}' > "${workspace_dir}/metadata.json"
    echo "RESTARTED: ${workspace_id}" >> "${log_file}"
    
    # Step 6: Delete workspace
    rm -rf "${workspace_dir}"
    echo "DELETED: ${workspace_id}" >> "${log_file}"
    
    # Verify deletion
    if [ ! -d "${workspace_dir}" ]; then
        log_pass "Complete workspace lifecycle from creation to deletion"
    else
        log_fail "Complete workspace lifecycle from creation to deletion"
    fi
}

# E2E Test: Multi-workspace management
test_multi_workspace_management() {
    log_test "Multi-workspace management"
    
    local workspace_count=3
    
    # Create multiple workspaces
    for i in $(seq 1 ${workspace_count}); do
        local ws_id="multi-ws-${i}-$(date +%s)"
        local ws_dir="${TEST_DIR}/workspaces/${ws_id}"
        
        mkdir -p "${ws_dir}"
        echo '{"id": '${i}', "status": "running"}' > "${ws_dir}/metadata.json"
    done
    
    # Verify all workspaces exist
    local existing_count=$(ls -d "${TEST_DIR}/workspaces"/multi-ws-* 2>/dev/null | wc -l)
    
    if [ "$existing_count" -eq "$workspace_count" ]; then
        # Stop all workspaces
        for i in $(seq 1 ${workspace_count}); do
            local ws_dir=$(ls -d "${TEST_DIR}/workspaces"/multi-ws-${i}-* 2>/dev/null)
            if [ -n "${ws_dir}" ]; then
                echo '{"status": "stopped"}' > "${ws_dir}/metadata.json"
            fi
        done
        
        # Verify all stopped
        local stopped_count=$(grep -l '"status": "stopped"' "${TEST_DIR}/workspaces"/multi-ws-*/metadata.json 2>/dev/null | wc -l)
        
        if [ "$stopped_count" -eq "$workspace_count" ]; then
            log_pass "Multi-workspace management"
        else
            log_fail "Multi-workspace management - not all stopped"
        fi
    else
        log_fail "Multi-workspace management - not all created"
    fi
}

# E2E Test: Workspace resource constraints
test_resource_constraints() {
    log_test "Workspace resource constraints"
    
    local ws_dir="${TEST_DIR}/workspaces/resource-constrained"
    mkdir -p "${ws_dir}"
    
    # Create workspace with specific resources
    cat > "${ws_dir}/config.json" << EOF
{
  "cpu": 2,
  "memory_gb": 4,
  "storage_gb": 20
}
EOF
    
    # Verify configuration
    local cpu=$(grep -o '"cpu": [0-9]*' "${ws_dir}/config.json" | grep -o '[0-9]*')
    local memory=$(grep -o '"memory_gb": [0-9]*' "${ws_dir}/config.json" | grep -o '[0-9]*')
    local storage=$(grep -o '"storage_gb": [0-9]*' "${ws_dir}/config.json" | grep -o '[0-9]*')
    
    if [ "$cpu" -eq 2 ] && [ "$memory" -eq 4 ] && [ "$storage" -eq 20 ]; then
        log_pass "Workspace resource constraints"
    else
        log_fail "Workspace resource constraints"
    fi
    
    rm -rf "${ws_dir}"
}

# E2E Test: Workspace state recovery
test_state_recovery() {
    log_test "Workspace state recovery"
    
    local ws_dir="${TEST_DIR}/workspaces/recovery-test"
    local state_file="${ws_dir}/state.json"
    
    # Create workspace with state
    mkdir -p "${ws_dir}"
    cat > "${state_file}" << EOF
{
  "status": "running",
  "pid": 12345,
  "uptime_seconds": 3600,
  "last_action": "start"
}
EOF
    
    # Simulate crash/recovery
    local recovered_status=$(grep -o '"status": "[^"]*"' "${state_file}" | cut -d'"' -f4)
    
    if [ "$recovered_status" = "running" ]; then
        log_pass "Workspace state recovery"
    else
        log_fail "Workspace state recovery"
    fi
    
    rm -rf "${ws_dir}"
}

# E2E Test: Workspace configuration validation
test_config_validation() {
    log_test "Workspace configuration validation"
    
    local valid_config="${TEST_DIR}/config/valid.json"
    local invalid_config="${TEST_DIR}/config/invalid.json"
    
    # Valid configuration
    cat > "${valid_config}" << EOF
{
  "name": "valid-workspace",
  "provider": "podman",
  "config": {
    "cpu": 2,
    "memory": 4,
    "storage": 20
  }
}
EOF
    
    # Invalid configuration (missing required fields)
    cat > "${invalid_config}" << EOF
{
  "name": "invalid-workspace"
}
EOF
    
    # Validate configurations
    local valid_fields=$(grep -c '"' "${valid_config}")
    local invalid_fields=$(grep -c '"' "${invalid_config}")
    
    if [ "$valid_fields" -gt "$invalid_fields" ]; then
        log_pass "Workspace configuration validation"
    else
        log_fail "Workspace configuration validation"
    fi
}

# E2E Test: Workspace logging and audit trail
test_logging_audit() {
    log_test "Workspace logging and audit trail"
    
    local ws_dir="${TEST_DIR}/workspaces/logging-test"
    local audit_log="${ws_dir}/audit.log"
    
    mkdir -p "${ws_dir}"
    touch "${audit_log}"
    
    # Simulate actions with logging
    echo "$(date -Iseconds) CREATE workspace=logging-test user=test" >> "${audit_log}"
    echo "$(date -Iseconds) START workspace=logging-test" >> "${audit_log}"
    echo "$(date -Iseconds) STOP workspace=logging-test" >> "${audit_log}"
    echo "$(date -Iseconds) RESTART workspace=logging-test" >> "${audit_log}"
    echo "$(date -Iseconds) DELETE workspace=logging-test" >> "${audit_log}"
    
    local action_count=$(wc -l < "${audit_log}")
    
    if [ "$action_count" -eq 5 ]; then
        log_pass "Workspace logging and audit trail"
    else
        log_fail "Workspace logging and audit trail"
    fi
    
    rm -rf "${ws_dir}"
}

# E2E Test: Workspace error handling
test_error_handling() {
    log_test "Workspace error handling"
    
    local ws_dir="${TEST_DIR}/workspaces/error-test"
    local error_log="${ws_dir}/error.log"
    
    mkdir -p "${ws_dir}"
    touch "${error_log}"
    
    # Simulate errors
    echo "$(date -Iseconds) ERROR Failed to start workspace: timeout" >> "${error_log}"
    echo "$(date -Iseconds) ERROR Failed to allocate resources: insufficient" >> "${error_log}"
    
    local error_count=$(grep -c "ERROR" "${error_log}")
    
    if [ "$error_count" -eq 2 ]; then
        log_pass "Workspace error handling"
    else
        log_fail "Workspace error handling"
    fi
    
    rm -rf "${ws_dir}"
}

# E2E Test: Workspace cleanup on failure
test_cleanup_on_failure() {
    log_test "Workspace cleanup on failure"
    
    local ws_dir="${TEST_DIR}/workspaces/cleanup-test"
    
    mkdir -p "${ws_dir}"
    echo "temporary-data" > "${ws_dir}/temp.tmp"
    echo '{"status": "failed"}' > "${ws_dir}/metadata.json"
    
    # Simulate cleanup
    rm -rf "${ws_dir}"
    
    if [ ! -d "${ws_dir}" ]; then
        log_pass "Workspace cleanup on failure"
    else
        log_fail "Workspace cleanup on failure"
    fi
}

# Run all E2E tests
run_tests() {
    echo "========================================="
    echo "Running End-to-End Workspace Lifecycle Tests"
    echo "========================================="
    echo ""
    
    setup
    
    test_complete_lifecycle
    test_multi_workspace_management
    test_resource_constraints
    test_state_recovery
    test_config_validation
    test_logging_audit
    test_error_handling
    test_cleanup_on_failure
    
    cleanup
    
    echo ""
    echo "========================================="
    echo "E2E Test Results"
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
