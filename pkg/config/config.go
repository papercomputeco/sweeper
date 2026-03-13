package config

type Config struct {
	TargetDir    string
	Concurrency  int
	TelemetryDir string
	DryRun       bool
}

func Default() Config {
	return Config{
		TargetDir:    ".",
		Concurrency:  3,
		TelemetryDir: ".sweeper/telemetry",
		DryRun:       false,
	}
}
