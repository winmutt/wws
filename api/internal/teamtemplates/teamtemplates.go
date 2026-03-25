package teamtemplates

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"wws/api/internal/db"
)

// TeamTemplateAccess represents team access to a workspace template
type TeamTemplateAccess struct {
	ID         int          `db:"id" json:"id"`
	TeamID     int          `db:"team_id" json:"team_id"`
	TemplateID int          `db:"template_id" json:"template_id"`
	Permission string       `db:"permission" json:"permission"` // "view" or "use"
	GrantedBy  int          `db:"granted_by" json:"granted_by"`
	GrantedAt  time.Time    `db:"granted_at" json:"granted_at"`
	Template   *TemplateInfo `db:"-" json:"template,omitempty"`
}

// TemplateInfo contains basic template information
type TemplateInfo struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	IsPublic    bool    `json:"is_public"`
}

// GrantTemplateAccess grants a team access to a workspace template
func GrantTemplateAccess(ctx context.Context, teamID, templateID, grantedBy int, permission string) (*TeamTemplateAccess, error) {
	// Verify template exists
	var templateExists bool
	err := db.DB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM workspace_templates WHERE id = ?)", templateID).Scan(&templateExists)
	if err != nil {
		return nil, fmt.Errorf("failed to check template existence: %w", err)
	}
	if !templateExists {
		return nil, fmt.Errorf("template not found")
	}

	// Verify team exists
	var teamExists bool
	err = db.DB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM teams WHERE id = ?)", teamID).Scan(&teamExists)
	if err != nil {
		return nil, fmt.Errorf("failed to check team existence: %w", err)
	}
	if !teamExists {
		return nil, fmt.Errorf("team not found")
	}

	// Check if access already exists
	var existingID int
	err = db.DB.QueryRowContext(ctx,
		"SELECT id FROM team_template_access WHERE team_id = ? AND template_id = ?",
		teamID, templateID).Scan(&existingID)

	if err == nil {
		// Update existing access
		_, err = db.DB.ExecContext(ctx,
			"UPDATE team_template_access SET permission = ?, granted_by = ?, granted_at = ? WHERE id = ?",
			permission, grantedBy, time.Now(), existingID)
		if err != nil {
			return nil, fmt.Errorf("failed to update template access: %w", err)
		}
		return GetTeamTemplateAccess(ctx, existingID)
	}

	// Create new access record
	query := `
		INSERT INTO team_template_access (team_id, template_id, permission, granted_by, granted_at)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := db.DB.ExecContext(ctx, query,
		teamID, templateID, permission, grantedBy, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to grant template access: %w", err)
	}

	accessID, _ := result.LastInsertId()
	return GetTeamTemplateAccess(ctx, int(accessID))
}

// GetTeamTemplateAccess retrieves team template access by ID
func GetTeamTemplateAccess(ctx context.Context, id int) (*TeamTemplateAccess, error) {
	var access TeamTemplateAccess
	err := db.DB.QueryRowContext(ctx, `
		SELECT id, team_id, template_id, permission, granted_by, granted_at
		FROM team_template_access WHERE id = ?
	`, id).Scan(&access.ID, &access.TeamID, &access.TemplateID,
		&access.Permission, &access.GrantedBy, &access.GrantedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("team template access not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query team template access: %w", err)
	}

	// Load template info
	access.Template, err = getTemplateInfo(ctx, access.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("failed to load template info: %w", err)
	}

	return &access, nil
}

// GetTeamTemplateAccessByTeam retrieves all template access for a team
func GetTeamTemplateAccessByTeam(ctx context.Context, teamID int) ([]TeamTemplateAccess, error) {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT id, team_id, template_id, permission, granted_by, granted_at
		FROM team_template_access WHERE team_id = ?
		ORDER BY granted_at DESC
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to query team template access: %w", err)
	}
	defer rows.Close()

	var accesses []TeamTemplateAccess
	for rows.Next() {
		var access TeamTemplateAccess
		if err := rows.Scan(&access.ID, &access.TeamID, &access.TemplateID,
			&access.Permission, &access.GrantedBy, &access.GrantedAt); err != nil {
			return nil, fmt.Errorf("failed to scan access: %w", err)
		}

		access.Template, _ = getTemplateInfo(ctx, access.TemplateID)
		accesses = append(accesses, access)
	}

	return accesses, nil
}

// GetTeamTemplateAccessByTemplate retrieves all team access for a template
func GetTeamTemplateAccessByTemplate(ctx context.Context, templateID int) ([]TeamTemplateAccess, error) {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT id, team_id, template_id, permission, granted_by, granted_at
		FROM team_template_access WHERE template_id = ?
		ORDER BY granted_at DESC
	`, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to query template team access: %w", err)
	}
	defer rows.Close()

	var accesses []TeamTemplateAccess
	for rows.Next() {
		var access TeamTemplateAccess
		if err := rows.Scan(&access.ID, &access.TeamID, &access.TemplateID,
			&access.Permission, &access.GrantedBy, &access.GrantedAt); err != nil {
			return nil, fmt.Errorf("failed to scan access: %w", err)
		}

		access.Template, _ = getTemplateInfo(ctx, access.TemplateID)
		accesses = append(accesses, access)
	}

	return accesses, nil
}

