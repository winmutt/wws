package handlers

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"wws/api/internal/crypto"

	"golang.org/x/oauth2"
)

func TestStoreOAuthTokenEncrypted(t *testing.T) {
	crypto.InitEncryption()

	oauthDB = testDB2

	ctx := context.Background()
	userID := 1

	token := &oauth2.Token{
		AccessToken:  "test-access-token-12345",
		RefreshToken: "test-refresh-token-67890",
		Expiry:       time.Now().Add(time.Hour),
	}

	err := storeOAuthToken(ctx, userID, token)
	if err != nil {
		t.Fatalf("storeOAuthToken failed: %v", err)
	}

	var encryptedAccessToken, encryptedRefreshToken string
	err = oauthDB.QueryRowContext(ctx,
		`SELECT encrypted_access_token, encrypted_refresh_token FROM oauth_tokens WHERE user_id = ?`,
		userID,
	).Scan(&encryptedAccessToken, &encryptedRefreshToken)

	if err != nil {
		t.Fatalf("Failed to query token: %v", err)
	}

	if encryptedAccessToken == "" {
		t.Error("encrypted access token should not be empty")
	}

	if encryptedRefreshToken == "" {
		t.Error("encrypted refresh token should not be empty")
	}

	if encryptedAccessToken == token.AccessToken {
		t.Error("stored token should be encrypted, not plaintext")
	}
}

func TestGetOAuthTokenDecrypted(t *testing.T) {
	os.Setenv("WWS_ENCRYPTION_KEY", "test-encryption-key-for-oauth")
	crypto.InitEncryption()

	oauthDB = testDB2

	ctx := context.Background()
	userID := 2

	token := &oauth2.Token{
		AccessToken:  "test-access-token-54321",
		RefreshToken: "test-refresh-token-09876",
		Expiry:       time.Now().Add(time.Hour),
	}

	err := storeOAuthToken(ctx, userID, token)
	if err != nil {
		t.Fatalf("storeOAuthToken failed: %v", err)
	}

	retrievedToken, err := GetOAuthToken(ctx, userID)
	if err != nil {
		t.Fatalf("GetOAuthToken failed: %v", err)
	}

	if retrievedToken.AccessToken != token.AccessToken {
		t.Errorf("retrieved access token '%s' does not match original '%s'",
			retrievedToken.AccessToken, token.AccessToken)
	}

	if retrievedToken.RefreshToken != token.RefreshToken {
		t.Errorf("retrieved refresh token '%s' does not match original '%s'",
			retrievedToken.RefreshToken, token.RefreshToken)
	}
}

func TestGetOAuthTokenNotFound(t *testing.T) {
	os.Setenv("WWS_ENCRYPTION_KEY", "test-encryption-key")
	crypto.InitEncryption()

	oauthDB = testDB2

	ctx := context.Background()
	nonExistentUserID := 99999

	_, err := GetOAuthToken(ctx, nonExistentUserID)
	if err == nil {
		t.Error("expected error for non-existent user")
	}
}

func TestStoreOAuthTokenWithoutRefresh(t *testing.T) {
	crypto.InitEncryption()

	oauthDB = testDB2

	ctx := context.Background()
	userID := 3

	token := &oauth2.Token{
		AccessToken:  "test-access-only",
		RefreshToken: "",
		Expiry:       time.Now().Add(time.Hour),
	}

	err := storeOAuthToken(ctx, userID, token)
	if err != nil {
		t.Fatalf("storeOAuthToken failed: %v", err)
	}

	var refreshTokenValue sql.NullString
	err = oauthDB.QueryRowContext(ctx,
		`SELECT encrypted_refresh_token FROM oauth_tokens WHERE user_id = ?`,
		userID,
	).Scan(&refreshTokenValue)

	if err != nil {
		t.Fatalf("Failed to query token: %v", err)
	}

	if refreshTokenValue.Valid {
		t.Error("refresh token should be NULL in database")
	}

	retrievedToken, err := GetOAuthToken(ctx, userID)
	if err != nil {
		t.Fatalf("GetOAuthToken failed: %v", err)
	}

	if retrievedToken.AccessToken != token.AccessToken {
		t.Errorf("access token mismatch")
	}

	if retrievedToken.RefreshToken != "" {
		t.Error("refresh token should be empty")
	}
}

