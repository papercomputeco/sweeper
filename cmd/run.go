package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/papercomputeco/sweeper/pkg/agent"
	"github.com/papercomputeco/sweeper/pkg/config"
	"github.com/papercomputeco/sweeper/pkg/linter"
	"github.com/papercomputeco/sweeper/pkg/provider"
	"github.com/papercomputeco/sweeper/pkg/vm"
	"github.com/papercomputeco/sweeper/pkg/worker"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var dryRun bool
	var maxRounds int
	var staleThreshold int
	var allowedTools []string
	var useVM bool
	var vmName string
	var vmJcard string
	var providerName string
	var providerModel string
	var providerAPI string
	cmd := &cobra.Command{
		Use:   "run [-- command ...]",
		Short: "Run sweeper against target directory",
		Long: `Run sweeper to lint and fix issues.

Examples:
  sweeper run                              # default: golangci-lint
  sweeper run --max-rounds 3               # retry up to 3 rounds
  sweeper run -- npm run lint              # arbitrary command
  npm run lint | sweeper run               # piped stdin`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			clamped := config.ClampConcurrency(concurrency)
			if clamped != concurrency {
				fmt.Printf("Concurrency clamped to %d (max %d)\n", clamped, config.MaxConcurrency)
			}
			tools := append([]string{}, config.DefaultAllowedTools...)
			if len(allowedTools) > 0 {
				tools = append(tools, allowedTools...)
			}
			cfg := config.Config{
				TargetDir:      targetDir,
				Concurrency:    clamped,
				RateLimit:      rateLimit,
				AllowedTools:   tools,
				TelemetryDir:   ".sweeper/telemetry",
				DryRun:         dryRun,
				NoTapes:        noTapes,
				MaxRounds:      maxRounds,
				StaleThreshold: staleThreshold,
				Provider:       providerName,
				ProviderModel:  providerModel,
				ProviderAPI:    providerAPI,
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

			if vmName != "" || vmJcard != "" {
				useVM = true
			}
			cfg.VM = useVM
			cfg.VMName = vmName
			cfg.VMJcard = vmJcard

			// Validate: --vm is only compatible with CLI providers.
			if useVM {
				p, err := provider.Get(cfg.Provider)
				if err != nil {
					return fmt.Errorf("provider %q: %w", cfg.Provider, err)
				}
				if p.Kind != provider.KindCLI {
					return fmt.Errorf("--vm is only compatible with CLI providers (got %q)", cfg.Provider)
				}
			}

			if useVM {
				absTarget, _ := filepath.Abs(cfg.TargetDir)
				if cfg.VMName != "" {
					vmHandle := vm.Attach(cfg.VMName, absTarget)
					opts = append(opts, agent.WithVM(vmHandle))
					opts = append(opts, agent.WithExecutor(worker.NewVMExecutor(vmHandle)))
					fmt.Printf("VM: using existing VM %s\n", cfg.VMName)
				} else {
					name := vm.NewVMName()
					jcardDir := filepath.Join(absTarget, ".sweeper", "vm")
					if cfg.VMJcard != "" {
						jcardDir = filepath.Dir(cfg.VMJcard)
					}
					vmHandle, err := vm.Boot(name, absTarget, jcardDir)
					if err != nil {
						return fmt.Errorf("booting VM: %w", err)
					}
					opts = append(opts, agent.WithVM(vmHandle))
					opts = append(opts, agent.WithExecutor(worker.NewVMExecutor(vmHandle)))
					fmt.Printf("VM: booted %s (managed, will teardown on exit)\n", name)
				}
			}

			a := agent.New(cfg, opts...)
			summary, err := a.Run(ctx)
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
	cmd.Flags().StringSliceVar(&allowedTools, "allowed-tools", nil, "additional tools for sub-agents (e.g. 'Bash(npm:*),Bash(cargo:*)')")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be fixed without making changes")
	cmd.Flags().IntVar(&maxRounds, "max-rounds", 1, "maximum retry rounds (1 = single pass)")
	cmd.Flags().IntVar(&staleThreshold, "stale-threshold", 2, "consecutive non-improving rounds before exploration mode")
	cmd.Flags().BoolVar(&useVM, "vm", false, "boot ephemeral stereOS VM, teardown on exit")
	cmd.Flags().StringVar(&vmName, "vm-name", "", "use existing VM by name (no managed lifecycle, implies --vm)")
	cmd.Flags().StringVar(&vmJcard, "vm-jcard", "", "custom jcard.toml path (implies --vm)")
	cmd.Flags().StringVar(&providerName, "provider", "claude", "AI provider (claude, codex, ollama)")
	cmd.Flags().StringVar(&providerModel, "model", "", "model name for the provider (e.g. qwen2.5-coder:7b)")
	cmd.Flags().StringVar(&providerAPI, "api-base", "", "API base URL for API providers (e.g. http://localhost:11434)")
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
