# WWS Developer Setup Guide

## Overview

This guide will help you set up a local development environment for WWS (Winmutt's Work Spaces).

## Prerequisites

- Go 1.21+
- Node.js 18+
- Podman (or Docker)
- KVM support (Linux with KVM module)
- GitHub OAuth App credentials

## Quick Start

### 1. Clone Repository

```bash
git clone https://github.com/winmutt/wws.git
cd wws
```

### 2. Set Up GitHub OAuth App

1. Go to GitHub Settings > Developer settings > OAuth Apps
2. Create new OAuth App
3. Set Authorization callback URL to: `http://localhost:8080/oauth/callback`
4. Copy Client ID and Client Secret

### 3. Configure Environment

```bash
cp .env.example .env
```

Edit `.env`:

```bash
GITHUB_CLIENT_ID=your_client_id_here
GITHUB_CLIENT_SECRET=your_client_secret_here
GITHUB_CALLBACK_URL=http://localhost:8080/oauth/callback
CORS_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
```

### 4. Run with Podman Compose

```bash
podman compose up -d
```

Access the application:
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080

## Development Setup

### Backend Development

```bash
cd api

# Install dependencies
go mod download

# Run tests
go test ./... -v

# Run development server
go run main.go
```

### Frontend Development

```bash
cd web

# Install dependencies
npm install

# Run development server
npm start
```

Access frontend: http://localhost:3000

## Project Structure

```
wws/
├── api/                    # Go backend
│   ├── handlers/          # HTTP handlers
│   ├── middleware/        # Auth, logging, RBAC
│   ├── models/            # Database models
│   ├── proto/             # Protocol buffers
│   └── main.go
├── web/                   # React frontend
│   ├── src/
│   ├── public/
│   └── package.json
├── provisioner/           # Provider implementations
│   ├── podman/
│   └── kvm/
├── workspace-agent/       # Workspace bootstrap
├── languages/             # Language modules
├── tests/                 # Integration tests
└── docs/                  # Documentation
```

## Building

### Build Backend

```bash
cd api
go build -o wws-api main.go
```

### Build Frontend

```bash
cd web
npm run build
```

### Build Containers

```bash
# Build API container
podman build -t wws-api -f api/Dockerfile .

# Build Web container
podman build -t wws-web -f web/Dockerfile .
```

## Testing

### Unit Tests

```bash
# Backend
go test ./api/... -v

# Frontend
cd web
npm test
```

### Integration Tests

```bash
cd tests
./workspace_lifecycle_test.sh
```

### E2E Tests

```bash
cd tests/e2e
./run_phase25_tests.sh
```

## Debugging

### Backend Debugging

Use Delve debugger:

```bash
cd api
dlv debug main.go
```

### Frontend Debugging

Use Chrome DevTools:
1. Open http://localhost:3000
2. Press F12 to open DevTools
3. Use Sources tab for debugging

## Database

### Migrations

The application uses SQLite for development. Database is automatically initialized on first run.

### Reset Database

```bash
rm data/wws.db
# Restart application to reinitialize
```

## CI/CD

### GitHub Actions

The project uses GitHub Actions for CI/CD:
- Tests run on push to feature branches
- Integration tests run on PR to main
- Deployments to staging on merge to main

## Contributing

1. Create feature branch from main
2. Make changes
3. Run tests
4. Submit PR

See [CONTRIBUTING.md](./CONTRIBUTING.md) for more details.

## Troubleshooting

### Podman Issues

If you encounter permission issues:

```bash
# Add user to podman group
sudo usermod -aG podman $USER

# Or use rootful podman
sudo podman compose up -d
```

### Port Already in Use

```bash
# Check what's using port 8080
lsof -i :8080

# Kill process or change port in .env
```

### KVM Not Available

For KVM provisioning, ensure:
1. KVM module loaded: `lsmod | grep kvm`
2. User in kvm group: `sudo usermod -aG kvm $USER`

## Next Steps

- Read [ARCHITECTURE.md](./ARCHITECTURE.md) for system overview
- Check [API.md](./REST_API.md) for API documentation
- Review [COMPONENTS.md](./COMPONENTS.md) for component details
