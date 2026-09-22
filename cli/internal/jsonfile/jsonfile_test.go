package jsonfile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
)

func TestRead(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		content  *string // nil = file does not exist
		wantErr  bool
		wantKeys []string
	}{
		{
			name:    "missing file yields empty doc",
			content: nil,
		},
		{
			name:     "valid object preserves keys",
			content:  strPtr(`{"b": 1, "a": {"nested": true}, "c": "x"}`),
			wantKeys: []string{"b", "a", "c"},
		},
		{
			name:    "invalid JSON is an error",
			content: strPtr(`{"a": `),
			wantErr: true,
		},
		{
			name:    "whole-file null is an error",
			content: strPtr(`null`),
			wantErr: true,
		},
		{
			name:    "array is an error",
			content: strPtr(`[1, 2]`),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "config.json")
			if tt.content != nil {
				if err := os.WriteFile(path, []byte(*tt.content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			doc, err := Read(path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			var got []string
			for pair := doc.Oldest(); pair != nil; pair = pair.Next() {
				got = append(got, pair.Key)
			}
			if len(got) != len(tt.wantKeys) {
				t.Fatalf("keys = %v, want %v", got, tt.wantKeys)
			}
			for i, k := range tt.wantKeys {
				if got[i] != k {
					t.Errorf("key[%d] = %q, want %q (order must be preserved)", i, got[i], k)
				}
			}
		})
	}
}

func TestReadOrBackup(t *testing.T) {
	t.Parallel()

	t.Run("valid file has no backup", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(`{"a": 1}`), 0o644); err != nil {
			t.Fatal(err)
		}
		doc, backedUp, err := ReadOrBackup(path)
		if err != nil {
			t.Fatal(err)
		}
		if backedUp {
			t.Error("valid file must not be backed up")
		}
		if _, ok := doc.Get("a"); !ok {
			t.Error("doc must carry the parsed content")
		}
		if _, statErr := os.Stat(path + ".bak"); !os.IsNotExist(statErr) {
			t.Error("no .bak file may exist for a valid config")
		}
	})

	t.Run("corrupt file backed up with original bytes", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "config.json")
		original := []byte(`{broken`)
		if err := os.WriteFile(path, original, 0o644); err != nil {
			t.Fatal(err)
		}
		doc, backedUp, err := ReadOrBackup(path)
		if err != nil {
			t.Fatal(err)
		}
		if !backedUp {
			t.Fatal("corrupt file must report backedUp")
		}
		if doc.Len() != 0 {
			t.Error("corrupt file must yield an empty doc")
		}
		bak, err := os.ReadFile(path + ".bak")
		if err != nil {
			t.Fatal(err)
		}
		if string(bak) != string(original) {
			t.Errorf(".bak = %q, want original bytes %q", bak, original)
		}
	})

	t.Run("backup write failure aborts", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "config.json")
		original := []byte(`{broken`)
		if err := os.WriteFile(path, original, 0o644); err != nil {
			t.Fatal(err)
		}
		// A directory at path+".bak" makes the backup rename fail.
		if err := os.MkdirAll(path+".bak", 0o755); err != nil {
			t.Fatal(err)
		}
		_, _, err := ReadOrBackup(path)
		if err == nil {
			t.Fatal("expected error when the backup cannot be written")
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if string(data) != string(original) {
			t.Error("original file must be untouched when backup fails")
		}
	})
}

func TestUnmarshalSection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		doc     string
		key     string
		wantErr bool
		wantLen int // entries decoded into the map
	}{
		{name: "missing key leaves target untouched", doc: `{}`, key: "hooks"},
		{name: "null value leaves target untouched", doc: `{"hooks": null}`, key: "hooks"},
		{name: "null with whitespace tolerated", doc: `{"hooks":  null }`, key: "hooks"},
		{name: "object decodes", doc: `{"hooks": {"a": [], "b": []}}`, key: "hooks", wantLen: 2},
		{name: "wrong type is an error", doc: `{"hooks": "oops"}`, key: "hooks", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc := NewDoc()
			if err := json.Unmarshal([]byte(tt.doc), doc); err != nil {
				t.Fatal(err)
			}
			target := map[string][]json.RawMessage{}
			err := UnmarshalSection(doc, tt.key, &target)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && len(target) != tt.wantLen {
				t.Errorf("decoded %d entries, want %d", len(target), tt.wantLen)
			}
		})
	}
}

