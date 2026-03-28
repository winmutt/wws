package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"wws/api/internal/db"
	"wws/api/internal/models"
)

type exportContextKey string

const exportUserIDKey exportContextKey = "user_id"

// ExportRequest represents a workspace export request
type ExportRequest struct {
	Format      string `json:"format"`       // json, tar, zip
	IncludeData bool   `json:"include_data"` // Include workspace data files
}

// ExportResponse represents a workspace export response
type ExportResponse struct {
	ID           int       `json:"id"`
	WorkspaceID  int       `json:"workspace_id"`
	ExportPath   string    `json:"export_path"`
	Format       string    `json:"format"`
	FileSizeMB   float64   `json:"file_size_mb,omitempty"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// ImportRequest represents a workspace import request
type ImportRequest struct {
	ExportID       int    `json:"export_id"`   // Reference to an existing export
	ExportPath     string `json:"export_path"` // Path to export file
	Format         string `json:"format"`      // json, tar, zip
	Name           string `json:"name"`        // New workspace name
	OrganizationID int    `json:"organization_id"`
}

// ImportResponse represents a workspace import response
type ImportResponse struct {
	ID                  int       `json:"id"`
	ExportID            int       `json:"export_id,omitempty"`
	Status              string    `json:"status"`
	ImportedWorkspaceID int       `json:"imported_workspace_id,omitempty"`
	Error               string    `json:"error,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

// ExportWorkspace exports a workspace configuration and optionally data
var ExportWorkspaceHandler Handler = func(w http.ResponseWriter, r *http.Request) error {
	userIDVal, ok := r.Context().Value(exportUserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil
	}
	userID := userIDVal

	workspaceID := r.URL.Query().Get("workspace_id")
	if workspaceID == "" {
		http.Error(w, "workspace_id parameter is required", http.StatusBadRequest)
		return nil
	}

	// Parse request body
	var req ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to JSON format if no body provided
		req.Format = "json"
		req.IncludeData = false
	}

	if req.Format == "" {
		req.Format = "json"
	}

	// Get workspace details
	var ws models.Workspace
	err := db.DB.QueryRow(
		"SELECT * FROM workspaces WHERE id = ? AND deleted_at IS NULL",
		workspaceID,
	).Scan(&ws.ID, &ws.Tag, &ws.Name, &ws.OrganizationID, &ws.OwnerID, &ws.Provider, &ws.Status, &ws.Config, &ws.Region, &ws.CreatedAt, &ws.UpdatedAt, &ws.DeletedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return nil
	}
	if err != nil {
		log.Printf("Failed to query workspace: %v", err)
		http.Error(w, "Failed to retrieve workspace", http.StatusInternalServerError)
		return nil
	}

	// Check ownership or membership
	if !canAccessWorkspace(userID, ws.ID) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return nil
	}

	// Create export directory if it doesn't exist
	exportDir := filepath.Join(os.TempDir(), "wws_exports")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		log.Printf("Failed to create export directory: %v", err)
		http.Error(w, "Failed to create export directory", http.StatusInternalServerError)
		return nil
	}

	// Generate unique export filename
	exportFilename := fmt.Sprintf("workspace-%s-%s.%s", ws.Tag, time.Now().Format("20060102-150405"), req.Format)
	exportPath := filepath.Join(exportDir, exportFilename)

	// Create export record in database
	exportRecord := struct {
		WorkspaceID int
		ExportPath  string
		Format      string
		Status      string
		CreatedBy   int
		ExpiresAt   time.Time
	}{
		WorkspaceID: ws.ID,
		ExportPath:  exportPath,
		Format:      req.Format,
		Status:      "pending",
		CreatedBy:   userID,
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour), // Expire in 7 days
	}

	result, err := db.DB.Exec(
		`INSERT INTO workspace_exports (workspace_id, export_path, export_format, status, created_by, expires_at) 
		 VALUES (?, ?, ?, ?, ?, ?)`,
		exportRecord.WorkspaceID, exportRecord.ExportPath, exportRecord.Format, exportRecord.Status, exportRecord.CreatedBy, exportRecord.ExpiresAt,
	)
	if err != nil {
		log.Printf("Failed to create export record: %v", err)
		http.Error(w, "Failed to create export record", http.StatusInternalServerError)
		return nil
	}

	exportID, _ := result.LastInsertId()

	// Perform the export
	exportErr := performWorkspaceExport(ws, req, exportPath)

	// Update export record status
	status := "completed"
	if exportErr != nil {
		status = "failed"
		log.Printf("Export failed: %v", exportErr)
		db.DB.Exec(
			"UPDATE workspace_exports SET status = ?, error_message = ? WHERE id = ?",
			status, exportErr.Error(), exportID,
		)
		http.Error(w, fmt.Sprintf("Export failed: %v", exportErr), http.StatusInternalServerError)
		return nil
	}

	// Get file size
	fileInfo, err := os.Stat(exportPath)
	fileSizeMB := float64(0)
	if err == nil {
		fileSizeMB = float64(fileInfo.Size()) / (1024 * 1024)
	}

	// Update export record with final status
	db.DB.Exec(
		"UPDATE workspace_exports SET status = ?, file_size_mb = ? WHERE id = ?",
		status, fileSizeMB, exportID,
	)

	// Log the export action
	logExportAction(userID, ws.ID, "export", "", fmt.Sprintf("Export format: %s, Include data: %v", req.Format, req.IncludeData))

	response := ExportResponse{
		ID:          int(exportID),
		WorkspaceID: ws.ID,
		ExportPath:  exportPath,
		Format:      req.Format,
		FileSizeMB:  fileSizeMB,
		Status:      status,
		CreatedAt:   time.Now(),
		ExpiresAt:   exportRecord.ExpiresAt,
	}

	json.NewEncoder(w).Encode(response)
	return nil
}

