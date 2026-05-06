package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runWhere mutates the process working directory; tests that call it must
// NOT use t.Parallel() since cwd is process-global.
func runWhere(t *testing.T, dir string, args ...string) (stdout string, err error) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Errorf("restore cwd: %v", err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	root := NewRootCmd("0.5.0")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(append([]string{"where"}, args...))
	root.SetContext(context.Background())
	err = root.Execute()
	return buf.String(), err
}

func TestWhere_HumanOutputOK(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := runWhere(t, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\nout: %s", err, out)
	}
	if !strings.Contains(out, "base dir") {
		t.Errorf("missing 'base dir' in output: %s", out)
	}
	if !strings.Contains(out, "status   : OK") {
		t.Errorf("missing 'status   : OK' in output: %s", out)
	}
	if !strings.Contains(out, ".git(found)") {
		t.Errorf("missing '.git(found)' marker line: %s", out)
	}
}

func TestWhere_HumanOutputNoMarkers(t *testing.T) {
	dir := t.TempDir()
	out, err := runWhere(t, dir)
	if err == nil {
		t.Fatal("expected error for no-markers, got nil")
	}
	var ec interface{ ExitCode() int }
	if !errors.As(err, &ec) {
		t.Fatalf("expected exitCodeError, got %T", err)
	}
	if ec.ExitCode() != whereExitNotResolved {
		t.Errorf("ExitCode() = %d, want %d", ec.ExitCode(), whereExitNotResolved)
	}
	if !strings.Contains(out, "ERR_NO_PROJECT") {
		t.Errorf("missing ERR_NO_PROJECT in output: %s", out)
	}
}

func TestWhere_JSONOutputOK(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := runWhere(t, dir, "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v\nout: %s", err, out)
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("decode JSON: %v\nout: %s", err, out)
	}
	if resp["ok"] != true {
		t.Errorf("ok = %v, want true", resp["ok"])
	}
	if resp["base_dir"] == nil {
		t.Errorf("base_dir missing")
	}
	if resp["cli_version"] != "v0.5.0" {
		t.Errorf("cli_version = %v, want v0.5.0", resp["cli_version"])
	}
}

func TestWhere_JSONOutputError(t *testing.T) {
	dir := t.TempDir()
	out, err := runWhere(t, dir, "--json")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("decode JSON: %v\nout: %s", err, out)
	}
	if resp["ok"] != false {
		t.Errorf("ok = %v, want false", resp["ok"])
	}
	probs, _ := resp["problems"].([]any)
	if len(probs) == 0 {
		t.Fatal("expected at least one problem")
	}
}

func TestWhere_BaseFlagOverride(t *testing.T) {
	cwdDir := t.TempDir()
	projDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Errorf("restore cwd: %v", err)
		}
	})
	if err := os.Chdir(cwdDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	root := NewRootCmd("0.5.0")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"--base-dir", projDir, "where"})
	root.SetContext(context.Background())
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nout: %s", err, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "source   : flag") {
		t.Errorf("expected source=flag, got: %s", out)
	}
}

func TestWhereExitFromErr_NotProject(t *testing.T) {
	// .git in another dir is irrelevant — base-dir to an empty path triggers
	// no-markers (exit 1, guards-failed) rather than ERR_NO_PROJECT (exit 2).
	empty := t.TempDir()
	root := NewRootCmd("0.5.0")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"--base-dir", empty, "where"})
	root.SetContext(context.Background())
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	var ec interface{ ExitCode() int }
	if !errors.As(err, &ec) {
		t.Fatalf("expected exitCodeError, got %T", err)
	}
	if ec.ExitCode() != 1 {
		t.Errorf("ExitCode() = %d, want 1 (guards failed, not exit 2)", ec.ExitCode())
	}
}
