package handlers

import "net/http"

func ListOrganizationsHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "List organizations"})
	return nil
}

func CreateOrganizationHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "Create organization"})
	return nil
}

func GetOrganizationHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "Get organization"})
	return nil
}

func UpdateOrganizationHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "Update organization"})
	return nil
}

func DeleteOrganizationHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "Delete organization"})
	return nil
}

func CreateInvitationHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "Create invitation"})
	return nil
}

func AcceptInvitationHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "Accept invitation"})
	return nil
}
