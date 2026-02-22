package handlers

import "net/http"

func GitHubAuthHandler(w http.ResponseWriter, r *http.Request) error {
	return WriteJSON(w, http.StatusOK, map[string]string{"message": "GitHub auth endpoint"})
}

func GitHubCallbackHandler(w http.ResponseWriter, r *http.Request) error {
	return WriteJSON(w, http.StatusOK, map[string]string{"message": "GitHub callback endpoint"})
}
