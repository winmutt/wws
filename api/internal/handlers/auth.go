package handlers

import "net/http"

func GitHubAuthHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "GitHub auth endpoint"})
	return nil
}

func GitHubCallbackHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"message": "GitHub callback endpoint"})
	return nil
}
