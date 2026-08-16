// Package telemetry sends anonymous product events to Archcore's first-party
// PostHog proxy.
//
// The contract is cli-update-telemetry.spec. Four
// properties of this package are load-bearing and must survive any refactor:
//
//  1. A build without a release key is inert. The key arrives through an -X
//     ldflag at release time, so a `go build`, a `go install`, a fork and a CI
//     build send nothing by construction — the same property install.sh gets
//     from deploy-time substitution.
//  2. Opting out leaves no trace on disk. All three guards run before the
//     install-id file is read or created.
//  3. Nothing here can fail a command. Capture returns no error, prints
//     nothing, and is safe on a nil receiver, so a caller needs no branch for
//     "telemetry is off".
//  4. The install-id file is shared with install.sh and install.ps1, so one
//     machine resolves to one distinct_id across the installer and the CLI.
//
// Capture reports delivery so the caller can print a disclosure line only for
// an event the endpoint accepted. A dropped event and a delivered one must not
// look the same in the terminal.
package telemetry

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"archcore-cli/internal/xdg"
)

// apiKey is injected at release build time:
//
//	-X archcore-cli/internal/telemetry.apiKey=phc_...
//
// It is empty in every other build, which is what makes those builds inert.
var apiKey string

const (
	// defaultEndpoint is the PostHog capture path install.sh already posts to.
	defaultEndpoint = "https://ph.archcore.ai/i/v0/e/"

	// keyPrefix marks a real PostHog project key. The guard tests the prefix
	// rather than comparing against a placeholder, so the release substitution
	// can never rewrite its own off-switch.
	keyPrefix = "phc_"

	libName   = "archcore-cli"
	sourceCLI = "cli"

	// installIDFile lives beside the update-check cache, under the same XDG
	// rules install_id_path() in install.sh applies.
	installIDFile = "install-id"

	// connectTimeout and totalTimeout mirror `curl --connect-timeout 2
	// --max-time 3` in install.sh. An update must never wait on analytics.
	connectTimeout = 2 * time.Second
	totalTimeout   = 3 * time.Second

	// maxResponseDrain caps the bytes read from the response before close. The
	// body is discarded either way; draining a little of it buys connection
	// reuse without letting an unexpected page stall the caller.
	maxResponseDrain = 4 << 10
)

// Client sends events for one CLI invocation.
//
// The zero value sends nothing: without a key the first guard refuses. Every
// field other than Version exists as a test seam and is resolved from a
// default when empty.
type Client struct {
	Version    string       // reported as $lib_version
	Key        string       // empty: the package key injected at release time
	Endpoint   string       // empty: defaultEndpoint
	HTTPClient *http.Client // nil: a client bounded by connect and total timeouts
	StateDir   string       // empty: xdg.StateDir(); isolates the install-id in tests
}

// NewClient returns a Client for the running binary's version.
func NewClient(version string) *Client {
	return &Client{Version: version}
}

// Capture sends one event and reports whether the endpoint accepted it.
//
// It is safe on a nil receiver. It returns true only on a 2xx response, which
// is what the caller gates its disclosure line on. It never returns an error,
// never prints, and never blocks longer than its timeouts. A refused guard, an
// unavailable identifier, a timeout, a DNS failure and a non-2xx status are all
// the same outcome to the caller: false, and the command carries on.
func (c *Client) Capture(ctx context.Context, event Event, props map[string]any) bool {
	if c == nil {
		return false
	}

	// Guard order is normative: all three decide before any filesystem access,
	// so an opted-out machine never gets an install-id file written to it.
	key := c.key()
	if !enabled(key) {
		return false
	}

	id := c.installID()
	if id == "" {
		// The identifier could be neither read nor created. A one-off id would
		// count this machine again on every run, so skip the event instead.
		return false
	}

	payload, err := json.Marshal(map[string]any{
		"api_key":     key,
		"event":       string(event),
		"distinct_id": id,
		"properties":  c.properties(props),
	})
	if err != nil {
		return false
	}

	// The bound holds even for a caller that passes context.Background() and
	// even for an injected client that sets no timeout of its own.
	ctx, cancel := context.WithTimeout(ctx, totalTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), bytes.NewReader(payload))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return false
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseDrain))
		_ = resp.Body.Close()
	}()

	return resp.StatusCode/100 == 2
}

