package loop

// DetectStagnation returns true if the file has had threshold or more
// consecutive rounds with zero improvement.
func DetectStagnation(history FileHistory, threshold int) bool {
	return history.ConsecutiveStale() >= threshold
}

// PickStrategy determines the prompt strategy for the next round based on
// the round number and stagnation state.
func PickStrategy(round int, history FileHistory, staleThreshold int) Strategy {
	if round == 0 {
		return StrategyStandard
	}
	if DetectStagnation(history, staleThreshold) {
		return StrategyExploration
	}
	return StrategyRetry
}
