package compliance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ComplianceReport represents a compliance report
type ComplianceReport struct {
	ID              string                `json:"id"`
	GeneratedAt     time.Time             `json:"generated_at"`
	Period          ReportPeriod          `json:"period"`
	Organization    *string               `json:"organization,omitempty"`
	Summary         ReportSummary         `json:"summary"`
	AuditLogs       []AuditLogEntry       `json:"audit_logs"`
	Violations      []ComplianceViolation `json:"violations"`
	Recommendations []Recommendation      `json:"recommendations"`
	Metrics         ComplianceMetrics     `json:"metrics"`
}

// ReportPeriod defines the report time period
type ReportPeriod struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

// ReportSummary provides a high-level summary
type ReportSummary struct {
	TotalAuditEntries  int     `json:"total_audit_entries"`
	TotalViolations    int     `json:"total_violations"`
	CriticalViolations int     `json:"critical_violations"`
	HighViolations     int     `json:"high_violations"`
	MediumViolations   int     `json:"medium_violations"`
	LowViolations      int     `json:"low_violations"`
	ComplianceScore    float64 `json:"compliance_score"`
	PreviousScore      float64 `json:"previous_score,omitempty"`
	ScoreChange        float64 `json:"score_change,omitempty"`
}

// AuditLogEntry represents an audit log entry in the report
type AuditLogEntry struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	Username     string    `json:"username"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   *int      `json:"resource_id,omitempty"`
	IPAddress    string    `json:"ip_address"`
	Success      bool      `json:"success"`
	ErrorMessage *string   `json:"error_message,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// ComplianceViolation represents a compliance violation
type ComplianceViolation struct {
	ID                string    `json:"id"`
	Type              string    `json:"type"`
	Severity          string    `json:"severity"`
	Description       string    `json:"description"`
	Reference         string    `json:"reference"` // e.g., SOC2-CC6.1
	DetectedAt        time.Time `json:"detected_at"`
	AffectedResources []string  `json:"affected_resources"`
	Remediation       string    `json:"remediation"`
	Status            string    `json:"status"` // "open", "in_progress", "resolved"
}

// Recommendation provides improvement suggestions
type Recommendation struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"` // "high", "medium", "low"
	Category    string   `json:"category"`
	Impact      string   `json:"impact"`
	Effort      string   `json:"effort"` // "high", "medium", "low"
	Actions     []string `json:"actions"`
}

// ComplianceMetrics contains various compliance metrics
type ComplianceMetrics struct {
	AuthenticationMetrics AuthMetrics  `json:"authentication"`
	AuthorizationMetrics  AuthzMetrics `json:"authorization"`
	DataProtectionMetrics DataMetrics  `json:"data_protection"`
	AuditMetrics          AuditMetrics `json:"audit"`
	InfrastructureMetrics InfraMetrics `json:"infrastructure"`
}

// AuthMetrics tracks authentication-related metrics
type AuthMetrics struct {
	TotalLogins            int     `json:"total_logins"`
	FailedLogins           int     `json:"failed_logins"`
	LoginSuccessRate       float64 `json:"login_success_rate"`
	MFAEnabledUsers        int     `json:"mfa_enabled_users"`
	MFAEnabledRate         float64 `json:"mfa_enabled_rate"`
	APIKeyCount            int     `json:"api_key_count"`
	ExpiredAPIKeys         int     `json:"expired_api_keys"`
	ActiveSessions         int     `json:"active_sessions"`
	RevokedSessions        int     `json:"revoked_sessions"`
	TempCredentialsCreated int     `json:"temp_credentials_created"`
	TempCredentialsExpired int     `json:"temp_credentials_expired"`
}

// AuthzMetrics tracks authorization-related metrics
type AuthzMetrics struct {
	TotalAccessRequests         int     `json:"total_access_requests"`
	DeniedAccessRequests        int     `json:"denied_access_requests"`
	AccessDenyRate              float64 `json:"access_deny_rate"`
	PrivilegeEscalationAttempts int     `json:"privilege_escalation_attempts"`
	RBACChanges                 int     `json:"rbac_changes"`
	PermissionGrants            int     `json:"permission_grants"`
	PermissionRevocations       int     `json:"permission_revocations"`
}

// DataMetrics tracks data protection metrics
type DataMetrics struct {
	EncryptedResources int     `json:"encrypted_resources"`
	TotalResources     int     `json:"total_resources"`
	EncryptionRate     float64 `json:"encryption_rate"`
	SecretsScanned     int     `json:"secrets_scanned"`
	SecretsFound       int     `json:"secrets_found"`
	DataAccessEvents   int     `json:"data_access_events"`
	DataExportEvents   int     `json:"data_export_events"`
	DataDeleteEvents   int     `json:"data_delete_events"`
}

