---
title: "Agent Context Delivery — Host Channel Limits, Session Baseline, and an Instruction-Block Pilot"
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

Decide where Archcore puts the sentence that tells an agent where to fetch project context, and how that context reaches the agent before a code edit. The design must survive host truncation on every supported host and must not grow with the corpus.

## Questions

1. Which agent channel reaches the model whole on Claude Code, Codex, Copilot CLI, Copilot in VS Code, and Cursor?
2. How do agents consult `.archcore/` before a code edit in a 300-document corpus today?
3. Does the pre-edit hint deliver the documents an edit needs?
4. Does an explicit managed block change agent behavior on a test-writing task?
5. Which delivery design survives both truncation and corpus growth?

## Approach

Method: one live marker probe per host, source and documentation reads, a transcript scan, real hook runs, and an A/B pilot. All runs took place on 2026-10-09.

Inputs:

- Marker probe. A stdio MCP server sent 20 000 characters of instructions with a marker every 500 characters, a 4 000-character tool description, and a 120 637-character tool result. Each marker carries a random suffix; every reported marker was checked against the generator output, and none was invented. Hosts: Claude Code 2.1.295, Codex CLI 0.159.0, Copilot CLI 1.0.93. The throwaway server lives in the session scratchpad, outside the repository.
- Claude Code documentation: <https://code.claude.com/docs/en/mcp>, <https://code.claude.com/docs/en/hooks>, <https://code.claude.com/docs/en/memory>, <https://code.claude.com/docs/en/skills> — accessed 2026-10-09.
- Codex source at tag `rust-v0.159.0` (commit `687a119f`): `codex-rs/codex-mcp/src/rmcp_client.rs`, `codex-rs/core/src/tools/handlers/mcp.rs`, `codex-rs/hooks/src/output_spill.rs`, `codex-rs/core/src/agents_md.rs`, `codex-rs/config/src/config_toml.rs`; config reference <https://learn.chatgpt.com/docs/config-file/config-reference> — accessed 2026-10-09.
- Copilot: `github/docs` commit `9f65179` (`cli-command-reference.md`, `context-management.md`, `tool-search.md`); `microsoft/vscode` commit `51ac693`, `extensions/copilot/src/` — accessed 2026-10-09.
- Cursor: <https://cursor.com/docs/context/mcp>, <https://cursor.com/docs/agent/hooks>, <https://cursor.com/blog/dynamic-context-discovery> — accessed 2026-10-09 through a summarizing fetch, so quotes are not verbatim.
- Transcripts: 33 Claude Code sessions with code edits in the Litres monorepo (302 documents), 250 `Edit`/`Write` calls on code files.
- Hook runs: `archcore hooks claude-code pre-tool-use` and `session-start`, CLI v0.11.0, on the same monorepo.
- Pilot: 12 `claude -p` runs, model Sonnet 5.5, prompt "Write unit tests for apps/next-app/src/utils/<file>.ts. Do not run them.", 3 files × 2 variants × 2 repeats. Variant A kept the current managed block; variant B carried a block with explicit tool calls and a plain-file fallback. Each variant and repeat ran in its own local clone. Cost: about 3.4 USD.

## Findings

1. Channel limits (Q1). "probe" marks a live probe; "source" and "docs" mark reads.

| Channel | Claude Code | Codex | Copilot CLI | Copilot in VS Code | Cursor |
|---|---|---|---|---|---|
| MCP server instructions | cut at 2 048 characters; probe: 5 of 40 markers | whole up to 512 KiB; probe: 40 of 40 | absent unless allowlisted or `--allow-all-mcp-server-instructions`; probe: 0 of 40, with the flag 40 of 40 | whole while the server's tools are available (source) | unknown |
| Tool description | docs: cut at 2 048; probe: 16 of 16 after deferred load | no cap; probe: 16 of 16 | probe: 16 of 16 | no cap found (source) | unknown |
| Tool result | 25 000 tokens, over 50 000 characters saved to a file (docs, probe) | about 12 000 tokens, cut in the middle (source); probe saw at least 99 000 characters | over 20 KiB saved to a file with a preview (docs, probe) | over 8 × 1 024 characters saved to a file (source) | saved to a file, threshold unknown |
| Hook context | 10 000 characters (docs) | 2 500 tokens per handler, `additionalContextLimit` (source, docs) | 10 MiB per invocation | no cap found | unknown |
| Instruction file | loaded whole; 200 lines advised (docs) | 32 KiB shared by all `AGENTS.md`, tail cut silently (source) | unknown | no cap found | 500 lines advised |

