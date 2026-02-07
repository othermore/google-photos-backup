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

// MinTruncationLength defines the minimum length to consider a filename for truncation matching
const MinTruncationLength = 40

// CorrectMetadata iterates over all media files and applies dates from JSON
func (m *Manager) CorrectMetadata() error {
	// 1. Build Index of JSONs and group by Directory for faster access
	// We store valid JSON paths in a map keyed by their "clean" name
	// but we also keep the raw filename for fuzzy matching.
	jsonFiles := make(map[string]bool)
	jsonByDir := make(map[string][]string)

	for path, meta := range m.FileIndex {
		if meta.IsJSON {
			jsonFiles[path] = true
			dir := filepath.Dir(path)
			jsonByDir[dir] = append(jsonByDir[dir], filepath.Base(path))
		}
	}

	logger.Info("   Indexed %d JSON sidecars.", len(jsonFiles))

	updated := 0
	ambiguousMatches := make(map[string][]string) // json -> [images...]
	secureMatches := make(map[string]string)      // image -> json

	// 2. Match Media Files - Pass 1: Secure Matches & Ambiguous Collection
	for mediaPath, meta := range m.FileIndex {
		if meta.IsJSON {
			continue
		}

		bestJSON, isAmbiguous, ambiguousCandidate := m.findBestJSON(mediaPath, jsonFiles, jsonByDir)
		if bestJSON != "" {
			secureMatches[mediaPath] = bestJSON
		} else if isAmbiguous {
			// Collect ambiguous matches for later user confirmation
			list := ambiguousMatches[ambiguousCandidate]
			list = append(list, mediaPath)
			ambiguousMatches[ambiguousCandidate] = list
		}
	}

	// Apply Secure Matches
	for mediaPath, jsonPath := range secureMatches {
		if err := m.applyDate(mediaPath, jsonPath); err == nil {
			updated++
		} else {
			logger.Debug("❌ Failed to apply date to %s: %v", filepath.Base(mediaPath), err)
		}
	}

	// 3. Handle Ambiguous Matches
	if len(ambiguousMatches) > 0 {
		secureCount := len(secureMatches)
		ambiguousCount := 0
		for _, imgs := range ambiguousMatches {
			ambiguousCount += len(imgs)
		}

		fixMode := strings.ToLower(m.FixAmbiguousMetadata)
		if fixMode == "" {
			fixMode = "interactive"
		}

		shouldApply := false

		// Logic:
		// IF yes -> apply automatically (log summary briefly)
		// IF no -> show full summary, do not prompt, do not apply
		// IF interactive -> show full summary, prompt, apply if user says yes

		if fixMode == "yes" {
			logger.Info("✅ Automatically applying %d ambiguous matches (fix-ambiguous-metadata=yes).", ambiguousCount)
			shouldApply = true
		} else {
			// Show Full Summary (Interactive or No)

			// Internationalized Messages
			msgEn := fmt.Sprintf("\n⚠️  Found %d secure matches. For %d other files, a secure JSON could not be identified, but %d of them have a POSSIBLE match if we ignore filename length safety checks.", secureCount, len(m.FileIndex)-len(jsonFiles)-secureCount, ambiguousCount)
			msgEs := fmt.Sprintf("\n⚠️  Se encontraron %d coincidencias seguras. Para otros %d archivos no se pudo identificar un JSON seguro, pero %d de ellos tienen una coincidencia POSIBLE si ignoramos las comprobaciones de seguridad de longitud de nombre.", secureCount, len(m.FileIndex)-len(jsonFiles)-secureCount, ambiguousCount)

			fmt.Println(msgEn)
			fmt.Println(msgEs)
			fmt.Println("---------------------------------------------------")

			// Show examples (up to 15)
			count := 0
			sortedJSONs := make([]string, 0, len(ambiguousMatches))
			for k := range ambiguousMatches {
				sortedJSONs = append(sortedJSONs, k)
			}

			for _, jsonPath := range sortedJSONs {
				images := ambiguousMatches[jsonPath]
				for _, img := range images {
					if count >= 15 {
						break
					}
					fmt.Printf("   📸 %s -> 📄 %s\n", filepath.Base(img), filepath.Base(jsonPath))
					count++
				}
				if count >= 15 {
					break
				}
			}
			if ambiguousCount > 15 {
				fmt.Printf("   ... and %d more / y %d más ...\n", ambiguousCount-15, ambiguousCount-15)
			}

			if fixMode == "interactive" {
				// Prompt user
				fmt.Println("\n❓ Apply these insecure matches? / ¿Aplicar estas coincidencias inseguras? (y/n): ")
				var response string
				fmt.Scanln(&response)

				if strings.ToLower(response) == "y" || strings.ToLower(response) == "s" {
					shouldApply = true
				} else {
					fmt.Println("Skipping insecure matches. / Saltando coincidencias inseguras.")
				}
			} else {
				// Mode "no"
				fmt.Println("\nSkipping insecure matches (fix-ambiguous-metadata=no). / Saltando coincidencias inseguras.")
			}
		}

		if shouldApply {
			if fixMode == "interactive" {
				fmt.Println("Applying insecure matches... / Aplicando coincidencias inseguras...")
			}
			for jsonPath, images := range ambiguousMatches {
				for _, img := range images {
					if err := m.applyDate(img, jsonPath); err == nil {
						updated++
					}
				}
			}
		}
	}

	logger.Info("✅ Metadata corrected for %d files.", updated)
	return nil
}