// key resolves the effective project key: the field first, then the key
// injected at build time.
func (c *Client) key() string {
	if c.Key != "" {
		return c.Key
	}
	return apiKey
}

func (c *Client) endpoint() string {
	if c.Endpoint != "" {
		return c.Endpoint
	}
	return defaultEndpoint
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return defaultHTTPClient()
}

// defaultHTTPClient is built once and shared. Capture sends at most one event
// per invocation, so this exists to keep the timeouts in one place rather than
// to pool connections.
var defaultHTTPClient = sync.OnceValue(func() *http.Client {
	return &http.Client{
		Timeout: totalTimeout,
		Transport: &http.Transport{
			// install.sh reaches the endpoint through curl, which honors
			// http_proxy/https_proxy/no_proxy. A hand-built Transport defaults
			// to no proxy at all, so on a machine that only egresses through one
			// the installer would report and the CLI would not — every event
			// from every proxied machine lost, and a 2 s connect stall added to
			// each update. This is the field that keeps the two in step.
			Proxy:               http.ProxyFromEnvironment,
			DialContext:         (&net.Dialer{Timeout: connectTimeout}).DialContext,
			TLSHandshakeTimeout: connectTimeout,
			ForceAttemptHTTP2:   true,
		},
	}
})

// enabled holds the three guards, in the order the spec fixes: an unofficial
// build first, then the two opt-out variables.
func enabled(key string) bool {
	if !strings.HasPrefix(key, keyPrefix) {
		return false
	}
	if optedOut(os.Getenv("DO_NOT_TRACK")) {
		return false
	}
	if optedOut(os.Getenv("ARCHCORE_TELEMETRY_OPTOUT")) {
		return false
	}
	return true
}

// optedOut mirrors telemetry_enabled() in install.sh: any value other than
// empty or "0" opts out. consoledonottrack.com defines DO_NOT_TRACK that way.
func optedOut(value string) bool {
	return value != "" && value != "0"
}

// properties merges the caller's event properties over the common ones. The
// caller wins on a collision: an event that needs to restate a common property
// knows more about the invocation than this package does.
func (c *Client) properties(props map[string]any) map[string]any {
	// `$lib` is what PostHog's Library column reads. It duplicates `source` on
	// purpose: source stays the stable field to query on, this one exists so
	// the built-in column is not empty.
	merged := map[string]any{
		"$lib":         libName,
		"$lib_version": c.Version,
		"source":       sourceCLI,
		"os":           runtime.GOOS,
		"arch":         runtime.GOARCH,
		"ci":           isCI(),
	}
	for k, v := range props {
		merged[k] = v
	}
	return merged
}

// Event is one name in the update-telemetry wire vocabulary.
//
// These strings are a contract with PostHog and with the landing repository's
// ExternalAnalyticsEventMap: renaming one silently empties a dashboard rather
// than failing a build — cli-update-telemetry.spec. They live here, once,
// because both senders are bound by the same contract — the unattended policy
// in internal/update and the typed `archcore update` in cmd — and each once
// carried its own untyped copy, which is how two spellings of one wire name
// come to exist without a compiler ever objecting.
type Event string

const (
	EventUpdated       Event = "cli_updated"
	EventUpdateFailed  Event = "cli_update_failed"
	EventUpdateSkipped Event = "cli_update_skipped"
)

// Trigger grades what caused an invocation: what a user typed, against what a
// mechanism did. Keeping those apart is the whole purpose of the property —
// cli-update-telemetry.spec §10.
type Trigger string

const (
	TriggerAuto   Trigger = "auto"
	TriggerManual Trigger = "manual"
)

// SkipReason is the reportable reason an unattended attempt stopped without
// replacing anything. These two are the only ones — unattended-update.spec.
type SkipReason string

const (
	SkipReasonCurrent     SkipReason = "current"
	SkipReasonNotWritable SkipReason = "not_writable"
)

