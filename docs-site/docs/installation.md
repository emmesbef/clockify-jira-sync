---
sidebar_position: 1
title: Installation
---

# Installation

## Download

Grab the latest release for your platform from the [GitHub Releases page](https://github.com/emmesbef/clockify-jira-sync/releases). Each release includes:

| Asset | Platform |
| --- | --- |
| `clockify-jira-sync-vX.Y.Z-macos-universal.zip` | macOS (Apple Silicon + Intel) |
| `clockify-jira-sync-vX.Y.Z-windows-amd64.zip` | Windows (64-bit) |
| `clockify-jira-sync-vX.Y.Z-SHA256SUMS.txt` | Checksums for verifying downloads |

## macOS

### 1. Unzip

Double-click the downloaded `.zip` file or run:

```bash
unzip clockify-jira-sync-*-macos-universal.zip
```

### 2. Open the app (Gatekeeper workaround)

Because the app is not notarized with an Apple Developer certificate, macOS Gatekeeper will block it on first launch with a message like:

> _"Apple konnte nicht überprüfen, ob „clockify-jira-sync" frei von Schadsoftware ist."_
>
> _"Apple could not verify whether 'clockify-jira-sync' is free of malware."_

This is expected for open-source apps that are not distributed through the Mac App Store. Choose **one** of these methods to open it:

**Option A — Right-click → Open (recommended)**

1. Right-click (or Control-click) `clockify-jira-sync.app` in Finder.
2. Select **Open** from the context menu.
3. In the dialog that appears, click **Open**.

macOS remembers your choice — subsequent launches work normally.

**Option B — Remove the quarantine flag**

Run this command once in Terminal:

```bash
xattr -cr /path/to/clockify-jira-sync.app
```

Then open the app normally by double-clicking it.

### 3. Optional: move to Applications

Drag `clockify-jira-sync.app` into your `/Applications` folder for easy access.

## Windows

### 1. Extract

Right-click the downloaded `.zip` file → **Extract All**, or use your preferred archive tool.

### 2. Run

Double-click `clockify-jira-sync.exe`.

Windows SmartScreen may show a warning for unsigned binaries. Click **More info → Run anyway** to proceed.

## Verify download integrity

Each release includes a `SHA256SUMS.txt` file. To verify your download:

```bash
# macOS / Linux
sha256sum -c clockify-jira-sync-vX.Y.Z-SHA256SUMS.txt

# Windows (PowerShell)
Get-FileHash clockify-jira-sync-vX.Y.Z-windows-amd64.zip -Algorithm SHA256
# Compare the output with the value in the SHA256SUMS.txt file
```

## Code signing status

The release binaries are **ad-hoc signed** on macOS, which means they carry a local signature but are **not notarized** by Apple. This is sufficient for personal and development use. Full Apple notarization requires an Apple Developer Program membership and may be added in a future release.

Windows binaries are currently unsigned. Authenticode signing may be added in the future.

## Next steps

Once the app is running, head to [Setup & Configuration](./setup-configuration.md) to connect your Clockify and Jira accounts.
