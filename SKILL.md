# Sweeper Agent Skill

You are operating the **sweeper** tool, an AI-powered lint fixer that dispatches parallel Claude Code sub-agents to fix golangci-lint issues.

## Quick Start

```bash
cd /path/to/target/project
sweeper run
```

## Commands

### `sweeper run`

Runs the full lint-fix loop:

1. Executes `golangci-lint run --out-format=line-number ./...` on the target directory
2. Parses issues and groups them by file
3. Dispatches parallel Claude Code sub-agents (default: 3) to fix each file
4. Records outcomes to `.sweeper/telemetry/`

**Flags:**
- `--target, -t <dir>` - Directory to lint and fix (default: `.`)
- `--concurrency, -c <n>` - Max parallel sub-agents (default: `3`)
- `--dry-run` - Show what would be fixed without running agents
- `--no-tapes` - Disable tapes session tracking

**Example runs:**
```bash
# Fix current directory
sweeper run

# Fix a specific project with 5 agents
sweeper run -t /path/to/project -c 5

# Preview fixes
sweeper run --dry-run
```

**Exit codes:**
- `0` - All tasks succeeded (or no issues found)
- `1` - One or more tasks failed

### `sweeper observe`

Analyzes past run telemetry and shows success rates per linter:

```bash
sweeper observe
sweeper observe --target /path/to/project
```

Output shows: linter name, attempt count, successes, success rate percentage, and token usage (if tapes is available).

### `sweeper version`

Prints the current version.

## Prerequisites

Before running sweeper, ensure these are available:

1. **golangci-lint** - Must be in PATH. Install: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
2. **claude** - Claude Code CLI must be in PATH. The tool invokes `claude --print --dangerously-skip-permissions <prompt>` for each fix task.
3. **tapes** (optional) - If `~/.tapes/tapes.db` exists, sweeper tracks token usage per session.

## Building from Source

```bash
cd /path/to/sweeper
go build -o sweeper .
```

The binary has no CGO dependencies (uses pure-Go SQLite) and cross-compiles cleanly.

## How It Works

Each sub-agent receives a prompt like:

```
Fix the following lint issues in path/to/file.go:

- Line 12: exported function Foo should have comment (golint)
- Line 45: unnecessary conversion (unconvert)

Fix each issue. Do not change behavior. Only fix lint issues. Commit nothing.
```

The agent fixes only the listed issues in that file. Multiple agents run concurrently across different files.

## Telemetry

Results are stored in `.sweeper/telemetry/YYYY-MM-DD.jsonl` relative to the target directory. Each line records: timestamp, file, success/failure, duration, issue count, and any error message.

Use `sweeper observe` to analyze this data.

## Troubleshooting

- **"golangci-lint: command not found"** - Install golangci-lint or add it to PATH
- **"claude: command not found"** - Install Claude Code CLI or add it to PATH
- **"No lint issues found"** - The target codebase is clean; nothing to fix
- **Tapes warning** - Tapes is optional; use `--no-tapes` to suppress the warning
- **Tasks failing** - Check the sub-agent output in the telemetry JSONL for error details
