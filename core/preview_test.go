package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPreviewReadmeHeading(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "README.md", "# Redis Streams Experiment\n\nDetails below.\n")

	if got, want := DirectoryPreview(dir, false), "Redis Streams Experiment"; got != want {
		t.Errorf("preview = %q, want %q", got, want)
	}
}

func TestPreviewReadmeSkipsDecoration(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "README.md",
		"[![CI](https://x/badge.svg)](https://x)\n\n===\n\nTesting consumer groups with the Go client\n")

	if got, want := DirectoryPreview(dir, false), "Testing consumer groups with the Go client"; got != want {
		t.Errorf("preview = %q, want %q", got, want)
	}
}

func TestPreviewPlainReadme(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "README", "quick scratchpad for JWT parsing\n")

	if got, want := DirectoryPreview(dir, false), "quick scratchpad for JWT parsing"; got != want {
		t.Errorf("preview = %q, want %q", got, want)
	}
}

func TestPreviewEmptyDirectory(t *testing.T) {
	if got, want := DirectoryPreview(t.TempDir(), false), "empty directory"; got != want {
		t.Errorf("preview = %q, want %q", got, want)
	}
}

func TestPreviewEntrySummary(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"main.go", "go.mod", "util.go", "notes.txt"} {
		writeFile(t, dir, name, "x")
	}
	writeFile(t, dir, ".hidden", "x") // dot entries are noise, not counted

	if got, want := DirectoryPreview(dir, false), "4 items: go.mod, main.go, notes.txt…"; got != want {
		t.Errorf("preview = %q, want %q", got, want)
	}
}

func TestPreviewSingleEntry(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "x")

	if got, want := DirectoryPreview(dir, false), "1 item: main.go"; got != want {
		t.Errorf("preview = %q, want %q", got, want)
	}
}

func TestPreviewGitLastCommit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "try streams consumer groups")

	// No README: falls through to the last commit subject.
	if got, want := DirectoryPreview(dir, true), "last commit: try streams consumer groups"; got != want {
		t.Errorf("preview = %q, want %q", got, want)
	}

	// A README outranks the commit subject.
	writeFile(t, dir, "README.md", "# The Real Description\n")
	if got, want := DirectoryPreview(dir, true), "The Real Description"; got != want {
		t.Errorf("preview = %q, want %q", got, want)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
