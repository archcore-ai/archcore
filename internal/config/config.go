package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"archcore-cli/internal/jsonfile"
)

const (
	dirName  = ".archcore"
	fileName = "settings.json"
)

// globalIDRe validates a global source id: lowercase alphanumeric with hyphens.
var globalIDRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// GlobalSource declares a read-only external knowledge base mounted into the
// local project for read operations.
type GlobalSource struct {
	ID string `json:"id"`
	// Path points at the global source's .archcore directory, e.g.
	// "../company-global/.archcore". It may be relative (including "../" for
	// sibling or parent directories) or absolute.
	//
	// Every declared global source is mandatory: if its directory is absent the
	// MCP server fails fast rather than running against an incomplete context.
	Path string `json:"path"`
}

// SyncType identifies the sync mode of a project.
type SyncType string

const (
	SyncTypeNone   SyncType = "none"
	SyncTypeCloud  SyncType = "cloud"
	SyncTypeOnPrem SyncType = "on-prem"
)

// ValidSyncTypeStrings returns all valid sync types as plain strings.
// Useful for assembling MCP enum schemas and error messages with strings.Join.
func ValidSyncTypeStrings() []string {
	return []string{string(SyncTypeNone), string(SyncTypeCloud), string(SyncTypeOnPrem)}
}

// CloudServerURL is the hardcoded URL for cloud sync. Var for test override.
var CloudServerURL = "https://app.archcore.ai"

// CodeAlignment tunes the pre-write context injection.
type CodeAlignment struct {
	// SourceRoots are the top-level directories treated as source code. A file
	// outside all of them gets no injection. Empty means the built-in defaults.
	SourceRoots []string `json:"sourceRoots,omitempty"`

	// Extra preserves keys a newer archcore wrote here. Settings guards the
	// top level the same way, but codeAlignment is a known key and is decoded
	// into this struct, so without its own escape hatch anything nested that
	// this binary does not recognize is dropped on the next write.
	Extra map[string]json.RawMessage `json:"-"`
}

// caKnownFields are the codeAlignment keys this binary understands; everything
// else is captured into Extra.
var caKnownFields = map[string]bool{"sourceRoots": true}

func (c CodeAlignment) MarshalJSON() ([]byte, error) {
	known, err := json.Marshal(struct {
		SourceRoots []string `json:"sourceRoots,omitempty"`
	}{SourceRoots: c.SourceRoots})
	if err != nil {
		return nil, err
	}
	if len(c.Extra) == 0 {
		return known, nil
	}
	var merged map[string]json.RawMessage
	if err := json.Unmarshal(known, &merged); err != nil {
		return nil, err
	}
	maps.Copy(merged, c.Extra)
	return json.Marshal(merged)
}

func (c *CodeAlignment) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("field \"codeAlignment\" must be an object: %w", err)
	}
	if rootsRaw, ok := raw["sourceRoots"]; ok {
		if err := json.Unmarshal(rootsRaw, &c.SourceRoots); err != nil {
			return fmt.Errorf("field \"codeAlignment.sourceRoots\" must be an array of strings: %w", err)
		}
	}
	for i, root := range c.SourceRoots {
		if root == "" {
			return errors.New("field \"codeAlignment.sourceRoots\" must not contain an empty entry")
		}
		// IsLocal rejects absolute paths, traversal, and reserved names in one
		// predicate. A plain Contains("..") also rejected "foo..bar", which is
		// an ordinary directory name.
		if !filepath.IsLocal(root) {
			return fmt.Errorf("field \"codeAlignment.sourceRoots\" entry %q must be a relative path inside the project", root)
		}
		// Normalize here, once, so consumers can stay a plain prefix test.
		// IsLocal accepts "./src" and (on Windows) "src\api", but document paths
		// are slash-separated and unprefixed — those roots validated cleanly and
		// then matched nothing, silently disabling the advisory for a config
		// that looked correct.
		cleaned := strings.Trim(path.Clean(filepath.ToSlash(root)), "/")
		if cleaned == "" || cleaned == "." {
			return fmt.Errorf("field \"codeAlignment.sourceRoots\" entry %q must name a directory inside the project", root)
		}
		c.SourceRoots[i] = cleaned
	}
	for k, v := range raw {
		if caKnownFields[k] {
			continue
		}
		if c.Extra == nil {
			c.Extra = map[string]json.RawMessage{}
		}
		c.Extra[k] = v
	}
	return nil
}

