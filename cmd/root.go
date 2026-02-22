package cmd

import (
	"fmt"
	"google-photos-backup/internal/config"
	"google-photos-backup/internal/i18n" // <--- Importante
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "google-photos-backup",
	Short: "Google Photos Hybrid Backup Tool", // Short description in English generally ok
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		i18n.Init()                // <--- Detectar idioma PRIMERO
		config.InitConfig(cfgFile) // Luego la config
	},
	// ... resto del código ...
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config-dir", "", "Configuration directory (default is ~/.config/google-photos-backup). Holds config.yaml and browser tokens.")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().Bool("non-interactive", false, "Disable interactive UI (progress bars)")
	rootCmd.PersistentFlags().String("log", "", "Path to global log file")
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	_ = viper.BindPFlag("non_interactive", rootCmd.PersistentFlags().Lookup("non-interactive"))
	_ = viper.BindPFlag("log", rootCmd.PersistentFlags().Lookup("log"))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
