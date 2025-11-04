# WebStack CLI

A comprehensive command-line tool for managing a complete web development stack on Linux systems with enterprise-grade security.

## Features

### Web Servers
- **Nginx**: Direct PHP-FPM processing on port 80/443
- **Apache**: Optional backend deployment with Nginx proxy
- **Automatic Firewall Management**: Ports 80/443 automatically opened/closed on install/uninstall

### Databases
- **MySQL/MariaDB**: Full support with remote access management
- **PostgreSQL**: Complete installation with remote access control
- **Automatic Database Ports**: Port 3306 (MySQL) / 5432 (PostgreSQL) managed by firewall
- **Remote Access Control**: Enable/disable remote connections with `sudo webstack system remote-access`

### Mail Server (Enterprise Features)
- **Exim4 SMTP**: Multiple version support (4.94, 4.95, 4.97+) with auto-detection
- **Dovecot IMAP/POP3**: Full email access with Sieve filtering
- **SpamAssassin**: Real-time spam detection with spamd socket integration
- **ClamAV**: Optional antivirus scanning for attachments
- **DKIM Signing**: Per-domain email authentication
- **DNSBL/RBL Checking**: Real-time spam list protection (SpamCop, Spamhaus, SURBL)
- **SRS (Sender Rewriting Scheme)**: Ensures SPF compliance for forwarded mail
- **SMTP Relay**: Per-domain upstream smarthost configuration
- **Automatic Mail Ports**: All 7 ports (25, 143, 110, 587, 465, 993, 995) auto-managed

### DNS Server (Bind9)
- **Master/Slave Replication**: Full master-slave DNS setup
- **DNSSEC Support**: Optional DNSSEC validation
- **Clustering**: Multi-server DNS clusters with replication
- **Query Logging**: Optional detailed query logging
- **Zone Management**: Easy zone configuration
- **Automatic DNS Ports**: Port 53 TCP/UDP auto-managed by firewall

### Security Features (Production-Ready)
- **Core Security Infrastructure**: iptables, ipset, fail2ban (auto-installed once)
- **SSH Protection**: Port 22 always protected, never locked out
- **Firewall Management**: Automatic port opening/closing with component installation
- **IPv4 & IPv6 Support**: All firewall rules support both protocols
- **Fail2Ban Integration**: Automatic brute-force protection for mail and SSH
- **ipset Blocking**: Efficient IP blocking with O(1) lookup for 100K+ IPs
- **Persistent Rules**: All firewall rules survive system reboots
- **Per-Component Security**: Each component integrates with core firewall automatically

### Firewall Management
- **Manual Port Control**: Open/close ports on demand with `webstack firewall`
- **IP Blocking**: Block/unblock malicious IP addresses
- **Rule Backup/Restore**: Save and restore firewall configurations
- **Port Status**: View all active firewall rules and statistics
- **Auto-Save Rules**: All rules persist across system reboots
- **IPv4 & IPv6**: Full support for both protocols
- **Reset Options**: Flush or restore to default configuration

### PHP Versions
- Support for PHP 5.6 to 8.4 with multiple FPM pools
- Isolated configurations per version

### Domain Management
- Add, edit, delete domains with backend selection
- Dynamic domain configuration per service

### SSL Management  
- Let's Encrypt certificate automation
- Self-signed certificate support
- Automatic certificate renewal

## Installation

### Quick Start (Recommended)

Complete system setup:
```bash
curl -fsSL https://your-domain.com/install.sh | sudo bash
```

### Manual Installation

```bash
wget https://github.com/script-php/webstack-cli/releases/latest/download/webstack-linux-amd64
chmod +x webstack-linux-amd64
sudo mv webstack-linux-amd64 /usr/local/bin/webstack
```

### Build from Source
```bash
git clone https://github.com/script-php/webstack-cli.git
cd webstack-cli
make build
sudo make install
```

## Usage

### Prerequisites
- Ubuntu/Debian Linux system (20.04, 22.04, 24.04 LTS recommended)
- Root privileges (run with sudo)

### Install Complete Stack

```bash
sudo webstack install all
```

### Install Individual Components

#### Web Servers
```bash
sudo webstack install nginx
sudo webstack install apache
```

#### Databases
```bash
sudo webstack install mysql
sudo webstack install mariadb
sudo webstack install postgresql
```

#### Mail Server (Enterprise Features)
```bash
# Install with all features
sudo webstack mail install example.com --spam --av

# Install basic mail (Exim4 + Dovecot only)
sudo webstack mail install example.com

# Check mail server status
sudo webstack mail status

# Uninstall (automatically closes firewall ports)
sudo webstack mail uninstall
```

