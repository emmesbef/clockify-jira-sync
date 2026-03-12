package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"jirafy-clockwork/internal/config"
	"jirafy-clockwork/internal/models"
)

// ADF types for parsing Jira v3 worklog comments in test mocks
type adfDoc struct {
	Type    string       `json:"type"`
	Version int          `json:"version"`
	Content []adfContent `json:"content"`
}

type adfContent struct {
	Type    string        `json:"type"`
	Content []adfTextNode `json:"content,omitempty"`
}

type adfTextNode struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type clockifyCreateCall struct {
	Description string
	Start       time.Time
	End         *time.Time
}

type clockifyUpdateCall struct {
	EntryID     string
	Description string
	Start       time.Time
	End         time.Time
}

type jiraWorklogCall struct {
	IssueKey         string
	WorklogID        string
	Comment          string
	Started          time.Time
	TimeSpentSeconds int64
}

type jiraDeleteCall struct {
	IssueKey  string
	WorklogID string
}

type clockifyEntry struct {
	ID          string
	Description string
	Start       time.Time
	End         *time.Time
}

type appFlowMock struct {
	t      *testing.T
	server *httptest.Server

	mu sync.Mutex

	nextClockifyID int
	nextWorklogID  int

	runningEntry *clockifyEntry
	history      []clockifyEntry

	createdClockify []clockifyCreateCall
	updatedClockify []clockifyUpdateCall
	deletedClockify []string
	historyStart    string
	historyEnd      string
	searchJQLs      []string
	addedWorklogs   []jiraWorklogCall
	updatedWorklogs []jiraWorklogCall
	deletedWorklogs []jiraDeleteCall
}

func newAppFlowMock(t *testing.T) *appFlowMock {
	t.Helper()

	historyEnd1 := time.Date(2024, time.March, 10, 10, 0, 0, 0, time.UTC)
	historyEnd2 := time.Date(2024, time.March, 11, 16, 30, 0, 0, time.UTC)

	mock := &appFlowMock{
		t: t,
		history: []clockifyEntry{
			{
				ID:          "history-1",
				Description: "PROJ-101 Past work",
				Start:       historyEnd1.Add(-1 * time.Hour),
				End:         &historyEnd1,
			},
			{
				ID:          "history-2",
				Description: "PROJ-202 Follow-up",
				Start:       historyEnd2.Add(-90 * time.Minute),
				End:         &historyEnd2,
			},
		},
	}

	mock.server = httptest.NewServer(http.HandlerFunc(mock.handle))
	t.Cleanup(mock.server.Close)

	return mock
}

func newFlowApp(t *testing.T) (*App, *appFlowMock) {
	t.Helper()

	mock := newAppFlowMock(t)
	app := NewApp(&config.Config{
		ClockifyAPIKey:    "key",
		ClockifyWorkspace: "workspace",
		JiraBaseURL:       "https://jira.example.com",
		JiraEmail:         "user@example.com",
		JiraAPIToken:      "token",
	}, "test")
	app.SetMockMode(mock.server.URL)

	if err := app.clockify.Init(); err != nil {
		t.Fatalf("failed to initialize mock Clockify client: %v", err)
	}

	return app, mock
}

