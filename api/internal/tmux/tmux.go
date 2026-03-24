package tmux

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"time"

	"wws/api/internal/db"
)

// TmuxSession represents a tmux session
type TmuxSession struct {
	ID          int          `db:"id" json:"id"`
	WorkspaceID int          `db:"workspace_id" json:"workspace_id"`
	SessionName string       `db:"session_name" json:"session_name"`
	OwnerID     int          `db:"owner_id" json:"owner_id"`
	SharedWith  []int        `db:"-" json:"shared_with"`
	IsActive    bool         `db:"is_active" json:"is_active"`
	CreatedAt   time.Time    `db:"created_at" json:"created_at"`
	ExpiresAt   sql.NullTime `db:"expires_at" json:"expires_at"`
}

// TmuxShare represents a shared tmux session access
type TmuxShare struct {
	ID         int          `db:"id" json:"id"`
	SessionID  int          `db:"session_id" json:"session_id"`
	UserID     int          `db:"user_id" json:"user_id"`
	Permission string       `db:"permission" json:"permission"` // "read" or "write"
	GrantedAt  time.Time    `db:"granted_at" json:"granted_at"`
	GrantedBy  int          `db:"granted_by" json:"granted_by"`
	ExpiresAt  sql.NullTime `db:"expires_at" json:"expires_at"`
}

