package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zengjie/try/core"
	"github.com/zengjie/try/ui"
)

func RunInteractiveSelector(initialQuery string) error {
	if err := core.EnsureTryDirectory(); err != nil {
		return fmt.Errorf("failed to ensure try directory: %w", err)
	}

	// Check if we're in an interactive terminal
	if !isInteractive() {
		return fmt.Errorf("not running in an interactive terminal")
	}

	m := ui.NewModel()
	m.SetQuery(initialQuery)
	if err := m.LoadDirectories(); err != nil {
		return fmt.Errorf("failed to load directories: %w", err)
	}
	
	// Create program with input/output options
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
	)
	
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run selector: %w", err)
	}
	
	// The program will handle output through messages
	return nil
}

func CreateNewDirectory(name string) error {
	path, err := core.CreateDirectory(name)
	if err != nil {
		return err
	}
	
	// Write to .try_cd file for shell integration
	home, _ := os.UserHomeDir()
	cdFile := filepath.Join(home, ".try_cd")
	os.WriteFile(cdFile, []byte(path), 0644)
	
	// Also print to stdout for backward compatibility
	fmt.Println(path)
	return nil
}

func isInteractive() bool {
	if os.Getenv("TERM") == "" {
		return false
	}
	
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}