package installer

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"
	"webstack-cli/internal/config"
	"webstack-cli/internal/templates"
)

// ComponentStatus represents the status of a component
type ComponentStatus int

const (
	NotInstalled ComponentStatus = iota
	Installed
	InstallError
)

// Component represents a component that can be installed
type Component struct {
	Name        string
	CheckCmd    []string
	PackageName string
	ServiceName string
}

// Common components
var components = map[string]Component{
	"nginx": {
		Name:        "Nginx",
		CheckCmd:    []string{"dpkg", "-l", "nginx"},
		PackageName: "nginx",
		ServiceName: "nginx",
	},
	"apache": {
		Name:        "Apache",
		CheckCmd:    []string{"dpkg", "-l", "apache2"},
		PackageName: "apache2",
		ServiceName: "apache2",
	},
	"mysql": {
		Name:        "MySQL",
		CheckCmd:    []string{"dpkg", "-l", "mysql-server"},
		PackageName: "mysql-server",
		ServiceName: "mysql",
	},
	"mariadb": {
		Name:        "MariaDB",
		CheckCmd:    []string{"dpkg", "-l", "mariadb-server"},
		PackageName: "mariadb-server",
		ServiceName: "mariadb",
	},
	"postgresql": {
		Name:        "PostgreSQL",
		CheckCmd:    []string{"dpkg", "-l", "postgresql"},
		PackageName: "postgresql postgresql-contrib",
		ServiceName: "postgresql",
	},
	"bind9": {
		Name:        "Bind9 DNS",
		CheckCmd:    []string{"dpkg", "-l", "bind9"},
		PackageName: "bind9 bind9-utils bind9-doc",
		ServiceName: "bind9",
	},
}

// checkComponentStatus checks if a component is already installed
func checkComponentStatus(component Component) ComponentStatus {
	// For packages, use dpkg -l and check for "ii" status (installed)
	if len(component.CheckCmd) == 3 && component.CheckCmd[0] == "dpkg" && component.CheckCmd[1] == "-l" {
		packageName := component.CheckCmd[2]
		if isPackageInstalled(packageName) {
			return Installed
		}
		return NotInstalled
	}

	// For other check commands, use exit code
	cmd := exec.Command(component.CheckCmd[0], component.CheckCmd[1:]...)
	err := cmd.Run()
	if err != nil {
		return NotInstalled
	}
	return Installed
}

// checkPHPVersion checks if a specific PHP version is installed
func checkPHPVersion(version string) ComponentStatus {
	packageName := fmt.Sprintf("php%s-fpm", version)
	// Use dpkg-query to check for "ii" (installed) status specifically
	// ii = installed and configured, rc = removed but config remains
	cmd := exec.Command("dpkg-query", "-W", "-f=${Status}", packageName)
	output, err := cmd.Output()
	if err != nil {
		return NotInstalled
	}
	// Check if package is installed and configured (first two chars should be "ii")
	if len(output) >= 2 && output[0] == 'i' && output[1] == 'i' {
		return Installed
	}
	return NotInstalled
}

// promptForAction asks user what to do when component is already installed
func promptForAction(componentName string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("⚠️  %s is already installed.\n", componentName)
	fmt.Println("What would you like to do?")
	fmt.Println("  [k] Keep current installation")
	fmt.Println("  [r] Remove and reinstall")
	fmt.Println("  [u] Remove/uninstall only")
	fmt.Println("  [s] Skip")
	fmt.Print("Choice (k/r/u/s): ")

	for {
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		response = strings.TrimSpace(strings.ToLower(response))
		switch response {
		case "k", "keep":
			return "keep"
		case "r", "reinstall":
			return "reinstall"
		case "u", "uninstall":
			return "uninstall"
		case "s", "skip":
			return "skip"
		default:
			fmt.Print("Please enter k, r, u, or s: ")
			continue
		}
	}
}

// improvedAskYesNo provides better interactive prompts that wait for user input
func improvedAskYesNo(question string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s (y/N): ", question)

	for {
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			fmt.Printf("%s (y/N): ", question)
			continue
		}

		response = strings.TrimSpace(strings.ToLower(response))
		switch response {
		case "y", "yes":
			return true
		case "n", "no", "":
			return false
		default:
			fmt.Print("Please enter y or n: ")
			continue
		}
	}
}

// uninstallComponent removes a component
func uninstallComponent(component Component) error {
	fmt.Printf("🗑️  Removing %s...\n", component.Name)

	// Stop service if it has one
	if component.ServiceName != "" {
		runCommand("systemctl", "stop", component.ServiceName)
		runCommand("systemctl", "disable", component.ServiceName)
	}

	// For MySQL/MariaDB, do aggressive cleanup of data directories first
	if component.PackageName == "mysql-server" || component.PackageName == "mariadb-server" {
		fmt.Println("🧹 Cleaning MySQL/MariaDB data directories...")

		// Remove all MySQL/MariaDB data directories using glob patterns
		// This ensures we remove /var/lib/mysql, /var/lib/mysql-8.0, /var/lib/mysql-files, etc.
		runCommandQuiet("bash", "-c", "rm -rf /var/lib/mysql*") // Catches mysql, mysql-8.0, mysql-files, etc.
		runCommandQuiet("bash", "-c", "rm -rf /var/log/mysql*") // Catches mysql, mysql-files logs, etc.
		runCommandQuiet("bash", "-c", "rm -rf /etc/mysql*")     // Catches mysql, mysqlrouter configs, etc.
		runCommandQuiet("bash", "-c", "rm -rf /run/mysqld*")    // Catches mysqld, mysqld_safe, etc.

		// Clean package cache to prevent stale files
		runCommandQuiet("apt", "clean")
		runCommandQuiet("apt", "autoclean")
	}

	// For PostgreSQL, do aggressive cleanup of data directories first
	if component.PackageName == "postgresql" {
		fmt.Println("🧹 Cleaning PostgreSQL data directories...")

		// Remove all PostgreSQL data directories using glob patterns
		runCommandQuiet("bash", "-c", "rm -rf /var/lib/postgresql*")
		runCommandQuiet("bash", "-c", "rm -rf /var/log/postgresql*")
		runCommandQuiet("bash", "-c", "rm -rf /etc/postgresql*")
		runCommandQuiet("bash", "-c", "rm -rf /run/postgresql*")

		// Clean package cache to prevent stale files
		runCommandQuiet("apt", "clean")
		runCommandQuiet("apt", "autoclean")
	}

	// Use purge to remove packages and config files
	cmd := exec.Command("apt", "purge", "-y", component.PackageName)
	cmd.Env = append(os.Environ(),
		"DEBIAN_FRONTEND=noninteractive",
		"DEBCONF_NONINTERACTIVE_SEEN=true")

	// Repair dpkg database BEFORE purge to ensure clean state
	fmt.Println("🔧 Repairing dpkg database state (before)...")
	runCommandQuiet("dpkg", "--configure", "-a")

	aptPurgeFailed := false
	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️  apt purge returned error (may not be critical): %v\n", err)
		aptPurgeFailed = true
	}

	// Repair dpkg database AFTER purge to fix any issues from uninstall
	fmt.Println("🔧 Repairing dpkg database state (after)...")
	runCommandQuiet("dpkg", "--configure", "-a")

	// For PostgreSQL, always run aggressive cleanup to ensure complete removal
	if strings.Contains(component.PackageName, "postgresql") {
		fmt.Println("🧹 Running PostgreSQL package cleanup...")
		runCommandQuiet("dpkg", "--purge", "--force-all", "postgresql", "postgresql-contrib", "postgresql-client", "postgresql-common")
		runCommandQuiet("apt", "autoremove", "-y")

		// Ask for reboot after PostgreSQL uninstall
		fmt.Println("")
		fmt.Println("✅ Uninstall completed")
		if improvedAskYesNo("⚠️  A system reboot is recommended to ensure all PostgreSQL processes are terminated. Reboot now?") {
			fmt.Println("🔄 Rebooting system...")
			runCommand("systemctl", "reboot")
		} else {
			fmt.Println("⚠️  Please manually reboot the system before reinstalling PostgreSQL")
		}

		// Return error if apt purge failed
		if aptPurgeFailed {
			return fmt.Errorf("apt purge had errors - running dpkg fallback")
		}
		return nil
	}

	// Also try dpkg --purge as fallback for MySQL/MariaDB
	if component.PackageName == "mysql-server" || component.PackageName == "mariadb-server" {
		runCommandQuiet("dpkg", "--purge", "--force-all", "mysql-server", "mysql-client", "mysql-server-core", "mysql-client-core")
		runCommandQuiet("dpkg", "--purge", "--force-all", "mariadb-server", "mariadb-client", "mariadb-server-core", "mariadb-client-core")
		runCommandQuiet("apt", "autoremove", "-y")

		// Ask for reboot after MySQL/MariaDB uninstall
		fmt.Println("")
		fmt.Println("✅ Uninstall completed")
		if improvedAskYesNo("⚠️  A system reboot is recommended to ensure all MySQL/MariaDB processes are terminated. Reboot now?") {
			fmt.Println("🔄 Rebooting system...")
			runCommand("systemctl", "reboot")
		} else {
			fmt.Println("⚠️  Please manually reboot the system before reinstalling MySQL/MariaDB")
		}
	}

	return nil
}

// uninstallPHP removes a specific PHP version
func uninstallPHP(version string) error {
	fmt.Printf("🗑️  Removing PHP %s...\n", version)

	serviceName := fmt.Sprintf("php%s-fpm", version)
	runCommand("systemctl", "stop", serviceName)
	runCommand("systemctl", "disable", serviceName)

	// Use purge with wildcard to remove all PHP packages including extensions
	phpPattern := fmt.Sprintf("php%s*", version)

	fmt.Println("🧹 Removing PHP packages and extensions...")
	return runCommand("apt", "purge", "-y", phpPattern)
}

// InstallAll runs interactive installation of the complete web stack
func InstallAll() {
	fmt.Println("🚀 WebStack Interactive Installation")
	fmt.Println("===================================")

	// Install base components
	fmt.Println("\n📋 Checking web servers...")
	InstallNginx()
	InstallApache()

	// Ask about database
	fmt.Println("\n📋 Database installation...")
	if improvedAskYesNo("Do you want to install MySQL?") {
		InstallMySQL()
	} else if improvedAskYesNo("Do you want to install MariaDB?") {
		InstallMariaDB()
	}

	// Ask about PostgreSQL
	if improvedAskYesNo("Do you want to install PostgreSQL?") {
		InstallPostgreSQL()
	}

	// Install PHP versions
	fmt.Println("\n📋 PHP installation...")
	phpVersions := []string{"5.6", "7.0", "7.1", "7.2", "7.3", "7.4", "8.0", "8.1", "8.2", "8.3", "8.4"}

	for _, version := range phpVersions {
		if improvedAskYesNo(fmt.Sprintf("Install PHP %s?", version)) {
			InstallPHP(version)
		}
	}

	fmt.Println("\n✅ Installation completed!")
}

