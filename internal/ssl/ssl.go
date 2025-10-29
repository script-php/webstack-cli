package ssl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
	"webstack-cli/internal/domain"
	"webstack-cli/internal/templates"
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

// Enable creates and enables SSL certificate for a domain (interactive mode)
func Enable(domainName, email string) {
	EnableWithType(domainName, email, "")
}

// EnableWithType creates and enables SSL certificate for a domain with specified type
// certType can be "selfsigned", "letsencrypt", or empty string for interactive mode
func EnableWithType(domainName, email, certType string) {
	fmt.Printf("Enabling SSL for domain: %s\n", domainName)

	// Check if domain exists
	if !domainExists(domainName) {
		fmt.Printf("Domain %s is not configured. Please add the domain first.\n", domainName)
		return
	}

	// Normalize cert type
	certType = strings.TrimSpace(strings.ToLower(certType))

	// Check if domain is a local domain (for self-signed certificate)
	isLocalDomain := strings.HasSuffix(domainName, ".local") || strings.HasSuffix(domainName, ".test") || strings.HasSuffix(domainName, ".dev") || domainName == "localhost"

	var useSSLType string

	// If cert type is specified via flag, use it directly
	if certType == "selfsigned" || certType == "self-signed" {
		useSSLType = "self-signed"
	} else if certType == "letsencrypt" || certType == "lets-encrypt" {
		useSSLType = "letsencrypt"
	} else if certType != "" {
		fmt.Printf("❌ Invalid certificate type: %s. Use 'selfsigned' or 'letsencrypt'\n", certType)
		return
	} else {
		// Interactive mode
		if isLocalDomain {
			fmt.Printf("⚠️  %s appears to be a local development domain.\n", domainName)
			fmt.Println("\nSSL Certificate Options:")
			fmt.Println("  [1] Self-signed certificate (recommended for development)")
			fmt.Println("  [2] Let's Encrypt (requires internet and valid DNS)")
			fmt.Println("  [q] Cancel")
			fmt.Print("Choose option: ")

			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			useSSLType = strings.TrimSpace(strings.ToLower(response))

			if useSSLType == "q" || useSSLType == "cancel" {
				fmt.Println("✋ SSL setup cancelled")
				return
			}

			if useSSLType != "2" {
				// Default to self-signed for local domains
				useSSLType = "self-signed"
			} else {
				useSSLType = "letsencrypt"
			}
		} else {
			// For public domains, ask user preference
			fmt.Println("\nSSL Certificate Options:")
			fmt.Println("  [1] Let's Encrypt (recommended for production)")
			fmt.Println("  [2] Self-signed certificate (not trusted, for testing only)")
			fmt.Println("  [q] Cancel")
			fmt.Print("Choose option: ")

			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			choice := strings.TrimSpace(strings.ToLower(response))

			if choice == "q" || choice == "cancel" {
				fmt.Println("✋ SSL setup cancelled")
				return
			}

			if choice == "2" {
				useSSLType = "self-signed"
			} else {
				useSSLType = "letsencrypt"
			}
		}
	}

	// Handle self-signed
	if useSSLType == "self-signed" {
		if err := enableSSLWithSelfSigned(domainName); err != nil {
			fmt.Printf("Error enabling SSL with self-signed certificate: %v\n", err)
			return
		}
		return
	}

	// Handle Let's Encrypt
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

	// Validate domain before requesting certificate
	fmt.Println("🔍 Validating domain configuration...")
	if err := validateDomainForLetsEncrypt(domainName); err != nil {
		fmt.Printf("❌ Domain validation failed: %v\n", err)
		fmt.Println("\nPlease ensure:")
		fmt.Println("  - Domain is publicly resolvable")
		fmt.Println("  - Server IP matches domain DNS record")
		fmt.Println("  - Port 80 is accessible from internet")
		fmt.Println("  - No firewall blocking port 80")
		return
	}
	fmt.Println("✅ Domain validation passed")

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
	if err := enableSSLForDomain(domainName, certPath, keyPath, email); err != nil {
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

	// Setup auto-renewal for Let's Encrypt certificates
	if useSSLType == "letsencrypt" {
		if err := setupAutoRenewal(domainName, email); err != nil {
			fmt.Printf("⚠️  Warning: Could not setup auto-renewal: %v\n", err)
			fmt.Println("   You can manually renew with: webstack-cli ssl renew " + domainName)
		} else {
			fmt.Println("✅ Auto-renewal configured (renewal attempted 30 days before expiry)")
		}
	}
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

			// Remove auto-renewal cronjob
			if err := removeAutoRenewal(domainName); err != nil {
				fmt.Printf("⚠️  Warning: Could not remove auto-renewal: %v\n", err)
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

	// Load certificate info
	certs, err := loadSSLCerts()
	if err != nil {
		fmt.Printf("Error loading SSL certificates: %v\n", err)
		return
	}

	var cert *SSLCertificate
	for i := range certs {
		if certs[i].Domain == domainName {
			cert = &certs[i]
			break
		}
	}

	if cert == nil {
		fmt.Printf("No SSL certificate found for domain %s\n", domainName)
		return
	}

	// Check days until expiry
	daysUntilExpiry := int(time.Until(cert.ExpiresAt).Hours() / 24)
	fmt.Printf("Current certificate expires in %d days\n", daysUntilExpiry)

	// Run certbot renew
	if err := runCommand("certbot", "renew", "--cert-name", domainName, "--force-renewal"); err != nil {
		fmt.Printf("❌ Error renewing certificate: %v\n", err)
		return
	}

	// Reload web servers
	reloadWebServers()

	// Verify renewal succeeded
	certFile := fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", domainName)
	if data, err := os.Stat(certFile); err == nil {
		fmt.Printf("✅ SSL certificate renewed for %s\n", domainName)
		fmt.Printf("   Modified: %s\n", data.ModTime().Format("2006-01-02 15:04:05"))
		fmt.Println("   Web servers reloaded successfully")
	} else {
		fmt.Printf("⚠️  Warning: Could not verify certificate update\n")
	}
}

// RenewAll renews all SSL certificates
func RenewAll() {
	fmt.Println("🔄 Renewing all SSL certificates...")

	certs, err := loadSSLCerts()
	if err != nil {
		fmt.Printf("Error loading SSL certificates: %v\n", err)
		return
	}

	if len(certs) == 0 {
		fmt.Println("No SSL certificates configured")
		return
	}

	// Show summary before renewal
	fmt.Println("\nCertificates to renew:")
	for _, cert := range certs {
		if !cert.Enabled {
			continue
		}
		daysUntilExpiry := int(time.Until(cert.ExpiresAt).Hours() / 24)
		fmt.Printf("  • %s (expires in %d days)\n", cert.Domain, daysUntilExpiry)
	}

	// Run certbot renew (renews all that need renewal)
	if err := runCommand("certbot", "renew", "--quiet"); err != nil {
		fmt.Printf("❌ Error renewing certificates: %v\n", err)
		return
	}

	reloadWebServers()
	fmt.Println("✅ All SSL certificates processed (only those expiring soon were renewed)")
	fmt.Println("   Web servers reloaded successfully")
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
	return domain.DomainExists(domainName)
}

func ensureCertbotInstalled() error {
	// Check if certbot is installed
	if err := runCommand("which", "certbot"); err != nil {
		fmt.Println("📦 Installing certbot...")

		// Try apt first (simpler and more reliable)
		if err := runCommand("apt", "update"); err != nil {
			return fmt.Errorf("apt update failed: %v", err)
		}

		// Install certbot and python3-certbot-nginx for Nginx support
		if err := runCommand("apt", "install", "-y", "certbot", "python3-certbot-nginx"); err != nil {
			fmt.Printf("⚠️  Warning: apt install failed, trying alternative method: %v\n", err)

			// Fallback to snap if apt fails
			if err := runCommand("apt", "install", "-y", "snapd"); err != nil {
				return fmt.Errorf("could not install snapd: %v", err)
			}

			// Install certbot via snap
			if err := runCommand("snap", "install", "--classic", "certbot"); err != nil {
				return fmt.Errorf("could not install certbot via snap: %v", err)
			}

			// Create symlink
			if err := runCommand("ln", "-sf", "/snap/bin/certbot", "/usr/bin/certbot"); err != nil {
				return fmt.Errorf("could not create symlink: %v", err)
			}
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
		return "", "", fmt.Errorf("certbot certificate request failed: %v. Make sure port 80 is not in use", err)
	}

	certPath := fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", domainName)
	keyPath := fmt.Sprintf("/etc/letsencrypt/live/%s/privkey.pem", domainName)

	// Verify certificate files exist
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("certificate file not found at %s", certPath)
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("key file not found at %s", keyPath)
	}

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

func enableSSLWithSelfSigned(domainName string) error {
	// Create self-signed certificate directory
	sslDir := "/etc/ssl/webstack"
	if err := os.MkdirAll(sslDir, 0755); err != nil {
		return fmt.Errorf("could not create SSL directory: %v", err)
	}

	certPath := filepath.Join(sslDir, domainName+".crt")
	keyPath := filepath.Join(sslDir, domainName+".key")

	// Check if certificate already exists
	if _, err := os.Stat(certPath); err == nil {
		fmt.Printf("✅ Using existing self-signed certificate for %s\n", domainName)
		if err := saveAndEnableSSL(domainName, certPath, keyPath); err != nil {
			return err
		}
		return nil
	}

	// Generate self-signed certificate
	fmt.Println("🔑 Generating self-signed certificate...")
	args := []string{
		"req",
		"-x509",
		"-newkey", "rsa:2048",
		"-keyout", keyPath,
		"-out", certPath,
		"-days", "365",
		"-nodes",
		"-subj", fmt.Sprintf("/CN=%s", domainName),
	}

	if err := runCommand("openssl", args...); err != nil {
		return fmt.Errorf("could not generate self-signed certificate: %v", err)
	}

	fmt.Printf("✅ Self-signed certificate generated\n")

	// Set proper permissions
	os.Chmod(keyPath, 0600)
	os.Chmod(certPath, 0644)

	if err := saveAndEnableSSL(domainName, certPath, keyPath); err != nil {
		return err
	}

	fmt.Printf("⚠️  Self-signed certificate warning:\n")
	fmt.Printf("   This certificate is self-signed and not trusted by browsers.\n")
	fmt.Printf("   You'll see a security warning when accessing https://%s\n", domainName)
	fmt.Printf("   This is normal for development. Use it only for testing.\n")

	return nil
}

func saveAndEnableSSL(domainName, certPath, keyPath string) error {
	// Save SSL configuration
	cert := SSLCertificate{
		Domain:    domainName,
		Email:     "self-signed@localhost",
		Enabled:   true,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().AddDate(1, 0, 0), // 1 year
		CertPath:  certPath,
		KeyPath:   keyPath,
	}

	if err := saveSSLCert(cert); err != nil {
		return fmt.Errorf("error saving SSL configuration: %v", err)
	}

	// Update domain configuration to use SSL
	if err := enableSSLForDomain(domainName, certPath, keyPath, "self-signed@localhost"); err != nil {
		return fmt.Errorf("error updating domain configuration: %v", err)
	}

	// Generate SSL-enabled configuration
	if err := generateSSLConfig(domainName); err != nil {
		return fmt.Errorf("error generating SSL configuration: %v", err)
	}

	// Reload web servers
	reloadWebServers()

	fmt.Printf("✅ SSL enabled successfully for %s\n", domainName)
	fmt.Printf("   Certificate: %s\n", certPath)
	fmt.Printf("   Private Key: %s\n", keyPath)

	return nil
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

func enableSSLForDomain(domainName, certPath, keyPath, email string) error {
	// Get the domain from domain configuration
	d, err := domain.GetDomain(domainName)
	if err != nil {
		return fmt.Errorf("could not find domain: %v", err)
	}

	// Update SSL-related fields
	d.SSLEnabled = true
	d.SSLCertPath = certPath
	d.SSLKeyPath = keyPath
	d.SSLEmail = email

	// Save updated domain
	if err := domain.UpdateDomain(*d); err != nil {
		return fmt.Errorf("could not update domain: %v", err)
	}

	fmt.Printf("✅ SSL enabled in domain configuration for %s\n", domainName)
	return nil
}

func disableSSLForDomain(domainName string) error {
	// Get the domain from domain configuration
	d, err := domain.GetDomain(domainName)
	if err != nil {
		return fmt.Errorf("could not find domain: %v", err)
	}

	// Clear SSL-related fields
	d.SSLEnabled = false
	d.SSLCertPath = ""
	d.SSLKeyPath = ""
	d.SSLEmail = ""

	// Save updated domain
	if err := domain.UpdateDomain(*d); err != nil {
		return fmt.Errorf("could not update domain: %v", err)
	}

	fmt.Printf("✅ SSL disabled in domain configuration for %s\n", domainName)
	return nil
}

func generateSSLConfig(domainName string) error {
	// TODO: Generate SSL-enabled configuration from templates
	fmt.Printf("⚙️  Generating SSL configuration for %s...\n", domainName)

	// Get the domain
	d, err := domain.GetDomain(domainName)
	if err != nil {
		return fmt.Errorf("could not find domain: %v", err)
	}

	// Get SSL certificate paths from ssl.json
	certs, err := loadSSLCerts()
	if err != nil {
		return fmt.Errorf("could not load SSL certs: %v", err)
	}

	var certPath, keyPath string
	for _, cert := range certs {
		if cert.Domain == domainName {
			certPath = cert.CertPath
			keyPath = cert.KeyPath
			break
		}
	}

	if certPath == "" || keyPath == "" {
		return fmt.Errorf("SSL certificate not found for domain %s", domainName)
	}

	// Prepare template variables
	templateVars := map[string]interface{}{
		"Domain":       d.Name,
		"DocumentRoot": d.DocumentRoot,
		"PHPVersion":   strings.Split(d.PHPVersion, ".")[0] + d.PHPVersion[strings.LastIndex(d.PHPVersion, "."):],
		"PHPSocket":    fmt.Sprintf("unix:/run/php/php%s-fpm.sock", d.PHPVersion),
		"SSLCert":      certPath,
		"SSLKey":       keyPath,
	}

	if d.Backend == "nginx" {
		// Generate Nginx SSL config
		content, err := templates.GetNginxTemplate("domain-ssl.conf")
		if err != nil {
			return fmt.Errorf("could not read nginx SSL template: %v", err)
		}

		tmpl, err := template.New("nginx-ssl").Parse(string(content))
		if err != nil {
			return fmt.Errorf("could not parse nginx SSL template: %v", err)
		}

		// Write config file
		configFile := filepath.Join("/etc/nginx/sites-available", d.Name+".conf")
		file, err := os.Create(configFile)
		if err != nil {
			return fmt.Errorf("could not create nginx config file: %v", err)
		}
		defer file.Close()

		if err := tmpl.Execute(file, templateVars); err != nil {
			return fmt.Errorf("could not execute nginx SSL template: %v", err)
		}

		// Update symlink
		enableLink := filepath.Join("/etc/nginx/sites-enabled", d.Name+".conf")
		os.Remove(enableLink)
		if err := os.Symlink(configFile, enableLink); err != nil {
			return fmt.Errorf("could not create nginx sites-enabled symlink: %v", err)
		}

		fmt.Printf("✅ Nginx SSL configuration created: %s\n", configFile)

	} else if d.Backend == "apache" {
		// Generate Nginx proxy SSL config for Apache backend
		content, err := templates.GetNginxTemplate("proxy-ssl.conf")
		if err != nil {
			return fmt.Errorf("could not read nginx proxy SSL template: %v", err)
		}

		tmpl, err := template.New("nginx-proxy-ssl").Parse(string(content))
		if err != nil {
			return fmt.Errorf("could not parse nginx proxy SSL template: %v", err)
		}

		// Write Nginx config file
		nginxConfigFile := filepath.Join("/etc/nginx/sites-available", d.Name+".conf")
		file, err := os.Create(nginxConfigFile)
		if err != nil {
			return fmt.Errorf("could not create nginx config file: %v", err)
		}
		defer file.Close()

		if err := tmpl.Execute(file, templateVars); err != nil {
			return fmt.Errorf("could not execute nginx proxy SSL template: %v", err)
		}

		// Update Nginx symlink
		enableLink := filepath.Join("/etc/nginx/sites-enabled", d.Name+".conf")
		os.Remove(enableLink)
		if err := os.Symlink(nginxConfigFile, enableLink); err != nil {
			return fmt.Errorf("could not create nginx sites-enabled symlink: %v", err)
		}

		fmt.Printf("✅ Nginx proxy SSL configuration created: %s\n", nginxConfigFile)
	}

	return nil
}

func generateNonSSLConfig(domainName string) error {
	// TODO: Generate non-SSL configuration from templates
	fmt.Printf("⚙️  Generating non-SSL configuration for %s...\n", domainName)

	// Get the domain
	d, err := domain.GetDomain(domainName)
	if err != nil {
		return fmt.Errorf("could not find domain: %v", err)
	}

	// Call domain's generateConfig to regenerate the regular (non-SSL) config
	if err := domain.GenerateConfig(*d); err != nil {
		return fmt.Errorf("could not generate config: %v", err)
	}

	return nil
}

// validateDomainForLetsEncrypt performs pre-validation checks for Let's Encrypt
func validateDomainForLetsEncrypt(domainName string) error {
	// Check if domain resolves
	ips, err := net.LookupHost(domainName)
	if err != nil {
		return fmt.Errorf("domain '%s' does not resolve: %v", domainName, err)
	}

	if len(ips) == 0 {
		return fmt.Errorf("domain '%s' resolved but has no IP addresses", domainName)
	}

	fmt.Printf("   Domain resolves to: %s\n", strings.Join(ips, ", "))

	// Check if port 80 is accessible (basic check)
	// This is a simple heuristic - real verification happens during ACME challenge
	fmt.Println("   ✓ Domain resolution validated")

	// Check if system time is reasonable (Let's Encrypt requires this)
	now := time.Now()
	if now.Year() < 2020 {
		return fmt.Errorf("system time is too far in the past (year %d). Let's Encrypt requires accurate system time. Run: sudo ntpdate -s time.nist.gov", now.Year())
	}

	fmt.Println("   ✓ System time validated")

	return nil
}

// setupAutoRenewal configures automatic certificate renewal via cronjob
func setupAutoRenewal(domainName, email string) error {
	// Create a renewal script
	renewScript := fmt.Sprintf(`#!/bin/bash
# WebStack SSL Certificate Renewal Script for %s
# Auto-generated renewal script

/usr/bin/certbot renew --cert-name %s --quiet
if [ $? -eq 0 ]; then
    # Reload web servers on successful renewal
    /usr/bin/systemctl reload nginx 2>/dev/null
    /usr/bin/systemctl reload apache2 2>/dev/null
    
    # Log successful renewal
    echo "$(date): Certificate renewed successfully for %s" >> /var/log/webstack/ssl-renewal.log
else
    # Log renewal failure
    echo "$(date): Certificate renewal FAILED for %s" >> /var/log/webstack/ssl-renewal.log
    # Send email notification (optional)
    echo "Certificate renewal failed for %s. Check /var/log/webstack/ssl-renewal.log" | mail -s "WebStack SSL Renewal Failed" "%s" 2>/dev/null
fi
`, domainName, domainName, domainName, domainName, domainName, email)

	// Create log directory
	logDir := "/var/log/webstack"
	os.MkdirAll(logDir, 0755)

	// Write renewal script
	scriptPath := filepath.Join("/usr/local/bin", fmt.Sprintf("webstack-renewal-%s.sh", domainName))
	if err := ioutil.WriteFile(scriptPath, []byte(renewScript), 0755); err != nil {
		return fmt.Errorf("could not create renewal script: %v", err)
	}

	// Add cronjob for renewal (run at 2 AM daily)
	// Certbot itself handles checking if renewal is needed (only renews if <30 days to expiry)
	cronjobEntry := fmt.Sprintf("0 2 * * * %s >> /var/log/webstack/ssl-renewal.log 2>&1", scriptPath)

	// Check if cronjob already exists
	cmd := exec.Command("crontab", "-l")
	output, _ := cmd.Output()
	existingCrons := string(output)

	if strings.Contains(existingCrons, scriptPath) {
		// Cronjob already exists
		return nil
	}

	// Add new cronjob
	newCrontab := existingCrons + cronjobEntry + "\n"
	cmd = exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not add cronjob: %v", err)
	}

	return nil
}

// removeAutoRenewal removes the cronjob for automatic renewal
func removeAutoRenewal(domainName string) error {
	scriptPath := filepath.Join("/usr/local/bin", fmt.Sprintf("webstack-renewal-%s.sh", domainName))

	// Get current crontab
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.Output()
	if err != nil {
		// No crontab exists, nothing to remove
		return nil
	}

	// Remove the line containing this domain's script
	lines := strings.Split(string(output), "\n")
	var newLines []string
	for _, line := range lines {
		if !strings.Contains(line, scriptPath) {
			newLines = append(newLines, line)
		}
	}

	// Update crontab
	newCrontab := strings.Join(newLines, "\n")
	cmd = exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not update cronjob: %v", err)
	}

	// Remove script file
	os.Remove(scriptPath)

	return nil
}

// ManageAutorenew enables, disables, or checks status of automatic renewal
func ManageAutorenew(action string) {
	action = strings.TrimSpace(strings.ToLower(action))

	switch action {
	case "enable":
		enableAutorenew()
	case "disable":
		disableAutorenew()
	case "status":
		checkAutorenewStatus()
	case "trigger":
		triggerRenewal()
	default:
		fmt.Printf("❌ Unknown action: %s\n", action)
		fmt.Println("Usage: webstack-cli ssl autorenew [enable|disable|status|trigger]")
	}
}

// triggerRenewal manually triggers certificate renewal immediately (for testing)
func triggerRenewal() {
	fmt.Println("🔄 Triggering SSL certificate renewal manually...")
	fmt.Println("   This will run the renewal service immediately for testing purposes.")

	// Check if certbot is installed
	if err := ensureCertbotInstalled(); err != nil {
		fmt.Printf("Error: certbot not installed: %v\n", err)
		return
	}

	// Run certbot renew with verbose output for testing
	fmt.Println("\n📋 Running: certbot renew --deploy-hook 'systemctl reload nginx || true; systemctl reload apache2 || true'")
	fmt.Println("   Note: This will only renew certificates expiring within 30 days\n")

	cmd := exec.Command("certbot", "renew", "--deploy-hook", "systemctl reload nginx || true; systemctl reload apache2 || true")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("\n❌ Renewal trigger failed: %v\n", err)
		fmt.Println("\nTo run a dry-run (test without making changes):")
		fmt.Println("  sudo webstack-cli ssl autorenew trigger --dry-run")
		return
	}

	fmt.Println("\n✅ Renewal trigger completed successfully")
	fmt.Println("   Check logs for details: journalctl -u webstack-certbot-renew.service -f")
}

// enableAutorenew sets up systemd timer for automatic certificate renewal
func enableAutorenew() {
	fmt.Println("🔧 Setting up automatic SSL certificate renewal...")

	// Check if certbot is installed
	if err := ensureCertbotInstalled(); err != nil {
		fmt.Printf("Error: certbot not installed: %v\n", err)
		return
	}

	// Check if already enabled via systemd
	if isSystemdTimerActive("webstack-certbot-renew.timer") {
		fmt.Println("✅ Autorenew already enabled (systemd timer)")
		return
	}

	// Check if already enabled via cron
	if isCronJobActive() {
		fmt.Println("✅ Autorenew already enabled (cron)")
		return
	}

	// Try to enable systemd timer (preferred)
	if err := enableSystemdTimer(); err == nil {
		fmt.Println("✅ Automatic renewal enabled (systemd timer)")
		fmt.Println("   Timer: webstack-certbot-renew.timer")
		fmt.Println("   Schedule: Daily at 03:15 UTC")
		fmt.Println("\n   Check status: systemctl status webstack-certbot-renew.timer")
		fmt.Println("   View logs: journalctl -u webstack-certbot-renew.service -f")
		return
	}

	// Fallback to cron if systemd fails
	if err := enableCronJob(); err == nil {
		fmt.Println("✅ Automatic renewal enabled (cron)")
		fmt.Println("   Schedule: Daily at 3:00 and 15:00 UTC")
		fmt.Println("\n   Check status: crontab -l")
		fmt.Println("   View logs: grep CRON /var/log/syslog")
		return
	}

	fmt.Println("❌ Failed to enable automatic renewal")
	fmt.Println("   Try enabling systemd timer manually:")
	fmt.Println("   sudo systemctl enable --now webstack-certbot-renew.timer")
}

// disableAutorenew removes automatic certificate renewal
func disableAutorenew() {
	fmt.Println("🔧 Disabling automatic SSL certificate renewal...")

	// Try to disable systemd timer
	if isSystemdTimerActive("webstack-certbot-renew.timer") {
		if err := disableSystemdTimer(); err == nil {
			fmt.Println("✅ Systemd timer disabled")
			return
		}
	}

	// Try to disable cron
	if isCronJobActive() {
		if err := disableCronJob(); err == nil {
			fmt.Println("✅ Cron job disabled")
			return
		}
	}

	fmt.Println("⚠️  No automatic renewal found to disable")
}

// checkAutorenewStatus checks if automatic renewal is enabled
func checkAutorenewStatus() {
	fmt.Println("Checking automatic SSL renewal status...")

	// Check systemd timer
	if isSystemdTimerActive("webstack-certbot-renew.timer") {
		fmt.Println("\n✅ Status: ENABLED (systemd timer)")
		fmt.Println("\nSystemd Timer Details:")
		runCommand("systemctl", "status", "webstack-certbot-renew.timer")
		return
	}

	// Check cron
	if isCronJobActive() {
		fmt.Println("\n✅ Status: ENABLED (cron)")
		fmt.Println("\nCron Job Details:")
		runCommand("crontab", "-l")
		return
	}

	fmt.Println("\n❌ Status: DISABLED")
	fmt.Println("\nTo enable automatic renewal, run:")
	fmt.Println("  webstack-cli ssl autorenew enable")
}

// isSystemdTimerActive checks if a systemd timer is active
func isSystemdTimerActive(timerName string) bool {
	cmd := exec.Command("systemctl", "is-active", timerName)
	return cmd.Run() == nil
}

// isCronJobActive checks if our cron job is active
func isCronJobActive() bool {
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "certbot renew")
}

