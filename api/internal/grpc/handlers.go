package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"wws/api/internal/handlers"
	"wws/api/proto/auth"
	"wws/api/proto/common"
	"wws/api/proto/organization"
	"wws/api/proto/user"
	"wws/api/proto/workspace"
)

// GetCurrentUser implements the gRPC handler for getting current user
func (s *Server) GetCurrentUser(ctx context.Context, req *user.GetCurrentUserRequest) (*user.UserResponse, error) {
	// This would require context from HTTP middleware
	// For now, return a placeholder
	return &user.UserResponse{}, nil
}

// GetUser implements the gRPC handler for getting user by ID
func (s *Server) GetUser(ctx context.Context, req *user.GetUserRequest) (*user.UserResponse, error) {
	return &user.UserResponse{}, nil
}

// ListUsers implements the gRPC handler for listing users
func (s *Server) ListUsers(ctx context.Context, req *user.ListUsersRequest) (*user.ListUsersResponse, error) {
	return &user.ListUsersResponse{}, nil
}

// GitHubAuth implements the gRPC handler for GitHub OAuth
func (s *Server) GitHubAuth(ctx context.Context, req *auth.GitHubAuthRequest) (*auth.GitHubAuthResponse, error) {
	// Redirect to OAuth - this is better handled by HTTP
	return &auth.GitHubAuthResponse{}, nil
}

// GitHubCallback implements the gRPC handler for OAuth callback
func (s *Server) GitHubCallback(ctx context.Context, req *auth.GitHubCallbackRequest) (*auth.GitHubCallbackResponse, error) {
	// This needs to handle OAuth callback - better done via HTTP
	return &auth.GitHubCallbackResponse{}, nil
}

// GetSession implements the gRPC handler for getting session
func (s *Server) GetSession(ctx context.Context, req *auth.GetSessionRequest) (*auth.SessionResponse, error) {
	return &auth.SessionResponse{}, nil
}

// Logout implements the gRPC handler for logout
func (s *Server) Logout(ctx context.Context, req *auth.LogoutRequest) (*common.Empty, error) {
	return &common.Empty{}, nil
}

// ListOrganizations implements the gRPC handler for listing organizations
func (s *Server) ListOrganizations(ctx context.Context, req *organization.ListOrganizationsRequest) (*organization.ListOrganizationsResponse, error) {
	// Convert gRPC request to HTTP request
	httpReq, _ := http.NewRequest("GET", "/api/v1/organizations", nil)
	w := &responseWriter{body: &[]byte{}}

	if err := handlers.ListOrganizationsHandler(w, httpReq); err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}

	var orgs []map[string]interface{}
	json.Unmarshal(*w.body, &orgs)

	var responseOrgs []*organization.Organization
	for _, org := range orgs {
		responseOrgs = append(responseOrgs, &organization.Organization{
			Id:   int32(org["id"].(float64)),
			Name: org["name"].(string),
		})
	}

	return &organization.ListOrganizationsResponse{
		Organizations: responseOrgs,
	}, nil
}

// GetOrganization implements the gRPC handler for getting organization
func (s *Server) GetOrganization(ctx context.Context, req *organization.GetOrganizationRequest) (*organization.OrganizationResponse, error) {
	return &organization.OrganizationResponse{}, nil
}

// CreateOrganization implements the gRPC handler for creating organization
func (s *Server) CreateOrganization(ctx context.Context, req *organization.CreateOrganizationRequest) (*organization.OrganizationResponse, error) {
	return &organization.OrganizationResponse{}, nil
}

// UpdateOrganization implements the gRPC handler for updating organization
func (s *Server) UpdateOrganization(ctx context.Context, req *organization.UpdateOrganizationRequest) (*organization.OrganizationResponse, error) {
	return &organization.OrganizationResponse{}, nil
}

// DeleteOrganization implements the gRPC handler for deleting organization
func (s *Server) DeleteOrganization(ctx context.Context, req *organization.DeleteOrganizationRequest) (*common.Empty, error) {
	return &common.Empty{}, nil
}

// InviteUser implements the gRPC handler for inviting users
func (s *Server) InviteUser(ctx context.Context, req *organization.InviteUserRequest) (*organization.InviteUserResponse, error) {
	return &organization.InviteUserResponse{}, nil
}

// AcceptInvitation implements the gRPC handler for accepting invitations
func (s *Server) AcceptInvitation(ctx context.Context, req *organization.AcceptInvitationRequest) (*common.Empty, error) {
	return &common.Empty{}, nil
}