// performWorkspaceExport performs the actual workspace export
func performWorkspaceExport(ws models.Workspace, req ExportRequest, exportPath string) error {
	// Create workspace export data
	exportData := map[string]interface{}{
		"version":     "1.0",
		"exported_at": time.Now().Format(time.RFC3339),
		"workspace": map[string]interface{}{
			"id":          ws.ID,
			"tag":         ws.Tag,
			"name":        ws.Name,
			"provider":    ws.Provider,
			"config":      ws.Config,
			"region":      ws.Region,
			"created_at":  ws.CreatedAt.Format(time.RFC3339),
			"languages":   getWorkspaceLanguages(ws.ID),
			"environment": getWorkspaceEnvVars(ws.ID),
		},
	}

	// Add workspace data if requested
	if req.IncludeData {
		// TODO: Implement actual data export (files, databases, etc.)
		exportData["data"] = map[string]interface{}{
			"note": "Data export not yet implemented",
		}
	}

	// Write export file
	file, err := os.Create(exportPath)
	if err != nil {
		return fmt.Errorf("failed to create export file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(exportData); err != nil {
		return fmt.Errorf("failed to write export data: %v", err)
	}

	return nil
}

// getWorkspaceLanguages retrieves language configurations for a workspace
func getWorkspaceLanguages(workspaceID int) []map[string]string {
	rows, err := db.DB.Query(
		"SELECT language, version, install_script FROM workspace_languages WHERE workspace_id = ?",
		workspaceID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var languages []map[string]string
	for rows.Next() {
		var lang, version, script sql.NullString
		if err := rows.Scan(&lang, &version, &script); err != nil {
			continue
		}
		languages = append(languages, map[string]string{
			"language":       lang.String,
			"version":        version.String,
			"install_script": script.String,
		})
	}
	return languages
}

// getWorkspaceEnvVars retrieves environment variables for a workspace
func getWorkspaceEnvVars(workspaceID int) map[string]string {
	// Environment variables would need a dedicated table
	// For now, return empty map
	return make(map[string]string)
}

// canAccessWorkspace checks if a user can access a workspace
func canAccessWorkspace(userID, workspaceID int) bool {
	// Check if user is the owner
	var count int
	err := db.DB.QueryRow(
		"SELECT COUNT(*) FROM workspaces WHERE id = ? AND owner_id = ?",
		workspaceID, userID,
	).Scan(&count)
	if err != nil || count > 0 {
		return count > 0
	}

	// Check if user is a workspace member
	err = db.DB.QueryRow(
		"SELECT COUNT(*) FROM workspace_members WHERE workspace_id = ? AND user_id = ? AND status = 'active'",
		workspaceID, userID,
	).Scan(&count)
	return count > 0
}

// logExportAction logs export/import actions
func logExportAction(userID, workspaceID int, action, resourceType, details string) {
	username := getUsernameByUserID(userID)
	db.DB.Exec(
		`INSERT INTO audit_logs (user_id, username, action, resource_type, resource_id, details, ip_address, user_agent, success)
		 VALUES (?, ?, ?, ?, ?, ?, '', '', 1)`,
		userID, username, action, "workspace", workspaceID, details,
	)
}

// getUsernameByUserID retrieves username by user ID
func getUsernameByUserID(userID int) string {
	var username string
	db.DB.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	return username
}

// ImportWorkspace imports a workspace from an export file
var ImportWorkspaceHandler Handler = func(w http.ResponseWriter, r *http.Request) error {
	userIDVal, ok := r.Context().Value(exportUserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil
	}
	userID := userIDVal

	// Parse request body
	var req ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return nil
	}

	// Validate required fields
	if req.OrganizationID == 0 {
		http.Error(w, "organization_id is required", http.StatusBadRequest)
		return nil
	}

	// Check organization membership
	if !isMemberOfOrg(userID, req.OrganizationID) {
		http.Error(w, "Not a member of this organization", http.StatusForbidden)
		return nil
	}

	if req.Name == "" {
		req.Name = fmt.Sprintf("imported-%s", time.Now().Format("20060102-150405"))
	}

	// Create import record in database
	importRecord := struct {
		ExportID       int
		ExportPath     string
		Format         string
		OrganizationID int
		ImportedBy     int
		Status         string
	}{
		ExportID:       req.ExportID,
		ExportPath:     req.ExportPath,
		Format:         req.Format,
		OrganizationID: req.OrganizationID,
		ImportedBy:     userID,
		Status:         "pending",
	}

	result, err := db.DB.Exec(
		`INSERT INTO workspace_imports (export_id, export_path, import_format, organization_id, imported_by, status) 
		 VALUES (?, ?, ?, ?, ?, ?)`,
		importRecord.ExportID, importRecord.ExportPath, importRecord.Format, importRecord.OrganizationID, importRecord.ImportedBy, importRecord.Status,
	)
	if err != nil {
		log.Printf("Failed to create import record: %v", err)
		http.Error(w, "Failed to create import record", http.StatusInternalServerError)
		return nil
	}

	importID, _ := result.LastInsertId()

	// Perform the import
	importedWorkspaceID, importErr := performWorkspaceImport(req, userID, req.OrganizationID)

	// Update import record status
	status := "completed"
	if importErr != nil {
		status = "failed"
		log.Printf("Import failed: %v", importErr)
		db.DB.Exec(
			"UPDATE workspace_imports SET status = ?, error_message = ? WHERE id = ?",
			status, importErr.Error(), importID,
		)

		response := ImportResponse{
			ID:        int(importID),
			ExportID:  req.ExportID,
			Status:    status,
			Error:     importErr.Error(),
			CreatedAt: time.Now(),
		}
		json.NewEncoder(w).Encode(response)
		return nil
	}

	// Update import record with imported workspace ID
	db.DB.Exec(
		"UPDATE workspace_imports SET status = ?, imported_workspace_id = ? WHERE id = ?",
		status, importedWorkspaceID, importID,
	)

	// Log the import action
	logExportAction(userID, importedWorkspaceID, "import", "workspace", fmt.Sprintf("Imported from format: %s", req.Format))

	response := ImportResponse{
		ID:                  int(importID),
		ExportID:            req.ExportID,
		Status:              status,
		ImportedWorkspaceID: importedWorkspaceID,
		CreatedAt:           time.Now(),
	}

	json.NewEncoder(w).Encode(response)
	return nil
}

// performWorkspaceImport performs the actual workspace import
func performWorkspaceImport(req ImportRequest, userID, orgID int) (int, error) {
	// Read export file
	var exportData map[string]interface{}

	if req.ExportID > 0 {
		// Get export path from database
		var exportPath string
		err := db.DB.QueryRow(
			"SELECT export_path FROM workspace_exports WHERE id = ?",
			req.ExportID,
		).Scan(&exportPath)

		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("export not found")
		}
		if err != nil {
			return 0, fmt.Errorf("failed to get export: %v", err)
		}

		req.ExportPath = exportPath
	}

	file, err := os.Open(req.ExportPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open export file: %v", err)
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return 0, fmt.Errorf("failed to read export file: %v", err)
	}

	if err := json.Unmarshal(data, &exportData); err != nil {
		return 0, fmt.Errorf("failed to parse export data: %v", err)
	}

	// Extract workspace data
	workspaceData, ok := exportData["workspace"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("invalid export format: missing workspace data")
	}

	// Create new workspace
	tag := generateWorkspaceTag(req.Name)
	configJSON, _ := json.Marshal(workspaceData["config"])

	// Insert workspace into database
	result, err := db.DB.Exec(
		`INSERT INTO workspaces (tag, name, organization_id, owner_id, provider, status, config, region, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tag, req.Name, orgID, userID, "podman", "pending", string(configJSON), workspaceData["region"], time.Now(), time.Now(),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create workspace: %v", err)
	}

	workspaceID, _ := result.LastInsertId()

	// Import languages if present
	if languages, ok := workspaceData["languages"].([]interface{}); ok {
		for _, lang := range languages {
			if langMap, ok := lang.(map[string]interface{}); ok {
				db.DB.Exec(
					`INSERT INTO workspace_languages (workspace_id, language, version, install_script)
					 VALUES (?, ?, ?, ?)`,
					workspaceID, langMap["language"], langMap["version"], langMap["install_script"],
				)
			}
		}
	}

	// TODO: Trigger workspace provisioning
	// This would call the provider to actually create the workspace

	return int(workspaceID), nil
}

// GetExportStatus gets the status of an export
var GetExportStatusHandler Handler = func(w http.ResponseWriter, r *http.Request) error {
	exportID := r.URL.Query().Get("id")
	if exportID == "" {
		http.Error(w, "id parameter is required", http.StatusBadRequest)
		return nil
	}

	var export ExportResponse
	var createdAt, expiresAt string
	var errorMessage sql.NullString

	err := db.DB.QueryRow(
		`SELECT id, workspace_id, export_path, export_format, file_size_mb, status, error_message, created_at, expires_at 
		 FROM workspace_exports WHERE id = ?`,
		exportID,
	).Scan(&export.ID, &export.WorkspaceID, &export.ExportPath, &export.Format, &export.FileSizeMB, &export.Status, &errorMessage, &createdAt, &expiresAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Export not found", http.StatusNotFound)
		return nil
	}
	if err != nil {
		log.Printf("Failed to query export: %v", err)
		http.Error(w, "Failed to retrieve export", http.StatusInternalServerError)
		return nil
	}

	export.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	export.ExpiresAt, _ = time.Parse("2006-01-02 15:04:05", expiresAt)
	if errorMessage.Valid {
		export.ErrorMessage = errorMessage.String
	}

	json.NewEncoder(w).Encode(export)
	return nil
}

// GetImportStatus gets the status of an import
var GetImportStatusHandler Handler = func(w http.ResponseWriter, r *http.Request) error {
	importID := r.URL.Query().Get("id")
	if importID == "" {
		http.Error(w, "id parameter is required", http.StatusBadRequest)
		return nil
	}

	var importResp ImportResponse
	var createdAt string
	var importedWorkspaceID sql.NullInt64

	err := db.DB.QueryRow(
		`SELECT id, export_id, status, imported_workspace_id, created_at 
		 FROM workspace_imports WHERE id = ?`,
		importID,
	).Scan(&importResp.ID, &importResp.ExportID, &importResp.Status, &importedWorkspaceID, &createdAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Import not found", http.StatusNotFound)
		return nil
	}
	if err != nil {
		log.Printf("Failed to query import: %v", err)
		http.Error(w, "Failed to retrieve import", http.StatusInternalServerError)
		return nil
	}

	importResp.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	if importedWorkspaceID.Valid {
		importResp.ImportedWorkspaceID = int(importedWorkspaceID.Int64)
	}

	json.NewEncoder(w).Encode(importResp)
	return nil
}

// ListExports lists all exports for a user
var ListExportsHandler Handler = func(w http.ResponseWriter, r *http.Request) error {
	userIDVal, ok := r.Context().Value(exportUserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil
	}
	userID := userIDVal

	rows, err := db.DB.Query(
		`SELECT id, workspace_id, export_path, export_format, file_size_mb, status, error_message, created_at, expires_at 
		 FROM workspace_exports WHERE created_by = ? ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		log.Printf("Failed to query exports: %v", err)
		http.Error(w, "Failed to retrieve exports", http.StatusInternalServerError)
		return nil
	}
	defer rows.Close()

	var exports []ExportResponse
	for rows.Next() {
		var exp ExportResponse
		var createdAt, expiresAt string
		var errorMessage sql.NullString
		if err := rows.Scan(&exp.ID, &exp.WorkspaceID, &exp.ExportPath, &exp.Format, &exp.FileSizeMB, &exp.Status, &errorMessage, &createdAt, &expiresAt); err != nil {
			continue
		}
		exp.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		exp.ExpiresAt, _ = time.Parse("2006-01-02 15:04:05", expiresAt)
		if errorMessage.Valid {
			exp.ErrorMessage = errorMessage.String
		}
		exports = append(exports, exp)
	}

	json.NewEncoder(w).Encode(exports)
	return nil
}

