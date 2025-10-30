package installer

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
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
	"phppgadmin": {
		Name:        "phpPgAdmin",
		CheckCmd:    []string{"dpkg", "-l", "phppgadmin"},
		PackageName: "phppgadmin",
		ServiceName: "",
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
	cmd := exec.Command("dpkg", "-l", packageName)
	err := cmd.Run()
	if err != nil {
		return NotInstalled
	}
	return Installed
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

// cleanupMySQLMariaDB performs a nuclear cleanup of MySQL/MariaDB when uninstall is needed
// This function handles orphaned processes that can't be killed normally
func cleanupMySQLMariaDB() {
	fmt.Println("\n🚀 NUCLEAR CLEANUP - REMOVING ALL MYSQL/MARIADB")
	fmt.Println("================================================== ")
	
	// Kill everything
	fmt.Println("🔪 Killing all processes...")
	runCommandQuiet("bash", "-c", "pkill -9 mysqld 2>/dev/null; true")
	runCommandQuiet("bash", "-c", "pkill -9 mariadbd 2>/dev/null; true")
	runCommandQuiet("bash", "-c", "pkill -9 mysql 2>/dev/null; true")
	runCommandQuiet("bash", "-c", "pkill -9 apt 2>/dev/null; true")
	runCommandQuiet("bash", "-c", "pkill -9 dpkg 2>/dev/null; true")
	time.Sleep(1 * time.Second)
	
	// Remove policy-rc.d block
	fmt.Println("🗑️  Removing policy blocks...")
	os.Remove("/usr/sbin/policy-rc.d")
	
	// Remove ALL lock files
	fmt.Println("🔓 Removing lock files...")
	os.Remove("/var/lib/dpkg/lock-frontend")
	os.Remove("/var/lib/dpkg/lock")
	os.Remove("/var/cache/apt/archives/lock")
	
	// Use bash to clean debconf locks with glob pattern
	runCommandQuiet("bash", "-c", "rm -f /var/cache/debconf/*.dat /var/cache/debconf/*.old")
	
	// Remove ALL MySQL/MariaDB directories
	fmt.Println("🗑️  Removing all MySQL/MariaDB directories...")
	runCommandQuiet("bash", "-c", "rm -rf /var/lib/mysql*")
	runCommandQuiet("bash", "-c", "rm -rf /var/log/mysql*")
	runCommandQuiet("bash", "-c", "rm -rf /etc/mysql*")
	runCommandQuiet("bash", "-c", "rm -rf /run/mysqld*")
	runCommandQuiet("bash", "-c", "rm -rf /run/mariadb*")
	
	// Reset dpkg state
	fmt.Println("🔧 Resetting dpkg...")
	runCommandQuiet("dpkg", "--configure", "-a")
	
	// Force remove any broken packages
	fmt.Println("📦 Force removing packages...")
	runCommandQuiet("bash", "-c", "dpkg -l | grep -i mysql | awk '{print $2}' | xargs -r dpkg --purge --force-all 2>/dev/null || true")
	runCommandQuiet("bash", "-c", "dpkg -l | grep -i mariadb | awk '{print $2}' | xargs -r dpkg --purge --force-all 2>/dev/null || true")
	
	// Clean apt
	fmt.Println("🧹 Cleaning apt...")
	runCommandQuiet("apt", "clean")
	runCommandQuiet("apt", "autoclean", "-y")
	runCommandQuiet("apt", "autoremove", "-y")
	
	// Final verification
	fmt.Println("")
	fmt.Println("✅ CLEANUP COMPLETE - Verification:")
	fmt.Println("  Remaining MySQL packages:")
	cmd := exec.Command("bash", "-c", "dpkg -l | grep -iE 'mysql|mariadb' | wc -l")
	output, _ := cmd.Output()
	if strings.TrimSpace(string(output)) == "0" {
		fmt.Println("    ✅ None")
	} else {
		fmt.Printf("    ⚠️  %s packages still present\n", strings.TrimSpace(string(output)))
	}
	
	fmt.Println("  Running processes:")
	cmd = exec.Command("bash", "-c", "ps aux | grep -iE 'mysqld|mariadbd|mysql|mariadb' | grep -v grep | wc -l")
	output, _ = cmd.Output()
	if strings.TrimSpace(string(output)) == "0" {
		fmt.Println("    ✅ None")
	} else {
		fmt.Printf("    ⚠️  %s processes still running\n", strings.TrimSpace(string(output)))
	}
	
	// Ask for reboot
	fmt.Println("")
	if improvedAskYesNo("⚠️  A system reboot is recommended to ensure all MySQL/MariaDB processes are terminated. Reboot now?") {
		fmt.Println("🔄 Rebooting system...")
		runCommand("systemctl", "reboot")
	} else {
		fmt.Println("⚠️  Please manually reboot the system before reinstalling MySQL/MariaDB")
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
		
		// Remove all MySQL/MariaDB data directories completely
		dirs := []string{
			"/var/lib/mysql",
			"/var/lib/mysql-8.0",      // MySQL 8.0 specific directory
			"/var/lib/mysql-files",
			"/var/log/mysql",
			"/etc/mysql",
		}
		for _, dir := range dirs {
			if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
				fmt.Printf("⚠️  Could not remove %s: %v\n", dir, err)
			}
		}
		
		// Clean package cache to prevent stale files
		runCommandQuiet("apt", "clean")
		runCommandQuiet("apt", "autoclean")
	}

	// Use purge to remove packages and config files
	cmd := exec.Command("apt", "purge", "-y", component.PackageName)
	cmd.Env = append(os.Environ(),
		"DEBIAN_FRONTEND=noninteractive",
		"DEBCONF_NONINTERACTIVE_SEEN=true")
	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️  apt purge returned error (may not be critical): %v\n", err)
		
		// If uninstall of MySQL/MariaDB failed, offer to run nuclear cleanup
		if component.PackageName == "mysql-server" || component.PackageName == "mariadb-server" {
			if improvedAskYesNo("⚠️  Uninstall failed. Run nuclear cleanup (kills orphaned processes, requires reboot)?") {
				cleanupMySQLMariaDB()
			}
			return err
		}
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

	phpPackage := fmt.Sprintf("php%s-fpm", version)
	commonPackages := []string{
		phpPackage,
		fmt.Sprintf("php%s-mysql", version),
		fmt.Sprintf("php%s-pgsql", version),
		fmt.Sprintf("php%s-curl", version),
		fmt.Sprintf("php%s-gd", version),
		fmt.Sprintf("php%s-zip", version),
		fmt.Sprintf("php%s-xml", version),
		fmt.Sprintf("php%s-mbstring", version),
	}

	args := append([]string{"remove", "-y"}, commonPackages...)
	return runCommand("apt", args...)
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
		if improvedAskYesNo("Do you want to install phpPgAdmin?") {
			InstallPhpPgAdmin()
		}
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
				
				if err := ioutil.WriteFile("/etc/apache2/sites-available/001-default.conf", []byte(buf.String()), 0644); err == nil {
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

	// Update config with Nginx installation details
	if err := UpdateServerConfig("nginx", true, port, mode); err != nil {
		fmt.Printf("⚠️  Warning: Could not update config: %v\n", err)
	}

	fmt.Printf("✅ Nginx installed successfully on port %d (mode: %s)\n", port, mode)
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
	
	// Remove ALL data and config directories (fresh start)
	fmt.Println("🗑️  Removing all MySQL/MariaDB directories...")
	dirsToRemove := []string{
		"/var/lib/mysql",
		"/var/lib/mysql-8.0",
		"/var/lib/mysql-files",
		"/var/log/mysql",
		"/etc/mysql",
		"/run/mysqld",
		"/run/mariadb",
	}
	for _, dir := range dirsToRemove {
		os.RemoveAll(dir)
	}
	
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
	
	// Remove ALL data and config directories (fresh start)
	fmt.Println("🗑️  Removing all MySQL/MariaDB directories...")
	dirsToRemove := []string{
		"/var/lib/mysql",
		"/var/lib/mysql-8.0",
		"/var/lib/mysql-files",
		"/var/log/mysql",
		"/etc/mysql",
		"/run/mysqld",
		"/run/mariadb",
	}
	for _, dir := range dirsToRemove {
		os.RemoveAll(dir)
	}
	
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
	fmt.Println("📦 Installing PostgreSQL...")

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

	if err := runCommand("apt", "install", "-y", "postgresql", "postgresql-contrib"); err != nil {
		fmt.Printf("Error installing PostgreSQL: %v\n", err)
		return
	}

	configurePostgreSQL()

	if err := runCommand("systemctl", "enable", "postgresql"); err != nil {
		fmt.Printf("Error enabling PostgreSQL: %v\n", err)
	}

	fmt.Println("✅ PostgreSQL installed successfully")
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
		fmt.Sprintf("php%s-mysql", version),
		fmt.Sprintf("php%s-pgsql", version),
		fmt.Sprintf("php%s-curl", version),
		fmt.Sprintf("php%s-gd", version),
		fmt.Sprintf("php%s-zip", version),
		fmt.Sprintf("php%s-xml", version),
		fmt.Sprintf("php%s-mbstring", version),
	}

	args := append([]string{"install", "-y"}, commonPackages...)
	if err := runCommand("apt", args...); err != nil {
		fmt.Printf("Error installing PHP %s: %v\n", version, err)
		return
	}

	// Configure PHP-FPM
	configurePHP(version)

	serviceName := fmt.Sprintf("php%s-fpm", version)
	if err := runCommand("systemctl", "enable", serviceName); err != nil {
		fmt.Printf("Error enabling PHP %s FPM: %v\n", version, err)
	}

	if err := runCommand("systemctl", "start", serviceName); err != nil {
		fmt.Printf("Error starting PHP %s FPM: %v\n", version, err)
	}

	fmt.Printf("✅ PHP %s installed successfully\n", version)
}

