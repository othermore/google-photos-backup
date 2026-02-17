package cmd

import (
	"fmt"
	"google-photos-backup/internal/browser"
	"google-photos-backup/internal/config"
	"google-photos-backup/internal/engine"
	"google-photos-backup/internal/i18n"
	"google-photos-backup/internal/registry"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google-photos-backup/internal/logger"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// SpeedSample represents a point in time for speed calculation
type SpeedSample struct {
	Time  time.Time
	Bytes int64
}

// ProgressTracker manages the display of download progress
type ProgressTracker struct {
	StartTime       time.Time
	TotalFiles      int
	TotalExportSize int64 // bytes
	History         []SpeedSample
	Files           []registry.DownloadFile
	LastRenderLines int // Number of lines printed last time (for clearing)
}

func (pt *ProgressTracker) UpdateSpeed(currentBytes int64) float64 {
	now := time.Now()
	// Add new sample
	pt.History = append(pt.History, SpeedSample{Time: now, Bytes: currentBytes})

	// Remove samples older than 30s
	validIdx := 0
	for i, s := range pt.History {
		if now.Sub(s.Time) <= 30*time.Second {
			validIdx = i
			break
		}
	}
	pt.History = pt.History[validIdx:]

	// Calculate speed
	if len(pt.History) < 2 {
		return 0
	}

	oldest := pt.History[0]
	newest := pt.History[len(pt.History)-1]

	duration := newest.Time.Sub(oldest.Time).Seconds()
	if duration == 0 {
		return 0
	}

	bytesDiff := newest.Bytes - oldest.Bytes
	return float64(bytesDiff) / duration // bytes/sec
}

func (pt *ProgressTracker) Render() {
	// 1. Calculate Stats
	var totalDownloaded int64
	var totalActiveSize int64
	activeCount := 0
	completedCount := 0

	for _, f := range pt.Files {
		totalDownloaded += f.DownloadedBytes
		totalActiveSize += f.SizeBytes
		if f.Status == "downloading" {
			activeCount++
		} else if f.Status == "completed" {
			completedCount++
		}
	}

	// Calculate Speed & ETA
	speedBps := pt.UpdateSpeed(totalDownloaded)
	speedMBps := speedBps / 1024 / 1024

	// Denominator
	denominator := pt.TotalExportSize
	if denominator == 0 {
		denominator = totalActiveSize
	}

	percent := 0.0
	if denominator > 0 {
		percent = (float64(totalDownloaded) / float64(denominator)) * 100
	}

	etaStr := "--:--"
	if speedBps > 0 && denominator > totalDownloaded {
		remainingBytes := denominator - totalDownloaded
		secondsRemaining := float64(remainingBytes) / speedBps
		eta := time.Duration(secondsRemaining) * time.Second
		etaStr = eta.String()
		// Simple format
		if secondsRemaining > 3600 {
			etaStr = fmt.Sprintf("%dh %02dm", int(secondsRemaining/3600), int(secondsRemaining)%3600/60)
		} else {
			etaStr = fmt.Sprintf("%02dm %02ds", int(secondsRemaining/60), int(secondsRemaining)%60)
		}
	}

	// 2. Prepare Output
	var lines []string

	// Summary Line
	currentGB := float64(totalDownloaded) / 1024 / 1024 / 1024
	totalGB := float64(denominator) / 1024 / 1024 / 1024

	lines = append(lines, fmt.Sprintf("⬇️  %s: %d | %s: %d/%d | %.2f GB / %.2f GB (%.1f%%) | ⚡ %.2f MB/s | ⏱️  %s: %s",
		i18n.T("progress_active"), activeCount, i18n.T("progress_done"), completedCount, pt.TotalFiles, currentGB, totalGB, percent, speedMBps, i18n.T("progress_eta"), etaStr))
	lines = append(lines, strings.Repeat("-", 80))

	// Detailed File List
	for _, f := range pt.Files {
		// e.g. "takeout-001.zip: [completed] 1.2 GB / 1.2 GB (100%)"
		// Shorten filename if needed?

		fPercent := 0.0
		if f.SizeBytes > 0 {
			fPercent = (float64(f.DownloadedBytes) / float64(f.SizeBytes)) * 100
		}

		statusIcon := "⚪"
		statusText := i18n.T("status_pending")
		switch f.Status {
		case "completed":
			statusIcon = "✅"
			statusText = i18n.T("status_completed")
		case "failed":
			statusIcon = "❌"
			statusText = i18n.T("status_failed")
		case "downloading":
			statusIcon = "⏳"
			statusText = i18n.T("status_downloading")
			if fPercent > 99.9 && f.SizeBytes > 0 {
				statusIcon = "💿"
				statusText = i18n.T("status_finalizing")
			}
		}

		fCurrentMB := float64(f.DownloadedBytes) / 1024 / 1024
		fTotalMB := float64(f.SizeBytes) / 1024 / 1024

		lines = append(lines, fmt.Sprintf("%s %-25s: %8.2f MB / %8.2f MB (%5.1f%%) [%s]",
			statusIcon, f.Filename, fCurrentMB, fTotalMB, fPercent, statusText))
	}
	lines = append(lines, "") // Empty footer

	// 3. Clear Previous Output & Print
	// Cursor Up logic
	if pt.LastRenderLines > 0 {
		// ANSI Escape: Move cursor up N lines
		fmt.Printf("\033[%dA", pt.LastRenderLines)
		// ANSI Escape: Clear from cursor to end of screen (optional, helps cleanly overwrite)
		// fmt.Print("\033[J")
	}

	for _, line := range lines {
		// Clear line first
		fmt.Print("\033[2K\r")
		fmt.Println(line)
	}

	pt.LastRenderLines = len(lines)
}

func init() {
	syncCmd.Flags().Bool("force", false, "Forzar nueva exportación ignorando la frecuencia configurada")
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Request a new Google Photos backup via Takeout",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(i18n.T("sync_start"))

		// Asegurarse de que la ruta de backup está configurada
		if config.AppConfig.WorkingPath == "" {
			logger.Error(i18n.T("backup_dir_error"))
			return
		}

		// Asegurarse de que el directorio de backup existe
		if err := os.MkdirAll(config.AppConfig.WorkingPath, 0755); err != nil {
			logger.Error(i18n.T("backup_mkdir_error"), err)
			return
		}

		userDataDir := filepath.Join(config.AppConfig.WorkingPath, "browser_data")

		// Cargar registro de exportaciones (history.json en la carpeta de backup)
		regPath := filepath.Join(config.AppConfig.WorkingPath, "history.json")
		reg, err := registry.New(regPath)
		if err != nil {
			fmt.Printf(i18n.T("sync_history_error")+"\n", err)
		}

		// CLEANUP: Remove ghost/stale entries (ID="") from previous failed runs
		// This prevents "requested" entries from piling up if the export wasn't actually created.
		validExports := []registry.ExportEntry{}
		for _, e := range reg.Exports {
			if e.ID != "" {
				validExports = append(validExports, e)
			}
		}
		if len(validExports) < len(reg.Exports) {
			logger.Info(i18n.T("sync_ghost_removed"), len(reg.Exports)-len(validExports))
			reg.Exports = validExports
			reg.Save()
		}

		// Lanzar navegador en modo headless
		bm := browser.New(userDataDir, false) // Headless false para depurar visualmente
		defer bm.Close()

		// 1. Comprobar estado actual
		statuses, err := bm.CheckExportStatus()
		if err != nil {
			logger.Error(i18n.T("status_check_error"), err)
			return
		}

		// Actualizar registro local con lo encontrado
		var inProgressStatus *browser.ExportStatus
		var completedStatus *browser.ExportStatus

		for _, st := range statuses {
			if st.ID == "" {
				continue
			}

			// Buscar si existe en el registro
			entry := reg.Get(st.ID)
			if entry == nil {
				// Si no existe, intentamos fusionar con una solicitud huérfana local
				if reg.MergeOrphan(st.ID, st.CreatedAt) {
					logger.Info(i18n.T("merging_orphan"), st.ID)
					entry = reg.Get(st.ID)
				} else {
					// Si no hay huérfanas, creamos una nueva (importación pura)
					logger.Info(i18n.T("importing_export"), st.ID, st.StatusText)
					newEntry := registry.ExportEntry{
						ID:           st.ID,
						RequestedAt:  st.CreatedAt,              // Puede ser zero si no se parseó
						Status:       registry.StatusInProgress, // Default, se actualizará abajo
						DownloadMode: st.DownloadMode,           // Saving detected mode
					}
					reg.Add(newEntry)
					entry = reg.Get(st.ID)
				}
			}

			// Actualizar estado
			updated := false

			// Update mode if we detected it and it was missing
			if st.DownloadMode != "" && entry.DownloadMode == "" {
				entry.DownloadMode = st.DownloadMode
				updated = true
			}

			if st.InProgress {
				inProgressStatus = &st
				if entry.Status != registry.StatusInProgress {
					entry.Status = registry.StatusInProgress
					updated = true
				}
				// Actualizar fecha si la tenemos y antes no
				if !st.CreatedAt.IsZero() && entry.RequestedAt.IsZero() {
					entry.RequestedAt = st.CreatedAt
					updated = true
				}
			} else if st.Completed {
				// 🚨 CRITICAL: If this export is marked as EXPIRED in our registry,
				// we must IGNORE it so that we don't try to download it again.
				// This forces the logic below to RequestTakeout() for a new one.
				if entry.Status == registry.StatusExpired {
					logger.Debug(i18n.T("ignoring_expired"), st.ID)
					continue
				}

				completedStatus = &st // Guardamos la última completada encontrada
				if entry.Status != registry.StatusReady && entry.Status != registry.StatusProcessed {
					entry.Status = registry.StatusReady // Lista para descargar
					entry.CompletedAt = time.Now()
					updated = true
				} else if (entry.Status == registry.StatusReady || entry.Status == registry.StatusProcessed) && entry.CompletedAt.IsZero() {
					// Si ya estaba marcada como lista pero no tenía fecha, le ponemos la actual (mejor que nada)
					entry.CompletedAt = time.Now()
					updated = true
				}
			} else if strings.Contains(strings.ToLower(st.StatusText), "cancel") {
				// Detecta "Canceled", "Cancelled", "Cancelado", etc.
				if entry.Status != registry.StatusCancelled {
					entry.Status = registry.StatusCancelled
					entry.CompletedAt = time.Now()
					updated = true
				} else if entry.Status == registry.StatusCancelled && entry.CompletedAt.IsZero() {
					entry.CompletedAt = time.Now()
					updated = true
				}
			}

			if updated {
				reg.Update(*entry)
			}
		}
		reg.Save()

		// Lógica de decisión
		if inProgressStatus != nil {
			logger.Info(i18n.T("sync_wait"))

			// Comprobar antigüedad
			// 1. Usar fecha detectada en la web (más fiable)
			createdAt := inProgressStatus.CreatedAt

			// Si tenemos fecha, comprobamos si es antigua (> 48h)
			if !createdAt.IsZero() && time.Since(createdAt) > 48*time.Hour {
				logger.Info(i18n.T("export_too_old"), createdAt)
				if err := bm.CancelExport(); err != nil {
					logger.Error(i18n.T("cancel_error"), err)
					return
				}
				// Continuamos para solicitar una nueva
			} else {
				return
			}
		}

		if completedStatus != nil {
			logger.Info(i18n.T("ready_to_download"))

			// 1. Obtener lista de ficheros (si no la tenemos ya en registro)
			entry := reg.Get(completedStatus.ID)

			// Check download mode - SYNC IS ALWAYS DIRECT DOWNLOAD
			// Legacy: If we find a 'driveDownload' here, we should warn or handle it.
			// The new registry logic should filter or we just process it if possible?
			// 'sync' command is specifically for direct download flow.
			// If we found a file-list, we just download it.

			// ... (ProgressTracker init code)

			// Crear carpeta de descargas específica para esta exportación
			// Ej: backup_path/downloads/ID_EXPORTACION
			downloadDir := filepath.Join(config.AppConfig.WorkingPath, "downloads", completedStatus.ID)
			if err := os.MkdirAll(downloadDir, 0755); err != nil {
				logger.Error(i18n.T("download_dir_error"), err)
				return
			}

			// NEW FLOW:
			logger.Info(i18n.T("starting_manager"))

			// Logic to migrate or load state
			statePath := filepath.Join(downloadDir, "state.json")
			var filesToDownload []registry.DownloadFile

			// 1. Check if we have legacy files in registry to migrate
			if len(entry.Files) > 0 {
				// Heuristic: If files have empty filenames and status failed, they might be from the "17 files" bug.
				// In that case, we discard them to force a re-scan.
				if entry.Files[0].Filename == "" {
					fmt.Println(i18n.T("discarding_bad_state"))
					// fmt.Println("⚠️  Discarding invalid legacy file list. Will re-scan.")
					entry.Files = nil
					reg.Update(*entry)
					reg.Save()
				} else {
					// Valid files, migrate them
					fmt.Println(i18n.T("migrating_state"))
					state := registry.DownloadState{
						ID:          entry.ID,
						Files:       entry.Files,
						LastUpdated: time.Now(),
					}
					if err := state.Save(statePath); err != nil {
						fmt.Printf(i18n.T("sync_migrate_fail")+"\n", err)
					} else {
						entry.Files = nil // Clear from registry
						reg.Update(*entry)
						reg.Save()
					}
				}
			}

			// 2. Load state from file if exists
			if state, err := registry.LoadDownloadState(statePath); err == nil {
				filesToDownload = state.Files
				logger.Info(i18n.T("recovering_list"), len(filesToDownload))

				// Check if any file is already downloaded (100% size) but not marked
				for i, f := range filesToDownload {
					if f.Status != "completed" && f.SizeBytes > 0 {
						targetFile := filepath.Join(downloadDir, f.Filename)
						// Check local file
						if info, err := os.Stat(targetFile); err == nil {
							if info.Size() >= f.SizeBytes {
								logger.Info(i18n.T("sync_found_completed"), f.Filename, browser.FormatSize(info.Size()))
								filesToDownload[i].Status = "completed"
								filesToDownload[i].DownloadedBytes = info.Size()
								// If we found it valid, ensure we don't try to download it again
							}
						}
					}
				}
			}

			// 3. If no state, fetch from Browser
			if len(filesToDownload) == 0 {
				fmt.Println(i18n.T("obtaining_list"))
				files, err := bm.GetDownloadList(completedStatus.ID)
				if err != nil {
					logger.Error(i18n.T("list_error"), err)
					return // Next export
				}
				filesToDownload = files

				// Save new state
				state := registry.DownloadState{
					ID:          entry.ID,
					Files:       files,
					LastUpdated: time.Now(),
				}
				if err := state.Save(statePath); err != nil {
					logger.Error(i18n.T("state_save_error"), err)
				}

				fmt.Printf(i18n.T("list_saved")+"\n", len(files))
			}

			// 4. Start Download with Progress (and Processing)

			// Init Engine
			eng := engine.New(config.AppConfig.WorkingPath, config.AppConfig.BackupPath)

			// Init Tracker
			tracker := &ProgressTracker{
				StartTime:       time.Now(),
				TotalFiles:      len(filesToDownload),
				TotalExportSize: browser.ParseSize(entry.TotalSize),
				Files:           filesToDownload,
			}

			nonInteractive := viper.GetBool("non_interactive")
			if !nonInteractive {
				tracker.Render() // Initial render
			} else {
				logger.Info(i18n.T("sync_export_set"), len(filesToDownload), entry.TotalSize)
			}

			err = bm.DownloadFiles(completedStatus.ID, filesToDownload, downloadDir, func(idx int, updatedFile registry.DownloadFile) {
				// Detect status changes for logging BEFORE updating memory
				oldStatus := filesToDownload[idx].Status
				newStatus := updatedFile.Status

				if nonInteractive {
					if oldStatus != "downloading" && newStatus == "downloading" {
						logger.Info(i18n.T("sync_download_start"), updatedFile.Filename, browser.FormatSize(updatedFile.SizeBytes))
					}
					// Completed logging handled below after processing
				}

				// If Completed, Trigger Processing
				if oldStatus != "completed" && newStatus == "completed" {
					// Download finished! Process Immediately!
					zipPath := filepath.Join(downloadDir, updatedFile.Filename)
					logger.Info("⚡ Downloaded %s. Processing immediately...", updatedFile.Filename)

					if err := eng.ProcessZip(zipPath); err != nil {
						logger.Error("❌ processing failed for %s: %v", updatedFile.Filename, err)
						// Mark as failed in tracker? Or just log?
						// Failure in processing shouldn't stop the next download, but is critical.
					} else {
						logger.Info("✅ Processed %s (Space freed).", updatedFile.Filename)
					}
				}

				// Update in memory list
				filesToDownload[idx] = updatedFile
				tracker.Files = filesToDownload

				// Re-render
				if !nonInteractive {
					tracker.Render()
				}

				// Save state to disk
				state := registry.DownloadState{
					ID:          entry.ID,
					Files:       filesToDownload,
					LastUpdated: time.Now(),
				}
				_ = state.Save(statePath)
			})
			fmt.Println()

			if err != nil {
				if err == browser.ErrQuotaExceeded {
					fmt.Println(i18n.T("sync_quota_exceeded"))
					// ... (cleanup logic) ...
					return
				}
				logger.Error(i18n.T("download_finished_error"), err)
			} else {
				logger.Info(i18n.T("download_completed"), downloadDir)

				// FINALIZE Pipeline
				logger.Info("🔄 Finalizing global processing...")
				if err := eng.Finalize(); err != nil {
					logger.Error("❌ Finalization failed: %v", err)
				} else {
					logger.Info("✅ All files processed and organized.")
					// Cleanup Download Dir (should be empty of zips, but might have state.json)
					os.RemoveAll(downloadDir)
				}
			}
			reg.Update(*entry)
			reg.Save()

			return
		}

		// 2. Si no hay nada en curso, solicitar nueva
		logger.Info(i18n.T("resquesting_new_direct"))

		if err := bm.RequestTakeout("directDownload"); err != nil {
			logger.Error(i18n.T("takeout_req_error"), err)
			return
		}

		// Double-check status
		time.Sleep(5 * time.Second)
		newStatuses, err := bm.CheckExportStatus()
		newID := ""
		if err == nil {
			for _, st := range newStatuses {
				if st.InProgress {
					newID = st.ID
					break
				}
			}
		}

		if newID != "" {
			logger.Info(i18n.T("sync_new_export"), newID)
			reg.Add(registry.ExportEntry{
				ID:          newID,
				RequestedAt: time.Now(),
				Status:      registry.StatusInProgress,
			})
		} else {
			logger.Info(i18n.T("sync_pending_export"))
			reg.Add(registry.ExportEntry{
				RequestedAt: time.Now(),
				Status:      registry.StatusRequested,
			})
		}

		if err := reg.Save(); err != nil {
			logger.Error(i18n.T("history_save_error"), err)
		} else {
			logger.Info(i18n.T("history_updated"), regPath)
		}

		fmt.Println(i18n.T("sync_success"))
	},
}
