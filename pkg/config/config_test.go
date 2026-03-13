package config

import "testing"

func TestDefaults(t *testing.T) {
	cfg := Default()
	if cfg.Concurrency != 3 {
		t.Errorf("expected default concurrency 3, got %d", cfg.Concurrency)
	}
	if cfg.TelemetryDir != ".sweeper/telemetry" {
		t.Errorf("unexpected telemetry dir: %s", cfg.TelemetryDir)
	}
}

func TestDefaultsIncludeTapes(t *testing.T) {
	cfg := Default()
	if cfg.NoTapes {
		t.Error("tapes should be enabled by default")
	}
}

func TestDefaultMaxRounds(t *testing.T) {
	cfg := Default()
	if cfg.MaxRounds != 1 {
		t.Errorf("expected default MaxRounds 1, got %d", cfg.MaxRounds)
	}
}

func TestDefaultStaleThreshold(t *testing.T) {
	cfg := Default()
	if cfg.StaleThreshold != 2 {
		t.Errorf("expected default StaleThreshold 2, got %d", cfg.StaleThreshold)
	}
}
