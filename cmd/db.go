package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/scottzirkel/routa/internal/services"
	"github.com/scottzirkel/routa/internal/systemd"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage routa database services",
}

var dbInstallPort string
var dbStartPort string

var dbInstallCmd = &cobra.Command{
	Use:   "install <engine> <version>",
	Short: "Write database config/unit and initialize its data directory",
	Args:  dbEngineVersionArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		engine := args[0]
		version := args[1]
		port, err := databasePort(engine, dbInstallPort)
		if err != nil {
			return err
		}
		if err := ensureDatabase(engine, version, port); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "installed %s\n", databaseUnitName(engine, version))
		return nil
	},
}

var dbStartCmd = &cobra.Command{
	Use:   "start <engine> <version>",
	Short: "Write database config/unit and start it",
	Args:  dbEngineVersionArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		engine := args[0]
		version := args[1]
		unit := databaseUnitName(engine, version)
		port, err := databasePort(engine, dbStartPort)
		if err != nil {
			return err
		}
		if portBound(localhostAddr(port)) && !systemd.IsActive(unit) {
			return portInUseError(localhostAddr(port), engine)
		}
		if err := ensureDatabase(engine, version, port); err != nil {
			return err
		}
		if err := systemd.EnableNow(unit); err != nil {
			return fmt.Errorf("start %s: %w", unit, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "started %s\n", unit)
		return nil
	},
}

var dbStopCmd = &cobra.Command{
	Use:   "stop <engine> <version>",
	Short: "Stop and disable a routa database service",
	Args:  dbEngineVersionArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		engine := args[0]
		version := args[1]
		unit := databaseUnitName(engine, version)
		if err := systemd.DisableNow(unit); err != nil {
			return fmt.Errorf("stop %s: %w", unit, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "stopped %s\n", unit)
		return nil
	},
}

var dbStatusCmd = &cobra.Command{
	Use:   "status <engine> <version>",
	Short: "Show routa database systemd status",
	Args:  dbEngineVersionArgs,
	RunE: func(_ *cobra.Command, args []string) error {
		return systemd.RunSystemctl("--user", "status", databaseUnitName(args[0], args[1]))
	},
}

var dbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List routa database services",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		mariadbInstances, err := services.InstalledMariaDBInstances()
		if err != nil {
			return err
		}
		postgresInstances, err := services.InstalledPostgresInstances()
		if err != nil {
			return err
		}
		if len(mariadbInstances) == 0 && len(postgresInstances) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no database services installed. `routa db install mariadb <version>` or `routa db install postgres <version>`")
			return nil
		}
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
		fmt.Fprintln(w, "ENGINE\tVERSION\tUNIT\tDATA_DIR")
		for _, instance := range mariadbInstances {
			fmt.Fprintf(w, "mariadb\t%s\t%s\t%s\n", instance.Version, instance.Unit, instance.DataDir)
		}
		for _, instance := range postgresInstances {
			fmt.Fprintf(w, "postgres\t%s\t%s\t%s\n", instance.Version, instance.Unit, instance.DataDir)
		}
		return w.Flush()
	},
}

func dbEngineVersionArgs(_ *cobra.Command, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("requires engine and version")
	}
	switch args[0] {
	case "mariadb":
		return services.ValidateMariaDBVersion(args[1])
	case "postgres":
		return services.ValidatePostgresVersion(args[1])
	default:
		return fmt.Errorf("unsupported database %q (supported: mariadb, postgres)", args[0])
	}
}

func ensureDatabase(engine, version, port string) error {
	switch engine {
	case "mariadb":
		return services.EnsureMariaDBWithPort(version, port)
	case "postgres":
		return services.EnsurePostgresWithPort(version, port)
	default:
		return fmt.Errorf("unsupported database %q (supported: mariadb, postgres)", engine)
	}
}

func databasePort(engine, port string) (string, error) {
	if port == "" {
		switch engine {
		case "mariadb":
			port = services.MariaDBDefaultPort
		case "postgres":
			port = services.PostgresDefaultPort
		}
	}
	if err := services.ValidateTCPPort(engine, port); err != nil {
		return "", err
	}
	return port, nil
}

func databaseUnitName(engine, version string) string {
	switch engine {
	case "mariadb":
		return services.MariaDBUnitName(version)
	case "postgres":
		return services.PostgresUnitName(version)
	default:
		return ""
	}
}

func init() {
	dbInstallCmd.Flags().StringVar(&dbInstallPort, "port", "", "database TCP port")
	dbStartCmd.Flags().StringVar(&dbStartPort, "port", "", "database TCP port")
	dbCmd.AddCommand(dbInstallCmd, dbStartCmd, dbStopCmd, dbStatusCmd, dbListCmd)
	rootCmd.AddCommand(dbCmd)
}
