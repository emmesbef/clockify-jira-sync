package clockify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetCurrentUser(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			t.Errorf("Expected path /user, got %s", r.URL.Path)
		}

		response := userResponse{ID: "test-user-id", Email: "test@example.com"}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	client := NewClient("test-key", "test-ws")
	client.baseURL = mockServer.URL // Inject mock server URL

	err := client.Init()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if client.GetUserID() != "test-user-id" {
		t.Errorf("Expected test-user-id, got %s", client.GetUserID())
	}
}

func TestStartTimer(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workspaces/test-ws/time-entries" {
			t.Errorf("Expected path /workspaces/test-ws/time-entries, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		response := timeEntryResponse{
			ID:          "new-entry-123",
			Description: "PROJ-123 Working on tests",
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	client := NewClient("test-key", "test-ws")
	client.baseURL = mockServer.URL

	entryID, err := client.StartTimer("PROJ-123 Working on tests", "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if entryID != "new-entry-123" {
		t.Errorf("Expected new-entry-123, got %s", entryID)
	}
}

func TestGetTimeEntries(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workspaces/ws-1/user/usr-1/time-entries" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("page-size") != "100" {
			t.Errorf("Expected page-size 100")
		}

		response := []timeEntryResponse{
			{
				ID:          "entry-1",
				Description: "Task 1",
				TimeInterval: struct {
					Start    string `json:"start"`
					End      string `json:"end"`
					Duration string `json:"duration"`
				}{
					Start: "2026-03-05T10:00:00Z",
					End:   "2026-03-05T11:00:00Z",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	client := NewClient("test-key", "ws-1")
	client.baseURL = mockServer.URL
	client.userID = "usr-1"

	start := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 5, 23, 59, 59, 0, time.UTC)

	entries, err := client.GetTimeEntries(start, end)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}
	if entries[0].ClockifyID != "entry-1" {
		t.Errorf("Expected ClockifyID entry-1, got %s", entries[0].ClockifyID)
	}
	if entries[0].Duration != 3600 {
		t.Errorf("Expected duration 3600s, got %d", entries[0].Duration)
	}
}

func TestGetWorkspaces(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workspaces" {
			t.Errorf("Expected path /workspaces, got %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-key" {
			t.Errorf("Expected API key header")
		}

		json.NewEncoder(w).Encode([]WorkspaceInfo{
			{ID: "ws-1", Name: "Workspace One"},
			{ID: "ws-2", Name: "Workspace Two"},
		})
	}))
	defer mockServer.Close()

	client := NewClient("test-key", "")
	client.baseURL = mockServer.URL

	workspaces, err := client.GetWorkspaces()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(workspaces) != 2 {
		t.Fatalf("Expected 2 workspaces, got %d", len(workspaces))
	}
	if workspaces[0].ID != "ws-1" || workspaces[0].Name != "Workspace One" {
		t.Errorf("Unexpected first workspace: %+v", workspaces[0])
	}
}
