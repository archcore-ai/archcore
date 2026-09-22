package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestNeedsUpdate(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{"same version", "v1.0.0", "v1.0.0", false},
		{"patch update", "v1.0.0", "v1.0.1", true},
		{"minor update", "v1.0.0", "v1.1.0", true},
		{"major update", "v1.0.0", "v2.0.0", true},
		{"current newer patch", "v1.0.2", "v1.0.1", false},
		{"current newer minor", "v1.2.0", "v1.1.0", false},
		{"current newer major", "v2.0.0", "v1.9.9", false},
		{"dev always updates", "dev", "v1.0.0", true},
		{"dev with v prefix", "vdev", "v1.0.0", true},
		{"no v prefix current", "1.0.0", "v1.1.0", true},
		{"no v prefix latest", "v1.0.0", "1.1.0", true},
		{"no v prefix both", "1.0.0", "1.1.0", true},
		{"pre-release to release", "v1.0.0-alpha.1", "v1.0.0", true},
		{"pre-release major update", "v0.9.0-beta.1", "v1.0.0", true},
		{"same pre-release", "v0.0.1-alpha.7", "v0.0.1-alpha.7", false},
		{"pre-release bump", "v0.0.1-alpha.7", "v0.0.1-alpha.8", true},
		{"pre-release newer current", "v0.0.1-alpha.8", "v0.0.1-alpha.7", false},
		{"release to pre-release", "v1.0.0", "v1.0.0-alpha.1", false},
		{"alpha to beta", "v1.0.0-alpha.1", "v1.0.0-beta.1", true},

		// An unparseable version on either side means no update. The old
		// string-inequality fallback answered true for every row below, which
		// on the unattended path is a replacement or a downgrade decided by a
		// tag nobody validated — unattended-update.spec §12.
		{"unparseable latest", "v1.0.0", "nightly", false},
		{"unparseable current", "nightly", "v1.0.0", false},
		{"both unparseable and different", "nightly", "edge", false},
		{"both unparseable and equal", "nightly", "nightly", false},
		{"two-part latest", "v1.0.0", "v1.1", false},
		{"two-part current", "v1.0", "v1.1.0", false},
		{"empty latest", "v1.0.0", "", false},
		{"empty current", "", "v1.0.0", false},
		// The `dev` case is checked before parsing, so it survives the
		// tightening: a development build has no parseable version at all.
		{"dev to unparseable latest", "dev", "nightly", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NeedsUpdate(tt.current, tt.latest)
			if got != tt.want {
				t.Errorf("NeedsUpdate(%q, %q) = %v, want %v",
					tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

// TestNewerSemver pins the ok half that NeedsUpdate discards: a caller that
// must not act on an unparseable version needs to tell "not newer" apart from
// "not comparable". `dev` is deliberately not special here — NeedsUpdate owns
// that case.
func TestNewerSemver(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		current   string
		latest    string
		wantNewer bool
		wantOK    bool
	}{
		{name: "patch update", current: "v1.0.0", latest: "v1.0.1", wantNewer: true, wantOK: true},
		{name: "minor update", current: "v1.0.0", latest: "v1.1.0", wantNewer: true, wantOK: true},
		{name: "major update", current: "v1.0.0", latest: "v2.0.0", wantNewer: true, wantOK: true},
		{name: "equal versions", current: "v1.2.3", latest: "v1.2.3", wantNewer: false, wantOK: true},
		{name: "equal without v prefix", current: "1.2.3", latest: "v1.2.3", wantNewer: false, wantOK: true},
		{name: "older latest patch", current: "v1.0.2", latest: "v1.0.1", wantNewer: false, wantOK: true},
		{name: "older latest major", current: "v2.0.0", latest: "v1.9.9", wantNewer: false, wantOK: true},
		{name: "pre-release to release", current: "v1.0.0-alpha.1", latest: "v1.0.0", wantNewer: true, wantOK: true},
		{name: "release to pre-release", current: "v1.0.0", latest: "v1.0.0-alpha.1", wantNewer: false, wantOK: true},
		{name: "equal pre-release", current: "v0.0.1-alpha.7", latest: "v0.0.1-alpha.7", wantNewer: false, wantOK: true},

		// Not comparable — both return values must say so.
		{name: "unparseable latest", current: "v1.0.0", latest: "nightly", wantNewer: false, wantOK: false},
		{name: "unparseable current", current: "nightly", latest: "v1.0.0", wantNewer: false, wantOK: false},
		{name: "dev is not special here", current: "dev", latest: "v1.0.0", wantNewer: false, wantOK: false},
		{name: "both unparseable", current: "nightly", latest: "edge", wantNewer: false, wantOK: false},
		{name: "two-part latest", current: "v1.0.0", latest: "v1.1", wantNewer: false, wantOK: false},
		{name: "non-integer patch", current: "v1.0.0", latest: "v1.0.x", wantNewer: false, wantOK: false},
		{name: "empty both", current: "", latest: "", wantNewer: false, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotNewer, gotOK := NewerSemver(tt.current, tt.latest)
			if gotNewer != tt.wantNewer || gotOK != tt.wantOK {
				t.Errorf("NewerSemver(%q, %q) = (%v, %v), want (%v, %v)",
					tt.current, tt.latest, gotNewer, gotOK, tt.wantNewer, tt.wantOK)
			}
		})
	}
}

func TestCheckLatest(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		location   string
		want       string
	}{
		// github.com answers /releases/latest with a 302 to the tag page.
		{"302 to tag page", http.StatusFound,
			"https://github.com/archcore-ai/cli/releases/tag/v1.2.3", "v1.2.3"},
		// Any 3xx is a legitimate way to point at the tag page.
		{"301 to tag page", http.StatusMovedPermanently,
			"https://github.com/archcore-ai/cli/releases/tag/v1.2.3", "v1.2.3"},
		{"307 to tag page", http.StatusTemporaryRedirect,
			"https://github.com/archcore-ai/cli/releases/tag/v1.2.3", "v1.2.3"},
		{"308 to tag page", http.StatusPermanentRedirect,
			"https://github.com/archcore-ai/cli/releases/tag/v1.2.3", "v1.2.3"},
		// The tag comes from the resolved path, so query and fragment never
		// reach it.
		{"query and fragment", http.StatusFound,
			"https://github.com/archcore-ai/cli/releases/tag/v1.2.3?a=b#frag", "v1.2.3"},
		// ... and neither can smuggle a second marker past the search (a
		// raw-string LastIndex would have returned "evil" for these two).
		{"query with a decoy tag marker", http.StatusFound,
			"https://github.com/archcore-ai/cli/releases/tag/v1.2.3?next=/releases/tag/evil", "v1.2.3"},
		{"fragment with a decoy tag marker", http.StatusFound,
			"https://github.com/archcore-ai/cli/releases/tag/v1.2.3#/releases/tag/evil", "v1.2.3"},
		// Relative Location resolved against the request URL.
		{"relative location", http.StatusFound,
			"/archcore-ai/cli/releases/tag/v1.2.3", "v1.2.3"},
		// Dot-segments are normalized before the marker search.
		{"dot segments inside the path", http.StatusFound,
			"https://github.com/archcore-ai/cli/releases/x/../tag/v1.2.3", "v1.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/archcore-ai/cli/releases/latest" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Location", tt.location)
				w.WriteHeader(tt.statusCode)
			}))
			defer srv.Close()

			u := NewUpdater("v1.0.0", "archcore-ai/cli", "archcore")
			// Point the github.com host at the test server by wrapping the transport.
			u.HTTPClient = &http.Client{
				Transport: &rewriteTransport{
					base:   http.DefaultTransport,
					target: srv.URL,
				},
			}

			got, err := u.CheckLatest(context.Background())
			if err != nil {
				t.Fatalf("CheckLatest() error: %v", err)
			}
			if got != tt.want {
				t.Errorf("CheckLatest() = %q, want %q", got, tt.want)
			}
			// Halting redirects must stay local to the call: Apply reuses the
			// caller's client to follow the release-asset redirect chain.
			if u.HTTPClient.CheckRedirect != nil {
				t.Error("CheckRedirect leaked onto the caller's client")
			}
		})
	}
}

