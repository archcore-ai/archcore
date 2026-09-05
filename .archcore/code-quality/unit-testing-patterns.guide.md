---
title: "Unit Testing Patterns for Archcore CLI"
status: accepted
tags:
  - "code-quality"
  - "code-review"
  - "golang"
  - "testing"
---

## Prerequisites

- Go 1.25+
- Familiarity with Go's `testing` package
- Read `go-code-quality.rule.md` for general conventions
- Read `comments-are-the-exception.rule.md` before commenting a test

## Steps

### 1. Test File Organization

Place tests in the **same package** as the source — not in a `_test` suffix package. This allows testing unexported functions directly.

```
cmd/sync.go       → cmd/sync_test.go       (package cmd)
internal/config/  → internal/config/config_test.go (package config)
internal/mcp/tools/ → internal/mcp/tools/create_document_test.go (package tools)
```

Use `t.Parallel()` at the start of tests that have no global state mutations (no `os.Chdir`, no shared package-level vars).

Every non-`main` package carries tests. `internal/projectroot` was the last package without them and now has `plugincache_test.go`.

### 2. Table-Driven Tests (Primary Pattern)

Every test with multiple scenarios uses the table-driven pattern:

```go
func TestCheckSyncPreconditions(t *testing.T) {
    tests := []struct {
        name        string
        setup       func(t *testing.T, baseDir string)
        wantErr     bool
        errContains string
        wantPID     *int
    }{
        {
            name:    "no .archcore dir",
            setup:   func(t *testing.T, baseDir string) {},
            wantErr: true,
            errContains: ".archcore/",
        },
        {
            name: "valid cloud with project_id",
            setup: func(t *testing.T, baseDir string) {
                config.InitDir(baseDir)
                s := config.NewCloudSettings()
                s.ProjectID = &pid
                config.Save(baseDir, s)
            },
            wantErr: false,
            wantPID: &pid,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            baseDir := t.TempDir()
            tt.setup(t, baseDir)
            // ... assertions
        })
    }
}
```

**Conventions:**

- `name string` is always the first field.
- Common fields: `setup func(t, baseDir)`, `wantErr bool`, `errContains string`, domain-specific `want*` fields.
- Loop variable: `tt` (not `tc`, `c`, or `test`). See `strict-go-naming-conventions.rule` §K.
- Each subtest is fully independent — no shared state between iterations.

### 3. Assertions

Use pure `testing.T` — **no assertion libraries** (no testify, no gomega).

**`t.Fatalf`** for precondition failures where the test cannot continue:

```go
if err != nil {
    t.Fatalf("unexpected error: %v", err)
}
```

**`t.Errorf`** for value mismatches where the test can check more things:

```go
if got != want {
    t.Errorf("FuncName() = %q, want %q", got, want)
}
```

**Error presence check:**

```go
if (err != nil) != tt.wantErr {
    t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
}
```

**Substring match for error messages:**

```go
if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
    t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
}
```

### 4. Filesystem Testing

**Always use `t.TempDir()`** — never hardcoded paths or shared directories.

**Setup helpers** are marked with `t.Helper()` and call `t.Fatal` on failure:

```go
func setupArchcoreDir(t *testing.T) string {
    t.Helper()
    base := t.TempDir()
    if err := os.MkdirAll(filepath.Join(base, ".archcore"), 0o755); err != nil {
        t.Fatal(err)
    }
    return base
}
```

**Document creation helper:**

```go
func writeDoc(t *testing.T, dir, subdir, filename, content string) {
    t.Helper()
    d := filepath.Join(dir, ".archcore", subdir)
    os.MkdirAll(d, 0o755)
    if err := os.WriteFile(filepath.Join(d, filename), []byte(content), 0o644); err != nil {
        t.Fatal(err)
    }
}
```

**Skip permission tests when running as root:**

```go
if os.Getuid() == 0 {
    t.Skip("cannot test permission errors as root")
}
```

### 5. TestMain and Ambient State

`t.TempDir()` contains a test's own writes. It does not contain a write the code under test aims at
`$HOME`, an XDG state directory, or a host CLI on `PATH`.

If the package reaches any of those, arm the isolation in `TestMain`:

```go
var isolation *testsupport.Isolation

func TestMain(m *testing.M) {
    testsupport.IsolateGit()
    isolation = testsupport.IsolateAmbientState()
    os.Exit(isolation.Finish(m.Run()))
}
```

Seven packages do this today. The full procedure, the incident behind it, and how to guard the guard
are in `isolating-the-machine-from-the-test-suite.guide`.

### 6. HTTP Testing

Use `httptest.NewServer` with inline handlers:

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/api/v1/status" {
        t.Errorf("unexpected path: %s", r.URL.Path)
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{"ready": true})
}))
defer srv.Close()

client := api.NewClient(srv.URL)
```

**Test scenarios to cover:** healthy response, server error (500), connection refused (close server before request), malformed JSON response, context cancellation.

For URL rewriting in tests (e.g., redirecting GitHub API calls to test server), use a custom `RoundTripper`:

```go
type testRewriteTransport struct{ target string }
func (t *testRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    req = req.Clone(req.Context())
    parsed, _ := url.Parse(t.target)
    req.URL.Scheme = parsed.Scheme
    req.URL.Host = parsed.Host
    return http.DefaultTransport.RoundTrip(req)
}
```

### 7. Mocking

Define a **minimal interface in the source file**, implement a mock struct in the test file:

```go
// Source file (cmd/sync.go):
type syncClient interface {
    Sync(ctx context.Context, payload *archsync.SyncPayload) (*api.SyncResponse, bool, error)
}

