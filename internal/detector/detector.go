package detector

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"clockify-jira-sync/internal/models"
)

// jiraKeyPattern matches Jira ticket keys like PROJ-123
var jiraKeyPattern = regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`)

// Detector monitors running IDEs and extracts Jira ticket keys from git branches
type Detector struct {
	mu           sync.RWMutex
	detections   map[string]models.BranchDetection // keyed by repoPath
	onDetection  func(models.BranchDetection)
	pollInterval time.Duration
	lastNotified map[string]string // repoPath -> lastTicketKey (dedup)
}

// NewDetector creates a new IDE/branch detector
func NewDetector(pollInterval time.Duration) *Detector {
	return &Detector{
		detections:   make(map[string]models.BranchDetection),
		lastNotified: make(map[string]string),
		pollInterval: pollInterval,
	}
}

// OnDetection sets the callback for new branch detections
func (d *Detector) OnDetection(fn func(models.BranchDetection)) {
	d.onDetection = fn
}

// GetDetections returns all currently detected branches
func (d *Detector) GetDetections() []models.BranchDetection {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make([]models.BranchDetection, 0, len(d.detections))
	for _, det := range d.detections {
		result = append(result, det)
	}
	return result
}

// Start begins the background polling loop
func (d *Detector) Start(ctx context.Context) {
	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	// Do an initial scan
	d.scan()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.scan()
		}
	}
}

func (d *Detector) scan() {
	workspaces := d.findIDEWorkspaces()

	d.mu.Lock()
	defer d.mu.Unlock()

	// Clear old detections
	newDetections := make(map[string]models.BranchDetection)

	for _, ws := range workspaces {
		branch := d.getGitBranch(ws.path)
		if branch == "" {
			continue
		}

		ticketKey := d.extractTicketKey(branch)
		if ticketKey == "" {
			continue
		}

		det := models.BranchDetection{
			TicketKey:  ticketKey,
			BranchName: branch,
			RepoPath:   ws.path,
			IDE:        ws.ide,
		}
		newDetections[ws.path] = det

		// Only notify if this is new or different
		if d.lastNotified[ws.path] != ticketKey {
			d.lastNotified[ws.path] = ticketKey
			if d.onDetection != nil {
				go d.onDetection(det)
			}
		}
	}

	d.detections = newDetections
}

type ideWorkspace struct {
	path string
	ide  string
}

func (d *Detector) findIDEWorkspaces() []ideWorkspace {
	var workspaces []ideWorkspace

	switch runtime.GOOS {
	case "darwin":
		workspaces = append(workspaces, d.findVSCodeWorkspacesDarwin()...)
	case "linux":
		workspaces = append(workspaces, d.findVSCodeWorkspacesLinux()...)
	case "windows":
		workspaces = append(workspaces, d.findVSCodeWorkspacesWindows()...)
	}

	return workspaces
}

func (d *Detector) findVSCodeWorkspacesDarwin() []ideWorkspace {
	// Use ps to find VS Code processes with their working directories
	out, err := exec.Command("ps", "aux").Output()
	if err != nil {
		return nil
	}

	var workspaces []ideWorkspace
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Electron") && strings.Contains(line, "Visual Studio Code") {
			continue // Skip the Electron helper, looking for the folder arg
		}

		// VS Code often has --folder-uri or the path as argument
		if strings.Contains(line, "code") || strings.Contains(line, "Code") {
			paths := d.extractPathsFromCmdLine(line)
			for _, p := range paths {
				workspaces = append(workspaces, ideWorkspace{path: p, ide: "VS Code"})
			}
		}
	}

	// Also try lsof approach — find open files/dirs by VS Code
	// Fallback: check recently opened workspaces from VS Code storage
	recentPaths := d.getVSCodeRecentPaths()
	for _, p := range recentPaths {
		workspaces = append(workspaces, ideWorkspace{path: p, ide: "VS Code"})
	}

	return d.deduplicateWorkspaces(workspaces)
}

func (d *Detector) findVSCodeWorkspacesLinux() []ideWorkspace {
	return d.findVSCodeWorkspacesDarwin() // Same approach on Linux
}

func (d *Detector) findVSCodeWorkspacesWindows() []ideWorkspace {
	out, err := exec.Command("tasklist", "/v", "/fi", "IMAGENAME eq Code.exe").Output()
	if err != nil {
		return nil
	}

	var workspaces []ideWorkspace
	if strings.Contains(string(out), "Code.exe") {
		recentPaths := d.getVSCodeRecentPaths()
		for _, p := range recentPaths {
			workspaces = append(workspaces, ideWorkspace{path: p, ide: "VS Code"})
		}
	}
	return workspaces
}

func (d *Detector) getVSCodeRecentPaths() []string {
	// Try to read VS Code's recent workspaces from storage.json
	var storagePaths []string

	switch runtime.GOOS {
	case "darwin":
		storagePaths = []string{
			fmt.Sprintf("%s/Library/Application Support/Code/storage.json", homeDir()),
			fmt.Sprintf("%s/Library/Application Support/Code/User/globalStorage/storage.json", homeDir()),
		}
	case "linux":
		storagePaths = []string{
			fmt.Sprintf("%s/.config/Code/storage.json", homeDir()),
		}
	case "windows":
		storagePaths = []string{
			fmt.Sprintf("%s\\AppData\\Roaming\\Code\\storage.json", homeDir()),
		}
	}

	var paths []string
	for _, sp := range storagePaths {
		extracted := d.parseVSCodeStorage(sp)
		paths = append(paths, extracted...)
	}
	return paths
}

func (d *Detector) parseVSCodeStorage(path string) []string {
	// Simplified: just try to find folder paths from VS Code storage
	// In practice, this reads the JSON and extracts openedPathsList
	out, err := exec.Command("cat", path).Output()
	if err != nil {
		return nil
	}

	var paths []string
	content := string(out)
	// Look for file:// URIs that point to directories
	parts := strings.Split(content, "file://")
	for _, part := range parts[1:] { // skip first empty element
		idx := strings.IndexAny(part, `"',}]`)
		if idx > 0 {
			p := part[:idx]
			// URL-decode basic characters
			p = strings.ReplaceAll(p, "%20", " ")
			if d.isGitRepo(p) {
				paths = append(paths, p)
			}
		}
	}
	return paths
}

func (d *Detector) extractPathsFromCmdLine(line string) []string {
	// Extract file paths from command line arguments
	var paths []string
	parts := strings.Fields(line)
	for _, part := range parts {
		// Skip flags
		if strings.HasPrefix(part, "-") {
			continue
		}
		// Check if it looks like a path and is a git repo
		if (strings.HasPrefix(part, "/") || strings.HasPrefix(part, "~")) && d.isGitRepo(part) {
			paths = append(paths, part)
		}
	}
	return paths
}

func (d *Detector) isGitRepo(path string) bool {
	_, err := exec.Command("git", "-C", path, "rev-parse", "--git-dir").Output()
	return err == nil
}

func (d *Detector) getGitBranch(repoPath string) string {
	out, err := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (d *Detector) extractTicketKey(branch string) string {
	match := jiraKeyPattern.FindString(branch)
	return match
}

func (d *Detector) deduplicateWorkspaces(workspaces []ideWorkspace) []ideWorkspace {
	seen := make(map[string]bool)
	var result []ideWorkspace
	for _, ws := range workspaces {
		if !seen[ws.path] {
			seen[ws.path] = true
			result = append(result, ws)
		}
	}
	return result
}

func homeDir() string {
	home, _ := exec.Command("sh", "-c", "echo $HOME").Output()
	return strings.TrimSpace(string(home))
}
