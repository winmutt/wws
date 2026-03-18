package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// AuditLogHandler handles audit log requests
type AuditLogHandler struct {
	DB *sql.DB
}

// GetAuditLogs returns audit logs with optional filtering
func (h *AuditLogHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	userID := r.URL.Query().Get("user_id")
	orgID := r.URL.Query().Get("organization_id")
	action := r.URL.Query().Get("action")
	resourceType := r.URL.Query().Get("resource_type")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	success := r.URL.Query().Get("success")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Set defaults
	if limitStr == "" {
		limitStr = "100"
	}
	if offsetStr == "" {
		offsetStr = "0"
	}

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	if limit > 1000 {
		limit = 1000
	}

	// Build query
	query := `
		SELECT id, user_id, username, organization_id, action, resource_type,
		       resource_id, ip_address, user_agent, details, success, 
		       error_message, created_at
		FROM audit_logs
		WHERE 1=1`

	var args []interface{}

	if userID != "" {
		query += " AND user_id = ?"
		args = append(args, userID)
	}

	if orgID != "" {
		query += " AND organization_id = ?"
		args = append(args, orgID)
	}

	if action != "" {
		query += " AND action LIKE ?"
		args = append(args, "%"+action+"%")
	}

	if resourceType != "" {
		query += " AND resource_type = ?"
		args = append(args, resourceType)
	}

	if startDate != "" {
		if t, err := time.Parse(time.RFC3339, startDate); err == nil {
			query += " AND created_at >= ?"
			args = append(args, t)
		}
	}

	if endDate != "" {
		if t, err := time.Parse(time.RFC3339, endDate); err == nil {
			query += " AND created_at <= ?"
			args = append(args, t)
		}
	}

	if success != "" {
		if success == "true" {
			query += " AND success = 1"
		} else if success == "false" {
			query += " AND success = 0"
		}
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	// Execute query
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to query audit logs: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Parse results
	var logs []map[string]interface{}
	for rows.Next() {
		var id, userID, resourceID sql.NullInt64
		var orgID sql.NullInt64
		var username, action, resourceType, ipAddress, userAgent, details sql.NullString
		var success sql.NullInt64
		var errorMessage sql.NullString
		var createdAt time.Time

		if err := rows.Scan(&id, &userID, &username, &orgID, &action, &resourceType,
			&resourceID, &ipAddress, &userAgent, &details, &success,
			&errorMessage, &createdAt); err != nil {
			http.Error(w, "Failed to scan audit log: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log := map[string]interface{}{
			"id":              id.Int64,
			"user_id":         userID.Int64,
			"username":        username.String,
			"organization_id": nullableInt64(orgID),
			"action":          action.String,
			"resource_type":   resourceType.String,
			"resource_id":     nullableInt64(resourceID),
			"ip_address":      ipAddress.String,
			"user_agent":      userAgent.String,
			"details":         details.String,
			"success":         success.Int64 == 1,
			"error_message":   nullableString(errorMessage),
			"created_at":      createdAt.Format(time.RFC3339),
		}
		logs = append(logs, log)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE 1=1`
	var countArgs []interface{}

	if userID != "" {
		countQuery += " AND user_id = ?"
		countArgs = append(countArgs, userID)
	}
	if orgID != "" {
		countQuery += " AND organization_id = ?"
		countArgs = append(countArgs, orgID)
	}
	if action != "" {
		countQuery += " AND action LIKE ?"
		countArgs = append(countArgs, "%"+action+"%")
	}
	if resourceType != "" {
		countQuery += " AND resource_type = ?"
		countArgs = append(countArgs, resourceType)
	}
	if startDate != "" {
		if t, err := time.Parse(time.RFC3339, startDate); err == nil {
			countQuery += " AND created_at >= ?"
			countArgs = append(countArgs, t)
		}
	}
	if endDate != "" {
		if t, err := time.Parse(time.RFC3339, endDate); err == nil {
			countQuery += " AND created_at <= ?"
			countArgs = append(countArgs, t)
		}
	}
	if success != "" {
		if success == "true" {
			countQuery += " AND success = 1"
		} else if success == "false" {
			countQuery += " AND success = 0"
		}
	}

	var total int
	h.DB.QueryRow(countQuery, countArgs...).Scan(&total)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":   logs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetAuditLogByID returns a specific audit log
func (h *AuditLogHandler) GetAuditLogByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid audit log ID", http.StatusBadRequest)
		return
	}

	query := `
		SELECT id, user_id, username, organization_id, action, resource_type,
		       resource_id, ip_address, user_agent, details, success, 
		       error_message, created_at
		FROM audit_logs
		WHERE id = ?`

	var id_, userID, resourceID sql.NullInt64
	var orgID sql.NullInt64
	var username, action, resourceType, ipAddress, userAgent, details sql.NullString
	var success sql.NullInt64
	var errorMessage sql.NullString
	var createdAt time.Time

	if err := h.DB.QueryRow(query, id).Scan(&id_, &userID, &username, &orgID, &action, &resourceType,
		&resourceID, &ipAddress, &userAgent, &details, &success,
		&errorMessage, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Audit log not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to query audit log: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	log := map[string]interface{}{
		"id":              id_.Int64,
		"user_id":         userID.Int64,
		"username":        username.String,
		"organization_id": nullableInt64(orgID),
		"action":          action.String,
		"resource_type":   resourceType.String,
		"resource_id":     nullableInt64(resourceID),
		"ip_address":      ipAddress.String,
		"user_agent":      userAgent.String,
		"details":         details.String,
		"success":         success.Int64 == 1,
		"error_message":   nullableString(errorMessage),
		"created_at":      createdAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(log)
}

// GetAuditLogSummary returns summary statistics for audit logs
func (h *AuditLogHandler) GetAuditLogSummary(w http.ResponseWriter, r *http.Request) {
	// Get summary by action
	actionQuery := `SELECT action, COUNT(*) as count FROM audit_logs GROUP BY action ORDER BY count DESC LIMIT 10`
	rows, _ := h.DB.Query(actionQuery)
	defer rows.Close()

	actionStats := make(map[string]int)
	for rows.Next() {
		var action string
		var count int
		rows.Scan(&action, &count)
		actionStats[action] = count
	}

	// Get summary by resource type
	resourceQuery := `SELECT resource_type, COUNT(*) as count FROM audit_logs GROUP BY resource_type`
	rows, _ = h.DB.Query(resourceQuery)
	defer rows.Close()

	resourceStats := make(map[string]int)
	for rows.Next() {
		var resourceType string
		var count int
		rows.Scan(&resourceType, &count)
		resourceStats[resourceType] = count
	}

	// Get success/failure ratio
	successQuery := `SELECT success, COUNT(*) as count FROM audit_logs GROUP BY success`
	rows, _ = h.DB.Query(successQuery)
	defer rows.Close()

	successCount := 0
	failCount := 0
	for rows.Next() {
		var success int
		var count int
		rows.Scan(&success, &count)
		if success == 1 {
			successCount = count
		} else {
			failCount = count
		}
	}

	// Get logs by hour for last 24 hours
	hourlyQuery := `
		SELECT strftime('%Y-%m-%d %H:00', created_at) as hour, COUNT(*) as count 
		FROM audit_logs 
		WHERE created_at >= datetime('now', '-24 hours')
		GROUP BY hour 
		ORDER BY hour`
	rows, _ = h.DB.Query(hourlyQuery)
	defer rows.Close()

	hourlyStats := make([]map[string]interface{}, 0)
	for rows.Next() {
		var hour string
		var count int
		rows.Scan(&hour, &count)
		hourlyStats = append(hourlyStats, map[string]interface{}{
			"hour":  hour,
			"count": count,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_logs":    successCount + failCount,
		"success_count": successCount,
		"failure_count": failCount,
		"success_rate":  float64(successCount) / float64(successCount+failCount) * 100,
		"by_action":     actionStats,
		"by_resource":   resourceStats,
		"hourly_24h":    hourlyStats,
	})
}

// nullableInt64 converts sql.NullInt64 to *int64
func nullableInt64(v sql.NullInt64) *int64 {
	if v.Valid {
		result := int64(v.Int64)
		return &result
	}
	return nil
}

// nullableString converts sql.NullString to *string
func nullableString(v sql.NullString) *string {
	if v.Valid {
		result := v.String
		return &result
	}
	return nil
}
