# 🧹 Sweeper

AI-powered code maintenance tool that automates lint fixes through parallel Claude Code sub-agents.

Sweeper runs your linter, groups issues by file, dispatches concurrent Claude Code agents to fix them, and records outcomes so it can learn from past runs.

## Approach

Sweeper follows a **read-decide-act-observe** loop inspired by reinforcement learning:

1. **Read** - Run a lint command on the target codebase, parse structured issues
2. **Decide** - Group issues by file into fix tasks, select prompt strategy based on history
3. **Act** - Dispatch parallel Claude Code sub-agents to apply fixes
4. **Observe** - Record outcomes to JSONL telemetry, extract success patterns per linter
5. **Retry** - Re-lint to check remaining issues, escalate prompt strategy, repeat

Each sub-agent receives a focused prompt for a single file. The prompt strategy escalates across retry rounds:

- **Standard** (round 1): Exact lint issues with line numbers and fix instructions
- **Retry** (round 2+): Includes prior attempt output with directive to try a different approach
- **Exploration** (after stagnation): WARNING directive to refactor surrounding code

Stagnation detection fires after consecutive non-improving rounds, triggering the exploration strategy. This mirrors the bounds/history/stagnation pattern from AlphaEvolve-style evolution loops.

## Prerequisites

- **Go 1.25+**
- **Claude Code CLI** (`claude`) in PATH
- **golangci-lint** in PATH (only needed for default mode)
- **Tapes** (optional) - local session database at `~/.tapes/tapes.db` for token tracking
- **mb** (optional) - Masterblaster CLI for stereOS VMs, required only with `--vm`

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

# Retry loop: re-lint after each round, escalate prompt strategy
./sweeper run --max-rounds 3
./sweeper run --max-rounds 5 --stale-threshold 3

# Run inside a stereOS VM (secret isolation, no nesting conflicts)
./sweeper run --vm -- npx eslint --quiet .
./sweeper run --vm --max-rounds 3 -c 5 -- npx eslint --quiet .

# Use an existing VM (skip boot/teardown)
./sweeper run --vm-name my-vm -- npx eslint --quiet .

# Analyze past run outcomes and historical trends
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
| `--max-rounds` | | `1` | Maximum retry rounds (1 = single pass) |
| `--stale-threshold` | | `2` | Consecutive non-improving rounds before exploration |
| `--vm` | | `false` | Boot ephemeral stereOS VM, teardown on exit |
| `--vm-name` | | | Use existing VM by name (no managed lifecycle) |
| `--vm-jcard` | | | Custom jcard.toml path (implies `--vm`) |

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
                          │  Executor     │
                          │ Local (claude)│
                          │ or VM (mb ssh)│
                          └──────┬────────┘
                                 │
                          ┌──────▼────────┐
                          │ stereOS VM    │
                          │ pkg/vm/       │
                          │ (optional)    │
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
| `pkg/agent/` | Orchestrates the lint-plan-fix-retry loop with prompt escalation |
| `pkg/loop/` | Shared types for retry loop: Strategy enum, FileHistory, stagnation detection |
| `pkg/linter/` | Runs lint commands, parses output into `Issue` structs via multi-format detection |
| `pkg/planner/` | Groups issues by file into `FixTask` slices |
| `pkg/worker/` | Bounded worker pool, task/result types, Claude executor, VM executor |
| `pkg/vm/` | stereOS VM lifecycle: boot, exec via SSH, shutdown, jcard generation |
| `pkg/telemetry/` | JSONL event writer to `.sweeper/telemetry/` |
| `pkg/observer/` | Reads telemetry, computes success rates per linter, historical trends |
| `pkg/tapes/` | Detects and reads tapes SQLite DB for token usage |
| `pkg/config/` | Config struct with defaults |

### Data Flow

1. `linter.Run()` / `linter.RunCommand()` runs the lint command, parses output via multi-format regex
2. `planner.GroupByFile()` groups issues into per-file tasks (or single raw task for unparseable output)
3. `worker.NewPool()` runs tasks through a semaphore-bounded goroutine pool
4. `worker.ClaudeExecutor()` shells out to `claude --print` locally, or `worker.NewVMExecutor()` runs it inside a stereOS VM via `mb ssh`
5. `telemetry.Publisher` writes each result as a JSONL event
6. `observer.Analyze()` reads JSONL files and computes per-linter success rates, optionally enriched with tapes token data

### Telemetry

Events are written to `.sweeper/telemetry/YYYY-MM-DD.jsonl`. Each line is a JSON object with timestamp, event type, and data.

Event types:
- **fix_attempt**: Per-file fix result with file, success, duration, issue count, linter, round number, and prompt strategy
- **round_complete**: Per-round summary with task count, fixed count, and failed count

### Tapes Integration

When enabled (default), sweeper checks for a tapes SQLite database and enriches observer insights with token usage data from recent sessions. Uses a pure-Go SQLite driver (`modernc.org/sqlite`) for zero-CGO cross-compilation.
