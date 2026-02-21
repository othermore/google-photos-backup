> [!IMPORTANT]
> This is still in development. Do not use it yet (or do at your own risk). I expect to have a first release by end of March 2026.

# Google Photos Backup (Linux/macOS)

[![en](https://img.shields.io/badge/lang-en-red.svg)](README.md)
[![es](https://img.shields.io/badge/lang-es-yellow.svg)](README.es.md)

CLI tool to maintain local, incremental backups of your Google Photos library and make them accessible from **Immich**, with minimal user intervention.

Designed to be run manually or via Cron on Linux servers (Debian, RedHat, etc.) and macOS.
> **Note on "Unattended" usage**: Due to Google's security policies (Passkeys and re-authentication), this tool is **not** totally unattended. You will need to manually interact with the browser window at least once a year (at best) when the scheduled exports expire or Google requires re-verification.

## Features

* **Four Modes of Operation:**
    *   **Direct**: Configure and download archives directly via Email links.
    *   **Drive**: Configure and automate recurring exports to Google Drive using `rclone`.
    *   **Import**: Manually process existing Takeout ZIPs.
    *   **Tool**: Technical tools for configuration, indexing, and Immich integrations.
*   **Optimized Storage Pipeline**: Downloads, Unzips, Corrections, Deduplication, and Cleanup happen in a streaming pipeline to minimize disk usage.
*   **Original Quality**: Ensures download of original files with full metadata (JSON dates fixed).
*   **Smart Deduplication**: Uses hardlinks for cross-snapshot deduplication (Zero Space for duplicates).
*   **Immich Integration**: Generates a read-only `immich-master` repository so your backup can be directly served by Immich without duplicating data.
*   **Email Alerts**: Notifies you if backups become stale (e.g., if Google stops sending exports or requires re-authentication) via system `msmtp`.
*   **Headless**: Configurable via files, perfect for servers without a GUI.

## Installation

### From Source (Requires Go 1.20+)

```bash
git clone https://github.com/your-username/google-photos-backup.git
cd google-photos-backup
go build -o gpb main.go
```

### Prerequisites
*   **Google Chrome / Chromium**: For browser automation (scheduling/requesting).
*   **Rclone**: Required for `drive` mode (downloading from Google Drive).
*   **msmtp** (Optional): For email alerts.

## Configuration

Run the configuration wizard:

```bash
./gpb tool configure
```

This will set up your:
*   Working Directory (temp space)
*   Backup Directory (final storage)
*   Rclone Remote (for Drive mode)
*   Email for alerts

## Usage

### 1. Automated Drive Backup (Recommended)
This method is fully automated and robust. It uses Google Drive to store temporary archives before downloading them using `rclone`, processing them sequentially, and deleting them from the cloud to save space.

**Step A: Schedule Exports**
Run this to configure Google Takeout to export your photos to Drive.
* `gpb drive schedule`: Configures recurring exports (every 2 months for 1 year).
* `gpb drive schedule-once`: Configures a single, one-time export.

**Step B: Unattended Drive Download**
Run this command via **Cron** (e.g., daily). It checks your Drive for new exports and seamlessly processes them.
```bash
./gpb drive download
```
**Example Cron:**
```bash
0 3 * * * /path/to/gpb drive download >> /var/log/gpb.log 2>&1
```

### 2. Direct Email Backup
Best if you do not use `rclone`. It configures exports via email and sequentially downloads them.

**Step A: Schedule Exports**
* `gpb direct schedule`: Configures recurring exports via Email.
* `gpb direct schedule-once`: Configures a single, one-time export via Email.

**Step B: Direct Download**
Wait until you receive the email from Google, then run:
```bash
./gpb direct download
```
> **Tip**: You can run `gpb direct download` daily via Cron. It will passively check for new exports and process them. If more than 2 months (60 days) pass without a successful backup, it will alert you via email (just like the Drive mode).

### 3. Technical Tools
The `tool` command regroups all configuration and maintenance tasks:
* `gpb tool configure`: Interactive configuration wizard.
* `gpb tool rebuild-index`: Rebuilds local indices.
* `gpb tool fix-hardlinks`: Validates and repairs cross-volume hardlinks.
* `gpb tool rebuild-immich-master`: Synchronizes snapshot with an `immich-master` read-only repository.

### 4. Manual Import
If you have manually downloaded Takeout ZIPs, you can import them directly:

```bash
./gpb import /path/to/folder_with_zips
```

## Storage & Deduplication

The tool organizes files into a `Backup/YYYY/MM` structure.
*   **Snapshots**: Each run can update the existing structure or create snapshots (configurable).
*   **Hardlinks**: Identical files across backups (or imported multiple times) are hardlinked, using no additional space.

## Troubleshooting

*   **Google Login**: If scheduling hangs at login, run `gpb tool configure` and choose "Yes" to login interactively.
*   **Rclone**: Ensure `rclone lsd remote:` works before running `gpb drive download`.
*   **Stale Backups**: If you haven't backed up in >90 days, `gpb drive download` will first try to **auto-renew** the schedule (headless, often works without Passkey). If that fails, it will send an email alert.

## Credits
Developed by http://antonio.mg with the help of gemini