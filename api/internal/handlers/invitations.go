package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"wws/api/internal/db"
)

type Invitation struct {
	ID             int           `json:"id"`
	OrganizationID int           `json:"organization_id"`
	Email          string        `json:"email"`
	Token          string        `json:"token"`
	Status         string        `json:"status"`
	CreatedByID    int           `json:"created_by"`
	AcceptedBy     sql.NullInt64 `json:"accepted_by,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	ExpiresAt      time.Time     `json:"expires_at"`
}

type CreateInvitationRequest struct {
	OrganizationID int    `json:"organization_id"`
	Email          string `json:"email"`
}

type AcceptInvitationRequest struct {
	Token string `json:"token"`
}

func generateInvitationToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func createInvitation(ctx context.Context, orgID, createdByID int, email string) (*Invitation, error) {
	token, err := generateInvitationToken()
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	result, err := db.DB.ExecContext(ctx,
		`INSERT INTO invitations (organization_id, email, token, status, created_by, created_at, expires_at)
		 VALUES (?, ?, ?, 'pending', ?, ?, ?)`,
		orgID, email, token, createdByID, time.Now(), expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get invitation ID: %w", err)
	}

	return &Invitation{
		ID:             int(id),
		OrganizationID: orgID,
		Email:          email,
		Token:          token,
		Status:         "pending",
		CreatedByID:    createdByID,
		CreatedAt:      time.Now(),
		ExpiresAt:      expiresAt,
	}, nil
}

func getInvitationByToken(ctx context.Context, token string) (*Invitation, error) {
	var invitation Invitation
	var acceptedBy sql.NullInt64

	err := db.DB.QueryRowContext(ctx,
		`SELECT id, organization_id, email, token, status, created_by, accepted_by, created_at, expires_at
		 FROM invitations WHERE token = ?`,
		token,
	).Scan(&invitation.ID, &invitation.OrganizationID, &invitation.Email, &invitation.Token,
		&invitation.Status, &invitation.CreatedByID, &acceptedBy, &invitation.CreatedAt, &invitation.ExpiresAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invitation not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch invitation: %w", err)
	}

	invitation.AcceptedBy = acceptedBy
	return &invitation, nil
}

func updateInvitationStatus(ctx context.Context, invitationID int, status string, userID *int) error {
	var userIDValue interface{}
	if userID != nil {
		userIDValue = *userID
	} else {
		userIDValue = nil
	}

	_, err := db.DB.ExecContext(ctx,
		`UPDATE invitations SET status = ?, accepted_by = ? WHERE id = ?`,
		status, userIDValue, invitationID,
	)
	if err != nil {
		return fmt.Errorf("failed to update invitation status: %w", err)
	}

	return nil
}

func getOrganizationByID(ctx context.Context, orgID int) (*Organization, error) {
	var org Organization
	err := db.DB.QueryRowContext(ctx,
		`SELECT id, name, owner_id, created_at, updated_at FROM organizations WHERE id = ?`,
		orgID,
	).Scan(&org.ID, &org.Name, &org.OwnerID, &org.CreatedAt, &org.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organization: %w", err)
	}

	return &org, nil
}

func getUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := db.DB.QueryRowContext(ctx,
		`SELECT id, github_id, username, email, created_at FROM users WHERE email = ?`,
		email,
	).Scan(&user.ID, &user.GitHubID, &user.Username, &user.Email, &user.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	return &user, nil
}

func addMemberToOrganization(ctx context.Context, userID, orgID, invitedBy int) error {
	_, err := db.DB.ExecContext(ctx,
		`INSERT INTO members (user_id, organization_id, role, invited_by, accepted, created_at)
		 VALUES (?, ?, 'member', ?, 1, ?)`,
		userID, orgID, invitedBy, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	return nil
}

func CreateInvitationHandler(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var req CreateInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}

	if req.Email == "" {
		return fmt.Errorf("email is required")
	}

	if req.OrganizationID == 0 {
		return fmt.Errorf("organization_id is required")
	}

	orgIDInt := req.OrganizationID

	org, err := getOrganizationByID(ctx, orgIDInt)
	if err != nil {
		return fmt.Errorf("organization not found: %w", err)
	}

	userID, err := requireAuth(r)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	if org.OwnerID != userID {
		return fmt.Errorf("only organization owners can create invitations")
	}

	existingUser, err := getUserByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	if existingUser != nil {
		existingMember, _ := getMemberByUserAndOrg(ctx, existingUser.ID, orgIDInt)
		if existingMember != nil {
			return fmt.Errorf("user is already a member of this organization")
		}
	}

	invitation, err := createInvitation(ctx, orgIDInt, userID, req.Email)
	if err != nil {
		return fmt.Errorf("failed to create invitation: %w", err)
	}

	WriteJSON(w, http.StatusCreated, invitation)
	log.Printf("Created invitation %d for %s to join organization %d",
		invitation.ID, invitation.Email, invitation.OrganizationID)

	return nil
}

func AcceptInvitationHandler(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var req AcceptInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}

	if req.Token == "" {
		return fmt.Errorf("invitation token is required")
	}

	invitation, err := getInvitationByToken(ctx, req.Token)
	if err != nil {
		return fmt.Errorf("invalid or expired invitation: %w", err)
	}

	if invitation.Status != "pending" {
		return fmt.Errorf("invitation has already been %s", invitation.Status)
	}

	if time.Now().After(invitation.ExpiresAt) {
		return fmt.Errorf("invitation has expired")
	}

	userID, err := requireAuth(r)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	user, err := getUserByEmail(ctx, invitation.Email)
	if err != nil {
		return fmt.Errorf("failed to fetch user: %w", err)
	}

	if user == nil {
		return fmt.Errorf("user not found for invitation email")
	}

	if user.ID != userID {
		return fmt.Errorf("invitation is not for this user")
	}

	if err := updateInvitationStatus(ctx, invitation.ID, "accepted", &userID); err != nil {
		return fmt.Errorf("failed to accept invitation: %w", err)
	}

	if err := addMemberToOrganization(ctx, userID, invitation.OrganizationID, invitation.CreatedByID); err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Successfully joined organization",
	})

	log.Printf("User %d accepted invitation %d and joined organization %d",
		userID, invitation.ID, invitation.OrganizationID)

	return nil
}

func getMemberByUserAndOrg(ctx context.Context, userID, orgID int) (*Member, error) {
	var member Member
	var invitedBy sql.NullInt64

	err := db.DB.QueryRowContext(ctx,
		`SELECT id, user_id, organization_id, role, invited_by, accepted, created_at
		 FROM members WHERE user_id = ? AND organization_id = ?`,
		userID, orgID,
	).Scan(&member.ID, &member.UserID, &member.OrganizationID, &member.Role,
		&invitedBy, &member.Accepted, &member.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch member: %w", err)
	}

	if invitedBy.Valid {
		member.InvitedBy = invitedBy
	}

	return &member, nil
}

func GetMemberByUserAndOrg(ctx context.Context, userID, orgID int) (*Member, error) {
	return getMemberByUserAndOrg(ctx, userID, orgID)
}
