package projectroot

import "testing"

func TestGuardsFor_AllowHomeReflectsLegacy(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		res       *Resolution
		strict    bool
		allowHome bool
		legacy    bool
	}{
		{
			name:      "nil",
			res:       nil,
			strict:    true,
			allowHome: false,
			legacy:    false,
		},
		{
			name:      "strict",
			res:       &Resolution{LegacyMode: false},
			strict:    true,
			allowHome: false,
			legacy:    false,
		},
		{
			name:      "legacy",
			res:       &Resolution{LegacyMode: true},
			strict:    false,
			allowHome: true,
			legacy:    true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			g := GuardsFor(c.res)
			if g.Strict != c.strict {
				t.Errorf("Strict = %v, want %v", g.Strict, c.strict)
			}
			if g.AllowHome != c.allowHome {
				t.Errorf("AllowHome = %v, want %v", g.AllowHome, c.allowHome)
			}
			if g.Legacy != c.legacy {
				t.Errorf("Legacy = %v, want %v", g.Legacy, c.legacy)
			}
		})
	}
}
