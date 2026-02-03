package processor

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"google-photos-backup/internal/logger"
	"google-photos-backup/internal/registry"
)

// extract.go handles extraction of archives

// ProcessExports parses history and handles per-export processing
// ProcessExports parses history and handles per-export processing
func (m *Manager) ProcessExports() (bool, error) {
	// 0. Load Global State (to know what's already done)
	if err := m.LoadState(m.OutputDir, true); err != nil {
		logger.Error("⚠️  Failed to load global state: %v", err)
	}

	// 1. Try to read history.json from parent of downloads
	historyPath := filepath.Join(filepath.Dir(m.InputDir), "history.json")

	logger.Info("🔍 Checking history at: %s", historyPath)
	content, err := os.ReadFile(historyPath)
	if err != nil {
		logger.Error("❌ Could not read history.json: %v", err)
		return false, err
	}

	lines := strings.Split(string(content), "\n")
	foundAny := false
	workPerformed := false

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if entry.ID == "" {
			continue
		}

		// FILTER: Target Export
		if m.TargetExport != "" && m.TargetExport != entry.ID {
			continue
		}

		isDone := m.ProcessedExports[entry.ID]

		// Logic for what parts to run
		shouldExtract := m.ForceExtraction || !isDone
		shouldMetadata := m.ForceMetadata || shouldExtract
		shouldDedup := m.ForceDedup // Dedup implies scanning if we are forcing it

		if !shouldExtract && !shouldMetadata && !shouldDedup {
			continue
		}

		// Check if directory exists
		exportDir := filepath.Join(m.InputDir, entry.ID)
		if info, err := os.Stat(exportDir); err == nil && info.IsDir() {
			foundAny = true

			// If we are here, we are about to do something (or at least check)
			workPerformed = true

			logger.Info("📂 Processing export: %s (Status: %s)", entry.ID, entry.Status)

			// --- Per-Export Context Setup ---
			m.FileIndex = make(map[string]FileMetadata)
			// ... (rest of logic proceeds naturally)
			m.ProcessedArchives = make(map[string]bool)

			// Load Local State (ALWAYS needed for index)
			m.LoadState(exportDir, false)

			// If Forcing Extraction, ignore loaded "ProcessedArchive" state
			if m.ForceExtraction {
				m.ProcessedArchives = make(map[string]bool)
			}

			localRaw := filepath.Join(exportDir, "raw")
			if err := os.MkdirAll(localRaw, 0755); err != nil {
				logger.Error("❌ Failed to create raw dir for export %s: %v", entry.ID, err)
				continue
			}

			// Extraction Step OR Forced Scan (for metadata/dedup update)
			if shouldExtract || shouldMetadata || shouldDedup {
				// Scan Existing Files
				// Optimization:
				// 1. If ForceExtraction is ON, we re-extract (and re-hash) everything from stream. ScanRaw hashing is duplicate work. -> needHash = false
				// 2. If Resuming (shouldExtract=true, Force=false), we NEED to hash existing files on disk, as we won't re-extract them. -> needHash = true
				// 3. If ForceDedup is ON, we need hashes. -> needHash = true (unless ForceExtraction overrides it)

				// Logic: We need disk hashes if we are keeping existing files (Resume or Dedup Check)
				// We don't need them if we are about to blow them away (ForceExtraction) or if we only want metadata.
				needHash := (shouldExtract || shouldDedup) && !m.ForceExtraction

				// If we are forcing a deep deduplication check, we must DISCARD the loaded index
				// to force a fresh re-scan of the files on disk.
				if shouldDedup {
					m.FileIndex = make(map[string]FileMetadata)
				}

				m.ScanRaw(localRaw, needHash)

				if shouldExtract {
					// Process (Extract missing archives)
					if err := m.processExport(entry.ID, exportDir, localRaw); err != nil {
						logger.Error("❌ Error processing %s: %v", entry.ID, err)
					}
				}
			} else {
				if len(m.FileIndex) == 0 {
					logger.Info("ℹ️  Index empty, scanning raw files (with hashes)...")
					m.ScanRaw(localRaw, true)
				}
			}

			// Metadata Correction Step
			if shouldMetadata {
				logger.Info("📅 Correcting metadata for export %s...", entry.ID)
				m.CorrectMetadata()
			}

			// Save Local State for this export
			m.SaveState(exportDir)

			// Mark the entire export as processed in the global state
			if shouldExtract && shouldMetadata {
				m.ProcessedExports[entry.ID] = true
				m.SaveState(m.OutputDir)
			}
		}
	}

	if !foundAny {
		logger.Info("⚠️  No export directories from history found in %s", m.InputDir)
	}

	return workPerformed, nil
}

