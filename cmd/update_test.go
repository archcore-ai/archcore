package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"archcore-cli/internal/update"
)

func TestUpdateCmd_AlreadyUpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.0.0"})
	}))
	defer srv.Close()

	out := runUpdateCmd(t, "v1.0.0", srv)

	if !strings.Contains(out, "Already up to date") {
		t.Errorf("expected 'Already up to date' in output, got: %s", out)
	}
	if !strings.Contains(out, "v1.0.0") {
		t.Errorf("expected version in output, got: %s", out)
	}
}

func TestUpdateCmd_UpdateAvailable(t *testing.T) {
	// This test exercises the tar.gz path; it currently runs only on
	// Linux/macOS CI. Windows CI builds a zip archive — see the zip-flavored
	// regression test in internal/update/update_test.go (TestApply_ZipArchive).
	if runtime.GOOS == "windows" {
		t.Skip("tar.gz path; see TestApply_ZipArchive for the windows regression test")
	}
	binaryContent := []byte("#!/bin/sh\necho archcore v2.0.0")
	archiveData := buildTestArchive(t, map[string][]byte{"archcore": binaryContent})
	archiveName := update.ArchiveName("archcore", runtime.GOOS, runtime.GOARCH)
	checksum := sha256.Sum256(archiveData)
	checksumLine := fmt.Sprintf("%x  %s\n", checksum, archiveName)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "releases/latest"):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"tag_name": "v2.0.0"})
		case strings.HasSuffix(r.URL.Path, archiveName):
			w.Write(archiveData)
		case strings.HasSuffix(r.URL.Path, "checksums.txt"):
			w.Write([]byte(checksumLine))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Create a fake binary for the updater to replace.
	dir := t.TempDir()
	fakeBinary := filepath.Join(dir, "archcore")
	if err := os.WriteFile(fakeBinary, []byte("old binary"), 0o755); err != nil {
		t.Fatalf("creating fake binary: %v", err)
	}

	u := &update.Updater{
		CurrentVersion: "v1.0.0",
		GitHubRepo:     "archcore-ai/cli",
		BinaryName:     "archcore",
		HTTPClient: &http.Client{
			Transport: &testRewriteTransport{target: srv.URL},
		},
		ExecPath: fakeBinary,
	}

	root := NewRootCmd("test")
	for _, cmd := range root.Commands() {
		if cmd.Use == "update" {
			root.RemoveCommand(cmd)
			break
		}
	}
	root.AddCommand(buildUpdateCmd("v1.0.0", u))

	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"update"})

	// Capture stdout since the command uses fmt.Println.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	oldStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	ctx := context.Background()
	root.SetContext(ctx)
	_ = root.Execute()
	w.Close()
	os.Stdout = oldStdout

	var out bytes.Buffer
	out.ReadFrom(r)
	output := out.String()

	if !strings.Contains(output, "Current: v1.0.0") {
		t.Errorf("expected 'Current: v1.0.0' in output, got: %s", output)
	}
	if !strings.Contains(output, "Latest:  v2.0.0") {
		t.Errorf("expected 'Latest:  v2.0.0' in output, got: %s", output)
	}
	if !strings.Contains(output, "Downloading") {
		t.Errorf("expected 'Downloading' in output, got: %s", output)
	}
	if strings.Contains(output, "Update failed") {
		t.Errorf("update should have succeeded, got: %s", output)
	}
	if !strings.Contains(output, "Checksum verified") {
		t.Errorf("expected 'Checksum verified' in output, got: %s", output)
	}
	if !strings.Contains(output, "Updated to v2.0.0") {
		t.Errorf("expected 'Updated to v2.0.0' in output, got: %s", output)
	}
}

func TestUpdateCmd_DevVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "releases/latest"):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.0.0"})
		default:
			// Return 404 for downloads — we just want to verify it attempts update.
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	out := runUpdateCmd(t, "vdev", srv)

	// Dev version should always try to update (not show "Already up to date").
	if strings.Contains(out, "Already up to date") {
		t.Errorf("dev version should not show 'Already up to date', got: %s", out)
	}
	if !strings.Contains(out, "Downloading") {
		t.Errorf("expected download attempt for dev version, got: %s", out)
	}
}

func TestUpdateCmd_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	out := runUpdateCmd(t, "v1.0.0", srv)

	if !strings.Contains(out, "Could not check for updates") {
		t.Errorf("expected error message in output, got: %s", out)
	}
}

// runUpdateCmd executes the update command with a test server and captures stdout.
func runUpdateCmd(t *testing.T, version string, srv *httptest.Server) string {
	t.Helper()

	client := &http.Client{
		Transport: &testRewriteTransport{target: srv.URL},
	}

	root := NewRootCmd("test")
	// Replace the update command with one using our test client.
	for _, cmd := range root.Commands() {
		if cmd.Use == "update" {
			root.RemoveCommand(cmd)
			break
		}
	}
	root.AddCommand(newUpdateCmdWithClient(version, client))

	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"update"})

	// Capture stdout since the command uses fmt.Println.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	oldStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	ctx := context.Background()
	root.SetContext(ctx)
	_ = root.Execute()
	w.Close()
	os.Stdout = oldStdout

	var out bytes.Buffer
	out.ReadFrom(r)

	return out.String()
}

// testRewriteTransport rewrites all request URLs to point at a test server.
type testRewriteTransport struct {
	target string
}

func (t *testRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	parsed, _ := url.Parse(t.target)
	req.URL.Scheme = parsed.Scheme
	req.URL.Host = parsed.Host
	return http.DefaultTransport.RoundTrip(req)
}

// buildTestArchive creates a tar.gz archive for testing.
func buildTestArchive(t *testing.T, files map[string][]byte) []byte {
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
			t.Fatalf("writing tar header: %v", err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatalf("writing tar content: %v", err)
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
