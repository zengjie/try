package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zengjie/try/cmd"
	"github.com/zengjie/try/shell"
)

func main() {
	if len(os.Args) < 2 {
		if err := cmd.RunInteractiveSelector(""); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	command := os.Args[1]

	switch command {
	case "--help", "-h", "help":
		showHelp()
		return
	case "init":
		var tryPath string
		if len(os.Args) > 2 {
			tryPath = os.Args[2]
		} else {
			home, _ := os.UserHomeDir()
			tryPath = filepath.Join(home, "src", "tries")
		}
		
		shellName := os.Getenv("SHELL")
		if shellName == "" {
			shellName = "bash"
		}
		shellName = filepath.Base(shellName)
		
		script := shell.GenerateShellScript(shellName, tryPath)
		fmt.Print(script)

	case "new":
		name := ""
		if len(os.Args) > 2 {
			name = strings.Join(os.Args[2:], " ")
		}
		if err := cmd.CreateNewDirectory(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case ".":
		name := ""
		if len(os.Args) > 2 {
			name = strings.Join(os.Args[2:], " ")
		}
		if err := cmd.CreateWorktree(".", name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "clone":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: git URL required\n")
			os.Exit(1)
		}
		if err := cmd.CloneRepository(os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "worktree":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: repository path required\n")
			os.Exit(1)
		}
		name := ""
		if len(os.Args) > 3 {
			name = strings.Join(os.Args[3:], " ")
		}
		if err := cmd.CreateWorktree(os.Args[2], name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	default:
		if strings.HasPrefix(command, "http://") || strings.HasPrefix(command, "https://") || 
		   strings.HasPrefix(command, "git@") || strings.HasSuffix(command, ".git") {
			if err := cmd.CloneRepository(command); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else {
			query := strings.Join(os.Args[1:], " ")
			if err := cmd.RunInteractiveSelector(query); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
	}
}

func showHelp() {
	help := `Try - Fresh Directories for Every Vibe

USAGE:
    try                     Open interactive directory selector
    try [query]             Search for directories matching query
    try new [name]          Create new dated directory
    try . [name]            Create worktree for current repository
    try clone <url>         Clone git repository with dated name
    try worktree <path>     Create worktree from repository
    try init [path]         Generate shell integration script
    try --help              Show this help message

SHORTCUTS:
    try <git-url>           Automatically clone if URL detected

KEYBOARD SHORTCUTS (Interactive Mode):
    ↑/↓, Ctrl-P/N          Navigate up/down
    Ctrl-J/K               Vim-style navigation
    Enter                  Select directory or create new
    Backspace              Delete character
    Ctrl-D                 Delete directory (with confirmation)
    ESC                    Cancel operation

ENVIRONMENT:
    TRY_PATH               Override default directory location
                          (default: ~/src/tries)

EXAMPLES:
    try                    # Open interactive selector
    try redis              # Search for "redis" directories
    try new experiment     # Create ~/src/tries/2025-08-30-experiment
    try clone https://github.com/user/repo.git
    try . feature-branch   # Create worktree from current repo

For more information, visit: https://github.com/zengjie/try`

	fmt.Println(help)
}