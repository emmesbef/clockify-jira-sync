---
sidebar_position: 2
title: Setup & configuration
---

# Setup & configuration

## Prerequisites

Before working on the project locally, install the following tools:

- Go `1.23+`
- Node.js `20+` and npm
- Wails CLI v2 (CI currently uses `github.com/wailsapp/wails/v2/cmd/wails@v2.11.0`)

## Clone and install dependencies

Install the app and docs dependencies separately so the Wails frontend and docs site stay isolated:

```bash
git clone https://gitlab.com/level-87/jirafy-clockwork.git
cd jirafy-clockwork

cd frontend && npm ci
cd ../docs-site && npm ci
```

If you only need to work on backend code, the `frontend/` install is still required before running a full Wails build.

## Configure integrations

The application reads integration settings from a local `.env` file in the repository root.

```env
CLOCKIFY_API_KEY=...
CLOCKIFY_WORKSPACE_ID=...
JIRA_BASE_URL=https://your-domain.atlassian.net
JIRA_EMAIL=you@example.com
JIRA_API_TOKEN=...
```

| Variable | Required | Purpose |
| --- | --- | --- |
| `CLOCKIFY_API_KEY` | Yes | Authenticates Clockify API requests. |
| `CLOCKIFY_WORKSPACE_ID` | Yes | Selects the Clockify workspace used for timers and entries. |
| `JIRA_BASE_URL` | Yes | Base URL for the Jira Cloud instance. |
| `JIRA_EMAIL` | Yes | Jira account email used for API authentication. |
| `JIRA_API_TOKEN` | Yes | Jira API token used for worklog and issue operations. |
| `MOCK_DATA` | Optional | Enables the local mock server when set to `true`. |

The Settings tab can update the `.env` values from inside the application, but creating the file first is the fastest way to bootstrap a development machine.

## Optional mock mode

For local development and testing without live SaaS credentials:

```env
MOCK_DATA=true
```

When mock mode is enabled, the app starts with mock defaults and routes integration calls through the local mock server endpoints.

## Running the docs site locally

The documentation site is a separate workspace under `docs-site/`.

```bash
cd docs-site
npm start
```

Useful companion commands:

```bash
cd docs-site
npm run build
npm run serve
```

These commands only affect the docs site; they do not build the Wails application or change the existing GitLab Pages publishing pipeline.
