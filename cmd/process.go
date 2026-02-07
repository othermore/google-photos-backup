package cmd

import (
	"path/filepath"

	"google-photos-backup/internal/logger"
	"google-photos-backup/internal/processor"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Extract, deduplicate, and organize downloaded archives",
	Long:  `Extracts ZIP/TGZ files from the download directory, corrects metadata using JSON sidecars, deduplicates files, and organizes them into albums.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Config is already loaded by rootCmd PersistentPreRun

		inputDir := viper.GetString("download_dir")
		outputDir := viper.GetString("output_dir")
		albumsDir := viper.GetString("albums_dir")

		// Overrides from flags
		if flagInput, _ := cmd.Flags().GetString("input"); flagInput != "" {
			inputDir = flagInput
		}
		if flagOutput, _ := cmd.Flags().GetString("output"); flagOutput != "" {
			outputDir = flagOutput
		}
		if flagAlbums, _ := cmd.Flags().GetString("albums"); flagAlbums != "" {
			albumsDir = flagAlbums
		}

		// Validate directories
		if inputDir == "" {
			// Try to infer from backup_path
			backupPath := viper.GetString("backup_path")
			if backupPath != "" {
				inputDir = filepath.Join(backupPath, "downloads")
			} else {
				inputDir = "downloads"
			}
		}
		if outputDir == "" {
			outputDir = "output"
		}
		if albumsDir == "" {
			albumsDir = filepath.Join(outputDir, "albums")
		}

		logger.Info("🚀 Starting Processing Phase")
		logger.Info("📂 Input: %s", inputDir)
		logger.Info("📂 Output: %s", outputDir)
		logger.Info("📂 Albums: %s", albumsDir)

		pm := processor.NewManager(inputDir, outputDir, albumsDir)
		pm.DeleteOrigin, _ = cmd.Flags().GetBool("delete-origin")
		pm.ForceMetadata, _ = cmd.Flags().GetBool("force-metadata")
		pm.ForceExtraction, _ = cmd.Flags().GetBool("force-extract")
		pm.ForceDedup, _ = cmd.Flags().GetBool("force-dedup")
		pm.TargetExport, _ = cmd.Flags().GetString("export")

		// Handle --fix-ambiguous-metadata
		// Priority: Flag > Config > Default
		flagAmbiguous, _ := cmd.Flags().GetString("fix-ambiguous-metadata")
		if flagAmbiguous != "" {
			pm.FixAmbiguousMetadata = flagAmbiguous
		} else {
			// Get from viper (config or default)
			pm.FixAmbiguousMetadata = viper.GetString("fix_ambiguous_metadata")
		}

		if err := pm.Run(); err != nil {
			logger.Error("❌ Processing failed: %v", err)
		} else {
			logger.Info("✅ Processing completed successfully.")
		}
	},
}

func init() {
	rootCmd.AddCommand(processCmd)

	processCmd.Flags().StringP("input", "i", "", "Directory containing downloaded ZIP/TGZ files")
	processCmd.Flags().StringP("output", "o", "", "Directory for extracted and processed files")
	processCmd.Flags().StringP("albums", "a", "", "Directory for album symlinks")
	processCmd.Flags().Bool("delete-origin", false, "Delete original ZIP/TGZ files after successful extraction (Saves space)")

	// Granular Force Flags
	processCmd.Flags().Bool("force-metadata", false, "Force metadata correction (dates) for already processed exports")
	processCmd.Flags().Bool("force-extract", false, "Force extraction for already processed exports")
	processCmd.Flags().Bool("force-dedup", false, "Force global deduplication check")
	processCmd.Flags().String("export", "", "Process only this specific Export ID")

	processCmd.Flags().String("fix-ambiguous-metadata", "", "Behavior for ambiguous metadata matches: yes, no, or interactive")
}
