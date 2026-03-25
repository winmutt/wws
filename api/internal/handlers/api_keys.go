package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"wws/api/internal/db"
	"wws/api/internal/models"
)

// APIKeyHandler handles API key operations
type APIKeyHandler struct {
	DB *sql.DB
}

// GenerateAPIKey generates a new API key
func GenerateAPIKey() (string, error) {
	key := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, key)
	if err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}

	// Prefix with "wws_" for identification
	prefix := "wws_"
	return prefix + base64.URLEncoding.EncodeToString(key), nil
}

// hashKey hashes an API key for storage
func hashKey(key string) string {
	// Use crypto package for consistent hashing
	hash := sha256.Sum256([]byte(key))
	return base64.URLEncoding.EncodeToString(hash[:])
}

// getKeyPrefix returns the first 8 characters of the key for identification
func getKeyPrefix(key string) string {
	if len(key) > 8 {
		return key[:8]
	}
	return key
}

// CreateAPIKey creates a new API key for a user
func (h *APIKeyHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)
	if userID == 0 {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("Authentication required"))
		return
	}

	var req models.APIKeyCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid request body: %v", err))
		return
	}

	// Validate name
	if req.Name == "" {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Name is required"))
		return
	}

	// Validate permissions
	if req.Permissions == "" {
		req.Permissions = "read"
	}
	validPermissions := []string{"read", "write", "admin"}
	permissionsValid := false
	for _, perm := range validPermissions {
		if req.Permissions == perm {
			permissionsValid = true
			break
		}
	}
	if !permissionsValid {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid permissions. Must be one of: read, write, admin"))
		return
	}

	// Generate new API key
	apiKey, err := GenerateAPIKey()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to generate API key: %v", err))
		return
	}

	// Calculate expiry if specified
	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		expiry := time.Now().Add(time.Duration(*req.ExpiresIn) * time.Hour)
		expiresAt = &expiry
	}

	// Hash the key for storage
	keyHash := hashKey(apiKey)
	keyPrefix := getKeyPrefix(apiKey)

	// Insert into database
	now := time.Now()
	result, err := h.DB.Exec(`
		INSERT INTO api_keys (user_id, name, key_hash, key_prefix, permissions, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, req.Name, keyHash, keyPrefix, req.Permissions, expiresAt, now)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to create API key: %v", err))
		return
	}

	keyID, _ := result.LastInsertId()

	// Return the full API key (only time it's shown)
	response := models.APIKeyResponse{
		ID:          int(keyID),
		UserID:      userID,
		Name:        req.Name,
		KeyPrefix:   keyPrefix,
		Permissions: req.Permissions,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		NewKey:      &apiKey,
	}

	WriteJSON(w, http.StatusCreated, response)
}

// ListAPIKeys lists all API keys for a user
func (h *APIKeyHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)
	if userID == 0 {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("Authentication required"))
		return
	}

	rows, err := h.DB.Query(`
		SELECT id, user_id, name, key_prefix, permissions, expires_at, created_at, last_used_at
		FROM api_keys WHERE user_id = ? ORDER BY created_at DESC`,
		userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to list API keys: %v", err))
		return
	}
	defer rows.Close()

	var apiKeys []models.APIKeyResponse
	for rows.Next() {
		var key models.APIKeyResponse
		var expiresAt, lastUsedAt sql.NullTime
		err := rows.Scan(&key.ID, &key.UserID, &key.Name, &key.KeyPrefix, &key.Permissions, &expiresAt, &key.CreatedAt, &lastUsedAt)
		if err != nil {
			continue
		}
		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Time
		}
		if lastUsedAt.Valid {
			key.LastUsedAt = &lastUsedAt.Time
		}
		apiKeys = append(apiKeys, key)
	}

	response := models.APIKeyListResponse{
		APIKeys: apiKeys,
		Total:   len(apiKeys),
	}

	WriteJSON(w, http.StatusOK, response)
}

// DeleteAPIKey deletes an API key
func (h *APIKeyHandler) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)
	if userID == 0 {
		WriteError(w, http.StatusUnauthorized, fmt.Errorf("Authentication required"))
		return
	}

	// Extract key ID from URL
	vars := mux.Vars(r)
	keyIDStr := vars["id"]

	var keyID int
	_, err := fmt.Sscanf(keyIDStr, "%d", &keyID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid key ID"))
		return
	}

	result, err := h.DB.Exec("DELETE FROM api_keys WHERE id = ? AND user_id = ?", keyID, userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to delete API key: %v", err))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		WriteError(w, http.StatusNotFound, fmt.Errorf("API key not found"))
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "API key deleted"})
}

// ValidateAPIKey validates an API key and returns user info
func (h *APIKeyHandler) ValidateAPIKey(apiKey string) (*models.User, error) {
	// Hash the provided key
	keyHash := hashKey(apiKey)

	// Check if key exists and is not expired
	var userID int
	var keyExpiresAt sql.NullTime
	err := h.DB.QueryRow(`
		SELECT user_id, expires_at FROM api_keys WHERE key_hash = ?`,
		keyHash).Scan(&userID, &keyExpiresAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid API key")
	}
	if err != nil {
		return nil, err
	}

	// Check if key is expired
	if keyExpiresAt.Valid && keyExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("API key expired")
	}

	// Update last used timestamp
	_, _ = h.DB.Exec("UPDATE api_keys SET last_used_at = ? WHERE key_hash = ?", time.Now(), keyHash)

	// Get user info
	var user models.User
	err = h.DB.QueryRow(`
		SELECT id, github_id, username, email, created_at FROM users WHERE id = ?`,
		userID).Scan(&user.ID, &user.GithubID, &user.Username, &user.Email, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// APIKeyMiddleware validates API keys from requests
func APIKeyMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip for certain paths
		if strings.HasPrefix(r.URL.Path, "/health") ||
			strings.HasPrefix(r.URL.Path, "/api/v1/auth") ||
			strings.HasPrefix(r.URL.Path, "/api/v1/sessions") {
			h.ServeHTTP(w, r)
			return
		}

		// Get API key from header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			// Try Authorization header with Bearer
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				apiKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if apiKey == "" {
			// No API key, let other middleware handle auth
			h.ServeHTTP(w, r)
			return
		}

		// Validate API key
		keyHandler := &APIKeyHandler{DB: db.DB}
		user, err := keyHandler.ValidateAPIKey(apiKey)
		if err != nil {
			WriteError(w, http.StatusUnauthorized, fmt.Errorf("Invalid or expired API key"))
			return
		}

		// Set user in context
		ctx := context.WithValue(r.Context(), "user", user)
		ctx = context.WithValue(ctx, "user_id", user.ID)
		ctx = context.WithValue(ctx, "auth_type", "api_key")

		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getUserIDFromRequest extracts user ID from request context
func getUserIDFromRequest(r *http.Request) int {
	if userID, ok := r.Context().Value("user_id").(int); ok {
		return userID
	}
	return 0
}

// getUsernameFromRequest extracts username from request context
func getUsernameFromRequest(r *http.Request) string {
	if username, ok := r.Context().Value("username").(string); ok {
		return username
	}
	return ""
}
