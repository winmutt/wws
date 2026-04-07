# Winmutt's Work Spaces (WWS)

A remote workspace provisioning system for engineering organizations. Spin up isolated development environments on-demand, manage their lifecycle, and connect via remote editors.

## Overview

WWS enables engineering teams to create, manage, and destroy isolated development workspaces for ticket-based development workflows. Each workspace is a self-contained environment with persistent storage, pre-configured language tooling, and remote editing capabilities.

## Key Features

- **Isolated Workspaces** - KVM/Podman-based environments with resource quotas
- **Persistent Storage** - Home directory preserved across restarts
- **Remote Editing** - code-server (VSCode) integration
- **Language Support** - Python, JavaScript, Go, Rust (extensible)
- **Dotfiles Management** - yadm for configuration synchronization
- **GitHub Integration** - Clone repos or create new ones, credentials injected
- **Bootstrap Scripts** - Custom initialization logic
- **Team Collaboration** - Organization management, shared access, RBAC
- **Idle Management** - Auto-shutdown after configurable timeout (4-8 hours default)

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     Web Management UI                   │
│              (React - Create React App)                 │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                  Go Backend API                         │
│              SQLite (Metadata Storage)                  │
└─────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│   KVM Provider  │ │  Podman Runtime │ │  DigitalOcean   │
│   (Local VMs)   │ │  (Containers)   │ │  (Future)       │
└─────────────────┘ └─────────────────┘ └─────────────────┘
                            │
                            ▼
        ┌────────────────────────────────┐
        │    Workspace Agent (Inside)    │
        │  - Zsh shell                   │
        │  - yadm dotfiles              │
        │  - code-server                │
        │  - Bootstrap scripts          │
        │  - GitHub credentials         │
        └────────────────────────────────┘
```

## Project Structure

```
wws/
├── api/                    # Go backend API
│   ├── handlers/          # HTTP request handlers
│   ├── middleware/        # Auth, RBAC, logging
│   └── models/            # Database models
├── provisioner/           # Provider abstraction layer
│   ├── podman/            # Podman container runtime
│   ├── kvm/               # KVM virtualization
│   └── digitalocean/      # Cloud droplets (future)
├── workspace-agent/       # Runs inside each workspace
│   ├── init/              # Bootstrap scripts
│   ├── dotfiles/          # Yadm configuration
│   ├── editors/           # Editor servers
│   └── credentials/       # GitHub token management
├── languages/             # Language support modules
│   ├── python/
│   ├── javascript/
│   ├── go/
│   └── rust/
├── web/                   # React frontend
│   ├── public/
│   ├── src/
│   └── package.json
├── scripts/               # Provisioning & management scripts
├── docs/                  # Documentation
│   ├── ARCHITECTURE.md
│   └── specs/
└── tests/
```

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go (net/http, gin) |
| Frontend | React (Create React App) |
| Database | SQLite (MVP) |
| Container | Podman |
| VM | KVM/QEMU |
| Shell | Zsh |
| Dotfiles | yadm |
| Editor | code-server (VSCode) |
| Authentication | GitHub OAuth2 |

## Getting Started

### Prerequisites

- Go 1.21+
- Node.js 18+
- Podman (or Docker)
- KVM support (Linux kernel with KVM module)
- GitHub OAuth App

### Development Setup

**Option 1: Docker/Podman Compose (Recommended)**

```bash
# Clone repository
git clone https://github.com/yourorg/wws.git
cd wws

# Configure environment
cp .env.example .env
# Edit .env with your GitHub OAuth credentials

# Build and start services
podman compose up -d

# View logs
podman compose logs -f

# Access application
# Frontend: http://localhost:3000
# Backend API: http://localhost:8080

# Stop services
podman compose down
```

**Option 2: Running Pre-built Docker Images**

```bash
# Build images locally
podman build -t wws-api -f api/Dockerfile .
podman build -t wws-web -f web/Dockerfile .

# Create data directory
mkdir -p data

