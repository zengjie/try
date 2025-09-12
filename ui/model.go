package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zengjie/try/core"
)

// Column width constants for consistent layout
const (
	NameColumnWidth     = 50
	TagsColumnWidth     = 15
	ModifiedColumnWidth = 15
)

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

	// Build the row components
	prefix := "  "
	if index == m.Index() {
		prefix = "▶ "
	}
	
	// Special rendering for "create new" items
	if i.IsCreateNew {
		// Format name with special styling
		name := i.Name
		if len(name) > NameColumnWidth {
			name = name[:NameColumnWidth-3] + "..."
		}
		// Pad name to exactly NameColumnWidth characters
		for len(name) < NameColumnWidth {
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
	// Format name
	name := i.Name
	if len(name) > NameColumnWidth {
		name = name[:NameColumnWidth-3] + "..."
	}
	// Pad name to exactly NameColumnWidth characters
	for len(name) < NameColumnWidth {
		name = name + " "
	}
	
	// Format tags
	tags := ""
	if i.IsGitRepo {
		tags = "git "
	}
	if i.IsWorktree {
		tags = tags + "worktree "
	}
	if tags == "" {
		tags = "-"
	} else {
		tags = strings.TrimSpace(tags)
	}
	// Pad tags to exactly TagsColumnWidth characters (more space for unicode)
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
	creatingWorktree  bool
	worktreeInput     string
	worktreeRepo      string
	initializingGit   bool
	gitInitConfirm    bool
	explicitCreating  bool
	err               error
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
		creatingWorktree:  false,
		worktreeInput:     "",
		worktreeRepo:      "",
		initializingGit:   false,
		gitInitConfirm:    false,
		explicitCreating:  false,
		err:               nil,
	}
}

func (m *Model) LoadDirectories() error {
	dirs, err := core.ScanDirectories()
	if err != nil {
		return err
	}
	
	m.directories = dirs
	m.updateFiltered()
	return nil
}

func (m *Model) updateFiltered() {
	// Filter and score directories using the new scoring system
	m.filteredDirs = core.FilterAndScoreDirectories(m.directories, m.query)
	
	// Sort by score when there's a query, by time otherwise
	if m.query == "" {
		core.SortDirectoriesByTime(m.filteredDirs)
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

func (m *Model) AppendToQuery(ch rune) {
	m.query += string(ch)
	m.explicitCreating = false  // Reset explicit creation when typing
	m.updateFiltered()
}

func (m *Model) DeleteFromQuery() {
	if len(m.query) > 0 {
		m.query = m.query[:len(m.query)-1]
		m.explicitCreating = false  // Reset explicit creation when deleting
		m.updateFiltered()
	}
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

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	
	// Update list with new delegate that has the correct width
	del := itemDelegate{maxWidth: width}
	m.list.SetDelegate(del)
	m.list.SetWidth(width)
	
	// Calculate available height for list
	// Account for: title (1), search box (3), status (1), help (1), header (1)
	// The header is now combined with the list view
	availableHeight := height - 7
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