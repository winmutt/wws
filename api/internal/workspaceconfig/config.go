package workspaceconfig

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"wws/api/internal/db"
)

// WorkspaceConfig represents the full configuration of a workspace
type WorkspaceConfig struct {
	WorkspaceID     int                    `json:"workspace_id"`
	Provider        string                 `json:"provider"`
	ProviderConfig  map[string]interface{} `json:"provider_config"`
	LanguageConfig  []LanguageConfig       `json:"language_config"`
	EnvVariables    map[string]string      `json:"env_variables"`
	Resources       ResourceConfig         `json:"resources"`
	EditorConfig    EditorConfig           `json:"editor_config"`
	NetworkConfig   NetworkConfig          `json:"network_config"`
	StorageConfig   StorageConfig          `json:"storage_config"`
	BootstrapScript string                 `json:"bootstrap_script"`
	CustomConfig    map[string]interface{} `json:"custom_config"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// LanguageConfig represents language configuration
type LanguageConfig struct {
	Language string `json:"language"`
	Version  string `json:"version"`
}

// ResourceConfig represents resource allocation
type ResourceConfig struct {
	CPU       int     `json:"cpu"`
	MemoryGB  float64 `json:"memory_gb"`
	StorageGB int     `json:"storage_gb"`
	GPUs      int     `json:"gpus,omitempty"`
}

// EditorConfig represents editor configuration
type EditorConfig struct {
	Type    string            `json:"type"`
	Port    int               `json:"port"`
	Plugins []string          `json:"plugins"`
	Config  map[string]string `json:"config"`
}

// NetworkConfig represents network configuration
type NetworkConfig struct {
	IPAddress       string   `json:"ip_address"`
	ExposedPorts    []int    `json:"exposed_ports"`
	NetworkMode     string   `json:"network_mode"`
	AllowedNetworks []string `json:"allowed_networks"`
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	HomeDirSizeGB   int      `json:"home_dir_size_gb"`
	DataDirSizeGB   int      `json:"data_dir_size_gb"`
	MountPoints     []string `json:"mount_points"`
	SnapshotEnabled bool     `json:"snapshot_enabled"`
}

// SaveWorkspaceConfig saves or updates workspace configuration
func SaveWorkspaceConfig(ctx context.Context, workspaceID int, config *WorkspaceConfig) error {
	config.UpdatedAt = time.Now()

	jsonBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	_, err = db.DB.ExecContext(ctx, `
		INSERT INTO workspace_configurations (workspace_id, config_data, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(workspace_id) DO UPDATE SET
			config_data = excluded.config_data,
			updated_at = excluded.updated_at
	`, workspaceID, string(jsonBytes), config.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// GetWorkspaceConfig retrieves workspace configuration
func GetWorkspaceConfig(ctx context.Context, workspaceID int) (*WorkspaceConfig, error) {
	var configJSON string
	err := db.DB.QueryRowContext(ctx,
		"SELECT config_data FROM workspace_configurations WHERE workspace_id = ?",
		workspaceID).Scan(&configJSON)

	if err == sql.ErrNoRows {
		// Return default config if none exists
		return getDefaultConfig(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query config: %w", err)
	}

	var config WorkspaceConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	config.WorkspaceID = workspaceID
	return &config, nil
}

// GetWorkspaceConfigSimple retrieves a simplified config from the workspaces table
func GetWorkspaceConfigSimple(ctx context.Context, workspaceID int) (string, error) {
	var config string
	err := db.DB.QueryRowContext(ctx,
		"SELECT config FROM workspaces WHERE id = ?", workspaceID).Scan(&config)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("workspace not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to query config: %w", err)
	}

	return config, nil
}

// UpdateWorkspaceConfigField updates a specific field in the workspace configuration
func UpdateWorkspaceConfigField(ctx context.Context, workspaceID int, field string, value interface{}) error {
	config, err := GetWorkspaceConfig(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	switch field {
	case "resources.cpu":
		if v, ok := value.(int); ok {
			config.Resources.CPU = v
		}
	case "resources.memory_gb":
		if v, ok := value.(float64); ok {
			config.Resources.MemoryGB = v
		}
	case "resources.storage_gb":
		if v, ok := value.(int); ok {
			config.Resources.StorageGB = v
		}
	case "editor.type":
		if v, ok := value.(string); ok {
			config.EditorConfig.Type = v
		}
	case "provider":
		if v, ok := value.(string); ok {
			config.Provider = v
		}
	default:
		if config.CustomConfig == nil {
			config.CustomConfig = make(map[string]interface{})
		}
		config.CustomConfig[field] = value
	}

	return SaveWorkspaceConfig(ctx, workspaceID, config)
}

// GetWorkspaceConfigHistory retrieves configuration history for a workspace
func GetWorkspaceConfigHistory(ctx context.Context, workspaceID int, limit int) ([]WorkspaceConfig, error) {
	query := `
		SELECT config_data FROM workspace_configurations_history
		WHERE workspace_id = ?
		ORDER BY version DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.DB.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query config history: %w", err)
	}
	defer rows.Close()

	var configs []WorkspaceConfig
	for rows.Next() {
		var configJSON string
		if err := rows.Scan(&configJSON); err != nil {
			continue
		}

		var config WorkspaceConfig
		if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
			continue
		}
		configs = append(configs, config)
	}

	return configs, nil
}

// ArchiveWorkspaceConfig creates a historical snapshot of the current config
func ArchiveWorkspaceConfig(ctx context.Context, workspaceID int) error {
	config, err := GetWorkspaceConfig(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	jsonBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	_, err = db.DB.ExecContext(ctx, `
		INSERT INTO workspace_configurations_history (workspace_id, config_data, archived_at)
		VALUES (?, ?, ?)
	`, workspaceID, string(jsonBytes), time.Now())

	if err != nil {
		return fmt.Errorf("failed to archive config: %w", err)
	}

	return nil
}

// RestoreWorkspaceConfig restores a configuration from history
func RestoreWorkspaceConfig(ctx context.Context, workspaceID int, version int) error {
	var configJSON string
	err := db.DB.QueryRowContext(ctx, `
		SELECT config_data FROM workspace_configurations_history
		WHERE workspace_id = ? AND version = ?
	`, workspaceID, version).Scan(&configJSON)

	if err == sql.ErrNoRows {
		return fmt.Errorf("config version not found")
	}
	if err != nil {
		return fmt.Errorf("failed to query config: %w", err)
	}

	var config WorkspaceConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return SaveWorkspaceConfig(ctx, workspaceID, &config)
}

// getDefaultConfig returns a default workspace configuration
func getDefaultConfig() *WorkspaceConfig {
	return &WorkspaceConfig{
		Provider: "podman",
		Resources: ResourceConfig{
			CPU:       2,
			MemoryGB:  4,
			StorageGB: 20,
		},
		EditorConfig: EditorConfig{
			Type: "code-server",
			Port: 8080,
		},
		NetworkConfig: NetworkConfig{
			NetworkMode: "bridge",
		},
		StorageConfig: StorageConfig{
			HomeDirSizeGB:   10,
			DataDirSizeGB:   10,
			SnapshotEnabled: true,
		},
		EnvVariables: make(map[string]string),
		CustomConfig: make(map[string]interface{}),
		UpdatedAt:    time.Now(),
	}
}
