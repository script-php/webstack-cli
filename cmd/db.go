package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"webstack-cli/internal/config"

	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database management commands",
	Long:  `Manage databases: users, backups, stats, and configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use 'webstack db --help' for available commands")
	},
}

var dbUserCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage database users",
	Long:  `Create, delete, list, and manage database users.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use 'webstack db user --help' for available commands")
	},
}

var dbUserCreateCmd = &cobra.Command{
	Use:   "create [database] [username] [password] [host]",
	Short: "Create a new database user",
	Long: `Create a new database user with specified privileges and settings.
Usage:
  webstack db user create mysql appuser apppass localhost
  webstack db user create mysql appuser apppass 192.168.1.% --privileges SELECT,INSERT --max-connections 10
  webstack db user create mysql appuser apppass 192.168.1.% --database mydb --require-ssl
  webstack db user create postgresql appuser apppass localhost`,
	Args: cobra.ExactArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		dbType := strings.ToLower(args[0])
		username := args[1]
		password := args[2]
		host := args[3]

		privileges, _ := cmd.Flags().GetString("privileges")
		database, _ := cmd.Flags().GetString("database")
		maxConnections, _ := cmd.Flags().GetInt("max-connections")
		requireSSL, _ := cmd.Flags().GetBool("require-ssl")

		switch dbType {
		case "mysql", "mariadb":
			createMySQLUserWithOptions(username, password, host, privileges, database, maxConnections, requireSSL)
		case "postgresql":
			createPostgresqlUser(username, password, host)
		default:
			fmt.Printf("Unknown database type: %s\n", dbType)
			fmt.Println("Supported: mysql, mariadb, postgresql")
		}
	},
}

func init_dbUserCreateCmd() {
	dbUserCreateCmd.Flags().StringP("privileges", "p", "ALL", "Comma-separated list of privileges (SELECT,INSERT,UPDATE,DELETE,CREATE,DROP,ALTER,EXECUTE). Default: ALL")
	dbUserCreateCmd.Flags().StringP("database", "d", "*", "Database name or '*' for all databases. Default: * (all databases)")
	dbUserCreateCmd.Flags().IntP("max-connections", "m", 0, "Max connections per hour (0 = unlimited)")
	dbUserCreateCmd.Flags().BoolP("require-ssl", "s", false, "Require SSL/TLS for connections")
}

var dbUserDeleteCmd = &cobra.Command{
	Use:   "delete [database] [username] [host]",
	Short: "Delete a database user",
	Long: `Delete a database user from specified host.
Usage:
  webstack db user delete mysql appuser localhost
  webstack db user delete postgresql appuser localhost`,
	Args: cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		dbType := strings.ToLower(args[0])
		username := args[1]
		host := args[2]

		switch dbType {
		case "mysql", "mariadb":
			deleteMySQLUser(username, host)
		case "postgresql":
			deletePostgresqlUser(username)
		default:
			fmt.Printf("Unknown database type: %s\n", dbType)
			fmt.Println("Supported: mysql, mariadb, postgresql")
		}
	},
}

var dbUserListCmd = &cobra.Command{
	Use:   "list [database]",
	Short: "List all database users",
	Long: `List all users in a database.
Usage:
  webstack db user list mysql
  webstack db user list postgresql
  webstack db user list (shows all databases)`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			os.Exit(1)
		}

		if len(args) == 0 {
			// Show all databases
			fmt.Println("Listing users from all databases...")
			fmt.Println()
			listMySQLUsers()
			fmt.Println()
			listPostgresqlUsers()
			return
		}

		dbType := strings.ToLower(args[0])

		switch dbType {
		case "mysql", "mariadb":
			listMySQLUsers()
		case "postgresql":
			listPostgresqlUsers()
		default:
			fmt.Printf("Unknown database type: %s\n", dbType)
			fmt.Println("Supported: mysql, mariadb, postgresql")
		}
	},
}

var dbUserPasswordCmd = &cobra.Command{
	Use:   "password [database] [username] [newpassword]",
	Short: "Change database user password",
	Long: `Change password for a database user.
Usage:
  webstack db user password mysql appuser newpass123
  webstack db user password postgresql appuser newpass123`,
	Args: cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		dbType := strings.ToLower(args[0])
		username := args[1]
		password := args[2]

		switch dbType {
		case "mysql", "mariadb":
			changeMySQLPassword(username, password)
		case "postgresql":
			changePostgresqlPassword(username, password)
		default:
			fmt.Printf("Unknown database type: %s\n", dbType)
			fmt.Println("Supported: mysql, mariadb, postgresql")
		}
	},
}

