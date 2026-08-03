---
title: "Universal Agent Specification"
status: accepted
tags:
  - "agents"
  - "plugin"
---

## Purpose & Scope

This spec defines the contract for the Archcore plugin's two subagents — `archcore-assistant` (read/write) and `archcore-auditor` (read-only) — across every host that loads them. Normative for both agent definitions, their per-host file formats, their system prompts, tool restrictions, invocation triggers, and domain expertise. Depended on by every host loader and by `@test/structure/agents.bats`. `single-universal-agent.adr` records the original rationale, `add-read-only-auditor-agent.adr` extends it, and `subagent-knowledge-tree-bootstrap.adr` is authoritative for the mandatory preamble. Out of scope: skills, which `skills-system.spec` governs.

## Surface

**Definitions.** Each agent is one definition, shipped in the format each host's loader accepts. Content is identical across formats; only the container differs. Canonical sources: `@plugins/archcore/agents/archcore-assistant.md` and `@plugins/archcore/agents/archcore-auditor.md`.

| Format | Location | Read by | Notes |
|---|---|---|---|
| `<name>.md` | `agents/` | Claude Code, Cursor | The canonical source; agent id comes from frontmatter `name:` |
| `<name>.toml` | `agents/` | Codex CLI | Adds `sandbox_mode` and `disabled_tools[]`; body kept in parity with the MD |
| `<name>.agent.md` | `copilot-agents/` | GitHub Copilot CLI | Byte-identical copy of the MD; agent id comes from the filename |

Copilot's copies sit in a directory of their own rather than beside the originals. Its loader accepts only the `*.agent.md` extension, and that extension still matches the `*.md` glob Claude Code and Cursor use, so a sibling copy would give both hosts two files declaring the same `name:`. The TOML variants never had this problem, because their extension is foreign to every md-globbing host.

**Runtime knobs.** Declared in each definition's frontmatter, referenced here rather than reproduced.

| Agent | Role | model | maxTurns | background |
|---|---|---|---|---|
| `archcore-assistant` | complex, multi-step documentation tasks requiring write access | sonnet | 20 | no |
| `archcore-auditor` | documentation health checks with no mutation capability | sonnet | 15 | yes |

**MCP tool naming.** The same MCP server appears under three names depending on registration: `mcp__archcore__*` (project `.mcp.json`), `mcp__plugin_archcore_archcore__*` (plugin-bundled on Claude Code), and the flat `archcore-<tool>` (Copilot, which joins server and tool with a hyphen). Every allow-list and deny-list in an agent file carries all three. The asymmetry matters: an allow-list missing a name loses a capability, while the auditor's TOML **deny**-list missing a name silently grants the read-only agent the power to mutate.

**Tool access matrix.**

| Tool | assistant | auditor |
|------|-----------|---------|
| list_documents | Yes | Yes |
| get_document | Yes | Yes |
| create_document | Yes | No |
| update_document | Yes | No |
| remove_document | Yes | No |
| add_relation | Yes | No |
| remove_relation | Yes | No |
| list_relations | Yes | Yes |
| Read | Yes | Yes |
| Grep | Yes | Yes |
| Glob | Yes | Yes |
| Write/Edit/Bash | No | No |

**Invocation triggers.** The host invokes `archcore-assistant` when the user requests several related documents, when the task decomposes requirements ("break this PRD into specifications"), when existing documentation structure is refactored, or when a decision needs the full relation graph. The host invokes `archcore-auditor` when the user asks for an audit, health check, or review; when the user asks what is missing or what needs attention; proactively after a batch of documents has been created; before a release or milestone; or to check documentation against current code.

**Shared domain knowledge.** Both agents cover all 18 document types across the three categories — knowledge (`adr`, `rfc`, `rule`, `guide`, `doc`, `spec`), vision (`prd`, `idea`, `plan`, `mrd`, `brd`, `urd`, `brs`, `strs`, `syrs`, `srs`), and experience (`task-type`, `cpat`) — including each type's purpose, its trigger, its required sections, and its differentiation from similar types. They cover three coexisting requirements tracks: product (idea → prd → plan), sources (mrd + brd + urd → prd), and ISO 29148 (brs → strs → syrs → srs). They cover the four relation types: `implements` (source fulfills target), `extends` (source builds on target), `depends_on` (source requires target), and `related` (general association).

**Output contracts.** `archcore-assistant` returns created and updated documents, relation changes, and the reasoning behind its choices. `archcore-auditor` returns a structured report with Audit Summary (counts, issue totals), Critical Issues (broken references, misleading content), Warnings (quality gaps), Code-Document Correlation (documents referencing source paths where code changed after the document was last modified), Info (suggestions), and Recommendations (prioritized actions).

