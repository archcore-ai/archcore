package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"archcore-cli/internal/telemetry"
	"archcore-cli/internal/update"

	"github.com/spf13/cobra"
)

func TestUpdateCmd_AlreadyUpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://github.com/archcore-ai/cli/releases/tag/v1.0.0")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	out, execErr := runUpdateCmd(t, "v1.0.0", srv)
	if execErr != nil {
		t.Fatalf("up-to-date check must exit zero, got: %v", execErr)
	}

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
	skipUnlessTarGz(t)

	srv := releaseArchiveServer(t, releaseFixture{tag: "v2.0.0"})
	output, execErr := runUpdateWith(t, "v1.0.0", srv, fakeBinary(t), nil)
	if execErr != nil {
		t.Fatalf("successful update must exit zero, got: %v", execErr)
	}

	if !strings.Contains(output, "Current: v1.0.0") {
		t.Errorf("expected 'Current: v1.0.0' in output, got: %s", output)
	}
	if !strings.Contains(output, "Latest:") || !strings.Contains(output, "v2.0.0") {
		t.Errorf("expected latest version v2.0.0 in output, got: %s", output)
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
		case strings.HasSuffix(r.URL.Path, "releases/latest"):
			w.Header().Set("Location", "https://github.com/archcore-ai/cli/releases/tag/v1.0.0")
			w.WriteHeader(http.StatusFound)
		default:
			// Return 404 for downloads — we just want to verify it attempts update.
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// The download 404s, so the attempted update fails — the command must
	// exit non-zero (new exit contract), which itself proves it tried.
	out, execErr := runUpdateCmd(t, "vdev", srv)
	if execErr == nil {
		t.Fatal("failed update attempt must exit non-zero")
	}

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

	out, execErr := runUpdateCmd(t, "v1.0.0", srv)
	if execErr == nil {
		t.Fatal("failed update check must exit non-zero")
	}

	if !strings.Contains(out, "Could not check for updates") {
		t.Errorf("expected error message in output, got: %s", out)
	}
}

// --- telemetry on the manual path --------------------------------------------
//
// The contract under test is cli-update-telemetry.spec: at most one event per
// invocation, `manual` on every one of them, a disclosure line for any manual
// event the endpoint accepted — a failure event as much as a success one, since
// both carry the install identifier and the machine facts off the box — and a
// payload that carries no error text and no path. A telemetry endpoint that is
// down, slow or opted out must be invisible in both the output and the exit
// code.

func TestUpdateCmd_SendsUpdatedEventOnSuccess(t *testing.T) {
	skipUnlessTarGz(t)
	isolateTelemetryEnv(t)

	rec := newTelemetryRecorder(t, http.StatusOK)
	srv := releaseArchiveServer(t, releaseFixture{tag: "v2.0.0"})

	out, execErr := runUpdateWith(t, "v1.0.0", srv, fakeBinary(t), rec.client(t, "v1.0.0"))
	if execErr != nil {
		t.Fatalf("successful update must exit zero, got: %v", execErr)
	}

	ev := rec.only(t)
	if ev.Event != "cli_updated" {
		t.Errorf("event = %q, want cli_updated", ev.Event)
	}
	wantProps := map[string]any{
		"trigger":      "manual",
		"from_version": "v1.0.0",
		"to_version":   "v2.0.0",
		"source":       "cli",
		"$lib":         "archcore-cli",
		"$lib_version": "v1.0.0",
		"os":           runtime.GOOS,
		"arch":         runtime.GOARCH,
		"ci":           false,
	}
	for key, want := range wantProps {
		if got := ev.Properties[key]; got != want {
			t.Errorf("property %s = %v, want %v", key, got, want)
		}
	}
	if ev.DistinctID == "" {
		t.Error("event carries no distinct_id")
	}
	assertDisclosed(t, out, true)
}

