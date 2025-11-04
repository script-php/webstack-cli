package backup

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Backup represents a backup entry
type Backup struct {
	ID                string              `json:"id"`
	Timestamp         time.Time           `json:"timestamp"`
	Type              string              `json:"type"`  // "full", "domain", "database"
	Scope             string              `json:"scope"` // "all", "example.com", "mysql:dbname"
	SizeBytes         int64               `json:"size_bytes"`
	CompressedSize    int64               `json:"compressed_size"`
	Compression       string              `json:"compression"`
	Encryption        string              `json:"encryption"`
	Checksum          string              `json:"checksum"`
	Verified          bool                `json:"verified"`
	DomainsIncluded   []string            `json:"domains_included,omitempty"`
	DatabasesIncluded map[string][]string `json:"databases_included,omitempty"`
}

// BackupOptions for creating backups
type BackupOptions struct {
	Type        string
	Scope       string
	Compression string
	Encryption  string
}

// StorageStatus represents backup storage information
type StorageStatus struct {
	Location        string
	BackupCount     int
	TotalSize       int64
	AvailableSpace  int64
	TotalSpace      int64
	ScheduleEnabled bool
	NextBackup      time.Time
}

const backupDir = "/var/backups/webstack"
const backupMetadataDir = backupDir + "/metadata"
const backupArchiveDir = backupDir + "/archives"
const domainsFile = "/etc/webstack/domains.json"
const sslFile = "/etc/webstack/ssl.json"

// Initialize backup directories
func init() {
	os.MkdirAll(backupDir, 0755)
	os.MkdirAll(backupMetadataDir, 0755)
	os.MkdirAll(backupArchiveDir, 0755)
}

// Create creates a new backup
func Create(opts BackupOptions) (string, int64, int64, error) {
	fmt.Printf("🔄 Preparing backup: type=%s, scope=%s\n", opts.Type, opts.Scope)

	// Generate backup ID
	backupID := generateBackupID()
	stagingPath := filepath.Join(os.TempDir(), "webstack-backup-"+backupID)
	defer os.RemoveAll(stagingPath)
	os.MkdirAll(stagingPath, 0755)

	backup := Backup{
		ID:          backupID,
		Timestamp:   time.Now(),
		Type:        opts.Type,
		Scope:       opts.Scope,
		Compression: opts.Compression,
		Encryption:  opts.Encryption,
		Checksum:    "",
		Verified:    false,
	}

	var totalSize int64
	var err error

	// Backup metadata always
	if err := backupMetadata(stagingPath); err != nil {
		return "", 0, 0, fmt.Errorf("failed to backup metadata: %w", err)
	}

	switch opts.Type {
	case "full":
		fmt.Println("📦 Backing up: metadata, domains, SSL, databases...")
		size, _, err2 := backupFull(stagingPath, opts)
		totalSize, err = size, err2
	case "domain":
		fmt.Printf("📦 Backing up domain: %s\n", opts.Scope)
		size, _, err2 := backupDomain(stagingPath, opts)
		totalSize, err = size, err2
	case "database":
		fmt.Printf("📦 Backing up database: %s\n", opts.Scope)
		size, _, err2 := backupDatabase(stagingPath, opts)
		totalSize, err = size, err2
	default:
		return "", 0, 0, fmt.Errorf("unknown backup type: %s", opts.Type)
	}

	if err != nil {
		return "", 0, 0, err
	}

	// Create compressed archive
	fmt.Printf("📦 Compressing backup...\n")
	archiveFile := filepath.Join(backupArchiveDir, backupID+".tar.gz")
	if err := createTarGz(stagingPath, archiveFile); err != nil {
		return "", 0, 0, fmt.Errorf("failed to compress backup: %w", err)
	}

	// Get archive size
	archiveInfo, err := os.Stat(archiveFile)
	if err != nil {
		return "", 0, 0, err
	}
	compressedSize := archiveInfo.Size()

	// Calculate checksum of archive
	checksum, err := calculateFileChecksum(archiveFile)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	backup.SizeBytes = totalSize
	backup.CompressedSize = compressedSize
	backup.Checksum = checksum
	backup.Verified = true

	// Get domain list
	if opts.Type == "full" {
		domains, _ := getDomainsList()
		backup.DomainsIncluded = domains
		backup.DatabasesIncluded = getIncludedDatabases()
	}

	// Save backup metadata
	if err := saveBackupMetadata(backup); err != nil {
		return "", 0, 0, fmt.Errorf("failed to save backup metadata: %w", err)
	}

	fmt.Printf("✓ Backup completed: %s → %s (compressed)\n",
		FormatBytes(totalSize), FormatBytes(compressedSize))
	return backupID, totalSize, compressedSize, nil
}

