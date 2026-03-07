package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"wws/api/internal/crypto"
	"wws/api/internal/db"

	_ "github.com/mattn/go-sqlite3"
)

func createInvitationTablesForTest(testDB *sql.DB) {
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
		`CREATE TABLE IF NOT EXISTS invitations (
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
			FOREIGN KEY (invited_by) REFERENCES users(id)
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

func setupInvitationTestDB(t *testing.T) (*sql.DB, int, int) {
	testDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	createInvitationTablesForTest(testDB)

	db.DB = testDB

	os.Setenv("WWS_ENCRYPTION_KEY", "test-encryption-key-for-invitation-tests")
	crypto.InitEncryption()

	ctx := context.Background()

	user1Result, _ := testDB.ExecContext(ctx,
		"INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"123", "owner", "owner@example.com",
	)
	user1ID, _ := user1Result.LastInsertId()

	_, _ = testDB.ExecContext(ctx,
		"INSERT INTO users (github_id, username, email) VALUES (?, ?, ?)",
		"456", "member", "member@example.com",
	)

	orgResult, _ := testDB.ExecContext(ctx,
		"INSERT INTO organizations (name, owner_id) VALUES (?, ?)",
		"Test Org", user1ID,
	)
	orgID, _ := orgResult.LastInsertId()

	sessionToken := "test-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		user1ID, sessionToken, expiresAt,
	)

	return testDB, int(user1ID), int(orgID)
}

func TestGenerateInvitationToken(t *testing.T) {
	token, err := generateInvitationToken()
	if err != nil {
		t.Fatalf("Failed to generate invitation token: %v", err)
	}

	if len(token) == 0 {
		t.Error("Generated invitation token is empty")
	}

	token2, _ := generateInvitationToken()
	if token == token2 {
		t.Error("Invitation tokens should be unique")
	}
}

func TestCreateInvitation(t *testing.T) {
	testDB, userID, orgID := setupInvitationTestDB(t)
	defer testDB.Close()

	ctx := context.Background()

	invitation, err := createInvitation(ctx, orgID, userID, "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create invitation: %v", err)
	}

	if invitation.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", invitation.Email)
	}

	if invitation.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", invitation.Status)
	}

	if time.Now().After(invitation.ExpiresAt) {
		t.Error("Invitation should not be expired")
	}
}

func TestGetInvitationByToken(t *testing.T) {
	testDB, userID, orgID := setupInvitationTestDB(t)
	defer testDB.Close()

	ctx := context.Background()

	invitation, err := createInvitation(ctx, orgID, userID, "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create invitation: %v", err)
	}

	found, err := getInvitationByToken(ctx, invitation.Token)
	if err != nil {
		t.Fatalf("Failed to get invitation by token: %v", err)
	}

	if found.ID != invitation.ID {
		t.Errorf("Expected invitation ID %d, got %d", invitation.ID, found.ID)
	}
}

func TestGetInvitationByTokenNotFound(t *testing.T) {
	testDB, _, _ := setupInvitationTestDB(t)
	defer testDB.Close()

	ctx := context.Background()

	_, err := getInvitationByToken(ctx, "nonexistent-token")
	if err == nil {
		t.Error("Expected error for nonexistent token")
	}
}

func TestUpdateInvitationStatus(t *testing.T) {
	testDB, userID, orgID := setupInvitationTestDB(t)
	defer testDB.Close()

	ctx := context.Background()

	invitation, _ := createInvitation(ctx, orgID, userID, "test@example.com")

	err := updateInvitationStatus(ctx, invitation.ID, "accepted", &userID)
	if err != nil {
		t.Fatalf("Failed to update invitation status: %v", err)
	}

	found, _ := getInvitationByToken(ctx, invitation.Token)
	if found.Status != "accepted" {
		t.Errorf("Expected status 'accepted', got '%s'", found.Status)
	}
}

func TestGetOrganizationByID(t *testing.T) {
	testDB, userID, orgID := setupInvitationTestDB(t)
	defer testDB.Close()

	ctx := context.Background()

	org, err := getOrganizationByID(ctx, orgID)
	if err != nil {
		t.Fatalf("Failed to get organization: %v", err)
	}

	if org.Name != "Test Org" {
		t.Errorf("Expected name 'Test Org', got '%s'", org.Name)
	}

	if org.OwnerID != userID {
		t.Errorf("Expected owner_id %d, got %d", userID, org.OwnerID)
	}
}

func TestGetOrganizationByIDNotFound(t *testing.T) {
	testDB, _, _ := setupInvitationTestDB(t)
	defer testDB.Close()

	ctx := context.Background()

	_, err := getOrganizationByID(ctx, 9999)
	if err == nil {
		t.Error("Expected error for nonexistent organization")
	}
}

func TestGetUserByEmail(t *testing.T) {
	testDB, _, _ := setupInvitationTestDB(t)
	defer testDB.Close()

	ctx := context.Background()

	user, err := getUserByEmail(ctx, "owner@example.com")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if user.Username != "owner" {
		t.Errorf("Expected username 'owner', got '%s'", user.Username)
	}

	nonexistent, _ := getUserByEmail(ctx, "nonexistent@example.com")
	if nonexistent != nil {
		t.Error("Expected nil for nonexistent user")
	}
}

func TestAddMemberToOrganization(t *testing.T) {
	testDB, userID, orgID := setupInvitationTestDB(t)
	defer testDB.Close()

	ctx := context.Background()

	err := addMemberToOrganization(ctx, userID, orgID, userID)
	if err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	var count int
	testDB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM members WHERE user_id = ? AND organization_id = ?",
		userID, orgID,
	).Scan(&count)

	if count != 1 {
		t.Errorf("Expected 1 member, got %d", count)
	}
}

func TestCreateInvitationHandler(t *testing.T) {
	testDB, _, orgID := setupInvitationTestDB(t)
	defer testDB.Close()

	SetOAuthDB(testDB)

	reqBody := map[string]interface{}{
		"organization_id": orgID,
		"email":           "newmember@example.com",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/invitations", bytes.NewBuffer(jsonBody))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "test-session-token"})
	w := httptest.NewRecorder()

	err := CreateInvitationHandler(w, req)
	if err != nil {
		t.Fatalf("CreateInvitationHandler returned error: %v", err)
	}

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["email"] != "newmember@example.com" {
		t.Errorf("Expected email 'newmember@example.com', got '%s'", response["email"])
	}
}

func TestCreateInvitationHandlerMissingEmail(t *testing.T) {
	testDB, _, orgID := setupInvitationTestDB(t)
	defer testDB.Close()

	SetOAuthDB(testDB)

	reqBody := map[string]interface{}{
		"organization_id": orgID,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/invitations", bytes.NewBuffer(jsonBody))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "test-session-token"})
	w := httptest.NewRecorder()

	err := CreateInvitationHandler(w, req)
	if err == nil {
		t.Error("Expected error for missing email")
	}

	if err.Error() != "email is required" {
		t.Errorf("Expected 'email is required' error, got '%s'", err.Error())
	}
}

func TestCreateInvitationHandlerInvalidOrg(t *testing.T) {
	testDB, _, _ := setupInvitationTestDB(t)
	defer testDB.Close()

	SetOAuthDB(testDB)

	reqBody := map[string]interface{}{
		"organization_id": 9999,
		"email":           "test@example.com",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/invitations", bytes.NewBuffer(jsonBody))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "test-session-token"})
	w := httptest.NewRecorder()

	err := CreateInvitationHandler(w, req)
	if err == nil {
		t.Error("Expected error for invalid organization")
	}
}

func TestAcceptInvitationHandler(t *testing.T) {
	testDB, _, orgID := setupInvitationTestDB(t)
	defer testDB.Close()

	SetOAuthDB(testDB)

	ctx := context.Background()

	invitation, _ := createInvitation(ctx, orgID, 1, "member@example.com")

	sessionToken := "member-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		2, sessionToken, expiresAt,
	)

	reqBody := map[string]interface{}{
		"token": invitation.Token,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/invitations/accept", bytes.NewBuffer(jsonBody))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	err := AcceptInvitationHandler(w, req)
	if err != nil {
		t.Fatalf("AcceptInvitationHandler returned error: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] != "Successfully joined organization" {
		t.Errorf("Expected success message, got '%s'", response["message"])
	}

	found, _ := getInvitationByToken(ctx, invitation.Token)
	if found.Status != "accepted" {
		t.Errorf("Expected invitation status 'accepted', got '%s'", found.Status)
	}
}

func TestAcceptInvitationHandlerInvalidToken(t *testing.T) {
	testDB, _, _ := setupInvitationTestDB(t)
	defer testDB.Close()

	SetOAuthDB(testDB)

	sessionToken := "member-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	ctx := context.Background()
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		2, sessionToken, expiresAt,
	)

	reqBody := map[string]interface{}{
		"token": "invalid-token",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/invitations/accept", bytes.NewBuffer(jsonBody))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	err := AcceptInvitationHandler(w, req)
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestAcceptInvitationHandlerExpired(t *testing.T) {
	testDB, _, orgID := setupInvitationTestDB(t)
	defer testDB.Close()

	SetOAuthDB(testDB)

	ctx := context.Background()

	invitation, _ := createInvitation(ctx, orgID, 1, "member@example.com")

	testDB.ExecContext(ctx,
		"UPDATE invitations SET expires_at = ? WHERE id = ?",
		time.Now().Add(-1*time.Hour), invitation.ID,
	)

	sessionToken := "member-session-token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		2, sessionToken, expiresAt,
	)

	reqBody := map[string]interface{}{
		"token": invitation.Token,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/invitations/accept", bytes.NewBuffer(jsonBody))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	err := AcceptInvitationHandler(w, req)
	if err == nil {
		t.Error("Expected error for expired invitation")
	}

	if err.Error() != "invitation has expired" {
		t.Errorf("Expected 'invitation has expired' error, got '%s'", err.Error())
	}
}
