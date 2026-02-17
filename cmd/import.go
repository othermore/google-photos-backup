package cmd

import (
	"google-photos-backup/internal/config"
	"google-photos-backup/internal/engine"
	"google-photos-backup/internal/i18n"
	"google-photos-backup/internal/logger"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import [directory]",
	Short: "Import and process manually downloaded zip files",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		importDir := args[0]

		// Validate Source
		info, err := os.Stat(importDir)
		if err != nil || !info.IsDir() {
			logger.Error(i18n.T("import_invalid_dir"), importDir)
			return
		}

		// Validate Output (Backup Path)
		if config.AppConfig.BackupPath == "" {
			logger.Error(i18n.T("backup_dir_error") + " (Check config)")
			return
		}

		logger.Info(i18n.T("import_start"), importDir)

		// Initialize Engine
		// Use working path for temp extraction
		eng := engine.New(config.AppConfig.WorkingPath, config.AppConfig.BackupPath)

		// 1. Find Zips
		files, err := os.ReadDir(importDir)
		if err != nil {
			logger.Error(i18n.T("import_read_fail"), err)
			return
		}

		var zips []string
		for _, f := range files {
			if !f.IsDir() && (strings.HasSuffix(strings.ToLower(f.Name()), ".zip") || strings.HasSuffix(strings.ToLower(f.Name()), ".tgz")) {
				zips = append(zips, filepath.Join(importDir, f.Name()))
			}
		}

		if len(zips) == 0 {
			logger.Warn(i18n.T("import_no_zips"), importDir)
			return
		}

		logger.Info(i18n.T("import_found_count"), len(zips))

		// 2. Process Zips Loop (Sequential)
		for i, zipPath := range zips {
			logger.Info(i18n.T("import_prog"), i+1, len(zips), filepath.Base(zipPath))

			// Copy to temp working dir first?
			// Or process directly from Source?
			// Engine deletes zip after process. We should COPY if we want to be safe,
			// OR we process directly but warn user.
			// "Delete Zip" is part of the space saving pipeline.
			// Let's COPY to temp first to preserve original manual files (User expects Import not to delete source usually?)
			// Users Guide says: "Import: Manual processing of user provided zip files".
			// BUT the pipeline is "Download -> Unzip -> Delete".
			// For Manual Import, maybe we should just Unzip and NOT delete the source zip?
			// Engine.ProcessZip deletes it.

			// Let's Copy to WorkingDir/temp_import first.
			tempZip := filepath.Join(config.AppConfig.WorkingPath, "temp_import", filepath.Base(zipPath))
			os.MkdirAll(filepath.Dir(tempZip), 0755)

			logger.Info(i18n.T("import_copying"))
			if err := copyFileLocal(zipPath, tempZip); err != nil {
				logger.Error(i18n.T("import_copy_fail"), err)
				continue
			}

			if err := eng.ProcessZip(tempZip); err != nil {
				logger.Error(i18n.T("import_process_fail"), err)
			}
		}

		// 3. Finalize
		if err := eng.Finalize(); err != nil {
			logger.Error(i18n.T("import_final_fail"), err)
		} else {
			logger.Info(i18n.T("import_done"))
		}
	},
}

func copyFileLocal(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func init() {
	rootCmd.AddCommand(importCmd)
}
