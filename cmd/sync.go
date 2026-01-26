package cmd

import (
	"fmt"
	"google-photos-backup/internal/browser"
	"google-photos-backup/internal/config"
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
		fmt.Println("Starting Google Takeout automation...")

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
			fmt.Println("⏳ Ya hay una exportación en curso.")

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
				fmt.Println("   Google está preparando tus archivos. Vuelve a intentarlo más tarde.")
				return
			}
		}

		if completedStatus != nil {
			fmt.Println("🎉 ¡Exportación lista para descargar!")
			fmt.Println("TODO: Implementar lógica de descarga en la siguiente fase.")
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

		fmt.Println("\n✅ Proceso de solicitud finalizado. Google te enviará un email cuando la exportación esté lista.")
	},
}
