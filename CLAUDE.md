# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Search Priority

When researching patterns, decisions, or conventions in this project, always search `.archcore/` documents FIRST (`list_documents` → `get_document`) before grepping the codebase or using external sources. See `.archcore/cli/archcore-first-search-priority.rule.md`.

The canonical architecture and how-to reference is `.archcore/cli-ui/building-the-cli.guide.md` — consult it before adding a command, document type, MCP tool, hook, or agent (it carries the step-by-step recipes this file only summarizes).

## Build & Test Commands

```bash
# Build (Go 1.25+)
go build -o archcore .

# Run all tests
go test ./...

# Run tests for a specific package
go test ./cmd/
go test ./internal/mcp/...
go test ./internal/config/
go test ./templates/

# Run a single test
go test ./cmd/ -run TestSetSettingsValue

# Run tests with verbose output
go test -v ./...
```

## Architecture

`archcore-cli` is a Go CLI — a git-native context layer for AI coding agents. It manages a local `.archcore/` directory of structured Markdown documents (YAML frontmatter: `title`, `status`, `tags`) and integrates with AI agents (Claude Code, Cursor, Gemini CLI, GitHub Copilot, and more) through two surfaces: an **MCP server** and **lifecycle hooks**.

The layout under `.archcore/` is **free-form** — organize by domain/feature/team. Documents are named `<slug>.<type>.md` (e.g. `use-postgres.adr.md`); the three **categories** (`vision`, `knowledge`, `experience`) are *virtual*, derived from the type suffix rather than physical directories. See `.archcore/dir/free-form-directory-structure.adr.md`.

**Document management happens through the MCP server, not CLI subcommands.** There is no `create` command — agents create/update/remove documents and relations via MCP tools. CLI subcommands handle setup, health, sync, and updates.

### Commands (`cmd/`, registered in `cmd/root.go`)

- `init` — interactive setup wizard: scaffolds `.archcore/`, detects agents, installs hooks + MCP config, optionally writes usage-nudge instruction files.
- `mcp` — runs the MCP server over stdio (the primary document interface for agents); `mcp install` wires it into an agent.
- `status` — structural checks only (naming, frontmatter, categories).
- `doctor` — health check: structure + settings + server connectivity.
- `config` — view/modify `.archcore/settings.json` (`config get|set <key> [value]`).
- `hooks` — install/manage SessionStart hooks per agent (`claude-code`, `cursor`, `gemini-cli`, `copilot`, plus `install`/`remove`).
- `instructions` — install/remove per-agent usage-nudge files (`AGENTS.md` / `GEMINI.md` / `.claude/rules/`).
- `sync` — one-way push sync of `.archcore/` to a cloud/on-prem server.
- `update` — self-update to the latest release.

### Packages (`internal/`)

- **`config/`** — `settings.json` load/validate and directory init. Sync mode (`none`/`cloud`/`on-prem`) drives which fields are allowed/required, enforced via custom JSON marshaling.
- **`mcp/`** — MCP server (`server.go`), built on `mark3labs/mcp-go`. Starts even without an `.archcore/` dir (exposes `init_project`).
  - `mcp/tools/` — 11 tools: `init_project`, `list_documents`, `get_document`, `search_documents`, `create_document`, `update_document`, `remove_document`, `add_relation`, `remove_relation`, `list_relations`, plus `install_host_config` (registered only when the cmd layer injects an executor via `mcpserver.WithHostWiring`). Shared helpers in `common.go`.
  - `mcp/prompts/` — the five document-track cascades (product, sources, ISO 29148, architecture, standard).
  - `mcp/integration/` — in-process MCP integration tests (Layer A of the E2E strategy).
- **`agents/`** — registry of supported AI agents (Claude Code, Cursor, Gemini CLI, Copilot, Cline, Codex CLI, OpenCode, Roo Code). Each defines detection, MCP-config writing, hooks, and instruction targets; shared instruction upsert in `instructions.go`.
- **`wiring/`** — host-wiring domain logic shared by `init --agent`, `hooks install`, `doctor --fix`, and the `install_host_config` MCP tool: hook-config surgery with the `archcore hooks ` ownership marker (`hooks_install.go`), per-agent installers, `Apply`/`EnsureProjectInitialized`, path/dedupe helpers. Cobra commands and MCP sanitization stay in `cmd/`.
- **`sync/`** — sync internals: content hashing, manifest diffing, payload building (`hash.go`, `diff.go`, `manifest.go`, `payload.go`).
- **`api/`** — HTTP client for the Archcore server (`/api/v1/status`, `/api/v1/projects`). Cloud URL: `https://app.archcore.ai`.
- **`update/`** — self-update logic (release lookup, binary replacement).
- **`git/`** — git origin URL detection (auto-fills `repo_url` on first sync).
- **`display/`** — lipgloss terminal formatting (banners, status lines, key/value output).

### Templates (`templates/`)

`templates.go` defines 19 document types mapped to virtual categories:

| Category | Types |
|----------|-------|
| `knowledge` | adr, rfc, rule, guide, doc, spec |
| `vision` | prd, idea, plan, rnd, mrd, brd, urd, brs, strs, syrs, srs |
| `experience` | task-type, cpat |

Vision types beyond prd/idea/plan include the requirement tracks (MRD/BRD/URD sources; ISO 29148 BRS/StRS/SyRS/SRS) plus `rnd`, a standalone research gate (not a track). See `.archcore/document-types/`.

### Key Design Patterns

- **MCP-first document model** — CRUD and relations go through MCP tools; the CLI never edits documents directly.
- **Sync modes drive validation** — `cloud` requires `project_id`; `on-prem` requires `project_id` + `archcore_url`; `none` forbids both. Lives in `internal/config/config.go`.
- **Constructor commands** — each command is `newXxxCmd() *cobra.Command` with logic extracted into testable functions that take a base directory; version-needing commands receive the version from `NewRootCmd`.
- **Shared SessionStart hook** — all hook-supporting agents use one `newSessionStartHookCmd` / `buildSessionContext` path, differing only in event name and config format. Only `SessionStart` is active.
- **Optional settings omit defaults** — optional fields use `omitempty` with code-level defaults (e.g. `language` defaults to `"en"`).
- **Safety invariants** — never expose absolute filesystem paths in MCP errors; validate all paths against traversal; store local relations in `.sync-state.json`.
- **Co-located tests** — every package has adjacent `_test.go` using `t.TempDir()`, `httptest`, and table-driven `t.Run()` subtests; plus in-process MCP integration tests under `internal/mcp/integration/`.

### Out-of-scope directories

- `reference-materials/` — vendored third-party reference code (the `entireio` CLI) and ISO/IEC standards PDFs. **Not part of the build; exclude from code searches.**
- `examples/` — sample `.archcore/` layouts (minimal, fullstack, monorepo, global sources) for documentation and manual testing.
