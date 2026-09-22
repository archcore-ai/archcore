//go:build unix

package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// The background-task tests re-execute the test binary the way
// TestShieldStdout_FDLevel does. A subprocess is the only honest venue: the
// claim is about what a host reads from the process's real stdout, and the
// shield's fd swap is process-global, so an in-process assertion would be
// measuring a buffer this package handed itself.

const (
	bgHelperEnv  = "ARCHCORE_BGTASK_HELPER"
	bgNoiseEnv   = "ARCHCORE_BGTASK_NOISE"
	bgBlockEnv   = "ARCHCORE_BGTASK_BLOCK"
	bgBaseDirEnv = "ARCHCORE_BGTASK_DIR"

	// The three stdout routes a careless trigger could take, and the stderr
	// marker that tells the parent all of them have already happened.
	bgNoiseRaw   = "TASK-RAW-FD1."
	bgNoisePrint = "TASK-GOPRINT"
	bgNoiseChild = "TASK-CHILD-FD1"
	bgNoiseDone  = "TASK-WROTE-NOISE"
	bgFDReport   = "TASK-FD-IDENTITY "
	bgChildStat  = "TASK-CHILD-SPAWN "

	// What the child writes to every descriptor above 2 it still holds. It can
	// only reach the parent's stdout if the protocol stream was inherited,
	// which close-on-exec exists to prevent.
	bgChildProbe = "TASK-CHILD-INHERITED-FD"

	// The task of the in-flight-attempt session: it reports that it is running
	// and then never finishes.
	bgBlockedMark = "TASK-BLOCKED"

	// Pinned so serverInfo.version is identical across the two runs of the
	// byte-identity test.
	bgHelperVersion = "v0.0.0-bgtask"

	bgHelperTimeout = 30 * time.Second
)

// A session long enough to produce protocol output on both sides of the
// negotiation. tools/list is the wider frame and mcp-go sorts it by tool name,
// so it is reproducible byte for byte.
var bgSessionRequests = []string{
	`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"archcore-bgtask-test","version":"1"}}}`,
	`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
	`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
}

// TestBackgroundTaskHelperProcess is not a test: it is the child half of the
// tests below, re-executed via the test binary. It runs a real RunStdio session
// with, on request, a task that reaches for stdout by every route open to it, or
// one that never finishes.
//
// It exits before the framework can print PASS, which would otherwise land on
// the stdout the parent is asserting about.
func TestBackgroundTaskHelperProcess(t *testing.T) {
	if os.Getenv(bgHelperEnv) != "1" {
		t.Skip("helper process for the background-task subprocess tests")
	}

	var background BackgroundTask
	switch {
	case os.Getenv(bgNoiseEnv) == "1":
		background = BackgroundTask(func(context.Context) {
			// The task's first act, before anything can have moved. Where the
			// noise below lands only shows the shield was up by the time of the
			// write; this shows fd 1 was ALREADY the shielded descriptor the
			// instant the goroutine started, which is the order
			// mcp-background-update.spec §2 fixes.
			_, _ = fmt.Fprintf(os.Stderr, "%s%s %s\n", bgFDReport, fdIdentity(1), fdIdentity(2))

			_, _ = syscall.Write(1, []byte(bgNoiseRaw))
			fmt.Println(bgNoisePrint)
			// The third route, and the one only a descriptor-level shield can
			// close. A real attempt takes it: the policy execs the staged binary
			// for its health probe, so "no child of this task reaches the
			// protocol stream" is part of what the session under test must show.
			// The status line lets the parent tell a contained child from one
			// that never ran.
			if err := bgSpawnOnInheritedFD1(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "%serr %v\n", bgChildStat, err)
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "%sok\n", bgChildStat)
			}
			// Written last, and the parent holds stdin open until it sees this,
			// so every write above is guaranteed to have happened while the
			// shield was still in place. Without that handshake the session
			// could end first and the test would prove nothing.
			_, _ = os.Stderr.WriteString(bgNoiseDone + "\n")
		})

	case os.Getenv(bgBlockEnv) == "1":
		background = BackgroundTask(func(context.Context) {
			_, _ = os.Stderr.WriteString(bgBlockedMark + "\n")
			// Never returns, and deliberately does not watch ctx: an attempt
			// mid-download is unreachable by cancellation too, and the session
			// must not care either way — mcp-background-update.spec §4, §5, §10.
			select {}
		})
	}

	_ = RunStdio(context.Background(), os.Getenv(bgBaseDirEnv), bgHelperVersion, background)
	os.Exit(0)
}

