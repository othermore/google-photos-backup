package cmd

import (
	"fmt"
	"google-photos-backup/internal/browser"
	"google-photos-backup/internal/config"
	"google-photos-backup/internal/registry"
	"os"
	"path/filepath"
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
		status, err := bm.CheckExportStatus()
		if err != nil {
			fmt.Printf("❌ Error comprobando estado: %v\n", err)
			return
		}

		if status.InProgress {
			fmt.Println("⏳ Ya hay una exportación en curso.")

			// Sincronizar con el registro si es una exportación huérfana
			if status.ID != "" {
				entry := reg.Get(status.ID)
				if entry == nil {
					fmt.Println("   - Detectada exportación no registrada. Añadiendo al historial...")
					reg.Add(registry.ExportEntry{
						ID:          status.ID,
						RequestedAt: status.CreatedAt, // Usamos la fecha detectada
						Status:      registry.StatusInProgress,
					})
					if err := reg.Save(); err != nil {
						fmt.Printf("❌ Error guardando historial: %v\n", err)
					}
				} else if entry.RequestedAt.IsZero() && !status.CreatedAt.IsZero() {
					fmt.Println("   - Corrigiendo fecha de solicitud en el historial...")
					entry.RequestedAt = status.CreatedAt
					reg.Update(*entry)
					if err := reg.Save(); err != nil {
						fmt.Printf("❌ Error guardando historial: %v\n", err)
					}
				}
			}

			// Comprobar antigüedad
			// 1. Usar fecha detectada en la web (más fiable)
			createdAt := status.CreatedAt

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

		if status.Completed {
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
