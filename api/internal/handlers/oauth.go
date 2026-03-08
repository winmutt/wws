package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"wws/api/internal/crypto"
	"wws/api/internal/db"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var (
	githubOAuthConfig *oauth2.Config
	once              sync.Once
	stateStore        map[string]bool
	stateStoreMutex   sync.RWMutex
	oauthDB           *sql.DB
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

func InitOAuthStateStore() {
	stateStore = make(map[string]bool)
	oauthDB = db.DB

	if err := crypto.InitEncryption(); err != nil {
		log.Printf("Warning: %v", err)
	}
}

func SetOAuthDB(db *sql.DB) {
	oauthDB = db
}

func StoreOAuthState(state string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hash := sha256.Sum256([]byte(state))
	hashStr := base64.URLEncoding.EncodeToString(hash[:])

	_, err := oauthDB.ExecContext(ctx,
		"INSERT INTO oauth_states (state) VALUES (?)",
		hashStr,
	)
	if err != nil {
		return fmt.Errorf("failed to store OAuth state: %w", err)
	}

	stateStoreMutex.Lock()
	stateStore[hashStr] = true
	stateStoreMutex.Unlock()

	if os.Getenv("WWS_TEST_MODE") != "true" {
		go cleanupExpiredStates()
	}

	return nil
}

func ValidateOAuthState(state string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hash := sha256.Sum256([]byte(state))
	hashStr := base64.URLEncoding.EncodeToString(hash[:])

	stateStoreMutex.RLock()
	if stateStore[hashStr] {
		stateStoreMutex.RUnlock()
		oauthDB.ExecContext(ctx, "DELETE FROM oauth_states WHERE state = ?", hashStr)
		stateStoreMutex.Lock()
		delete(stateStore, hashStr)
		stateStoreMutex.Unlock()
		return true
	}
	stateStoreMutex.RUnlock()

	var exists int
	err := oauthDB.QueryRowContext(ctx, "SELECT 1 FROM oauth_states WHERE state = ?", hashStr).Scan(&exists)
	if err == nil {
		oauthDB.ExecContext(ctx, "DELETE FROM oauth_states WHERE state = ?", hashStr)
		return true
	}

	return false
}

func cleanupExpiredStates() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := oauthDB.ExecContext(ctx,
		"DELETE FROM oauth_states WHERE created_at < datetime('now', '-10 minutes')",
	)
	if err != nil {
		log.Printf("Failed to cleanup expired OAuth states: %v", err)
	}

	stateStoreMutex.Lock()
	for state, _ := range stateStore {
		var exists int
		err := oauthDB.QueryRowContext(ctx, "SELECT 1 FROM oauth_states WHERE state = ?", state).Scan(&exists)
		if err != nil {
			delete(stateStore, state)
		}
	}
	stateStoreMutex.Unlock()
}

