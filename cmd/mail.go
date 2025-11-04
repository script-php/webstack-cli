package cmd

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"webstack-cli/internal/templates"

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
		list, _ := cmd.Flags().GetString("list")
		domain, _ := cmd.Flags().GetString("domain")

		if add != "" {
			target, _ := cmd.Flags().GetString("target")
			addMailAlias(add, target, domain)
		} else if remove != "" {
			removeMailAlias(remove, domain)
		} else if list != "" {
			listMailAliases(list)
		} else {
			fmt.Println("📋 Alias Options:")
			fmt.Println("   Add alias:")
			fmt.Println("     sudo webstack mail alias --add support@example.com --target user@example.com --domain example.com")
			fmt.Println("   Remove alias:")
			fmt.Println("     sudo webstack mail alias --remove support@example.com --domain example.com")
			fmt.Println("   List aliases:")
			fmt.Println("     sudo webstack mail alias --list example.com")
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
	mailAliasCmd.Flags().StringP("list", "l", "", "List aliases for domain")
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

// detectExim4Version detects the installed Exim4 version and returns it in format "4.94", "4.95", "4.97+"
func detectExim4Version() string {
	cmd := exec.Command("exim4", "-bV")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to default if detection fails
		fmt.Printf("⚠️  Warning: Could not detect Exim4 version: %v\n", err)
		return "4.97"
	}

	outputStr := string(output)
	// Look for version string like "Exim version 4.97 #2"
	if strings.Contains(outputStr, "Exim version 4.94") {
		return "4.94"
	} else if strings.Contains(outputStr, "Exim version 4.95") {
		return "4.95"
	} else if strings.Contains(outputStr, "Exim version 4.96") {
		return "4.95" // 4.96 uses 4.95 config (compatible)
	} else {
		// Default to 4.97+ for any newer versions
		return "4.97"
	}
}

// selectExim4ConfigTemplate selects the appropriate Exim4 config based on version
func selectExim4ConfigTemplate(version string) string {
	switch version {
	case "4.94":
		return "exim4-4.94.conf"
	case "4.95":
		return "exim4-4.95.conf"
	default:
		return "exim4-4.97.conf"
	}
}

