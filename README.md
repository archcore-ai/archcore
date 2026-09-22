# Archcore

> **One repository since 2026-09-22.** This repository was `archcore-ai/plugin`; GitHub redirects that address. The CLI moved here from [`archcore-ai/cli`](https://github.com/archcore-ai/cli), now archived, and lives under [`cli/`](cli/). Both components release together from one tag.

Git-native project context for AI coding agents. Archcore keeps specifications,
decisions, rules, plans, and project knowledge in Git, and gives agents access
through MCP tools, hooks, and skills.

## Install

```bash
curl -fsSL https://archcore.ai/install.sh | bash
archcore init
```

For Windows, run `irm https://archcore.ai/install.ps1 | iex`, then `archcore init`.
See [archcore.ai](https://archcore.ai) for supported hosts and setup instructions.

## Source layout

This `dev` branch contains both components of Archcore:

| Path | Content |
| --- | --- |
| [cli/](cli/) | Go CLI, MCP server, hooks, installers, and CLI tests |
| [plugin/](plugin/) | Agent skills, host adapters, marketplace catalogs, and plugin tests |
| [.archcore/](.archcore/) | Shared architecture, specifications, rules, plans, and document relations |
| [.github/workflows/](.github/workflows/) | Repository CI and the release of both components |

The CLI was imported with its Git history. Both components use the root
`.archcore/`; the small `.archcore/` directories in `cli/examples/` are test and
documentation fixtures, not separate component knowledge bases.

## Development

Install Go as specified in `cli/go.mod`, Bats, jq, ShellCheck, and golangci-lint.
Initialize the plugin test helpers after cloning:

```bash
git switch dev
git submodule update --init --recursive
make build
make test
make lint
```

Run integration tests with a development version that satisfies the plugin's
existing CLI capability gates:

```bash
make test-integration CLI_VERSION=0.10.1-dev
```

The default `make build` version is `dev`. Neither command installs or updates
the system CLI. Run component commands from `cli/` or `plugin/`, and run
`archcore mcp` from the repository root to use the shared project context.

## Publication

One `vX.Y.Z` tag releases both components at the same version. The Release
workflow checks that the four plugin manifests equal the tag, runs the plugin
and CLI test suites, regenerates `main` from the exported `plugin/` tree, and
runs GoReleaser from `cli/` to attach the CLI archives, `checksums.txt`, and
both installers to the GitHub Release. Marketplace catalogs stay at the
published root and resolve `plugins/archcore/`, preserving existing host
installation paths. See [the release process](plugin/docs/release.md).

`archcore-ai/cli` is the former CLI repository and `archcore-ai/plugin` the former
name of this one (renamed 2026-09-22; GitHub redirects the old address). Neither
receives separate releases. The copies under `cli/.github/workflows/`
are migration reference only.

## License

[Apache-2.0](LICENSE). See [NOTICE](NOTICE) and [cli/NOTICE](cli/NOTICE).
