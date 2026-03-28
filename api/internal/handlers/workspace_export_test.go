package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"wws/api/internal/db"
	"wws/api/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

func setupExportTestDB(t *testing.T) *sql.DB {
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
		`CREATE TABLE IF NOT EXISTS members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			organization_id INTEGER NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			invited_by INTEGER,
			accepted INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS organizations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			owner_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
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
			export_id INTEGER,
			export_path TEXT NOT NULL,
			import_format TEXT NOT NULL DEFAULT 'json',
			organization_id INTEGER NOT NULL,
			imported_by INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			imported_workspace_id INTEGER,
			error_message TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS workspace_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			username TEXT NOT NULL,
			email TEXT,
			role TEXT NOT NULL DEFAULT 'viewer',
			permissions TEXT NOT NULL DEFAULT '{}',
			invited_by INTEGER,
			invited_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			joined_at DATETIME,
			status TEXT NOT NULL DEFAULT 'pending'
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			username TEXT NOT NULL,
			organization_id INTEGER,
			action TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			resource_id INTEGER,
			ip_address TEXT NOT NULL,
			user_agent TEXT,
			details TEXT,
			success INTEGER NOT NULL DEFAULT 1,
			error_message TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
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

// initExportTestDB is deprecated - use setupExportTestDB instead
func initExportTestDB(t *testing.T) {
	testDB := setupExportTestDB(t)

	// Temporarily set db.DB to our test database
	originalDB := db.DB
	db.DB = testDB

	// Store the testDB in the test context for cleanup
	t.Cleanup(func() {
		testDB.Close()
		db.DB = originalDB
	})
}

func TestExportWorkspaceHandler_Unauthorized(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/workspaces/export?workspace_id=1", nil)
	rr := httptest.NewRecorder()

	err := ExportWorkspaceHandler(rr, req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}
}

func TestImportWorkspaceHandler_Unauthorized(t *testing.T) {
	reqBody := ImportRequest{
		OrganizationID: 1,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/workspaces/import", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	err := ImportWorkspaceHandler(rr, req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}
}

func TestGetExportStatusHandler_NotFound(t *testing.T) {
	// Set up test database
	initExportTestDB(t)

	req := httptest.NewRequest("GET", "/api/v1/workspaces/export/status?id=9999", nil)
	rr := httptest.NewRecorder()

	err := GetExportStatusHandler(rr, req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestListExportsHandler_Unauthorized(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/workspaces/exports", nil)
	rr := httptest.NewRecorder()

	err := ListExportsHandler(rr, req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}
}

func TestDeleteExportHandler_Unauthorized(t *testing.T) {
	req := httptest.NewRequest("DELETE", "/api/v1/workspaces/export?id=1", nil)
	rr := httptest.NewRecorder()

	err := DeleteExportHandler(rr, req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}
}

func TestGenerateWorkspaceTag(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"My Workspace", "my-workspace-"},
		{"Test-123", "test-123-"},
		{"Special@#Chars!", "specialchars-"},
		{"UPPERCASE", "uppercase-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := generateWorkspaceTag(tt.name)
			if len(tag) < len(tt.expected) {
				t.Errorf("Tag too short: %s", tag)
			}
			if tag[:len(tt.expected)] != tt.expected {
				t.Errorf("Tag %s does not start with expected prefix %s", tag, tt.expected)
			}
		})
	}
}

func TestGenerateWorkspaceTag_SpecialChars(t *testing.T) {
	tag := generateWorkspaceTag("My Workspace! @#$%")
	// Tag should start with my-workspace and have UUID appended
	// The tag format is: my-workspace-<uuid>
	if len(tag) < 20 {
		t.Errorf("Expected tag to be at least 20 chars, got: %s", tag)
	}
	// Check that it starts with "my-workspace"
	if tag[:12] != "my-workspace" {
		t.Errorf("Expected tag to start with 'my-workspace', got: %s", tag)
	}
}

func TestPerformWorkspaceExport(t *testing.T) {
	exportPath := filepath.Join(os.TempDir(), "test-perform-export.json")
	defer os.Remove(exportPath)

	req := ExportRequest{
		Format:      "json",
		IncludeData: false,
	}

	workspace := models.Workspace{
		ID:             1,
		Tag:            "test-workspace",
		Name:           "Test Workspace",
		OrganizationID: 1,
		OwnerID:        1,
		Provider:       "podman",
		Status:         "running",
		Config:         "{}",
		Region:         "us-east-1",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := performWorkspaceExport(workspace, req, exportPath)
	if err != nil {
		t.Fatalf("performWorkspaceExport returned error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Error("Export file was not created")
	}

	// Verify file content
	content, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	var exportData map[string]interface{}
	if err := json.Unmarshal(content, &exportData); err != nil {
		t.Fatalf("Failed to parse export data: %v", err)
	}

	if exportData["version"] != "1.0" {
		t.Errorf("Expected version 1.0, got %v", exportData["version"])
	}

	workspaceData, ok := exportData["workspace"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing workspace data in export")
	}

	if workspaceData["tag"] != "test-workspace" {
		t.Errorf("Expected tag test-workspace, got %v", workspaceData["tag"])
	}
}
