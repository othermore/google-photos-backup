package cmd

import (
	"fmt"
	"google-photos-backup/internal/browser"
	"google-photos-backup/internal/config"
	"path/filepath"

	"github.com/spf13/cobra"
)

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

		userDataDir := filepath.Join(config.AppConfig.BackupPath, "browser_data")

		// Lanzar navegador en modo headless
		bm := browser.New(userDataDir, false) // Headless false para depurar visualmente
		defer bm.Close()

		if err := bm.RequestTakeout(); err != nil {
			fmt.Printf("❌ Error durante la solicitud de Takeout: %v\n", err)
			return
		}

		fmt.Println("\n✅ Proceso de solicitud finalizado. Google te enviará un email cuando la exportación esté lista.")
	},
}
