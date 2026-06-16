---
title: "Implementation Plan: MCP Track Prompts"
status: accepted
tags:
  - "mcp"
---

## Goal

Implement the five MCP track prompts (`iso_track`, `sources_track`, `product_track`, `standard_track`, `architecture_track`) per `.archcore/mcp/mcp-prompts-for-tracks-only.adr.md`. The "what" and "why" live in the ADR; the "how to add a prompt" lives in `.archcore/mcp/adding-mcp-track-prompts.guide.md`. This plan captures only the sequencing, structural decisions, and open-question outcomes for the rollout itself.

## Tasks (phased)

### Phase 1 — Wiring proof (sequential)

1. Create `internal/mcp/prompts/` package with skeleton: `prompts.go` (`RegisterAll`), `args.go` (`requireStringArg`, `optionalStringArg`), `messages.go` (`framingMessage`, `phaseMessage`, `confirmationGate` constant, `verificationMessage`).
2. Implement `iso_track.go` (BRS → StRS → SyRS → SRS, three `implements` edges, six `PromptMessage`s).
3. Wire `internal/mcp/server.go`: add `server.WithPromptCapabilities(true)` to `NewServer`, call `prompts.RegisterAll(s)` after the `AddTool` block.
4. Add unit tests: `args_test.go`, `messages_test.go`, `iso_track_test.go`, `register_test.go` (expecting 1 prompt at this stage).
5. Run `go test ./internal/mcp/...` — must be green before Phase 2.

### Phase 2 — Remaining tracks (parallelizable)

Helpers from Phase 1 are frozen. The four track files can be implemented in parallel:

6. `sources_track.go` + `sources_track_test.go` — MRD → BRD → URD, two `related` edges (peers, not `implements`; canonical per `internal/mcp/server.go:123-134`).
7. `product_track.go` + `product_track_test.go` — PRD → Plan, `plan implements prd`.
8. `standard_track.go` + `standard_track_test.go` — ADR → Rule → Guide; `rule implements adr`, `guide related rule`.
9. `architecture_track.go` + `architecture_track_test.go` — ADR → Spec → Plan; `spec implements adr`, `plan implements spec`. Optional argument `component_name` extends the framing message.

10. Update `register_test.go` to expect all 5 prompts; finalize `RegisterAll`.

### Phase 3 — Integration tests (after Phase 2)

11. Create `internal/mcp/integration/prompts_test.go`:
    - `ListPrompts` → exactly 5 names, every prompt has `feature_name` required argument, every description contains `→`.
    - `GetPrompt` per track: assert message count, first-message role, doc-type substrings in messages, `list_relations` in final message.
    - Negative: `GetPrompt` with empty arguments — observe and tighten.
    - Optional-arg subtest for `architecture_track` with/without `component_name`.
    - Initialize round-trip: assert `result.Capabilities.Prompts != nil`.
12. `internal/mcp/integration/lifecycle_test.go` — no change to `expectedTools`; canary stays.

### Phase 4 — Instructions & finalization

13. Append `WORKFLOW PROMPTS` block to `mcpServerInstructions` in `internal/mcp/server.go`. Exact wording (six lines, ~70 tokens), placed after `CODE REFERENCES`, before the closing `NEVER create…` paragraph:

    ```
    WORKFLOW PROMPTS (when client supports MCP prompts):
      iso_track          — BRS → StRS → SyRS → SRS cascade
      sources_track      — MRD → BRD → URD discovery flow
      product_track      — PRD → plan
      standard_track     — ADR → rule → guide
      architecture_track — ADR → spec → plan
    ```

14. Extend `internal/mcp/server_test.go`:
    - Block presence in `buildInstructions("")` and `buildInstructions("ru")`.
    - `strings.Count(out, "WORKFLOW PROMPTS") == 1` for both languages.
    - All five prompt names appear.
15. Final `go test ./...` and `go vet ./...`. Commit per phase or as one PR — implementer's choice.

## Acceptance Criteria

- `go test ./...` and `go vet ./...` are green.
- `go build -o archcore .` succeeds.
- A connected MCP client sees exactly 5 prompts via `prompts/list`.
- Each `GetPrompt` returns the expected `Description`, message count, and message structure (covered by integration tests).
- `mcpServerInstructions` contains the `WORKFLOW PROMPTS` block exactly once per language.
- No new dependencies (`go.mod` unchanged; `mcp-go v0.49.0` already pins required APIs).
- No tool registration regressions (`expectedTools` canary in `lifecycle_test.go` still green).

## Dependencies

- `.archcore/mcp/mcp-prompts-for-tracks-only.adr.md` — accepted, defines scope.
- `.archcore/mcp/mcp-prompts-orchestration-only.rule.md` — accepted, constrains future additions.
- `.archcore/mcp/adding-mcp-track-prompts.guide.md` — accepted, step-by-step reference (note: argument-map type and `Role` constants need correction — see Implementation Notes).
- `.archcore/code-quality/in-process-mcp-integration-tests.adr.md` — defines the integration test harness this plan extends.

## Implementation Notes

Resolved during planning, must be honored during implementation:

1. **Argument map type.** `mcp-go v0.49.0` exposes `GetPromptParams.Arguments` as `map[string]string`, not `map[string]interface{}`. The guide needs a follow-up edit; implementation uses the correct string-map form.
2. **Role constants.** `mcp-go` defines only `RoleUser` and `RoleAssistant`. The "system framing" message uses `RoleUser` with explicit "You are running…" prose. Add a code comment explaining the mapping to avoid future churn.
3. **Capability assertion.** Lives in `internal/mcp/integration/prompts_test.go`, not in `internal/mcp/server_test.go` — avoids importing the in-process client into the `mcp` package.
4. **Missing required arg behavior.** `mcp-go` may either reject at the protocol layer or pass through to the handler. Integration test starts permissive; tighten after observation.
5. **Instruction block placement.** Inside the existing `var mcpServerInstructions` block, six lines, ~70 tokens. Acceptable per `cli/mcp-token-optimization.idea.md` budget.
6. **Naming collision with plugin skill.** Out of code scope. PR description must remind reviewers that `/archcore:iso-track` (plugin) and `/mcp__archcore__iso_track` (this work) coexist by design.

Key files: @internal/mcp/server.go, @internal/mcp/prompts/, @internal/mcp/integration/prompts_test.go, @internal/mcp/server_test.go.
