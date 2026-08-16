package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// hookEntry / hookMatcher are the decode shapes cmd tests use to assert on
// Claude-style hook configs; the production types live in internal/wiring.
type hookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type hookMatcher struct {
	Matcher string      `json:"matcher"`
	Hooks   []hookEntry `json:"hooks"`
}

func configPathFor(base, relPath string) string {
	return filepath.Join(base, filepath.FromSlash(relPath))
}

func seedConfig(t *testing.T, base, relPath, content string) string {
	t.Helper()
	path := configPathFor(base, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// writeArchcoreDoc writes content at a slash-separated path under base/.archcore/.
func writeArchcoreDoc(t *testing.T, base, relPath, content string) {
	t.Helper()
	full := filepath.Join(base, ".archcore", filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The helpers below swap a process global, so a test that uses one must not
// call t.Parallel().

// captureOutput runs fn with os.Stdout and os.Stderr redirected and returns what
// each received.
//
// Backed by files rather than pipes: a pipe holds ~64 KB before the writer
// blocks, and reading it only after fn returns deadlocks on anything larger.
// Session-start recap output is close enough to that limit to matter.
func captureOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()
	dir := t.TempDir()
	outFile := mustCreate(t, filepath.Join(dir, "stdout"))
	errFile := mustCreate(t, filepath.Join(dir, "stderr"))

	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outFile, errFile
	defer func() {
		os.Stdout, os.Stderr = oldOut, oldErr
		outFile.Close()
		errFile.Close()
	}()

	fn()

	return mustRead(t, outFile.Name()), mustRead(t, errFile.Name())
}

// captureStdout returns only what fn wrote to stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	out, _ := captureOutput(t, fn)
	return out
}

// withStdin points os.Stdin at input for the rest of the test. A file rather
// than a pipe so a payload larger than the pipe buffer can be fed without a
// writer goroutine.
func withStdin(t *testing.T, input string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stdin")
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open stdin: %v", err)
	}
	old := os.Stdin
	os.Stdin = f
	t.Cleanup(func() {
		os.Stdin = old
		f.Close()
	})
}

// withOpenStdin points os.Stdin at a pipe that stays open, and returns the
// write end so the caller decides when the session ends.
//
// withStdin cannot serve a test that has to observe something while the session
// is still live: a file is at EOF the moment it is opened, so RunStdio's Listen
// returns immediately and the session is over before the assertion runs.
func withOpenStdin(t *testing.T) *os.File {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = old
		_ = w.Close()
		_ = r.Close()
	})
	return w
}

// withHookExit captures the exit code the hook layer selects instead of ending
// the process. The returned pointer holds 0 until hookExit is called.
func withHookExit(t *testing.T) *int {
	t.Helper()
	code := 0
	old := hookExit
	hookExit = func(c int) { code = c }
	t.Cleanup(func() { hookExit = old })
	return &code
}

func mustCreate(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", filepath.Base(path), err)
	}
	return f
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", filepath.Base(path), err)
	}
	return string(data)
}
