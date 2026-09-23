# Archcore CLI

The Go half of [Archcore](../README.md): a CLI and a local stdio MCP server that keep a project's `.archcore/` documents and serve them to coding agents through MCP tools and lifecycle hooks. Product overview and installation: the [repository README](../README.md) and [docs.archcore.ai](https://docs.archcore.ai). This file is for people and agents who change the CLI.

## Layout

| Path | Owns |
| --- | --- |
| `main.go` | Entry point; `version` is injected at build time |
| `cmd/` | Cobra commands `init`, `mcp`, `status`, `doctor`, `config`, `hooks`, `instructions`, `plugin`, `sync`, `update`, and the hook handlers |
| `internal/docs/` | The document model, the filesystem scan, global-source predicates, path guards |
| `internal/config/` | `settings.json`, initialization, globals resolution |
| `internal/mcp/`, `internal/mcp/tools/` | The MCP server, the session-following root provider, the stdio shield, one file per tool |
| `internal/advisory/` | The hook advisories: code alignment, precision, restatement, staleness |
| `internal/agents/`, `internal/wiring/` | The host registry and the host wiring: hooks, MCP config, instruction files |
| `internal/plugin/` | Plugin delivery: evidence, planner, executor, source migration |
| `internal/update/`, `internal/telemetry/`, `internal/stamp/`, `internal/xdg/` | Self-update and the unattended policy, update events, cross-process claims, the state directory |
| `internal/sync/`, `internal/api/` | Sync state, hashing, payloads, and the server client |
| `internal/git/`, `internal/jsonfile/`, `internal/projectroot/`, `internal/display/`, `internal/testsupport/` | Git metadata, order-preserving JSON edits, project-root checks, terminal output, shared test helpers |
| `templates/` | The 23 document types and their templates |
| `examples/` | Generated instruction-file fixtures; regenerate with `scripts/regen-examples.sh` |
| `scripts/` | The release inertness guard and fixture generation |

The design lives in the repository's `.archcore/`: `architecture/` maps the system and binds the import graph, `cli-ui/building-the-cli.doc` holds the procedure for adding a command, a document type, an MCP tool, a hook, or an agent, and the documents tagged `code-quality` bind the Go conventions. `CLAUDE.md` in this directory lists them.

## Build and test

```bash
go build -o archcore .                  # local build, version "dev"
go test ./...
go vet ./... && golangci-lint run ./...
./scripts/regen-examples.sh             # after a change to the instruction writer
```

From the repository root, `make build` writes `bin/archcore` and `make test-integration` runs the plugin's MCP suite against it. Use Go 1.25 or newer.

## Surface

### Commands

| Command | Description |
| --- | --- |
| `archcore init` | Create `.archcore/`, pick hosts on a screen where detected agents are pre-checked, wire them, install the plugin for the selected hosts |
| `archcore mcp` | Run the MCP stdio server; `--project` or `ARCHCORE_PROJECT_ROOT` pins the root |
| `archcore mcp install` | Write MCP config for detected agents |
| `archcore hooks install` | Install lifecycle hooks for detected agents |
| `archcore hooks <host> <event>` | The hook entry points the plugin launchers call: `session-start`, `pre-tool-use`, `post-tool-use` |
| `archcore instructions` | Manage the Archcore block in instruction files |
| `archcore plugin` | `install`, `update`, `remove`, or `status` for the plugin per host |
| `archcore status` | Check `.archcore/` structure and document health |
| `archcore doctor` | Check the setup and fix issues |
| `archcore config` | Read or set `.archcore/settings.json` |
| `archcore sync` | Push `.archcore/` state to a configured server |
| `archcore update` | Update the binary, then the plugin on every host that carries it |

### MCP tools

11 tools: `list_documents`, `search_documents`, `get_document`, `create_document`, `update_document`, `remove_document`, `add_relation`, `remove_relation`, `list_relations`, `init_project`, `install_host_config`. The server starts in an empty repository; an agent bootstraps `.archcore/` through `init_project`.

### Hooks

Three events. `session-start` injects the recap of decisions and work in progress. `pre-tool-use` refuses direct writes to `.archcore/` documents and injects the documents that name the edited path. `post-tool-use` reports precision findings on a written document. Each host has a dialect for its payload and its deny shape; the plugin's `bin/` launchers call `archcore hooks <host> <event>`.

### Configuration

`.archcore/settings.json`: `sync` (`none`, `cloud`, `on-prem`), `language` (document language, default `en`), and `globals` (read-only external sources, each with an `id` and a `path`). `archcore config`, `archcore config get <key>`, `archcore config set <key> <value>`.

### Update and telemetry

`archcore update` resolves the newest release from `https://github.com/archcore-ai/archcore/releases/latest`, verifies the SHA-256 checksum, replaces the binary atomically, then updates the plugin where a host carries it. `archcore mcp` runs the same check unattended at most once per 24 hours per machine and only for an official build. A release build sends one event per update attempt without paths, names, or repository data; `DO_NOT_TRACK=1` or `ARCHCORE_TELEMETRY_OPTOUT=1` stops the events and leaves updates working. Details: [archcore.ai/privacy](https://archcore.ai/privacy).

## Design constraints

- MCP-first document management; the CLI subcommands handle setup, health, sync, host wiring, and updates.
- Free-form `.archcore/` layout; a document's category is derived from its type suffix.
- Path traversal protection and no absolute filesystem paths in MCP errors.
- Shared host-wiring logic in `internal/wiring/`; co-located, table-driven tests.
- Global sources are read-only everywhere.
- Unattended update refuses without the official-build marker; telemetry stays inert without an injected key.
- The plugin surface never changes behavior outside itself.
