# JiraFy Clockwork

[![Pipeline](https://gitlab.com/level-87/clockify-jira-sync/badges/main/pipeline.svg)](https://gitlab.com/level-87/clockify-jira-sync/-/pipelines)
[![Latest release](https://img.shields.io/gitlab/v/release/level-87/clockify-jira-sync?include_prereleases)](https://gitlab.com/level-87/clockify-jira-sync/-/releases)
[![Combined coverage](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Flevel-87.gitlab.io%2Fcoverage%2Fcombined-coverage.json&query=%24.combined.coverage_percent&suffix=%25&label=combined%20coverage)](https://level-87.gitlab.io/coverage/)

Desktop app built with Wails (Go backend + Vite frontend) to track time on Jira issues and keep Clockify/Jira worklogs in sync.

Repository and technical identifiers now use the app-aligned slug `jirafy-clockwork`.

## Installation

Download the latest release for your platform from [GitLab Releases](https://gitlab.com/level-87/clockify-jira-sync/-/releases).

### macOS

Use Homebrew tap for easiest install:

```bash
brew install --cask emmesbef/tap/jirafy-clockwork
```

1. Download the `*-macos-universal.zip` file and unzip it.
2. On first launch, macOS Gatekeeper will ask for confirmation since the app is not notarized:
   - **Right-click** (or Control-click) the app → **Open** → click **Open** in the dialog.
   - If the "Open" button doesn't appear: go to **System Settings → Privacy & Security** → click **"Open Anyway"**.
3. After the first successful launch, macOS remembers your choice and the app opens normally.

### Windows

1. Download the `*-windows-amd64.zip` file and extract it.
2. Run `jirafy-clockwork.exe`. Windows SmartScreen may show a warning for unsigned binaries — click **More info → Run anyway**.

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
- Wails CLI v2 (CI installs `github.com/wailsapp/wails/v2/cmd/wails@v2.11.0`)

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
(cd frontend && npm ci)

# install docs site dependencies
(cd docs-site && npm ci)

# run docs site locally
(cd docs-site && npm start)

# run desktop app in live development mode
wails dev

# backend tests
go test ./...

# frontend tests (watch mode)
(cd frontend && npm test)

# frontend coverage (single-run style, same intent as CI)
(cd frontend && CI=1 npm run test:coverage)

# build the docs site locally
(cd docs-site && npm run build)

# backend coverage profile + combined coverage JSON/summary
mkdir -p coverage
go test ./... -coverprofile=coverage/backend.coverprofile
scripts/ci/generate-combined-coverage.sh

# production builds
(cd frontend && npm run build)
go build ./...
wails build
```

## Release versioning

- Releases remain tag-driven in GitLab CI (`v*` tags).
- Default-branch pipelines now auto-create and push a missing `vX.Y.Z` tag from `wails.json` (`info.productVersion`) after build/test/docs/deploy stages succeed.
- That pushed tag triggers the release pipeline, which packages assets and publishes/updates the matching GitLab Release.
- Manual tagging is still supported:

```bash
git tag v1.11.0
git push origin v1.11.0
```

- Optional `GITLAB_TOKEN` (API scope) can be provided in CI variables for tag push/release API operations when job-token restrictions are enabled.

## CI / release / docs pages overview

- **GitLab CI config**: [`/.gitlab-ci.yml`](./.gitlab-ci.yml)
  - Runs ordered stages: `build -> test -> docs -> deploy -> release`.
  - Includes backend/frontend tests, combined coverage generation, docs freshness/build, docs deployment, and release packaging/publishing.
- **Pipelines**: https://gitlab.com/level-87/clockify-jira-sync/-/pipelines
- **GitLab Releases**: https://gitlab.com/level-87/clockify-jira-sync/-/releases
- **GitLab Pages site**: https://level-87.gitlab.io/
  - Docs home: https://level-87.gitlab.io/
  - Coverage dashboard: https://level-87.gitlab.io/coverage/
  - Combined coverage JSON (badge source): https://level-87.gitlab.io/coverage/combined-coverage.json
  - Frontend LCOV report: https://level-87.gitlab.io/coverage/frontend/lcov-report/index.html
- **Local docs workspace**: `docs-site/`
  - Standalone Docusaurus site for Markdown-based project documentation.
  - GitLab Pages publishes the Docusaurus build at the site root, while CI keeps coverage assets under `/coverage/`.
