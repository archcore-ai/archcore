// Package update replaces the running binary with a newer release. The download
// is bounded, the archive is validated before anything is written, and the
// replacement is atomic with a rollback — a failed update must leave a working
// binary, not a truncated one. An optional pre-commit probe runs the staged
// binary before the rename, so an unattended caller never commits one that
// cannot start.
package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	maxArchiveSize   = 50 << 20 // 50 MB
	maxChecksumsSize = 1 << 20  // 1 MB
)

// Updater checks for and applies updates from GitHub Releases.
type Updater struct {
	CurrentVersion string
	GitHubRepo     string // e.g. "archcore-ai/cli"
	BinaryName     string // e.g. "archcore"
	HTTPClient     *http.Client
	ExecPath       string // Override for os.Executable(); used in tests.

	// PreCommitProbe, when set, runs against the staged temporary binary after
	// the synced write and before the rename. An error abandons the
	// replacement, and the staged file is removed. The unattended policy
	// installs healthProbe here so a download that cannot report its own
	// version never becomes the installed binary; the manual path leaves it
	// nil, so typed `archcore update` is unchanged —
	// unattended-update.spec §15.
	PreCommitProbe func(ctx context.Context, stagedPath string) error
}

// NewUpdater creates an Updater with sensible defaults.
func NewUpdater(currentVersion, repo, binaryName string) *Updater {
	return &Updater{
		CurrentVersion: currentVersion,
		GitHubRepo:     repo,
		BinaryName:     binaryName,
		HTTPClient:     defaultHTTPClient(),
	}
}

// httpTimeout bounds one release request end to end.
//
// The budget it protects is the whole unattended attempt: unattendedCeiling
// (120s) has to cover the tag lookup plus two sequential downloads (the archive
// and the checksums) at this timeout, so raising it past half the ceiling makes
// the ceiling, not the timeout, the thing that cancels a slow download.
const httpTimeout = 60 * time.Second

// defaultHTTPClient is the client every Updater uses unless a caller supplies
// its own. It is built once: a zero-value Updater must never reach
// http.DefaultClient, which carries no timeout at all.
var defaultHTTPClient = sync.OnceValue(func() *http.Client {
	return &http.Client{Timeout: httpTimeout}
})

// tagPathMarker separates the repo path from the tag in a /releases/latest
// redirect: https://github.com/OWNER/REPO/releases/tag/vX.Y.Z
const tagPathMarker = "/releases/tag/"

// maxRedirectBodyDrain caps the bytes read from a redirect body before close.
// A 3xx body is empty, so this still buys connection reuse; an unexpected HTML
// error page can no longer stall the caller (`update --check` runs inside
// editor hooks on a 2s budget).
const maxRedirectBodyDrain = 4 << 10

// client returns the HTTP client to use. A zero-value Updater — the shape
// cmd/update.go builds when it supplies no client of its own — falls back to
// the shared timeout-carrying client, never to http.DefaultClient.
func (u *Updater) client() *http.Client {
	if u.HTTPClient != nil {
		return u.HTTPClient
	}
	return defaultHTTPClient()
}

// CheckLatest resolves the tag of the newest published release.
//
// It deliberately reads the github.com web redirect rather than calling
// api.github.com/repos/.../releases/latest. The REST API allows only 60
// unauthenticated requests per hour *per IP*, and this check runs from editor
// hooks on every installed machine — so a team behind one egress address
// (corporate NAT, CGNAT, CI runners) burns that budget and silently stops
// seeing updates. GET /releases/latest answers with a redirect whose Location
// already carries the tag, costs no rate-limit budget, and needs no token.
//
// Every error carries StageCheck; read it with StageOf. The message is the one
// checkLatest produced, unchanged.
func (u *Updater) CheckLatest(ctx context.Context) (string, error) {
	tag, err := u.checkLatest(ctx)
	return tag, stageErr(StageCheck, err)
}

