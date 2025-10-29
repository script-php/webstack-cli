#!/bin/bash
# WebStack - Complete System Installation Script
# Installs WebStack CLI with all system integration
# Foundation for CLI, service management, and future web control panel
# Usage: curl -fsSL https://your-domain.com/install.sh | sudo bash

set -e

# Configuration
REPO_URL="https://github.com/script-php/webstack-cli"
BINARY_NAME="webstack"
INSTALL_DIR="/usr/local/bin"
SERVICE_DIR="/etc/systemd/system"
CONFIG_DIR="/etc/webstack"
LOG_DIR="/var/log/webstack"
WEB_ROOT="/var/www"
SHARE_DIR="/usr/local/share/webstack"
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

# Install system dependencies
install_dependencies() {
    log "Installing system dependencies..."
    
    # Detect package manager
    if command -v apt-get >/dev/null 2>&1; then
        apt-get update -qq
        apt-get install -y curl wget software-properties-common apt-transport-https ca-certificates gnupg lsb-release
    elif command -v yum >/dev/null 2>&1; then
        yum update -y
        yum install -y curl wget
    elif command -v dnf >/dev/null 2>&1; then
        dnf update -y
        dnf install -y curl wget
    else
        warn "Unknown package manager. Please ensure curl and wget are installed."
    fi
}

# Create initial directory structure
setup_directories() {
    log "Setting up system directories..."
    
    # Create all required directories
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$LOG_DIR"
    mkdir -p "$WEB_ROOT"
    mkdir -p "$SHARE_DIR"
    mkdir -p /etc/nginx/sites-{available,enabled}
    mkdir -p /etc/apache2/sites-{available,enabled}
    mkdir -p /var/cache/nginx/fastcgi
    mkdir -p /etc/ssl/webstack
    mkdir -p /var/log/letsencrypt
    
    # Set proper ownership and permissions
    chown -R www-data:www-data "$WEB_ROOT"
    chmod 755 "$CONFIG_DIR"
    chmod 755 "$LOG_DIR"
    chmod 755 "$SHARE_DIR"
    chmod 755 /etc/ssl/webstack
    
    # Create initial log file
    touch "$LOG_DIR/webstack.log"
    chown root:root "$LOG_DIR/webstack.log"
    chmod 644 "$LOG_DIR/webstack.log"
    
    log "✓ Directories created with proper permissions"
}

# Ask if user wants to install as service
ask_service_install() {
    echo ""
    read -p "Do you want to install WebStack CLI as a system service for automatic management? (y/N): " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        install_service
    else
        log "WebStack CLI installed as command-line tool only"
    fi
}

# Install systemd service (optional)
install_service() {
    log "Installing WebStack as system service..."
    
    # Create service file
    cat > "$SERVICE_DIR/webstack.service" << 'EOF'
[Unit]
Description=WebStack System Service
Documentation=https://github.com/script-php/webstack-cli
After=network.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/bin/true
ExecReload=/usr/local/bin/webstack reload --quiet
ExecStop=/bin/true
User=root
Group=root
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

    # Create logrotate configuration
    cat > /etc/logrotate.d/webstack << 'EOF'
/var/log/webstack/*.log {
    daily
    missingok
    rotate 52
    compress
    delaycompress
    notifempty
    create 644 root root
    postrotate
        systemctl reload webstack.service >/dev/null 2>&1 || true
    endscript
}
EOF

    # Enable service
    systemctl daemon-reload
    systemctl enable webstack.service
    
    log "✓ WebStack service installed and enabled"
}

# Install logrotate configuration for Certbot logs
install_logrotate_letsencrypt() {
    log "Setting up log rotation for SSL certificates..."
    
    cat > /etc/logrotate.d/letsencrypt << 'EOF'
/var/log/letsencrypt/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 root root
}
EOF

    log "✓ Log rotation configured"
}

# Main installation function
main() {
    info "WebStack System Installation"
    info "============================"
    
    check_root
    install_dependencies
    setup_directories
    install_binary
    install_logrotate_letsencrypt
    
    # Ask about service installation
    ask_service_install
    
    log "Installation completed successfully!"
    log ""
    log "Next Steps:"
    log "  1. sudo webstack install all              # Install web stack components"
    log "  2. sudo webstack domain add example.com   # Add your first domain"
    log "  3. sudo webstack ssl enable example.com   # Enable SSL with auto-renewal"
    log ""
    log "Management Commands:"
    log "  webstack --help                           # Show all available commands"
    log "  webstack version                          # Display version information"
    log "  webstack ssl autorenew status             # Check SSL renewal status"
    log ""
    log "System Service:"
    if systemctl is-enabled webstack.service >/dev/null 2>&1; then
        log "  systemctl status webstack                 # Check service status"
        log "  systemctl reload webstack                 # Reload configurations"
    fi
    log ""
    log "Logs:"
    log "  tail -f $LOG_DIR/webstack.log             # View WebStack logs"
    log "  journalctl -u webstack -f                 # View systemd logs"
    log ""
    log "Documentation:"
    log "  https://github.com/script-php/webstack-cli"
}

# Run main function
main "$@"