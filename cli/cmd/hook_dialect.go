package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"archcore-cli/internal/agents"
)

// Host dialects.
//
// Every host runs the same guards and speaks a different protocol. The
// differences are not cosmetic: a host that does not recognize the envelope
// reads no context at all, and one that misreads an exit code blocks the user's
// edit. Each difference below is the host's documented behavior, not a
// preference.

type hookEvent string

const (
	eventSessionStart hookEvent = "SessionStart"
	eventPreToolUse   hookEvent = "PreToolUse"
	eventPostToolUse  hookEvent = "PostToolUse"
)

// contextEnvelope is how a host wants advisory context framed.
type contextEnvelope int

const (
	// envelopeClaudeWrapper — {"hookSpecificOutput":{"hookEventName":…,"additionalContext":…}}
	envelopeClaudeWrapper contextEnvelope = iota
	// envelopeCursorFlat — {"additional_context":…} (snake_case, no wrapper)
	envelopeCursorFlat
	// envelopeCopilotFlat — {"additionalContext":…} (camelCase, no wrapper)
	envelopeCopilotFlat
	// envelopePlainText — the bridge reads raw text, not JSON.
	envelopePlainText
)

// denyStyle is how a host is told to block a tool call.
type denyStyle int

const (
	// denyExitTwo — reason on stderr, stdout empty, exit 2.
	denyExitTwo denyStyle = iota
	// denyCopilotJSON — a decision document on stdout and exit 0. Copilot treats
	// ANY non-zero exit as a deny too, but discards the reason with it, so the
	// zero exit is what makes the explanation reach the user.
	denyCopilotJSON
)

// hostDialect is one host's protocol.
type hostDialect struct {
	id agents.AgentID
	// session is the SessionStart envelope, which is not always the same as the
	// tool-event envelope: Cursor takes the Claude wrapper at session start and
	// a flat snake_case key after a tool call.
	session  contextEnvelope
	envelope contextEnvelope
	deny     denyStyle
	// preToolContext reports whether the host's pre-write event can carry
	// advisory context at all. Copilot's cannot: its preToolUse accepts only a
	// permission decision, so context emitted there is discarded (github/
	// copilot-cli#2585). Sending it anyway would add a second stdout document
	// next to the decision and break the single-parse contract.
	preToolContext bool
	// wireEvents overrides the event name a host expects inside its envelope.
	// Absent means the canonical name. Only the name-carrying envelopes need an
	// entry; the flat ones carry no event name at all.
	wireEvents map[hookEvent]string
}

// wireEvent returns the event name this host expects to see in its envelope.
func (d hostDialect) wireEvent(e hookEvent) string {
	if name, ok := d.wireEvents[e]; ok {
		return name
	}
	return string(e)
}

// hookDialects is the registry of hosts the hook commands serve, which is a
// different question from which hosts `hooks install` writes a config for
// (wiring.hooksInstallers). Every host here answers the leaves; opencode is the
// one that answers them with no config written, because it loads hooks as
// plugin code and there is no declarative file to write. Membership here is
// what lets `hooks install` tell that host apart from a genuinely hookless one
// (cmd.servesHookEvents), and it is why an unregistered host is no longer met by
// cobra's help printer writing into the protocol channel.
//
// opencode is plain text on BOTH the session and the tool events, not only the
// tool ones. Its route is a TypeScript bridge that carries no decision logic:
// the plugin's bin/session-start streams this command's stdout through
// unchanged, and the bridge appends those bytes to the session verbatim. There
// is no JSON parse anywhere on that path, so a wrapper emitted here does not
// frame the recap — it lands in the model's context as literal JSON with the
// recap escaped inside it.
//
// The corollary is that opencode has no slot for the SessionStart banner, which
// the plain-text envelope therefore drops. That is the same call Copilot's
// envelope makes, and for the same reason: the banner is a line for the user,
// and smuggling it into the context channel makes it input for the model.
var hookDialects = []hostDialect{
	{id: agents.ClaudeCode, session: envelopeClaudeWrapper, envelope: envelopeClaudeWrapper, deny: denyExitTwo, preToolContext: true},
	{id: agents.Cursor, session: envelopeClaudeWrapper, envelope: envelopeCursorFlat, deny: denyExitTwo, preToolContext: true},
	{id: agents.GeminiCLI, session: envelopeClaudeWrapper, envelope: envelopeClaudeWrapper, deny: denyExitTwo, preToolContext: true,
		// Gemini is wired as BeforeTool / AfterTool, and its envelope carries the
		// event name — sending the canonical one means the host sees a name it
		// never registered.
		wireEvents: map[hookEvent]string{eventPreToolUse: "BeforeTool", eventPostToolUse: "AfterTool"}},
	{id: agents.Copilot, session: envelopeCopilotFlat, envelope: envelopeCopilotFlat, deny: denyCopilotJSON, preToolContext: false},
	{id: agents.CodexCLI, session: envelopeClaudeWrapper, envelope: envelopeClaudeWrapper, deny: denyExitTwo, preToolContext: true},
	{id: agents.OpenCode, session: envelopePlainText, envelope: envelopePlainText, deny: denyExitTwo, preToolContext: true},
}

