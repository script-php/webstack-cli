package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

const configFile = "/etc/webstack/config.json"

// ServerConfig represents configuration for a server
type ServerConfig struct {
	Installed bool   `json:"installed"`
	Port      int    `json:"port"`
	Mode      string `json:"mode"` // "standalone", "proxy", "backend"
	Username  string `json:"username,omitempty"` // For databases
	Password  string `json:"password,omitempty"` // For databases
}

// Config represents the main configuration structure
type Config struct {
	Version  string                   `json:"version"`
	Servers  map[string]ServerConfig `json:"servers"`
	Defaults map[string]interface{}  `json:"defaults"`
}

// DefaultConfig returns a new config with default values
func DefaultConfig() *Config {
	return &Config{
		Version: "1.0",
		Servers: map[string]ServerConfig{
			"nginx": {
				Installed: false,
				Port:      80,
				Mode:      "standalone",
			},
			"apache": {
				Installed: false,
				Port:      8080,
				Mode:      "backend",
			},
			"mysql": {
				Installed: false,
				Port:      3306,
				Mode:      "backend",
			},
			"mariadb": {
				Installed: false,
				Port:      3306,
				Mode:      "backend",
			},
			"postgresql": {
				Installed: false,
				Port:      5432,
				Mode:      "backend",
			},
		},
		Defaults: map[string]interface{}{
			"php_version":  "8.1",
			"ssl_provider": "letsencrypt",
		},
	}
}

// Load reads config from file
func Load() (*Config, error) {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &cfg, nil
}

// Save writes config to file
func (c *Config) Save() error {
	// Ensure directory exists
	dir := filepath.Dir(configFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}

	if err := ioutil.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

// GetServer returns server config by name
func (c *Config) GetServer(name string) (ServerConfig, bool) {
	srv, ok := c.Servers[name]
	return srv, ok
}

// SetServer updates server configuration
func (c *Config) SetServer(name string, srv ServerConfig) {
	if c.Servers == nil {
		c.Servers = make(map[string]ServerConfig)
	}
	c.Servers[name] = srv
}

// IsInstalled checks if a server is installed
func (c *Config) IsInstalled(name string) bool {
	if srv, ok := c.Servers[name]; ok {
		return srv.Installed
	}
	return false
}

// GetPort returns the port for a server
func (c *Config) GetPort(name string) int {
	if srv, ok := c.Servers[name]; ok {
		return srv.Port
	}
	return 0
}

// GetMode returns the mode for a server
func (c *Config) GetMode(name string) string {
	if srv, ok := c.Servers[name]; ok {
		return srv.Mode
	}
	return ""
}

// SetDefault sets a default value
func (c *Config) SetDefault(key string, value interface{}) {
	if c.Defaults == nil {
		c.Defaults = make(map[string]interface{})
	}
	c.Defaults[key] = value
}

// GetDefault gets a default value
func (c *Config) GetDefault(key string, defaultValue interface{}) interface{} {
	if val, ok := c.Defaults[key]; ok {
		return val
	}
	return defaultValue
}
