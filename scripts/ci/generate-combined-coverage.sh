#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

usage() {
  cat <<'EOF'
Usage: scripts/ci/generate-combined-coverage.sh [options]

Compute combined backend/frontend coverage and write machine-readable outputs.

Options:
  --backend-coverprofile <path>  Go coverage profile (default: coverage/backend.coverprofile)
  --frontend-summary <path>      Frontend coverage summary JSON (default: frontend/coverage/coverage-summary.json)
  --output-json <path>           Combined coverage JSON output (default: coverage/combined-coverage.json)
  --output-summary <path>        Optional Markdown summary output (default: coverage/combined-coverage-summary.md)
  --skip-summary                 Do not write a Markdown summary file
  -h, --help                     Show this help message
EOF
}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BACKEND_COVERPROFILE="coverage/backend.coverprofile"
FRONTEND_SUMMARY="frontend/coverage/coverage-summary.json"
OUTPUT_JSON="coverage/combined-coverage.json"
OUTPUT_SUMMARY="coverage/combined-coverage-summary.md"
WRITE_SUMMARY=true

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
    --backend-coverprofile)
      shift
      [[ $# -gt 0 ]] || { echo "Error: --backend-coverprofile requires a path argument." >&2; exit 1; }
      BACKEND_COVERPROFILE="$1"
      ;;
    --frontend-summary)
      shift
      [[ $# -gt 0 ]] || { echo "Error: --frontend-summary requires a path argument." >&2; exit 1; }
      FRONTEND_SUMMARY="$1"
      ;;
    --output-json)
      shift
      [[ $# -gt 0 ]] || { echo "Error: --output-json requires a path argument." >&2; exit 1; }
      OUTPUT_JSON="$1"
      ;;
    --output-summary)
      shift
      [[ $# -gt 0 ]] || { echo "Error: --output-summary requires a path argument." >&2; exit 1; }
      OUTPUT_SUMMARY="$1"
      ;;
    --skip-summary)
      WRITE_SUMMARY=false
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

BACKEND_COVERPROFILE="$(resolve_path "$BACKEND_COVERPROFILE")"
FRONTEND_SUMMARY="$(resolve_path "$FRONTEND_SUMMARY")"
OUTPUT_JSON="$(resolve_path "$OUTPUT_JSON")"
if [[ "$WRITE_SUMMARY" == true ]]; then
  OUTPUT_SUMMARY="$(resolve_path "$OUTPUT_SUMMARY")"
else
  OUTPUT_SUMMARY="__SKIP_SUMMARY__"
fi

if [[ ! -f "$BACKEND_COVERPROFILE" ]]; then
  echo "Error: Go coverprofile not found at '$BACKEND_COVERPROFILE'." >&2
  exit 1
fi

if [[ ! -f "$FRONTEND_SUMMARY" ]]; then
  echo "Error: frontend coverage summary not found at '$FRONTEND_SUMMARY'." >&2
  exit 1
fi

python3 - "$BACKEND_COVERPROFILE" "$FRONTEND_SUMMARY" "$OUTPUT_JSON" "$OUTPUT_SUMMARY" <<'PY'
import json
from decimal import Decimal, ROUND_HALF_UP
from pathlib import Path
import sys

backend_coverprofile = Path(sys.argv[1])
frontend_summary = Path(sys.argv[2])
output_json = Path(sys.argv[3])
output_summary_arg = sys.argv[4]
output_summary = None if output_summary_arg == "__SKIP_SUMMARY__" else Path(output_summary_arg)


def parse_backend_statements(path: Path) -> tuple[int, int]:
    covered = 0
    total = 0
    with path.open("r", encoding="utf-8") as handle:
        for line_number, line in enumerate(handle, start=1):
            stripped = line.strip()
            if not stripped or stripped.startswith("mode:"):
                continue

            parts = stripped.split()
            if len(parts) != 3:
                raise ValueError(f"Unexpected Go coverprofile format at line {line_number}: {stripped}")

            try:
                statement_count = int(parts[1])
                hit_count = int(parts[2])
            except ValueError as exc:
                raise ValueError(
                    f"Invalid numeric values in Go coverprofile at line {line_number}: {stripped}"
                ) from exc

            total += statement_count
            if hit_count > 0:
                covered += statement_count

    return covered, total


def parse_frontend_statements(path: Path) -> tuple[int, int]:
    with path.open("r", encoding="utf-8") as handle:
        summary_data = json.load(handle)

    totals = summary_data.get("total")
    if not isinstance(totals, dict):
        raise ValueError("Frontend coverage summary is missing the 'total' section.")

    statements = totals.get("statements")
    if not isinstance(statements, dict):
        raise ValueError("Frontend coverage summary is missing 'total.statements'.")

    covered = statements.get("covered")
    total = statements.get("total")
    if not isinstance(covered, (int, float)) or not isinstance(total, (int, float)):
        raise ValueError("Frontend coverage summary has invalid numeric values for statements.")

    covered_int = int(covered)
    total_int = int(total)
    if covered_int < 0 or total_int < 0 or covered_int > total_int:
        raise ValueError(
            "Frontend coverage summary has inconsistent statement counts "
            f"(covered={covered_int}, total={total_int})."
        )

    return covered_int, total_int


def to_percent(covered: int, total: int) -> Decimal:
    if total == 0:
        return Decimal("0")
    return (Decimal(covered) * Decimal("100")) / Decimal(total)


def round_percent(value: Decimal) -> float:
    return float(value.quantize(Decimal("0.01"), rounding=ROUND_HALF_UP))


backend_covered, backend_total = parse_backend_statements(backend_coverprofile)
frontend_covered, frontend_total = parse_frontend_statements(frontend_summary)
combined_covered = backend_covered + frontend_covered
combined_total = backend_total + frontend_total

backend_percent = to_percent(backend_covered, backend_total)
frontend_percent = to_percent(frontend_covered, frontend_total)
combined_percent = to_percent(combined_covered, combined_total)

coverage_result = {
    "formula": (
        "combined_percent = "
        "(backend_covered_statements + frontend_covered_statements) / "
        "(backend_total_statements + frontend_total_statements) * 100"
    ),
    "units": "statements",
    "backend": {
        "covered_statements": backend_covered,
        "total_statements": backend_total,
        "coverage_percent": round_percent(backend_percent),
    },
    "frontend": {
        "covered_statements": frontend_covered,
        "total_statements": frontend_total,
        "coverage_percent": round_percent(frontend_percent),
    },
    "combined": {
        "covered_statements": combined_covered,
        "total_statements": combined_total,
        "coverage_percent": round_percent(combined_percent),
    },
}

output_json.parent.mkdir(parents=True, exist_ok=True)
with output_json.open("w", encoding="utf-8") as handle:
    json.dump(coverage_result, handle, indent=2)
    handle.write("\n")

if output_summary is not None:
    output_summary.parent.mkdir(parents=True, exist_ok=True)
    output_summary.write_text(
        "\n".join(
            [
                "# Combined Coverage Summary",
                "",
                f"- Formula: `{coverage_result['formula']}`",
                "- Units: statements",
                "",
                "| Scope | Covered | Total | Coverage |",
                "| --- | ---: | ---: | ---: |",
                f"| Backend (Go) | {backend_covered} | {backend_total} | {coverage_result['backend']['coverage_percent']:.2f}% |",
                f"| Frontend (Vitest) | {frontend_covered} | {frontend_total} | {coverage_result['frontend']['coverage_percent']:.2f}% |",
                f"| **Combined** | **{combined_covered}** | **{combined_total}** | **{coverage_result['combined']['coverage_percent']:.2f}%** |",
                "",
            ]
        ),
        encoding="utf-8",
    )

print(
    f"Combined coverage: {coverage_result['combined']['coverage_percent']:.2f}% "
    f"({combined_covered}/{combined_total} statements)"
)
print(f"JSON output: {output_json}")
if output_summary is not None:
    print(f"Summary output: {output_summary}")
PY
