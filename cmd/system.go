package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var systemCmd = &cobra.Command{
	Use:   "system",
	Short: "System management commands",
	Long:  `System-level management commands for WebStack CLI service integration.`,
}

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload all web server configurations",
	Run:   reloadConfigurations,
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate all configurations",
	Run:   validateConfigurations,
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up temporary files and logs",
	Run:   cleanupSystem,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show system status",
	Run:   showSystemStatus,
}

var remoteAccessCmd = &cobra.Command{
	Use:   "remote-access",
	Short: "Configure remote database access",
	Long:  `Enable or disable remote connections to MySQL/MariaDB/PostgreSQL.`,
}

var remoteAccessEnableCmd = &cobra.Command{
	Use:   "enable [database] [user] [password]",
	Short: "Enable remote access for a database",
	Long: `Enable remote connections for MySQL, MariaDB, or PostgreSQL.
Usage: 
  webstack system remote-access enable mysql (interactive prompts)
  webstack system remote-access enable mysql root rootpass (with args)
  webstack system remote-access enable mysql appuser apppass`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dbType := strings.ToLower(args[0])
		var user, password string
		if len(args) >= 3 {
			user = args[1]
			password = args[2]
			enableRemoteAccessWithArgs(dbType, user, password)
		} else {
			enableRemoteAccess(dbType)
		}
	},
}

var remoteAccessDisableCmd = &cobra.Command{
	Use:   "disable [database] [user]",
	Short: "Disable remote access for a database",
	Long: `Disable remote connections for MySQL, MariaDB, or PostgreSQL.
Usage:
  webstack system remote-access disable mysql (interactive prompts)
  webstack system remote-access disable mysql root (with user)`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dbType := strings.ToLower(args[0])
		var user string
		if len(args) >= 2 {
			user = args[1]
			disableRemoteAccessWithArgs(dbType, user)
		} else {
			disableRemoteAccess(dbType)
		}
	},
}

var remoteAccessStatusCmd = &cobra.Command{
	Use:   "status [database]",
	Short: "Check remote access status for a database",
	Long:  `Check if remote connections are enabled for MySQL, MariaDB, or PostgreSQL. Usage: webstack system remote-access status mysql`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dbType := strings.ToLower(args[0])
		checkRemoteAccessStatus(dbType)
	},
}

func reloadConfigurations(cmd *cobra.Command, args []string) {
	quiet, _ := cmd.Flags().GetBool("quiet")

	if !quiet {
		fmt.Println("🔄 Reloading WebStack configurations...")
	}

	// Reload Nginx
	if isServiceActive("nginx") {
		if err := runSystemCommand("systemctl", "reload", "nginx"); err != nil {
			if !quiet {
				fmt.Printf("❌ Failed to reload Nginx: %v\n", err)
			}
		} else if !quiet {
			fmt.Println("✅ Nginx configuration reloaded")
		}
	}

	// Reload Apache
	if isServiceActive("apache2") {
		if err := runSystemCommand("systemctl", "reload", "apache2"); err != nil {
			if !quiet {
				fmt.Printf("❌ Failed to reload Apache: %v\n", err)
			}
		} else if !quiet {
			fmt.Println("✅ Apache configuration reloaded")
		}
	}

	// Reload PHP-FPM services
	phpServices := []string{"php5.6-fpm", "php7.0-fpm", "php7.1-fpm", "php7.2-fpm", "php7.3-fpm", "php7.4-fpm", "php8.0-fpm", "php8.1-fpm", "php8.2-fpm", "php8.3-fpm", "php8.4-fpm"}

	for _, service := range phpServices {
		if isServiceActive(service) {
			if err := runSystemCommand("systemctl", "reload", service); err != nil {
				if !quiet {
					fmt.Printf("❌ Failed to reload %s: %v\n", service, err)
				}
			} else if !quiet {
				fmt.Printf("✅ %s configuration reloaded\n", service)
			}
		}
	}

	if !quiet {
		fmt.Println("🎉 Configuration reload completed")
	}
}

func validateConfigurations(cmd *cobra.Command, args []string) {
	quiet, _ := cmd.Flags().GetBool("quiet")

	if !quiet {
		fmt.Println("🔍 Validating WebStack configurations...")
	}

	errors := 0

	// Validate Nginx configuration
	if isServiceInstalled("nginx") {
		if err := runSystemCommand("nginx", "-t"); err != nil {
			if !quiet {
				fmt.Printf("❌ Nginx configuration validation failed: %v\n", err)
			}
			errors++
		} else if !quiet {
			fmt.Println("✅ Nginx configuration is valid")
		}
	}

	// Validate Apache configuration
	if isServiceInstalled("apache2") {
		if err := runSystemCommand("apache2ctl", "configtest"); err != nil {
			if !quiet {
				fmt.Printf("❌ Apache configuration validation failed: %v\n", err)
			}
			errors++
		} else if !quiet {
			fmt.Println("✅ Apache configuration is valid")
		}
	}

	// Check domain configurations
	// TODO: Implement domain configuration validation

	// Check SSL certificates
	// TODO: Implement SSL certificate validation

	if !quiet {
		if errors == 0 {
			fmt.Println("🎉 All configurations are valid")
		} else {
			fmt.Printf("⚠️  Found %d configuration errors\n", errors)
		}
	}

	if errors > 0 {
		os.Exit(1)
	}
}

