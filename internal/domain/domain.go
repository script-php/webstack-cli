package domain

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Domain represents a domain configuration
type Domain struct {
	Name         string `json:"name"`
	Backend      string `json:"backend"` // "nginx" or "apache"
	PHPVersion   string `json:"php_version"`
	DocumentRoot string `json:"document_root"`
	SSLEnabled   bool   `json:"ssl_enabled"`
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

	domain := Domain{
		Name:         domainName,
		Backend:      backend,
		PHPVersion:   phpVersion,
		DocumentRoot: fmt.Sprintf("/var/www/%s", domainName),
		SSLEnabled:   false,
	}

	// Create document root
	if err := os.MkdirAll(domain.DocumentRoot, 0755); err != nil {
		fmt.Printf("Error creating document root: %v\n", err)
		return
	}

	// Create default index.php
	createDefaultIndex(domain.DocumentRoot, domainName, phpVersion)

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

			// Remove from domains slice
			domains = append(domains[:i], domains[i+1:]...)

			// Save updated domains
			if err := saveDomains(domains); err != nil {
				fmt.Printf("Error saving domains: %v\n", err)
				return
			}

			reloadWebServers()

			fmt.Printf("✅ Domain %s deleted successfully\n", domainName)
			fmt.Printf("Note: Document root %s was preserved\n", domain.DocumentRoot)
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

func generateConfig(domain Domain) error {
	// TODO: Generate Nginx and Apache configurations from templates
	fmt.Printf("⚙️  Generating configuration for %s...\n", domain.Name)

	// This will be implemented with template processing
	return nil
}

func removeConfig(domain Domain) {
	// TODO: Remove Nginx and Apache configuration files
	fmt.Printf("⚙️  Removing configuration for %s...\n", domain.Name)
}

func reloadWebServers() {
	fmt.Println("⚙️  Reloading web servers...")
	// TODO: Reload Nginx and Apache configurations
}
