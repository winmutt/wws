package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"wws/api/internal/models"
)

func setupTestDBWithAPIKeys(t *testing.T) *sql.DB {
	// Create temporary test database
	tmpDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create tables
	createTestTablesForAPIKeys(tmpDB)

	return tmpDB
}

func createTestTablesForAPIKeys(db *sql.DB) {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			github_id TEXT UNIQUE NOT NULL,
			username TEXT NOT NULL,
			email TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			key_hash TEXT NOT NULL UNIQUE,
			key_prefix TEXT NOT NULL,
			permissions TEXT NOT NULL DEFAULT 'read',
			expires_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_used_at DATETIME,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
		CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys(key_prefix);
		CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at);`,
	}

	for _, table := range tables {
		_, err := db.Exec(table)
		if err != nil {
			panic(err)
		}
	}
}

func TestAPIKeyHandlerCreateAPIKey(t *testing.T) {
	testDB := setupTestDBWithAPIKeys(t)
	defer testDB.Close()

	// Create test user
	_, err := testDB.Exec("INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"123", "testuser", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	handler := &APIKeyHandler{DB: testDB}

	// Test creating API key
	createData := models.APIKeyCreateRequest{
		Name:        "Test API Key",
		Permissions: "write",
		ExpiresIn:   intPtr(24), // 24 hours
	}

	jsonData, _ := json.Marshal(createData)
	req := httptest.NewRequest("POST", "/api/v1/api-keys", bytes.NewBuffer(jsonData))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", 1))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateAPIKey(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var response models.APIKeyResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Name != "Test API Key" {
		t.Errorf("Expected name 'Test API Key', got %s", response.Name)
	}
	if response.Permissions != "write" {
		t.Errorf("Expected permissions 'write', got %s", response.Permissions)
	}
	if response.NewKey == nil || *response.NewKey == "" {
		t.Error("Expected new_key to be returned on creation")
	}
	if response.UserID != 1 {
		t.Errorf("Expected user_id 1, got %d", response.UserID)
	}
}

func TestAPIKeyHandlerCreateAPIKeyNoExpiry(t *testing.T) {
	testDB := setupTestDBWithAPIKeys(t)
	defer testDB.Close()

	// Create test user
	_, err := testDB.Exec("INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"123", "testuser", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	handler := &APIKeyHandler{DB: testDB}

	// Test creating API key without expiry
	createData := models.APIKeyCreateRequest{
		Name:        "Permanent API Key",
		Permissions: "read",
	}

	jsonData, _ := json.Marshal(createData)
	req := httptest.NewRequest("POST", "/api/v1/api-keys", bytes.NewBuffer(jsonData))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", 1))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateAPIKey(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response models.APIKeyResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.ExpiresAt != nil {
		t.Error("Expected expires_at to be nil when not specified")
	}
}

func TestAPIKeyHandlerCreateAPIKeyUnauthorized(t *testing.T) {
	testDB := setupTestDBWithAPIKeys(t)
	defer testDB.Close()

	handler := &APIKeyHandler{DB: testDB}

	createData := models.APIKeyCreateRequest{
		Name:        "Test API Key",
		Permissions: "read",
	}

	jsonData, _ := json.Marshal(createData)
	req := httptest.NewRequest("POST", "/api/v1/api-keys", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateAPIKey(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAPIKeyHandlerCreateAPIKeyInvalidPermissions(t *testing.T) {
	testDB := setupTestDBWithAPIKeys(t)
	defer testDB.Close()

	// Create test user
	_, err := testDB.Exec("INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"123", "testuser", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	handler := &APIKeyHandler{DB: testDB}

	createData := models.APIKeyCreateRequest{
		Name:        "Test API Key",
		Permissions: "invalid",
	}

	jsonData, _ := json.Marshal(createData)
	req := httptest.NewRequest("POST", "/api/v1/api-keys", bytes.NewBuffer(jsonData))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", 1))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateAPIKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAPIKeyHandlerListAPIKeys(t *testing.T) {
	testDB := setupTestDBWithAPIKeys(t)
	defer testDB.Close()

	// Create test user
	_, err := testDB.Exec("INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"123", "testuser", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create test API keys
	key1, _ := GenerateAPIKey()
	key2, _ := GenerateAPIKey()
	hash1 := hashKey(key1)
	hash2 := hashKey(key2)
	prefix1 := getKeyPrefix(key1)
	prefix2 := getKeyPrefix(key2)

	_, err = testDB.Exec("INSERT INTO api_keys (user_id, name, key_hash, key_prefix, permissions) VALUES (?, ?, ?, ?, ?)",
		1, "Key 1", hash1, prefix1, "read")
	if err != nil {
		t.Fatalf("Failed to create test API key 1: %v", err)
	}
	_, err = testDB.Exec("INSERT INTO api_keys (user_id, name, key_hash, key_prefix, permissions) VALUES (?, ?, ?, ?, ?)",
		1, "Key 2", hash2, prefix2, "write")
	if err != nil {
		t.Fatalf("Failed to create test API key 2: %v", err)
	}

	handler := &APIKeyHandler{DB: testDB}

	req := httptest.NewRequest("GET", "/api/v1/api-keys", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", 1))
	w := httptest.NewRecorder()

	handler.ListAPIKeys(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIKeyListResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Total != 2 {
		t.Errorf("Expected 2 API keys, got %d", response.Total)
	}

	if len(response.APIKeys) != 2 {
		t.Errorf("Expected 2 API keys in response, got %d", len(response.APIKeys))
	}
}

func TestAPIKeyHandlerDeleteAPIKey(t *testing.T) {
	testDB := setupTestDBWithAPIKeys(t)
	defer testDB.Close()

	// Create test user
	_, err := testDB.Exec("INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"123", "testuser", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create test API key
	key, _ := GenerateAPIKey()
	hash := hashKey(key)
	prefix := getKeyPrefix(key)

	_, err = testDB.Exec("INSERT INTO api_keys (user_id, name, key_hash, key_prefix, permissions) VALUES (?, ?, ?, ?, ?)",
		1, "Test Key", hash, prefix, "read")
	if err != nil {
		t.Fatalf("Failed to create test API key: %v", err)
	}

	handler := &APIKeyHandler{DB: testDB}

	// Set up mux router to properly extract URL variables
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/api-keys/{id}", func(w http.ResponseWriter, r *http.Request) {
		handler.DeleteAPIKey(w, r)
	})

	req := httptest.NewRequest("DELETE", "/api/v1/api-keys/1", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", 1))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
		t.Logf("Response: %s", w.Body.String())
	}

	// Verify key was deleted
	var count int
	err = testDB.QueryRow("SELECT COUNT(*) FROM api_keys WHERE id = ?", 1).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count API keys: %v", err)
	}
	if count != 0 {
		t.Error("Expected API key to be deleted")
	}
}

func TestAPIKeyHandlerDeleteAPIKeyNotFound(t *testing.T) {
	testDB := setupTestDBWithAPIKeys(t)
	defer testDB.Close()

	// Create test user
	_, err := testDB.Exec("INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"123", "testuser", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	handler := &APIKeyHandler{DB: testDB}

	// Set up mux router to properly extract URL variables
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/api-keys/{id}", func(w http.ResponseWriter, r *http.Request) {
		handler.DeleteAPIKey(w, r)
	})

	req := httptest.NewRequest("DELETE", "/api/v1/api-keys/999", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", 1))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
		t.Logf("Response: %s", w.Body.String())
	}
}

func TestAPIKeyHandlerValidateAPIKey(t *testing.T) {
	testDB := setupTestDBWithAPIKeys(t)
	defer testDB.Close()

	// Create test user
	_, err := testDB.Exec("INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"123", "testuser", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create test API key
	key, _ := GenerateAPIKey()
	hash := hashKey(key)
	prefix := getKeyPrefix(key)

	_, err = testDB.Exec("INSERT INTO api_keys (user_id, name, key_hash, key_prefix, permissions) VALUES (?, ?, ?, ?, ?)",
		1, "Test Key", hash, prefix, "read")
	if err != nil {
		t.Fatalf("Failed to create test API key: %v", err)
	}

	handler := &APIKeyHandler{DB: testDB}

	// Test valid key
	user, err := handler.ValidateAPIKey(key)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if user == nil {
		t.Error("Expected user to be returned")
	}
	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", user.Username)
	}

	// Test invalid key
	_, err = handler.ValidateAPIKey("invalid_key")
	if err == nil {
		t.Error("Expected error for invalid key")
	}
}

func TestAPIKeyHandlerValidateExpiredAPIKey(t *testing.T) {
	testDB := setupTestDBWithAPIKeys(t)
	defer testDB.Close()

	// Create test user
	_, err := testDB.Exec("INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"123", "testuser", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create expired API key
	key, _ := GenerateAPIKey()
	hash := hashKey(key)
	prefix := getKeyPrefix(key)
	expiresAt := time.Now().Add(-24 * time.Hour) // Expired yesterday

	_, err = testDB.Exec("INSERT INTO api_keys (user_id, name, key_hash, key_prefix, permissions, expires_at) VALUES (?, ?, ?, ?, ?, ?)",
		1, "Expired Key", hash, prefix, "read", expiresAt)
	if err != nil {
		t.Fatalf("Failed to create expired API key: %v", err)
	}

	handler := &APIKeyHandler{DB: testDB}

	// Test expired key
	_, err = handler.ValidateAPIKey(key)
	if err == nil {
		t.Error("Expected error for expired key")
	}
}

func TestGenerateAPIKey(t *testing.T) {
	// Test key generation
	key1, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	key2, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	// Keys should be unique
	if key1 == key2 {
		t.Error("Expected unique API keys")
	}

	// Keys should have prefix
	if len(key1) <= 4 {
		t.Error("Expected key to have prefix")
	}
	if key1[:4] != "wws_" {
		t.Errorf("Expected key prefix 'wws_', got %s", key1[:4])
	}
}

func TestHashKey(t *testing.T) {
	key := "test_api_key_123"
	hash := hashKey(key)

	if hash == "" {
		t.Error("Expected non-empty hash")
	}

	// Same key should produce same hash
	hash2 := hashKey(key)
	if hash != hash2 {
		t.Error("Expected same hash for same key")
	}
}

func TestGetKeyPrefix(t *testing.T) {
	longKey := "wws_abcdefghijklmnopqrstuvwxyz123456"
	prefix := getKeyPrefix(longKey)

	if prefix != longKey[:8] {
		t.Errorf("Expected prefix %s, got %s", longKey[:8], prefix)
	}

	shortKey := "wws_abc"
	prefix2 := getKeyPrefix(shortKey)
	if prefix2 != shortKey {
		t.Errorf("Expected prefix %s, got %s", shortKey, prefix2)
	}
}

// Helper function
func intPtr(i int) *int {
	return &i
}