func (m *appFlowMock) handle(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/user":
		m.writeJSON(w, http.StatusOK, map[string]string{
			"id":    "mock-user-123",
			"email": "mock@example.com",
		})
	case r.URL.Path == "/rest/api/3/myself":
		m.writeJSON(w, http.StatusOK, map[string]string{
			"accountId": "mock-account-id",
		})
	case r.URL.Path == "/rest/api/3/project":
		m.writeJSON(w, http.StatusOK, []map[string]string{
			{"key": "PROJ", "name": "Project"},
		})
	case r.URL.Path == "/rest/api/3/search/jql":
		m.handleJiraSearch(w, r)
	case strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/"):
		m.handleJiraIssue(w, r)
	case strings.HasPrefix(r.URL.Path, "/workspaces/"):
		m.handleClockify(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (m *appFlowMock) handleJiraSearch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		JQL string `json:"jql"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	m.mu.Lock()
	m.searchJQLs = append(m.searchJQLs, body.JQL)
	m.mu.Unlock()

	m.writeJSON(w, http.StatusOK, map[string]any{
		"total": 2,
		"issues": []map[string]any{
			{
				"key": "PROJ-101",
				"fields": map[string]any{
					"summary":   "Mock Ticket PROJ-101",
					"status":    map[string]string{"name": "In Progress"},
					"assignee":  map[string]string{"displayName": "Mock User"},
					"issuetype": map[string]string{"name": "Task"},
				},
			},
			{
				"key": "PROJ-202",
				"fields": map[string]any{
					"summary":   "Mock Ticket PROJ-202",
					"status":    map[string]string{"name": "To Do"},
					"assignee":  map[string]string{"displayName": "Mock User"},
					"issuetype": map[string]string{"name": "Story"},
				},
			},
		},
	})
}

func (m *appFlowMock) handleJiraIssue(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 5 {
		http.NotFound(w, r)
		return
	}

	issueKey := parts[4]
	switch {
	case len(parts) == 5 && r.Method == http.MethodGet:
		m.writeJSON(w, http.StatusOK, map[string]any{
			"key": issueKey,
			"fields": map[string]any{
				"summary":   fmt.Sprintf("Mock Ticket %s", issueKey),
				"status":    map[string]string{"name": "In Progress"},
				"issuetype": map[string]string{"name": "Task"},
			},
		})
	case len(parts) == 6 && parts[5] == "worklog" && r.Method == http.MethodGet:
		// Return worklogs for this issue
		m.mu.Lock()
		var worklogs []map[string]interface{}
		for _, wl := range m.addedWorklogs {
			if wl.IssueKey == issueKey {
				worklogs = append(worklogs, map[string]interface{}{
					"id":      wl.WorklogID,
					"started": wl.Started.Format("2006-01-02T15:04:05.000+0000"),
				})
			}
		}
		m.mu.Unlock()
		m.writeJSON(w, http.StatusOK, map[string]interface{}{"worklogs": worklogs})
	case len(parts) == 6 && parts[5] == "worklog" && r.Method == http.MethodPost:
		var body struct {
			Comment          *adfDoc `json:"comment"`
			Started          string  `json:"started"`
			TimeSpentSeconds int64   `json:"timeSpentSeconds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			m.writeHandlerError(w, "failed to decode Jira worklog body: %v", err)
			return
		}

		started, err := time.Parse("2006-01-02T15:04:05.000-0700", body.Started)
		if err != nil {
			m.writeHandlerError(w, "failed to parse Jira worklog start %q: %v", body.Started, err)
			return
		}

		comment := ""
		if body.Comment != nil && len(body.Comment.Content) > 0 && len(body.Comment.Content[0].Content) > 0 {
			comment = body.Comment.Content[0].Content[0].Text
		}

		m.mu.Lock()
		m.nextWorklogID++
		worklogID := fmt.Sprintf("wl-%d", m.nextWorklogID)
		m.addedWorklogs = append(m.addedWorklogs, jiraWorklogCall{
			IssueKey:         issueKey,
			WorklogID:        worklogID,
			Comment:          comment,
			Started:          started,
			TimeSpentSeconds: body.TimeSpentSeconds,
		})
		m.mu.Unlock()

		m.writeJSON(w, http.StatusCreated, map[string]string{"id": worklogID})
	case len(parts) == 7 && parts[5] == "worklog" && r.Method == http.MethodPut:
		var body struct {
			Comment          *adfDoc `json:"comment"`
			Started          string  `json:"started"`
			TimeSpentSeconds int64   `json:"timeSpentSeconds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			m.writeHandlerError(w, "failed to decode Jira worklog update body: %v", err)
			return
		}

		started, err := time.Parse("2006-01-02T15:04:05.000-0700", body.Started)
		if err != nil {
			m.writeHandlerError(w, "failed to parse Jira worklog update start %q: %v", body.Started, err)
			return
		}

		comment := ""
		if body.Comment != nil && len(body.Comment.Content) > 0 && len(body.Comment.Content[0].Content) > 0 {
			comment = body.Comment.Content[0].Content[0].Text
		}

		m.mu.Lock()
		m.updatedWorklogs = append(m.updatedWorklogs, jiraWorklogCall{
			IssueKey:         issueKey,
			WorklogID:        parts[6],
			Comment:          comment,
			Started:          started,
			TimeSpentSeconds: body.TimeSpentSeconds,
		})
		m.mu.Unlock()

		w.WriteHeader(http.StatusOK)
	case len(parts) == 7 && parts[5] == "worklog" && r.Method == http.MethodDelete:
		m.mu.Lock()
		m.deletedWorklogs = append(m.deletedWorklogs, jiraDeleteCall{
			IssueKey:  issueKey,
			WorklogID: parts[6],
		})
		m.mu.Unlock()

		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, r)
	}
}

func (m *appFlowMock) handleClockify(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.NotFound(w, r)
		return
	}

	switch {
	case len(parts) == 3 && parts[2] == "time-entries" && r.Method == http.MethodPost:
		var body struct {
			Start       string `json:"start"`
			End         string `json:"end"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			m.writeHandlerError(w, "failed to decode Clockify create body: %v", err)
			return
		}

		started, err := time.Parse(time.RFC3339, body.Start)
		if err != nil {
			m.writeHandlerError(w, "failed to parse Clockify start %q: %v", body.Start, err)
			return
		}

		var ended *time.Time
		if body.End != "" {
			parsedEnd, err := time.Parse(time.RFC3339, body.End)
			if err != nil {
				m.writeHandlerError(w, "failed to parse Clockify end %q: %v", body.End, err)
				return
			}
			ended = &parsedEnd
		}

		m.mu.Lock()
		m.nextClockifyID++
		entryID := fmt.Sprintf("clk-%d", m.nextClockifyID)
		m.createdClockify = append(m.createdClockify, clockifyCreateCall{
			Description: body.Description,
			Start:       started,
			End:         ended,
		})
		if ended == nil {
			m.runningEntry = &clockifyEntry{
				ID:          entryID,
				Description: body.Description,
				Start:       started,
			}
		}
		m.mu.Unlock()

		timeInterval := map[string]string{
			"start": started.UTC().Format("2006-01-02T15:04:05Z"),
		}
		if ended != nil {
			timeInterval["end"] = ended.UTC().Format("2006-01-02T15:04:05Z")
		}

		m.writeJSON(w, http.StatusCreated, map[string]any{
			"id":           entryID,
			"description":  body.Description,
			"timeInterval": timeInterval,
		})
	case len(parts) == 5 && parts[2] == "user" && parts[4] == "time-entries" && r.Method == http.MethodPatch:
		var body struct {
			End string `json:"end"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			m.writeHandlerError(w, "failed to decode Clockify stop body: %v", err)
			return
		}

		ended, err := time.Parse(time.RFC3339, body.End)
		if err != nil {
			m.writeHandlerError(w, "failed to parse Clockify stop end %q: %v", body.End, err)
			return
		}

		m.mu.Lock()
		entry := m.runningEntry
		m.runningEntry = nil
		m.mu.Unlock()

		if entry == nil {
			m.writeHandlerError(w, "received Clockify stop without a running entry")
			return
		}

		m.writeJSON(w, http.StatusOK, map[string]any{
			"id":          entry.ID,
			"description": entry.Description,
			"timeInterval": map[string]string{
				"start": entry.Start.UTC().Format("2006-01-02T15:04:05Z"),
				"end":   ended.UTC().Format("2006-01-02T15:04:05Z"),
			},
		})
	case len(parts) == 5 && parts[2] == "user" && parts[4] == "time-entries" && r.Method == http.MethodGet:
		m.mu.Lock()
		m.historyStart = r.URL.Query().Get("start")
		m.historyEnd = r.URL.Query().Get("end")
		history := make([]clockifyEntry, len(m.history))
		copy(history, m.history)
		m.mu.Unlock()

		resp := make([]map[string]any, 0, len(history))
		for _, entry := range history {
			timeInterval := map[string]string{
				"start": entry.Start.UTC().Format("2006-01-02T15:04:05Z"),
			}
			if entry.End != nil {
				timeInterval["end"] = entry.End.UTC().Format("2006-01-02T15:04:05Z")
			}
			resp = append(resp, map[string]any{
				"id":           entry.ID,
				"description":  entry.Description,
				"timeInterval": timeInterval,
			})
		}

		m.writeJSON(w, http.StatusOK, resp)
	case len(parts) == 4 && parts[2] == "time-entries" && r.Method == http.MethodPut:
		var body struct {
			Start       string `json:"start"`
			End         string `json:"end"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			m.writeHandlerError(w, "failed to decode Clockify update body: %v", err)
			return
		}

		started, err := time.Parse(time.RFC3339, body.Start)
		if err != nil {
			m.writeHandlerError(w, "failed to parse Clockify update start %q: %v", body.Start, err)
			return
		}
		ended, err := time.Parse(time.RFC3339, body.End)
		if err != nil {
			m.writeHandlerError(w, "failed to parse Clockify update end %q: %v", body.End, err)
			return
		}

		m.mu.Lock()
		m.updatedClockify = append(m.updatedClockify, clockifyUpdateCall{
			EntryID:     parts[3],
			Description: body.Description,
			Start:       started,
			End:         ended,
		})
		m.mu.Unlock()

		w.WriteHeader(http.StatusOK)
	case len(parts) == 4 && parts[2] == "time-entries" && r.Method == http.MethodDelete:
		m.mu.Lock()
		m.deletedClockify = append(m.deletedClockify, parts[3])
		m.mu.Unlock()

		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, r)
	}
}

