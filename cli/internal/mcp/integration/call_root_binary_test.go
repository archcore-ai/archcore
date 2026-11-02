package integration

// End to end through the built binary: `archcore mcp` on stdio, started the
// way a host starts it — from a working directory, or pinned with --project —
// and spoken to in line-delimited JSON-RPC.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	binaryOnce     sync.Once
	binaryPath     string
	errBinaryBuild error
)

// archcoreBinary builds the CLI once per test process.
func archcoreBinary(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("builds the CLI binary")
	}
	binaryOnce.Do(func() {
		dir, err := os.MkdirTemp("", "archcore-e2e-")
		if err != nil {
			errBinaryBuild = err
			return
		}
		binaryPath = filepath.Join(dir, "archcore")
		out, err := exec.Command("go", "build", "-o", binaryPath, "archcore-cli").CombinedOutput()
		if err != nil {
			errBinaryBuild = fmt.Errorf("go build: %v\n%s", err, out)
		}
	})
	if errBinaryBuild != nil {
		t.Fatal(errBinaryBuild)
	}
	return binaryPath
}

// stdioSession is one `archcore mcp` process and its JSON-RPC stream.
type stdioSession struct {
	t      *testing.T
	stdin  io.WriteCloser
	lines  chan string
	nextID int
}

func startStdioSession(t *testing.T, cwd string, args ...string) *stdioSession {
	t.Helper()
	home := t.TempDir()
	cmd := exec.Command(archcoreBinary(t), append([]string{"mcp"}, args...)...)
	cmd.Dir = cwd
	cmd.Env = []string{
		"HOME=" + home,
		"XDG_STATE_HOME=" + filepath.Join(home, "state"),
		"XDG_CONFIG_HOME=" + filepath.Join(home, "config"),
		"PATH=" + os.Getenv("PATH"),
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	s := &stdioSession{t: t, stdin: stdin, lines: make(chan string, 64)}
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1<<20), 1<<22)
		for scanner.Scan() {
			s.lines <- scanner.Text()
		}
		close(s.lines)
	}()
	t.Cleanup(func() {
		_ = stdin.Close()
		done := make(chan struct{})
		go func() { _ = cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
		}
	})
	s.request("initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "e2e", "version": "0"},
	})
	s.send(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	return s
}

func (s *stdioSession) send(msg map[string]any) {
	data, err := json.Marshal(msg)
	if err != nil {
		s.t.Fatal(err)
	}
	if _, err := s.stdin.Write(append(data, '\n')); err != nil {
		s.t.Fatalf("write to archcore mcp: %v", err)
	}
}

func (s *stdioSession) request(method string, params map[string]any) map[string]any {
	s.t.Helper()
	s.nextID++
	id := s.nextID
	s.send(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	deadline := time.After(10 * time.Second)
	for {
		select {
		case line, ok := <-s.lines:
			if !ok {
				s.t.Fatalf("archcore mcp closed stdout before answering %s", method)
			}
			var msg map[string]any
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				s.t.Fatalf("stdout carried a non-JSON line %q: the stdio shield leaked", line)
			}
			if got, _ := msg["id"].(float64); int(got) == id {
				return msg
			}
		case <-deadline:
			s.t.Fatalf("no answer to %s", method)
		}
	}
}

// call invokes a tool and returns its text payload and whether it is an error.
func (s *stdioSession) call(name string, args map[string]any) (string, bool) {
	s.t.Helper()
	msg := s.request("tools/call", map[string]any{"name": name, "arguments": args})
	result, _ := msg["result"].(map[string]any)
	if result == nil {
		s.t.Fatalf("%s: no result in %v", name, msg)
	}
	isError, _ := result["isError"].(bool)
	content, _ := result["content"].([]any)
	if len(content) == 0 {
		s.t.Fatalf("%s: empty content", name)
	}
	text, _ := content[0].(map[string]any)["text"].(string)
	return text, isError
}

func (s *stdioSession) mustCall(name string, args map[string]any) map[string]any {
	s.t.Helper()
	text, isError := s.call(name, args)
	if isError {
		s.t.Fatalf("%s(%v) failed: %s", name, args, text)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		s.t.Fatalf("%s: %v\n%s", name, err, text)
	}
	return payload
}

func TestBinary_CallRoot(t *testing.T) {
	t.Parallel()
	w := newWorkspace(t)
	a := w.project(t, "a")
	b := w.project(t, "b")
	plain := w.plain(t, "plain")

	for _, tt := range []struct {
		name    string
		slug    string
		cwd     string
		args    []string
		session string // the root a call without project serves, "" when uninitialized
	}{
		{name: "started in project A", slug: "cwd-session", cwd: a, session: a},
		{name: "pinned to A from elsewhere", slug: "pinned-session", cwd: plain, args: []string{"--project", a}, session: a},
		{name: "started in a folder without .archcore", slug: "outside-session", cwd: plain},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := startStdioSession(t, tt.cwd, tt.args...)
			slug := tt.slug

			created := s.mustCall("create_document", adrArgs(slug, map[string]any{"project": b}))
			if created["project"] != b || created["path"] != ".archcore/"+slug+".adr.md" {
				t.Errorf("create_document = %v, want B and a relative path", created)
			}
			if !documentExists(t, b, ".archcore/"+slug+".adr.md") {
				t.Error("the document did not land in B")
			}

			if tt.session != "" {
				if got := s.mustCall("list_documents", nil)["project"]; got != tt.session {
					t.Errorf("a call without project served %v, want %q", got, tt.session)
				}
			} else {
				text, _ := s.call("list_documents", nil)
				if strings.Contains(text, `"project":"`+b) {
					t.Errorf("an uninitialized session answered from B: %s", text)
				}
			}

			text, isError := s.call("create_document", adrArgs("x", map[string]any{"project": "b"}))
			if !isError || !strings.Contains(text, "absolute path") || strings.Contains(text, w.dir) {
				t.Errorf("relative project: isError %v, text %q", isError, text)
			}
			text, isError = s.call("create_document", adrArgs("x", map[string]any{"project": plain}))
			if !isError || !strings.Contains(text, "init_project") || strings.Contains(text, w.dir) {
				t.Errorf("project without .archcore: isError %v, text %q", isError, text)
			}
			if _, err := os.Stat(filepath.Join(plain, ".archcore")); !os.IsNotExist(err) {
				t.Error("the session created .archcore/ in a folder without one")
			}
		})
	}
}