func TestUpdateCmd_SendsFailedEventWhenCheckFails(t *testing.T) {
	isolateTelemetryEnv(t)

	rec := newTelemetryRecorder(t, http.StatusOK)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	out, execErr := runUpdateWith(t, "v1.0.0", srv, fakeBinary(t), rec.client(t, "v1.0.0"))
	if execErr == nil {
		t.Fatal("failed update check must exit non-zero")
	}

	ev := rec.only(t)
	if ev.Event != "cli_update_failed" {
		t.Errorf("event = %q, want cli_update_failed", ev.Event)
	}
	if got := ev.Properties["stage"]; got != "check" {
		t.Errorf("stage = %v, want check", got)
	}
	if got := ev.Properties["trigger"]; got != "manual" {
		t.Errorf("trigger = %v, want manual", got)
	}
	// The tag never resolved, so a target version would be invented data.
	if _, ok := ev.Properties["to_version"]; ok {
		t.Errorf("to_version must be absent before the tag resolves, got %v", ev.Properties["to_version"])
	}
	// A delivered failure event left the machine, so the run discloses it.
	assertDisclosed(t, out, true)
}

func TestUpdateCmd_ReportsTheFailedStage(t *testing.T) {
	skipUnlessTarGz(t)

	cases := []struct {
		name      string
		fixture   releaseFixture
		brokenExe bool
		wantStage string
	}{
		{
			name:      "archive missing",
			fixture:   releaseFixture{tag: "v2.0.0", archiveMissing: true},
			wantStage: "download",
		},
		{
			name:      "checksum mismatch",
			fixture:   releaseFixture{tag: "v2.0.0", badChecksum: true},
			wantStage: "checksum",
		},
		{
			name:      "archive holds no binary",
			fixture:   releaseFixture{tag: "v2.0.0", entries: map[string][]byte{"README": []byte("no binary here")}},
			wantStage: "extract",
		},
		{
			name:      "target binary path unresolvable",
			fixture:   releaseFixture{tag: "v2.0.0"},
			brokenExe: true,
			wantStage: "replace",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			isolateTelemetryEnv(t)
			rec := newTelemetryRecorder(t, http.StatusOK)
			srv := releaseArchiveServer(t, tt.fixture)

			exe := fakeBinary(t)
			if tt.brokenExe {
				exe = filepath.Join(t.TempDir(), "absent-dir", "archcore")
			}

			out, execErr := runUpdateWith(t, "v1.0.0", srv, exe, rec.client(t, "v1.0.0"))
			if execErr == nil {
				t.Fatal("failed update must exit non-zero")
			}

			ev := rec.only(t)
			if ev.Event != "cli_update_failed" {
				t.Errorf("event = %q, want cli_update_failed", ev.Event)
			}
			if got := ev.Properties["stage"]; got != tt.wantStage {
				t.Errorf("stage = %v, want %v", got, tt.wantStage)
			}
			if got := ev.Properties["from_version"]; got != "v1.0.0" {
				t.Errorf("from_version = %v, want v1.0.0", got)
			}
			if got := ev.Properties["to_version"]; got != "v2.0.0" {
				t.Errorf("to_version = %v, want v2.0.0", got)
			}
			assertDisclosed(t, out, true)
		})
	}
}

// A failed update must report where it stopped and nothing else. The error the
// user reads names an absolute path inside their home directory; none of it may
// reach the endpoint — cli-update-telemetry.spec.
func TestUpdateCmd_EventCarriesNoErrorTextOrPath(t *testing.T) {
	skipUnlessTarGz(t)
	home := isolateTelemetryEnv(t)

	rec := newTelemetryRecorder(t, http.StatusOK)
	srv := releaseArchiveServer(t, releaseFixture{tag: "v2.0.0"})
	// A path under the user's home whose parent directory does not exist: the
	// replace stage fails with a message quoting that directory.
	exe := filepath.Join(home, "Secret Projects", "bin", "archcore")

	out, execErr := runUpdateWith(t, "v1.0.0", srv, exe, rec.client(t, "v1.0.0"))
	if execErr == nil {
		t.Fatal("failed update must exit non-zero")
	}
	if !strings.Contains(out, "Secret Projects") || !strings.Contains(out, "no such file") {
		t.Fatalf("the printed hint should quote the failing path, got: %s", out)
	}

	// The home directory is compared in both forms: on macOS the temp home is a
	// symlink and the error message quotes the resolved one.
	resolvedHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		t.Fatalf("resolving the test home: %v", err)
	}
	body := string(rec.only(t).Raw)
	for _, forbidden := range []string{home, resolvedHome, "Secret Projects", "no such file", "resolving binary path"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("payload leaks %q: %s", forbidden, body)
		}
	}
	// Key allowlist: an unexpected property is the way error text would arrive.
	got := propertyKeys(rec.only(t).Properties)
	want := []string{"$lib", "$lib_version", "arch", "ci", "from_version", "os", "source", "stage", "to_version", "trigger"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("property keys = %v, want %v", got, want)
	}
}

