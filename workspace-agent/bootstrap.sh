#!/bin/bash
# WWS Workspace Agent Bootstrap Script
# This script initializes a workspace with all required tools and configurations

set -e

# Configuration
GITHUB_USERNAME="${GITHUB_USERNAME:-}"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"
DOTFILES_REPO="${DOTFILES_REPO:-}"
WORKSPACE_ID="${WORKSPACE_ID:-}"

# Logging
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1" >&2
    exit 1
}

# Check required environment variables
check_requirements() {
    log "Checking requirements..."
    
    if [ -z "$GITHUB_USERNAME" ]; then
        error "GITHUB_USERNAME environment variable is required"
    fi
    
    if [ -z "$GITHUB_TOKEN" ]; then
        error "GITHUB_TOKEN environment variable is required"
    fi
    
    log "Requirements check passed"
}

# Install Zsh shell
install_zsh() {
    log "Installing Zsh shell..."
    
    if ! command -v zsh &> /dev/null; then
        if [ -f /etc/debian_version ]; then
            apt-get update && apt-get install -y zsh
        elif [ -f /etc/redhat-release ]; then
            yum install -y zsh
        else
            error "Unsupported OS distribution"
        fi
    else
        log "Zsh is already installed"
    fi
    
    # Set Zsh as default shell
    chsh -s $(which zsh) || true
    
    log "Zsh installation complete"
}

# Install yadm (Yet Another Dotfiles Manager)
install_yadm() {
    log "Installing yadm..."
    
    YADM_INSTALL_URL="https://github.com/TheLocehiliosan/yadm/releases/latest/download/yadm"
    
    if ! command -v yadm &> /dev/null; then
        curl -L -o /tmp/yadm "${YADM_INSTALL_URL}"
        chmod a+rx /tmp/yadm
        mv /tmp/yadm /usr/local/bin/yadm
    else
        log "yadm is already installed"
    fi
    
    log "yadm installation complete"
}

# Initialize dotfiles repository
init_dotfiles() {
    log "Initializing dotfiles repository..."
    
    if [ -z "$DOTFILES_REPO" ]; then
        log "DOTFILES_REPO not set, skipping dotfiles initialization"
        return 0
    fi
    
    # Configure git for yadm
    yadm config user.email "${GITHUB_USERNAME}@users.noreply.github.com"
    yadm config user.name "${GITHUB_USERNAME}"
    
    # Clone dotfiles repository
    if [ ! -d "$HOME/.local/share/yadm/repo.git" ]; then
        GIT_ASKPASS=/bin/true GIT_TERMINAL_PROMPT=0 \
        yadm clone "https://${GITHUB_TOKEN}@github.com/${GITHUB_USERNAME}/${DOTFILES_REPO}.git" || {
            log "Failed to clone dotfiles repository, creating empty repo"
            yadm init
        }
    else
        log "Dotfiles repository already exists, pulling latest"
        yadm pull
    fi
    
    log "Dotfiles initialization complete"
}

# Install gh CLI
install_gh_cli() {
    log "Installing GitHub CLI..."
    
    if ! command -v gh &> /dev/null; then
        if [ -f /etc/debian_version ]; then
            type -p curl >/dev/null || apt-get install curl -y
            curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
            chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg
            echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null
            apt-get update && apt-get install gh -y
        elif [ -f /etc/redhat-release ]; then
            yum install -y 'dnf-command(config-manager)'
            yum config-manager --add-repo https://packages.github.com/stable/copr/fedora/github-cli.repo
            yum install -y gh
        else
            error "Unsupported OS distribution"
        fi
    else
        log "gh CLI is already installed"
    fi
    
    # Authenticate gh CLI
    if [ -n "$GITHUB_TOKEN" ]; then
        echo "$GITHUB_TOKEN" | gh auth login --with-token
        log "gh CLI authenticated"
    fi
    
    log "gh CLI installation complete"
}

# Install and configure code-server
install_codeserver() {
    log "Installing code-server..."
    
    if ! command -v code-server &> /dev/null; then
        curl -fsSL https://code-server.dev/install.sh | sh
    else
        log "code-server is already installed"
    fi
    
    # Create configuration directory
    mkdir -p ~/.config/code-server
    
    # Generate default configuration
    if [ ! -f ~/.config/code-server/config.yaml ]; then
        code-server --bind-addr 0.0.0.0:8080 &
        sleep 2
        pkill code-server || true
    fi
    
    log "code-server installation complete"
}

# Configure SSH daemon
configure_ssh() {
    log "Configuring SSH..."
    
    # Install OpenSSH server if not present
    if ! command -v sshd &> /dev/null; then
        if [ -f /etc/debian_version ]; then
            apt-get update && apt-get install -y openssh-server
        elif [ -f /etc/redhat-release ]; then
            yum install -y openssh-server
        else
            error "Unsupported OS distribution"
        fi
    fi
    
    # Create SSH directory
    mkdir -p ~/.ssh
    chmod 700 ~/.ssh
    
    # Generate SSH key if not exists
    if [ ! -f ~/.ssh/id_ed25519 ]; then
        ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_ed25519 -C "${GITHUB_USERNAME}@workspace"
        chmod 600 ~/.ssh/id_ed25519
        chmod 644 ~/.ssh/id_ed25519.pub
    fi
    
    # Add public key to authorized_keys
    if [ -f ~/.ssh/id_ed25519.pub ]; then
        cat ~/.ssh/id_ed25519.pub >> ~/.ssh/authorized_keys
        chmod 600 ~/.ssh/authorized_keys
    fi
    
    # Start SSH daemon
    if ! pgrep sshd > /dev/null; then
        mkdir -p /run/sshd
        /usr/sbin/sshd
    fi
    
    log "SSH configuration complete"
}

# Set up persistent home directory
setup_persistent_storage() {
    log "Setting up persistent storage..."
    
    # Create workspace data directory
    mkdir -p /workspace-data/"${WORKSPACE_ID}"
    
    # Create symlink to home directory data
    if [ ! -L ~/workspace-data ]; then
        ln -s /workspace-data/"${WORKSPACE_ID}" ~/workspace-data
    fi
    
    log "Persistent storage setup complete"
}

# Main execution
main() {
    log "Starting workspace agent bootstrap..."
    log "Workspace ID: ${WORKSPACE_ID}"
    log "GitHub Username: ${GITHUB_USERNAME}"
    
    check_requirements
    install_zsh
    install_yadm
    init_dotfiles
    install_gh_cli
    install_codeserver
    configure_ssh
    setup_persistent_storage
    
    log "Workspace agent bootstrap complete!"
    log "Workspace is ready for use"
}

# Run main function
main "$@"
