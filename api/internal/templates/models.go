package templates

import "time"

// WorkspaceTemplate represents a reusable workspace template
type WorkspaceTemplate struct {
	ID              int              `db:"id" json:"id"`
	Name            string           `db:"name" json:"name"`
	Description     string           `db:"description" json:"description"`
	OrganizationID  *int             `db:"organization_id" json:"organization_id,omitempty"` // nil for global templates
	Provider        string           `db:"provider" json:"provider"`
	LanguageConfig  []LanguageConfig `db:"-" json:"language_config"`
	EnvVariables    []EnvVariable    `db:"-" json:"env_variables"`
	Resources       ResourceConfig   `db:"resources" json:"resources"`
	EditorConfig    EditorConfig     `db:"-" json:"editor_config"`
	BootstrapScript string           `db:"bootstrap_script" json:"bootstrap_script"`
	IsPublic        bool             `db:"is_public" json:"is_public"`
	CreatedBy       int              `db:"created_by" json:"created_by"`
	CreatedAt       time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time        `db:"updated_at" json:"updated_at"`
}

// LanguageConfig represents language configuration for a template
type LanguageConfig struct {
	Language string `json:"language"`
	Version  string `json:"version"`
}

// EnvVariable represents an environment variable
type EnvVariable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ResourceConfig represents resource configuration for a workspace
type ResourceConfig struct {
	CPU       int     `json:"cpu"`
	MemoryGB  float64 `json:"memory_gb"`
	StorageGB int     `json:"storage_gb"`
}

// EditorConfig represents editor configuration
type EditorConfig struct {
	Type    string            `json:"type"`    // "code-server", "cursor", "vim", etc.
	Plugins []string          `json:"plugins"` // List of plugin IDs
	Config  map[string]string `json:"config"`  // Editor-specific configuration
}

// TemplateCreateRequest represents a request to create a template
type TemplateCreateRequest struct {
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	Provider        string           `json:"provider"`
	LanguageConfig  []LanguageConfig `json:"language_config"`
	EnvVariables    []EnvVariable    `json:"env_variables"`
	CPU             int              `json:"cpu"`
	MemoryGB        float64          `json:"memory_gb"`
	StorageGB       int              `json:"storage_gb"`
	EditorType      string           `json:"editor_type"`
	EditorPlugins   []string         `json:"editor_plugins"`
	BootstrapScript string           `json:"bootstrap_script"`
	IsPublic        bool             `json:"is_public"`
}

// TemplateUpdateRequest represents a request to update a template
type TemplateUpdateRequest struct {
	Name            *string          `json:"name,omitempty"`
	Description     *string          `json:"description,omitempty"`
	LanguageConfig  []LanguageConfig `json:"language_config"`
	EnvVariables    []EnvVariable    `json:"env_variables"`
	CPU             *int             `json:"cpu,omitempty"`
	MemoryGB        *float64         `json:"memory_gb,omitempty"`
	StorageGB       *int             `json:"storage_gb,omitempty"`
	EditorType      *string          `json:"editor_type,omitempty"`
	EditorPlugins   []string         `json:"editor_plugins"`
	BootstrapScript *string          `json:"bootstrap_script,omitempty"`
	IsPublic        *bool            `json:"is_public,omitempty"`
}

// TemplateUsage represents usage statistics for a template
type TemplateUsage struct {
	TemplateID       int       `json:"template_id"`
	TemplateName     string    `json:"template_name"`
	UsageCount       int       `json:"usage_count"`
	LastUsedAt       time.Time `json:"last_used_at"`
	ActiveWorkspaces int       `json:"active_workspaces"`
}

// PopularTemplate represents a popular template with usage stats
type PopularTemplate struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	UsageCount  int      `json:"usage_count"`
	Rating      float64  `json:"rating"`
	Tags        []string `json:"tags"`
}
