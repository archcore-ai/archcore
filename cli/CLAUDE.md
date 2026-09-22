# CLAUDE.md

Read and follow `AGENTS.md` before creating or editing technical documentation, Archcore documents, CLI help, MCP tool descriptions, prompts, agent instructions, or user-facing Markdown.

The writing policy in `AGENTS.md` applies two profiles: an ASD-STE100-inspired profile constrains the sentence, and an ISO 24495-1-inspired profile constrains the structure. The document type decides which half binds. The per-type assignment — profile, line format, and metric — lives in the shared Archcore rule `concepts/document-prose-canon`.

Do not claim formal ASD-STE100 or ISO 24495-1 compliance.

## Search Priority

When researching patterns, decisions, or conventions in this project, always search `.archcore/` documents first (`list_documents` → `get_document`) before grepping the codebase or using external sources.

`.archcore/architecture/` holds the structure of the system: `system-overview.doc` is the map,
`package-dependency-direction.rule` binds the import graph, `process-and-concurrency-model.spec` binds
shared state, and `advisory-subsystem.doc` describes the hook advisories. Read the map before a change
that crosses a package boundary.

`.archcore/cli-ui/building-the-cli.doc.md` holds the procedures. Consult it before adding a command,
document type, MCP tool, hook, or agent.

## Go Code Quality — Mandatory

`.archcore/code-quality/` holds the binding coding agreements for this repository. They are not
advisory.

Before you write or edit any `.go` file in `cmd/`, `internal/`, `templates/`, or `main.go`, load the
code-quality documents. Load them once per session and keep them in context for the rest of the
session.

1. Call `list_documents` with `tags: ["code-quality"]`.
2. Call `get_document` for each returned path.
3. Apply every clause to the code you write.

The documents and what each one governs:

| Document                                          | Binds                                                                                    |
| ------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `go-code-quality.rule`                            | Error handling, exit codes, package layout, imports, allocation, general Go conventions. |
| `strict-go-naming-conventions.rule`               | Every identifier name. Absolute for new code.                                            |
| `comments-are-the-exception.rule`                 | Whether a comment is written at all. The default is none.                                |
| `fail-open-or-fail-closed-reads.rule`             | Every read of `settings.json`, a host config, the filesystem, or git.                    |
| `choosing-an-atomic-write.rule`                   | Every write that must not be observable half-finished.                                   |
| `bounded-and-deterministic-output.rule`           | Every collection or stream that leaves the process.                                      |
| `cite-the-governing-document-from-code.rule`      | Comments on code that exists because of a recorded decision.                             |
| `shared-guards-return-classified-sentinels.rule`  | A predicate more than one surface consults.                                              |
| `rendering-happens-at-the-boundary.rule`          | What a domain package returns, and who formats it.                                       |
| `platform-splits-are-files.rule`                  | Behavior that differs by `GOOS` or `GOARCH`.                                             |
| `the-shape-of-an-mcp-tool-file.rule`              | Every file under `internal/mcp/tools/` that adds a tool.                                 |
| `unit-testing-patterns.guide`                     | Every `_test.go` file.                                                                   |
| `isolating-the-machine-from-the-test-suite.guide` | `TestMain` in a package whose code reaches `$HOME`, XDG state, a host CLI, or git.       |
| `registry-agreement-and-test-seams.guide`         | A new registry, a matcher over one, or a test seam into production code.                 |
| `in-process-mcp-integration-tests.adr`            | Tests that cross MCP tool boundaries.                                                    |
| `e2e-testing-for-cli.idea`                        | End-to-end coverage layers. Historical record; read the ADR above for current state.     |

`.archcore/architecture/package-dependency-direction.rule` binds Go source too. It governs which
package may import which.

The pre-write hook injects the documents that name the directory you are editing — see
`.archcore/architecture/advisory-subsystem.doc`. That injection is a safety net over a directory
match, not the delivery mechanism for the set: a rule that names no directory reaches you only through
the tag load above.

Rules to follow when the agreements bind:

1. Apply the rule. Do not restate it in the response.
2. IF a change must deviate from a clause, THEN add an inline comment that names the clause and the
   reason.
3. IF the code raises a question no document answers, THEN state the gap and propose a document
   rather than inventing an undocumented convention.
4. Run `golangci-lint run ./...` before you report Go work as complete. The configuration in
   `.golangci.yml` enforces a subset of `strict-go-naming-conventions.rule`; passing the linter is
   not proof that the other documents are satisfied.

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

### Archcore + Serena

Applies when the session lists any Serena tool, including a deferred tool that shows only its name.

Session start:

