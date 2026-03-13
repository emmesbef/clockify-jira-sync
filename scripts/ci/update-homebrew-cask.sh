#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

APP_NAME="${APP_NAME:-jirafy-clockwork}"
ASSETS_DIR="${ASSETS_DIR:-release-assets}"
TAG="${CI_COMMIT_TAG:-${RELEASE_TAG:-}}"
HOMEBREW_TAP_TOKEN="${HOMEBREW_TAP_TOKEN:-${RELEASE_PLEASE_TOKEN:-${GITHUB_TOKEN:-${GH_TOKEN:-}}}}"
HOMEBREW_TAP_OWNER="${HOMEBREW_TAP_OWNER:-emmesbef}"
HOMEBREW_TAP_REPO="${HOMEBREW_TAP_REPO:-homebrew-tap}"
HOMEBREW_TAP_BRANCH="${HOMEBREW_TAP_BRANCH:-main}"
HOMEBREW_CASK_PATH="${HOMEBREW_CASK_PATH:-Casks/jirafy-clockwork.rb}"
HOMEBREW_CASK_DOWNLOAD_BASE_URL="${HOMEBREW_CASK_DOWNLOAD_BASE_URL:-https://gitlab.com/level-87/clockify-jira-sync/-/raw/main/downloads}"
HOMEBREW_CASK_APP_BUNDLE_NAME="${HOMEBREW_CASK_APP_BUNDLE_NAME:-JiraFy Clockwork.app}"

if [[ -z "${TAG}" ]]; then
  echo "Error: CI_COMMIT_TAG or RELEASE_TAG is required." >&2
  exit 1
fi

if [[ -z "${HOMEBREW_TAP_TOKEN}" ]]; then
  echo "Error: a GitHub token is required (HOMEBREW_TAP_TOKEN, RELEASE_PLEASE_TOKEN, GITHUB_TOKEN, or GH_TOKEN)." >&2
  exit 1
fi

version="${TAG#v}"
macos_asset="downloads/${APP_NAME}-${TAG}-macos-universal.zip"
if [[ ! -f "${macos_asset}" ]]; then
  echo "Error: expected prebuilt macOS asset not found at ${macos_asset}" >&2
  echo "Generate it on macOS with 'wails build -clean -platform darwin/universal' and commit the zipped app to downloads/." >&2
  exit 1
fi

app_bundle_name="$(
  zipinfo -1 "${macos_asset}" | awk -F/ '/\.app\/$/ {print $1; exit}'
)"
if [[ -z "${app_bundle_name}" ]]; then
  echo "Error: no .app bundle directory found in ${macos_asset}" >&2
  exit 1
fi
if [[ "${app_bundle_name}" != "${HOMEBREW_CASK_APP_BUNDLE_NAME}" ]]; then
  echo "Error: expected app bundle '${HOMEBREW_CASK_APP_BUNDLE_NAME}' in ${macos_asset}, found '${app_bundle_name}'." >&2
  echo "Rebuild and zip the macOS app using the branded bundle directory name." >&2
  exit 1
fi

sha256="$(sha256sum "${macos_asset}" | awk '{print $1}')"
echo "Updating Homebrew cask to version ${version} (sha256 ${sha256}, app ${HOMEBREW_CASK_APP_BUNDLE_NAME})"

cask_content="$(
  cat <<EOF
cask "jirafy-clockwork" do
  version "${version}"
  sha256 "${sha256}"

  url "${HOMEBREW_CASK_DOWNLOAD_BASE_URL}/${APP_NAME}-v#{version}-macos-universal.zip"
  name "JiraFy Clockwork"
  desc "Desktop app to sync Clockify time entries with Jira worklogs"
  homepage "https://level-87.gitlab.io/"

  app "${HOMEBREW_CASK_APP_BUNDLE_NAME}"

  postflight do
    system_command "/usr/bin/xattr",
                   args: ["-cr", "#{appdir}/${HOMEBREW_CASK_APP_BUNDLE_NAME}"]
  end

  zap trash: [
    "~/Library/Application Support/jirafy-clockwork",
    "~/Library/Application Support/clockify-jira-sync",
    "~/Library/Preferences/com.wails.jirafy-clockwork.plist",
    "~/Library/Preferences/com.wails.clockify-jira-sync.plist",
  ]
end
EOF
)"

api_url="https://api.github.com/repos/${HOMEBREW_TAP_OWNER}/${HOMEBREW_TAP_REPO}/contents/${HOMEBREW_CASK_PATH}"
existing_response="$(
  curl --silent --show-error \
    --header "Authorization: Bearer ${HOMEBREW_TAP_TOKEN}" \
    --header "Accept: application/vnd.github+json" \
    --header "X-GitHub-Api-Version: 2022-11-28" \
    "${api_url}?ref=${HOMEBREW_TAP_BRANCH}" || true
)"
existing_sha="$(jq -r '.sha // empty' <<<"${existing_response}" 2>/dev/null || true)"

encoded_content="$(printf '%s' "${cask_content}" | base64 | tr -d '\n')"
commit_message="chore: bump jirafy-clockwork cask to ${version}"

payload="$(
  jq -n \
    --arg message "${commit_message}" \
    --arg content "${encoded_content}" \
    --arg branch "${HOMEBREW_TAP_BRANCH}" \
    --arg sha "${existing_sha}" \
    '{
      message: $message,
      content: $content,
      branch: $branch
    } + (if $sha != "" then { sha: $sha } else {} end)'
)"

curl --fail --silent --show-error \
  --request PUT \
  --header "Authorization: Bearer ${HOMEBREW_TAP_TOKEN}" \
  --header "Accept: application/vnd.github+json" \
  --header "X-GitHub-Api-Version: 2022-11-28" \
  --header "Content-Type: application/json" \
  --data "${payload}" \
  "${api_url}" >/dev/null

echo "Homebrew cask updated: ${HOMEBREW_TAP_OWNER}/${HOMEBREW_TAP_REPO}/${HOMEBREW_CASK_PATH}"
