package cmd

import (
	"fmt"

	"github.com/papercomputeco/sweeper/pkg/observer"
	"github.com/papercomputeco/sweeper/pkg/tapes"
	"github.com/spf13/cobra"
)

func newObserveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "observe",
		Short: "Analyze past runs and show learned patterns",
		RunE: func(cmd *cobra.Command, args []string) error {
			var opts []observer.ObserverOption

			if !noTapes {
				dbPath := tapes.FindDB(".")
				if dbPath != "" {
					reader, err := tapes.NewReader(dbPath)
					if err == nil {
						defer reader.Close()
						opts = append(opts, observer.WithTapesReader(reader))
					}
				}
			}

			obs := observer.New(".sweeper/telemetry", opts...)
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
				line := fmt.Sprintf("  %-20s %d/%d (%.0f%%)", i.Linter, i.Successes, i.Attempts, i.SuccessRate*100)
				if i.TotalTokens > 0 {
					line += fmt.Sprintf("  [%d tokens]", i.TotalTokens)
				}
				fmt.Println(line)
			}
			return nil
		},
	}
}
