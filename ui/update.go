package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zengjie/try/core"
)

type DirectorySelectedMsg struct {
	Path string
}

type DirectoryCreatedMsg struct {
	Path string
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
		// If help is showing, any key returns to main view
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		// Handle special input modes
		if m.creatingWorktree {
			switch msg.String() {
			case "enter":
				if err := createWorktreeFromPath(m.worktreeRepo, m.worktreeInput); err != nil {
					m.err = err
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
				if len(m.worktreeInput) > 0 {
					m.worktreeInput = m.worktreeInput[:len(m.worktreeInput)-1]
				}
				return m, nil
			default:
				if len(msg.String()) == 1 {
					r := []rune(msg.String())[0]
					if r >= 32 && r < 127 {
						m.worktreeInput += string(r)
					}
				}
				return m, nil
			}
		}

		if m.cloning {
			switch msg.String() {
			case "enter":
				if m.cloneInput != "" {
					if err := cloneRepository(m.cloneInput); err != nil {
						m.err = err
					} else {
						m.cloning = false
						m.cloneInput = ""
						m.LoadDirectories()
					}
				}
				return m, nil
			case "esc":
				m.cloning = false
				m.cloneInput = ""
				return m, nil
			case "backspace":
				if len(m.cloneInput) > 0 {
					m.cloneInput = m.cloneInput[:len(m.cloneInput)-1]
				}
				return m, nil
			default:
				if len(msg.String()) == 1 {
					r := []rune(msg.String())[0]
					if r >= 32 && r < 127 {
						m.cloneInput += string(r)
					}
				}
				return m, nil
			}
		}

		if m.deleting && m.deleteConfirm {
			switch msg.String() {
			case "enter":
				dir := m.filteredDirs[m.selectedForDelete]
				if m.deleteInput == "yes" || m.deleteInput == dir.Name {
					if err := m.ConfirmDelete(); err != nil {
						m.err = err
					}
				}
				m.CancelDelete()
				return m, nil
			case "esc":
				m.CancelDelete()
				return m, nil
			case "backspace":
				if len(m.deleteInput) > 0 {
					m.deleteInput = m.deleteInput[:len(m.deleteInput)-1]
				}
				return m, nil
			default:
				if len(msg.String()) == 1 {
					r := []rune(msg.String())[0]
					if r >= 32 && r < 127 {
						m.deleteInput += string(r)
					}
				}
				return m, nil
			}
		}


		// Normal mode key handling
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "?":
			m.showHelp = true
			return m, nil

		case "/":
			// Start search mode by appending to query
			m.AppendToQuery('/')
			return m, nil

		case "enter":
			if m.IsCreating() {
				// Create new directory
				path, err := core.CreateDirectory(m.query)
				if err != nil {
					m.err = err
					return m, nil
				}
				writeCdPath(path)
				return m, tea.Quit
			}
			
			// Select existing directory
			if selected := m.GetSelected(); selected != nil {
				writeCdPath(selected.Path)
				return m, tea.Quit
			}
			return m, nil

		case "tab":
			// Auto-complete
			if len(m.filteredDirs) > 0 && m.query != "" {
				m.SetQuery(m.filteredDirs[0].Name)
			}
			return m, nil

		case "ctrl+d":
			m.StartDelete()
			if m.deleting {
				m.deleteConfirm = true
			}
			return m, nil

		case "ctrl+w":
			if selected := m.GetSelected(); selected != nil {
				if isGitRepository(selected.Path) {
					m.creatingWorktree = true
					m.worktreeRepo = selected.Path
					m.worktreeInput = filepath.Base(selected.Path) + "-worktree"
				}
			}
			return m, nil

		case "ctrl+g":
			m.cloning = true
			return m, nil

		case "ctrl+u":
			m.SetQuery("")
			return m, nil

		case "ctrl+j":
			// Move down (hidden navigation feature)
			m.list.CursorDown()
			return m, nil

		case "ctrl+k":
			// Move up (hidden navigation feature)
			m.list.CursorUp()
			return m, nil

		case "ctrl+p":
			// Page up (hidden navigation feature)
			m.list.Paginator.PrevPage()
			return m, nil

		case "ctrl+n":
			// Page down (hidden navigation feature)
			m.list.Paginator.NextPage()
			return m, nil

		case "backspace":
			m.DeleteFromQuery()
			return m, nil

		case "esc":
			if m.query != "" {
				m.SetQuery("")
			} else if m.deleting {
				m.CancelDelete()
			} else {
				return m, tea.Quit
			}
			return m, nil

		default:
			// Handle character input for search
			if len(msg.String()) == 1 {
				r := []rune(msg.String())[0]
				if r >= 32 && r < 127 {
					m.AppendToQuery(r)
				}
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

	case error:
		m.err = msg
		return m, nil

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

func loadDirectories() tea.Cmd {
	return func() tea.Msg {
		dirs, err := core.ScanDirectories()
		if err != nil {
			return err
		}
		return directoriesLoadedMsg{dirs: dirs}
	}
}

func writeCdPath(path string) {
	home, _ := os.UserHomeDir()
	cdFile := filepath.Join(home, ".try_cd")
	os.WriteFile(cdFile, []byte(path), 0644)
}

func isGitRepository(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	return info.IsDir()
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