func TestOAuthTokenNotStoredPlaintext(t *testing.T) {
	crypto.InitEncryption()

	oauthDB = testDB2

	ctx := context.Background()
	userID := 4

	token := &oauth2.Token{
		AccessToken:  "ghp_plaintext-test-token",
		RefreshToken: "ghr_plaintext-refresh",
		Expiry:       time.Now().Add(time.Hour),
	}

	err := storeOAuthToken(ctx, userID, token)
	if err != nil {
		t.Fatalf("storeOAuthToken failed: %v", err)
	}

	var accessToken, refreshToken sql.NullString
	err = oauthDB.QueryRowContext(ctx,
		`SELECT access_token, refresh_token FROM oauth_tokens WHERE user_id = ?`,
		userID,
	).Scan(&accessToken, &refreshToken)

	if err != nil {
		t.Fatalf("Failed to query token: %v", err)
	}

	if accessToken.Valid && accessToken.String == token.AccessToken {
		t.Error("access_token should not be stored in plaintext")
	}

	if refreshToken.Valid && refreshToken.String == token.RefreshToken {
		t.Error("refresh_token should not be stored in plaintext")
	}
}

func TestValidateSession(t *testing.T) {
	crypto.InitEncryption()
	oauthDB = testDB2

	ctx := context.Background()

	validToken := "test-valid-session-token-12345"
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err := oauthDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at, created_at) VALUES (?, ?, ?, ?)",
		1, validToken, expiresAt, time.Now(),
	)
	if err != nil {
		t.Fatalf("Failed to insert session: %v", err)
	}

	sessionInfo, err := ValidateSession(ctx, validToken)
	if err != nil {
		t.Fatalf("ValidateSession failed: %v", err)
	}

	if sessionInfo.UserID != 1 {
		t.Errorf("Expected user ID 1, got %d", sessionInfo.UserID)
	}

	if !sessionInfo.IsValid {
		t.Error("Session should be valid")
	}
}

func TestValidateSessionExpired(t *testing.T) {
	crypto.InitEncryption()
	oauthDB = testDB2

	ctx := context.Background()

	expiredToken := "test-expired-session-token"
	expiresAt := time.Now().Add(-24 * time.Hour)

	_, err := oauthDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at, created_at) VALUES (?, ?, ?, ?)",
		1, expiredToken, expiresAt, time.Now(),
	)
	if err != nil {
		t.Fatalf("Failed to insert session: %v", err)
	}

	_, err = ValidateSession(ctx, expiredToken)
	if err == nil {
		t.Error("Expected error for expired session")
	}
}

func TestValidateSessionNotFound(t *testing.T) {
	crypto.InitEncryption()
	oauthDB = testDB2

	ctx := context.Background()

	_, err := ValidateSession(ctx, "non-existent-token")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

func TestRefreshSession(t *testing.T) {
	crypto.InitEncryption()
	oauthDB = testDB2

	ctx := context.Background()

	refreshToken := "test-refresh-token-12345"
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err := oauthDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at, created_at) VALUES (?, ?, ?, ?)",
		1, refreshToken, expiresAt, time.Now(),
	)
	if err != nil {
		t.Fatalf("Failed to insert session: %v", err)
	}

	sessionInfo, err := RefreshSession(ctx, refreshToken)
	if err != nil {
		t.Fatalf("RefreshSession failed: %v", err)
	}

	if sessionInfo.UserID != 1 {
		t.Errorf("Expected user ID 1, got %d", sessionInfo.UserID)
	}

	if time.Now().After(sessionInfo.ExpiresAt) {
		t.Error("Session should have been extended")
	}
}

