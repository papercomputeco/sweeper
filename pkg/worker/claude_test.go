package worker

import (
	"strings"
	"testing"

	"github.com/papercomputeco/sweeper/pkg/linter"
)

func TestBuildPrompt(t *testing.T) {
	task := Task{
		File: "server.go",
		Issues: []linter.Issue{
			{File: "server.go", Line: 42, Message: "err is not used", Linter: "ineffassign"},
			{File: "server.go", Line: 55, Message: "exported function should have comment", Linter: "revive"},
		},
	}
	prompt := BuildPrompt(task)
	if !strings.Contains(prompt, "server.go") {
		t.Error("prompt should reference the file")
	}
	if !strings.Contains(prompt, "Line 42") {
		t.Error("prompt should include line numbers")
	}
	if !strings.Contains(prompt, "ineffassign") {
		t.Error("prompt should include linter names")
	}
}

func TestBuildRawPrompt(t *testing.T) {
	task := Task{
		Dir:       "/tmp/project",
		RawOutput: "ERROR: something went wrong\n  --> src/lib.rs:45\n",
	}
	prompt := BuildRawPrompt(task)
	if !strings.Contains(prompt, "ERROR: something went wrong") {
		t.Error("raw prompt should contain the original output")
	}
	if !strings.Contains(prompt, "Analyze it") {
		t.Error("raw prompt should instruct the agent to analyze")
	}
	if !strings.Contains(prompt, "Commit nothing") {
		t.Error("raw prompt should instruct not to commit")
	}
}
