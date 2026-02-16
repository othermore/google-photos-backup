package cmd

import (
	"os"
	"path/filepath"
	"sort"

	"google-photos-backup/internal/config"
	"google-photos-backup/internal/i18n"
	"google-photos-backup/internal/logger"
	"google-photos-backup/internal/processor"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rebuildIndexCmd = &cobra.Command{
	Use:   "rebuild-index",
	Short: "Rebuild index.json for all snapshots",
	Long:  `Scans all timestamped snapshots in the backup directory and generates/updates their index.json file. It uses Inode optimization to speed up re-indexing.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("🏗️  Starting Index Rebuild...")

		backupPath := config.AppConfig.BackupPath
		if backupPath == "" {
			backupPath = viper.GetString("backup_path")
		}
		backupPath = expandPath(backupPath)

		if backupPath == "" {
			logger.Error(i18n.T("update_backup_no_config"))
			return
		}

		// 1. List Snapshots
		entries, err := os.ReadDir(backupPath)
		if err != nil {
			logger.Error("Error reading backup dir: %v", err)
			return
		}

		var snapshots []string
		for _, e := range entries {
			if e.IsDir() && isTimestamp(e.Name()) {
				snapshots = append(snapshots, e.Name())
			}
		}
		sort.Strings(snapshots)

		if len(snapshots) == 0 {
			logger.Info("⚠️  No snapshots found.")
			return
		}

		logger.Info("found %d snapshots to process", len(snapshots))

		successCount := 0
		for _, snapName := range snapshots {
			snapPath := filepath.Join(backupPath, snapName)
			// logger.Info("Indexing snapshot: %s", snapName)

			if _, err := processor.EnsureSnapshotIndex(snapPath); err != nil {
				logger.Error("Failed to index %s: %v", snapName, err)
			} else {
				successCount++
			}
		}

		logger.Info("✅ Index Rebuild Complete. Processed %d/%d snapshots.", successCount, len(snapshots))
	},
}

func init() {
	rootCmd.AddCommand(rebuildIndexCmd)
}
