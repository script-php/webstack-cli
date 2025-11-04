package backup

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"
	"webstack-cli/internal/cron"
)

// BackupSchedule represents a backup schedule configuration
type BackupSchedule struct {
	Enabled       bool
	Frequency     string // "daily", "weekly", "monthly"
	Time          string // "HH:MM"
	Type          string // "full", "incremental"
	RetentionDays int
	Compression   string
}

const systemdServiceFile = "/etc/systemd/system/webstack-backup.service"
const systemdTimerFile = "/etc/systemd/system/webstack-backup.timer"
const scheduleConfigFile = "/etc/webstack/backup-schedule.conf"

// EnableSchedule enables automatic backups with systemd timer
func EnableSchedule(time, backupType string, retentionDays int, compression string) error {
	if compression == "" {
		compression = "gzip"
	}

	// Create service file
	serviceContent := fmt.Sprintf(`[Unit]
Description=WebStack Automatic Backup
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/webstack backup create --all --compress %s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=webstack-backup

[Install]
WantedBy=multi-user.target
`, compression)

	if err := ioutil.WriteFile(systemdServiceFile, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to create service file: %w", err)
	}

	// Create timer file
	timerContent := fmt.Sprintf(`[Unit]
Description=WebStack Daily Backup Timer
Requires=webstack-backup.service

[Timer]
OnCalendar=daily
OnCalendar=*-*-* %s:00
Persistent=true
OnBootSec=5min

[Install]
WantedBy=timers.target
`, time)

	if err := ioutil.WriteFile(systemdTimerFile, []byte(timerContent), 0644); err != nil {
		return fmt.Errorf("failed to create timer file: %w", err)
	}

	// Reload systemd daemon
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Enable timer
	if err := exec.Command("systemctl", "enable", "webstack-backup.timer").Run(); err != nil {
		return fmt.Errorf("failed to enable timer: %w", err)
	}

	// Start timer
	if err := exec.Command("systemctl", "start", "webstack-backup.timer").Run(); err != nil {
		return fmt.Errorf("failed to start timer: %w", err)
	}

	// Save schedule configuration
	schedule := BackupSchedule{
		Enabled:       true,
		Frequency:     "daily",
		Time:          time,
		Type:          backupType,
		RetentionDays: retentionDays,
		Compression:   compression,
	}

	if err := saveScheduleConfig(schedule); err != nil {
		return fmt.Errorf("failed to save schedule config: %w", err)
	}

	// Setup cleanup cron job
	if err := setupCleanupCron(retentionDays); err != nil {
		fmt.Printf("⚠️  Warning: Could not setup cleanup cron: %v\n", err)
	}

	return nil
}

// DisableSchedule disables automatic backups
func DisableSchedule() error {
	// Stop and disable timer
	exec.Command("systemctl", "stop", "webstack-backup.timer").Run()
	exec.Command("systemctl", "disable", "webstack-backup.timer").Run()

	// Remove systemd files
	os.Remove(systemdServiceFile)
	os.Remove(systemdTimerFile)

	// Reload systemd daemon
	exec.Command("systemctl", "daemon-reload").Run()

	// Update schedule config
	schedule := BackupSchedule{Enabled: false}
	saveScheduleConfig(schedule)

	// Remove cleanup cron
	removecleanupCron()

	return nil
}

// GetScheduleStatus returns schedule status information
func GetScheduleStatus() (bool, time.Time, error) {
	schedule, err := loadScheduleConfig()
	if err != nil {
		return false, time.Time{}, err
	}

	if !schedule.Enabled {
		return false, time.Time{}, nil
	}

	// Parse schedule time
	var hour, minute int
	fmt.Sscanf(schedule.Time, "%d:%d", &hour, &minute)

	// Calculate next run
	now := time.Now()
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

	if nextRun.Before(now) {
		nextRun = nextRun.AddDate(0, 0, 1)
	}

	return true, nextRun, nil
}

