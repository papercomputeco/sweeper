# 🧹 Sweeper Agent

Multi-threaded code maintenance with resource-isolated sub-agents.

Sweeper dispatches parallel Claude Code agents to fix lint issues across your codebase, each running in its own isolated environment. It groups issues by file, fans out concurrent fixes, escalates strategy when fixes stall, and records outcomes so it learns what works. With VM isolation enabled, each sub-agent runs inside a dedicated stereOS virtual machine with its own CPU, memory, and secrets boundary, safe to scale to 10+ concurrent agents.

```
                        sweeper run --vm -c 10
                              │
                    ┌─────────┼─────────┐
                    ▼         ▼         ▼
              ┌──────────────────────────────┐
              │        Worker Pool           │
              │   (semaphore-bounded, N=10)  │
              └──┬───┬───┬───┬───┬───┬──────┘
                 │   │   │   │   │   │
                 ▼   ▼   ▼   ▼   ▼   ▼
               ┌───┐┌───┐┌───┐┌───┐┌───┐┌───┐
               │VM ││VM ││VM ││VM ││VM ││VM │  ◄── stereOS isolation
               │ 1 ││ 2 ││ 3 ││ 4 ││ 5 ││...│      (secrets, CPU, memory)
               └─┬─┘└─┬─┘└─┬─┘└─┬─┘└─┬─┘└─┬─┘
                 │     │     │     │     │     │
                 ▼     ▼     ▼     ▼     ▼     ▼
              claude  claude claude claude claude claude
              --print --print --print --print --print --print
                 │     │     │     │     │     │
                 └─────┴─────┴──┬──┴─────┴─────┘
                                │
                    ┌───────────┼───────────┐
                    ▼           ▼           ▼
               streaming    telemetry    tapes
               progress     (.jsonl)    (SQLite)
```

Each sub-agent works on a single file. Results stream back as they complete, giving real-time progress instead of blocking until the entire round finishes.

## Why Sub-Agents

The main thread never reads or edits source files. It runs the linter, builds prompts, dispatches work, and collects results. All file-level reasoning happens inside sub-agents via `claude --print`, which are stateless, single-shot processes.

This matters because the orchestrator's context window stays small and predictable. It holds lint output, task metadata, and result summaries, not the contents of every file being fixed. A run that touches 50 files uses roughly the same orchestrator context as one that touches 5. The complexity scales in parallelism, not in context size.

```
  Orchestrator (main thread)              Sub-agents (disposable)
  ┌────────────────────────┐
  │ lint output            │              ┌──────────────────────┐
  │ file groupings         │  ──dispatch──▶ claude --print       │
  │ strategy decisions     │              │  reads auth.go       │
  │ result summaries       │  ◀──result── │  writes fix          │
  │                        │              └──────────────────────┘
  │ (never sees file       │              ┌──────────────────────┐
  │  contents directly)    │  ──dispatch──▶ claude --print       │
  │                        │              │  reads router.go     │
  │                        │  ◀──result── │  writes fix          │
  └────────────────────────┘              └──────────────────────┘
```

Sub-agents are fire-and-forget. Each one gets a prompt with the lint issues for its file, does the work, and exits. If it fails, the orchestrator knows from the exit code and can retry with an escalated strategy on the next round. No conversation state carries over between rounds, which keeps each attempt clean.

## Setup

### Go CLI (standalone)

The core binary. All integrations below (except Pi) require this.

```bash
go install github.com/papercomputeco/sweeper@latest
sweeper run                              # default: golangci-lint
sweeper run --vm -c 5 --max-rounds 3    # VM isolation, 5 agents, 3 rounds
sweeper run -- npm run lint              # any linter
sweeper observe                          # review success rates + token spend
```

### Claude Code

To use sweeper as a skill in [Claude Code](https://docs.anthropic.com/en/docs/claude-code):

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

To use sweeper as a skill in [opencode](https://opencode.ai) (a terminal-based AI coding agent):

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

3. Tell opencode: "Run sweeper on this project"

### Pi

[Pi](https://github.com/anthropics/pi) is a Claude-native IDE extension. Its sweeper integration reimplements the linting and telemetry loop in TypeScript using Pi's own tool system, so it does **not** need the Go binary.

```bash
pi install sweeper
```

This gives you `init_sweep`, `run_linter`, and `log_result` tools plus a dashboard widget. To start a sweep, tell Pi: "Sweep this project for lint issues"

## How It Works

This describes the Go CLI and skill-based integrations (Claude Code, opencode). Pi manages its own lint-fix loop through built-in tools and does not use the CLI.

1. **Lint**: run any linter, parse structured output (or fall back to raw mode)
2. **Plan**: group issues by file, pick strategy per file based on history
3. **Dispatch**: fan out to N concurrent sub-agents (default 5, up to 10+ with VMs)
4. **Stream**: results arrive in real time as each file completes
5. **Escalate**: stalled files get retry prompts, then exploration prompts that consider surrounding code
6. **Record**: outcomes logged to `.sweeper/telemetry/` and tapes captures token usage
7. **Learn**: `sweeper observe` shows success rates by strategy, round, and linter

## Tapes: The Learning Center

Every sub-agent session is recorded in [tapes](https://github.com/papercomputeco/tapes). This gives you:

- **Token spend per linter**: know what each fix costs
- **Strategy effectiveness**: standard vs retry vs exploration success rates
- **Round effectiveness**: which retry rounds contribute most fixes
- **Trend tracking**: are you fixing more issues with fewer tokens over time?

Run `sweeper observe` after each sweep to see insights and tune your next run.

## VM Isolation

Sub-agents can run inside ephemeral [stereOS](https://stereos.ai) virtual machines, managed by the `mb` (Masterblaster) CLI. This is what makes high concurrency safe.

Without VMs, sub-agents share the host process, filesystem, and API keys. At low concurrency (5 or fewer) this works fine. At higher concurrency, you want each agent isolated so a runaway process or leaked credential stays contained.

With `--vm`, each sub-agent gets:

- **Own CPU and memory**: 4 cores, 8GB RAM per VM (configurable). No resource contention between agents.
- **Secret boundary**: `ANTHROPIC_API_KEY` is injected into the VM and never touches the host filesystem.
- **Nesting safety**: `claude --print` fails inside active Claude Code sessions due to nesting detection. VMs sidestep this entirely.
- **Clean teardown**: VMs are ephemeral. On exit (success, failure, or SIGINT), the VM is destroyed automatically.

```bash
sweeper run --vm -c 10 --max-rounds 3    # 10 isolated agents, 3 retry rounds
```

## Session State

Session state lives in `sweeper.md` for resume across restarts. The CLI generates this automatically, and the skill uses it to track progress and token spend.
