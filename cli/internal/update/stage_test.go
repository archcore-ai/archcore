package update

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestStageVocabulary_IsFrozen pins the wire values, not the constant names.
// The five strings are the whole `stage` vocabulary of `cli_update_failed`
// (cli-update-telemetry.spec), shared with the event
// declaration in `archcore-ai/landing` and with every PostHog query built on
// it. Every other test in this file compares a Stage constant against a Stage
// constant, so a renamed value would pass the whole suite and break the
// analytics contract silently. This test is the only thing standing there.
func TestStageVocabulary_IsFrozen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		stage Stage
		want  string
	}{
		{name: "check", stage: StageCheck, want: "check"},
		{name: "download", stage: StageDownload, want: "download"},
		{name: "checksum", stage: StageChecksum, want: "checksum"},
		{name: "extract", stage: StageExtract, want: "extract"},
		{name: "replace", stage: StageReplace, want: "replace"},
	}

	seen := make(map[string]struct{}, len(tests))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.stage) != tt.want {
				t.Errorf("stage = %q, want %q", string(tt.stage), tt.want)
			}
		})
		if _, dup := seen[tt.want]; dup {
			t.Errorf("stage %q is declared twice", tt.want)
		}
		seen[tt.want] = struct{}{}
	}
	if len(seen) != 5 {
		t.Errorf("vocabulary has %d values, want 5 — a sixth changes the telemetry contract", len(seen))
	}
}

