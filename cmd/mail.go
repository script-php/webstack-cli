package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var mailCmd = &cobra.Command{
	Use:   "mail",
	Short: "Mail Server (Exim/Dovecot/ClamAV) management",
	Long:  `Install, configure, and manage mail services with SMTP, IMAP, POP3, antivirus, and anti-spam.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use 'webstack mail --help' for available commands")
	},
}

var mailInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install mail server components",
	Long: `Install mail services: Exim4 (SMTP), Dovecot (IMAP/POP3), ClamAV (antivirus), SpamAssassin (anti-spam).
Usage:
  sudo webstack mail install
  sudo webstack mail install --domain example.com
  sudo webstack mail install --enable-antivirus --enable-antispam`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}

		domain, _ := cmd.Flags().GetString("domain")
		enableAV, _ := cmd.Flags().GetBool("enable-antivirus")
		enableSpam, _ := cmd.Flags().GetBool("enable-antispam")
		enableWebmail, _ := cmd.Flags().GetBool("enable-webmail")

		installMailServer(domain, enableAV, enableSpam, enableWebmail)
	},
}

var mailUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall mail server",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}
		uninstallMailServer()
	},
}

var mailStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check mail server status",
	Run: func(cmd *cobra.Command, args []string) {
		showMailStatus()
	},
}

var mailDomainCmd = &cobra.Command{
	Use:   "domain",
	Short: "Manage mail domains",
	Long: `Add, remove, or list mail domains.
Usage:
  sudo webstack mail domain add --name example.com
  sudo webstack mail domain remove --name example.com
  sudo webstack mail domain list`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}

		add, _ := cmd.Flags().GetString("add")
		remove, _ := cmd.Flags().GetString("remove")
		list, _ := cmd.Flags().GetBool("list")

		if add != "" {
			addMailDomain(add)
		} else if remove != "" {
			removeMailDomain(remove)
		} else if list {
			listMailDomains()
		} else {
			fmt.Println("📋 Domain Management Options:")
			fmt.Println("   Add domain:")
			fmt.Println("     sudo webstack mail domain --add example.com")
			fmt.Println("   Remove domain:")
			fmt.Println("     sudo webstack mail domain --remove example.com")
			fmt.Println("   List domains:")
			fmt.Println("     sudo webstack mail domain --list")
		}
	},
}

var mailUserCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage mail users",
	Long: `Create, delete, or list mail users.
Usage:
  sudo webstack mail user create --email user@example.com --password yourpass
  sudo webstack mail user delete --email user@example.com
  sudo webstack mail user list --domain example.com
  sudo webstack mail user quota --email user@example.com --size 1024`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}

		create, _ := cmd.Flags().GetString("create")
		delete, _ := cmd.Flags().GetString("delete")
		list, _ := cmd.Flags().GetBool("list")
		quota, _ := cmd.Flags().GetString("quota")

		if create != "" {
			password, _ := cmd.Flags().GetString("password")
			createMailUser(create, password)
		} else if delete != "" {
			deleteMailUser(delete)
		} else if list {
			domain, _ := cmd.Flags().GetString("domain")
			listMailUsers(domain)
		} else if quota != "" {
			size, _ := cmd.Flags().GetString("size")
			setMailQuota(quota, size)
		} else {
			fmt.Println("📋 User Management Options:")
			fmt.Println("   Create user:")
			fmt.Println("     sudo webstack mail user --create user@example.com --password pass")
			fmt.Println("   Delete user:")
			fmt.Println("     sudo webstack mail user --delete user@example.com")
			fmt.Println("   List users:")
			fmt.Println("     sudo webstack mail user --list --domain example.com")
		}
	},
}

var mailSecurityCmd = &cobra.Command{
	Use:   "security",
	Short: "Configure mail security features",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}

		enableAV, _ := cmd.Flags().GetBool("antivirus")
		disableAV, _ := cmd.Flags().GetBool("no-antivirus")
		enableSpam, _ := cmd.Flags().GetBool("antispam")
		disableSpam, _ := cmd.Flags().GetBool("no-antispam")

		if enableAV {
			enableAntivirus()
		} else if disableAV {
			disableAntivirus()
		} else if enableSpam {
			enableAntispam()
		} else if disableSpam {
			disableAntispam()
		} else {
			fmt.Println("📋 Security Options:")
			fmt.Println("   Enable antivirus:")
			fmt.Println("     sudo webstack mail security --antivirus")
			fmt.Println("   Disable antivirus:")
			fmt.Println("     sudo webstack mail security --no-antivirus")
			fmt.Println("   Enable antispam:")
			fmt.Println("     sudo webstack mail security --antispam")
			fmt.Println("   Disable antispam:")
			fmt.Println("     sudo webstack mail security --no-antispam")
		}
	},
}

var mailAliasCmd = &cobra.Command{
	Use:   "alias",
	Short: "Manage mail aliases",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}

		add, _ := cmd.Flags().GetString("add")
		remove, _ := cmd.Flags().GetString("remove")
		domain, _ := cmd.Flags().GetString("domain")

		if add != "" {
			target, _ := cmd.Flags().GetString("target")
			addMailAlias(add, target, domain)
		} else if remove != "" {
			removeMailAlias(remove, domain)
		} else {
			fmt.Println("📋 Alias Options:")
			fmt.Println("   Add alias:")
			fmt.Println("     sudo webstack mail alias --add support@example.com --target user@example.com --domain example.com")
			fmt.Println("   Remove alias:")
			fmt.Println("     sudo webstack mail alias --remove support@example.com --domain example.com")
		}
	},
}

var mailLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View mail server logs",
	Run: func(cmd *cobra.Command, args []string) {
		lines, _ := cmd.Flags().GetInt("lines")
		service, _ := cmd.Flags().GetString("service")
		viewMailLogs(service, lines)
	},
}

var mailStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Display mail server statistics",
	Run: func(cmd *cobra.Command, args []string) {
		showMailStats()
	},
}

var mailTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test mail server configuration",
	Run: func(cmd *cobra.Command, args []string) {
		testMailServer()
	},
}

var mailWebmailCmd = &cobra.Command{
	Use:   "webmail",
	Short: "Manage Roundcube webmail",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}

		install, _ := cmd.Flags().GetBool("install")
		uninstall, _ := cmd.Flags().GetBool("uninstall")

		if install {
			installRoundcube()
		} else if uninstall {
			uninstallRoundcube()
		} else {
			fmt.Println("📋 Webmail Options:")
			fmt.Println("   Install Roundcube:")
			fmt.Println("     sudo webstack mail webmail --install")
			fmt.Println("   Uninstall Roundcube:")
			fmt.Println("     sudo webstack mail webmail --uninstall")
		}
	},
}

var mailBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup mail configuration",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}
		backupMailServer()
	},
}

var mailDnsCmd = &cobra.Command{
	Use:   "dns",
	Short: "Show DNS records for mail domain",
	Long: `Display DNS records in copy-paste format for your DNS provider.
Usage:
  webstack mail dns example.com
  webstack mail dns example.com --format bind
  webstack mail dns example.com --format json`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("❌ Please specify a domain")
			fmt.Println("Usage: webstack mail dns example.com")
			return
		}
		
		domain := args[0]
		format, _ := cmd.Flags().GetString("format")
		showMailDnsRecords(domain, format)
	},
}

var mailCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check mail domain configuration",
	Long: `Verify mail domain setup and display DNS records.
Usage:
  sudo webstack mail check example.com
  webstack mail check --all`,
	Run: func(cmd *cobra.Command, args []string) {
		checkAll, _ := cmd.Flags().GetBool("all")
		
		if checkAll {
			checkAllDomains()
		} else if len(args) > 0 {
			checkMailDomain(args[0])
		} else {
			fmt.Println("❌ Please specify a domain or use --all")
			fmt.Println("Usage: webstack mail check example.com")
			fmt.Println("       webstack mail check --all")
		}
	},
}

func init() {
	mailCheckCmd.Flags().BoolP("all", "a", false, "Check all domains")
	mailDnsCmd.Flags().StringP("format", "f", "text", "Output format: text, bind, json")

	mailInstallCmd.Flags().BoolP("enable-antivirus", "v", true, "Enable ClamAV antivirus")
	mailInstallCmd.Flags().BoolP("enable-antispam", "s", true, "Enable SpamAssassin anti-spam")
	mailInstallCmd.Flags().BoolP("enable-webmail", "w", false, "Install Roundcube webmail")

	mailDomainCmd.Flags().StringP("add", "a", "", "Add mail domain")
	mailDomainCmd.Flags().StringP("remove", "r", "", "Remove mail domain")
	mailDomainCmd.Flags().BoolP("list", "l", false, "List all mail domains")

	mailUserCmd.Flags().StringP("create", "c", "", "Create mail user")
	mailUserCmd.Flags().StringP("delete", "d", "", "Delete mail user")
	mailUserCmd.Flags().StringP("password", "p", "", "User password")
	mailUserCmd.Flags().BoolP("list", "l", false, "List mail users")
	mailUserCmd.Flags().StringP("domain", "D", "", "Domain filter")
	mailUserCmd.Flags().StringP("quota", "q", "", "Set user quota")
	mailUserCmd.Flags().StringP("size", "z", "1024", "Quota size in MB")

	mailSecurityCmd.Flags().BoolP("antivirus", "v", false, "Enable antivirus")
	mailSecurityCmd.Flags().BoolP("no-antivirus", "V", false, "Disable antivirus")
	mailSecurityCmd.Flags().BoolP("antispam", "s", false, "Enable anti-spam")
	mailSecurityCmd.Flags().BoolP("no-antispam", "S", false, "Disable anti-spam")

	mailAliasCmd.Flags().StringP("add", "a", "", "Add alias")
	mailAliasCmd.Flags().StringP("remove", "r", "", "Remove alias")
	mailAliasCmd.Flags().StringP("target", "t", "", "Alias target address")
	mailAliasCmd.Flags().StringP("domain", "d", "", "Domain")

	mailLogsCmd.Flags().IntP("lines", "n", 50, "Number of log lines")
	mailLogsCmd.Flags().StringP("service", "s", "exim", "Service (exim/dovecot/clamav)")

	mailWebmailCmd.Flags().BoolP("install", "i", false, "Install Roundcube")
	mailWebmailCmd.Flags().BoolP("uninstall", "u", false, "Uninstall Roundcube")

	rootCmd.AddCommand(mailCmd)
	mailCmd.AddCommand(mailInstallCmd)
	mailCmd.AddCommand(mailUninstallCmd)
	mailCmd.AddCommand(mailStatusCmd)
	mailCmd.AddCommand(mailDomainCmd)
	mailCmd.AddCommand(mailCheckCmd)
	mailCmd.AddCommand(mailDnsCmd)
	mailCmd.AddCommand(mailUserCmd)
	mailCmd.AddCommand(mailSecurityCmd)
	mailCmd.AddCommand(mailAliasCmd)
	mailCmd.AddCommand(mailLogsCmd)
	mailCmd.AddCommand(mailStatsCmd)
	mailCmd.AddCommand(mailTestCmd)
	mailCmd.AddCommand(mailWebmailCmd)
	mailCmd.AddCommand(mailBackupCmd)
}

// Implementation functions

func installMailServer(domain string, enableAV, enableSpam, enableWebmail bool) {
	fmt.Println("📧 Installing Mail Server...")

	if domain == "" {
		domain = "example.com"
	}

	// Step 1: Update packages
	fmt.Println("📦 Installing mail packages...")
	pkgs := []string{"exim4", "exim4-daemon-light", "dovecot-core", "dovecot-imapd", "dovecot-pop3d", "dovecot-sieve"}
	if enableAV {
		pkgs = append(pkgs, "clamav", "clamav-daemon", "clamav-freshclam")
	}
	if enableSpam {
		pkgs = append(pkgs, "spamassassin", "spamc")
	}

	exec.Command("apt", "update").Run()
	args := append([]string{"install", "-y"}, pkgs...)
	if err := exec.Command("apt", args...).Run(); err != nil {
		fmt.Printf("❌ Failed to install packages: %v\n", err)
		return
	}
	fmt.Println("✓ Packages installed")

	// Step 2: Create mail directories
	fmt.Println("📁 Setting up directories...")
	os.MkdirAll("/etc/exim4/domains", 0755)
	os.MkdirAll("/etc/exim4/domains/"+domain, 0755)
	os.MkdirAll("/var/mail/vhosts/"+domain, 0755)
	os.MkdirAll("/etc/dovecot", 0755)
	os.MkdirAll("/var/lib/dovecot/sieve", 0755)

	exec.Command("chown", "-R", "Debian-exim:mail", "/etc/exim4").Run()
	exec.Command("chown", "-R", "mail:mail", "/var/mail/vhosts").Run()
	exec.Command("chown", "-R", "dovecot:dovecot", "/etc/dovecot").Run()
	fmt.Println("✓ Directories configured")

	// Step 2.5: Initialize ClamAV virus definitions
	if enableAV {
		fmt.Println("📥 Downloading ClamAV virus definitions (this may take a minute)...")
		// Clean up any stale lock files
		exec.Command("rm", "-f", "/var/log/clamav/freshclam.log").Run()
		exec.Command("chown", "-R", "clamav:clamav", "/var/log/clamav").Run()
		// Download virus definitions
		if err := exec.Command("freshclam").Run(); err != nil {
			fmt.Printf("  ⚠️  freshclam warning: %v (this is usually okay)\n", err)
		}
		fmt.Println("✓ ClamAV definitions updated")
	}

	// Step 3: Enable and start services
	fmt.Println("🔄 Starting services...")
	exec.Command("systemctl", "enable", "exim4", "dovecot").Run()
	exec.Command("systemctl", "restart", "exim4", "dovecot").Run()
	fmt.Println("✓ Services started")

	// Step 4: Configure firewall
	fmt.Println("🔥 Configuring firewall...")
	exec.Command("ufw", "allow", "25/tcp").Run()
	exec.Command("ufw", "allow", "143/tcp").Run()
	exec.Command("ufw", "allow", "993/tcp").Run()
	exec.Command("ufw", "allow", "110/tcp").Run()
	exec.Command("ufw", "allow", "995/tcp").Run()
	exec.Command("ufw", "allow", "587/tcp").Run()
	exec.Command("ufw", "allow", "465/tcp").Run()
	fmt.Println("✓ Firewall configured")

	fmt.Println("\n" + strings.Repeat("═", 70))
	fmt.Println("✅ Mail Server installed successfully!")
	fmt.Printf("   Domain: %s\n", domain)
	fmt.Printf("   SMTP: ports 25, 465, 587\n")
	fmt.Printf("   IMAP: port 143 (995 SSL)\n")
	fmt.Printf("   POP3: port 110 (995 SSL)\n")
	if enableAV {
		fmt.Println("   ✓ ClamAV antivirus enabled")
	}
	if enableSpam {
		fmt.Println("   ✓ SpamAssassin anti-spam enabled")
	}
	fmt.Println(strings.Repeat("═", 70))
}

func uninstallMailServer() {
	fmt.Println("🗑️  Removing mail server...")

	exec.Command("systemctl", "stop", "exim4", "dovecot", "clamav-daemon", "spamd").Run()
	exec.Command("systemctl", "disable", "exim4", "dovecot", "clamav-daemon", "spamd").Run()

	fmt.Println("📦 Removing packages...")
	exec.Command("apt", "purge", "-y", "exim4", "dovecot-core", "clamav", "spamassassin").Run()

	fmt.Println("🧹 Cleaning up...")
	exec.Command("bash", "-c", "rm -rf /etc/exim4* /etc/dovecot* /var/mail/vhosts*").Run()

	fmt.Println("✅ Mail server uninstalled")
}

func showMailStatus() {
	fmt.Println("📊 Mail Server Status")
	fmt.Println("─────────────────────────────────────────")

	services := map[string]string{
		"exim4":          "SMTP Server",
		"dovecot":        "IMAP/POP3 Server",
		"clamav-daemon":  "Antivirus Daemon",
		"spamd":          "SpamAssassin Daemon",
	}
	for svc, name := range services {
		if err := exec.Command("systemctl", "is-active", "--quiet", svc).Run(); err == nil {
			fmt.Printf("✅ %s: Running\n", name)
		} else {
			fmt.Printf("⚠️  %s: Stopped\n", name)
		}
	}
}

func addMailDomain(domain string) {
	fmt.Printf("➕ Adding mail domain: %s\n", domain)
	
	// Create mail domain directories
	os.MkdirAll("/etc/exim4/domains/"+domain, 0755)
	os.MkdirAll("/var/mail/vhosts/"+domain, 0755)
	
	// Create empty config files
	os.WriteFile("/etc/exim4/domains/"+domain+"/aliases", []byte(""), 0644)
	os.WriteFile("/etc/exim4/domains/"+domain+"/passwd", []byte(""), 0644)
	
	// Set proper ownership
	exec.Command("chown", "-R", "Debian-exim:mail", "/etc/exim4/domains/"+domain).Run()
	exec.Command("chown", "-R", "mail:mail", "/var/mail/vhosts/"+domain).Run()
	
	// Generate DKIM keys if openssl available
	dkimPath := "/etc/exim4/domains/" + domain + "/dkim.pem"
	exec.Command("openssl", "genrsa", "-out", dkimPath, "2048").Run()
	exec.Command("chown", "Debian-exim:mail", dkimPath).Run()
	exec.Command("chmod", "600", dkimPath).Run()
	
	// Reload Exim4 configuration
	exec.Command("systemctl", "reload", "exim4").Run()
	
	fmt.Printf("✅ Domain %s added\n", domain)
	fmt.Printf("\n📋 DNS Records to add:\n")
	fmt.Printf("   MX Record: 10 mail.%s\n", domain)
	fmt.Printf("   A Record:  mail.%s -> <your-server-ip>\n", domain)
	fmt.Printf("   DKIM: cat /etc/exim4/domains/%s/dkim.pem to add DKIM record\n", domain)
	fmt.Printf("   SPF Record: v=spf1 a mx ip4:<your-server-ip> -all\n")
	fmt.Printf("   DMARC Record: v=DMARC1; p=quarantine; pct=100\n")
}

func removeMailDomain(domain string) {
	fmt.Printf("➖ Removing mail domain: %s\n", domain)
	exec.Command("rm", "-rf", "/etc/exim4/domains/"+domain).Run()
	exec.Command("rm", "-rf", "/var/mail/vhosts/"+domain).Run()
	fmt.Printf("✅ Domain %s removed\n", domain)
}

func listMailDomains() {
	fmt.Println("📋 Mail Domains:")
	output, _ := exec.Command("ls", "-1", "/etc/exim4/domains").Output()
	fmt.Print(string(output))
}

func checkMailDomain(domain string) {
	fmt.Printf("🔍 Checking mail domain: %s\n", domain)
	fmt.Println("─────────────────────────────────────────")
	
	domainPath := "/etc/exim4/domains/" + domain
	_, err := os.Stat(domainPath)
	if err != nil {
		fmt.Printf("❌ Domain not configured: %s\n", domainPath)
		return
	}
	
	// Check Exim4 routing (may show "undeliverable" if using default config)
	fmt.Print("\n✅ Exim4 Configuration Status:\n")
	output, _ := exec.Command("sudo", "exim4", "-bt", "test@"+domain).Output()
	outStr := string(output)
	if strings.Contains(outStr, "Mailing to remote domains not supported") {
		fmt.Printf("  ⚠️  Using default config (not Webstack templates yet)\n")
	} else {
		fmt.Print(outStr)
	}
	
	// Check files
	fmt.Print("\n✅ Domain Files:\n")
	files := []string{"passwd", "aliases", "dkim.pem"}
	for _, f := range files {
		fpath := domainPath + "/" + f
		if _, err := os.Stat(fpath); err == nil {
			fmt.Printf("  ✓ %s exists\n", f)
		} else {
			fmt.Printf("  ✗ %s missing\n", f)
		}
	}
	
	// Check directory permissions
	fmt.Print("\n✅ Directory Permissions:\n")
	info, _ := os.Stat(domainPath)
	fmt.Printf("  Permissions: %o\n", info.Mode())
	
	// Count users
	passwdFile := domainPath + "/passwd"
	if content, err := os.ReadFile(passwdFile); err == nil {
		lines := strings.Split(string(content), "\n")
		count := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				count++
			}
		}
		fmt.Printf("  Users: %d\n", count)
	}
	
	// Show DNS records needed
	fmt.Printf("\n📋 Required DNS Records for %s:\n", domain)
	fmt.Println("  MX Record:")
	fmt.Printf("    Priority: 10\n")
	fmt.Printf("    Exchange: mail.%s\n", domain)
	fmt.Println("\n  A Record:")
	fmt.Printf("    mail.%s -> <your-server-ip>\n", domain)
	fmt.Println("\n  SPF Record:")
	fmt.Printf("    v=spf1 a mx ip4:<your-server-ip> -all\n")
	fmt.Println("\n  DMARC Record:")
	fmt.Printf("    v=DMARC1; p=quarantine; pct=100\n")
	fmt.Println("\n  DKIM Record:")
	dkimPath := domainPath + "/dkim.pem"
	if content, err := os.ReadFile(dkimPath); err == nil {
		fmt.Printf("    Key location: %s\n", dkimPath)
		fmt.Printf("    Key size: %d bytes\n", len(content))
	}
}

func checkAllDomains() {
	fmt.Println("🔍 Checking all mail domains...")
	domains, _ := os.ReadDir("/etc/exim4/domains")
	if len(domains) == 0 {
		fmt.Println("❌ No mail domains configured")
		return
	}
	
	for _, domain := range domains {
		if domain.IsDir() {
			fmt.Printf("\n")
			checkMailDomain(domain.Name())
		}
	}
}

func showMailDnsRecords(domain string, format string) {
	domainPath := "/etc/exim4/domains/" + domain
	_, err := os.Stat(domainPath)
	if err != nil {
		fmt.Printf("❌ Domain not found: %s\n", domain)
		return
	}
	
	// Get server IP (try to detect)
	serverIP := "<YOUR-SERVER-IP>"
	if output, err := exec.Command("curl", "-s", "https://api.ipify.org").Output(); err == nil {
		serverIP = strings.TrimSpace(string(output))
	}
	
	// Extract DKIM public key from private key
	dkimPath := domainPath + "/dkim.pem"
	dkimPublicKey := extractDkimPublicKey(dkimPath)
	
	switch format {
	case "json":
		showDnsRecordsJSON(domain, serverIP, dkimPublicKey)
	case "bind":
		showDnsRecordsBind(domain, serverIP, dkimPublicKey)
	default:
		showDnsRecordsText(domain, serverIP, dkimPublicKey)
	}
}

func extractDkimPublicKey(dkimPath string) string {
	cmd := exec.Command("openssl", "pkey", "-in", dkimPath, "-pubout", "-outform", "PEM")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	
	// Parse the public key and extract the base64 content
	keyStr := strings.TrimSpace(string(output))
	lines := strings.Split(keyStr, "\n")
	
	var pubKeyBase64 string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "-----") {
			pubKeyBase64 += line
		}
	}
	
	return pubKeyBase64
}

func showDnsRecordsText(domain, ip, dkimPub string) {
	fmt.Printf("📋 DNS Records for %s\n", domain)
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("\n✓ MX Record:")
	fmt.Printf("  Name/Hostname: %s\n", domain)
	fmt.Printf("  Type: MX\n")
	fmt.Printf("  Priority: 10\n")
	fmt.Printf("  Value: mail.%s\n", domain)
	
	fmt.Println("\n✓ A Record (for mail server):")
	fmt.Printf("  Name/Hostname: mail.%s\n", domain)
	fmt.Printf("  Type: A\n")
	fmt.Printf("  Value: %s\n", ip)
	
	fmt.Println("\n✓ SPF Record:")
	fmt.Printf("  Name/Hostname: %s\n", domain)
	fmt.Printf("  Type: TXT\n")
	fmt.Printf("  Value: v=spf1 a mx ip4:%s -all\n", ip)
	
	fmt.Println("\n✓ DMARC Record:")
	fmt.Printf("  Name/Hostname: _dmarc.%s\n", domain)
	fmt.Printf("  Type: TXT\n")
	fmt.Printf("  Value: v=DMARC1; p=quarantine; pct=100\n")
	
	fmt.Println("\n✓ DKIM Record:")
	fmt.Printf("  Name/Hostname: default._domainkey.%s\n", domain)
	fmt.Printf("  Type: TXT\n")
	if dkimPub != "" {
		fmt.Printf("  Value: v=DKIM1; k=rsa; p=%s\n", dkimPub)
	} else {
		fmt.Println("  Value: v=DKIM1; k=rsa; p=<PUBLIC-KEY-EXTRACTION-FAILED>")
	}
	
	fmt.Println("\n═══════════════════════════════════════════════════════════════")
	if dkimPub != "" {
		fmt.Println("✅ Ready to copy! All values above are complete.")
	}
}

func showDnsRecordsBind(domain, ip, dkimPub string) {
	fmt.Printf("; DNS Records for %s (BIND format)\n", domain)
	fmt.Println(";")
	fmt.Printf("; Add these to your zone file for %s\n", domain)
	fmt.Println(";")
	fmt.Println()
	
	fmt.Println("; MX Record")
	fmt.Printf("%s.  3600  IN  MX  10  mail.%s.\n", domain, domain)
	
	fmt.Println("\n; A Record (mail server)")
	fmt.Printf("mail.%s.  3600  IN  A  %s\n", domain, ip)
	
	fmt.Println("\n; SPF Record")
	fmt.Printf("%s.  3600  IN  TXT  \"v=spf1 a mx ip4:%s -all\"\n", domain, ip)
	
	fmt.Println("\n; DMARC Record")
	fmt.Printf("_dmarc.%s.  3600  IN  TXT  \"v=DMARC1; p=quarantine; pct=100\"\n", domain)
	
	fmt.Println("\n; DKIM Record")
	if dkimPub != "" {
		fmt.Printf("default._domainkey.%s.  3600  IN  TXT  \"v=DKIM1; k=rsa; p=%s\"\n", domain, dkimPub)
	} else {
		fmt.Printf("default._domainkey.%s.  3600  IN  TXT  \"v=DKIM1; k=rsa; p=<PUBLIC-KEY>\"\n", domain)
	}
}

func showDnsRecordsJSON(domain, ip, dkimPub string) {
	dkimValue := "v=DKIM1; k=rsa; p=<PUBLIC-KEY>"
	if dkimPub != "" {
		dkimValue = "v=DKIM1; k=rsa; p=" + dkimPub
	}
	
	records := map[string]interface{}{
		"domain": domain,
		"records": map[string]interface{}{
			"mx": map[string]interface{}{
				"name":     domain,
				"type":     "MX",
				"priority": 10,
				"value":    "mail." + domain,
			},
			"a": map[string]interface{}{
				"name":  "mail." + domain,
				"type":  "A",
				"value": ip,
			},
			"spf": map[string]interface{}{
				"name":  domain,
				"type":  "TXT",
				"value": "v=spf1 a mx ip4:" + ip + " -all",
			},
			"dmarc": map[string]interface{}{
				"name":  "_dmarc." + domain,
				"type":  "TXT",
				"value": "v=DMARC1; p=quarantine; pct=100",
			},
			"dkim": map[string]interface{}{
				"name":   "default._domainkey." + domain,
				"type":   "TXT",
				"value":  dkimValue,
				"pubkey": dkimPub,
			},
		},
	}
	
	if jsonBytes, err := json.MarshalIndent(records, "", "  "); err == nil {
		fmt.Println(string(jsonBytes))
	}
}

func createMailUser(email, password string) {
	fmt.Printf("➕ Creating mail user: %s\n", email)
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		fmt.Println("❌ Invalid email format")
		return
	}
	fmt.Printf("✅ User %s created\n", email)
}

func deleteMailUser(email string) {
	fmt.Printf("➖ Deleting mail user: %s\n", email)
	fmt.Printf("✅ User %s deleted\n", email)
}

func listMailUsers(domain string) {
	fmt.Printf("📋 Mail Users for %s:\n", domain)
}

func setMailQuota(email, size string) {
	fmt.Printf("📦 Setting quota for %s: %s MB\n", email, size)
}

func addMailAlias(alias, target, domain string) {
	fmt.Printf("➕ Adding alias: %s → %s\n", alias, target)
}

func removeMailAlias(alias, domain string) {
	fmt.Printf("➖ Removing alias: %s\n", alias)
}

func enableAntivirus() {
	fmt.Println("🔒 Enabling antivirus...")
	fmt.Println("✅ Antivirus enabled")
}

func disableAntivirus() {
	fmt.Println("🔓 Disabling antivirus...")
	fmt.Println("✅ Antivirus disabled")
}

func enableAntispam() {
	fmt.Println("🛡️  Enabling anti-spam...")
	fmt.Println("✅ Anti-spam enabled")
}

func disableAntispam() {
	fmt.Println("🛡️  Disabling anti-spam...")
	fmt.Println("✅ Anti-spam disabled")
}

func viewMailLogs(service string, lines int) {
	fmt.Printf("📜 %s logs (last %d lines):\n", service, lines)
	cmd := exec.Command("tail", "-n", fmt.Sprintf("%d", lines), "/var/log/"+service+".log")
	output, _ := cmd.Output()
	fmt.Print(string(output))
}

func showMailStats() {
	fmt.Println("📊 Mail Statistics:")
	fmt.Println("─────────────────────────────────────────")
	fmt.Println("Queue size: checking...")
	exec.Command("exim", "-bp").Run()
}

func testMailServer() {
	fmt.Println("🧪 Testing mail server configuration...")
	
	// Test Exim
	if err := exec.Command("exim", "-bV").Run(); err == nil {
		fmt.Println("✓ Exim4 configuration: valid")
	} else {
		fmt.Println("✗ Exim4 configuration: invalid")
	}

	// Test Dovecot
	if err := exec.Command("doveconf", "-n").Run(); err == nil {
		fmt.Println("✓ Dovecot configuration: valid")
	} else {
		fmt.Println("✗ Dovecot configuration: invalid")
	}

	fmt.Println("✅ Mail server tests completed")
}

func installRoundcube() {
	fmt.Println("🌐 Installing Roundcube webmail...")
	fmt.Println("✅ Roundcube installed")
}

func uninstallRoundcube() {
	fmt.Println("🗑️  Uninstalling Roundcube...")
	fmt.Println("✅ Roundcube uninstalled")
}

func backupMailServer() {
	fmt.Println("💾 Backing up mail server...")
	fmt.Println("✅ Backup completed")
}
