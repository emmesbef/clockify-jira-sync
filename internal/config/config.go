package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ClockifyAPIKey    string
	ClockifyWorkspace string
	JiraBaseURL       string
	JiraEmail         string
	JiraAPIToken      string
	MockMode          bool
}

func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{
		ClockifyAPIKey:    os.Getenv("CLOCKIFY_API_KEY"),
		ClockifyWorkspace: os.Getenv("CLOCKIFY_WORKSPACE_ID"),
		JiraBaseURL:       os.Getenv("JIRA_BASE_URL"),
		JiraEmail:         os.Getenv("JIRA_EMAIL"),
		JiraAPIToken:      os.Getenv("JIRA_API_TOKEN"),
		MockMode:          os.Getenv("MOCK_DATA") == "true",
	}

	if cfg.MockMode {
		if cfg.ClockifyAPIKey == "" {
			cfg.ClockifyAPIKey = "mock-key"
		}
		if cfg.ClockifyWorkspace == "" {
			cfg.ClockifyWorkspace = "mock-workspace"
		}
		if cfg.JiraEmail == "" {
			cfg.JiraEmail = "mock@example.com"
		}
		if cfg.JiraAPIToken == "" {
			cfg.JiraAPIToken = "mock-token"
		}
		return cfg, nil
	}

	if cfg.ClockifyAPIKey == "" {
		return nil, fmt.Errorf("CLOCKIFY_API_KEY is required")
	}
	if cfg.ClockifyWorkspace == "" {
		return nil, fmt.Errorf("CLOCKIFY_WORKSPACE_ID is required")
	}
	if cfg.JiraBaseURL == "" {
		return nil, fmt.Errorf("JIRA_BASE_URL is required")
	}
	if cfg.JiraEmail == "" {
		return nil, fmt.Errorf("JIRA_EMAIL is required")
	}
	if cfg.JiraAPIToken == "" {
		return nil, fmt.Errorf("JIRA_API_TOKEN is required")
	}

	return cfg, nil
}

// Save writes the configuration to the .env file
func Save(cfg *Config) error {
	envMap, err := godotenv.Read()
	if err != nil {
		envMap = make(map[string]string)
	}

	envMap["CLOCKIFY_API_KEY"] = cfg.ClockifyAPIKey
	envMap["CLOCKIFY_WORKSPACE_ID"] = cfg.ClockifyWorkspace
	envMap["JIRA_BASE_URL"] = cfg.JiraBaseURL
	envMap["JIRA_EMAIL"] = cfg.JiraEmail
	envMap["JIRA_API_TOKEN"] = cfg.JiraAPIToken

	return godotenv.Write(envMap, ".env")
}
