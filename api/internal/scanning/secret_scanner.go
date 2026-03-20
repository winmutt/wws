package scanning

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SecretScanResult represents the result of a secret scan
type SecretScanResult struct {
	FilesScanned int             `json:"files_scanned"`
	SecretsFound int             `json:"secrets_found"`
	Findings     []SecretFinding `json:"findings"`
	SkippedFiles []string        `json:"skipped_files,omitempty"`
}

// SecretFinding represents a single secret finding
type SecretFinding struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	SecretType string `json:"secret_type"`
	Match      string `json:"match"`
	Severity   string `json:"severity"`
	RuleID     string `json:"rule_id"`
}

// SecretScanner scans files for secrets
type SecretScanner struct {
	patterns map[string]*regexp.Regexp
	severity map[string]string
	ruleIDs  map[string]string
}

// NewSecretScanner creates a new secret scanner
func NewSecretScanner() *SecretScanner {
	scanner := &SecretScanner{
		patterns: make(map[string]*regexp.Regexp),
		severity: make(map[string]string),
		ruleIDs:  make(map[string]string),
	}

	// Initialize patterns
	scanner.initPatterns()

	return scanner
}

// initPatterns initializes the secret detection patterns
func (s *SecretScanner) initPatterns() {
	// AWS credentials
	s.patterns["aws_access_key"] = regexp.MustCompile(`AKIA[0-9A-Z]{16}`)
	s.severity["aws_access_key"] = "critical"
	s.ruleIDs["aws_access_key"] = "AWS-ACCESS-KEY"

	s.patterns["aws_secret_key"] = regexp.MustCompile(`(?i)aws[_\-\s]?secret[_\-\s]?access[_\-\s]?key[_\s]*[=:]\s*["']?([0-9a-zA-Z/+=]{40})["']?`)
	s.severity["aws_secret_key"] = "critical"
	s.ruleIDs["aws_secret_key"] = "AWS-SECRET-KEY"

	// GitHub tokens
	s.patterns["github_pat"] = regexp.MustCompile(`ghp_[0-9a-zA-Z]{36}`)
	s.severity["github_pat"] = "critical"
	s.ruleIDs["github_pat"] = "GITHUB-PAT"

	s.patterns["github_oauth"] = regexp.MustCompile(`gho_[0-9a-zA-Z]{36}`)
	s.severity["github_oauth"] = "critical"
	s.ruleIDs["github_oauth"] = "GITHUB-OAUTH"

	s.patterns["github_app"] = regexp.MustCompile(`ghu_[0-9a-zA-Z]{36}`)
	s.severity["github_app"] = "critical"
	s.ruleIDs["github_app"] = "GITHUB-APP"

	s.patterns["github_refresh"] = regexp.MustCompile(`ghr_[0-9A-Za-z]{36}`)
	s.severity["github_refresh"] = "critical"
	s.ruleIDs["github_refresh"] = "GITHUB-REFRESH"

	// Slack tokens
	s.patterns["slack_token"] = regexp.MustCompile(`xox[baprs]-[0-9a-zA-Z]{10,48}`)
	s.severity["slack_token"] = "critical"
	s.ruleIDs["slack_token"] = "SLACK-TOKEN"

	// Google API keys
	s.patterns["google_api_key"] = regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`)
	s.severity["google_api_key"] = "high"
	s.ruleIDs["google_api_key"] = "GOOGLE-API-KEY"

	// Stripe keys
	s.patterns["stripe_secret"] = regexp.MustCompile(`sk_live_[0-9a-zA-Z]{24}`)
	s.severity["stripe_secret"] = "critical"
	s.ruleIDs["stripe_secret"] = "STRIPE-SECRET"

	s.patterns["stripe_restricted"] = regexp.MustCompile(`rk_live_[0-9a-zA-Z]{24}`)
	s.severity["stripe_restricted"] = "critical"
	s.ruleIDs["stripe_restricted"] = "STRIPE-RESTRICTED"

	// Private keys
	s.patterns["private_key"] = regexp.MustCompile(`-----BEGIN\s+(RSA\s+|EC\s+|DSA\s+|OPENSSH\s+)?PRIVATE\s+KEY-----`)
	s.severity["private_key"] = "critical"
	s.ruleIDs["private_key"] = "PRIVATE-KEY"

	// Generic secrets in code
	s.patterns["password_assignment"] = regexp.MustCompile(`(?i)password\s*[=:]\s*["'][^"']+["']`)
	s.severity["password_assignment"] = "high"
	s.ruleIDs["password_assignment"] = "PASSWORD-ASSIGNMENT"

	s.patterns["api_key_assignment"] = regexp.MustCompile(`(?i)api[_-]?key\s*[=:]\s*["'][^"']+["']`)
	s.severity["api_key_assignment"] = "high"
	s.ruleIDs["api_key_assignment"] = "API-KEY-ASSIGNMENT"

	s.patterns["secret_assignment"] = regexp.MustCompile(`(?i)secret\s*[=:]\s*["'][^"']+["']`)
	s.severity["secret_assignment"] = "high"
	s.ruleIDs["secret_assignment"] = "SECRET-ASSIGNMENT"

	// Bearer tokens
	s.patterns["bearer_token"] = regexp.MustCompile(`Bearer\s+[0-9a-zA-Z\-_\.]+`)
	s.severity["bearer_token"] = "medium"
	s.ruleIDs["bearer_token"] = "BEARER-TOKEN"

	// JWT tokens
	s.patterns["jwt_token"] = regexp.MustCompile(`eyJ[a-zA-Z0-9\-_]+\.eyJ[a-zA-Z0-9\-_]+\.[a-zA-Z0-9\-_]+`)
	s.severity["jwt_token"] = "medium"
	s.ruleIDs["jwt_token"] = "JWT-TOKEN"

	// Generic API keys
	s.patterns["generic_api_key"] = regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[=:]\s*["']?[a-zA-Z0-9]{20,}["']?`)
	s.severity["generic_api_key"] = "medium"
	s.ruleIDs["generic_api_key"] = "GENERIC-API-KEY"
}

