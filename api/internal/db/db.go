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
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME,
		FOREIGN KEY (organization_id) REFERENCES organizations(id),
		FOREIGN KEY (owner_id) REFERENCES users(id)
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
		auditLogsIndex1,
		auditLogsIndex2,
		auditLogsIndex3,
		auditLogsIndex4,
		auditLogsIndex5,
		quotaIndexes1,
		quotaIndexes2,
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
