package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func GenerateShellScript(shellName string, tryPath string) string {
	shellName = filepath.Base(shellName)
	
	switch {
	case strings.Contains(shellName, "zsh"):
		return generateZshScript(tryPath)
	case strings.Contains(shellName, "fish"):
		return generateFishScript(tryPath)
	default:
		return generateBashScript(tryPath)
	}
}

func generateBashScript(tryPath string) string {
	return fmt.Sprintf(`# Try shell integration for Bash
export TRY_PATH="%s"
export TRY_BINARY="%s"

try() {
    # Run the try binary with its output going to the terminal
    "${TRY_BINARY}" "$@"
    local exit_code=$?
    
    # If successful, check if a .try_cd file was created with a path to cd to
    if [ $exit_code -eq 0 ] && [ -f "$HOME/.try_cd" ]; then
        local dir=$(cat "$HOME/.try_cd")
        rm -f "$HOME/.try_cd"
        if [ -d "$dir" ]; then
            cd "$dir"
        fi
    fi
    
    return $exit_code
}
`, tryPath, getTryBinaryPath())
}

func generateZshScript(tryPath string) string {
	return fmt.Sprintf(`# Try shell integration for Zsh
export TRY_PATH="%s"
export TRY_BINARY="%s"

try() {
    # Run the try binary with its output going to the terminal
    "${TRY_BINARY}" "$@"
    local exit_code=$?
    
    # If successful, check if a .try_cd file was created with a path to cd to
    if [ $exit_code -eq 0 ] && [ -f "$HOME/.try_cd" ]; then
        local dir=$(cat "$HOME/.try_cd")
        rm -f "$HOME/.try_cd"
        if [ -d "$dir" ]; then
            cd "$dir"
        fi
    fi
    
    return $exit_code
}
`, tryPath, getTryBinaryPath())
}

func generateFishScript(tryPath string) string {
	return fmt.Sprintf(`# Try shell integration for Fish
set -x TRY_PATH "%s"
set -x TRY_BINARY "%s"

function try
    # Run the try binary with its output going to the terminal
    $TRY_BINARY $argv
    set -l exit_code $status
    
    # If successful, check if a .try_cd file was created with a path to cd to
    if test $exit_code -eq 0; and test -f "$HOME/.try_cd"
        set -l dir (cat "$HOME/.try_cd")
        rm -f "$HOME/.try_cd"
        if test -d "$dir"
            cd "$dir"
        end
    end
    
    return $exit_code
end
`, tryPath, getTryBinaryPath())
}

func getTryBinaryPath() string {
	// Try to find the try binary in PATH, otherwise use the command name
	if execPath, err := os.Executable(); err == nil {
		return execPath
	}
	return "try"
}