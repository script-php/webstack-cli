# WebStack Uninstall Commands

Now you can easily uninstall any component of your web stack using the `webstack uninstall` command.

## Available Commands

### Uninstall Everything
```bash
sudo webstack uninstall all
```
Uninstalls the complete web stack with multiple confirmation prompts:
- Nginx
- Apache
- MySQL (optional prompt)
- MariaDB (optional prompt)
- PostgreSQL (optional prompt)
- All PHP versions (optional prompts)
- phpMyAdmin (optional prompt)
- phpPgAdmin (optional prompt)

**Preserves:** Domain configurations and SSL certificates in `/etc/webstack/`

### Uninstall Individual Web Servers

#### Uninstall Nginx
```bash
sudo webstack uninstall nginx
```

#### Uninstall Apache
```bash
sudo webstack uninstall apache
```

### Uninstall Databases

#### Uninstall MySQL
```bash
sudo webstack uninstall mysql
```

#### Uninstall MariaDB
```bash
sudo webstack uninstall mariadb
```

#### Uninstall PostgreSQL
```bash
sudo webstack uninstall postgresql
```

### Uninstall PHP Versions

#### Uninstall Specific PHP Version
```bash
sudo webstack uninstall php 8.2
sudo webstack uninstall php 7.4
sudo webstack uninstall php 5.6
```

Supports versions: 5.6, 7.0, 7.1, 7.2, 7.3, 7.4, 8.0, 8.1, 8.2, 8.3, 8.4

### Uninstall Web Interfaces

#### Uninstall phpMyAdmin
```bash
sudo webstack uninstall phpmyadmin
```

#### Uninstall phpPgAdmin
```bash
sudo webstack uninstall phppgadmin
```

## Uninstall Behavior

### Safety Features
✅ **Confirmation Prompts** - Every uninstall asks for confirmation before removing
✅ **Not Installed Check** - Shows friendly message if component is not installed
✅ **Graceful Uninstall** - Stops services before removing packages
✅ **Data Preservation** - Preserves important data like domains and SSL certificates

### What Gets Removed
- Package files
- Service configurations
- System services registration
- Associated dependencies

### What Gets Preserved
✅ Domain configurations (`/etc/webstack/domains.json`)
✅ SSL certificates and configuration (`/etc/webstack/ssl.json`, `/etc/ssl/webstack/`, `/etc/letsencrypt/`)
✅ Application data (document roots remain unless explicitly deleted)
✅ Database data (if you're just removing the database server package)

## Examples

### Remove Just Apache While Keeping Nginx
```bash
sudo webstack uninstall apache
# ℹ️  Are you sure? (y/N): y
# ✅ Apache uninstalled successfully
```

### Remove a Specific PHP Version
```bash
sudo webstack uninstall php 7.4
# ℹ️  Are you sure? (y/N): y
# ✅ PHP 7.4 uninstalled successfully
```

### Completely Clean Up (Keep Domains)
```bash
# Uninstall everything
sudo webstack uninstall all
# Follows through confirmation prompts
# Answer yes to all optional uninstalls

# Your domain data remains in /etc/webstack/
# You can reinstall components later and reuse the domains
```

### Selective Uninstall
```bash
# Uninstall only database servers but keep web servers and PHP
sudo webstack uninstall mysql
sudo webstack uninstall postgresql

# Reinstall only what you need
sudo webstack install apache
```

## Reinstalling After Uninstall

You can reinstall any component at any time:

```bash
# Reinstall after uninstalling
sudo webstack install nginx
sudo webstack install php 8.2
sudo webstack install mysql

# Your domains and SSL certificates remain intact
sudo webstack domain list
sudo webstack ssl status
```

## Recovery

### If You Accidentally Uninstalled Everything
1. Domains and SSL configurations are preserved in `/etc/webstack/`
2. Reinstall the components:
   ```bash
   sudo webstack install all
   ```
3. Rebuild domain configurations:
   ```bash
   sudo webstack domain rebuild-configs
   ```
4. Your domains are ready to use again!

## Command Comparison

### Install Commands
```bash
webstack install all           # Install complete stack
webstack install nginx         # Install specific component
webstack install php 8.2       # Install specific PHP version
```

### Uninstall Commands
```bash
webstack uninstall all         # Uninstall complete stack (with confirmations)
webstack uninstall nginx       # Uninstall specific component
webstack uninstall php 8.2     # Uninstall specific PHP version
```

Both install and uninstall commands follow the same component naming convention for consistency and ease of use.

## FAQ

**Q: Will uninstalling Apache delete my websites?**
A: No! Your websites are stored in document roots and preserved. Only the Apache package is removed.

**Q: Can I uninstall just PHP 7.4 and keep 8.2?**
A: Yes! Each PHP version is independent and can be uninstalled separately.

**Q: What happens to my databases when I uninstall MySQL?**
A: The MySQL package is removed, but if you have a backup, you can reinstall and restore.

**Q: Will my domains still work after uninstalling Nginx?**
A: Not until you reinstall Nginx or another web server. But your domain configurations are preserved, so reinstalling will restore everything.

**Q: How do I completely remove a domain?**
A: Use `webstack domain delete <domain>` to remove a domain and its configurations.