// ListImports lists all imports for an organization
var ListImportsHandler Handler = func(w http.ResponseWriter, r *http.Request) error {
	userIDVal, ok := r.Context().Value(exportUserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil
	}
	userID := userIDVal

	orgIDStr := r.URL.Query().Get("organization_id")
	if orgIDStr == "" {
		http.Error(w, "organization_id parameter is required", http.StatusBadRequest)
		return nil
	}

	var orgID int
	fmt.Sscanf(orgIDStr, "%d", &orgID)

	// Check organization membership
	if !isMemberOfOrg(userID, orgID) {
		http.Error(w, "Not a member of this organization", http.StatusForbidden)
		return nil
	}

	rows, err := db.DB.Query(
		`SELECT id, export_id, status, imported_workspace_id, created_at 
		 FROM workspace_imports WHERE organization_id = ? ORDER BY created_at DESC`,
		orgID,
	)
	if err != nil {
		log.Printf("Failed to query imports: %v", err)
		http.Error(w, "Failed to retrieve imports", http.StatusInternalServerError)
		return nil
	}
	defer rows.Close()

	var imports []ImportResponse
	for rows.Next() {
		var imp ImportResponse
		var createdAt string
		var importedWorkspaceID sql.NullInt64
		if err := rows.Scan(&imp.ID, &imp.ExportID, &imp.Status, &importedWorkspaceID, &createdAt); err != nil {
			continue
		}
		imp.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		if importedWorkspaceID.Valid {
			imp.ImportedWorkspaceID = int(importedWorkspaceID.Int64)
		}
		imports = append(imports, imp)
	}

	json.NewEncoder(w).Encode(imports)
	return nil
}

