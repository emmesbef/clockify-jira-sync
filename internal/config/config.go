package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// configDirOverride allows tests to redirect config storage to a temp directory.
var configDirOverride string

// SetConfigDir overrides the config directory (for tests only).
func SetConfigDir(dir string) {
	configDirOverride = dir
}

// ConfigDir returns the directory used for storing .env configuration.
// Uses os.UserConfigDir (~/Library/Application Support on macOS, %AppData% on
// Windows) with a clockify-jira-sync subdirectory.
func ConfigDir() (string, error) {
	if configDirOverride != "" {
		return configDirOverride, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine config directory: %w", err)
	}
	return filepath.Join(base, "clockify-jira-sync"), nil
}

// FilePath returns the full path to the .env config file.
func FilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".env"), nil
}

type Config struct {
	ClockifyAPIKey    string
	ClockifyWorkspace string
	JiraBaseURL       string
	JiraEmail         string
	JiraAPIToken      string
	MockMode          bool
	AutoUpdate        bool
	BetaChannel       bool
}

func Load() (*Config, error) {
	// Try user config dir first, fall back to local .env for development
	if p, err := FilePath(); err == nil {
		_ = godotenv.Load(p)
	}
	_ = godotenv.Load() // local .env (dev); vars already set above win

	cfg := &Config{
		ClockifyAPIKey:    os.Getenv("CLOCKIFY_API_KEY"),
		ClockifyWorkspace: os.Getenv("CLOCKIFY_WORKSPACE_ID"),
		JiraBaseURL:       os.Getenv("JIRA_BASE_URL"),
		JiraEmail:         os.Getenv("JIRA_EMAIL"),
		JiraAPIToken:      os.Getenv("JIRA_API_TOKEN"),
		MockMode:          os.Getenv("MOCK_DATA") == "true",
		AutoUpdate:        os.Getenv("AUTO_UPDATE") != "false",
		BetaChannel:       os.Getenv("BETA_CHANNEL") == "true",
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
	// ClockifyWorkspace may be empty at startup; the UI auto-fetches it from
	// the Clockify API when the user provides an API key.
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

// Save writes the configuration to the .env file in the user config directory.
func Save(cfg *Config) error {
	p, err := FilePath()
	if err != nil {
		return err
	}

	// Ensure the config directory exists
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}

	envMap, err := godotenv.Read(p)
	if err != nil {
		envMap = make(map[string]string)
	}

	envMap["CLOCKIFY_API_KEY"] = cfg.ClockifyAPIKey
	envMap["CLOCKIFY_WORKSPACE_ID"] = cfg.ClockifyWorkspace
	envMap["JIRA_BASE_URL"] = cfg.JiraBaseURL
	envMap["JIRA_EMAIL"] = cfg.JiraEmail
	envMap["JIRA_API_TOKEN"] = cfg.JiraAPIToken
	envMap["AUTO_UPDATE"] = boolToStr(cfg.AutoUpdate)
	envMap["BETA_CHANNEL"] = boolToStr(cfg.BetaChannel)

	return godotenv.Write(envMap, p)
}

// Save is a convenience method that delegates to the package-level Save function.
func (c *Config) Save() error {
	return Save(c)
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
