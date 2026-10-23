package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zengjie/try/core"
)

type DirectorySelectedMsg struct {
	Path string
}

type DirectoryCreatedMsg struct {
	Path string
}

type clearStatusMsg struct{}

type cloneFinishedMsg struct {
	err error
}

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

func (m *Model) setStatus(message string, isError bool) tea.Cmd {
	m.statusMessage = message
	m.statusIsError = isError
	return clearStatusAfter(3 * time.Second)
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tea.WindowSize(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If help is showing, dismiss on specific keys only
		if m.showHelp {
			switch msg.String() {
			case "esc", "enter", "?", "ctrl+c", "q":
				m.showHelp = false
				return m, nil
			}
			return m, nil
		}

		// Handle special input modes
		if m.creatingWorktree {
			switch msg.String() {
			case "enter":
				if err := createWorktreeFromPath(m.worktreeRepo, m.worktreeInput); err != nil {
					statusCmd := m.setStatus(fmt.Sprintf("Worktree failed: %v", err), true)
					m.creatingWorktree = false
					m.worktreeInput = ""
					m.worktreeRepo = ""
					return m, statusCmd
				} else {
					m.creatingWorktree = false
					m.worktreeInput = ""
					m.worktreeRepo = ""
					m.LoadDirectories()
				}
				return m, nil
			case "esc":
				m.creatingWorktree = false
				m.worktreeInput = ""
				m.worktreeRepo = ""
				return m, nil
			case "backspace":
				m.worktreeInput = trimLastRune(m.worktreeInput)
				return m, nil
			default:
				m.worktreeInput += insertableText(msg)
				return m, nil
			}
		}

		if m.cloning && !m.cloneInProgress {
			switch msg.String() {
			case "enter":
				if url := strings.TrimSpace(m.cloneInput); url != "" {
					m.cloneInProgress = true
					statusCmd := m.setStatus("Cloning repository...", false)
					return m, tea.Batch(statusCmd, runClone(url))
				}
				return m, nil
			case "esc":
				m.cloning = false
				m.cloneInput = ""
				return m, nil
			case "backspace":
				m.cloneInput = trimLastRune(m.cloneInput)
				return m, nil
			default:
				m.cloneInput += insertableText(msg)
				return m, nil
			}
		}

		if m.deleting && m.deleteConfirm {
			switch msg.String() {
			case "enter":
				dir := m.filteredDirs[m.selectedForDelete]
				shortName := core.ExtractNameFromDirectory(dir.Name)
				if m.deleteInput == "yes" || m.deleteInput == dir.Name || m.deleteInput == shortName {
					if err := m.ConfirmDelete(); err != nil {
						statusCmd := m.setStatus(fmt.Sprintf("Delete failed: %v", err), true)
						m.CancelDelete()
						return m, statusCmd
					}
				}
				m.CancelDelete()
				return m, nil
			case "esc":
				m.CancelDelete()
				return m, nil
			case "backspace":
				m.deleteInput = trimLastRune(m.deleteInput)
				return m, nil
			default:
				m.deleteInput += insertableText(msg)
				return m, nil
			}
		}

		if m.initializingGit && m.gitInitConfirm {
			switch msg.String() {
			case "y", "Y", "enter":
				if selected := m.GetSelected(); selected != nil && !selected.IsCreateNew {
					if err := initializeGitRepository(selected.Path); err != nil {
						statusCmd := m.setStatus(fmt.Sprintf("Git init failed: %v", err), true)
						m.CancelGitInit()
						return m, statusCmd
					}
					m.LoadDirectories()
				}
				m.CancelGitInit()
				return m, nil
			case "n", "N", "esc":
				m.CancelGitInit()
				return m, nil
			}
			return m, nil
		}

		// Normal mode key handling
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "?":
			// Show help when not searching (common TUI convention)
			if m.query == "" {
				m.showHelp = true
				return m, nil
			}
			// Otherwise treat as search input
			m.AppendToQuery("?")
			return m, nil

		case "enter":
			// Check what's selected
			if selected := m.GetSelected(); selected != nil {
				if selected.IsCreateNew {
					// Create new directory
					path, err := core.CreateDirectory(selected.CreateQuery)
					if err != nil {
						statusCmd := m.setStatus(fmt.Sprintf("Create failed: %v", err), true)
						return m, statusCmd
					}
					writeCdPath(path)
					return m, tea.Quit
				} else {
					// Select existing directory
					writeCdPath(selected.Path)
					return m, tea.Quit
				}
			}
			return m, nil

		case "tab":
			// Auto-complete
			if len(m.filteredDirs) > 0 && m.query != "" {
				m.SetQuery(m.filteredDirs[0].Name)
			}
			return m, nil

		case "ctrl+x":
			// Delete directory (X = delete)
			m.StartDelete()
			if m.deleting {
				m.deleteConfirm = true
			} else {
				statusCmd := m.setStatus("No directory selected to delete", false)
				return m, statusCmd
			}
			return m, nil

		case "ctrl+t":
			// Create worktree (T = Tree)
			if selected := m.GetSelected(); selected != nil && !selected.IsCreateNew {
				if isGitRepository(selected.Path) {
					m.creatingWorktree = true
					m.worktreeRepo = selected.Path
					m.worktreeInput = filepath.Base(selected.Path) + "-worktree"
				} else {
					statusCmd := m.setStatus("Not a git repository", false)
					return m, statusCmd
				}
			}
			return m, nil

		case "ctrl+g":
			m.cloning = true
			return m, nil

		case "ctrl+r":
			if selected := m.GetSelected(); selected != nil && !selected.IsCreateNew {
				if selected.IsGitRepo || selected.IsWorktree {
					statusCmd := m.setStatus("Already a git repository", false)
					return m, statusCmd
				}
			}
			m.StartGitInit()
			return m, nil

		case "ctrl+u":
			// Clear entire search (macOS standard: kill to beginning of line)
			m.SetQuery("")
			return m, nil

		case "ctrl+w":
			// Delete word backward (macOS terminal standard)
			m.DeleteWordFromQuery()
			return m, nil

		case "ctrl+n", "ctrl+j":
			// Move down (Emacs/macOS standard navigation)
			m.list.CursorDown()
			return m, nil

		case "ctrl+p", "ctrl+k":
			// Move up (Emacs/macOS standard navigation)
			m.list.CursorUp()
			return m, nil

		case "ctrl+f":
			// Page down (Emacs/less standard)
			m.list.Paginator.NextPage()
			return m, nil

		case "ctrl+b":
			// Page up (Emacs/less standard)
			m.list.Paginator.PrevPage()
			return m, nil

		case "ctrl+e":
			// Open in system file manager (E = Explore)
			if selected := m.GetSelected(); selected != nil && !selected.IsCreateNew {
				if err := openInFileManager(selected.Path); err != nil {
					statusCmd := m.setStatus(fmt.Sprintf("Open failed: %v", err), true)
					return m, statusCmd
				}
				statusCmd := m.setStatus("Opened in file manager", false)
				return m, statusCmd
			}
			return m, nil

		case "ctrl+o":
			// Create new directory explicitly (O = Open/create new)
			if m.query == "" {
				statusCmd := m.setStatus("Type a name first", false)
				return m, statusCmd
			}
			m.StartExplicitCreate()
			return m, nil

		case "backspace", "ctrl+h":
			// Delete character (ctrl+h = backspace in macOS terminal)
			m.DeleteFromQuery()
			return m, nil

		case "esc":
			if m.explicitCreating {
				m.CancelExplicitCreate()
			} else if m.query != "" {
				m.SetQuery("")
			} else if m.deleting {
				m.CancelDelete()
			} else {
				return m, tea.Quit
			}
			return m, nil

		default:
			// Handle character input for search (including pasted text)
			if text := insertableText(msg); text != "" {
				m.AppendToQuery(text)
			} else {
				// Let list handle other navigation keys
				m.list, cmd = m.list.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		m.list, cmd = m.list.Update(msg)
		return m, cmd

	case directoriesLoadedMsg:
		m.directories = msg.dirs
		m.updateFiltered()
		return m, nil

	case clearStatusMsg:
		m.statusMessage = ""
		m.statusIsError = false
		return m, nil

	case cloneFinishedMsg:
		m.cloneInProgress = false
		if msg.err != nil {
			statusCmd := m.setStatus(fmt.Sprintf("Clone failed: %v", msg.err), true)
			m.cloning = false
			m.cloneInput = ""
			return m, statusCmd
		}
		statusCmd := m.setStatus("Repository cloned successfully", false)
		m.cloning = false
		m.cloneInput = ""
		m.LoadDirectories()
		return m, statusCmd

	case error:
		statusCmd := m.setStatus(fmt.Sprintf("Error: %v", msg), true)
		return m, statusCmd

	default:
		// Pass other messages to the list
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

type directoriesLoadedMsg struct {
	dirs []core.Directory
}

func writeCdPath(path string) {
	// Single choke point: writes ~/.try_cd for the shell wrapper and records the
	// access for frecency ranking.
	core.WriteCdPath(path)
}

func isGitRepository(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func runClone(url string) tea.Cmd {
	return func() tea.Msg {
		err := cloneRepository(url)
		return cloneFinishedMsg{err: err}
	}
}

func cloneRepository(url string) error {
	core.EnsureTryDirectory()
	
	// Extract name from URL
	name := core.ExtractNameFromGitURL(url)
	dirName := core.GenerateDatedName(name)
	fullPath := filepath.Join(core.GetTryPath(), dirName)
	
	// Clone the repository
	cmd := exec.Command("git", "clone", url, fullPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	
	// Write the path for cd
	writeCdPath(fullPath)
	
	return nil
}

func createWorktreeFromPath(repoPath, name string) error {
	if name == "" {
		name = filepath.Base(repoPath) + "-worktree"
	}
	
	// Generate dated name for the worktree
	worktreeName := core.GenerateDatedName(name)
	worktreePath := filepath.Join(core.GetTryPath(), worktreeName)
	
	// Create a unique branch name
	branchName := fmt.Sprintf("worktree-%s-%d", name, time.Now().Unix())
	
	// Create the worktree with -f flag in case it already exists
	cmd := exec.Command("git", "worktree", "add", "-f", "-b", branchName, worktreePath)
	cmd.Dir = repoPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w\nOutput: %s", err, string(output))
	}
	
	// Write path for cd
	writeCdPath(worktreePath)
	
	return nil
}

func openInFileManager(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}

func initializeGitRepository(path string) error {
	// Initialize Git repository in the specified directory
	cmd := exec.Command("git", "init")
	cmd.Dir = path
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to initialize Git repository: %w\nOutput: %s", err, string(output))
	}
	
	return nil
}
// insertableText extracts the printable text carried by a key message. A key
// message may hold more than one rune: bracketed paste delivers the whole
// clipboard in a single tea.KeyRunes event, which is why pasting a git URL used
// to be swallowed by the single-character checks. Control characters (including
// the newlines of a multi-line paste) are dropped, and modifier-prefixed keys
// such as alt+x are not text at all.
func insertableText(msg tea.KeyMsg) string {
	if msg.Alt {
		return ""
	}
	if msg.Type != tea.KeyRunes && msg.Type != tea.KeySpace {
		return ""
	}
	var b strings.Builder
	for _, r := range msg.Runes {
		if r == '\n' || r == '\r' || r == '\t' {
			// Treat line breaks inside a paste as separators, not as content.
			b.WriteRune(' ')
			continue
		}
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// trimLastRune removes the final character of s, counting runes rather than
// bytes so multi-byte input is not cut in half.
func trimLastRune(s string) string {
	if s == "" {
		return ""
	}
	_, size := utf8.DecodeLastRuneInString(s)
	return s[:len(s)-size]
}
