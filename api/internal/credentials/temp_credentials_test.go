package credentials

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.DefaultTTL != 24*time.Hour {
		t.Errorf("Expected DefaultTTL 24h, got %v", config.DefaultTTL)
	}

	if config.MaxTTL != 7*24*time.Hour {
		t.Errorf("Expected MaxTTL 168h, got %v", config.MaxTTL)
	}

	if !config.AutoRefresh {
		t.Error("Expected AutoRefresh to be true by default")
	}
}

func TestNewCredentialManager(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewCredentialManager(nil, storage)

	if manager == nil {
		t.Fatal("Expected non-nil credential manager")
	}

	if manager.Config().DefaultTTL != 24*time.Hour {
		t.Error("Expected default TTL to be set")
	}
}

func TestNewCredentialManagerWithConfig(t *testing.T) {
	storage := NewMemoryStorage()
	config := &TempCredentialConfig{
		DefaultTTL:       12 * time.Hour,
		MaxTTL:           3 * 24 * time.Hour,
		RefreshThreshold: 30 * time.Minute,
		AutoRefresh:      false,
	}

	manager := NewCredentialManager(config, storage)

	if manager.Config().DefaultTTL != 12*time.Hour {
		t.Errorf("Expected DefaultTTL 12h, got %v", manager.Config().DefaultTTL)
	}

	if manager.Config().AutoRefresh != false {
		t.Error("Expected AutoRefresh to be false")
	}
}

func TestGenerateTempCredential(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewCredentialManager(nil, storage)

	cred, err := manager.GenerateTempCredential(1, "jwt", nil, nil)
	if err != nil {
		t.Fatalf("Failed to generate credential: %v", err)
	}

	if cred.ID == "" {
		t.Error("Expected non-empty credential ID")
	}

	if cred.UserID != 1 {
		t.Errorf("Expected user ID 1, got %d", cred.UserID)
	}

	if cred.Type != "jwt" {
		t.Errorf("Expected type 'jwt', got %s", cred.Type)
	}

	if cred.Token == "" {
		t.Error("Expected non-empty token")
	}

	if !strings.HasPrefix(cred.Token, "wws_temp_") {
		t.Errorf("Expected token to start with 'wws_temp_', got %s", cred.Token[:10])
	}

	if time.Now().After(cred.ExpiresAt) {
		t.Error("Expected credential to be valid")
	}
}

func TestGenerateTempCredentialWithWorkspace(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewCredentialManager(nil, storage)

	workspaceID := 42
	cred, err := manager.GenerateTempCredential(1, "api_key", &workspaceID, nil)
	if err != nil {
		t.Fatalf("Failed to generate credential: %v", err)
	}

	if cred.WorkspaceID == nil {
		t.Error("Expected workspace ID to be set")
	} else if *cred.WorkspaceID != 42 {
		t.Errorf("Expected workspace ID 42, got %d", *cred.WorkspaceID)
	}
}

func TestGenerateTempCredentialInvalidType(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewCredentialManager(nil, storage)

	_, err := manager.GenerateTempCredential(1, "invalid_type", nil, nil)
	if err == nil {
		t.Error("Expected error for invalid credential type")
	}
}

func TestGenerateTempCredentialWithCustomTTL(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewCredentialManager(nil, storage)

	customTTL := 2 * time.Hour
	cred, err := manager.GenerateTempCredential(1, "jwt", nil, &customTTL)
	if err != nil {
		t.Fatalf("Failed to generate credential: %v", err)
	}

	expectedExpiry := time.Now().Add(customTTL)
	if cred.ExpiresAt.After(expectedExpiry.Add(time.Minute)) || cred.ExpiresAt.Before(expectedExpiry.Add(-time.Minute)) {
		t.Errorf("Expected expiry around %v, got %v", expectedExpiry, cred.ExpiresAt)
	}
}