#### DNS Server
```bash
# Master DNS server
sudo webstack dns install --mode master

# Slave DNS server (replicates from master)
sudo webstack dns install --mode slave --master-ip 192.168.1.10

# With clustering
sudo webstack dns install --mode master --cluster-name prod-cluster

# Uninstall (automatically closes DNS port 53)
sudo webstack dns uninstall
```

#### PHP Versions
```bash
sudo webstack install php 8.2
sudo webstack install php 7.4
```

### Domain Management

```bash
# Add a domain
sudo webstack domain add example.com

# With specific backend and PHP version
sudo webstack domain add example.com --backend nginx --php 8.2

# Edit domain
sudo webstack domain edit example.com --backend apache --php 8.3

# List all domains
sudo webstack domain list

# Delete domain
sudo webstack domain delete example.com
```

### SSL Management

```bash
# Enable Let's Encrypt SSL
sudo webstack ssl enable example.com --email admin@example.com --type letsencrypt

# Enable self-signed SSL
sudo webstack ssl enable example.com --email admin@example.com --type selfsigned

# Disable SSL
sudo webstack ssl disable example.com

# Renew specific certificate
sudo webstack ssl renew example.com

# Renew all certificates
sudo webstack ssl renew

# Check SSL status
sudo webstack ssl status example.com
sudo webstack ssl status  # All domains
```

### Mail Server Management

#### Add Mail Users
```bash
sudo webstack mail add user@example.com
sudo webstack mail delete user@example.com
sudo webstack mail list example.com
```

#### Check Mail Status
```bash
sudo webstack mail status
```

#### Mail Features
- **Spam Detection**: Emails automatically scored by SpamAssassin
  - View scores: `tail -f /var/log/exim4/mainlog | grep spam`
- **Antivirus Scanning**: Optional ClamAV integration
  - Enable: Add `--av` flag during install
- **DKIM Signing**: Automatic per-domain
  - Public key location: `/etc/exim4/domains/[domain]/dkim.pem`
- **Fail2Ban Protection**: Auto-bans after 5 failed login attempts
  - Check bans: `sudo fail2ban-client status exim4`

### DNS Server Management

#### Service Control
```bash
sudo webstack dns status
sudo webstack dns restart
sudo webstack dns reload
sudo webstack dns check
```

#### Zone Management
```bash
sudo webstack dns zones
sudo webstack dns config --zone example.com --type master
sudo webstack dns config --zone example.com --type slave
sudo webstack dns config --add-slave 192.168.1.20
sudo webstack dns config --remove-slave 192.168.1.20
```

#### Advanced Features
```bash
# Enable DNSSEC validation
sudo webstack dns dnssec --enable

# Enable query logging
sudo webstack dns querylog --enable

# Backup configuration
sudo webstack dns backup

# Restore from backup
sudo webstack dns restore /tmp/dns-backup-20251103_012855.tar.gz

# Test DNS query
sudo webstack dns query example.com
```

### Security & Firewall Management

#### Firewall Management Commands
```bash
# View all firewall rules
sudo webstack firewall status

# Open a specific port
sudo webstack firewall open 8080 tcp       # Open TCP port 8080
sudo webstack firewall open 5353 udp       # Open UDP port 5353
sudo webstack firewall open 9000 both      # Open both TCP and UDP

# Close a specific port
sudo webstack firewall close 8080 tcp
sudo webstack firewall close 5353 both

# Block/Unblock IP addresses
sudo webstack firewall block 192.168.1.100
sudo webstack firewall unblock 192.168.1.100
sudo webstack firewall blocked              # List all blocked IPs

# Firewall rules management
sudo webstack firewall save                 # Backup firewall rules
sudo webstack firewall load /path/to/backup # Restore from backup
sudo webstack firewall flush                # Remove custom rules (keeps SSH)
sudo webstack firewall restore              # Restore default config
sudo webstack firewall stats                # Show rule statistics
```

#### System Security Setup
```bash
# Core security is auto-installed by first component
# No manual action needed, but can verify:
sudo iptables -L -n | grep "dpt:22"  # SSH always open
sudo fail2ban-client status          # Check Fail2Ban jails
```

#### Database Remote Access Management
```bash
# Enable remote access for MySQL/MariaDB
sudo webstack system remote-access enable mysql root password

# Disable remote access
sudo webstack system remote-access disable mysql

# Check status
sudo webstack system remote-access status mysql

# Same for PostgreSQL
sudo webstack system remote-access enable postgresql postgres password
sudo webstack system remote-access disable postgresql
sudo webstack system remote-access status postgresql
```

