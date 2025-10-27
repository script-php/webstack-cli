#!/bin/bash
# WebStack CLI - Minimal Installation Script (CLI tool only)
# Usage: curl -fsSL https://your-domain.com/install-cli.sh | sudo bash

set -e

# Configuration
REPO_URL="https://github.com/script-php/webstack-cli"
BINARY_NAME="webstack"
INSTALL_DIR="/usr/local/bin"
VERSION="latest"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }
info() { echo -e "${BLUE}[INFO]${NC} $1"; }

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Detect OS and architecture
detect_platform() {
    local os=""
    local arch=""
    
    case "$(uname -s)" in
        Linux*)     os="linux";;
        Darwin*)    os="darwin";;
        *)          error "Unsupported operating system: $(uname -s)"; exit 1;;
    esac
    
    case "$(uname -m)" in
        x86_64)     arch="amd64";;
        arm64)      arch="arm64";;
        aarch64)    arch="arm64";;
        armv7l)     arch="arm";;
        *)          error "Unsupported architecture: $(uname -m)"; exit 1;;
    esac
    
    echo "${os}-${arch}"
}

# Download and install binary
install_binary() {
    local platform=$(detect_platform)
    local download_url=""
    
    if [[ "$VERSION" == "latest" ]]; then
        # GitHub releases API to get latest version
        local api_url="https://api.github.com/repos/script-php/webstack-cli/releases/latest"
        VERSION=$(curl -s "$api_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        
        if [[ -z "$VERSION" ]]; then
            error "Failed to fetch latest version"
            exit 1
        fi
    fi
    
    download_url="${REPO_URL}/releases/download/${VERSION}/webstack-${platform}"
    
    log "Downloading WebStack CLI ${VERSION} for ${platform}..."
    
    # Download binary
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "/tmp/${BINARY_NAME}" "$download_url"
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "/tmp/${BINARY_NAME}" "$download_url"
    else
        error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi
    
    # Make executable and move to install directory
    chmod +x "/tmp/${BINARY_NAME}"
    mv "/tmp/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    
    log "WebStack CLI installed to ${INSTALL_DIR}/${BINARY_NAME}"
}

# Install minimal dependencies
install_dependencies() {
    log "Installing minimal dependencies..."
    
    # Only install curl/wget if not present
    if command -v apt-get >/dev/null 2>&1; then
        if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
            apt-get update -qq
            apt-get install -y curl
        fi
    elif command -v yum >/dev/null 2>&1; then
        if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
            yum install -y curl
        fi
    elif command -v dnf >/dev/null 2>&1; then
        if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
            dnf install -y curl
        fi
    fi
}

# Create minimal directory structure (only if needed)
setup_directories() {
    log "Setting up minimal directory structure..."
    
    # Only create directories that the CLI tool needs for configuration storage
    mkdir -p /etc/webstack
    chmod 755 /etc/webstack
    
    log "Configuration directory created at /etc/webstack"
}

# Main installation function
main() {
    info "WebStack CLI - Minimal Installation"
    info "==================================="
    info "Installing CLI tool only (no system service)"
    info ""
    
    check_root
    install_dependencies
    setup_directories
    install_binary
    
    log "WebStack CLI installed successfully!"
    log ""
    log "Quick Start:"
    log "  sudo webstack install all              # Install complete web stack"
    log "  sudo webstack domain add example.com   # Add a domain"
    log "  sudo webstack ssl enable example.com   # Enable SSL"
    log ""
    log "Get Help:"
    log "  webstack --help                        # Show all commands"
    log "  webstack version                       # Show version info"
    log ""
    log "Updates:"
    log "  sudo webstack update                   # Update to latest version"
    log ""
    log "Note: This is a CLI-only installation."
    log "   Use the full installer if you need service integration."
}

# Run main function
main "$@"