func TestCheckLatestError(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		location    string
		errContains string
	}{
		{"server error", http.StatusInternalServerError, "", "returned status 500"},
		{"not found", http.StatusNotFound, "", "returned status 404"},
		{"rate limited", http.StatusForbidden, "", "returned status 403"},
		// The status guard is load-bearing on its own: without it this row
		// would decode to v9.9.9 from a body-bearing 200.
		{"non-redirect with tag location", http.StatusOK,
			"https://github.com/archcore-ai/cli/releases/tag/v9.9.9", "returned status 200"},
		// A repo with no published release redirects to the bare /releases
		// page, verified against github/gitignore and golang/go (both have an
		// empty releases list): 302 -> https://github.com/OWNER/REPO/releases.
		{"no release published", http.StatusFound,
			"https://github.com/archcore-ai/cli/releases", "unexpected redirect"},
		// A redirect that lands somewhere else entirely — e.g. an interstitial
		// or a repo rename, whose target is another /releases/latest.
		{"redirect without tag segment", http.StatusFound,
			"https://github.com/login", "unexpected redirect"},
		// Traversal in the redirect target must not survive into the tag: the
		// resolved path is /archcore-ai/evil, which has no tag marker at all.
		{"redirect with traversal segments", http.StatusFound,
			"https://github.com/archcore-ai/cli/releases/tag/../../../evil", "unexpected redirect"},
		// Well-formed tag URL with nothing after the marker.
		{"redirect with empty tag", http.StatusFound,
			"https://github.com/archcore-ai/cli/releases/tag/", "empty tag in redirect"},
		// 3xx with the Location header missing entirely — a distinct branch
		// from "redirected somewhere that is not a tag page".
		{"redirect without location", http.StatusFound, "", "no Location header"},
		// An unparseable Location. The status must be a 3xx the http.Client
		// does not itself follow (300 is not in its redirect set), because for
		// 301/302/303/307/308 the client parses Location before consulting
		// CheckRedirect and fails the whole call earlier.
		{"unparseable location", http.StatusMultipleChoices, "http://[::1",
			"parsing redirect location"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.location != "" {
					w.Header().Set("Location", tt.location)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer srv.Close()

			u := NewUpdater("v1.0.0", "archcore-ai/cli", "archcore")
			u.HTTPClient = &http.Client{
				Transport: &rewriteTransport{
					base:   http.DefaultTransport,
					target: srv.URL,
				},
			}

			_, err := u.CheckLatest(context.Background())
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error = %v, want it to contain %q", err, tt.errContains)
			}
		})
	}
}

// TestCheckLatest_NilHTTPClient pins the http.DefaultClient fallback: an
// Updater built without an HTTPClient must not nil-panic. The context is
// cancelled up front, so the transport returns before touching the network.
func TestCheckLatest_NilHTTPClient(t *testing.T) {
	t.Parallel()
	u := &Updater{CurrentVersion: "v1.0.0", GitHubRepo: "archcore-ai/cli", BinaryName: "archcore"}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := u.CheckLatest(ctx); err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if u.HTTPClient != nil {
		t.Error("client() must not populate HTTPClient")
	}
	if http.DefaultClient.CheckRedirect != nil {
		t.Error("CheckRedirect leaked onto http.DefaultClient")
	}
}

