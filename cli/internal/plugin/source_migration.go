package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/jsonfile"
)

// These ceilings bound user config reads and remote marketplace validation on
// the manual update path — plugin-source-migration.spec.
const (
	maxSourceConfigBytes     = 1 << 20
	maxMigrationCatalogBytes = 64 << 10
	migrationProbeTimeout    = 10 * time.Second
)

type sourceEdit struct {
	path   string
	before []byte
	after  []byte
}

func migrateSource(ctx context.Context, a Action, repository string, client *http.Client) error {
	if repository != canonicalRepoID || (a.Host != HostClaudeCode && a.Host != HostCodexCLI) {
		return nil
	}
	edits, err := collectSourceEdits(ctx, a)
	if err != nil {
		// Guard: an unreadable source never authorizes rewriting another file.
		return err
	}
	if len(edits) == 0 {
		return nil
	}
	if err := verifyMigrationCatalog(ctx, a.Host, client); err != nil {
		// Guard: retain the working legacy source until the target is published.
		return err
	}
	return applySourceEdits(ctx, edits)
}

func collectSourceEdits(ctx context.Context, a Action) ([]sourceEdit, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolving the marketplace configuration home: %w", err)
	}
	if a.Host == HostCodexCLI {
		if !codexSourceIsMigratable(ctx) {
			return nil, nil
		}
		root := hostConfigRoot("CODEX_HOME", filepath.Join(home, ".codex"))
		path, data, err := readSourceFile(filepath.Join(root, "config.toml"))
		if err != nil || data == nil {
			return nil, err
		}
		after := rewriteCodexSource(data)
		if after == nil {
			return nil, nil
		}
		return []sourceEdit{{path: path, before: data, after: after}}, nil
	}

	root := hostConfigRoot("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
	known := filepath.Join(root, "plugins", "known_marketplaces.json")
	paths := []string{known, filepath.Join(root, "settings.json")}
	for _, install := range orderInstalls(a.Evidence.Installs) {
		if (install.Scope == ScopeProject || install.Scope == ScopeLocal) && install.ProjectPresent {
			paths = append(paths, filepath.Join(install.ProjectPath, ".claude", "settings.json"),
				filepath.Join(install.ProjectPath, ".claude", "settings.local.json"))
		}
	}
	slices.Sort(paths)
	paths = slices.Compact(paths)
	// Refuse rather than migrate only a prefix of settings that share one origin.
	if len(paths) > 2+2*maxAddressedInstalls {
		return nil, errors.New("too many Claude Code source declarations to migrate")
	}
	edits := make([]sourceEdit, 0, len(paths))
	seen := make(map[string]bool, len(paths))
	for _, path := range paths {
		resolved, data, err := readSourceFile(path)
		if err != nil {
			return nil, err
		}
		if data == nil {
			if path == known {
				return nil, nil
			}
			continue
		}
		after, blocked, err := rewriteClaudeSource(data, path != known)
		if err != nil {
			return nil, err
		}
		if blocked {
			return nil, nil
		}
		if after != nil {
			if !seen[resolved] {
				edits = append(edits, sourceEdit{path: resolved, before: data, after: after})
				seen[resolved] = true
			}
		}
	}
	return edits, nil
}

func codexSourceIsMigratable(ctx context.Context) bool {
	out := runCommand(ctx, Command{Name: "codex", Args: []string{"plugin", "marketplace", "list", "--json"}})
	// Guard: config layers can override the user file; the host must confirm
	// the effective unpinned official source before that file can be migrated.
	if out.Failed || out.Truncated {
		return false
	}
	var listing struct {
		Marketplaces []struct {
			Name   string                     `json:"name"`
			Source map[string]json.RawMessage `json:"marketplaceSource"`
		} `json:"marketplaces"`
	}
	if json.Unmarshal([]byte(out.Stdout), &listing) != nil {
		return false
	}
	found := false
	for _, entry := range listing.Marketplaces {
		if entry.Name != MarketplaceID {
			continue
		}
		if found || len(entry.Source) != 2 {
			return false
		}
		var kind, source string
		if json.Unmarshal(entry.Source["sourceType"], &kind) != nil || kind != "git" ||
			json.Unmarshal(entry.Source["source"], &source) != nil {
			return false
		}
		if _, ok := canonicalSource(source); !ok {
			return false
		}
		found = true
	}
	return found
}

func hostConfigRoot(key, fallback string) string {
	if root := os.Getenv(key); root != "" {
		return root
	}
	return fallback
}