// bgSpawnOnInheritedFD1 runs a child that writes to its own stdout, with the
// parent's fds 0/1/2 handed over unchanged. The child then writes to every
// higher descriptor it might have inherited, which reaches the host only if the
// protocol stream survived the exec.
//
// ForkExec rather than os/exec: exec.Cmd resolves a nil Stdout to the null
// device and an os.Stdout to whatever Go currently holds there, and neither is
// the case under test. Plain descriptor inheritance is, and only the unix
// shield's dup2 of fd 1 can contain it.
//
// It waits for the child, so the caller's later stderr marker still orders the
// child's writes ahead of the parent's assertions.
func bgSpawnOnInheritedFD1() error {
	// Each probe runs in a subshell with its errors discarded, and the script
	// ends in exit 0: writing to a descriptor the shield closed is the expected
	// case, so neither its diagnostic nor its status may read as a failure to
	// spawn. What the first echo reached is asserted on the streams.
	script := "echo " + bgNoiseChild +
		"; for fd in 3 4 5 6 7 8 9; do (echo " + bgChildProbe + " >&$fd) 2>/dev/null; done; exit 0"

	pid, err := syscall.ForkExec("/bin/sh", []string{"sh", "-c", script},
		&syscall.ProcAttr{Files: []uintptr{0, 1, 2}})
	if err != nil {
		return err
	}
	var ws syscall.WaitStatus
	if _, err := syscall.Wait4(pid, &ws, 0, nil); err != nil {
		return err
	}
	if ws.ExitStatus() != 0 {
		return fmt.Errorf("child exited %d", ws.ExitStatus())
	}
	return nil
}

// fdIdentity names the open file a descriptor points at. Two descriptors report
// the same name only when one is a dup of the other, which is exactly what the
// shield does to fd 1.
func fdIdentity(fd int) string {
	var st syscall.Stat_t
	if err := syscall.Fstat(fd, &st); err != nil {
		return "fstat-failed"
	}
	// No conversion: Dev is int32 on darwin and uint64 on linux, so any explicit
	// width is redundant on one of them. %d takes either.
	return fmt.Sprintf("%d:%d", st.Dev, st.Ino)
}

// bgSession drives one helper subprocess: it feeds requests on stdin and returns
// the bytes the parent observed on each stream.
type bgSession struct {
	baseDir string
	noise   bool
	// block installs a task that never returns, so the session runs its whole
	// course with an attempt still in flight.
	block bool
	// awaitStderr, when set, is a marker the parent waits for before it closes
	// stdin. It is how a test keeps the session alive past a background write.
	awaitStderr string
	// awaitFirst moves that wait ahead of the first request, which turns
	// "the task had started" into a fact about every frame on stdout rather
	// than a race the assertions would have to tolerate.
	awaitFirst bool
}

