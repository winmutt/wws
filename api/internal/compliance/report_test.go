package compliance

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewComplianceReportGenerator(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewComplianceReportGenerator(filepath.Join(tmpDir, "audit"), filepath.Join(tmpDir, "storage"))

	if generator == nil {
		t.Fatal("Expected non-nil generator")
	}
	if generator.auditLogPath != filepath.Join(tmpDir, "audit") {
		t.Errorf("Expected auditLogPath %s, got %s", filepath.Join(tmpDir, "audit"), generator.auditLogPath)
	}
}

func TestGenerateReport(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewComplianceReportGenerator(filepath.Join(tmpDir, "audit"), filepath.Join(tmpDir, "storage"))

	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()

	report, err := generator.GenerateReport(nil, startDate, endDate)
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}

	if report.ID == "" {
		t.Error("Expected report ID")
	}
	if !report.GeneratedAt.After(startDate) {
		t.Error("Expected generated at to be after start date")
	}
	if report.Period.StartDate != startDate {
		t.Errorf("Expected start date %v, got %v", startDate, report.Period.StartDate)
	}
	if report.Period.EndDate != endDate {
		t.Errorf("Expected end date %v, got %v", endDate, report.Period.EndDate)
	}
	if report.Summary.TotalAuditEntries != 0 {
		t.Errorf("Expected 0 audit entries, got %d", report.Summary.TotalAuditEntries)
	}
}

func TestGenerateReportWithOrg(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewComplianceReportGenerator(filepath.Join(tmpDir, "audit"), filepath.Join(tmpDir, "storage"))

	orgID := 123
	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()

	report, err := generator.GenerateReport(&orgID, startDate, endDate)
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}

	if report.Organization == nil {
		t.Error("Expected organization to be set")
	} else if *report.Organization != "123" {
		t.Errorf("Expected organization 123, got %s", *report.Organization)
	}
}

func TestCalculateComplianceScore(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewComplianceReportGenerator(filepath.Join(tmpDir, "audit"), filepath.Join(tmpDir, "storage"))

	tests := []struct {
		name       string
		violations []ComplianceViolation
		expected   float64
	}{
		{
			name:       "no violations",
			violations: []ComplianceViolation{},
			expected:   100.0,
		},
		{
			name: "one critical violation",
			violations: []ComplianceViolation{
				{Severity: "critical"},
			},
			expected: 85.0,
		},
		{
			name: "one high violation",
			violations: []ComplianceViolation{
				{Severity: "high"},
			},
			expected: 90.0,
		},
		{
			name: "one medium violation",
			violations: []ComplianceViolation{
				{Severity: "medium"},
			},
			expected: 95.0,
		},
		{
			name: "one low violation",
			violations: []ComplianceViolation{
				{Severity: "low"},
			},
			expected: 98.0,
		},
		{
			name: "multiple violations",
			violations: []ComplianceViolation{
				{Severity: "critical"},
				{Severity: "high"},
				{Severity: "medium"},
			},
			expected: 70.0, // 100 - 15 - 10 - 5 = 70
		},
		{
			name: "score clamped to zero",
			violations: []ComplianceViolation{
				{Severity: "critical"}, {Severity: "critical"}, {Severity: "critical"},
				{Severity: "critical"}, {Severity: "critical"}, {Severity: "critical"},
				{Severity: "critical"}, {Severity: "critical"}, {Severity: "critical"},
				{Severity: "critical"},
			},
			expected: 0.0, // Would be -50, but clamped to 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := generator.calculateComplianceScore(tt.violations)
			if score != tt.expected {
				t.Errorf("Expected score %f, got %f", tt.expected, score)
			}
		})
	}
}

func TestGetScoreClass(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewComplianceReportGenerator(filepath.Join(tmpDir, "audit"), filepath.Join(tmpDir, "storage"))

	tests := []struct {
		score  float64
		expect string
	}{
		{score: 100, expect: "good"},
		{score: 80, expect: "good"},
		{score: 75, expect: "warning"},
		{score: 60, expect: "warning"},
		{score: 50, expect: "bad"},
		{score: 0, expect: "bad"},
	}

	for _, tt := range tests {
		result := generator.getScoreClass(tt.score)
		if result != tt.expect {
			t.Errorf("For score %f, expected %s, got %s", tt.score, tt.expect, result)
		}
	}
}

func TestSaveReport(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewComplianceReportGenerator(filepath.Join(tmpDir, "audit"), filepath.Join(tmpDir, "storage"))

	report := &ComplianceReport{
		ID:          "test-report-123",
		GeneratedAt: time.Now(),
		Period: ReportPeriod{
			StartDate: time.Now().AddDate(0, -1, 0),
			EndDate:   time.Now(),
		},
		Summary: ReportSummary{
			TotalAuditEntries: 10,
		},
	}

	err := generator.SaveReport(report)
	if err != nil {
		t.Fatalf("Failed to save report: %v", err)
	}

	// Check file exists
	expectedPath := filepath.Join(tmpDir, "storage", "compliance-reports", "compliance-test-report-123.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected report file to exist at %s", expectedPath)
	}
}

func TestExportReportJSON(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewComplianceReportGenerator(filepath.Join(tmpDir, "audit"), filepath.Join(tmpDir, "storage"))

	report := &ComplianceReport{
		ID:          "test-export",
		GeneratedAt: time.Now(),
		Period: ReportPeriod{
			StartDate: time.Now().AddDate(0, -1, 0),
			EndDate:   time.Now(),
		},
		Summary: ReportSummary{
			TotalAuditEntries: 5,
		},
	}

	data, err := generator.ExportReport(report, "json")
	if err != nil {
		t.Fatalf("Failed to export report: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty JSON output")
	}

	// Verify it's valid JSON by checking for opening brace
	if string(data[0]) != "{" {
		t.Errorf("Expected JSON to start with '{', got '%c'", data[0])
	}
}

