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
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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
}

// NewUpdater creates an Updater with sensible defaults.
func NewUpdater(currentVersion, repo, binaryName string) *Updater {
	return &Updater{
		CurrentVersion: currentVersion,
		GitHubRepo:     repo,
		BinaryName:     binaryName,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// tagPathMarker separates the repo path from the tag in a /releases/latest
// redirect: https://github.com/OWNER/REPO/releases/tag/vX.Y.Z
const tagPathMarker = "/releases/tag/"

// maxRedirectBodyDrain caps the bytes read from a redirect body before close.
// A 3xx body is empty, so this still buys connection reuse; an unexpected HTML
// error page can no longer stall the caller (`update --check` runs inside
// editor hooks on a 2s budget).
const maxRedirectBodyDrain = 4 << 10

// client returns the HTTP client to use, falling back to http.DefaultClient
// for a zero-value Updater.
func (u *Updater) client() *http.Client {
	if u.HTTPClient != nil {
		return u.HTTPClient
	}
	return http.DefaultClient
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
func (u *Updater) CheckLatest(ctx context.Context) (string, error) {
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
		resp.Body.Close()
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

// NeedsUpdate compares current and latest versions.
// Returns true if latest is newer than current or if current is "dev".
func NeedsUpdate(current, latest string) bool {
	current = stripV(current)
	latest = stripV(latest)

	if current == "dev" {
		return true
	}

	curParts, curPre := parseSemver(current)
	latParts, latPre := parseSemver(latest)

	if curParts == nil || latParts == nil {
		// Fall back to string comparison if parsing fails.
		return current != latest
	}

	for i := range 3 {
		if latParts[i] > curParts[i] {
			return true
		}
		if latParts[i] < curParts[i] {
			return false
		}
	}

	// Major.minor.patch are equal — compare pre-release.
	// Per SemVer: release (no pre-release) > any pre-release.
	if curPre != "" && latPre == "" {
		return true // current is pre-release, latest is release
	}
	if curPre == "" && latPre != "" {
		return false // current is release, latest is pre-release
	}
	if curPre == "" && latPre == "" {
		return false // both are releases, identical
	}

	return comparePreRelease(curPre, latPre) < 0
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
func (u *Updater) Apply(ctx context.Context, version string) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	archive := ArchiveName(u.BinaryName, goos, goarch)

	// 1. Download the archive.
	archiveData, err := u.download(ctx, version, archive)
	if err != nil {
		return fmt.Errorf("downloading archive: %w", err)
	}

	// 2. Download and verify checksum.
	checksums, err := u.download(ctx, version, "checksums.txt")
	if err != nil {
		return fmt.Errorf("downloading checksums: %w", err)
	}

	if err := VerifyChecksum(archiveData, checksums, archive); err != nil {
		return err
	}

	// 3. Extract binary from the archive.
	// GoReleaser may name the binary either as the configured binary name
	// or as the repo basename; on Windows both carry an .exe suffix.
	binaryData, err := ExtractBinary(archiveData, binaryCandidates(u.BinaryName, u.GitHubRepo, goos)...)
	if err != nil {
		return fmt.Errorf("extracting binary: %w", err)
	}

	// 4. Find current binary path (resolve symlinks).
	execPath := u.ExecPath
	if execPath == "" {
		execPath, err = os.Executable()
		if err != nil {
			return fmt.Errorf("locating current binary: %w", err)
		}
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolving binary path: %w", err)
	}

	// 5. Atomic replace.
	if err := atomicReplace(execPath, binaryData); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
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
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
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
	defer gr.Close()

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
		rc.Close()
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

// atomicReplace writes data to a temporary file next to target, then
// renames it over target for an atomic update.
//
// On Windows the running .exe cannot be overwritten while it executes, but
// it can be renamed. We move it to "<target>.old" first, then rename the
// freshly-written temp file into place. The .old file remains locked until
// this process exits; a best-effort cleanup runs on the next update.
func atomicReplace(target string, data []byte) error {
	dir := filepath.Dir(target)
	base := filepath.Base(target)
	tmpPath := filepath.Join(dir, fmt.Sprintf("%s.tmp.%d", base, os.Getpid()))

	// Write + fsync before rename: without the sync, a power loss in the
	// writeback window can leave a zero-length binary with the old one gone.
	if err := writeBinarySynced(tmpPath, data); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		oldPath := target + ".old"
		_ = os.Remove(oldPath) // sweep stale leftover from a prior update
		if err := os.Rename(target, oldPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			os.Remove(tmpPath)
			return fmt.Errorf("renaming target aside: %w", err)
		}
		if err := os.Rename(tmpPath, target); err != nil {
			// Roll back so the user is not left without a binary.
			os.Remove(tmpPath)
			if rollbackErr := os.Rename(oldPath, target); rollbackErr != nil && !errors.Is(rollbackErr, fs.ErrNotExist) {
				return errors.Join(
					fmt.Errorf("renaming temp to target: %w", err),
					fmt.Errorf("rolling back original binary: %w", rollbackErr),
				)
			}
			return fmt.Errorf("renaming temp to target: %w", err)
		}
		_ = os.Remove(oldPath) // best-effort; will fail while .exe is running
		return nil
	}

	if err := os.Rename(tmpPath, target); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming temp to target: %w", err)
	}

	return nil
}

// writeBinarySynced writes data to path (0o755) and fsyncs it before close,
// removing the file on any failure.
func writeBinarySynced(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("writing temp binary: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(path)
		return fmt.Errorf("writing temp binary: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(path)
		return fmt.Errorf("syncing temp binary: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return fmt.Errorf("closing temp binary: %w", err)
	}
	return nil
}
