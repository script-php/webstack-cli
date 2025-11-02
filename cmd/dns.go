package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"webstack-cli/internal/templates"

	"github.com/spf13/cobra"
)

var dnsCmd = &cobra.Command{
	Use:   "dns",
	Short: "DNS Server (Bind9) management",
	Long:  `Install, configure, and manage Bind9 DNS server with clustering and replication support.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use 'webstack dns --help' for available commands")
	},
}

var dnsInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install and configure Bind9 DNS server",
	Long: `Install Bind9 DNS server with optional clustering configuration.
Usage:
  sudo webstack dns install
  sudo webstack dns install --mode master
  sudo webstack dns install --mode slave --master-ip 192.168.1.10
  sudo webstack dns install --mode slave --master-ip 192.168.1.10 --cluster-name datacenter-1`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}

		mode, _ := cmd.Flags().GetString("mode")
		masterIP, _ := cmd.Flags().GetString("master-ip")
		clusterName, _ := cmd.Flags().GetString("cluster-name")
		serverIP, _ := cmd.Flags().GetString("server-ip")

		installDNS(mode, masterIP, serverIP, clusterName)
	},
}

var dnsUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Bind9 DNS server",
	Long: `Remove Bind9 DNS server and all configurations.
Usage:
  sudo webstack dns uninstall`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}

		uninstallDNS()
	},
}

var dnsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check Bind9 DNS server status",
	Long: `Display DNS server installation and replication status.
Usage:
  sudo webstack dns status`,
	Run: func(cmd *cobra.Command, args []string) {
		showDNSStatus()
	},
}

var dnsConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure DNS zone transfers and clustering",
	Long: `Configure zone transfers, replication, and clustering settings.
Usage:
  sudo webstack dns config --add-slave 192.168.1.20
  sudo webstack dns config --remove-slave 192.168.1.20
  sudo webstack dns config --zone example.com --type master`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}

		addSlave, _ := cmd.Flags().GetString("add-slave")
		removeSlave, _ := cmd.Flags().GetString("remove-slave")
		zone, _ := cmd.Flags().GetString("zone")
		zoneType, _ := cmd.Flags().GetString("type")

		if addSlave != "" {
			configureDNSSlave(addSlave, true)
		} else if removeSlave != "" {
			configureDNSSlave(removeSlave, false)
		} else if zone != "" {
			if zoneType == "" {
				fmt.Println("❌ --type flag required when specifying --zone")
				fmt.Println("   Options: master or slave")
				return
			}
			configureZone(zone, zoneType)
		} else {
			fmt.Println("📋 DNS Configuration Options:")
			fmt.Println("   Add slave server:")
			fmt.Println("     sudo webstack dns config --add-slave <IP>")
			fmt.Println("   Remove slave server:")
			fmt.Println("     sudo webstack dns config --remove-slave <IP>")
			fmt.Println("   Configure zone:")
			fmt.Println("     sudo webstack dns config --zone example.com --type master")
		}
	},
}

var dnsRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart Bind9 DNS service",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}
		fmt.Println("🔄 Restarting Bind9 DNS service...")
		if err := exec.Command("systemctl", "restart", "bind9").Run(); err != nil {
			fmt.Printf("❌ Failed to restart Bind9: %v\n", err)
			return
		}
		fmt.Println("✅ Bind9 restarted successfully")
	},
}

var dnsReloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload Bind9 configuration",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}
		fmt.Println("🔄 Reloading Bind9 configuration...")
		if err := exec.Command("systemctl", "reload", "bind9").Run(); err != nil {
			fmt.Printf("❌ Failed to reload Bind9: %v\n", err)
			return
		}
		fmt.Println("✅ Bind9 configuration reloaded")
	},
}

var dnsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate Bind9 configuration",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔍 Checking Bind9 configuration...")
		if err := exec.Command("named-checkconf").Run(); err != nil {
			fmt.Println("❌ Configuration is invalid")
			return
		}
		fmt.Println("✅ Configuration is valid")
	},
}

var dnsZonesCmd = &cobra.Command{
	Use:   "zones",
	Short: "List all configured DNS zones",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("📋 Configured DNS Zones:")
		listDNSZones()
	},
}

var dnsLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View Bind9 logs",
	Run: func(cmd *cobra.Command, args []string) {
		lines, _ := cmd.Flags().GetInt("lines")
		viewDNSLogs(lines)
	},
}

var dnsQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Test DNS query",
	Long:  "Test DNS query: webstack dns query example.com",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("❌ Please specify a domain to query")
			fmt.Println("   Usage: webstack dns query example.com")
			return
		}
		testDNSQuery(args[0])
	},
}

var dnsBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup DNS configuration and zones",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}
		backupDNS()
	},
}

var dnsRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore DNS configuration from backup",
	Long:  "Restore DNS configuration: sudo webstack dns restore /path/to/backup.tar.gz",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}
		if len(args) == 0 {
			fmt.Println("❌ Please specify backup file path")
			return
		}
		restoreDNS(args[0])
	},
}

var dnsDNSSECCmd = &cobra.Command{
	Use:   "dnssec",
	Short: "Manage DNSSEC settings",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}
		enable, _ := cmd.Flags().GetBool("enable")
		disable, _ := cmd.Flags().GetBool("disable")
		
		if enable {
			manageDNSSEC(true)
		} else if disable {
			manageDNSSEC(false)
		} else {
			fmt.Println("📋 DNSSEC Options:")
			fmt.Println("   Enable DNSSEC validation:")
			fmt.Println("     sudo webstack dns dnssec --enable")
			fmt.Println("   Disable DNSSEC validation:")
			fmt.Println("     sudo webstack dns dnssec --disable")
		}
	},
}

var dnsStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Display DNS query statistics",
	Run: func(cmd *cobra.Command, args []string) {
		showDNSStats()
	},
}

var dnsQuerylogCmd = &cobra.Command{
	Use:   "querylog",
	Short: "Enable/disable query logging",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}
		enable, _ := cmd.Flags().GetBool("enable")
		disable, _ := cmd.Flags().GetBool("disable")
		
		if enable {
			manageQueryLog(true)
		} else if disable {
			manageQueryLog(false)
		} else {
			fmt.Println("📋 Query Log Options:")
			fmt.Println("   Enable query logging:")
			fmt.Println("     sudo webstack dns querylog --enable")
			fmt.Println("   Disable query logging:")
			fmt.Println("     sudo webstack dns querylog --disable")
		}
	},
}

func init() {
	dnsInstallCmd.Flags().StringP("mode", "m", "master", "DNS server mode: master or slave")
	dnsInstallCmd.Flags().StringP("master-ip", "M", "", "Master server IP (required if mode is slave)")
	dnsInstallCmd.Flags().StringP("server-ip", "s", "", "This server's IP address (auto-detect if not specified)")
	dnsInstallCmd.Flags().StringP("cluster-name", "c", "", "Cluster name for replication group")

	dnsConfigCmd.Flags().StringP("add-slave", "a", "", "Add slave server IP to replication")
	dnsConfigCmd.Flags().StringP("remove-slave", "r", "", "Remove slave server IP from replication")
	dnsConfigCmd.Flags().StringP("zone", "z", "", "Zone name to manage")
	dnsConfigCmd.Flags().StringP("type", "t", "", "Zone type: master or slave")

	dnsLogsCmd.Flags().IntP("lines", "n", 50, "Number of log lines to display")
	
	dnsDNSSECCmd.Flags().BoolP("enable", "e", false, "Enable DNSSEC validation")
	dnsDNSSECCmd.Flags().BoolP("disable", "d", false, "Disable DNSSEC validation")
	
	dnsQuerylogCmd.Flags().BoolP("enable", "e", false, "Enable query logging")
	dnsQuerylogCmd.Flags().BoolP("disable", "d", false, "Disable query logging")

	rootCmd.AddCommand(dnsCmd)
	dnsCmd.AddCommand(dnsInstallCmd)
	dnsCmd.AddCommand(dnsUninstallCmd)
	dnsCmd.AddCommand(dnsStatusCmd)
	dnsCmd.AddCommand(dnsConfigCmd)
	dnsCmd.AddCommand(dnsRestartCmd)
	dnsCmd.AddCommand(dnsReloadCmd)
	dnsCmd.AddCommand(dnsCheckCmd)
	dnsCmd.AddCommand(dnsZonesCmd)
	dnsCmd.AddCommand(dnsLogsCmd)
	dnsCmd.AddCommand(dnsQueryCmd)
	dnsCmd.AddCommand(dnsBackupCmd)
	dnsCmd.AddCommand(dnsRestoreCmd)
	dnsCmd.AddCommand(dnsDNSSECCmd)
	dnsCmd.AddCommand(dnsStatsCmd)
	dnsCmd.AddCommand(dnsQuerylogCmd)
}

// Implementation functions

func installDNS(mode, masterIP, serverIP, clusterName string) {
	fmt.Println("🚀 Installing Bind9 DNS Server...")

	// Default to master if not specified
	if mode == "" {
		mode = "master"
	}

	// Validate master-slave setup
	if mode == "slave" && masterIP == "" {
		fmt.Println("❌ Slave mode requires --master-ip flag")
		return
	}

	// Auto-detect server IP if not provided
	if serverIP == "" {
		serverIP = detectServerIP()
		if serverIP == "" {
			fmt.Println("❌ Could not detect server IP. Please specify with --server-ip")
			return
		}
		fmt.Printf("✓ Auto-detected server IP: %s\n", serverIP)
	}

	// Step 1: Update packages and install Bind9
	fmt.Println("📦 Installing Bind9...")
	if err := exec.Command("apt", "update").Run(); err != nil {
		fmt.Printf("❌ Failed to update package list: %v\n", err)
		return
	}

	if err := exec.Command("apt", "install", "-y", "bind9", "bind9-utils", "bind9-doc").Run(); err != nil {
		fmt.Printf("❌ Failed to install Bind9: %v\n", err)
		return
	}
	fmt.Println("✓ Bind9 installed")

	// Step 2: Create configuration directories
	fmt.Println("📁 Setting up directories...")
	os.MkdirAll("/etc/bind/zones/master", 0755)
	os.MkdirAll("/etc/bind/zones/slave", 0755)
	os.MkdirAll("/var/cache/bind", 0755)
	os.MkdirAll("/var/log/named", 0755)
	os.MkdirAll("/var/lib/bind", 0755)
	exec.Command("chown", "-R", "bind:bind", "/etc/bind").Run()
	exec.Command("chown", "-R", "bind:bind", "/var/cache/bind").Run()
	exec.Command("chown", "-R", "bind:bind", "/var/log/named").Run()
	exec.Command("chown", "-R", "bind:bind", "/var/lib/bind").Run()
	
	// Create log file with proper permissions
	logFile := "/var/log/named/default.log"
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		os.WriteFile(logFile, []byte(""), 0640)
		exec.Command("chown", "bind:bind", logFile).Run()
	}
	fmt.Println("✓ Directories configured")

	// Step 3: Deploy named.conf configuration
	fmt.Println("⚙️  Generating Bind9 configuration...")
	if !deployNamedConf(serverIP, mode, masterIP, clusterName) {
		fmt.Println("❌ Failed to deploy Bind9 configuration")
		return
	}
	fmt.Println("✓ Configuration deployed")

	// Step 4: Test configuration
	fmt.Println("🔍 Testing Bind9 configuration...")
	if err := exec.Command("named-checkconf").Run(); err != nil {
		fmt.Println("❌ Bind9 configuration test failed")
		fmt.Println("   Run 'sudo named-checkconf' for details")
		return
	}
	fmt.Println("✓ Configuration valid")

	// Step 5: Enable and start service
	fmt.Println("🔄 Starting Bind9 service...")
	exec.Command("systemctl", "enable", "bind9").Run()
	if err := exec.Command("systemctl", "restart", "bind9").Run(); err != nil {
		fmt.Printf("❌ Failed to start Bind9: %v\n", err)
		return
	}
	fmt.Println("✓ Bind9 service started")

	// Step 6: Configure firewall
	fmt.Println("🔥 Configuring firewall...")
	exec.Command("ufw", "allow", "53/tcp").Run()
	exec.Command("ufw", "allow", "53/udp").Run()
	fmt.Println("✓ Firewall configured")

	// Success message
	fmt.Println("\n" + strings.Repeat("═", 70))
	fmt.Println("✅ Bind9 DNS Server installed successfully!")
	fmt.Printf("   Mode: %s\n", mode)
	fmt.Printf("   Server IP: %s\n", serverIP)
	if mode == "slave" {
		fmt.Printf("   Master IP: %s\n", masterIP)
	}
	if clusterName != "" {
		fmt.Printf("   Cluster: %s\n", clusterName)
	}
	fmt.Println("   Query DNS: dig @" + serverIP)
	fmt.Println(strings.Repeat("═", 70))
}

func uninstallDNS() {
	fmt.Println("🗑️  Removing Bind9 DNS Server...")

	// Stop service
	fmt.Println("🛑 Stopping Bind9...")
	exec.Command("systemctl", "stop", "bind9").Run()
	exec.Command("systemctl", "disable", "bind9").Run()

	// Backup configuration
	fmt.Println("💾 Backing up configuration...")
	exec.Command("bash", "-c", "tar -czf /tmp/bind9-backup-$(date +%s).tar.gz /etc/bind 2>/dev/null").Run()

	// Remove package
	fmt.Println("📦 Removing Bind9 package...")
	exec.Command("apt", "purge", "-y", "bind9", "bind9-utils", "bind9-doc").Run()

	// Clean up directories
	fmt.Println("🧹 Cleaning up...")
	exec.Command("bash", "-c", "rm -rf /etc/bind* /var/cache/bind* /var/log/named/default.log* /var/lib/bind*").Run()

	// Remove firewall rules
	exec.Command("ufw", "delete", "allow", "53/tcp").Run()
	exec.Command("ufw", "delete", "allow", "53/udp").Run()

	fmt.Println("✅ Bind9 DNS Server uninstalled successfully")
}

func showDNSStatus() {
	fmt.Println("📊 Bind9 DNS Server Status")
	fmt.Println("─────────────────────────────────────────")

	// Check if installed
	if err := exec.Command("which", "named").Run(); err != nil {
		fmt.Println("❌ Bind9: Not installed")
		return
	}
	fmt.Println("✅ Bind9: Installed")

	// Check service status
	if err := exec.Command("systemctl", "is-active", "--quiet", "bind9").Run(); err != nil {
		fmt.Println("   Service: ⚠️  Stopped")
	} else {
		fmt.Println("   Service: ✅ Running")
	}

	// Read configuration
	if data, err := os.ReadFile("/etc/bind/named.conf.local"); err == nil {
		content := string(data)
		if strings.Contains(content, "zone") {
			fmt.Println("   Zones: ✅ Configured")
		}
		if strings.Contains(content, "notify") {
			fmt.Println("   Replication: ✅ Enabled")
		}
	}

	// Get server IP
	if data, err := os.ReadFile("/etc/bind/named.conf"); err == nil {
		content := string(data)
		if idx := strings.Index(content, "listen-on"); idx != -1 {
			line := strings.Split(content[idx:], "\n")[0]
			fmt.Printf("   Config: %s\n", strings.TrimSpace(line))
		}
	}

	// DNS query test
	fmt.Println("   DNS Test:")
	output, _ := exec.Command("dig", "@127.0.0.1", "google.com", "+short").Output()
	if len(output) > 0 {
		fmt.Printf("   ✅ Recursion working: %s\n", strings.TrimSpace(string(output)))
	} else {
		fmt.Println("   ⚠️  Recursion not responding")
	}
}

func deployNamedConf(serverIP, mode, masterIP, clusterName string) bool {
	// Get named.conf template
	templateContent, err := templates.GetDNSTemplate("named.conf")
	if err != nil {
		fmt.Printf("⚠️  Could not read DNS template: %v\n", err)
		return false
	}

	// Parse and execute template
	tmpl, err := template.New("dns").Parse(string(templateContent))
	if err != nil {
		fmt.Printf("⚠️  Could not parse DNS template: %v\n", err)
		return false
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, map[string]interface{}{
		"ServerIP":    serverIP,
		"Mode":        mode,
		"MasterIP":    masterIP,
		"ClusterName": clusterName,
	})
	if err != nil {
		fmt.Printf("⚠️  Could not execute DNS template: %v\n", err)
		return false
	}

	// Write to named.conf
	configPath := "/etc/bind/named.conf"
	if err := os.WriteFile(configPath, []byte(buf.String()), 0644); err != nil {
		fmt.Printf("❌ Failed to write DNS config: %v\n", err)
		return false
	}

	exec.Command("chown", "bind:bind", configPath).Run()
	exec.Command("chmod", "644", configPath).Run()

	return true
}

func configureDNSSlave(slaveIP string, add bool) {
	fmt.Printf("🔧 %s slave server: %s\n", map[bool]string{true: "Adding", false: "Removing"}[add], slaveIP)

	// Read current config
	data, err := os.ReadFile("/etc/bind/named.conf.local")
	if err != nil {
		fmt.Printf("❌ Could not read DNS config: %v\n", err)
		return
	}

	content := string(data)

	if add {
		// Add slave to notify list
		notifyLine := fmt.Sprintf("    notify { %s; };\n", slaveIP)
		if !strings.Contains(content, slaveIP) {
			content = strings.Replace(content, "    notify {", notifyLine, 1)
		}
		fmt.Printf("✅ Slave %s added to replication\n", slaveIP)
	} else {
		// Remove slave from notify list
		content = strings.ReplaceAll(content, fmt.Sprintf("    notify { %s; };\n", slaveIP), "")
		fmt.Printf("✅ Slave %s removed from replication\n", slaveIP)
	}

	// Write back config
	if err := os.WriteFile("/etc/bind/named.conf.local", []byte(content), 0644); err != nil {
		fmt.Printf("❌ Failed to update config: %v\n", err)
		return
	}

	// Test and reload
	if err := exec.Command("named-checkconf").Run(); err != nil {
		fmt.Println("❌ Configuration invalid, reverting...")
		return
	}

	exec.Command("systemctl", "reload", "bind9").Run()
	fmt.Println("✓ Bind9 reloaded")
}

func configureZone(zoneName, zoneType string) {
	fmt.Printf("⚙️  Configuring zone: %s (type: %s)\n", zoneName, zoneType)

	// Read current config
	data, err := os.ReadFile("/etc/bind/named.conf.local")
	if err != nil {
		// If file doesn't exist, create it with the zone
		data = []byte("")
	}

	content := string(data)

	// Check if zone already exists
	if strings.Contains(content, fmt.Sprintf(`zone "%s"`, zoneName)) {
		fmt.Printf("⚠️  Zone %s already configured\n", zoneName)
		return
	}

	// Build zone configuration
	zoneConfig := fmt.Sprintf("\nzone \"%s\" {\n", zoneName)
	if zoneType == "slave" {
		zoneConfig += "\ttype slave;\n"
		zoneConfig += fmt.Sprintf("\tfile \"/var/lib/bind/db.%s\";\n", zoneName)
		zoneConfig += "\tmasters { <master-ip>; };\n"
	} else {
		zoneConfig += "\ttype master;\n"
		zoneConfig += fmt.Sprintf("\tfile \"/var/lib/bind/db.%s\";\n", zoneName)
		zoneConfig += "\tallow-transfer { any; };\n"
		zoneConfig += "\tnotify yes;\n"
	}
	zoneConfig += "};\n"

	// Append zone configuration
	content += zoneConfig

	// Write back config
	if err := os.WriteFile("/etc/bind/named.conf.local", []byte(content), 0644); err != nil {
		fmt.Printf("❌ Failed to write zone config: %v\n", err)
		return
	}

	// Test configuration
	if err := exec.Command("named-checkconf").Run(); err != nil {
		fmt.Println("❌ Configuration invalid")
		// Revert by removing the zone config
		originalContent := strings.ReplaceAll(content, zoneConfig, "")
		os.WriteFile("/etc/bind/named.conf.local", []byte(originalContent), 0644)
		return
	}

	// Reload Bind9
	exec.Command("systemctl", "reload", "bind9").Run()
	fmt.Printf("✅ Zone %s configured successfully\n", zoneName)
	
	if zoneType == "slave" {
		fmt.Printf("   Remember to set master IP in: /etc/bind/named.conf.local\n")
	} else {
		fmt.Printf("   Remember to create zone file: /var/lib/bind/db.%s\n", zoneName)
	}
}

func detectServerIP() string {
	// Try to get IP from hostname -I
	output, err := exec.Command("hostname", "-I").Output()
	if err == nil {
		ips := strings.Fields(strings.TrimSpace(string(output)))
		if len(ips) > 0 {
			return ips[0]
		}
	}

	// Fallback: try to get from ip route
	output, err = exec.Command("bash", "-c", "ip route get 8.8.8.8 | awk -F' ' '{print $NF;exit}'").Output()
	if err == nil {
		return strings.TrimSpace(string(output))
	}

	return ""
}

// New commands implementation

func listDNSZones() {
	data, err := os.ReadFile("/etc/bind/named.conf.local")
	if err != nil {
		fmt.Println("❌ Could not read zone configuration")
		return
	}

	content := string(data)
	lines := strings.Split(content, "\n")
	
	zoneCount := 0
	for _, line := range lines {
		if strings.Contains(line, `zone "`) {
			zoneName := strings.TrimSpace(strings.Split(strings.Split(line, `"`)[1], `"`)[0])
			if zoneName != "" {
				zoneCount++
				fmt.Printf("   %d. %s\n", zoneCount, zoneName)
			}
		}
	}
	
	if zoneCount == 0 {
		fmt.Println("   No zones configured")
	}
}

