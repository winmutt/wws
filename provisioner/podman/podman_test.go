package podman

import (
	"context"
	"testing"

	"wws/provisioner/provider"
)

func TestNewPodmanProvider(t *testing.T) {
	p := NewPodmanProvider("")
	if p == nil {
		t.Fatal("Expected non-nil PodmanProvider")
	}
	if p.socketPath != "/run/podman/podman.sock" {
		t.Errorf("Expected default socket path, got %s", p.socketPath)
	}
}

func TestNewPodmanProviderCustom(t *testing.T) {
	customPath := "/custom/socket"
	p := NewPodmanProvider(customPath)
	if p.socketPath != customPath {
		t.Errorf("Expected %s, got %s", customPath, p.socketPath)
	}
}

func TestPodmanProviderImplementsProvider(t *testing.T) {
	var _ provider.Provider = (*PodmanProvider)(nil)
}

func TestPodmanProviderValidate(t *testing.T) {
	p := NewPodmanProvider("")
	ctx := context.Background()

	// This test will fail if podman is not available
	// It's meant to be run in an environment with podman
	err := p.Validate(ctx)
	if err != nil {
		t.Logf("Podman validation failed (expected if podman not available): %v", err)
	}
}

func TestPodmanProviderCreateWorkspace(t *testing.T) {
	p := NewPodmanProvider("")
	ctx := context.Background()

	config := &provider.WorkspaceConfig{
		WorkspaceID:    "test-123",
		OrganizationID: 1,
		UserID:         1,
		Name:           "Test Workspace",
		Tag:            "test-123",
		CPU:            2,
		Memory:         2048,
		Storage:        10,
	}

	// This test will fail if podman is not available
	info, err := p.CreateWorkspace(ctx, config)
	if err != nil {
		t.Logf("CreateWorkspace failed (expected if podman not available): %v", err)
		return
	}

	if info == nil {
		t.Fatal("Expected non-nil WorkspaceInfo")
	}
	if info.WorkspaceID != "test-123" {
		t.Errorf("Expected workspace ID test-123, got %s", info.WorkspaceID)
	}
	if info.Provider != "podman" {
		t.Errorf("Expected provider podman, got %s", info.Provider)
	}
}

func TestPodmanProviderGetWorkspace(t *testing.T) {
	p := NewPodmanProvider("")
	ctx := context.Background()

	_, err := p.GetWorkspace(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent workspace")
	}
}

func TestPodmanProviderDeleteWorkspace(t *testing.T) {
	p := NewPodmanProvider("")
	ctx := context.Background()

	err := p.DeleteWorkspace(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent workspace")
	}
}

func TestPodmanProviderStartWorkspace(t *testing.T) {
	p := NewPodmanProvider("")
	ctx := context.Background()

	_, err := p.StartWorkspace(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent workspace")
	}
}

func TestPodmanProviderStopWorkspace(t *testing.T) {
	p := NewPodmanProvider("")
	ctx := context.Background()

	_, err := p.StopWorkspace(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent workspace")
	}
}

func TestPodmanProviderRestartWorkspace(t *testing.T) {
	p := NewPodmanProvider("")
	ctx := context.Background()

	_, err := p.RestartWorkspace(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent workspace")
	}
}

func TestPodmanProviderGetWorkspaceStatus(t *testing.T) {
	p := NewPodmanProvider("")
	ctx := context.Background()

	_, err := p.GetWorkspaceStatus(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent workspace")
	}
}

func TestPodmanProviderGetWorkspaceResources(t *testing.T) {
	p := NewPodmanProvider("")
	ctx := context.Background()

	_, err := p.GetWorkspaceResources(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent workspace")
	}
}