func (s bgSession) run(t *testing.T) (stdout, stderr []byte) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), bgHelperTimeout)
	defer cancel()

	// Anchored at both ends. Without the leading ^, any future test whose name
	// ends with this one's would also be selected in the child — the same form
	// internal/update's re-exec uses.
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestBackgroundTaskHelperProcess$")
	cmd.Env = append(os.Environ(),
		bgHelperEnv+"=1",
		bgBaseDirEnv+"="+s.baseDir,
	)
	if s.noise {
		cmd.Env = append(cmd.Env, bgNoiseEnv+"=1")
	}
	if s.block {
		cmd.Env = append(cmd.Env, bgBlockEnv+"=1")
	}

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}

	var outBuf, errBuf bytes.Buffer
	outDone := make(chan struct{})
	go func() {
		defer close(outDone)
		_, _ = io.Copy(&outBuf, stdoutPipe)
	}()

	marked := make(chan struct{})
	errDone := make(chan struct{})
	go func() {
		defer close(errDone)
		buf := make([]byte, 4096)
		seen := s.awaitStderr == ""
		for {
			n, readErr := stderrPipe.Read(buf)
			if n > 0 {
				errBuf.Write(buf[:n])
				if !seen && bytes.Contains(errBuf.Bytes(), []byte(s.awaitStderr)) {
					seen = true
					close(marked)
				}
			}
			if readErr != nil {
				return
			}
		}
	}()

	// Both failure paths below end the test without reaching the Wait at the end
	// of this function, which would leave the child's pipe-copying goroutines
	// running. Killing the child releases them; the deferred cancel would do it
	// too, but only once this function's context is unwound, and t.Fatalf does
	// not unwind it from a helper.
	failWithoutLeaking := func(format string, args ...any) {
		cancel()
		_ = cmd.Wait()
		t.Fatalf(format, args...)
	}

	awaitMark := func() {
		if s.awaitStderr == "" {
			return
		}
		select {
		case <-marked:
		case <-outDone:
			failWithoutLeaking("helper exited before writing %q to stderr", s.awaitStderr)
		case <-ctx.Done():
			failWithoutLeaking("helper never wrote %q to stderr", s.awaitStderr)
		}
	}

	if s.awaitFirst {
		awaitMark()
	}
	for _, req := range bgSessionRequests {
		if _, err := io.WriteString(stdinPipe, req+"\n"); err != nil {
			t.Fatalf("write request: %v", err)
		}
	}
	if !s.awaitFirst {
		awaitMark()
	}
	_ = stdinPipe.Close() // EOF ends the session

	// Both pipes must be drained before Wait: Wait closes them, and a read
	// racing that close loses the tail of the very output under test.
	<-outDone
	<-errDone
	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			t.Fatalf("helper never exited within %s — the session is waiting on something it must not wait on\nstderr:\n%s",
				bgHelperTimeout, errBuf.String())
		}
		t.Fatalf("helper failed: %v\nstderr:\n%s", err, errBuf.String())
	}
	return outBuf.Bytes(), errBuf.Bytes()
}

// bgReportedFDs pulls the two identities the task wrote at its first
// instruction out of the child's stderr.
func bgReportedFDs(stderr []byte) (fd1, fd2 string, ok bool) {
	for _, line := range bytes.Split(stderr, []byte("\n")) {
		rest, found := bytes.CutPrefix(line, []byte(bgFDReport))
		if !found {
			continue
		}
		parts := bytes.Fields(rest)
		if len(parts) != 2 {
			return "", "", false
		}
		return string(parts[0]), string(parts[1]), true
	}
	return "", "", false
}

// bgChildSpawnStatus reports how the task's fork went, so a child that never
// ran cannot pass for a child the shield contained.
func bgChildSpawnStatus(stderr []byte) (string, bool) {
	for _, line := range bytes.Split(stderr, []byte("\n")) {
		if rest, found := bytes.CutPrefix(line, []byte(bgChildStat)); found {
			return string(rest), true
		}
	}
	return "", false
}