// TestDownload_NilHTTPClient pins the same fallback on the download path, so
// one Updater cannot be defended in CheckLatest and nil-panic here.
func TestDownload_NilHTTPClient(t *testing.T) {
	t.Parallel()
	u := &Updater{CurrentVersion: "v1.0.0", GitHubRepo: "archcore-ai/cli", BinaryName: "archcore"}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := u.download(ctx, "v1.0.0", "checksums.txt"); err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte("hello world")
	hash := sha256.Sum256(data)
	validHash := fmt.Sprintf("%x", hash)

	tests := []struct {
		name      string
		data      []byte
		checksums string
		filename  string
		wantErr   bool
	}{
		{
			name:      "valid checksum",
			data:      data,
			checksums: fmt.Sprintf("%s  archcore_linux_amd64.tar.gz\n", validHash),
			filename:  "archcore_linux_amd64.tar.gz",
			wantErr:   false,
		},
		{
			name:      "valid with multiple entries",
			data:      data,
			checksums: fmt.Sprintf("abc123  archcore_darwin_arm64.tar.gz\n%s  archcore_linux_amd64.tar.gz\n", validHash),
			filename:  "archcore_linux_amd64.tar.gz",
			wantErr:   false,
		},
		{
			name:      "checksum mismatch",
			data:      data,
			checksums: "0000000000000000000000000000000000000000000000000000000000000000  archcore_linux_amd64.tar.gz\n",
			filename:  "archcore_linux_amd64.tar.gz",
			wantErr:   true,
		},
		{
			name:      "file not in checksums",
			data:      data,
			checksums: fmt.Sprintf("%s  archcore_darwin_arm64.tar.gz\n", validHash),
			filename:  "archcore_linux_amd64.tar.gz",
			wantErr:   true,
		},
		{
			name:      "empty checksums",
			data:      data,
			checksums: "",
			filename:  "archcore_linux_amd64.tar.gz",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyChecksum(tt.data, []byte(tt.checksums), tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyChecksum() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractBinary(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho archcore")

	tests := []struct {
		name       string
		files      map[string][]byte // filename -> content
		candidates []string
		wantErr    bool
		wantData   []byte
	}{
		{
			name:       "extract by primary name",
			files:      map[string][]byte{"archcore": binaryContent},
			candidates: []string{"archcore", "cli"},
			wantErr:    false,
			wantData:   binaryContent,
		},
		{
			name:       "fallback to secondary name",
			files:      map[string][]byte{"cli": binaryContent},
			candidates: []string{"archcore", "cli"},
			wantErr:    false,
			wantData:   binaryContent,
		},
		{
			name:       "binary in subdirectory",
			files:      map[string][]byte{"archcore_v1.0.0_linux_amd64/archcore": binaryContent},
			candidates: []string{"archcore"},
			wantErr:    false,
			wantData:   binaryContent,
		},
		{
			name:       "binary not found",
			files:      map[string][]byte{"README.md": []byte("readme")},
			candidates: []string{"archcore", "cli"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archive := createTarGz(t, tt.files)

			got, err := ExtractBinary(archive, tt.candidates...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ExtractBinary() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !bytes.Equal(got, tt.wantData) {
				t.Errorf("ExtractBinary() data mismatch: got %q, want %q", got, tt.wantData)
			}
		})
	}
}

func TestArchiveName(t *testing.T) {
	tests := []struct {
		name       string
		binaryName string
		goos       string
		goarch     string
		want       string
	}{
		{"darwin arm64", "archcore", "darwin", "arm64", "archcore_darwin_arm64.tar.gz"},
		{"linux amd64", "archcore", "linux", "amd64", "archcore_linux_amd64.tar.gz"},
		{"windows amd64", "archcore", "windows", "amd64", "archcore_windows_amd64.zip"},
		{"windows arm64", "archcore", "windows", "arm64", "archcore_windows_arm64.zip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ArchiveName(tt.binaryName, tt.goos, tt.goarch)
			if got != tt.want {
				t.Errorf("ArchiveName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []int
		wantPre string
	}{
		{"simple", "1.2.3", []int{1, 2, 3}, ""},
		{"zeroes", "0.0.1", []int{0, 0, 1}, ""},
		{"multi-digit", "10.20.30", []int{10, 20, 30}, ""},
		{"alpha pre-release", "1.2.3-alpha.1", []int{1, 2, 3}, "alpha.1"},
		{"beta pre-release", "0.0.1-beta.2", []int{0, 0, 1}, "beta.2"},
		{"non-numeric", "invalid", nil, ""},
		{"too few parts", "1.2", nil, ""},
		{"non-integer patch", "1.2.x", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotPre := parseSemver(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("parseSemver(%q) = %v, want nil", tt.input, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("parseSemver(%q) = nil, want %v", tt.input, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("parseSemver(%q)[%d] = %d, want %d", tt.input, i, got[i], tt.want[i])
				}
			}
			if gotPre != tt.wantPre {
				t.Errorf("parseSemver(%q) pre-release = %q, want %q", tt.input, gotPre, tt.wantPre)
			}
		})
	}
}

func TestComparePreRelease(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"equal", "alpha.1", "alpha.1", 0},
		{"numeric bump", "alpha.7", "alpha.8", -1},
		{"numeric reverse", "alpha.8", "alpha.7", 1},
		{"alpha vs beta", "alpha.1", "beta.1", -1},
		{"numeric vs alpha", "1", "alpha", -1},
		{"shorter is less", "alpha", "alpha.1", -1},
		{"longer is more", "alpha.1", "alpha", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := comparePreRelease(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("comparePreRelease(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// createTarGz builds a tar.gz archive in memory from a map of filename -> content.
func createTarGz(t *testing.T, files map[string][]byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o755,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("writing tar header for %s: %v", name, err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatalf("writing tar content for %s: %v", name, err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("closing tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("closing gzip writer: %v", err)
	}

	return buf.Bytes()
}

// rewriteTransport rewrites all request URLs to point at a test server.
type rewriteTransport struct {
	base   http.RoundTripper
	target string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	parsed, _ := url.Parse(t.target)
	req.URL.Scheme = parsed.Scheme
	req.URL.Host = parsed.Host
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

func TestAtomicReplace(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "binary")

	// Create initial file.
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatalf("creating initial file: %v", err)
	}

	// Replace it. A nil probe is the manual `archcore update` path.
	newData := []byte("new content")
	if err := atomicReplace(context.Background(), target, newData, nil); err != nil {
		t.Fatalf("atomicReplace() error: %v", err)
	}

	// Verify content.
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading replaced file: %v", err)
	}
	if !bytes.Equal(got, newData) {
		t.Errorf("file content = %q, want %q", got, newData)
	}

	// Verify permissions.
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat replaced file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o755 {
		t.Errorf("file permissions = %o, want 755", perm)
	}
}

func TestAtomicReplace_NonexistentDir(t *testing.T) {
	target := filepath.Join(t.TempDir(), "nonexistent", "binary")

	err := atomicReplace(context.Background(), target, []byte("data"), nil)
	if err == nil {
		t.Fatal("expected error for nonexistent directory, got nil")
	}
}

func TestExtractBinary_CorruptArchive(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"not gzip", []byte("not a gzip archive")},
		{"truncated gzip header", []byte{0x1f, 0x8b}},
		{"empty input", []byte{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExtractBinary(tt.data, "archcore")
			if err == nil {
				t.Fatal("expected error for corrupt archive, got nil")
			}
		})
	}
}

func TestApply(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho archcore v2.0.0")
	archiveName := ArchiveName("archcore", runtime.GOOS, runtime.GOARCH)
	archiveData := createTarGz(t, map[string][]byte{"archcore": binaryContent})
	checksum := sha256.Sum256(archiveData)
	checksumLine := fmt.Sprintf("%x  %s\n", checksum, archiveName)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, archiveName):
			w.Write(archiveData)
		case strings.HasSuffix(r.URL.Path, "checksums.txt"):
			w.Write([]byte(checksumLine))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	fakeBinary := filepath.Join(dir, "archcore")
	if err := os.WriteFile(fakeBinary, []byte("old binary"), 0o755); err != nil {
		t.Fatalf("creating fake binary: %v", err)
	}

	u := &Updater{
		CurrentVersion: "v1.0.0",
		GitHubRepo:     "archcore-ai/cli",
		BinaryName:     "archcore",
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{target: srv.URL},
		},
		ExecPath: fakeBinary,
	}

	if err := u.Apply(context.Background(), "v2.0.0"); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	got, err := os.ReadFile(fakeBinary)
	if err != nil {
		t.Fatalf("reading replaced binary: %v", err)
	}
	if !bytes.Equal(got, binaryContent) {
		t.Errorf("binary content mismatch: got %d bytes, want %d bytes", len(got), len(binaryContent))
	}

	info, err := os.Stat(fakeBinary)
	if err != nil {
		t.Fatalf("stat binary: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o755 {
		t.Errorf("binary permissions = %o, want 755", perm)
	}
}

func TestApply_ChecksumMismatch(t *testing.T) {
	archiveName := ArchiveName("archcore", runtime.GOOS, runtime.GOARCH)
	archiveData := createTarGz(t, map[string][]byte{"archcore": []byte("binary")})
	badChecksumLine := fmt.Sprintf("%s  %s\n",
		"0000000000000000000000000000000000000000000000000000000000000000", archiveName)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, archiveName):
			w.Write(archiveData)
		case strings.HasSuffix(r.URL.Path, "checksums.txt"):
			w.Write([]byte(badChecksumLine))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	fakeBinary := filepath.Join(dir, "archcore")
	if err := os.WriteFile(fakeBinary, []byte("old"), 0o755); err != nil {
		t.Fatalf("creating fake binary: %v", err)
	}

	u := &Updater{
		CurrentVersion: "v1.0.0",
		GitHubRepo:     "archcore-ai/cli",
		BinaryName:     "archcore",
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{target: srv.URL},
		},
		ExecPath: fakeBinary,
	}

	err := u.Apply(context.Background(), "v2.0.0")
	if err == nil {
		t.Fatal("expected checksum mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("expected checksum mismatch error, got: %v", err)
	}
}

func TestApply_DownloadFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	fakeBinary := filepath.Join(dir, "archcore")
	if err := os.WriteFile(fakeBinary, []byte("old"), 0o755); err != nil {
		t.Fatalf("creating fake binary: %v", err)
	}

	u := &Updater{
		CurrentVersion: "v1.0.0",
		GitHubRepo:     "archcore-ai/cli",
		BinaryName:     "archcore",
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{target: srv.URL},
		},
		ExecPath: fakeBinary,
	}

	err := u.Apply(context.Background(), "v2.0.0")
	if err == nil {
		t.Fatal("expected download error, got nil")
	}
	if !strings.Contains(err.Error(), "downloading") {
		t.Errorf("expected downloading error, got: %v", err)
	}
}

func TestExtractBinary_Zip(t *testing.T) {
	binaryContent := []byte("MZ\x90\x00fake-windows-binary")

	tests := []struct {
		name       string
		files      map[string][]byte
		candidates []string
		wantErr    bool
		wantData   []byte
	}{
		{
			name:       "extract by primary name",
			files:      map[string][]byte{"archcore.exe": binaryContent},
			candidates: []string{"archcore.exe", "cli.exe"},
			wantData:   binaryContent,
		},
		{
			name:       "fallback to secondary name",
			files:      map[string][]byte{"cli.exe": binaryContent},
			candidates: []string{"archcore.exe", "cli.exe"},
			wantData:   binaryContent,
		},
		{
			name:       "binary in subdirectory",
			files:      map[string][]byte{"archcore_v1.0.0_windows_amd64/archcore.exe": binaryContent},
			candidates: []string{"archcore.exe"},
			wantData:   binaryContent,
		},
		{
			name:       "binary not found",
			files:      map[string][]byte{"README.md": []byte("readme")},
			candidates: []string{"archcore.exe", "cli.exe"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archive := createZip(t, tt.files)

			got, err := ExtractBinary(archive, tt.candidates...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ExtractBinary() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !bytes.Equal(got, tt.wantData) {
				t.Errorf("ExtractBinary() data mismatch: got %q, want %q", got, tt.wantData)
			}
		})
	}
}

func TestExtractBinary_CorruptZip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		// PK\x03\x04 prefix passes magic-byte detection but the rest is garbage.
		{"zip magic with no body", []byte{'P', 'K', 0x03, 0x04}},
		{"zip magic with truncated body", []byte("PK\x03\x04corrupted-payload")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExtractBinary(tt.data, "archcore.exe")
			if err == nil {
				t.Fatal("expected error for corrupt zip, got nil")
			}
		})
	}
}

func TestIsZipArchive(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"zip magic", []byte{'P', 'K', 0x03, 0x04, 0x00}, true},
		{"zip magic exact 4 bytes", []byte{'P', 'K', 0x03, 0x04}, true},
		{"gzip magic", []byte{0x1f, 0x8b, 0x08, 0x00}, false},
		{"tar header letters", []byte("ustar0000"), false},
		{"too short", []byte{'P', 'K', 0x03}, false},
		{"empty", []byte{}, false},
		{"nil", nil, false},
		{"zip end-of-central-dir signature", []byte{'P', 'K', 0x05, 0x06}, false}, // we only accept local-file-header
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isZipArchive(tt.data); got != tt.want {
				t.Errorf("isZipArchive(%v) = %v, want %v", tt.data, got, tt.want)
			}
		})
	}
}

