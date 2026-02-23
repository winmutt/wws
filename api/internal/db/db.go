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

	statements := []string{
		usersTable,
		organizationsTable,
		workspacesTable,
		workspaceLanguagesTable,
		membersTable,
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