// checkLatest holds the resolution itself. CheckLatest wraps it so the network
// detail below stays intact while the caller still learns the stage.
func (u *Updater) checkLatest(ctx context.Context) (string, error) {
	latestURL := fmt.Sprintf("https://github.com/%s/releases/latest", u.GitHubRepo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	// Copy the client so halting redirects stays local to this call and does
	// not leak into the caller's client (which Apply reuses to follow the
	// release-asset redirect chain).
	halted := *u.client()
	halted.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := halted.Do(req)
	if err != nil {
		return "", fmt.Errorf("checking latest release: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxRedirectBodyDrain))
		_ = resp.Body.Close()
	}()

	// Any 3xx is accepted: GitHub answers 302 today but 301/303/307/308 are
	// all legitimate ways to say "the tag lives over there". The Location
	// check below is the real gate — a redirect that does not land on a tag
	// page fails regardless of which 3xx carried it.
	if resp.StatusCode/100 != 3 {
		return "", fmt.Errorf("github.com returned status %d for %s", resp.StatusCode, latestURL)
	}

	// resp.Location resolves the header against the request URL and normalizes
	// dot-segments, and reading .Path drops query and fragment — so none of
	// those can smuggle a tag past the marker search below.
	loc, err := resp.Location()
	if errors.Is(err, http.ErrNoLocation) {
		return "", fmt.Errorf("no Location header in %d response from %s", resp.StatusCode, latestURL)
	}
	if err != nil {
		return "", fmt.Errorf("parsing redirect location from %s: %w", latestURL, err)
	}

	idx := strings.LastIndex(loc.Path, tagPathMarker)
	if idx < 0 {
		return "", fmt.Errorf("unexpected redirect resolving latest release: %q", loc.String())
	}

	tag := strings.Trim(loc.Path[idx+len(tagPathMarker):], "/")
	if tag == "" {
		return "", fmt.Errorf("empty tag in redirect %q", loc.String())
	}

	return tag, nil
}

// NeedsUpdate reports whether the CLI should replace itself: true when latest
// is strictly newer than current, or when current is "dev".
//
// A version that does not parse as semver on either side means "no update".
// The comparison once fell back to string inequality there, which in an
// unattended path turns an odd tag into a replacement or a downgrade —
// unattended-update.spec §12. The "dev" case survives that tightening because
// a development build has no parseable version at all and is still behind any
// release; the unattended policy refuses "dev" builds before it ever compares.
func NeedsUpdate(current, latest string) bool {
	if stripV(current) == "dev" {
		return true
	}
	newer, _ := NewerSemver(current, latest)
	return newer
}

// NewerSemver reports whether latest is strictly newer than current, and
// whether both sides parsed. Callers that must not act on an unparseable
// version check ok; NeedsUpdate keeps its own `dev` special case.
func NewerSemver(current, latest string) (newer, ok bool) {
	current = stripV(current)
	latest = stripV(latest)

	curParts, curPre := parseSemver(current)
	latParts, latPre := parseSemver(latest)

	if curParts == nil || latParts == nil {
		return false, false
	}

	for i := range 3 {
		if latParts[i] > curParts[i] {
			return true, true
		}
		if latParts[i] < curParts[i] {
			return false, true
		}
	}

	// Major.minor.patch are equal — compare pre-release.
	// Per SemVer: release (no pre-release) > any pre-release.
	if curPre != "" && latPre == "" {
		return true, true // current is pre-release, latest is release
	}
	if curPre == "" && latPre != "" {
		return false, true // current is release, latest is pre-release
	}
	if curPre == "" && latPre == "" {
		return false, true // both are releases, identical
	}

	return comparePreRelease(curPre, latPre) < 0, true
}

// stripV removes the leading "v" prefix from a version string.
func stripV(v string) string {
	return strings.TrimPrefix(v, "v")
}

// parseSemver splits a version string like "1.2.3" or "1.2.3-beta.1" into
// [major, minor, patch] and an optional pre-release string.
// Returns nil if parsing fails.
func parseSemver(v string) ([]int, string) {
	var preRelease string
	if idx := strings.Index(v, "-"); idx != -1 {
		preRelease = v[idx+1:]
		v = v[:idx]
	}

	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return nil, ""
	}

	result := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, ""
		}
		result[i] = n
	}
	return result, preRelease
}

