import { Type } from "@sinclair/typebox";

// --- State types ---

interface SweepResult {
  run: string;
  file: string;
  round: number;
  strategy: string;
  status: "fixed" | "failed";
  issuesBefore: number;
  issuesAfter: number;
}

interface SweepState {
  name: string;
  linterCommand: string;
  targetDir: string;
  maxRounds: number;
  staleThreshold: number;
  results: SweepResult[];
  currentRound: number;
  totalFixed: number;
  totalFailed: number;
}

// --- Helpers ---

function todayFileName(): string {
  const d = new Date();
  const yyyy = d.getFullYear();
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${yyyy}-${mm}-${dd}.jsonl`;
}

function telemetryDir(targetDir: string): string {
  const path = require("path");
  return path.join(targetDir, ".sweeper", "telemetry");
}

function reconstructStateFromDir(dir: string): SweepState | null {
  const fs = require("fs");
  const path = require("path");

  if (!fs.existsSync(dir)) return null;

  const files: string[] = fs
    .readdirSync(dir)
    .filter((f: string) => f.endsWith(".jsonl"))
    .sort();

  if (files.length === 0) return null;

  let state: SweepState | null = null;

  for (const file of files) {
    const content = fs.readFileSync(path.join(dir, file), "utf-8");
    const lines = content.split("\n").filter((l: string) => l.trim());

    for (const line of lines) {
      let event: any;
      try {
        event = JSON.parse(line);
      } catch {
        continue;
      }

      if (event.type === "init") {
        state = {
          name: event.data?.name ?? "sweeper",
          linterCommand: event.data?.linterCommand ?? "golangci-lint run --out-format=line-number ./...",
          targetDir: event.data?.targetDir ?? ".",
          maxRounds: event.data?.maxRounds ?? 1,
          staleThreshold: event.data?.staleThreshold ?? 2,
          results: [],
          currentRound: 1,
          totalFixed: 0,
          totalFailed: 0,
        };
      }

      if (event.type === "fix_attempt" && state) {
        const success = !!event.data?.success;
        const result: SweepResult = {
          run: state.name,
          file: event.data?.file ?? "unknown",
          round: event.data?.round ?? 1,
          strategy: event.data?.strategy ?? "standard",
          status: success ? "fixed" : "failed",
          issuesBefore: event.data?.issuesBefore ?? event.data?.issues ?? 0,
          issuesAfter: event.data?.issuesAfter ?? 0,
        };
        state.results.push(result);
        state.currentRound = Math.max(state.currentRound, result.round);
        if (success) {
          state.totalFixed++;
        } else {
          state.totalFailed++;
        }
      }
    }
  }

  return state;
}

// --- Extension ---

export default function sweeper({ tool, widget }: any) {
  let state: SweepState | null = null;

  // ---- Tool: init_sweep ----
  tool({
    name: "init_sweep",
    description:
      "Configure a sweeper session. Sets the linter command, target directory, max rounds, and stale threshold. Tries to resume from existing telemetry first.",
    parameters: Type.Object({
      name: Type.String({ description: "Session name for this sweep run" }),
      linterCommand: Type.String({
        description: "Shell command to run the linter (e.g. 'golangci-lint run --out-format=line-number ./...')",
      }),
      targetDir: Type.Optional(
        Type.String({ description: "Target directory to lint (default: '.')" })
      ),
      maxRounds: Type.Optional(
        Type.Number({ description: "Maximum retry rounds (default: 1)" })
      ),
      staleThreshold: Type.Optional(
        Type.Number({
          description: "Consecutive non-improving rounds before exploration (default: 2)",
        })
      ),
    }),
    execute: async (params: any) => {
      const fs = require("fs");
      const path = require("path");

      const targetDir = params.targetDir ?? ".";
      const tDir = telemetryDir(targetDir);

      // Try to resume from existing telemetry
      const existing = reconstructStateFromDir(tDir);
      if (existing && existing.name === params.name) {
        state = existing;
        // Update config fields in case they changed
        state.linterCommand = params.linterCommand;
        state.maxRounds = params.maxRounds ?? state.maxRounds;
        state.staleThreshold = params.staleThreshold ?? state.staleThreshold;
        return {
          resumed: true,
          name: state.name,
          existingResults: state.results.length,
          currentRound: state.currentRound,
          totalFixed: state.totalFixed,
          totalFailed: state.totalFailed,
        };
      }

      // Fresh session
      state = {
        name: params.name,
        linterCommand: params.linterCommand,
        targetDir: targetDir,
        maxRounds: params.maxRounds ?? 1,
        staleThreshold: params.staleThreshold ?? 2,
        results: [],
        currentRound: 1,
        totalFixed: 0,
        totalFailed: 0,
      };

      // Write init event to date-named JSONL
      fs.mkdirSync(tDir, { recursive: true });
      const eventLine =
        JSON.stringify({
          timestamp: new Date().toISOString(),
          type: "init",
          data: {
            name: state.name,
            linterCommand: state.linterCommand,
            targetDir: state.targetDir,
            maxRounds: state.maxRounds,
            staleThreshold: state.staleThreshold,
          },
        }) + "\n";
      fs.appendFileSync(path.join(tDir, todayFileName()), eventLine);

      return {
        resumed: false,
        name: state.name,
        linterCommand: state.linterCommand,
        targetDir: state.targetDir,
        maxRounds: state.maxRounds,
        staleThreshold: state.staleThreshold,
      };
    },
  });

  // ---- Tool: run_linter ----
  tool({
    name: "run_linter",
    description:
      "Execute the configured linter command, parse the output into structured issues grouped by file, and return a summary.",
    parameters: Type.Object({}),
    execute: async () => {
      const { execSync } = require("child_process");
      const path = require("path");

      if (!state) {
        return { error: "No active session. Call init_sweep first." };
      }

      let rawOutput: string;
      try {
        rawOutput = execSync(state.linterCommand, {
          cwd: state.targetDir,
          encoding: "utf-8",
          stdio: ["pipe", "pipe", "pipe"],
          timeout: 120_000,
        });
      } catch (err: any) {
        // Linters often exit non-zero when issues are found
        rawOutput = (err.stdout ?? "") + (err.stderr ?? "");
        if (!rawOutput) {
          return { error: `Linter command failed: ${err.message}` };
        }
      }

      // Parse with three regex patterns matching Go CLI convention
      const golangciPattern =
        /^(.+?):(\d+):(\d+):\s+(.+)\s+\(([@\w][\w./@-]*)\)$/;
      const genericPattern = /^(.+?):(\d+):(\d+):\s+(.+)$/;
      const minimalPattern = /^(.+?):(\d+):\s+(.+)$/;
      const eslintStylishIssue =
        /^\s+(\d+):(\d+)\s+(error|warning)\s+(.+?)\s{2,}(\S+)\s*$/;

      const issues: Array<{
        file: string;
        line: number;
        col: number;
        message: string;
        linter: string;
      }> = [];

      // Try ESLint stylish (multi-line block) format first
      {
        let currentFile = "";
        for (const line of rawOutput.split("\n")) {
          const trimmed = line.trim();
          if (!trimmed) continue;
          if (
            trimmed.includes("problem") &&
            (trimmed.includes("\u2716") ||
              (trimmed.includes("error") && trimmed.includes("warning")))
          )
            continue;

          const sm = line.match(eslintStylishIssue);
          if (sm && currentFile) {
            issues.push({
              file: currentFile,
              line: parseInt(sm[1], 10),
              col: parseInt(sm[2], 10),
              message: sm[4].trim(),
              linter: sm[5],
            });
            continue;
          }

          // File header: non-indented, non-empty
          if (line === trimmed && trimmed.length > 0 && !trimmed.startsWith("\u2716")) {
            currentFile = trimmed;
          }
        }
      }

      // If stylish parse found nothing, fall back to line-by-line patterns
      if (issues.length === 0) {
        for (const line of rawOutput.split("\n")) {
          const trimmed = line.trim();
          if (!trimmed) continue;

          let m: RegExpMatchArray | null;

          m = trimmed.match(golangciPattern);
          if (m) {
            issues.push({
              file: m[1],
              line: parseInt(m[2], 10),
              col: parseInt(m[3], 10),
              message: m[4],
              linter: m[5],
            });
            continue;
          }

          m = trimmed.match(genericPattern);
          if (m) {
            issues.push({
              file: m[1],
              line: parseInt(m[2], 10),
              col: parseInt(m[3], 10),
              message: m[4],
              linter: "custom",
            });
            continue;
          }

          m = trimmed.match(minimalPattern);
          if (m) {
            issues.push({
              file: m[1],
              line: parseInt(m[2], 10),
              col: 0,
              message: m[3],
              linter: "custom",
            });
            continue;
          }
        }
      }

      // Group by file with absolute paths
      const byFile: Record<
        string,
        Array<{ line: number; col: number; message: string; linter: string }>
      > = {};
      for (const issue of issues) {
        const absPath = path.resolve(state.targetDir, issue.file);
        if (!byFile[absPath]) byFile[absPath] = [];
        byFile[absPath].push({
          line: issue.line,
          col: issue.col,
          message: issue.message,
          linter: issue.linter,
        });
      }

      const fileCount = Object.keys(byFile).length;
      return {
        totalIssues: issues.length,
        fileCount,
        files: byFile,
        rawLineCount: rawOutput.split("\n").length,
        parsed: issues.length > 0,
      };
    },
  });

  // ---- Tool: log_result ----
  tool({
    name: "log_result",
    description:
      "Record a fix attempt result. Appends to the date-named JSONL telemetry file and updates session state. Auto-commits on success.",
    parameters: Type.Object({
      file: Type.String({ description: "File that was fixed (or attempted)" }),
      success: Type.Boolean({ description: "Whether the fix succeeded" }),
      round: Type.Number({ description: "Round number (1-based)" }),
      strategy: Type.String({
        description: "Prompt strategy used: standard, retry, or exploration",
      }),
      issuesBefore: Type.Optional(
        Type.Number({ description: "Number of issues before the fix attempt" })
      ),
      issuesAfter: Type.Optional(
        Type.Number({ description: "Number of issues after the fix attempt" })
      ),
    }),
    execute: async (params: any) => {
      const fs = require("fs");
      const path = require("path");

      if (!state) {
        return { error: "No active session. Call init_sweep first." };
      }

      const result: SweepResult = {
        run: state.name,
        file: params.file,
        round: params.round,
        strategy: params.strategy,
        status: params.success ? "fixed" : "failed",
        issuesBefore: params.issuesBefore ?? 0,
        issuesAfter: params.issuesAfter ?? 0,
      };

      state.results.push(result);
      state.currentRound = Math.max(state.currentRound, params.round);
      if (params.success) {
        state.totalFixed++;
      } else {
        state.totalFailed++;
      }

      // Append fix_attempt event to telemetry
      const tDir = telemetryDir(state.targetDir);
      fs.mkdirSync(tDir, { recursive: true });
      const eventLine =
        JSON.stringify({
          timestamp: new Date().toISOString(),
          type: "fix_attempt",
          data: {
            file: params.file,
            success: params.success,
            round: params.round,
            strategy: params.strategy,
            issuesBefore: params.issuesBefore ?? 0,
            issuesAfter: params.issuesAfter ?? 0,
            run: state.name,
          },
        }) + "\n";
      fs.appendFileSync(path.join(tDir, todayFileName()), eventLine);

      // Auto-commit on success
      if (params.success) {
        try {
          const { execSync } = require("child_process");
          execSync(`git add "${params.file}" && git commit -m "sweeper: fix lint issues in ${path.basename(params.file)}"`, {
            cwd: state.targetDir,
            encoding: "utf-8",
            stdio: ["pipe", "pipe", "pipe"],
          });
        } catch {
          // Commit failure is non-fatal (e.g. nothing to commit)
        }
      }

      return {
        recorded: true,
        file: params.file,
        status: result.status,
        round: params.round,
        strategy: params.strategy,
        sessionTotals: {
          fixed: state.totalFixed,
          failed: state.totalFailed,
          attempts: state.results.length,
        },
      };
    },
  });

  // ---- Widget ----
  widget({
    name: "sweeper_status",
    description: "Sweeper session status dashboard",
    render: () => {
      if (!state) {
        return {
          collapsed: "sweeper: no active session",
          expanded: "Call init_sweep to start a session.",
        };
      }

      const total = state.results.length;
      const pct = total > 0 ? Math.round((state.totalFixed / total) * 100) : 0;

      const collapsed = `sweeper: ${total} attempts | ${state.totalFixed} fixed (${pct}%) | round ${state.currentRound}`;

      // Last 10 results for expanded view
      const recent = state.results.slice(-10);
      const lines = recent.map((r) => {
        const icon = r.status === "fixed" ? "+" : "-";
        return `  [${icon}] ${r.file} (${r.strategy}, round ${r.round})`;
      });

      const expanded = [
        `Session: ${state.name}`,
        `Linter: ${state.linterCommand}`,
        `Target: ${state.targetDir}`,
        `Rounds: ${state.currentRound} / ${state.maxRounds}`,
        `Fixed: ${state.totalFixed} | Failed: ${state.totalFailed} | Total: ${total}`,
        "",
        "Recent results:",
        ...lines,
      ].join("\n");

      return { collapsed, expanded };
    },
  });
}
