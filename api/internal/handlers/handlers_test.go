package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"wws/api/internal/crypto"
	"wws/api/internal/db"

	_ "github.com/mattn/go-sqlite3"
)

var testDB2 *sql.DB

func TestMain(m *testing.M) {
	var err error
	testDB2, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}

	os.Setenv("WWS_ENCRYPTION_KEY", "test-encryption-key-for-tests")
	os.Setenv("WWS_TEST_MODE", "true")
	crypto.InitEncryption()

	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			github_id TEXT UNIQUE NOT NULL,
			username TEXT NOT NULL,
			email TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS oauth_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL UNIQUE,
			access_token TEXT,
			encrypted_access_token TEXT,
			refresh_token TEXT,
			encrypted_refresh_token TEXT,
			expiry DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token TEXT UNIQUE NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS organizations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			owner_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS oauth_states (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			state TEXT UNIQUE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS invitations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			email TEXT NOT NULL,
			token TEXT UNIQUE NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			expires_at DATETIME NOT NULL,
			created_by INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			organization_id INTEGER NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			invited_by INTEGER,
			accepted INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS teams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			created_by INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS workspace_templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			organization_id INTEGER,
			provider TEXT NOT NULL DEFAULT 'podman',
			bootstrap_script TEXT,
			is_public INTEGER DEFAULT 0,
			created_by INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS team_template_access (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_id INTEGER NOT NULL,
			template_id INTEGER NOT NULL,
			permission TEXT NOT NULL DEFAULT 'view',
			granted_by INTEGER NOT NULL,
			granted_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS workspaces (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tag TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			organization_id INTEGER NOT NULL,
			owner_id INTEGER NOT NULL,
			provider TEXT NOT NULL,
			status TEXT NOT NULL,
			config TEXT,
			region TEXT,
			template_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS workspace_languages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id INTEGER NOT NULL,
			language TEXT NOT NULL,
			version TEXT,
			install_script TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS workspace_exports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id INTEGER NOT NULL,
			export_path TEXT NOT NULL,
			export_format TEXT NOT NULL DEFAULT 'json',
			file_size_mb REAL,
			status TEXT NOT NULL DEFAULT 'pending',
			error_message TEXT,
			created_by INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS workspace_imports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			import_path TEXT NOT NULL,
			import_format TEXT NOT NULL DEFAULT 'json',
			status TEXT NOT NULL DEFAULT 'pending',
			error_message TEXT,
			workspace_id INTEGER,
			created_by INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			key_hash TEXT UNIQUE NOT NULL,
			prefix TEXT NOT NULL,
			name TEXT,
			permissions TEXT NOT NULL,
			expires_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS quotas (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			resource_type TEXT NOT NULL,
			quota_limit INTEGER NOT NULL,
			used INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS workspace_sharing (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			role TEXT NOT NULL,
			granted_by INTEGER NOT NULL,
			granted_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			organization_id INTEGER,
			action TEXT NOT NULL,
			resource_type TEXT,
			resource_id INTEGER,
			details TEXT,
			ip_address TEXT,
			user_agent TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, stmt := range statements {
		_, err := testDB2.Exec(stmt)
		if err != nil {
			panic(err)
		}
	}

	InitOAuthStateStore()
	SetOAuthDB(testDB2)
	db.DB = testDB2

	code := m.Run()

	testDB2.Close()
	os.Exit(code)
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	err := HealthHandler(w, req)
	if err != nil {
		t.Errorf("HealthHandler returned error: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestOAuthCallbackHandlerMissingState(t *testing.T) {
	os.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	os.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	os.Setenv("GITHUB_CALLBACK_URL", "http://localhost:8080/oauth/callback")

	req := httptest.NewRequest(http.MethodGet, "/auth/github/callback?code=testcode", nil)
	w := httptest.NewRecorder()

	err := OAuthCallbackHandler(w, req)
	if err == nil {
		t.Error("Expected error for missing state parameter")
	}

	expectedMsg := "missing state parameter"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestOAuthCallbackHandlerMissingCode(t *testing.T) {
	os.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	os.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	os.Setenv("GITHUB_CALLBACK_URL", "http://localhost:8080/oauth/callback")

	InitOAuthStateStore()
	SetOAuthDB(testDB2)
	state, _ := generateStateToken()
	StoreOAuthState(state)

	req := httptest.NewRequest(http.MethodGet, "/auth/github/callback?state="+state, nil)
	w := httptest.NewRecorder()

	err := OAuthCallbackHandler(w, req)
	if err == nil {
		t.Error("Expected error for missing code parameter")
	}

	expectedMsg := "missing authorization code"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestOAuthCallbackHandlerInvalidState(t *testing.T) {
	os.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	os.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	os.Setenv("GITHUB_CALLBACK_URL", "http://localhost:8080/oauth/callback")

	InitOAuthStateStore()
	SetOAuthDB(testDB2)

	req := httptest.NewRequest(http.MethodGet, "/auth/github/callback?code=testcode&state=invalidstate", nil)
	w := httptest.NewRecorder()

	err := OAuthCallbackHandler(w, req)
	if err == nil {
		t.Error("Expected error for invalid state parameter")
	}

	if !strings.Contains(err.Error(), "invalid or expired state") {
		t.Errorf("Expected state validation error, got '%s'", err.Error())
	}
}

func TestOAuthCallbackHandlerValidStateInvalidCode(t *testing.T) {
	os.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	os.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	os.Setenv("GITHUB_CALLBACK_URL", "http://localhost:8080/oauth/callback")

	InitOAuthStateStore()
	SetOAuthDB(testDB2)
	state, _ := generateStateToken()
	StoreOAuthState(state)

	req := httptest.NewRequest(http.MethodGet, "/auth/github/callback?code=invalidcode&state="+state, nil)
	w := httptest.NewRecorder()

	err := OAuthCallbackHandler(w, req)
	if err == nil {
		t.Error("Expected error for invalid code")
	}

	if err.Error() == "" || !strings.Contains(err.Error(), "failed to exchange token") {
		t.Logf("OAuthCallbackHandler returned expected error for invalid token exchange: %v", err)
	}
}
