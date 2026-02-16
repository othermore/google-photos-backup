package cmd

import (
	"os"
	"path/filepath"
	"sort"

	"google-photos-backup/internal/config"
	"google-photos-backup/internal/i18n"
	"google-photos-backup/internal/logger"
	"google-photos-backup/internal/processor"
	"google-photos-backup/internal/registry"

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
		if immichPath == "" {
			immichPath = "immich-master"
		}
		masterRoot := filepath.Join(backupPath, immichPath)

		logger.Info("📂 Backup Path: %s", backupPath)
		logger.Info("📸 Immich Master Path: %s", masterRoot)

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

		// 2. Load Master Index
		masterIndexPath := filepath.Join(masterRoot, "index.json")
		masterIndex, err := registry.LoadIndex(masterIndexPath)
		if err != nil {
			logger.Info("⚠️  Could not load master index (will start fresh): %v", err)
			masterIndex = registry.NewIndex()
		}
		masterHashMap := processor.GetMasterHashMap(masterIndex)
		logger.Info("Loaded Master Index: %d files known.", len(masterHashMap))

		// 3. Process Snapshots
		totalFiles := 0
		processedFiles := 0

		for _, snapName := range snapshots {
			snapPath := filepath.Join(backupPath, snapName)
			logger.Info("Processing snapshot: %s", snapName)

			// A. Ensure Index Exists
			// We can call EnsureSnapshotIndex directly. It will reuse existing index if valid.
			idx, err := processor.EnsureSnapshotIndex(snapPath)
			if err != nil {
				logger.Error("Failed to ensure index for %s: %v", snapName, err)
				continue
			}

			// B. Link to Master
			if err := processor.LinkSnapshotToMaster(snapPath, idx, masterRoot, masterIndex, masterHashMap); err != nil {
				logger.Error("Failed to link snapshot %s to master: %v", snapName, err)
			}

			totalFiles += len(idx.Files)
			processedFiles++
		}

		// 4. Save Master Index
		if err := masterIndex.Save(masterIndexPath); err != nil {
			logger.Error("Failed to save Master Index: %v", err)
		} else {
			logger.Info("✅ Master Index Saved (%d entries).", len(masterIndex.Files))
		}

		logger.Info("✅ Rebuild Complete.")
		logger.Info("Total Snapshots Processed: %d", processedFiles)
	},
}

func init() {
	rootCmd.AddCommand(rebuildImmichCmd)
}
