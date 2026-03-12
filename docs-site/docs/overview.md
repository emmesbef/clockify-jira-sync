---
sidebar_position: 1
slug: /
---

# Overview

JiraFy Clockwork is a desktop application built with Wails to help developers track time against Jira issues and keep Clockify/Jira worklogs in sync.

## Core capabilities

- Search Jira issues, including quick access to tickets assigned to the current user.
- Start and stop a running timer while syncing the resulting worklog between Clockify and Jira.
- Add manual time entries for an explicit date and time range.
- View, edit, and delete synced entries from the desktop app.
- Surface live integration status for Clockify and Jira credentials.
- Detect Jira ticket keys from VS Code workspaces and git branches.

## Repository layout

| Path | Purpose |
| --- | --- |
| `internal/` | Go packages for app orchestration, config, Jira/Clockify clients, branch detection, and mock services. |
| `frontend/` | The isolated Wails/Vite frontend workspace used by the desktop application. |
| `docs-site/` | The standalone Docusaurus documentation workspace added for project docs authoring and local builds. |
| `scripts/ci/` | CI scripts for README/docs freshness, coverage aggregation, and final GitLab Pages assembly around the Docusaurus build. |
| `build/` | Wails packaging assets and generated desktop build outputs. |

## Documentation scope

This Docusaurus site is focused on the living project documentation that developers need most often:

- [Setup & configuration](./setup-configuration.md)
- [Development, build, and test workflow](./development-build-test.md)
- [Releases, versioning, and CI/CD overview](./release-cicd.md)

The live GitLab Pages site now publishes this Docusaurus build as the primary documentation experience, while the CI scripts in `scripts/ci/` keep the coverage dashboard and badge JSON available under `/coverage/`.

## Architecture at a glance

The application uses a Wails v2 architecture:

- Go code in `internal/` exposes methods on the `App` struct.
- The frontend calls those methods through generated Wails bindings.
- Configuration is loaded from `.env` and can also be updated from the in-app Settings screen.
- Optional mock mode routes Jira and Clockify traffic to the local mock server to support development and testing.

## When to use this docs site

Use this site for project-level documentation, operational guidance, and onboarding notes. The desktop app implementation remains isolated in `frontend/`, and the CI/docs assembly scripts continue to handle coverage aggregation plus final GitLab Pages packaging around this Docusaurus build.
