package cmd

import (
	"webstack-cli/internal/installer"

	"github.com/spf13/cobra"
)

var mailCmd = &cobra.Command{
	Use:   "mail",
	Short: "Mail server management",
	Long:  `Manage mail server accounts, domains, and configuration.`,
}

var mailAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add mail accounts or domains",
	Long:  `Add new mail accounts or domains to the mail server.`,
}

var mailAccountCmd = &cobra.Command{
	Use:   "account <email> <password>",
	Short: "Add a mail account",
	Long:  `Add a new mail account with format: webstack mail add account user@domain.tld password`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		installer.AddMailAccount(args[0], args[1])
	},
}

var mailDomainCmd = &cobra.Command{
	Use:   "domain <domain>",
	Short: "Add a mail domain",
	Long:  `Add a new mail domain: webstack mail add domain mydomain.tld`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		installer.AddMailDomain(args[0])
	},
}

var mailListCmd = &cobra.Command{
	Use:   "list",
	Short: "List mail accounts and domains",
	Long:  `List all configured mail accounts and domains.`,
}

var mailListAccountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "List all mail accounts",
	Run: func(cmd *cobra.Command, args []string) {
		installer.ListMailAccounts()
	},
}

var mailListDomainsCmd = &cobra.Command{
	Use:   "domains",
	Short: "List all mail domains",
	Run: func(cmd *cobra.Command, args []string) {
		installer.ListMailDomains()
	},
}

var mailDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete mail accounts or domains",
	Long:  `Delete mail accounts or domains.`,
}

var mailDeleteAccountCmd = &cobra.Command{
	Use:   "account <email>",
	Short: "Delete a mail account",
	Long:  `Delete a mail account: webstack mail delete account user@domain.tld`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		installer.DeleteMailAccount(args[0])
	},
}

var mailDeleteDomainCmd = &cobra.Command{
	Use:   "domain <domain>",
	Short: "Delete a mail domain",
	Long:  `Delete a mail domain: webstack mail delete domain mydomain.tld`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		installer.DeleteMailDomain(args[0])
	},
}

var mailShowDNSCmd = &cobra.Command{
	Use:   "show-dns-records <domain>",
	Short: "Show DNS records for a domain",
	Long:  `Display SPF, DKIM, and DMARC records to add to your DNS provider: webstack mail show-dns-records mydomain.tld`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		installer.ShowDNSRecords(args[0])
	},
}

var mailDNSCmd = &cobra.Command{
	Use:   "dns",
	Short: "Mail DNS records management",
	Long:  `Manage mail DNS records (SPF, DKIM, DMARC) for domains.`,
}

var mailDNSShowCmd = &cobra.Command{
	Use:   "show <domain>",
	Short: "Show DNS records for a domain",
	Long:  `Display SPF, DKIM, and DMARC records for a domain: webstack mail dns show mydomain.tld`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		installer.ShowDNSRecords(args[0])
	},
}

var mailDNSBindCmd = &cobra.Command{
	Use:   "bind <domain>",
	Short: "Import DNS records into BIND",
	Long:  `Import SPF, DKIM, and DMARC records into BIND (if installed): webstack mail dns bind mydomain.tld`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		installer.ImportMailDNSToBind(args[0])
	},
}

var mailFirewallCmd = &cobra.Command{
	Use:   "firewall",
	Short: "Mail firewall port management",
	Long:  `Open or close firewall ports required for mail server.`,
}

var mailFirewallOpenCmd = &cobra.Command{
	Use:   "open",
	Short: "Open mail server ports in firewall",
	Long:  `Open all necessary mail ports (SMTP, POP3, IMAP, etc.) in firewall if present: sudo webstack mail firewall open`,
	Run: func(cmd *cobra.Command, args []string) {
		installer.AddMailFirewallRules()
	},
}

var mailFirewallCloseCmd = &cobra.Command{
	Use:   "close",
	Short: "Close mail server ports in firewall",
	Long:  `Close all mail ports in firewall if present: sudo webstack mail firewall close`,
	Run: func(cmd *cobra.Command, args []string) {
		installer.RemoveMailFirewallRules()
	},
}

func init() {
	rootCmd.AddCommand(mailCmd)

	// Add subcommands
	mailCmd.AddCommand(mailAddCmd)
	mailCmd.AddCommand(mailListCmd)
	mailCmd.AddCommand(mailDeleteCmd)
	mailCmd.AddCommand(mailShowDNSCmd)
	mailCmd.AddCommand(mailDNSCmd)
	mailCmd.AddCommand(mailFirewallCmd)

	// Mail add subcommands
	mailAddCmd.AddCommand(mailAccountCmd)
	mailAddCmd.AddCommand(mailDomainCmd)

	// Mail list subcommands
	mailListCmd.AddCommand(mailListAccountsCmd)
	mailListCmd.AddCommand(mailListDomainsCmd)

	// Mail delete subcommands
	mailDeleteCmd.AddCommand(mailDeleteAccountCmd)
	mailDeleteCmd.AddCommand(mailDeleteDomainCmd)

	// Mail DNS subcommands
	mailDNSCmd.AddCommand(mailDNSShowCmd)
	mailDNSCmd.AddCommand(mailDNSBindCmd)

	// Mail firewall subcommands
	mailFirewallCmd.AddCommand(mailFirewallOpenCmd)
	mailFirewallCmd.AddCommand(mailFirewallCloseCmd)
}
