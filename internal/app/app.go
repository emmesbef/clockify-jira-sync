package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"jirafy-clockwork/internal/clockify"
	"jirafy-clockwork/internal/config"
	"jirafy-clockwork/internal/detector"
	"jirafy-clockwork/internal/jira"
	"jirafy-clockwork/internal/models"
	"jirafy-clockwork/internal/tray"
	"jirafy-clockwork/internal/updater"
)

// App is the main application struct exposed to the Wails frontend
type App struct {
	ctx      context.Context
	cfg      *config.Config
	version  string
	clockify *clockify.Client
	jira     *jira.Client
	detector *detector.Detector
	updater  *updater.Updater

	mu            sync.RWMutex
	timer         models.TimerState
	entries       []models.TimeEntry // local cache of entries
	mockURL       string             // url for local testing (if enabled)
	windowVisible bool               // tracks window visibility for tray menu
	trayCancel    context.CancelFunc
	quitRequested bool
}

var startDetachedProcess = func(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Start()
}
var applyLaunchOnStartup = setLaunchOnStartup

var trayTicketKeyPattern = regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`)

// NewApp creates a new application instance
func NewApp(cfg *config.Config, version string) *App {
	return &App{
		cfg:           cfg,
		version:       version,
		clockify:      clockify.NewClient(cfg.ClockifyAPIKey, cfg.ClockifyWorkspace),
		jira:          jira.NewClient(cfg.JiraBaseURL, cfg.JiraEmail, cfg.JiraAPIToken),
		detector:      detector.NewDetector(15 * time.Second),
		updater:       updater.New(),
		entries:       make([]models.TimeEntry, 0),
		windowVisible: true,
	}
}

// SetMockMode enables the mock HTTP server endpoints for the clients
func (a *App) SetMockMode(mockURL string) {
	a.mockURL = mockURL
	a.clockify.SetBaseURL(mockURL)
	a.jira.SetBaseURL(mockURL)
}

// InitTray sets up the macOS system tray icon and context menu.
// The tray callbacks toggle window visibility and quit the app.
func (a *App) InitTray(version string, icon []byte) {
	tray.Init(version, icon, func() {
		// Toggle window visibility
		a.mu.RLock()
		visible := !a.windowVisible
		a.mu.RUnlock()

		go func(targetVisible bool) {
			if targetVisible {
				a.showFromTray()
			} else {
				a.hideToTray()
			}
		}(visible)
	}, func() {
		// Quit the app
		go a.requestQuit()
	}, func() {
		// Check for updates from tray
		go func() {
			info, err := a.CheckForUpdates()
			if err != nil {
				log.Printf("Update check failed: %v", err)
				return
			}
			if info != nil && a.ctx != nil {
				wailsRuntime.EventsEmit(a.ctx, "update-available", info)
			}
		}()
	}, func(ticketKey, description string) {
		// Start timer directly from a tray-native popover without showing the main app window.
		go func() {
			ticketKey = strings.TrimSpace(ticketKey)
			description = strings.TrimSpace(description)
			if ticketKey == "" {
				ticketKey = extractTicketKey(description)
			}
			if ticketKey == "" {
				if a.ctx != nil {
					wailsRuntime.EventsEmit(a.ctx, "tray-start-timer-error", "Please choose or type a Jira ticket key (for example PROJ-123)")
				}
				return
			}
			if description == "" {
				description = ticketKey
			}

			if _, err := a.StartTimer(ticketKey, "", description); err != nil {
				log.Printf("Tray start timer failed: %v", err)
				if a.ctx != nil {
					wailsRuntime.EventsEmit(a.ctx, "tray-start-timer-error", err.Error())
				}
				return
			}

			if a.ctx != nil {
				wailsRuntime.EventsEmit(a.ctx, "tray-timer-started", ticketKey)
			}
		}()
	}, func(stopComment string) {
		go func() {
			entry, err := a.StopTimerWithComment(stopComment)
			if err != nil {
				log.Printf("Tray stop timer failed: %v", err)
				if strings.Contains(err.Error(), "no timer is running") {
					tray.SetTimerRunning(false)
				}
				if a.ctx != nil {
					wailsRuntime.EventsEmit(a.ctx, "tray-stop-timer-error", err.Error())
				}
				return
			}

			if a.ctx != nil {
				ticketKey := ""
				if entry != nil {
					ticketKey = entry.TicketKey
				}
				wailsRuntime.EventsEmit(a.ctx, "tray-timer-stopped", ticketKey)
			}
		}()
	}, func() {
		go func() {
			ticketKey := ""
			a.mu.RLock()
			if a.timer.Running {
				ticketKey = a.timer.TicketKey
			}
			a.mu.RUnlock()

			if err := a.CancelTimer(); err != nil {
				log.Printf("Tray cancel timer failed: %v", err)
				if strings.Contains(err.Error(), "no timer is running") {
					tray.SetTimerRunning(false)
				}
				if a.ctx != nil {
					wailsRuntime.EventsEmit(a.ctx, "tray-cancel-timer-error", err.Error())
				}
				return
			}

			if a.ctx != nil {
				wailsRuntime.EventsEmit(a.ctx, "tray-timer-canceled", ticketKey)
			}
		}()
	}, func() string {
		tickets, err := a.GetMyTickets()
		if err != nil {
			log.Printf("Tray assigned tickets fetch failed: %v", err)
			return "[]"
		}
		return marshalTrayTickets(tickets, 5)
	}, func(query string) string {
		query = strings.TrimSpace(query)
		if query == "" {
			tickets, err := a.GetMyTickets()
			if err != nil {
				log.Printf("Tray assigned tickets fetch failed: %v", err)
				return "[]"
			}
			return marshalTrayTickets(tickets, 5)
		}

		tickets, err := a.SearchTickets(query)
		if err != nil {
			log.Printf("Tray ticket search failed: %v", err)
			return "[]"
		}
		return marshalTrayTickets(tickets, 10)
	})

	a.mu.RLock()
	running := a.timer.Running
	a.mu.RUnlock()
	tray.SetTimerRunning(running)
	a.refreshTrayStatus()
}

// --- Configuration Methods ---

// GetConfig returns the current configuration
func (a *App) GetConfig() *config.Config {
	return a.cfg
}

// SaveConfig updates and saves the configuration
func (a *App) SaveConfig(newCfg config.Config) error {
	prevLaunchOnStartup := a.cfg.LaunchOnStartup

	a.cfg.ClockifyAPIKey = newCfg.ClockifyAPIKey
	a.cfg.ClockifyWorkspace = newCfg.ClockifyWorkspace
	a.cfg.JiraBaseURL = newCfg.JiraBaseURL
	a.cfg.JiraEmail = newCfg.JiraEmail
	a.cfg.JiraAPIToken = newCfg.JiraAPIToken
	a.cfg.LaunchOnStartup = newCfg.LaunchOnStartup
	a.cfg.SummaryWordLimit = config.NormalizeSummaryWordLimit(newCfg.SummaryWordLimit)
	a.cfg.LogRoundingMin = config.NormalizeLogRoundingMin(newCfg.LogRoundingMin)
	if newCfg.TrayTimerFormat != "" {
		a.cfg.TrayTimerFormat = config.NormalizeTrayTimerFormat(newCfg.TrayTimerFormat)
		a.cfg.TrayShowTimer = newCfg.TrayShowTimer
	}

	if prevLaunchOnStartup != a.cfg.LaunchOnStartup {
		if err := applyLaunchOnStartup(a.cfg.LaunchOnStartup); err != nil {
			return fmt.Errorf("failed to update launch on startup: %w", err)
		}
	}

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

	a.refreshTrayStatus()

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

// GetVersion returns the application version string
func (a *App) GetVersion() string {
	return a.version
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

// GetProjects returns Clockify projects for the configured workspace
func (a *App) GetProjects() ([]clockify.ProjectInfo, error) {
	return a.clockify.GetProjects()
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

func (a *App) setWindowVisible(visible bool) {
	a.mu.Lock()
	a.windowVisible = visible
	a.mu.Unlock()

	tray.SetWindowVisible(visible)
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "window-visibility-changed", visible)
	}
}

func (a *App) hideToTray() {
	a.setWindowVisible(false)

	if a.ctx != nil {
		wailsRuntime.WindowHide(a.ctx)
		wailsRuntime.Hide(a.ctx)
	}
	tray.SetAppBackgroundMode()
	go func() {
		time.Sleep(200 * time.Millisecond)
		tray.SetAppBackgroundMode()
	}()
}

func (a *App) showFromTray() {
	a.setWindowVisible(true)

	tray.SetAppForegroundMode()
	if a.ctx != nil {
		ctx := a.ctx
		go func() {
			time.Sleep(80 * time.Millisecond)
			wailsRuntime.Show(ctx)
			wailsRuntime.WindowShow(ctx)
		}()
	}
}

func (a *App) requestQuit() {
	a.mu.Lock()
	a.quitRequested = true
	a.mu.Unlock()

	if a.ctx != nil {
		wailsRuntime.Quit(a.ctx)
	}
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

// DomReady is called after the frontend DOM is loaded.
func (a *App) DomReady(ctx context.Context) {
	// Ensure the main window is visible and foregrounded on first launch.
	go func() {
		time.Sleep(120 * time.Millisecond)
		a.showFromTray()
	}()

	// Check for updates now that the frontend can receive events
	go a.CheckStartupUpdate()
}

// BeforeClose is called before the window closes. We keep the app running in
// tray mode and switch to background (accessory) app behavior.
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	a.mu.RLock()
	quitting := a.quitRequested
	a.mu.RUnlock()
	if quitting {
		return false
	}

	a.hideToTray()
	return true
}

// Shutdown is called when the app is closing
func (a *App) Shutdown(ctx context.Context) {
	// Stop any running timer gracefully
	a.mu.RLock()
	running := a.timer.Running
	a.mu.RUnlock()

	if running {
		_, _ = a.StopTimer()
		return
	}

	a.mu.Lock()
	a.stopTrayUpdatesLocked()
	a.mu.Unlock()
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
	if len(strings.TrimSpace(query)) < 1 {
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
func (a *App) StartTimer(ticketKey string, projectID string, description string) (*models.TimerState, error) {
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

	// Use provided description or fall back to key + summary
	clockifyDesc := description
	if clockifyDesc == "" {
		clockifyDesc = fmt.Sprintf("%s %s", ticketKey, ticket.Summary)
	}

	// Start in Clockify
	clockifyID, err := a.clockify.StartTimer(clockifyDesc, projectID)
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
	tray.SetTimerRunning(true)
	a.startTrayUpdatesLocked()

	return &a.timer, nil
}

// StopTimer stops the running timer and syncs to Jira.
// Kept for backwards compatibility with existing frontend bindings.
func (a *App) StopTimer() (*models.TimeEntry, error) {
	return a.StopTimerWithComment("")
}

// StopTimerWithComment stops the running timer, syncs to Jira,
// and optionally posts an issue comment when stopComment is provided.
func (a *App) StopTimerWithComment(stopComment string) (*models.TimeEntry, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.stopTimerLocked(strings.TrimSpace(stopComment))
}

func (a *App) stopTimerLocked(stopComment string) (*models.TimeEntry, error) {
	if !a.timer.Running {
		return nil, fmt.Errorf("no timer is running")
	}

	// Stop in Clockify
	clockifyEntry, err := a.clockify.StopTimer()
	if err != nil {
		return nil, fmt.Errorf("failed to stop Clockify timer: %w", err)
	}

	startedAt := a.timer.StartedAt
	endedAt := time.Now()
	duration := int64(endedAt.Sub(startedAt).Seconds())
	if duration < 60 {
		duration = 60 // Jira requires minimum 1 minute
	}
	roundingMin := config.NormalizeLogRoundingMin(a.cfg.LogRoundingMin)
	if roundingMin > 0 {
		duration = roundDurationUp(duration, roundingMin)
		endedAt = startedAt.Add(time.Duration(duration) * time.Second)
	}

	// Create worklog in Jira
	comment := fmt.Sprintf("%s %s", a.timer.TicketKey, a.timer.TicketSummary)
	if roundingMin > 0 && clockifyEntry.ClockifyID != "" {
		if err := a.clockify.UpdateTimeEntry(clockifyEntry.ClockifyID, comment, startedAt, endedAt); err != nil {
			log.Printf("Warning: Failed to apply Clockify rounding: %v", err)
		}
	}
	jiraWorklogID, err := a.jira.AddWorklog(a.timer.TicketKey, startedAt, duration, comment)
	if err != nil {
		log.Printf("Warning: Failed to add Jira worklog: %v", err)
		// Don't fail — Clockify entry was already stopped
	}
	if stopComment != "" {
		if err := a.jira.AddIssueComment(a.timer.TicketKey, stopComment); err != nil {
			log.Printf("Warning: Failed to add Jira issue comment: %v", err)
		}
	}

	entry := models.TimeEntry{
		ID:            fmt.Sprintf("entry_%d", time.Now().UnixNano()),
		TicketKey:     a.timer.TicketKey,
		TicketSummary: a.timer.TicketSummary,
		Description:   comment,
		Start:         startedAt,
		End:           endedAt,
		Duration:      duration,
		ClockifyID:    clockifyEntry.ClockifyID,
		JiraWorklogID: jiraWorklogID,
	}

	a.entries = append([]models.TimeEntry{entry}, a.entries...)

	// Reset timer
	a.stopTrayUpdatesLocked()
	a.timer = models.TimerState{}

	return &entry, nil
}

// CancelTimer discards the currently running timer without creating Jira worklogs.
func (a *App) CancelTimer() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.timer.Running {
		return fmt.Errorf("no timer is running")
	}

	clockifyID := strings.TrimSpace(a.timer.ClockifyID)
	if clockifyID == "" {
		return fmt.Errorf("cannot cancel timer: missing Clockify entry ID")
	}
	if err := a.clockify.DeleteTimeEntry(clockifyID); err != nil {
		return fmt.Errorf("failed to cancel Clockify timer: %w", err)
	}

	a.stopTrayUpdatesLocked()
	a.timer = models.TimerState{}
	return nil
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
	roundingMin := config.NormalizeLogRoundingMin(a.cfg.LogRoundingMin)
	if roundingMin > 0 {
		duration = roundDurationUp(duration, roundingMin)
		end = start.Add(time.Duration(duration) * time.Second)
	}

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
	clockifyID, err := a.clockify.CreateTimeEntry(description, start, end, req.ProjectID)
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
// and stores them in the local cache so subsequent GetHistory calls return them.
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

	// Update the local cache so refreshHistory() shows these entries
	a.mu.Lock()
	a.entries = entries
	a.mu.Unlock()

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

	// Update Jira worklog
	if entry != nil && entry.TicketKey != "" {
		worklogID := entry.JiraWorklogID
		// If we don't have the worklog ID cached, look it up by start time
		if worklogID == "" && !entry.Start.IsZero() {
			if found, err := a.jira.FindWorklogID(entry.TicketKey, entry.Start); err != nil {
				log.Printf("Warning: Failed to find Jira worklog: %v", err)
			} else {
				worklogID = found
			}
		}
		if worklogID != "" {
			duration := int64(end.Sub(start).Seconds())
			if err := a.jira.UpdateWorklog(entry.TicketKey, worklogID, start, duration, description); err != nil {
				log.Printf("Warning: Failed to update Jira worklog: %v", err)
			}
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
	if entry != nil && entry.TicketKey != "" {
		worklogID := entry.JiraWorklogID
		// If we don't have the worklog ID cached, look it up by start time
		if worklogID == "" && !entry.Start.IsZero() {
			if found, err := a.jira.FindWorklogID(entry.TicketKey, entry.Start); err != nil {
				log.Printf("Warning: Failed to find Jira worklog: %v", err)
			} else {
				worklogID = found
			}
		}
		if worklogID != "" {
			if err := a.jira.DeleteWorklog(entry.TicketKey, worklogID); err != nil {
				log.Printf("Warning: Failed to delete Jira worklog: %v", err)
			}
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

// --- Update Methods ---

// CheckForUpdates checks GitHub releases for a newer version.
// Returns nil if already up-to-date.
func (a *App) CheckForUpdates() (*models.UpdateInfo, error) {
	return a.updater.CheckForUpdate(a.version, a.cfg.BetaChannel)
}

// ApplyUpdate downloads and applies the given update.
func (a *App) ApplyUpdate(info models.UpdateInfo) error {
	// Safety net: ensure config is persisted before replacing the binary
	if _, err := config.EnsurePersisted(a.cfg); err != nil {
		log.Printf("Warning: could not ensure config persistence before update: %v", err)
	}
	return a.updater.DownloadAndApply(&info)
}

// RestartApplication relaunches the current app and quits this process.
func (a *App) RestartApplication() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	if err := relaunchExecutable(exePath); err != nil {
		return err
	}

	if a.ctx != nil {
		go func() {
			time.Sleep(200 * time.Millisecond)
			a.requestQuit()
		}()
	}

	return nil
}

func relaunchExecutable(exePath string) error {
	if runtime.GOOS == "darwin" {
		if bundlePath := macAppBundlePath(exePath); bundlePath != "" {
			if err := startDetachedProcess("open", "-n", bundlePath); err != nil {
				return fmt.Errorf("failed to relaunch macOS app bundle: %w", err)
			}
			return nil
		}
	}

	if err := startDetachedProcess(exePath); err != nil {
		return fmt.Errorf("failed to relaunch executable: %w", err)
	}

	return nil
}

func macAppBundlePath(exePath string) string {
	clean := filepath.Clean(exePath)
	const marker = ".app/Contents/MacOS/"
	idx := strings.Index(clean, marker)
	if idx == -1 {
		return ""
	}
	return clean[:idx+len(".app")]
}

// GetUpdatePreferences returns the current update settings.
func (a *App) GetUpdatePreferences() models.UpdatePreferences {
	return models.UpdatePreferences{
		AutoCheck:   a.cfg.AutoUpdate,
		BetaChannel: a.cfg.BetaChannel,
	}
}

// SetUpdatePreferences saves new update settings.
func (a *App) SetUpdatePreferences(prefs models.UpdatePreferences) error {
	a.cfg.AutoUpdate = prefs.AutoCheck
	a.cfg.BetaChannel = prefs.BetaChannel
	return a.cfg.Save()
}

// ConfigPersistenceResult holds the result of EnsureConfigPersisted.
type ConfigPersistenceResult struct {
	Created bool   `json:"created"`
	Path    string `json:"path"`
}

// EnsureConfigPersisted checks that the config dir .env exists. If missing,
// creates it from the current in-memory credentials. Returns whether a new
// file was created and the path. Existing files are never overwritten.
func (a *App) EnsureConfigPersisted() ConfigPersistenceResult {
	p, err := config.FilePath()
	if err != nil {
		log.Printf("Config path resolution failed: %v", err)
		return ConfigPersistenceResult{Created: false, Path: ""}
	}
	created, err := config.EnsurePersisted(a.cfg)
	if err != nil {
		log.Printf("Config persistence failed: %v", err)
	}
	return ConfigPersistenceResult{Created: created, Path: p}
}

// CheckStartupUpdate runs the auto-update check on startup.
// If the current version is a pre-release and beta is disabled, it forces an update.
func (a *App) CheckStartupUpdate() {
	// Beta guard: if running a pre-release with beta disabled, force stable update
	if updater.IsPreReleaseVersion(a.version) && !a.cfg.BetaChannel {
		info, err := a.updater.GetLatestStable(a.version)
		if err != nil {
			log.Printf("Startup update check failed: %v", err)
			return
		}
		if info != nil && a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, "update-forced", info)
		}
		return
	}

	if !a.cfg.AutoUpdate {
		return
	}

	info, err := a.updater.CheckForUpdate(a.version, a.cfg.BetaChannel)
	if err != nil {
		log.Printf("Startup update check failed: %v", err)
		return
	}
	if info != nil && a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "update-available", info)
	}
}

func (a *App) startTrayUpdatesLocked() {
	a.stopTrayUpdatesLocked()
	tray.SetTimerRunning(a.timer.Running)

	if !a.timer.Running {
		tray.SetStatusText("")
		return
	}
	if !a.cfg.TrayShowTimer {
		tray.SetStatusText("")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.trayCancel = cancel

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		a.refreshTrayStatus()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.refreshTrayStatus()
			}
		}
	}()
}

func (a *App) stopTrayUpdatesLocked() {
	if a.trayCancel != nil {
		a.trayCancel()
		a.trayCancel = nil
	}
	tray.SetStatusText("")
	tray.SetTooltip("")
	tray.SetTimerRunning(false)
}

func (a *App) refreshTrayStatus() {
	a.mu.RLock()
	timer := a.timer
	show := a.cfg.TrayShowTimer
	format := config.NormalizeTrayTimerFormat(a.cfg.TrayTimerFormat)
	wordLimit := config.NormalizeSummaryWordLimit(a.cfg.SummaryWordLimit)
	a.mu.RUnlock()

	if !timer.Running || !show {
		tray.SetStatusText("")
		tray.SetTooltip("")
		return
	}

	elapsed := time.Since(timer.StartedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	timerText := formatTrayDuration(elapsed, format)
	text := timerText
	ticketLabel := strings.TrimSpace(timer.TicketKey)
	if timer.TicketSummary != "" {
		summary := truncateSummary(timer.TicketSummary, wordLimit)
		if ticketLabel != "" {
			ticketLabel = fmt.Sprintf("%s %s", ticketLabel, summary)
		} else {
			ticketLabel = summary
		}
	}
	if ticketLabel != "" {
		text = fmt.Sprintf("%s · %s", ticketLabel, timerText)
	}
	tray.SetStatusText(text)

	// Hover tooltip/popup is only needed when summary is truncated by word limit.
	if wordLimit <= 0 {
		tray.SetTooltip("")
		return
	}
	if timer.TicketKey != "" && timer.TicketSummary != "" {
		tray.SetTooltip(fmt.Sprintf("%s — %s", timer.TicketKey, timer.TicketSummary))
	} else if timer.TicketKey != "" {
		tray.SetTooltip(timer.TicketKey)
	} else {
		tray.SetTooltip("")
	}
}

func formatTrayDuration(elapsed time.Duration, format string) string {
	totalSeconds := int64(elapsed.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if config.NormalizeTrayTimerFormat(format) == "hh:mm" {
		return fmt.Sprintf("%02d:%02d", hours, minutes)
	}

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func roundDurationUp(durationSeconds int64, roundingMinutes int) int64 {
	if durationSeconds <= 0 {
		return 0
	}
	roundingMinutes = config.NormalizeLogRoundingMin(roundingMinutes)
	if roundingMinutes == 0 {
		return durationSeconds
	}

	interval := int64(roundingMinutes * 60)
	return ((durationSeconds + interval - 1) / interval) * interval
}

// truncateSummary limits the summary to wordLimit words.
// A wordLimit of 0 means no truncation (full summary).
func truncateSummary(summary string, wordLimit int) string {
	if wordLimit <= 0 {
		return summary
	}
	words := strings.Fields(summary)
	if len(words) <= wordLimit {
		return summary
	}
	return strings.Join(words[:wordLimit], " ") + "…"
}

func extractTicketKey(input string) string {
	normalized := strings.ToUpper(strings.TrimSpace(input))
	if normalized == "" {
		return ""
	}
	return trayTicketKeyPattern.FindString(normalized)
}

func marshalTrayTickets(tickets []models.JiraTicket, limit int) string {
	if limit > 0 && len(tickets) > limit {
		tickets = tickets[:limit]
	}

	payload, err := json.Marshal(tickets)
	if err != nil {
		log.Printf("Tray tickets marshaling failed: %v", err)
		return "[]"
	}

	return string(payload)
}