// The disclosure follows any delivered `manual` event, not only a delivered
// success. A `cli_update_failed` carries the install identifier and the same
// machine facts off the box as `cli_updated` does, so the run that sends one
// owes the reader the same notice — cli-update-telemetry.spec §13. What it must
// never follow is a send the endpoint refused — §14.
//
// The four cases are the whole matrix, because the narrow reading and the wide
// one agree on the two success rows: only a failure the endpoint accepted tells
// them apart.
func TestUpdateCmd_DisclosureFollowsAnyDeliveredManualEvent(t *testing.T) {
	skipUnlessTarGz(t)

	cases := []struct {
		name          string
		brokenExe     bool // the replace stage fails, so the run reports a failure
		status        int  // what the capture endpoint answers
		wantEvent     string
		wantDisclosed bool
	}{
		{
			name:          "an accepted success discloses",
			status:        http.StatusOK,
			wantEvent:     "cli_updated",
			wantDisclosed: true,
		},
		{
			name:          "an accepted failure discloses too",
			brokenExe:     true,
			status:        http.StatusOK,
			wantEvent:     "cli_update_failed",
			wantDisclosed: true,
		},
		{
			name:          "a refused success discloses nothing",
			status:        http.StatusInternalServerError,
			wantEvent:     "cli_updated",
			wantDisclosed: false,
		},
		{
			name:          "a refused failure discloses nothing",
			brokenExe:     true,
			status:        http.StatusInternalServerError,
			wantEvent:     "cli_update_failed",
			wantDisclosed: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			isolateTelemetryEnv(t)
			rec := newTelemetryRecorder(t, tt.status)
			srv := releaseArchiveServer(t, releaseFixture{tag: "v2.0.0"})

			exe := fakeBinary(t)
			if tt.brokenExe {
				exe = filepath.Join(t.TempDir(), "absent-dir", "archcore")
			}

			out, execErr := runUpdateWith(t, "v1.0.0", srv, exe, rec.client(t, "v1.0.0"))
			if (execErr != nil) != tt.brokenExe {
				t.Fatalf("exit error = %v, want one: %v", execErr, tt.brokenExe)
			}

			// only() is half the invariant: one invocation sends one event, and
			// assertDisclosed holds the other half at one notice.
			if ev := rec.only(t); ev.Event != tt.wantEvent {
				t.Errorf("event = %q, want %q", ev.Event, tt.wantEvent)
			}
			assertDisclosed(t, out, tt.wantDisclosed)
		})
	}
}

// On the failure path the notice comes after the failure lines: what went wrong
// is what the user came for, and the report of it is the footnote.
func TestUpdateCmd_DisclosureFollowsTheFailureLines(t *testing.T) {
	skipUnlessTarGz(t)
	isolateTelemetryEnv(t)

	rec := newTelemetryRecorder(t, http.StatusOK)
	srv := releaseArchiveServer(t, releaseFixture{tag: "v2.0.0"})
	exe := filepath.Join(t.TempDir(), "absent-dir", "archcore")

	out, execErr := runUpdateWith(t, "v1.0.0", srv, exe, rec.client(t, "v1.0.0"))
	if execErr == nil {
		t.Fatal("failed update must exit non-zero")
	}

	failed := strings.Index(out, "Update failed")
	notice := strings.Index(out, privacyURL)
	if failed < 0 || notice < 0 {
		t.Fatalf("expected both the failure line and the disclosure, got: %s", out)
	}
	if notice < failed {
		t.Errorf("the disclosure precedes the failure it reports, got: %s", out)
	}
}

// frozenStages is the whole vocabulary of the `stage` property. The literals
// are written out rather than read from internal/update on purpose: the values
// are shared with the PostHog queries and with the event map in
// archcore-ai/landing, so asserting them against the constants that define them
// would pin nothing — a rename would stay green here and group into nothing
// there — cli-update-telemetry.spec.
var frozenStages = []string{"check", "download", "checksum", "extract", "replace"}

