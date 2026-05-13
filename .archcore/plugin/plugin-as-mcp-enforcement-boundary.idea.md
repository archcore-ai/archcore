---
title: "Plugin as Host-Side MCP Enforcement Boundary — Reframe and Close Hook Coverage Gaps"
status: draft
tags:
  - "architecture"
  - "hooks"
  - "multi-host"
  - "plugin"
  - "validation"
---

## Idea

Reframe the plugin's primary responsibility from "helps the agent work with Archcore MCP" to **"host-side enforcement boundary that physically prevents any `.archcore/` mutation outside of MCP tools"**, and close the remaining coverage gaps in the `PreToolUse` matcher set so the guarantee actually holds.

The plugin and CLI are already decoupled at the lifecycle level (v0.4.0 — see `remove-bundled-launcher-global-cli.idea.md`): the user installs the CLI globally, the plugin resolves `archcore` via PATH, and every hook degrades silently when the CLI is missing. What is **not** yet decoupled cleanly is the conceptual contract — the plugin still presents itself as a "helper around MCP" while the load-bearing role (the only thing that actually makes the MCP-only invariant true on disk) is the `PreToolUse` block hook.

If we name the plugin's role correctly, we can audit it against that role and find concrete coverage gaps that today let an agent silently bypass MCP and write directly to `.archcore/`.

## Value

### A clear contract between plugin and CLI

After this reframe:

- **Plugin** owns: host-side hook installation, agent-facing skills/commands, the PreToolUse fence around `.archcore/`, the staleness/cascade/precision warnings.
- **CLI** owns: the MCP server, the on-disk format of `.archcore/`, the sync manifest, `archcore doctor`.
- **Contract between them**: the MCP protocol. Nothing else. No shared files, no version coupling, no auto-install logic.

This is the architecture the bundled-launcher removal was already pointing at. The reframe just names it.

### Closes the user's actual fear

The fear isn't that the CLI has bugs — it's that the agent writes to `.archcore/` *without going through MCP*, corrupting the manifest, skipping templates, missing relation discovery. That fear is real because today's PreToolUse fence only covers `Write` and `Edit`. The agent has other ways to mutate files.

### Forces a host-coverage matrix

Once "enforcement boundary" is the plugin's job, every new host (Codex, Gemini CLI, Continue, future entries) inherits a clear acceptance test: *does this host's PreToolUse / equivalent surface cover all tools that can mutate filesystem paths?* Hosts that fail this test cannot be supported with the MCP-only guarantee; they fall back to advisory-only (skills + rule docs).

## Possible Implementation

### Gap 1 — Bash tool not in PreToolUse matcher

Today: matcher is `Write|Edit` (Claude Code) and `Write` (Cursor). An agent can bypass entirely with:

```
Bash("cat <<EOF > .archcore/foo.md\n...\nEOF")
Bash("sed -i 's/draft/accepted/' .archcore/foo.adr.md")
Bash("echo '...' >> .archcore/foo.adr.md")
Bash("printf '...' > .archcore/foo.md")
```

Approach: add `Bash` to the matcher and have `check-archcore-write` (or a sibling `check-archcore-bash`) parse `tool_input.command` for shell output redirection or in-place editors targeting `.archcore/*.md`.

Detection patterns to block:

- `> .archcore/*.md`, `>> .archcore/*.md` (any redirection target inside `.archcore/`)
- `tee` / `tee -a` writing into `.archcore/`
- `sed -i`, `sed -i.bak`, `gsed -i` on a path under `.archcore/`
- `perl -i`, `awk -i inplace` on paths under `.archcore/`
- `mv`, `cp`, `install` whose destination is under `.archcore/*.md`
- `rm .archcore/*.md` (less critical, but the manifest expects MCP-mediated removal)

Allow list: `cat .archcore/...`, `grep .archcore/...`, `head/tail .archcore/...`, anything that only reads.

Risk: command parsing is heuristic. A determined agent can still obfuscate. But the goal is to make the easy path go through MCP, not to defeat adversarial bypass. Document the heuristic explicitly so users know it's "best-effort, defense in depth".

