// Package jsonfile provides order- and content-preserving surgery on JSON
// config files that archcore does not own (agent settings, MCP configs).
// Values are held as opaque json.RawMessage inside an ordered map, so a
// rewrite never strips unknown fields or reorders the user's keys; writes are
// atomic (tmp + rename) so a crash can never truncate a live config file.
package jsonfile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// Doc is an order-preserving JSON object whose values stay opaque. Only keys
// the caller explicitly Sets are ever re-encoded.
type Doc = orderedmap.OrderedMap[string, json.RawMessage]

// NewDoc returns an empty Doc.
func NewDoc() *Doc {
	return orderedmap.New[string, json.RawMessage]()
}

// Read parses the JSON object at path preserving key order. A missing file
// yields an empty Doc and no error; a read failure or invalid JSON is an error.
func Read(path string) (*Doc, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return NewDoc(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	doc := NewDoc()
	if err := json.Unmarshal(data, doc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return doc, nil
}

// ReadOrBackup is Read plus the corrupt-file policy from
// backup-invalid-configs.adr: when the file exists but is not a valid JSON
// object, the original bytes are written atomically to path+".bak" and an
// empty Doc is returned with backedUp=true (the caller prints the warning).
// If the backup write fails, an error is returned and the caller MUST abort —
// never proceed to overwrite unrecoverable user data.
func ReadOrBackup(path string) (doc *Doc, backedUp bool, err error) {
	data, readErr := os.ReadFile(path)
	if errors.Is(readErr, fs.ErrNotExist) {
		return NewDoc(), false, nil
	}
	if readErr != nil {
		return nil, false, fmt.Errorf("reading %s: %w", path, readErr)
	}
	doc = NewDoc()
	if json.Unmarshal(data, doc) == nil {
		return doc, false, nil
	}
	if err := WriteAtomic(path+".bak", data); err != nil {
		return nil, false, fmt.Errorf("backing up corrupted %s: %w", path, err)
	}
	return NewDoc(), true, nil
}

// UnmarshalSection decodes doc's value at key into v. A missing key or a JSON
// null value leaves v untouched and returns nil — the single fix point for
// `"hooks": null` / `"mcpServers": null` inputs, which would otherwise leave
// a nil map behind and panic on first assignment. A present non-null value
// that fails to decode is an error.
func UnmarshalSection(doc *Doc, key string, v any) error {
	raw, ok := doc.Get(key)
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("parsing %s section: %w", key, err)
	}
	return nil
}

// WriteAtomic writes data to path via a temp file + rename (0o644), so a
// crash mid-write can never leave a truncated file. The temp file is created
// in path's own directory, so the rename stays on one filesystem; it is
// removed on any failure.
//
// The temp name is per-attempt (os.CreateTemp), which is what the freshness
// cache needs and the earlier fixed `path + ".tmp"` could not give it: that
// cache is rewritten by the unattended update policy running in a background
// goroutine while hook-driven `archcore update --check` processes rewrite it
// too, and one shared temp path across processes lets a second writer truncate
// the first one's half-written temp — after which either rename publishes a
// torn file. Every other caller writes a target only one process owns and is
// unaffected — choosing-an-atomic-write.rule §6.
func WriteAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	prefix := filepath.Base(path) + ".tmp"
	sweepStaleTemps(dir, prefix)

	mode, exists := publishMode(path)
	f, tmp, err := createTemp(dir, prefix, mode)
	if err != nil {
		return fmt.Errorf("creating a temporary file next to %s: %w", path, err)
	}

	// A file that already exists keeps its own mode exactly, so a user who
	// tightened ~/.claude/settings.json to 0o600 does not get it widened back.
	// open(2) masked the perm above with the process umask; chmod does not.
	if exists {
		if err := f.Chmod(mode); err != nil {
			_ = f.Close()
			_ = os.Remove(tmp)
			return fmt.Errorf("setting the mode of %s: %w", tmp, err)
		}
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("writing %s: %w", tmp, err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("writing %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming %s: %w", tmp, err)
	}
	return nil
}

// publishMode reports the mode WriteAtomic publishes path with, and whether the
// file already exists. A new file takes 0o644 through open(2), so the process
// umask applies to it exactly as it did when this helper called os.WriteFile.
func publishMode(path string) (fs.FileMode, bool) {
	if info, err := os.Stat(path); err == nil {
		return info.Mode().Perm(), true
	}
	return 0o644, false
}

// tempSweepGrace is how long an orphaned temporary is left alone before the
// next write removes it. It has to outlast any write still in flight, and a
// write here is one Write call on a config file measured in kilobytes.
const tempSweepGrace = time.Hour

// sweepStaleTemps removes temporaries an interrupted earlier write orphaned.
//
// The per-attempt temp name below is what the freshness cache needs, but it
// also means a process killed between createTemp and Rename leaves its
// temporary behind forever — the old fixed `path + ".tmp"` name self-healed on
// the next write, and this helper writes into directories archcore does not
// own. Only names carrying this target's own prefix are considered, and only
// after they are old enough that no live writer could still hold them.
func sweepStaleTemps(dir, prefix string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), prefix) || e.Name() == prefix {
			continue
		}
		info, err := e.Info()
		if err != nil || time.Since(info.ModTime()) < tempSweepGrace {
			continue
		}
		_ = os.Remove(filepath.Join(dir, e.Name()))
	}
}

