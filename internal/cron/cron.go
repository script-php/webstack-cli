package cron

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const cronDir = "/var/spool/cron/crontabs"
const cronUser = "root"
const cronMetadataDir = "/etc/webstack/cron"

// Job represents a cron job
type Job struct {
	ID          int       `json:"id"`
	Schedule    string    `json:"schedule"`
	Command     string    `json:"command"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	Created     time.Time `json:"created"`
	LastRun     time.Time `json:"last_run,omitempty"`
	LastStatus  int       `json:"last_status"`
	Source      string    `json:"source"` // "manual", "backup", "ssl", etc.
}

// Status represents cron system status
type Status struct {
	TotalJobs    int
	WebStackJobs int
	CustomJobs   int
	EnabledJobs  int
	DisabledJobs int
	SystemStatus string
	LastJobTime  string
	NextJobTime  string
}

// Initialize cron system
func init() {
	os.MkdirAll(cronMetadataDir, 0755)
	// Sync existing WebStack crons to metadata
	syncWebStackCrons()
}

// AddJob adds a new cron job
func AddJob(schedule, command, description string) (int, error) {
	// Validate schedule format
	if !isValidSchedule(schedule) {
		return 0, fmt.Errorf("invalid crontab schedule format: %s", schedule)
	}

	// Get next available ID
	jobID := getNextJobID()

	// Create job
	job := Job{
		ID:          jobID,
		Schedule:    schedule,
		Command:     command,
		Description: description,
		Enabled:     true,
		Created:     time.Now(),
		LastStatus:  0,
		Source:      "manual",
	}

	// Save job metadata
	if err := saveJobMetadata(job); err != nil {
		return 0, err
	}

	// Add to crontab
	if err := addJobToCrontab(job); err != nil {
		return 0, err
	}

	return jobID, nil
}

// ListJobs lists all cron jobs - discovers from both metadata and actual crontab
func ListJobs(webstackOnly bool) ([]Job, error) {
	// First, sync any crons that exist in crontab but not in metadata
	syncCrontabToDB()

	// Also sync systemd timers to metadata
	syncSystemdTimersToDB()

	// Now read from metadata
	files, err := ioutil.ReadDir(cronMetadataDir)
	if err != nil {
		return nil, err
	}

	var jobs []Job
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		data, err := ioutil.ReadFile(filepath.Join(cronMetadataDir, file.Name()))
		if err != nil {
			continue
		}

		var job Job
		if err := json.Unmarshal(data, &job); err != nil {
			continue
		}

		if webstackOnly && !strings.Contains(job.Command, "webstack") {
			continue
		}

		jobs = append(jobs, job)
	}

	// Sort by ID
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].ID < jobs[j].ID
	})

	return jobs, nil
}

// GetJob gets a specific cron job by ID
func GetJob(jobID int) (*Job, error) {
	metadataFile := filepath.Join(cronMetadataDir, fmt.Sprintf("job-%d.json", jobID))
	data, err := ioutil.ReadFile(metadataFile)
	if err != nil {
		return nil, fmt.Errorf("job not found: %d", jobID)
	}

	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, err
	}

	return &job, nil
}

// UpdateJob updates a cron job
func UpdateJob(jobID int, schedule, command, description string) error {
	// Validate schedule
	if !isValidSchedule(schedule) {
		return fmt.Errorf("invalid crontab schedule format: %s", schedule)
	}

	job, err := GetJob(jobID)
	if err != nil {
		return err
	}

	job.Schedule = schedule
	job.Command = command
	job.Description = description

	// Save updated metadata
	if err := saveJobMetadata(*job); err != nil {
		return err
	}

	// Update crontab
	if err := removeJobFromCrontab(jobID); err != nil {
		return err
	}

	if err := addJobToCrontab(*job); err != nil {
		return err
	}

	return nil
}

// DeleteJob deletes a cron job
func DeleteJob(jobID int) error {
	metadataFile := filepath.Join(cronMetadataDir, fmt.Sprintf("job-%d.json", jobID))

	// Remove from crontab
	if err := removeJobFromCrontab(jobID); err != nil {
		return err
	}

	// Remove metadata file
	if err := os.Remove(metadataFile); err != nil {
		return err
	}

	return nil
}

// RunJob runs a cron job immediately
func RunJob(jobID int) (int, error) {
	job, err := GetJob(jobID)
	if err != nil {
		return -1, err
	}

	// Execute command
	cmd := exec.Command("sh", "-c", job.Command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	// Update last run info
	job.LastRun = time.Now()
	job.LastStatus = exitCode
	saveJobMetadata(*job)

	return exitCode, nil
}

// EnableJob enables a disabled cron job
func EnableJob(jobID int) error {
	job, err := GetJob(jobID)
	if err != nil {
		return err
	}

	if job.Enabled {
		return fmt.Errorf("job %d is already enabled", jobID)
	}

	job.Enabled = true
	if err := saveJobMetadata(*job); err != nil {
		return err
	}

	// Re-add to crontab
	if err := addJobToCrontab(*job); err != nil {
		return err
	}

	return nil
}

// DisableJob disables a cron job (keeps it in metadata)
func DisableJob(jobID int) error {
	job, err := GetJob(jobID)
	if err != nil {
		return err
	}

	if !job.Enabled {
		return fmt.Errorf("job %d is already disabled", jobID)
	}

	job.Enabled = false
	if err := saveJobMetadata(*job); err != nil {
		return err
	}

	// Remove from crontab
	if err := removeJobFromCrontab(jobID); err != nil {
		return err
	}

	return nil
}

// GetStatus returns cron system status
func GetStatus() (*Status, error) {
	jobs, err := ListJobs(false)
	if err != nil {
		return nil, err
	}

	webstackCount := 0
	enabledCount := 0
	disabledCount := 0

	for _, job := range jobs {
		if strings.Contains(job.Command, "webstack") {
			webstackCount++
		}
		if job.Enabled {
			enabledCount++
		} else {
			disabledCount++
		}
	}

	// Check if cron daemon is running
	systemStatus := "✓ Running"
	if !isCronRunning() {
		systemStatus = "⊘ Not running"
	}

	status := &Status{
		TotalJobs:    len(jobs),
		WebStackJobs: webstackCount,
		CustomJobs:   len(jobs) - webstackCount,
		EnabledJobs:  enabledCount,
		DisabledJobs: disabledCount,
		SystemStatus: systemStatus,
	}

	// Get last job run time
	if len(jobs) > 0 {
		for _, job := range jobs {
			if !job.LastRun.IsZero() {
				status.LastJobTime = job.LastRun.Format("2006-01-02 15:04:05")
				break
			}
		}
	}

	return status, nil
}

// GetLogs returns recent cron logs
func GetLogs(lines int, pattern string) ([]string, error) {
	// Try to read from syslog or cron log
	logFile := "/var/log/syslog"
	if _, err := os.Stat("/var/log/cron"); err == nil {
		logFile = "/var/log/cron"
	}

	data, err := ioutil.ReadFile(logFile)
	if err != nil {
		return nil, err
	}

	allLines := strings.Split(string(data), "\n")
	var cronLogs []string

	// Filter for cron entries
	for _, line := range allLines {
		if strings.Contains(line, "CRON") || strings.Contains(line, "cron") {
			if pattern != "" {
				if matched, _ := regexp.MatchString(pattern, line); !matched {
					continue
				}
			}
			cronLogs = append(cronLogs, line)
		}
	}

	// Return last N lines
	if len(cronLogs) > lines {
		cronLogs = cronLogs[len(cronLogs)-lines:]
	}

	return cronLogs, nil
}

// Helper functions

// isValidSchedule validates crontab schedule format
func isValidSchedule(schedule string) bool {
	parts := strings.Fields(schedule)
	if len(parts) != 5 {
		return false
	}

	// Basic validation - just check if it's 5 fields
	// Full validation would check ranges, but this is sufficient
	return true
}

// getNextJobID gets the next available job ID
func getNextJobID() int {
	files, err := ioutil.ReadDir(cronMetadataDir)
	if err != nil {
		return 1
	}

	maxID := 0
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			var id int
			fmt.Sscanf(file.Name(), "job-%d.json", &id)
			if id > maxID {
				maxID = id
			}
		}
	}

	return maxID + 1
}

// saveJobMetadata saves job metadata to JSON
func saveJobMetadata(job Job) error {
	metadataFile := filepath.Join(cronMetadataDir, fmt.Sprintf("job-%d.json", job.ID))
	data, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(metadataFile, data, 0644)
}

// addJobToCrontab adds a job to the crontab
func addJobToCrontab(job Job) error {
	// Only add if enabled
	if !job.Enabled {
		return nil
	}

	cronContent, err := readCrontab()
	if err != nil {
		cronContent = ""
	}

	// Add webstack comment and job
	marker := fmt.Sprintf("# webstack-job-%d\n", job.ID)
	jobLine := fmt.Sprintf("%s %s\n", job.Schedule, job.Command)

	cronContent += marker + jobLine

	return writeCrontab(cronContent)
}

// removeJobFromCrontab removes a job from crontab
func removeJobFromCrontab(jobID int) error {
	cronContent, err := readCrontab()
	if err != nil {
		return nil // No crontab yet
	}

	// Remove the marker and job lines
	marker := fmt.Sprintf("# webstack-job-%d", jobID)
	lines := strings.Split(cronContent, "\n")
	var newLines []string
	skipNext := false

	for _, line := range lines {
		if strings.Contains(line, marker) {
			skipNext = true
			continue
		}
		if skipNext && strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			skipNext = false
			continue
		}
		newLines = append(newLines, line)
	}

	newContent := strings.Join(newLines, "\n")
	// Clean up multiple blank lines
	newContent = strings.ReplaceAll(newContent, "\n\n\n", "\n\n")
	return writeCrontab(newContent)
}

// readCrontab reads the current crontab
func readCrontab() (string, error) {
	cronFile := filepath.Join(cronDir, cronUser)
	data, err := ioutil.ReadFile(cronFile)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// writeCrontab writes to the crontab
func writeCrontab(content string) error {
	cronFile := filepath.Join(cronDir, cronUser)

	// Ensure directory exists
	os.MkdirAll(cronDir, 0700)

	// Write to temp file first
	tmpFile := cronFile + ".tmp"
	if err := ioutil.WriteFile(tmpFile, []byte(content), 0600); err != nil {
		return err
	}

	// Use crontab command to install
	cmd := exec.Command("crontab", tmpFile)
	if err := cmd.Run(); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to install crontab: %w", err)
	}

	// Remove temp file
	os.Remove(tmpFile)

	return nil
}

// isCronRunning checks if cron daemon is running
func isCronRunning() bool {
	cmd := exec.Command("systemctl", "is-active", "cron")
	return cmd.Run() == nil
}

// syncWebStackCrons discovers and syncs WebStack-created crons from actual crontab
func syncWebStackCrons() {
	cronContent, err := readCrontab()
	if err != nil {
		return // No crontab yet
	}

	// Pattern: # webstack-job-<id>
	pattern := regexp.MustCompile(`# webstack-(.*?)-(\d+)`)
	lines := strings.Split(cronContent, "\n")

	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "#") {
			match := pattern.FindStringSubmatch(lines[i])
			if len(match) >= 3 {
				source := match[1] // "backup", "ssl", etc.
				jobIDStr := match[2]

				// Next line should be the actual job
				if i+1 < len(lines) && lines[i+1] != "" {
					jobLine := lines[i+1]
					parts := strings.Fields(jobLine)
					if len(parts) >= 6 {
						schedule := strings.Join(parts[:5], " ")
						command := strings.Join(parts[5:], " ")

						var jobID int
						fmt.Sscanf(jobIDStr, "%d", &jobID)

						// Sync to metadata if not already there
						metadataFile := filepath.Join(cronMetadataDir, fmt.Sprintf("job-%d.json", jobID))
						if _, err := os.Stat(metadataFile); os.IsNotExist(err) {
							job := Job{
								ID:          jobID,
								Schedule:    schedule,
								Command:     command,
								Description: fmt.Sprintf("Auto-synced from %s", source),
								Enabled:     true,
								Created:     time.Now(),
								LastStatus:  0,
								Source:      source,
							}
							saveJobMetadata(job)
						}
					}
				}
			}
		}
	}
}