func readSourceFile(path string) (string, []byte, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if errors.Is(err, fs.ErrNotExist) {
		if _, statErr := os.Lstat(path); errors.Is(statErr, fs.ErrNotExist) {
			return "", nil, nil
		}
	}
	if err != nil {
		return "", nil, fmt.Errorf("resolving marketplace settings: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", nil, fmt.Errorf("inspecting marketplace settings: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", nil, errors.New("marketplace settings are not a regular file")
	}
	f, err := os.Open(resolved)
	if err != nil {
		return "", nil, fmt.Errorf("reading marketplace settings: %w", err)
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxSourceConfigBytes+1))
	if err != nil {
		return "", nil, fmt.Errorf("reading marketplace settings: %w", err)
	}
	if len(data) > maxSourceConfigBytes {
		return "", nil, errors.New("marketplace settings exceed the migration size limit")
	}
	return resolved, data, nil
}

func decodeSourceObject(data []byte) (*jsonfile.Doc, error) {
	if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) {
		return nil, errors.New("marketplace settings are not a JSON object")
	}
	doc := jsonfile.NewDoc()
	if err := json.Unmarshal(data, doc); err != nil {
		return nil, fmt.Errorf("parsing marketplace settings: %w", err)
	}
	return doc, nil
}

func rewriteClaudeSource(data []byte, settings bool) ([]byte, bool, error) {
	doc, err := decodeSourceObject(data)
	if err != nil {
		return nil, false, err
	}
	marketplaces := doc
	if settings {
		raw, ok := doc.Get(extraKnownMarketplacesKey)
		if !ok {
			return nil, false, nil
		}
		marketplaces, err = decodeSourceObject(raw)
		if err != nil {
			return nil, false, err
		}
	}
	raw, ok := marketplaces.Get(MarketplaceID)
	if !ok {
		return nil, !settings, nil
	}
	entry, err := decodeSourceObject(raw)
	if err != nil {
		return nil, false, err
	}
	raw, ok = entry.Get(marketplaceSourceKey)
	if !ok {
		return nil, true, nil
	}
	source, err := decodeSourceObject(raw)
	if err != nil {
		return nil, false, err
	}
	// A ref, header, sparse path or unknown source field can change what is
	// fetched. Preserve that declaration instead of guessing its semantics.
	if source.Len() != 2 {
		return nil, true, nil
	}
	kind := sourceString(source, "source")
	key := "repo"
	if kind == "git" {
		key = "url"
	} else if kind != "github" {
		return nil, true, nil
	}
	old := sourceString(source, key)
	next, recognized := canonicalSource(old)
	if !recognized {
		return nil, true, nil
	}
	if old == next {
		return nil, false, nil
	}
	encoded, err := json.Marshal(next)
	if err != nil {
		return nil, false, fmt.Errorf("encoding marketplace source: %w", err)
	}
	source.Set(key, encoded)
	if err := setSourceObject(entry, marketplaceSourceKey, source); err != nil {
		return nil, false, err
	}
	if err := setSourceObject(marketplaces, MarketplaceID, entry); err != nil {
		return nil, false, err
	}
	if settings {
		if err := setSourceObject(doc, extraKnownMarketplacesKey, marketplaces); err != nil {
			return nil, false, err
		}
	}
	after, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, false, fmt.Errorf("encoding marketplace settings: %w", err)
	}
	return append(after, '\n'), false, nil
}

func sourceString(doc *jsonfile.Doc, key string) string {
	raw, _ := doc.Get(key)
	var value string
	_ = json.Unmarshal(raw, &value)
	return value
}

func setSourceObject(doc *jsonfile.Doc, key string, value *jsonfile.Doc) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encoding marketplace object: %w", err)
	}
	doc.Set(key, raw)
	return nil
}

func canonicalSource(source string) (string, bool) {
	for _, prefix := range []string{"", "https://github.com/", "ssh://git@github.com/", "git@github.com:"} {
		for _, suffix := range []string{"", ".git"} {
			if prefix == "" && suffix != "" {
				continue
			}
			if source == prefix+legacyRepoID+suffix || source == prefix+canonicalRepoID+suffix {
				return prefix + canonicalRepoID + suffix, true
			}
		}
	}
	return "", false
}

var codexSourceTableRe = regexp.MustCompile(`^\s*\[marketplaces\.(?:archcore-plugins|"archcore-plugins"|'archcore-plugins')\]\s*(?:#.*)?$`)
var codexSourceFieldRe = regexp.MustCompile(`^\s*(source_type|source|last_updated|last_revision)\s*=\s*("[^"\\]*"|'[^']*')\s*(?:#.*)?$`)

