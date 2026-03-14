---
name: sweeper
description: Autonomous lint-fix loop with parallel sub-agents, VM isolation, and tapes-driven learning. Orchestrates sweeper CLI to dispatch concurrent fixers, track token spend, and learn from outcomes.
---

# Sweeper - Autonomous Lint Fixer

You orchestrate the **sweeper** CLI to fix lint issues across a codebase using parallel Claude Code sub-agents with optional VM isolation. Tapes records every sub-agent session, enabling you to learn from past runs and optimize token spend.

## Prerequisites

The `sweeper` binary must be in PATH. Build it if needed:

```bash
cd /path/to/sweeper && go build -o sweeper . && export PATH="$PWD:$PATH"
```

## Setup

If no `sweeper.md` exists in the working directory, gather this information:

1. **Target directory** - Which directory to lint (default: `.`)
2. **Lint command** - What linter to run (default: `golangci-lint run --out-format=line-number ./...`)
3. **Concurrency** - How many parallel sub-agents (default: `3`)
4. **Max rounds** - Retry rounds before stopping (default: `3`)
5. **VM mode** - Whether to isolate sub-agents in a stereOS VM (recommended for 5+ agents or sensitive repos)
6. **Constraints** - Files/directories off-limits, behavioral invariants to preserve

Then create a session document:

### `sweeper.md` - Living Session Document

```md
# Sweeper Session

**Started:** <ISO timestamp>
**Objective:** Fix all lint issues in `<target>`
**Linter:** `<command>`
**Concurrency:** <n> sub-agents
**Max rounds:** <n>
**VM:** <yes/no>
**Constraints:** <constraints>

## Status
- Round: 0
- Issues found: (pending first run)
- Issues fixed: 0
- Token spend: (pending — check tapes after first run)

## What's Been Tried
(Updated after each round)

## Token Budget
(Updated via `sweeper observe` after each run)
```

Commit on a new branch: `sweeper/<goal>-<date>`

## Running Sweeper

Use the CLI to orchestrate the full loop. The CLI handles linting, parsing, parallel sub-agent dispatch, retry escalation, telemetry, and tapes integration.

### Basic runs

```bash
# Default: golangci-lint with 3 parallel agents
sweeper run

# Custom linter
sweeper run -- npm run lint

# Multi-round with escalation
sweeper run --max-rounds 3

# High concurrency in VM isolation
sweeper run --vm -c 5 --max-rounds 3 -- npx eslint --quiet .

# Preview what would be fixed
sweeper run --dry-run
```

### VM isolation (recommended for production)

```bash
# Ephemeral VM — boots before sweep, tears down after
sweeper run --vm -- npx eslint --quiet .

# Reuse existing VM
sweeper run --vm-name my-vm -- cargo clippy 2>&1
```

Use `--vm` when:
- Running inside a Claude Code session (avoids nesting conflicts)
- Working with sensitive API keys (secrets stay in VM)
- High concurrency (dedicated resources)
- CI/CD (hermetic environment)

## How the CLI Orchestrates

Each `sweeper run` executes this loop:

1. **Lint**: Run linter command, parse structured output
2. **Group**: Issues grouped by file into parallel fix tasks
3. **Strategy**: Pick prompt strategy per file based on round + history:
   - **Round 0** (no prior history): `standard` — straightforward fix
   - **Round 1+** (any prior history): `retry` — different approach
   - **Consecutive stale >= threshold**: `exploration` — refactor surrounding code
   - **Stagnant after exploration**: file dropped
4. **Dispatch**: Parallel sub-agents fix each file (bounded by `--concurrency`)
5. **Record**: Each outcome logged to `.sweeper/telemetry/` JSONL + tapes captures token usage
6. **Re-lint**: Verify fixes, filter retryable issues
7. **Repeat or stop**: Continue if issues remain and rounds left

## Tapes — The Learning Center

Tapes is the backbone for self-learning. Every sub-agent session is recorded in tapes, giving you token usage, session transcripts, and outcome data.

### Check tapes status

```bash
sweeper observe
```

This shows:
- **Success rate per linter** — which linters sweeper handles best
- **Round effectiveness** — which retry rounds contribute most fixes
- **Strategy effectiveness** — standard vs retry vs exploration success rates
- **Token usage per linter** — how much each linter costs to fix (from tapes)

### Use tapes data to make decisions

Before starting a sweep, check historical performance:

```bash
sweeper observe --target /path/to/project
```

Use the insights to tune your run:
- If round 1 fixes 90% of issues, `--max-rounds 1` saves tokens
- If exploration strategy has <10% success, lower `--stale-threshold` to skip stagnant files faster
- If a specific linter has low success rate, consider excluding those rules
- Compare token spend across runs to track improvement over time

### Token budget tracking

After each run, update `sweeper.md` with token spend from tapes:

```
## Token Budget
- Run 1: 45,230 prompt + 12,100 completion = 57,330 total
- Run 2: 31,000 prompt + 8,200 completion = 39,200 total (31% reduction)
- Trend: improving — retry prompts getting more targeted
```

The goal is to fix more issues with fewer tokens over time. Tapes makes this measurable.

## Resume

If `sweeper.md` already exists when you start:
1. Read it to understand what's been tried and token spend so far
2. Run `sweeper observe` to check recent success patterns
3. Read git log for recent sweeper commits
4. Choose `--max-rounds` and `--concurrency` based on tapes insights
5. Continue from where you left off

## Updating sweeper.md

After each `sweeper run` completes, update the session document:
1. Record round results (issues found/fixed/remaining)
2. Run `sweeper observe` and record token spend
3. Note which strategies worked and which files are stagnant
4. Commit the updated `sweeper.md`
