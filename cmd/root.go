package cmd

import (
	"github.com/spf13/cobra"
)

var (
	targetDir   string
	concurrency int
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "sweeper",
		Short: "AI-powered code sweeper",
		Long:  "Runs linters, dispatches Claude Code sub-agents to fix issues in parallel, and learns from outcomes.",
	}
	root.PersistentFlags().StringVarP(&targetDir, "target", "t", ".", "target directory to maintain")
	root.PersistentFlags().IntVarP(&concurrency, "concurrency", "c", 3, "max parallel sub-agents")
	root.AddCommand(newVersionCmd())
	return root
}

func Execute() error {
	return NewRootCmd().Execute()
}
