package processor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"google-photos-backup/internal/logger"
)

// dedup.go handles global deduplication and organization

// Instance represents a single occurrence of a file
type Instance struct {
	Path     string
	ExportID string
}

func (m *Manager) DeduplicateAndOrganize() error {
	logger.Info("🔄 Starting Phase 2: Global Deduplication & Organization...")

	// 1. Build Global Hash Map
	// Hash -> []Instance
	hashMap := make(map[string][]Instance)
	totalFiles := 0

	// Iterate over all known exports (from global state or directory walk)
	entries, err := os.ReadDir(m.InputDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		exportID := entry.Name()

		// Skip if not processed (check global state)
		if !m.ProcessedExports[exportID] {
			continue
		}

		// Load Local Index
		indexPath := filepath.Join(m.InputDir, exportID, IndexFileName)
		data, err := os.ReadFile(indexPath)
		if err != nil {
			logger.Info("⚠️  Could not read index for %s: %v", exportID, err)
			continue
		}

		var savedState State
		if err := json.Unmarshal(data, &savedState); err != nil {
			logger.Error("❌ Failed to parse index for %s: %v", exportID, err)
			continue
		}

		for path, meta := range savedState.FileIndex {
			if meta.IsJSON || meta.Hash == "" {
				continue
			}
			hashMap[meta.Hash] = append(hashMap[meta.Hash], Instance{
				Path:     path, // Absolute path
				ExportID: exportID,
			})
			totalFiles++
		}
	}

	logger.Info("📊 Validation: Analyzed %d files. Found %d unique hashes.", totalFiles, len(hashMap))

	// 2. Deduplicate In-Place
	dedupedCount := 0

	for _, instances := range hashMap {
		if len(instances) < 2 {
			continue
		}

		// A. Select Primary (Best path among existing files)
		primary := m.selectPrimary(instances) // Returns the Instance that should be the REAL FILE

		// B. Process others
		for _, instance := range instances {
			if instance.Path == primary.Path {
				continue
			}

			// Sanity check: If primary == instance (same path), skip
			if instance.Path == primary.Path {
				continue
			}

			// Logic:
			// instance -> primary

			// Check if instance is ALREADY a symlink to primary
			if m.isCorrectSymlink(instance.Path, primary.Path) {
				continue
			}

			// If it's a file, delete it
			if info, err := os.Lstat(instance.Path); err == nil {
				if info.Mode()&os.ModeSymlink == 0 {
					// It's a real file (duplicate). Delete it.
					if err := os.Remove(instance.Path); err != nil {
						logger.Error("❌ Failed to remove duplicate %s: %v", instance.Path, err)
						continue
					}
					// logger.Info("🗑️  Removed duplicate: %s", filepath.Base(instance.Path))
				} else {
					// It's a symlink pointing somewhere else. Remove it to re-link correctly.
					os.Remove(instance.Path)
				}
			}

			// Create Relative Symlink
			if err := m.createRelativeSymlink(primary.Path, instance.Path); err != nil {
				logger.Error("❌ Failed to link %s -> %s: %v", instance.Path, primary.Path, err)
			} else {
				dedupedCount++
			}
		}
	}

	logger.Info("✅ Deduplication Complete. %d duplicates linked.", dedupedCount)

	return nil
}

// selectPrimary chooses the "best" path to keep as real file
// Heuristic: Prefer path with "Year" or sensible Album name over "Photos from 2023" or "Hangout"
// If multiple matches, prefers shortest path length (root albums) or alphabetically?
func (m *Manager) selectPrimary(instances []Instance) Instance {
	best := instances[0]
	bestScore := m.scorePath(best.Path)

	for _, inst := range instances {
		score := m.scorePath(inst.Path)
		if score > bestScore {
			best = inst
			bestScore = score
		} else if score == bestScore {
			// Tie-breaker: Alphabetical (Stable)
			if inst.Path < best.Path {
				best = inst
			}
		}
	}
	return best
}

func (m *Manager) scorePath(path string) int {
	score := 0
	dir := filepath.Base(filepath.Dir(path))
	lower := strings.ToLower(dir)

	// 1. Penalize "Photos from 20XX" (Generic)
	if strings.Contains(lower, "photos from") || strings.Contains(lower, "fotos de") {
		score -= 50
	}

	// 2. Penalize "Trash" or "Bin"
	if strings.Contains(lower, "trash") || strings.Contains(lower, "bin") || strings.Contains(lower, "papelera") {
		score -= 100
	}

	// 3. Penalize "Hangout" or "Chat"
	if strings.Contains(lower, "hangout") {
		score -= 20
	}

	// 4. Boost "Named" albums (Anything not year-based usually)
	// Simple heuristic: If it doesn't look like a year "20xx" and isn't generic
	// (Skipping regex for simplicity, relying on penalties above)

	// 5. Shortest path depth? (Closer to root = better?)
	// Actually, usually deeper is "Live/2023/Album" vs "Photos from 2023".
	// The penalties above handle the most common Google Takeout noise.

	return score
}

// isCorrectSymlink checks if 'path' is a symlink pointing to 'target'
func (m *Manager) isCorrectSymlink(path, target string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return false
	}

	// Read link
	dest, err := os.Readlink(path)
	if err != nil {
		return false
	}

	// Resolve absolute paths to compare
	// 'dest' might be relative (../foo.jpg)

	// Construct absolute path of destination
	var absDest string
	if filepath.IsAbs(dest) {
		absDest = dest
	} else {
		absDest = filepath.Join(filepath.Dir(path), dest)
	}

	// Compare Absolutes
	absTarget, _ := filepath.Abs(target)
	absDest, _ = filepath.Abs(absDest)

	return absDest == absTarget
}

// createRelativeSymlink creates a relative link at 'linkAbs' pointing to 'targetAbs'
func (m *Manager) createRelativeSymlink(targetAbs, linkAbs string) error {
	// Ensure parent dir exists
	if err := os.MkdirAll(filepath.Dir(linkAbs), 0755); err != nil {
		return err
	}

	// Calculate relative path: linkDir -> target
	rel, err := filepath.Rel(filepath.Dir(linkAbs), targetAbs)
	if err != nil {
		// Fallback to absolute
		return os.Symlink(targetAbs, linkAbs)
	}

	return os.Symlink(rel, linkAbs)
}
