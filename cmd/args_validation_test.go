package cmd

import (
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
		{"hooks", "claude-code", "session-start", "x"}, // hidden hook: stdin JSON, no args
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

// config is the deliberate exception: it dispatches on positional args
// (`config get <key>`, `config set <key> <value>`), so it must NOT carry a
// NoArgs restriction. Guard that the exception stays intentional.
func TestConfigCmd_AcceptsPositionalArgs(t *testing.T) {
	t.Parallel()
	if newConfigCmd().Args != nil {
		t.Error("config must not restrict positional args — it dispatches on get/set <key> [value]")
	}
}
