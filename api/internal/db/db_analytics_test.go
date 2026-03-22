package db

import (
	"testing"
	"time"
)

func TestAnalyticsTables(t *testing.T) {
	// Test that analytics tables are created
	var count int

	// Check workspace_usage table
	err := DB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='workspace_usage'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check workspace_usage table: %v", err)
	}
	if count != 1 {
		t.Error("workspace_usage table not created")
	}

	// Check usage_alerts table
	err = DB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='usage_alerts'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check usage_alerts table: %v", err)
	}
	if count != 1 {
		t.Error("usage_alerts table not created")
	}

	// Check indexes
	err = DB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_workspace_usage_workspace_id'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check workspace_usage index: %v", err)
	}
	if count != 1 {
		t.Error("idx_workspace_usage_workspace_id index not created")
	}
}

func TestWorkspaceUsageCRUD(t *testing.T) {
	// Insert test data
	_, err := DB.Exec("INSERT INTO workspaces (tag, name, organization_id, owner_id, provider, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"test-analytics", "Analytics Test", 1, 1, "podman", "running", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert workspace: %v", err)
	}

	// Insert usage record
	_, err = DB.Exec("INSERT INTO workspace_usage (workspace_id, cpu_usage, memory_usage, storage_used_gb, network_in_mb, network_out_mb, uptime_seconds, timestamp) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		1, 50.5, 75.0, 10.5, 100.0, 50.0, 3600, time.Now())
	if err != nil {
		t.Fatalf("Failed to insert usage record: %v", err)
	}

	// Query usage record
	var cpuUsage, memoryUsage, storageGB float64
	var uptimeSeconds int64
	err = DB.QueryRow("SELECT cpu_usage, memory_usage, storage_used_gb, uptime_seconds FROM workspace_usage WHERE workspace_id = ?", 1).Scan(
		&cpuUsage, &memoryUsage, &storageGB, &uptimeSeconds)
	if err != nil {
		t.Fatalf("Failed to query usage record: %v", err)
	}

	if cpuUsage != 50.5 {
		t.Errorf("Expected CPU usage 50.5, got %f", cpuUsage)
	}
	if memoryUsage != 75.0 {
		t.Errorf("Expected memory usage 75.0, got %f", memoryUsage)
	}
}

func TestUsageAlertsCRUD(t *testing.T) {
	// Insert test organization
	_, err := DB.Exec("INSERT INTO organizations (id, name, owner_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		1, "Test Org", 1, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert organization: %v", err)
	}

	// Insert alert
	_, err = DB.Exec("INSERT INTO usage_alerts (organization_id, workspace_id, alert_type, severity, message, value, threshold, resolved, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		1, nil, "cpu_high", "high", "CPU usage is high", 95.0, 80.0, 0, time.Now())
	if err != nil {
		t.Fatalf("Failed to insert alert: %v", err)
	}

	// Query alert
	var alertType, severity, message string
	var value, threshold float64
	var resolved int
	err = DB.QueryRow("SELECT alert_type, severity, message, value, threshold, resolved FROM usage_alerts WHERE organization_id = ?", 1).Scan(
		&alertType, &severity, &message, &value, &threshold, &resolved)
	if err != nil {
		t.Fatalf("Failed to query alert: %v", err)
	}

	if alertType != "cpu_high" {
		t.Errorf("Expected alert_type 'cpu_high', got '%s'", alertType)
	}
	if resolved != 0 {
		t.Error("Expected alert to be unresolved")
	}

	// Resolve alert
	_, err = DB.Exec("UPDATE usage_alerts SET resolved = 1, resolved_at = ? WHERE organization_id = ?", time.Now(), 1)
	if err != nil {
		t.Fatalf("Failed to resolve alert: %v", err)
	}

	// Verify resolution
	err = DB.QueryRow("SELECT resolved FROM usage_alerts WHERE organization_id = ?", 1).Scan(&resolved)
	if err != nil {
		t.Fatalf("Failed to verify alert resolution: %v", err)
	}

	if resolved != 1 {
		t.Error("Expected alert to be resolved")
	}
}
