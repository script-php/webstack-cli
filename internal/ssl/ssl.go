package ssl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SSLCertificate represents an SSL certificate
type SSLCertificate struct {
	Domain    string    `json:"domain"`
	Email     string    `json:"email"`
	Enabled   bool      `json:"enabled"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	CertPath  string    `json:"cert_path"`
	KeyPath   string    `json:"key_path"`
}

const sslConfigFile = "/etc/webstack/ssl.json"

// Enable creates and enables SSL certificate for a domain
func Enable(domainName, email string) {
	fmt.Printf("Enabling SSL for domain: %s\n", domainName)

	// Check if domain exists
	if !domainExists(domainName) {
		fmt.Printf("Domain %s is not configured. Please add the domain first.\n", domainName)
		return
	}

	// Prompt for email if not provided
	if email == "" {
		email = promptEmail()
	}

	if email == "" {
		fmt.Println("Email is required for Let's Encrypt registration")
		return
	}

	// Install certbot if not installed
	if err := ensureCertbotInstalled(); err != nil {
		fmt.Printf("Error installing certbot: %v\n", err)
		return
	}

	// Stop web servers temporarily for standalone mode
	fmt.Println("⚙️  Temporarily stopping web servers...")
	stopWebServers()

	// Request certificate
	fmt.Println("🔒 Requesting SSL certificate...")
	certPath, keyPath, err := requestCertificate(domainName, email)
	if err != nil {
		fmt.Printf("Error requesting certificate: %v\n", err)
		startWebServers()
		return
	}

	// Start web servers again
	startWebServers()

	// Save SSL configuration
	cert := SSLCertificate{
		Domain:    domainName,
		Email:     email,
		Enabled:   true,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().AddDate(0, 3, 0), // 3 months
		CertPath:  certPath,
		KeyPath:   keyPath,
	}

	if err := saveSSLCert(cert); err != nil {
		fmt.Printf("Error saving SSL configuration: %v\n", err)
		return
	}

	// Update domain configuration to use SSL
	if err := enableSSLForDomain(domainName); err != nil {
		fmt.Printf("Error updating domain configuration: %v\n", err)
		return
	}

	// Generate SSL-enabled configuration
	if err := generateSSLConfig(domainName); err != nil {
		fmt.Printf("Error generating SSL configuration: %v\n", err)
		return
	}

	// Reload web servers
	reloadWebServers()

	fmt.Printf("✅ SSL enabled successfully for %s\n", domainName)
	fmt.Printf("   Certificate: %s\n", certPath)
	fmt.Printf("   Private Key: %s\n", keyPath)
}

// Disable removes SSL certificate for a domain
func Disable(domainName string) {
	fmt.Printf("Disabling SSL for domain: %s\n", domainName)

	certs, err := loadSSLCerts()
	if err != nil {
		fmt.Printf("Error loading SSL certificates: %v\n", err)
		return
	}

	found := false
	for i, cert := range certs {
		if cert.Domain == domainName {
			found = true
			certs[i].Enabled = false

			// Save updated configuration
			if err := saveSSLCerts(certs); err != nil {
				fmt.Printf("Error saving SSL configuration: %v\n", err)
				return
			}

			// Update domain configuration to disable SSL
			if err := disableSSLForDomain(domainName); err != nil {
				fmt.Printf("Error updating domain configuration: %v\n", err)
				return
			}

			// Generate non-SSL configuration
			if err := generateNonSSLConfig(domainName); err != nil {
				fmt.Printf("Error generating configuration: %v\n", err)
				return
			}

			reloadWebServers()

			fmt.Printf("✅ SSL disabled for %s\n", domainName)
			fmt.Println("Note: Certificate files are preserved for future use")
			break
		}
	}

	if !found {
		fmt.Printf("No SSL certificate found for domain %s\n", domainName)
	}
}

// Renew renews SSL certificate for a specific domain
func Renew(domainName string) {
	fmt.Printf("Renewing SSL certificate for: %s\n", domainName)

	if err := runCommand("certbot", "renew", "--cert-name", domainName); err != nil {
		fmt.Printf("Error renewing certificate: %v\n", err)
		return
	}

	reloadWebServers()
	fmt.Printf("✅ SSL certificate renewed for %s\n", domainName)
}

// RenewAll renews all SSL certificates
func RenewAll() {
	fmt.Println("Renewing all SSL certificates...")

	if err := runCommand("certbot", "renew"); err != nil {
		fmt.Printf("Error renewing certificates: %v\n", err)
		return
	}

	reloadWebServers()
	fmt.Println("✅ All SSL certificates renewed")
}

// Status shows SSL certificate status for a domain
func Status(domainName string) {
	certs, err := loadSSLCerts()
	if err != nil {
		fmt.Printf("Error loading SSL certificates: %v\n", err)
		return
	}

	for _, cert := range certs {
		if cert.Domain == domainName {
			fmt.Printf("SSL Status for %s:\n", domainName)
			fmt.Printf("  Enabled: %t\n", cert.Enabled)
			fmt.Printf("  Email: %s\n", cert.Email)
			fmt.Printf("  Issued: %s\n", cert.IssuedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Expires: %s\n", cert.ExpiresAt.Format("2006-01-02 15:04:05"))

			daysUntilExpiry := int(time.Until(cert.ExpiresAt).Hours() / 24)
			fmt.Printf("  Days until expiry: %d\n", daysUntilExpiry)

			if daysUntilExpiry <= 30 {
				fmt.Println("  ⚠️  Certificate expires soon!")
			}
			return
		}
	}

	fmt.Printf("No SSL certificate found for domain %s\n", domainName)
}

// StatusAll shows SSL certificate status for all domains
func StatusAll() {
	certs, err := loadSSLCerts()
	if err != nil {
		fmt.Printf("Error loading SSL certificates: %v\n", err)
		return
	}

	if len(certs) == 0 {
		fmt.Println("No SSL certificates configured")
		return
	}

	fmt.Println("SSL Certificate Status:")
	fmt.Println("======================")

	for _, cert := range certs {
		status := "Disabled"
		if cert.Enabled {
			status = "Enabled"
		}

		daysUntilExpiry := int(time.Until(cert.ExpiresAt).Hours() / 24)

		fmt.Printf("Domain: %s\n", cert.Domain)
		fmt.Printf("  Status: %s\n", status)
		fmt.Printf("  Expires: %s (%d days)\n", cert.ExpiresAt.Format("2006-01-02"), daysUntilExpiry)

		if daysUntilExpiry <= 30 && cert.Enabled {
			fmt.Println("  ⚠️  Expires soon!")
		}
		fmt.Println()
	}
}

// Helper functions
func promptEmail() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter email address for Let's Encrypt: ")

	response, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}

	return strings.TrimSpace(response)
}

func domainExists(domainName string) bool {
	// TODO: Check if domain exists in domain configuration
	return true
}

func ensureCertbotInstalled() error {
	// Check if certbot is installed
	if err := runCommand("which", "certbot"); err != nil {
		fmt.Println("📦 Installing certbot...")

		// Install snapd if not available
		if err := runCommand("apt", "update"); err != nil {
			return err
		}

		if err := runCommand("apt", "install", "-y", "snapd"); err != nil {
			return err
		}

		// Install certbot via snap
		if err := runCommand("snap", "install", "--classic", "certbot"); err != nil {
			return err
		}

		// Create symlink
		if err := runCommand("ln", "-sf", "/snap/bin/certbot", "/usr/bin/certbot"); err != nil {
			return err
		}
	}

	return nil
}

func requestCertificate(domainName, email string) (string, string, error) {
	// Use certbot standalone mode
	args := []string{
		"certonly",
		"--standalone",
		"--non-interactive",
		"--agree-tos",
		"--email", email,
		"-d", domainName,
	}

	if err := runCommand("certbot", args...); err != nil {
		return "", "", err
	}

	certPath := fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", domainName)
	keyPath := fmt.Sprintf("/etc/letsencrypt/live/%s/privkey.pem", domainName)

	return certPath, keyPath, nil
}

func stopWebServers() {
	runCommand("systemctl", "stop", "nginx")
	runCommand("systemctl", "stop", "apache2")
}

func startWebServers() {
	runCommand("systemctl", "start", "nginx")
	runCommand("systemctl", "start", "apache2")
}

func reloadWebServers() {
	runCommand("systemctl", "reload", "nginx")
	runCommand("systemctl", "reload", "apache2")
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func loadSSLCerts() ([]SSLCertificate, error) {
	var certs []SSLCertificate

	if _, err := os.Stat(sslConfigFile); os.IsNotExist(err) {
		// Create directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(sslConfigFile), 0755); err != nil {
			return nil, err
		}
		return certs, nil
	}

	data, err := ioutil.ReadFile(sslConfigFile)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &certs); err != nil {
		return nil, err
	}

	return certs, nil
}

func saveSSLCert(cert SSLCertificate) error {
	certs, err := loadSSLCerts()
	if err != nil {
		return err
	}

	// Check if certificate already exists
	for i, c := range certs {
		if c.Domain == cert.Domain {
			certs[i] = cert
			return saveSSLCerts(certs)
		}
	}

	// Add new certificate
	certs = append(certs, cert)
	return saveSSLCerts(certs)
}

func saveSSLCerts(certs []SSLCertificate) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(sslConfigFile), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(certs, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(sslConfigFile, data, 0644)
}

func enableSSLForDomain(domainName string) error {
	// TODO: Update domain configuration to enable SSL
	fmt.Printf("⚙️  Enabling SSL in domain configuration for %s...\n", domainName)
	return nil
}

func disableSSLForDomain(domainName string) error {
	// TODO: Update domain configuration to disable SSL
	fmt.Printf("⚙️  Disabling SSL in domain configuration for %s...\n", domainName)
	return nil
}

func generateSSLConfig(domainName string) error {
	// TODO: Generate SSL-enabled configuration from templates
	fmt.Printf("⚙️  Generating SSL configuration for %s...\n", domainName)
	return nil
}

func generateNonSSLConfig(domainName string) error {
	// TODO: Generate non-SSL configuration from templates
	fmt.Printf("⚙️  Generating non-SSL configuration for %s...\n", domainName)
	return nil
}
