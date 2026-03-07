package app

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"clockify-jira-sync/internal/clockify"
	"clockify-jira-sync/internal/config"
	"clockify-jira-sync/internal/detector"
	"clockify-jira-sync/internal/jira"
	"clockify-jira-sync/internal/models"
)

// App is the main application struct exposed to the Wails frontend
type App struct {
	ctx      context.Context
	cfg      *config.Config
	clockify *clockify.Client
	jira     *jira.Client
	detector *detector.Detector

	mu      sync.RWMutex
	timer   models.TimerState
	entries []models.TimeEntry // local cache of entries
	mockURL string             // url for local testing (if enabled)
}

// NewApp creates a new application instance
func NewApp(cfg *config.Config) *App {
	return &App{
		cfg:      cfg,
		clockify: clockify.NewClient(cfg.ClockifyAPIKey, cfg.ClockifyWorkspace),
		jira:     jira.NewClient(cfg.JiraBaseURL, cfg.JiraEmail, cfg.JiraAPIToken),
		detector: detector.NewDetector(15 * time.Second),
		entries:  make([]models.TimeEntry, 0),
	}
}

// SetMockMode enables the mock HTTP server endpoints for the clients
func (a *App) SetMockMode(mockURL string) {
	a.mockURL = mockURL
	a.clockify.SetBaseURL(mockURL)
	a.jira.SetBaseURL(mockURL)
}

// --- Configuration Methods ---

// GetConfig returns the current configuration
func (a *App) GetConfig() *config.Config {
	return a.cfg
}

// SaveConfig updates and saves the configuration
func (a *App) SaveConfig(newCfg config.Config) error {
	a.cfg.ClockifyAPIKey = newCfg.ClockifyAPIKey
	a.cfg.ClockifyWorkspace = newCfg.ClockifyWorkspace
	a.cfg.JiraBaseURL = newCfg.JiraBaseURL
	a.cfg.JiraEmail = newCfg.JiraEmail
	a.cfg.JiraAPIToken = newCfg.JiraAPIToken

	err := config.Save(a.cfg)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Reinitialize clients with new config
	a.clockify = clockify.NewClient(a.cfg.ClockifyAPIKey, a.cfg.ClockifyWorkspace)
	a.jira = jira.NewClient(a.cfg.JiraBaseURL, a.cfg.JiraEmail, a.cfg.JiraAPIToken)

	if a.mockURL != "" {
		a.clockify.SetBaseURL(a.mockURL)
		a.jira.SetBaseURL(a.mockURL)
	} else {
		// Try to fetch clockify user to ensure fresh init
		_ = a.clockify.Init()
	}

	return nil
}

// GetConfigPath returns the path to the .env config file so the UI can display it.
func (a *App) GetConfigPath() string {
	p, err := config.FilePath()
	if err != nil {
		return "(unknown)"
	}
	return p
}

// FetchWorkspaces returns Clockify workspaces for the given API key.
// It uses a temporary client so it works before config is saved.
func (a *App) FetchWorkspaces(apiKey string) ([]clockify.WorkspaceInfo, error) {
	tmp := clockify.NewClient(apiKey, "")
	if a.mockURL != "" {
		tmp.SetBaseURL(a.mockURL)
	}
	return tmp.GetWorkspaces()
}

// GetIntegrationStatus checks whether Clockify and Jira are currently reachable
// with the configured credentials.
func (a *App) GetIntegrationStatus() models.IntegrationStatus {
	status := models.IntegrationStatus{}

	if err := a.clockify.Init(); err != nil {
		status.ClockifyError = err.Error()
	} else {
		status.ClockifyConnected = true
	}

	if err := a.jira.Ping(); err != nil {
		status.JiraError = err.Error()
	} else {
		status.JiraConnected = true
	}

	return status
}

// Startup is called when the Wails app starts
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize Clockify client (fetch user ID)
	if err := a.clockify.Init(); err != nil {
		log.Printf("Warning: Failed to initialize Clockify: %v", err)
	}

	// Set up branch detection callback
	a.detector.OnDetection(func(det models.BranchDetection) {
		wailsRuntime.EventsEmit(a.ctx, "branch-detected", det)
	})

	// Start the IDE detector in background
	go a.detector.Start(ctx)
}

// Shutdown is called when the app is closing
func (a *App) Shutdown(ctx context.Context) {
	// Stop any running timer gracefully
	a.mu.RLock()
	running := a.timer.Running
	a.mu.RUnlock()

	if running {
		_, _ = a.StopTimer()
	}
}

// --- Jira Ticket Methods ---

