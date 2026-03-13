# Sweeper

AI-powered code maintenance tool that automates lint fixes through parallel Claude Code sub-agents.

Sweeper runs your linter, groups issues by file, dispatches concurrent Claude Code agents to fix them, and records outcomes so it can learn from past runs.

## Approach

Sweeper follows a **read-decide-act-observe** loop:

1. **Read** - Run a lint command on the target codebase, parse structured issues
2. **Decide** - Group issues by file into fix tasks
3. **Act** - Dispatch parallel Claude Code sub-agents to apply fixes
4. **Observe** - Record outcomes to JSONL telemetry, extract success patterns per linter

Each sub-agent receives a focused prompt for a single file listing the exact lint issues (line numbers, messages, linter names) and is instructed to fix only those issues without changing behavior.

## Prerequisites

- **Go 1.25+**
- **Claude Code CLI** (`claude`) in PATH
- **golangci-lint** in PATH (only needed for default mode)
- **Tapes** (optional) - local session database at `~/.tapes/tapes.db` for token tracking

## Setup

```bash
# Clone and build
git clone https://github.com/papercomputeco/sweeper.git
cd sweeper
go build -o sweeper .

# Run tests
go test ./...
```

## Usage

```bash
# Fix lint issues in current directory using golangci-lint (default)
./sweeper run

# Target a specific directory with higher concurrency
./sweeper run --target /path/to/project --concurrency 5

# Use an arbitrary lint command
./sweeper run -- npm run lint
./sweeper run -- npx eslint --format unix .
./sweeper run -- cargo clippy 2>&1

# Pipe existing lint output
cat lint-results.txt | ./sweeper run
npm run lint 2>&1 | ./sweeper run

# Preview what would be fixed without running agents
./sweeper run --dry-run
./sweeper run --dry-run -- npm run lint

# Analyze past run outcomes
./sweeper observe

# Disable tapes integration
./sweeper run --no-tapes
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--target` | `-t` | `.` | Target directory to maintain |
| `--concurrency` | `-c` | `3` | Max parallel sub-agents |
| `--no-tapes` | | `false` | Disable tapes integration |
| `--dry-run` | | `false` | Show plan without executing (run only) |

### Input Modes

| Mode | Example | When to use |
|------|---------|-------------|
| Default | `sweeper run` | Go projects with golangci-lint |
| Custom command | `sweeper run -- npm run lint` | Any linter you can run as a command |
| Piped input | `npm run lint \| sweeper run` | Pre-existing lint output or CI pipelines |

### Output Parsing

Sweeper tries three regex patterns in order of specificity:

1. **golangci-lint**: `file:line:col: message (linter-name)`
2. **Generic**: `file:line:col: message`
3. **Minimal**: `file:line: message`

If no lines match any pattern, the full output is sent to a single agent for analysis (raw fallback). This preserves per-file parallelism for ~80% of linters and degrades gracefully for the rest.

## Architecture

```
┌─────────────────────────────────────────────────┐
│                   CLI (cmd/)                     │
│         root.go / run.go / observe.go            │
└──────────────────────┬──────────────────────────┘
                       │
              ┌────────▼────────┐
              │   Agent Loop    │
              │  pkg/agent/     │
              └──┬─────┬─────┬──┘
                 │     │     │
        ┌────────▼┐ ┌──▼──┐ ┌▼────────┐
        │ Linter  │ │Plan │ │ Worker  │
        │pkg/     │ │pkg/ │ │ Pool    │
        │linter/  │ │plan │ │pkg/     │
        │         │ │ner/ │ │worker/  │
        └─────────┘ └─────┘ └────┬────┘
                                  │
                          ┌───────▼───────┐
                          │ Claude Code   │
                          │ Executor      │
                          │ (claude CLI)  │
                          └───────────────┘
        ┌─────────┐  ┌──────────┐
        │Telemetry│  │ Observer │◄── reads telemetry
        │pkg/     │  │ pkg/     │    + tapes sessions
        │telemetry│  │observer/ │
        └─────────┘  └──────────┘
        ┌─────────┐
        │ Tapes   │  Optional SQLite
        │pkg/tapes│  session tracking
        └─────────┘
```

### Packages

| Package | Purpose |
|---------|---------|
| `cmd/` | CLI commands via Cobra |
| `pkg/agent/` | Orchestrates the lint-plan-fix-record loop |
| `pkg/linter/` | Runs lint commands, parses output into `Issue` structs via multi-format detection |
| `pkg/planner/` | Groups issues by file into `FixTask` slices |
| `pkg/worker/` | Bounded worker pool, task/result types, Claude executor |
| `pkg/telemetry/` | JSONL event writer to `.sweeper/telemetry/` |
| `pkg/observer/` | Reads telemetry, computes success rates per linter |
| `pkg/tapes/` | Detects and reads tapes SQLite DB for token usage |
| `pkg/config/` | Config struct with defaults |

### Data Flow

1. `linter.Run()` / `linter.RunCommand()` runs the lint command, parses output via multi-format regex
2. `planner.GroupByFile()` groups issues into per-file tasks (or single raw task for unparseable output)
3. `worker.NewPool()` runs tasks through a semaphore-bounded goroutine pool
4. `worker.ClaudeExecutor()` shells out to `claude --print --dangerously-skip-permissions` per task
5. `telemetry.Publisher` writes each result as a JSONL event
6. `observer.Analyze()` reads JSONL files and computes per-linter success rates, optionally enriched with tapes token data

### Telemetry

Events are written to `.sweeper/telemetry/YYYY-MM-DD.jsonl`. Each line is a JSON object with timestamp, event type, and data (file, success, duration, issue count, linter name, error).

### Tapes Integration

When enabled (default), sweeper checks for a tapes SQLite database and enriches observer insights with token usage data from recent sessions. Uses a pure-Go SQLite driver (`modernc.org/sqlite`) for zero-CGO cross-compilation.