// telemetryStage's fallback is unreachable from the command today — every
// CheckLatest and Apply error arrives tagged — which is exactly why it needs a
// direct test: a sixth stage value invented here would leave the suite green
// and the dashboards grouping on a category that exists nowhere else.
func TestTelemetryStage(t *testing.T) {
	stageErr := func(s update.Stage) error {
		return &update.StageError{Stage: s, Err: errors.New("boom")}
	}

	tests := []struct {
		name string
		err  error
		want string
	}{
		{"an untagged error can only have come from resolving the tag", errors.New("boom"), "check"},
		{"check", stageErr(update.StageCheck), "check"},
		{"download", stageErr(update.StageDownload), "download"},
		{"checksum", stageErr(update.StageChecksum), "checksum"},
		{"extract", stageErr(update.StageExtract), "extract"},
		{"replace", stageErr(update.StageReplace), "replace"},
		{"a wrapped stage is still found", fmt.Errorf("outer: %w", stageErr(update.StageExtract)), "extract"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := telemetryStage(tt.err)
			if got != tt.want {
				t.Errorf("telemetryStage() = %q, want %q", got, tt.want)
			}
			if !slices.Contains(frozenStages, got) {
				t.Errorf("telemetryStage() = %q, outside the contract's vocabulary %v", got, frozenStages)
			}
		})
	}
}

// Every human-readable line must reach the command's writer. A line that goes
// to the process stdout instead is invisible to any caller that redirected the
// command — the root's SetOut, a hook harness, this suite — and no assertion on
// the buffer can catch it, because the escaped line is simply absent from what
// the buffer holds rather than wrong in it.
func TestUpdateCmd_WritesNothingToTheProcessStdout(t *testing.T) {
	skipUnlessTarGz(t)

	cases := []struct {
		name        string
		args        []string
		fixture     releaseFixture
		checkFails  bool
		unwritable  bool
		wantExitErr bool
	}{
		{name: "update succeeds", fixture: releaseFixture{tag: "v2.0.0"}},
		{name: "already up to date", fixture: releaseFixture{tag: "v1.0.0"}},
		{name: "the check fails", checkFails: true, wantExitErr: true},
		{name: "apply fails", fixture: releaseFixture{tag: "v2.0.0"}, unwritable: true, wantExitErr: true},
		{name: "--check", args: []string{"--check"}, fixture: releaseFixture{tag: "v9.9.9"}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// --check writes its freshness cache under XDG_STATE_HOME.
			isolateTelemetryEnv(t)

			srv := releaseArchiveServer(t, tt.fixture)
			if tt.checkFails {
				srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "down", http.StatusInternalServerError)
				}))
				t.Cleanup(srv.Close)
			}
			exe := fakeBinary(t)
			if tt.unwritable {
				exe = filepath.Join(t.TempDir(), "absent-dir", "archcore")
			}

			rec := newTelemetryRecorder(t, http.StatusOK)
			cmd := buildUpdateCmd("v1.0.0", testUpdater("v1.0.0", srv, exe), rec.client(t, "v1.0.0"))

			var (
				out     string
				execErr error
			)
			leaked := captureStdout(t, func() { out, execErr = execUpdateCmd(t, cmd, tt.args...) })

			if (execErr != nil) != tt.wantExitErr {
				t.Fatalf("exit error = %v, want one: %v", execErr, tt.wantExitErr)
			}
			if leaked != "" {
				t.Errorf("this output bypassed the command's writer and went to the process stdout: %q", leaked)
			}
			if out == "" {
				t.Error("the command's writer received nothing at all")
			}
		})
	}
}

