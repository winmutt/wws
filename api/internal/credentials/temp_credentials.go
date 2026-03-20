package credentials

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// TempCredential represents a temporary credential
type TempCredential struct {
	ID          string     `json:"id"`
	UserID      int        `json:"user_id"`
	WorkspaceID *int       `json:"workspace_id,omitempty"`
	Type        string     `json:"type"` // "jwt", "api_key", "oauth"
	Token       string     `json:"-"`
	ExpiresAt   time.Time  `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	Metadata    string     `json:"metadata,omitempty"`
}

// TempCredentialConfig represents configuration for temporary credentials
type TempCredentialConfig struct {
	DefaultTTL        time.Duration `json:"default_ttl"`
	MaxTTL            time.Duration `json:"max_ttl"`
	RefreshThreshold  time.Duration `json:"refresh_threshold"`
	AutoRefresh       bool          `json:"auto_refresh"`
	CleanupInterval   time.Duration `json:"cleanup_interval"`
	EncryptionKeyPath string        `json:"encryption_key_path"`
}

// DefaultConfig returns default credential configuration
func DefaultConfig() *TempCredentialConfig {
	return &TempCredentialConfig{
		DefaultTTL:       24 * time.Hour,
		MaxTTL:           7 * 24 * time.Hour,
		RefreshThreshold: 1 * time.Hour,
		AutoRefresh:      true,
		CleanupInterval:  1 * time.Hour,
	}
}

// CredentialManager manages temporary credentials
type CredentialManager struct {
	config  *TempCredentialConfig
	storage CredentialStorage
}

// CredentialStorage interface for credential persistence
type CredentialStorage interface {
	Save(cred *TempCredential) error
	Get(id string) (*TempCredential, error)
	Delete(id string) error
	List(userID int, limit int) ([]*TempCredential, error)
	DeleteExpired() (int, error)
}

// MemoryStorage implements CredentialStorage using in-memory storage
type MemoryStorage struct {
	credentials map[string]*TempCredential
}

// NewMemoryStorage creates a new in-memory credential storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		credentials: make(map[string]*TempCredential),
	}
}

// Save stores a credential
func (s *MemoryStorage) Save(cred *TempCredential) error {
	s.credentials[cred.ID] = cred
	return nil
}

// Get retrieves a credential by ID
func (s *MemoryStorage) Get(id string) (*TempCredential, error) {
	cred, ok := s.credentials[id]
	if !ok {
		return nil, errors.New("credential not found")
	}
	return cred, nil
}

// Delete removes a credential
func (s *MemoryStorage) Delete(id string) error {
	delete(s.credentials, id)
	return nil
}

// List returns credentials for a user
func (s *MemoryStorage) List(userID int, limit int) ([]*TempCredential, error) {
	var result []*TempCredential
	for _, cred := range s.credentials {
		if cred.UserID == userID {
			result = append(result, cred)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// DeleteExpired removes expired credentials
func (s *MemoryStorage) DeleteExpired() (int, error) {
	count := 0
	now := time.Now()
	for id, cred := range s.credentials {
		if cred.ExpiresAt.Before(now) {
			delete(s.credentials, id)
			count++
		}
	}
	return count, nil
}

// NewCredentialManager creates a new credential manager
func NewCredentialManager(config *TempCredentialConfig, storage CredentialStorage) *CredentialManager {
	if config == nil {
		config = DefaultConfig()
	}
	return &CredentialManager{
		config:  config,
		storage: storage,
	}
}

// GenerateTempCredential creates a new temporary credential
func (cm *CredentialManager) GenerateTempCredential(userID int, credType string, workspaceID *int, ttl *time.Duration) (*TempCredential, error) {
	// Validate credential type
	validTypes := []string{"jwt", "api_key", "oauth"}
	typeValid := false
	for _, t := range validTypes {
		if credType == t {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return nil, fmt.Errorf("invalid credential type: %s", credType)
	}

	// Determine TTL
	if ttl == nil {
		ttl = &cm.config.DefaultTTL
	}
	if *ttl > cm.config.MaxTTL {
		ttl = &cm.config.MaxTTL
	}
	if *ttl <= 0 {
		ttl = &cm.config.DefaultTTL
	}

	// Generate credential ID
	credID, err := generateSecureID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate credential ID: %w", err)
	}

	// Generate token
	token, err := generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	now := time.Now()
	cred := &TempCredential{
		ID:          credID,
		UserID:      userID,
		WorkspaceID: workspaceID,
		Type:        credType,
		Token:       token,
		ExpiresAt:   now.Add(*ttl),
		CreatedAt:   now,
	}

	// Save to storage
	if err := cm.storage.Save(cred); err != nil {
		return nil, fmt.Errorf("failed to save credential: %w", err)
	}

	return cred, nil
}

// ValidateCredential validates a credential token
func (cm *CredentialManager) ValidateCredential(token string) (*TempCredential, error) {
	// Hash the token to find the credential
	tokenHash := hashToken(token)

	// Search for credential with matching token hash
	// This is a simplified implementation - in production, you'd use proper database queries
	// We need to iterate through all stored credentials
	// Since MemoryStorage doesn't support listing all, we'll check by trying different user IDs
	// In production, you'd use a proper database query

	// For memory storage, iterate through the internal map
	if memStorage, ok := cm.storage.(*MemoryStorage); ok {
		for _, cred := range memStorage.credentials {
			if hashToken(cred.Token) == tokenHash {
				// Check if expired
				if time.Now().After(cred.ExpiresAt) {
					// Delete expired credential
					cm.storage.Delete(cred.ID)
					return nil, errors.New("credential expired")
				}

				// Update last used
				now := time.Now()
				cred.LastUsedAt = &now
				cm.storage.Save(cred)

				return cred, nil
			}
		}
	}

	return nil, errors.New("invalid credential")
}

// RefreshCredential extends a credential's expiry
func (cm *CredentialManager) RefreshCredential(credID string, extendBy *time.Duration) (*TempCredential, error) {
	cred, err := cm.storage.Get(credID)
	if err != nil {
		return nil, err
	}

	// Check if already expired
	if time.Now().After(cred.ExpiresAt) {
		return nil, errors.New("credential already expired")
	}

	// Check refresh threshold
	timeUntilExpiry := time.Until(cred.ExpiresAt)
	if timeUntilExpiry > cm.config.RefreshThreshold && !cm.config.AutoRefresh {
		return nil, errors.New("credential not due for refresh yet")
	}

	// Determine extension
	if extendBy == nil {
		extendBy = &cm.config.DefaultTTL
	}

	// Extend expiry from current expiry time
	cred.ExpiresAt = cred.ExpiresAt.Add(*extendBy)

	// Save updated credential
	if err := cm.storage.Save(cred); err != nil {
		return nil, fmt.Errorf("failed to save refreshed credential: %w", err)
	}

	return cred, nil
}

// RevokeCredential revokes a credential immediately
func (cm *CredentialManager) RevokeCredential(credID string) error {
	return cm.storage.Delete(credID)
}

// RevokeUserCredentials revokes all credentials for a user
func (cm *CredentialManager) RevokeUserCredentials(userID int) error {
	creds, err := cm.storage.List(userID, 1000)
	if err != nil {
		return err
	}

	for _, cred := range creds {
		if err := cm.storage.Delete(cred.ID); err != nil {
			return fmt.Errorf("failed to revoke credential %s: %w", cred.ID, err)
		}
	}

	return nil
}

// CleanupExpired removes all expired credentials
func (cm *CredentialManager) CleanupExpired() (int, error) {
	return cm.storage.DeleteExpired()
}

// GetCredential returns a credential by ID
func (cm *CredentialManager) GetCredential(credID string) (*TempCredential, error) {
	return cm.storage.Get(credID)
}

// ListUserCredentials lists credentials for a user
func (cm *CredentialManager) ListUserCredentials(userID int, limit int) ([]*TempCredential, error) {
	return cm.storage.List(userID, limit)
}

// Config returns the current configuration
func (cm *CredentialManager) Config() *TempCredentialConfig {
	return cm.config
}

// generateSecureID generates a secure credential ID
func generateSecureID() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return fmt.Sprintf("cred_%s", base64.URLEncoding.EncodeToString(b)[:22]), nil
}

// generateSecureToken generates a secure token
func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}

	prefix := "wws_temp_"
	return prefix + base64.URLEncoding.EncodeToString(b), nil
}

// hashToken hashes a token for storage/comparison
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(hash[:])
}

// LoadConfig loads configuration from environment or file
func LoadConfig() (*TempCredentialConfig, error) {
	config := DefaultConfig()

	// Check for environment variables
	if ttl := os.Getenv("TEMP_CRED_TTL"); ttl != "" {
		if d, err := time.ParseDuration(ttl); err == nil {
			config.DefaultTTL = d
		}
	}

	if maxTTL := os.Getenv("TEMP_CRED_MAX_TTL"); maxTTL != "" {
		if d, err := time.ParseDuration(maxTTL); err == nil {
			config.MaxTTL = d
		}
	}

	if autoRefresh := os.Getenv("TEMP_CRED_AUTO_REFRESH"); autoRefresh != "" {
		config.AutoRefresh = strings.ToLower(autoRefresh) == "true"
	}

	// Check for config file
	if configFile := os.Getenv("TEMP_CRED_CONFIG_FILE"); configFile != "" {
		data, err := os.ReadFile(configFile)
		if err == nil {
			var loadedConfig TempCredentialConfig
			if err := json.Unmarshal(data, &loadedConfig); err == nil {
				if loadedConfig.DefaultTTL > 0 {
					config.DefaultTTL = loadedConfig.DefaultTTL
				}
				if loadedConfig.MaxTTL > 0 {
					config.MaxTTL = loadedConfig.MaxTTL
				}
				if loadedConfig.RefreshThreshold > 0 {
					config.RefreshThreshold = loadedConfig.RefreshThreshold
				}
				config.AutoRefresh = loadedConfig.AutoRefresh
				config.CleanupInterval = loadedConfig.CleanupInterval
			}
		}
	}

	return config, nil
}
