package cmd

import (
	"encoding/json"
	"io"
	"regexp"
	"strings"
)

// Hook payload decoding.
//
// Six hosts send six shapes for the same event, and two nest a JSON document
// inside a string field, so the payload is held as a generic tree and read
// through explicit paths rather than a typed struct.

// hookPayload is the decoded hook input. A payload that failed to decode is not
// an error state: it is an empty payload, and every guard reading it must then
// allow. A hook that cannot parse its input must never block the user's work.
type hookPayload struct {
	root map[string]any
}

// maxPayloadBytes bounds how much of the hook input is read. A PostToolUse
// payload echoes tool_response, and a Write payload carries the whole file body,
// so the input is only as small as the host chooses to make it. Decoded into
// map[string]any it costs several times its own size in live objects.
//
// 4 MiB rather than something tighter: a payload whose leading object is cut off
// by the cap fails to parse and therefore allows, so the cap has to sit above
// any plausible single-file edit or it becomes a way to slip a write past the
// guard.
const maxPayloadBytes = 4 << 20

// decodeHookPayload reads and parses the hook input. It never fails: empty,
// truncated, or non-JSON stdin yields an empty payload.
//
// The stream decoder reads the leading JSON object and stops, so anything after
// it — a second object, padding, binary junk — is ignored rather than rejected.
// That is the safe direction and is deliberate: requiring the whole input to be
// exactly one object would turn "valid object plus trailing bytes" from a
// payload the write guard can act on into an empty one, which every guard
// treats as allow. Strictness here would BE the bypass.
//
// The consequence to keep in mind: the cap bounds the bytes read, not the size
// of what the host sent.
func decodeHookPayload(r io.Reader) *hookPayload {
	var root map[string]any
	if json.NewDecoder(io.LimitReader(r, maxPayloadBytes)).Decode(&root) != nil {
		return &hookPayload{}
	}
	return &hookPayload{root: root}
}

// str reads a string at a path of nested keys. An intermediate value that is
// itself a JSON document in string form is decoded on the way through, which is
// what makes Copilot's toolArgs and Cursor's tool_input readable by path.
func (p *hookPayload) str(path ...string) string {
	var cur any = p.root
	for _, key := range path {
		node, ok := asObject(cur)
		if !ok {
			return ""
		}
		cur, ok = node[key]
		if !ok {
			return ""
		}
	}
	s, _ := cur.(string)
	return s
}

// firstStr returns the first non-empty value among several candidate paths.
func (p *hookPayload) firstStr(paths ...[]string) string {
	for _, path := range paths {
		if v := p.str(path...); v != "" {
			return v
		}
	}
	return ""
}

// asObject coerces a node to an object, decoding an embedded JSON document when
// the node is a string carrying one.
func asObject(v any) (map[string]any, bool) {
	switch t := v.(type) {
	case map[string]any:
		return t, true
	case string:
		trimmed := strings.TrimSpace(t)
		if !strings.HasPrefix(trimmed, "{") {
			return nil, false
		}
		var nested map[string]any
		if json.Unmarshal([]byte(trimmed), &nested) != nil {
			return nil, false
		}
		return nested, true
	default:
		return nil, false
	}
}

// sessionID returns the host's identifier for the conversation. Claude Code and
// Codex send session_id; Cursor sends conversation_id.
func (p *hookPayload) sessionID() string {
	return p.firstStr([]string{"session_id"}, []string{"conversation_id"})
}

func (p *hookPayload) source() string { return p.str("source") }

func (p *hookPayload) cwd() string { return p.str("cwd") }

// toolName returns the invoked tool, folded to its canonical form.
func (p *hookPayload) toolName() string { return foldToolName(p.rawToolName()) }

// rawToolName returns the tool name exactly as the host sent it.
func (p *hookPayload) rawToolName() string {
	return p.firstStr([]string{"tool_name"}, []string{"toolName"})
}

