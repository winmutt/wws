#!/bin/bash
# WWS Language Support Module
# Handles installation and configuration of programming languages

set -e

# Configuration
LANG_DIR="${HOME}/.wws/languages"
VERSIONS_DIR="${LANG_DIR}/versions"
CURRENT_DIR="${LANG_DIR}/current"

# Logging
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [LANG] $1"
}

error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [LANG] ERROR: $1" >&2
    exit 1
}

# Initialize language environment
init_language_env() {
    log "Initializing language environment..."
    
    mkdir -p "${LANG_DIR}"
    mkdir -p "${VERSIONS_DIR}"
    mkdir -p "${CURRENT_DIR}"
    
    # Create language-specific directories
    mkdir -p "${CURRENT_DIR}/python"
    mkdir -p "${CURRENT_DIR}/node"
    mkdir -p "${CURRENT_DIR}/go"
    mkdir -p "${CURRENT_DIR}/rust"
    
    log "Language environment initialized"
}

# Update PATH for current session
update_path() {
    log "Updating PATH..."
    
    local new_path=""
    
    # Add Python to PATH
    if [ -d "${CURRENT_DIR}/python/bin" ]; then
        new_path="${new_path}:${CURRENT_DIR}/python/bin"
    fi
    
    # Add Node.js to PATH
    if [ -d "${CURRENT_DIR}/node/bin" ]; then
        new_path="${new_path}:${CURRENT_DIR}/node/bin"
    fi
    
    # Add Go to PATH
    if [ -d "${CURRENT_DIR}/go/bin" ]; then
        new_path="${new_path}:${CURRENT_DIR}/go/bin"
    fi
    
    # Add Rust to PATH
    if [ -d "${CURRENT_DIR}/rust/bin" ]; then
        new_path="${new_path}:${CURRENT_DIR}/rust/bin"
    fi
    
    # Export updated PATH
    if [ -n "$new_path" ]; then
        export PATH="${new_path}:${PATH}"
        log "PATH updated for current session"
    fi
    
    # Persist PATH in shell config
    local path_line="export PATH=\"${LANG_DIR}/current/\${LANG}:\\\$PATH\""
    local shell_config=""
    
    if [ -n "$ZSH_VERSION" ]; then
        shell_config="$HOME/.zshrc"
    elif [ -n "$BASH_VERSION" ]; then
        shell_config="$HOME/.bashrc"
    fi
    
    if [ -n "$shell_config" ] && ! grep -q "wws languages" "$shell_config" 2>/dev/null; then
        cat >> "$shell_config" << EOF

# WWS Language Support
export PATH="${LANG_DIR}/current/\\\$LANG:\\\$PATH"
EOF
        log "PATH configuration persisted to $shell_config"
    fi
}

# Python installation and management
install_python() {
    local version="${1:-3.11}"
    
    log "Installing Python ${version}..."
    
    local python_dir="${VERSIONS_DIR}/python/${version}"
    
    # Check if already installed
    if [ -f "${python_dir}/bin/python3" ]; then
        log "Python ${version} is already installed"
        set_python_current "$version"
        return 0
    fi
    
    # Install Python using pyenv or system package
    if command -v pyenv &> /dev/null; then
        pyenv install "${version}" 2>/dev/null || {
            error "Failed to install Python ${version} via pyenv"
        }
        ln -sf "${HOME}/.pyenv/versions/${version}" "${python_dir}"
    else
        # Install via system packages
        if [ -f /etc/debian_version ]; then
            apt-get update && apt-get install -y python${version} python${version}-venv python${version}-dev
            ln -sf "/usr/bin/python${version}" "${python_dir}/bin/python3"
        elif [ -f /etc/redhat-release ]; then
            yum install -y python${version} python${version}-devel
            ln -sf "/usr/bin/python${version}" "${python_dir}/bin/python3"
        else
            error "Unsupported OS for Python installation"
        fi
    fi
    
    # Create symlink to current
    set_python_current "$version"
    
    log "Python ${version} installed successfully"
}

set_python_current() {
    local version="$1"
    rm -f "${CURRENT_DIR}/python"
    ln -s "${VERSIONS_DIR}/python/${version}" "${CURRENT_DIR}/python"
    log "Python ${version} set as current"
}

get_python_current() {
    if [ -L "${CURRENT_DIR}/python" ]; then
        basename "$(readlink "${CURRENT_DIR}/python")"
    else
        echo "none"
    fi
}

