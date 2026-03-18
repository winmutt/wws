package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"wws/api/internal/models"
)

// AuditResponseWriter wraps http.ResponseWriter to capture status code
type AuditResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func (w *AuditResponseWriter) WriteHeader(code int) {
	w.StatusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// AuditMiddleware logs all requests for audit purposes
func AuditMiddleware(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Skip auditing for health checks and static assets
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/health") ||
				strings.HasPrefix(r.URL.Path, "/static") ||
				strings.HasPrefix(r.URL.Path, "/api/docs") {
				next.ServeHTTP(w, r)
				return
			}

			// Create audit response writer to capture status code
			aw := &AuditResponseWriter{ResponseWriter: w, StatusCode: http.StatusOK}

			// Get user info from context (set by auth middleware)
			userID := 0
			username := "anonymous"
			if user, ok := r.Context().Value("user").(*models.User); ok && user != nil {
				userID = user.ID
				username = user.Username
			}

			// Get organization from context if available
			orgID := getOrgIDFromContext(r)

			// Extract IP address
			ipAddress := r.Header.Get("X-Forwarded-For")
			if ipAddress == "" {
				ipAddress = r.RemoteAddr
			}
			// Clean up IP address (remove port if present)
			if idx := strings.Index(ipAddress, ":"); idx > 0 {
				ipAddress = ipAddress[:idx]
			}

			// Record start time
			start := time.Now()

			// Serve the request
			next.ServeHTTP(aw, r)

			// Calculate duration
			duration := time.Since(start)

			// Determine if request was successful
			success := aw.StatusCode >= 200 && aw.StatusCode < 400

			// Create audit log entry
			auditLog := &models.AuditLog{
				UserID:         userID,
				Username:       username,
				OrganizationID: orgID,
				Action:         r.Method + " " + r.URL.Path,
				ResourceType:   getResourceType(r.URL.Path),
				IPAddress:      ipAddress,
				UserAgent:      r.UserAgent(),
				Details:        formatAuditDetails(r, aw.StatusCode, duration),
				Success:        success,
				CreatedAt:      time.Now(),
			}

			// Save audit log asynchronously (don't block request)
			go func() {
				if err := CreateAuditLog(db, auditLog); err != nil {
					// Log error but don't fail the request
					// This is expected in some cases (e.g., during shutdown)
				}
			}()
		})
	}
}

// getOrgIDFromContext extracts organization ID from request context
func getOrgIDFromContext(r *http.Request) *int {
	if orgID, ok := r.Context().Value("organization_id").(int); ok {
		return &orgID
	}
	return nil
}

// getResourceType extracts resource type from URL path
func getResourceType(path string) string {
	path = strings.ToLower(path)
	if strings.Contains(path, "/workspaces/") {
		return "workspace"
	} else if strings.Contains(path, "/organizations/") {
		return "organization"
	} else if strings.Contains(path, "/users/") {
		return "user"
	} else if strings.Contains(path, "/auth/") {
		return "auth"
	} else if strings.Contains(path, "/languages/") {
		return "language"
	}
	return "unknown"
}

// formatAuditDetails creates a JSON string with request details
func formatAuditDetails(r *http.Request, statusCode int, duration time.Duration) string {
	details := map[string]interface{}{
		"method":       r.Method,
		"path":         r.URL.Path,
		"query":        r.URL.RawQuery,
		"status_code":  statusCode,
		"duration_ms":  duration.Milliseconds(),
		"content_type": r.Header.Get("Content-Type"),
	}

	jsonBytes, _ := json.Marshal(details)
	return string(jsonBytes)
}

// CreateAuditLog inserts an audit log entry into the database
func CreateAuditLog(db *sql.DB, auditLog *models.AuditLog) error {
	var orgID, resourceID sql.NullInt64
	var errorMessage sql.NullString

	if auditLog.OrganizationID != nil {
		orgID = sql.NullInt64{Int64: int64(*auditLog.OrganizationID), Valid: true}
	}
	if auditLog.ResourceID != nil {
		resourceID = sql.NullInt64{Int64: int64(*auditLog.ResourceID), Valid: true}
	}
	if auditLog.ErrorMessage != nil {
		errorMessage = sql.NullString{String: *auditLog.ErrorMessage, Valid: true}
	}

	query := `
		INSERT INTO audit_logs (user_id, username, organization_id, action, resource_type, 
			resource_id, ip_address, user_agent, details, success, error_message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(query,
		auditLog.UserID,
		auditLog.Username,
		orgID,
		auditLog.Action,
		auditLog.ResourceType,
		resourceID,
		auditLog.IPAddress,
		auditLog.UserAgent,
		auditLog.Details,
		auditLog.Success,
		errorMessage,
		auditLog.CreatedAt,
	)

	return err
}

// LogAudit creates a manual audit log entry (for use in handlers)
func LogAudit(db *sql.DB, r *http.Request, userID int, username string, orgID *int, action string, resourceType string, resourceID *int, success bool, errorMessage *string) {
	ipAddress := r.Header.Get("X-Forwarded-For")
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
	}
	if idx := strings.Index(ipAddress, ":"); idx > 0 {
		ipAddress = ipAddress[:idx]
	}

	auditLog := &models.AuditLog{
		UserID:         userID,
		Username:       username,
		OrganizationID: orgID,
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		IPAddress:      ipAddress,
		UserAgent:      r.UserAgent(),
		Details:        "",
		Success:        success,
		ErrorMessage:   errorMessage,
		CreatedAt:      time.Now(),
	}

	// Save audit log asynchronously
	go func() {
		if err := CreateAuditLog(db, auditLog); err != nil {
			// Log error to stderr
			// This is expected in some cases
		}
	}()
}

// ContextKey for storing audit context
type ContextKey string

const (
	// AuditContextKey is the key for audit information in context
	AuditContextKey ContextKey = "audit"
)

// AuditContext holds audit information for the current request
type AuditContext struct {
	UserID         int
	Username       string
	OrganizationID *int
}

// SetAuditContext sets audit information in the request context
func SetAuditContext(ctx context.Context, userID int, username string, orgID *int) context.Context {
	return context.WithValue(ctx, AuditContextKey, &AuditContext{
		UserID:         userID,
		Username:       username,
		OrganizationID: orgID,
	})
}

// GetAuditContext retrieves audit information from the request context
func GetAuditContext(r *http.Request) *AuditContext {
	if ctx, ok := r.Context().Value(AuditContextKey).(*AuditContext); ok {
		return ctx
	}
	return nil
}
