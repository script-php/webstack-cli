package templates

import (
	"embed"
)

//go:embed nginx/* apache/* mysql/* php-fpm/*
var FS embed.FS

// GetTemplate reads a template file from the embedded filesystem
func GetTemplate(path string) ([]byte, error) {
	return FS.ReadFile(path)
}

// GetNginxTemplate reads an nginx template
func GetNginxTemplate(filename string) ([]byte, error) {
	return GetTemplate("nginx/" + filename)
}

// GetApacheTemplate reads an apache template
func GetApacheTemplate(filename string) ([]byte, error) {
	return GetTemplate("apache/" + filename)
}

// GetMySQLTemplate reads a mysql template
func GetMySQLTemplate(filename string) ([]byte, error) {
	return GetTemplate("mysql/" + filename)
}

// GetPHPTemplate reads a php-fpm template
func GetPHPTemplate(filename string) ([]byte, error) {
	return GetTemplate("php-fpm/" + filename)
}
