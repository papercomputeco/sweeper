package worker

import "time"

type Result struct {
	TaskID       int
	File         string
	Success      bool
	Output       string
	Error        string
	Duration     time.Duration
	IssuesFix    int
	IssuesNew    int
	Provider     string
	Model        string
	PromptTokens int
	OutputTokens int
}
