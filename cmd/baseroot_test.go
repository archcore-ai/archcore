package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"archcore-cli/internal/projectroot"
)

func makeBaseDirTestCmd(t *testing.T) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "test", SilenceErrors: true, SilenceUsage: true}
	addBaseDirFlag(root)
	return root
}

func mkProject(t *testing.T) string {
	t.Helper()
	p := t.TempDir()
	if err := os.MkdirAll(filepath.Join(p, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestResolveProjectRoot_FlagViaPersistent(t *testing.T) {
	t.Parallel()
	proj := mkProject(t)

	var got *projectroot.Resolution
	root := makeBaseDirTestCmd(t)
	sub := &cobra.Command{
		Use: "sub",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := resolveProjectRoot(cmd, projectroot.ModeRuntime)
			got = r
			return err
		},
	}
	root.AddCommand(sub)
	root.SetArgs([]string{"--base-dir", proj, "sub"})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetContext(context.Background())
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got == nil {
		t.Fatal("nil resolution")
	}
	if got.Source != projectroot.SourceFlag {
		t.Errorf("Source = %q, want %q", got.Source, projectroot.SourceFlag)
	}
}

func TestResolveProjectRoot_StderrFormatOnError(t *testing.T) {
	t.Parallel()
	root := makeBaseDirTestCmd(t)
	sub := &cobra.Command{
		Use: "sub",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := resolveProjectRoot(cmd, projectroot.ModeRuntime)
			return err
		},
	}
	root.AddCommand(sub)
	root.SetArgs([]string{"--base-dir", "/no/such/path", "sub"})
	var errBuf bytes.Buffer
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&errBuf)
	root.SetContext(context.Background())
	err := root.Execute()
	if err == nil {
		t.Fatal("expected resolve error, got nil")
	}
	s := errBuf.String()
	if !strings.Contains(s, "--- archcore error ---") {
		t.Errorf("missing error block in stderr:\n%s", s)
	}
	if !strings.Contains(s, projectroot.CodePathInvalid) {
		t.Errorf("missing %s in stderr:\n%s", projectroot.CodePathInvalid, s)
	}
}

func TestResolveProjectRoot_ContextMemoized(t *testing.T) {
	t.Parallel()
	proj := mkProject(t)

	var first, second *projectroot.Resolution
	root := makeBaseDirTestCmd(t)
	sub := &cobra.Command{
		Use: "sub",
		RunE: func(cmd *cobra.Command, args []string) error {
			r1, err := resolveProjectRoot(cmd, projectroot.ModeRuntime)
			if err != nil {
				return err
			}
			first = r1
			// Force "drift": flip the flag. Memoization should ignore it.
			if err := cmd.Flags().Set(flagBaseDir, "/elsewhere"); err != nil {
				return err
			}
			r2, err := resolveProjectRoot(cmd, projectroot.ModeRuntime)
			if err != nil {
				return err
			}
			second = r2
			return nil
		},
	}
	root.AddCommand(sub)
	root.SetArgs([]string{"--base-dir", proj, "sub"})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetContext(context.Background())
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if first != second {
		t.Errorf("memoization broken: returned distinct *Resolution pointers")
	}
}

func TestBaseDirFlag_NilSafe(t *testing.T) {
	t.Parallel()
	if got := baseDirFlag(nil); got != "" {
		t.Errorf("baseDirFlag(nil) = %q, want empty", got)
	}
}

func TestResolveProjectRoot_Sentinels(t *testing.T) {
	t.Parallel()
	root := makeBaseDirTestCmd(t)
	sub := &cobra.Command{
		Use: "sub",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := resolveProjectRoot(cmd, projectroot.ModeRuntime)
			if !errors.Is(err, projectroot.ErrBaseDirNotExist) {
				t.Errorf("err is not ErrBaseDirNotExist: %v", err)
			}
			return nil
		},
	}
	root.AddCommand(sub)
	root.SetArgs([]string{"--base-dir", "/no/such/path", "sub"})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetContext(context.Background())
	_ = root.Execute()
}