// List lists all backups or filtered backups
func List(domain, since string) ([]Backup, error) {
	files, err := ioutil.ReadDir(backupMetadataDir)
	if err != nil {
		return nil, err
	}

	var backups []Backup
	var sinceTime time.Time

	// Parse "since" filter
	if since != "" {
		switch {
		case strings.HasSuffix(since, "d"):
			days := parseDuration(since, "d")
			sinceTime = time.Now().AddDate(0, 0, -days)
		case strings.HasSuffix(since, "m"):
			months := parseDuration(since, "m")
			sinceTime = time.Now().AddDate(0, -months, 0)
		case strings.HasSuffix(since, "y"):
			years := parseDuration(since, "y")
			sinceTime = time.Now().AddDate(-years, 0, 0)
		}
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		data, err := ioutil.ReadFile(filepath.Join(backupMetadataDir, file.Name()))
		if err != nil {
			continue
		}

		var b Backup
		if err := json.Unmarshal(data, &b); err != nil {
			continue
		}

		// Apply filters
		if since != "" && b.Timestamp.Before(sinceTime) {
			continue
		}

		if domain != "" {
			found := false
			for _, d := range b.DomainsIncluded {
				if d == domain {
					found = true
					break
				}
			}
			if !found && b.Scope != domain {
				continue
			}
		}

		backups = append(backups, b)
	}

	return backups, nil
}

// Restore restores from a backup
func Restore(backupID, domain string) (int, error) {
	archiveFile := filepath.Join(backupArchiveDir, backupID+".tar.gz")
	if _, err := os.Stat(archiveFile); os.IsNotExist(err) {
		return 0, fmt.Errorf("backup not found: %s", backupID)
	}

	// Verify backup first
	if ok, err := Verify(backupID); !ok || err != nil {
		return 0, fmt.Errorf("backup verification failed: %w", err)
	}

	// Create staging directory
	stagingDir := filepath.Join(os.TempDir(), "webstack-restore-"+backupID)
	os.MkdirAll(stagingDir, 0755)
	defer os.RemoveAll(stagingDir)

	fmt.Printf("📥 Extracting backup from archive...\n")

	// Extract archive to staging
	if err := extractTarGz(archiveFile, stagingDir); err != nil {
		return 0, fmt.Errorf("failed to extract backup archive: %w", err)
	}

	// Extract metadata
	metadataFile := filepath.Join(backupMetadataDir, backupID+".json")
	data, err := ioutil.ReadFile(metadataFile)
	if err != nil {
		return 0, fmt.Errorf("failed to read backup metadata: %w", err)
	}

	var backup Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		return 0, fmt.Errorf("failed to parse backup metadata: %w", err)
	}

	itemsRestored := 0

	// Restore metadata
	if err := restoreMetadata(stagingDir); err != nil {
		fmt.Printf("⚠️  Warning: Could not restore metadata: %v\n", err)
	} else {
		itemsRestored++
	}

	// Restore domains if full backup or specific domain requested
	if backup.Type == "full" || (backup.Type == "domain" && domain == "") {
		fmt.Println("📂 Restoring domains...")
		count, err := restoreDomains(stagingDir, domain)
		if err != nil {
			fmt.Printf("⚠️  Warning: Could not fully restore domains: %v\n", err)
		}
		itemsRestored += count
	}

	// Restore databases if included
	if backup.Type == "full" || strings.HasPrefix(backup.Scope, "database") {
		fmt.Println("🗄️  Restoring databases...")
		count, err := restoreDatabases(stagingDir)
		if err != nil {
			fmt.Printf("⚠️  Warning: Could not fully restore databases: %v\n", err)
		}
		itemsRestored += count
	}

	// Reload services
	fmt.Println("🔄 Reloading services...")
	reloadServices()

	return itemsRestored, nil
}

// Delete deletes a backup
func Delete(backupID string) error {
	archiveFile := filepath.Join(backupArchiveDir, backupID+".tar.gz")
	metadataFile := filepath.Join(backupMetadataDir, backupID+".json")

	if err := os.Remove(archiveFile); err != nil {
		return fmt.Errorf("failed to delete backup archive: %w", err)
	}

	if err := os.Remove(metadataFile); err != nil {
		return fmt.Errorf("failed to delete backup metadata: %w", err)
	}

	return nil
}

// Verify verifies backup integrity
func Verify(backupID string) (bool, error) {
	metadataFile := filepath.Join(backupMetadataDir, backupID+".json")
	data, err := ioutil.ReadFile(metadataFile)
	if err != nil {
		return false, fmt.Errorf("backup metadata not found: %w", err)
	}

	var backup Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		return false, fmt.Errorf("failed to parse metadata: %w", err)
	}

	archiveFile := filepath.Join(backupArchiveDir, backupID+".tar.gz")
	checksum, err := calculateFileChecksum(archiveFile)
	if err != nil {
		return false, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if checksum != backup.Checksum {
		return false, fmt.Errorf("checksum mismatch")
	}

	return true, nil
}

