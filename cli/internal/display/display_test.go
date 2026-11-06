package display

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
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

func TestDownsample(t *testing.T) {
	rendered := Success.Style.Render("ok")
	tests := []struct {
		name      string
		profile   colorprofile.Profile
		wantColor bool
	}{
		{"not a terminal strips color", colorprofile.NoTTY, false},
		{"ascii terminal strips color", colorprofile.ASCII, false},
		{"ansi terminal keeps color", colorprofile.ANSI, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := downsample(rendered, tt.profile)
			if hasColor := strings.Contains(got, "\x1b[32m"); hasColor != tt.wantColor {
				t.Fatalf("downsample(%q, %v) = %q, green present = %v, want %v", rendered, tt.profile, got, hasColor, tt.wantColor)
			}
			if !strings.Contains(got, "ok") {
				t.Fatalf("downsample(%q, %v) = %q, lost the text", rendered, tt.profile, got)
			}
		})
	}
}

func TestRender_AppliesTheStdoutProfile(t *testing.T) {
	got := Success.Render("ok")
	if want := downsample(Success.Style.Render("ok"), stdoutProfile()); got != want {
		t.Fatalf("Success.Render(%q) = %q, want the stdout profile applied: %q", "ok", got, want)
	}
	if want := detectProfile(os.Stdout, os.Environ()); stdoutProfile() != want {
		t.Fatalf("stdoutProfile() = %v, want the profile detected for stdout: %v", stdoutProfile(), want)
	}
}

func TestDetectProfile(t *testing.T) {
	tests := []struct {
		name string
		env  []string
		want colorprofile.Profile
	}{
		{"pipe writes no escape", []string{"TERM=xterm-256color"}, colorprofile.NoTTY},
		{"NO_COLOR on a terminal writes no escape, as Lip Gloss v1 did", []string{"TERM=xterm-256color", "TTY_FORCE=1", "NO_COLOR=1"}, colorprofile.NoTTY},
		{"forced color keeps color", []string{"TERM=xterm-256color", "CLICOLOR_FORCE=1"}, colorprofile.ANSI256},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectProfile(&bytes.Buffer{}, tt.env); got != tt.want {
				t.Fatalf("detectProfile(buffer, %q) = %v, want %v", tt.env, got, tt.want)
			}
		})
	}
}
