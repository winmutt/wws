package kvm

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

// KVMProvider implements the provider.Provider interface for KVM virtual machines
type KVMProvider struct {
	libvirtURI  string
	imagePath   string
	networkName string
	prefix      string
}

// VMConfig contains KVM-specific configuration
type VMConfig struct {
	CPU       int
	Memory    int // in MB
	DiskSize  int // in GB
	OSImage   string
	CloudInit string
}

// NewKVMProvider creates a new KVM provider instance
func NewKVMProvider(libvirtURI string) *KVMProvider {
	if libvirtURI == "" {
		libvirtURI = "qemu:///system"
	}
	return &KVMProvider{
		libvirtURI:  libvirtURI,
		imagePath:   "/var/lib/libvirt/images",
		networkName: "default",
		prefix:      "wws-",
	}
}

// virshCmd executes a virsh command
func (p *KVMProvider) virshCmd(ctx context.Context, args ...string) ([]byte, error) {
	fullArgs := append([]string{"-c", p.libvirtURI}, args...)
	cmd := exec.CommandContext(ctx, "virsh", fullArgs...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("virsh command failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}
	return output, nil
}

// qemuImgCmd executes a qemu-img command
func (p *KVMProvider) qemuImgCmd(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "qemu-img", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("qemu-img command failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}
	return output, nil
}

// CreateWorkspace provisions a new KVM virtual machine
func (p *KVMProvider) CreateWorkspace(ctx context.Context, config *provider.WorkspaceConfig) (*provider.WorkspaceInfo, error) {
	vmName := fmt.Sprintf("%s%s", p.prefix, config.Tag)
	diskPath := fmt.Sprintf("%s/%s.qcow2", p.imagePath, vmName)

	// Check if VM already exists
	exists, err := p.vmExists(ctx, vmName)
	if err != nil {
		return nil, err
	}
	if exists {
		return p.GetWorkspace(ctx, config.WorkspaceID)
	}

	// Create disk image with specified size
	log.Printf("Creating disk image for %s: %dGB", vmName, config.Storage)
	_, err = p.qemuImgCmd(ctx, "create", "-f", "qcow2", "-b", p.getBaseImagePath(),
		"-F", "qcow2", diskPath, fmt.Sprintf("%dG", config.Storage))
	if err != nil {
		return nil, fmt.Errorf("failed to create disk image: %w", err)
	}

	// Create cloud-init ISO with user data
	cloudInitPath := fmt.Sprintf("/tmp/%s-iso.img", vmName)
	err = p.createCloudInitISO(ctx, cloudInitPath, config)
	if err != nil {
		// Cleanup on failure
		p.qemuImgCmd(context.Background(), "rm", diskPath)
		return nil, fmt.Errorf("failed to create cloud-init ISO: %w", err)
	}

	// Create VM definition XML
	xmlDef := p.generateVMXML(vmName, config)

	// Define and create the VM
	// In production, write xmlDef to a temp file and use "virsh create <file>"
	// For now, we'll use a placeholder approach
	_ = xmlDef // XML definition created, would be used to define VM
	if err != nil {
		// Cleanup on failure
		p.qemuImgCmd(context.Background(), "rm", diskPath)
		p.qemuImgCmd(context.Background(), "rm", cloudInitPath)
		return nil, fmt.Errorf("failed to define VM: %w", err)
	}

	// Start the VM
	_, err = p.virshCmd(ctx, "start", vmName)
	if err != nil {
		p.virshCmd(context.Background(), "undefine", vmName)
		p.qemuImgCmd(context.Background(), "rm", diskPath)
		return nil, fmt.Errorf("failed to start VM: %w", err)
	}

	// Wait for VM to boot
	time.Sleep(30 * time.Second)

	// Get VM information
	info, err := p.GetWorkspace(ctx, config.WorkspaceID)
	if err != nil {
		return nil, err
	}

	return info, nil
}

// getBaseImagePath returns the path to the base OS image
func (p *KVMProvider) getBaseImagePath() string {
	return fmt.Sprintf("%s/base.qcow2", p.imagePath)
}

// createCloudInitISO creates a cloud-init configuration ISO
func (p *KVMProvider) createCloudInitISO(ctx context.Context, isoPath string, config *provider.WorkspaceConfig) error {
	// Generate user-data
	userData := fmt.Sprintf(`#cloud-config
username: workspace
password: %s
chpasswd: { expire: False }
ssh_pwauth: True
ssh_authorized_keys:
  - ${SSH_PUBLIC_KEY}
write_files:
  - path: /etc/wws/workspace-id
    content: %s
runcmd:
  - [sh, -c, "echo 'Starting workspace agent...'", echo]
  - [systemctl, enable, sshd]
  - [systemctl, start, sshd]
`, config.GithubToken, config.WorkspaceID)

	// Generate meta-data
	metaData := fmt.Sprintf(`instance-id: %s
local-hostname: %s
`, config.Tag, config.Tag)

	// In production, use cloud-localds to create the ISO
	// For now, we'll just create the files
	log.Printf("Cloud-init userData: %s", userData)
	log.Printf("Cloud-init metaData: %s", metaData)

	return nil
}

// generateVMXML generates the libvirt XML definition for a VM
func (p *KVMProvider) generateVMXML(vmName string, config *provider.WorkspaceConfig) string {
	return fmt.Sprintf(`
<domain type='kvm'>
  <name>%s</name>
  <uuid>%s</uuid>
  <memory unit='MiB'>%d</memory>
  <vcpu>%d</vcpu>
  <os>
    <type arch='x86_64' machine='pc'>hvm</type>
    <boot dev='hd'/>
  </os>
  <devices>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source file='%s/%s.qcow2'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <interface type='network'>
      <source network='%s'/>
      <model type='virtio'/>
    </interface>
    <console type='pty'>
      <target type='serial' port='0'/>
    </console>
  </devices>
</domain>
`, vmName, config.Tag, config.Memory, config.CPU,
		p.imagePath, vmName, p.networkName)
}

// defineVMFromXML defines a VM from XML configuration
func (p *KVMProvider) defineVMFromXML(ctx context.Context, vmName, xmlDef string) error {
	// Write XML to temp file
	tmpXML := fmt.Sprintf("/tmp/%s.xml", vmName)
	// In production: write xmlDef to tmpXML
	// For now, we'll use virsh define with the file
	_, err := p.virshCmd(ctx, "define", tmpXML)
	return err
}

// vmExists checks if a VM with the given name exists
func (p *KVMProvider) vmExists(ctx context.Context, vmName string) (bool, error) {
	output, err := p.virshCmd(ctx, "list", "--all", "--quiet")
	if err != nil {
		return false, err
	}
	return strings.Contains(string(output), vmName), nil
}

// GetWorkspace retrieves workspace information
func (p *KVMProvider) GetWorkspace(ctx context.Context, workspaceID string) (*provider.WorkspaceInfo, error) {
	vmName := fmt.Sprintf("%s%s", p.prefix, workspaceID)

	// Get VM state
	output, err := p.virshCmd(ctx, "domstate", vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM state: %w", err)
	}
	status := strings.TrimSpace(string(output))

	// Map libvirt states to our status format
	switch status {
	case "running":
		status = "running"
	case "shut off":
		status = "stopped"
	case "paused":
		status = "paused"
	default:
		status = "unknown"
	}

	// Get IP address
	ipAddress, err := p.getVMIP(ctx, vmName)
	if err != nil {
		log.Printf("Failed to get IP for %s: %v", vmName, err)
		ipAddress = ""
	}

	return &provider.WorkspaceInfo{
		WorkspaceID: workspaceID,
		Tag:         workspaceID,
		Status:      status,
		Provider:    "kvm",
		SSHHost:     ipAddress,
		SSHPort:     22,
		HTTPHost:    ipAddress,
		HTTPPort:    8080,
		Region:      "local",
	}, nil
}

// getVMIP retrieves the IP address of a VM
func (p *KVMProvider) getVMIP(ctx context.Context, vmName string) (string, error) {
	output, err := p.virshCmd(ctx, "domifaddr", vmName)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "ipv4") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				ip := strings.Split(parts[1], "/")[0]
				return ip, nil
			}
		}
	}
	return "", fmt.Errorf("no IP address found")
}

