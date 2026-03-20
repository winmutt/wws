package scanning

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ScanResult represents the result of a security scan
type ScanResult struct {
	ID              string          `json:"id"`
	Target          string          `json:"target"`
	ScanType        string          `json:"scan_type"`
	Status          string          `json:"status"`
	StartedAt       time.Time       `json:"started_at"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	Summary         ScanSummary     `json:"summary"`
	Error           *string         `json:"error,omitempty"`
}

// Vulnerability represents a single security vulnerability
type Vulnerability struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"` // "critical", "high", "medium", "low", "info"
	CVE         string    `json:"cve,omitempty"`
	CVSS        float64   `json:"cvss,omitempty"`
	Package     string    `json:"package,omitempty"`
	Version     string    `json:"version,omitempty"`
	Location    string    `json:"location,omitempty"`
	Line        int       `json:"line,omitempty"`
	Solution    string    `json:"solution,omitempty"`
	References  []string  `json:"references,omitempty"`
	DetectedAt  time.Time `json:"detected_at"`
}

// ScanSummary provides a summary of scan results
type ScanSummary struct {
	TotalVulnerabilities int           `json:"total"`
	Critical             int           `json:"critical"`
	High                 int           `json:"high"`
	Medium               int           `json:"medium"`
	Low                  int           `json:"low"`
	Info                 int           `json:"info"`
	ScanDuration         time.Duration `json:"scan_duration"`
}

// ScanConfig represents configuration for security scanning
type ScanConfig struct {
	EnableSAST        bool     `json:"enable_sast"`
	EnableDAST        bool     `json:"enable_dast"`
	EnableSCA         bool     `json:"enable_sca"`
	EnableSecretScan  bool     `json:"enable_secret_scan"`
	SeverityThreshold string   `json:"severity_threshold"`
	ExcludedPaths     []string `json:"excluded_paths"`
	IncludePaths      []string `json:"include_paths"`
	OutputFormat      string   `json:"output_format"`
	ReportDirectory   string   `json:"report_directory"`
}

// DefaultScanConfig returns default scan configuration
func DefaultScanConfig() *ScanConfig {
	return &ScanConfig{
		EnableSAST:        true,
		EnableDAST:        false, // Disabled by default - requires running target
		EnableSCA:         true,
		EnableSecretScan:  true,
		SeverityThreshold: "low",
		ExcludedPaths:     []string{"vendor", "node_modules", ".git", "dist", "build"},
		OutputFormat:      "json",
		ReportDirectory:   "./security-reports",
	}
}

// SecurityScanner performs security scans
type SecurityScanner struct {
	config *ScanConfig
}

// NewSecurityScanner creates a new security scanner
func NewSecurityScanner(config *ScanConfig) *SecurityScanner {
	if config == nil {
		config = DefaultScanConfig()
	}
	return &SecurityScanner{
		config: config,
	}
}

// Scan performs a full security scan on the target directory
func (s *SecurityScanner) Scan(target string) (*ScanResult, error) {
	result := &ScanResult{
		ID:        fmt.Sprintf("scan-%d", time.Now().UnixNano()),
		Target:    target,
		ScanType:  "full",
		Status:    "running",
		StartedAt: time.Now(),
	}

	var allVulnerabilities []Vulnerability

	// Run SAST scan
	if s.config.EnableSAST {
		sastVulns, err := s.runSASTScan(target)
		if err != nil {
			s.logError(fmt.Sprintf("SAST scan failed: %v", err))
		}
		allVulnerabilities = append(allVulnerabilities, sastVulns...)
	}

	// Run SCA scan (dependency scanning)
	if s.config.EnableSCA {
		scaVulns, err := s.runSCAScan(target)
		if err != nil {
			s.logError(fmt.Sprintf("SCA scan failed: %v", err))
		}
		allVulnerabilities = append(allVulnerabilities, scaVulns...)
	}

	// Run secret scanning
	if s.config.EnableSecretScan {
		secretVulns, err := s.runSecretScan(target)
		if err != nil {
			s.logError(fmt.Sprintf("Secret scan failed: %v", err))
		}
		allVulnerabilities = append(allVulnerabilities, secretVulns...)
	}

	// Filter by severity threshold
	filteredVulns := s.filterBySeverity(allVulnerabilities)

	// Populate result
	result.Vulnerabilities = filteredVulns
	result.Summary = s.generateSummary(filteredVulns)
	now := time.Now()
	result.CompletedAt = &now
	result.Status = "completed"

	// Save report
	if err := s.saveReport(result); err != nil {
		s.logError(fmt.Sprintf("Failed to save report: %v", err))
	}

	return result, nil
}

