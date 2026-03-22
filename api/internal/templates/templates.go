package templates

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"wws/api/internal/db"
)

// CreateTemplate creates a new workspace template
func CreateTemplate(ctx context.Context, orgID *int, userID int, req TemplateCreateRequest) (*WorkspaceTemplate, error) {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert template
	query := `
		INSERT INTO workspace_templates 
		(name, description, organization_id, provider, bootstrap_script, is_public, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := tx.ExecContext(ctx, query,
		req.Name, req.Description, orgID, req.Provider, req.BootstrapScript,
		func() int {
			if req.IsPublic {
				return 1
			}
			return 0
		}(),
		userID, time.Now(), time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert template: %w", err)
	}

	templateID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get template ID: %w", err)
	}

	// Insert language configurations
	for _, lang := range req.LanguageConfig {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO template_languages (template_id, language, version) VALUES (?, ?, ?)",
			templateID, lang.Language, lang.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert language config: %w", err)
		}
	}

	// Insert environment variables
	for _, env := range req.EnvVariables {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO template_env_vars (template_id, key, value) VALUES (?, ?, ?)",
			templateID, env.Key, env.Value,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert env var: %w", err)
		}
	}

	// Insert resource configuration
	_, err = tx.ExecContext(ctx,
		"INSERT INTO template_resources (template_id, cpu, memory_gb, storage_gb) VALUES (?, ?, ?, ?)",
		templateID, req.CPU, req.MemoryGB, req.StorageGB,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert resources: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Fetch created template
	return GetTemplateByID(ctx, int(templateID))
}

// GetTemplateByID retrieves a template by ID
func GetTemplateByID(ctx context.Context, id int) (*WorkspaceTemplate, error) {
	query := `
		SELECT id, name, description, organization_id, provider, bootstrap_script, is_public, created_by, created_at, updated_at
		FROM workspace_templates WHERE id = ?
	`

	var template WorkspaceTemplate
	var orgID sql.NullInt64

	err := db.DB.QueryRowContext(ctx, query, id).Scan(
		&template.ID, &template.Name, &template.Description, &orgID,
		&template.Provider, &template.BootstrapScript, &template.IsPublic,
		&template.CreatedBy, &template.CreatedAt, &template.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query template: %w", err)
	}

	if orgID.Valid {
		orgIDInt := int(orgID.Int64)
		template.OrganizationID = &orgIDInt
	}

	// Load language configurations
	template.LanguageConfig, err = getLanguageConfigs(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load language configs: %w", err)
	}

	// Load environment variables
	template.EnvVariables, err = getEnvVariables(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load env variables: %w", err)
	}

	// Load resource configuration
	template.Resources, err = getResourceConfig(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load resources: %w", err)
	}

	return &template, nil
}

// ListTemplates retrieves all templates with optional filtering
func ListTemplates(ctx context.Context, orgID *int, isPublic *bool) ([]WorkspaceTemplate, error) {
	query := "SELECT id, name, description, organization_id, provider, bootstrap_script, is_public, created_by, created_at, updated_at FROM workspace_templates WHERE 1=1"
	args := []interface{}{}

	if orgID != nil {
		query += " AND (organization_id = ? OR is_public = 1)"
		args = append(args, *orgID)
	}

	if isPublic != nil {
		if *isPublic {
			query += " AND is_public = 1"
		} else {
			query += " AND is_public = 0"
		}
	}

	rows, err := db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query templates: %w", err)
	}
	defer rows.Close()

	var templates []WorkspaceTemplate
	for rows.Next() {
		var template WorkspaceTemplate
		var orgID sql.NullInt64

		err := rows.Scan(
			&template.ID, &template.Name, &template.Description, &orgID,
			&template.Provider, &template.BootstrapScript, &template.IsPublic,
			&template.CreatedBy, &template.CreatedAt, &template.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}

		if orgID.Valid {
			orgIDInt := int(orgID.Int64)
			template.OrganizationID = &orgIDInt
		}

		template.LanguageConfig, _ = getLanguageConfigs(ctx, template.ID)
		template.EnvVariables, _ = getEnvVariables(ctx, template.ID)
		template.Resources, _ = getResourceConfig(ctx, template.ID)

		templates = append(templates, template)
	}

	return templates, nil
}

// UpdateTemplate updates an existing template
func UpdateTemplate(ctx context.Context, id int, req TemplateUpdateRequest) (*WorkspaceTemplate, error) {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Build update query dynamically
	updateParts := []string{"updated_at = ?"}
	args := []interface{}{time.Now()}

	if req.Name != nil {
		updateParts = append(updateParts, "name = ?")
		args = append(args, *req.Name)
	}
	if req.Description != nil {
		updateParts = append(updateParts, "description = ?")
		args = append(args, *req.Description)
	}
	if req.BootstrapScript != nil {
		updateParts = append(updateParts, "bootstrap_script = ?")
		args = append(args, *req.BootstrapScript)
	}
	if req.IsPublic != nil {
		updateParts = append(updateParts, "is_public = ?")
		args = append(args, func() int {
			if *req.IsPublic {
				return 1
			}
			return 0
		}())
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE workspace_templates SET %s WHERE id = ?",
		joinStrings(updateParts, ", "))

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	// Update language configurations
	if req.LanguageConfig != nil {
		_, err := tx.ExecContext(ctx, "DELETE FROM template_languages WHERE template_id = ?", id)
		if err != nil {
			return nil, fmt.Errorf("failed to delete language configs: %w", err)
		}
		for _, lang := range req.LanguageConfig {
			_, err := tx.ExecContext(ctx,
				"INSERT INTO template_languages (template_id, language, version) VALUES (?, ?, ?)",
				id, lang.Language, lang.Version,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to insert language config: %w", err)
			}
		}
	}

	// Update environment variables
	if req.EnvVariables != nil {
		_, err := tx.ExecContext(ctx, "DELETE FROM template_env_vars WHERE template_id = ?", id)
		if err != nil {
			return nil, fmt.Errorf("failed to delete env vars: %w", err)
		}
		for _, env := range req.EnvVariables {
			_, err := tx.ExecContext(ctx,
				"INSERT INTO template_env_vars (template_id, key, value) VALUES (?, ?, ?)",
				id, env.Key, env.Value,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to insert env var: %w", err)
			}
		}
	}

	// Update resource configuration
	if req.CPU != nil || req.MemoryGB != nil || req.StorageGB != nil {
		var cpu, memoryGB, storageGB float64

		err := db.DB.QueryRowContext(ctx,
			"SELECT cpu, memory_gb, storage_gb FROM template_resources WHERE template_id = ?", id).Scan(
			&cpu, &memoryGB, &storageGB)

		if req.CPU != nil {
			cpu = float64(*req.CPU)
		}
		if req.MemoryGB != nil {
			memoryGB = *req.MemoryGB
		}
		if req.StorageGB != nil {
			storageGB = float64(*req.StorageGB)
		}

		_, err = tx.ExecContext(ctx,
			"UPDATE template_resources SET cpu = ?, memory_gb = ?, storage_gb = ? WHERE template_id = ?",
			cpu, memoryGB, storageGB, id)
		if err != nil {
			return nil, fmt.Errorf("failed to update resources: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return GetTemplateByID(ctx, id)
}

// DeleteTemplate deletes a template
func DeleteTemplate(ctx context.Context, id int) error {
	_, err := db.DB.ExecContext(ctx, "DELETE FROM workspace_templates WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}
	return nil
}

// GetTemplateUsage retrieves usage statistics for a template
func GetTemplateUsage(ctx context.Context, templateID int) (*TemplateUsage, error) {
	query := `
		SELECT 
			t.id, t.name,
			COUNT(DISTINCT w.id) as usage_count,
			MAX(w.created_at) as last_used_at,
			COUNT(DISTINCT CASE WHEN w.status = 'running' THEN w.id END) as active_workspaces
		FROM workspace_templates t
		LEFT JOIN workspaces w ON w.template_id = t.id
		WHERE t.id = ?
		GROUP BY t.id
	`

	var usage TemplateUsage
	err := db.DB.QueryRowContext(ctx, query, templateID).Scan(
		&usage.TemplateID, &usage.TemplateName,
		&usage.UsageCount, &usage.LastUsedAt, &usage.ActiveWorkspaces,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query template usage: %w", err)
	}

	return &usage, nil
}

// getLanguageConfigs loads language configurations for a template
func getLanguageConfigs(ctx context.Context, templateID int) ([]LanguageConfig, error) {
	rows, err := db.DB.QueryContext(ctx,
		"SELECT language, version FROM template_languages WHERE template_id = ?", templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []LanguageConfig
	for rows.Next() {
		var config LanguageConfig
		err := rows.Scan(&config.Language, &config.Version)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	return configs, nil
}

// getEnvVariables loads environment variables for a template
func getEnvVariables(ctx context.Context, templateID int) ([]EnvVariable, error) {
	rows, err := db.DB.QueryContext(ctx,
		"SELECT key, value FROM template_env_vars WHERE template_id = ?", templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vars []EnvVariable
	for rows.Next() {
		var v EnvVariable
		err := rows.Scan(&v.Key, &v.Value)
		if err != nil {
			return nil, err
		}
		vars = append(vars, v)
	}

	return vars, nil
}

// getResourceConfig loads resource configuration for a template
func getResourceConfig(ctx context.Context, templateID int) (ResourceConfig, error) {
	var config ResourceConfig
	err := db.DB.QueryRowContext(ctx,
		"SELECT cpu, memory_gb, storage_gb FROM template_resources WHERE template_id = ?", templateID).Scan(
		&config.CPU, &config.MemoryGB, &config.StorageGB)

	if err == sql.ErrNoRows {
		return ResourceConfig{CPU: 2, MemoryGB: 4, StorageGB: 20}, nil // Default resources
	}
	if err != nil {
		return config, err
	}

	return config, nil
}

// joinStrings joins strings with a separator
func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, part := range parts[1:] {
		result += sep + part
	}
	return result
}

// ToJSON converts a template to JSON
func (t *WorkspaceTemplate) ToJSON() (string, error) {
	bytes, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// FromJSON parses a template from JSON
func FromJSON(jsonStr string) (*WorkspaceTemplate, error) {
	var template WorkspaceTemplate
	err := json.Unmarshal([]byte(jsonStr), &template)
	if err != nil {
		return nil, err
	}
	return &template, nil
}
