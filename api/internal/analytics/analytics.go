package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"wws/api/internal/db"
)

// RecordWorkspaceUsage records usage metrics for a workspace
func RecordWorkspaceUsage(ctx context.Context, workspaceID int, cpuUsage, memoryUsage, storageGB, networkIn, networkOut float64, uptimeSeconds int64) error {
	query := `
		INSERT INTO workspace_usage (workspace_id, cpu_usage, memory_usage, storage_used_gb, network_in_mb, network_out_mb, uptime_seconds, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.DB.ExecContext(ctx, query,
		workspaceID, cpuUsage, memoryUsage, storageGB, networkIn, networkOut, uptimeSeconds, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to record workspace usage: %w", err)
	}

	return nil
}

// GetWorkspaceUsage retrieves usage history for a workspace
func GetWorkspaceUsage(ctx context.Context, workspaceID int, startTime, endTime time.Time, limit int) ([]WorkspaceUsage, error) {
	query := `
		SELECT id, workspace_id, cpu_usage, memory_usage, storage_used_gb, 
		       network_in_mb, network_out_mb, uptime_seconds, timestamp
		FROM workspace_usage
		WHERE workspace_id = ? AND timestamp BETWEEN ? AND ?
		ORDER BY timestamp DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.DB.QueryContext(ctx, query, workspaceID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query workspace usage: %w", err)
	}
	defer rows.Close()

	var usages []WorkspaceUsage
	for rows.Next() {
		var usage WorkspaceUsage
		err := rows.Scan(
			&usage.ID, &usage.WorkspaceID, &usage.CPUUsage, &usage.MemoryUsage,
			&usage.StorageUsedGB, &usage.NetworkInMB, &usage.NetworkOutMB,
			&usage.UptimeSeconds, &usage.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan usage record: %w", err)
		}
		usages = append(usages, usage)
	}

	return usages, nil
}

// GetOrganizationUsage retrieves aggregated usage for an organization
func GetOrganizationUsage(ctx context.Context, orgID int) (*OrganizationUsage, error) {
	query := `
		SELECT 
			w.organization_id,
			COUNT(DISTINCT w.id) as total_workspaces,
			SUM(CASE WHEN w.status = 'running' THEN 1 ELSE 0 END) as active_workspaces,
			COALESCE(AVG(usage.cpu_usage), 0) as total_cpu_usage,
			COALESCE(AVG(usage.memory_usage), 0) as total_memory_usage,
			COALESCE(SUM(usage.storage_used_gb), 0) as total_storage_gb,
			COALESCE(SUM(usage.network_in_mb + usage.network_out_mb), 0) as total_network_mb,
			COALESCE(AVG(usage.uptime_seconds), 0) as average_uptime,
			MAX(usage.timestamp) as last_updated
		FROM workspaces w
		LEFT JOIN workspace_usage usage ON w.id = usage.workspace_id
		WHERE w.organization_id = ? AND w.deleted_at IS NULL
		GROUP BY w.organization_id
	`

	var usage OrganizationUsage
	var lastUpdated sql.NullTime
	err := db.DB.QueryRowContext(ctx, query, orgID).Scan(
		&usage.OrganizationID, &usage.TotalWorkspaces, &usage.ActiveWorkspaces,
		&usage.TotalCPUUsage, &usage.TotalMemoryUsage, &usage.TotalStorageGB,
		&usage.TotalNetworkMB, &usage.AverageUptime, &lastUpdated,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query organization usage: %w", err)
	}

	if lastUpdated.Valid {
		usage.LastUpdated = lastUpdated.Time
	} else {
		usage.LastUpdated = time.Now()
	}

	return &usage, nil
}

