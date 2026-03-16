# WWS Protocol Buffer API Documentation

This document provides comprehensive documentation for all gRPC-Web endpoints exposed by the WWS (Winmutt's Work Spaces) backend API using Protocol Buffers.

## Overview

The WWS API uses Protocol Buffers (protobuf) with gRPC-Web for type-safe, efficient communication between the frontend React application and the Go backend. All endpoints are accessed through a gRPC-Web gateway.

## Base URL

```
http://localhost:8080
```

## Authentication

All authenticated endpoints require a session token obtained through the GitHub OAuth flow. The session is maintained through HTTP cookies.

### Authentication Flow

1. **Initiate OAuth**: Call `GitHubAuth` to get the GitHub OAuth URL
2. **Redirect User**: User authenticates with GitHub
3. **Callback**: GitHub redirects to `/oauth/callback` with authorization code
4. **Complete**: Call `GitHubCallback` to exchange code for tokens and session
5. **Use Session**: Subsequent requests use the established session

---

## Common Types

All services import `common/common.proto` which defines shared types:

### Timestamp

```protobuf
message Timestamp {
  string value = 1;  // RFC3339 formatted timestamp
}
```

### Status

```protobuf
enum Status {
  STATUS_UNSPECIFIED = 0;
  STATUS_ACTIVE = 1;
  STATUS_INACTIVE = 2;
  STATUS_RUNNING = 3;
  STATUS_STOPPED = 4;
  STATUS_CREATING = 5;
  STATUS_DELETING = 6;
  STATUS_ERROR = 7;
}
```

### PaginationRequest

```protobuf
message PaginationRequest {
  int32 page_size = 1;   // Number of items per page
  string page_token = 2; // Token for next/prev page
}
```

### PaginationResponse

```protobuf
message PaginationResponse {
  int32 total_count = 1;       // Total number of items
  string next_page_token = 2;  // Token for next page
}
```

### Empty

```protobuf
message Empty {}  // Used for operations that don't return data
```

---

## Authentication Service

**Service**: `wws.auth.AuthService`  
**Route Prefix**: `/api/auth`

### GitHubAuth

Initiates the GitHub OAuth authentication flow.

**Request**:
```protobuf
message GitHubAuthRequest {}
```

**Response**:
```protobuf
message GitHubAuthResponse {
  string auth_url = 1;  // GitHub OAuth URL to redirect user
  string state = 2;     // CSRF protection state token
}
```

**Usage**:
```typescript
import { auth } from './proto/auth/auth_pb';

const request = new auth.GitHubAuthRequest();
const response = await authService.gitHubAuth(request);
// Redirect user to response.getAuthUrl()
```

### GitHubCallback

Completes the OAuth flow by exchanging the authorization code for user credentials.

**Request**:
```protobuf
message GitHubCallbackRequest {
  string code = 1;  // Authorization code from GitHub
  string state = 2; // State token from initial request
}
```

**Response**:
```protobuf
message GitHubCallbackResponse {
  wws.user.User user = 1;       // User information
  string access_token = 2;      // Session token
  bool new_user = 3;            // True if user just registered
}
```

**Usage**:
```typescript
import { auth } from './proto/auth/auth_pb';

const request = new auth.GitHubCallbackRequest();
request.setCode(codeFromURL);
request.setState(stateFromURL);

const response = await authService.gitHubCallback(request);
// Store access_token, redirect to dashboard
```

### GetSession

Retrieves the current user's session information.

**Request**:
```protobuf
message GetSessionRequest {}
```

**Response**:
```protobuf
message SessionResponse {
  bool authenticated = 1;    // True if session is valid
  wws.user.User user = 2;    // User information
  wws.common.Timestamp expires_at = 3;  // Session expiration
}
```

**Usage**:
```typescript
import { auth } from './proto/auth/auth_pb';

const request = new auth.GetSessionRequest();
const response = await authService.getSession(request);

if (response.getAuthenticated()) {
  // User is logged in
  const user = response.getUser();
}
```

### Logout

Invalidates the current session.

**Request**:
```protobuf
message LogoutRequest {}
```

**Response**: `common.Empty`

**Usage**:
```typescript
import { auth } from './proto/auth/auth_pb';

const request = new auth.LogoutRequest();
await authService.logout(request);
// Clear local state, redirect to login
```

---

## Organization Service

**Service**: `wws.organization.OrganizationService`  
**Route Prefix**: `/api/organizations`

### ListOrganizations

Lists all organizations the current user is a member of.

**Request**:
```protobuf
message ListOrganizationsRequest {
  wws.common.PaginationRequest pagination = 1;
}
```

**Response**:
```protobuf
message ListOrganizationsResponse {
  repeated Organization organizations = 1;
  wws.common.PaginationResponse pagination = 2;
}
```

**Organization Message**:
```protobuf
message Organization {
  int32 id = 1;
  string name = 2;
  int32 owner_id = 3;
  string description = 4;
  string avatar_url = 5;
  wws.common.Timestamp created_at = 6;
  wws.common.Timestamp updated_at = 7;
}
```

**Example Request**:
```typescript
import { organization } from './proto/organization/organization_pb';

const request = new organization.ListOrganizationsRequest();
const pagination = new common.PaginationRequest();
pagination.setPageSize(10);
request.setPagination(pagination);

const response = await organizationService.listOrganizations(request);
```

### GetOrganization

Retrieves a specific organization by ID.

**Request**:
```protobuf
message GetOrganizationRequest {
  int32 id = 1;
}
```

**Response**:
```protobuf
message OrganizationResponse {
  Organization organization = 1;
}
```

### CreateOrganization

Creates a new organization.

**Request**:
```protobuf
message CreateOrganizationRequest {
  string name = 1;        // Required: Organization name
  string description = 2; // Optional: Description
}
```

**Response**: `OrganizationResponse`

**Validation**:
- `name`: Required, 1-100 characters, alphanumeric and hyphens only
- `description`: Optional, max 500 characters

**Example**:
```typescript
const request = new organization.CreateOrganizationRequest();
request.setName("my-team");
request.setDescription("My development team");

const response = await organizationService.createOrganization(request);
```

### UpdateOrganization

Updates an existing organization.

**Request**:
```protobuf
message UpdateOrganizationRequest {
  int32 id = 1;
  string name = 2;
  string description = 3;
}
```

**Response**: `OrganizationResponse`

### DeleteOrganization

Deletes an organization. **Warning**: This also deletes all associated workspaces.

**Request**:
```protobuf
message DeleteOrganizationRequest {
  int32 id = 1;
}
```

**Response**: `common.Empty`

**Permissions**: Only organization admins can delete.

### InviteUser

Sends an invitation to join the organization.

**Request**:
```protobuf
message InviteUserRequest {
  int32 organization_id = 1;
  string email = 2;      // Required: User email
  string role = 3;       // Required: "admin", "member", or "viewer"
}
```

**Response**:
```protobuf
message InviteUserResponse {
  Invitation invitation = 1;
}
```

**Invitation Message**:
```protobuf
message Invitation {
  int32 id = 1;
  int32 organization_id = 2;
  string email = 3;
  string role = 4;
  int32 invited_by = 5;
  bool accepted = 6;
  wws.common.Timestamp expires_at = 7;  // 7 days from creation
  wws.common.Timestamp created_at = 8;
}
```

### AcceptInvitation

Accepts an organization invitation.

**Request**:
```protobuf
message AcceptInvitationRequest {
  int32 invitation_id = 1;
}
```

**Response**: `common.Empty`

### ListMembers

Lists all members of an organization.

**Request**:
```protobuf
message ListMembersRequest {
  int32 organization_id = 1;
  wws.common.PaginationRequest pagination = 2;
}
```

**Response**:
```protobuf
message ListMembersResponse {
  repeated Member members = 1;
  wws.common.PaginationResponse pagination = 2;
}
```

**Member Message**:
```protobuf
message Member {
  int32 id = 1;
  wws.user.User user = 2;
  string role = 3;           // "admin", "member", "viewer"
  bool accepted = 4;
  wws.common.Timestamp invited_at = 5;
  wws.common.Timestamp joined_at = 6;
}
```

### UpdateMemberRole

Updates a member's role in the organization.

**Request**:
```protobuf
message UpdateMemberRoleRequest {
  int32 organization_id = 1;
  int32 user_id = 2;
  string role = 3;  // "admin", "member", "viewer"
}
```

**Response**: `MemberResponse`

**Permissions**: Only admins can update roles. Cannot demote yourself.

### RemoveMember

Removes a member from the organization.

**Request**:
```protobuf
message RemoveMemberRequest {
  int32 organization_id = 1;
  int32 user_id = 2;
}
```

**Response**: `common.Empty`

**Permissions**: Only admins can remove members. Cannot remove yourself.

---

## User Service

**Service**: `wws.user.UserService`  
**Route Prefix**: `/api/users`

### GetCurrentUser

Retrieves the authenticated user's profile.

**Request**:
```protobuf
message GetCurrentUserRequest {}
```

**Response**:
```protobuf
message UserResponse {
  User user = 1;
}
```

**User Message**:
```protobuf
message User {
  int32 id = 1;
  string github_id = 2;      // GitHub user ID
  string username = 3;       // GitHub username
  string email = 4;          // Primary email
  string avatar_url = 5;     // GitHub avatar URL
  wws.common.Timestamp created_at = 6;
}
```

### GetUser

Retrieves a specific user by ID.

**Request**:
```protobuf
message GetUserRequest {
  int32 id = 1;
}
```

**Response**: `UserResponse`

**Permissions**: Only accessible within the same organization.

### ListUsers

Lists all users in an organization.

**Request**:
```protobuf
message ListUsersRequest {
  int32 organization_id = 1;
  wws.common.PaginationRequest pagination = 2;
}
```

**Response**:
```protobuf
message ListUsersResponse {
  repeated User users = 1;
  wws.common.PaginationResponse pagination = 2;
}
```

**Permissions**: Only organization members can list users.

---

## Workspace Service

**Service**: `wws.workspace.WorkspaceService`  
**Route Prefix**: `/api/workspaces`

### ListWorkspaces

Lists all workspaces in an organization.

**Request**:
```protobuf
message ListWorkspacesRequest {
  int32 organization_id = 1;
  wws.common.PaginationRequest pagination = 2;
}
```

**Response**:
```protobuf
message ListWorkspacesResponse {
  repeated Workspace workspaces = 1;
  wws.common.PaginationResponse pagination = 2;
}
```

**Workspace Message**:
```protobuf
message Workspace {
  int32 id = 1;
  string tag = 2;              // Unique workspace tag
  string name = 3;             // Display name
  int32 organization_id = 4;
  int32 owner_id = 5;
  string provider = 6;         // "kvm", "podman", "digitalocean"
  wws.common.Status status = 7; // Current status
  string endpoint = 8;         // Main endpoint
  string ssh_host = 9;         // SSH host
  int32 ssh_port = 10;         // SSH port
  string http_host = 11;       // HTTP host
  int32 http_port = 12;        // HTTP port
  string code_host = 13;       // code-server host
  int32 code_port = 14;        // code-server port
  int32 cpu = 15;              // CPU cores
  int32 memory = 16;           // Memory in GB
  int32 storage = 17;          // Storage in GB
  string region = 18;          // Deployment region
  string config = 19;          // JSON config
  wws.common.Timestamp created_at = 20;
  wws.common.Timestamp updated_at = 21;
}
```

### GetWorkspace

Retrieves a specific workspace.

**Request**:
```protobuf
message GetWorkspaceRequest {
  int32 id = 1;
}
```

**Response**: `WorkspaceResponse`

### CreateWorkspace

Creates a new workspace.

**Request**:
```protobuf
message CreateWorkspaceRequest {
  string name = 1;                    // Required: Workspace name
  int32 organization_id = 2;          // Required: Parent organization
  int32 cpu = 3;                      // Optional: Default 2
  int32 memory = 4;                   // Optional: Default 4
  int32 storage = 5;                  // Optional: Default 20
  repeated string languages = 6;      // Optional: ["python", "javascript", "go", "rust"]
  string region = 7;                  // Optional: Default "local"
}
```

**Response**: `WorkspaceResponse`

**Validation**:
- `name`: Required, 1-100 characters
- `organization_id`: Required, must exist and user must be member
- `cpu`: 1-16 cores
- `memory`: 1-32 GB
- `storage`: 10-500 GB

**Example**:
```typescript
import { workspace } from './proto/workspace/workspace_pb';

const request = new workspace.CreateWorkspaceRequest();
request.setName("feature-branch-dev");
request.setOrganizationId(1);
request.setCpu(2);
request.setMemory(4);
request.setStorage(20);
request.setLanguagesList(["python", "javascript"]);

const response = await workspaceService.createWorkspace(request);
// Workspace will be created with status STATUS_CREATING
```

### UpdateWorkspace

Updates an existing workspace configuration.

**Request**:
```protobuf
message UpdateWorkspaceRequest {
  int32 id = 1;
  string name = 2;
  int32 cpu = 3;
  int32 memory = 4;
  int32 storage = 5;
  repeated string languages = 6;
  string region = 7;
}
```

**Response**: `WorkspaceResponse`

**Note**: Resource changes may require workspace restart.

### DeleteWorkspace

Destroys a workspace and all associated resources.

**Request**:
```protobuf
message DeleteWorkspaceRequest {
  int32 id = 1;
}
```

**Response**: `common.Empty`

**Warning**: This action is irreversible. All data will be lost.

### StartWorkspace

Starts a stopped workspace.

**Request**:
```protobuf
message StartWorkspaceRequest {
  int32 id = 1;
}
```

**Response**: `WorkspaceResponse`

**Status Transition**: STATUS_STOPPED → STATUS_RUNNING

### StopWorkspace

Stops a running workspace.

**Request**:
```protobuf
message StopWorkspaceRequest {
  int32 id = 1;
}
```

**Response**: `WorkspaceResponse`

**Status Transition**: STATUS_RUNNING → STATUS_STOPPED

### RestartWorkspace

Restarts a workspace (stop then start).

**Request**:
```protobuf
message RestartWorkspaceRequest {
  int32 id = 1;
}
```

**Response**: `WorkspaceResponse`

### GetWorkspaceLogs

Retrieves logs for a workspace.

**Request**:
```protobuf
message GetWorkspaceLogsRequest {
  int32 id = 1;
  int32 tail_lines = 2;  // Optional: Number of lines, default 100
}
```

**Response**:
```protobuf
message WorkspaceLogsResponse {
  string logs = 1;  // Log output
}
```

### InstallLanguage

Installs a language runtime in a workspace.

**Request**:
```protobuf
message InstallLanguageRequest {
  int32 workspace_id = 1;
  string language = 2;  // "python", "javascript", "go", "rust"
  string version = 3;   // Optional: Specific version
}
```

**Response**:
```protobuf
message InstallLanguageResponse {
  Language language = 1;
}
```

**Language Message**:
```protobuf
message Language {
  int32 id = 1;
  int32 workspace_id = 2;
  string name = 3;
  string version = 4;
  string install_script = 5;
  wws.common.Timestamp installed_at = 6;
}
```

### ListLanguages

Lists all installed languages in a workspace.

**Request**:
```protobuf
message ListLanguagesRequest {
  int32 workspace_id = 1;
}
```

**Response**:
```protobuf
message ListLanguagesResponse {
  repeated Language languages = 1;
}
```

---

## Error Handling

All gRPC methods may return errors with the following codes:

| Code | Description |
|------|-------------|
| 0 (OK) | Success |
| 1 (CANCELLED) | Operation cancelled by client |
| 2 (UNKNOWN) | Unknown error |
| 3 (INVALID_ARGUMENT) | Invalid request parameters |
| 4 (DEADLINE_EXCEEDED) | Request timeout |
| 5 (NOT_FOUND) | Resource not found |
| 6 (ALREADY_EXISTS) | Resource already exists |
| 7 (PERMISSION_DENIED) | Insufficient permissions |
| 8 (RESOURCE_EXHAUSTED) | Quota exceeded |
| 9 (FAILED_PRECONDITION) | Precondition failed |
| 10 (ABORTED) | Operation aborted |
| 11 (OUT_OF_RANGE) | Out of range |
| 12 (UNIMPLEMENTED) | Method not implemented |
| 13 (INTERNAL) | Internal server error |
| 14 (UNAVAILABLE) | Service unavailable |
| 15 (DATA_LOSS) | Data loss |
| 16 (UNAUTHENTICATED) | Authentication required |

**Error Response**:
```protobuf
message Error {
  int32 code = 1;
  string message = 2;
  string field = 3;  // For INVALID_ARGUMENT errors
}
```

---

## Client Setup (TypeScript)

### Installing Dependencies

```bash
npm install @grpc/grpc-js @grpc/proto-loader
```

### Creating a Client

```typescript
import { ChannelCredentials } from '@grpc/grpc-js';
import { WorkspaceServiceClient } from './proto/workspace/workspace_grpc_web_pb';

// Create client
const client = new WorkspaceServiceClient(
  'http://localhost:8080',
  ChannelCredentials.createInsecure(),
  {}
);

// Make request
const request = new workspace.GetWorkspaceRequest();
request.setId(1);

client.getWorkspace(request, {}, (err, response) => {
  if (err) {
    console.error(err);
    return;
  }
  console.log(response.getWorkspace());
});
```

### Using with React

```typescript
import { useState, useEffect } from 'react';
import { WorkspaceServiceClient } from './proto/workspace/workspace_grpc_web_pb';
import { workspace } from './proto/workspace/workspace_pb';

function WorkspaceList({ organizationId }: { organizationId: number }) {
  const [workspaces, setWorkspaces] = useState<workspace.Workspace[]>([]);
  
  useEffect(() => {
    const client = new WorkspaceServiceClient(
      'http://localhost:8080',
      ChannelCredentials.createInsecure(),
      {}
    );
    
    const request = new workspace.ListWorkspacesRequest();
    request.setOrganizationId(organizationId);
    
    client.listWorkspaces(request, {}, (err, response) => {
      if (!err) {
        setWorkspaces(response.getWorkspacesList());
      }
    });
  }, [organizationId]);
  
  return (
    <div>
      {workspaces.map(ws => (
        <div key={ws.getId()}>{ws.getName()}</div>
      ))}
    </div>
  );
}
```

---

## Rate Limiting

- **Authentication endpoints**: 10 requests/minute
- **Read endpoints**: 100 requests/minute
- **Write endpoints**: 30 requests/minute
- **Delete endpoints**: 10 requests/minute

Rate limit headers are included in responses:
- `X-RateLimit-Limit`: Maximum requests per window
- `X-RateLimit-Remaining`: Remaining requests
- `X-RateLimit-Reset`: Unix timestamp when limit resets

---

## Versioning

API version is included in the gRPC service name. Current version: **v1**

Future breaking changes will increment the version (e.g., `AuthServiceV2`).

---

## Changelog

### v1.0.0 (Current)
- Initial protobuf migration
- All endpoints migrated from REST to gRPC-Web
- Added pagination support
- Added type-safe message definitions

---

## Support

For issues or questions:
- GitHub Issues: https://github.com/winmutt/wws/issues
- Documentation: https://github.com/winmutt/wws/docs
