package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

// OpenCode bridge contract.
//
// OpenCode is the one host that reaches this binary through a TypeScript bridge
// instead of a declarative hook config, and that route inverts two assumptions
// the other five share:
//
//  1. Nothing on the path parses JSON. The plugin's bin/session-start streams
//     this command's stdout through unchanged and the bridge appends those bytes
//     to the session verbatim, so an envelope is not framing — it is literal
//     text delivered to the model.
//  2. Its MCP tools arrive as "archcore_<tool>", a spelling no other host sends.
//
// The fixtures below are the ones the plugin repository pins for this bridge
// (test/fixtures/stdin/opencode/), so a contract change is caught on both sides
// of it rather than in a live session.

// TestOpenCode_SessionStartIsPlainText: the recap reaches OpenCode as text or
// not at all. A wrapper here does not frame the recap for the host — nothing
// unwraps it — it lands in the model's context as literal JSON with the whole
// recap escaped inside.
func TestOpenCode_SessionStartIsPlainText(t *testing.T) {
	base := setupArchcoreDir(t)

	out := emitSessionStart(t, "opencode", base, "v0.0.0-test")

	if out == "" {
		t.Fatal("session-start wrote nothing to stdout")
	}
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("session-start emitted a JSON document; the bridge appends stdout verbatim:\n%s", out)
	}
	var probe any
	if json.Unmarshal([]byte(out), &probe) == nil {
		t.Errorf("session-start output parses as JSON, so it is an envelope, not text:\n%s", out)
	}
	if !strings.Contains(out, "Archcore") {
		t.Errorf("session-start output does not carry the recap:\n%s", out)
	}
}

// TestOpenCode_SessionStartDropsTheBanner: the plain-text channel is context for
// the model, and the banner is a line for the user. There is no slot for it on
// this host, so it is dropped rather than smuggled into the context — the same
// call Copilot's envelope makes.
func TestOpenCode_SessionStartDropsTheBanner(t *testing.T) {
	d := dialectByID(t, "opencode")

	out := captureStdout(t, func() {
		emitDecision(d, eventSessionStart, adviseSession("recap text", "banner line"))
	})

	if got, want := out, "recap text\n"; got != want {
		t.Errorf("stdout = %q, want %q — the banner has no slot here", got, want)
	}
}

// TestOpenCode_BannerAloneWritesNothing: an empty recap with a banner must not
// send a bare newline. Every byte on this channel becomes model context.
func TestOpenCode_BannerAloneWritesNothing(t *testing.T) {
	d := dialectByID(t, "opencode")

	out := captureStdout(t, func() {
		emitDecision(d, eventSessionStart, adviseSession("", "banner line"))
	})

	if out != "" {
		t.Errorf("stdout = %q, want nothing", out)
	}
}

// TestOpenCode_BridgeFixtures drives the exact payloads the plugin repository
// pins as this bridge's stdin contract.
//
// The MCP row is the load-bearing one, and it fails in the dangerous direction:
// with "archcore_update_document" unrecognized, filePath() falls through to the
// bare "path" key, the write guard reads a sanctioned MCP write as a direct
// edit, and Archcore blocks its own document tools on this host.
func TestOpenCode_BridgeFixtures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		payload  string
		wantDeny bool
	}{
		{
			name:     "write-regular.json passes",
			payload:  `{"tool_name":"write","tool_input":{"file_path":"src/app.py","content":"print('hello')"}}`,
			wantDeny: false,
		},
		{
			name:     "write-archcore.json is blocked",
			payload:  `{"tool_name":"write","tool_input":{"file_path":".archcore/my.rule.md","content":"# Rule"}}`,
			wantDeny: true,
		},
		{
			name:     "mcp-update.json passes under the canonical spelling",
			payload:  `{"tool_name":"mcp__archcore__update_document","tool_input":{"path":".archcore/auth/jwt.adr.md"}}`,
			wantDeny: false,
		},
		{
			name:     "mcp-update.json passes under the opencode spelling",
			payload:  `{"tool_name":"archcore_update_document","tool_input":{"path":".archcore/auth/jwt.adr.md"}}`,
			wantDeny: false,
		},
		// The exemption above is what a foreign server would inherit if the
		// prefix were unbounded. This host joins the server name to the tool
		// name with the separator that also appears inside both, so
		// "archcore_docs_write_file" is a tool of a server called archcore_docs
		// and nothing in the string says otherwise — only the tool set does.
		// Claimed as ours, it stops being read as a direct write and gets to
		// edit a document.
		{
			name:     "a foreign server prefixed with ours is still guarded",
			payload:  `{"tool_name":"archcore_docs_write_file","tool_input":{"path":".archcore/my.rule.md"}}`,
			wantDeny: true,
		},
		{
			name:     "a foreign server prefixed with ours may still write source",
			payload:  `{"tool_name":"archcore_docs_write_file","tool_input":{"path":"src/app.go"}}`,
			wantDeny: false,
		},
		// OpenCode's own write and edit tools name the argument filePath, not
		// file_path (verified against packages/opencode/src/tool/{write,edit}.ts).
		// The bridge is expected to rename it, and this repository cannot enforce
		// that it does — so the guard reads both. Missing this one fails silently:
		// no path found, allow, and an unprotected session is indistinguishable
		// from a clean one.
		{
			name:     "opencode camelCase filePath is still guarded",
			payload:  `{"tool_name":"write","tool_input":{"filePath":".archcore/my.rule.md","content":"# Rule"}}`,
			wantDeny: true,
		},
		{
			name:     "opencode camelCase filePath on a regular file passes",
			payload:  `{"tool_name":"edit","tool_input":{"filePath":"src/app.py","oldString":"a","newString":"b"}}`,
			wantDeny: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupArchcoreDir(t)

			dec := preToolUseHandler(bg(), reqFor(t, "opencode", base, tt.payload))

			if dec.deny != tt.wantDeny {
				t.Errorf("preToolUseHandler.deny = %v, want %v (payload %s)", dec.deny, tt.wantDeny, tt.payload)
			}
		})
	}
}

