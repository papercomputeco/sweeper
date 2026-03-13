package linter

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type Issue struct {
	File    string
	Line    int
	Col     int
	Message string
	Linter  string
}

var issuePattern = regexp.MustCompile(`^(.+?):(\d+):(\d+):\s+(.+)\s+\((\w+)\)$`)

func ParseOutput(raw string) []Issue {
	var issues []Issue
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		m := issuePattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		lineNum, _ := strconv.Atoi(m[2])
		col, _ := strconv.Atoi(m[3])
		issues = append(issues, Issue{
			File:    m[1],
			Line:    lineNum,
			Col:     col,
			Message: m[4],
			Linter:  m[5],
		})
	}
	return issues
}

func Run(ctx context.Context, dir string) ([]Issue, error) {
	cmd := exec.CommandContext(ctx, "golangci-lint", "run", "--out-format=line-number", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return nil, fmt.Errorf("running golangci-lint: %w", err)
		}
	}
	return ParseOutput(string(out)), nil
}