func cleanupSystem(cmd *cobra.Command, args []string) {
	quiet, _ := cmd.Flags().GetBool("quiet")

	if !quiet {
		fmt.Println("🧹 Cleaning up WebStack temporary files...")
	}

	// Clean temporary files
	if !quiet {
		fmt.Println("  • Cleaning temporary files...")
	}

	// Clean WebStack temporary files
	runSystemCommand("find", "/tmp", "-name", "webstack-*", "-type", "f", "-mtime", "+7", "-delete")
	runSystemCommand("find", "/var/tmp", "-name", "webstack-*", "-type", "f", "-mtime", "+7", "-delete")

	// Clean Nginx cache if it exists
	runSystemCommand("find", "/var/cache/nginx", "-type", "f", "-mtime", "+7", "-delete")

	// Rotate large logs
	if !quiet {
		fmt.Println("  • Rotating large log files...")
	}
	runSystemCommand("find", "/var/log/webstack", "-name", "*.log", "-size", "+100M", "-exec", "truncate", "-s", "0", "{}", "\\;")

	// Clean old SSL certificates (expired + 30 days)
	// TODO: Implement SSL cleanup

	if !quiet {
		fmt.Println("✅ Cleanup completed")
	}
}

func showSystemStatus(cmd *cobra.Command, args []string) {
	fmt.Println("WebStack System Status")
	fmt.Println("=====================")
	fmt.Println()

	// Check services
	services := []string{"nginx", "apache2", "mysql", "mariadb", "postgresql"}

	fmt.Println("🔧 Services:")
	for _, service := range services {
		if isServiceInstalled(service) {
			if isServiceActive(service) {
				fmt.Printf("  ✅ %s: Running\n", service)
			} else {
				fmt.Printf("  ❌ %s: Stopped\n", service)
			}
		}
	}

	// Check PHP-FPM versions
	fmt.Println("\n🐘 PHP-FPM Services:")
	phpServices := []string{"php5.6-fpm", "php7.0-fpm", "php7.1-fpm", "php7.2-fpm", "php7.3-fpm", "php7.4-fpm", "php8.0-fpm", "php8.1-fpm", "php8.2-fpm", "php8.3-fpm", "php8.4-fpm"}

	phpCount := 0
	for _, service := range phpServices {
		if isServiceActive(service) {
			version := service[3:6] // Extract version like "8.2" from "php8.2-fpm"
			fmt.Printf("  ✅ PHP %s: Running\n", version)
			phpCount++
		}
	}

	if phpCount == 0 {
		fmt.Println("  ⚠️  No PHP-FPM services running")
	}

	// Check disk space
	fmt.Println("\n💾 Disk Usage:")
	runSystemCommand("df", "-h", "/var/www", "/var/log", "/etc")

	// Check domains
	// TODO: Show domain count and status

	// Check SSL certificates
	// TODO: Show SSL certificate status
}

// Helper functions
func isServiceInstalled(service string) bool {
	err := runSystemCommand("systemctl", "list-unit-files", service)
	return err == nil
}

func isServiceActive(service string) bool {
	err := runSystemCommand("systemctl", "is-active", "--quiet", service)
	return err == nil
}

func runSystemCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// Helper functions with arguments (non-interactive)
func enableRemoteAccessWithArgs(dbType, user, password string) {
	fmt.Printf("🔓 Enabling remote access for %s (user: %s)...\n", dbType, user)

	switch dbType {
	case "mysql":
		enableMySQLRemoteAccessWithArgs(user, password)
	case "mariadb":
		enableMySQLRemoteAccessWithArgs(user, password) // Same as MySQL
	case "postgresql":
		enablePostgreSQLRemoteAccessWithArgs(user, password)
	default:
		fmt.Printf("❌ Unknown database type: %s\n", dbType)
		fmt.Println("Supported: mysql, mariadb, postgresql")
	}
}

func disableRemoteAccessWithArgs(dbType, user string) {
	fmt.Printf("🔒 Disabling remote access for %s (user: %s)...\n", dbType, user)

	switch dbType {
	case "mysql":
		disableMySQLRemoteAccessWithArgs(user)
	case "mariadb":
		disableMySQLRemoteAccessWithArgs(user)
	case "postgresql":
		disablePostgreSQLRemoteAccessWithArgs(user)
	default:
		fmt.Printf("❌ Unknown database type: %s\n", dbType)
		fmt.Println("Supported: mysql, mariadb, postgresql")
	}
}

// Remote access functions for MySQL/MariaDB
func enableRemoteAccess(dbType string) {
	fmt.Printf("🔓 Enabling remote access for %s...\n", dbType)

	switch dbType {
	case "mysql":
		enableMySQLRemoteAccess()
	case "mariadb":
		enableMariaDBRemoteAccess()
	case "postgresql":
		enablePostgreSQLRemoteAccess()
	default:
		fmt.Printf("❌ Unknown database type: %s\n", dbType)
		fmt.Println("Supported: mysql, mariadb, postgresql")
	}
}

