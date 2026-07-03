package tools

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	archsync "archcore-cli/internal/sync"
)

// manifestEntry is one cached parsed manifest, valid while the on-disk
// .sync-state.json keeps the same (mtime, size).
type manifestEntry struct {
	modTime  time.Time
	size     int64
	manifest *archsync.Manifest
}

// manifestStore serializes manifest mutations from MCP handlers and caches
// parsed manifests for readers, keyed by baseDir and invalidated on the
// manifest file's (mtime, size).
//
// mcp-go's stdio server dispatches tools/call on a worker pool, so two
// concurrent handlers doing load-modify-save directly would lose updates
// (last SaveManifest rename wins). All handler-side mutations go through
// mutate, all handler-side reads through load. Cross-process safety is
// unchanged: sync.SaveManifest's tmp+rename remains the on-disk atomicity
// boundary, and CLI commands in other processes keep calling
// sync.LoadManifest directly.
type manifestStore struct {
	mu      sync.Mutex
	entries map[string]manifestEntry
}

// sharedManifestStore is package-level: one MCP server process serves one
// primary, and the per-baseDir keying keeps parallel tests isolated.
var sharedManifestStore = manifestStore{entries: map[string]manifestEntry{}}

// load returns the parsed manifest for reading. Callers MUST NOT mutate the
// returned value — it is a shared snapshot; mutations go through mutate.
// A missing file yields a fresh empty manifest (mirrors sync.LoadManifest).
func (m *manifestStore) load(baseDir string) (*archsync.Manifest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.loadLocked(baseDir)
}

func (m *manifestStore) loadLocked(baseDir string) (*archsync.Manifest, error) {
	manifestFile := filepath.Join(baseDir, ".archcore", archsync.ManifestFile)
	info, statErr := os.Stat(manifestFile)
	if statErr == nil {
		if entry, ok := m.entries[baseDir]; ok &&
			entry.modTime.Equal(info.ModTime()) && entry.size == info.Size() {
			return entry.manifest, nil
		}
	}

	loaded, err := archsync.LoadManifest(baseDir)
	if err != nil {
		delete(m.entries, baseDir)
		return nil, err
	}
	if statErr == nil {
		m.entries[baseDir] = manifestEntry{modTime: info.ModTime(), size: info.Size(), manifest: loaded}
	} else {
		delete(m.entries, baseDir) // missing file: nothing to key the cache on
	}
	return loaded, nil
}

// mutate runs fn on a deep clone of the current manifest under the store lock
// and, when fn reports a change, saves the clone via sync.SaveManifest and
// publishes it. Clone-and-swap keeps readers holding the previous snapshot
// race-free — the published manifest is never modified in place.
func (m *manifestStore) mutate(baseDir string, fn func(*archsync.Manifest) bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, err := m.loadLocked(baseDir)
	if err != nil {
		return err
	}
	clone := current.Clone()
	if !fn(clone) {
		return nil
	}
	if err := archsync.SaveManifest(baseDir, clone); err != nil {
		return err
	}

	manifestFile := filepath.Join(baseDir, ".archcore", archsync.ManifestFile)
	if info, statErr := os.Stat(manifestFile); statErr == nil {
		m.entries[baseDir] = manifestEntry{modTime: info.ModTime(), size: info.Size(), manifest: clone}
	} else {
		delete(m.entries, baseDir)
	}
	return nil
}
