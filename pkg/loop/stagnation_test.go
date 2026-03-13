package loop

import "testing"

func TestDetectStagnationEmpty(t *testing.T) {
	if DetectStagnation(FileHistory{}, 2) {
		t.Error("empty history should not be stagnant")
	}
}

func TestDetectStagnationBelowThreshold(t *testing.T) {
	fh := FileHistory{Rounds: []RoundResult{{Fixed: 0}}}
	if DetectStagnation(fh, 2) {
		t.Error("1 stale round should not trigger threshold of 2")
	}
}

func TestDetectStagnationAtThreshold(t *testing.T) {
	fh := FileHistory{Rounds: []RoundResult{{Fixed: 0}, {Fixed: 0}}}
	if !DetectStagnation(fh, 2) {
		t.Error("2 stale rounds should trigger threshold of 2")
	}
}

func TestDetectStagnationAboveThreshold(t *testing.T) {
	fh := FileHistory{Rounds: []RoundResult{{Fixed: 0}, {Fixed: 0}, {Fixed: 0}}}
	if !DetectStagnation(fh, 2) {
		t.Error("3 stale rounds should trigger threshold of 2")
	}
}

func TestDetectStagnationResetByImprovement(t *testing.T) {
	fh := FileHistory{
		Rounds: []RoundResult{{Fixed: 0}, {Fixed: 1}, {Fixed: 0}},
	}
	if DetectStagnation(fh, 2) {
		t.Error("improvement should reset stale count")
	}
}

func TestPickStrategyRoundZero(t *testing.T) {
	fh := FileHistory{Rounds: []RoundResult{{Fixed: 0}, {Fixed: 0}, {Fixed: 0}}}
	if got := PickStrategy(0, fh, 2); got != StrategyStandard {
		t.Errorf("round 0 should be Standard, got %s", got)
	}
}

func TestPickStrategyRetry(t *testing.T) {
	fh := FileHistory{Rounds: []RoundResult{{Fixed: 1}}}
	if got := PickStrategy(1, fh, 2); got != StrategyRetry {
		t.Errorf("round 1 with improvement should be Retry, got %s", got)
	}
}

func TestPickStrategyExploration(t *testing.T) {
	fh := FileHistory{Rounds: []RoundResult{{Fixed: 0}, {Fixed: 0}}}
	if got := PickStrategy(2, fh, 2); got != StrategyExploration {
		t.Errorf("stagnant should be Exploration, got %s", got)
	}
}

func TestPickStrategyEmptyHistoryRoundOne(t *testing.T) {
	if got := PickStrategy(1, FileHistory{}, 2); got != StrategyRetry {
		t.Errorf("empty history at round 1 should be Retry, got %s", got)
	}
}
