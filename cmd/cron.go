package cmd

import (
	"fmt"
	"os"
	"strings"

	"webstack-cli/internal/cron"

	"github.com/spf13/cobra"
)

// cronCmd represents the cron command
var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Manage scheduled cron jobs",
	Long: `Manage scheduled cron jobs for WebStack services.

Cron jobs are stored in /var/spool/cron/crontabs/root and include:
- WebStack backup schedules
- SSL certificate renewal
- Log rotation and cleanup
- Database maintenance
- Custom user tasks

Usage:
  webstack cron add "0 2 * * *" "sudo webstack backup create --all"  # Daily backup at 2 AM
  webstack cron list                                                   # Show all jobs
  webstack cron edit 1                                                 # Edit job 1
  webstack cron delete 1                                               # Delete job 1
  webstack cron run 1                                                  # Run job immediately
  webstack cron status                                                 # Show status
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(cmd.Help())
	},
}

// cronAddCmd adds a new cron job
var cronAddCmd = &cobra.Command{
	Use:   "add [schedule] [command]",
	Short: "Add a new cron job",
	Long: `Add a new scheduled cron job.

Schedule format (crontab format):
  Minute(0-59) Hour(0-23) Day(1-31) Month(1-12) DayOfWeek(0-6)

Examples:
  0 2 * * *     - Daily at 2 AM
  */15 * * * *  - Every 15 minutes
  0 3 * * 0     - Every Sunday at 3 AM
  0 */6 * * *   - Every 6 hours
  30 1 1 * *    - Monthly on 1st at 1:30 AM

Command examples:
  sudo webstack backup create --all
  sudo webstack ssl renew
  sudo webstack system cleanup
  sudo certbot renew --quiet
  sudo mysql -e "OPTIMIZE TABLE ..."
`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		schedule := args[0]
		command := args[1]
		description, _ := cmd.Flags().GetString("description")

		jobID, err := cron.AddJob(schedule, command, description)
		if err != nil {
			fmt.Printf("❌ Failed to add cron job: %v\n", err)
			return
		}

		fmt.Printf("✅ Cron job added successfully\n")
		fmt.Printf("   ID: %d\n", jobID)
		fmt.Printf("   Schedule: %s\n", schedule)
		fmt.Printf("   Command: %s\n", command)
		if description != "" {
			fmt.Printf("   Description: %s\n", description)
		}
		fmt.Printf("\n   Commands:\n")
		fmt.Printf("   - View: webstack cron list | grep %d\n", jobID)
		fmt.Printf("   - Edit: webstack cron edit %d\n", jobID)
		fmt.Printf("   - Run now: webstack cron run %d\n", jobID)
		fmt.Printf("   - Delete: webstack cron delete %d\n", jobID)
	},
}

// cronListCmd lists all cron jobs
var cronListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all cron jobs",
	Long: `Display all scheduled cron jobs with their details.

Shows:
  - Job ID
  - Schedule (crontab format)
  - Command to execute
  - Description (if set)
  - Last run status
`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		webstackOnly, _ := cmd.Flags().GetBool("webstack-only")
		jobs, err := cron.ListJobs(webstackOnly)
		if err != nil {
			fmt.Printf("❌ Failed to list cron jobs: %v\n", err)
			return
		}

		if len(jobs) == 0 {
			fmt.Println("No cron jobs found")
			return
		}

		fmt.Println("Scheduled Cron Jobs:")
		fmt.Println(strings.Repeat("─", 100))
		fmt.Printf("%-4s %-20s %-15s %-55s %-5s\n", "ID", "Schedule", "Type", "Command", "Status")
		fmt.Println(strings.Repeat("─", 100))

		for _, job := range jobs {
			jobType := "custom"
			if strings.Contains(job.Command, "webstack") {
				jobType = "webstack"
			}

			status := "✓"
			if !job.Enabled {
				status = "⊘"
			}

			// Truncate command for display
			cmdDisplay := job.Command
			if len(cmdDisplay) > 55 {
				cmdDisplay = cmdDisplay[:52] + "..."
			}

			fmt.Printf("%-4d %-20s %-15s %-55s %-5s\n", job.ID, job.Schedule, jobType, cmdDisplay, status)
		}

		fmt.Println(strings.Repeat("─", 100))
		fmt.Printf("Total: %d cron jobs\n", len(jobs))

		// Count by type
		webstackCount := 0
		for _, job := range jobs {
			if strings.Contains(job.Command, "webstack") {
				webstackCount++
			}
		}
		fmt.Printf("  - WebStack: %d\n", webstackCount)
		fmt.Printf("  - Custom: %d\n", len(jobs)-webstackCount)
	},
}

// cronEditCmd edits a cron job
var cronEditCmd = &cobra.Command{
	Use:   "edit [job-id]",
	Short: "Edit a cron job",
	Long: `Edit an existing cron job's schedule or command.

