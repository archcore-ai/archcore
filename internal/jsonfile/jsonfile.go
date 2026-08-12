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
	"os"

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
// backup-invalid-configs.adr.md: when the file exists but is not a valid JSON
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
// crash mid-write can never leave a truncated file (mirrors sync.SaveManifest).
// The temp file lives next to path so the rename stays on one filesystem; it
// is removed on rename failure.
func WriteAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming %s: %w", tmp, err)
	}
	return nil
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