func (m *Manager) processExport(id, dir, rawDir string) error {
	// 1. Validate State
	statePath := filepath.Join(dir, "state.json")
	state, err := registry.LoadDownloadState(statePath)
	if err != nil {
		// Warn but maybe proceed scan? No, user wants strict validation.
		return fmt.Errorf("missing or invalid state.json: %w", err)
	}

	// Check completeness
	for _, f := range state.Files {
		key := id + "/" + f.Filename

		// If we already processed this specific archive, it's fine if it's missing
		if m.ProcessedArchives[key] {
			continue
		}

		// Otherwise, it MUST be present and completed
		if f.Status != "completed" {
			return fmt.Errorf("file %s is not marked as completed (status: %s)", f.Filename, f.Status)
		}

		fPath := filepath.Join(dir, f.Filename)
		info, err := os.Stat(fPath)
		if os.IsNotExist(err) {
			return fmt.Errorf("missing file: %s", f.Filename)
		}

		// Size check
		if f.SizeBytes > 0 && info.Size() != f.SizeBytes {
			return fmt.Errorf("size mismatch for %s: expected %d, got %d", f.Filename, f.SizeBytes, info.Size())
		}
	}

	// 2. Extract
	for _, f := range state.Files {
		key := id + "/" + f.Filename
		if m.ProcessedArchives[key] {
			continue
		}

		path := filepath.Join(dir, f.Filename)
		ext := strings.ToLower(filepath.Ext(path))

		logger.Info("➡️  Processing archive: %s", f.Filename)

		if ext == ".zip" {
			if err := m.extractZip(path, rawDir); err != nil {
				return fmt.Errorf("failed to extract %s: %w", f.Filename, err)
			}
		} else if ext == ".tgz" || strings.HasSuffix(strings.ToLower(f.Filename), ".tar.gz") {
			if err := m.extractTgz(path, rawDir); err != nil {
				return fmt.Errorf("failed to extract %s: %w", f.Filename, err)
			}
		}

		// Success!
		m.ProcessedArchives[key] = true
		m.SaveState(dir)

		if m.DeleteOrigin {
			os.Remove(path)
			logger.Info("🗑️  Deleted original archive: %s", f.Filename)
		}
	}

	return nil
}

func (m *Manager) extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if err := m.extractFile(f.Name, f, dest); err != nil {
			// Log but continue, don't fail the entire archive for one file
			logger.Error("⚠️  Failed to extract file %s from zip: %v", f.Name, err)
		}
	}
	return nil
}

func (m *Manager) extractTgz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if header.Typeflag == tar.TypeReg {
			if err := m.extractReader(header.Name, tr, dest, header.FileInfo().Mode()); err != nil {
				logger.Error("⚠️  Failed to extract file %s from tgz: %v", header.Name, err)
			}
		}
	}
	return nil
}

func (m *Manager) extractFile(name string, f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	return m.extractReader(name, rc, dest, f.FileInfo().Mode())
}

func (m *Manager) extractReader(name string, r io.Reader, dest string, mode os.FileMode) error {
	fpath := filepath.Join(dest, name)

	// ZipSlip check: Ensure fpath is inside dest
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	absFpath, err := filepath.Abs(fpath)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(absFpath, absDest) {
		return fmt.Errorf("invalid file path (ZipSlip detected): %s -> %s", name, fpath)
	}

	// Use absolute path for indexing
	absPath := absFpath

	// If it's a directory
	if strings.HasSuffix(name, "/") || mode.IsDir() {
		return os.MkdirAll(fpath, 0755)
	}

	// Create parent dirs
	if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
		return err
	}

	// Create file
	outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Multi-writer to calculate Hash on the fly
	hasher := sha256.New()
	writer := io.MultiWriter(outFile, hasher)

	written, err := io.Copy(writer, r)
	if err != nil {
		return err
	}

	// Store Metadata
	hash := hex.EncodeToString(hasher.Sum(nil))
	ext := strings.ToLower(filepath.Ext(fpath))

	m.FileIndex[absPath] = FileMetadata{
		Path:      absPath,
		Hash:      hash,
		Size:      written,
		Extension: ext,
		IsJSON:    ext == ".json",
	}

	return nil
}
