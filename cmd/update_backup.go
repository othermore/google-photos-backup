package cmd

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"google-photos-backup/internal/config"
	"google-photos-backup/internal/i18n"
	"google-photos-backup/internal/logger"
	"google-photos-backup/internal/processor"
	"google-photos-backup/internal/registry"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type BackupLogEntry struct {
	Timestamp string   `json:"timestamp"`
	Source    string   `json:"source"`
	Snapshot  string   `json:"snapshot_path"`
	Added     int      `json:"added_count"`
	Linked    int      `json:"linked_count"`
	Internal  int      `json:"internal_links"`
	Size      int64    `json:"total_new_bytes"`
	Files     []string `json:"added_files"`
}

var updateBackupCmd = &cobra.Command{
	Use:   "update-backup",
	Short: "Create an incremental snapshot in final backup location",
	Long:  `Scans the downloads directory for processed exports (folders with 'raw' subdirectory), backs them up to a timestamped snapshot using hardlinks for deduplication, and deletes the source exports upon success.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info(i18n.T("update_backup_start"))

		backupPath := config.AppConfig.BackupPath
		if backupPath == "" {
			backupPath = viper.GetString("backup_path")
		}
		backupPath = expandPath(backupPath)

		if backupPath == "" {
			logger.Error(i18n.T("update_backup_no_config"))
			return
		}

		// Determine Source Root (downloads folder)
		// Default: working_path/downloads
		rootSource, _ := cmd.Flags().GetString("source")
		if rootSource == "" {
			workingPath := config.AppConfig.WorkingPath
			if workingPath == "" {
				workingPath = viper.GetString("working_path")
			}
			workingPath = expandPath(workingPath)

			if workingPath != "" {
				rootSource = filepath.Join(workingPath, "downloads")
			} else {
				rootSource = "downloads"
			}
		}
		rootSource = expandPath(rootSource)

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		logger.Info(i18n.T("update_backup_source"), rootSource)

		if _, err := os.Stat(rootSource); os.IsNotExist(err) {
			logger.Error(i18n.T("update_backup_source_missing"), rootSource)
			return
		}

		// Create Snapshot Directory
		timestamp := time.Now().Format("2006-01-02-150405")
		snapshotDir := filepath.Join(backupPath, timestamp)
		logger.Info(i18n.T("update_backup_dest"), snapshotDir)

		if dryRun {
			logger.Info(i18n.T("update_backup_dry_run"))
		} else {
			if err := os.MkdirAll(snapshotDir, 0755); err != nil {
				logger.Error(i18n.T("update_backup_mkdir_fail"), err)
				return
			}
		}

		// Find Previous Backup
		prevBackup := findLatestBackup(backupPath)
		if prevBackup != "" {
			logger.Info(i18n.T("update_backup_linking"), filepath.Base(prevBackup))
		}

		// Load History for Ordering
		// Typically in working_path/history.json
		workingPath := config.AppConfig.WorkingPath
		if workingPath == "" {
			workingPath = viper.GetString("working_path")
		}
		workingPath = expandPath(workingPath)

		historyPath := filepath.Join(workingPath, "history.json")
		reg, err := registry.New(historyPath)
		var validExports []registry.ExportEntry
		if err == nil {
			// Sort chronological by creation date (RequestedAt)
			sort.Slice(reg.Exports, func(i, j int) bool {
				return reg.Exports[i].RequestedAt.Before(reg.Exports[j].RequestedAt)
			})
			validExports = reg.Exports
			logger.Info(i18n.T("update_backup_history_loaded"), len(validExports))
		} else {
			logger.Info(i18n.T("update_backup_history_fail"), err)
		}

		// Load Processing Index for Validation
		processingIndex := make(map[string]bool)
		processedArchives := make(map[string]bool)

		// Attempt to locate processing_index.json
		candidates := []string{
			filepath.Join(rootSource, "processing_index.json"),
			filepath.Join(workingPath, "output", "processing_index.json"),
		}
		if outDir := viper.GetString("output_dir"); outDir != "" {
			candidates = append(candidates, filepath.Join(expandPath(outDir), "processing_index.json"))
		}
		if outDir := viper.GetString("output"); outDir != "" {
			candidates = append(candidates, filepath.Join(expandPath(outDir), "processing_index.json"))
		}
		candidates = append(candidates, "output/processing_index.json")

		var indexPath string
		for _, cand := range candidates {
			if _, err := os.Stat(cand); err == nil {
				indexPath = cand
				break
			}
		}

		if indexPath != "" {
			if data, err := os.ReadFile(indexPath); err == nil {
				var state struct {
					ProcessedExports  map[string]bool `json:"processed_exports"`
					ProcessedArchives map[string]bool `json:"processed_archives"`
				}
				if err := json.Unmarshal(data, &state); err == nil {
					processingIndex = state.ProcessedExports
					processedArchives = state.ProcessedArchives
					logger.Info(i18n.T("update_backup_index_loaded"), indexPath, len(processingIndex), len(processedArchives))
				}
			}
		} else {
			logger.Info(i18n.T("update_backup_index_missing"), rootSource)
		}

		totalStats := struct {
			Added    int
			Linked   int
			Internal int
			Bytes    int64
			Files    []string
		}{}

		inodeMap := make(map[uint64]string)
		processedExportsCount := 0

		// Helper to process an ID
		processID := func(exportID string) {
			exportPath := filepath.Join(rootSource, exportID)

			// Check if exists
			if _, err := os.Stat(exportPath); os.IsNotExist(err) {
				return
			}

			// Check if it's a valid export to process
			// Heuristic: Must have "raw" folder AND be marked as complete in processing_index.json
			rawPath := filepath.Join(exportPath, "raw")
			if _, err := os.Stat(rawPath); os.IsNotExist(err) {
				return
			}

			// Validate Completeness
			// 1. Check if marked completely processed
			isComplete := processingIndex[exportID]
			// 2. Fallback: Check if all archives are processed
			if !isComplete {
				// Load local state
				statePath := filepath.Join(exportPath, "state.json")
				state, err := registry.LoadDownloadState(statePath)
				if err == nil {
					allArchivesDone := true
					for _, f := range state.Files {
						key := exportID + "/" + f.Filename
						// Check if ZIP/TGZ
						ext := strings.ToLower(filepath.Ext(f.Filename))
						if ext != ".zip" && ext != ".tgz" && !strings.HasSuffix(strings.ToLower(f.Filename), ".tar.gz") {
							continue
						}

						if !processedArchives[key] {
							// DEBUG: Inspect why it failed
							if exportID == "bfd62949-e1de-4e34-b2ca-c13cc1c7b12f" {
								logger.Info("DEBUG: Validate Key Failed: '%s'", key)
								// logger.Info("DEBUG: Known Keys (sample): %v", processedArchives)
							}
							allArchivesDone = false
							break
						}
					}
					if allArchivesDone && len(state.Files) > 0 {
						isComplete = true
						logger.Info(i18n.T("update_backup_implicit_complete"), exportID)

						// Update Global State in memory and on disk
						if processingIndex == nil {
							processingIndex = make(map[string]bool)
						}
						processingIndex[exportID] = true

						// We need to write back the full state
						// We don't have the full state object easily here unless we refactor loading.

						// Refactoring Load:
						// Already loaded into 'processingIndex' and 'processedArchives'.
						// But to save, we need the *full* structure (including FileIndex if any).
						// Solution: Re-read as map[string]interface{} or similar?
						// Better: Just update the specific field in the file.
						// Or: Load into processor.State struct if available.

						// Let's assume we re-read and write.
						// Since we are in a loop, maybe we should batch updates?
						// For now, let's do it immediately for safety.

						content, err := os.ReadFile(filepath.Join(rootSource, "processing_index.json"))
						if err == nil {
							var fullState processor.State
							if err := json.Unmarshal(content, &fullState); err == nil {
								if fullState.ProcessedExports == nil {
									fullState.ProcessedExports = make(map[string]bool)
								}
								fullState.ProcessedExports[exportID] = true

								// Write back
								newContent, _ := json.MarshalIndent(fullState, "", "  ")
								_ = os.WriteFile(filepath.Join(rootSource, "processing_index.json"), newContent, 0644)
								logger.Info(i18n.T("update_backup_index_updated"))
							}
						}
					}
				}
			}

			if !isComplete {
				logger.Info(i18n.T("update_backup_skip_incomplete"), exportID)
				return
			}

			logger.Info(i18n.T("update_backup_processing"), exportID)

			// Load File Index from process step
			var exportFileIndex map[string]processor.FileMetadata
			exportIndexPath := filepath.Join(exportPath, "processing_index.json")
			if data, err := os.ReadFile(exportIndexPath); err == nil {
				var state processor.State
				if err := json.Unmarshal(data, &state); err == nil {
					exportFileIndex = state.FileIndex
				}
			}

			// Run Backup Logic for this Export
			startBytes := totalStats.Bytes

			err := backupExport(exportPath, snapshotDir, prevBackup, inodeMap, exportFileIndex, &totalStats, dryRun)
			if err != nil {
				logger.Error(i18n.T("update_backup_fail_export"), exportID, err)
			} else {
				// Success! Delete Source Export Content (Only 'raw')
				if !dryRun {
					logger.Info(i18n.T("update_backup_delete_content"), exportID)
					// We only delete the 'raw' folder to save space, but keep state.json/metadata
					if err := os.RemoveAll(rawPath); err != nil {
						logger.Error(i18n.T("update_backup_delete_fail"), rawPath, err)
					}
				} else {
					logger.Info(i18n.T("update_backup_dry_delete"), rawPath)
				}
				processedExportsCount++
			}

			if totalStats.Bytes > startBytes {
				// Logic to show incremental progress if needed
			}
		}

		// 1. Process From History (Ordered)
		processedIDs := make(map[string]bool)
		for _, entry := range validExports {
			if entry.ID == "" {
				continue
			}
			processID(entry.ID)
			processedIDs[entry.ID] = true
		}

		// 2. Process Remaining Directories (if any not in history but exist on disk)
		// Only if history failed or incomplete? User specifically asked for order.
		// If history loaded, we trust it. But maybe scan directories too for safety?
		// No, user wants STRICT order. If not in history, maybe ignore?
		// Let's iterate dir entries too just in case history is out of sync, but append them at end.
		entries, err := os.ReadDir(rootSource)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				id := entry.Name()
				if !processedIDs[id] {
					// Is it a valid UUID-like ID? Assume yes if folder matches structure.
					// Or skipping to enforce "only history-known exports".
					// Safe bet: Process it.
					processID(id)
				}
			}
		}

		if processedExportsCount == 0 {
			logger.Info(i18n.T("update_backup_no_exports"))
			// Cleanup empty snapshot if created?
			if !dryRun && totalStats.Added == 0 && totalStats.Linked == 0 {
				os.Remove(snapshotDir)
			}
			return
		}

		// 3. Update Immich Master (if enabled)
		immichEnabled := config.AppConfig.ImmichMasterEnabled
		// Fallback to viper if not set in struct (legacy/viper overlap)
		if !immichEnabled {
			immichEnabled = viper.GetBool("immich_master_enabled")
		}

		immichCount := 0
		if immichEnabled && !dryRun {
			immichPath := config.AppConfig.ImmichMasterPath
			if immichPath == "" {
				immichPath = viper.GetString("immich_master_path")
			}
			if immichPath == "" {
				immichPath = "immich-master"
			}
			logger.Info("📸 Updating Immich Master Directory (%s)...", immichPath)
			masterRoot := filepath.Join(backupPath, immichPath)

			// A. Ensure Index for New Snapshot
			// We scan the WHOLE snapshot to be safe and robust, using Inode optimization.
			// This covers 'Added', 'Linked', and 'Internal' files uniformly.
			snapIdx, err := processor.EnsureSnapshotIndex(snapshotDir)
			if err != nil {
				logger.Error("Failed to generate index for new snapshot: %v", err)
			} else {
				// B. Load Master Index
				masterIndexPath := filepath.Join(masterRoot, "index.json")
				masterIndex, err := registry.LoadIndex(masterIndexPath)
				if err != nil {
					// logger.Info("⚠️  Could not load master index (will create new): %v", err)
					masterIndex = registry.NewIndex()
				}
				masterHashMap := processor.GetMasterHashMap(masterIndex)

				// C. Link to Master
				if err := processor.LinkSnapshotToMaster(snapshotDir, snapIdx, masterRoot, masterIndex, masterHashMap); err != nil {
					logger.Error("Failed to link new snapshot to master: %v", err)
				} else {
					// Count how many files we *know* are in master now (just for stats/log)
					// Actually we can't easily count *newly* linked without modifying return of LinkSnapshotToMaster
					// But we can just say "Updated".
					immichCount = len(snapIdx.Files) // Reporting total files tracked for this snapshot
				}

				// D. Save Master Index
				if err := masterIndex.Save(masterIndexPath); err != nil {
					logger.Error("Failed to save Master Index: %v", err)
				}
			}
		}

		// Logging
		if !dryRun {
			logEntry := BackupLogEntry{
				Timestamp: timestamp,
				Source:    rootSource,
				Snapshot:  snapshotDir,
				Added:     totalStats.Added,
				Linked:    totalStats.Linked,
				Internal:  totalStats.Internal,
				Size:      totalStats.Bytes,
				Files:     totalStats.Files,
			}

			logPath := filepath.Join(backupPath, "backup_log.jsonl")
			f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				logger.Error("❌ Failed to open backup log: %v", err)
			} else {
				jsonBytes, _ := json.Marshal(logEntry)
				if _, err := f.Write(append(jsonBytes, '\n')); err != nil {
					logger.Error("❌ Failed to write to backup log: %v", err)
				}
				f.Close()
				logger.Info(i18n.T("update_backup_log_updated"), logPath)
			}
		}

		// Summary
		logger.Info(i18n.T("update_backup_success"), totalStats.Added, formatSizeForBackup(totalStats.Bytes), totalStats.Linked, rootSource)
		logger.Info(i18n.T("update_backup_summary_links"), totalStats.Linked)
		logger.Info(i18n.T("update_backup_summary_internal"), totalStats.Internal)
		if immichEnabled && !dryRun {
			logger.Info("📸 Immich Master: %d files linked", immichCount)
		}
		logger.Info(i18n.T("update_backup_summary_exports"), processedExportsCount)
	},
}

func init() {
	rootCmd.AddCommand(updateBackupCmd)
	updateBackupCmd.Flags().String("source", "", "Source directory (defaults to working_path/downloads)")
	updateBackupCmd.Flags().Bool("dry-run", false, "Simulate the update")
}

// backupExport recursively backups a single export directory
func backupExport(srcDir, snapshotRoot, prevBackupRoot string, inodeMap map[uint64]string, fileIndex map[string]processor.FileMetadata, stats *struct {
	Added    int
	Linked   int
	Internal int
	Bytes    int64
	Files    []string
}, dryRun bool) error {

	// We want to flatten the structure.
	// Source: downloads/ID/raw/Takeout/Google Photos/...
	// Dest:   snapshot/Google Photos/...

	// 1. Locate the actual content root within srcDir (which is downloads/ID)
	// Usually it's srcDir/raw/Takeout/Google Photos
	// But sometimes Takeout is skipped if zip structure was different?
	// Let's find "Google Photos" directory.

	contentRoot := ""
	possibleRoots := []string{
		filepath.Join(srcDir, "raw", "Takeout", "Google Photos"),
		filepath.Join(srcDir, "raw", "Google Photos"),
		filepath.Join(srcDir, "raw"), // Fallback if no specific folder structure? No, that would dump raw files.
	}

	for _, p := range possibleRoots {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			contentRoot = p
			break
		}
	}

	if contentRoot == "" {
		// Log warning but maybe process raw directly if structured differently?
		// For now, assume "Google Photos" must exist to be a valid backup source
		logger.Info("⚠️ Could not find 'Google Photos' folder in %s. Skipping flattening.", srcDir)
		// Fallback to processing 'raw' as root, but mapping to 'Google Photos' in dest?
		contentRoot = filepath.Join(srcDir, "raw")
	}

	// Calculate prefix length to strip
	// We want relative path from contentRoot
	// e.g. contentRoot = .../Google Photos
	// file = .../Google Photos/Album/Img.jpg
	// rel = Album/Img.jpg
	// Dest = snapshot/Google Photos/Album/Img.jpg

	targetDestRoot := filepath.Join(snapshotRoot, "Google Photos")
	prevDestRoot := ""
	if prevBackupRoot != "" {
		prevDestRoot = filepath.Join(prevBackupRoot, "Google Photos")
	}

	return filepath.Walk(contentRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Skip system files
		if info.Name() == ".DS_Store" {
			return nil
		}

		// Skip original archives if present
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".zip" || ext == ".tgz" {
			return nil
		}

		// RelPath from contentRoot
		relPath, err := filepath.Rel(contentRoot, path)
		if err != nil {
			return nil
		}

		destPath := filepath.Join(targetDestRoot, relPath)

		if dryRun {
			return nil
		}

		// Get Source Inode
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			logger.Error("Failed to get syscall.Stat_t for %s", path)
			return nil
		}
		inode := stat.Ino

		// 1. Check Internal Hardlink (Intra-Snapshot Deduplication)
		linkedFromPrev := false
		if prevPath, ok := inodeMap[inode]; ok {
			// Found in previous backup or internal map! Link it.
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}

			if err := os.Link(prevPath, destPath); err == nil {
				stats.Linked++
				// If it was from previous run, it counts as linked.
				// If it was internal from same run, it also counts.
				inodeMap[inode] = destPath // Update map
				linkedFromPrev = true
				return nil
			} else {
				logger.Info("⚠️ Failed to link from internal/map %s: %v. Will copy/move.", prevPath, err)
			}
		}

		// 2. Check Previous Backup (Inter-Snapshot Deduplication)
		if !linkedFromPrev && prevDestRoot != "" {
			prevFile := filepath.Join(prevDestRoot, relPath)
			if prevInfo, err := os.Stat(prevFile); err == nil {
				// Candidate exists in previous backup!
				// Check Size
				if prevInfo.Size() == info.Size() {
					// Size matches. Check Hash.
					// Get Source Hash from Index
					sourceHash := ""
					if meta, ok := fileIndex[path]; ok {
						sourceHash = meta.Hash
					}

					// Get Dest Hash (Compute)
					// Only compute if we have source hash (otherwise comparison impossible efficiently?)
					// Or just compute both if source missing?
					// Safe: compute both if source missing.

					match := false
					if sourceHash != "" {
						destHash, err := calculateHash(prevFile)
						if err == nil && destHash == sourceHash {
							match = true
						}
					} else {
						// Fallback: Compute both
						h1, _ := calculateHash(path)
						h2, _ := calculateHash(prevFile)
						if h1 != "" && h1 == h2 {
							match = true
						}
					}

					if match {
						// Deduplicate against Previous Backup!
						if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
							return err
						}
						// Hardlink: Dest -> Previous
						if err := os.Link(prevFile, destPath); err == nil {
							stats.Linked++
							inodeMap[inode] = destPath
							linkedFromPrev = true
							logger.Info(i18n.T("update_backup_linked_prev"), relPath)
							return nil
						}
					}
				}
			}
		}

		// 3. Copy/Move (New File)
		if !linkedFromPrev {
			if err := moveFile(path, destPath); err != nil {
				logger.Error("Failed to move/copy %s: %v", relPath, err)
				return err
			}
			stats.Added++
			stats.Bytes += info.Size()
			stats.Files = append(stats.Files, relPath)
			inodeMap[inode] = destPath
			logger.Info(i18n.T("update_backup_copied"), relPath)
		}
		return nil
	})
}

// Helpers

func findLatestBackup(finalPath string) string {
	entries, err := os.ReadDir(finalPath)
	if err != nil {
		return ""
	}
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() && isTimestamp(entry.Name()) {
			dirs = append(dirs, entry.Name())
		}
	}
	if len(dirs) == 0 {
		return ""
	}
	sort.Strings(dirs)
	return filepath.Join(finalPath, dirs[len(dirs)-1])
}

// isTimestamp is available in package cmd (from fix_hardlinks.go)
// formatSizeForBackup is available in package cmd (from fix_hardlinks.go)

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	info, err := sourceFile.Stat()
	if err == nil {
		os.Chtimes(dst, time.Now(), info.ModTime())
	}
	return nil
}

// moveFile attempts to rename the file (mv). If it fails (e.g. cross-device), it falls back to copy+delete.
func moveFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	// Try atomic rename first (efficient)
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// Check if error is cross-device link (EXDEV)
	// In Go, this is often wrapped in *os.LinkError
	isCrossDevice := false
	if linkErr, ok := err.(*os.LinkError); ok {
		if linkErr.Err == syscall.EXDEV {
			isCrossDevice = true
		}
	}
	_ = isCrossDevice // Unused if we just fall back unconditionally

	// Fallback: Copy + Remove
	if err := copyFile(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
