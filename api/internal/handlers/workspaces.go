package handlers

import "net/http"

func ListWorkspacesHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "List workspaces"})
	return nil
}

func CreateWorkspaceHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "Create workspace"})
	return nil
}

func GetWorkspaceHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "Get workspace"})
	return nil
}

func UpdateWorkspaceHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "Update workspace"})
	return nil
}

func DeleteWorkspaceHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "Delete workspace"})
	return nil
}

func StartWorkspaceHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "Start workspace"})
	return nil
}

func StopWorkspaceHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "Stop workspace"})
	return nil
}

func RestartWorkspaceHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "Restart workspace"})
	return nil
}
