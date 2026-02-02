package cmd

import (
	"fmt"
	"google-photos-backup/internal/config"
	"google-photos-backup/internal/i18n" // <--- Importante
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	rootCmd.AddCommand(configureCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
