# Plugin release process

## Branch model

The `dev` branch owns the monorepo source: `cli/`, `plugin/`, the shared root
`.archcore/`, and CI. The `main` branch remains the generated plugin distribution.
The CLI and plugin release versions are still independent at this migration stage.

The release workflow builds the published tree with `scripts/export-plugin.sh`.
It exports only named runtime files from `plugin/`, removing that source prefix.
This preserves the existing marketplace and Copilot installation paths.

## Published files

| Source on dev | Path on main |
| --- | --- |
| `plugin/plugins/archcore/` | `plugins/archcore/` |
| `plugin/.agents/plugins/marketplace.json` | `.agents/plugins/marketplace.json` |
| `plugin/.claude-plugin/marketplace.json` | `.claude-plugin/marketplace.json` |
| `plugin/.cursor-plugin/marketplace.json` | `.cursor-plugin/marketplace.json` |
| `plugin/docs/TERMS.md` | `docs/TERMS.md` |
| `plugin/docs/cursor.mcp.example.json` | `docs/cursor.mcp.example.json` |
| `plugin/README.md`, `plugin/demo.gif`, `plugin/3-commands.png` | Their root filenames |
| `LICENSE`, `NOTICE` | Their root filenames |

All three catalogs continue to resolve `./plugins/archcore`. The runtime includes
its four host manifests, MCP configs, skills, agents, copilot-agents, commands,
rules, hooks, bin scripts, and assets/. Codex composerIcon and logo paths still
resolve from the plugin root. Copilot still installs the published
`archcore-ai/plugin:plugins/archcore` subdirectory.

## Excluded files

The exporter starts with an empty destination. Files outside the published list
never enter it, including `cli/`, the shared `.archcore/`, `test/`,
`reference-materials/`, `.github/`, `.claude/`, `.codex/`, `.gitmodules`,
`Makefile`, `docs/release.md`, `AGENTS.md`, `CLAUDE.md`, and the dev-only `.mcp.json`.

A recursive check rejects nested development directories, development instruction
files, dev MCP config, and symlinks inside the exported runtime. The existing check
for literal references to bundled internal Archcore documents also runs on the
actual output. Plugin cache guards in the CLI and runtime remain additional defenses.

The shared `.archcore/` contains the team's development context. Publishing it
could make a host that starts MCP in the plugin cache serve that context as the
user's project. CLI examples contain fixture `.archcore/` directories, so removing
only a root `.archcore/` from a whole monorepo checkout would be insufficient.

## Cutting a plugin release

Run Git commands from the repository root.

1. Update all four manifests in `plugin/plugins/archcore/` to the plugin release version.
2. Merge the version change into `dev`.
3. Push the corresponding `vX.Y.Z` tag on that commit.
4. Verify the release workflow and the generated `main` tree.

The workflow runs plugin checks and MCP integration against both the pinned CLI
and the CLI source in the same commit. It verifies the tag's dev lineage, exports
the public tree, creates an orphan commit, force-pushes `main`, and publishes the
plugin GitHub Release. It does not publish CLI binaries in this migration stage.

## Local verification

Run these commands from the repository root:

```sh
make -C plugin all
./scripts/export-plugin.sh /tmp/archcore-plugin-release-check
```

The destination must be empty and must not be a symlink. Inspect the resulting
catalogs and `plugins/archcore/`. `plugin/test/structure/release-blocklist.bats`
checks the actual export, including rejection of nested development context.

## Recovery

Rerun a failed tag workflow after correcting its cause. A manual `source_ref`
input remains in the workflow, but GitHub requires a workflow file on the default
branch for manual dispatch. The generated `main` excludes `.github/`, so manual
availability is not guaranteed by this layout. A dedicated default-branch
dispatcher belongs to the later release redesign.
