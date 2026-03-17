package config

import "time"

// DefaultAllowedTools is the baseline set of tools sweeper agents can use.
// Users can extend this via --allowed-tools without reverting to a blanket bypass.
var DefaultAllowedTools = []string{
	"Read",
	"Write",
	"Edit",
	"Glob",
	"Grep",
}

type Config struct {
	TargetDir      string
	Concurrency    int
	RateLimit      time.Duration // minimum delay between agent dispatches
	AllowedTools   []string      // tools sub-agents are permitted to use
	TelemetryDir   string
	DryRun         bool
	NoTapes        bool
	LintCommand    []string
	LinterName     string
	MaxRounds      int
	StaleThreshold int
	VM             bool   // --vm: boot ephemeral stereOS VM
	VMName         string // --vm-name: use existing VM (no managed lifecycle)
	VMJcard        string // --vm-jcard: custom jcard.toml path
	Provider       string // AI provider name (e.g. "claude", "codex", "ollama")
	ProviderModel  string // model override for the provider
	ProviderAPI    string // API base URL for API-only providers
}

// MaxConcurrency is the hard ceiling for parallel sub-agents regardless of
// user-supplied flags. Keeps API volume within responsible limits.
const MaxConcurrency = 5

func Default() Config {
	return Config{
		TargetDir:      ".",
		Concurrency:    2,
		RateLimit:      2 * time.Second,
		AllowedTools:   append([]string{}, DefaultAllowedTools...),
		TelemetryDir:   ".sweeper/telemetry",
		DryRun:         false,
		MaxRounds:      1,
		StaleThreshold: 2,
		Provider:       "claude",
	}
}

// ClampConcurrency enforces MaxConcurrency and returns the clamped value.
func ClampConcurrency(n int) int {
	if n < 1 {
		return 1
	}
	if n > MaxConcurrency {
		return MaxConcurrency
	}
	return n
}
