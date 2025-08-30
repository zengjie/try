# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

"Try" is a command-line tool for managing experimental projects and quick coding sessions. It provides a centralized, date-prefixed system for organizing temporary directories with fuzzy search capabilities.

## Implementation Status

Currently in planning phase with PRD complete. Implementation will be in Go for single binary distribution.

## Architecture

The project follows a modular Go architecture:
- `main.go` - Entry point and command parsing
- `ui/` - Terminal UI using Bubble Tea framework (model/view/update pattern)
- `core/` - Directory scanning, scoring algorithm, and management operations
- `shell/` - Shell-specific integrations (bash, zsh, fish)
- `cmd/` - Command implementations (cd, clone, worktree)

## Key Implementation Notes

### Terminal UI
- Use Bubble Tea (`github.com/charmbracelet/bubbletea`) for interactive UI
- Lip Gloss (`github.com/charmbracelet/lipgloss`) for styling
- Model-View-Update architecture pattern

### Core Functionality
- Date prefix format: `YYYY-MM-DD-name`
- Default storage: `~/src/tries` (configurable via `TRY_PATH`)
- Fuzzy search with smart scoring (text matching + time-based decay)

### Commands to Implement
```bash
try                     # Interactive directory selector
try [query]            # Search and select/create with query
try new [name]         # Create new dated directory
try . [name]           # Create worktree for current repo
try clone <git-url>    # Clone repository
try worktree <path>    # Create worktree from path
```

## Development Commands

### Build
```bash
go build -o try main.go
```

### Test
```bash
go test ./...
```

### Cross-compilation
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