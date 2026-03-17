package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/papercomputeco/sweeper/pkg/dotdir"
)

// LoadTOML loads configuration with precedence:
//  1. Defaults
//  2. ~/.sweeper/config.toml (home)
//  3. .sweeper/config.toml (project, resolved via dotdir from targetDir)
//  4. Explicit configPath (if non-empty, replaces both file layers)
//  5. SWEEPER_* environment variables
//
// CLI flags are applied by the caller after LoadTOML returns.
func LoadTOML(targetDir, configPath string) (TOMLConfig, error) {
	tc := NewDefaultTOMLConfig()

	if configPath != "" {
		if err := decodeTOMLFile(configPath, &tc); err != nil {
			return tc, err
		}
		applyEnvOverrides(&tc)
		return tc, nil
	}

	// Layer 1: home config (~/.sweeper/config.toml)
	if home, err := os.UserHomeDir(); err == nil {
		homePath := filepath.Join(home, ".sweeper", "config.toml")
		_ = decodeTOMLFile(homePath, &tc) // ignore missing
	}

	// Layer 2: project config (.sweeper/config.toml via dotdir)
	if targetDir != "" {
		mgr := dotdir.NewManager()
		dir, err := mgr.TargetIn(targetDir, "")
		if err == nil && dir != "" {
			_ = decodeTOMLFile(filepath.Join(dir, "config.toml"), &tc)
		}
	}

	// Layer 3: env vars
	applyEnvOverrides(&tc)

	return tc, nil
}

func decodeTOMLFile(path string, tc *TOMLConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = toml.Decode(string(data), tc)
	return err
}
