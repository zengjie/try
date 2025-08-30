package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zengjie/try/core"
)

func CloneRepository(url string) error {
	repoName := core.ExtractNameFromGitURL(url)
	dirName := core.GenerateDatedName(repoName)
	fullPath := filepath.Join(core.GetTryPath(), dirName)
	
	if err := core.EnsureTryDirectory(); err != nil {
		return fmt.Errorf("failed to ensure try directory: %w", err)
	}
	
	cmd := exec.Command("git", "clone", url, fullPath)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	
	// Write to .try_cd file for shell integration
	home, _ := os.UserHomeDir()
	cdFile := filepath.Join(home, ".try_cd")
	os.WriteFile(cdFile, []byte(fullPath), 0644)
	
	fmt.Println(fullPath)
	return nil
}

func CreateWorktree(repoPath string, name string) error {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	
	if !isGitRepository(absPath) {
		return fmt.Errorf("%s is not a git repository", absPath)
	}
	
	if name == "" {
		name = filepath.Base(absPath) + "-worktree"
	}
	
	dirName := core.GenerateDatedName(name)
	fullPath := filepath.Join(core.GetTryPath(), dirName)
	
	if err := core.EnsureTryDirectory(); err != nil {
		return fmt.Errorf("failed to ensure try directory: %w", err)
	}
	
	gitDir := findGitDir(absPath)
	if gitDir == "" {
		return fmt.Errorf("could not find .git directory for %s", absPath)
	}
	
	cmd := exec.Command("git", "worktree", "add", "--detach", fullPath)
	cmd.Dir = filepath.Dir(gitDir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		if strings.Contains(err.Error(), "not a git repository") {
			path, err := core.CreateDirectory(name)
			if err != nil {
				return err
			}
			// Write to .try_cd file for shell integration
			home, _ := os.UserHomeDir()
			cdFile := filepath.Join(home, ".try_cd")
			os.WriteFile(cdFile, []byte(path), 0644)
			
			fmt.Println(path)
			return nil
		}
		return fmt.Errorf("failed to create worktree: %w", err)
	}
	
	// Write to .try_cd file for shell integration
	home, _ := os.UserHomeDir()
	cdFile := filepath.Join(home, ".try_cd")
	os.WriteFile(cdFile, []byte(fullPath), 0644)
	
	fmt.Println(fullPath)
	return nil
}

func isGitRepository(path string) bool {
	gitPath := filepath.Join(path, ".git")
	if _, err := os.Stat(gitPath); err == nil {
		return true
	}
	
	parent := filepath.Dir(path)
	if parent != path && parent != "/" && parent != "." {
		return isGitRepository(parent)
	}
	
	return false
}

func findGitDir(path string) string {
	gitPath := filepath.Join(path, ".git")
	if info, err := os.Stat(gitPath); err == nil {
		if info.IsDir() {
			return gitPath
		}
		
		content, err := os.ReadFile(gitPath)
		if err == nil && strings.HasPrefix(string(content), "gitdir:") {
			gitDir := strings.TrimSpace(strings.TrimPrefix(string(content), "gitdir:"))
			if !filepath.IsAbs(gitDir) {
				gitDir = filepath.Join(path, gitDir)
			}
			return gitDir
		}
	}
	
	parent := filepath.Dir(path)
	if parent != path && parent != "/" && parent != "." {
		return findGitDir(parent)
	}
	
	return ""
}