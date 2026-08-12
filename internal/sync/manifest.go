package sync

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"archcore-cli/internal/jsonfile"
)

// RelationType represents the kind of relationship between two documents.
type RelationType string

const (
	RelRelated    RelationType = "related"
	RelImplements RelationType = "implements"
	RelExtends    RelationType = "extends"
	RelDependsOn  RelationType = "depends_on"
)

var validRelationTypes = map[RelationType]bool{
	RelRelated: true, RelImplements: true, RelExtends: true, RelDependsOn: true,
}

// ValidRelationTypes returns all supported relation type strings.
func ValidRelationTypes() []string {
	return []string{string(RelRelated), string(RelImplements), string(RelExtends), string(RelDependsOn)}
}

// IsValidRelationType reports whether t is a recognised relation type.
func IsValidRelationType(t string) bool {
	return validRelationTypes[RelationType(t)]
}

// Relation represents a directed edge between two documents.
type Relation struct {
	Source string       `json:"source"`
	Target string       `json:"target"`
	Type   RelationType `json:"type"`
}

const manifestVersion = 1

const ManifestFile = ".sync-state.json"

// Manifest tracks per-file sync state. Keys are paths relative to the
// .archcore/ directory (e.g., "vision/my-doc.adr.md"), values are SHA-256 hashes.
type Manifest struct {
	Version   int               `json:"version"`
	Files     map[string]string `json:"files"`
	Relations []Relation        `json:"relations,omitempty"`
}

// NewManifest creates an empty manifest.
func NewManifest() *Manifest {
	return &Manifest{
		Version: manifestVersion,
		Files:   make(map[string]string),
	}
}

const maxManifestFiles = 10000

var sha256Re = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ValidateManifestJSON performs raw JSON checks on manifest bytes.
// Returns a list of issues (empty = valid).
func ValidateManifestJSON(data []byte) []string {
	var issues []string

	if len(data) == 0 {
		return []string{"manifest file is empty (0 bytes)"}
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return []string{fmt.Sprintf("invalid JSON: %v", err)}
	}

	// Unknown root fields are deliberately fail-closed (unlike settings.json's
	// tolerated Settings.Extra): the manifest is machine-owned state, and an
	// unrecognized field means a newer binary wrote it — rewriting the file
	// here would silently drop that field. Bump manifestVersion to migrate.
	allowedRoot := map[string]bool{"version": true, "files": true, "relations": true}
	for key := range raw {
		if !allowedRoot[key] {
			issues = append(issues, fmt.Sprintf("unknown root field %q", key))
		}
	}

	if vRaw, ok := raw["version"]; ok {
		if string(vRaw) == "null" {
			issues = append(issues, "version is null")
		} else {
			var v int
			if err := json.Unmarshal(vRaw, &v); err != nil {
				issues = append(issues, "version has wrong type (expected integer)")
			}
		}
	}

	if filesRaw, ok := raw["files"]; ok {
		var entries map[string]json.RawMessage
		if err := json.Unmarshal(filesRaw, &entries); err == nil {
			for path, val := range entries {
				if string(val) == "null" {
					issues = append(issues, fmt.Sprintf("null hash in file entry %q", path))
				}
			}
		}
	}

	if relRaw, ok := raw["relations"]; ok {
		var rels []json.RawMessage
		if err := json.Unmarshal(relRaw, &rels); err != nil {
			issues = append(issues, "relations must be an array")
		}
	}

	return issues
}

// ValidateManifest performs semantic checks on a parsed manifest.
// Returns a list of issues (empty = valid).
func ValidateManifest(m *Manifest) []string {
	var issues []string

	if m.Version != manifestVersion {
		issues = append(issues, fmt.Sprintf("unsupported version %d (expected %d)", m.Version, manifestVersion))
	}

	if len(m.Files) > maxManifestFiles {
		issues = append(issues, fmt.Sprintf("too many file entries (%d, max %d)", len(m.Files), maxManifestFiles))
	}

	for path, hash := range m.Files {
		issues = append(issues, validateFileEntry(path, hash)...)
	}

	issues = append(issues, validateRelations(m)...)

	return issues
}

// validateFileEntry checks a single file entry for path and hash issues.
func validateFileEntry(path string, hash string) []string {
	var issues []string

	// Path checks: reuse validateRelPath logic.
	if err := validateRelPath(path); err != nil {
		issues = append(issues, fmt.Sprintf("file %q: %v", path, err))
	}

	if strings.Contains(path, "//") {
		issues = append(issues, fmt.Sprintf("file %q: contains empty path segment (//)", path))
	}
	if strings.HasSuffix(path, "/") {
		issues = append(issues, fmt.Sprintf("file %q: has trailing slash", path))
	}

	// Hash checks.
	if hash == "" {
		issues = append(issues, fmt.Sprintf("file %q: hash is empty", path))
	} else if !sha256Re.MatchString(hash) {
		issues = append(issues, fmt.Sprintf("file %q: hash is not valid SHA-256 (expected 64 lowercase hex chars)", path))
	}

	return issues
}

func manifestPath(baseDir string) string {
	return filepath.Join(baseDir, ".archcore", ManifestFile)
}