// GetUserWorkspaceActivity retrieves workspace activity for a user
func GetUserWorkspaceActivity(ctx context.Context, userID int) ([]UserWorkspaceActivity, error) {
	query := `
		SELECT 
			w.owner_id as user_id,
			w.id as workspace_id,
			COUNT(DISTINCT al.id) as login_count,
			COALESCE(SUM(TIMESTAMPDIFF(SECOND, al.created_at, COALESCE((
				SELECT MIN(al2.created_at) 
				FROM audit_logs al2 
				WHERE al2.user_id = al.user_id 
				AND al2.resource_id = al.resource_id 
				AND al2.created_at > al.created_at
			), NOW()))), 0) as total_session_time,
			MAX(al.created_at) as last_login,
			MIN(al.created_at) as first_login
		FROM workspaces w
		LEFT JOIN audit_logs al ON al.resource_id = w.id AND al.resource_type = 'workspace'
		WHERE w.owner_id = ? AND w.deleted_at IS NULL
		GROUP BY w.id, w.owner_id
	`

	rows, err := db.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user workspace activity: %w", err)
	}
	defer rows.Close()

	var activities []UserWorkspaceActivity
	for rows.Next() {
		var activity UserWorkspaceActivity
		err := rows.Scan(
			&activity.UserID, &activity.WorkspaceID, &activity.LoginCount,
			&activity.TotalSessionTime, &activity.LastLogin, &activity.FirstLogin,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan activity record: %w", err)
		}
		activities = append(activities, activity)
	}

	return activities, nil
}

// GetWorkspaceStats retrieves detailed statistics for a workspace
func GetWorkspaceStats(ctx context.Context, workspaceID int) (*WorkspaceStats, error) {
	// Get workspace details
	var workspaceTag, workspaceName string
	var createdAt time.Time

	err := db.DB.QueryRowContext(ctx,
		"SELECT tag, name, created_at FROM workspaces WHERE id = ? AND deleted_at IS NULL",
		workspaceID,
	).Scan(&workspaceTag, &workspaceName, &createdAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workspace not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query workspace: %w", err)
	}

	// Get usage statistics
	query := `
		SELECT 
			COALESCE(AVG(cpu_usage), 0) as avg_cpu,
			COALESCE(AVG(memory_usage), 0) as avg_memory,
			COALESCE(AVG(storage_used_gb), 0) as avg_storage,
			COALESCE(SUM(network_in_mb + network_out_mb), 0) as total_network,
			COALESCE(MAX(uptime_seconds), 0) as max_uptime,
			MAX(timestamp) as last_active
		FROM workspace_usage
		WHERE workspace_id = ?
	`

	var stats WorkspaceStats
	stats.WorkspaceID = workspaceID
	stats.WorkspaceTag = workspaceTag
	stats.WorkspaceName = workspaceName
	stats.CreatedAt = createdAt

	var lastActiveStr sql.NullString
	err = db.DB.QueryRowContext(ctx, query, workspaceID).Scan(
		&stats.AverageCPU, &stats.AverageMemory, &stats.AverageStorage,
		&stats.TotalNetworkMB, &stats.UptimeSeconds, &lastActiveStr,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query workspace stats: %w", err)
	}

	if lastActiveStr.Valid && lastActiveStr.String != "" {
		// Try to parse as time, fallback to created_at
		if t, err := time.Parse("2006-01-02 15:04:05", lastActiveStr.String); err == nil {
			stats.LastActiveAt = t
		} else if t, err := time.Parse(time.RFC3339, lastActiveStr.String); err == nil {
			stats.LastActiveAt = t
		} else {
			stats.LastActiveAt = stats.CreatedAt
		}
	} else {
		stats.LastActiveAt = stats.CreatedAt
	}

	stats.TotalUptimeHours = float64(stats.UptimeSeconds) / 3600.0

	return &stats, nil
}

