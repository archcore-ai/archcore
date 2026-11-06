//go:build darwin

package update

import (
	"context"
	"debug/macho"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestParseMacOSVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    uint32
		wantOK  bool
	}{
		{"major only", "13", 13 << 16, true},
		{"major and minor", "12.7", 12<<16 | 7<<8, true},
		{"full with newline", "15.6.1\n", 15<<16 | 6<<8 | 1, true},
		{"too many fields", "1.2.3.4", 0, false},
		{"not a number", "thirteen", 0, false},
		{"minor out of range", "13.256", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseMacOSVersion(tt.version)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("parseMacOSVersion(%q) = %#x, %v; want %#x, %v", tt.version, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestCheckOSMinimum(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	host, err := syscall.Sysctl("kern.osproductversion")
	if err != nil {
		t.Fatal(err)
	}
	running, ok := parseMacOSVersion(host)
	if !ok {
		t.Fatalf("cannot parse this Mac's version %q", host)
	}
	tests := []struct {
		name    string
		path    func(t *testing.T) string
		wantErr string
	}{
		{"a binary built for this Mac passes", func(*testing.T) string { return self }, ""},
		{"a binary that needs a newer macOS is refused", func(t *testing.T) string { return withMacOSMinimum(t, self, 99<<16) }, "needs macOS 99.0 or later"},
		{"a binary built for exactly this macOS passes", func(t *testing.T) string { return withMacOSMinimum(t, self, running) }, ""},
		{"a binary one minor version ahead is refused", func(t *testing.T) string { return withMacOSMinimum(t, self, running&^0xff+1<<8) }, "or later; this Mac runs"},
		{"a file that is not Mach-O is left to the version run", func(t *testing.T) string {
			path := filepath.Join(t.TempDir(), "archcore")
			if err := os.WriteFile(path, []byte("#!/bin/sh\necho archcore\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			return path
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkOSMinimum(tt.path(t))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("checkOSMinimum() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("checkOSMinimum() = %v, want an error containing %q", err, tt.wantErr)
			}
		})
	}
}

// withMacOSMinimum copies the Mach-O binary at path with its LC_BUILD_VERSION
// minos set to minimum.
func withMacOSMinimum(t *testing.T, path string, minimum uint32) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := macho.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	offset := 28 // the 32-bit Mach-O header
	if file.Magic == macho.Magic64 {
		offset = 32
	}
	for _, load := range file.Loads {
		raw := load.Raw()
		if file.ByteOrder.Uint32(raw) == loadCommandBuildVersion {
			file.ByteOrder.PutUint32(data[offset+12:], minimum)
			copied := filepath.Join(t.TempDir(), "archcore")
			if err := os.WriteFile(copied, data, 0o755); err != nil {
				t.Fatal(err)
			}
			return copied
		}
		offset += len(raw)
	}
	t.Fatal("the test binary carries no LC_BUILD_VERSION")
	return ""
}

func TestHealthProbe_RefusesABinaryForANewerMacOS(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	err = healthProbe(context.Background(), withMacOSMinimum(t, self, 99<<16))
	if err == nil || !strings.Contains(err.Error(), "needs macOS 99.0 or later") {
		t.Fatalf("healthProbe() = %v, want the macOS minimum refusal", err)
	}
}