You can update:
  - Schedule (crontab format)
  - Command to execute
  - Description
  - Enable/disable status
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		jobID := 0
		fmt.Sscanf(args[0], "%d", &jobID)

		job, err := cron.GetJob(jobID)
		if err != nil {
			fmt.Printf("❌ Cron job not found: %v\n", err)
			return
		}

		fmt.Printf("Editing cron job %d:\n", jobID)
		fmt.Printf("Current schedule: %s\n", job.Schedule)
		fmt.Printf("Current command: %s\n", job.Command)

		// Get new values from flags
		newSchedule, _ := cmd.Flags().GetString("schedule")
		newCommand, _ := cmd.Flags().GetString("command")
		newDescription, _ := cmd.Flags().GetString("description")

		if newSchedule == "" && newCommand == "" && newDescription == "" {
			fmt.Println("ℹ️  Use --schedule, --command, or --description to update")
			return
		}

		if newSchedule == "" {
			newSchedule = job.Schedule
		}
		if newCommand == "" {
			newCommand = job.Command
		}
		if newDescription == "" {
			newDescription = job.Description
		}

		if err := cron.UpdateJob(jobID, newSchedule, newCommand, newDescription); err != nil {
			fmt.Printf("❌ Failed to update cron job: %v\n", err)
			return
		}

		fmt.Printf("✅ Cron job %d updated\n", jobID)
		fmt.Printf("   New schedule: %s\n", newSchedule)
		fmt.Printf("   New command: %s\n", newCommand)
	},
}

// cronDeleteCmd deletes a cron job
var cronDeleteCmd = &cobra.Command{
	Use:   "delete [job-id]",
	Short: "Delete a cron job",
	Long: `Remove a scheduled cron job permanently.

The job is immediately removed from the cron schedule.
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		jobID := 0
		fmt.Sscanf(args[0], "%d", &jobID)

		job, err := cron.GetJob(jobID)
		if err != nil {
			fmt.Printf("❌ Cron job not found\n")
			return
		}

		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("⚠️  This will delete cron job: %d\n", jobID)
			fmt.Printf("   Schedule: %s\n", job.Schedule)
			fmt.Printf("   Command: %s\n", job.Command)
			fmt.Print("Type 'yes' to confirm: ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "yes" {
				fmt.Println("Delete cancelled")
				return
			}
		}

		if err := cron.DeleteJob(jobID); err != nil {
			fmt.Printf("❌ Failed to delete cron job: %v\n", err)
			return
		}

		fmt.Printf("✅ Cron job %d deleted\n", jobID)
	},
}

// cronRunCmd runs a cron job immediately
var cronRunCmd = &cobra.Command{
	Use:   "run [job-id]",
	Short: "Run a cron job immediately",
	Long: `Execute a scheduled cron job immediately (without waiting for schedule).

Useful for testing or running a job outside its normal schedule.
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		jobID := 0
		fmt.Sscanf(args[0], "%d", &jobID)

		job, err := cron.GetJob(jobID)
		if err != nil {
			fmt.Printf("❌ Cron job not found\n")
			return
		}

		fmt.Printf("🔄 Running cron job %d...\n", jobID)
		fmt.Printf("   Schedule: %s\n", job.Schedule)
		fmt.Printf("   Command: %s\n\n", job.Command)

		exitCode, err := cron.RunJob(jobID)
		if err != nil {
			fmt.Printf("❌ Failed to run cron job: %v\n", err)
			return
		}

		fmt.Printf("\n✅ Cron job %d completed\n", jobID)
		fmt.Printf("   Exit code: %d\n", exitCode)
	},
}

