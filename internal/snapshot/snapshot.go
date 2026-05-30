// Package snapshot persists the last-seen ranking state to a local JSON file so
// the next run has a baseline to diff against. Writes are atomic (temp + rename)
// so an interrupted run never leaves a half-written snapshot.
package snapshot

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/freebooters/p2p-ranking-board/internal/source"
)

// Snapshot is the full ranking state at one point in time, keyed by Entry.Key.
type Snapshot struct {
	TakenAt time.Time               `json:"taken_at"`
	Entries map[string]source.Entry `json:"entries"`
}

// DefaultPath resolves ${XDG_STATE_HOME:-~/.local/state}/p2p-ranking-board/snapshot.json.
func DefaultPath() string {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "p2p-ranking-board", "snapshot.json")
}

// Load reads the snapshot at path. The bool is false (with nil error) when no
// snapshot exists yet — the caller treats that as "first run, establish baseline".
func Load(path string) (*Snapshot, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var s Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, false, fmt.Errorf("parse snapshot %s: %w", path, err)
	}
	if s.Entries == nil {
		s.Entries = map[string]source.Entry{}
	}
	return &s, true, nil
}

// Save writes the snapshot atomically: a sibling temp file is written, fsync'd,
// then renamed over the target.
func Save(path string, s *Snapshot) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".snapshot-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once renamed

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
