---
title: "install_host_config Is Gated, Non-Destructive, and Never Nudged"
status: accepted
tags:
  - "cli"
  - "integrations"
  - "mcp"
---

## Context

The `install_host_config` MCP tool (@internal/mcp/tools/install_host_config.go) lets an agent wire a project's host configs — hooks, MCP server entry, instruction nudge — producing the same artifacts as `archcore init --agent`. It exists for parity: the Claude plugin's init flow must reach the identical end state as the CLI wizard. But it is the only MCP tool that writes files *outside* `.archcore/`, into configs the user hand-edits, so its invocation contract needs to be pinned.

## Decision

Three linked decisions form the tool's contract:

**1. Invocation is gated.** The tool description opens with an explicit gate: call ONLY when the user has explicitly asked to set up/wire Archcore into their host AND has confirmed a plan the agent stated first. The description carries negative examples (not because hooks look missing; not for a generic "set up my project" request). An agent noticing an unwired project must ask, not act.

**2. `DestructiveHint` is false — and stays honest structurally.** The tool writes to user-owned config files, yet declares itself non-destructive. That is truthful only because of three contracts it builds on:
- hook installs are append/update-in-place scoped to marker-recognized entries (hook-command-marker-prefix.adr) — foreign entries are never touched;
- MCP config writes merge only owned fields (mcp-config-converge-ownership.adr) — user fields and foreign servers survive;
- instruction-nudge writes are fenced (only the archcore marker span is replaced) AND atomic (a sibling temp file renamed into place, preserving the file's permissions and following a symlink to its target) — so a crash mid-write cannot shred a user's `CLAUDE.md`/`AGENTS.md`/`GEMINI.md`.
If any contract weakens, `DestructiveHint` must be revisited in the same change.

**3. Instruction-nudge files never mention the tool.** The usage-nudge instructions (CLAUDE.md / AGENTS.md / GEMINI.md) deliberately omit `install_host_config` — nudging agents toward a gated tool contradicts the gate. This omission is by design, not an oversight.

**The report names every artifact, per agent.** The JSON report lists each artifact path the wiring ensured (project-relative, sanitized errors). Instruction files carry a two-field shape: `instructions_path` (the agent's primary file) plus additive `instructions_extra_paths` for multi-target agents (claude-code: `CLAUDE.md` primary + `AGENTS.md` extra) — an MCP client relays this report, so a file written to the user's repo but absent from it would be a silent write. Derived from `Agent.AllInstructionsPaths` via `AgentResult.ExtraInstructions`, the same source as the CLI confirmation line. The report reflects **what actually landed on disk** (`agents.InstructionBlockPresent`), not the all-or-nothing write error: on a partial multi-file write the file that was written is still named (the one that failed is omitted), and a failure on a *shared* file (AGENTS.md) is attributed to every agent that shares it, not just the dedupe representative — so a non-representative agent (e.g. opencode behind cursor) never silently loses its instructions error.

**Registration is conditional.** The tool is registered only when the cmd layer injects an executor via `mcpserver.WithHostWiring` (@internal/mcp/server.go); a bare `NewServer` does not expose it. The executor lives in cmd (@cmd/host_wiring.go) and adapts @internal/wiring — this injection avoids a cmd→internal/mcp import cycle and keeps the MCP boundary responsible for sanitization (project-relative paths, sanitized errors per no-absolute-paths-in-mcp-errors.rule).

## Alternatives

- **Ungated tool + nudge in instructions**: agents would wire hosts opportunistically, rewriting user configs on vague prompts. Rejected.
- **`DestructiveHint: true`**: technically cautious but false — the tool structurally cannot delete user data, and the hint would push clients to over-prompt for an idempotent operation. Rejected.
- **Always-registered tool with executor in internal/**: pulls the agent installers under internal/mcp or creates an import cycle; wiring is a CLI concern that the MCP layer merely fronts. Rejected.

## Consequences

- Plugin `/archcore:init` and `archcore init --agent` converge on one code path (@internal/wiring), so parity cannot drift silently.
- Headless or test servers built without `WithHostWiring` are guaranteed free of the tool (pinned by `TestNewServer_HostWiringToolRegistration`).
- Changes to the marker or converge contracts must re-check this ADR's decision 2; changes to the tool description must preserve the gate of decision 1.
