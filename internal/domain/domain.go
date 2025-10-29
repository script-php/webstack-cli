package domain

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"webstack-cli/internal/config"
	"webstack-cli/internal/templates"
)

// Domain represents a domain configuration
type Domain struct {
	Name         string `json:"name"`
	Backend      string `json:"backend"` // "nginx" or "apache"
	PHPVersion   string `json:"php_version"`
	DocumentRoot string `json:"document_root"`
	SSLEnabled   bool   `json:"ssl_enabled"`
	SSLCertPath  string `json:"ssl_cert_path,omitempty"`  // Path to SSL certificate
	SSLKeyPath   string `json:"ssl_key_path,omitempty"`   // Path to SSL private key
	SSLEmail     string `json:"ssl_email,omitempty"`      // Email used for Let's Encrypt
}

const domainsFile = "/etc/webstack/domains.json"

// Add creates a new domain configuration
func Add(domainName, backend, phpVersion string) {
	fmt.Printf("Adding domain: %s\n", domainName)

	// Interactive prompts if flags not provided
	if backend == "" {
		backend = promptBackend()
	}

	if phpVersion == "" {
		phpVersion = promptPHPVersion()
	}

	// Validate inputs
	if !isValidBackend(backend) {
		fmt.Printf("Invalid backend: %s. Must be 'nginx' or 'apache'\n", backend)
		return
	}

	if !isValidPHPVersion(phpVersion) {
		fmt.Printf("Invalid PHP version: %s\n", phpVersion)
		return
	}

	// Set up domain directory structure
	baseDir := fmt.Sprintf("/var/www/%s", domainName)
	htdocsDir := filepath.Join(baseDir, "htdocs")
	
	domain := Domain{
		Name:         domainName,
		Backend:      backend,
		PHPVersion:   phpVersion,
		DocumentRoot: htdocsDir, // Point to htdocs as the web root
		SSLEnabled:   false,
	}

	// Create directory structure: /var/www/domain/{ htdocs, logs, configs, error }
	dirs := []string{
		htdocsDir,
		filepath.Join(baseDir, "logs"),
		filepath.Join(baseDir, "configs"),
		filepath.Join(baseDir, "error"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("Error creating directory %s: %v\n", dir, err)
			return
		}
	}

	fmt.Printf("📁 Created domain directory structure:\n")
	fmt.Printf("   %s/htdocs     - Web root (public files)\n", baseDir)
	fmt.Printf("   %s/logs       - Log files\n", baseDir)
	fmt.Printf("   %s/configs    - Additional nginx configurations\n", baseDir)
	fmt.Printf("   %s/error      - Error pages symlink\n", baseDir)

	// Create default index.php
	createDefaultIndex(domain.DocumentRoot, domainName, phpVersion)

	// Create error folder (error pages served from /etc/webstack/error/)
	os.MkdirAll(filepath.Join(baseDir, "error"), 0755)

	// Save domain configuration
	if err := saveDomain(domain); err != nil {
		fmt.Printf("Error saving domain: %v\n", err)
		return
	}

	// Generate web server configuration
	if err := generateConfig(domain); err != nil {
		fmt.Printf("Error generating configuration: %v\n", err)
		return
	}

	// Reload web servers
	reloadWebServers()

	fmt.Printf("✅ Domain %s added successfully\n", domainName)
	fmt.Printf("   Backend: %s\n", backend)
	fmt.Printf("   PHP Version: %s\n", phpVersion)
	fmt.Printf("   Document Root: %s\n", domain.DocumentRoot)
}

