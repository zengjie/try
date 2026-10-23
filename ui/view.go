package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
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

	statusMsgStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
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
	// Show help view if requested
	if m.showHelp {
		return m.renderHelp()
	}

	var output strings.Builder

	// Title bar
	title := renderTitleBar(m.width)
	output.WriteString(title)
	output.WriteString("\n")

	// Status message banner (errors and feedback)
	if m.statusMessage != "" {
		if m.statusIsError {
			output.WriteString(errorStyle.Render(" "+m.statusMessage))
		} else {
			output.WriteString(statusMsgStyle.Render(" "+m.statusMessage))
		}
		output.WriteString("\n")
	}

	// Clone in-progress indicator
	if m.cloneInProgress {
		output.WriteString(highlightStyle.Render(" Cloning repository... please wait"))
		output.WriteString("\n")
	}

	// Search box (always show our custom search)
	searchBox := renderSearchBox(m.query, m.IsCreating(), m.explicitCreating, m.width)
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
		deleteSection := deleteWarningStyle.Render(fmt.Sprintf("Delete '%s'?", dir.Name)) + "\n" +
			dimStyle.Render("Type 'yes' or directory name to confirm: ") + highlightStyle.Render(m.deleteInput+cursorChar) + "\n" +
			helpStyle.Render("Press ESC to cancel")
		output.WriteString(deleteSection)
		output.WriteString("\n")
	}

	// Git init confirmation (if active)
	if m.initializingGit && m.gitInitConfirm {
		if selected := m.GetSelected(); selected != nil && !selected.IsCreateNew {
			gitInitSection := highlightStyle.Render(fmt.Sprintf("🔧 Initialize Git repository in '%s'?", selected.Name)) + "\n" +
				dimStyle.Render("Press Y to confirm or N/ESC to cancel")
			output.WriteString(gitInitSection)
			output.WriteString("\n")
		}
	}

	// Creation preview (if in explicit creation mode)
	if m.explicitCreating && m.query != "" {
		creationPreview := renderCreationPreview(m.query)
		output.WriteString(creationPreview)
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

	// Content preview of the highlighted directory
	if preview := m.SelectedPreview(); preview != "" {
		output.WriteString(renderPreviewLine(preview, m.width))
		output.WriteString("\n")
	}

	// Status bar with pagination
	page := m.list.Paginator.Page + 1
	totalPages := m.list.Paginator.TotalPages
	statusText := renderStatusBar(m.list.Index()+1, len(m.filteredDirs), m.query, page, totalPages)
	output.WriteString(statusText)
	output.WriteString("\n")

	// Help bar
	helpText := renderHelpBar()
	output.WriteString(helpText)

	return output.String()
}

const cursorChar = "█"

func renderSearchBox(query string, isCreating bool, explicitCreating bool, width int) string {
	if explicitCreating && query != "" {
		content := fmt.Sprintf("Creating: %s%s", query, cursorChar)
		box := searchBoxStyle.Width(width - 2).Render(content)
		return box
	}

	if query == "" {
		content := "Type to search..." + dimStyle.Render(cursorChar)
		box := searchBoxStyle.Width(width - 2).Render(content)
		return box
	}

	content := query + cursorChar
	if isCreating {
		content += dimStyle.Render(" (new)")
	}

	box := searchBoxStyle.Width(width - 2).Render(content)
	return box
}

func renderInputPrompt(title, prompt, input string) string {
	titleLine := highlightStyle.Render(title)
	inputLine := dimStyle.Render(prompt) + " " + highlightStyle.Render(input+cursorChar)
	helpLine := helpStyle.Render("Press Enter to confirm, ESC to cancel")
	return fmt.Sprintf("%s\n%s\n%s", titleLine, inputLine, helpLine)
}

func renderEmptyState(query string) string {
	if query == "" {
		return dimStyle.Render("  No directories yet. Start typing to create one!")
	}
	return dimStyle.Render(fmt.Sprintf("  ✨ Press Enter to create '%s'", core.GenerateDatedName(query)))
}

func renderCreationPreview(query string) string {
	if query == "" {
		return ""
	}
	
	actualName := core.GenerateDatedName(query)
	
	title := highlightStyle.Render("📁 Create New Directory:")
	input := fmt.Sprintf("Name: %s", highlightStyle.Render(query))
	preview := fmt.Sprintf("Will create: %s", highlightStyle.Render(actualName))
	help := dimStyle.Render("Press Enter to create, ESC to cancel")
	
	return fmt.Sprintf("%s\n%s\n%s\n%s", title, input, preview, help)
}

// renderPreviewLine renders the one-line content hint for the highlighted
// directory. Truncation is display-width aware so CJK/emoji README lines don't
// break the layout.
func renderPreviewLine(preview string, width int) string {
	const prefix = " ▸ "
	maxWidth := width - lipgloss.Width(prefix) - 1
	if maxWidth < 1 {
		return ""
	}
	preview = runewidth.Truncate(preview, maxWidth, "…")
	return dimStyle.Render(prefix + preview)
}

func renderStatusBar(current, total int, query string, page, totalPages int) string {
	if total == 0 {
		return statusBarStyle.Render(" Ready to create new directory")
	}

	status := fmt.Sprintf(" %d/%d", current, total)
	if query != "" {
		status += fmt.Sprintf(" matching '%s'", query)
	}
	if totalPages > 1 {
		status += fmt.Sprintf("  (page %d/%d)", page, totalPages)
	}
	return statusBarStyle.Render(status)
}

func renderHelpBar() string {
	shortcuts := []string{
		"^N/^P Navigate",
		"⏎ Select",
		"^O Create",
		"^X Delete",
		"^G Clone",
		"^T Worktree",
		"^E Finder",
		"? Help",
		"^C Quit",
	}
	return helpStyle.Render(" " + strings.Join(shortcuts, " | "))
}

func renderTitleBar(width int) string {
	title := "🚀 Try - Fresh Directories for Every Vibe"
	// Terminal narrower than the title (or width not yet known): render as-is.
	// Otherwise let titleStyle center it; its Width includes the padding.
	if width < lipgloss.Width(titleStyle.Render(title)) {
		return titleStyle.Render(title)
	}
	return titleStyle.Width(width).Render(title)
}

func renderTableHeader(width int) string {
	nameColWidth := NameColumnWidth(width)
	header := fmt.Sprintf("  %-*s %-*s %-*s",
		nameColWidth, "Name",
		TagsColumnWidth, "Tags",
		ModifiedColumnWidth, "Modified")

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		Underline(true)

	return headerStyle.Render(header)
}

func (m Model) renderHelp() string {
	return m.RenderInteractiveHelp()
}