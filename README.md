# Google Photos Backup (Linux/macOS)

[![en](https://img.shields.io/badge/lang-en-red.svg)](README.md)
[![es](https://img.shields.io/badge/lang-es-yellow.svg)](README.es.md)

CLI application to perform local and incremental backups of your Google Photos library.

Designed to be run manually or via Cron on Linux servers (Debian, RedHat, etc.) and macOS.

## Features

* **Hybrid:** Uses the official API to list files and a direct download system (browser automation) to get maximum quality (bypassing API compression).
* **Incremental:** Maintains a local index (`index.jsonl`) to download only new photos.
* **Headless:** Configurable via files, perfect for servers without a GUI.
* **Portable:** Single binary without complex external dependencies.

## Installation

### From Source (Requires Go 1.20+)

Prerequisites:
*   **Google Chrome** or **Chromium**: Must be installed on the system for the downloader to work.
*   **Go 1.20+**: To build the project.

```bash
git clone https://github.com/your-username/google-photos-backup.git
cd google-photos-backup
go build -o gpb main.go
```

## Configuration

Before using the app, you need to obtain Google credentials.

### 1. Get Google Credentials

1.  Go to [Google Cloud Console](https://console.cloud.google.com/).
2.  Create a **New Project**.
3.  Enable **"Google Photos Library API"**.
4.  Configure **OAuth Consent Screen** (User Type: External). Add your email to "Test Users".
5.  Create **Credentials** -> **OAuth Client ID** (Type: Desktop App).
6.  **Important:** Add `http://localhost:8085/callback` to "Authorized redirect URIs".
7.  Copy your **Client ID** and **Client Secret**.

### 2. Run the configurator

Run the following command in your terminal:

```bash
./gpb configure
```

Follow the on-screen instructions. This will generate a config file at `~/.config/google-photos-backup/config.yaml`.

## Usage

### Available Commands

#### `configure`
Starts the interactive wizard to configure credentials and directories.

```bash
./gpb configure
```

## Developer Info

Project structure and file descriptions. Keep this list updated when adding or modifying files.

*   `.gitignore`: Git ignored files.
*   `.project_context.md`: Context and rules for the AI assistant.
*   `cmd/`: Application commands (Cobra).
    *   `configure.go`: Logic for the `configure` command.
    *   `root.go`: CLI entry point.
    *   `sync.go`: Logic for the `sync` command.
    *   `utils.go`: Shared utilities for commands.
*   `go.mod` / `go.sum`: Go dependency management.
*   `internal/`: Internal application code.
    *   `api/client.go`: HTTP client for Google Photos Library API.
    *   `auth/auth.go`: OAuth2 flow and token management.
    *   `config/config.go`: Configuration management (viper).
    *   `downloader/browser.go`: Browser automation engine (go-rod).
    *   `i18n/i18n.go`: Internationalization system (EN/ES).
    *   `index/store.go`: Local database for file tracking.
    *   `utils/browser.go`: General browser utilities (opening URLs).
*   `main.go`: Binary entrypoint.
*   `README.es.md`: Documentation in Spanish.
*   `README.md`: Documentation in English.

## Credits
Developed by http://antonio.mg with help from Gemini.