package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"wws/api/internal/teamtemplates"
)

// TeamTemplateHandler handles team template access HTTP requests
type TeamTemplateHandler struct{}

// GrantTemplateAccessRequest represents a request to grant template access
type GrantTemplateAccessRequest struct {
	TeamID     int    `json:"team_id"`
	TemplateID int    `json:"template_id"`
	Permission string `json:"permission"` // "view" or "use"
}

// GrantTemplateAccessHandler grants a team access to a workspace template
func (h *TeamTemplateHandler) GrantTemplateAccess(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)
	if userID == 0 {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	var req GrantTemplateAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid request body: %v", err))
		return
	}

	if req.TeamID == 0 || req.TemplateID == 0 {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Team ID and Template ID are required"))
		return
	}

	validPermissions := map[string]bool{"view": true, "use": true}
	if !validPermissions[req.Permission] {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid permission. Use 'view' or 'use'"))
		return
	}

	access, err := teamtemplates.GrantTemplateAccess(r.Context(), req.TeamID, req.TemplateID, userID, req.Permission)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to grant template access: %v", err))
		return
	}

	WriteJSON(w, http.StatusCreated, access)
}

// GetTeamTemplatesHandler retrieves all templates accessible by a team
func (h *TeamTemplateHandler) GetTeamTemplates(w http.ResponseWriter, r *http.Request) {
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

	accesses, err := teamtemplates.GetTeamTemplateAccessByTeam(r.Context(), teamID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to get team templates: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, accesses)
}

// GetTemplateTeamsHandler retrieves all teams that have access to a template
func (h *TeamTemplateHandler) GetTemplateTeams(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	templateIDStr := vars.Get("template_id")
	if templateIDStr == "" {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Template ID required"))
		return
	}

	templateID, err := strconv.Atoi(templateIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid template ID"))
		return
	}

	accesses, err := teamtemplates.GetTeamTemplateAccessByTemplate(r.Context(), templateID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to get template teams: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, accesses)
}

// RevokeTemplateAccessHandler revokes a team's access to a template
func (h *TeamTemplateHandler) RevokeTemplateAccess(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	accessIDStr := vars.Get("access_id")
	if accessIDStr == "" {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Access ID required"))
		return
	}

	accessID, err := strconv.Atoi(accessIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid access ID"))
		return
	}

	err = teamtemplates.RevokeTemplateAccess(r.Context(), accessID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to revoke access: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Template access revoked successfully"})
}

// CheckTemplatePermissionHandler checks if a team has permission to use a template
func (h *TeamTemplateHandler) CheckTemplatePermission(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeamID     int    `json:"team_id"`
		TemplateID int    `json:"template_id"`
		Permission string `json:"permission"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid request body: %v", err))
		return
	}

	if req.TeamID == 0 || req.TemplateID == 0 {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Team ID and Template ID are required"))
		return
	}

	hasPermission, err := teamtemplates.HasTeamTemplatePermission(r.Context(), req.TeamID, req.TemplateID, req.Permission)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to check permission: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"has_permission": hasPermission,
		"team_id":        req.TeamID,
		"template_id":    req.TemplateID,
	})
}

// GetUsableTemplatesHandler retrieves all templates a team can use to create workspaces
func (h *TeamTemplateHandler) GetUsableTemplates(w http.ResponseWriter, r *http.Request) {
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

	templates, err := teamtemplates.GetTeamUsableTemplates(r.Context(), teamID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to get usable templates: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, templates)
}
