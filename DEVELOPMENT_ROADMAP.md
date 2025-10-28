# WebStack CLI - Development Roadmap & Next Steps

## 🎯 EXECUTIVE SUMMARY

**Current State**: 65% of core functionality complete, ready for testing
**Time to MVP**: 2-3 weeks (complete remaining critical features)
**Time to v1.0**: 6-8 weeks (including testing and distribution)
**Recommendation**: Focus on database/PHP configuration and SSL automation

---

## 📋 IMMEDIATE TASKS (Next 1-2 weeks)

### Task 1: Database Configuration Implementation
**Priority**: 🔴 CRITICAL | **Effort**: 6-8 hours | **Files**: internal/installer/installer.go

#### What's Missing
- `configureMySQL()` function - Empty stub
- `configureMariaDB()` function - Empty stub
- `configurePostgreSQL()` function - Empty stub

#### What to Implement
```go
configureMySQL() {
  // 1. Apply my.cnf template from internal/templates/mysql/my.cnf
  // 2. Set proper permissions (644)
  // 3. Verify MySQL service is running
  // 4. Create webstack database (optional)
  // 5. Create webstack user with privileges
  // 6. Test connectivity: mysql -u root -p
}

configureMariaDB() {
  // 1. Apply my.cnf template
  // 2. Set proper permissions
  // 3. Verify MariaDB service is running
  // 4. Create webstack database
  // 5. Create webstack user
  // 6. Test connectivity
}

configurePostgreSQL() {
  // 1. Apply postgresql.conf if template exists
  // 2. Create webstack user/role
  // 3. Create webstack database
  // 4. Grant privileges
  // 5. Test connectivity: psql -U webstack
}
```

#### Testing
```bash
sudo webstack install all
# Choose to install MySQL/MariaDB/PostgreSQL
# Verify databases are configured
mysql -u webstack -p
psql -U webstack
```

---

### Task 2: PHP-FPM Per-Version Pool Configuration
**Priority**: 🔴 CRITICAL | **Effort**: 8-10 hours | **Files**: internal/installer/installer.go

#### What's Missing
- `configurePHP()` function - Empty stub
- Per-version pool configuration not applied

#### What to Implement
```go
configurePHP(version string) {
  // 1. Create pool.conf from internal/templates/php-fpm/pool.conf
  // 2. Place in /etc/php/{version}/fpm/pool.d/webstack.conf
  // 3. Apply version-specific settings:
  //    - pm.max_children based on available RAM
  //    - pm.start_servers
  //    - pm.min_spare_servers
  //    - pm.max_spare_servers
  // 4. Set proper file ownership: www-data:www-data
  // 5. Test pool config: php-fpm{version} -t
  // 6. Restart PHP-FPM service
}
```

#### Pool Configuration Strategy
```
For PHP-FPM, calculate worker processes:
- RAM per worker: ~30-50MB
- If 4GB RAM: 4096 / 40 = ~100 max_children
- Start servers: 25% of max
- Min spare: 10
- Max spare: 50
```

#### Testing
```bash
sudo webstack install php 8.2
# Verify pool.conf created
ls -la /etc/php/8.2/fpm/pool.d/
# Check PHP-FPM status
sudo systemctl status php8.2-fpm
# Test with domain
sudo webstack domain add test.local -b nginx -p 8.2
```

---

### Task 3: SSL Renewal Automation
**Priority**: 🟠 HIGH | **Effort**: 4-6 hours | **Files**: internal/ssl/ssl.go, cmd/system.go

#### What's Missing
- No automated renewal schedule
- No renewal notifications
- No expiry warnings

#### What to Implement

**Option A: Systemd Timer (Recommended)**
```go
func SetupRenewalTimer() {
  // 1. Create /etc/systemd/system/webstack-ssl-renew.timer
  // 2. Create /etc/systemd/system/webstack-ssl-renew.service
  // 3. Enable and start timer
  // 4. Timer runs daily at 3 AM
  // 5. Service calls: webstack ssl renew --quiet
}

// Service file content:
[Unit]
Description=WebStack SSL Certificate Renewal
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/webstack ssl renew --quiet
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target

// Timer file content:
[Unit]
Description=WebStack SSL Certificate Renewal Timer
Requires=webstack-ssl-renew.service

[Timer]
OnCalendar=*-*-* 03:00:00
Persistent=true
OnBootSec=24h
OnUnitActiveSec=24h

[Install]
WantedBy=timers.target
```

