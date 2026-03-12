---
sidebar_position: 1
title: Installation
---

# Installation

## Download

Grab the latest release for your platform from the [GitLab Releases page](https://gitlab.com/level-87/jirafy-clockwork/-/releases). Each release includes:

| Asset | Platform |
| --- | --- |
| `jirafy-clockwork-vX.Y.Z-macos-universal.zip` | macOS (Apple Silicon + Intel) |
| `jirafy-clockwork-vX.Y.Z-windows-amd64.zip` | Windows (64-bit) |
| `jirafy-clockwork-vX.Y.Z-SHA256SUMS.txt` | Checksums for verifying downloads |

## macOS

### Option A: Manual download (Recommended)

### 1. Unzip

Double-click the downloaded `.zip` file or run:

```bash
unzip jirafy-clockwork-*-macos-universal.zip
```

### 2. Open the app (first launch only)

Because the app is ad-hoc signed (not notarized by Apple), macOS Gatekeeper will ask for confirmation on first launch:

1. **Right-click** (or Control-click) `jirafy-clockwork.app` in Finder.
2. Select **Open** from the context menu.
3. In the dialog that appears, click **Open**.

macOS remembers your choice — subsequent launches work normally by double-clicking.

:::tip
If right-click → Open doesn't show an "Open" button on your macOS version, go to **System Settings → Privacy & Security**, scroll down, and click **"Open Anyway"** next to the blocked app. Alternatively, run `xattr -cr /path/to/jirafy-clockwork.app` in Terminal.
:::

### 3. Optional: move to Applications

Drag `jirafy-clockwork.app` into your `/Applications` folder for easy access.

### Option B: Homebrew cask

Install from the tap:

```bash
brew install --cask level-87/tap/jirafy-clockwork
```

To update:

```bash
brew upgrade --cask jirafy-clockwork
```

Tap maintainers should point the cask URL at GitLab-hosted release assets:

```ruby
url "https://gitlab.com/level-87/jirafy-clockwork/-/packages/generic/jirafy-clockwork/v#{version}/jirafy-clockwork-v#{version}-macos-universal.zip"
```

This keeps Homebrew distribution independent from GitHub release hosting.

## Windows

### 1. Extract

Right-click the downloaded `.zip` file → **Extract All**, or use your preferred archive tool.

### 2. Run

Double-click `jirafy-clockwork.exe`.

Windows SmartScreen may show a warning for unsigned binaries. Click **More info → Run anyway** to proceed.

## Verify download integrity

Each release includes a `SHA256SUMS.txt` file. To verify your download:

```bash
# macOS / Linux
sha256sum -c jirafy-clockwork-vX.Y.Z-SHA256SUMS.txt

# Windows (PowerShell)
Get-FileHash jirafy-clockwork-vX.Y.Z-windows-amd64.zip -Algorithm SHA256
# Compare the output with the value in the SHA256SUMS.txt file
```

## Code signing status

The release binaries are **ad-hoc signed** on macOS, which means they carry a local signature but are **not notarized** by Apple. This is sufficient for personal and development use. Full Apple notarization requires an Apple Developer Program membership and may be added in a future release.

Windows binaries are currently unsigned. Authenticode signing may be added in the future.

## Next steps

Once the app is running, head to [Setup & Configuration](./setup-configuration.md) to connect your Clockify and Jira accounts.