// InstallNginx installs and configures Nginx on port 80
func InstallNginx() {
	fmt.Println("📦 Installing Nginx...")

	// Check if already installed
	component := components["nginx"]
	status := checkComponentStatus(component)

	if status == Installed {
		action := promptForAction(component.Name)
		switch action {
		case "keep":
			fmt.Println("✅ Keeping existing Nginx installation")
			return
		case "skip":
			fmt.Println("⏭️  Skipping Nginx installation")
			return
		case "uninstall":
			if err := uninstallComponent(component); err != nil {
				fmt.Printf("Error uninstalling Nginx: %v\n", err)
			}
			UpdateServerConfig("nginx", false, 0, "")
			fmt.Println("✅ Nginx uninstalled")
			return
		case "reinstall":
			fmt.Println("🔄 Reinstalling Nginx...")
			if err := uninstallComponent(component); err != nil {
				fmt.Printf("Error uninstalling Nginx: %v\n", err)
				return
			}
		}
	}

	if err := runCommand("apt", "update"); err != nil {
		fmt.Printf("Error updating package list: %v\n", err)
		return
	}

	if err := runCommand("apt", "install", "-y", "nginx"); err != nil {
		fmt.Printf("Error installing Nginx: %v\n", err)
		return
	}

	// Determine Nginx mode based on whether Apache is installed
	mode, port := determineNginxMode()

	// If Nginx is in proxy mode, we need to move Apache to port 8080
	if mode == "proxy" && isPackageInstalled("apache2") {
		fmt.Println("🔄 Apache detected - configuring for backend mode...")
		// Stop Apache first
		runCommand("systemctl", "stop", "apache2")

		// Regenerate Apache config for port 8080
		apachePort := 8080
		portConfContent := fmt.Sprintf(`# WebStack CLI - Apache Ports Configuration
# Apache listens on port %d

Listen %d

<IfModule ssl_module>
    Listen %d ssl
</IfModule>

<IfModule mod_gnutls.c>
    Listen %d ssl
</IfModule>
`, apachePort, apachePort, apachePort+363, apachePort+363)

		if err := ioutil.WriteFile("/etc/apache2/ports.conf", []byte(portConfContent), 0644); err != nil {
			fmt.Printf("⚠️  Warning: Could not update Apache ports.conf: %v\n", err)
		} else {
			fmt.Printf("✅ Apache reconfigured for port %d (backend mode)\n", apachePort)
		}

		// Regenerate default VirtualHost for port 8080
		if defaultConfig, err := templates.GetApacheTemplate("default.conf"); err == nil {
			tmpl, err := template.New("apache-default").Parse(string(defaultConfig))
			if err == nil {
				var buf strings.Builder
				tmpl.Execute(&buf, map[string]interface{}{
					"ApachePort": apachePort,
				})

				if err := ioutil.WriteFile("/etc/apache2/sites-available/000-default.conf", []byte(buf.String()), 0644); err == nil {
					fmt.Println("✅ Apache default VirtualHost updated for port 8080")
				}
			}
		}

		// Update Apache config
		if err := UpdateServerConfig("apache", true, 8080, "backend"); err != nil {
			fmt.Printf("⚠️  Warning: Could not update Apache config: %v\n", err)
		}
	}

	// Configure Nginx
	configureNginx()

	if err := runCommand("systemctl", "enable", "nginx"); err != nil {
		fmt.Printf("Error enabling Nginx: %v\n", err)
	}

	if err := runCommand("systemctl", "start", "nginx"); err != nil {
		fmt.Printf("Error starting Nginx: %v\n", err)
	}

	// If Apache was moved to backend, restart it
	if mode == "proxy" && isPackageInstalled("apache2") {
		if err := runCommand("systemctl", "start", "apache2"); err != nil {
			fmt.Printf("⚠️  Warning: Could not restart Apache: %v\n", err)
		} else {
			fmt.Println("✅ Apache restarted on port 8080")
		}
	}

	// Configure firewall - open ports 80 and 443 for HTTP/HTTPS
	fmt.Println("🔥 Configuring firewall for HTTP/HTTPS...")
	webPorts := []int{80, 443}
	for _, port := range webPorts {
		portStr := fmt.Sprintf("%d", port)
		// Add both IPv4 and IPv6 rules
		runCommand("iptables", "-A", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT")
		runCommand("ip6tables", "-A", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT")
	}
	// Persist rules
	runCommand("bash", "-c", "iptables-save > /etc/iptables/rules.v4 2>/dev/null || true")
	runCommand("bash", "-c", "ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true")

	// Update config with Nginx installation details
	if err := UpdateServerConfig("nginx", true, port, mode); err != nil {
		fmt.Printf("⚠️  Warning: Could not update config: %v\n", err)
	}

	fmt.Printf("✅ Nginx installed successfully on port %d (mode: %s)\n", port, mode)
}

// InstallNginxVersion installs a specific version of Nginx or the latest if version is empty
func InstallNginxVersion(version string) {
	if version == "" {
		InstallNginx()
		return
	}

	fmt.Printf("📦 Installing Nginx version %s...\n", version)

	if err := runCommand("apt", "update"); err != nil {
		fmt.Printf("Error updating package list: %v\n", err)
		return
	}

	pkg := fmt.Sprintf("nginx=%s*", version)

	done := make(chan error, 1)
	go func() {
		done <- runCommand("apt", "install", "-y", pkg)
	}()

	select {
	case err := <-done:
		if err != nil {
			fmt.Printf("⚠️  Error installing Nginx %s: %v\n", version, err)
			return
		}
	case <-time.After(5 * time.Minute):
		fmt.Println("⚠️  Installation timed out after 5 minutes")
		return
	}

	configureNginx()
	runCommand("systemctl", "enable", "nginx")
	runCommand("systemctl", "start", "nginx")

	// Configure firewall - open ports 80 and 443 for HTTP/HTTPS
	fmt.Println("🔥 Configuring firewall for HTTP/HTTPS...")
	webPorts := []int{80, 443}
	for _, port := range webPorts {
		portStr := fmt.Sprintf("%d", port)
		// Add both IPv4 and IPv6 rules
		runCommand("iptables", "-A", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT")
		runCommand("ip6tables", "-A", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT")
	}
	// Persist rules
	runCommand("bash", "-c", "iptables-save > /etc/iptables/rules.v4 2>/dev/null || true")
	runCommand("bash", "-c", "ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true")

	// Update config to mark Nginx as installed and configured
	if err := UpdateServerConfig("nginx", true, 80, "standalone"); err != nil {
		fmt.Printf("⚠️  Warning: Could not update config: %v\n", err)
	}

	fmt.Printf("✅ Nginx %s installed successfully\n", version)
}

// InstallApache installs and configures Apache
func InstallApache() {
	fmt.Println("📦 Installing Apache...")

	// Check if already installed
	component := components["apache"]
	status := checkComponentStatus(component)

	if status == Installed {
		action := promptForAction(component.Name)
		switch action {
		case "keep":
			fmt.Println("✅ Keeping existing Apache installation")
			return
		case "skip":
			fmt.Println("⏭️  Skipping Apache installation")
			return
		case "uninstall":
			if err := uninstallComponent(component); err != nil {
				fmt.Printf("Error uninstalling Apache: %v\n", err)
			}
			UpdateServerConfig("apache", false, 0, "")
			fmt.Println("✅ Apache uninstalled")
			return
		case "reinstall":
			fmt.Println("🔄 Reinstalling Apache...")
			if err := uninstallComponent(component); err != nil {
				fmt.Printf("Error uninstalling Apache: %v\n", err)
				return
			}
		}
	}

	if err := runCommand("apt", "install", "-y", "apache2"); err != nil {
		fmt.Printf("Error installing Apache: %v\n", err)
		return
	}

	// Determine Apache port and mode based on whether Nginx is installed
	port, mode := determineApachePort()

	// Configure Apache
	configureApache()

	// If Nginx is installed (Apache is backend), need to update Nginx to proxy mode
	if mode == "backend" && isPackageInstalled("nginx") {
		fmt.Println("🔄 Nginx detected - updating to proxy mode...")
		// Update Nginx mode to proxy
		if err := UpdateServerConfig("nginx", true, 80, "proxy"); err != nil {
			fmt.Printf("⚠️  Warning: Could not update Nginx config: %v\n", err)
		}
		// Restart Nginx to activate proxy mode
		if err := runCommand("systemctl", "restart", "nginx"); err != nil {
			fmt.Printf("⚠️  Warning: Could not restart Nginx: %v\n", err)
		}
		// Apache is backend, enable and start it
		runCommand("systemctl", "enable", "apache2")
		runCommand("systemctl", "start", "apache2")
		fmt.Println("✅ Nginx configured as proxy on port 80, Apache enabled on port 8080")
	} else {
		// Apache is standalone, enable and start it
		runCommand("systemctl", "enable", "apache2")
		runCommand("systemctl", "start", "apache2")
	}

	// Configure firewall - open ports 80 and 443 for HTTP/HTTPS
	fmt.Println("🔥 Configuring firewall for HTTP/HTTPS...")
	webPorts := []int{80, 443}
	for _, port := range webPorts {
		portStr := fmt.Sprintf("%d", port)
		// Add both IPv4 and IPv6 rules
		runCommand("iptables", "-A", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT")
		runCommand("ip6tables", "-A", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT")
	}
	// Persist rules
	runCommand("bash", "-c", "iptables-save > /etc/iptables/rules.v4 2>/dev/null || true")
	runCommand("bash", "-c", "ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true")

	// Update config with Apache installation details
	if err := UpdateServerConfig("apache", true, port, mode); err != nil {
		fmt.Printf("⚠️  Warning: Could not update config: %v\n", err)
	}

	if mode == "backend" {
		fmt.Printf("✅ Apache installed successfully on port %d (mode: backend)\n", port)
	} else {
		fmt.Printf("✅ Apache installed successfully on port %d (mode: standalone)\n", port)
	}
}

// InstallApacheVersion installs a specific version of Apache or the latest if version is empty
func InstallApacheVersion(version string) {
	if version == "" {
		InstallApache()
		return
	}

	fmt.Printf("📦 Installing Apache version %s...\n", version)

	if err := runCommand("apt", "update"); err != nil {
		fmt.Printf("Error updating package list: %v\n", err)
		return
	}

	pkg := fmt.Sprintf("apache2=%s*", version)

	done := make(chan error, 1)
	go func() {
		done <- runCommand("apt", "install", "-y", pkg)
	}()

	select {
	case err := <-done:
		if err != nil {
			fmt.Printf("⚠️  Error installing Apache %s: %v\n", version, err)
			return
		}
	case <-time.After(5 * time.Minute):
		fmt.Println("⚠️  Installation timed out after 5 minutes")
		return
	}

	configureApache()
	runCommand("systemctl", "enable", "apache2")
	runCommand("systemctl", "start", "apache2")

	// Configure firewall - open ports 80 and 443 for HTTP/HTTPS
	fmt.Println("🔥 Configuring firewall for HTTP/HTTPS...")
	webPorts := []int{80, 443}
	for _, port := range webPorts {
		portStr := fmt.Sprintf("%d", port)
		// Add both IPv4 and IPv6 rules
		runCommand("iptables", "-A", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT")
		runCommand("ip6tables", "-A", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT")
	}
	// Persist rules
	runCommand("bash", "-c", "iptables-save > /etc/iptables/rules.v4 2>/dev/null || true")
	runCommand("bash", "-c", "ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true")

	// Update config to mark Apache as installed and configured
	if err := UpdateServerConfig("apache", true, 8080, "standalone"); err != nil {
		fmt.Printf("⚠️  Warning: Could not update config: %v\n", err)
	}

	fmt.Printf("✅ Apache %s installed successfully\n", version)
}

// InstallMySQL installs MySQL server
func InstallMySQL() {
	fmt.Println("📦 Installing MySQL...")

	// Check if MariaDB is already installed (conflict)
	if isPackageInstalled("mariadb-server") {
		fmt.Println("⚠️  MariaDB is already installed")
		fmt.Println("   MySQL and MariaDB cannot run simultaneously (port/socket conflict)")
		if improvedAskYesNo("Do you want to uninstall MariaDB first?") {
			if err := uninstallComponent(components["mariadb"]); err != nil {
				fmt.Printf("Error uninstalling MariaDB: %v\n", err)
				return
			}
		} else {
			fmt.Println("⏭️  Skipping MySQL installation")
			return
		}
	}

	// Check if already installed
	component := components["mysql"]
	status := checkComponentStatus(component)

	if status == Installed {
		action := promptForAction(component.Name)
		switch action {
		case "keep":
			fmt.Println("✅ Keeping existing MySQL installation")
			return
		case "skip":
			fmt.Println("⏭️  Skipping MySQL installation")
			return
		case "uninstall":
			if err := uninstallComponent(component); err != nil {
				fmt.Printf("Error uninstalling MySQL: %v\n", err)
			}
			fmt.Println("✅ MySQL uninstalled")
			return
		case "reinstall":
			fmt.Println("🔄 Reinstalling MySQL...")
			if err := uninstallComponent(component); err != nil {
				fmt.Printf("Error uninstalling MySQL: %v\n", err)
				return
			}
		}
	}

	// CLEAN SLATE APPROACH: Remove all MySQL/MariaDB packages and data
	fmt.Println("🧹 Performing clean-slate removal of MySQL/MariaDB...")

	// AGGRESSIVE PRE-KILL: Force kill ALL processes before anything else
	fmt.Println("🔪 Force-killing any running MySQL/MariaDB processes...")
	runCommandQuiet("bash", "-c", "pkill -9 mysqld 2>/dev/null; true")
	runCommandQuiet("bash", "-c", "pkill -9 mariadbd 2>/dev/null; true")
	runCommandQuiet("bash", "-c", "pkill -9 mysql 2>/dev/null; true")
	time.Sleep(1 * time.Second)

	// Stop the service (may fail, that's ok)
	runCommandQuiet("systemctl", "stop", "mysql")
	runCommandQuiet("systemctl", "stop", "mariadb")
	time.Sleep(1 * time.Second)

	// Purge ALL MySQL and MariaDB packages
	fmt.Println("📦 Removing existing packages...")
	purgeCmd := exec.Command("bash", "-c", "apt-get purge -y 'mysql*' 'mariadb*' 2>/dev/null; true")
	purgeCmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	_ = purgeCmd.Run()

	// Remove ALL data and config directories (fresh start) using glob patterns
	cleanupMySQLMariaDBDirectories()

	// Clean apt cache to prevent conflicts
	runCommandQuiet("apt", "clean")
	runCommandQuiet("apt", "autoclean")
	runCommandQuiet("apt", "autoremove", "-y")

	// Update package lists for fresh install
	fmt.Println("🔄 Updating package lists...")
	if err := runCommand("apt", "update"); err != nil {
		fmt.Printf("Error updating package list: %v\n", err)
		return
	}

	// Install MySQL in clean environment with full noninteractive mode
	// Use --no-install-recommends to skip optional packages that cause dependency issues
	fmt.Println("📦 Installing MySQL server (this may take a while)...")
	cmd := exec.Command("bash", "-c", "DEBIAN_FRONTEND=noninteractive DEBCONF_NONINTERACTIVE_SEEN=true apt-get install -y --no-install-recommends mysql-server 2>&1 | head -200")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run with timeout to prevent hanging
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	// Wait up to 5 minutes for install to complete
	select {
	case err := <-done:
		if err != nil {
			// Installation had an error but may have partially succeeded
			fmt.Printf("⚠️  Install completed with status: %v (this may be normal)\n", err)
		}
	case <-time.After(5 * time.Minute):
		fmt.Println("⚠️  Installation timed out after 5 minutes")
		fmt.Println("   This can happen if MySQL postinst scripts hang")
		fmt.Println("   Attempting to continue...")
	}

	// Give postinst scripts a moment to finish
	time.Sleep(2 * time.Second)

	// Try to verify service (it might be partially installed)
	fmt.Println("🔍 Verifying MySQL service...")
	if err := runCommand("systemctl", "restart", "mysql"); err != nil {
		fmt.Printf("⚠️  MySQL service may not be fully installed: %v\n", err)
		fmt.Println("   Continuing with configuration anyway...")
	}

	// Configure MySQL
	configureMySQL()

	// Secure root user if service is active
	if isServiceActive("mysql") {
		secureRootUser("mysql")
	} else {
		fmt.Println("⚠️  MySQL service is not running. Skipping password setup.")
	}

	// Enable on boot
	if err := runCommand("systemctl", "enable", "mysql"); err != nil {
		fmt.Printf("Error enabling MySQL: %v\n", err)
	}

	// Update config to mark MySQL as installed and configured
	if err := UpdateServerConfig("mysql", true, 3306, "backend"); err != nil {
		fmt.Printf("⚠️  Warning: Could not update config: %v\n", err)
	}

	fmt.Println("✅ MySQL installed successfully")
}

// InstallMariaDB installs MariaDB server
func InstallMariaDB() {
	fmt.Println("📦 Installing MariaDB...")

	// Check if MySQL is already installed (conflict)
	if isPackageInstalled("mysql-server") {
		fmt.Println("⚠️  MySQL is already installed")
		fmt.Println("   MariaDB and MySQL cannot run simultaneously (port/socket conflict)")
		if improvedAskYesNo("Do you want to uninstall MySQL first?") {
			if err := uninstallComponent(components["mysql"]); err != nil {
				fmt.Printf("Error uninstalling MySQL: %v\n", err)
				return
			}
		} else {
			fmt.Println("⏭️  Skipping MariaDB installation")
			return
		}
	}

	// Check if already installed
	component := components["mariadb"]
	status := checkComponentStatus(component)

	if status == Installed {
		action := promptForAction(component.Name)
		switch action {
		case "keep":
			fmt.Println("✅ Keeping existing MariaDB installation")
			return
		case "skip":
			fmt.Println("⏭️  Skipping MariaDB installation")
			return
		case "uninstall":
			if err := uninstallComponent(component); err != nil {
				fmt.Printf("Error uninstalling MariaDB: %v\n", err)
			}
			fmt.Println("✅ MariaDB uninstalled")
			return
		case "reinstall":
			fmt.Println("🔄 Reinstalling MariaDB...")
			if err := uninstallComponent(component); err != nil {
				fmt.Printf("Error uninstalling MariaDB: %v\n", err)
				return
			}
		}
	}

	// CLEAN SLATE APPROACH: Remove all MySQL/MariaDB packages and data
	fmt.Println("🧹 Performing clean-slate removal of MySQL/MariaDB...")

	// AGGRESSIVE PRE-KILL: Force kill ALL processes before anything else
	fmt.Println("🔪 Force-killing any running MySQL/MariaDB processes...")
	runCommandQuiet("bash", "-c", "pkill -9 mysqld 2>/dev/null; true")
	runCommandQuiet("bash", "-c", "pkill -9 mariadbd 2>/dev/null; true")
	runCommandQuiet("bash", "-c", "pkill -9 mysql 2>/dev/null; true")
	time.Sleep(1 * time.Second)

	// Stop the service (may fail, that's ok)
	runCommandQuiet("systemctl", "stop", "mysql")
	runCommandQuiet("systemctl", "stop", "mariadb")
	time.Sleep(1 * time.Second)

	// Purge ALL MySQL and MariaDB packages
	fmt.Println("📦 Removing existing packages...")
	purgeCmd := exec.Command("bash", "-c", "apt-get purge -y 'mysql*' 'mariadb*' 2>/dev/null; true")
	purgeCmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	_ = purgeCmd.Run()

	// Remove ALL data and config directories (fresh start) using glob patterns
	cleanupMySQLMariaDBDirectories()

	// Clean apt cache to prevent conflicts
	runCommandQuiet("apt", "clean")
	runCommandQuiet("apt", "autoclean")
	runCommandQuiet("apt", "autoremove", "-y")

	// Update package lists for fresh install
	fmt.Println("🔄 Updating package lists...")
	if err := runCommand("apt", "update"); err != nil {
		fmt.Printf("Error updating package list: %v\n", err)
		return
	}

	// Install MariaDB in clean environment with full noninteractive mode
	// Use --no-install-recommends to skip plugin packages that cause dependency issues
	fmt.Println("📦 Installing MariaDB server (this may take a while)...")
	cmd := exec.Command("bash", "-c", "DEBIAN_FRONTEND=noninteractive DEBCONF_NONINTERACTIVE_SEEN=true apt-get install -y --no-install-recommends mariadb-server 2>&1 | head -200")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run with timeout to prevent hanging
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	// Wait up to 5 minutes for install to complete
	select {
	case err := <-done:
		if err != nil {
			// Installation had an error but may have partially succeeded
			fmt.Printf("⚠️  Install completed with status: %v (this may be normal)\n", err)
		}
	case <-time.After(5 * time.Minute):
		fmt.Println("⚠️  Installation timed out after 5 minutes")
		fmt.Println("   This can happen if MariaDB postinst scripts hang")
		fmt.Println("   Attempting to continue...")
	}

	// Give postinst scripts a moment to finish
	time.Sleep(2 * time.Second)

	// Try to verify service (it might be partially installed)
	fmt.Println("🔍 Verifying MariaDB service...")
	if err := runCommand("systemctl", "restart", "mariadb"); err != nil {
		fmt.Printf("⚠️  MariaDB service may not be fully installed: %v\n", err)
		fmt.Println("   Continuing with configuration anyway...")
	}

	// Configure MariaDB
	configureMariaDB()

	// Secure root user if service is active
	if isServiceActive("mariadb") {
		secureRootUser("mariadb")
	} else {
		fmt.Println("⚠️  MariaDB service is not running. Skipping password setup.")
	}

	// Enable on boot
	if err := runCommand("systemctl", "enable", "mariadb"); err != nil {
		fmt.Printf("Error enabling MariaDB: %v\n", err)
	}

	fmt.Println("✅ MariaDB installed successfully")
}

// InstallPostgreSQL installs PostgreSQL server
func InstallPostgreSQL() {
	InstallPostgreSQLVersion("")
}

// InstallPostgreSQLVersion installs a specific version of PostgreSQL or latest if version is empty
func InstallPostgreSQLVersion(version string) {
	if version == "" {
		fmt.Println("📦 Installing PostgreSQL...")
	} else {
		fmt.Printf("📦 Installing PostgreSQL version %s...\n", version)
	}

	// Check if already installed
	component := components["postgresql"]
	status := checkComponentStatus(component)

	if status == Installed {
		action := promptForAction(component.Name)
		switch action {
		case "keep":
			fmt.Println("✅ Keeping existing PostgreSQL installation")
			return
		case "skip":
			fmt.Println("⏭️  Skipping PostgreSQL installation")
			return
		case "uninstall":
			if err := uninstallComponent(component); err != nil {
				fmt.Printf("Error uninstalling PostgreSQL: %v\n", err)
			}
			fmt.Println("✅ PostgreSQL uninstalled")
			return
		case "reinstall":
			fmt.Println("🔄 Reinstalling PostgreSQL...")
			if err := uninstallComponent(component); err != nil {
				fmt.Printf("Error uninstalling PostgreSQL: %v\n", err)
				return
			}
		}
	}

	// Pre-install cleanup
	fmt.Println("🧹 Cleaning up previous PostgreSQL installations...")
	runCommandQuiet("apt", "purge", "-y", "postgresql*")
	runCommandQuiet("dpkg", "--purge", "--force-all", "postgresql", "postgresql-contrib", "postgresql-client", "postgresql-common")
	runCommandQuiet("rm", "-rf", "/var/lib/postgresql*")
	runCommandQuiet("rm", "-rf", "/etc/postgresql*")
	runCommandQuiet("rm", "-rf", "/run/postgresql*")
	runCommandQuiet("apt", "autoremove", "-y")
	runCommandQuiet("apt", "clean")

	// Build package specification
	var pgPackage string
	if version == "" {
		pgPackage = "postgresql"
	} else {
		pgPackage = fmt.Sprintf("postgresql=%s*", version)
	}

	// Setup timeout for installation (prevent hanging)
	done := make(chan error, 1)
	go func() {
		if err := runCommand("apt", "update"); err != nil {
			done <- fmt.Errorf("apt update failed: %v", err)
			return
		}

		// Fix broken dependencies before install
		fmt.Println("🔧 Fixing broken dependencies (before install)...")
		runCommandQuiet("apt", "--fix-broken", "install", "-y")

		if err := runCommand("apt", "install", "-y", pgPackage, "postgresql-contrib"); err != nil {
			done <- fmt.Errorf("postgres installation failed: %v", err)
			return
		}

		// Fix broken dependencies after install
		fmt.Println("🔧 Fixing broken dependencies (after install)...")
		runCommandQuiet("apt", "--fix-broken", "install", "-y")

		done <- nil
	}()

	// 5-minute timeout
	select {
	case err := <-done:
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return
		}
	case <-time.After(5 * time.Minute):
		fmt.Println("❌ PostgreSQL installation timed out (5 minutes)")
		fmt.Println("💡 Try manually: sudo apt install postgresql postgresql-contrib")
		return
	}

	configurePostgreSQL()

	if err := runCommand("systemctl", "enable", "postgresql"); err != nil {
		fmt.Printf("Error enabling PostgreSQL: %v\n", err)
	}

	// Update config to mark PostgreSQL as installed and configured
	if err := UpdateServerConfig("postgresql", true, 5432, "backend"); err != nil {
		fmt.Printf("⚠️  Warning: Could not update config: %v\n", err)
	}

	if version == "" {
		fmt.Println("✅ PostgreSQL installed successfully")
	} else {
		fmt.Printf("✅ PostgreSQL %s installed successfully\n", version)
	}
}

// InstallPHP installs specific PHP-FPM version
func InstallPHP(version string) {
	fmt.Printf("📦 Installing PHP %s...\n", version)

	// Check if already installed
	status := checkPHPVersion(version)

	if status == Installed {
		action := promptForAction(fmt.Sprintf("PHP %s", version))
		switch action {
		case "keep":
			fmt.Printf("✅ Keeping existing PHP %s installation\n", version)
			// Even when keeping an existing PHP install, ensure the WebStack FPM pool
			// is present and the service is restarted so new pool configs take effect.
			configurePHP(version)
			serviceName := fmt.Sprintf("php%s-fpm", version)
			if err := runCommand("systemctl", "enable", serviceName); err != nil {
				fmt.Printf("Error enabling PHP %s FPM: %v\n", version, err)
			}
			if err := runCommand("systemctl", "restart", serviceName); err != nil {
				fmt.Printf("Error restarting PHP %s FPM: %v\n", version, err)
			}
			return
		case "skip":
			fmt.Printf("⏭️  Skipping PHP %s installation\n", version)
			return
		case "uninstall":
			if err := uninstallPHP(version); err != nil {
				fmt.Printf("Error uninstalling PHP %s: %v\n", version, err)
			}
			fmt.Printf("✅ PHP %s uninstalled\n", version)
			return
		case "reinstall":
			fmt.Printf("🔄 Reinstalling PHP %s...\n", version)
			if err := uninstallPHP(version); err != nil {
				fmt.Printf("Error uninstalling PHP %s: %v\n", version, err)
				return
			}
		}
	}

	// Add PHP repository
	if err := runCommand("apt", "install", "-y", "software-properties-common"); err != nil {
		fmt.Printf("Error installing prerequisites: %v\n", err)
		return
	}

	if err := runCommand("add-apt-repository", "-y", "ppa:ondrej/php"); err != nil {
		fmt.Printf("Error adding PHP repository: %v\n", err)
		return
	}

	if err := runCommand("apt", "update"); err != nil {
		fmt.Printf("Error updating package list: %v\n", err)
		return
	}

	phpPackage := fmt.Sprintf("php%s-fpm", version)
	commonPackages := []string{
		phpPackage,
		// Core & CLI
		fmt.Sprintf("php%s-cli", version),
		fmt.Sprintf("php%s-common", version),
		// Database extensions
		fmt.Sprintf("php%s-mysql", version),
		fmt.Sprintf("php%s-pgsql", version),
		// Web & content management
		fmt.Sprintf("php%s-curl", version),
		fmt.Sprintf("php%s-gd", version),
		fmt.Sprintf("php%s-xml", version),
		// Compression & archives
		fmt.Sprintf("php%s-zip", version),
		fmt.Sprintf("php%s-bz2", version),
		// String & encoding
		fmt.Sprintf("php%s-mbstring", version),
		// Security & hashing
		fmt.Sprintf("php%s-bcmath", version),
		// Mail (Roundcube, WordPress, etc.)
		fmt.Sprintf("php%s-imap", version),
		fmt.Sprintf("php%s-intl", version),
		// Image processing
		fmt.Sprintf("php%s-imagick", version),
		// Caching
		fmt.Sprintf("php%s-memcached", version),
		fmt.Sprintf("php%s-redis", version),
		// LDAP
		fmt.Sprintf("php%s-ldap", version),
		// SOAP
		fmt.Sprintf("php%s-soap", version),
	}

	args := append([]string{"install", "-y", "--no-install-recommends"}, commonPackages...)
	if err := runCommand("apt", args...); err != nil {
		fmt.Printf("⚠️  Warning: PHP installation had issues: %v\n", err)
		fmt.Println("   Attempting to configure and recover...")
	}

	// Stop the service to prevent conflicts during configuration
	runCommandQuiet("systemctl", "stop", fmt.Sprintf("php%s-fpm", version))
	time.Sleep(1 * time.Second)

	// Configure PHP-FPM (this removes default www.conf and creates webstack pool)
	configurePHP(version)

	// Fix dpkg database in case of issues
	fmt.Println("🔧 Repairing package configuration...")
	runCommandQuiet("dpkg", "--configure", "-a")

	// Now try to enable and start the service
	serviceName := fmt.Sprintf("php%s-fpm", version)
	if err := runCommand("systemctl", "enable", serviceName); err != nil {
		fmt.Printf("⚠️  Warning: Could not enable PHP %s FPM: %v\n", version, err)
	}

	fmt.Printf("🚀 Starting PHP %s FPM service...\n", version)
	if err := runCommand("systemctl", "restart", serviceName); err != nil {
		fmt.Printf("❌ Error starting PHP %s FPM: %v\n", version, err)
		fmt.Println("   Troubleshooting steps:")
		fmt.Printf("   1. Check status: sudo systemctl status php%s-fpm\n", version)
		fmt.Printf("   2. View logs: sudo journalctl -xeu php%s-fpm.service\n", version)
		fmt.Printf("   3. Check config: php-fpm%s -t\n", version)
		return
	}

	fmt.Printf("✅ PHP %s installed and started successfully\n", version)
}

// InstallMySQLVersion installs a specific version of MySQL or latest if version is empty
func InstallMySQLVersion(version string) {
	if version == "" {
		InstallMySQL()
		return
	}

	fmt.Printf("📦 Installing MySQL version %s...\n", version)

	// Check if MariaDB is already installed (conflict)
	if isPackageInstalled("mariadb-server") {
		fmt.Println("⚠️  MariaDB is already installed")
		fmt.Println("   MySQL and MariaDB cannot run simultaneously (port/socket conflict)")
		if improvedAskYesNo("Do you want to uninstall MariaDB first?") {
			if err := uninstallComponent(components["mariadb"]); err != nil {
				fmt.Printf("Error uninstalling MariaDB: %v\n", err)
				return
			}
		} else {
			fmt.Println("⏭️  Skipping MySQL installation")
			return
		}
	}

	// Clean slate
	fmt.Println("🧹 Performing clean-slate removal of MySQL/MariaDB...")
	fmt.Println("🔪 Force-killing any running MySQL/MariaDB processes...")
	runCommandQuiet("bash", "-c", "pkill -9 mysqld 2>/dev/null; true")
	runCommandQuiet("bash", "-c", "pkill -9 mariadbd 2>/dev/null; true")
	runCommandQuiet("bash", "-c", "pkill -9 mysql 2>/dev/null; true")
	time.Sleep(1 * time.Second)

	runCommandQuiet("systemctl", "stop", "mysql")
	runCommandQuiet("systemctl", "stop", "mariadb")
	time.Sleep(1 * time.Second)

	fmt.Println("📦 Removing existing packages...")
	purgeCmd := exec.Command("bash", "-c", "apt-get purge -y 'mysql*' 'mariadb*' 2>/dev/null; true")
	purgeCmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	_ = purgeCmd.Run()

	// Remove ALL data and config directories (fresh start) using glob patterns
	cleanupMySQLMariaDBDirectories()

	runCommandQuiet("apt", "clean")
	runCommandQuiet("apt", "autoclean")
	runCommandQuiet("apt", "autoremove", "-y")

	fmt.Println("🔄 Updating package lists...")
	if err := runCommand("apt", "update"); err != nil {
		fmt.Printf("Error updating package list: %v\n", err)
		return
	}

	// Install specific MySQL version
	fmt.Printf("📦 Installing MySQL version %s...\n", version)

	// Fix broken dependencies before install
	fmt.Println("🔧 Fixing broken dependencies (before install)...")
	runCommandQuiet("apt", "--fix-broken", "install", "-y")

	packageSpec := fmt.Sprintf("mysql-server=%s*", version)
	cmd := exec.Command("bash", "-c", fmt.Sprintf("DEBIAN_FRONTEND=noninteractive DEBCONF_NONINTERACTIVE_SEEN=true apt-get install -y --no-install-recommends '%s' 2>&1 | head -200", packageSpec))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			fmt.Printf("⚠️  Install completed with status: %v (this may be normal)\n", err)
		}
	case <-time.After(5 * time.Minute):
		fmt.Println("⚠️  Installation timed out after 5 minutes")
		fmt.Println("   Continuing anyway...")
	}

	// Fix broken dependencies after install
	fmt.Println("🔧 Fixing broken dependencies (after install)...")
	runCommandQuiet("apt", "--fix-broken", "install", "-y")

	time.Sleep(2 * time.Second)

	fmt.Println("🔍 Verifying MySQL service...")
	if err := runCommand("systemctl", "restart", "mysql"); err != nil {
		fmt.Printf("⚠️  MySQL service may not be fully installed: %v\n", err)
		fmt.Println("   Continuing with configuration anyway...")
	}

	configureMySQL()

	if isServiceActive("mysql") {
		secureRootUser("mysql")
	} else {
		fmt.Println("⚠️  MySQL service is not running. Skipping password setup.")
	}

	if err := runCommand("systemctl", "enable", "mysql"); err != nil {
		fmt.Printf("Error enabling MySQL: %v\n", err)
	}

	// Update config to mark MySQL as installed and configured
	if err := UpdateServerConfig("mysql", true, 3306, "backend"); err != nil {
		fmt.Printf("⚠️  Warning: Could not update config: %v\n", err)
	}

	fmt.Printf("✅ MySQL %s installed successfully\n", version)
}

// InstallMariaDBVersion installs a specific version of MariaDB or latest if version is empty
func InstallMariaDBVersion(version string) {
	if version == "" {
		InstallMariaDB()
		return
	}

	fmt.Printf("📦 Installing MariaDB version %s...\n", version)

	// Check if MySQL is already installed (conflict)
	if isPackageInstalled("mysql-server") {
		fmt.Println("⚠️  MySQL is already installed")
		fmt.Println("   MariaDB and MySQL cannot run simultaneously (port/socket conflict)")
		if improvedAskYesNo("Do you want to uninstall MySQL first?") {
			if err := uninstallComponent(components["mysql"]); err != nil {
				fmt.Printf("Error uninstalling MySQL: %v\n", err)
				return
			}
		} else {
			fmt.Println("⏭️  Skipping MariaDB installation")
			return
		}
	}

	// Clean slate
	fmt.Println("🧹 Performing clean-slate removal of MySQL/MariaDB...")
	fmt.Println("🔪 Force-killing any running MySQL/MariaDB processes...")
	runCommandQuiet("bash", "-c", "pkill -9 mysqld 2>/dev/null; true")
	runCommandQuiet("bash", "-c", "pkill -9 mariadbd 2>/dev/null; true")
	runCommandQuiet("bash", "-c", "pkill -9 mysql 2>/dev/null; true")
	time.Sleep(1 * time.Second)

	runCommandQuiet("systemctl", "stop", "mysql")
	runCommandQuiet("systemctl", "stop", "mariadb")
	time.Sleep(1 * time.Second)

	fmt.Println("📦 Removing existing packages...")
	purgeCmd := exec.Command("bash", "-c", "apt-get purge -y 'mysql*' 'mariadb*' 2>/dev/null; true")
	purgeCmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	_ = purgeCmd.Run()

	// Remove ALL data and config directories (fresh start) using glob patterns
	cleanupMySQLMariaDBDirectories()

	runCommandQuiet("apt", "clean")
	runCommandQuiet("apt", "autoclean")
	runCommandQuiet("apt", "autoremove", "-y")

	fmt.Println("🔄 Updating package lists...")
	if err := runCommand("apt", "update"); err != nil {
		fmt.Printf("Error updating package list: %v\n", err)
		return
	}

	// Install specific MariaDB version
	fmt.Printf("📦 Installing MariaDB version %s...\n", version)

	// Fix broken dependencies before install
	fmt.Println("🔧 Fixing broken dependencies (before install)...")
	runCommandQuiet("apt", "--fix-broken", "install", "-y")

	packageSpec := fmt.Sprintf("mariadb-server=%s*", version)
	cmd := exec.Command("bash", "-c", fmt.Sprintf("DEBIAN_FRONTEND=noninteractive DEBCONF_NONINTERACTIVE_SEEN=true apt-get install -y --no-install-recommends '%s' 2>&1 | head -200", packageSpec))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			fmt.Printf("⚠️  Install completed with status: %v (this may be normal)\n", err)
		}
	case <-time.After(5 * time.Minute):
		fmt.Println("⚠️  Installation timed out after 5 minutes")
		fmt.Println("   Continuing anyway...")
	}

	// Fix broken dependencies after install
	fmt.Println("🔧 Fixing broken dependencies (after install)...")
	runCommandQuiet("apt", "--fix-broken", "install", "-y")

	time.Sleep(2 * time.Second)

	fmt.Println("🔍 Verifying MariaDB service...")
	if err := runCommand("systemctl", "restart", "mariadb"); err != nil {
		fmt.Printf("⚠️  MariaDB service may not be fully installed: %v\n", err)
		fmt.Println("   Continuing with configuration anyway...")
	}

	configureMariaDB()

	if isServiceActive("mariadb") {
		secureRootUser("mariadb")
	} else {
		fmt.Println("⚠️  MariaDB service is not running. Skipping password setup.")
	}

	if err := runCommand("systemctl", "enable", "mariadb"); err != nil {
		fmt.Printf("Error enabling MariaDB: %v\n", err)
	}

	// Update config to mark MariaDB as installed and configured
	if err := UpdateServerConfig("mariadb", true, 3306, "backend"); err != nil {
		fmt.Printf("⚠️  Warning: Could not update config: %v\n", err)
	}

	fmt.Printf("✅ MariaDB %s installed successfully\n", version)
}

// ==================== UNINSTALL FUNCTIONS ====================

// UninstallAll uninstalls the complete web stack with confirmation
func UninstallAll() {
	fmt.Println("🚨 WebStack Complete Uninstall")
	fmt.Println("==============================")
	fmt.Println("⚠️  This will remove ALL components (Nginx, Apache, databases, PHP versions)")
	fmt.Println("⚠️  Your domain data and SSL certificates will be preserved")

	if !improvedAskYesNo("Are you sure you want to uninstall everything?") {
		fmt.Println("Uninstall cancelled.")
		return
	}

	if !improvedAskYesNo("This action cannot be undone. Continue?") {
		fmt.Println("Uninstall cancelled.")
		return
	}

	fmt.Println("\n🗑️  Uninstalling components...")

	// Uninstall web servers
	UninstallNginx()
	UninstallApache()

	// Uninstall databases
	if improvedAskYesNo("Uninstall MySQL?") {
		UninstallMySQL()
	}
	if improvedAskYesNo("Uninstall MariaDB?") {
		UninstallMariaDB()
	}
	if improvedAskYesNo("Uninstall PostgreSQL?") {
		UninstallPostgreSQL()
	}

	// Uninstall PHP versions
	phpVersions := []string{"5.6", "7.0", "7.1", "7.2", "7.3", "7.4", "8.0", "8.1", "8.2", "8.3", "8.4"}
	for _, version := range phpVersions {
		if checkPHPVersion(version) == Installed {
			if improvedAskYesNo(fmt.Sprintf("Uninstall PHP %s?", version)) {
				UninstallPHP(version)
			}
		}
	}

	fmt.Println("\n✅ Uninstall completed!")
	fmt.Println("📝 Your domain configurations and SSL certificates remain in /etc/webstack/")
}

// UninstallNginx removes Nginx
func UninstallNginx() {
	component := components["nginx"]
	status := checkComponentStatus(component)

	if status != Installed {
		fmt.Println("ℹ️  Nginx is not installed")
		return
	}

	if !improvedAskYesNo("Uninstall Nginx?") {
		fmt.Println("⏭️  Skipping Nginx uninstall")
		return
	}

	if err := uninstallComponent(component); err != nil {
		fmt.Printf("❌ Error uninstalling Nginx: %v\n", err)
		return
	}

	// Clean up nginx includes directory used for modules like phpmyadmin, pgadmin, etc.
	os.RemoveAll("/etc/nginx/includes")

	// Remove firewall rules
	fmt.Println("🔒 Removing firewall rules...")
	webPorts := []int{80, 443}
	for _, port := range webPorts {
		portStr := fmt.Sprintf("%d", port)
		// Remove both IPv4 and IPv6 rules
		runCommand("iptables", "-D", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT")
		runCommand("ip6tables", "-D", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT")
	}
	// Persist rules
	runCommand("bash", "-c", "iptables-save > /etc/iptables/rules.v4 2>/dev/null || true")
	runCommand("bash", "-c", "ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true")

	// Update config
	if err := UpdateServerConfig("nginx", false, 0, ""); err != nil {
		fmt.Printf("⚠️  Warning: Could not update config: %v\n", err)
	}

	fmt.Println("✅ Nginx uninstalled successfully (firewall ports 80/443 closed)")
}

// UninstallApache removes Apache
func UninstallApache() {
	component := components["apache"]
	status := checkComponentStatus(component)

	if status != Installed {
		fmt.Println("ℹ️  Apache is not installed")
		return
	}

	if !improvedAskYesNo("Uninstall Apache?") {
		fmt.Println("⏭️  Skipping Apache uninstall")
		return
	}

	if err := uninstallComponent(component); err != nil {
		fmt.Printf("❌ Error uninstalling Apache: %v\n", err)
		return
	}

	// Clean up apache includes directory used for modules like phpmyadmin, pgadmin, etc.
	os.RemoveAll("/etc/apache2/includes")

	// Remove firewall rules
	fmt.Println("🔒 Removing firewall rules...")
	webPorts := []int{80, 443}
	for _, port := range webPorts {
		portStr := fmt.Sprintf("%d", port)
		// Remove both IPv4 and IPv6 rules
		runCommand("iptables", "-D", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT")
		runCommand("ip6tables", "-D", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT")
	}
	// Persist rules
	runCommand("bash", "-c", "iptables-save > /etc/iptables/rules.v4 2>/dev/null || true")
	runCommand("bash", "-c", "ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true")

	// Update config
	if err := UpdateServerConfig("apache", false, 0, ""); err != nil {
		fmt.Printf("⚠️  Warning: Could not update config: %v\n", err)
	}

	fmt.Println("✅ Apache uninstalled successfully (firewall ports 80/443 closed)")
}

// UninstallMySQL removes MySQL
func UninstallMySQL() {
	component := components["mysql"]
	status := checkComponentStatus(component)

	if status != Installed {
		fmt.Println("ℹ️  MySQL is not installed")
		return
	}

	fmt.Println("⚠️  Uninstalling MySQL will remove the database server")
	if !improvedAskYesNo("Continue uninstalling MySQL?") {
		fmt.Println("⏭️  Skipping MySQL uninstall")
		return
	}

	if err := uninstallComponent(component); err != nil {
		fmt.Printf("❌ Error uninstalling MySQL: %v\n", err)
		return
	}

	fmt.Println("✅ MySQL uninstalled successfully")
}

// UninstallMariaDB removes MariaDB
func UninstallMariaDB() {
	component := components["mariadb"]
	status := checkComponentStatus(component)

	if status != Installed {
		fmt.Println("ℹ️  MariaDB is not installed")
		return
	}

	fmt.Println("⚠️  Uninstalling MariaDB will remove the database server")
	if !improvedAskYesNo("Continue uninstalling MariaDB?") {
		fmt.Println("⏭️  Skipping MariaDB uninstall")
		return
	}

	if err := uninstallComponent(component); err != nil {
		fmt.Printf("❌ Error uninstalling MariaDB: %v\n", err)
		return
	}

	fmt.Println("✅ MariaDB uninstalled successfully")
}

// UninstallPostgreSQL removes PostgreSQL
func UninstallPostgreSQL() {
	component := components["postgresql"]
	status := checkComponentStatus(component)

	if status != Installed {
		fmt.Println("ℹ️  PostgreSQL is not installed")
		return
	}

	fmt.Println("⚠️  Uninstalling PostgreSQL will remove the database server")
	if !improvedAskYesNo("Continue uninstalling PostgreSQL?") {
		fmt.Println("⏭️  Skipping PostgreSQL uninstall")
		return
	}

	if err := uninstallComponent(component); err != nil {
		fmt.Printf("⚠️  Uninstall returned error: %v\n", err)
		// The uninstallComponent handles the cleanup and reboot prompts, so we're good
	}
}

// UninstallPHP removes a specific PHP version
func UninstallPHP(version string) {
	// For uninstall, check if package exists in ANY state (ii, rc, etc.)
	packageName := fmt.Sprintf("php%s-fpm", version)
	cmd := exec.Command("dpkg-query", "-W", "-f=${Status}", packageName)
	output, err := cmd.Output()
	packageExists := err == nil && len(output) >= 1

	if !packageExists {
		fmt.Printf("ℹ️  PHP %s is not installed\n", version)
		return
	}

	if !improvedAskYesNo(fmt.Sprintf("Uninstall PHP %s?", version)) {
		fmt.Printf("⏭️  Skipping PHP %s uninstall\n", version)
		return
	}

	if err := uninstallPHP(version); err != nil {
		fmt.Printf("❌ Error uninstalling PHP %s: %v\n", version, err)
		return
	}

	fmt.Printf("✅ PHP %s uninstalled successfully\n", version)
}

// cleanupMySQLMariaDBDirectories removes all MySQL/MariaDB related directories using glob patterns
func cleanupMySQLMariaDBDirectories() {
	fmt.Println("🗑️  Removing all MySQL/MariaDB directories...")

	// Use bash glob patterns to catch all variants (* wildcards)
	// This ensures we remove /var/lib/mysql, /var/lib/mysql-8.0, /var/lib/mysql-files, etc.
	cleanupPatterns := []string{
		"/var/lib/mysql*", // Catches mysql, mysql-8.0, mysql-files, etc.
		"/var/log/mysql*", // Catches mysql, mysql-files logs, etc.
		"/etc/mysql*",     // Catches mysql, mysqlrouter configs, etc.
		"/run/mysqld*",    // Catches mysqld, mysqld_safe, etc.
		"/run/mariadb*",   // Catches mariadb, mariadb-init, etc.
	}

	for _, pattern := range cleanupPatterns {
		// Use bash glob expansion to handle wildcards properly
		runCommandQuiet("bash", "-c", fmt.Sprintf("rm -rf %s 2>/dev/null || true", pattern))
	}
}

// Helper function at line around where runCommand is defined

func runCommandQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func configureNginx() {
	fmt.Println("⚙️  Configuring Nginx...")

	// Read template from embedded filesystem
	content, err := templates.GetNginxTemplate("nginx.conf")
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not read nginx template: %v\n", err)
		return
	}

	// Ensure cache directory exists
	if err := os.MkdirAll("/var/cache/nginx/fastcgi", 0755); err != nil {
		fmt.Printf("⚠️  Warning: Could not create nginx cache directory: %v\n", err)
	}

	// Create nginx includes directory for modules like phpmyadmin, pgadmin, etc.
	if err := os.MkdirAll("/etc/nginx/includes", 0755); err != nil {
		fmt.Printf("⚠️  Warning: Could not create nginx includes directory: %v\n", err)
	}

	// Create WebStack welcome directory
	if err := os.MkdirAll("/var/www/webstack", 0755); err != nil {
		fmt.Printf("⚠️  Warning: Could not create webstack welcome directory: %v\n", err)
	}

	// Deploy welcome page
	if welcomeContent, err := templates.GetNginxTemplate("welcome.html"); err == nil {
		if err := ioutil.WriteFile("/var/www/webstack/welcome.html", welcomeContent, 0644); err != nil {
			fmt.Printf("⚠️  Warning: Could not write welcome page: %v\n", err)
		} else {
			fmt.Println("✅ Welcome page deployed")
		}
	}

	// Deploy default server config
	if defaultConfig, err := templates.GetNginxTemplate("default.conf"); err == nil {
		if err := os.MkdirAll("/etc/nginx/sites-available", 0755); err == nil {
			if err := ioutil.WriteFile("/etc/nginx/sites-available/default", defaultConfig, 0644); err == nil {
				// Create symlink in sites-enabled
				os.Remove("/etc/nginx/sites-enabled/default")
				os.Symlink("/etc/nginx/sites-available/default", "/etc/nginx/sites-enabled/default")
				fmt.Println("✅ Default server block deployed")
			}
		}
	}

	// Deploy error pages to /etc/webstack/error/
	os.MkdirAll("/etc/webstack/error", 0755)
	errorPages := []string{"403.html", "404.html", "50x.html"}
	for _, page := range errorPages {
		if content, err := templates.GetErrorTemplate(page); err == nil {
			if err := ioutil.WriteFile("/etc/webstack/error/"+page, content, 0644); err == nil {
				// Silently succeed
			}
		}
	}
	fmt.Println("✅ Error pages deployed to /etc/webstack/error/")

	// Generate unified DH parameters for SSL/TLS
	dhparamPath := "/etc/ssl/dhparam.pem"

	// Check if openssl is available
	if err := exec.Command("which", "openssl").Run(); err != nil {
		fmt.Println("⚠️  Warning: OpenSSL not found, skipping DH parameter generation")
		fmt.Println("   Install it later with: sudo apt install -y openssl")
	} else if _, err := os.Stat(dhparamPath); os.IsNotExist(err) {
		fmt.Println("🔐 Generating SSL DH parameters (this may take a minute)...")

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
		fmt.Println("✓ SSL DH parameters already exist at " + dhparamPath)
	}

	// Write to /etc/nginx/nginx.conf
	if err := ioutil.WriteFile("/etc/nginx/nginx.conf", content, 0644); err != nil {
		fmt.Printf("⚠️  Warning: Could not write nginx configuration: %v\n", err)
		return
	}

	fmt.Println("✅ Nginx configuration applied")
}

func configureApache() {
	fmt.Println("⚙️  Configuring Apache...")

	// Enable required Apache modules
	requiredModules := []string{
		"rewrite",
		"headers",
		"proxy",
		"proxy_http",
		"ssl",
		"php-fpm",
	}

	for _, module := range requiredModules {
		if err := runCommandQuiet("a2enmod", module); err != nil {
			// Some modules might not exist depending on Apache version
			// Continue anyway as they're optional
		}
	}
	fmt.Println("✅ Apache modules enabled")

	// Create apache includes directory for modules like phpmyadmin, pgadmin, etc.
	if err := os.MkdirAll("/etc/apache2/includes", 0755); err != nil {
		fmt.Printf("⚠️  Warning: Could not create apache includes directory: %v\n", err)
	}

	// Determine Apache port based on whether Nginx is installed
	apachePort, apacheMode := determineApachePort()

	// Generate ports.conf dynamically based on Apache port
	portConfContent := fmt.Sprintf(`# WebStack CLI - Apache Ports Configuration
# Apache listens on port %d

Listen %d

<IfModule ssl_module>
    Listen %d ssl
</IfModule>

<IfModule mod_gnutls.c>
    Listen %d ssl
</IfModule>
`, apachePort, apachePort, apachePort+363, apachePort+363)

	if err := ioutil.WriteFile("/etc/apache2/ports.conf", []byte(portConfContent), 0644); err != nil {
		fmt.Printf("⚠️  Warning: Could not write /etc/apache2/ports.conf: %v\n", err)
	} else {
		fmt.Printf("✅ Updated /etc/apache2/ports.conf (port %d, mode: %s)\n", apachePort, apacheMode)
	}

	// Optionally update apache2.conf if template exists
	if data, err := templates.GetApacheTemplate("apache2.conf"); err == nil {
		if err := ioutil.WriteFile("/etc/apache2/apache2.conf", data, 0644); err != nil {
			fmt.Printf("⚠️  Warning: Could not write /etc/apache2/apache2.conf: %v\n", err)
		} else {
			fmt.Println("✅ Updated /etc/apache2/apache2.conf from template")
		}
	}

	// Ensure webstack welcome directory exists
	if err := os.MkdirAll("/var/www/webstack", 0755); err != nil {
		fmt.Printf("⚠️  Warning: Could not create webstack welcome directory: %v\n", err)
	}

	// Deploy welcome page to Apache webstack folder
	if welcomeContent, err := templates.GetNginxTemplate("welcome.html"); err == nil {
		if err := ioutil.WriteFile("/var/www/webstack/welcome.html", welcomeContent, 0644); err != nil {
			fmt.Printf("⚠️  Warning: Could not write welcome page: %v\n", err)
		} else {
			fmt.Println("✅ Welcome page deployed")
		}
	}

	// Deploy default Apache VirtualHost config with dynamic port
	if defaultConfig, err := templates.GetApacheTemplate("default.conf"); err == nil {
		// Parse and render the template with Apache port
		tmpl, err := template.New("apache-default").Parse(string(defaultConfig))
		if err == nil {
			var buf strings.Builder
			tmpl.Execute(&buf, map[string]interface{}{
				"ApachePort": apachePort,
			})

			if err := os.MkdirAll("/etc/apache2/sites-available", 0755); err == nil {
				if err := ioutil.WriteFile("/etc/apache2/sites-available/000-default.conf", []byte(buf.String()), 0644); err == nil {
					// Enable the default site
					runCommandQuiet("a2ensite", "000-default.conf")
					fmt.Println("✅ Default VirtualHost deployed")
				}
			}
		}
	}
}

func configureMySQL() bool {
	fmt.Println("⚙️  Configuring MySQL...")

	// Read MySQL configuration template from embedded filesystem
	configData, err := templates.GetMySQLTemplate("my.cnf")
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not read MySQL config template: %v\n", err)
		fmt.Println("   Using system defaults")
		return false
	}

	// Write configuration to MySQL config directory
	destPath := "/etc/mysql/mysql.conf.d/99-webstack.cnf"
	if err := ioutil.WriteFile(destPath, configData, 0644); err != nil {
		fmt.Printf("⚠️  Warning: Could not write MySQL config: %v\n", err)
		return false
	}

	fmt.Printf("✓ MySQL configuration written to %s\n", destPath)

	// Create required MySQL directories and set permissions
	requiredDirs := []struct {
		path string
		mode os.FileMode
		user string
	}{
		{"/var/log/mysql", 0755, "mysql"},
		{"/var/lib/mysql-files", 0770, "mysql"},
	}

	for _, dir := range requiredDirs {
		if err := os.MkdirAll(dir.path, dir.mode); err != nil {
			fmt.Printf("⚠️  Warning: Could not create %s: %v\n", dir.path, err)
			return false
		} else {
			// Change ownership to mysql user
			runCommandQuiet("chown", "mysql:mysql", dir.path)
			runCommandQuiet("chmod", fmt.Sprintf("%o", dir.mode), dir.path)
		}
	}

	// Restart MySQL to apply configuration
	if err := runCommand("systemctl", "restart", "mysql"); err != nil {
		fmt.Printf("⚠️  Warning: Could not restart MySQL: %v\n", err)
		fmt.Println("   Run 'sudo systemctl restart mysql' manually to apply configuration")
		return false
	}

	fmt.Println("✓ MySQL restarted with new configuration")
	return true
}

func configureMariaDB() bool {
	fmt.Println("⚙️  Configuring MariaDB...")

	// Read MySQL configuration template from embedded filesystem (works for MariaDB too)
	configData, err := templates.GetMySQLTemplate("my.cnf")
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not read MariaDB config template: %v\n", err)
		fmt.Println("   Using system defaults")
		return false
	}

	// Write configuration to MariaDB config directory
	destPath := "/etc/mysql/mariadb.conf.d/99-webstack.cnf"
	if err := ioutil.WriteFile(destPath, configData, 0644); err != nil {
		fmt.Printf("⚠️  Warning: Could not write MariaDB config: %v\n", err)
		return false
	}

	fmt.Printf("✓ MariaDB configuration written to %s\n", destPath)

	// Create required MariaDB directories and set permissions
	requiredDirs := []struct {
		path string
		mode os.FileMode
		user string
	}{
		{"/var/log/mysql", 0755, "mysql"},
		{"/var/lib/mysql-files", 0770, "mysql"},
	}

	for _, dir := range requiredDirs {
		if err := os.MkdirAll(dir.path, dir.mode); err != nil {
			fmt.Printf("⚠️  Warning: Could not create %s: %v\n", dir.path, err)
			return false
		} else {
			// Change ownership to mysql user
			runCommandQuiet("chown", "mysql:mysql", dir.path)
			runCommandQuiet("chmod", fmt.Sprintf("%o", dir.mode), dir.path)
		}
	}

	// Restart MariaDB to apply configuration
	if err := runCommand("systemctl", "restart", "mariadb"); err != nil {
		fmt.Printf("⚠️  Warning: Could not restart MariaDB: %v\n", err)
		fmt.Println("   Run 'sudo systemctl restart mariadb' manually to apply configuration")
		return false
	}

	fmt.Println("✓ MariaDB restarted with new configuration")
	return true
}

func configurePostgreSQL() {
	fmt.Println("⚙️  Configuring PostgreSQL...")

	fmt.Println("🔐 Securing database postgres user...")

	// Ask user if they want to set a password or auto-generate one
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter password for postgres user (press Enter for auto-generated password): ")

	userInput, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		return
	}

	userInput = strings.TrimSpace(userInput)
	var postgresPassword string

	if userInput == "" {
		// Auto-generate password
		postgresPassword = generateRandomPassword(24)
		fmt.Println("✓ Auto-generated password will be used")
	} else {
		postgresPassword = userInput
		fmt.Println("✓ Password set")
	}

	// Set password for postgres user using sudo
	// PostgreSQL stores the password encrypted, so we use psql to set it
	sqlCommand := fmt.Sprintf("ALTER USER postgres WITH PASSWORD '%s';", postgresPassword)
	cmd := exec.Command("sudo", "-u", "postgres", "psql", "-c", sqlCommand)
	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️  Warning: Could not set postgres password: %v\n", err)
		fmt.Println("   You can manually set it with: sudo -u postgres psql -c \"ALTER USER postgres WITH PASSWORD 'newpassword';\"")
		return
	}

	// Save credentials to secure file
	os.MkdirAll("/etc/webstack", 0755)
	credsPath := "/etc/webstack/postgresql-root-credentials.txt"
	creds := fmt.Sprintf(`PostgreSQL Superuser Credentials
================================
User: postgres
Host: localhost
Password: %s

Location: %s
Permissions: 600 (readable by root only)

How to use:
  psql -U postgres -h localhost
  (enter password when prompted)

Or with password in connection string:
  psql -U postgres -h localhost -W

Connection String:
  postgresql://postgres:%s@localhost:5432/postgres

Security Notes:
- This file is only readable by root
- PostgreSQL user 'postgres' is the superuser account
- Do not share this password
- Consider using peer authentication for local connections
`, postgresPassword, credsPath, postgresPassword)

	if err := os.WriteFile(credsPath, []byte(creds), 0600); err != nil {
		fmt.Printf("⚠️  Warning: Could not save credentials: %v\n", err)
	} else {
		fmt.Printf("✅ Credentials saved to %s (readable by root only)\n", credsPath)
	}
}

func configurePHP(version string) {
	fmt.Printf("⚙️  Configuring PHP %s FPM pool...\n", version)

	// Remove the default www.conf to avoid socket conflicts
	defaultPoolPath := fmt.Sprintf("/etc/php/%s/fpm/pool.d/www.conf", version)
	// Use rm command to ensure it works with proper privileges
	runCommandQuiet("rm", "-f", defaultPoolPath)
	fmt.Println("✓ Default www pool configuration removed")

	// Read PHP-FPM pool template from embedded filesystem
	poolData, err := templates.GetPHPTemplate("pool.conf")
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not read PHP-FPM pool template: %v\n", err)
		fmt.Println("   Using system defaults")
		return
	}

	// Process template with PHP version
	tmpl, err := template.New("pool").Parse(string(poolData))
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not parse PHP-FPM pool template: %v\n", err)
		return
	}

	type PoolData struct {
		PHPVersion string
		PoolName   string
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, PoolData{
		PHPVersion: version,
		PoolName:   "webstack",
	}); err != nil {
		fmt.Printf("⚠️  Warning: Could not render PHP-FPM pool template: %v\n", err)
		return
	}

	// Write configuration to PHP-FPM pool directory
	destDir := fmt.Sprintf("/etc/php/%s/fpm/pool.d", version)
	destPath := filepath.Join(destDir, "webstack.conf")

	if err := os.MkdirAll(destDir, 0755); err != nil {
		fmt.Printf("⚠️  Warning: Could not create %s: %v\n", destDir, err)
		return
	}

	if err := ioutil.WriteFile(destPath, buf.Bytes(), 0644); err != nil {
		fmt.Printf("⚠️  Warning: Could not write PHP-FPM pool config: %v\n", err)
		return
	}

	fmt.Printf("✓ PHP %s FPM pool configuration written to %s\n", version, destPath)
}