func disableRemoteAccess(dbType string) {
	fmt.Printf("🔒 Disabling remote access for %s...\n", dbType)

	switch dbType {
	case "mysql":
		disableMySQLRemoteAccess()
	case "mariadb":
		disableMariaDBRemoteAccess()
	case "postgresql":
		disablePostgreSQLRemoteAccess()
	default:
		fmt.Printf("❌ Unknown database type: %s\n", dbType)
		fmt.Println("Supported: mysql, mariadb, postgresql")
	}
}

func checkRemoteAccessStatus(dbType string) {
	switch dbType {
	case "mysql", "mariadb":
		checkMySQLRemoteAccessStatus(dbType)
	case "postgresql":
		checkPostgreSQLRemoteAccessStatus()
	default:
		fmt.Printf("❌ Unknown database type: %s\n", dbType)
		fmt.Println("Supported: mysql, mariadb, postgresql")
	}
}

func enableMySQLRemoteAccess() {
	configFile := "/etc/mysql/mariadb.conf.d/99-webstack.cnf"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		configFile = "/etc/mysql/mysql.conf.d/mysqld.cnf"
	}

	// Prompt user for IP/network
	fmt.Println("\n📋 MySQL/MariaDB Remote Access Configuration")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Allow connections from:")
	fmt.Println("  1. Any IP (%) - LESS SECURE")
	fmt.Println("  2. Specific IP address")
	fmt.Println("  3. Specific subnet (e.g., 192.168.1.%)")
	fmt.Print("\nEnter choice (1-3) or custom address: ")

	var input string
	fmt.Scanln(&input)

	var bindAddress, hostPattern string
	switch input {
	case "1":
		bindAddress = "0.0.0.0"
		hostPattern = "%"
		fmt.Println("⚠️  WARNING: Allowing connections from ANY IP is less secure!")
	case "2":
		fmt.Print("Enter IP address: ")
		fmt.Scanln(&bindAddress)
		hostPattern = bindAddress
	case "3":
		fmt.Print("Enter subnet pattern (e.g., 192.168.1.%): ")
		fmt.Scanln(&hostPattern)
		bindAddress = "0.0.0.0"
	default:
		bindAddress = "0.0.0.0"
		hostPattern = input
	}

	fmt.Printf("\n✓ Allowing connections from: %s\n", hostPattern)

	// Update config file
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("❌ Error reading config: %v\n", err)
		return
	}

	content := string(data)
	// Replace bind-address with new value
	if strings.Contains(content, "bind-address") {
		// Match bind-address lines with various formats
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.Contains(line, "bind-address") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
				lines[i] = "bind-address = " + bindAddress
				break
			}
		}
		content = strings.Join(lines, "\n")
	} else {
		// If not found, add it
		content += "\nbind-address = " + bindAddress + "\n"
	}

	if err := ioutil.WriteFile(configFile, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Error writing config: %v\n", err)
		return
	}

	service := "mysql"
	if _, err := exec.Command("systemctl", "is-active", "mariadb").Output(); err == nil {
		service = "mariadb"
	}

	if err := exec.Command("systemctl", "restart", service).Run(); err != nil {
		fmt.Printf("❌ Error restarting %s: %v\n", service, err)
		return
	}

	fmt.Println("✓ Updated bind-address in config")

	// Get admin user (for running GRANT command)
	fmt.Print("\n� Enter MySQL/MariaDB admin user (default: root): ")
	var adminUser string
	fmt.Scanln(&adminUser)
	if adminUser == "" {
		adminUser = "root"
	}

	// Get admin password
	fmt.Print("🔐 Enter admin user password: ")
	var adminPassword string
	fmt.Scanln(&adminPassword)

	// Ask which user to grant privileges to
	fmt.Print("\n👤 Enter database user to grant remote access (default: root): ")
	var dbUser string
	fmt.Scanln(&dbUser)
	if dbUser == "" {
		dbUser = "root"
	}

	// Ask for user password (for IDENTIFIED BY)
	fmt.Printf("Enter password for user '%s': ", dbUser)
	var userPassword string
	fmt.Scanln(&userPassword)

	// Update database user privileges
	fmt.Printf("✓ Granting privileges to %s@%s...\n", dbUser, hostPattern)
	grantCmd := fmt.Sprintf("GRANT ALL PRIVILEGES ON *.* TO '%s'@'%s' IDENTIFIED BY '%s' WITH GRANT OPTION; FLUSH PRIVILEGES;",
		dbUser, hostPattern, userPassword)

	mysqlCmd := exec.Command("mysql", "-u", adminUser, "-p"+adminPassword, "-e", grantCmd)
	if err := mysqlCmd.Run(); err != nil {
		fmt.Printf("Error granting privileges: %v\n", err)
		fmt.Println("   You may need to run manually:")
		fmt.Printf("   mysql -u %s -p -e \"GRANT ALL PRIVILEGES ON *.* TO '%s'@'%s' WITH GRANT OPTION; FLUSH PRIVILEGES;\"\n", adminUser, dbUser, hostPattern)
		return
	}

	// Open firewall port 3306 for MySQL/MariaDB
	fmt.Println("Opening firewall port 3306 for MySQL/MariaDB...")
	exec.Command("iptables", "-A", "INPUT", "-p", "tcp", "--dport", "3306", "-j", "ACCEPT").Run()
	exec.Command("ip6tables", "-A", "INPUT", "-p", "tcp", "--dport", "3306", "-j", "ACCEPT").Run()
	// Persist rules
	exec.Command("bash", "-c", "iptables-save > /etc/iptables/rules.v4 2>/dev/null || true").Run()
	exec.Command("bash", "-c", "ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true").Run()

	fmt.Printf("Remote access enabled for %s\n", service)
	fmt.Printf("   Listening on: %s:3306\n", bindAddress)
	fmt.Printf("   User '%s' can connect from: %s\n", dbUser, hostPattern)
	fmt.Printf("   Connect from: mysql -u %s -h <server-ip> -p\n", dbUser)
}