// cronEnableCmd enables a cron job
var cronEnableCmd = &cobra.Command{
	Use:   "enable [job-id]",
	Short: "Enable a disabled cron job",
	Long: `Re-enable a previously disabled cron job.

The job will resume its normal schedule.
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		jobID := 0
		fmt.Sscanf(args[0], "%d", &jobID)

		if err := cron.EnableJob(jobID); err != nil {
			fmt.Printf("❌ Failed to enable cron job: %v\n", err)
			return
		}

		fmt.Printf("✅ Cron job %d enabled\n", jobID)
	},
}

// cronDisableCmd disables a cron job
var cronDisableCmd = &cobra.Command{
	Use:   "disable [job-id]",
	Short: "Disable a cron job",
	Long: `Temporarily disable a cron job without deleting it.

The job remains in the list but won't execute. Re-enable with 'enable' command.
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		jobID := 0
		fmt.Sscanf(args[0], "%d", &jobID)

		if err := cron.DisableJob(jobID); err != nil {
			fmt.Printf("❌ Failed to disable cron job: %v\n", err)
			return
		}

		fmt.Printf("✅ Cron job %d disabled\n", jobID)
	},
}

// cronStatusCmd shows cron status
var cronStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show cron system status",
	Long: `Display the status of the cron scheduler and statistics.

Shows:
  - Total jobs configured
  - WebStack-specific jobs
  - Custom user jobs
  - Recent job execution logs
  - Next scheduled jobs
`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		status, err := cron.GetStatus()
		if err != nil {
			fmt.Printf("❌ Failed to get cron status: %v\n", err)
			return
		}

		fmt.Println("Cron Scheduler Status:")
		fmt.Println("─────────────────────────────────────────")
		fmt.Printf("Total Jobs:      %d\n", status.TotalJobs)
		fmt.Printf("WebStack Jobs:   %d\n", status.WebStackJobs)
		fmt.Printf("Custom Jobs:     %d\n", status.CustomJobs)
		fmt.Printf("Enabled:         %d\n", status.EnabledJobs)
		fmt.Printf("Disabled:        %d\n", status.DisabledJobs)
		fmt.Printf("System Status:   %s\n", status.SystemStatus)

		if status.LastJobTime != "" {
			fmt.Printf("Last Job Run:    %s\n", status.LastJobTime)
		}

		if status.NextJobTime != "" {
			fmt.Printf("Next Job Due:    %s\n", status.NextJobTime)
		}
	},
}

// cronLogsCmd shows cron logs
var cronLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show recent cron job logs",
	Long: `Display recent cron job execution logs from the system.

Shows when jobs ran and their exit status.
`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Geteuid() != 0 {
			fmt.Println("This command requires root privileges (use sudo)")
			return
		}

		lines, _ := cmd.Flags().GetInt("lines")
		pattern, _ := cmd.Flags().GetString("filter")

		logs, err := cron.GetLogs(lines, pattern)
		if err != nil {
			fmt.Printf("❌ Failed to get cron logs: %v\n", err)
			return
		}

		if len(logs) == 0 {
			fmt.Println("No cron logs found")
			return
		}

		fmt.Printf("Recent Cron Logs (last %d lines):\n", lines)
		fmt.Println(strings.Repeat("─", 120))
		for _, log := range logs {
			fmt.Println(log)
		}
	},
}

func init() {
	rootCmd.AddCommand(cronCmd)

	// Add subcommands
	cronCmd.AddCommand(cronAddCmd)
	cronCmd.AddCommand(cronListCmd)
	cronCmd.AddCommand(cronEditCmd)
	cronCmd.AddCommand(cronDeleteCmd)
	cronCmd.AddCommand(cronRunCmd)
	cronCmd.AddCommand(cronEnableCmd)
	cronCmd.AddCommand(cronDisableCmd)
	cronCmd.AddCommand(cronStatusCmd)
	cronCmd.AddCommand(cronLogsCmd)

	// Add command flags
	cronAddCmd.Flags().StringP("description", "d", "", "Description for the cron job")

	cronEditCmd.Flags().StringP("schedule", "s", "", "New crontab schedule")
	cronEditCmd.Flags().StringP("command", "c", "", "New command to execute")
	cronEditCmd.Flags().StringP("description", "d", "", "New description")

	cronDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	cronListCmd.Flags().BoolP("webstack-only", "w", false, "Show only WebStack cron jobs")

	cronLogsCmd.Flags().IntP("lines", "n", 50, "Number of log lines to display")
	cronLogsCmd.Flags().StringP("filter", "f", "", "Filter logs by pattern")
}
