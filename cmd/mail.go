package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/scottzirkel/routa/internal/services"
	"github.com/scottzirkel/routa/internal/site"
	"github.com/scottzirkel/routa/internal/systemd"
)

var mailCmd = &cobra.Command{
	Use:   "mail",
	Short: "Manage routa Mailpit",
}

var mailStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Write Mailpit unit and start routa-mailpit",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := services.Ensure(services.Mailpit()); err != nil {
			return err
		}
		if err := systemd.EnableNow(services.MailpitUnitName); err != nil {
			return fmt.Errorf("start %s: %w", services.MailpitUnitName, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "started %s\n", services.MailpitUnitName)
		return nil
	},
}

var mailStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop and disable routa-mailpit",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := systemd.DisableNow(services.MailpitUnitName); err != nil {
			return fmt.Errorf("stop %s: %w", services.MailpitUnitName, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "stopped %s\n", services.MailpitUnitName)
		return nil
	},
}

var mailRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Rewrite Mailpit unit and restart routa-mailpit",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := services.Ensure(services.Mailpit()); err != nil {
			return err
		}
		if err := systemd.RunSystemctl("--user", "restart", services.MailpitUnitName); err != nil {
			return fmt.Errorf("restart %s: %w", services.MailpitUnitName, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "restarted %s\n", services.MailpitUnitName)
		return nil
	},
}

var mailStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show routa-mailpit systemd status",
	RunE: func(_ *cobra.Command, _ []string) error {
		return systemd.RunSystemctl("--user", "status", services.MailpitUnitName)
	},
}

var mailProxyCmd = &cobra.Command{
	Use:   "proxy [name]",
	Short: "Proxy Mailpit's web UI as <name>.test (default: mail.test)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		link, err := mailpitProxyLink(args)
		if err != nil {
			return err
		}
		s, err := site.Load()
		if err != nil {
			return err
		}
		site.AddLink(s, link)
		return commitAndReload(s, fmt.Sprintf("proxy %s.test -> %s", link.Name, link.Target))
	},
}

func mailpitProxyLink(args []string) (site.Link, error) {
	name := "mail"
	if len(args) == 1 {
		name = args[0]
	}
	normalized, err := normalizeSiteName(name)
	if err != nil {
		return site.Link{}, err
	}
	return site.Link{Name: normalized, Target: services.MailpitWebAddr, Secure: true}, nil
}

func init() {
	mailCmd.AddCommand(mailStartCmd, mailStopCmd, mailRestartCmd, mailStatusCmd, mailProxyCmd)
	rootCmd.AddCommand(mailCmd)
}