// isServiceActive checks if a systemd service is running
func isServiceActive(serviceName string) bool {
	cmd := exec.Command("systemctl", "is-active", serviceName)
	err := cmd.Run()
	return err == nil
}

// generateRandomPassword generates a random password of specified length
func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	password := make([]byte, length)
	rand.Seed(time.Now().UnixNano())
	for i := range password {
		password[i] = charset[rand.Intn(len(charset))]
	}
	return string(password)
}

// executeSQLAsRoot executes SQL commands as the mysql system user (for initial setup without password)
func executeSQLAsRoot(sqlCommands string) error {
	// Connect to MySQL as root user via Unix socket authentication
	// We pipe the SQL to stdin to avoid shell escaping issues with passwords
	cmd := exec.Command("mysql", "-u", "root")
	cmd.Stdin = strings.NewReader(sqlCommands)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Debug: MySQL execution output: %s\n", string(output))
		return fmt.Errorf("failed to execute SQL: %v", err)
	}

	return nil
}

// secureRootUser sets a secure password for MySQL/MariaDB root user
func secureRootUser(dbType string) {
	fmt.Println("🔐 Securing database root user...")

	// Ask user if they want to set a password or auto-generate one
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter password for root user (press Enter for auto-generated password): ")

	userInput, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		return
	}

	userInput = strings.TrimSpace(userInput)
	var rootPassword string

	if userInput == "" {
		// Auto-generate password
		rootPassword = generateRandomPassword(24)
		fmt.Println("✓ Auto-generated password will be used")
	} else {
		rootPassword = userInput
		fmt.Println("✓ Password set")
	}

	// SQL commands to set root password
	sqlCommands := fmt.Sprintf(`
ALTER USER 'root'@'localhost' IDENTIFIED BY '%s';
FLUSH PRIVILEGES;
`, rootPassword)

	// Execute SQL with sudo for initial setup
	if err := executeSQLAsRoot(sqlCommands); err != nil {
		fmt.Printf("⚠️  Warning: Could not set root password: %v\n", err)
		fmt.Println("   You can manually secure root with: sudo mysql -u root")
		return
	}

	// Save credentials to secure file
	os.MkdirAll("/etc/webstack", 0755)
	credsPath := fmt.Sprintf("/etc/webstack/%s-root-credentials.txt", dbType)
	creds := fmt.Sprintf(`%s Root User Credentials
================================
User: root
Host: localhost
Password: %s

Location: /etc/webstack/%s-root-credentials.txt
Permissions: 600 (readable by root only)

How to use:
  sudo mysql -u root -p
  Then enter the password above

Security Notes:
- Keep this file secure on the server
- Do not commit to version control
- Rotate password regularly
`, strings.ToUpper(dbType), rootPassword, dbType)

	if err := ioutil.WriteFile(credsPath, []byte(creds), 0600); err != nil {
		fmt.Printf("Warning: Could not save credentials file: %v\n", err)
	} else {
		fmt.Printf("✓ %s root credentials saved to %s (mode 600)\n", strings.ToUpper(dbType), credsPath)
	}

	// Also save password to config file for CLI access
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Warning: Could not load config: %v\n", err)
		return
	}

	// Store password in config defaults
	configKey := fmt.Sprintf("%s_root_password", dbType)
	cfg.SetDefault(configKey, rootPassword)

	if err := cfg.Save(); err != nil {
		fmt.Printf("Warning: Could not save password to config: %v\n", err)
	} else {
		fmt.Printf("✓ Password saved to config at key '%s'\n", configKey)
	}
}

