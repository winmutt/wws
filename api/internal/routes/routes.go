package routes

import "github.com/gorilla/mux"

func SetupRoutes(r *mux.Router) {
	api := r.PathPrefix("/api/v1").Subrouter()
	
	// Health check
	api.HandleFunc("/health", HealthHandler).Methods("GET")
	
	// Auth routes
	auth := api.PathPrefix("/auth").Subrouter()
	auth.HandleFunc("/github", GitHubAuthHandler).Methods("GET")
	auth.HandleFunc("/github/callback", GitHubCallbackHandler).Methods("GET")
	
	// User routes
	users := api.PathPrefix("/users").Subrouter()
	users.HandleFunc("", ListUsersHandler).Methods("GET")
	users.HandleFunc("/{id}", GetUserHandler).Methods("GET")
	
	// Organization routes
	orgs := api.PathPrefix("/organizations").Subrouter()
	orgs.HandleFunc("", ListOrganizationsHandler).Methods("GET")
	orgs.HandleFunc("", CreateOrganizationHandler).Methods("POST")
	orgs.HandleFunc("/{id}", GetOrganizationHandler).Methods("GET")
	orgs.HandleFunc("/{id}", UpdateOrganizationHandler).Methods("PUT")
	orgs.HandleFunc("/{id}", DeleteOrganizationHandler).Methods("DELETE")
	orgs.HandleFunc("/{id}/invitations", CreateInvitationHandler).Methods("POST")
	orgs.HandleFunc("/invitations/{id}/accept", AcceptInvitationHandler).Methods("POST")
	
	// Workspace routes
	workspaces := api.PathPrefix("/workspaces").Subrouter()
	workspaces.HandleFunc("", ListWorkspacesHandler).Methods("GET")
	workspaces.HandleFunc("", CreateWorkspaceHandler).Methods("POST")
	workspaces.HandleFunc("/{id}", GetWorkspaceHandler).Methods("GET")
	workspaces.HandleFunc("/{id}", UpdateWorkspaceHandler).Methods("PUT")
	workspaces.HandleFunc("/{id}", DeleteWorkspaceHandler).Methods("DELETE")
	workspaces.HandleFunc("/{id}/start", StartWorkspaceHandler).Methods("POST")
	workspaces.HandleFunc("/{id}/stop", StopWorkspaceHandler).Methods("POST")
	workspaces.HandleFunc("/{id}/restart", RestartWorkspaceHandler).Methods("POST")
}
