# core

Source of truth for architecture, decisions and coding rules: Archcore documents in `.archcore/` (MCP tools `list_documents` / `get_document`). These memories only point there; do not duplicate or override them.

## Entry points
- `CLAUDE.md`, `AGENTS.md` (repo root): binding agent instructions and writing policy. Content between `archcore:start` / `archcore:end` markers is generated; do not edit.
- `.archcore/architecture/system-overview.doc.md`: system map. Read before a change crossing package boundaries.
- `.archcore/architecture/package-dependency-direction.rule.md`: allowed import graph.
- `.archcore/cli-ui/building-the-cli.doc.md`: procedures for adding command, doc type, MCP tool, hook, agent.

## Source map
- `main.go`: entry; `version` injected via ldflags.
- `cmd/`: cobra commands (init, mcp, status, doctor, config, hooks, instructions, plugin, sync, update) plus hook handlers `hook_*.go`.
- `internal/`: 18+ packages; list and ownership in `CLAUDE.md` §Internal packages. `internal/mcp/tools/` = MCP tools, `internal/mcp/integration/` = in-process MCP tests.
- `templates/`: document templates and document types.
- `scripts/`: `regen-examples.sh` (regenerates `examples/`), `assert-not-inert*.sh` (release guard), `prose-conformance.py`.
- Out of scope for normal searches: `reference-materials/`, `examples/` (generated fixtures).

## Invariants (non-obvious)
- `.archcore/` docs are modified only via Archcore MCP tools, never direct file writes. Global sources are read-only.
- Shipped instruction files (root `AGENTS.md`/`CLAUDE.md`/`GEMINI.md`, `examples/`) are pinned by `TestShippedInstructionFixtures`; changing the writer requires regenerating fixtures.
- MCP errors must not contain absolute filesystem paths.

More: toolchain `mem:tech_stack`; commands `mem:suggested_commands`; Go rules loading `mem:conventions`; done-checklist `mem:task_completion`.
