package idlemgmt

import "time"

// IdleConfig represents idle timeout configuration for an organization
type IdleConfig struct {
	ID                    int       `db:"id" json:"id"`
	OrganizationID        int       `db:"organization_id" json:"organization_id"`
	IdleTimeoutHours      int       `db:"idle_timeout_hours" json:"idle_timeout_hours"`           // Default: 6 hours
	WarningThresholdHours int       `db:"warning_threshold_hours" json:"warning_threshold_hours"` // Warn before shutdown
	AutoShutdownEnabled   bool      `db:"auto_shutdown_enabled" json:"auto_shutdown_enabled"`
	ShutdownGracePeriod   int       `db:"shutdown_grace_period" json:"shutdown_grace_period"` // Minutes to graceful shutdown
	ExemptUsers           []int     `db:"-" json:"exempt_users"`                              // User IDs exempt from idle shutdown
	ExemptWorkspaces      []int     `db:"-" json:"exempt_workspaces"`                         // Workspace IDs exempt from idle shutdown
	UpdatedAt             time.Time `db:"updated_at" json:"updated_at"`
}

// WorkspaceIdleStatus represents the idle status of a workspace
type WorkspaceIdleStatus struct {
	WorkspaceID    int           `json:"workspace_id"`
	WorkspaceTag   string        `json:"workspace_tag"`
	LastActiveAt   time.Time     `json:"last_active_at"`
	IdleDuration   time.Duration `json:"idle_duration"`
	IsIdle         bool          `json:"is_idle"`
	WillShutdownAt *time.Time    `json:"will_shutdown_at,omitempty"`
	ShutdownReason *string       `json:"shutdown_reason,omitempty"`
	Exempt         bool          `json:"exempt"`
}

// IdleShutdownEvent represents an event when a workspace is shut down due to idle
type IdleShutdownEvent struct {
	ID             int       `db:"id" json:"id"`
	WorkspaceID    int       `db:"workspace_id" json:"workspace_id"`
	OrganizationID int       `db:"organization_id" json:"organization_id"`
	IdleDuration   int       `db:"idle_duration_minutes" json:"idle_duration_minutes"`
	ShutdownAt     time.Time `db:"shutdown_at" json:"shutdown_at"`
	TriggeredBy    string    `db:"triggered_by" json:"triggered_by"` // "auto", "manual", "scheduled"
	Reason         string    `db:"reason" json:"reason"`
}

// IdleWarning represents a warning about impending idle shutdown
type IdleWarning struct {
	WorkspaceID      int       `json:"workspace_id"`
	WorkspaceTag     string    `json:"workspace_tag"`
	LastActiveAt     time.Time `json:"last_active_at"`
	WarningSentAt    time.Time `json:"warning_sent_at"`
	ShutdownAt       time.Time `json:"shutdown_at"`
	MinutesRemaining int       `json:"minutes_remaining"`
}
