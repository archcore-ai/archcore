# CLAUDE.md

Read and follow `AGENTS.md` before creating or editing technical documentation, Archcore documents, CLI help, MCP tool descriptions, prompts, agent instructions, or user-facing Markdown.

The writing policy in `AGENTS.md` uses:

- an ASD-STE100-inspired profile for procedures, requirements, rules, specifications, CLI instructions, MCP contracts, and agent workflows;
- ISO 24495-1-inspired plain-language principles for architecture explanations, ADRs, RFCs, README content, and product documents.

Do not claim formal ASD-STE100 or ISO 24495-1 compliance.

## Search Priority

When researching patterns, decisions, or conventions in this project, always search `.archcore/` documents first (`list_documents` → `get_document`) before grepping the codebase or using external sources.

The canonical architecture and how-to reference is `.archcore/cli-ui/building-the-cli.guide.md`. Consult it before adding a command, document type, MCP tool, hook, or agent.

## Archcore Operations

Use Archcore MCP tools for all `.archcore/` document operations.

- Create documents with `create_document`.
- Update documents with `update_document`.
- Remove documents with `remove_document`.
- Read documents with `list_documents`, `search_documents`, and `get_document`.
- Manage document relations with `add_relation`, `remove_relation`, and `list_relations`.

Do not use direct file-writing tools to modify `.archcore/` documents.

Before creating or updating an Archcore document:

1. Search existing documents for relevant decisions, rules, specifications, and duplicates.
2. Read the applicable document-type guidance.
3. Apply the controlled technical writing policy in `AGENTS.md`.
4. Preserve code identifiers, commands, flags, paths, MCP tool names, configuration keys, and literal values exactly.
5. Mark unsupported technical claims with `[assumption]`.

Treat mounted global documents as read-only. Do not edit them or create relations to them.

## Managed Blocks

Do not edit content inside an Archcore-managed block:

```text
<!-- archcore:start -->
...
<!-- archcore:end -->
```

Keep repository-specific instructions outside the managed block.

## Build and Test Commands

```bash
# Build
go build -o archcore .

# Run all tests
go test ./...

# Run tests for a package
go test ./cmd/
go test ./internal/mcp/...
go test ./internal/config/
go test ./templates/

# Run one test
go test ./cmd/ -run TestSetSettingsValue

# Run tests with verbose output
go test -v ./...
```

Use Go 1.25 or newer.

## Architecture

`archcore-cli` is a Go CLI and local stdio MCP server. It manages a local `.archcore/` directory of structured Markdown documents and integrates with coding agents through MCP and lifecycle hooks.

The `.archcore/` directory is free-form. Document files use the form `<slug>.<type>.md`. The category is derived from the type suffix.

Document management happens through MCP tools. CLI subcommands handle setup, health, sync, host wiring, and updates.

### Commands

Commands are registered under `cmd/`.

- `init` initializes `.archcore/` and host integrations.
- `mcp` runs the MCP server.
- `status` checks document structure.
- `doctor` checks project and integration health.
- `config` reads and updates `.archcore/settings.json`.
- `hooks` manages lifecycle hooks.
- `instructions` manages agent instruction files.
- `sync` pushes `.archcore/` state to a configured server.
- `update` updates the CLI.

When adding a command:

1. Read `.archcore/cli-ui/building-the-cli.guide.md`.
2. Use the constructor-command pattern.
3. Keep command logic in testable functions.
4. Add co-located tests.
5. Update user-facing documentation when the command surface changes.

### Internal packages

- `internal/config/` manages settings and initialization.
- `internal/mcp/` implements the MCP server.
- `internal/mcp/tools/` implements MCP tools.
- `internal/mcp/prompts/` implements document-track prompts.
- `internal/mcp/integration/` contains in-process MCP integration tests.
- `internal/agents/` defines supported agent integrations.
- `internal/wiring/` implements host wiring.
- `internal/sync/` implements sync state, hashing, and payload construction.
- `internal/api/` implements the server API client.
- `internal/update/` implements self-update.
- `internal/git/` detects repository metadata.
- `internal/display/` formats terminal output.
- `templates/` defines document templates and document types.

### Design constraints

Preserve these constraints:

- MCP-first document management.
- Free-form `.archcore/` directory layout.
- Virtual document categories derived from type suffixes.
- Path traversal protection.
- No absolute filesystem paths in MCP errors.
- Shared host-wiring logic in `internal/wiring/`.
- Co-located table-driven Go tests.
- Optional settings omit defaults where defined.
- Global Archcore sources are read-only.

## Out-of-scope Directories

Exclude these directories from normal implementation searches unless the task explicitly concerns them:

- `reference-materials/` — vendored references and standards material; not part of the build.
- `examples/` — example project layouts and manual-test fixtures.

<!-- archcore:start --> managed by `archcore init` — edit outside these markers
## Archcore — project context for this repo

This repo's architecture, decisions, rules, specs and patterns live in `.archcore/`,
reachable through the Archcore MCP tools. Consult them even on code you think you
know — a decision or rule may already constrain it.

- Touching this repo's real code or behavior → search first; read only what matches.
- A decision was made ("we'll use X", "from now on Y") → record it.
- A module / API / system has no doc — or a search comes back empty → capture it.
- Planning a feature or refactor → scope it against what's already decided.

A `.archcore/` may also mount read-only **global sources** — shared, org-wide
context not shown in the session-start list. `list_documents` / `search_documents`
surface them alongside local docs, tagged `source_kind: "global"`. When present,
treat them as defaults a local doc can override — never edit or relate to one.

The search is cheap — lean on it. Skip it only for turns this repo would have no
opinion on: syntax trivia, throwaway snippets, pure mechanics.
<!-- archcore:end -->
