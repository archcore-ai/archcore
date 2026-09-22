# Archcore

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
| [.github/workflows/](.github/workflows/) | Repository CI and plugin publication |

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
make test-integration CLI_VERSION=0.8.7-dev
```

The default `make build` version is `dev`. Neither command installs or updates
the system CLI. Run component commands from `cli/` or `plugin/`, and run
`archcore mcp` from the repository root to use the shared project context.

## Publication

This migration changes source layout and CI. CLI and plugin release versions
and public repository identifiers remain as before during this stage.

The plugin's `main` branch is generated from tagged `dev` source. Its marketplace
catalogs stay at the published root and resolve `plugins/archcore/`, preserving
existing host installation paths. The publication excludes CLI source and our
shared project context. See [the plugin release process](plugin/docs/release.md).

The CLI's original publication workflows remain under `cli/.github/workflows/`
as migration reference; GitHub Actions runs the workflows at the repository
root. A shared release version and repository rename are separate work.

## License

[Apache-2.0](LICENSE). See [NOTICE](NOTICE) and [cli/NOTICE](cli/NOTICE).
