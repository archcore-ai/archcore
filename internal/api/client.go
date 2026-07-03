package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	archsync "archcore-cli/internal/sync"
)

const maxResponseSize = 10 << 20 // 10 MB

type Project struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

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
			Timeout: 10 * time.Second,
		},
	}
}

// NewSyncClient creates a client tuned for sync pushes. The http.Client
// timeout is TOTAL (it includes uploading the request body), and a first sync
// sends the full content of every document in one POST — the default 10s
// client can time out mid-upload on a slow uplink and never succeed.
func NewSyncClient(serverURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(serverURL, "/") + "/api/v1",
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *Client) applyAuth(req *http.Request) {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}

// readErrorBody reads up to 512 bytes from the response body for error context.
func readErrorBody(body io.Reader) string {
	b := make([]byte, 512)
	n, _ := io.ReadAtLeast(body, b, 1)
	if n == 0 {
		return ""
	}
	return strings.TrimSpace(string(b[:n]))
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
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
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
		return err
	}
	if !result.Ready {
		return fmt.Errorf("server is not ready")
	}
	return nil
}

func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	var projects []Project
	if err := c.get(ctx, "/projects", &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (c *Client) GetProject(ctx context.Context, id int64) (*Project, error) {
	var project Project
	if err := c.get(ctx, fmt.Sprintf("/projects/%d", id), &project); err != nil {
		return nil, err
	}
	return &project, nil
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
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
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