var dbUserUpdateCmd = &cobra.Command{
	Use:   "update [database] [username]",
	Short: "Update database user settings",
	Long: `Update settings for an existing database user (privileges, SSL requirement, connection limits).
Usage:
  webstack db user update mysql appuser --privileges SELECT,INSERT --max-connections 10
  webstack db user update mysql appuser --require-ssl
  webstack db user update mysql appuser --privileges ALL --require-ssl --max-connections 5`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		dbType := strings.ToLower(args[0])
		username := args[1]

		privileges, _ := cmd.Flags().GetString("privileges")
		maxConnections, _ := cmd.Flags().GetInt("max-connections")
		requireSSL, _ := cmd.Flags().GetBool("require-ssl")
		noSSL, _ := cmd.Flags().GetBool("no-ssl")

		switch dbType {
		case "mysql", "mariadb":
			updateMySQLUser(username, privileges, maxConnections, requireSSL, noSSL)
		case "postgresql":
			fmt.Println("PostgreSQL user updates coming soon")
		default:
			fmt.Printf("Unknown database type: %s\n", dbType)
			fmt.Println("Supported: mysql, mariadb, postgresql")
		}
	},
}

var dbUserInfoCmd = &cobra.Command{
	Use:   "info [database] [username]",
	Short: "Show user account information and settings",
	Long: `Display detailed information about a database user including privileges, hosts, and settings.
Usage:
  webstack db user info mysql appuser
  webstack db user info postgresql appuser`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		dbType := strings.ToLower(args[0])
		username := args[1]

		switch dbType {
		case "mysql", "mariadb":
			showMySQLUserInfo(username)
		case "postgresql":
			showPostgresqlUserInfo(username)
		default:
			fmt.Printf("Unknown database type: %s\n", dbType)
			fmt.Println("Supported: mysql, mariadb, postgresql")
		}
	},
}