// setupCleanupCron sets up automatic cleanup of old backups
func setupCleanupCron(retentionDays int) error {
	// Create a cleanup script
	cleanupScript := fmt.Sprintf(`#!/bin/bash
# Cleanup old WebStack backups
find /var/backups/webstack/data -type d -name "backup-*" -mtime +%d -exec rm -rf {} \; 2>/dev/null
find /var/backups/webstack/metadata -type f -name "*.json" -mtime +%d -exec rm {} \; 2>/dev/null
`, retentionDays, retentionDays)

	scriptPath := "/usr/local/bin/webstack-backup-cleanup.sh"
	if err := ioutil.WriteFile(scriptPath, []byte(cleanupScript), 0755); err != nil {
		return fmt.Errorf("failed to create cleanup script: %w", err)
	}

	// Add to crontab
	cronjobLine := fmt.Sprintf("0 4 * * * %s\n", scriptPath)

	// Get current crontab
	cmd := exec.Command("crontab", "-l")
	output, _ := cmd.Output() // Ignore error if no crontab exists
	currentCrontab := string(output)

	// Check if job already exists
	if strings.Contains(currentCrontab, "webstack-backup-cleanup.sh") {
		// Register with cron manager
		cron.RegisterSystemCron("0 4 * * *", scriptPath, "Cleanup old backups (keep last 30 days)", "backup")
		return nil
	}

	// Add new job
	newCrontab := currentCrontab + cronjobLine
	cmd = exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	// Register with cron manager
	cron.RegisterSystemCron("0 4 * * *", scriptPath, "Cleanup old backups (keep last 30 days)", "backup")

	return nil
}

// removecleanupCron removes the cleanup cron job
func removecleanupCron() error {
	// Get current crontab
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.Output()
	if err != nil {
		return nil // No crontab
	}

	// Remove cleanup job
	lines := strings.Split(string(output), "\n")
	var newLines []string
	for _, line := range lines {
		if !strings.Contains(line, "webstack-backup-cleanup.sh") {
			newLines = append(newLines, line)
		}
	}

	// Update crontab
	newCrontab := strings.Join(newLines, "\n")
	cmd = exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update crontab: %w", err)
	}

	// Remove script
	os.Remove("/usr/local/bin/webstack-backup-cleanup.sh")

	return nil
}

// saveScheduleConfig saves schedule configuration
func saveScheduleConfig(schedule BackupSchedule) error {
	content := fmt.Sprintf(`# WebStack Backup Schedule Configuration
enabled=%v
frequency=%s
time=%s
type=%s
retention_days=%d
compression=%s
`, schedule.Enabled, schedule.Frequency, schedule.Time, schedule.Type, schedule.RetentionDays, schedule.Compression)

	return ioutil.WriteFile(scheduleConfigFile, []byte(content), 0644)
}

// loadScheduleConfig loads schedule configuration
func loadScheduleConfig() (*BackupSchedule, error) {
	data, err := ioutil.ReadFile(scheduleConfigFile)
	if err != nil {
		return &BackupSchedule{Enabled: false}, nil
	}

	schedule := &BackupSchedule{}
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "enabled":
			schedule.Enabled = value == "true"
		case "frequency":
			schedule.Frequency = value
		case "time":
			schedule.Time = value
		case "type":
			schedule.Type = value
		case "retention_days":
			fmt.Sscanf(value, "%d", &schedule.RetentionDays)
		case "compression":
			schedule.Compression = value
		}
	}

	return schedule, nil
}

// CleanupOldBackups removes backups older than retention days
func CleanupOldBackups(retentionDays int) (int, error) {
	entries, err := os.ReadDir(backupMetadataDir)
	if err != nil {
		return 0, err
	}

	deleted := 0
	now := time.Now()
	cutoffTime := now.AddDate(0, 0, -retentionDays)

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoffTime) {
			// Extract backup ID from filename
			backupID := strings.TrimSuffix(entry.Name(), ".json")

			if err := Delete(backupID); err != nil {
				fmt.Printf("⚠️  Could not delete old backup %s: %v\n", backupID, err)
				continue
			}

			deleted++
			fmt.Printf("✓ Deleted old backup: %s\n", backupID)
		}
	}

	return deleted, nil
}
