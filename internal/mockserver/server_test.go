package mockserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"clockify-jira-sync/internal/clockify"
	"clockify-jira-sync/internal/jira"
)

func TestStartServesJiraEndpoints(t *testing.T) {
	server := Start()
	defer server.Close()

	client := jira.NewClient(server.URL, "user@example.com", "token")

	tickets, err := client.SearchIssues("DEV")
	if err != nil {
		t.Fatalf("SearchIssues returned error: %v", err)
	}
	if len(tickets) != 3 {
		t.Fatalf("expected 3 tickets from mock search, got %d", len(tickets))
	}
	if tickets[0].Key != "DEV-101" || tickets[0].Summary != "Implement Mock Data Server" {
		t.Fatalf("unexpected first mock ticket: %+v", tickets[0])
	}
	if tickets[2].Status != "Done" || tickets[2].IssueType != "Task" {
		t.Fatalf("unexpected final mock ticket: %+v", tickets[2])
	}

	issue, err := client.GetIssue("DEV-999")
	if err != nil {
		t.Fatalf("GetIssue returned error: %v", err)
	}
	if issue.Key != "DEV-999" || issue.Summary != "Mock Ticket DEV-999" {
		t.Fatalf("unexpected mock issue response: %+v", issue)
	}
	if issue.Status != "In Progress" || issue.IssueType != "Task" {
		t.Fatalf("unexpected mock issue metadata: %+v", issue)
	}

	if err := client.Ping(); err != nil {
		t.Fatalf("Ping returned error: %v", err)
	}

	started := time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC)
	worklogID, err := client.AddWorklog("DEV-101", started, 1800, "Investigated mock issue")
	if err != nil {
		t.Fatalf("AddWorklog returned error: %v", err)
	}
	if !strings.HasPrefix(worklogID, "wl-mock-") {
		t.Fatalf("expected mock worklog id prefix, got %q", worklogID)
	}

	if err := client.UpdateWorklog("DEV-101", worklogID, started, 900, "Updated mock issue"); err != nil {
		t.Fatalf("UpdateWorklog returned error: %v", err)
	}
	if err := client.DeleteWorklog("DEV-101", worklogID); err != nil {
		t.Fatalf("DeleteWorklog returned error: %v", err)
	}
}

func TestStartServesClockifyEndpoints(t *testing.T) {
	server := Start()
	defer server.Close()

	client := clockify.NewClient("api-key", "workspace-1")
	client.SetBaseURL(server.URL)

	if err := client.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if client.GetUserID() != "mock-user-123" {
		t.Fatalf("expected mock user id, got %q", client.GetUserID())
	}

	start := time.Date(2026, 3, 5, 9, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	startedID, err := client.StartTimer("DEV-101 Investigating mock response")
	if err != nil {
		t.Fatalf("StartTimer returned error: %v", err)
	}
	if !strings.HasPrefix(startedID, "clk-mock-") {
		t.Fatalf("expected mock timer id prefix, got %q", startedID)
	}

	manualID, err := client.CreateTimeEntry("DEV-102 Manual work", start, end)
	if err != nil {
		t.Fatalf("CreateTimeEntry returned error: %v", err)
	}
	if !strings.HasPrefix(manualID, "clk-mock-") {
		t.Fatalf("expected mock manual entry id prefix, got %q", manualID)
	}

	stoppedEntry, err := client.StopTimer()
	if err != nil {
		t.Fatalf("StopTimer returned error: %v", err)
	}
	if stoppedEntry.Description != "Stopped Mock Entry" {
		t.Fatalf("unexpected stopped timer response: %+v", stoppedEntry)
	}
	assertApproxOneHour(t, stoppedEntry.Duration)

	entries, err := client.GetTimeEntries(start, end)
	if err != nil {
		t.Fatalf("GetTimeEntries returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 mock time entries, got %d", len(entries))
	}
	if entries[0].ClockifyID != "mock-entry-1" || !strings.Contains(entries[0].Description, "DEV-101") {
		t.Fatalf("unexpected first mock time entry: %+v", entries[0])
	}
	assertApproxOneHour(t, entries[0].Duration)
	assertApproxOneHour(t, entries[1].Duration)

	if err := client.UpdateTimeEntry(entries[0].ClockifyID, "Updated mock entry", start, end); err != nil {
		t.Fatalf("UpdateTimeEntry returned error: %v", err)
	}
	if err := client.DeleteTimeEntry(entries[0].ClockifyID); err != nil {
		t.Fatalf("DeleteTimeEntry returned error: %v", err)
	}
}

func TestStartSearchAndFallbackResponses(t *testing.T) {
	server := Start()
	defer server.Close()

	resp, err := http.Post(server.URL+"/rest/api/3/search/jql", "application/json", strings.NewReader(`{"jql":"order by created DESC","maxResults":20}`))
	if err != nil {
		t.Fatalf("search request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected search endpoint to return 200, got %d", resp.StatusCode)
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		t.Fatalf("expected JSON content type for search endpoint, got %q", contentType)
	}

	var searchPayload struct {
		Total  int `json:"total"`
		Issues []struct {
			Key string `json:"key"`
		} `json:"issues"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&searchPayload); err != nil {
		t.Fatalf("failed to decode search payload: %v", err)
	}
	if searchPayload.Total != 3 || len(searchPayload.Issues) != 3 {
		t.Fatalf("unexpected search payload: %+v", searchPayload)
	}

	fallbackResp, err := http.Get(server.URL + "/unknown/path")
	if err != nil {
		t.Fatalf("fallback request failed: %v", err)
	}
	defer fallbackResp.Body.Close()

	if fallbackResp.StatusCode != http.StatusOK {
		t.Fatalf("expected fallback endpoint to return 200, got %d", fallbackResp.StatusCode)
	}
	if contentType := fallbackResp.Header.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		t.Fatalf("expected JSON content type for fallback endpoint, got %q", contentType)
	}

	body, err := io.ReadAll(fallbackResp.Body)
	if err != nil {
		t.Fatalf("failed to read fallback response body: %v", err)
	}
	if strings.TrimSpace(string(body)) != "{}" {
		t.Fatalf("expected fallback body to be {}, got %q", string(body))
	}
}

func assertApproxOneHour(t *testing.T, seconds int64) {
	t.Helper()
	if seconds < 3599 || seconds > 3601 {
		t.Fatalf("expected duration close to one hour, got %d seconds", seconds)
	}
}
