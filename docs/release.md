# Release process

## Branch model

- **`dev`** is the source of truth. All PRs, all tests, all reviews land here.
  `dev` carries the plugin team's own `.archcore/` knowledge base,
  reference materials, the bats test suite, the Makefile, CI workflows,
  and our local host configs (`.claude/`, `.codex/`).

- **`main`** is the public distribution surface. It is **synthesized**
  by `.github/workflows/release.yml` from a tagged commit on `dev` and
  **must not** be edited directly. Anyone who clones the repo as a
  Cursor / Claude Code / Codex plugin gets `main`.

## Why the split exists

Host plugin systems (notably Cursor 2.5+) clone the entire repository
into their plugin cache and may spawn the plugin's MCP server with
`cwd` pointing at the install directory — not at the user's workspace
([forum #99215](https://forum.cursor.com/t/how-get-the-correct-current-work-directory-in-mcp-server/99215),
[forum #74861](https://forum.cursor.com/t/allow-workspacefolder-in-mcp-project-configration/74861)).
If our own `.archcore/` ships with the plugin, the MCP server reads it
and serves the plugin team's dev docs to every user as if they were the
user's own knowledge base. The dev → main split removes that bundled
`.archcore/` (and other dev-only artifacts) from the public surface.

This complements two other defenses:

- **CLI guard** in `archcore mcp` — refuses to serve when the working
  directory looks like a plugin install dir (sibling `.cursor-plugin/`,
  `.claude-plugin/`, `.codex-plugin/`, or `.plugin/` manifests). See `archcore-cli`
  repo.
- **Hook guard** in `bin/session-start` — same detection, silent exit.

The dev → main split is the first line of defense; the guards are
last-line safety nets.

## Blocklist (what gets removed when synthesizing `main`)

The release workflow strips these paths before force-pushing to `main`.
Any addition or removal MUST update the workflow at
`.github/workflows/release.yml` AND this section together.

| Path                  | Why                                                    |
| --------------------- | ------------------------------------------------------ |
| `.archcore/`          | Plugin team's own knowledge base. The smoking gun.     |
| `reference-materials/`| Vendored docs (ISO PDFs, sample projects). Dev-only.   |
| `test/`               | bats test suite. Runs only in CI on `dev`.             |
| `.claude/`            | Plugin team's local Claude Code settings.              |
| `.codex/`             | Plugin team's local Codex settings.                    |
| `.github/`            | CI workflows. `main` is synthesized, not edited.       |
| `Makefile`            | Dev-targets. Users never run `make` on installed plugins. |
| `docs/release.md`     | This file. Lives on `dev` only.                        |
| `cursor.mcp.json`     | Legacy path; must already be gone but stripped defensively. |

Everything else ships. The plugin itself lives under **`plugins/archcore/`**
— a dedicated subdirectory required so Codex can discover it (Codex
marketplace `source.path` must point at a subdir, not the repo root; see the
multi-host layout ADR and issue #2). That directory carries `skills/`,
`agents/`, `commands/`, `rules/`, `hooks/`, `bin/`, `assets/` (icon + logo
for marketplace surfaces), the per-host manifests
(`plugins/archcore/.claude-plugin/plugin.json`,
`plugins/archcore/.cursor-plugin/plugin.json`,
`plugins/archcore/.codex-plugin/plugin.json`,
`plugins/archcore/.plugin/plugin.json`), and the MCP configs
(`plugins/archcore/.mcp.json`, `plugins/archcore/.codex.mcp.json`).

At the **repo root** the marketplace catalogs ship and point at the
subdirectory: `.agents/plugins/marketplace.json` (Codex),
`.claude-plugin/marketplace.json` (Claude), and
`.cursor-plugin/marketplace.json` (Cursor) — each with
`source`/`path` = `./plugins/archcore`. Also at the root:
`docs/cursor.mcp.example.json`, `docs/TERMS.md`, `README.md`, `LICENSE`,
`NOTICE`.

> **Note on `assets/`.** Required by `plugins/archcore/.codex-plugin/plugin.json`
> (`interface.composerIcon`, `interface.logo`). The plugin.json paths are
> relative to plugin root (`plugins/archcore/`), so `plugins/archcore/assets/`
> must exist alongside the manifests in the published `main` tree. Don't strip.

## Cutting a release

1. Bump `version` in all four manifests
   (`plugins/archcore/.claude-plugin/plugin.json`,
   `plugins/archcore/.cursor-plugin/plugin.json`,
   `plugins/archcore/.codex-plugin/plugin.json`,
   `plugins/archcore/.plugin/plugin.json`).
2. Merge the bump PR to `dev`.
3. Tag the merge commit: `git tag v0.4.1 && git push origin v0.4.1`.
4. The `release.yml` workflow runs:
   - Re-runs `make all` against `dev` (full bats + shellcheck + JSON validation).
   - Verifies the tag is reachable from `origin/dev` (refuses to release
     from any other lineage).
   - Strips the blocklist.
   - Verifies no distributable file references plugin-internal
     `.archcore/<category>/<slug>.<type>.md` paths (mirrors
     `test/structure/no-bundled-archcore-refs.bats`).
   - Creates an orphan commit and force-pushes to `main`.
   - Publishes a GitHub Release with auto-generated notes.
5. Users update via their host's plugin marketplace (which pulls from
   `main`).

## Manual sync

`workflow_dispatch` is provided for first-time sync and emergencies. Run
it from the GitHub Actions UI with `source_ref` pointing at the dev
commit to publish. Manual dispatch does not create a GitHub Release —
tag the source ref afterwards if you want one.

## Adding a new item to the blocklist

1. Edit `.github/workflows/release.yml` — add the `rm` line in the
   "Strip dev-only artifacts" step.
2. Edit this file — add a row in the blocklist table with a one-line
   justification.
3. Sanity-check locally:
   ```sh
   git worktree add /tmp/release-test HEAD
   cd /tmp/release-test
   # run the rm commands manually, inspect the result
   ```

## Troubleshooting

**Release workflow fails on "tag is not on dev lineage."**
The tag was created on a feature branch, not on `dev`. Re-tag the dev
commit that the feature branch landed at.

**Release workflow fails on "distributable files reference plugin-internal
.archcore docs."** A skill / agent / rule introduced a literal path into
`.archcore/plugin/...` (or `.archcore/<other-category>/...`). Replace it
with a semantic reference (`.archcore/` as a generic concept describing
the user's knowledge base, not a path to our docs). The same check runs
in `test/structure/no-bundled-archcore-refs.bats` on every PR.
