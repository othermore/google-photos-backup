> [!IMPORTANT]
> This is still in development. Do not use it yet (or do at your own risk). I expect to have a first release by end of March 2026.

# Google Photos Backup (Linux/macOS)

[![en](https://img.shields.io/badge/lang-en-red.svg)](README.md)
[![es](https://img.shields.io/badge/lang-es-yellow.svg)](README.es.md)

CLI tool to perform local, incremental backups of your Google Photos library.

Designed to be run manually or via Cron on Linux servers (Debian, RedHat, etc.) and macOS.

## Features

* **Three Modes of Operation:**
    *   **Sync**: Interactive direct download from Google Takeout.
    *   **Schedule + Drive**: Configure recurring 2-monthly exports to Drive and automatically download/sync them using `rclone`.
    *   **Import**: Manually process existing Takeout ZIPs.
*   **Optimized Storage Pipeline**: Downloads, Unzips, Corrections, Deduplication, and Cleanup happen in a streaming pipeline to minimize disk usage.
*   **Original Quality**: Ensures download of original files with full metadata (JSON dates fixed).
*   **Smart Deduplication**: Uses hardlinks for cross-snapshot deduplication (Zero Space for duplicates).
*   **Email Alerts**: Notifies you if backups become stale (via system `msmtp`).
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
./gpb configure
```

This will set up your:
*   Working Directory (temp space)
*   Backup Directory (final storage)
*   Rclone Remote (for Drive mode)
*   Email for alerts

## Usage

### 1. Interactive Sync (Manual Request)
Best for one-off backups or initial runs. It logs into Google, requests a download, and creates a local backup.

```bash
./gpb sync
```

### 2. Automated Drive Backup (Recommended)
This method is fully automated and robust.

**Step A: Schedule Recurring Exports**
Run this **once** to configure Google Takeout to export your photos to Drive every 2 months for 1 year.

```bash
./gpb schedule
```

**Step B: Unattended Drive Sync**
Run this command via **Cron** (e.g., daily). It checks your Drive for new exports, downloads them, processes them, and deletes them from Drive to save cloud space.

```bash
./gpb drive
```

**Example Cron:**
```bash
0 3 * * * /path/to/gpb drive >> /var/log/gpb.log 2>&1
```

### 3. Manual Import
If you have manually downloaded Takeout ZIPs, you can import them:

```bash
./gpb import /path/to/folder_with_zips
```

## Storage & Deduplication

The tool organizes files into a `Backup/YYYY/MM` structure.
*   **Snapshots**: Each run can update the existing structure or create snapshots (configurable).
*   **Hardlinks**: Identical files across backups (or imported multiple times) are hardlinked, using no additional space.

## Troubleshooting

*   **Google Login**: If `schedule` or `sync` hangs at login, run `gpb configure` and chose "Yes" to login interactively.
*   **Rclone**: Ensure `rclone lsd remote:` works before running `gpb drive`.
*   **Stale Backups**: If you haven't backed up in >30 days, `gpb drive` will try to send an email alert if configured.

## Credits
Developed by http://antonio.mg with the help of gemini