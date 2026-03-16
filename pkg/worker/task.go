package worker

import (
	"fmt"
	"strings"

	"github.com/papercomputeco/sweeper/pkg/linter"
)

// agentPreamble identifies this as an automated tool to comply with
// Anthropic's agentic system usage requirements.
const agentPreamble = "You are a sub-agent of Sweeper, an automated code maintenance tool. " +
	"A human developer initiated this run and will review all changes. " +
	"Your sole task is to fix lint issues in source code. " +
	"Do not modify behavior, do not commit, and do not access external services.\n\n"

type Task struct {
	ID        int
	File      string
	Dir       string
	Issues    []linter.Issue
	Prompt    string
	RawOutput string
}

func BuildPrompt(task Task) string {
	var b strings.Builder
	b.WriteString(agentPreamble)
	b.WriteString("Fix the following lint issues in " + task.File + ":\n\n")
	for _, iss := range task.Issues {
		fmt.Fprintf(&b, "- Line %d: %s (%s)\n", iss.Line, iss.Message, iss.Linter)
	}
	b.WriteString("\nFix each issue. Do not change behavior. Only fix lint issues. Commit nothing.")
	return b.String()
}

func BuildRawPrompt(task Task) string {
	var b strings.Builder
	b.WriteString(agentPreamble)
	b.WriteString("The following lint output was produced. Analyze it, identify the issues, and fix them:\n\n")
	b.WriteString(task.RawOutput)
	b.WriteString("\n\nFix each issue you can identify. Do not change behavior. Only fix lint issues. Commit nothing.")
	return b.String()
}

// BuildRetryPrompt creates a retry prompt that includes the prior attempt output
// and instructs the agent to try a different approach.
func BuildRetryPrompt(task Task, priorOutput string) string {
	var b strings.Builder
	b.WriteString(agentPreamble)
	b.WriteString("Fix the following lint issues in " + task.File + ":\n\n")
	for _, iss := range task.Issues {
		fmt.Fprintf(&b, "- Line %d: %s (%s)\n", iss.Line, iss.Message, iss.Linter)
	}
	b.WriteString("\nA previous attempt did not fully resolve these issues. Here is what was tried:\n\n")
	b.WriteString(truncateOutput(priorOutput, 2000))
	b.WriteString("\n\nTry a different approach. Do not repeat what was already tried. Fix each issue. Do not change behavior. Commit nothing.")
	return b.String()
}

// BuildExplorationPrompt creates an exploration prompt triggered after stagnation.
// It instructs the agent to consider refactoring surrounding code.
func BuildExplorationPrompt(task Task, priorOutput string) string {
	var b strings.Builder
	b.WriteString(agentPreamble)
	b.WriteString("Fix the following lint issues in " + task.File + ":\n\n")
	for _, iss := range task.Issues {
		fmt.Fprintf(&b, "- Line %d: %s (%s)\n", iss.Line, iss.Message, iss.Linter)
	}
	b.WriteString("\nPrevious approaches have not resolved these issues.")
	b.WriteString("\n\nPrior attempt output:\n\n")
	b.WriteString(truncateOutput(priorOutput, 2000))
	b.WriteString("\n\nConsider refactoring the surrounding code. The lint issues may stem from a deeper structural problem. ")
	b.WriteString("You may modify adjacent functions or extract code as needed, but do not change observable behavior. Commit nothing.")
	return b.String()
}

func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