// LoadManifest reads the manifest from disk. If the file does not exist,
// it returns a fresh empty manifest (first sync scenario).
func LoadManifest(baseDir string) (*Manifest, error) {
	data, err := os.ReadFile(manifestPath(baseDir))
	if errors.Is(err, fs.ErrNotExist) {
		return NewManifest(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	if jsonIssues := ValidateManifestJSON(data); len(jsonIssues) > 0 {
		return nil, fmt.Errorf("invalid manifest: %s", strings.Join(jsonIssues, "; "))
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid manifest JSON: %w", err)
	}
	if m.Files == nil {
		m.Files = make(map[string]string)
	}

	if semIssues := ValidateManifest(&m); len(semIssues) > 0 {
		return nil, fmt.Errorf("invalid manifest: %s", strings.Join(semIssues, "; "))
	}

	return &m, nil
}

const maxManifestRelations = 50000

func validateRelations(m *Manifest) []string {
	var issues []string
	if len(m.Relations) > maxManifestRelations {
		issues = append(issues, fmt.Sprintf("too many relations (%d, max %d)", len(m.Relations), maxManifestRelations))
	}
	seen := make(map[string]bool)
	for i, rel := range m.Relations {
		prefix := fmt.Sprintf("relation[%d]", i)
		if !validRelationTypes[rel.Type] {
			issues = append(issues, fmt.Sprintf("%s: invalid type %q", prefix, rel.Type))
		}
		if err := validateRelPath(rel.Source); err != nil {
			issues = append(issues, fmt.Sprintf("%s: source %v", prefix, err))
		}
		if err := validateRelPath(rel.Target); err != nil {
			issues = append(issues, fmt.Sprintf("%s: target %v", prefix, err))
		}
		if rel.Source == rel.Target {
			issues = append(issues, fmt.Sprintf("%s: source and target are the same (%q)", prefix, rel.Source))
		}
		key := rel.Source + "|" + rel.Target + "|" + string(rel.Type)
		if seen[key] {
			issues = append(issues, fmt.Sprintf("%s: duplicate relation", prefix))
		}
		seen[key] = true
	}
	return issues
}

// Clone returns a deep copy of the manifest. The MCP server mutates a clone
// under its store lock and swaps it in, so readers holding the previous
// snapshot never observe concurrent modification.
func (m *Manifest) Clone() *Manifest {
	out := &Manifest{Version: m.Version, Files: make(map[string]string, len(m.Files))}
	for k, v := range m.Files {
		out.Files[k] = v
	}
	out.Relations = slices.Clone(m.Relations)
	return out
}

// AddRelation appends a relation if it does not already exist. Returns true if added.
func (m *Manifest) AddRelation(source, target string, relType RelationType) bool {
	for _, r := range m.Relations {
		if r.Source == source && r.Target == target && r.Type == relType {
			return false
		}
	}
	m.Relations = append(m.Relations, Relation{Source: source, Target: target, Type: relType})
	return true
}

// RemoveRelation removes a relation matching source, target, and type. Returns true if removed.
func (m *Manifest) RemoveRelation(source, target string, relType RelationType) bool {
	for i, r := range m.Relations {
		if r.Source == source && r.Target == target && r.Type == relType {
			m.Relations = append(m.Relations[:i], m.Relations[i+1:]...)
			return true
		}
	}
	return false
}

// RelationsFor returns all outgoing and incoming relations for the given path.
func (m *Manifest) RelationsFor(path string) (outgoing, incoming []Relation) {
	for _, r := range m.Relations {
		if r.Source == path {
			outgoing = append(outgoing, r)
		}
		if r.Target == path {
			incoming = append(incoming, r)
		}
	}
	return
}

// CleanupRelations removes relations where the source or target file does not
// exist on disk. Paths in relations are relative to archcoreDir. Returns the
// number of removed relations.
func (m *Manifest) CleanupRelations(archcoreDir string) int {
	kept := m.Relations[:0]
	for _, rel := range m.Relations {
		srcPath := filepath.Join(archcoreDir, rel.Source)
		tgtPath := filepath.Join(archcoreDir, rel.Target)
		srcExists := true
		tgtExists := true
		if _, err := os.Stat(srcPath); errors.Is(err, fs.ErrNotExist) {
			srcExists = false
		}
		if _, err := os.Stat(tgtPath); errors.Is(err, fs.ErrNotExist) {
			tgtExists = false
		}
		if srcExists && tgtExists {
			kept = append(kept, rel)
		}
	}
	removed := len(m.Relations) - len(kept)
	m.Relations = kept
	return removed
}

// SaveManifest writes the manifest to disk atomically (write temp + rename).
//
// The write itself is jsonfile.WriteAtomic rather than a local copy of it. The
// manifest is archcore-owned state, which is the case that helper serves; the
// copy that used to live here was identical to it byte for byte, and two
// identical implementations of a crash-safety primitive are one place for a fix
// to land and one place for it to be missed.
func SaveManifest(baseDir string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	data = append(data, '\n')

	if err := jsonfile.WriteAtomic(manifestPath(baseDir), data); err != nil {
		return fmt.Errorf("saving manifest: %w", err)
	}
	return nil
}
