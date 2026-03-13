package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// configDirOverride allows tests to redirect config storage to a temp directory.
var configDirOverride string

const (
	configDirName       = "jirafy-clockwork"
	legacyConfigDirName = "clockify-jira-sync"
	configFileName      = ".env"
)

// SetConfigDir overrides the config directory (for tests only).
func SetConfigDir(dir string) {
	configDirOverride = dir
}

// ConfigDir returns the directory used for storing .env configuration.
// Uses os.UserConfigDir (~/Library/Application Support on macOS, %AppData% on
// Windows) with a jirafy-clockwork subdirectory.
func ConfigDir() (string, error) {
	if configDirOverride != "" {
		return configDirOverride, nil
	}

	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine config directory: %w", err)
	}

	newDir := filepath.Join(base, configDirName)
	legacyDir := filepath.Join(base, legacyConfigDirName)
	if err := migrateLegacyConfigFile(legacyDir, newDir); err != nil {
		return "", fmt.Errorf("cannot migrate config directory: %w", err)
	}
	return newDir, nil
}

// FilePath returns the full path to the .env config file.
func FilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
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
	TrayTimerFormat   string
	TrayShowTimer     bool
	LaunchOnStartup   bool
	SummaryWordLimit  int
	LogRoundingMin    int
}

const (
	trayTimerFormatHHMM   = "hh:mm"
	trayTimerFormatHHMMSS = "hh:mm:ss"
	summaryWordLimitMax   = 5
)

// NormalizeTrayTimerFormat returns a supported tray timer format.
func NormalizeTrayTimerFormat(format string) string {
	switch strings.TrimSpace(format) {
	case trayTimerFormatHHMM:
		return trayTimerFormatHHMM
	case trayTimerFormatHHMMSS:
		return trayTimerFormatHHMMSS
	default:
		return trayTimerFormatHHMMSS
	}
}

// NormalizeSummaryWordLimit returns a supported summary word limit.
// 0 means "full summary".
func NormalizeSummaryWordLimit(limit int) int {
	switch {
	case limit < 0:
		return 0
	case limit > summaryWordLimitMax:
		return summaryWordLimitMax
	default:
		return limit
	}
}

func parseSummaryWordLimit(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return NormalizeSummaryWordLimit(parsed)
}

// NormalizeLogRoundingMin returns a supported rounding interval (in minutes).
// 0 means no rounding.
func NormalizeLogRoundingMin(minutes int) int {
	switch minutes {
	case 0, 5, 10, 15, 30, 60:
		return minutes
	default:
		return 0
	}
}

func parseLogRoundingMin(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return NormalizeLogRoundingMin(parsed)
}

