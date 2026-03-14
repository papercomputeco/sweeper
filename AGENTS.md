# AGENTS.md

### Don't

- Do not write design documents or implementation plans to disk (no `docs/plans/` or similar).
  Discuss plans in conversation only.

### Do

- Follow idiomatic Go and prefer using the `func NewExampleStruct() *ExampleStruct`
  paradigm seen throughout.
- Run `go test ./...` before committing.

### Project Overview

`sweeper` is an agent-powered code maintenance tool that dispatches parallel
Claude Code sub-agents to fix lint issues across a codebase, each running in
its own isolated environment.

**Language:** Go 1.25+
**Go Module:** `github.com/papercomputeco/sweeper`

### Project Structure

- `skills/` - Skill definitions for Claude Code and opencode integrations.
- `.claude-plugin/` - Claude Code plugin manifest and marketplace catalog.
- `.github/` - GitHub metadata and action workflows.

### Releases

Releases follow the same pipeline as [tapes](https://github.com/papercomputeco/tapes):

- **`cut-release.yaml`** — manual `workflow_dispatch` to create a new GitHub release (first release: `v0.1.0`).
- **`release.yaml`** — triggers on release publish, builds cross-platform binaries and uploads them as release artifacts.
- **`ci.yaml`** — builds binaries and runs cross-platform smoke tests on push to `main` and PRs.

### PR and Commit Conventions

PR titles are validated by CI and feed into auto-generated release notes.
Use this format:

```
<emoji> <type>: <Short description>
```

| Type    | Emoji | When to use                        |
|---------|-------|------------------------------------|
| `feat`  | ✨    | New functionality                  |
| `fix`   | 🐛    | Bug fix                            |
| `chore` | 🧹    | Maintenance, CI, deps, cleanup     |
| `perf`  | ⚡    | Performance improvement            |

Examples:
- `✨ feat: Add VM isolation for sub-agents`
- `🐛 fix: Correct retry strategy escalation`
- `🧹 chore: Update CI workflow to match tapes`

Squash-merge commits inherit the PR title, so the PR title **is** the commit
message that lands on `main`.
