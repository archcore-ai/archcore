# Contributing to Archcore

This file covers the repository layout, the local setup, the checks a change has to pass, and how a release happens. The product overview is the [README](README.md); user documentation lives at [docs.archcore.ai](https://docs.archcore.ai).

## Repository layout

| Path | Content |
| --- | --- |
| `cli/` | The Go CLI: MCP server, hooks, installers, and their tests. Component notes: [`cli/README.md`](cli/README.md) |
| `plugin/` | The agent plugin: skills, commands, agents, hook launchers, host manifests, marketplace catalogs, and bats tests. Component notes: [`plugin/README.md`](plugin/README.md) |
| `.archcore/` | The project's own context: architecture, decisions, rules, specs, and plans for both components |
| `.github/workflows/` | CI for both components and the shared release |
| `scripts/export-plugin.sh` | Builds the public plugin tree that a release publishes to `main` |

Branches: `dev` holds the source, and `main` is generated from a tag and carries only the published plugin tree. Open pull requests against `dev`.

## Setup

Prerequisites: Go as pinned in `cli/go.mod`, [bats-core](https://github.com/bats-core/bats-core), jq, ShellCheck, golangci-lint, and the Archcore CLI on `PATH` for the MCP integration tests (`archcore --version`).

```bash
git clone --branch dev https://github.com/archcore-ai/archcore.git
cd archcore
git submodule update --init --recursive        # bats helpers under plugin/test/helpers
make build                                     # cli → bin/archcore, version "dev"
make test                                      # go test, bats unit and structure tests
make lint                                      # go vet, golangci-lint, shellcheck
make verify                                    # lint, test, JSON and permission checks
make test-integration CLI_VERSION=0.10.1-dev   # the plugin's MCP suite against the CLI built from this tree
```

Component commands run from their directories: `cd cli && go test ./...`, `cd plugin && make verify`. Nothing here installs or updates the system CLI.

## Project context

The repository documents itself in `.archcore/`. Run `archcore mcp` from the repository root, or open the repository in an agent that has Archcore wired, and the documents are available through the MCP tools. Before changing behavior, search for the decision or rule that covers it. Record a new decision as an ADR with status `draft`; the maintainer sets `accepted`.

Two instruction files bind contributors and agents alike:

- `AGENTS.md` — the writing policy for documentation and Archcore documents.
- `cli/AGENTS.md` and the documents tagged `code-quality` — the Go conventions the linter and reviewers hold the CLI to.

## Making a change

1. Branch from `dev`.
2. Keep a change inside one component when you can. A change that crosses `cli/` and `plugin/` ships in one pull request, because both release together.
3. Add or update tests beside the code. Go tests are co-located and table-driven; plugin tests are bats files under `plugin/test/structure`, `plugin/test/unit`, and `plugin/test/integration`.
4. Run `make verify`. For a CLI change, run `make test-integration` as well.
5. Write the commit subject as `<type>: <summary>` with `feat`, `fix`, `docs`, `test`, `refactor`, or `chore`.
6. Open the pull request against `dev`. CI runs the CLI checks, the plugin checks, and the installer smoke when an installer changed.

## Releasing

One tag releases both components at the same version, and the tag is the only place that version is written. The Release workflow refuses a tag that is not the next patch, minor, or major version after the previous release tag. It then runs both test suites, regenerates `main` with the tag's version in the four plugin manifests, and publishes the CLI archives, `checksums.txt`, and the installers to the GitHub Release.

```bash
# on dev, with CI green
git tag vX.Y.Z && git push origin vX.Y.Z
```

The plugin manifests under `plugin/plugins/archcore/` stay at `0.0.0` on `dev`. Do not bump them.

Details and recovery steps: [`plugin/docs/release.md`](plugin/docs/release.md).

## Reporting problems

Open an [issue](https://github.com/archcore-ai/archcore/issues) with the host, the output of `archcore --version`, and the plugin version from the host's plugin list.
