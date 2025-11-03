# WebStack CLI

A comprehensive command-line tool for managing a complete web development stack on Linux systems.

## Features

- **Web Servers**: Install and configure Nginx (port 80) and Apache (port 8080)
- **Databases**: Interactive installation of MySQL/MariaDB and PostgreSQL
- **DNS Server**: Full-featured Bind9 DNS server with master/slave replication and clustering
- **PHP Versions**: Support for PHP-FPM versions 5.6 to 8.4
- **Domain Management**: Add, edit, and delete domains with backend selection
- **SSL Management**: Let's Encrypt SSL certificate management

## Installation

### Quick Start (Recommended)

Complete system setup with CLI, service integration, and directory structure:
```bash
curl -fsSL https://your-domain.com/install.sh | sudo bash
```

This installs:
- ✅ WebStack CLI binary to `/usr/local/bin`
- ✅ System directories with proper permissions
- ✅ Systemd service integration
- ✅ Log rotation configuration
- ✅ Foundation for web control panel (coming soon)

### Manual Installation

For manual control:
```bash
# Download latest release
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
- Ubuntu/Debian Linux system
- Root privileges (run with sudo)

### Install Complete Stack

Install everything with interactive prompts:
```bash
sudo webstack install all
```

### Install Individual Components

```bash
# Web servers
sudo webstack install nginx
sudo webstack install apache

# Databases
sudo webstack install mysql
sudo webstack install mariadb
sudo webstack install postgresql

# PHP versions
sudo webstack install php 8.2
sudo webstack install php 7.4
```

### Domain Management

```bash
# Add a domain (interactive)
sudo webstack domain add example.com

# Add domain with specific backend and PHP version
sudo webstack domain add example.com --backend nginx --php 8.2
sudo webstack domain add api.example.com --backend apache --php 7.4

# Edit domain configuration
sudo webstack domain edit example.com --backend apache --php 8.3

# List all domains
sudo webstack domain list

# Delete a domain
sudo webstack domain delete example.com
```

### SSL Management

```bash
# Enable SSL for a domain
sudo webstack ssl enable example.com --email admin@example.com --type letsencrypt
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

### DNS Server Management (Bind9)

#### Installation

```bash
# Install as master DNS server
sudo webstack dns install

# Install as slave DNS server with master IP
sudo webstack dns install --mode slave --master-ip 192.168.1.10

# Install with clustering support
sudo webstack dns install --mode master --cluster-name datacenter-1
sudo webstack dns install --mode slave --master-ip 192.168.1.10 --cluster-name datacenter-1
```

#### Service Control

```bash
# Check DNS status
sudo webstack dns status

# Restart DNS service
sudo webstack dns restart

# Reload configuration (without restarting)
sudo webstack dns reload

# Validate configuration
sudo webstack dns check
```

#### Zone Management

```bash
# List all configured zones
sudo webstack dns zones

# Add a new master zone
sudo webstack dns config --zone example.com --type master

# Add a slave zone
sudo webstack dns config --zone example.com --type slave

# Add slave server to replication
sudo webstack dns config --add-slave 192.168.1.20

# Remove slave server from replication
sudo webstack dns config --remove-slave 192.168.1.20
```

#### Monitoring & Diagnostics

```bash
# Test DNS query
sudo webstack dns query example.com

# View DNS logs
sudo webstack dns logs --lines 50

# Display query statistics
sudo webstack dns stats
```

#### Advanced Features

```bash
# Enable DNSSEC validation
sudo webstack dns dnssec --enable

# Disable DNSSEC validation
sudo webstack dns dnssec --disable

# Enable query logging
sudo webstack dns querylog --enable

# Disable query logging
sudo webstack dns querylog --disable

# Backup DNS configuration
sudo webstack dns backup

# Restore from backup
sudo webstack dns restore /tmp/dns-backup-20251103_012855.tar.gz

# Uninstall DNS server
sudo webstack dns uninstall
```

#### Master-Slave Setup Example

**Master Server (192.168.1.10):**
```bash
# Install as master
sudo webstack dns install --mode master --server-ip 192.168.1.10

# Add zone
sudo webstack dns config --zone example.com --type master

# Add slave server
sudo webstack dns config --add-slave 192.168.1.20
```

**Slave Server (192.168.1.20):**
```bash
# Install as slave pointing to master
sudo webstack dns install --mode slave --master-ip 192.168.1.10 --server-ip 192.168.1.20

# Add slave zone for same domain
sudo webstack dns config --zone example.com --type slave
```