// ScanDirectory scans a directory for secrets
func (s *SecretScanner) ScanDirectory(dir string) (*SecretScanResult, error) {
	result := &SecretScanResult{}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip common non-code directories
			switch info.Name() {
			case "node_modules", "vendor", ".git", "dist", "build", ".cache":
				result.SkippedFiles = append(result.SkippedFiles, path)
				return filepath.SkipDir
			}
			return nil
		}

		// Skip binary files and non-text files
		if s.isBinaryFile(path) {
			result.SkippedFiles = append(result.SkippedFiles, path)
			return nil
		}

		// Scan the file
		findings, err := s.scanFile(path)
		if err != nil {
			return err
		}

		result.FilesScanned++
		result.Findings = append(result.Findings, findings...)

		return nil
	})

	if err != nil {
		return nil, err
	}

	result.SecretsFound = len(result.Findings)

	return result, nil
}

// ScanFile scans a single file for secrets
func (s *SecretScanner) scanFile(path string) ([]SecretFinding, error) {
	var findings []SecretFinding

	file, err := os.Open(path)
	if err != nil {
		return findings, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check each pattern
		for ruleID, pattern := range s.patterns {
			if matches := pattern.FindAllString(line, -1); len(matches) > 0 {
				finding := SecretFinding{
					File:       path,
					Line:       lineNum,
					SecretType: ruleID,
					Match:      s.redactSecret(matches[0]),
					Severity:   s.severity[ruleID],
					RuleID:     s.ruleIDs[ruleID],
				}
				findings = append(findings, finding)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return findings, err
	}

	return findings, nil
}

// isBinaryFile checks if a file is binary
func (s *SecretScanner) isBinaryFile(path string) bool {
	// Skip common binary file extensions
	binaryExtensions := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".pdf": true, ".zip": true, ".tar": true, ".gz": true,
		".exe": true, ".bin": true, ".so": true, ".dll": true,
		".ico": true, ".svg": true, ".webp": true, ".avif": true,
	}

	ext := strings.ToLower(filepath.Ext(path))
	if binaryExtensions[ext] {
		return true
	}

	// Check file content for binary indicators
	file, err := os.Open(path)
	if err != nil {
		return true
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil || n == 0 {
		return false
	}

	// Check for null bytes (common in binary files)
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return true
		}
	}

	return false
}

