package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var systemCmd = &cobra.Command{
	Use:   "system",
	Short: "System management commands",
	Long:  `System-level management commands for WebStack CLI service integration.`,
}

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload all web server configurations",
	Run:   reloadConfigurations,
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate all configurations",
	Run:   validateConfigurations,
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up temporary files and logs",
	Run:   cleanupSystem,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show system status",
	Run:   showSystemStatus,
}

func reloadConfigurations(cmd *cobra.Command, args []string) {
	quiet, _ := cmd.Flags().GetBool("quiet")

	if !quiet {
		fmt.Println("🔄 Reloading WebStack configurations...")
	}

	// Reload Nginx
	if isServiceActive("nginx") {
		if err := runSystemCommand("systemctl", "reload", "nginx"); err != nil {
			if !quiet {
				fmt.Printf("❌ Failed to reload Nginx: %v\n", err)
			}
		} else if !quiet {
			fmt.Println("✅ Nginx configuration reloaded")
		}
	}

	// Reload Apache
	if isServiceActive("apache2") {
		if err := runSystemCommand("systemctl", "reload", "apache2"); err != nil {
			if !quiet {
				fmt.Printf("❌ Failed to reload Apache: %v\n", err)
			}
		} else if !quiet {
			fmt.Println("✅ Apache configuration reloaded")
		}
	}

	// Reload PHP-FPM services
	phpServices := []string{"php5.6-fpm", "php7.0-fpm", "php7.1-fpm", "php7.2-fpm", "php7.3-fpm", "php7.4-fpm", "php8.0-fpm", "php8.1-fpm", "php8.2-fpm", "php8.3-fpm", "php8.4-fpm"}

	for _, service := range phpServices {
		if isServiceActive(service) {
			if err := runSystemCommand("systemctl", "reload", service); err != nil {
				if !quiet {
					fmt.Printf("❌ Failed to reload %s: %v\n", service, err)
				}
			} else if !quiet {
				fmt.Printf("✅ %s configuration reloaded\n", service)
			}
		}
	}

	if !quiet {
		fmt.Println("🎉 Configuration reload completed")
	}
}

func validateConfigurations(cmd *cobra.Command, args []string) {
	quiet, _ := cmd.Flags().GetBool("quiet")

	if !quiet {
		fmt.Println("🔍 Validating WebStack configurations...")
	}

	errors := 0

	// Validate Nginx configuration
	if isServiceInstalled("nginx") {
		if err := runSystemCommand("nginx", "-t"); err != nil {
			if !quiet {
				fmt.Printf("❌ Nginx configuration validation failed: %v\n", err)
			}
			errors++
		} else if !quiet {
			fmt.Println("✅ Nginx configuration is valid")
		}
	}

	// Validate Apache configuration
	if isServiceInstalled("apache2") {
		if err := runSystemCommand("apache2ctl", "configtest"); err != nil {
			if !quiet {
				fmt.Printf("❌ Apache configuration validation failed: %v\n", err)
			}
			errors++
		} else if !quiet {
			fmt.Println("✅ Apache configuration is valid")
		}
	}

	// Check domain configurations
	// TODO: Implement domain configuration validation

	// Check SSL certificates
	// TODO: Implement SSL certificate validation

	if !quiet {
		if errors == 0 {
			fmt.Println("🎉 All configurations are valid")
		} else {
			fmt.Printf("⚠️  Found %d configuration errors\n", errors)
		}
	}

	if errors > 0 {
		os.Exit(1)
	}
}

func cleanupSystem(cmd *cobra.Command, args []string) {
	quiet, _ := cmd.Flags().GetBool("quiet")

	if !quiet {
		fmt.Println("🧹 Cleaning up WebStack temporary files...")
	}

	// Clean temporary files
	if !quiet {
		fmt.Println("  • Cleaning temporary files...")
	}

	// Clean WebStack temporary files
	runSystemCommand("find", "/tmp", "-name", "webstack-*", "-type", "f", "-mtime", "+7", "-delete")
	runSystemCommand("find", "/var/tmp", "-name", "webstack-*", "-type", "f", "-mtime", "+7", "-delete")

	// Clean Nginx cache if it exists
	runSystemCommand("find", "/var/cache/nginx", "-type", "f", "-mtime", "+7", "-delete")

	// Rotate large logs
	if !quiet {
		fmt.Println("  • Rotating large log files...")
	}
	runSystemCommand("find", "/var/log/webstack", "-name", "*.log", "-size", "+100M", "-exec", "truncate", "-s", "0", "{}", "\\;")

	// Clean old SSL certificates (expired + 30 days)
	// TODO: Implement SSL cleanup

	if !quiet {
		fmt.Println("✅ Cleanup completed")
	}
}

func showSystemStatus(cmd *cobra.Command, args []string) {
	fmt.Println("WebStack System Status")
	fmt.Println("=====================")
	fmt.Println()

	// Check services
	services := []string{"nginx", "apache2", "mysql", "mariadb", "postgresql"}

	fmt.Println("🔧 Services:")
	for _, service := range services {
		if isServiceInstalled(service) {
			if isServiceActive(service) {
				fmt.Printf("  ✅ %s: Running\n", service)
			} else {
				fmt.Printf("  ❌ %s: Stopped\n", service)
			}
		}
	}

	// Check PHP-FPM versions
	fmt.Println("\n🐘 PHP-FPM Services:")
	phpServices := []string{"php5.6-fpm", "php7.0-fpm", "php7.1-fpm", "php7.2-fpm", "php7.3-fpm", "php7.4-fpm", "php8.0-fpm", "php8.1-fpm", "php8.2-fpm", "php8.3-fpm", "php8.4-fpm"}

	phpCount := 0
	for _, service := range phpServices {
		if isServiceActive(service) {
			version := service[3:6] // Extract version like "8.2" from "php8.2-fpm"
			fmt.Printf("  ✅ PHP %s: Running\n", version)
			phpCount++
		}
	}

	if phpCount == 0 {
		fmt.Println("  ⚠️  No PHP-FPM services running")
	}

	// Check disk space
	fmt.Println("\n💾 Disk Usage:")
	runSystemCommand("df", "-h", "/var/www", "/var/log", "/etc")

	// Check domains
	// TODO: Show domain count and status

	// Check SSL certificates
	// TODO: Show SSL certificate status
}

// Helper functions
func isServiceInstalled(service string) bool {
	err := runSystemCommand("systemctl", "list-unit-files", service)
	return err == nil
}

func isServiceActive(service string) bool {
	err := runSystemCommand("systemctl", "is-active", "--quiet", service)
	return err == nil
}

func runSystemCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func init() {
	rootCmd.AddCommand(systemCmd)
	systemCmd.AddCommand(reloadCmd)
	systemCmd.AddCommand(validateCmd)
	systemCmd.AddCommand(cleanupCmd)
	systemCmd.AddCommand(statusCmd)

	// Add quiet flag to system commands
	reloadCmd.Flags().Bool("quiet", false, "Suppress output")
	validateCmd.Flags().Bool("quiet", false, "Suppress output")
	cleanupCmd.Flags().Bool("quiet", false, "Suppress output")
}
