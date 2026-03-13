package planner

import (
	"testing"

	"github.com/papercomputeco/sweeper/pkg/linter"
)

func TestGroupByFile(t *testing.T) {
	issues := []linter.Issue{
		{File: "a.go", Line: 1, Linter: "revive", Message: "msg1"},
		{File: "a.go", Line: 5, Linter: "revive", Message: "msg2"},
		{File: "b.go", Line: 3, Linter: "ineffassign", Message: "msg3"},
	}
	tasks := GroupByFile(issues)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].File != "a.go" {
		t.Errorf("expected first task file a.go, got %s", tasks[0].File)
	}
	if len(tasks[0].Issues) != 2 {
		t.Errorf("expected 2 issues for a.go, got %d", len(tasks[0].Issues))
	}
}

func TestGroupByFileEmpty(t *testing.T) {
	tasks := GroupByFile(nil)
	if len(tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(tasks))
	}
}
