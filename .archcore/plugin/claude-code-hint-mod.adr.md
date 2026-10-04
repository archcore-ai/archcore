---
title: "Next-Step Hints Ship as a Claude Code Mod Inside the Archcore Plugin"
status: accepted
tags:
  - "component:plugin"
  - "hooks"
  - "integrations"
  - "plugin"
---

## Context

Claude Code 2.1.287 and later loads mods — TypeScript or JavaScript hooks modules a plugin names under `modules` in `hooks/hooks.json` — and runs them by default; Anthropic's mods overview states that "a mod can ship in the same plugin as a skill and an MCP server". The next-step hints mod shows one suggested Archcore command above the Claude Code prompt and offers it as the Tab suggestion, and the user asked for it to reach every Archcore user with the next plugin update. `stack-and-tooling.rule` items 7, 18 and 20 forbid TypeScript and a second test runner in `plugins/archcore/`, and `engine-runtime-boundary.adr` gives code alignment and staleness to the engine, so a runtime mod cannot compute coverage or drift. The decision hint matches phrases in the user's prompt, the technique `disable-stop-and-prompt-hooks.adr` removed from the CLI hooks because their output reached the agent.

## Decision

Ship the next-step hints as a Claude Code mod inside the published plugin — `hooks/next-step.tsx` named under `modules` in `@plugin/plugins/archcore/hooks/hooks.json`, its `$.state` contract in `@plugin/plugins/archcore/types/index.d.ts`, on by default behind the `next_step_hints` boolean `userConfig` option, tested with `claude plugin test` from `@plugin/plugins/archcore/tests/next-step.test.ts` — with no hint that computes document coverage or drift, and with the decision hint shown only to the user.

## Alternatives Considered

1. A separate `archcore-next-step` plugin in the same marketplace — rejected because the user asked for the hints to arrive with the plugin update, and a separate plugin needs its own install by each user.
2. A local experiment under `experiments/` loaded with `--plugin-dir` — superseded by this record because nothing under `experiments/` reaches a release (`@scripts/export-plugin.sh` copies only `plugin/` subtrees).
3. Ship the drift, uncovered-directory and closeout hints with their checks inside the mod — rejected because those checks recompute what `@cli/internal/advisory/code_alignment.go` and `@cli/internal/advisory/staleness.go` own, which `engine-runtime-boundary.adr` forbids ("Neither side reimplements the other's half").
4. Restore the hints as CLI `UserPromptSubmit` and `Stop` hooks — rejected because their output reaches the agent, the failure mode `disable-stop-and-prompt-hooks.adr` removed; the mod draws in the interface and never adds text to the model's context.

## Consequences

Positive:

- Every Archcore user on Claude Code 2.1.287 or later gets the hints with the next plugin update, without a separate install; the `/config` row "Next-step hints" turns them off, and a change reloads the module.
- Other hosts are unaffected: only the Claude Code `hooks/hooks.json` names the module; `cursor.hooks.json`, `codex.hooks.json` and `copilot.hooks.json` do not.
- `@plugin/test/structure/next-step-mod.bats` pins the wiring, the `userConfig` default, the TypeScript boundary, the absence of `path_ref` and `specificity` reads, and the export that drops the type declarations Claude Code generates.

Negative and trade-offs:

- The plugin tree gains TypeScript and a second test runner under the `stack-and-tooling.rule` exception (items 31–38); `make test` skips the mod tests where `claude` is not on `PATH`, which includes the current CI runners.
- [expected] The mod API is early access and changes between Claude Code releases; a breaking change shows as a module that does not load, reported in the `/plugin` Errors tab, while the rest of the plugin keeps working.
- [expected] The decision hint still shows on some non-decision prompts; the pattern accepts only explicit first-person decision statements and skips questions.
- [assumption] A Claude Code release older than 2.1.287 ignores the `modules` key and the `types` manifest field (the manifest reference strips unknown top-level fields), so the settings hooks keep working there.

## Superseded when

- The CLI exposes coverage and drift results for one file; then a new record adds the drift and uncovered-directory hints on top of that result.
- Within two plugin releases, more than 1 in 5 decision hints are hidden as wrong in local trials [assumption]; then the decision hint is removed.
- A Claude Code release makes `claude plugin validate --strict` or `claude plugin test` fail for the mod without a code change.