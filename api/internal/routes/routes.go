package routes

import (
	"github.com/gorilla/mux"

	"wws/api/internal/analytics"
	"wws/api/internal/handlers"
)

func SetupRoutes(r *mux.Router) {
	api := r.PathPrefix("/api/v1").Subrouter()

	// Health check
	api.HandleFunc("/health", handlers.Adapter(handlers.HealthHandler)).Methods("GET")

	// Auth routes
	auth := api.PathPrefix("/auth").Subrouter()
	auth.HandleFunc("/github", handlers.Adapter(handlers.GitHubAuthHandler)).Methods("GET")
	auth.HandleFunc("/github/callback", handlers.Adapter(handlers.OAuthCallbackHandler)).Methods("GET")

	// Session routes
	sessions := api.PathPrefix("/sessions").Subrouter()
	sessions.HandleFunc("", handlers.Adapter(handlers.GetSessionHandler)).Methods("GET")
	sessions.HandleFunc("/refresh", handlers.Adapter(handlers.RefreshSessionHandler)).Methods("POST")
	sessions.HandleFunc("/revoke", handlers.Adapter(handlers.RevokeSessionHandler)).Methods("DELETE")
	sessions.HandleFunc("/revoke-all", handlers.Adapter(handlers.RevokeAllSessionsHandler)).Methods("DELETE")
	sessions.HandleFunc("/user", handlers.Adapter(handlers.ListUserSessionsHandler)).Methods("GET")

	// User routes
	users := api.PathPrefix("/users").Subrouter()
	users.HandleFunc("", handlers.Adapter(handlers.ListUsersHandler)).Methods("GET")
	users.HandleFunc("/{id}", handlers.Adapter(handlers.GetUserHandler)).Methods("GET")

	// Organization routes
	orgs := api.PathPrefix("/organizations").Subrouter()
	orgs.HandleFunc("", handlers.Adapter(handlers.ListOrganizationsHandler)).Methods("GET")
	orgs.HandleFunc("", handlers.Adapter(handlers.CreateOrganizationHandler)).Methods("POST")

	// Organization member routes
	orgs.HandleFunc("/members", handlers.Adapter(handlers.ListMembersHandler)).Methods("GET")
	orgs.HandleFunc("/members", handlers.Adapter(handlers.GetMemberHandler)).Methods("GET")
	orgs.HandleFunc("/members", handlers.Adapter(handlers.RemoveMemberHandler)).Methods("DELETE")

	// Organization role routes
	orgs.HandleFunc("/roles", handlers.Adapter(handlers.AssignRoleHandler)).Methods("POST")

	// Invitation routes
	orgs.HandleFunc("/invitations", handlers.Adapter(handlers.CreateInvitationHandler)).Methods("POST")
	orgs.HandleFunc("/invitations/accept", handlers.Adapter(handlers.AcceptInvitationHandler)).Methods("POST")

	// Workspace routes
	workspaces := api.PathPrefix("/workspaces").Subrouter()
	workspaces.HandleFunc("", handlers.Adapter(handlers.ListWorkspacesHandler)).Methods("GET")
	workspaces.HandleFunc("", handlers.Adapter(handlers.CreateWorkspaceHandler)).Methods("POST")
	workspaces.HandleFunc("/{id}", handlers.Adapter(handlers.GetWorkspaceHandler)).Methods("GET")
	workspaces.HandleFunc("/{id}", handlers.Adapter(handlers.UpdateWorkspaceHandler)).Methods("PUT")
	workspaces.HandleFunc("/{id}", handlers.Adapter(handlers.DeleteWorkspaceHandler)).Methods("DELETE")
	workspaces.HandleFunc("/{id}/start", handlers.Adapter(handlers.StartWorkspaceHandler)).Methods("POST")
	workspaces.HandleFunc("/{id}/stop", handlers.Adapter(handlers.StopWorkspaceHandler)).Methods("POST")
	workspaces.HandleFunc("/{id}/restart", handlers.Adapter(handlers.RestartWorkspaceHandler)).Methods("POST")

	// Audit log routes (admin only)
	audit := api.PathPrefix("/audit").Subrouter()
	audit.HandleFunc("", handlers.GetAuditLogsHandler).Methods("GET")
	audit.HandleFunc("/{id}", handlers.GetAuditLogByIDHandler).Methods("GET")
	audit.HandleFunc("/summary", handlers.GetAuditLogSummaryHandler).Methods("GET")

	// Resource quota routes
	quotas := api.PathPrefix("/quotas").Subrouter()
	quotas.HandleFunc("", handlers.QuotaGetHandler).Methods("GET")
	quotas.HandleFunc("", handlers.QuotaUpdateHandler).Methods("PUT")
	quotas.HandleFunc("/usage", handlers.QuotaUsageHandler).Methods("GET")
	quotas.HandleFunc("/usage", handlers.QuotaUpdateUsageHandler).Methods("POST")
	quotas.HandleFunc("/check", handlers.QuotaCheckHandler).Methods("POST")

	// API key routes
	apiKeys := api.PathPrefix("/api-keys").Subrouter()
	apiKeys.HandleFunc("", handlers.CreateAPIKeyHandler).Methods("POST")
	apiKeys.HandleFunc("", handlers.ListAPIKeysHandler).Methods("GET")
	apiKeys.HandleFunc("/{id}", handlers.DeleteAPIKeyHandler).Methods("DELETE")

	// Compliance routes
	compliance := api.PathPrefix("/compliance").Subrouter()
	compliance.HandleFunc("/report", handlers.ComplianceGenerateReportHandler).Methods("POST")
	compliance.HandleFunc("/report/{id}", handlers.ComplianceGetReportHandler).Methods("GET")
	compliance.HandleFunc("/export", handlers.ComplianceExportReportHandler).Methods("POST")
	compliance.HandleFunc("/score", handlers.ComplianceGetScoreHandler).Methods("POST")
	compliance.HandleFunc("/reports", handlers.ComplianceListReportsHandler).Methods("GET")
	compliance.HandleFunc("/status", handlers.ComplianceCheckComplianceStatusHandler).Methods("GET")

	// Analytics routes
	analyticsRoutes := api.PathPrefix("/analytics").Subrouter()
	analyticsRoutes.HandleFunc("/usage", handlers.Adapter(analytics.Adapter(analytics.RecordUsageHandler))).Methods("POST")
	analyticsRoutes.HandleFunc("/usage/workspace", handlers.Adapter(analytics.Adapter(analytics.GetWorkspaceUsageHandler))).Methods("GET")
	analyticsRoutes.HandleFunc("/usage/organization", handlers.Adapter(analytics.Adapter(analytics.GetOrganizationUsageHandler))).Methods("GET")
	analyticsRoutes.HandleFunc("/stats/workspace", handlers.Adapter(analytics.Adapter(analytics.GetWorkspaceStatsHandler))).Methods("GET")
	analyticsRoutes.HandleFunc("/summary", handlers.Adapter(analytics.Adapter(analytics.GetAnalyticsSummaryHandler))).Methods("GET")

	// Usage alerts routes
	analyticsRoutes.HandleFunc("/alerts", handlers.Adapter(analytics.Adapter(analytics.GetActiveAlertsHandler))).Methods("GET")
	analyticsRoutes.HandleFunc("/alerts/resolve", handlers.Adapter(analytics.Adapter(analytics.ResolveAlertHandler))).Methods("POST")
}
