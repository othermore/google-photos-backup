package processor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"

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

			// Check if ALREADY hardlinked (same inode)
			if m.areHardlinked(instance.Path, primary.Path) {
				continue
			}

			// If it's a file, delete it so we can create the link
			// (We rely on HASH equality to know content is same)
			if _, err := os.Stat(instance.Path); err == nil {
				if err := os.Remove(instance.Path); err != nil {
					logger.Error("❌ Failed to remove duplicate %s: %v", instance.Path, err)
					continue
				}
			}

			// Create Hardlink
			// primary.Path (existing) -> instance.Path (new link)
			if err := os.Link(primary.Path, instance.Path); err != nil {
				logger.Error("❌ Failed to hardlink %s -> %s: %v", instance.Path, primary.Path, err)
			} else {
				dedupedCount++
			}
		}
	}

	logger.Info("✅ Deduplication Complete. %d duplicates hardlinked.", dedupedCount)

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

// areHardlinked checks if two paths point to the same inode on the same device
func (m *Manager) areHardlinked(p1, p2 string) bool {
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
