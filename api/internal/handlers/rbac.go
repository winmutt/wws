package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"wws/api/internal/db"
)

const (
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleViewer = "viewer"
	RoleOwner  = "owner"
)

var validRoles = map[string]bool{
	RoleAdmin:  true,
	RoleMember: true,
	RoleViewer: true,
	RoleOwner:  true,
}

type RoleAssignment struct {
	ID             int       `json:"id"`
	UserID         int       `json:"user_id"`
	OrganizationID int       `json:"organization_id"`
	Role           string    `json:"role"`
	AssignedBy     int       `json:"assigned_by"`
	AssignedAt     time.Time `json:"assigned_at"`
}

type AssignRoleRequest struct {
	UserID         int    `json:"user_id"`
	OrganizationID int    `json:"organization_id"`
	Role           string `json:"role"`
}

type MemberDetail struct {
	ID             int       `json:"id"`
	UserID         int       `json:"user_id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	OrganizationID int       `json:"organization_id"`
	Role           string    `json:"role"`
	InvitedBy      int       `json:"invited_by,omitempty"`
	Accepted       bool      `json:"accepted"`
	CreatedAt      time.Time `json:"created_at"`
}

type ListMembersRequest struct {
	OrganizationID int `json:"organization_id" form:"organization_id"`
}

func isValidRole(role string) bool {
	return validRoles[role]
}

func assignRole(ctx context.Context, userID, orgID, assignedBy int, role string) error {
	if !isValidRole(role) {
		return fmt.Errorf("invalid role: %s", role)
	}

	result, err := db.DB.ExecContext(ctx,
		`INSERT INTO members (user_id, organization_id, role, invited_by, accepted, created_at)
		 VALUES (?, ?, ?, ?, 1, ?)
		 ON CONFLICT(user_id, organization_id) DO UPDATE SET role = ?`,
		userID, orgID, role, assignedBy, time.Now(), role,
	)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows affected")
	}

	return nil
}

func getUserByID(ctx context.Context, userID int) (*User, error) {
	var user User
	err := db.DB.QueryRowContext(ctx,
		`SELECT id, github_id, username, email, created_at FROM users WHERE id = ?`,
		userID,
	).Scan(&user.ID, &user.GitHubID, &user.Username, &user.Email, &user.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	return &user, nil
}

func listMembersByOrganization(ctx context.Context, orgID int) ([]MemberDetail, error) {
	rows, err := db.DB.QueryContext(ctx,
		`SELECT m.id, m.user_id, m.organization_id, m.role, m.invited_by, m.accepted, m.created_at,
			 u.username, u.email
		 FROM members m
		 JOIN users u ON m.user_id = u.id
		 WHERE m.organization_id = ?
		 ORDER BY m.created_at ASC`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list members: %w", err)
	}
	defer rows.Close()

	var members []MemberDetail
	for rows.Next() {
		var member MemberDetail
		var invitedBy sql.NullInt64
		var acceptedInt int

		err := rows.Scan(
			&member.ID, &member.UserID, &member.OrganizationID, &member.Role,
			&invitedBy, &acceptedInt, &member.CreatedAt,
			&member.Username, &member.Email,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}

		if invitedBy.Valid {
			member.InvitedBy = int(invitedBy.Int64)
		}
		member.Accepted = acceptedInt == 1

		members = append(members, member)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating members: %w", err)
	}

	return members, nil
}

func canModifyRole(requesterID, targetUserID, orgID int, requesterRole, targetRole string) bool {
	if requesterID == targetUserID {
		return requesterRole == RoleOwner
	}

	roleHierarchy := map[string]int{
		RoleViewer: 1,
		RoleMember: 2,
		RoleAdmin:  3,
		RoleOwner:  4,
	}

	return roleHierarchy[requesterRole] > roleHierarchy[targetRole]
}

func AssignRoleHandler(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var req AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}

	if req.UserID == 0 {
		return fmt.Errorf("user_id is required")
	}

	if req.OrganizationID == 0 {
		return fmt.Errorf("organization_id is required")
	}

	if req.Role == "" {
		return fmt.Errorf("role is required")
	}

	if !isValidRole(req.Role) {
		return fmt.Errorf("invalid role. Valid roles are: admin, member, viewer")
	}

	requesterID, err := requireAuth(r)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	_, err = getOrganizationByID(ctx, req.OrganizationID)
	if err != nil {
		return fmt.Errorf("organization not found: %w", err)
	}

	requesterMember, err := getMemberByUserAndOrg(ctx, requesterID, req.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to get requester membership: %w", err)
	}

	if requesterMember == nil {
		return fmt.Errorf("you are not a member of this organization")
	}

	if requesterMember.Role != RoleOwner && requesterMember.Role != RoleAdmin {
		return fmt.Errorf("only admins and owners can assign roles")
	}

	targetMember, err := getMemberByUserAndOrg(ctx, req.UserID, req.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to get target member: %w", err)
	}

	if targetMember == nil {
		return fmt.Errorf("user is not a member of this organization")
	}

	if requesterMember.Role != RoleOwner {
		if !canModifyRole(requesterID, req.UserID, req.OrganizationID, requesterMember.Role, targetMember.Role) {
			return fmt.Errorf("you cannot modify the role of users with equal or higher privileges")
		}
	}

	if err := assignRole(ctx, req.UserID, req.OrganizationID, requesterID, req.Role); err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	updatedMember, _ := getMemberByUserAndOrg(ctx, req.UserID, req.OrganizationID)

	WriteJSON(w, http.StatusOK, updatedMember)

	log.Printf("User %d assigned role '%s' to user %d in organization %d",
		requesterID, req.Role, req.UserID, req.OrganizationID)

	return nil
}

func ListMembersHandler(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	orgIDStr := r.URL.Query().Get("organization_id")
	if orgIDStr == "" {
		return fmt.Errorf("organization_id query parameter is required")
	}

	var orgID int
	fmt.Sscanf(orgIDStr, "%d", &orgID)
	if orgID == 0 {
		return fmt.Errorf("invalid organization_id")
	}

	_, err := requireAuth(r)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	org, err := getOrganizationByID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("organization not found: %w", err)
	}

	members, err := listMembersByOrganization(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to list members: %w", err)
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"organization_id":   orgID,
		"organization_name": org.Name,
		"members":           members,
		"total":             len(members),
	})

	log.Printf("Listed %d members for organization %d", len(members), orgID)

	return nil
}

func GetMemberHandler(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	userIDStr := r.URL.Query().Get("user_id")
	orgIDStr := r.URL.Query().Get("organization_id")

	if userIDStr == "" || orgIDStr == "" {
		return fmt.Errorf("user_id and organization_id query parameters are required")
	}

	var userID, orgID int
	fmt.Sscanf(userIDStr, "%d", &userID)
	fmt.Sscanf(orgIDStr, "%d", &orgID)

	if userID == 0 || orgID == 0 {
		return fmt.Errorf("invalid user_id or organization_id")
	}

	_, err := requireAuth(r)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	member, err := getMemberByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		return fmt.Errorf("failed to get member: %w", err)
	}

	if member == nil {
		return fmt.Errorf("member not found")
	}

	WriteJSON(w, http.StatusOK, member)

	log.Printf("Retrieved member %d in organization %d", userID, orgID)

	return nil
}

func RemoveMemberHandler(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	userIDStr := r.URL.Query().Get("user_id")
	orgIDStr := r.URL.Query().Get("organization_id")

	if userIDStr == "" || orgIDStr == "" {
		return fmt.Errorf("user_id and organization_id query parameters are required")
	}

	var userID, orgID int
	fmt.Sscanf(userIDStr, "%d", &userID)
	fmt.Sscanf(orgIDStr, "%d", &orgID)

	if userID == 0 || orgID == 0 {
		return fmt.Errorf("invalid user_id or organization_id")
	}

	requesterID, err := requireAuth(r)
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	if userID == requesterID {
		return fmt.Errorf("you cannot remove yourself from the organization")
	}

	orgCheck, err := getOrganizationByID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("organization not found: %w", err)
	}

	if orgCheck.OwnerID == userID {
		return fmt.Errorf("you cannot remove the organization owner")
	}

	requesterMember, err := getMemberByUserAndOrg(ctx, requesterID, orgID)
	if err != nil {
		return fmt.Errorf("failed to get requester membership: %w", err)
	}

	if requesterMember == nil {
		return fmt.Errorf("you are not a member of this organization")
	}

	if requesterMember.Role != RoleOwner && requesterMember.Role != RoleAdmin {
		return fmt.Errorf("only admins and owners can remove members")
	}

	targetMember, err := getMemberByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		return fmt.Errorf("failed to get target member: %w", err)
	}

	if targetMember == nil {
		return fmt.Errorf("member not found")
	}

	if requesterMember.Role != RoleOwner && targetMember.Role == RoleAdmin {
		return fmt.Errorf("only owners can remove admins")
	}

	_, err = db.DB.ExecContext(ctx,
		`DELETE FROM members WHERE user_id = ? AND organization_id = ?`,
		userID, orgID,
	)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	log.Printf("User %d removed user %d from organization %d",
		requesterID, userID, orgID)

	WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Member removed successfully",
	})

	return nil
}
