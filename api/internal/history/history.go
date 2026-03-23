package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"wws/api/internal/db"
)

// WorkspaceAction represents an action performed on a workspace
type WorkspaceAction string

const (
	ActionCreated   WorkspaceAction = "created"
	ActionUpdated   WorkspaceAction = "updated"
	ActionDeleted   WorkspaceAction = "deleted"
	ActionStarted   WorkspaceAction = "started"
	ActionStopped   WorkspaceAction = "stopped"
	ActionRestarted WorkspaceAction = "restarted"
	ActionPaused    WorkspaceAction = "paused"
	ActionResumed   WorkspaceAction = "resumed"
	ActionCloned    WorkspaceAction = "cloned"
	ActionRestored  WorkspaceAction = "restored"
	ActionBackup    WorkspaceAction = "backup"
	ActionSnapshot  WorkspaceAction = "snapshot"
)

// WorkspaceHistory represents a historical record of workspace changes
type WorkspaceHistory struct {
	ID            int             `db:"id" json:"id"`
	WorkspaceID   int             `db:"workspace_id" json:"workspace_id"`
	Action        WorkspaceAction `db:"action" json:"action"`
	PreviousState sql.NullString  `db:"previous_state" json:"previous_state"`
	NewState      sql.NullString  `db:"new_state" json:"new_state"`
	PerformedBy   sql.NullInt64   `db:"performed_by" json:"performed_by"`
	CreatedAt     time.Time       `db:"created_at" json:"created_at"`
}

// RecordHistory records a workspace action in the history
func RecordHistory(ctx context.Context, workspaceID int, action WorkspaceAction, previousState, newState map[string]interface{}, performedBy int) error {
	var prevStateJSON, newStateJSON sql.NullString

	if previousState != nil {
		prevBytes, err := json.Marshal(previousState)
		if err == nil {
			prevStateJSON = sql.NullString{String: string(prevBytes), Valid: true}
		}
	}

	if newState != nil {
		newBytes, err := json.Marshal(newState)
		if err == nil {
			newStateJSON = sql.NullString{String: string(newBytes), Valid: true}
		}
	}

	query := `
		INSERT INTO workspace_history (workspace_id, action, previous_state, new_state, performed_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := db.DB.ExecContext(ctx, query, workspaceID, action, prevStateJSON, newStateJSON, performedBy, time.Now())
	if err != nil {
		return fmt.Errorf("failed to record history: %w", err)
	}

	return nil
}

// GetWorkspaceHistory retrieves the history for a workspace
func GetWorkspaceHistory(ctx context.Context, workspaceID int, limit, offset int) ([]WorkspaceHistory, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := db.DB.QueryContext(ctx, `
		SELECT id, workspace_id, action, previous_state, new_state, performed_by, created_at
		FROM workspace_history
		WHERE workspace_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, workspaceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	var histories []WorkspaceHistory
	for rows.Next() {
		var history WorkspaceHistory
		if err := rows.Scan(&history.ID, &history.WorkspaceID, &history.Action,
			&history.PreviousState, &history.NewState, &history.PerformedBy, &history.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan history: %w", err)
		}
		histories = append(histories, history)
	}

	return histories, nil
}

// GetHistoryByAction retrieves history entries filtered by action type
func GetHistoryByAction(ctx context.Context, workspaceID int, action WorkspaceAction, limit int) ([]WorkspaceHistory, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.DB.QueryContext(ctx, `
		SELECT id, workspace_id, action, previous_state, new_state, performed_by, created_at
		FROM workspace_history
		WHERE workspace_id = ? AND action = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, workspaceID, action, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query history by action: %w", err)
	}
	defer rows.Close()

	var histories []WorkspaceHistory
	for rows.Next() {
		var history WorkspaceHistory
		if err := rows.Scan(&history.ID, &history.WorkspaceID, &history.Action,
			&history.PreviousState, &history.NewState, &history.PerformedBy, &history.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan history: %w", err)
		}
		histories = append(histories, history)
	}

	return histories, nil
}

// GetStateAtTime retrieves the workspace state at a specific point in time
func GetStateAtTime(ctx context.Context, workspaceID int, timestamp time.Time) (map[string]interface{}, error) {
	query := `
		SELECT new_state FROM workspace_history
		WHERE workspace_id = ? AND created_at <= ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	var stateJSON sql.NullString
	err := db.DB.QueryRowContext(ctx, query, workspaceID, timestamp).Scan(&stateJSON)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no state found at specified time")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query state: %w", err)
	}

	if !stateJSON.Valid || stateJSON.String == "" {
		return nil, nil
	}

	var state map[string]interface{}
	if err := json.Unmarshal([]byte(stateJSON.String), &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return state, nil
}

// GetHistorySummary retrieves a summary of workspace history
func GetHistorySummary(ctx context.Context, workspaceID int) (map[string]interface{}, error) {
	query := `
		SELECT 
			action,
			COUNT(*) as count,
			MIN(created_at) as first_occurrence,
			MAX(created_at) as last_occurrence
		FROM workspace_history
		WHERE workspace_id = ?
		GROUP BY action
		ORDER BY count DESC
	`

	rows, err := db.DB.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query history summary: %w", err)
	}
	defer rows.Close()

	summary := make(map[string]interface{})
	actionCounts := make(map[string]int)

	for rows.Next() {
		var action string
		var count int
		var firstTime, lastTime time.Time
		if err := rows.Scan(&action, &count, &firstTime, &lastTime); err != nil {
			continue
		}
		actionCounts[action] = count
		summary[action] = map[string]interface{}{
			"count":            count,
			"first_occurrence": firstTime,
			"last_occurrence":  lastTime,
		}
	}

	summary["total_actions"] = len(actionCounts)
	summary["action_counts"] = actionCounts

	return summary, nil
}

// DeleteOldHistory deletes history entries older than a specified date
func DeleteOldHistory(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM workspace_history WHERE created_at < ?`
	result, err := db.DB.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old history: %w", err)
	}

	affected, _ := result.RowsAffected()
	return affected, nil
}

// GetWorkspaceLifecycle retrieves the complete lifecycle of a workspace
func GetWorkspaceLifecycle(ctx context.Context, workspaceID int) (map[string]interface{}, error) {
	// Get first and last history entries
	query := `
		SELECT 
			MIN(created_at) as created,
			MAX(CASE WHEN action = 'started' THEN created_at END) as first_started,
			MAX(CASE WHEN action = 'stopped' THEN created_at END) as last_stopped,
			MAX(created_at) as last_activity
		FROM workspace_history
		WHERE workspace_id = ?
	`

	var created, firstStarted, lastStopped, lastActivity sql.NullTime
	err := db.DB.QueryRowContext(ctx, query, workspaceID).Scan(&created, &firstStarted, &lastStopped, &lastActivity)
	if err != nil {
		return nil, fmt.Errorf("failed to query lifecycle: %w", err)
	}

	lifecycle := map[string]interface{}{
		"workspace_id": workspaceID,
	}

	if created.Valid {
		lifecycle["created"] = created.Time
	}
	if firstStarted.Valid {
		lifecycle["first_started"] = firstStarted.Time
	}
	if lastStopped.Valid {
		lifecycle["last_stopped"] = lastStopped.Time
	}
	if lastActivity.Valid {
		lifecycle["last_activity"] = lastActivity.Time
	}

	// Calculate total uptime
	if firstStarted.Valid && lastStopped.Valid {
		lifecycle["total_uptime_seconds"] = lastStopped.Time.Sub(firstStarted.Time).Seconds()
	}

	return lifecycle, nil
}
