---
title: "File Context Resolution — The Documents That Constrain an Edit to One File"
status: draft
tags:
  - "component:cli"
  - "hooks"
  - "integrations"
  - "mcp"
---

## Purpose & Scope

This spec defines the file-context result: for one repository file, the ordered, bounded set of `.archcore/` documents that constrain an edit to it. Normative for the engine in @cli/internal/advisory/code_alignment.go and its two renderers: the pre-edit hook (`preToolUseHandler` in @cli/cmd/hook_command.go) and the `for_path` mode of `search_documents` (@cli/internal/mcp/tools/search_documents.go). Dependents: coding agents on every host Archcore wires, the plugin's next-step mod (@plugin/plugins/archcore/hooks/next-step.tsx), and the context-panel pilot.

Out of scope: the write guard; the other `search_documents` modes; the context address that tells an agent to request this result.

## Surface

- Input: one file path, relative to the project root or absolute inside it.
- Row fields: `path` (with the `.archcore/` prefix), `type`, `title`, `status`, `source_kind`, `reason`.
- `reason` values: `file` (the body names the file), `kind` (an accepted `rule` or `cpat` names the file's compound extension, such as `.test.tsx`), `directory` (the body names a directory of the file), `general` (an accepted pathless `rule` or `cpat`).
- Result parts, in order: matched rows (`file`, `kind`, then `directory`), general rows, one remainder line per part.
- Header line of the hook text: `[Archcore Context] Read these before editing <file>:`.
- Hook rendering: plain text opened by the `[Archcore Context]` marker line.
- MCP rendering: the JSON envelope of `search_documents` under `for_path`.
- Ranked types: `rule`, `cpat`, `adr`, `spec`, `scenario`, `guide`, `doc`, in that priority.

## Normative Behavior

1. WHEN a document names the edited file by full path, `@`-path, or file name, the engine MUST give it reason `file`.
2. The engine MUST order matched rows by reason: `file`, then `kind`, then `directory`.
3. WHEN two rows tie on reason and matched depth, the engine MUST rank an `accepted` document above a `draft`.
4. WHEN two rows still tie, the engine MUST rank by type priority, then by the number of matches.
5. The engine MUST NOT order rows by document path except as the final tie-break.
6. The engine MUST rank `doc` below every other ranked type.
7. The engine MUST list each accepted `rule` and `cpat` that carries no code reference as a `general` row.
8. The engine MUST place general rows after matched rows.
9. The engine MUST render every row path with the `.archcore/` prefix that `get_document` accepts.
10. WHEN a part holds more rows than its cap, the engine MUST state the omitted count and the call that returns them.
11. WHEN the edited file lies outside every source root, the engine MUST still return `file` rows and general rows.
12. WHEN the hook renders a non-empty result, the hook MUST open it with one line telling the agent to read the rows before editing.
13. The hook and the `for_path` mode MUST return the same matched rows, in the same order, for one file and one corpus.
14. The engine MUST mark a row from a global source as global in both renderings.
15. WHEN the session already received this exact result for this file, the hook SHOULD emit one reminder line instead of the rows.
16. WHEN an accepted `rule` or `cpat` names the compound extension of the edited file, the engine MUST give it reason `kind`.

## Constraints & Invariants

- Constraint: matched rows MUST NOT exceed 5, so five documents fit the hook budget with their titles.
- Constraint: general rows MUST NOT exceed 10. The monorepo holds 15 pathless rules (2026-10-09), so the remainder line is exercised.
- Constraint: the hook text MUST NOT exceed 2 048 runes. Codex caps hook context at 2 500 tokens per handler.
- Constraint: the `for_path` response MUST NOT exceed 8 000 characters. Copilot in VS Code stores a result over 8 × 1 024 characters in a file.
- Constraint: the hook computation MUST finish inside the one-second host budget of the hook runtime contract.
- Invariant: the result size is a function of the caps, never of corpus size.
- Invariant: a document carries no code reference when no reference the `path_ref` extractor (`docs.ExtractPathRefs`) finds names an existing file or directory of the project.
- Invariant: a document that qualifies for `kind` and for `general` takes `kind`.
- Invariant: a document names a compound extension only where no file-name character precedes it: `*.test.tsx` names the kind, `Form.test.tsx` names one file.

## Failure Behavior

1. IF no matched row and no general row exists, THEN the hook MUST emit nothing.
2. IF the scan cannot finish inside the host budget, THEN the hook MUST emit nothing and allow the edit.
3. IF `for_path` names a path outside the project, THEN `search_documents` MUST return an error that holds no absolute path.
4. IF one document cannot be read, THEN the engine MUST skip it and keep the other rows.

## Conformance

An implementation is conformant when it satisfies behaviors 1–14 and 16, holds every constraint and invariant, and degrades per the failure rules. Behavior 15 is conformant when absent; a session stamp hides the rows after `/compact`, so it waits for a compaction signal.

Given `FullLayout.tsx` is named in `router-shim-hybrid-seam.doc.md` and only its directory in an accepted `rule`,
When the agent calls `search_documents(for_path="apps/next-app/src/app/_components/FullLayout.tsx")`,
Then the `doc` row with reason `file` precedes the `rule` row with reason `directory`.
