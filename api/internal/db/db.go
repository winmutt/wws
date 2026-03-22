package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init(dbPath string) {
	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}

	createTables()
}

func createTables() {
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		github_id TEXT UNIQUE NOT NULL,
		username TEXT NOT NULL,
		email TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	organizationsTable := `
	CREATE TABLE IF NOT EXISTS organizations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		owner_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (owner_id) REFERENCES users(id)
	);`

	workspacesTable := `
	CREATE TABLE IF NOT EXISTS workspaces (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tag TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		organization_id INTEGER NOT NULL,
		owner_id INTEGER NOT NULL,
		provider TEXT NOT NULL,
		status TEXT NOT NULL,
		config TEXT,
		region TEXT,
		template_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME,
		FOREIGN KEY (organization_id) REFERENCES organizations(id),
		FOREIGN KEY (owner_id) REFERENCES users(id),
		FOREIGN KEY (template_id) REFERENCES workspace_templates(id)
	);`

	workspaceLanguagesTable := `
	CREATE TABLE IF NOT EXISTS workspace_languages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workspace_id INTEGER NOT NULL,
		language TEXT NOT NULL,
		version TEXT,
		install_script TEXT,
		FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
	);`

	membersTable := `
	CREATE TABLE IF NOT EXISTS members (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		organization_id INTEGER NOT NULL,
		role TEXT NOT NULL DEFAULT 'member',
		invited_by INTEGER,
		accepted INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id),
		FOREIGN KEY (organization_id) REFERENCES organizations(id),
		FOREIGN KEY (invited_by) REFERENCES users(id)
	);`

	oauthTokensTable := `
	CREATE TABLE IF NOT EXISTS oauth_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL UNIQUE,
		access_token TEXT,
		encrypted_access_token TEXT,
		refresh_token TEXT,
		encrypted_refresh_token TEXT,
		expiry DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	sessionsTable := `
	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		token TEXT UNIQUE NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	oauthStatesTable := `
	CREATE TABLE IF NOT EXISTS oauth_states (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		state TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	invitationsTable := `
	CREATE TABLE IF NOT EXISTS invitations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		email TEXT NOT NULL,
		token TEXT UNIQUE NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		created_by INTEGER NOT NULL,
		accepted_by INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		FOREIGN KEY (organization_id) REFERENCES organizations(id),
		FOREIGN KEY (created_by) REFERENCES users(id),
		FOREIGN KEY (accepted_by) REFERENCES users(id)
	);`

	auditLogsTable := `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		username TEXT NOT NULL,
		organization_id INTEGER,
		action TEXT NOT NULL,
		resource_type TEXT NOT NULL,
		resource_id INTEGER,
		ip_address TEXT NOT NULL,
		user_agent TEXT,
		details TEXT,
		success INTEGER NOT NULL DEFAULT 1,
		error_message TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id),
		FOREIGN KEY (organization_id) REFERENCES organizations(id)
	);`

	resourceQuotasTable := `
	CREATE TABLE IF NOT EXISTS resource_quotas (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL UNIQUE,
		max_workspaces INTEGER NOT NULL DEFAULT 10,
		max_users INTEGER NOT NULL DEFAULT 5,
		max_storage_gb INTEGER NOT NULL DEFAULT 50,
		max_compute_hours INTEGER NOT NULL DEFAULT 100,
		max_network_bandwidth INTEGER NOT NULL DEFAULT 1000,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
	);`

	quotaUsageTable := `
	CREATE TABLE IF NOT EXISTS quota_usage (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL UNIQUE,
		workspaces_count INTEGER NOT NULL DEFAULT 0,
		users_count INTEGER NOT NULL DEFAULT 0,
		storage_used_gb INTEGER NOT NULL DEFAULT 0,
		compute_hours_used INTEGER NOT NULL DEFAULT 0,
		network_bandwidth_used INTEGER NOT NULL DEFAULT 0,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
	);`

	// Create indexes for common queries
	auditLogsIndex1 := `CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);`
	auditLogsIndex2 := `CREATE INDEX IF NOT EXISTS idx_audit_logs_org_id ON audit_logs(organization_id);`
	auditLogsIndex3 := `CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);`
	auditLogsIndex4 := `CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);`
	auditLogsIndex5 := `CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type ON audit_logs(resource_type);`

	// Create indexes for resource quotas
	quotaIndexes1 := `CREATE INDEX IF NOT EXISTS idx_resource_quotas_org_id ON resource_quotas(organization_id);`
	quotaIndexes2 := `CREATE INDEX IF NOT EXISTS idx_quota_usage_org_id ON quota_usage(organization_id);`

	// Workspace members table for shared workspaces
	workspaceMembersTable := `
	CREATE TABLE IF NOT EXISTS workspace_members (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workspace_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		username TEXT NOT NULL,
		email TEXT,
		role TEXT NOT NULL DEFAULT 'viewer',
		permissions TEXT NOT NULL DEFAULT '{}',
		invited_by INTEGER,
		invited_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		joined_at DATETIME,
		status TEXT NOT NULL DEFAULT 'pending',
		FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id),
		FOREIGN KEY (invited_by) REFERENCES users(id)
	);`

	workspaceMembersIndex1 := `CREATE INDEX IF NOT EXISTS idx_workspace_members_workspace_id ON workspace_members(workspace_id);`
	workspaceMembersIndex2 := `CREATE INDEX IF NOT EXISTS idx_workspace_members_user_id ON workspace_members(user_id);`
	workspaceMembersIndex3 := `CREATE INDEX IF NOT EXISTS idx_workspace_members_status ON workspace_members(status);`

	// Workspace template index
	workspaceTemplateIndex := `CREATE INDEX IF NOT EXISTS idx_workspaces_template_id ON workspaces(template_id);`

	// API keys table
	apiKeysTable := `
	CREATE TABLE IF NOT EXISTS api_keys (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		key_hash TEXT NOT NULL UNIQUE,
		key_prefix TEXT NOT NULL,
		permissions TEXT NOT NULL DEFAULT 'read',
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_used_at DATETIME,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	apiKeyIndexes1 := `CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);`
	apiKeyIndexes2 := `CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);`
	apiKeyIndexes3 := `CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at);`

	// Workspace templates tables
	workspaceTemplatesTable := `
	CREATE TABLE IF NOT EXISTS workspace_templates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		organization_id INTEGER,
		provider TEXT NOT NULL DEFAULT 'podman',
		bootstrap_script TEXT,
		is_public INTEGER DEFAULT 0,
		created_by INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
		FOREIGN KEY (created_by) REFERENCES users(id)
	);`

	templateLanguagesTable := `
	CREATE TABLE IF NOT EXISTS template_languages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		template_id INTEGER NOT NULL,
		language TEXT NOT NULL,
		version TEXT,
		FOREIGN KEY (template_id) REFERENCES workspace_templates(id) ON DELETE CASCADE
	);`

	templateEnvVarsTable := `
	CREATE TABLE IF NOT EXISTS template_env_vars (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		template_id INTEGER NOT NULL,
		key TEXT NOT NULL,
		value TEXT,
		FOREIGN KEY (template_id) REFERENCES workspace_templates(id) ON DELETE CASCADE
	);`

	templateResourcesTable := `
	CREATE TABLE IF NOT EXISTS template_resources (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		template_id INTEGER NOT NULL UNIQUE,
		cpu INTEGER NOT NULL DEFAULT 2,
		memory_gb REAL NOT NULL DEFAULT 4,
		storage_gb INTEGER NOT NULL DEFAULT 20,
		FOREIGN KEY (template_id) REFERENCES workspace_templates(id) ON DELETE CASCADE
	);`

	// Usage analytics tables
	workspaceUsageTable := `
	CREATE TABLE IF NOT EXISTS workspace_usage (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workspace_id INTEGER NOT NULL,
		cpu_usage REAL NOT NULL DEFAULT 0,
		memory_usage REAL NOT NULL DEFAULT 0,
		storage_used_gb REAL NOT NULL DEFAULT 0,
		network_in_mb REAL NOT NULL DEFAULT 0,
		network_out_mb REAL NOT NULL DEFAULT 0,
		uptime_seconds INTEGER NOT NULL DEFAULT 0,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
	);`

	usageAlertsTable := `
	CREATE TABLE IF NOT EXISTS usage_alerts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		workspace_id INTEGER,
		alert_type TEXT NOT NULL,
		severity TEXT NOT NULL,
		message TEXT NOT NULL,
		value REAL NOT NULL,
		threshold REAL NOT NULL,
		resolved INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		resolved_at DATETIME,
		FOREIGN KEY (organization_id) REFERENCES organizations(id),
		FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
	);`

	// Idle management tables
	idleConfigTable := `
	CREATE TABLE IF NOT EXISTS idle_config (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL UNIQUE,
		idle_timeout_hours INTEGER NOT NULL DEFAULT 6,
		warning_threshold_hours INTEGER NOT NULL DEFAULT 5,
		auto_shutdown_enabled INTEGER DEFAULT 1,
		shutdown_grace_period INTEGER NOT NULL DEFAULT 15,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
	);`

	idleExemptUsersTable := `
	CREATE TABLE IF NOT EXISTS idle_exempt_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);`

	idleExemptWorkspacesTable := `
	CREATE TABLE IF NOT EXISTS idle_exempt_workspaces (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		workspace_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
		FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
	);`

	idleShutdownEventsTable := `
	CREATE TABLE IF NOT EXISTS idle_shutdown_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workspace_id INTEGER NOT NULL,
		organization_id INTEGER NOT NULL,
		idle_duration_minutes INTEGER NOT NULL DEFAULT 0,
		shutdown_at DATETIME NOT NULL,
		triggered_by TEXT NOT NULL,
		reason TEXT,
		FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
		FOREIGN KEY (organization_id) REFERENCES organizations(id)
	);`

	// Indexes for templates
	templatesIndex1 := `CREATE INDEX IF NOT EXISTS idx_workspace_templates_org_id ON workspace_templates(organization_id);`
	templatesIndex2 := `CREATE INDEX IF NOT EXISTS idx_workspace_templates_is_public ON workspace_templates(is_public);`
	templatesIndex3 := `CREATE INDEX IF NOT EXISTS idx_template_languages_template_id ON template_languages(template_id);`
	templatesIndex4 := `CREATE INDEX IF NOT EXISTS idx_template_env_vars_template_id ON template_env_vars(template_id);`

	// Indexes for usage analytics
	workspaceUsageIndex1 := `CREATE INDEX IF NOT EXISTS idx_workspace_usage_workspace_id ON workspace_usage(workspace_id);`
	workspaceUsageIndex2 := `CREATE INDEX IF NOT EXISTS idx_workspace_usage_timestamp ON workspace_usage(timestamp);`
	workspaceUsageIndex3 := `CREATE INDEX IF NOT EXISTS idx_workspace_usage_workspace_timestamp ON workspace_usage(workspace_id, timestamp);`

	usageAlertsIndex1 := `CREATE INDEX IF NOT EXISTS idx_usage_alerts_org_id ON usage_alerts(organization_id);`
	usageAlertsIndex2 := `CREATE INDEX IF NOT EXISTS idx_usage_alerts_workspace_id ON usage_alerts(workspace_id);`
	usageAlertsIndex3 := `CREATE INDEX IF NOT EXISTS idx_usage_alerts_resolved ON usage_alerts(resolved);`
	usageAlertsIndex4 := `CREATE INDEX IF NOT EXISTS idx_usage_alerts_created_at ON usage_alerts(created_at);`

	statements := []string{
		usersTable,
		organizationsTable,
		workspacesTable,
		workspaceLanguagesTable,
		membersTable,
		oauthTokensTable,
		sessionsTable,
		oauthStatesTable,
		invitationsTable,
		auditLogsTable,
		resourceQuotasTable,
		quotaUsageTable,
		workspaceMembersTable,
		apiKeysTable,
		workspaceTemplatesTable,
		templateLanguagesTable,
		templateEnvVarsTable,
		templateResourcesTable,
		workspaceUsageTable,
		usageAlertsTable,
		idleConfigTable,
		idleExemptUsersTable,
		idleExemptWorkspacesTable,
		idleShutdownEventsTable,
		auditLogsIndex1,
		auditLogsIndex2,
		auditLogsIndex3,
		auditLogsIndex4,
		auditLogsIndex5,
		quotaIndexes1,
		quotaIndexes2,
		workspaceMembersIndex1,
		workspaceMembersIndex2,
		workspaceMembersIndex3,
		workspaceTemplateIndex,
		apiKeyIndexes1,
		apiKeyIndexes2,
		apiKeyIndexes3,
		templatesIndex1,
		templatesIndex2,
		templatesIndex3,
		templatesIndex4,
		workspaceUsageIndex1,
		workspaceUsageIndex2,
		workspaceUsageIndex3,
		usageAlertsIndex1,
		usageAlertsIndex2,
		usageAlertsIndex3,
		usageAlertsIndex4,
		// Indexes for idle management
		`CREATE INDEX IF NOT EXISTS idx_idle_exempt_users_org ON idle_exempt_users(organization_id)`,
		`CREATE INDEX IF NOT EXISTS idx_idle_exempt_workspaces_org ON idle_exempt_workspaces(organization_id)`,
		`CREATE INDEX IF NOT EXISTS idx_idle_shutdown_events_workspace ON idle_shutdown_events(workspace_id)`,
		`CREATE INDEX IF NOT EXISTS idx_idle_shutdown_events_org ON idle_shutdown_events(organization_id)`,
	}

	for _, stmt := range statements {
		_, err := DB.Exec(stmt)
		if err != nil {
			log.Printf("Error creating table: %v", err)
		}
	}
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
