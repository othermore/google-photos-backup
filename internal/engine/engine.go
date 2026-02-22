package engine

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"google-photos-backup/internal/config"
	"google-photos-backup/internal/i18n"
	"google-photos-backup/internal/logger"
	"google-photos-backup/internal/processor"
	"google-photos-backup/internal/registry"
)

// Engine handles the optimized processing pipeline
type Engine struct {
	WorkingDir string
	BackupDir  string
	AlbumsDir  string

	// Global Index (Hash -> Absolute Path) for Cross-Volume Dedup
	GlobalIndex map[string]string

	// Config
	FixAmbiguousMetadata string
}

func New(workingDir, backupDir string) *Engine {
	return &Engine{
		WorkingDir:           workingDir,
		BackupDir:            backupDir,
		AlbumsDir:            filepath.Join(backupDir, "albums"),
		FixAmbiguousMetadata: config.AppConfig.FixAmbiguousMetadata,
		GlobalIndex:          make(map[string]string),
	}
}

// LoadGlobalIndex scans the BackupDir for index.json files and builds an in-memory map
func (e *Engine) LoadGlobalIndex() error {
	logger.Info(i18n.T("drive_global_index_load"), e.BackupDir)
	count := 0

	// Walk BackupDir/YYYY/MM structure
	err := filepath.Walk(e.BackupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			return nil
		}
		if info.Name() == "index.json" {
			// Found an index file! Load it.
			idx, err := registry.LoadIndex(path)
			if err != nil {
				logger.Warn(i18n.T("drive_global_index_fail"), path, err)
				return nil
			}

			// Dir of this index (e.g. Backup/2015/10)
			dir := filepath.Dir(path)

			for _, entry := range idx.Files {
				// We map Hash -> Absolute Path
				// entry.RelPath is relative to the index location?
				// Usually index.json stores relative paths to the folder it's in.
				absPath := filepath.Join(dir, entry.RelPath)
				e.GlobalIndex[entry.Hash] = absPath
				count++
			}
		}
		return nil
	})

	logger.Info(i18n.T("drive_global_index_loaded"), count)
	return err
}

