package routes

import (
	"github.com/gorilla/mux"
	"net/http"
	"wws/api/internal/handlers"
)

func SetupRoutes(r *mux.Router) {
	api := r.PathPrefix("/api/v1").Subrouter()

	// Health check
	api.HandleFunc("/health", httpHandler(handlers.HealthHandler)).Methods("GET")

	// Auth routes
	auth := api.PathPrefix("/auth").Subrouter()
	auth.HandleFunc("/github", httpHandler(handlers.GitHubAuthHandler)).Methods("GET")
	auth.HandleFunc("/github/callback", httpHandler(handlers.GitHubCallbackHandler)).Methods("GET")

	// User routes
	users := api.PathPrefix("/users").Subrouter()
	users.HandleFunc("", httpHandler(handlers.ListUsersHandler)).Methods("GET")
	users.HandleFunc("/{id}", httpHandler(handlers.GetUserHandler)).Methods("GET")

	// Organization routes
	orgs := api.PathPrefix("/organizations").Subrouter()
	orgs.HandleFunc("", httpHandler(handlers.ListOrganizationsHandler)).Methods("GET")
	orgs.HandleFunc("", httpHandler(handlers.CreateOrganizationHandler)).Methods("POST")
	orgs.HandleFunc("/{id}", httpHandler(handlers.GetOrganizationHandler)).Methods("GET")
	orgs.HandleFunc("/{id}", httpHandler(handlers.UpdateOrganizationHandler)).Methods("PUT")
	orgs.HandleFunc("/{id}", httpHandler(handlers.DeleteOrganizationHandler)).Methods("DELETE")
	orgs.HandleFunc("/{id}/invitations", httpHandler(handlers.CreateInvitationHandler)).Methods("POST")
	orgs.HandleFunc("/invitations/{id}/accept", httpHandler(handlers.AcceptInvitationHandler)).Methods("POST")

	// Workspace routes
	workspaces := api.PathPrefix("/workspaces").Subrouter()
	workspaces.HandleFunc("", httpHandler(handlers.ListWorkspacesHandler)).Methods("GET")
	workspaces.HandleFunc("", httpHandler(handlers.CreateWorkspaceHandler)).Methods("POST")
	workspaces.HandleFunc("/{id}", httpHandler(handlers.GetWorkspaceHandler)).Methods("GET")
	workspaces.HandleFunc("/{id}", httpHandler(handlers.UpdateWorkspaceHandler)).Methods("PUT")
	workspaces.HandleFunc("/{id}", httpHandler(handlers.DeleteWorkspaceHandler)).Methods("DELETE")
	workspaces.HandleFunc("/{id}/start", httpHandler(handlers.StartWorkspaceHandler)).Methods("POST")
	workspaces.HandleFunc("/{id}/stop", httpHandler(handlers.StopWorkspaceHandler)).Methods("POST")
	workspaces.HandleFunc("/{id}/restart", httpHandler(handlers.RestartWorkspaceHandler)).Methods("POST")
}

func httpHandler(h handlers.Handler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = h(w, r)
	}
}