# Run API container
podman run -d \
  --name wws-api \
  -p 8080:8080 \
  -e GITHUB_CLIENT_ID=your_client_id \
  -e GITHUB_CLIENT_SECRET=your_client_secret \
  -e GITHUB_CALLBACK_URL=http://localhost:8080/oauth/callback \
  -v $(pwd)/data:/data \
  wws-api

# Run Web container (note: Podman doesn't need --link)
podman run -d \
  --name wws-web \
  -p 3000:80 \
  wws-web

# View logs
podman logs -f wws-api
podman logs -f wws-web

# Stop containers
podman stop wws-api wws-web
podman rm wws-api wws-web
```

**Access the application:**
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080

**Option 3: Local Development**

**Option 2: Local Development**

```bash
# Clone repository
git clone https://github.com/yourorg/wws.git
cd wws

# Backend
cd api
go mod download
go run cmd/main.go

# Frontend (in new terminal)
cd web
npm install
npm start
```

### Configuration

**Option 1: Environment Variables (Recommended for Docker/Podman)**

Copy `.env.example` to `.env`:
```bash
cp .env.example .env
```

Edit `.env` and set your GitHub OAuth credentials:
```bash
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GITHUB_CALLBACK_URL=http://localhost:8080/oauth/callback
CORS_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
```

**Option 2: Config File (For local development)**

Create `api/config.yaml`:
```yaml
server:
  port: 8080
  cors:
    origins: ["http://localhost:3000"]

database:
  path: "./data/wws.db"

github:
  client_id: "your_github_oauth_client_id"
  client_secret: "your_github_oauth_client_secret"
  callback_url: "http://localhost:8080/oauth/callback"

workspaces:
  idle_timeout_hours: 6
  default_storage_gb: 20
  default_cpu: 2
  default_memory_gb: 4
```

## Usage

### For Users

1. **Login** - Authenticate with GitHub
2. **Create Workspace** - Select organization, provide repo URL (or create new), assign unique tag
3. **Configure** - Choose languages, editor preferences
4. **Start Workspace** - Workspace provisions via Podman/KVM
5. **Connect** - Use code-server or SSH to access
6. **Work** - Develop on your ticket/issue
7. **Stop/Destroy** - When done, stop or destroy workspace

### For Administrators

1. **Create Organization** - Manage team structure
2. **Invite Users** - Add team members
3. **Monitor** - View workspace usage, resource consumption
4. **Configure** - Set idle timeouts, quotas, templates
5. **Audit** - Review action logs

## Security Considerations

- GitHub OAuth2 authentication
- RBAC for organization/workspace permissions
- Network isolation between workspaces
- Encrypted storage (Phase 2)
- Audit logging for all operations
- Resource quotas per workspace/user
- Auto-expiring credentials

## Roadmap

### Phase 1 (MVP)
- GitHub OAuth authentication
- Organization + team management
- KVM + Podman provisioning
- Workspace CRUD operations
- code-server integration
- Zsh + yadm dotfiles
- Bootstrap script execution
- Language checklist

### Phase 2 (Current - Team Features) ✅ Complete
- [x] Shared workspace access
- [x] Team-based permissions
- [x] Resource monitoring dashboard
- [x] Workspace templates
- [x] Usage analytics
- [x] Backup/restore
- [x] Encrypted storage
- [x] Tmux session sharing
- [x] Shared terminal
- [x] Workspace export/import
- [x] Idle timeout management
- [x] Comprehensive E2E test suite

### Phase 3
- DigitalOcean droplet support
- Kubernetes orchestration
- Additional editors (Cursor, IntelliJ)
- More language runtimes
- Custom provisioning plugins

## Contributing

1. Fork repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

## License

MIT License - see LICENSE file for details

## Acknowledgments

- [code-server](https://github.com/coder/code-server) - VSCode in browser
- [yadm](https://github.com/yadm-dev/yadm) - Yet Another Dotfiles Manager
- [Podman](https://podman.io) - Container runtime
- [KVM](https://www.linux-kvm.org) - Kernel-based Virtual Machine
