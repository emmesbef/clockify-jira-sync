#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
README_FILE="$ROOT_DIR/README.md"
GENERATOR="$ROOT_DIR/scripts/ci/generate-docs-bundle.sh"

if [[ ! -x "$GENERATOR" ]]; then
  echo "Error: docs generator is missing or not executable: $GENERATOR" >&2
  exit 1
fi

if [[ ! -f "$README_FILE" ]]; then
  echo "Error: README.md is missing at '$README_FILE'." >&2
  exit 1
fi

if grep -qi "official Wails Vanilla template" "$README_FILE"; then
  echo "Error: README.md still contains template wording and is not project-current." >&2
  exit 1
fi

required_patterns=(
  '^# JiraFy Clockwork'
  'gitlab\.com/level-87/clockify-jira-sync/badges/main/pipeline\.svg'
  'coverage/combined-coverage.json'
  '^## What it does'
  '^## Development, build, and test commands'
  '^## CI / release / docs pages overview'
)

for pattern in "${required_patterns[@]}"; do
  if ! grep -Eq "$pattern" "$README_FILE"; then
    echo "Error: README.md is missing required content matching pattern: $pattern" >&2
    exit 1
  fi
done

TMP_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMP_ROOT"' EXIT

EXPECTED_BUNDLE_DIR="$TMP_ROOT/generated-docs-expected"
"$GENERATOR" --output "$EXPECTED_BUNDLE_DIR"

if [[ ! -f "$EXPECTED_BUNDLE_DIR/README.md" ]]; then
  echo "Error: docs generator did not produce README.md in generated bundle." >&2
  exit 1
fi

if ! find "$EXPECTED_BUNDLE_DIR/go" -type f -name '*.txt' | grep -q .; then
  echo "Error: docs generator did not produce any Go package docs." >&2
  exit 1
fi

CURRENT_BUNDLE_DIR="$ROOT_DIR/generated-docs"
if [[ -d "$CURRENT_BUNDLE_DIR" ]]; then
  DIFF_FILE="$TMP_ROOT/generated-docs.diff"
  if ! diff -ruN "$EXPECTED_BUNDLE_DIR" "$CURRENT_BUNDLE_DIR" > "$DIFF_FILE"; then
    echo "Error: generated-docs/ is stale compared to generator output." >&2
    echo "Regenerate with: scripts/ci/generate-docs-bundle.sh" >&2
    echo >&2
    cat "$DIFF_FILE" >&2
    exit 1
  fi
fi

echo "README/docs freshness checks passed."