// syncCrontabToDB syncs all actual crontab entries to metadata database
func syncCrontabToDB() {
	cronContent, err := readCrontab()
	if err != nil {
		return
	}

	lines := strings.Split(cronContent, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse cron line
		parts := strings.Fields(line)
		if len(parts) < 6 {
			continue
		}

		schedule := strings.Join(parts[:5], " ")
		command := strings.Join(parts[5:], " ")

		// Check if this is a webstack cron (has our marker)
		isWebStack := strings.Contains(command, "webstack") ||
			(i > 0 && strings.Contains(lines[i-1], "webstack-"))

		// Determine source
		source := "manual"
		if strings.Contains(command, "webstack-backup-cleanup") {
			source = "backup"
		} else if strings.Contains(command, "webstack") {
			source = "webstack"
		}

		// Check if already in metadata
		var exists bool
		files, err := ioutil.ReadDir(cronMetadataDir)
		if err == nil {
			for _, file := range files {
				if !strings.HasSuffix(file.Name(), ".json") {
					continue
				}
				data, _ := ioutil.ReadFile(filepath.Join(cronMetadataDir, file.Name()))
				var job Job
				if json.Unmarshal(data, &job) == nil {
					if job.Schedule == schedule && job.Command == command {
						exists = true
						break
					}
				}
			}
		}

		// If not exists and is WebStack-related, add it to metadata
		if !exists && isWebStack {
			jobID := getNextJobID()
			job := Job{
				ID:          jobID,
				Schedule:    schedule,
				Command:     command,
				Description: fmt.Sprintf("Auto-discovered from %s", source),
				Enabled:     true,
				Created:     time.Now(),
				LastStatus:  0,
				Source:      source,
			}
			saveJobMetadata(job)
		}
	}
}

