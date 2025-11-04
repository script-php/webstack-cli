package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var firewallCmd = &cobra.Command{
	Use:   "firewall",
	Short: "Firewall rules management",
	Long:  `Manage firewall rules, view open ports, and control access to services.`,
}

var firewallStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show all firewall rules",
	Long:  `Display all active firewall rules and open ports.`,
	Run: func(cmd *cobra.Command, args []string) {
		firewallStatus()
	},
}

var firewallOpenPortCmd = &cobra.Command{
	Use:   "open [port] [protocol]",
	Short: "Open a port in the firewall",
	Long:  `Open a specific port. Protocol can be 'tcp', 'udp', or 'both' (default: both).`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		port := args[0]
		protocol := "both"
		if len(args) > 1 {
			protocol = args[1]
		}
		openFirewallPort(port, protocol)
	},
}

var firewallClosePortCmd = &cobra.Command{
	Use:   "close [port] [protocol]",
	Short: "Close a port in the firewall",
	Long:  `Close a specific port. Protocol can be 'tcp', 'udp', or 'both' (default: both).`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		port := args[0]
		protocol := "both"
		if len(args) > 1 {
			protocol = args[1]
		}
		closeFirewallPort(port, protocol)
	},
}

var firewallBlockIPCmd = &cobra.Command{
	Use:   "block [ip]",
	Short: "Block an IP address",
	Long:  `Add an IP address to the blocklist.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		blockIP(args[0])
	},
}

var firewallUnblockIPCmd = &cobra.Command{
	Use:   "unblock [ip]",
	Short: "Unblock an IP address",
	Long:  `Remove an IP address from the blocklist.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		unblockIP(args[0])
	},
}

var firewallListBlockedCmd = &cobra.Command{
	Use:   "blocked",
	Short: "List blocked IP addresses",
	Long:  `Display all currently blocked IP addresses.`,
	Run: func(cmd *cobra.Command, args []string) {
		listBlockedIPs()
	},
}

var firewallFlushCmd = &cobra.Command{
	Use:   "flush",
	Short: "Flush all custom firewall rules",
	Long:  `Remove all custom firewall rules (keeps SSH and established connections).`,
	Run: func(cmd *cobra.Command, args []string) {
		confirmed := confirmAction("Are you sure you want to flush all firewall rules?")
		if confirmed {
			flushFirewallRules()
		} else {
			fmt.Println("Operation cancelled.")
		}
	},
}

var firewallRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore default firewall configuration",
	Long:  `Reset firewall to default WebStack configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		confirmed := confirmAction("Are you sure you want to restore default firewall rules?")
		if confirmed {
			restoreDefaultFirewall()
		} else {
			fmt.Println("Operation cancelled.")
		}
	},
}

var firewallSaveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save firewall rules to file",
	Long:  `Backup current firewall rules to a file.`,
	Run: func(cmd *cobra.Command, args []string) {
		saveFirewallRules()
	},
}

var firewallLoadCmd = &cobra.Command{
	Use:   "load [file]",
	Short: "Load firewall rules from file",
	Long:  `Restore firewall rules from a backup file.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		loadFirewallRules(args[0])
	},
}

var firewallStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show firewall statistics",
	Long:  `Display packet and byte statistics for firewall rules.`,
	Run: func(cmd *cobra.Command, args []string) {
		firewallStats()
	},
}

// Implementation functions

func firewallStatus() {
	fmt.Println("\n🔥 WebStack Firewall Status")
	fmt.Println("═══════════════════════════════════════════")

	// Show IPv4 rules
	fmt.Println("\n📋 IPv4 Rules (iptables):")
	fmt.Println("───────────────────────────────────────────")
	output, err := exec.Command("iptables", "-L", "-n", "-v").Output()
	if err != nil {
		fmt.Printf("❌ Error reading IPv4 rules: %v\n", err)
	} else {
		fmt.Print(string(output))
	}

	// Show IPv6 rules
	fmt.Println("\n📋 IPv6 Rules (ip6tables):")
	fmt.Println("───────────────────────────────────────────")
	output6, err := exec.Command("ip6tables", "-L", "-n", "-v").Output()
	if err != nil {
		fmt.Printf("❌ Error reading IPv6 rules: %v\n", err)
	} else {
		fmt.Print(string(output6))
	}

	// Show blocked IPs
	fmt.Println("\n🚫 Blocked IP Addresses (ipset):")
	fmt.Println("───────────────────────────────────────────")
	ipsetOutput, err := exec.Command("ipset", "list", "banned_ips").Output()
	if err != nil {
		fmt.Println("No blocked IPs or ipset not available")
	} else {
		fmt.Print(string(ipsetOutput))
	}
}

