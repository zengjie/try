package ui

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zengjie/try/core"
)

// Fixed column widths for tags and modified; name column flexes
const (
	TagsColumnWidth     = 15
	ModifiedColumnWidth = 15
	MinNameColumnWidth  = 20
	FixedColumnsWidth   = TagsColumnWidth + ModifiedColumnWidth + 6 // 6 for prefix + spacing
)

// NameColumnWidth calculates the name column width based on terminal width
func NameColumnWidth(termWidth int) int {
	w := termWidth - FixedColumnsWidth
	if w < MinNameColumnWidth {
		return MinNameColumnWidth
	}
	return w
}

// DirectoryItem implements list.Item interface
type DirectoryItem struct {
	core.Directory
	IsCreateNew bool // Special flag to indicate this is a "create new" option
	CreateQuery string // Query to create if this is a create new item
}

func (i DirectoryItem) FilterValue() string {
	return i.Name
}

func (i DirectoryItem) Title() string {
	return i.Name
}

func (i DirectoryItem) Description() string {
	tags := []string{}
	
	if i.IsGitRepo {
		tags = append(tags, "🔷 git")
	}
	if i.IsWorktree {
		tags = append(tags, "🌿 worktree")
	}
	
	// Add modified time
	age := core.GetRelativeAge(i.ModifiedTime)
	
	if len(tags) > 0 {
		return fmt.Sprintf("%s • %s", strings.Join(tags, " "), age)
	}
	return age
}

// usageTagsContent builds the content of the "Tags" column. It leads with a
// frecency marker — the visit count, e.g. "12x", for directories opened through
// try before — so frequently-used directories visibly explain why they sort
// high, then appends git/worktree markers. Returns "-" when there is nothing to
// show. Output is ASCII so its byte length equals its display width.
func usageTagsContent(accessCount int, isGitRepo, isWorktree bool) string {
	parts := ""
	if accessCount > 0 {
		parts = fmt.Sprintf("%dx ", accessCount)
	}
	if isGitRepo {
		parts += "git "
	}
	if isWorktree {
		parts += "worktree "
	}
	if parts == "" {
		return "-"
	}
	return strings.TrimSpace(parts)
}

// Custom item delegate for rendering
type itemDelegate struct{
	maxWidth int
}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(DirectoryItem)
	if !ok {
		return
	}

	nameColWidth := NameColumnWidth(d.maxWidth)

	// Build the row components
	prefix := "  "
	if index == m.Index() {
		prefix = "▶ "
	}

	// Special rendering for "create new" items
	if i.IsCreateNew {
		// Format name with special styling
		name := i.Name
		if len(name) > nameColWidth {
			name = name[:nameColWidth-3] + "..."
		}
		for len(name) < nameColWidth {
			name = name + " "
		}

		// Empty tags and age for create new items
		tags := strings.Repeat(" ", TagsColumnWidth)
		age := strings.Repeat(" ", ModifiedColumnWidth)

		// Build the complete row
		row := fmt.Sprintf("%s%s %s %s", prefix, name, tags, age)

		// Apply style with special color for create new
		createItemStyle := lipgloss.NewStyle().
			PaddingLeft(0).
			Foreground(lipgloss.Color("#A9B665")).
			Italic(true)

		selectedCreateItemStyle := lipgloss.NewStyle().
			PaddingLeft(0).
			Foreground(lipgloss.Color("#A9B665")).
			Background(lipgloss.Color("#45475A")).
			Italic(true).
			Bold(true)

		if index == m.Index() {
			fmt.Fprint(w, selectedCreateItemStyle.Render(row))
		} else {
			fmt.Fprint(w, createItemStyle.Render(row))
		}
		return
	}

	// Normal directory rendering
	name := i.Name
	if len(name) > nameColWidth {
		name = name[:nameColWidth-3] + "..."
	}
	for len(name) < nameColWidth {
		name = name + " "
	}
	
	// Format the tags/usage column: a leading frecency marker (visit count)
	// followed by git/worktree markers.
	tags := usageTagsContent(i.AccessCount, i.IsGitRepo, i.IsWorktree)
	// Truncate then pad to the column width. Content is ASCII, so byte length
	// equals display width here.
	if len(tags) > TagsColumnWidth {
		tags = tags[:TagsColumnWidth]
	}
	for len(tags) < TagsColumnWidth {
		tags = tags + " "
	}
	
	// Format modified time
	age := core.GetRelativeAge(i.ModifiedTime)
	if len(age) > ModifiedColumnWidth {
		age = age[:ModifiedColumnWidth-3] + "..."
	}
	// Pad age to exactly ModifiedColumnWidth characters
	for len(age) < ModifiedColumnWidth {
		age = age + " "
	}
	
	// Build the complete row
	row := fmt.Sprintf("%s%s %s %s", prefix, name, tags, age)
	
	// Apply style
	if index == m.Index() {
		fmt.Fprint(w, selectedItemStyle.Render(row))
	} else {
		fmt.Fprint(w, itemStyle.Render(row))
	}
}

