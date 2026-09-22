package mcp

import (
	"testing"
)

// install_host_config is registered only when the cmd layer injects an
// executor via WithHostWiring — a bare NewServer (tests, embedders) must not
// expose a tool that writes host config outside .archcore/.
func TestNewServer_HostWiringToolRegistration(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	bare := NewServer(base, "v0.0.0-test")
	if bare.GetTool("install_host_config") != nil {
		t.Error("bare NewServer must not register install_host_config")
	}

	wired := NewServer(base, "v0.0.0-test",
		WithHostWiring(func(baseDir, host string, allDetected bool) ([]byte, error) {
			return []byte(`{}`), nil
		}))
	if wired.GetTool("install_host_config") == nil {
		t.Error("WithHostWiring must register install_host_config")
	}
}