**Option B: Cron Job (Simpler)**
```bash
# Add to /etc/cron.d/webstack-ssl-renew
0 3 * * * root /usr/local/bin/webstack ssl renew --quiet
```

#### Enhancements
```go
// Add expiry warning (30 days before)
func CheckExpiringSoon() {
  // Check all certificates
  // If expires < 30 days: send warning
  // Log to /var/log/webstack/ssl-warnings.log
}

// Add renewal history
func LogRenewalResult(domain, status, message) {
  // Write to /var/log/webstack/ssl-renewals.log
  // Format: [2025-10-28 03:15:22] domain.com SUCCESS
}

// Add error notifications (optional)
func NotifyRenewalFailure(domain, error) {
  // Send email or write to syslog
  // Only if administrator email is configured
}
```

#### Testing
```bash
# Force renewal check
sudo webstack ssl renew --verbose

# Check renewal logs
sudo tail -f /var/log/webstack/ssl-renewals.log

# Verify timer
sudo systemctl list-timers webstack-ssl-renew
```

---

### Task 4: System Validation Completion
**Priority**: 🟠 HIGH | **Effort**: 4-6 hours | **Files**: cmd/system.go, internal/domain/domain.go, internal/ssl/ssl.go

#### What's Missing
- Domain configuration validation
- SSL certificate validation
- Certificate expiry checking
- File permission validation

#### What to Implement

```go
// In cmd/system.go - validateDomainConfigs()
func validateDomainConfigs() {
  // 1. Load domains.json
  // 2. For each domain:
  //    a. Check if domain files exist:
  //       - /etc/nginx/sites-available/{domain}
  //       - /etc/nginx/sites-enabled/{domain}
  //       - /etc/apache2/sites-available/{domain}
  //    b. Check document root exists and readable
  //    c. Check PHP version configured
  //    d. If SSL enabled: check certificates exist
  // 3. Report any missing files or config issues
}

// In cmd/system.go - validateSSLCertificates()
func validateSSLCertificates() {
  // 1. Load ssl.json
  // 2. For each domain with SSL:
  //    a. Check certificate file exists
  //    b. Check private key file exists
  //    c. Check file permissions (400 for key, 644 for cert)
  //    d. Parse certificate and check:
  //       - Expiry date
  //       - Subject matches domain
  //       - Valid from < now < valid to
  //    e. Warn if expires within 30 days
  // 3. Report any issues
}

// In internal/ssl/ssl.go - CertificateInfo struct
type CertificateInfo struct {
  Domain      string
  Issuer      string
  Subject     string
  NotBefore   time.Time
  NotAfter    time.Time
  DaysLeft    int
  IsExpired   bool
  IsSelfSigned bool
}

// In internal/ssl/ssl.go - ParseCertificate()
func ParseCertificate(certPath string) (*CertificateInfo, error) {
  // 1. Read certificate file
  // 2. Parse X509 certificate
  // 3. Extract certificate info
  // 4. Calculate days remaining
  // 5. Return CertificateInfo
}
```

#### Testing
```bash
# Run full validation
sudo webstack system validate

# Should show:
# ✅ Nginx configuration is valid
# ✅ Apache configuration is valid
# ✅ Domain myapp.local: Configuration OK
# ✅ Domain myapp.local: SSL certificate valid (345 days remaining)
# ⚠️ Domain legacy.local: SSL certificate expires in 15 days
```

---

### Task 5: SSL Status Reporting Enhancement
**Priority**: 🟡 MEDIUM | **Effort**: 2-3 hours | **Files**: internal/ssl/ssl.go

#### Current Implementation
- `Status()` function exists but returns minimal info
- `StatusAll()` returns minimal info