// syncSystemdTimersToDB discovers systemd timers and syncs them to metadata
func syncSystemdTimersToDB() {
	// List all systemd timers with webstack prefix
	cmd := exec.Command("systemctl", "list-timers", "webstack*", "--all", "--output=json")
	output, err := cmd.Output()
	if err != nil {
		// systemd timers not available or no timers found
		return
	}

	// Parse JSON output (simplified - just look for timer names)
	// systemctl list-timers --output=json returns structured data
	var timers []map[string]interface{}
	if err := json.Unmarshal(output, &timers); err != nil {
		// Try parsing simple output format
		return
	}

	// Process each timer
	for _, timer := range timers {
		if unitStr, ok := timer["unit"].(string); ok {
			if !strings.HasPrefix(unitStr, "webstack") {
				continue
			}

			// Extract timer name
			timerName := strings.TrimSuffix(unitStr, ".timer")

			// Determine schedule from timer name and get service details
			schedule, command, description := extractSystemdTimerInfo(timerName)

			if schedule == "" || command == "" {
				continue
			}

			// Check if already in metadata (look for exact match or similar timer)
			var exists bool
			var isDuplicate bool
			files, err := ioutil.ReadDir(cronMetadataDir)
			if err == nil {
				for _, file := range files {
					if !strings.HasSuffix(file.Name(), ".json") {
						continue
					}
					data, _ := ioutil.ReadFile(filepath.Join(cronMetadataDir, file.Name()))
					var job Job
					if json.Unmarshal(data, &job) == nil {
						// Check for exact match
						if job.Schedule == schedule && job.Command == command {
							exists = true
							break
						}
						// Check for duplicate - same timer name but messy JSON format
						if job.Schedule == schedule && job.Description == description && strings.Contains(job.Command, "{") {
							// This is an old messy entry for the same timer
							isDuplicate = true
							// Remove the duplicate (old format)
							os.Remove(filepath.Join(cronMetadataDir, file.Name()))
							break
						}
					}
				}
			}

			// If it's a duplicate, continue (we'll add the clean version)
			if isDuplicate {
				continue
			}

			// If not exists, add it to metadata
			if !exists {
				jobID := getNextJobID()
				source := "systemd"

				// Determine source from timer name
				if strings.Contains(timerName, "backup") {
					source = "backup"
				} else if strings.Contains(timerName, "certbot") || strings.Contains(timerName, "ssl") {
					source = "ssl"
				} else if strings.Contains(timerName, "dns") {
					source = "dns"
				}

				job := Job{
					ID:          jobID,
					Schedule:    schedule,
					Command:     command,
					Description: description,
					Enabled:     true,
					Created:     time.Now(),
					LastStatus:  0,
					Source:      source,
				}
				saveJobMetadata(job)
			}
		}
	}
}

