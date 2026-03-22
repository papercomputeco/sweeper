package dotdir

import (
	"os"
	"path/filepath"
)

const dirName = ".sweeper"

type Manager struct{}

func NewManager() *Manager { return &Manager{} }

// Target resolves the .sweeper/ directory.
// Precedence: override > ./.sweeper/ > ~/.sweeper/ > ""
func (m *Manager) Target(overrideDir string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return m.TargetWithHome(cwd, home, overrideDir)
}

// TargetIn resolves using a specific base directory instead of cwd.
func (m *Manager) TargetIn(baseDir, overrideDir string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return m.TargetWithHome(baseDir, home, overrideDir)
}

// TargetWithHome is the testable core: explicit cwd and home.
func (m *Manager) TargetWithHome(cwd, home, overrideDir string) (string, error) {
	if overrideDir != "" {
		return overrideDir, nil
	}
	local := filepath.Join(cwd, dirName)
	if info, err := os.Stat(local); err == nil && info.IsDir() {
		return local, nil
	}
	if home != "" {
		homeDir := filepath.Join(home, dirName)
		if info, err := os.Stat(homeDir); err == nil && info.IsDir() {
			return homeDir, nil
		}
	}
	return "", nil
}
