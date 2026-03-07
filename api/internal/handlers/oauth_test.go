package handlers

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func createTablesForTest(db *sql.DB) {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			github_id TEXT UNIQUE NOT NULL,
			username TEXT NOT NULL,
			email TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS oauth_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL UNIQUE,
			access_token TEXT NOT NULL,
			refresh_token TEXT,
			expiry DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token TEXT UNIQUE NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS oauth_states (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			state TEXT UNIQUE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, stmt := range statements {
		_, err := db.Exec(stmt)
		if err != nil {
			panic(err)
		}
	}
}

func TestGenerateStateToken(t *testing.T) {
	state, err := generateStateToken()
	if err != nil {
		t.Fatalf("Failed to generate state token: %v", err)
	}

	if len(state) == 0 {
		t.Error("Generated state token is empty")
	}
}

func TestStoreAndValidateOAuthState(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	defer db.Close()

	createTablesForTest(db)

	InitOAuthStateStore()
	SetOAuthDB(db)

	state, err := generateStateToken()
	if err != nil {
		t.Fatalf("Failed to generate state token: %v", err)
	}

	err = StoreOAuthState(state)
	if err != nil {
		t.Fatalf("Failed to store OAuth state: %v", err)
	}

	valid := ValidateOAuthState(state)
	if !valid {
		t.Error("Failed to validate stored OAuth state")
	}

	validAgain := ValidateOAuthState(state)
	if validAgain {
		t.Error("OAuth state should be invalidated after first use")
	}
}

func TestCleanupExpiredStates(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	defer db.Close()

	createTablesForTest(db)

	InitOAuthStateStore()
	SetOAuthDB(db)

	state1, _ := generateStateToken()
	state2, _ := generateStateToken()

	StoreOAuthState(state1)
	StoreOAuthState(state2)

	if !ValidateOAuthState(state1) {
		t.Error("State1 should be valid")
	}

	cleanupExpiredStates()

	valid := ValidateOAuthState(state2)
	if !valid {
		t.Error("State2 should still be valid after cleanup")
	}
}

func TestCreateSession(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	defer db.Close()

	createTablesForTest(db)
	SetOAuthDB(db)

	user := map[string]interface{}{
		"login": "testuser",
		"id":    float64(12345),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sessionToken, err := createSession(ctx, 1, user)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if len(sessionToken) == 0 {
		t.Error("Session token should not be empty")
	}
}

func TestFindOrCreateUser(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	defer db.Close()

	createTablesForTest(db)
	SetOAuthDB(db)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user := map[string]interface{}{
		"login": "newuser",
		"id":    float64(99999),
		"email": "newuser@example.com",
	}

	userID, err := findOrCreateUser(ctx, user)
	if err != nil {
		t.Fatalf("Failed to find or create user: %v", err)
	}

	if userID <= 0 {
		t.Error("User ID should be positive")
	}

	user2 := map[string]interface{}{
		"login": "newuser",
		"id":    float64(99999),
	}

	userID2, err := findOrCreateUser(ctx, user2)
	if err != nil {
		t.Fatalf("Failed to find existing user: %v", err)
	}

	if userID != userID2 {
		t.Error("Same user should return same ID")
	}
}
