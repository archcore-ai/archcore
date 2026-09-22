package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	mcpserver "archcore-cli/internal/mcp"
)

// Patch-target extraction.
//
// A patch tool names its files only inside the patch body, so the write guard
// has nothing to check unless this reads them out. The guard-level consequences
// are covered against a host in hooks_opencode_test.go; what follows pins the
// extraction itself — which keys carry a patch, which lines name a path, and
// where the scan stops.

func payloadFrom(t *testing.T, raw string) *hookPayload {
	t.Helper()
	return decodeHookPayload(strings.NewReader(raw))
}

// patchIn builds a payload carrying patch text under one key path.
func patchIn(t *testing.T, keyPath []string, text string) *hookPayload {
	t.Helper()
	var doc any = text
	for i := len(keyPath) - 1; i >= 0; i-- {
		doc = map[string]any{keyPath[i]: doc}
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return payloadFrom(t, string(raw))
}

// TestPatchPaths_ReadsEveryCarrierKey: hosts disagree on what the patch argument
// is called, and a key missed here is a guard that silently never runs. The
// "input" and "patch" spellings are the unverified ones — Codex CLI and Copilot
// both match apply_patch in their installed matchers, so a path-less patch call
// reaches this guard on hosts the CLI wires, not only on OpenCode.
func TestPatchPaths_ReadsEveryCarrierKey(t *testing.T) {
	t.Parallel()
	const patch = "*** Begin Patch\n*** Update File: .archcore/my.rule.md\n*** End Patch"
	keyPaths := [][]string{
		{"tool_input", "patchText"},
		{"toolArgs", "patchText"},
		{"tool_args", "patchText"},
		{"patchText"},
		{"tool_input", "input"},
		{"toolArgs", "input"},
		{"tool_args", "input"},
		{"tool_input", "patch"},
		{"toolArgs", "patch"},
		{"tool_args", "patch"},
	}
	for _, keyPath := range keyPaths {
		t.Run(strings.Join(keyPath, "."), func(t *testing.T) {
			t.Parallel()
			got := patchIn(t, keyPath, patch).patchPaths()
			if len(got) != 1 || got[0] != ".archcore/my.rule.md" {
				t.Errorf("patchPaths() = %v, want [.archcore/my.rule.md]", got)
			}
		})
	}
}

// TestPatchPaths_ReadsEveryDirective: the four lines that name a path, including
// the rename destination. "*** Move to:" is the one that names a path no other
// directive mentions — a rename is an update of the source plus this line, so
// reading only the first three sees where the bytes came from and misses where
// they land.
func TestPatchPaths_ReadsEveryDirective(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "add",
			text: "*** Begin Patch\n*** Add File: a.md\n+body\n*** End Patch",
			want: []string{"a.md"},
		},
		{
			name: "update",
			text: "*** Begin Patch\n*** Update File: b.md\n@@\n-x\n+y\n*** End Patch",
			want: []string{"b.md"},
		},
		{
			name: "delete",
			text: "*** Begin Patch\n*** Delete File: c.md\n*** End Patch",
			want: []string{"c.md"},
		},
		{
			name: "rename names both ends",
			text: "*** Begin Patch\n*** Update File: src/old.md\n*** Move to: .archcore/new.rule.md\n*** End Patch",
			want: []string{"src/old.md", ".archcore/new.rule.md"},
		},
		{
			name: "several files in one patch",
			text: "*** Begin Patch\n*** Update File: a.go\n*** Add File: b.go\n*** Delete File: c.go\n*** End Patch",
			want: []string{"a.go", "b.go", "c.go"},
		},
		{
			name: "a directive naming nothing is skipped",
			text: "*** Begin Patch\n*** Update File:   \n*** End Patch",
			want: nil,
		},
		// Case and inner spacing are accepted because the parser that will apply
		// the patch is not this one. Requiring the exact spelling would make the
		// guard's coverage a bet on how forgiving that parser is.
		{
			name: "a lowercase directive still names its file",
			text: "*** Begin Patch\n*** update file: .archcore/my.rule.md\n*** End Patch",
			want: []string{".archcore/my.rule.md"},
		},
		{
			name: "spacing inside the directive still names its file",
			text: "*** Begin Patch\n***   Update  File  :  .archcore/my.rule.md\n*** End Patch",
			want: []string{".archcore/my.rule.md"},
		},
		{
			name: "a doubled space after the stars still names its file",
			text: "*** Begin Patch\n***  Update File: .archcore/my.rule.md\n*** End Patch",
			want: []string{".archcore/my.rule.md"},
		},
		{
			name: "a space before the colon still names its file",
			text: "*** Begin Patch\n*** Move to : .archcore/my.rule.md\n*** End Patch",
			want: []string{".archcore/my.rule.md"},
		},
		{
			name: "an indented directive still names its file",
			text: "*** Begin Patch\n  *** Update File: .archcore/my.rule.md\n*** End Patch",
			want: []string{".archcore/my.rule.md"},
		},
		{
			name: "a starred line that is not a directive names nothing",
			text: "*** Begin Patch\n*** Rewrite File: .archcore/my.rule.md\n*** End Patch",
			want: nil,
		},
		{
			name: "envelope markers name no path",
			text: "*** Begin Patch\n*** End Patch",
			want: nil,
		},
		{
			name: "prose is not a patch",
			text: "not a patch at all",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := patchIn(t, []string{"tool_input", "patchText"}, tt.text).patchPaths()
			if strings.Join(got, "|") != strings.Join(tt.want, "|") {
				t.Errorf("patchPaths() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestPatchPaths_AbsentPatchYieldsNothing: no patch argument is the ordinary
// case — every non-patch tool call takes this path — and it must not invent a
// target.
func TestPatchPaths_AbsentPatchYieldsNothing(t *testing.T) {
	t.Parallel()
	p := payloadFrom(t, `{"tool_name":"Write","tool_input":{"file_path":"a.go","content":"x"}}`)

	if got := p.patchPaths(); got != nil {
		t.Errorf("patchPaths() = %v, want nil", got)
	}
}

// TestPatchPaths_LineLimitIsTheDocumentedOne pins the bound's value.
//
// The two tests below place their target relative to maxPatchLines, so they hold
// the loop's arithmetic and nothing else: change the constant and they follow it
// silently. The value itself is published — the hook runtime contract and the
// CLI hooks reference both state it, and a target past that line is documented as
// unguarded — so it needs an assertion that does not move when the code does.
func TestPatchPaths_LineLimitIsTheDocumentedOne(t *testing.T) {
	t.Parallel()
	if maxPatchLines != 20000 {
		t.Errorf("maxPatchLines = %d, want 20000 — the documented bound; "+
			"change the hook runtime contract and the CLI hooks reference with it", maxPatchLines)
	}
}

// TestPatchPaths_StopsAtTheLineLimit: the payload cap bounds the bytes, this
// bounds the lines, and the pre-write guard blocks the user while it runs. The
// bound is real rather than advisory — it is asserted here in the direction that
// documents its cost, since a target past the limit is a target not guarded.
func TestPatchPaths_StopsAtTheLineLimit(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	b.WriteString("*** Begin Patch\n")
	for range maxPatchLines {
		b.WriteString("+filler\n")
	}
	b.WriteString("*** Update File: .archcore/past-the-limit.rule.md\n")

	got := patchIn(t, []string{"tool_input", "patchText"}, b.String()).patchPaths()

	if len(got) != 0 {
		t.Errorf("patchPaths() = %v, want nothing past line %d", got, maxPatchLines)
	}
}

// TestPatchPaths_ScansUpToTheLimit: the other half of the bound. A target on the
// last line the scan reaches is still found, so the limit is what stops the scan
// and not something shorter.
func TestPatchPaths_ScansUpToTheLimit(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	for range maxPatchLines - 1 {
		b.WriteString("+filler\n")
	}
	b.WriteString("*** Update File: .archcore/at-the-limit.rule.md\n")

	got := patchIn(t, []string{"tool_input", "patchText"}, b.String()).patchPaths()

	if len(got) != 1 || got[0] != ".archcore/at-the-limit.rule.md" {
		t.Errorf("patchPaths() = %v, want the target on line %d", got, maxPatchLines)
	}
}

// TestArchcoreMCPTools_MatchesTheServer holds the fold's tool set against the
// server's own registration.
//
// The three loose prefixes fold only onto a name in this set, because a single
// separator cannot mark where a server name ends. A tool the server gains and
// this map does not is therefore not folded on Gemini CLI, Copilot, or OpenCode,
// and the write guard denies it as a direct edit — which is why the two have to
// be checked against each other rather than kept in step by hand.
//
// The names are read back off a built server rather than re-listed here. An
// earlier version of this test listed the constructors instead, and a second
// hand-written list is not the registration — it is a copy of it, and one copy
// agrees with another no matter what AddTool says. Verified by adding a tool to
// newServerWithConfig: the constructor form passed, this form fails.
//
// Host wiring is supplied because install_host_config is registered only on that
// option, and the guard sees the tools the running server exposes — `archcore
// mcp` always passes it.
func TestArchcoreMCPTools_MatchesTheServer(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	srv := mcpserver.NewServer(base, "test", mcpserver.WithHostWiring(hostWiringExecutor))

	registered := srv.ListTools()
	// ListTools returns nil for a server holding nothing, which every assertion
	// below would read as agreement.
	if len(registered) == 0 {
		t.Fatal("the server registered no tools, so this test proves nothing")
	}

	for name := range registered {
		if !archcoreMCPTools[name] {
			t.Errorf("the server registers %q and archcoreMCPTools does not carry it: "+
				"the hook guard will deny that tool's writes on every host that flattens tool names", name)
		}
	}
	for name := range archcoreMCPTools {
		if _, ok := registered[name]; !ok {
			t.Errorf("archcoreMCPTools carries %q and the server does not register it", name)
		}
	}
}

// TestMutatingToolsAreArchcoreTools: the mutating set is a subset by
// construction, and isMutatingArchcoreTool reads it only after the fold. A name
// in one and not the other would be a post-write check gated on a spelling that
// can never arrive.
func TestMutatingToolsAreArchcoreTools(t *testing.T) {
	t.Parallel()
	for name := range mutatingMCPTools {
		if !archcoreMCPTools[name] {
			t.Errorf("mutatingMCPTools carries %q, which archcoreMCPTools does not fold onto", name)
		}
		flattened := fmt.Sprintf("archcore_%s", name)
		if !isMutatingArchcoreTool(flattened) {
			t.Errorf("isMutatingArchcoreTool(%q) = false: the post-write check never runs on a flattening host", flattened)
		}
	}
}
