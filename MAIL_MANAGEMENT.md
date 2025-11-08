# Mail Management Commands

Complete mail management system for WebStack CLI with support for adding, listing, and deleting mail domains and accounts.

## Installation

First, install the mail server stack:

```bash
sudo webstack install mail
```

## Usage

### Add a Mail Domain

```bash
sudo webstack mail add domain mydomain.tld
```

**Example:**
```bash
sudo webstack mail add domain example.com
```

### Add a Mail Account

```bash
sudo webstack mail add account user@domain.tld password
```

**Example:**
```bash
sudo webstack mail add account admin@example.com MySecurePass123
```

**Notes:**
- Email must be in format: `user@domain.tld`
- Domain must already exist (add domain first if needed)
- Password will be hashed using SHA512-CRYPT for security
- Mailbox created automatically at `/var/mail/vhosts/domain/user/`

### List Mail Accounts

```bash
sudo webstack mail list accounts
```

**Output:**
```
📋 Mail Accounts
================
  • admin@example.com
  • support@example.org

✅ Total: 2 account(s)
```

### List Mail Domains

```bash
sudo webstack mail list domains
```

**Output:**
```
📋 Mail Domains
===============
  • example.com
  • example.org

✅ Total: 2 domain(s)
```

### Delete a Mail Account

```bash
sudo webstack mail delete account user@domain.tld
```

**Example:**
```bash
sudo webstack mail delete account admin@example.com
```

**Notes:**
- Will ask for confirmation before deleting
- Removes mailbox directory and password entry
- Reloads Postfix configuration

### Delete a Mail Domain

```bash
sudo webstack mail delete domain mydomain.tld
```

**Example:**
```bash
sudo webstack mail delete domain example.com
```

**Notes:**
- Will ask for confirmation before deleting
- Removes all mailboxes in the domain
- Reloads Postfix configuration
- ⚠️ WARNING: Deletes all accounts and their mailboxes in that domain

## File Locations

**Mail Configuration Files:**
- Virtual domains: `/etc/postfix/vdomains`
- Virtual mailboxes: `/etc/postfix/vmailbox`
- Mail user passwords: `/etc/dovecot/passwd.d/*.passwd`

**Mail Storage:**
- Mailbox directories: `/var/mail/vhosts/domain/user/`

## Example Workflow

```bash
# 1. Install mail stack
sudo webstack install mail

# 2. Add domains
sudo webstack mail add domain example.com
sudo webstack mail add domain info.example.com

# 3. Add accounts
sudo webstack mail add account admin@example.com admin123
sudo webstack mail add account support@example.com support456
sudo webstack mail add account info@info.example.com info789

# 4. List all
sudo webstack mail list domains
sudo webstack mail list accounts

# 5. Connect with email client using:
# IMAP: mail.example.com:143 (TLS)
# SMTP: mail.example.com:25
# POP3: mail.example.com:110 (TLS)
```

## Testing

Use the provided test script to verify everything is working:

```bash
sudo /home/dev/Desktop/webstack/test_mail.sh
```

## Troubleshooting

**Account not working after creation:**
- Check Postfix is running: `sudo systemctl status postfix`
- Reload Postfix: `sudo postfix reload`
- Check virtual mailbox file: `sudo postmap /etc/postfix/vmailbox`

**Can't connect with mail client:**
- Verify IMAP/SMTP ports: `sudo ss -tulpn | grep -E '143|25|110'`
- Check Dovecot logs: `sudo journalctl -u dovecot -f`
- Ensure SSL certificates are configured

**Password issues:**
- Passwords are stored as SHA512-CRYPT hashes
- If doveadm is not available, falls back to plaintext
- To manually change a password: `sudo doveadm pw -s SHA512-CRYPT -p newpassword`

## Security Notes

✅ **Best Practices:**
1. Use strong, complex passwords
2. Enable TLS/SSL for all connections
3. Configure firewall rules appropriately
4. Regular backups of `/var/mail/vhosts/` directory
5. Monitor mail logs: `sudo tail -f /var/log/mail.log`
6. Use SPF, DKIM, and DMARC DNS records
7. Keep antivirus/spam definitions updated (if enabled)

## Advanced Configuration

For production use, consider:

1. **Database Backend:** Migrate to PostgreSQL/MySQL virtual user database
2. **Backup:** Set up automated mailbox backups
3. **Monitoring:** Configure Nagios/Zabbix monitoring
4. **Clustering:** Set up mail server redundancy
5. **Quotas:** Configure mailbox size limits per user
6. **Aliases:** Set up mail aliases for multiple addresses
7. **Auto-responders:** Configure out-of-office messages
