package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func GetTryPath() string {
	if tryPath := os.Getenv("TRY_PATH"); tryPath != "" {
		return tryPath
	}
	
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", "tries")
	}
	
	return filepath.Join(home, "src", "tries")
}

func EnsureTryDirectory() error {
	tryPath := GetTryPath()
	return os.MkdirAll(tryPath, 0755)
}

func GenerateDatedName(name string) string {
	if name == "" {
		name = "experiment"
	}
	
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ToLower(name)
	
	date := time.Now().Format("2006-01-02")
	baseName := fmt.Sprintf("%s-%s", date, name)
	
	tryPath := GetTryPath()
	finalName := baseName
	counter := 1
	
	for {
		fullPath := filepath.Join(tryPath, finalName)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			break
		}
		finalName = fmt.Sprintf("%s-%d", baseName, counter)
		counter++
	}
	
	return finalName
}

func CreateDirectory(name string) (string, error) {
	if err := EnsureTryDirectory(); err != nil {
		return "", fmt.Errorf("failed to ensure try directory: %w", err)
	}
	
	dirName := GenerateDatedName(name)
	fullPath := filepath.Join(GetTryPath(), dirName)
	
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}
	
	return fullPath, nil
}

func DeleteDirectory(path string) error {
	tryPath := GetTryPath()
	
	if !strings.HasPrefix(path, tryPath) {
		return fmt.Errorf("can only delete directories within %s", tryPath)
	}
	
	// Check if this is a git worktree and remove it properly if so
	if isWorktree(path) {
		if err := removeWorktree(path); err != nil {
			// Log the error but continue with deletion
			// The worktree might already be unregistered or the parent repo might be gone
			fmt.Fprintf(os.Stderr, "Warning: failed to unregister worktree: %v\n", err)
		}
	}
	
	return os.RemoveAll(path)
}

// isWorktree checks if a directory is a git worktree
func isWorktree(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	
	// Worktrees have .git as a file, not a directory
	return !info.IsDir()
}

// removeWorktree unregisters a worktree from its parent repository
func removeWorktree(worktreePath string) error {
	// Read the .git file to find the parent repository
	gitFile := filepath.Join(worktreePath, ".git")
	content, err := os.ReadFile(gitFile)
	if err != nil {
		return fmt.Errorf("failed to read .git file: %w", err)
	}
	
	// Parse the gitdir path
	gitdirLine := strings.TrimSpace(string(content))
	if !strings.HasPrefix(gitdirLine, "gitdir: ") {
		return fmt.Errorf("invalid .git file format")
	}
	
	gitdir := strings.TrimPrefix(gitdirLine, "gitdir: ")
	
	// The gitdir points to something like /path/to/repo/.git/worktrees/name
	// We need to find the main repository path
	var mainRepoPath string
	if strings.Contains(gitdir, "/.git/worktrees/") {
		parts := strings.Split(gitdir, "/.git/worktrees/")
		if len(parts) > 0 {
			mainRepoPath = parts[0]
		}
	} else {
		return fmt.Errorf("unexpected gitdir format: %s", gitdir)
	}
	
	if mainRepoPath == "" {
		return fmt.Errorf("could not determine main repository path")
	}
	
	// Run git worktree remove from the main repository
	cmd := exec.Command("git", "worktree", "remove", worktreePath)
	cmd.Dir = mainRepoPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try with force flag if the first attempt fails
		cmd = exec.Command("git", "worktree", "remove", "-f", worktreePath)
		cmd.Dir = mainRepoPath
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to remove worktree: %w\nOutput: %s", err, string(output))
		}
	}
	
	return nil
}

func ExtractNameFromGitURL(url string) string {
	url = strings.TrimSuffix(url, ".git")
	
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		name := parts[len(parts)-1]
		
		if strings.Contains(url, "://") {
			if len(parts) > 1 {
				name = parts[len(parts)-1]
			}
		} else if strings.HasPrefix(url, "git@") {
			colonParts := strings.Split(url, ":")
			if len(colonParts) > 1 {
				pathParts := strings.Split(colonParts[1], "/")
				if len(pathParts) > 0 {
					name = pathParts[len(pathParts)-1]
				}
			}
		}
		
		return name
	}
	
	return "repo"
}