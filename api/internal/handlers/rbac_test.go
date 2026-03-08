package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"wws/api/internal/crypto"
	"wws/api/internal/db"

	_ "github.com/mattn/go-sqlite3"
)

func createRBACTablesForTest(testDB *sql.DB) {
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

func setupRBACTestDB(t *testing.T) (*sql.DB, int, int, int) {
	testDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	createRBACTablesForTest(testDB)

	db.DB = testDB

	os.Setenv("WWS_ENCRYPTION_KEY", "test-encryption-key-for-rbac-tests")
	crypto.InitEncryption()

	ctx := context.Background()

	user1Result, _ := testDB.ExecContext(ctx,
		"INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"123", "owner", "owner@example.com",
	)
	user1ID, _ := user1Result.LastInsertId()

	user2Result, _ := testDB.ExecContext(ctx,
		"INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"456", "admin", "admin@example.com",
	)
	user2ID, _ := user2Result.LastInsertId()

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

	testDB.ExecContext(ctx,
		`INSERT INTO members (user_id, organization_id, role, invited_by, accepted, created_at)
		 VALUES (?, ?, 'admin', ?, 1, ?)`,
		user2ID, orgID, user1ID, time.Now(),
	)

	sessionToken := "test-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		user1ID, sessionToken, expiresAt,
	)

	return testDB, int(user1ID), int(user2ID), int(orgID)
}

func TestIsValidRole(t *testing.T) {
	validRoles := []string{"admin", "member", "viewer", "owner"}
	for _, role := range validRoles {
		if !isValidRole(role) {
			t.Errorf("Expected '%s' to be a valid role", role)
		}
	}

	invalidRoles := []string{"superadmin", "guest", "moderator", ""}
	for _, role := range invalidRoles {
		if isValidRole(role) {
			t.Errorf("Expected '%s' to be an invalid role", role)
		}
	}
}

func TestAssignRole(t *testing.T) {
	testDB, userID, _, orgID := setupRBACTestDB(t)
	defer testDB.Close()

	ctx := context.Background()

	err := assignRole(ctx, userID, orgID, userID, RoleViewer)
	if err != nil {
		t.Fatalf("Failed to assign role: %v", err)
	}

	member, err := getMemberByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		t.Fatalf("Failed to get member: %v", err)
	}

	if member.Role != RoleViewer {
		t.Errorf("Expected role '%s', got '%s'", RoleViewer, member.Role)
	}
}

func TestAssignRoleInvalidRole(t *testing.T) {
	testDB, userID, _, orgID := setupRBACTestDB(t)
	defer testDB.Close()

	ctx := context.Background()

	err := assignRole(ctx, userID, orgID, userID, "invalid_role")
	if err == nil {
		t.Error("Expected error for invalid role")
	}
}

func TestGetMemberByUserAndOrg(t *testing.T) {
	testDB, userID, _, orgID := setupRBACTestDB(t)
	defer testDB.Close()

	ctx := context.Background()

	member, err := getMemberByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		t.Fatalf("Failed to get member: %v", err)
	}

	if member.UserID != userID {
		t.Errorf("Expected user_id %d, got %d", userID, member.UserID)
	}

	if member.Role != RoleOwner {
		t.Errorf("Expected role '%s', got '%s'", RoleOwner, member.Role)
	}
}

func TestGetMemberByUserAndOrgNotFound(t *testing.T) {
	testDB, _, _, orgID := setupRBACTestDB(t)
	defer testDB.Close()

	ctx := context.Background()

	member, err := getMemberByUserAndOrg(ctx, 9999, orgID)
	if err != nil {
		t.Fatalf("Failed to get member: %v", err)
	}

	if member != nil {
		t.Error("Expected nil member for nonexistent user")
	}
}

func TestListMembersByOrganization(t *testing.T) {
	testDB, _, _, orgID := setupRBACTestDB(t)
	defer testDB.Close()

	ctx := context.Background()

	members, err := listMembersByOrganization(ctx, orgID)
	if err != nil {
		t.Fatalf("Failed to list members: %v", err)
	}

	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}
}

