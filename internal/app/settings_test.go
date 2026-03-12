package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"

	"jirafy-clockwork/internal/config"
	"jirafy-clockwork/internal/mockserver"
	"jirafy-clockwork/internal/models"
)

func TestSaveConfigPersistsWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetConfigDir(tmpDir)
	defer config.SetConfigDir("")

	mockSrv := mockserver.Start()
	defer mockSrv.Close()

	a := NewApp(&config.Config{
		ClockifyAPIKey:    "old-key",
		ClockifyWorkspace: "old-workspace",
		JiraBaseURL:       "https://example.atlassian.net",
		JiraEmail:         "old@example.com",
		JiraAPIToken:      "old-token",
	}, "test")
	a.SetMockMode(mockSrv.URL)

	err := a.SaveConfig(config.Config{
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

func TestSaveConfigPreservesUpdatePreferences(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetConfigDir(tmpDir)
	defer config.SetConfigDir("")

	mockSrv := mockserver.Start()
	defer mockSrv.Close()

	a := NewApp(&config.Config{
		ClockifyAPIKey:    "old-key",
		ClockifyWorkspace: "old-workspace",
		JiraBaseURL:       "https://example.atlassian.net",
		JiraEmail:         "old@example.com",
		JiraAPIToken:      "old-token",
		AutoUpdate:        true,
		BetaChannel:       true,
	}, "test")
	a.SetMockMode(mockSrv.URL)

	err := a.SaveConfig(config.Config{
		ClockifyAPIKey:    "new-key",
		ClockifyWorkspace: "new-workspace",
		JiraBaseURL:       "https://new-example.atlassian.net",
		JiraEmail:         "new@example.com",
		JiraAPIToken:      "new-token",
		// Update preferences intentionally omitted (zero values).
	})
	if err != nil {
		t.Fatalf("SaveConfig returned error: %v", err)
	}

	if !a.cfg.AutoUpdate {
		t.Fatalf("expected AutoUpdate to stay true")
	}
	if !a.cfg.BetaChannel {
		t.Fatalf("expected BetaChannel to stay true")
	}

	envPath := filepath.Join(tmpDir, ".env")
	envMap, err := godotenv.Read(envPath)
	if err != nil {
		t.Fatalf("failed to read persisted .env file: %v", err)
	}

	if envMap["AUTO_UPDATE"] != "true" {
		t.Fatalf("expected AUTO_UPDATE to remain true, got %q", envMap["AUTO_UPDATE"])
	}
	if envMap["BETA_CHANNEL"] != "true" {
		t.Fatalf("expected BETA_CHANNEL to remain true, got %q", envMap["BETA_CHANNEL"])
	}
}

func TestSetUpdatePreferencesPersistsToConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetConfigDir(tmpDir)
	defer config.SetConfigDir("")

	a := NewApp(&config.Config{
		ClockifyAPIKey:    "key",
		ClockifyWorkspace: "workspace",
		JiraBaseURL:       "https://example.atlassian.net",
		JiraEmail:         "user@example.com",
		JiraAPIToken:      "token",
		AutoUpdate:        true,
		BetaChannel:       false,
	}, "test")

	if err := a.SetUpdatePreferences(models.UpdatePreferences{
		AutoCheck:   false,
		BetaChannel: true,
	}); err != nil {
		t.Fatalf("SetUpdatePreferences returned error: %v", err)
	}

	if a.cfg.AutoUpdate {
		t.Fatalf("expected AutoUpdate to be false after update preference save")
	}
	if !a.cfg.BetaChannel {
		t.Fatalf("expected BetaChannel to be true after update preference save")
	}

	envPath := filepath.Join(tmpDir, ".env")
	envMap, err := godotenv.Read(envPath)
	if err != nil {
		t.Fatalf("failed to read persisted .env file: %v", err)
	}
	if envMap["AUTO_UPDATE"] != "false" {
		t.Fatalf("expected AUTO_UPDATE=false, got %q", envMap["AUTO_UPDATE"])
	}
	if envMap["BETA_CHANNEL"] != "true" {
		t.Fatalf("expected BETA_CHANNEL=true, got %q", envMap["BETA_CHANNEL"])
	}
}

func TestEnsureConfigPersistedCreatesNewFile(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetConfigDir(tmpDir)
	defer config.SetConfigDir("")

	a := NewApp(&config.Config{
		ClockifyAPIKey:    "test-key",
		ClockifyWorkspace: "test-workspace",
		JiraBaseURL:       "https://example.atlassian.net",
		JiraEmail:         "test@example.com",
		JiraAPIToken:      "test-token",
	}, "test")

	result := a.EnsureConfigPersisted()

	if !result.Created {
		t.Fatalf("expected Created to be true when no config file exists")
	}
	expectedPath := filepath.Join(tmpDir, ".env")
	if result.Path != expectedPath {
		t.Fatalf("expected Path to be %q, got %q", expectedPath, result.Path)
	}
}

func TestEnsureConfigPersistedExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	config.SetConfigDir(tmpDir)
	defer config.SetConfigDir("")

	envPath := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envPath, []byte("CLOCKIFY_API_KEY=existing-key\n"), 0o600); err != nil {
		t.Fatalf("failed to create pre-existing config file: %v", err)
	}

	a := NewApp(&config.Config{
		ClockifyAPIKey:    "test-key",
		ClockifyWorkspace: "test-workspace",
		JiraBaseURL:       "https://example.atlassian.net",
		JiraEmail:         "test@example.com",
		JiraAPIToken:      "test-token",
	}, "test")

	result := a.EnsureConfigPersisted()

	if result.Created {
		t.Fatalf("expected Created to be false when config file already exists")
	}
	expectedPath := filepath.Join(tmpDir, ".env")
	if result.Path != expectedPath {
		t.Fatalf("expected Path to be %q, got %q", expectedPath, result.Path)
	}
}

func TestEnsureConfigPersistedMigratesLegacyConfigPath(t *testing.T) {
	config.SetConfigDir("")
	defer config.SetConfigDir("")

	home := t.TempDir()
	t.Setenv("HOME", home)

	base, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("failed to resolve user config dir: %v", err)
	}

	legacyPath := filepath.Join(base, "clockify-jira-sync", ".env")
	newPath := filepath.Join(base, "jirafy-clockwork", ".env")

	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o700); err != nil {
		t.Fatalf("failed to create legacy config dir: %v", err)
	}
	if err := godotenv.Write(map[string]string{
		"CLOCKIFY_API_KEY": "legacy-key",
		"JIRA_EMAIL":       "legacy@example.com",
	}, legacyPath); err != nil {
		t.Fatalf("failed to write legacy config: %v", err)
	}

	a := NewApp(&config.Config{
		ClockifyAPIKey:    "new-key",
		ClockifyWorkspace: "new-workspace",
		JiraBaseURL:       "https://example.atlassian.net",
		JiraEmail:         "new@example.com",
		JiraAPIToken:      "new-token",
	}, "test")

	result := a.EnsureConfigPersisted()
	if result.Created {
		t.Fatalf("expected Created to be false after migration copied legacy config")
	}
	if result.Path != newPath {
		t.Fatalf("expected migrated Path %q, got %q", newPath, result.Path)
	}

	envMap, err := godotenv.Read(newPath)
	if err != nil {
		t.Fatalf("failed to read migrated config: %v", err)
	}
	if envMap["CLOCKIFY_API_KEY"] != "legacy-key" {
		t.Fatalf("expected migrated CLOCKIFY_API_KEY=legacy-key, got %q", envMap["CLOCKIFY_API_KEY"])
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
	}, "test")
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
	}, "test")
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
