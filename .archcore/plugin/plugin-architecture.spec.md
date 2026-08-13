---
title: "Plugin Architecture — Four-Command Surface over Gated Tracks"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Purpose & Scope

This spec defines how the plugin's components — intent skills, agents, hooks, and the MCP server — compose into one multi-host system across Claude Code, Cursor, Codex CLI, and GitHub Copilot CLI: the runtime model, the invocation pathways, the data flow, and the cross-component invariants. Normative for the whole plugin runtime, from a user message or model decision through skill activation, MCP tool calls, hook enforcement, and validation feedback. Depended on by every component spec and by the host adapters.

Component contracts live in dedicated specs — `command-surface-v2.spec`, `track-layer.spec`, `agent-system.spec`, `hooks-validation-system.spec`. On a conflict, the dedicated spec wins for its own component and this spec wins for cross-component interaction. The current skill surface is fixed by `four-command-palette.adr`, which supersedes the track tier of `intent-based-skill-architecture.adr`, the `actualize`/`review` split of `merge-review-status-remove-graph.adr`, the mainstream/niche stratification of `inverted-invocation-policy.adr`, and the standalone `standard` and `verify` intents.

## Surface

The plugin exposes four auto-invocable commands (`init`, `plan`, `document`, `review`) plus a non-palette gated track layer per `track-layer.spec`. Every user-facing entry point is one of them; gate logic lives in track files that the routed skill loads on demand.

```
┌─────────────────────────────────────────────────────────────────┐
│                        User / Host Model                        │
│                                                                 │
│  "plan this feature"   "record decision"   "/archcore:document" │
└──────┬──────────────────────┬──────────────────────┬────────────┘
       │                      │                      │
       ▼                      ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│  COMMANDS (4, auto-invocable)                                   │
│                                                                 │
│  /archcore:init     /archcore:plan      /archcore:document      │
│  /archcore:review                                               │
│                                                                 │
│  → Routes user intent to a track, a type, or an analysis mode   │
│  → Gated tracks under skills/_shared/tracks/                    │
│  → Content contracts under skills/_shared/                      │
├─────────────────────────────────────────────────────────────────┤
│  MCP PRIMITIVES (INFRASTRUCTURE)                                │
│                                                                 │
│  create_document  update_document  remove_document              │
│  list_documents   get_document     search_documents             │
│  add_relation  remove_relation  list_relations  init_project    │
│                                                                 │
│  → Atomic CRUD + relations over .archcore/                      │
│  → Accepts any document type by `type` parameter                │
│  → Used by all skills, also directly callable                   │
├─────────────────────────────────────────────────────────────────┤
│  HOOKS LAYER (CROSS-CUTTING, event-driven)                      │
│                                                                 │
│  SessionStart → CLI recap + staleness                           │
│  PreToolUse → block direct writes + inject code-aligned context │
│  PostToolUse → validate after MCP mutations + cascade detection │
│              + precision check                                  │
└─────────────────────────────────────────────────────────────────┘
```

| Skill | Role | Invocation |
|---|---|---|
| `init` | Seed `.archcore/` on first install | Auto + user |
| `plan` | Plan a feature or initiative; route into a track | Auto + user |
| `document` | Record decisions and document modules/components (absorbs `capture`, `decide`) | Auto + user |
| `review` | Documentation health, drift, and status (absorbs `audit`; `context` and `help` removed) | Auto + user |

| Component | Count | Layer | Files |
|---|---|---|---|
| Track files | 8 | shared, loaded by `plan`, `document`, `review` | `skills/_shared/tracks/{sdd,requirements-cascade,research,decision,describe,actualize,experience,closeout}.md` |
| Shared content contracts | 5 | shared | `skills/_shared/{precision-rules,adr-contract,prd-contract,spec-contract,rule-contract}.md` |
| Other shared assets | 6 | shared | `skills/_shared/{elicitation-contract,gate-contract,branch-state,coverage-taxonomy,globals}.md` + `skills/_shared/grounding/` (13 catalogs) |
| Agents | 2 | cross-cutting | `agents/archcore-{assistant,auditor}.{md,toml}` + `copilot-agents/archcore-{assistant,auditor}.agent.md` |
| Hooks | 3 entries across 3 events | cross-cutting | `hooks/{hooks,cursor.hooks,codex.hooks,copilot.hooks}.json` |
| Bin scripts | 5 + 3 libraries | cross-cutting | `bin/{session-start,pre-tool-use,post-tool-use,detect-host,cli-gte}` + `bin/lib/{normalize-stdin,plugin-cache-guard,empty-state}.sh` |
| MCP server | 1 | infra | Provided by archcore CLI |
| Command wrappers | 4 | host adapter | `commands/<name>.md` (Codex requires them; Copilot loads them behind its skills) |

