package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"
)

type User struct {
	ID        int       `json:"id"`
	GitHubID  string    `json:"github_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Organization struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	OwnerID   int       `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Member struct {
	ID             int           `json:"id"`
	UserID         int           `json:"user_id"`
	OrganizationID int           `json:"organization_id"`
	Role           string        `json:"role"`
	InvitedBy      sql.NullInt64 `json:"invited_by"`
	Accepted       int           `json:"accepted"`
	CreatedAt      time.Time     `json:"created_at"`
}

func requireAuth(r *http.Request) (int, error) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return 0, fmt.Errorf("missing session cookie")
	}

	if cookie.Value == "" {
		return 0, fmt.Errorf("empty session token")
	}

	ctx := context.Background()
	sessionInfo, err := ValidateSession(ctx, cookie.Value)
	if err != nil {
		return 0, fmt.Errorf("invalid session: %w", err)
	}

	return sessionInfo.UserID, nil
}
