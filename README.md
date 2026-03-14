# 🧹 Sweeper Agent

Agent-powered code maintenance tool that automates lint fixes through parallel Claude Code sub-agents.

Sweeper runs your linter, groups issues by file, dispatches concurrent Claude Code agents to fix them, and records outcomes so it can learn from past runs. Inspired by [autoresearch](https://github.com/karpathy/autoresearch), it follows a read-decide-act-observe loop with RL-style prompt escalation and stagnation detection.

## Setup

### Claude Code

To use sweeper as a skill in Claude Code:

1. Build the binary:
```bash
go build -o sweeper .
export PATH="$PWD:$PATH"
```

2. Copy the skill into your project:
```bash
cp -r skills/sweeper/ /path/to/your-project/.claude/skills/sweeper/
```

3. Tell Claude: "Run sweeper on this project"

Claude will orchestrate `sweeper run` with the right flags based on your project.

### opencode

To use sweeper as a skill in opencode:

1. Build the binary:
```bash
go build -o sweeper .
export PATH="$PWD:$PATH"
```

2. Copy the skill into your project's agents directory:
```bash
mkdir -p /path/to/your-project/.opencode/agents/
cp skills/sweeper/SKILL.md /path/to/your-project/.opencode/agents/sweeper.md
```

3. Use the sweeper agent in opencode to run lint-fix loops.

### Pi

```bash
pi install sweeper
```

Provides `init_sweep`, `run_linter`, and `log_result` tools plus a dashboard widget.

### Go CLI (standalone)

```bash
go install github.com/papercomputeco/sweeper@latest
sweeper run                              # default: golangci-lint
sweeper run --vm -c 5 --max-rounds 3    # VM isolation, 5 agents, 3 rounds
sweeper run -- npm run lint              # any linter
sweeper observe                          # review success rates + token spend
```

## How It Works

1. **Lint** — run any linter, parse structured output
2. **Dispatch** — parallel Claude Code sub-agents fix each file concurrently
3. **Retry** — escalate strategy: standard -> retry -> exploration
4. **Record** — outcomes logged to `.sweeper/telemetry/` + tapes captures token usage
5. **Learn** — `sweeper observe` shows what works, what costs too much, what to tune
6. **Isolate** — optional stereOS VM for secrets, resources, and nesting safety

## Tapes — The Learning Center

Every sub-agent session is recorded in [tapes](https://github.com/papercomputeco/tapes). This gives you:

- **Token spend per linter** — know what each fix costs
- **Strategy effectiveness** — standard vs retry vs exploration success rates
- **Round effectiveness** — which retry rounds contribute most fixes
- **Trend tracking** — are you fixing more issues with fewer tokens over time?

Run `sweeper observe` after each sweep to see insights and tune your next run.

## VM Isolation

Use `--vm` to run sub-agents inside a stereOS virtual machine:

- **Secret isolation** — API keys stay in the VM
- **Resource isolation** — dedicated CPU/memory
- **No nesting conflicts** — works inside Claude Code sessions
- **Clean teardown** — ephemeral VMs destroyed on exit

## Session State

Session state lives in `sweeper.md` for resume across restarts. The CLI generates this automatically, and the skill uses it to track progress and token spend.
