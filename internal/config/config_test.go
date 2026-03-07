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

	// Cleanup
	defer func() {
		os.Unsetenv("CLOCKIFY_API_KEY")
		os.Unsetenv("CLOCKIFY_WORKSPACE_ID")
		os.Unsetenv("JIRA_BASE_URL")
		os.Unsetenv("JIRA_EMAIL")
		os.Unsetenv("JIRA_API_TOKEN")
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
