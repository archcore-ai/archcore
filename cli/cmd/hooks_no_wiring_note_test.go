package cmd

import (
	"slices"
	"strings"
	"testing"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/wiring"
)

// What `hooks install` says when it writes no hook config.
//
// Two different facts used to share one sentence. An agent with no hook wiring
// either has no hooks at all, or loads them as plugin code this CLI cannot
// write. Reporting the second as the first tells an OpenCode user the opposite
// of the truth: the plugin gives them the same three events every wired host
// runs, and the message would send them looking for a feature they already have.

// TestNoHookWiringNote_SeparatesPluginCodeFromHookless drives every registered
// agent through the branch `hooks install` actually takes.
//
// The classification is not hardcoded here: an agent is asked whether an install
// writes anything, and only the ones that write nothing reach this note. That
// keeps the test honest about which fact each agent carries, rather than
// restating the table it is meant to check.
func TestNoHookWiringNote_SeparatesPluginCodeFromHookless(t *testing.T) {
	t.Parallel()
	pluginCoded := map[agents.AgentID]bool{}

	for _, agent := range agents.All() {
		t.Run(string(agent.ID), func(t *testing.T) {
			base := setupArchcoreDir(t)

			installed, err := wiring.InstallHooksForAgent(base, agent)
			if err != nil {
				t.Fatalf("InstallHooksForAgent(%s): %v", agent.ID, err)
			}
			if installed {
				// A written config is not this note's case at all.
				return
			}

			note := noHookWiringNote(agent)
			if !strings.Contains(note, agent.DisplayName) {
				t.Errorf("note does not name the agent: %q", note)
			}

			if !servesHookEvents(agent.ID) {
				if !strings.Contains(note, "does not support hooks") {
					t.Errorf("an agent serving no hook events should be reported as hookless, got %q", note)
				}
				return
			}

			pluginCoded[agent.ID] = true
			if strings.Contains(note, "does not support hooks") {
				t.Errorf("an agent whose hooks load as plugin code was reported as hookless: %q", note)
			}
			if !strings.Contains(note, "plugin code") {
				t.Errorf("note does not say the hooks load as plugin code: %q", note)
			}
			// The note is only useful if it names the command the plugin calls,
			// which is what turns "no config written" into an action.
			if want := "archcore hooks " + string(agent.ID); !strings.Contains(note, want) {
				t.Errorf("note does not name %q: %q", want, note)
			}
		})
	}

	t.Cleanup(func() {
		if len(pluginCoded) != 1 || !pluginCoded[agents.OpenCode] {
			t.Errorf("agents taking the plugin-code branch = %v, want exactly {opencode}", pluginCoded)
		}
	})
}

// TestServesHookEvents_TracksTheDialectRegistry: the note reads this predicate,
// and the predicate reads hookDialects. A host gains its leaves by being added
// there, so a host added to the registry without wiring stops being described as
// hookless on its own — which is the property that keeps the two facts from
// collapsing back into one sentence.
// The negative direction is asserted against the agent registry rather than
// against hookDialects. Comparing the predicate to a set derived from the same
// table it reads would be a tautology — it holds no matter what either says.
func TestServesHookEvents_TracksTheDialectRegistry(t *testing.T) {
	t.Parallel()
	registered := map[agents.AgentID]bool{}
	for _, d := range hookDialects {
		registered[d.id] = true
		if !servesHookEvents(d.id) {
			t.Errorf("servesHookEvents(%s) = false for a registered dialect", d.id)
		}
	}

	// Agents that are known to the CLI and are not in the dialect table. These
	// are the ones `hooks install` must call hookless, and the count is asserted
	// so that adding a dialect for one of them fails here rather than silently
	// changing what a user is told.
	var hookless []agents.AgentID
	for _, agent := range agents.All() {
		if !registered[agent.ID] {
			hookless = append(hookless, agent.ID)
			if servesHookEvents(agent.ID) {
				t.Errorf("servesHookEvents(%s) = true though no dialect is registered", agent.ID)
			}
		}
	}
	if want := []agents.AgentID{agents.RooCode, agents.Cline}; !slices.Equal(hookless, want) {
		t.Errorf("agents with no dialect = %v, want %v — update this list with the dialect table", hookless, want)
	}
}
