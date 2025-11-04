# Firewall Architecture: iptables + ipset + Fail2Ban

## Overview

Your mail server now uses **enterprise-grade firewall protection** with three layers:

1. **iptables** - The kernel firewall engine
2. **ipset** - Efficient IP list management  
3. **Fail2Ban** - Automatic brute-force protection

## Layer 1: iptables (Kernel Firewall)

### What is iptables?

iptables is the **direct interface to the Linux kernel's firewall**. Every packet entering/leaving the system is processed by iptables rules.

### How Rules Work

```
PACKET ARRIVES
    ↓
iptables checks: Does this packet match a rule?
    ↓
If YES: Execute ACTION (ACCEPT, DROP, REJECT)
If NO: Check next rule
    ↓
If no rule matches: Apply default policy
```

### Mail Ports - iptables Rules

When you run `sudo ./build/webstack mail install`, these rules are added:

```bash
iptables -A INPUT -p tcp --dport 25 -j ACCEPT    # SMTP
iptables -A INPUT -p tcp --dport 143 -j ACCEPT   # IMAP
iptables -A INPUT -p tcp --dport 993 -j ACCEPT   # IMAP+SSL
iptables -A INPUT -p tcp --dport 110 -j ACCEPT   # POP3
iptables -A INPUT -p tcp --dport 995 -j ACCEPT   # POP3+SSL
iptables -A INPUT -p tcp --dport 587 -j ACCEPT   # Submission
iptables -A INPUT -p tcp --dport 465 -j ACCEPT   # SMTPS
```

### Breaking Down a Rule

```
iptables -A INPUT -p tcp --dport 25 -j ACCEPT
│       │ │     │ │  │    │      │ │
│       │ │     │ │  │    │      │ └─ ACTION: ACCEPT this packet
│       │ │     │ │  │    │      │
│       │ │     │ │  │    │      └─ Match type (destination port 25)
│       │ │     │ │  │    │
│       │ │     │ │  │    └─ Port number (SMTP)
│       │ │     │ │  │
│       │ │     │ │  └─ Protocol type (TCP)
│       │ │     │ │
│       │ │     │ └─ Direction (INPUT = incoming)
│       │ │     │
│       │ │     └─ Match if (protocol tcp, port 25)
│       │ │
│       │ └─ TABLE: INPUT chain (incoming traffic)
│       │
│       └─ ACTION: Append rule
│
└─ COMMAND: iptables
```

### Example Packet Flow

**Email arrives at port 25:**
```
CLIENT CONNECTS TO PORT 25
    ↓
iptables checks rules in order:
    1. Is it UDP? No → continue
    2. Is it TCP to port 25? YES!
    3. Action = ACCEPT
    ↓
Email server receives connection ✓
```

**Attack packet arrives at port 999:**
```
ATTACKER CONNECTS TO PORT 999
    ↓
iptables checks rules:
    1. Is it UDP? No → continue
    2. Is it TCP to port 25? No
    3. Is it TCP to port 143? No
    ... (all port rules)
    4. No rule matched, apply default policy
    ↓
Default policy = DROP (or implicit deny)
    ↓
Connection blocked ✓
```

## Layer 2: ipset (IP List Management)

### What is ipset?

ipset is a **fast IP list storage system** that works WITH iptables. Instead of creating 1000 individual iptables rules, you create ONE rule that references an ipset list.

### Why ipset is Needed

**Without ipset (BAD - slow):**
```bash
iptables -A INPUT -s 192.0.2.1 -j DROP
iptables -A INPUT -s 192.0.2.2 -j DROP
iptables -A INPUT -s 192.0.2.3 -j DROP
... (repeat 1000 times for 1000 bad IPs)
```
❌ Slow: Linear search through 1000 rules
❌ Hard to manage: Each IP needs a separate rule
❌ Memory intensive: Each rule consumes memory

**With ipset (GOOD - fast):**
```bash
# Create an IP set called "spam_ips"
ipset create spam_ips hash:ip

# Add bad IPs to the set
ipset add spam_ips 192.0.2.1
ipset add spam_ips 192.0.2.2
ipset add spam_ips 192.0.2.3
... (1000 IPs)

# Create ONE iptables rule that references the set
iptables -A INPUT -m set --match-set spam_ips src -j DROP
```
✅ Fast: Hash-based lookup (O(1) instead of O(n))
✅ Easy to manage: Add/remove IPs without touching iptables
✅ Memory efficient: One rule for 1000 IPs

