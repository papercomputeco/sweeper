# Sweeper Skill

Autonomous lint-fix agent skill powered by the sweeper CLI. Orchestrates parallel sub-agents with VM isolation and tapes-driven learning.

## Install in Claude Code

Copy the skill into your project:

```bash
cp -r skills/sweeper/ /path/to/your-project/.claude/skills/sweeper/
```

Then tell Claude: "Run sweeper on this project using ESLint"

## Install in opencode

Copy the skill into your project's agents directory:

```bash
mkdir -p /path/to/your-project/.opencode/agents/
cp skills/sweeper/SKILL.md /path/to/your-project/.opencode/agents/sweeper.md
```

## Install in Pi

```bash
pi install path/to/sweeper
```

Then tell Pi: "Run sweeper on this project"

## Prerequisites

The `sweeper` binary must be in PATH:

```bash
cd /path/to/sweeper && go build -o sweeper .
export PATH="/path/to/sweeper:$PATH"
```

For tapes integration (token tracking):

```bash
go install github.com/papercomputeco/tapes/cli/tapes@latest
tapes init
```

## What It Does

1. Orchestrates `sweeper run` to dispatch parallel Claude Code sub-agents
2. Each sub-agent fixes a file's lint issues concurrently (bounded by `--concurrency`)
3. Retries with escalating strategies (standard -> retry -> exploration)
4. Optional VM isolation via stereOS for security and resource isolation
5. Tapes records every sub-agent session for token tracking and self-learning
6. `sweeper observe` shows success rates, strategy effectiveness, and token spend
7. Session state tracked in `sweeper.md` for resume across restarts

## Tapes — The Learning Center

Tapes is the self-learning backbone. Every sub-agent session is recorded, giving:

- Token usage per linter and strategy
- Success rate trends over time
- Round/strategy effectiveness to optimize future runs
- Token budget tracking to reduce spend over time

Run `sweeper observe` after each sweep to see insights and tune your next run.
