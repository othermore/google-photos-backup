package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type DownloadState struct {
	ID          string         `json:"id"`
	LastUpdated time.Time      `json:"last_updated"`
	Files       []DownloadFile `json:"files"`
}

func LoadDownloadState(path string) (*DownloadState, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var state DownloadState
	if err := json.NewDecoder(f).Decode(&state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *DownloadState) Save(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
}