// runSASTScan performs Static Application Security Testing
func (s *SecurityScanner) runSASTScan(target string) ([]Vulnerability, error) {
	var vulnerabilities []Vulnerability

	// Check for gosec (Go security scanner)
	if s.commandExists("gosec") {
		vulns, err := s.runGosec(target)
		if err != nil {
			return vulnerabilities, err
		}
		vulnerabilities = append(vulnerabilities, vulns...)
	}

	// Check for semgrep (multi-language scanner)
	if s.commandExists("semgrep") {
		vulns, err := s.runSemgrep(target)
		if err != nil {
			return vulnerabilities, err
		}
		vulnerabilities = append(vulnerabilities, vulns...)
	}

	// Fallback: Basic pattern-based scanning
	if len(vulnerabilities) == 0 {
		vulns := s.basicPatternScan(target)
		vulnerabilities = append(vulnerabilities, vulns...)
	}

	return vulnerabilities, nil
}

// runGosec runs gosec security scanner for Go code
func (s *SecurityScanner) runGosec(target string) ([]Vulnerability, error) {
	// Run gosec and parse output
	cmd := exec.Command("gosec", "-fmt", "json", "-out", "-", "./...")
	cmd.Dir = target

	output, err := cmd.CombinedOutput()
	if err != nil {
		// gosec returns non-zero when vulnerabilities are found
		// This is expected behavior
	}

	// Parse gosec JSON output
	var gosecResult map[string]interface{}
	if err := json.Unmarshal(output, &gosecResult); err != nil {
		return nil, fmt.Errorf("failed to parse gosec output: %w", err)
	}

	// Extract issues
	var vulnerabilities []Vulnerability
	if issues, ok := gosecResult["Issues"].([]interface{}); ok {
		for _, issue := range issues {
			if issueMap, ok := issue.(map[string]interface{}); ok {
				vuln := Vulnerability{
					ID:          fmt.Sprintf("GOSSEC-%d", len(vulnerabilities)+1),
					Title:       getString(issueMap, "issue"),
					Description: getString(issueMap, "details"),
					Severity:    strings.ToLower(getString(issueMap, "severity")),
					Location:    getString(issueMap, "file"),
					Line:        getInt(issueMap, "line"),
					CVSS:        getFloat(issueMap, "confidence"),
					DetectedAt:  time.Now(),
				}
				vulnerabilities = append(vulnerabilities, vuln)
			}
		}
	}

	return vulnerabilities, nil
}

// runSemgrep runs semgrep for multi-language scanning
func (s *SecurityScanner) runSemgrep(target string) ([]Vulnerability, error) {
	cmd := exec.Command("semgrep", "--config", "auto", "--json", "-o", "-")
	cmd.Dir = target

	output, err := cmd.CombinedOutput()
	if err != nil && len(output) == 0 {
		return nil, fmt.Errorf("semgrep not configured or failed: %w", err)
	}

	if len(output) == 0 {
		return nil, nil
	}

	var semgrepResult map[string]interface{}
	if err := json.Unmarshal(output, &semgrepResult); err != nil {
		return nil, fmt.Errorf("failed to parse semgrep output: %w", err)
	}

	var vulnerabilities []Vulnerability
	if results, ok := semgrepResult["results"].([]interface{}); ok {
		for _, result := range results {
			if resultMap, ok := result.(map[string]interface{}); ok {
				vuln := Vulnerability{
					ID:          getString(resultMap, "check_id"),
					Title:       getString(resultMap, "extra", "message"),
					Description: getString(resultMap, "extra", "metavars"),
					Severity:    strings.ToLower(getString(resultMap, "extra", "severity")),
					Location:    getString(resultMap, "path"),
					DetectedAt:  time.Now(),
				}
				vulnerabilities = append(vulnerabilities, vuln)
			}
		}
	}

	return vulnerabilities, nil
}