// isPackageInstalled checks if a package is installed on the system
func isPackageInstalled(packageName string) bool {
	cmd := exec.Command("dpkg", "-l", packageName)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	// Check if output contains "ii" (installed) status
	return strings.Contains(string(output), "ii  "+packageName)
}

// determineApachePort checks if Nginx is installed and assigns appropriate port
func determineApachePort() (int, string) {
	if isPackageInstalled("nginx") {
		// Nginx is installed, so Apache becomes backend on port 8080
		return 8080, "backend"
	}
	// Nginx not installed, Apache runs standalone on port 80
	return 80, "standalone"
}

// determineNginxMode checks if Apache is installed and determines Nginx mode
func determineNginxMode() (string, int) {
	if isPackageInstalled("apache2") {
		// Apache is installed, Nginx becomes proxy on port 80
		return "proxy", 80
	}
	// Apache not installed, Nginx runs direct on port 80
	return "standalone", 80
}

// LoadOrCreateConfig loads config from file, or creates new one if missing
func LoadOrCreateConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return cfg, nil
}

// UpdateServerConfig updates a server's configuration in the config file
func UpdateServerConfig(serverName string, installed bool, port int, mode string) error {
	cfg, err := LoadOrCreateConfig()
	if err != nil {
		return err
	}

	srv := config.ServerConfig{
		Installed: installed,
		Port:      port,
		Mode:      mode,
	}
	cfg.SetServer(serverName, srv)

	return cfg.Save()
}

