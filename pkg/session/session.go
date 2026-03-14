package session

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Objective   string
	LintCommand string
	TargetDir   string
	MaxRounds   int
	Constraints string
}

func Generate(dir string, cfg Config) (string, error) {
	path := filepath.Join(dir, "sweeper.md")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	content := fmt.Sprintf("# Sweeper Session\n\n**Started:** %s\n**Objective:** %s\n**Linter:** `%s`\n**Target:** %s\n**Max rounds:** %d\n**Constraints:** %s\n\n## Status\n- Round: 0\n- Issues found: (pending first run)\n- Issues fixed: 0\n- Files remaining: (pending)\n\n## What's Been Tried\n(Updated after each round)\n",
		time.Now().Format(time.RFC3339), cfg.Objective, cfg.LintCommand, cfg.TargetDir, cfg.MaxRounds, cfg.Constraints)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func UpdateStatus(path string, round, found, fixed, remaining int) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	entry := fmt.Sprintf("\n### Round %d (%s)\n- Issues at start: %d\n- Fixed: %d\n- Remaining: %d\n",
		round, time.Now().Format("15:04:05"), found, fixed, remaining)
	_, err = f.WriteString(entry)
	return err
}
