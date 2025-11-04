#!/usr/bin/env bash
set -e

REPO="saravenpi/chime"
INSTALL_DIR="$HOME/.local/bin"
BINARY_NAME="chime"

echo "🚀 Installing Chime iMessage client..."

if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed. Please install Go first: https://golang.org/dl/"
    exit 1
fi

echo "✓ Go found: $(go version)"

if [ "$(uname)" != "Darwin" ]; then
    echo "❌ Error: Chime only works on macOS (requires iMessage database access)"
    exit 1
fi

echo "✓ macOS detected"

if [ ! -d "$INSTALL_DIR" ]; then
    echo "📁 Creating $INSTALL_DIR..."
    mkdir -p "$INSTALL_DIR"
fi

TEMP_DIR=$(mktemp -d)
echo "📦 Using temporary directory: $TEMP_DIR"

cleanup() {
    echo "🧹 Cleaning up temporary files..."
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

cd "$TEMP_DIR"

echo "⬇️  Downloading Chime..."
if command -v git &> /dev/null; then
    git clone --depth 1 "https://github.com/$REPO.git" chime
    cd chime
else
    curl -fsSL "https://github.com/$REPO/archive/refs/heads/master.tar.gz" | tar xz
    cd chime-master
fi

echo "🔨 Building Chime..."
go build -o "$BINARY_NAME" .

echo "📥 Installing to $INSTALL_DIR/$BINARY_NAME..."
mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

echo ""
echo "✅ Chime installed successfully!"
echo ""
echo "📋 Next steps:"
echo "  1. Ensure $INSTALL_DIR is in your PATH"
echo "     Add this to your ~/.bashrc or ~/.zshrc:"
echo "     export PATH=\"\$HOME/.local/bin:\$PATH\""
echo ""
echo "  2. Grant Full Disk Access to your terminal:"
echo "     System Preferences → Security & Privacy → Privacy → Full Disk Access"
echo "     Add Terminal.app or iTerm.app"
echo ""
echo "  3. Ensure Messages.app is signed in with your Apple ID"
echo ""
echo "  4. Run: chime"
echo ""

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "⚠️  Warning: $INSTALL_DIR is not in your PATH"
    echo "   Run: export PATH=\"\$HOME/.local/bin:\$PATH\""
fi
