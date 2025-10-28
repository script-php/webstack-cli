package cmd

import (
	"webstack-cli/internal/ssl"

	"github.com/spf13/cobra"
)

var sslCmd = &cobra.Command{
	Use:   "ssl",
	Short: "Manage SSL certificates",
	Long:  `Enable, disable, and renew SSL certificates using Let's Encrypt.`,
}

var sslEnableCmd = &cobra.Command{
	Use:   "enable [domain]",
	Short: "Enable SSL certificate for a domain",
	Long:  `Enable SSL certificate for a domain. Use --type to specify certificate type: selfsigned or letsencrypt.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		email, _ := cmd.Flags().GetString("email")
		certType, _ := cmd.Flags().GetString("type")
		ssl.EnableWithType(args[0], email, certType)
	},
}

var sslDisableCmd = &cobra.Command{
	Use:   "disable [domain]",
	Short: "Disable SSL certificate for a domain",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ssl.Disable(args[0])
	},
}

var sslRenewCmd = &cobra.Command{
	Use:   "renew [domain]",
	Short: "Renew SSL certificate for a domain",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			ssl.RenewAll()
		} else {
			ssl.Renew(args[0])
		}
	},
}

var sslStatusCmd = &cobra.Command{
	Use:   "status [domain]",
	Short: "Check SSL certificate status",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			ssl.StatusAll()
		} else {
			ssl.Status(args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(sslCmd)
	sslCmd.AddCommand(sslEnableCmd)
	sslCmd.AddCommand(sslDisableCmd)
	sslCmd.AddCommand(sslRenewCmd)
	sslCmd.AddCommand(sslStatusCmd)

	// Flags for SSL enable
	sslEnableCmd.Flags().StringP("email", "e", "", "Email address for Let's Encrypt registration")
	sslEnableCmd.Flags().StringP("type", "t", "", "Certificate type: selfsigned or letsencrypt (default: auto-detect)")
}
