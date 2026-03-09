package provider

import (
	"context"
)

// Provider defines the interface for workspace provisioning providers
type Provider interface {
	// CreateWorkspace provisions a new workspace
	CreateWorkspace(ctx context.Context, config *WorkspaceConfig) (*WorkspaceInfo, error)

	// GetWorkspace retrieves workspace status and information
	GetWorkspace(ctx context.Context, workspaceID string) (*WorkspaceInfo, error)

	// UpdateWorkspace updates workspace configuration
	UpdateWorkspace(ctx context.Context, workspaceID string, config *WorkspaceConfig) (*WorkspaceInfo, error)

	// DeleteWorkspace removes a workspace and its resources
	DeleteWorkspace(ctx context.Context, workspaceID string) error

	// StartWorkspace starts a stopped workspace
	StartWorkspace(ctx context.Context, workspaceID string) (*WorkspaceInfo, error)

	// StopWorkspace stops a running workspace
	StopWorkspace(ctx context.Context, workspaceID string) (*WorkspaceInfo, error)

	// RestartWorkspace restarts a workspace
	RestartWorkspace(ctx context.Context, workspaceID string) (*WorkspaceInfo, error)

	// GetWorkspaceStatus retrieves the current status of a workspace
	GetWorkspaceStatus(ctx context.Context, workspaceID string) (string, error)

	// GetWorkspaceResources returns resource usage information
	GetWorkspaceResources(ctx context.Context, workspaceID string) (*ResourceInfo, error)

	// Validate checks if the provider is properly configured
	Validate(ctx context.Context) error
}

// WorkspaceConfig contains configuration for workspace provisioning
type WorkspaceConfig struct {
	WorkspaceID     string
	OrganizationID  int
	UserID          int
	Name            string
	Tag             string
	Region          string
	CPU             int
	Memory          int
	Storage         int
	Languages       []string
	BootstrapScript string
	GithubToken     string
}

// WorkspaceInfo contains information about a provisioned workspace
type WorkspaceInfo struct {
	WorkspaceID string
	Tag         string
	Name        string
	Status      string
	Provider    string
	Region      string
	Endpoint    string
	SSHHost     string
	SSHPort     int
	HTTPHost    string
	HTTPPort    int
	CreatedAt   string
	UpdatedAt   string
}

// ResourceInfo contains resource usage information
type ResourceInfo struct {
	CPUUsage    float64 // percentage
	MemoryUsage int64   // bytes
	StorageUsed int64   // bytes
	NetworkIn   int64   // bytes
	NetworkOut  int64   // bytes
}