func TestStageOf(t *testing.T) {
	t.Parallel()

	base := errors.New("boom")

	tests := []struct {
		name  string
		err   error
		want  Stage
		found bool
	}{
		{"nil error", nil, "", false},
		{"plain error", base, "", false},
		{"plain wrapped error", fmt.Errorf("context: %w", base), "", false},
		{"direct stage error", &StageError{Stage: StageExtract, Err: base}, StageExtract, true},
		{
			"stage error behind fmt.Errorf",
			fmt.Errorf("context: %w", &StageError{Stage: StageReplace, Err: base}),
			StageReplace, true,
		},
		{
			"stage error behind two wraps",
			fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", &StageError{Stage: StageCheck, Err: base})),
			StageCheck, true,
		},
		{
			"stage error joined with a plain error",
			errors.Join(base, &StageError{Stage: StageChecksum, Err: base}),
			StageChecksum, true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, found := StageOf(tt.err)
			if found != tt.found {
				t.Fatalf("StageOf(%v) found = %v, want %v", tt.err, found, tt.found)
			}
			if got != tt.want {
				t.Errorf("StageOf(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}

// TestStageError_MessageIsUnchanged pins the invariant the whole design rests
// on: tagging a stage must not alter a single byte of what the user sees.
func TestStageError_MessageIsUnchanged(t *testing.T) {
	t.Parallel()

	wrapped := fmt.Errorf("downloading archive: %w", errors.New("HTTP 404"))
	tagged := &StageError{Stage: StageDownload, Err: wrapped}

	if tagged.Error() != wrapped.Error() {
		t.Errorf("StageError.Error() = %q, want %q", tagged.Error(), wrapped.Error())
	}
}

func TestStageError_Unwrap(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("sentinel")
	err := stageErr(StageDownload, fmt.Errorf("downloading archive: %w", sentinel))

	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is(%v, sentinel) = false, want true", err)
	}
}

func TestStageErr(t *testing.T) {
	t.Parallel()

	t.Run("nil stays nil", func(t *testing.T) {
		t.Parallel()
		if err := stageErr(StageCheck, nil); err != nil {
			t.Errorf("stageErr(check, nil) = %v, want nil", err)
		}
	})

	t.Run("keeps the innermost stage", func(t *testing.T) {
		t.Parallel()
		inner := stageErr(StageDownload, errors.New("boom"))
		outer := stageErr(StageReplace, fmt.Errorf("context: %w", inner))

		got, found := StageOf(outer)
		if !found {
			t.Fatal("StageOf() found no stage")
		}
		if got != StageDownload {
			t.Errorf("StageOf() = %q, want %q", got, StageDownload)
		}
	})
}

// TestCheckLatest_FailureStage pins the check stage on the resolution path.
// The message must still carry the network detail the user sees today.
func TestCheckLatest_FailureStage(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	u := NewUpdater("v1.0.0", "archcore-ai/cli", "archcore")
	u.HTTPClient = &http.Client{Transport: &rewriteTransport{target: srv.URL}}

	_, err := u.CheckLatest(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	stage, found := StageOf(err)
	if !found {
		t.Fatalf("CheckLatest() error carries no stage: %v", err)
	}
	if stage != StageCheck {
		t.Errorf("stage = %q, want %q", stage, StageCheck)
	}
	// Exact, not Contains: the stage tag must add no text. The URL in the
	// message is the real github.com one — rewriteTransport redirects the
	// connection, not the string checkLatest formatted.
	wantMsg := "github.com returned status 500 for https://github.com/archcore-ai/cli/releases/latest"
	if err.Error() != wantMsg {
		t.Errorf("error = %q, want %q", err.Error(), wantMsg)
	}
	assertTagAddsNoText(t, err)
}

// assertTagAddsNoText checks the StageError in err against the error it wraps.
// A tag that prints anything of its own would change `archcore update`'s output,
// which `@cmd/update.go` renders with err.Error().
func assertTagAddsNoText(t *testing.T, err error) {
	t.Helper()

	var se *StageError
	if !errors.As(err, &se) {
		t.Fatalf("error carries no stage: %v", err)
	}
	if se.Error() != se.Err.Error() {
		t.Errorf("StageError.Error() = %q, want the wrapped %q", se.Error(), se.Err.Error())
	}
}

// TestCheckLatest_SuccessCarriesNoStage guards the wrapper's nil path: a
// successful check must not manufacture an error.
func TestCheckLatest_SuccessCarriesNoStage(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://github.com/archcore-ai/cli/releases/tag/v1.2.3")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	u := NewUpdater("v1.0.0", "archcore-ai/cli", "archcore")
	u.HTTPClient = &http.Client{Transport: &rewriteTransport{target: srv.URL}}

	got, err := u.CheckLatest(context.Background())
	if err != nil {
		t.Fatalf("CheckLatest() error: %v", err)
	}
	if got != "v1.2.3" {
		t.Errorf("CheckLatest() = %q, want %q", got, "v1.2.3")
	}
}

// TestApply_FailureStages drives one failure per pipeline stage and pins both
// the stage and the message. The message column is the byte-identity guard:
// the stage tag adds no text to what `archcore update` prints.
func TestApply_FailureStages(t *testing.T) {
	archiveName := ArchiveName("archcore", runtime.GOOS, runtime.GOARCH)
	validArchive := createTarGz(t, map[string][]byte{"archcore": []byte("#!/bin/sh\ntrue\n")})

	// matchingChecksums is the default: a checksums.txt that agrees with
	// whatever bytes the server served, so a case reaches extract or replace.
	matchingChecksums := func(name string, data []byte) string {
		return fmt.Sprintf("%x  %s\n", sha256.Sum256(data), name)
	}

	// writableBinary is the default exec target: an existing file in a
	// writable directory, the shape Apply expects.
	writableBinary := func(t *testing.T) string {
		t.Helper()
		bin := filepath.Join(t.TempDir(), "archcore")
		if err := os.WriteFile(bin, []byte("old binary"), 0o755); err != nil {
			t.Fatalf("creating fake binary: %v", err)
		}
		return bin
	}

	tests := []struct {
		name            string
		archive         []byte                                // served archive bytes; nil means the valid one
		archiveStatus   int                                   // non-zero replaces the archive response
		checksumsStatus int                                   // non-zero replaces the checksums response
		checksums       func(name string, data []byte) string // nil means a matching line
		setupExec       func(t *testing.T) string             // nil means a writable temp binary
		wantStage       Stage
		wantMsg         string
		// wantExactMsg pins the whole rendered message for the rows whose text
		// update.go authors end to end. Contains alone would let a wrap point
		// prepend "stage: " and still pass, which is exactly the user-visible
		// change this unit must not make. Nil where the tail comes from the
		// stdlib or the OS and would turn a Go upgrade into a test failure.
		wantExactMsg func(archiveName string, archive []byte) string
	}{
		{
			name:          "archive missing from the release",
			archiveStatus: http.StatusNotFound,
			wantStage:     StageDownload,
			wantMsg:       "downloading archive",
			wantExactMsg: func(archiveName string, _ []byte) string {
				return fmt.Sprintf("downloading archive: download %s: HTTP 404", archiveName)
			},
		},
		{
			// A checksums.txt that will not transfer stopped the pipeline in a
			// transfer, so it reports download, not checksum.
			name:            "checksums file missing from the release",
			checksumsStatus: http.StatusNotFound,
			wantStage:       StageDownload,
			wantMsg:         "downloading checksums",
			wantExactMsg: func(string, []byte) string {
				return "downloading checksums: download checksums.txt: HTTP 404"
			},
		},
		{
			name: "no checksum entry for the archive",
			checksums: func(name string, data []byte) string {
				return fmt.Sprintf("%x  some_other_archive.tar.gz\n", sha256.Sum256(data))
			},
			wantStage: StageChecksum,
			wantMsg:   "checksum not found",
			wantExactMsg: func(archiveName string, _ []byte) string {
				return fmt.Sprintf("checksum not found for %s", archiveName)
			},
		},
		{
			name: "checksum mismatch",
			checksums: func(name string, data []byte) string {
				return fmt.Sprintf("%s  %s\n",
					"0000000000000000000000000000000000000000000000000000000000000000", name)
			},
			wantStage: StageChecksum,
			wantMsg:   "checksum mismatch",
			wantExactMsg: func(archiveName string, archive []byte) string {
				return fmt.Sprintf("checksum mismatch for %s: expected %s, got %x",
					archiveName,
					"0000000000000000000000000000000000000000000000000000000000000000",
					sha256.Sum256(archive))
			},
		},
		{
			name:      "corrupt archive",
			archive:   []byte("this is not an archive"),
			wantStage: StageExtract,
			wantMsg:   "extracting binary",
		},
		{
			name:      "empty archive",
			archive:   []byte{},
			wantStage: StageExtract,
			wantMsg:   "extracting binary",
		},
		{
			// Path resolution runs only to name the file the replacement
			// overwrites, so its failure belongs to replace.
			name: "exec path does not resolve",
			setupExec: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "missing-dir", "archcore")
			},
			wantStage: StageReplace,
			wantMsg:   "resolving binary path",
		},
		{
			name: "target directory is not writable",
			setupExec: func(t *testing.T) string {
				t.Helper()
				if runtime.GOOS == "windows" {
					t.Skip("directory mode does not block writes on Windows")
				}
				if os.Geteuid() == 0 {
					t.Skip("root ignores directory permissions")
				}
				dir := t.TempDir()
				bin := filepath.Join(dir, "archcore")
				if err := os.WriteFile(bin, []byte("old binary"), 0o755); err != nil {
					t.Fatalf("creating fake binary: %v", err)
				}
				if err := os.Chmod(dir, 0o500); err != nil {
					t.Fatalf("making directory read-only: %v", err)
				}
				// Restore write access before t.TempDir removes the directory.
				t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
				return bin
			},
			wantStage: StageReplace,
			wantMsg:   "replacing binary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archive := validArchive
			if tt.archive != nil {
				archive = tt.archive
			}
			checksums := matchingChecksums
			if tt.checksums != nil {
				checksums = tt.checksums
			}
			checksumsBody := checksums(archiveName, archive)

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.HasSuffix(r.URL.Path, archiveName):
					if tt.archiveStatus != 0 {
						w.WriteHeader(tt.archiveStatus)
						return
					}
					_, _ = w.Write(archive)
				case strings.HasSuffix(r.URL.Path, "checksums.txt"):
					if tt.checksumsStatus != 0 {
						w.WriteHeader(tt.checksumsStatus)
						return
					}
					_, _ = w.Write([]byte(checksumsBody))
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()

			setupExec := writableBinary
			if tt.setupExec != nil {
				setupExec = tt.setupExec
			}

			u := &Updater{
				CurrentVersion: "v1.0.0",
				GitHubRepo:     "archcore-ai/cli",
				BinaryName:     "archcore",
				HTTPClient: &http.Client{
					Transport: &rewriteTransport{target: srv.URL},
				},
				ExecPath: setupExec(t),
			}

			err := u.Apply(context.Background(), "v2.0.0")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			stage, found := StageOf(err)
			if !found {
				t.Fatalf("Apply() error carries no stage: %v", err)
			}
			if stage != tt.wantStage {
				t.Errorf("stage = %q, want %q (error: %v)", stage, tt.wantStage, err)
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("error = %v, want it to contain %q", err, tt.wantMsg)
			}
			if tt.wantExactMsg != nil {
				if want := tt.wantExactMsg(archiveName, archive); err.Error() != want {
					t.Errorf("error = %q, want %q", err.Error(), want)
				}
			}
			assertTagAddsNoText(t, err)
		})
	}
}

// TestApply_SuccessCarriesNoStage guards against a wrap point that tags a nil
// error: a completed update must return nil, not a stage-bearing error.
func TestApply_SuccessCarriesNoStage(t *testing.T) {
	archiveName := ArchiveName("archcore", runtime.GOOS, runtime.GOARCH)
	if strings.HasSuffix(archiveName, ".zip") {
		t.Skip("tar.gz fixture; the zip path is covered by TestApply_ZipArchive")
	}
	archiveData := createTarGz(t, map[string][]byte{"archcore": []byte("new binary")})
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
	defer srv.Close()

	bin := filepath.Join(t.TempDir(), "archcore")
	if err := os.WriteFile(bin, []byte("old binary"), 0o755); err != nil {
		t.Fatalf("creating fake binary: %v", err)
	}

	u := &Updater{
		CurrentVersion: "v1.0.0",
		GitHubRepo:     "archcore-ai/cli",
		BinaryName:     "archcore",
		HTTPClient:     &http.Client{Transport: &rewriteTransport{target: srv.URL}},
		ExecPath:       bin,
	}

	err := u.Apply(context.Background(), "v2.0.0")
	if err != nil {
		t.Fatalf("Apply() error: %v", err)
	}
	if _, found := StageOf(err); found {
		t.Error("StageOf(nil) reported a stage")
	}
}
