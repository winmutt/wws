package permissions

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"wws/api/internal/db"
)

// CreateTeam creates a new team
func CreateTeam(ctx context.Context, orgID, userID int, name, description string) (*Team, error) {
	query := `
		INSERT INTO teams (organization_id, name, description, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := db.DB.ExecContext(ctx, query, orgID, name, description, userID, time.Now(), time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to create team: %w", err)
	}

	teamID, _ := result.LastInsertId()
	return &Team{
		ID:             int(teamID),
		OrganizationID: orgID,
		Name:           name,
		Description:    description,
		CreatedBy:      userID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

// ListTeams retrieves all teams for an organization
func ListTeams(ctx context.Context, orgID int) ([]Team, error) {
	rows, err := db.DB.QueryContext(ctx,
		"SELECT id, organization_id, name, description, created_by, created_at, updated_at FROM teams WHERE organization_id = ?",
		orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query teams: %w", err)
	}
	defer rows.Close()

	var teams []Team
	for rows.Next() {
		var team Team
		if err := rows.Scan(&team.ID, &team.OrganizationID, &team.Name, &team.Description,
			&team.CreatedBy, &team.CreatedAt, &team.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan team: %w", err)
		}
		teams = append(teams, team)
	}

	return teams, nil
}

// GetTeamMembers retrieves all members of a team
func GetTeamMembers(ctx context.Context, teamID int) ([]TeamMember, error) {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT tm.id, tm.team_id, tm.user_id, u.username, u.email, r.name as role_name, tm.joined_at, tm.status
		FROM team_members tm
		JOIN users u ON tm.user_id = u.id
		LEFT JOIN team_roles r ON tm.role_id = r.id
		WHERE tm.team_id = ?
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to query team members: %w", err)
	}
	defer rows.Close()

	var members []TeamMember
	for rows.Next() {
		var member TeamMember
		if err := rows.Scan(&member.ID, &member.TeamID, &member.UserID, &member.Username,
			&member.Email, &member.RoleName, &member.JoinedAt, &member.Status); err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		members = append(members, member)
	}

	return members, nil
}

// AddTeamMember adds a user to a team
func AddTeamMember(ctx context.Context, teamID, userID, roleID, assignedBy int) error {
	query := `
		INSERT INTO team_members (team_id, user_id, role_id, joined_at, status, added_by, added_at)
		VALUES (?, ?, ?, ?, 'active', ?, ?)
	`
	_, err := db.DB.ExecContext(ctx, query, teamID, userID, roleID, time.Now(), assignedBy, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add team member: %w", err)
	}

	return nil
}

// RemoveTeamMember removes a user from a team
func RemoveTeamMember(ctx context.Context, teamID, userID int) error {
	_, err := db.DB.ExecContext(ctx,
		"DELETE FROM team_members WHERE team_id = ? AND user_id = ?", teamID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove team member: %w", err)
	}
	return nil
}

// CreateTeamRole creates a new team role
func CreateTeamRole(ctx context.Context, name, description string, permissions []TeamPermission, isDefault bool) (*TeamRole, error) {
	permsJSON, _ := json.Marshal(permissions)

	query := `
		INSERT INTO team_roles (name, description, permissions, is_default, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := db.DB.ExecContext(ctx, query, name, description, string(permsJSON), isDefault, time.Now(), time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to create team role: %w", err)
	}

	roleID, _ := result.LastInsertId()
	return &TeamRole{
		ID:          int(roleID),
		Name:        name,
		Description: description,
		Permissions: permissions,
		IsDefault:   isDefault,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// CheckPermission checks if a user has permission for a resource
func CheckPermission(ctx context.Context, userID int, permission TeamPermission) *PermissionCheckResult {
	// Get user's roles across all teams
	query := `
		SELECT DISTINCT tr.permissions FROM team_members tm
		JOIN team_roles tr ON tm.role_id = tr.id
		WHERE tm.user_id = ? AND tm.status = 'active'
	`
	rows, err := db.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return &PermissionCheckResult{
			Allowed: false,
			Reason:  "Failed to query user roles",
		}
	}
	defer rows.Close()

	var allPermissions []TeamPermission
	for rows.Next() {
		var permsJSON string
		if err := rows.Scan(&permsJSON); err != nil {
			continue
		}
		var perms []TeamPermission
		if err := json.Unmarshal([]byte(permsJSON), &perms); err != nil {
			continue
		}
		allPermissions = append(allPermissions, perms...)
	}

	// Check if user has the required permission
	for _, perm := range allPermissions {
		if perm == permission || perm == PermAdminAll {
			return &PermissionCheckResult{
				Allowed: true,
				Reason:  "Permission granted",
			}
		}
	}

	return &PermissionCheckResult{
		Allowed: false,
		Reason:  "Permission denied",
		Missing: []string{string(permission)},
	}
}

// GrantWorkspaceAccess grants a team access to a workspace
func GrantWorkspaceAccess(ctx context.Context, teamID, workspaceID int, accessLevel string, grantedBy int) error {
	query := `
		INSERT INTO team_workspace_access (team_id, workspace_id, access_level, granted_at, granted_by)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := db.DB.ExecContext(ctx, query, teamID, workspaceID, accessLevel, time.Now(), grantedBy)
	if err != nil {
		return fmt.Errorf("failed to grant workspace access: %w", err)
	}
	return nil
}

// GetTeamWorkspaceAccess retrieves workspace access for a team
func GetTeamWorkspaceAccess(ctx context.Context, teamID int) ([]TeamWorkspaceAccess, error) {
	rows, err := db.DB.QueryContext(ctx,
		"SELECT id, team_id, workspace_id, access_level, granted_at, granted_by FROM team_workspace_access WHERE team_id = ?",
		teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to query workspace access: %w", err)
	}
	defer rows.Close()

	var accesses []TeamWorkspaceAccess
	for rows.Next() {
		var access TeamWorkspaceAccess
		if err := rows.Scan(&access.ID, &access.TeamID, &access.WorkspaceID,
			&access.AccessLevel, &access.GrantedAt, &access.GrantedBy); err != nil {
			return nil, fmt.Errorf("failed to scan access: %w", err)
		}
		accesses = append(accesses, access)
	}

	return accesses, nil
}

// GetUserTeams retrieves all teams a user belongs to
func GetUserTeams(ctx context.Context, userID int) ([]Team, error) {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT t.id, t.organization_id, t.name, t.description, t.created_by, t.created_at, t.updated_at
		FROM teams t
		JOIN team_members tm ON t.id = tm.team_id
		WHERE tm.user_id = ? AND tm.status = 'active'
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user teams: %w", err)
	}
	defer rows.Close()

	var teams []Team
	for rows.Next() {
		var team Team
		if err := rows.Scan(&team.ID, &team.OrganizationID, &team.Name, &team.Description,
			&team.CreatedBy, &team.CreatedAt, &team.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan team: %w", err)
		}
		teams = append(teams, team)
	}

	return teams, nil
}

// HasWorkspaceAccess checks if a team has access to a workspace
func HasWorkspaceAccess(ctx context.Context, teamID, workspaceID int, accessLevel string) bool {
	var accessLevelInt int
	switch accessLevel {
	case "admin":
		accessLevelInt = 3
	case "write":
		accessLevelInt = 2
	case "read":
		accessLevelInt = 1
	default:
		return false
	}

	var userLevel sql.NullInt64
	err := db.DB.QueryRowContext(ctx, `
		SELECT CASE 
			WHEN access_level = 'admin' THEN 3
			WHEN access_level = 'write' THEN 2
			WHEN access_level = 'read' THEN 1
			ELSE 0
		END FROM team_workspace_access
		WHERE team_id = ? AND workspace_id = ?
	`, teamID, workspaceID).Scan(&userLevel)

	if err != nil || !userLevel.Valid {
		return false
	}

	return int(userLevel.Int64) >= accessLevelInt
}
