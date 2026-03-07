package mockserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// Start creates and starts a mock HTTP server returning synthetic Clockify and Jira data
func Start() *httptest.Server {
	mux := http.NewServeMux()

	// ---- Jira Mock Endpoints ---- //
	mux.HandleFunc("/rest/api/3/project", func(w http.ResponseWriter, r *http.Request) {
		resp := []map[string]string{
			{"key": "DEV", "name": "Development"},
			{"key": "DSGN", "name": "Design"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"total": 3,
			"issues": []interface{}{
				map[string]interface{}{
					"key": "DEV-101",
					"fields": map[string]interface{}{
						"summary":   "Implement Mock Data Server",
						"status":    map[string]string{"name": "In Progress"},
						"issuetype": map[string]string{"name": "Story"},
					},
				},
				map[string]interface{}{
					"key": "DEV-102",
					"fields": map[string]interface{}{
						"summary":   "Fix navigation bug",
						"status":    map[string]string{"name": "To Do"},
						"issuetype": map[string]string{"name": "Bug"},
					},
				},
				map[string]interface{}{
					"key": "DSGN-42",
					"fields": map[string]interface{}{
						"summary":   "Update color palette",
						"status":    map[string]string{"name": "Done"},
						"issuetype": map[string]string{"name": "Task"},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Jira get issue
		if strings.Contains(r.URL.Path, "/rest/api/3/issue/") && !strings.Contains(r.URL.Path, "worklog") {
			parts := strings.Split(r.URL.Path, "/")
			key := parts[len(parts)-1]

			resp := map[string]interface{}{
				"key": key,
				"fields": map[string]interface{}{
					"summary":   fmt.Sprintf("Mock Ticket %s", key),
					"status":    map[string]string{"name": "In Progress"},
					"issuetype": map[string]string{"name": "Task"},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Jira connection check
		if r.URL.Path == "/rest/api/3/myself" {
			json.NewEncoder(w).Encode(map[string]string{
				"accountId":    "mock-account-id",
				"emailAddress": "mock@example.com",
			})
			return
		}

		// Jira Add Worklog
		if strings.HasSuffix(r.URL.Path, "worklog") && r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"id": fmt.Sprintf("wl-mock-%d", time.Now().UnixNano())})
			return
		}

		// Jira Update/Delete Worklog
		if strings.Contains(r.URL.Path, "worklog/") {
			if r.Method == "DELETE" {
				w.WriteHeader(http.StatusNoContent)
			} else {
				w.WriteHeader(http.StatusOK)
			}
			return
		}

		// ---- Clockify Mock Endpoints ---- //
		if r.URL.Path == "/user" {
			json.NewEncoder(w).Encode(map[string]string{"id": "mock-user-123", "email": "mock@example.com"})
			return
		}

		if r.URL.Path == "/workspaces" {
			json.NewEncoder(w).Encode([]map[string]string{
				{"id": "ws-mock-001", "name": "My Workspace"},
				{"id": "ws-mock-002", "name": "Second Workspace"},
			})
			return
		}

		// Clockify Projects
		if strings.HasSuffix(r.URL.Path, "/projects") && r.Method == "GET" {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "proj-mock-001", "name": "Project Alpha", "clientName": "Client A", "archived": false},
				{"id": "proj-mock-002", "name": "Project Beta", "clientName": "Client B", "archived": false},
			})
			return
		}

		// Start manual entry / start timer
		if strings.HasSuffix(r.URL.Path, "/time-entries") && r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          fmt.Sprintf("clk-mock-%d", time.Now().UnixNano()),
				"description": "Mock Time Entry",
				"timeInterval": map[string]string{
					"start": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
				},
			})
			return
		}

		// Stop Timer (PATCH)
		if r.Method == "PATCH" && strings.Contains(r.URL.Path, "/time-entries") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          fmt.Sprintf("clk-mock-%d", time.Now().UnixNano()),
				"description": "Stopped Mock Entry",
				"timeInterval": map[string]string{
					"start":    time.Now().Add(-1 * time.Hour).UTC().Format("2006-01-02T15:04:05Z"),
					"end":      time.Now().UTC().Format("2006-01-02T15:04:05Z"),
					"duration": "PT1H",
				},
			})
			return
		}

		// Update Timer (PUT / DELETE)
		if r.Method == "PUT" || r.Method == "DELETE" {
			if r.Method == "DELETE" {
				w.WriteHeader(http.StatusNoContent)
			} else {
				w.WriteHeader(http.StatusOK)
			}
			return
		}

		// Get Time Entries (GET)
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/time-entries") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":          "mock-entry-1",
					"description": "DEV-101 Implement Mock Data Server",
					"timeInterval": map[string]string{
						"start":    time.Now().Add(-2 * time.Hour).UTC().Format("2006-01-02T15:04:05Z"),
						"end":      time.Now().Add(-1 * time.Hour).UTC().Format("2006-01-02T15:04:05Z"),
						"duration": "PT1H",
					},
				},
				{
					"id":          "mock-entry-2",
					"description": "DEV-102 Fix navigation bug",
					"timeInterval": map[string]string{
						"start":    time.Now().Add(-5 * time.Hour).UTC().Format("2006-01-02T15:04:05Z"),
						"end":      time.Now().Add(-4 * time.Hour).UTC().Format("2006-01-02T15:04:05Z"),
						"duration": "PT1H",
					},
				},
			})
			return
		}

		// Default fallback
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	})

	server := httptest.NewServer(mux)
	return server
}