### How ipset Works

```
PACKET ARRIVES WITH SOURCE IP 192.0.2.50
    ↓
iptables rule: "Drop if source in spam_ips set"
    ↓
ipset checks: Is 192.0.2.50 in spam_ips?
    ↓
Hash lookup: O(1) time - INSTANT
    ↓
If YES: DROP
If NO: Continue to next rule
```

### ipset Data Structures

```bash
# Hash set (best for random IP lookups)
ipset create spam_ips hash:ip

# List (for ordered small sets)
ipset create whitelist list:set

# Bitmap (for IP ranges)
ipset create range_spam bitmap:ip range 192.0.2.0-192.0.2.255

# Net hash (for CIDR ranges)
ipset create subnet_spam hash:net
```

### Common ipset Commands

```bash
# Create a set
ipset create spam_ips hash:ip

# Add IPs
ipset add spam_ips 192.0.2.1
ipset add spam_ips 203.0.113.0/24    # CIDR range

# Remove IPs
ipset del spam_ips 192.0.2.1

# List IPs in set
ipset list spam_ips

# Clear entire set
ipset flush spam_ips

# Delete set
ipset destroy spam_ips

# Save sets to persist after reboot
ipset save > /etc/ipset.rules
ipset restore < /etc/ipset.rules
```

## Layer 3: Fail2Ban (Automatic Blocking)

### What is Fail2Ban?

Fail2Ban is a **log monitoring daemon** that automatically detects attack patterns and blocks attacking IPs using iptables + ipset.

### How Fail2Ban Works

```
1. MONITOR LOGS
   ↓
   Watch /var/log/exim4/mainlog for:
   - Failed authentication attempts
   - Multiple connection attempts
   - Spam indicators
   
2. DETECT PATTERN
   ↓
   If 5 failed logins from same IP in 10 minutes:
   - MATCH: This is an attack pattern!
   
3. CREATE BAN
   ↓
   Add IP to ipset:
   ipset add fail2ban-exim 192.0.2.100
   
4. BLOCK WITH IPTABLES
   ↓
   iptables rule (already in place):
   iptables -A INPUT -m set --match-set fail2ban-exim src -j DROP
   
5. IP IS BLOCKED
   ↓
   All packets from 192.0.2.100 are dropped
   
6. AUTO-UNBAN
   ↓
   After 10 minutes (configurable):
   ipset del fail2ban-exim 192.0.2.100
   ↓
   Access restored
```

### Fail2Ban Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     FAIL2BAN DAEMON                     │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  FILTERS (Pattern Matching)                            │
│  ├─ exim4 filter: Detect SMTP failures                 │
│  ├─ dovecot filter: Detect IMAP/POP3 failures          │
│  └─ sshd filter: Detect SSH failures                   │
│                                                         │
│  JAILS (Action Rules)                                  │
│  ├─ exim4-prison: 5 fails → ban for 10 min             │
│  ├─ dovecot-prison: 5 fails → ban for 10 min           │
│  └─ sshd-prison: 3 fails → ban for 30 min              │
│                                                         │
│  ACTIONS (What to do on ban)                           │
│  ├─ Add IP to ipset                                    │
│  ├─ Drop packets with iptables                         │
│  └─ Send notification email                            │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

## Complete Example: Attack Flow

### Scenario: Brute-force attack on IMAP

