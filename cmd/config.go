package cmd

import (
	"fmt"
	"os/exec"
	"webstack-cli/internal/config"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage WebStack configuration",
	Long:  `Manage WebStack configuration settings like default PHP version and SSL provider.`,
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Long: `Set a configuration value. Examples:
  webstack config set php_version 8.3
  webstack config set ssl_provider letsencrypt`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]

		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}

		switch key {
		case "php_version":
			validVersions := []string{"5.6", "7.0", "7.1", "7.2", "7.3", "7.4", "8.0", "8.1", "8.2", "8.3", "8.4"}
			valid := false
			for _, v := range validVersions {
				if v == value {
					valid = true
					break
				}
			}
			if !valid {
				fmt.Printf("Invalid PHP version: %s\n", value)
				fmt.Printf("Valid versions: %v\n", validVersions)
				return
			}

			// Check if PHP version is installed
			phpFpmService := fmt.Sprintf("php%s-fpm", value)
			checkCmd := exec.Command("systemctl", "is-enabled", phpFpmService)
			err := checkCmd.Run()
			if err != nil {
				fmt.Printf("PHP %s is not installed\n", value)
				fmt.Println("Use 'webstack install php [version]' to install it first")
				return
			}

			cfg.SetDefault("php_version", value)
			fmt.Printf("Default PHP version set to %s\n", value)

		case "ssl_provider":
			if value != "letsencrypt" && value != "custom" {
				fmt.Printf("Invalid SSL provider: %s\n", value)
				fmt.Println("Valid providers: letsencrypt, custom")
				return
			}
			cfg.SetDefault("ssl_provider", value)
			fmt.Printf("Default SSL provider set to %s\n", value)

		default:
			fmt.Printf("Unknown configuration key: %s\n", key)
			return
		}

		if err := cfg.Save(); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			return
		}
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Long: `Get a configuration value. Examples:
  webstack config get php_version
  webstack config get ssl_provider`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]

		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}

		value := cfg.GetDefault(key, nil)
		if value == nil {
			fmt.Printf("Configuration key '%s' not found\n", key)
			return
		}

		fmt.Printf("%s = %v\n", key, value)
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show all configuration values",
	Long:  `Display all current configuration values.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}

		fmt.Println("WebStack Configuration")
		fmt.Println("======================")
		fmt.Printf("Version: %s\n", cfg.Version)
		fmt.Println("\nDefaults:")
		for key, value := range cfg.Defaults {
			fmt.Printf("  %s = %v\n", key, value)
		}
		fmt.Println("\nServers:")
		for name, srv := range cfg.Servers {
			status := "Not installed"
			if srv.Installed {
				status = "Installed"
			}
			fmt.Printf("  %s: %s (Port: %d, Mode: %s)\n", name, status, srv.Port, srv.Mode)
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configShowCmd)
}
