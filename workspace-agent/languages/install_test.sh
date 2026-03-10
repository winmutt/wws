#!/bin/bash
# Unit tests for language support module

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_SCRIPT="${SCRIPT_DIR}/install.sh"
TEST_DIR="/tmp/wws-language-test"

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
    export LANG_DIR="${TEST_DIR}/languages"
    export VERSIONS_DIR="${LANG_DIR}/versions"
    export CURRENT_DIR="${LANG_DIR}/current"
}

# Cleanup test environment
cleanup() {
    rm -rf "${TEST_DIR}"
}

# Test: Check script exists and is executable
test_script_exists() {
    log_test "Language install script exists and is executable"
    
    if [ -f "${INSTALL_SCRIPT}" ]; then
        log_pass "Language install script exists"
    else
        log_fail "Language install script not found"
    fi
}

# Test: Check script has proper shebang
test_script_shebang() {
    log_test "Language install script has proper shebang"
    
    if head -1 "${INSTALL_SCRIPT}" | grep -q "#!/bin/bash"; then
        log_pass "Language install script has proper shebang"
    else
        log_fail "Language install script missing proper shebang"
    fi
}

# Test: Check required functions exist
test_required_functions() {
    log_test "Language install script contains required functions"
    
    REQUIRED_FUNCTIONS=(
        "init_language_env"
        "update_path"
        "install_python"
        "install_node"
        "install_go"
        "install_rust"
        "install_python_packages"
        "install_node_packages"
        "install_go_packages"
        "install_rust_packages"
        "list_languages"
    )
    
    ALL_FOUND=true
    for func in "${REQUIRED_FUNCTIONS}"; do
        if ! grep -q "^${func}()" "${INSTALL_SCRIPT}"; then
            log_fail "Missing function: ${func}"
            ALL_FOUND=false
        fi
    done
    
    if [ "${ALL_FOUND}" = true ]; then
        log_pass "All required functions found"
    fi
}

# Test: Check language directories initialization
test_language_dirs() {
    log_test "Language install script creates language directories"
    
    if grep -q "LANG_DIR" "${INSTALL_SCRIPT}" && \
       grep -q "VERSIONS_DIR" "${INSTALL_SCRIPT}" && \
       grep -q "CURRENT_DIR" "${INSTALL_SCRIPT}"; then
        log_pass "Language directory configuration present"
    else
        log_fail "Missing language directory configuration"
    fi
}

# Test: Check Python installation
test_python_installation() {
    log_test "Language install script supports Python installation"
    
    if grep -q "install_python()" "${INSTALL_SCRIPT}" && \
       grep -q "python3" "${INSTALL_SCRIPT}" && \
       grep -q "pip" "${INSTALL_SCRIPT}"; then
        log_pass "Python installation logic present"
    else
        log_fail "Missing Python installation logic"
    fi
}

# Test: Check Node.js installation
test_node_installation() {
    log_test "Language install script supports Node.js installation"
    
    if grep -q "install_node()" "${INSTALL_SCRIPT}" && \
       grep -q "nvm\|n " "${INSTALL_SCRIPT}" && \
       grep -q "npm" "${INSTALL_SCRIPT}"; then
        log_pass "Node.js installation logic present"
    else
        log_fail "Missing Node.js installation logic"
    fi
}

# Test: Check Go installation
test_go_installation() {
    log_test "Language install script supports Go installation"
    
    if grep -q "install_go()" "${INSTALL_SCRIPT}" && \
       grep -q "go.dev" "${INSTALL_SCRIPT}" && \
       grep -q "gomod" "${INSTALL_SCRIPT}"; then
        log_pass "Go installation logic present"
    else
        log_fail "Missing Go installation logic"
    fi
}

# Test: Check Rust installation
test_rust_installation() {
    log_test "Language install script supports Rust installation"
    
    if grep -q "install_rust()" "${INSTALL_SCRIPT}" && \
       grep -q "rustup" "${INSTALL_SCRIPT}" && \
       grep -q "cargo" "${INSTALL_SCRIPT}"; then
        log_pass "Rust installation logic present"
    else
        log_fail "Missing Rust installation logic"
    fi
}

# Test: Check PATH configuration
test_path_configuration() {
    log_test "Language install script updates PATH"
    
    if grep -q "update_path()" "${INSTALL_SCRIPT}" && \
       grep -q "export PATH" "${INSTALL_SCRIPT}"; then
        log_pass "PATH configuration logic present"
    else
        log_fail "Missing PATH configuration logic"
    fi
}

# Test: Check version management
test_version_management() {
    log_test "Language install script supports version management"
    
    if grep -q "set_python_current" "${INSTALL_SCRIPT}" && \
       grep -q "set_node_current" "${INSTALL_SCRIPT}" && \
       grep -q "set_go_current" "${INSTALL_SCRIPT}" && \
       grep -q "set_rust_current" "${INSTALL_SCRIPT}"; then
        log_pass "Version management logic present"
    else
        log_fail "Missing version management logic"
    fi
}

# Test: Check package installation
test_package_installation() {
    log_test "Language install script supports package installation"
    
    if grep -q "install_python_packages" "${INSTALL_SCRIPT}" && \
       grep -q "install_node_packages" "${INSTALL_SCRIPT}" && \
       grep -q "install_go_packages" "${INSTALL_SCRIPT}" && \
       grep -q "install_rust_packages" "${INSTALL_SCRIPT}"; then
        log_pass "Package installation logic present"
    else
        log_fail "Missing package installation logic"
    fi
}

# Test: Check list languages function
test_list_languages() {
    log_test "Language install script can list installed languages"
    
    if grep -q "list_languages()" "${INSTALL_SCRIPT}" && \
       grep -q "Installed Languages" "${INSTALL_SCRIPT}"; then
        log_pass "List languages logic present"
    else
        log_fail "Missing list languages logic"
    fi
}

# Test: Check error handling
test_error_handling() {
    log_test "Language install script has error handling"
    
    if grep -q "set -e" "${INSTALL_SCRIPT}" && \
       grep -q "error()" "${INSTALL_SCRIPT}"; then
        log_pass "Error handling present"
    else
        log_fail "Missing error handling"
    fi
}

# Test: Check logging function
test_logging() {
    log_test "Language install script has logging function"
    
    if grep -q "^log()" "${INSTALL_SCRIPT}"; then
        log_pass "Logging function present"
    else
        log_fail "Missing logging function"
    fi
}

# Test: Check OS detection
test_os_detection() {
    log_test "Language install script detects OS distribution"
    
    if grep -q "/etc/debian_version" "${INSTALL_SCRIPT}" && \
       grep -q "/etc/redhat-release" "${INSTALL_SCRIPT}"; then
        log_pass "OS detection present"
    else
        log_fail "Missing OS detection"
    fi
}

# Test: Check help command
test_help_command() {
    log_test "Language install script has help command"
    
    if grep -q "help" "${INSTALL_SCRIPT}" && \
       grep -q "Usage:" "${INSTALL_SCRIPT}"; then
        log_pass "Help command present"
    else
        log_fail "Missing help command"
    fi
}

# Run all tests
run_tests() {
    echo "========================================="
    echo "Running Language Support Unit Tests"
    echo "========================================="
    echo ""
    
    setup
    
    test_script_exists
    test_script_shebang
    test_required_functions
    test_language_dirs
    test_python_installation
    test_node_installation
    test_go_installation
    test_rust_installation
    test_path_configuration
    test_version_management
    test_package_installation
    test_list_languages
    test_error_handling
    test_logging
    test_os_detection
    test_help_command
    
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
