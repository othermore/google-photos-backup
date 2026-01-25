# Google Photos Backup (Linux/macOS)

[![en](https://img.shields.io/badge/lang-en-red.svg)](README.md)
[![es](https://img.shields.io/badge/lang-es-yellow.svg)](README.es.md)

CLI tool to perform local, incremental backups of your Google Photos library.

Designed to be run manually or via Cron on Linux servers (Debian, RedHat, etc.) and macOS.

## Features

* **Hybrid Approach:** Uses the official API for metadata and listing, and a headless browser scraper for full-quality downloads (bypassing API compression).
* **Incremental:** Maintains a local index (`index.jsonl`) to download only new items.
* **Headless:** Configurable via files, perfect for servers without a GUI.
* **Portable:** Single static binary with no complex dependencies.

## Installation

### From Source (Requires Go 1.20+)

```bash
git clone [https://github.com/your-username/google-photos-backup.git](https://github.com/your-username/google-photos-backup.git)
cd google-photos-backup
go build -o gpb main.go
```

## Configuration

To use this app, you must create your own Google Cloud credentials.

### 1. Get Google Credentials

1.  Go to [Google Cloud Console](https://console.cloud.google.com/).
2.  Create a **New Project**.
3.  Enable **"Google Photos Library API"**.
4.  Configure **OAuth Consent Screen** (User Type: External). Add your email to "Test Users".
5.  Create **Credentials** -> **OAuth Client ID** (Type: Desktop App).
6.  **Important:** Add `http://localhost:8085/callback` to "Authorized redirect URIs".
7.  Copy your **Client ID** and **Client Secret**.

### 2. Setup

Run the configuration wizard:

```bash
./gpb configure
```

Follow the on-screen instructions. This will generate a config file at `~/.config/google-photos-backup/config.yaml`.

## Usage

(Coming soon)

## Credits
Developed by http://ntonio.mg with the help of gemini