// extractSystemdTimerInfo extracts schedule and command info from a systemd timer
func extractSystemdTimerInfo(timerName string) (string, string, string) {
	// Query systemd for timer details
	cmd := exec.Command("systemctl", "show", timerName+".timer", "-p", "OnCalendar", "--value")
	output, err := cmd.Output()
	if err != nil {
		return "", "", ""
	}

	calendarSpec := strings.TrimSpace(string(output))

	// Get the service that this timer activates
	cmd = exec.Command("systemctl", "show", timerName+".timer", "-p", "Activates", "--value")
	output, err = cmd.Output()
	if err != nil {
		return "", "", ""
	}

	activatesService := strings.TrimSpace(string(output))
	if activatesService == "" {
		activatesService = timerName + ".service"
	}

	// Get friendly description
	description := fmt.Sprintf("Systemd timer: %s", strings.TrimSuffix(activatesService, ".service"))

	// Convert OnCalendar format to crontab format (simplified)
	schedule := convertOnCalendarToCron(calendarSpec)

	// Create a simple command representation for systemd timers
	command := fmt.Sprintf("systemctl start %s", activatesService)

	return schedule, command, description
}

// convertOnCalendarToCron converts systemd OnCalendar format to crontab format
func convertOnCalendarToCron(calendarSpec string) string {
	// Examples:
	// "daily" -> "0 0 * * *"
	// "*-*-* 03:15:00" -> "15 3 * * *"
	// "Mon-Sun 03:00" -> "0 3 * * *"

	calendarSpec = strings.TrimSpace(calendarSpec)

	// Handle common patterns
	switch calendarSpec {
	case "daily":
		return "0 0 * * *"
	case "weekly":
		return "0 0 * * 0"
	case "monthly":
		return "0 0 1 * *"
	}

	// Parse time format like "03:15:00"
	if strings.Contains(calendarSpec, ":") {
		parts := strings.Fields(calendarSpec)
		for _, part := range parts {
			if strings.Contains(part, ":") {
				timeParts := strings.Split(part, ":")
				if len(timeParts) >= 2 {
					hour := timeParts[0]
					minute := timeParts[1]
					// Return crontab format: minute hour * * *
					return fmt.Sprintf("%s %s * * *", minute, hour)
				}
			}
		}
	}

	// Default to daily if we can't parse
	return "0 0 * * *"
}

