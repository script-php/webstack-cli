package cmd

import (
	"fmt"
	"os"

	"webstack-cli/internal/backup"

	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage system backups",
	Long:  `Create, list, restore, and manage backups of domains, databases, and configurations.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use 'webstack backup --help' for available commands")
	},
}

var backupCreateCmd = &cobra.Command{
	Use:   "create [type]",
	Short: "Create a new backup",
	Long: `Create a backup of domains, databases, or entire system.
Usage:
  webstack backup create --all                          # Full system backup
  webstack backup create --domain example.com           # Single domain
  webstack backup create --all --compress gzip          # With compression
  webstack backup create --mysql wordpress              # Single MySQL database
  webstack backup create --postgresql crm               # Single PostgreSQL database`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		backupAll, _ := cmd.Flags().GetBool("all")
		domain, _ := cmd.Flags().GetString("domain")
		mysqlDB, _ := cmd.Flags().GetString("mysql")
		postgresDB, _ := cmd.Flags().GetString("postgresql")
		compression, _ := cmd.Flags().GetString("compress")
		encryption, _ := cmd.Flags().GetString("encrypt")

		// Determine backup type and scope
		var backupType, scope string

		if backupAll {
			backupType = "full"
			scope = "all"
		} else if domain != "" {
			backupType = "domain"
			scope = domain
		} else if mysqlDB != "" {
			backupType = "database"
			scope = "mysql:" + mysqlDB
		} else if postgresDB != "" {
			backupType = "database"
			scope = "postgresql:" + postgresDB
		} else {
			fmt.Println("Please specify --all, --domain, --mysql, or --postgresql")
			return
		}

		opts := backup.BackupOptions{
			Type:        backupType,
			Scope:       scope,
			Compression: compression,
			Encryption:  encryption,
		}

		backupID, size, compressedSize, err := backup.Create(opts)
		if err != nil {
			fmt.Printf("❌ Backup failed: %v\n", err)
			return
		}

		backupPath := backup.GetBackupPath(backupID)
		fmt.Printf("✅ Backup created successfully\n")
		fmt.Printf("   ID: %s\n", backupID)
		fmt.Printf("   Location: %s\n", backupPath)
		fmt.Printf("   Type: %s (%s)\n", backupType, scope)
		fmt.Printf("   Size: %s → %s (compressed)\n",
			backup.FormatBytes(size), backup.FormatBytes(compressedSize))
		fmt.Printf("\n   Commands:\n")
		fmt.Printf("   - List details: webstack backup list | grep %s\n", backupID[:8])
		fmt.Printf("   - Restore: sudo webstack backup restore %s\n", backupID)
		fmt.Printf("   - Export: sudo webstack backup export %s /path/to/file.tar.gz\n", backupID)
		fmt.Printf("   - Verify: sudo webstack backup verify %s\n", backupID)
	},
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all backups",
	Long: `Show all available backups with details.
Usage:
  webstack backup list                        # All backups
  webstack backup list --domain example.com   # Backups for domain
  webstack backup list --since 7d             # Last 7 days
  webstack backup list --format json          # JSON output`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		domain, _ := cmd.Flags().GetString("domain")
		since, _ := cmd.Flags().GetString("since")
		format, _ := cmd.Flags().GetString("format")

		backups, err := backup.List(domain, since)
		if err != nil {
			fmt.Printf("❌ Error listing backups: %v\n", err)
			return
		}

		if len(backups) == 0 {
			fmt.Println("No backups found")
			return
		}

		if format == "json" {
			backup.PrintJSON(backups)
			return
		}

		fmt.Println("Available Backups:")
		fmt.Println("─────────────────────────────────────────────────────────────────")
		fmt.Printf("%-20s %-15s %-20s %-15s %-12s\n", "ID", "Type", "Created", "Size", "Compressed")
		fmt.Println("─────────────────────────────────────────────────────────────────")

		for _, b := range backups {
			idShort := b.ID
			if len(idShort) > 20 {
				idShort = idShort[:17] + "..."
			}
			fmt.Printf("%-20s %-15s %-20s %-15s %-12s\n",
				idShort,
				b.Type,
				b.Timestamp.Format("2006-01-02 15:04"),
				backup.FormatBytes(b.SizeBytes),
				backup.FormatBytes(b.CompressedSize),
			)
		}

		fmt.Printf("\nTotal: %d backups | Total size: %s\n",
			len(backups),
			backup.FormatBytes(backup.GetTotalSize(backups)),
		)
		fmt.Printf("\nBackup location: /var/backups/webstack/archives/\n")
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore [backup-id]",
	Short: "Restore from a backup",
	Long: `Restore system from a backup.
Usage:
  webstack backup restore abc123                  # Restore full backup
  webstack backup restore abc123 --domain example.com  # Restore single domain
  webstack backup restore abc123 --verify-only   # Check backup integrity
  webstack backup restore abc123 --force          # Skip confirmation`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		backupID := args[0]
		domain, _ := cmd.Flags().GetString("domain")
		verifyOnly, _ := cmd.Flags().GetBool("verify-only")
		force, _ := cmd.Flags().GetBool("force")

		if verifyOnly {
			fmt.Printf("🔍 Verifying backup integrity: %s\n", backupID)
			ok, err := backup.Verify(backupID)
			if err != nil {
				fmt.Printf("❌ Verification failed: %v\n", err)
				return
			}
			if ok {
				fmt.Println("✅ Backup integrity verified - safe to restore")
			}
			return
		}

		if !force {
			fmt.Printf("⚠️  This will restore from backup: %s\n", backupID)
			if domain != "" {
				fmt.Printf("   Domain: %s\n", domain)
			} else {
				fmt.Println("   Scope: Full system")
			}
			fmt.Print("Type 'yes' to confirm: ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "yes" {
				fmt.Println("Restore cancelled")
				return
			}
		}

		fmt.Printf("📥 Starting restore from backup: %s\n", backupID)
		itemsRestored, err := backup.Restore(backupID, domain)
		if err != nil {
			fmt.Printf("❌ Restore failed: %v\n", err)
			return
		}

		fmt.Printf("✅ Restore completed successfully\n")
		fmt.Printf("   Items restored: %d\n", itemsRestored)
		fmt.Println("   Next steps:")
		fmt.Println("   - Verify your sites are working: webstack domain list")
		fmt.Println("   - Check service status: webstack system status")
		fmt.Println("   - Reload configs if needed: webstack system reload")
	},
}

