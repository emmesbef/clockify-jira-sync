---
sidebar_position: 4
title: Releases, versioning, and CI/CD
---

# Releases, versioning, and CI/CD

## CI workflow overview

The main CI workflow lives at [`ci.yml`](https://github.com/emmesbef/clockify-jira-sync/actions/workflows/ci.yml) and currently runs on pushes, pull requests, and manual dispatches.

| Job | Purpose |
| --- | --- |
| `docs` | Verifies README/docs freshness and builds the Docusaurus site artifact. |
| `test` | Runs Go tests, frontend coverage, and combined coverage generation. |
| `build` | Builds the frontend bundle and Go packages. |
| `pages` | Assembles the GitHub Pages artifact from the Docusaurus build and coverage outputs. |
| `deploy-pages` | Deploys the Pages artifact after the `pages` job succeeds on `main`. |

## Current Pages behavior

GitHub Pages now uses the `docs-site/` Docusaurus production build as the primary documentation site. The custom helper under `scripts/ci/assemble-pages-site.sh` still contributes the coverage dashboard and stable coverage artifact paths under `/coverage/`, so the live README badge source remains unchanged.

## Release workflow overview

Release Please manages release PRs in [`release-please.yml`](https://github.com/emmesbef/clockify-jira-sync/actions/workflows/release-please.yml). When a release PR is merged, it creates the `v*` tag and GitHub Release, then invokes [`release.yml`](https://github.com/emmesbef/clockify-jira-sync/actions/workflows/release.yml) as a reusable workflow to build and upload release assets. The same workflow can also be run manually for an existing tag with `workflow_dispatch`.

`release-please.yml` uses `RELEASE_PLEASE_TOKEN` when that secret is configured and otherwise falls back to the default GitHub Actions token. For release PR creation to work, choose one of these repository setups:

1. Enable **Settings → Actions → General → Workflow permissions → Allow GitHub Actions to create and approve pull requests**.
2. Or add a `RELEASE_PLEASE_TOKEN` repository secret backed by a token that can write **contents**, **issues**, and **pull requests**.

Current release behavior:

- Builds macOS and Windows Wails artifacts.
- Packages the generated binaries into zip archives.
- Produces a `SHA256SUMS` file for release assets.
- Publishes the artifacts to GitHub Releases.

## Versioning status

Versioning is now Release Please-driven. In practice, this means:

- Conventional Commits on `main` determine the next release version.
- Release Please updates `wails.json`, `frontend/package.json`, `frontend/package-lock.json`, `docs-site/package.json`, and `docs-site/package-lock.json` in the release PR.
- Merging the release PR creates the version tag and GitHub Release before macOS/Windows artifacts are attached.
- `release.yml` remains available as a manual backfill/rebuild path for an existing `v*` tag.

## Useful references

- [CI workflow](https://github.com/emmesbef/clockify-jira-sync/actions/workflows/ci.yml)
- [Release workflow](https://github.com/emmesbef/clockify-jira-sync/actions/workflows/release.yml)
- [GitHub Releases](https://github.com/emmesbef/clockify-jira-sync/releases)
- [Setup & configuration](./setup-configuration.md)
- [Development, build, and test](./development-build-test.md)
