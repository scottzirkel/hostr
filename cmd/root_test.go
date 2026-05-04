package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestConfigureRootExecutionSilencesCobraNoise(t *testing.T) {
	cmd := &cobra.Command{Use: "routa"}

	configureRootExecution(cmd)

	if !cmd.SilenceErrors {
		t.Fatal("root command should silence Cobra error printing")
	}
	if !cmd.SilenceUsage {
		t.Fatal("root command should silence Cobra usage printing")
	}
}
