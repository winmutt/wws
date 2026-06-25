package mock

import (
	"context"
	"os"
	"time"

	"wws/api/internal/db"
	"wws/api/provisioner/provider"
)

// MockProvider is a test provider that works without Podman/Docker
type MockProvider struct {
	workspaces map[string]*provider.WorkspaceInfo
}

// NewMockProvider creates a new MockProvider
func NewMockProvider() *MockProvider {
	if os.Getenv("DEBUG_SKIP_AUTH") == "true" {
		return &MockProvider{
			workspaces: make(map[string]*provider.WorkspaceInfo),
		}
	}
	return nil
}

// CreateWorkspace creates a mock workspace
func (p *MockProvider) CreateWorkspace(ctx context.Context, config *provider.WorkspaceConfig) (*provider.WorkspaceInfo, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	info := &provider.WorkspaceInfo{
		WorkspaceID: config.Tag,
		Tag:         config.Tag,
		Name:        config.Name,
		Status:      "running",
		Provider:    "mock",
		Region:      config.Region,
		Endpoint:    "http://localhost:8080",
		SSHHost:     "localhost",
		SSHPort:     2222,
		HTTPHost:    "localhost",
		HTTPPort:    8080,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	p.workspaces[config.Tag] = info
	return info, nil
}

// GetWorkspace returns mock workspace info
func (p *MockProvider) GetWorkspace(ctx context.Context, workspaceID string) (*provider.WorkspaceInfo, error) {
	// Check in-memory map first
	if info, ok := p.workspaces[workspaceID]; ok {
		return info, nil
	}

	// Fallback to database lookup
	if db.DB != nil {
		var tag, name, status string
		var orgID, ownerID int
		err := db.DB.QueryRowContext(ctx,
			"SELECT tag, name, organization_id, owner_id, status FROM workspaces WHERE tag = ? AND deleted_at IS NULL",
			workspaceID,
		).Scan(&tag, &name, &orgID, &ownerID, &status)

		if err == nil {
			return &provider.WorkspaceInfo{
				WorkspaceID: workspaceID,
				Tag:         tag,
				Name:        name,
				Status:      status,
				Provider:    "mock",
				SSHHost:     "localhost",
				SSHPort:     2222,
				HTTPHost:    "localhost",
				HTTPPort:    8080,
			}, nil
		}
	}

	return nil, provider.ErrWorkspaceNotFound
}

// UpdateWorkspace updates mock workspace
func (p *MockProvider) UpdateWorkspace(ctx context.Context, workspaceID string, config *provider.WorkspaceConfig) (*provider.WorkspaceInfo, error) {
	if info, ok := p.workspaces[workspaceID]; ok {
		info.Name = config.Name
		info.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return info, nil
	}

	// Check database
	if db.DB != nil {
		var tag, name string
		var orgID, ownerID int
		var status string
		err := db.DB.QueryRowContext(ctx,
			"SELECT tag, name, organization_id, owner_id, status FROM workspaces WHERE tag = ? AND deleted_at IS NULL",
			workspaceID,
		).Scan(&tag, &name, &orgID, &ownerID, &status)

		if err == nil {
			// Update database
			_, err := db.DB.ExecContext(ctx,
				"UPDATE workspaces SET name = ?, status = ?, updated_at = datetime('now') WHERE tag = ?",
				config.Name, status, workspaceID,
			)
			if err == nil {
				return &provider.WorkspaceInfo{
					WorkspaceID: workspaceID,
					Tag:         tag,
					Name:        config.Name,
					Status:      status,
					Provider:    "mock",
					UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
				}, nil
			}
		}
	}

	return nil, provider.ErrWorkspaceNotFound
}

// DeleteWorkspace removes mock workspace
func (p *MockProvider) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	delete(p.workspaces, workspaceID)

	// Also soft delete from database
	if db.DB != nil {
		_, err := db.DB.ExecContext(ctx,
			"UPDATE workspaces SET deleted_at = datetime('now'), status = 'deleted' WHERE tag = ?",
			workspaceID,
		)
		return err
	}
	return nil
}

