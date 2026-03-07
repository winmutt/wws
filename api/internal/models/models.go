package models

import "time"

type Organization struct {
	ID        int       `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	OwnerID   int       `db:"owner_id" json:"owner_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type User struct {
	ID        int       `db:"id" json:"id"`
	GithubID  string    `db:"github_id" json:"github_id"`
	Username  string    `db:"username" json:"username"`
	Email     string    `db:"email" json:"email"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type Workspace struct {
	ID             int        `db:"id" json:"id"`
	Tag            string     `db:"tag" json:"tag"`
	Name           string     `db:"name" json:"name"`
	OrganizationID int        `db:"organization_id" json:"organization_id"`
	OwnerID        int        `db:"owner_id" json:"owner_id"`
	Provider       string     `db:"provider" json:"provider"`
	Status         string     `db:"status" json:"status"`
	Config         string     `db:"config" json:"config"`
	Region         string     `db:"region" json:"region"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt      *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

type WorkspaceLanguage struct {
	ID            int    `db:"id" json:"id"`
	WorkspaceID   int    `db:"workspace_id" json:"workspace_id"`
	Language      string `db:"language" json:"language"`
	Version       string `db:"version" json:"version"`
	InstallScript string `db:"install_script" json:"install_script"`
}

type OAuthToken struct {
	ID           int       `db:"id" json:"id"`
	UserID       int       `db:"user_id" json:"user_id"`
	AccessToken  string    `db:"access_token" json:"access_token"`
	RefreshToken string    `db:"refresh_token" json:"refresh_token"`
	Expiry       time.Time `db:"expiry" json:"expiry"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type Session struct {
	ID        int       `db:"id" json:"id"`
	UserID    int       `db:"user_id" json:"user_id"`
	Token     string    `db:"token" json:"token"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type OAuthState struct {
	ID        int       `db:"id" json:"id"`
	State     string    `db:"state" json:"state"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