// ComponentStatusSummary is a small struct returned to CLI status/menu
type ComponentStatusSummary struct {
	ConfigInstalled bool
	DpkgInstalled   bool
	ServiceRunning  bool
}

// GetComponentsStatus returns status info for all known components
func GetComponentsStatus() map[string]ComponentStatusSummary {
	results := make(map[string]ComponentStatusSummary)
	cfg, _ := LoadOrCreateConfig()

	for name, comp := range components {
		cfgInstalled := false
		if cfg != nil {
			cfgInstalled = cfg.IsInstalled(name)
		}

		dpkgInstalled := false
		// component.CheckCmd uses dpkg -l; reuse isPackageInstalled when possible
		if len(comp.CheckCmd) == 3 && comp.CheckCmd[0] == "dpkg" && comp.CheckCmd[1] == "-l" {
			pkg := comp.CheckCmd[2]
			dpkgInstalled = isPackageInstalled(pkg)
		} else {
			// fallback: try running check command
			cmd := exec.Command(comp.CheckCmd[0], comp.CheckCmd[1:]...)
			if err := cmd.Run(); err == nil {
				dpkgInstalled = true
			}
		}

		running := false
		if comp.ServiceName != "" {
			running = isServiceActive(comp.ServiceName)
		}

		results[name] = ComponentStatusSummary{
			ConfigInstalled: cfgInstalled,
			DpkgInstalled:   dpkgInstalled,
			ServiceRunning:  running,
		}
	}

	return results
}

