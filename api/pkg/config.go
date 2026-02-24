package pkg

import (
	"fmt"
	"os"
)

type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	GitHub     GitHubConfig
	Workspaces WorkspaceConfig
}

type ServerConfig struct {
	Port string
	CORS CORSConfig
}

type CORSConfig struct {
	Origins []string
}

type DatabaseConfig struct {
	Path string
}

type GitHubConfig struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
}

type WorkspaceConfig struct {
	IdleTimeoutHours int
	DefaultStorageGB int
	DefaultCPU       int
	DefaultMemoryGB  int
}

func LoadConfig() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			CORS: CORSConfig{
				Origins: []string{"http://localhost:3000"},
			},
		},
		Database: DatabaseConfig{
			Path: getEnv("DATABASE_URL", "./data/wws.db"),
		},
		GitHub: GitHubConfig{
			ClientID:     getEnv("GITHUB_CLIENT_ID", ""),
			ClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
			CallbackURL:  getEnv("GITHUB_CALLBACK_URL", "http://localhost:8080/oauth/callback"),
		},
		Workspaces: WorkspaceConfig{
			IdleTimeoutHours: 6,
			DefaultStorageGB: 20,
			DefaultCPU:       2,
			DefaultMemoryGB:  4,
		},
	}

	if config.GitHub.ClientID == "" || config.GitHub.ClientSecret == "" {
		return nil, fmt.Errorf("GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET must be set")
	}

	return config, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
