package ui

import (
	"fmt"
	"strings"
)

// HelpContent centralizes all help information
type HelpContent struct {
	Navigation []HelpItem
	Actions    []HelpItem
	Search     []HelpItem
	Other      []HelpItem
	Tips       []string
}

type HelpItem struct {
	Key         string
	Description string
}

// GetHelpContent returns the centralized help information
func GetHelpContent() HelpContent {
	return HelpContent{
		Navigation: []HelpItem{
			{"^N/^J/↓", "Move down"},
			{"^P/^K/↑", "Move up"},
			{"^F/→", "Page down"},
			{"^B/←", "Page up"},
			{"Home/End", "Go to top/bottom"},
		},
		Actions: []HelpItem{
			{"Enter", "Select/Create directory"},
			{"ESC", "Clear search/Cancel"},
			{"Tab", "Auto-complete search"},
			{"Ctrl+O", "Create new directory"},
			{"Ctrl+X", "Delete directory"},
			{"Ctrl+T", "Create worktree (git repos)"},
			{"Ctrl+G", "Clone git repository"},
			{"Ctrl+R", "Initialize git repository"},
			{"Ctrl+E", "Open in Finder"},
		},
		Search: []HelpItem{
			{"Type", "Filter directories"},
			{"Backspace", "Delete character"},
			{"Ctrl+W", "Delete word"},
			{"Ctrl+U", "Clear search"},
		},
		Other: []HelpItem{
			{"?", "Show/hide help"},
			{"Ctrl+C", "Quit"},
		},
		Tips: []string{
			"Results are ranked by how often and recently you open them (frecency)",
			"The 'Nx' marker shows how many times you've opened a directory",
			"When searching, text relevance is blended with that usage signal",
			"The preview line shows the highlighted directory's README, last commit, or contents",
			"New directories get today's date prefix automatically",
			"Git repositories and worktrees have special indicators",
		},
	}
}

// RenderInteractiveHelp generates the formatted help screen for interactive mode
func (m Model) RenderInteractiveHelp() string {
	help := GetHelpContent()
	
	var sections []string
	sections = append(sections, "🚀 Try - Keyboard Shortcuts")
	sections = append(sections, "")
	
	// Navigation section
	sections = append(sections, "Navigation:")
	for _, item := range help.Navigation {
		sections = append(sections, fmt.Sprintf("  %-12s %s", item.Key, item.Description))
	}
	sections = append(sections, "")
	
	// Actions section
	sections = append(sections, "Actions:")
	for _, item := range help.Actions {
		sections = append(sections, fmt.Sprintf("  %-12s %s", item.Key, item.Description))
	}
	sections = append(sections, "")
	
	// Search section
	sections = append(sections, "Search:")
	for _, item := range help.Search {
		sections = append(sections, fmt.Sprintf("  %-12s %s", item.Key, item.Description))
	}
	sections = append(sections, "")
	
	// Other section
	sections = append(sections, "Other:")
	for _, item := range help.Other {
		sections = append(sections, fmt.Sprintf("  %-12s %s", item.Key, item.Description))
	}
	sections = append(sections, "")
	
	// Tips section
	sections = append(sections, "Tips:")
	for _, tip := range help.Tips {
		sections = append(sections, fmt.Sprintf("  • %s", tip))
	}
	
	return helpViewStyle.Render(strings.Join(sections, "\n"))
}

// RenderCLIKeyboardShortcuts generates the keyboard shortcuts section for CLI help
func RenderCLIKeyboardShortcuts() string {
	help := GetHelpContent()
	
	var lines []string
	lines = append(lines, "KEYBOARD SHORTCUTS (Interactive Mode):")
	
	// Combine all shortcuts into a flat list for CLI
	allItems := []HelpItem{}
	allItems = append(allItems, help.Navigation...)
	allItems = append(allItems, help.Actions...)
	allItems = append(allItems, help.Search...)
	allItems = append(allItems, help.Other...)
	
	// Format for CLI (with consistent spacing)
	for _, item := range allItems {
		lines = append(lines, fmt.Sprintf("    %-15s %s", item.Key, item.Description))
	}
	
	return strings.Join(lines, "\n")
}