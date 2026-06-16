---
title: "E2E Testing Strategy for Archcore CLI"
status: accepted
tags:
  - "code-quality"
  - "testing"
---

## Status

**Layer A (Tier 1 in this idea's terminology) shipped 2026-04-23** as in-process MCP integration tests at `internal/mcp/integration/`. See `code-quality/in-process-mcp-integration-tests.adr.md` for the decision record.

The shipped Layer A diverges from the original Tier 1 sketch below in one mechanism: tests run **in-process** via `mcp-go`'s `client.NewInProcessClient` (sub-second, no subprocess, no build tag) rather than calling `archcore mcp` as a subprocess over real stdio. The subprocess approach remains valid as a future Layer B for catching CLI command wiring + stdio framing bugs. Tier 2 (Agent) remains future work.

The text below preserves the original idea as historical record; its Tier 1 design is **not** the current implementation.

## Idea

Add end-to-end tests that verify the full user journey through archcore CLI — from `init` through hook-driven agent context delivery and MCP tool interactions — complementing the existing unit test suite which validates individual functions in isolation.

The approach is split into two tiers:

- **Tier 1 — Canary** (deterministic, no LLM, runs in CI on every commit): exercises the CLI commands and MCP tools as real subprocesses/JSON-RPC calls against a real git repository and filesystem, with no AI agent involved.
- **Tier 2 — Agent** (non-deterministic, costs tokens, runs on-demand): sends natural-language prompts to real AI agents (Claude Code, Cursor) and asserts that the agent correctly discovers, creates, and modifies documents through MCP tools.

Inspired by [entire.io's e2e suite](reference-materials/entireio/e2e/) which tests their CLI against 7+ real AI agents using a `ForEachAgent` matrix and a deterministic canary agent (Vogon).

## Value

### What unit tests cannot cover

The existing unit test suite (12,500+ LOC, 1.65x test-to-code ratio) thoroughly validates individual Go functions: config parsing, template generation, frontmatter extraction, API client, manifest operations, hook JSON structure. But archcore's **primary user is an AI agent**, not a human typing commands. The critical paths that unit tests structurally miss:

1. **MCP tool chain integration.** Units test `HandleCreateDocument` as a Go function. They don't test that the MCP server starts, registers tools, accepts JSON-RPC calls, and returns parseable responses — the actual protocol path an agent uses.

2. **Hook → context → agent behavior loop.** Units test that `session-start` produces JSON with `additionalContext`. They don't test that Claude Code actually invokes the hook, receives the context, and uses it to inform MCP tool calls. A subtle change in hook output format or timing can silently break the loop.

3. **`init` → hooks → mcp → agent-ready pipeline.** No test verifies that after `archcore init`, the hooks are installed, MCP server is registered, and an agent starting a session immediately sees the project's documents. This is the first-run experience — if it breaks, users get zero value.

4. **Cross-command consistency.** `create_document` → `list_documents` → `get_document` → `update_document` → `status` → `doctor` as a sequence through real filesystem state. Units test each in isolation with hand-crafted fixtures; no test verifies they compose correctly on a real `.archcore/` directory evolving over time.

5. **System instruction effectiveness.** MCP tool descriptions and server instructions guide the agent's behavior. Rewording a description can cause the agent to misuse a tool (wrong parameters, wrong tool choice). Only a real agent test catches this.

### Why two tiers

|                 | Canary (Tier 1)                        | Agent (Tier 2)                            |
| --------------- | -------------------------------------- | ----------------------------------------- |
| Cost            | Zero (no API calls)                    | Tokens per run                            |
| Determinism     | 100% reproducible                      | Non-deterministic (LLM)                   |
| CI frequency    | Every commit                           | On-demand / nightly                       |
| Coverage        | CLI + MCP protocol + filesystem        | Agent understanding of tools              |
| Catches         | Integration bugs, protocol regressions | System instruction regressions, UX issues |
| Estimated value | ~80% of integration risk               | ~20% but highest-severity issues          |

Canary alone covers 80% of integration risk at zero cost. Agent tests cover the remaining 20% — but that 20% includes the hardest-to-detect issues (agent misunderstanding tool semantics, broken context delivery).

## Possible Implementation

### Build tag isolation

```go
//go:build e2e

package e2e
```

All e2e tests live under a top-level `e2e/` directory with the `e2e` build tag. They are excluded from `go test ./...` and run via a dedicated command:

```bash
# Canary only (CI)
go test -tags=e2e ./e2e/... -run TestCanary

# Agent tests (on-demand)
E2E_AGENT=claude-code go test -tags=e2e ./e2e/... -run TestAgent -timeout 10m
```

### Tier 1 — Canary scenarios

Canary tests call the archcore binary as a subprocess and MCP tools via direct JSON-RPC to a locally-started MCP server process. No AI agent, no LLM tokens.

**Scenario: Init-to-ready pipeline**

```
archcore init (non-interactive, via flags)
  → .archcore/ directory exists
  → .archcore/.sync-state.json exists with version: 1
  → settings.json has correct sync mode
  → hooks installed for detected agent
  → MCP server config written
  → start MCP server subprocess
  → call list_documents → empty array
  → call create_document (type: adr, filename: test-decision)
  → call list_documents → 1 document
  → call get_document → correct content with template sections
  → archcore status → exit 0, no validation errors
  → archcore doctor → exit 0, no issues
```

**Scenario: Full document lifecycle**

```
create_document (adr, "use-postgres")
  → create_document (prd, "data-layer-requirements")
  → add_relation (adr implements prd)
  → list_relations → 1 relation
  → get_document (adr) → outgoing_relations includes prd
  → get_document (prd) → incoming_relations includes adr
  → update_document (adr, status: accepted, add tags: ["backend"])
  → get_document (adr) → status=accepted, tags=["backend"]
  → list_documents (tags: ["backend"]) → 1 result
  → list_documents (tags: ["frontend"]) → 0 results
  → remove_relation (adr implements prd)
  → list_relations → 0 relations
  → remove_document (adr)
  → list_documents → 1 document (only prd remains)
  → archcore status → clean
  → archcore doctor → no dangling relations
```

**Scenario: Session-start context delivery**

```
Create 5 documents across different types/directories with tags
  → add 3 relations between them
  → invoke: archcore hooks claude-code session-start (with stdin JSON)
  → parse stdout JSON
  → assert additionalContext contains all 5 documents
  → assert tags are aggregated
  → assert relations count matches
  → assert MCP tool references are present
```

**Scenario: Status catches real problems**

```
Manually create a file with invalid filename format
  → archcore status → exit 1, reports filename error
Manually create a file with missing frontmatter title
  → archcore status → exit 1, reports frontmatter error
Manually corrupt .sync-state.json
  → archcore status → exit 1, reports manifest error
  → archcore doctor --fix → repairs manifest
  → archcore status → exit 0
```

**Scenario: Config roundtrip**

```
archcore config set sync cloud
  → archcore config get sync → "cloud"
  → archcore config set project_id "proj-123"
  → archcore config get project_id → "proj-123"
  → archcore config set language "ru"
  → archcore config get language → "ru"
```

**Scenario: Multi-agent hook installation**

```
archcore hooks install --agent claude-code
  → .claude/settings.json contains hook commands
archcore hooks install --agent cursor
  → .cursor/hooks.json contains hook commands
archcore hooks install --agent gemini-cli
  → .gemini/settings.json contains hook commands
  → each agent's MCP config also written
  → all three coexist without conflicts
```

### Tier 2 — Agent scenarios

Agent tests launch a real AI agent with a prompt and verify the resulting filesystem state. Use the `ForEachAgent` pattern from entire.io: one test definition runs against all registered agents.

**Agent registry:**

```go
type Agent interface {
    RunPrompt(t *testing.T, repoDir string, prompt string) error
    Bootstrap(t *testing.T) error
    IsTransientError(err error) bool
    TimeoutMultiplier() float64
}
```

Initial agents: Claude Code (via `claude -p`), Cursor CLI (if available). More added later.

**Scenario: Agent creates a document via MCP**

```
Prompt: "Create an ADR documenting the decision to use PostgreSQL for primary persistence. Use the create_document MCP tool."
  → .archcore/ contains a new .adr.md file
  → file has valid frontmatter (title, status: draft)
  → file contains ADR template sections (Context, Decision, Alternatives, Consequences)
  → list_documents returns the new document
```

**Scenario: Agent discovers and updates an existing document**

```
Setup: create .archcore/auth/jwt-strategy.adr.md with status: draft
Prompt: "Find the ADR about JWT strategy and change its status to accepted."
  → agent calls list_documents → get_document → update_document
  → file frontmatter now has status: accepted
  → no other files modified
```

**Scenario: Agent links related documents**

```
Setup: create an ADR and a PRD
Prompt: "The ADR about caching implements the PRD about performance requirements. Link them with the appropriate relation."
  → agent calls add_relation with correct source, target, type: implements
  → list_relations shows the new relation
```

**Scenario: Agent follows a rule from session context**

```
Setup: create .archcore/naming.rule.md with rule "all document slugs must use kebab-case"
  → session-start hook delivers this rule in context
Prompt: "Create a guide about database migrations."
  → agent creates a file with kebab-case slug (e.g., database-migrations.guide.md)
```

### Test infrastructure

**Repo setup helper:**

```go
func SetupRepo(t *testing.T) *RepoState {
    dir := os.MkdirTemp("", "archcore-e2e-*")
    // git init, git config user, initial commit
    // archcore init --sync none (non-interactive)
    // archcore hooks install
    // archcore mcp install
    return &RepoState{Dir: dir, ...}
}
```

**MCP client helper** (for canary tests):

```go
func StartMCPServer(t *testing.T, repoDir string) *MCPClient {
    // Start `archcore mcp` as subprocess with stdio transport
    // Return client that can call tools via JSON-RPC
}

func (c *MCPClient) CallTool(name string, args map[string]any) (json.RawMessage, error)
```

**Artifact collection** (from entire.io): on test failure, save `.archcore/` directory state, git log, MCP server stderr, agent transcript to `e2e/artifacts/<timestamp>/<TestName>/`.

**Transient retry** (for agent tests): wrap test body in panic/recover loop, retry up to 2 times on `IsTransientError` (rate limits, agent confusion). Each retry starts with a clean repo.

### Makefile / mise integration

```toml
[tasks."test:e2e:canary"]
run = "go test -tags=e2e ./e2e/... -run TestCanary -timeout 2m"

[tasks."test:e2e:agent"]
run = "go test -tags=e2e ./e2e/... -run TestAgent -timeout 10m"
depends = ["build"]

[tasks."test:ci"]
depends = ["test:unit", "test:e2e:canary"]
```

Canary runs in CI on every commit alongside unit tests. Agent tests run on-demand or nightly.

## Risks and Constraints

### Agent test non-determinism

LLM responses are inherently non-deterministic. An agent might ask for confirmation, create extra files, or choose the wrong tool. Mitigations: explicit prompts ("Do not ask for confirmation"), retry logic, generous assertions (check file exists + has correct frontmatter, don't assert exact content).

### MCP test harness doesn't exist yet

There's no standard Go library for testing MCP servers as a client. We need to build a thin JSON-RPC client over stdio that starts `archcore mcp` as a subprocess. This is ~100-200 LOC of infrastructure but is novel and may have edge cases (buffering, process lifecycle).

### Agent binary availability

Agent tests require `claude`, `cursor` etc. to be installed and authenticated. CI needs secrets for API keys. Canary tier has no such dependency — it only needs the `archcore` binary.

### Token costs

Each agent test run costs tokens. A full suite of 5-10 agent scenarios × 2 agents ≈ 10-20 API calls. With retry, potentially 30-60. This is manageable for nightly runs but prohibitive for per-commit CI — hence the two-tier split.

### Test maintenance burden

Agent test prompts must be maintained as the tool descriptions evolve. If `create_document` adds a required parameter, agent prompts may need updating. Canary tests (direct JSON-RPC) are more stable but still need parameter updates.

### Scope creep

E2E tests should not duplicate unit test coverage. Units test validation logic, error messages, edge cases. E2E tests verify the integration path works. If a scenario can be fully tested as a unit test, it should stay a unit test.