```
ATTACKER SIDE
└─ Tries to login 10 times with wrong password

TIME    ACTION                          RESULT
────────────────────────────────────────────────────────

T=0s    Login attempt #1 (wrong pwd)    → Dovecot rejects
        Log: "login attempt #1 failed"
        Fail2Ban reads log: +1 count

T=5s    Login attempt #2 (wrong pwd)    → Dovecot rejects
        Log: "login attempt #2 failed"
        Fail2Ban reads log: count=2

T=10s   Login attempt #3 (wrong pwd)    → Dovecot rejects
        Log: "login attempt #3 failed"
        Fail2Ban reads log: count=3

T=15s   Login attempt #4 (wrong pwd)    → Dovecot rejects
        Log: "login attempt #4 failed"
        Fail2Ban reads log: count=4

T=20s   Login attempt #5 (wrong pwd)    → Dovecot rejects
        Log: "login attempt #5 failed"
        Fail2Ban reads log: count=5 ← THRESHOLD REACHED!

        ⚠️ FAIL2BAN ACTION TRIGGERED:
        1. ipset add fail2ban-dovecot 192.0.2.100
        2. iptables rule now active for this IP
        3. Email sent: "192.0.2.100 banned - 5 failures"

T=21s   Login attempt #6                → TCP connection DROPPED
        Packet never reaches Dovecot!   ✓ BLOCKED

T=22s   Login attempt #7                → TCP connection DROPPED ✓

...

T=600s  (10 minutes later)              

        ⏰ AUTO-UNBAN TIMER EXPIRES:
        1. ipset del fail2ban-dovecot 192.0.2.100
        2. Reset counter to 0
        3. Access restored (if attacker still trying)
```

## Configuration Files

### /etc/fail2ban/jail.local

```ini
[DEFAULT]
bantime = 600              # Ban for 10 minutes
findtime = 600             # Time window for counting failures
maxretry = 5               # Ban after 5 failures

[exim4]
enabled = true
port = 25,465,587
filter = exim4
logpath = /var/log/exim4/mainlog
maxretry = 5
bantime = 600

[dovecot]
enabled = true
port = 143,993,110,995
filter = dovecot
logpath = /var/log/mail.log
maxretry = 5
bantime = 600
```

### /etc/fail2ban/filter.d/exim4.conf

```
[Definition]
failregex = authentication failed for .* \[<HOST>\]
            Failed SMTP authentication
ignoreregex =
```

### /etc/ipset.rules (Persistent IP Sets)

```
create spam_ips hash:ip
create fail2ban-exim hash:ip
create fail2ban-dovecot hash:ip
add spam_ips 192.0.2.50
add spam_ips 203.0.113.100
```

## How They Work Together - Complete Flow

### Normal User Connection

```
LEGITIMATE USER (192.0.2.200)
│
├─ Attempts to connect to port 143 (IMAP)
│
├─ iptables checks: Port 143 in allow list? YES ✓
│
├─ ipset checks: Is 192.0.2.200 in any ban set? NO ✓
│
├─ Dovecot receives connection ✓
│
├─ User enters credentials
│
├─ Dovecot verifies (succeeds)
│
├─ Fail2Ban sees: Valid authentication ✓
│   └─ Counter reset to 0
│
└─ User connected to IMAP ✓
```

### Attacker Connection

```
ATTACKER (192.0.2.100)
│
├─ Attempt 1: Connect to port 143
│  ├─ iptables: Port 143 allowed? YES ✓
│  ├─ ipset: IP banned? NO ✓
│  ├─ Dovecot receives connection ✓
│  └─ Fail2Ban: Count=1 (failure detected)
│
├─ Attempt 2-4: Same as above, Count=2,3,4
│
├─ Attempt 5: 
│  ├─ Dovecot: Rejects (wrong password)
│  ├─ Fail2Ban: Count=5 ← THRESHOLD!
│  ├─ ACTION: ipset add fail2ban-dovecot 192.0.2.100
│  └─ Logs: "Banned 192.0.2.100"
│
├─ Attempt 6: 
│  ├─ iptables checks: Port 143 allowed? YES ✓
│  ├─ ipset checks: Is 192.0.2.100 in fail2ban-dovecot? YES!
│  ├─ ACTION: DROP (per iptables rule)
│  └─ Connection rejected at kernel level ✓ (Dovecot never sees it)
│
├─ Attempt 7-N: All dropped ✓
│
└─ After 10 minutes: Auto-unban, counter reset
```

## Checking Status

### View Current iptables Rules

