package cmd

import (
	"fmt"
	"google-photos-backup/internal/config"
	"google-photos-backup/internal/i18n" // <--- Importante
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "google-photos-backup",
	Short: "Google Photos Hybrid Backup Tool", // Short description in English generally ok
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		i18n.Init()         // <--- Detectar idioma PRIMERO
		config.InitConfig() // Luego la config
	},
	// ... resto del código ...
}

func init() {
	rootCmd.AddCommand(configureCmd) // <--- AÑADIR ESTA LÍNEA
	rootCmd.AddCommand(syncCmd)      // <--- Registrar comando sync
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