// GetPHPVersionsStatus returns status for all PHP versions
func GetPHPVersionsStatus() map[string]ComponentStatusSummary {
	results := make(map[string]ComponentStatusSummary)

	// Common PHP versions
	phpVersions := []string{"5.6", "7.0", "7.1", "7.2", "7.3", "7.4", "8.0", "8.1", "8.2", "8.3", "8.4"}

	for _, version := range phpVersions {
		packageName := fmt.Sprintf("php%s-fpm", version)
		installed := isPackageInstalled(packageName)

		serviceName := fmt.Sprintf("php%s-fpm", version)
		running := false
		if installed {
			running = isServiceActive(serviceName)
		}

		results[fmt.Sprintf("php%s", version)] = ComponentStatusSummary{
			DpkgInstalled:  installed,
			ServiceRunning: running,
		}
	}

	return results
}

// ==================== MAIL SERVER FUNCTIONS ====================

// InstallMailStack installs complete mail server with optional security features
func InstallMailStack() {
	fmt.Println("📧 Mail Server Installation")
	fmt.Println("===========================")

	if err := runCommand("apt", "update"); err != nil {
		fmt.Printf("Error updating package list: %v\n", err)
		return
	}

	// Install Postfix
	installPostfixInternal()

	// Install Dovecot
	installDovecotInternal()

	// Ask about optional security features
	fmt.Println("")
	fmt.Println("📋 Optional Security Features")

	if improvedAskYesNo("Install ClamAV antivirus scanner?") {
		installClamAVInternal()
	}

	if improvedAskYesNo("Install SpamAssassin spam filter?") {
		installSpamAssassinInternal()
	}

	fmt.Println("")

	// Configure firewall for mail ports if a firewall tool is present
	AddMailFirewallRules()

	fmt.Println("✅ Mail server stack installation completed!")
	fmt.Println("💡 Configure mail accounts and domains as needed")
}

// installPostfixInternal is the internal Postfix installation
func installPostfixInternal() {
	fmt.Println("📦 Installing Postfix mail server...")

	// Check if already installed
	if isPackageInstalled("postfix") {
		fmt.Println("ℹ️  Postfix is already installed")
		if !improvedAskYesNo("Reconfigure Postfix?") {
			return
		}
	}

	// Install Postfix without interactive prompts
	cmd := exec.Command("bash", "-c", "DEBIAN_FRONTEND=noninteractive apt-get install -y postfix")
	cmd.Env = append(os.Environ(),
		"DEBIAN_FRONTEND=noninteractive",
		"DEBCONF_NONINTERACTIVE_SEEN=true",
		"postfix/main_mailer_type=string Internet Site",
		"postfix/mailname=string localhost")

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error installing Postfix: %v\n", err)
		return
	}

	// Configure Postfix
	configurePostfix()

	if err := runCommand("systemctl", "enable", "postfix"); err != nil {
		fmt.Printf("Error enabling Postfix: %v\n", err)
	}

	if err := runCommand("systemctl", "restart", "postfix"); err != nil {
		fmt.Printf("Error restarting Postfix: %v\n", err)
	}

	fmt.Println("✅ Postfix installed successfully")
}

// installDovecotInternal is the internal Dovecot installation
func installDovecotInternal() {
	fmt.Println("📦 Installing Dovecot mail server...")

	// Check if already installed
	if isPackageInstalled("dovecot-core") {
		fmt.Println("ℹ️  Dovecot is already installed")
		if !improvedAskYesNo("Reconfigure Dovecot?") {
			return
		}
	}

	// Install Dovecot packages
	dovecotPackages := []string{
		"dovecot-core",
		"dovecot-imapd",
		"dovecot-pop3d",
		"dovecot-lmtpd",
		"dovecot-mysql",
	}

	args := append([]string{"install", "-y"}, dovecotPackages...)
	if err := runCommand("apt", args...); err != nil {
		fmt.Printf("Error installing Dovecot: %v\n", err)
		return
	}

	// Configure Dovecot
	configureDovecot()

	if err := runCommand("systemctl", "enable", "dovecot"); err != nil {
		fmt.Printf("Error enabling Dovecot: %v\n", err)
	}

	if err := runCommand("systemctl", "restart", "dovecot"); err != nil {
		fmt.Printf("Error restarting Dovecot: %v\n", err)
	}

	fmt.Println("✅ Dovecot installed successfully")
}

// installClamAVInternal is the internal ClamAV installation
func installClamAVInternal() {
	fmt.Println("📦 Installing ClamAV antivirus scanner...")

	// Install ClamAV and Amavis
	clamavPackages := []string{
		"clamav",
		"clamav-daemon",
		"amavis",
		"amavisd-new",
	}

	args := append([]string{"install", "-y"}, clamavPackages...)
	if err := runCommand("apt", args...); err != nil {
		fmt.Printf("Error installing ClamAV: %v\n", err)
		return
	}

	// Update virus definitions
	fmt.Println("🔄 Updating virus definitions (this may take a while)...")
	runCommand("freshclam")

	if err := runCommand("systemctl", "enable", "clamav-daemon"); err != nil {
		fmt.Printf("Error enabling ClamAV daemon: %v\n", err)
	}

	if err := runCommand("systemctl", "restart", "clamav-daemon"); err != nil {
		fmt.Printf("Error restarting ClamAV daemon: %v\n", err)
	}

	fmt.Println("✅ ClamAV installed successfully")
}

// installSpamAssassinInternal is the internal SpamAssassin installation
func installSpamAssassinInternal() {
	fmt.Println("📦 Installing SpamAssassin spam filter...")

	// Install SpamAssassin
	spamassassinPackages := []string{
		"spamassassin",
		"spamc",
	}

	args := append([]string{"install", "-y"}, spamassassinPackages...)
	if err := runCommand("apt", args...); err != nil {
		fmt.Printf("Error installing SpamAssassin: %v\n", err)
		return
	}

	// Configure SpamAssassin
	configureSpamAssassin()

	// Try to enable and start spamd service (may not exist on all systems)
	// SpamAssassin typically runs as a daemon via spamd
	runCommandQuiet("systemctl", "enable", "spamd")
	runCommandQuiet("systemctl", "restart", "spamd")

	fmt.Println("✅ SpamAssassin installed successfully")
	fmt.Println("💡 SpamAssassin is configured and ready for integration with mail server")
}

// Old individual functions kept for backward compatibility (deprecated)
// InstallPostfix installs and configures Postfix mail server (deprecated - use InstallMailStack)
func InstallPostfix() {
	installPostfixInternal()
}

// InstallDovecot installs and configures Dovecot mail server (deprecated - use InstallMailStack)
func InstallDovecot() {
	installDovecotInternal()
}

// InstallClamAV installs and configures ClamAV antivirus scanner (deprecated - use InstallMailStack)
func InstallClamAV() {
	installClamAVInternal()
}

// InstallSpamAssassin installs and configures SpamAssassin spam filter (deprecated - use InstallMailStack)
func InstallSpamAssassin() {
	installSpamAssassinInternal()
}

// UninstallMailStack uninstalls complete mail server stack
func UninstallMailStack() {
	fmt.Println("🗑️  Mail Server Uninstall")
	fmt.Println("========================")

	if !improvedAskYesNo("Uninstall complete mail server stack (Postfix, Dovecot, and optional security features)?") {
		fmt.Println("⏭️  Skipping mail server uninstall")
		return
	}

	if !improvedAskYesNo("This action cannot be undone. Continue?") {
		fmt.Println("⏭️  Uninstall cancelled")
		return
	}

	// Uninstall components
	uninstallSpamAssassinInternal()
	uninstallClamAVInternal()
	uninstallDovecotInternal()
	uninstallPostfixInternal()

	fmt.Println("✅ Mail server stack uninstalled successfully")
}

// uninstallPostfixInternal removes Postfix
func uninstallPostfixInternal() {
	if !isPackageInstalled("postfix") {
		fmt.Println("ℹ️  Postfix is not installed")
		return
	}

	fmt.Println("🗑️  Removing Postfix...")
	runCommand("systemctl", "stop", "postfix")
	runCommand("systemctl", "disable", "postfix")
	runCommand("apt", "purge", "-y", "postfix")
	fmt.Println("✓ Postfix removed")
}

// uninstallDovecotInternal removes Dovecot
func uninstallDovecotInternal() {
	if !isPackageInstalled("dovecot-core") {
		fmt.Println("ℹ️  Dovecot is not installed")
		return
	}

	fmt.Println("🗑️  Removing Dovecot...")
	runCommand("systemctl", "stop", "dovecot")
	runCommand("systemctl", "disable", "dovecot")
	runCommand("apt", "purge", "-y", "dovecot*")
	fmt.Println("✓ Dovecot removed")
}

// uninstallClamAVInternal removes ClamAV
func uninstallClamAVInternal() {
	if !isPackageInstalled("clamav-daemon") {
		return
	}

	fmt.Println("🗑️  Removing ClamAV...")
	runCommand("systemctl", "stop", "clamav-daemon")
	runCommand("systemctl", "disable", "clamav-daemon")
	runCommand("apt", "purge", "-y", "clamav*", "amavis*")
	fmt.Println("✓ ClamAV removed")
}

// uninstallSpamAssassinInternal removes SpamAssassin
func uninstallSpamAssassinInternal() {
	if !isPackageInstalled("spamassassin") {
		return
	}

	fmt.Println("🗑️  Removing SpamAssassin...")
	runCommandQuiet("systemctl", "stop", "spamd")
	runCommandQuiet("systemctl", "disable", "spamd")
	runCommand("apt", "purge", "-y", "spamassassin", "spamc")
	fmt.Println("✓ SpamAssassin removed")
}

// Deprecated uninstall functions - kept for backward compatibility
// UninstallPostfix removes Postfix (deprecated - use UninstallMailStack)
func UninstallPostfix() {
	uninstallPostfixInternal()
}

// UninstallDovecot removes Dovecot (deprecated - use UninstallMailStack)
func UninstallDovecot() {
	uninstallDovecotInternal()
}

// UninstallClamAV removes ClamAV (deprecated - use UninstallMailStack)
func UninstallClamAV() {
	uninstallClamAVInternal()
}

// UninstallSpamAssassin removes SpamAssassin (deprecated - use UninstallMailStack)
func UninstallSpamAssassin() {
	uninstallSpamAssassinInternal()
}

// Helper functions for mail configuration

func configurePostfix() {
	// Only configure if Postfix is installed
	if !isPackageInstalled("postfix") {
		return
	}

	fmt.Println("⚙️  Configuring Postfix...")

	// Ensure postfix dkim and dns-records directories exist
	os.MkdirAll("/etc/postfix/dkim", 0755)
	os.MkdirAll("/etc/postfix/dns-records", 0755)
	os.MkdirAll("/etc/postfix", 0755)
	runCommandQuiet("chown", "-R", "postfix:postfix", "/etc/postfix/dkim")
	runCommandQuiet("chown", "-R", "postfix:postfix", "/etc/postfix/dns-records")

	// Create empty vdomains and vmailbox files if they don't exist
	vdomainsFile := "/etc/postfix/vdomains"
	vmailboxFile := "/etc/postfix/vmailbox"

	if _, err := os.Stat(vdomainsFile); os.IsNotExist(err) {
		ioutil.WriteFile(vdomainsFile, []byte(""), 0644)
		runCommandQuiet("postmap", vdomainsFile)
	}

	if _, err := os.Stat(vmailboxFile); os.IsNotExist(err) {
		ioutil.WriteFile(vmailboxFile, []byte(""), 0644)
		runCommandQuiet("postmap", vmailboxFile)
	}

	// Check if we have Dovecot for SASL and LMTP delivery
	hasDovecot := isPackageInstalled("dovecot-core")

	// Configure Postfix using postconf for proper settings
	configCmds := [][]string{
		{"postconf", "-e", "virtual_mailbox_base=/var/mail/vhosts"},
		{"postconf", "-e", "virtual_mailbox_maps=hash:/etc/postfix/vmailbox"},
		{"postconf", "-e", "virtual_mailbox_domains=hash:/etc/postfix/vdomains"},
		{"postconf", "-e", "virtual_minimum_uid=1"},
		{"postconf", "-e", "mailbox_size_limit=0"},
		{"postconf", "-e", "recipient_delimiter=+"},
		{"postconf", "-e", "inet_interfaces=all"},
		{"postconf", "-e", "inet_protocols=all"},
	}

	// If Dovecot is available, use LMTP for delivery and SASL for authentication
	if hasDovecot {
		configCmds = append(configCmds, [][]string{
			{"postconf", "-e", "virtual_transport=lmtp:unix:private/dovecot-lmtp"},
			{"postconf", "-e", "smtpd_sasl_auth_enable=yes"},
			{"postconf", "-e", "smtpd_sasl_type=dovecot"},
			{"postconf", "-e", "smtpd_sasl_path=private/auth"},
			{"postconf", "-e", "smtpd_recipient_restrictions=permit_mynetworks,permit_sasl_authenticated,reject_unauth_destination"},
		}...)
	} else {
		// Fallback to standard virtual delivery (mail user must exist)
		configCmds = append(configCmds, [][]string{
			{"postconf", "-e", "virtual_uid_maps=static:8"},
			{"postconf", "-e", "virtual_gid_maps=static:8"},
		}...)
	}

	// Execute all postconf commands
	for _, cmd := range configCmds {
		runCommandQuiet(cmd[0], cmd[1:]...)
	}

	// Add submission port to master.cf if not already present
	masterCfPath := "/etc/postfix/master.cf"
	if masterContent, err := ioutil.ReadFile(masterCfPath); err == nil {
		masterStr := string(masterContent)
		if !strings.Contains(masterStr, "submission inet") {
			// Add submission port for authenticated SMTP
			submissionConfig := `submission inet n - y - - smtpd
  -o syslog_name=postfix/submission
  -o smtpd_tls_security_level=may
  -o smtpd_recipient_restrictions=permit_mynetworks,permit_sasl_authenticated,reject_unauth_destination
  -o smtpd_relay_restrictions=permit_sasl_authenticated,reject
  -o smtpd_sasl_auth_enable=yes
  -o smtpd_sasl_type=dovecot
  -o smtpd_sasl_path=private/auth
`
			masterStr += "\n" + submissionConfig
			if err := ioutil.WriteFile(masterCfPath, []byte(masterStr), 0644); err != nil {
				fmt.Printf("⚠️  Warning: Could not update master.cf: %v\n", err)
			}
		}
	}

	// Reload Postfix configuration
	runCommandQuiet("postfix", "reload")
}

