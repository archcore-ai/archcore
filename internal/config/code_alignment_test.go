package config

import (
	"encoding/json"
	"path"
	"strings"
	"testing"
)

// Forward compatibility of the codeAlignment section.
//
// Settings tolerates unknown top-level keys by capturing them into Extra, but
// codeAlignment is a known key, so it is decoded into a struct — and anything
// inside it that this binary does not recognize used to be dropped on the next
// write. That is the one section the delivery plan expects to grow, so a newer
// archcore's key would be silently erased by an older one's `config set`.

// roundTrip decodes settings JSON and re-encodes it.
func roundTrip(t *testing.T, in string) string {
	t.Helper()
	var s Settings
	if err := json.Unmarshal([]byte(in), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(out)
}

func TestCodeAlignment_UnknownNestedKeySurvives(t *testing.T) {
	t.Parallel()
	in := `{"sync":"none","codeAlignment":{"sourceRoots":["src"],"futureKnob":42}}`

	got := roundTrip(t, in)

	if !strings.Contains(got, `"futureKnob":42`) {
		t.Errorf("an unknown key inside codeAlignment was dropped:\n%s", got)
	}
	if !strings.Contains(got, `"sourceRoots":["src"]`) {
		t.Errorf("the known key did not survive:\n%s", got)
	}
}

func TestCodeAlignment_NullStaysNull(t *testing.T) {
	t.Parallel()
	got := roundTrip(t, `{"sync":"none","codeAlignment":null}`)

	if strings.Contains(got, `"codeAlignment":{}`) {
		t.Errorf(`"codeAlignment": null was rewritten as an empty object: %s`, got)
	}
	if strings.Contains(got, "codeAlignment") {
		t.Errorf("a null section should be omitted entirely, got: %s", got)
	}
}

// TestCodeAlignment_AbsentSectionStaysAbsent guards the no-churn property: a
// config that never mentioned the key must not gain it.
func TestCodeAlignment_AbsentSectionStaysAbsent(t *testing.T) {
	t.Parallel()
	got := roundTrip(t, `{"sync":"none"}`)

	if strings.Contains(got, "codeAlignment") {
		t.Errorf("marshaling invented a codeAlignment section: %s", got)
	}
}

func TestCodeAlignment_SourceRootValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		roots   string
		wantErr bool
	}{
		{name: "relative path", roots: `["src"]`},
		{name: "nested relative path", roots: `["packages/app"]`},
		{name: "a directory with a double dot in its name", roots: `["foo..bar"]`},
		{name: "a dotted directory", roots: `["..hidden"]`},
		{name: "dot-slash prefixed", roots: `["./src"]`},
		{name: "trailing slash", roots: `["src/"]`},
		{name: "traversal is rejected", roots: `["../outside"]`, wantErr: true},
		{name: "the project root itself is rejected", roots: `["."]`, wantErr: true},
		{name: "traversal in the middle is rejected", roots: `["src/../../etc"]`, wantErr: true},
		{name: "absolute path is rejected", roots: `["/etc"]`, wantErr: true},
		{name: "empty entry is rejected", roots: `[""]`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var s Settings
			err := json.Unmarshal([]byte(`{"sync":"none","codeAlignment":{"sourceRoots":`+tt.roots+`}}`), &s)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !strings.Contains(err.Error(), "sourceRoots") {
					t.Errorf("error %q should name the offending field", err)
				}
				return
			}
			// An accepted root must come out in the coordinate space the
			// advisory matches in: cleaned, with no "./" prefix and no trailing
			// slash. Accepting a root that can never match is the same defect as
			// rejecting a valid one, only silent.
			for _, root := range s.CodeAlignment.SourceRoots {
				if root != path.Clean(root) || strings.HasSuffix(root, "/") {
					t.Errorf("root %q was accepted un-normalized", root)
				}
			}
		})
	}
}

// TestCodeAlignment_TypeErrorNamesTheCause: two different malformed shapes used
// to produce the identical message because the decode error was discarded.
func TestCodeAlignment_TypeErrorNamesTheCause(t *testing.T) {
	t.Parallel()
	var s Settings
	err := json.Unmarshal([]byte(`{"sync":"none","codeAlignment":42}`), &s)
	if err == nil {
		t.Fatal("expected an error for a non-object codeAlignment")
	}
	if !strings.Contains(err.Error(), "codeAlignment") {
		t.Errorf("error %q should name the field", err)
	}
}
