package network

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

// NetworkIsolationConfig represents the network isolation configuration for a workspace
type NetworkIsolationConfig struct {
	WorkspaceID    int      `json:"workspace_id"`
	WorkspaceTag   string   `json:"workspace_tag"`
	NetworkName    string   `json:"network_name"`
	Subnet         string   `json:"subnet"`
	Gateway        string   `json:"gateway"`
	AllowOutbound  bool     `json:"allow_outbound"`
	AllowedPorts   []int    `json:"allowed_ports"`
	BlockedIPs     []string `json:"blocked_ips"`
	DNSServers     []string `json:"dns_servers"`
	EnableFirewall bool     `json:"enable_firewall"`
}

// NetworkManager handles network isolation operations
type NetworkManager struct {
	enabled bool
}

// NewNetworkManager creates a new network manager
func NewNetworkManager() *NetworkManager {
	return &NetworkManager{
		enabled: true, // Enable by default, can be disabled via config
	}
}

// IsEnabled returns whether network isolation is enabled
func (nm *NetworkManager) IsEnabled() bool {
	return nm.enabled
}

// GenerateShortID generates a short random ID
func generateShortID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)[:8]
}

// GenerateNetworkName generates a unique network name for a workspace
func (nm *NetworkManager) GenerateNetworkName(workspaceTag string) string {
	return fmt.Sprintf("wws-%s-%s", workspaceTag, generateShortID())
}

// GenerateSubnet generates a unique subnet for the workspace network
func (nm *NetworkManager) GenerateSubnet(workspaceID int) string {
	// Use private subnet range 10.X.Y.0/24
	// X = (workspaceID / 256) % 256
	// Y = workspaceID % 256
	x := (workspaceID / 256) % 256
	y := workspaceID % 256
	return fmt.Sprintf("10.%d.%d.0/24", x, y)
}

// GenerateGateway generates the gateway IP for a subnet
func (nm *NetworkManager) GenerateGateway(subnet string) string {
	// Gateway is typically the first usable IP in the subnet
	parts := strings.Split(subnet, "/")
	if len(parts) == 0 {
		return ""
	}
	ipParts := strings.Split(parts[0], ".")
	if len(ipParts) < 3 {
		return ""
	}
	return fmt.Sprintf("%s.1", strings.Join(ipParts[:3], "."))
}

// CreateNetwork creates an isolated network for the workspace
func (nm *NetworkManager) CreateNetwork(config *NetworkIsolationConfig) error {
	if !nm.enabled {
		return nil
	}

	// Generate network configuration if not provided
	if config.NetworkName == "" {
		config.NetworkName = nm.GenerateNetworkName(config.WorkspaceTag)
	}
	if config.Subnet == "" {
		config.Subnet = nm.GenerateSubnet(config.WorkspaceID)
	}
	if config.Gateway == "" {
		config.Gateway = nm.GenerateGateway(config.Subnet)
	}

	// Set default values
	if config.AllowOutbound {
		config.AllowOutbound = true
	}
	if config.EnableFirewall {
		config.EnableFirewall = true
	}
	if len(config.DNSServers) == 0 {
		config.DNSServers = []string{"8.8.8.8", "8.8.4.4"}
	}

	// Create network namespace
	if err := nm.createNetworkNamespace(config.NetworkName); err != nil {
		return fmt.Errorf("failed to create network namespace: %w", err)
	}

	// Configure network interface
	if err := nm.configureNetworkInterface(config); err != nil {
		return fmt.Errorf("failed to configure network interface: %w", err)
	}

	// Configure firewall rules if enabled
	if config.EnableFirewall {
		if err := nm.configureFirewallRules(config); err != nil {
			return fmt.Errorf("failed to configure firewall rules: %w", err)
		}
	}

	return nil
}

// createNetworkNamespace creates a new network namespace
func (nm *NetworkManager) createNetworkNamespace(networkName string) error {
	// Check if ip netns command is available
	if !nm.commandExists("ip") {
		return fmt.Errorf("ip command not found - network isolation requires iproute2")
	}

	// Create network namespace
	cmd := exec.Command("ip", "netns", "add", networkName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create network namespace: %w", err)
	}

	return nil
}

