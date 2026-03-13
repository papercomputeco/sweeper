package cmd

import (
	"context"
	"fmt"
	"os"
	"github.com/papercomputeco/sweeper/pkg/agent"
	"github.com/papercomputeco/sweeper/pkg/config"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run sweeper against target directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Config{
				TargetDir:    targetDir,
				Concurrency:  concurrency,
				TelemetryDir: ".sweeper/telemetry",
				DryRun:       dryRun,
				NoTapes:      noTapes,
			}
			a := agent.New(cfg)
			summary, err := a.Run(context.Background())
			if err != nil {
				return err
			}
			fmt.Printf("\nSummary: %d issues found, %d fixed, %d tasks failed\n",
				summary.TotalIssues, summary.Fixed, summary.Failed)
			if summary.Failed > 0 {
				os.Exit(1)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be fixed without making changes")
	return cmd
}
