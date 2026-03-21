package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"wws/api/internal/models"

	"github.com/google/uuid"
)

// WorkspaceSharingHandler handles workspace sharing requests
type WorkspaceSharingHandler struct {
	DB *sql.DB
}

// NewWorkspaceSharingHandler creates a new workspace sharing handler
func NewWorkspaceSharingHandler(db *sql.DB) *WorkspaceSharingHandler {
	return &WorkspaceSharingHandler{DB: db}
}

// ShareWorkspace adds a member to a workspace
// @Summary Share workspace with a user
// @Description Add a user as a member to a workspace with specified role and permissions
// @Tags workspace-sharing
// @Accept json
// @Produce json
// @Param workspace_id path int true "Workspace ID"
// @Param request body models.WorkspaceShareRequest true "Share request"
// @Success 200 {object} models.WorkspaceShareResponse
// @Router /api/v1/workspaces/{id}/share [post]
func (h *WorkspaceSharingHandler) ShareWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get workspace ID from URL
	parts := strings.Split(r.URL.Path, "/")
	var workspaceID int
	for i, part := range parts {
		if part == "workspaces" && i+1 < len(parts) {
			fmt.Sscanf(parts[i+1], "%d", &workspaceID)
			break
		}
	}

	if workspaceID == 0 {
		http.Error(w, "Invalid workspace ID", http.StatusBadRequest)
		return
	}

	var req models.WorkspaceShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate role
	validRoles := map[string]bool{"owner": true, "admin": true, "editor": true, "viewer": true}
	if !validRoles[req.Role] {
		http.Error(w, "Invalid role. Use: owner, admin, editor, or viewer", http.StatusBadRequest)
		return
	}

	// Find user by ID, username, or email
	var userID int
	var username string
	var email string

	if req.UserID != nil {
		userID = *req.UserID
		err := h.DB.QueryRow("SELECT username, email FROM users WHERE id = ?", userID).Scan(&username, &email)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
	} else if req.Username != nil {
		err := h.DB.QueryRow("SELECT id, email FROM users WHERE username = ?", *req.Username).Scan(&userID, &email)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		username = *req.Username
	} else if req.Email != nil {
		err := h.DB.QueryRow("SELECT id, username FROM users WHERE email = ?", *req.Email).Scan(&userID, &username)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		email = *req.Email
	} else {
		http.Error(w, "Must provide user_id, username, or email", http.StatusBadRequest)
		return
	}

	// Check if user is already a member
	var existingID int
	err := h.DB.QueryRow("SELECT id FROM workspace_members WHERE workspace_id = ? AND user_id = ?", workspaceID, userID).Scan(&existingID)
	if err == nil {
		http.Error(w, "User is already a member of this workspace", http.StatusConflict)
		return
	}

	// Get current user ID from context (would be set by auth middleware)
	currentUserID := 1 // Placeholder - would come from auth context

	// Build permissions JSON
	permissions := models.WorkspaceMemberPermissions{}
	if req.Permissions != nil {
		permissions = *req.Permissions
	} else {
		// Set default permissions based on role
		permissions = getDefaultPermissions(req.Role)
	}

	permissionsJSON, _ := json.Marshal(permissions)

	// Insert workspace member
	_, err = h.DB.Exec(`
		INSERT INTO workspace_members (workspace_id, user_id, username, email, role, permissions, invited_by, status, invited_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'pending', ?)
	`, workspaceID, userID, username, email, req.Role, permissionsJSON, currentUserID, time.Now())

	if err != nil {
		http.Error(w, "Failed to share workspace: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the share response
	response := models.WorkspaceShareResponse{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Username:    username,
		Email:       email,
		Role:        req.Role,
		Permissions: permissions,
		Status:      "pending",
		InvitedAt:   time.Now(),
		InvitedBy:   currentUserID,
	}

	WriteJSON(w, http.StatusOK, response)
}

// ListWorkspaceMembers lists all members of a workspace
// @Summary List workspace members
// @Description Get all members with access to a workspace
// @Tags workspace-sharing
// @Produce json
// @Param workspace_id path int true "Workspace ID"
// @Success 200 {object} models.WorkspaceListMembersResponse
// @Router /api/v1/workspaces/{id}/members [get]
func (h *WorkspaceSharingHandler) ListWorkspaceMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get workspace ID from URL
	parts := strings.Split(r.URL.Path, "/")
	var workspaceID int
	for i, part := range parts {
		if part == "workspaces" && i+1 < len(parts) {
			fmt.Sscanf(parts[i+1], "%d", &workspaceID)
			break
		}
	}

	if workspaceID == 0 {
		http.Error(w, "Invalid workspace ID", http.StatusBadRequest)
		return
	}

	// Query workspace members
	rows, err := h.DB.Query(`
		SELECT id, workspace_id, user_id, username, email, role, permissions, invited_at, status, invited_by
		FROM workspace_members
		WHERE workspace_id = ? AND status = 'active'
		ORDER BY invited_at DESC
	`, workspaceID)

	if err != nil {
		http.Error(w, "Failed to list members: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	members := []models.WorkspaceShareResponse{}
	for rows.Next() {
		var member models.WorkspaceShareResponse
		var permissionsJSON string
		var invitedAt time.Time

		err := rows.Scan(
			&member.ID, &member.WorkspaceID, &member.UserID, &member.Username, &member.Email,
			&member.Role, &permissionsJSON, &invitedAt, &member.Status, &member.InvitedBy,
		)
		if err != nil {
			continue
		}

		// Parse permissions
		json.Unmarshal([]byte(permissionsJSON), &member.Permissions)
		member.InvitedAt = invitedAt

		members = append(members, member)
	}

	response := models.WorkspaceListMembersResponse{
		Members: members,
		Total:   len(members),
	}

	WriteJSON(w, http.StatusOK, response)
}

// RemoveWorkspaceMember removes a member from a workspace
// @Summary Remove workspace member
// @Description Remove a user from a workspace
// @Tags workspace-sharing
// @Produce json
// @Param workspace_id path int true "Workspace ID"
// @Param user_id path int true "User ID"
// @Success 204 No Content
// @Router /api/v1/workspaces/{id}/members/{user_id} [delete]
func (h *WorkspaceSharingHandler) RemoveWorkspaceMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get workspace ID and user ID from URL
	parts := strings.Split(r.URL.Path, "/")
	var workspaceID, userID int
	for i, part := range parts {
		if part == "workspaces" && i+1 < len(parts) {
			fmt.Sscanf(parts[i+1], "%d", &workspaceID)
		}
		if part == "members" && i+1 < len(parts) {
			fmt.Sscanf(parts[i+1], "%d", &userID)
		}
	}

	if workspaceID == 0 || userID == 0 {
		http.Error(w, "Invalid workspace or user ID", http.StatusBadRequest)
		return
	}

	// Update member status to removed
	_, err := h.DB.Exec(`
		UPDATE workspace_members
		SET status = 'removed'
		WHERE workspace_id = ? AND user_id = ? AND status = 'active'
	`, workspaceID, userID)

	if err != nil {
		http.Error(w, "Failed to remove member: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateWorkspaceMemberRole updates a member's role and permissions
// @Summary Update workspace member
// @Description Update a member's role and permissions in a workspace
// @Tags workspace-sharing
// @Accept json
// @Produce json
// @Param workspace_id path int true "Workspace ID"
// @Param user_id path int true "User ID"
// @Param request body map[string]interface{} true "Update request with role and/or permissions"
// @Success 200 {object} models.WorkspaceShareResponse
// @Router /api/v1/workspaces/{id}/members/{user_id} [put]
func (h *WorkspaceSharingHandler) UpdateWorkspaceMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get workspace ID and user ID from URL
	parts := strings.Split(r.URL.Path, "/")
	var workspaceID, userID int
	for i, part := range parts {
		if part == "workspaces" && i+1 < len(parts) {
			fmt.Sscanf(parts[i+1], "%d", &workspaceID)
		}
		if part == "members" && i+1 < len(parts) {
			fmt.Sscanf(parts[i+1], "%d", &userID)
		}
	}

	if workspaceID == 0 || userID == 0 {
		http.Error(w, "Invalid workspace or user ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Role        *string                            `json:"role,omitempty"`
		Permissions *models.WorkspaceMemberPermissions `json:"permissions,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate role if provided
	if req.Role != nil {
		validRoles := map[string]bool{"owner": true, "admin": true, "editor": true, "viewer": true}
		if !validRoles[*req.Role] {
			http.Error(w, "Invalid role", http.StatusBadRequest)
			return
		}
	}

	// Build update query
	var permissionsJSON []byte
	if req.Permissions != nil {
		permissionsJSON, _ = json.Marshal(req.Permissions)
	}

	// Update member
	if permissionsJSON != nil {
		_, err := h.DB.Exec(`
			UPDATE workspace_members
			SET role = COALESCE(?, role), permissions = ?
			WHERE workspace_id = ? AND user_id = ?
		`, *req.Role, permissionsJSON, workspaceID, userID)

		if err != nil {
			http.Error(w, "Failed to update member: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else if req.Role != nil {
		_, err := h.DB.Exec(`
			UPDATE workspace_members
			SET role = ?
			WHERE workspace_id = ? AND user_id = ?
		`, *req.Role, workspaceID, userID)

		if err != nil {
			http.Error(w, "Failed to update member: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Fetch updated member
	var member models.WorkspaceShareResponse
	var permissionsJSONStr string
	err := h.DB.QueryRow(`
		SELECT id, workspace_id, user_id, username, email, role, permissions, invited_at, status, invited_by
		FROM workspace_members
		WHERE workspace_id = ? AND user_id = ?
	`, workspaceID, userID).Scan(
		&member.ID, &member.WorkspaceID, &member.UserID, &member.Username, &member.Email,
		&member.Role, &permissionsJSONStr, &member.InvitedAt, &member.Status, &member.InvitedBy,
	)

	if err != nil {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}

	json.Unmarshal([]byte(permissionsJSONStr), &member.Permissions)
	WriteJSON(w, http.StatusOK, member)
}

// AcceptWorkspaceInvitation accepts a workspace invitation
// @Summary Accept workspace invitation
// @Description Accept an invitation to join a workspace
// @Tags workspace-sharing
// @Produce json
// @Param invitation_token path string true "Invitation token"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/workspaces/invitations/accept [post]
func (h *WorkspaceSharingHandler) AcceptWorkspaceInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context (would be set by auth middleware)
	userID := 1 // Placeholder - would come from auth context

	type Request struct {
		InvitationToken string `json:"invitation_token"`
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// In a real implementation, we would validate the invitation token
	// For now, we'll just activate the pending membership
	_, err := h.DB.Exec(`
		UPDATE workspace_members
		SET status = 'active', joined_at = ?
		WHERE user_id = ? AND status = 'pending'
	`, time.Now(), userID)

	if err != nil {
		http.Error(w, "Failed to accept invitation: "+err.Error(), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Workspace invitation accepted",
	})
}

// GetWorkspaceMember gets a specific member's details
func (h *WorkspaceSharingHandler) GetWorkspaceMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get workspace ID and user ID from URL
	parts := strings.Split(r.URL.Path, "/")
	var workspaceID, userID int
	for i, part := range parts {
		if part == "workspaces" && i+1 < len(parts) {
			fmt.Sscanf(parts[i+1], "%d", &workspaceID)
		}
		if part == "members" && i+1 < len(parts) {
			fmt.Sscanf(parts[i+1], "%d", &userID)
		}
	}

	if workspaceID == 0 || userID == 0 {
		http.Error(w, "Invalid workspace or user ID", http.StatusBadRequest)
		return
	}

	var member models.WorkspaceShareResponse
	var permissionsJSON string
	var invitedAt time.Time

	err := h.DB.QueryRow(`
		SELECT id, workspace_id, user_id, username, email, role, permissions, invited_at, status, invited_by
		FROM workspace_members
		WHERE workspace_id = ? AND user_id = ?
	`, workspaceID, userID).Scan(
		&member.ID, &member.WorkspaceID, &member.UserID, &member.Username, &member.Email,
		&member.Role, &permissionsJSON, &invitedAt, &member.Status, &member.InvitedBy,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "Failed to get member: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.Unmarshal([]byte(permissionsJSON), &member.Permissions)
	member.InvitedAt = invitedAt

	WriteJSON(w, http.StatusOK, member)
}

// getDefaultPermissions returns default permissions based on role
func getDefaultPermissions(role string) models.WorkspaceMemberPermissions {
	switch role {
	case "owner":
		return models.WorkspaceMemberPermissions{
			CanView: true, CanEdit: true, CanShare: true, CanDelete: true,
			CanManageUsers: true, CanViewLogs: true, CanStartStop: true,
		}
	case "admin":
		return models.WorkspaceMemberPermissions{
			CanView: true, CanEdit: true, CanShare: true, CanDelete: false,
			CanManageUsers: true, CanViewLogs: true, CanStartStop: true,
		}
	case "editor":
		return models.WorkspaceMemberPermissions{
			CanView: true, CanEdit: true, CanShare: false, CanDelete: false,
			CanManageUsers: false, CanViewLogs: false, CanStartStop: true,
		}
	case "viewer":
		return models.WorkspaceMemberPermissions{
			CanView: true, CanEdit: false, CanShare: false, CanDelete: false,
			CanManageUsers: false, CanViewLogs: false, CanStartStop: false,
		}
	default:
		return models.WorkspaceMemberPermissions{
			CanView: true, CanEdit: false, CanShare: false, CanDelete: false,
			CanManageUsers: false, CanViewLogs: false, CanStartStop: false,
		}
	}
}

// GenerateInvitationLink generates an invitation link for a workspace
func (h *WorkspaceSharingHandler) GenerateInvitationLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get workspace ID from URL
	parts := strings.Split(r.URL.Path, "/")
	var workspaceID int
	for i, part := range parts {
		if part == "workspaces" && i+1 < len(parts) {
			fmt.Sscanf(parts[i+1], "%d", &workspaceID)
			break
		}
	}

	if workspaceID == 0 {
		http.Error(w, "Invalid workspace ID", http.StatusBadRequest)
		return
	}

	type Request struct {
		Role string `json:"role"`
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate invitation token
	token := uuid.New().String()

	// In a real implementation, we would store this invitation
	// and send an email to the user

	response := map[string]interface{}{
		"invitation_token": token,
		"invitation_url":   fmt.Sprintf("/api/v1/workspaces/invitations/accept?token=%s", token),
		"role":             req.Role,
	}

	WriteJSON(w, http.StatusOK, response)
}