// RevokeTemplateAccess revokes a team's access to a template
func RevokeTemplateAccess(ctx context.Context, id int) error {
	_, err := db.DB.ExecContext(ctx, "DELETE FROM team_template_access WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to revoke template access: %w", err)
	}
	return nil
}

// RevokeTeamTemplateAccess revokes all template access for a specific team
func RevokeTeamTemplateAccess(ctx context.Context, teamID int) error {
	_, err := db.DB.ExecContext(ctx, "DELETE FROM team_template_access WHERE team_id = ?", teamID)
	if err != nil {
		return fmt.Errorf("failed to revoke team template access: %w", err)
	}
	return nil
}

// HasTeamTemplatePermission checks if a team has permission to use a template
func HasTeamTemplatePermission(ctx context.Context, teamID, templateID int, requiredPermission string) (bool, error) {
	var permission sql.NullString
	err := db.DB.QueryRowContext(ctx, `
		SELECT permission FROM team_template_access
		WHERE team_id = ? AND template_id = ?
	`, teamID, templateID).Scan(&permission)

	if err == sql.ErrNoRows {
		// Check if template is public
		var isPublic int
		err := db.DB.QueryRowContext(ctx, "SELECT is_public FROM workspace_templates WHERE id = ?", templateID).Scan(&isPublic)
		if err != nil {
			return false, fmt.Errorf("template not found: %w", err)
		}
		return isPublic == 1, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	// "use" permission grants all access, "view" only allows viewing
	if requiredPermission == "use" {
		return permission.String == "use", nil
	}
	return permission.String == "use" || permission.String == "view", nil
}

// GetTeamUsableTemplates retrieves all templates a team can use
func GetTeamUsableTemplates(ctx context.Context, teamID int) ([]TemplateInfo, error) {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT DISTINCT wt.id, wt.name, wt.description, wt.is_public
		FROM workspace_templates wt
		INNER JOIN team_template_access tta ON tta.template_id = wt.id
		WHERE tta.team_id = ? AND tta.permission = 'use'
		UNION
		SELECT id, name, description, is_public
		FROM workspace_templates
		WHERE is_public = 1
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to query team templates: %w", err)
	}
	defer rows.Close()

	var templates []TemplateInfo
	for rows.Next() {
		var template TemplateInfo
		if err := rows.Scan(&template.ID, &template.Name, &template.Description, &template.IsPublic); err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		templates = append(templates, template)
	}

	return templates, nil
}

// getTemplateInfo loads basic template information
func getTemplateInfo(ctx context.Context, templateID int) (*TemplateInfo, error) {
	var info TemplateInfo
	err := db.DB.QueryRowContext(ctx, `
		SELECT id, name, description, is_public
		FROM workspace_templates WHERE id = ?
	`, templateID).Scan(&info.ID, &info.Name, &info.Description, &info.IsPublic)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found")
	}
	if err != nil {
		return nil, err
	}

	return &info, nil
}