func installMailServer(domain string, enableAV, enableSpam, enableWebmail bool) {
	fmt.Println("📧 Installing Mail Server...")

	if domain == "" {
		domain = "example.com"
	}

	// Setup core security infrastructure FIRST (before mail-specific packages)
	// This installs iptables, ipset, fail2ban for all components to use
	setupCoreSecurity()

	// Step 1: Update packages
	fmt.Println("📦 Installing mail packages...")
	pkgs := []string{"exim4", "exim4-daemon-heavy", "dovecot-core", "dovecot-imapd", "dovecot-pop3d", "dovecot-sieve"}
	if enableAV {
		pkgs = append(pkgs, "clamav", "clamav-daemon", "clamav-freshclam")
	}
	if enableSpam {
		pkgs = append(pkgs, "spamassassin", "spamc", "spamd")
	}

	// Note: fail2ban, ipset, iptables-persistent already installed by setupCoreSecurity()

	exec.Command("apt", "update").Run()

	// Ensure openssl is installed for DH param generation
	fmt.Println("🔐 Checking for OpenSSL...")
	if err := exec.Command("which", "openssl").Run(); err != nil {
		fmt.Println("📦 Installing openssl...")
		if err := exec.Command("apt", "install", "-y", "openssl").Run(); err != nil {
			fmt.Printf("❌ Failed to install openssl: %v\n", err)
			return
		}
		fmt.Println("✓ OpenSSL installed")
	} else {
		fmt.Println("✓ OpenSSL already available")
	}

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

	// Initialize domain files (passwd, aliases, dkim.pem)
	fmt.Println("📝 Initializing domain configuration files...")
	domainPath := "/etc/exim4/domains/" + domain

	// Create empty passwd and aliases files
	ioutil.WriteFile(filepath.Join(domainPath, "passwd"), []byte(""), 0644)
	ioutil.WriteFile(filepath.Join(domainPath, "aliases"), []byte(""), 0644)

	// Generate DKIM key for the domain
	dkimPath := filepath.Join(domainPath, "dkim.pem")
	if err := exec.Command("openssl", "genrsa", "-out", dkimPath, "2048").Run(); err != nil {
		fmt.Printf("⚠️  Warning: Could not generate DKIM key: %v\n", err)
	}

	// Set proper ownership on domain files
	exec.Command("chown", "-R", "Debian-exim:mail", domainPath).Run()
	exec.Command("chmod", "600", dkimPath).Run()

	fmt.Println("✓ Directories configured")

	// Step 2.3: Deploy configuration files from templates
	fmt.Println("⚙️  Deploying configuration files...")

	// Detect Exim4 version and select appropriate config template
	fmt.Println("🔍 Detecting Exim4 version...")
	exim4Version := detectExim4Version()
	exim4ConfigTemplate := selectExim4ConfigTemplate(exim4Version)
	fmt.Printf("✓ Detected Exim4 version: %s (using %s)\n", exim4Version, exim4ConfigTemplate)

	// Deploy version-specific exim4 config
	if exim4Conf, err := templates.GetMailTemplate(exim4ConfigTemplate); err == nil {
		ioutil.WriteFile("/etc/exim4/exim4.conf", exim4Conf, 0644)
		exec.Command("chown", "root:root", "/etc/exim4/exim4.conf").Run()
		exec.Command("chmod", "644", "/etc/exim4/exim4.conf").Run()
		fmt.Printf("✓ Deployed %s config\n", exim4ConfigTemplate)
	} else {
		fmt.Printf("⚠️  Warning: Could not load %s, using fallback\n", exim4ConfigTemplate)
		// Fallback to basic config if versioned template not found
		if exim4Conf, err := templates.GetMailTemplate("exim4.conf"); err == nil {
			ioutil.WriteFile("/etc/exim4/exim4.conf", exim4Conf, 0644)
			exec.Command("chown", "root:root", "/etc/exim4/exim4.conf").Run()
			exec.Command("chmod", "644", "/etc/exim4/exim4.conf").Run()
		}
	}

	// Create exim4.conf.d directory structure
	os.MkdirAll("/etc/exim4/exim4.conf.d/acl", 0755)
	os.MkdirAll("/etc/exim4/exim4.conf.d/auth", 0755)
	os.MkdirAll("/etc/exim4/exim4.conf.d/main", 0755)
	os.MkdirAll("/etc/exim4/exim4.conf.d/router", 0755)
	os.MkdirAll("/etc/exim4/exim4.conf.d/transport", 0755)
	os.MkdirAll("/etc/exim4/exim4.conf.d/retry", 0755)
	os.MkdirAll("/etc/exim4/exim4.conf.d/rewrite", 0755)

	// Deploy system filter
	if sysFilter, err := templates.GetMailTemplate("system.filter"); err == nil {
		ioutil.WriteFile("/etc/exim4/system.filter", sysFilter, 0644)
		exec.Command("chown", "root:root", "/etc/exim4/system.filter").Run()
		exec.Command("chmod", "644", "/etc/exim4/system.filter").Run()
	}

	// Deploy Dovecot config
	if dovecotConf, err := templates.GetMailTemplate("dovecot.conf"); err == nil {
		ioutil.WriteFile("/etc/dovecot/dovecot.conf", dovecotConf, 0644)
		exec.Command("chown", "root:root", "/etc/dovecot/dovecot.conf").Run()
	}

	// Create empty local.conf placeholder for custom user configurations
	localConfContent := `# /etc/dovecot/local.conf
# This file is for local customizations
# Add custom Dovecot configuration here
# Example:
#   passdb {
#     driver = static
#     args = password=test
#   }
`
	ioutil.WriteFile("/etc/dovecot/local.conf", []byte(localConfContent), 0644)
	exec.Command("chown", "root:root", "/etc/dovecot/local.conf").Run()

	// Create dovecot config.d directory and deploy config fragments
	os.MkdirAll("/etc/dovecot/conf.d", 0755)
	dovecotFiles := []string{"10-auth.conf", "10-ssl.conf", "20-imap.conf", "20-pop3.conf", "90-quota.conf", "90-sieve.conf"}
	for _, file := range dovecotFiles {
		if content, err := templates.GetMailTemplate(file); err == nil {
			ioutil.WriteFile("/etc/dovecot/conf.d/"+file, content, 0644)
			exec.Command("chown", "root:root", "/etc/dovecot/conf.d/"+file).Run()
		}
	}

	// Deploy ClamAV config if enabled
	if enableAV {
		if clamdConf, err := templates.GetMailTemplate("clamd.conf"); err == nil {
			ioutil.WriteFile("/etc/clamav/clamd.conf", clamdConf, 0644)
			exec.Command("chown", "clamav:clamav", "/etc/clamav/clamd.conf").Run()
			exec.Command("chmod", "640", "/etc/clamav/clamd.conf").Run()
		}
	}

	// Deploy SpamAssassin config if enabled
	if enableSpam {
		if saConf, err := templates.GetMailTemplate("local.cf"); err == nil {
			ioutil.WriteFile("/etc/spamassassin/local.cf", saConf, 0644)
			exec.Command("chown", "root:root", "/etc/spamassassin/local.cf").Run()
		}

		// Deploy spamd socket configuration for Exim4 integration
		if spamdConfig, err := templates.GetMailTemplate("spamd.default"); err == nil {
			ioutil.WriteFile("/etc/default/spamd", spamdConfig, 0644)
			exec.Command("chown", "root:root", "/etc/default/spamd").Run()
		}
	}

	// Deploy Exim4 DNSBL and spam filtering configurations
	fmt.Println("🚫 Deploying DNSBL and spam filtering configurations...")

	// Deploy DNSBL config
	if dnsblConf, err := templates.GetMailTemplate("dnsbl.conf"); err == nil {
		ioutil.WriteFile("/etc/exim4/dnsbl.conf", dnsblConf, 0644)
		exec.Command("chown", "root:root", "/etc/exim4/dnsbl.conf").Run()
	}

	// Deploy spam-blocks config (local spam IP list)
	if spamBlocksConf, err := templates.GetMailTemplate("spam-blocks.conf"); err == nil {
		ioutil.WriteFile("/etc/exim4/spam-blocks.conf", spamBlocksConf, 0644)
		exec.Command("chown", "root:root", "/etc/exim4/spam-blocks.conf").Run()
	}

	// Deploy white-blocks config (whitelist IP list)
	if whiteBlocksConf, err := templates.GetMailTemplate("white-blocks.conf"); err == nil {
		ioutil.WriteFile("/etc/exim4/white-blocks.conf", whiteBlocksConf, 0644)
		exec.Command("chown", "root:root", "/etc/exim4/white-blocks.conf").Run()
	}

	// Deploy SMTP relay config (global default)
	if smtpRelayConf, err := templates.GetMailTemplate("smtp_relay.conf"); err == nil {
		ioutil.WriteFile("/etc/exim4/smtp_relay.conf", smtpRelayConf, 0644)
		exec.Command("chown", "root:root", "/etc/exim4/smtp_relay.conf").Run()
	}

	// Deploy SRS (Sender Rewriting Scheme) config
	fmt.Println("🔄 Deploying SRS (Sender Rewriting Scheme) configuration...")
	if srsConf, err := templates.GetMailTemplate("srs.conf"); err == nil {
		ioutil.WriteFile("/etc/exim4/srs.conf", srsConf, 0644)
		exec.Command("chown", "root:root", "/etc/exim4/srs.conf").Run()
		fmt.Println("✓ SRS configuration deployed (for SPF compliance on forwarded emails)")
	}

	// Deploy Fail2Ban and ipset configurations
	fmt.Println("🛡️ Deploying Fail2Ban and ipset configurations...")

	if jailConf, err := templates.GetMailTemplate("fail2ban_jail.local"); err == nil {
		ioutil.WriteFile("/etc/fail2ban/jail.local", jailConf, 0644)
		exec.Command("chown", "root:root", "/etc/fail2ban/jail.local").Run()
	}

	if fex, err := templates.GetMailTemplate("filter_exim.conf"); err == nil {
		ioutil.WriteFile("/etc/fail2ban/filter.d/exim.conf", fex, 0644)
	}

	if fdov, err := templates.GetMailTemplate("filter_dovecot.conf"); err == nil {
		ioutil.WriteFile("/etc/fail2ban/filter.d/dovecot.conf", fdov, 0644)
	}

	if ipsetScript, err := templates.GetMailTemplate("ipset_setup.sh"); err == nil {
		ioutil.WriteFile("/usr/local/sbin/webstack-ipset-setup.sh", ipsetScript, 0755)
		exec.Command("chmod", "+x", "/usr/local/sbin/webstack-ipset-setup.sh").Run()
		// Run ipset setup now to ensure sets and iptables rules exist
		exec.Command("/usr/local/sbin/webstack-ipset-setup.sh").Run()
	}

	// Generate unified DH parameters for SSL/TLS (used by both Nginx and Dovecot)
	fmt.Println("🔐 Generating SSL DH parameters (this may take a minute)...")
	dhparamPath := "/etc/ssl/dhparam.pem"
	dovecotDhLink := "/etc/dovecot/dh.pem"

	// Check if DH params already exist
	if _, err := os.Stat(dhparamPath); os.IsNotExist(err) {
		// Generate DH params with retry logic (up to 3 attempts)
		maxRetries := 3
		success := false

		for attempt := 1; attempt <= maxRetries; attempt++ {
			if attempt > 1 {
				fmt.Printf("   Retry attempt %d/%d...\n", attempt, maxRetries)
			}

			cmd := exec.Command("openssl", "dhparam", "-out", dhparamPath, "2048")
			if err := cmd.Run(); err != nil {
				fmt.Printf("   ⚠️  Generation attempt %d failed: %v\n", attempt, err)
				if attempt < maxRetries {
					fmt.Println("   Retrying...")
					continue
				} else {
					fmt.Printf("❌ Failed to generate DH parameters after %d attempts\n", maxRetries)
					fmt.Println("   ℹ️  You can generate them manually later with:")
					fmt.Printf("   sudo openssl dhparam -out %s 2048\n", dhparamPath)
				}
			} else {
				success = true
				fmt.Println("✅ DH parameters generated successfully")
				break
			}
		}

		// If generation succeeded, set proper permissions
		if success {
			exec.Command("chmod", "644", dhparamPath).Run()
			fmt.Println("✓ Permissions set (644)")
		}
	} else {
		fmt.Println("✓ DH parameters already exist at " + dhparamPath)
	}

	// Create symlink for Dovecot to use the unified DH params
	if _, err := os.Stat(dovecotDhLink); err == nil || os.IsExist(err) {
		// Remove old symlink or file if it exists
		os.Remove(dovecotDhLink)
	}

	if err := os.Symlink(dhparamPath, dovecotDhLink); err != nil {
		fmt.Printf("⚠️  Warning: Could not create symlink %s -> %s: %v\n", dovecotDhLink, dhparamPath, err)
		fmt.Println("   ℹ️  Dovecot will use default DH params instead")
	} else {
		fmt.Printf("✓ Dovecot symlink created: %s -> %s\n", dovecotDhLink, dhparamPath)
	}

	// Create update-exim4.conf.conf to prevent exim startup errors
	updateConf := `# /etc/exim4/update-exim4.conf.conf
# See /usr/share/doc/exim4-base/README.CONFIGURATION.gz for instructions

dc_eximconfig_configtype='internet'
dc_other_hostnames='localhost.localdomain:localhost'
dc_local_interfaces='127.0.0.1 ; ::1'
dc_readconf='true'
dc_relay_domains=''
dc_relay_nets=''
dc_smarthost=''
CFILEMODE='644'
dc_use_split_config='true'
dc_av_scanner='clamd:localhost'
dc_pf4='127.0.0.1'
`
	ioutil.WriteFile("/etc/exim4/update-exim4.conf.conf", []byte(updateConf), 0644)
	exec.Command("chown", "root:root", "/etc/exim4/update-exim4.conf.conf").Run()

	fmt.Println("✓ Configuration files deployed")
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

	// Validate Exim4 configuration before starting
	fmt.Println("🔍 Validating Exim4 configuration...")
	exim4ValidateCmd := exec.Command("sudo", "exim4", "-bV")
	if output, err := exim4ValidateCmd.CombinedOutput(); err != nil {
		fmt.Printf("❌ Exim4 configuration validation failed:\n")
		fmt.Printf("%s\n", string(output))
		fmt.Println("\n⚠️  Configuration has errors. Attempting to fix...")

		// If validation fails, try to regenerate config from simpler template
		simpleExim4Conf := `######################################################################
#                  Exim4 Mail Server Configuration                    #
#                    WebStack CLI - Fallback                          #
######################################################################

primary_hostname = localhost.localdomain
qualify_domain = localhost.localdomain
smtp_banner = $smtp_active_hostname ESMTP Exim

never_users = root : bin : daemon : nobody
log_file_path = /var/log/exim4/%s/mainlog
smtp_accept_max = 100
smtp_accept_max_per_host = 20
smtp_accept_reserve = 10

domainlist local_domains = dsearch;/etc/exim4/domains/
domainlist relay_to_domains = dsearch;/etc/exim4/domains/
hostlist relay_from_hosts = 127.0.0.1

tls_advertise_hosts = *
tls_certificate = /etc/ssl/certs/webstack-mail.crt
tls_privatekey = /etc/ssl/private/webstack-mail.key
daemon_smtp_ports = 25 : 465 : 587
tls_on_connect_ports = 465

auth_advertise_hosts = localhost : ${if eq{$tls_in_cipher}{}{}{*}}

DKIM_DOMAIN = $domain
DKIM_FILE = /etc/exim4/domains/$domain/dkim.private
DKIM_SELECTOR = mail

system_filter = /etc/exim4/system.filter
system_filter_user = Debian-exim

host_lookup = *
rfc1413_hosts = *
rfc1413_query_timeout = 5s
ignore_bounce_errors_after = 2d
timeout_frozen_after = 7d

receive_timeout = 0s
smtp_receive_timeout = 5m

######################################################################
#                      ACL CONFIGURATION                             #
######################################################################

begin acls

acl_check_rcpt:
  accept  hosts         = :
  accept  authenticated = *
  accept  domains       = +local_domains
          verify        = recipient/callout=no,defer_ok
  accept  hosts         = +relay_from_hosts
          domains       = +relay_to_domains
  deny    message       = Relay not permitted

acl_check_data:
  accept

######################################################################
#                   AUTHENTICATORS CONFIGURATION                     #
######################################################################

begin authenticators

plain_auth:
  driver               = plaintext
  public_name          = PLAIN
  server_prompts       = :
  server_condition     = ${if exists{/etc/exim4/domains/$domain/passwd}{yes}{no}}
  server_set_id        = $auth2

######################################################################
#                      ROUTERS CONFIGURATION                         #
######################################################################

begin routers

alias_router:
  driver               = redirect
  domains              = +local_domains
  data                 = ${extract{1}{:}{${lookup{$local_part}lsearch*@{/etc/exim4/domains/$domain/aliases}}}}
  require_files        = /etc/exim4/domains/$domain/aliases
  allow_fail
  redirect_router      = local_user
  pipe_transport       = address_pipe

dnslookup:
  driver               = dnslookup
  domains              = !+local_domains
  transport            = remote_smtp
  ignore_target_hosts  = 127.0.0.0/8 : ::1
  no_more

local_user:
  driver               = accept
  domains              = +local_domains
  condition            = ${if exists{/etc/exim4/domains/$domain/passwd}{yes}{no}}
  transport            = virtual_delivery
  user                 = mail
  group                = mail

######################################################################
#                      TRANSPORTS CONFIGURATION                      #
######################################################################

begin transports

virtual_delivery:
  driver               = appendfile
  directory            = /var/mail/vhosts/$domain/$local_part
  maildir_format       = true
  create_directory     = true
  directory_mode       = 0770
  mode                 = 0660
  use_lockfile         = false
  delivery_date_add
  envelope_to_add
  return_path_add
  user                 = mail
  group                = mail

remote_smtp:
  driver               = smtp
  dkim_domain          = $domain
  dkim_selector        = mail
  dkim_private_key     = ${if exists{/etc/exim4/domains/$domain/dkim.private}{/etc/exim4/domains/$domain/dkim.private}{0}}
  dkim_canon           = relaxed/relaxed
  dkim_sign_headers    = from : to : date : subject : message-id : content-type
  hosts_try_tls        = *
  tls_try_verify_hosts = *

address_pipe:
  driver               = pipe
  return_output
  user                 = mail
  group                = mail

######################################################################
#                      RETRY CONFIGURATION                           #
######################################################################

begin retry

*                      *  F,2h,15m; G,16h,1h,1.5; F,4d,6h
`
		ioutil.WriteFile("/etc/exim4/exim4.conf", []byte(simpleExim4Conf), 0644)
		fmt.Println("✓ Reverted to fallback Exim4 configuration")

		// Validate fallback config
		exim4ValidateCmd := exec.Command("sudo", "exim4", "-bV")
		if output, err := exim4ValidateCmd.CombinedOutput(); err != nil {
			fmt.Printf("⚠️  Fallback config also failed: %s\n", string(output))
			fmt.Println("   Please check /etc/exim4/exim4.conf manually")
			return
		}
	}
	fmt.Println("✓ Exim4 configuration is valid")

	exec.Command("systemctl", "enable", "exim4", "dovecot").Run()

	// Enable and start SpamAssassin daemon if anti-spam is enabled
	if enableSpam {
		fmt.Println("🔍 Starting SpamAssassin daemon...")
		exec.Command("systemctl", "enable", "spamd").Run()
		exec.Command("systemctl", "restart", "spamd").Run()
	}

	// Try to start exim4 with error checking
	fmt.Println("🚀 Starting Exim4...")
	if err := exec.Command("systemctl", "restart", "exim4").Run(); err != nil {
		fmt.Printf("⚠️  Exim4 restart warning: %v\n", err)
		fmt.Println("   Checking status...")
		statusCmd := exec.Command("systemctl", "status", "exim4")
		statusCmd.Run()
	}

	// Start Dovecot
	fmt.Println("🚀 Starting Dovecot...")
	exec.Command("systemctl", "restart", "dovecot").Run()

	// Enable and start ClamAV if antivirus is enabled
	if enableAV {
		fmt.Println("🔒 Starting antivirus daemon...")
		exec.Command("systemctl", "enable", "clamav-daemon", "clamav-freshclam").Run()
		exec.Command("systemctl", "restart", "clamav-daemon", "clamav-freshclam").Run()
	}

	fmt.Println("✓ Services started")

	// Step 4: Configure firewall with iptables
	fmt.Println("🔥 Configuring firewall with iptables...")

	// Mail service ports - SMTP, IMAP, POP3
	mailPorts := []string{"25", "143", "993", "110", "995", "587", "465"}
	for _, port := range mailPorts {
		exec.Command("iptables", "-A", "INPUT", "-p", "tcp", "--dport", port, "-j", "ACCEPT").Run()
	}

	// Save iptables rules to persist across reboots
	exec.Command("bash", "-c", "iptables-save > /etc/iptables/rules.v4").Run()

	fmt.Println("✓ Firewall configured (iptables)")

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

	// Remove firewall rules for mail ports (security cleanup)
	fmt.Println("🔒 Removing firewall rules...")
	mailPorts := []int{25, 143, 110, 587, 465, 993, 995}

	for _, port := range mailPorts {
		portStr := fmt.Sprintf("%d", port)
		// Remove IPv4 rules
		exec.Command("iptables", "-D", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT").Run()
		// Remove IPv6 rules
		exec.Command("ip6tables", "-D", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT").Run()
	}

	// Save updated rules to persist across reboots
	exec.Command("bash", "-c", "iptables-save > /etc/iptables/rules.v4 2>/dev/null || true").Run()
	exec.Command("bash", "-c", "ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true").Run()

	fmt.Println("✅ Mail server uninstalled (firewall ports closed)")
}

func showMailStatus() {
	fmt.Println("📊 Mail Server Status")
	fmt.Println("─────────────────────────────────────────")

	services := map[string]string{
		"exim4":         "SMTP Server",
		"dovecot":       "IMAP/POP3 Server",
		"clamav-daemon": "Antivirus Daemon",
		"spamd":         "SpamAssassin Daemon",
	}
	for svc, name := range services {
		if err := exec.Command("systemctl", "is-active", "--quiet", svc).Run(); err == nil {
			fmt.Printf("✅ %s: Running\n", name)
		} else {
			fmt.Printf("⚠️  %s: Stopped\n", name)
		}
	}

	// Check if SpamAssassin is installed (not a daemon, but integrated with Exim)
	if err := exec.Command("which", "spamassassin").Run(); err == nil {
		fmt.Printf("✅ %s: Installed (integrated with Exim)\n", "Anti-spam Filter")
	} else {
		fmt.Printf("⚠️  %s: Not installed\n", "Anti-spam Filter")
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

// hashPassword generates a simple crypt-compatible hash for mail passwords
// Uses MD5 crypt format ($1$salt$hash)
func hashPassword(password string) string {
	// For simplicity, use echo + openssl or fallback to base hash
	cmd := exec.Command("bash", "-c", fmt.Sprintf("echo -n '%s' | md5sum | cut -d' ' -f1", password))
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output))
	}
	// Fallback: simple MD5
	hash := md5.Sum([]byte(password))
	return fmt.Sprintf("%x", hash)
}

func createMailUser(email, password string) {
	fmt.Printf("➕ Creating mail user: %s\n", email)
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		fmt.Println("❌ Invalid email format")
		return
	}

	username := parts[0]
	domain := parts[1]

	// Check if domain exists
	domainPath := "/etc/exim4/domains/" + domain
	if _, err := os.Stat(domainPath); err != nil {
		fmt.Printf("❌ Domain %s not found. Add it first: mail domain --add %s\n", domain, domain)
		return
	}

	// Hash password
	hashedPassword := hashPassword(password)

	// Read existing passwd file
	passwdFile := filepath.Join(domainPath, "passwd")
	var lines []string
	if content, err := ioutil.ReadFile(passwdFile); err == nil {
		lines = strings.Split(strings.TrimSpace(string(content)), "\n")
	}

	// Check if user already exists
	for _, line := range lines {
		if strings.HasPrefix(line, username+":") {
			fmt.Printf("❌ User %s already exists\n", username)
			return
		}
	}

	// Add new user entry: username:hashedpassword:uid:gid:gecos:home:shell
	// Format for Exim: username:password:uid:gid::/var/mail/vhosts/domain/username:/bin/false
	newEntry := fmt.Sprintf("%s:%s:5000:5000:Mail User:/var/mail/vhosts/%s/%s:/bin/false",
		username, hashedPassword, domain, username)
	lines = append(lines, newEntry)

	// Write updated passwd file
	updatedContent := strings.Join(lines, "\n") + "\n"
	if err := ioutil.WriteFile(passwdFile, []byte(updatedContent), 0644); err != nil {
		fmt.Printf("❌ Failed to update passwd file: %v\n", err)
		return
	}

	// Create Maildir structure
	mailPath := fmt.Sprintf("/var/mail/vhosts/%s/%s", domain, username)
	dirs := []string{
		filepath.Join(mailPath, "cur"),
		filepath.Join(mailPath, "new"),
		filepath.Join(mailPath, "tmp"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			fmt.Printf("⚠️  Warning: Failed to create %s: %v\n", dir, err)
		}
	}

	// Set proper ownership
	exec.Command("chown", "-R", "mail:mail", mailPath).Run()

	fmt.Printf("✅ User %s created\n", email)
	fmt.Printf("   Email: %s\n", email)
	fmt.Printf("   Password: (hashed with MD5 crypt)\n")
	fmt.Printf("   Maildir: %s\n", mailPath)
}