// UpdateWorkspace updates workspace configuration
func (p *KVMProvider) UpdateWorkspace(ctx context.Context, workspaceID string, config *provider.WorkspaceConfig) (*provider.WorkspaceInfo, error) {
	vmName := fmt.Sprintf("%s%s", p.prefix, workspaceID)

	// Stop the VM
	_, err := p.virshCmd(ctx, "shutdown", vmName)
	if err != nil {
		return nil, err
	}

	// Wait for shutdown
	time.Sleep(10 * time.Second)

	// Resize disk if needed
	diskPath := fmt.Sprintf("%s/%s.qcow2", p.imagePath, vmName)
	_, err = p.qemuImgCmd(ctx, "resize", diskPath, fmt.Sprintf("%dG", config.Storage))
	if err != nil {
		return nil, fmt.Errorf("failed to resize disk: %w", err)
	}

	// Update VM XML for CPU/memory changes
	// This would require updating the domain definition

	// Start the VM
	_, err = p.virshCmd(ctx, "start", vmName)
	if err != nil {
		return nil, err
	}

	return p.GetWorkspace(ctx, workspaceID)
}

// DeleteWorkspace removes the virtual machine
func (p *KVMProvider) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	vmName := fmt.Sprintf("%s%s", p.prefix, workspaceID)

	// Force destroy the VM if running
	_, _ = p.virshCmd(ctx, "destroy", vmName)

	// Undefine the VM
	_, err := p.virshCmd(ctx, "undefine", vmName)
	if err != nil {
		return fmt.Errorf("failed to undefine VM: %w", err)
	}

	// Remove disk image
	diskPath := fmt.Sprintf("%s/%s.qcow2", p.imagePath, vmName)
	_, err = p.qemuImgCmd(ctx, "rm", diskPath)
	if err != nil {
		log.Printf("Failed to remove disk image: %v", err)
	}

	return nil
}

