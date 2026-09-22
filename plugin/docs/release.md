# Release process

## Branch and version model

The `dev` branch owns the monorepo source: `cli/`, `plugin/`, the shared root
`.archcore/`, and CI. The `main` branch is the generated plugin distribution.

One `vX.Y.Z` tag releases both components at the same version. The tag is the
version source of truth: the four plugin manifests under `plugin/plugins/archcore/`
carry the same string, and GoReleaser injects it into the CLI binary as
`main.version`. `archcore-ai/cli` receives no further releases; its history
lives under `cli/`.

## What one tag does

`.github/workflows/release.yml` runs on every `v*` tag, in this order:

1. `verify-version` compares the tag with the four manifests and fails on any
   mismatch.
2. `test-plugin` runs `.github/workflows/test.yml`: the plugin checks and the
   MCP integration suite against the pinned CLI and the CLI built from the
   tagged commit.
3. `test-cli` runs `.github/workflows/cli-test.yml`: gofmt, vet, golangci-lint,
   `go test ./...`, the inertness self-test, and the examples fixture check.
4. `publish-plugin` exports the public tree with `scripts/export-plugin.sh`,
   commits it as an orphan, and force-pushes `main`.
5. `publish-cli` runs GoReleaser from `cli/`: six archives, `checksums.txt`, and
   the GitHub Release for the tag with those assets plus `install.sh` and
   `install.ps1`.

Every GitHub Release therefore carries the CLI assets, and
`https://github.com/archcore-ai/plugin/releases/latest` resolves to a release
that `archcore update` and both installers consume.

## Published files (plugin tree on main)

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

## Cutting a release

Run Git commands from the repository root.

1. Run `/bump-plugin-version X.Y.Z`, or set `version` in the four manifests by hand.
2. Commit the bump on `dev` and push it. Wait for `Plugin Tests` and `CLI Tests`.
3. Push the tag: `git tag vX.Y.Z && git push origin vX.Y.Z`.
4. Watch the Release run. Expected order: `verify-version`, the two test jobs,
   `publish-plugin`, `publish-cli`.
5. Verify the outputs with the next section.

## Verification

- `gh release view vX.Y.Z` lists 9 assets: 6 archives, `checksums.txt`,
  `install.sh`, and `install.ps1`.
- `curl -sI https://github.com/archcore-ai/plugin/releases/latest` returns a
  `location:` header ending in `/releases/tag/vX.Y.Z`.
- `git fetch origin main && git ls-tree --name-only origin/main` shows the plugin
  layout only.
- `ARCHCORE_VERSION=vX.Y.Z bash cli/install.sh` from the checkout installs the
  tag, and `archcore --version` prints `archcore X.Y.Z (commit: <sha>)`.

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

The workflow has no manual dispatch: the generated default branch carries no
workflow files, so GitHub cannot offer one.

- IF `verify-version` fails, THEN fix the manifests on `dev`, delete the tag
  locally and remotely, and tag the corrected commit. Deleting a pushed tag
  rewrites published state; do it only while no GitHub Release exists for it.
- IF a test job fails, THEN nothing was published. Fix on `dev` and re-tag the
  same way.
- IF `publish-cli` fails after `publish-plugin`, THEN `main` already carries the
  new tree. Fix the cause, delete the partial GitHub Release if one exists while
  keeping the tag, and re-run the failed job from the Actions page.