// ProcessZipWithIndex handles a single zip file with incremental deduplication
func (e *Engine) ProcessZipWithIndex(zipPath, batchDir string) error {
	logger.Info(i18n.T("engine_zip_process"), filepath.Base(zipPath))

	// 1. Unzip to batchDir/extracted
	extractDir := filepath.Join(batchDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf(i18n.T("engine_extract_dir_fail"), err)
	}

	logger.Info(i18n.T("engine_extracting"))
	extractedFiles, err := e.unzipAndList(zipPath, extractDir)
	if err != nil {
		logger.LogToFile("ERROR: Extraction failed for %s: %v", filepath.Base(zipPath), err)
		return fmt.Errorf(i18n.T("engine_extract_fail"), err)
	}
	logger.LogToFile("EXTRACT: Unzipped %d files from %s", len(extractedFiles), filepath.Base(zipPath))

	// 2. Incremental Deduplication (Local Batch Index)
	indexFile := filepath.Join(batchDir, "index.json")

	// Load using registry.LoadIndex (Standard Format)
	batchIndex, err := registry.LoadIndex(indexFile)
	if err != nil {
		logger.Warn(i18n.T("engine_batch_index_fail"), err)
		batchIndex = registry.NewIndex()
	} else {
		if len(batchIndex.Files) > 0 {
			logger.Info(i18n.T("engine_batch_index_loaded"), len(batchIndex.Files))
		}
	}

	logger.Info(i18n.T("engine_dedup_batch"))
	filesDedupedLocal := 0
	filesDedupedGlobal := 0

	for _, relPath := range extractedFiles {
		fullPath := filepath.Join(extractDir, relPath)

		info, err := os.Stat(fullPath)
		if err != nil || info.IsDir() {
			continue
		}

		// Hash
		hash, err := hashFile(fullPath)
		if err != nil {
			logger.Warn(i18n.T("engine_hash_fail"), relPath, err)
			continue
		}

		// OPTIMIZATION: Check Global Index (Backup) First
		// If we find it in Backup, we link to it immediately and skip local batch logic for this file.
		if backupPath, ok := e.GlobalIndex[hash]; ok {
			// Verify it exists
			if _, err := os.Stat(backupPath); err == nil {
				// We found it in backup!
				// Is it same volume?
				if e.isSameVolume(extractDir, filepath.Dir(backupPath)) {
					// Great! Hardlink to backup
					os.Remove(fullPath)
					if err := os.Link(backupPath, fullPath); err == nil {
						filesDedupedGlobal++
						// We can skip adding to batchIndex?
						// NO. We MUST add to batchIndex so subsequent files in this batch
						// that match this hash ALSO link to this (now linked) file.
						// The file at fullPath is now a link to backupPath.
						// So passing fullPath to others is fine.

						// Create Entry for Batch Index
						entry := registry.FileIndexEntry{
							RelPath: relPath,
							Hash:    hash,
							Size:    info.Size(),
							ModTime: info.ModTime(),
						}
						// Get Inode of the LINKED file
						if stat, ok := os.Stat(fullPath); ok == nil {
							if sys, ok := stat.Sys().(*syscall.Stat_t); ok {
								entry.Inode = sys.Ino
							}
						}
						batchIndex.AddOrUpdate(entry)

						continue // Done with this file
					} else {
						logger.Warn(i18n.T("engine_link_backup_fail"), backupPath, err)
					}
				}
			} else {
				// Stale index entry? Remove it?
				delete(e.GlobalIndex, hash)
			}
		}

		// Create Entry
		entry := registry.FileIndexEntry{
			RelPath: relPath,
			Hash:    hash,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			entry.Inode = stat.Ino
		}

		// Check Index (Standard Format)
		// We look up by Hash. existing index is by RelPath (Files map).
		// We need a reverse lookup or iterate?
		// registry.Index.Files is map[string]FileIndexEntry where Key is RelPath.
		// Constructing a Hash map for fast lookup:
		// TODO: Optimization - Maintain a separate Hash map in memory if too slow.
		// For now, let's iterate or assume we need to add a "GetByHash" to registry?
		// Actually, standard registry.Index is Key=RelPath.
		// BUT for deduplication we need Key=Hash.

		// Let's implement a quick local lookup map from the loaded index
		existingRel := ""
		for _, e := range batchIndex.Files {
			if e.Hash == hash {
				existingRel = e.RelPath
				break
			}
		}

		if existingRel != "" {
			// Collision found!
			if existingRel != relPath {
				existingPath := filepath.Join(extractDir, existingRel)

				// Ensure existing file still exists
				if _, err := os.Stat(existingPath); err == nil {
					// Link: RelPath -> ExistingPath
					os.Remove(fullPath)
					if err := os.Link(existingPath, fullPath); err == nil {
						filesDedupedLocal++
						// Update inode in entry to match the linked one?
						// Actually, we should probably store the NEW entry too, pointing to its path?
						// Or just rely on the fact it matches?
						// Use the existing entry's inode for the new file?
						if stat, ok := os.Stat(fullPath); ok == nil {
							if sys, ok := stat.Sys().(*syscall.Stat_t); ok {
								entry.Inode = sys.Ino
							}
						}
					} else {
						logger.Warn(i18n.T("engine_link_local_fail"), relPath, existingRel, err)
					}
				} else {
					// Original missing?
					// Just add this new one as the source of truth
				}
			}
		}

		// Always add/update the index with the current file info
		batchIndex.AddOrUpdate(entry)
	}
	logger.Info(i18n.T("engine_dedup_stats"), filesDedupedGlobal, filesDedupedLocal)
	logger.Info(i18n.T("engine_index_updated"), len(batchIndex.Files))
	logger.LogToFile("DEDUP: Archive %s deduplicated. Global hardlinks: %d. Local hardlinks: %d", filepath.Base(zipPath), filesDedupedGlobal, filesDedupedLocal)

	// Save Index (Standard Format)
	if err := batchIndex.Save(indexFile); err != nil {
		logger.Warn(i18n.T("engine_index_save_fail"), err)
	}

	// 4. Delete Zip (Space Saving)
	logger.Info(i18n.T("engine_zip_delete"))
	if err := os.Remove(zipPath); err != nil {
		logger.Warn(i18n.T("engine_zip_del_fail"), zipPath, err)
	}

	return nil
}

