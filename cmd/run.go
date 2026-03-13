package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/papercomputeco/sweeper/pkg/agent"
	"github.com/papercomputeco/sweeper/pkg/config"
	"github.com/papercomputeco/sweeper/pkg/linter"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "run [-- command ...]",
		Short: "Run sweeper against target directory",
		Long: `Run sweeper to lint and fix issues.

Examples:
  sweeper run                              # default: golangci-lint
  sweeper run -- npm run lint              # arbitrary command
  npm run lint | sweeper run               # piped stdin`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Config{
				TargetDir:    targetDir,
				Concurrency:  concurrency,
				TelemetryDir: ".sweeper/telemetry",
				DryRun:       dryRun,
				NoTapes:      noTapes,
			}

			piped := isPiped()
			dashArgs := argsAfterDash(cmd, args)

			if piped && len(dashArgs) > 0 {
				return fmt.Errorf("cannot use both piped input and -- command; choose one")
			}

			var opts []agent.Option

			if piped {
				cfg.LinterName = "custom"
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("reading stdin: %w", err)
				}
				raw := string(data)
				opts = append(opts, agent.WithLinterFunc(
					func(ctx context.Context, dir string) (linter.ParseResult, error) {
						return linter.ParseOutput(raw), nil
					},
				))
			} else if len(dashArgs) > 0 {
				cfg.LintCommand = dashArgs
				cfg.LinterName = filepath.Base(dashArgs[0])
				opts = append(opts, agent.WithLinterFunc(
					func(ctx context.Context, dir string) (linter.ParseResult, error) {
						return linter.RunCommand(ctx, dir, dashArgs)
					},
				))
			}

			a := agent.New(cfg, opts...)
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

func isPiped() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice == 0
}

func argsAfterDash(cmd *cobra.Command, args []string) []string {
	idx := cmd.ArgsLenAtDash()
	if idx < 0 {
		return nil
	}
	return args[idx:]
}