func OAuthCallbackHandler(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	state := r.URL.Query().Get("state")
	if state == "" {
		return fmt.Errorf("missing state parameter")
	}

	if !ValidateOAuthState(state) {
		return fmt.Errorf("invalid or expired state parameter")
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

	userID, err := findOrCreateUser(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	if err := storeOAuthToken(ctx, userID, token); err != nil {
		return fmt.Errorf("failed to store OAuth token: %w", err)
	}

	sessionToken, err := createSession(ctx, userID, user)
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

func generateSessionToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func findOrCreateUser(ctx context.Context, user map[string]interface{}) (int, error) {
	githubID, ok := user["id"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid github ID type")
	}
	githubIDStr := fmt.Sprintf("%.0f", githubID)

	username, ok := user["login"].(string)
	if !ok {
		return 0, fmt.Errorf("invalid username")
	}

	var email string
	emailVal, exists := user["email"]
	if exists && emailVal != nil {
		email = emailVal.(string)
	}

	var existingUserID int
	err := oauthDB.QueryRowContext(ctx,
		"SELECT id FROM users WHERE github_id = ?",
		githubIDStr,
	).Scan(&existingUserID)
	if err == nil {
		return existingUserID, nil
	}

	result, err := oauthDB.ExecContext(ctx,
		"INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		githubIDStr, username, email,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get user ID: %w", err)
	}

	log.Printf("Created new user: %s (%s)", username, githubIDStr)

	return int(id), nil
}

func storeOAuthToken(ctx context.Context, userID int, token *oauth2.Token) error {
	expiry := token.Expiry
	if expiry.IsZero() {
		expiry = time.Now().Add(3600 * time.Second)
	}

	encryptedAccessToken, err := crypto.Encrypt(token.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt access token: %w", err)
	}

	var encryptedRefreshToken interface{}
	if token.RefreshToken != "" {
		encryptedRT, err := crypto.Encrypt(token.RefreshToken)
		if err != nil {
			return fmt.Errorf("failed to encrypt refresh token: %w", err)
		}
		encryptedRefreshToken = encryptedRT
	} else {
		encryptedRefreshToken = nil
	}

	_, err = oauthDB.ExecContext(ctx,
		`INSERT INTO oauth_tokens (user_id, access_token, encrypted_access_token, refresh_token, encrypted_refresh_token, expiry)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		 encrypted_access_token = excluded.encrypted_access_token,
		 encrypted_refresh_token = excluded.encrypted_refresh_token,
		 expiry = excluded.expiry,
		 updated_at = CURRENT_TIMESTAMP`,
		userID,
		nil,
		encryptedAccessToken,
		nil,
		encryptedRefreshToken,
		expiry,
	)
	if err != nil {
		return fmt.Errorf("failed to store OAuth token: %w", err)
	}

	log.Printf("Stored encrypted OAuth token for user ID: %d", userID)

	return nil
}

func createSession(ctx context.Context, userID int, user map[string]interface{}) (string, error) {
	sessionToken := generateSessionToken()
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	_, err := oauthDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		userID, sessionToken, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("failed to store session: %w", err)
	}

	username, _ := user["login"].(string)
	log.Printf("Created session for user: %s (ID: %d)", username, userID)

	return sessionToken, nil
}

func GetOAuthToken(ctx context.Context, userID int) (*oauth2.Token, error) {
	var encryptedAccessToken, encryptedRefreshToken sql.NullString
	var expiry time.Time

	err := oauthDB.QueryRowContext(ctx,
		`SELECT encrypted_access_token, encrypted_refresh_token, expiry 
		 FROM oauth_tokens WHERE user_id = ?`,
		userID,
	).Scan(&encryptedAccessToken, &encryptedRefreshToken, &expiry)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no OAuth token found for user ID: %d", userID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OAuth token: %w", err)
	}

	if !encryptedAccessToken.Valid {
		return nil, fmt.Errorf("access token not found")
	}

	decryptedAccessToken, err := crypto.Decrypt(encryptedAccessToken.String)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt access token: %w", err)
	}

	var decryptedRefreshToken string
	if encryptedRefreshToken.Valid && encryptedRefreshToken.String != "" {
		decryptedRefreshToken, err = crypto.Decrypt(encryptedRefreshToken.String)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt refresh token: %w", err)
		}
	}

	return &oauth2.Token{
		AccessToken:  decryptedAccessToken,
		RefreshToken: decryptedRefreshToken,
		Expiry:       expiry,
	}, nil
}