# Node.js installation and management
install_node() {
    local version="${1:-20}"
    
    log "Installing Node.js ${version}..."
    
    local node_dir="${VERSIONS_DIR}/node/${version}"
    
    # Check if already installed
    if [ -f "${node_dir}/bin/node" ]; then
        log "Node.js ${version} is already installed"
        set_node_current "$version"
        return 0
    fi
    
    # Install Node.js using nvm or n
    if command -v nvm &> /dev/null; then
        nvm install "${version}"
        nvm use "${version}"
        # Get actual installation path
        local nvm_dir="${HOME}/.nvm/versions/node/v${version}"
        ln -sf "${nvm_dir}" "${node_dir}"
    elif command -v n &> /dev/null; then
        n "${version}"
        # Get actual installation path
        local node_path=$(which node)
        local actual_version=$(node --version | sed 's/v//')
        ln -sf "$(dirname "$(dirname "${node_path}")")" "${node_dir}"
    else
        # Install via system packages
        if [ -f /etc/debian_version ]; then
            curl -fsSL https://deb.nodesource.com/setup_${version}.x | bash -
            apt-get install -y nodejs
        elif [ -f /etc/redhat-release ]; then
            curl -fsSL https://rpm.nodesource.com/setup_${version}.x | bash -
            yum install -y nodejs
        else
            error "Unsupported OS for Node.js installation"
        fi
    fi
    
    # Create symlink to current
    set_node_current "$version"
    
    log "Node.js ${version} installed successfully"
}

set_node_current() {
    local version="$1"
    rm -f "${CURRENT_DIR}/node"
    ln -s "${VERSIONS_DIR}/node/${version}" "${CURRENT_DIR}/node"
    log "Node.js ${version} set as current"
}

get_node_current() {
    if [ -L "${CURRENT_DIR}/node" ]; then
        basename "$(readlink "${CURRENT_DIR}/node")"
    else
        echo "none"
    fi
}

# Go installation and management
install_go() {
    local version="${1:-1.21}"
    
    log "Installing Go ${version}..."
    
    local go_dir="${VERSIONS_DIR}/go/${version}"
    
    # Check if already installed
    if [ -f "${go_dir}/bin/go" ]; then
        log "Go ${version} is already installed"
        set_go_current "$version"
        return 0
    fi
    
    # Download and install Go
    local go_url="https://go.dev/dl/go${version}.linux-amd64.tar.gz"
    local go_tmp="/tmp/go${version}.tar.gz"
    
    log "Downloading Go ${version}..."
    curl -L -o "${go_tmp}" "${go_url}"
    
    log "Extracting Go ${version}..."
    rm -rf "${go_dir}"
    mkdir -p "${go_dir}"
    tar -C "${go_dir}" -xzf "${go_tmp}" --strip-components=1
    
    rm -f "${go_tmp}"
    
    # Create symlink to current
    set_go_current "$version"
    
    log "Go ${version} installed successfully"
}

set_go_current() {
    local version="$1"
    rm -f "${CURRENT_DIR}/go"
    ln -s "${VERSIONS_DIR}/go/${version}" "${CURRENT_DIR}/go"
    log "Go ${version} set as current"
}

get_go_current() {
    if [ -L "${CURRENT_DIR}/go" ]; then
        basename "$(readlink "${CURRENT_DIR}/go")"
    else
        echo "none"
    fi
}

# Rust installation and management
install_rust() {
    local version="${1:-stable}"
    
    log "Installing Rust ${version}..."
    
    local rust_dir="${VERSIONS_DIR}/rust/${version}"
    
    # Check if already installed
    if [ -f "${rust_dir}/bin/cargo" ]; then
        log "Rust ${version} is already installed"
        set_rust_current "$version"
        return 0
    fi
    
    # Install Rust using rustup
    if command -v rustup &> /dev/null; then
        rustup install "${version}"
        rustup default "${version}"
        # Get actual installation path
        local rustup_home="${HOME}/.rustup"
        local toolchain_dir=$(rustup toolchain list | grep "${version}" | awk '{print $1}')
        ln -sf "${rustup_home}/toolchains/${toolchain_dir}" "${rust_dir}"
    else
        # Install rustup
        curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
        source "$HOME/.cargo/env"
        rustup install "${version}"
        rustup default "${version}"
    fi
    
    # Create symlink to current
    set_rust_current "$version"
    
    log "Rust ${version} installed successfully"
}

set_rust_current() {
    local version="$1"
    rm -f "${CURRENT_DIR}/rust"
    ln -s "${VERSIONS_DIR}/rust/${version}" "${CURRENT_DIR}/rust"
    log "Rust ${version} set as current"
}

get_rust_current() {
    if [ -L "${CURRENT_DIR}/rust" ]; then
        basename "$(readlink "${CURRENT_DIR}/rust")"
    else
        echo "none"
    fi
}

