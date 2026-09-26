package tools

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"archcore-cli/internal/sync"
)

func TestManifestStore_LoadRevalidatesAgainstDisk(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		appended string
		shift    time.Duration
		wantSame bool
	}{
		{name: "unchanged file", wantSame: true},
		{name: "size changed under same mtime", appended: " "},
		{name: "mtime changed under same size", shift: time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupTestArchcore(t)
			if err := sync.SaveManifest(base, sync.NewManifest()); err != nil {
				t.Fatal(err)
			}
			store := manifestStore{entries: map[string]manifestEntry{}}
			first, err := store.load(base)
			if err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(base, ".archcore", sync.ManifestFile)
			info, err := os.Stat(file)
			if err != nil {
				t.Fatal(err)
			}
			f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.WriteString(tt.appended); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
			mtime := info.ModTime().Add(tt.shift)
			if err := os.Chtimes(file, mtime, mtime); err != nil {
				t.Fatal(err)
			}
			second, err := store.load(base)
			if err != nil {
				t.Fatal(err)
			}
			if got := first == second; got != tt.wantSame {
				t.Errorf("second load reused the cached manifest = %v, want %v", got, tt.wantSame)
			}
		})
	}
}

func TestManifestStore_MutatePublishesSavedClone(t *testing.T) {
	t.Parallel()
	base := setupTestArchcore(t)
	store := manifestStore{entries: map[string]manifestEntry{}}
	var published *sync.Manifest
	err := store.mutate(base, func(m *sync.Manifest) bool {
		published = m
		return m.AddRelation("a.adr.md", "b.prd.md", sync.RelRelated)
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.load(base)
	if err != nil {
		t.Fatal(err)
	}
	if got != published {
		t.Error("load after mutate did not serve the published clone")
	}
}