// StartWorkspace starts mock workspace
func (p *MockProvider) StartWorkspace(ctx context.Context, workspaceID string) (*provider.WorkspaceInfo, error) {
	if info, ok := p.workspaces[workspaceID]; ok {
		info.Status = "running"
		return info, nil
	}

	// Check database
	if db.DB != nil {
		var tag, name string
		var orgID, ownerID int
		err := db.DB.QueryRowContext(ctx,
			"SELECT tag, name, organization_id, owner_id FROM workspaces WHERE tag = ? AND deleted_at IS NULL",
			workspaceID,
		).Scan(&tag, &name, &orgID, &ownerID)

		if err == nil {
			_, err := db.DB.ExecContext(ctx,
				"UPDATE workspaces SET status = 'running', updated_at = datetime('now') WHERE tag = ?",
				workspaceID,
			)
			if err == nil {
				return &provider.WorkspaceInfo{
					WorkspaceID: workspaceID,
					Tag:         tag,
					Name:        name,
					Status:      "running",
					Provider:    "mock",
				}, nil
			}
		}
	}

	return nil, provider.ErrWorkspaceNotFound
}

// StopWorkspace stops mock workspace
func (p *MockProvider) StopWorkspace(ctx context.Context, workspaceID string) (*provider.WorkspaceInfo, error) {
	if info, ok := p.workspaces[workspaceID]; ok {
		info.Status = "stopped"
		return info, nil
	}

	// Check database
	if db.DB != nil {
		var tag, name string
		var orgID, ownerID int
		err := db.DB.QueryRowContext(ctx,
			"SELECT tag, name, organization_id, owner_id FROM workspaces WHERE tag = ? AND deleted_at IS NULL",
			workspaceID,
		).Scan(&tag, &name, &orgID, &ownerID)

		if err == nil {
			_, err := db.DB.ExecContext(ctx,
				"UPDATE workspaces SET status = 'stopped', updated_at = datetime('now') WHERE tag = ?",
				workspaceID,
			)
			if err == nil {
				return &provider.WorkspaceInfo{
					WorkspaceID: workspaceID,
					Tag:         tag,
					Name:        name,
					Status:      "stopped",
					Provider:    "mock",
				}, nil
			}
		}
	}

	return nil, provider.ErrWorkspaceNotFound
}

// RestartWorkspace restarts mock workspace
func (p *MockProvider) RestartWorkspace(ctx context.Context, workspaceID string) (*provider.WorkspaceInfo, error) {
	if info, ok := p.workspaces[workspaceID]; ok {
		info.Status = "running"
		return info, nil
	}

	// Check database
	if db.DB != nil {
		var tag, name string
		var orgID, ownerID int
		err := db.DB.QueryRowContext(ctx,
			"SELECT tag, name, organization_id, owner_id FROM workspaces WHERE tag = ? AND deleted_at IS NULL",
			workspaceID,
		).Scan(&tag, &name, &orgID, &ownerID)

		if err == nil {
			_, err := db.DB.ExecContext(ctx,
				"UPDATE workspaces SET status = 'running', updated_at = datetime('now') WHERE tag = ?",
				workspaceID,
			)
			if err == nil {
				return &provider.WorkspaceInfo{
					WorkspaceID: workspaceID,
					Tag:         tag,
					Name:        name,
					Status:      "running",
					Provider:    "mock",
				}, nil
			}
		}
	}

	return nil, provider.ErrWorkspaceNotFound
}

// GetWorkspaceStatus returns mock workspace status
func (p *MockProvider) GetWorkspaceStatus(ctx context.Context, workspaceID string) (string, error) {
	if info, ok := p.workspaces[workspaceID]; ok {
		return info.Status, nil
	}
	return "", provider.ErrWorkspaceNotFound
}

// GetWorkspaceResources returns mock resource info
func (p *MockProvider) GetWorkspaceResources(ctx context.Context, workspaceID string) (*provider.ResourceInfo, error) {
	if _, ok := p.workspaces[workspaceID]; ok {
		return &provider.ResourceInfo{
			CPUUsage:    0.5,
			MemoryUsage: 512 * 1024 * 1024,
			StorageUsed: 1 * 1024 * 1024 * 1024,
		}, nil
	}
	return nil, provider.ErrWorkspaceNotFound
}

// Validate checks if mock provider is configured
func (p *MockProvider) Validate(ctx context.Context) error {
	return nil
}
