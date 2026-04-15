package handlers

import (
	"fmt"
	"net/http"
	"os"
)

func GitHubAuthHandler(w http.ResponseWriter, r *http.Request) error {
	config := getGitHubOAuthConfig()
	if config == nil {
		return fmt.Errorf("GitHub OAuth not configured")
	}

	host := r.Host
	scheme := "http"
	if forwardedProto := r.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = forwardedProto
	} else if r.TLS != nil {
		scheme = "https"
	}

	callbackURL := os.Getenv("GITHUB_CALLBACK_URL")
	if callbackURL == "" {
		callbackURL = fmt.Sprintf("%s://%s/oauth/callback", scheme, host)
	}

	config.RedirectURL = callbackURL

	state, err := generateStateToken()
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}

	if err := StoreOAuthState(state); err != nil {
		return fmt.Errorf("failed to store OAuth state: %w", err)
	}

	authURL := config.AuthCodeURL(state)
	http.Redirect(w, r, authURL, http.StatusFound)
	return nil
}

func GitHubCallbackHandler(w http.ResponseWriter, r *http.Request) error {
	return WriteJSON(w, http.StatusOK, map[string]string{"message": "GitHub callback endpoint"})
}