#### Firewall Rules Status
```bash
# View all firewall rules
sudo iptables -L -n

# View specific port
sudo iptables -L -n | grep "dpt:80"

# Check persistent rules
sudo cat /etc/iptables/rules.v4

# Check Fail2Ban status
sudo fail2ban-client status
sudo fail2ban-client status exim4
sudo fail2ban-client status dovecot

# View banned IPs
sudo ipset list banned_ips
```

## Firewall & Security Architecture

### Automatic Port Management

When you install components, ports are **automatically opened**:

| Component | Ports | Action |
|-----------|-------|--------|
| **Nginx** | 80, 443 | Auto-open on install, auto-close on uninstall |
| **Apache** | 80, 443 | Auto-open on install, auto-close on uninstall |
| **Mail Server** | 25, 143, 110, 587, 465, 993, 995 | Auto-open on install, auto-close on uninstall |
| **DNS (Bind9)** | 53 (TCP/UDP) | Auto-open on install, auto-close on uninstall |
| **MySQL/MariaDB** | 3306 | Auto-open when remote access enabled, auto-close when disabled |
| **PostgreSQL** | 5432 | Auto-open when remote access enabled, auto-close when disabled |
| **SSH** | 22 | Always open (protected by Fail2Ban) |

### Manual Firewall Management

Use the `webstack firewall` command to manually manage ports and IP blocking:

```bash
# View current firewall status
sudo webstack firewall status

# Open custom ports
sudo webstack firewall open 8080 tcp          # TCP only
sudo webstack firewall open 5353 udp          # UDP only
sudo webstack firewall open 9000 both         # Both TCP and UDP

# Close ports
sudo webstack firewall close 8080 tcp
sudo webstack firewall close 9000 both

# Block malicious IPs
sudo webstack firewall block 192.168.1.100    # Add to blocklist
sudo webstack firewall unblock 192.168.1.100  # Remove from blocklist
sudo webstack firewall blocked                # Show all blocked IPs

# Backup and restore rules
sudo webstack firewall save                   # Backup to /etc/webstack/firewall-backup.tar.gz
sudo webstack firewall load /path/to/backup   # Restore from backup
sudo webstack firewall stats                  # Show rule statistics

# Reset firewall
sudo webstack firewall flush                  # Remove custom rules (SSH preserved)
sudo webstack firewall restore                # Restore default configuration
```

### Three-Layer Security Model

```
LAYER 1: Core Infrastructure (System-Level)
  ├─ iptables        (Kernel firewall engine)
  ├─ iptables-persistent (Persist rules across reboots)
  ├─ ipset           (Efficient IP list management - O(1) lookup)
  └─ fail2ban        (Automatic brute-force protection)
         ▲
         │ Shared by all components
         │ ⚠️ UFW automatically removed (conflicts with iptables)
         │
LAYER 2: Component-Specific
  ├─ Mail (Exim4, Dovecot, SpamAssassin)
  ├─ DNS (Bind9)
  ├─ Web (Nginx, Apache)
  └─ Database (MySQL, PostgreSQL)
         ▲
         │
LAYER 3: Component Configuration
  ├─ Fail2Ban jails per service
  ├─ iptables rules per service
  └─ ipset lists per service
```

### Fail2Ban Integration

Automatic brute-force protection for:

```
Mail:
  ├─ exim4 jail      (SMTP AUTH failures)
  └─ dovecot jail    (IMAP/POP3 AUTH failures)
  
SSH:
  └─ sshd jail       (SSH login failures)
```

**Auto-ban behavior**: 5 failures in 10 minutes → 10-minute ban

**View active bans**:
```bash
sudo fail2ban-client status exim4
sudo fail2ban-client status dovecot
sudo ipset list banned_ips
```

### Firewall Rules Status
```bash
# View all firewall rules
sudo iptables -L -n

# View specific port
sudo iptables -L -n | grep "dpt:80"

# Check persistent rules
sudo cat /etc/iptables/rules.v4

# Check Fail2Ban status
sudo fail2ban-client status
sudo fail2ban-client status exim4
sudo fail2ban-client status dovecot

# View blocked IPs
sudo ipset list banned_ips
```

## Configuration Examples

### Complete Enterprise Setup

