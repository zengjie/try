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

// DirectoryItem implements list.Item interface
type DirectoryItem struct {
	core.Directory
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
	
	// Format name (35 chars)
	name := i.Name
	if len(name) > 35 {
		name = name[:32] + "..."
	}
	// Pad name to exactly 35 characters
	for len(name) < 35 {
		name = name + " "
	}
	
	// Format tags (30 chars to account for emoji width)
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
	// Pad tags to exactly 30 characters (more space for unicode)
	for len(tags) < 30 {
		tags = tags + " "
	}
	
	// Format modified time (15 chars)
	age := core.GetRelativeAge(i.ModifiedTime)
	if len(age) > 15 {
		age = age[:12] + "..."
	}
	// Pad age to exactly 15 characters
	for len(age) < 15 {
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
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle() // Empty style
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))
	
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
	// Update filtered directories
	if m.query == "" {
		m.filteredDirs = m.directories
		core.ScoreDirectories(m.filteredDirs, "")
		core.SortDirectoriesByTime(m.filteredDirs)
	} else {
		m.filteredDirs = core.FilterDirectories(m.directories, m.query)
		core.ScoreDirectories(m.filteredDirs, m.query)
		core.SortDirectoriesByScore(m.filteredDirs)
	}
	
	// Convert to list items
	items := make([]list.Item, len(m.filteredDirs))
	for i, dir := range m.filteredDirs {
		items[i] = DirectoryItem{dir}
	}
	
	m.list.SetItems(items)
}

func (m *Model) SetQuery(q string) {
	m.query = q
	m.updateFiltered()
	
	// Update the list's filter state
	if q != "" {
		m.list.FilterInput.SetValue(q)
	}
}

func (m *Model) AppendToQuery(ch rune) {
	m.query += string(ch)
	m.updateFiltered()
}

func (m *Model) DeleteFromQuery() {
	if len(m.query) > 0 {
		m.query = m.query[:len(m.query)-1]
		m.updateFiltered()
	}
}

func (m *Model) GetSelected() *core.Directory {
	selected := m.list.SelectedItem()
	if selected != nil {
		if item, ok := selected.(DirectoryItem); ok {
			for i := range m.filteredDirs {
				if m.filteredDirs[i].Path == item.Path {
					return &m.filteredDirs[i]
				}
			}
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

func (m *Model) IsCreating() bool {
	return len(m.filteredDirs) == 0 && m.query != ""
}

func (m *Model) StartDelete() {
	selected := m.list.Index()
	if selected >= 0 && selected < len(m.filteredDirs) {
		m.deleting = true
		m.selectedForDelete = selected
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