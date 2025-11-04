#!/bin/bash
# Create ipsets for blacklists and whitelists
ipset create webstack-blacklist hash:ip -exist
ipset create webstack-whitelist hash:ip -exist

# Add any static entries (example)
# ipset add webstack-whitelist 192.0.2.1 -exist
# ipset add webstack-blacklist 203.0.113.0 -exist

# Ensure iptables rules exist to DROP traffic from blacklist (for SMTP)
iptables -C INPUT -p tcp --dport 25 -m set --match-set webstack-blacklist src -j DROP 2>/dev/null || \
iptables -I INPUT -p tcp --dport 25 -m set --match-set webstack-blacklist src -j DROP

# Allow whitelist entries (optional)
iptables -C INPUT -m set --match-set webstack-whitelist src -j ACCEPT 2>/dev/null || \
iptables -I INPUT -m set --match-set webstack-whitelist src -j ACCEPT