func configureDovecot() {
	// Only configure if Dovecot is installed
	if !isPackageInstalled("dovecot-core") {
		return
	}

	fmt.Println("⚙️  Configuring Dovecot for virtual mail...")

	// Ensure dovecot config directory exists
	os.MkdirAll("/etc/dovecot/conf.d", 0755)
	os.MkdirAll("/var/mail/vhosts", 0755)
	os.MkdirAll("/etc/dovecot", 0755)

	// Create/update users file if it doesn't exist
	usersPath := "/etc/dovecot/users"
	if _, err := os.Stat(usersPath); os.IsNotExist(err) {
		ioutil.WriteFile(usersPath, []byte(""), 0644)
	}
	os.Chmod(usersPath, 0644)

	// Disable system authentication (PAM, passwd) - use only passwd-file for virtual mail
	systemAuthConfig := `# WebStack CLI - Disable system auth
# Use only passwd-file for virtual mail authentication
# This prevents PAM authentication from interfering with virtual mail
`
	ioutil.WriteFile("/etc/dovecot/conf.d/10-auth-disable-system.conf", []byte(systemAuthConfig), 0644)

	// Update auth-passwdfile.conf.ext to use PLAIN scheme and point to /etc/dovecot/users
	passwdFileConfig := `# WebStack CLI - passwd-file configuration for virtual mail
# Stores virtual user credentials in /etc/dovecot/users
# Format: email:{PLAIN}password:uid:gid::homedir::
passdb {
  driver = passwd-file
  args = scheme=PLAIN /etc/dovecot/users
}

userdb {
  driver = passwd-file
  args = /etc/dovecot/users
}
`
	ioutil.WriteFile("/etc/dovecot/conf.d/auth-passwdfile.conf.ext", []byte(passwdFileConfig), 0644)

	// Disable system auth includes in main auth config
	authConfPath := "/etc/dovecot/conf.d/10-auth.conf"
	if authContent, err := ioutil.ReadFile(authConfPath); err == nil {
		authStr := string(authContent)

		// Comment out auth-system.conf.ext if enabled
		if strings.Contains(authStr, "!include auth-system.conf.ext") {
			authStr = strings.ReplaceAll(authStr, "!include auth-system.conf.ext", "#!include auth-system.conf.ext")
		}

		// Enable auth-passwdfile.conf.ext if disabled
		if strings.Contains(authStr, "#!include auth-passwdfile.conf.ext") {
			authStr = strings.ReplaceAll(authStr, "#!include auth-passwdfile.conf.ext", "!include auth-passwdfile.conf.ext")
		}

		ioutil.WriteFile(authConfPath, []byte(authStr), 0644)
	}

	// Create virtual mail configuration with Maildir format and UID/GID settings
	dovecotConfig := `# WebStack CLI - Dovecot Configuration for Virtual Mail
# Override mail location for virtual domains using Maildir format
mail_location = maildir:/var/mail/vhosts/%d/%n
mail_privileged_group = mail

# Allow system users (mail user has uid 8)
first_valid_uid = 0
last_valid_uid = 0
`
	ioutil.WriteFile("/etc/dovecot/conf.d/99-webstack-mail.conf", []byte(dovecotConfig), 0644)

	// Configure Dovecot SASL socket for Postfix SMTP authentication
	saslConfig := `# WebStack CLI - Dovecot SASL socket for Postfix SMTP
service auth {
  unix_listener private/auth {
    mode = 0660
    user = postfix
    group = postfix
  }
}
`
	ioutil.WriteFile("/etc/dovecot/conf.d/95-postfix-sasl.conf", []byte(saslConfig), 0644)

	// Configure Dovecot LMTP socket for Postfix mail delivery
	lmtpConfig := `# WebStack CLI - Dovecot LMTP for Postfix delivery
service lmtp {
  unix_listener /var/spool/postfix/private/dovecot-lmtp {
    mode = 0660
    user = postfix
    group = postfix
  }
}
`
	ioutil.WriteFile("/etc/dovecot/conf.d/96-postfix-lmtp.conf", []byte(lmtpConfig), 0644)

	// Set proper permissions
	runCommandQuiet("chown", "-R", "mail:mail", "/var/mail/vhosts")
	os.Chmod("/var/mail/vhosts", 0755) // IMPORTANT: Must have execute permission for mail user
	runCommandQuiet("chown", "root:root", "/etc/dovecot/users")
	os.Chmod("/etc/dovecot/users", 0644)

	// Restart Dovecot to apply changes
	runCommandQuiet("systemctl", "restart", "dovecot")

	fmt.Println("✓ Dovecot virtual mail configuration updated")
}

func configureSpamAssassin() {
	// Only configure if SpamAssassin is installed
	if !isPackageInstalled("spamassassin") {
		return
	}

	fmt.Println("⚙️  Configuring SpamAssassin...")

	// Basic SpamAssassin configuration
	saConfig := `# WebStack CLI - SpamAssassin Configuration
required_score 5.0
rewrite_header Subject [SPAM]
report_safe 1
trusted_networks 127.0.0.0/8 ::1
`

	if err := ioutil.WriteFile("/etc/spamassassin/local.cf.webstack", []byte(saConfig), 0644); err != nil {
		fmt.Printf("⚠️  Warning: Could not write SpamAssassin config: %v\n", err)
	} else {
		fmt.Println("✓ SpamAssassin configuration prepared")
	}
}

// addMailFirewallRules opens common mail ports when a firewall tool is available
// AddMailFirewallRules opens mail ports in firewall if firewall tool is present
func AddMailFirewallRules() {
	fmt.Println("🔥 Configuring firewall for mail ports (if firewall present)...")

	// Mail ports to open (TCP)
	mailPorts := []int{25, 465, 587, 110, 995, 143, 993, 4190}

	// If ufw exists, prefer using it
	if runCommandQuiet("which", "ufw") == nil {
		fmt.Println("ℹ️  UFW detected - adding rules via ufw")
		for _, p := range mailPorts {
			portStr := fmt.Sprintf("%d/tcp", p)
			runCommandQuiet("ufw", "allow", portStr)
		}
		runCommandQuiet("ufw", "reload")
		fmt.Println("✅ Mail ports opened in UFW firewall")
		return
	}

	// Fall back to iptables if available
	if runCommandQuiet("which", "iptables") == nil {
		fmt.Println("ℹ️  iptables detected - adding rules via iptables")
		for _, p := range mailPorts {
			portStr := fmt.Sprintf("%d", p)
			runCommandQuiet("iptables", "-A", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT")
			// Add IPv6 rule if ip6tables exists
			if runCommandQuiet("which", "ip6tables") == nil {
				runCommandQuiet("ip6tables", "-A", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT")
			}
		}
		// Persist rules (best-effort)
		runCommandQuiet("bash", "-c", "iptables-save > /etc/iptables/rules.v4 2>/dev/null || true")
		runCommandQuiet("bash", "-c", "ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true")
		fmt.Println("✅ Mail ports opened in iptables firewall")
		return
	}

	// If no recognized firewall tool is present, just inform the user
	fmt.Println("⚠️  No firewall management tool (ufw/iptables) detected. Please open these mail ports manually if needed:")
	for _, p := range mailPorts {
		fmt.Printf("  - %d/tcp\n", p)
	}
}

// RemoveMailFirewallRules closes mail ports in firewall if firewall tool is present
func RemoveMailFirewallRules() {
	fmt.Println("🔥 Removing mail ports from firewall (if firewall present)...")

	// Mail ports to close (TCP)
	mailPorts := []int{25, 465, 587, 110, 995, 143, 993, 4190}

	// If ufw exists, prefer using it
	if runCommandQuiet("which", "ufw") == nil {
		fmt.Println("ℹ️  UFW detected - removing rules via ufw")
		for _, p := range mailPorts {
			portStr := fmt.Sprintf("%d/tcp", p)
			runCommandQuiet("ufw", "delete", "allow", portStr)
		}
		runCommandQuiet("ufw", "reload")
		fmt.Println("✅ Mail ports closed in UFW firewall")
		return
	}

	// Fall back to iptables if available
	if runCommandQuiet("which", "iptables") == nil {
		fmt.Println("ℹ️  iptables detected - removing rules via iptables")
		for _, p := range mailPorts {
			portStr := fmt.Sprintf("%d", p)
			runCommandQuiet("iptables", "-D", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT")
			// Remove IPv6 rule if ip6tables exists
			if runCommandQuiet("which", "ip6tables") == nil {
				runCommandQuiet("ip6tables", "-D", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT")
			}
		}
		// Persist rules (best-effort)
		runCommandQuiet("bash", "-c", "iptables-save > /etc/iptables/rules.v4 2>/dev/null || true")
		runCommandQuiet("bash", "-c", "ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true")
		fmt.Println("✅ Mail ports closed in iptables firewall")
		return
	}

	// If no recognized firewall tool is present, just inform the user
	fmt.Println("⚠️  No firewall management tool (ufw/iptables) detected. Please close these mail ports manually if needed:")
	for _, p := range mailPorts {
		fmt.Printf("  - %d/tcp\n", p)
	}
}

// ==================== MAIL ACCOUNT & DOMAIN MANAGEMENT ====================

// AddMailAccount adds a new mail account
func AddMailAccount(email, password string) {
	fmt.Printf("📧 Adding mail account: %s\n", email)

	// Extract domain from email
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		fmt.Println("❌ Invalid email format. Use: user@domain.tld")
		return
	}

	domain := parts[1]
	user := parts[0]

	// Create mailbox directory with proper Maildir structure
	mailDir := fmt.Sprintf("/var/mail/vhosts/%s/%s", domain, user)
	if err := os.MkdirAll(mailDir, 0755); err != nil {
		fmt.Printf("❌ Error creating mailbox directory: %v\n", err)
		return
	}

	// Create Maildir subdirectories (new, cur, tmp)
	for _, subdir := range []string{"new", "cur", "tmp"} {
		subdirPath := filepath.Join(mailDir, subdir)
		if err := os.MkdirAll(subdirPath, 0700); err != nil {
			fmt.Printf("⚠️  Warning: Could not create %s directory: %v\n", subdir, err)
		}
	}

	// Set ownership to mail user for entire domain tree
	if err := runCommand("chown", "-R", "mail:mail", fmt.Sprintf("/var/mail/vhosts/%s", domain)); err != nil {
		fmt.Printf("⚠️  Warning: Could not set directory ownership: %v\n", err)
	}

	// Ensure vhosts directory has proper permissions (needed for mail user access)
	os.Chmod("/var/mail/vhosts", 0755)

	// Create virtual mailbox maps file if it doesn't exist
	vhostFile := "/etc/postfix/vmailbox"
	content, _ := ioutil.ReadFile(vhostFile)
	contentStr := string(content)

	// Check if account already exists
	if strings.Contains(contentStr, email) {
		fmt.Printf("⚠️  Account %s already exists\n", email)
		return
	}

	// Add account to virtual mailbox file
	newEntry := fmt.Sprintf("%s\t%s/%s/\n", email, domain, user)
	if err := ioutil.WriteFile(vhostFile, []byte(contentStr+newEntry), 0644); err != nil {
		fmt.Printf("❌ Error writing mailbox file: %v\n", err)
		return
	}

	// Add account to Dovecot users file (format: email:{PLAIN}password:uid:gid::homedir::)
	os.MkdirAll("/etc/dovecot", 0755)

	usersFile := "/etc/dovecot/users"
	usersContent, _ := ioutil.ReadFile(usersFile)
	usersStr := string(usersContent)

	// Check if account already in users file
	if strings.Contains(usersStr, email+":") {
		fmt.Printf("⚠️  Account %s already exists in Dovecot\n", email)
		return
	}

	// Create dovecot users file entry
	// Format: email:{PLAIN}password:uid:gid::homedir::
	homeDir := fmt.Sprintf("/var/mail/vhosts/%s/%s", domain, user)
	dovecotEntry := fmt.Sprintf("%s:{PLAIN}%s:mail:mail::%s::\n", email, password, homeDir)

	if err := ioutil.WriteFile(usersFile, append(usersContent, []byte(dovecotEntry)...), 0644); err != nil {
		fmt.Printf("❌ Error writing Dovecot users file: %v\n", err)
		return
	}

	// Reload Postfix maps - regenerate database from text files
	fmt.Println("🔄 Updating Postfix mailbox maps...")
	runCommandQuiet("postmap", vhostFile)
	runCommandQuiet("postfix", "reload")

	fmt.Printf("✅ Mail account %s added successfully\n", email)
	fmt.Printf("💡 Mailbox location: %s\n", mailDir)
}

// generateDKIMKeyPair generates DKIM keys for a domain
func generateDKIMKeyPair(domain string) (string, string, error) {
	dkimDir := "/etc/postfix/dkim"

	// Create DKIM directory if it doesn't exist
	if err := os.MkdirAll(dkimDir, 0700); err != nil {
		return "", "", fmt.Errorf("failed to create DKIM directory: %v", err)
	}

	privateKeyPath := filepath.Join(dkimDir, domain+".private.key")
	publicKeyPath := filepath.Join(dkimDir, domain+".public.key")

	// Generate 2048-bit RSA key pair
	fmt.Println("🔐 Generating DKIM keypair...")
	cmd := exec.Command("openssl", "genrsa", "-out", privateKeyPath, "2048")
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %v - %s", err, string(output))
	}

	// Extract public key
	cmd = exec.Command("openssl", "rsa", "-in", privateKeyPath, "-pubout", "-out", publicKeyPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("failed to extract public key: %v - %s", err, string(output))
	}

	// Set proper permissions
	os.Chmod(privateKeyPath, 0600)
	os.Chmod(publicKeyPath, 0644)

	// Read and format public key for DKIM record
	pubKeyContent, _ := ioutil.ReadFile(publicKeyPath)
	pubKeyStr := string(pubKeyContent)

	// Extract just the key part (between BEGIN and END)
	lines := strings.Split(pubKeyStr, "\n")
	var keyPart []string
	for _, line := range lines {
		if !strings.HasPrefix(line, "-----") && strings.TrimSpace(line) != "" {
			keyPart = append(keyPart, strings.TrimSpace(line))
		}
	}
	dkimPublicKey := strings.Join(keyPart, "")

	return privateKeyPath, dkimPublicKey, nil
}

// getServerIP returns the primary server IP address
func getServerIP() string {
	// Try to get IP from hostname resolution
	cmd := exec.Command("hostname", "-I")
	output, err := cmd.Output()
	if err == nil {
		ips := strings.Fields(strings.TrimSpace(string(output)))
		if len(ips) > 0 {
			// Filter out IPv6 and return first IPv4
			for _, ip := range ips {
				if !strings.Contains(ip, ":") {
					return ip
				}
			}
		}
	}

	// Fallback to getting IP from ip route
	cmd = exec.Command("ip", "route", "get", "1")
	output, err = cmd.Output()
	if err == nil {
		parts := strings.Fields(string(output))
		for i, part := range parts {
			if part == "src" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
	}

	return "YOUR_SERVER_IP"
}

// generateDNSRecords creates SPF, DKIM, and DMARC records for a domain
func generateDNSRecords(domain, dkimPublicKey string) string {
	serverIP := getServerIP()

	spfRecord := fmt.Sprintf("v=spf1 a mx ip4:%s -all", serverIP)
	dkimRecord := fmt.Sprintf("v=DKIM1; k=rsa; p=%s", dkimPublicKey)
	dmarcRecord := "v=DMARC1; p=quarantine; pct=100; rua=mailto:dmarc-reports@" + domain

	dnsRecords := fmt.Sprintf(`SPF Record (add as TXT record):
  Name: %s
  Value: %s

DKIM Record (add as TXT record):
  Name: default._domainkey.%s
  Value: %s

DMARC Record (add as TXT record):
  Name: _dmarc.%s
  Value: %s
`, domain, spfRecord, domain, dkimRecord, domain, dmarcRecord)

	return dnsRecords
}

// saveDNSRecords saves DNS records to a file for user reference
func saveDNSRecords(domain, dnsRecords string) error {
	dnsDir := "/etc/postfix/dns-records"
	if err := os.MkdirAll(dnsDir, 0755); err != nil {
		return fmt.Errorf("failed to create DNS records directory: %v", err)
	}

	filePath := filepath.Join(dnsDir, domain+".txt")
	if err := ioutil.WriteFile(filePath, []byte(dnsRecords), 0644); err != nil {
		return fmt.Errorf("failed to save DNS records: %v", err)
	}

	return nil
}

// AddMailDomain adds a new mail domain
func AddMailDomain(domain string) {
	fmt.Printf("🌐 Adding mail domain: %s\n", domain)

	// Create virtual domain directory
	domainDir := fmt.Sprintf("/var/mail/vhosts/%s", domain)
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		fmt.Printf("❌ Error creating domain directory: %v\n", err)
		return
	}

	// Set ownership
	if err := runCommand("chown", "-R", "mail:mail", domainDir); err != nil {
		fmt.Printf("⚠️  Warning: Could not set directory ownership: %v\n", err)
	}

	// Add domain to virtual domains file
	vdomainFile := "/etc/postfix/vdomains"
	content, _ := ioutil.ReadFile(vdomainFile)
	contentStr := string(content)

	if strings.Contains(contentStr, domain) {
		fmt.Printf("⚠️  Domain %s already exists\n", domain)
		return
	}

	newEntry := fmt.Sprintf("%s\tOK\n", domain)
	if err := ioutil.WriteFile(vdomainFile, []byte(contentStr+newEntry), 0644); err != nil {
		fmt.Printf("❌ Error writing domains file: %v\n", err)
		return
	}

	// Regenerate vdomains.db map
	fmt.Println("🔄 Updating Postfix domain maps...")
	runCommandQuiet("postmap", vdomainFile)

	// Generate DKIM keys
	_, dkimPublicKey, err := generateDKIMKeyPair(domain)
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not generate DKIM keys: %v\n", err)
	} else {
		fmt.Println("✅ DKIM keys generated successfully")
	}

	// Generate DNS records (SPF, DKIM, DMARC)
	dnsRecords := generateDNSRecords(domain, dkimPublicKey)
	if err := saveDNSRecords(domain, dnsRecords); err != nil {
		fmt.Printf("⚠️  Warning: Could not save DNS records: %v\n", err)
	}

	// Reload Postfix (only reload, don't map vmailbox since we didn't change it)
	fmt.Println("🔄 Reloading Postfix configuration...")
	runCommandQuiet("postfix", "reload")

	fmt.Printf("✅ Mail domain %s added successfully\n", domain)
	fmt.Printf("💡 Domain directory: %s\n", domainDir)
	fmt.Printf("💡 DKIM keys: /etc/postfix/dkim/%s.{private,public}.key\n", domain)
	fmt.Printf("💡 DNS records: /etc/postfix/dns-records/%s.txt\n", domain)
	fmt.Println("\n📋 DNS Records to add to your DNS provider:")
	fmt.Println(dnsRecords)
}

// ListMailAccounts lists all configured mail accounts
func ListMailAccounts() {
	fmt.Println("📋 Mail Accounts")
	fmt.Println("================")

	usersFile := "/etc/dovecot/users"
	content, err := ioutil.ReadFile(usersFile)
	if err != nil {
		fmt.Println("❌ No mail accounts configured yet")
		return
	}

	lines := strings.Split(string(content), "\n")
	accountCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			email := strings.Split(line, ":")[0]
			fmt.Printf("  • %s\n", email)
			accountCount++
		}
	}

	if accountCount == 0 {
		fmt.Println("❌ No mail accounts configured yet")
	} else {
		fmt.Printf("\n✅ Total: %d account(s)\n", accountCount)
	}
}