func TestCanModifyRole(t *testing.T) {
	tests := []struct {
		name           string
		requesterRole  string
		targetRole     string
		requesterID    int
		targetUserID   int
		expectedResult bool
	}{
		{"Owner can modify self", RoleOwner, RoleViewer, 1, 1, true},
		{"Owner can modify admin", RoleOwner, RoleAdmin, 1, 2, true},
		{"Owner can modify member", RoleOwner, RoleMember, 1, 3, true},
		{"Admin can modify member", RoleAdmin, RoleMember, 2, 3, true},
		{"Admin can modify viewer", RoleAdmin, RoleViewer, 2, 3, true},
		{"Admin cannot modify admin", RoleAdmin, RoleAdmin, 2, 3, false},
		{"Admin cannot modify owner", RoleAdmin, RoleOwner, 2, 1, false},
		{"Member cannot modify member", RoleMember, RoleMember, 3, 4, false},
		{"Member cannot modify admin", RoleMember, RoleAdmin, 3, 2, false},
		{"Viewer cannot modify member", RoleViewer, RoleMember, 4, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canModifyRole(tt.requesterID, tt.targetUserID, 1, tt.requesterRole, tt.targetRole)
			if result != tt.expectedResult {
				t.Errorf("Expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestAssignRoleHandler(t *testing.T) {
	testDB, _, adminID, orgID := setupRBACTestDB(t)
	defer testDB.Close()

	SetOAuthDB(testDB)

	ctx := context.Background()

	memberResult, _ := testDB.ExecContext(ctx,
		"INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"101", "newmember", "newmember@example.com",
	)
	memberID, _ := memberResult.LastInsertId()

	testDB.ExecContext(ctx,
		`INSERT INTO members (user_id, organization_id, role, invited_by, accepted, created_at)
		 VALUES (?, ?, 'member', ?, 1, ?)`,
		memberID, orgID, 1, time.Now(),
	)

	adminSessionToken := "admin-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		adminID, adminSessionToken, expiresAt,
	)

	reqBody := map[string]interface{}{
		"user_id":         memberID,
		"organization_id": orgID,
		"role":            RoleAdmin,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/roles", bytes.NewBuffer(jsonBody))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: adminSessionToken})
	w := httptest.NewRecorder()

	err := AssignRoleHandler(w, req)
	if err != nil {
		t.Fatalf("AssignRoleHandler returned error: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["role"] != RoleAdmin {
		t.Errorf("Expected role '%s', got '%s'", RoleAdmin, response["role"])
	}
}

func TestAssignRoleHandlerMissingFields(t *testing.T) {
	testDB, _, adminID, orgID := setupRBACTestDB(t)
	defer testDB.Close()

	SetOAuthDB(testDB)

	adminSessionToken := "admin-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	ctx := context.Background()
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		adminID, adminSessionToken, expiresAt,
	)

	reqBody := map[string]interface{}{
		"user_id":         1,
		"organization_id": orgID,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/roles", bytes.NewBuffer(jsonBody))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: adminSessionToken})
	w := httptest.NewRecorder()

	err := AssignRoleHandler(w, req)
	if err == nil {
		t.Error("Expected error for missing role")
	}

	if err.Error() != "role is required" {
		t.Errorf("Expected 'role is required' error, got '%s'", err.Error())
	}
}

func TestAssignRoleHandlerInvalidRole(t *testing.T) {
	testDB, _, adminID, orgID := setupRBACTestDB(t)
	defer testDB.Close()

	SetOAuthDB(testDB)

	adminSessionToken := "admin-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	ctx := context.Background()
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		adminID, adminSessionToken, expiresAt,
	)

	reqBody := map[string]interface{}{
		"user_id":         1,
		"organization_id": orgID,
		"role":            "superadmin",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/roles", bytes.NewBuffer(jsonBody))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: adminSessionToken})
	w := httptest.NewRecorder()

	err := AssignRoleHandler(w, req)
	if err == nil {
		t.Error("Expected error for invalid role")
	}
}

func TestAssignRoleHandlerNotMember(t *testing.T) {
	testDB, _, _, orgID := setupRBACTestDB(t)
	defer testDB.Close()

	SetOAuthDB(testDB)

	ctx := context.Background()

	nonMemberResult, _ := testDB.ExecContext(ctx,
		"INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"202", "nonmember", "nonmember@example.com",
	)
	nonMemberID, _ := nonMemberResult.LastInsertId()

	nonMemberSessionToken := "nonmember-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		nonMemberID, nonMemberSessionToken, expiresAt,
	)

	reqBody := map[string]interface{}{
		"user_id":         1,
		"organization_id": orgID,
		"role":            RoleAdmin,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/roles", bytes.NewBuffer(jsonBody))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: nonMemberSessionToken})
	w := httptest.NewRecorder()

	err := AssignRoleHandler(w, req)
	if err == nil {
		t.Error("Expected error for non-member")
	}

	if err.Error() != "you are not a member of this organization" {
		t.Errorf("Expected 'you are not a member of this organization' error, got '%s'", err.Error())
	}
}

func TestAssignRoleHandlerCannotModifyHigherPrivilege(t *testing.T) {
	testDB, _, adminID, orgID := setupRBACTestDB(t)
	defer testDB.Close()

	SetOAuthDB(testDB)

	ctx := context.Background()

	adminSessionToken := "admin-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		adminID, adminSessionToken, expiresAt,
	)

	reqBody := map[string]interface{}{
		"user_id":         1,
		"organization_id": orgID,
		"role":            RoleOwner,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/roles", bytes.NewBuffer(jsonBody))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: adminSessionToken})
	w := httptest.NewRecorder()

	err := AssignRoleHandler(w, req)
	if err == nil {
		t.Error("Expected error for modifying higher privilege user")
	}

	if err.Error() != "you cannot modify the role of users with equal or higher privileges" {
		t.Errorf("Expected privilege error, got '%s'", err.Error())
	}
}

func TestListMembersHandler(t *testing.T) {
	testDB, ownerID, _, orgID := setupRBACTestDB(t)
	defer testDB.Close()

	SetOAuthDB(testDB)

	ownerSessionToken := "owner-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	ctx := context.Background()
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		ownerID, ownerSessionToken, expiresAt,
	)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organizations/members?organization_id=%d", orgID), nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: ownerSessionToken})
	w := httptest.NewRecorder()

	err := ListMembersHandler(w, req)
	if err != nil {
		t.Fatalf("ListMembersHandler returned error: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	members := response["members"].([]interface{})
	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}
}

func TestListMembersHandlerMissingOrgID(t *testing.T) {
	testDB, ownerID, _, _ := setupRBACTestDB(t)
	defer testDB.Close()

	SetOAuthDB(testDB)

	ownerSessionToken := "owner-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	ctx := context.Background()
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		ownerID, ownerSessionToken, expiresAt,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/members", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: ownerSessionToken})
	w := httptest.NewRecorder()

	err := ListMembersHandler(w, req)
	if err == nil {
		t.Error("Expected error for missing organization_id")
	}
}

func TestGetMemberHandler(t *testing.T) {
	testDB, ownerID, _, orgID := setupRBACTestDB(t)
	defer testDB.Close()

	SetOAuthDB(testDB)

	ownerSessionToken := "owner-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	ctx := context.Background()
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		ownerID, ownerSessionToken, expiresAt,
	)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organizations/members?user_id=1&organization_id=%d", orgID), nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: ownerSessionToken})
	w := httptest.NewRecorder()

	err := GetMemberHandler(w, req)
	if err != nil {
		t.Fatalf("GetMemberHandler returned error: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRemoveMemberHandler(t *testing.T) {
	testDB, ownerID, _, orgID := setupRBACTestDB(t)
	defer testDB.Close()

	SetOAuthDB(testDB)

	ctx := context.Background()

	memberResult, _ := testDB.ExecContext(ctx,
		"INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"303", "tomove", "toremove@example.com",
	)
	memberID, _ := memberResult.LastInsertId()

	testDB.ExecContext(ctx,
		`INSERT INTO members (user_id, organization_id, role, invited_by, accepted, created_at)
		 VALUES (?, ?, 'member', ?, 1, ?)`,
		memberID, orgID, ownerID, time.Now(),
	)

	ownerSessionToken := "owner-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		ownerID, ownerSessionToken, expiresAt,
	)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/organizations/members?user_id=%d&organization_id=%d", memberID, orgID), nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: ownerSessionToken})
	w := httptest.NewRecorder()

	err := RemoveMemberHandler(w, req)
	if err != nil {
		t.Fatalf("RemoveMemberHandler returned error: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] != "Member removed successfully" {
		t.Errorf("Expected success message, got '%s'", response["message"])
	}

	member, _ := getMemberByUserAndOrg(ctx, int(memberID), orgID)
	if member != nil {
		t.Error("Expected member to be removed")
	}
}

func TestRemoveMemberHandlerCannotRemoveSelf(t *testing.T) {
	testDB, ownerID, _, orgID := setupRBACTestDB(t)
	defer testDB.Close()

	SetOAuthDB(testDB)

	ownerSessionToken := "owner-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	ctx := context.Background()
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		ownerID, ownerSessionToken, expiresAt,
	)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/organizations/members?user_id=1&organization_id=%d", orgID), nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: ownerSessionToken})
	w := httptest.NewRecorder()

	err := RemoveMemberHandler(w, req)
	if err == nil {
		t.Error("Expected error for removing self")
	}

	if err.Error() != "you cannot remove yourself from the organization" {
		t.Errorf("Expected self-removal error, got '%s'", err.Error())
	}
}

func TestRemoveMemberHandlerCannotRemoveOwner(t *testing.T) {
	testDB, _, adminID, orgID := setupRBACTestDB(t)
	defer testDB.Close()

	SetOAuthDB(testDB)

	adminSessionToken := "admin-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	ctx := context.Background()
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		adminID, adminSessionToken, expiresAt,
	)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/organizations/members?user_id=1&organization_id=%d", orgID), nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: adminSessionToken})
	w := httptest.NewRecorder()

	err := RemoveMemberHandler(w, req)
	if err == nil {
		t.Error("Expected error for removing owner")
	}

	if err.Error() != "you cannot remove the organization owner" {
		t.Errorf("Expected owner-removal error, got '%s'", err.Error())
	}
}

func TestRemoveMemberHandlerAdminCannotRemoveAdmin(t *testing.T) {
	testDB, _, adminID, orgID := setupRBACTestDB(t)
	defer testDB.Close()

	SetOAuthDB(testDB)

	ctx := context.Background()

	anotherAdminResult, _ := testDB.ExecContext(ctx,
		"INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"404", "anotheradmin", "anotheradmin@example.com",
	)
	anotherAdminID, _ := anotherAdminResult.LastInsertId()

	testDB.ExecContext(ctx,
		`INSERT INTO members (user_id, organization_id, role, invited_by, accepted, created_at)
		 VALUES (?, ?, 'admin', ?, 1, ?)`,
		anotherAdminID, orgID, 1, time.Now(),
	)

	adminSessionToken := "admin-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		adminID, adminSessionToken, expiresAt,
	)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/organizations/members?user_id=%d&organization_id=%d", anotherAdminID, orgID), nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: adminSessionToken})
	w := httptest.NewRecorder()

	err := RemoveMemberHandler(w, req)
	if err == nil {
		t.Error("Expected error for admin removing admin")
	}

	if err.Error() != "only owners can remove admins" {
		t.Errorf("Expected admin-removal error, got '%s'", err.Error())
	}
}
