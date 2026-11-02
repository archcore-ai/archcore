package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"archcore-cli/internal/advisory"

	"github.com/spf13/cobra"
)

// Hook command tree. Every host gets the same three hidden leaves — a protocol
// surface, not a user surface, so `archcore --help` still lists nine commands.

// hookRequest is everything a handler needs to decide. Carrying the dialect and
// the event means a handler can skip work the host would discard, instead of
// doing it and having the command layer throw the result away.
type hookRequest struct {
	baseDir string
	dialect hostDialect
	event   hookEvent
	payload *hookPayload
}

// hookHandler decides what a hook event should do. It never writes output and
// never exits — the command layer owns both, so one place enforces the per-host
// protocol and the safety rules.
//
// ctx is the command's, so a host that gives up on a slow hook cancels the git
// calls underneath it rather than leaving them to their own deadlines.
type hookHandler func(ctx context.Context, r hookRequest) hookDecision

// hookExit ends the process with the code the dialect chose. A variable so a
// test can observe the code instead of having the test binary exit; production
// never reassigns it.
var hookExit = os.Exit

// newHookHostCmd builds one host's command group.
func newHookHostCmd(d hostDialect, version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:    string(d.id),
		Short:  fmt.Sprintf("Handle %s hook events", d.id),
		Hidden: true,
		// Arbitrary args plus a silent RunE closes a trap: a group with neither
		// prints cobra's help to STDOUT and exits 0. On a hook that stdout is the
		// protocol channel, so an unknown or renamed event would deliver several
		// hundred bytes of usage text straight into the model's context.
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd.AddCommand(
		newHookEventCmd(d, eventSessionStart, "session-start", sessionStartHandler(version)),
		newHookEventCmd(d, eventPreToolUse, "pre-tool-use", preToolUseHandler),
		newHookEventCmd(d, eventPostToolUse, "post-tool-use", postToolUseHandler),
	)
	return cmd
}

// newHookEventCmd builds one hidden event leaf.
func newHookEventCmd(d hostDialect, event hookEvent, use string, handler hookHandler) *cobra.Command {
	return &cobra.Command{
		Use:    use,
		Short:  fmt.Sprintf("Handle the %s %s hook event", d.id, event),
		Hidden: true,
		// No positional args are read — the payload arrives on stdin — but an
		// unexpected one must not fail: cobra rejects it before RunE, skipping
		// every fail-open path and exiting non-zero. Copilot reads any non-zero
		// exit as a deny and discards the reason with it.
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			p := decodeHookPayload(os.Stdin)

			baseDir, err := resolveBaseDir(p.cwd())
			if err != nil {
				// Without a project root there is nothing to check. Allow.
				return nil
			}

			dec := safeHandle(cmd.Context(), handler, hookRequest{
				baseDir: baseDir,
				dialect: d,
				event:   event,
				payload: p,
			})
			if event == eventPostToolUse {
				// A post-event cannot block anything — the tool has already run —
				// so a deny here would only be a confusing non-zero exit.
				dec.deny, dec.reason = false, ""
			}
			if code := emitDecision(d, event, dec); code != 0 {
				hookExit(code)
			}
			return nil
		},
	}
}

// safeHandle runs a guard and converts any panic into an allow.
//
// A panic would otherwise exit 2, which several hosts read as an explicit deny —
// an internal defect would silently start blocking the user's edits. On Copilot
// it is worse: every non-zero exit denies, and the reason is discarded, so the
// user would be blocked with no explanation at all.
func safeHandle(ctx context.Context, handler hookHandler, r hookRequest) (dec hookDecision) {
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Fprintf(os.Stderr, "archcore hook: recovered from internal error: %v\n", rec)
			dec = allowHook()
		}
	}()
	return handler(ctx, r)
}

// resolveBaseDir returns the project root from the hook payload, falling back to
// the process working directory.
func resolveBaseDir(payloadCWD string) (string, error) {
	if payloadCWD != "" {
		return payloadCWD, nil
	}
	return os.Getwd()
}

// deriveDedupKey derives the SessionStart dedup key.
//
// The event source (startup/resume/clear/compact on Claude Code) is folded in so
// a legitimate re-injection after a compact — where the earlier context was
// summarized away — is not suppressed by the startup stamp.
//
// The host id is folded in too. Copilot reads .claude/settings.json as well as
// its own config, so one session start can run both leaves with identical stdin;
// without the host in the key they race for one stamp and the loser emits
// nothing. Each speaks a different dialect, so the wrong winner leaves the
// session with no context at all. Empty when the host sent no id: dedup then
// fails open and always emits.
func deriveDedupKey(r hookRequest) string {
	id := r.payload.sessionID()
	if id == "" {
		return ""
	}
	return id + "\x00" + r.payload.source() + "\x00" + string(r.dialect.id)
}

// sessionStartHandler emits the session recap, deduplicated per session.
func sessionStartHandler(version string) hookHandler {
	return func(ctx context.Context, r hookRequest) hookDecision {
		context, banner, emitted := handleSessionStartDeduped(
			ctx, r.baseDir, version, deriveDedupKey(r), defaultSessionStampDir())
		if !emitted {
			return allowHook()
		}
		return adviseSession(context, banner)
	}
}

// preToolUseHandler is the only guard that can block. It runs the write guard
// first and returns its deny unchanged, so a failure in the advisory half can
// never alter the verdict on someone's edit.
func preToolUseHandler(ctx context.Context, r hookRequest) hookDecision {
	filePath := r.payload.filePath()
	patchPaths := r.payload.patchPaths()
	targets := append([]string{filePath}, patchPaths...)

	// One guard per project for the whole call. Every target of one project asks
	// the same questions about it, and a patch can carry thousands of them.
	roots := newHookRoots(r.baseDir)

	// A patch call names its targets only inside the patch body, and one call can
	// carry several. Any guarded target blocks the whole call: a patch applies as
	// a unit, so allowing it because only one of its files is protected would
	// write that file anyway.
	for _, target := range targets {
		root := roots.rootOf(target)
		if dec := roots.guard(root).decide(roots.guardPath(target, root)); dec.deny {
			return withProjectHint(dec, root, r.baseDir)
		}
	}

	// Copilot's preToolUse carries only a permission decision, so context sent
	// there is discarded. Skipping it here rather than dropping it downstream is
	// the difference between not doing the corpus scan and doing it for nothing.
	if !r.dialect.preToolContext {
		return allowHook()
	}
	var notes []string
	// A patch is how Codex edits every file, so its targets name the projects a
	// call touches as a file path does (hook-runtime.spec §16).
	noted := map[string]bool{r.baseDir: true}
	for _, target := range targets {
		if target == "" {
			continue
		}
		if other := roots.rootOf(target); !noted[other] {
			noted[other] = true
			if note := projectTouchAdvisory(r, other); note != "" {
				notes = append(notes, note)
			}
		}
	}
	// Codex edits every file through a patch, so a call without file_path aligns
	// on the patch's first target (hook-runtime.spec §18): one file, one scan.
	alignTarget := filePath
	if alignTarget == "" && len(patchPaths) > 0 {
		alignTarget = patchPaths[0]
	}
	root := roots.rootOf(alignTarget)
	if note := advisory.CodeAlignment(root, alignTarget); note != "" {
		notes = append(notes, note)
	}
	if len(notes) == 0 {
		return allowHook()
	}
	return adviseHook(strings.Join(notes, "\n\n"))
}