// configureNetworkInterface configures network interfaces for the workspace
func (nm *NetworkManager) configureNetworkInterface(config *NetworkIsolationConfig) error {
	networkName := config.NetworkName

	// Create veth pair
	vethName := fmt.Sprintf("veth-%s", config.WorkspaceTag[:8])
	containerVeth := fmt.Sprintf("veth-ct-%s", config.WorkspaceTag[:8])

	// Create veth pair
	cmd := exec.Command("ip", "link", "add", vethName, "type", "veth", "peer", "name", containerVeth)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create veth pair: %w", err)
	}

	// Move container side to network namespace
	cmd = exec.Command("ip", "link", "set", containerVeth, "netns", networkName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to move interface to namespace: %w", err)
	}

	// Configure host side
	cmd = exec.Command("ip", "link", "set", vethName, "up")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to bring up host interface: %w", err)
	}

	// Configure container side (in namespace)
	ipAddr := fmt.Sprintf("%s/24", nm.generateContainerIP(config.Gateway))
	cmd = exec.Command("ip", "netns", "exec", networkName, "ip", "addr", "add", ipAddr, "dev", containerVeth)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set container IP: %w", err)
	}

	cmd = exec.Command("ip", "netns", "exec", networkName, "ip", "link", "set", containerVeth, "up")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to bring up container interface: %w", err)
	}

	// Set default route in namespace
	cmd = exec.Command("ip", "netns", "exec", networkName, "ip", "route", "add", "default", "via", config.Gateway)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set default route: %w", err)
	}

	// Configure DNS
	if err := nm.configureDNS(networkName, config.DNSServers); err != nil {
		return fmt.Errorf("failed to configure DNS: %w", err)
	}

	return nil
}

// configureDNS configures DNS settings in the network namespace
func (nm *NetworkManager) configureDNS(networkName string, dnsServers []string) error {
	// Build new resolv.conf with custom DNS
	var dnsContent strings.Builder
	dnsContent.WriteString("nameserver 127.0.0.53\n") // Local resolver
	for _, dns := range dnsServers {
		dnsContent.WriteString(fmt.Sprintf("nameserver %s\n", dns))
	}

	// Write to namespace
	cmd := exec.Command("ip", "netns", "exec", networkName, "sh", "-c",
		fmt.Sprintf("echo '%s' > /etc/resolv.conf", dnsContent.String()))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to write resolv.conf: %w", err)
	}

	return nil
}

// configureFirewallRules configures iptables rules for the workspace
func (nm *NetworkManager) configureFirewallRules(config *NetworkIsolationConfig) error {
	networkName := config.NetworkName

	// Enable IP forwarding
	if err := nm.enableIPForwarding(); err != nil {
		return fmt.Errorf("failed to enable IP forwarding: %w", err)
	}

	// Set up NAT for outbound traffic if allowed
	if config.AllowOutbound {
		if err := nm.setupNAT(networkName); err != nil {
			return fmt.Errorf("failed to setup NAT: %w", err)
		}
	}

	// Configure default policies
	if err := nm.setDefaultPolicies(networkName, config); err != nil {
		return fmt.Errorf("failed to set default policies: %w", err)
	}

	// Block specific IPs if configured
	for _, blockedIP := range config.BlockedIPs {
		if err := nm.blockIP(networkName, blockedIP); err != nil {
			return fmt.Errorf("failed to block IP %s: %w", blockedIP, err)
		}
	}

	// Allow specific ports
	for _, port := range config.AllowedPorts {
		if err := nm.allowPort(networkName, port); err != nil {
			return fmt.Errorf("failed to allow port %d: %w", port, err)
		}
	}

	return nil
}

// setupNAT sets up NAT rules for outbound traffic
func (nm *NetworkManager) setupNAT(networkName string) error {
	// Masquerade outbound traffic
	cmd := exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-o", "eth0", "-j", "MASQUERADE")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to setup NAT masquerade: %w", err)
	}

	return nil
}