// AuditMetrics tracks audit-related metrics
type AuditMetrics struct {
	TotalAuditEntries       int     `json:"total_audit_entries"`
	AuditLogStorageGB       float64 `json:"audit_log_storage_gb"`
	AuditRetentionDays      int     `json:"audit_retention_days"`
	AuditLogIntegrityChecks int     `json:"audit_log_integrity_checks"`
	IntegrityCheckFailures  int     `json:"integrity_check_failures"`
}

// InfraMetrics tracks infrastructure security metrics
type InfraMetrics struct {
	NetworkIsolatedWorkspaces int     `json:"network_isolated_workspaces"`
	TotalWorkspaces           int     `json:"total_workspaces"`
	IsolationRate             float64 `json:"isolation_rate"`
	FirewallRulesCount        int     `json:"firewall_rules_count"`
	SecurityScansPerformed    int     `json:"security_scans_performed"`
	VulnerabilitiesFound      int     `json:"vulnerabilities_found"`
	CriticalVulns             int     `json:"critical_vulnerabilities"`
	HighVulns                 int     `json:"high_vulnerabilities"`
}

// ComplianceReportGenerator generates compliance reports
type ComplianceReportGenerator struct {
	auditLogPath string
	storagePath  string
}

// NewComplianceReportGenerator creates a new report generator
func NewComplianceReportGenerator(auditLogPath, storagePath string) *ComplianceReportGenerator {
	return &ComplianceReportGenerator{
		auditLogPath: auditLogPath,
		storagePath:  storagePath,
	}
}

// GenerateReport generates a compliance report for the given period
func (g *ComplianceReportGenerator) GenerateReport(orgID *int, startDate, endDate time.Time) (*ComplianceReport, error) {
	report := &ComplianceReport{
		ID:          fmt.Sprintf("compliance-%d", time.Now().UnixNano()),
		GeneratedAt: time.Now(),
		Period: ReportPeriod{
			StartDate: startDate,
			EndDate:   endDate,
		},
		Organization: func() *string {
			if orgID != nil {
				s := fmt.Sprintf("%d", *orgID)
				return &s
			}
			return nil
		}(),
	}

	// Load audit logs
	auditLogs, err := g.loadAuditLogs(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to load audit logs: %w", err)
	}

	report.AuditLogs = auditLogs
	report.Summary.TotalAuditEntries = len(auditLogs)

	// Analyze for violations
	violations := g.analyzeViolations(auditLogs)
	report.Violations = violations
	report.Summary.TotalViolations = len(violations)

	// Count by severity
	for _, v := range violations {
		switch v.Severity {
		case "critical":
			report.Summary.CriticalViolations++
		case "high":
			report.Summary.HighViolations++
		case "medium":
			report.Summary.MediumViolations++
		case "low":
			report.Summary.LowViolations++
		}
	}

	// Calculate compliance score
	report.Summary.ComplianceScore = g.calculateComplianceScore(violations)

	// Generate metrics
	report.Metrics = g.generateMetrics(auditLogs, violations)

	// Generate recommendations
	report.Recommendations = g.generateRecommendations(violations, report.Metrics)

	return report, nil
}

// loadAuditLogs loads audit logs from the audit log system
func (g *ComplianceReportGenerator) loadAuditLogs(startDate, endDate time.Time) ([]AuditLogEntry, error) {
	// This would integrate with the audit log handler
	// For now, return empty list
	// In production, this would query the audit_log table
	return []AuditLogEntry{}, nil
}

