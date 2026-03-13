# Sweeper Agent Skill

You are operating the **sweeper** tool, an AI-powered lint fixer that dispatches parallel Claude Code sub-agents to fix lint issues from any linter.

## Quick Start

```bash
cd /path/to/target/project
sweeper run                              # default: golangci-lint
sweeper run -- npm run lint              # arbitrary command
npm run lint | sweeper run               # piped stdin
```

## Commands

### `sweeper run`

Runs the full lint-fix-retry loop:

1. Executes a lint command (default: `golangci-lint run --out-format=line-number ./...`)
2. Parses output using multi-format detection (golangci-lint, generic `file:line:col`, minimal `file:line`, or raw fallback)
3. Groups structured issues by file into parallel fix tasks
4. Selects prompt strategy based on round number and file history (standard → retry → exploration)
5. Dispatches parallel Claude Code sub-agents (default: 3) to fix each file
6. Records outcomes to `.sweeper/telemetry/` with round and strategy metadata
7. Re-lints to check remaining issues; repeats with escalated prompts (if `--max-rounds > 1`)

**Input modes:**
- `sweeper run` - Default: runs golangci-lint
- `sweeper run -- <command>` - Run an arbitrary lint command (e.g., `npm run lint`, `cargo clippy`)
- `<command> | sweeper run` - Pipe existing lint output via stdin

**Flags:**
- `--target, -t <dir>` - Directory to lint and fix (default: `.`)
- `--concurrency, -c <n>` - Max parallel sub-agents (default: `3`)
- `--dry-run` - Show what would be fixed without running agents
- `--no-tapes` - Disable tapes session tracking
- `--max-rounds <n>` - Maximum retry rounds (default: `1` = single pass)
- `--stale-threshold <n>` - Consecutive non-improving rounds before exploration mode (default: `2`)

**Example runs:**
```bash
# Fix current directory with golangci-lint (default)
sweeper run

# Fix a specific project with 5 agents
sweeper run -t /path/to/project -c 5

# Use ESLint
sweeper run -- npx eslint --format unix .

# Use cargo clippy
sweeper run -- cargo clippy 2>&1

# Pipe existing lint output
cat lint-results.txt | sweeper run

# Preview fixes
sweeper run --dry-run
sweeper run --dry-run -- npm run lint

# Retry loop: re-lint after each round, escalate prompt strategy
sweeper run --max-rounds 3
sweeper run --max-rounds 5 --stale-threshold 3
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

1. **claude** - Claude Code CLI must be in PATH. The tool invokes `claude --print --dangerously-skip-permissions <prompt>` for each fix task.
2. **golangci-lint** (only for default mode) - Must be in PATH. Install: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
3. **tapes** (optional) - If `~/.tapes/tapes.db` exists, sweeper tracks token usage per session.

When using `-- <command>` or piped input, golangci-lint is not required.

## Building from Source

```bash
cd /path/to/sweeper
go build -o sweeper .
```

The binary has no CGO dependencies (uses pure-Go SQLite) and cross-compiles cleanly.

## How It Works

### Structured output (parsed)

When lint output matches a recognized format (`file:line:col: message`), each sub-agent receives a focused prompt for a single file:

```
Fix the following lint issues in path/to/file.go:

- Line 12: exported function Foo should have comment (golint)
- Line 45: unnecessary conversion (unconvert)

Fix each issue. Do not change behavior. Only fix lint issues. Commit nothing.
```

Multiple agents run concurrently across different files.

### Retry loop (RL-inspired)

When `--max-rounds > 1`, sweeper re-lints after each round and retries files with remaining issues. The prompt strategy escalates:

- **Round 1 (standard)**: Normal fix prompt with issue list
- **Round 2+ (retry)**: Includes prior attempt output, instructs agent to try a different approach
- **After stagnation (exploration)**: WARNING directive, instructs agent to refactor surrounding code

Stagnation is detected after `--stale-threshold` consecutive rounds with zero improvement on a file. After exploration is attempted and fails, the file is dropped from further retries.

Telemetry events include `round` and `strategy` fields, enabling `sweeper observe` to show which rounds and strategies are most effective across runs.

### Raw output (fallback)

When output cannot be parsed into structured issues, the full output is sent to a single agent for analysis:

```
The following lint output was produced. Analyze it, identify the issues, and fix them:

<full lint output>

Fix each issue you can identify. Do not change behavior. Only fix lint issues. Commit nothing.
```

## Telemetry

Results are stored in `.sweeper/telemetry/YYYY-MM-DD.jsonl` relative to the target directory.

Event types:
- **fix_attempt**: Per-file fix result with file, success, duration, issue count, linter, round number, and prompt strategy
- **round_complete**: Per-round summary with task count, fixed count, and failed count

Use `sweeper observe` to analyze this data. It shows success rates per linter and, when multi-round telemetry exists, round effectiveness and strategy effectiveness trends.

## Troubleshooting

- **"golangci-lint: command not found"** - Install golangci-lint or use `-- <command>` to specify a different linter
- **"claude: command not found"** - Install Claude Code CLI or add it to PATH
- **"cannot use both piped input and -- command"** - Choose one input method: pipe or `--`
- **"No lint issues found"** - The target codebase is clean; nothing to fix
- **Custom command produces no parseable output** - Sweeper falls back to raw mode; the agent will analyze the full output
- **Tapes warning** - Tapes is optional; use `--no-tapes` to suppress the warning
- **Tasks failing** - Check the sub-agent output in the telemetry JSONL for error details