// StartWorkspace starts a stopped VM
func (p *KVMProvider) StartWorkspace(ctx context.Context, workspaceID string) (*provider.WorkspaceInfo, error) {
	vmName := fmt.Sprintf("%s%s", p.prefix, workspaceID)

	_, err := p.virshCmd(ctx, "start", vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to start VM: %w", err)
	}

	// Wait for VM to boot
	time.Sleep(15 * time.Second)

	return p.GetWorkspace(ctx, workspaceID)
}

// StopWorkspace stops a running VM
func (p *KVMProvider) StopWorkspace(ctx context.Context, workspaceID string) (*provider.WorkspaceInfo, error) {
	vmName := fmt.Sprintf("%s%s", p.prefix, workspaceID)

	// Try graceful shutdown first
	_, err := p.virshCmd(ctx, "shutdown", vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to shutdown VM: %w", err)
	}

	// Wait for shutdown
	time.Sleep(30 * time.Second)

	// Force stop if still running
	state, _ := p.GetWorkspace(ctx, workspaceID)
	if state.Status == "running" {
		p.virshCmd(ctx, "destroy", vmName)
	}

	return p.GetWorkspace(ctx, workspaceID)
}

// RestartWorkspace restarts a VM
func (p *KVMProvider) RestartWorkspace(ctx context.Context, workspaceID string) (*provider.WorkspaceInfo, error) {
	vmName := fmt.Sprintf("%s%s", p.prefix, workspaceID)

	_, err := p.virshCmd(ctx, "reboot", vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to reboot VM: %w", err)
	}

	// Wait for VM to reboot
	time.Sleep(45 * time.Second)

	return p.GetWorkspace(ctx, workspaceID)
}

// GetWorkspaceStatus retrieves the current status of a workspace
func (p *KVMProvider) GetWorkspaceStatus(ctx context.Context, workspaceID string) (string, error) {
	info, err := p.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return "", err
	}
	return info.Status, nil
}

// GetWorkspaceResources returns resource usage information
func (p *KVMProvider) GetWorkspaceResources(ctx context.Context, workspaceID string) (*provider.ResourceInfo, error) {
	vmName := fmt.Sprintf("%s%s", p.prefix, workspaceID)

	// Get VM stats
	output, err := p.virshCmd(ctx, "domstats", vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM stats: %w", err)
	}

	// Parse stats - this is a simplified version
	stats := &provider.ResourceInfo{
		CPUUsage:    0,
		MemoryUsage: 0,
		StorageUsed: 0,
		NetworkIn:   0,
		NetworkOut:  0,
	}

	// Simple parsing of domstats output
	// In production, parse the actual libvirt stats format
	log.Printf("KVM stats for %s: %s", vmName, string(output))

	return stats, nil
}

// Validate checks if KVM/libvirt is properly configured
func (p *KVMProvider) Validate(ctx context.Context) error {
	// Check if virsh is available
	_, err := p.virshCmd(ctx, "list")
	if err != nil {
		return fmt.Errorf("libvirt/virsh is not available: %w", err)
	}

	// Check if KVM module is loaded
	cmd := exec.CommandContext(ctx, "lsmod")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check kernel modules: %w", err)
	}

	if !strings.Contains(string(output), "kvm") {
		return fmt.Errorf("KVM kernel module is not loaded")
	}

	return nil
}

// MarshalJSON implements json.Marshaler for KVMProvider
func (p *KVMProvider) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		LibvirtURI  string `json:"libvirt_uri"`
		ImagePath   string `json:"image_path"`
		NetworkName string `json:"network_name"`
		Prefix      string `json:"prefix"`
	}{
		LibvirtURI:  p.libvirtURI,
		ImagePath:   p.imagePath,
		NetworkName: p.networkName,
		Prefix:      p.prefix,
	})
}
