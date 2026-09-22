package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const migrationClaudeCatalog = `{"name":"archcore-plugins","plugins":[{"name":"archcore","source":"./plugins/archcore"}]}`
const migrationCodexCatalog = `{"name":"archcore-plugins","plugins":[{"name":"archcore","source":{"source":"local","path":"./plugins/archcore"}}]}`
const migrationKnownSource = `{"archcore-plugins":{"source":{"source":"github","repo":"archcore-ai/plugin"},"autoUpdate":false,"custom":{"keep":true}}}`
const migrationClaudeSettings = `{"enabledPlugins":{"archcore@archcore-plugins":false},"extraKnownMarketplaces":{"archcore-plugins":{"source":{"source":"github","repo":"archcore-ai/plugin"},"autoUpdate":false,"custom":{"keep":true}}},"model":"keep-me"}`
const migrationCodexSettings = "# keep comment\nmodel = \"keep-me\"\n[marketplaces.archcore-plugins]\nsource_type = \"git\"\nsource = \"https://github.com/archcore-ai/plugin.git\" # keep comment\n[plugins.\"archcore@archcore-plugins\"]\nenabled = false\n"

type migrationTransport func(*http.Request) (*http.Response, error)

func (m migrationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m(req)
}

func migrationClient(status int, body string, calls *int) *http.Client {
	return &http.Client{Transport: migrationTransport(func(req *http.Request) (*http.Response, error) {
		*calls++
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})}
}

func writeMigrationFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func migrationFixture(t *testing.T, host Host) (Action, []string) {
	t.Helper()
	home := isolateHome(t)
	claude := filepath.Join(home, ".claude")
	codex := filepath.Join(home, ".codex")
	t.Setenv("CLAUDE_CONFIG_DIR", claude)
	t.Setenv("CODEX_HOME", codex)
	a := Plan(VerbUpdate, []Evidence{listedEvidence(host)})[0]
	if host == HostCodexCLI {
		path := filepath.Join(codex, "config.toml")
		writeMigrationFile(t, path, migrationCodexSettings)
		stubRuns(t, func(c Command) commandOutcome {
			if c.String() != "codex plugin marketplace list --json" {
				t.Fatalf("unexpected command %s", c.String())
			}
			return commandOutcome{Stdout: `{"marketplaces":[{"name":"archcore-plugins","marketplaceSource":{"sourceType":"git","source":"https://github.com/archcore-ai/plugin.git"}}]}`}
		})
		return a, []string{path}
	}
	known := filepath.Join(claude, "plugins", "known_marketplaces.json")
	settings := filepath.Join(claude, "settings.json")
	writeMigrationFile(t, known, migrationKnownSource)
	writeMigrationFile(t, settings, migrationClaudeSettings)
	return a, []string{known, settings}
}

func TestRewriteClaudeSource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		source  string
		want    string
		blocked bool
	}{
		{name: "github", source: `{"source":"github","repo":"archcore-ai/plugin"}`, want: canonicalRepoID},
		{name: "https", source: `{"source":"git","url":"https://github.com/archcore-ai/plugin.git"}`, want: "https://github.com/archcore-ai/archcore.git"},
		{name: "ssh", source: `{"source":"git","url":"git@github.com:archcore-ai/plugin.git"}`, want: "git@github.com:archcore-ai/archcore.git"},
		{name: "canonical", source: `{"source":"github","repo":"archcore-ai/archcore"}`},
		{name: "fork", source: `{"source":"github","repo":"someone/archcore"}`, blocked: true},
		{name: "ref", source: `{"source":"github","repo":"archcore-ai/plugin","ref":"dev"}`, blocked: true},
		{name: "unknown field", source: `{"source":"github","repo":"archcore-ai/plugin","future":true}`, blocked: true},
		{name: "local", source: `{"source":"directory","path":"archcore-ai/plugin"}`, blocked: true},
		{name: "other host", source: `{"source":"git","url":"https://example.invalid/archcore-ai/plugin.git"}`, blocked: true},
		{name: "suffix attack", source: `{"source":"github","repo":"archcore-ai/plugin-custom"}`, blocked: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.Replace(migrationClaudeSettings, `{"source":"github","repo":"archcore-ai/plugin"}`, tt.source, 1)
			after, blocked, err := rewriteClaudeSource([]byte(input), true)
			if err != nil || blocked != tt.blocked {
				t.Fatalf("blocked=%v err=%v", blocked, err)
			}
			if tt.want == "" {
				if after != nil {
					t.Fatal("unchanged source was rewritten")
				}
				return
			}
			var doc map[string]any
			if err := json.Unmarshal(after, &doc); err != nil {
				t.Fatal(err)
			}
			entry := doc[extraKnownMarketplacesKey].(map[string]any)[MarketplaceID].(map[string]any)
			source := entry[marketplaceSourceKey].(map[string]any)
			if source["repo"] != tt.want && source["url"] != tt.want {
				t.Fatalf("source=%v", source)
			}
			if entry["autoUpdate"] != false || doc["enabledPlugins"].(map[string]any)[PluginID] != false || doc["model"] != "keep-me" || entry["custom"].(map[string]any)["keep"] != true {
				t.Fatalf("unrelated settings changed: %s", after)
			}
		})
	}
}