// DownloadExport downloads an export file
var DownloadExportHandler Handler = func(w http.ResponseWriter, r *http.Request) error {
	userIDVal, ok := r.Context().Value(exportUserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil
	}
	userID := userIDVal

	exportID := r.URL.Query().Get("id")
	if exportID == "" {
		http.Error(w, "id parameter is required", http.StatusBadRequest)
		return nil
	}

	// Get export details
	var exportPath string
	var workspaceID int
	var createdBy int
	var expiresAt string

	err := db.DB.QueryRow(
		`SELECT export_path, workspace_id, created_by, expires_at FROM workspace_exports WHERE id = ?`,
		exportID,
	).Scan(&exportPath, &workspaceID, &createdBy, &expiresAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Export not found", http.StatusNotFound)
		return nil
	}
	if err != nil {
		log.Printf("Failed to query export: %v", err)
		http.Error(w, "Failed to retrieve export", http.StatusInternalServerError)
		return nil
	}

	// Check if user has access
	if !canAccessWorkspace(userID, workspaceID) && createdBy != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return nil
	}

	// Check if export has expired
	expiry, _ := time.Parse("2006-01-02 15:04:05", expiresAt)
	if time.Now().After(expiry) {
		http.Error(w, "Export has expired", http.StatusGone)
		return nil
	}

	// Check if file exists
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		http.Error(w, "Export file not found", http.StatusNotFound)
		return nil
	}

	// Serve the file
	http.ServeFile(w, r, exportPath)
	return nil
}

