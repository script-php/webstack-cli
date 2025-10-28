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
