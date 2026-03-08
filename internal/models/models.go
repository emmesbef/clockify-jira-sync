package models

import "time"

// JiraTicket represents a Jira issue
type JiraTicket struct {
	Key       string `json:"key"`
	Summary   string `json:"summary"`
	Status    string `json:"status"`
	Assignee  string `json:"assignee"`
	IssueType string `json:"issueType"`
}

// TimeEntry represents a tracked time entry (synced to both Clockify + Jira)
type TimeEntry struct {
	ID            string    `json:"id"`
	TicketKey     string    `json:"ticketKey"`
	TicketSummary string    `json:"ticketSummary"`
	Description   string    `json:"description"`
	Start         time.Time `json:"start"`
	End           time.Time `json:"end"`
	Duration      int64     `json:"duration"` // seconds
	ClockifyID    string    `json:"clockifyId"`
	JiraWorklogID string    `json:"jiraWorklogId"`
}

// TimerState represents the current running timer
type TimerState struct {
	Running       bool      `json:"running"`
	TicketKey     string    `json:"ticketKey"`
	TicketSummary string    `json:"ticketSummary"`
	StartedAt     time.Time `json:"startedAt"`
	ClockifyID    string    `json:"clockifyId"`
}

// BranchDetection represents a detected Jira ticket from an IDE's git branch
type BranchDetection struct {
	TicketKey  string `json:"ticketKey"`
	BranchName string `json:"branchName"`
	RepoPath   string `json:"repoPath"`
	IDE        string `json:"ide"`
}

// IntegrationStatus reports real-time connectivity state for external services.
type IntegrationStatus struct {
	ClockifyConnected bool   `json:"clockifyConnected"`
	ClockifyError     string `json:"clockifyError,omitempty"`
	JiraConnected     bool   `json:"jiraConnected"`
	JiraError         string `json:"jiraError,omitempty"`
}

// ManualEntryRequest is the request body for creating a manual time entry
type ManualEntryRequest struct {
	TicketKey   string `json:"ticketKey"`
	Description string `json:"description"`
	Date        string `json:"date"`      // YYYY-MM-DD
	StartTime   string `json:"startTime"` // HH:MM
	EndTime     string `json:"endTime"`   // HH:MM
	ProjectID   string `json:"projectId"` // Clockify project ID (optional)
}

// UpdateEntryRequest is the request body for updating a time entry
type UpdateEntryRequest struct {
	ID          string `json:"id"`
	TicketKey   string `json:"ticketKey"`
	Description string `json:"description"`
	Start       string `json:"start"` // ISO 8601
	End         string `json:"end"`   // ISO 8601
}

// UpdateInfo describes an available update from GitHub Releases
type UpdateInfo struct {
	Version      string `json:"version"`
	IsPreRelease bool   `json:"isPreRelease"`
	DownloadURL  string `json:"downloadUrl"`
	ReleaseNotes string `json:"releaseNotes"`
	Size         int64  `json:"size"`
	PublishedAt  string `json:"publishedAt"`
}

// UpdatePreferences holds user preferences for the auto-update system
type UpdatePreferences struct {
	AutoCheck   bool `json:"autoCheck"`
	BetaChannel bool `json:"betaChannel"`
}
