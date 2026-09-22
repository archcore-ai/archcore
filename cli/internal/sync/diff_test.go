package sync

import "testing"

func TestDiff(t *testing.T) {
	tests := []struct {
		name          string
		current       []FileState
		manifest      *Manifest
		wantCreated   int
		wantModified  int
		wantDeleted   int
		wantUnchanged int
	}{
		{
			name: "first sync - all created",
			current: []FileState{
				{RelPath: "vision/adr-001.md", Hash: "aaa"},
				{RelPath: "knowledge/rfc-001.md", Hash: "bbb"},
			},
			manifest:    NewManifest(),
			wantCreated: 2,
		},
		{
			name: "no changes",
			current: []FileState{
				{RelPath: "vision/adr-001.md", Hash: "aaa"},
			},
			manifest: &Manifest{Files: map[string]string{
				"vision/adr-001.md": "aaa",
			}},
			wantUnchanged: 1,
		},
		{
			name: "one modified",
			current: []FileState{
				{RelPath: "vision/adr-001.md", Hash: "new-hash"},
			},
			manifest: &Manifest{Files: map[string]string{
				"vision/adr-001.md": "old-hash",
			}},
			wantModified: 1,
		},
		{
			name:    "one deleted",
			current: []FileState{},
			manifest: &Manifest{Files: map[string]string{
				"vision/adr-001.md": "aaa",
			}},
			wantDeleted: 1,
		},
		{
			name: "mixed: created + modified + deleted + unchanged",
			current: []FileState{
				{RelPath: "vision/new.md", Hash: "new"},
				{RelPath: "vision/changed.md", Hash: "v2"},
				{RelPath: "vision/same.md", Hash: "unchanged"},
			},
			manifest: &Manifest{Files: map[string]string{
				"vision/changed.md": "v1",
				"vision/same.md":    "unchanged",
				"vision/removed.md": "gone",
			}},
			wantCreated:   1,
			wantModified:  1,
			wantDeleted:   1,
			wantUnchanged: 1,
		},
		{
			name:     "empty current and empty manifest",
			current:  []FileState{},
			manifest: NewManifest(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := Diff(tt.current, tt.manifest)

			gotCreated := len(FilterByAction(entries, ActionCreated))
			gotModified := len(FilterByAction(entries, ActionModified))
			gotDeleted := len(FilterByAction(entries, ActionDeleted))
			gotUnchanged := len(FilterByAction(entries, ActionUnchanged))

			if gotCreated != tt.wantCreated {
				t.Errorf("created = %d, want %d", gotCreated, tt.wantCreated)
			}
			if gotModified != tt.wantModified {
				t.Errorf("modified = %d, want %d", gotModified, tt.wantModified)
			}
			if gotDeleted != tt.wantDeleted {
				t.Errorf("deleted = %d, want %d", gotDeleted, tt.wantDeleted)
			}
			if gotUnchanged != tt.wantUnchanged {
				t.Errorf("unchanged = %d, want %d", gotUnchanged, tt.wantUnchanged)
			}
		})
	}
}

func TestHasChanges(t *testing.T) {
	tests := []struct {
		name    string
		entries []DiffEntry
		want    bool
	}{
		{"empty", nil, false},
		{"all unchanged", []DiffEntry{{Action: ActionUnchanged}}, false},
		{"has created", []DiffEntry{{Action: ActionCreated}}, true},
		{"has deleted", []DiffEntry{{Action: ActionDeleted}}, true},
		{"has modified", []DiffEntry{{Action: ActionModified}}, true},
		{"mixed", []DiffEntry{{Action: ActionUnchanged}, {Action: ActionCreated}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasChanges(tt.entries); got != tt.want {
				t.Errorf("HasChanges = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterByAction(t *testing.T) {
	entries := []DiffEntry{
		{RelPath: "a", Action: ActionCreated},
		{RelPath: "b", Action: ActionModified},
		{RelPath: "c", Action: ActionCreated},
		{RelPath: "d", Action: ActionUnchanged},
	}

	created := FilterByAction(entries, ActionCreated)
	if len(created) != 2 {
		t.Errorf("created = %d, want 2", len(created))
	}

	modified := FilterByAction(entries, ActionModified)
	if len(modified) != 1 {
		t.Errorf("modified = %d, want 1", len(modified))
	}

	deleted := FilterByAction(entries, ActionDeleted)
	if len(deleted) != 0 {
		t.Errorf("deleted = %d, want 0", len(deleted))
	}
}
