package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"wws/api/internal/crypto"
	"wws/api/internal/db"
	"wws/api/internal/handlers"

	_ "github.com/mattn/go-sqlite3"
)

func createAuthMiddlewareTables(testDB *sql.DB) {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			github_id TEXT UNIQUE NOT NULL,
			username TEXT NOT NULL,
			email TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS organizations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			owner_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (owner_id) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			organization_id INTEGER NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			invited_by INTEGER,
			accepted INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (organization_id) REFERENCES organizations(id),
			FOREIGN KEY (invited_by) REFERENCES users(id),
			UNIQUE(user_id, organization_id)
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token TEXT UNIQUE NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
	}

	for _, stmt := range statements {
		_, err := testDB.Exec(stmt)
		if err != nil {
			panic(err)
		}
	}
}

func setupMiddlewareTestDB(t *testing.T) (*sql.DB, int, int, int) {
	testDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	createAuthMiddlewareTables(testDB)

	db.DB = testDB

	os.Setenv("WWS_ENCRYPTION_KEY", "test-encryption-key-for-middleware-tests")
	crypto.InitEncryption()

	handlers.SetOAuthDB(testDB)

	ctx := context.Background()

	user1Result, _ := testDB.ExecContext(ctx,
		"INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"123", "owner", "owner@example.com",
	)
	user1ID, _ := user1Result.LastInsertId()

	testDB.ExecContext(ctx,
		"INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"456", "admin", "admin@example.com",
	)

	testDB.ExecContext(ctx,
		"INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"789", "member", "member@example.com",
	)

	orgResult, _ := testDB.ExecContext(ctx,
		"INSERT INTO organizations (name, owner_id) VALUES (?, ?)",
		"Test Org", user1ID,
	)
	orgID, _ := orgResult.LastInsertId()

	testDB.ExecContext(ctx,
		`INSERT INTO members (user_id, organization_id, role, invited_by, accepted, created_at)
		 VALUES (?, ?, 'owner', ?, 1, ?)`,
		user1ID, orgID, user1ID, time.Now(),
	)

	return testDB, int(user1ID), 2, int(orgID)
}

func TestAuthMiddlewareValidSession(t *testing.T) {
	testDB, userID, _, _ := setupMiddlewareTestDB(t)
	defer testDB.Close()

	ctx := context.Background()

	sessionToken := "test-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		userID, sessionToken, expiresAt,
	)

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAuthMiddlewareMissingCookie(t *testing.T) {
	testDB, _, _, _ := setupMiddlewareTestDB(t)
	defer testDB.Close()

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuthMiddlewareEmptyCookie(t *testing.T) {
	testDB, _, _, _ := setupMiddlewareTestDB(t)
	defer testDB.Close()

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: ""})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuthMiddlewareInvalidSession(t *testing.T) {
	testDB, _, _, _ := setupMiddlewareTestDB(t)
	defer testDB.Close()

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "invalid-token"})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestRequireRoleAdminNotAllowed(t *testing.T) {
	testDB, userID, orgID, _ := setupMiddlewareTestDB(t)
	defer testDB.Close()

	ctx := context.Background()

	sessionToken := "test-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		userID, sessionToken, expiresAt,
	)

	handler := AuthMiddleware(ExtractOrgIDFromQuery(RequireRole(RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))))

	req := httptest.NewRequest(http.MethodGet, "/test?organization_id="+fmt.Sprintf("%d", orgID), nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestRequireRoleMissingOrgID(t *testing.T) {
	testDB, userID, _, _ := setupMiddlewareTestDB(t)
	defer testDB.Close()

	ctx := context.Background()

	sessionToken := "test-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		userID, sessionToken, expiresAt,
	)

	handler := AuthMiddleware(RequireRole(RoleOwner)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestGetUserID(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserIDKey, 123)
	r := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)

	userID, ok := GetUserID(r)
	if !ok {
		t.Error("Expected to get user ID")
	}
	if userID != 123 {
		t.Errorf("Expected user ID 123, got %d", userID)
	}
}

func TestGetUserIDNotSet(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	userID, ok := GetUserID(r)
	if ok {
		t.Error("Expected user ID to not be set")
	}
	if userID != 0 {
		t.Errorf("Expected user ID 0, got %d", userID)
	}
}

func TestGetUserRole(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserRoleKey, RoleAdmin)
	r := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)

	role, ok := GetUserRole(r)
	if !ok {
		t.Error("Expected to get user role")
	}
	if role != RoleAdmin {
		t.Errorf("Expected role 'admin', got '%s'", role)
	}
}

func TestGetUserRoleNotSet(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	role, ok := GetUserRole(r)
	if ok {
		t.Error("Expected role to not be set")
	}
	if role != "" {
		t.Errorf("Expected empty role, got '%s'", role)
	}
}

func TestGetOrgID(t *testing.T) {
	ctx := context.WithValue(context.Background(), OrgIDKey, 456)
	r := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)

	orgID, ok := GetOrgID(r)
	if !ok {
		t.Error("Expected to get org ID")
	}
	if orgID != 456 {
		t.Errorf("Expected org ID 456, got %d", orgID)
	}
}

func TestGetOrgIDNotSet(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	orgID, ok := GetOrgID(r)
	if ok {
		t.Error("Expected org ID to not be set")
	}
	if orgID != 0 {
		t.Errorf("Expected org ID 0, got %d", orgID)
	}
}

func TestExtractOrgIDFromQuery(t *testing.T) {
	handler := ExtractOrgIDFromQuery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := GetOrgID(r)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if orgID != 123 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test?organization_id=123", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestExtractOrgIDFromQueryMissing(t *testing.T) {
	handler := ExtractOrgIDFromQuery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestExtractOrgIDFromQueryInvalid(t *testing.T) {
	handler := ExtractOrgIDFromQuery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test?organization_id=invalid", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
