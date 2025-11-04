package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// createTarGz creates a tar.gz archive from a directory
func createTarGz(sourcePath, targetPath string) error {
	file, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}
		}

		return nil
	})
}

// extractTarGz extracts a tar.gz archive
func extractTarGz(archivePath, targetDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar: %w", err)
		}

		path := filepath.Join(targetDir, header.Name)
		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(path, header.FileInfo().Mode()); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			file.Close()
		}
	}

	return nil
}

// backupDirectory backs up a directory structure
func backupDirectory(sourceDir, destDir, name string) (int64, error) {
	archivePath := filepath.Join(destDir, name+".tar.gz")

	if err := createTarGz(sourceDir, archivePath); err != nil {
		return 0, fmt.Errorf("failed to archive %s: %w", name, err)
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		return 0, err
	}

	return info.Size(), nil
}

// backupFile backs up a single file
func backupFile(sourceFile, destDir string) (int64, error) {
	destFile := filepath.Join(destDir, filepath.Base(sourceFile))

	src, err := os.Open(sourceFile)
	if err != nil {
		return 0, fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(destFile)
	if err != nil {
		return 0, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, src)
	if err != nil {
		return 0, fmt.Errorf("failed to copy file: %w", err)
	}

	return written, nil
}

// backupMetadata backs up all metadata files
func backupMetadata(backupPath string) error {
	metadataDir := filepath.Join(backupPath, "metadata")
	os.MkdirAll(metadataDir, 0755)

	// Backup domains.json
	if _, err := os.Stat(domainsFile); err == nil {
		if _, err := backupFile(domainsFile, metadataDir); err != nil {
			return fmt.Errorf("failed to backup domains.json: %w", err)
		}
	}

	// Backup ssl.json
	if _, err := os.Stat(sslFile); err == nil {
		if _, err := backupFile(sslFile, metadataDir); err != nil {
			return fmt.Errorf("failed to backup ssl.json: %w", err)
		}
	}

	// Backup mail.json if exists
	mailFile := "/etc/webstack/mail.json"
	if _, err := os.Stat(mailFile); err == nil {
		if _, err := backupFile(mailFile, metadataDir); err != nil {
			return fmt.Errorf("failed to backup mail.json: %w", err)
		}
	}

	return nil
}

// restoreMetadata restores metadata files
func restoreMetadata(backupPath string) error {
	metadataDir := filepath.Join(backupPath, "metadata")

	files := []string{"domains.json", "ssl.json", "mail.json"}
	for _, file := range files {
		srcFile := filepath.Join(metadataDir, file)
		if _, err := os.Stat(srcFile); err == nil {
			dstFile := filepath.Join("/etc/webstack", file)
			os.MkdirAll("/etc/webstack", 0755)

			if err := copyFile(srcFile, dstFile); err != nil {
				return fmt.Errorf("failed to restore %s: %w", file, err)
			}
		}
	}

	return nil
}

// backupFull performs a full system backup
func backupFull(backupPath string, opts BackupOptions) (int64, int64, error) {
	totalSize := int64(0)
	compressedSize := int64(0)

	// Backup metadata
	if err := backupMetadata(backupPath); err != nil {
		return 0, 0, err
	}

	// Backup all domains
	domainsBackupDir := filepath.Join(backupPath, "domains")
	os.MkdirAll(domainsBackupDir, 0755)

	domains, err := getDomainsList()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get domains list: %w", err)
	}

	for _, domain := range domains {
		domainPath := filepath.Join("/var/www", domain)
		domainBackupPath := filepath.Join(domainsBackupDir, domain)
		os.MkdirAll(domainBackupPath, 0755)

		size, err := backupDirectory(domainPath, domainBackupPath, "htdocs")
		if err != nil {
			fmt.Printf("⚠️  Warning: Could not backup domain %s: %v\n", domain, err)
			continue
		}
		totalSize += size
	}

	// Backup databases
	databasesBackupDir := filepath.Join(backupPath, "databases")
	os.MkdirAll(databasesBackupDir, 0755)

	mysqlSize, _ := backupMySQLDatabases(databasesBackupDir)
	postgresSize, _ := backupPostgreSQLDatabases(databasesBackupDir)
	totalSize += mysqlSize + postgresSize

	// Backup web server configs
	configsBackupDir := filepath.Join(backupPath, "configs")
	os.MkdirAll(configsBackupDir, 0755)

	backupDirectory("/etc/nginx", configsBackupDir, "nginx")
	backupDirectory("/etc/apache2", configsBackupDir, "apache2")

	// Backup SSL certificates
	sslBackupDir := filepath.Join(backupPath, "ssl")
	os.MkdirAll(sslBackupDir, 0755)

	backupDirectory("/etc/ssl/webstack", sslBackupDir, "selfsigned")
	backupDirectory("/etc/letsencrypt", sslBackupDir, "letsencrypt")

	// Backup firewall rules
	fwBackupDir := filepath.Join(backupPath, "firewall")
	os.MkdirAll(fwBackupDir, 0755)

	backupFile("/etc/iptables/rules.v4", fwBackupDir)
	backupFile("/etc/iptables/rules.v6", fwBackupDir)

	// Calculate compressed size
	filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			compressedSize += info.Size()
		}
		return nil
	})

	// Compress if requested
	if opts.Compression != "none" {
		compressedSize = compressBackup(backupPath, opts.Compression)
	}

	return totalSize, compressedSize, nil
}

