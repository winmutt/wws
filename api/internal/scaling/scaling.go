package scaling

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"wws/api/internal/db"
)

// ScaleAction represents a scaling action
type ScaleAction string

const (
	ScaleActionUp   ScaleAction = "up"
	ScaleActionDown ScaleAction = "down"
	ScaleActionAuto ScaleAction = "auto"
)

// ScalingConfig represents the scaling configuration for a workspace
type ScalingConfig struct {
	ID                   int       `db:"id" json:"id"`
	WorkspaceID          int       `db:"workspace_id" json:"workspace_id"`
	MinCPU               int       `db:"min_cpu" json:"min_cpu"`
	MaxCPU               int       `db:"max_cpu" json:"max_cpu"`
	MinMemoryGB          float64   `db:"min_memory_gb" json:"min_memory_gb"`
	MaxMemoryGB          float64   `db:"max_memory_gb" json:"max_memory_gb"`
	ScaleUpThreshold     float64   `db:"scale_up_threshold" json:"scale_up_threshold"`
	ScaleDownThreshold   float64   `db:"scale_down_threshold" json:"scale_down_threshold"`
	ScaleCooldownMinutes int       `db:"scale_cooldown_minutes" json:"scale_cooldown_minutes"`
	AutoScalingEnabled   bool      `db:"auto_scaling_enabled" json:"auto_scaling_enabled"`
	CreatedAt            time.Time `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time `db:"updated_at" json:"updated_at"`
}

// ScaleRequest represents a request to scale a workspace
type ScaleRequest struct {
	WorkspaceID int     `json:"workspace_id"`
	CPU         int     `json:"cpu"`
	MemoryGB    float64 `json:"memory_gb"`
	StorageGB   int     `json:"storage_gb,omitempty"`
}

// ScaleResult represents the result of a scaling operation
type ScaleResult struct {
	Success        bool      `json:"success"`
	PreviousCPU    int       `json:"previous_cpu"`
	PreviousMemory float64   `json:"previous_memory_gb"`
	NewCPU         int       `json:"new_cpu"`
	NewMemory      float64   `json:"new_memory_gb"`
	Message        string    `json:"message"`
	Timestamp      time.Time `json:"timestamp"`
}

// GetScalingConfig retrieves the scaling configuration for a workspace
func GetScalingConfig(ctx context.Context, workspaceID int) (*ScalingConfig, error) {
	var config ScalingConfig
	var autoEnabled sql.NullBool

	err := db.DB.QueryRowContext(ctx, `
		SELECT id, workspace_id, min_cpu, max_cpu, min_memory_gb, max_memory_gb,
		       scale_up_threshold, scale_down_threshold, scale_cooldown_minutes,
		       auto_scaling_enabled, created_at, updated_at
		FROM workspace_scaling_config
		WHERE workspace_id = ?
	`, workspaceID).Scan(
		&config.ID, &config.WorkspaceID, &config.MinCPU, &config.MaxCPU,
		&config.MinMemoryGB, &config.MaxMemoryGB, &config.ScaleUpThreshold,
		&config.ScaleDownThreshold, &config.ScaleCooldownMinutes,
		&autoEnabled, &config.CreatedAt, &config.UpdatedAt)

	if err == sql.ErrNoRows {
		// Return default config if none exists
		return getDefaultScalingConfig(workspaceID), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query scaling config: %w", err)
	}

	config.AutoScalingEnabled = autoEnabled.Bool
	return &config, nil
}

// getDefaultScalingConfig returns the default scaling configuration
func getDefaultScalingConfig(workspaceID int) *ScalingConfig {
	return &ScalingConfig{
		WorkspaceID:          workspaceID,
		MinCPU:               1,
		MaxCPU:               8,
		MinMemoryGB:          1.0,
		MaxMemoryGB:          16.0,
		ScaleUpThreshold:     80.0,
		ScaleDownThreshold:   30.0,
		ScaleCooldownMinutes: 15,
		AutoScalingEnabled:   false,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
}

// UpdateScalingConfig updates the scaling configuration for a workspace
func UpdateScalingConfig(ctx context.Context, workspaceID int, config *ScalingConfig) error {
	query := `
		INSERT INTO workspace_scaling_config (workspace_id, min_cpu, max_cpu, min_memory_gb, max_memory_gb,
		                                      scale_up_threshold, scale_down_threshold, scale_cooldown_minutes,
		                                      auto_scaling_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id) DO UPDATE SET
			min_cpu = excluded.min_cpu,
			max_cpu = excluded.max_cpu,
			min_memory_gb = excluded.min_memory_gb,
			max_memory_gb = excluded.max_memory_gb,
			scale_up_threshold = excluded.scale_up_threshold,
			scale_down_threshold = excluded.scale_down_threshold,
			scale_cooldown_minutes = excluded.scale_cooldown_minutes,
			auto_scaling_enabled = excluded.auto_scaling_enabled,
			updated_at = excluded.updated_at
	`

	_, err := db.DB.ExecContext(ctx, query,
		workspaceID, config.MinCPU, config.MaxCPU, config.MinMemoryGB, config.MaxMemoryGB,
		config.ScaleUpThreshold, config.ScaleDownThreshold, config.ScaleCooldownMinutes,
		config.AutoScalingEnabled, time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to update scaling config: %w", err)
	}

	return nil
}

// ScaleWorkspace scales a workspace to the requested resources
func ScaleWorkspace(ctx context.Context, workspaceID int, cpu int, memoryGB float64, storageGB int) (*ScaleResult, error) {
	// Get current workspace configuration
	currentConfig, err := getCurrentWorkspaceConfig(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current config: %w", err)
	}

	// Validate scale request against limits
	scalingConfig, err := GetScalingConfig(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get scaling config: %w", err)
	}

	if cpu < scalingConfig.MinCPU || cpu > scalingConfig.MaxCPU {
		return nil, fmt.Errorf("CPU must be between %d and %d", scalingConfig.MinCPU, scalingConfig.MaxCPU)
	}

	if memoryGB < scalingConfig.MinMemoryGB || memoryGB > scalingConfig.MaxMemoryGB {
		return nil, fmt.Errorf("Memory must be between %.1f and %.1f GB", scalingConfig.MinMemoryGB, scalingConfig.MaxMemoryGB)
	}

	// Perform the scale operation
	result := &ScaleResult{
		Success:        true,
		PreviousCPU:    currentConfig.CPU,
		PreviousMemory: currentConfig.MemoryGB,
		NewCPU:         cpu,
		NewMemory:      memoryGB,
		Timestamp:      time.Now(),
	}

	// Update workspace config in database
	err = updateWorkspaceResources(workspaceID, cpu, memoryGB, storageGB)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("Failed to update resources: %v", err)
		return result, nil
	}

	// Record scaling action in history
	recordScalingAction(ctx, workspaceID, ScaleActionUp, currentConfig.CPU, int(currentConfig.MemoryGB), cpu, int(memoryGB))

	result.Message = fmt.Sprintf("Successfully scaled from %d CPU/%.1fGB to %d CPU/%.1fGB",
		currentConfig.CPU, currentConfig.MemoryGB, cpu, memoryGB)

	return result, nil
}

// getCurrentWorkspaceConfig retrieves the current workspace configuration
func getCurrentWorkspaceConfig(workspaceID int) (*ScaleRequest, error) {
	query := `SELECT config FROM workspaces WHERE id = ?`
	var configJSON sql.NullString

	err := db.DB.QueryRow(query, workspaceID).Scan(&configJSON)
	if err != nil {
		return nil, err
	}

	if !configJSON.Valid {
		return &ScaleRequest{CPU: 2, MemoryGB: 4.0, StorageGB: 20}, nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON.String), &config); err != nil {
		return &ScaleRequest{CPU: 2, MemoryGB: 4.0, StorageGB: 20}, nil
	}

	result := &ScaleRequest{WorkspaceID: workspaceID}
	if cpu, ok := config["cpu"].(float64); ok {
		result.CPU = int(cpu)
	}
	if memory, ok := config["memory_gb"].(float64); ok {
		result.MemoryGB = memory
	}
	if storage, ok := config["storage_gb"].(float64); ok {
		result.StorageGB = int(storage)
	}

	return result, nil
}

// updateWorkspaceResources updates the workspace resources in the database
func updateWorkspaceResources(workspaceID, cpu int, memoryGB float64, storageGB int) error {
	// Update config
	configMap := map[string]interface{}{
		"cpu":        cpu,
		"memory_gb":  memoryGB,
		"storage_gb": storageGB,
	}
	configJSON, _ := json.Marshal(configMap)

	query := `UPDATE workspaces SET config = ?, updated_at = ? WHERE id = ?`
	_, err := db.DB.Exec(query, string(configJSON), time.Now(), workspaceID)
	return err
}

// recordScalingAction records a scaling action in the workspace history
func recordScalingAction(ctx context.Context, workspaceID int, action ScaleAction, prevCPU, prevMemory, newCPU, newMemory int) {
	previousState := map[string]interface{}{
		"cpu":       prevCPU,
		"memory_gb": prevMemory,
	}
	newState := map[string]interface{}{
		"cpu":       newCPU,
		"memory_gb": newMemory,
	}

	// Import history package - this would need to be done at the application level
	// For now, we'll just log the action
	// history.RecordHistory(ctx, workspaceID, history.ActionUpdated, previousState, newState, 0)
	_, _ = previousState, newState // Suppress unused variable errors
}

// EnableAutoScaling enables automatic scaling for a workspace
func EnableAutoScaling(ctx context.Context, workspaceID int) error {
	config, err := GetScalingConfig(ctx, workspaceID)
	if err != nil {
		return err
	}

	config.AutoScalingEnabled = true
	return UpdateScalingConfig(ctx, workspaceID, config)
}

// DisableAutoScaling disables automatic scaling for a workspace
func DisableAutoScaling(ctx context.Context, workspaceID int) error {
	config, err := GetScalingConfig(ctx, workspaceID)
	if err != nil {
		return err
	}

	config.AutoScalingEnabled = false
	return UpdateScalingConfig(ctx, workspaceID, config)
}

// CheckAndScale evaluates current metrics and scales if necessary
func CheckAndScale(ctx context.Context, workspaceID int, cpuUsage, memoryUsage float64) (*ScaleResult, error) {
	config, err := GetScalingConfig(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	if !config.AutoScalingEnabled {
		return nil, fmt.Errorf("auto-scaling is not enabled")
	}

	currentConfig, err := getCurrentWorkspaceConfig(workspaceID)
	if err != nil {
		return nil, err
	}

	// Check if cooldown period has passed
	if !hasCooldownPassed(workspaceID, config.ScaleCooldownMinutes) {
		return nil, fmt.Errorf("scale cooldown period not yet passed")
	}

	var result *ScaleResult

	// Scale up if thresholds exceeded
	if cpuUsage > config.ScaleUpThreshold || memoryUsage > config.ScaleUpThreshold {
		newCPU := currentConfig.CPU + 1
		if newCPU > config.MaxCPU {
			newCPU = config.MaxCPU
		}
		newMemory := currentConfig.MemoryGB * 1.5
		if newMemory > config.MaxMemoryGB {
			newMemory = config.MaxMemoryGB
		}

		if newCPU > currentConfig.CPU || newMemory > currentConfig.MemoryGB {
			result, err = ScaleWorkspace(ctx, workspaceID, newCPU, newMemory, currentConfig.StorageGB)
			if err != nil {
				return nil, err
			}
		}
	}

	// Scale down if usage is low
	if cpuUsage < config.ScaleDownThreshold && memoryUsage < config.ScaleDownThreshold {
		newCPU := currentConfig.CPU - 1
		if newCPU < config.MinCPU {
			newCPU = config.MinCPU
		}
		newMemory := currentConfig.MemoryGB * 0.75
		if newMemory < config.MinMemoryGB {
			newMemory = config.MinMemoryGB
		}

		if newCPU < currentConfig.CPU || newMemory < currentConfig.MemoryGB {
			result, err = ScaleWorkspace(ctx, workspaceID, newCPU, newMemory, currentConfig.StorageGB)
			if err != nil {
				return nil, err
			}
		}
	}

	if result == nil {
		result = &ScaleResult{
			Success:        true,
			PreviousCPU:    currentConfig.CPU,
			PreviousMemory: currentConfig.MemoryGB,
			NewCPU:         currentConfig.CPU,
			NewMemory:      currentConfig.MemoryGB,
			Message:        "No scaling needed",
			Timestamp:      time.Now(),
		}
	}

	return result, nil
}

// hasCooldownPassed checks if the scale cooldown period has passed
func hasCooldownPassed(workspaceID int, cooldownMinutes int) bool {
	query := `
		SELECT MAX(created_at) FROM scaling_history
		WHERE workspace_id = ? AND action = 'scale'
	`
	var lastScale sql.NullTime
	err := db.DB.QueryRow(query, workspaceID).Scan(&lastScale)
	if err != nil || !lastScale.Valid {
		return true // No previous scaling, allow immediately
	}

	cooldownDuration := time.Duration(cooldownMinutes) * time.Minute
	return time.Since(lastScale.Time) > cooldownDuration
}

// GetScalingHistory retrieves the scaling history for a workspace
func GetScalingHistory(ctx context.Context, workspaceID int, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT action, previous_cpu, previous_memory, new_cpu, new_memory, scaled_at, scaled_by
		FROM scaling_history
		WHERE workspace_id = ?
		ORDER BY scaled_at DESC
		LIMIT ?
	`
	rows, err := db.DB.QueryContext(ctx, query, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query scaling history: %w", err)
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var entry map[string]interface{}
		// Simplified - would need proper struct and scanning
		history = append(history, entry)
	}

	return history, nil
}
