# WWS Architecture Documentation

## Overview

Winmutt's Work Spaces (WWS) is a remote workspace provisioning system designed for engineering organizations. It provides isolated development environments on-demand with full lifecycle management.

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Client Layer                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │
│  │   Web UI     │  │    SSH       │  │  code-server │               │
│  │  (React)     │  │   Client     │  │   (VSCode)   │               │
│  └──────────────┘  └──────────────┘  └──────────────┘               │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      API Gateway Layer                               │
│  ┌────────────────────────────────────────────────────────────┐     │
│  │              Go HTTP Server (net/http)                      │     │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐      │     │
│  │  │  Auth    │ │   RBAC   │ │ Logging  │ │ Rate Lim.│      │     │
│  │  │ Middleware│ │Middleware│ │Middleware│ │Middleware│      │     │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘      │     │
│  └────────────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Business Logic Layer                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐      │
│  │   Handlers      │  │    Models       │  │   Services      │      │
│  │  (HTTP API)     │  │  (Database)     │  │  (Logic)        │      │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘      │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Data Layer                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    SQLite Database                           │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       │    │
│  │  │ Users    │ │Orgs      │ │Workspaces│ │  Audit   │       │    │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘       │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Provisioning Layer                                │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐      │
│  │   Podman        │  │       KVM       │  │   DigitalOcean  │      │
│  │  (Containers)   │  │     (VMs)       │  │    (Cloud)      │      │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘      │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                   Workspace Agent Layer                              │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                  Inside Each Workspace                       │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       │    │
│  │  │   Zsh    │ │   yadm   │ │code-     │ │   SSH    │       │    │
│  │  │  Shell   │ │Dotfiles  │ │server    │ │  Daemon  │       │    │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘       │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

## Component Architecture

### 1. Frontend (React)

**Location:** `web/`

**Responsibilities:**
- User authentication UI
- Organization management
- Workspace dashboard
- Resource monitoring
- Team collaboration features

**Key Components:**
```
web/src/
├── components/
│   ├── Authentication/    # Login, Callback
│   ├── Organization/      # Org management
│   ├── Workspace/         # Workspace CRUD
│   ├── Team/              # Team features
│   └── Monitoring/        # Resource dashboards
├── hooks/                 # Custom React hooks
├── services/              # API client (gRPC-Web)
├── store/                 # State management
└── types/                 # TypeScript definitions
```

### 2. Backend API (Go)

**Location:** `api/`

**Responsibilities:**
- REST API endpoints
- Business logic
- Database operations
- Authentication & authorization

**Key Packages:**
```
api/
├── handlers/              # HTTP request handlers
│   ├── auth.go           # Authentication
│   ├── organizations.go  # Org CRUD
│   ├── workspaces.go     # Workspace management
│   ├── quota.go          # Resource quotas
│   └── compliance.go     # Audit logging
├── middleware/            # HTTP middleware
│   ├── auth.go           # JWT verification
│   ├── rbac.go           # Role-based access
│   ├── ratelimit.go      # Rate limiting
│   └── logging.go        # Request logging
├── models/                # Database models
├── pkg/                   # Shared utilities
│   ├── config.go         # Configuration
│   └── crypto.go         # Encryption
└── proto/                 # Protocol buffers
```

### 3. Provisioning Layer

**Location:** `provisioner/`

**Responsibilities:**
- Infrastructure provisioning
- Resource allocation
- Lifecycle management

**Providers:**
```
provisioner/
├── podman/               # Container runtime
│   ├── provider.go      # Podman implementation
│   └── container.go     # Container management
├── kvm/                  # Virtual machines
│   ├── provider.go      # KVM implementation
│   └── vm.go           # VM management
└── digitalocean/         # Cloud provider (future)
    └── provider.go      # DO implementation
```

### 4. Workspace Agent

**Location:** `workspace-agent/`

**Responsibilities:**
- Environment setup
- Dotfiles management
- Editor configuration
- Credential injection

**Components:**
```
workspace-agent/
├── init/                 # Bootstrap scripts
│   ├── bootstrap.sh     # Main initialization
│   └── packages.sh      # Package installation
├── dotfiles/            # yadm configuration
├── editors/             # Editor setup
│   └── codeserver.sh   # code-server installation
└── credentials/         # GitHub token management
```

### 5. Language Modules

**Location:** `languages/`

**Responsibilities:**
- Language runtime installation
- Package management
- Toolchain configuration

**Supported Languages:**
```
languages/
├── python/              # Python setup
├── javascript/          # Node.js setup
├── go/                  # Go toolchain
└── rust/                # Rust toolchain
```

## Data Flow

### Workspace Creation Flow