// Edit modifies an existing domain configuration
func Edit(domainName, backend, phpVersion string) {
	fmt.Printf("Editing domain: %s\n", domainName)

	domains, err := loadDomains()
	if err != nil {
		fmt.Printf("Error loading domains: %v\n", err)
		return
	}

	found := false
	for i, domain := range domains {
		if domain.Name == domainName {
			found = true

			// Update backend if provided
			if backend != "" {
				if !isValidBackend(backend) {
					fmt.Printf("Invalid backend: %s\n", backend)
					return
				}
				domains[i].Backend = backend
			}

			// Update PHP version if provided
			if phpVersion != "" {
				if !isValidPHPVersion(phpVersion) {
					fmt.Printf("Invalid PHP version: %s\n", phpVersion)
					return
				}
				domains[i].PHPVersion = phpVersion
			}

			// Interactive prompts if no flags provided
			if backend == "" && phpVersion == "" {
				fmt.Printf("Current backend: %s\n", domain.Backend)
				newBackend := promptBackend()
				if newBackend != domain.Backend {
					domains[i].Backend = newBackend
				}

				fmt.Printf("Current PHP version: %s\n", domain.PHPVersion)
				newPHP := promptPHPVersion()
				if newPHP != domain.PHPVersion {
					domains[i].PHPVersion = newPHP
				}
			}

			// Save updated configuration
			if err := saveDomains(domains); err != nil {
				fmt.Printf("Error saving domains: %v\n", err)
				return
			}

			// Regenerate configuration
			if err := generateConfig(domains[i]); err != nil {
				fmt.Printf("Error generating configuration: %v\n", err)
				return
			}

			reloadWebServers()

			fmt.Printf("✅ Domain %s updated successfully\n", domainName)
			break
		}
	}

	if !found {
		fmt.Printf("Domain %s not found\n", domainName)
	}
}

// Delete removes a domain configuration
func Delete(domainName string) {
	fmt.Printf("Deleting domain: %s\n", domainName)

	domains, err := loadDomains()
	if err != nil {
		fmt.Printf("Error loading domains: %v\n", err)
		return
	}

	found := false
	for i, domain := range domains {
		if domain.Name == domainName {
			found = true

			// Remove configuration files
			removeConfig(domain)

			// Ask if user wants to delete the domain folder
			baseDir := filepath.Join("/var/www", domainName)
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("Delete domain folder %s? (y/N): ", baseDir)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response == "y" || response == "yes" {
				// Delete the entire domain folder
				if err := os.RemoveAll(baseDir); err != nil {
					fmt.Printf("⚠️  Warning: Could not delete domain folder: %v\n", err)
				} else {
					fmt.Printf("✅ Domain folder deleted: %s\n", baseDir)
				}
			} else {
				fmt.Printf("ℹ️  Domain folder preserved: %s\n", baseDir)
				fmt.Printf("   Contains: htdocs/, logs/, configs/, error/\n")
			}

			// Remove from domains slice
			domains = append(domains[:i], domains[i+1:]...)

			// Save updated domains
			if err := saveDomains(domains); err != nil {
				fmt.Printf("Error saving domains: %v\n", err)
				return
			}

			reloadWebServers()

			fmt.Printf("✅ Domain %s deleted successfully\n", domainName)
			break
		}
	}

	if !found {
		fmt.Printf("Domain %s not found\n", domainName)
	}
}

// List displays all configured domains
func List() {
	domains, err := loadDomains()
	if err != nil {
		fmt.Printf("Error loading domains: %v\n", err)
		return
	}

	if len(domains) == 0 {
		fmt.Println("No domains configured")
		return
	}

	fmt.Println("Configured Domains:")
	fmt.Println("===================")
	for _, domain := range domains {
		sslStatus := "No"
		if domain.SSLEnabled {
			sslStatus = "Yes"
		}
		fmt.Printf("Domain: %s\n", domain.Name)
		fmt.Printf("  Backend: %s\n", domain.Backend)
		fmt.Printf("  PHP Version: %s\n", domain.PHPVersion)
		fmt.Printf("  Document Root: %s\n", domain.DocumentRoot)
		fmt.Printf("  SSL: %s\n", sslStatus)
		fmt.Println()
	}
}