// filePath returns the target of a file-writing tool. The candidate list spans
// every host shape; the first hit wins, and an absent path means "no file
// involved", which every guard treats as allow.
//
// The bare "path" key is consulted only when the caller is NOT an archcore MCP
// tool. The same key means two different things depending on who sent it: a
// file being edited, or the document an MCP call is acting on. Reading it
// unconditionally would let a sanctioned MCP write look like a direct one and
// get blocked by its own guard.
//
// tool_input.filePath is listed because OpenCode's write and edit tools name
// their argument that way, and its bridge is expected — but not able to be
// forced — to rename it on the way in. Missing it fails in the silent
// direction: the guard finds no path, allows, and a session with no protection
// looks exactly like a clean one. The snake_case key stays first, so a bridge
// that does rename still wins the tie.
func (p *hookPayload) filePath() string {
	paths := [][]string{
		{"tool_input", "file_path"},
		{"tool_input", "filePath"},
		{"toolArgs", "file_path"},
		{"toolArgs", "filePath"},
		{"tool_args", "file_path"},
		{"tool_args", "filePath"},
		{"file_path"},
		{"filePath"},
	}
	if !isArchcoreMCPTool(p.rawToolName()) {
		paths = append(paths,
			[]string{"tool_input", "path"},
			[]string{"toolArgs", "path"},
			[]string{"tool_args", "path"})
	}
	return p.firstStr(paths...)
}

// patchDirective matches an apply-patch line that names a file. The format is
// the OpenAI patch envelope, and these four verbs are the only ways it addresses
// a path.
//
// "Move to" is the rename destination, and it is the one that names a path the
// other three never mention. A rename is written as an update of the source
// followed by this line, so reading only the first three sees the source and
// misses where the bytes actually land: a patch that updates a file outside the
// store and moves it into .archcore/ would pass the guard and then write the
// document anyway.
//
// The pattern is deliberately looser than the format it reads. Case and inner
// spacing are accepted because this guard cannot see the parser that will apply
// the patch, and matching a host's exact strictness would make the guard's
// coverage an assumption about someone else's code. Loose costs a denied patch
// that merely quotes a directive; strict costs a write that slipped past a
// parser more forgiving than this one.
var patchDirective = regexp.MustCompile(`(?i)^\*\*\*\s*(?:(?:add|update|delete)\s+file|move\s+to)\s*:\s*(\S.*)$`)

// maxPatchLines bounds the scan. The payload cap already bounds the bytes; this
// bounds the work on a pathological one, because the pre-write guard blocks the
// user while it runs. The split is lazy so the bound holds on the work as well
// as on the checks — materializing every line first would allocate the whole
// payload before the limit could refuse any of it.
const maxPatchLines = 20000

// patchPaths returns every file an apply-patch call would touch.
//
// This exists because a patch tool names its targets nowhere else. OpenCode's
// apply_patch takes a single patchText argument and no path at all, so
// filePath() finds nothing and the write guard allows — and on that host the
// tool is not an alternative to write and edit but a replacement for them: its
// registry enables apply_patch and DISABLES write and edit for gpt- models
// (packages/opencode/src/tool/registry.ts). A session on such a model would
// otherwise have no write protection whatsoever, and nothing would distinguish
// it from a protected one.
//
// The hole is not OpenCode's alone. Codex CLI and Copilot both match apply_patch
// in their installed pre-write matchers (@internal/wiring/hooks_agents.go), so
// the same path-less call reaches this guard on hosts the CLI does wire.
// [assumption] Their argument keys are unverified from this repository, so
// "input" and "patch" are read alongside OpenCode's "patchText". A key that
// names something other than a patch costs a scan that finds no directives.
//
// Unparsable or empty patch text yields no paths, which allows. That is the
// same direction every other read in this file fails. A line is matched after
// its leading whitespace is trimmed, so a hunk's context line quoting a
// directive is read as one — a false deny, which is the safe direction here.
func (p *hookPayload) patchPaths() []string {
	text := p.firstStr(
		[]string{"tool_input", "patchText"},
		[]string{"toolArgs", "patchText"},
		[]string{"tool_args", "patchText"},
		[]string{"patchText"},
		[]string{"tool_input", "input"},
		[]string{"toolArgs", "input"},
		[]string{"tool_args", "input"},
		[]string{"tool_input", "patch"},
		[]string{"toolArgs", "patch"},
		[]string{"tool_args", "patch"},
	)
	if text == "" {
		return nil
	}

	var out []string
	seen := 0
	for line := range strings.SplitSeq(text, "\n") {
		if seen >= maxPatchLines {
			break
		}
		seen++
		line = strings.TrimSpace(line)
		// The envelope markers start this way too, so the prefix only keeps the
		// regexp off the hunk body — the pattern itself decides.
		if !strings.HasPrefix(line, "***") {
			continue
		}
		if m := patchDirective.FindStringSubmatch(line); m != nil {
			out = append(out, strings.TrimSpace(m[1]))
		}
	}
	return out
}