func viewDNSLogs(lines int) {
	cmd := fmt.Sprintf("tail -n %d /var/log/named/default.log 2>/dev/null || echo 'No logs available'", lines)
	output, _ := exec.Command("bash", "-c", cmd).Output()
	fmt.Print(string(output))
}

func testDNSQuery(domain string) {
	fmt.Printf("🔍 Testing DNS query for: %s\n", domain)
	output, err := exec.Command("dig", "@127.0.0.1", domain, "+short").Output()
	if err != nil {
		fmt.Printf("❌ Query failed: %v\n", err)
		return
	}
	
	result := strings.TrimSpace(string(output))
	if result == "" {
		fmt.Println("⚠️  No results returned")
	} else {
		fmt.Printf("✅ Results:\n%s\n", result)
	}
}

func backupDNS() {
	fmt.Println("💾 Backing up DNS configuration...")
	timestampOutput, _ := exec.Command("date", "+%Y%m%d_%H%M%S").Output()
	backupName := fmt.Sprintf("/tmp/dns-backup-%s.tar.gz", strings.TrimSpace(string(timestampOutput)))
	
	cmd := fmt.Sprintf("tar -czf %s /etc/bind /var/lib/bind 2>/dev/null", backupName)
	if err := exec.Command("bash", "-c", cmd).Run(); err != nil {
		fmt.Printf("❌ Backup failed: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Backup created: %s\n", backupName)
}

func restoreDNS(backupPath string) {
	fmt.Printf("📥 Restoring DNS from: %s\n", backupPath)
	
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		fmt.Println("❌ Backup file not found")
		return
	}
	
	fmt.Println("🛑 Stopping Bind9 for restore...")
	exec.Command("systemctl", "stop", "bind9").Run()
	
	cmd := fmt.Sprintf("tar -xzf %s -C / 2>/dev/null", backupPath)
	if err := exec.Command("bash", "-c", cmd).Run(); err != nil {
		fmt.Printf("❌ Restore failed: %v\n", err)
		fmt.Println("🔄 Attempting to restart Bind9...")
		exec.Command("systemctl", "start", "bind9").Run()
		return
	}
	
	// Fix permissions
	exec.Command("chown", "-R", "bind:bind", "/etc/bind").Run()
	exec.Command("chown", "-R", "bind:bind", "/var/lib/bind").Run()
	
	fmt.Println("🔄 Starting Bind9...")
	if err := exec.Command("systemctl", "start", "bind9").Run(); err != nil {
		fmt.Printf("❌ Failed to start Bind9: %v\n", err)
		return
	}
	
	fmt.Println("✅ DNS restored successfully")
}

