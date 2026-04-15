#!/bin/bash
# Install git hooks for the project

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BACKEND_DIR/.." && pwd)"
GIT_DIR="$REPO_ROOT/.git"

if [ ! -d "$GIT_DIR" ]; then
    echo "Error: Not in a git repository (looking for $GIT_DIR)"
    exit 1
fi

HOOKS_DIR="$GIT_DIR/hooks"
SOURCE_HOOKS="$BACKEND_DIR/.githooks"

echo "Installing git hooks..."
echo "  Backend: $BACKEND_DIR"
echo "  Git dir: $GIT_DIR"

# Create hooks directory if it doesn't exist
mkdir -p "$HOOKS_DIR"

# Install pre-commit hook
if [ -f "$SOURCE_HOOKS/pre-commit" ]; then
    cp "$SOURCE_HOOKS/pre-commit" "$HOOKS_DIR/pre-commit"
    chmod +x "$HOOKS_DIR/pre-commit"
    echo "✅ pre-commit hook installed"
else
    echo "⚠️  pre-commit hook not found in $SOURCE_HOOKS"
fi

echo "Git hooks installation complete!"