// docPath returns the .archcore/ document an MCP tool acted on.
func (p *hookPayload) docPath() string {
	return p.firstStr(
		[]string{"tool_input", "path"},
		[]string{"toolArgs", "path"},
		[]string{"tool_args", "path"},
		[]string{"path"},
	)
}

// MCP tool-name prefixes. One server reaches the CLI under five names,
// depending on whether the plugin bundles it, how the host flattens namespaces,
// and which separator convention the host uses.
const (
	mcpCanonicalPrefix = "mcp__archcore__"
	mcpPluginPrefix    = "mcp__plugin_archcore_archcore__"
	mcpCopilotPrefix   = "archcore-"
	mcpGeminiPrefix    = "mcp_archcore_"
	mcpOpenCodePrefix  = "archcore_"
)

// archcoreMCPTools is every tool the archcore MCP server registers. It exists to
// bound the loose prefixes below, and TestArchcoreMCPTools_MatchesTheServer holds
// it against the server's own registration.
//
// A tool missing here is not folded under a loose spelling, so the write guard
// reads its "path" argument as a direct edit and denies it. That is the visible
// direction to be wrong in; the silent one is what this whole guard exists to
// prevent.
var archcoreMCPTools = map[string]bool{
	"init_project":        true,
	"install_host_config": true,
	"list_documents":      true,
	"get_document":        true,
	"search_documents":    true,
	"create_document":     true,
	"update_document":     true,
	"remove_document":     true,
	"add_relation":        true,
	"remove_relation":     true,
	"list_relations":      true,
}

// foldToolName rewrites any known spelling of an archcore MCP tool to the
// canonical one. A tool from a different server is returned untouched: folding
// it would make an unrelated write look like a document mutation.
//
// The prefixes cannot overlap — CutPrefix anchors at the start — so the order
// below is for reading, not for correctness.
//
// Two of them delimit the server name unambiguously with a double underscore.
// The other three are host flattenings that join the server name to the tool
// name with a single separator, and a separator that appears inside names
// cannot mark where one ends: "archcore_docs_create_document" is a tool of a
// server called archcore_docs, and nothing in the string says so. Left
// unbounded, those three claim it, and the cost is not a mislabeled name — it is
// that filePath() then stops reading the bare "path" key and the write guard
// waves a foreign server's write into .archcore/ through. So the loose three
// fold only onto a tool this server actually has.
func foldToolName(name string) string {
	if rest, ok := strings.CutPrefix(name, mcpPluginPrefix); ok {
		return mcpCanonicalPrefix + rest
	}
	for _, prefix := range []string{mcpGeminiPrefix, mcpCopilotPrefix, mcpOpenCodePrefix} {
		if rest, ok := strings.CutPrefix(name, prefix); ok && archcoreMCPTools[rest] {
			return mcpCanonicalPrefix + rest
		}
	}
	return name
}

// isArchcoreMCPTool reports whether name is an archcore document tool, in any
// spelling.
func isArchcoreMCPTool(name string) bool {
	return strings.HasPrefix(foldToolName(name), mcpCanonicalPrefix)
}

// mutatingMCPTools are the archcore tools that change the knowledge base. It is
// the same set the host-side matcher filters on (wiring.mcpDocumentTools), and
// a cross-package test holds the two in agreement.
var mutatingMCPTools = map[string]bool{
	"create_document": true,
	"update_document": true,
	"remove_document": true,
	"add_relation":    true,
	"remove_relation": true,
}

// isMutatingArchcoreTool reports whether name is an archcore tool that wrote
// something, in any spelling.
func isMutatingArchcoreTool(name string) bool {
	rest, ok := strings.CutPrefix(foldToolName(name), mcpCanonicalPrefix)
	return ok && mutatingMCPTools[rest]
}
