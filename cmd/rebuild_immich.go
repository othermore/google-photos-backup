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

var rebuildImmichCmd = &cobra.Command{
	Use:   "rebuild-immich-master",
	Short: "Rebuild the Immich Master directory from existing snapshots",
	Long:  `Scans all timestamped snapshots in the backup directory and links every file to the Immich Master directory (YYYY/MM structure). This is useful to populate the master directory for the first time or after changing configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("🏗️  Starting Immich Master Rebuild...")

		backupPath := config.AppConfig.BackupPath
		if backupPath == "" {
			backupPath = viper.GetString("backup_path")
		}
		backupPath = expandPath(backupPath)

		if backupPath == "" {
			logger.Error(i18n.T("update_backup_no_config"))
			return
		}

		immichPath := config.AppConfig.ImmichMasterPath
		if immichPath == "" {
			immichPath = viper.GetString("immich_master_path")
		}

		logger.Info("📂 Backup Path: %s", backupPath)
		logger.Info("📸 Immich Master Path: %s", filepath.Join(backupPath, immichPath))

		// 1. List Snapshots (Oldest to Newest)
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
		sort.Strings(snapshots) // Chronological order ensures consistent behavior for duplicates

		if len(snapshots) == 0 {
			logger.Info("⚠️  No snapshots found.")
			return
		}

		totalFiles := 0
		processedFiles := 0
		linkedCount := 0

		for _, snapName := range snapshots {
			snapPath := filepath.Join(backupPath, snapName)
			logger.Info("Scanning snapshot: %s", snapName)

			err := filepath.Walk(snapPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if info.IsDir() {
					return nil
				}
				// Skip system files
				if info.Name() == ".DS_Store" {
					return nil
				}
				// Skip log files
				if filepath.Ext(info.Name()) == ".jsonl" {
					return nil
				}

				totalFiles++

				// Link to Master
				// We use ModTime as the date source.
				// For more accuracy, we could try to read Exif, but that requires external libs or slow parsing.
				// Since we set ModTime during backup, it should be reasonably correct (Creation Date).
				if err := processor.LinkToImmichMaster(path, backupPath, immichPath, info.ModTime()); err == nil {
					linkedCount++
				} else {
					// logger.Error("Failed to link %s: %v", info.Name(), err)
				}
				processedFiles++
				return nil
			})
			if err != nil {
				logger.Error("Error walking snapshot %s: %v", snapName, err)
			}
		}

		logger.Info("✅ Rebuild Complete.")
		logger.Info("Total Files Scanned: %d", totalFiles)
		logger.Info("Files Linked/Verified in Master: %d", linkedCount)
	},
}

func init() {
	rootCmd.AddCommand(rebuildImmichCmd)
}
