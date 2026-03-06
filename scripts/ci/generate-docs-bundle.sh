#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

usage() {
  cat <<'EOF'
Usage: scripts/ci/generate-docs-bundle.sh [--output <path>]

Generates a deterministic documentation bundle for CI.
Defaults to: generated-docs/
EOF
}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUTPUT_DIR="$ROOT_DIR/generated-docs"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --output)
      shift
      if [[ $# -eq 0 ]]; then
        echo "Error: --output requires a path argument." >&2
        exit 1
      fi
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

if [[ "$OUTPUT_DIR" != /* ]]; then
  OUTPUT_DIR="$ROOT_DIR/$OUTPUT_DIR"
fi

if [[ -z "$OUTPUT_DIR" || "$OUTPUT_DIR" == "/" || "$OUTPUT_DIR" == "$ROOT_DIR" || "$OUTPUT_DIR" == "$ROOT_DIR/" ]]; then
  echo "Error: refusing to write docs bundle to '$OUTPUT_DIR'." >&2
  exit 1
fi

TMP_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMP_ROOT"' EXIT

TMP_BUNDLE="$TMP_ROOT/generated-docs"
PACKAGE_LIST_FILE="$TMP_ROOT/go-packages.txt"
mkdir -p "$TMP_BUNDLE/go"
cp "$ROOT_DIR/README.md" "$TMP_BUNDLE/README.md"

(cd "$ROOT_DIR" && go list ./... | LC_ALL=C sort) > "$PACKAGE_LIST_FILE"
while IFS= read -r PKG; do
  [[ -n "$PKG" ]] || continue
  SAFE_NAME="${PKG//\//__}"
  SAFE_NAME="${SAFE_NAME//./__}"
  (cd "$ROOT_DIR" && go doc -all "$PKG") > "$TMP_BUNDLE/go/${SAFE_NAME}.txt"
done < "$PACKAGE_LIST_FILE"

rm -rf "$OUTPUT_DIR"
mkdir -p "$(dirname "$OUTPUT_DIR")"
mv "$TMP_BUNDLE" "$OUTPUT_DIR"
