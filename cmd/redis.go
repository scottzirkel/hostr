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

var redisStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Write Redis config/unit and start routa-redis",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := services.Ensure(services.Redis()); err != nil {
			return err
		}
		if err := systemd.EnableNow(services.RedisUnitName); err != nil {
			return fmt.Errorf("start %s: %w", services.RedisUnitName, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "started %s\n", services.RedisUnitName)
		return nil
	},
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
	Use:   "restart",
	Short: "Rewrite Redis config/unit and restart routa-redis",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := services.Ensure(services.Redis()); err != nil {
			return err
		}
		if err := systemd.RunSystemctl("--user", "restart", services.RedisUnitName); err != nil {
			return fmt.Errorf("restart %s: %w", services.RedisUnitName, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "restarted %s\n", services.RedisUnitName)
		return nil
	},
}

var redisStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show routa-redis systemd status",
	RunE: func(_ *cobra.Command, _ []string) error {
		return systemd.RunSystemctl("--user", "status", services.RedisUnitName)
	},
}

func init() {
	redisCmd.AddCommand(redisStartCmd, redisStopCmd, redisRestartCmd, redisStatusCmd)
	rootCmd.AddCommand(redisCmd)
}
