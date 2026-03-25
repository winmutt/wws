package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"wws/api/internal/terminal"
)

// TerminalHandler handles shared terminal HTTP requests
type TerminalHandler struct{}

// CreateTerminalSessionRequest represents a request to create a terminal session
type CreateTerminalSessionRequest struct {
	WorkspaceID int    `json:"workspace_id"`
	SessionType string `json:"session_type"` // "tmux" or "direct"
}

// CreateTerminalSessionHandler creates a new shared terminal session
func (h *TerminalHandler) CreateTerminalSession(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)
	if userID == 0 {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	var req CreateTerminalSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid request body: %v", err))
		return
	}

	if req.WorkspaceID == 0 {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Workspace ID is required"))
		return
	}

	if req.SessionType == "" {
		req.SessionType = "tmux"
	}

	session, err := terminal.CreateTerminalSession(r.Context(), req.WorkspaceID, req.SessionType)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to create terminal session: %v", err))
		return
	}

	WriteJSON(w, http.StatusCreated, session)
}

// GetTerminalSessionsHandler retrieves all terminal sessions for a user
func (h *TerminalHandler) GetTerminalSessions(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)
	if userID == 0 {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	participants, err := terminal.GetSessionParticipantsByUser(r.Context(), userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to get terminal sessions: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, participants)
}

// GetSessionParticipantsHandler retrieves all participants for a terminal session
func (h *TerminalHandler) GetSessionParticipants(w http.ResponseWriter, r *http.Request) {
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

	participants, err := terminal.GetSessionParticipants(r.Context(), sessionID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to get participants: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, participants)
}

// JoinTerminalSessionHandler upgrades the connection to WebSocket and joins a terminal session
func (h *TerminalHandler) JoinTerminalSession(w http.ResponseWriter, r *http.Request) {
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

	userID := getUserIDFromRequest(r)
	if userID == 0 {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	username := getUsernameFromRequest(r)
	if username == "" {
		username = fmt.Sprintf("User%d", userID)
	}

	permission := vars.Get("permission")
	if permission == "" {
		permission = "viewer"
	}

	terminal.HandleWebSocket(w, r, sessionID, userID, username, permission)
}

// BroadcastInputHandler broadcasts terminal input to all participants
func (h *TerminalHandler) BroadcastInput(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID int    `json:"session_id"`
		Data      string `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid request body: %v", err))
		return
	}

	userID := getUserIDFromRequest(r)
	if userID == 0 {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	terminal.BroadcastInput(req.SessionID, userID, req.Data)
	WriteJSON(w, http.StatusOK, map[string]string{"message": "Input broadcasted"})
}

// BroadcastOutputHandler broadcasts terminal output to all participants
func (h *TerminalHandler) BroadcastOutput(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID int    `json:"session_id"`
		Data      string `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid request body: %v", err))
		return
	}

	terminal.BroadcastOutput(req.SessionID, req.Data)
	WriteJSON(w, http.StatusOK, map[string]string{"message": "Output broadcasted"})
}

// BroadcastCursorHandler broadcasts cursor position updates
func (h *TerminalHandler) BroadcastCursor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID int `json:"session_id"`
		X         int `json:"x"`
		Y         int `json:"y"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid request body: %v", err))
		return
	}

	userID := getUserIDFromRequest(r)
	if userID == 0 {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("Unauthorized"))
		return
	}

	terminal.BroadcastCursor(req.SessionID, userID, req.X, req.Y)
	WriteJSON(w, http.StatusOK, map[string]string{"message": "Cursor updated"})
}