// RebuildConfigs regenerates configuration files for all domains
func RebuildConfigs() {
	fmt.Println("🔄 Rebuilding all domain configurations...")
	fmt.Println("==========================================")

	domains, err := loadDomains()
	if err != nil {
		fmt.Printf("Error loading domains: %v\n", err)
		return
	}

	if len(domains) == 0 {
		fmt.Println("No domains configured")
		return
	}

	successCount := 0
	errorCount := 0

	for _, domain := range domains {
		fmt.Printf("\n📝 Rebuilding config for %s (%s)...\n", domain.Name, domain.Backend)

		// Remove old configs
		removeConfig(domain)

		// Generate new configs
		if err := generateConfig(domain); err != nil {
			fmt.Printf("❌ Error generating configuration for %s: %v\n", domain.Name, err)
			errorCount++
		} else {
			fmt.Printf("✅ Configuration rebuilt for %s\n", domain.Name)
			successCount++
		}
	}

	// Reload web servers once after all configs are regenerated
	reloadWebServers()

	fmt.Println("\n==========================================")
	fmt.Printf("✅ Rebuilt: %d domain(s)\n", successCount)
	if errorCount > 0 {
		fmt.Printf("❌ Failed: %d domain(s)\n", errorCount)
	}
}

// Helper functions
func promptBackend() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Choose backend (nginx/apache) [nginx]: ")

	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)

	if response == "" {
		return "nginx"
	}

	return strings.ToLower(response)
}

func promptPHPVersion() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Choose PHP version (5.6-8.4) [8.2]: ")

	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)

	if response == "" {
		return "8.2"
	}

	return response
}

func isValidBackend(backend string) bool {
	return backend == "nginx" || backend == "apache"
}

func isValidPHPVersion(version string) bool {
	validVersions := []string{"5.6", "7.0", "7.1", "7.2", "7.3", "7.4", "8.0", "8.1", "8.2", "8.3", "8.4"}
	for _, v := range validVersions {
		if v == version {
			return true
		}
	}
	return false
}

func createDefaultIndex(docRoot, domainName, phpVersion string) {
	indexContent := fmt.Sprintf(`<?php
echo "<h1>Welcome to %s</h1>";
echo "<p>PHP Version: " . phpversion() . "</p>";
echo "<p>Expected PHP: %s</p>";
echo "<p>Server: " . $_SERVER['SERVER_SOFTWARE'] . "</p>";
phpinfo();
?>`, domainName, phpVersion)

	indexPath := filepath.Join(docRoot, "index.php")
	if err := ioutil.WriteFile(indexPath, []byte(indexContent), 0644); err != nil {
		fmt.Printf("Warning: Could not create index.php: %v\n", err)
	}
}

func loadDomains() ([]Domain, error) {
	var domains []Domain

	if _, err := os.Stat(domainsFile); os.IsNotExist(err) {
		// Create directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(domainsFile), 0755); err != nil {
			return nil, err
		}
		// Return empty slice if file doesn't exist
		return domains, nil
	}

	data, err := ioutil.ReadFile(domainsFile)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &domains); err != nil {
		return nil, err
	}

	return domains, nil
}

func saveDomain(domain Domain) error {
	domains, err := loadDomains()
	if err != nil {
		return err
	}

	// Check if domain already exists
	for i, d := range domains {
		if d.Name == domain.Name {
			domains[i] = domain
			return saveDomains(domains)
		}
	}

	// Add new domain
	domains = append(domains, domain)
	return saveDomains(domains)
}

func saveDomains(domains []Domain) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(domainsFile), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(domains, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(domainsFile, data, 0644)
}

func GenerateConfig(d Domain) error {
	return generateConfig(d)
}