// ListMailDomains lists all configured mail domains
func ListMailDomains() {
	fmt.Println("📋 Mail Domains")
	fmt.Println("===============")

	vdomainFile := "/etc/postfix/vdomains"
	content, err := ioutil.ReadFile(vdomainFile)
	if err != nil {
		fmt.Println("❌ No mail domains configured yet")
		return
	}

	lines := strings.Split(string(content), "\n")
	domainCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			domain := strings.Fields(line)[0]
			fmt.Printf("  • %s\n", domain)
			domainCount++
		}
	}

	if domainCount == 0 {
		fmt.Println("❌ No mail domains configured yet")
	} else {
		fmt.Printf("\n✅ Total: %d domain(s)\n", domainCount)
	}
}

// DeleteMailAccount deletes a mail account
func DeleteMailAccount(email string) {
	fmt.Printf("🗑️  Deleting mail account: %s\n", email)

	if !improvedAskYesNo("Are you sure you want to delete this account?") {
		fmt.Println("⏭️  Deletion cancelled")
		return
	}

	// Extract domain from email
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		fmt.Println("❌ Invalid email format")
		return
	}

	domain := parts[1]
	user := parts[0]

	// Remove from virtual mailbox file
	vhostFile := "/etc/postfix/vmailbox"
	content, err := ioutil.ReadFile(vhostFile)
	if err != nil {
		fmt.Printf("❌ Error reading mailbox file: %v\n", err)
		return
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string

	for _, line := range lines {
		if !strings.HasPrefix(line, email) {
			newLines = append(newLines, line)
		}
	}

	if err := ioutil.WriteFile(vhostFile, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
		fmt.Printf("❌ Error updating mailbox file: %v\n", err)
		return
	}

	// Remove mailbox directory
	mailDir := fmt.Sprintf("/var/mail/vhosts/%s/%s", domain, user)
	if err := os.RemoveAll(mailDir); err != nil {
		fmt.Printf("⚠️  Warning: Could not remove mailbox directory: %v\n", err)
	}

	// Remove from password file
	passwordDir := "/etc/dovecot/passwd.d"
	passFile := filepath.Join(passwordDir, strings.ReplaceAll(domain, ".", "_")+".passwd")
	passContent, _ := ioutil.ReadFile(passFile)

	passLines := strings.Split(string(passContent), "\n")
	var newPassLines []string

	for _, line := range passLines {
		if !strings.HasPrefix(line, email) {
			newPassLines = append(newPassLines, line)
		}
	}

	ioutil.WriteFile(passFile, []byte(strings.Join(newPassLines, "\n")), 0600)

	// Reload Postfix
	runCommandQuiet("postmap", vhostFile)
	runCommandQuiet("postfix", "reload")

	fmt.Printf("✅ Mail account %s deleted successfully\n", email)
}

// DeleteMailDomain deletes a mail domain
func DeleteMailDomain(domain string) {
	fmt.Printf("🗑️  Deleting mail domain: %s\n", domain)

	if !improvedAskYesNo("Are you sure you want to delete this domain and all its accounts?") {
		fmt.Println("⏭️  Deletion cancelled")
		return
	}

	// Remove from virtual domains file
	vdomainFile := "/etc/postfix/vdomains"
	content, err := ioutil.ReadFile(vdomainFile)
	if err != nil {
		fmt.Printf("❌ Error reading domains file: %v\n", err)
		return
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string

	for _, line := range lines {
		if !strings.HasPrefix(line, domain) {
			newLines = append(newLines, line)
		}
	}

	if err := ioutil.WriteFile(vdomainFile, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
		fmt.Printf("❌ Error updating domains file: %v\n", err)
		return
	}

	// Remove domain directory
	domainDir := fmt.Sprintf("/var/mail/vhosts/%s", domain)
	if err := os.RemoveAll(domainDir); err != nil {
		fmt.Printf("⚠️  Warning: Could not remove domain directory: %v\n", err)
	}

	// Reload Postfix
	runCommandQuiet("postmap", vdomainFile)
	runCommandQuiet("postfix", "reload")

	fmt.Printf("✅ Mail domain %s deleted successfully\n", domain)
}

// ShowDNSRecords displays DNS records for a domain
func ShowDNSRecords(domain string) {
	dnsRecordsFile := fmt.Sprintf("/etc/postfix/dns-records/%s.txt", domain)

	content, err := ioutil.ReadFile(dnsRecordsFile)
	if err != nil {
		fmt.Printf("❌ No DNS records found for domain %s\n", domain)
		fmt.Printf("💡 Add the domain first: webstack mail add domain %s\n", domain)
		return
	}

	fmt.Printf("📋 DNS Records for %s\n", domain)
	fmt.Println("=" + strings.Repeat("=", len(domain)))
	fmt.Println(string(content))
	fmt.Printf("💡 File location: %s\n", dnsRecordsFile)
}

// ImportMailDNSToBind imports mail DNS records into BIND
func ImportMailDNSToBind(domain string) {
	fmt.Printf("🔗 Importing mail DNS records to BIND for %s\n", domain)

	// Check if BIND is installed
	if err := runCommandQuiet("which", "named"); err != nil {
		fmt.Println("❌ BIND DNS server is not installed")
		fmt.Println("💡 Install with: sudo webstack dns install")
		return
	}

	// Read DNS records from file
	dnsRecordsFile := fmt.Sprintf("/etc/postfix/dns-records/%s.txt", domain)
	content, err := ioutil.ReadFile(dnsRecordsFile)
	if err != nil {
		fmt.Printf("❌ DNS records not found for %s\n", domain)
		fmt.Printf("💡 Add the domain first: webstack mail add domain %s\n", domain)
		return
	}

	// Check if zone is already configured in BIND
	namedConfLocal := "/etc/bind/named.conf.local"
	bindConfig, err := ioutil.ReadFile(namedConfLocal)
	if err != nil {
		fmt.Println("❌ Could not read BIND configuration")
		return
	}

	bindConfigStr := string(bindConfig)
	if !strings.Contains(bindConfigStr, fmt.Sprintf(`zone "%s"`, domain)) {
		fmt.Println("⚠️  Zone not configured in BIND")
		fmt.Printf("💡 Configure zone first: sudo webstack dns config --zone %s --type master\n", domain)
		return
	}

	// Extract the zone file path from BIND config
	var zoneFilePath string
	lines := strings.Split(bindConfigStr, "\n")
	inZone := false
	for _, line := range lines {
		if strings.Contains(line, fmt.Sprintf(`zone "%s"`, domain)) {
			inZone = true
		}
		if inZone && strings.Contains(line, "file") {
			// Extract file path from line like: file "/var/lib/bind/db.example.com";
			parts := strings.Split(line, "\"")
			if len(parts) >= 2 {
				zoneFilePath = parts[1]
			}
			break
		}
	}

	if zoneFilePath == "" {
		zoneFilePath = fmt.Sprintf("/var/lib/bind/db.%s", domain)
	}

	// Check if zone file exists, if not create a basic one
	if _, err := os.Stat(zoneFilePath); os.IsNotExist(err) {
		fmt.Printf("📝 Creating zone file: %s\n", zoneFilePath)
		if err := createBasicZoneFile(zoneFilePath, domain); err != nil {
			fmt.Printf("❌ Could not create zone file: %v\n", err)
			return
		}
	}

	// Parse DNS records and add them to zone file
	if err := addMailRecordsToZone(zoneFilePath, domain, string(content)); err != nil {
		fmt.Printf("❌ Error adding records to zone file: %v\n", err)
		return
	}

	// Check BIND configuration
	if err := runCommandQuiet("named-checkconf"); err != nil {
		fmt.Println("⚠️  BIND configuration check failed")
		return
	}

	// Reload BIND
	fmt.Println("🔄 Reloading BIND configuration...")
	if err := runCommandQuiet("systemctl", "reload", "bind9"); err != nil {
		fmt.Println("⚠️  Could not reload BIND service")
		return
	}

	fmt.Printf("✅ Mail DNS records imported for %s\n", domain)
	fmt.Printf("💡 Zone file: %s\n", zoneFilePath)
	fmt.Printf("💡 To verify: dig @127.0.0.1 %s\n", domain)
}

// createBasicZoneFile creates a basic BIND zone file for a domain
func createBasicZoneFile(filePath, domain string) error {
	// Get serial number from current date/time
	serialCmd := exec.Command("date", "+%Y%m%d01")
	serialOutput, _ := serialCmd.Output()
	serial := strings.TrimSpace(string(serialOutput))

	serverIP := getServerIP()

	zoneContent := fmt.Sprintf(`; Zone file for %s
$TTL 3600
@   IN  SOA ns1.%s. hostmaster.%s. (
        %s  ; Serial
        10800       ; Refresh
        3600        ; Retry
        604800      ; Expire
        3600 )      ; Minimum TTL
    IN  NS  ns1.%s.
    IN  A   %s
ns1 IN  A   %s
mail IN A   %s
`, domain, domain, domain, serial, domain, serverIP, serverIP, serverIP)

	if err := ioutil.WriteFile(filePath, []byte(zoneContent), 0644); err != nil {
		return err
	}

	// Set proper ownership
	runCommandQuiet("chown", "bind:bind", filePath)
	runCommandQuiet("chmod", "644", filePath)

	return nil
}

// addMailRecordsToZone adds SPF, DKIM, and DMARC records to a zone file
func addMailRecordsToZone(filePath, domain, dnsRecordsContent string) error {
	// Read current zone file
	currentContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	zoneContent := string(currentContent)

	// Parse SPF, DKIM, and DMARC records from the DNS records content
	lines := strings.Split(dnsRecordsContent, "\n")
	var spfRecord, dkimRecord, dmarcRecord string

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "v=spf1") {
			// Extract SPF value (should be on the next line or same line)
			if strings.HasPrefix(line, "Value:") {
				spfRecord = strings.TrimPrefix(line, "Value:")
				spfRecord = strings.TrimSpace(spfRecord)
			} else if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if strings.HasPrefix(nextLine, "Value:") {
					spfRecord = strings.TrimPrefix(nextLine, "Value:")
					spfRecord = strings.TrimSpace(spfRecord)
				}
			}
		} else if strings.Contains(line, "v=DKIM1") {
			if strings.HasPrefix(line, "Value:") {
				dkimRecord = strings.TrimPrefix(line, "Value:")
				dkimRecord = strings.TrimSpace(dkimRecord)
			} else if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if strings.HasPrefix(nextLine, "Value:") {
					dkimRecord = strings.TrimPrefix(nextLine, "Value:")
					dkimRecord = strings.TrimSpace(dkimRecord)
				}
			}
		} else if strings.Contains(line, "v=DMARC1") {
			if strings.HasPrefix(line, "Value:") {
				dmarcRecord = strings.TrimPrefix(line, "Value:")
				dmarcRecord = strings.TrimSpace(dmarcRecord)
			} else if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if strings.HasPrefix(nextLine, "Value:") {
					dmarcRecord = strings.TrimPrefix(nextLine, "Value:")
					dmarcRecord = strings.TrimSpace(dmarcRecord)
				}
			}
		}
	}

	// Add records to zone file if not already present
	if spfRecord != "" && !strings.Contains(zoneContent, "v=spf1") {
		zoneContent += fmt.Sprintf("\n; SPF Record\n@   IN  TXT \"%s\"\n", spfRecord)
	}

	if dkimRecord != "" && !strings.Contains(zoneContent, "v=DKIM1") {
		zoneContent += fmt.Sprintf("\n; DKIM Record\ndefault._domainkey IN TXT \"%s\"\n", dkimRecord)
	}

	if dmarcRecord != "" && !strings.Contains(zoneContent, "v=DMARC1") {
		zoneContent += fmt.Sprintf("\n; DMARC Record\n_dmarc IN TXT \"%s\"\n", dmarcRecord)
	}

	// Increment serial number
	zoneContent = incrementSerial(zoneContent)

	// Write updated zone file
	if err := ioutil.WriteFile(filePath, []byte(zoneContent), 0644); err != nil {
		return err
	}

	// Set proper ownership
	runCommandQuiet("chown", "bind:bind", filePath)

	// Check zone file syntax
	cmd := exec.Command("named-checkzone", domain, filePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("zone file validation failed: %s", string(output))
	}

	return nil
}

// incrementSerial increments the serial number in a zone file
func incrementSerial(zoneContent string) string {
	lines := strings.Split(zoneContent, "\n")

	for i, line := range lines {
		if strings.Contains(line, "; Serial") {
			// Previous line should contain the serial number
			if i > 0 {
				prevLine := strings.TrimSpace(lines[i-1])
				// Extract current serial
				if serialStr := strings.Fields(prevLine)[0]; serialStr != "" {
					if currentSerial, err := strconv.Atoi(serialStr); err == nil {
						newSerial := currentSerial + 1
						lines[i-1] = fmt.Sprintf("        %d  ; Serial", newSerial)
						return strings.Join(lines, "\n")
					}
				}
			}
		}
	}

	return zoneContent
}