// newUpdateCmd is the constructor the root command uses and the only place the
// real sender is wired, so it is run here end-to-end with nothing replaced but
// the two hosts it talks to. Building the command through buildUpdateCmd — what
// every other test does — supplies a sender and therefore cannot see that line
// break, and its breakage is silent: a command wired without a sender is
// indistinguishable from an inert build.
func TestNewUpdateCmd_ReportsThroughTheWiredSender(t *testing.T) {
	skipUnlessTarGz(t)
	isolateTelemetryEnv(t)

	rec := newTelemetryRecorder(t, http.StatusOK)
	srv := releaseArchiveServer(t, releaseFixture{tag: "v2.0.0"})
	redirectUpdateDeps(t, srv, fakeBinary(t), func(tel *telemetry.Client) {
		// The release key an official build carries; everything else about the
		// sender is what newUpdateCmd built.
		tel.Key = "phc_test_key"
		tel.Endpoint = rec.url
		tel.StateDir = t.TempDir()
		tel.HTTPClient = &http.Client{Timeout: telemetryTestTimeout}
	})

	out, execErr := execUpdateCmd(t, newUpdateCmd("v1.0.0"))
	if execErr != nil {
		t.Fatalf("successful update must exit zero, got: %v", execErr)
	}

	ev := rec.only(t)
	if ev.Event != "cli_updated" {
		t.Errorf("event = %q, want cli_updated", ev.Event)
	}
	if got := ev.Properties["trigger"]; got != "manual" {
		t.Errorf("trigger = %v, want manual", got)
	}
	// The sender must carry the version the command was built for; a mismatch
	// splits one release across two $lib_version values.
	if got := ev.Properties["$lib_version"]; got != "v1.0.0" {
		t.Errorf("$lib_version = %v, want v1.0.0", got)
	}
	assertDisclosed(t, out, true)
}

// The spec's conformance case: a `go build` binary carries no release key, so a
// successful update on it makes no request, creates no identifier file and
// prints no disclosure — cli-update-telemetry.spec. Only the destinations are
// redirected here; the key is left as the build left it.
func TestNewUpdateCmd_InertWithoutAReleaseKey(t *testing.T) {
	skipUnlessTarGz(t)
	isolateTelemetryEnv(t)

	rec := newTelemetryRecorder(t, http.StatusOK)
	srv := releaseArchiveServer(t, releaseFixture{tag: "v2.0.0"})
	stateDir := t.TempDir()
	redirectUpdateDeps(t, srv, fakeBinary(t), func(tel *telemetry.Client) {
		tel.Endpoint = rec.url
		tel.StateDir = stateDir
		tel.HTTPClient = &http.Client{Timeout: telemetryTestTimeout}
	})

	out, execErr := execUpdateCmd(t, newUpdateCmd("v1.0.0"))
	if execErr != nil {
		t.Fatalf("successful update must exit zero, got: %v", execErr)
	}
	if !strings.Contains(out, "Updated to v2.0.0") {
		t.Fatalf("expected the success path, got: %s", out)
	}
	if n := rec.count(); n != 0 {
		t.Errorf("a build with no release key must send nothing, saw %d event(s)", n)
	}
	if entries, err := os.ReadDir(stateDir); err != nil {
		t.Fatal(err)
	} else if len(entries) != 0 {
		t.Errorf("a build with no release key must write no identifier, found %d entr(ies)", len(entries))
	}
	assertDisclosed(t, out, false)
}

func TestUpdateCmd_NoEventWhenAlreadyCurrent(t *testing.T) {
	isolateTelemetryEnv(t)

	rec := newTelemetryRecorder(t, http.StatusOK)
	srv := releaseServer(t, "v1.0.0")

	out, execErr := runUpdateWith(t, "v1.0.0", srv, fakeBinary(t), rec.client(t, "v1.0.0"))
	if execErr != nil {
		t.Fatalf("up-to-date check must exit zero, got: %v", execErr)
	}
	if !strings.Contains(out, "Already up to date") {
		t.Fatalf("expected the up-to-date path, got: %s", out)
	}
	if n := rec.count(); n != 0 {
		t.Errorf("an up-to-date run must send nothing, saw %d event(s)", n)
	}
	assertDisclosed(t, out, false)
}

func TestUpdateCmd_NoEventOnCheckFlag(t *testing.T) {
	isolateTelemetryEnv(t)

	rec := newTelemetryRecorder(t, http.StatusOK)
	srv := releaseServer(t, "v9.9.9")

	cmd := buildUpdateCmd("v1.0.0", testUpdater("v1.0.0", srv, ""), rec.client(t, "v1.0.0"))
	out, execErr := execUpdateCmd(t, cmd, "--check")
	if execErr != nil {
		t.Fatalf("update --check must always exit 0, got: %v", execErr)
	}
	if !strings.Contains(out, "update available: v9.9.9") {
		t.Fatalf("expected the advisory line, got: %q", out)
	}
	if n := rec.count(); n != 0 {
		t.Errorf("--check must send nothing, saw %d event(s)", n)
	}
	assertDisclosed(t, out, false)
}

