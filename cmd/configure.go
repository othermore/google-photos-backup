package cmd

import (
	"fmt"
	"google-photos-backup/internal/browser"
	"google-photos-backup/internal/config"
	"google-photos-backup/internal/i18n" // Importar paquete i18n
	"os"
	"path/filepath"
	"strings"

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
		confirm := prompt(i18n.T("login_ask"), "")
		ans := strings.ToLower(confirm)
		if ans == "s" || ans == "y" || ans == "yes" || ans == "si" {
			loginFlow(absPath)
		}
	},
}

func loginFlow(backupPath string) {
	fmt.Println(i18n.T("login_start"))
	fmt.Println(i18n.T("browser_open"))

	// Usamos el directorio de backup para guardar la sesión del navegador (carpeta 'browser_data')
	userDataDir := filepath.Join(backupPath, "browser_data")

	// Headless = false para que el usuario pueda ver y escribir
	bm := browser.New(userDataDir, false)
	bm.ManualLogin()

	// Verificación Headless inmediata
	fmt.Println("\nValidando credenciales guardadas...")
	bmHeadless := browser.New(userDataDir, true)
	defer bmHeadless.Close()

	if bmHeadless.VerifySession() {
		fmt.Println("\n✅ Sesión guardada y verificada correctamente. Las próximas ejecuciones usarán estas cookies.")
	} else {
		fmt.Println("\n⚠️  No se pudo verificar la sesión. Es posible que el login no se completara o Google pida 2FA de nuevo.")
	}
}
