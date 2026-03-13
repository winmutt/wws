package grpc

import (
	"context"
	"fmt"
	"time"

	"wws/api/internal/db"
	"wws/api/internal/models"
	"wws/api/proto/auth"
	"wws/api/proto/common"
	"wws/api/proto/organization"
	"wws/api/proto/user"
	"wws/api/proto/workspace"
)

// Organization handlers

// ListOrganizations implements the gRPC handler for listing organizations
func (s *Server) ListOrganizations(ctx context.Context, req *organization.ListOrganizationsRequest) (*organization.ListOrganizationsResponse, error) {
	rows, err := db.DB.QueryContext(ctx,
		`SELECT id, name, owner_id, created_at, updated_at FROM organizations ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	defer rows.Close()

	var orgs []*organization.Organization
	for rows.Next() {
		var id, ownerID int
		var name string
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &name, &ownerID, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan organization: %w", err)
		}

		orgs = append(orgs, &organization.Organization{
			Id:        int32(id),
			Name:      name,
			OwnerId:   int32(ownerID),
			CreatedAt: &common.Timestamp{Value: createdAt.Format(time.RFC3339)},
			UpdatedAt: &common.Timestamp{Value: updatedAt.Format(time.RFC3339)},
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating organizations: %w", err)
	}

	return &organization.ListOrganizationsResponse{
		Organizations: orgs,
		Pagination:    convertPagination(req.Pagination),
	}, nil
}

// GetOrganization implements the gRPC handler for getting organization
func (s *Server) GetOrganization(ctx context.Context, req *organization.GetOrganizationRequest) (*organization.OrganizationResponse, error) {
	var org models.Organization
	err := db.DB.QueryRowContext(ctx,
		`SELECT id, name, owner_id, created_at, updated_at FROM organizations WHERE id = ?`,
		req.Id,
	).Scan(&org.ID, &org.Name, &org.OwnerID, &org.CreatedAt, &org.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return &organization.OrganizationResponse{
		Organization: &organization.Organization{
			Id:        int32(org.ID),
			Name:      org.Name,
			OwnerId:   int32(org.OwnerID),
			CreatedAt: &common.Timestamp{Value: org.CreatedAt.Format(time.RFC3339)},
			UpdatedAt: &common.Timestamp{Value: org.UpdatedAt.Format(time.RFC3339)},
		},
	}, nil
}

// CreateOrganization implements the gRPC handler for creating organization
func (s *Server) CreateOrganization(ctx context.Context, req *organization.CreateOrganizationRequest) (*organization.OrganizationResponse, error) {
	// Get current user from context (would be set by auth middleware)
	userID := int32(1) // Placeholder - get from context
	now := time.Now()
	result, err := db.DB.ExecContext(ctx,
		`INSERT INTO organizations (name, owner_id, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		req.Name, userID, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	orgID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get organization ID: %w", err)
	}

	return &organization.OrganizationResponse{
		Organization: &organization.Organization{
			Id:        int32(orgID),
			Name:      req.Name,
			OwnerId:   userID,
			CreatedAt: &common.Timestamp{Value: now.Format(time.RFC3339)},
			UpdatedAt: &common.Timestamp{Value: now.Format(time.RFC3339)},
		},
	}, nil
}

// UpdateOrganization implements the gRPC handler for updating organization
func (s *Server) UpdateOrganization(ctx context.Context, req *organization.UpdateOrganizationRequest) (*organization.OrganizationResponse, error) {
	now := time.Now()
	_, err := db.DB.ExecContext(ctx,
		`UPDATE organizations SET name = ?, description = ?, updated_at = ? WHERE id = ?`,
		req.Name, req.Description, now, req.Id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}

	return &organization.OrganizationResponse{
		Organization: &organization.Organization{
			Id:          req.Id,
			Name:        req.Name,
			Description: req.Description,
			UpdatedAt:   &common.Timestamp{Value: now.Format(time.RFC3339)},
		},
	}, nil
}

// DeleteOrganization implements the gRPC handler for deleting organization
func (s *Server) DeleteOrganization(ctx context.Context, req *organization.DeleteOrganizationRequest) (*common.Empty, error) {
	_, err := db.DB.ExecContext(ctx,
		`DELETE FROM organizations WHERE id = ?`,
		req.Id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to delete organization: %w", err)
	}

	return &common.Empty{}, nil
}

// InviteUser implements the gRPC handler for inviting users
func (s *Server) InviteUser(ctx context.Context, req *organization.InviteUserRequest) (*organization.InviteUserResponse, error) {
	// TODO: Implement invitation logic
	return &organization.InviteUserResponse{}, nil
}

// AcceptInvitation implements the gRPC handler for accepting invitations
func (s *Server) AcceptInvitation(ctx context.Context, req *organization.AcceptInvitationRequest) (*common.Empty, error) {
	// TODO: Implement invitation acceptance
	return &common.Empty{}, nil
}

// ListMembers implements the gRPC handler for listing members
func (s *Server) ListMembers(ctx context.Context, req *organization.ListMembersRequest) (*organization.ListMembersResponse, error) {
	// TODO: Implement member listing
	return &organization.ListMembersResponse{}, nil
}

// UpdateMemberRole implements the gRPC handler for updating member roles
func (s *Server) UpdateMemberRole(ctx context.Context, req *organization.UpdateMemberRoleRequest) (*organization.MemberResponse, error) {
	// TODO: Implement role update
	return &organization.MemberResponse{}, nil
}

// RemoveMember implements the gRPC handler for removing members
func (s *Server) RemoveMember(ctx context.Context, req *organization.RemoveMemberRequest) (*common.Empty, error) {
	// TODO: Implement member removal
	return &common.Empty{}, nil
}

// User handlers

// GetCurrentUser implements the gRPC handler for getting current user
func (s *Server) GetCurrentUser(ctx context.Context, req *user.GetCurrentUserRequest) (*user.UserResponse, error) {
	// This would require context from HTTP middleware
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

// Auth handlers

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

// Workspace handlers

// ListWorkspaces implements the gRPC handler for listing workspaces
func (s *Server) ListWorkspaces(ctx context.Context, req *workspace.ListWorkspacesRequest) (*workspace.ListWorkspacesResponse, error) {
	rows, err := db.DB.QueryContext(ctx,
		`SELECT id, tag, name, organization_id, owner_id, provider, status, config, region, created_at, updated_at 
		 FROM workspaces WHERE organization_id = ? AND deleted_at IS NULL`,
		req.OrganizationId,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}
	defer rows.Close()

	var workspaces []*workspace.Workspace
	for rows.Next() {
		var ws models.Workspace
		if err := rows.Scan(&ws.ID, &ws.Tag, &ws.Name, &ws.OrganizationID, &ws.OwnerID, &ws.Provider, &ws.Status, &ws.Config, &ws.Region, &ws.CreatedAt, &ws.UpdatedAt); err != nil {
			continue
		}

		workspaces = append(workspaces, &workspace.Workspace{
			Id:             int32(ws.ID),
			Tag:            ws.Tag,
			Name:           ws.Name,
			OrganizationId: int32(ws.OrganizationID),
			OwnerId:        int32(ws.OwnerID),
			Provider:       ws.Provider,
			Status:         convertStatus(ws.Status),
			CreatedAt:      &common.Timestamp{Value: ws.CreatedAt.Format(time.RFC3339)},
			UpdatedAt:      &common.Timestamp{Value: ws.UpdatedAt.Format(time.RFC3339)},
		})
	}

	return &workspace.ListWorkspacesResponse{
		Workspaces: workspaces,
	}, nil
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

// Helper functions

func convertPagination(req *common.PaginationRequest) *common.PaginationResponse {
	if req == nil {
		return nil
	}
	return &common.PaginationResponse{
		TotalCount: 0,
	}
}

func convertStatus(status string) common.Status {
	switch status {
	case "running", "active":
		return common.Status_STATUS_RUNNING
	case "stopped", "inactive":
		return common.Status_STATUS_STOPPED
	case "creating":
		return common.Status_STATUS_CREATING
	case "deleting":
		return common.Status_STATUS_DELETING
	case "error":
		return common.Status_STATUS_ERROR
	default:
		return common.Status_STATUS_ACTIVE
	}
}

func convertUser(u models.User) *user.User {
	return &user.User{
		Id:        int32(u.ID),
		GithubId:  u.GithubID,
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: &common.Timestamp{Value: u.CreatedAt.Format(time.RFC3339)},
	}
}