func TestRevokeSession(t *testing.T) {
	crypto.InitEncryption()
	oauthDB = testDB2

	ctx := context.Background()

	revokeToken := "test-revoke-token-12345"
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err := oauthDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at, created_at) VALUES (?, ?, ?, ?)",
		1, revokeToken, expiresAt, time.Now(),
	)
	if err != nil {
		t.Fatalf("Failed to insert session: %v", err)
	}

	err = RevokeSession(ctx, revokeToken)
	if err != nil {
		t.Fatalf("RevokeSession failed: %v", err)
	}

	var count int
	err = oauthDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE token = ?", revokeToken).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count sessions: %v", err)
	}

	if count != 0 {
		t.Error("Session should have been deleted")
	}
}

func TestRevokeAllUserSessions(t *testing.T) {
	crypto.InitEncryption()
	oauthDB = testDB2

	ctx := context.Background()

	_, err := oauthDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at, created_at) VALUES (?, ?, ?, ?)",
		1, "session-1", time.Now().Add(24*time.Hour), time.Now(),
	)
	if err != nil {
		t.Fatalf("Failed to insert session 1: %v", err)
	}

	_, err = oauthDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at, created_at) VALUES (?, ?, ?, ?)",
		1, "session-2", time.Now().Add(24*time.Hour), time.Now(),
	)
	if err != nil {
		t.Fatalf("Failed to insert session 2: %v", err)
	}

	err = RevokeAllUserSessions(ctx, 1)
	if err != nil {
		t.Fatalf("RevokeAllUserSessions failed: %v", err)
	}

	var count int
	err = oauthDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE user_id = ?", 1).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count sessions: %v", err)
	}

	if count != 0 {
		t.Error("All sessions should have been deleted")
	}
}

func TestCleanupExpiredSessions(t *testing.T) {
	crypto.InitEncryption()
	oauthDB = testDB2

	ctx := context.Background()

	_, err := oauthDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at, created_at) VALUES (?, ?, ?, ?)",
		1, "expired-session", time.Now().Add(-24*time.Hour), time.Now(),
	)
	if err != nil {
		t.Fatalf("Failed to insert expired session: %v", err)
	}

	_, err = oauthDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at, created_at) VALUES (?, ?, ?, ?)",
		1, "valid-session", time.Now().Add(24*time.Hour), time.Now(),
	)
	if err != nil {
		t.Fatalf("Failed to insert valid session: %v", err)
	}

	err = CleanupExpiredSessions()
	if err != nil {
		t.Fatalf("CleanupExpiredSessions failed: %v", err)
	}

	var count int
	err = oauthDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE token = ?", "expired-session").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count expired sessions: %v", err)
	}

	if count != 0 {
		t.Error("Expired session should have been deleted")
	}

	var validCount int
	err = oauthDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE token = ?", "valid-session").Scan(&validCount)
	if err != nil {
		t.Fatalf("Failed to count valid sessions: %v", err)
	}

	if validCount != 1 {
		t.Error("Valid session should still exist")
	}
}

func TestGetUserSessions(t *testing.T) {
	crypto.InitEncryption()
	oauthDB = testDB2

	ctx := context.Background()

	_, err := oauthDB.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = ?", 1)
	if err != nil {
		t.Fatalf("Failed to cleanup existing sessions: %v", err)
	}

	_, err = oauthDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at, created_at) VALUES (?, ?, ?, ?)",
		1, "user-session-1", time.Now().Add(24*time.Hour), time.Now(),
	)
	if err != nil {
		t.Fatalf("Failed to insert session 1: %v", err)
	}

	_, err = oauthDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at, created_at) VALUES (?, ?, ?, ?)",
		1, "user-session-2", time.Now().Add(24*time.Hour), time.Now(),
	)
	if err != nil {
		t.Fatalf("Failed to insert session 2: %v", err)
	}

	sessions, err := GetUserSessions(ctx, 1)
	if err != nil {
		t.Fatalf("GetUserSessions failed: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}

	for _, session := range sessions {
		if session.UserID != 1 {
			t.Errorf("Expected user ID 1, got %d", session.UserID)
		}
	}
}
