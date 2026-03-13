package loop

// Strategy represents the prompt escalation level for a retry round.
type Strategy int

const (
	StrategyStandard    Strategy = iota // Round 0: normal prompt
	StrategyRetry                       // Round 1+: include prior output
	StrategyExploration                 // After stagnation: refactoring directive
)

func (s Strategy) String() string {
	switch s {
	case StrategyStandard:
		return "standard"
	case StrategyRetry:
		return "retry"
	case StrategyExploration:
		return "exploration"
	default:
		return "unknown"
	}
}

// RoundResult captures the outcome of a single round for a specific file.
type RoundResult struct {
	File         string
	Round        int
	Strategy     Strategy
	IssuesBefore int
	IssuesAfter  int
	Fixed        int
	Output       string
	Success      bool
	Error        string
}

// FileHistory is the accumulated attempt history for a single file across rounds.
type FileHistory struct {
	File   string
	Rounds []RoundResult
}

// Improved returns true if the latest round fixed at least one issue.
func (fh FileHistory) Improved() bool {
	if len(fh.Rounds) == 0 {
		return false
	}
	return fh.Rounds[len(fh.Rounds)-1].Fixed > 0
}

// ConsecutiveStale returns how many consecutive most-recent rounds had zero improvement.
func (fh FileHistory) ConsecutiveStale() int {
	count := 0
	for i := len(fh.Rounds) - 1; i >= 0; i-- {
		if fh.Rounds[i].Fixed > 0 {
			break
		}
		count++
	}
	return count
}

// LastOutput returns the Output from the most recent round, or "" if empty.
func (fh FileHistory) LastOutput() string {
	if len(fh.Rounds) == 0 {
		return ""
	}
	return fh.Rounds[len(fh.Rounds)-1].Output
}
