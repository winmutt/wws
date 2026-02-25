package handlers

import (
	"fmt"
	"net/http"
)

func GitHubAuthHandler(w http.ResponseWriter, r *http.Request) error {
	config := getGitHubOAuthConfig()
	if config == nil {
		return fmt.Errorf("GitHub OAuth not configured")
	}

	state, err := generateStateToken()
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}

	authURL := config.AuthCodeURL(state)
	http.Redirect(w, r, authURL, http.StatusFound)
	return nil
}

func GitHubCallbackHandler(w http.ResponseWriter, r *http.Request) error {
	return WriteJSON(w, http.StatusOK, map[string]string{"message": "GitHub callback endpoint"})
}
