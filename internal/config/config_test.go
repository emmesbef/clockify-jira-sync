package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
)

func TestLoad_Success(t *testing.T) {
	// Setup env
	os.Setenv("CLOCKIFY_API_KEY", "test-key")
	os.Setenv("CLOCKIFY_WORKSPACE_ID", "test-ws")
	os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
	os.Setenv("JIRA_EMAIL", "test@example.com")
	os.Setenv("JIRA_API_TOKEN", "test-token")
	os.Setenv("TRAY_TIMER_FORMAT", "hh:mm")
	os.Setenv("TRAY_SHOW_TIMER", "false")

	// Cleanup
	defer func() {
		os.Unsetenv("CLOCKIFY_API_KEY")
		os.Unsetenv("CLOCKIFY_WORKSPACE_ID")
		os.Unsetenv("JIRA_BASE_URL")
		os.Unsetenv("JIRA_EMAIL")
		os.Unsetenv("JIRA_API_TOKEN")
		os.Unsetenv("TRAY_TIMER_FORMAT")
		os.Unsetenv("TRAY_SHOW_TIMER")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.ClockifyAPIKey != "test-key" {
		t.Errorf("Expected test-key, got %s", cfg.ClockifyAPIKey)
	}
	if cfg.JiraEmail != "test@example.com" {
		t.Errorf("Expected test@example.com, got %s", cfg.JiraEmail)
	}
	if cfg.TrayTimerFormat != "hh:mm" {
		t.Errorf("Expected hh:mm tray format, got %s", cfg.TrayTimerFormat)
	}
	if cfg.TrayShowTimer {
		t.Errorf("Expected tray timer visibility to be false")
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	// Isolate from real config files
	SetConfigDir(t.TempDir())
	defer SetConfigDir("")

	os.Unsetenv("CLOCKIFY_API_KEY")
	os.Unsetenv("CLOCKIFY_WORKSPACE_ID")
	os.Unsetenv("JIRA_BASE_URL")
	os.Unsetenv("JIRA_EMAIL")
	os.Unsetenv("JIRA_API_TOKEN")
	os.Unsetenv("MOCK_DATA")

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error for missing env var, got nil")
	}
}

func TestSave_WritesToConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	SetConfigDir(tmpDir)
	defer SetConfigDir("")

	cfg := &Config{
		ClockifyAPIKey:    "save-key",
		ClockifyWorkspace: "save-ws",
		JiraBaseURL:       "https://save.atlassian.net",
		JiraEmail:         "save@example.com",
		JiraAPIToken:      "save-token",
		TrayTimerFormat:   "hh:mm",
		TrayShowTimer:     false,
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	envPath := filepath.Join(tmpDir, ".env")
	envMap, err := godotenv.Read(envPath)
	if err != nil {
		t.Fatalf("failed to read .env: %v", err)
	}

	if envMap["CLOCKIFY_API_KEY"] != "save-key" {
		t.Errorf("expected CLOCKIFY_API_KEY=save-key, got %q", envMap["CLOCKIFY_API_KEY"])
	}
	if envMap["CLOCKIFY_WORKSPACE_ID"] != "save-ws" {
		t.Errorf("expected CLOCKIFY_WORKSPACE_ID=save-ws, got %q", envMap["CLOCKIFY_WORKSPACE_ID"])
	}
	if envMap["TRAY_TIMER_FORMAT"] != "hh:mm" {
		t.Errorf("expected TRAY_TIMER_FORMAT=hh:mm, got %q", envMap["TRAY_TIMER_FORMAT"])
	}
	if envMap["TRAY_SHOW_TIMER"] != "false" {
		t.Errorf("expected TRAY_SHOW_TIMER=false, got %q", envMap["TRAY_SHOW_TIMER"])
	}
}

func TestFilePath_ReturnsExpectedPath(t *testing.T) {
	tmpDir := t.TempDir()
	SetConfigDir(tmpDir)
	defer SetConfigDir("")

	p, err := FilePath()
	if err != nil {
		t.Fatalf("FilePath returned error: %v", err)
	}
	expected := filepath.Join(tmpDir, ".env")
	if p != expected {
		t.Errorf("expected %q, got %q", expected, p)
	}
}

func TestEnsurePersisted_CreatesWhenMissing(t *testing.T) {
	tmpDir := t.TempDir()
	SetConfigDir(tmpDir)
	defer SetConfigDir("")

	cfg := &Config{
		ClockifyAPIKey:    "persist-key",
		ClockifyWorkspace: "persist-ws",
		JiraBaseURL:       "https://persist.atlassian.net",
		JiraEmail:         "persist@example.com",
		JiraAPIToken:      "persist-token",
		AutoUpdate:        true,
	}

	created, err := EnsurePersisted(cfg)
	if err != nil {
		t.Fatalf("EnsurePersisted returned error: %v", err)
	}
	if !created {
		t.Error("expected created=true when .env is missing")
	}

	// Verify the file was written with correct values
	envPath := filepath.Join(tmpDir, ".env")
	envMap, err := godotenv.Read(envPath)
	if err != nil {
		t.Fatalf("failed to read created .env: %v", err)
	}
	if envMap["CLOCKIFY_API_KEY"] != "persist-key" {
		t.Errorf("expected CLOCKIFY_API_KEY=persist-key, got %q", envMap["CLOCKIFY_API_KEY"])
	}
	if envMap["JIRA_BASE_URL"] != "https://persist.atlassian.net" {
		t.Errorf("expected JIRA_BASE_URL=https://persist.atlassian.net, got %q", envMap["JIRA_BASE_URL"])
	}
}

func TestEnsurePersisted_SkipsWhenExists(t *testing.T) {
	tmpDir := t.TempDir()
	SetConfigDir(tmpDir)
	defer SetConfigDir("")

	// Pre-create .env with original credentials
	envPath := filepath.Join(tmpDir, ".env")
	original := map[string]string{
		"CLOCKIFY_API_KEY":      "original-key",
		"CLOCKIFY_WORKSPACE_ID": "original-ws",
		"JIRA_BASE_URL":         "https://original.atlassian.net",
		"JIRA_EMAIL":            "original@example.com",
		"JIRA_API_TOKEN":        "original-token",
	}
	if err := godotenv.Write(original, envPath); err != nil {
		t.Fatalf("failed to write seed .env: %v", err)
	}

	// Call EnsurePersisted with DIFFERENT values
	cfg := &Config{
		ClockifyAPIKey:    "new-key",
		ClockifyWorkspace: "new-ws",
		JiraBaseURL:       "https://new.atlassian.net",
		JiraEmail:         "new@example.com",
		JiraAPIToken:      "new-token",
	}

	created, err := EnsurePersisted(cfg)
	if err != nil {
		t.Fatalf("EnsurePersisted returned error: %v", err)
	}
	if created {
		t.Error("expected created=false when .env already exists")
	}

	// Verify original values are PRESERVED (not overwritten)
	envMap, err := godotenv.Read(envPath)
	if err != nil {
		t.Fatalf("failed to read .env: %v", err)
	}
	if envMap["CLOCKIFY_API_KEY"] != "original-key" {
		t.Errorf("expected original-key to be preserved, got %q", envMap["CLOCKIFY_API_KEY"])
	}
	if envMap["JIRA_BASE_URL"] != "https://original.atlassian.net" {
		t.Errorf("expected original URL to be preserved, got %q", envMap["JIRA_BASE_URL"])
	}
}