## Configuration

### Backend Options

- **nginx**: Direct PHP-FPM processing through Nginx
- **apache**: Nginx proxy to Apache (Apache handles PHP)

### Supported PHP Versions

- PHP 5.6, 7.0, 7.1, 7.2, 7.3, 7.4
- PHP 8.0, 8.1, 8.2, 8.3, 8.4

### Default Ports

- **Nginx**: 80 (HTTP), 443 (HTTPS)
- **Apache**: 8080 (HTTP), 8443 (HTTPS)
- **Bind9 DNS**: 53 (TCP/UDP)
- **MySQL/MariaDB**: 3306
- **PostgreSQL**: 5432

## Directory Structure

```
/var/www/[domain]/          # Domain document roots
/etc/webstack/              # Configuration storage
  ├── domains.json          # Domain configurations
  └── ssl.json              # SSL certificate info
/etc/nginx/sites-enabled/   # Nginx domain configs
/etc/apache2/sites-enabled/ # Apache domain configs
```

## Examples

### Complete Setup Example

1. Install the full stack:
   ```bash
   sudo webstack install all
   ```

2. Setup DNS server:
   ```bash
   sudo webstack dns install --mode master
   sudo webstack dns config --zone example.com --type master
   ```

3. Add a WordPress site:
   ```bash
   sudo webstack domain add mysite.com --backend nginx --php 8.2
   ```

4. Enable SSL:
   ```bash
   sudo webstack ssl enable mysite.com --email admin@mysite.com
   ```

5. Add an API subdomain using Apache:
   ```bash
   sudo webstack domain add api.mysite.com --backend apache --php 8.1
   ```

### Multi-Server DNS Cluster

**Server 1 (Master - 192.168.1.10):**
```bash
# Install DNS master
sudo webstack dns install --mode master --cluster-name prod-cluster

# Add zones
sudo webstack dns config --zone example.com --type master
sudo webstack dns config --zone api.example.com --type master

# Add slave servers
sudo webstack dns config --add-slave 192.168.1.20
sudo webstack dns config --add-slave 192.168.1.30
```

**Server 2 & 3 (Slaves):**
```bash
# Install DNS slave
sudo webstack dns install --mode slave --master-ip 192.168.1.10 --cluster-name prod-cluster

# Configure slave zones
sudo webstack dns config --zone example.com --type slave
sudo webstack dns config --zone api.example.com --type slave
```

### Multi-PHP Setup

Run different sites with different PHP versions:
```bash
# Legacy site with old PHP
sudo webstack domain add legacy.com --backend apache --php 7.4

# Modern site with latest PHP
sudo webstack domain add modern.com --backend nginx --php 8.4
```

## Troubleshooting

### Check Service Status
```bash
sudo systemctl status nginx
sudo systemctl status apache2
sudo systemctl status mysql
sudo systemctl status postgresql
sudo systemctl status bind9
sudo systemctl status php8.2-fpm
```

### View Logs
```bash
# Nginx logs
sudo tail -f /var/log/nginx/error.log
sudo tail -f /var/log/nginx/[domain].error.log

# Apache logs
sudo tail -f /var/log/apache2/error.log
sudo tail -f /var/log/apache2/[domain].error.log

# Bind9 DNS logs
sudo tail -f /var/log/named/default.log

# PHP-FPM logs
sudo tail -f /var/log/php8.2-fpm.log
```

### DNS Troubleshooting
```bash
# Validate DNS configuration
sudo webstack dns check

# Test DNS queries
sudo webstack dns query example.com
dig @127.0.0.1 example.com

# View DNS logs
sudo webstack dns logs --lines 100

# Check DNS statistics
sudo webstack dns stats

# Restart DNS service if issues occur
sudo webstack dns restart
```

### Reload Configurations
```bash
sudo systemctl reload nginx
sudo systemctl reload apache2
sudo webstack dns reload
sudo systemctl restart php8.2-fpm
```

## Security Notes

- All installations use secure defaults
- PHP-FPM pools are isolated per version
- SSL certificates are automatically managed
- Security headers are enabled by default
- Sensitive files are protected via web server rules

## Contributing

This tool is designed to be modular and extensible. Template files are located in the `templates/` directory and can be customized as needed.

## License

This project adapts configuration templates from Hestia Control Panel while creating an independent CLI tool for web stack management.