// redactSecret redacts most of the secret, showing only first and last few characters
func (s *SecretScanner) redactSecret(secret string) string {
	if len(secret) <= 8 {
		return "****"
	}

	prefix := secret[:4]
	suffix := secret[len(secret)-4:]

	return prefix + "****" + suffix
}

// ScanGitDiff scans the git diff for secrets (for pre-commit hooks)
func (s *SecretScanner) ScanGitDiff(staged bool) (*SecretScanResult, error) {
	result := &SecretScanResult{}

	// Get list of staged/modified files
	var files []string

	if staged {
		files = getStagedFiles()
	} else {
		files = getModifiedFiles()
	}

	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			continue
		}

		// Get file content (staged or working directory)
		content, err := getFileContent(file, staged)
		if err != nil || content == "" {
			continue
		}

		// Scan content
		findings := s.scanContent(file, content)
		result.FilesScanned++
		result.Findings = append(result.Findings, findings...)
	}

	result.SecretsFound = len(result.Findings)

	return result, nil
}

// scanContent scans content for secrets
func (s *SecretScanner) scanContent(filename, content string) []SecretFinding {
	var findings []SecretFinding

	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		for ruleID, pattern := range s.patterns {
			if matches := pattern.FindAllString(line, -1); len(matches) > 0 {
				finding := SecretFinding{
					File:       filename,
					Line:       lineNum + 1,
					SecretType: ruleID,
					Match:      s.redactSecret(matches[0]),
					Severity:   s.severity[ruleID],
					RuleID:     s.ruleIDs[ruleID],
				}
				findings = append(findings, finding)
			}
		}
	}

	return findings
}

// Helper functions for git operations
func getStagedFiles() []string {
	// This would use git diff --cached --name-only
	// For now, return empty
	return []string{}
}

func getModifiedFiles() []string {
	// This would use git diff --name-only
	// For now, return empty
	return []string{}
}

func getFileContent(file string, staged bool) (string, error) {
	if staged {
		// Get staged content using git show
		// For now, read from file system
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// PrintResults prints scan results in a human-readable format
func PrintResults(result *SecretScanResult) {
	fmt.Printf("\n🔍 Secret Scan Results\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Files scanned: %d\n", result.FilesScanned)
	fmt.Printf("Secrets found: %d\n", result.SecretsFound)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if len(result.Findings) == 0 {
		fmt.Println("✅ No secrets detected!")
		return
	}

	// Group by severity
	critical := 0
	high := 0
	medium := 0

	for _, finding := range result.Findings {
		switch finding.Severity {
		case "critical":
			critical++
		case "high":
			high++
		case "medium":
			medium++
		}
	}

	if critical > 0 {
		fmt.Printf("🚨 CRITICAL: %d issue(s)\n", critical)
	}
	if high > 0 {
		fmt.Printf("⚠️  HIGH: %d issue(s)\n", high)
	}
	if medium > 0 {
		fmt.Printf("ℹ️  MEDIUM: %d issue(s)\n", medium)
	}

	fmt.Println("\n📋 Findings:")
	for _, finding := range result.Findings {
		fmt.Printf("  [%s] %s:%d - %s\n",
			strings.ToUpper(finding.Severity),
			finding.File,
			finding.Line,
			finding.Match)
	}

	fmt.Println("\n💡 Recommendations:")
	fmt.Println("  • Remove secrets from code immediately")
	fmt.Println("  • Use environment variables or secrets managers")
	fmt.Println("  • Rotate any exposed credentials")
	fmt.Println("  • Add .gitignore rules for sensitive files")
}