var backupDeleteCmd = &cobra.Command{
	Use:   "delete [backup-id]",
	Short: "Delete a backup",
	Long: `Remove a backup from storage.
Usage:
  webstack backup delete abc123           # Delete specific backup
  webstack backup delete abc123 --force   # Skip confirmation`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		backupID := args[0]
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("⚠️  Delete backup: %s\n", backupID)
			fmt.Print("Type 'yes' to confirm: ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "yes" {
				fmt.Println("Deletion cancelled")
				return
			}
		}

		err := backup.Delete(backupID)
		if err != nil {
			fmt.Printf("❌ Delete failed: %v\n", err)
			return
		}

		fmt.Printf("✅ Backup deleted: %s\n", backupID)
	},
}

var backupVerifyCmd = &cobra.Command{
	Use:   "verify [backup-id]",
	Short: "Verify backup integrity",
	Long: `Check if a backup is valid and can be restored.
Usage:
  webstack backup verify abc123   # Verify specific backup`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		backupID := args[0]

		fmt.Printf("🔍 Verifying backup: %s\n", backupID)
		ok, err := backup.Verify(backupID)
		if err != nil {
			fmt.Printf("❌ Verification failed: %v\n", err)
			return
		}

		if ok {
			fmt.Println("✅ Backup is valid and ready to restore")
		} else {
			fmt.Println("❌ Backup integrity check failed")
		}
	},
}

var backupScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Configure automatic backups",
	Long:  `Set up automatic scheduled backups.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use 'webstack backup schedule --help' for available commands")
	},
}

var backupScheduleEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable automatic backups",
	Long: `Set up automatic daily backups.
Usage:
  webstack backup schedule enable --time 02:00 --type full --keep 30
  webstack backup schedule enable --time 03:00 --type full --compress gzip`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		backupTime, _ := cmd.Flags().GetString("time")
		backupType, _ := cmd.Flags().GetString("type")
		keepDays, _ := cmd.Flags().GetInt("keep")
		compression, _ := cmd.Flags().GetString("compress")

		if backupTime == "" {
			backupTime = "02:00"
		}
		if backupType == "" {
			backupType = "full"
		}
		if keepDays == 0 {
			keepDays = 30
		}

		fmt.Printf("📅 Enabling automatic backups\n")
		fmt.Printf("   Time: %s UTC daily\n", backupTime)
		fmt.Printf("   Type: %s\n", backupType)
		fmt.Printf("   Retention: %d days\n", keepDays)

		err := backup.EnableSchedule(backupTime, backupType, keepDays, compression)
		if err != nil {
			fmt.Printf("❌ Failed to enable schedule: %v\n", err)
			return
		}

		fmt.Println("✅ Automatic backups enabled")
		fmt.Println("   Check status: webstack backup status")
		fmt.Println("   View logs: sudo journalctl -u webstack-backup.timer -f")
	},
}

var backupScheduleDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable automatic backups",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		err := backup.DisableSchedule()
		if err != nil {
			fmt.Printf("❌ Failed to disable schedule: %v\n", err)
			return
		}

		fmt.Println("✅ Automatic backups disabled")
	},
}

var backupScheduleStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show backup schedule status",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		enabled, nextRun, err := backup.GetScheduleStatus()
		if err != nil {
			fmt.Printf("❌ Error getting schedule status: %v\n", err)
			return
		}

		if !enabled {
			fmt.Println("❌ Automatic backups are disabled")
			fmt.Println("   Enable with: webstack backup schedule enable")
			return
		}

		fmt.Println("✅ Automatic backups are enabled")
		fmt.Printf("   Next backup: %s\n", nextRun.Format("2006-01-02 15:04 UTC"))
		fmt.Println("   View logs: sudo journalctl -u webstack-backup.timer -f")
	},
}

var backupStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show backup storage status",
	Long: `Display backup storage usage and statistics.
