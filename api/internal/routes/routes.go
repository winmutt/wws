package routes

import (
	"github.com/gorilla/mux"

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
}
