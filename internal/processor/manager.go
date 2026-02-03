package processor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google-photos-backup/internal/logger"
)

type Manager struct {
	InputDir        string
	OutputDir       string
	AlbumsDir       string
	DeleteOrigin    bool
	TargetExport    string // If set, process only this Export ID
	ForceMetadata   bool   // Force metadata correction even if export is done
	ForceExtraction bool   // Force extraction even if export is done
	ForceDedup      bool   // Force global deduplication check

	// Index: Key = Absolute Path, Value = Metadata
	FileIndex map[string]FileMetadata

	// Set of processed Export IDs to avoid reprocessing
	ProcessedExports map[string]bool

	// Set of processed Archive paths (relative to ID?) to support partial reprocessing
	// Key: ID/Filename
	ProcessedArchives map[string]bool
}

type FileMetadata struct {
	Path      string
	Hash      string
	Size      int64
	Extension string
	IsJSON    bool
}

func NewManager(inputDir, outputDir, albumsDir string) *Manager {
	return &Manager{
		InputDir:          inputDir,
		OutputDir:         outputDir,
		AlbumsDir:         albumsDir,
		FileIndex:         make(map[string]FileMetadata),
		ProcessedExports:  make(map[string]bool),
		ProcessedArchives: make(map[string]bool),
	}
}

func (m *Manager) Run() error {
	start := time.Now()

	// Phase 1: Scan History and Process Each Export
	// Also returns true if any work was actually performed (any export was processed).
	logger.Info("📦 Phase 1: Processing exports from history...")
	workPerformed, err := m.ProcessExports()
	if err != nil {
		return fmt.Errorf("processing list failed: %w", err)
	}

	// Phase 2: Global Deduplication
	// Runs if we did some work OR if explicitly forced.
	// If the user just runs 'process' with no new files, we skip this to be idempotent and fast.
	if workPerformed || m.ForceDedup {
		if err := m.DeduplicateAndOrganize(); err != nil {
			return fmt.Errorf("deduplication failed: %w", err)
		}
	} else {
		logger.Info("⏭️  Skipping deduplication (no changes detected and --force-dedup not set)")
	}

	logger.Info("✨ Processing finished in %s", time.Since(start))
	return nil
}

func (m *Manager) ScanRaw(dir string, computeHash bool) error {
	logger.Info("🔍 Scanning existing files in %s...", dir)
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if info.Name() == "processing_index.json" || info.Name() == "state.json" || info.Name() == ".DS_Store" {
			return nil
		}

		// Skip Symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		absPath, _ := filepath.Abs(path)
		// Skip if already in index
		if _, ok := m.FileIndex[absPath]; ok {
			return nil
		}

		hash := ""
		if computeHash {
			// Hash file
			f, err := os.Open(path)
			if err != nil {
				return nil
			}
			defer f.Close()

			hasher := sha256.New()
			if _, err := io.Copy(hasher, f); err != nil {
				return nil
			}
			hash = hex.EncodeToString(hasher.Sum(nil))
		}

		ext := strings.ToLower(filepath.Ext(path))
		m.FileIndex[absPath] = FileMetadata{
			Path:      absPath,
			Hash:      hash,
			Size:      info.Size(),
			Extension: ext,
			IsJSON:    ext == ".json",
		}
		return nil
	})
}
