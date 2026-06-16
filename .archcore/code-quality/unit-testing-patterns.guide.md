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

- Go 1.22+ (for `slices`, `t.Context()`)
- Familiarity with Go's `testing` package
- Read `go-code-quality.rule.md` for general conventions

## Steps

### 1. Test File Organization

Place tests in the **same package** as the source — not in a `_test` suffix package. This allows testing unexported functions directly.

```
cmd/sync.go       → cmd/sync_test.go       (package cmd)
internal/config/  → internal/config/config_test.go (package config)
internal/mcp/tools/ → internal/mcp/tools/create_document_test.go (package tools)
```

Use `t.Parallel()` at the start of tests that have no global state mutations (no `os.Chdir`, no shared package-level vars).

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
- Loop variable: `tt` (not `tc` or `test`).
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

### 5. HTTP Testing

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

### 6. Mocking

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

### 7. MCP Tool Testing

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
result, _ := callTool(HandleGetDocument(base), map[string]any{"path": ".archcore/../etc/passwd"})
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

### 8. Test Data

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

### 9. Stdout Capture

For commands that print to stdout:

```go
r, w, _ := os.Pipe()
oldStdout := os.Stdout
os.Stdout = w
defer func() { os.Stdout = oldStdout }()

// Execute command here

w.Close()
var out bytes.Buffer
out.ReadFrom(r)
// Assert on out.String()
```

### 10. Test Naming

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

## Verification

After writing tests, verify with:

```bash
go test ./...                    # Run all tests
go test -v ./cmd/ -run TestXxx   # Run specific test with verbose output
go test -race ./...              # Check for race conditions
go test -count=1 ./...           # Disable test caching
```

## Common Issues

### Test fails only in CI

- Check for `os.Getuid() == 0` skip guards — CI may run as root.
- Ensure no hardcoded paths — always use `t.TempDir()`.
- Check for `t.Parallel()` races on shared state.

### Flaky HTTP tests

- Always `defer srv.Close()` — leaked servers cause port exhaustion.
- Use `context.WithCancel` + immediate cancel for testing cancellation, not timeouts.

### Filesystem test pollution

- Never `os.Chdir()` without saving and restoring the original directory.
- When `os.Chdir` is necessary, do NOT use `t.Parallel()` — working directory is process-global.

### MCP tool test assertion failures

- Remember to check `result.IsError` before accessing `result.Content`.
- MCP text content requires type assertion: `result.Content[0].(mcp.TextContent).Text`.
