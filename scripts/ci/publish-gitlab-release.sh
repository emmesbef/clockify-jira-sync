#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

APP_NAME="${APP_NAME:-jirafy-clockwork}"
ASSETS_DIR="${ASSETS_DIR:-release-assets}"
TAG="${CI_COMMIT_TAG:-}"
API_BASE="${CI_API_V4_URL:-}"
PROJECT_ID="${CI_PROJECT_ID:-}"
PROJECT_URL="${CI_PROJECT_URL:-}"
JOB_TOKEN="${CI_JOB_TOKEN:-}"

if [[ -z "$TAG" ]]; then
  echo "Error: CI_COMMIT_TAG is required." >&2
  exit 1
fi

if [[ -z "$API_BASE" || -z "$PROJECT_ID" || -z "$PROJECT_URL" || -z "$JOB_TOKEN" ]]; then
  echo "Error: CI_API_V4_URL, CI_PROJECT_ID, CI_PROJECT_URL, and CI_JOB_TOKEN are required." >&2
  exit 1
fi

if [[ ! -d "$ASSETS_DIR" ]]; then
  echo "Error: release assets directory not found: $ASSETS_DIR" >&2
  exit 1
fi

shopt -s nullglob
zip_assets=("$ASSETS_DIR"/*.zip)
shopt -u nullglob

if [[ ${#zip_assets[@]} -eq 0 ]]; then
  echo "Error: no zip artifacts found in $ASSETS_DIR" >&2
  exit 1
fi

checksums_file="$ASSETS_DIR/${APP_NAME}-${TAG}-SHA256SUMS.txt"
(
  cd "$ASSETS_DIR"
  sha256sum ./*.zip > "$(basename "$checksums_file")"
)

version="${TAG#v}"
notes_file="$(mktemp)"
if [[ -f CHANGELOG.md ]]; then
  release_notes="$(awk "/^## \\[${version}\\]/{found=1; next} /^## \\[/{if(found) exit} found{print}" CHANGELOG.md)"
fi
if [[ -n "${release_notes:-}" ]]; then
  printf '%s\n' "$release_notes" > "$notes_file"
else
  printf 'Release %s\n\nAutomated release from GitLab CI.' "$TAG" > "$notes_file"
fi

package_base="${API_BASE}/projects/${PROJECT_ID}/packages/generic/${APP_NAME}/${TAG}"
download_base="${PROJECT_URL}/-/packages/generic/${APP_NAME}/${TAG}"

uploaded_names=()
for file_path in "${zip_assets[@]}" "$checksums_file"; do
  file_name="$(basename "$file_path")"
  echo "Uploading ${file_name} to GitLab Package Registry"
  curl --fail --silent --show-error \
    --header "JOB-TOKEN: ${JOB_TOKEN}" \
    --upload-file "$file_path" \
    "${package_base}/${file_name}"
  uploaded_names+=("$file_name")
done

assets_links_json='[]'
for file_name in "${uploaded_names[@]}"; do
  assets_links_json="$(
    jq -c \
      --arg name "$file_name" \
      --arg url "${download_base}/${file_name}" \
      '. + [{name: $name, url: $url, link_type: "package"}]' \
      <<<"$assets_links_json"
  )"
done

add_alias_link() {
  local alias_name="$1"
  local target_name="$2"
  assets_links_json="$(
    jq -c \
      --arg name "$alias_name" \
      --arg url "${download_base}/${target_name}" \
      '. + [{name: $name, url: $url, link_type: "package"}]' \
      <<<"$assets_links_json"
  )"
}

macos_versioned_name="${APP_NAME}-${TAG}-macos-universal.zip"
windows_versioned_name="${APP_NAME}-${TAG}-windows-amd64.zip"
checksums_versioned_name="${APP_NAME}-${TAG}-SHA256SUMS.txt"

if printf '%s\n' "${uploaded_names[@]}" | grep -Fxq "$macos_versioned_name"; then
  add_alias_link "${APP_NAME}-macos-universal.zip" "$macos_versioned_name"
fi
if printf '%s\n' "${uploaded_names[@]}" | grep -Fxq "$windows_versioned_name"; then
  add_alias_link "${APP_NAME}-windows-amd64.zip" "$windows_versioned_name"
fi
if printf '%s\n' "${uploaded_names[@]}" | grep -Fxq "$checksums_versioned_name"; then
  add_alias_link "${APP_NAME}-SHA256SUMS.txt" "$checksums_versioned_name"
fi

release_description="$(cat "$notes_file")"
create_payload="$(
  jq -n \
    --arg name "$TAG" \
    --arg tag "$TAG" \
    --arg description "$release_description" \
    --argjson links "$assets_links_json" \
    '{
      name: $name,
      tag_name: $tag,
      description: $description,
      assets: { links: $links }
    }'
)"

update_payload="$(
  jq -n \
    --arg name "$TAG" \
    --arg description "$release_description" \
    --argjson links "$assets_links_json" \
    '{
      name: $name,
      description: $description,
      assets: { links: $links }
    }'
)"

auth_header_name="JOB-TOKEN"
auth_token="$JOB_TOKEN"
if [[ -n "${GITLAB_TOKEN:-}" ]]; then
  auth_header_name="PRIVATE-TOKEN"
  auth_token="$GITLAB_TOKEN"
fi

tag_encoded="$(printf '%s' "$TAG" | jq -sRr @uri)"
release_url="${API_BASE}/projects/${PROJECT_ID}/releases/${tag_encoded}"
collection_url="${API_BASE}/projects/${PROJECT_ID}/releases"

echo "Publishing release ${TAG}"
if curl --silent --show-error --fail \
  --header "${auth_header_name}: ${auth_token}" \
  "$release_url" >/dev/null; then
  curl --fail --silent --show-error \
    --request PUT \
    --header "${auth_header_name}: ${auth_token}" \
    --header "Content-Type: application/json" \
    --data "$update_payload" \
    "$release_url" >/dev/null
else
  curl --fail --silent --show-error \
    --request POST \
    --header "${auth_header_name}: ${auth_token}" \
    --header "Content-Type: application/json" \
    --data "$create_payload" \
    "$collection_url" >/dev/null
fi

echo "Release ${TAG} published successfully."
