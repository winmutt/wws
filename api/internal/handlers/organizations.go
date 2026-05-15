package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"wws/api/internal/db"

	"github.com/gorilla/mux"
)

type CreateOrganizationRequest struct {
	Name string `json:"name"`
}

func ListOrganizationsHandler(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	rows, err := db.DB.QueryContext(ctx,
		`SELECT id, name, owner_id, created_at, updated_at FROM organizations ORDER BY created_at DESC`,
	)
	if err != nil {
		return fmt.Errorf("failed to list organizations: %w", err)
	}
	defer rows.Close()

	var orgs []map[string]interface{}
	for rows.Next() {
		var org map[string]interface{}
		var id, ownerID int
		var name string
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &name, &ownerID, &createdAt, &updatedAt); err != nil {
			return fmt.Errorf("failed to scan organization: %w", err)
		}

		org = map[string]interface{}{
			"id":         id,
			"name":       name,
			"owner_id":   ownerID,
			"created_at": createdAt.Format(time.RFC3339),
			"updated_at": updatedAt.Format(time.RFC3339),
		}
		orgs = append(orgs, org)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating organizations: %w", err)
	}

	return WriteJSON(w, http.StatusOK, orgs)
}

func CreateOrganizationHandler(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	var req CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return fmt.Errorf("failed to decode request body: %w", err)
	}

	if req.Name == "" {
		return fmt.Errorf("organization name is required")
	}

	if len(req.Name) < 3 || len(req.Name) > 50 {
		return fmt.Errorf("organization name must be between 3 and 50 characters")
	}

	ownerID := r.URL.Query().Get("owner_id")
	if ownerID == "" {
		return fmt.Errorf("owner_id is required")
	}

	var userID int
	_, err := fmt.Sscanf(ownerID, "%d", &userID)
	if err != nil || userID <= 0 {
		return fmt.Errorf("invalid owner_id")
	}

	now := time.Now()
	result, err := db.DB.ExecContext(ctx,
		`INSERT INTO organizations (name, owner_id, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		req.Name, userID, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to create organization: %w", err)
	}

	orgID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get organization ID: %w", err)
	}

	org := map[string]interface{}{
		"id":         orgID,
		"name":       req.Name,
		"owner_id":   userID,
		"created_at": now.Format(time.RFC3339),
		"updated_at": now.Format(time.RFC3339),
	}

	return WriteJSON(w, http.StatusCreated, org)
}

func GetOrganizationHandler(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	idStr := mux.Vars(r)["id"]
	if idStr == "" {
		return fmt.Errorf("organization id is required")
	}

	var id int
	_, err := fmt.Sscanf(idStr, "%d", &id)
	if err != nil || id <= 0 {
		return fmt.Errorf("invalid organization id")
	}

	var orgID, ownerID int
	var name string
	var createdAt, updatedAt time.Time
	err = db.DB.QueryRowContext(ctx,
		`SELECT id, name, owner_id, created_at, updated_at FROM organizations WHERE id = ?`,
		id,
	).Scan(&orgID, &name, &ownerID, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return fmt.Errorf("organization not found")
	}
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	org := map[string]interface{}{
		"id":         orgID,
		"name":       name,
		"owner_id":   ownerID,
		"created_at": createdAt.Format(time.RFC3339),
		"updated_at": updatedAt.Format(time.RFC3339),
	}

	return WriteJSON(w, http.StatusOK, org)
}

func UpdateOrganizationHandler(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	idStr := mux.Vars(r)["id"]
	if idStr == "" {
		return fmt.Errorf("organization id is required")
	}

	var id int
	_, err := fmt.Sscanf(idStr, "%d", &id)
	if err != nil || id <= 0 {
		return fmt.Errorf("invalid organization id")
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return fmt.Errorf("invalid request body")
	}

	if req.Name == "" {
		return fmt.Errorf("name is required")
	}

	_, err = db.DB.ExecContext(ctx,
		`UPDATE organizations SET name = ?, updated_at = ? WHERE id = ?`,
		req.Name, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}

	org := map[string]interface{}{
		"id":         id,
		"name":       req.Name,
		"updated_at": time.Now().Format(time.RFC3339),
	}

	return WriteJSON(w, http.StatusOK, org)
}

func DeleteOrganizationHandler(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	idStr := mux.Vars(r)["id"]
	if idStr == "" {
		return fmt.Errorf("organization id is required")
	}

	var id int
	_, err := fmt.Sscanf(idStr, "%d", &id)
	if err != nil || id <= 0 {
		return fmt.Errorf("invalid organization id")
	}

	_, err = db.DB.ExecContext(ctx,
		`DELETE FROM organizations WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	return WriteJSON(w, http.StatusOK, map[string]string{"message": "Organization deleted"})
}
