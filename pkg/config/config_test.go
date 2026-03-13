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