func TestGenerateTempCredentialMaxTTL(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewCredentialManager(nil, storage)

	// Try to set TTL longer than max
	longTTL := 10 * 24 * time.Hour
	cred, err := manager.GenerateTempCredential(1, "jwt", nil, &longTTL)
	if err != nil {
		t.Fatalf("Failed to generate credential: %v", err)
	}

	// Should be capped at max TTL (7 days)
	maxExpiry := time.Now().Add(7 * 24 * time.Hour)
	if cred.ExpiresAt.After(maxExpiry.Add(time.Minute)) {
		t.Errorf("Expected expiry capped at max TTL, got %v", cred.ExpiresAt)
	}
}

func TestValidateCredential(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewCredentialManager(nil, storage)

	cred, err := manager.GenerateTempCredential(1, "jwt", nil, nil)
	if err != nil {
		t.Fatalf("Failed to generate credential: %v", err)
	}

	// Validate the credential
	validatedCred, err := manager.ValidateCredential(cred.Token)
	if err != nil {
		t.Fatalf("Failed to validate credential: %v", err)
	}

	if validatedCred.ID != cred.ID {
		t.Errorf("Expected credential ID %s, got %s", cred.ID, validatedCred.ID)
	}

	if validatedCred.LastUsedAt == nil {
		t.Error("Expected LastUsedAt to be set")
	}
}