// A telemetry endpoint the user never asked about must not be able to change
// what `archcore update` prints or what it exits with — cli-update-telemetry.spec.
func TestUpdateCmd_TelemetryFailureLeavesOutputAndExitUnchanged(t *testing.T) {
	skipUnlessTarGz(t)

	runOnce := func(t *testing.T, tel *telemetry.Client) (string, error) {
		t.Helper()
		isolateTelemetryEnv(t)
		srv := releaseArchiveServer(t, releaseFixture{tag: "v2.0.0"})
		return runUpdateWith(t, "v1.0.0", srv, fakeBinary(t), tel)
	}

	// The baseline is taken in the parent, not in a subtest: a subtest can be
	// skipped or filtered out by -run, and an empty baseline would let both
	// comparisons below pass for a reason that has nothing to do with telemetry.
	baseline, err := runOnce(t, nil)
	if err != nil {
		t.Fatalf("update must exit zero, got: %v", err)
	}
	if baseline == "" {
		t.Fatal("the telemetry-off run printed nothing; there is no baseline to compare against")
	}
	assertDisclosed(t, baseline, false)

	t.Run("endpoint returns 500", func(t *testing.T) {
		rec := newTelemetryRecorder(t, http.StatusInternalServerError)
		out, err := runOnce(t, rec.client(t, "v1.0.0"))
		if err != nil {
			t.Fatalf("update must exit zero, got: %v", err)
		}
		if rec.count() != 1 {
			t.Errorf("the send is still attempted once, saw %d", rec.count())
		}
		if out != baseline {
			t.Errorf("output differs from the telemetry-off run:\n got: %q\nwant: %q", out, baseline)
		}
	})

	t.Run("endpoint hangs", func(t *testing.T) {
		// The handler holds the request until the test is over. It drains the
		// body first: net/http only starts watching for a client disconnect
		// once the request body has been read, so an undrained handler would
		// never see the send time out.
		release := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			select {
			case <-release:
			case <-r.Context().Done():
			}
		}))
		// LIFO: release first, then Close, which waits for the handler.
		t.Cleanup(srv.Close)
		t.Cleanup(func() { close(release) })

		out, err := runOnce(t, telemetryClient("v1.0.0", srv.URL, t.TempDir()))
		if err != nil {
			t.Fatalf("update must exit zero, got: %v", err)
		}
		if out != baseline {
			t.Errorf("output differs from the telemetry-off run:\n got: %q\nwant: %q", out, baseline)
		}
	})
}

// --- helpers -----------------------------------------------------------------

// telemetryTestTimeout keeps a stalled endpoint from stalling the suite. The
// production bound lives in internal/telemetry; this one only has to be shorter
// than the test timeout.
const telemetryTestTimeout = 300 * time.Millisecond

// capturedEvent is one PostHog capture payload as the endpoint received it.
type capturedEvent struct {
	Event      string         `json:"event"`
	DistinctID string         `json:"distinct_id"`
	APIKey     string         `json:"api_key"`
	Properties map[string]any `json:"properties"`

	// Raw is the undecoded body, so a leak assertion can search the bytes that
	// actually left the process rather than a re-encoding of them.
	Raw []byte `json:"-"`
}

// telemetryRecorder stands in for the capture endpoint. It is a separate server
// from the release server on purpose: the events must be observable without
// going through testRewriteTransport, which would hide a stray request.
type telemetryRecorder struct {
	url    string
	status int

	mu     sync.Mutex
	events []capturedEvent
}

func newTelemetryRecorder(t *testing.T, status int) *telemetryRecorder {
	t.Helper()
	rec := &telemetryRecorder{status: status}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := &bytes.Buffer{}
		_, _ = body.ReadFrom(r.Body)

		ev := capturedEvent{Raw: body.Bytes()}
		if err := json.Unmarshal(ev.Raw, &ev); err != nil {
			t.Errorf("telemetry payload is not JSON: %v", err)
		}

		rec.mu.Lock()
		rec.events = append(rec.events, ev)
		rec.mu.Unlock()

		w.WriteHeader(rec.status)
	}))
	t.Cleanup(srv.Close)
	rec.url = srv.URL
	return rec
}

func (r *telemetryRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events)
}