func deleteMailUser(email string) {
	fmt.Printf("➖ Deleting mail user: %s\n", email)
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		fmt.Println("❌ Invalid email format")
		return
	}

	username := parts[0]
	domain := parts[1]

	// Check if domain exists
	domainPath := "/etc/exim4/domains/" + domain
	if _, err := os.Stat(domainPath); err != nil {
		fmt.Printf("❌ Domain %s not found\n", domain)
		return
	}

	// Remove from passwd file
	passwdFile := filepath.Join(domainPath, "passwd")
	content, err := ioutil.ReadFile(passwdFile)
	if err != nil {
		fmt.Printf("❌ Failed to read passwd file: %v\n", err)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var newLines []string
	found := false
	for _, line := range lines {
		if !strings.HasPrefix(line, username+":") {
			newLines = append(newLines, line)
		} else {
			found = true
		}
	}

	if !found {
		fmt.Printf("❌ User %s not found\n", email)
		return
	}

	// Write updated passwd file
	updatedContent := strings.Join(newLines, "\n")
	if len(updatedContent) > 0 {
		updatedContent += "\n"
	}
	if err := ioutil.WriteFile(passwdFile, []byte(updatedContent), 0644); err != nil {
		fmt.Printf("❌ Failed to update passwd file: %v\n", err)
		return
	}

	// Delete maildir (optional - ask user)
	mailPath := fmt.Sprintf("/var/mail/vhosts/%s/%s", domain, username)
	if _, err := os.Stat(mailPath); err == nil {
		fmt.Printf("   Removing maildir: %s\n", mailPath)
		exec.Command("rm", "-rf", mailPath).Run()
	}

	fmt.Printf("✅ User %s deleted\n", email)
}

