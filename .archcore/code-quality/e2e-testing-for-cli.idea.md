---
title: "E2E Testing Strategy for Archcore CLI"
status: accepted
tags:
  - "code-quality"
  - "component:cli"
  - "testing"
---

## Status

**Layer A shipped** as in-process MCP integration tests at `internal/mcp/integration/`. The decision record is `in-process-mcp-integration-tests.adr`. Those tests run through `mcp-go`'s `client.NewInProcessClient` (sub-second, no subprocess, no build tag).

## Idea

Complement the unit suite with end-to-end tests through the real user path: init, hooks, MCP, and agent behavior. Layer A covers the in-process part. Layers B and C stay open.

## Value

Unit tests miss stdio framing, command wiring, and agent comprehension of tool descriptions. These layers catch them, at rising cost.

## Open Layers

- **Layer B**: run `archcore mcp` as a subprocess over real stdio, and call the CLI commands (`init`, `status`, `doctor`, `hooks install`) as real processes against a git repository. It catches command wiring and stdio framing bugs that the in-process client cannot. It needs a thin JSON-RPC stdio client of roughly 100 to 200 lines, with edge cases in buffering and process lifecycle. Isolate it behind an `e2e` build tag so `go test ./...` skips it.
- **Layer C**: send natural-language prompts to real agents (Claude Code, Cursor) and assert the resulting filesystem state. It is the only layer that catches a tool description reworded into a misuse. It is non-deterministic and costs tokens, so it runs on demand or nightly, with retry on transient errors and generous assertions.

## Risks and Constraints

- Agent tests need installed and authenticated agent binaries and API secrets in CI. Layer B needs only the `archcore` binary.
- Agent prompts need maintenance as tool descriptions evolve.
- E2E tests must not duplicate unit coverage. A scenario that a unit test can cover stays a unit test.
