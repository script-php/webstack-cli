#!/bin/bash
# WebStack CLI - APT Repository Setup Script

set -e

REPO_KEY_URL="https://your-domain.com/gpg-key.asc"
REPO_URL="https://your-domain.com/apt"
REPO_NAME="webstack"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${GREEN}[REPO]${NC} $1"; }
info() { echo -e "${BLUE}[INFO]${NC} $1"; }

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    echo "This script must be run as root (use sudo)"
    exit 1
fi

# Detect OS
if [[ ! -f /etc/os-release ]]; then
    echo "Cannot detect OS. /etc/os-release not found."
    exit 1
fi

source /etc/os-release

# Check if supported OS
case "$ID" in
    ubuntu|debian)
        log "Detected supported OS: $PRETTY_NAME"
        ;;
    *)
        echo "Unsupported OS: $PRETTY_NAME"
        echo "Supported: Ubuntu, Debian"
        exit 1
        ;;
esac

# Install prerequisites
log "Installing prerequisites..."
apt-get update -qq
apt-get install -y curl gnupg lsb-release

# Add GPG key
log "Adding repository GPG key..."
curl -fsSL "$REPO_KEY_URL" | gpg --dearmor -o /usr/share/keyrings/webstack-archive-keyring.gpg

# Add repository
log "Adding WebStack repository..."
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/webstack-archive-keyring.gpg] $REPO_URL $(lsb_release -cs) main" > /etc/apt/sources.list.d/webstack.list

# Update package list
log "Updating package list..."
apt-get update -qq

# Install WebStack CLI
log "Installing WebStack CLI..."
apt-get install -y webstack-cli

log "WebStack CLI installed successfully!"
log "Usage: sudo webstack --help"