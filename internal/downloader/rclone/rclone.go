package rclone

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"google-photos-backup/internal/logger"
)

// File represents a file in rclone lsjson output
type File struct {
	Path     string    `json:"Path"`
	Name     string    `json:"Name"`
	Size     int64     `json:"Size"`
	MimeType string    `json:"MimeType"`
	ModTime  time.Time `json:"ModTime"`
	IsDir    bool      `json:"IsDir"`
	ID       string    `json:"ID"`
}

// Client handles rclone operations
type Client struct {
	Remote string // e.g., "drive:"
}

// New creates a new rclone client
func New(remote string) *Client {
	return &Client{
		Remote: remote,
	}
}

// ListExports list files in the Takeout directory of the remote
// It looks for zip or tgz files in "Takeout/"
func (c *Client) ListExports() ([]File, error) {
	// rclone lsjson drive:Takeout --recursive --files-only --include "*.zip" --include "*.tgz"
	// Note: Takeout/ might be in the root or elsewhere. We assume root for now or user configures "drive/Takeout:"?
	// User guide says configure "drive:", so we assume "Takeout" folder exists at root.

	target := fmt.Sprintf("%sTakeout", c.Remote)

	args := []string{
		"lsjson",
		target,
		"--recursive",
		"--files-only",
		"--include", "*.zip",
		"--include", "*.tgz",
	}

	output, err := runRclone(args...)
	if err != nil {
		// If directory not found, it means no exports yet (fresh start)
		if strings.Contains(err.Error(), "directory not found") || strings.Contains(err.Error(), "exit status 3") {
			return []File{}, nil
		}
		return nil, fmt.Errorf("rclone list failed: %w", err)
	}

	var files []File
	if err := json.Unmarshal(output, &files); err != nil {
		return nil, fmt.Errorf("failed to parse rclone output: %w", err)
	}

	return files, nil
}

// MoveFile moves a file from remote to local directory
// This is a MOVE operation, so it deletes from source upon success.
func (c *Client) MoveFile(remoteFile string, localDir string) error {
	// rclone move drive:Takeout/path/to/file.zip /local/dir/ --progress

	source := fmt.Sprintf("%sTakeout/%s", c.Remote, remoteFile)

	args := []string{
		"move",
		source,
		localDir,
		"--progress",
		"--stats-one-line",
	}

	logger.Info("⬇️  Downloading from Drive (and deleting): %s", source)

	// We want to stream output to show progress?
	// runRclone captures stdout. For progress, maybe we should let it print to stdout/stderr?
	// But our logger uses different format.
	// For now, let's just run it and block. 'rclone move' is reliable.

	cmd := exec.Command("rclone", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rclone move failed: %w", err)
	}

	return nil
}

func runRclone(args ...string) ([]byte, error) {
	cmd := exec.Command("rclone", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s (stderr: %s)", err, stderr.String())
	}
	return stdout.Bytes(), nil
}