// CreateTmuxSession creates a new tmux session for a workspace
func CreateTmuxSession(ctx context.Context, workspaceID, userID int, sessionName string, durationHours int) (*TmuxSession, error) {
	if sessionName == "" {
		sessionName = fmt.Sprintf("ws-%d-%d", workspaceID, time.Now().Unix())
	}

	session := &TmuxSession{
		WorkspaceID: workspaceID,
		SessionName: sessionName,
		OwnerID:     userID,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}

	// Calculate expiration time
	if durationHours > 0 {
		expiresAt := time.Now().Add(time.Duration(durationHours) * time.Hour)
		session.ExpiresAt = sql.NullTime{Time: expiresAt, Valid: true}
	}

	// Create database record
	query := `
		INSERT INTO tmux_sessions (workspace_id, session_name, owner_id, is_active, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := db.DB.ExecContext(ctx, query,
		session.WorkspaceID, session.SessionName, session.OwnerID,
		session.IsActive, session.CreatedAt, session.ExpiresAt.Time)
	if err != nil {
		return nil, fmt.Errorf("failed to create tmux session record: %w", err)
	}

	sessionID, _ := result.LastInsertId()
	session.ID = int(sessionID)

	// Create actual tmux session
	if err := createTmuxCommand(session.SessionName); err != nil {
		return nil, fmt.Errorf("failed to create tmux session: %w", err)
	}

	return session, nil
}

// createTmuxCommand creates an actual tmux session
func createTmuxCommand(sessionName string) error {
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName)
	return cmd.Run()
}

// GetTmuxSession retrieves a tmux session by ID
func GetTmuxSession(ctx context.Context, sessionID int) (*TmuxSession, error) {
	var session TmuxSession
	var expiresAt sql.NullTime

	err := db.DB.QueryRowContext(ctx, `
		SELECT id, workspace_id, session_name, owner_id, is_active, created_at, expires_at
		FROM tmux_sessions WHERE id = ?
	`, sessionID).Scan(
		&session.ID, &session.WorkspaceID, &session.SessionName,
		&session.OwnerID, &session.IsActive, &session.CreatedAt, &expiresAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	session.ExpiresAt = expiresAt
	return &session, nil
}

// GetUserTmuxSessions retrieves all tmux sessions for a user
func GetUserTmuxSessions(ctx context.Context, userID int) ([]TmuxSession, error) {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT ts.id, ts.workspace_id, ts.session_name, ts.owner_id, ts.is_active, ts.created_at, ts.expires_at
		FROM tmux_sessions ts
		JOIN tmux_shares tshare ON ts.id = tshare.session_id
		WHERE tshare.user_id = ? OR ts.owner_id = ?
		ORDER BY ts.created_at DESC
	`, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user sessions: %w", err)
	}
	defer rows.Close()

	var sessions []TmuxSession
	for rows.Next() {
		var session TmuxSession
		var expiresAt sql.NullTime
		if err := rows.Scan(&session.ID, &session.WorkspaceID, &session.SessionName,
			&session.OwnerID, &session.IsActive, &session.CreatedAt, &expiresAt); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		session.ExpiresAt = expiresAt
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// ShareTmuxSession shares a tmux session with another user
func ShareTmuxSession(ctx context.Context, sessionID, userID, grantedBy int, permission string, durationHours int) (*TmuxShare, error) {
	// Verify session exists
	_, err := GetTmuxSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	share := &TmuxShare{
		SessionID:  sessionID,
		UserID:     userID,
		Permission: permission,
		GrantedAt:  time.Now(),
		GrantedBy:  grantedBy,
	}

	if durationHours > 0 {
		expiresAt := time.Now().Add(time.Duration(durationHours) * time.Hour)
		share.ExpiresAt = sql.NullTime{Time: expiresAt, Valid: true}
	}

	query := `
		INSERT INTO tmux_shares (session_id, user_id, permission, granted_at, granted_by, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := db.DB.ExecContext(ctx, query,
		share.SessionID, share.UserID, share.Permission,
		share.GrantedAt, share.GrantedBy, share.ExpiresAt.Time)
	if err != nil {
		return nil, fmt.Errorf("failed to create share record: %w", err)
	}

	shareID, _ := result.LastInsertId()
	share.ID = int(shareID)

	return share, nil
}

// GetTmuxSessionShares retrieves all shares for a session
func GetTmuxSessionShares(ctx context.Context, sessionID int) ([]TmuxShare, error) {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT id, session_id, user_id, permission, granted_at, granted_by, expires_at
		FROM tmux_shares WHERE session_id = ?
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query shares: %w", err)
	}
	defer rows.Close()

	var shares []TmuxShare
	for rows.Next() {
		var share TmuxShare
		var expiresAt sql.NullTime
		if err := rows.Scan(&share.ID, &share.SessionID, &share.UserID,
			&share.Permission, &share.GrantedAt, &share.GrantedBy, &expiresAt); err != nil {
			return nil, fmt.Errorf("failed to scan share: %w", err)
		}
		share.ExpiresAt = expiresAt
		shares = append(shares, share)
	}

	return shares, nil
}

// RevokeTmuxShare revokes access to a shared session
func RevokeTmuxShare(ctx context.Context, shareID int) error {
	_, err := db.DB.ExecContext(ctx, "DELETE FROM tmux_shares WHERE id = ?", shareID)
	return err
}

// DeleteTmuxSession deletes a tmux session
func DeleteTmuxSession(ctx context.Context, sessionID int) error {
	// Get session name for tmux command
	var sessionName string
	err := db.DB.QueryRowContext(ctx, "SELECT session_name FROM tmux_sessions WHERE id = ?", sessionID).Scan(&sessionName)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// Kill actual tmux session (ignore error if session doesn't exist)
	if sessionName != "" {
		_ = exec.Command("tmux", "kill-session", "-t", sessionName).Run()
	}

	// Delete database records
	_, err = db.DB.ExecContext(ctx, "DELETE FROM tmux_shares WHERE session_id = ?", sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete shares: %w", err)
	}

	_, err = db.DB.ExecContext(ctx, "DELETE FROM tmux_sessions WHERE id = ?", sessionID)
	return err
}

// HasTmuxAccess checks if a user has access to a tmux session
func HasTmuxAccess(ctx context.Context, sessionID, userID int) (bool, string, error) {
	var permission sql.NullString
	err := db.DB.QueryRowContext(ctx, `
		SELECT permission FROM tmux_shares
		WHERE session_id = ? AND user_id = ? AND (expires_at IS NULL OR expires_at > ?)
	`, sessionID, userID, time.Now()).Scan(&permission)

	if err == sql.ErrNoRows {
		// Check if user is owner
		var ownerID int
		err := db.DB.QueryRowContext(ctx, "SELECT owner_id FROM tmux_sessions WHERE id = ?", sessionID).Scan(&ownerID)
		if err != nil {
			return false, "", fmt.Errorf("session not found: %w", err)
		}
		return userID == ownerID, "owner", nil
	}
	if err != nil {
		return false, "", fmt.Errorf("failed to check access: %w", err)
	}

	return true, permission.String, nil
}

// GetTmuxSessionURL generates a URL for accessing the tmux session
func GetTmuxSessionURL(sessionID int, workspaceTag string) string {
	return fmt.Sprintf("/workspace/%s/tmux/%d", workspaceTag, sessionID)
}