func (m *appFlowMock) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		m.t.Errorf("failed to encode mock response: %v", err)
	}
}

func (m *appFlowMock) writeHandlerError(w http.ResponseWriter, format string, args ...any) {
	m.t.Errorf(format, args...)
	http.Error(w, fmt.Sprintf(format, args...), http.StatusInternalServerError)
}

func TestStartTimerAndStopTimerLifecycle(t *testing.T) {
	app, mock := newFlowApp(t)

	timer, err := app.StartTimer("PROJ-101", "", "")
	if err != nil {
		t.Fatalf("StartTimer returned error: %v", err)
	}
	if !timer.Running {
		t.Fatalf("expected timer to be running")
	}
	if timer.TicketKey != "PROJ-101" {
		t.Fatalf("expected timer ticket key PROJ-101, got %q", timer.TicketKey)
	}
	if timer.TicketSummary != "Mock Ticket PROJ-101" {
		t.Fatalf("expected timer summary to come from Jira, got %q", timer.TicketSummary)
	}
	if timer.ClockifyID == "" {
		t.Fatalf("expected Clockify timer ID to be populated")
	}
	runningClockifyID := timer.ClockifyID

	status := app.GetTimerStatus()
	if status.ClockifyID != runningClockifyID || !status.Running {
		t.Fatalf("expected GetTimerStatus to reflect the running timer, got %+v", status)
	}

	mock.mu.Lock()
	if len(mock.createdClockify) != 1 {
		mock.mu.Unlock()
		t.Fatalf("expected one Clockify create call, got %d", len(mock.createdClockify))
	}
	createCall := mock.createdClockify[0]
	mock.mu.Unlock()
	if createCall.Description != "PROJ-101 Mock Ticket PROJ-101" {
		t.Fatalf("expected Clockify timer description to include ticket summary, got %q", createCall.Description)
	}
	if createCall.End != nil {
		t.Fatalf("expected timer start call to omit an end time")
	}

	app.mu.Lock()
	expectedStartedAt := time.Now().Add(-45 * time.Second).Truncate(time.Millisecond)
	app.timer.StartedAt = expectedStartedAt
	app.mu.Unlock()

	entry, err := app.StopTimer()
	if err != nil {
		t.Fatalf("StopTimer returned error: %v", err)
	}
	if entry.ClockifyID != runningClockifyID {
		t.Fatalf("expected Clockify timer ID %q to carry over, got %q", runningClockifyID, entry.ClockifyID)
	}
	if entry.Duration != 60 {
		t.Fatalf("expected StopTimer to enforce Jira's 60-second minimum, got %d", entry.Duration)
	}
	if entry.JiraWorklogID == "" {
		t.Fatalf("expected StopTimer to create a Jira worklog")
	}
	if app.GetTimerStatus().Running {
		t.Fatalf("expected timer state to be reset after StopTimer")
	}

	history := app.GetHistory()
	if len(history) != 1 || history[0].ID != entry.ID {
		t.Fatalf("expected stopped entry to be cached in history, got %+v", history)
	}

	mock.mu.Lock()
	if len(mock.addedWorklogs) != 1 {
		mock.mu.Unlock()
		t.Fatalf("expected one Jira worklog to be added, got %d", len(mock.addedWorklogs))
	}
	worklog := mock.addedWorklogs[0]
	mock.mu.Unlock()

	if worklog.IssueKey != "PROJ-101" {
		t.Fatalf("expected Jira worklog to be attached to PROJ-101, got %q", worklog.IssueKey)
	}
	if worklog.Comment != "PROJ-101 Mock Ticket PROJ-101" {
		t.Fatalf("expected Jira worklog comment to mirror timer description, got %q", worklog.Comment)
	}
	if worklog.TimeSpentSeconds != 60 {
		t.Fatalf("expected Jira worklog duration 60, got %d", worklog.TimeSpentSeconds)
	}
	if !worklog.Started.Equal(expectedStartedAt) {
		t.Fatalf("expected Jira worklog start %s, got %s", expectedStartedAt, worklog.Started)
	}
}