func disableMySQLRemoteAccess() {
	configFile := "/etc/mysql/mariadb.conf.d/99-webstack.cnf"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		configFile = "/etc/mysql/mysql.conf.d/mysqld.cnf"
	}

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		return
	}

	content := string(data)
	if strings.Contains(content, "bind-address") {
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.Contains(line, "bind-address") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
				lines[i] = "bind-address = 127.0.0.1"
				break
			}
		}
		content = strings.Join(lines, "\n")
	}

	if err := ioutil.WriteFile(configFile, []byte(content), 0644); err != nil {
		fmt.Printf("Error writing config: %v\n", err)
		return
	}

	service := "mysql"
	if _, err := exec.Command("systemctl", "is-active", "mariadb").Output(); err == nil {
		service = "mariadb"
	}

	if err := exec.Command("systemctl", "restart", service).Run(); err != nil {
		fmt.Printf("Error restarting %s: %v\n", service, err)
		return
	}

	fmt.Println("Updated bind-address in config")

	// Get admin user (for running queries)
	fmt.Print("\nEnter MySQL/MariaDB admin user (default: root): ")
	var adminUser string
	fmt.Scanln(&adminUser)
	if adminUser == "" {
		adminUser = "root"
	}

	// Get admin password
	fmt.Print("Enter admin user password: ")
	var adminPassword string
	fmt.Scanln(&adminPassword)

	// Ask which user to revoke privileges from
	fmt.Print("\nEnter database user to revoke remote access (default: root): ")
	var dbUser string
	fmt.Scanln(&dbUser)
	if dbUser == "" {
		dbUser = "root"
	}

	// Revoke remote privileges and keep only localhost
	fmt.Printf("✓ Revoking remote access privileges for %s...\n", dbUser)
	revokeCmd := fmt.Sprintf("DELETE FROM mysql.user WHERE User='%s' AND Host NOT IN ('localhost', '127.0.0.1', '::1'); FLUSH PRIVILEGES;", dbUser)

	mysqlCmd := exec.Command("mysql", "-u", adminUser, "-p"+adminPassword, "-e", revokeCmd)
	if err := mysqlCmd.Run(); err != nil {
		fmt.Printf("⚠️  Warning: Could not revoke remote privileges: %v\n", err)
		fmt.Println("   You may need to run manually:")
		fmt.Printf("   mysql -u %s -p -e \"DELETE FROM mysql.user WHERE User='%s' AND Host NOT IN ('localhost', '127.0.0.1', '::1'); FLUSH PRIVILEGES;\"\n", adminUser, dbUser)
	}

	// Close firewall port 3306 for MySQL/MariaDB
	fmt.Println("🔒 Closing firewall port 3306...")
	exec.Command("iptables", "-D", "INPUT", "-p", "tcp", "--dport", "3306", "-j", "ACCEPT").Run()
	exec.Command("ip6tables", "-D", "INPUT", "-p", "tcp", "--dport", "3306", "-j", "ACCEPT").Run()
	// Persist rules
	exec.Command("bash", "-c", "iptables-save > /etc/iptables/rules.v4 2>/dev/null || true").Run()
	exec.Command("bash", "-c", "ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true").Run()

	fmt.Printf("✅ Remote access disabled for %s (localhost only)\n", service)
}

