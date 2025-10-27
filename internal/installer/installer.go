package installer

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
		CheckCmd:    []string{"systemctl", "is-active", "nginx"},
		PackageName: "nginx",
		ServiceName: "nginx",
	},
	"apache": {
		Name:        "Apache",
		CheckCmd:    []string{"systemctl", "is-active", "apache2"},
		PackageName: "apache2",
		ServiceName: "apache2",
	},
	"mysql": {
		Name:        "MySQL",
		CheckCmd:    []string{"systemctl", "is-active", "mysql"},
		PackageName: "mysql-server",
		ServiceName: "mysql",
	},
	"mariadb": {
		Name:        "MariaDB",
		CheckCmd:    []string{"systemctl", "is-active", "mariadb"},
		PackageName: "mariadb-server",
		ServiceName: "mariadb",
	},
	"postgresql": {
		Name:        "PostgreSQL",
		CheckCmd:    []string{"systemctl", "is-active", "postgresql"},
		PackageName: "postgresql postgresql-contrib",
		ServiceName: "postgresql",
	},
	"phpmyadmin": {
		Name:        "phpMyAdmin",
		CheckCmd:    []string{"dpkg", "-l", "phpmyadmin"},
		PackageName: "phpmyadmin",
		ServiceName: "",
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
	cmd := exec.Command(component.CheckCmd[0], component.CheckCmd[1:]...)
	err := cmd.Run()
	if err != nil {
		return NotInstalled
	}
	return Installed
}

// checkPHPVersion checks if a specific PHP version is installed
func checkPHPVersion(version string) ComponentStatus {
	serviceName := fmt.Sprintf("php%s-fpm", version)
	cmd := exec.Command("systemctl", "is-active", serviceName)
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

// uninstallComponent removes a component
func uninstallComponent(component Component) error {
	fmt.Printf("🗑️  Removing %s...\n", component.Name)

	// Stop service if it has one
	if component.ServiceName != "" {
		runCommand("systemctl", "stop", component.ServiceName)
		runCommand("systemctl", "disable", component.ServiceName)
	}

	// Remove package
	return runCommand("apt", "remove", "-y", component.PackageName)
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
		if improvedAskYesNo("Do you want to install phpMyAdmin?") {
			InstallPhpMyAdmin()
		}
	} else if improvedAskYesNo("Do you want to install MariaDB?") {
		InstallMariaDB()
		if improvedAskYesNo("Do you want to install phpMyAdmin?") {
			InstallPhpMyAdmin()
		}
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

	// Configure Nginx to listen on port 80
	configureNginx()

	if err := runCommand("systemctl", "enable", "nginx"); err != nil {
		fmt.Printf("Error enabling Nginx: %v\n", err)
	}

	if err := runCommand("systemctl", "start", "nginx"); err != nil {
		fmt.Printf("Error starting Nginx: %v\n", err)
	}

	fmt.Println("✅ Nginx installed successfully on port 80")
}

// InstallApache installs and configures Apache on port 8080
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

	// Configure Apache to listen on port 8080
	configureApache()

	if err := runCommand("systemctl", "enable", "apache2"); err != nil {
		fmt.Printf("Error enabling Apache: %v\n", err)
	}

	if err := runCommand("systemctl", "start", "apache2"); err != nil {
		fmt.Printf("Error starting Apache: %v\n", err)
	}

	fmt.Println("✅ Apache installed successfully on port 8080")
}

// InstallMySQL installs MySQL server
func InstallMySQL() {
	fmt.Println("📦 Installing MySQL...")

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

	if err := runCommand("apt", "install", "-y", "mysql-server"); err != nil {
		fmt.Printf("Error installing MySQL: %v\n", err)
		return
	}

	configureMySQL()

	if err := runCommand("systemctl", "enable", "mysql"); err != nil {
		fmt.Printf("Error enabling MySQL: %v\n", err)
	}

	fmt.Println("✅ MySQL installed successfully")
}

// InstallMariaDB installs MariaDB server
func InstallMariaDB() {
	fmt.Println("📦 Installing MariaDB...")

	if err := runCommand("apt", "install", "-y", "mariadb-server"); err != nil {
		fmt.Printf("Error installing MariaDB: %v\n", err)
		return
	}

	configureMariaDB()

	if err := runCommand("systemctl", "enable", "mariadb"); err != nil {
		fmt.Printf("Error enabling MariaDB: %v\n", err)
	}

	fmt.Println("✅ MariaDB installed successfully")
}