## Normative Behavior

1. Each agent MUST call `list_documents` and `list_relations` in parallel as the first tool calls of every invocation, before any domain action.
2. WHEN both bootstrap calls return, the agent MUST note the categories present, the most common tags, recent accepted decisions, and any draft plans before proceeding. This synthesis is a read-only transformation over data already in hand and adds no tool call.
3. Each agent's system prompt MUST carry a `# First Step — Bootstrap Knowledge Tree` section as the first content section after the YAML frontmatter.
4. That preamble MUST cross-reference `subagent-knowledge-tree-bootstrap.adr` for the rationale.
5. That preamble MUST cross-reference `remove-skill-verify-mcp-preamble.cpat`, so the section is not removed by analogy with the retired MCP-verification preamble.
6. Each agent MUST perform every `.archcore/` operation through an MCP tool.
7. Each agent MUST list every MCP tool it uses under all three namings, so the definition works whether the server was registered by the project, by the plugin, or by Copilot's flattening.
8. Each agent SHOULD explain its reasoning when it chooses a document type or a relation type.
9. `archcore-assistant` MUST create a relation between documents it creates whenever a semantic link exists.
10. `archcore-assistant` SHOULD present a plan for user approval before creating several documents.
11. `archcore-assistant` MUST NOT create more than 10 documents in one invocation without user confirmation.
12. `archcore-assistant` MAY skip `list_relations` during the bootstrap only when the task is a strictly single-document read with an explicit path; `list_documents` remains required.
13. `archcore-auditor` MUST NOT create, update, or delete a document.
14. `archcore-auditor` MUST perform the full bootstrap with no exception, because an audit without the graph produces incomplete findings.
15. `archcore-auditor` MUST return a structured audit report rather than free-form commentary.
16. `archcore-auditor` SHOULD cross-reference documentation against code through Read, Grep, and Glob.
17. `archcore-auditor` SHOULD use `Grep` to find path references in document bodies and then check with `git log` whether those paths changed after the document was last modified.
18. `archcore-auditor` SHOULD prioritize specs, ADRs, and guides describing specific code modules when correlating documents with code.

## Constraints & Invariants

- Constraint: an agent system prompt MUST NOT exceed 2000 lines.
- Constraint: an agent MUST NOT modify a file outside `.archcore/` by any means.
- Constraint: an agent MUST respect an existing document status.
- Constraint: a format variant exists only where a host's loader requires one. Variants are copies, not forks: no host-specific instruction may enter one (`host-adapter-contract.spec`).
- Invariant: `archcore-assistant` never uses Write, Edit, or Bash on a `.archcore/` file.
- Invariant: `archcore-auditor` holds zero write tools, enforced by the tool allow-list and by the TOML deny-list.
- Invariant: both agents check for an existing document before suggesting creation.
- Invariant: the first tool calls of every invocation are `list_documents` and `list_relations`. `archcore-auditor` has no exception; `archcore-assistant` has the narrow exception of item 12.
- Invariant: both system prompts carry the bootstrap section with both cross-references and with the synthesis anchor literal `recent accepted decisions` present.
- Invariant: every agent in `agents/*.md` has a byte-identical `copilot-agents/<name>.agent.md` counterpart.
- Invariant: no Copilot-only tool entry exists without its canonical twin.

## Failure Behavior

1. IF the MCP server is unavailable, THEN the agent MUST inform the user and exit without further tool calls.
2. IF a document operation fails, THEN the agent MUST report the error and continue with the remaining tasks.
3. IF a relation target does not exist, THEN the agent MUST skip that relation and report the skip to the user.

## Conformance

An agent is conformant when:

1. It resides at `agents/<name>.md` with the required frontmatter, and every format variant its hosts require exists: `agents/<name>.toml` for Codex and `copilot-agents/<name>.agent.md` for Copilot.
2. Its tool list matches the tool access matrix exactly, under all three MCP namings.
3. Its system prompt covers the shared domain knowledge above.
4. It satisfies the normative behavior for its role.
5. `archcore-auditor` produces no mutation, and `archcore-assistant` produces structured output.
6. Its system prompt carries the `# First Step — Bootstrap Knowledge Tree` section with both cross-references and the grep-able anchor literal `recent accepted decisions`.
7. `@test/structure/agents.bats` asserts the bootstrap preamble, the synthesis anchor, the three-way tool naming, and byte-identity of the Copilot copies.
