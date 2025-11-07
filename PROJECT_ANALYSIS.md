# WebStack CLI - Comprehensive Project Analysis

## Executive Summary

**WebStack CLI** is a lightweight, CLI-first management tool for complete web development stacks on Linux. Built in Go, it provides a single binary (~13MB) for installing, configuring, and managing production web servers, databases, DNS, SSL certificates, backups, and firewall security—without the bloat of traditional web panels like Hestia Control Panel.

**Philosophy**: *"Copy one file and everything works"*

---

## What It Is

### Core Purpose
WebStack CLI is a **web infrastructure automation tool** that:
- Installs and configures web servers (Nginx, Apache)
- Manages databases (MySQL/MariaDB, PostgreSQL)
- Handles DNS services (Bind9 master/slave with clustering)
- Automates SSL certificate management (Let's Encrypt, self-signed)
- Provides enterprise-grade backup/restore with scheduling
- Implements production-grade security (iptables, ipset, fail2ban)
- Manages domains and PHP versions
- Provides firewall automation with port management
- Offers cron job discovery and management

### Technology Stack
- **Language**: Go 1.25.3
- **CLI Framework**: Cobra (spf13)
- **Target OS**: Linux (Ubuntu/Debian recommended)
- **Requirements**: Root privileges only
- **Binary Size**: ~13MB (fully statically linked)
- **Code Size**: ~13,000 lines of Go code
- **Templates**: 26 embedded configuration files

---

## Architecture Overview

### Directory Structure

```
webstack-cli/
├── main.go                          # Root entry point (checks for root)
├── go.mod / go.sum                  # Dependency management
├── Makefile                         # Multi-platform build system
├── README.md                        # Comprehensive documentation (761 lines)
├── QUICK_REFERENCE.md               # Command reference cheat sheet
├── build/                           # Compiled binaries
├── cmd/                             # CLI command implementations
│   ├── root.go                      # Root command setup
│   ├── install.go                   # install all|nginx|apache|mysql|mariadb|postgresql|php
│   ├── domain.go                    # domain add|edit|delete|list
│   ├── ssl.go                       # ssl enable|disable|renew|status
│   ├── backup.go                    # backup create|restore|list|verify|export|import|delete
│   ├── cron.go                      # cron add|edit|delete|list|enable|disable|run|status
│   ├── dns.go                       # dns install|uninstall|status|restart|check|config|zones
│   ├── firewall.go                  # firewall open|close|block|unblock|status|save|load
│   ├── system.go                    # system remote-access|reload|cleanup|validate
│   ├── db.go                        # database operations
│   ├── config.go                    # Configuration management
│   ├── menu.go                      # Display component status menu
│   ├── phpmyadmin.go                # PhpMyAdmin management
│   ├── version.go                   # Version information
│   ├── uninstall.go                 # Component uninstall
│   └── other supporting commands
│
├── internal/                        # Core functionality (non-exported)
│   ├── installer/                   # Installation logic (~2200 lines)
│   │   └── installer.go             # Component detection, installation, status checks
│   │
│   ├── templates/                   # Configuration templates (embedded)
│   │   ├── templates.go             # Template embedding & retrieval
│   │   ├── nginx/                   # Nginx config templates
│   │   ├── apache/                  # Apache config templates
│   │   ├── mysql/                   # MySQL config templates
│   │   ├── php-fpm/                 # PHP-FPM config templates
│   │   ├── dns/                     # Bind9 DNS config templates
│   │   ├── error/                   # Error page templates (403, 404, 50x)
│   │   ├── web/                     # General web templates
│   │   ├── security/                # Fail2Ban, iptables security
│   │   └── [26 template files total]
│   │
│   ├── backup/                      # Backup system
│   │   ├── backup.go                # Main backup logic
│   │   ├── archiver.go              # Compression and archiving
│   │   ├── database.go              # Database dump handling
│   │   └── schedule.go              # Cron scheduling
│   │
│   ├── domain/                      # Domain management
│   │   └── domain.go                # Domain CRUD operations
│   │
│   ├── ssl/                         # SSL certificate management
│   │   └── ssl.go                   # Let's Encrypt and self-signed certs
│   │
│   ├── cron/                        # Cron job management
│   │   └── cron.go                  # Cron operations
│   │
│   ├── config/                      # Configuration file handling
│   │   ├── config.go                # Main config logic
│   │   └── template.go              # Template rendering
│   │
│   └── installer/                   # Installation logic
│       └── installer.go             # Core installer implementation
│
└── template and configs from hestia/  # Reference configs (not used)
```

---

## What It Does

### 1. **Web Server Management**

#### Nginx
- Install with automatic port 80/443 setup
- Direct PHP-FPM processing
- Automatic firewall port opening on install
- Automatic firewall port closing on uninstall
- Template-based virtual host configuration

#### Apache
- Install for backend deployment
- Nginx reverse proxy setup (Apache on port 8080)
- Automatic port management
- Integration with Nginx proxy

### 2. **Database Management**

#### MySQL/MariaDB
- Installation with version selection
- Remote access enable/disable
- Automatic port 3306 management
- Remote access control via firewall
- Configuration templates

#### PostgreSQL
- Installation with version selection
- Remote access enable/disable
- Automatic port 5432 management
- Configuration templates

### 3. **DNS Server (Bind9)**

- **Master/Slave Replication**: Full DNS replication between servers
- **Clustering**: Multi-server DNS clusters with automatic sync
- **DNSSEC Support**: Optional DNSSEC validation
- **Query Logging**: Optional detailed query logging
- **Zone Management**: Easy zone configuration
- **Automatic Port Management**: Port 53 TCP/UDP auto-managed
- Bind9 installation with templates

### 4. **PHP-FPM**

- Support for PHP 5.6 through 8.4
- Multiple simultaneous versions
- Isolated FPM pool configurations per version
- Version-specific socket/port setup

### 5. **Domain Management**

- Add new domains with backend selection
- Edit domain configuration (backend, PHP version)
- Delete domains with cleanup
- List all configured domains
- Automatic web root creation
- Automatic SSL integration

### 6. **SSL Certificate Management**

- **Let's Encrypt**: Automatic provisioning and renewal
- **Self-Signed**: Generate self-signed certificates on demand
- **Certificate Renewal**: Manual or automatic via cron
- **Status Checking**: View certificate info and expiry dates
- **Domain-Specific**: Per-domain certificate management

### 7. **Firewall & Security**

#### Core Security Infrastructure (auto-installed once)
- **iptables**: Kernel firewall engine
- **iptables-persistent**: Rules survive system reboots
- **ipset**: O(1) lookup for IP blocking (100K+ IPs)
- **fail2ban**: Automatic brute-force protection

#### Port Management
- Automatic opening/closing on component install/uninstall
- Manual port control on demand
- Support for TCP, UDP, or both
- IPv4 & IPv6 dual-stack support

#### IP Blocking
- Block/unblock individual IPs
- View all blocked IPs
- ipset integration for efficient lookup
- Persistent across reboots

#### Fail2Ban
- SSH protection (port 22)
- Auto-ban after 5 failures in 10 minutes
- 1-hour ban duration
- Automatic integration with all services

#### UFW Auto-Removal
- Automatically removes UFW if present
- Prevents firewall conflicts
- Clean iptables-only setup

### 8. **Backup & Restore System** (Enterprise-Grade)

#### Full System Backups
- All domains and their configurations
- Database dumps (MySQL and PostgreSQL)
- SSL certificates
- Web server configs (Nginx, Apache)
- Firewall rules and ipset lists
- All metadata (domains.json, ssl.json)

#### Backup Options
- Full system backup (`--all`)
- Single domain backup (`--domain example.com`)
- Database-only backup (`--database mysql:wordpress`)
- Compression selection: gzip, bzip2, xz, none
- Compression level control

#### Backup Verification
- SHA256 checksums on all backups
- Metadata JSON with timestamps and contents
- Pre-restore integrity verification
- Checksum validation

#### Restore Operations
- Full system restore from backup
- Domain-only restore (`--domain example.com`)
- Database-only restore
- Dry-run mode (verify without restoring)
- Staging extraction for safety
- Force mode for automation

#### Backup Management
- List all backups with metadata
- Filter by date range (`--since 7d`, `--since 3m`)
- Delete old backups
- View storage usage
- Export backups to external media
- Import backups from other servers

#### Scheduled Backups
- Daily automatic backups with configurable time
- Retention policy (e.g., keep 30 days)
- Enable/disable scheduling
- Systemd timer integration
- Automatic cleanup of old backups

#### Storage
- Location: `/var/backups/webstack/archives/`
- Format: Compressed `.tar.gz` with SHA256 checksums
- Metadata: JSON files with backup contents
- Typical size: ~25 MB per backup

### 9. **Cron Job Management**

#### Manual Cron Operations
- Add new cron jobs with schedule and command
- Edit existing crons (schedule or command)
- Delete cron jobs permanently
- Enable/disable crons without deletion

#### Auto-Discovery
- Discovers manual crons created via `webstack cron add`
- Discovers backup system cleanup crons
- Discovers SSL certificate renewal timers
- Discovers systemd timers with `webstack-*` prefix

#### Cron Features
- Run jobs immediately for testing
- View execution logs
- Detailed status for all crons
- Shows cron type (webstack/custom/auto-discovered)

#### Standard Cron Schedule
- 5-field format: `minute hour day month weekday`
- Examples: `0 0 * * *` (daily), `0 * * * *` (hourly), `*/15 * * * *` (every 15 min)

### 10. **System Management**

#### Remote Access Control
- Enable/disable database remote access
- MySQL: port 3306
- PostgreSQL: port 5432
- Firewall automatic integration

#### System Validation
- Check all component configurations
- Verify all services are running
- Configuration syntax checking
- File permission verification

#### System Cleanup
- Remove temporary files
- Clean cache directories
- Purge old logs
- Free disk space

#### System Reload
- Reload all service configurations
- Without restarting services
- Apply pending changes

#### Component Status Menu
- View all installed components
- See which are running vs stopped
- PHP version status display
- Color-coded output (green/red)

### 11. **Package Management**

#### Installation Detection
- Checks if components already installed
- Prompts for action (keep/reinstall/remove/skip)
- Prevents duplicate installations
- Clean upgrade path

#### Automatic Installation
- `install all`: Interactive full stack setup
- Component-by-component installation
- Version selection where available
- Dependency management

#### Uninstall
- Component-specific uninstall
- Automatic port closing on firewall
- Config cleanup
- "Nuclear" option for complete removal

---

## Key Features Summary

| Feature | Status | Details |
|---------|--------|---------|
| **Multi-Platform Build** | ✅ Complete | Linux AMD64/ARM64/ARM, macOS, Windows |
| **Static Binary** | ✅ Complete | Single 13MB file, no dependencies |
| **Root-Only Execution** | ✅ Complete | Security check at startup |
| **Nginx Installation** | ✅ Complete | Ports 80/443, PHP-FPM integration |
| **Apache Installation** | ✅ Complete | Nginx reverse proxy setup |
| **MySQL/MariaDB** | ✅ Complete | Version selection, remote access |
| **PostgreSQL** | ✅ Complete | Version selection, remote access |
| **PHP Versions 5.6-8.4** | ✅ Complete | Multiple simultaneous versions |
| **Domain Management** | ✅ Complete | Add/edit/delete/list domains |
| **SSL (Let's Encrypt)** | ✅ Complete | Auto-provisioning and renewal |
| **SSL (Self-Signed)** | ✅ Complete | On-demand generation |
| **DNS (Bind9)** | ✅ Complete | Master/slave, clustering, DNSSEC |
| **Firewall (iptables)** | ✅ Complete | Automatic and manual port management |
| **IP Blocking (ipset)** | ✅ Complete | O(1) blocking, 100K+ IP support |
| **Brute-Force Protection** | ✅ Complete | Fail2Ban for SSH and services |
| **UFW Auto-Removal** | ✅ Complete | Prevents firewall conflicts |
| **Backup Creation** | ✅ Complete | Full system, domains, databases |
| **Backup Restoration** | ✅ Complete | Full or selective restore |
| **Backup Scheduling** | ✅ Complete | Daily with retention management |
| **Backup Verification** | ✅ Complete | SHA256 checksums, integrity check |
| **Backup Export/Import** | ✅ Complete | Move between servers |
| **Cron Management** | ✅ Complete | Add/edit/delete/list/enable/disable |
| **Cron Auto-Discovery** | ✅ Complete | Backup, SSL, systemd timers |
| **Service Status Menu** | ✅ Complete | Color-coded component status |
| **Component Detection** | ✅ Complete | Pre-install checks |
| **Configuration Templates** | ✅ Complete | 26 embedded templates |
| **Error Pages** | ✅ Complete | 403, 404, 50x templates |
| **Security Infrastructure** | ✅ Complete | iptables, ipset, fail2ban |

---

## Command Reference

### Installation
```bash
sudo webstack install all                    # Full stack install
sudo webstack install nginx                  # Nginx only
sudo webstack install apache                 # Apache only
sudo webstack install mysql                  # MySQL only
sudo webstack install mariadb                # MariaDB only
sudo webstack install postgresql             # PostgreSQL only
sudo webstack install php 8.2                # PHP-FPM version
```

### Domain Management
```bash
sudo webstack domain add example.com --backend nginx --php 8.2
sudo webstack domain edit example.com --backend apache --php 8.3
sudo webstack domain delete example.com
sudo webstack domain list
```

### SSL Management
```bash
sudo webstack ssl enable example.com --email admin@example.com --type letsencrypt
sudo webstack ssl enable example.com --email admin@example.com --type selfsigned
sudo webstack ssl disable example.com
sudo webstack ssl renew
sudo webstack ssl status
```

### Backup Management
```bash
sudo webstack backup create --all
sudo webstack backup create --domain example.com
sudo webstack backup list
sudo webstack backup verify backup-id
sudo webstack backup restore backup-id
sudo webstack backup export backup-id /path/to/export.tar.gz
sudo webstack backup schedule enable --time 02:00 --keep 30
```

### Cron Management
```bash
sudo webstack cron add "0 3 * * *" "sudo webstack ssl renew"
sudo webstack cron edit 2 "0 2 * * *" "sudo webstack ssl renew"
sudo webstack cron delete 2
sudo webstack cron list
sudo webstack cron enable 2
sudo webstack cron disable 2
```

### DNS Management
```bash
sudo webstack dns install --mode master
sudo webstack dns install --mode slave --master-ip 192.168.1.10
sudo webstack dns config --zone example.com --type master
sudo webstack dns status
sudo webstack dns restart
```

### Firewall Management
```bash
sudo webstack firewall open 8080 tcp
sudo webstack firewall close 8080 tcp
sudo webstack firewall block 192.168.1.100
sudo webstack firewall unblock 192.168.1.100
sudo webstack firewall status
sudo webstack firewall save
sudo webstack firewall restore
```

### System Management
```bash
sudo webstack system remote-access enable mysql root password
sudo webstack system remote-access status mysql
sudo webstack system remote-access disable mysql
sudo webstack system validate
sudo webstack system cleanup
sudo webstack system reload
```

### Other Commands
```bash
webstack menu                               # Display status menu
webstack version                            # Show version info
sudo webstack uninstall                     # Uninstall components
```

---

## Technical Specifications

### Performance
- **Binary Size**: 13 MB (statically linked)
- **Memory Footprint**: Minimal (Go runtime only)
- **Startup Time**: <100ms
- **Code Complexity**: ~13,000 lines of Go code
- **Dependency Count**: 1 (Cobra CLI framework only)

### Scalability
- **ipset Blocking**: O(1) lookup for 100,000+ IPs
- **Backup Size**: Typical ~25 MB per backup
- **Concurrent Operations**: Sequential execution (safe)
- **Network Scale**: Multi-server DNS clustering support

### Security
- **Root Check**: Enforced at startup
- **Firewall**: Dual-stack IPv4 & IPv6
- **SSH Protection**: Always protected, never locked out
- **Fail2Ban**: 5-strike auto-ban with 1-hour lockout
- **Certificate Validation**: SHA256 checksums on backups

### Compatibility
- **OS Target**: Linux (Ubuntu 20.04, 22.04, 24.04 LTS recommended)
- **Databases**: MySQL 5.7+, MariaDB 10.5+, PostgreSQL 12+
- **Web Servers**: Nginx 1.18+, Apache 2.4+
- **PHP Versions**: 5.6, 7.0-7.4, 8.0-8.4
- **DNS**: Bind9 9.11+

---

## Design Philosophy

### Core Principles
1. **CLI-First**: Everything through command line, no web UI bloat
2. **Single Binary**: One file, copy and run, no installers
3. **Zero Dependencies**: Everything statically linked
4. **Production-Ready**: Enterprise-grade security and reliability
5. **Automation-Friendly**: Scripts, Ansible, Terraform compatible
6. **Extensible**: Open-source Go code, easy customization

### Unlike Hestia Panel
- ✅ Lightweight: 13 MB vs 2 GB+ resource requirements
- ✅ Production-Compatible: Works alongside production workloads
- ✅ Automation-Ready: CLI first, no web UI dependencies
- ✅ Fast Deployment: Single binary, instant setup
- ✅ Clean Upgrades: No complex installers, atomic updates

### Problem It Solves
- Bloated control panels consuming server resources
- Web UI requirements conflicting with production workloads
- Complex dependency chains
- Hard-to-automate infrastructure
- Vendor lock-in and brittle upgrades

---

## Build & Deployment

### Build System (Makefile)
- Multi-platform builds: Linux, macOS, Windows
- Architecture support: AMD64, ARM64, ARM
- SHA256 checksum generation
- Template archiving
- Debian package generation
- Docker support

### Installation Methods
1. **From Binary**: Download and copy to `/usr/local/bin/`
2. **From Source**: `git clone` + `make build` + `make install`
3. **Shell Script**: One-line install with curl
4. **Distribution Packages**: Debian .deb packages

### Version Management
- Git-based versioning
- Build-time version embedding
- Update checking via GitHub API
- Release tagging system

---

## Use Cases

### For Developers
- Local development environment management
- Testing different PHP versions
- Domain/SSL testing
- Backup/restore testing

### For DevOps Teams
- Infrastructure automation via scripts
- Terraform/Ansible integration
- Multi-server deployments
- Backup automation and scheduling

### For Hosting Providers
- Shared hosting management
- Client domain provisioning
- Automated backups with retention
- Multi-server DNS clusters
- Cost-effective compared to control panels

### For System Administrators
- Easy server initialization
- Automated maintenance tasks
- Disaster recovery via backups
- Firewall rule management
- Security hardening

---

## Future Extensibility

### Architecture for Growth
- Modular command structure (easy to add new commands)
- Template-based configuration (easy to add new services)
- Plugin-friendly codebase
- Open-source for community contributions

### Possible Additions
- Kubernetes integration
- Docker container management
- CDN integration
- Load balancing
- Advanced monitoring
- API layer
- Web UI (optional, non-required layer)

---

## Conclusion

**WebStack CLI** is a modern, lightweight alternative to traditional web control panels. It provides enterprise-grade functionality (DNS clustering, enterprise backups, security hardening) in a single 13MB binary with zero runtime dependencies. Perfect for developers, DevOps teams, and hosting providers who want tools that work *with* their infrastructure instead of against it.

**The key innovation**: *Everything you need, nothing you don't. One file. Copy and run.*
