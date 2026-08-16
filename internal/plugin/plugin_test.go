package plugin

import (
	"slices"
	"testing"

	"archcore-cli/internal/agents"
)

// TestFrozenIdentifiers pins the three identifiers requirement 11 of
// plugin-cli-compatibility.rule freezes. A rename here is a rename of the
// plugin's public surface, and it may only ship in step with the
// archcore-ai/plugin repository.
func TestFrozenIdentifiers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "repository", got: RepoID, want: "archcore-ai/plugin"},
		{name: "marketplace", got: MarketplaceID, want: "archcore-plugins"},
		{name: "plugin id", got: PluginID, want: "archcore@archcore-plugins"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("frozen identifier = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestHosts(t *testing.T) {
	t.Parallel()
	want := []Host{HostClaudeCode, HostCursor, HostCodexCLI, HostCopilot}
	if got := Hosts(); !slices.Equal(got, want) {
		t.Errorf("Hosts() = %q, want %q", got, want)
	}
}

// TestHostsIsACopy proves a caller cannot reorder the canonical order for the
// rest of the process. Plan's determinism depends on that order.
func TestHostsIsACopy(t *testing.T) {
	t.Parallel()
	first := Hosts()
	first[0] = "rewritten"
	if second := Hosts(); second[0] != HostClaudeCode {
		t.Errorf("Hosts()[0] = %q after a caller wrote to an earlier result, want %q", second[0], HostClaudeCode)
	}
}

func TestHostFromAgent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		agentID string
		want    Host
		wantOK  bool
	}{
		{name: "claude code", agentID: "claude-code", want: HostClaudeCode, wantOK: true},
		{name: "cursor", agentID: "cursor", want: HostCursor, wantOK: true},
		{name: "codex cli", agentID: "codex-cli", want: HostCodexCLI, wantOK: true},
		{name: "copilot", agentID: "copilot", want: HostCopilot, wantOK: true},
		{name: "gemini cli ships no plugin", agentID: "gemini-cli"},
		{name: "opencode ships no plugin", agentID: "opencode"},
		{name: "roo code ships no plugin", agentID: "roo-code"},
		{name: "cline ships no plugin", agentID: "cline"},
		{name: "unknown agent", agentID: "not-an-agent"},
		{name: "empty id", agentID: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := HostFromAgent(tt.agentID)
			if ok != tt.wantOK {
				t.Fatalf("HostFromAgent(%q) ok = %v, want %v", tt.agentID, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("HostFromAgent(%q) = %q, want %q", tt.agentID, got, tt.want)
			}
		})
	}
}

// TestHostsMapOntoTheAgentRegistry guards the join with internal/agents. Every
// plugin host must name a registered agent, so a rename in the registry cannot
// leave the plugin surface addressing a host that no longer exists.
func TestHostsMapOntoTheAgentRegistry(t *testing.T) {
	t.Parallel()
	registered := make(map[string]bool)
	for _, id := range agents.AllIDs() {
		registered[string(id)] = true
	}
	for _, host := range Hosts() {
		if !registered[string(host)] {
			t.Errorf("plugin host %q names no agent in internal/agents", host)
		}
	}
	for _, id := range agents.AllIDs() {
		host, ok := HostFromAgent(string(id))
		if ok && host != Host(id) {
			t.Errorf("HostFromAgent(%q) = %q, want the same string", id, host)
		}
	}
}

func TestCommandString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cmd  Command
		want string
	}{
		{name: "zero command", cmd: Command{}, want: ""},
		{name: "name only", cmd: Command{Name: "claude"}, want: "claude"},
		{
			name: "name and args",
			cmd:  Command{Name: "claude", Args: []string{"plugin", "update", PluginID}},
			want: "claude plugin update archcore@archcore-plugins",
		},
		{
			name: "empty name with args stays empty",
			cmd:  Command{Args: []string{"plugin", "list"}},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cmd.String(); got != tt.want {
				t.Errorf("Command.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVerbString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		verb Verb
		want string
	}{
		{name: "install", verb: VerbInstall, want: "install"},
		{name: "update", verb: VerbUpdate, want: "update"},
		{name: "remove", verb: VerbRemove, want: "remove"},
		{name: "status", verb: VerbStatus, want: "status"},
		{name: "out of range", verb: Verb(99), want: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.verb.String(); got != tt.want {
				t.Errorf("Verb.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestActionKindString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		kind ActionKind
		want string
	}{
		{name: "run", kind: ActionRun, want: "run"},
		{name: "print command", kind: ActionPrintCommand, want: "print-command"},
		{name: "print ui note", kind: ActionPrintUINote, want: "print-ui-note"},
		{name: "report installed", kind: ActionReportInstalled, want: "report-installed"},
		{name: "report status", kind: ActionReportStatus, want: "report-status"},
		{name: "out of range", kind: ActionKind(99), want: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.want {
				t.Errorf("ActionKind.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
