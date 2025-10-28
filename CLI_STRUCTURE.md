# WebStack CLI - Complete Command Structure

## Main Command Categories

```
webstack
├── install          - Install web stack components
├── uninstall        - Uninstall web stack components (NEW!)
├── domain           - Manage domains
├── ssl              - Manage SSL certificates
├── system           - System management
├── version          - Version information
└── update           - Update the tool
```

---

## Install Commands

```bash
webstack install all                    # Install complete web stack (interactive)
webstack install nginx                  # Install Nginx web server (port 80)
webstack install apache                 # Install Apache web server (port 8080)
webstack install mysql                  # Install MySQL database
webstack install mariadb                # Install MariaDB database
webstack install postgresql             # Install PostgreSQL database
webstack install php [version]          # Install PHP-FPM (5.6-8.4)
webstack install phpmyadmin             # Install phpMyAdmin (MySQL/MariaDB)
webstack install phppgadmin             # Install phpPgAdmin (PostgreSQL)
```

### Install Features
✅ Component detection before installation
✅ User choice if already installed (keep/reinstall/uninstall/skip)
✅ Interactive mode with prompts
✅ Automatic configuration from embedded templates
✅ Service management (enable/start)

---

## Uninstall Commands (NEW!)

```bash
webstack uninstall all                  # Uninstall complete web stack (with confirmations)
webstack uninstall nginx                # Uninstall Nginx
webstack uninstall apache               # Uninstall Apache
webstack uninstall mysql                # Uninstall MySQL
webstack uninstall mariadb              # Uninstall MariaDB
webstack uninstall postgresql           # Uninstall PostgreSQL
webstack uninstall php [version]        # Uninstall PHP-FPM version
webstack uninstall phpmyadmin           # Uninstall phpMyAdmin
webstack uninstall phppgadmin           # Uninstall phpPgAdmin
```

### Uninstall Features
✅ Component detection before uninstall
✅ Confirmation prompts (safety first!)
✅ Service stop and disable
✅ Graceful handling of missing components
✅ Preserves domains and SSL certificates

---

## Domain Commands

```bash
webstack domain add [domain]            # Add new domain (interactive)
webstack domain add [domain]            # Add with flags:
  -b, --backend [nginx|apache]          #   Backend type
  -p, --php [version]                   #   PHP version

webstack domain edit [domain]           # Edit existing domain
webstack domain delete [domain]         # Delete domain (keeps files)
webstack domain list                    # List all domains
webstack domain rebuild-configs         # Regenerate all configs from templates
```

### Domain Features
✅ Support for Nginx direct serving or Apache reverse proxy
✅ PHP version selection per domain
✅ Interactive configuration
✅ Persistent JSON storage
✅ Config auto-generation from templates
✅ Clean deletion (preserves document root)

---

## SSL Commands

```bash
webstack ssl enable [domain]            # Enable SSL (interactive)
webstack ssl enable [domain]            # Enable with options:
  -t, --type [selfsigned|letsencrypt]   #   Certificate type
  -e, --email [email]                   #   Email for Let's Encrypt

webstack ssl disable [domain]           # Disable SSL (reverts to HTTP)
webstack ssl renew [domain]             # Renew certificate
webstack ssl renew                      # Renew all certificates
webstack ssl status [domain]            # Check certificate status
webstack ssl status                     # Check all certificates
```

### SSL Features
✅ Self-signed certificates (development)
✅ Let's Encrypt certificates (production)
✅ Automatic local domain detection
✅ Interactive mode with choices
✅ CLI flag support for automation
✅ Certificate status reporting
✅ Config generation with proper redirect handling

---

## System Commands

```bash
webstack system reload                  # Reload all web server configs
webstack system validate                # Validate all configurations
webstack system cleanup                 # Clean temporary files and logs
webstack system status                  # Show system status
  -q, --quiet                           # Suppress output
```

### System Features
✅ Nginx reload with error checking
✅ Apache reload with validation
✅ PHP-FPM reload for all versions
✅ Configuration validation
✅ Service status checking
✅ Log rotation for large files
✅ Temporary file cleanup

