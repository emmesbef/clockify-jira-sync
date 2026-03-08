package detector

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
	// Use ps to find VS Code Electron processes
	out, err := exec.Command("ps", "-eo", "args").Output()
	if err != nil {
		return nil
	}

	var workspaces []ideWorkspace
	for _, line := range strings.Split(string(out), "\n") {
		// Only match actual VS Code Electron processes
		if !strings.Contains(line, "Visual Studio Code") && !strings.Contains(line, "Contents/MacOS/Electron") {
			continue
		}

		// Extract workspace paths from --folder-uri arguments
		for _, uri := range extractFolderURIs(line) {
			if !isProtectedPath(uri) && d.isGitRepo(uri) {
				workspaces = append(workspaces, ideWorkspace{path: uri, ide: "VS Code"})
			}
		}
	}

	return d.deduplicateWorkspaces(workspaces)
}

// extractFolderURIs extracts workspace directory paths from VS Code command-line
// arguments. It handles both --folder-uri=file:///path and bare /path forms.
// On Windows, file URIs use the form file:///C:/path, so the leading slash
// before the drive letter is stripped to produce a valid Windows path.
func extractFolderURIs(line string) []string {
	var paths []string

	// Match --folder-uri=file:///path (primary mechanism VS Code uses)
	for _, part := range strings.Fields(line) {
		const prefix = "--folder-uri=file://"
		if strings.HasPrefix(part, prefix) {
			p := strings.TrimPrefix(part, prefix)
			p = strings.ReplaceAll(p, "%20", " ")
			// Windows file URIs encode drive-letter paths as /C:/path.
			// Strip the leading slash so the path is usable on Windows.
			if len(p) >= 3 && p[0] == '/' && isWindowsDriveLetter(p[1]) && p[2] == ':' {
				p = p[1:]
			}
			if p != "" {
				paths = append(paths, p)
			}
		}
	}

	return paths
}

// isWindowsDriveLetter reports whether b is an ASCII letter (A-Z or a-z).
func isWindowsDriveLetter(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func (d *Detector) findVSCodeWorkspacesLinux() []ideWorkspace {
	return d.findVSCodeWorkspacesDarwin() // Same ps/args approach on Linux
}

func (d *Detector) findVSCodeWorkspacesWindows() []ideWorkspace {
	out, err := exec.Command(
		"powershell",
		"-NoProfile",
		"-Command",
		`Get-CimInstance Win32_Process -Filter "Name = 'Code.exe'" | Select-Object -ExpandProperty CommandLine`,
	).Output()
	if err != nil {
		return nil
	}

	var workspaces []ideWorkspace
	for _, line := range strings.Split(string(out), "\n") {
		for _, uri := range extractFolderURIs(line) {
			if d.isGitRepo(uri) {
				workspaces = append(workspaces, ideWorkspace{path: uri, ide: "VS Code"})
			}
		}
	}
	return workspaces
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
			// On macOS, skip TCC-protected directories; on other OSes, include them.
			if runtime.GOOS != "darwin" || !isProtectedPath(part) {
				paths = append(paths, part)
			}
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

// isProtectedPath returns true if the path is inside a macOS TCC-protected
// directory or any other directory that the app should not access.
// Uses a blocklist of known protected directories AND requires paths under ~/
// to NOT be in the home root (e.g. ~/someproject is blocked, only explicit
// well-known dev paths or paths outside ~ are allowed).
func isProtectedPath(path string) bool {
	home := homeDir()
	if home == "" {
		return false
	}

	// Block ALL known TCC-protected directories
	protected := []string{
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "Library"),
		filepath.Join(home, "Movies"),
		filepath.Join(home, "Music"),
		filepath.Join(home, "Pictures"),
	}
	for _, dir := range protected {
		if strings.HasPrefix(path, dir+"/") || path == dir {
			return true
		}
	}

	// Block any /Volumes/ path (external drives, iDrive, cloud mounts)
	if strings.HasPrefix(path, "/Volumes/") {
		return true
	}

	return false
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
