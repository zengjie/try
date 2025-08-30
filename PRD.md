# Product Requirements Document: Try - Fresh Directories for Every Vibe

## Executive Summary

**Try** is a lightweight command-line tool designed to help developers manage experimental projects and quick coding sessions. It provides an organized, searchable workspace for all temporary projects, experiments, and learning exercises that would otherwise clutter the filesystem or get lost in temporary directories.

## Problem Statement

Developers frequently create disposable directories for:
- Learning new technologies
- Testing libraries and frameworks
- Experimenting with code snippets
- Quick prototypes and proof-of-concepts

This leads to:
- Scattered test directories across the filesystem (`test`, `test2`, `new-test`, `actually-working-test`)
- Lost work in `/tmp` directories that get cleared
- Difficulty finding previous experiments
- No systematic organization for experimental code

## Solution Overview

Try provides a centralized, time-aware system for managing experimental directories with:
- **Instant fuzzy search** across all experiments
- **Automatic date prefixing** for chronological organization
- **Smart sorting** based on recency and relevance
- **Zero configuration** - single binary executable, no dependencies
- **Shell integration** for seamless workflow

## Core Features

### 1. Directory Management

#### 1.1 Centralized Storage
- Default location: `~/src/tries` (configurable via `TRY_PATH`)
- All experiments stored in one searchable location
- Automatic directory creation if not exists

#### 1.2 Automatic Date Prefixing
- Format: `YYYY-MM-DD-name`
- Example: `2025-08-17-redis-experiment`
- Provides chronological context for experiments
- Helps identify when work was done

#### 1.3 Smart Naming
- Spaces automatically converted to hyphens
- Handles versioning for duplicate names on same day
- Supports custom naming or auto-generation

### 2. Interactive Terminal UI

#### 2.1 Fuzzy Search Interface
- Real-time filtering as you type
- Smart matching algorithm:
  - Matches across word boundaries
  - Rewards proximity of matched characters
  - Prefers shorter names for equal matches
  - Case-insensitive matching

#### 2.2 Visual Elements
- Clean, minimal interface
- Syntax highlighting for matches
- Shows directory age and relevance score
- Scroll indicators for long lists
- Dark mode by default

#### 2.3 Keyboard Navigation
- **↑/↓** or **Ctrl-P/N** - Navigate up/down
- **Ctrl-J/K** - Vim-style navigation
- **Enter** - Select or create directory
- **Backspace** - Delete character
- **Ctrl-D** - Delete directory (with confirmation)
- **ESC** - Cancel operation

### 3. Git Integration

#### 3.1 Repository Cloning
- Clone directly into date-prefixed directories
- Automatic name extraction from Git URLs
- Support for multiple Git hosting services:
  - GitHub (HTTPS and SSH)
  - GitLab
  - Custom Git hosts

#### 3.2 Git Worktree Support
- Create detached worktrees from existing repositories
- Useful for experimenting without affecting main repository
- Automatic detection of Git repositories
- Falls back to regular directory creation for non-Git folders

### 4. Smart Scoring System

#### 4.1 Scoring Factors
- **Text matching score** (when searching):
  - Base points for character matches
  - Bonus for word boundary matches
  - Proximity bonus for consecutive matches
  - Density bonus for compact matches
  - Length penalty (shorter names score higher)
  
- **Time-based scoring** (always applied):
  - Creation time bonus (newer = higher score)
  - Access time bonus (recently accessed = higher score)
  - Uses sqrt decay for smooth time-based scoring

#### 4.2 Default Preferences
- Date-prefixed directories get bonus points
- Recently accessed items bubble to top
- Maintains relevance even with hundreds of directories

### 5. Shell Integration

#### 5.1 Supported Shells
- Bash
- Zsh
- Fish

#### 5.2 Integration Method
- Shell function wrapper around Ruby script
- Captures output for directory changes
- Transparent error handling
- Works with existing shell configurations

### 6. Command-Line Interface

#### 6.1 Basic Commands

```bash
try                     # Interactive directory selector
try [query]            # Search and select/create with query
try new [name]         # Create new dated directory
try . [name]           # Create worktree for current repo
try clone <git-url>    # Clone repository
try worktree <path>    # Create worktree from path
```

#### 6.2 URL Shorthand
- Recognizes Git URLs automatically
- `try https://github.com/user/repo.git` → clones repository
- No need to type `clone` for recognized URLs

## Implementation Language

The tool will be implemented in Go to provide a single static binary with fast startup time, easy cross-platform distribution, and no runtime dependencies.

## Technical Architecture (Go Implementation)

### Components

#### 1. UI Package
- Terminal UI using Bubble Tea framework
- Model-View-Update architecture
- ANSI escape sequence management via Lip Gloss
- TTY detection and fallback
- Cross-platform terminal handling

#### 2. Core Package
- Main application logic
- Directory scanning with filepath.Walk
- Scoring algorithm implementation
- Interactive selection loop
- File operations using os and filepath packages

#### 3. Shell Package
- Command parsing with flag package
- Git operations via os/exec
- Shell-specific script generation
- Path management with filepath package

#### 4. Suggested Go Libraries
- **TUI Framework**: `github.com/charmbracelet/bubbletea`
- **Styling**: `github.com/charmbracelet/lipgloss`
- **Fuzzy Search**: `github.com/sahilm/fuzzy` or custom implementation
- **Key Bindings**: `github.com/charmbracelet/bubbles/key`
- **Testing**: Standard library `testing` package

### Design Principles

1. **Single Binary Distribution**
   - Compiled Go binary
   - No runtime dependencies
   - Easy to install and distribute

