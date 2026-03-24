package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"wws/api/internal/tmux"
)

// TmuxHandler handles tmux session HTTP requests
type TmuxHandler struct{}

// CreateTmuxSessionRequest represents a request to create a tmux session
type CreateTmuxSessionRequest struct {
	WorkspaceID   int    `json:"workspace_id"`
	SessionName   string `json:"session_name,omitempty"`
	DurationHours int    `json:"duration_hours,omitempty"`
}

// CreateTmuxSessionHandler creates a new tmux session
func (h *TmuxHandler) CreateTmuxSession(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)
	if userID == 0 {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	var req CreateTmuxSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid request body: %v", err))
		return
	}

	if req.WorkspaceID == 0 {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Workspace ID is required"))
		return
	}

	session, err := tmux.CreateTmuxSession(r.Context(), req.WorkspaceID, userID, req.SessionName, req.DurationHours)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to create tmux session: %v", err))
		return
	}

	WriteJSON(w, http.StatusCreated, session)
}

// GetTmuxSessionsHandler retrieves all tmux sessions for a user
func (h *TmuxHandler) GetTmuxSessions(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)
	if userID == 0 {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	sessions, err := tmux.GetUserTmuxSessions(r.Context(), userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to get tmux sessions: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, sessions)
}

// ShareTmuxSessionRequest represents a request to share a tmux session
type ShareTmuxSessionRequest struct {
	SessionID     int    `json:"session_id"`
	UserID        int    `json:"user_id"`
	Permission    string `json:"permission"` // "read" or "write"
	DurationHours int    `json:"duration_hours,omitempty"`
}

// ShareTmuxSessionHandler shares a tmux session with another user
func (h *TmuxHandler) ShareTmuxSession(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)
	if userID == 0 {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	var req ShareTmuxSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid request body: %v", err))
		return
	}

	if req.SessionID == 0 || req.UserID == 0 {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Session ID and User ID are required"))
		return
	}

	validPermissions := map[string]bool{"read": true, "write": true}
	if !validPermissions[req.Permission] {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid permission"))
		return
	}

	share, err := tmux.ShareTmuxSession(r.Context(), req.SessionID, req.UserID, userID, req.Permission, req.DurationHours)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to share tmux session: %v", err))
		return
	}

	WriteJSON(w, http.StatusCreated, share)
}

// GetTmuxSessionSharesHandler retrieves all shares for a session
func (h *TmuxHandler) GetTmuxSessionShares(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	sessionIDStr := vars.Get("session_id")
	if sessionIDStr == "" {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Session ID required"))
		return
	}

	sessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid session ID"))
		return
	}

	shares, err := tmux.GetTmuxSessionShares(r.Context(), sessionID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to get shares: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, shares)
}

// RevokeTmuxShareHandler revokes access to a shared session
func (h *TmuxHandler) RevokeTmuxShare(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	shareIDStr := vars.Get("share_id")
	if shareIDStr == "" {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Share ID required"))
		return
	}

	shareID, err := strconv.Atoi(shareIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid share ID"))
		return
	}

	err = tmux.RevokeTmuxShare(r.Context(), shareID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to revoke share: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Share revoked successfully"})
}

// DeleteTmuxSessionHandler deletes a tmux session
func (h *TmuxHandler) DeleteTmuxSession(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	sessionIDStr := vars.Get("session_id")
	if sessionIDStr == "" {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Session ID required"))
		return
	}

	sessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid session ID"))
		return
	}

	err = tmux.DeleteTmuxSession(r.Context(), sessionID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to delete session: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Session deleted successfully"})
}

// CheckTmuxAccessHandler checks if a user has access to a tmux session
func (h *TmuxHandler) CheckTmuxAccess(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	sessionIDStr := vars.Get("session_id")
	userIDStr := vars.Get("user_id")

	if sessionIDStr == "" || userIDStr == "" {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Session ID and User ID required"))
		return
	}

	sessionID, _ := strconv.Atoi(sessionIDStr)
	userID, _ := strconv.Atoi(userIDStr)

	hasAccess, permission, err := tmux.HasTmuxAccess(r.Context(), sessionID, userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to check access: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"has_access": hasAccess,
		"permission": permission,
	})
}
