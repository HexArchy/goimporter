#!/bin/bash

set -e

echo "🚀 Installing goimporter with VK configuration..."

# Setup variables
INSTALL_DIR="$HOME/.goimporter"
CONFIG_DIR="$HOME/.config/goimporter"
CONFIG_FILE="$CONFIG_DIR/config.json"
BINARY_PATH="$HOME/go/bin/goimporter"
REPO_URL="https://github.com/HexArchy/goimporter.git"

# Create directories
mkdir -p "$INSTALL_DIR" "$CONFIG_DIR"

# Clone repository
echo "📦 Downloading goimporter..."
git clone --depth 1 "$REPO_URL" "$INSTALL_DIR" 2>/dev/null || (cd "$INSTALL_DIR" && git pull)
cd "$INSTALL_DIR"

# Build and install
echo "🔨 Building and installing..."
go build -v -ldflags="-s -w" -o "$BINARY_PATH" ./cmd/goimporter

# Create VK configuration
echo "⚙️  Creating VK configuration..."
cat > "$CONFIG_FILE" <<EOF
{
  "org_prefix": "gitlab.mvk.com",
  "repo_prefix": "gitlab.mvk.com/go/vkgo",
  "common_prefix": "gitlab.mvk.com/go/vkgo/pkg",
  "domain_prefix": "gitlab.mvk.com/go/vkgo/projects/health/pkg",
  "projects_template": "gitlab.mvk.com/go/vkgo/projects/health/%s",
  "additional_common_prefixes": [
    "gitlab.mvk.com/vkapi/vk-go-sdk-private"
  ]
}
EOF

# Create wrapper script to automatically use config
WRAPPER_SCRIPT="$HOME/go/bin/goimporter-wrapper.sh"
echo "📝 Creating wrapper script..."
cat > "$WRAPPER_SCRIPT" <<EOF
#!/bin/bash
exec "$BINARY_PATH" -config "$CONFIG_FILE" "\$@"
EOF
chmod +x "$WRAPPER_SCRIPT"

# Check if ~/go/bin is in PATH
SHELL_CONFIG=""
if [ -f "$HOME/.zshrc" ]; then
  SHELL_CONFIG="$HOME/.zshrc"
elif [ -f "$HOME/.bash_profile" ]; then
  SHELL_CONFIG="$HOME/.bash_profile"
else
  SHELL_CONFIG="$HOME/.bashrc"
fi

if [[ ":$PATH:" != *":$HOME/go/bin:"* ]]; then
  echo "⚠️  Warning: ~/go/bin is not in your PATH"
  echo "   Adding it to $SHELL_CONFIG..."
  echo 'export PATH="$PATH:$HOME/go/bin"' >> "$SHELL_CONFIG"
  echo "   Run 'source $SHELL_CONFIG' to update your PATH"
fi

# Move the wrapper to replace the original binary
mv "$WRAPPER_SCRIPT" "$BINARY_PATH"

echo "✨ Installation complete! ✨"
echo "You can now use 'goimporter' to format Go code with VK-specific import ordering."
echo "The VK configuration is automatically applied."
echo "For example: goimporter -r -dir=/path/to/project"