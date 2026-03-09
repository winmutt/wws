package handlers

import (
	"net/http"

	"wws/api/workspace"
)

// Workspace handlers - adapters for the workspace package
var (
	CreateWorkspaceHandler Handler = func(w http.ResponseWriter, r *http.Request) error {
		h := workspace.NewWorkspaceHandler()
		h.NewWorkspace(w, r)
		return nil
	}

	ListWorkspacesHandler Handler = func(w http.ResponseWriter, r *http.Request) error {
		h := workspace.NewWorkspaceHandler()
		h.GetWorkspaces(w, r)
		return nil
	}

	GetWorkspaceHandler Handler = func(w http.ResponseWriter, r *http.Request) error {
		h := workspace.NewWorkspaceHandler()
		h.GetWorkspace(w, r)
		return nil
	}

	UpdateWorkspaceHandler Handler = func(w http.ResponseWriter, r *http.Request) error {
		h := workspace.NewWorkspaceHandler()
		h.UpdateWorkspace(w, r)
		return nil
	}

	DeleteWorkspaceHandler Handler = func(w http.ResponseWriter, r *http.Request) error {
		h := workspace.NewWorkspaceHandler()
		h.DeleteWorkspace(w, r)
		return nil
	}

	StartWorkspaceHandler Handler = func(w http.ResponseWriter, r *http.Request) error {
		h := workspace.NewWorkspaceHandler()
		h.StartWorkspace(w, r)
		return nil
	}

	StopWorkspaceHandler Handler = func(w http.ResponseWriter, r *http.Request) error {
		h := workspace.NewWorkspaceHandler()
		h.StopWorkspace(w, r)
		return nil
	}

	RestartWorkspaceHandler Handler = func(w http.ResponseWriter, r *http.Request) error {
		h := workspace.NewWorkspaceHandler()
		h.RestartWorkspace(w, r)
		return nil
	}
)
