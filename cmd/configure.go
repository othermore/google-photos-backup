package cmd

import (
	"bufio"
	"fmt"
	"google-photos-backup/internal/config"
	"google-photos-backup/internal/i18n" // Importar paquete i18n
	"google-photos-backup/internal/utils"
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
		reader := bufio.NewReader(os.Stdin)

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

		// 1. Client ID
		fmt.Print(i18n.T("prompt_client_id"))
		clientID, _ := reader.ReadString('\n')
		clientID = strings.TrimSpace(clientID)

		// 2. Client Secret
		fmt.Print(i18n.T("prompt_client_secret"))
		clientSecret, _ := reader.ReadString('\n')
		clientSecret = strings.TrimSpace(clientSecret)

		// 3. Backup Dir
		fmt.Println("")
		fmt.Printf(i18n.T("prompt_backup_dir"), config.AppConfig.BackupPath)
		backupPath, _ := reader.ReadString('\n')
		backupPath = strings.TrimSpace(backupPath)
		if backupPath == "" {
			backupPath = config.AppConfig.BackupPath
		}
		
		absPath, _ := filepath.Abs(backupPath)

		// 4. Guardar
		viper.Set("client_id", clientID)
		viper.Set("client_secret", clientSecret)
		viper.Set("backup_path", absPath)
		viper.Set("index_path", filepath.Join(absPath, "index.jsonl"))

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
		fmt.Print(i18n.T("login_ask"))
		confirm, _ := reader.ReadString('\n')
		// Aceptamos 's' (spanish) o 'y' (english)
		ans := strings.TrimSpace(strings.ToLower(confirm))
		if ans == "s" || ans == "y" {
			loginFlow()
		}
	},
}

func loginFlow() {
	fmt.Println(i18n.T("login_start"))
	fmt.Println(i18n.T("browser_open"))
	utils.OpenBrowser("https://google.com")
}