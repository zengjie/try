package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/zengjie/try/core"
)

func TestUsageTagsContent(t *testing.T) {
	cases := []struct {
		count    int
		git      bool
		worktree bool
		want     string
	}{
		{0, false, false, "-"},
		{0, true, false, "git"},
		{0, false, true, "worktree"},
		{0, true, true, "git worktree"},
		{5, false, false, "5x"},
		{12, true, false, "12x git"},
		{3, false, true, "3x worktree"},
		{1, true, true, "1x git worktree"},
	}
	for _, c := range cases {
		if got := usageTagsContent(c.count, c.git, c.worktree); got != c.want {
			t.Errorf("usageTagsContent(%d, git=%v, wt=%v) = %q, want %q",
				c.count, c.git, c.worktree, got, c.want)
		}
	}
}

func TestSelectedPreview(t *testing.T) {
	tryDir := t.TempDir()
	t.Setenv("TRY_PATH", tryDir)

	dir := filepath.Join(tryDir, "2026-07-06-exp")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# My Experiment\n"), 0644); err != nil {
		t.Fatal(err)
	}

	m := NewModel()
	m.list.SetItems([]list.Item{
		DirectoryItem{Directory: core.Directory{Name: "2026-07-06-exp", Path: dir}},
	})

	if got, want := m.SelectedPreview(), "My Experiment"; got != want {
		t.Errorf("preview = %q, want %q", got, want)
	}
	// Second call must be served from the cache.
	if _, ok := m.previewCache[dir]; !ok {
		t.Errorf("preview was not cached")
	}
	os.Remove(filepath.Join(dir, "README.md"))
	if got := m.SelectedPreview(); got != "My Experiment" {
		t.Errorf("expected cached preview, got %q", got)
	}

	// The "create new" item previews the full path it would create.
	m.list.SetItems([]list.Item{
		DirectoryItem{IsCreateNew: true, CreateQuery: "fresh idea",
			Directory: core.Directory{Name: "✨ Create new: x"}},
	})
	got := m.SelectedPreview()
	if !strings.HasPrefix(got, "will create "+tryDir) || !strings.HasSuffix(got, "-fresh-idea") {
		t.Errorf("create-new preview should show the full dated path, got %q", got)
	}
}