// comparePreRelease compares two pre-release strings per SemVer 2.0.0 §11.
// Returns -1 if a < b, 0 if a == b, +1 if a > b.
func comparePreRelease(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	n := min(len(aParts), len(bParts))

	for i := range n {
		aNum, aErr := strconv.Atoi(aParts[i])
		bNum, bErr := strconv.Atoi(bParts[i])

		switch {
		case aErr == nil && bErr == nil:
			// Both numeric: compare as integers.
			if aNum < bNum {
				return -1
			}
			if aNum > bNum {
				return 1
			}
		case aErr == nil && bErr != nil:
			// Numeric < alphanumeric per SemVer.
			return -1
		case aErr != nil && bErr == nil:
			return 1
		default:
			// Both alphanumeric: compare lexically.
			if aParts[i] < bParts[i] {
				return -1
			}
			if aParts[i] > bParts[i] {
				return 1
			}
		}
	}

	// All compared identifiers are equal — shorter set has lower precedence.
	if len(aParts) < len(bParts) {
		return -1
	}
	if len(aParts) > len(bParts) {
		return 1
	}
	return 0
}

// ArchiveName returns the expected archive filename for the current platform.
// Windows releases use .zip; all others use .tar.gz (matches .goreleaser.yaml).
func ArchiveName(binaryName, goos, goarch string) string {
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("%s_%s_%s.%s", binaryName, goos, goarch, ext)
}

// Apply downloads and installs the specified version.
//
// Every error carries the pipeline stage that failed; read it with StageOf.
// The tag adds no text — the messages below are the ones the caller prints.
func (u *Updater) Apply(ctx context.Context, version string) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	archive := ArchiveName(u.BinaryName, goos, goarch)

	// 1. Download the archive.
	archiveData, err := u.download(ctx, version, archive)
	if err != nil {
		return stageErr(StageDownload, fmt.Errorf("downloading archive: %w", err))
	}

	// 2. Download and verify checksum. A failed fetch of checksums.txt counts
	// as download, not checksum: the stage names where the pipeline stopped,
	// and this stopped in a transfer. Only the lookup and the comparison —
	// findChecksum and VerifyChecksum — report checksum.
	checksums, err := u.download(ctx, version, "checksums.txt")
	if err != nil {
		return stageErr(StageDownload, fmt.Errorf("downloading checksums: %w", err))
	}

	if err := VerifyChecksum(archiveData, checksums, archive); err != nil {
		return stageErr(StageChecksum, err)
	}

	// 3. Extract binary from the archive.
	// GoReleaser may name the binary either as the configured binary name
	// or as the repo basename; on Windows both carry an .exe suffix.
	binaryData, err := ExtractBinary(archiveData, binaryCandidates(u.BinaryName, u.GitHubRepo, goos)...)
	if err != nil {
		return stageErr(StageExtract, fmt.Errorf("extracting binary: %w", err))
	}

	// 4. Find current binary path (resolve symlinks). Path resolution belongs
	// to replace: it exists only to name the file step 5 overwrites.
	execPath := u.ExecPath
	if execPath == "" {
		execPath, err = os.Executable()
		if err != nil {
			return stageErr(StageReplace, fmt.Errorf("locating current binary: %w", err))
		}
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return stageErr(StageReplace, fmt.Errorf("resolving binary path: %w", err))
	}

	// 5. Atomic replace, with the pre-commit probe when one is installed. A
	// probe failure is a replace failure: the pipeline stopped with a staged
	// file it refused to commit.
	if err := atomicReplace(ctx, execPath, binaryData, u.PreCommitProbe); err != nil {
		return stageErr(StageReplace, fmt.Errorf("replacing binary: %w", err))
	}

	return nil
}

