package main

import (
	"fmt"
	"os"

	"webstack-cli/cmd"
)

func main() {
	// Check if running as root on Linux
	if os.Geteuid() != 0 {
		fmt.Println("This tool requires root privileges. Please run with sudo.")
		os.Exit(1)
	}

	cmd.Execute()
}
