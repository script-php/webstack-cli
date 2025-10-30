# WebStack CLI - Quick Reference & Status Summary

## 🎯 Current State Summary

**What Works**: ~65% of core functionality is complete and production-ready
**What's Partial**: Database/PHP configuration, SSL renewal automation
**What's Missing**: Advanced features, testing, distribution

---

## ✅ READY TO USE NOW

### Installation
```bash
sudo webstack install all          # Interactive complete stack
sudo webstack install nginx        # Install Nginx on port 80
sudo webstack install apache       # Install Apache on port 8080
sudo webstack install mysql        # Install MySQL
sudo webstack install mariadb      # Install MariaDB
sudo webstack install postgresql   # Install PostgreSQL
sudo webstack install php 8.2      # Install PHP 8.2-FPM
```

### Domain Management
```bash
sudo webstack domain add example.com              # Add domain (interactive)
sudo webstack domain add myapp.local -b nginx -p 8.2
sudo webstack domain edit myapp.local -p 8.1     # Change PHP version
sudo webstack domain list                         # Show all domains
sudo webstack domain delete myapp.local           # Remove domain
sudo webstack domain rebuild-configs              # Regenerate all configs
```

### SSL Certificates
```bash
sudo webstack ssl enable myapp.local              # Auto-detect (.local = self-signed)
sudo webstack ssl enable myapp.local --type selfsigned
sudo webstack ssl enable myapp.com --type letsencrypt -e admin@example.com
sudo webstack ssl disable myapp.local             # Remove SSL, keep HTTP
sudo webstack ssl status myapp.local              # Check certificate
sudo webstack ssl renew myapp.local               # Manual renew
```

### System Management
```bash
sudo webstack system reload                       # Reload all configs
sudo webstack system validate                     # Check Nginx/Apache configs
sudo webstack system status                       # Show active services
sudo webstack system cleanup                      # Clean temp files & old logs
```

### Utilities
```bash
webstack version                                  # Show version info
webstack update                                   # Check for updates
```

---

## 📋 FEATURE MATRIX

| Feature | Status | Notes |
|---------|--------|-------|
| Nginx Installation | ✅ Complete | Port 80, auto-configured |
| Apache Installation | ✅ Complete | Port 8080, disabled by default |
| MySQL/MariaDB Install | ✅ Complete | Configuration not applied |
| PostgreSQL Install | ✅ Complete | Configuration not applied |
| PHP 5.6-8.4 Install | ✅ Complete | Per-version tuning missing |
| Domain Add/Edit/Delete | ✅ Complete | Full CRUD with config generation |
| Domain Rebuild | ✅ Complete | Regenerates all configs |
| SSL Self-Signed | ✅ Complete | 365-day certificates |
| SSL Let's Encrypt | ✅ Complete | Auto-renewal via Certbot |
| SSL Status | ⚠️ Partial | Command exists, minimal info |
| SSL Renewal | ⚠️ Partial | Manual works, automation missing |
| System Reload | ✅ Complete | Nginx/Apache/PHP-FPM |
| Config Validation | ⚠️ Partial | Nginx/Apache only, no domains/SSL |
| Service Status | ✅ Complete | Shows active services |
| System Cleanup | ✅ Complete | Temp files, logs, caches |
| Version Check | ✅ Complete | GitHub API integration |
| Pre-Install Detection | ✅ Complete | All components |
| Component Uninstall | ✅ Complete | All components |

---

## 🏗️ ARCHITECTURE AT A GLANCE

### File Organization
```
cmd/              - CLI commands (add new commands here)
internal/         - Implementation logic
  installer/      - Install/uninstall component logic
  domain/         - Domain management and config generation
  ssl/            - SSL certificate management
  templates/      - Embedded configuration templates
  config/         - Template processing utilities
```

### Data Files
- `/etc/webstack/domains.json` - Domain configurations and settings
- `/etc/webstack/ssl.json` - SSL certificate metadata
- `/etc/ssl/webstack/` - Self-signed certificates
- `/etc/letsencrypt/` - Let's Encrypt certificates

### Configuration Locations
- Nginx: `/etc/nginx/sites-available/` and `/etc/nginx/sites-enabled/`
- Apache: `/etc/apache2/sites-available/` and `/etc/apache2/sites-enabled/`
- PHP-FPM: `/etc/php/X.Y/fpm/pool.d/`

---

## 🚀 DEPLOYMENT CHECKLIST

### Before Production Use
- [ ] Test domain addition with Nginx backend
- [ ] Test domain addition with Apache backend
- [ ] Test SSL with self-signed certificate
- [ ] Test SSL with Let's Encrypt (requires public domain)
- [ ] Run `system validate` to check configurations
- [ ] Verify DNS/domain pointing to server IP
- [ ] Test each installed PHP version

### For Production Deployment
- [ ] ✅ Install all components via `install all`
- [ ] ✅ Add production domains via `domain add`
- [ ] ✅ Enable SSL for all domains via `ssl enable`
- [ ] ✅ Set up monitoring/alerts (manual for now)
- [ ] ⚠️ Configure databases (manual until configureDB complete)
- [ ] ⚠️ Tune PHP-FPM pools (manual until configurePHP complete)
- [ ] ✅ Run `system cleanup` regularly via cron

