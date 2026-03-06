# Clockify ↔ Jira Time Sync docs site

This workspace contains the standalone Docusaurus documentation site for the project.

## Local development

```bash
cd docs-site
npm ci
npm start
```

## Production build

```bash
cd docs-site
npm run build
npm run serve
```

The docs workspace is intentionally isolated from the Wails app frontend in `../frontend/`. GitHub Pages publishes the `docs-site/build/` output at `https://emmesbef.github.io/clockify-jira-sync/`, while CI keeps coverage assets available under `/coverage/`.
