package models

import "time"

type Organization struct {
	ID        int       `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	OwnerID   int       `db:"owner_id" json:"owner_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type User struct {
	ID        int       `db:"id" json:"id"`
	GithubID  string    `db:"github_id" json:"github_id"`
	Username  string    `db:"username" json:"username"`
	Email     string    `db:"email" json:"email"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type Workspace struct {
	ID             int        `db:"id" json:"id"`
	Tag            string     `db:"tag" json:"tag"`
	Name           string     `db:"name" json:"name"`
	OrganizationID int        `db:"organization_id" json:"organization_id"`
	OwnerID        int        `db:"owner_id" json:"owner_id"`
	Provider       string     `db:"provider" json:"provider"`
	Status         string     `db:"status" json:"status"`
	Config         string     `db:"config" json:"config"`
	Region         string     `db:"region" json:"region"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt      *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

type WorkspaceLanguage struct {
	ID            int    `db:"id" json:"id"`
	WorkspaceID   int    `db:"workspace_id" json:"workspace_id"`
	Language      string `db:"language" json:"language"`
	Version       string `db:"version" json:"version"`
	InstallScript string `db:"install_script" json:"install_script"`
}

type OAuthToken struct {
	ID                    int       `db:"id" json:"id"`
	UserID                int       `db:"user_id" json:"user_id"`
	AccessToken           string    `db:"access_token" json:"-"`
	EncryptedAccessToken  string    `db:"encrypted_access_token" json:"-"`
	RefreshToken          string    `db:"refresh_token" json:"-"`
	EncryptedRefreshToken string    `db:"encrypted_refresh_token" json:"-"`
	Expiry                time.Time `db:"expiry" json:"expiry"`
	CreatedAt             time.Time `db:"created_at" json:"created_at"`
	UpdatedAt             time.Time `db:"updated_at" json:"updated_at"`
}

type Session struct {
	ID        int       `db:"id" json:"id"`
	UserID    int       `db:"user_id" json:"user_id"`
	Token     string    `db:"token" json:"token"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type OAuthState struct {
	ID        int       `db:"id" json:"id"`
	State     string    `db:"state" json:"state"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// AuditLog represents an audit trail entry for tracking all operations
type AuditLog struct {
	ID             int       `db:"id" json:"id"`
	UserID         int       `db:"user_id" json:"user_id"`
	Username       string    `db:"username" json:"username"`
	OrganizationID *int      `db:"organization_id" json:"organization_id,omitempty"`
	Action         string    `db:"action" json:"action"`
	ResourceType   string    `db:"resource_type" json:"resource_type"`
	ResourceID     *int      `db:"resource_id" json:"resource_id,omitempty"`
	IPAddress      string    `db:"ip_address" json:"ip_address"`
	UserAgent      string    `db:"user_agent" json:"user_agent"`
	Details        string    `db:"details" json:"details"`
	Success        bool      `db:"success" json:"success"`
	ErrorMessage   *string   `db:"error_message" json:"error_message,omitempty"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

// AuditLogFilter for querying audit logs
type AuditLogFilter struct {
	UserID         *int
	OrganizationID *int
	Action         string
	ResourceType   string
	StartDate      time.Time
	EndDate        time.Time
	Success        *bool
	Limit          int
	Offset         int
}

// ResourceQuota represents quota limits for an organization
type ResourceQuota struct {
	ID                  int       `db:"id" json:"id"`
	OrganizationID      int       `db:"organization_id" json:"organization_id"`
	MaxWorkspaces       int       `db:"max_workspaces" json:"max_workspaces"`
	MaxUsers            int       `db:"max_users" json:"max_users"`
	MaxStorageGB        int       `db:"max_storage_gb" json:"max_storage_gb"`
	MaxComputeHours     int       `db:"max_compute_hours" json:"max_compute_hours"`
	MaxNetworkBandwidth int       `db:"max_network_bandwidth" json:"max_network_bandwidth"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time `db:"updated_at" json:"updated_at"`
}

// QuotaUsage represents current usage of resources
type QuotaUsage struct {
	ID                   int       `db:"id" json:"id"`
	OrganizationID       int       `db:"organization_id" json:"organization_id"`
	WorkspacesCount      int       `db:"workspaces_count" json:"workspaces_count"`
	UsersCount           int       `db:"users_count" json:"users_count"`
	StorageUsedGB        int       `db:"storage_used_gb" json:"storage_used_gb"`
	ComputeHoursUsed     int       `db:"compute_hours_used" json:"compute_hours_used"`
	NetworkBandwidthUsed int       `db:"network_bandwidth_used" json:"network_bandwidth_used"`
	UpdatedAt            time.Time `db:"updated_at" json:"updated_at"`
}

// QuotaCheckResult represents the result of a quota check
type QuotaCheckResult struct {
	Allowed  bool   `json:"allowed"`
	Resource string `json:"resource"`
	Current  int    `json:"current"`
	Limit    int    `json:"limit"`
	Message  string `json:"message"`
}

// QuotaUpdateRequest for updating quotas
type QuotaUpdateRequest struct {
	MaxWorkspaces       int `json:"max_workspaces"`
	MaxUsers            int `json:"max_users"`
	MaxStorageGB        int `json:"max_storage_gb"`
	MaxComputeHours     int `json:"max_compute_hours"`
	MaxNetworkBandwidth int `json:"max_network_bandwidth"`
}