func TestBinaryCandidates(t *testing.T) {
	tests := []struct {
		name       string
		binaryName string
		repo       string
		goos       string
		want       []string
	}{
		{"linux", "archcore", "archcore-ai/cli", "linux", []string{"archcore", "cli"}},
		{"darwin", "archcore", "archcore-ai/cli", "darwin", []string{"archcore", "cli"}},
		{"windows adds .exe to both", "archcore", "archcore-ai/cli", "windows", []string{"archcore.exe", "cli.exe"}},
		{"repo without slash uses path basename", "archcore", "cli", "linux", []string{"archcore", "cli"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := binaryCandidates(tt.binaryName, tt.repo, tt.goos)
			if len(got) != len(tt.want) {
				t.Fatalf("binaryCandidates() = %v, want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("binaryCandidates()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestAtomicReplace_Windows exercises the rename-aside branch used on Windows
// where the running .exe cannot be overwritten in place. The branch is reached
// only when runtime.GOOS == "windows", so this test is skipped elsewhere.
func TestAtomicReplace_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific atomicReplace branch")
	}

	t.Run("happy path leaves no .old", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "archcore.exe")
		if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
			t.Fatalf("creating target: %v", err)
		}

		newData := []byte("new content")
		if err := atomicReplace(context.Background(), target, newData, nil); err != nil {
			t.Fatalf("atomicReplace() error: %v", err)
		}

		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("reading target: %v", err)
		}
		if !bytes.Equal(got, newData) {
			t.Errorf("target content = %q, want %q", got, newData)
		}

		// The test process does not hold a lock on the temp file, so the
		// best-effort aside cleanup at the end of commitStaged should succeed.
		// The aside is per-attempt, so match the prefix rather than one name.
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "archcore.exe.old.") {
				t.Errorf("expected aside %s to be cleaned up", e.Name())
			}
		}
	})

	t.Run("target does not exist yet", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "archcore.exe")

		newData := []byte("fresh install")
		if err := atomicReplace(context.Background(), target, newData, nil); err != nil {
			t.Fatalf("atomicReplace() error: %v", err)
		}

		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("reading target: %v", err)
		}
		if !bytes.Equal(got, newData) {
			t.Errorf("target content = %q, want %q", got, newData)
		}
	})
}

