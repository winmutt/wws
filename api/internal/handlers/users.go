package handlers

import "net/http"

func ListUsersHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "List users"})
	return nil
}

func GetUserHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "Get user"})
	return nil
}
