package cmd

import (
	"context"
	"io"
	"strings"
	"testing"
)

// Every command that takes no positional argument must reject a stray one at
// validation time, so a mistyped `<cmd> foo` fails loudly instead of silently
// ignoring foo or falling through to a default branch — the footgun behind
// `instructions remove claude-code` wiping every target. For the sub-commands
// this is the explicit cobra.NoArgs we added; for the bare `bogus` case it is
// cobra's built-in root-command handling (legacyArgs) — both surface the same
// "unknown command" contract, which is what this guards.
//
// Validation runs BEFORE RunE, so exercising these through the real command
// tree is side-effect-free: none of the handlers (server start, network,
// filesystem writes) run — the args check short-circuits first.
func TestCommands_RejectStrayPositionalArgs(t *testing.T) {
	t.Parallel()
	cases := [][]string{
		{"bogus"},                        // root: unknown top-level word (cobra legacyArgs)
		{"init", "x"},                    // leaf, flags only
		{"status", "x"},                  // leaf
		{"doctor", "x"},                  // leaf
		{"sync", "x"},                    // leaf (hidden)
		{"update", "x"},                  // leaf
		{"mcp", "x"},                     // parent-with-RunE: must not start the server
		{"mcp", "install", "x"},          // --agent only
		{"hooks", "install", "x"},        // --agent only
		{"instructions", "install", "x"}, // --agent only
		{"instructions", "remove", "x"},  // the original footgun
	}
	for _, args := range cases {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			t.Parallel()
			root := NewRootCmd("0.0.0-test")
			root.SetArgs(args)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)

			err := root.Execute()
			if err == nil {
				t.Fatalf("%v: expected an error for the stray positional arg, got nil", args)
			}
			if !strings.Contains(err.Error(), "unknown command") {
				t.Errorf("%v: error %q should reject the positional arg as an unknown command", args, err.Error())
			}
		})
	}
}

// The hook event leaves are the other deliberate exception, and the opposite
// one: they must NOT reject a stray argument.
//
// "Fail loudly" assumes a human reads the error. On a hook leaf the reader is
// the host, and every one of them treats a non-zero exit as a verdict —
// Copilot as a deny whose reason is discarded. Rejecting the argument at
// validation time also skips safeHandle and emitDecision, so the fail-open
// paths never run. The contract is owned by TestHooksCLI_StrayArgument* in
// hook_cli_e2e_test.go: tolerate the argument, still run the guard.
func TestHookLeaves_AcceptStrayPositionalArgs(t *testing.T) {
	t.Parallel()
	for _, event := range []string{"session-start", "pre-tool-use", "post-tool-use"} {
		t.Run(event, func(t *testing.T) {
			t.Parallel()
			leaf := newHookEventCmd(hookDialects[0], eventSessionStart, event,
				func(context.Context, hookRequest) hookDecision { return allowHook() })
			if leaf.Args == nil {
				t.Fatal("hook leaf must set Args explicitly, not inherit cobra's default")
			}
			if err := leaf.Args(leaf, []string{"x"}); err != nil {
				t.Errorf("a stray argument must not fail validation: %v", err)
			}
		})
	}
}

// config is the deliberate exception: it dispatches on positional args
// (`config get <key>`, `config set <key> <value>`), so it must NOT restrict
// them. Guard that the exception stays intentional.
//
// Asserted by running the validator rather than by requiring Args to be nil.
// Both cobra.ArbitraryArgs and no declaration at all accept everything, and the
// convention is to state the policy: a nil Args is indistinguishable from a
// command whose author never considered the question.
func TestConfigCmd_AcceptsPositionalArgs(t *testing.T) {
	t.Parallel()
	cmd := newConfigCmd()
	if cmd.Args == nil {
		t.Fatal("config must declare its Args policy explicitly, even though it accepts everything")
	}
	for _, args := range [][]string{
		{"get", "sync"},
		{"set", "language", "ru"},
		{"set", "language", "en", "US"}, // runConfig joins the tail; cobra must not refuse first
	} {
		if err := cmd.Args(cmd, args); err != nil {
			t.Errorf("config rejected %v at validation time: %v", args, err)
		}
	}
}