// TestApply_ZipArchive is the regression test for the Windows 404 bug: prior
// to the fix, Apply downloaded a .tar.gz on every platform. Forcing the goos
// argument exercises the zip code path independently of the host runtime.GOOS,
// so the test runs on all CI platforms.
func TestApply_ZipArchive(t *testing.T) {
	binaryContent := []byte("MZ\x90\x00fake-windows-binary")
	archiveName := ArchiveName("archcore", "windows", "amd64")
	if !strings.HasSuffix(archiveName, ".zip") {
		t.Fatalf("expected zip archive name for windows, got %q", archiveName)
	}
	archiveData := createZip(t, map[string][]byte{"archcore.exe": binaryContent})
	checksum := sha256.Sum256(archiveData)
	checksumLine := fmt.Sprintf("%x  %s\n", checksum, archiveName)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, archiveName):
			w.Write(archiveData)
		case strings.HasSuffix(r.URL.Path, "checksums.txt"):
			w.Write([]byte(checksumLine))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Exercise the full download + verify + extract pipeline against a zip
	// payload by calling the package primitives directly. We avoid Apply()
	// because it bakes in runtime.GOOS; the regression is in extraction +
	// archive-name resolution, both of which are covered here.
	u := &Updater{
		CurrentVersion: "v1.0.0",
		GitHubRepo:     "archcore-ai/cli",
		BinaryName:     "archcore",
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{target: srv.URL},
		},
	}

	gotArchive, err := u.download(context.Background(), "v2.0.0", archiveName)
	if err != nil {
		t.Fatalf("download archive: %v", err)
	}
	gotChecksums, err := u.download(context.Background(), "v2.0.0", "checksums.txt")
	if err != nil {
		t.Fatalf("download checksums: %v", err)
	}
	if err := VerifyChecksum(gotArchive, gotChecksums, archiveName); err != nil {
		t.Fatalf("VerifyChecksum: %v", err)
	}
	got, err := ExtractBinary(gotArchive, binaryCandidates("archcore", "archcore-ai/cli", "windows")...)
	if err != nil {
		t.Fatalf("ExtractBinary: %v", err)
	}
	if !bytes.Equal(got, binaryContent) {
		t.Errorf("extracted binary mismatch: got %q, want %q", got, binaryContent)
	}
}

// createZip builds a zip archive in memory from a map of filename -> content.
func createZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for name, content := range files {
		fw, err := zw.Create(name)
		if err != nil {
			t.Fatalf("creating zip entry %s: %v", name, err)
		}
		if _, err := fw.Write(content); err != nil {
			t.Fatalf("writing zip content for %s: %v", name, err)
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("closing zip writer: %v", err)
	}

	return buf.Bytes()
}

func TestDownload_SizeLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := bytes.Repeat([]byte("a"), maxChecksumsSize+1)
		w.Write(data)
	}))
	defer srv.Close()

	u := &Updater{
		GitHubRepo: "test/repo",
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{target: srv.URL},
		},
	}

	_, err := u.download(context.Background(), "v1.0.0", "checksums.txt")
	if err == nil {
		t.Fatal("expected size limit error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds size limit") {
		t.Errorf("expected size limit error, got: %v", err)
	}
}

// TestAtomicReplace_RenameFailureCleansTmp pins the non-Windows rollback path:
// when the final rename fails, the temp binary is removed and the error
// surfaces — the interrupted-update case.
func TestAtomicReplace_RenameFailureCleansTmp(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("non-Windows rename path")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "occupied")
	// A non-empty directory at the target makes os.Rename fail.
	if err := os.MkdirAll(filepath.Join(target, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := atomicReplace(context.Background(), target, []byte("binary"), nil)
	if err == nil {
		t.Fatal("expected error renaming onto a non-empty directory")
	}
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatal(readErr)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp.") {
			t.Errorf("temp binary %s left behind after failed rename", e.Name())
		}
	}
}

// releaseServer serves one release: the platform archive carrying
// binaryContent under the expected binary name, plus a matching checksums.txt.
// Apply bakes in runtime.GOOS, so the archive format follows it.
func releaseServer(t *testing.T, binaryContent []byte) *httptest.Server {
	t.Helper()

	archiveName := ArchiveName("archcore", runtime.GOOS, runtime.GOARCH)
	archiveData := createTarGz(t, map[string][]byte{"archcore": binaryContent})
	if runtime.GOOS == "windows" {
		archiveData = createZip(t, map[string][]byte{"archcore.exe": binaryContent})
	}
	checksumLine := fmt.Sprintf("%x  %s\n", sha256.Sum256(archiveData), archiveName)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, archiveName):
			_, _ = w.Write(archiveData)
		case strings.HasSuffix(r.URL.Path, "checksums.txt"):
			_, _ = w.Write([]byte(checksumLine))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// stagedFiles returns the names of staged temporaries left in dir. An
// abandoned attempt must leave none.
func stagedFiles(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	var left []string
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp.") {
			left = append(left, e.Name())
		}
	}
	return left
}

// TestApply_PreCommitProbe runs the real healthProbe against the staged file,
// so every row reproduces a failure mode a downloaded binary actually has. It
// needs a POSIX shell and permission to exec a freshly staged file, so it is
// skipped on Windows; the seam itself is covered there by
// TestApply_PreCommitProbeSeam, which starts no child process.
func TestApply_PreCommitProbe(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs a POSIX shell to stage a probe-able binary")
	}

	tests := []struct {
		name        string
		staged      string
		wantErr     bool
		errContains string
	}{
		{
			name:   "probe exits zero and the replacement commits",
			staged: "#!/bin/sh\nexit 0\n",
		},
		{
			name:        "probe exits nonzero",
			staged:      "#!/bin/sh\nexit 3\n",
			wantErr:     true,
			errContains: "--version",
		},
		{
			// exec replaces the shell, so the deadline kill lands on the
			// sleeping process itself rather than on a shell that outlives it.
			name:        "probe times out",
			staged:      "#!/bin/sh\nexec sleep 30\n",
			wantErr:     true,
			errContains: "did not finish within",
		},
		{
			// Not a recognized executable format, so execve fails outright —
			// the "fails to start" arm, distinct from a nonzero exit.
			name:        "probe cannot start",
			staged:      "\x00\x01\x02 not an executable",
			wantErr:     true,
			errContains: "probing staged binary",
		},
	}

	const original = "old binary"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := releaseServer(t, []byte(tt.staged))

			dir := t.TempDir()
			fakeBinary := filepath.Join(dir, "archcore")
			if err := os.WriteFile(fakeBinary, []byte(original), 0o755); err != nil {
				t.Fatalf("creating fake binary: %v", err)
			}

			u := &Updater{
				CurrentVersion: "v1.0.0",
				GitHubRepo:     "archcore-ai/cli",
				BinaryName:     "archcore",
				HTTPClient:     &http.Client{Transport: &rewriteTransport{target: srv.URL}},
				ExecPath:       fakeBinary,
				PreCommitProbe: healthProbe,
			}

			err := u.Apply(context.Background(), "v2.0.0")

			got, readErr := os.ReadFile(fakeBinary)
			if readErr != nil {
				t.Fatalf("reading binary after Apply: %v", readErr)
			}

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("Apply() error: %v", err)
				}
				if string(got) != tt.staged {
					t.Errorf("binary content = %q, want the staged payload", got)
				}
				if left := stagedFiles(t, dir); len(left) != 0 {
					t.Errorf("staged files left after a successful commit: %v", left)
				}
				return
			}

			if err == nil {
				t.Fatal("expected the probe to abandon the replacement, got nil")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error = %v, want it to contain %q", err, tt.errContains)
			}
			// A probe refusal is a replace failure: the pipeline stopped with a
			// staged file it declined to commit.
			if stage, ok := StageOf(err); !ok || stage != StageReplace {
				t.Errorf("StageOf(err) = (%q, %v), want (%q, true)", stage, ok, StageReplace)
			}
			if string(got) != original {
				t.Errorf("binary content = %q, want the original %q untouched", got, original)
			}
			if left := stagedFiles(t, dir); len(left) != 0 {
				t.Errorf("staged files left after an abandoned attempt: %v", left)
			}
		})
	}
}

