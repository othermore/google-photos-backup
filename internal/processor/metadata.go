package processor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google-photos-backup/internal/logger"
)

// PhotoMetadata represents the structure of Google Photos JSON sidecars
type PhotoMetadata struct {
	Title        string `json:"title"`
	CreationTime struct {
		Timestamp string `json:"timestamp"`
		Formatted string `json:"formatted"`
	} `json:"creationTime"`
	PhotoTakenTime struct {
		Timestamp string `json:"timestamp"`
		Formatted string `json:"formatted"`
	} `json:"photoTakenTime"`
}

var (
	// Matches -edited, -edit, -edi, -ed, -e, - (at end of name, before ext)
	reEdited = regexp.MustCompile(`[-_]e?d?i?t?e?d?$`)

	// Matches (1), (12)... at the end of the string
	reNumbering = regexp.MustCompile(`\(\d+\)$`)
)

// CorrectMetadata iterates over all media files and applies dates from JSON
func (m *Manager) CorrectMetadata() error {
	// 1. Build Index of JSONs
	// We store valid JSON paths in a map keyed by their "clean" name
	// but we also keep the raw filename for fuzzy matching.
	jsonFiles := make([]string, 0)

	for path, meta := range m.FileIndex {
		if meta.IsJSON {
			jsonFiles = append(jsonFiles, path)
		}
	}

	logger.Info("   Indexed %d JSON sidecars.", len(jsonFiles))

	updated := 0
	// 2. Match Media Files
	for mediaPath, meta := range m.FileIndex {
		if meta.IsJSON {
			continue
		}

		bestJSON := m.findBestJSON(mediaPath, jsonFiles)
		if bestJSON != "" {
			if err := m.applyDate(mediaPath, bestJSON); err == nil {
				updated++
			} else {
				logger.Debug("❌ Failed to apply date to %s: %v", filepath.Base(mediaPath), err)
			}
		} else {
			logger.Info("⚠️  No JSON found for: %s", filepath.Base(mediaPath))
		}
	}

	logger.Info("✅ Metadata corrected for %d files.", updated)
	return nil
}