```bash
$ sudo iptables -L -n

Chain INPUT (policy ACCEPT)
target     prot opt source               destination
ACCEPT     tcp  --  0.0.0.0/0            0.0.0.0/0     tcp dpt:25
ACCEPT     tcp  --  0.0.0.0/0            0.0.0.0/0     tcp dpt:143
ACCEPT     tcp  --  0.0.0.0/0            0.0.0.0/0     tcp dpt:587
... (mail ports)
DROP       all  --  0.0.0.0/0            0.0.0.0/0     match-set fail2ban-dovecot src
DROP       all  --  0.0.0.0/0            0.0.0.0/0     match-set fail2ban-exim src
```

### View ipset Lists

```bash
$ sudo ipset list

Name: fail2ban-dovecot
Type: hash:ip
Revision: 4
Header: family inet hashsize 1024 maxelem 65536
Size in memory: 16504
References: 1
Members:
192.0.2.100
203.0.113.50
```

### View Fail2Ban Status

```bash
$ sudo fail2ban-client status

Status
|- Number of jail:    3
`- Jail list:         exim4, dovecot, sshd

$ sudo fail2ban-client status dovecot

Status for the dovecot jail:
|- Filter
|  |- Currently failed:  1
|  |- Total failed:     12
|  `- File list:        /var/log/mail.log
|- Actions
|  |- Currently banned: 2
|  |- Total banned:     4
|  `- Banned IP list:   192.0.2.100 203.0.113.50
```

### View Logs

```bash
# Fail2Ban events
$ sudo tail -f /var/log/fail2ban.log

2025-11-04 10:15:32 WARNING [dovecot] Ban 192.0.2.100
2025-11-04 10:15:33 WARNING [dovecot] Unban 203.0.113.50

# Authentication failures
$ sudo tail -f /var/log/exim4/mainlog

2025-11-04 10:14:15 authentication failed for user@example.com [192.0.2.100]
```

## Performance Impact

| Component | Memory | CPU | Latency |
|-----------|--------|-----|---------|
| iptables rules | ~2KB per rule | Negligible | 0-1μs |
| ipset lookup | ~500KB base + 16B per IP | Negligible | 0-10μs |
| Fail2Ban scanning | ~50MB | ~1% CPU | ~1-2s (per log scan) |
| **Total** | **~100MB** | **~2% CPU** | **Imperceptible** |

## Security Benefits

✅ **Automatic brute-force protection** - Ban after N failures  
✅ **Real-time blocking** - Dropped at kernel level  
✅ **Efficient** - ipset handles 100K+ IPs with no slowdown  
✅ **Customizable** - Adjust thresholds per service  
✅ **Persistent** - Rules survive reboot  
✅ **Visible** - Easy to monitor and audit  
✅ **Reversible** - Easy to whitelist legitimate IPs  

## Common Tasks

### Whitelist an IP

```bash
# Create whitelist
ipset create whitelist hash:ip
ipset add whitelist 203.0.114.1

# Tell Fail2Ban to ignore it
# Edit /etc/fail2ban/jail.local:
ignoreip = 127.0.0.1/8 203.0.114.1
```

### Manually Ban an IP

```bash
ipset add fail2ban-exim 192.0.2.100
```

### Manually Unban an IP

```bash
ipset del fail2ban-exim 192.0.2.100
```

### Adjust Ban Duration

```bash
# Edit /etc/fail2ban/jail.local
bantime = 1800  # 30 minutes instead of 10

# Restart
sudo systemctl restart fail2ban
```

### View Real-Time Activity

```bash
$ sudo watch -n1 'fail2ban-client status dovecot'

Status for the dovecot jail:
|- Filter
|  |- Currently failed:  3    ← How many current failures
|  |- Total failed:     145   ← Total since service start
|  `- File list:        /var/log/mail.log
|- Actions
|  |- Currently banned: 2     ← IPs currently banned
|  |- Total banned:    23     ← Total bans issued
|  `- Banned IP list:   192.0.2.100 203.0.113.50
```

---

**Summary:** Your mail server now has three-layer protection:
1. **iptables** = Direct kernel firewall (allow/deny rules)
2. **ipset** = Fast IP list management (efficient blocking)
3. **Fail2Ban** = Automatic brute-force detection & blocking

All three work together seamlessly to protect your mail services! 🔒