// TestApply_PreCommitProbeSeam covers the seam without a child process: a probe
// that refuses abandons the replacement, and a nil probe — the manual
// `archcore update` path — replaces exactly as before.
func TestApply_PreCommitProbeSeam(t *testing.T) {
	newContent := []byte("new binary")
	const original = "old binary"

	newUpdater := func(t *testing.T, probe func(context.Context, string) error) (*Updater, string, string) {
		t.Helper()
		srv := releaseServer(t, newContent)
		dir := t.TempDir()
		fakeBinary := filepath.Join(dir, "archcore")
		if err := os.WriteFile(fakeBinary, []byte(original), 0o755); err != nil {
			t.Fatalf("creating fake binary: %v", err)
		}
		return &Updater{
			CurrentVersion: "v1.0.0",
			GitHubRepo:     "archcore-ai/cli",
			BinaryName:     "archcore",
			HTTPClient:     &http.Client{Transport: &rewriteTransport{target: srv.URL}},
			ExecPath:       fakeBinary,
			PreCommitProbe: probe,
		}, dir, fakeBinary
	}

	t.Run("refusing probe abandons the replacement", func(t *testing.T) {
		var probedPath string
		u, dir, fakeBinary := newUpdater(t, func(_ context.Context, stagedPath string) error {
			probedPath = stagedPath
			// The staged file exists while the probe runs — that is the whole
			// point of running between the synced write and the rename.
			if _, err := os.Stat(stagedPath); err != nil {
				t.Errorf("staged file missing during the probe: %v", err)
			}
			return errors.New("probe refused")
		})

		err := u.Apply(context.Background(), "v2.0.0")
		if err == nil {
			t.Fatal("expected the probe refusal to fail Apply")
		}
		if !strings.Contains(err.Error(), "probe refused") {
			t.Errorf("error = %v, want it to carry the probe's message", err)
		}
		if stage, ok := StageOf(err); !ok || stage != StageReplace {
			t.Errorf("StageOf(err) = (%q, %v), want (%q, true)", stage, ok, StageReplace)
		}
		if !strings.Contains(filepath.Base(probedPath), ".tmp.") {
			t.Errorf("probe received %q, want the staged temporary", probedPath)
		}
		got, readErr := os.ReadFile(fakeBinary)
		if readErr != nil {
			t.Fatalf("reading binary: %v", readErr)
		}
		if string(got) != original {
			t.Errorf("binary content = %q, want the original %q untouched", got, original)
		}
		if left := stagedFiles(t, dir); len(left) != 0 {
			t.Errorf("staged files left after an abandoned attempt: %v", left)
		}
	})

	t.Run("nil probe replaces as before", func(t *testing.T) {
		u, dir, fakeBinary := newUpdater(t, nil)

		if err := u.Apply(context.Background(), "v2.0.0"); err != nil {
			t.Fatalf("Apply() error: %v", err)
		}
		got, err := os.ReadFile(fakeBinary)
		if err != nil {
			t.Fatalf("reading binary: %v", err)
		}
		if !bytes.Equal(got, newContent) {
			t.Errorf("binary content = %q, want %q", got, newContent)
		}
		if left := stagedFiles(t, dir); len(left) != 0 {
			t.Errorf("staged files left after a successful commit: %v", left)
		}
	})
}

// TestHealthProbe_TimeoutIgnoresCallerDeadline pins the reason the bound is
// derived from context.WithoutCancel: the unattended policy runs under a 120 s
// ceiling that must not interrupt the replacement, and the probe sits between
// the synced write and the rename. An already-cancelled caller context must not
// reach the child process.
func TestHealthProbe_TimeoutIgnoresCallerDeadline(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs a POSIX shell to stage a probe-able binary")
	}
	t.Parallel()

	staged := filepath.Join(t.TempDir(), "archcore")
	if err := os.WriteFile(staged, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("staging probe target: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := healthProbe(ctx, staged); err != nil {
		t.Errorf("healthProbe() error on a cancelled caller context: %v", err)
	}
}

