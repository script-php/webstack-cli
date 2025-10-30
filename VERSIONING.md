# Database Version Selection Guide

## Overview

WebStack CLI now supports installing specific versions of MySQL and MariaDB. By default, it installs the latest available version from your Ubuntu/Debian repositories.

## MySQL Version Selection

### Install Latest MySQL (Default)
```bash
sudo webstack install mysql
```

### Install Specific MySQL Version

**Available versions** depend on your Ubuntu/Debian distribution. Common versions:
- `5.7` - Older stable release
- `8.0` - Current stable (LTS)
- `8.1` - Newer release

```bash
# Install MySQL 8.0
sudo webstack install mysql 8.0

# Install MySQL 5.7
sudo webstack install mysql 5.7

# Install MySQL 8.1
sudo webstack install mysql 8.1
```

**Note:** The installer will match `8.0*`, so it may install `8.0.43` or similar patch versions.

## MariaDB Version Selection

### Install Latest MariaDB (Default)
```bash
sudo webstack install mariadb
```

### Install Specific MariaDB Version

**Available versions** depend on your Ubuntu/Debian distribution. Common versions:
- `10.5` - Older stable
- `10.6` - Stable LTS
- `10.11` - Latest stable
- `11.0` - New series

```bash
# Install MariaDB 10.11 (recommended)
sudo webstack install mariadb 10.11

# Install MariaDB 10.6
sudo webstack install mariadb 10.6

# Install MariaDB 10.5
sudo webstack install mariadb 10.5

# Install MariaDB 11.0
sudo webstack install mariadb 11.0
```

**Note:** The installer will match `10.11*`, so it may install `10.11.13` or similar patch versions.

## Check Available Versions

To see what database versions are available in your distribution:

```bash
# Check MySQL versions
apt-cache policy mysql-server

# Check MariaDB versions
apt-cache policy mariadb-server
```

## Examples

### Scenario 1: Production Server with Specific Requirements
```bash
# Install MariaDB 10.11 (latest stable)
sudo webstack install mariadb 10.11

# Then set up your domains
sudo webstack domain add api.example.com --backend nginx --php 8.2
sudo webstack ssl enable api.example.com --email admin@example.com
```

### Scenario 2: Legacy Application Compatibility
```bash
# Install MySQL 5.7 for legacy app
sudo webstack install mysql 5.7

# Install domain with PHP 7.4 (compatible version)
sudo webstack domain add legacy.example.com --backend apache --php 7.4
```

### Scenario 3: Multi-Version Setup (Not Supported)
**Note:** WebStack only supports one database server at a time (MySQL OR MariaDB, not both).

If you need to switch databases:
```bash
# Uninstall current database
sudo webstack uninstall mariadb

# Wait for reboot prompt or manually reboot
sudo reboot

# Install different version or database
sudo webstack install mysql 8.0
```

## Version Compatibility

### Ubuntu 24.04 LTS (Noble)

**MySQL versions available:**
- MySQL 8.0 (default)

**MariaDB versions available:**
- MariaDB 10.11 (default)

### Ubuntu 22.04 LTS (Jammy)

**MySQL versions available:**
- MySQL 5.7
- MySQL 8.0 (default)

**MariaDB versions available:**
- MariaDB 10.6 (default)
- MariaDB 10.7
- MariaDB 10.8

### Ubuntu 20.04 LTS (Focal)

**MySQL versions available:**
- MySQL 5.7 (default)
- MySQL 8.0

**MariaDB versions available:**
- MariaDB 10.3
- MariaDB 10.4
- MariaDB 10.5 (default)

## Upgrade Path

### MariaDB to MariaDB (Different Version)

1. **Backup your databases** (if needed)
2. **Uninstall current version:**
   ```bash
   sudo webstack uninstall mariadb
   sudo reboot
   ```
3. **Install new version:**
   ```bash
   sudo webstack install mariadb 10.11
   ```

### MySQL to MariaDB (or vice versa)

1. **Backup your databases**
2. **Uninstall current:**
   ```bash
   sudo webstack uninstall mysql
   sudo reboot
   ```
3. **Install alternative:**
   ```bash
   sudo webstack install mariadb 10.11
   ```

**Important:** Always backup your databases before switching!

## Troubleshooting

### Version Not Found
```
Error: Package mysql-server=5.6* is not available
```

**Solution:** The version may not be available in your distribution. Check available versions:
```bash
apt-cache policy mysql-server
```

### Partial Version Match
Your Ubuntu/Debian repo may only have certain patch versions. When you specify `8.0`, the system will install `8.0.43` (or whatever is latest).

This is intentional - it ensures you get security patches automatically.

### Need Exact Version
If you need a specific patch version (e.g., `8.0.35` exactly):
```bash
# Manual installation (without webstack version selection)
sudo apt-get install -y mysql-server=8.0.35-0ubuntu0.24.04.1
```

## PHP Version Compatibility Notes

**Recommended combinations:**

| Database | PHP Versions | Notes |
|----------|-------------|-------|
| MySQL 8.0 | 7.4+ | Works with modern PHP |
| MySQL 5.7 | 5.6-7.4 | Works with older PHP |
| MariaDB 10.11 | 7.4+ | Recommended setup |
| MariaDB 10.6 | 7.0+ | Good compatibility |
| MariaDB 10.5 | 7.0+ | Older stable |

## More Information

For more details on database administration:
```bash
# Check database status
sudo systemctl status mysql
sudo systemctl status mariadb

# View logs
sudo tail -f /var/log/mysql/error.log

# Rebuild configurations (if needed)
sudo webstack domain rebuild-configs
```