Usage:
  webstack backup status   # Show storage info`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		info, err := backup.GetStorageStatus()
		if err != nil {
			fmt.Printf("❌ Error getting storage status: %v\n", err)
			return
		}

		fmt.Println("Backup Storage Status:")
		fmt.Println("─────────────────────────────────────────")
		fmt.Printf("Location: %s\n", info.Location)
		fmt.Printf("Total Backups: %d\n", info.BackupCount)
		fmt.Printf("Total Size: %.2f GB\n", float64(info.TotalSize)/1e9)
		fmt.Printf("Available Space: %.2f GB\n", float64(info.AvailableSpace)/1e9)

		percentUsed := float64(info.TotalSize) / float64(info.TotalSpace) * 100
		fmt.Printf("Space Used: %.1f%%\n", percentUsed)

		if percentUsed > 90 {
			fmt.Println("⚠️  Warning: Storage usage is high!")
		}

		if info.ScheduleEnabled {
			fmt.Printf("Scheduled Backups: Enabled (next: %s)\n", info.NextBackup.Format("2006-01-02 15:04"))
		} else {
			fmt.Println("Scheduled Backups: Disabled")
		}
	},
}

var backupExportCmd = &cobra.Command{
	Use:   "export [backup-id] [destination]",
	Short: "Export backup to file",
	Long: `Export a backup to an external location.
Usage:
  webstack backup export abc123 /mnt/external/backup.tar.gz`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		backupID := args[0]
		destination := args[1]

		fmt.Printf("📤 Exporting backup: %s\n", backupID)
		fmt.Printf("   To: %s\n", destination)

		err := backup.Export(backupID, destination)
		if err != nil {
			fmt.Printf("❌ Export failed: %v\n", err)
			return
		}

		fmt.Println("✅ Backup exported successfully")
	},
}

var backupImportCmd = &cobra.Command{
	Use:   "import [source]",
	Short: "Import backup from file",
	Long: `Import a backup from an external file.
Usage:
  webstack backup import /mnt/external/backup.tar.gz`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		source := args[0]

		fmt.Printf("📥 Importing backup from: %s\n", source)

		backupID, err := backup.Import(source)
		if err != nil {
			fmt.Printf("❌ Import failed: %v\n", err)
			return
		}

		fmt.Printf("✅ Backup imported successfully\n")
		fmt.Printf("   ID: %s\n", backupID)
		fmt.Printf("   Restore with: webstack backup restore %s\n", backupID)
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupDeleteCmd)
	backupCmd.AddCommand(backupVerifyCmd)
	backupCmd.AddCommand(backupScheduleCmd)
	backupCmd.AddCommand(backupStatusCmd)
	backupCmd.AddCommand(backupExportCmd)
	backupCmd.AddCommand(backupImportCmd)

	// Schedule subcommands
	backupScheduleCmd.AddCommand(backupScheduleEnableCmd)
	backupScheduleCmd.AddCommand(backupScheduleDisableCmd)
	backupScheduleCmd.AddCommand(backupScheduleStatusCmd)

	// Create flags
	backupCreateCmd.Flags().BoolP("all", "a", false, "Backup entire system")
	backupCreateCmd.Flags().StringP("domain", "d", "", "Domain name to backup")
	backupCreateCmd.Flags().String("mysql", "", "MySQL database name")
	backupCreateCmd.Flags().String("postgresql", "", "PostgreSQL database name")
	backupCreateCmd.Flags().StringP("compress", "c", "gzip", "Compression: gzip, bzip2, xz, none")
	backupCreateCmd.Flags().StringP("encrypt", "e", "none", "Encryption: none, aes-256")

	// List flags
	backupListCmd.Flags().StringP("domain", "d", "", "Filter by domain")
	backupListCmd.Flags().StringP("since", "s", "", "Filter by time (e.g., 7d, 30d, 1y)")
	backupListCmd.Flags().StringP("format", "f", "table", "Output format: table, json")

	// Restore flags
	backupRestoreCmd.Flags().StringP("domain", "d", "", "Restore specific domain only")
	backupRestoreCmd.Flags().BoolP("verify-only", "v", false, "Verify backup without restoring")
	backupRestoreCmd.Flags().BoolP("force", "f", false, "Skip confirmation")

	// Delete flags
	backupDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation")

	// Schedule flags
	backupScheduleEnableCmd.Flags().StringP("time", "t", "02:00", "Backup time in HH:MM format")
	backupScheduleEnableCmd.Flags().StringP("type", "T", "full", "Backup type: full, incremental")
	backupScheduleEnableCmd.Flags().IntP("keep", "k", 30, "Keep backups for N days")
	backupScheduleEnableCmd.Flags().StringP("compress", "c", "gzip", "Compression: gzip, bzip2, xz, none")
}
