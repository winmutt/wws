package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"wws/api/internal/models"
)

func setupWorkspaceSharingTestDB(t *testing.T) *sql.DB {
	t.Helper()

	testDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	createWorkspaceSharingTablesForTest(testDB)

	return testDB
}

func TestNewWorkspaceSharingHandler(t *testing.T) {
	testDB := setupWorkspaceSharingTestDB(t)
	defer testDB.Close()

	handler := NewWorkspaceSharingHandler(testDB)
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
	if handler.DB != testDB {
		t.Error("Expected handler DB to match test DB")
	}
}

func TestShareWorkspace(t *testing.T) {
	testDB := setupWorkspaceSharingTestDB(t)
	defer testDB.Close()

	handler := NewWorkspaceSharingHandler(testDB)

	_, err := testDB.Exec("INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"github123", "testuser", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO workspaces (tag, name, organization_id, owner_id, provider, status) VALUES (?, ?, ?, ?, ?, ?)",
		"test-workspace", "Test Workspace", 1, 1, "podman", "running")
	if err != nil {
		t.Fatalf("Failed to create test workspace: %v", err)
	}

	shareReq := models.WorkspaceShareRequest{
		UserID: func() *int { i := 1; return &i }(),
		Role:   "editor",
	}

	jsonData, _ := json.Marshal(shareReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/1/share", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ShareWorkspace(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response models.WorkspaceShareResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Role != "editor" {
		t.Errorf("Expected role 'editor', got '%s'", response.Role)
	}
	if response.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", response.Status)
	}
}

func TestShareWorkspaceInvalidRole(t *testing.T) {
	testDB := setupWorkspaceSharingTestDB(t)
	defer testDB.Close()

	handler := NewWorkspaceSharingHandler(testDB)

	shareReq := models.WorkspaceShareRequest{
		UserID: func() *int { i := 1; return &i }(),
		Role:   "invalid_role",
	}

	jsonData, _ := json.Marshal(shareReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/1/share", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ShareWorkspace(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestRemoveWorkspaceMember(t *testing.T) {
	testDB := setupWorkspaceSharingTestDB(t)
	defer testDB.Close()

	handler := NewWorkspaceSharingHandler(testDB)

	_, err := testDB.Exec(`
		INSERT INTO workspace_members (workspace_id, user_id, username, email, role, status, invited_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, 1, 1, "testuser", "test@example.com", "editor", "active", time.Now())
	if err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/1/members/1", nil)
	w := httptest.NewRecorder()
	handler.RemoveWorkspaceMember(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	var status string
	err = testDB.QueryRow("SELECT status FROM workspace_members WHERE user_id = 1").Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query member: %v", err)
	}
	if status != "removed" {
		t.Errorf("Expected status 'removed', got '%s'", status)
	}
}

func TestGetDefaultPermissions(t *testing.T) {
	tests := []struct {
		role     string
		expected models.WorkspaceMemberPermissions
	}{
		{
			role: "owner",
			expected: models.WorkspaceMemberPermissions{
				CanView: true, CanEdit: true, CanShare: true, CanDelete: true,
				CanManageUsers: true, CanViewLogs: true, CanStartStop: true,
			},
		},
		{
			role: "admin",
			expected: models.WorkspaceMemberPermissions{
				CanView: true, CanEdit: true, CanShare: true, CanDelete: false,
				CanManageUsers: true, CanViewLogs: true, CanStartStop: true,
			},
		},
		{
			role: "editor",
			expected: models.WorkspaceMemberPermissions{
				CanView: true, CanEdit: true, CanShare: false, CanDelete: false,
				CanManageUsers: false, CanViewLogs: false, CanStartStop: true,
			},
		},
		{
			role: "viewer",
			expected: models.WorkspaceMemberPermissions{
				CanView: true, CanEdit: false, CanShare: false, CanDelete: false,
				CanManageUsers: false, CanViewLogs: false, CanStartStop: false,
			},
		},
	}

	for _, tt := range tests {
		permissions := getDefaultPermissions(tt.role)
		if permissions != tt.expected {
			t.Errorf("Role %s: expected %+v, got %+v", tt.role, tt.expected, permissions)
		}
	}
}

func createWorkspaceSharingTablesForTest(testDB *sql.DB) {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			github_id TEXT UNIQUE NOT NULL,
			username TEXT NOT NULL,
			email TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
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
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME
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
	}

	for _, table := range tables {
		_, err := testDB.Exec(table)
		if err != nil {
			panic(err)
		}
	}
}