// GetAnalyticsSummary retrieves a comprehensive analytics summary
func GetAnalyticsSummary(ctx context.Context, orgID *int) (*AnalyticsSummary, error) {
	summary := &AnalyticsSummary{}

	// Get total workspaces
	var totalWorkspaces int
	orgQuery := "SELECT COUNT(*) FROM workspaces WHERE deleted_at IS NULL"
	if orgID != nil {
		orgQuery += " AND organization_id = ?"
		rows, err := db.DB.QueryContext(ctx, orgQuery, *orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to count workspaces: %w", err)
		}
		defer rows.Close()
		rows.Next()
		rows.Scan(&totalWorkspaces)
	} else {
		err := db.DB.QueryRowContext(ctx, orgQuery).Scan(&totalWorkspaces)
		if err != nil {
			return nil, fmt.Errorf("failed to count workspaces: %w", err)
		}
	}
	summary.TotalWorkspaces = totalWorkspaces

	// Get active workspaces
	var activeWorkspaces int
	activeQuery := "SELECT COUNT(*) FROM workspaces WHERE status = 'running' AND deleted_at IS NULL"
	if orgID != nil {
		activeQuery += " AND organization_id = ?"
		rows, err := db.DB.QueryContext(ctx, activeQuery, *orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to count active workspaces: %w", err)
		}
		defer rows.Close()
		rows.Next()
		rows.Scan(&activeWorkspaces)
	} else {
		err := db.DB.QueryRowContext(ctx, activeQuery).Scan(&activeWorkspaces)
		if err != nil {
			return nil, fmt.Errorf("failed to count active workspaces: %w", err)
		}
	}
	summary.ActiveWorkspaces = activeWorkspaces

	// Get top workspaces by usage
	topQuery := `
		SELECT w.id, w.tag, w.name,
		       COALESCE(AVG(usage.cpu_usage), 0) as avg_cpu,
		       COALESCE(AVG(usage.memory_usage), 0) as avg_memory,
		       COALESCE(AVG(usage.storage_used_gb), 0) as avg_storage,
		       COALESCE(SUM(usage.network_in_mb + usage.network_out_mb), 0) as total_network,
		       COALESCE(MAX(usage.uptime_seconds), 0) as uptime_seconds,
		       MAX(usage.timestamp) as last_active,
		       w.created_at
		FROM workspaces w
		LEFT JOIN workspace_usage usage ON w.id = usage.workspace_id
		WHERE w.deleted_at IS NULL
		`
	if orgID != nil {
		topQuery += " AND w.organization_id = ?"
	}
	topQuery += ` GROUP BY w.id 
		ORDER BY avg_cpu DESC, avg_memory DESC
		LIMIT 10
	`

	rows, err := db.DB.QueryContext(ctx, topQuery, func() interface{} {
		if orgID != nil {
			return *orgID
		}
		return nil
	}())
	if err != nil {
		return nil, fmt.Errorf("failed to query top workspaces: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var stats WorkspaceStats
		var uptimeSeconds int64
		var lastActive sql.NullTime

		err := rows.Scan(
			&stats.WorkspaceID, &stats.WorkspaceTag, &stats.WorkspaceName,
			&stats.AverageCPU, &stats.AverageMemory, &stats.AverageStorage,
			&stats.TotalNetworkMB, &uptimeSeconds, &lastActive, &stats.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workspace stats: %w", err)
		}
		stats.TotalUptimeHours = float64(uptimeSeconds) / 3600.0
		if lastActive.Valid {
			stats.LastActiveAt = lastActive.Time
		} else {
			stats.LastActiveAt = stats.CreatedAt
		}
		summary.TopWorkspacesByUsage = append(summary.TopWorkspacesByUsage, stats)
	}

	return summary, nil
}

// CreateUsageAlert creates a new usage alert
func CreateUsageAlert(ctx context.Context, orgID int, workspaceID *int, alertType, severity, message string, value, threshold float64) error {
	query := `
		INSERT INTO usage_alerts (organization_id, workspace_id, alert_type, severity, message, value, threshold, resolved, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?)
	`

	_, err := db.DB.ExecContext(ctx, query,
		orgID, workspaceID, alertType, severity, message, value, threshold, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to create usage alert: %w", err)
	}

	return nil
}

// GetActiveAlerts retrieves unresolved usage alerts
func GetActiveAlerts(ctx context.Context, orgID int) ([]UsageAlert, error) {
	query := `
		SELECT id, organization_id, workspace_id, alert_type, severity, message,
		       value, threshold, resolved, created_at, resolved_at
		FROM usage_alerts
		WHERE organization_id = ? AND resolved = 0
		ORDER BY created_at DESC
	`

	rows, err := db.DB.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query active alerts: %w", err)
	}
	defer rows.Close()

	var alerts []UsageAlert
	for rows.Next() {
		var alert UsageAlert
		err := rows.Scan(
			&alert.ID, &alert.OrganizationID, &alert.WorkspaceID,
			&alert.AlertType, &alert.Severity, &alert.Message,
			&alert.Value, &alert.Threshold, &alert.Resolved,
			&alert.CreatedAt, &alert.ResolvedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// ResolveAlert marks an alert as resolved
func ResolveAlert(ctx context.Context, alertID int) error {
	_, err := db.DB.ExecContext(ctx,
		"UPDATE usage_alerts SET resolved = 1, resolved_at = ? WHERE id = ?",
		time.Now(), alertID,
	)
	if err != nil {
		return fmt.Errorf("failed to resolve alert: %w", err)
	}

	return nil
}
