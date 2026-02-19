package cmd

import (
	"fmt"
	"google-photos-backup/internal/browser"
	"google-photos-backup/internal/config"
	"google-photos-backup/internal/downloader/rclone"
	"google-photos-backup/internal/engine"
	"google-photos-backup/internal/i18n"
	"google-photos-backup/internal/logger"
	"google-photos-backup/internal/notifier"
	"google-photos-backup/internal/registry"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var driveCmd = &cobra.Command{
	Use:   "drive",
	Short: "Automated Drive Backup (Cron mode)",
	Long:  `Checks Google Drive for new Takeout archives (batches). If found and ready, downloads and processes them. If not found and backup is stale, attempts auto-renewal or sends an alert.`,
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

		// Load Global Index ONCE for the session
		if err := eng.LoadGlobalIndex(); err != nil {
			logger.Warn("Failed to load global index: %v", err)
		}

		processedBatches := 0

		// SCENARIO A: Files Found - Batch Processing
		if len(files) > 0 {
			// Group files by Timestamp pattern: takeout-YYYYMMDDTHHMMSSZ-*.zip
			groups := make(map[string][]rclone.File)
			// Regex to capture timestamp. Matches: takeout-20260217T143536Z-001.zip or takeout-20260217T143536Z-3-016.zip
			// We want the YYYYMMDDTHHMMSSZ part.
			re := regexp.MustCompile(`takeout-(\d{8}T\d{6}Z)-.*\.zip`)

			for _, f := range files {
				matches := re.FindStringSubmatch(f.Name)
				if len(matches) > 1 {
					ts := matches[1]
					groups[ts] = append(groups[ts], f)
				} else {
					logger.Warn("File %s does not match expected pattern, skipping batch grouping.", f.Name)
				}
			}

			logger.Info("📂 Found %d potential batches in Drive.", len(groups))

			// Process each group
			for ts, groupFiles := range groups {
				logger.Info("   - Analyzing batch: %s (%d files)", ts, len(groupFiles))

				// Ready Check: Look for ...-001.zip (Signal)
				var signalFile *rclone.File
				for i := range groupFiles {
					f := &groupFiles[i]
					// Check for "special" small file
					if f.Name == fmt.Sprintf("takeout-%s-001.zip", ts) {
						signalFile = f
						break
					}
				}

				if signalFile == nil {
					// Fallback: If no explicit signal file, check if single file export
					if len(groupFiles) == 1 && strings.HasSuffix(groupFiles[0].Name, ".zip") {
						signalFile = &groupFiles[0] // Treat as signal (it's the only one)
					} else {
						logger.Info("   - Batch %s NOT READY (Waiting for -001.zip signal). Skipping.", ts)
						continue
					}
				}

				logger.Info("✅ Batch %s is READY. Processing...", ts)

				batchWorkDir := filepath.Join(config.AppConfig.WorkingPath, "processing", ts)
				if err := os.MkdirAll(batchWorkDir, 0755); err != nil {
					logger.Error("Failed to create batch dir: %v", err)
					continue
				}

				// Create Engine scoped to this batch directory
				// This ensures Finalize() looks in batchWorkDir/extracted
				batchEng := engine.New(batchWorkDir, config.AppConfig.BackupPath)
				// Share the Global Index (Reference copy)
				batchEng.GlobalIndex = eng.GlobalIndex

				// Resume Info: Check if we have an existing index
				indexPath := filepath.Join(batchWorkDir, "index.json")
				if idx, err := registry.LoadIndex(indexPath); err == nil && len(idx.Files) > 0 {
					logger.Info("🔄 Resuming Batch: Found local index with %d files processed.", len(idx.Files))
				}

				// 1. Recover Orphans (Downloaded but not processed/deleted)
				// If script crashed after download but before delete
				logger.Info("   - Checking for orphan files...")
				localBatches, _ := filepath.Glob(filepath.Join(batchWorkDir, "*.zip"))
				for _, zipPath := range localBatches {
					if err := batchEng.ProcessZipWithIndex(zipPath, batchWorkDir); err != nil {
						logger.Error(i18n.T("drive_process_fail"), filepath.Base(zipPath), err)
					}
				}

				// Sort Drive files
				sort.Slice(groupFiles, func(i, j int) bool {
					return groupFiles[i].Name < groupFiles[j].Name
				})

				// 2. Sequential Pipeline (Download -> Process -> Delete)
				failed := false
				for i, file := range groupFiles {
					// Skip Signal File (Process Last)
					if file.Name == signalFile.Name {
						continue
					}

					logger.Info(i18n.T("drive_download_prog"), i+1, len(groupFiles), file.Name)

					// Check if already processed (not in Drive? logic is: iterate Drive files)
					// If it is in `groupFiles`, it IS in Drive. So it needs processing.

					// Resume Cleanup: Remove partial downloads
					partial := filepath.Join(batchWorkDir, file.Name+".crdownload") // rclone partial?
					os.Remove(partial)                                              // just in case

					// Download (Move)
					// Verify rc.MoveFile logic in rclone.go: it moves file to localDir.
					if err := rc.MoveFile(file.Name, batchWorkDir); err != nil {
						logger.Error(i18n.T("drive_dl_move_fail"), file.Name, err)
						failed = true
						continue // Skip processing this file
					}

					// Process immediately
					localPath := filepath.Join(batchWorkDir, file.Name)
					if err := batchEng.ProcessZipWithIndex(localPath, batchWorkDir); err != nil {
						logger.Error(i18n.T("drive_process_fail"), file.Name, err)
						failed = true
					}
				}

				if failed {
					logger.Error("⚠️  Batch %s had failures. Stopping before Signal File to allow retry.", ts)
					continue
				}

				// 3. Process Signal File (Last)
				// If we are here, all content files are processed and deleted from Drive.
				logger.Info("🏁 Processing Signal File: %s", signalFile.Name)

				// It might be the ONLY file (if len=1, loop above was skipped)
				// Download Signal
				if err := rc.MoveFile(signalFile.Name, batchWorkDir); err != nil {
					logger.Error(i18n.T("drive_dl_move_fail"), signalFile.Name, err)
					continue
				}

				// Process Signal
				localPath := filepath.Join(batchWorkDir, signalFile.Name)
				if err := batchEng.ProcessZipWithIndex(localPath, batchWorkDir); err != nil {
					logger.Error(i18n.T("drive_process_fail"), signalFile.Name, err)
					continue
				}

				// 4. Finalize Batch
				if err := batchEng.Finalize(); err != nil {
					logger.Error(i18n.T("drive_final_fail"), err)
				} else {
					logger.Info(i18n.T("drive_processed_success"))
					updateHistorySuccess()

					// Cleanup Batch Dir (Index, empty extracted)
					os.RemoveAll(batchWorkDir)
					processedBatches++
				}
			}

			// Finalize Engine (Shared Phase)
			if processedBatches > 0 {
				if err := eng.Finalize(); err != nil {
					logger.Error(i18n.T("drive_final_fail"), err)
				} else {
					logger.Info(i18n.T("drive_processed_success"))
					updateHistorySuccess()
				}
			}

		}

		// SCENARIO B: No Files Found (or all skipped)
		if processedBatches == 0 {
			if len(files) == 0 {
				logger.Info(i18n.T("drive_no_files"))
			} else {
				logger.Info("ℹ️  Files found but no batches were ready to process.")
			}
			checkStaleAndAlert()
		}
	},
}

