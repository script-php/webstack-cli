package cmd

import (
	"webstack-cli/internal/domain"

	"github.com/spf13/cobra"
)

var domainCmd = &cobra.Command{
	Use:   "domain",
	Short: "Manage domains",
	Long:  `Add, edit, and delete domains with backend and PHP version selection.`,
}

var domainAddCmd = &cobra.Command{
	Use:   "add [domain]",
	Short: "Add a new domain",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		backend, _ := cmd.Flags().GetString("backend")
		phpVersion, _ := cmd.Flags().GetString("php")
		domain.Add(args[0], backend, phpVersion)
	},
}

var domainEditCmd = &cobra.Command{
	Use:   "edit [domain]",
	Short: "Edit an existing domain",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		backend, _ := cmd.Flags().GetString("backend")
		phpVersion, _ := cmd.Flags().GetString("php")
		domain.Edit(args[0], backend, phpVersion)
	},
}

var domainDeleteCmd = &cobra.Command{
	Use:   "delete [domain]",
	Short: "Delete a domain",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		domain.Delete(args[0])
	},
}

var domainListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all domains",
	Run: func(cmd *cobra.Command, args []string) {
		domain.List()
	},
}

var domainRebuildCmd = &cobra.Command{
	Use:   "rebuild-configs",
	Short: "Rebuild configuration files for all domains",
	Long:  `Regenerate Nginx and Apache configuration files for all domains from templates. Useful after updating templates or fixing configuration issues.`,
	Run: func(cmd *cobra.Command, args []string) {
		domain.RebuildConfigs()
	},
}

func init() {
	rootCmd.AddCommand(domainCmd)
	domainCmd.AddCommand(domainAddCmd)
	domainCmd.AddCommand(domainEditCmd)
	domainCmd.AddCommand(domainDeleteCmd)
	domainCmd.AddCommand(domainListCmd)
	domainCmd.AddCommand(domainRebuildCmd)

	// Flags for domain add/edit
	domainAddCmd.Flags().StringP("backend", "b", "", "Backend type: nginx or apache (default: nginx)")
	domainAddCmd.Flags().StringP("php", "p", "", "PHP version (5.6-8.4)")

	domainEditCmd.Flags().StringP("backend", "b", "", "Backend type: nginx or apache")
	domainEditCmd.Flags().StringP("php", "p", "", "PHP version (5.6-8.4)")
}
