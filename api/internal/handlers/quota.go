package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"wws/api/internal/models"
)

// QuotaHandler handles resource quota operations
type QuotaHandler struct {
	DB *sql.DB
}

// GetQuota retrieves the quota for an organization
func (h *QuotaHandler) GetQuota(w http.ResponseWriter, r *http.Request) {
	orgID := getOrgIDFromRequest(r)
	if orgID == 0 {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Organization ID required"))
		return
	}

	quota := &models.ResourceQuota{}
	err := h.DB.QueryRow(`
		SELECT id, organization_id, max_workspaces, max_users, max_storage_gb, 
		       max_compute_hours, max_network_bandwidth, created_at, updated_at
		FROM resource_quotas WHERE organization_id = ?`, orgID).
		Scan(&quota.ID, &quota.OrganizationID, &quota.MaxWorkspaces, &quota.MaxUsers,
			&quota.MaxStorageGB, &quota.MaxComputeHours, &quota.MaxNetworkBandwidth,
			&quota.CreatedAt, &quota.UpdatedAt)

	if err == sql.ErrNoRows {
		// Create default quota if none exists
		quota = h.createDefaultQuota(orgID)
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to retrieve quota: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, quota)
}

// createDefaultQuota creates a default quota for an organization
func (h *QuotaHandler) createDefaultQuota(orgID int) *models.ResourceQuota {
	quota := &models.ResourceQuota{
		OrganizationID:      orgID,
		MaxWorkspaces:       10,
		MaxUsers:            5,
		MaxStorageGB:        50,
		MaxComputeHours:     100,
		MaxNetworkBandwidth: 1000,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	_, err := h.DB.Exec(`
		INSERT INTO resource_quotas (organization_id, max_workspaces, max_users, 
			max_storage_gb, max_compute_hours, max_network_bandwidth, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		quota.OrganizationID, quota.MaxWorkspaces, quota.MaxUsers, quota.MaxStorageGB,
		quota.MaxComputeHours, quota.MaxNetworkBandwidth, quota.CreatedAt, quota.UpdatedAt)

	if err != nil {
		return nil
	}

	return quota
}

// UpdateQuota updates the quota for an organization
func (h *QuotaHandler) UpdateQuota(w http.ResponseWriter, r *http.Request) {
	orgID := getOrgIDFromRequest(r)
	if orgID == 0 {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Organization ID required"))
		return
	}

	var req models.QuotaUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Invalid request body: %v", err))
		return
	}

	// Validate request
	if req.MaxWorkspaces < 0 || req.MaxUsers < 0 || req.MaxStorageGB < 0 ||
		req.MaxComputeHours < 0 || req.MaxNetworkBandwidth < 0 {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Quota values must be non-negative"))
		return
	}

	// Check if quota exists
	var exists bool
	h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM resource_quotas WHERE organization_id = ?)", orgID).Scan(&exists)

	now := time.Now()
	var err error

	if exists {
		_, err = h.DB.Exec(`
			UPDATE resource_quotas SET 
				max_workspaces = ?, max_users = ?, max_storage_gb = ?,
				max_compute_hours = ?, max_network_bandwidth = ?, updated_at = ?
			WHERE organization_id = ?`,
			req.MaxWorkspaces, req.MaxUsers, req.MaxStorageGB,
			req.MaxComputeHours, req.MaxNetworkBandwidth, now, orgID)
	} else {
		_, err = h.DB.Exec(`
			INSERT INTO resource_quotas (organization_id, max_workspaces, max_users,
				max_storage_gb, max_compute_hours, max_network_bandwidth, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			orgID, req.MaxWorkspaces, req.MaxUsers, req.MaxStorageGB,
			req.MaxComputeHours, req.MaxNetworkBandwidth, now, now)
	}

	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to update quota: %v", err))
		return
	}

	// Get updated quota
	quota := &models.ResourceQuota{}
	h.DB.QueryRow(`
		SELECT id, organization_id, max_workspaces, max_users, max_storage_gb,
		       max_compute_hours, max_network_bandwidth, created_at, updated_at
		FROM resource_quotas WHERE organization_id = ?`, orgID).
		Scan(&quota.ID, &quota.OrganizationID, &quota.MaxWorkspaces, &quota.MaxUsers,
			&quota.MaxStorageGB, &quota.MaxComputeHours, &quota.MaxNetworkBandwidth,
			&quota.CreatedAt, &quota.UpdatedAt)

	WriteJSON(w, http.StatusOK, quota)
}

