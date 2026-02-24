package pkg

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	os.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	os.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")

	config, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig returned error: %v", err)
	}

	if config.Server.Port != "8080" {
		t.Errorf("Expected default port 8080, got %s", config.Server.Port)
	}

	if config.Workspaces.IdleTimeoutHours != 6 {
		t.Errorf("Expected default idle timeout 6, got %d", config.Workspaces.IdleTimeoutHours)
	}
}

func TestLoadConfigMissingGitHub(t *testing.T) {
	os.Unsetenv("GITHUB_CLIENT_ID")
	os.Unsetenv("GITHUB_CLIENT_SECRET")

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error for missing GitHub credentials")
	}
}
