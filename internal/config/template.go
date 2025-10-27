package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"text/template"
)

// TemplateData contains data for template processing
type TemplateData struct {
	Domain       string
	DocumentRoot string
	PHPVersion   string
	PHPSocket    string
	SSLCert      string
	SSLKey       string
	Port         string
}

// TemplateProcessor handles template generation
type TemplateProcessor struct {
	templatesDir string
}

// NewTemplateProcessor creates a new template processor
func NewTemplateProcessor(templatesDir string) *TemplateProcessor {
	return &TemplateProcessor{
		templatesDir: templatesDir,
	}
}

// ProcessNginxTemplate generates Nginx configuration from template
func (tp *TemplateProcessor) ProcessNginxTemplate(templateName string, data TemplateData) (string, error) {
	templatePath := filepath.Join(tp.templatesDir, "nginx", templateName)
	return tp.processTemplate(templatePath, data)
}

// ProcessApacheTemplate generates Apache configuration from template
func (tp *TemplateProcessor) ProcessApacheTemplate(templateName string, data TemplateData) (string, error) {
	templatePath := filepath.Join(tp.templatesDir, "apache", templateName)
	return tp.processTemplate(templatePath, data)
}

// ProcessPHPTemplate generates PHP-FPM configuration from template
func (tp *TemplateProcessor) ProcessPHPTemplate(templateName string, data TemplateData) (string, error) {
	templatePath := filepath.Join(tp.templatesDir, "php-fpm", templateName)
	return tp.processTemplate(templatePath, data)
}

func (tp *TemplateProcessor) processTemplate(templatePath string, data TemplateData) (string, error) {
	// Read template file
	templateContent, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %v", templatePath, err)
	}

	// Parse template
	tmpl, err := template.New("config").Parse(string(templateContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %v", templatePath, err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %v", templatePath, err)
	}

	return buf.String(), nil
}

// GetPHPSocket returns the PHP-FPM socket path for a given version
func GetPHPSocket(version string) string {
	return fmt.Sprintf("unix:/run/php/php%s-fpm.sock", version)
}

// GetPHPServiceName returns the service name for a PHP version
func GetPHPServiceName(version string) string {
	return fmt.Sprintf("php%s-fpm", version)
}

// CreateTemplateData creates TemplateData from domain configuration
func CreateTemplateData(domain, documentRoot, phpVersion string, sslEnabled bool, sslCert, sslKey string) TemplateData {
	data := TemplateData{
		Domain:       domain,
		DocumentRoot: documentRoot,
		PHPVersion:   phpVersion,
		PHPSocket:    GetPHPSocket(phpVersion),
	}

	if sslEnabled {
		data.SSLCert = sslCert
		data.SSLKey = sslKey
	}

	return data
}