// TestOpenCode_ApplyPatchIsGuarded closes the hole a path-less write tool opens.
//
// OpenCode's apply_patch carries only patchText, so the ordinary file-path
// lookup finds nothing and the guard allows. That is not a corner case on this
// host: its registry enables apply_patch and disables write and edit for gpt-
// models, so those sessions have exactly one file-mutation tool and it is this
// one. Reading the targets out of the patch body is what keeps the guard from
// being silently absent for a whole class of sessions.
func TestOpenCode_ApplyPatchIsGuarded(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		payload  string
		wantDeny bool
	}{
		{
			name:     "update of an .archcore document is blocked",
			payload:  `{"tool_name":"apply_patch","tool_input":{"patchText":"*** Begin Patch\n*** Update File: .archcore/my.rule.md\n@@\n-a\n+b\n*** End Patch"}}`,
			wantDeny: true,
		},
		{
			name:     "add of an .archcore document is blocked",
			payload:  `{"tool_name":"apply_patch","tool_input":{"patchText":"*** Begin Patch\n*** Add File: .archcore/new.adr.md\n+body\n*** End Patch"}}`,
			wantDeny: true,
		},
		{
			name:     "delete of an .archcore document is blocked",
			payload:  `{"tool_name":"apply_patch","tool_input":{"patchText":"*** Begin Patch\n*** Delete File: .archcore/old.adr.md\n*** End Patch"}}`,
			wantDeny: true,
		},
		{
			// A patch applies as a unit, so one guarded file in the set has to
			// block all of it — allowing would write that file regardless.
			name:     "a mixed patch is blocked by its guarded member",
			payload:  `{"tool_name":"apply_patch","tool_input":{"patchText":"*** Begin Patch\n*** Update File: src/app.go\n*** Update File: .archcore/my.rule.md\n*** End Patch"}}`,
			wantDeny: true,
		},
		{
			// A rename names its destination on a line of its own, and that is
			// the only line that names it. Reading the other three directives
			// sees an ordinary source file and lets the patch write a document.
			name:     "a rename INTO the store is blocked by its destination",
			payload:  `{"tool_name":"apply_patch","tool_input":{"patchText":"*** Begin Patch\n*** Update File: src/notes.md\n*** Move to: .archcore/smuggled.rule.md\n*** End Patch"}}`,
			wantDeny: true,
		},
		{
			// The other direction is a document leaving the store, which the
			// source directive already names.
			name:     "a rename OUT of the store is blocked by its source",
			payload:  `{"tool_name":"apply_patch","tool_input":{"patchText":"*** Begin Patch\n*** Update File: .archcore/my.rule.md\n*** Move to: docs/my.md\n*** End Patch"}}`,
			wantDeny: true,
		},
		{
			name:     "a rename touching neither end of the store passes",
			payload:  `{"tool_name":"apply_patch","tool_input":{"patchText":"*** Begin Patch\n*** Update File: src/a.go\n*** Move to: src/b.go\n*** End Patch"}}`,
			wantDeny: false,
		},
		{
			name:     "a patch touching only source passes",
			payload:  `{"tool_name":"apply_patch","tool_input":{"patchText":"*** Begin Patch\n*** Update File: src/app.go\n@@\n-a\n+b\n*** End Patch"}}`,
			wantDeny: false,
		},
		{
			// settings.json is not a document, and the guard already allows it by
			// path. The patch route must not become stricter than the direct one.
			name:     "a patch touching .archcore/settings.json passes",
			payload:  `{"tool_name":"apply_patch","tool_input":{"patchText":"*** Begin Patch\n*** Update File: .archcore/settings.json\n*** End Patch"}}`,
			wantDeny: false,
		},
		{
			name:     "an empty or unparsable patch allows",
			payload:  `{"tool_name":"apply_patch","tool_input":{"patchText":"not a patch at all"}}`,
			wantDeny: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupArchcoreDir(t)

			dec := preToolUseHandler(bg(), reqFor(t, "opencode", base, tt.payload))

			if dec.deny != tt.wantDeny {
				t.Errorf("preToolUseHandler.deny = %v, want %v (payload %s)", dec.deny, tt.wantDeny, tt.payload)
			}
		})
	}
}

// TestOpenCode_PostToolUseGatesOnTheOpenCodeSpelling: post-write checks are
// gated on "did this change a document", and that gate reads the folded tool
// name. Unfolded, every check silently never runs on this host.
func TestOpenCode_PostToolUseGatesOnTheOpenCodeSpelling(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tool string
		want bool
	}{
		{tool: "archcore_create_document", want: true},
		{tool: "archcore_update_document", want: true},
		{tool: "archcore_remove_document", want: true},
		{tool: "archcore_add_relation", want: true},
		{tool: "archcore_remove_relation", want: true},
		// A read changes nothing, and matching one costs a full corpus scan.
		{tool: "archcore_get_document", want: false},
		{tool: "archcore_search_documents", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			t.Parallel()
			if got := isMutatingArchcoreTool(tt.tool); got != tt.want {
				t.Errorf("isMutatingArchcoreTool(%q) = %v, want %v", tt.tool, got, tt.want)
			}
		})
	}
}
