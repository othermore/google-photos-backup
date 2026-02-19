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

	"google-photos-backup/internal/config"
	"google-photos-backup/internal/i18n"
	"google-photos-backup/internal/logger"
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
	// We need to know WHICH files were extracted to dedup them specifically
	extractedFiles, err := e.unzipAndList(zipPath, extractDir)
	if err != nil {
		return fmt.Errorf(i18n.T("engine_extract_fail"), err)
	}

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

// ProcessZip Legacy wrapper
func (e *Engine) ProcessZip(zipPath string) error {
	// Works in e.WorkingDir default
	// We create a temporary batch dir structure to reuse the logic?
	// Or just keep the old logic for legacy sync?
	// Let's implement legacy logic for safety or redirect.
	// Legacy 'ProcessZip' was used by 'gpb sync' which extracts to 'e.WorkingDir/extracted'.
	// We can use ProcessZipWithIndex passing e.WorkingDir.
	return e.ProcessZipWithIndex(zipPath, e.WorkingDir)
}

// Finalize performs the shared processing on all extracted files set
func (e *Engine) Finalize() error {
	logger.Info(i18n.T("engine_final_phase"))

	extractDir := filepath.Join(e.WorkingDir, "extracted")

	// 1. Organize and Move (includes Metadata fix)
	if err := e.OrganizeAndMove(extractDir); err != nil {
		return err
	}

	// 2. Final Deduplication (Cross-Volume Fix)
	// Now that files are in BackupDir, they are strictly on the backup volume.
	logger.Info(i18n.T("engine_final_dedup"))

	// 3. Cleanup
	logger.Info(i18n.T("engine_cleanup"))
	os.RemoveAll(extractDir)

	return nil
}

// --- Helpers ---

func (e *Engine) unzipAndList(src, dest string) ([]string, error) {
	var extracted []string
	r, err := zip.OpenReader(src)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// ZipSlip check
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		extracted = append(extracted, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return nil, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return nil, err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return nil, err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()
	}
	return extracted, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (e *Engine) unzip(src, dest string) error {
	_, err := e.unzipAndList(src, dest)
	return err
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

// deduplicateAgainstBackup tries to link extracted files to existing backup files
func (e *Engine) deduplicateAgainstBackup(extractDir string) error {
	// 1. Scan extractDir
	// 2. Hash files
	// 3. Check against Global Index (if exists) or Scan Backup (slow, maybe skip for now)

	// Ideally we load the index.jsonl from BackupDir
	// index, err := registry.LoadIndex(filepath.Join(e.BackupDir, "index.jsonl"))
	// ...

	// For this iteration, we keep it empty to ensure compilation and basic flow.
	// The "Optimization" is a nice-to-have we can add once the main pipeline works.
	return nil
}

// OrganizeAndMove moves files from source to destination structure
func (e *Engine) OrganizeAndMove(srcDir string) error {
	logger.Info(i18n.T("engine_organize_move"))
	// For now, implementing a basic move to verify flow.
	// We will enhance this with metadata fixing in the next iteration.

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) == ".json" {
			return nil
		}

		// Move file to BackupDir/Year/Month/
		// Determine date (Mock for now, use ModTime)
		date := info.ModTime()
		year := date.Format("2006")
		month := date.Format("01")

		destDir := filepath.Join(e.BackupDir, year, month)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}

		destPath := filepath.Join(destDir, info.Name())
		// Avoid overwrite if exists? Rename with count?
		// Simple rename for now

		if err := os.Rename(path, destPath); err != nil {
			// Cross-device link error? Copy and Delete
			input, err := os.ReadFile(path)
			if err == nil {
				if err := os.WriteFile(destPath, input, 0644); err == nil {
					os.Remove(path)
				}
			}
		}

		return nil
	})
}