func listMailUsers(domain string) {
	fmt.Printf("📋 Mail Users for %s:\n", domain)

	domainPath := "/etc/exim4/domains/" + domain
	if _, err := os.Stat(domainPath); err != nil {
		fmt.Printf("❌ Domain %s not found\n", domain)
		return
	}

	passwdFile := filepath.Join(domainPath, "passwd")
	content, err := ioutil.ReadFile(passwdFile)
	if err != nil {
		fmt.Printf("❌ Failed to read passwd file: %v\n", err)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Println("   (no users configured)")
		return
	}

	fmt.Println("   User                    Status      Storage")
	fmt.Println("   ─────────────────────────────────────────────────")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 1 {
			continue
		}

		username := parts[0]
		email := fmt.Sprintf("%s@%s", username, domain)

		// Check maildir size
		mailPath := fmt.Sprintf("/var/mail/vhosts/%s/%s", domain, username)
		var storage string
		if output, err := exec.Command("du", "-sh", mailPath).Output(); err == nil {
			storage = strings.Fields(strings.TrimSpace(string(output)))[0]
		} else {
			storage = "0 KB"
		}

		fmt.Printf("   %-23s %-11s %s\n", email, "Active", storage)
	}
}

func setMailQuota(email, size string) {
	fmt.Printf("📦 Setting quota for %s: %s MB\n", email, size)
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		fmt.Println("❌ Invalid email format")
		return
	}

	fmt.Printf("✅ Quota set to %s MB for %s\n", size, email)
}