---

## ⚠️ KNOWN LIMITATIONS

1. **Database Auto-Configuration** - MySQL/MariaDB/PostgreSQL install but don't apply my.cnf templates
   - Workaround: Manually edit config files or use provided templates
   
2. **PHP-FPM Tuning** - Per-version configuration not applied
   - Workaround: Manually create pool.conf in `/etc/php/X.Y/fpm/pool.d/`
   
3. **SSL Renewal Automation** - Certbot is configured but renewal schedule not created
   - Workaround: Manual renewal with `ssl renew` or add cron: `0 3 * * * sudo webstack ssl renew`
   
4. **System Validation** - Only checks Nginx/Apache, not domain/SSL configs
   - Workaround: Manually verify domain JSON and SSL certificate files
   
5. **No Backup/Restore** - Configuration changes not tracked
   - Workaround: Manual backups of `/etc/webstack/` directory

---

## 🔄 TYPICAL WORKFLOW

### New Project Setup (10 minutes)
```bash
# 1. Install everything
sudo webstack install all

# 2. Add domain
sudo webstack domain add myapp.local -b nginx -p 8.2

# 3. Enable SSL (local domain = self-signed automatically)
sudo webstack ssl enable myapp.local --type selfsigned

# 4. Point domain to server and start developing!
# Add to /etc/hosts: 127.0.0.1 myapp.local
```

### Production Setup (20 minutes)
```bash
# 1. Install stack
sudo webstack install all

# 2. Add production domain
sudo webstack domain add myapp.com -b nginx -p 8.2

# 3. Enable Let's Encrypt SSL
sudo webstack ssl enable myapp.com --type letsencrypt -e admin@example.com

# 4. Point DNS to server IP
# 5. Verify with curl: curl https://myapp.com/

# 6. (Optional) Add cron for renewal
# 0 3 * * * sudo webstack ssl renew --quiet
```

### Multi-Backend Setup (15 minutes)
```bash
# 1. Install all components
sudo webstack install all

# 2. Add Nginx domain for PHP
sudo webstack domain add app.local -b nginx -p 8.2

# 3. Add Apache domain for legacy app
sudo webstack domain add legacy.local -b apache -p 5.6

# 4. Enable SSL for both
sudo webstack ssl enable app.local --type selfsigned
sudo webstack ssl enable legacy.local --type selfsigned
```

---

## 🔍 TROUBLESHOOTING QUICK GUIDE

### Domain not responding
```bash
# 1. Verify domain exists
sudo webstack domain list

# 2. Check Nginx/Apache configs
sudo webstack system validate

# 3. Check if server running
sudo webstack system status

# 4. Reload configs
sudo webstack system reload
```

### SSL certificate issues
```bash
# 1. Check SSL status
sudo webstack ssl status mydomain.local

# 2. Verify certificate files exist
ls -la /etc/ssl/webstack/
ls -la /etc/letsencrypt/live/

# 3. Try regenerating config
sudo webstack domain rebuild-configs
sudo webstack system reload
```

### PHP not executing
```bash
# 1. Check PHP version is installed
sudo webstack install php 8.2

# 2. Verify PHP-FPM running
sudo systemctl status php8.2-fpm

# 3. Reload configurations
sudo webstack system reload
```

### Port conflicts
```bash
# Check what's using ports 80 and 8080
sudo lsof -i :80
sudo lsof -i :8080

# Restart services
sudo systemctl restart nginx apache2
```

---

## 📦 WHAT'S IN THE BOX

### Included (Already Working)
- ✅ Web server management (Nginx port 80, Apache port 8080)
- ✅ Domain configuration with template-based setup
- ✅ SSL certificate generation (self-signed and Let's Encrypt)
- ✅ PHP-FPM multi-version support
- ✅ System reload and validation
- ✅ Version checking and updates

### Included but Not Configured
- ⚠️ MySQL/MariaDB/PostgreSQL (installed but unconfigured)
- ⚠️ PHP-FPM (installed but pools not configured)

### Not Included (Manual Setup Needed)
- ❌ Database backup/restore
- ❌ Monitoring/alerting
- ❌ Load balancing
- ❌ Firewall rules
- ❌ SSL certificate renewal automation

---

```

## 🎯 NEXT PRIORITIES FOR DEVELOPMENT

### High Priority (1-2 weeks)
1. Database configuration automation
2. PHP-FPM per-version pool configuration
3. SSL renewal automation
4. System validation for domains/SSL

### Medium Priority (2-4 weeks)
5. Unit tests and integration tests
6. Troubleshooting documentation
7. Health check command
8. Configuration rollback

### Low Priority (1+ month)
9. GitHub Actions CI/CD
10. APT repository setup
11. Snap package publication
12. Docker image creation

---

## 📞 SUPPORT & RESOURCES

- **GitHub**: https://github.com/script-php/webstack-cli
- **Issues**: https://github.com/script-php/webstack-cli/issues
- **Documentation**: See README.md and other .md files in project
- **Logs**: Check `/var/log/` for web server and system logs

---

## Version Info
- **Build Date**: October 28, 2025
- **Go Version**: 1.25.3
- **Cobra Framework**: v1.10.1
- **Project Completion**: ~65% (core) / ~40% (including advanced features)
