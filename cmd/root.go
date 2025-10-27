package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "webstack",
	Short: "A CLI tool for managing web stack (Nginx, Apache, PHP-FPM, MySQL/MariaDB, PostgreSQL)",
	Long: `WebStack CLI is a comprehensive tool for installing and managing a complete web development stack.
	
Features:
- Install Nginx (port 80) and Apache (port 8080)
- Install MariaDB/MySQL with phpMyAdmin
- Install PostgreSQL with phpPgAdmin
- Install PHP-FPM versions 5.6 to 8.4
- Domain management with SSL support
- Let's Encrypt SSL certificate management`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("version", "v", false, "Show version information")
}