// Database management commands
var dbDatabaseCmd = &cobra.Command{
	Use:   "database",
	Short: "Manage databases",
	Long:  `Create, delete, list, and manage databases.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use 'webstack db database --help' for available commands")
	},
}

var dbDatabaseCreateCmd = &cobra.Command{
	Use:   "create [database-type] [database-name]",
	Short: "Create a new database",
	Long: `Create a new database in MySQL/MariaDB or PostgreSQL.
Usage:
  webstack db database create mysql myapp
  webstack db database create mysql myapp --charset utf8mb4 --collation utf8mb4_unicode_ci
  webstack db database create postgresql myapp --owner postgres`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		dbType := strings.ToLower(args[0])
		dbName := args[1]
		charset, _ := cmd.Flags().GetString("charset")
		collation, _ := cmd.Flags().GetString("collation")
		owner, _ := cmd.Flags().GetString("owner")

		switch dbType {
		case "mysql", "mariadb":
			createMySQLDatabase(dbName, charset, collation)
		case "postgresql":
			createPostgresqlDatabase(dbName, owner)
		default:
			fmt.Printf("Unknown database type: %s\n", dbType)
			fmt.Println("Supported: mysql, mariadb, postgresql")
		}
	},
}

var dbDatabaseDeleteCmd = &cobra.Command{
	Use:   "delete [database-type] [database-name]",
	Short: "Delete a database",
	Long: `Delete a database from MySQL/MariaDB or PostgreSQL (requires confirmation).
Usage:
  webstack db database delete mysql myapp
  webstack db database delete postgresql myapp --force`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		dbType := strings.ToLower(args[0])
		dbName := args[1]
		force, _ := cmd.Flags().GetBool("force")

		switch dbType {
		case "mysql", "mariadb":
			deleteMySQLDatabase(dbName, force)
		case "postgresql":
			deletePostgresqlDatabase(dbName, force)
		default:
			fmt.Printf("Unknown database type: %s\n", dbType)
			fmt.Println("Supported: mysql, mariadb, postgresql")
		}
	},
}

var dbDatabaseListCmd = &cobra.Command{
	Use:   "list [database-type]",
	Short: "List all databases",
	Long: `List all databases with size and other information.
Usage:
  webstack db database list mysql
  webstack db database list postgresql`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		dbType := strings.ToLower(args[0])

		switch dbType {
		case "mysql", "mariadb":
			listMySQLDatabases()
		case "postgresql":
			listPostgresqlDatabases()
		default:
			fmt.Printf("Unknown database type: %s\n", dbType)
			fmt.Println("Supported: mysql, mariadb, postgresql")
		}
	},
}

var dbDatabaseInfoCmd = &cobra.Command{
	Use:   "info [database-type] [database-name]",
	Short: "Show database information",
	Long: `Display detailed information about a database including size, tables, and charset.
Usage:
  webstack db database info mysql myapp
  webstack db database info postgresql myapp`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		dbType := strings.ToLower(args[0])
		dbName := args[1]

		switch dbType {
		case "mysql", "mariadb":
			showMySQLDatabaseInfo(dbName)
		case "postgresql":
			showPostgresqlDatabaseInfo(dbName)
		default:
			fmt.Printf("Unknown database type: %s\n", dbType)
			fmt.Println("Supported: mysql, mariadb, postgresql")
		}
	},
}

func init_dbDatabaseCreateCmd() {
	dbDatabaseCreateCmd.Flags().StringP("charset", "c", "utf8mb4", "Character set for MySQL/MariaDB (default: utf8mb4)")
	dbDatabaseCreateCmd.Flags().StringP("collation", "l", "utf8mb4_unicode_ci", "Collation for MySQL/MariaDB (default: utf8mb4_unicode_ci)")
	dbDatabaseCreateCmd.Flags().StringP("owner", "o", "postgres", "Owner for PostgreSQL (default: postgres)")
}

func init_dbDatabaseDeleteCmd() {
	dbDatabaseDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}

func init_dbUserUpdateCmd() {
	dbUserUpdateCmd.Flags().StringP("privileges", "p", "", "Comma-separated list of privileges (SELECT,INSERT,UPDATE,DELETE,CREATE,DROP,ALTER,EXECUTE)")
	dbUserUpdateCmd.Flags().IntP("max-connections", "m", -1, "Max connections per hour (-1 = unlimited, unchanged)")
	dbUserUpdateCmd.Flags().BoolP("require-ssl", "s", false, "Require SSL/TLS for connections")
	dbUserUpdateCmd.Flags().BoolP("no-ssl", "n", false, "Remove SSL/TLS requirement")
}

// MySQL/MariaDB user management functions
func createMySQLUser(username, password, host string) {
	createMySQLUserWithOptions(username, password, host, "ALL", "*", 0, false)
}

func createMySQLUserWithOptions(username, password, host, privileges, database string, maxConnections int, requireSSL bool) {
	fmt.Printf("👤 Creating MySQL user '%s'@'%s'...\n", username, host)

	// Load config to get admin password from defaults
	cfg, err := config.Load()
	var adminPass string

	if err == nil {
		// Try to get password from defaults
		if pass, ok := cfg.GetDefault("mysql_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		} else if pass, ok := cfg.GetDefault("mariadb_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		}
	}

	// Fallback to prompt if config not available
	if adminPass == "" {
		fmt.Print("Enter MySQL/MariaDB admin password: ")
		fmt.Scanln(&adminPass)
	}

	if host == "" {
		host = "localhost"
	}

	// Create user
	createCmd := fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%s' IDENTIFIED BY '%s';", username, host, password)

	mysqlCmd := exec.Command("mysql", "-u", "root", "-p"+adminPass, "-e", createCmd)
	if err := mysqlCmd.Run(); err != nil {
		fmt.Printf("Error creating user: %v\n", err)
		fmt.Println("   Try manually: mysql -u root -p")
		return
	}

	// Build privilege string
	dbSpec := database
	if database == "*" {
		dbSpec = "*.*"
	} else {
		dbSpec = database + ".*"
	}

	privStr := privileges
	if privileges == "ALL" {
		privStr = "ALL PRIVILEGES"
	}

	// Grant privileges
	grantCmd := fmt.Sprintf("GRANT %s ON %s TO '%s'@'%s' WITH GRANT OPTION;", privStr, dbSpec, username, host)

	mysqlCmd = exec.Command("mysql", "-u", "root", "-p"+adminPass, "-e", grantCmd)
	if err := mysqlCmd.Run(); err != nil {
		fmt.Printf("Error granting privileges: %v\n", err)
		return
	}

	// Set resource limits if specified
	if maxConnections > 0 || requireSSL {
		alterCmd := fmt.Sprintf("ALTER USER '%s'@'%s'", username, host)

		if requireSSL {
			alterCmd += " REQUIRE SSL"
		}

		if maxConnections > 0 {
			if requireSSL {
				alterCmd += " "
			}
			alterCmd += fmt.Sprintf("WITH MAX_CONNECTIONS_PER_HOUR %d", maxConnections)
		}

		alterCmd += ";"

		mysqlCmd = exec.Command("mysql", "-u", "root", "-p"+adminPass, "-e", alterCmd)
		if err := mysqlCmd.Run(); err != nil {
			fmt.Printf("Warning: Could not set user limits: %v\n", err)
		}
	}

	// Flush privileges
	flushCmd := "FLUSH PRIVILEGES;"
	mysqlCmd = exec.Command("mysql", "-u", "root", "-p"+adminPass, "-e", flushCmd)
	mysqlCmd.Run()

	fmt.Printf("User '%s'@'%s' created successfully\n", username, host)
	if privileges != "ALL" {
		fmt.Printf("   Privileges: %s on %s\n", privileges, dbSpec)
	}
	if requireSSL {
		fmt.Printf("   SSL/TLS required for connections\n")
	}
	if maxConnections > 0 {
		fmt.Printf("   Max connections/hour: %d\n", maxConnections)
	}
	fmt.Printf("   Connect with: mysql -u %s -h <server> -p\n", username)
}

func deleteMySQLUser(username, host string) {
	fmt.Printf("🗑️  Deleting MySQL user '%s'@'%s'...\n", username, host)

	// Load config to get admin password from defaults
	cfg, err := config.Load()
	var adminPass string

	if err == nil {
		// Try to get password from defaults
		if pass, ok := cfg.GetDefault("mysql_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		} else if pass, ok := cfg.GetDefault("mariadb_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		}
	}

	// Fallback to prompt if config not available
	if adminPass == "" {
		fmt.Print("Enter MySQL/MariaDB admin password: ")
		fmt.Scanln(&adminPass)
	}

	deleteCmd := fmt.Sprintf("DROP USER IF EXISTS '%s'@'%s'; FLUSH PRIVILEGES;", username, host)

	mysqlCmd := exec.Command("mysql", "-u", "root", "-p"+adminPass, "-e", deleteCmd)
	if err := mysqlCmd.Run(); err != nil {
		fmt.Printf("Error deleting user: %v\n", err)
		return
	}

	fmt.Printf("User '%s'@'%s' deleted successfully\n", username, host)
}

func listMySQLUsers() {
	fmt.Println("MySQL Users:")
	fmt.Println("─────────────────────────────────────────")

	// Load config to get password from defaults
	cfg, err := config.Load()
	var adminPass string

	if err == nil {
		// Try to get password from defaults
		if pass, ok := cfg.GetDefault("mysql_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		} else if pass, ok := cfg.GetDefault("mariadb_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		}
	}

	// Fallback to prompt if config not available
	if adminPass == "" {
		fmt.Print("Enter MySQL/MariaDB root password: ")
		fmt.Scanln(&adminPass)
	}

	executeMySQLQuery("SELECT User, Host FROM mysql.user ORDER BY User, Host;", "root", adminPass)
}

func executeMySQLQuery(query, user, password string) {
	mysqlCmd := exec.Command("mysql", "-u", user, "-p"+password, "-e", query)
	output, err := mysqlCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Print(string(output))
}

func changeMySQLPassword(username, password string) {
	fmt.Printf("Changing password for user '%s'...\n", username)

	// Load config to get admin password from defaults
	cfg, err := config.Load()
	var adminPass string

	if err == nil {
		// Try to get password from defaults
		if pass, ok := cfg.GetDefault("mysql_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		} else if pass, ok := cfg.GetDefault("mariadb_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		}
	}

	// Fallback to prompt if config not available
	if adminPass == "" {
		fmt.Print("Enter MySQL/MariaDB admin password: ")
		fmt.Scanln(&adminPass)
	}

	// Get current host for the user
	getHostCmd := fmt.Sprintf("SELECT Host FROM mysql.user WHERE User='%s' LIMIT 1;", username)
	output, err := exec.Command("mysql", "-u", "root", "-p"+adminPass, "-sNe", getHostCmd).Output()
	if err != nil {
		fmt.Printf("User not found: %s\n", username)
		return
	}

	host := strings.TrimSpace(string(output))
	if host == "" {
		fmt.Printf("User '%s' not found\n", username)
		return
	}

	// Update password
	updateCmd := fmt.Sprintf("ALTER USER '%s'@'%s' IDENTIFIED BY '%s'; FLUSH PRIVILEGES;", username, host, password)

	mysqlCmd := exec.Command("mysql", "-u", "root", "-p"+adminPass, "-e", updateCmd)
	if err := mysqlCmd.Run(); err != nil {
		fmt.Printf("Error changing password: %v\n", err)
		return
	}

	fmt.Printf("Password changed for '%s'@'%s'\n", username, host)
}

// PostgreSQL user management functions
func createPostgresqlUser(username, password, host string) {
	fmt.Printf("Creating PostgreSQL user '%s'...\n", username)

	createCmd := fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s' CREATEDB;", username, password)

	psqlCmd := exec.Command("sudo", "-u", "postgres", "psql", "-c", createCmd)
	if err := psqlCmd.Run(); err != nil {
		fmt.Printf("Error creating user: %v\n", err)
		return
	}

	// Grant privileges
	grantCmd := fmt.Sprintf("GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO %s;", username)
	psqlCmd = exec.Command("sudo", "-u", "postgres", "psql", "-c", grantCmd)
	psqlCmd.Run() // Ignore error if schema doesn't exist yet

	fmt.Printf("PostgreSQL user '%s' created successfully\n", username)
	fmt.Printf("   Connect with: psql -U %s -h <server> -d postgres\n", username)
}

func deletePostgresqlUser(username string) {
	fmt.Printf("Deleting PostgreSQL user '%s'...\n", username)

	// Drop owned objects first
	dropCmd := fmt.Sprintf("DROP OWNED BY %s CASCADE; DROP USER IF EXISTS %s;", username, username)

	psqlCmd := exec.Command("sudo", "-u", "postgres", "psql", "-c", dropCmd)
	if err := psqlCmd.Run(); err != nil {
		fmt.Printf("Error deleting user: %v\n", err)
		return
	}

	fmt.Printf("PostgreSQL user '%s' deleted successfully\n", username)
}

func listPostgresqlUsers() {
	fmt.Println("PostgreSQL Users:")
	fmt.Println("─────────────────────────────────────────")

	listCmd := `\du`

	psqlCmd := exec.Command("sudo", "-u", "postgres", "psql", "-c", listCmd)
	if err := psqlCmd.Run(); err != nil {
		fmt.Printf("Error listing users: %v\n", err)
		return
	}
}

func changePostgresqlPassword(username, password string) {
	fmt.Printf("Changing password for user '%s'...\n", username)

	updateCmd := fmt.Sprintf("ALTER USER %s WITH PASSWORD '%s';", username, password)

	psqlCmd := exec.Command("sudo", "-u", "postgres", "psql", "-c", updateCmd)
	if err := psqlCmd.Run(); err != nil {
		fmt.Printf("Error changing password: %v\n", err)
		return
	}

	fmt.Printf("Password changed for user '%s'\n", username)
}

func updateMySQLUser(username string, privileges string, maxConnections int, requireSSL, noSSL bool) {
	fmt.Printf("Updating MySQL user '%s'...\n", username)

	// Load config to get admin password from defaults
	cfg, err := config.Load()
	var adminPass string

	if err == nil {
		if pass, ok := cfg.GetDefault("mysql_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		} else if pass, ok := cfg.GetDefault("mariadb_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		}
	}

	if adminPass == "" {
		fmt.Print("Enter MySQL/MariaDB admin password: ")
		fmt.Scanln(&adminPass)
	}

	if requireSSL && noSSL {
		fmt.Println("Cannot use both --require-ssl and --no-ssl")
		return
	}

	// Get user hosts
	hostCmd := fmt.Sprintf("SELECT DISTINCT Host FROM mysql.user WHERE User='%s';", username)
	output, err := exec.Command("mysql", "-u", "root", "-p"+adminPass, "-sNe", hostCmd).Output()
	if err != nil {
		fmt.Printf("User not found: %s\n", username)
		return
	}

	hosts := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(hosts) == 0 || hosts[0] == "" {
		fmt.Printf("User '%s' not found\n", username)
		return
	}

	updated := false

	// Update privileges if specified
	if privileges != "" {
		for _, host := range hosts {
			host = strings.TrimSpace(host)
			if host == "" {
				continue
			}

			privStr := privileges
			if privileges == "ALL" {
				privStr = "ALL PRIVILEGES"
			}

			revokeCmd := fmt.Sprintf("REVOKE ALL PRIVILEGES ON *.* FROM '%s'@'%s';", username, host)
			mysqlCmd := exec.Command("mysql", "-u", "root", "-p"+adminPass, "-e", revokeCmd)
			mysqlCmd.Run() // Ignore errors

			grantCmd := fmt.Sprintf("GRANT %s ON *.* TO '%s'@'%s' WITH GRANT OPTION;", privStr, username, host)
			mysqlCmd = exec.Command("mysql", "-u", "root", "-p"+adminPass, "-e", grantCmd)
			if err := mysqlCmd.Run(); err != nil {
				fmt.Printf("Could not update privileges for %s@%s: %v\n", username, host, err)
				continue
			}

			fmt.Printf("Privileges updated for '%s'@'%s': %s\n", username, host, privileges)
			updated = true
		}
	}

	// Update resource limits or SSL
	if maxConnections >= 0 || requireSSL || noSSL {
		for _, host := range hosts {
			host = strings.TrimSpace(host)
			if host == "" {
				continue
			}

			alterCmd := fmt.Sprintf("ALTER USER '%s'@'%s'", username, host)

			if requireSSL {
				alterCmd += " REQUIRE SSL"
			} else if noSSL {
				alterCmd += " REQUIRE NONE"
			}

			if maxConnections >= 0 {
				if requireSSL || noSSL {
					alterCmd += " "
				}
				if maxConnections == 0 {
					alterCmd += "WITH MAX_CONNECTIONS_PER_HOUR UNLIMITED"
				} else {
					alterCmd += fmt.Sprintf("WITH MAX_CONNECTIONS_PER_HOUR %d", maxConnections)
				}
			}

			alterCmd += ";"

			mysqlCmd := exec.Command("mysql", "-u", "root", "-p"+adminPass, "-e", alterCmd)
			if err := mysqlCmd.Run(); err != nil {
				fmt.Printf("Warning: Could not update settings for %s@%s: %v\n", username, host, err)
				continue
			}

			updated = true
			if requireSSL {
				fmt.Printf("SSL/TLS now required for '%s'@'%s'\n", username, host)
			}
			if noSSL {
				fmt.Printf("SSL/TLS requirement removed for '%s'@'%s'\n", username, host)
			}
			if maxConnections >= 0 {
				if maxConnections == 0 {
					fmt.Printf("Max connections set to unlimited for '%s'@'%s'\n", username, host)
				} else {
					fmt.Printf("Max connections/hour set to %d for '%s'@'%s'\n", maxConnections, username, host)
				}
			}
		}
	}

	// Flush privileges
	flushCmd := "FLUSH PRIVILEGES;"
	mysqlCmd := exec.Command("mysql", "-u", "root", "-p"+adminPass, "-e", flushCmd)
	mysqlCmd.Run()

	if !updated {
		fmt.Println("No changes specified. Use --privileges, --max-connections, --require-ssl, or --no-ssl")
	} else {
		fmt.Println("User settings updated successfully")
	}
}

func showMySQLUserInfo(username string) {
	fmt.Printf("MySQL User Information: %s\n", username)
	fmt.Println("─────────────────────────────────────────")

	// Load config to get admin password from defaults
	cfg, err := config.Load()
	var adminPass string

	if err == nil {
		if pass, ok := cfg.GetDefault("mysql_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		} else if pass, ok := cfg.GetDefault("mariadb_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		}
	}

	if adminPass == "" {
		fmt.Print("Enter MySQL/MariaDB admin password: ")
		fmt.Scanln(&adminPass)
	}

	// Get user hosts and info
	hostsCmd := fmt.Sprintf("SELECT Host FROM mysql.user WHERE User='%s';", username)
	output, err := exec.Command("mysql", "-u", "root", "-p"+adminPass, "-sNe", hostsCmd).Output()
	if err != nil {
		fmt.Printf("User not found: %s\n", username)
		return
	}

	hosts := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Show privileges for each host
	for _, host := range hosts {
		host = strings.TrimSpace(host)
		if host == "" {
			continue
		}

		fmt.Printf("\nHost: %s\n", host)

		// Get grants
		grantsCmd := fmt.Sprintf("SHOW GRANTS FOR '%s'@'%s';", username, host)
		grantsOutput, _ := exec.Command("mysql", "-u", "root", "-p"+adminPass, "-sNe", grantsCmd).Output()
		if grantsOutput != nil {
			for _, line := range strings.Split(string(grantsOutput), "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					fmt.Printf("   %s\n", line)
				}
			}
		}
	}
}

func showPostgresqlUserInfo(username string) {
	fmt.Printf("PostgreSQL User Information: %s\n", username)
	fmt.Println("─────────────────────────────────────────")

	// List user info using \du in PostgreSQL
	listCmd := `\du`

	psqlCmd := exec.Command("sudo", "-u", "postgres", "psql", "-c", listCmd)
	output, _ := psqlCmd.Output()

	// Simple display - PostgreSQL doesn't have as granular controls as MySQL
	lines := strings.Split(string(output), "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, username) {
			if !found {
				fmt.Printf("   %s\n", line)
				found = true
			}
		}
	}

	if !found {
		fmt.Printf("User '%s' not found\n", username)
	}
}

// MySQL/MariaDB database functions
func createMySQLDatabase(dbName, charset, collation string) {
	fmt.Printf("Creating MySQL database '%s'...\n", dbName)

	cfg, err := config.Load()
	var adminPass string

	if err == nil {
		if pass, ok := cfg.GetDefault("mysql_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		} else if pass, ok := cfg.GetDefault("mariadb_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		}
	}

	if adminPass == "" {
		fmt.Print("Enter MySQL/MariaDB admin password: ")
		fmt.Scanln(&adminPass)
	}

	createCmd := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET %s COLLATE %s;", dbName, charset, collation)

	mysqlCmd := exec.Command("mysql", "-u", "root", "-p"+adminPass, "-e", createCmd)
	if err := mysqlCmd.Run(); err != nil {
		fmt.Printf("Error creating database: %v\n", err)
		return
	}

	fmt.Printf("Database '%s' created successfully\n", dbName)
	fmt.Printf("   Charset: %s | Collation: %s\n", charset, collation)
}

func deleteMySQLDatabase(dbName string, force bool) {
	if !force {
		fmt.Printf("Are you sure you want to delete database '%s'? This cannot be undone!\n", dbName)
		fmt.Print("Type 'yes' to confirm: ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("Deletion cancelled")
			return
		}
	}

	fmt.Printf("Deleting MySQL database '%s'...\n", dbName)

	cfg, err := config.Load()
	var adminPass string

	if err == nil {
		if pass, ok := cfg.GetDefault("mysql_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		} else if pass, ok := cfg.GetDefault("mariadb_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		}
	}

	if adminPass == "" {
		fmt.Print("Enter MySQL/MariaDB admin password: ")
		fmt.Scanln(&adminPass)
	}

	deleteCmd := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", dbName)

	mysqlCmd := exec.Command("mysql", "-u", "root", "-p"+adminPass, "-e", deleteCmd)
	if err := mysqlCmd.Run(); err != nil {
		fmt.Printf("Error deleting database: %v\n", err)
		return
	}

	fmt.Printf("Database '%s' deleted successfully\n", dbName)
}

func listMySQLDatabases() {
	fmt.Println("MySQL Databases:")
	fmt.Println("─────────────────────────────────────────")

	cfg, err := config.Load()
	var adminPass string

	if err == nil {
		if pass, ok := cfg.GetDefault("mysql_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		} else if pass, ok := cfg.GetDefault("mariadb_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		}
	}

	if adminPass == "" {
		fmt.Print("Enter MySQL/MariaDB admin password: ")
		fmt.Scanln(&adminPass)
	}

	query := `SELECT 
		SCHEMA_NAME as 'Database',
		ROUND(SUM(DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024, 2) as 'Size(MB)',
		DEFAULT_CHARACTER_SET_NAME as 'Charset',
		DEFAULT_COLLATION_NAME as 'Collation'
	FROM INFORMATION_SCHEMA.SCHEMATA
	LEFT JOIN INFORMATION_SCHEMA.TABLES ON INFORMATION_SCHEMA.TABLES.TABLE_SCHEMA = INFORMATION_SCHEMA.SCHEMATA.SCHEMA_NAME
	GROUP BY SCHEMA_NAME, DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME
	ORDER BY SCHEMA_NAME;`

	mysqlCmd := exec.Command("mysql", "-u", "root", "-p"+adminPass, "-e", query)
	output, err := mysqlCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error listing databases: %v\n", err)
		return
	}
	fmt.Print(string(output))
}

func showMySQLDatabaseInfo(dbName string) {
	fmt.Printf("MySQL Database Information: %s\n", dbName)
	fmt.Println("─────────────────────────────────────────")

	cfg, err := config.Load()
	var adminPass string

	if err == nil {
		if pass, ok := cfg.GetDefault("mysql_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		} else if pass, ok := cfg.GetDefault("mariadb_root_password", "").(string); ok && pass != "" {
			adminPass = pass
		}
	}

	if adminPass == "" {
		fmt.Print("Enter MySQL/MariaDB admin password: ")
		fmt.Scanln(&adminPass)
	}

	// Database exists?
	checkCmd := fmt.Sprintf("SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = '%s';", dbName)
	mysqlCmd := exec.Command("mysql", "-u", "root", "-p"+adminPass, "-sNe", checkCmd)
	if err := mysqlCmd.Run(); err != nil {
		fmt.Printf("Database '%s' not found\n", dbName)
		return
	}

	// Get database info
	infoCmd := fmt.Sprintf(`
	SELECT 
		'Database:' as 'Info', '%s' as 'Value' UNION
	SELECT 'Charset:', DEFAULT_CHARACTER_SET_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME='%s' UNION
	SELECT 'Collation:', DEFAULT_COLLATION_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME='%s' UNION
	SELECT 'Tables:', CAST(COUNT(*) as CHAR) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA='%s' UNION
	SELECT 'Size (MB):', ROUND(SUM(DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024, 2) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA='%s';
	`, dbName, dbName, dbName, dbName, dbName)

	mysqlCmd = exec.Command("mysql", "-u", "root", "-p"+adminPass, "-e", infoCmd)
	output, _ := mysqlCmd.CombinedOutput()
	fmt.Print(string(output))
}

// PostgreSQL database functions
func createPostgresqlDatabase(dbName, owner string) {
	fmt.Printf("Creating PostgreSQL database '%s'...\n", dbName)

	createCmd := fmt.Sprintf("CREATE DATABASE \"%s\" OWNER %s;", dbName, owner)

	psqlCmd := exec.Command("sudo", "-u", "postgres", "psql", "-c", createCmd)
	if err := psqlCmd.Run(); err != nil {
		fmt.Printf("Error creating database: %v\n", err)
		return
	}

	fmt.Printf("PostgreSQL database '%s' created successfully\n", dbName)
	fmt.Printf("   Owner: %s\n", owner)
}

func deletePostgresqlDatabase(dbName string, force bool) {
	if !force {
		fmt.Printf("Are you sure you want to delete database '%s'? This cannot be undone!\n", dbName)
		fmt.Print("Type 'yes' to confirm: ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("Deletion cancelled")
			return
		}
	}

	fmt.Printf("Deleting PostgreSQL database '%s'...\n", dbName)

	// Terminate connections first
	terminateCmd := fmt.Sprintf(`
	SELECT pg_terminate_backend(pg_stat_activity.pid)
	FROM pg_stat_activity
	WHERE pg_stat_activity.datname = '%s' AND pid <> pg_backend_pid();
	`, dbName)

	psqlCmd := exec.Command("sudo", "-u", "postgres", "psql", "-c", terminateCmd)
	psqlCmd.Run() // Ignore errors

	// Drop database
	dropCmd := fmt.Sprintf("DROP DATABASE IF EXISTS \"%s\";", dbName)
	psqlCmd = exec.Command("sudo", "-u", "postgres", "psql", "-c", dropCmd)
	if err := psqlCmd.Run(); err != nil {
		fmt.Printf("Error deleting database: %v\n", err)
		return
	}

	fmt.Printf("PostgreSQL database '%s' deleted successfully\n", dbName)
}

func listPostgresqlDatabases() {
	fmt.Println("PostgreSQL Databases:")
	fmt.Println("─────────────────────────────────────────")

	query := `\l`

	psqlCmd := exec.Command("sudo", "-u", "postgres", "psql", "-c", query)
	output, err := psqlCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error listing databases: %v\n", err)
		return
	}
	fmt.Print(string(output))
}

func showPostgresqlDatabaseInfo(dbName string) {
	fmt.Printf("PostgreSQL Database Information: %s\n", dbName)
	fmt.Println("─────────────────────────────────────────")

	// Connect to specific database and get info
	query := fmt.Sprintf(`
	SELECT 'Database:' as Key, datname as Value FROM pg_database WHERE datname = '%s' UNION
	SELECT 'Owner:', pg_get_userbyid(datdba) FROM pg_database WHERE datname = '%s' UNION
	SELECT 'Tables:', CAST(COUNT(*) as TEXT) FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE' UNION
	SELECT 'Connections:', CAST(COUNT(*) as TEXT) FROM pg_stat_activity WHERE datname = '%s';
	`, dbName, dbName, dbName)

	psqlCmd := exec.Command("sudo", "-u", "postgres", "psql", "-c", query)
	output, err := psqlCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error retrieving database info: %v\n", err)
		return
	}
	fmt.Print(string(output))
}

func init() {
	rootCmd.AddCommand(dbCmd)

	// User management commands
	dbCmd.AddCommand(dbUserCmd)
	dbUserCmd.AddCommand(dbUserCreateCmd)
	dbUserCmd.AddCommand(dbUserDeleteCmd)
	dbUserCmd.AddCommand(dbUserListCmd)
	dbUserCmd.AddCommand(dbUserPasswordCmd)
	dbUserCmd.AddCommand(dbUserUpdateCmd)
	dbUserCmd.AddCommand(dbUserInfoCmd)

	// Database management commands
	dbCmd.AddCommand(dbDatabaseCmd)
	dbDatabaseCmd.AddCommand(dbDatabaseCreateCmd)
	dbDatabaseCmd.AddCommand(dbDatabaseDeleteCmd)
	dbDatabaseCmd.AddCommand(dbDatabaseListCmd)
	dbDatabaseCmd.AddCommand(dbDatabaseInfoCmd)

	// Initialize flags
	init_dbUserCreateCmd()
	init_dbUserUpdateCmd()
	init_dbDatabaseCreateCmd()
	init_dbDatabaseDeleteCmd()
}
