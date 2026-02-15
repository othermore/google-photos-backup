package cmd

import (
	"fmt"
	"google-photos-backup/internal/browser"
	"google-photos-backup/internal/config"
	"google-photos-backup/internal/i18n" // Importar paquete i18n
	"google-photos-backup/internal/logger"
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

		// 1. Working Dir (Download/Process)
		workingPath := prompt(i18n.T("prompt_working_dir"), config.AppConfig.WorkingPath)
		absWorkingPath, _ := filepath.Abs(workingPath)

		// 2. Download Mode
		currentMode := config.AppConfig.DownloadMode
		if currentMode == "" {
			currentMode = config.ModeDirectDownload
		}
		modePrompt := fmt.Sprintf(i18n.T("prompt_download_mode"),
			config.ModeDirectDownload, config.ModeDriveDownload, currentMode)

		dlMode := prompt(modePrompt, currentMode)
		if dlMode != config.ModeDirectDownload && dlMode != config.ModeDriveDownload {
			logger.Info(i18n.T("invalid_mode"), config.ModeDirectDownload)
			dlMode = config.ModeDirectDownload
		}

		// 3. Fix Ambiguous Metadata
		currentFix := config.AppConfig.FixAmbiguousMetadata
		if currentFix == "" {
			currentFix = "interactive"
		}

		fixPrompt := fmt.Sprintf(i18n.T("prompt_fix_ambiguous"), currentFix)

		fixMode := prompt(fixPrompt, currentFix)
		validFixes := map[string]bool{"yes": true, "no": true, "interactive": true}
		if !validFixes[fixMode] {
			fixMode = "interactive"
		}

		// 4. Backup Path (Storage)
		currentBackupPath := config.AppConfig.BackupPath
		backupPathPrompt := i18n.T("prompt_backup_path")
		if currentBackupPath != "" {
			backupPathPrompt = fmt.Sprintf("%s [default: %s]", backupPathPrompt, currentBackupPath)
		}
		backupPath := prompt(backupPathPrompt, currentBackupPath)
		absBackupPath, _ := filepath.Abs(backupPath)

		// 5. Guardar
		viper.Set("working_path", absWorkingPath)
		viper.Set("download_mode", dlMode)
		viper.Set("fix_ambiguous_metadata", fixMode)
		viper.Set("backup_path", absBackupPath)

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
			loginFlow(absWorkingPath)
		}
	},
}

func loginFlow(workingPath string) {
	fmt.Println(i18n.T("login_start"))
	fmt.Println(i18n.T("browser_open"))

	// Usamos el directorio de backup para guardar la sesión del navegador (carpeta 'browser_data')
	userDataDir := filepath.Join(workingPath, "browser_data")

	// Headless = false para que el usuario pueda ver y escribir
	bm := browser.New(userDataDir, false)
	bm.ManualLogin()

	// Verificación Headless inmediata
	logger.Info(i18n.T("validating_creds"))
	bmHeadless := browser.New(userDataDir, true)
	defer bmHeadless.Close()

	if bmHeadless.VerifySession() {
		logger.Info(i18n.T("session_valid"))
	} else {
		logger.Error(i18n.T("session_invalid"))
	}
}