Since v0.7.0 the hook policy executes inside the archcore CLI and the plugin ships only launchers, per `cli-owns-layers-4-5.adr`. The per-flow references, the `audit` lib, and the `decide` continuations that earlier revisions of this table listed were removed at the v2 track cutover; their logic lives in the track files above.

**Invocation paths.** Four paths exist, and every one converges on the MCP tool layer.

1. **Skill invocation.** The user types `/archcore:plan auth-redesign sdd`, or the model auto-invokes from natural language; the routing table picks the track, the skill loads `skills/_shared/tracks/sdd.md` on demand, runs its gates in sequence (evaluate `skip_when` → ask within budget → create → relate), and hooks validate each mutation.
2. **Agent delegation.** The host judges the task complex, or the user asks for agent help; the agent calls MCP tools directly and hooks validate. The definition the host loads depends on its loader — `.md` on Claude Code and Cursor, `.toml` on Codex, `*.agent.md` from `copilot-agents/` on Copilot — same content, three containers.
3. **Direct MCP.** The model or the user calls `mcp__archcore__create_document(type=<any>, …)`; no skill is required. The server is plugin-shipped on Claude Code and Codex and project-registered on Cursor and Copilot (`cursor-mcp-architecture.adr`, `copilot-mcp-architecture.adr`). Tool names differ with it — `mcp__archcore__*`, `mcp__plugin_archcore_archcore__*`, or Copilot's flat `archcore-<tool>` — which is why every matcher and allow-list carries all three and the normalizer folds them into one.
4. **Staleness detection.** SessionStart emits the CLI recap, which carries the staleness advisory; the post-tool-use launcher's CLI handler emits the cascade notice after `update_document`; `/archcore:review --drift` runs the actualize track with per-finding verdicts and confirmed fixes.

**Direct-write interception.** When a component calls Write or Edit against a `.archcore/` markdown path, the CLI's pre-tool-use handler — reached through `bin/pre-tool-use` — extracts the target from stdin, matches the pattern, denies through the host's mechanism, and the model retries via `create_document` or `update_document`. The deny mechanism is the one architectural detail that cannot be host-neutral: exit 2 plus stderr on Claude Code, Codex, and Cursor; on Copilot the guard writes `{"permissionDecision":"deny",…}` to stdout with exit 0, because there **every** non-zero exit denies and only the JSON form carries the reason back to the user.

That asymmetry propagates one level up, into how hooks are wired rather than what they decide. Where a non-zero exit is the deny channel, a guard that cannot start degrades to no enforcement; where every non-zero exit denies, it degrades to refusing every matched edit, indistinguishable from a real verdict. Hook bootstrap is therefore part of the architecture on such a host — see `host-adapter-contract.spec`.

**Hook event matrix.** One launcher per event; the policy behind each runs in the CLI (`hooks-validation-system.spec`).

| Launcher | Event (Claude Code) | Event (Cursor) | Event (Codex) | Event (Copilot) | Policy behind it |
|---|---|---|---|---|---|
| `bin/session-start` | SessionStart | sessionStart | SessionStart | sessionStart | Ranked context recap, staleness advisory, CLI-missing and wiring guidance |
| `bin/pre-tool-use` | PreToolUse (`Write\|Edit`) | preToolUse (`Write`) | PreToolUse (`Write\|Edit\|apply_patch`) | preToolUse (`create\|edit\|str_replace_editor\|apply_patch`) | Block direct `.archcore/*.md` writes; inject code-aligned context for source edits |
| `bin/post-tool-use` | PostToolUse (five mutation tools, both namings) | afterMCPExecution (no matcher) | PostToolUse (five mutation tools) | postToolUse (no matcher — the CLI gates on the tool name) | Doctor validation, cascade notice, precision scan |

Four properties of this matrix are load-bearing. Where a host offers no matcher, selection moves into the CLI, so the *set* of tool calls that trigger a behavior does not differ by host — only the mechanism that selects them. Copilot's entries are structurally different (flat objects, `bash` instead of `command`, `timeoutSec` instead of `timeout`), so a config or a test produced by copying another host's row loads cleanly and does nothing. The Copilot column's commands reach `bin/` through a chain of candidate plugin roots rather than one substitution, because GitHub documents no plugin-root variable for hook processes — a matrix row says nothing about whether the script is reachable, which is why reachability is tested by execution. And every launcher fails OPEN: a missing CLI, a CLI below 0.7.0, or a plugin-cache working directory yields a silent exit 0, so a deny can originate only in the CLI's own verdict.

