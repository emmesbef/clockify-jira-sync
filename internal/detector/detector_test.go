package detector

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"clockify-jira-sync/internal/models"
)

func TestExtractTicketKey(t *testing.T) {
	d := NewDetector(0)

	tests := []struct {
		name     string
		branch   string
		expected string
	}{
		{"Standard feature branch", "feature/PROJ-123-new-login", "PROJ-123"},
		{"Standard bugfix branch", "bugfix/APP-45-fix-crash", "APP-45"},
		{"Key at beginning", "CORE-999_refactor_db", "CORE-999"},
		{"Multiple numbers", "TKT-12345-something", "TKT-12345"},
		{"No key", "main", ""},
		{"Lowercase key does not match", "feature/proj-123", ""},
		{"Letters only do not match", "feature/PROJ-ABC", ""},
		{"Embedded key is extracted", "release-branch-QA-12-final", "QA-12"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := d.extractTicketKey(tt.branch); got != tt.expected {
				t.Errorf("extractTicketKey(%q) = %q, want %q", tt.branch, got, tt.expected)
			}
		})
	}
}

func TestDeduplicateWorkspaces(t *testing.T) {
	d := NewDetector(0)

	input := []ideWorkspace{
		{path: "/repo/a", ide: "VS Code"},
		{path: "/repo/b", ide: "VS Code"},
		{path: "/repo/a", ide: "VS Code"},
		{path: "/repo/c", ide: "Visual Studio"},
	}

	result := d.deduplicateWorkspaces(input)
	if len(result) != 3 {
		t.Fatalf("expected 3 workspaces after deduplication, got %d", len(result))
	}

	if result[0].path != "/repo/a" || result[1].path != "/repo/b" || result[2].path != "/repo/c" {
		t.Fatalf("deduplicateWorkspaces preserved unexpected order: %+v", result)
	}
}

func TestExtractPathsFromCmdLineFiltersFlagsAndNonRepos(t *testing.T) {
	repo := createGitRepo(t, "feature/APP-45-cmdline")
	nonRepo := filepath.Join(t.TempDir(), "plain-folder")
	if err := os.MkdirAll(nonRepo, 0o755); err != nil {
		t.Fatalf("failed to create non-repo directory: %v", err)
	}

	d := NewDetector(time.Second)
	line := fmt.Sprintf("user 123 0.0 0.1 Code --reuse-window %s %s -g README.md", repo, nonRepo)
	paths := d.extractPathsFromCmdLine(line)

	if len(paths) != 1 || paths[0] != repo {
		t.Fatalf("expected only git repo path %q, got %v", repo, paths)
	}
}

func TestGitRepoHelpers(t *testing.T) {
	repo := createGitRepo(t, "feature/CORE-999-helpers")
	nonRepo := filepath.Join(t.TempDir(), "plain-folder")
	if err := os.MkdirAll(nonRepo, 0o755); err != nil {
		t.Fatalf("failed to create non-repo directory: %v", err)
	}

	d := NewDetector(time.Second)
	if !d.isGitRepo(repo) {
		t.Fatalf("expected %q to be detected as a git repository", repo)
	}
	if d.isGitRepo(nonRepo) {
		t.Fatalf("expected %q not to be detected as a git repository", nonRepo)
	}
	if branch := d.getGitBranch(repo); branch != "feature/CORE-999-helpers" {
		t.Fatalf("expected branch feature/CORE-999-helpers, got %q", branch)
	}
	if branch := d.getGitBranch(nonRepo); branch != "" {
		t.Fatalf("expected non-repo branch lookup to be empty, got %q", branch)
	}
}

func TestFindIDEWorkspacesUsesProcessList(t *testing.T) {
	skipDetectorShellTestsOnWindows(t)

	repoFromProcess := createGitRepo(t, "feature/PROJ-101-process")
	home := t.TempDir()
	t.Setenv("HOME", home)
	psOutputFile := installFakePS(t)
	setFakePSOutput(t, psOutputFile, strings.Join([]string{
		"Electron Visual Studio Code helper",
		fmt.Sprintf("/Applications/Visual Studio Code.app/Contents/MacOS/Electron --folder-uri=file://%s", repoFromProcess),
		fmt.Sprintf("/Applications/Visual Studio Code.app/Contents/MacOS/Electron --folder-uri=file://%s", filepath.Join(t.TempDir(), "not-a-repo")),
	}, "\n"))

	d := NewDetector(time.Second)
	workspaces := d.findIDEWorkspaces()

	if len(workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d (%+v)", len(workspaces), workspaces)
	}

	assertSameStringSet(t, workspacePaths(workspaces), []string{repoFromProcess})
	for _, ws := range workspaces {
		if ws.ide != "VS Code" {
			t.Fatalf("expected IDE label to be VS Code, got %q", ws.ide)
		}
	}
}

