---
title: "Pre-Code Context Injection Hook Implementation Plan"
status: draft
---

## Goal

Ship the push-mode counterpart to `/archcore:context`: a `PreToolUse Write|Edit` hook (`bin/check-code-alignment`) that injects applicable rules/ADRs/specs/cpats as `additionalContext` when an agent is about to edit source code outside `.archcore/`. This closes JTBD #1 into an engineered guarantee — pull (user-driven) + push (automatic on edit).

Together with the already-shipped `/archcore:context` skill, this retires the single biggest README-vs-reality gap identified in `jtbd-alignment-analysis.idea.md`.

## Scope of this iteration

MVP: grep-based, shell-only. No CLI subcommand, no persistent path index. Runs on every `Write|Edit` outside `.archcore/`, emits top-3 matches by specificity → type priority. Multi-host parity (Claude Code + Cursor).

Explicitly **out of scope** for this iteration (deferred):

- CLI `archcore search` subcommand (MCP tool exists; CLI wrapper is separate work)
- Persistent path index in `.sync-state.json`
- Session-level deduplication (same doc re-injected across session)
- Telemetry / metrics
- Greenfield proactive bootstrap

## Architecture

```
PreToolUse Write|Edit
  ├─ bin/check-archcore-write  ── blocks .archcore/*.md writes (existing)
  └─ bin/check-code-alignment  ── injects context for source edits (NEW)
```

Both hooks coexist on the same matcher. `check-archcore-write` short-circuits on `.archcore/*.md` (exit 2 blocks). `check-code-alignment` short-circuits on everything INSIDE `.archcore/` (exit 0 silent). On source paths only the alignment hook does work.

### bin/check-code-alignment algorithm

1. **Preconditions** — bail silently if: no `$ARCHCORE_FILE_PATH`, no `.archcore/` dir, path is inside `.archcore/`, env `ARCHCORE_DISABLE_INJECTION=1`.
2. **Path normalization** — absolute paths under `$CWD` → relative; `./foo` → `foo`.
3. **Source-root filter** — file path must start with a configured source root. Default roots: `src lib app pkg cmd internal apps packages modules components`. Override via `.archcore/settings.json` → `codeAlignment.sourceRoots` (array).
4. **Token generation** — from file path, emit directory prefixes longest-first. `src/api/handlers/users.ts` → `src/api/handlers/`, `src/api/`, `src/`. Cap at 5 tokens.
5. **Document scan** — for each token (longest first): `grep -rlF <token> .archcore --include='*.md'`. Skip a doc already matched by a longer (more specific) token. Classify by filename extension: only `.rule.md`, `.cpat.md`, `.adr.md`, `.spec.md`, `.guide.md` are eligible (skip `.prd`, `.idea`, `.plan`, etc.).
6. **Ranking** — sort by `specificity_length × 10 + type_priority`, where priority is `rule=5, cpat=4, adr=3, spec=2, guide=1`. Take top 3.
7. **Render** — `[Archcore Context] Before editing <path>:\n- <type>: <title> [<short-path>]\n...` Cap output at 2 KB.
8. **Emit** — per host:
   - `claude-code` / `copilot`: `{"hookSpecificOutput":{"hookEventName":"PreToolUse","additionalContext":"..."}}`
   - `cursor`: `{"additional_context":"..."}` (preToolUse — may be ignored by current Cursor; graceful degradation, document as known limitation for this iteration)

### hook registration

`hooks/hooks.json` — second entry in existing `PreToolUse Write|Edit` array, after `check-archcore-write`:

```json
{"matcher":"Write|Edit","hooks":[
  {"type":"command","command":"${CLAUDE_PLUGIN_ROOT}/bin/check-archcore-write","timeout":1},
  {"type":"command","command":"${CLAUDE_PLUGIN_ROOT}/bin/check-code-alignment","timeout":1}
]}
```

`hooks/cursor.hooks.json` — parallel addition under `preToolUse` `Write` matcher.

### shared helper

Add `archcore_hook_pretool_info` to `bin/lib/normalize-stdin.sh` — mirrors `archcore_hook_info` but emits `hookEventName: PreToolUse`. Other hooks in this iteration do not use it, but the helper keeps JSON-shape logic centralized.

## Acceptance Criteria

1. `bin/check-code-alignment` exists, executable, POSIX-shell only.
2. On a source-file Write/Edit that has matching `.archcore/` documents: emits valid JSON with `additionalContext` containing top-3 docs ranked by specificity → type → nothing else.
3. On any short-circuit condition: exit 0, no output.
4. Never returns a non-zero exit — injection is strictly additive; failures must not block edits.
5. Both `hooks/hooks.json` and `hooks/cursor.hooks.json` list the new script.
6. Existing hook bats (`test/structure/hooks.bats`): event set invariants still pass.
7. New unit tests (`test/unit/check-code-alignment.bats`) cover: source root filter, .archcore skip, token specificity ranking, type priority ranking, top-3 truncation, empty `.archcore/` silent pass, non-source path silent pass, escape hatch env var.
8. `hooks-validation-system.spec.md` updated: 5 hooks documented.
9. `component-registry.doc.md` updated: new script listed.
10. Plugin version bumped (0.2.3 → 0.3.0 — new capability).
11. README `## Try these 3 prompts first` section adjusted so hero claim no longer over-promises (context injection now actually happens automatically).

## Manual smoke tests (on this repo post-merge)

- `Write` to `skills/capture/SKILL.md` — should NOT inject (inside plugin file, but not a source root).
- `Write` to a hypothetical `src/hooks/foo.sh` — should inject hook-related docs.
- `Write` to `.archcore/foo.adr.md` — `check-archcore-write` blocks first; no injection output (checked on exit-2 path).
- `ARCHCORE_DISABLE_INJECTION=1` set → no injection on any path.

## Dependencies

- `bin/lib/normalize-stdin.sh` (existing) — provides `$ARCHCORE_FILE_PATH`, `$ARCHCORE_HOST`, helpers.
- No new CLI version required; MCP tool `search_documents` shipped in CLI 0.1.7 is orthogonal (used by `/archcore:context` skill, not by this hook).

## Risks

- **Performance:** grep-per-token on N docs scales as O(tokens × docs). For 40 docs × 5 tokens ≈ 200 grep invocations. Each ~2–5 ms → 400 ms–1 s. Near the 1-second timeout; acceptable for MVP, must monitor. Phase 2 (path index) eliminates this.
- **False positives:** a doc mentioning `src/` generically will match every source edit. Ranking by longest prefix penalizes this, but not to zero. User feedback will tell if top-3 signal-to-noise is adequate.
- **Cursor preToolUse context injection:** uncertain whether Cursor respects `additional_context` in preToolUse. If not, the output is silently ignored. Cursor parity remains via SessionStart + afterMCPExecution (other JTBDs) — not regression.
- **Hook fatigue / repetition:** no session dedup. Same rule may be injected on 20 consecutive edits in the same directory. Monitor user feedback before adding dedup (it adds state and failure modes).

## Relations

- `implements` → `jtbd-alignment-analysis.idea` (closes JTBD #1 push-mode)
- `implements` → `pre-code-context-injection.idea`
- `related` → `code-alignment-intent-skill.idea` (pull counterpart idea)
- `related` → `context-skill-implementation.plan` (shipped pull-mode plan)
- `extends` → `hooks-validation-system.spec` (adds 5th hook)