func TestRewriteCodexSource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, input string
		changed     bool
	}{
		{name: "native table", input: migrationCodexSettings, changed: true},
		{name: "quoted table", input: strings.ReplaceAll(migrationCodexSettings, "[marketplaces.archcore-plugins]", `[marketplaces."archcore-plugins"]`), changed: true},
		{name: "literal quotes", input: strings.ReplaceAll(migrationCodexSettings, `"https://github.com/archcore-ai/plugin.git"`, `'https://github.com/archcore-ai/plugin.git'`), changed: true},
		{name: "canonical", input: strings.ReplaceAll(migrationCodexSettings, legacyRepoID, canonicalRepoID)},
		{name: "fork", input: strings.ReplaceAll(migrationCodexSettings, legacyRepoID, "other/fork")},
		{name: "ref", input: strings.ReplaceAll(migrationCodexSettings, "source_type", "ref = \"dev\"\nsource_type")},
		{name: "nested fetch fields", input: migrationCodexSettings + "[marketplaces.archcore-plugins.headers]\nToken=\"keep\"\n"},
		{name: "duplicate source", input: strings.ReplaceAll(migrationCodexSettings, "source_type", "source = \"other/fork\"\nsource_type")},
		{name: "multiline decoy", input: "note = '''\n" + migrationCodexSettings + "'''\n"},
		{name: "inline table", input: `[marketplaces]
archcore-plugins = {source_type = "git", source = "https://github.com/archcore-ai/plugin.git"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			after := rewriteCodexSource([]byte(tt.input))
			if !tt.changed {
				if after != nil {
					t.Fatalf("unsupported source rewritten: %s", after)
				}
				return
			}
			want := strings.Replace(tt.input, legacyRepoID, canonicalRepoID, 1)
			if string(after) != want {
				t.Fatalf("got %q want %q", after, want)
			}
		})
	}
}

func TestMigrateSourcePreservesSettingsAndIsIdempotent(t *testing.T) {
	for _, host := range []Host{HostClaudeCode, HostCodexCLI} {
		t.Run(string(host), func(t *testing.T) {
			a, paths := migrationFixture(t, host)
			before := make(map[string][]byte, len(paths))
			for _, path := range paths {
				before[path], _ = os.ReadFile(path)
			}
			catalog := migrationClaudeCatalog
			if host == HostCodexCLI {
				catalog = migrationCodexCatalog
			}
			calls := 0
			client := migrationClient(http.StatusOK, catalog, &calls)
			if err := migrateSource(t.Context(), a, canonicalRepoID, client); err != nil {
				t.Fatal(err)
			}
			for _, path := range paths {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Contains(data, []byte(canonicalRepoID)) || bytes.Contains(data, []byte(legacyRepoID)) {
					t.Fatalf("source not migrated: %s", data)
				}
				backups, err := filepath.Glob(path + ".archcore-source-*.bak")
				if err != nil || len(backups) != 1 {
					t.Fatalf("backups=%v err=%v", backups, err)
				}
				backup, err := os.ReadFile(backups[0])
				if err != nil || !bytes.Equal(backup, before[path]) {
					t.Fatalf("backup lost original bytes: %v", err)
				}
				aged := ageFile(t, path)
				if err := migrateSource(t.Context(), a, canonicalRepoID, client); err != nil {
					t.Fatal(err)
				}
				if !modTime(t, path).Equal(aged) {
					t.Fatal("repeat migration rewrote settings")
				}
			}
			if calls != 1 {
				t.Fatalf("catalog probes=%d want 1", calls)
			}
		})
	}
}

func TestMigrationCatalogFailuresLeaveSourcesIntact(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "unpublished", status: 404},
		{name: "wrong marketplace", status: 200, body: strings.Replace(migrationClaudeCatalog, "archcore-plugins", "other", 1)},
		{name: "wrong plugin path", status: 200, body: strings.Replace(migrationClaudeCatalog, "./plugins/archcore", "./other", 1)},
		{name: "invalid JSON", status: 200, body: "{"},
		{name: "oversized", status: 200, body: strings.Repeat(" ", maxMigrationCatalogBytes+1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, paths := migrationFixture(t, HostClaudeCode)
			calls := 0
			err := migrateSource(t.Context(), a, canonicalRepoID, migrationClient(tt.status, tt.body, &calls))
			if err == nil {
				t.Fatal("invalid target accepted")
			}
			for i, path := range paths {
				data, _ := os.ReadFile(path)
				want := migrationKnownSource
				if i == 1 {
					want = migrationClaudeSettings
				}
				if string(data) != want {
					t.Fatal("source changed on failed target verification")
				}
				backups, _ := filepath.Glob(path + ".archcore-source-*.bak")
				if len(backups) != 0 {
					t.Fatal("backup created before target verification")
				}
			}
		})
	}
}

func TestMigrationPreservesCustomClaudeDeclarations(t *testing.T) {
	a, paths := migrationFixture(t, HostClaudeCode)
	custom := strings.Replace(migrationClaudeSettings, legacyRepoID, "someone/fork", 1)
	writeMigrationFile(t, paths[1], custom)
	calls := 0
	if err := migrateSource(t.Context(), a, canonicalRepoID, migrationClient(200, migrationClaudeCatalog, &calls)); err != nil {
		t.Fatal(err)
	}
	known, _ := os.ReadFile(paths[0])
	if string(known) != migrationKnownSource || calls != 0 {
		t.Fatal("custom declaration caused a migration")
	}
}

func TestMigrationIncludesOnlyAddressableClaudeProjects(t *testing.T) {
	a, paths := migrationFixture(t, HostClaudeCode)
	project, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(project, ".claude", "settings.local.json")
	writeMigrationFile(t, path, migrationClaudeSettings)
	managed := filepath.Join(t.TempDir(), ".claude", "settings.json")
	writeMigrationFile(t, managed, migrationClaudeSettings)
	a.Evidence.Installs = []Install{{Scope: ScopeLocal, ProjectPath: project, ProjectPresent: true}, {Scope: ScopeManaged, ProjectPath: filepath.Dir(filepath.Dir(managed)), ProjectPresent: true}}
	calls := 0
	if err := migrateSource(t.Context(), a, canonicalRepoID, migrationClient(200, migrationClaudeCatalog, &calls)); err != nil {
		t.Fatal(err)
	}
	for _, file := range append(paths, path) {
		data, _ := os.ReadFile(file)
		if !bytes.Contains(data, []byte(canonicalRepoID)) {
			t.Fatalf("source not migrated: %s", file)
		}
	}
	data, _ := os.ReadFile(managed)
	if string(data) != migrationClaudeSettings {
		t.Fatal("managed settings changed")
	}
}

func TestSourceMigrationIsOnlyRequestedByUpdates(t *testing.T) {
	t.Parallel()
	for _, verb := range []Verb{VerbInstall, VerbUpdate, VerbRemove, VerbStatus} {
		for _, action := range Plan(verb, []Evidence{listedEvidence(HostClaudeCode), listedEvidence(HostCodexCLI)}) {
			if action.MigrateSource != (verb == VerbUpdate) {
				t.Errorf("%v planned migration=%v", verb, action.MigrateSource)
			}
		}
	}
}

func TestMigrationDoesNotRunBeforeCutoverOrInPrintOnly(t *testing.T) {
	a, paths := migrationFixture(t, HostClaudeCode)
	calls := 0
	if err := migrateSource(t.Context(), a, legacyRepoID, migrationClient(200, migrationClaudeCatalog, &calls)); err != nil {
		t.Fatal(err)
	}
	rec := recordRuns(t)
	results := Execute(t.Context(), []Action{a}, &recordingReporter{}, ExecuteOptions{Repository: canonicalRepoID, PrintOnly: true})
	if calls != 0 || len(rec.commands) != 0 || results[0].Kind != ActionPrintCommand {
		t.Fatal("inactive migration performed work")
	}
	data, _ := os.ReadFile(paths[0])
	if string(data) != migrationKnownSource {
		t.Fatal("inactive migration changed settings")
	}
}

func TestMigrationFailureSkipsHostCommandsAndContinues(t *testing.T) {
	a, paths := migrationFixture(t, HostClaudeCode)
	writeMigrationFile(t, paths[1], "{")
	stubRuns(t, func(c Command) commandOutcome {
		if c.Name != "copilot" {
			t.Fatalf("failed migration ran %s", c.String())
		}
		return commandOutcome{Stdout: "updated"}
	})
	other := Plan(VerbUpdate, []Evidence{listedEvidence(HostCopilot)})[0]
	r := &recordingReporter{}
	results := Execute(t.Context(), []Action{a, other}, r, ExecuteOptions{Repository: canonicalRepoID})
	if len(results) != 2 || !results[0].Failed || results[0].Changed || results[1].Failed || !results[1].Changed {
		t.Fatalf("results=%+v", results)
	}
	if len(r.texts("note")) != 1 {
		t.Fatal("migration failure was not reported")
	}
}

func TestApplySourceEditsRefusesConcurrentChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	writeMigrationFile(t, path, "changed by user")
	err := applySourceEdits(t.Context(), []sourceEdit{{path: path, before: []byte("original"), after: []byte("migration")}})
	if err == nil {
		t.Fatal("concurrent edit overwritten")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "changed by user" {
		t.Fatal("user edit lost")
	}
}

func TestApplySourceEditsHonorsCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	writeMigrationFile(t, path, "original")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := applySourceEdits(ctx, []sourceEdit{{path: path, before: []byte("original"), after: []byte("migration")}}); err == nil {
		t.Fatal("cancelled migration succeeded")
	}
	data, _ := os.ReadFile(path)
	backups, _ := filepath.Glob(path + ".archcore-source-*.bak")
	if string(data) != "original" || len(backups) != 0 {
		t.Fatal("cancelled migration wrote to disk")
	}
}

func TestReadSourceFileRefusesOversizedSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	writeMigrationFile(t, path, strings.Repeat(" ", maxSourceConfigBytes+1))
	if _, _, err := readSourceFile(path); err == nil {
		t.Fatal("oversized settings accepted")
	}
}

func TestReadSourceFileRefusesDirectories(t *testing.T) {
	if _, _, err := readSourceFile(t.TempDir()); err == nil {
		t.Fatal("directory accepted as settings")
	}
}

func TestCodexMigrationPreservesEffectiveCustomSource(t *testing.T) {
	a, paths := migrationFixture(t, HostCodexCLI)
	stubRuns(t, func(Command) commandOutcome {
		return commandOutcome{Stdout: `{"marketplaces":[{"name":"archcore-plugins","marketplaceSource":{"sourceType":"git","source":"https://github.com/custom/fork.git"}}]}`}
	})
	calls := 0
	if err := migrateSource(t.Context(), a, canonicalRepoID, migrationClient(200, migrationCodexCatalog, &calls)); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(paths[0])
	if string(data) != migrationCodexSettings || calls != 0 {
		t.Fatal("effective custom source was migrated")
	}
}

func TestCopilotCanonicalRegistryIsDetected(t *testing.T) {
	for _, entry := range []string{"archcore-ai--plugin--plugins-archcore", "archcore-ai--archcore--plugins-archcore"} {
		t.Run(entry, func(t *testing.T) {
			home := isolateHome(t)
			writeRegistryEntry(t, home, ".copilot/installed-plugins/_direct/"+entry)
			spec, _ := SpecFor(HostCopilot)
			if !registryListsPlugin(spec) {
				t.Fatal("official direct installation missed")
			}
			actions := Plan(VerbUpdate, []Evidence{{Host: HostCopilot, RegistryListed: true}})
			if len(actions) != 1 || !slices.Equal(commandLines(actions[0].Commands), copilotUpdateLines) {
				t.Fatalf("plan=%+v", actions)
			}
		})
	}
}