func Load() (*Config, error) {
	// Try user config dir first, fall back to local .env for development
	p, err := FilePath()
	if err != nil {
		return nil, fmt.Errorf("cannot resolve config file path: %w", err)
	}
	if err := godotenv.Load(p); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("cannot load config from %q: %w", p, err)
	}
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("cannot load local .env: %w", err)
	}

	cfg := &Config{
		ClockifyAPIKey:    os.Getenv("CLOCKIFY_API_KEY"),
		ClockifyWorkspace: os.Getenv("CLOCKIFY_WORKSPACE_ID"),
		JiraBaseURL:       os.Getenv("JIRA_BASE_URL"),
		JiraEmail:         os.Getenv("JIRA_EMAIL"),
		JiraAPIToken:      os.Getenv("JIRA_API_TOKEN"),
		MockMode:          os.Getenv("MOCK_DATA") == "true",
		AutoUpdate:        os.Getenv("AUTO_UPDATE") != "false",
		BetaChannel:       os.Getenv("BETA_CHANNEL") == "true",
		TrayTimerFormat:   NormalizeTrayTimerFormat(os.Getenv("TRAY_TIMER_FORMAT")),
		TrayShowTimer:     os.Getenv("TRAY_SHOW_TIMER") != "false",
		LaunchOnStartup:   os.Getenv("LAUNCH_ON_STARTUP") == "true",
		SummaryWordLimit:  parseSummaryWordLimit(os.Getenv("SUMMARY_WORD_LIMIT")),
		LogRoundingMin:    parseLogRoundingMin(os.Getenv("LOG_ROUNDING_MINUTES")),
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
		if !os.IsNotExist(err) {
			return fmt.Errorf("cannot read config file %q: %w", p, err)
		}
		envMap = make(map[string]string)
	}

	envMap["CLOCKIFY_API_KEY"] = cfg.ClockifyAPIKey
	envMap["CLOCKIFY_WORKSPACE_ID"] = cfg.ClockifyWorkspace
	envMap["JIRA_BASE_URL"] = cfg.JiraBaseURL
	envMap["JIRA_EMAIL"] = cfg.JiraEmail
	envMap["JIRA_API_TOKEN"] = cfg.JiraAPIToken
	envMap["AUTO_UPDATE"] = boolToStr(cfg.AutoUpdate)
	envMap["BETA_CHANNEL"] = boolToStr(cfg.BetaChannel)
	envMap["TRAY_TIMER_FORMAT"] = NormalizeTrayTimerFormat(cfg.TrayTimerFormat)
	envMap["TRAY_SHOW_TIMER"] = boolToStr(cfg.TrayShowTimer)
	envMap["LAUNCH_ON_STARTUP"] = boolToStr(cfg.LaunchOnStartup)
	envMap["SUMMARY_WORD_LIMIT"] = strconv.Itoa(NormalizeSummaryWordLimit(cfg.SummaryWordLimit))
	envMap["LOG_ROUNDING_MINUTES"] = strconv.Itoa(NormalizeLogRoundingMin(cfg.LogRoundingMin))

	if err := godotenv.Write(envMap, p); err != nil {
		return fmt.Errorf("cannot write config file %q: %w", p, err)
	}
	return nil
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

// EnsurePersisted checks whether the config dir .env file exists.
// If missing, it creates it from the current in-memory config.
// If the file already exists, it is left untouched — credentials are never overwritten.
// Returns true if a new file was created by this call.
func EnsurePersisted(cfg *Config) (bool, error) {
	p, err := FilePath()
	if err != nil {
		return false, fmt.Errorf("cannot resolve config file path: %w", err)
	}
	if _, statErr := os.Stat(p); statErr != nil {
		if !os.IsNotExist(statErr) {
			return false, fmt.Errorf("cannot stat config file %q: %w", p, statErr)
		}
		if err := Save(cfg); err != nil {
			return false, fmt.Errorf("cannot persist config file %q: %w", p, err)
		}
		return true, nil
	}
	return false, nil // file exists — nothing to do
}

func migrateLegacyConfigFile(legacyDir, newDir string) error {
	legacyPath := filepath.Join(legacyDir, configFileName)
	newPath := filepath.Join(newDir, configFileName)

	if _, err := os.Stat(newPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("cannot stat config file %q: %w", newPath, err)
	}

	info, err := os.Stat(legacyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("cannot stat legacy config file %q: %w", legacyPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("legacy config file path is a directory: %q", legacyPath)
	}

	if err := os.MkdirAll(newDir, 0o700); err != nil {
		return fmt.Errorf("cannot create config directory %q: %w", newDir, err)
	}

	if err := copyFileNoOverwrite(legacyPath, newPath, info.Mode().Perm()); err != nil {
		return fmt.Errorf("cannot migrate legacy config file to %q: %w", newPath, err)
	}
	return nil
}

func copyFileNoOverwrite(srcPath, dstPath string, mode os.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("cannot open source file %q: %w", srcPath, err)
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("cannot create destination file %q: %w", dstPath, err)
	}
	wroteFile := false
	defer func() {
		if dst != nil {
			_ = dst.Close()
		}
		if !wroteFile {
			_ = os.Remove(dstPath)
		}
	}()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("cannot copy file content: %w", err)
	}
	if err := dst.Sync(); err != nil {
		return fmt.Errorf("cannot sync destination file: %w", err)
	}
	if err := dst.Close(); err != nil {
		return fmt.Errorf("cannot close destination file %q: %w", dstPath, err)
	}
	dst = nil
	wroteFile = true
	return nil
}