func TestScanUpdatesDetectionsAndDeduplicatesNotifications(t *testing.T) {
	skipDetectorShellTestsOnWindows(t)

	repo := createGitRepo(t, "feature/PROJ-101-initial")
	home := t.TempDir()
	t.Setenv("HOME", home)
	psOutputFile := installFakePS(t)
	setFakePSOutput(t, psOutputFile, fmt.Sprintf("/Applications/Visual Studio Code.app/Contents/MacOS/Electron --folder-uri=file://%s\n", repo))

	d := NewDetector(10 * time.Millisecond)
	notifications := make(chan models.BranchDetection, 4)
	d.OnDetection(func(det models.BranchDetection) {
		notifications <- det
	})

	d.scan()
	first := waitForDetection(t, notifications)
	if first.TicketKey != "PROJ-101" || first.BranchName != "feature/PROJ-101-initial" || first.RepoPath != repo {
		t.Fatalf("unexpected first detection: %+v", first)
	}

	detections := d.GetDetections()
	if len(detections) != 1 || detections[0].TicketKey != "PROJ-101" {
		t.Fatalf("expected one stored detection for PROJ-101, got %+v", detections)
	}

	d.scan()
	assertNoDetection(t, notifications)

	checkoutBranch(t, repo, "bugfix/PROJ-202-follow-up")
	d.scan()
	second := waitForDetection(t, notifications)
	if second.TicketKey != "PROJ-202" || second.BranchName != "bugfix/PROJ-202-follow-up" {
		t.Fatalf("unexpected updated detection: %+v", second)
	}

	checkoutBranch(t, repo, "main")
	d.scan()
	assertNoDetection(t, notifications)
	if detections := d.GetDetections(); len(detections) != 0 {
		t.Fatalf("expected detections to be cleared when branch has no ticket, got %+v", detections)
	}
}

func TestStartPerformsInitialScanAndStopsOnCancel(t *testing.T) {
	skipDetectorShellTestsOnWindows(t)

	repo := createGitRepo(t, "feature/PROJ-303-start")
	home := t.TempDir()
	t.Setenv("HOME", home)
	psOutputFile := installFakePS(t)
	setFakePSOutput(t, psOutputFile, fmt.Sprintf("/Applications/Visual Studio Code.app/Contents/MacOS/Electron --folder-uri=file://%s\n", repo))

	d := NewDetector(25 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		d.Start(ctx)
		close(done)
	}()

	waitForCondition(t, time.Second, func() bool {
		detections := d.GetDetections()
		return len(detections) == 1 && detections[0].TicketKey == "PROJ-303"
	}, "detector did not perform initial scan")

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("detector did not stop after context cancellation")
	}
}

func TestIsProtectedPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tests := []struct {
		name      string
		path      string
		protected bool
	}{
		{"Documents dir", filepath.Join(home, "Documents"), true},
		{"Inside Documents", filepath.Join(home, "Documents", "project"), true},
		{"Desktop dir", filepath.Join(home, "Desktop"), true},
		{"Downloads dir", filepath.Join(home, "Downloads"), true},
		{"Library dir", filepath.Join(home, "Library"), true},
		{"Inside Library", filepath.Join(home, "Library", "Application Support"), true},
		{"iCloud Drive path", filepath.Join(home, "Library", "Mobile Documents", "com~apple~CloudDocs"), true},
		{"Movies dir", filepath.Join(home, "Movies"), true},
		{"Music dir", filepath.Join(home, "Music"), true},
		{"Pictures dir", filepath.Join(home, "Pictures"), true},
		{"External volume", "/Volumes/iDrive/project", true},
		{"External drive", "/Volumes/Backup/repo", true},
		{"Home itself", home, false},
		{"Projects dir", filepath.Join(home, "Projects", "my-app"), false},
		{"Git dir", filepath.Join(home, "git", "my-repo"), false},
		{"Similar prefix", filepath.Join(home, "DocumentsBackup"), false},
		{"Unrelated path", "/tmp/some-repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isProtectedPath(tt.path); got != tt.protected {
				t.Errorf("isProtectedPath(%q) = %v, want %v", tt.path, got, tt.protected)
			}
		})
	}
}

