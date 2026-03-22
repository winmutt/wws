package analytics

import (
	"os"
	"testing"
	"time"

	"wws/api/internal/db"
)

func TestMain(m *testing.M) {
	// Setup test database
	dbPath := ":memory:"
	db.Init(dbPath)
	defer db.Close()

	// Create required tables for testing
	db.DB.Exec(`
		CREATE TABLE IF NOT EXISTS workspaces (
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
		)
	`)

	db.DB.Exec(`
		CREATE TABLE IF NOT EXISTS workspace_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id INTEGER NOT NULL,
			cpu_usage REAL NOT NULL DEFAULT 0,
			memory_usage REAL NOT NULL DEFAULT 0,
			storage_used_gb REAL NOT NULL DEFAULT 0,
			network_in_mb REAL NOT NULL DEFAULT 0,
			network_out_mb REAL NOT NULL DEFAULT 0,
			uptime_seconds INTEGER NOT NULL DEFAULT 0,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
		)
	`)

	db.DB.Exec(`
		CREATE TABLE IF NOT EXISTS organizations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			owner_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)

	db.DB.Exec(`
		CREATE TABLE IF NOT EXISTS usage_alerts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			workspace_id INTEGER,
			alert_type TEXT NOT NULL,
			severity TEXT NOT NULL,
			message TEXT NOT NULL,
			value REAL NOT NULL,
			threshold REAL NOT NULL,
			resolved INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			resolved_at DATETIME,
			FOREIGN KEY (organization_id) REFERENCES organizations(id),
			FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
		)
	`)

	os.Exit(m.Run())
}

func TestRecordWorkspaceUsage(t *testing.T) {
	// Create a test workspace
	db.DB.Exec("INSERT INTO workspaces (tag, name, organization_id, owner_id, provider, status) VALUES (?, ?, ?, ?, ?, ?)",
		"test-1", "Test Workspace", 1, 1, "podman", "running")

	ctx := t.Context()
	err := RecordWorkspaceUsage(ctx, 1, 50.5, 75.0, 10.5, 100.0, 50.0, 3600)
	if err != nil {
		t.Fatalf("Failed to record workspace usage: %v", err)
	}

	// Verify the record was created
	var count int
	err = db.DB.QueryRow("SELECT COUNT(*) FROM workspace_usage WHERE workspace_id = ?", 1).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count usage records: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 usage record, got %d", count)
	}
}

func TestGetWorkspaceUsage(t *testing.T) {
	// Insert test data
	db.DB.Exec("DELETE FROM workspace_usage")
	db.DB.Exec("INSERT INTO workspaces (tag, name, organization_id, owner_id, provider, status) VALUES (?, ?, ?, ?, ?, ?)",
		"test-2", "Test Workspace 2", 1, 1, "podman", "running")

	startTime := time.Now().Add(-48 * time.Hour)
	endTime := time.Now().Add(24 * time.Hour)
	for i := 0; i < 5; i++ {
		RecordWorkspaceUsage(t.Context(), 2, float64(30+i*10), 50.0, 5.0, 10.0, 5.0, 1800)
	}

	usages, err := GetWorkspaceUsage(t.Context(), 2, startTime, endTime, 10)
	if err != nil {
		t.Fatalf("Failed to get workspace usage: %v", err)
	}

	if len(usages) != 5 {
		t.Errorf("Expected 5 usage records, got %d", len(usages))
	}
}

func TestGetOrganizationUsage(t *testing.T) {
	// Setup test data
	db.DB.Exec("DELETE FROM workspace_usage")
	db.DB.Exec("DELETE FROM workspaces")
	db.DB.Exec("DELETE FROM organizations")

	db.DB.Exec("INSERT INTO organizations (id, name, owner_id) VALUES (?, ?, ?)", 1, "Test Org", 1)
	db.DB.Exec("INSERT INTO workspaces (tag, name, organization_id, owner_id, provider, status) VALUES (?, ?, ?, ?, ?, ?)",
		"org-test", "Org Workspace", 1, 1, "podman", "running")
	db.DB.Exec("INSERT INTO workspace_usage (workspace_id, cpu_usage, memory_usage, storage_used_gb, network_in_mb, network_out_mb, uptime_seconds) VALUES (?, ?, ?, ?, ?, ?, ?)",
		1, 45.0, 60.0, 15.0, 200.0, 100.0, 7200)

	usage, err := GetOrganizationUsage(t.Context(), 1)
	if err != nil {
		t.Fatalf("Failed to get organization usage: %v", err)
	}

	if usage == nil {
		t.Fatal("Expected organization usage, got nil")
	}

	if usage.TotalWorkspaces != 1 {
		t.Errorf("Expected 1 total workspace, got %d", usage.TotalWorkspaces)
	}

	if usage.ActiveWorkspaces != 1 {
		t.Errorf("Expected 1 active workspace, got %d", usage.ActiveWorkspaces)
	}
}

func TestGetWorkspaceStats(t *testing.T) {
	// Setup test data
	db.DB.Exec("DELETE FROM workspace_usage")
	db.DB.Exec("DELETE FROM workspaces")

	db.DB.Exec("INSERT INTO workspaces (id, tag, name, organization_id, owner_id, provider, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		1, "stats-test", "Stats Test Workspace", 1, 1, "podman", "running", time.Now())

	// Insert multiple usage records
	for i := 0; i < 3; i++ {
		RecordWorkspaceUsage(t.Context(), 1, 30.0+float64(i*10), 50.0, 5.0, 10.0, 5.0, 1800)
	}

	stats, err := GetWorkspaceStats(t.Context(), 1)
	if err != nil {
		t.Fatalf("Failed to get workspace stats: %v", err)
	}

	if stats.WorkspaceID != 1 {
		t.Errorf("Expected workspace ID 1, got %d", stats.WorkspaceID)
	}

	if stats.WorkspaceTag != "stats-test" {
		t.Errorf("Expected tag 'stats-test', got '%s'", stats.WorkspaceTag)
	}

	if stats.TotalUptimeHours != 1.0 {
		t.Errorf("Expected 1.0 uptime hours, got %f", stats.TotalUptimeHours)
	}
}

func TestCreateUsageAlert(t *testing.T) {
	// Setup test data
	db.DB.Exec("DELETE FROM usage_alerts")
	db.DB.Exec("DELETE FROM organizations")
	db.DB.Exec("INSERT INTO organizations (id, name, owner_id) VALUES (?, ?, ?)", 1, "Test Org", 1)

	err := CreateUsageAlert(t.Context(), 1, nil, "cpu_high", "high", "CPU usage is high", 95.0, 80.0)
	if err != nil {
		t.Fatalf("Failed to create usage alert: %v", err)
	}

	var count int
	err = db.DB.QueryRow("SELECT COUNT(*) FROM usage_alerts WHERE organization_id = ?", 1).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count alerts: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 alert, got %d", count)
	}
}

func TestGetActiveAlerts(t *testing.T) {
	// Setup test data
	db.DB.Exec("DELETE FROM usage_alerts")

	CreateUsageAlert(t.Context(), 1, nil, "cpu_high", "high", "CPU alert", 95.0, 80.0)
	CreateUsageAlert(t.Context(), 1, nil, "memory_high", "medium", "Memory alert", 85.0, 75.0)

	alerts, err := GetActiveAlerts(t.Context(), 1)
	if err != nil {
		t.Fatalf("Failed to get active alerts: %v", err)
	}

	if len(alerts) != 2 {
		t.Errorf("Expected 2 active alerts, got %d", len(alerts))
	}
}

func TestResolveAlert(t *testing.T) {
	// Setup test data
	db.DB.Exec("DELETE FROM usage_alerts")

	CreateUsageAlert(t.Context(), 1, nil, "cpu_high", "high", "CPU alert", 95.0, 80.0)

	// Get the alert ID
	var alertID int
	err := db.DB.QueryRow("SELECT id FROM usage_alerts WHERE organization_id = ? ORDER BY created_at DESC LIMIT 1", 1).Scan(&alertID)
	if err != nil {
		t.Fatalf("Failed to get alert ID: %v", err)
	}

	err = ResolveAlert(t.Context(), alertID)
	if err != nil {
		t.Fatalf("Failed to resolve alert: %v", err)
	}

	// Verify alert is resolved
	var resolved int
	err = db.DB.QueryRow("SELECT resolved FROM usage_alerts WHERE id = ?", alertID).Scan(&resolved)
	if err != nil {
		t.Fatalf("Failed to check alert resolution: %v", err)
	}

	if resolved != 1 {
		t.Error("Expected alert to be resolved")
	}
}

func TestGetAnalyticsSummary(t *testing.T) {
	// Setup test data
	db.DB.Exec("DELETE FROM workspace_usage")
	db.DB.Exec("DELETE FROM workspaces")

	db.DB.Exec("INSERT INTO workspaces (tag, name, organization_id, owner_id, provider, status) VALUES (?, ?, ?, ?, ?, ?)",
		"summary-1", "Summary Test 1", 1, 1, "podman", "running")
	db.DB.Exec("INSERT INTO workspaces (tag, name, organization_id, owner_id, provider, status) VALUES (?, ?, ?, ?, ?, ?)",
		"summary-2", "Summary Test 2", 1, 1, "podman", "stopped")

	RecordWorkspaceUsage(t.Context(), 1, 80.0, 70.0, 20.0, 100.0, 50.0, 3600)

	summary, err := GetAnalyticsSummary(t.Context(), nil)
	if err != nil {
		t.Fatalf("Failed to get analytics summary: %v", err)
	}

	if summary.TotalWorkspaces != 2 {
		t.Errorf("Expected 2 total workspaces, got %d", summary.TotalWorkspaces)
	}

	if summary.ActiveWorkspaces != 1 {
		t.Errorf("Expected 1 active workspace, got %d", summary.ActiveWorkspaces)
	}
}
