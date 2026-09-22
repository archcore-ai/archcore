---
title: "In-Process MCP Integration Tests as Layer A of the E2E Strategy"
status: accepted
tags:
  - "code-quality"
  - "golang"
  - "testing"
---

## Context

Per-tool unit tests under `internal/mcp/tools/*_test.go` invoke handlers directly via a `callTool(handler, args)` helper (`internal/mcp/tools/list_documents_test.go:13`). They are thorough at the algorithm level — `search_documents_test.go` alone is 610 lines covering 22 cases — but they bypass three production-code paths:

1. **Tool registration** through `server.MCPServer.AddTool` in `internal/mcp/server.go:NewServer`. A deleted `s.AddTool(...)` line ships green because every per-tool test calls the handler directly.
2. **JSON marshalling** through the `mcp-go` client/server layer. A breaking change to a request/response shape is not caught.
3. **Multi-tool composition** — where one handler writes state (filesystem or `.archcore/.sync-state.json` manifest) and another reads it. The cross-tool coupling between `add_relation` (strips the `.archcore/` prefix at `internal/mcp/tools/add_relation.go:14`) and `search_documents` (reconstructs the prefix at `internal/mcp/tools/search_documents.go:284,291`) is the canonical example: both unit suites pass against hand-written manifests, but a regression at either site silently breaks the loop in production.

The new `search_documents` tool (8 filter parameters, complex specificity scoring, manifest-aware relation enrichment) is the immediate motivator: its algorithmic surface is fully covered by unit tests, but its composition surface with `create_document`, `update_document`, `add_relation`, and `remove_document` was not.

## Decision

Add an in-process integration test layer at `internal/mcp/integration/`. The layer:

- Wires the real `mcp.NewServer(baseDir)` to a real `client.NewInProcessClient` — the same client API a subprocess test would use, with no stdio framing and no subprocess involved.
- Lives in a new package, **not** behind a build tag — sub-second runtime is cheap enough to run on every `go test ./...`.
- Exercises **multi-tool scenarios**, not single-tool happy paths. Composition is the only coverage this layer adds over the existing unit suite; redoing single-tool algorithm checks here would be waste.
- Uses the in-process client for both setup and assertion (the JSON round-trip is part of what's tested); falls back to direct manifest reads only as corroborating assertions on mutation scenarios — a divergence between client view and on-disk manifest pinpoints the failure site.

Initial scope, as decided: 7 scenarios across 4 files (`harness_test.go`, `lifecycle_test.go`, `relations_test.go`, `tags_test.go`):

1. Tool-registration canary asserting every registered tool name.
2. `init_project` → `create_document` → `get_document` → `list_documents` round-trip.
3. `create` × 2 → `add_relation` → `search_documents` cross-tool with `.archcore/` prefix reconstruction.
4. `create` + `add_relation` → `remove_document` → `CleanupRelations` runs and persists to the manifest.
5. `add_relation` accepts both prefixed and unprefixed path forms with byte-identical manifest state.
6. `remove_relation` undoes `add_relation` — both the client view and the manifest see no relation.
7. `update_document` three-way tag semantics: omit preserves, empty array clears, non-empty array replaces.

A small, reusable harness (`initArchcore`, `newTestClient`, `mustCallTool`, `expectToolError`, `decodeJSON[T]`, `loadManifest`, `createADR`) is the contract future composition tests build on; setup-by-fixture is reserved for the rare cases where the in-process client cannot produce the required state.

The canary asserts the registered set, not a number. The count in that assertion moves whenever a tool is added, and this document does not track it — `cli-ui/building-the-cli.doc` names the current surface.

## Alternatives Considered

- **Expand existing per-tool unit tests** to cover composition. Rejected: cross-tool setup pollutes single-tool test files with state that has nothing to do with the tool under test, and `tools/*_test.go` files become unwieldy. Composition is its own concern and deserves its own location.
- **Subprocess-based integration via `archcore mcp` over real stdio** (Layer B in `code-quality/e2e-testing-for-cli.idea.md`). Rejected for now: the bugs it adds coverage for (stdio framing, command wiring, Windows path edge cases) are largely covered upstream by `mcp-go` itself; ROI is lower than Layer A and adds build-time + binary-path complexity. Reconsider when a concrete regression demands it.
- **Real-agent E2E via Claude / Cursor** (Layer C). Rejected for now: token cost, flakiness, and the fact that Layer A catches a strict superset of the deterministic regressions we currently know how to write tests for. Layer C remains valuable for prompt-regression detection (e.g., "did the agent stop using `path_ref` after we shortened the description") and is preserved as future work in the parent idea doc.
- **Behind a `//go:build integration` tag.** Rejected: gating adds zero defensive value for in-process tests (no API calls, no subprocess, no flakiness) but means the per-commit `go test ./...` loop misses exactly the regressions this layer exists to catch.

## Consequences

- **Per-commit test runtime grows ~0.5s.** New layer is in-process, fully parallelized with `t.Parallel()`, and uses `t.TempDir()` per test — no shared state, no ordering coupling, clean under `-race`.
- **No production code changed and no new dependencies added.** `mcp-go` was already pinned at the time of this decision (v0.44.0), and `client.NewInProcessClient` shipped upstream and required no shim. The pin has since moved; the decision does not depend on the version and this document does not track it. `go.mod` is the source of truth.
- **Future tools and tool-shape changes get a low-cost regression net.** Adding a tool means adding one entry to `expectedTools` in `lifecycle_test.go` and (optionally) one composition scenario; renaming a JSON field on a response struct surfaces in the next test run instead of in agent confusion days later.
- **Layer B and Layer C from `e2e-testing-for-cli.idea.md` remain valid future work** — Layer A explicitly does NOT cover stdio framing, CLI command wiring, agent prompt comprehension, or system-instruction effectiveness. Defer those layers until a concrete regression motivates them.
- **Establishes the harness pattern.** Future composition tests (e.g., when relation cascades or sync-state migrations land) should extend `internal/mcp/integration/` rather than duplicate setup elsewhere.
- **The layer grew as intended.** It stands at 11 files as of September 2026, adding `globals_test.go`, `globals_worktree_test.go`, `concurrency_test.go`, `forward_compat_test.go`, `roots_test.go`, `prompts_test.go`, and `rnd_test.go` — all composition-level, which is the shape this decision asked for.
