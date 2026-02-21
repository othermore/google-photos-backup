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

func LoadDownloadState(path string) (statePtr *DownloadState, err error) {
	f, openErr := os.Open(path)
	if openErr != nil {
		return nil, openErr
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	var state DownloadState
	if decodeErr := json.NewDecoder(f).Decode(&state); decodeErr != nil {
		return nil, decodeErr
	}
	return &state, nil
}

func (s *DownloadState) Save(path string) (err error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if mkErr := os.MkdirAll(dir, 0755); mkErr != nil {
		return mkErr
	}

	f, createErr := os.Create(path)
	if createErr != nil {
		return createErr
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if encErr := enc.Encode(s); encErr != nil {
		return encErr
	}
	return nil
}
