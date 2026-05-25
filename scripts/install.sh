#!/bin/bash
# ZPM Installer - Universal entry point for all platforms
# This script detects the OS and runs the appropriate installer

set -e

VERSION="${1:-latest}"
REPO="shellhaki/zpm"
GITHUB_RAW="https://raw.githubusercontent.com/${REPO}/main/scripts"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)
            echo "Linux"
            ;;
        Darwin*)
            echo "macOS"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            echo "Windows"
            ;;
        *)
            echo "Unknown"
            ;;
    esac
}

main() {
    local os=$(detect_os)
    
    case "$os" in
        Linux)
            echo -e "${YELLOW}Detected Linux system${NC}"
            local script_url="${GITHUB_RAW}/install-linux.sh"
            bash <(curl -sL "$script_url") "$VERSION"
            ;;
        macOS)
            echo -e "${YELLOW}Detected macOS system${NC}"
            local script_url="${GITHUB_RAW}/install-macos.sh"
            bash <(curl -sL "$script_url") "$VERSION"
            ;;
        Windows)
            echo -e "${YELLOW}Detected Windows system - requires PowerShell${NC}"
            echo -e "${YELLOW}Run the following command in PowerShell as Administrator:${NC}"
            echo "irm https://raw.githubusercontent.com/${REPO}/main/scripts/install-windows.ps1 | iex"
            exit 0
            ;;
        *)
            echo -e "${RED}Error: Unsupported OS: $os${NC}" >&2
            exit 1
            ;;
    esac
}

main "$@"
