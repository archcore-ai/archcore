package docs

import (
	"sync"
	"time"

	"archcore-cli/templates"
)

// docCacheEntry is the parsed form of one document, valid while the file's
// (mtime, size) pair is unchanged.
//
// content is populated only when a caller asked for it. hasContent distinguishes
// "cached without a body" from "cached with an empty body", so a metadata scan
// cannot satisfy a later content scan by handing back "".
type docCacheEntry struct {
	modTime    time.Time
	size       int64
	fm         templates.Frontmatter
	content    string
	hasContent bool
}

// maxCachedContentBytes bounds the bodies the cache retains. Past it, entries
// are still cached for their frontmatter but drop their body, so the cost
// degrades to re-reading rather than growing without limit.
//
// No eviction policy: an LRU would add a second data structure and an ordering
// invariant to state the MCP worker pool touches concurrently, to bound
// something a flat cap already bounds.
const maxCachedContentBytes = 32 << 20

// scanCache is a lookaside cache for buildDoc. The directory walk still runs
// on every scan (adds and removes are detected by enumeration); only the
// per-file ReadFile + frontmatter parse is skipped on a (mtime, size) hit —
// the walk's DirEntry.Info() already provides both, so a hit costs zero extra
// syscalls. Mutex-protected: the MCP server handles tools/call on a worker
// pool. Keys are absolute file paths, so declared globals are covered.
//
// The cache is process-local. It pays off in the long-lived MCP server; a
// short-lived hook process always starts cold, which is why the hook path must
// stay cheap on a cold cache rather than relying on warmth.
type scanCache struct {
	mu           sync.Mutex
	entries      map[string]docCacheEntry
	contentBytes int
}

var sharedScanCache = scanCache{entries: map[string]docCacheEntry{}}

// lookup returns the cached parse when it is still valid for the file's current
// (mtime, size). An entry cached without a body is a miss for a caller that
// needs one.
func (s *scanCache) lookup(absPath string, modTime time.Time, size int64, needContent bool) (docCacheEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[absPath]
	if !ok || !e.modTime.Equal(modTime) || e.size != size {
		return docCacheEntry{}, false
	}
	if needContent && !e.hasContent {
		return docCacheEntry{}, false
	}
	return e, true
}

// store records a parse, dropping the body when retaining it would push the
// cache past its byte cap.
func (s *scanCache) store(absPath string, e docCacheEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if prev, ok := s.entries[absPath]; ok {
		s.contentBytes -= len(prev.content)
	}
	if e.hasContent && s.contentBytes+len(e.content) > maxCachedContentBytes {
		e.content, e.hasContent = "", false
	}
	s.contentBytes += len(e.content)
	s.entries[absPath] = e
}

// invalidate drops one entry. Write handlers call it after a successful
// create/update/remove — belt and braces against coarse mtime granularity.
func (s *scanCache) invalidate(absPath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if prev, ok := s.entries[absPath]; ok {
		s.contentBytes -= len(prev.content)
		delete(s.entries, absPath)
	}
}

// needsPrune reports whether the cache has outgrown the corpus enough to be
// worth sweeping. Checked before building the key set, which is otherwise a
// full-corpus allocation on every scan for a sweep that almost never runs.
func (s *scanCache) needsPrune(corpusSize int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries) > 2*corpusSize
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
	for k, e := range s.entries {
		if !seen[k] {
			s.contentBytes -= len(e.content)
			delete(s.entries, k)
		}
	}
}

// reset empties the cache.
func (s *scanCache) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = map[string]docCacheEntry{}
	s.contentBytes = 0
}

// InvalidateCache drops the cached parse of one document. Callers that mutate a
// document on disk must call it so the next scan re-reads the file.
func InvalidateCache(absPath string) {
	sharedScanCache.invalidate(absPath)
}

// ResetCache empties the process-wide scan cache. A hook process always starts
// cold, so a benchmark that does not reset between iterations measures the warm
// path and reports a number the real invocation never sees.
func ResetCache() {
	sharedScanCache.reset()
}
