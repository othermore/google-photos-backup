package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"google-photos-backup/internal/config"
	"google-photos-backup/internal/i18n"
	"google-photos-backup/internal/logger"
	"google-photos-backup/internal/registry"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var fixHardlinksCmd = &cobra.Command{
	Use:   "fix-hardlinks",
	Short: "Deduplicate final backup using index.json",
	Long:  `Scans the final backup directory (containing timestamped snapshots) and utilizes the existing index.json files to identify and hardlink duplicate files across snapshots instantly.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Starting Fix Hardlinks (Index-Based)...")

		backupPath := config.AppConfig.BackupPath
		if backupPath == "" {
			backupPath = viper.GetString("backup_path")
		}

		if backupPath == "" {
			logger.Error(i18n.T("update_backup_no_config"))
			return
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		logger.Info(i18n.T("fix_hardlinks_scan"), backupPath)

		if dryRun {
			logger.Info(i18n.T("fix_hardlinks_dry"))
		}

		// 1. List Snapshots (Oldest First)
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
		sort.Strings(snapshots) // Chronological order is crucial

		if len(snapshots) < 1 {
			logger.Info(i18n.T("fix_hardlinks_not_enough"))
			return
		}

		// 2. Global Index: Map[Hash] -> OriginalPath
		// We track the FIRST occurrence of a file content.
		globalIndex := make(map[string]string)

		// Stats
		totalFiles := 0
		dedupedCount := 0
		savedBytes := int64(0)
		snapshotsProcessed := 0

		for _, snapName := range snapshots {
			snapPath := filepath.Join(backupPath, snapName)
			indexPath := filepath.Join(snapPath, "index.json")

			// Check if index exists
			if _, err := os.Stat(indexPath); os.IsNotExist(err) {
				logger.Info("⚠️  Snapshot %s has no index.json. Skipping. Run 'rebuild-index' first.", snapName)
				continue
			}

			logger.Info("🔍 Analyzing snapshot: %s", snapName)

			// Load Index
			idx, err := registry.LoadIndex(indexPath)
			if err != nil {
				logger.Error("Failed to load index for %s: %v", snapName, err)
				continue
			}

			snapTotal := 0
			snapDedup := 0

			// Iterate files in index
			for relPath, fileData := range idx.Files {
				fullPath := filepath.Join(snapPath, relPath)
				snapTotal++
				totalFiles++

				// Check if Hash is already known
				if originalPath, found := globalIndex[fileData.Hash]; found {
					// Duplicate content candidate!

					// 1. Check if it's the SAME file (e.g. self-reference or same path)
					if originalPath == fullPath {
						continue
					}

					// 2. Check if already hardlinked (Inode check)
					// We need to stat the current file to check Inode
					if areHardlinked(originalPath, fullPath) {
						// Already optimized.
						continue
					}

					// 3. Not hardlinked. Fix it.
					if !dryRun {
						// Remove duplicate
						if err := os.Remove(fullPath); err != nil {
							// If file doesn't exist (index desync?), skip
							if !os.IsNotExist(err) {
								logger.Error("Failed to remove duplicate %s: %v", fullPath, err)
							}
							continue
						}
						// Link to original
						if err := os.Link(originalPath, fullPath); err != nil {
							logger.Error("Failed to link %s -> %s: %v", fullPath, originalPath, err)
						} else {
							snapDedup++
							dedupedCount++
							savedBytes += fileData.Size

							// Update the index? Content is same, Inode changed.
							// Ideally we should update the index with the new Inode to avoid re-hashing later.
							// But that requires saving the index. For now, let's just do the filesystem op.
							// Next 'rebuild-index' will pick up the new Inode.
						}
					} else {
						snapDedup++
						dedupedCount++
						savedBytes += fileData.Size
						// logger.Info("Would link: %s -> %s", fullPath, originalPath)
					}

				} else {
					// New unique content. Register it as the "Source of Truth".
					// We verify it exists before registering, just in case index is stale.
					if _, err := os.Stat(fullPath); err == nil {
						globalIndex[fileData.Hash] = fullPath
					}
				}
			}

			// logger.Info("   Snapshot stats: %d files, %d deduplicated", snapTotal, snapDedup)
			snapshotsProcessed++
		}

		logger.Info(i18n.T("fix_hardlinks_complete"))
		logger.Info("   Snapshots Scanned: %d", snapshotsProcessed)
		logger.Info(i18n.T("fix_hardlinks_processed"), totalFiles)
		logger.Info(i18n.T("fix_hardlinks_linked"), dedupedCount)
		logger.Info(i18n.T("fix_hardlinks_saved"), formatSizeForBackup(savedBytes))
	},
}

func isTimestamp(name string) bool {
	const format = "2006-01-02-150405"
	if len(name) < len(format) {
		return false
	}
	prefix := name[:len(format)]
	_, err := time.Parse(format, prefix)
	return err == nil
}

func formatSizeForBackup(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func init() {
	rootCmd.AddCommand(fixHardlinksCmd)
	fixHardlinksCmd.Flags().Bool("dry-run", false, "Simulate deduplication")
}

// areHardlinked checks if two files share the same inode and device
func areHardlinked(p1, p2 string) bool {
	fi1, err := os.Stat(p1)
	if err != nil {
		return false
	}
	fi2, err := os.Stat(p2)
	if err != nil {
		return false
	}

	stat1, ok1 := fi1.Sys().(*syscall.Stat_t)
	stat2, ok2 := fi2.Sys().(*syscall.Stat_t)
	if !ok1 || !ok2 {
		return false
	}

	return stat1.Ino == stat2.Ino && stat1.Dev == stat2.Dev
}
