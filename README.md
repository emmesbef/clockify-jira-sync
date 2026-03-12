# JiraFy Clockwork

[![Pipeline](https://gitlab.com/level-87/jirafy-clockwork/badges/main/pipeline.svg)](https://gitlab.com/level-87/jirafy-clockwork/-/pipelines)
[![Latest release](https://img.shields.io/gitlab/v/release/level-87/jirafy-clockwork?include_prereleases)](https://gitlab.com/level-87/jirafy-clockwork/-/releases)
[![Combined coverage](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Flevel-87.gitlab.io%2Fjirafy-clockwork%2Fcoverage%2Fcombined-coverage.json&query=%24.combined.coverage_percent&suffix=%25&label=combined%20coverage)](https://level-87.gitlab.io/jirafy-clockwork/coverage/)

Desktop app built with Wails (Go backend + Vite frontend) to track time on Jira issues and keep Clockify/Jira worklogs in sync.

Repository and technical identifiers now use the app-aligned slug `jirafy-clockwork`.

## Installation

Download the latest release for your platform from [GitLab Releases](https://gitlab.com/level-87/jirafy-clockwork/-/releases).

### macOS

Use Homebrew tap for easiest install:

```bash
brew install --cask level-87/tap/jirafy-clockwork
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

- Releases are tag-driven in GitLab CI (`v*` tags).
- Create and push a tag to trigger release packaging:

```bash
git tag v1.11.0
git push origin v1.11.0
```

- The release pipeline builds macOS and Windows assets, generates a `SHA256SUMS` file, uploads assets to GitLab Package Registry, and publishes/updates the matching GitLab Release.
- Optional `GITLAB_TOKEN` (API scope) can be provided in CI variables to create/update releases when job-token restrictions are enabled.

## CI / release / docs pages overview

- **GitLab CI config**: [`/.gitlab-ci.yml`](./.gitlab-ci.yml)
  - Runs backend/frontend tests, combined coverage generation, build checks, docs checks/build, release packaging, and Pages deployment.
- **Pipelines**: https://gitlab.com/level-87/jirafy-clockwork/-/pipelines
- **GitLab Releases**: https://gitlab.com/level-87/jirafy-clockwork/-/releases
- **GitLab Pages site**: https://level-87.gitlab.io/jirafy-clockwork/
  - Docs home: https://level-87.gitlab.io/jirafy-clockwork/
  - Legacy docs URL redirect: https://level-87.gitlab.io/jirafy-clockwork/docs/
  - Coverage dashboard: https://level-87.gitlab.io/jirafy-clockwork/coverage/
  - Combined coverage JSON (badge source): https://level-87.gitlab.io/jirafy-clockwork/coverage/combined-coverage.json
  - Frontend LCOV report: https://level-87.gitlab.io/jirafy-clockwork/coverage/frontend/lcov-report/index.html
- **Local docs workspace**: `docs-site/`
  - Standalone Docusaurus site for Markdown-based project documentation.
  - GitLab Pages publishes the Docusaurus build at the site root, while CI keeps coverage assets under `/coverage/`.