// Styles for list items
var (
	itemStyle = lipgloss.NewStyle().
		PaddingLeft(0).
		Foreground(lipgloss.Color("#CDD6F4"))
		
	selectedItemStyle = lipgloss.NewStyle().
		PaddingLeft(0).
		Foreground(lipgloss.Color("#F9E2AF")).
		Background(lipgloss.Color("#45475A"))
)

type Model struct {
	list              list.Model
	directories       []core.Directory
	filteredDirs      []core.Directory
	query             string
	height            int
	width             int
	creating          bool
	deleting          bool
	deleteConfirm     bool
	deleteInput       string
	selectedForDelete int
	showHelp          bool
	cloning           bool
	cloneInput        string
	cloneInProgress   bool
	creatingWorktree  bool
	worktreeInput     string
	worktreeRepo      string
	initializingGit   bool
	gitInitConfirm    bool
	explicitCreating  bool
	statusMessage     string
	statusIsError     bool
	previewCache      map[string]string // path -> one-line content preview
}

func NewModel() Model {
	// Create items list
	items := []list.Item{}
	
	// Create the list with custom delegate
	del := itemDelegate{maxWidth: 80}
	l := list.New(items, del, 0, 0)
	l.SetShowTitle(false) // Disable title completely
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false) // Disable built-in filtering - we use our own scoring system
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle() // Empty style
	
	// Remove pagination dots
	l.SetShowPagination(false)
	
	return Model{
		list:              l,
		directories:       []core.Directory{},
		filteredDirs:      []core.Directory{},
		query:             "",
		height:            24,
		width:             80,
		creating:          false,
		deleting:          false,
		deleteConfirm:     false,
		deleteInput:       "",
		selectedForDelete: -1,
		showHelp:          false,
		cloning:           false,
		cloneInput:        "",
		cloneInProgress:   false,
		creatingWorktree:  false,
		worktreeInput:     "",
		worktreeRepo:      "",
		initializingGit:   false,
		gitInitConfirm:    false,
		explicitCreating:  false,
		statusMessage:     "",
		statusIsError:     false,
		previewCache:      map[string]string{},
	}
}

func (m *Model) LoadDirectories() error {
	dirs, err := core.ScanDirectories()
	if err != nil {
		return err
	}

	m.directories = dirs
	// Directory contents may have changed (clone, git init, delete), so cached
	// previews are stale.
	m.previewCache = map[string]string{}
	m.updateFiltered()
	return nil
}

func (m *Model) updateFiltered() {
	// Filter and score directories using the new scoring system
	m.filteredDirs = core.FilterAndScoreDirectories(m.directories, m.query)

	// With no query, list directories by last access (newest first). With a
	// query, order by text relevance blended with usage.
	if strings.TrimSpace(m.query) == "" {
		core.SortDirectoriesByRecency(m.filteredDirs)
	} else {
		core.SortDirectoriesByScore(m.filteredDirs)
	}

	// Convert to list items
	items := make([]list.Item, len(m.filteredDirs))
	for i, dir := range m.filteredDirs {
		items[i] = DirectoryItem{Directory: dir, IsCreateNew: false}
	}
	
	// Add "Create new directory" option if query doesn't exactly match any directory
	if m.query != "" && !m.HasExactMatch() {
		createItem := DirectoryItem{
			Directory: core.Directory{
				Name: fmt.Sprintf("✨ Create new: %s", core.GenerateDatedName(m.query)),
				Path: "", // Empty path indicates this is a create option
			},
			IsCreateNew: true,
			CreateQuery: m.query,
		}
		items = append(items, createItem)
	}
	
	m.list.SetItems(items)
}

func (m *Model) SetQuery(q string) {
	m.query = q
	m.explicitCreating = false  // Reset explicit creation when changing query
	m.updateFiltered()
}

func (m *Model) AppendToQuery(text string) {
	if text == "" {
		return
	}
	m.query += text
	m.explicitCreating = false  // Reset explicit creation when typing
	m.updateFiltered()
}

