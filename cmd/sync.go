package cmd

import (
	"fmt"
	"google-photos-backup/internal/browser"
	"google-photos-backup/internal/config"
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
		if config.AppConfig.BackupPath == "" {
			logger.Error(i18n.T("backup_dir_error"))
			return
		}

		// Asegurarse de que el directorio de backup existe
		if err := os.MkdirAll(config.AppConfig.BackupPath, 0755); err != nil {
			logger.Error(i18n.T("backup_mkdir_error"), err)
			return
		}

		userDataDir := filepath.Join(config.AppConfig.BackupPath, "browser_data")

		// Cargar registro de exportaciones (history.json en la carpeta de backup)
		regPath := filepath.Join(config.AppConfig.BackupPath, "history.json")
		reg, err := registry.New(regPath)
		if err != nil {
			fmt.Printf("⚠️  No se pudo cargar el historial: %v\n", err)
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
			logger.Info("🧹 Removed %d incomplete/ghost entries from history.", len(reg.Exports)-len(validExports))
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
						ID:          st.ID,
						RequestedAt: st.CreatedAt,              // Puede ser zero si no se parseó
						Status:      registry.StatusInProgress, // Default, se actualizará abajo
					}
					reg.Add(newEntry)
					entry = reg.Get(st.ID)
				}
			}

			// Actualizar estado
			updated := false
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

			// Crear carpeta de descargas específica para esta exportación
			// Ej: backup_path/downloads/ID_EXPORTACION
			downloadDir := filepath.Join(config.AppConfig.BackupPath, "downloads", completedStatus.ID)
			if err := os.MkdirAll(downloadDir, 0755); err != nil {
				logger.Error(i18n.T("download_dir_error"), err)
				return
			}

			count, size, err := bm.DownloadExport(completedStatus.ID, downloadDir)
			// DEPRECATED: Using new flow below.
			// Ideally we remove above lines.
			_ = count
			_ = size
			_ = err

			// NEW FLOW:
			logger.Info(i18n.T("starting_manager"))

			// 1. Obtener lista de ficheros (si no la tenemos ya en registro)
			entry := reg.Get(completedStatus.ID)

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
						fmt.Printf("❌ Failed to migrate state: %v\n", err)
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

			// 4. Start Download with Progress
			// Check download mode
			mode := entry.DownloadMode
			if mode == "" {
				mode = config.ModeDirectDownload
			}

			if mode == config.ModeDriveDownload {
				logger.Info(i18n.T("drive_mode_warning"))
				return
			}

			// Init Tracker
			tracker := &ProgressTracker{
				StartTime:       time.Now(),
				TotalFiles:      len(filesToDownload),
				TotalExportSize: browser.ParseSize(entry.TotalSize),
				Files:           filesToDownload,
			}
			tracker.Render() // Initial render

			err = bm.DownloadFiles(completedStatus.ID, filesToDownload, downloadDir, func(idx int, updatedFile registry.DownloadFile) {
				// Update in memory list
				filesToDownload[idx] = updatedFile
				tracker.Files = filesToDownload // Sync files to tracker (ref)

				// Re-render
				tracker.Render()

				// Save state to disk
				state := registry.DownloadState{
					ID:          entry.ID,
					Files:       filesToDownload,
					LastUpdated: time.Now(),
				}
				_ = state.Save(statePath)
			})
			fmt.Println() // Newline after loop or progress

			if err != nil {
				if err == browser.ErrQuotaExceeded {
					fmt.Println("⛔ Limite de descargas excedido (Quota Exceeded).")
					fmt.Println("⚠️  Marcando exportación como EXPIRADA y limpiando datos parciales.")

					// CLEANUP: Wipe the directory to save space and remove bad state
					if err := os.RemoveAll(downloadDir); err != nil {
						fmt.Printf("❌ Error eliminando directorio de descarga: %v\n", err)
					} else {
						fmt.Println("🧹 Directorio de descarga eliminado.")
					}

					entry.Status = registry.StatusExpired
					entry.Files = nil
					reg.Update(*entry)
					reg.Save()
					return // Loop will continue? Or return to main? Current loop is over exports.
					// We return to allow next run to request new.
				}
				// We need a "Downloaded" status.
				// For now, let's leave as Downloading but maybe set a flag or logic?
				// Or assume Processed means Downloaded? No, Processed means Metadata fixed.

				// Let's just say "Descarga finalizada"
				logger.Error(i18n.T("download_finished_error"), err)
				// Don't mark as downloaded if failed
			} else {
				logger.Info(i18n.T("download_completed"), downloadDir)
			}
			reg.Update(*entry)
			reg.Save()

			return
		}

		// 2. Si no hay nada en curso, comprobar frecuencia antes de solicitar nueva
		lastSuccess := reg.GetLastSuccessful()
		frequency := viper.GetDuration("backup_frequency")
		force, _ := cmd.Flags().GetBool("force")

		// Si hay una copia exitosa reciente, no hacemos nada
		if !force && lastSuccess != nil && time.Since(lastSuccess.CompletedAt) < frequency {
			nextBackup := lastSuccess.CompletedAt.Add(frequency)
			logger.Info(i18n.T("last_success"), lastSuccess.CompletedAt.Format("02/01/2006 15:04"))
			logger.Info(i18n.T("last_stats"),
				lastSuccess.FileCount, lastSuccess.TotalSize, lastSuccess.NewPhotosCount)

			logger.Info(i18n.T("next_backup"), frequency, nextBackup.Format("02/01/2006 15:04"))
			logger.Info(i18n.T("use_force"))
			return
		}

		// Check config mode for new export
		mode := config.AppConfig.DownloadMode
		if mode == "" {
			mode = config.ModeDirectDownload
		}

		if mode == config.ModeDriveDownload {
			logger.Info(i18n.T("drive_mode_new"))
			return
		}

		if err := bm.RequestTakeout(mode); err != nil {
			logger.Error(i18n.T("takeout_req_error"), err)
			return
		}

		// Double-check status to get the new ID immediately
		// This ensures we don't save a ghost entry.
		time.Sleep(5 * time.Second) // Give it a moment
		newStatuses, err := bm.CheckExportStatus()
		newID := ""
		if err == nil {
			for _, st := range newStatuses {
				// If we find one that is InProgress (or Created recently), use it
				if st.InProgress {
					newID = st.ID
					break
				}
			}
		}

		if newID != "" {
			logger.Info("✅ New export created with ID: %s", newID)
			reg.Add(registry.ExportEntry{
				ID:          newID,
				RequestedAt: time.Now(),
				Status:      registry.StatusInProgress,
			})
		} else {
			// Fallback if we can't find the ID yet (maybe slow backend)
			// We save it as "requested" but without ID.
			// Ideally we shouldn't do this if we want to avoid ghosts,
			// but we need to record that we tried.
			// With the cleanup logic at start, this is safe-ish.
			logger.Info("⚠️  Export created but ID not yet visible. Saving as pending.")
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