type SessionInfo struct {
	UserID    int       `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	IsValid   bool      `json:"is_valid"`
}

func ValidateSession(ctx context.Context, sessionToken string) (*SessionInfo, error) {
	var sessionInfo SessionInfo
	var expiresAt, createdAt time.Time
	var userID int

	err := oauthDB.QueryRowContext(ctx,
		`SELECT user_id, expires_at, created_at FROM sessions WHERE token = ?`,
		sessionToken,
	).Scan(&userID, &expiresAt, &createdAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to validate session: %w", err)
	}

	sessionInfo.UserID = userID
	sessionInfo.ExpiresAt = expiresAt
	sessionInfo.CreatedAt = createdAt
	sessionInfo.IsValid = time.Now().Before(expiresAt)

	if !sessionInfo.IsValid {
		oauthDB.ExecContext(ctx, "DELETE FROM sessions WHERE token = ?", sessionToken)
		return nil, fmt.Errorf("session expired")
	}

	return &sessionInfo, nil
}

func RefreshSession(ctx context.Context, sessionToken string) (*SessionInfo, error) {
	newExpiresAt := time.Now().Add(7 * 24 * time.Hour)

	_, err := oauthDB.ExecContext(ctx,
		"UPDATE sessions SET expires_at = ? WHERE token = ?",
		newExpiresAt, sessionToken,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh session: %w", err)
	}

	var userID int
	var createdAt time.Time
	err = oauthDB.QueryRowContext(ctx,
		"SELECT user_id, created_at FROM sessions WHERE token = ?",
		sessionToken,
	).Scan(&userID, &createdAt)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch session after refresh: %w", err)
	}

	return &SessionInfo{
		UserID:    userID,
		ExpiresAt: newExpiresAt,
		CreatedAt: createdAt,
		IsValid:   true,
	}, nil
}

func RevokeSession(ctx context.Context, sessionToken string) error {
	result, err := oauthDB.ExecContext(ctx, "DELETE FROM sessions WHERE token = ?", sessionToken)
	if err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check session revocation: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}

func RevokeAllUserSessions(ctx context.Context, userID int) error {
	result, err := oauthDB.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to revoke user sessions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check session revocation: %w", err)
	}

	log.Printf("Revoked %d sessions for user ID: %d", rowsAffected, userID)
	return nil
}

func CleanupExpiredSessions() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := oauthDB.ExecContext(ctx,
		"DELETE FROM sessions WHERE expires_at < datetime('now')",
	)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check cleanup results: %w", err)
	}

	if rowsAffected > 0 {
		log.Printf("Cleaned up %d expired sessions", rowsAffected)
	}

	return nil
}

func GetUserSessions(ctx context.Context, userID int) ([]SessionInfo, error) {
	rows, err := oauthDB.QueryContext(ctx,
		`SELECT user_id, expires_at, created_at FROM sessions WHERE user_id = ? ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionInfo
	for rows.Next() {
		var session SessionInfo
		if err := rows.Scan(&session.UserID, &session.ExpiresAt, &session.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		session.IsValid = time.Now().Before(session.ExpiresAt)
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}

func GetSessionHandler(w http.ResponseWriter, r *http.Request) error {
	sessionToken := r.Header.Get("Authorization")
	if sessionToken == "" {
		return fmt.Errorf("missing authorization header")
	}

	sessionToken = trimBearer(sessionToken)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	sessionInfo, err := ValidateSession(ctx, sessionToken)
	if err != nil {
		return fmt.Errorf("invalid session: %w", err)
	}

	return WriteJSON(w, http.StatusOK, sessionInfo)
}

func RefreshSessionHandler(w http.ResponseWriter, r *http.Request) error {
	sessionToken := r.Header.Get("Authorization")
	if sessionToken == "" {
		return fmt.Errorf("missing authorization header")
	}

	sessionToken = trimBearer(sessionToken)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	sessionInfo, err := RefreshSession(ctx, sessionToken)
	if err != nil {
		return fmt.Errorf("failed to refresh session: %w", err)
	}

	return WriteJSON(w, http.StatusOK, sessionInfo)
}

func RevokeSessionHandler(w http.ResponseWriter, r *http.Request) error {
	sessionToken := r.Header.Get("Authorization")
	if sessionToken == "" {
		return fmt.Errorf("missing authorization header")
	}

	sessionToken = trimBearer(sessionToken)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := RevokeSession(ctx, sessionToken); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

func RevokeAllSessionsHandler(w http.ResponseWriter, r *http.Request) error {
	sessionToken := r.Header.Get("Authorization")
	if sessionToken == "" {
		return fmt.Errorf("missing authorization header")
	}

	sessionToken = trimBearer(sessionToken)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	sessionInfo, err := ValidateSession(ctx, sessionToken)
	if err != nil {
		return fmt.Errorf("invalid session: %w", err)
	}

	if err := RevokeAllUserSessions(ctx, sessionInfo.UserID); err != nil {
		return fmt.Errorf("failed to revoke sessions: %w", err)
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

func ListUserSessionsHandler(w http.ResponseWriter, r *http.Request) error {
	sessionToken := r.Header.Get("Authorization")
	if sessionToken == "" {
		return fmt.Errorf("missing authorization header")
	}

	sessionToken = trimBearer(sessionToken)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	sessionInfo, err := ValidateSession(ctx, sessionToken)
	if err != nil {
		return fmt.Errorf("invalid session: %w", err)
	}

	sessions, err := GetUserSessions(ctx, sessionInfo.UserID)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	return WriteJSON(w, http.StatusOK, sessions)
}

func trimBearer(token string) string {
	if len(token) > 7 && token[:7] == "Bearer " {
		return token[7:]
	}
	return token
}