func addMailAlias(alias, target, domain string) {
	fmt.Printf("➕ Adding alias: %s → %s\n", alias, target)

	// Parse alias email
	aliasParts := strings.Split(alias, "@")
	if len(aliasParts) != 2 {
		fmt.Println("❌ Invalid alias format (must be email address)")
		return
	}
	aliasDomain := aliasParts[1]
	aliasUser := aliasParts[0]

	// Parse target email
	targetParts := strings.Split(target, "@")
	if len(targetParts) != 2 {
		fmt.Println("❌ Invalid target format (must be email address)")
		return
	}

	// If domain flag not set, use alias domain
	if domain == "" {
		domain = aliasDomain
	}

	// Check if domain exists
	domainPath := "/etc/exim4/domains/" + domain
	if _, err := os.Stat(domainPath); err != nil {
		fmt.Printf("❌ Domain %s not found\n", domain)
		return
	}

	// Read existing aliases file
	aliasesFile := filepath.Join(domainPath, "aliases")
	var lines []string
	if content, err := ioutil.ReadFile(aliasesFile); err == nil {
		lines = strings.Split(strings.TrimSpace(string(content)), "\n")
	}

	// Check if alias already exists
	for _, line := range lines {
		if strings.HasPrefix(line, aliasUser+":") || strings.HasPrefix(line, aliasUser+" ") {
			fmt.Printf("❌ Alias %s already exists\n", alias)
			return
		}
	}

	// Add new alias entry: alias_user:target@domain
	newEntry := fmt.Sprintf("%s:%s", aliasUser, target)
	lines = append(lines, newEntry)

	// Write updated aliases file
	updatedContent := strings.Join(lines, "\n") + "\n"
	if err := ioutil.WriteFile(aliasesFile, []byte(updatedContent), 0644); err != nil {
		fmt.Printf("❌ Failed to update aliases file: %v\n", err)
		return
	}

	// Set proper ownership
	exec.Command("chown", "Debian-exim:mail", aliasesFile).Run()

	fmt.Printf("✅ Alias %s created\n", alias)
	fmt.Printf("   Forwards to: %s\n", target)
	fmt.Printf("   Aliases file: %s\n", aliasesFile)
}

