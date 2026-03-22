package idlemgmt

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"wws/api/internal/db"
)

const (
	DefaultIdleTimeoutHours   = 6
	DefaultWarningThreshold   = 5 // hours
	DefaultGracePeriodMinutes = 15
)

// GetIdleConfig retrieves idle configuration for an organization
func GetIdleConfig(ctx context.Context, orgID int) (*IdleConfig, error) {
	query := `
		SELECT id, organization_id, idle_timeout_hours, warning_threshold_hours,
		       auto_shutdown_enabled, shutdown_grace_period, updated_at
		FROM idle_config WHERE organization_id = ?
	`

	var config IdleConfig
	var updatedAt time.Time

	err := db.DB.QueryRowContext(ctx, query, orgID).Scan(
		&config.ID, &config.OrganizationID, &config.IdleTimeoutHours,
		&config.WarningThresholdHours, &config.AutoShutdownEnabled,
		&config.ShutdownGracePeriod, &updatedAt,
	)

	if err == sql.ErrNoRows {
		// Return default config if none exists
		return &IdleConfig{
			OrganizationID:        orgID,
			IdleTimeoutHours:      DefaultIdleTimeoutHours,
			WarningThresholdHours: DefaultWarningThreshold,
			AutoShutdownEnabled:   true,
			ShutdownGracePeriod:   DefaultGracePeriodMinutes,
			UpdatedAt:             time.Now(),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query idle config: %w", err)
	}

	config.UpdatedAt = updatedAt

	// Load exempt users
	exemptUsers, err := getExemptUsers(ctx, orgID)
	if err == nil {
		config.ExemptUsers = exemptUsers
	}

	// Load exempt workspaces
	exemptWorkspaces, err := getExemptWorkspaces(ctx, orgID)
	if err == nil {
		config.ExemptWorkspaces = exemptWorkspaces
	}

	return &config, nil
}

// UpdateIdleConfig updates idle configuration for an organization
func UpdateIdleConfig(ctx context.Context, orgID int, timeoutHours int, warningHours int, autoShutdown bool, gracePeriod int) (*IdleConfig, error) {
	if timeoutHours < 1 {
		return nil, fmt.Errorf("idle timeout must be at least 1 hour")
	}
	if warningHours >= timeoutHours {
		return nil, fmt.Errorf("warning threshold must be less than idle timeout")
	}
	if gracePeriod < 0 {
		return nil, fmt.Errorf("grace period cannot be negative")
	}

	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Upsert idle config
	query := `
		INSERT INTO idle_config (organization_id, idle_timeout_hours, warning_threshold_hours, auto_shutdown_enabled, shutdown_grace_period, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(organization_id) DO UPDATE SET
			idle_timeout_hours = excluded.idle_timeout_hours,
			warning_threshold_hours = excluded.warning_threshold_hours,
			auto_shutdown_enabled = excluded.auto_shutdown_enabled,
			shutdown_grace_period = excluded.shutdown_grace_period,
			updated_at = excluded.updated_at
	`

	_, err = tx.ExecContext(ctx, query,
		orgID, timeoutHours, warningHours, func() int {
			if autoShutdown {
				return 1
			}
			return 0
		}(),
		gracePeriod, time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update idle config: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return GetIdleConfig(ctx, orgID)
}

// GetWorkspaceIdleStatus checks the idle status of a workspace
func GetWorkspaceIdleStatus(ctx context.Context, workspaceID int, orgConfig *IdleConfig) (*WorkspaceIdleStatus, error) {
	// Check if workspace is exempt
	if isWorkspaceExempt(workspaceID, orgConfig.ExemptWorkspaces) {
		return &WorkspaceIdleStatus{
			WorkspaceID: workspaceID,
			Exempt:      true,
		}, nil
	}

	// Get last activity time from audit logs
	var lastActiveAt sql.NullTime
	err := db.DB.QueryRowContext(ctx, `
		SELECT MAX(created_at) FROM audit_logs 
		WHERE resource_id = ? AND resource_type = 'workspace'
	`, workspaceID).Scan(&lastActiveAt)

	if err != nil {
		return nil, fmt.Errorf("failed to query last activity: %w", err)
	}

	status := &WorkspaceIdleStatus{
		WorkspaceID: workspaceID,
	}

	if !lastActiveAt.Valid {
		// No activity recorded, consider it active
		status.LastActiveAt = time.Now()
		status.IsIdle = false
		return status, nil
	}

	status.LastActiveAt = lastActiveAt.Time
	idleDuration := time.Since(status.LastActiveAt)
	status.IdleDuration = idleDuration

	// Check if workspace is idle
	if orgConfig != nil && idleDuration > time.Duration(orgConfig.IdleTimeoutHours)*time.Hour {
		status.IsIdle = true
		shutdownTime := status.LastActiveAt.Add(time.Duration(orgConfig.IdleTimeoutHours) * time.Hour)
		status.WillShutdownAt = &shutdownTime

		reason := fmt.Sprintf("Idle for %v exceeding threshold of %vh",
			idleDuration, orgConfig.IdleTimeoutHours)
		status.ShutdownReason = &reason
	}

	return status, nil
}

// GetIdleWorkspaces retrieves all idle workspaces for an organization
func GetIdleWorkspaces(ctx context.Context, orgID int) ([]WorkspaceIdleStatus, error) {
	config, err := GetIdleConfig(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get idle config: %w", err)
	}

	// Get all active workspaces for the organization
	rows, err := db.DB.QueryContext(ctx, `
		SELECT id FROM workspaces 
		WHERE organization_id = ? AND status = 'running' AND deleted_at IS NULL
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query workspaces: %w", err)
	}
	defer rows.Close()

	var idleWorkspaces []WorkspaceIdleStatus
	for rows.Next() {
		var workspaceID int
		if err := rows.Scan(&workspaceID); err != nil {
			continue
		}

		status, err := GetWorkspaceIdleStatus(ctx, workspaceID, config)
		if err != nil {
			continue
		}

		if status.IsIdle {
			idleWorkspaces = append(idleWorkspaces, *status)
		}
	}

	return idleWorkspaces, nil
}

// ScheduleIdleShutdown schedules a shutdown for an idle workspace
func ScheduleIdleShutdown(ctx context.Context, workspaceID int, reason string) error {
	// Get workspace details
	var orgID int
	err := db.DB.QueryRowContext(ctx,
		"SELECT organization_id FROM workspaces WHERE id = ?", workspaceID).Scan(&orgID)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	// Record the shutdown event
	_, err = db.DB.ExecContext(ctx, `
		INSERT INTO idle_shutdown_events (workspace_id, organization_id, idle_duration_minutes, shutdown_at, triggered_by, reason)
		VALUES (?, ?, 0, ?, 'auto', ?)
	`, workspaceID, orgID, time.Now(), reason)
	if err != nil {
		return fmt.Errorf("failed to record shutdown event: %w", err)
	}

	return nil
}

// CheckAndShutdownIdleWorkspaces checks for idle workspaces and shuts them down
func CheckAndShutdownIdleWorkspaces(ctx context.Context, orgID int) ([]WorkspaceIdleStatus, error) {
	config, err := GetIdleConfig(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get idle config: %w", err)
	}

	if !config.AutoShutdownEnabled {
		return nil, nil
	}

	idleWorkspaces, err := GetIdleWorkspaces(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get idle workspaces: %w", err)
	}

	var shutdownWorkspaces []WorkspaceIdleStatus
	for _, ws := range idleWorkspaces {
		if err := ScheduleIdleShutdown(ctx, ws.WorkspaceID,
			fmt.Sprintf("Auto-shutdown due to idle for %v", ws.IdleDuration)); err != nil {
			continue
		}
		shutdownWorkspaces = append(shutdownWorkspaces, ws)
	}

	return shutdownWorkspaces, nil
}

// getExemptUsers retrieves exempt user IDs for an organization
func getExemptUsers(ctx context.Context, orgID int) ([]int, error) {
	rows, err := db.DB.QueryContext(ctx,
		"SELECT user_id FROM idle_exempt_users WHERE organization_id = ?", orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []int
	for rows.Next() {
		var userID int
		if err := rows.Scan(&userID); err != nil {
			continue
		}
		users = append(users, userID)
	}

	return users, nil
}

// getExemptWorkspaces retrieves exempt workspace IDs for an organization
func getExemptWorkspaces(ctx context.Context, orgID int) ([]int, error) {
	rows, err := db.DB.QueryContext(ctx,
		"SELECT workspace_id FROM idle_exempt_workspaces WHERE organization_id = ?", orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []int
	for rows.Next() {
		var wsID int
		if err := rows.Scan(&wsID); err != nil {
			continue
		}
		workspaces = append(workspaces, wsID)
	}

	return workspaces, nil
}

// isWorkspaceExempt checks if a workspace is in the exempt list
func isWorkspaceExempt(workspaceID int, exemptList []int) bool {
	for _, id := range exemptList {
		if id == workspaceID {
			return true
		}
	}
	return false
}
