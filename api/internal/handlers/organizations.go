package handlers

import "net/http"

func ListOrganizationsHandler(w http.ResponseWriter, r *http.Request) error {
	return WriteJSON(w, http.StatusOK, map[string]string{"message": "List organizations"})
}

func CreateOrganizationHandler(w http.ResponseWriter, r *http.Request) error {
	return WriteJSON(w, http.StatusOK, map[string]string{"message": "Create organization"})
}

func GetOrganizationHandler(w http.ResponseWriter, r *http.Request) error {
	return WriteJSON(w, http.StatusOK, map[string]string{"message": "Get organization"})
}

func UpdateOrganizationHandler(w http.ResponseWriter, r *http.Request) error {
	return WriteJSON(w, http.StatusOK, map[string]string{"message": "Update organization"})
}

func DeleteOrganizationHandler(w http.ResponseWriter, r *http.Request) error {
	return WriteJSON(w, http.StatusOK, map[string]string{"message": "Delete organization"})
}

func CreateInvitationHandler(w http.ResponseWriter, r *http.Request) error {
	return WriteJSON(w, http.StatusOK, map[string]string{"message": "Create invitation"})
}

func AcceptInvitationHandler(w http.ResponseWriter, r *http.Request) error {
	return WriteJSON(w, http.StatusOK, map[string]string{"message": "Accept invitation"})
}
