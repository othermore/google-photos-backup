package cmd

import (
	"fmt"
	"google-photos-backup/internal/browser"
	"google-photos-backup/internal/config"
	"google-photos-backup/internal/i18n"
	"google-photos-backup/internal/logger"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var directCmd = &cobra.Command{
	Use:   "direct",
	Short: "Direct Download Backup management",
	Long:  `Commands to schedule, download, and manage Takeout backups using direct email links.`,
}

var directScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Configure recurring Takeout export via Email",
	Long:  `Automatically configures Google Takeout to export Photos via Email every 2 months for 1 year, split into 50GB files.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("========================================")
		fmt.Println(i18n.T("schedule_title"))
		fmt.Println("========================================")

		if config.AppConfig.WorkingPath == "" {
			logger.Error(i18n.T("backup_dir_error"))
			return
		}

		userDataDir := filepath.Join(config.AppConfig.WorkingPath, "browser_data")
		if err := os.MkdirAll(userDataDir, 0755); err != nil {
			logger.Error("Failed to create user data dir: %v", err)
			return
		}

		logger.Info(i18n.T("starting_manager") + " (Gui Mode)")
		bm := browser.New(userDataDir, false)
		defer bm.Close()

		if !bm.VerifySession() {
			logger.Error(i18n.T("session_invalid"))
			logger.Info(i18n.T("schedule_login_info"))
			bm.ManualLogin()
			if !bm.VerifySession() {
				logger.Error(i18n.T("schedule_login_fail"))
				return
			}
		}

		if err := bm.RequestTakeout("email", "multiple"); err != nil {
			logger.Error(i18n.T("schedule_failed"), err)
			return
		}

		logger.Info(i18n.T("schedule_complete_msg"))
	},
}

var directScheduleOnceCmd = &cobra.Command{
	Use:   "schedule-once",
	Short: "Configure a single Takeout export via Email",
	Long:  `Automatically configures Google Takeout to export Photos via Email one time, split into 50GB files.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("========================================")
		fmt.Println(i18n.T("schedule_title"))
		fmt.Println("========================================")

		if config.AppConfig.WorkingPath == "" {
			logger.Error(i18n.T("backup_dir_error"))
			return
		}

		userDataDir := filepath.Join(config.AppConfig.WorkingPath, "browser_data")
		if err := os.MkdirAll(userDataDir, 0755); err != nil {
			logger.Error("Failed to create user data dir: %v", err)
			return
		}

		logger.Info(i18n.T("starting_manager") + " (Gui Mode)")
		bm := browser.New(userDataDir, false)
		defer bm.Close()

		if !bm.VerifySession() {
			logger.Error(i18n.T("session_invalid"))
			logger.Info(i18n.T("schedule_login_info"))
			bm.ManualLogin()
			if !bm.VerifySession() {
				logger.Error(i18n.T("schedule_login_fail"))
				return
			}
		}

		if err := bm.RequestTakeout("email", "once"); err != nil {
			logger.Error(i18n.T("schedule_failed"), err)
			return
		}

		logger.Info(i18n.T("schedule_complete_msg"))
	},
}

func init() {
	rootCmd.AddCommand(directCmd)
	directCmd.AddCommand(directScheduleCmd)
	directCmd.AddCommand(directScheduleOnceCmd)
}