// Test file (cmd/sync_test.go):
type mockSyncClient struct {
    called         bool
    payload        *archsync.SyncPayload
    resp           *api.SyncResponse
    projectCreated bool
    err            error
}
func (m *mockSyncClient) Sync(_ context.Context, payload *archsync.SyncPayload) (*api.SyncResponse, bool, error) {
    m.called = true
    m.payload = payload
    return m.resp, m.projectCreated, m.err
}
```

**Conventions:**

- Mock fields: `called bool` for invocation tracking, captured args for verification, predefined return values.
- Factory helpers for common test preconditions: `testPreconditions(baseDir)`.
- Verify mock state after execution: `if !mock.called { t.Fatal("...") }`.

For a package-level variable that exists so a test can replace behavior, see
`registry-agreement-and-test-seams.guide` — a seam carries a comment saying production never
reassigns it.

### 8. MCP Tool Testing

A handler takes a `RootProvider`, not a base directory. Unit tests use `StaticRoot`. See
`the-shape-of-an-mcp-tool-file.rule`.

Use a `callTool` helper that wraps MCP request construction:

```go
func callTool(handler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error), args map[string]any) (*mcp.CallToolResult, error) {
    req := mcp.CallToolRequest{}
    req.Params.Arguments = args
    return handler(context.Background(), req)
}
```

**Assert error results:**

```go
result, _ := callTool(HandleGetDocument(StaticRoot(base)), map[string]any{"path": ".archcore/../etc/passwd"})
if !result.IsError {
    t.Error("expected error for path traversal")
}
```

**Parse JSON results:**

```go
var doc LocalDocument
if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &doc); err != nil {
    t.Fatal(err)
}
if doc.Title != "Expected Title" {
    t.Errorf("title = %q, want %q", doc.Title, "Expected Title")
}
```

A test that must cross tool boundaries — one handler writes, another reads — belongs in
`internal/mcp/integration/` instead. See `in-process-mcp-integration-tests.adr`.

### 9. Test Data

Use **inline strings** — not fixture files:

```go
const validFrontmatter = "---\ntitle: Test Doc\nstatus: draft\n---\n\nBody.\n"
```

For complex fixtures (tar.gz archives), use builder helpers:

```go
func buildTestArchive(t *testing.T, files map[string][]byte) []byte {
    var buf bytes.Buffer
    gw := gzip.NewWriter(&buf)
    tw := tar.NewWriter(gw)
    for name, content := range files {
        tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))})
        tw.Write(content)
    }
    tw.Close()
    gw.Close()
    return buf.Bytes()
}
```

### 10. Benchmarks

A benchmark over a document corpus builds it with `testsupport.BuildCorpus(tb, dir, n)`.

Do not hand-roll a corpus of one document type. `corpusTypes` in @internal/testsupport/corpus.go spans
both halves of the code-alignment allowlist — the five ranked types and the seven ignored ones — so a
benchmark measures the filter rather than a corpus that happens to be all one type. Every generated
body mentions `src/api/`, so code-alignment correlation has something to match.

Four benchmarks exist: `BenchmarkRealisticReadTools` and `BenchmarkReadToolsScaling` in
`internal/mcp/tools`, `BenchmarkCodeAlignment` and `BenchmarkPrecision` in `internal/advisory`.

Report allocations. A read-path regression in this repository has historically shown up as allocation
count before it showed up as wall clock:

```go
b.ReportAllocs()
b.ResetTimer()
```

### 11. Test Naming

**Function names:** `Test<FunctionName>` or `Test<Feature>_<Scenario>`:

```
TestCleanVersion                         — simple function test
TestRunSync_DryRun_DoesNotUpdateManifest — feature + scenario + expected behavior
TestDoctor_NotInitialized                — feature + error condition
TestHandleCreateDocument_DuplicatePrevented — tool + edge case
```

**Subtest names** in table-driven tests: short, descriptive phrases:

```
"simple", "with v prefix", "prerelease"
"none", "cloud", "on-prem"
"healthy", "not ready", "server error"
```

**Property-pinning files.** A test file that pins normative properties rather than coverage is named
`<subject>_spec_test.go` and carries a doc comment listing them —
`strict-go-naming-conventions.rule` §I. Seven such files exist; use one when the thing under test is a
contract another document states, not an implementation detail.

## Verification

```bash
go test ./...                    # Run all tests
go test -v ./cmd/ -run TestXxx   # Run specific test with verbose output
go test -race ./...              # Check for race conditions
go test -count=1 ./...           # Disable test caching
go test -bench=. -benchmem ./internal/mcp/tools/   # Run benchmarks with allocations
```

A test that passes is not yet a test that guards. Break the behavior deliberately and confirm the test
fails, then restore. A test that stays green against an injected fault pins nothing.

## Common Issues

### Test fails only in CI

- Check for `os.Getuid() == 0` skip guards — CI may run as root.
- Ensure no hardcoded paths — always use `t.TempDir()`.
- Check for `t.Parallel()` races on shared state.

### The suite is green and the machine changed

The package is missing `TestMain`, or `Isolation.Finish` is not on the exit path. See step 5.

### Flaky HTTP tests

- Always `defer srv.Close()` — leaked servers cause port exhaustion.
- Use `context.WithCancel` + immediate cancel for testing cancellation, not timeouts.

### Filesystem test pollution

- Never `os.Chdir()` without saving and restoring the original directory; prefer `t.Chdir`.
- When the working directory must move, do NOT use `t.Parallel()` — it is process-global.

### A test asserts a platform property and fails elsewhere

`filepath.ToSlash` and friends are no-ops off Windows. A property that depends on the host OS belongs
in a build-tagged file — see `platform-splits-are-files.rule` §7.

### MCP tool test assertion failures

- Remember to check `result.IsError` before accessing `result.Content`.
- MCP text content requires type assertion: `result.Content[0].(mcp.TextContent).Text`.
