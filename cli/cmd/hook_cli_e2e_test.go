package cmd

import (
	"strings"
	"testing"
)

// End-to-end tests for the hook leaves: stdin in, exit code and stdout out.
//
// Everything below drives the real cobra command rather than calling a handler,
// because the defects these pin live in the command layer — argument parsing,
// the exit code, and which writer owns stdout.

// hookRun is what one hook invocation produced.
type hookRun struct {
	exitCode int
	stdout   string
	stderr   string
	err      error
}

// runHookCLI invokes `archcore hooks <host> <event> [args...]` in baseDir with
// stdin fed from a string.
//
// t.Chdir is load-bearing: resolveBaseDir falls back to os.Getwd() when the
// payload carries no cwd, so without it an empty-stdin row would scan whatever
// project the test binary happens to run in. It also rules out t.Parallel().
func runHookCLI(t *testing.T, baseDir, host, event string, extraArgs []string, stdin string) hookRun {
	t.Helper()
	t.Chdir(baseDir)
	withStdin(t, stdin)
	code := withHookExit(t)

	args := append([]string{"hooks", host, event}, extraArgs...)
	root := NewRootCmd("test")
	root.SetArgs(args)

	var err error
	out, errOut := captureOutput(t, func() { err = root.Execute() })
	return hookRun{exitCode: *code, stdout: out, stderr: errOut, err: err}
}

// denyPayload targets a document write, which the guard must block.
const denyPayload = `{"tool_name":"Write","tool_input":{"file_path":".archcore/knowledge/a.adr.md"}}`

// TestHooksCLI_StrayArgumentDoesNotFail pins that an unexpected positional
// argument stays on the fail-open path.
//
// cobra.NoArgs on the event leaves rejected it before RunE ran, so the process
// exited 1 without reaching safeHandle or emitDecision. On Copilot every
// non-zero exit is a deny whose reason is discarded, so a malformed invocation
// silently blocked the user's edit.
func TestHooksCLI_StrayArgumentDoesNotFail(t *testing.T) {
	tests := []struct {
		name  string
		host  string
		event string
	}{
		{name: "claude-code pre-tool-use", host: "claude-code", event: "pre-tool-use"},
		{name: "copilot pre-tool-use", host: "copilot", event: "pre-tool-use"},
		{name: "copilot session-start", host: "copilot", event: "session-start"},
		{name: "claude-code post-tool-use", host: "claude-code", event: "post-tool-use"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := setupArchcoreDir(t)
			run := runHookCLI(t, base, tt.host, tt.event, []string{"EXTRA"}, "")

			if run.err != nil {
				t.Errorf("a stray argument returned an error: %v", run.err)
			}
			if run.exitCode != 0 {
				t.Errorf("exit code = %d, want 0", run.exitCode)
			}
			if strings.Contains(run.stderr, "unknown command") {
				t.Errorf("cobra rejected the argument instead of the handler:\n%s", run.stderr)
			}
		})
	}
}

// TestHooksCLI_StrayArgumentStillRunsTheGuard: tolerating the argument must not
// mean ignoring the payload. The guard still has to reach its verdict.
func TestHooksCLI_StrayArgumentStillRunsTheGuard(t *testing.T) {
	base := setupArchcoreDir(t)
	run := runHookCLI(t, base, "copilot", "pre-tool-use", []string{"EXTRA"}, denyPayload)

	if run.exitCode != 0 {
		t.Errorf("exit code = %d, want 0 (Copilot signals a deny in JSON)", run.exitCode)
	}
	if !strings.Contains(run.stdout, `"permissionDecision":"deny"`) {
		t.Errorf("the guard did not run; stdout:\n%s", run.stdout)
	}
}
