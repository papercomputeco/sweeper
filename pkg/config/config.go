package config

type Config struct {
	TargetDir      string
	Concurrency    int
	TelemetryDir   string
	DryRun         bool
	NoTapes        bool
	LintCommand    []string
	LinterName     string
	MaxRounds      int
	StaleThreshold int
}

func Default() Config {
	return Config{
		TargetDir:      ".",
		Concurrency:    3,
		TelemetryDir:   ".sweeper/telemetry",
		DryRun:         false,
		MaxRounds:      1,
		StaleThreshold: 2,
	}
}