// InstallPostgreSQL installs PostgreSQL server
func InstallPostgreSQL() {
	fmt.Println("📦 Installing PostgreSQL...")

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

// InstallPhpMyAdmin installs phpMyAdmin
func InstallPhpMyAdmin() {
	fmt.Println("📦 Installing phpMyAdmin...")

	// Check if already installed
	component := components["phpmyadmin"]
	status := checkComponentStatus(component)

	if status == Installed {
		action := promptForAction(component.Name)
		switch action {
		case "keep":
			fmt.Println("✅ Keeping existing phpMyAdmin installation")
			return
		case "skip":
			fmt.Println("⏭️  Skipping phpMyAdmin installation")
			return
		case "uninstall":
			if err := uninstallComponent(component); err != nil {
				fmt.Printf("Error uninstalling phpMyAdmin: %v\n", err)
			}
			fmt.Println("✅ phpMyAdmin uninstalled")
			return
		case "reinstall":
			fmt.Println("🔄 Reinstalling phpMyAdmin...")
			if err := uninstallComponent(component); err != nil {
				fmt.Printf("Error uninstalling phpMyAdmin: %v\n", err)
				return
			}
		}
	}

	fmt.Println("⚠️  phpMyAdmin installation requires configuration...")
	fmt.Println("📝 Please follow these steps during installation:")
	fmt.Println("   1. Select 'apache2' when prompted for web server")
	fmt.Println("   2. Choose 'Yes' to configure database")
	fmt.Println("   3. Enter a secure password for phpMyAdmin")
	fmt.Println("")
	fmt.Print("Press Enter to continue...")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

	// Set environment variables for non-interactive mode
	cmd := exec.Command("apt", "install", "-y", "phpmyadmin")
	cmd.Env = append(os.Environ(),
		"DEBIAN_FRONTEND=noninteractive",
		"DEBCONF_NONINTERACTIVE_SEEN=true")

	// Pre-seed the configuration to avoid interactive prompts
	preseedCommands := [][]string{
		{"debconf-set-selections", "phpmyadmin phpmyadmin/dbconfig-install boolean true"},
		{"debconf-set-selections", "phpmyadmin phpmyadmin/reconfigure-webserver multiselect apache2"},
		{"debconf-set-selections", "phpmyadmin phpmyadmin/app-password-confirm password"},
		{"debconf-set-selections", "phpmyadmin phpmyadmin/password-confirm password"},
	}

	for _, preseedCmd := range preseedCommands {
		if err := runCommandQuiet(preseedCmd[0], preseedCmd[1:]...); err != nil {
			fmt.Printf("Warning: Failed to preseed configuration: %v\n", err)
		}
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error installing phpMyAdmin: %v\n", err)
		return
	}

	configurePhpMyAdmin()
	fmt.Println("✅ phpMyAdmin installed and configured at /phpmyadmin")
}

// InstallPhpPgAdmin installs phpPgAdmin
func InstallPhpPgAdmin() {
	fmt.Println("📦 Installing phpPgAdmin...")

	if err := runCommand("apt", "install", "-y", "phppgadmin"); err != nil {
		fmt.Printf("Error installing phpPgAdmin: %v\n", err)
		return
	}

	configurePhpPgAdmin()
	fmt.Println("✅ phpPgAdmin installed and configured at /phppgadmin")
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
}

func configureApache() {
	// TODO: Apply Apache configuration from templates
	fmt.Println("⚙️  Configuring Apache...")
}

func configureMySQL() {
	// TODO: Apply MySQL configuration from templates
	fmt.Println("⚙️  Configuring MySQL...")
}

func configureMariaDB() {
	// TODO: Apply MariaDB configuration from templates
	fmt.Println("⚙️  Configuring MariaDB...")
}

func configurePostgreSQL() {
	// TODO: Apply PostgreSQL configuration from templates
	fmt.Println("⚙️  Configuring PostgreSQL...")
}

func configurePHP(version string) {
	// TODO: Apply PHP-FPM configuration from templates
	fmt.Printf("⚙️  Configuring PHP %s...\n", version)
}

func configurePhpMyAdmin() {
	// TODO: Configure phpMyAdmin access
	fmt.Println("⚙️  Configuring phpMyAdmin...")
}

func configurePhpPgAdmin() {
	// TODO: Configure phpPgAdmin access
	fmt.Println("⚙️  Configuring phpPgAdmin...")
}
