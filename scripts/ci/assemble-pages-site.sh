#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

usage() {
  cat <<'EOF'
Usage: scripts/ci/assemble-pages-site.sh [options]

Assemble GitLab Pages content from the docs-site Docusaurus build and coverage outputs.

Options:
  --docs-dir <path>               Docusaurus build directory (default: docs-site/build)
  --coverage-dir <path>           Backend/combined coverage directory (default: coverage)
  --frontend-coverage-dir <path>  Frontend coverage directory (default: frontend/coverage)
  --output <path>                 Output site directory (default: site)
  -h, --help                      Show this help message
EOF
}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DOCS_DIR="docs-site/build"
COVERAGE_DIR="coverage"
FRONTEND_COVERAGE_DIR="frontend/coverage"
OUTPUT_DIR="site"

resolve_path() {
  local value="$1"
  if [[ "$value" = /* ]]; then
    printf '%s\n' "$value"
  else
    printf '%s/%s\n' "$ROOT_DIR" "$value"
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --docs-dir)
      shift
      [[ $# -gt 0 ]] || { echo "Error: --docs-dir requires a path argument." >&2; exit 1; }
      DOCS_DIR="$1"
      ;;
    --coverage-dir)
      shift
      [[ $# -gt 0 ]] || { echo "Error: --coverage-dir requires a path argument." >&2; exit 1; }
      COVERAGE_DIR="$1"
      ;;
    --frontend-coverage-dir)
      shift
      [[ $# -gt 0 ]] || { echo "Error: --frontend-coverage-dir requires a path argument." >&2; exit 1; }
      FRONTEND_COVERAGE_DIR="$1"
      ;;
    --output)
      shift
      [[ $# -gt 0 ]] || { echo "Error: --output requires a path argument." >&2; exit 1; }
      OUTPUT_DIR="$1"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Error: unknown argument '$1'." >&2
      usage >&2
      exit 1
      ;;
  esac
  shift
done

DOCS_DIR="$(resolve_path "$DOCS_DIR")"
COVERAGE_DIR="$(resolve_path "$COVERAGE_DIR")"
FRONTEND_COVERAGE_DIR="$(resolve_path "$FRONTEND_COVERAGE_DIR")"
OUTPUT_DIR="$(resolve_path "$OUTPUT_DIR")"

if [[ -z "$OUTPUT_DIR" || "$OUTPUT_DIR" == "/" || "$OUTPUT_DIR" == "$ROOT_DIR" || "$OUTPUT_DIR" == "$ROOT_DIR/" ]]; then
  echo "Error: refusing to write Pages output to '$OUTPUT_DIR'." >&2
  exit 1
fi

if [[ ! -d "$DOCS_DIR" ]]; then
  echo "Error: docs directory not found at '$DOCS_DIR'." >&2
  exit 1
fi

if [[ ! -f "$DOCS_DIR/index.html" ]]; then
  echo "Error: docs directory does not look like a Docusaurus build (missing index.html at '$DOCS_DIR/index.html')." >&2
  exit 1
fi

if [[ ! -d "$COVERAGE_DIR" ]]; then
  echo "Error: coverage directory not found at '$COVERAGE_DIR'." >&2
  exit 1
fi

if [[ ! -d "$FRONTEND_COVERAGE_DIR" ]]; then
  echo "Error: frontend coverage directory not found at '$FRONTEND_COVERAGE_DIR'." >&2
  exit 1
fi

BACKEND_COVERPROFILE="$COVERAGE_DIR/backend.coverprofile"
COMBINED_JSON="$COVERAGE_DIR/combined-coverage.json"
COMBINED_SUMMARY="$COVERAGE_DIR/combined-coverage-summary.md"
FRONTEND_SUMMARY="$FRONTEND_COVERAGE_DIR/coverage-summary.json"
FRONTEND_LCOV="$FRONTEND_COVERAGE_DIR/lcov.info"
FRONTEND_DASHBOARD_DIR="$FRONTEND_COVERAGE_DIR/lcov-report"

for required in \
  "$BACKEND_COVERPROFILE" \
  "$COMBINED_JSON" \
  "$COMBINED_SUMMARY" \
  "$FRONTEND_SUMMARY" \
  "$FRONTEND_LCOV"; do
  if [[ ! -f "$required" ]]; then
    echo "Error: required coverage file missing at '$required'." >&2
    exit 1
  fi
done

if [[ ! -d "$FRONTEND_DASHBOARD_DIR" ]]; then
  echo "Error: frontend coverage dashboard directory missing at '$FRONTEND_DASHBOARD_DIR'." >&2
  exit 1
fi

rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

cp -R "$DOCS_DIR"/. "$OUTPUT_DIR/"
mkdir -p "$OUTPUT_DIR/docs" "$OUTPUT_DIR/coverage/frontend"
cp "$BACKEND_COVERPROFILE" "$OUTPUT_DIR/coverage/backend.coverprofile"
cp "$COMBINED_JSON" "$OUTPUT_DIR/coverage/combined-coverage.json"
cp "$COMBINED_SUMMARY" "$OUTPUT_DIR/coverage/combined-coverage-summary.md"
cp "$FRONTEND_SUMMARY" "$OUTPUT_DIR/coverage/frontend/coverage-summary.json"
cp "$FRONTEND_LCOV" "$OUTPUT_DIR/coverage/frontend/lcov.info"
cp -R "$FRONTEND_DASHBOARD_DIR" "$OUTPUT_DIR/coverage/frontend/lcov-report"

touch "$OUTPUT_DIR/.nojekyll"

cat > "$OUTPUT_DIR/docs/index.html" <<'EOF'
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta http-equiv="refresh" content="0; url=../" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Redirecting to documentation</title>
  </head>
  <body>
    <p>Documentation now lives at the GitLab Pages site root. <a href="../">Continue to the docs home</a>.</p>
    <p><a href="../coverage/">Open the coverage dashboard</a></p>
  </body>
</html>
EOF

python3 - "$OUTPUT_DIR/coverage/combined-coverage.json" "$OUTPUT_DIR/coverage/index.html" <<'PY'
import json
from pathlib import Path
import sys

combined_json = Path(sys.argv[1])
coverage_index = Path(sys.argv[2])
data = json.loads(combined_json.read_text(encoding="utf-8"))

backend = data.get("backend", {})
frontend = data.get("frontend", {})
combined = data.get("combined", {})

def fmt_percent(section: dict) -> str:
    value = section.get("coverage_percent", 0)
    return f"{float(value):.2f}%"

def fmt_int(section: dict, key: str) -> int:
    value = section.get(key, 0)
    return int(value)

html_content = f"""<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Coverage Dashboard</title>
    <style>
      body {{ font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 2rem; line-height: 1.5; }}
      table {{ border-collapse: collapse; width: 100%; max-width: 48rem; }}
      th, td {{ border: 1px solid #d0d7de; padding: 0.5rem 0.75rem; text-align: right; }}
      th:first-child, td:first-child {{ text-align: left; }}
      code {{ background: #f6f8fa; padding: 0.1rem 0.3rem; border-radius: 0.25rem; }}
    </style>
  </head>
  <body>
    <h1>Coverage Dashboard</h1>
    <p>Combined coverage formula: <code>{data.get("formula", "")}</code></p>
    <table>
      <thead>
        <tr>
          <th>Scope</th>
          <th>Covered</th>
          <th>Total</th>
          <th>Coverage</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td>Backend (Go)</td>
          <td>{fmt_int(backend, "covered_statements")}</td>
          <td>{fmt_int(backend, "total_statements")}</td>
          <td>{fmt_percent(backend)}</td>
        </tr>
        <tr>
          <td>Frontend (Vitest)</td>
          <td>{fmt_int(frontend, "covered_statements")}</td>
          <td>{fmt_int(frontend, "total_statements")}</td>
          <td>{fmt_percent(frontend)}</td>
        </tr>
        <tr>
          <td><strong>Combined</strong></td>
          <td><strong>{fmt_int(combined, "covered_statements")}</strong></td>
          <td><strong>{fmt_int(combined, "total_statements")}</strong></td>
          <td><strong>{fmt_percent(combined)}</strong></td>
        </tr>
      </tbody>
    </table>
    <h2>Coverage artifacts</h2>
     <ul>
       <li><a href="combined-coverage-summary.md">combined-coverage-summary.md</a></li>
       <li><a href="combined-coverage.json">combined-coverage.json</a></li>
       <li><a href="backend.coverprofile">backend.coverprofile</a></li>
       <li><a href="frontend/coverage-summary.json">frontend/coverage-summary.json</a></li>
       <li><a href="frontend/lcov.info">frontend/lcov.info</a></li>
       <li><a href="frontend/lcov-report/index.html">Frontend LCOV dashboard</a></li>
     </ul>
     <p><a href="../index.html">Back to documentation home</a></p>
   </body>
 </html>
"""

coverage_index.write_text(html_content, encoding="utf-8")
PY

echo "GitLab Pages site assembled at '$OUTPUT_DIR'."