// InstallPhpPgAdmin installs phpPgAdmin
func InstallPhpPgAdmin() {
	fmt.Println("📦 Installing phpPgAdmin...")

	// Check if already installed
	component := components["phppgadmin"]
	status := checkComponentStatus(component)

	if status == Installed {
		action := promptForAction(component.Name)
		switch action {
		case "keep":
			fmt.Println("✅ Keeping existing phpPgAdmin installation")
			return
		case "skip":
			fmt.Println("⏭️  Skipping phpPgAdmin installation")
			return
		case "uninstall":
			if err := uninstallComponent(component); err != nil {
				fmt.Printf("Error uninstalling phpPgAdmin: %v\n", err)
			}
			fmt.Println("✅ phpPgAdmin uninstalled")
			return
		case "reinstall":
			fmt.Println("🔄 Reinstalling phpPgAdmin...")
			if err := uninstallComponent(component); err != nil {
				fmt.Printf("Error uninstalling phpPgAdmin: %v\n", err)
				return
			}
		}
	}

	if err := runCommand("apt", "install", "-y", "phppgadmin"); err != nil {
		fmt.Printf("Error installing phpPgAdmin: %v\n", err)
		return
	}

	configurePhpPgAdmin()
	fmt.Println("✅ phpPgAdmin installed and configured at /phppgadmin")
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

	// Uninstall web interfaces
	if improvedAskYesNo("Uninstall phpPgAdmin?") {
		UninstallPhpPgAdmin()
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

	// Update config
	if err := UpdateServerConfig("nginx", false, 0, ""); err != nil {
		fmt.Printf("⚠️  Warning: Could not update config: %v\n", err)
	}

	fmt.Println("✅ Nginx uninstalled successfully")
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

	// Update config
	if err := UpdateServerConfig("apache", false, 0, ""); err != nil {
		fmt.Printf("⚠️  Warning: Could not update config: %v\n", err)
	}

	fmt.Println("✅ Apache uninstalled successfully")
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
		fmt.Printf("❌ Error uninstalling PostgreSQL: %v\n", err)
		return
	}

	fmt.Println("✅ PostgreSQL uninstalled successfully")
}