// findBestJSON implements the heuristics to find the matching JSON file
func (m *Manager) findBestJSON(mediaPath string, jsonFiles []string) string {
	mediaName := filepath.Base(mediaPath)
	dir := filepath.Dir(mediaPath)

	// --- Level 1: Direct Exact Matches ---
	// 1. "file.jpg" -> "file.jpg.json"
	c1 := mediaPath + ".json"
	if _, ok := m.FileIndex[c1]; ok {
		return c1
	}
	// 2. "file.jpg" -> "file.jpg.supplemental-metadata.json"
	c2 := mediaPath + ".supplemental-metadata.json"
	if _, ok := m.FileIndex[c2]; ok {
		return c2
	}

	// --- Level 2: Clean Name Matches (Strip Extension & Suffixes) ---
	// Normalize the media name to find its "Root"
	// Example: "DSC00189-edited.JPG" -> "DSC00189"
	// Example: "_DSC0188(2).JPG" -> "_DSC0188"

	ext := filepath.Ext(mediaName)
	withoutExt := strings.TrimSuffix(mediaName, ext)

	cleanName := withoutExt

	// Strip "Numbering" suffixes: "file(1)" -> "file"
	// Note: We only strip (N) if it's at the very end
	if loc := reNumbering.FindStringIndex(cleanName); loc != nil {
		cleanName = cleanName[:loc[0]]
	}

	// Strip "Edited" suffixes: "file-edited" -> "file"
	if loc := reEdited.FindStringIndex(cleanName); loc != nil {
		cleanName = cleanName[:loc[0]]
	}

	// Candidate List based on CleanName
	// We verify presence using the Index
	candidates := []string{
		cleanName + ".json",       // "DSC00189.json"
		cleanName + ext + ".json", // "DSC00189.JPG.json" (Re-add ext to clean root)
		cleanName + ".jpg.json",   // "MVC...jpg.json" (for .mp4)
		cleanName + ".JPG.json",
		cleanName + ".jpeg.json",
		cleanName + ".heic.json",
		cleanName + ext + ".supplemental-metadata.json",
		cleanName + ".supplemental-metadata.json",
	}

	for _, candName := range candidates {
		fullPath := filepath.Join(dir, candName)
		if _, ok := m.FileIndex[fullPath]; ok {
			return fullPath
		}
	}

	// --- Level 3: Directory Scan (Fuzzy & Truncation) ---
	// Only iterate files in the same directory (optimization)

	// Optimization: Filter JSONs in the same directory if not cached
	// To avoid re-iterating m.FileIndex every time, we rely on the caller passing 'jsonFiles'
	// BUT jsonFiles contains ALL json files. We should filter by directory.

	dirJSONs := []string{}
	// Filter jsonFiles by Directory
	for _, jPath := range jsonFiles {
		if filepath.Dir(jPath) == dir {
			dirJSONs = append(dirJSONs, filepath.Base(jPath))
		}
	}

	for _, jsonName := range dirJSONs {
		// Prepare JSON stems
		jStem := jsonName
		jStem = strings.TrimSuffix(jStem, ".json")
		jStem = strings.TrimSuffix(jStem, ".supplemental-metadata")
		jStem = strings.TrimSuffix(jStem, ".metadata")
		// jStem could be "DSC00189.JPG" or "DSC00189"

		// 1. Prefix Match (Truncation)
		// Check if MediaName starts with JSON stem (Reverse of usual, because MediaName is long)
		// AND Check if CleanMediaName starts with JSON stem
		// OR Check if JSON stem starts with CleanMediaName (if JSON has extra junk?)

		// Case: "00100..._BURST..._CO.jpg" vs "00100..._BURST..._C.json"
		// jStem="00100..._C", Media="00100..._CO.jpg"
		// Media starts with jStem? Yes.
		if strings.HasPrefix(withoutExt, jStem) {
			return filepath.Join(dir, jsonName)
		}

		// Case: "PXL...ORIGINAL-edi.jpg" vs "PXL...ORIGINAL.jp.json"
		// CleanMedia="PXL...ORIGINAL"
		// jStem="PXL...ORIGINAL.jp"
		// Match? No. CleanMedia doesn't start with jStem.
		// Does jStem start with CleanMedia? Yes. "PXL...ORIGINAL.jp" starts with "PXL...ORIGINAL"
		if strings.HasPrefix(jStem, cleanName) {
			return filepath.Join(dir, jsonName)
		}

		// Case: "MVIMG...MP4" vs "MVIMG...jpg.json"
		// CleanMedia="MVIMG..."
		// jStem="MVIMG...jpg"
		// jStem starts with CleanMedia? Yes.

		// This should cover most cases.
	}

	return ""
}

func isMediaExt(ext string) bool {
	ext = strings.ToLower(ext)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".heic", ".webp", ".mp4", ".mov", ".gif", ".avi", ".3gp", ".mkv", ".m4v", ".wmv":
		return true
	case ".nef", ".cr2", ".orf", ".arw", ".dng", ".raf", ".rw2", ".srw", ".pef": // RAW formats
		return true
	}
	return false
}

func (m *Manager) applyDate(mediaPath, jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return err
	}

	var meta PhotoMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}

	// Prefer PhotoTakenTime, fall back to CreationTime
	tsStr := meta.PhotoTakenTime.Timestamp
	if tsStr == "" || tsStr == "0" {
		tsStr = meta.CreationTime.Timestamp
	}

	if tsStr == "" || tsStr == "0" {
		return fmt.Errorf("no valid timestamp found")
	}

	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return err
	}

	t := time.Unix(ts, 0)

	// Apply to file (Mtime and Atime)
	return os.Chtimes(mediaPath, t, t)
}