// download fetches a file from a GitHub release.
func (u *Updater) download(ctx context.Context, version, filename string) ([]byte, error) {
	downloadURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s",
		u.GitHubRepo, version, filename)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", filename, err)
	}

	resp, err := u.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", filename, err)
	}
	defer func() {
		// Bounded like the redirect drain: this runs on the non-200 branch with
		// nothing consumed yet, and on the oversize branch with everything past
		// the limit still queued, so an unbounded copy would read a body this
		// process has already refused.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxRedirectBodyDrain))
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: HTTP %d", filename, resp.StatusCode)
	}

	limit := int64(maxArchiveSize)
	if strings.HasSuffix(filename, ".txt") {
		limit = maxChecksumsSize
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filename, err)
	}

	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s exceeds size limit (%d bytes)", filename, limit)
	}

	return data, nil
}

// VerifyChecksum validates the SHA-256 checksum of data against the
// checksums file content. The checksums file is expected to have lines
// in the format: "<hash>  <filename>".
func VerifyChecksum(data, checksums []byte, filename string) error {
	expected, err := findChecksum(checksums, filename)
	if err != nil {
		return err
	}

	actual := sha256sum(data)
	if actual != expected {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", filename, expected, actual)
	}

	return nil
}

// findChecksum looks up a filename in checksums.txt content and returns
// the corresponding SHA-256 hash.
func findChecksum(checksums []byte, filename string) (string, error) {
	for line := range strings.SplitSeq(string(checksums), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "<hash>  <filename>" (two spaces between hash and name).
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		if parts[1] == filename {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("checksum not found for %s", filename)
}

// sha256sum computes the hex-encoded SHA-256 digest of data.
func sha256sum(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// binaryCandidates returns the binary filenames to look for inside the archive.
// GoReleaser may use either the configured binary name or the repo basename;
// Windows builds carry an .exe suffix.
func binaryCandidates(binaryName, repo, goos string) []string {
	names := []string{binaryName, filepath.Base(repo)}
	if goos == "windows" {
		for i := range names {
			names[i] += ".exe"
		}
	}
	return names
}

// ExtractBinary extracts a binary from a release archive. The archive format
// is auto-detected from the magic bytes: tar.gz on Unix, zip on Windows.
// Returns the first candidate name that matches.
func ExtractBinary(archiveData []byte, candidates ...string) ([]byte, error) {
	if isZipArchive(archiveData) {
		return extractFromZip(archiveData, candidates...)
	}
	return extractFromTarGz(archiveData, candidates...)
}

// isZipArchive checks for the PK\x03\x04 local-file-header signature.
func isZipArchive(data []byte) bool {
	return len(data) >= 4 && data[0] == 'P' && data[1] == 'K' && data[2] == 0x03 && data[3] == 0x04
}

// makeCandidateSet returns a presence-only set of the candidate names.
func makeCandidateSet(candidates []string) map[string]struct{} {
	set := make(map[string]struct{}, len(candidates))
	for _, c := range candidates {
		set[c] = struct{}{}
	}
	return set
}

// hasUnsafePath rejects archive entries with directory-traversal segments or
// absolute paths. Defense in depth — checksum verification runs before
// extraction, but a tampered archive should still fail closed.
func hasUnsafePath(name string) bool {
	return strings.Contains(name, "..") || path.IsAbs(name) || filepath.IsAbs(name)
}

func errBinaryNotFound(candidates []string) error {
	return fmt.Errorf("binary not found in archive (tried: %s)", strings.Join(candidates, ", "))
}

func extractFromTarGz(archiveData []byte, candidates ...string) ([]byte, error) {
	candidateSet := makeCandidateSet(candidates)

	gr, err := gzip.NewReader(bytes.NewReader(archiveData))
	if err != nil {
		return nil, fmt.Errorf("opening gzip: %w", err)
	}
	defer func() { _ = gr.Close() }() // read-only handle

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading tar: %w", err)
		}

		if hasUnsafePath(hdr.Name) {
			continue
		}

		// The binary may be at the root or inside a directory.
		name := path.Base(filepath.ToSlash(hdr.Name))
		if _, ok := candidateSet[name]; ok && hdr.Typeflag == tar.TypeReg {
			// Read limit+1 so exceeding the cap is an error, not a silent
			// truncation that would install a broken binary (the checksum is
			// verified on the compressed archive, not the extracted bytes).
			data, err := io.ReadAll(io.LimitReader(tr, maxArchiveSize+1))
			if err != nil {
				return nil, fmt.Errorf("reading %s from archive: %w", name, err)
			}
			if int64(len(data)) > maxArchiveSize {
				return nil, fmt.Errorf("%s exceeds size limit (%d bytes)", name, int64(maxArchiveSize))
			}
			return data, nil
		}
	}

	return nil, errBinaryNotFound(candidates)
}

func extractFromZip(archiveData []byte, candidates ...string) ([]byte, error) {
	candidateSet := makeCandidateSet(candidates)

	zr, err := zip.NewReader(bytes.NewReader(archiveData), int64(len(archiveData)))
	if err != nil {
		return nil, fmt.Errorf("opening zip: %w", err)
	}

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if hasUnsafePath(f.Name) {
			continue
		}
		// zip paths always use forward slashes per spec.
		name := path.Base(f.Name)
		if _, ok := candidateSet[name]; !ok {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("opening %s in zip: %w", name, err)
		}
		data, err := io.ReadAll(io.LimitReader(rc, maxArchiveSize+1))
		_ = rc.Close()
		if err != nil {
			return nil, fmt.Errorf("reading %s from zip: %w", name, err)
		}
		if int64(len(data)) > maxArchiveSize {
			return nil, fmt.Errorf("%s exceeds size limit (%d bytes)", name, int64(maxArchiveSize))
		}
		return data, nil
	}

	return nil, errBinaryNotFound(candidates)
}