func TestAddManualEntryValidatesAndCreatesEntries(t *testing.T) {
	t.Run("rejects invalid requests before hitting integrations", func(t *testing.T) {
		testCases := []struct {
			name    string
			req     models.ManualEntryRequest
			wantErr string
		}{
			{
				name: "invalid date",
				req: models.ManualEntryRequest{
					TicketKey: "PROJ-101",
					Date:      "2024/03/15",
					StartTime: "09:30",
					EndTime:   "11:45",
				},
				wantErr: "invalid date format",
			},
			{
				name: "invalid time format",
				req: models.ManualEntryRequest{
					TicketKey: "PROJ-101",
					Date:      "2024-03-15",
					StartTime: "09:30:45",
					EndTime:   "11:45",
				},
				wantErr: "invalid time format, use HH:MM",
			},
			{
				name: "end before start",
				req: models.ManualEntryRequest{
					TicketKey: "PROJ-101",
					Date:      "2024-03-15",
					StartTime: "11:45",
					EndTime:   "09:30",
				},
				wantErr: "end time must be after start time",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				app, mock := newFlowApp(t)

				_, err := app.AddManualEntry(tc.req)
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}

				mock.mu.Lock()
				creates := len(mock.createdClockify)
				worklogs := len(mock.addedWorklogs)
				mock.mu.Unlock()

				if creates != 0 || worklogs != 0 {
					t.Fatalf("expected invalid request to fail before external calls, got %d Clockify creates and %d Jira worklogs", creates, worklogs)
				}
			})
		}
	})

	t.Run("creates a synced entry with derived description", func(t *testing.T) {
		app, mock := newFlowApp(t)

		entry, err := app.AddManualEntry(models.ManualEntryRequest{
			TicketKey: "PROJ-101",
			Date:      "2024-03-15",
			StartTime: "09:30",
			EndTime:   "11:45",
		})
		if err != nil {
			t.Fatalf("AddManualEntry returned error: %v", err)
		}

		if entry.Description != "PROJ-101 Mock Ticket PROJ-101" {
			t.Fatalf("expected default description from Jira summary, got %q", entry.Description)
		}
		if entry.Duration != 8100 {
			t.Fatalf("expected duration 8100 seconds, got %d", entry.Duration)
		}
		if entry.ClockifyID == "" || entry.JiraWorklogID == "" {
			t.Fatalf("expected both Clockify and Jira IDs to be populated, got %+v", entry)
		}

		history := app.GetHistory()
		if len(history) != 1 || history[0].ID != entry.ID {
			t.Fatalf("expected manual entry to be cached locally, got %+v", history)
		}

		mock.mu.Lock()
		if len(mock.createdClockify) != 1 {
			mock.mu.Unlock()
			t.Fatalf("expected one Clockify create call, got %d", len(mock.createdClockify))
		}
		if len(mock.addedWorklogs) != 1 {
			mock.mu.Unlock()
			t.Fatalf("expected one Jira worklog call, got %d", len(mock.addedWorklogs))
		}
		createCall := mock.createdClockify[0]
		worklog := mock.addedWorklogs[0]
		mock.mu.Unlock()

		if createCall.Description != entry.Description {
			t.Fatalf("expected Clockify entry description %q, got %q", entry.Description, createCall.Description)
		}
		if createCall.End == nil {
			t.Fatalf("expected manual Clockify entry to include an end time")
		}
		if !createCall.Start.Equal(entry.Start) || !createCall.End.Equal(entry.End) {
			t.Fatalf("expected Clockify payload times to match entry, got start=%s end=%s", createCall.Start, *createCall.End)
		}
		if worklog.IssueKey != "PROJ-101" || worklog.Comment != entry.Description || worklog.TimeSpentSeconds != 8100 {
			t.Fatalf("unexpected Jira worklog payload: %+v", worklog)
		}
	})
}

