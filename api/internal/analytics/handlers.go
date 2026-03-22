package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"wws/api/internal/handlers"
)

// UsageMetricsRequest represents a request to record usage metrics
type UsageMetricsRequest struct {
	WorkspaceID   int     `json:"workspace_id"`
	CPUUsage      float64 `json:"cpu_usage"`
	MemoryUsage   float64 `json:"memory_usage"`
	StorageUsedGB float64 `json:"storage_used_gb"`
	NetworkInMB   float64 `json:"network_in_mb"`
	NetworkOutMB  float64 `json:"network_out_mb"`
	UptimeSeconds int64   `json:"uptime_seconds"`
}

// UsageMetricsResponse represents the response for usage metrics
type UsageMetricsResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    *Usage `json:"data,omitempty"`
}

// Usage represents usage data in the response
type Usage struct {
	WorkspaceID   int       `json:"workspace_id"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsage   float64   `json:"memory_usage"`
	StorageUsedGB float64   `json:"storage_used_gb"`
	NetworkInMB   float64   `json:"network_in_mb"`
	NetworkOutMB  float64   `json:"network_out_mb"`
	Timestamp     time.Time `json:"timestamp"`
}

// RecordUsageHandler handles recording workspace usage metrics
func RecordUsageHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var req UsageMetricsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.WorkspaceID == 0 {
		http.Error(w, "workspace_id is required", http.StatusBadRequest)
		return
	}

	if err := RecordWorkspaceUsage(ctx, req.WorkspaceID, req.CPUUsage, req.MemoryUsage,
		req.StorageUsedGB, req.NetworkInMB, req.NetworkOutMB, req.UptimeSeconds); err != nil {
		http.Error(w, fmt.Sprintf("Failed to record usage: %v", err), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, UsageMetricsResponse{
		Success: true,
		Message: "Usage metrics recorded successfully",
		Data: &Usage{
			WorkspaceID:   req.WorkspaceID,
			CPUUsage:      req.CPUUsage,
			MemoryUsage:   req.MemoryUsage,
			StorageUsedGB: req.StorageUsedGB,
			NetworkInMB:   req.NetworkInMB,
			NetworkOutMB:  req.NetworkOutMB,
			Timestamp:     time.Now(),
		},
	})

	log.Printf("Recorded usage metrics for workspace %d", req.WorkspaceID)
}

// GetWorkspaceUsageHandler handles retrieving workspace usage history
func GetWorkspaceUsageHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	workspaceIDStr := r.URL.Query().Get("workspace_id")
	if workspaceIDStr == "" {
		http.Error(w, "workspace_id query parameter is required", http.StatusBadRequest)
		return
	}

	workspaceID, err := strconv.Atoi(workspaceIDStr)
	if err != nil {
		http.Error(w, "Invalid workspace_id", http.StatusBadRequest)
		return
	}

	// Default time range: last 24 hours
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	if startTimeStr := r.URL.Query().Get("start_time"); startTimeStr != "" {
		startTime, _ = time.Parse(time.RFC3339, startTimeStr)
	}
	if endTimeStr := r.URL.Query().Get("end_time"); endTimeStr != "" {
		endTime, _ = time.Parse(time.RFC3339, endTimeStr)
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	}

	usages, err := GetWorkspaceUsage(ctx, workspaceID, startTime, endTime, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve usage: %v", err), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"workspace_id":  workspaceID,
		"usage_history": usages,
		"total_records": len(usages),
		"time_range": map[string]string{
			"start": startTime.Format(time.RFC3339),
			"end":   endTime.Format(time.RFC3339),
		},
	})
}

// GetOrganizationUsageHandler handles retrieving organization usage
func GetOrganizationUsageHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	orgIDStr := r.URL.Query().Get("organization_id")
	if orgIDStr == "" {
		http.Error(w, "organization_id query parameter is required", http.StatusBadRequest)
		return
	}

	orgID, err := strconv.Atoi(orgIDStr)
	if err != nil {
		http.Error(w, "Invalid organization_id", http.StatusBadRequest)
		return
	}

	usage, err := GetOrganizationUsage(ctx, orgID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve organization usage: %v", err), http.StatusInternalServerError)
		return
	}

	if usage == nil {
		WriteJSON(w, http.StatusOK, map[string]interface{}{
			"organization_id": orgID,
			"message":         "No usage data available",
		})
		return
	}

	WriteJSON(w, http.StatusOK, usage)
}

// GetWorkspaceStatsHandler handles retrieving workspace statistics
func GetWorkspaceStatsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	workspaceIDStr := r.URL.Query().Get("workspace_id")
	if workspaceIDStr == "" {
		http.Error(w, "workspace_id query parameter is required", http.StatusBadRequest)
		return
	}

	workspaceID, err := strconv.Atoi(workspaceIDStr)
	if err != nil {
		http.Error(w, "Invalid workspace_id", http.StatusBadRequest)
		return
	}

	stats, err := GetWorkspaceStats(ctx, workspaceID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve workspace stats: %v", err), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, stats)
}

// GetAnalyticsSummaryHandler handles retrieving analytics summary
func GetAnalyticsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var orgID *int
	if orgIDStr := r.URL.Query().Get("organization_id"); orgIDStr != "" {
		id, err := strconv.Atoi(orgIDStr)
		if err == nil {
			orgID = &id
		}
	}

	summary, err := GetAnalyticsSummary(ctx, orgID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve analytics summary: %v", err), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, summary)
}

// GetActiveAlertsHandler handles retrieving active usage alerts
func GetActiveAlertsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	orgIDStr := r.URL.Query().Get("organization_id")
	if orgIDStr == "" {
		http.Error(w, "organization_id query parameter is required", http.StatusBadRequest)
		return
	}

	orgID, err := strconv.Atoi(orgIDStr)
	if err != nil {
		http.Error(w, "Invalid organization_id", http.StatusBadRequest)
		return
	}

	alerts, err := GetActiveAlerts(ctx, orgID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve alerts: %v", err), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"organization_id": orgID,
		"alerts":          alerts,
		"total":           len(alerts),
	})
}

// ResolveAlertHandler handles resolving a usage alert
func ResolveAlertHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	alertIDStr := r.URL.Query().Get("alert_id")
	if alertIDStr == "" {
		http.Error(w, "alert_id query parameter is required", http.StatusBadRequest)
		return
	}

	alertID, err := strconv.Atoi(alertIDStr)
	if err != nil {
		http.Error(w, "Invalid alert_id", http.StatusBadRequest)
		return
	}

	if err := ResolveAlert(ctx, alertID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to resolve alert: %v", err), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Alert resolved successfully",
	})
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Adapter wraps analytics handlers to match the Handler signature
func Adapter(h func(http.ResponseWriter, *http.Request)) handlers.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		h(w, r)
		return nil
	}
}
