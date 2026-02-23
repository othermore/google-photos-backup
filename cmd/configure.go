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
	"time"

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
		workingPrompt := fmt.Sprintf(i18n.T("prompt_working_dir"), config.AppConfig.WorkingPath)
		workingPath := prompt(workingPrompt, config.AppConfig.WorkingPath)
		absWorkingPath, _ := filepath.Abs(workingPath)

		// 2. Rclone Remote (For 'drive' command)
		rcloneRemote := config.AppConfig.RcloneRemote
		if rcloneRemote == "" {
			rcloneRemote = "drive:"
		}
		// Always ask for rclone remote as it is needed for 'drive' command
		rclonePrompt := fmt.Sprintf(i18n.T("prompt_rclone_remote"), rcloneRemote)
		rcloneRemote = prompt(rclonePrompt, rcloneRemote)

		// 2.5 Email Alert To (New)
		currentEmail := config.AppConfig.EmailAlertTo
		emailPrompt := fmt.Sprintf(i18n.T("prompt_email_alert"), currentEmail)

		emailAlertTo := prompt(emailPrompt, currentEmail)

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
		viper.Set("working_path", absWorkingPath)
		viper.Set("email_alert_to", emailAlertTo)
		viper.Set("rclone_remote", rcloneRemote)
		viper.Set("fix_ambiguous_metadata", fixMode)
		viper.Set("backup_path", absBackupPath)

		// Ensure the directory for the config exists
		var configDir string
		if cfgFile != "" {
			configDir = cfgFile
		} else {
			configDir = filepath.Dir(viper.ConfigFileUsed())
			if configDir == "" || configDir == "." {
				home, _ := os.UserHomeDir()
				configDir = filepath.Join(home, ".config", "google-photos-backup")
			}
		}

		viper.SetConfigFile(filepath.Join(configDir, "config.yaml"))

		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Printf(i18n.T("error_mkdir")+"\n", err)
			return
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

	fmt.Print(i18n.T("ssh_headless_tips"))

	// Usamos el directorio de backup para guardar la sesión del navegador (carpeta 'browser_data')
	userDataDir := filepath.Join(workingPath, "browser_data")

	// Headless = false para que el usuario pueda ver y escribir
	bm := browser.New(userDataDir, false)
	bm.ManualLogin()
	bm.Close()                  // Close specifically to release the User Data Dir lock
	time.Sleep(2 * time.Second) // Wait for process to exit and lock to be released

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

func init() {
	toolCmd.AddCommand(configureCmd)
}