// createTemp creates a uniquely named file in dir with the given permission.
//
// os.CreateTemp cannot serve here: it opens at 0o600 unconditionally, and the
// chmod that has to follow bypasses the process umask that open(2) applies —
// so under `umask 077` every user-owned config file this package publishes
// (~/.claude/settings.json, .cursor/mcp.json) was widened from 0o600 to a
// world-readable 0o644.
func createTemp(dir, prefix string, perm fs.FileMode) (*os.File, string, error) {
	for range 10_000 {
		name := filepath.Join(dir, prefix+strconv.FormatUint(rand.Uint64(), 36))
		f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, perm)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return nil, "", err
		}
		return f, name, nil
	}
	return nil, "", errors.New("too many temporary-name collisions")
}

// MergeEntryFields overlays the desired entry's fields onto an existing entry,
// preserving any extra fields the user added (an "env" key on an MCP server, a
// setting archcore does not know). Key order of the existing entry is kept and
// desired-only keys append; a non-object existing entry is replaced wholesale.
// It reports whether the result differs semantically from what was there, so a
// caller can skip a write that would change nothing.
func MergeEntryFields(existing, desired json.RawMessage) (json.RawMessage, bool) {
	entry := NewDoc()
	if json.Unmarshal(existing, entry) != nil {
		return desired, !Equal(existing, desired)
	}
	fields := NewDoc()
	if json.Unmarshal(desired, fields) != nil {
		return desired, !Equal(existing, desired)
	}
	for pair := fields.Oldest(); pair != nil; pair = pair.Next() {
		entry.Set(pair.Key, pair.Value)
	}
	merged, err := json.Marshal(entry)
	if err != nil {
		return desired, !Equal(existing, desired)
	}
	return merged, !Equal(existing, merged)
}

// Equal reports semantic equality of two JSON values, insensitive to key order
// and whitespace.
//
// A value that is not valid JSON has no semantics to compare, so the two are
// compared literally instead. Without that fallback Equal is not reflexive —
// Equal(x, x) answered false for every invalid x, and the callers that use it
// to decide "did anything change?" would rewrite an unparsable entry on every
// pass.
func Equal(a, b json.RawMessage) bool {
	var av, bv any
	if json.Unmarshal(a, &av) != nil || json.Unmarshal(b, &bv) != nil {
		return bytes.Equal(a, b)
	}
	ac, errA := json.Marshal(av)
	bc, errB := json.Marshal(bv)
	return errA == nil && errB == nil && string(ac) == string(bc)
}

// SaveDoc marshals doc with two-space indent, appends a trailing newline,
// and writes it atomically.
func SaveDoc(path string, doc *Doc) error {
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling %s: %w", path, err)
	}
	out = append(out, '\n')
	return WriteAtomic(path, out)
}