func TestExtractFolderURIs(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected []string
	}{
		{
			"Standard folder-uri",
			"/Applications/Visual Studio Code.app/Contents/MacOS/Electron --folder-uri=file:///Users/dev/project",
			[]string{"/Users/dev/project"},
		},
		{
			"Space encoded in path",
			"Electron --folder-uri=file:///Users/dev/my%20project",
			[]string{"/Users/dev/my project"},
		},
		{
			"Multiple folder-uris",
			"Electron --folder-uri=file:///repo/a --folder-uri=file:///repo/b",
			[]string{"/repo/a", "/repo/b"},
		},
		{
			"Windows drive-letter path",
			`Electron --folder-uri=file:///C:/Users/dev/project`,
			[]string{`C:/Users/dev/project`},
		},
		{
			"Windows path with space encoded",
			`Electron --folder-uri=file:///C:/Users/dev/my%20project`,
			[]string{`C:/Users/dev/my project`},
		},
		{
			"No folder-uri",
			"Electron --reuse-window /some/path",
			nil,
		},
		{
			"Empty line",
			"",
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFolderURIs(tt.line)
			if len(got) != len(tt.expected) {
				t.Fatalf("extractFolderURIs() = %v, want %v", got, tt.expected)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("extractFolderURIs()[%d] = %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func skipDetectorShellTestsOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("detector shell-based tests require POSIX process utilities")
	}
}

func createGitRepo(t *testing.T, branch string) string {
	t.Helper()
	dirName := strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(branch)
	return createNamedGitRepo(t, dirName, branch)
}

func createNamedGitRepo(t *testing.T, dirName, branch string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), dirName)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("failed to create repo directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("test repo\n"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "-m", "initial commit")
	runGit(t, root, "checkout", "-B", branch)

	return root
}

func checkoutBranch(t *testing.T, repo, branch string) {
	t.Helper()
	runGit(t, repo, "checkout", "-B", branch)
}

func runGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmdArgs := append([]string{"-C", repo}, args...)
	cmd := exec.Command("git", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}

func installFakePS(t *testing.T) string {
	t.Helper()
	fakeBin := t.TempDir()
	psOutputFile := filepath.Join(t.TempDir(), "ps-output.txt")
	if err := os.WriteFile(psOutputFile, nil, 0o644); err != nil {
		t.Fatalf("failed to create ps output file: %v", err)
	}

	scriptPath := filepath.Join(fakeBin, "ps")
	script := "#!/bin/sh\ncat \"$FAKE_PS_OUTPUT_FILE\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake ps script: %v", err)
	}

	t.Setenv("FAKE_PS_OUTPUT_FILE", psOutputFile)
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return psOutputFile
}

func setFakePSOutput(t *testing.T, path, output string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(output), 0o644); err != nil {
		t.Fatalf("failed to update fake ps output: %v", err)
	}
}

func workspacePaths(workspaces []ideWorkspace) []string {
	paths := make([]string, 0, len(workspaces))
	for _, ws := range workspaces {
		paths = append(paths, ws.path)
	}
	return paths
}

func assertSameStringSet(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d paths, got %d (%v)", len(want), len(got), got)
	}

	seen := make(map[string]int, len(got))
	for _, item := range got {
		seen[item]++
	}
	for _, item := range want {
		seen[item]--
	}
	for item, count := range seen {
		if count != 0 {
			t.Fatalf("unexpected path counts for %q: got=%v want=%v", item, got, want)
		}
	}
}

func waitForDetection(t *testing.T, ch <-chan models.BranchDetection) models.BranchDetection {
	t.Helper()
	select {
	case det := <-ch:
		return det
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for detection callback")
		return models.BranchDetection{}
	}
}

func assertNoDetection(t *testing.T, ch <-chan models.BranchDetection) {
	t.Helper()
	select {
	case det := <-ch:
		t.Fatalf("unexpected detection callback: %+v", det)
	case <-time.After(150 * time.Millisecond):
	}
}

func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal(message)
}
