package handlers

import (
	"fmt"
	"net/http"
)

type APIDocumentation struct {
	Title       string         `json:"title"`
	Version     string         `json:"version"`
	Description string         `json:"description"`
	BaseURL     string         `json:"base_url"`
	Auth        AuthInfo       `json:"authentication"`
	Endpoints   []EndpointInfo `json:"endpoints"`
}

type AuthInfo struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Methods     []string `json:"methods"`
}

type EndpointInfo struct {
	Path         string `json:"path"`
	Method       string `json:"method"`
	Description  string `json:"description"`
	AuthRequired bool   `json:"auth_required"`
	RequestBody  string `json:"request_body,omitempty"`
	ResponseBody string `json:"response_body,omitempty"`
}

func APIDocsHandler(w http.ResponseWriter, r *http.Request) error {
	host := r.Host
	scheme := "http"
	if forwardedProto := r.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = forwardedProto
	} else if r.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s/api/v1", scheme, host)

	docs := APIDocumentation{
		Title:       "Winmutt Work Spaces API",
		Version:     "1.0.0",
		Description: "REST API for Winmutt Work Spaces - A remote workspace provisioning system for engineering organizations",
		BaseURL:     baseURL,
		Auth: AuthInfo{
			Type:        "OAuth2 / Session-based",
			Description: "Authenticate via GitHub OAuth2 to receive a session token. Include the session token in the Cookie header for subsequent requests.",
			Methods:     []string{"GitHub OAuth2", "Session Cookie", "API Keys"},
		},
		Endpoints: []EndpointInfo{
			{Path: "/api/v1/health", Method: "GET", Description: "Health check endpoint", AuthRequired: false, ResponseBody: `{"status": "ok"}`},
			{Path: "/api/v1/auth/github", Method: "GET", Description: "Initiate GitHub OAuth2", AuthRequired: false},
			{Path: "/api/v1/auth/github/callback", Method: "GET", Description: "GitHub OAuth2 callback", AuthRequired: false},
			{Path: "/api/v1/sessions", Method: "GET", Description: "Get current session", AuthRequired: true},
			{Path: "/api/v1/sessions/refresh", Method: "POST", Description: "Refresh session", AuthRequired: true},
			{Path: "/api/v1/sessions/revoke", Method: "DELETE", Description: "Revoke current session", AuthRequired: true},
			{Path: "/api/v1/sessions/revoke-all", Method: "DELETE", Description: "Revoke all sessions", AuthRequired: true},
			{Path: "/api/v1/sessions/user", Method: "GET", Description: "List user sessions", AuthRequired: true},
			{Path: "/api/v1/users", Method: "GET", Description: "List all users", AuthRequired: true},
			{Path: "/api/v1/users/{id}", Method: "GET", Description: "Get user by ID", AuthRequired: true},
			{Path: "/api/v1/organizations", Method: "GET", Description: "List organizations", AuthRequired: true},
			{Path: "/api/v1/organizations", Method: "POST", Description: "Create organization", AuthRequired: true, RequestBody: `{"name": "..."}`},
			{Path: "/api/v1/organizations/members", Method: "GET", Description: "List members", AuthRequired: true},
			{Path: "/api/v1/organizations/members", Method: "DELETE", Description: "Remove member", AuthRequired: true},
			{Path: "/api/v1/organizations/roles", Method: "POST", Description: "Assign role", AuthRequired: true, RequestBody: `{"user_id": 1, "role": "admin|member|viewer"}`},
			{Path: "/api/v1/organizations/invitations", Method: "POST", Description: "Create invitation", AuthRequired: true, RequestBody: `{"email": "..."}`},
			{Path: "/api/v1/organizations/invitations/accept", Method: "POST", Description: "Accept invitation", AuthRequired: true},
			{Path: "/api/v1/workspaces", Method: "GET", Description: "List workspaces", AuthRequired: true},
			{Path: "/api/v1/workspaces", Method: "POST", Description: "Create workspace", AuthRequired: true, RequestBody: `{"name": "...", "tag": "...", "provider": "podman"}`},
			{Path: "/api/v1/workspaces/{id}", Method: "GET", Description: "Get workspace", AuthRequired: true},
			{Path: "/api/v1/workspaces/{id}", Method: "PUT", Description: "Update workspace", AuthRequired: true},
			{Path: "/api/v1/workspaces/{id}", Method: "DELETE", Description: "Delete workspace", AuthRequired: true},
			{Path: "/api/v1/workspaces/{id}/start", Method: "POST", Description: "Start workspace", AuthRequired: true},
			{Path: "/api/v1/workspaces/{id}/stop", Method: "POST", Description: "Stop workspace", AuthRequired: true},
			{Path: "/api/v1/workspaces/{id}/restart", Method: "POST", Description: "Restart workspace", AuthRequired: true},
			{Path: "/api/v1/workspaces/export", Method: "POST", Description: "Export workspace", AuthRequired: true},
			{Path: "/api/v1/workspaces/import", Method: "POST", Description: "Import workspace", AuthRequired: true},
			{Path: "/api/v1/workspaces/export/status", Method: "GET", Description: "Get export status", AuthRequired: true},
			{Path: "/api/v1/workspaces/import/status", Method: "GET", Description: "Get import status", AuthRequired: true},
			{Path: "/api/v1/workspaces/exports", Method: "GET", Description: "List exports", AuthRequired: true},
			{Path: "/api/v1/workspaces/imports", Method: "GET", Description: "List imports", AuthRequired: true},
			{Path: "/api/v1/workspaces/export/download", Method: "GET", Description: "Download export", AuthRequired: true},
			{Path: "/api/v1/workspaces/export", Method: "DELETE", Description: "Delete export", AuthRequired: true},
			{Path: "/api/v1/audit", Method: "GET", Description: "Get audit logs (admin)", AuthRequired: true},
			{Path: "/api/v1/audit/{id}", Method: "GET", Description: "Get audit log by ID", AuthRequired: true},
			{Path: "/api/v1/audit/summary", Method: "GET", Description: "Get audit summary", AuthRequired: true},
			{Path: "/api/v1/quotas", Method: "GET", Description: "Get quotas", AuthRequired: true},
			{Path: "/api/v1/quotas", Method: "PUT", Description: "Update quotas", AuthRequired: true},
			{Path: "/api/v1/quotas/usage", Method: "GET", Description: "Get usage", AuthRequired: true},
			{Path: "/api/v1/quotas/usage", Method: "POST", Description: "Update usage", AuthRequired: true},
			{Path: "/api/v1/quotas/check", Method: "POST", Description: "Check quota", AuthRequired: true},
			{Path: "/api/v1/api-keys", Method: "POST", Description: "Create API key", AuthRequired: true},
			{Path: "/api/v1/api-keys", Method: "GET", Description: "List API keys", AuthRequired: true},
			{Path: "/api/v1/api-keys/{id}", Method: "DELETE", Description: "Delete API key", AuthRequired: true},
			{Path: "/api/v1/compliance/report", Method: "POST", Description: "Generate compliance report", AuthRequired: true},
			{Path: "/api/v1/compliance/report/{id}", Method: "GET", Description: "Get compliance report", AuthRequired: true},
			{Path: "/api/v1/compliance/export", Method: "POST", Description: "Export compliance report", AuthRequired: true},
			{Path: "/api/v1/compliance/score", Method: "POST", Description: "Get compliance score", AuthRequired: true},
			{Path: "/api/v1/compliance/reports", Method: "GET", Description: "List compliance reports", AuthRequired: true},
			{Path: "/api/v1/compliance/status", Method: "GET", Description: "Check compliance status", AuthRequired: true},
			{Path: "/api/v1/analytics/usage", Method: "POST", Description: "Record usage", AuthRequired: true},
			{Path: "/api/v1/analytics/usage/workspace", Method: "GET", Description: "Get workspace usage", AuthRequired: true},
			{Path: "/api/v1/analytics/usage/organization", Method: "GET", Description: "Get org usage", AuthRequired: true},
			{Path: "/api/v1/analytics/stats/workspace", Method: "GET", Description: "Get workspace stats", AuthRequired: true},
			{Path: "/api/v1/analytics/summary", Method: "GET", Description: "Get analytics summary", AuthRequired: true},
			{Path: "/api/v1/analytics/alerts", Method: "GET", Description: "Get alerts", AuthRequired: true},
			{Path: "/api/v1/analytics/alerts/resolve", Method: "POST", Description: "Resolve alert", AuthRequired: true},
			{Path: "/api/v1/tmux", Method: "POST", Description: "Create tmux session", AuthRequired: true},
			{Path: "/api/v1/tmux", Method: "GET", Description: "List tmux sessions", AuthRequired: true},
			{Path: "/api/v1/tmux/share", Method: "POST", Description: "Share tmux session", AuthRequired: true},
			{Path: "/api/v1/tmux/shares", Method: "GET", Description: "Get tmux shares", AuthRequired: true},
			{Path: "/api/v1/tmux/shares/revoke", Method: "DELETE", Description: "Revoke tmux share", AuthRequired: true},
			{Path: "/api/v1/tmux/access/check", Method: "GET", Description: "Check tmux access", AuthRequired: true},
			{Path: "/api/v1/teams", Method: "POST", Description: "Create team", AuthRequired: true},
			{Path: "/api/v1/teams", Method: "GET", Description: "List teams", AuthRequired: true},
			{Path: "/api/v1/teams/get", Method: "GET", Description: "Get team", AuthRequired: true},
			{Path: "/api/v1/teams/members", Method: "GET", Description: "Get team members", AuthRequired: true},
			{Path: "/api/v1/teams/my-teams", Method: "GET", Description: "Get user teams", AuthRequired: true},
			{Path: "/api/v1/teams/members/add", Method: "POST", Description: "Add team member", AuthRequired: true},
			{Path: "/api/v1/teams/members/remove", Method: "DELETE", Description: "Remove team member", AuthRequired: true},
			{Path: "/api/v1/teams/roles", Method: "POST", Description: "Create team role", AuthRequired: true},
			{Path: "/api/v1/teams/roles", Method: "GET", Description: "List team roles", AuthRequired: true},
			{Path: "/api/v1/teams/permissions/check", Method: "POST", Description: "Check permissions", AuthRequired: true},
			{Path: "/api/v1/teams/workspace-access/grant", Method: "POST", Description: "Grant workspace access", AuthRequired: true},
			{Path: "/api/v1/teams/workspace-access", Method: "GET", Description: "Get workspace access", AuthRequired: true},
			{Path: "/api/v1/teams/workspace-access/check", Method: "GET", Description: "Check workspace access", AuthRequired: true},
			{Path: "/api/v1/terminal", Method: "POST", Description: "Create terminal session", AuthRequired: true},
			{Path: "/api/v1/terminal", Method: "GET", Description: "Get terminal sessions", AuthRequired: true},
			{Path: "/api/v1/terminal/participants", Method: "GET", Description: "Get participants", AuthRequired: true},
			{Path: "/api/v1/terminal/join", Method: "GET", Description: "Join terminal session", AuthRequired: true},
			{Path: "/api/v1/terminal/input", Method: "POST", Description: "Broadcast input", AuthRequired: true},
			{Path: "/api/v1/terminal/output", Method: "POST", Description: "Broadcast output", AuthRequired: true},
			{Path: "/api/v1/terminal/cursor", Method: "POST", Description: "Broadcast cursor", AuthRequired: true},
			{Path: "/api/v1/teams/templates/grant", Method: "POST", Description: "Grant template access", AuthRequired: true},
			{Path: "/api/v1/teams/templates", Method: "GET", Description: "Get team templates", AuthRequired: true},
			{Path: "/api/v1/teams/templates/teams", Method: "GET", Description: "Get template teams", AuthRequired: true},
			{Path: "/api/v1/teams/templates/revoke", Method: "DELETE", Description: "Revoke template access", AuthRequired: true},
			{Path: "/api/v1/teams/templates/permission", Method: "POST", Description: "Check template permission", AuthRequired: true},
			{Path: "/api/v1/teams/templates/usable", Method: "GET", Description: "Get usable templates", AuthRequired: true},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	return WriteJSON(w, http.StatusOK, docs)
}
