package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/scottzirkel/routa/internal/services"
)

func localhostAddr(port string) string {
	return "127.0.0.1:" + port
}

func portInUseError(addr, service string) error {
	return fmt.Errorf("%s is already in use; stop the existing %s process or choose another port", addr, service)
}

func portFromCommand(cmd *cobra.Command, args []string, flagName, flagPort, fallback, label string) (string, error) {
	port := fallback
	flagChanged := cmd.Flags().Changed(flagName)
	if flagChanged {
		port = flagPort
	}
	if len(args) > 0 {
		if len(args) != 2 || args[0] != "on" {
			return "", fmt.Errorf("usage: %s", cmd.UseLine())
		}
		if flagChanged {
			return "", fmt.Errorf("use either --%s or 'on <port>', not both", flagName)
		}
		port = args[1]
	}
	if err := services.ValidateTCPPort(label, port); err != nil {
		return "", err
	}
	return port, nil
}
