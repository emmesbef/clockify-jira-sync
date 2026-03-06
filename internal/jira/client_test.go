package jira

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSearchIssues(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/rest/api/2/search") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		jql := r.URL.Query().Get("jql")
		if !strings.Contains(jql, "test-query") {
			t.Errorf("Expected jql to contain 'test-query'")
		}

		resp := searchResponse{
			Total: 1,
			Issues: []issueResponse{
				{
					Key: "PROJ-123",
					Fields: issueFields{
						Summary:   "Test Ticket",
						Status:    statusField{Name: "In Progress"},
						IssueType: typeField{Name: "Task"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	client := NewClient(mockServer.URL, "test@example.com", "token")

	tickets, err := client.SearchIssues("test-query")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(tickets) != 1 {
		t.Fatalf("Expected 1 ticket, got %d", len(tickets))
	}
	if tickets[0].Key != "PROJ-123" {
		t.Errorf("Expected PROJ-123, got %s", tickets[0].Key)
	}
	if tickets[0].Status != "In Progress" {
		t.Errorf("Expected In Progress, got %s", tickets[0].Status)
	}
}

func TestAddWorklog(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/issue/PROJ-123/worklog") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		var req worklogRequest
		json.Unmarshal(body, &req)

		if req.TimeSpentSeconds != 3600 {
			t.Errorf("Expected 3600 seconds, got %d", req.TimeSpentSeconds)
		}
		if req.Comment != "Worked on PROJ-123" {
			t.Errorf("Expected comment 'Worked on PROJ-123'")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(worklogResponse{ID: "wl-999"})
	}))
	defer mockServer.Close()

	client := NewClient(mockServer.URL, "test@example.com", "token")

	id, err := client.AddWorklog("PROJ-123", time.Now(), 3600, "Worked on PROJ-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if id != "wl-999" {
		t.Errorf("Expected id wl-999, got %s", id)
	}
}
