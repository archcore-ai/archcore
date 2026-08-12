package display

import (
	"strings"
	"testing"
)

func TestVersionSuffix(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{"empty", "", ""},
		{"dev", "dev", ""},
		{"real version", "v0.5.4", " v0.5.4"},
		{"another version", "v1.2.3", " v1.2.3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := versionSuffix(tt.version)
			if got != tt.want {
				t.Fatalf("versionSuffix(%q) = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}

// The "dev version" row asserts on "Archcore dev", not on a bare "dev": what it
// checks is that no version suffix was rendered, and versionSuffix renders one
// as a space then the version, directly after the name. A bare substring would
// also match any tagline that happens to contain the letters.
func TestBanner(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		wantContain string
		notContain  string
	}{
		{"no version", "", "Archcore", "v0.5.4"},
		{"dev version", "dev", "Archcore", "Archcore dev"},
		{"real version", "v0.5.4", "v0.5.4", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := Banner(tt.version)
			if !strings.Contains(b, tt.wantContain) {
				t.Fatalf("Banner(%q) missing %q: %q", tt.version, tt.wantContain, b)
			}
			if tt.notContain != "" && strings.Contains(b, tt.notContain) {
				t.Fatalf("Banner(%q) should not contain %q: %q", tt.version, tt.notContain, b)
			}
		})
	}
}

func TestWelcomeBanner(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		wantContain string
		notContain  string
	}{
		{"no version", "", "Archcore", "v0.5.4"},
		{"dev version", "dev", "Archcore", "Archcore dev"},
		{"real version", "v0.5.4", "v0.5.4", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wb := WelcomeBanner(tt.version)
			if !strings.Contains(wb, tt.wantContain) {
				t.Fatalf("WelcomeBanner(%q) missing %q: %q", tt.version, tt.wantContain, wb)
			}
			if tt.notContain != "" && strings.Contains(wb, tt.notContain) {
				t.Fatalf("WelcomeBanner(%q) should not contain %q: %q", tt.version, tt.notContain, wb)
			}
		})
	}
}