func TestExportReportCSV(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewComplianceReportGenerator(filepath.Join(tmpDir, "audit"), filepath.Join(tmpDir, "storage"))

	report := &ComplianceReport{
		ID:          "test-csv",
		GeneratedAt: time.Now(),
		Violations: []ComplianceViolation{
			{
				ID:          "VULN-001",
				Type:        "test_type",
				Severity:    "high",
				Description: "Test violation",
				DetectedAt:  time.Now(),
				Status:      "open",
			},
		},
	}

	data, err := generator.ExportReport(report, "csv")
	if err != nil {
		t.Fatalf("Failed to export report: %v", err)
	}

	csv := string(data)
	if len(csv) == 0 {
		t.Error("Expected non-empty CSV output")
	}

	// Check for header (accounting for newline)
	expectedHeader := "ID,Type,Severity,Description,DetectedAt,Status\n"
	if len(csv) < len(expectedHeader) || csv[:len(expectedHeader)] != expectedHeader {
		t.Errorf("Expected CSV header not found, got: %s", csv[:min(50, len(csv))])
	}
}

func TestExportReportHTML(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewComplianceReportGenerator(filepath.Join(tmpDir, "audit"), filepath.Join(tmpDir, "storage"))

	report := &ComplianceReport{
		ID:          "test-html",
		GeneratedAt: time.Now(),
		Period: ReportPeriod{
			StartDate: time.Now().AddDate(0, -1, 0),
			EndDate:   time.Now(),
		},
		Summary: ReportSummary{
			TotalAuditEntries: 10,
			ComplianceScore:   85.0,
		},
	}

	data, err := generator.ExportReport(report, "html")
	if err != nil {
		t.Fatalf("Failed to export report: %v", err)
	}

	html := string(data)
	if len(html) == 0 {
		t.Error("Expected non-empty HTML output")
	}

	// Check for HTML structure
	if !contains(html, "<!DOCTYPE html>") {
		t.Error("Expected HTML doctype")
	}
	if !contains(html, "Compliance Report") {
		t.Error("Expected report title in HTML")
	}
}

func TestExportReportUnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewComplianceReportGenerator(filepath.Join(tmpDir, "audit"), filepath.Join(tmpDir, "storage"))

	report := &ComplianceReport{ID: "test"}
	_, err := generator.ExportReport(report, "xml")

	if err == nil {
		t.Error("Expected error for unsupported format")
	}
	if err.Error() != "unsupported format: xml" {
		t.Errorf("Expected 'unsupported format: xml', got '%s'", err.Error())
	}
}

func TestAnalyzeViolations(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewComplianceReportGenerator(filepath.Join(tmpDir, "audit"), filepath.Join(tmpDir, "storage"))

	// Test with no audit logs
	violations := generator.analyzeViolations([]AuditLogEntry{})
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations, got %d", len(violations))
	}

	// Test with excessive failed logins
	auditLogs := []AuditLogEntry{}
	for i := 0; i < 15; i++ {
		auditLogs = append(auditLogs, AuditLogEntry{
			UserID:    1,
			Action:    "login_failed",
			Success:   false,
			Timestamp: time.Now(),
		})
	}

	violations = generator.analyzeViolations(auditLogs)
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for excessive failed logins, got %d", len(violations))
	}
	if violations[0].Type != "excessive_failed_logins" {
		t.Errorf("Expected violation type 'excessive_failed_logins', got '%s'", violations[0].Type)
	}
}

func TestGetComplianceScore(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewComplianceReportGenerator(filepath.Join(tmpDir, "audit"), filepath.Join(tmpDir, "storage"))

	score, err := generator.GetComplianceScore(nil)
	if err != nil {
		t.Fatalf("Failed to get compliance score: %v", err)
	}

	if score < 0 || score > 100 {
		t.Errorf("Expected score between 0-100, got %f", score)
	}
}

func TestComplianceReportStruct(t *testing.T) {
	// Test all struct fields are properly initialized
	report := ComplianceReport{
		ID:          "test-123",
		GeneratedAt: time.Now(),
		Period: ReportPeriod{
			StartDate: time.Now().AddDate(0, -1, 0),
			EndDate:   time.Now(),
		},
		Summary: ReportSummary{
			TotalAuditEntries: 100,
			ComplianceScore:   85.5,
		},
		Metrics: ComplianceMetrics{
			AuthenticationMetrics: AuthMetrics{
				TotalLogins:  50,
				FailedLogins: 5,
			},
		},
	}

	if report.ID != "test-123" {
		t.Error("ID not set correctly")
	}
	if report.Summary.ComplianceScore != 85.5 {
		t.Error("ComplianceScore not set correctly")
	}
}

func TestRecommendationGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewComplianceReportGenerator(filepath.Join(tmpDir, "audit"), filepath.Join(tmpDir, "storage"))

	// Test with high failed logins
	metrics := ComplianceMetrics{
		AuthenticationMetrics: AuthMetrics{
			FailedLogins: 150,
		},
	}

	recommendations := generator.generateRecommendations([]ComplianceViolation{}, metrics)

	foundAccountLockout := false
	for _, rec := range recommendations {
		if rec.Title == "Implement Account Lockout Policy" {
			foundAccountLockout = true
			break
		}
	}

	if !foundAccountLockout {
		t.Error("Expected recommendation for account lockout policy")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr))))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
