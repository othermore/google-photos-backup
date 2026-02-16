package processor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"google-photos-backup/internal/logger"
	"google-photos-backup/internal/registry"
)

// EnsureSnapshotIndex scans a snapshot directory, generates a file index with hashes,
// and saves it to index.json. It optimizes by reusing hashes from an existing index
// if the Inode and ModTime match.
func EnsureSnapshotIndex(snapshotPath string) (*registry.Index, error) {
	indexPath := filepath.Join(snapshotPath, "index.json")

	// 1. Load existing index for optimization
	existingIndex, err := registry.LoadIndex(indexPath)
	if err != nil {
		logger.Error("Failed to load existing index (will rebuild): %v", err)
		existingIndex = registry.NewIndex()
	}

	newIndex := registry.NewIndex()
	totalFiles := 0
	rehashedFiles := 0

	err = filepath.Walk(snapshotPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		// Skip index.json itself
		if filepath.Base(path) == "index.json" {
			return nil
		}
		// Skip system files
		if info.Name() == ".DS_Store" || filepath.Ext(info.Name()) == ".jsonl" {
			return nil
		}

		totalFiles++
		relPath, _ := filepath.Rel(snapshotPath, path)

		// Get Inode
		stat, ok := info.Sys().(*syscall.Stat_t)
		var inode uint64
		if ok {
			inode = stat.Ino
		}

		hash := ""
		// Inode Optimization check
		if existingEntry, ok := existingIndex.Get(relPath); ok {
			// Check if Inode matches (and ModTime/Size for safety)
			if existingEntry.Inode == inode &&
				existingEntry.ModTime.Equal(info.ModTime()) &&
				existingEntry.Size == info.Size() {
				hash = existingEntry.Hash
			}
		}

		if hash == "" {
			// Calculate Hash
			h, err := calculateHash(path)
			if err != nil {
				logger.Error("Failed to hash %s: %v", path, err)
				return nil
			}
			hash = h
			rehashedFiles++
		}

		newIndex.AddOrUpdate(registry.FileIndexEntry{
			RelPath: relPath,
			Hash:    hash,
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Inode:   inode,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	logger.Info("Index generated for %s: %d files (%d re-hashed)", filepath.Base(snapshotPath), totalFiles, rehashedFiles)

	// Save Index
	if err := newIndex.Save(indexPath); err != nil {
		return nil, fmt.Errorf("failed to save index: %w", err)
	}

	return newIndex, nil
}

// LinkSnapshotToMaster integrates a snapshot into the master directory.
// masterHashMap: Map[Hash] -> RelPath (in master)
func LinkSnapshotToMaster(snapshotPath string, snapshotIndex *registry.Index, masterRoot string, masterIndex *registry.Index, masterHashMap map[string]string) error {

	for relPath, entry := range snapshotIndex.Files {
		// 1. Check Deduplication
		if _, exists := masterHashMap[entry.Hash]; exists {
			// Already in Master
			continue
		}

		// 2. Not in Master: Link it
		srcPath := filepath.Join(snapshotPath, relPath)

		// Destination: YYYY/MM/Filename
		year := entry.ModTime.Format("2006")
		month := entry.ModTime.Format("01")
		filename := filepath.Base(relPath)

		destDir := filepath.Join(masterRoot, year, month)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}

		destRelPath := filepath.Join(year, month, filename)
		destFullPath := filepath.Join(masterRoot, destRelPath)

		// Collision Handling (Same filename, different hash)
		// Since we checked Hash above, we know this is NEW content.
		// If file exists, it's a collision.
		counter := 1
		ext := filepath.Ext(filename)
		name := filename[:len(filename)-len(ext)]

		for {
			if _, err := os.Stat(destFullPath); os.IsNotExist(err) {
				break
			}
			// Collision! Rename.
			newFilename := fmt.Sprintf("%s_%d%s", name, counter, ext)
			destRelPath = filepath.Join(year, month, newFilename)
			destFullPath = filepath.Join(masterRoot, destRelPath)
			counter++
		}

		// Create Hardlink
		if err := os.Link(srcPath, destFullPath); err != nil {
			logger.Error("Failed to link to master %s: %v", destRelPath, err)
			continue
		}

		// Update Master Index & Hash Map
		// Get Inode of the new link
		var inode uint64
		if info, err := os.Stat(destFullPath); err == nil {
			if stat, ok := info.Sys().(*syscall.Stat_t); ok {
				inode = stat.Ino
			}
		}

		newEntry := registry.FileIndexEntry{
			RelPath: destRelPath,
			Hash:    entry.Hash,
			Size:    entry.Size,
			ModTime: entry.ModTime,
			Inode:   inode,
		}
		masterIndex.AddOrUpdate(newEntry)
		masterHashMap[entry.Hash] = destRelPath
	}
	return nil
}

// Helpers

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

func GetMasterHashMap(idx *registry.Index) map[string]string {
	m := make(map[string]string)
	for path, entry := range idx.Files {
		m[entry.Hash] = path
	}
	return m
}
