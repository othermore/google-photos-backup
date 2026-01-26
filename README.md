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
* **Portable:** Single static binary with no complex dependencies.

## Installation

### From Source (Requires Go 1.20+)

```bash
git clone [https://github.com/your-username/google-photos-backup.git](https://github.com/your-username/google-photos-backup.git)
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

Follow the on-screen instructions. This will generate a config file at `~/.config/google-photos-backup/config.yaml`.

## Usage

(Coming soon)

## Credits
Developed by http://antonio.mg with the help of gemini