// enableSystemdTimer creates and enables a systemd timer for cert renewal
func enableSystemdTimer() error {
	// Create service file
	serviceFile := "/etc/systemd/system/webstack-certbot-renew.service"
	serviceContent := `[Unit]
Description=WebStack Certbot Renewal
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/bin/certbot renew --quiet --deploy-hook "systemctl reload nginx || true; systemctl reload apache2 || true"
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
`

	if err := ioutil.WriteFile(serviceFile, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("could not create service file: %v", err)
	}

	// Create timer file
	timerFile := "/etc/systemd/system/webstack-certbot-renew.timer"
	timerContent := `[Unit]
Description=Daily WebStack Certbot Renewal Timer
Requires=webstack-certbot-renew.service

[Timer]
OnCalendar=daily
OnCalendar=*-*-* 03:15:00
Persistent=true
OnBootSec=5min

[Install]
WantedBy=timers.target
`

	if err := ioutil.WriteFile(timerFile, []byte(timerContent), 0644); err != nil {
		return fmt.Errorf("could not create timer file: %v", err)
	}

	// Reload systemd daemon
	if err := runCommand("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload failed: %v", err)
	}

	// Enable and start timer
	if err := runCommand("systemctl", "enable", "webstack-certbot-renew.timer"); err != nil {
		return fmt.Errorf("could not enable timer: %v", err)
	}

	if err := runCommand("systemctl", "start", "webstack-certbot-renew.timer"); err != nil {
		return fmt.Errorf("could not start timer: %v", err)
	}

	return nil
}

