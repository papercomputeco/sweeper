package cmd

import (
	"fmt"
	"github.com/papercomputeco/sweeper/pkg/observer"
	"github.com/spf13/cobra"
)

func newObserveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "observe",
		Short: "Analyze past runs and show learned patterns",
		RunE: func(cmd *cobra.Command, args []string) error {
			obs := observer.New(".sweeper/telemetry")
			insights, err := obs.Analyze()
			if err != nil {
				return err
			}
			if len(insights) == 0 {
				fmt.Println("No past runs found. Run `sweeper run` first.")
				return nil
			}
			fmt.Println("Fix success rates by linter:")
			for _, i := range insights {
				fmt.Printf("  %-20s %d/%d (%.0f%%)\n",
					i.Linter, i.Successes, i.Attempts, i.SuccessRate*100)
			}
			return nil
		},
	}
}
