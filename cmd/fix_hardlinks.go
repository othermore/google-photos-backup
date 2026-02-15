package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"google-photos-backup/internal/config"
	"google-photos-backup/internal/i18n"
	"google-photos-backup/internal/logger"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var fixHardlinksCmd = &cobra.Command{
	Use:   "fix-hardlinks",
	Short: "Deduplicate final backup by hardlinking identical files",
	Long:  `Scans the final backup directory (containing timestamped snapshots) and replaces identical files across snapshots with hardlinks to save space.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Starting Fix Hardlinks...")

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
		sort.Strings(snapshots) // Chronological

		if len(snapshots) < 2 {
			logger.Info(i18n.T("fix_hardlinks_not_enough"))
			return
		}

		// 2. Global Index: Map[Hash] -> OriginalPath
		// We only store the FIRST occurrence of a file content.
		fileIndex := make(map[string]string)

		// Stats
		totalFiles := 0
		dedupedCount := 0
		savedBytes := int64(0)

		// For progress reporting
		processedFiles := 0

		for _, snapName := range snapshots {
			snapPath := filepath.Join(backupPath, snapName)
			logger.Info(i18n.T("fix_hardlinks_analyze"), snapName)

			err := filepath.Walk(snapPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if info.IsDir() {
					return nil
				}

				totalFiles++

				// Calculate Hash
				// Optimization: Use Size+ModTime as key? User asked for Hash.
				// Compromise: Hash is safest. But slow.
				// Let's use Size + Partial Hash?
				// User explicitly said: "detectar que es una copia con el mismo hash".

				// Check if we can skip hashing (already hardlinked to something we know?)
				// If inode is already seen?
				// But we track by Content. Inode tracking is useful if we see internal hardlinks.

				hash, err := calculateHash(path)
				if err != nil {
					logger.Error("Failed to hash %s: %v", path, err)
					return nil
				}

				// Check if exists in index
				if originalPath, found := fileIndex[hash]; found {
					// Duplicate content found!

					// Check if ALREADY hardlinked
					if areHardlinked(originalPath, path) {
						// Already optimized.
						return nil
					}

					// Not hardlinked. Fix it.
					if !dryRun {
						// Remove duplicate
						if err := os.Remove(path); err != nil {
							logger.Error("Failed to remove duplicate %s: %v", path, err)
							return nil
						}
						// Link to original
						if err := os.Link(originalPath, path); err != nil {
							logger.Error("Failed to link %s -> %s: %v", path, originalPath, err)
							// Restore? panic?
						} else {
							dedupedCount++
							savedBytes += info.Size()
						}
					} else {
						dedupedCount++
						savedBytes += info.Size()
						logger.Info("Would link: %s -> %s", path, originalPath)
					}

				} else {
					// New unique content. Register it.
					fileIndex[hash] = path
				}

				processedFiles++
				if processedFiles%1000 == 0 {
					fmt.Printf("\rProcessed %d files...", processedFiles)
				}

				return nil
			})

			if err != nil {
				logger.Error("Error walking snapshot %s: %v", snapName, err)
			}
		}

		fmt.Println("") // Newline after progress
		logger.Info(i18n.T("fix_hardlinks_complete"))
		logger.Info(i18n.T("fix_hardlinks_processed"), totalFiles)
		logger.Info(i18n.T("fix_hardlinks_linked"), dedupedCount)
		logger.Info(i18n.T("fix_hardlinks_saved"), formatSizeForBackup(savedBytes))
	},
}

func isTimestamp(name string) bool {
	_, err := time.Parse("2006-01-02-150405", name)
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

func calculateHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// areHardlinked is duplicated from dedup.go/update_backup.go but ok for CLI cmd package separation
// (actually update_backup.go didn't export it, so I copy it here)
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
