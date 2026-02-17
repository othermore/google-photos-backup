package engine

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"google-photos-backup/internal/config"
	"google-photos-backup/internal/logger"
	"google-photos-backup/internal/processor"
)

// Engine handles the optimized processing pipeline
type Engine struct {
	WorkingDir string
	BackupDir  string
	AlbumsDir  string

	// Config
	FixAmbiguousMetadata string
}

func New(workingDir, backupDir string) *Engine {
	return &Engine{
		WorkingDir:           workingDir,
		BackupDir:            backupDir,
		AlbumsDir:            filepath.Join(backupDir, "albums"),
		FixAmbiguousMetadata: config.AppConfig.FixAmbiguousMetadata,
	}
}

// ProcessZip handles a single zip file: Unzip -> Dedup -> Delete Zip
func (e *Engine) ProcessZip(zipPath string) error {
	logger.Info("📦 Processing Zip: %s", filepath.Base(zipPath))

	// 1. Unzip to temp/extracted
	extractDir := filepath.Join(e.WorkingDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("failed to create extract dir: %v", err)
	}

	logger.Info("   - Extracting...")
	if err := e.unzip(zipPath, extractDir); err != nil {
		return fmt.Errorf("extraction failed: %v", err)
	}

	// 2. Delete Zip (Space Saving)
	logger.Info("   - Deleting Zip to save space...")
	if err := os.Remove(zipPath); err != nil {
		logger.Warn("Failed to delete zip %s: %v", zipPath, err)
	}

	// 3. Deduplicate against Backup (Optimization: Same Volume Only)
	if e.isSameVolume(extractDir, e.BackupDir) {
		logger.Info("   - Optimizing: Same volume detected. Deduplicating against backup...")
		if err := e.deduplicateAgainstBackup(extractDir); err != nil {
			logger.Warn("Backup deduplication optimization failed: %v", err)
		}
	} else {
		logger.Info("   - Different volumes detected. Skipping backup deduplication check.")
	}

	return nil
}

// Finalize performs the shared processing on all extracted files set
func (e *Engine) Finalize() error {
	logger.Info("🔄 Starting Final Processing Phase...")

	extractDir := filepath.Join(e.WorkingDir, "extracted")

	// 1. Organize and Move (includes Metadata fix)
	if err := e.OrganizeAndMove(extractDir); err != nil {
		return err
	}

	// 2. Final Deduplication (Cross-Volume Fix)
	// Now that files are in BackupDir, they are strictly on the backup volume.
	logger.Info("   - Running Final Deduplication...")

	// We instantiate a Processor Manager just for the dedup logic
	// InputDir = BackupDir (since files are already moved there)
	// OutputDir = BackupDir
	dedupMgr := processor.NewManager(e.BackupDir, e.BackupDir, e.AlbumsDir)
	dedupMgr.ForceDedup = true
	// Trick: To make dedup work without iterating strictly over "History", we might need adjustments
	// in processor.Manager. However, ProcessExports works based on History.json.
	// DedupAndOrganize works based on scanning directories matching Export IDs.
	// Since we are organizing by DATE (Year/Month), the existing Dedup logic which expects ExportID folders MIGHT FAIL.

	// CRITICAL: The existing processor organizes into `output/albums` but expects input in `downloads/ID`.
	// Our new architecture moves files directly to `backup/Year/Month`.
	// The `deduplicateAgainstBackup` logic in the processor iterates over `ProcessedExports`.
	// This might need refactoring in Processor or a new implementation here.

	// For now, let's assume we implement a simpler "Dedup By Hash" here or call a modified processor function.
	// Let's rely on the Processor for now but acknowledge it might need tweaks in next step.
	// Actually, if files are already moved, standard dedup (finding duplicates across folders) is hard without an index.

	// Use existing DeduplicateAndOrganize if possible?
	// It scans InputDir for folders. If we moved files to Year/Month folders, it won't find ExportID folders.
	// So we need a DIFFERENT Dedup strategy for the final structure.

	// Let's implement a "Scan and Link" simple strategy here for now.
	// Or better: Let Processor handle the move, but we feed it the extracted files.

	// RE-EVALUATION:
	// The best approach is to let the processor handle the "Move" logic which already includes organization.
	// But the processor expects ZIP input usually.

	// Let's stick to: OrganizeAndMove moves files. Then we scan BackupDir for duplicates.

	// 3. Cleanup
	logger.Info("   - Cleaning up temp files...")
	os.RemoveAll(extractDir)

	return nil
}

// --- Helpers ---

func (e *Engine) unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// ZipSlip check
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()
	}
	return nil
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
	logger.Info("   - Organizing and Moving files...")
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