1. Before the first code read, search Archcore for accepted decisions, specifications, and rules on the affected area.
2. Treat accepted Archcore records as constraints on the change.
3. If an accepted Archcore record conflicts with the request, report the conflict to the user.
4. Make no code edit until the user resolves the Archcore conflict.
5. If the host shows Serena tools by name only, load their schemas through tool search before the first code read.
6. Before the first code read, call Serena `initial_instructions` once per session.
7. If no Serena project is active, call `activate_project` for the current repository.

Tool choice:

8. To find where a symbol is used, call `find_referencing_symbols` instead of text search.
9. To find where a symbol is defined, call `find_symbol` instead of text search.
10. To inspect a code file, call `get_symbols_overview` before reading the whole file.
11. To read one symbol, call `find_symbol` with `include_body`.
12. To rename a symbol, call `rename_symbol`.
13. To delete a symbol, call `safe_delete_symbol` when the toolset has it.
14. To find a string, JSON key, configuration key, path, or prose phrase, use text search.
15. To find a project decision, call Archcore `search_documents` instead of reading Serena memory.

Changes:

16. Before changing a symbol's name, type, signature, fields, or serialization annotation, call `find_referencing_symbols` on it.
17. Call `find_referencing_symbols` for a one-line symbol change too.
18. For serialized field names and string keys, also run a text search across code, configuration, and docs.
19. When any skill, including `/archcore:*`, or a plan step says to read code, use Serena tools.
20. To replace a whole symbol, call `replace_symbol_body`.
21. Use ordinary file edits for local changes inside a symbol, prose, configuration, and generated files.
22. If a Serena tool is missing, fails, or lacks language support, use ordinary tools for that file.
23. Name each Serena limitation in the task result.
24. After code edits, run the relevant tests.
25. Report `get_diagnostics_for_file` output beside the test results, never instead of them.

Records:

26. Record architecture and product decisions in Archcore, not in Serena memory.
27. Create every new `.archcore/**/*.md` record with status `draft`.
28. Set status `accepted` only after the user accepts the record.
29. If a document carries `read_only: true`, name its owning project instead of writing to it.
30. If more than one writable `.archcore/` is present, confirm the target project before writing a record.
31. If the project has no `.archcore/`, do not call `init_project` without the user's request.

Result:

32. After a task that changed code, end with `Serena: <tools used>` or `Serena: not used — <reason>`.
33. For a change covered by an Archcore specification or plan, name that record in the result.
34. If another connected recipe conflicts with these instructions, report the conflict in the task result.
35. If another connected recipe conflicts with these instructions, keep that recipe's applicable contribution.

## Managed Blocks

Do not edit content inside an Archcore-managed block. Archcore delimits a
managed block with a start marker and an end marker, each an HTML comment
that names the block (`archcore:start`, `archcore:end`).

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
- `plugin` installs, updates, removes, and reports the Archcore plugin on hosts that ship one.
- `sync` pushes `.archcore/` state to a configured server.
- `update` updates the CLI and the plugin. `archcore mcp` runs the same update unattended, under the policy in `internal/update`.

When adding a command:

1. Read `.archcore/cli-ui/building-the-cli.doc.md`.
2. Use the constructor-command pattern.
3. Keep command logic in testable functions.
4. Add co-located tests.
5. Update user-facing documentation when the command surface changes.

### Internal packages

There are 18 packages under `internal/`. Keep this list complete.

- `internal/docs/` owns the `.archcore/` document domain: the document model, the filesystem scan, the global-source predicates, and the path guards. It carries no MCP dependency.
- `internal/config/` manages settings, initialization, and globals resolution.
- `internal/mcp/` implements the MCP server, the session-following root provider, and the stdio shield.
- `internal/mcp/tools/` implements MCP tools.
- `internal/mcp/integration/` contains in-process MCP integration tests.
- `internal/advisory/` implements the four hook advisories: code alignment, precision, restatement, and staleness.
- `internal/agents/` defines supported agent integrations.
- `internal/wiring/` implements host wiring.
- `internal/sync/` implements sync state, hashing, and payload construction.
- `internal/api/` implements the server API client.
- `internal/update/` implements self-update and the unattended update policy.
- `internal/plugin/` plans and executes plugin actions per host.
- `internal/telemetry/` sends the update events.
- `internal/stamp/` records cross-process claims in the shared state directory.
- `internal/xdg/` resolves the shared state directory.
- `internal/git/` detects repository metadata.
- `internal/jsonfile/` performs order-preserving atomic surgery on JSON config files archcore does not own.
- `internal/projectroot/` holds the checks a project root must pass before it is served.
- `internal/display/` formats terminal output.
- `internal/testsupport/` holds test-only helpers shared by more than one package.
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
- Unattended update refuses without the official-build marker.
- Telemetry stays inert without an injected key.
- The plugin surface never changes behavior outside itself.

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
