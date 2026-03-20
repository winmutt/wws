package network

import (
	"testing"
)

func TestGenerateShortID(t *testing.T) {
	id1 := generateShortID()
	id2 := generateShortID()

	if len(id1) != 8 {
		t.Errorf("Expected ID length 8, got %d", len(id1))
	}

	if id1 == id2 {
		t.Error("Expected unique IDs")
	}
}

func TestGenerateNetworkName(t *testing.T) {
	nm := NewNetworkManager()
	name := nm.GenerateNetworkName("test-workspace")

	expectedPrefix := "wws-test-workspace-"
	if len(name) < len(expectedPrefix) {
		t.Errorf("Expected name to start with %s, got %s", expectedPrefix, name)
	}

	if len(name) != len(expectedPrefix)+8 {
		t.Errorf("Expected name length %d, got %d", len(expectedPrefix)+8, len(name))
	}
}

func TestGenerateSubnet(t *testing.T) {
	nm := NewNetworkManager()

	subnet1 := nm.GenerateSubnet(1)
	if subnet1 != "10.0.1.0/24" {
		t.Errorf("Expected subnet 10.0.1.0/24, got %s", subnet1)
	}

	subnet256 := nm.GenerateSubnet(256)
	if subnet256 != "10.1.0.0/24" {
		t.Errorf("Expected subnet 10.1.0.0/24, got %s", subnet256)
	}

	subnetLarge := nm.GenerateSubnet(1000)
	if subnetLarge != "10.3.232.0/24" {
		t.Errorf("Expected subnet 10.3.232.0/24, got %s", subnetLarge)
	}
}

func TestGenerateGateway(t *testing.T) {
	nm := NewNetworkManager()

	gateway := nm.GenerateGateway("10.0.1.0/24")
	if gateway != "10.0.1.1" {
		t.Errorf("Expected gateway 10.0.1.1, got %s", gateway)
	}

	gateway2 := nm.GenerateGateway("10.5.10.0/24")
	if gateway2 != "10.5.10.1" {
		t.Errorf("Expected gateway 10.5.10.1, got %s", gateway2)
	}
}

func TestGenerateContainerIP(t *testing.T) {
	nm := NewNetworkManager()

	ip := nm.generateContainerIP("10.0.1.1")
	if ip != "10.0.2" {
		t.Errorf("Expected container IP 10.0.2, got %s", ip)
	}
}

func TestValidateNetworkConfig(t *testing.T) {
	nm := NewNetworkManager()

	// Valid config
	config := &NetworkIsolationConfig{
		Subnet:       "10.0.1.0/24",
		AllowedPorts: []int{80, 443, 8080},
		BlockedIPs:   []string{"192.168.1.100"},
	}

	err := nm.ValidateNetworkConfig(config)
	if err != nil {
		t.Errorf("Expected no error for valid config, got %v", err)
	}

	// Invalid subnet
	config.Subnet = "invalid"
	err = nm.ValidateNetworkConfig(config)
	if err == nil {
		t.Error("Expected error for invalid subnet")
	}

	// Invalid port
	config.Subnet = "10.0.1.0/24"
	config.AllowedPorts = []int{70000}
	err = nm.ValidateNetworkConfig(config)
	if err == nil {
		t.Error("Expected error for invalid port")
	}

	// Invalid IP
	config.AllowedPorts = []int{80}
	config.BlockedIPs = []string{"not-an-ip"}
	err = nm.ValidateNetworkConfig(config)
	if err == nil {
		t.Error("Expected error for invalid IP")
	}
}

func TestNewNetworkManager(t *testing.T) {
	nm := NewNetworkManager()

	if nm == nil {
		t.Fatal("Expected non-nil network manager")
	}

	if !nm.IsEnabled() {
		t.Error("Expected network manager to be enabled by default")
	}
}

func TestGetNetworkInfo(t *testing.T) {
	nm := NewNetworkManager()

	info, err := nm.GetNetworkInfo("test-workspace")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if info.WorkspaceTag != "test-workspace" {
		t.Errorf("Expected workspace tag 'test-workspace', got %s", info.WorkspaceTag)
	}

	if !info.AllowOutbound {
		t.Error("Expected AllowOutbound to be true by default")
	}

	if !info.EnableFirewall {
		t.Error("Expected EnableFirewall to be true by default")
	}
}
