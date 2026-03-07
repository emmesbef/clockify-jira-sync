package clockify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"clockify-jira-sync/internal/models"
)

// Client wraps the Clockify REST API
type Client struct {
	baseURL     string
	apiKey      string
	workspaceID string
	httpClient  *http.Client
	userID      string
}

// NewClient creates a new Clockify API client
func NewClient(apiKey, workspaceID string) *Client {
	return &Client{
		baseURL:     "https://api.clockify.me/api/v1",
		apiKey:      apiKey,
		workspaceID: workspaceID,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// SetBaseURL overrides the default API base URL (used for testing/mocking)
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

// userResponse represents the Clockify user info response
type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// timeEntryResponse represents a Clockify time entry
type timeEntryResponse struct {
	ID           string `json:"id"`
	Description  string `json:"description"`
	TimeInterval struct {
		Start    string `json:"start"`
		End      string `json:"end"`
		Duration string `json:"duration"`
	} `json:"timeInterval"`
}

// Init fetches the current user ID
func (c *Client) Init() error {
	user, err := c.getCurrentUser()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}
	c.userID = user.ID
	return nil
}

// GetUserID returns the authenticated user ID
func (c *Client) GetUserID() string {
	return c.userID
}

func (c *Client) getCurrentUser() (*userResponse, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/user", nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("clockify API error %d: %s", resp.StatusCode, string(body))
	}

	var user userResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetTimeEntries fetches time entries for the current user within a date range
func (c *Client) GetTimeEntries(start, end time.Time) ([]models.TimeEntry, error) {
	url := fmt.Sprintf("%s/workspaces/%s/user/%s/time-entries?start=%s&end=%s&page-size=100",
		c.baseURL, c.workspaceID, c.userID,
		start.UTC().Format("2006-01-02T15:04:05Z"),
		end.UTC().Format("2006-01-02T15:04:05Z"),
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("clockify API error %d: %s", resp.StatusCode, string(body))
	}

	var entries []timeEntryResponse
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, err
	}

	result := make([]models.TimeEntry, 0, len(entries))
	for _, e := range entries {
		te := models.TimeEntry{
			ClockifyID:  e.ID,
			Description: e.Description,
		}
		if t, err := time.Parse("2006-01-02T15:04:05Z", e.TimeInterval.Start); err == nil {
			te.Start = t
		}
		if e.TimeInterval.End != "" {
			if t, err := time.Parse("2006-01-02T15:04:05Z", e.TimeInterval.End); err == nil {
				te.End = t
				te.Duration = int64(te.End.Sub(te.Start).Seconds())
			}
		}
		result = append(result, te)
	}
	return result, nil
}

// StartTimer starts a new running time entry in Clockify
func (c *Client) StartTimer(description string) (string, error) {
	body := map[string]interface{}{
		"start":       time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		"description": description,
	}
	jsonBody, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/workspaces/%s/time-entries", c.baseURL, c.workspaceID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("clockify API error %d: %s", resp.StatusCode, string(respBody))
	}

	var entry timeEntryResponse
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return "", err
	}
	return entry.ID, nil
}

// StopTimer stops the currently running time entry
func (c *Client) StopTimer() (*models.TimeEntry, error) {
	body := map[string]interface{}{
		"end": time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}
	jsonBody, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/workspaces/%s/user/%s/time-entries", c.baseURL, c.workspaceID, c.userID)
	req, err := http.NewRequest("PATCH", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("clockify API error %d: %s", resp.StatusCode, string(respBody))
	}

	var entry timeEntryResponse
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, err
	}

	te := &models.TimeEntry{
		ClockifyID:  entry.ID,
		Description: entry.Description,
	}
	if t, err := time.Parse("2006-01-02T15:04:05Z", entry.TimeInterval.Start); err == nil {
		te.Start = t
	}
	if entry.TimeInterval.End != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", entry.TimeInterval.End); err == nil {
			te.End = t
			te.Duration = int64(te.End.Sub(te.Start).Seconds())
		}
	}
	return te, nil
}

// CreateTimeEntry creates a completed time entry (manual entry)
func (c *Client) CreateTimeEntry(description string, start, end time.Time) (string, error) {
	body := map[string]interface{}{
		"start":       start.UTC().Format("2006-01-02T15:04:05.000Z"),
		"end":         end.UTC().Format("2006-01-02T15:04:05.000Z"),
		"description": description,
	}
	jsonBody, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/workspaces/%s/time-entries", c.baseURL, c.workspaceID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("clockify API error %d: %s", resp.StatusCode, string(respBody))
	}

	var entry timeEntryResponse
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return "", err
	}
	return entry.ID, nil
}

// UpdateTimeEntry updates an existing time entry
func (c *Client) UpdateTimeEntry(entryID, description string, start, end time.Time) error {
	body := map[string]interface{}{
		"start":       start.UTC().Format("2006-01-02T15:04:05.000Z"),
		"end":         end.UTC().Format("2006-01-02T15:04:05.000Z"),
		"description": description,
	}
	jsonBody, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/workspaces/%s/time-entries/%s", c.baseURL, c.workspaceID, entryID)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("clockify API error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// DeleteTimeEntry deletes a time entry
func (c *Client) DeleteTimeEntry(entryID string) error {
	url := fmt.Sprintf("%s/workspaces/%s/time-entries/%s", c.baseURL, c.workspaceID, entryID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("clockify API error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// WorkspaceInfo represents a Clockify workspace
type WorkspaceInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetWorkspaces returns all workspaces accessible to the authenticated user
func (c *Client) GetWorkspaces() ([]WorkspaceInfo, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/workspaces", nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("clockify API error %d: %s", resp.StatusCode, string(body))
	}

	var workspaces []WorkspaceInfo
	if err := json.NewDecoder(resp.Body).Decode(&workspaces); err != nil {
		return nil, err
	}
	return workspaces, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
}
