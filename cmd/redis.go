package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/scottzirkel/routa/internal/services"
	"github.com/scottzirkel/routa/internal/systemd"
)

var redisCmd = &cobra.Command{
	Use:   "redis",
	Short: "Manage routa Redis",
}

var redisStartPort string
var redisRestartPort string

var redisStartCmd = &cobra.Command{
	Use:   "start [on <port>]",
	Short: "Write Redis config/unit and start routa-redis",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		currentPort, err := services.RedisConfiguredPort()
		if err != nil {
			return err
		}
		port, err := redisPortFromCommand(cmd, args, redisStartPort, currentPort)
		if err != nil {
			return err
		}
		active := systemd.IsActive(services.RedisUnitName)
		if redisPortBound(port) && (!active || port != currentPort) {
			return redisPortConflictError(port)
		}
		if err := services.Ensure(services.RedisWithPort(port)); err != nil {
			return err
		}
		_ = systemd.RunSystemctl("--user", "reset-failed", services.RedisUnitName)
		if active && port != currentPort {
			if err := systemd.RunSystemctl("--user", "restart", services.RedisUnitName); err != nil {
				return fmt.Errorf("restart %s: %w", services.RedisUnitName, err)
			}
		} else if err := systemd.EnableNow(services.RedisUnitName); err != nil {
			return fmt.Errorf("start %s: %w", services.RedisUnitName, err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), redisStartedMessage(port))
		return nil
	},
}

func redisStartedMessage(port string) string {
	return fmt.Sprintf("started %s on %s", services.RedisUnitName, services.RedisAddr(port))
}

var redisStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop and disable routa-redis",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := systemd.DisableNow(services.RedisUnitName); err != nil {
			return fmt.Errorf("stop %s: %w", services.RedisUnitName, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "stopped %s\n", services.RedisUnitName)
		return nil
	},
}

var redisRestartCmd = &cobra.Command{
	Use:   "restart [on <port>]",
	Short: "Rewrite Redis config/unit and restart routa-redis",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		currentPort, err := services.RedisConfiguredPort()
		if err != nil {
			return err
		}
		port, err := redisPortFromCommand(cmd, args, redisRestartPort, currentPort)
		if err != nil {
			return err
		}
		if redisPortBound(port) && port != currentPort {
			return redisPortConflictError(port)
		}
		if err := services.Ensure(services.RedisWithPort(port)); err != nil {
			return err
		}
		if err := systemd.RunSystemctl("--user", "restart", services.RedisUnitName); err != nil {
			return fmt.Errorf("restart %s: %w", services.RedisUnitName, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "restarted %s on %s\n", services.RedisUnitName, services.RedisAddr(port))
		return nil
	},
}

var redisStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show routa-redis systemd status",
	RunE: func(cmd *cobra.Command, _ []string) error {
		port, err := services.RedisConfiguredPort()
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), redisStatusHeader(port))
		return systemd.RunSystemctl("--user", "status", services.RedisUnitName)
	},
}

func redisStatusHeader(port string) string {
	return fmt.Sprintf("%s listens on %s", services.RedisUnitName, services.RedisAddr(port))
}

func redisPortBound(port string) bool {
	return portBound(services.RedisAddr(port))
}

func redisPortConflictError(port string) error {
	return fmt.Errorf("%s is already in use by another process; stop that Redis/Valkey instance or choose another Redis port", services.RedisAddr(port))
}

func redisPortFromCommand(cmd *cobra.Command, args []string, flagPort, fallback string) (string, error) {
	port := fallback
	flagChanged := cmd.Flags().Changed("port")
	if flagChanged {
		port = flagPort
	}
	if len(args) > 0 {
		if len(args) != 2 || args[0] != "on" {
			return "", fmt.Errorf("usage: %s", cmd.UseLine())
		}
		if flagChanged {
			return "", fmt.Errorf("use either --port or 'on <port>', not both")
		}
		port = args[1]
	}
	if err := services.ValidateRedisPort(port); err != nil {
		return "", err
	}
	return port, nil
}

func init() {
	redisStartCmd.Flags().StringVar(&redisStartPort, "port", services.RedisDefaultPort, "Redis TCP port")
	redisRestartCmd.Flags().StringVar(&redisRestartPort, "port", "", "Redis TCP port")
	redisCmd.AddCommand(redisStartCmd, redisStopCmd, redisRestartCmd, redisStatusCmd)
	rootCmd.AddCommand(redisCmd)
}
