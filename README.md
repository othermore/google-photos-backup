> [!IMPORTANT]
> Utility in Beta state. Use it at your own risk.
> * **v0.9.0 (Current)**: Tool is functionally mature, but limited testing has been performed.
> * **v0.9.1 (Target: End of March)**: Will introduce a mechanism interacting with the Immich API to automatically recreate your Google Photos albums based on the downloaded directory structure. (Assumes images are added to Immich as an external library).
> * **v1.0.0 (TBD)**: Final, stable release. Will be published once the tool has been battle-tested by multiple users over a few months. 
> 
> **👋 Are you using `gpb` or does it look interesting to you?**
> We'd love to hear from you! Please consider [dropping a message in this issue](https://github.com/othermore/google-photos-backup/issues/3) to let us know, or create new issues to share your feedback. Your support keeps the project motivated! Check out [CONTRIBUTING.md](CONTRIBUTING.md) to see how you can help.

# Google Photos Hybrid Backup Tool (gpb) v0.9.0

A command-line tool written in Go that automates Google Photos Takeout backups, incrementally downloading and extracting files into a year/month folder structure structure (`Backup/YYYY/MM`), whilst automatically eliminating 100% of duplicates both within the current backup batch and across the entire historical backup archive using zero-space hardlinks.

[![en](https://img.shields.io/badge/lang-en-red.svg)](README.md)
[![es](https://img.shields.io/badge/lang-es-yellow.svg)](README.es.md)

CLI tool to maintain local, incremental backups of your Google Photos library and make them accessible from **Immich**, with minimal user intervention.

Designed to be run manually or via Cron on Linux servers (Debian, RedHat, etc.) and macOS.
> **Note on "Unattended" usage**: Due to Google's security policies (Passkeys and re-authentication), this tool is **not** totally unattended. You will need to manually interact with the browser window at least once a year (at best) when the scheduled exports expire or Google requires re-verification.

## Features

*   **Direct**: Configure exports via "Email link", and the tool will download them directly by periodically polling Takeout via the browser.
*   **Drive**: Configure and automate recurring exports to Google Drive using `rclone`.
*   **Import**: Manually process existing Takeout ZIPs.
*   **Tool**: Technical tools for configuration, indexing, and Immich integrations.
*   **Optimized Storage Pipeline**: Downloads, Unzips, Corrections, Deduplication, and Cleanup happen in a streaming pipeline to minimize disk usage.
*   **Original Quality**: Ensures download of original files with full metadata (JSON dates fixed).
*   **Smart Deduplication**: Uses hardlinks for cross-snapshot deduplication (Zero Space for duplicates).
*   **Immich Integration**: Generates a read-only `immich-master` repository so your backup can be directly served by Immich without duplicating data.
*   **Email Alerts**: Notifies you if backups become stale (e.g., if Google stops sending exports or requires re-authentication) via system `msmtp`.
*   **Headless**: Configurable via files, perfect for servers without a GUI.

## 🚀 Installation & Build

You can install the tool by downloading a pre-compiled binary or building it yourself.

### Method 1: Download from Releases (Recommended)
You can download the latest pre-compiled binaries for Linux (amd64, arm64) and macOS (Intel, Apple Silicon) directly from the **[Releases page](https://github.com/othermore/google-photos-backup/releases)**.

1. Download the `.tar.gz` for your operating system.
2. Extract it and place the `gpb` binary in a directory in your PATH (e.g., `/usr/local/bin`).
3. Make it executable: `chmod +x /usr/local/bin/gpb`

### Method 2: Compile from Source (Requires Go 1.21+)

Alternatively, you can clone the repository and build the binary yourself.

```bash
git clone https://github.com/othermore/google-photos-backup.git
cd google-photos-backup
go build -o gpb main.go
chmod +x gpb
```

## Basic Usage

1. Create a `config.yaml` using the example provided below.
2. Ensure you have installed chromium or google chrome locally.
3. If using `gpb drive`, authorize `rclone` and test with `rclone ls <your_remote>:Takeout`

## ⚡ Getting Started: Quick setup (Recommended)

The most robust, automated, and recommended long-term setup is the **Drive Mode**. In this mode, Google exports photos to your Google Drive cloud account, and the tool silently picks them up in the background from your local server.

Initial configuration from scratch:
1. **Configure Rclone**: Install and authorize `rclone` with your Google account (see [Rclone (For Drive Mode)](#1-rclone-for-drive-mode) section).
2. **Run the Wizard**: Launch `./gpb tool configure`. Follow the interactive instructions to define your storage paths and log into Google via the embedded browser to authorize the tool.
3. **Schedule Execution**: Draft a job in your system's `cron` to execute `./gpb drive download` periodically (for example, every night). The tool will automatically and silently check for new Takeout exports to fetch out of Drive (see [Automated Drive Backup](#1-automated-drive-backup-recommended)).
4. **Schedule Exports (Optional)**: Run `./gpb drive schedule` manually if you want the tool to instruct Google to automatically generate bi-monthly exports for you, or configure them yourself from the Google Takeout dashboard.
5. **Emails (Optional)**: Install `msmtp` on your server and link it to the tool to receive email alerts whenever something fails or the automated Google Takeout exports expire (see [msmtp (For Email Alerts)](#2-msmtp-for-email-alerts) section).



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

### 2. Direct Browser Backup (Takeout "Email link")
Best if you do not use `rclone`. Although Takeout calls this "Send download link via email", this tool does not actually read your emails. Instead, it periodically checks Google Takeout directly via the browser to see if new exports have been generated, and sequentially downloads them.

**Step A: Schedule Exports**
* `gpb direct schedule`: Configures recurring exports (every 2 months for 1 year).
* `gpb direct schedule-once`: Configures a single, one-time export.

**Step B: Unattended Direct Download**
Run this command via **Cron** (e.g., daily). It will passively check for new exports without any user intervention and process them automatically.
```bash
./gpb direct download
```
> **Tip**: If more than 2.5 months (75 days) pass without a successful backup (e.g., the scheduled exports expired), it will alert you via email every 7 days so you can manually run the schedule command again (just like Drive mode).

### 3. Technical Tools
The `tool` command regroups all configuration and maintenance tasks:
* `gpb tool configure`: Interactive configuration wizard.
* `gpb tool fix-metadata`: Retroactively corrects filesystem modification dates using JSON sidecars.
* `gpb tool rebuild-index`: Rebuilds local indices.
* `gpb tool fix-hardlinks`: Validates and repairs cross-volume hardlinks.
* `gpb tool rebuild-immich-master`: Synchronizes snapshot with an `immich-master` read-only repository.

**Correct Execution Order for Rebuilding**
If you ever need to rebuild your entire database or fix massive date corruption issues inside the `immich-master` library, it is vital to execute the tools in this exact sequential order to prevent dragging errors:
1. `fix-metadata` to physically repair the files first.
2. `rebuild-index` to capture the healthy dates inside the indexes.
3. `fix-hardlinks` to tie duplicates together based on the proper indexes.
4. `rebuild-immich-master` to move the correct and deduplicated structure over to Immich.

*Chained command example:*
```bash
./gpb tool fix-metadata ; ./gpb tool rebuild-index ; ./gpb tool fix-hardlinks ; ./gpb tool rebuild-immich-master
```

### 4. Manual Import
If you have manually downloaded Takeout ZIPs, you can import them directly:

```bash
./gpb import /path/to/folder_with_zips
```

By default, the `import` command copies the ZIP files to a temporary working directory to protect your original files. If you are low on disk space, you can use the `--move-original` flag to move the files instead of copying them (the original files will be deleted after successful processing):

```bash
./gpb import /path/to/folder_with_zips --move-original
```

## Storage & Deduplication

The tool organizes files into a `Backup/YYYY/MM` structure.
*   **Snapshots**: Each run can update the existing structure or create snapshots (configurable).
*   **Hardlinks**: Identical files across backups (or imported multiple times) are hardlinked, using no additional space.
*   **Directory Splitting**: To prevent severe I/O timeouts on Network Attached Storage (NAS) devices, the central `immich-master` repository enforces a strict limit of 500 files per folder. If a month exceeds this limit, files are safely overflowed into subdirectories (e.g., `YYYY/MM/Part_1`, `YYYY/MM/Part_2`).

## Advanced Configuration (`config.yaml`)

Configuration is stored in `config.yaml`. The tool looks for it in the following locations by default:
* **Linux**: `/etc/google-photos-backup/` or `~/.config/google-photos-backup/`
* **macOS**: `~/.config/google-photos-backup/` or the current directory `./`

### Managing Multiple Accounts (The `--config-dir` flag)
You can manage backups for multiple Google accounts by creating separate configuration directories for each. This ensures isolated browser tokens and configurations. Use the global `--config-dir` flag with any command to specify which configuration directory to use:
```bash
# Example: Using a specific config directory for a second account
./gpb --config-dir ~/.config/gpb/account2 tool configure
./gpb --config-dir ~/.config/gpb/account2 drive download
```
This flag is global and works with all commands (`drive`, `direct`, `tool`, `import`).

### 🖥️ Headless / Remote Servers (GUI Workaround)
If you are running `gpb` on a headless server without a desktop environment (via SSH), any command that requires Google authentication (like `tool configure` or `drive request`) will fail or freeze when attempting to launch the Chromium login window, since it cannot render the graphical interface (`Could not open the default X display`).

The solution is to perform the initial login on your personal computer and transfer the portable session:

1. **On your local computer (Mac/Windows/Linux Desktop)**, run:
   ```bash
   ./gpb tool configure --config-dir ./my-server-config
   ```
2. The browser will open. Log into your Google Account normally and close the browser.
3. You will now have a local folder called `my-server-config` containing your `config.yaml` and a `browser_data/` folder holding your authenticated session.
4. **Transfer this entire folder to your server** via `scp`, `rsync`, or FTP:
   ```bash
   scp -r ./my-server-config user@my-server:~/my-server-config
   ```
5. You can now run all automated commands directly on your server by pointing to the transferred directory. The Chrome browser on the server will run in standard headless mode and flawlessly re-use your imported session:
   ```bash
   ./gpb drive download --config-dir ~/my-server-config
   ```

### Global File Logging (The `--log` flag)
You can instruct the tool to write execution traces (in English) to a standard Unix log file by using the global `--log` flag.
This is particularly useful when running the tool automatically via Cron, as it keeps a persistent record of downloaded files, extraction progress, deduplicated links, and any errors/summaries.
```bash
# Example: Writing execution logs to a file
./gpb --log /var/log/gpb.log drive download
```
This flag is supported by the `drive download`, `direct download`, and `import` commands.

### Configuration Parameters
*   `working_path`: Directory for temporary files, processing, and browser session data (`browser_data/`). **Pro tip**: If your final destination (`backup_path`) is a remote NAS or a slow hard drive (HDD), set your `working_path` to a fast local drive (SSD/NVMe). The tool will use this drive first to download, unzip and do all the heavy deduplication processing at maximum speed before transferring the final photos to the slower storage, saving hours of networking and disk bottlenecks.
*   `backup_path`: Final destination for organized photos (`Backup/YYYY/MM/...`).
*   `rclone_remote`: Name of your rclone remote (e.g., `drive:`).
*   `email_alert_to`: Email address to receive stale backup alerts (requires `msmtp`).
*   `immich_master_enabled`: (`true`/`false`) Enables the Immich read-only repository integration.
*   `immich_master_path`: The path where the `immich-master` folder will be kept (usually inside `backup_path`).
*   `fix_ambiguous_metadata`: (`yes`, `no`, `interactive`) How to handle photos with missing/ambiguous JSON dates.
*   `valid_media_extensions`: An array of allowed file extensions (e.g. `jpg`, `mp4`, `dng`, `heic`). Files without these extensions will be dropped during extraction or parsing, except `.json` files which are kept strictly for deduplication/metadata purposes.
*   `ignored_files`: An array of exact names or glob patterns to block system trash entirely across all scanning phases (e.g. `SYNOINDEX_MEDIA_INFO`, `@eaDir`, `*@synoeastream`).

### Example `config.yaml`
```yaml
working_path: "/var/lib/gpb/work"
backup_path: "/mnt/storage/photos"
rclone_remote: "gdrive:"
email_alert_to: "alerts@mydomain.com"
immich_master_enabled: true
immich_master_path: "/mnt/storage/photos/immich-master"
fix_ambiguous_metadata: "yes"
valid_media_extensions: ["jpg", "jpeg", "png", "gif", "mp4", "mov", "heic", "dng"]
ignored_files: ["SYNOINDEX_MEDIA_INFO", "@eaDir", "*@synoeastream"]
```

### Authentication Session Details
The tool uses a headless Chrome/Chromium browser to automate Google Takeout.
*   **Where is it saved?** All session cookies, Passkey trust tokens, and logins are saved inside `[working_path]/browser_data`.
*   **Keep it safe**: Do not delete this folder, or you will need to re-authenticate manually (which might require a physical passkey or 2FA device).

## System Tooling Setup

### 1. Rclone (For Drive Mode)
To use `gpb drive`, you need `rclone` authorized with your Google account.
*   **macOS / Linux**: Install via `sudo curl https://rclone.org/install.sh | sudo bash` or `brew install rclone`.
*   Run `rclone config`.
*   Create a `New remote` (`n`). Name it exactly as your `rclone_remote` in `config.yaml` (default is `drive`).
*   Select `Google Drive` (`drive`).
*   Leave Custom client credentials blank (or provide your own API keys for higher limits).
*   **Permissions**: When prompted for scope, you MUST select `1` (Full Access / Read and Write). Without write access, the tool will download your Takeouts but will fail to delete them from Google Drive afterwards, eventually causing you to run out of cloud storage space.
*   Follow the browser prompt to grant Rclone access to your Drive.

### 2. msmtp (For Email Alerts)
The tool uses the system's `msmtp` binary to send emails when backups are older than 2.5 months (75 days).
*   **macOS**: `brew install msmtp`
*   **Linux (Debian/Ubuntu)**: `sudo apt install msmtp msmtp-mta`
*   Configure `~/.msmtprc` (or `/etc/msmtprc`) with your SMTP details. Example for Gmail:
    ```ini
    defaults
    auth           on
    tls            on
    tls_trust_file /etc/ssl/certs/ca-certificates.crt
    
    account        default
    host           smtp.gmail.com
    port           587
    user           youremail@gmail.com
    password       your_app_password
    from           youremail@gmail.com
    ```
*   Set permissions: `chmod 600 ~/.msmtprc`

## Troubleshooting

*   **Google Login**: If scheduling hangs at login, run `gpb tool configure` and choose "Yes" to login interactively.
*   **Rclone**: Ensure `rclone lsd remote:` works before running `gpb drive download`.
*   **Stale Backups**: If you haven't backed up in >75 days, the download command will send an email alert every 7 days reminding you to manually run the schedule command (which requires a Passkey).

## Credits
Developed by http://antonio.mg with the help of gemini