// MySQL/MariaDB functions with direct arguments (non-interactive)
func enableMySQLRemoteAccessWithArgs(user, password string) {
	configFile := "/etc/mysql/mariadb.conf.d/99-webstack.cnf"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		configFile = "/etc/mysql/mysql.conf.d/mysqld.cnf"
	}

	// Set to allow from any host (%)
	hostPattern := "%"
	bindAddress := "0.0.0.0"

	// Update config file
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("❌ Error reading config: %v\n", err)
		return
	}

	content := string(data)
	if strings.Contains(content, "bind-address") {
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.Contains(line, "bind-address") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
				lines[i] = "bind-address = " + bindAddress
				break
			}
		}
		content = strings.Join(lines, "\n")
	} else {
		content += "\nbind-address = " + bindAddress + "\n"
	}

	if err := ioutil.WriteFile(configFile, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Error writing config: %v\n", err)
		return
	}

	service := "mysql"
	if _, err := exec.Command("systemctl", "is-active", "mariadb").Output(); err == nil {
		service = "mariadb"
	}

	if err := exec.Command("systemctl", "restart", service).Run(); err != nil {
		fmt.Printf("❌ Error restarting %s: %v\n", service, err)
		return
	}

	fmt.Println("✓ Updated bind-address in config")

	// Grant privileges using provided credentials
	fmt.Printf("✓ Granting privileges to %s@%s...\n", user, hostPattern)
	grantCmd := fmt.Sprintf("GRANT ALL PRIVILEGES ON *.* TO '%s'@'%s' IDENTIFIED BY '%s' WITH GRANT OPTION; FLUSH PRIVILEGES;",
		user, hostPattern, password)

	mysqlCmd := exec.Command("mysql", "-u", "root", "-p"+password, "-e", grantCmd)
	if err := mysqlCmd.Run(); err != nil {
		// Try with the provided user as admin
		mysqlCmd = exec.Command("mysql", "-u", user, "-p"+password, "-e", grantCmd)
		if err := mysqlCmd.Run(); err != nil {
			fmt.Printf("❌ Error granting privileges: %v\n", err)
			fmt.Println("   You may need to run manually:")
			fmt.Printf("   mysql -u root -p -e \"GRANT ALL PRIVILEGES ON *.* TO '%s'@'%s' WITH GRANT OPTION; FLUSH PRIVILEGES;\"\n", user, hostPattern)
			return
		}
	}

	fmt.Printf("✅ Remote access enabled for %s\n", service)
	fmt.Printf("   Listening on: %s:3306\n", bindAddress)
	fmt.Printf("   User '%s' can connect from: %s\n", user, hostPattern)
	fmt.Printf("   Connect from: mysql -u %s -h <server-ip> -p\n", user)
}

func disableMySQLRemoteAccessWithArgs(user string) {
	configFile := "/etc/mysql/mariadb.conf.d/99-webstack.cnf"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		configFile = "/etc/mysql/mysql.conf.d/mysqld.cnf"
	}

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("❌ Error reading config: %v\n", err)
		return
	}

	content := string(data)
	if strings.Contains(content, "bind-address") {
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.Contains(line, "bind-address") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
				lines[i] = "bind-address = 127.0.0.1"
				break
			}
		}
		content = strings.Join(lines, "\n")
	}

	if err := ioutil.WriteFile(configFile, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Error writing config: %v\n", err)
		return
	}

	service := "mysql"
	if _, err := exec.Command("systemctl", "is-active", "mariadb").Output(); err == nil {
		service = "mariadb"
	}

	if err := exec.Command("systemctl", "restart", service).Run(); err != nil {
		fmt.Printf("❌ Error restarting %s: %v\n", service, err)
		return
	}

	fmt.Println("✓ Updated bind-address in config")
	fmt.Printf("✅ Remote access disabled for %s (localhost only)\n", service)
	fmt.Printf("   User '%s' - remote connections revoked\n", user)
}

func checkMySQLRemoteAccessStatus(dbType string) {
	configFile := "/etc/mysql/mariadb.conf.d/99-webstack.cnf"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		configFile = "/etc/mysql/mysql.conf.d/mysqld.cnf"
	}

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("❌ Error reading config: %v\n", err)
		return
	}

	content := string(data)
	if strings.Contains(content, "#bind-address") || !strings.Contains(content, "bind-address") {
		fmt.Printf("🔓 Remote access is ENABLED for %s\n", dbType)
		fmt.Println("   Any client can connect if they have valid credentials")
	} else {
		fmt.Printf("🔒 Remote access is DISABLED for %s\n", dbType)
		fmt.Println("   Only localhost connections are allowed")
	}
}

func enableMariaDBRemoteAccess() {
	enableMySQLRemoteAccess()
}

func disableMariaDBRemoteAccess() {
	disableMySQLRemoteAccess()
}

