---
title: "Document Write Latency: Update Payloads, Writer Models, and Parallel Writers"
status: draft
tags:
  - "component:cli"
  - "component:plugin"
  - "mcp"
  - "performance"
---

## Goal

Decide how to cut the time an agent spends creating and updating `.archcore/` documents without weakening the checks that `create_document` and `update_document` apply. The maintainer reported slow document writes on 2026-09-26; this investigation measured where the time goes and tested four ways to reduce it.

## Questions / Hypotheses

- Q1: Does the write time go to tool execution, to the PostToolUse hook, or to model generation?
- Q2: Is native `Write`/`Edit` with post-write validation faster than the MCP write tools?
- Q3: Does a faster or cheaper model writing the documents cut the time of a document package?
- Q4: Do several writers working in parallel cut the time of a document package?
- Q5: How much does an update that sends only the changed text save on one review round?

## Approach

### Inputs

| Input | Scope | Date |
|---|---|---|
| Claude Code transcripts under `~/.claude/projects/*archcore*` | 2676 calls of `create_document`, `update_document`, `Write`, `Edit` | 2026-09-26 |
| Codex rollouts under `~/.codex/sessions` | model outputs of 800 characters or more, per model | 2026-09-26 |
| Claude Code docs: code.claude.com/docs/en/skills, code.claude.com/docs/en/sub-agents | skill and subagent `model` fields, fast mode | 2026-09-26 |

The transcript analysis used ad-hoc scripts that are not committed. Generation time is the gap between a tool call's transcript timestamp and the previous transcript entry. Every bench run used commit `811827f` plus the working-tree diff of @cli/internal/mcp/tools/update_document.go and its test.

### Options

| Option | Keeps MCP-only writes | Status |
|---|---|---|
| O1 — native `Write`/`Edit` on `.archcore/` plus post-write validation | No | not built |
| O2 — `edits` on `update_document`: exact-match replacements instead of a full body | Yes | implemented in @cli/internal/mcp/tools/update_document.go; not released |
| O3 — a skill or subagent `model` switch: a cheaper model writes the documents | Yes | tested by fan-out variant B |
| O4 — writer fan-out: the lead model writes briefs, parallel writers create one document each | Yes | tested by fan-out variants C and D |

### Experiments

- @plugin/test/behavioral/edits-bench.py gives one agent a document and three reviewer comments, 4 real documents of 3.2k-13.1k characters from @plugin/test/behavioral/fixtures/edits-bench.json, 3 rounds per host. Variant `before` serves the CLI of `811827f`; variant `after` adds `edits`. The prompt does not mention `edits`. Checks are deterministic: requested text present, replaced text absent, frontmatter and two untouched sections byte-identical.
- @plugin/test/behavioral/writer-fanout-bench.py writes one package of 5 documents and 6 relations from @plugin/test/behavioral/fixtures/writer-fanout-package.json, 3 runs per variant and host, in a fresh worktree with its own CLI build.
- @plugin/test/behavioral/writer-fanout-judge.py scores the median run of each group blind with `claude-opus-5-5` and `gpt-6-astra`; both judges run with no tools, and every verdict records 0 tool calls.

| Fan-out variant | Lead agent | Writers |
|---|---|---|
| A | strong model writes the whole package in one session | none |
| B | strong model writes briefs | 5 parallel fast-model writers that read the contracts themselves |
| C | strong model writes briefs | 5 parallel strong-model writers that read the contracts themselves |
| D | strong model writes briefs | 5 parallel strong-model writers with the contract and `precision-rules.md` inline and no reads |

Hosts and models: Claude Code with `claude-opus-5-5` and `claude-sonnet-5`; Codex with `gpt-6-astra` and `gpt-5.6-luna`.

## Findings

Model generation sets the write time on Claude Code, and `update_document` spends most of it on unchanged text; `edits` halves a review round there. On Codex fixed per-task overhead sets the time. Neither a cheaper writer nor parallel writers gave a gain worth its cost.

### Where the time goes (Q1, Q2)

| Tool | Calls | Median payload, chars | Median generation, s | Median execution, s | Generation, s per 1000 chars |
|---|---|---|---|---|---|
| `Edit` | 971 | 440 | 2.7 | 0.09 | 4.3 |
| `Write` | 781 | 2125 | 9.8 | 0.08 | 4.2 |
| `create_document` | 419 | 2497 | 11.5 | 0.09 | 4.4 |
| `update_document` | 505 | 4508 | 15.1 | 0.11 | 3.5 |

- `archcore hooks claude-code post-tool-use` took 0.03-0.07 s on the 269-document corpus.
- `Write` and `create_document` generate at the same rate (4.2 and 4.4 s per 1000 characters), so O1 gives no gain for creation.
- In 410 `update_document` calls with a known previous body, the median new body kept 91% of the old body unchanged.
- 165 of 419 created documents (40%) received an update in the same session, and those updates took 46% of the generation time spent on the created documents (1.5 h of 3.2 h).
- In the sequential fan-out runs, writing took 44-68% of the time, reading code and searching 14-48%, reading contracts 4-12%, and adding relations 2-7%, so creation has no lever above about 10% outside generation itself.

### Model speed (Q3)

| Host | Model | Samples | Median output, chars/s |
|---|---|---|---|
| Claude Code | `claude-opus-5-5` | 175 | 336 |
| Claude Code | `claude-sonnet-5` | 151 | 295 |
| Claude Code | `claude-haiku-4-5-20251001` | 22 | 286 |

