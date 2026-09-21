// Normative properties pinned here — tool-descriptions-fit-the-host-cap.adr:
//
//  1. A tool description holds at most hostDescriptionCap UTF-16 units, unless
//     overCapTools lists the tool with the reason.
//  2. A tool listed in overCapTools still exceeds the cap and is still registered.
//  3. The part of the add_relation description that the host passes on carries
//     the relation policy.
package mcp

import (
	"strings"
	"testing"
	"unicode/utf16"
)

// hostDescriptionCap is how many UTF-16 units of a tool description Claude Code
// passes to the model. The rest never reaches it, so a rule placed past the cap
// does not exist for that host — tool-descriptions-fit-the-host-cap.adr.
const hostDescriptionCap = 2048

var overCapTools = map[string]string{
	"create_document":  "the document-type catalog fills the cap, and the Returns paragraph sits past it",
	"search_documents": "the closing sentence ends 9 units past the cap; host-truncation-safe-read-tools.plan shortens this description",
}

func registeredTools(t *testing.T) map[string]string {
	t.Helper()
	srv := NewServer(t.TempDir(), "test",
		WithHostWiring(func(string, string, bool) ([]byte, error) { return nil, nil }))
	descriptions := make(map[string]string)
	for name, tool := range srv.ListTools() {
		descriptions[name] = tool.Tool.Description
	}
	if len(descriptions) == 0 {
		t.Fatal("the server registered no tools, so this test proves nothing")
	}
	return descriptions
}

func hostVisible(description string) string {
	units := utf16.Encode([]rune(description))
	if len(units) > hostDescriptionCap {
		units = units[:hostDescriptionCap]
	}
	return string(utf16.Decode(units))
}

func TestToolDescriptions_FitTheHostCap(t *testing.T) {
	t.Parallel()
	descriptions := registeredTools(t)
	for name, description := range descriptions {
		units := len(utf16.Encode([]rune(description)))
		reason, exempt := overCapTools[name]
		switch {
		case units > hostDescriptionCap && !exempt:
			t.Errorf("%s: the description holds %d UTF-16 units and the host passes on %d, so its tail never reaches the model",
				name, units, hostDescriptionCap)
		case units <= hostDescriptionCap && exempt:
			t.Errorf("%s: the description now fits the cap at %d units, so remove it from overCapTools (%s)",
				name, units, reason)
		}
	}
	for name := range overCapTools {
		if _, ok := descriptions[name]; !ok {
			t.Errorf("overCapTools lists %q and the server does not register it", name)
		}
	}
}

func TestAddRelationDescription_CarriesTheRelationPolicy(t *testing.T) {
	t.Parallel()
	visible := hostVisible(registeredTools(t)["add_relation"])
	for _, phrase := range []string{
		"list_relations",
		"joint reading task",
		"do not use related as a fallback",
		"reverse related",
		"leave the pair unlinked",
		"warnings",
	} {
		if !strings.Contains(visible, phrase) {
			t.Errorf("the host-visible add_relation description lacks %q: the relation policy has no other surface that reaches the model whole", phrase)
		}
	}
}