// backupDomain backs up a single domain
func backupDomain(backupPath string, opts BackupOptions) (int64, int64, error) {
	domain := opts.Scope
	totalSize := int64(0)

	// Create domain backup directory
	domainBackupDir := filepath.Join(backupPath, "domains", domain)
	os.MkdirAll(domainBackupDir, 0755)

	// Backup domain files
	domainPath := filepath.Join("/var/www", domain)
	size, err := backupDirectory(domainPath, domainBackupDir, "files")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to backup domain files: %w", err)
	}
	totalSize += size

	// Backup web server configs
	configDir := filepath.Join(backupPath, "configs")
	os.MkdirAll(configDir, 0755)

	nginxConfig := filepath.Join("/etc/nginx/sites-available", domain+".conf")
	if _, err := os.Stat(nginxConfig); err == nil {
		size, _ := backupFile(nginxConfig, configDir)
		totalSize += size
	}

	apacheConfig := filepath.Join("/etc/apache2/sites-available", domain+".conf")
	if _, err := os.Stat(apacheConfig); err == nil {
		size, _ := backupFile(apacheConfig, configDir)
		totalSize += size
	}

	// Backup SSL certificates if enabled
	sslDir := filepath.Join(backupPath, "ssl")
	os.MkdirAll(sslDir, 0755)

	letsencryptPath := filepath.Join("/etc/letsencrypt/live", domain)
	if _, err := os.Stat(letsencryptPath); err == nil {
		size, _ := backupDirectory(letsencryptPath, sslDir, domain)
		totalSize += size
	}

	selfsignedCert := filepath.Join("/etc/ssl/webstack", domain+".crt")
	if _, err := os.Stat(selfsignedCert); err == nil {
		backupFile(selfsignedCert, sslDir)
		backupFile(filepath.Join("/etc/ssl/webstack", domain+".key"), sslDir)
	}

	// Backup metadata
	if err := backupMetadata(backupPath); err != nil {
		fmt.Printf("⚠️  Warning: Could not backup metadata: %v\n", err)
	}

	// Calculate compressed size
	var compressedSize int64
	filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			compressedSize += info.Size()
		}
		return nil
	})

	// Compress if requested
	if opts.Compression != "none" {
		compressedSize = compressBackup(backupPath, opts.Compression)
	}

	return totalSize, compressedSize, nil
}

