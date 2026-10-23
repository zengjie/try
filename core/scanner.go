package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Directory struct {
	Name         string
	Path         string
	CreatedTime  time.Time
	ModifiedTime time.Time
	AccessTime   time.Time
	AccessCount  int       // times opened through try (from frecency history)
	LastAccess   time.Time // last time opened through try (zero if never)
	Score        float64
	TextScore    float64
	TimeScore    float64
	IsGitRepo    bool
	IsWorktree   bool
}

func ScanDirectories() ([]Directory, error) {
	tryPath := GetTryPath()
	
	if _, err := os.Stat(tryPath); os.IsNotExist(err) {
		return []Directory{}, nil
	}
	
	entries, err := os.ReadDir(tryPath)
	if err != nil {
		return nil, err
	}

	history := LoadHistory()
	var directories []Directory

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		fullPath := filepath.Join(tryPath, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		dir := Directory{
			Name:         entry.Name(),
			Path:         fullPath,
			CreatedTime:  info.ModTime(),
			ModifiedTime: info.ModTime(),
			AccessTime:   info.ModTime(),
		}

		// Merge in frecency history (times this directory was opened via try).
		if rec, ok := history[fullPath]; ok {
			dir.AccessCount = rec.Count
			dir.LastAccess = rec.LastAccess
		}
		
		// Check if it's a git repository or worktree
		gitPath := filepath.Join(fullPath, ".git")
		if gitInfo, err := os.Stat(gitPath); err == nil {
			if gitInfo.IsDir() {
				// Regular git repository
				dir.IsGitRepo = true
			} else {
				// Git worktree (has .git file instead of directory)
				dir.IsWorktree = true
			}
		}
		
		directories = append(directories, dir)
	}
	
	return directories, nil
}

// FilterAndScoreDirectories scores each directory by blending a text-match
// signal with a usage (frecency) signal. The blend weight adapts to query
// length: an empty/short query leans on usage (what you actually open), while a
// longer, more specific query trusts the text match. When a query is present,
// directories with no text match are excluded.
func FilterAndScoreDirectories(directories []Directory, query string) []Directory {
	scorer := NewScorerFromConfig(LoadConfig())
	qlen := len([]rune(strings.TrimSpace(query)))
	wText := scorer.textWeight(qlen)

	var scored []Directory
	for _, dir := range directories {
		name := ExtractNameFromDirectory(dir.Name)
		textScore := scorer.calculateTextScore(name, query)

		// Exclude non-matches when searching (an empty query matches everything).
		if query != "" && textScore <= 0 {
			continue
		}

		usageScore := scorer.calculateUsageScore(dir.ModifiedTime, dir.AccessCount, dir.LastAccess)

		dir.TextScore = textScore
		dir.TimeScore = usageScore
		dir.Score = wText*textScore + (1-wText)*usageScore
		scored = append(scored, dir)
	}

	return scored
}

func SortDirectoriesByScore(directories []Directory) {
	sort.Slice(directories, func(i, j int) bool {
		if directories[i].Score != directories[j].Score {
			return directories[i].Score > directories[j].Score
		}

		return directories[i].ModifiedTime.After(directories[j].ModifiedTime)
	})
}

func GetRelativeAge(t time.Time) string {
	duration := time.Since(t)
	
	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if duration < 30*24*time.Hour {
		weeks := int(duration.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if duration < 365*24*time.Hour {
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
	
	years := int(duration.Hours() / 24 / 365)
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}