type Settings struct {
	Sync          SyncType       `json:"sync"`
	ProjectID     *int           `json:"project_id,omitempty"`
	ArchcoreURL   string         `json:"archcore_url,omitempty"`
	Language      string         `json:"language,omitempty"`
	Globals       []GlobalSource `json:"globals,omitempty"`
	CodeAlignment *CodeAlignment `json:"codeAlignment,omitempty"`

	// Extra holds fields present in settings.json that this binary does not
	// recognize — typically a field added by a newer archcore version. They are
	// tolerated on read (forward compatibility) and preserved verbatim on write
	// so an older binary never silently drops a newer config field. The custom
	// (Un)MarshalJSON methods own serialization; the json:"-" tag only documents
	// that Extra is not a normal field and guards against accidental exposure if
	// those methods are ever removed.
	Extra map[string]json.RawMessage `json:"-"`
}

// NewNoneSettings creates settings with sync disabled.
func NewNoneSettings() *Settings {
	return &Settings{Sync: SyncTypeNone}
}

// NewCloudSettings creates settings for cloud sync.
func NewCloudSettings() *Settings {
	return &Settings{Sync: SyncTypeCloud}
}

// NewOnPremSettings creates settings for on-prem sync with the given URL.
func NewOnPremSettings(url string) *Settings {
	return &Settings{Sync: SyncTypeOnPrem, ArchcoreURL: url}
}

// Validate checks that the settings are internally consistent.
func (s *Settings) Validate() error {
	switch s.Sync {
	case SyncTypeNone:
		if s.ProjectID != nil {
			return fmt.Errorf("sync %q does not allow project_id", SyncTypeNone)
		}
		if s.ArchcoreURL != "" {
			return fmt.Errorf("sync %q does not allow archcore_url", SyncTypeNone)
		}
	case SyncTypeCloud:
		if s.ArchcoreURL != "" {
			return fmt.Errorf("sync %q does not allow archcore_url", SyncTypeCloud)
		}
	case SyncTypeOnPrem:
		if s.ArchcoreURL == "" {
			return fmt.Errorf("sync %q requires archcore_url", SyncTypeOnPrem)
		}
	default:
		return fmt.Errorf("unknown sync type %q", s.Sync)
	}
	if s.Language != "" && strings.Contains(s.Language, " ") {
		return errors.New("language must not contain spaces")
	}
	seen := make(map[string]int, len(s.Globals))
	for i, g := range s.Globals {
		if g.ID == "" {
			return fmt.Errorf("globals[%d]: \"id\" must not be empty", i)
		}
		if !globalIDRe.MatchString(g.ID) {
			return fmt.Errorf("globals[%d]: \"id\" %q must be lowercase alphanumeric with hyphens (e.g. \"company\")", i, g.ID)
		}
		if g.ID == "local" {
			return fmt.Errorf("globals[%d]: \"id\" %q is reserved", i, g.ID)
		}
		if g.Path == "" {
			return fmt.Errorf("globals[%d]: \"path\" must not be empty", i)
		}
		if j, dup := seen[g.ID]; dup {
			return fmt.Errorf("globals[%d] and globals[%d]: duplicate id %q", j, i, g.ID)
		}
		seen[g.ID] = i
	}
	return nil
}

// ServerURL returns the server URL for the current sync type.
func (s *Settings) ServerURL() string {
	switch s.Sync {
	case SyncTypeCloud:
		return CloudServerURL
	case SyncTypeOnPrem:
		return s.ArchcoreURL
	default:
		return ""
	}
}

// allowedFields defines which JSON fields are valid per sync type (besides "sync" itself).
// "globals" is always allowed regardless of sync mode.
var allowedFields = map[SyncType]map[string]bool{
	SyncTypeNone:   {"language": true, "globals": true, "codeAlignment": true},
	SyncTypeCloud:  {"project_id": true, "language": true, "globals": true, "codeAlignment": true},
	SyncTypeOnPrem: {"project_id": true, "archcore_url": true, "language": true, "globals": true, "codeAlignment": true},
}

// requiredFields defines which JSON fields must be present per sync type.
var requiredFields = map[SyncType][]string{
	SyncTypeNone:   {},
	SyncTypeCloud:  {},
	SyncTypeOnPrem: {"archcore_url"},
}

