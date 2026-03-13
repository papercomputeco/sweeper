package tapes

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Status struct {
	Available bool
	DBPath    string
	Message   string
}

// FindDB searches for the tapes SQLite database in standard locations.
func FindDB(projectDir string) string {
	candidates := []string{
		filepath.Join(projectDir, ".tapes", "tapes.db"),
		filepath.Join(projectDir, ".tapes", "tapes.sqlite"),
	}

	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".tapes", "tapes.db"),
			filepath.Join(home, ".tapes", "tapes.sqlite"),
		)
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// CheckInstallation returns the tapes availability status.
func CheckInstallation(dbPath string) Status {
	if dbPath == "" {
		dbPath = FindDB(".")
	}

	if dbPath != "" {
		return Status{Available: true, DBPath: dbPath}
	}

	if _, err := exec.LookPath("tapes"); err == nil {
		return Status{
			Available: false,
			Message:   "tapes CLI found but no database. Run `tapes init` to set up local tapes.",
		}
	}

	return Status{
		Available: false,
		Message: fmt.Sprintf("tapes is not installed. Install it to track sub-agent sessions:\n" +
			"  go install github.com/papercomputeco/tapes/cli/tapes@latest\n" +
			"  tapes init\n"),
	}
}