func TestValidateCredentialInvalid(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewCredentialManager(nil, storage)

	_, err := manager.ValidateCredential("invalid_token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestValidateCredentialExpired(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewCredentialManager(nil, storage)

	// Create expired credential directly
	cred := &TempCredential{
		ID:        "test_exp",
		UserID:    1,
		Type:      "jwt",
		Token:     "test_token",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	storage.Save(cred)

	_, err := manager.ValidateCredential(cred.Token)
	if err == nil {
		t.Error("Expected error for expired credential")
	}
}

func TestRefreshCredential(t *testing.T) {
	storage := NewMemoryStorage()
	config := &TempCredentialConfig{
		DefaultTTL:       1 * time.Hour,
		RefreshThreshold: 1 * time.Hour, // Set to full TTL so any credential can be refreshed
		AutoRefresh:      true,
	}
	manager := NewCredentialManager(config, storage)

	cred, err := manager.GenerateTempCredential(1, "jwt", nil, nil)
	if err != nil {
		t.Fatalf("Failed to generate credential: %v", err)
	}

	// Store original expiry before refresh
	originalExpiry := cred.ExpiresAt

	// Refresh the credential
	refreshedCred, err := manager.RefreshCredential(cred.ID, nil)
	if err != nil {
		t.Fatalf("Failed to refresh credential: %v", err)
	}

	// Should have extended expiry
	if !refreshedCred.ExpiresAt.After(originalExpiry) {
		t.Errorf("Expected refreshed credential to have later expiry. Original: %v, Refreshed: %v", originalExpiry, refreshedCred.ExpiresAt)
	}
}

func TestRefreshCredentialExpired(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewCredentialManager(nil, storage)

	// Create expired credential
	cred := &TempCredential{
		ID:        "test_exp",
		UserID:    1,
		Type:      "jwt",
		Token:     "test_token",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	storage.Save(cred)

	_, err := manager.RefreshCredential(cred.ID, nil)
	if err == nil {
		t.Error("Expected error for refreshing expired credential")
	}
}

func TestRevokeCredential(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewCredentialManager(nil, storage)

	cred, err := manager.GenerateTempCredential(1, "jwt", nil, nil)
	if err != nil {
		t.Fatalf("Failed to generate credential: %v", err)
	}

	// Revoke the credential
	err = manager.RevokeCredential(cred.ID)
	if err != nil {
		t.Fatalf("Failed to revoke credential: %v", err)
	}

	// Should not be able to validate
	_, err = manager.ValidateCredential(cred.Token)
	if err == nil {
		t.Error("Expected error for revoked credential")
	}
}

func TestRevokeUserCredentials(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewCredentialManager(nil, storage)

	// Create multiple credentials for user 1
	_, err := manager.GenerateTempCredential(1, "jwt", nil, nil)
	if err != nil {
		t.Fatalf("Failed to generate credential: %v", err)
	}
	_, err = manager.GenerateTempCredential(1, "api_key", nil, nil)
	if err != nil {
		t.Fatalf("Failed to generate credential: %v", err)
	}

	// Create credential for user 2
	_, err = manager.GenerateTempCredential(2, "jwt", nil, nil)
	if err != nil {
		t.Fatalf("Failed to generate credential: %v", err)
	}

	// Revoke all credentials for user 1
	err = manager.RevokeUserCredentials(1)
	if err != nil {
		t.Fatalf("Failed to revoke user credentials: %v", err)
	}

	// User 1's credentials should be gone
	creds, _ := manager.ListUserCredentials(1, 100)
	if len(creds) != 0 {
		t.Errorf("Expected 0 credentials for user 1, got %d", len(creds))
	}

	// User 2's credentials should still exist
	creds, _ = manager.ListUserCredentials(2, 100)
	if len(creds) != 1 {
		t.Errorf("Expected 1 credential for user 2, got %d", len(creds))
	}
}

func TestCleanupExpired(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewCredentialManager(nil, storage)

	// Create expired credential
	cred := &TempCredential{
		ID:        "test_exp",
		UserID:    1,
		Type:      "jwt",
		Token:     "test_token",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	storage.Save(cred)

	// Create valid credential
	_, err := manager.GenerateTempCredential(1, "jwt", nil, nil)
	if err != nil {
		t.Fatalf("Failed to generate credential: %v", err)
	}

	// Cleanup expired credentials
	count, err := manager.CleanupExpired()
	if err != nil {
		t.Fatalf("Failed to cleanup expired credentials: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected to cleanup 1 credential, got %d", count)
	}
}

func TestListUserCredentials(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewCredentialManager(nil, storage)

	// Create credentials for user 1
	cred1, _ := manager.GenerateTempCredential(1, "jwt", nil, nil)
	cred2, _ := manager.GenerateTempCredential(1, "api_key", nil, nil)

	// Create credential for user 2
	_, _ = manager.GenerateTempCredential(2, "jwt", nil, nil)

	// List credentials for user 1
	creds, err := manager.ListUserCredentials(1, 10)
	if err != nil {
		t.Fatalf("Failed to list credentials: %v", err)
	}

	if len(creds) != 2 {
		t.Errorf("Expected 2 credentials, got %d", len(creds))
	}

	// Check that we got the right credentials
	found1 := false
	found2 := false
	for _, c := range creds {
		if c.ID == cred1.ID {
			found1 = true
		}
		if c.ID == cred2.ID {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Error("Expected to find both credentials")
	}
}

func TestGetCredential(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewCredentialManager(nil, storage)

	cred, err := manager.GenerateTempCredential(1, "jwt", nil, nil)
	if err != nil {
		t.Fatalf("Failed to generate credential: %v", err)
	}

	// Get the credential
	retrieved, err := manager.GetCredential(cred.ID)
	if err != nil {
		t.Fatalf("Failed to get credential: %v", err)
	}

	if retrieved.ID != cred.ID {
		t.Errorf("Expected credential ID %s, got %s", cred.ID, retrieved.ID)
	}
}

func TestMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage()

	// Test Save
	cred := &TempCredential{
		ID:        "test",
		UserID:    1,
		Type:      "jwt",
		Token:     "token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	err := storage.Save(cred)
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Test Get
	retrieved, err := storage.Get("test")
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}

	if retrieved.ID != "test" {
		t.Errorf("Expected ID 'test', got %s", retrieved.ID)
	}

	// Test Delete
	err = storage.Delete("test")
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	_, err = storage.Get("test")
	if err == nil {
		t.Error("Expected error for deleted credential")
	}
}

func TestHashToken(t *testing.T) {
	token := "test_token_123"
	hash1 := hashToken(token)
	hash2 := hashToken(token)

	if hash1 != hash2 {
		t.Error("Expected same hash for same token")
	}

	if hash1 == "" {
		t.Error("Expected non-empty hash")
	}

	// Different tokens should have different hashes
	hash3 := hashToken("different_token")
	if hash1 == hash3 {
		t.Error("Expected different hashes for different tokens")
	}
}