// knownFields is every JSON field this binary recognizes: "sync" plus the union
// of allowedFields across all sync types. A field outside this set is unknown to
// this binary (e.g. added by a newer archcore version) and is tolerated and
// preserved rather than rejected. Derived from allowedFields so it extends
// automatically when a new field is added there.
var knownFields = func() map[string]bool {
	known := map[string]bool{"sync": true}
	for _, fields := range allowedFields {
		for f := range fields {
			known[f] = true
		}
	}
	return known
}()

func (s Settings) MarshalJSON() ([]byte, error) {
	var known []byte
	var err error
	switch s.Sync {
	// Every optional field must appear in all three shapes below. A field added
	// to only one is silently dropped when the project switches sync mode.
	case SyncTypeNone:
		known, err = json.Marshal(struct {
			Sync          SyncType       `json:"sync"`
			Language      string         `json:"language,omitempty"`
			Globals       []GlobalSource `json:"globals,omitempty"`
			CodeAlignment *CodeAlignment `json:"codeAlignment,omitempty"`
		}{Sync: s.Sync, Language: s.Language, Globals: s.Globals, CodeAlignment: s.CodeAlignment})

	case SyncTypeCloud:
		known, err = json.Marshal(struct {
			Sync          SyncType       `json:"sync"`
			ProjectID     *int           `json:"project_id,omitempty"`
			Language      string         `json:"language,omitempty"`
			Globals       []GlobalSource `json:"globals,omitempty"`
			CodeAlignment *CodeAlignment `json:"codeAlignment,omitempty"`
		}{Sync: s.Sync, ProjectID: s.ProjectID, Language: s.Language, Globals: s.Globals, CodeAlignment: s.CodeAlignment})

	case SyncTypeOnPrem:
		known, err = json.Marshal(struct {
			Sync          SyncType       `json:"sync"`
			ProjectID     *int           `json:"project_id,omitempty"`
			ArchcoreURL   string         `json:"archcore_url"`
			Language      string         `json:"language,omitempty"`
			Globals       []GlobalSource `json:"globals,omitempty"`
			CodeAlignment *CodeAlignment `json:"codeAlignment,omitempty"`
		}{Sync: s.Sync, ProjectID: s.ProjectID, ArchcoreURL: s.ArchcoreURL, Language: s.Language, Globals: s.Globals, CodeAlignment: s.CodeAlignment})

	default:
		return nil, fmt.Errorf("unknown sync type %q", s.Sync)
	}
	if err != nil {
		return nil, err
	}

	// No unknown fields → emit the known fields verbatim (byte-identical to the
	// pre-forward-compat output, so existing settings.json files don't churn).
	if len(s.Extra) == 0 {
		return known, nil
	}

	// Merge captured unknown fields back in. Extra keys are disjoint from the
	// known fields by construction (a field lands in Extra only if it is not in
	// knownFields), so there is never a collision. The result re-sorts object
	// keys alphabetically, which only affects configs that carry unknown fields.
	var merged map[string]json.RawMessage
	if err := json.Unmarshal(known, &merged); err != nil {
		return nil, err
	}
	maps.Copy(merged, s.Extra)
	return json.Marshal(merged)
}