### Gap 2 — Cursor matcher missing `Edit`

Today Cursor's `cursor.hooks.json` matcher is `Write` only. The Claude Code matcher is `Write|Edit`. Per `hooks-validation-system.spec.md` this is a known asymmetry. Resolution depends on whether Cursor's `preToolUse` matcher accepts pipe alternation — if yes, change to `Write|Edit`; if no, register two separate matcher entries.

### Gap 3 — MultiEdit / NotebookEdit / patch-like tools

Audit the current Claude Code tool surface for any tool that can mutate files outside `Write|Edit|Bash`. If a tool is host-specific and not announced in `cursor.hooks.json` form, document it in a supported-host matrix.

### Gap 4 — Host-coverage matrix as a structural test

Add a structural test that enumerates the matcher sets across `hooks/hooks.json` and `hooks/cursor.hooks.json` and fails CI if a known mutation tool is missing from the matcher for a supported host. Today this gap is invisible until a user trips it.

### Gap 5 — Document the contract surface

Update or replace `always-use-mcp-tools.adr.md` framing: today it talks about agent instructions and a single PreToolUse hook. The reframe should explicitly state:

- The plugin is the *only* on-host component responsible for enforcing MCP-only.
- The CLI does **not** police writers — it is just a server that interprets whatever it finds on disk.
- "MCP-only" is not an architectural invariant on `.archcore/` — it is a property the plugin *enforces* via PreToolUse hooks. Disabling the plugin disables the guarantee.

Add a `mcp-enforcement.spec.md` (new) or extend `hooks-validation-system.spec.md` with a "Coverage Matrix" section that names every mutation tool the plugin claims to cover per host.

### Gap 6 — Optional: stronger-than-hook guarantees

PreToolUse hooks are advisory in the sense that the user can turn them off in host config. For users who want a hard guarantee, an opt-in mode could:

- Make `.archcore/` files immutable to the user via filesystem ACL / `chflags uchg` / `chattr +i` and lift the flag only inside MCP write paths. Heavyweight and OS-specific — propose only as a Phase 3 idea.
- Or: route writes through a FUSE / file-watcher that rejects non-MCP-originated writes. Even heavier.

Not recommended for v1. Mentioned to show the design space.

## Risks

- **Bash command parsing is heuristic.** False positives (block a legitimate Bash command that mentions `.archcore/` in a string but doesn't write) and false negatives (clever obfuscation). Mitigation: clear allow/block lexicon, escape hatch env var, error message that points the agent at the MCP tool.
- **Cursor host coverage uncertain.** Whether Cursor honors `Bash` in `preToolUse` matchers is not verified. Falls back to Write-only protection if not.
- **Reframing without behavior change feels like a rename.** The reframe is only valuable if it lands together with closing Gap 1 (Bash) at minimum. Otherwise it's documentation churn.
- **Reframe collides with `mcp-only-operations.rule.md` framing.** Need to reconcile: the rule says "all operations MUST use MCP" addressing the agent; this idea says "the plugin enforces it" addressing the architecture. They are compatible but need cross-references.
- **Plugin disabled = no enforcement.** Worth explicit user-facing acknowledgment in README, so the guarantee is not overpromised.

## Related work in this repo

- `always-use-mcp-tools.adr.md` — the original decision that this idea reframes architecturally
- `mcp-only-operations.rule.md` — the agent-facing rule that this idea reinforces structurally
- `hooks-validation-system.spec.md` — the contract surface that would be extended (new matcher entries, coverage matrix section)
- `remove-bundled-launcher-global-cli.idea.md` — the lifecycle-level decoupling already shipped in v0.4.0; this idea is the conceptual completion
- `cursor-mcp-architecture.adr.md` — example of the "plugin as boundary, not as MCP host" principle applied to Cursor
- `multi-host-compatibility-layer.spec.md` — host-coverage matrix would live here or extend it
- `cli-integration-tests.rule.md` — the same discipline that locks CLI subcommand surface would lock host matcher coverage