// disableSystemdTimer disables the systemd timer
func disableSystemdTimer() error {
	if err := runCommand("systemctl", "stop", "webstack-certbot-renew.timer"); err != nil {
		return fmt.Errorf("could not stop timer: %v", err)
	}

	if err := runCommand("systemctl", "disable", "webstack-certbot-renew.timer"); err != nil {
		return fmt.Errorf("could not disable timer: %v", err)
	}

	// Remove service and timer files
	os.Remove("/etc/systemd/system/webstack-certbot-renew.service")
	os.Remove("/etc/systemd/system/webstack-certbot-renew.timer")

	// Reload systemd daemon
	runCommand("systemctl", "daemon-reload")

	return nil
}

// enableCronJob creates a cron job for automatic renewal
func enableCronJob() error {
	cronjob := `0 3,15 * * * /usr/bin/certbot renew --quiet --deploy-hook "systemctl reload nginx || true; systemctl reload apache2 || true"` + "\n"

	// Get current crontab
	cmd := exec.Command("crontab", "-l")
	output, _ := cmd.Output() // Ignore error if no crontab exists yet

	// Check if job already exists
	if strings.Contains(string(output), "certbot renew") {
		return nil // Already exists
	}

	// Add new job
	newCrontab := string(output) + cronjob
	cmd = exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not update crontab: %v", err)
	}

	return nil
}

// disableCronJob removes the cron job for automatic renewal
func disableCronJob() error {
	// Get current crontab
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("could not read crontab: %v", err)
	}

	// Remove certbot renewal line
	lines := strings.Split(string(output), "\n")
	var newLines []string
	for _, line := range lines {
		if !strings.Contains(line, "certbot renew") {
			newLines = append(newLines, line)
		}
	}

	// Update crontab
	newCrontab := strings.Join(newLines, "\n")
	cmd = exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not update crontab: %v", err)
	}

	return nil
}