func rewriteCodexSource(data []byte) []byte {
	text := string(data)
	// This edits only the native table: source_type and source, plus the refresh
	// fields last_updated and last_revision, which pin no revision —
	// plugin-source-migration.spec. Inline tables and multiline strings stay
	// untouched; a partial TOML parser would misread them.
	if strings.Contains(text, `"""`) || strings.Contains(text, "'''") {
		return nil
	}
	inside, found := false, false
	fields := make(map[string]string, 2)
	start, end, offset := 0, 0, 0
	for line := range strings.Lines(text) {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			inside = codexSourceTableRe.MatchString(trimmed)
			if !inside && (strings.HasPrefix(trimmed, "[marketplaces.archcore-plugins") ||
				strings.HasPrefix(trimmed, `[marketplaces."archcore-plugins"`) ||
				strings.HasPrefix(trimmed, "[marketplaces.'archcore-plugins'")) {
				return nil
			}
			if inside {
				if found {
					return nil
				}
				found = true
			}
		} else if inside && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			match := codexSourceFieldRe.FindStringSubmatchIndex(strings.TrimSuffix(line, "\n"))
			if match == nil {
				return nil
			}
			key := line[match[2]:match[3]]
			if _, duplicate := fields[key]; duplicate {
				return nil
			}
			fields[key] = line[match[4]+1 : match[5]-1]
			if key == "source" {
				start, end = offset+match[4]+1, offset+match[5]-1
			}
		}
		offset += len(line)
	}
	if !found || fields["source_type"] != "git" {
		return nil
	}
	next, ok := canonicalSource(fields["source"])
	if !ok || next == fields["source"] {
		return nil
	}
	return []byte(text[:start] + next + text[end:])
}

func verifyMigrationCatalog(ctx context.Context, host Host, client *http.Client) error {
	path := ".claude-plugin/marketplace.json"
	if host == HostCodexCLI {
		path = ".agents/plugins/marketplace.json"
	}
	ctx, cancel := context.WithTimeout(ctx, migrationProbeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://raw.githubusercontent.com/"+canonicalRepoID+"/main/"+path, nil)
	if err != nil {
		return fmt.Errorf("creating the marketplace probe: %w", err)
	}
	if client == nil {
		client = &http.Client{Timeout: migrationProbeTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("checking the canonical marketplace: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checking the canonical marketplace: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxMigrationCatalogBytes+1))
	if err != nil || len(data) > maxMigrationCatalogBytes {
		return errors.New("reading the canonical marketplace catalog failed")
	}
	var catalog struct {
		Name    string `json:"name"`
		Plugins []struct {
			Name   string          `json:"name"`
			Source json.RawMessage `json:"source"`
		} `json:"plugins"`
	}
	if json.Unmarshal(data, &catalog) != nil || catalog.Name != MarketplaceID {
		return errors.New("canonical marketplace identity does not match")
	}
	for _, entry := range catalog.Plugins {
		if entry.Name != pluginName {
			continue
		}
		var source string
		if host == HostClaudeCode && json.Unmarshal(entry.Source, &source) == nil && source == "./plugins/archcore" {
			return nil
		}
		var local struct {
			Source string `json:"source"`
			Path   string `json:"path"`
		}
		if host == HostCodexCLI && json.Unmarshal(entry.Source, &local) == nil && local.Source == "local" && local.Path == "./plugins/archcore" {
			return nil
		}
	}
	return errors.New("canonical marketplace does not contain the expected plugin path")
}

func applySourceEdits(ctx context.Context, edits []sourceEdit) error {
	for _, edit := range edits {
		_, current, err := readSourceFile(edit.path)
		if err != nil || !bytes.Equal(current, edit.before) {
			// Guard: never overwrite a declaration edited since collection.
			return errors.New("marketplace settings changed before migration")
		}
	}
	for _, edit := range edits {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, current, err := readSourceFile(edit.path)
		if err != nil || !bytes.Equal(current, edit.before) {
			// Guard: retain concurrent edits; a later run completes remaining files.
			return errors.New("marketplace settings changed during migration")
		}
		backup, err := os.CreateTemp(filepath.Dir(edit.path), filepath.Base(edit.path)+".archcore-source-*.bak")
		if err != nil {
			return fmt.Errorf("reserving the marketplace settings backup: %w", err)
		}
		if err := backup.Close(); err != nil {
			return fmt.Errorf("closing the marketplace settings backup: %w", err)
		}
		if err := agents.WriteConfigFile(backup.Name(), edit.before); err != nil {
			return fmt.Errorf("backing up marketplace settings: %w", err)
		}
		if err := agents.WriteConfigFile(edit.path, edit.after); err != nil {
			return fmt.Errorf("writing migrated marketplace settings: %w", err)
		}
	}
	return nil
}