#### What to Implement
```go
func Status(domainName string) {
  // 1. Load domains.json
  // 2. If domain doesn't exist: show error
  // 3. If domain exists but no SSL: show "No SSL configured"
  // 4. If SSL enabled:
  //    a. Get certificate type (self-signed or Let's Encrypt)
  //    b. Parse certificate file
  //    c. Display:
  //       - Domain name
  //       - Certificate type
  //       - Issuer
  //       - Subject
  //       - Valid from date
  //       - Expires date
  //       - Days remaining
  //       - Certificate path
  // 5. Show renewal schedule if available
}

// Example output:
/*
Domain: myapp.local
Status: ✅ SSL Enabled
Type: Self-Signed
Issuer: CN=myapp.local
Subject: CN=myapp.local
Valid From: 2025-10-28
Expires: 2026-10-28
Days Remaining: 365
Certificate Path: /etc/ssl/webstack/myapp.local.crt
Renewal: Not Applicable (Self-Signed)
*/
```

---

## 🎯 PHASE 2: TESTING & DOCUMENTATION (Weeks 3-4)

### Task 6: Create Unit Tests
**Priority**: 🟠 HIGH | **Effort**: 10-12 hours | **Files**: *_test.go

#### Test Coverage Needed
```go
// domain_test.go
- TestAddDomain()           // Add domain, verify JSON
- TestEditDomain()          // Edit, verify changes
- TestDeleteDomain()        // Delete, verify removal
- TestListDomains()         // List all domains
- TestGenerateConfig()      // Config generation from template
- TestDomainExists()        // Existence check
- TestGetDomain()           // Retrieval
- TestUpdateDomain()        // Persistence

// ssl_test.go
- TestEnableWithType()      // Enable with selfsigned/letsencrypt
- TestDisable()             // Disable SSL
- TestParseCertificate()    // Certificate parsing
- TestSelfSignedGeneration()// OpenSSL cert generation
- TestConfigGeneration()    // SSL config file generation

// installer_test.go
- TestCheckComponentStatus()     // Detection logic
- TestConfigureNginx()           // Nginx configuration
- TestConfigureApache()          // Apache configuration
- TestConfigureMySQL()           // MySQL configuration (after implementation)
- TestConfigurePHP()             // PHP pool configuration (after implementation)
```

### Task 7: Integration Tests
**Priority**: 🟠 HIGH | **Effort**: 8-10 hours | **Files**: tests/integration_test.go

#### Scenarios to Test
1. **Complete Installation**: `install all` → verify all services running
2. **Domain Creation**: Add domain → verify configs exist → test via curl
3. **SSL Enable**: Enable SSL → verify cert created → verify HTTPS works
4. **Domain Deletion**: Delete domain → verify configs removed
5. **Multi-Domain**: Add multiple domains → verify isolation
6. **Backend Switch**: Add Nginx + Apache domains → verify both work
7. **PHP Versions**: Multiple PHP versions → verify routing correct
8. **SSL Renewal**: Create certificate → verify renewal logic

