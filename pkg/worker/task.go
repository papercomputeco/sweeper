package worker

import (
	"fmt"
	"strings"

	"github.com/papercomputeco/sweeper/pkg/linter"
)

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
	b.WriteString("Fix the following lint issues in " + task.File + ":\n\n")
	for _, iss := range task.Issues {
		fmt.Fprintf(&b, "- Line %d: %s (%s)\n", iss.Line, iss.Message, iss.Linter)
	}
	b.WriteString("\nFix each issue. Do not change behavior. Only fix lint issues. Commit nothing.")
	return b.String()
}

func BuildRawPrompt(task Task) string {
	var b strings.Builder
	b.WriteString("The following lint output was produced. Analyze it, identify the issues, and fix them:\n\n")
	b.WriteString(task.RawOutput)
	b.WriteString("\n\nFix each issue you can identify. Do not change behavior. Only fix lint issues. Commit nothing.")
	return b.String()
}

// BuildRetryPrompt creates a retry prompt that includes the prior attempt output
// and instructs the agent to try a different approach.
func BuildRetryPrompt(task Task, priorOutput string) string {
	var b strings.Builder
	b.WriteString("Fix the following lint issues in " + task.File + ":\n\n")
	for _, iss := range task.Issues {
		fmt.Fprintf(&b, "- Line %d: %s (%s)\n", iss.Line, iss.Message, iss.Linter)
	}
	b.WriteString("\nYour previous attempt did not fully resolve these issues. Here is what you tried:\n\n")
	b.WriteString(truncateOutput(priorOutput, 2000))
	b.WriteString("\n\nTry a different approach. Do not repeat what was already tried. Fix each issue. Do not change behavior. Commit nothing.")
	return b.String()
}

// BuildExplorationPrompt creates an exploration prompt triggered after stagnation.
// It instructs the agent to consider refactoring surrounding code.
func BuildExplorationPrompt(task Task, priorOutput string) string {
	var b strings.Builder
	b.WriteString("Fix the following lint issues in " + task.File + ":\n\n")
	for _, iss := range task.Issues {
		fmt.Fprintf(&b, "- Line %d: %s (%s)\n", iss.Line, iss.Message, iss.Linter)
	}
	b.WriteString("\nWARNING: Multiple previous attempts have failed to resolve these issues.")
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
