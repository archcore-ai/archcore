---
title: "Who Triggers a CLI Update"
status: draft
tags:
  - "cli"
  - "integrations"
  - "update"
---

## Summary

The update trigger lives inside `archcore mcp`. After the server begins serving, a background goroutine waits out a 60 s delay and then invokes a guarded update policy. The server never blocks on it, never restarts, and keeps serving the image it was launched with; the new binary takes effect the next time anything launches `archcore`.

The caller supplies the moment; the binary supplies every condition. Both live in this repository, so no plugin release has to stay in step with a binary release.

## Motivation

Nothing updates the CLI unless a user types `archcore update`. The only nudge is the plugin's SessionStart advisory. Two constraints decide where an automatic trigger can go.

**Not every user is on the plugin.** `archcore init --agent` writes project-level hooks — `@internal/wiring/hooks_agents.go` carries SessionStart rows for claude-shaped hosts, Cursor, Gemini CLI, and Copilot — and those call `archcore hooks <host> session-start`, the CLI's own leaf. The plugin's `bin/session-start` delegates to the same leaf. A plugin-owned trigger therefore serves a subset of the population the CLI already runs in. The MCP surface is wider still: the plugin ships an entry on Claude Code, Cursor, and Codex, and `archcore init` writes one per its agent registry.

**No hook event is a good moment.** Three events are wired on every host: SessionStart, PreToolUse, PostToolUse. The last two carry 2 s and 4 s budgets. SessionStart is not a session boundary — it carries a source of `startup`, `resume`, `clear`, or `compact` (`@cmd/hook_command.go:128`), and `compact` fires while a user is deep in a long task. Every hook process is short-lived, so any work it does is work someone waits for.

The MCP server has neither problem. It is long-lived, and its stdout discipline is already absolute — every human-readable line in `@cmd/mcp.go` goes to stderr.

## Detailed Design

**The split.** The trigger answers "when": after the server is serving, once per process, after a delay, on a goroutine, never blocking startup or shutdown. The policy answers "whether": a development build, a CI environment, a claim already held, an explicit opt-out, and a cache showing nothing newer each stop the attempt before any download; write access is verified before any bytes are fetched. Both are specified separately.

**Which commands carry what.** Three tiers, covering the whole command surface.

| Tier | Commands | Behavior |
|---|---|---|
| Attempts a replacement | `mcp` | Background goroutine, 60 s delay, silent on stderr |
| Reports only | `doctor` | One cached advisory line in the health report; never writes a binary |
| Untouched | `init`, `config`, `hooks`, `instructions`, `status`, `sync`, `update` | No change |

`doctor` gains a report because a stale CLI is a health fact, because it reuses `runUpdateCheck` unchanged (24 h cache, 2 s timeout, silent on failure, exit code untouched), and because it is the one path a terminal-only user reaches without an agent. `hooks` is excluded — its stdout is the host's protocol channel and its budget is seconds. `status` is excluded — it reports document structure, not tooling health. The rest are short commands whose output a user is waiting on.

**What changes in the plugin: nothing structural.** No new hook row, no new `bin/` script, no change to `.claude.mcp.json`. The plugin keeps exactly one job here: the case where the MCP server cannot start. `checkGlobals` aborts `RunE` before `RunStdio` is reached, so a project with a broken global mount never reaches the goroutine, and the SessionStart hook is then the only Archcore code still running. The existing advisory in `bin/session-start`, gated by `bin/cli-gte` and rate-limited by its 24 h `last-update-advisory` stamp, covers that case and needs no edit. Its impression rate doubles as the control signal: if the silent path works, impressions decay on their own.

**Why the process is not restarted.** Replacing the file does not disturb the running server: on POSIX it keeps the inode it opened, and on Windows renaming a running image is permitted — the property `atomicReplace` already depends on (`@internal/update/update.go:536`). Restarting would drop the JSON-RPC connection the host depends on, for a benefit that arrives anyway at the next launch.

**What phase 1 leaves out.** A provenance receipt — refusing to replace a binary a package manager owns — is specified but deferred. No Homebrew tap, Scoop manifest, or winget package exists today, so the guard has nothing to distinguish against, and the opt-out variable covers anyone who disagrees. It activates with the first package-manager channel.

## Drawbacks

- **The benefit lands at the next launch.** Accepted deliberately: sessions end, and the next initialization happens on its own.
- **A session shorter than 60 s never updates.** The goroutine is cancelled before it runs. Over many sessions this self-corrects; a user whose sessions are consistently brief updates rarely.
- **An attempt in flight dies with the process.** The server does not wait at shutdown, so a session ending mid-download abandons the work and leaves a temporary file. The running binary is never at risk — the rename is the last step — and the next attempt sweeps the leftover.
- **No runtime disclosure reaches the user.** The one stderr line lands in the host's server log, so the published privacy page is the only user-facing statement that the binary updates itself.
- **The claim window doubles as a failure backoff.** One transient network failure suppresses retries for the whole 24 h window.
- **A broken project mount disables the trigger.** The one case where the plugin advisory is the only remaining actor is also the case where the user most needs a working CLI.
- **A terminal-only user gets a report, not an update.** `doctor` tells them; it does not act for them.

## Alternatives

| # | Trigger site | Coverage | Blocks the user | Effect lands | Verdict |
|---|---|---|---|---|---|
| A | Manual only | n/a | none | when the user acts | Current state — the gap this addresses |
| B | Any CLI command | everyone | yes, on unrelated work | immediately | Rejected — protocol risk, no session concept, and new concurrency machinery needed |
| C | Plugin hook running `archcore update` as-is | plugin users | yes, minutes worst case | next session | Rejected — the banner goes to stdout, which is the hook's protocol channel; the 24 h cache is never read, since it lives only in `runUpdateCheck`; the HTTP client allows 60 s per request across three requests; and `NeedsUpdate` treats `dev` as behind, so every locally built binary self-replaces |
| D | Plugin hook running a guarded entry point | plugin users | bounded | next session | Rejected — narrower reach, and the trigger policy would ship from a repository that releases independently of the binary it drives |
| E | The agent offers, the user consents | any host with an agent | none | when the user accepts | Kept alongside I — the only option where the user sees the decision, and the advisory already reaches the model's context |
| F | Only to clear a hard version gate | plugin users | seconds, when already broken | immediately | Kept as the plugin's residual job — the broken-CLI case above |
| G | A package manager | its own users | none | on the user's schedule | Endgame, blocked on channels that do not exist yet |
| H | Staged download, swap at next launch | everyone | none | next launch | Rejected — the swap has to happen inside some other command's startup. Revisit if a versioned install layout is adopted |
| I | **`archcore mcp`, background goroutine** | **every wired host, plugin or not** | **none** | next launch | **Proposed** |

Every option except B and C is concurrency-safe under the policy's binary-path claim.