// probeTimeout bounds the staged binary's `--version` run. It protects the
// unattended policy's 120 s ceiling, which the probe would otherwise be able to
// consume on its own: a downloaded binary that hangs on startup would hold the
// caller open until the ceiling elapsed, with a staged file already written.
// Three seconds is well beyond the cost of printing a version string —
// unattended-update.spec §15.
const probeTimeout = 3 * time.Second

// healthProbe runs the staged binary with `--version`. It returns an error when
// the binary fails to start, exits nonzero, or does not finish within
// probeTimeout. The unattended policy installs it as Updater.PreCommitProbe, so
// a download that cannot report its own version never replaces a working
// binary — unattended-update.spec §15.
//
// The bound comes from context.WithoutCancel(ctx) on purpose. The policy runs
// under a 120 s ceiling that must not interrupt the synced write or the rename,
// and the probe sits between them; inheriting a deadline that may already have
// expired would abandon a replacement that was one rename from done, for a
// reason that has nothing to do with the binary's health.
//
// The probe reads no output. It needs the exit status only, buffering the
// output would be an unbounded read from a binary this process has not yet
// trusted, and a caller such as `archcore mcp` owns stdout as a protocol
// stream. A nil Stdout and Stderr connect the child to the null device rather
// than to ours.
func healthProbe(ctx context.Context, stagedPath string) error {
	probeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), probeTimeout)
	defer cancel()

	// The staged name, not the staged path: a probe failure reaches the user
	// through `cli_update_failed` and through `archcore update` output, and
	// neither carries an absolute filesystem path.
	name := filepath.Base(stagedPath)

	cmd := exec.CommandContext(probeCtx, stagedPath, "--version")
	if err := cmd.Run(); err != nil {
		if errors.Is(probeCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("%s --version did not finish within %s", name, probeTimeout)
		}
		return fmt.Errorf("running %s --version: %w", name, err)
	}
	return nil
}

