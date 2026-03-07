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