func (m *Model) DeleteFromQuery() {
	if len(m.query) > 0 {
		_, size := utf8.DecodeLastRuneInString(m.query)
		m.query = m.query[:len(m.query)-size]
		m.explicitCreating = false  // Reset explicit creation when deleting
		m.updateFiltered()
	}
}

func (m *Model) DeleteWordFromQuery() {
	if len(m.query) == 0 {
		return
	}
	// Skip trailing spaces
	i := len(m.query) - 1
	for i >= 0 && m.query[i] == ' ' {
		i--
	}
	// Delete back to previous space or beginning
	for i >= 0 && m.query[i] != ' ' {
		i--
	}
	m.query = m.query[:i+1]
	m.explicitCreating = false
	m.updateFiltered()
}

func (m *Model) GetSelected() *DirectoryItem {
	selected := m.list.SelectedItem()
	if selected != nil {
		if item, ok := selected.(DirectoryItem); ok {
			return &item
		}
	}
	return nil
}

// SelectedPreview returns a one-line content hint for the highlighted item:
// the directory's README first line, last commit, or entry summary — or, for
// the "create new" item, the full path that would be created. Previews are
// computed lazily and cached per path so navigating stays fast.
func (m *Model) SelectedPreview() string {
	selected := m.GetSelected()
	if selected == nil {
		return ""
	}
	if selected.IsCreateNew {
		return "will create " + filepath.Join(core.GetTryPath(), core.GenerateDatedName(selected.CreateQuery))
	}
	if cached, ok := m.previewCache[selected.Path]; ok {
		return cached
	}
	preview := core.DirectoryPreview(selected.Path, selected.IsGitRepo || selected.IsWorktree)
	m.previewCache[selected.Path] = preview
	return preview
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	
	// Update list with new delegate that has the correct width
	del := itemDelegate{maxWidth: width}
	m.list.SetDelegate(del)
	m.list.SetWidth(width)
	
	// Calculate available height for list
	// Account for: title (1), search box (3), header (1), preview (1),
	// status (1), help (1)
	availableHeight := height - 8
	if availableHeight < 5 {
		availableHeight = 5
	}
	m.list.SetHeight(availableHeight)
}

func (m *Model) HasExactMatch() bool {
	if m.query == "" {
		return false
	}
	
	queryLower := strings.ToLower(m.query)
	for _, dir := range m.filteredDirs {
		dirNameLower := strings.ToLower(core.ExtractNameFromDirectory(dir.Name))
		if dirNameLower == queryLower {
			return true
		}
	}
	return false
}

func (m *Model) IsCreating() bool {
	// This method is now less relevant since creation is handled via list selection
	// Keep it for backward compatibility with explicit creation (Ctrl+N)
	return m.explicitCreating
}

func (m *Model) StartDelete() {
	if selectedItem := m.GetSelected(); selectedItem != nil && !selectedItem.IsCreateNew {
		// Find the index in filteredDirs
		for i, dir := range m.filteredDirs {
			if dir.Path == selectedItem.Path {
				m.deleting = true
				m.selectedForDelete = i
				break
			}
		}
	}
}

func (m *Model) CancelDelete() {
	m.deleting = false
	m.deleteConfirm = false
	m.deleteInput = ""
	m.selectedForDelete = -1
}

func (m *Model) ConfirmDelete() error {
	if m.selectedForDelete >= 0 && m.selectedForDelete < len(m.filteredDirs) {
		dir := m.filteredDirs[m.selectedForDelete]
		if err := core.DeleteDirectory(dir.Path); err != nil {
			return err
		}
		
		// Remove from directories
		newDirs := []core.Directory{}
		for _, d := range m.directories {
			if d.Path != dir.Path {
				newDirs = append(newDirs, d)
			}
		}
		m.directories = newDirs
		
		m.updateFiltered()
		m.CancelDelete()
	}
	return nil
}

func (m *Model) StartGitInit() {
	if selectedItem := m.GetSelected(); selectedItem != nil && !selectedItem.IsCreateNew {
		// Only allow Git init on regular directories (not already Git repos or worktrees)
		if !selectedItem.IsGitRepo && !selectedItem.IsWorktree {
			m.initializingGit = true
			m.gitInitConfirm = true
		}
	}
}

func (m *Model) CancelGitInit() {
	m.initializingGit = false
	m.gitInitConfirm = false
}

func (m *Model) StartExplicitCreate() {
	if m.query != "" {
		m.explicitCreating = true
	}
}

func (m *Model) CancelExplicitCreate() {
	m.explicitCreating = false
}