package filehistory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Dir returns the file history directory for a project.
func Dir(projectRoot string) string {
	return filepath.Join(projectRoot, ".ergate", "file-history")
}

// Snapshot represents a saved version of a file.
type Snapshot struct {
	Path      string    `json:"path"`
	SavedAt   time.Time `json:"saved_at"`
	Version   int       `json:"version"`
	BackupPath string   `json:"backup_path"`
}

// Tracker manages file edit history within a project.
type Tracker struct {
	mu       sync.Mutex
	dir      string
	versions map[string]int // file path -> last version number
}

// NewTracker creates a file history tracker.
func NewTracker(projectRoot string) *Tracker {
	return &Tracker{
		dir:      Dir(projectRoot),
		versions: make(map[string]int),
	}
}

// SaveBackup saves the current content of a file before modification.
// Returns the backup path and version number.
func (t *Tracker) SaveBackup(filePath string) (*Snapshot, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file for backup: %w", err)
	}

	os.MkdirAll(t.dir, 0o700)

	abs, _ := filepath.Abs(filePath)
	// Track version per file
	t.versions[abs]++
	version := t.versions[abs]

	// Backup filename: <sanitized_path>_v<version>_<timestamp>
	safeName := strings.ReplaceAll(abs, "/", "_")
	safeName = strings.ReplaceAll(safeName, "\\", "_")
	backupName := fmt.Sprintf("%s_v%d_%d", safeName, version, time.Now().Unix())
	backupPath := filepath.Join(t.dir, backupName)

	if err := os.WriteFile(backupPath, data, 0o644); err != nil {
		return nil, fmt.Errorf("write backup: %w", err)
	}

	return &Snapshot{
		Path:       abs,
		SavedAt:    time.Now(),
		Version:    version,
		BackupPath: backupPath,
	}, nil
}

// GetLastBackup returns the most recent backup for a file.
func (t *Tracker) GetLastBackup(filePath string) (*Snapshot, error) {
	abs, _ := filepath.Abs(filePath)
	t.mu.Lock()
	version := t.versions[abs]
	t.mu.Unlock()

	if version == 0 {
		return nil, fmt.Errorf("no backup found for %s", filePath)
	}

	safeName := strings.ReplaceAll(abs, "/", "_")
	safeName = strings.ReplaceAll(safeName, "\\", "_")

	// Find the most recent backup file
	entries, err := os.ReadDir(t.dir)
	if err != nil {
		return nil, err
	}

	prefix := safeName + "_v"
	for i := len(entries) - 1; i >= 0; i-- {
		if strings.HasPrefix(entries[i].Name(), prefix) {
			backupPath := filepath.Join(t.dir, entries[i].Name())
			return &Snapshot{
				Path:       abs,
				Version:    version,
				BackupPath: backupPath,
			}, nil
		}
	}
	return nil, fmt.Errorf("backup file not found for %s", filePath)
}

// List returns all saved backups.
func (t *Tracker) List() []Snapshot {
	t.mu.Lock()
	defer t.mu.Unlock()

	var snapshots []Snapshot
	entries, _ := os.ReadDir(t.dir)
	for _, e := range entries {
		if !e.IsDir() {
			info, _ := e.Info()
			snapshots = append(snapshots, Snapshot{
				BackupPath: filepath.Join(t.dir, e.Name()),
				SavedAt:    info.ModTime(),
			})
		}
	}
	return snapshots
}
