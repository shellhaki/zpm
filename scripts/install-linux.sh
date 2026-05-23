#!/bin/bash
# ZPM (Zen Process Manager) Installer for Linux
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
        aarch64|arm64)
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
    
    local url=$(echo "$release_info" | grep -o "https://github.com/${REPO}/releases/download/[^\"]*zpm-linux-${arch}\.tar\.gz" | head -1)
    
    if [ -z "$url" ]; then
        echo "Error: Could not find release for linux-${arch}" >&2
        echo "Release info: $release_info" >&2
        exit 1
    fi
    
    echo "$url"
}

# Main installation
main() {
    echo -e "${YELLOW}=== ZPM Installer for Linux ===${NC}"
    
    local arch=$(detect_arch)
    echo -e "${GREEN}✓ Detected architecture: linux-${arch}${NC}"
    
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
    local extracted_dir=$(ls -d $temp_dir/zpm-linux-* 2>/dev/null | head -1)
    
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
    
    if [ -n "$ZSH_VERSION" ]; then
        shell_rc="$HOME/.zshrc"
    else
        shell_rc="$HOME/.bashrc"
    fi
    
    if [ -f "$shell_rc" ] && ! grep -q "zpm" "$shell_rc"; then
        echo "" >> "$shell_rc"
        echo "# ZPM (Zen Process Manager)" >> "$shell_rc"
        echo "$path_export" >> "$shell_rc"
        echo -e "${GREEN}✓ Added to PATH in $shell_rc${NC}"
    fi
    
    # Create systemd service for zpmd if using systemd
    if command -v systemctl &> /dev/null; then
        local service_dir="$HOME/.config/systemd/user"
        mkdir -p "$service_dir"
        
        cat > "$service_dir/zpmd.service" << 'EOF'
[Unit]
Description=ZPM Daemon (Zen Process Manager)
After=network.target

[Service]
Type=simple
ExecStart=%h/.local/bin/zpmd
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
EOF
        
        systemctl --user daemon-reload
        echo -e "${GREEN}✓ Created systemd service: $service_dir/zpmd.service${NC}"
        echo -e "${YELLOW}To enable zpmd on startup, run:${NC}"
        echo -e "${YELLOW}  systemctl --user enable zpmd${NC}"
        echo -e "${YELLOW}  systemctl --user start zpmd${NC}"
    else
        echo -e "${YELLOW}Tip: Add zpmd to your system's autostart to run it on boot${NC}"
    fi
    
    echo ""
    echo -e "${GREEN}=== Installation Complete ===${NC}"
    echo -e "${YELLOW}Next steps:${NC}"
    echo "1. Run: source $shell_rc"
    echo "2. Start zpmd: zpmd"
    echo "3. Test installation: zpm --help"
}

main "$@"
