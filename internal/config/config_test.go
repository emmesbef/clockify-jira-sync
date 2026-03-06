package config

import (
	"os"
	"testing"
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
	// Clear all explicitly
	os.Unsetenv("CLOCKIFY_API_KEY")

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error for missing env var, got nil")
	}
}
