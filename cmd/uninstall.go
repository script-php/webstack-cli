package cmd

import (
	"webstack-cli/internal/installer"

	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall web stack components",
	Long:  `Uninstall and remove web servers, databases, and PHP-FPM versions.`,
}

var uninstallAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Uninstall complete web stack with confirmation",
	Long:  `Uninstall Nginx, Apache, databases, and PHP versions with user confirmation.`,
	Run: func(cmd *cobra.Command, args []string) {
		installer.UninstallAll()
	},
}

var uninstallNginxCmd = &cobra.Command{
	Use:   "nginx",
	Short: "Uninstall Nginx web server",
	Run: func(cmd *cobra.Command, args []string) {
		installer.UninstallNginx()
	},
}

var uninstallApacheCmd = &cobra.Command{
	Use:   "apache",
	Short: "Uninstall Apache web server",
	Run: func(cmd *cobra.Command, args []string) {
		installer.UninstallApache()
	},
}

var uninstallMysqlCmd = &cobra.Command{
	Use:   "mysql",
	Short: "Uninstall MySQL database server",
	Run: func(cmd *cobra.Command, args []string) {
		installer.UninstallMySQL()
	},
}

var uninstallMariadbCmd = &cobra.Command{
	Use:   "mariadb",
	Short: "Uninstall MariaDB database server",
	Run: func(cmd *cobra.Command, args []string) {
		installer.UninstallMariaDB()
	},
}

var uninstallPostgresqlCmd = &cobra.Command{
	Use:   "postgresql",
	Short: "Uninstall PostgreSQL database server",
	Run: func(cmd *cobra.Command, args []string) {
		installer.UninstallPostgreSQL()
	},
}

var uninstallPhpCmd = &cobra.Command{
	Use:   "php [version]",
	Short: "Uninstall PHP-FPM version (5.6-8.4)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		installer.UninstallPHP(args[0])
	},
}

var uninstallPhppgadminCmd = &cobra.Command{
	Use:   "phppgadmin",
	Short: "Uninstall phpPgAdmin web interface",
	Run: func(cmd *cobra.Command, args []string) {
		installer.UninstallPhpPgAdmin()
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
	uninstallCmd.AddCommand(uninstallAllCmd)
	uninstallCmd.AddCommand(uninstallNginxCmd)
	uninstallCmd.AddCommand(uninstallApacheCmd)
	uninstallCmd.AddCommand(uninstallMysqlCmd)
	uninstallCmd.AddCommand(uninstallMariadbCmd)
	uninstallCmd.AddCommand(uninstallPostgresqlCmd)
	uninstallCmd.AddCommand(uninstallPhpCmd)
	uninstallCmd.AddCommand(uninstallPhppgadminCmd)
}