// GetUsage retrieves the current usage for an organization
func (h *QuotaHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	orgID := getOrgIDFromRequest(r)
	if orgID == 0 {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Organization ID required"))
		return
	}

	usage := &models.QuotaUsage{}
	err := h.DB.QueryRow(`
		SELECT id, organization_id, workspaces_count, users_count, storage_used_gb,
		       compute_hours_used, network_bandwidth_used, updated_at
		FROM quota_usage WHERE organization_id = ?`, orgID).
		Scan(&usage.ID, &usage.OrganizationID, &usage.WorkspacesCount, &usage.UsersCount,
			&usage.StorageUsedGB, &usage.ComputeHoursUsed, &usage.NetworkBandwidthUsed,
			&usage.UpdatedAt)

	if err == sql.ErrNoRows {
		// Calculate and create usage if none exists
		usage = h.calculateUsage(orgID)
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Errorf("Failed to retrieve usage: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, usage)
}

// calculateUsage calculates the current usage for an organization
func (h *QuotaHandler) calculateUsage(orgID int) *models.QuotaUsage {
	usage := &models.QuotaUsage{
		OrganizationID: orgID,
		UpdatedAt:      time.Now(),
	}

	// Count workspaces
	h.DB.QueryRow("SELECT COUNT(*) FROM workspaces WHERE organization_id = ? AND deleted_at IS NULL", orgID).Scan(&usage.WorkspacesCount)

	// Count users (members)
	h.DB.QueryRow("SELECT COUNT(*) FROM members WHERE organization_id = ?", orgID).Scan(&usage.UsersCount)

	// For storage, compute hours, and network bandwidth, we'd need additional tracking
	// For now, set to 0 or use estimates
	usage.StorageUsedGB = 0
	usage.ComputeHoursUsed = 0
	usage.NetworkBandwidthUsed = 0

	// Save usage
	_, err := h.DB.Exec(`
		INSERT INTO quota_usage (organization_id, workspaces_count, users_count,
			storage_used_gb, compute_hours_used, network_bandwidth_used, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(organization_id) DO UPDATE SET
			workspaces_count = excluded.workspaces_count,
			users_count = excluded.users_count,
			updated_at = excluded.updated_at`,
		usage.OrganizationID, usage.WorkspacesCount, usage.UsersCount,
		usage.StorageUsedGB, usage.ComputeHoursUsed, usage.NetworkBandwidthUsed, usage.UpdatedAt)

	if err != nil {
		// Log error but continue
	}

	return usage
}

// CheckQuota checks if an action would exceed quota limits
func (h *QuotaHandler) CheckQuota(w http.ResponseWriter, r *http.Request) {
	orgID := getOrgIDFromRequest(r)
	if orgID == 0 {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Organization ID required"))
		return
	}

	var req struct {
		Resource string `json:"resource"`
		Amount   int    `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to checking workspace creation
		req.Resource = "workspaces"
		req.Amount = 1
	}

	results := h.checkResourceQuota(orgID, req.Resource, req.Amount)
	WriteJSON(w, http.StatusOK, results)
}

// checkResourceQuota checks a specific resource quota
func (h *QuotaHandler) checkResourceQuota(orgID int, resource string, amount int) []models.QuotaCheckResult {
	var results []models.QuotaCheckResult

	// Get quota and usage
	var quota models.ResourceQuota
	var usage models.QuotaUsage

	h.DB.QueryRow(`
		SELECT max_workspaces, max_users, max_storage_gb,
		       max_compute_hours, max_network_bandwidth
		FROM resource_quotas WHERE organization_id = ?`, orgID).
		Scan(&quota.MaxWorkspaces, &quota.MaxUsers, &quota.MaxStorageGB,
			&quota.MaxComputeHours, &quota.MaxNetworkBandwidth)

	h.DB.QueryRow(`
		SELECT workspaces_count, users_count, storage_used_gb,
		       compute_hours_used, network_bandwidth_used
		FROM quota_usage WHERE organization_id = ?`, orgID).
		Scan(&usage.WorkspacesCount, &usage.UsersCount, &usage.StorageUsedGB,
			&usage.ComputeHoursUsed, &usage.NetworkBandwidthUsed)

	// Check based on resource type
	switch resource {
	case "workspaces":
		current := usage.WorkspacesCount
		limit := quota.MaxWorkspaces
		allowed := current+amount <= limit
		results = append(results, models.QuotaCheckResult{
			Allowed:  allowed,
			Resource: "workspaces",
			Current:  current,
			Limit:    limit,
			Message:  map[bool]string{true: "Quota available", false: "Workspace quota exceeded"}[allowed],
		})

	case "users":
		current := usage.UsersCount
		limit := quota.MaxUsers
		allowed := current+amount <= limit
		results = append(results, models.QuotaCheckResult{
			Allowed:  allowed,
			Resource: "users",
			Current:  current,
			Limit:    limit,
			Message:  map[bool]string{true: "Quota available", false: "User quota exceeded"}[allowed],
		})

	case "storage":
		current := usage.StorageUsedGB
		limit := quota.MaxStorageGB
		allowed := current+amount <= limit
		results = append(results, models.QuotaCheckResult{
			Allowed:  allowed,
			Resource: "storage",
			Current:  current,
			Limit:    limit,
			Message:  map[bool]string{true: "Quota available", false: "Storage quota exceeded"}[allowed],
		})

	case "compute":
		current := usage.ComputeHoursUsed
		limit := quota.MaxComputeHours
		allowed := current+amount <= limit
		results = append(results, models.QuotaCheckResult{
			Allowed:  allowed,
			Resource: "compute_hours",
			Current:  current,
			Limit:    limit,
			Message:  map[bool]string{true: "Quota available", false: "Compute hours quota exceeded"}[allowed],
		})

	case "network":
		current := usage.NetworkBandwidthUsed
		limit := quota.MaxNetworkBandwidth
		allowed := current+amount <= limit
		results = append(results, models.QuotaCheckResult{
			Allowed:  allowed,
			Resource: "network_bandwidth",
			Current:  current,
			Limit:    limit,
			Message:  map[bool]string{true: "Quota available", false: "Network bandwidth quota exceeded"}[allowed],
		})

	default:
		// Check all resources
		resources := []string{"workspaces", "users", "storage", "compute", "network"}
		for _, res := range resources {
			checkResults := h.checkResourceQuota(orgID, res, amount)
			results = append(results, checkResults...)
		}
	}

	return results
}

// UpdateUsage updates the usage statistics for an organization
func (h *QuotaHandler) UpdateUsage(w http.ResponseWriter, r *http.Request) {
	orgID := getOrgIDFromRequest(r)
	if orgID == 0 {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("Organization ID required"))
		return
	}

	// Recalculate usage
	usage := h.calculateUsage(orgID)
	WriteJSON(w, http.StatusOK, usage)
}

// getOrgIDFromRequest extracts organization ID from request
func getOrgIDFromRequest(r *http.Request) int {
	if orgID, ok := r.Context().Value("organization_id").(int); ok {
		return orgID
	}
	return 0
}