// GetWebStackCrons gets only WebStack-managed crons
func GetWebStackCrons() ([]Job, error) {
	jobs, err := ListJobs(false)
	if err != nil {
		return nil, err
	}

	var webstackJobs []Job
	for _, job := range jobs {
		if job.Source != "manual" || strings.Contains(job.Command, "webstack") {
			webstackJobs = append(webstackJobs, job)
		}
	}

	return webstackJobs, nil
}

// RegisterSystemCron registers a cron job created by a WebStack system component
// Used by backup, ssl, dns, and other components to track their own crons
func RegisterSystemCron(schedule, command, description, source string) error {
	// Validate schedule format
	if !isValidSchedule(schedule) {
		return fmt.Errorf("invalid crontab schedule format: %s", schedule)
	}

	// Check if this cron already exists
	jobs, err := ListJobs(false)
	if err == nil {
		for _, job := range jobs {
			if job.Schedule == schedule && job.Command == command && job.Source == source {
				return nil // Already exists
			}
		}
	}

	// Get next available ID
	jobID := getNextJobID()

	// Create job
	job := Job{
		ID:          jobID,
		Schedule:    schedule,
		Command:     command,
		Description: description,
		Enabled:     true,
		Created:     time.Now(),
		LastStatus:  0,
		Source:      source,
	}

	// Save job metadata
	if err := saveJobMetadata(job); err != nil {
		return err
	}

	// Don't add to crontab - it should already be there
	// This function just registers it in our metadata
	return nil
}
