#!/bin/bash

# Install script for Try

echo "Installing Try..."

# Build the binary
echo "Building Try binary..."
go build -o try main.go || { echo "Build failed"; exit 1; }

# Create ~/.local/bin if it doesn't exist
mkdir -p ~/.local/bin

# Copy binary to ~/.local/bin
echo "Installing binary to ~/.local/bin/try..."
cp try ~/.local/bin/try
chmod +x ~/.local/bin/try

# Detect shell
SHELL_NAME=$(basename "$SHELL")
SHELL_CONFIG=""

if [[ "$SHELL_NAME" == *"zsh"* ]]; then
    SHELL_CONFIG="$HOME/.zshrc"
elif [[ "$SHELL_NAME" == *"bash"* ]]; then
    SHELL_CONFIG="$HOME/.bashrc"
elif [[ "$SHELL_NAME" == *"fish"* ]]; then
    SHELL_CONFIG="$HOME/.config/fish/config.fish"
else
    SHELL_CONFIG="$HOME/.bashrc"
fi

echo ""
echo "✅ Try has been installed to ~/.local/bin/try"
echo ""
echo "To complete the installation, add the following to your $SHELL_CONFIG:"
echo ""
echo "----------------------------------------"
echo '# Try - Fresh Directories for Every Vibe'
echo 'export PATH="$HOME/.local/bin:$PATH"'
echo 'eval "$(~/.local/bin/try init ~/src/tries)"'
echo "----------------------------------------"
echo ""
echo "Then reload your shell configuration:"
echo "  source $SHELL_CONFIG"
echo ""
echo "After that, you can use 'try' and it will automatically cd to selected/created directories!"