2. **Performance**
   - Single-pass directory scanning
   - Memoized directory listing
   - Efficient scoring algorithm
   - Handles thousands of directories

3. **User Experience**
   - Instant responsiveness
   - No configuration required
   - Intuitive keyboard shortcuts
   - Clear visual feedback

4. **Compatibility**
   - Cross-platform binary (macOS, Linux, Windows)
   - ARM and x86 architectures
   - Multiple shell support
   - Static linking for maximum portability

## Non-Functional Requirements

### Performance
- Sub-second response for directory selection
- Handle 1000+ directories efficiently
- Minimal memory footprint
- No background processes

### Usability
- Zero configuration setup
- Intuitive without documentation
- Consistent keyboard shortcuts
- Clear error messages

### Reliability
- Graceful handling of:
  - Missing directories
  - Permission errors
  - Malformed input
  - Terminal resize events

### Portability
- Statically compiled Go binary
- No runtime dependencies
- Cross-compilation support
- Shell-agnostic core functionality

## Installation Requirements

### Quick Install (Go Binary)
```bash
# Download pre-compiled binary (example)
curl -L https://github.com/user/try/releases/latest/download/try-$(uname -s)-$(uname -m) -o ~/.local/bin/try
chmod +x ~/.local/bin/try
echo 'eval "$(try init ~/src/tries)"' >> ~/.zshrc
```

### Build from Source
```bash
go install github.com/user/try@latest
```

### System Requirements
- No runtime dependencies (statically compiled)
- Unix-like operating system (Linux, macOS, BSD)
- Terminal with ANSI support
- One of: Bash, Zsh, or Fish shell

### Platform Support
- **Linux**: amd64, arm64, arm
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)
- **Windows**: amd64 (experimental)
- **BSD**: FreeBSD, OpenBSD

## Configuration

### Environment Variables
- `TRY_PATH`: Override default directory location
- `SHELL`: Auto-detected for shell integration

### Customization Points
- Directory location
- Date format (hardcoded but modifiable)
- Scoring weights (in source code)
- UI tokens and colors

## Use Cases

### Primary Use Cases
1. **Learning new technology**: Create isolated environment for tutorials
2. **Testing libraries**: Quick setup for library evaluation
3. **Debugging code**: Isolated reproduction of issues
4. **Prototyping**: Rapid idea validation
5. **Code snippets**: Organized storage of examples

### User Personas
1. **The Learner**: Following tutorials, needs many test directories
2. **The Experimenter**: Trying different approaches to problems
3. **The Debugger**: Creating minimal reproductions
4. **The ADHD Developer**: Needs organization for chaotic workflow

## Success Metrics

1. **Adoption**: Number of GitHub stars/forks
2. **Retention**: Continued use after 30 days
3. **Efficiency**: Time saved finding old experiments
4. **Organization**: Reduction in scattered test directories
5. **User Satisfaction**: Positive feedback and contributions

## Go Implementation Details

### Project Structure
```
try/
├── main.go              # Entry point and command parsing
├── ui/
│   ├── model.go        # Bubble Tea model
│   ├── view.go         # Rendering logic
│   └── update.go       # Event handling
├── core/
│   ├── scanner.go      # Directory scanning
│   ├── scorer.go       # Fuzzy search scoring
│   └── manager.go      # Directory operations
├── shell/
│   ├── bash.go         # Bash integration
│   ├── zsh.go          # Zsh integration
│   └── fish.go         # Fish integration
└── cmd/
    ├── cd.go           # Interactive selector
    ├── clone.go        # Git clone command
    └── worktree.go     # Worktree command
```

### Key Implementation Considerations

#### Concurrency
- Use goroutines for parallel directory scanning
- Channel-based communication for UI updates
- Context for cancellation support

#### Error Handling
- Wrapped errors with context
- Graceful degradation for non-critical failures
- User-friendly error messages

#### Testing Strategy
- Unit tests for scoring algorithm
- Integration tests for Git operations
- Mock filesystem for directory operations
- Terminal UI testing with golden files

#### Build Configuration
```makefile
# Cross-compilation targets
build-all:
	GOOS=linux GOARCH=amd64 go build -o dist/try-linux-amd64
	GOOS=darwin GOARCH=amd64 go build -o dist/try-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build -o dist/try-darwin-arm64
	GOOS=windows GOARCH=amd64 go build -o dist/try-windows-amd64.exe
```

#### Performance Optimizations
- Directory metadata caching
- Lazy loading for large directories
- Compiled regex patterns
- String builder for shell script generation

## Future Enhancements

### Potential Features
1. **Tags/Categories**: Additional organization layer
2. **Templates**: Boilerplate for common experiments
3. **Export/Archive**: Backup old experiments
4. **Search History**: Remember common searches
5. **Multi-machine Sync**: Share experiments across devices
6. **IDE Integration**: VSCode/Vim extensions
7. **Metrics Dashboard**: Usage statistics and insights

### Technical Improvements
1. **Parallel Directory Scanning**: For very large collections
2. **SQLite Cache**: Persistent metadata storage
3. **Configurable Shortcuts**: User-defined key bindings
4. **Plugin System**: Extensibility for workflows
5. **Web UI**: Browser-based interface option

## Philosophy

"Your brain doesn't work in neat folders. You have ideas, you try things, you context-switch like a caffeinated squirrel. This tool embraces that."

Try is built for developers who:
- Value speed over structure
- Need findability without organization overhead  
- Work on many small experiments
- Want their 2am coding sessions preserved
- Have ADHD or similar working styles

The tool prioritizes:
- **Immediate utility** over perfect organization
- **Low friction** over feature richness
- **Discoverability** over hierarchical structure
- **Simplicity** over configurability