// basicPatternScan performs basic pattern-based security scanning
func (s *SecurityScanner) basicPatternScan(target string) []Vulnerability {
	var vulnerabilities []Vulnerability

	// Common security patterns to check
	patterns := map[string]string{
		`password\s*[=:]\s*["'][^"']+["']`:    "Hardcoded password detected",
		`api[_-]?key\s*[=:]\s*["'][^"']+["']`: "Hardcoded API key detected",
		`secret\s*[=:]\s*["'][^"']+["']`:      "Hardcoded secret detected",
		`aws[_-]?access[_-]?key`:              "AWS access key pattern detected",
		`PRIVATE\s+KEY`:                       "Private key detected",
		`BEGIN\s+(RSA\s+)?PRIVATE\s+KEY`:      "Private key block detected",
	}

	// Scan Go files
	goFiles := s.findFiles(target, "*.go")
	for _, file := range goFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		for pattern, description := range patterns {
			if strings.Contains(string(content), strings.Split(pattern, `\s`)[0]) {
				vuln := Vulnerability{
					ID:          fmt.Sprintf("PATTERN-%d", len(vulnerabilities)+1),
					Title:       description,
					Description: fmt.Sprintf("Potential security issue in %s", file),
					Severity:    "medium",
					Location:    file,
					DetectedAt:  time.Now(),
				}
				vulnerabilities = append(vulnerabilities, vuln)
			}
		}
	}

	return vulnerabilities
}

// runSCAScan performs Software Composition Analysis (dependency scanning)
func (s *SecurityScanner) runSCAScan(target string) ([]Vulnerability, error) {
	var vulnerabilities []Vulnerability

	// Check for go mod scanning
	goModPath := filepath.Join(target, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		vulns, err := s.scanGoModules(target)
		if err != nil {
			return vulnerabilities, err
		}
		vulnerabilities = append(vulnerabilities, vulns...)
	}

	// Check for npm scanning
	packageJSONPath := filepath.Join(target, "package.json")
	if _, err := os.Stat(packageJSONPath); err == nil {
		vulns, err := s.scanNPM(target)
		if err != nil {
			return vulnerabilities, err
		}
		vulnerabilities = append(vulnerabilities, vulns...)
	}

	return vulnerabilities, nil
}

// scanGoModules scans Go dependencies for vulnerabilities
func (s *SecurityScanner) scanGoModules(target string) ([]Vulnerability, error) {
	var vulnerabilities []Vulnerability

	// Try govulncheck if available
	if s.commandExists("govulncheck") {
		cmd := exec.Command("govulncheck", "./...")
		cmd.Dir = target
		output, err := cmd.CombinedOutput()

		if err != nil && len(output) > 0 {
			// Parse govulncheck output
			lines := strings.Split(string(output), "\n")
			for i, line := range lines {
				if strings.Contains(strings.ToLower(line), "vulnerability") ||
					strings.Contains(strings.ToLower(line), "found") {
					vuln := Vulnerability{
						ID:          fmt.Sprintf("GOVULN-%d", len(vulnerabilities)+1),
						Title:       "Go dependency vulnerability",
						Description: line,
						Severity:    "high",
						Location:    "go.mod",
						Line:        i + 1,
						DetectedAt:  time.Now(),
					}
					vulnerabilities = append(vulnerabilities, vuln)
				}
			}
		}
	}

	return vulnerabilities, nil
}

// scanNPM scans NPM dependencies for vulnerabilities
func (s *SecurityScanner) scanNPM(target string) ([]Vulnerability, error) {
	var vulnerabilities []Vulnerability

	if s.commandExists("npm") {
		cmd := exec.Command("npm", "audit", "--json")
		cmd.Dir = target
		output, err := cmd.CombinedOutput()

		if err == nil && len(output) > 0 {
			var auditResult map[string]interface{}
			if err := json.Unmarshal(output, &auditResult); err == nil {
				if vulnerabilitiesData, ok := auditResult["advisories"].(map[string]interface{}); ok {
					for id, adv := range vulnerabilitiesData {
						if advMap, ok := adv.(map[string]interface{}); ok {
							vuln := Vulnerability{
								ID:          fmt.Sprintf("NPM-%s", id),
								Title:       getString(advMap, "title"),
								Description: getString(advMap, "description"),
								Severity:    strings.ToLower(getString(advMap, "severity")),
								Package:     getString(advMap, "module_name"),
								Version:     getString(advMap, "vulnerable_versions"),
								Solution:    getString(advMap, "recommendation"),
								DetectedAt:  time.Now(),
							}
							vulnerabilities = append(vulnerabilities, vuln)
						}
					}
				}
			}
		}
	}

	return vulnerabilities, nil
}