func (s *Settings) UnmarshalJSON(data []byte) error {
	// Decode into a raw map to check fields.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Extract and validate sync field.
	syncRaw, ok := raw["sync"]
	if !ok {
		return errors.New("missing required field \"sync\"")
	}
	var syncRawString string
	if err := json.Unmarshal(syncRaw, &syncRawString); err != nil {
		return errors.New("field \"sync\" must be a string")
	}
	syncType := SyncType(syncRawString)

	allowed, knownType := allowedFields[syncType]
	if !knownType {
		return fmt.Errorf("unknown sync type %q", syncType)
	}

	// Classify every non-sync field:
	//   - allowed for this sync mode      → decoded below.
	//   - known to this binary, wrong mode → hard error (misconfiguration).
	//   - unknown to this binary           → tolerated and captured into Extra so
	//     an older binary does not crash on, or silently drop, a newer field.
	for key := range raw {
		if key == "sync" {
			continue
		}
		if allowed[key] {
			continue
		}
		if knownFields[key] {
			return fmt.Errorf("field %q is not allowed for sync type %q", key, syncType)
		}
		if s.Extra == nil {
			s.Extra = make(map[string]json.RawMessage)
		}
		s.Extra[key] = raw[key]
	}

	// Check for required fields.
	for _, req := range requiredFields[syncType] {
		if _, ok := raw[req]; !ok {
			return fmt.Errorf("missing required field %q for sync type %q", req, syncType)
		}
	}

	s.Sync = syncType

	// Decode project_id if present.
	if pidRaw, ok := raw["project_id"]; ok {
		// Accept null or number.
		if string(pidRaw) == "null" {
			s.ProjectID = nil
		} else {
			var pid int
			if err := json.Unmarshal(pidRaw, &pid); err != nil {
				return errors.New("field \"project_id\" must be null or a number")
			}
			s.ProjectID = &pid
		}
	}

	// Decode archcore_url if present.
	if urlRaw, ok := raw["archcore_url"]; ok {
		var url string
		if err := json.Unmarshal(urlRaw, &url); err != nil {
			return errors.New("field \"archcore_url\" must be a string")
		}
		if url == "" {
			return errors.New("field \"archcore_url\" must not be empty")
		}
		s.ArchcoreURL = url
	}

	// Decode language if present.
	if langRaw, ok := raw["language"]; ok {
		var lang string
		if err := json.Unmarshal(langRaw, &lang); err != nil {
			return errors.New("field \"language\" must be a string")
		}
		if lang == "" {
			return errors.New("field \"language\" must not be empty")
		}
		if strings.Contains(lang, " ") {
			return errors.New("field \"language\" must not contain spaces")
		}
		s.Language = lang
	}

	// Decode globals if present — always allowed regardless of sync mode.
	if globalsRaw, ok := raw["globals"]; ok {
		var globals []GlobalSource
		if err := json.Unmarshal(globalsRaw, &globals); err != nil {
			return errors.New("field \"globals\" must be an array of global source objects")
		}
		s.Globals = globals
	}

	// Decode codeAlignment if present. Explicit null leaves the section unset
	// rather than producing an empty object the next write would persist, the
	// same treatment project_id gets above.
	if caRaw, ok := raw["codeAlignment"]; ok && string(caRaw) != "null" {
		var ca CodeAlignment
		if err := json.Unmarshal(caRaw, &ca); err != nil {
			return err
		}
		s.CodeAlignment = &ca
	}

	return nil
}

// UnknownFieldNames returns the sorted names of settings.json fields this binary
// does not recognize (captured into Extra on load). Empty when the config is
// fully understood. Entry-point commands use it to warn the user that their
// archcore may be older than the project's config.
func (s *Settings) UnknownFieldNames() []string {
	if len(s.Extra) == 0 {
		return nil
	}
	names := make([]string, 0, len(s.Extra))
	for k := range s.Extra {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func settingsPath(baseDir string) string {
	return filepath.Join(baseDir, dirName, fileName)
}

func Load(baseDir string) (*Settings, error) {
	data, err := os.ReadFile(settingsPath(baseDir))
	if err != nil {
		return nil, err
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("invalid settings: %w", err)
	}
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("invalid settings: %w", err)
	}
	return &s, nil
}

func Save(baseDir string, s *Settings) error {
	if err := s.Validate(); err != nil {
		return fmt.Errorf("refusing to save invalid settings: %w", err)
	}
	dir := filepath.Join(baseDir, dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	// Atomic write: a crash mid-write must never leave a truncated
	// settings.json — every command fails to Load one.
	return jsonfile.WriteAtomic(settingsPath(baseDir), data)
}

func InitDir(baseDir string) error {
	return os.MkdirAll(filepath.Join(baseDir, dirName), 0o755)
}

func DirExists(baseDir string) bool {
	info, err := os.Stat(filepath.Join(baseDir, dirName))
	return err == nil && info.IsDir()
}

// ReadGlobals returns the declared global sources for baseDir.
// Returns nil if settings cannot be loaded (missing file, parse error, etc.).
// Use this on read paths where a degraded "no globals" view is acceptable; write
// guards that must fail closed should use LoadGlobals instead.
func ReadGlobals(baseDir string) []GlobalSource {
	globals, _ := LoadGlobals(baseDir)
	return globals
}

// LoadGlobals returns the declared global sources for baseDir. A missing
// settings.json is not an error — it yields no globals. A present-but-invalid
// settings.json returns the parse/validation error so callers that must fail
// closed (e.g. write guards protecting read-only sources) can reject the
// operation rather than silently treating it as "no globals".
func LoadGlobals(baseDir string) ([]GlobalSource, error) {
	s, err := Load(baseDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return s.Globals, nil
}
