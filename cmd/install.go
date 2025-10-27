package cmd

import (
	"webstack-cli/internal/installer"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install web stack components",
	Long:  `Install and configure web servers, databases, and PHP-FPM versions.`,
}

var installAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Install complete web stack with interactive prompts",
	Long:  `Install Nginx, Apache, and interactively choose database and PHP options.`,
	Run: func(cmd *cobra.Command, args []string) {
		installer.InstallAll()
	},
}

var installNginxCmd = &cobra.Command{
	Use:   "nginx",
	Short: "Install Nginx web server (port 80)",
	Run: func(cmd *cobra.Command, args []string) {
		installer.InstallNginx()
	},
}

var installApacheCmd = &cobra.Command{
	Use:   "apache",
	Short: "Install Apache web server (port 8080)",
	Run: func(cmd *cobra.Command, args []string) {
		installer.InstallApache()
	},
}

var installMysqlCmd = &cobra.Command{
	Use:   "mysql",
	Short: "Install MySQL database server",
	Run: func(cmd *cobra.Command, args []string) {
		installer.InstallMySQL()
	},
}

var installMariadbCmd = &cobra.Command{
	Use:   "mariadb",
	Short: "Install MariaDB database server",
	Run: func(cmd *cobra.Command, args []string) {
		installer.InstallMariaDB()
	},
}

var installPostgresqlCmd = &cobra.Command{
	Use:   "postgresql",
	Short: "Install PostgreSQL database server",
	Run: func(cmd *cobra.Command, args []string) {
		installer.InstallPostgreSQL()
	},
}

var installPhpCmd = &cobra.Command{
	Use:   "php [version]",
	Short: "Install PHP-FPM version (5.6-8.4)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		installer.InstallPHP(args[0])
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.AddCommand(installAllCmd)
	installCmd.AddCommand(installNginxCmd)
	installCmd.AddCommand(installApacheCmd)
	installCmd.AddCommand(installMysqlCmd)
	installCmd.AddCommand(installMariadbCmd)
	installCmd.AddCommand(installPostgresqlCmd)
	installCmd.AddCommand(installPhpCmd)
}