# Install Python packages
install_python_packages() {
    shift # Remove function name from args
    local packages=("$@")
    
    if [ ${#packages[@]} -eq 0 ]; then
        error "No Python packages specified"
    fi
    
    local python=$(which python3)
    if [ -z "$python" ]; then
        error "Python not found in PATH"
    fi
    
    log "Installing Python packages: ${packages[*]}"
    
    # Upgrade pip first
    "${python}" -m pip install --upgrade pip
    
    # Install packages
    "${python}" -m pip install "${packages[@]}"
    
    log "Python packages installed successfully"
}

# Install Node.js packages
install_node_packages() {
    shift # Remove function name from args
    local packages=("$@")
    
    if [ ${#packages[@]} -eq 0 ]; then
        error "No Node.js packages specified"
    fi
    
    local npm=$(which npm)
    if [ -z "$npm" ]; then
        error "npm not found in PATH"
    fi
    
    log "Installing Node.js packages: ${packages[*]}"
    
    # Install packages globally
    npm install -g "${packages[@]}"
    
    log "Node.js packages installed successfully"
}

# Install Go packages
install_go_packages() {
    shift # Remove function name from args
    local packages=("$@")
    
    if [ ${#packages[@]} -eq 0 ]; then
        error "No Go packages specified"
    fi
    
    local go=$(which go)
    if [ -z "$go" ]; then
        error "Go not found in PATH"
    fi
    
    log "Installing Go packages: ${packages[*]}"
    
    # Install packages
    for pkg in "${packages[@]}"; do
        go install "${pkg}"
    done
    
    log "Go packages installed successfully"
}

# Install Rust packages
install_rust_packages() {
    shift # Remove function name from args
    local packages=("$@")
    
    if [ ${#packages[@]} -eq 0 ]; then
        error "No Rust packages specified"
    fi
    
    local cargo=$(which cargo)
    if [ -z "$cargo" ]; then
        error "cargo not found in PATH"
    fi
    
    log "Installing Rust packages: ${packages[*]}"
    
    # Install packages
    for pkg in "${packages[@]}"; do
        cargo install "${pkg}"
    done
    
    log "Rust packages installed successfully"
}

# List installed languages
list_languages() {
    echo "Installed Languages:"
    echo "===================="
    
    echo -e "\nPython:"
    if [ -d "${VERSIONS_DIR}/python" ]; then
        ls -1 "${VERSIONS_DIR}/python" | while read version; do
            if [ "$(get_python_current)" = "$version" ]; then
                echo "  ✓ ${version} (current)"
            else
                echo "  ${version}"
            fi
        done
    else
        echo "  None installed"
    fi
    
    echo -e "\nNode.js:"
    if [ -d "${VERSIONS_DIR}/node" ]; then
        ls -1 "${VERSIONS_DIR}/node" | while read version; do
            if [ "$(get_node_current)" = "$version" ]; then
                echo "  ✓ ${version} (current)"
            else
                echo "  ${version}"
            fi
        done
    else
        echo "  None installed"
    fi
    
    echo -e "\nGo:"
    if [ -d "${VERSIONS_DIR}/go" ]; then
        ls -1 "${VERSIONS_DIR}/go" | while read version; do
            if [ "$(get_go_current)" = "$version" ]; then
                echo "  ✓ ${version} (current)"
            else
                echo "  ${version}"
            fi
        done
    else
        echo "  None installed"
    fi
    
    echo -e "\nRust:"
    if [ -d "${VERSIONS_DIR}/rust" ]; then
        ls -1 "${VERSIONS_DIR}/rust" | while read version; do
            if [ "$(get_rust_current)" = "$version" ]; then
                echo "  ✓ ${version} (current)"
            else
                echo "  ${version}"
            fi
        done
    else
        echo "  None installed"
    fi
}

# Main execution
main() {
    local command="${1:-help}"
    shift || true
    
    case "${command}" in
        init)
            init_language_env
            update_path
            ;;
        python)
            install_python "$@"
            ;;
        node)
            install_node "$@"
            ;;
        go)
            install_go "$@"
            ;;
        rust)
            install_rust "$@"
            ;;
        pip)
            install_python_packages "$@"
            ;;
        npm)
            install_node_packages "$@"
            ;;
        gomod)
            install_go_packages "$@"
            ;;
        cargo)
            install_rust_packages "$@"
            ;;
        list)
            list_languages
            ;;
        path)
            update_path
            ;;
        help|*)
            echo "WWS Language Support Module"
            echo ""
            echo "Usage: $0 <command> [options]"
            echo ""
            echo "Commands:"
            echo "  init              Initialize language environment"
            echo "  python [version]  Install Python (default: 3.11)"
            echo "  node [version]    Install Node.js (default: 20)"
            echo "  go [version]      Install Go (default: 1.21)"
            echo "  rust [version]    Install Rust (default: stable)"
            echo "  pip <packages>    Install Python packages"
            echo "  npm <packages>    Install Node.js packages"
            echo "  gomod <packages>  Install Go packages"
            echo "  cargo <packages>  Install Rust packages"
            echo "  list              List installed languages"
            echo "  path              Update PATH"
            echo "  help              Show this help"
            ;;
    esac
}

# Run main function
main "$@"