func openFirewallPort(port, protocol string) {
	fmt.Printf("🔓 Opening port %s (%s)...\n", port, protocol)

	protocols := []string{}
	if protocol == "both" || protocol == "tcp" {
		protocols = append(protocols, "tcp")
	}
	if protocol == "both" || protocol == "udp" {
		protocols = append(protocols, "udp")
	}

	for _, proto := range protocols {
		// IPv4
		cmd := exec.Command("iptables", "-A", "INPUT", "-p", proto, "--dport", port, "-j", "ACCEPT")
		if err := cmd.Run(); err != nil {
			fmt.Printf("⚠️  IPv4 rule may already exist or error occurred: %v\n", err)
		}

		// IPv6
		cmd6 := exec.Command("ip6tables", "-A", "INPUT", "-p", proto, "--dport", port, "-j", "ACCEPT")
		if err := cmd6.Run(); err != nil {
			fmt.Printf("⚠️  IPv6 rule may already exist or error occurred: %v\n", err)
		}
	}

	// Persist rules
	persistFirewallRules()
	fmt.Printf("✅ Port %s (%s) opened and persisted\n", port, protocol)
}

func closeFirewallPort(port, protocol string) {
	fmt.Printf("🔒 Closing port %s (%s)...\n", port, protocol)

	protocols := []string{}
	if protocol == "both" || protocol == "tcp" {
		protocols = append(protocols, "tcp")
	}
	if protocol == "both" || protocol == "udp" {
		protocols = append(protocols, "udp")
	}

	for _, proto := range protocols {
		// IPv4
		cmd := exec.Command("iptables", "-D", "INPUT", "-p", proto, "--dport", port, "-j", "ACCEPT")
		if err := cmd.Run(); err != nil {
			fmt.Printf("⚠️  IPv4 rule may not exist: %v\n", err)
		}

		// IPv6
		cmd6 := exec.Command("ip6tables", "-D", "INPUT", "-p", proto, "--dport", port, "-j", "ACCEPT")
		if err := cmd6.Run(); err != nil {
			fmt.Printf("⚠️  IPv6 rule may not exist: %v\n", err)
		}
	}

	// Persist rules
	persistFirewallRules()
	fmt.Printf("✅ Port %s (%s) closed and persisted\n", port, protocol)
}

func blockIP(ip string) {
	fmt.Printf("🚫 Blocking IP %s...\n", ip)

	// Create ipset if not exists
	exec.Command("ipset", "create", "banned_ips", "hash:ip", "forcreate").Run()

	// Add IP to ipset
	cmd := exec.Command("ipset", "add", "banned_ips", ip)
	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Error adding IP to blocklist: %v\n", err)
		return
	}

	// Add iptables rule to block the IP
	exec.Command("iptables", "-A", "INPUT", "-m", "set", "--match-set", "banned_ips", "src", "-j", "DROP").Run()
	exec.Command("ip6tables", "-A", "INPUT", "-m", "set", "--match-set", "banned_ips", "src", "-j", "DROP").Run()

	// Persist
	persistFirewallRules()
	fmt.Printf("✅ IP %s blocked and persisted\n", ip)
}

func unblockIP(ip string) {
	fmt.Printf("✅ Unblocking IP %s...\n", ip)

	// Remove from ipset
	cmd := exec.Command("ipset", "del", "banned_ips", ip)
	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Error removing IP from blocklist: %v\n", err)
		return
	}

	// Persist
	persistFirewallRules()
	fmt.Printf("✅ IP %s unblocked and persisted\n", ip)
}

func listBlockedIPs() {
	fmt.Println("\n🚫 Blocked IP Addresses")
	fmt.Println("═══════════════════════════════════════════")

	output, err := exec.Command("ipset", "list", "banned_ips").Output()
	if err != nil {
		fmt.Println("No blocked IPs found or ipset not available")
		return
	}

	fmt.Print(string(output))
}

func flushFirewallRules() {
	fmt.Println("🧹 Flushing firewall rules...")

	// Keep SSH and localhost, remove everything else
	exec.Command("iptables", "-F", "INPUT").Run()
	exec.Command("ip6tables", "-F", "INPUT").Run()

	// Re-add core security rules
	exec.Command("iptables", "-A", "INPUT", "-i", "lo", "-j", "ACCEPT").Run()
	exec.Command("iptables", "-A", "INPUT", "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT").Run()
	exec.Command("iptables", "-A", "INPUT", "-p", "tcp", "--dport", "22", "-j", "ACCEPT").Run()

	exec.Command("ip6tables", "-A", "INPUT", "-i", "lo", "-j", "ACCEPT").Run()
	exec.Command("ip6tables", "-A", "INPUT", "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT").Run()
	exec.Command("ip6tables", "-A", "INPUT", "-p", "tcp", "--dport", "22", "-j", "ACCEPT").Run()

	persistFirewallRules()
	fmt.Println("✅ Firewall rules flushed (SSH and established connections preserved)")
}