// backupDatabase backs up a single database
func backupDatabase(backupPath string, opts BackupOptions) (int64, int64, error) {
	scope := opts.Scope
	totalSize := int64(0)

	// Parse database type and name
	parts := filepath.SplitList(scope)
	if len(parts) < 2 {
		parts = []string{"mysql", scope}
	}

	dbType := parts[0]
	dbName := parts[1]

	databasesDir := filepath.Join(backupPath, "databases", dbType)
	os.MkdirAll(databasesDir, 0755)

	switch dbType {
	case "mysql":
		size, err := dumpMySQLDatabase(dbName, databasesDir)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to backup MySQL database: %w", err)
		}
		totalSize = size
	case "postgresql":
		size, err := dumpPostgreSQLDatabase(dbName, databasesDir)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to backup PostgreSQL database: %w", err)
		}
		totalSize = size
	default:
		return 0, 0, fmt.Errorf("unknown database type: %s", dbType)
	}

	// Backup metadata
	if err := backupMetadata(backupPath); err != nil {
		fmt.Printf("⚠️  Warning: Could not backup metadata: %v\n", err)
	}

	// Calculate compressed size
	var compressedSize int64
	filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			compressedSize += info.Size()
		}
		return nil
	})

	// Compress if requested
	if opts.Compression != "none" {
		compressedSize = compressBackup(backupPath, opts.Compression)
	}

	return totalSize, compressedSize, nil
}

// restoreDomains restores domain backups
func restoreDomains(backupPath, domain string) (int, error) {
	domainsDir := filepath.Join(backupPath, "domains")
	if _, err := os.Stat(domainsDir); os.IsNotExist(err) {
		return 0, fmt.Errorf("no domains backup found")
	}

	entries, err := os.ReadDir(domainsDir)
	if err != nil {
		return 0, err
	}

	restored := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		domainName := entry.Name()
		if domain != "" && domainName != domain {
			continue
		}

		// Look for htdocs.tar.gz (from full backup) or files.tar.gz (from domain-specific backup)
		sourcePath := filepath.Join(domainsDir, domainName, "htdocs.tar.gz")
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			sourcePath = filepath.Join(domainsDir, domainName, "files.tar.gz")
		}

		destPath := filepath.Join("/var/www", domainName)
		os.MkdirAll(destPath, 0755)

		if err := extractTarGz(sourcePath, destPath); err != nil {
			fmt.Printf("⚠️  Could not restore domain %s: %v\n", domainName, err)
			continue
		}

		// Restore web server configs
		configsDir := filepath.Join(backupPath, "configs")
		nginxConfig := filepath.Join(configsDir, domainName+".conf")
		if _, err := os.Stat(nginxConfig); err == nil {
			destConfig := filepath.Join("/etc/nginx/sites-available", domainName+".conf")
			copyFile(nginxConfig, destConfig)
			os.Symlink(destConfig, filepath.Join("/etc/nginx/sites-enabled", domainName+".conf"))
		}

		restored++
	}

	return restored, nil
}

// restoreDatabases restores database backups
func restoreDatabases(backupPath string) (int, error) {
	databasesDir := filepath.Join(backupPath, "databases")
	if _, err := os.Stat(databasesDir); os.IsNotExist(err) {
		return 0, nil
	}

	restored := 0
	entries, err := os.ReadDir(databasesDir)
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dbType := entry.Name()
		dbDir := filepath.Join(databasesDir, dbType)

		dbFiles, err := os.ReadDir(dbDir)
		if err != nil {
			continue
		}

		for _, dbFile := range dbFiles {
			if dbFile.IsDir() {
				continue
			}

			dbName := filepath.Base(dbFile.Name())
			dbName = dbName[:len(dbName)-len(filepath.Ext(dbName))] // Remove .sql extension

			sqlPath := filepath.Join(dbDir, dbFile.Name())

			switch dbType {
			case "mysql":
				if err := restoreMySQLDatabase(dbName, sqlPath); err != nil {
					fmt.Printf("⚠️  Could not restore MySQL database %s: %v\n", dbName, err)
					continue
				}
			case "postgresql":
				if err := restorePostgreSQLDatabase(dbName, sqlPath); err != nil {
					fmt.Printf("⚠️  Could not restore PostgreSQL database %s: %v\n", dbName, err)
					continue
				}
			}

			restored++
		}
	}

	return restored, nil
}

// compressBackup compresses backup directory
func compressBackup(backupPath, compression string) int64 {
	// For now, just count files
	var totalSize int64
	filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	return totalSize
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	os.MkdirAll(filepath.Dir(dst), 0755)
	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