func manageDNSSEC(enable bool) {
	fmt.Printf("🔒 %s DNSSEC validation...\n", map[bool]string{true: "Enabling", false: "Disabling"}[enable])
	
	data, err := os.ReadFile("/etc/bind/named.conf")
	if err != nil {
		fmt.Println("❌ Could not read named.conf")
		return
	}
	
	content := string(data)
	
	if enable {
		if !strings.Contains(content, "dnssec-validation auto;") {
			content = strings.Replace(content, "dnssec-validation auto;", "dnssec-validation auto;", 1)
			if !strings.Contains(content, "dnssec-validation") {
				// Add it if it doesn't exist
				content = strings.Replace(content, "options {", "options {\n    dnssec-validation auto;", 1)
			}
		}
	} else {
		content = strings.Replace(content, "dnssec-validation auto;", "dnssec-validation no;", -1)
	}
	
	if err := os.WriteFile("/etc/bind/named.conf", []byte(content), 0644); err != nil {
		fmt.Println("❌ Failed to update configuration")
		return
	}
	
	exec.Command("systemctl", "reload", "bind9").Run()
	fmt.Printf("✅ DNSSEC %s\n", map[bool]string{true: "enabled", false: "disabled"}[enable])
}

func showDNSStats() {
	fmt.Println("📊 DNS Query Statistics")
	fmt.Println("─────────────────────────────────────────")
	
	// Try to get stats from rndc
	output, err := exec.Command("rndc", "stats").Output()
	if err == nil {
		fmt.Printf("Stats command: %s\n", strings.TrimSpace(string(output)))
	}
	
	// Show log summary
	cmd := "tail -1000 /var/log/named/default.log 2>/dev/null | grep -c 'query' || echo '0'"
	output, _ = exec.Command("bash", "-c", cmd).Output()
	fmt.Printf("Queries in last 1000 log entries: %s", string(output))
	
	cmd = "tail -1000 /var/log/named/default.log 2>/dev/null | grep -c 'NXDOMAIN' || echo '0'"
	output, _ = exec.Command("bash", "-c", cmd).Output()
	fmt.Printf("NXDOMAIN responses: %s", string(output))
	
	cmd = "tail -1000 /var/log/named/default.log 2>/dev/null | grep -c 'SERVFAIL' || echo '0'"
	output, _ = exec.Command("bash", "-c", cmd).Output()
	fmt.Printf("SERVFAIL responses: %s", string(output))
}

func manageQueryLog(enable bool) {
	fmt.Printf("📝 %s query logging...\n", map[bool]string{true: "Enabling", false: "Disabling"}[enable])
	
	data, err := os.ReadFile("/etc/bind/named.conf")
	if err != nil {
		fmt.Println("❌ Could not read named.conf")
		return
	}
	
	content := string(data)
	
	if enable {
		// Add query logging config if not present
		if !strings.Contains(content, "querylog yes;") {
			content = strings.Replace(content, "querylog no;", "querylog yes;", -1)
			if !strings.Contains(content, "querylog") {
				content = strings.Replace(content, "options {", "options {\n    querylog yes;", 1)
			}
		}
	} else {
		content = strings.Replace(content, "querylog yes;", "querylog no;", -1)
	}
	
	if err := os.WriteFile("/etc/bind/named.conf", []byte(content), 0644); err != nil {
		fmt.Println("❌ Failed to update configuration")
		return
	}
	
	exec.Command("systemctl", "reload", "bind9").Run()
	fmt.Printf("✅ Query logging %s\n", map[bool]string{true: "enabled", false: "disabled"}[enable])
}