func restoreDefaultFirewall() {
	fmt.Println("🔄 Restoring default firewall configuration...")

	// Flush all
	exec.Command("iptables", "-F").Run()
	exec.Command("ip6tables", "-F").Run()

	// Set default policies
	exec.Command("iptables", "-P", "INPUT", "DROP").Run()
	exec.Command("iptables", "-P", "FORWARD", "DROP").Run()
	exec.Command("iptables", "-P", "OUTPUT", "ACCEPT").Run()

	exec.Command("ip6tables", "-P", "INPUT", "DROP").Run()
	exec.Command("ip6tables", "-P", "FORWARD", "DROP").Run()
	exec.Command("ip6tables", "-P", "OUTPUT", "ACCEPT").Run()

	// Core security rules
	for _, ipVersion := range []string{"iptables", "ip6tables"} {
		ipt := ipVersion
		// Allow localhost
		exec.Command(ipt, "-A", "INPUT", "-i", "lo", "-j", "ACCEPT").Run()
		// Allow established connections
		exec.Command(ipt, "-A", "INPUT", "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT").Run()
		// Allow SSH
		exec.Command(ipt, "-A", "INPUT", "-p", "tcp", "--dport", "22", "-j", "ACCEPT").Run()
	}

	persistFirewallRules()
	fmt.Println("✅ Firewall restored to default configuration")
}

func saveFirewallRules() {
	fmt.Println("💾 Saving firewall rules...")

	backupFile := "/etc/webstack/firewall-backup.tar.gz"

	// Create backup directory if needed
	os.MkdirAll("/etc/webstack", 0755)

	// Save rules
	cmd := exec.Command("bash", "-c",
		"tar -czf "+backupFile+
			" /etc/iptables/rules.v4 /etc/iptables/rules.v6 2>/dev/null || true && "+
			"iptables-save > /etc/webstack/iptables-v4.backup && "+
			"ip6tables-save > /etc/webstack/iptables-v6.backup")

	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Error saving rules: %v\n", err)
		return
	}

	fmt.Printf("✅ Firewall rules saved to %s\n", backupFile)
}

func loadFirewallRules(filePath string) {
	fmt.Printf("📂 Loading firewall rules from %s...\n", filePath)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("❌ File not found: %s\n", filePath)
		return
	}

	// Load IPv4 rules
	cmd := exec.Command("iptables-restore", filePath)
	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️  Error loading IPv4 rules: %v\n", err)
	}

	// Try IPv6
	ipv6File := strings.Replace(filePath, "v4", "v6", -1)
	if _, err := os.Stat(ipv6File); err == nil {
		cmd6 := exec.Command("ip6tables-restore", ipv6File)
		if err := cmd6.Run(); err != nil {
			fmt.Printf("⚠️  Error loading IPv6 rules: %v\n", err)
		}
	}

	persistFirewallRules()
	fmt.Println("✅ Firewall rules loaded and persisted")
}

func firewallStats() {
	fmt.Println("\n📊 Firewall Statistics")
	fmt.Println("═══════════════════════════════════════════")

	fmt.Println("\n📈 IPv4 Statistics:")
	fmt.Println("───────────────────────────────────────────")
	output, err := exec.Command("iptables", "-L", "-n", "-v").Output()
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Print(string(output))
	}

	fmt.Println("\n📈 IPv6 Statistics:")
	fmt.Println("───────────────────────────────────────────")
	output6, err := exec.Command("ip6tables", "-L", "-n", "-v").Output()
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Print(string(output6))
	}

	// Show ipset stats
	fmt.Println("\n📈 ipset Statistics:")
	fmt.Println("───────────────────────────────────────────")
	ipsetOutput, err := exec.Command("ipset", "list").Output()
	if err != nil {
		fmt.Println("No ipsets available")
	} else {
		fmt.Print(string(ipsetOutput))
	}
}

func persistFirewallRules() {
	// Save IPv4 rules
	exec.Command("bash", "-c", "iptables-save > /etc/iptables/rules.v4 2>/dev/null || true").Run()

	// Save IPv6 rules
	exec.Command("bash", "-c", "ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true").Run()

	// Also save ipset rules
	exec.Command("bash", "-c", "ipset save > /etc/iptables/ipset.rules 2>/dev/null || true").Run()
}

func confirmAction(message string) bool {
	fmt.Print(message + " (yes/no): ")
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(response) == "yes" || strings.ToLower(response) == "y"
}

func init() {
	rootCmd.AddCommand(firewallCmd)

	firewallCmd.AddCommand(firewallStatusCmd)
	firewallCmd.AddCommand(firewallOpenPortCmd)
	firewallCmd.AddCommand(firewallClosePortCmd)
	firewallCmd.AddCommand(firewallBlockIPCmd)
	firewallCmd.AddCommand(firewallUnblockIPCmd)
	firewallCmd.AddCommand(firewallListBlockedCmd)
	firewallCmd.AddCommand(firewallFlushCmd)
	firewallCmd.AddCommand(firewallRestoreCmd)
	firewallCmd.AddCommand(firewallSaveCmd)
	firewallCmd.AddCommand(firewallLoadCmd)
	firewallCmd.AddCommand(firewallStatsCmd)
}