- Every sampled `claude-opus-5-5` message carried `speed: standard`, so fast mode was not active.
- The first Codex estimate, 123 chars/s for `gpt-6-astra` and 195 for `gpt-5.6-luna`, timed the gap between log items and so included reasoning; in the update bench `gpt-6-astra` generated a 13k-character body in about 9 s.
- Per the Claude Code docs, a skill's `model` field switches the model for the rest of the turn, and a plugin has no way to enable fast mode.

### Update bench (Q5)

| Host | Variant | Median s per task | Median output tokens | Median update payload, chars | `edits` chosen | Checks passed | Median USD per task |
|---|---|---|---|---|---|---|---|
| Claude Code | before | 32 | 3221 | 7835 | 0/12 | 12/12 | 0.64 |
| Claude Code | after | 15 | 717 | 818 | 12/12 | 12/12 | 0.24 |
| Codex | before | 44 | 611 | 7837 | 0/12 | 12/12 | not captured |
| Codex | after | 40 | 551 | 1297 | 12/12 | 12/12 | not captured |

| Phase, median per task | Claude before | Claude after | Codex before | Codex after |
|---|---|---|---|---|
| start to `get_document` done, s | 7.0 | 7.0 | 20.1 | 19.2 |
| `get_document` done to `update_document` done, s | 15.8 | 3.3 | 9.4 | 11.3 |
| after the update, s | 3.5 | 3.8 | 8.3 | 8.1 |

- On the 13.1k-character spec, a Claude Code round took 47 s before and 17 s after.
- Agents chose `edits` in 24 of 24 tasks without an instruction to use it.

### Writer fan-out bench (Q3, Q4)

Median of 3 runs per group.

| Host | Variant | Median total, s | Range, s | Writer phase, s | Input tokens | USD |
|---|---|---|---|---|---|---|
| Claude Code | A | 170 | 122-236 | none | 2.11M | 1.42 |
| Claude Code | B | 348 | 291-355 | 258 | 4.00M | 3.73 |
| Claude Code | C | 156 | 132-186 | 61 | 2.60M | 3.56 |
| Claude Code | D | 145 | 140-164 | 58 | 1.66M | 3.06 |
| Codex | A | 206 | 192-226 | none | 0.84M | not captured |
| Codex | B | 263 | 217-270 | 116 | 3.05M | not captured |
| Codex | C | 199 | 190-210 | 68 | 1.56M | not captured |
| Codex | D | 175 | 170-175 | 51 | 1.00M | not captured |

- D cut the median time by 15% on both hosts; on Claude Code it cost 2.2 times as much as A.
- Cheaper writers were the slowest variant on both hosts: B took 348 s against 170 s on Claude Code and 263 s against 206 s on Codex.
- The brief phase took 64-117 s, about half of a fan-out run.
- An earlier round of the same bench did not restrict Claude's built-in tools (a lead agent ran `Bash`) and let the judge's working directory hold the run metadata; the rerun fixed both, and its time ranking matches the earlier one.

### Document quality

- In all 6 sequential A runs, the scenario cited 5-9 numbered spec clauses.
- In 9 of 9 parallel Claude Code runs, the scenario cited no spec clause, because its writer could not see the spec that another writer produced.
- In 5 of 9 parallel Codex runs, the scenario cited spec clauses taken from the lead's brief; whether those numbers match the parallel spec was not checked.
- Within Codex, both judges ranked A first and B last: `claude-opus-5-5` A 6.4, C 5.6, D 5.0, B 4.4; `gpt-6-astra` A 8.8, D 8.0, C 7.4, B 6.4.
- Within Claude Code, A scored at least as high as every other variant: `claude-opus-5-5` A 7.6, C 7.6, B 7.2, D 7.2; `gpt-6-astra` A 6.4, C 6.4, D 6.2, B 5.2.
- Each judge scored its own host higher, and the two judges ranked the plan documents in opposite order (0 concordant and 20 discordant pairs).
- The precision hook reported 0 findings on every final document of every run.

## Implications

- On Claude Code the write cost follows the number of generated characters, so sending fewer characters per call is the lever, and O2 delivers it on the update path.
- On Codex the per-task overhead of about 28 s dominates, so O2 cuts payload and tokens there but not time.
- O3 bought no speed on either host and lowered judged quality on Codex.
- O4 saves about 15% and loses spec traceability for dependent documents; a second writing wave for dependents would remove most of the saving.

## Recommendation

**proceed** with O2, `edits` on `update_document`, and do not pursue O3 or O4 in the plugin. The update bench halved a Claude Code review round (32 s to 15 s, 0.64 to 0.24 USD) with 24 of 24 agents choosing `edits` unprompted, while cheaper writers were slower and parallel writers saved 15% at 2.2 times the cost and lost spec traceability.

## Next Action

- [ ] No follow-up is needed in the plugin: agents chose `edits` unprompted, and the decision and its contract are recorded as drafts for the maintainer's review.

## Risks & Unknowns

- The fan-out bench covered one package of 5 mostly dependent documents; a package of independent documents can favor O4.
- The update bench comments are designed to have checkable outcomes; free-form review comments can change a larger share of the body.
- Codex reports fewer output tokens than its payloads imply (611 tokens for a 7.8k-character body), so Codex token counts are not comparable with Claude Code's.
- The judges show self-preference, and they disagree on the plan documents.
- Run time varies: Claude Code variant A ranged from 122 to 236 s across 3 runs.
