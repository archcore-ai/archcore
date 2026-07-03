package tools

import (
	"sync"
	"time"

	"archcore-cli/templates"
)

// docCacheEntry is the parsed form of one document, valid while the file's
// (mtime, size) pair is unchanged.
type docCacheEntry struct {
	modTime time.Time
	size    int64
	fm      templates.Frontmatter
	content string
}

// scanCache is a lookaside cache for buildDoc. The directory walk still runs
// on every scan (adds and removes are detected by enumeration); only the
// per-file ReadFile + frontmatter parse is skipped on a (mtime, size) hit —
// the walk's DirEntry.Info() already provides both, so a hit costs zero extra
// syscalls. Mutex-protected: the MCP server handles tools/call on a worker
// pool. Keys are absolute file paths, so declared globals are covered and
// parallel projects (tests) stay isolated.
type scanCache struct {
	mu      sync.Mutex
	entries map[string]docCacheEntry
}

var sharedScanCache = scanCache{entries: map[string]docCacheEntry{}}

func (s *scanCache) lookup(absPath string, modTime time.Time, size int64) (docCacheEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[absPath]
	if !ok || !e.modTime.Equal(modTime) || e.size != size {
		return docCacheEntry{}, false
	}
	return e, true
}

func (s *scanCache) store(absPath string, e docCacheEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[absPath] = e
}

// invalidate drops one entry. Write handlers call it after a successful
// create/update/remove — belt and braces against coarse mtime granularity.
func (s *scanCache) invalidate(absPath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, absPath)
}

// prune drops entries absent from seen, but only once the cache has grown
// well past the current corpus — an amortized bound on stale entries from
// deleted or renamed files, with no timers or background goroutines.
func (s *scanCache) prune(seen map[string]bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.entries) <= 2*len(seen) {
		return
	}
	for k := range s.entries {
		if !seen[k] {
			delete(s.entries, k)
		}
	}
}
