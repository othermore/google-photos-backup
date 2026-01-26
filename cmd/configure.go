package cmd

import (
	"fmt"
	"google-photos-backup/internal/config"
	"google-photos-backup/internal/i18n" // Importar paquete i18n
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure credentials and directories",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("========================================")
		fmt.Println(i18n.T("header_title"))
		fmt.Println("========================================")
		fmt.Println(i18n.T("intro_1"))
		fmt.Println(i18n.T("intro_2"))
		fmt.Println("")
		fmt.Println(i18n.T("steps_title"))
		fmt.Println(i18n.T("step_1"))
		fmt.Println(i18n.T("step_2"))
		fmt.Println(i18n.T("step_3"))
		fmt.Println(i18n.T("step_4"))
		fmt.Println(i18n.T("step_5"))
		fmt.Println("")
		fmt.Println(i18n.T("readme_hint"))
		fmt.Println("========================================")
		fmt.Println("")

		// 1. Backup Dir
		backupPath := prompt(i18n.T("prompt_backup_dir"), config.AppConfig.BackupPath)
		absPath, _ := filepath.Abs(backupPath)

		// 2. Guardar
		viper.Set("backup_path", absPath)

		if viper.ConfigFileUsed() == "" {
			home, _ := os.UserHomeDir()
			configDir := filepath.Join(home, ".config", "google-photos-backup")
			if err := os.MkdirAll(configDir, 0755); err != nil {
				fmt.Printf(i18n.T("error_mkdir")+"\n", err)
				return
			}
			newConfigPath := filepath.Join(configDir, "config.yaml")
			viper.SetConfigFile(newConfigPath)
		}

		if err := viper.WriteConfig(); err != nil {
			if err := viper.WriteConfigAs(viper.ConfigFileUsed()); err != nil {
				fmt.Printf(i18n.T("error_save")+"\n", err)
				return
			}
		}

		fmt.Printf(i18n.T("success_msg")+"\n", viper.ConfigFileUsed())

		// 6. Login Ask
		fmt.Println("")
		fmt.Println("TODO: Implementar inicio de sesión con navegador (Go-Rod)")
		fmt.Println("En la próxima fase implementaremos la apertura del navegador para que inicies sesión.")
	},
}
