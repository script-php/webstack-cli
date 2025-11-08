#!/bin/bash

# Mail Server Testing Script
# Tests Postfix, Dovecot, ClamAV, and SpamAssassin functionality

echo "📧 Mail Server Stack Testing"
echo "============================"
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Function to test service status
test_service() {
    local service=$1
    echo -n "Testing $service... "
    
    if systemctl is-active --quiet $service; then
        echo -e "${GREEN}✓ Running${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ Not running${NC}"
        ((TESTS_FAILED++))
    fi
}

# Function to test package installation
test_package() {
    local package=$1
    echo -n "Testing $package package... "
    
    if dpkg -l | grep -q "^ii  $package"; then
        echo -e "${GREEN}✓ Installed${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${YELLOW}⊘ Not installed${NC}"
    fi
}

# Function to test port listening
test_port() {
    local port=$1
    local service=$2
    echo -n "Testing $service listening on port $port... "
    
    if ss -tulpn 2>/dev/null | grep -q ":$port "; then
        echo -e "${GREEN}✓ Listening${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${YELLOW}⊘ Not listening${NC}"
    fi
}

# Function to test configuration file
test_config() {
    local file=$1
    local component=$2
    echo -n "Testing $component config file... "
    
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓ Found${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ Missing${NC}"
        ((TESTS_FAILED++))
    fi
}

echo -e "${BLUE}1. SERVICE STATUS${NC}"
echo "=================="
test_service "postfix"
test_service "dovecot"
test_package "postfix"
test_package "dovecot-core"
echo ""

echo -e "${BLUE}2. PORT LISTENING${NC}"
echo "=================="
test_port "25" "Postfix SMTP"
test_port "143" "Dovecot IMAP"
test_port "110" "Dovecot POP3"
echo ""

echo -e "${BLUE}3. CONFIGURATION FILES${NC}"
echo "======================="
test_config "/etc/postfix/main.cf" "Postfix"
test_config "/etc/postfix/main.cf.webstack" "Postfix WebStack"
test_config "/etc/dovecot/dovecot.conf" "Dovecot"
test_config "/etc/dovecot/conf.d/99-webstack.conf" "Dovecot WebStack"
echo ""

echo -e "${BLUE}4. OPTIONAL SECURITY FEATURES${NC}"
echo "=============================="
test_package "clamav-daemon"
test_package "spamassassin"

# Test ClamAV if installed
if systemctl is-active --quiet clamav-daemon 2>/dev/null; then
    echo -n "Testing ClamAV daemon... "
    echo -e "${GREEN}✓ Running${NC}"
    ((TESTS_PASSED++))
fi

# Test SpamAssassin if installed
if systemctl is-active --quiet spamd 2>/dev/null; then
    echo -n "Testing SpamAssassin daemon... "
    echo -e "${GREEN}✓ Running${NC}"
    ((TESTS_PASSED++))
fi
echo ""

echo -e "${BLUE}5. POSTFIX FUNCTIONALITY${NC}"
echo "========================"
echo -n "Testing Postfix SMTP connection... "
if timeout 2 bash -c "echo 'QUIT' | nc -w 1 localhost 25" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Connected${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}⊘ Could not connect${NC}"
fi

echo -n "Testing Postfix queue status... "
if sudo postqueue -p >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Queue accessible${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ Queue error${NC}"
    ((TESTS_FAILED++))
fi
echo ""

echo -e "${BLUE}6. DOVECOT FUNCTIONALITY${NC}"
echo "========================"
echo -n "Testing Dovecot IMAP connection... "
if timeout 2 bash -c "echo 'LOGOUT' | nc -w 1 localhost 143" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Connected${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}⊘ Could not connect${NC}"
fi

echo -n "Testing Dovecot POP3 connection... "
if timeout 2 bash -c "echo 'QUIT' | nc -w 1 localhost 110" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Connected${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}⊘ Could not connect${NC}"
fi
echo ""

echo -e "${BLUE}7. LOG FILES${NC}"
echo "============"
test_config "/var/log/mail.log" "Mail log"
test_config "/var/log/mail.err" "Mail errors"
echo ""

echo -e "${BLUE}SUMMARY${NC}"
echo "======="
echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"
echo ""

# Show important info
echo -e "${BLUE}IMPORTANT INFORMATION${NC}"
echo "====================="
echo ""
echo "📬 Postfix:"
echo "   - SMTP port: 25"
echo "   - Config: /etc/postfix/main.cf"
echo "   - Queue: sudo postqueue -p"
echo "   - Logs: tail -f /var/log/mail.log"
echo ""
echo "📩 Dovecot:"
echo "   - IMAP port: 143 (TLS: 993)"
echo "   - POP3 port: 110 (TLS: 995)"
echo "   - Config: /etc/dovecot/dovecot.conf"
echo "   - Logs: sudo journalctl -u dovecot -f"
echo ""

if dpkg -l | grep -q "^ii  clamav-daemon"; then
    echo "🦠 ClamAV:"
    echo "   - Status: Installed"
    echo "   - Update definitions: sudo freshclam"
    echo "   - Scan: clamscan /path/to/file"
    echo ""
fi

if dpkg -l | grep -q "^ii  spamassassin"; then
    echo "🚫 SpamAssassin:"
    echo "   - Status: Installed"
    echo "   - Daemon: spamd"
    echo "   - Client: spamc"
    echo "   - Config: /etc/spamassassin/local.cf"
    echo ""
fi

echo -e "${BLUE}NEXT STEPS${NC}"
echo "==========="
echo "1. Create mail users (virtual mailbox users)"
echo "2. Configure domain names in Postfix"
echo "3. Set up SSL/TLS certificates"
echo "4. Configure DNS MX records"
echo "5. Test sending/receiving emails"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All critical tests passed!${NC}"
    exit 0
else
    echo -e "${RED}⚠️  Some tests failed. Check the output above.${NC}"
    exit 1
fi
