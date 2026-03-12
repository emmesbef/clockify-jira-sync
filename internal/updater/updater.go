package updater

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"jirafy-clockwork/internal/models"
)

const (
	defaultGitLabAPI = "https://gitlab.com/api/v4"
	projectPath      = "level-87/clockify-jira-sync"
)

// Updater checks for and applies application updates from GitLab Releases.
type Updater struct {
	baseURL    string
	httpClient *http.Client
}

// New creates an Updater with default settings.
func New() *Updater {
	return &Updater{
		baseURL:    defaultGitLabAPI,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// SetBaseURL overrides the GitLab API base URL (for tests).
func (u *Updater) SetBaseURL(url string) {
	u.baseURL = url
}

type gitlabRelease struct {
	TagName         string             `json:"tag_name"`
	Name            string             `json:"name"`
	Description     string             `json:"description"`
	ReleasedAt      string             `json:"released_at"`
	UpcomingRelease bool               `json:"upcoming_release"`
	Assets          gitlabReleaseAsset `json:"assets"`
}

type gitlabReleaseAsset struct {
	Links []gitlabAssetLink `json:"links"`
}

type gitlabAssetLink struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	DirectAssetURL string `json:"direct_asset_url"`
}

type releaseAsset struct {
	Name        string
	DownloadURL string
	Size        int64
}

func (u *Updater) releasesURL() string {
	return fmt.Sprintf("%s/projects/%s/releases", u.baseURL, url.PathEscape(projectPath))
}

func (u *Updater) fetchReleases() ([]gitlabRelease, error) {
	req, err := http.NewRequest("GET", u.releasesURL(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "jirafy-clockwork-updater")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitLab API error %d", resp.StatusCode)
	}

	var releases []gitlabRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse releases: %w", err)
	}
	return releases, nil
}

func resolveAssetURL(apiBase, assetURL string) string {
	parsed, err := url.Parse(assetURL)
	if err != nil {
		return assetURL
	}
	if parsed.IsAbs() {
		return assetURL
	}
	base, err := url.Parse(apiBase)
	if err != nil {
		return assetURL
	}
	base = &url.URL{Scheme: base.Scheme, Host: base.Host}
	return base.ResolveReference(parsed).String()
}

func formatReleasedAt(raw string) string {
	if raw == "" {
		return ""
	}
	publishedAt, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return publishedAt.Format(time.RFC3339)
}

// CheckForUpdate queries GitLab Releases and returns info about the latest
// available update, or nil if the current version is up to date.
func (u *Updater) CheckForUpdate(currentVersion string, includeBeta bool) (*models.UpdateInfo, error) {
	releases, err := u.fetchReleases()
	if err != nil {
		return nil, err
	}

	// Find the best candidate release
	var best *gitlabRelease
	for i := range releases {
		r := &releases[i]
		if r.UpcomingRelease {
			continue
		}
		isPreRelease := IsPreReleaseVersion(r.TagName)
		if isPreRelease && !includeBeta {
			continue
		}
		ver := normalizeVersion(r.TagName)
		if CompareVersions(ver, normalizeVersion(currentVersion)) > 0 {
			if best == nil || CompareVersions(normalizeVersion(r.TagName), normalizeVersion(best.TagName)) > 0 {
				best = r
			}
		}
	}

	if best == nil {
		return nil, nil // up to date
	}

	asset := pickAsset(best.Assets.Links)
	downloadURL := ""
	var size int64
	if asset != nil {
		downloadURL = resolveAssetURL(u.baseURL, asset.DownloadURL)
		size = asset.Size
	}

	return &models.UpdateInfo{
		Version:      normalizeVersion(best.TagName),
		IsPreRelease: IsPreReleaseVersion(best.TagName),
		DownloadURL:  downloadURL,
		ReleaseNotes: best.Description,
		Size:         size,
		PublishedAt:  formatReleasedAt(best.ReleasedAt),
	}, nil
}

// GetLatestStable returns the latest non-prerelease version, used for
// forcing downgrade when beta channel is disabled.
func (u *Updater) GetLatestStable(currentVersion string) (*models.UpdateInfo, error) {
	releases, err := u.fetchReleases()
	if err != nil {
		return nil, err
	}

	var best *gitlabRelease
	for i := range releases {
		r := &releases[i]
		if r.UpcomingRelease || IsPreReleaseVersion(r.TagName) {
			continue
		}
		if best == nil || CompareVersions(normalizeVersion(r.TagName), normalizeVersion(best.TagName)) > 0 {
			best = r
		}
	}

	if best == nil {
		return nil, nil
	}

	asset := pickAsset(best.Assets.Links)
	downloadURL := ""
	var size int64
	if asset != nil {
		downloadURL = resolveAssetURL(u.baseURL, asset.DownloadURL)
		size = asset.Size
	}

	return &models.UpdateInfo{
		Version:      normalizeVersion(best.TagName),
		IsPreRelease: false,
		DownloadURL:  downloadURL,
		ReleaseNotes: best.Description,
		Size:         size,
		PublishedAt:  formatReleasedAt(best.ReleasedAt),
	}, nil
}

