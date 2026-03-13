package loop

import "testing"

func TestStrategyString(t *testing.T) {
	tests := []struct {
		s    Strategy
		want string
	}{
		{StrategyStandard, "standard"},
		{StrategyRetry, "retry"},
		{StrategyExploration, "exploration"},
		{Strategy(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("Strategy(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}

func TestFileHistoryImprovedEmpty(t *testing.T) {
	fh := FileHistory{}
	if fh.Improved() {
		t.Error("empty history should not be improved")
	}
}

func TestFileHistoryImprovedTrue(t *testing.T) {
	fh := FileHistory{
		Rounds: []RoundResult{
			{Fixed: 0},
			{Fixed: 2},
		},
	}
	if !fh.Improved() {
		t.Error("expected improved when last round fixed > 0")
	}
}

func TestFileHistoryImprovedFalse(t *testing.T) {
	fh := FileHistory{
		Rounds: []RoundResult{
			{Fixed: 2},
			{Fixed: 0},
		},
	}
	if fh.Improved() {
		t.Error("expected not improved when last round fixed 0")
	}
}

func TestConsecutiveStaleEmpty(t *testing.T) {
	fh := FileHistory{}
	if got := fh.ConsecutiveStale(); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestConsecutiveStaleAllStale(t *testing.T) {
	fh := FileHistory{
		Rounds: []RoundResult{
			{Fixed: 0},
			{Fixed: 0},
			{Fixed: 0},
		},
	}
	if got := fh.ConsecutiveStale(); got != 3 {
		t.Errorf("expected 3, got %d", got)
	}
}

func TestConsecutiveStaleMixed(t *testing.T) {
	fh := FileHistory{
		Rounds: []RoundResult{
			{Fixed: 0},
			{Fixed: 1},
			{Fixed: 0},
			{Fixed: 0},
		},
	}
	if got := fh.ConsecutiveStale(); got != 2 {
		t.Errorf("expected 2, got %d", got)
	}
}

func TestConsecutiveStaleRecentImproved(t *testing.T) {
	fh := FileHistory{
		Rounds: []RoundResult{
			{Fixed: 0},
			{Fixed: 0},
			{Fixed: 3},
		},
	}
	if got := fh.ConsecutiveStale(); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestLastOutputEmpty(t *testing.T) {
	fh := FileHistory{}
	if got := fh.LastOutput(); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestLastOutput(t *testing.T) {
	fh := FileHistory{
		Rounds: []RoundResult{
			{Output: "first"},
			{Output: "second"},
		},
	}
	if got := fh.LastOutput(); got != "second" {
		t.Errorf("expected %q, got %q", "second", got)
	}
}
