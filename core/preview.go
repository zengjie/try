package core

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// readmeCandidates are checked in order when looking for a description.
var readmeCandidates = []string{"README.md", "README", "README.txt", "readme.md", "Readme.md"}

// maxPreviewEntries is how many entry names the listing summary shows.
const maxPreviewEntries = 3

// DirectoryPreview returns a one-line description of what's inside a
// directory, used by the selector to tell similar experiments apart without
// opening them. Priority: the first meaningful README line (deliberate
// documentation), then the last git commit subject (recent activity), then a
// summary of the directory's entries.
func DirectoryPreview(path string, isGit bool) string {
	if line := readmeFirstLine(path); line != "" {
		return line
	}
	if isGit {
		if subject := lastCommitSubject(path); subject != "" {
			return "last commit: " + subject
		}
	}
	return entrySummary(path)
}

// readmeFirstLine returns the first meaningful line of the directory's README:
// markdown heading markers are stripped, and blank, badge-only, and
// decoration-only lines are skipped.
func readmeFirstLine(dir string) string {
	for _, name := range readmeCandidates {
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		// Only look at the top of the file; a README whose first real content
		// is further down won't make a useful one-liner anyway.
		for i := 0; i < 20 && scanner.Scan(); i++ {
			line := strings.TrimSpace(scanner.Text())
			line = strings.TrimLeft(line, "# ")
			if line == "" || isDecorationLine(line) {
				continue
			}
			return line
		}
		return "" // README exists but has no usable line; don't try other names
	}
	return ""
}

// isDecorationLine reports whether a README line carries no descriptive value:
// badge images, horizontal rules, or setext heading underlines.
func isDecorationLine(line string) bool {
	if strings.HasPrefix(line, "[![") || strings.HasPrefix(line, "![") {
		return true
	}
	trimmed := strings.Trim(line, "=-* _")
	return trimmed == ""
}

// lastCommitSubject returns the subject of the most recent commit, or "" if it
// cannot be determined (no commits yet, git missing, etc.).
func lastCommitSubject(dir string) string {
	cmd := exec.Command("git", "-C", dir, "log", "-1", "--format=%s")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// entrySummary describes the directory by its contents, e.g.
// "5 items: main.go, go.mod, ui…". Dot entries are skipped as noise.
func entrySummary(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		names = append(names, e.Name())
	}
	if len(names) == 0 {
		return "empty directory"
	}

	shown := names
	suffix := ""
	if len(names) > maxPreviewEntries {
		shown = names[:maxPreviewEntries]
		suffix = "…"
	}
	noun := "items"
	if len(names) == 1 {
		noun = "item"
	}
	return fmt.Sprintf("%d %s: %s%s", len(names), noun, strings.Join(shown, ", "), suffix)
}
