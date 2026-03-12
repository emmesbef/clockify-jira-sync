# JiraFy Clockwork docs site

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

The docs workspace is intentionally isolated from the Wails app frontend in `../frontend/`. GitLab Pages publishes the `docs-site/build/` output at `https://level-87.gitlab.io/jirafy-clockwork/`, while CI keeps coverage assets available under `/coverage/`.
