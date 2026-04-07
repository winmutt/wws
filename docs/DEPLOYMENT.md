# WWS Deployment Guide

## Overview

This guide covers deploying WWS to production environments.

## Deployment Options

### Option 1: Podman Compose (Simple)

For small teams and development:

```bash
# Clone repository
git clone https://github.com/winmutt/wws.git
cd wws

# Configure environment
cp .env.example .env
# Edit .env with production values

# Start services
podman compose up -d

# View logs
podman compose logs -f

# Stop services
podman compose down
```

### Option 2: Docker Swarm

For medium-scale deployments:

```bash
# Initialize swarm
docker swarm init

# Deploy stack
docker stack deploy -c docker-compose.yml wws

# Scale services
docker service scale wws_api=3
```

### Option 3: Kubernetes (Production)

For large-scale production deployments.

#### Prerequisites

- Kubernetes cluster 1.21+
- Helm 3+
- Persistent storage (PV/PVC)
- Ingress controller

#### Installation

```bash
# Add Helm repo
helm repo add wws https://charts.winmutt.github.io/wws

# Install
helm install wws wws/wws \
  --namespace wws \
  --create-namespace \
  --values values-production.yaml
```

#### Kubernetes Manifests

```yaml
# api-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: wws-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: wws-api
  template:
    metadata:
      labels:
        app: wws-api
    spec:
      containers:
      - name: api
        image: winmutt/wws-api:latest
        ports:
        - containerPort: 8080
        env:
        - name: GITHUB_CLIENT_ID
          valueFrom:
            secretKeyRef:
              name: wws-secrets
              key: github-client-id
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: wws-secrets
              key: database-url
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /api/health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /api/health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: wws-api
spec:
  selector:
    app: wws-api
  ports:
  - port: 8080
    targetPort: 8080
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: wws-ingress
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - wws.yourcompany.com
    secretName: wws-tls
  rules:
  - host: wws.yourcompany.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: wws-api
            port:
              number: 8080
```

## Environment Configuration

### Production Environment Variables

```bash
# GitHub OAuth
GITHUB_CLIENT_ID=your_production_client_id
GITHUB_CLIENT_SECRET=your_production_client_secret
GITHUB_CALLBACK_URL=https://wws.yourcompany.com/oauth/callback

# Server
SERVER_PORT=8080
CORS_ORIGINS=https://wws.yourcompany.com

# Database
DATABASE_URL=postgresql://user:pass@db-host:5432/wws

# Security
ENCRYPTION_KEY=your-32-byte-encryption-key
JWT_SECRET=your-jwt-secret

# Workspaces
WORKSPACE_IDLE_TIMEOUT_HOURS=6
WORKSPACE_DEFAULT_STORAGE_GB=20
WORKSPACE_DEFAULT_CPU=2
WORKSPACE_DEFAULT_MEMORY_GB=4
```

## Database Setup

### PostgreSQL (Production)

```bash
# Create database
createdb wws_production

# Run migrations
go run api/main.go migrate up
```

### Connection Pooling

For PostgreSQL with connection pooling:

```yaml
database:
  url: ${DATABASE_URL}
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m
```

## Scaling

### Horizontal Scaling

```bash
# Podman Compose
podman compose up -d --scale api=3

# Kubernetes
kubectl scale deployment wws-api --replicas=5
```

### Load Balancing

Use a load balancer (nginx, HAProxy, cloud LB) to distribute traffic across API instances.

### Session Management

For multi-instance deployments, use Redis for session storage:

```yaml
session:
  store: redis
  redis:
    addr: redis-host:6379
    password: ${REDIS_PASSWORD}
    db: 0
```

## Monitoring

### Prometheus Metrics

Enable metrics endpoint at `/api/metrics`:

```yaml
metrics:
  enabled: true
  path: /metrics
  port: 9090
```

### Logging

Configure structured logging:

```yaml
logging:
  level: info
  format: json
  output: stdout
```

### Alerting

Set up alerts for:
- API error rate > 1%
- Response time > 500ms
- Database connection failures
- Workspace provisioning failures

## Backup Strategy

### Database Backups

```bash
# Daily backup script
#!/bin/bash
pg_dump wws_production | gzip > /backups/wws-$(date +%Y%m%d).sql.gz
```

### Workspace Backups

Workspaces are automatically backed up based on configuration:

```yaml
backup:
  enabled: true
  schedule: "0 2 * * *"  # Daily at 2 AM
  retention_days: 30
```

## Security Hardening

### TLS/SSL

Use Let's Encrypt for certificates:

```yaml
tls:
  enabled: true
  cert_file: /etc/ssl/certs/wws.crt
  key_file: /etc/ssl/private/wws.key
```

### Network Policies

Restrict network access:

```yaml
network:
  allowed_origins:
    - https://wws.yourcompany.com
  rate_limit:
    requests_per_minute: 100
```

### Secrets Management

Use Kubernetes secrets or external secrets manager:

```bash
kubectl create secret generic wws-secrets \
  --from-literal=github-client-id=xxx \
  --from-literal=github-client-secret=xxx \
  --from-literal=database-url=xxx
```

## CI/CD Pipeline

### GitHub Actions Example

```yaml
name: Deploy to Production

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Build and push API
      uses: docker/build-push-action@v4
      with:
        context: ./api
        push: true
        tags: winmutt/wws-api:${{ github.sha }}
    
    - name: Deploy to Kubernetes
      run: |
        kubectl set image deployment/wws-api \
          api=winmutt/wws-api:${{ github.sha }}
        kubectl rollout restart deployment/wws-api
```

## Troubleshooting

### Common Issues

1. **Database Connection Failed**
   - Check DATABASE_URL
   - Verify network connectivity
   - Check credentials

2. **OAuth Callback Failed**
   - Verify GITHUB_CALLBACK_URL matches GitHub app settings
   - Check CORS configuration

3. **Workspace Provisioning Failed**
   - Check provider configuration
   - Verify sufficient resources
   - Check logs: `podman compose logs api`

### Logs

```bash
# View all logs
podman compose logs -f

# View specific service
podman compose logs -f api

# Kubernetes logs
kubectl logs -f deployment/wws-api -n wws
```

## Upgrades

### Rolling Update

```bash
# Kubernetes
kubectl set image deployment/wws-api api=new-image:tag
kubectl rollout status deployment/wws-api
```

### Database Migrations

```bash
# Run migrations before deployment
go run api/main.go migrate up
```

## Performance Tuning

### API Server

```yaml
server:
  max_requests: 1000
  timeout: 30s
  keep_alive: 5m
```

### Database

```yaml
database:
  pool_size: 50
  statement_timeout: 30s
```

### Cache

```yaml
cache:
  enabled: true
  ttl: 300s
```
