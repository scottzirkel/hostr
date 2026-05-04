package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/scottzirkel/routa/internal/services"
	"github.com/scottzirkel/routa/internal/systemd"
)

var storageCmd = &cobra.Command{
	Use:   "storage",
	Short: "Manage routa object storage services",
}

var storageInstallPort string
var storageInstallConsolePort string
var storageStartPort string
var storageStartConsolePort string

var storageInstallCmd = &cobra.Command{
	Use:   "install minio <version> [on <port>]",
	Short: "Write MinIO unit and prepare its data directory",
	Args:  storageMinIOVersionPortArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		version := args[1]
		port, consolePort, err := minIOPortsFromCommand(cmd, args[2:], storageInstallPort, storageInstallConsolePort)
		if err != nil {
			return err
		}
		if err := services.EnsureMinIOWithPorts(version, port, consolePort); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "installed %s\n", services.MinIOUnitName(version))
		return nil
	},
}

var storageStartCmd = &cobra.Command{
	Use:   "start minio <version> [on <port>]",
	Short: "Write MinIO unit and start it",
	Args:  storageMinIOVersionPortArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		version := args[1]
		unit := services.MinIOUnitName(version)
		port, consolePort, err := minIOPortsFromCommand(cmd, args[2:], storageStartPort, storageStartConsolePort)
		if err != nil {
			return err
		}
		for label, p := range map[string]string{"minio": port, "minio console": consolePort} {
			if portBound(localhostAddr(p)) && !systemd.IsActive(unit) {
				return portInUseError(localhostAddr(p), label)
			}
		}
		if err := services.EnsureMinIOWithPorts(version, port, consolePort); err != nil {
			return err
		}
		if err := systemd.EnableNow(unit); err != nil {
			return fmt.Errorf("start %s: %w", unit, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "started %s\n", unit)
		return nil
	},
}

var storageStopCmd = &cobra.Command{
	Use:   "stop minio <version>",
	Short: "Stop and disable a routa MinIO service",
	Args:  storageMinIOVersionArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		version := args[1]
		unit := services.MinIOUnitName(version)
		if err := systemd.DisableNow(unit); err != nil {
			return fmt.Errorf("stop %s: %w", unit, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "stopped %s\n", unit)
		return nil
	},
}

var storageStatusCmd = &cobra.Command{
	Use:   "status minio <version>",
	Short: "Show routa MinIO systemd status",
	Args:  storageMinIOVersionArgs,
	RunE: func(_ *cobra.Command, args []string) error {
		return systemd.RunSystemctl("--user", "status", services.MinIOUnitName(args[1]))
	},
}

var storageListCmd = &cobra.Command{
	Use:   "list",
	Short: "List routa object storage services",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		instances, err := services.InstalledMinIOInstances()
		if err != nil {
			return err
		}
		if len(instances) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no object storage services installed. `routa storage install minio <version>`")
			return nil
		}
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
		fmt.Fprintln(w, "ENGINE\tVERSION\tUNIT\tDATA_DIR")
		for _, instance := range instances {
			fmt.Fprintf(w, "minio\t%s\t%s\t%s\n", instance.Version, instance.Unit, instance.DataDir)
		}
		return w.Flush()
	},
}

func storageMinIOVersionArgs(_ *cobra.Command, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("requires minio and version")
	}
	if args[0] != "minio" {
		return fmt.Errorf("unsupported storage engine %q (supported: minio)", args[0])
	}
	return services.ValidateMinIOVersion(args[1])
}

func storageMinIOVersionPortArgs(cmd *cobra.Command, args []string) error {
	if len(args) != 2 && len(args) != 4 {
		return fmt.Errorf("usage: %s", cmd.UseLine())
	}
	return storageMinIOVersionArgs(cmd, args[:2])
}

func minIOPorts(port, consolePort string) (string, string, error) {
	if port == "" {
		port = services.MinIODefaultPort
	}
	if consolePort == "" {
		consolePort = services.MinIODefaultConsolePort
	}
	if err := services.ValidateTCPPort("MinIO", port); err != nil {
		return "", "", err
	}
	if err := services.ValidateTCPPort("MinIO console", consolePort); err != nil {
		return "", "", err
	}
	return port, consolePort, nil
}

func minIOPortsFromCommand(cmd *cobra.Command, args []string, flagPort, consolePort string) (string, string, error) {
	_, fallbackConsolePort, err := minIOPorts("", consolePort)
	if err != nil {
		return "", "", err
	}
	port, err := portFromCommand(cmd, args, "port", flagPort, services.MinIODefaultPort, "MinIO")
	if err != nil {
		return "", "", err
	}
	return port, fallbackConsolePort, nil
}

func init() {
	storageInstallCmd.Flags().StringVar(&storageInstallPort, "port", "", "MinIO API TCP port")
	storageInstallCmd.Flags().StringVar(&storageInstallConsolePort, "console-port", "", "MinIO console TCP port")
	storageStartCmd.Flags().StringVar(&storageStartPort, "port", "", "MinIO API TCP port")
	storageStartCmd.Flags().StringVar(&storageStartConsolePort, "console-port", "", "MinIO console TCP port")
	storageCmd.AddCommand(storageInstallCmd, storageStartCmd, storageStopCmd, storageStatusCmd, storageListCmd)
	rootCmd.AddCommand(storageCmd)
}
