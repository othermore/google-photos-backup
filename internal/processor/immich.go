package processor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"google-photos-backup/internal/logger"
)

// LinkToImmichMaster links a file to the Immich Master directory structure:
// backupRoot/immichParamPath/YYYY/MM/filename.ext
func LinkToImmichMaster(srcPath, backupRoot, immichParamPath string, fileDate time.Time) error {
	if immichParamPath == "" {
		immichParamPath = "immich-master"
	}

	// 1. Determine Destination Directory: YYYY/MM
	year := fileDate.Format("2006")
	month := fileDate.Format("01")
	destDir := filepath.Join(backupRoot, immichParamPath, year, month)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create immich master dir %s: %w", destDir, err)
	}

	// 2. Determine Destination Filename
	filename := filepath.Base(srcPath)
	destPath := filepath.Join(destDir, filename)

	// 3. Check for collisions
	if info, err := os.Stat(destPath); err == nil {
		// File exists!
		// Is it the same file (Hardlinked)?
		if areHardlinked(srcPath, destPath) {
			// Already linked, nothing to do.
			return nil
		}

		// Is it identical content?
		// Check size first
		srcInfo, _ := os.Stat(srcPath)
		if info.Size() == srcInfo.Size() {
			// Check Hash
			srcHash, _ := calculateHash(srcPath)
			destHash, _ := calculateHash(destPath)
			if srcHash != "" && srcHash == destHash {
				// Content is identical but not linked.
				// We should ideally replace dest with hardlink to src to save space?
				// Yes, let's fix it.
				if err := os.Remove(destPath); err != nil {
					return fmt.Errorf("failed to remove duplicate for linking: %w", err)
				}
				if err := os.Link(srcPath, destPath); err != nil {
					return fmt.Errorf("failed to link identical content: %w", err)
				}
				logger.Info("🔗 Relinked existing identical file in Immich Master: %s", filename)
				return nil
			}
		}

		// Content is DIFFERENT (collision).
		// We must verify uniqueness.
		// Strategy: Append counter suffix _1, _2 until unique.
		ext := filepath.Ext(filename)
		name := filename[:len(filename)-len(ext)]
		counter := 1
		for {
			newFilename := fmt.Sprintf("%s_%d%s", name, counter, ext)
			destPath = filepath.Join(destDir, newFilename)
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				break
			}
			// Check if THIS new candidate is the same file? (Unlikely but possible if re-running)
			if areHardlinked(srcPath, destPath) {
				return nil
			}
			counter++
		}
	}

	// 4. Create Hardlink
	if err := os.Link(srcPath, destPath); err != nil {
		return fmt.Errorf("failed to link to immich master: %w", err)
	}
	// logger.Info("📸 Linked to Immich Master: %s/%s/%s", year, month, filepath.Base(destPath))
	return nil
}

// Helpers (Duplicated from other packages to keep processor independent if needed,
// or should use shared utils? processor package is fine provided imports allow).

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
