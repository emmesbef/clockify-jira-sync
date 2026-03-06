# Clockify ↔ Jira Time Sync

[![CI](https://github.com/emmesbef/clockify-jira-sync/actions/workflows/ci.yml/badge.svg)](https://github.com/emmesbef/clockify-jira-sync/actions/workflows/ci.yml)
[![Combined coverage](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Femmesbef.github.io%2Fclockify-jira-sync%2Fcoverage%2Fcombined-coverage.json&query=%24.combined.coverage_percent&suffix=%25&label=combined%20coverage)](https://emmesbef.github.io/clockify-jira-sync/coverage/)

Desktop app built with Wails (Go backend + Vite frontend) to track time on Jira issues and keep Clockify/Jira worklogs in sync.

## What it does

- Search Jira issues and quickly pick from "assigned to me" tickets.
- Start/stop a running timer (Clockify timer + Jira worklog sync on stop).
- Add manual time entries for a selected date/time range.
- View, edit, and delete synced entries.
- Show live integration status for both Clockify and Jira credentials.
- Detect Jira ticket keys from active VS Code/git branches and suggest tracking.

## Setup and configuration basics

### Prerequisites

- Go `1.23+`
- Node.js `20+` and npm
- Wails CLI v2 (CI uses `github.com/wailsapp/wails/v2/cmd/wails@v2.11.0`)

### Configure integrations

The app reads configuration from `.env` (and can update it from the Settings tab in-app):

```env
CLOCKIFY_API_KEY=...
CLOCKIFY_WORKSPACE_ID=...
JIRA_BASE_URL=https://your-domain.atlassian.net
JIRA_EMAIL=you@example.com
JIRA_API_TOKEN=...
```

Optional for local development/testing:

```env
MOCK_DATA=true
```

When `MOCK_DATA=true`, the app starts with mock defaults and uses the local mock server endpoints.

## Development, build, and test commands

```bash
# install frontend dependencies
cd frontend && npm ci

# run desktop app in live development mode
wails dev

# backend tests
go test ./...

# frontend tests (watch mode)
cd frontend && npm test

# frontend coverage (single-run style, same intent as CI)
cd frontend && CI=1 npm run test:coverage

# backend coverage profile + combined coverage JSON/summary
mkdir -p coverage
go test ./... -coverprofile=coverage/backend.coverprofile
scripts/ci/generate-combined-coverage.sh

# production builds
cd frontend && npm run build
go build ./...
wails build
```

## CI / release / docs pages overview

- **CI workflow**: https://github.com/emmesbef/clockify-jira-sync/actions/workflows/ci.yml
  - Runs docs freshness checks, backend/frontend tests, combined coverage generation, and build checks.
- **Release workflow**: https://github.com/emmesbef/clockify-jira-sync/actions/workflows/release.yml
  - Builds release artifacts for macOS and Windows on version tags (`v*`) and publishes GitHub Releases.
- **GitHub Releases**: https://github.com/emmesbef/clockify-jira-sync/releases
- **GitHub Pages site**: https://emmesbef.github.io/clockify-jira-sync/
  - Docs index: https://emmesbef.github.io/clockify-jira-sync/docs/
  - Coverage dashboard: https://emmesbef.github.io/clockify-jira-sync/coverage/
  - Combined coverage JSON (badge source): https://emmesbef.github.io/clockify-jira-sync/coverage/combined-coverage.json
  - Frontend LCOV report: https://emmesbef.github.io/clockify-jira-sync/coverage/frontend/lcov-report/index.html
