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

Reframe the plugin's primary responsibility from "helps the agent work with Archcore MCP" to **a host-side enforcement boundary that physically prevents any `.archcore/` mutation outside the MCP tools**, and close the remaining coverage gaps in the `PreToolUse` matcher set so the guarantee actually holds.

The plugin and CLI are already decoupled at the lifecycle level since v0.4.0: the user installs the CLI globally, the plugin resolves `archcore` through PATH, and every hook degrades silently when the CLI is missing. What is not yet decoupled cleanly is the conceptual contract — the plugin still presents itself as a helper around MCP, while the load-bearing role, the only thing that makes the MCP-only invariant true on disk, is the `PreToolUse` block hook. Naming the role correctly makes it auditable, and the audit finds concrete gaps that today let an agent bypass MCP and write directly.

## Value

**A clear contract between plugin and CLI.** After the reframe, the plugin owns host-side hook installation, the agent-facing skills and commands, the PreToolUse fence around `.archcore/`, and the staleness, cascade, and precision warnings; the CLI owns the MCP server, the on-disk format, the sync manifest, and `archcore doctor`; and the contract between them is the MCP protocol and nothing else — no shared files, no version coupling, no auto-install logic. This is the architecture the launcher removal was already pointing at, and the reframe names it.

**It closes the user's actual fear.** The fear is not that the CLI has bugs; it is that the agent writes to `.archcore/` without going through MCP, corrupting the manifest, skipping templates, and missing relation discovery. That fear is well founded, because today's fence covers only `Write` and `Edit` while the agent has other ways to mutate a file.

**It forces a host-coverage matrix.** Once enforcement is the plugin's stated job, every new host inherits a clear acceptance test: does this host's pre-mutation surface cover every tool that can mutate a filesystem path? A host that fails cannot be supported with the MCP-only guarantee and falls back to advisory-only.

## Possible Implementation

**Gap 1 — the Bash tool is absent from the matcher.** The matcher is `Write|Edit` on Claude Code and `Write` on Cursor, so an agent bypasses the fence entirely with a shell redirect into `.archcore/`, an in-place `sed`, an append, or a `printf`. The approach is to add `Bash` to the matcher and have `check-archcore-write`, or a sibling script, parse `tool_input.command` for a write targeting a `.archcore/` markdown path. The block set is: `>` and `>>` redirection into `.archcore/`; `tee` and `tee -a` writing there; `sed -i`, `sed -i.bak`, and `gsed -i` on such a path; `perl -i` and `awk -i inplace`; `mv`, `cp`, and `install` whose destination is a `.archcore/` markdown file; and `rm` of one, which matters less but still bypasses manifest-mediated removal. The allow set is anything read-only — `cat`, `grep`, `head`, `tail`. Command parsing is heuristic and a determined agent can obfuscate, so the goal is to make the easy path go through MCP rather than to defeat an adversary, and the heuristic must be documented as best-effort defense in depth.

**Gap 2 — the Cursor matcher lacks `Edit`.** Cursor's matcher is `Write` only against Claude Code's `Write|Edit`, a known asymmetry. The resolution depends on whether Cursor's `preToolUse` matcher accepts pipe alternation: if it does, widen the matcher; if not, register two entries.

**Gap 3 — other mutating tools.** Audit the current host tool surface for anything that can mutate a file outside `Write`, `Edit`, and `Bash`, and where a tool is host-specific and has no counterpart in another host's config, record it in the supported-host matrix.

**Gap 4 — make the matrix a structural test.** Add a test that enumerates the matcher sets across every host hooks config and fails CI when a known mutation tool is missing for a supported host. Today that gap stays invisible until a user trips it.

**Gap 5 — document the contract surface.** Restate the framing of `always-use-mcp-tools.adr`, which today addresses agent instructions and a single hook, to say plainly that the plugin is the only on-host component enforcing MCP-only; that the CLI does not police writers and simply interprets whatever it finds on disk; and that MCP-only is not an architectural invariant of `.archcore/` but a property the plugin enforces through PreToolUse, so disabling the plugin disables the guarantee. Add a coverage-matrix section naming every mutation tool the plugin claims to cover per host.

**Gap 6 — stronger-than-hook guarantees, optional.** A PreToolUse hook is advisory in the sense that a user can switch it off in host config. A hard guarantee would require making `.archcore/` files immutable through a filesystem flag and lifting it only inside MCP write paths, or routing writes through a watcher that rejects non-MCP origins. Both are heavyweight and OS-specific, and neither is recommended for a first version; they are named to show the design space.

## Risks

- **Bash command parsing is heuristic**, producing false positives when a legitimate command merely mentions `.archcore/` in a string and false negatives under obfuscation. Mitigated by a clear allow and block lexicon, an escape-hatch environment variable, and an error message that points the agent at the MCP tool.
- **Cursor host coverage is unverified.** Whether Cursor honors `Bash` in a `preToolUse` matcher is not known, and the fallback is Write-only protection there.
- **A reframe with no behavior change reads as a rename.** The reframe is worth doing only if it lands together with Gap 1 at minimum; otherwise it is documentation churn.
- **The reframe meets the existing rule framing.** `mcp-only-operations.rule` addresses the agent and says every operation must use MCP; this idea addresses the architecture and says the plugin enforces it. They are compatible and need explicit cross-references.
- **A disabled plugin means no enforcement**, which deserves an explicit user-facing acknowledgement in the README so the guarantee is not overpromised.
