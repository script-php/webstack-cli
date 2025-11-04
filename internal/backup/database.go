package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// dumpMySQLDatabase creates a SQL dump of a MySQL database
func dumpMySQLDatabase(dbName, outputDir string) (int64, error) {
	outputFile := filepath.Join(outputDir, dbName+".sql")

	cmd := exec.Command("mysqldump", "-u", "root", "--all-databases")
	if dbName != "all" {
		cmd = exec.Command("mysqldump", "-u", "root", dbName)
	}

	output, err := os.Create(outputFile)
	if err != nil {
		return 0, fmt.Errorf("failed to create output file: %w", err)
	}
	defer output.Close()

	cmd.Stdout = output
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("mysqldump failed: %w", err)
	}

	info, err := os.Stat(outputFile)
	if err != nil {
		return 0, err
	}

	return info.Size(), nil
}

// dumpPostgreSQLDatabase creates a SQL dump of a PostgreSQL database
func dumpPostgreSQLDatabase(dbName, outputDir string) (int64, error) {
	outputFile := filepath.Join(outputDir, dbName+".sql")

	cmd := exec.Command("sudo", "-u", "postgres", "pg_dump", dbName)

	output, err := os.Create(outputFile)
	if err != nil {
		return 0, fmt.Errorf("failed to create output file: %w", err)
	}
	defer output.Close()

	cmd.Stdout = output
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("pg_dump failed: %w", err)
	}

	info, err := os.Stat(outputFile)
	if err != nil {
		return 0, err
	}

	return info.Size(), nil
}

// restoreMySQLDatabase restores a MySQL database from SQL dump
func restoreMySQLDatabase(dbName, sqlFile string) error {
	// Create database if not exists
	createCmd := exec.Command("mysql", "-u", "root", "-e", fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`;", dbName))
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Read SQL file and execute
	sqlData, err := os.ReadFile(sqlFile)
	if err != nil {
		return fmt.Errorf("failed to read SQL file: %w", err)
	}

	restoreCmd := exec.Command("mysql", "-u", "root", dbName)
	restoreCmd.Stdin = strings.NewReader(string(sqlData))

	if err := restoreCmd.Run(); err != nil {
		return fmt.Errorf("failed to restore database: %w", err)
	}

	return nil
}

// restorePostgreSQLDatabase restores a PostgreSQL database from SQL dump
func restorePostgreSQLDatabase(dbName, sqlFile string) error {
	// Create database if not exists
	createCmd := exec.Command("sudo", "-u", "postgres", "createdb", "-i", dbName)
	createCmd.Run() // Ignore error if database exists

	// Read SQL file and execute
	sqlData, err := os.ReadFile(sqlFile)
	if err != nil {
		return fmt.Errorf("failed to read SQL file: %w", err)
	}

	restoreCmd := exec.Command("sudo", "-u", "postgres", "psql", dbName)
	restoreCmd.Stdin = strings.NewReader(string(sqlData))

	if err := restoreCmd.Run(); err != nil {
		return fmt.Errorf("failed to restore database: %w", err)
	}

	return nil
}

// backupMySQLDatabases backs up all MySQL databases
func backupMySQLDatabases(outputDir string) (int64, error) {
	// Create MySQL subdirectory
	mysqlDir := filepath.Join(outputDir, "mysql")
	os.MkdirAll(mysqlDir, 0755)

	// Get list of databases
	listCmd := exec.Command("mysql", "-u", "root", "-se", "SHOW DATABASES;")
	output, err := listCmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to list databases: %w", err)
	}

	var totalSize int64
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		dbName := strings.TrimSpace(line)
		if dbName == "" || strings.HasPrefix(dbName, "information_schema") || strings.HasPrefix(dbName, "mysql") {
			continue
		}

		size, err := dumpMySQLDatabase(dbName, mysqlDir)
		if err != nil {
			fmt.Printf("⚠️  Could not backup MySQL database %s: %v\n", dbName, err)
			continue
		}
		totalSize += size
	}

	return totalSize, nil
}

// backupPostgreSQLDatabases backs up all PostgreSQL databases
func backupPostgreSQLDatabases(outputDir string) (int64, error) {
	// Create PostgreSQL subdirectory
	postgresDir := filepath.Join(outputDir, "postgresql")
	os.MkdirAll(postgresDir, 0755)

	// Get list of databases
	listCmd := exec.Command("sudo", "-u", "postgres", "psql", "-lqt")
	output, err := listCmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to list databases: %w", err)
	}

	var totalSize int64
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) > 0 {
			dbName := strings.TrimSpace(parts[0])
			if dbName == "" || strings.HasPrefix(dbName, "template") || strings.HasPrefix(dbName, "postgres") {
				continue
			}

			size, err := dumpPostgreSQLDatabase(dbName, postgresDir)
			if err != nil {
				fmt.Printf("⚠️  Could not backup PostgreSQL database %s: %v\n", dbName, err)
				continue
			}
			totalSize += size
		}
	}

	return totalSize, nil
}

// listMySQLDatabases returns list of MySQL databases
func listMySQLDatabases() ([]string, error) {
	listCmd := exec.Command("mysql", "-u", "root", "-se", "SHOW DATABASES;")
	output, err := listCmd.Output()
	if err != nil {
		return nil, err
	}

	var databases []string
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		dbName := strings.TrimSpace(line)
		if dbName != "" {
			databases = append(databases, dbName)
		}
	}

	return databases, nil
}

// listPostgreSQLDatabases returns list of PostgreSQL databases
func listPostgreSQLDatabases() ([]string, error) {
	listCmd := exec.Command("sudo", "-u", "postgres", "psql", "-lqt")
	output, err := listCmd.Output()
	if err != nil {
		return nil, err
	}

	var databases []string
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) > 0 {
			dbName := strings.TrimSpace(parts[0])
			if dbName != "" {
				databases = append(databases, dbName)
			}
		}
	}

	return databases, nil
}
