#!/bin/bash
# ZPM (Zen Process Manager) Installer for macOS
# Downloads and installs both zpm and zpmd

set -e

VERSION="${1:-latest}"
INSTALL_DIR="${2:-.local/bin}"
REPO="shellhaki/zpm"
GITHUB_API="https://api.github.com/repos/${REPO}/releases"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect architecture
detect_arch() {
    local arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)
            echo "amd64"
            ;;
        arm64|aarch64)
            echo "arm64"
            ;;
        *)
            echo "Error: Unsupported architecture: $arch" >&2
            exit 1
            ;;
    esac
}

# Get download URL
get_download_url() {
    local version="$1"
    local arch="$2"
    
    if [ "$version" = "latest" ]; then
        # Get latest release
        local release_info=$(curl -s "${GITHUB_API}/latest")
    else
        # Get specific version
        local release_info=$(curl -s "${GITHUB_API}/tags/${version}")
    fi
    
    local url=$(echo "$release_info" | grep -o "https://github.com/${REPO}/releases/download/[^\"]*zpm-darwin-${arch}\.tar\.gz" | head -1)
    
    if [ -z "$url" ]; then
        echo "Error: Could not find release for darwin-${arch}" >&2
        echo "Release info: $release_info" >&2
        exit 1
    fi
    
    echo "$url"
}

# Main installation
main() {
    echo -e "${YELLOW}=== ZPM Installer for macOS ===${NC}"
    
    local arch=$(detect_arch)
    echo -e "${GREEN}✓ Detected architecture: darwin-${arch}${NC}"
    
    # Create install directory
    mkdir -p "$HOME/$INSTALL_DIR"
    echo -e "${GREEN}✓ Install directory: $HOME/$INSTALL_DIR${NC}"
    
    # Get download URL
    echo -e "${YELLOW}Fetching release information...${NC}"
    local download_url=$(get_download_url "$VERSION" "$arch")
    echo -e "${GREEN}✓ Download URL: $download_url${NC}"
    
    # Create temporary directory
    local temp_dir=$(mktemp -d)
    trap "rm -rf $temp_dir" EXIT
    
    echo -e "${YELLOW}Downloading ZPM $VERSION...${NC}"
    if ! curl -sL "$download_url" -o "$temp_dir/zpm.tar.gz"; then
        echo -e "${RED}✗ Failed to download ZPM${NC}" >&2
        exit 1
    fi
    echo -e "${GREEN}✓ Downloaded successfully${NC}"
    
    # Extract
    echo -e "${YELLOW}Extracting binaries...${NC}"
    tar -xzf "$temp_dir/zpm.tar.gz" -C "$temp_dir"
    local extracted_dir=$(ls -d $temp_dir/zpm-darwin-* 2>/dev/null | head -1)
    
    if [ ! -d "$extracted_dir" ]; then
        echo -e "${RED}✗ Failed to extract archive${NC}" >&2
        exit 1
    fi
    
    # Install binaries
    if [ ! -f "$extracted_dir/zpm" ] || [ ! -f "$extracted_dir/zpmd" ]; then
        echo -e "${RED}✗ Required binaries not found in archive${NC}" >&2
        exit 1
    fi
    
    cp "$extracted_dir/zpm" "$HOME/$INSTALL_DIR/zpm"
    cp "$extracted_dir/zpmd" "$HOME/$INSTALL_DIR/zpmd"
    chmod +x "$HOME/$INSTALL_DIR/zpm" "$HOME/$INSTALL_DIR/zpmd"
    echo -e "${GREEN}✓ Binaries installed${NC}"
    
    # Add to PATH if needed
    local path_export="export PATH=\"\$HOME/$INSTALL_DIR:\$PATH\""
    local shell_rc=""
    
    # Detect shell
    if [ -n "$ZSH_VERSION" ] || [ "$(basename "$SHELL")" = "zsh" ]; then
        shell_rc="$HOME/.zshrc"
    else
        shell_rc="$HOME/.bash_profile"
    fi
    
    if [ -f "$shell_rc" ] && ! grep -q "zpm" "$shell_rc"; then
        echo "" >> "$shell_rc"
        echo "# ZPM (Zen Process Manager)" >> "$shell_rc"
        echo "$path_export" >> "$shell_rc"
        echo -e "${GREEN}✓ Added to PATH in $shell_rc${NC}"
    fi
    
    # Create LaunchAgent for zpmd
    local launchagent_dir="$HOME/Library/LaunchAgents"
    mkdir -p "$launchagent_dir"
    
    cat > "$launchagent_dir/com.zpm.daemon.plist" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.zpm.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>%HOME%/.local/bin/zpmd</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%HOME%/.zpm/zpmd.log</string>
    <key>StandardErrorPath</key>
    <string>%HOME%/.zpm/zpmd.error.log</string>
</dict>
</plist>
EOF
    
    # Replace %HOME% with actual home directory
    sed -i '' "s|%HOME%|$HOME|g" "$launchagent_dir/com.zpm.daemon.plist"
    
    # Create log directory
    mkdir -p "$HOME/.zpm"
    
    echo -e "${GREEN}✓ Created LaunchAgent: $launchagent_dir/com.zpm.daemon.plist${NC}"
    echo -e "${YELLOW}To enable zpmd on startup, run:${NC}"
    echo -e "${YELLOW}  launchctl load ~/Library/LaunchAgents/com.zpm.daemon.plist${NC}"
    
    echo ""
    echo -e "${GREEN}=== Installation Complete ===${NC}"
    echo -e "${YELLOW}Next steps:${NC}"
    echo "1. Run: source $shell_rc"
    echo "2. Load LaunchAgent: launchctl load ~/Library/LaunchAgents/com.zpm.daemon.plist"
    echo "3. Test installation: zpm --help"
}

main "$@"