// TestStageBinary_SweepsAttemptLeftovers covers the per-attempt sweep: staged
// temporaries and Windows asides from other pids go, a leftover that cannot be
// removed is skipped instead of failing the run, and a name that only looks
// similar survives — unattended-update.spec §14.
func TestStageBinary_SweepsAttemptLeftovers(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "archcore")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatalf("creating target: %v", err)
	}

	// Suffixes that cannot equal this process's pid, so the rows stay
	// deterministic whatever pid the test runs under.
	otherPid := os.Getpid() + 1
	stale := []string{
		fmt.Sprintf("archcore.tmp.%d", otherPid),
		fmt.Sprintf("archcore.old.%d", otherPid),
	}
	for _, name := range stale {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("stale"), 0o755); err != nil {
			t.Fatalf("creating leftover %s: %v", name, err)
		}
	}

	// A non-empty directory stands in for the file a live process still holds
	// on Windows: os.Remove refuses it, and the sweep must move on. os.ReadDir
	// returns names sorted, so the leftover below sorts after this one and only
	// disappears if the sweep kept going past the failure.
	stuck := filepath.Join(dir, "archcore.tmp.held")
	if err := os.MkdirAll(filepath.Join(stuck, "child"), 0o755); err != nil {
		t.Fatalf("creating unremovable leftover: %v", err)
	}
	afterStuck := fmt.Sprintf("archcore.tmp.z%d", otherPid)
	if err := os.WriteFile(filepath.Join(dir, afterStuck), []byte("stale"), 0o755); err != nil {
		t.Fatalf("creating leftover %s: %v", afterStuck, err)
	}
	stale = append(stale, afterStuck)

	// Prefix-adjacent names that are not attempt leftovers.
	keep := []string{"archcore.tmpfile", "archcore.older", "archcore2.tmp.1"}
	for _, name := range keep {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("keep"), 0o644); err != nil {
			t.Fatalf("creating neighbour %s: %v", name, err)
		}
	}

	// The sweep leaves a young leftover alone, so every name it is expected to
	// remove has to be older than the grace period.
	for _, name := range append(slices.Clone(stale), "archcore.tmp.held") {
		ageBeyondSweepGrace(t, filepath.Join(dir, name))
	}

	newData := []byte("new content")
	if err := atomicReplace(context.Background(), target, newData, nil); err != nil {
		t.Fatalf("atomicReplace() error: %v", err)
	}

	for _, name := range stale {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Errorf("leftover %s survived the sweep, stat err = %v", name, err)
		}
	}
	if _, err := os.Stat(stuck); err != nil {
		t.Errorf("unremovable leftover should have been skipped, stat err = %v", err)
	}
	for _, name := range keep {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("neighbour %s must not be swept, stat err = %v", name, err)
		}
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading target: %v", err)
	}
	if !bytes.Equal(got, newData) {
		t.Errorf("target content = %q, want %q", got, newData)
	}
}

// TestStageBinary_PerAttemptNames pins the pid in both names. A fixed aside
// collides when a second update starts while an older process still holds the
// previous one. Both wanted names are spelled out as literals rather than built
// from the package's own constants, so renaming a constant cannot rename the
// expectation with it.
func TestStageBinary_PerAttemptNames(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "archcore")

	tmpPath, err := stageBinary(target, []byte("staged"))
	if err != nil {
		t.Fatalf("stageBinary() error: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(tmpPath) })

	wantTmp := filepath.Join(dir, fmt.Sprintf("archcore.tmp.%d", os.Getpid()))
	if tmpPath != wantTmp {
		t.Errorf("stageBinary() = %q, want %q", tmpPath, wantTmp)
	}
	wantAside := filepath.Join(dir, fmt.Sprintf("archcore.old.%d", os.Getpid()))
	if got := asidePath(target); got != wantAside {
		t.Errorf("asidePath() = %q, want %q", got, wantAside)
	}
	if _, err := os.Stat(tmpPath); err != nil {
		t.Errorf("staged file missing: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("stageBinary must not touch the target, stat err = %v", err)
	}
}

// TestSweepCoversTheAsideItWrites closes the loop between the two names: the
// sweep must remove a file the aside constructor produced. Pinning each name
// on its own leaves a rename of one of them green.
func TestSweepCoversTheAsideItWrites(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "archcore")
	aside := asidePath(target)
	if err := os.WriteFile(aside, []byte("aside"), 0o755); err != nil {
		t.Fatalf("creating aside: %v", err)
	}
	ageBeyondSweepGrace(t, aside)

	sweepAttemptLeftovers(dir, "archcore")

	if _, err := os.Stat(aside); !os.IsNotExist(err) {
		t.Errorf("the sweep left the aside %q it writes itself, stat err = %v", aside, err)
	}
}

// ageBeyondSweepGrace backdates a file past leftoverGrace, so the sweep reads it
// as abandoned rather than as an attempt still in flight.
func ageBeyondSweepGrace(t *testing.T, path string) {
	t.Helper()
	old := time.Now().Add(-2 * leftoverGrace)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("backdating %s: %v", path, err)
	}
}

// TestSweepSparesAnAttemptStillInFlight is the reason the grace period exists.
//
// "A file another live process still holds fails to delete" is a Windows
// property; on Unix os.Remove unlinks the name whatever else has it open. So a
// typed `archcore update` starting while the unattended policy sits in its
// pre-commit probe would unlink the policy's staged binary, and the policy
// would then fail at commitStaged with ENOENT — a failure this surface caused
// itself, reported to telemetry as a real one.
func TestSweepSparesAnAttemptStillInFlight(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "archcore")

	// A peer process staged this a moment ago and has not committed it yet.
	inFlight := filepath.Join(dir, fmt.Sprintf("archcore.tmp.%d", os.Getpid()+1))
	if err := os.WriteFile(inFlight, []byte("staged by a peer"), 0o755); err != nil {
		t.Fatalf("creating the in-flight staged file: %v", err)
	}

	abandoned := filepath.Join(dir, fmt.Sprintf("archcore.tmp.%d", os.Getpid()+2))
	if err := os.WriteFile(abandoned, []byte("abandoned"), 0o755); err != nil {
		t.Fatalf("creating the abandoned staged file: %v", err)
	}
	ageBeyondSweepGrace(t, abandoned)

	sweepAttemptLeftovers(dir, filepath.Base(target))

	if _, err := os.Stat(inFlight); err != nil {
		t.Errorf("the sweep unlinked a staged binary a peer was still committing, stat err = %v", err)
	}
	if _, err := os.Stat(abandoned); !os.IsNotExist(err) {
		t.Errorf("the grace period must not spare a genuinely abandoned leftover, stat err = %v", err)
	}
}

