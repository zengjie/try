# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

"Try" is a command-line tool for managing experimental projects and quick coding sessions. It provides a centralized, date-prefixed system for organizing temporary directories with fuzzy search capabilities.

## Implementation Status

Implementation is complete with full working functionality. The project is built in Go for single binary distribution.

## Architecture

The project follows a modular Go architecture:
- `main.go` - Entry point and command parsing
- `ui/` - Terminal UI using Bubble Tea framework (model/view/update pattern)
  - `model.go` - UI state and data model
  - `view.go` - UI rendering logic
  - `update.go` - Event handling and state updates
  - `help.go` - Help screen implementation
- `core/` - Directory scanning, scoring algorithm, and management operations
  - `manager.go` - Main directory management operations
  - `scanner.go` - Directory discovery and parsing
  - `scoring.go` - Fuzzy search scoring algorithm
  - `scoring_test.go` - Test suite for scoring algorithm
- `shell/` - Shell-specific integrations
  - `shell.go` - Shell detection and integration utilities
- `cmd/` - Command implementations
  - `selector.go` - Interactive directory selection UI
  - `git.go` - Git repository operations (clone, worktree)
- `Makefile` - Build automation and development commands
- `install.sh` - Installation script

## Key Implementation Notes

### Terminal UI
- Use Bubble Tea (`github.com/charmbracelet/bubbletea`) for interactive UI
- Lip Gloss (`github.com/charmbracelet/lipgloss`) for styling
- Model-View-Update architecture pattern

### Core Functionality
- Date prefix format: `YYYY-MM-DD-name`
- Default storage: `~/src/tries` (configurable via `TRY_PATH`)
- Fuzzy search with smart scoring (text matching + time-based decay)

### Available Commands
```bash
try                     # Interactive directory selector
try [query]            # Search and select/create with query
try new [name]         # Create new dated directory
try . [name]           # Create worktree for current repo
try clone <git-url>    # Clone repository
try worktree <path>    # Create worktree from path
```

## Development Commands

The project includes a Makefile with standard development commands:

### Build
```bash
make build              # Build the binary
make                    # Default build target
```

### Test
```bash
make test               # Run all tests
go test ./...           # Alternative test command
```

### Install
```bash
make install            # Install to GOBIN or ~/bin
./install.sh            # Run installation script
```

### Cross-compilation
```bash
make dist               # Build for all platforms
```

Individual platform builds:
```bash
GOOS=linux GOARCH=amd64 go build -o dist/try-linux-amd64
GOOS=darwin GOARCH=amd64 go build -o dist/try-darwin-amd64
GOOS=darwin GOARCH=arm64 go build -o dist/try-darwin-arm64
```

## Key Design Principles
- Single static binary with no runtime dependencies
- Sub-second response time for directory operations
- Handle 1000+ directories efficiently
- Zero configuration required
- Cross-platform support (Linux, macOS, Windows experimental)