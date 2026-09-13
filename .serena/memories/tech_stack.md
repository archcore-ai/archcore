# tech_stack

- Go, module `archcore-cli`, `go 1.25.11` in `go.mod` (Go 1.25+ required).
- CLI: `spf13/cobra`. TUI prompts: `charmbracelet/huh`, styling `charmbracelet/lipgloss`.
- MCP server (stdio): `mark3labs/mcp-go`.
- YAML frontmatter: `gopkg.in/yaml.v3`.
- Lint: golangci-lint v2 (CI pins `v2.12.2`), config `.golangci.yml` (errcheck, errname, exhaustive, revive, staticcheck incl. ST checks, unconvert).
- Release: goreleaser via `.github/workflows/release.yml`; post-build hook runs `scripts/assert-not-inert.sh`.
- CI `test.yml`: gofmt check → `go vet` → golangci-lint → `go test` → inertness guard self-test → `examples/` sync check.
