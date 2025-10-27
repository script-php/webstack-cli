#!/bin/bash
# WebStack CLI - Service Installation Script
# Usage: curl -fsSL https://your-domain.com/install-service.sh | sudo bash

set -e

BINARY_NAME="webstack"
INSTALL_DIR="/usr/local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() { echo -e "${GREEN}[SERVICE]${NC} $1"; }
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

# Check if WebStack CLI is installed
check_binary() {
    if [[ ! -f "${INSTALL_DIR}/${BINARY_NAME}" ]]; then
        error "WebStack CLI not found at ${INSTALL_DIR}/${BINARY_NAME}"
        error "Please install WebStack CLI first:"
        error "  curl -fsSL https://your-domain.com/install-cli.sh | sudo bash"
        exit 1
    fi
    log "Found WebStack CLI at ${INSTALL_DIR}/${BINARY_NAME}"
}

# Install systemd service
install_systemd_service() {
    log "Installing WebStack CLI systemd service..."
    
    cat > /etc/systemd/system/webstack.service << 'EOF'
[Unit]
Description=WebStack CLI Management Service
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

# Security settings
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/etc/webstack /var/www /var/log/webstack
PrivateTmp=yes

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable webstack.service
    log "WebStack systemd service installed and enabled"
}

# Install logrotate configuration
install_logrotate() {
    log "Installing logrotate configuration..."
    
    cat > /etc/logrotate.d/webstack << 'EOF'
/var/log/webstack/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 root root
    postrotate
        systemctl reload webstack.service >/dev/null 2>&1 || true
    endscript
}
EOF

    log "Logrotate configuration installed"
}

# Install cron jobs for maintenance
install_cron_jobs() {
    log "Installing maintenance cron jobs..."
    
    # Create cron file for WebStack maintenance
    cat > /etc/cron.d/webstack << 'EOF'
# WebStack CLI maintenance tasks
SHELL=/bin/bash
PATH=/usr/local/bin:/usr/bin:/bin

# SSL certificate renewal (daily at 2:30 AM)
30 2 * * * root /usr/local/bin/webstack ssl renew --quiet >/dev/null 2>&1

# Configuration validation (weekly on Sunday at 3:00 AM)
0 3 * * 0 root /usr/local/bin/webstack validate --quiet >/dev/null 2>&1

# Cleanup temporary files (daily at 4:00 AM)
0 4 * * * root /usr/local/bin/webstack cleanup --quiet >/dev/null 2>&1
EOF

    log "Maintenance cron jobs installed"
}

# Setup monitoring and alerting
setup_monitoring() {
    log "Setting up basic monitoring..."
    
    # Create monitoring script
    mkdir -p /usr/local/share/webstack
    
    cat > /usr/local/share/webstack/monitor.sh << 'EOF'
#!/bin/bash
# WebStack CLI monitoring script

LOGFILE="/var/log/webstack/monitor.log"
DATE=$(date '+%Y-%m-%d %H:%M:%S')

# Function to log messages
log_message() {
    echo "[$DATE] $1" >> "$LOGFILE"
}

# Check service status
check_services() {
    for service in nginx apache2 mysql postgresql; do
        if systemctl is-active --quiet "$service" 2>/dev/null; then
            log_message "✓ $service is running"
        else
            log_message "✗ $service is not running"
        fi
    done
}

# Check SSL certificates
check_ssl() {
    /usr/local/bin/webstack ssl status --quiet >> "$LOGFILE" 2>&1
}

# Run checks
check_services
check_ssl

log_message "Monitoring check completed"
EOF

    chmod +x /usr/local/share/webstack/monitor.sh
    
    # Add monitoring to cron
    cat >> /etc/cron.d/webstack << 'EOF'

# System monitoring (every 6 hours)
0 */6 * * * root /usr/local/share/webstack/monitor.sh
EOF

    log "Basic monitoring setup completed"
}

# Create service management commands
create_service_commands() {
    log "Creating service management helpers..."
    
    # Add reload command to WebStack CLI
    cat > /usr/local/bin/webstack-reload << 'EOF'
#!/bin/bash
# WebStack CLI reload helper

echo "Reloading WebStack configurations..."

# Reload nginx if running
if systemctl is-active --quiet nginx; then
    systemctl reload nginx
    echo "✓ Nginx reloaded"
fi

# Reload apache if running
if systemctl is-active --quiet apache2; then
    systemctl reload apache2
    echo "✓ Apache reloaded"
fi

# Reload PHP-FPM pools
for php_service in $(systemctl list-units --type=service --state=active | grep 'php.*-fpm' | awk '{print $1}'); do
    systemctl reload "$php_service"
    echo "✓ $php_service reloaded"
done

echo "Configuration reload completed"
EOF

    chmod +x /usr/local/bin/webstack-reload
    log "Service management helpers created"
}

# Setup directories with proper permissions
setup_service_directories() {
    log "Setting up service directories..."
    
    # Create all required directories
    mkdir -p /etc/webstack
    mkdir -p /var/www
    mkdir -p /var/log/webstack
    mkdir -p /etc/nginx/sites-enabled
    mkdir -p /etc/apache2/sites-enabled
    mkdir -p /usr/local/share/webstack
    
    # Set proper ownership and permissions
    chown -R www-data:www-data /var/www
    chmod 755 /etc/webstack
    chmod 755 /var/log/webstack
    chmod 755 /usr/local/share/webstack
    
    # Create initial log file
    touch /var/log/webstack/webstack.log
    chown root:root /var/log/webstack/webstack.log
    chmod 644 /var/log/webstack/webstack.log
    
    log "Service directories configured"
}

# Main installation function
main() {
    info "WebStack CLI - Service Installation"
    info "===================================="
    info "Installing service integration for WebStack CLI"
    info ""
    
    check_root
    check_binary
    setup_service_directories
    install_systemd_service
    install_logrotate
    install_cron_jobs
    setup_monitoring
    create_service_commands
    
    # Start the service
    systemctl start webstack.service
    
    log "WebStack CLI service installation completed!"
    log ""
    log "Service Management:"
    log "  systemctl status webstack              # Check service status"
    log "  systemctl reload webstack              # Reload configurations"
    log "  systemctl restart webstack             # Restart service"
    log ""
    log "Monitoring:"
    log "  tail -f /var/log/webstack/webstack.log # View logs"
    log "  tail -f /var/log/webstack/monitor.log  # View monitoring logs"
    log ""
    log "Maintenance:"
    log "  SSL certificates will auto-renew daily"
    log "  System monitoring runs every 6 hours"
    log "  Configuration validation runs weekly"
    log ""
    log "WebStack CLI is now integrated as a system service!"
}

# Run main function
main "$@"