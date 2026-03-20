package scanning

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultScanConfig(t *testing.T) {
	config := DefaultScanConfig()

	if !config.EnableSAST {
		t.Error("Expected SAST to be enabled by default")
	}

	if config.EnableDAST {
		t.Error("Expected DAST to be disabled by default")
	}

	if !config.EnableSCA {
		t.Error("Expected SCA to be enabled by default")
	}

	if !config.EnableSecretScan {
		t.Error("Expected secret scanning to be enabled by default")
	}

	if config.SeverityThreshold != "low" {
		t.Errorf("Expected severity threshold 'low', got %s", config.SeverityThreshold)
	}
}

func TestNewSecurityScanner(t *testing.T) {
	scanner := NewSecurityScanner(nil)

	if scanner == nil {
		t.Fatal("Expected non-nil scanner")
	}

	defaultConfig := DefaultScanConfig()
	if defaultConfig.EnableSAST != scanner.config.EnableSAST {
		t.Error("Expected default config to be used")
	}
}

func TestNewSecurityScannerWithConfig(t *testing.T) {
	config := &ScanConfig{
		EnableSAST:        false,
		EnableSCA:         false,
		EnableSecretScan:  false,
		SeverityThreshold: "high",
	}

	scanner := NewSecurityScanner(config)

	if scanner.config.EnableSAST {
		t.Error("Expected SAST to be disabled")
	}

	if scanner.config.SeverityThreshold != "high" {
		t.Errorf("Expected severity threshold 'high', got %s", scanner.config.SeverityThreshold)
	}
}

func TestGetSeverityLevel(t *testing.T) {
	scanner := NewSecurityScanner(nil)

	if scanner.getSeverityLevel("critical") != 5 {
		t.Error("Expected critical to be 5")
	}

	if scanner.getSeverityLevel("high") != 4 {
		t.Error("Expected high to be 4")
	}

	if scanner.getSeverityLevel("medium") != 3 {
		t.Error("Expected medium to be 3")
	}

	if scanner.getSeverityLevel("low") != 2 {
		t.Error("Expected low to be 2")
	}

	if scanner.getSeverityLevel("info") != 1 {
		t.Error("Expected info to be 1")
	}

	if scanner.getSeverityLevel("unknown") != 0 {
		t.Error("Expected unknown to be 0")
	}
}

func TestFilterBySeverity(t *testing.T) {
	scanner := NewSecurityScanner(nil)
	scanner.config.SeverityThreshold = "medium"

	vulns := []Vulnerability{
		{ID: "1", Severity: "critical"},
		{ID: "2", Severity: "high"},
		{ID: "3", Severity: "medium"},
		{ID: "4", Severity: "low"},
		{ID: "5", Severity: "info"},
	}

	filtered := scanner.filterBySeverity(vulns)

	if len(filtered) != 3 {
		t.Errorf("Expected 3 vulnerabilities, got %d", len(filtered))
	}

	for _, v := range filtered {
		if v.Severity == "low" || v.Severity == "info" {
			t.Errorf("Expected no low or info vulnerabilities, got %s", v.Severity)
		}
	}
}

func TestGenerateSummary(t *testing.T) {
	scanner := NewSecurityScanner(nil)

	vulns := []Vulnerability{
		{ID: "1", Severity: "critical"},
		{ID: "2", Severity: "critical"},
		{ID: "3", Severity: "high"},
		{ID: "4", Severity: "medium"},
		{ID: "5", Severity: "low"},
		{ID: "6", Severity: "info"},
	}

	summary := scanner.generateSummary(vulns)

	if summary.TotalVulnerabilities != 6 {
		t.Errorf("Expected 6 total, got %d", summary.TotalVulnerabilities)
	}

	if summary.Critical != 2 {
		t.Errorf("Expected 2 critical, got %d", summary.Critical)
	}

	if summary.High != 1 {
		t.Errorf("Expected 1 high, got %d", summary.High)
	}

	if summary.Medium != 1 {
		t.Errorf("Expected 1 medium, got %d", summary.Medium)
	}

	if summary.Low != 1 {
		t.Errorf("Expected 1 low, got %d", summary.Low)
	}

	if summary.Info != 1 {
		t.Errorf("Expected 1 info, got %d", summary.Info)
	}
}

func TestCommandExists(t *testing.T) {
	scanner := NewSecurityScanner(nil)

	// sh should exist on Unix
	if !scanner.commandExists("sh") {
		t.Error("Expected sh to exist")
	}

	// nonexistent_cmd should not exist
	if scanner.commandExists("nonexistent_cmd_12345") {
		t.Error("Expected nonexistent_cmd to not exist")
	}
}

func TestGetString(t *testing.T) {
	m := map[string]interface{}{
		"key": "value",
		"nested": map[string]interface{}{
			"inner": "inner_value",
		},
	}

	if getString(m, "key") != "value" {
		t.Error("Expected 'value'")
	}

	if getString(m, "nested", "inner") != "inner_value" {
		t.Error("Expected 'inner_value'")
	}

	if getString(m, "nonexistent") != "" {
		t.Error("Expected empty string for nonexistent key")
	}
}