// PostgreSQL remote access functions
func enablePostgreSQLRemoteAccess() {
	matches, _ := exec.Command("bash", "-c", "ls /etc/postgresql/*/main/postgresql.conf 2>/dev/null | head -1").Output()
	if len(matches) == 0 {
		fmt.Println("❌ PostgreSQL configuration file not found")
		return
	}

	configFile := strings.TrimSpace(string(matches))

	// Prompt user for IP/network
	fmt.Println("\n📋 PostgreSQL Remote Access Configuration")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Allow connections from:")
	fmt.Println("  1. Any IP (0.0.0.0/0) - LESS SECURE")
	fmt.Println("  2. Specific IP address")
	fmt.Println("  3. Specific subnet (e.g., 192.168.1.0/24)")
	fmt.Print("\nEnter choice (1-3) or custom address: ")

	var input string
	fmt.Scanln(&input)

	var cidrAddress string
	switch input {
	case "1":
		cidrAddress = "0.0.0.0/0"
		fmt.Println("⚠️  WARNING: Allowing connections from ANY IP is less secure!")
	case "2":
		fmt.Print("Enter IP address (will use /32 for single host): ")
		fmt.Scanln(&input)
		cidrAddress = input + "/32"
	case "3":
		fmt.Print("Enter subnet (e.g., 192.168.1.0/24): ")
		fmt.Scanln(&cidrAddress)
	default:
		cidrAddress = input
	}

	fmt.Printf("\n✓ Allowing connections from: %s\n", cidrAddress)

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("❌ Error reading config: %v\n", err)
		return
	}

	content := string(data)
	if strings.Contains(content, "#listen_addresses = 'localhost'") {
		content = strings.ReplaceAll(content, "#listen_addresses = 'localhost'", "listen_addresses = '*'")
	} else if strings.Contains(content, "listen_addresses = 'localhost'") {
		content = strings.ReplaceAll(content, "listen_addresses = 'localhost'", "listen_addresses = '*'")
	}

	if err := ioutil.WriteFile(configFile, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Error writing config: %v\n", err)
		return
	}

	pgHbaFile := strings.ReplaceAll(configFile, "postgresql.conf", "pg_hba.conf")
	pgHbaData, _ := ioutil.ReadFile(pgHbaFile)
	pgHbaContent := string(pgHbaData)

	// Remove any existing remote connection lines
	lines := strings.Split(pgHbaContent, "\n")
	var filteredLines []string
	for _, line := range lines {
		if !strings.Contains(line, "# Remote connections") && !strings.Contains(line, "host    all") {
			filteredLines = append(filteredLines, line)
		}
	}
	pgHbaContent = strings.Join(filteredLines, "\n")

	// Add new remote connection line with md5 auth
	pgHbaContent += fmt.Sprintf("\n# Remote connections\nhost    all             all             %s               md5\n", cidrAddress)
	ioutil.WriteFile(pgHbaFile, []byte(pgHbaContent), 0644)
	fmt.Println("✓ Updated pg_hba.conf to allow remote connections")

	if err := exec.Command("systemctl", "restart", "postgresql").Run(); err != nil {
		fmt.Printf("❌ Error restarting PostgreSQL: %v\n", err)
		return
	}

	// Grant privileges to postgres user
	fmt.Print("\n� Enter PostgreSQL user to grant remote access (default: postgres): ")
	var dbUser string
	fmt.Scanln(&dbUser)
	if dbUser == "" {
		dbUser = "postgres"
	}

	// Get user password
	fmt.Printf("🔐 Enter password for user '%s': ", dbUser)
	var password string
	fmt.Scanln(&password)

	fmt.Printf("✓ Setting password for %s user...\n", dbUser)
	altersqlCmd := fmt.Sprintf("ALTER USER %s WITH PASSWORD '%s';", dbUser, password)
	psqlCmd := exec.Command("sudo", "-u", "postgres", "psql", "-c", altersqlCmd)
	if err := psqlCmd.Run(); err != nil {
		fmt.Printf("⚠️  Warning: Could not set password: %v\n", err)
		fmt.Println("   You may need to run manually:")
		fmt.Printf("   sudo -u postgres psql -c \"ALTER USER %s WITH PASSWORD 'your_password';\"\n", dbUser)
	}

	// Open firewall port 5432 for PostgreSQL
	fmt.Println("🔥 Opening firewall port 5432 for PostgreSQL...")
	exec.Command("iptables", "-A", "INPUT", "-p", "tcp", "--dport", "5432", "-j", "ACCEPT").Run()
	exec.Command("ip6tables", "-A", "INPUT", "-p", "tcp", "--dport", "5432", "-j", "ACCEPT").Run()
	// Persist rules
	exec.Command("bash", "-c", "iptables-save > /etc/iptables/rules.v4 2>/dev/null || true").Run()
	exec.Command("bash", "-c", "ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true").Run()

	fmt.Println("✅ Remote access enabled for PostgreSQL")
	fmt.Printf("   Listening on: 0.0.0.0:5432 (from %s)\n", cidrAddress)
	fmt.Printf("   User '%s' can connect from: psql -U %s -h <server-ip> -d postgres\n", dbUser, dbUser)
}

