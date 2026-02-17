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

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Configure recurring Takeout export to Google Drive",
	Long:  `Automatically configures Google Takeout to export Photos to Drive every 2 months for 1 year, split into 50GB files.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("========================================")
		fmt.Println(i18n.T("schedule_title"))
		fmt.Println("========================================")

		// 1. Verify Config
		if config.AppConfig.WorkingPath == "" {
			logger.Error(i18n.T("backup_dir_error"))
			return
		}

		userDataDir := filepath.Join(config.AppConfig.WorkingPath, "browser_data")
		if err := os.MkdirAll(userDataDir, 0755); err != nil {
			logger.Error("Failed to create user data dir: %v", err)
			return
		}

		// 2. Launch Browser (Graphical Mode for visibility)
		// We want the user to see what's happening or intervene if needed (2FA etc)
		// But ideally it should be automated if session is valid.
		logger.Info(i18n.T("starting_manager") + " (Gui Mode)")
		bm := browser.New(userDataDir, false)
		defer bm.Close()

		// 3. Check Session
		if !bm.VerifySession() {
			logger.Error(i18n.T("session_invalid"))
			logger.Info(i18n.T("schedule_login_info"))
			// Let user login manually now?
			bm.ManualLogin()
			if !bm.VerifySession() {
				logger.Error(i18n.T("schedule_login_fail"))
				return
			}
		}

		// 4. Perform Scheduling
		if err := bm.ScheduleRecurringTakeout(); err != nil {
			logger.Error(i18n.T("schedule_failed"), err)
			return
		}

		// Success message handled in browser.go, just print final user instructions
		logger.Info(i18n.T("schedule_complete_msg"))
		logger.Info(i18n.T("schedule_next_steps"))
	},
}

func init() {
	rootCmd.AddCommand(scheduleCmd)
}
