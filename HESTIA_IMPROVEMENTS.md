# WebStack CLI - Hestia-Inspired Improvements

This document details the advanced mail configuration features implemented in WebStack CLI, inspired by Hestia Control Panel's sophisticated mail server setup.

## Table of Contents

1. [Dynamic SSL Per Domain](#dynamic-ssl-per-domain)
2. [DNSBL and Spam Filtering](#dnsbl-and-spam-filtering)
3. [SMTP Relay Support](#smtp-relay-support)
4. [Improved DKIM Configuration](#improved-dkim-configuration)
5. [Configuration Files](#configuration-files)
6. [Usage Examples](#usage-examples)
7. [Security Considerations](#security-considerations)

---

## Dynamic SSL Per Domain

### Overview
WebStack now supports **per-domain SSL certificates** for mail services, allowing multiple domains to have their own TLS certificates while maintaining a default fallback certificate.

### How It Works
The Exim4 configuration uses conditional logic to detect the SNI (Server Name Indication) hostname and dynamically select the appropriate certificate:

```
tls_certificate = ${if and {
    { eq {${domain:test@$tls_in_sni}} {$tls_in_sni}}
    { exists{/etc/ssl/certs/webstack-$tls_in_sni.crt} }
}{
    /etc/ssl/certs/webstack-$tls_in_sni.crt
}{
    /etc/ssl/certs/webstack-mail.crt
}}
```

### Setup Instructions

**For a new domain certificate:**

```bash
# Generate private key
openssl genrsa -out /etc/ssl/private/webstack-example.com.key 2048

# Generate certificate signing request
openssl req -new -key /etc/ssl/private/webstack-example.com.key \
    -out /etc/ssl/example.com.csr

# Sign with Let's Encrypt or your CA
# Then place the certificate at:
/etc/ssl/certs/webstack-example.com.crt

# Set proper permissions
sudo chown root:root /etc/ssl/certs/webstack-example.com.crt
sudo chmod 644 /etc/ssl/certs/webstack-example.com.crt
sudo chown root:root /etc/ssl/private/webstack-example.com.key
sudo chmod 600 /etc/ssl/private/webstack-example.com.key
```

**Important:** The naming convention must be `webstack-{domain}.{crt|key}` for automatic detection.

### Benefits
- ✅ Multi-domain mail services with proper TLS certificates
- ✅ Automatic fallback to default certificate if domain-specific cert missing
- ✅ Improved client trust score (domain-specific certificates)
- ✅ Better compatibility with modern mail clients
- ✅ Supports both STARTTLS and implicit TLS (465 port)

---

## DNSBL and Spam Filtering

### Overview
WebStack now includes comprehensive DNSBL (DNS Black List) integration for real-time spam and abuse database checking, plus local IP blocking/whitelisting.

### Components

#### 1. **Connection-Level Spam Checking**
Performed at SMTP connection time (earliest possible point):

```
acl_check_spammers:
  deny    hosts = +spammers          # Local spam IP blocklist
  deny    dnslists = zen.spamhaus.org : bl.spamcop.net  # DNSBL checks
  accept  hosts = +whitelist         # Trusted IPs bypass checks
  accept
```

#### 2. **Configuration Files**

**`/etc/exim4/spam-blocks.conf`** - Local spam IP list
```
# Format: CIDR notation with optional comment
192.0.2.1/32 : Known spam source
203.0.113.0/24 : Spam subnet
```

**`/etc/exim4/white-blocks.conf`** - Whitelist trusted IPs
```
# Bypass all spam/DNSBL checks
203.0.114.0/24 : Our ISP range
192.0.2.50/32 : Trusted partner
```

**`/etc/exim4/dnsbl.conf`** - DNSBL services to query
```
zen.spamhaus.org
bl.spamcop.net
```

### Supported DNSBL Services
- **Spamhaus Zen** (zen.spamhaus.org) - Comprehensive spam/malware database
- **SpamCop** (bl.spamcop.net) - User-reported spam sources

### Setup Instructions

**Add local spam IPs:**
```bash
echo "192.0.2.100/32 : Spam source" | sudo tee -a /etc/exim4/spam-blocks.conf
sudo systemctl restart exim4
```

**Add trusted IPs:**
```bash
echo "203.0.114.50/32 : Partner server" | sudo tee -a /etc/exim4/white-blocks.conf
sudo systemctl restart exim4
```

**Monitor DNSBL rejections:**
```bash
sudo grep "listed in DNSBL" /var/log/exim4/mainlog
```

### Benefits
- ✅ Real-time spam database checking (DNSBL/RBL)
- ✅ Reduces spam volume before delivery
- ✅ Local control over accepted/rejected IPs
- ✅ Whitelist for trusted partners
- ✅ Logged DNSBL hits for analysis
- ✅ Improves server reputation

### Performance Note
DNSBL checks add ~50-100ms per connection. For high-volume servers, consider selective enabling per domain.

---

## SMTP Relay Support

### Overview
WebStack supports SMTP relay configuration for sending mail through external providers (Gmail, SendGrid, AWS SES, etc.).

### How It Works
Mail can be relayed through external SMTP servers on a per-domain basis:

```
SMTP_RELAY_FILE = /etc/exim4/domains/$sender_address_domain/smtp_relay.conf
SMTP_RELAY_HOST = ${lookup{host}lsearch{SMTP_RELAY_FILE}}
SMTP_RELAY_PORT = ${lookup{port}lsearch{SMTP_RELAY_FILE}}
SMTP_RELAY_USER = ${lookup{user}lsearch{SMTP_RELAY_FILE}}
SMTP_RELAY_PASS = ${lookup{pass}lsearch{SMTP_RELAY_FILE}}
```

### Configuration Files

**Global default:** `/etc/exim4/smtp_relay.conf`
```
host smtp.gmail.com
port 587
user your-email@gmail.com
pass your-app-password
```

**Per-domain override:** `/etc/exim4/domains/example.com/smtp_relay.conf`
```
host mail.sendgrid.net
port 587
user apikey
pass SG.your-sendgrid-api-key
```

### Setup Instructions

**For Gmail:**
```bash
cat > /etc/exim4/domains/example.com/smtp_relay.conf << EOF
host smtp.gmail.com
port 587
user your-email@gmail.com
pass your-app-password
EOF

sudo chown root:root /etc/exim4/domains/example.com/smtp_relay.conf
sudo chmod 600 /etc/exim4/domains/example.com/smtp_relay.conf
```

**Enable relay in Exim4 config:**
Uncomment the `.ifdef RELAY_ENABLED` section in `/etc/exim4/exim4.conf`

**For SendGrid:**
```bash
cat > /etc/exim4/domains/example.com/smtp_relay.conf << EOF
host smtp.sendgrid.net
port 587
user apikey
pass SG.your-key-here
EOF
```

### Benefits
- ✅ Relay through external SMTP services
- ✅ Use Gmail, SendGrid, AWS SES, Mailgun, etc.
- ✅ Per-domain relay configuration
- ✅ Fallback to direct delivery if relay unavailable
- ✅ Improved deliverability for shared IP environments

### Security Recommendations
1. Store relay passwords in restricted files (600 permissions)
2. Use app-specific passwords or API keys, not main account passwords
3. Limit relay functionality to trusted domains
4. Monitor relay usage in logs
5. Consider using TLS-only relay (port 587 with STARTTLS)

---

## Improved DKIM Configuration

### Overview
WebStack implements proper DKIM (DomainKeys Identified Mail) configuration with per-domain signing and modern cipher selection.

### Features
- ✅ Automatic DKIM key generation per domain
- ✅ Per-domain DKIM signing
- ✅ Support for domain-specific selectors
- ✅ Relaxed canonicalization (better compatibility)
- ✅ Headers: from, to, date, subject, message-id, content-type

### DKIM File Structure
```
/etc/exim4/domains/example.com/
├── dkim.pem (or dkim.private)
├── dkim.pub (public key for DNS)
├── passwd (user accounts)
└── aliases (email aliases)
```

### Setup DKIM DNS Record

**Extract public key:**
```bash
openssl pkey -in /etc/exim4/domains/example.com/dkim.pem \
    -pubout -out /etc/exim4/domains/example.com/dkim.pub
```

**Create DNS TXT record:**
```bash
mail._domainkey.example.com TXT "v=DKIM1; h=sha256; k=rsa; p=PUBLICKEY"
```

### Verification
```bash
# Check DKIM record
dig mail._domainkey.example.com TXT

# Test DKIM signing
sudo exim -bhc 127.0.0.1 << 'EOF'
MAIL FROM:<test@example.com>
RCPT TO:<test@gmail.com>
DATA
From: Test <test@example.com>
To: Test <test@gmail.com>
Subject: DKIM Test

This is a test message.
.
QUIT
EOF
```

---

## Configuration Files

### New Templates Created

| File | Location | Purpose |
|------|----------|---------|
| `exim4.conf` | `/etc/exim4/` | Main Exim4 config (enhanced with dynamic SSL, DNSBL, relay) |
| `dnsbl.conf` | `/etc/exim4/` | DNSBL services list (SpamCop, Spamhaus) |
| `spam-blocks.conf` | `/etc/exim4/` | Local spam IP blocklist |
| `white-blocks.conf` | `/etc/exim4/` | Trusted IP whitelist |
| `smtp_relay.conf` | `/etc/exim4/` | Global SMTP relay config (optional) |

### Directory Structure
```
/etc/exim4/
├── exim4.conf                      # Main configuration
├── dnsbl.conf                      # DNSBL services
├── spam-blocks.conf                # Local spam IPs
├── white-blocks.conf               # Whitelist
├── smtp_relay.conf                 # Default relay config
└── domains/
    ├── example.com/
    │   ├── passwd                  # User accounts
    │   ├── aliases                 # Email aliases
    │   ├── dkim.private            # DKIM private key
    │   ├── dkim.pub                # DKIM public key
    │   └── smtp_relay.conf         # Domain-specific relay
    └── example2.com/
        └── ...
```

---

## Usage Examples

### Example 1: Multi-Domain Setup with Per-Domain SSL

```bash
# Add domain with its own certificate
sudo webstack mail domain add example.com

# Upload certificate
sudo cp example.com.crt /etc/ssl/certs/webstack-example.com.crt
sudo cp example.com.key /etc/ssl/private/webstack-example.com.key

# Set permissions
sudo chown root:root /etc/ssl/certs/webstack-example.com.crt
sudo chmod 644 /etc/ssl/certs/webstack-example.com.crt
sudo chown root:root /etc/ssl/private/webstack-example.com.key
sudo chmod 600 /etc/ssl/private/webstack-example.com.key

# Test
openssl s_client -connect localhost:465 -servername example.com
```

### Example 2: Gmail SMTP Relay

```bash
# Configure relay for domain
cat > /etc/exim4/domains/example.com/smtp_relay.conf << EOF
host smtp.gmail.com
port 587
user your-email@gmail.com
pass app-password-from-google
EOF

sudo chown Debian-exim:mail /etc/exim4/domains/example.com/smtp_relay.conf
sudo chmod 600 /etc/exim4/domains/example.com/smtp_relay.conf

# Enable relay in exim4.conf (uncomment RELAY_ENABLED section)
# Then restart
sudo systemctl restart exim4
```

### Example 3: Block a Spam Subnet, Whitelist Partner

```bash
# Block spam range
echo "192.0.2.0/24 : Known spam subnet" | sudo tee -a /etc/exim4/spam-blocks.conf

# Whitelist partner
echo "203.0.114.0/24 : Partner ISP" | sudo tee -a /etc/exim4/white-blocks.conf

# Reload
sudo systemctl reload exim4
```

### Example 4: Monitor DNSBL Activity

```bash
# Show all DNSBL hits
sudo grep "listed in DNSBL" /var/log/exim4/mainlog | tail -20

# Count by DNSBL service
sudo grep "listed in" /var/log/exim4/mainlog | grep -o "\..*:" | sort | uniq -c

# Monitor in real-time
sudo tail -f /var/log/exim4/mainlog | grep "DNSBL"
```

---

## Security Considerations

### TLS Configuration
- ✅ TLS 1.2 and 1.3 only (no legacy protocols)
- ✅ Strong cipher suite (PERFORMANCE grade)
- ✅ 2048-bit DH parameters minimum
- ✅ Per-domain certificates prevent certificate reuse attacks

### DNSBL Recommendations
1. **Performance Trade-off**: DNSBL queries add ~50-100ms per connection
2. **DNS Failures**: If DNS fails, mail is queued and retried (graceful degradation)
3. **False Positives**: Review whitelist regularly to prevent business impact
4. **Whitelisting**: Use `.ifconfig` sections for testing

### SMTP Relay Security
1. **API Keys**: Use domain-specific API keys, not master passwords
2. **File Permissions**: Store relay configs with mode 600
3. **Encryption**: Always use TLS (port 587 with STARTTLS)
4. **Auditing**: Monitor relay usage in logs for abuse
5. **Rate Limiting**: Exim's `smtp_accept_max_per_host` provides basic protection

### Mail Authentication
- ✅ DKIM signing on all outgoing mail
- ✅ SPF records recommended (separate DNS setup)
- ✅ DMARC policy recommended (separate DNS setup)
- ✅ Per-domain authentication with passwd files

---

## Migration from Standard Exim4

If upgrading from basic Exim4 configuration:

1. **Backup current config:**
   ```bash
   sudo cp /etc/exim4/exim4.conf /etc/exim4/exim4.conf.backup
   ```

2. **Backup domain data:**
   ```bash
   sudo tar czf ~/exim4-backup.tar.gz /etc/exim4/domains/
   ```

3. **Install updated WebStack:**
   ```bash
   cd /home/dev/Desktop/webstack
   go build -o build/webstack main.go
   ```

4. **Test new configuration:**
   ```bash
   sudo exim4 -bV | grep -E "Configuration|Support"
   ```

5. **Verify no syntax errors:**
   ```bash
   sudo exim -C /etc/exim4/exim4.conf -bV
   ```

6. **Gradually enable features:**
   - Test DNSBL with small list first
   - Enable per-domain SSL one domain at a time
   - Test relay on non-critical domain first

---

## Performance Tuning

### For High-Volume Servers

**Reduce DNSBL latency:**
```bash
# /etc/exim4/exim4.conf
dnslookup_type = mx
timeout_dns = 2s  # Fail fast if DNS slow
```

**Disable DNSBL per domain:**
Comment out `acl_check_spammers` in production if not needed.

**Connection limits:**
```bash
smtp_accept_max = 500          # Increase for high-volume
smtp_accept_max_per_host = 50  # Increase selective
```

### Monitoring
```bash
# Check mail queue
sudo exim -bp | tail -20

# Monitor connections
sudo netstat -an | grep :25 | wc -l

# Check service health
sudo systemctl status exim4 dovecot clamav-daemon
```

---

## References

- Exim4 Documentation: https://www.exim.org/
- Spamhaus DNSBL: https://www.spamhaus.org/
- SpamCop RBL: https://www.spamcop.net/
- DKIM Specification: RFC 6376
- DMARC Specification: RFC 7489
- SPF Specification: RFC 7208

---

## Implementation Status

| Feature | Status | Notes |
|---------|--------|-------|
| Dynamic SSL per domain | ✅ Implemented | Conditional cert selection via SNI |
| DNSBL/RBL blocking | ✅ Implemented | SpamCop + Spamhaus Zen |
| Local IP blocklist | ✅ Implemented | `spam-blocks.conf` |
| Whitelist support | ✅ Implemented | `white-blocks.conf` |
| SMTP relay | ✅ Implemented | Per-domain override support |
| DKIM signing | ✅ Implemented | Per-domain with selectors |
| TLS hardening | ✅ Implemented | TLS 1.2+ only, strong ciphers |
| Connection ACLs | ✅ Implemented | Spammer check at connection level |

---

**Last Updated:** November 3, 2025
**WebStack Version:** With Hestia-Inspired Improvements
**Compatibility:** Ubuntu 20.04, 22.04, 24.04 LTS
