# Winmutt's Work Spaces (WWS) - Development Guide

## File Editing Conventions

### Go Files
- Package name matches directory name
- Use `fmt`, `log`, `errors` imports
- Follow Go naming conventions (camelCase for variables, PascalCase for types)
- Add doc comments for exported functions
- Use context for long-running operations

### React/TypeScript Files
- Component names in PascalCase
- Use functional components with hooks
- Add prop types with TypeScript interfaces
- Follow React naming conventions

### Configuration Files
- YAML format for configs
- Use meaningful variable names
- Document all required fields

### Documentation Files
- Markdown format
- Use clear headings and subheadings
- Include code examples with language identifiers

## TODO Processing Workflow

### Step-by-Step Process

1. **Always checkout from main first**
   ```bash
   git checkout main
   git pull origin main
   ```

2. **Each step = one git commit + unit test**
   - Create branch: `git checkout -b feature/<subsection>/<step-number>`
   - Implement step
   - Add unit test: `go test ./<package> -run Test<Feature>`
   - Commit: `git commit -m "<subsection>: <step description>"`
   - Push: `git push origin feature/<subsection>/<step-number>`

3. **Each subsection = separate branch + PR**
   - Create PR from subsection branch to main
   - Run integration tests
   - Get review and merge

### Branch Naming Convention

```
feature/<subsection>/<step-number>
e.g., feature/auth/1.2.1
```

### Commit Message Format

```
<subsection>: <description>

- Implemented <feature>
- Added unit test
- Closes #<issue-number> (if applicable)
```

### Testing Requirements

- **Unit tests**: For each step, test individual functionality
- **Integration tests**: For subsection PR, test all steps together
- **Test coverage**: Minimum 80% for backend code

## Project Structure

```
wws/
├── api/              # Go backend
│   ├── handlers/     # HTTP handlers
│   ├── middleware/   # Auth, logging, etc.
│   ├── models/       # Database models
│   └── main.go
├── provisioner/      # Provider implementations
│   ├── podman/
│   ├── kvm/
│   └── digitalocean/
├── workspace-agent/  # Inside workspace code
├── languages/        # Language modules
├── web/              # React frontend
│   ├── src/
│   └── public/
├── scripts/          # Provisioning scripts
├── docs/             # Documentation
├── tests/            # Integration tests
└── AGENTS.md         # This file
```

## Quick Start Commands

```bash
# Setup
git clone https://github.com/winmutt/wws.git
cd wws

# Configure environment (see README.md for required variables)
export GITHUB_CLIENT_ID=your_github_client_id
export GITHUB_CLIENT_SECRET=your_github_client_secret
export GITHUB_CALLBACK_URL=http://localhost:8080/oauth/callback
export CORS_ORIGINS=http://localhost:3000,http://127.0.0.1:3000

# Create new feature branch
git checkout main
git pull origin main
git checkout -b feature/auth/1.2.1

# Build and run with Docker
docker-compose build && docker-compose up -d

# Build and run with Podman
podman compose build && podman compose up -d

# Run tests inside container
docker-compose exec api go test ./... -v
```

## Code Review Checklist

- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Code follows conventions
- [ ] Documentation updated
- [ ] No secrets in code
- [ ] Error handling added
- [ ] Logging added
- [ ] Type safety ensured

## CI/CD Pipeline

- **Push to feature branch**: Run unit tests
- **PR to main**: Run integration tests
- **Merge to main**: Deploy to staging
- **Tag release**: Deploy to production

## Docker & Podman Development

### Building and Running with Compose

**Using Docker:**
```bash
# Build and start services
docker-compose build && docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

**Using Podman:**
```bash
# Build and start services
podman compose build && podman compose up -d

# View logs
podman compose logs -f

# Stop services
podman compose down
```

### Directory Structure

```
wws/
├── api/              # Go backend
│   ├── Dockerfile    # Backend container
│   ├── handlers/     # HTTP handlers
│   ├── middleware/   # Auth, logging, etc.
│   ├── models/       # Database models
│   └── main.go
├── web/              # React frontend
│   ├── Dockerfile    # Frontend container
│   ├── src/
│   └── public/
├── provisioner/      # Provider implementations
│   ├── podman/
│   ├── kvm/
│   └── digitalocean/
├── workspace-agent/  # Inside workspace code
├── languages/        # Language modules
├── scripts/          # Provisioning scripts
├── docs/             # Documentation
├── tests/            # Integration tests
├── docker-compose.yml
└── AGENTS.md         # This file
```

### Environment Variables

See [README.md - Getting Started](../README.md#quick-start-with-dockerpodman-compose) for required environment variables and configuration instructions.

### Container Orchestration Roadmap

- **Current**: Podman for local development
- **Future**: Kubernetes for production deployment
- **Approach**: Modular architecture supporting both
