package registry

import (
	"encoding/json"
	"os"
	"time"
)

// FileIndexEntry represents a single file's metadata for deduplication
type FileIndexEntry struct {
	RelPath string    `json:"rel_path"`
	Hash    string    `json:"hash"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	Inode   uint64    `json:"inode,omitempty"` // Optimization for local filesystem
}

// Index represents the complete index of a directory (snapshot or master)
type Index struct {
	Files map[string]FileIndexEntry `json:"files"` // Key can be RelPath or Hash depending on usage, usually RelPath
}

// NewIndex creates a new empty Index
func NewIndex() *Index {
	return &Index{
		Files: make(map[string]FileIndexEntry),
	}
}

// LoadIndex loads an index from a JSON file
func LoadIndex(path string) (*Index, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewIndex(), nil
		}
		return nil, err
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	if idx.Files == nil {
		idx.Files = make(map[string]FileIndexEntry)
	}
	return &idx, nil
}

// Save writes the index to a JSON file
func (idx *Index) Save(path string) error {
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// AddOrUpdate updates an entry in the index
func (idx *Index) AddOrUpdate(entry FileIndexEntry) {
	idx.Files[entry.RelPath] = entry
}

// Get returns an entry by relative path
func (idx *Index) Get(relPath string) (FileIndexEntry, bool) {
	entry, ok := idx.Files[relPath]
	return entry, ok
}