// analyzeViolations analyzes audit logs for compliance violations
func (g *ComplianceReportGenerator) analyzeViolations(auditLogs []AuditLogEntry) []ComplianceViolation {
	var violations []ComplianceViolation

	// Check for failed authentication attempts (potential brute force)
	failedLogins := 0
	userFailedLogins := make(map[int]int)

	for _, log := range auditLogs {
		if log.Action == "login_failed" {
			failedLogins++
			userFailedLogins[log.UserID]++
		}
	}

	// Flag users with excessive failed logins
	for userID, count := range userFailedLogins {
		if count > 10 {
			violations = append(violations, ComplianceViolation{
				ID:          fmt.Sprintf("VULN-AUTH-%d", len(violations)+1),
				Type:        "excessive_failed_logins",
				Severity:    "medium",
				Description: fmt.Sprintf("User %d has %d failed login attempts", userID, count),
				Reference:   "SOC2-CC6.1",
				DetectedAt:  time.Now(),
				Remediation: "Review user account and consider temporary lockout",
				Status:      "open",
			})
		}
	}

	// Check for unauthorized access attempts
	for _, log := range auditLogs {
		if !log.Success && (log.Action == "resource_access" || log.Action == "permission_check") {
			violations = append(violations, ComplianceViolation{
				ID:                fmt.Sprintf("VULN-AUTHZ-%d", len(violations)+1),
				Type:              "unauthorized_access_attempt",
				Severity:          "high",
				Description:       fmt.Sprintf("Unauthorized access attempt by user %d on %s", log.UserID, log.ResourceType),
				Reference:         "SOC2-CC6.6",
				DetectedAt:        log.Timestamp,
				AffectedResources: []string{fmt.Sprintf("%s:%d", log.ResourceType, *log.ResourceID)},
				Remediation:       "Review access controls and user permissions",
				Status:            "open",
			})
		}
	}

	return violations
}

