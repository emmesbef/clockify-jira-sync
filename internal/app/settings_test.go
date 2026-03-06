package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"

	"clockify-jira-sync/internal/config"
	"clockify-jira-sync/internal/mockserver"
)

func TestSaveConfigPersistsWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to switch to temp directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	mockSrv := mockserver.Start()
	defer mockSrv.Close()

	a := NewApp(&config.Config{
		ClockifyAPIKey:    "old-key",
		ClockifyWorkspace: "old-workspace",
		JiraBaseURL:       "https://example.atlassian.net",
		JiraEmail:         "old@example.com",
		JiraAPIToken:      "old-token",
	})
	a.SetMockMode(mockSrv.URL)

	err = a.SaveConfig(config.Config{
		ClockifyAPIKey:    "new-key",
		ClockifyWorkspace: "new-workspace",
		JiraBaseURL:       "https://new-example.atlassian.net",
		JiraEmail:         "new@example.com",
		JiraAPIToken:      "new-token",
	})
	if err != nil {
		t.Fatalf("SaveConfig returned error: %v", err)
	}

	if a.cfg.ClockifyWorkspace != "new-workspace" {
		t.Fatalf("expected ClockifyWorkspace to be updated, got %q", a.cfg.ClockifyWorkspace)
	}

	envPath := filepath.Join(tmpDir, ".env")
	envMap, err := godotenv.Read(envPath)
	if err != nil {
		t.Fatalf("failed to read persisted .env file: %v", err)
	}

	if envMap["CLOCKIFY_WORKSPACE_ID"] != "new-workspace" {
		t.Fatalf("expected CLOCKIFY_WORKSPACE_ID to be persisted, got %q", envMap["CLOCKIFY_WORKSPACE_ID"])
	}
}

func TestGetIntegrationStatusSuccess(t *testing.T) {
	mockSrv := mockserver.Start()
	defer mockSrv.Close()

	a := NewApp(&config.Config{
		ClockifyAPIKey:    "key",
		ClockifyWorkspace: "workspace",
		JiraBaseURL:       "https://unused.atlassian.net",
		JiraEmail:         "user@example.com",
		JiraAPIToken:      "token",
	})
	a.SetMockMode(mockSrv.URL)

	status := a.GetIntegrationStatus()

	if !status.ClockifyConnected {
		t.Fatalf("expected Clockify to be connected, got error %q", status.ClockifyError)
	}
	if !status.JiraConnected {
		t.Fatalf("expected Jira to be connected, got error %q", status.JiraError)
	}
}

func TestGetIntegrationStatusFailure(t *testing.T) {
	a := NewApp(&config.Config{
		ClockifyAPIKey:    "key",
		ClockifyWorkspace: "workspace",
		JiraBaseURL:       "http://127.0.0.1:1",
		JiraEmail:         "user@example.com",
		JiraAPIToken:      "token",
	})
	a.clockify.SetBaseURL("http://127.0.0.1:1")
	a.jira.SetBaseURL("http://127.0.0.1:1")

	status := a.GetIntegrationStatus()

	if status.ClockifyConnected {
		t.Fatalf("expected Clockify connection check to fail")
	}
	if status.JiraConnected {
		t.Fatalf("expected Jira connection check to fail")
	}
	if status.ClockifyError == "" {
		t.Fatalf("expected Clockify error message to be populated")
	}
	if status.JiraError == "" {
		t.Fatalf("expected Jira error message to be populated")
	}
}