// atomicReplace installs data over target: it stages a synced temporary file
// next to it, runs probe against that staged file when one is set, and commits
// the staged file over target.
//
// This is the one binary replacement in the repository that stays a bespoke
// temp-file-plus-rename instead of calling a shared helper. It fsyncs before
// close and carries a Windows rename-aside with rollback, which no other write
// target needs — choosing-an-atomic-write.rule records it as the documented
// exception rather than a precedent.
func atomicReplace(ctx context.Context, target string, data []byte, probe func(ctx context.Context, stagedPath string) error) error {
	tmpPath, err := stageBinary(target, data)
	if err != nil {
		return err
	}

	// The probe runs after the file is on disk and before the rename, because
	// that is the only point where the new binary is executable and the old one
	// is still installed. A refusal here costs a download and leaves the
	// working binary untouched.
	if probe != nil {
		if err := probe(ctx, tmpPath); err != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("probing staged binary: %w", err)
		}
	}

	return commitStaged(target, tmpPath)
}

// stagedSuffix and asideSuffix are the fixed halves of the per-attempt names
// this package writes next to the target: "<base>.tmp.<pid>" holds a staged
// replacement, and "<base>.old.<pid>" holds the Windows rename-aside. Both
// constructors and the sweep build from these constants, so renaming one name
// cannot leave the sweep hunting for the other — unattended-update.spec §14.
const (
	stagedSuffix = ".tmp."
	asideSuffix  = ".old."
)

// stageBinary writes data to "<base>.tmp.<pid>" next to target and returns that
// path. The caller either commits the staged file or removes it.
//
// The name carries the pid so two attempts cannot collide on one temporary, and
// a sweep of earlier attempts runs first — unattended-update.spec §14. The
// sweep lives here rather than in the unattended policy so the manually typed
// `archcore update` clears leftovers too. It is best-effort: a file another
// live process still holds fails to delete and is skipped, and either way a
// leftover temporary is invisible to a user, who sees only the binary name.
func stageBinary(target string, data []byte) (string, error) {
	dir := filepath.Dir(target)
	base := filepath.Base(target)

	sweepAttemptLeftovers(dir, base)

	tmpPath := filepath.Join(dir, base+stagedSuffix+strconv.Itoa(os.Getpid()))

	// Write + fsync before rename: without the sync, a power loss in the
	// writeback window can leave a zero-length binary with the old one gone.
	if err := writeBinarySynced(tmpPath, data, binaryMode(target)); err != nil {
		return "", err
	}
	return tmpPath, nil
}

// binaryMode returns the mode the replacement binary must carry: the mode of
// the binary being replaced, so a group-managed 0o775 install is not silently
// narrowed to 0o755, and 0o755 for a fresh install.
//
// A target whose mode carries no owner-execute bit is treated as a fresh
// install, so a broken mode is not inherited forward.
func binaryMode(target string) fs.FileMode {
	if info, err := os.Stat(target); err == nil {
		if mode := info.Mode().Perm(); mode&0o100 != 0 {
			return mode
		}
	}
	return 0o755
}

// leftoverGrace is how long a staged temporary is left alone before the sweep
// treats it as abandoned.
//
// It exists because "a file another live process still holds fails to delete"
// is a Windows property, not a POSIX one: os.Remove unlinks a name on Unix
// whatever else has it open. Without a grace period, a typed `archcore update`
// starting while the unattended policy sits in its pre-commit probe would
// unlink the policy's staged binary, and the policy would then fail at
// commitStaged with ENOENT — a spurious failure this surface caused itself.
//
// The budget it covers is one whole unattended attempt (unattendedCeiling,
// 120s) plus room for a loaded machine.
const leftoverGrace = 5 * time.Minute

