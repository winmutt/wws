package alerts

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"wws/api/internal/db"
)

// AlertSeverity represents the severity of an alert
type AlertSeverity string

const (
	SeverityLow      AlertSeverity = "low"
	SeverityMedium   AlertSeverity = "medium"
	SeverityHigh     AlertSeverity = "high"
	SeverityCritical AlertSeverity = "critical"
)

// AlertType represents the type of resource alert
type AlertType string

const (
	AlertTypeCPUUsage      AlertType = "cpu_usage"
	AlertTypeMemoryUsage   AlertType = "memory_usage"
	AlertTypeStorageUsage  AlertType = "storage_usage"
	AlertTypeNetworkUsage  AlertType = "network_usage"
	AlertTypeQuotaExceeded AlertType = "quota_exceeded"
	AlertTypeIdleTimeout   AlertType = "idle_timeout"
	AlertTypeBackupFailed  AlertType = "backup_failed"
	AlertTypeProvisioning  AlertType = "provisioning"
	AlertTypeCostThreshold AlertType = "cost_threshold"
)

// ResourceAlert represents a resource alert
type ResourceAlert struct {
	ID             int             `db:"id" json:"id"`
	OrganizationID int             `db:"organization_id" json:"organization_id"`
	WorkspaceID    sql.NullInt64   `db:"workspace_id" json:"workspace_id"`
	AlertType      AlertType       `db:"alert_type" json:"alert_type"`
	Severity       AlertSeverity   `db:"severity" json:"severity"`
	Message        string          `db:"message" json:"message"`
	Value          sql.NullFloat64 `db:"value" json:"value"`
	Threshold      sql.NullFloat64 `db:"threshold" json:"threshold"`
	Acknowledged   bool            `db:"acknowledged" json:"acknowledged"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	AcknowledgedAt sql.NullTime    `db:"acknowledged_at" json:"acknowledged_at"`
}

// CreateAlert creates a new resource alert
func CreateAlert(ctx context.Context, orgID int, workspaceID *int, alertType AlertType, severity AlertSeverity, message string, value, threshold *float64) (*ResourceAlert, error) {
	alert := &ResourceAlert{
		OrganizationID: orgID,
		AlertType:      alertType,
		Severity:       severity,
		Message:        message,
		CreatedAt:      time.Now(),
	}

	if workspaceID != nil {
		alert.WorkspaceID = sql.NullInt64{Int64: int64(*workspaceID), Valid: true}
	}

	if value != nil {
		alert.Value = sql.NullFloat64{Float64: *value, Valid: true}
	}

	if threshold != nil {
		alert.Threshold = sql.NullFloat64{Float64: *threshold, Valid: true}
	}

	query := `
		INSERT INTO resource_alerts (organization_id, workspace_id, alert_type, severity, message, value, threshold, acknowledged, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?)
	`
	result, err := db.DB.ExecContext(ctx, query,
		alert.OrganizationID,
		alert.WorkspaceID.Int64,
		alert.AlertType,
		alert.Severity,
		alert.Message,
		alert.Value.Float64,
		alert.Threshold.Float64,
		alert.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create alert: %w", err)
	}

	alertID, _ := result.LastInsertId()
	alert.ID = int(alertID)

	return alert, nil
}

// GetActiveAlerts retrieves all active (unacknowledged) alerts
func GetActiveAlerts(ctx context.Context, orgID int, workspaceID *int, alertType *AlertType, severity *AlertSeverity) ([]ResourceAlert, error) {
	query := `
		SELECT id, organization_id, workspace_id, alert_type, severity, message, value, threshold, acknowledged, created_at, acknowledged_at
		FROM resource_alerts
		WHERE organization_id = ? AND acknowledged = 0
	`
	args := []interface{}{orgID}

	if workspaceID != nil {
		query += " AND workspace_id = ?"
		args = append(args, *workspaceID)
	}

	if alertType != nil {
		query += " AND alert_type = ?"
		args = append(args, *alertType)
	}

	if severity != nil {
		query += " AND severity = ?"
		args = append(args, *severity)
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query alerts: %w", err)
	}
	defer rows.Close()

	var alerts []ResourceAlert
	for rows.Next() {
		var alert ResourceAlert
		if err := rows.Scan(&alert.ID, &alert.OrganizationID, &alert.WorkspaceID,
			&alert.AlertType, &alert.Severity, &alert.Message,
			&alert.Value, &alert.Threshold, &alert.Acknowledged,
			&alert.CreatedAt, &alert.AcknowledgedAt); err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// AcknowledgeAlert marks an alert as acknowledged
func AcknowledgeAlert(ctx context.Context, alertID int) error {
	query := `
		UPDATE resource_alerts 
		SET acknowledged = 1, acknowledged_at = ? 
		WHERE id = ?
	`
	_, err := db.DB.ExecContext(ctx, query, time.Now(), alertID)
	if err != nil {
		return fmt.Errorf("failed to acknowledge alert: %w", err)
	}
	return nil
}

// GetAlertHistory retrieves alert history for an organization
func GetAlertHistory(ctx context.Context, orgID int, workspaceID *int, startDate, endDate *time.Time, limit int) ([]ResourceAlert, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, organization_id, workspace_id, alert_type, severity, message, value, threshold, acknowledged, created_at, acknowledged_at
		FROM resource_alerts
		WHERE organization_id = ?
	`
	args := []interface{}{orgID}

	if workspaceID != nil {
		query += " AND workspace_id = ?"
		args = append(args, *workspaceID)
	}

	if startDate != nil {
		query += " AND created_at >= ?"
		args = append(args, *startDate)
	}

	if endDate != nil {
		query += " AND created_at <= ?"
		args = append(args, *endDate)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query alert history: %w", err)
	}
	defer rows.Close()

	var alerts []ResourceAlert
	for rows.Next() {
		var alert ResourceAlert
		if err := rows.Scan(&alert.ID, &alert.OrganizationID, &alert.WorkspaceID,
			&alert.AlertType, &alert.Severity, &alert.Message,
			&alert.Value, &alert.Threshold, &alert.Acknowledged,
			&alert.CreatedAt, &alert.AcknowledgedAt); err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// GetAlertSummary retrieves a summary of alerts for an organization
func GetAlertSummary(ctx context.Context, orgID int) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total_alerts,
			SUM(CASE WHEN acknowledged = 0 THEN 1 ELSE 0 END) as active_alerts,
			SUM(CASE WHEN acknowledged = 1 THEN 1 ELSE 0 END) as acknowledged_alerts,
			SUM(CASE WHEN severity = 'critical' AND acknowledged = 0 THEN 1 ELSE 0 END) as critical_alerts,
			SUM(CASE WHEN severity = 'high' AND acknowledged = 0 THEN 1 ELSE 0 END) as high_alerts
		FROM resource_alerts
		WHERE organization_id = ?
	`

	var totalAlerts, activeAlerts, acknowledgedAlerts, criticalAlerts, highAlerts sql.NullInt64
	err := db.DB.QueryRowContext(ctx, query, orgID).Scan(
		&totalAlerts, &activeAlerts, &acknowledgedAlerts, &criticalAlerts, &highAlerts)
	if err != nil {
		return nil, fmt.Errorf("failed to query alert summary: %w", err)
	}

	summary := map[string]interface{}{
		"total_alerts":        totalAlerts.Int64,
		"active_alerts":       activeAlerts.Int64,
		"acknowledged_alerts": acknowledgedAlerts.Int64,
		"critical_alerts":     criticalAlerts.Int64,
		"high_alerts":         highAlerts.Int64,
	}

	return summary, nil
}

// CheckAndCreateAlert checks if an alert should be created based on thresholds
func CheckAndCreateAlert(ctx context.Context, orgID int, workspaceID int, alertType AlertType, currentValue, threshold float64, severity AlertSeverity) (*ResourceAlert, error) {
	if currentValue <= threshold {
		return nil, nil // No alert needed
	}

	message := fmt.Sprintf("%s threshold exceeded: %.2f > %.2f", alertType, currentValue, threshold)
	return CreateAlert(ctx, orgID, &workspaceID, alertType, severity, message, &currentValue, &threshold)
}

// CheckCPUUsage checks CPU usage and creates alert if threshold exceeded
func CheckCPUUsage(ctx context.Context, orgID int, workspaceID int, cpuUsage, threshold float64) (*ResourceAlert, error) {
	severity := SeverityMedium
	if cpuUsage > 90 {
		severity = SeverityCritical
	} else if cpuUsage > 80 {
		severity = SeverityHigh
	}

	return CheckAndCreateAlert(ctx, orgID, workspaceID, AlertTypeCPUUsage, cpuUsage, threshold, severity)
}

// CheckMemoryUsage checks memory usage and creates alert if threshold exceeded
func CheckMemoryUsage(ctx context.Context, orgID int, workspaceID int, memoryUsage, threshold float64) (*ResourceAlert, error) {
	severity := SeverityMedium
	if memoryUsage > 90 {
		severity = SeverityCritical
	} else if memoryUsage > 80 {
		severity = SeverityHigh
	}

	return CheckAndCreateAlert(ctx, orgID, workspaceID, AlertTypeMemoryUsage, memoryUsage, threshold, severity)
}

// CheckStorageUsage checks storage usage and creates alert if threshold exceeded
func CheckStorageUsage(ctx context.Context, orgID int, workspaceID int, storageUsage, threshold float64) (*ResourceAlert, error) {
	severity := SeverityLow
	if storageUsage > 90 {
		severity = SeverityHigh
	} else if storageUsage > 80 {
		severity = SeverityMedium
	}

	return CheckAndCreateAlert(ctx, orgID, workspaceID, AlertTypeStorageUsage, storageUsage, threshold, severity)
}

// DeleteOldAlerts deletes acknowledged alerts older than specified days
func DeleteOldAlerts(ctx context.Context, orgID int, daysOld int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -daysOld)
	query := `
		DELETE FROM resource_alerts 
		WHERE organization_id = ? AND acknowledged = 1 AND acknowledged_at < ?
	`
	result, err := db.DB.ExecContext(ctx, query, orgID, cutoffDate)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old alerts: %w", err)
	}

	affected, _ := result.RowsAffected()
	return affected, nil
}
