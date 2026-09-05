// Package api is the HTTP client for the Archcore server. Every call is bounded
// by an explicit timeout, and the sync client carries its own longer one because
// a push is not a health check.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	archsync "archcore-cli/internal/sync"
)

const (
	// maxResponseSize bounds a decoded server response, so a server that answers
	// with an unbounded stream cannot exhaust the CLI's memory.
	maxResponseSize = 10 << 20
	// maxErrorBodyBytes bounds the excerpt quoted back to the user from a failed
	// response. It protects the terminal, not the heap.
	maxErrorBodyBytes = 512
	// healthTimeout bounds the readiness probe, which answers from memory on a
	// healthy server and must not hold a command open on an unhealthy one.
	healthTimeout = 10 * time.Second
	// syncPushTimeout bounds the whole push. The http.Client timeout is TOTAL —
	// it includes uploading the request body, and a first sync sends the full
	// content of every document in one POST, which the health budget cannot carry
	// on a slow uplink.
	syncPushTimeout = 120 * time.Second
)

// SyncAcceptedEntry represents a file that was successfully processed.
type SyncAcceptedEntry struct {
	Path   string `json:"path"`
	Action string `json:"action"`
}

// SyncErrorEntry represents a file that failed to process.
type SyncErrorEntry struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// SyncResponse is the server's response to POST /sync.
type SyncResponse struct {
	ProjectID int64               `json:"project_id"`
	Accepted  []SyncAcceptedEntry `json:"accepted"`
	Deleted   []string            `json:"deleted"`
	Errors    []SyncErrorEntry    `json:"errors"`
}

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func NewClient(serverURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(serverURL, "/") + "/api/v1",
		HTTPClient: &http.Client{
			Timeout: healthTimeout,
		},
	}
}

// NewSyncClient creates a client tuned for sync pushes.
func NewSyncClient(serverURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(serverURL, "/") + "/api/v1",
		HTTPClient: &http.Client{
			Timeout: syncPushTimeout,
		},
	}
}

func (c *Client) applyAuth(req *http.Request) {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}

// readErrorBody quotes the start of a failed response back to the user. It reads
// one byte past the cap so an oversized body is reported as truncated instead of
// arriving as a valid-looking prefix.
func readErrorBody(body io.Reader) string {
	b, _ := io.ReadAll(io.LimitReader(body, maxErrorBodyBytes+1))
	if len(b) > maxErrorBodyBytes {
		// The cut lands at a byte offset, so a non-ASCII body can end on half a
		// rune and reach the terminal as a replacement glyph.
		cut := strings.ToValidUTF8(string(b[:maxErrorBodyBytes]), "")
		return strings.TrimSpace(cut) + "… (truncated)"
	}
	return strings.TrimSpace(string(b))
}

func (c *Client) get(ctx context.Context, path string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	c.applyAuth(req)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		detail := readErrorBody(resp.Body)
		if detail != "" {
			return fmt.Errorf("server returned status %d: %s", resp.StatusCode, detail)
		}
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, maxResponseSize)
	if err := json.NewDecoder(limited).Decode(dest); err != nil {
		return fmt.Errorf("invalid response: %w", err)
	}
	return nil
}

func (c *Client) CheckHealth(ctx context.Context) error {
	var result struct {
		Ready bool `json:"ready"`
	}
	if err := c.get(ctx, "/status", &result); err != nil {
		return fmt.Errorf("checking server readiness: %w", err)
	}
	if !result.Ready {
		return errors.New("server is not ready")
	}
	return nil
}

// Sync pushes document changes to POST /sync.
// Returns the response, whether the project was auto-created (201), and any error.
func (c *Client) Sync(ctx context.Context, payload *archsync.Payload) (*SyncResponse, bool, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, false, fmt.Errorf("marshaling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/sync", bytes.NewReader(jsonData))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusMultiStatus:
		// All are success responses with a JSON body.
	default:
		detail := readErrorBody(resp.Body)
		if detail != "" {
			return nil, false, fmt.Errorf("sync request failed: server returned status %d: %s", resp.StatusCode, detail)
		}
		return nil, false, fmt.Errorf("sync request failed: server returned status %d", resp.StatusCode)
	}

	var syncResp SyncResponse
	limited := io.LimitReader(resp.Body, maxResponseSize)
	if err := json.NewDecoder(limited).Decode(&syncResp); err != nil {
		return nil, false, fmt.Errorf("invalid sync response: %w", err)
	}

	projectCreated := resp.StatusCode == http.StatusCreated
	return &syncResp, projectCreated, nil
}
