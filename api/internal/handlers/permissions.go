package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"wws/api/internal/permissions"
)

// TeamHandler handles team-related HTTP requests
type TeamHandler struct{}

// CreateTeamRequest represents a request to create a team
type CreateTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CreateTeamHandler creates a new team
func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)
	if userID == 0 {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	orgID := getOrgIDFromRequest(r)
	if orgID == 0 {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Organization ID required"))
		return
	}

	var req CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid request body: %v", err))
		return
	}

	if req.Name == "" {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Team name is required"))
		return
	}

	team, err := permissions.CreateTeam(context.Background(), orgID, userID, req.Name, req.Description)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to create team: %v", err))
		return
	}

	WriteJSON(w, http.StatusCreated, team)
}

// ListTeamsHandler retrieves all teams for an organization
func (h *TeamHandler) ListTeams(w http.ResponseWriter, r *http.Request) {
	orgID := getOrgIDFromRequest(r)
	if orgID == 0 {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Organization ID required"))
		return
	}

	teams, err := permissions.ListTeams(context.Background(), orgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to list teams: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, teams)
}

// GetTeamHandler retrieves a specific team
func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	teamIDStr := vars.Get("id")
	if teamIDStr == "" {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Team ID required"))
		return
	}

	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid team ID"))
		return
	}

	teams, err := permissions.ListTeams(context.Background(), getOrgIDFromRequest(r))
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to list teams: %v", err))
		return
	}

	for _, team := range teams {
		if team.ID == teamID {
			WriteJSON(w, http.StatusOK, team)
			return
		}
	}

	WriteError(w, http.StatusNotFound, fmt.Errorf("Team not found"))
}

// GetTeamMembersHandler retrieves all members of a team
func (h *TeamHandler) GetTeamMembers(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	teamIDStr := vars.Get("team_id")
	if teamIDStr == "" {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Team ID required"))
		return
	}

	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid team ID"))
		return
	}

	members, err := permissions.GetTeamMembers(context.Background(), teamID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to get team members: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, members)
}

// GetUserTeamsHandler retrieves all teams a user belongs to
func (h *TeamHandler) GetUserTeams(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)
	if userID == 0 {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	teams, err := permissions.GetUserTeams(context.Background(), userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to get user teams: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, teams)
}

// AddTeamMemberRequest represents a request to add a team member
type AddTeamMemberRequest struct {
	UserID int `json:"user_id"`
	RoleID int `json:"role_id"`
}

// AddTeamMemberHandler adds a member to a team
func (h *TeamHandler) AddTeamMember(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)
	if userID == 0 {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	var req AddTeamMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid request body: %v", err))
		return
	}

	vars := r.URL.Query()
	teamIDStr := vars.Get("team_id")
	if teamIDStr == "" {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Team ID required"))
		return
	}

	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid team ID"))
		return
	}

	if req.UserID == 0 || req.RoleID == 0 {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("User ID and Role ID are required"))
		return
	}

	err = permissions.AddTeamMember(context.Background(), teamID, req.UserID, req.RoleID, userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to add team member: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Team member added successfully"})
}

// RemoveTeamMemberHandler removes a member from a team
func (h *TeamHandler) RemoveTeamMember(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)
	if userID == 0 {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}
	_ = userID // Could be used for audit logging

	vars := r.URL.Query()
	teamIDStr := vars.Get("team_id")
	userIDStr := vars.Get("user_id")

	if teamIDStr == "" || userIDStr == "" {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Team ID and User ID required"))
		return
	}

	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid team ID"))
		return
	}

	targetUserID, err := strconv.Atoi(userIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid user ID"))
		return
	}

	err = permissions.RemoveTeamMember(context.Background(), teamID, targetUserID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to remove team member: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Team member removed successfully"})
}

// RoleHandler handles role-related HTTP requests
type RoleHandler struct{}

// CreateRoleRequest represents a request to create a role
type CreateRoleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	IsDefault   bool     `json:"is_default"`
}

// CreateRoleHandler creates a new team role
func (h *RoleHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid request body: %v", err))
		return
	}

	if req.Name == "" {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Role name is required"))
		return
	}

	// Convert string permissions to TeamPermission type
	perms := make([]permissions.TeamPermission, len(req.Permissions))
	for i, p := range req.Permissions {
		perms[i] = permissions.TeamPermission(p)
	}

	role, err := permissions.CreateTeamRole(context.Background(), req.Name, req.Description, perms, req.IsDefault)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to create role: %v", err))
		return
	}

	WriteJSON(w, http.StatusCreated, role)
}

// ListRolesHandler retrieves all team roles
func (h *RoleHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	// For now, return empty list - implement listing when needed
	WriteJSON(w, http.StatusOK, []permissions.TeamRole{})
}

// PermissionHandler handles permission checking HTTP requests
type PermissionHandler struct{}

// CheckPermissionRequest represents a permission check request
type CheckPermissionRequest struct {
	UserID     int    `json:"user_id"`
	Permission string `json:"permission"`
}

// CheckPermissionHandler checks if a user has a specific permission
func (h *PermissionHandler) CheckPermission(w http.ResponseWriter, r *http.Request) {
	var req CheckPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid request body: %v", err))
		return
	}

	result := permissions.CheckPermission(context.Background(), req.UserID, permissions.TeamPermission(req.Permission))
	WriteJSON(w, http.StatusOK, result)
}

// WorkspaceAccessHandler handles workspace access HTTP requests
type WorkspaceAccessHandler struct{}

// GrantWorkspaceAccessRequest represents a request to grant workspace access
type GrantWorkspaceAccessRequest struct {
	TeamID      int    `json:"team_id"`
	WorkspaceID int    `json:"workspace_id"`
	AccessLevel string `json:"access_level"`
}

// GrantWorkspaceAccessHandler grants a team access to a workspace
func (h *WorkspaceAccessHandler) GrantWorkspaceAccess(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)
	if userID == 0 {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	var req GrantWorkspaceAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid request body: %v", err))
		return
	}

	if req.TeamID == 0 || req.WorkspaceID == 0 {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Team ID and Workspace ID are required"))
		return
	}

	validLevels := map[string]bool{"read": true, "write": true, "admin": true}
	if !validLevels[req.AccessLevel] {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid access level"))
		return
	}

	err := permissions.GrantWorkspaceAccess(context.Background(), req.TeamID, req.WorkspaceID, req.AccessLevel, userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to grant workspace access: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Workspace access granted successfully"})
}

// GetTeamWorkspaceAccessHandler retrieves workspace access for a team
func (h *WorkspaceAccessHandler) GetTeamWorkspaceAccess(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	teamIDStr := vars.Get("team_id")
	if teamIDStr == "" {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Team ID required"))
		return
	}

	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid team ID"))
		return
	}

	access, err := permissions.GetTeamWorkspaceAccess(context.Background(), teamID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to get workspace access: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, access)
}

// HasWorkspaceAccessHandler checks if a team has access to a workspace
func (h *WorkspaceAccessHandler) HasWorkspaceAccess(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	teamIDStr := vars.Get("team_id")
	workspaceIDStr := vars.Get("workspace_id")
	accessLevel := vars.Get("access_level")

	if teamIDStr == "" || workspaceIDStr == "" {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Team ID and Workspace ID required"))
		return
	}

	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid team ID"))
		return
	}

	workspaceID, err := strconv.Atoi(workspaceIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid workspace ID"))
		return
	}

	hasAccess := permissions.HasWorkspaceAccess(context.Background(), teamID, workspaceID, accessLevel)
	WriteJSON(w, http.StatusOK, map[string]bool{"has_access": hasAccess})
}
