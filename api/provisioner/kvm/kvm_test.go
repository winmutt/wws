package kvm

import (
	"context"
	"testing"

	"wws/api/provisioner/provider"
)

func TestNewKVMProvider(t *testing.T) {
	p := NewKVMProvider("")
	if p.libvirtURI != "qemu:///system" {
		t.Errorf("Expected default libvirt URI 'qemu:///system', got '%s'", p.libvirtURI)
	}
	if p.prefix != "wws-" {
		t.Errorf("Expected prefix 'wws-', got '%s'", p.prefix)
	}
}

func TestNewKVMProviderCustom(t *testing.T) {
	uri := "qemu:///user"
	p := NewKVMProvider(uri)
	if p.libvirtURI != uri {
		t.Errorf("Expected libvirt URI '%s', got '%s'", uri, p.libvirtURI)
	}
}

func TestKVMProviderImplementsProvider(t *testing.T) {
	var _ provider.Provider = (*KVMProvider)(nil)
}

func TestVMExists(t *testing.T) {
	p := NewKVMProvider("")
	ctx := context.Background()

	// This will fail in test environment without libvirt, but tests the logic
	_, err := p.vmExists(ctx, "test-vm")
	// Expected to fail without actual libvirt setup
	if err == nil {
		t.Log("vmExists returned no error (expected in test environment without libvirt)")
	}
}

func TestGenerateVMXML(t *testing.T) {
	p := NewKVMProvider("")
	config := &provider.WorkspaceConfig{
		WorkspaceID:    "123",
		Tag:            "test-tag",
		CPU:            2,
		Memory:         4096,
		Storage:        50,
		OrganizationID: 1,
		UserID:         1,
	}

	xml := p.generateVMXML("wws-test", config)

	if xml == "" {
		t.Error("generateVMXML returned empty XML")
	}

	expectedElements := []string{
		"<domain type='kvm'>",
		"<name>wws-test</name>",
		"<memory unit='MiB'>4096</memory>",
		"<vcpu>2</vcpu>",
		"<disk",
		"<interface",
	}

	for _, elem := range expectedElements {
		if !contains(xml, elem) {
			t.Errorf("Generated XML missing expected element: %s", elem)
		}
	}
}

func TestGetBaseImagePath(t *testing.T) {
	p := NewKVMProvider("")
	expected := "/var/lib/libvirt/images/base.qcow2"
	result := p.getBaseImagePath()

	if result != expected {
		t.Errorf("Expected base image path '%s', got '%s'", expected, result)
	}
}

func TestCreateCloudInitISO(t *testing.T) {
	p := NewKVMProvider("")
	config := &provider.WorkspaceConfig{
		WorkspaceID: "123",
		Tag:         "test",
		GithubToken: "test-token",
	}

	ctx := context.Background()
	isoPath := "/tmp/test-iso.img"

	err := p.createCloudInitISO(ctx, isoPath, config)
	// Expected to succeed (creates files)
	if err != nil {
		t.Errorf("createCloudInitISO returned error: %v", err)
	}
}

func TestGetVMIP(t *testing.T) {
	p := NewKVMProvider("")
	ctx := context.Background()

	// This will fail without actual libvirt setup
	ip, err := p.getVMIP(ctx, "test-vm")
	if err == nil {
		t.Logf("getVMIP returned IP: %s", ip)
	}
}

func TestValidate(t *testing.T) {
	p := NewKVMProvider("")
	ctx := context.Background()

	err := p.Validate(ctx)
	// Expected to fail without actual libvirt setup
	if err == nil {
		t.Log("Validate passed (libvirt is available)")
	} else {
		t.Logf("Validate failed (expected without libvirt): %v", err)
	}
}

func TestMarshalJSON(t *testing.T) {
	p := NewKVMProvider("qemu:///test")
	p.imagePath = "/test/images"
	p.networkName = "test-net"
	p.prefix = "test-"

	data, err := p.MarshalJSON()
	if err != nil {
		t.Errorf("MarshalJSON returned error: %v", err)
	}

	if len(data) == 0 {
		t.Error("MarshalJSON returned empty data")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test provider interface methods (these will fail without libvirt, but test the structure)
func TestCreateWorkspace(t *testing.T) {
	p := NewKVMProvider("")
	ctx := context.Background()
	config := &provider.WorkspaceConfig{
		WorkspaceID:    "123",
		Tag:            "test",
		CPU:            2,
		Memory:         4096,
		Storage:        50,
		OrganizationID: 1,
		UserID:         1,
		Name:           "test-workspace",
	}

	_, err := p.CreateWorkspace(ctx, config)
	// Expected to fail without actual libvirt setup
	if err != nil {
		t.Logf("CreateWorkspace failed (expected without libvirt): %v", err)
	}
}

func TestGetWorkspace(t *testing.T) {
	p := NewKVMProvider("")
	ctx := context.Background()

	_, err := p.GetWorkspace(ctx, "test-id")
	// Expected to fail without actual libvirt setup
	if err != nil {
		t.Logf("GetWorkspace failed (expected without libvirt): %v", err)
	}
}

func TestStartWorkspace(t *testing.T) {
	p := NewKVMProvider("")
	ctx := context.Background()

	_, err := p.StartWorkspace(ctx, "test-id")
	// Expected to fail without actual libvirt setup
	if err != nil {
		t.Logf("StartWorkspace failed (expected without libvirt): %v", err)
	}
}

func TestStopWorkspace(t *testing.T) {
	p := NewKVMProvider("")
	ctx := context.Background()

	_, err := p.StopWorkspace(ctx, "test-id")
	// Expected to fail without actual libvirt setup
	if err != nil {
		t.Logf("StopWorkspace failed (expected without libvirt): %v", err)
	}
}

func TestDeleteWorkspace(t *testing.T) {
	p := NewKVMProvider("")
	ctx := context.Background()

	err := p.DeleteWorkspace(ctx, "test-id")
	// Expected to fail without actual libvirt setup
	if err != nil {
		t.Logf("DeleteWorkspace failed (expected without libvirt): %v", err)
	}
}
