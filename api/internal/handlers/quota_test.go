package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"wws/api/internal/models"
)

func setupTestDB(t *testing.T) *sql.DB {
	// Create temporary test database
	tmpDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create tables
	createTestTables(tmpDB)

	return tmpDB
}

func createTestTables(db *sql.DB) {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			github_id TEXT UNIQUE NOT NULL,
			username TEXT NOT NULL,
			email TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS organizations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			owner_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
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
		);`,
		`CREATE TABLE IF NOT EXISTS members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			organization_id INTEGER NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			invited_by INTEGER,
			accepted INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS resource_quotas (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL UNIQUE,
			max_workspaces INTEGER NOT NULL DEFAULT 10,
			max_users INTEGER NOT NULL DEFAULT 5,
			max_storage_gb INTEGER NOT NULL DEFAULT 50,
			max_compute_hours INTEGER NOT NULL DEFAULT 100,
			max_network_bandwidth INTEGER NOT NULL DEFAULT 1000,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS quota_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL UNIQUE,
			workspaces_count INTEGER NOT NULL DEFAULT 0,
			users_count INTEGER NOT NULL DEFAULT 0,
			storage_used_gb INTEGER NOT NULL DEFAULT 0,
			compute_hours_used INTEGER NOT NULL DEFAULT 0,
			network_bandwidth_used INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, table := range tables {
		_, err := db.Exec(table)
		if err != nil {
			panic(err)
		}
	}
}

