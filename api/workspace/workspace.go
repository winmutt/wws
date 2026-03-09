package workspace

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"wws/api/internal/db"
	"wws/api/internal/models"
	"wws/api/provisioner/podman"
	"wws/api/provisioner/provider"

	"github.com/google/uuid"
)

type contextKey string

const userIDKey contextKey = "user_id"

// WorkspaceHandler handles workspace-related HTTP requests
type WorkspaceHandler struct {
	provider provider.Provider
}

// NewWorkspaceHandler creates a new WorkspaceHandler
func NewWorkspaceHandler() *WorkspaceHandler {
	return &WorkspaceHandler{
		provider: podman.NewPodmanProvider(""),
	}
}

// WorkspaceRequest represents a workspace creation/update request
type WorkspaceRequest struct {
	Name           string   `json:"name"`
	OrganizationID int      `json:"organization_id"`
	CPU            int      `json:"cpu,omitempty"`
	Memory         int      `json:"memory,omitempty"`
	Storage        int      `json:"storage,omitempty"`
	Languages      []string `json:"languages,omitempty"`
	Region         string   `json:"region,omitempty"`
}

// WorkspaceResponse represents a workspace response
type WorkspaceResponse struct {
	ID             int       `json:"id"`
	Tag            string    `json:"tag"`
	Name           string    `json:"name"`
	OrganizationID int       `json:"organization_id"`
	OwnerID        int       `json:"owner_id"`
	Provider       string    `json:"provider"`
	Status         string    `json:"status"`
	Endpoint       string    `json:"endpoint,omitempty"`
	SSHHost        string    `json:"ssh_host,omitempty"`
	SSHPort        int       `json:"ssh_port,omitempty"`
	HTTPHost       string    `json:"http_host,omitempty"`
	HTTPPort       int       `json:"http_port,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// NewWorkspace creates a new workspace
func (h *WorkspaceHandler) NewWorkspace(w http.ResponseWriter, r *http.Request) {
	userIDVal, ok := r.Context().Value(userIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userIDVal

	var req WorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" || req.OrganizationID == 0 {
		http.Error(w, "Name and organization_id are required", http.StatusBadRequest)
		return
	}

	// Check organization membership
	if !isMemberOfOrg(userID, req.OrganizationID) {
		http.Error(w, "Not a member of this organization", http.StatusForbidden)
		return
	}

	// Generate unique tag
	tag := generateWorkspaceTag(req.Name)

	// Create workspace config
	config := &provider.WorkspaceConfig{
		WorkspaceID:    tag,
		OrganizationID: req.OrganizationID,
		UserID:         userID,
		Name:           req.Name,
		Tag:            tag,
		CPU:            req.CPU,
		Memory:         req.Memory,
		Storage:        req.Storage,
		Languages:      req.Languages,
		Region:         req.Region,
	}

	// Provision workspace
	info, err := h.provider.CreateWorkspace(r.Context(), config)
	if err != nil {
		log.Printf("Failed to create workspace: %v", err)
		http.Error(w, "Failed to create workspace", http.StatusInternalServerError)
		return
	}

	// Save to database
	workspace := models.Workspace{
		Tag:            tag,
		Name:           req.Name,
		OrganizationID: req.OrganizationID,
		OwnerID:        userID,
		Provider:       "podman",
		Status:         info.Status,
		Config:         "{}",
		Region:         req.Region,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err = db.DB.Exec(
		"INSERT INTO workspaces (tag, name, organization_id, owner_id, provider, status, config, region, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		workspace.Tag, workspace.Name, workspace.OrganizationID, workspace.OwnerID, workspace.Provider, workspace.Status, workspace.Config, workspace.Region, workspace.CreatedAt, workspace.UpdatedAt,
	)
	if err != nil {
		log.Printf("Failed to save workspace: %v", err)
		http.Error(w, "Failed to save workspace", http.StatusInternalServerError)
		return
	}

	response := WorkspaceResponse{
		ID:             workspace.ID,
		Tag:            workspace.Tag,
		Name:           workspace.Name,
		OrganizationID: workspace.OrganizationID,
		OwnerID:        workspace.OwnerID,
		Provider:       workspace.Provider,
		Status:         workspace.Status,
		SSHHost:        info.SSHHost,
		SSHPort:        info.SSHPort,
		HTTPHost:       info.HTTPHost,
		HTTPPort:       info.HTTPPort,
		CreatedAt:      workspace.CreatedAt,
		UpdatedAt:      workspace.UpdatedAt,
	}

	json.NewEncoder(w).Encode(response)
}

// GetWorkspaces retrieves all workspaces for an organization
func (h *WorkspaceHandler) GetWorkspaces(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("organization_id")
	if orgID == "" {
		http.Error(w, "organization_id parameter is required", http.StatusBadRequest)
		return
	}

	var rows *sql.Rows
	var err error

	rows, err = db.DB.Query(
		"SELECT * FROM workspaces WHERE organization_id = ? AND deleted_at IS NULL",
		orgID,
	)

	if err != nil {
		log.Printf("Failed to query workspaces: %v", err)
		http.Error(w, "Failed to retrieve workspaces", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var workspaces []WorkspaceResponse
	for rows.Next() {
		var ws models.Workspace
		if err := rows.Scan(&ws.ID, &ws.Tag, &ws.Name, &ws.OrganizationID, &ws.OwnerID, &ws.Provider, &ws.Status, &ws.Config, &ws.Region, &ws.CreatedAt, &ws.UpdatedAt, &ws.DeletedAt); err != nil {
			log.Printf("Failed to scan workspace: %v", err)
			continue
		}

		workspaces = append(workspaces, WorkspaceResponse{
			ID:             ws.ID,
			Tag:            ws.Tag,
			Name:           ws.Name,
			OrganizationID: ws.OrganizationID,
			OwnerID:        ws.OwnerID,
			Provider:       ws.Provider,
			Status:         ws.Status,
			CreatedAt:      ws.CreatedAt,
			UpdatedAt:      ws.UpdatedAt,
		})
	}

	json.NewEncoder(w).Encode(workspaces)
}

// GetWorkspace retrieves a specific workspace
func (h *WorkspaceHandler) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id parameter is required", http.StatusBadRequest)
		return
	}

	var ws models.Workspace
	err := db.DB.QueryRow(
		"SELECT * FROM workspaces WHERE id = ? AND deleted_at IS NULL",
		id,
	).Scan(&ws.ID, &ws.Tag, &ws.Name, &ws.OrganizationID, &ws.OwnerID, &ws.Provider, &ws.Status, &ws.Config, &ws.Region, &ws.CreatedAt, &ws.UpdatedAt, &ws.DeletedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Failed to query workspace: %v", err)
		http.Error(w, "Failed to retrieve workspace", http.StatusInternalServerError)
		return
	}

	// Get fresh status from provider
	info, err := h.provider.GetWorkspace(r.Context(), ws.Tag)
	if err == nil {
		ws.Status = info.Status
	}

	response := WorkspaceResponse{
		ID:             ws.ID,
		Tag:            ws.Tag,
		Name:           ws.Name,
		OrganizationID: ws.OrganizationID,
		OwnerID:        ws.OwnerID,
		Provider:       ws.Provider,
		Status:         ws.Status,
		CreatedAt:      ws.CreatedAt,
		UpdatedAt:      ws.UpdatedAt,
	}

	json.NewEncoder(w).Encode(response)
}

// UpdateWorkspace updates a workspace
func (h *WorkspaceHandler) UpdateWorkspace(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id parameter is required", http.StatusBadRequest)
		return
	}

	var ws models.Workspace
	err := db.DB.QueryRow(
		"SELECT * FROM workspaces WHERE id = ? AND deleted_at IS NULL",
		id,
	).Scan(&ws.ID, &ws.Tag, &ws.Name, &ws.OrganizationID, &ws.OwnerID, &ws.Provider, &ws.Status, &ws.Config, &ws.Region, &ws.CreatedAt, &ws.UpdatedAt, &ws.DeletedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Failed to query workspace: %v", err)
		http.Error(w, "Failed to retrieve workspace", http.StatusInternalServerError)
		return
	}

	var req WorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update workspace config
	config := &provider.WorkspaceConfig{
		WorkspaceID:    ws.Tag,
		OrganizationID: ws.OrganizationID,
		UserID:         ws.OwnerID,
		Name:           req.Name,
		Tag:            ws.Tag,
		CPU:            req.CPU,
		Memory:         req.Memory,
		Storage:        req.Storage,
		Languages:      req.Languages,
	}

	// Update in provider
	info, err := h.provider.UpdateWorkspace(r.Context(), ws.Tag, config)
	if err != nil {
		log.Printf("Failed to update workspace: %v", err)
		http.Error(w, "Failed to update workspace", http.StatusInternalServerError)
		return
	}

	// Update database
	updatedAt := time.Now()
	_, err = db.DB.Exec(
		"UPDATE workspaces SET name = ?, status = ?, config = ?, region = ?, updated_at = ? WHERE id = ?",
		req.Name, info.Status, "{}", req.Region, updatedAt, id,
	)
	if err != nil {
		log.Printf("Failed to update workspace in database: %v", err)
		http.Error(w, "Failed to update workspace", http.StatusInternalServerError)
		return
	}

	response := WorkspaceResponse{
		ID:             ws.ID,
		Tag:            ws.Tag,
		Name:           req.Name,
		OrganizationID: ws.OrganizationID,
		OwnerID:        ws.OwnerID,
		Provider:       ws.Provider,
		Status:         info.Status,
		CreatedAt:      ws.CreatedAt,
		UpdatedAt:      updatedAt,
	}

	json.NewEncoder(w).Encode(response)
}

// DeleteWorkspace deletes a workspace
func (h *WorkspaceHandler) DeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id parameter is required", http.StatusBadRequest)
		return
	}

	var ws models.Workspace
	err := db.DB.QueryRow(
		"SELECT * FROM workspaces WHERE id = ? AND deleted_at IS NULL",
		id,
	).Scan(&ws.ID, &ws.Tag, &ws.Name, &ws.OrganizationID, &ws.OwnerID, &ws.Provider, &ws.Status, &ws.Config, &ws.Region, &ws.CreatedAt, &ws.UpdatedAt, &ws.DeletedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Failed to query workspace: %v", err)
		http.Error(w, "Failed to retrieve workspace", http.StatusInternalServerError)
		return
	}

	// Delete from provider
	if err := h.provider.DeleteWorkspace(r.Context(), ws.Tag); err != nil {
		log.Printf("Failed to delete workspace from provider: %v", err)
		http.Error(w, "Failed to delete workspace", http.StatusInternalServerError)
		return
	}

	// Soft delete from database
	deletedAt := time.Now()
	_, err = db.DB.Exec(
		"UPDATE workspaces SET deleted_at = ?, status = ? WHERE id = ?",
		deletedAt, "deleted", id,
	)
	if err != nil {
		log.Printf("Failed to soft delete workspace: %v", err)
		http.Error(w, "Failed to delete workspace", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// StartWorkspace starts a workspace
func (h *WorkspaceHandler) StartWorkspace(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id parameter is required", http.StatusBadRequest)
		return
	}

	var ws models.Workspace
	err := db.DB.QueryRow(
		"SELECT * FROM workspaces WHERE id = ? AND deleted_at IS NULL",
		id,
	).Scan(&ws.ID, &ws.Tag, &ws.Name, &ws.OrganizationID, &ws.OwnerID, &ws.Provider, &ws.Status, &ws.Config, &ws.Region, &ws.CreatedAt, &ws.UpdatedAt, &ws.DeletedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Failed to query workspace: %v", err)
		http.Error(w, "Failed to retrieve workspace", http.StatusInternalServerError)
		return
	}

	// Start workspace in provider
	info, err := h.provider.StartWorkspace(r.Context(), ws.Tag)
	if err != nil {
		log.Printf("Failed to start workspace: %v", err)
		http.Error(w, "Failed to start workspace", http.StatusInternalServerError)
		return
	}

	// Update database
	updatedAt := time.Now()
	_, err = db.DB.Exec(
		"UPDATE workspaces SET status = ?, updated_at = ? WHERE id = ?",
		info.Status, updatedAt, id,
	)
	if err != nil {
		log.Printf("Failed to update workspace status: %v", err)
	}

	response := WorkspaceResponse{
		ID:             ws.ID,
		Tag:            ws.Tag,
		Name:           ws.Name,
		OrganizationID: ws.OrganizationID,
		OwnerID:        ws.OwnerID,
		Provider:       ws.Provider,
		Status:         info.Status,
		CreatedAt:      ws.CreatedAt,
		UpdatedAt:      updatedAt,
	}

	json.NewEncoder(w).Encode(response)
}

// StopWorkspace stops a workspace
func (h *WorkspaceHandler) StopWorkspace(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id parameter is required", http.StatusBadRequest)
		return
	}

	var ws models.Workspace
	err := db.DB.QueryRow(
		"SELECT * FROM workspaces WHERE id = ? AND deleted_at IS NULL",
		id,
	).Scan(&ws.ID, &ws.Tag, &ws.Name, &ws.OrganizationID, &ws.OwnerID, &ws.Provider, &ws.Status, &ws.Config, &ws.Region, &ws.CreatedAt, &ws.UpdatedAt, &ws.DeletedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Failed to query workspace: %v", err)
		http.Error(w, "Failed to retrieve workspace", http.StatusInternalServerError)
		return
	}

	// Stop workspace in provider
	info, err := h.provider.StopWorkspace(r.Context(), ws.Tag)
	if err != nil {
		log.Printf("Failed to stop workspace: %v", err)
		http.Error(w, "Failed to stop workspace", http.StatusInternalServerError)
		return
	}

	// Update database
	updatedAt := time.Now()
	_, err = db.DB.Exec(
		"UPDATE workspaces SET status = ?, updated_at = ? WHERE id = ?",
		info.Status, updatedAt, id,
	)
	if err != nil {
		log.Printf("Failed to update workspace status: %v", err)
	}

	response := WorkspaceResponse{
		ID:             ws.ID,
		Tag:            ws.Tag,
		Name:           ws.Name,
		OrganizationID: ws.OrganizationID,
		OwnerID:        ws.OwnerID,
		Provider:       ws.Provider,
		Status:         info.Status,
		CreatedAt:      ws.CreatedAt,
		UpdatedAt:      updatedAt,
	}

	json.NewEncoder(w).Encode(response)
}

// RestartWorkspace restarts a workspace
func (h *WorkspaceHandler) RestartWorkspace(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id parameter is required", http.StatusBadRequest)
		return
	}

	var ws models.Workspace
	err := db.DB.QueryRow(
		"SELECT * FROM workspaces WHERE id = ? AND deleted_at IS NULL",
		id,
	).Scan(&ws.ID, &ws.Tag, &ws.Name, &ws.OrganizationID, &ws.OwnerID, &ws.Provider, &ws.Status, &ws.Config, &ws.Region, &ws.CreatedAt, &ws.UpdatedAt, &ws.DeletedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Failed to query workspace: %v", err)
		http.Error(w, "Failed to retrieve workspace", http.StatusInternalServerError)
		return
	}

	// Restart workspace in provider
	info, err := h.provider.RestartWorkspace(r.Context(), ws.Tag)
	if err != nil {
		log.Printf("Failed to restart workspace: %v", err)
		http.Error(w, "Failed to restart workspace", http.StatusInternalServerError)
		return
	}

	// Update database
	updatedAt := time.Now()
	_, err = db.DB.Exec(
		"UPDATE workspaces SET status = ?, updated_at = ? WHERE id = ?",
		info.Status, updatedAt, id,
	)
	if err != nil {
		log.Printf("Failed to update workspace status: %v", err)
	}

	response := WorkspaceResponse{
		ID:             ws.ID,
		Tag:            ws.Tag,
		Name:           ws.Name,
		OrganizationID: ws.OrganizationID,
		OwnerID:        ws.OwnerID,
		Provider:       ws.Provider,
		Status:         info.Status,
		CreatedAt:      ws.CreatedAt,
		UpdatedAt:      updatedAt,
	}

	json.NewEncoder(w).Encode(response)
}

// generateWorkspaceTag generates a unique tag for a workspace
func generateWorkspaceTag(name string) string {
	// Convert name to lowercase and replace spaces with hyphens
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	// Remove special characters
	var cleaned strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			cleaned.WriteRune(r)
		}
	}

	// Add UUID to ensure uniqueness
	uuid := uuid.New().String()[:8]
	return fmt.Sprintf("%s-%s", cleaned.String(), uuid)
}

// isMemberOfOrg checks if a user is a member of an organization
func isMemberOfOrg(userID, orgID int) bool {
	var count int
	err := db.DB.QueryRow(
		"SELECT COUNT(*) FROM members WHERE user_id = ? AND organization_id = ? AND accepted = 1",
		userID, orgID,
	).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}
