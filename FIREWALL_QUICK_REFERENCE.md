# Quick Reference: iptables + ipset + Fail2Ban

## One-Sentence Explanation

**iptables** = firewall rules engine | **ipset** = fast IP lists | **Fail2Ban** = auto-block attackers

## Three-Layer Defense

```
INCOMING PACKET
    ↓
Layer 1: iptables rules check
    ├─ Is port 25 allowed? → ACCEPT
    └─ Is this port allowed? → DROP (default)
    ↓
Layer 2: ipset check (if rule references ipset)
    ├─ Is source IP in ban list? → DROP
    └─ Otherwise → ACCEPT
    ↓
Layer 3: Service receives packet (if allowed)
    └─ Fail2Ban monitors logs for failures
    └─ If threshold exceeded: adds IP to ipset
```

## Quick Commands

### iptables

```bash
# List all rules
sudo iptables -L -n

# Add rule (allow port 25)
sudo iptables -A INPUT -p tcp --dport 25 -j ACCEPT

# Delete rule
sudo iptables -D INPUT -p tcp --dport 25 -j ACCEPT

# Save rules (persistent)
sudo iptables-save > /etc/iptables/rules.v4
```

### ipset

```bash
# Create set
sudo ipset create spam_ips hash:ip

# Add IP
sudo ipset add spam_ips 192.0.2.1

# List IPs in set
sudo ipset list spam_ips

# Delete IP
sudo ipset del spam_ips 192.0.2.1

# Delete entire set
sudo ipset destroy spam_ips
```

### Fail2Ban

```bash
# Check status
sudo fail2ban-client status

# Status of specific jail
sudo fail2ban-client status dovecot

# Manually ban IP
sudo fail2ban-client set dovecot banip 192.0.2.100

# Manually unban IP
sudo fail2ban-client set dovecot unbanip 192.0.2.100

# Restart service
sudo systemctl restart fail2ban
```

## What Gets Blocked When?

| Scenario | Layer | Action | Result |
|----------|-------|--------|--------|
| **Normal IMAP login** | iptables | ALLOW (port 143) | ✓ Works |
| | ipset | Check (not in ban list) | ✓ Works |
| | Dovecot | Auth succeeds | ✓ Connected |
| **5 failed IMAP logins** | Fail2Ban | Detect pattern | ADD to ipset |
| | ipset | Add attacker IP | Stored |
| | iptables | Apply DROP rule | ← Using ipset |
| **6th IMAP attempt** | ipset | IP in ban list | DROP |
| | Result | Kernel drops packet | ✗ Blocked |
| **After 10 minutes** | Fail2Ban | Auto-unban timer | REMOVE from ipset |
| | Result | Next connection | ✓ Allowed |

## Files Involved

```
iptables
├─ /etc/iptables/rules.v4          (persistent rules)
├─ /etc/iptables/rules.v6          (IPv6 rules)
└─ Loaded on boot via iptables-persistent

ipset
├─ /etc/ipset.rules               (persistent IP sets)
└─ Loaded on boot via ipset systemd service

Fail2Ban
├─ /etc/fail2ban/jail.local       (configuration)
├─ /etc/fail2ban/filter.d/        (pattern definitions)
├─ /var/log/fail2ban.log          (activity log)
└─ Runs as daemon: fail2ban-server
```

## When Each Layer Activates

```
1. iptables (ALWAYS ACTIVE)
   └─ Every packet checked immediately
   └─ Kernel level (fastest)
   └─ Response time: microseconds

2. ipset (ALWAYS ACTIVE when referenced)
   └─ Checked by iptables if rule references ipset
   └─ Kernel level (very fast)
   └─ Response time: microseconds

3. Fail2Ban (PERIODIC SCANNING)
   └─ Monitors log files every 1-2 seconds
   └─ User space (slower but sufficient)
   └─ Detects patterns & updates ipset
   └─ Response time: 1-5 seconds
```

## Performance

- **iptables rules:** 1000s of rules = no slowdown (still microseconds)
- **ipset with 100K IPs:** Still microseconds (hash-based lookup)
- **Fail2Ban scanning:** ~1% CPU, ~50MB memory
- **Total impact:** Imperceptible to users

## Visualization

```
┌──────────────────────────────────────────────┐
│          WEBSTACK MAIL SECURITY              │
└──────────────────────────────────────────────┘

┌─ KERNEL LEVEL (FAST) ─────────────────────┐
│                                           │
│  iptables rules + ipset lists            │
│  - Check EVERY packet                    │
│  - Microsecond response time             │
│  - Automatic DROP if IP in ban list      │
│  - Survive reboot (persistent)           │
│                                           │
└───────────────────────────────────────────┘
                    ↓
┌─ USER SPACE (SLOWER BUT SUFFICIENT) ──────┐
│                                           │
│  Fail2Ban daemon                          │
│  - Scan logs every 1-2 seconds           │
│  - Detect attack patterns                 │
│  - Add/remove IPs from ipset lists       │
│  - Auto-unban after timeout              │
│                                           │
└───────────────────────────────────────────┘
                    ↓
┌─ SERVICES (PROTECTED) ────────────────────┐
│                                           │
│  Exim4 (SMTP)                            │
│  Dovecot (IMAP/POP3)                     │
│  ClamAV (Antivirus)                      │
│  SpamAssassin (Anti-spam)                │
│                                           │
│  All protected by Fail2Ban               │
│  Monitor their logs → auto-block attackers
│                                           │
└───────────────────────────────────────────┘
```

## Troubleshooting

**Issue:** Rules don't persist after reboot
- Solution: `sudo iptables-save > /etc/iptables/rules.v4`

**Issue:** Fail2Ban not banning IPs
- Check: `sudo fail2ban-client status dovecot`
- Check logs: `sudo tail -f /var/log/fail2ban.log`

**Issue:** Legitimate user getting banned
- Check: `sudo fail2ban-client status <jail>`
- Unban: `sudo fail2ban-client set <jail> unbanip <IP>`
- Whitelist: Edit `/etc/fail2ban/jail.local` → ignoreip

**Issue:** Need to temporarily disable a rule
```bash
sudo iptables -D INPUT -p tcp --dport 25 -j ACCEPT
# Rules reload on restart, or save current state
sudo iptables-save > /etc/iptables/rules.v4
```

## Summary Table

| Layer | Tool | Scope | Speed | Persistence |
|-------|------|-------|-------|-------------|
| **1** | iptables | All packets | μs | rules.v4 |
| **2** | ipset | IP matching | μs | ipset.rules |
| **3** | Fail2Ban | Log monitoring | 1-5s | Daemon |

---

**Result:** Brute-force resistant, efficient, automatic mail server security ✓

