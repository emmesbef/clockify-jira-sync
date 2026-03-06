package detector

import (
	"testing"
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
		{"Lowercase key (should fail match per regex)", "feature/proj-123", ""},
		{"Letters only", "feature/PROJ-ABC", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.extractTicketKey(tt.branch)
			if got != tt.expected {
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
		{path: "/repo/a", ide: "VS Code"}, // duplicate
		{path: "/repo/c", ide: "Visual Studio"},
	}

	result := d.deduplicateWorkspaces(input)

	if len(result) != 3 {
		t.Errorf("Expected 3 workspaces after deduplication, got %d", len(result))
	}
}
