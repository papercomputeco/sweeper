# Sweeper Telemetry Format

All sweeper implementations (Go CLI, Claude Code skill, Pi extension) produce the same JSONL event format in `.sweeper/telemetry/`.

## Events

### init

Emitted once per session start.

```json
{
  "timestamp": "2026-03-14T10:00:00Z",
  "type": "init",
  "data": {
    "name": "session-name",
    "linterCommand": "golangci-lint run --out-format=line-number ./...",
    "targetDir": ".",
    "maxRounds": 3,
    "staleThreshold": 2
  }
}
```

### fix_attempt

Emitted per file per round.

```json
{
  "timestamp": "2026-03-14T10:01:00Z",
  "type": "fix_attempt",
  "data": {
    "file": "server.go",
    "success": true,
    "round": 1,
    "strategy": "standard",
    "issues_before": 3,
    "issues_after": 0,
    "linter": "golangci-lint",
    "duration": "2.3s"
  }
}
```

### round_complete

Emitted after all files in a round are processed.

```json
{
  "timestamp": "2026-03-14T10:02:00Z",
  "type": "round_complete",
  "data": {
    "round": 1,
    "linter": "golangci-lint",
    "tasks": 5,
    "fixed": 4,
    "failed": 1
  }
}
```

## File Location

All implementations write to `.sweeper/telemetry/YYYY-MM-DD.jsonl` (date-named files, append-only).

All implementations read all `*.jsonl` files in the directory for analysis and session resume.

## Tapes Integration

Tapes is the primary source for token usage data. The JSONL telemetry tracks fix outcomes (success/failure, strategy, round). Tapes tracks the cost (prompt tokens, completion tokens, session duration).

`sweeper observe` joins both data sources:
1. Read `.sweeper/telemetry/*.jsonl` for fix attempt outcomes
2. Query `tapes.db` for recent session token counts
3. Proportionally allocate tokens across linters based on attempt counts
4. Report combined insights: success rates + token spend

This separation keeps telemetry lightweight (no token counting in the hot path) while tapes provides the authoritative cost data.