**Routing to a component.**

| Scenario | Component |
|---|---|
| "Plan this feature" | `/archcore:plan` (skill; routes to the sdd track) |
| "Record this decision" | `/archcore:document` (decision track, ADR path) |
| "Draft an RFC" | `/archcore:document` (decision track, RFC path) |
| "Resolve the RFC" | `/archcore:document` (decision track, `decision.resolve` entry) |
| "Establish a standard" | `/archcore:document` (decision track, ADR + rule + guide cascade) |
| "Document this module" | `/archcore:document` (describe track) |
| "Investigate X before we plan" | `/archcore:plan` (research track, produces `rnd`) |
| "Show docs dashboard / counts" | `/archcore:review` (skill, default short mode) |
| "Audit docs health" | `/archcore:review --deep` (skill) |
| "Are any docs stale?" | `/archcore:review --drift` (actualize track) |
| "Close out the feature" | `/archcore:review` (closeout track) |
| "What rules apply to src/X/" | CLI hooks + command grounding (no command; `/archcore:context` removed — see `remove-context-command.adr`) |
| Run ISO requirements cascade | `/archcore:plan iso` (requirements-cascade track, iso mode) |
| Build full standard with pattern change | `/archcore:document` (decision track; `cpat` offered at the cascade gate) |
| Create a single niche document directly | direct `mcp__archcore__create_document` |
| Restructure all auth docs with relations | `archcore-assistant` agent |
| Audit documentation quality | `archcore-auditor` agent |
| Run plugin integrity checks | `make verify` from the repository root (no skill) |

Agents are an escalation path, not the primary interface. Both are restricted to MCP plus Read, Grep, and Glob: the assistant holds every MCP tool, the auditor only the read tools. Every entry appears under all three MCP namings, because a deny-list that misses one fails open.

## Normative Behavior

1. Every document operation MUST flow through an MCP tool. This is the MCP-only principle, enforced at PreToolUse.
2. Every intent skill MUST be auto-invocable.
3. A skill MUST NOT carry `disable-model-invocation`.
4. A skill MUST provide an explicit routing table with bounded decision branches.
5. A flow-style skill MAY substitute a numbered step sequence for that routing table.
6. A skill MUST default to the minimum viable path.
7. WHEN a skill needs to expand beyond that path, the skill MUST offer the expansion as one scope question.
8. A skill that creates documents MUST carry an inline creation recipe or route into a track file.
9. The executing skill MUST hold gate logic in `skills/_shared/tracks/<track>.md`, not in the skill body.
10. A skill MUST NOT instruct a direct file write to `.archcore/`.
11. A skill MUST reference each MCP tool by its exact name.
12. A skill MUST NOT branch on the host; host differences live in `bin/` and in per-host configuration.
13. An agent MUST use MCP tools exclusively for `.archcore/` operations.
14. WHEN a relevant tool call occurs, its hook MUST fire, whichever path initiated the call.
15. The PreToolUse guard MUST block a `.archcore/**/*.md` write through the deny mechanism its host honors.
16. A hook MUST NOT influence a tool call for any reason other than its own verdict.
17. IF a hook cannot reach its script on a host where every non-zero exit denies, THEN the hook MUST exit 0.
18. WHEN an MCP document mutation completes, the CLI's post-tool-use handler MUST run the doctor validation.
19. WHEN `update_document` completes, that handler MUST emit the cascade notice.
20. WHEN `create_document` or `update_document` completes, that handler MUST run the precision scan.
21. The plugin MUST NOT register a PostToolUse hook for `Write|Edit`, because PreToolUse already blocks a `.archcore/` write before it succeeds and the entry would fork a shell on every write anywhere in the repository.
22. WHEN a session starts, the session-start launcher MUST emit the recap carrying the staleness advisory.
23. A behavior's trigger set MUST be identical across hosts; only the selecting mechanism — matcher or in-CLI filter — may differ.
24. WHEN a track gate produces a second document on a topic, the executing skill MUST assign each statement to one owning document per the content-kind ownership table in `skills/_shared/prd-contract.md`.

## Constraints & Invariants

