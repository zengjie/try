package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zengjie/try/core"
)

var (
	// Main theme colors
	primaryColor   = lipgloss.Color("#7C3AED") // Modern purple
	secondaryColor = lipgloss.Color("#10B981") // Soft green
	accentColor    = lipgloss.Color("#F59E0B") // Warm amber
	dangerColor    = lipgloss.Color("#EF4444") // Soft red
	
	bgColor    = lipgloss.Color("#1E1E2E") // Dark background
	fgColor    = lipgloss.Color("#CDD6F4") // Light foreground
	dimColor   = lipgloss.Color("#6C7086") // Muted text
	
	// Title bar
	titleStyle = lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Align(lipgloss.Center).
			Padding(0, 1)
	
	// Search box (for when we're not filtering)
	searchBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1)
	
	// Help view
	helpViewStyle = lipgloss.NewStyle().
			Padding(2, 4).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor)
	
	// Status and help bars
	statusBarStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("#E5E7EB")).
			Padding(0, 1)
	
	helpStyle = lipgloss.NewStyle().
			Foreground(dimColor)
	
	// Error and warning styles
	errorStyle = lipgloss.NewStyle().
			Foreground(dangerColor).
			Bold(true)
	
	deleteWarningStyle = lipgloss.NewStyle().
			Foreground(dangerColor).
			Bold(true)
	
	// Other styles
	dimStyle = lipgloss.NewStyle().
			Foreground(dimColor)
	
	highlightStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)
)

func (m Model) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	// Show help view if requested
	if m.showHelp {
		return m.renderHelp()
	}

	var output strings.Builder

	// Title bar
	title := renderTitleBar(m.width)
	output.WriteString(title)
	output.WriteString("\n")

	// Search box (always show our custom search)
	searchBox := renderSearchBox(m.query, m.IsCreating(), m.width)
	output.WriteString(searchBox)
	output.WriteString("\n")

	// Worktree input (if active)
	if m.creatingWorktree {
		prompt := renderInputPrompt("🌿 Create Worktree", "Enter worktree name:", m.worktreeInput)
		output.WriteString(prompt)
		output.WriteString("\n")
	}

	// Clone input (if active)
	if m.cloning {
		prompt := renderInputPrompt("📦 Clone Repository", "Enter Git URL:", m.cloneInput)
		output.WriteString(prompt)
		output.WriteString("\n")
	}

	// Delete confirmation (if active)
	if m.deleting && m.deleteConfirm {
		dir := m.filteredDirs[m.selectedForDelete]
		deleteSection := deleteWarningStyle.Render(fmt.Sprintf("⚠️  Delete '%s'?", dir.Name)) + "\n" +
			dimStyle.Render("Type 'yes' or directory name to confirm: ") + highlightStyle.Render(m.deleteInput) + "\n" +
			helpStyle.Render("Press ESC to cancel")
		output.WriteString(deleteSection)
		output.WriteString("\n")
	}

	// Main list view
	if len(m.filteredDirs) == 0 {
		emptyMsg := renderEmptyState(m.query)
		output.WriteString(emptyMsg)
		output.WriteString("\n")
	} else {
		// Add table header
		header := renderTableHeader(m.width)
		output.WriteString(header)
		output.WriteString("\n")
		
		// Get list view and strip any leading empty lines
		listView := m.list.View()
		listView = strings.TrimLeft(listView, "\n")
		output.WriteString(listView)
		output.WriteString("\n")
	}

	// Status bar
	statusText := renderStatusBar(m.list.Index()+1, len(m.filteredDirs), m.query)
	output.WriteString(statusText)
	output.WriteString("\n")

	// Help bar
	helpText := renderHelpBar()
	output.WriteString(helpText)

	return output.String()
}

func renderSearchBox(query string, isCreating bool, width int) string {
	content := fmt.Sprintf("🔍 %s", query)
	if query == "" {
		content = "🔍 Type to search with fuzzy matching..."
	}
	if isCreating {
		content += dimStyle.Render(" (new)")
	}
	
	box := searchBoxStyle.Width(width - 2).Render(content)
	return box
}

func renderInputPrompt(title, prompt, input string) string {
	titleLine := highlightStyle.Render(title)
	inputLine := dimStyle.Render(prompt) + " " + highlightStyle.Render(input)
	helpLine := helpStyle.Render("Press Enter to confirm, ESC to cancel")
	return fmt.Sprintf("%s\n%s\n%s", titleLine, inputLine, helpLine)
}

func renderEmptyState(query string) string {
	if query == "" {
		return dimStyle.Render("  No directories yet. Start typing to create one!")
	}
	return dimStyle.Render(fmt.Sprintf("  ✨ Press Enter to create '%s'", core.GenerateDatedName(query)))
}

func renderStatusBar(current, total int, query string) string {
	if total == 0 {
		return statusBarStyle.Render(" Ready to create new directory")
	}
	
	status := fmt.Sprintf(" %d/%d", current, total)
	if query != "" {
		status += fmt.Sprintf(" matching '%s'", query)
	}
	return statusBarStyle.Render(status)
}

func renderHelpBar() string {
	shortcuts := []string{
		"↑↓←→ Navigate",
		"⏎ Select",
		"^W Worktree",
		"^G Clone",
		"^D Delete",
		"? Help",
		"^C Quit",
	}
	return helpStyle.Render(" " + strings.Join(shortcuts, " │ "))
}

func renderTitleBar(width int) string {
	title := " 🚀 Try - Fresh Directories for Every Vibe "
	padding := (width - lipgloss.Width(title)) / 2
	if padding < 0 {
		padding = 0
	}
	paddedTitle := strings.Repeat(" ", padding) + title + strings.Repeat(" ", width-padding-lipgloss.Width(title))
	return titleStyle.Render(paddedTitle)
}

func renderTableHeader(width int) string {
	// Fixed column widths matching the delegate
	// Using tab separation for better alignment
	header := fmt.Sprintf("  %-*s %-*s %-*s", 
		NameColumnWidth, "Name", 
		TagsColumnWidth, "Tags", 
		ModifiedColumnWidth, "Modified")
	
	// Style the header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		Underline(true)
	
	return headerStyle.Render(header)
}

func (m Model) renderHelp() string {
	return m.RenderInteractiveHelp()
}