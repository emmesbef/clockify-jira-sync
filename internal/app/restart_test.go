package app

import (
	"context"
	"reflect"
	"runtime"
	"testing"

	"jirafy-clockwork/internal/config"
)

func TestMacAppBundlePath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "bundle executable path",
			in:   "/Applications/JiraFy Clockwork.app/Contents/MacOS/jirafy-clockwork",
			want: "/Applications/JiraFy Clockwork.app",
		},
		{
			name: "non bundle path",
			in:   "/usr/local/bin/jirafy-clockwork",
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := macAppBundlePath(tc.in)
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestRelaunchExecutable_CommandSelection(t *testing.T) {
	orig := startDetachedProcess
	defer func() { startDetachedProcess = orig }()

	type call struct {
		name string
		args []string
	}
	var got []call
	startDetachedProcess = func(name string, args ...string) error {
		got = append(got, call{name: name, args: append([]string(nil), args...)})
		return nil
	}

	if runtime.GOOS == "darwin" {
		exePath := "/Applications/JiraFy Clockwork.app/Contents/MacOS/jirafy-clockwork"
		if err := relaunchExecutable(exePath); err != nil {
			t.Fatalf("relaunchExecutable returned error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected one process launch call, got %d", len(got))
		}
		if got[0].name != "open" {
			t.Fatalf("expected relaunch command 'open', got %q", got[0].name)
		}
		wantArgs := []string{"-n", "/Applications/JiraFy Clockwork.app"}
		if !reflect.DeepEqual(got[0].args, wantArgs) {
			t.Fatalf("expected args %v, got %v", wantArgs, got[0].args)
		}
		return
	}

	exePath := "/usr/local/bin/jirafy-clockwork"
	if err := relaunchExecutable(exePath); err != nil {
		t.Fatalf("relaunchExecutable returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one process launch call, got %d", len(got))
	}
	if got[0].name != exePath {
		t.Fatalf("expected relaunch command %q, got %q", exePath, got[0].name)
	}
	if len(got[0].args) != 0 {
		t.Fatalf("expected no args for executable relaunch, got %v", got[0].args)
	}
}

func TestBeforeCloseMarksWindowHidden(t *testing.T) {
	a := NewApp(&config.Config{}, "test")
	a.windowVisible = true

	prevent := a.BeforeClose(context.Background())
	if !prevent {
		t.Fatal("expected BeforeClose to intercept close and hide to tray")
	}

	if a.windowVisible {
		t.Fatal("expected BeforeClose to mark window as hidden")
	}
}

func TestBeforeCloseAllowsQuitWhenRequested(t *testing.T) {
	a := NewApp(&config.Config{}, "test")
	a.windowVisible = true
	a.quitRequested = true

	prevent := a.BeforeClose(context.Background())
	if prevent {
		t.Fatal("expected BeforeClose to allow close when quit was explicitly requested")
	}
}

func TestExtractTicketKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "exact key prefix", input: "PROJ-123 Investigate bug", want: "PROJ-123"},
		{name: "lowercase key", input: "proj-88 review", want: "PROJ-88"},
		{name: "key in middle", input: "Work on OPS-7 today", want: "OPS-7"},
		{name: "no key", input: "team sync meeting", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractTicketKey(tc.input)
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
