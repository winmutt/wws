# WWS REST API Documentation

## Overview

The WWS API provides REST interfaces for workspace management, organization administration, and team collaboration.

## Base URL

```
Development: http://localhost:8080/api
Production: https://api.yourcompany.com/api
```

## Authentication

All API endpoints require authentication via JWT tokens obtained through GitHub OAuth2.

### Authentication Header

```
Authorization: Bearer <jwt_token>
```

## Endpoints

### Authentication

#### POST /api/auth/login
Initiate GitHub OAuth2 flow.

**Response:** 302 Redirect to GitHub

#### GET /api/auth/callback
Handle GitHub OAuth2 callback.

**Query Parameters:**
- `code` - Authorization code from GitHub
- `state` - CSRF protection token

**Response:** 302 Redirect to dashboard with JWT cookie

#### POST /api/auth/logout
Terminate user session.

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response:** 200 OK

---

### Organizations

#### GET /api/organizations
List all organizations for the current user.

**Response:**
```json
{
  "organizations": [
    {
      "id": "org-123",
      "name": "Engineering Team",
      "owner_id": "user-456",
      "created_at": "2024-01-15T10:30:00Z",
      "member_count": 12
    }
  ]
}
```

#### POST /api/organizations
Create a new organization.

**Request Body:**
```json
{
  "name": "New Organization",
  "description": "Team organization description"
}
```

**Response:** 201 Created

---

### Workspaces

#### GET /api/workspaces
List workspaces with optional filtering.

**Query Parameters:**
- `organization_id` - Filter by organization
- `status` - Filter by status
- `page` - Page number
- `limit` - Items per page

**Response:**
```json
{
  "workspaces": [
    {
      "id": "ws-123",
      "name": "Feature Development",
      "organization_id": "org-456",
      "provider": "podman",
      "status": "running",
      "repo_url": "https://github.com/org/repo",
      "tag": "feature-123",
      "created_at": "2024-01-15T10:30:00Z",
      "connection": {
        "ssh_url": "ssh://user@host:2222",
        "codeserver_url": "https://host:8080"
      }
    }
  ]
}
```

#### POST /api/workspaces
Create workspace.

**Request Body:**
```json
{
  "organization_id": "org-456",
  "name": "Feature Development",
  "repo_url": "https://github.com/org/repo",
  "tag": "feature-123",
  "provider": "podman",
  "config": {
    "cpu": 2,
    "memory_gb": 4,
    "storage_gb": 20
  }
}
```

**Response:** 202 Accepted

#### GET /api/workspaces/{id}
Get workspace details.

#### PUT /api/workspaces/{id}
Update workspace configuration.

#### DELETE /api/workspaces/{id}
Delete workspace.

---

### Workspace Actions

#### POST /api/workspaces/{id}/start
Start workspace.

**Response:** 202 Accepted

#### POST /api/workspaces/{id}/stop
Stop workspace.

**Response:** 202 Accepted

#### POST /api/workspaces/{id}/restart
Restart workspace.

**Response:** 202 Accepted

---

### Resource Monitoring

#### GET /api/workspaces/{id}/metrics
Get workspace resource metrics.

**Response:**
```json
{
  "cpu_usage_percent": 45.2,
  "memory_usage_gb": 2.1,
  "storage_usage_gb": 12.5
}
```

---

### API Keys

#### GET /api/apikeys
List API keys.

#### POST /api/apikeys
Create API key.

**Request Body:**
```json
{
  "name": "Production Access",
  "expires_in_days": 365
}
```

**Response:** 201 Created (includes key - shown once!)

#### DELETE /api/apikeys/{id}
Revoke API key.

---

### Audit Logging

#### GET /api/audit
Get audit logs with filtering.

**Query Parameters:**
- `user_id` - Filter by user
- `action` - Filter by action
- `start_time` - Start time (ISO 8601)
- `end_time` - End time (ISO 8601)

---

### Health Check

#### GET /api/health
Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0"
}
```

---

## Error Responses

### Standard Error Format

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable error message"
  }
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `UNAUTHORIZED` | 401 | Invalid authentication |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `VALIDATION_ERROR` | 400 | Invalid request |
| `QUOTA_EXCEEDED` | 409 | Quota exceeded |
| `INTERNAL_ERROR` | 500 | Server error |

---

## Rate Limiting

API requests are rate limited:

- **Standard:** 100 requests per minute

### Rate Limit Headers

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1705416000
```

---

## gRPC-Web API

For detailed gRPC-Web documentation, see [PROTOBUF_API.md](./PROTOBUF_API.md).