// Finalize performs the shared processing on all extracted files set
func (e *Engine) Finalize(snapshotName string) error {
	logger.Info(i18n.T("engine_final_phase"))

	if snapshotName == "" {
		snapshotName = time.Now().Format("2006-01-02-150405")
	}

	extractDir := filepath.Join(e.WorkingDir, "extracted")
	if _, err := os.Stat(extractDir); os.IsNotExist(err) {
		return nil // Nothing to finalize
	}

	// 1. Metadata Fix
	logger.Info("📅 Applying metadata fixes from JSON sidecars...")
	pm := processor.NewManager(extractDir, extractDir, e.AlbumsDir)
	pm.FixAmbiguousMetadata = e.FixAmbiguousMetadata
	if err := pm.ScanRaw(extractDir, false); err == nil {
		pm.CorrectMetadata()
	}

	// 2. Move to Snapshot directory
	snapshotDir := filepath.Join(e.BackupDir, snapshotName)
	logger.Info("📦 Moving processed files to snapshot: %s", snapshotDir)
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return err
	}

	err := filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || path == extractDir {
			return err
		}

		relPath, _ := filepath.Rel(extractDir, path)
		destPath := filepath.Join(snapshotDir, relPath)

		if info.IsDir() {
			os.MkdirAll(destPath, 0755)
			return nil
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		// Move file
		if err := os.Rename(path, destPath); err != nil {
			// Cross-device fallback
			input, err := os.ReadFile(path)
			if err == nil {
				if err := os.WriteFile(destPath, input, info.Mode()); err == nil {
					os.Chtimes(destPath, info.ModTime(), info.ModTime())
					os.Remove(path)
				}
			}
		}
		return nil
	})
	if err != nil {
		logger.Error("Failed to move files to snapshot: %v", err)
	}

	// Generate and Save Snapshot Index
	logger.Info("📝 Generating snapshot index...")

	// Load batch index to use as cache (prevents rehashing!)
	batchIndexPath := filepath.Join(e.WorkingDir, "index.json")
	batchIndex, _ := registry.LoadIndex(batchIndexPath) // If error, nil is fine

	snapIdx, err := processor.EnsureSnapshotIndex(snapshotDir, batchIndex)
	if err != nil {
		logger.Warn("Failed to generate snapshot index: %v", err)
	}

	// 3. Immich Master Integration
	immichEnabled := config.AppConfig.ImmichMasterEnabled
	if immichEnabled {
		immichPath := config.AppConfig.ImmichMasterPath
		if immichPath == "" {
			immichPath = "immich-master"
		}
		masterRoot := filepath.Join(e.BackupDir, immichPath)
		logger.Info("📸 Updating Immich Master Directory (%s)...", immichPath)

		masterIndexPath := filepath.Join(masterRoot, "index.json")
		masterIndex, err := registry.LoadIndex(masterIndexPath)
		if err != nil {
			masterIndex = registry.NewIndex()
		}
		masterHashMap := processor.GetMasterHashMap(masterIndex)

		if err := processor.LinkSnapshotToMaster(snapshotDir, snapIdx, masterRoot, masterIndex, masterHashMap); err != nil {
			logger.Error("Failed to link new snapshot to master: %v", err)
		} else {
			if err := masterIndex.Save(masterIndexPath); err != nil {
				logger.Error("Failed to save Master Index: %v", err)
			}
		}
	}

	// 4. Cleanup
	logger.Info(i18n.T("engine_cleanup"))
	os.RemoveAll(extractDir)

	return nil
}

// --- Helpers ---

func (e *Engine) unzipAndList(src, dest string) (extracted []string, err error) {
	r, openErr := zip.OpenReader(src)
	if openErr != nil {
		return nil, openErr
	}
	defer func() {
		if closeErr := r.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	for _, f := range r.File {
		// Strip leading "Takeout/" if present to flatten the structure
		cleanName := strings.TrimPrefix(f.Name, "Takeout/")

		// If it's just the "Takeout/" root folder, skip it
		if cleanName == "" {
			continue
		}

		fpath := filepath.Join(dest, cleanName)

		// ZipSlip check
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		extracted = append(extracted, cleanName)

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if errMk := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); errMk != nil {
			return nil, errMk
		}

		outFile, errCreate := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if errCreate != nil {
			return nil, errCreate
		}

		rc, errOpen := f.Open()
		if errOpen != nil {
			_ = outFile.Close()
			return nil, errOpen
		}

		_, copyErr := io.Copy(outFile, rc)

		if closeErr := outFile.Close(); closeErr != nil && copyErr == nil {
			copyErr = closeErr
		}
		_ = rc.Close()

		if copyErr != nil {
			return nil, copyErr
		}
	}
	return extracted, nil
}

func hashFile(path string) (hash string, err error) {
	f, openErr := os.Open(path)
	if openErr != nil {
		return "", openErr
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	h := sha256.New()
	if _, copyErr := io.Copy(h, f); copyErr != nil {
		return "", copyErr
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (e *Engine) isSameVolume(path1, path2 string) bool {
	stat1 := &syscall.Stat_t{}
	stat2 := &syscall.Stat_t{}

	// Create if not exists to check volume
	os.MkdirAll(path1, 0755)
	os.MkdirAll(path2, 0755)

	if err := syscall.Stat(path1, stat1); err != nil {
		return false
	}
	if err := syscall.Stat(path2, stat2); err != nil {
		return false
	}

	return stat1.Dev == stat2.Dev
}
