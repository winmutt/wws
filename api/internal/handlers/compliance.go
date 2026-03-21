package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"wws/api/internal/compliance"
)

var _ = time.Now // Suppress unused import if needed

// ComplianceHandler handles compliance reporting requests
type ComplianceHandler struct {
	reportGenerator *compliance.ComplianceReportGenerator
}

// NewComplianceHandler creates a new compliance handler
func NewComplianceHandler(reportGenerator *compliance.ComplianceReportGenerator) *ComplianceHandler {
	return &ComplianceHandler{
		reportGenerator: reportGenerator,
	}
}

// GenerateReportRequest defines the request for generating a report
type GenerateReportRequest struct {
	OrganizationID *int   `json:"organization_id,omitempty"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date"`
}

// GenerateReport handles compliance report generation
// @Summary Generate a compliance report
// @Description Generate a compliance report for a specified period
// @Tags compliance
// @Accept json
// @Produce json
// @Param request body GenerateReportRequest true "Report generation request"
// @Success 200 {object} compliance.ComplianceReport
// @Router /api/v1/compliance/report [post]
func (h *ComplianceHandler) GenerateReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GenerateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Parse dates
	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		http.Error(w, "Invalid start_date format (use RFC3339)", http.StatusBadRequest)
		return
	}

	endDate, err := time.Parse(time.RFC3339, req.EndDate)
	if err != nil {
		http.Error(w, "Invalid end_date format (use RFC3339)", http.StatusBadRequest)
		return
	}

	// Validate date range
	if endDate.Before(startDate) {
		http.Error(w, "end_date must be after start_date", http.StatusBadRequest)
		return
	}

	// Generate report
	report, err := h.reportGenerator.GenerateReport(req.OrganizationID, startDate, endDate)
	if err != nil {
		http.Error(w, "Failed to generate report: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Save report
	if err := h.reportGenerator.SaveReport(report); err != nil {
		http.Error(w, "Failed to save report: "+err.Error(), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, report)
}

// GetReport retrieves a saved compliance report
// @Summary Get a compliance report by ID
// @Description Retrieve a previously generated compliance report
// @Tags compliance
// @Produce json
// @Param report_id path string true "Report ID"
// @Success 200 {object} compliance.ComplianceReport
// @Router /api/v1/compliance/report/{report_id} [get]
func (h *ComplianceHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get report ID from URL path
	// URL format: /api/v1/compliance/report/{report_id}
	// This is handled by the router extracting the ID
	http.Error(w, "Not implemented - use report ID from path", http.StatusNotImplemented)
}

// ExportReportRequest defines the request for exporting a report
type ExportReportRequest struct {
	ReportID string `json:"report_id"`
	Format   string `json:"format"` // json, csv, html
}

// ExportReport exports a compliance report in the specified format
// @Summary Export a compliance report
// @Description Export a compliance report in JSON, CSV, or HTML format
// @Tags compliance
// @Accept json
// @Produce json, text/csv, text/html
// @Param request body ExportReportRequest true "Export request"
// @Success 200 {object} byte "Report data in requested format"
// @Router /api/v1/compliance/export [post]
func (h *ComplianceHandler) ExportReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExportReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ReportID == "" {
		http.Error(w, "report_id is required", http.StatusBadRequest)
	}

	if req.Format == "" {
		req.Format = "json" // Default to JSON
	}

	// Validate format
	validFormats := map[string]bool{"json": true, "csv": true, "html": true}
	if !validFormats[req.Format] {
		http.Error(w, "Invalid format. Use json, csv, or html", http.StatusBadRequest)
		return
	}

	// For now, generate a new report (in production, load from storage)
	now := time.Now()
	startDate := now.AddDate(0, -1, 0)
	report, err := h.reportGenerator.GenerateReport(nil, startDate, now)
	if err != nil {
		http.Error(w, "Failed to generate report: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Export in requested format
	data, err := h.reportGenerator.ExportReport(report, req.Format)
	if err != nil {
		http.Error(w, "Failed to export report: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set content type based on format
	switch req.Format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=compliance-report.csv")
	case "html":
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Disposition", "attachment; filename=compliance-report.html")
	default:
		w.Header().Set("Content-Type", "application/json")
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// GetScoreRequest defines the request for getting compliance score
type GetScoreRequest struct {
	OrganizationID *int `json:"organization_id,omitempty"`
}

// GetScore retrieves the current compliance score
// @Summary Get current compliance score
// @Description Get the current compliance score for an organization
// @Tags compliance
// @Accept json
// @Produce json
// @Param request body GetScoreRequest true "Score request"
// @Success 200 {object} map[string]float64
// @Router /api/v1/compliance/score [post]
func (h *ComplianceHandler) GetScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GetScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	score, err := h.reportGenerator.GetComplianceScore(req.OrganizationID)
	if err != nil {
		http.Error(w, "Failed to get compliance score: "+err.Error(), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"compliance_score": score,
		"timestamp":        time.Now().Format(time.RFC3339),
	})
}

// ListReportsResponse defines the response for listing reports
type ListReportsResponse struct {
	Reports []ReportSummary `json:"reports"`
	Total   int             `json:"total"`
}

// ReportSummary is a summary of a compliance report
type ReportSummary struct {
	ID          string  `json:"id"`
	GeneratedAt string  `json:"generated_at"`
	StartDate   string  `json:"start_date"`
	EndDate     string  `json:"end_date"`
	Score       float64 `json:"score"`
	Violations  int     `json:"violations"`
}

// ListReports lists all generated compliance reports
// @Summary List compliance reports
// @Description List all previously generated compliance reports
// @Tags compliance
// @Produce json
// @Param limit query int false "Limit number of results"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} ListReportsResponse
// @Router /api/v1/compliance/reports [get]
func (h *ComplianceHandler) ListReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters (reserved for future pagination)
	_ = r.URL.Query().Get("limit")
	_ = r.URL.Query().Get("offset")

	// For now, return empty list
	// In production, this would query the database for saved reports
	response := ListReportsResponse{
		Reports: []ReportSummary{},
		Total:   0,
	}

	WriteJSON(w, http.StatusOK, response)
}

// CheckComplianceStatus checks the current compliance status
// @Summary Check compliance status
// @Description Check the current compliance status and recent violations
// @Tags compliance
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/compliance/status [get]
func (h *ComplianceHandler) CheckComplianceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	now := time.Now()
	startDate := now.AddDate(0, -1, 0)

	// Generate current report
	report, err := h.reportGenerator.GenerateReport(nil, startDate, now)
	if err != nil {
		http.Error(w, "Failed to generate compliance status: "+err.Error(), http.StatusInternalServerError)
		return
	}

	status := map[string]interface{}{
		"compliance_score":    report.Summary.ComplianceScore,
		"status":              getStatus(report.Summary.ComplianceScore),
		"total_violations":    report.Summary.TotalViolations,
		"critical_violations": report.Summary.CriticalViolations,
		"high_violations":     report.Summary.HighViolations,
		"total_audit_entries": report.Summary.TotalAuditEntries,
		"last_updated":        report.GeneratedAt.Format(time.RFC3339),
	}

	WriteJSON(w, http.StatusOK, status)
}

// getStatus returns a status string based on compliance score
func getStatus(score float64) string {
	if score >= 80 {
		return "compliant"
	} else if score >= 60 {
		return "warning"
	}
	return "non_compliant"
}