```bash
# 1. Install core stack
sudo webstack install all

# 2. Install mail server with full features
sudo webstack mail install mail.example.com --spam --av

# 3. Setup master DNS server
sudo webstack dns install --mode master --cluster-name prod

# 4. Add mail domain
sudo webstack domain add mail.example.com --backend nginx --php 8.2

# 5. Enable SSL
sudo webstack ssl enable mail.example.com --email admin@example.com

# 6. Enable database remote access (if needed)
sudo webstack system remote-access enable mysql dbadmin password

# Result: Fully configured production system with:
# ✅ Mail server (7 ports auto-managed)
# ✅ DNS server (port 53 auto-managed)
# ✅ Web services (ports 80/443 auto-managed)
# ✅ Database (port 3306 auto-managed)
# ✅ SSH protected (port 22)
# ✅ Fail2Ban monitoring (auto-banning brute-forcers)
```

### Multi-Server DNS Cluster

**Master Server (192.168.1.10)**:
```bash
sudo webstack dns install --mode master --cluster-name datacenter-1
sudo webstack dns config --zone example.com --type master
sudo webstack dns config --add-slave 192.168.1.20
sudo webstack dns config --add-slave 192.168.1.30
```

**Slave Servers (192.168.1.20, 192.168.1.30)**:
```bash
sudo webstack dns install --mode slave --master-ip 192.168.1.10 --cluster-name datacenter-1
sudo webstack dns config --zone example.com --type slave
```

### Mail Server with Spam/Antivirus Protection

```bash
# Install with full protection
sudo webstack mail install mail.example.com --spam --av

# Add users
sudo webstack mail add user1@mail.example.com
sudo webstack mail add user2@mail.example.com

# Monitor spam scoring
tail -f /var/log/exim4/mainlog | grep "X-Spam-Score"

# Check antivirus activity
tail -f /var/log/clamav/clamd.log

# View Fail2Ban activity
sudo fail2ban-client status exim4
```

## Troubleshooting

### Check Service Status
```bash
sudo systemctl status nginx
sudo systemctl status apache2
sudo systemctl status mysql
sudo systemctl status postgresql
sudo systemctl status exim4
sudo systemctl status dovecot
sudo systemctl status bind9
```

### View Security Logs
```bash
# Mail logs with spam scores
sudo tail -f /var/log/exim4/mainlog

# Dovecot authentication logs
sudo tail -f /var/log/dovecot

# Fail2Ban activity
sudo tail -f /var/log/fail2ban.log

# Firewall rules
sudo iptables -L -n -v

# Blocked IPs
sudo ipset list banned_ips
```

### DNS Troubleshooting
```bash
# Validate configuration
sudo webstack dns check

# Test DNS query
sudo webstack dns query example.com
dig @127.0.0.1 example.com

# Check DNS logs
sudo webstack dns logs --lines 100

# Restart service
sudo webstack dns restart
```

### Mail Troubleshooting
```bash
# Check if mail services are running
sudo systemctl status exim4
sudo systemctl status dovecot
sudo systemctl status spamassassin

# Verify DKIM key exists
ls -la /etc/exim4/domains/example.com/dkim.pem

# Test spam scoring
echo "VIAGRA BUY NOW" | spamc -U /run/spamd.sock -c

# View mail queue
sudo exim4 -bp

# Restart mail services
sudo systemctl restart exim4 dovecot
```

### Firewall Troubleshooting
```bash
# View all rules
sudo iptables -L -n -v

# View specific port
sudo iptables -L -n | grep "dpt:80"

# Check if SSH is still accessible
sudo iptables -L -n | grep "dpt:22"

# View persistent rules
sudo cat /etc/iptables/rules.v4

# Reload rules if modified
sudo systemctl restart iptables-persistent
```

## Security Best Practices

1. **UFW Automatically Removed**: When core security is installed, UFW is automatically removed if present (to avoid conflicts with iptables)
2. **Always Enable SSH Protection**: Port 22 is automatically protected by Fail2Ban
3. **Use Remote Access Carefully**: Only enable database remote access when needed
4. **Monitor Logs**: Regularly check `/var/log/fail2ban.log` for activity
5. **Update Certificates**: SSL certificates auto-renew, verify with `sudo webstack ssl status`
6. **Backup DNS**: Use `sudo webstack dns backup` regularly
7. **Monitor Mail**: Check spam scores with `tail -f /var/log/exim4/mainlog`

## Performance Notes

- **ipset**: O(1) lookup time for IP blocking (efficient even with 100K+ IPs)
- **iptables-persistent**: Rules loaded at boot (zero runtime overhead)
- **Fail2Ban**: Regex-based log monitoring (minimal CPU impact)
- **SpamAssassin**: spamd daemon (pre-forked, efficient)

## Contributing

This tool is modular and extensible. Configuration templates are in `templates/` directory.

## License

Adapted from Hestia Control Panel configurations, built as independent CLI tool for web stack management.