---

## Version Commands

```bash
webstack version                        # Show version information
webstack update                         # Check and install updates
```

### Version Features
✅ Display current version
✅ Show build time and git commit
✅ Check GitHub for latest release
✅ Auto-download and install updates
✅ Backup before updating

---

## Command Symmetry

The CLI maintains perfect symmetry between install and uninstall:

| Feature | Install | Uninstall |
|---------|---------|-----------|
| Web Servers | ✅ nginx, apache | ✅ nginx, apache |
| Databases | ✅ mysql, mariadb, postgresql | ✅ mysql, mariadb, postgresql |
| PHP Versions | ✅ 5.6-8.4 | ✅ 5.6-8.4 |
| Interfaces | ✅ phpmyadmin, phppgadmin | ✅ phpmyadmin, phppgadmin |
| Confirmation | Interactive choice | Confirmation prompts |
| All at Once | ✅ install all | ✅ uninstall all |

---

## Data Flow Architecture

```
User Input
    ↓
CLI Command (Cobra)
    ↓
Internal Package Functions
    ├── installer/ (install/uninstall components)
    ├── domain/ (domain management)
    ├── ssl/ (certificate management)
    └── templates/ (configuration templates)
    ↓
System Operations
    ├── apt package management
    ├── systemctl service control
    ├── File I/O (configs, certs)
    └── Nginx/Apache reload
    ↓
User Feedback (stdout/stderr)
```

---

## Common Workflows

### Development Setup
```bash
# Fresh installation
sudo webstack install all

# Add local domain with SSL
sudo webstack domain add myapp.local --backend nginx --php 8.2
sudo webstack ssl enable myapp.local --type selfsigned

# Check status
sudo webstack domain list
sudo webstack ssl status
```

### Production Setup
```bash
# Install web servers only
sudo webstack install nginx
sudo webstack install apache

# Install PHP versions needed
sudo webstack install php 8.2
sudo webstack install php 7.4

# Install database
sudo webstack install mariadb

# Add production domain
sudo webstack domain add example.com --backend nginx --php 8.2
sudo webstack ssl enable example.com --type letsencrypt -e admin@example.com

# Check everything
sudo webstack system validate
sudo webstack system status
```

### Cleanup Unused Components
```bash
# Remove old PHP version
sudo webstack uninstall php 7.3

# Remove MySQL, keep MariaDB
sudo webstack uninstall mysql

# Or completely reset (preserving domains)
sudo webstack uninstall all
```

### Maintenance
```bash
# Validate configurations
sudo webstack system validate

# Reload after manual config edits
sudo webstack system reload

# Check certificate expiry
sudo webstack ssl status

# Clean old logs
sudo webstack system cleanup
```

---

## Help System

```bash
# Main help
webstack --help
webstack -h

# Command help
webstack install --help
webstack domain --help
webstack ssl --help

# Subcommand help
webstack install nginx --help
webstack domain add --help
webstack ssl enable --help
```

---

## Exit Codes

- `0` - Success
- `1` - General error or validation failed
- `2` - Missing required arguments
- `3` - Permission denied (should use sudo)

---

## Complete Feature Matrix

| Feature | Status | Details |
|---------|--------|---------|
| Web Server Installation | ✅ | Nginx, Apache with auto-config |
| Database Installation | ✅ | MySQL, MariaDB, PostgreSQL |
| PHP Versions | ✅ | 5.6-8.4 support |
| Web Interfaces | ✅ | phpMyAdmin, phpPgAdmin |
| Domain Management | ✅ | Add/edit/delete/list/rebuild |
| SSL Certificates | ✅ | Self-signed & Let's Encrypt |
| Component Uninstall | ✅ | All components removable |
| Service Management | ✅ | Reload/validate/status |
| System Cleanup | ✅ | Log rotation, temp file cleanup |
| Updates | ✅ | Auto-check and install |
| Embedded Templates | ✅ | Single binary, no external files |
| Error Handling | ✅ | Graceful errors with feedback |
| Help System | ✅ | Complete Cobra help |

All features are production-ready and fully implemented! 🚀
