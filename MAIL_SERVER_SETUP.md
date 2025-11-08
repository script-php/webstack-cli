# WebStack Mail Server Configuration

## Overview
Complete mail server setup with Postfix (MTA), Dovecot (IMAP/POP3), optional ClamAV, and SpamAssassin.

## Components Installed

### Core Services
- **Postfix**: Mail Transport Agent (MTA)
  - SMTP on port 25
  - Submission (authenticated SMTP) on port 587
  - Virtual mailbox delivery via Dovecot LMTP
  
- **Dovecot**: IMAP/POP3 server
  - IMAP on port 143 (unencrypted)
  - IMAPS on port 993 (TLS)
  - POP3 on port 110 (unencrypted)
  - POP3S on port 995 (TLS)
  - LMTP delivery to mailboxes
  - SASL authentication for Postfix

### Optional Security Features
- **ClamAV**: Antivirus scanning (optional)
- **SpamAssassin**: Spam filtering (optional)

## Architecture

### Virtual Mail Structure
```
/var/mail/vhosts/
├── domain1.com/
│   ├── user1/
│   │   ├── new/       (Maildir structure)
│   │   ├── cur/
│   │   └── tmp/
│   └── user2/
└── domain2.org/
    └── admin/
```

### Authentication
- **Virtual Users**: Stored in `/etc/dovecot/users` (passwd-file format)
- **Format**: `email:{PLAIN}password:uid:gid::homedir::`
- **Example**: `test@test.local:{PLAIN}illegall:mail:mail::/var/mail/vhosts/test.local/test::`

### Postfix Configuration
Key settings for virtual mail delivery:

```
virtual_mailbox_base = /var/mail/vhosts
virtual_mailbox_maps = hash:/etc/postfix/vmailbox
virtual_mailbox_domains = hash:/etc/postfix/vdomains
virtual_transport = lmtp:unix:private/dovecot-lmtp
virtual_minimum_uid = 1
smtpd_sasl_auth_enable = yes
smtpd_sasl_type = dovecot
smtpd_sasl_path = private/auth
```

### Dovecot Configuration
Key settings for virtual mail and LMTP:

```
mail_location = maildir:/var/mail/vhosts/%d/%n
first_valid_uid = 0
last_valid_uid = 0
```

## DNS Records
When adding a mail domain, generate:

- **SPF Record**: `v=spf1 ip4:YOUR_IP -all`
- **DKIM Record**: Generated RSA key and public key record
- **DMARC Record**: `v=DMARC1; p=quarantine; rua=mailto:admin@domain`

## Key Fixes Applied

### 1. Maildir Format Configuration
- Changed from mbox to Maildir format
- Created `new/`, `cur/`, `tmp/` directories for each account
- Enables proper synchronization with IMAP clients

### 2. Directory Permissions
- `/var/mail/vhosts` must be `0755` (755 = rwxr-xr-x)
- Allows mail user to traverse directories
- Prevents "Permission denied" errors in Dovecot

### 3. UID/GID Settings
- **Dovecot**: `first_valid_uid = 0` allows mail user (UID 8)
- **Postfix**: `virtual_minimum_uid = 1` allows mail user delivery
- Without these, mail delivery fails with "bad uid" errors

### 4. Dovecot LMTP for Postfix
- Dovecot provides LMTP socket at `/var/spool/postfix/private/dovecot-lmtp`
- Postfix uses `virtual_transport = lmtp:unix:private/dovecot-lmtp`
- Enables reliable mail delivery with proper Maildir handling

### 5. SASL Authentication
- Dovecot socket at `/var/spool/postfix/private/auth`
- Allows SMTP clients to authenticate via submission port (587)
- Uses virtual user database from `/etc/dovecot/users`

## Usage

### Add Mail Domain
```bash
webstack mail add domain example.com
```

### Add Mail Account
```bash
webstack mail add account user@example.com password123
```

### List Accounts
```bash
webstack mail list accounts
```

### List Domains
```bash
webstack mail list domains
```

### View DNS Records
```bash
webstack mail dns show example.com
```

### Add Firewall Rules
```bash
webstack mail firewall open
```

Opens ports: 25, 465, 587, 110, 995, 143, 993, 4190

## Testing

### Send Email (SMTP)
```python
import smtplib
server = smtplib.SMTP("localhost", 587)
server.starttls()
server.login("user@domain.com", "password")
server.sendmail("user@domain.com", "recipient@domain.com", "message")
server.quit()
```

### Receive Email (IMAP)
```python
import imaplib
server = imaplib.IMAP4_SSL("localhost", 993)
server.login("user@domain.com", "password")
status, messages = server.search(None, 'ALL')
```

## Troubleshooting

### Mails stuck in queue
- Check: `/var/log/mail.log`
- Issue: Wrong `virtual_transport` or `virtual_mailbox_base`
- Fix: Verify Postfix configuration with `postconf`

### Cannot connect IMAP
- Check: `/var/log/syslog` for "Permission denied" errors
- Issue: Directory permissions or UID settings
- Fix: Ensure `/var/mail/vhosts` has 0755 permissions

### SMTP authentication fails
- Check: `sudo doveadm auth test email@domain.com password`
- Issue: User not in `/etc/dovecot/users` or wrong password format
- Fix: Verify user entry format and regenerate password

### Maildir structure missing
- Issue: Accounts added without new/cur/tmp directories
- Fix: Run `AddMailAccount` which creates proper structure

## Security Considerations

1. **Use TLS** for all connections (IMAPS 993, SMTPS 465, SMTP+STARTTLS 587)
2. **Generate certificates** with proper domain names
3. **Enable firewall** rules for mail ports only
4. **Regular backups** of `/etc/dovecot/users` and `/var/mail/vhosts`
5. **Monitor logs** for security issues in `/var/log/mail.log`

## Performance Tuning

- **Connection limits**: `smtpd_client_connection_limit`
- **Message size**: `message_size_limit` (default 10MB)
- **Mailbox size**: `mailbox_size_limit` (0 = unlimited)
- **LMTP concurrency**: Adjust in Dovecot LMTP service

## Backup Strategy

```bash
# Backup virtual mail
tar -czf backup-mail-vhosts.tar.gz /var/mail/vhosts/

# Backup user database
cp /etc/dovecot/users /backup/dovecot-users.backup

# Backup Postfix maps
cp /etc/postfix/vmailbox /backup/postfix-vmailbox.backup
cp /etc/postfix/vdomains /backup/postfix-vdomains.backup

# Backup DKIM keys
tar -czf backup-dkim-keys.tar.gz /etc/postfix/dkim/
```

## References

- [Postfix Virtual Mailbox Documentation](http://www.postfix.org/VIRTUAL_README.html)
- [Dovecot LMTP Documentation](https://doc.dovecot.org/configuration_manual/protocols/lmtp/)
- [Dovecot Virtual Users](https://doc.dovecot.org/configuration_manual/authentication/passwd/)