// only returns the single event of the invocation, failing when the
// one-event-per-invocation invariant does not hold.
func (r *telemetryRecorder) only(t *testing.T) capturedEvent {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.events) != 1 {
		t.Fatalf("want exactly one event, got %d", len(r.events))
	}
	return r.events[0]
}

func (r *telemetryRecorder) client(t *testing.T, version string) *telemetry.Client {
	t.Helper()
	return telemetryClient(version, r.url, t.TempDir())
}

// telemetryClient builds a sender that passes the release-key guard and keeps
// its install-id inside stateDir.
func telemetryClient(version, endpoint, stateDir string) *telemetry.Client {
	return &telemetry.Client{
		Version:    version,
		Key:        "phc_test_key",
		Endpoint:   endpoint,
		StateDir:   stateDir,
		HTTPClient: &http.Client{Timeout: telemetryTestTimeout},
	}
}

// redirectUpdateDeps points the production wiring at the test's own hosts.
//
// Only the destinations change: newUpdateCmd still builds both dependencies
// itself, which is the wiring under test. tune receives the sender that wiring
// produced, so a case decides whether the build carries a release key.
//
// The test must not call t.Parallel: this replaces a package-level variable
// that every NewRootCmd call reads.
func redirectUpdateDeps(t *testing.T, srv *httptest.Server, execPath string, tune func(*telemetry.Client)) {
	t.Helper()

	production := updateDeps
	t.Cleanup(func() { updateDeps = production })

	updateDeps = func(version string) (*update.Updater, *telemetry.Client) {
		u, tel := production(version)
		if u == nil || tel == nil {
			// A nil sender stops every manual event and fails nothing, so name
			// it here rather than let the run look like an inert build.
			t.Errorf("the production wiring returned updater=%v sender=%v, want both", u, tel)
			return u, tel
		}
		u.HTTPClient = &http.Client{Transport: &testRewriteTransport{target: srv.URL}}
		u.ExecPath = execPath
		tune(tel)
		return u, tel
	}
}

// isolatePluginStep detaches the plugin step from the machine running the
// suite. `archcore update` runs the step after the binary phase, so a test that
// executes the command reaches it, and the two funnels above call this before
// every run.
//
// The hazard is not noise, it is mutation: the step queries whatever host CLIs
// the developer has on PATH, and on a machine that carries the Archcore plugin
// the update tier is the one that runs `claude plugin update` — during
// `go test`. PATH and home are the whole observation surface (the collector
// resolves each host CLI on PATH and falls back to the on-disk registry under
// the home directory), so emptying both leaves every host without evidence,
// which is the silent tier.
// It delegates rather than repeating the three t.Setenv calls: two spellings of
// one isolation in a single package drift, and the one that drifts is the one a
// test stopped exercising. The returned bin directory is unused here — a caller
// that stages its own host fixture uses isolatePluginRun directly.
func isolatePluginStep(t *testing.T) {
	t.Helper()
	isolatePluginRun(t)
}

// isolateTelemetryEnv detaches the test from the machine running it: a private
// home and state directory, no opt-out variable inherited from the developer's
// shell, and no CI variable that would flip the `ci` property. It returns the
// home directory.
func isolateTelemetryEnv(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir reads this one on Windows
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("DO_NOT_TRACK", "")
	t.Setenv("ARCHCORE_TELEMETRY_OPTOUT", "")
	// The set comes from the one declaration, not from a fourth copy of the
	// list; a CI provider added there must reach this isolation too.
	for _, name := range telemetry.CIVars {
		t.Setenv(name, "")
	}
	return home
}

// privacyURL is the half of the disclosure the spec fixes verbatim, so it is
// what "the disclosure is present" is probed on. Probing on DO_NOT_TRACK would
// make every negative assertion here vacuous the day the line names
// ARCHCORE_TELEMETRY_OPTOUT instead — the other variable §13 allows — because
// the probe would then match nothing whatever the command printed.
const privacyURL = "https://archcore.ai/privacy"

// assertDisclosed checks the post-delivery notice: exactly one line, naming an
// opt-out variable and the privacy page — cli-update-telemetry.spec §13.
func assertDisclosed(t *testing.T, out string, want bool) {
	t.Helper()

	var lines []string
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, privacyURL) {
			lines = append(lines, line)
		}
	}

	if got := len(lines) > 0; got != want {
		t.Errorf("disclosure present = %v, want %v; output: %q", got, want, out)
	}
	if !want {
		return
	}
	if len(lines) != 1 {
		t.Errorf("disclosure printed on %d lines, want one", len(lines))
		return
	}
	if !strings.Contains(lines[0], "DO_NOT_TRACK") && !strings.Contains(lines[0], "ARCHCORE_TELEMETRY_OPTOUT") {
		t.Errorf("disclosure names no opt-out variable, got: %q", lines[0])
	}
}