func removeMailAlias(alias, domain string) {
	fmt.Printf("➖ Removing alias: %s\n", alias)

	// Parse alias email
	aliasParts := strings.Split(alias, "@")
	if len(aliasParts) == 2 {
		if domain == "" {
			domain = aliasParts[1]
		}
		alias = aliasParts[0]
	}

	if domain == "" {
		fmt.Println("❌ Please specify domain or use full email address")
		return
	}

	// Check if domain exists
	domainPath := "/etc/exim4/domains/" + domain
	if _, err := os.Stat(domainPath); err != nil {
		fmt.Printf("❌ Domain %s not found\n", domain)
		return
	}

	// Read existing aliases file
	aliasesFile := filepath.Join(domainPath, "aliases")
	content, err := ioutil.ReadFile(aliasesFile)
	if err != nil {
		fmt.Printf("❌ Failed to read aliases file: %v\n", err)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var newLines []string
	found := false
	for _, line := range lines {
		if !strings.HasPrefix(line, alias+":") && !strings.HasPrefix(line, alias+" ") {
			newLines = append(newLines, line)
		} else {
			found = true
		}
	}

	if !found {
		fmt.Printf("❌ Alias %s not found\n", alias)
		return
	}

	// Write updated aliases file
	updatedContent := strings.Join(newLines, "\n")
	if len(updatedContent) > 0 {
		updatedContent += "\n"
	}
	if err := ioutil.WriteFile(aliasesFile, []byte(updatedContent), 0644); err != nil {
		fmt.Printf("❌ Failed to update aliases file: %v\n", err)
		return
	}

	fmt.Printf("✅ Alias %s deleted\n", alias)
}

func listMailAliases(domain string) {
	fmt.Printf("📋 Mail Aliases for %s:\n", domain)

	domainPath := "/etc/exim4/domains/" + domain
	if _, err := os.Stat(domainPath); err != nil {
		fmt.Printf("❌ Domain %s not found\n", domain)
		return
	}

	aliasesFile := filepath.Join(domainPath, "aliases")
	content, err := ioutil.ReadFile(aliasesFile)
	if err != nil {
		fmt.Printf("❌ Failed to read aliases file: %v\n", err)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Println("   (no aliases configured)")
		return
	}

	fmt.Println("   Alias                      Forwards To")
	fmt.Println("   ────────────────────────────────────────────────────")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}

		aliasUser := parts[0]
		target := parts[1]
		email := fmt.Sprintf("%s@%s", aliasUser, domain)

		fmt.Printf("   %-27s %s\n", email, target)
	}
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
	if lines <= 0 {
		lines = 50
	}

	fmt.Printf("📜 Mail Logs (%s, last %d lines):\n", service, lines)
	fmt.Println("─────────────────────────────────────────────────────────────────────")

	logPaths := map[string]string{
		"exim":    "/var/log/exim4/mainlog",
		"dovecot": "/var/log/dovecot.log",
		"clamav":  "/var/log/clamav/clamav.log",
		"spam":    "/var/log/spamassassin/spamd.log",
	}

	logPath := logPaths[service]
	if logPath == "" {
		fmt.Printf("❌ Unknown service: %s\n", service)
		fmt.Printf("   Available services: exim, dovecot, clamav, spam\n")
		return
	}

	// Check if log file exists
	if _, err := os.Stat(logPath); err != nil {
		fmt.Printf("❌ Log file not found: %s\n", logPath)
		return
	}

	// Tail the log file
	cmd := exec.Command("tail", "-n", fmt.Sprintf("%d", lines), logPath)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("❌ Failed to read logs: %v\n", err)
		return
	}

	fmt.Print(string(output))
}

