package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var (
	githubOAuthConfig *oauth2.Config
	once              sync.Once
)

func getGitHubOAuthConfig() *oauth2.Config {
	once.Do(func() {
		clientID := os.Getenv("GITHUB_CLIENT_ID")
		clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
		callbackURL := os.Getenv("GITHUB_CALLBACK_URL")

		if clientID == "" || clientSecret == "" || callbackURL == "" {
			log.Fatal("GitHub OAuth configuration is incomplete")
		}

		githubOAuthConfig = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  callbackURL,
			Endpoint:     github.Endpoint,
			Scopes:       []string{"user:email", "read:user"},
		}
	})

	return githubOAuthConfig
}

type OAuthStateStore struct {
	state string
}

func generateStateToken() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func OAuthCallbackHandler(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	state := r.URL.Query().Get("state")
	if state == "" {
		return fmt.Errorf("missing state parameter")
	}

	if !strings.HasPrefix(state, "gh_") {
		return fmt.Errorf("invalid state parameter")
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		return fmt.Errorf("missing authorization code")
	}

	config := getGitHubOAuthConfig()
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("failed to exchange token: %w", err)
	}

	client := config.Client(ctx, token)
	user, err := getUserInfo(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	sessionToken, err := createSession(ctx, token, user)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7,
	})

	http.Redirect(w, r, "/dashboard", http.StatusFound)
	return nil
}

func getUserInfo(ctx context.Context, client *http.Client) (map[string]interface{}, error) {
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var user map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return user, nil
}

func createSession(ctx context.Context, token *oauth2.Token, user map[string]interface{}) (string, error) {
	githubID, ok := user["id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid github ID")
	}

	username, ok := user["login"].(string)
	if !ok {
		return "", fmt.Errorf("invalid username")
	}

	_, _ = getPrimaryEmail(ctx, token)

	sessionToken := generateSessionToken()

	log.Printf("OAuth session created for user: %s (%s)", username, githubID)

	return sessionToken, nil
}

func getPrimaryEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	config := getGitHubOAuthConfig()
	client := config.Client(ctx, token)
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", fmt.Errorf("failed to fetch emails: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var emails []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("failed to decode emails: %w", err)
	}

	for _, email := range emails {
		if email["primary"].(bool) {
			return email["email"].(string), nil
		}
	}

	return "", nil
}

func generateSessionToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
