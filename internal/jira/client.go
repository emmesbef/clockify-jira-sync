package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"clockify-jira-sync/internal/models"
)

// escapeJQLText escapes Lucene special characters for the JQL ~ operator.
// Without escaping, characters like - are interpreted as boolean operators.
func escapeJQLText(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\\\`,
		`+`, `\\+`,
		`-`, `\\-`,
		`&`, `\\&`,
		`|`, `\\|`,
		`!`, `\\!`,
		`(`, `\\(`,
		`)`, `\\)`,
		`{`, `\\{`,
		`}`, `\\}`,
		`[`, `\\[`,
		`]`, `\\]`,
		`^`, `\\^`,
		`~`, `\\~`,
		`*`, `\\*`,
		`?`, `\\?`,
		`/`, `\\/`,
	)
	return replacer.Replace(s)
}

// Client wraps the Jira REST API v3
type Client struct {
	baseURL          string
	email            string
	apiToken         string
	httpClient       *http.Client
	projectCache     []jiraProjectInfo
	projectCacheTime time.Time
}

type jiraProjectInfo struct {
	Key  string `json:"key"`
	Name string `json:"name"`
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

// getProjects fetches and caches all Jira projects accessible to the user
func (c *Client) getProjects() ([]jiraProjectInfo, error) {
	if time.Since(c.projectCacheTime) < 5*time.Minute && c.projectCache != nil {
		return c.projectCache, nil
	}

	apiURL := fmt.Sprintf("%s/rest/api/3/project", c.baseURL)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
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
		return nil, fmt.Errorf("projects API returned %d", resp.StatusCode)
	}

	var projects []jiraProjectInfo
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, err
	}

	c.projectCache = projects
	c.projectCacheTime = time.Now()
	return projects, nil
}

// findProjectsByPrefix returns project keys that start with the given prefix
func (c *Client) findProjectsByPrefix(prefix string) []string {
	projects, err := c.getProjects()
	if err != nil {
		return nil
	}
	upper := strings.ToUpper(prefix)
	var keys []string
	for _, p := range projects {
		if strings.HasPrefix(strings.ToUpper(p.Key), upper) {
			keys = append(keys, p.Key)
		}
	}
	return keys
}

// keyWithDashPattern matches "P-", "P-1", "PROJ-123" (case-insensitive)
var keyWithDashPattern = regexp.MustCompile(`(?i)^([A-Z][A-Z0-9]*)-(\d*)$`)

// projectKeyPattern matches "P", "PROJ", "PRO" (uppercase only — likely a project key)
var projectKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]*$`)

// SearchIssues searches for Jira issues matching a query.
// It detects key-like patterns (e.g., SCR, PROJ-1) and finds matching projects
// by prefix, then searches across all of them. For plain text queries it
// searches summary and description.
func (c *Client) SearchIssues(query string) ([]models.JiraTicket, error) {
	trimmed := strings.TrimSpace(query)

	// Key with dash: "SCR-1", "PROJ-123", "P-"
	if matches := keyWithDashPattern.FindStringSubmatch(trimmed); matches != nil {
		projectPrefix := strings.ToUpper(matches[1])
		number := matches[2]

		matchingProjects := c.findProjectsByPrefix(projectPrefix)
		if len(matchingProjects) > 0 {
			projectList := `"` + strings.Join(matchingProjects, `", "`) + `"`
			var jql string
			if number != "" {
				jql = fmt.Sprintf(
					`project in (%s) AND status != Done ORDER BY key ASC`,
					projectList,
				)
			} else {
				jql = fmt.Sprintf(
					`project in (%s) AND status != Done ORDER BY updated DESC`,
					projectList,
				)
			}

			results, err := c.searchWithJQL(jql, 50)
			if err == nil {
				if number != "" {
					// Filter by issue number prefix across all matching projects
					var filtered []models.JiraTicket
					for _, t := range results {
						parts := strings.SplitN(t.Key, "-", 2)
						if len(parts) == 2 && strings.HasPrefix(parts[1], number) {
							filtered = append(filtered, t)
						}
					}
					return filtered, nil
				}
				return results, nil
			}
		}
		// No matching projects — fall through to text search
	}

	// All-uppercase letters: try as a project key prefix (e.g., "SCR", "PROJ")
	if projectKeyPattern.MatchString(trimmed) {
		matchingProjects := c.findProjectsByPrefix(trimmed)
		if len(matchingProjects) > 0 {
			projectList := `"` + strings.Join(matchingProjects, `", "`) + `"`
			jql := fmt.Sprintf(
				`project in (%s) AND status != Done ORDER BY updated DESC`,
				projectList,
			)
			results, err := c.searchWithJQL(jql, 50)
			if err == nil && len(results) > 0 {
				return results, nil
			}
		}
		// No matching projects — fall through to text search
	}

	// Text search on summary and description
	escaped := escapeJQLText(trimmed)
	jql := fmt.Sprintf(
		`(summary ~ "%s" OR description ~ "%s") AND status != Done ORDER BY updated DESC`,
		escaped, escaped,
	)
	return c.searchWithJQL(jql, 20)
}

func (c *Client) searchWithJQL(jql string, maxResults int) ([]models.JiraTicket, error) {
	apiURL := fmt.Sprintf("%s/rest/api/3/search/jql", c.baseURL)

	body, err := json.Marshal(map[string]interface{}{
		"jql":        jql,
		"maxResults": maxResults,
		"fields":     []string{"summary", "status", "assignee", "issuetype"},
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

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
	apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s?fields=summary,status,assignee,issuetype",
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
	apiURL := fmt.Sprintf("%s/rest/api/3/myself", c.baseURL)

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

// adfText wraps a plain text string in Atlassian Document Format (ADF),
// which is required by Jira v3 API for rich-text fields like worklog comments.
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

func newADFComment(text string) *adfDoc {
	if text == "" {
		return nil
	}
	return &adfDoc{
		Type:    "doc",
		Version: 1,
		Content: []adfContent{{
			Type:    "paragraph",
			Content: []adfTextNode{{Type: "text", Text: text}},
		}},
	}
}

// worklogRequest is the request body for creating/updating a worklog
type worklogRequest struct {
	Comment          *adfDoc `json:"comment,omitempty"`
	Started          string  `json:"started"`
	TimeSpentSeconds int64   `json:"timeSpentSeconds"`
}

// worklogResponse is the Jira worklog response
type worklogResponse struct {
	ID string `json:"id"`
}

// AddWorklog adds a worklog entry to a Jira issue
func (c *Client) AddWorklog(issueKey string, started time.Time, timeSpentSeconds int64, comment string) (string, error) {
	apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s/worklog", c.baseURL, issueKey)

	body := worklogRequest{
		Comment:          newADFComment(comment),
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
	apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s/worklog/%s", c.baseURL, issueKey, worklogID)

	body := worklogRequest{
		Comment:          newADFComment(comment),
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
	apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s/worklog/%s", c.baseURL, issueKey, worklogID)

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