// calculateComplianceScore calculates the overall compliance score
func (g *ComplianceReportGenerator) calculateComplianceScore(violations []ComplianceViolation) float64 {
	if len(violations) == 0 {
		return 100.0
	}

	// Deduct points based on severity
	score := 100.0
	for _, v := range violations {
		switch v.Severity {
		case "critical":
			score -= 15
		case "high":
			score -= 10
		case "medium":
			score -= 5
		case "low":
			score -= 2
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

// generateMetrics generates compliance metrics
func (g *ComplianceReportGenerator) generateMetrics(auditLogs []AuditLogEntry, violations []ComplianceViolation) ComplianceMetrics {
	metrics := ComplianceMetrics{}

	// Count authentication metrics
	for _, log := range auditLogs {
		switch log.Action {
		case "login_success":
			metrics.AuthenticationMetrics.TotalLogins++
		case "login_failed":
			metrics.AuthenticationMetrics.FailedLogins++
		}
	}

	if metrics.AuthenticationMetrics.TotalLogins > 0 {
		metrics.AuthenticationMetrics.LoginSuccessRate = float64(metrics.AuthenticationMetrics.TotalLogins-metrics.AuthenticationMetrics.FailedLogins) / float64(metrics.AuthenticationMetrics.TotalLogins) * 100
	}

	// Count authorization metrics
	for _, log := range auditLogs {
		if !log.Success && (log.Action == "resource_access" || log.Action == "permission_check") {
			metrics.AuthorizationMetrics.DeniedAccessRequests++
		}
	}

	metrics.AuthorizationMetrics.TotalAccessRequests = len(auditLogs)
	if metrics.AuthorizationMetrics.TotalAccessRequests > 0 {
		metrics.AuthorizationMetrics.AccessDenyRate = float64(metrics.AuthorizationMetrics.DeniedAccessRequests) / float64(metrics.AuthorizationMetrics.TotalAccessRequests) * 100
	}

	// Audit metrics
	metrics.AuditMetrics.TotalAuditEntries = len(auditLogs)
	metrics.AuditMetrics.AuditRetentionDays = 90 // Default retention

	return metrics
}

// generateRecommendations generates compliance recommendations
func (g *ComplianceReportGenerator) generateRecommendations(violations []ComplianceViolation, metrics ComplianceMetrics) []Recommendation {
	var recommendations []Recommendation

	// Recommendation based on violations
	if metrics.AuthenticationMetrics.FailedLogins > 100 {
		recommendations = append(recommendations, Recommendation{
			ID:          "REC-001",
			Title:       "Implement Account Lockout Policy",
			Description: "High number of failed login attempts detected",
			Priority:    "high",
			Category:    "authentication",
			Impact:      "Prevents brute force attacks",
			Effort:      "low",
			Actions:     []string{"Implement account lockout after 5 failed attempts", "Add CAPTCHA after 3 failed attempts", "Send security alerts to users"}})
	}

	if metrics.AuthenticationMetrics.LoginSuccessRate < 90 {
		recommendations = append(recommendations, Recommendation{
			ID:          "REC-002",
			Title:       "Improve Authentication Success Rate",
			Description: "Login success rate below 90%",
			Priority:    "medium",
			Category:    "authentication",
			Impact:      "Better user experience",
			Effort:      "medium",
			Actions:     []string{"Review authentication flow", "Add password reset functionality", "Implement SSO options"}})
	}

	if len(violations) > 10 {
		recommendations = append(recommendations, Recommendation{
			ID:          "REC-003",
			Title:       "Address Compliance Violations",
			Description: "Multiple compliance violations detected",
			Priority:    "high",
			Category:    "compliance",
			Impact:      "Improve security posture",
			Effort:      "high",
			Actions:     []string{"Review all violations", "Create remediation plan", "Implement controls"}})
	}

	return recommendations
}

// SaveReport saves the compliance report to a file
func (g *ComplianceReportGenerator) SaveReport(report *ComplianceReport) error {
	// Create reports directory
	reportDir := filepath.Join(g.storagePath, "compliance-reports")
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	// Generate filename
	filename := filepath.Join(reportDir, fmt.Sprintf("compliance-%s.json", report.ID))

	// Marshal to JSON
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	// Write file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	return nil
}

// ExportReport exports the report in various formats
func (g *ComplianceReportGenerator) ExportReport(report *ComplianceReport, format string) ([]byte, error) {
	switch format {
	case "json":
		return json.MarshalIndent(report, "", "  ")
	case "csv":
		return g.exportCSV(report)
	case "html":
		return g.exportHTML(report)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// exportCSV exports report as CSV
func (g *ComplianceReportGenerator) exportCSV(report *ComplianceReport) ([]byte, error) {
	// Simplified CSV export for violations
	var csv string
	csv += "ID,Type,Severity,Description,DetectedAt,Status\n"

	for _, v := range report.Violations {
		csv += fmt.Sprintf("%s,%s,%s,\"%s\",%s,%s\n",
			v.ID, v.Type, v.Severity, v.Description,
			v.DetectedAt.Format(time.RFC3339), v.Status)
	}

	return []byte(csv), nil
}

// exportHTML exports report as HTML
func (g *ComplianceReportGenerator) exportHTML(report *ComplianceReport) ([]byte, error) {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Compliance Report - %s</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { background: #f5f5f5; padding: 20px; border-radius: 8px; }
        .score { font-size: 48px; font-weight: bold; }
        .good { color: #28a745; }
        .warning { color: #ffc107; }
        .bad { color: #dc3545; }
	table { width: 100%%; border-collapse: collapse; margin-top: 20px; }
        th, td { border: 1px solid #ddd; padding: 12px; text-align: left; }
        th { background: #f5f5f5; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Compliance Report</h1>
        <p>Generated: %s</p>
        <p>Period: %s to %s</p>
    </div>
    
    <h2>Compliance Score</h2>
    <p class="%s">%0.1f/100</p>
    
    <h2>Summary</h2>
    <table>
        <tr><th>Metric</th><th>Value</th></tr>
        <tr><td>Total Audit Entries</td><td>%d</td></tr>
        <tr><td>Total Violations</td><td>%d</td></tr>
        <tr><td>Critical Violations</td><td>%d</td></tr>
        <tr><td>High Violations</td><td>%d</td></tr>
    </table>
    
    <h2>Violations</h2>
    <table>
        <tr><th>ID</th><th>Type</th><th>Severity</th><th>Description</th></tr>
        %s
    </table>
</body>
</html>`,
		report.ID,
		report.GeneratedAt.Format(time.RFC3339),
		report.Period.StartDate.Format("2006-01-02"),
		report.Period.EndDate.Format("2006-01-02"),
		g.getScoreClass(report.Summary.ComplianceScore),
		report.Summary.ComplianceScore,
		report.Summary.TotalAuditEntries,
		report.Summary.TotalViolations,
		report.Summary.CriticalViolations,
		report.Summary.HighViolations,
		g.generateViolationRows(report.Violations),
	)

	return []byte(html), nil
}

// getScoreClass returns CSS class based on score
func (g *ComplianceReportGenerator) getScoreClass(score float64) string {
	if score >= 80 {
		return "good"
	} else if score >= 60 {
		return "warning"
	}
	return "bad"
}

// generateViolationRows generates HTML table rows for violations
func (g *ComplianceReportGenerator) generateViolationRows(violations []ComplianceViolation) string {
	var rows string
	for _, v := range violations {
		rows += fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
			v.ID, v.Type, v.Severity, v.Description)
	}
	return rows
}

// GetComplianceScore returns the current compliance score
func (g *ComplianceReportGenerator) GetComplianceScore(orgID *int) (float64, error) {
	now := time.Now()
	startDate := now.AddDate(0, -1, 0) // Last month

	report, err := g.GenerateReport(orgID, startDate, now)
	if err != nil {
		return 0, err
	}

	return report.Summary.ComplianceScore, nil
}
