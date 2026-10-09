package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"archcore-cli/internal/agents"
	"archcore-cli/internal/config"
)

func TestNewServer_HasTools(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	for _, sub := range []string{"vision", "knowledge", "experience"} {
		if err := os.MkdirAll(filepath.Join(base, ".archcore", sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	s := NewServer(base, "test")
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestBuildInstructions_DefaultEnglish(t *testing.T) {
	t.Parallel()
	for _, lang := range []string{"", "en"} {
		if strings.Contains(buildInstructions(lang, nil, nil), "LANGUAGE:") {
			t.Errorf("buildInstructions(%q): an English project must receive no language notice", lang)
		}
	}
}

func TestBuildInstructions_NonEnglish(t *testing.T) {
	t.Parallel()
	for _, lang := range []string{"ru", "ja", "de"} {
		result := buildInstructions(lang, nil, nil)
		if !strings.Contains(result, "LANGUAGE: write titles and bodies in \""+lang+"\"") {
			t.Errorf("buildInstructions(%q): the language notice does not name the language", lang)
		}
		for _, token := range []string{`"##" headings`, "MUST NOT", "WHERE"} {
			if !strings.Contains(result, token) {
				t.Errorf("buildInstructions(%q): the structure-token list should name %q", lang, token)
			}
		}
	}
}

// TestBuildInstructions_AddressLeadsAndNoticesPrecedeRules pins the order a
// truncating host depends on — context-address-delivery.spec §4 and §5: the
// address comes first and every notice comes before the working rules.
func TestBuildInstructions_AddressLeadsAndNoticesPrecedeRules(t *testing.T) {
	t.Parallel()
	got := buildInstructions("ru", []string{"org"}, []string{"gone"})
	address := strings.Index(got, agents.ContextAddress)
	rules := strings.Index(got, "READ:")
	if address < 0 || address > 100 {
		t.Fatalf("the context address is not at the start of the instructions:\n%s", got)
	}
	for _, notice := range []string{"GLOBAL SOURCES", "NOT CLONED", "LANGUAGE:"} {
		if i := strings.Index(got, notice); i < address || i > rules {
			t.Errorf("%s does not sit between the address and the rules", notice)
		}
	}
}

// TestBuildInstructions_FitTheHostCap holds context-address-delivery.spec §5 on
// the largest instructions a project can produce: many declared sources, some
// mounted and some not, and a non-English language.
func TestBuildInstructions_FitTheHostCap(t *testing.T) {
	t.Parallel()
	many := []string{"archcore", "platform-standards", "security-baseline", "team-frontend", "team-backend", "data-contracts", "mobile"}
	for _, tt := range []struct {
		name     string
		language string
		mounted  []string
		missing  []string
	}{
		{name: "bare", language: ""},
		{name: "worst case", language: "zh-Hans", mounted: many, missing: many},
	} {
		got := buildInstructions(tt.language, tt.mounted, tt.missing)
		if units := len(utf16.Encode([]rune(got))); units > hostInstructionsCap {
			t.Errorf("%s: the instructions hold %d UTF-16 units; the host passes on %d", tt.name, units, hostInstructionsCap)
		}
	}
}

// TestBuildInstructions_CarriesTheTypeContracts pins the clauses other accepted
// specs place in the server instructions: research-and-evidence-types.spec §10
// and §11, evidential-and-temporal-relations.spec §8, and
// scenario-and-journey-types.spec §10 to §12.
func TestBuildInstructions_CarriesTheTypeContracts(t *testing.T) {
	t.Parallel()
	got := buildInstructions("", nil, nil)
	for _, phrase := range []string{
		"rnd closes on a recommendation",
		"research on scope coverage",
		"evidence records one external material",
		"structural", "evidential", "temporal",
		"material → the claim it backs or disputes",
		"newer → older",
		"a journey is a user path no spec covers yet",
		"a scenario step takes the actor as subject",
		"scenario depends_on spec", "scenario implements journey", "journey related prd",
	} {
		if !strings.Contains(got, phrase) {
			t.Errorf("the instructions lack %q", phrase)
		}
	}
}

func TestJoinSourceIDs_StatesTheRemainder(t *testing.T) {
	t.Parallel()
	many := []string{"archcore", "platform-standards", "security-baseline", "team-frontend", "team-backend", "data-contracts"}
	got := joinSourceIDs(many)
	if len(got) > sourceListCap+len(" and 9 more") {
		t.Errorf("joinSourceIDs = %q, longer than the cap", got)
	}
	if !strings.HasSuffix(got, " more") {
		t.Errorf("joinSourceIDs = %q, does not state the remainder", got)
	}
	if got := joinSourceIDs([]string{"org"}); got != "org" {
		t.Errorf("joinSourceIDs([org]) = %q, want org", got)
	}
}

func TestNewServer_WithLanguageSetting(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".archcore"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(base, ".archcore", "settings.json"),
		[]byte(`{"sync":"none","language":"ru"}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	s := NewServer(base, "test")
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestNewServer_MissingSettings_FallsBack(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	// No .archcore/settings.json — server should still create successfully.
	s := NewServer(base, "test")
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
}

// runStdioTimeout bounds every wait in the RunStdio tests. They drive a real
// server loop, so a regression must surface as a failure, never as a suite that
// hangs until the CI job is killed.
const runStdioTimeout = 10 * time.Second

// The three tests below drive RunStdio in this process, which swaps fd 1
// process-wide for the duration. None of them may call t.Parallel(): Go releases
// parallel tests only after the serial ones finish, and that ordering is what
// keeps the swap from landing under another test's output.

// The background task must receive the session's own context. The trigger's
// cancellation story — a host that closes stdio before the delay elapses gets no
// update attempt at all — is unenforceable if RunStdio hands the task a detached
// context: the task would outlive the session and reach the policy anyway.
// mcp-background-update.spec §9.
func TestRunStdio_BackgroundTaskInheritsSessionContext(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	taskCtx := make(chan context.Context, 1)
	served := make(chan error, 1)
	go func() {
		served <- RunStdio(ctx, dir, "test",
			func(c context.Context) { taskCtx <- c })
	}()

	var got context.Context
	select {
	case got = <-taskCtx:
	case <-time.After(runStdioTimeout):
		t.Fatal("RunStdio never started the background task")
	}

	cancel()

	select {
	case <-got.Done():
	case <-time.After(runStdioTimeout):
		t.Fatal("cancelling RunStdio's context left the task's context live")
	}
	select {
	case <-served:
	case <-time.After(runStdioTimeout):
		t.Fatal("RunStdio did not return after its context was cancelled")
	}
}

// Every existing caller and embedder passes no background task, so a nil task
// must be skipped rather than invoked — a nil call would take the whole process
// down with the session. The byte-level half of "serves exactly as today" is
// TestRunStdio_StdoutIdenticalWithAndWithoutBackgroundTask.
func TestRunStdio_WithoutABackgroundTask_ReturnsWithoutPanicking(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	served := make(chan error, 1)
	go func() { served <- RunStdio(ctx, dir, "test", nil) }()

	select {
	case <-served:
	case <-time.After(runStdioTimeout):
		t.Fatal("RunStdio did not return under a cancelled context")
	}
}

// RunStdio delegates option application to NewServer, so each option runs once.
// The hazard this pins is the shape RunStdio used to have: it applied the
// options itself to reach a field NewServer ignored, and the obvious repair —
// keep the NewServer call and apply them again — would run every
// side-effecting option twice.
func TestRunStdio_AppliesEachOptionOnce(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	applied := make(chan struct{}, 8)
	count := ServerOption(func(*serverConfig) { applied <- struct{}{} })

	served := make(chan error, 1)
	go func() { served <- RunStdio(ctx, dir, "test", nil, count, count) }()

	select {
	case <-served:
	case <-time.After(runStdioTimeout):
		t.Fatal("RunStdio did not return under a cancelled context")
	}
	if got := len(applied); got != 2 {
		t.Errorf("two options passed: applied %d times, want 2", got)
	}
}

// TestRunStdio_CancelsTheBackgroundTaskWhenItReturns closes the lifetime gap.
//
// Listen returns on stdin EOF as well as on cancellation, and EOF is how a host
// normally ends a session. Before RunStdio derived the task's context, a task
// started under a still-live parent context kept running after RunStdio
// returned and restore() closed the protocol stream — with a context that still
// reported itself live. `archcore mcp` exits immediately so nothing showed, but
// the in-process tests and any embedder kept the goroutine.
//
// The task is cancelled, never joined: mcp-background-update.spec §10 says a
// session must not wait on an attempt.
func TestRunStdio_CancelsTheBackgroundTaskWhenItReturns(t *testing.T) {
	dir := t.TempDir()

	taskCtx := make(chan context.Context, 1)
	served := make(chan error, 1)
	go func() {
		// stdin is already at EOF under `go test`, so Listen returns on its own
		// with the parent context still live — the exact shape that leaked.
		served <- RunStdio(context.Background(), dir, "test",
			func(c context.Context) { taskCtx <- c })
	}()

	var got context.Context
	select {
	case got = <-taskCtx:
	case <-time.After(runStdioTimeout):
		t.Fatal("RunStdio never started the background task")
	}
	select {
	case <-served:
	case <-time.After(runStdioTimeout):
		t.Fatal("RunStdio did not return on stdin EOF")
	}

	select {
	case <-got.Done():
	case <-time.After(runStdioTimeout):
		t.Error("RunStdio returned but left the background task's context live")
	}
}

// TestBuildInstructions_EmptySearchRule pins the empty-result rule of
// search-documents.spec §13 in the instructions every project receives, and the
// trigger of the global retry in the notice a project with globals adds.
func TestBuildInstructions_EmptySearchRule(t *testing.T) {
	t.Parallel()
	if !strings.Contains(buildInstructions("", nil, nil), "near_misses") {
		t.Error("a project without globals must receive the empty-result rule")
	}
	if !strings.Contains(buildInstructions("", []string{"org"}, nil), "If a retry returns only local rows") {
		t.Error("the global retry does not state its trigger: a page of only local rows")
	}
}

// TestGlobalIDsByPresence pins missing-global-degrades-to-local.adr on the
// instructions: a declared source that is not on disk must not be announced as
// mounted.
func TestGlobalIDsByPresence(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".archcore", "global", "org"), 0o755); err != nil {
		t.Fatal(err)
	}
	mounted, missing := globalIDsByPresence(base, []config.GlobalSource{
		{ID: "org", Path: ".archcore/global/org"},
		{ID: "gone", Path: ".archcore/global/gone"},
	})
	if len(mounted) != 1 || mounted[0] != "org" {
		t.Errorf("mounted = %v, want [org]", mounted)
	}
	if len(missing) != 1 || missing[0] != "gone" {
		t.Errorf("missing = %v, want [gone]", missing)
	}
}

func TestBuildInstructions_MissingGlobalIsNotMounted(t *testing.T) {
	t.Parallel()
	got := buildInstructions("", nil, []string{"gone"})
	if strings.Contains(got, "GLOBAL SOURCES") {
		t.Error("a source that is not on disk is announced as mounted")
	}
	if !strings.Contains(got, "NOT CLONED: gone.") {
		t.Error("the instructions do not name the source that is not on disk")
	}
}
