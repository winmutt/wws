#!/bin/bash
# Unit tests for workspace agent bootstrap script

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BOOTSTRAP_SCRIPT="${SCRIPT_DIR}/bootstrap.sh"
TEST_DIR="/tmp/wws-bootstrap-test"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Logging
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

# Setup test environment
setup() {
    rm -rf "${TEST_DIR}"
    mkdir -p "${TEST_DIR}"
    mkdir -p "${TEST_DIR}/workspace-data"
}

# Cleanup test environment
cleanup() {
    rm -rf "${TEST_DIR}"
}

# Test: Check script exists and is executable
test_script_exists() {
    log_test "Bootstrap script exists and is executable"
    
    if [ -f "${BOOTSTRAP_SCRIPT}" ] && [ -x "${BOOTSTRAP_SCRIPT}" ]; then
        log_pass "Bootstrap script exists and is executable"
    else
        log_fail "Bootstrap script not found or not executable"
    fi
}

# Test: Check script has proper shebang
test_script_shebang() {
    log_test "Bootstrap script has proper shebang"
    
    if head -1 "${BOOTSTRAP_SCRIPT}" | grep -q "#!/bin/bash"; then
        log_pass "Bootstrap script has proper shebang"
    else
        log_fail "Bootstrap script missing proper shebang"
    fi
}

# Test: Check required functions exist
test_required_functions() {
    log_test "Bootstrap script contains required functions"
    
    REQUIRED_FUNCTIONS=(
        "check_requirements"
        "install_zsh"
        "install_yadm"
        "init_dotfiles"
        "install_gh_cli"
        "install_codeserver"
        "configure_ssh"
        "setup_persistent_storage"
    )
    
    ALL_FOUND=true
    for func in "${REQUIRED_FUNCTIONS}"; do
        if ! grep -q "^${func}()" "${BOOTSTRAP_SCRIPT}"; then
            log_fail "Missing function: ${func}"
            ALL_FOUND=false
        fi
    done
    
    if [ "${ALL_FOUND}" = true ]; then
        log_pass "All required functions found"
    fi
}

# Test: Check environment variable validation
test_env_validation() {
    log_test "Bootstrap script validates environment variables"
    
    if grep -q "GITHUB_USERNAME" "${BOOTSTRAP_SCRIPT}" && \
       grep -q "GITHUB_TOKEN" "${BOOTSTRAP_SCRIPT}" && \
       grep -q "error.*GITHUB_USERNAME" "${BOOTSTRAP_SCRIPT}" && \
       grep -q "error.*GITHUB_TOKEN" "${BOOTSTRAP_SCRIPT}"; then
        log_pass "Environment variable validation present"
    else
        log_fail "Missing environment variable validation"
    fi
}

# Test: Check logging function
test_logging() {
    log_test "Bootstrap script has logging function"
    
    if grep -q "^log()" "${BOOTSTRAP_SCRIPT}" && \
       grep -q "^error()" "${BOOTSTRAP_SCRIPT}"; then
        log_pass "Logging functions present"
    else
        log_fail "Missing logging functions"
    fi
}

# Test: Check error handling
test_error_handling() {
    log_test "Bootstrap script has error handling"
    
    if grep -q "set -e" "${BOOTSTRAP_SCRIPT}"; then
        log_pass "Error handling enabled (set -e)"
    else
        log_fail "Missing error handling"
    fi
}

# Test: Check OS detection
test_os_detection() {
    log_test "Bootstrap script detects OS distribution"
    
    if grep -q "/etc/debian_version" "${BOOTSTRAP_SCRIPT}" && \
       grep -q "/etc/redhat-release" "${BOOTSTRAP_SCRIPT}"; then
        log_pass "OS detection present"
    else
        log_fail "Missing OS detection"
    fi
}

# Test: Check Zsh installation
test_zsh_installation() {
    log_test "Bootstrap script installs Zsh"
    
    if grep -q "install_zsh" "${BOOTSTRAP_SCRIPT}" && \
       grep -q "apt-get install.*zsh" "${BOOTSTRAP_SCRIPT}" && \
       grep -q "yum install.*zsh" "${BOOTSTRAP_SCRIPT}"; then
        log_pass "Zsh installation logic present"
    else
        log_fail "Missing Zsh installation logic"
    fi
}

# Test: Check yadm installation
test_yadm_installation() {
    log_test "Bootstrap script installs yadm"
    
    if grep -q "install_yadm" "${BOOTSTRAP_SCRIPT}" && \
       grep -q "yadm" "${BOOTSTRAP_SCRIPT}"; then
        log_pass "yadm installation logic present"
    else
        log_fail "Missing yadm installation logic"
    fi
}

# Test: Check gh CLI installation
test_gh_cli_installation() {
    log_test "Bootstrap script installs gh CLI"
    
    if grep -q "install_gh_cli" "${BOOTSTRAP_SCRIPT}" && \
       grep -q "gh auth login" "${BOOTSTRAP_SCRIPT}"; then
        log_pass "gh CLI installation logic present"
    else
        log_fail "Missing gh CLI installation logic"
    fi
}

# Test: Check code-server installation
test_codeserver_installation() {
    log_test "Bootstrap script installs code-server"
    
    if grep -q "install_codeserver" "${BOOTSTRAP_SCRIPT}" && \
       grep -q "code-server" "${BOOTSTRAP_SCRIPT}"; then
        log_pass "code-server installation logic present"
    else
        log_fail "Missing code-server installation logic"
    fi
}

# Test: Check SSH configuration
test_ssh_configuration() {
    log_test "Bootstrap script configures SSH"
    
    if grep -q "configure_ssh" "${BOOTSTRAP_SCRIPT}" && \
       grep -q "ssh-keygen" "${BOOTSTRAP_SCRIPT}" && \
       grep -q "authorized_keys" "${BOOTSTRAP_SCRIPT}"; then
        log_pass "SSH configuration logic present"
    else
        log_fail "Missing SSH configuration logic"
    fi
}

# Test: Check persistent storage setup
test_persistent_storage() {
    log_test "Bootstrap script sets up persistent storage"
    
    if grep -q "setup_persistent_storage" "${BOOTSTRAP_SCRIPT}" && \
       grep -q "workspace-data" "${BOOTSTRAP_SCRIPT}"; then
        log_pass "Persistent storage setup logic present"
    else
        log_fail "Missing persistent storage setup logic"
    fi
}

# Test: Check dotfiles initialization
test_dotfiles_initialization() {
    log_test "Bootstrap script initializes dotfiles"
    
    if grep -q "init_dotfiles" "${BOOTSTRAP_SCRIPT}" && \
       grep -q "yadm clone" "${BOOTSTRAP_SCRIPT}"; then
        log_pass "Dotfiles initialization logic present"
    else
        log_fail "Missing dotfiles initialization logic"
    fi
}

# Run all tests
run_tests() {
    echo "========================================="
    echo "Running Bootstrap Script Unit Tests"
    echo "========================================="
    echo ""
    
    setup
    
    test_script_exists
    test_script_shebang
    test_required_functions
    test_env_validation
    test_logging
    test_error_handling
    test_os_detection
    test_zsh_installation
    test_yadm_installation
    test_gh_cli_installation
    test_codeserver_installation
    test_ssh_configuration
    test_persistent_storage
    test_dotfiles_initialization
    
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
