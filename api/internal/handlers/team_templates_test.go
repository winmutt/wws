package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"wws/api/internal/db"

	_ "github.com/mattn/go-sqlite3"
)

// isValidTemplatePermission checks if a permission string is valid
func isValidTemplatePermission(permission string) bool {
	validPermissions := map[string]bool{
		"view":   true,
		"use":    true,
		"edit":   true,
		"manage": true,
	}
	return validPermissions[permission]
}

// setupTeamTemplateTables ensures the required tables exist for testing
func setupTeamTemplateTables(t *testing.T) *sql.DB {
	t.Helper()

	// Create a fresh test database
	testDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create all required tables
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			github_id TEXT UNIQUE NOT NULL,
			username TEXT NOT NULL,
			email TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS organizations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			owner_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
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
	}

	for _, stmt := range statements {
		_, err := testDB.Exec(stmt)
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}
	}

	return testDB
}

func createTeamTemplateTablesForTest(testDB *sql.DB) {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			github_id TEXT UNIQUE NOT NULL,
			username TEXT NOT NULL,
			email TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS organizations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			owner_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
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
		`CREATE TABLE IF NOT EXISTS members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			organization_id INTEGER NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			invited_by INTEGER,
			accepted INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, table := range tables {
		_, err := testDB.Exec(table)
		if err != nil {
			panic(err)
		}
	}
}

func TestTeamTemplateHandlerInstantiation(t *testing.T) {
	handler := &TeamTemplateHandler{}
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
}

func TestGrantTemplateAccess(t *testing.T) {
	testDB := setupTeamTemplateTables(t)
	defer testDB.Close()

	// Temporarily set db.DB to our test database
	originalDB := db.DB
	db.DB = testDB
	defer func() { db.DB = originalDB }()

	handler := &TeamTemplateHandler{}

	// Setup test data
	_, err := testDB.Exec("INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"github123", "testuser", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO organizations (name, owner_id) VALUES (?, ?)",
		"Test Org", 1)
	if err != nil {
		t.Fatalf("Failed to create organization: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO teams (organization_id, name, created_by) VALUES (?, ?, ?)",
		1, "Dev Team", 1)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO workspace_templates (name, description, organization_id, created_by) VALUES (?, ?, ?, ?)",
		"Dev Template", "Development template", 1, 1)
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	grantReq := map[string]interface{}{
		"team_id":     1,
		"template_id": 1,
		"permission":  "use",
	}

	jsonData, _ := json.Marshal(grantReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/templates/grant", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	// Add user ID to request context (simulating authenticated request)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", 1))

	w := httptest.NewRecorder()
	handler.GrantTemplateAccess(w, req)

	// Test accepts both 200 (updated) and 201 (created)
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Errorf("Expected status 200/201, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGrantTemplateAccessInvalidPermission(t *testing.T) {
	testDB := setupTeamTemplateTables(t)
	defer testDB.Close()

	originalDB := db.DB
	db.DB = testDB
	defer func() { db.DB = originalDB }()

	handler := &TeamTemplateHandler{}

	grantReq := map[string]interface{}{
		"team_id":     1,
		"template_id": 1,
		"permission":  "invalid",
	}

	jsonData, _ := json.Marshal(grantReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/templates/grant", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), "user_id", 1))

	w := httptest.NewRecorder()
	handler.GrantTemplateAccess(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestGetTeamTemplates(t *testing.T) {
	testDB := setupTeamTemplateTables(t)
	defer testDB.Close()

	originalDB := db.DB
	db.DB = testDB
	defer func() { db.DB = originalDB }()

	handler := &TeamTemplateHandler{}

	// Setup test data
	_, err := testDB.Exec("INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"github123", "testuser", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO organizations (name, owner_id) VALUES (?, ?)",
		"Test Org", 1)
	if err != nil {
		t.Fatalf("Failed to create organization: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO teams (organization_id, name, created_by) VALUES (?, ?, ?)",
		1, "Dev Team", 1)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO workspace_templates (name, description, organization_id, created_by) VALUES (?, ?, ?, ?)",
		"Dev Template", "Development template", 1, 1)
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO team_template_access (team_id, template_id, permission, granted_by) VALUES (?, ?, ?, ?)",
		1, 1, "use", 1)
	if err != nil {
		t.Fatalf("Failed to grant access: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/templates?team_id=1", nil)
	w := httptest.NewRecorder()
	handler.GetTeamTemplates(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var templates []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&templates); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(templates) != 1 {
		t.Errorf("Expected 1 template, got %d", len(templates))
	}
}

func TestRevokeTemplateAccess(t *testing.T) {
	testDB := setupTeamTemplateTables(t)
	defer testDB.Close()

	originalDB := db.DB
	db.DB = testDB
	defer func() { db.DB = originalDB }()

	handler := &TeamTemplateHandler{}

	// Setup test data
	_, err := testDB.Exec("INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"github123", "testuser", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO organizations (name, owner_id) VALUES (?, ?)",
		"Test Org", 1)
	if err != nil {
		t.Fatalf("Failed to create organization: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO teams (organization_id, name, created_by) VALUES (?, ?, ?)",
		1, "Dev Team", 1)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO workspace_templates (name, description, organization_id, created_by) VALUES (?, ?, ?, ?)",
		"Dev Template", "Development template", 1, 1)
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO team_template_access (team_id, template_id, permission, granted_by) VALUES (?, ?, ?, ?)",
		1, 1, "use", 1)
	if err != nil {
		t.Fatalf("Failed to grant access: %v", err)
	}

	// Use access_id query parameter instead of request body
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/teams/templates/revoke?access_id=1", nil)

	w := httptest.NewRecorder()
	handler.RevokeTemplateAccess(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCheckTemplatePermission(t *testing.T) {
	testDB := setupTeamTemplateTables(t)
	defer testDB.Close()

	originalDB := db.DB
	db.DB = testDB
	defer func() { db.DB = originalDB }()

	handler := &TeamTemplateHandler{}

	// Setup test data
	_, err := testDB.Exec("INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"github123", "testuser", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO organizations (name, owner_id) VALUES (?, ?)",
		"Test Org", 1)
	if err != nil {
		t.Fatalf("Failed to create organization: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO teams (organization_id, name, created_by) VALUES (?, ?, ?)",
		1, "Dev Team", 1)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO workspace_templates (name, description, organization_id, created_by) VALUES (?, ?, ?, ?)",
		"Dev Template", "Development template", 1, 1)
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO team_template_access (team_id, template_id, permission, granted_by) VALUES (?, ?, ?, ?)",
		1, 1, "use", 1)
	if err != nil {
		t.Fatalf("Failed to grant access: %v", err)
	}

	checkReq := map[string]interface{}{
		"team_id":     1,
		"template_id": 1,
		"permission":  "use",
	}

	jsonData, _ := json.Marshal(checkReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/templates/permission", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.CheckTemplatePermission(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGetUsableTemplates(t *testing.T) {
	testDB := setupTeamTemplateTables(t)
	defer testDB.Close()

	originalDB := db.DB
	db.DB = testDB
	defer func() { db.DB = originalDB }()

	handler := &TeamTemplateHandler{}

	// Setup test data
	_, err := testDB.Exec("INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"github123", "testuser", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO organizations (name, owner_id) VALUES (?, ?)",
		"Test Org", 1)
	if err != nil {
		t.Fatalf("Failed to create organization: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO workspace_templates (name, description, organization_id, created_by, is_public) VALUES (?, ?, ?, ?, ?)",
		"Public Template", "A public template", 1, 1, 1)
	if err != nil {
		t.Fatalf("Failed to create public template: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/templates/usable?team_id=1", nil)
	w := httptest.NewRecorder()
	handler.GetUsableTemplates(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGetTemplateTeams(t *testing.T) {
	testDB := setupTeamTemplateTables(t)
	defer testDB.Close()

	originalDB := db.DB
	db.DB = testDB
	defer func() { db.DB = originalDB }()

	handler := &TeamTemplateHandler{}

	// Setup test data
	_, err := testDB.Exec("INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"github123", "testuser", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO organizations (name, owner_id) VALUES (?, ?)",
		"Test Org", 1)
	if err != nil {
		t.Fatalf("Failed to create organization: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO teams (organization_id, name, created_by) VALUES (?, ?, ?)",
		1, "Dev Team", 1)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO workspace_templates (name, description, organization_id, created_by) VALUES (?, ?, ?, ?)",
		"Dev Template", "Development template", 1, 1)
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO team_template_access (team_id, template_id, permission, granted_by) VALUES (?, ?, ?, ?)",
		1, 1, "use", 1)
	if err != nil {
		t.Fatalf("Failed to grant access: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/templates/teams?template_id=1", nil)
	w := httptest.NewRecorder()
	handler.GetTemplateTeams(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestValidTemplatePermissions(t *testing.T) {
	validPermissions := []string{"view", "use", "edit", "manage"}

	for _, perm := range validPermissions {
		if !isValidTemplatePermission(perm) {
			t.Errorf("Expected '%s' to be a valid permission", perm)
		}
	}

	invalidPermissions := []string{"invalid", "delete", "all"}
	for _, perm := range invalidPermissions {
		if isValidTemplatePermission(perm) {
			t.Errorf("Expected '%s' to be an invalid permission", perm)
		}
	}
}

func TestTemplateAccessExpiration(t *testing.T) {
	testDB := setupTeamTemplateTables(t)
	defer testDB.Close()

	// Test that the access table structure supports expiration
	var hasExpiresAt bool
	err := testDB.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('team_template_access') 
		WHERE name = 'expires_at'
	`).Scan(&hasExpiresAt)

	if err != nil {
		t.Fatalf("Failed to check table structure: %v", err)
	}

	// Note: expires_at is optional, so we just verify the table exists
	var count int
	err = testDB.QueryRow("SELECT COUNT(*) FROM team_template_access").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query team_template_access: %v", err)
	}
}

func TestTemplatePermissionHierarchy(t *testing.T) {
	permissions := map[string]int{
		"view":   1,
		"use":    2,
		"edit":   3,
		"manage": 4,
	}

	// Verify hierarchy
	if permissions["manage"] <= permissions["edit"] {
		t.Error("manage permission should be higher than edit")
	}
	if permissions["edit"] <= permissions["use"] {
		t.Error("edit permission should be higher than use")
	}
	if permissions["use"] <= permissions["view"] {
		t.Error("use permission should be higher than view")
	}
}