// TestCommitViaAside exercises the two-rename commit and its rollback on every
// platform. commitStaged reaches it only when runtime.GOOS is "windows", and
// the test suite runs on Linux — so calling it directly is the only way the
// aside, the second rename, and the rollback are ever executed. Without this
// the whole Windows replacement path, including the guarantee that a failed
// second rename restores the original binary, ships unexecuted.
func TestCommitViaAside(t *testing.T) {
	const original = "old binary"

	t.Run("replaces the target and cleans the aside", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "archcore.exe")
		if err := os.WriteFile(target, []byte(original), 0o755); err != nil {
			t.Fatalf("creating target: %v", err)
		}
		tmpPath, err := stageBinary(target, []byte("new content"))
		if err != nil {
			t.Fatalf("stageBinary() error: %v", err)
		}

		if err := commitViaAside(target, tmpPath); err != nil {
			t.Fatalf("commitViaAside() error: %v", err)
		}

		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("reading target: %v", err)
		}
		if string(got) != "new content" {
			t.Errorf("target content = %q, want %q", got, "new content")
		}
		// This process holds no lock on the aside, so the best-effort cleanup
		// succeeds here and no leftover survives the commit.
		if _, err := os.Stat(asidePath(target)); !os.IsNotExist(err) {
			t.Errorf("aside survived the commit, stat err = %v", err)
		}
		if left := stagedFiles(t, dir); len(left) != 0 {
			t.Errorf("staged files left after a successful commit: %v", left)
		}
	})

	t.Run("absent target is a fresh install", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "archcore.exe")
		tmpPath, err := stageBinary(target, []byte("fresh install"))
		if err != nil {
			t.Fatalf("stageBinary() error: %v", err)
		}

		// The first rename fails with fs.ErrNotExist, which is not a failure:
		// there is no running binary to move aside.
		if err := commitViaAside(target, tmpPath); err != nil {
			t.Fatalf("commitViaAside() error: %v", err)
		}
		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("reading target: %v", err)
		}
		if string(got) != "fresh install" {
			t.Errorf("target content = %q, want %q", got, "fresh install")
		}
	})

	t.Run("failed second rename rolls the original back", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "archcore.exe")
		if err := os.WriteFile(target, []byte(original), 0o755); err != nil {
			t.Fatalf("creating target: %v", err)
		}
		// A staged path that does not exist makes the second rename fail after
		// the target has already been moved aside — the window in which the
		// target name is absent and the rollback is the only thing that puts a
		// binary back.
		missing := filepath.Join(dir, "archcore.exe.tmp.absent")

		err := commitViaAside(target, missing)
		if err == nil {
			t.Fatal("expected an error when the staged file is missing")
		}
		if !strings.Contains(err.Error(), "renaming temp to target") {
			t.Errorf("error = %v, want it to name the failed rename", err)
		}

		got, readErr := os.ReadFile(target)
		if readErr != nil {
			t.Fatalf("the rollback did not restore the target: %v", readErr)
		}
		if string(got) != original {
			t.Errorf("target content = %q, want the original %q", got, original)
		}
		if _, statErr := os.Stat(asidePath(target)); !os.IsNotExist(statErr) {
			t.Errorf("aside survived the rollback, stat err = %v", statErr)
		}
	})
}

// TestCommitStaged_PosixUsesOneRename pins the dispatch in commitStaged: away
// from Windows the commit is a single rename, which is atomic, and the aside is
// never touched. The two-rename path leaves the target name absent between the
// renames — an accepted platform gap on Windows and a regression anywhere else.
//
// A non-empty directory parked on the aside name is the detector: os.Rename
// cannot move the target onto it, so a commit that reaches for the aside fails
// here while a single rename never looks at the name at all.
func TestCommitStaged_PosixUsesOneRename(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows commits through the aside by design")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "archcore")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatalf("creating target: %v", err)
	}
	// The sweep cannot remove a non-empty directory, so it survives staging.
	if err := os.MkdirAll(filepath.Join(asidePath(target), "child"), 0o755); err != nil {
		t.Fatalf("parking a directory on the aside name: %v", err)
	}

	newData := []byte("new content")
	if err := atomicReplace(context.Background(), target, newData, nil); err != nil {
		t.Fatalf("atomicReplace() error: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading target: %v", err)
	}
	if !bytes.Equal(got, newData) {
		t.Errorf("target content = %q, want %q", got, newData)
	}
}

// TestProbeTimeoutBound pins the documented 3 s bound as a literal. The value
// is the spec's, not the code's — unattended-update.spec §15 — so a change to
// the constant has to be a deliberate edit here too.
func TestProbeTimeoutBound(t *testing.T) {
	t.Parallel()
	if probeTimeout != 3*time.Second {
		t.Errorf("probeTimeout = %s, want 3s", probeTimeout)
	}
}

// probeStreamsChildEnv switches this test binary into the child role for
// TestHealthProbe_DoesNotInheritOurStreams, and probeStreamsPathEnv hands the
// child the staged binary to probe. No machine sets either one, and the child's
// sentinel proves the branch was taken rather than silently skipped.
const (
	probeStreamsChildEnv = "ARCHCORE_TEST_PROBE_STREAMS_CHILD"
	probeStreamsPathEnv  = "ARCHCORE_TEST_PROBE_STREAMS_PATH"
)

// TestHealthProbe_DoesNotInheritOurStreams pins the reason healthProbe leaves
// Stdout and Stderr nil: os/exec then connects the staged binary to the null
// device instead of to ours, and `archcore mcp` owns stdout as a JSON-RPC
// stream that one stray line from a downloaded binary would corrupt.
//
// The check needs a parent whose own streams are observable, so the test
// re-execs itself: the child runs the probe against a binary that prints on
// both streams, and the parent — which owns the pipe — must see neither line.
func TestHealthProbe_DoesNotInheritOurStreams(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs a POSIX shell to stage a probe-able binary")
	}

	const (
		stdoutMarker = "PROBE-LEAKED-STDOUT"
		stderrMarker = "PROBE-LEAKED-STDERR"
		sentinel     = "PROBE-CHILD-RAN"
	)

	if os.Getenv(probeStreamsChildEnv) == "1" {
		if err := healthProbe(context.Background(), os.Getenv(probeStreamsPathEnv)); err != nil {
			t.Fatalf("healthProbe() error: %v", err)
		}
		fmt.Fprintln(os.Stderr, sentinel)
		return
	}

	staged := filepath.Join(t.TempDir(), "archcore")
	script := fmt.Sprintf("#!/bin/sh\necho %s\necho %s >&2\nexit 0\n", stdoutMarker, stderrMarker)
	if err := os.WriteFile(staged, []byte(script), 0o755); err != nil {
		t.Fatalf("staging probe target: %v", err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestHealthProbe_DoesNotInheritOurStreams$")
	cmd.Env = append(os.Environ(),
		probeStreamsChildEnv+"=1",
		probeStreamsPathEnv+"="+staged,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("re-exec failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), sentinel) {
		t.Fatalf("the child never ran the probe; output:\n%s", out)
	}
	if strings.Contains(string(out), stdoutMarker) {
		t.Errorf("the staged binary's stdout reached the caller's stdout:\n%s", out)
	}
	if strings.Contains(string(out), stderrMarker) {
		t.Errorf("the staged binary's stderr reached the caller's stderr:\n%s", out)
	}
}

// TestCheckLatest_ConnectionRefused pins the transport-error path.
func TestCheckLatest_ConnectionRefused(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close() // close immediately

	u := NewUpdater("v1.0.0", "archcore-ai/cli", "archcore")
	u.HTTPClient = &http.Client{Transport: &rewriteTransport{base: http.DefaultTransport, target: srv.URL}}

	if _, err := u.CheckLatest(context.Background()); err == nil {
		t.Fatal("expected error for closed server")
	}
}

// TestCheckLatest_ContextCancelled pins cancellation propagation.
func TestCheckLatest_ContextCancelled(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://github.com/archcore-ai/cli/releases/tag/v9.9.9")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	u := NewUpdater("v1.0.0", "archcore-ai/cli", "archcore")
	u.HTTPClient = &http.Client{Transport: &rewriteTransport{base: http.DefaultTransport, target: srv.URL}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := u.CheckLatest(ctx); err == nil {
		t.Fatal("expected error for cancelled context")
	}
}