func TestGetInt(t *testing.T) {
	m := map[string]interface{}{
		"key": 42.0,
	}

	if getInt(m, "key") != 42 {
		t.Error("Expected 42")
	}

	if getInt(m, "nonexistent") != 0 {
		t.Error("Expected 0 for nonexistent key")
	}
}

func TestGetFloat(t *testing.T) {
	m := map[string]interface{}{
		"key": 3.14,
	}

	if getFloat(m, "key") != 3.14 {
		t.Error("Expected 3.14")
	}

	if getFloat(m, "nonexistent") != 0 {
		t.Error("Expected 0 for nonexistent key")
	}
}

func TestFindFiles(t *testing.T) {
	scanner := NewSecurityScanner(nil)

	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test.js"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("test"), 0644)

	files := scanner.findFiles(tmpDir, "*.go", "*.js")

	if len(files) < 2 {
		t.Errorf("Expected at least 2 files, got %d", len(files))
	}
}

func TestBasicPatternScan(t *testing.T) {
	scanner := NewSecurityScanner(nil)

	// Create temp directory with test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	content := `
package test
var password = "secret123"
var apiKey = "abc123"
`
	os.WriteFile(testFile, []byte(content), 0644)

	vulns := scanner.basicPatternScan(tmpDir)

	if len(vulns) == 0 {
		t.Error("Expected to find some vulnerabilities")
	}
}

func TestBasicSecretScan(t *testing.T) {
	scanner := NewSecurityScanner(nil)

	// Create temp directory with test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.env")

	content := `
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
PRIVATE_KEY=-----BEGIN RSA PRIVATE KEY-----
`
	os.WriteFile(testFile, []byte(content), 0644)

	vulns := scanner.basicSecretScan(tmpDir)

	if len(vulns) == 0 {
		t.Error("Expected to find some secrets")
	}

	for _, vuln := range vulns {
		if vuln.Severity != "critical" {
			t.Errorf("Expected critical severity for secrets, got %s", vuln.Severity)
		}
	}
}

func TestSaveReport(t *testing.T) {
	scanner := NewSecurityScanner(nil)
	scanner.config.ReportDirectory = t.TempDir()

	result := &ScanResult{
		ID:          "test-scan-123",
		Target:      "/test",
		ScanType:    "full",
		Status:      "completed",
		StartedAt:   time.Now().Add(-1 * time.Hour),
		CompletedAt: func() *time.Time { t := time.Now(); return &t }(),
		Vulnerabilities: []Vulnerability{
			{ID: "1", Title: "Test Vuln", Severity: "high"},
		},
		Summary: ScanSummary{
			TotalVulnerabilities: 1,
			High:                 1,
		},
	}

	err := scanner.saveReport(result)
	if err != nil {
		t.Fatalf("Failed to save report: %v", err)
	}

	// Check if file was created
	filename := filepath.Join(scanner.config.ReportDirectory, "test-scan-123.json")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("Expected report file to be created")
	}
}

func TestScanResultCreation(t *testing.T) {
	result := &ScanResult{
		ID:        "test-123",
		Target:    "/test",
		ScanType:  "full",
		Status:    "running",
		StartedAt: time.Now(),
	}

	if result.ID != "test-123" {
		t.Errorf("Expected ID 'test-123', got %s", result.ID)
	}

	if result.Status != "running" {
		t.Errorf("Expected status 'running', got %s", result.Status)
	}
}

func TestVulnerabilityStruct(t *testing.T) {
	vuln := Vulnerability{
		ID:          "CVE-2024-1234",
		Title:       "Test Vulnerability",
		Description: "Test description",
		Severity:    "high",
		CVE:         "CVE-2024-1234",
		CVSS:        7.5,
		Package:     "test-package",
		Version:     "1.0.0",
		Location:    "test.go",
		Line:        42,
		Solution:    "Upgrade to version 2.0.0",
		References:  []string{"https://example.com"},
		DetectedAt:  time.Now(),
	}

	if vuln.ID != "CVE-2024-1234" {
		t.Error("Expected correct ID")
	}

	if vuln.CVSS != 7.5 {
		t.Error("Expected correct CVSS")
	}

	if len(vuln.References) != 1 {
		t.Error("Expected 1 reference")
	}
}

func TestScanSummaryStruct(t *testing.T) {
	summary := ScanSummary{
		TotalVulnerabilities: 10,
		Critical:             2,
		High:                 3,
		Medium:               3,
		Low:                  2,
	}

	if summary.TotalVulnerabilities != 10 {
		t.Error("Expected 10 total")
	}

	if summary.Critical+summary.High+summary.Medium+summary.Low != summary.TotalVulnerabilities {
		t.Error("Sum should equal total")
	}
}