// The task starts behind the shield, so no route to stdout reaches the host:
// raw fd-1 writes, fmt prints and a child holding the inherited descriptor all
// land on stderr, and stdout carries JSON-RPC frames only.
// mcp-background-update.spec §2 and §6.
func TestRunStdio_BackgroundTaskWritesBehindShield(t *testing.T) {
	t.Parallel()

	stdout, stderr := bgSession{
		baseDir:     t.TempDir(),
		noise:       true,
		awaitStderr: bgNoiseDone,
	}.run(t)

	// The ordering claim, checked first because it is the precise one: the
	// parent's stdout and stderr are separate pipes, so fd 1 and fd 2 name the
	// same open file in the child only after the shield has aliased them.
	fd1, fd2, ok := bgReportedFDs(stderr)
	if !ok {
		t.Fatalf("task never reported its fd identities, stderr:\n%s", stderr)
	}
	if fd1 != fd2 {
		t.Errorf("task started before the shield: at its first instruction fd 1 was %s and fd 2 was %s", fd1, fd2)
	}

	// Without protocol bytes the checks below would pass on an empty stdout,
	// which proves nothing about a shield.
	if len(stdout) == 0 {
		t.Fatal("session produced no protocol output; the shield assertions would be vacuous")
	}
	for _, frame := range bytes.Split(bytes.TrimRight(stdout, "\n"), []byte("\n")) {
		var probe struct {
			JSONRPC string `json:"jsonrpc"`
		}
		if err := json.Unmarshal(frame, &probe); err != nil || probe.JSONRPC != "2.0" {
			t.Errorf("stdout must carry JSON-RPC frames only, got line %q", frame)
		}
	}

	// The child must have run: without this the two assertions below hold
	// trivially on a fork that failed.
	if status, ok := bgChildSpawnStatus(stderr); !ok || status != "ok" {
		t.Fatalf("the task's child never ran (status %q, reported %v); the child-route assertions would be vacuous\nstderr:\n%s",
			status, ok, stderr)
	}

	for _, route := range []struct{ name, marker string }{
		{"raw fd-1 write", bgNoiseRaw},
		{"fmt print", bgNoisePrint},
		{"child process on the inherited descriptor", bgNoiseChild},
	} {
		if bytes.Contains(stdout, []byte(route.marker)) {
			t.Errorf("%s from the task reached stdout:\n%s", route.name, stdout)
		}
		if !bytes.Contains(stderr, []byte(route.marker)) {
			t.Errorf("%s must be diverted to stderr, stderr:\n%s", route.name, stderr)
		}
	}

	// The protocol stream itself must not survive the exec. Nothing else guards
	// the CloseOnExec on the duplicated descriptor: without it a child of the
	// task holds a second writer on the host's frame stream, and the shield's
	// dup2 of fd 1 does not help.
	if bytes.Contains(stdout, []byte(bgChildProbe)) {
		t.Errorf("the task's child inherited the protocol stream and wrote to it:\n%s", stdout)
	}
}

// The server answers on its own schedule while an attempt is in flight, and the
// session ends without waiting for one. The task here never returns, so a
// RunStdio that joined its goroutine — or that started it before serving rather
// than beside it — hangs until the helper is killed instead of replying.
// mcp-background-update.spec §4, §5 and §10.
func TestRunStdio_ServesAndEndsWithTheAttemptStillInFlight(t *testing.T) {
	t.Parallel()

	// awaitFirst: the marker lands before the first request is written, so both
	// replies below were produced with the attempt already running.
	stdout, _ := bgSession{
		baseDir:     t.TempDir(),
		block:       true,
		awaitStderr: bgBlockedMark,
		awaitFirst:  true,
	}.run(t)

	answered := map[float64]bool{}
	for _, frame := range bytes.Split(bytes.TrimRight(stdout, "\n"), []byte("\n")) {
		var reply struct {
			ID     *float64        `json:"id"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(frame, &reply); err != nil {
			t.Fatalf("stdout must carry JSON-RPC frames only, got line %q", frame)
		}
		if reply.ID != nil && len(reply.Result) > 0 {
			answered[*reply.ID] = true
		}
	}
	// initialize is readiness; tools/list is a request served after it.
	for _, id := range []float64{1, 2} {
		if !answered[id] {
			t.Errorf("request id %v went unanswered while the attempt was in flight; stdout:\n%s", id, stdout)
		}
	}
}

// The release criterion for the seam: a host must not be able to tell from its
// side of the pipe whether a background task was wired at all. Anything weaker
// than byte identity — a Contains check, a frame count — would still pass while
// a stray byte corrupted a frame boundary. mcp-background-update.spec §6, and
// the "serves exactly as today" requirement for the option-less path.
func TestRunStdio_StdoutIdenticalWithAndWithoutBackgroundTask(t *testing.T) {
	t.Parallel()

	// One base directory for both runs: instructions are built from its
	// settings and global mounts, so two directories would differ for a reason
	// that has nothing to do with the background task.
	dir := t.TempDir()

	withTask, _ := bgSession{baseDir: dir, noise: true, awaitStderr: bgNoiseDone}.run(t)
	withoutTask, _ := bgSession{baseDir: dir}.run(t)

	if len(withoutTask) == 0 {
		t.Fatal("baseline session produced no stdout; byte identity would be vacuous")
	}
	if !bytes.Equal(withTask, withoutTask) {
		t.Fatalf("stdout differs when a background task is wired\nwith task (%d bytes):\n%s\nwithout task (%d bytes):\n%s",
			len(withTask), withTask, len(withoutTask), withoutTask)
	}
}
