package registry

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
	"time"
)

type ExportStatus string

const (
	StatusRequested   ExportStatus = "requested"
	StatusInProgress  ExportStatus = "in_progress"
	StatusReady       ExportStatus = "ready"       // Ready to download
	StatusDownloading ExportStatus = "downloading" // Downloading
	StatusProcessed   ExportStatus = "processed"   // Extracted and organized (Success)
	StatusExpired     ExportStatus = "expired"
	StatusFailed      ExportStatus = "failed"
	StatusCancelled   ExportStatus = "cancelled"
)

type ExportEntry struct {
	ID             string       `json:"id"` // Archive ID from Google
	RequestedAt    time.Time    `json:"requested_at"`
	Status         ExportStatus `json:"status"`
	CompletedAt    time.Time    `json:"completed_at,omitempty"`
	FileCount      int          `json:"file_count,omitempty"`       // Number of zip files
	TotalSize      string       `json:"total_size,omitempty"`       // String like "50 GB"
	NewPhotosCount int          `json:"new_photos_count,omitempty"` // Added to backup
	Error          string       `json:"error,omitempty"`
	// Deprecated: Files are now stored in a separate state.json file per export.
	// This field is kept for migration purposes only.
	Files []DownloadFile `json:"files,omitempty"` // List of files to download
}

type DownloadFile struct {
	PartNumber      int    `json:"part_number"` // 1-based index
	Filename        string `json:"filename"`    // e.g. "takeout-20240201-001.zip"
	Size            string `json:"size"`        // e.g. "50 GB"
	SizeBytes       int64  `json:"size_bytes,omitempty"`
	DownloadedBytes int64  `json:"downloaded_bytes,omitempty"`
	Status          string `json:"status"` // "pending", "downloading", "completed", "failed"
	URL             string `json:"url,omitempty"`
}

type Registry struct {
	FilePath string        `json:"-"`
	Exports  []ExportEntry `json:"exports"`
	mu       sync.RWMutex
}

func New(path string) (*Registry, error) {
	r := &Registry{
		FilePath: path,
		Exports:  []ExportEntry{},
	}
	if err := r.Load(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Registry) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	f, err := os.Open(r.FilePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	// Leemos línea a línea (JSONL)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry ExportEntry
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if err := json.Unmarshal(line, &entry); err == nil {
			r.Exports = append(r.Exports, entry)
		}
	}
	return scanner.Err()
}

func (r *Registry) Save() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f, err := os.Create(r.FilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	// Guardamos en formato JSONL (una línea por entrada) para que sea tipo log legible
	for _, entry := range r.Exports {
		if err := enc.Encode(entry); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) Add(entry ExportEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Exports = append(r.Exports, entry)
}

// GetLastSuccessful returns the last export with StatusProcessed (fully completed)
func (r *Registry) GetLastSuccessful() *ExportEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := len(r.Exports) - 1; i >= 0; i-- {
		if r.Exports[i].Status == StatusProcessed {
			return &r.Exports[i]
		}
	}
	return nil
}

// Get devuelve un puntero a la entrada si existe, o nil
func (r *Registry) Get(id string) *ExportEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := range r.Exports {
		if r.Exports[i].ID == id {
			return &r.Exports[i]
		}
	}
	return nil
}

// MergeOrphan intenta asignar un ID a una solicitud pendiente (sin ID) existente.
// Devuelve true si encontró una huérfana y la actualizó.
func (r *Registry) MergeOrphan(id string, createdAt time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Buscamos la solicitud pendiente más reciente (iterando desde el final)
	for i := len(r.Exports) - 1; i >= 0; i-- {
		if r.Exports[i].ID == "" && r.Exports[i].Status == StatusRequested {
			r.Exports[i].ID = id
			if !createdAt.IsZero() {
				r.Exports[i].RequestedAt = createdAt
			}
			return true
		}
	}
	return false
}

// Update actualiza una entrada existente
func (r *Registry) Update(entry ExportEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, e := range r.Exports {
		if e.ID == entry.ID {
			r.Exports[i] = entry
			return
		}
	}
}

// Exists comprueba si existe una exportación con ese ID
func (r *Registry) Exists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, e := range r.Exports {
		if e.ID == id {
			return true
		}
	}
	return false
}
