package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// accessRecord tracks how a directory has been used through try.
type accessRecord struct {
	Count      int       `json:"count"`
	LastAccess time.Time `json:"last_access"`
}

// History maps an absolute directory path to its access record. It powers the
// frecency ranking: directories you open often and recently bubble to the top.
type History map[string]accessRecord

// getDataDir returns the directory where try keeps its state. It honors
// TRY_DATA_DIR (used by tests and power users) and XDG_CONFIG_HOME, defaulting
// to ~/.config/try.
func getDataDir() string {
	if d := os.Getenv("TRY_DATA_DIR"); d != "" {
		return d
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "try")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "try")
	}
	return filepath.Join(home, ".config", "try")
}

func getHistoryPath() string {
	return filepath.Join(getDataDir(), "history.json")
}

// LoadHistory reads the access history. It never fails the caller: a missing or
// corrupt file yields an empty history so ranking degrades gracefully.
func LoadHistory() History {
	h := History{}
	data, err := os.ReadFile(getHistoryPath())
	if err != nil {
		return h
	}
	if err := json.Unmarshal(data, &h); err != nil || h == nil {
		return History{}
	}
	return h
}

// saveHistory writes the history atomically (temp file + rename) so a crash or
// concurrent run can't leave a half-written file behind.
func saveHistory(h History) error {
	dir := getDataDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, fmt.Sprintf(".history.%d.tmp", os.Getpid()))
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, getHistoryPath()); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// RecordAccess bumps the access count and last-access time for a directory that
// the user just opened through try. Only paths inside the try directory are
// tracked, and any storage error is returned but safe to ignore (ranking simply
// won't improve). Records for directories that no longer exist are pruned.
func RecordAccess(path string) error {
	tryPath := GetTryPath()
	if !strings.HasPrefix(path, tryPath) {
		return nil
	}

	h := LoadHistory()
	rec := h[path]
	rec.Count++
	rec.LastAccess = time.Now()
	h[path] = rec

	pruneMissing(h)
	return saveHistory(h)
}

// pruneMissing drops history entries whose directory no longer exists on disk,
// keeping the file from growing without bound as throwaway directories are
// deleted.
func pruneMissing(h History) {
	for path := range h {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			delete(h, path)
		}
	}
}

// WriteCdPath is the single choke point for "we are about to cd into this
// directory": it records the path for the shell wrapper to pick up and bumps
// the directory's frecency. All commands route their cd handoff through here.
func WriteCdPath(path string) error {
	if home, err := os.UserHomeDir(); err == nil {
		cdFile := filepath.Join(home, ".try_cd")
		_ = os.WriteFile(cdFile, []byte(path), 0644)
	}
	return RecordAccess(path)
}
