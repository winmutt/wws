package handlers

import "net/http"

func ListUsersHandler(w http.ResponseWriter, r *http.Request) error {
	return WriteJSON(w, http.StatusOK, map[string]string{"message": "List users"})
}

func GetUserHandler(w http.ResponseWriter, r *http.Request) error {
	return WriteJSON(w, http.StatusOK, map[string]string{"message": "Get user"})
}
