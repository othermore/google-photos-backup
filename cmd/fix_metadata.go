package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"google-photos-backup/internal/config"
	"google-photos-backup/internal/logger"
	"google-photos-backup/internal/processor"

	"github.com/spf13/cobra"
)

var fixMetadataCmd = &cobra.Command{
	Use:   "fix-metadata",
	Short: "Retroactively apply JSON metadata fixes to snapshots",
	Long:  `Scans the target directory or all snapshot directories in the main Backup location, matching media files with Google Takeout JSON sidecars to correct their filesystem modification dates. This strictly uses the same advanced heuristic engine as the import/drive process to find the correct JSON and only alters filesystem dates (not EXIF).`,
	Run: func(cmd *cobra.Command, args []string) {
		targetDir, _ := cmd.Flags().GetString("target-dir")

		// Create a clean instance of Manager using global ambiguous config
		pm := &processor.Manager{
			FixAmbiguousMetadata: config.AppConfig.FixAmbiguousMetadata,
			FileIndex:            make(map[string]processor.FileMetadata),
		}

		var dirsToProcess []string

		if targetDir != "" {
			dirsToProcess = append(dirsToProcess, targetDir)
		} else {
			logger.Info("🔍 Discovering snapshots in Backup Path: %s", config.AppConfig.BackupPath)
			entries, err := os.ReadDir(config.AppConfig.BackupPath)
			if err != nil {
				logger.Fatalf("Failed to read backup directory: %v", err)
			}
			for _, entry := range entries {
				// Avoid . and the master dir.
				if entry.IsDir() && entry.Name() != "immich-master" && !strings.HasPrefix(entry.Name(), ".") {
					dirsToProcess = append(dirsToProcess, filepath.Join(config.AppConfig.BackupPath, entry.Name()))
				}
			}
		}

		if len(dirsToProcess) == 0 {
			logger.Info("⚠️  No directories found to process.")
			return
		}

		for _, dir := range dirsToProcess {
			logger.Info("==================================================")
			logger.Info("📂 Target Directory: %s", dir)

			// Reset the FileIndex map for this run
			pm.FileIndex = make(map[string]processor.FileMetadata)

			// Walk to populate FileIndex purely based on extensions
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				absPath, _ := filepath.Abs(path)
				ext := strings.ToLower(filepath.Ext(path))
				isJson := ext == ".json" || strings.HasSuffix(ext, ".json")

				pm.FileIndex[absPath] = processor.FileMetadata{
					IsJSON: isJson,
				}
				return nil
			})

			if err != nil {
				logger.Error("❌ Failed to scan directory %s: %v", dir, err)
				continue
			}

			// Apply the metadata fix using the shared exact logic
			pm.CorrectMetadata()
		}

		logger.Info("✅ fix-metadata pass completed.")
	},
}

func init() {
	toolCmd.AddCommand(fixMetadataCmd)
	fixMetadataCmd.Flags().String("target-dir", "", "Target directory to scan (if empty, processes all snapshots in the Backup path)")
}
