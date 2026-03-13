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
		if r.URL.Path != "/rest/api/3/search/jql" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)
		jql, _ := req["jql"].(string)
		if !strings.Contains(jql, "test") {
			t.Errorf("Expected jql to contain 'test', got %q", jql)
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

func TestSearchIssuesExactKeyUsesDirectLookup(t *testing.T) {
	var searchJQLs []string
	projectCalls := 0

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/search/jql":
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			_ = json.Unmarshal(body, &req)
			jql, _ := req["jql"].(string)
			searchJQLs = append(searchJQLs, jql)

			resp := searchResponse{
				Total: 1,
				Issues: []issueResponse{
					{
						Key: "PROJ-123",
						Fields: issueFields{
							Summary:   "Exact Match",
							Status:    statusField{Name: "Done"},
							IssueType: typeField{Name: "Task"},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		case "/rest/api/3/project":
			projectCalls++
			_ = json.NewEncoder(w).Encode([]jiraProjectInfo{{Key: "PROJ", Name: "Project"}})
		default:
			t.Fatalf("Unexpected path: %s", r.URL.Path)
		}
	}))
	defer mockServer.Close()

	client := NewClient(mockServer.URL, "test@example.com", "token")

	tickets, err := client.SearchIssues("proj-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(tickets) != 1 || tickets[0].Key != "PROJ-123" {
		t.Fatalf("Expected exact PROJ-123 match, got %+v", tickets)
	}

	if len(searchJQLs) != 1 {
		t.Fatalf("Expected exactly one JQL search call, got %d (%v)", len(searchJQLs), searchJQLs)
	}
	if !strings.Contains(searchJQLs[0], `key = "PROJ-123"`) {
		t.Fatalf("Expected direct key JQL lookup, got %q", searchJQLs[0])
	}
	if strings.Contains(searchJQLs[0], "status != Done") {
		t.Fatalf("Expected direct lookup to include all statuses, got %q", searchJQLs[0])
	}
	if projectCalls != 0 {
		t.Fatalf("Expected no project metadata calls on exact key lookup, got %d", projectCalls)
	}
}

func TestGetMyIssuesJQLIncludesAllStatuses(t *testing.T) {
	var gotJQL string

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/search/jql" {
			t.Fatalf("Unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		_ = json.Unmarshal(body, &req)
		gotJQL, _ = req["jql"].(string)
		_ = json.NewEncoder(w).Encode(searchResponse{Total: 0, Issues: []issueResponse{}})
	}))
	defer mockServer.Close()

	client := NewClient(mockServer.URL, "test@example.com", "token")

	if _, err := client.GetMyIssues(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !strings.Contains(gotJQL, "assignee=currentUser()") {
		t.Fatalf("Expected assignee filter in JQL, got %q", gotJQL)
	}
	if strings.Contains(gotJQL, "status != Done") {
		t.Fatalf("Expected GetMyIssues JQL to include all statuses, got %q", gotJQL)
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
		if req.Comment == nil || req.Comment.Content[0].Content[0].Text != "Worked on PROJ-123" {
			t.Errorf("Expected ADF comment with 'Worked on PROJ-123'")
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

func TestPing(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/myself" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"accountId": "abc123"})
	}))
	defer mockServer.Close()

	client := NewClient(mockServer.URL, "test@example.com", "token")
	if err := client.Ping(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}
