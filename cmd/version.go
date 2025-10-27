package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run:   showVersion,
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update WebStack CLI to the latest version",
	Run:   updateCLI,
}

func showVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("WebStack CLI %s\n", Version)
	fmt.Printf("Build Time: %s\n", BuildTime)
	fmt.Printf("Git Commit: %s\n", GitCommit)
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

func updateCLI(cmd *cobra.Command, args []string) {
	fmt.Println("🔍 Checking for updates...")

	// Get latest release from GitHub
	resp, err := http.Get("https://api.github.com/repos/yourusername/webstack-cli/releases/latest")
	if err != nil {
		fmt.Printf("❌ Failed to check for updates: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("❌ Failed to fetch release information (HTTP %d)\n", resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Failed to read response: %v\n", err)
		return
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		fmt.Printf("❌ Failed to parse release information: %v\n", err)
		return
	}

	if release.TagName == Version {
		fmt.Printf("✅ You're already running the latest version (%s)\n", Version)
		return
	}

	fmt.Printf("🆕 New version available: %s (current: %s)\n", release.TagName, Version)
	fmt.Printf("📝 Release notes: %s\n", release.Name)

	if !askConfirmation("Do you want to update now?") {
		fmt.Println("Update cancelled.")
		return
	}

	// Download and install update
	if err := downloadAndInstall(release.TagName); err != nil {
		fmt.Printf("❌ Update failed: %v\n", err)
		return
	}

	fmt.Printf("✅ Successfully updated to version %s\n", release.TagName)
	fmt.Println("Please restart your terminal or run 'webstack version' to verify the update.")
}

func downloadAndInstall(version string) error {
	platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		platform += ".exe"
	}

	downloadURL := fmt.Sprintf("https://github.com/yourusername/webstack-cli/releases/download/%s/webstack-%s", version, platform)

	fmt.Printf("📥 Downloading %s...\n", downloadURL)

	// Download the new binary
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed with HTTP %d", resp.StatusCode)
	}

	// Read the binary data
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read downloaded data: %v", err)
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %v", err)
	}

	// Create backup
	backupPath := execPath + ".backup"
	if err := copyFile(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %v", err)
	}

	// Write new binary
	tmpPath := execPath + ".new"
	if err := ioutil.WriteFile(tmpPath, data, 0755); err != nil {
		return fmt.Errorf("failed to write new binary: %v", err)
	}

	// Replace current binary
	if err := os.Rename(tmpPath, execPath); err != nil {
		// Restore backup on failure
		os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to replace binary: %v", err)
	}

	// Remove backup
	os.Remove(backupPath)

	return nil
}

func copyFile(src, dst string) error {
	data, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dst, data, 0755)
}

func askConfirmation(question string) bool {
	fmt.Printf("%s (y/N): ", question)

	var response string
	fmt.Scanln(&response)

	return response == "y" || response == "Y" || response == "yes" || response == "Yes"
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateCmd)
}