func TestUnmarshalSection_NullIntoOrderedMap(t *testing.T) {
	t.Parallel()
	// Regression pin: orderedmap.UnmarshalJSON errors on null, and a plain map
	// stays nil — both previously caused panics or errors downstream. The
	// section helper must treat null as "absent".
	doc := NewDoc()
	if err := json.Unmarshal([]byte(`{"mcpServers": null}`), doc); err != nil {
		t.Fatal(err)
	}
	servers := NewDoc()
	if err := UnmarshalSection(doc, "mcpServers", servers); err != nil {
		t.Fatalf("null section must not error: %v", err)
	}
	servers.Set("archcore", json.RawMessage(`{}`))
	if servers.Len() != 1 {
		t.Error("section map must be usable after null input")
	}
}

func TestWriteAtomic(t *testing.T) {
	t.Parallel()

	t.Run("writes content without tmp leftover", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "out.json")
		if err := WriteAtomic(path, []byte("data")); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "data" {
			t.Errorf("content = %q", data)
		}
		if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
			t.Error("tmp file must not remain")
		}
		assertOnlyEntries(t, filepath.Dir(path), "out.json")
	})

	// The mode is the half of the contract no content assertion can see: drop
	// it and every file this helper publishes — the sync manifest, a host
	// config, the update freshness cache — silently changes permission while
	// every test that reads its content back stays green.
	//
	// A rename replaces the destination's mode along with its content, so the
	// two directions are separate cases. A file that already exists keeps its
	// own mode: this helper writes ~/.claude/settings.json and .cursor/mcp.json,
	// files archcore does not own, and a user who tightened one to 0o600 must
	// not have it widened back on the next merge. 0o644 is the mode the rule
	// tabulates for a file this helper creates —
	// choosing-an-atomic-write.rule.
	t.Run("preserves an existing mode and creates at 0o644", func(t *testing.T) {
		t.Parallel()
		if runtime.GOOS == "windows" {
			t.Skip("windows does not carry unix mode bits")
		}
		tests := []struct {
			name   string
			mode   fs.FileMode
			create bool
			want   fs.FileMode
		}{
			{name: "fresh target", create: false, want: 0o644},
			{name: "tightened target", create: true, mode: 0o600, want: 0o600},
			{name: "widened target", create: true, mode: 0o664, want: 0o664},
			{name: "already 0o644", create: true, mode: 0o644, want: 0o644},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				path := filepath.Join(t.TempDir(), "target.json")
				if tt.create {
					if err := os.WriteFile(path, []byte("old"), tt.mode); err != nil {
						t.Fatal(err)
					}
					// os.WriteFile is umask-masked; force the mode we mean to test.
					if err := os.Chmod(path, tt.mode); err != nil {
						t.Fatal(err)
					}
				}
				if err := WriteAtomic(path, []byte("data")); err != nil {
					t.Fatal(err)
				}
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				if got := info.Mode().Perm(); got != tt.want {
					t.Errorf("published at %#o, want %#o", got, tt.want)
				}
			})
		}
	})

	// The umask half of this contract needs syscall.Umask, which does not exist
	// on windows, so it lives in jsonfile_umask_unix_test.go.

	t.Run("rename failure removes tmp", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		target := filepath.Join(dir, "occupied")
		if err := os.MkdirAll(target, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := WriteAtomic(target, []byte("data")); err == nil {
			t.Fatal("expected error renaming onto an existing directory")
		}
		if _, err := os.Stat(target + ".tmp"); !os.IsNotExist(err) {
			t.Error("tmp file must be removed after failed rename")
		}
		assertOnlyEntries(t, dir, "occupied")
	})

	// Two processes rewriting one target is the update freshness cache: the
	// unattended policy refreshes it from a background goroutine while
	// hook-driven `archcore update --check` runs rewrite it too. A shared temp
	// name lets one writer truncate the other's half-written temp and the
	// rename publish those bytes, so a reader sees a value neither writer
	// wrote. Every read here must return one whole input.
	t.Run("concurrent writers never publish a torn file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "cache")

		// The payloads differ in length as well as in content: a same-length
		// pair would hide a half-write that happened to land on a boundary.
		payloads := [][]byte{
			bytes.Repeat([]byte("a"), 4096),
			bytes.Repeat([]byte("bb"), 16384),
			bytes.Repeat([]byte("ccc"), 8192),
		}
		whole := make(map[string]bool, len(payloads))
		for _, p := range payloads {
			whole[string(p)] = true
		}

		var writers, reader sync.WaitGroup
		stop := make(chan struct{})
		readErr := make(chan error, 1)

		// One reader, running for the whole write storm.
		reader.Add(1)
		go func() {
			defer reader.Done()
			fail := func(err error) {
				select {
				case readErr <- err:
				default:
				}
			}
			for {
				select {
				case <-stop:
					return
				default:
				}
				data, err := os.ReadFile(path)
				if os.IsNotExist(err) {
					continue // no writer has published yet
				}
				if err != nil {
					fail(err)
					return
				}
				if !whole[string(data)] {
					fail(fmt.Errorf("read a torn file: %d bytes, prefix %.16q", len(data), data))
					return
				}
			}
		}()

		for _, payload := range payloads {
			writers.Add(1)
			go func(data []byte) {
				defer writers.Done()
				for n := 0; n < 40; n++ {
					if err := WriteAtomic(path, data); err != nil {
						t.Errorf("WriteAtomic: %v", err)
						return
					}
				}
			}(payload)
		}

		writers.Wait()
		close(stop)
		reader.Wait()

		select {
		case err := <-readErr:
			t.Error(err)
		default:
		}
		assertOnlyEntries(t, dir, "cache")
	})
}