// hookDecision is what a guard concluded. The zero value allows silently, which
// is what every failure path must produce.
type hookDecision struct {
	deny    bool
	reason  string
	context string
	// banner is the SessionStart systemMessage — a line for the user, not
	// context for the model. It rides the decision so SessionStart needs no
	// separate writer; an envelope without a slot for it simply drops it.
	banner string
}

// allowHook is the safe answer: no output, no block.
func allowHook() hookDecision { return hookDecision{} }

// denyHook blocks the tool call and explains why.
func denyHook(reason string) hookDecision { return hookDecision{deny: true, reason: reason} }

// adviseHook injects context without blocking anything.
func adviseHook(context string) hookDecision { return hookDecision{context: context} }

// adviseSession is adviseHook plus the user-facing banner only SessionStart has.
func adviseSession(context, banner string) hookDecision {
	return hookDecision{context: context, banner: banner}
}

// emitDecision writes the decision in the host's dialect and returns the process
// exit code.
//
// On Copilot stdout must hold exactly one JSON document: the host strips
// single-line progress objects, concatenates every remaining line, and runs one
// JSON.parse over the result. A failed parse is treated as no output at all, so
// a stray diagnostic line does not merely add noise — it discards the whole
// payload. Nothing here writes to stdout except the one document.
func emitDecision(d hostDialect, event hookEvent, dec hookDecision) int {
	if dec.deny {
		return emitDeny(d, dec.reason)
	}
	if dec.context != "" || dec.banner != "" {
		env := d.envelope
		if event == eventSessionStart {
			env = d.session
		}
		emitContext(env, d.wireEvent(event), dec.context, dec.banner)
	}
	return 0
}

func emitDeny(d hostDialect, reason string) int {
	if d.deny == denyCopilotJSON {
		payload := map[string]string{
			"permissionDecision":       "deny",
			"permissionDecisionReason": reason,
		}
		writeJSON(payload)
		return 0
	}
	fmt.Fprintln(os.Stderr, reason)
	return 2
}

func emitContext(env contextEnvelope, wireEvent, text, banner string) {
	switch env {
	case envelopePlainText:
		// A banner alone reaches here when the context is empty, and this
		// envelope has nowhere to put one. Printing anyway would send a bare
		// newline down a channel whose every byte becomes model context.
		if text == "" {
			return
		}
		fmt.Fprintln(os.Stdout, text)
	case envelopeCursorFlat:
		writeJSON(map[string]string{"additional_context": text})
	case envelopeCopilotFlat:
		// No slot for a banner here, so it is dropped rather than smuggled into
		// the context: it is a line for the user, not input for the model.
		writeJSON(map[string]string{"additionalContext": text})
	case envelopeClaudeWrapper:
		out := map[string]any{
			"hookSpecificOutput": map[string]any{
				"hookEventName":     wireEvent,
				"additionalContext": text,
			},
		}
		if banner != "" {
			out["systemMessage"] = banner
		}
		writeJSON(out)
	}
}

// writeJSON emits one document. A marshal failure writes nothing rather than a
// fragment: on Copilot a fragment would poison the parse of the whole stream.
func writeJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	_, _ = os.Stdout.Write(data)
}
