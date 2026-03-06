package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"clockify-jira-sync/internal/models"
)

// Client wraps the Jira REST API v2
type Client struct {
	baseURL    string
	email      string
	apiToken   string
	httpClient *http.Client
}

// NewClient creates a new Jira API client
func NewClient(baseURL, email, apiToken string) *Client {
	return &Client{
		baseURL:    baseURL,
		email:      email,
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// SetBaseURL overrides the default API base URL (used for testing/mocking)
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

// searchResponse is the Jira search API response
type searchResponse struct {
	Issues []issueResponse `json:"issues"`
	Total  int             `json:"total"`
}

type issueResponse struct {
	Key    string      `json:"key"`
	Fields issueFields `json:"fields"`
}

type issueFields struct {
	Summary   string      `json:"summary"`
	Status    statusField `json:"status"`
	Assignee  *userField  `json:"assignee"`
	IssueType typeField   `json:"issuetype"`
}

type statusField struct {
	Name string `json:"name"`
}

type userField struct {
	DisplayName string `json:"displayName"`
}

type typeField struct {
	Name string `json:"name"`
}

// GetMyIssues fetches issues assigned to the authenticated user
func (c *Client) GetMyIssues() ([]models.JiraTicket, error) {
	jql := "assignee=currentUser() AND status != Done ORDER BY updated DESC"
	return c.searchWithJQL(jql, 50)
}

// SearchIssues searches for Jira issues matching a query
func (c *Client) SearchIssues(query string) ([]models.JiraTicket, error) {
	jql := fmt.Sprintf(
		`(summary ~ "%s" OR key = "%s") AND status != Done ORDER BY updated DESC`,
		query, query,
	)
	return c.searchWithJQL(jql, 20)
}

func (c *Client) searchWithJQL(jql string, maxResults int) ([]models.JiraTicket, error) {
	apiURL := fmt.Sprintf("%s/rest/api/2/search?jql=%s&maxResults=%d&fields=summary,status,assignee,issuetype",
		c.baseURL, url.QueryEscape(jql), maxResults)

	req, err := http.NewRequest("GET", apiURL, nil)
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
		return nil, fmt.Errorf("jira API error %d: %s", resp.StatusCode, string(body))
	}

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	tickets := make([]models.JiraTicket, 0, len(result.Issues))
	for _, issue := range result.Issues {
		t := models.JiraTicket{
			Key:       issue.Key,
			Summary:   issue.Fields.Summary,
			Status:    issue.Fields.Status.Name,
			IssueType: issue.Fields.IssueType.Name,
		}
		if issue.Fields.Assignee != nil {
			t.Assignee = issue.Fields.Assignee.DisplayName
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

// GetIssue fetches a single Jira issue by key
func (c *Client) GetIssue(key string) (*models.JiraTicket, error) {
	apiURL := fmt.Sprintf("%s/rest/api/2/issue/%s?fields=summary,status,assignee,issuetype",
		c.baseURL, key)

	req, err := http.NewRequest("GET", apiURL, nil)
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
		return nil, fmt.Errorf("jira API error %d: %s", resp.StatusCode, string(body))
	}

	var issue issueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, err
	}

	t := &models.JiraTicket{
		Key:       issue.Key,
		Summary:   issue.Fields.Summary,
		Status:    issue.Fields.Status.Name,
		IssueType: issue.Fields.IssueType.Name,
	}
	if issue.Fields.Assignee != nil {
		t.Assignee = issue.Fields.Assignee.DisplayName
	}
	return t, nil
}

// Ping checks whether Jira credentials/base URL are valid.
func (c *Client) Ping() error {
	apiURL := fmt.Sprintf("%s/rest/api/2/myself", c.baseURL)

	req, err := http.NewRequest("GET", apiURL, nil)
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("jira API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// worklogRequest is the request body for creating/updating a worklog
type worklogRequest struct {
	Comment          string `json:"comment,omitempty"`
	Started          string `json:"started"`
	TimeSpentSeconds int64  `json:"timeSpentSeconds"`
}

// worklogResponse is the Jira worklog response
type worklogResponse struct {
	ID string `json:"id"`
}

// AddWorklog adds a worklog entry to a Jira issue
func (c *Client) AddWorklog(issueKey string, started time.Time, timeSpentSeconds int64, comment string) (string, error) {
	apiURL := fmt.Sprintf("%s/rest/api/2/issue/%s/worklog", c.baseURL, issueKey)

	body := worklogRequest{
		Comment:          comment,
		Started:          started.Format("2006-01-02T15:04:05.000-0700"),
		TimeSpentSeconds: timeSpentSeconds,
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
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
		return "", fmt.Errorf("jira API error %d: %s", resp.StatusCode, string(respBody))
	}

	var worklog worklogResponse
	if err := json.NewDecoder(resp.Body).Decode(&worklog); err != nil {
		return "", err
	}
	return worklog.ID, nil
}

// UpdateWorklog updates an existing worklog
func (c *Client) UpdateWorklog(issueKey, worklogID string, started time.Time, timeSpentSeconds int64, comment string) error {
	apiURL := fmt.Sprintf("%s/rest/api/2/issue/%s/worklog/%s", c.baseURL, issueKey, worklogID)

	body := worklogRequest{
		Comment:          comment,
		Started:          started.Format("2006-01-02T15:04:05.000-0700"),
		TimeSpentSeconds: timeSpentSeconds,
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("PUT", apiURL, bytes.NewReader(jsonBody))
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
		return fmt.Errorf("jira API error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// DeleteWorklog deletes a worklog from a Jira issue
func (c *Client) DeleteWorklog(issueKey, worklogID string) error {
	apiURL := fmt.Sprintf("%s/rest/api/2/issue/%s/worklog/%s", c.baseURL, issueKey, worklogID)

	req, err := http.NewRequest("DELETE", apiURL, nil)
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
		return fmt.Errorf("jira API error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}