func updateHistorySuccess() {
	// Simple tracker for last success
	regPath := filepath.Join(config.AppConfig.WorkingPath, "history.json")
	reg, _ := registry.New(regPath)
	if reg != nil {
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
	// User requested increase to 3 months (approx 90 days)
	if time.Since(last.CompletedAt) > 90*24*time.Hour {
		logger.Warn(i18n.T("drive_stale_warn"))

		// Smart Stale Handling: Limit alerts frequency (7 days)
		alertStatePath := filepath.Join(config.AppConfig.WorkingPath, "alert_state.txt")
		lastAlert := time.Time{}
		if data, err := os.ReadFile(alertStatePath); err == nil {
			lastAlert, _ = time.Parse(time.RFC3339, string(data))
		}

		if time.Since(lastAlert) < 7*24*time.Hour {
			logger.Info(i18n.T("drive_alert_skip"), lastAlert.Format("2006-01-02"))
			return
		}

		// Attempt Auto-Renewal (Headless Schedule)
		logger.Info("🔄 Attempting auto-renewal of Takeout schedule (Headless)...")

		userDataDir := filepath.Join(config.AppConfig.WorkingPath, "browser_data")
		// Headless = true
		bm := browser.New(userDataDir, true)
		defer bm.Close()

		// Verify Session & Schedule
		if bm.VerifySession() {
			if err := bm.ScheduleRecurringTakeout(); err == nil {
				logger.Info("✅ Auto-renewal successful! Google should prepare a new export soon.")
				// We treat this as a "partial success" to reset alert timer?
				// Or update history "RequestedAt"?
				// Let's just NOT alert.
				// Update state to prevent immediate retry?
				// Maybe add a history entry "drive-auto-renew"?
				reg.Add(registry.ExportEntry{
					ID:          "drive-auto-renew-" + time.Now().Format("20060102"),
					Status:      registry.StatusPending, // It's requested
					RequestedAt: time.Now(),
					CompletedAt: last.CompletedAt, // Keep last completed same
				})
				reg.Save()
				return
			} else {
				logger.Warn("Auto-renewal failed: %v", err)
			}
		} else {
			logger.Warn("Auto-renewal skipped: Session invalid.")
		}

		// Fallback: Send Alert Email
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
	}
}

func init() {
	rootCmd.AddCommand(driveCmd)
}