func disablePostgreSQLRemoteAccess() {
	matches, _ := exec.Command("bash", "-c", "ls /etc/postgresql/*/main/postgresql.conf 2>/dev/null | head -1").Output()
	if len(matches) == 0 {
		fmt.Println("❌ PostgreSQL configuration file not found")
		return
	}

	configFile := strings.TrimSpace(string(matches))
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("❌ Error reading config: %v\n", err)
		return
	}

	content := string(data)
	if strings.Contains(content, "listen_addresses = '*'") {
		content = strings.ReplaceAll(content, "listen_addresses = '*'", "#listen_addresses = 'localhost'")
	}

	if err := ioutil.WriteFile(configFile, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Error writing config: %v\n", err)
		return
	}

	if err := exec.Command("systemctl", "restart", "postgresql").Run(); err != nil {
		fmt.Printf("❌ Error restarting PostgreSQL: %v\n", err)
		return
	}

	// Ask which user to revoke privileges from
	fmt.Print("\n👤 Enter PostgreSQL user to revoke remote access (default: postgres): ")
	var dbUser string
	fmt.Scanln(&dbUser)
	if dbUser == "" {
		dbUser = "postgres"
	}

	// Reset user password (optional)
	fmt.Print("Reset password for user? (y/n, default: n): ")
	var resetPass string
	fmt.Scanln(&resetPass)

	if resetPass == "y" || resetPass == "Y" {
		fmt.Printf("Enter new password for user '%s': ", dbUser)
		var password string
		fmt.Scanln(&password)

		resetCmd := fmt.Sprintf("ALTER USER %s WITH PASSWORD '%s';", dbUser, password)
		psqlCmd := exec.Command("sudo", "-u", "postgres", "psql", "-c", resetCmd)
		if err := psqlCmd.Run(); err != nil {
			fmt.Printf("⚠️  Warning: Could not reset password: %v\n", err)
		}
	}

	// Close firewall port 5432 for PostgreSQL
	fmt.Println("🔒 Closing firewall port 5432...")
	exec.Command("iptables", "-D", "INPUT", "-p", "tcp", "--dport", "5432", "-j", "ACCEPT").Run()
	exec.Command("ip6tables", "-D", "INPUT", "-p", "tcp", "--dport", "5432", "-j", "ACCEPT").Run()
	// Persist rules
	exec.Command("bash", "-c", "iptables-save > /etc/iptables/rules.v4 2>/dev/null || true").Run()
	exec.Command("bash", "-c", "ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true").Run()

	fmt.Printf("✅ Remote access disabled for PostgreSQL (localhost only)\n")
	fmt.Printf("   User '%s' - remote connections revoked\n", dbUser)
}

// PostgreSQL functions with direct arguments (non-interactive)
func enablePostgreSQLRemoteAccessWithArgs(user, password string) {
	matches, _ := exec.Command("bash", "-c", "ls /etc/postgresql/*/main/postgresql.conf 2>/dev/null | head -1").Output()
	if len(matches) == 0 {
		fmt.Println("❌ PostgreSQL configuration file not found")
		return
	}

	configFile := strings.TrimSpace(string(matches))
	cidrAddress := "0.0.0.0/0"

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("❌ Error reading config: %v\n", err)
		return
	}

	content := string(data)
	if strings.Contains(content, "#listen_addresses = 'localhost'") {
		content = strings.ReplaceAll(content, "#listen_addresses = 'localhost'", "listen_addresses = '*'")
	} else if strings.Contains(content, "listen_addresses = 'localhost'") {
		content = strings.ReplaceAll(content, "listen_addresses = 'localhost'", "listen_addresses = '*'")
	}

	if err := ioutil.WriteFile(configFile, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Error writing config: %v\n", err)
		return
	}

	pgHbaFile := strings.ReplaceAll(configFile, "postgresql.conf", "pg_hba.conf")
	pgHbaData, _ := ioutil.ReadFile(pgHbaFile)
	pgHbaContent := string(pgHbaData)

	// Remove any existing remote connection lines
	lines := strings.Split(pgHbaContent, "\n")
	var filteredLines []string
	for _, line := range lines {
		if !strings.Contains(line, "# Remote connections") && !strings.Contains(line, "host    all") {
			filteredLines = append(filteredLines, line)
		}
	}
	pgHbaContent = strings.Join(filteredLines, "\n")

	// Add new remote connection line with md5 auth
	pgHbaContent += fmt.Sprintf("\n# Remote connections\nhost    all             all             %s               md5\n", cidrAddress)
	ioutil.WriteFile(pgHbaFile, []byte(pgHbaContent), 0644)
	fmt.Println("✓ Updated pg_hba.conf to allow remote connections")

	if err := exec.Command("systemctl", "restart", "postgresql").Run(); err != nil {
		fmt.Printf("❌ Error restarting PostgreSQL: %v\n", err)
		return
	}

	fmt.Printf("✓ Setting password for %s user...\n", user)
	altersqlCmd := fmt.Sprintf("ALTER USER %s WITH PASSWORD '%s';", user, password)
	psqlCmd := exec.Command("sudo", "-u", "postgres", "psql", "-c", altersqlCmd)
	if err := psqlCmd.Run(); err != nil {
		fmt.Printf("⚠️  Warning: Could not set password: %v\n", err)
		fmt.Println("   You may need to run manually:")
		fmt.Printf("   sudo -u postgres psql -c \"ALTER USER %s WITH PASSWORD 'your_password';\"\n", user)
	}

	fmt.Println("✅ Remote access enabled for PostgreSQL")
	fmt.Printf("   Listening on: 0.0.0.0:5432 (from %s)\n", cidrAddress)
	fmt.Printf("   User '%s' can connect from: psql -U %s -h <server-ip> -d postgres\n", user, user)
}

func disablePostgreSQLRemoteAccessWithArgs(user string) {
	matches, _ := exec.Command("bash", "-c", "ls /etc/postgresql/*/main/postgresql.conf 2>/dev/null | head -1").Output()
	if len(matches) == 0 {
		fmt.Println("❌ PostgreSQL configuration file not found")
		return
	}

	configFile := strings.TrimSpace(string(matches))
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("❌ Error reading config: %v\n", err)
		return
	}

	content := string(data)
	if strings.Contains(content, "listen_addresses = '*'") {
		content = strings.ReplaceAll(content, "listen_addresses = '*'", "#listen_addresses = 'localhost'")
	}

	if err := ioutil.WriteFile(configFile, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Error writing config: %v\n", err)
		return
	}

	if err := exec.Command("systemctl", "restart", "postgresql").Run(); err != nil {
		fmt.Printf("❌ Error restarting PostgreSQL: %v\n", err)
		return
	}

	fmt.Printf("✅ Remote access disabled for PostgreSQL (localhost only)\n")
	fmt.Printf("   User '%s' - remote connections revoked\n", user)
}