func TestQuotaHandlerGetQuota(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.Close()

	// Create test organization
	_, err := testDB.Exec("INSERT INTO organizations (name, owner_id) VALUES (?, ?)", "Test Org", 1)
	if err != nil {
		t.Fatalf("Failed to create test organization: %v", err)
	}

	handler := &QuotaHandler{DB: testDB}

	// Test with existing quota
	_, err = testDB.Exec("INSERT INTO resource_quotas (organization_id) VALUES (?)", 1)
	if err != nil {
		t.Fatalf("Failed to create test quota: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/quotas", nil)
	req = req.WithContext(context.WithValue(req.Context(), "organization_id", 1))
	w := httptest.NewRecorder()

	handler.GetQuota(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var quota models.ResourceQuota
	if err := json.NewDecoder(w.Body).Decode(&quota); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if quota.OrganizationID != 1 {
		t.Errorf("Expected organization_id 1, got %d", quota.OrganizationID)
	}
}

func TestQuotaHandlerUpdateQuota(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.Close()

	// Create test organization
	_, err := testDB.Exec("INSERT INTO organizations (name, owner_id) VALUES (?, ?)", "Test Org", 1)
	if err != nil {
		t.Fatalf("Failed to create test organization: %v", err)
	}

	handler := &QuotaHandler{DB: testDB}

	// Update quota
	updateData := models.QuotaUpdateRequest{
		MaxWorkspaces:       20,
		MaxUsers:            10,
		MaxStorageGB:        100,
		MaxComputeHours:     200,
		MaxNetworkBandwidth: 2000,
	}

	jsonData, _ := json.Marshal(updateData)
	req := httptest.NewRequest("PUT", "/api/v1/quotas", bytes.NewBuffer(jsonData))
	req = req.WithContext(context.WithValue(req.Context(), "organization_id", 1))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateQuota(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var quota models.ResourceQuota
	if err := json.NewDecoder(w.Body).Decode(&quota); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if quota.MaxWorkspaces != 20 {
		t.Errorf("Expected max_workspaces 20, got %d", quota.MaxWorkspaces)
	}
	if quota.MaxUsers != 10 {
		t.Errorf("Expected max_users 10, got %d", quota.MaxUsers)
	}
}

func TestQuotaHandlerGetUsage(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.Close()

	// Create test organization
	_, err := testDB.Exec("INSERT INTO organizations (name, owner_id) VALUES (?, ?)", "Test Org", 1)
	if err != nil {
		t.Fatalf("Failed to create test organization: %v", err)
	}

	// Create test workspace
	_, err = testDB.Exec("INSERT INTO workspaces (tag, name, organization_id, owner_id, provider, status) VALUES (?, ?, ?, ?, ?, ?)",
		"ws1", "Test Workspace", 1, 1, "podman", "running")
	if err != nil {
		t.Fatalf("Failed to create test workspace: %v", err)
	}

	// Create test member
	_, err = testDB.Exec("INSERT INTO members (user_id, organization_id) VALUES (?, ?)", 2, 1)
	if err != nil {
		t.Fatalf("Failed to create test member: %v", err)
	}

	handler := &QuotaHandler{DB: testDB}

	req := httptest.NewRequest("GET", "/api/v1/quotas/usage", nil)
	req = req.WithContext(context.WithValue(req.Context(), "organization_id", 1))
	w := httptest.NewRecorder()

	handler.GetUsage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var usage models.QuotaUsage
	if err := json.NewDecoder(w.Body).Decode(&usage); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if usage.WorkspacesCount != 1 {
		t.Errorf("Expected workspaces_count 1, got %d", usage.WorkspacesCount)
	}
	if usage.UsersCount != 1 {
		t.Errorf("Expected users_count 1, got %d", usage.UsersCount)
	}
}

func TestQuotaHandlerCheckQuota(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.Close()

	// Create test organization
	_, err := testDB.Exec("INSERT INTO organizations (name, owner_id) VALUES (?, ?)", "Test Org", 1)
	if err != nil {
		t.Fatalf("Failed to create test organization: %v", err)
	}

	// Create quota with limited workspaces
	_, err = testDB.Exec("INSERT INTO resource_quotas (organization_id, max_workspaces) VALUES (?, ?)", 1, 2)
	if err != nil {
		t.Fatalf("Failed to create test quota: %v", err)
	}

	// Create existing workspaces
	_, err = testDB.Exec("INSERT INTO workspaces (tag, name, organization_id, owner_id, provider, status) VALUES (?, ?, ?, ?, ?, ?)",
		"ws1", "Test Workspace 1", 1, 1, "podman", "running")
	if err != nil {
		t.Fatalf("Failed to create test workspace: %v", err)
	}
	_, err = testDB.Exec("INSERT INTO quota_usage (organization_id, workspaces_count) VALUES (?, ?)", 1, 1)
	if err != nil {
		t.Fatalf("Failed to create test usage: %v", err)
	}

	handler := &QuotaHandler{DB: testDB}

	checkData := struct {
		Resource string `json:"resource"`
		Amount   int    `json:"amount"`
	}{
		Resource: "workspaces",
		Amount:   1,
	}

	jsonData, _ := json.Marshal(checkData)
	req := httptest.NewRequest("POST", "/api/v1/quotas/check", bytes.NewBuffer(jsonData))
	req = req.WithContext(context.WithValue(req.Context(), "organization_id", 1))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CheckQuota(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var results []models.QuotaCheckResult
	if err := json.NewDecoder(w.Body).Decode(&results); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	// Should be allowed (1 workspace used, limit 2, requesting 1 more)
	if !results[0].Allowed {
		t.Errorf("Expected allowed=true, got allowed=false")
	}
}

func TestQuotaHandlerCheckQuotaExceeded(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.Close()

	// Create test organization
	_, err := testDB.Exec("INSERT INTO organizations (name, owner_id) VALUES (?, ?)", "Test Org", 1)
	if err != nil {
		t.Fatalf("Failed to create test organization: %v", err)
	}

	// Create quota with limited workspaces
	_, err = testDB.Exec("INSERT INTO resource_quotas (organization_id, max_workspaces) VALUES (?, ?)", 1, 1)
	if err != nil {
		t.Fatalf("Failed to create test quota: %v", err)
	}

	// Create usage showing quota is full
	_, err = testDB.Exec("INSERT INTO quota_usage (organization_id, workspaces_count) VALUES (?, ?)", 1, 1)
	if err != nil {
		t.Fatalf("Failed to create test usage: %v", err)
	}

	handler := &QuotaHandler{DB: testDB}

	checkData := struct {
		Resource string `json:"resource"`
		Amount   int    `json:"amount"`
	}{
		Resource: "workspaces",
		Amount:   1,
	}

	jsonData, _ := json.Marshal(checkData)
	req := httptest.NewRequest("POST", "/api/v1/quotas/check", bytes.NewBuffer(jsonData))
	req = req.WithContext(context.WithValue(req.Context(), "organization_id", 1))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CheckQuota(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var results []models.QuotaCheckResult
	if err := json.NewDecoder(w.Body).Decode(&results); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should not be allowed (1 workspace used, limit 1, requesting 1 more)
	if results[0].Allowed {
		t.Errorf("Expected allowed=false, got allowed=true")
	}
}

// Test getOrgIDFromRequest
func TestGetOrgIDFromRequest(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(context.WithValue(req.Context(), "organization_id", 42))

	orgID := getOrgIDFromRequest(req)
	if orgID != 42 {
		t.Errorf("Expected orgID 42, got %d", orgID)
	}
}

func TestGetOrgIDFromRequestMissing(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)

	orgID := getOrgIDFromRequest(req)
	if orgID != 0 {
		t.Errorf("Expected orgID 0, got %d", orgID)
	}
}