// setDefaultPolicies sets default firewall policies
func (nm *NetworkManager) setDefaultPolicies(networkName string, config *NetworkIsolationConfig) error {
	// Default policy: allow established connections
	cmd := exec.Command("iptables", "-A", "INPUT", "-m", "state", "--state", "ESTABLISHED,RELATED", "-j", "ACCEPT")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set INPUT policy: %w", err)
	}

	// Allow loopback
	cmd = exec.Command("iptables", "-A", "INPUT", "-i", "lo", "-j", "ACCEPT")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to allow loopback: %w", err)
	}

	// Drop all other incoming traffic
	cmd = exec.Command("iptables", "-A", "INPUT", "-j", "DROP")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set DROP policy: %w", err)
	}

	return nil
}

// blockIP blocks a specific IP address
func (nm *NetworkManager) blockIP(networkName string, ip string) error {
	cmd := exec.Command("iptables", "-A", "OUTPUT", "-d", ip, "-j", "DROP")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to block IP: %w", err)
	}
	return nil
}

// allowPort allows traffic on a specific port
func (nm *NetworkManager) allowPort(networkName string, port int) error {
	cmd := exec.Command("iptables", "-A", "INPUT", "-p", "tcp", "--dport", fmt.Sprintf("%d", port), "-j", "ACCEPT")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to allow port: %w", err)
	}
	return nil
}

// enableIPForwarding enables IP forwarding in the kernel
func (nm *NetworkManager) enableIPForwarding() error {
	cmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable IP forwarding: %w", err)
	}
	return nil
}

// generateContainerIP generates a container IP address in the subnet
func (nm *NetworkManager) generateContainerIP(gateway string) string {
	// Gateway is typically .1, so use .2 for container
	parts := strings.Split(gateway, ".")
	if len(parts) < 3 {
		return "10.0.0.2"
	}
	return fmt.Sprintf("%s.%s.2", parts[0], parts[1])
}

// commandExists checks if a command exists in PATH
func (nm *NetworkManager) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// DeleteNetwork removes the isolated network for a workspace
func (nm *NetworkManager) DeleteNetwork(networkName string) error {
	if !nm.enabled {
		return nil
	}

	// Flush iptables rules for this namespace
	nm.flushIptablesRules(networkName)

	// Delete network namespace
	cmd := exec.Command("ip", "netns", "del", networkName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete network namespace: %w", err)
	}

	return nil
}

// flushIptablesRules removes iptables rules for a network namespace
func (nm *NetworkManager) flushIptablesRules(networkName string) {
	// Flush custom chains
	cmd := exec.Command("iptables", "-F")
	_ = cmd.Run()

	cmd = exec.Command("iptables", "-X")
	_ = cmd.Run()
}

// GetNetworkInfo returns information about a workspace's network
func (nm *NetworkManager) GetNetworkInfo(workspaceTag string) (*NetworkIsolationConfig, error) {
	// This would query the actual network configuration
	// For now, return a placeholder
	return &NetworkIsolationConfig{
		WorkspaceTag:   workspaceTag,
		NetworkName:    nm.GenerateNetworkName(workspaceTag),
		Subnet:         "10.0.0.0/24",
		Gateway:        "10.0.0.1",
		AllowOutbound:  true,
		EnableFirewall: true,
	}, nil
}

// ValidateNetworkConfig validates a network isolation configuration
func (nm *NetworkManager) ValidateNetworkConfig(config *NetworkIsolationConfig) error {
	// Validate subnet format
	if config.Subnet != "" {
		_, _, err := net.ParseCIDR(config.Subnet)
		if err != nil {
			return fmt.Errorf("invalid subnet: %w", err)
		}
	}

	// Validate port range
	for _, port := range config.AllowedPorts {
		if port < 1 || port > 65535 {
			return fmt.Errorf("invalid port %d: must be between 1 and 65535", port)
		}
	}

	// Validate IP addresses
	for _, ip := range config.BlockedIPs {
		if net.ParseIP(ip) == nil {
			return fmt.Errorf("invalid IP address: %s", ip)
		}
	}

	return nil
}

// WaitForNetworkReady waits for the network to be ready
func (nm *NetworkManager) WaitForNetworkReady(networkName string, timeout time.Duration) error {
	start := time.Now()
	for time.Since(start) < timeout {
		cmd := exec.Command("ip", "netns", "exec", networkName, "ip", "addr")
		if err := cmd.Run(); err == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for network to be ready")
}