// findBestJSON implements the heuristics to find the matching JSON file
func (m *Manager) findBestJSON(mediaPath string, allJsonFiles map[string]bool, jsonByDir map[string][]string) (string, bool, string) {
	mediaName := filepath.Base(mediaPath)
	dir := filepath.Dir(mediaPath)

	// --- Level 1: Direct Exact Matches ---
	// 1. "file.jpg" -> "file.jpg.json"
	c1 := mediaPath + ".json"
	if allJsonFiles[c1] {
		return c1, false, ""
	}
	// 2. "file.jpg" -> "file.jpg.supplemental-metadata.json"
	c2 := mediaPath + ".supplemental-metadata.json"
	if allJsonFiles[c2] {
		return c2, false, ""
	}

	// --- Level 2: Clean Name Matches (Strip Extension & Suffixes) ---
	ext := filepath.Ext(mediaName)
	withoutExt := strings.TrimSuffix(mediaName, ext)
	cleanName := withoutExt

	// Strip "Numbering" suffixes: "file(1)" -> "file"
	if loc := reNumbering.FindStringIndex(cleanName); loc != nil {
		cleanName = cleanName[:loc[0]]
	}

	// Strip "Edited" suffixes: "file-edited" -> "file"
	if loc := reEdited.FindStringIndex(cleanName); loc != nil {
		cleanName = cleanName[:loc[0]]
	}

	candidates := []string{
		cleanName + ".json",       // "DSC00189.json"
		cleanName + ext + ".json", // "DSC00189.JPG.json"
		cleanName + ".jpg.json",
		cleanName + ".JPG.json",
		cleanName + ".jpeg.json",
		cleanName + ".heic.json",
		cleanName + ext + ".supplemental-metadata.json",
		cleanName + ".supplemental-metadata.json",
	}

	for _, candName := range candidates {
		fullPath := filepath.Join(dir, candName)
		if allJsonFiles[fullPath] {
			return fullPath, false, ""
		}
	}

	// --- Level 3: Directory Scan (Fuzzy & Truncation) ---
	dirJSONs := jsonByDir[dir]

	for _, jsonName := range dirJSONs {
		// Prepare JSON stems
		jStem := jsonName
		jStem = strings.TrimSuffix(jStem, ".json")
		jStem = strings.TrimSuffix(jStem, ".supplemental-metadata")
		jStem = strings.TrimSuffix(jStem, ".metadata")

		fullJsonPath := filepath.Join(dir, jsonName)

		// Case A: JSON is shorter prefix of Media (Truncation)
		if strings.HasPrefix(withoutExt, jStem) {
			if len(jStem) >= MinTruncationLength {
				return fullJsonPath, false, "" // Secure Match
			}
			return "", true, fullJsonPath // Ambiguous Match
		}

		// Case B: Media is shorter prefix of JSON
		if strings.HasPrefix(jStem, cleanName) {
			if len(cleanName) >= MinTruncationLength {
				return fullJsonPath, false, "" // Secure Match
			}
			return "", true, fullJsonPath // Ambiguous Match
		}
	}

	return "", false, ""
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