// ListMembers implements the gRPC handler for listing members
func (s *Server) ListMembers(ctx context.Context, req *organization.ListMembersRequest) (*organization.ListMembersResponse, error) {
	return &organization.ListMembersResponse{}, nil
}

// UpdateMemberRole implements the gRPC handler for updating member roles
func (s *Server) UpdateMemberRole(ctx context.Context, req *organization.UpdateMemberRoleRequest) (*organization.MemberResponse, error) {
	return &organization.MemberResponse{}, nil
}

// RemoveMember implements the gRPC handler for removing members
func (s *Server) RemoveMember(ctx context.Context, req *organization.RemoveMemberRequest) (*common.Empty, error) {
	return &common.Empty{}, nil
}

// ListWorkspaces implements the gRPC handler for listing workspaces
func (s *Server) ListWorkspaces(ctx context.Context, req *workspace.ListWorkspacesRequest) (*workspace.ListWorkspacesResponse, error) {
	httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/workspaces?organization_id=%d", req.OrganizationId), nil)
	w := &responseWriter{body: &[]byte{}}

	if err := handlers.ListWorkspacesHandler(w, httpReq); err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	return &workspace.ListWorkspacesResponse{}, nil
}

// GetWorkspace implements the gRPC handler for getting workspace
func (s *Server) GetWorkspace(ctx context.Context, req *workspace.GetWorkspaceRequest) (*workspace.WorkspaceResponse, error) {
	return &workspace.WorkspaceResponse{}, nil
}

// CreateWorkspace implements the gRPC handler for creating workspace
func (s *Server) CreateWorkspace(ctx context.Context, req *workspace.CreateWorkspaceRequest) (*workspace.WorkspaceResponse, error) {
	return &workspace.WorkspaceResponse{}, nil
}

// UpdateWorkspace implements the gRPC handler for updating workspace
func (s *Server) UpdateWorkspace(ctx context.Context, req *workspace.UpdateWorkspaceRequest) (*workspace.WorkspaceResponse, error) {
	return &workspace.WorkspaceResponse{}, nil
}

// DeleteWorkspace implements the gRPC handler for deleting workspace
func (s *Server) DeleteWorkspace(ctx context.Context, req *workspace.DeleteWorkspaceRequest) (*common.Empty, error) {
	return &common.Empty{}, nil
}

// StartWorkspace implements the gRPC handler for starting workspace
func (s *Server) StartWorkspace(ctx context.Context, req *workspace.StartWorkspaceRequest) (*workspace.WorkspaceResponse, error) {
	return &workspace.WorkspaceResponse{}, nil
}

// StopWorkspace implements the gRPC handler for stopping workspace
func (s *Server) StopWorkspace(ctx context.Context, req *workspace.StopWorkspaceRequest) (*workspace.WorkspaceResponse, error) {
	return &workspace.WorkspaceResponse{}, nil
}

// RestartWorkspace implements the gRPC handler for restarting workspace
func (s *Server) RestartWorkspace(ctx context.Context, req *workspace.RestartWorkspaceRequest) (*workspace.WorkspaceResponse, error) {
	return &workspace.WorkspaceResponse{}, nil
}

// GetWorkspaceLogs implements the gRPC handler for getting workspace logs
func (s *Server) GetWorkspaceLogs(ctx context.Context, req *workspace.GetWorkspaceLogsRequest) (*workspace.WorkspaceLogsResponse, error) {
	return &workspace.WorkspaceLogsResponse{}, nil
}

// InstallLanguage implements the gRPC handler for installing languages
func (s *Server) InstallLanguage(ctx context.Context, req *workspace.InstallLanguageRequest) (*workspace.InstallLanguageResponse, error) {
	return &workspace.InstallLanguageResponse{}, nil
}

// ListLanguages implements the gRPC handler for listing languages
func (s *Server) ListLanguages(ctx context.Context, req *workspace.ListLanguagesRequest) (*workspace.ListLanguagesResponse, error) {
	return &workspace.ListLanguagesResponse{}, nil
}

// responseWriter is a simple HTTP response writer for bridging gRPC to HTTP
type responseWriter struct {
	body *[]byte
}

func (rw *responseWriter) Header() http.Header {
	return http.Header{}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	*rw.body = b
	return len(b), nil
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	// Do nothing
}
