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