// DeleteExport deletes an export record and file
var DeleteExportHandler Handler = func(w http.ResponseWriter, r *http.Request) error {
	userIDVal, ok := r.Context().Value(exportUserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil
	}
	userID := userIDVal

	exportID := r.URL.Query().Get("id")
	if exportID == "" {
		http.Error(w, "id parameter is required", http.StatusBadRequest)
		return nil
	}

	// Get export details
	var exportPath string
	var workspaceID int
	var createdBy int

	err := db.DB.QueryRow(
		`SELECT export_path, workspace_id, created_by FROM workspace_exports WHERE id = ?`,
		exportID,
	).Scan(&exportPath, &workspaceID, &createdBy)

	if err == sql.ErrNoRows {
		http.Error(w, "Export not found", http.StatusNotFound)
		return nil
	}
	if err != nil {
		log.Printf("Failed to query export: %v", err)
		http.Error(w, "Failed to retrieve export", http.StatusInternalServerError)
		return nil
	}

	// Check if user has access
	if !canAccessWorkspace(userID, workspaceID) && createdBy != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return nil
	}

	// Delete the file
	if err := os.Remove(exportPath); err != nil && !os.IsNotExist(err) {
		log.Printf("Failed to delete export file: %v", err)
	}

	// Delete the record
	_, err = db.DB.Exec("DELETE FROM workspace_exports WHERE id = ?", exportID)
	if err != nil {
		log.Printf("Failed to delete export record: %v", err)
		http.Error(w, "Failed to delete export", http.StatusInternalServerError)
		return nil
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// isMemberOfOrg checks if a user is a member of an organization
func isMemberOfOrg(userID, orgID int) bool {
	var count int
	err := db.DB.QueryRow(
		"SELECT COUNT(*) FROM members WHERE user_id = ? AND organization_id = ? AND accepted = 1",
		userID, orgID,
	).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

// generateWorkspaceTag generates a unique tag for a workspace
func generateWorkspaceTag(name string) string {
	// Convert name to lowercase and replace spaces with hyphens
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	// Remove special characters
	var cleaned strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			cleaned.WriteRune(r)
		}
	}

	// Add UUID to ensure uniqueness
	uuid := uuid.New().String()[:8]
	return fmt.Sprintf("%s-%s", cleaned.String(), uuid)
}