### Task 8: Documentation Completion
**Priority**: 🟡 MEDIUM | **Effort**: 6-8 hours | **Files**: docs/*.md

#### Missing Documentation
- [ ] API Documentation (if programmatic use needed)
- [ ] Architecture Design Document
- [ ] Troubleshooting Guide (extended)
- [ ] Performance Tuning Guide
- [ ] Security Hardening Guide
- [ ] Backup & Recovery Procedures
- [ ] Sample Configuration Files

---

## 🚀 PHASE 3: DISTRIBUTION & RELEASE (Week 4+)

### Task 9: GitHub Actions CI/CD
**Priority**: 🟡 MEDIUM | **Effort**: 4-6 hours | **Files**: .github/workflows/*.yml

#### Setup Needed
```yaml
# .github/workflows/build.yml
- Trigger: on push to main
- Build for: linux-amd64, linux-arm64, darwin-amd64, darwin-arm64
- Run tests: go test ./...
- Upload artifacts as release assets

# .github/workflows/release.yml
- Trigger: on tag push (v*.*.*)
- Build binaries
- Create GitHub release
- Upload binaries
- Auto-generate changelog
```

### Task 10: Package Distribution
**Priority**: 🟡 MEDIUM | **Effort**: 8-10 hours | **Files**: distribution/

#### Options
- [ ] **GitHub Releases** - Binary downloads (already supported via `update`)
- [ ] **APT Repository** - Debian/Ubuntu packages
- [ ] **Snap Package** - Snap store distribution
- [ ] **Docker Image** - Container distribution
- [ ] **Homebrew Tap** - macOS distribution (future)

---

## 📊 ESTIMATED EFFORT & TIMELINE

| Task | Priority | Effort | Timeline |
|------|----------|--------|----------|
| Database Configuration | 🔴 CRITICAL | 6-8 hrs | Week 1 |
| PHP-FPM Configuration | 🔴 CRITICAL | 8-10 hrs | Week 1 |
| SSL Renewal Automation | 🟠 HIGH | 4-6 hrs | Week 1-2 |
| System Validation | 🟠 HIGH | 4-6 hrs | Week 1-2 |
| SSL Status Reporting | 🟡 MEDIUM | 2-3 hrs | Week 2 |
| Unit Tests | 🟠 HIGH | 10-12 hrs | Week 2-3 |
| Integration Tests | 🟠 HIGH | 8-10 hrs | Week 2-3 |
| Documentation | 🟡 MEDIUM | 6-8 hrs | Week 3 |
| CI/CD Pipeline | 🟡 MEDIUM | 4-6 hrs | Week 3-4 |
| Distribution Setup | 🟡 MEDIUM | 8-10 hrs | Week 4 |
| **TOTAL** | | **60-79 hrs** | **4 weeks** |

---

## 🎯 QUICK WINS (If Short on Time)

If you only have 1 week, prioritize:
1. ✅ Database Configuration (2-3 hours)
2. ✅ PHP-FPM Configuration (3-4 hours)
3. ✅ SSL Renewal Automation (2-3 hours)
4. ✅ Update README with usage examples (1 hour)

This would make the tool ~80-85% production-ready.

---

## 🔍 DETAILED IMPLEMENTATION GUIDES

### Database Configuration Code Template
```go
// configureMySQL() in internal/installer/installer.go
func configureMySQL() {
    fmt.Println("⚙️  Configuring MySQL...")
    
    // Get my.cnf template
    mycnfData, err := templates.GetMySQLTemplate()
    if err != nil {
        fmt.Printf("Error loading MySQL template: %v\n", err)
        return
    }
    
    // Apply template with substitutions
    config := strings.NewReplacer(
        "{{MAX_CONNECTIONS}}", "1000",
        "{{BUFFER_POOL_SIZE}}", "1G",
        "{{LOG_ERROR}}", "/var/log/mysql/error.log",
    ).Replace(string(mycnfData))
    
    // Write configuration
    if err := ioutil.WriteFile("/etc/mysql/mysql.conf.d/webstack.cnf", []byte(config), 0644); err != nil {
        fmt.Printf("Error writing MySQL config: %v\n", err)
        return
    }
    
    // Restart service
    if err := runCommand("systemctl", "restart", "mysql"); err != nil {
        fmt.Printf("Error restarting MySQL: %v\n", err)
        return
    }
    
    fmt.Println("✅ MySQL configured successfully")
}
```

### PHP-FPM Configuration Code Template
```go
// configurePHP(version) in internal/installer/installer.go
func configurePHP(version string) {
    fmt.Printf("⚙️  Configuring PHP %s-FPM...\n", version)
    
    // Get pool.conf template
    poolData, err := templates.GetPHPTemplate()
    if err != nil {
        fmt.Printf("Error loading PHP template: %v\n", err)
        return
    }
    
    // Calculate worker processes (example: 4GB RAM available)
    maxChildren := "100"      // Adjust based on available RAM
    startServers := "25"
    minSpareServers := "10"
    maxSpareServers := "50"
    
    // Apply template with substitutions
    config := strings.NewReplacer(
        "{{POOL_NAME}}", "webstack",
        "{{USER}}", "www-data",
        "{{GROUP}}", "www-data",
        "{{MAX_CHILDREN}}", maxChildren,
        "{{START_SERVERS}}", startServers,
        "{{MIN_SPARE}}", minSpareServers,
        "{{MAX_SPARE}}", maxSpareServers,
        "{{ERROR_LOG}}", fmt.Sprintf("/var/log/php%s-fpm.log", version),
    ).Replace(string(poolData))
    
    // Create pool directory
    poolDir := fmt.Sprintf("/etc/php/%s/fpm/pool.d", version)
    os.MkdirAll(poolDir, 0755)
    
    // Write pool configuration
    poolPath := filepath.Join(poolDir, "webstack.conf")
    if err := ioutil.WriteFile(poolPath, []byte(config), 0644); err != nil {
        fmt.Printf("Error writing PHP config: %v\n", err)
        return
    }
    
    // Set ownership
    runCommand("chown", "root:root", poolPath)
    
    // Test configuration
    serviceName := fmt.Sprintf("php%s-fpm", version)
    if err := runCommand(serviceName, "-t"); err != nil {
        fmt.Printf("PHP configuration test failed: %v\n", err)
        return
    }
    
    // Restart service
    if err := runCommand("systemctl", "restart", serviceName); err != nil {
        fmt.Printf("Error restarting PHP-FPM: %v\n", err)
        return
    }
    
    fmt.Printf("✅ PHP %s-FPM configured successfully\n", version)
}
```

### SSL Renewal Automation Code Template
```go
// SetupRenewalTimer() in internal/ssl/ssl.go
func SetupRenewalTimer() error {
    fmt.Println("⚙️  Setting up SSL renewal automation...")
    
    // Create systemd service file
    serviceContent := `[Unit]
Description=WebStack SSL Certificate Renewal
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/webstack ssl renew --quiet
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
`
    
    if err := ioutil.WriteFile("/etc/systemd/system/webstack-ssl-renew.service", []byte(serviceContent), 0644); err != nil {
        return fmt.Errorf("error creating service file: %v", err)
    }
    
    // Create systemd timer file
    timerContent := `[Unit]
Description=WebStack SSL Certificate Renewal Timer
Requires=webstack-ssl-renew.service

[Timer]
OnCalendar=*-*-* 03:00:00
Persistent=true
OnBootSec=24h
OnUnitActiveSec=24h

[Install]
WantedBy=timers.target
`
    
    if err := ioutil.WriteFile("/etc/systemd/system/webstack-ssl-renew.timer", []byte(timerContent), 0644); err != nil {
        return fmt.Errorf("error creating timer file: %v", err)
    }
    
    // Enable and start timer
    if err := runCommand("systemctl", "daemon-reload"); err != nil {
        return fmt.Errorf("error reloading systemd: %v", err)
    }
    
    if err := runCommand("systemctl", "enable", "webstack-ssl-renew.timer"); err != nil {
        return fmt.Errorf("error enabling timer: %v", err)
    }
    
    if err := runCommand("systemctl", "start", "webstack-ssl-renew.timer"); err != nil {
        return fmt.Errorf("error starting timer: %v", err)
    }
    
    fmt.Println("✅ SSL renewal automation configured")
    fmt.Println("   Renewal will run daily at 3:00 AM")
    return nil
}
```

---

## 📝 Notes

- All functions use consistent error handling patterns
- All operations provide user feedback with ✅/❌ indicators
- All new configuration files are placed in appropriate system directories
- All code follows the existing style in the project
- All logs should go to `/var/log/webstack/` when applicable

---

## ✅ Success Criteria

When complete, the tool should:
1. ✅ Install all components without errors
2. ✅ Auto-configure all components appropriately
3. ✅ Support multiple domains with both backends
4. ✅ Generate SSL certificates (both types) automatically
5. ✅ Renew certificates automatically
6. ✅ Validate all configurations
7. ✅ Provide clear status information
8. ✅ Have comprehensive tests
9. ✅ Be easily distributable

---

Version: October 28, 2025
WebStack CLI Development Roadmap
