---
title: "Agent Context Delivery: Bench, File Context Engine, search_documents for_path, Address in Every Channel"
status: draft
tags:
  - "component:cli"
  - "component:plugin"
  - "hooks"
  - "integrations"
  - "mcp"
  - "multi-host"
---

## Goal

Ship the two capabilities of the agent-context-delivery initiative — the file-context result and the context address in every channel — and measure them with one bench before and after the change.

## Tasks

### Phase 0 — Bench baseline

1. Build a bench fixture: test rules, one pathless rule, one root config doc, no hand-written `.archcore/` section. @plugin/test/behavioral/fixtures/
2. Add three bench tasks: a unit test, an edit under a pathless rule, a root config edit. @plugin/test/behavioral/edits-bench.py, @plugin/test/behavioral/benchlib.py
3. Score each transcript: rule read and MCP call before the first write, hint documents opened. @plugin/test/behavioral/benchlib.py
4. Run the bench on the current CLI and record the baseline in the bench output directory.

### Phase 1 — File-context engine

5. Move `extractPathRefs` and `filterBareMentions` into `internal/docs`, so the advisory package can import them. @cli/internal/mcp/tools/search_documents.go, @cli/internal/docs/
6. Return a structured file-context result from the engine: reasons, ranking, general rows, caps, remainder lines. @cli/internal/advisory/code_alignment.go
7. Add `doc` to the ranked types and measure the hook time on the 302-document monorepo against the one-second budget. @cli/internal/advisory/code_alignment.go
8. Render the hook text from the structured result: header line, `.archcore/` paths, remainder lines. @cli/internal/advisory/code_alignment.go, @cli/cmd/hook_command.go
9. Suppress a repeated identical result within one session through an `internal/stamp` claim. @cli/internal/stamp/, @cli/cmd/hook_command.go
10. Test each file context clause, including the FullLayout case and a file outside the source roots. @cli/internal/advisory/code_alignment_test.go, @cli/cmd/hook_code_alignment_handler_test.go

### Phase 2 — `search_documents` `for_path`

11. Add the `for_path` parameter and the `reason` row field; reuse the engine result. @cli/internal/mcp/tools/search_documents.go
12. Rewrite the `search_documents` description to explain `for_path` inside 2 048 characters. @cli/internal/mcp/tools/search_documents.go, @cli/internal/mcp/tool_description_spec_test.go
13. Add the parity test: hook and `for_path` return the same matched rows for one fixture. @cli/internal/mcp/integration/

### Phase 3 — Context address in every channel

14. Add the address constant and rebuild the managed block around it; insert new blocks at the file top. @cli/internal/agents/instructions.go, @cli/internal/agents/instructions_test.go
15. Regenerate the instruction fixtures. @cli/scripts/regen-examples.sh, @cli/examples/
16. Cut the server instructions to 2 048 characters, address and notices first, under a size test. @cli/internal/mcp/server.go, @cli/internal/mcp/server_test.go
17. Move the type catalog and relation guidance into tool descriptions and skills, each under 2 048 characters. @cli/internal/mcp/tools/create_document.go, @cli/internal/mcp/tools/add_relation.go, @plugin/plugins/archcore/skills/_shared/
18. Put the address first in the session-start recap and drop the closing pointer. @cli/cmd/hooks_common.go, @cli/cmd/hook_session_start_test.go, @cli/cmd/hooks_common_test.go
19. Rewrite the plugin CLI-missing message around the address file form; update the goldens. @plugin/plugins/archcore/bin/session-start, @plugin/test/unit/session-start-goldens.bats, @plugin/test/unit/session-start-codex-stdout.bats
20. Add a test that compares the plugin copy of the address with the CLI constant. @plugin/test/unit/
21. Remove the claim that server instructions are always in context from the assistant agent files. @plugin/plugins/archcore/agents/archcore-assistant.md, @plugin/plugins/archcore/copilot-agents/archcore-assistant.agent.md, @plugin/plugins/archcore/agents/archcore-assistant.toml

### Phase 4 — Canon sync and measurement

22. Update the advisory subsystem doc in `architecture/` for reasons, general rows, `doc`, and the new bounds.
23. Rerun the bench on the changed CLI and plugin, and record the result beside the baseline.
24. Rerun the marker probe on Claude Code, Codex, and Copilot CLI with the real server instructions.

## Acceptance Criteria

- `go test ./...` and `golangci-lint run ./...` pass in `cli/`.
- `make test-plugin` passes, the bats goldens included.
- `TestShippedInstructionFixtures` passes after the fixture regeneration.
- The bench report holds baseline and after runs for the same tasks, model, and run count.
- The marker probe report names each host and the address markers it delivered.
- `/archcore:review` on the branch reports no `code-wrong` finding against the file context spec, the context address spec, the `search_documents` spec, or the session-start spec.

## Dependencies

- Claude Code 2.1.295, Codex CLI 0.159.0, and Copilot CLI 1.0.93 for the probe and the bench (installed 2026-10-09).
- A Claude model budget for the bench: the pilot cost about 3.4 USD for 12 runs [assumption: the bench needs 3 to 5 times that].
- Task 5 must land before tasks 6 and 11.
- The accepted `search_documents` and session-start specs already describe the new behavior (edited 2026-10-09 at the user's request), so `/archcore:review` reports `code-wrong` on them until Phases 2 and 3 land.

## Declared Delta

- `creates=[file-context-resolution, context-address-delivery]`; `modifies=[search-documents, session-start-context]`; `retires=[]`; `decision=[address placement and server-instruction budget, file context as search_documents mode, pathless rule is general]`; `intent_gap=yes`.
- Route: umbrella, size XL. Two consumer-relied behaviors, so one `prd` and one `spec` per capability. M=stone: accepted hook-runtime, session-start, and search contracts cover the zone. R=external-contract: agents on every host consume the hook text, the `search_documents` surface, and the managed block. One raise step covers both flags.
- Π: machine for code paths and probe results; world and empirical gathered in the `rnd`; undecided and user resolved by four answers on 2026-10-09.
- Verdicts: `search-documents` spec-wrong and `session-start-context` spec-wrong; both specs edited on 2026-10-09 after the user's confirmation.