// sweepAttemptLeftovers removes staged temporaries and Windows asides that
// earlier attempts left next to the target — "<base>.tmp.*" and "<base>.old.*".
// Every failure is ignored: the sweep is housekeeping, and a leftover temporary
// is invisible to a user, who sees only the binary name.
func sweepAttemptLeftovers(dir, base string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, base+stagedSuffix) && !strings.HasPrefix(name, base+asideSuffix) {
			continue
		}
		// A file young enough to belong to an attempt still in flight is left
		// alone. An unreadable entry is skipped for the same reason: without an
		// age there is no evidence it was abandoned.
		info, err := e.Info()
		if err != nil || time.Since(info.ModTime()) < leftoverGrace {
			continue
		}
		// The loop never stops on a failure. One name a live process still
		// holds must not shield the leftovers that sort after it.
		_ = os.Remove(filepath.Join(dir, name))
	}
}

// commitStaged renames the staged file over target, removing it on any failure
// so an abandoned attempt leaves nothing behind. Windows cannot overwrite a
// running .exe, so it commits through commitViaAside instead.
func commitStaged(target, tmpPath string) error {
	if runtime.GOOS == "windows" {
		return commitViaAside(target, tmpPath)
	}

	if err := os.Rename(tmpPath, target); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming temp to target: %w", err)
	}
	// writeBinarySynced made the contents durable; this makes the directory
	// entry that points at them durable too. Best-effort on purpose: the binary
	// is already in place, so a failure here must not report a completed update
	// as failed.
	_ = syncDir(filepath.Dir(target))

	return nil
}

// syncDir fsyncs a directory so a rename inside it survives a power loss.
// Windows exposes no directory handle to sync and answers nil.
func syncDir(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	return d.Sync()
}

// asidePath names the Windows rename-aside for one attempt: "<target>.old.<pid>".
//
// The pid is what makes the failure attributable when a hook starts inside the
// window where the target name is absent — the accepted platform gap in
// unattended-update.spec. A fixed name would also collide when a second update
// starts while an older process still holds the previous aside.
func asidePath(target string) string {
	return target + asideSuffix + strconv.Itoa(os.Getpid())
}

// commitViaAside commits the staged file with two renames: the target moves to
// asidePath, then the staged file takes its name. If the second rename fails,
// the aside rolls back so the user is never left without a binary.
//
// This is the Windows path — a running .exe cannot be overwritten while it
// executes, but it can be renamed. It carries no runtime.GOOS check of its own
// so a test can reach it on any platform; commitStaged owns the dispatch. A
// missing target is a fresh install, not a failure, so the first rename
// tolerates fs.ErrNotExist.
//
// The aside cleanup at the end is best-effort: on Windows it fails while the
// old .exe is still running, and the next attempt's sweep removes it.
func commitViaAside(target, tmpPath string) error {
	oldPath := asidePath(target)
	if err := os.Rename(target, oldPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming target aside: %w", err)
	}
	if err := os.Rename(tmpPath, target); err != nil {
		// Roll back so the user is not left without a binary.
		_ = os.Remove(tmpPath)
		if rollbackErr := os.Rename(oldPath, target); rollbackErr != nil && !errors.Is(rollbackErr, fs.ErrNotExist) {
			return errors.Join(
				fmt.Errorf("renaming temp to target: %w", err),
				fmt.Errorf("rolling back original binary: %w", rollbackErr),
			)
		}
		return fmt.Errorf("renaming temp to target: %w", err)
	}
	_ = os.Remove(oldPath)
	return nil
}

// writeBinarySynced writes data to path with the given mode and fsyncs it
// before close, removing the file on any failure.
//
// The mode is applied with an explicit chmod as well as through the open(2)
// perm argument, because open masks the perm with the process umask and chmod
// does not: under `umask 077` the perm argument alone publishes a 0o700 binary,
// and every other account on the machine loses the command.
func writeBinarySynced(path string, data []byte, mode fs.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("writing temp binary: %w", err)
	}
	if err := f.Chmod(mode); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return fmt.Errorf("setting the mode of the temp binary: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return fmt.Errorf("writing temp binary: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return fmt.Errorf("syncing temp binary: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("closing temp binary: %w", err)
	}
	return nil
}
