---
title: "Escaping Relative globals Paths Resolve From the Repository Main Checkout"
status: accepted
tags:
  - "config"
  - "globals"
  - "mcp"
---

## Context

`config.ResolveGlobalPath` joins a relative `globals` path onto the project root (@internal/config/globals.go). A git worktree is a different project root than the main checkout, so a declared `../global/.archcore` resolves to a directory that does not exist.

Every declared global source is mandatory (@.archcore/globals/globals-are-mandatory.adr.md), so that miss is fatal twice: `checkGlobals` aborts before `RunStdio` (@cmd/mcp.go), and `scanAll` fails every scan (@internal/docs/scan.go). A session started inside a worktree of this repository gets no Archcore MCP server at all. `archcore status` and `archcore doctor` report the same source as an issue.

Worktree tooling makes the miss systematic. Claude Code creates worktrees under `<main>/.claude/worktrees/<name>`; other tooling uses `~/conductor/workspaces/<repo>/<name>`. Neither shares the main checkout's parent directory.

Three facts constrain the fix, all measured on git 2.50.1:

1. `git worktree list --porcelain` run inside a linked worktree names the main worktree in its first `worktree ` line. `git rev-parse --git-common-dir` returns the main checkout's `.git`, from which the checkout is the parent directory.
2. Inside a submodule both queries are wrong. `--git-common-dir` returns `<super>/.git/modules/<name>`, and `worktree list --porcelain` reports the worktree as that same path, while the real checkout is `<super>/<name>`.
3. git answers with symlink-evaluated paths. On macOS a project reached as `/var/…` is reported as `/private/var/…`, so the two spellings must be reconciled before they are compared.

Two layouts constrain it further:

- **An in-tree relative path.** @examples/08-global-in-archcore declares `{ "id": "company", "path": ".archcore/global/company" }`. That directory is version-controlled, so a worktree holds its own branch's copy.
- **A project below the working tree root.** @examples/05-global-single-source is a project inside this repository, and its `../_global_/company-standards/.archcore` is declared relative to the example directory, not to the repository root.

## Decision

A relative `globals` path that escapes the project root resolves against the project root's own position inside the repository main checkout. Every other path keeps today's resolution.

1. The path is cleaned and classified. A path that stays inside the project root is in-tree and resolves against the project root, so a worktree reads its own branch's content. An absolute path is unaffected — @.archcore/globals/global-sources.spec.md §1.1 already returns it cleaned.
2. The anchor is `<main checkout>/<project root relative to its own working tree root>`. Mapping the position, rather than anchoring on the main checkout root, keeps a project below the working tree root resolving exactly as before.
3. Both working trees come from @internal/git/: `rev-parse --show-toplevel` for the current one, `worktree list --porcelain` for the main one, each under the package's existing 500 ms bound.
4. Both sides are symlink-evaluated before the relative position is computed.
5. The derived anchor is accepted only when it is an existing directory that contains `.archcore/`. The submodule case fails this check, and so does a bare repository, a non-git project, and a machine without git.
6. IF the current working tree equals the main one, or the derivation is rejected or unavailable, THEN resolution falls back to the project root — today's behavior.
7. The mandatory-source contract is unchanged. A source that is still missing after resolution stays fatal at startup and at scan time.

`ResolveGlobalPathFrom(baseDir, anchor, path)` is the pure core: it runs no subprocess and touches no filesystem. `ResolveGlobalPath(baseDir, path)` wraps it with the anchor lookup, which is memoized per project root and reached only for an escaping path — a project that declares none never spawns git.

## Alternatives

- **Anchor on the main checkout root rather than the project's position inside it.** Rejected during implementation: it broke every nested fixture under @examples/, which resolved their declared sibling globals against the repository root instead of the example directory.
- **Resolve every relative path from the anchor, with no in-tree carve-out.** One rule, no classification step. Rejected: @examples/08-global-in-archcore would read the main checkout's branch content from inside a worktree, which is a silent wrong answer rather than a visible failure.
- **Keep resolution and downgrade a missing source to a warning** (option 2 of the issue). Rejected: it reverses @.archcore/globals/globals-are-mandatory.adr.md and @.archcore/globals/global-sources.spec.md §6, including the `No silent-skip` constraint and the invariant that the startup gate and the runtime scan classify a source identically. It also costs four surfaces — `checkGlobals`, `scanAll`, `checkGlobalSources` in @cmd/status.go, and the SessionStart `GLOBALS` block — and every worktree session would then run on a smaller corpus with no in-band signal.
- **Derive the main checkout from `git rev-parse --git-common-dir` and take the parent directory.** Rejected: the parent is the checkout only when the common directory is named `.git`. A bare repository and `git init --separate-git-dir` both break it, and the acceptance check would carry the whole burden.
- **Require absolute `globals` paths.** Rejected: @.archcore/globals/declaring-global-sources.rule.md rule 8 permits `../`, the committed declaration is the portable form, and no migration path exists for the fixtures under @examples/.
- **Thread the anchor through every resolution call site.** Rejected: `ResolveGlobalPath` is reached from the scan, the read-path validation, the write guard, the health reporter, and the sync hash, and the anchor would have to cross five packages to arrive unchanged. A memoized lookup behind one named seam keeps the pure core testable without that churn.

## Consequences

- @internal/git/ gains two read-only queries and a `Roots` pair. It is the only package that runs git, and the new helpers inherit the 500 ms bound, the `ErrGitAbsent` sentinel, and the empty-result fallback contract of `DetectRepoURL` and `CurrentBranch`.
- @internal/config/globals.go gains a process-lifetime memo keyed by project root, and a package-level seam for the lookup — the pattern `lookPath` already uses in @internal/git/git.go. The answer cannot change while one process serves one checkout.
- The escaping branch returns a symlink-evaluated path. The walk root and the prefix used for source matching both come from that same value, so annotation and scanning stay consistent.
- @.archcore/globals/global-sources.spec.md §1 states the resolution rule and is amended with it.
- A clone placed at a different depth than the declared path expects stays broken. The main checkout is that clone, so no derivation helps. Distribution of global sources remains out of scope per @.archcore/globals/global-sources.spec.md.
- `archcore status` and `archcore doctor` report a healthy source inside a worktree, so a worktree stops producing a permanent non-zero exit. Measured in a worktree of this repository: `status` exits zero and the MCP server answers `list_documents` with `by_source: {"archcore": 43, "local": 108}`, while the released v0.8.1 binary in the same directory reports the source missing.
- This decision is a prerequisite for the usefulness of @.archcore/mcp/session-following-project-root.adr.md: the acceptance gate there refuses a candidate root whose globals do not resolve, and in this repository every worktree is exactly that candidate.