2. Server instructions measure 19 539 characters (20 110 in a project with a `language` setting) in @cli/internal/mcp/server.go. `LANGUAGE REQUIREMENT` and `GLOBAL SOURCES` are appended after character 19 500, so Claude Code never shows them. `create_document` describes itself in 3 696 characters.
3. Session baseline (Q2). In 29 of 33 sessions the agent read `.archcore/` before the first code edit. Reads went through the shell 463 times (`cat`/`sed` 253, `grep` 163, `ls`/`find` 47) and through MCP 293 times. Twenty read-tool results overflowed the host limit.
4. Pre-edit hint (Q3). The hook fired 154 times with 29 distinct texts. The agent later opened 72 of the 462 listed documents (16%). The hint prints paths without `.archcore/`, and `get_document("integrations/session-start-context.spec.md")` returns `invalid path: must start with ".archcore/"`.
5. Pre-edit hint misses on real files (Q3). For `Foo.test.tsx` under `apps/next-app/src/components/` the hint listed no unit-test rule. For `packages/ui/src/Button/Button.tsx` it listed captcha, iframe-bridge, and tracker rules. For `vitest.config.mjs` and `helm/values.yaml` it listed nothing, because both lie outside the default source roots and `doc` is not an injected type. 15 of 54 rules name no `apps/` or `packages/` path and can never match.
6. Session-start recap. The CLI emits 4 273 characters, mostly the `IN PROGRESS` list. No line names the call that returns context for a file. The closing line points to server instructions that Claude Code cuts and Copilot CLI omits.
7. Pilot (Q4).

| Measure | A (current block) | B (explicit block) |
|---|---|---|
| Read a unit-test rule before the first write | 5 of 6 | 6 of 6 |
| Called an Archcore MCP tool | 0 of 6 | 3 of 6 |
| Runs without a scored rule violation | 4 of 6 | 5 of 6 |

The monorepo's hand-written `CLAUDE.md` section routes both variants to `.archcore/code-quality/tests/units/`, so the pilot cannot separate the block's effect from that section. Twelve runs on one task are a pilot, not a statistic.

8. Bench after the change (2026-10-09, `plugin/test/behavioral/context-bench.py`, Sonnet 5.5, 3 tasks × 3 runs, fixture without a hand-written `.archcore/` section). The CLI built from the branch replaced v0.11.0 on `PATH`; the plugin stayed v0.11.0.

| Measure | v0.11.0 | Branch |
|---|---|---|
| Read the applicable rule before the first write | 4 of 9 | 9 of 9 |
| Called an Archcore MCP tool before the first write | 9 of 9 | 9 of 9 |
| Outcome complied with the rule | 6 of 9 | 9 of 9 |
| Cost | 1.07 USD | 1.92 USD |

The pathless guard rule moved from 0 of 3 to 3 of 3; the root-config doc from 1 of 3 to 3 of 3. Every branch run called `search_documents` with `for_path`. Three runs per task show a direction, not a statistic.

9. Address probe after the change (2026-10-09, server instructions of the branch CLI, no instruction file in the project). Claude Code 2.1.295 quoted the address from the server instructions and from the session-start recap. Copilot CLI 1.0.93 quoted it from the server instructions. Codex CLI 0.159.0 saw neither the tools nor the instructions until tool search loaded the server namespace; it then quoted the address from the namespace description, and before that only the session-start recap carried it.
10. Monorepo pre-edit hint for `Foo.test.tsx` after the change: two unit-test rules with reason `kind`, and the 10 general rules including `guard-clauses-first.rule.md`. Rules that name a single test file (`Form.test.tsx`) no longer take `kind`.
11. Unanswered: Cursor limits on every channel; the Copilot CLI tool-description cap; the token count of the session-start recap under the Codex tokenizer.

## Recommendation

proceed. Supporting findings: 1, 2, 4, 5, 6.

- One short context address leads every channel that reaches the agent, because no single channel reaches it whole on every host (finding 1).
- Server instructions fit 2 048 characters, and the managed block, which every probed host loads whole, carries the address first (findings 1, 2).
- One file-context result serves both the pre-edit hook and an agent call, with `.archcore/`-prefixed paths and a fixed size (findings 3, 4, 5).
- A rule that names no path reaches the file-context result as a general rule (finding 5).
- A bench on a project without a hand-written `.archcore/` section measures the effect before and after (finding 7).

## Next Action

`/archcore:plan` continues this route: decision records, the `prd`, the two `spec` documents, and the `plan`.