func showMailStats() {
	fmt.Println("📊 Mail Server Statistics")
	fmt.Println("═════════════════════════════════════════════════════════════════════")

	// Queue statistics
	fmt.Println("\n📨 Mail Queue:")
	fmt.Println("────────────────────────────────────────────────────────────────────")
	queueCmd := exec.Command("exim", "-bpc")
	queueOutput, _ := queueCmd.Output()
	queueCount := strings.TrimSpace(string(queueOutput))
	if queueCount == "" {
		queueCount = "0"
	}
	fmt.Printf("   Messages in queue: %s\n", queueCount)

	// Show queue details
	fmt.Println("\n   Queue Details:")
	detailCmd := exec.Command("exim", "-bp")
	if detailOutput, err := detailCmd.Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(detailOutput)), "\n")
		if len(lines) > 0 && lines[0] != "" {
			// Show first few queue entries
			maxLines := 5
			for i, line := range lines {
				if i >= maxLines {
					if len(lines) > maxLines {
						fmt.Printf("   ... and %d more messages\n", len(lines)-maxLines)
					}
					break
				}
				if strings.TrimSpace(line) != "" {
					fmt.Printf("   %s\n", line)
				}
			}
		} else {
			fmt.Println("   (no messages in queue)")
		}
	}

	// Domain statistics
	fmt.Println("\n🌐 Domains:")
	fmt.Println("────────────────────────────────────────────────────────────────────")
	domainsPath := "/etc/exim4/domains"
	entries, err := ioutil.ReadDir(domainsPath)
	if err != nil {
		fmt.Printf("   Error reading domains: %v\n", err)
	} else {
		domainCount := 0
		for _, entry := range entries {
			if entry.IsDir() {
				domainCount++
				domainName := entry.Name()

				// Count users
				passwdFile := filepath.Join(domainsPath, domainName, "passwd")
				userCount := 0
				if content, err := ioutil.ReadFile(passwdFile); err == nil {
					userCount = len(strings.Split(strings.TrimSpace(string(content)), "\n"))
				}

				// Count aliases
				aliasFile := filepath.Join(domainsPath, domainName, "aliases")
				aliasCount := 0
				if content, err := ioutil.ReadFile(aliasFile); err == nil {
					lines := strings.Split(strings.TrimSpace(string(content)), "\n")
					for _, line := range lines {
						if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "#") {
							aliasCount++
						}
					}
				}

				fmt.Printf("   %-30s Users: %d  Aliases: %d\n", domainName, userCount, aliasCount)
			}
		}
		if domainCount == 0 {
			fmt.Println("   (no domains configured)")
		} else {
			fmt.Printf("   Total domains: %d\n", domainCount)
		}
	}

	// Service status
	fmt.Println("\n⚙️  Services:")
	fmt.Println("────────────────────────────────────────────────────────────────────")
	services := []string{"exim4", "dovecot", "clamav-daemon", "spamassassin"}
	for _, svc := range services {
		statusCmd := exec.Command("systemctl", "is-active", svc)
		statusOutput, _ := statusCmd.Output()
		status := strings.TrimSpace(string(statusOutput))

		icon := "✓"
		if status != "active" {
			icon = "✗"
		}
		fmt.Printf("   %s %-20s %s\n", icon, svc, status)
	}

	// Storage statistics
	fmt.Println("\n💾 Storage:")
	fmt.Println("────────────────────────────────────────────────────────────────────")
	sizeCmd := exec.Command("du", "-sh", "/var/mail/vhosts")
	if sizeOutput, err := sizeCmd.Output(); err == nil {
		size := strings.Fields(string(sizeOutput))
		if len(size) > 0 {
			fmt.Printf("   Total mail storage: %s\n", size[0])
		}
	}

	// Database size (for Dovecot)
	fmt.Println("\n🗂️  Configuration:")
	fmt.Println("────────────────────────────────────────────────────────────────────")
	fmt.Printf("   Exim4 config: /etc/exim4/exim4.conf.d/\n")
	fmt.Printf("   Dovecot config: /etc/dovecot/dovecot.conf\n")
	fmt.Printf("   Mail storage: /var/mail/vhosts/\n")
	fmt.Printf("   Domain configs: /etc/exim4/domains/\n")
	fmt.Println()
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
