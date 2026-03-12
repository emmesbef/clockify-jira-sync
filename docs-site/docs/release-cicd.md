---
sidebar_position: 4
title: Releases, versioning, and CI/CD
---

# Releases, versioning, and CI/CD

## CI workflow overview

The main CI pipeline lives in [`.gitlab-ci.yml`](https://gitlab.com/level-87/clockify-jira-sync/-/blob/main/.gitlab-ci.yml) and runs on branch pushes, merge requests, and tags.

| Job | Purpose |
| --- | --- |
| `test` | Runs Go tests, frontend coverage, and combined coverage generation. |
| `build` | Builds the frontend bundle and Go packages. |
| `docs` | Verifies README/docs freshness and builds the Docusaurus site artifact. |
| `pages` | Assembles and publishes the GitLab Pages artifact from docs and coverage outputs on the default branch. |
| `release_macos` | Builds and packages the macOS universal app zip on tagged commits (requires a `macos` runner tag). |
| `release_windows` | Builds and packages the Windows amd64 app zip on tagged commits (requires a `windows` runner tag). |
| `release_publish` | Generates checksums, uploads release assets to GitLab Package Registry, and creates/updates the GitLab Release. |

## Current Pages behavior

GitLab Pages uses the `docs-site/` Docusaurus production build as the primary documentation site. The helper under `scripts/ci/assemble-pages-site.sh` contributes the coverage dashboard and stable coverage artifact paths under `/coverage/`, so the README badge source stays stable.

## Release workflow overview

Releases are tag-driven. Pushing a `v*` tag triggers `release_macos`, `release_windows`, and `release_publish`:

1. Build macOS and Windows artifacts.
2. Package zip files per platform.
3. Generate `SHA256SUMS`.
4. Upload assets to GitLab Package Registry.
5. Create or update the matching GitLab Release.

Optional secret:

- `GITLAB_TOKEN` (API scope): used by `release_publish` if job-token restrictions prevent release API writes.

Runner requirements:

- A macOS runner tagged `macos` with Go, Node.js/npm, and signing tooling available.
- A Windows runner tagged `windows` with Go and Node.js/npm available.

Current release behavior:

- Builds macOS and Windows Wails artifacts.
- Packages the generated binaries into zip archives.
- Produces a `SHA256SUMS` file for release assets.
- Publishes the artifacts to GitLab Releases.

## Versioning status

Versioning is tag-driven in GitLab CI. In practice:

- Create and push a version tag (for example, `v1.11.0`) to trigger a release.
- CI sets `wails.json` `info.productVersion` at build time from the tag value.
- Repository version files can still be updated conventionally in regular commits when needed.

## Code signing

macOS release binaries are **ad-hoc signed** with entitlements and hardened runtime (`codesign --force --deep --sign - --entitlements ... --options runtime`). This produces a valid local code signature that allows users to open the app via right-click → Open on first launch. The app is **not notarized** by Apple (which would require a paid Developer account), so Gatekeeper still asks for one-time confirmation.

Users need to right-click → Open (or run `xattr -cr`) on first launch. See the [Installation guide](./installation.md) for details.

Windows binaries are currently unsigned. Authenticode signing may be added in the future.

## Useful references

- [GitLab CI config](https://gitlab.com/level-87/clockify-jira-sync/-/blob/main/.gitlab-ci.yml)
- [Pipelines](https://gitlab.com/level-87/clockify-jira-sync/-/pipelines)
- [GitLab Releases](https://gitlab.com/level-87/clockify-jira-sync/-/releases)
- [Setup & configuration](./setup-configuration.md)
- [Development, build, and test](./development-build-test.md)
