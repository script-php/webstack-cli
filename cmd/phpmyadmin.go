package cmd

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"webstack-cli/internal/config"
	"webstack-cli/internal/templates"

	"github.com/spf13/cobra"
)

var phpmyadminCmd = &cobra.Command{
	Use:   "phpmyadmin",
	Short: "phpMyAdmin management",
	Long:  `Install, configure, and manage phpMyAdmin for database administration.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use 'webstack phpmyadmin --help' for available commands")
	},
}

var phpmyadminInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install phpMyAdmin",
	Long: `Install phpMyAdmin with automatic configuration.
Usage:
  sudo webstack phpmyadmin install
  sudo webstack phpmyadmin install --php-version 8.2
  sudo webstack phpmyadmin install --version 5.2.1
  sudo webstack phpmyadmin install --version 5.2.1 --php-version 8.2`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}

		version, _ := cmd.Flags().GetString("version")
		phpVersion, _ := cmd.Flags().GetString("php-version")

		installPhpMyAdmin(version, phpVersion)
	},
}

var phpmyadminUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall phpMyAdmin",
	Long: `Remove phpMyAdmin installation.
Usage:
  sudo webstack phpmyadmin uninstall`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("❌ This command requires root privileges (use sudo)")
			return
		}

		uninstallPhpMyAdmin()
	},
}

var phpmyadminStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check phpMyAdmin status",
	Long: `Display phpMyAdmin installation status.
Usage:
  sudo webstack phpmyadmin status`,
	Run: func(cmd *cobra.Command, args []string) {
		showPhpMyAdminStatus()
	},
}

func init() {
	phpmyadminInstallCmd.Flags().StringP("version", "v", "5.2.1", "phpMyAdmin version (e.g., 5.2.1, 5.1.4)")
	phpmyadminInstallCmd.Flags().StringP("php-version", "p", "", "PHP version to use (auto-detect if not specified)")

	rootCmd.AddCommand(phpmyadminCmd)
	phpmyadminCmd.AddCommand(phpmyadminInstallCmd)
	phpmyadminCmd.AddCommand(phpmyadminUninstallCmd)
	phpmyadminCmd.AddCommand(phpmyadminStatusCmd)
}

// Implementation functions
func installPhpMyAdmin(version, phpVersion string) {
	fmt.Println("🚀 Installing phpMyAdmin...")

	// Default version if not specified
	if version == "" {
		version = "5.2.1"
	}

	// Step 1: Detect web server
	webServer := detectWebServer()
	if webServer == "" {
		fmt.Println("❌ No web server (Nginx/Apache) detected on port 80")
		fmt.Println("   Please install Nginx or Apache first")
		return
	}
	fmt.Printf("✓ Detected web server: %s\n", webServer)

	// Step 2: Detect/validate PHP versions
	installedVersions := getInstalledPhpVersions()
	if len(installedVersions) == 0 {
		fmt.Println("❌ No PHP-FPM versions installed")
		return
	}

	if phpVersion == "" {
		phpVersion = installedVersions[0] // Use first (usually latest)
		fmt.Printf("✓ Auto-selected PHP version: %s\n", phpVersion)
	} else {
		found := false
		for _, v := range installedVersions {
			if v == phpVersion {
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("❌ PHP %s is not installed\n", phpVersion)
			fmt.Println("   Available versions:")
			for _, v := range installedVersions {
				fmt.Printf("   - %s\n", v)
			}
			return
		}
		fmt.Printf("✓ Using PHP version: %s\n", phpVersion)
	}

	// Step 3: Setup directories
	fmt.Println("📁 Setting up directories...")
	phpmyadminPath := "/var/www/phpmyadmin"
	exec.Command("rm", "-rf", phpmyadminPath).Run()

	if err := os.MkdirAll(phpmyadminPath, 0755); err != nil {
		fmt.Printf("❌ Failed to create directory: %v\n", err)
		return
	}
	exec.Command("chown", "-R", "www-data:www-data", phpmyadminPath).Run()
	fmt.Println("✓ Directories configured")

	// Step 4: Download and extract phpMyAdmin
	fmt.Printf("⬇️  Downloading phpMyAdmin %s...\n", version)
	if !downloadAndExtractPhpMyAdmin(version, phpmyadminPath) {
		fmt.Println("❌ Failed to download phpMyAdmin")
		fmt.Println("   Make sure curl or wget is installed")
		return
	}
	fmt.Println("✓ phpMyAdmin extracted")

	// Step 5: Generate configuration
	fmt.Println("⚙️  Generating configuration...")
	if !generatePhpMyAdminConfig(phpVersion) {
		fmt.Println("❌ Failed to generate configuration")
		return
	}
	fmt.Println("✓ Configuration generated")

	// Step 6: Deploy web server config
	fmt.Printf("🔧 Configuring %s...\n", webServer)
	if !deployWebServerConfig(webServer, phpVersion) {
		fmt.Println("❌ Failed to deploy web server configuration")
		return
	}
	fmt.Println("✓ Web server configured")

	// Step 7: Reload web server
	fmt.Printf("🔄 Reloading %s...\n", webServer)
	if !reloadWebServer(webServer) {
		fmt.Println("⚠️  Warning: Could not reload web server")
	} else {
		fmt.Println("✓ Web server reloaded")
	}

	// Success message
	fmt.Println("\n" + strings.Repeat("═", 70))
	fmt.Println("✅ phpMyAdmin installed successfully!")
	fmt.Println("   Access it at: http://YOUR_SERVER_IP/phpmyadmin")
	fmt.Println("   or           http://localhost/phpmyadmin")
	fmt.Println(strings.Repeat("═", 70))
}

func uninstallPhpMyAdmin() {
	fmt.Println("🗑️  Removing phpMyAdmin...")

	phpmyadminPath := "/var/www/phpmyadmin"
	if err := os.RemoveAll(phpmyadminPath); err != nil {
		fmt.Printf("❌ Failed to remove %s: %v\n", phpmyadminPath, err)
		return
	}

	// Remove web server configs
	webServer := detectWebServer()
	if webServer == "nginx" {
		os.Remove("/etc/nginx/includes/phpmyadmin.conf")
	} else if webServer == "apache" {
		os.Remove("/etc/apache2/includes/phpmyadmin.conf")
		exec.Command("a2disconf", "phpmyadmin").Run()
	}

	// Reload web server
	reloadWebServer(webServer)

	fmt.Println("✅ phpMyAdmin uninstalled successfully")
}

func showPhpMyAdminStatus() {
	fmt.Println("📊 phpMyAdmin Status")
	fmt.Println("─────────────────────────────────────────")

	phpmyadminPath := "/var/www/phpmyadmin"
	if _, err := os.Stat(phpmyadminPath); err != nil {
		fmt.Println("❌ phpMyAdmin: Not installed")
		return
	}

	fmt.Println("✅ phpMyAdmin: Installed")
	fmt.Printf("   Location: %s\n", phpmyadminPath)

	// Check config
	if _, err := os.Stat(filepath.Join(phpmyadminPath, "config.inc.php")); err == nil {
		fmt.Println("   Configuration: ✓ Configured")
	}

	// Detect web server
	webServer := detectWebServer()
	if webServer != "" {
		fmt.Printf("   Web Server: %s\n", webServer)
	}

	// Detect PHP version
	phpVersions := getInstalledPhpVersions()
	if len(phpVersions) > 0 {
		fmt.Printf("   Available PHP versions: %s\n", strings.Join(phpVersions, ", "))
	}
}

// Helper functions

func detectWebServer() string {
	// Check if Nginx is installed and running
	if output, err := exec.Command("systemctl", "is-active", "nginx").Output(); err == nil && strings.TrimSpace(string(output)) == "active" {
		return "nginx"
	}

	// Check if Apache is installed and running
	if output, err := exec.Command("systemctl", "is-active", "apache2").Output(); err == nil && strings.TrimSpace(string(output)) == "active" {
		return "apache"
	}

	return ""
}

func getInstalledPhpVersions() []string {
	var versions []string

	// Check for PHP-FPM pools
	poolDir := "/etc/php"
	if entries, err := os.ReadDir(poolDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				version := entry.Name()
				// Check if pool.d exists
				poolPath := filepath.Join(poolDir, entry.Name(), "fpm", "pool.d")
				if _, err := os.Stat(poolPath); err == nil {
					versions = append(versions, version)
				}
			}
		}
	}

	// Return in reverse to get newest first
	if len(versions) > 0 {
		return versions
	}

	return []string{}
}

func downloadAndExtractPhpMyAdmin(version, targetPath string) bool {
	// Map versions to download URLs
	versionMap := map[string]string{
		"5.2.1": "https://files.phpmyadmin.net/phpMyAdmin/5.2.1/phpMyAdmin-5.2.1-all-languages.tar.gz",
		"5.2.0": "https://files.phpmyadmin.net/phpMyAdmin/5.2.0/phpMyAdmin-5.2.0-all-languages.tar.gz",
		"5.1.4": "https://files.phpmyadmin.net/phpMyAdmin/5.1.4/phpMyAdmin-5.1.4-all-languages.tar.gz",
		"5.1.3": "https://files.phpmyadmin.net/phpMyAdmin/5.1.3/phpMyAdmin-5.1.3-all-languages.tar.gz",
		"5.0.4": "https://files.phpmyadmin.net/phpMyAdmin/5.0.4/phpMyAdmin-5.0.4-all-languages.tar.gz",
	}

	downloadURL := versionMap[version]
	if downloadURL == "" {
		fmt.Printf("❌ Unsupported phpMyAdmin version: %s\n", version)
		fmt.Println("   Supported versions:")
		for v := range versionMap {
			fmt.Printf("   - %s\n", v)
		}
		return false
	}

	// Create temp directory
	tmpDir := "/tmp/phpmyadmin-download"
	exec.Command("rm", "-rf", tmpDir).Run()
	os.MkdirAll(tmpDir, 0755)

	tarPath := filepath.Join(tmpDir, "phpmyadmin.tar.gz")

	// Download
	cmd := exec.Command("curl", "-L", "-o", tarPath, downloadURL)
	if err := cmd.Run(); err != nil {
		// Fallback to wget
		cmd = exec.Command("wget", "-O", tarPath, downloadURL)
		if err := cmd.Run(); err != nil {
			return false
		}
	}

	// Check file size
	fileInfo, err := os.Stat(tarPath)
	if err != nil || fileInfo.Size() < 1000 {
		return false
	}

	// Extract
	cmd = exec.Command("tar", "-xzf", tarPath, "-C", targetPath, "--strip-components=1")
	if err := cmd.Run(); err != nil {
		return false
	}

	// Set permissions
	exec.Command("chown", "-R", "www-data:www-data", targetPath).Run()
	exec.Command("chmod", "-R", "755", targetPath).Run()

	// Cleanup
	exec.Command("rm", "-rf", tmpDir).Run()

	return true
}

func generatePhpMyAdminConfig(phpVersion string) bool {
	// Load database credentials
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("⚠️  Could not load config, using defaults")
	}

	var dbPassword string
	if cfg != nil {
		if pass, ok := cfg.GetDefault("mysql_root_password", "").(string); ok && pass != "" {
			dbPassword = pass
		} else if pass, ok := cfg.GetDefault("mariadb_root_password", "").(string); ok && pass != "" {
			dbPassword = pass
		}
	}

	// Generate blowfish secret
	blowfishSecret := generateBlowfishSecret()

	// Create config.inc.php
	configContent := fmt.Sprintf(`<?php
// phpMyAdmin Configuration File - Generated by WebStack CLI

// Display errors
$cfg['ShowErrors'] = false;
$cfg['ShowChgPassword'] = true;
$cfg['ShowCreateDb'] = true;

// Server configuration
$i = 1;
$cfg['Servers'][$i]['host'] = 'localhost';
$cfg['Servers'][$i]['port'] = '3306';
$cfg['Servers'][$i]['socket'] = '/var/run/mysqld/mysqld.sock';
$cfg['Servers'][$i]['connect_type'] = 'tcp';
$cfg['Servers'][$i]['compress'] = false;
$cfg['Servers'][$i]['auth_type'] = 'cookie';
$cfg['Servers'][$i]['user'] = ''; // root
$cfg['Servers'][$i]['password'] = ''; // %s
$cfg['Servers'][$i]['extension'] = 'mysqli';

// phpMyAdmin database
$cfg['Servers'][$i]['controluser'] = '';
$cfg['Servers'][$i]['controlpass'] = '';
$cfg['Servers'][$i]['pmadb'] = 'phpmyadmin';
$cfg['Servers'][$i]['bookmarktable'] = 'pma_bookmark';
$cfg['Servers'][$i]['relation'] = 'pma_relation';
$cfg['Servers'][$i]['table_info'] = 'pma_table_info';
$cfg['Servers'][$i]['table_coords'] = 'pma_table_coords';
$cfg['Servers'][$i]['pdf_pages'] = 'pma_pdf_pages';
$cfg['Servers'][$i]['column_info'] = 'pma_column_info';
$cfg['Servers'][$i]['history'] = 'pma_history';
$cfg['Servers'][$i]['recent'] = 'pma_recent';
$cfg['Servers'][$i]['table_uistats'] = 'pma_table_uistats';
$cfg['Servers'][$i]['tracking'] = 'pma_tracking';
$cfg['Servers'][$i]['userconfig'] = 'pma_userconfig';

// General settings
$cfg['blowfish_secret'] = '%s';
$cfg['UploadDir'] = '/var/lib/phpmyadmin/upload';
$cfg['SaveDir'] = '/var/lib/phpmyadmin/save';
$cfg['TempDir'] = '/var/lib/phpmyadmin/tmp';

// Query/History
$cfg['QueryHistoryDB'] = true;
$cfg['QueryHistoryMax'] = 100;

// Size limits
$cfg['MaxRows'] = 25;
$cfg['MaxTableList'] = 250;

?>
`, dbPassword, blowfishSecret)

	configPath := filepath.Join("/var/www/phpmyadmin", "config.inc.php")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return false
	}

	// Create necessary directories
	dirsToCreate := []string{
		"/var/lib/phpmyadmin",
		"/var/lib/phpmyadmin/upload",
		"/var/lib/phpmyadmin/save",
		"/var/lib/phpmyadmin/tmp",
	}
	for _, dir := range dirsToCreate {
		os.MkdirAll(dir, 0755)
		exec.Command("chown", "-R", "www-data:www-data", dir).Run()
	}

	return true
}

func deployWebServerConfig(webServer, phpVersion string) bool {
	if webServer == "nginx" {
		return deployNginxConfig(phpVersion)
	} else if webServer == "apache" {
		return deployApacheConfig(phpVersion)
	}
	return false
}

func deployNginxConfig(phpVersion string) bool {
	// Get nginx phpmyadmin template
	templateContent, err := templates.GetNginxTemplate("phpmyadmin.conf")
	if err != nil {
		fmt.Printf("⚠️  Could not read nginx template: %v\n", err)
		return false
	}

	// Parse and execute template
	tmpl, err := template.New("phpmyadmin").Parse(string(templateContent))
	if err != nil {
		fmt.Printf("⚠️  Could not parse nginx template: %v\n", err)
		return false
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, map[string]interface{}{
		"PHPVersion": phpVersion,
	})
	if err != nil {
		fmt.Printf("⚠️  Could not execute nginx template: %v\n", err)
		return false
	}

	// Write to includes directory
	configPath := "/etc/nginx/includes/phpmyadmin.conf"
	if err := os.WriteFile(configPath, []byte(buf.String()), 0644); err != nil {
		return false
	}

	return true
}

func deployApacheConfig(phpVersion string) bool {
	// Get apache phpmyadmin template
	templateContent, err := templates.GetApacheTemplate("phpmyadmin.conf")
	if err != nil {
		fmt.Printf("⚠️  Could not read apache template: %v\n", err)
		return false
	}

	// Parse and execute template
	tmpl, err := template.New("phpmyadmin").Parse(string(templateContent))
	if err != nil {
		fmt.Printf("⚠️  Could not parse apache template: %v\n", err)
		return false
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, map[string]interface{}{
		"PHPVersion": phpVersion,
	})
	if err != nil {
		fmt.Printf("⚠️  Could not execute apache template: %v\n", err)
		return false
	}

	// Write to includes directory
	configPath := "/etc/apache2/includes/phpmyadmin.conf"
	if err := os.WriteFile(configPath, []byte(buf.String()), 0644); err != nil {
		return false
	}

	// Enable required modules
	exec.Command("a2enmod", "proxy").Run()
	exec.Command("a2enmod", "proxy_fcgi").Run()
	exec.Command("a2enmod", "alias").Run()

	return true
}

func reloadWebServer(webServer string) bool {
	if webServer == "nginx" {
		if err := exec.Command("nginx", "-t").Run(); err != nil {
			return false
		}
		return exec.Command("systemctl", "reload", "nginx").Run() == nil
	} else if webServer == "apache" {
		if err := exec.Command("apache2ctl", "configtest").Run(); err != nil {
			return false
		}
		return exec.Command("systemctl", "reload", "apache2").Run() == nil
	}
	return false
}

func generateBlowfishSecret() string {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "0123456789ABCDEFGHIJKLMNOPQRSTUV"
	}
	encoded := base64.StdEncoding.EncodeToString(randomBytes)
	if len(encoded) > 32 {
		return encoded[:32]
	}
	return encoded
}