func generateConfig(domain Domain) error {
	fmt.Printf("⚙️  Generating configuration for %s...\n", domain.Name)

	// Load server config to determine ports and modes
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("could not load server config: %v", err)
	}
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// Get template variables
	templateVars := map[string]interface{}{
		"Domain":       domain.Name,
		"DocumentRoot": domain.DocumentRoot,
		"PHPVersion":   strings.Split(domain.PHPVersion, ".")[0] + domain.PHPVersion[strings.LastIndex(domain.PHPVersion, "."):],
		"PHPSocket":    fmt.Sprintf("unix:/run/php/php%s-fpm.sock", domain.PHPVersion),
		"ApachePort":   cfg.GetPort("apache"), // Get Apache port from config
	}

	// If SSL is enabled for this domain, try to include certificate paths and use SSL templates
	useSSL := false
	if domain.SSLEnabled {
		certPath, keyPath, err := loadSSLCertPaths(domain.Name)
		if err != nil {
			// SSL is enabled but cert paths are missing or empty
			// Fall back to non-SSL config and warn user
			fmt.Printf("⚠️  SSL enabled but certificate paths missing for %s. Generating non-SSL config.\n", domain.Name)
			fmt.Printf("    Reason: %v\n", err)
		} else {
			// Add cert paths to template variables
			templateVars["SSLCert"] = certPath
			templateVars["SSLKey"] = keyPath
			useSSL = true
		}
	}

	if useSSL {
		// SSL-enabled paths
		if domain.Backend == "nginx" {
			if err := generateNginxConfig(domain.Name, templateVars, "domain-ssl"); err != nil {
				return err
			}
		} else if domain.Backend == "apache" {
			// For Apache backend, check if Nginx is in proxy mode
			nginxMode := cfg.GetMode("nginx")
			if nginxMode == "proxy" {
				// Nginx will proxy to Apache (proxy-ssl)
				if err := generateNginxConfig(domain.Name, templateVars, "proxy-ssl"); err != nil {
					return err
				}
				// Still need to generate Apache config for Nginx to proxy to
				if err := generateApacheConfig(domain.Name, templateVars); err != nil {
					return err
				}
			} else if !cfg.IsInstalled("nginx") || nginxMode == "standalone" {
				// Generate Apache config for standalone mode
				if err := generateApacheConfig(domain.Name, templateVars); err != nil {
					return err
				}
			}
		}
	} else {
		// Non-SSL paths
		if domain.Backend == "nginx" {
			// Direct Nginx backend
			if err := generateNginxConfig(domain.Name, templateVars, "domain"); err != nil {
				return err
			}
		} else if domain.Backend == "apache" {
			// For Apache backend, check server configuration
			nginxMode := cfg.GetMode("nginx")
			if nginxMode == "proxy" {
				// Nginx is in proxy mode, generate proxy config
				if err := generateNginxConfig(domain.Name, templateVars, "proxy"); err != nil {
					return err
				}
				// Still need to generate Apache config for Nginx to proxy to
				if err := generateApacheConfig(domain.Name, templateVars); err != nil {
					return err
				}
			} else if !cfg.IsInstalled("nginx") || nginxMode == "standalone" {
				// Generate Apache config for standalone mode
				if err := generateApacheConfig(domain.Name, templateVars); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func generateNginxConfig(domainName string, vars map[string]interface{}, configType string) error {
	// configType can be "domain" (direct PHP-FPM) or "proxy" (Apache reverse proxy)

	// Read template from embedded filesystem
	templateFilename := "domain.conf"
	if configType == "proxy" {
		templateFilename = "proxy.conf"
	} else if configType == "domain-ssl" {
		templateFilename = "domain-ssl.conf"
	} else if configType == "proxy-ssl" {
		templateFilename = "proxy-ssl.conf"
	}

	content, err := templates.GetNginxTemplate(templateFilename)
	if err != nil {
		return fmt.Errorf("could not read nginx template (%s): %v", templateFilename, err)
	}

	// Parse template
	tmpl, err := template.New("nginx").Parse(string(content))
	if err != nil {
		return fmt.Errorf("could not parse nginx template: %v", err)
	}

	// Render into buffer so we can post-process (e.g., remove fastcgi_cache lines if main nginx.conf doesn't define them)
	var buf strings.Builder
	if err := tmpl.Execute(&buf, vars); err != nil {
		return fmt.Errorf("could not execute nginx template: %v", err)
	}
	rendered := buf.String()

	// If the running nginx configuration doesn't define a fastcgi_cache zone, strip fastcgi_cache lines from rendered output
	if data, err := ioutil.ReadFile("/etc/nginx/nginx.conf"); err == nil {
		if !strings.Contains(string(data), "fastcgi_cache_path") && strings.Contains(rendered, "fastcgi_cache") {
			// Remove any lines related to the FastCGI cache block inserted by template
			outLines := []string{}
			for _, line := range strings.Split(rendered, "\n") {
				// skip cache-related lines
				if strings.Contains(line, "# FastCGI cache settings") || strings.Contains(line, "fastcgi_cache ") || strings.Contains(line, "fastcgi_cache_valid") || strings.Contains(line, "fastcgi_cache_bypass") || strings.Contains(line, "fastcgi_no_cache") || strings.Contains(line, "$no_cache") {
					continue
				}
				outLines = append(outLines, line)
			}
			rendered = strings.Join(outLines, "\n")
		}
	}

	// Ensure sites-available directory exists
	siteDir := "/etc/nginx/sites-available"
	if err := os.MkdirAll(siteDir, 0755); err != nil {
		return fmt.Errorf("could not create nginx sites-available directory: %v", err)
	}

	// Write config file
	configFile := filepath.Join(siteDir, domainName+".conf")
	if err := ioutil.WriteFile(configFile, []byte(rendered), 0644); err != nil {
		return fmt.Errorf("could not write nginx config file: %v", err)
	}

	// Enable site by creating symlink in sites-enabled
	enableDir := "/etc/nginx/sites-enabled"
	if err := os.MkdirAll(enableDir, 0755); err != nil {
		return fmt.Errorf("could not create nginx sites-enabled directory: %v", err)
	}

	enableLink := filepath.Join(enableDir, domainName+".conf")
	os.Remove(enableLink) // Remove existing symlink if it exists
	if err := os.Symlink(configFile, enableLink); err != nil {
		return fmt.Errorf("could not create nginx sites-enabled symlink: %v", err)
	}

	fmt.Printf("✅ Nginx configuration created: %s\n", configFile)
	return nil
}

func generateApacheConfig(domainName string, vars map[string]interface{}) error {
	// Read template from embedded filesystem
	content, err := templates.GetApacheTemplate("domain.conf")
	if err != nil {
		return fmt.Errorf("could not read apache template: %v", err)
	}

	// Parse and execute template
	tmpl, err := template.New("apache").Parse(string(content))
	if err != nil {
		return fmt.Errorf("could not parse apache template: %v", err)
	}

	// Ensure sites-available directory exists
	siteDir := "/etc/apache2/sites-available"
	if err := os.MkdirAll(siteDir, 0755); err != nil {
		return fmt.Errorf("could not create apache sites-available directory: %v", err)
	}

	// Write config file
	configFile := filepath.Join(siteDir, domainName+".conf")
	file, err := os.Create(configFile)
	if err != nil {
		return fmt.Errorf("could not create apache config file: %v", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, vars); err != nil {
		return fmt.Errorf("could not execute apache template: %v", err)
	}

	// Enable site using a2ensite
	cmd := exec.Command("a2ensite", domainName)
	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️  Warning: Could not enable Apache site: %v\n", err)
		// Don't fail, just warn
	}

	// Ensure required Apache modules for php-fpm proxying are enabled
	mods := [][]string{
		{"a2enmod", "proxy_fcgi"},
		{"a2enmod", "proxy"},
		{"a2enmod", "setenvif"},
		{"a2enmod", "remoteip"},
	}
	for _, m := range mods {
		cmd := exec.Command(m[0], m[1])
		if err := cmd.Run(); err != nil {
			fmt.Printf("⚠️  Warning: Could not enable Apache module %s: %v\n", m[1], err)
		}
	}

	fmt.Printf("✅ Apache configuration created: %s\n", configFile)
	return nil
}

func removeConfig(domain Domain) {
	fmt.Printf("⚙️  Removing configuration for %s...\n", domain.Name)

	// Always remove Nginx config (both direct PHP and proxy configs)
	siteAvailablePath := filepath.Join("/etc/nginx/sites-available", domain.Name+".conf")
	siteEnabledPath := filepath.Join("/etc/nginx/sites-enabled", domain.Name+".conf")

	if err := os.Remove(siteAvailablePath); err != nil && !os.IsNotExist(err) {
		fmt.Printf("⚠️  Warning: Could not remove nginx config: %v\n", err)
	}

	if err := os.Remove(siteEnabledPath); err != nil && !os.IsNotExist(err) {
		fmt.Printf("⚠️  Warning: Could not remove nginx symlink: %v\n", err)
	}

	if domain.Backend == "apache" {
		// Disable site using a2dissite
		cmd := exec.Command("a2dissite", domain.Name)
		if err := cmd.Run(); err != nil {
			fmt.Printf("⚠️  Warning: Could not disable Apache site: %v\n", err)
		}

		// Remove apache config file
		apacheSiteAvailablePath := filepath.Join("/etc/apache2/sites-available", domain.Name+".conf")
		if err := os.Remove(apacheSiteAvailablePath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("⚠️  Warning: Could not remove apache config: %v\n", err)
		}

		fmt.Printf("✅ Apache and Nginx configurations removed for %s\n", domain.Name)
	} else {
		fmt.Printf("✅ Nginx configuration removed for %s\n", domain.Name)
	}
}

func reloadWebServers() {
	fmt.Println("⚙️  Reloading web servers...")

	// Reload Nginx
	nginxReloadCmd := exec.Command("systemctl", "reload", "nginx")
	if err := nginxReloadCmd.Run(); err != nil {
		fmt.Printf("⚠️  Warning: Could not reload Nginx: %v\n", err)
	} else {
		fmt.Println("✅ Nginx reloaded")
	}

	// Reload Apache
	apacheReloadCmd := exec.Command("systemctl", "reload", "apache2")
	if err := apacheReloadCmd.Run(); err != nil {
		fmt.Printf("⚠️  Warning: Could not reload Apache: %v\n", err)
	} else {
		fmt.Println("✅ Apache reloaded")
	}
}

// DomainExists checks if a domain exists in the configuration
func DomainExists(domainName string) bool {
	domains, err := loadDomains()
	if err != nil {
		return false
	}

	for _, d := range domains {
		if d.Name == domainName {
			return true
		}
	}
	return false
}

// GetDomain returns a domain by name
func GetDomain(domainName string) (*Domain, error) {
	domains, err := loadDomains()
	if err != nil {
		return nil, err
	}

	for _, d := range domains {
		if d.Name == domainName {
			return &d, nil
		}
	}
	return nil, fmt.Errorf("domain %s not found", domainName)
}

// UpdateDomain updates a domain in the configuration
func UpdateDomain(domain Domain) error {
	return saveDomain(domain)
}

// loadSSLCertPaths loads certificate and key paths for a domain from domains.json
func loadSSLCertPaths(domainName string) (string, string, error) {
	domains, err := loadDomains()
	if err != nil {
		return "", "", fmt.Errorf("could not load domains: %v", err)
	}

	for _, d := range domains {
		if d.Name == domainName && d.SSLEnabled {
			if d.SSLCertPath == "" || d.SSLKeyPath == "" {
				return "", "", fmt.Errorf("domain %s has SSL enabled but cert paths are empty", domainName)
			}
			return d.SSLCertPath, d.SSLKeyPath, nil
		}
	}

	return "", "", fmt.Errorf("no enabled SSL certificate found for %s", domainName)
}

