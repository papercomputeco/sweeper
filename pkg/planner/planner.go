package planner

import (
	"sort"

	"github.com/papercomputeco/sweeper/pkg/linter"
)

type FixTask struct {
	File   string
	Issues []linter.Issue
}

func GroupByFile(issues []linter.Issue) []FixTask {
	if len(issues) == 0 {
		return nil
	}
	grouped := make(map[string][]linter.Issue)
	for _, iss := range issues {
		grouped[iss.File] = append(grouped[iss.File], iss)
	}
	tasks := make([]FixTask, 0, len(grouped))
	for file, issues := range grouped {
		tasks = append(tasks, FixTask{File: file, Issues: issues})
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].File < tasks[j].File
	})
	return tasks
}
