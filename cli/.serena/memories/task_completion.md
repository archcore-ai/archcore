# task_completion

Run before reporting Go work done (mirrors CI `test.yml`):
1. `gofmt -l .` — must print nothing.
2. `go vet ./...`
3. `golangci-lint run ./...` — required by `CLAUDE.md`; passing it does not prove code-quality rules are met.
4. `go test ./...`
5. If instruction writers, templates or hook/instruction output changed: `./scripts/regen-examples.sh` and commit resulting `examples/` diff.
6. If release scripts or ldflags-injected variables changed: `./scripts/assert-not-inert-selftest.sh`.

Also: update user-facing docs when command surface changes; keep link from code change to governing Archcore spec/plan in the result.
