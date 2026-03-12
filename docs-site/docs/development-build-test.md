---
sidebar_position: 3
title: Development, build, and test
---

# Development, build, and test

## Recommended day-to-day workflow

1. Install dependencies in both `frontend/` and `docs-site/`.
2. Start the docs site when you are editing documentation.
3. Run `wails dev` when you are changing the desktop application.
4. Use focused Go or frontend test commands while iterating.
5. Run production-style builds before opening a pull request.

## Common commands

| Command | Purpose |
| --- | --- |
| `cd docs-site && npm start` | Run the Docusaurus docs site locally with live reload. |
| `cd docs-site && npm run build` | Create a production docs build in `docs-site/build/`. |
| `wails dev` | Run the desktop application in live development mode. |
| `go test ./...` | Run all Go tests. |
| `cd frontend && npm test` | Run frontend tests in watch mode. |
| `cd frontend && CI=1 npm run test:coverage` | Run frontend tests once and write coverage output. |
| `cd frontend && npm run build` | Build the Vite frontend bundle used by Wails. |
| `go build ./...` | Build all Go packages. |
| `wails build` | Produce the packaged desktop application. |

## Coverage and docs-related checks

The CI flow still relies on a few helper scripts in `scripts/ci/`:

```bash
scripts/ci/verify-docs-freshness.sh
scripts/ci/generate-docs-bundle.sh
scripts/ci/generate-combined-coverage.sh
scripts/ci/assemble-pages-site.sh
```

`verify-docs-freshness.sh` still uses `generate-docs-bundle.sh` to check the deterministic README/Go docs snapshot, while GitLab Pages now publishes the Docusaurus build from `docs-site/build/` and `assemble-pages-site.sh` keeps the coverage dashboard plus badge JSON under `/coverage/`.

## Production-style validation

A broader validation pass usually looks like this:

```bash
go test ./...
cd frontend && CI=1 npm run test:coverage && npm run build
cd ../docs-site && npm run build
```

Use `wails build` when you need to validate the final desktop packaging flow in addition to the backend/frontend/doc site build steps.