func TestGetHistoryFromClockifyParsesEntriesAndDateRange(t *testing.T) {
	app, mock := newFlowApp(t)

	entries, err := app.GetHistoryFromClockify("2024-03-10", "2024-03-12")
	if err != nil {
		t.Fatalf("GetHistoryFromClockify returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected two history entries, got %d", len(entries))
	}

	mock.mu.Lock()
	start := mock.historyStart
	end := mock.historyEnd
	mock.mu.Unlock()

	if start != "2024-03-10T00:00:00Z" {
		t.Fatalf("expected Clockify start query to begin at the start of day, got %q", start)
	}
	if end != "2024-03-12T23:59:59Z" {
		t.Fatalf("expected Clockify end query to include the full end date, got %q", end)
	}

	if entries[0].ID != "history-1" || entries[0].TicketKey != "PROJ-101" || entries[0].TicketSummary != "Past work" {
		t.Fatalf("expected first history entry to extract ticket metadata, got %+v", entries[0])
	}
	if entries[1].ID != "history-2" || entries[1].TicketKey != "PROJ-202" || entries[1].TicketSummary != "Follow-up" {
		t.Fatalf("expected second history entry to extract ticket metadata, got %+v", entries[1])
	}
}

func TestUpdateEntryUsesCachedIDsAndFallbackDescription(t *testing.T) {
	app, mock := newFlowApp(t)

	entry, err := app.AddManualEntry(models.ManualEntryRequest{
		TicketKey:   "PROJ-101",
		Description: "Initial note",
		Date:        "2024-03-15",
		StartTime:   "09:00",
		EndTime:     "10:00",
	})
	if err != nil {
		t.Fatalf("AddManualEntry setup failed: %v", err)
	}

	newStart := time.Date(2024, time.March, 15, 12, 0, 0, 0, time.FixedZone("UTC-2", -2*60*60))
	newEnd := newStart.Add(90 * time.Minute)
	if err := app.UpdateEntry(models.UpdateEntryRequest{
		ID:        entry.ID,
		TicketKey: "PROJ-202",
		Start:     newStart.Format(time.RFC3339),
		End:       newEnd.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("UpdateEntry returned error: %v", err)
	}

	mock.mu.Lock()
	if len(mock.updatedClockify) != 1 {
		mock.mu.Unlock()
		t.Fatalf("expected one Clockify update call, got %d", len(mock.updatedClockify))
	}
	if len(mock.updatedWorklogs) != 1 {
		mock.mu.Unlock()
		t.Fatalf("expected one Jira worklog update, got %d", len(mock.updatedWorklogs))
	}
	clockifyUpdate := mock.updatedClockify[0]
	worklogUpdate := mock.updatedWorklogs[0]
	mock.mu.Unlock()

	if clockifyUpdate.EntryID != entry.ClockifyID {
		t.Fatalf("expected cached Clockify ID %q, got %q", entry.ClockifyID, clockifyUpdate.EntryID)
	}
	if clockifyUpdate.Description != "PROJ-202" {
		t.Fatalf("expected empty description to fall back to the new ticket key, got %q", clockifyUpdate.Description)
	}
	if !clockifyUpdate.Start.Equal(newStart) || !clockifyUpdate.End.Equal(newEnd) {
		t.Fatalf("expected Clockify update times %s-%s, got %s-%s", newStart, newEnd, clockifyUpdate.Start, clockifyUpdate.End)
	}

	if worklogUpdate.IssueKey != "PROJ-101" {
		t.Fatalf("expected Jira worklog to stay attached to original ticket key, got %q", worklogUpdate.IssueKey)
	}
	if worklogUpdate.WorklogID != entry.JiraWorklogID {
		t.Fatalf("expected Jira worklog ID %q, got %q", entry.JiraWorklogID, worklogUpdate.WorklogID)
	}
	if worklogUpdate.Comment != "PROJ-202" || worklogUpdate.TimeSpentSeconds != 5400 {
		t.Fatalf("unexpected Jira worklog update payload: %+v", worklogUpdate)
	}

	history := app.GetHistory()
	if len(history) != 1 {
		t.Fatalf("expected one cached entry after update, got %d", len(history))
	}
	if history[0].TicketKey != "PROJ-202" || history[0].Description != "PROJ-202" {
		t.Fatalf("expected local cache to reflect updated entry, got %+v", history[0])
	}
	if !history[0].Start.Equal(newStart) || !history[0].End.Equal(newEnd) || history[0].Duration != 5400 {
		t.Fatalf("expected local cache timestamps to be updated, got %+v", history[0])
	}
}

func TestDeleteEntryRemovesCachedEntryAndCallsIntegrations(t *testing.T) {
	app, mock := newFlowApp(t)

	entry, err := app.AddManualEntry(models.ManualEntryRequest{
		TicketKey:   "PROJ-101",
		Description: "Delete me",
		Date:        "2024-03-15",
		StartTime:   "13:00",
		EndTime:     "14:00",
	})
	if err != nil {
		t.Fatalf("AddManualEntry setup failed: %v", err)
	}

	if err := app.DeleteEntry(entry.ID); err != nil {
		t.Fatalf("DeleteEntry returned error: %v", err)
	}

	if len(app.GetHistory()) != 0 {
		t.Fatalf("expected entry to be removed from local history")
	}

	mock.mu.Lock()
	if len(mock.deletedClockify) != 1 {
		mock.mu.Unlock()
		t.Fatalf("expected one Clockify delete call, got %d", len(mock.deletedClockify))
	}
	if len(mock.deletedWorklogs) != 1 {
		mock.mu.Unlock()
		t.Fatalf("expected one Jira delete call, got %d", len(mock.deletedWorklogs))
	}
	deletedClockifyID := mock.deletedClockify[0]
	deletedWorklog := mock.deletedWorklogs[0]
	mock.mu.Unlock()

	if deletedClockifyID != entry.ClockifyID {
		t.Fatalf("expected Clockify delete to use cached ID %q, got %q", entry.ClockifyID, deletedClockifyID)
	}
	if deletedWorklog.IssueKey != entry.TicketKey || deletedWorklog.WorklogID != entry.JiraWorklogID {
		t.Fatalf("unexpected Jira delete payload: %+v", deletedWorklog)
	}
}

func TestGetMyTicketsAndSearchTickets(t *testing.T) {
	app, mock := newFlowApp(t)

	if app.GetConfig().ClockifyWorkspace != "workspace" {
		t.Fatalf("expected GetConfig to expose the current app config")
	}
	if len(app.GetDetectedBranches()) != 0 {
		t.Fatalf("expected no detected branches before startup")
	}

	tickets, err := app.GetMyTickets()
	if err != nil {
		t.Fatalf("GetMyTickets returned error: %v", err)
	}
	if len(tickets) != 2 || tickets[0].Key != "PROJ-101" {
		t.Fatalf("unexpected Jira tickets: %+v", tickets)
	}

	mock.mu.Lock()
	initialSearches := append([]string(nil), mock.searchJQLs...)
	mock.mu.Unlock()
	if len(initialSearches) != 1 || !strings.Contains(initialSearches[0], "assignee=currentUser()") {
		t.Fatalf("expected GetMyTickets to issue the assigned-user JQL, got %v", initialSearches)
	}

	results, err := app.SearchTickets("")
	if err != nil {
		t.Fatalf("SearchTickets empty query returned error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected empty search query to return no results, got %+v", results)
	}

	mock.mu.Lock()
	if len(mock.searchJQLs) != 1 {
		mock.mu.Unlock()
		t.Fatalf("expected empty search query to avoid Jira calls, got %d", len(mock.searchJQLs))
	}
	mock.mu.Unlock()

	results, err = app.SearchTickets("PROJ")
	if err != nil {
		t.Fatalf("SearchTickets returned error: %v", err)
	}
	if len(results) != 2 || results[1].Key != "PROJ-202" {
		t.Fatalf("unexpected search results: %+v", results)
	}

	mock.mu.Lock()
	searches := append([]string(nil), mock.searchJQLs...)
	mock.mu.Unlock()

	if len(searches) != 2 {
		t.Fatalf("expected exactly two Jira search calls, got %d", len(searches))
	}
	if !strings.Contains(searches[1], `project in ("PROJ")`) {
		t.Fatalf("expected Jira search JQL to include project filter, got %q", searches[1])
	}
}

func TestShutdownStopsRunningTimer(t *testing.T) {
	app, mock := newFlowApp(t)

	timer, err := app.StartTimer("PROJ-303", "", "")
	if err != nil {
		t.Fatalf("StartTimer returned error: %v", err)
	}
	runningClockifyID := timer.ClockifyID

	app.mu.Lock()
	app.timer.StartedAt = time.Now().Add(-2 * time.Minute).Truncate(time.Millisecond)
	app.mu.Unlock()

	app.Shutdown(context.Background())

	if app.GetTimerStatus().Running {
		t.Fatalf("expected Shutdown to stop the running timer")
	}

	history := app.GetHistory()
	if len(history) != 1 {
		t.Fatalf("expected Shutdown to persist the stopped entry, got %+v", history)
	}
	if history[0].ClockifyID != runningClockifyID {
		t.Fatalf("expected Shutdown to stop the active Clockify entry %q, got %+v", runningClockifyID, history[0])
	}

	mock.mu.Lock()
	worklogs := append([]jiraWorklogCall(nil), mock.addedWorklogs...)
	mock.mu.Unlock()
	if len(worklogs) != 1 || worklogs[0].IssueKey != "PROJ-303" {
		t.Fatalf("expected Shutdown to create one Jira worklog for the running timer, got %+v", worklogs)
	}
}
