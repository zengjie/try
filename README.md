# Try - Fresh Directories for Every Vibe

A lightweight command-line tool for managing experimental projects and quick coding sessions. Try provides an organized, searchable workspace for all your temporary projects, experiments, and learning exercises.

## Features

- **Instant fuzzy search** across all experiments
- **Automatic date prefixing** for chronological organization  
- **Smart scoring** based on recency and relevance
- **Git integration** for cloning and worktree creation
- **Zero configuration** - single binary executable
- **Shell integration** for seamless cd navigation

## Installation

### Quick Install

```bash
# Build from source
go install github.com/zengjie/try@latest

# Or clone and build locally
git clone https://github.com/zengjie/try.git
cd try
make install
```

### Shell Integration (Required for auto-cd)

**Important:** To enable automatic directory changing when you select or create a directory, you MUST set up shell integration.

Add to your shell configuration file:

```bash
# For Bash (~/.bashrc)
export PATH="$HOME/.local/bin:$PATH"
eval "$(try init ~/src/tries)"

# For Zsh (~/.zshrc)
export PATH="$HOME/.local/bin:$PATH"
eval "$(try init ~/src/tries)"

# For Fish (~/.config/fish/config.fish)
set -x PATH $HOME/.local/bin $PATH
try init ~/src/tries | source
```

Then reload your shell:
```bash
source ~/.zshrc  # or ~/.bashrc for Bash
```

The `~/src/tries` path is where all your experimental directories will be stored. You can customize this location.

**How it works:** The shell integration creates a `try` function that wraps the binary. When you select or create a directory, this function automatically `cd`s you there.

## Usage

### Interactive Directory Selection

```bash
try                     # Open interactive selector
try redis              # Search for directories containing "redis"
```

### Create New Directories

```bash
try new my-experiment  # Creates: ~/src/tries/2025-08-30-my-experiment
try new                # Creates: ~/src/tries/2025-08-30-experiment
```

### Git Operations

```bash
# Clone repository with automatic naming
try clone https://github.com/user/repo.git
# Creates: ~/src/tries/2025-08-30-repo

# URL shorthand - automatically detects Git URLs
try https://github.com/user/repo.git

# Create worktree from current repository
try . feature-branch
# Creates: ~/src/tries/2025-08-30-feature-branch

# Create worktree from specific repository
try worktree /path/to/repo branch-name
```

### Keyboard Shortcuts

- **↑/↓** or **Ctrl-P/N** - Navigate up/down
- **Ctrl-J/K** - Vim-style navigation  
- **Enter** - Select directory or create new
- **Backspace** - Delete character
- **Ctrl-D** - Delete directory (with confirmation)
- **ESC** - Cancel operation

## Environment Variables

- `TRY_PATH` - Override default directory location (default: `~/src/tries`)

## Directory Naming

Try automatically prefixes directories with the current date:

```
2025-08-30-redis-experiment
2025-08-30-react-tutorial
2025-08-30-bug-reproduction
```

If you create multiple directories with the same name on the same day, Try adds a number suffix:

```
2025-08-30-test
2025-08-30-test-1
2025-08-30-test-2
```

## Building from Source

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Clean build artifacts
make clean
```

## Philosophy

Your brain doesn't work in neat folders. You have ideas, you try things, you context-switch like a caffeinated squirrel. Try embraces that chaos and gives you a searchable, time-aware workspace for all your experiments.

Perfect for:
- Learning new technologies
- Testing libraries and frameworks
- Quick prototypes and proof-of-concepts
- Debugging and minimal reproductions
- Those 2am coding sessions you want to preserve

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.