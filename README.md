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
| `working_path` | `./work` | Directory to store downloads and temporary processing files. |
| `backup_frequency` | `168h` | How often to request a new backup (e.g., `24h`, `168h` = 7 days). |
| `download_mode` | `directDownload` | Mode of operation. Currently only `directDownload` is supported. |
| `fix_ambiguous_metadata` | `interactive` | Default behavior for ambiguous matches (`yes`, `no`, `interactive`). |
| `backup_path` | (empty) | Path to the final backup destination (snapshots). |
| `user_data_dir` | (auto) | Path to Chrome user profile data (do not change unless necessary). |

## Export Status Lifecycle

The system manages the lifecycle of each Takeout through various statuses in `history.json`:

*   **`requested`**: Request initiated but not yet confirmed by Google.
*   **`in_progress`**: Google is preparing the archives. `sync` monitors this state.
*   **`ready`**: Google completed preparation. Files are ready to download (or have been downloaded).
    *   *Note*: `sync` downloads files in this state. `process` only acts on `ready` exports that are fully downloaded (verified via `state.json`).
*   **`expired`**: The export expired on Google's servers and cannot be downloaded.
*   **`cancelled`**: Export cancelled by user or system.
*   **`failed`**: Google failed to generate the export.

### Data Integrity

To ensure safety and avoid conflicts:
1.  **`history.json`**: Source of truth for exports. Modified by `sync`, read-only for `process`.
2.  **`state.json`**: Located in each download folder (e.g., `downloads/ID_XXX/state.json`), tracks download progress of individual ZIP files. It contains:
    *   `files`: List of expected files. Each has `status` ("completed", "pending"), `size_bytes` (expected size), and `downloaded_bytes`.
    *   *Constraint*: The `process` command STRICTLY validates that `status` is "completed" and `size_bytes` matches the actual file size before extracting.
3.  **`processing_index.json`**: Maintained by `process`, tracks which exports and files have been processed and organized to prevent duplicates.

## Usage

### 1. Sync (Main Command)

Checks the status of your exports and handles the flow (Request -> Wait -> Download).

```bash
./gpb sync [flags]
```

**Flags:**

* `-v, --verbose`: Enable detailed debug logging (shows browser clicks, navigation URLs, etc.).
* `--force`: Ignore the `backup_frequency` check and force a new export request immediately.

### 2. Process (Organization)

After `sync` downloads the files, this command extracts, fixes metadata, and organizes them.

```bash
./gpb process [flags]
```

**Workflow:**
1.  **Extraction**: Unzips new archives found in `downloads/` to a `raw/` subdirectory.
2.  **Metadata Correction**: Uses Google's JSON sidecar files to fix the "Date Modified" of your images/videos.
    *   **Matching Levels**:
        *   **Level 1 (Exact)**: Matches `file.json` or `file.supplemental-metadata.json`.
        *   **Level 2 (Clean)**: Matches filenames by stripping extensions (e.g., `IMG_123.jpg` -> `IMG_123.json`).
        *   **Level 3 (Fuzzy/Secure)**: Matches truncated filenames if the match length is **>40 characters** (prevents `IMG.json` from matching `IMG_1234.jpg`).
    *   **Ambiguous Matches**: Partial matches shorter than 40 chars use the behavior defined by `--fix-ambiguous-metadata`.
3.  **Global Deduplication**: Scans all processed files, identifies duplicates (SHA256), and keeps the best version (prioritizing album names). Duplicates are replaced with relative symlinks to save space.

**Flags:**

*   `--fix-ambiguous-metadata`: Behavior for ambiguous metadata matches (`yes`=apply, `no`=skip, `interactive`=ask). Default: `interactive`.
*   `--delete-origin`: Delete original ZIP/TGZ files after successful extraction to save space. Default: `true`.
*   `--force-metadata`: Re-runs date correction on already processed exports.
*   `--force-dedup`: Re-runs the duplicate check within the working directory.
*   `--force-extract`: Re-extracts archives. Overwrites existing files.
*   `--export <ID>`: Runs processing ONLY on the specified Export ID.

### 3. Update Backup (Final Sync)

After processing, this command synchronizes the organized files to your final storage location (e.g., NAS, external drive).

```bash
./gpb update-backup [flags]
```

**Features:**
*   **Snapshots:** Creates a timestamped folder for each backup run (`YYYY-MM-DD-HHMMSS`). **Suffixes are supported** (e.g., `2024-05-20-173000-MyTag`).
*   **Smart Deduplication:** Checks if files already exist in the *previous backup*. If content matches (Hash check), it creates a **hardlink** instead of copying. This saves massive Space!
*   **Logging:** Exact details of every operation are saved to `backup_log.jsonl` in the final directory.
*   **Cleanup:** If successful, deletes the processed files from `working_path` to free up space.

**Flags:**
*   `--dry-run`: Simulate the update without moving or deleting files.
*   `--source <dir>`: Manually specify the source directory (defaults to `working_path/downloads`).

### 4. Fix Hardlinks (Deduplication)

Scans your legacy or manually modified backup snapshots to maximize space savings by hardlinking identical files across snapshots.

```bash
./gpb fix-hardlinks [flags]
```

**Flags:**
*   `--dry-run`: Simulate the deduplication without modifying files.
*   `--path <dir>`: Path to the backup root (defaults to configured `final_backup_path`).

### 5. Immich Master Directory (Optional)

You can maintain a flattened, deduplicated directory structure optimized for **Immich** (or any external library). This folder organizes all your photos by Year/Month using **hardlinks only**.

**Key Features:**
*   **Zero Space:** Files are hardlinked to your snapshots. No physical copies are made.
*   **Index-Based:** Uses efficient `index.json` files to track content and avoid redundant scanning.
*   **Auto-Update:** `update-backup` automatically indexes new snapshots and links them to the master directory.

**Configuration (in `config.yaml`):**
```yaml
immich_master_enabled: true
immich_master_path: "immich-master" # relative to backup_path
```

**Commands:**

*   **Rebuild Master:**
    If you enable this feature later or change the path, you can regenerate the master directory from all existing snapshots:
    ```bash
    ./gpb rebuild-immich-master
    ```

### 6. Rebuild Indexes (Maintenance)

If you need to regenerate the `index.json` files for your snapshots (e.g., after manual changes or for fresh deduplication):

```bash
./gpb rebuild-index
```
*   **Optimized:** Uses Inode tracking to speed up re-indexing of unchanged files.


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