- Constraint: the surface holds exactly four visible commands. A fifth requires a new ADR.
- Constraint: the plugin ships at most 2 agents. A third requires an ADR.
- Constraint: a PreToolUse hook MUST complete within 2 seconds, and a PostToolUse hook within 4 seconds, with enough margin that a host whose pre-mutation timeout fails open never reaches it.
- Constraint: a `SKILL.md` MUST NOT exceed 300 lines.
- Constraint: a track file MUST NOT exceed 200 lines.
- Constraint: a new host costs a manifest, a hooks config, a normalizer case, a resolvable path from that config to `bin/`, and enrollment in the coverage matrix — and no change to skills, agents, or `bin/` logic.
- Constraint: the `Makefile` lives at the repository root while the plugin lives in `plugins/archcore/`, so `make verify` runs from the repository root.
- Invariant: every user-facing entry point maps to one of the four commands.
- Invariant: adding a track changes one new track file plus one routing-table row per calling skill, and no other file.
- Invariant: every document mutation passes through the MCP tool layer, and every MCP mutation triggers PostToolUse validation.
- Invariant: every `update_document` triggers cascade detection in addition to validation.
- Invariant: every `create_document` and every `update_document` triggers the precision check.
- Invariant: every direct `.archcore/` markdown write attempt is blocked at PreToolUse on every host that supports pre-mutation hooks.
- Invariant: no tool call is ever denied by the plugin for a reason other than the CLI's verdict.
- Invariant: every session starts with project context loaded and the staleness check run, or with a warning when the CLI is missing.
- Invariant: no agent holds Write, Edit, or Bash access to a `.archcore/` file.
- Invariant: staleness detection never modifies a document autonomously; only the actualize and closeout tracks modify, and only on per-document confirmation.
- Invariant: every Archcore document type is reachable through at least one intent skill, or directly through MCP.
- Invariant: skill and agent content is byte-identical across hosts; only the container format differs.
- Invariant: one kind of statement has one owning document across the documents a track produces on one topic, per `prd-spec-plan-content-ownership.adr`.

## Failure Behavior

1. IF the MCP server is unavailable, THEN the skill MUST inform the user with install and init instructions. On Cursor and Copilot this is also the expected pre-wiring state, whose fix is `archcore init --agent <host>` rather than a plugin reinstall.
2. IF `create_document` fails on a duplicate, THEN the skill MUST suggest an alternative filename.
3. IF intent routing is ambiguous, THEN the skill MUST ask one scope-confirmation question.
4. IF routing stays ambiguous after that question, THEN the skill MUST fall back to `document`.
5. IF a track is interrupted mid-cascade, THEN the executing skill MUST resume from the `archcore:track` state block in the draft artifact, at the earliest gate whose exit checks have not passed.
6. IF a hook cannot reach its script, THEN the hook MUST exit 0 and name the script in a stderr warning. Enforcement is off for that session, and the warning is the only signal, which is why silence there is a defect.
7. WHEN a PostToolUse hook times out, the host fails open. PreToolUse timeout behavior is the host's: fail-closed on Claude Code, Codex, and Cursor, and fail-**open** on Copilot, which is why both PreToolUse guards are held far inside budget rather than trusted to the host.
8. IF an agent exceeds its turn limit, THEN the agent MUST return its partial results.
9. IF git is unavailable, THEN SessionStart MUST skip the staleness check.
10. IF git is unavailable, THEN the actualize track MUST skip code-drift analysis and still run the cascade and temporal analyses.

## Conformance

The architecture is conformant when:

1. Every document operation flows through MCP tools.
2. The skill surface is exactly `init`, `plan`, `document`, `review`, and all are auto-invocable.
3. The PreToolUse guard blocks every direct `.archcore/**/*.md` write, on every host, through that host's honored deny mechanism.
4. PostToolUse validation fires after every MCP document mutation.
5. PostToolUse cascade detection fires after every `update_document`.
6. PostToolUse precision check fires after every `create_document` and `update_document`.
7. No PostToolUse hook is registered for `Write|Edit`.
8. SessionStart emits the recap carrying the staleness advisory.
9. Gate logic for every multi-document flow lives under `skills/_shared/tracks/<track>.md`, and each track's gate records match its golden in `@test/fixtures/goldens/`.
10. Every Archcore document type is reachable through at least one intent skill.
11. The event matrix lists every implemented host, and each row is backed by a row in `@test/structure/host-coverage-matrix.bats`.
12. Each host's hook commands are proven to reach `bin/` by executing them rather than by inspecting them — the assertion that pinned Copilot's commands as strings matched a broken command exactly and shipped it.