// GetMyTickets returns Jira tickets assigned to the current user
func (a *App) GetMyTickets() ([]models.JiraTicket, error) {
	tickets, err := a.jira.GetMyIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tickets: %w", err)
	}
	return tickets, nil
}

// SearchTickets searches Jira issues matching the query
func (a *App) SearchTickets(query string) ([]models.JiraTicket, error) {
	if len(strings.TrimSpace(query)) < 2 {
		return []models.JiraTicket{}, nil
	}
	tickets, err := a.jira.SearchIssues(query)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	return tickets, nil
}

// --- Timer Methods ---

// StartTimer begins tracking time for a ticket
func (a *App) StartTimer(ticketKey string) (*models.TimerState, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.timer.Running {
		return nil, fmt.Errorf("timer is already running for %s", a.timer.TicketKey)
	}

	// Get ticket summary
	ticket, err := a.jira.GetIssue(ticketKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	// Start in Clockify
	description := fmt.Sprintf("%s %s", ticketKey, ticket.Summary)
	clockifyID, err := a.clockify.StartTimer(description)
	if err != nil {
		return nil, fmt.Errorf("failed to start Clockify timer: %w", err)
	}

	a.timer = models.TimerState{
		Running:       true,
		TicketKey:     ticketKey,
		TicketSummary: ticket.Summary,
		StartedAt:     time.Now(),
		ClockifyID:    clockifyID,
	}

	return &a.timer, nil
}

// StopTimer stops the running timer and syncs to Jira
func (a *App) StopTimer() (*models.TimeEntry, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.timer.Running {
		return nil, fmt.Errorf("no timer is running")
	}

	// Stop in Clockify
	clockifyEntry, err := a.clockify.StopTimer()
	if err != nil {
		return nil, fmt.Errorf("failed to stop Clockify timer: %w", err)
	}

	// Calculate duration
	duration := int64(time.Since(a.timer.StartedAt).Seconds())
	if duration < 60 {
		duration = 60 // Jira requires minimum 1 minute
	}

	// Create worklog in Jira
	comment := fmt.Sprintf("%s %s", a.timer.TicketKey, a.timer.TicketSummary)
	jiraWorklogID, err := a.jira.AddWorklog(a.timer.TicketKey, a.timer.StartedAt, duration, comment)
	if err != nil {
		log.Printf("Warning: Failed to add Jira worklog: %v", err)
		// Don't fail — Clockify entry was already stopped
	}

	entry := models.TimeEntry{
		ID:            fmt.Sprintf("entry_%d", time.Now().UnixNano()),
		TicketKey:     a.timer.TicketKey,
		TicketSummary: a.timer.TicketSummary,
		Description:   comment,
		Start:         a.timer.StartedAt,
		End:           time.Now(),
		Duration:      duration,
		ClockifyID:    clockifyEntry.ClockifyID,
		JiraWorklogID: jiraWorklogID,
	}

	a.entries = append([]models.TimeEntry{entry}, a.entries...)

	// Reset timer
	a.timer = models.TimerState{}

	return &entry, nil
}

// GetTimerStatus returns the current timer state
func (a *App) GetTimerStatus() models.TimerState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.timer
}

// --- Manual Entry Methods ---

// AddManualEntry creates a time entry for both Clockify and Jira
func (a *App) AddManualEntry(req models.ManualEntryRequest) (*models.TimeEntry, error) {
	// Parse times
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	startParts := strings.Split(req.StartTime, ":")
	endParts := strings.Split(req.EndTime, ":")
	if len(startParts) != 2 || len(endParts) != 2 {
		return nil, fmt.Errorf("invalid time format, use HH:MM")
	}

	var startH, startM, endH, endM int
	fmt.Sscanf(req.StartTime, "%d:%d", &startH, &startM)
	fmt.Sscanf(req.EndTime, "%d:%d", &endH, &endM)

	loc := time.Now().Location()
	start := time.Date(date.Year(), date.Month(), date.Day(), startH, startM, 0, 0, loc)
	end := time.Date(date.Year(), date.Month(), date.Day(), endH, endM, 0, 0, loc)

	if end.Before(start) || end.Equal(start) {
		return nil, fmt.Errorf("end time must be after start time")
	}

	duration := int64(end.Sub(start).Seconds())

	// Get ticket info
	ticket, err := a.jira.GetIssue(req.TicketKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	description := req.Description
	if description == "" {
		description = fmt.Sprintf("%s %s", req.TicketKey, ticket.Summary)
	}

	// Create in Clockify
	clockifyID, err := a.clockify.CreateTimeEntry(description, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to create Clockify entry: %w", err)
	}

	// Create worklog in Jira
	jiraWorklogID, err := a.jira.AddWorklog(req.TicketKey, start, duration, description)
	if err != nil {
		log.Printf("Warning: Failed to add Jira worklog: %v", err)
	}

	entry := models.TimeEntry{
		ID:            fmt.Sprintf("entry_%d", time.Now().UnixNano()),
		TicketKey:     req.TicketKey,
		TicketSummary: ticket.Summary,
		Description:   description,
		Start:         start,
		End:           end,
		Duration:      duration,
		ClockifyID:    clockifyID,
		JiraWorklogID: jiraWorklogID,
	}

	a.mu.Lock()
	a.entries = append([]models.TimeEntry{entry}, a.entries...)
	a.mu.Unlock()

	return &entry, nil
}

// --- History Methods ---

// GetHistory returns time entries for the current session
func (a *App) GetHistory() []models.TimeEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.entries
}

