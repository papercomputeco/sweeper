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

type ParseResult struct {
	Issues    []Issue
	RawOutput string
	Parsed    bool
}

var (
	// golangci-lint format: file:line:col: message (linter)
	golangciPattern = regexp.MustCompile(`^(.+?):(\d+):(\d+):\s+(.+)\s+\((\w[\w-]*)\)$`)
	// generic file:line:col: message
	genericPattern = regexp.MustCompile(`^(.+?):(\d+):(\d+):\s+(.+)$`)
	// minimal file:line: message
	minimalPattern = regexp.MustCompile(`^(.+?):(\d+):\s+(.+)$`)
)

func ParseOutput(raw string) ParseResult {
	result := ParseResult{RawOutput: raw}
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if iss, ok := parseGolangci(line); ok {
			result.Issues = append(result.Issues, iss)
			continue
		}
		if iss, ok := parseGeneric(line); ok {
			result.Issues = append(result.Issues, iss)
			continue
		}
		if iss, ok := parseMinimal(line); ok {
			result.Issues = append(result.Issues, iss)
			continue
		}
	}
	result.Parsed = len(result.Issues) > 0
	return result
}

func parseGolangci(line string) (Issue, bool) {
	m := golangciPattern.FindStringSubmatch(line)
	if m == nil {
		return Issue{}, false
	}
	lineNum, _ := strconv.Atoi(m[2])
	col, _ := strconv.Atoi(m[3])
	return Issue{
		File:    m[1],
		Line:    lineNum,
		Col:     col,
		Message: m[4],
		Linter:  m[5],
	}, true
}

func parseGeneric(line string) (Issue, bool) {
	m := genericPattern.FindStringSubmatch(line)
	if m == nil {
		return Issue{}, false
	}
	lineNum, _ := strconv.Atoi(m[2])
	col, _ := strconv.Atoi(m[3])
	return Issue{
		File:    m[1],
		Line:    lineNum,
		Col:     col,
		Message: m[4],
		Linter:  "custom",
	}, true
}

func parseMinimal(line string) (Issue, bool) {
	m := minimalPattern.FindStringSubmatch(line)
	if m == nil {
		return Issue{}, false
	}
	lineNum, _ := strconv.Atoi(m[2])
	return Issue{
		File:    m[1],
		Line:    lineNum,
		Message: m[3],
		Linter:  "custom",
	}, true
}

func RunCommand(ctx context.Context, dir string, cmd []string) (ParseResult, error) {
	c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return ParseResult{}, fmt.Errorf("running %s: %w", cmd[0], err)
		}
	}
	return ParseOutput(string(out)), nil
}

func Run(ctx context.Context, dir string) (ParseResult, error) {
	return RunCommand(ctx, dir, []string{"golangci-lint", "run", "--out-format=line-number", "./..."})
}
