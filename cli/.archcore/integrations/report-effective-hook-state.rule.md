---
title: "Report Whether a Written Host Config Can Take Effect"
status: accepted
tags:
  - "cli"
  - "integrations"
---

## Rule

Writing a config file and reporting success is honest only when the host will read it.

1. WHEN `hooks install` finishes writing a host config, the CLI MUST report whether that host can run
   the hooks as written.
2. IF a host cannot run the written hooks, THEN the CLI MUST state that the file was written, state
   why the hooks are inert, and name the action that enables them.
3. WHEN `doctor` inspects a wired host, the CLI MUST apply requirements 1 and 2.
4. WHEN the CLI wires Codex CLI, the CLI MUST report the state of the hooks feature flag, the Windows
   limitation, and the `.codex/` trust requirement.
5. WHEN the CLI wires Copilot, the CLI MUST report that pre-write context injection is unavailable on
   that host.
6. WHEN the CLI wires Gemini CLI, the CLI MUST report that its tool events come from the published
   reference and are not confirmed against a running host.
7. IF the CLI cannot read a host's configuration to determine the answer, THEN the CLI MUST treat the
   host as unable to run the hooks.
8. The CLI MUST print nothing for a host whose wiring works as installed.
9. A host counts as wired only when its config file holds a hook command archcore wrote. `doctor`
   MUST skip every other host, and MUST NOT claim the wired hosts are healthy when it examined none.

## Rationale

Three hosts accept a config file and then do nothing with it, each for a different reason. Codex
keeps hooks behind an experimental flag that is off by default. Copilot's `preToolUse` event carries
only a permission decision, so context emitted there is discarded. Gemini's event names come from a
document rather than a probe.

Reporting "installed" in any of these cases is a claim the user discovers is false weeks later, while
wondering why nothing ever fires. The failure is silent by construction: no error appears anywhere,
because nothing ran.

Requirement 7 follows from what the question means. "Can this run?" answered from an unreadable
config is not "yes" — it is "unknown", and the useful default for a diagnostic is the answer that
sends the reader to look.

Requirement 8 keeps the notes readable. A message printed for every host on every install becomes
scenery, and the one host with a real problem stops standing out.

Requirement 9 exists because agent detection answers a different question. `agents.Detect` reports
which hosts the repository uses, from the presence of a `.claude/` or `.codex/` directory — which
says nothing about whether archcore ever wrote hooks into them. Reporting from that list warned about
Codex configuration in a project that had never installed a Codex hook, and printed the green
summary line in a project with no wiring at all. Both describe wiring that does not exist.

## Examples

Non-normative examples.

### Good

```
Codex CLI: hooks are written but will not run — the hooks feature is experimental and off by
default. Enable it with `codex --enable hooks`, or add `[features]` with `hooks = true`
(`codex_hooks = true` before Codex 0.129.0) to ~/.codex/config.toml.
```

### Bad

```
✓ Codex CLI hooks installed
```

The file exists. Nothing will read it.

```
✓ Wired hosts can act on their hook configs
```

Printed in a project with no host wired. The claim is about an empty set.

## Enforcement

- `@internal/wiring/hooks_effective.go` — `DescribeEffectiveHooks` returns the notes for one agent;
  an empty result means the wiring works. `DescribePluginConflict` describes the machine, not a host,
  so a caller covering several agents reports it once.
- `@internal/wiring/hooks_agents.go` — `HookConfigPath` and `CarriesArchcoreHooks` answer "is this
  host wired?" from the same path map the installers write through, so the two cannot drift.
- `@cmd/hooks.go` — `printEffectiveHookNotes`, called after every successful hook install. The host
  was just written to, so it needs no wired check.
- `@cmd/doctor.go` — `reportEffectiveHooks`, which applies requirement 9 before reporting.
- A host added to `hooksInstallers` without a matching entry in `hookConfigPaths` is never reported
  as wired, and one added without a review of `DescribeEffectiveHooks` reports nothing — which
  requirement 1 permits only when the wiring works as installed.
