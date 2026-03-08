package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"2.0.0", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},
		{"1.2.0", "1.1.0", 1},
		{"1.0.2", "1.0.1", 1},
		{"v1.8.7", "1.8.7", 0},
		{"1.9.0", "1.8.7", 1},
		{"1.8.7", "1.9.0", -1},
		{"1.10.0", "1.9.0", 1},
		{"2.0.0", "1.99.99", 1},
	}

	for _, tt := range tests {
		got := CompareVersions(tt.a, tt.b)
		if (tt.want > 0 && got <= 0) || (tt.want < 0 && got >= 0) || (tt.want == 0 && got != 0) {
			t.Errorf("CompareVersions(%q, %q) = %d, want sign of %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestIsPreReleaseVersion(t *testing.T) {
	tests := []struct {
		v    string
		want bool
	}{
		{"1.8.7", false},
		{"v1.8.7", false},
		{"1.9.0-beta.1", true},
		{"v2.0.0-rc.1", true},
		{"dev", false},
	}

	for _, tt := range tests {
		got := IsPreReleaseVersion(tt.v)
		if got != tt.want {
			t.Errorf("IsPreReleaseVersion(%q) = %v, want %v", tt.v, got, tt.want)
		}
	}
}

func TestNormalizeVersion(t *testing.T) {
	if got := normalizeVersion("v1.2.3"); got != "1.2.3" {
		t.Errorf("normalizeVersion(v1.2.3) = %q", got)
	}
	if got := normalizeVersion("1.2.3"); got != "1.2.3" {
		t.Errorf("normalizeVersion(1.2.3) = %q", got)
	}
}

func mockGitHubServer(releases []ghRelease) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(releases)
	}))
}

func TestCheckForUpdate_NewerAvailable(t *testing.T) {
	server := mockGitHubServer([]ghRelease{
		{
			TagName:    "v2.0.0",
			PreRelease: false,
			Assets: []ghAsset{
				{Name: "clockify-jira-sync-v2.0.0-macos-universal.zip", BrowserDownloadURL: "https://example.com/mac.zip", Size: 1000},
				{Name: "clockify-jira-sync-v2.0.0-windows-amd64.zip", BrowserDownloadURL: "https://example.com/win.zip", Size: 2000},
			},
		},
		{
			TagName:    "v1.8.0",
			PreRelease: false,
		},
	})
	defer server.Close()

	u := New()
	u.SetBaseURL(server.URL)

	info, err := u.CheckForUpdate("1.8.7", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected update info, got nil")
	}
	if info.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", info.Version)
	}
	if info.IsPreRelease {
		t.Error("expected stable release")
	}
}

func TestCheckForUpdate_UpToDate(t *testing.T) {
	server := mockGitHubServer([]ghRelease{
		{TagName: "v1.8.7", PreRelease: false},
		{TagName: "v1.8.6", PreRelease: false},
	})
	defer server.Close()

	u := New()
	u.SetBaseURL(server.URL)

	info, err := u.CheckForUpdate("1.8.7", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Errorf("expected nil (up to date), got %+v", info)
	}
}

func TestCheckForUpdate_BetaExcluded(t *testing.T) {
	server := mockGitHubServer([]ghRelease{
		{TagName: "v2.0.0-beta.1", PreRelease: true},
		{TagName: "v1.8.7", PreRelease: false},
	})
	defer server.Close()

	u := New()
	u.SetBaseURL(server.URL)

	// Without beta, should be up to date
	info, err := u.CheckForUpdate("1.8.7", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Errorf("expected nil (beta excluded), got %+v", info)
	}
}

func TestCheckForUpdate_BetaIncluded(t *testing.T) {
	server := mockGitHubServer([]ghRelease{
		{TagName: "v2.0.0-beta.1", PreRelease: true,
			Assets: []ghAsset{{Name: "clockify-jira-sync-v2.0.0-beta.1-macos-universal.zip", BrowserDownloadURL: "https://example.com/beta.zip", Size: 500}}},
		{TagName: "v1.8.7", PreRelease: false},
	})
	defer server.Close()

	u := New()
	u.SetBaseURL(server.URL)

	// With beta, should find the pre-release
	info, err := u.CheckForUpdate("1.8.7", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected beta update, got nil")
	}
	if !info.IsPreRelease {
		t.Error("expected pre-release flag")
	}
}

func TestGetLatestStable(t *testing.T) {
	server := mockGitHubServer([]ghRelease{
		{TagName: "v2.0.0-beta.1", PreRelease: true},
		{TagName: "v1.9.0", PreRelease: false,
			Assets: []ghAsset{{Name: "clockify-jira-sync-v1.9.0-macos-universal.zip", BrowserDownloadURL: "https://example.com/stable.zip", Size: 800}}},
		{TagName: "v1.8.7", PreRelease: false},
	})
	defer server.Close()

	u := New()
	u.SetBaseURL(server.URL)

	info, err := u.GetLatestStable("2.0.0-beta.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected stable info, got nil")
	}
	if info.Version != "1.9.0" {
		t.Errorf("expected version 1.9.0, got %s", info.Version)
	}
	if info.IsPreRelease {
		t.Error("expected stable release")
	}
}

func TestCheckForUpdate_DraftsSkipped(t *testing.T) {
	server := mockGitHubServer([]ghRelease{
		{TagName: "v99.0.0", Draft: true},
		{TagName: "v1.8.7", PreRelease: false},
	})
	defer server.Close()

	u := New()
	u.SetBaseURL(server.URL)

	info, err := u.CheckForUpdate("1.8.7", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Errorf("expected nil (draft skipped), got %+v", info)
	}
}
