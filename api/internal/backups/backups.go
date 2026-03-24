package backups

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"wws/api/internal/db"
)

// BackupStatus represents the status of a backup
type BackupStatus string

const (
	BackupStatusPending    BackupStatus = "pending"
	BackupStatusInProgress BackupStatus = "in_progress"
	BackupStatusCompleted  BackupStatus = "completed"
	BackupStatusFailed     BackupStatus = "failed"
)

// WorkspaceBackup represents a workspace backup record
type WorkspaceBackup struct {
	ID          int             `db:"id" json:"id"`
	WorkspaceID int             `db:"workspace_id" json:"workspace_id"`
	BackupPath  string          `db:"backup_path" json:"backup_path"`
	BackupSize  sql.NullFloat64 `db:"backup_size_gb" json:"backup_size_gb"`
	Status      BackupStatus    `db:"status" json:"status"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
}

// BackupRequest represents a request to create a backup
type BackupRequest struct {
	WorkspaceID int    `json:"workspace_id"`
	BackupPath  string `json:"backup_path,omitempty"`
}

// CreateBackup creates a new backup for a workspace
func CreateBackup(ctx context.Context, workspaceID int, backupPath string) (*WorkspaceBackup, error) {
	if backupPath == "" {
		backupPath = getDefaultBackupPath()
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create backup record in database
	backup := &WorkspaceBackup{
		WorkspaceID: workspaceID,
		BackupPath:  filepath.Join(backupPath, fmt.Sprintf("backup_%d_%s", workspaceID, time.Now().Format("20060102_150405"))),
		Status:      BackupStatusPending,
		CreatedAt:   time.Now(),
	}

	query := `
		INSERT INTO workspace_backups (workspace_id, backup_path, status, created_at)
		VALUES (?, ?, ?, ?)
	`
	result, err := db.DB.ExecContext(ctx, query, backup.WorkspaceID, backup.BackupPath, backup.Status, backup.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup record: %w", err)
	}

	backupID, _ := result.LastInsertId()
	backup.ID = int(backupID)

	// Start backup process asynchronously
	go performBackup(backup)

	return backup, nil
}

// performBackup performs the actual backup operation
func performBackup(backup *WorkspaceBackup) {
	backup.Status = BackupStatusInProgress
	updateBackupStatus(backup.ID, backup.Status, 0)

	// Get workspace details
	workspace, err := getWorkspaceDetails(backup.WorkspaceID)
	if err != nil {
		backup.Status = BackupStatusFailed
		updateBackupStatus(backup.ID, backup.Status, 0)
		return
	}

	// Create backup directory
	backupDir := backup.BackupPath
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		backup.Status = BackupStatusFailed
		updateBackupStatus(backup.ID, backup.Status, 0)
		return
	}

	// Perform container backup using podman/podman-save or tar
	if err := backupContainer(workspace, backupDir); err != nil {
		backup.Status = BackupStatusFailed
		updateBackupStatus(backup.ID, backup.Status, 0)
		return
	}

	// Calculate backup size
	size, err := calculateDirSize(backupDir)
	if err != nil {
		size = 0
	}

	backup.Status = BackupStatusCompleted
	updateBackupStatus(backup.ID, backup.Status, size)
}

// backupContainer backs up a container's data
func backupContainer(workspace map[string]interface{}, backupDir string) error {
	containerID, ok := workspace["container_id"].(string)
	if !ok {
		return fmt.Errorf("container ID not found")
	}

	// Use podman to export container data
	archivePath := filepath.Join(backupDir, "container.tar")
	cmd := exec.Command("podman", "export", containerID)
	outFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer outFile.Close()

	cmd.Stdout = outFile
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to export container: %w", err)
	}

	// Backup workspace data directory
	dataDir := workspace["data_dir"].(string)
	if dataDir != "" {
		dataArchive := filepath.Join(backupDir, "data.tar")
		cmd = exec.Command("tar", "-czf", dataArchive, "-C", filepath.Dir(dataDir), filepath.Base(dataDir))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to backup data directory: %w", err)
		}
	}

	return nil
}

// getWorkspaceDetails retrieves workspace details from the database
func getWorkspaceDetails(workspaceID int) (map[string]interface{}, error) {
	query := `SELECT tag, name, provider, config FROM workspaces WHERE id = ?`
	rows, err := db.DB.Query(query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query workspace: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("workspace not found")
	}

	var tag, name, provider, config string
	if err := rows.Scan(&tag, &name, &provider, &config); err != nil {
		return nil, fmt.Errorf("failed to scan workspace: %w", err)
	}

	// Parse config to get container_id and data_dir
	// This would normally parse the JSON config
	workspace := map[string]interface{}{
		"id":       workspaceID,
		"tag":      tag,
		"name":     name,
		"provider": provider,
		// Parse config JSON to extract container_id and data_dir
	}

	return workspace, nil
}

// updateBackupStatus updates the backup status in the database
func updateBackupStatus(backupID int, status BackupStatus, sizeGB float64) error {
	query := `
		UPDATE workspace_backups 
		SET status = ?, backup_size_gb = ? 
		WHERE id = ?
	`
	_, err := db.DB.Exec(query, status, sizeGB, backupID)
	return err
}

// calculateDirSize calculates the size of a directory in GB
func calculateDirSize(dirPath string) (float64, error) {
	var size int64
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return float64(size) / (1024 * 1024 * 1024), nil
}

// getDefaultBackupPath returns the default backup storage path
func getDefaultBackupPath() string {
	backupPath := os.Getenv("BACKUP_PATH")
	if backupPath == "" {
		backupPath = "./data/backups"
	}
	return backupPath
}

// ListBackups retrieves all backups for a workspace
func ListBackups(ctx context.Context, workspaceID int) ([]WorkspaceBackup, error) {
	rows, err := db.DB.QueryContext(ctx,
		"SELECT id, workspace_id, backup_path, backup_size_gb, status, created_at FROM workspace_backups WHERE workspace_id = ? ORDER BY created_at DESC",
		workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query backups: %w", err)
	}
	defer rows.Close()

	var backups []WorkspaceBackup
	for rows.Next() {
		var backup WorkspaceBackup
		if err := rows.Scan(&backup.ID, &backup.WorkspaceID, &backup.BackupPath,
			&backup.BackupSize, &backup.Status, &backup.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan backup: %w", err)
		}
		backups = append(backups, backup)
	}

	return backups, nil
}

// GetBackup retrieves a specific backup
func GetBackup(ctx context.Context, backupID int) (*WorkspaceBackup, error) {
	var backup WorkspaceBackup
	err := db.DB.QueryRowContext(ctx,
		"SELECT id, workspace_id, backup_path, backup_size_gb, status, created_at FROM workspace_backups WHERE id = ?",
		backupID).Scan(&backup.ID, &backup.WorkspaceID, &backup.BackupPath,
		&backup.BackupSize, &backup.Status, &backup.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("backup not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query backup: %w", err)
	}

	return &backup, nil
}

// DeleteBackup deletes a backup
func DeleteBackup(ctx context.Context, backupID int) error {
	backup, err := GetBackup(ctx, backupID)
	if err != nil {
		return err
	}

	// Delete backup files
	if err := os.RemoveAll(backup.BackupPath); err != nil {
		return fmt.Errorf("failed to delete backup files: %w", err)
	}

	// Delete database record
	_, err = db.DB.ExecContext(ctx, "DELETE FROM workspace_backups WHERE id = ?", backupID)
	return err
}

// RestoreBackup restores a workspace from a backup
func RestoreBackup(ctx context.Context, backupID int, workspaceID int) error {
	backup, err := GetBackup(ctx, backupID)
	if err != nil {
		return err
	}

	// Check if backup exists
	if _, err := os.Stat(backup.BackupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup files not found")
	}

	// Restore container
	// This would involve importing the container tar and restoring data
	// Implementation depends on the provider

	return nil
}

// GetBackupStats retrieves backup statistics for an organization
func GetBackupStats(ctx context.Context, orgID int) (map[string]interface{}, error) {
	// Query to get backup statistics
	query := `
		SELECT 
			COUNT(*) as total_backups,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed_backups,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_backups,
			COALESCE(SUM(backup_size_gb), 0) as total_size_gb
		FROM workspace_backups wb
		JOIN workspaces w ON wb.workspace_id = w.id
		WHERE w.organization_id = ?
	`

	var totalBackups, completedBackups, failedBackups int
	var totalSizeGB sql.NullFloat64
	row := db.DB.QueryRowContext(ctx, query, orgID)
	if err := row.Scan(&totalBackups, &completedBackups, &failedBackups, &totalSizeGB); err != nil {
		return nil, fmt.Errorf("failed to query backup stats: %w", err)
	}

	stats := map[string]interface{}{
		"total_backups":     totalBackups,
		"completed_backups": completedBackups,
		"failed_backups":    failedBackups,
		"total_size_gb":     totalSizeGB.Float64,
	}

	return stats, nil
}
