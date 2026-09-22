# suggested_commands

- Build: `go build -o archcore .`
- All tests: `go test ./...`; one package: `go test ./cmd/`; one test: `go test ./cmd/ -run TestName`
- MCP integration tests: `go test ./internal/mcp/...`
- Format check: `gofmt -l .` (fix: `gofmt -w .`)
- Vet: `go vet ./...`
- Lint: `golangci-lint run ./...`
- Regenerate example instruction fixtures: `./scripts/regen-examples.sh`, then `git diff --exit-code -- examples/`
- Release guard self-test: `./scripts/assert-not-inert-selftest.sh`

## Darwin notes
- User shell is fish: no `$(...)` heredoc habits in interactive snippets; bash tool calls are unaffected.
- BSD `sed -i` needs `''` argument; prefer Go/Serena edits over sed.