func propertyKeys(props map[string]any) []string {
	keys := make([]string, 0, len(props))
	for key := range props {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// releaseFixture describes what the release server serves. The zero value plus
// a tag is a healthy release; each other field breaks exactly one step of Apply,
// which is how a test reaches one stage and no other.
type releaseFixture struct {
	tag            string
	entries        map[string][]byte // files inside the archive; nil → a plausible binary
	archiveMissing bool              // the archive request 404s
	badChecksum    bool              // checksums.txt names a checksum that does not match
}

// releaseArchiveServer answers the three requests an update makes: the
// latest-release redirect, the archive, and checksums.txt.
func releaseArchiveServer(t *testing.T, f releaseFixture) *httptest.Server {
	t.Helper()

	entries := f.entries
	if entries == nil {
		entries = map[string][]byte{"archcore": []byte("#!/bin/sh\necho archcore " + f.tag)}
	}
	archive := buildTestArchive(t, entries)
	name := update.ArchiveName("archcore", runtime.GOOS, runtime.GOARCH)

	checksums := fmt.Sprintf("%x  %s\n", sha256.Sum256(archive), name)
	if f.badChecksum {
		checksums = fmt.Sprintf("%s  %s\n", strings.Repeat("0", 64), name)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "releases/latest"):
			w.Header().Set("Location", "https://github.com/archcore-ai/cli/releases/tag/"+f.tag)
			w.WriteHeader(http.StatusFound)
		case strings.HasSuffix(r.URL.Path, name):
			if f.archiveMissing {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(archive)
		case strings.HasSuffix(r.URL.Path, "checksums.txt"):
			_, _ = w.Write([]byte(checksums))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// fakeBinary returns a throwaway file for the updater to replace.
func fakeBinary(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "archcore")
	if err := os.WriteFile(path, []byte("old binary"), 0o755); err != nil {
		t.Fatalf("creating fake binary: %v", err)
	}
	return path
}

func skipUnlessTarGz(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("tar.gz path; see TestApply_ZipArchive for the windows regression test")
	}
}

func testUpdater(version string, srv *httptest.Server, execPath string) *update.Updater {
	return &update.Updater{
		CurrentVersion: version,
		GitHubRepo:     "archcore-ai/cli",
		BinaryName:     "archcore",
		HTTPClient:     &http.Client{Transport: &testRewriteTransport{target: srv.URL}},
		ExecPath:       execPath,
	}
}

// runUpdateWith executes `archcore update` against a release server, replacing
// the binary at execPath and reporting to tel. A nil tel is the telemetry-off
// case. The output is read from the command's own writer: the command writes
// every line there, so nothing has to be captured from the process stdout.
func runUpdateWith(t *testing.T, version string, srv *httptest.Server, execPath string, tel *telemetry.Client) (string, error) {
	t.Helper()
	return execUpdateCmd(t, buildUpdateCmd(version, testUpdater(version, srv, execPath), tel))
}

// execUpdateCmd runs cmd with args and returns what its writer received.
//
// args is normalized to a non-nil slice on purpose: cobra falls back to
// os.Args[1:] when a command's args were never set, which under `go test` feeds
// it the -test.* flags and fails the command for a reason no assertion names.
func execUpdateCmd(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	isolatePluginStep(t)

	if args == nil {
		args = []string{}
	}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	cmd.SetContext(context.Background())
	// The root command silences both, so the buffer holds what a user sees and
	// nothing cobra adds on top of it.
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	execErr := cmd.Execute()
	return buf.String(), execErr
}

// runUpdateCmd executes the update command through the root command with a test
// release server, returning the output and the execution error so callers can
// assert the exit contract alongside the output.
func runUpdateCmd(t *testing.T, version string, srv *httptest.Server) (string, error) {
	t.Helper()
	isolatePluginStep(t)

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
	root.SetContext(context.Background())

	execErr := root.Execute()
	return buf.String(), execErr
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