```
1. User submits workspace creation request
   ↓
2. Frontend validates input
   ↓
3. Backend receives request via POST /api/workspaces
   ↓
4. Authentication middleware verifies JWT
   ↓
5. RBAC middleware checks permissions
   ↓
6. Handler validates organization access
   ↓
7. Provider selected (Podman/KVM)
   ↓
8. Provisioner creates workspace
   ↓
9. Agent bootstrap script executed
   ↓
10. Workspace status updated to "running"
    ↓
11. Connection details returned to frontend
```

### Authentication Flow

```
1. User clicks "Login with GitHub"
   ↓
2. Redirect to GitHub OAuth2
   ↓
3. User authorizes application
   ↓
4. GitHub redirects back with code
   ↓
5. Backend exchanges code for tokens
   ↓
6. Tokens encrypted and stored
   ↓
7. JWT session token created
   ↓
8. User redirected to dashboard
```

## Security Architecture

### Authentication
- GitHub OAuth2 for user authentication
- JWT tokens for session management
- Encrypted OAuth token storage (AES-256)

### Authorization
- Role-Based Access Control (RBAC)
- Organization-level permissions
- Workspace-level permissions
- Team-based access control

### Data Protection
- Encryption at rest for sensitive data
- Auto-expiring temporary credentials
- Audit logging for all operations
- Rate limiting to prevent abuse

### Network Security
- Network isolation between workspaces
- Secure credential injection
- TLS for all communications

## Scalability Considerations

### Current Architecture
- Single SQLite database (MVP)
- Monolithic Go backend
- Stateless API servers

### Future Scaling
- PostgreSQL migration for multi-tenant support
- Horizontal API scaling with load balancer
- Redis for session caching
- Message queue for async operations

## Deployment Architecture

### Development
```
┌─────────────────┐
│  Podman Compose │
│  ┌───────────┐  │
│  │ API       │  │
│  │ Web       │  │
│  └───────────┘  │
└─────────────────┘
```

### Production (Future)
```
┌────────────────────────────────┐
│     Load Balancer              │
│  ┌──────────┐ ┌──────────┐    │
│  │  API 1   │ │  API 2   │    │
│  └──────────┘ └──────────┘    │
└────────────────────────────────┘
           │
┌──────────┴──────────┐
│   PostgreSQL Cluster │
└─────────────────────┘
```

## API Architecture

### REST Endpoints

**Authentication:**
- `POST /api/auth/login` - GitHub OAuth initiation
- `GET /api/auth/callback` - OAuth callback handler
- `POST /api/auth/logout` - Session termination

**Organizations:**
- `GET /api/organizations` - List organizations
- `POST /api/organizations` - Create organization
- `GET /api/organizations/{id}` - Get organization
- `PUT /api/organizations/{id}` - Update organization
- `DELETE /api/organizations/{id}` - Delete organization

**Workspaces:**
- `GET /api/workspaces` - List workspaces
- `POST /api/workspaces` - Create workspace
- `GET /api/workspaces/{id}` - Get workspace
- `PUT /api/workspaces/{id}` - Update workspace
- `DELETE /api/workspaces/{id}` - Delete workspace
- `POST /api/workspaces/{id}/start` - Start workspace
- `POST /api/workspaces/{id}/stop` - Stop workspace
- `POST /api/workspaces/{id}/restart` - Restart workspace

**Team Features:**
- `POST /api/organizations/{id}/members` - Add member
- `GET /api/organizations/{id}/members` - List members
- `PUT /api/organizations/{id}/members/{userId}` - Update role
- `DELETE /api/organizations/{id}/members/{userId}` - Remove member

### gRPC-Web (Protocol Buffers)

**Services:**
- `AuthService` - Authentication operations
- `OrganizationService` - Organization management
- `WorkspaceService` - Workspace lifecycle
- `TeamService` - Team collaboration
- `QuotaService` - Resource quotas
- `AuditService` - Audit logging

## Technology Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| Frontend | React 18 + TypeScript | User interface |
| Backend | Go 1.21+ | API server |
| Database | SQLite (MVP) / PostgreSQL (Future) | Data persistence |
| Container | Podman | Container runtime |
| VM | KVM/QEMU | Virtualization |
| Auth | GitHub OAuth2 + JWT | Authentication |
| Communication | REST + gRPC-Web | API protocols |
| Cache | (Future: Redis) | Session caching |

## Future Enhancements

### Phase 3
- DigitalOcean provider integration
- Kubernetes orchestration support
- Additional editor support (Cursor, IntelliJ)
- Expanded language runtimes
- Plugin system for extensibility

### Phase 4
- Comprehensive documentation
- API documentation with examples
- Developer setup guides
- Deployment guides
- Security best practices
- Contributing guidelines