func checkPostgreSQLRemoteAccessStatus() {
	matches, _ := exec.Command("bash", "-c", "ls /etc/postgresql/*/main/postgresql.conf 2>/dev/null | head -1").Output()
	if len(matches) == 0 {
		fmt.Println("❌ PostgreSQL configuration file not found")
		return
	}

	configFile := strings.TrimSpace(string(matches))
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("❌ Error reading config: %v\n", err)
		return
	}

	content := string(data)
	if strings.Contains(content, "listen_addresses = '*'") {
		fmt.Println("🔓 Remote access is ENABLED for PostgreSQL")
		fmt.Println("   Any client can connect if they have valid credentials")
	} else {
		fmt.Println("🔒 Remote access is DISABLED for PostgreSQL")
		fmt.Println("   Only localhost connections are allowed")
	}
}

func setupCoreSecurity() {
	fmt.Println("🔒 Setting up core security infrastructure...")

	// Remove UFW if installed (conflicts with iptables)
	fmt.Println("   Checking for UFW conflicts...")
	ufwOutput, err := exec.Command("dpkg", "-l").Output()
	if err == nil && strings.Contains(string(ufwOutput), "ufw") {
		fmt.Println("   ⚠️  UFW detected, removing to avoid conflicts with iptables...")
		exec.Command("bash", "-c", "systemctl disable ufw 2>/dev/null || true").Run()
		exec.Command("bash", "-c", "systemctl stop ufw 2>/dev/null || true").Run()
		exec.Command("apt", "remove", "-y", "ufw").Run()
		fmt.Println("   ✓ UFW removed")
	}

	// Core security packages - installed once for all components
	coreSecurityPkgs := []string{
		"iptables",            // Kernel firewall engine
		"iptables-persistent", // Persist firewall rules across reboot
		"ipset",               // Efficient IP list management
		"fail2ban",            // Automatic brute-force protection
	}

	// Update package list
	fmt.Println("   Updating package list...")
	exec.Command("apt", "update").Run()

	// Install core security packages
	fmt.Println("   Installing security packages...")
	args := append([]string{"install", "-y", "--no-install-recommends"}, coreSecurityPkgs...)
	if err := exec.Command("apt", args...).Run(); err != nil {
		fmt.Printf("⚠️  Warning installing security packages: %v\n", err)
		// Don't return - these might already be installed
	}

	// Ensure SSH port 22 is always allowed (prevent lockout)
	fmt.Println("   Ensuring SSH access...")
	exec.Command("iptables", "-A", "INPUT", "-p", "tcp", "--dport", "22", "-j", "ACCEPT").Run()
	exec.Command("ip6tables", "-A", "INPUT", "-p", "tcp", "--dport", "22", "-j", "ACCEPT").Run()

	// Allow localhost traffic (always safe)
	exec.Command("iptables", "-A", "INPUT", "-i", "lo", "-j", "ACCEPT").Run()
	exec.Command("ip6tables", "-A", "INPUT", "-i", "lo", "-j", "ACCEPT").Run()

	// Allow established/related connections (traffic from existing connections)
	exec.Command("iptables", "-A", "INPUT", "-m", "state", "--state", "ESTABLISHED,RELATED", "-j", "ACCEPT").Run()
	exec.Command("ip6tables", "-A", "INPUT", "-m", "state", "--state", "ESTABLISHED,RELATED", "-j", "ACCEPT").Run()

	// Persist rules
	exec.Command("bash", "-c", "iptables-save > /etc/iptables/rules.v4 2>/dev/null || true").Run()
	exec.Command("bash", "-c", "ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true").Run()

	fmt.Println("✓ Core security packages ready (iptables, ipset, fail2ban)")
	fmt.Println("✓ SSH access preserved (port 22 allowed)")
}

func init() {
	rootCmd.AddCommand(systemCmd)
	systemCmd.AddCommand(reloadCmd)
	systemCmd.AddCommand(validateCmd)
	systemCmd.AddCommand(cleanupCmd)
	systemCmd.AddCommand(statusCmd)
	systemCmd.AddCommand(remoteAccessCmd)

	// Add remote-access subcommands
	remoteAccessCmd.AddCommand(remoteAccessEnableCmd)
	remoteAccessCmd.AddCommand(remoteAccessDisableCmd)
	remoteAccessCmd.AddCommand(remoteAccessStatusCmd)

	// Add quiet flag to system commands
	reloadCmd.Flags().Bool("quiet", false, "Suppress output")
	validateCmd.Flags().Bool("quiet", false, "Suppress output")
	cleanupCmd.Flags().Bool("quiet", false, "Suppress output")
}
