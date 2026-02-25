package processor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"strings"

	"google-photos-backup/internal/config"
	"google-photos-backup/internal/i18n"
	"google-photos-backup/internal/logger"
	"google-photos-backup/internal/registry"
)

// EnsureSnapshotIndex scans a snapshot directory, generates a file index with hashes,
// and saves it to index.json. It optimizes by reusing hashes from an existing index
// if the Inode and ModTime match, OR from an optional cacheIndex if provided.
func EnsureSnapshotIndex(snapshotPath string, cacheIndex *registry.Index) (*registry.Index, error) {
	indexPath := filepath.Join(snapshotPath, "index.json")

	// 1. Load existing index for optimization
	existingIndex, err := registry.LoadIndex(indexPath)
	if err != nil {
		logger.Debug("No existing index at %s (will build new)", indexPath)
		existingIndex = registry.NewIndex()
	}

	newIndex := registry.NewIndex()
	totalFiles := 0
	rehashedFiles := 0

	// 2. Pre-build an Inode cache from existing indexes
	// This helps us avoid rehashing files that were recently hardlinked (changed Inode)
	// but the NEW Inode is already known in the existing index elsewhere.
	inodeMap := make(map[uint64]string)

	if existingIndex != nil {
		for _, entry := range existingIndex.Files {
			if entry.Inode != 0 && entry.Hash != "" {
				inodeMap[entry.Inode] = entry.Hash
			}
		}
	}
	if cacheIndex != nil {
		for _, entry := range cacheIndex.Files {
			if entry.Inode != 0 && entry.Hash != "" {
				inodeMap[entry.Inode] = entry.Hash
			}
		}
	}

	err = filepath.Walk(snapshotPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if config.IsIgnored(path) {
			if info.IsDir() {
				return filepath.SkipDir
			}
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

		// Caching Logic:
		// We DO NOT copy the Inode from the cache because the file has just been moved
		// and might have crossed a volume boundary (getting a new Inode).
		// We strictly read the fresh Inode from the filesystem above.
		// 1. Check external cacheIndex first (Trust it if Size matches)
		if cacheIndex != nil {
			if cacheEntry, ok := cacheIndex.Get(relPath); ok {
				if cacheEntry.Size == info.Size() {
					hash = cacheEntry.Hash
				}
			}
		}

		// 2. Check local existingIndex strictly (Requires Inode/ModTime/Size match)
		oldHash := ""
		if hash == "" {
			if existingEntry, ok := existingIndex.Get(relPath); ok {
				oldHash = existingEntry.Hash // Keep track of what this file used to be
				if existingEntry.Inode == inode &&
					existingEntry.ModTime.Equal(info.ModTime()) &&
					existingEntry.Size == info.Size() {
					hash = existingEntry.Hash
				}
			}
		}

		// 3. Fallback: Check if the Inode is already known in the index
		// Very strict optimization: if the new Inode maps to a known file, we ONLY reuse
		// the hash if it matches the 'oldHash' this file was known to have.
		// This guarantees it's a deduplication fix and not a file content substitution.
		if hash == "" && oldHash != "" {
			if knownHash, ok := inodeMap[inode]; ok && knownHash == oldHash {
				hash = knownHash
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
			ModTime: info.ModTime(), // Always trust the physical disk timestamp (post-JSON fix)
			Inode:   inode,
		})

		// Register new Inodes dynamically during the walk
		if inode != 0 && hash != "" {
			inodeMap[inode] = hash
		}

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

type yearMonthState struct {
	DirectCount int
	PartCounts  map[int]int
	MaxPart     int
}

// LinkSnapshotToMaster integrates a snapshot into the master directory.
// masterHashMap: Map[Hash] -> RelPath (in master)
func LinkSnapshotToMaster(snapshotPath string, snapshotIndex *registry.Index, masterRoot string, masterIndex *registry.Index, masterHashMap map[string]string) error {

	// 0. Pre-calculate directory states to manage the 500-file part limits
	ymStates := make(map[string]*yearMonthState)

	for _, entry := range masterIndex.Files {
		dir := filepath.ToSlash(filepath.Dir(entry.RelPath))
		parts := strings.Split(dir, "/")

		if len(parts) == 2 {
			ym := dir // "YYYY/MM"
			if ymStates[ym] == nil {
				ymStates[ym] = &yearMonthState{PartCounts: make(map[int]int)}
			}
			ymStates[ym].DirectCount++
		} else if len(parts) == 3 && strings.HasPrefix(parts[2], "Part_") {
			ym := parts[0] + "/" + parts[1]
			if ymStates[ym] == nil {
				ymStates[ym] = &yearMonthState{PartCounts: make(map[int]int)}
			}
			partNum, _ := strconv.Atoi(strings.TrimPrefix(parts[2], "Part_"))
			ymStates[ym].PartCounts[partNum]++
			if partNum > ymStates[ym].MaxPart {
				ymStates[ym].MaxPart = partNum
			}
		}
	}

	for relPath, entry := range snapshotIndex.Files {
		// 1. Check Deduplication
		if _, exists := masterHashMap[entry.Hash]; exists {
			continue // Already in Master
		}

		// Limit master linking exclusively to valid media types (no json, no garbage)
		if !config.IsValidMedia(relPath) || config.IsIgnored(relPath) || IsIgnoredFile(relPath) {
			continue
		}

		// 2. Not in Master: Link it
		srcPath := filepath.Join(snapshotPath, relPath)

		year := entry.ModTime.Format("2006")
		month := entry.ModTime.Format("01")
		ym := year + "/" + month

		state := ymStates[ym]
		if state == nil {
			state = &yearMonthState{PartCounts: make(map[int]int)}
			ymStates[ym] = state
		}

		// SPLIT LOGIC: If YYYY/MM has >= 500 files directly, move them to Part_001
		if state.MaxPart == 0 && state.DirectCount >= 500 {
			part1RelDir := filepath.Join(year, month, "Part_001")
			part1AbsDir := filepath.Join(masterRoot, part1RelDir)
			os.MkdirAll(part1AbsDir, 0755)

			var toMove []registry.FileIndexEntry
			for mRelPath, mEntry := range masterIndex.Files {
				if filepath.ToSlash(filepath.Dir(mRelPath)) == ym {
					toMove = append(toMove, mEntry)
				}
			}

			logger.Info(i18n.T("immich_split_part1"), ym, len(toMove))

			for _, mEntry := range toMove {
				oldAbs := filepath.Join(masterRoot, mEntry.RelPath)
				filename := filepath.Base(mEntry.RelPath)
				newRel := filepath.Join(part1RelDir, filename)
				newAbs := filepath.Join(masterRoot, newRel)

				if err := os.Rename(oldAbs, newAbs); err == nil {
					delete(masterIndex.Files, mEntry.RelPath)
					mEntry.RelPath = filepath.ToSlash(newRel)
					masterIndex.Files[mEntry.RelPath] = mEntry
					masterHashMap[mEntry.Hash] = mEntry.RelPath
				}
			}

			state.MaxPart = 1
			state.PartCounts[1] = state.DirectCount
			state.DirectCount = 0
		}

		// SPLIT LOGIC: If the current MaxPart has >= 500 files, create next Part
		if state.MaxPart > 0 && state.PartCounts[state.MaxPart] >= 500 {
			state.MaxPart++
			logger.Info(i18n.T("immich_split_next"), ym, state.MaxPart)
		}

		var destRelDir string
		if state.MaxPart > 0 {
			destRelDir = filepath.Join(year, month, fmt.Sprintf("Part_%03d", state.MaxPart))
			state.PartCounts[state.MaxPart]++
		} else {
			destRelDir = filepath.Join(year, month)
			state.DirectCount++
		}

		filename := filepath.Base(relPath)
		destDir := filepath.Join(masterRoot, destRelDir)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}

		destRelPath := filepath.Join(destRelDir, filename)
		destFullPath := filepath.Join(masterRoot, destRelPath)

		// Collision Handling & Linking
		// Optimistically attempt to link. If it fails due to existence, rename and retry.
		// This saves hundreds of thousands of os.Stat calls over the network.
		counter := 1
		ext := filepath.Ext(filename)
		name := filename[:len(filename)-len(ext)]
		linkSuccess := false

		for {
			err := os.Link(srcPath, destFullPath)
			if err == nil {
				linkSuccess = true
				break
			}
			if os.IsExist(err) {
				// Collision! Rename.
				newFilename := fmt.Sprintf("%s_%d%s", name, counter, ext)
				destRelPath = filepath.Join(destRelDir, newFilename)
				destFullPath = filepath.Join(masterRoot, destRelPath)
				counter++
			} else {
				logger.Error("Failed to link to master %s: %v", destRelPath, err)
				break
			}
		}

		if !linkSuccess {
			continue
		}

		// The Inode of a hardlink is IDENTICAL to the source. No need to os.Stat the NAS!
		newEntry := registry.FileIndexEntry{
			RelPath: filepath.ToSlash(destRelPath),
			Hash:    entry.Hash,
			Size:    entry.Size,
			ModTime: entry.ModTime,
			Inode:   entry.Inode, // Reuse known Inode instantly
		}
		masterIndex.AddOrUpdate(newEntry)
		masterHashMap[entry.Hash] = newEntry.RelPath
	}
	return nil
}

// Helpers

func calculateHash(filePath string) (hashStr string, err error) {
	file, openErr := os.Open(filePath)
	if openErr != nil {
		return "", openErr
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	hash := sha256.New()
	if _, copyErr := io.Copy(hash, file); copyErr != nil {
		return "", copyErr
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

// IsIgnoredFile checks if a file should be excluded from Immich Master
func IsIgnoredFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json", ".zip", ".tar", ".tgz", ".rar", ".ini", ".lnk", ".docx", ".txt", ".html":
		return true
	case ".crdownload", ".tmp":
		return true
	}
	// Also ignore system files starting with .
	if strings.HasPrefix(filepath.Base(filename), ".") {
		return true
	}
	return false
}
