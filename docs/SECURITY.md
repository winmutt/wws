# WWS Security Best Practices

## Overview

This document outlines security best practices for deploying and operating WWS.

## Authentication & Authorization

### OAuth2 Configuration

1. **Use Strong OAuth Settings**
   - Enable GitHub organization restriction
   - Require two-factor authentication for team members
   - Regularly rotate OAuth client secrets

2. **Token Management**
   - JWT tokens expire after 24 hours
   - OAuth tokens encrypted at rest (AES-256)
   - Automatic token refresh before expiration

### RBAC Implementation

1. **Role Hierarchy**
   - `admin`: Full organization access
   - `member`: Standard workspace access
   - `viewer`: Read-only access

2. **Principle of Least Privilege**
   - Assign minimum required permissions
   - Regular access reviews
   - Remove unused accounts

## Data Protection

### Encryption

1. **Encryption at Rest**
   - Database fields encrypted with AES-256
   - OAuth tokens stored encrypted
   - API keys encrypted before storage

2. **Encryption in Transit**
   - TLS 1.3 for all communications
   - Internal service-to-service TLS
   - Secure WebSocket connections

### Sensitive Data Handling

1. **What's Encrypted**
   - GitHub OAuth tokens
   - API keys
   - Database credentials
   - Temporary credentials

2. **What's Not Encrypted**
   - Public workspace metadata
   - User emails (GDPR considerations)
   - Audit log timestamps

## Network Security

### Workspace Isolation

1. **Network Segmentation**
   - Each workspace in isolated network
   - No inter-workspace communication
   - Outbound traffic filtering

2. **Firewall Rules**
   - Default deny all
   - Allow only required ports
   - SSH access on non-standard ports

### API Security

1. **Rate Limiting**
   - 100 requests/minute per user
   - Burst protection (200/10s)
   - DDoS protection enabled

2. **Input Validation**
   - All inputs validated
   - SQL injection prevention
   - XSS protection headers

## Audit & Compliance

### Audit Logging

1. **What's Logged**
   - All authentication events
   - Workspace CRUD operations
   - Permission changes
   - Admin actions

2. **Log Protection**
   - Logs immutable
   - Retention: 90 days
   - Secure storage

### Compliance

1. **Security Standards**
   - SOC 2 Type II ready
   - GDPR compliant
   - Data residency options

2. **Regular Audits**
   - Quarterly security reviews
   - Penetration testing annually
   - Dependency scanning weekly

## Secrets Management

### Best Practices

1. **Environment Variables**
   ```bash
   # Store secrets in environment
   export GITHUB_CLIENT_SECRET="..."
   export ENCRYPTION_KEY="..."
   ```

2. **Secret Rotation**
   - Rotate encryption keys annually
   - Rotate API keys quarterly
   - Rotate database credentials monthly

3. **Secret Storage**
   - Never commit secrets to version control
   - Use secrets management tools (Vault, AWS Secrets Manager)
   - Access logged and audited

## Vulnerability Management

### Scanning

1. **Dependency Scanning**
   - `go mod tidy` for Go dependencies
   - `npm audit` for Node.js dependencies
   - Weekly automated scans

2. **Container Scanning**
   - Scan images before deployment
   - Use trusted base images
   - Minimal attack surface

### Patching

1. **Update Schedule**
   - Security patches: within 48 hours
   - Minor updates: weekly
   - Major updates: monthly testing

2. **Testing**
   - Staging environment testing
   - Rollback capability
   - Zero-downtime deployments

## Incident Response

### Detection

1. **Monitoring**
   - Failed login attempts
   - Unusual API usage patterns
   - Resource utilization spikes

2. **Alerting**
   - Real-time alerts for security events
   - Daily security reports
   - Monthly security summaries

### Response

1. **Incident Classification**
   - Critical: Immediate response (< 1 hour)
   - High: Response within 4 hours
   - Medium: Response within 24 hours
   - Low: Response within 1 week

2. **Response Procedures**
   - Identify and contain
   - Eradicate threat
   - Recover systems
   - Lessons learned

## Security Checklist

### Pre-Deployment

- [ ] TLS certificates configured
- [ ] OAuth app configured with restrictions
- [ ] Database encryption enabled
- [ ] Rate limiting configured
- [ ] Audit logging enabled
- [ ] Secrets properly stored
- [ ] Firewall rules configured
- [ ] Backups configured

### Post-Deployment

- [ ] Security scan completed
- [ ] Penetration test scheduled
- [ ] Monitoring configured
- [ ] Alerting configured
- [ ] Backup testing performed
- [ ] Documentation updated
- [ ] Team trained on security procedures

### Ongoing

- [ ] Weekly dependency scans
- [ ] Monthly security reviews
- [ ] Quarterly access reviews
- [ ] Annual penetration testing
- [ ] Regular security training

## Secure Development

### Code Review

1. **Security Review Checklist**
   - Input validation
   - Authentication/authorization
   - Error handling (no information leakage)
   - Logging (no sensitive data)
   - Dependency security

2. **Static Analysis**
   - `golangci-lint` for Go
   - ESLint for TypeScript
   - Secret scanning in CI/CD

### Testing

1. **Security Tests**
   - Authentication bypass tests
   - Authorization tests
   - Injection tests
   - Rate limit tests

## Configuration Security

### Environment Variables

```bash
# Production security settings
ENCRYPTION_KEY=32-byte-key-here
JWT_SECRET=strong-jwt-secret
DATABASE_URL=postgresql://user:pass@host/db
```

### File Permissions

```bash
# Restrict sensitive files
chmod 600 .env
chmod 600 data/wws.db
chmod 700 .githooks/
```

## Provider-Specific Security

### Podman Security

```bash
# Run containers with minimal privileges
podman run --security-opt=no-new-privileges \
  --cap-drop=ALL \
  --read-only \
  wws-api
```

### KVM Security

```bash
# VM isolation
libvirt-network --isolated
qemu-user-mode networking
```

## Reporting Security Issues

If you discover a security vulnerability:

1. **Do NOT**
   - Create public issues
   - Discuss in public forums
   - Test on production systems

2. **DO**
   - Email security@winmutt.github.com
   - Provide reproduction steps
   - Allow 90 days for disclosure

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CWE/SANS Top 25](https://cwe.mitre.org/top25/)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
