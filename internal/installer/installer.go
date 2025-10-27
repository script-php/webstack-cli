package installer

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// InstallAll runs interactive installation of the complete web stack
func InstallAll() {
	fmt.Println("🚀 WebStack Interactive Installation")
	fmt.Println("===================================")

	// Install base components
	fmt.Println("Installing Nginx and Apache...")
	InstallNginx()
	InstallApache()

	// Ask about database
	if askYesNo("Do you want to install MySQL?") {
		InstallMySQL()
		if askYesNo("Do you want to install phpMyAdmin?") {
			InstallPhpMyAdmin()
		}
	} else if askYesNo("Do you want to install MariaDB?") {
		InstallMariaDB()
		if askYesNo("Do you want to install phpMyAdmin?") {
			InstallPhpMyAdmin()
		}
	}

	// Ask about PostgreSQL
	if askYesNo("Do you want to install PostgreSQL?") {
		InstallPostgreSQL()
		if askYesNo("Do you want to install phpPgAdmin?") {
			InstallPhpPgAdmin()
		}
	}

	// Install PHP versions
	fmt.Println("Installing PHP-FPM versions...")
	phpVersions := []string{"5.6", "7.0", "7.1", "7.2", "7.3", "7.4", "8.0", "8.1", "8.2", "8.3", "8.4"}

	for _, version := range phpVersions {
		if askYesNo(fmt.Sprintf("Install PHP %s?", version)) {
			InstallPHP(version)
		}
	}

	fmt.Println("✅ Installation completed!")
}

// InstallNginx installs and configures Nginx on port 80
func InstallNginx() {
	fmt.Println("📦 Installing Nginx...")

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

	if err := runCommand("apt", "install", "-y", "phpmyadmin"); err != nil {
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

func askYesNo(question string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s (y/N): ", question)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// Configuration functions (will be implemented)
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
