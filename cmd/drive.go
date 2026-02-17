package cmd

import (
	"fmt"
	"google-photos-backup/internal/config"
	"google-photos-backup/internal/downloader/rclone"
	"google-photos-backup/internal/engine"
	"google-photos-backup/internal/i18n"
	"google-photos-backup/internal/logger"
	"google-photos-backup/internal/notifier"
	"google-photos-backup/internal/registry"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var driveCmd = &cobra.Command{
	Use:   "drive",
	Short: "Automated Drive Backup (Cron mode)",
	Long:  `Checks Google Drive for new Takeout archives. If found, downloads and processes them. If not found and backup is stale, sends an alert email.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info(i18n.T("drive_robot_start"))

		// 1. Config Check
		if config.AppConfig.BackupPath == "" {
			logger.Error(i18n.T("backup_dir_error"))
			return
		}

		// 2. Initialize Rclone
		rc := rclone.New(config.AppConfig.RcloneRemote)

		// 3. Check for Files in Drive
		logger.Info(i18n.T("drive_check"))
		files, err := rc.ListExports()
		if err != nil {
			logger.Error(i18n.T("drive_list_fail"), err)
			return
		}

		eng := engine.New(config.AppConfig.WorkingPath, config.AppConfig.BackupPath)

		// SCENARIO A: Files Found
		if len(files) > 0 {
			logger.Info(i18n.T("drive_found_count"), len(files))

			for i, file := range files {
				logger.Info(i18n.T("drive_download_prog"), i+1, len(files), file.Name)

				// Download to temp
				tempZip := filepath.Join(config.AppConfig.WorkingPath, "drive_temp", file.Name)
				os.MkdirAll(filepath.Dir(tempZip), 0755)

				if err := rc.MoveFile(file.Path, tempZip); err != nil {
					logger.Error(i18n.T("drive_dl_move_fail"), file.Name, err)
					continue
				}

				// Process Loop
				if err := eng.ProcessZip(tempZip); err != nil {
					logger.Error(i18n.T("drive_process_fail"), file.Name, err)
				}
			}

			// Finalize
			if err := eng.Finalize(); err != nil {
				logger.Error(i18n.T("drive_final_fail"), err)
			} else {
				logger.Info(i18n.T("drive_processed_success"))
				// Update History?
				// The engine doesn't update registry yet, maybe we should track success.
				updateHistorySuccess()
			}

		} else {
			// SCENARIO B: No Files Found
			logger.Info(i18n.T("drive_no_files"))
			checkStaleAndAlert()
		}
	},
}

func updateHistorySuccess() {
	// Simple tracker for last success
	regPath := filepath.Join(config.AppConfig.WorkingPath, "history.json")
	reg, _ := registry.New(regPath)
	if reg != nil {
		// Add a generic "Drive Backup" entry or update last success
		// Ideally we track real Export IDs but with Drive we might lose ID context if not in filename.
		// Let's just create a synthetic success entry.
		reg.Add(registry.ExportEntry{
			ID:          "drive-auto-" + time.Now().Format("20060102"),
			Status:      registry.StatusProcessed,
			CompletedAt: time.Now(),
			RequestedAt: time.Now(),
		})
		reg.Save()
	}
}

func checkStaleAndAlert() {
	regPath := filepath.Join(config.AppConfig.WorkingPath, "history.json")
	reg, err := registry.New(regPath)
	if err != nil {
		return
	}

	last := reg.GetLastSuccessful()
	if last == nil {
		return // Never backed up, maybe new install
	}

	// Check if > 3 months (90 days)
	if time.Since(last.CompletedAt) > 90*24*time.Hour {
		logger.Warn(i18n.T("drive_stale_warn"))

		// Check last alert time (Stored in config? or distinct file?)
		// Let's use a simple state file for alerts
		alertStatePath := filepath.Join(config.AppConfig.WorkingPath, "alert_state.txt")
		lastAlert := time.Time{}
		if data, err := os.ReadFile(alertStatePath); err == nil {
			lastAlert, _ = time.Parse(time.RFC3339, string(data))
		}

		// Alert every 7 days
		if time.Since(lastAlert) > 7*24*time.Hour {
			subject := i18n.T("drive_alert_subject")
			body := fmt.Sprintf(i18n.T("drive_alert_body"),
				last.CompletedAt.Format("2006-01-02"),
				time.Since(last.CompletedAt).String())

			if err := notifier.SendAlert(subject, body); err == nil {
				logger.Info(i18n.T("drive_alert_sent"))
				os.WriteFile(alertStatePath, []byte(time.Now().Format(time.RFC3339)), 0644)
			} else {
				logger.Error(i18n.T("drive_alert_fail"), err)
			}
		} else {
			logger.Info(i18n.T("drive_alert_skip"), lastAlert.Format("2006-01-02"))
		}
	}
}

func init() {
	rootCmd.AddCommand(driveCmd)
}