// runSecretScan performs secret detection scanning
func (s *SecurityScanner) runSecretScan(target string) ([]Vulnerability, error) {
	var vulnerabilities []Vulnerability

	// Check for gitleaks if available
	if s.commandExists("gitleaks") {
		cmd := exec.Command("gitleaks", "detect", "--source", target, "--no-git", "-f", "json")
		output, err := cmd.CombinedOutput()

		if err == nil && len(output) > 0 {
			var gitleaksResults []map[string]interface{}
			if err := json.Unmarshal(output, &gitleaksResults); err == nil {
				for _, result := range gitleaksResults {
					vuln := Vulnerability{
						ID:          getString(result, "RuleID"),
						Title:       getString(result, "Description"),
						Description: fmt.Sprintf("Secret detected: %s", getString(result, "Match")),
						Severity:    "critical",
						Location:    getString(result, "File"),
						Line:        getInt(result, "StartLine"),
						DetectedAt:  time.Now(),
					}
					vulnerabilities = append(vulnerabilities, vuln)
				}
			}
		}
	}

	// Fallback: Basic secret pattern detection
	if len(vulnerabilities) == 0 {
		vulns := s.basicSecretScan(target)
		vulnerabilities = append(vulnerabilities, vulns...)
	}

	return vulnerabilities, nil
}

// basicSecretScan performs basic secret pattern detection
func (s *SecurityScanner) basicSecretScan(target string) []Vulnerability {
	var vulnerabilities []Vulnerability

	// Secret patterns
	secretPatterns := map[string]string{
		`AKIA[0-9A-Z]{16}`:                         "AWS Access Key ID",
		`-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----`: "Private Key",
		`ghp_[0-9a-zA-Z]{36}`:                      "GitHub Personal Access Token",
		`xox[baprs]-[0-9a-zA-Z]{10,48}`:            "Slack Token",
		`AIza[0-9A-Za-z\-_]{35}`:                   "Google API Key",
	}

	files := s.findFiles(target, "*.go", "*.js", "*.ts", "*.py", "*.yml", "*.yaml", "*.env")

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		contentStr := string(content)
		for pattern, description := range secretPatterns {
			if strings.Contains(contentStr, strings.Split(pattern, `[`)[0]) {
				vuln := Vulnerability{
					ID:          fmt.Sprintf("SECRET-%d", len(vulnerabilities)+1),
					Title:       description,
					Description: fmt.Sprintf("Potential secret detected in %s", file),
					Severity:    "critical",
					Location:    file,
					DetectedAt:  time.Now(),
				}
				vulnerabilities = append(vulnerabilities, vuln)
			}
		}
	}

	return vulnerabilities
}

// filterBySeverity filters vulnerabilities by severity threshold
func (s *SecurityScanner) filterBySeverity(vulnerabilities []Vulnerability) []Vulnerability {
	threshold := s.getSeverityLevel(s.config.SeverityThreshold)
	var filtered []Vulnerability

	for _, vuln := range vulnerabilities {
		vulnLevel := s.getSeverityLevel(vuln.Severity)
		if vulnLevel >= threshold {
			filtered = append(filtered, vuln)
		}
	}

	return filtered
}

// getSeverityLevel returns numeric severity level
func (s *SecurityScanner) getSeverityLevel(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

// generateSummary generates a summary of vulnerabilities
func (s *SecurityScanner) generateSummary(vulnerabilities []Vulnerability) ScanSummary {
	summary := ScanSummary{
		TotalVulnerabilities: len(vulnerabilities),
	}

	for _, vuln := range vulnerabilities {
		switch strings.ToLower(vuln.Severity) {
		case "critical":
			summary.Critical++
		case "high":
			summary.High++
		case "medium":
			summary.Medium++
		case "low":
			summary.Low++
		default:
			summary.Info++
		}
	}

	return summary
}

// saveReport saves the scan report to a file
func (s *SecurityScanner) saveReport(result *ScanResult) error {
	// Create report directory if it doesn't exist
	if err := os.MkdirAll(s.config.ReportDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	// Generate filename
	filename := fmt.Sprintf("%s/%s.%s", s.config.ReportDirectory, result.ID, s.config.OutputFormat)

	// Marshal to JSON
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	// Write file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	return nil
}

// findFiles finds files matching patterns in a directory
func (s *SecurityScanner) findFiles(dir string, patterns ...string) []string {
	var files []string

	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(dir, "**", pattern))
		if err == nil {
			files = append(files, matches...)
		}

		// Also check root directory
		rootMatches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err == nil {
			files = append(files, rootMatches...)
		}
	}

	return files
}

// commandExists checks if a command is available
func (s *SecurityScanner) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// logError logs an error message
func (s *SecurityScanner) logError(msg string) {
	// In production, use proper logging
	fmt.Fprintf(os.Stderr, "[SecurityScanner] %s\n", msg)
}

// Helper functions for parsing JSON
func getString(m map[string]interface{}, keys ...string) string {
	var current interface{} = m
	for _, key := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[key]
		} else {
			return ""
		}
	}
	if s, ok := current.(string); ok {
		return s
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		}
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if val, ok := v.(float64); ok {
			return val
		}
	}
	return 0
}
