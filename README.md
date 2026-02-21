> [!IMPORTANT]
> This is still in development. Do not use it yet (or do at your own risk). I expect to have a first release by end of March 2026.

# Google Photos Backup (Linux/macOS)

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

## Advanced Configuration (`config.yaml`)

Configuration is stored in `config.yaml`. The tool looks for it in the following locations:
* **Linux**: `/etc/google-photos-backup/` or `~/.config/google-photos-backup/`
* **macOS**: `~/.config/google-photos-backup/` or the current directory `./`

### Configuration Parameters
*   `working_path`: Directory for temporary files, processing, and browser session data (`browser_data/`).
*   `backup_path`: Final destination for organized photos (`Backup/YYYY/MM/...`).
*   `rclone_remote`: Name of your rclone remote (e.g., `drive:`).
*   `email_alert_to`: Email address to receive stale backup alerts (requires `msmtp`).
*   `immich_master_enabled`: (`true`/`false`) Enables the Immich read-only repository integration.
*   `immich_master_path`: The path where the `immich-master` folder will be kept (usually inside `backup_path`).
*   `fix_ambiguous_metadata`: (`yes`, `no`, `interactive`) How to handle photos with missing/ambiguous JSON dates.
*   *Legacy Fields*: `client_id`, `client_secret`, and `token_path` are deprecated since authentication uses the browser directly.

### Example `config.yaml`
```yaml
working_path: "/var/lib/gpb/work"
backup_path: "/mnt/storage/photos"
rclone_remote: "gdrive:"
email_alert_to: "alerts@mydomain.com"
immich_master_enabled: true
immich_master_path: "/mnt/storage/photos/immich-master"
fix_ambiguous_metadata: "yes"
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