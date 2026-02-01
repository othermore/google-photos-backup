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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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
			fmt.Println("❌ Error: El directorio de backup no está configurado. Por favor, ejecuta './gpb configure' primero.")
			return
		}

		// Asegurarse de que el directorio de backup existe
		if err := os.MkdirAll(config.AppConfig.BackupPath, 0755); err != nil {
			fmt.Printf("❌ Error creando directorio de backup: %v\n", err)
			return
		}

		userDataDir := filepath.Join(config.AppConfig.BackupPath, "browser_data")

		// Cargar registro de exportaciones (history.json en la carpeta de backup)
		regPath := filepath.Join(config.AppConfig.BackupPath, "history.json")
		reg, err := registry.New(regPath)
		if err != nil {
			fmt.Printf("⚠️  No se pudo cargar el historial: %v\n", err)
		}

		// Lanzar navegador en modo headless
		bm := browser.New(userDataDir, false) // Headless false para depurar visualmente
		defer bm.Close()

		// 1. Comprobar estado actual
		statuses, err := bm.CheckExportStatus()
		if err != nil {
			fmt.Printf("❌ Error comprobando estado: %v\n", err)
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
					fmt.Printf("   - Asociando exportación %s a solicitud local pendiente.\n", st.ID)
					entry = reg.Get(st.ID)
				} else {
					// Si no hay huérfanas, creamos una nueva (importación pura)
					fmt.Printf("   - Importando exportación externa: %s (%s)\n", st.ID, st.StatusText)
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
					fmt.Printf("⚠️  Ignorando exportación expirada (Quota Exceeded previo): %s\n", st.ID)
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
			fmt.Println(i18n.T("sync_wait"))

			// Comprobar antigüedad
			// 1. Usar fecha detectada en la web (más fiable)
			createdAt := inProgressStatus.CreatedAt

			// Si tenemos fecha, comprobamos si es antigua (> 48h)
			if !createdAt.IsZero() && time.Since(createdAt) > 48*time.Hour {
				fmt.Printf("⚠️  La exportación lleva demasiado tiempo (%v). Se cancelará.\n", createdAt)
				if err := bm.CancelExport(); err != nil {
					fmt.Printf("❌ Error cancelando: %v\n", err)
					return
				}
				// Continuamos para solicitar una nueva
			} else {
				return
			}
		}

		if completedStatus != nil {
			fmt.Println("🎉 ¡Exportación lista para descargar!")

			// Crear carpeta de descargas específica para esta exportación
			// Ej: backup_path/downloads/ID_EXPORTACION
			downloadDir := filepath.Join(config.AppConfig.BackupPath, "downloads", completedStatus.ID)
			if err := os.MkdirAll(downloadDir, 0755); err != nil {
				fmt.Printf("❌ Error creando directorio de descarga: %v\n", err)
				return
			}

			count, size, err := bm.DownloadExport(completedStatus.ID, downloadDir)
			// DEPRECATED: Using new flow below.
			// Ideally we remove above lines.
			_ = count
			_ = size
			_ = err

			// NEW FLOW:
			fmt.Println("🚀 Iniciando gestor de descargas robusto...")

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
					fmt.Println(i18n.T("discarding_bad_state")) // You might want to add this key or just print
					// fmt.Println("⚠️  Discarding invalid legacy file list. Will re-scan.")
					entry.Files = nil
					reg.Update(*entry)
					reg.Save()
				} else {
					// Valid files, migrate them
					fmt.Println("📦 Migrating download state to separate file...")
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
				fmt.Printf("📋 Recuperando lista de ficheros guardada (%d ficheros).\n", len(filesToDownload))
			}

			// 3. If no state, fetch from Browser
			if len(filesToDownload) == 0 {
				fmt.Println(i18n.T("obtaining_list"))
				files, err := bm.GetDownloadList(completedStatus.ID)
				if err != nil {
					fmt.Printf("Error obteniendo lista de descarga: %v\n", err)
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
					fmt.Printf("❌ Failed to save download state: %v\n", err)
				}

				fmt.Printf("✅ Lista guardada: %d ficheros.\n", len(files))
			}

			// 4. Start Download with Progress
			pwd := config.AppConfig.Password
			err = bm.DownloadFiles(completedStatus.ID, filesToDownload, downloadDir, pwd, func(idx int, updatedFile registry.DownloadFile) {
				// Update in memory list
				filesToDownload[idx] = updatedFile

				// Save state to disk
				state := registry.DownloadState{
					ID:          entry.ID,
					Files:       filesToDownload,
					LastUpdated: time.Now(),
				}
				// We ignore error here to avoid spamming, but ideally log it
				_ = state.Save(statePath)

				// Print Global Progress
				// We need to sum up all files' bytes
				var totalDownloaded int64
				var totalSize int64
				activeCount := 0
				completedCount := 0
				totalFiles := len(filesToDownload)

				for _, f := range filesToDownload {
					totalDownloaded += f.DownloadedBytes
					totalSize += f.SizeBytes
					if f.Status == "downloading" {
						activeCount++
					} else if f.Status == "completed" {
						completedCount++
					}
				}

				// Calculate Total GB and Percent
				// Note: totalSize will increase as downloads start and report their size.
				// Initial totalSize might be 0.

				currentGB := float64(totalDownloaded) / 1024 / 1024 / 1024
				totalGB := float64(totalSize) / 1024 / 1024 / 1024
				percent := 0.0
				if totalSize > 0 {
					percent = (float64(totalDownloaded) / float64(totalSize)) * 100
				}

				// Format status line
				// Clear line partially? \r
				// "Active: 2 | Done: 1/16 | 4.50 GB / 20.00 GB (22.5%)"
				fmt.Printf("\r⬇️  Active: %d | Done: %d/%d | %.2f GB / %.2f GB (%.1f%%)   ",
					activeCount, completedCount, totalFiles, currentGB, totalGB, percent)
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
				fmt.Printf("⚠️  La descarga finalizó con errores: %v\n", err)
				// Don't mark as downloaded if failed
			} else {
				fmt.Println("✅ Descarga completada. Ficheros guardados en:", downloadDir)
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
			fmt.Printf("✅ La última copia exitosa fue el: %s\n", lastSuccess.CompletedAt.Format("02/01/2006 15:04"))
			fmt.Printf("   Archivos: %d | Tamaño: %s | Nuevas fotos: %d\n",
				lastSuccess.FileCount, lastSuccess.TotalSize, lastSuccess.NewPhotosCount)

			fmt.Printf("⏳ No toca nueva copia todavía (Frecuencia: %s). Próxima: %s\n", frequency, nextBackup.Format("02/01/2006 15:04"))
			fmt.Println("   Usa --force para ignorar esta comprobación.")
			return
		}

		if err := bm.RequestTakeout(); err != nil {
			fmt.Printf("❌ Error durante la solicitud de Takeout: %v\n", err)
			return
		}

		// Guardar fecha de solicitud
		reg.Add(registry.ExportEntry{
			RequestedAt: time.Now(),
			Status:      registry.StatusRequested,
		})
		if err := reg.Save(); err != nil {
			fmt.Printf("❌ Error guardando historial: %v\n", err)
		} else {
			fmt.Printf("📝 Historial actualizado en: %s\n", regPath)
		}

		fmt.Println(i18n.T("sync_success"))
	},
}