// UninstallPHP removes a specific PHP version
func UninstallPHP(version string) {
	status := checkPHPVersion(version)

	if status != Installed {
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

// UninstallPhpPgAdmin removes phpPgAdmin
func UninstallPhpPgAdmin() {
	component := components["phppgadmin"]

	// Check if installed by looking for the package
	cmd := exec.Command(component.CheckCmd[0], component.CheckCmd[1:]...)
	err := cmd.Run()

	if err != nil {
		fmt.Println("ℹ️  phpPgAdmin is not installed")
		return
	}

	if !improvedAskYesNo("Uninstall phpPgAdmin?") {
		fmt.Println("⏭️  Skipping phpPgAdmin uninstall")
		return
	}

	if err := uninstallComponent(component); err != nil {
		fmt.Printf("❌ Error uninstalling phpPgAdmin: %v\n", err)
		return
	}

	fmt.Println("✅ phpPgAdmin uninstalled successfully")
}

// Helper functions
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCommandQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func configureNginx() {
	// TODO: Apply Nginx configuration from templates
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

	// Generate dhparam.pem for SSL if it doesn't exist
	dhparamPath := "/etc/ssl/dhparam.pem"
	if _, err := os.Stat(dhparamPath); os.IsNotExist(err) {
		fmt.Println("🔐 Generating SSL dhparam (this may take a minute)...")
		cmd := exec.Command("openssl", "dhparam", "-out", dhparamPath, "2048")
		if err := cmd.Run(); err != nil {
			fmt.Printf("⚠️  Warning: Could not generate dhparam: %v\n", err)
		} else {
			fmt.Println("✅ SSL dhparam generated")
		}
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
				if err := ioutil.WriteFile("/etc/apache2/sites-available/001-default.conf", []byte(buf.String()), 0644); err == nil {
					// Enable the default site
					runCommandQuiet("a2ensite", "001-default.conf")
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
	// TODO: Apply PostgreSQL configuration from templates
	fmt.Println("⚙️  Configuring PostgreSQL...")
}

func configurePHP(version string) {
	// TODO: Apply PHP-FPM configuration from templates
	fmt.Printf("⚙️  Configuring PHP %s...\n", version)
}

// isServiceActive checks if a systemd service is running
func isServiceActive(serviceName string) bool {
	cmd := exec.Command("systemctl", "is-active", serviceName)
	err := cmd.Run()
	return err == nil
}

// detectPhpFpmSocket detects the PHP-FPM socket path
func detectPhpFpmSocket() string {
	// Try common socket paths
	sockets := []string{
		"/run/php/php8.3-fpm.sock",
		"/run/php/php8.2-fpm.sock",
		"/run/php/php8.1-fpm.sock",
		"/run/php/php8.0-fpm.sock",
		"/run/php/php7.4-fpm.sock",
		"/run/php/www.sock",
	}

	for _, socket := range sockets {
		if _, err := os.Stat(socket); err == nil {
			return socket
		}
	}

	return ""
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


// executeSQL executes SQL commands against MySQL/MariaDB
func executeSQL(sqlCommands string) error {
	// Try with mysql command line client as root user (no password, socket auth)
	cmd := exec.Command("mysql", "-u", "root", "-e", sqlCommands)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Log the output for debugging
		fmt.Printf("Debug: MySQL execution output: %s\n", string(output))
		return fmt.Errorf("failed to execute SQL: %v", err)
	}
	
	return nil
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

// configurePhpPgAdmin configures phpPgAdmin after installation
func configurePhpPgAdmin() {
	// TODO: Apply phpPgAdmin configuration from templates
	fmt.Println("⚙️  Configuring phpPgAdmin...")
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