// DownloadAndApply downloads the update ZIP and replaces the running binary.
func (u *Updater) DownloadAndApply(info *models.UpdateInfo) error {
	if info.DownloadURL == "" {
		return fmt.Errorf("no download URL for this platform")
	}

	// Download to temp file
	resp, err := u.httpClient.Get(info.DownloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "clockify-update-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to save update: %w", err)
	}
	tmpFile.Close()

	// Extract and apply
	if runtime.GOOS == "darwin" {
		return applyMacOS(tmpPath)
	}
	return applyWindows(tmpPath)
}

// applyMacOS extracts the .app bundle from the ZIP and replaces the running app.
func applyMacOS(zipPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}

	// The executable is at AppName.app/Contents/MacOS/binary
	// We need to replace the entire .app bundle
	appBundle := exe
	for i := 0; i < 3; i++ {
		appBundle = filepath.Dir(appBundle)
	}
	if !strings.HasSuffix(appBundle, ".app") {
		// Dev mode — just replace the binary
		return extractBinary(zipPath, exe)
	}

	parentDir := filepath.Dir(appBundle)
	backupPath := appBundle + ".backup"

	// Rename current app to backup
	if err := os.Rename(appBundle, backupPath); err != nil {
		return fmt.Errorf("failed to backup current app: %w", err)
	}

	// Extract new .app from ZIP
	if err := extractApp(zipPath, parentDir); err != nil {
		// Restore backup on failure
		os.Rename(backupPath, appBundle)
		return fmt.Errorf("failed to extract update: %w", err)
	}

	os.RemoveAll(backupPath)
	return nil
}

// applyWindows extracts the .exe from the ZIP and replaces the running binary.
func applyWindows(zipPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}

	exeDir := filepath.Dir(exe)
	backupPath := exe + ".backup"

	// On Windows, rename running exe (Windows allows this)
	if err := os.Rename(exe, backupPath); err != nil {
		return fmt.Errorf("failed to backup current exe: %w", err)
	}

	if err := extractToDir(zipPath, exeDir); err != nil {
		os.Rename(backupPath, exe)
		return fmt.Errorf("failed to extract update: %w", err)
	}

	os.Remove(backupPath)
	return nil
}

// extractApp finds a .app bundle in the ZIP and extracts it to destDir.
func extractApp(zipPath, destDir string) error {
	return extractToDir(zipPath, destDir)
}

// extractBinary finds the main binary in the ZIP and writes it to destPath (dev mode).
func extractBinary(zipPath, destPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := filepath.Base(f.Name)
		if name == "jirafy-clockwork" || name == "jirafy-clockwork.exe" {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
			if err != nil {
				return err
			}
			defer out.Close()

			_, err = io.Copy(out, rc)
			return err
		}
	}
	return fmt.Errorf("binary not found in update archive")
}

// extractToDir extracts all files from a ZIP to the destination directory.
func extractToDir(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(destDir, f.Name)

		// Prevent zip slip
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(target, f.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, copyErr := io.Copy(out, rc)
		rc.Close()
		out.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

// pickAsset selects the correct platform asset from a release.
func pickAsset(assets []gitlabAssetLink) *releaseAsset {
	var suffix string
	switch runtime.GOOS {
	case "darwin":
		suffix = "macos-universal.zip"
	case "windows":
		suffix = "windows-amd64.zip"
	default:
		return nil
	}

	for i := range assets {
		if strings.HasSuffix(assets[i].Name, suffix) {
			assetURL := strings.TrimSpace(assets[i].DirectAssetURL)
			if assetURL == "" {
				assetURL = strings.TrimSpace(assets[i].URL)
			}
			if assetURL == "" {
				continue
			}
			return &releaseAsset{
				Name:        assets[i].Name,
				DownloadURL: assetURL,
				Size:        0,
			}
		}
	}
	return nil
}

// normalizeVersion strips a leading "v" prefix from a version string.
func normalizeVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

// CompareVersions compares two semantic version strings (major.minor.patch).
// Returns >0 if a > b, <0 if a < b, 0 if equal.
func CompareVersions(a, b string) int {
	aParts := parseVersion(a)
	bParts := parseVersion(b)
	for i := 0; i < 3; i++ {
		if aParts[i] != bParts[i] {
			return aParts[i] - bParts[i]
		}
	}
	return 0
}

// IsPreReleaseVersion returns true if the version string contains a pre-release
// suffix (e.g., "1.9.0-beta.1").
func IsPreReleaseVersion(version string) bool {
	v := normalizeVersion(version)
	return strings.Contains(v, "-")
}

func parseVersion(v string) [3]int {
	v = normalizeVersion(v)
	// Strip pre-release suffix for comparison (e.g., "1.9.0-beta.1" → "1.9.0")
	if idx := strings.Index(v, "-"); idx >= 0 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	var result [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		result[i], _ = strconv.Atoi(parts[i])
	}
	return result
}
