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
