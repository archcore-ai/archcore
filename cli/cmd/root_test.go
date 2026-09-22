package cmd

import (
	"errors"
	"strings"
	"testing"
)

func TestCleanVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "1.2.3", "v1.2.3"},
		{"with v prefix", "v1.2.3", "v1.2.3"},
		{"prerelease", "v0.0.1-alpha.5", "v0.0.1-alpha.5"},
		{"pseudo-version", "v0.0.1-alpha.5.0.20260310123439-e33e445c4e4e", "v0.0.1-alpha.5"},
		{"pseudo-version+dirty", "v0.0.1-alpha.5.0.20260310123439-e33e445c4e4e+dirty", "v0.0.1-alpha.5"},
		{"release+dirty", "v1.0.0+dirty", "v1.0.0"},
		{"dev", "dev", "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanVersion(tt.input)
			if got != tt.want {
				t.Errorf("cleanVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatExecuteError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantEmpty  bool
		wantSubstr []string
	}{
		{
			name:      "nil error",
			err:       nil,
			wantEmpty: true,
		},
		{
			name:       "unknown command",
			err:        errors.New(`unknown command "foo" for "archcore"`),
			wantSubstr: []string{`unknown command "foo"`, "--help", "available commands"},
		},
		{
			name:       "unknown command with suggestion",
			err:        errors.New("unknown command \"ini\" for \"archcore\"\n\nDid you mean this?\n\tinit"),
			wantSubstr: []string{`unknown command "ini"`, "Did you mean this?", "init", "--help"},
		},
		{
			name:       "unknown flag",
			err:        errors.New("unknown flag: --bogus"),
			wantSubstr: []string{"unknown flag: --bogus", "--help", "available options"},
		},
		{
			name:       "unknown shorthand flag",
			err:        errors.New(`unknown shorthand flag: 'x' in -x`),
			wantSubstr: []string{"unknown shorthand flag", "--help", "available options"},
		},
		{
			name:      "unrelated error",
			err:       errors.New("something else went wrong"),
			wantEmpty: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatExecuteError(tt.err)
			if tt.wantEmpty {
				if got != "" {
					t.Errorf("expected empty string, got %q", got)
				}
				return
			}
			for _, sub := range tt.wantSubstr {
				if !strings.Contains(got, sub) {
					t.Errorf("output missing %q\ngot: %s", sub, got)
				}
			}
		})
	}
}

func TestUnknownCommandOutput(t *testing.T) {
	root := NewRootCmd("test")
	root.SetArgs([]string{"foo"})

	err := root.ExecuteContext(t.Context())
	if err == nil {
		t.Fatal("expected error for unknown command")
	}

	msg := FormatExecuteError(err)
	if !strings.Contains(msg, `"foo"`) {
		t.Errorf("output should mention the unknown command, got: %s", msg)
	}
	if !strings.Contains(msg, "--help") {
		t.Errorf("output should mention --help, got: %s", msg)
	}
}

func TestUnknownFlagOutput(t *testing.T) {
	root := NewRootCmd("test")
	root.SetArgs([]string{"--bogus"})

	err := root.ExecuteContext(t.Context())
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}

	msg := FormatExecuteError(err)
	if !strings.Contains(msg, "--bogus") {
		t.Errorf("output should mention the unknown flag, got: %s", msg)
	}
	if !strings.Contains(msg, "--help") {
		t.Errorf("output should mention --help, got: %s", msg)
	}
}
