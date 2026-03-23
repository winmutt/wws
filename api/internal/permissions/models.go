package permissions

import "time"

// TeamPermission represents a team-level permission
type TeamPermission string

const (
	// Workspace permissions
	PermWorkspaceCreate TeamPermission = "workspace:create"
	PermWorkspaceRead   TeamPermission = "workspace:read"
	PermWorkspaceUpdate TeamPermission = "workspace:update"
	PermWorkspaceDelete TeamPermission = "workspace:delete"
	PermWorkspaceStart  TeamPermission = "workspace:start"
	PermWorkspaceStop   TeamPermission = "workspace:stop"

	// Organization permissions
	PermOrgManage      TeamPermission = "org:manage"
	PermOrgInvite      TeamPermission = "org:invite"
	PermOrgRemove      TeamPermission = "org:remove"
	PermOrgViewBilling TeamPermission = "org:view_billing"

	// Admin permissions
	PermAdminAll TeamPermission = "admin:all"
)

// TeamRole represents a team role with associated permissions
type TeamRole struct {
	ID          int              `db:"id" json:"id"`
	Name        string           `db:"name" json:"name"`
	Description string           `db:"description" json:"description"`
	Permissions []TeamPermission `db:"-" json:"permissions"`
	IsDefault   bool             `db:"is_default" json:"is_default"`
	CreatedAt   time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time        `db:"updated_at" json:"updated_at"`
}

// TeamMemberRole represents a user's role in a team
type TeamMemberRole struct {
	ID                int       `db:"id" json:"id"`
	TeamID            int       `db:"team_id" json:"team_id"`
	UserID            int       `db:"user_id" json:"user_id"`
	RoleID            int       `db:"role_id" json:"role_id"`
	CustomPermissions []string  `db:"-" json:"custom_permissions"`
	AssignedAt        time.Time `db:"assigned_at" json:"assigned_at"`
	AssignedBy        int       `db:"assigned_by" json:"assigned_by"`
}

// Team represents a team within an organization
type Team struct {
	ID             int       `db:"id" json:"id"`
	OrganizationID int       `db:"organization_id" json:"organization_id"`
	Name           string    `db:"name" json:"name"`
	Description    string    `db:"description" json:"description"`
	CreatedBy      int       `db:"created_by" json:"created_by"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// TeamMember represents a member of a team
type TeamMember struct {
	ID       int       `db:"id" json:"id"`
	TeamID   int       `db:"team_id" json:"team_id"`
	UserID   int       `db:"user_id" json:"user_id"`
	Username string    `db:"username" json:"username"`
	Email    string    `db:"email" json:"email"`
	RoleName string    `db:"role_name" json:"role_name"`
	JoinedAt time.Time `db:"joined_at" json:"joined_at"`
	Status   string    `db:"status" json:"status"` // "active", "invited", "suspended"
}

// TeamWorkspaceAccess represents workspace access for a team
type TeamWorkspaceAccess struct {
	ID          int       `db:"id" json:"id"`
	TeamID      int       `db:"team_id" json:"team_id"`
	WorkspaceID int       `db:"workspace_id" json:"workspace_id"`
	AccessLevel string    `db:"access_level" json:"access_level"` // "read", "write", "admin"
	GrantedAt   time.Time `db:"granted_at" json:"granted_at"`
	GrantedBy   int       `db:"granted_by" json:"granted_by"`
}

// PermissionCheckResult represents the result of a permission check
type PermissionCheckResult struct {
	Allowed bool     `json:"allowed"`
	Reason  string   `json:"reason"`
	Missing []string `json:"missing,omitempty"`
}