// assertOnlyEntries fails when dir holds anything besides want — a temp file
// the writer did not clean up shows here and nowhere else.
func assertOnlyEntries(t *testing.T, dir string, want ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("directory holds %v, want %v", got, want)
	}
}

func TestSaveDoc_RoundTripFidelity(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	input := `{
  "zeta": {"deep": {"unknown": [1, 2, {"x": "a\"b<c>"}]}},
  "alpha": "second",
  "hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "lint.sh", "timeout": 120}]}]}
}`
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}

	doc, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveDoc(path, doc); err != nil {
		t.Fatal(err)
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.HasSuffix(text, "\n") {
		t.Error("saved doc must end with a newline")
	}
	// Key order preserved: zeta before alpha before hooks.
	if zi, ai, hi := strings.Index(text, `"zeta"`), strings.Index(text, `"alpha"`), strings.Index(text, `"hooks"`); !(zi < ai && ai < hi) {
		t.Errorf("key order not preserved: zeta@%d alpha@%d hooks@%d", zi, ai, hi)
	}
	// Unknown nested fields survive.
	if !strings.Contains(text, `"timeout": 120`) {
		t.Error("unknown nested field timeout must survive the round trip")
	}
	// Escaped string content survives semantically.
	reparsed := NewDoc()
	if err := json.Unmarshal(out, reparsed); err != nil {
		t.Fatalf("re-parsing saved doc: %v", err)
	}
	zeta, _ := reparsed.Get("zeta")
	if !strings.Contains(string(zeta), `a\"b`) {
		t.Errorf("escaped string content lost: %s", zeta)
	}
}

func strPtr(s string) *string { return &s }