// CIVars mirrors the set install.sh's is_ci() enumerates. A CI runner is
// ephemeral and mints a fresh identifier per run, so events from one have to be
// separable from events from a real machine.
//
// One declaration, three readers: this package grades an event, the unattended
// update policy decides whether to replace a binary at all, and the plugin step
// decides whether to run a host command. Those decisions stay separate and are
// spelled out where they are made. "Which environment variables mean CI" is not
// a decision, it is one fact, and three copies of it meant a fourth provider
// took three edits with nothing to catch a missed one.
var CIVars = []string{
	"CI",
	"GITHUB_ACTIONS",
	"GITLAB_CI",
	"BUILDKITE",
	"JENKINS_URL",
	"TEAMCITY_VERSION",
}

// DetectedCI reports whether any variable in CIVars carries a value.
func DetectedCI() bool {
	for _, name := range CIVars {
		if os.Getenv(name) != "" {
			return true
		}
	}
	return false
}

// isCI is the in-package spelling of DetectedCI, kept so the grading call sites
// below read the same as they did.
func isCI() bool { return DetectedCI() }

// stateDir resolves the directory holding the install-id.
func (c *Client) stateDir() string {
	if c.StateDir != "" {
		return c.StateDir
	}
	return xdg.StateDir()
}

// installID returns the per-machine identifier, creating it on first use. An
// empty result means the identifier is unavailable and the event must be
// skipped.
//
// The identifier is random and opaque: it is derived from no host name, user
// name or hardware id, so it carries nothing about the machine it names.
func (c *Client) installID() string {
	dir := c.stateDir()
	if dir == "" {
		return ""
	}
	path := filepath.Join(dir, installIDFile)

	if id := readInstallID(path); id != "" {
		return id
	}

	id, err := newInstallID()
	if err != nil {
		return ""
	}
	if os.MkdirAll(dir, 0o755) != nil {
		return ""
	}
	return createInstallID(path, id)
}

// createInstallID persists id and returns the identifier this machine now
// answers to — which is not always the id passed in.
//
// The create is exclusive because two invocations can reach here at once on a
// fresh machine: the MCP trigger and a typed `archcore update`, or the CLI and
// install.sh. A plain write would let the second one clobber the first, and the
// machine would report two distinct_ids for the same minute. Exclusive create
// makes the loser adopt the winner's id instead.
//
// An existing file that holds something other than an identifier is still
// replaced: that is behavior 9's "unreadable" case, and no identity is lost by
// overwriting content that never named a machine.
func createInstallID(path, id string) string {
	// The trailing newline matches what install.sh writes, so a file created
	// here and a file created there are byte-identical in shape.
	line := []byte(id + "\n")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if errors.Is(err, fs.ErrExist) {
		if existing := readInstallID(path); existing != "" {
			return existing
		}
		if os.WriteFile(path, line, 0o644) != nil {
			return ""
		}
		return id
	}
	if err != nil {
		return ""
	}
	_, writeErr := f.Write(line)
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		return ""
	}
	return id
}

// readInstallID returns the stored identifier, or an empty string when the file
// is absent, unreadable, or holds something that is not an identifier.
//
// A valid id is used verbatim. install.sh writes this file too, and rewriting
// or reformatting it would split one machine into two distinct_ids.
func readInstallID(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	id := strings.TrimSpace(string(data))
	if !isInstallID(id) {
		return ""
	}
	return id
}

// newInstallID returns installIDLen lowercase hexadecimal characters — the
// format install.sh and install.ps1 write.
func newInstallID() (string, error) {
	var b [installIDLen / 2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// installIDLen is the length of the identifier install.sh, install.ps1 and
// newInstallID all write: 16 random bytes, hex encoded.
const installIDLen = 32

// isInstallID reports whether s is exactly the identifier format this file and
// both installers write.
//
// The length is load-bearing, not decoration. Accepting any run of hex digits
// meant a short prefix validated: a reader landing inside the truncate-and-write
// window of the fs.ErrExist branch in createInstallID, or inside the gap between
// the exclusive create and the first Write, took the partial content as a real
// identifier and the machine reported a second distinct_id — the exact thing
// the exclusive create exists to prevent. A wrong length is a partial write, so
// it reads as "no identifier yet" and the caller falls through to the create.
func isInstallID(s string) bool {
	if len(s) != installIDLen {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}