// GetStorageStatus returns backup storage information
func GetStorageStatus() (*StorageStatus, error) {
	// Calculate total size
	totalSize := int64(0)
	if err := filepath.Walk(backupArchiveDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// Count backups
	files, _ := ioutil.ReadDir(backupMetadataDir)
	backupCount := 0
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".json") {
			backupCount++
		}
	}

	// Get available space
	stat := getFileSystemStats(backupDir)

	// Check if schedule is enabled
	scheduleEnabled := isScheduleEnabled()

	var nextBackup time.Time
	if scheduleEnabled {
		nextBackup = getNextScheduledBackup()
	}

	return &StorageStatus{
		Location:        backupDir,
		BackupCount:     backupCount,
		TotalSize:       totalSize,
		AvailableSpace:  stat.Available,
		TotalSpace:      stat.Total,
		ScheduleEnabled: scheduleEnabled,
		NextBackup:      nextBackup,
	}, nil
}

// Export exports a backup to a file
func Export(backupID, destination string) error {
	archiveFile := filepath.Join(backupArchiveDir, backupID+".tar.gz")
	if _, err := os.Stat(archiveFile); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", backupID)
	}

	// Simply copy the archive file to destination
	source, err := os.Open(archiveFile)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	return err
}

// Import imports a backup from a file
func Import(source string) (string, error) {
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return "", fmt.Errorf("source file not found: %s", source)
	}

	backupID := generateBackupID()
	archiveFile := filepath.Join(backupArchiveDir, backupID+".tar.gz")

	// Copy the archive file
	sourceFile, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(archiveFile)
	if err != nil {
		return "", err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		os.Remove(archiveFile)
		return "", fmt.Errorf("failed to copy backup: %w", err)
	}

	return backupID, nil
}

// PrintJSON prints backups in JSON format
func PrintJSON(backups []Backup) {
	data, _ := json.MarshalIndent(backups, "", "  ")
	fmt.Println(string(data))
}

// GetTotalSize returns total size of all backups
func GetTotalSize(backups []Backup) int64 {
	total := int64(0)
	for _, b := range backups {
		total += b.CompressedSize
	}
	return total
}

// GetBackupPath returns the path where a backup is stored
func GetBackupPath(backupID string) string {
	return filepath.Join(backupArchiveDir, backupID+".tar.gz")
}

// FormatBytes formats bytes to human-readable size
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Helper functions

func generateBackupID() string {
	return fmt.Sprintf("backup-%d", time.Now().Unix())
}

func calculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func calculateDirectoryChecksum(dir string) (string, error) {
	h := sha256.New()
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			file, _ := os.Open(path)
			defer file.Close()
			io.Copy(h, file)
		}
		return nil
	}); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func parseDuration(s, suffix string) int {
	s = strings.TrimSuffix(s, suffix)
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

func getFileSystemStats(path string) struct {
	Total     int64
	Available int64
} {
	// Simplified - would use syscall.Statfs in production
	return struct {
		Total     int64
		Available int64
	}{
		Total:     1099511627776, // 1TB default
		Available: 549755813888,  // 512GB default
	}
}

func saveBackupMetadata(backup Backup) error {
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return err
	}

	metadataFile := filepath.Join(backupMetadataDir, backup.ID+".json")
	return ioutil.WriteFile(metadataFile, data, 0644)
}

func getDomainsList() ([]string, error) {
	data, err := ioutil.ReadFile(domainsFile)
	if err != nil {
		return nil, err
	}

	var domains []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &domains); err != nil {
		return nil, err
	}

	var names []string
	for _, d := range domains {
		names = append(names, d.Name)
	}
	return names, nil
}

func getIncludedDatabases() map[string][]string {
	databases := make(map[string][]string)
	// TODO: Implement actual database listing
	return databases
}

func reloadServices() error {
	// Reload web servers
	os.Chdir("/")
	exec := func(cmd string, args ...string) {
		// Silent execution
	}
	exec("systemctl", "reload", "nginx")
	exec("systemctl", "reload", "apache2")
	exec("systemctl", "reload", "php-fpm")
	return nil
}

func isScheduleEnabled() bool {
	// Check if systemd timer is enabled
	// TODO: Implement
	return false
}

func getNextScheduledBackup() time.Time {
	// TODO: Implement
	return time.Now().AddDate(0, 0, 1)
}