// GetHistoryFromClockify fetches entries from Clockify for a date range
func (a *App) GetHistoryFromClockify(startDate, endDate string) ([]models.TimeEntry, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}
	// Set end to end of day
	end = end.Add(24*time.Hour - time.Second)

	entries, err := a.clockify.GetTimeEntries(start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch entries: %w", err)
	}

	// Try to extract ticket keys from descriptions
	for i := range entries {
		entries[i].ID = entries[i].ClockifyID
		parts := strings.SplitN(entries[i].Description, " ", 2)
		if len(parts) >= 1 {
			entries[i].TicketKey = parts[0]
		}
		if len(parts) >= 2 {
			entries[i].TicketSummary = parts[1]
		}
	}

	return entries, nil
}

// UpdateEntry updates a time entry in both Clockify and Jira
func (a *App) UpdateEntry(req models.UpdateEntryRequest) error {
	start, err := time.Parse(time.RFC3339, req.Start)
	if err != nil {
		return fmt.Errorf("invalid start time: %w", err)
	}
	end, err := time.Parse(time.RFC3339, req.End)
	if err != nil {
		return fmt.Errorf("invalid end time: %w", err)
	}

	description := req.Description
	if description == "" {
		description = req.TicketKey
	}

	// Find the entry in our local cache
	a.mu.Lock()
	var entry *models.TimeEntry
	for i := range a.entries {
		if a.entries[i].ID == req.ID {
			entry = &a.entries[i]
			break
		}
	}
	a.mu.Unlock()

	// Update Clockify
	clockifyID := req.ID
	if entry != nil && entry.ClockifyID != "" {
		clockifyID = entry.ClockifyID
	}
	if err := a.clockify.UpdateTimeEntry(clockifyID, description, start, end); err != nil {
		return fmt.Errorf("failed to update Clockify entry: %w", err)
	}

	// Update Jira worklog if we have one
	if entry != nil && entry.JiraWorklogID != "" {
		duration := int64(end.Sub(start).Seconds())
		if err := a.jira.UpdateWorklog(entry.TicketKey, entry.JiraWorklogID, start, duration, description); err != nil {
			log.Printf("Warning: Failed to update Jira worklog: %v", err)
		}
	}

	// Update local cache
	if entry != nil {
		a.mu.Lock()
		entry.Start = start
		entry.End = end
		entry.Duration = int64(end.Sub(start).Seconds())
		entry.Description = description
		entry.TicketKey = req.TicketKey
		a.mu.Unlock()
	}

	return nil
}

// DeleteEntry deletes a time entry from both Clockify and Jira
func (a *App) DeleteEntry(id string) error {
	a.mu.Lock()
	var entry *models.TimeEntry
	var entryIdx int
	for i := range a.entries {
		if a.entries[i].ID == id {
			entry = &a.entries[i]
			entryIdx = i
			break
		}
	}
	a.mu.Unlock()

	clockifyID := id
	if entry != nil && entry.ClockifyID != "" {
		clockifyID = entry.ClockifyID
	}

	// Delete from Clockify
	if err := a.clockify.DeleteTimeEntry(clockifyID); err != nil {
		return fmt.Errorf("failed to delete Clockify entry: %w", err)
	}

	// Delete from Jira
	if entry != nil && entry.JiraWorklogID != "" {
		if err := a.jira.DeleteWorklog(entry.TicketKey, entry.JiraWorklogID); err != nil {
			log.Printf("Warning: Failed to delete Jira worklog: %v", err)
		}
	}

	// Remove from local cache
	if entry != nil {
		a.mu.Lock()
		a.entries = append(a.entries[:entryIdx], a.entries[entryIdx+1:]...)
		a.mu.Unlock()
	}

	return nil
}

// --- Detection Methods ---

// GetDetectedBranches returns currently detected IDE branches with Jira tickets
func (a *App) GetDetectedBranches() []models.BranchDetection {
	return a.detector.GetDetections()
}
