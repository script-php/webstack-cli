package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
    "webstack-cli/internal/installer"
)

var menuCmd = &cobra.Command{
    Use:   "menu",
    Short: "Show component status menu",
    Long:  `Display components and their state (installed / running). Colors indicate running state (green) or stopped (red).`,
    Run: func(cmd *cobra.Command, args []string) {
        statuses := installer.GetComponentsStatus()
        phpVersions := installer.GetPHPVersionsStatus()

        fmt.Printf("%-12s %-12s %-8s\n", "Component", "Installed", "Running")
        fmt.Println("---------------------------------------------")

        green := "\033[32m"
        red := "\033[31m"
        reset := "\033[0m"

        // Display main components
        for name, s := range statuses {
            inst := "no"
            if s.DpkgInstalled {
                inst = "yes"
            }
            running := fmt.Sprintf("%s%s%s", red, "stopped", reset)
            if s.ServiceRunning {
                running = fmt.Sprintf("%s%s%s", green, "running", reset)
            }

            fmt.Printf("%-12s %-12s %-8s\n", name, inst, running)
        }

        // Display PHP versions if any are installed
        hasPhp := false
        for _, s := range phpVersions {
            if s.DpkgInstalled {
                hasPhp = true
                break
            }
        }

        if hasPhp {
            fmt.Println("")
            fmt.Printf("%-12s %-12s %-8s\n", "PHP Versions", "Installed", "Running")
            fmt.Println("---------------------------------------------")
            for name, s := range phpVersions {
                inst := "no"
                if s.DpkgInstalled {
                    inst = "yes"
                }
                running := fmt.Sprintf("%s%s%s", red, "stopped", reset)
                if s.ServiceRunning {
                    running = fmt.Sprintf("%s%s%s", green, "running", reset)
                }

                fmt.Printf("%-12s %-12s %-8s\n", name, inst, running)
            }
        }
    },
}

func init() {
    rootCmd.AddCommand(menuCmd)
}
