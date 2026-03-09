package podman

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"wws/api/provisioner/provider"
)

// PodmanProvider implements the provider.Provider interface for Podman containers
type PodmanProvider struct {
	socketPath string
	podPrefix  string
}

// NewPodmanProvider creates a new Podman provider instance
func NewPodmanProvider(socketPath string) *PodmanProvider {
	if socketPath == "" {
		socketPath = "/run/podman/podman.sock"
	}
	return &PodmanProvider{
		socketPath: socketPath,
		podPrefix:  "wws-",
	}
}

// podmanCmd executes a podman command with the socket
func (p *PodmanProvider) podmanCmd(ctx context.Context, args ...string) ([]byte, error) {
	fullArgs := append([]string{"--url", p.socketPath}, args...)
	cmd := exec.CommandContext(ctx, "podman", fullArgs...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("podman command failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}
	return output, nil
}

// CreateWorkspace provisions a new workspace container
func (p *PodmanProvider) CreateWorkspace(ctx context.Context, config *provider.WorkspaceConfig) (*provider.WorkspaceInfo, error) {
	podName := fmt.Sprintf("%s%s", p.podPrefix, config.Tag)

	// Create pod with port mappings
	createArgs := []string{
		"pod", "create",
		"--name", podName,
		"--publish", "8080",
		"--publish", "2222",
		"--label", fmt.Sprintf("wws.workspace_id=%s", config.WorkspaceID),
		"--label", fmt.Sprintf("wws.organization_id=%d", config.OrganizationID),
		"--label", fmt.Sprintf("wws.user_id=%d", config.UserID),
		"--label", fmt.Sprintf("wws.tag=%s", config.Tag),
	}

	_, err := p.podmanCmd(ctx, createArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to create pod: %w", err)
	}

	// Pull and run the workspace image
	imageName := "ghcr.io/winmutt/wws/workspace:latest"
	runArgs := []string{
		"run", "-d",
		"--pod", podName,
		"--name", fmt.Sprintf("%s-main", podName),
		"--env", fmt.Sprintf("WORKSPACE_ID=%s", config.WorkspaceID),
		"--env", fmt.Sprintf("ORGANIZATION_ID=%d", config.OrganizationID),
		"--env", fmt.Sprintf("USER_ID=%d", config.UserID),
	}

	// Add resource limits
	if config.CPU > 0 {
		runArgs = append(runArgs, "--cpus", fmt.Sprintf("%d", config.CPU))
	}
	if config.Memory > 0 {
		runArgs = append(runArgs, "--memory", fmt.Sprintf("%dm", config.Memory))
	}
	if config.Storage > 0 {
		runArgs = append(runArgs, "--storage-opt", fmt.Sprintf("size=%dG", config.Storage))
	}

	runArgs = append(runArgs, imageName)

	_, err = p.podmanCmd(ctx, runArgs...)
	if err != nil {
		// Cleanup on failure
		p.podmanCmd(context.Background(), "pod", "rm", "-f", podName)
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Wait for container to be ready
	time.Sleep(2 * time.Second)

	// Get port mappings
	info, err := p.GetWorkspace(ctx, config.WorkspaceID)
	if err != nil {
		return nil, err
	}

	return info, nil
}

// GetWorkspace retrieves workspace information
func (p *PodmanProvider) GetWorkspace(ctx context.Context, workspaceID string) (*provider.WorkspaceInfo, error) {
	podName := fmt.Sprintf("%s%s", p.podPrefix, workspaceID)

	// Get pod inspect data
	output, err := p.podmanCmd(ctx, "pod", "inspect", podName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect pod: %w", err)
	}

	var podInfo map[string]interface{}
	if err := json.Unmarshal(output, &podInfo); err != nil {
		return nil, fmt.Errorf("failed to parse pod info: %w", err)
	}

	// Extract status
	status := "unknown"
	if state, ok := podInfo["State"].(map[string]interface{}); ok {
		if statusRaw, ok := state["Status"].(string); ok {
			status = statusRaw
		}
	}

	// Extract ports
	ports := []int{}
	if hostConfig, ok := podInfo["HostConfig"].(map[string]interface{}); ok {
		if portBindings, ok := hostConfig["PortBindings"].(map[string]interface{}); ok {
			for key := range portBindings {
				parts := strings.Split(key, "/")
				var port int
				fmt.Sscanf(parts[0], "%d", &port)
				if port > 0 {
					ports = append(ports, port)
				}
			}
		}
	}

	return &provider.WorkspaceInfo{
		WorkspaceID: workspaceID,
		Tag:         workspaceID,
		Status:      status,
		Provider:    "podman",
		SSHPort:     2222,
		HTTPPort:    8080,
		SSHHost:     "localhost",
		HTTPHost:    "localhost",
	}, nil
}

// UpdateWorkspace updates workspace configuration
func (p *PodmanProvider) UpdateWorkspace(ctx context.Context, workspaceID string, config *provider.WorkspaceConfig) (*provider.WorkspaceInfo, error) {
	// Stop the container
	if _, err := p.StopWorkspace(ctx, workspaceID); err != nil {
		return nil, err
	}

	// Remove old container
	podName := fmt.Sprintf("%s%s-main", p.podPrefix, workspaceID)
	p.podmanCmd(ctx, "container", "rm", podName)

	// Create new container with updated config
	_, err := p.CreateWorkspace(ctx, config)
	if err != nil {
		return nil, err
	}

	return p.GetWorkspace(ctx, workspaceID)
}

// DeleteWorkspace removes the workspace pod and all containers
func (p *PodmanProvider) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	podName := fmt.Sprintf("%s%s", p.podPrefix, workspaceID)

	// Force remove pod (includes all containers)
	_, err := p.podmanCmd(ctx, "pod", "rm", "-f", podName)
	if err != nil {
		return fmt.Errorf("failed to remove pod: %w", err)
	}

	return nil
}

// StartWorkspace starts a stopped workspace
func (p *PodmanProvider) StartWorkspace(ctx context.Context, workspaceID string) (*provider.WorkspaceInfo, error) {
	podName := fmt.Sprintf("%s%s", p.podPrefix, workspaceID)

	_, err := p.podmanCmd(ctx, "pod", "start", podName)
	if err != nil {
		return nil, fmt.Errorf("failed to start pod: %w", err)
	}

	return p.GetWorkspace(ctx, workspaceID)
}

// StopWorkspace stops a running workspace
func (p *PodmanProvider) StopWorkspace(ctx context.Context, workspaceID string) (*provider.WorkspaceInfo, error) {
	podName := fmt.Sprintf("%s%s", p.podPrefix, workspaceID)

	_, err := p.podmanCmd(ctx, "pod", "stop", podName)
	if err != nil {
		return nil, fmt.Errorf("failed to stop pod: %w", err)
	}

	return p.GetWorkspace(ctx, workspaceID)
}

// RestartWorkspace restarts a workspace
func (p *PodmanProvider) RestartWorkspace(ctx context.Context, workspaceID string) (*provider.WorkspaceInfo, error) {
	podName := fmt.Sprintf("%s%s", p.podPrefix, workspaceID)

	_, err := p.podmanCmd(ctx, "pod", "restart", podName)
	if err != nil {
		return nil, fmt.Errorf("failed to restart pod: %w", err)
	}

	return p.GetWorkspace(ctx, workspaceID)
}

// GetWorkspaceStatus retrieves the current status of a workspace
func (p *PodmanProvider) GetWorkspaceStatus(ctx context.Context, workspaceID string) (string, error) {
	info, err := p.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return "", err
	}
	return info.Status, nil
}

// GetWorkspaceResources returns resource usage information
func (p *PodmanProvider) GetWorkspaceResources(ctx context.Context, workspaceID string) (*provider.ResourceInfo, error) {
	podName := fmt.Sprintf("%s%s", p.podPrefix, workspaceID)

	// Get container stats
	output, err := p.podmanCmd(ctx, "pod", "stats", "--no-stream", podName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	// Parse stats output
	stats := &provider.ResourceInfo{
		CPUUsage:    0,
		MemoryUsage: 0,
		StorageUsed: 0,
		NetworkIn:   0,
		NetworkOut:  0,
	}

	// Simple parsing - in production, use proper JSON stats
	log.Printf("Podman stats for %s: %s", podName, string(output))

	return stats, nil
}

// Validate checks if Podman is properly configured
func (p *PodmanProvider) Validate(ctx context.Context) error {
	_, err := p.podmanCmd(ctx, "info")
	if err != nil {
		return fmt.Errorf("podman is not available: %w", err)
	}
	return nil
}
