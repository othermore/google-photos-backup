# Google Photos Backup (Linux/macOS)

[![en](https://img.shields.io/badge/lang-en-red.svg)](README.md)
[![es](https://img.shields.io/badge/lang-es-yellow.svg)](README.es.md)

CLI tool to perform local, incremental backups of your Google Photos library.

Designed to be run manually or via Cron on Linux servers (Debian, RedHat, etc.) and macOS.

## Features

* **Automated Takeout:** Automates the request and download of full backups via Google Takeout.
* **Original Quality:** Ensures download of original files with full metadata.
* **Smart Organization:** Processes downloaded files to fix EXIF dates (using Google's JSONs) and organizes photos into albums.
* **Headless:** Configurable via files, perfect for servers without a GUI.
* **Robust Downloading:**
    * Resumes interrupted downloads automatically.
    * Handles "Quota Exceeded" errors by marking exports as expired.
    * Validates file sizes before marking as complete.
* **User Experience:**
    * Detailed progress dashboard with Speed and ETA.
    * Verbose output option for debugging.
    * Internationalized (English/Spanish).

## Installation

### From Source (Requires Go 1.20+)

```bash
git clone https://github.com/your-username/google-photos-backup.git
cd google-photos-backup
go build -o gpb main.go
```

## Configuration

The app uses an automated browser (Chrome/Chromium). You will need to log in manually the first time to save the session.

### Setup

Run the configuration wizard:

```bash
./gpb configure
```

Follow the instructions. This will authorize the tool to access your Google Takeout data.

### Configuration File

The config is stored at `~/.config/google-photos-backup/config.yaml`.

**Available Options:**

| Key | Default | Description |
| :--- | :--- | :--- |
| `backup_path` | `./backup` | Main directory where backups will be stored. |
| `backup_frequency` | `168h` | How often to request a new backup (e.g., `24h`, `168h` = 7 days). |
| `download_mode` | `directDownload` | Mode of operation. Currently only `directDownload` is supported. |
| `user_data_dir` | (auto) | Path to Chrome user profile data (do not change unless necessary). |

## Usage

### 1. Sync (Main Command)

Checks the status of your exports and handles the flow (Request -> Wait -> Download).

```bash
./gpb sync [flags]
```

**Flags:**

* `-v, --verbose`: Enable detailed debug logging (shows browser clicks, navigation URLs, etc.).
* `--force`: Ignore the `backup_frequency` check and force a new export request immediately.

### How it Works

1.  **Check Status:** The tool logs into Google Takeout to check for active exports.
    *   **In Progress:** If an export is creating, it waits. If it's older than 48h, it cancels it.
    *   **Ready:** If an export is ready (`Download` button available), it starts downloading.
    *   **None:** If no export is active and the frequency timer has passed, it requests a new one (50GB split).
2.  **Downloading:**
    *   Files are downloaded to your system's default Downloads folder initially.
    *   They are automatically moved to `<backup_path>/downloads/<EXPORT_ID>/`.
    *   A `.download_state.json` file tracks progress, allowing resumes if the process is interrupted.
3.  **Completion:**
    *   Once all files are downloaded, the tool updates the `history.json` index.

## Troubleshooting

*   **Google Login Issues:** If the tool gets stuck verifying session, try running `./gpb configure` again and ensuring you complete the login flow in the browser window.
*   **"Quota Exceeded":** Google limits the number of times you can download a Takeout archive (usually 5-10 times). If this error occurs, the tool will mark the export as expired and request a new one in the next run.
*   **Verbose Mode:** Run with `gpb sync -v` to see exactly what the browser automation is doing.

## Credits
Developed by http://antonio.mg with the help of gemini