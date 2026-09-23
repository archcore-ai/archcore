---
title: "Open the init Selection Screen on Every Interactive Run"
status: draft
tags:
  - "cli"
  - "component:cli"
  - "integrations"
---

## Goal

Make interactive `archcore init` open the agent selection screen on every run, with the hosts `agents.Detect` found pre-checked, so a project that already carries `.claude/` can add hosts and consent to the plugin from init. Clauses 29–32 of the plugin-delivery spec are the contract; the ADR on the selection screen records the decision.

## Tasks

### Phase 1 — Routing and picker

1. Change `agentPicker` to `func(preselected []*agents.Agent) (agentSelection, error)` — @cli/cmd/init.go.
2. Add `selectAgentsForInit`: interactive calls `pickAgents(agents.Detect(baseDir))`, otherwise `resolveAgents(baseDir)` — @cli/cmd/init.go.
3. Keep `resolveAgents` detection-first for `hooks install`, `mcp install`, `instructions install`; its picker call becomes `pickAgents(nil)` — @cli/cmd/init.go.
4. In init's `RunE`, call `selectAgentsForInit(cwd)` in place of `resolveAgents(cwd)` — @cli/cmd/init.go.
5. In `defaultPickAgents`, seed `picked` with the preselected IDs before `.Value(&picked)` — @cli/cmd/init.go.
6. Give the picker a title for the pre-checked case and keep the existing one otherwise — @cli/cmd/init.go.
7. Rewrite the `resolveAgents` doc comment and the `outcomeDetected` doc comment for the non-interactive-only path — @cli/cmd/init.go.

### Phase 2 — Tests

8. Adapt `withPickAgents`, `withPickAgentsFn`, and `installFromPicker` to the new seam — @cli/cmd/init_test.go.
9. Fix the raw picker closures at `hooks_test.go:71` and `init_plugin_test.go:272` — @cli/cmd/hooks_test.go, @cli/cmd/init_plugin_test.go.
10. Add a test: `.claude/` present, interactive, picker receives Claude Code preselected, outcome `outcomePicked` — @cli/cmd/init_test.go.
11. Add a test: detected host, interactive, picker returns `outcomeSkipped`, nothing is wired — @cli/cmd/init_test.go.
12. Add a test: `.claude/` present, non-interactive, `selectAgentsForInit` returns `outcomeDetected` — @cli/cmd/init_test.go.
13. Add the interactive counterpart of `TestResolveAgentsSeparatesDetectionFromAPick` against `selectAgentsForInit` — @cli/cmd/init_plugin_test.go.
14. Add a test: `resolveAgents` with `.claude/` and a terminal still returns `outcomeDetected` — @cli/cmd/init_plugin_test.go.
15. Run `go test ./...` and `golangci-lint run ./...` from `cli/`.

### Phase 3 — Documents and user-facing text

16. Update the Full-integration sentence and the init entry-point bullet — @.archcore/integrations/supported-ai-agents.doc.md, via `update_document`.
17. Update lines 54–57 on detection and the picker — @.archcore/cli-ui/building-the-cli.doc.md, via `update_document`.
18. Update the `archcore init` row — @cli/README.md; update line 118 — @README.md.
19. Run `archcore init` in this repository by hand and record the screen outcome for `closeout.accept`.
20. File the docs.archcore.ai "Connect your agent" edit in `archcore-ai/landing` [assumption on its wording].

## Acceptance Criteria

- An interactive `archcore init` in a directory containing `.claude/` opens the selection screen with Claude Code checked before any host is wired.
- Confirming the pre-checked screen on a rerun wires the same hosts as v0.10.2 did and reports the plugin as already installed when it is.
- Unchecking a pre-checked host and confirming leaves that host's hook and MCP files untouched in that run.
- A run with no `/dev/tty` and no `--agent` wires detected hosts without a screen and prints one plugin hint, as v0.10.2 does.
- `archcore init --agent <id>` produces the same output as v0.10.2 for the same inputs.
- `archcore hooks install`, `archcore mcp install`, and `archcore instructions install` without `--agent` behave as in v0.10.2: detected hosts are acted on without a screen.
- `go test ./...` and `golangci-lint run ./...` pass from `cli/`.
- The scenario examples on the init screen hold on a manual run in this repository.

## Dependencies

- The ADR on the selection screen records the decision; clauses 29–32 of the plugin-delivery spec are the contract.
- Task 1 blocks tasks 2, 5, 8, and 9: the seam signature changes first.
- `resolveAgents` is shared with `hooks install`, `mcp install`, and `instructions install` (`@cli/cmd/hooks.go`, `@cli/cmd/mcp.go`, `@cli/cmd/instructions.go`); the decision covers init only, so init gets its own selector.
- Phase 3 follows Phase 2: the documents describe the shipped behavior, and the accepted documents change in the same commit as the code.
- huh v1.0.0 (`@cli/go.mod`): `MultiSelect.Value` marks the options already in the pointed slice as selected (`Accessor` in `field_multiselect.go`), so no library change is needed.
- Task 20 is out of this repository and does not block the release of the CLI change.

## Declared Delta

- creates: none.
- modifies: the plugin-delivery init step (selection screen and the consent it carries). Covering spec: plugin-delivery. Verdict `spec-wrong`: the spec's Surface line and two Invariants described detection-driven routing that the decision replaces; the spec was edited in this package after the user chose variant 1 with the spec edit named.
- retires: none.
- decision: open the selection screen on every interactive run and pre-check detected hosts (ADR in this package).
- intent_gap: no. The one-install-command intent is recorded in the accepted idea on delivering the plugin from init and in the keeping-the-cli-current PRD.
- Route rationale: `modifies` non-empty gives amendment (base S); zone maturity stone (an accepted spec with three dependents) raised the label to M; R none; instruments in order: decision, amendment verdict with the spec edit, illustrate, decompose.
