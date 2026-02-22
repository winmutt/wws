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

# Create new feature branch
git checkout main
git pull origin main
git checkout -b feature/auth/1.2.1

# Run tests
go test ./api/... -v
cd web && npm test

# Build
go build ./api
cd web && npm run build
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

### Building and Running Locally

```bash
# Build the backend
podman build -f api/Dockerfile -t wws-api .

# Build the frontend
podman build -f web/Dockerfile -t wws-web .

# Run with podman compose
podman compose up -d

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

### Container Orchestration Roadmap

- **Current**: Podman for local development
- **Future**: Kubernetes for production deployment
- **Approach**: Modular architecture supporting both
