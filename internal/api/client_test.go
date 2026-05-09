package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	archsync "archcore-cli/internal/sync"
)

func TestNewClient(t *testing.T) {
	c := NewClient("http://example.com")
	want := "http://example.com/api/v1"
	if c.BaseURL != want {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, want)
	}
}

func TestCheckHealth(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			name: "healthy",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"ready":true}`))
			},
			wantErr: false,
		},
		{
			name: "not ready",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"ready":false}`))
			},
			wantErr: true,
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name: "bad JSON",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`not json`))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			c := NewClient(srv.URL)
			err := c.CheckHealth(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckHealth error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckHealth_ConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close() // close immediately so connection is refused

	c := NewClient(srv.URL)
	if err := c.CheckHealth(context.Background()); err == nil {
		t.Fatal("expected error for closed server")
	}
}

func TestCheckHealth_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ready":true}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before request

	c := NewClient(srv.URL)
	if err := c.CheckHealth(ctx); err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestListProjects(t *testing.T) {
	tests := []struct {
		name      string
		handler   http.HandlerFunc
		wantErr   bool
		wantCount int
	}{
		{
			name: "success with projects",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/projects" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				if r.Method != http.MethodGet {
					t.Errorf("unexpected method: %s", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`[{"id":1,"name":"project-a"},{"id":2,"name":"project-b"}]`))
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "empty list",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`[]`))
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name: "bad JSON",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`not json`))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			c := NewClient(srv.URL)
			projects, err := c.ListProjects(context.Background())
			if (err != nil) != tt.wantErr {
				t.Fatalf("ListProjects error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(projects) != tt.wantCount {
				t.Errorf("got %d projects, want %d", len(projects), tt.wantCount)
			}
		})
	}
}

func TestListProjects_ValidatesFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":42,"name":"my-project"}]`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	projects, err := c.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("got %d projects, want 1", len(projects))
	}
	if projects[0].ID != 42 {
		t.Errorf("ID = %d, want 42", projects[0].ID)
	}
	if projects[0].Name != "my-project" {
		t.Errorf("Name = %q, want %q", projects[0].Name, "my-project")
	}
}

func TestGetProject(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/projects/42" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"id":42,"name":"my-project"}`))
			},
			wantErr: false,
		},
		{
			name: "not found",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: true,
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name: "bad JSON",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`not json`))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			c := NewClient(srv.URL)
			project, err := c.GetProject(context.Background(), 42)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetProject error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if project.ID != 42 {
					t.Errorf("ID = %d, want 42", project.ID)
				}
				if project.Name != "my-project" {
					t.Errorf("Name = %q, want %q", project.Name, "my-project")
				}
			}
		})
	}
}

func TestGetProject_ConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	c := NewClient(srv.URL)
	_, err := c.GetProject(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for closed server")
	}
}

func TestListProjects_ConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	c := NewClient(srv.URL)
	_, err := c.ListProjects(context.Background())
	if err == nil {
		t.Fatal("expected error for closed server")
	}
}

func TestNewAuthenticatedClient(t *testing.T) {
	c := NewAuthenticatedClient("http://example.com", "my-token")
	if c.BaseURL != "http://example.com/api/v1" {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, "http://example.com/api/v1")
	}
	if c.Token != "my-token" {
		t.Errorf("Token = %q, want %q", c.Token, "my-token")
	}
}

func TestApplyAuth_NoToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("unexpected Authorization header: %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ready":true}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	c.CheckHealth(context.Background())
}

func TestSync(t *testing.T) {
	tests := []struct {
		name          string
		handler       http.HandlerFunc
		wantErr       bool
		wantCreated   bool
		wantProjectID int64
		wantAccepted  int
	}{
		{
			name: "200 OK - existing project",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("method = %s, want POST", r.Method)
				}
				if r.URL.Path != "/api/v1/sync" {
					t.Errorf("path = %s, want /api/v1/sync", r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Errorf("auth = %q, want 'Bearer test-token'", r.Header.Get("Authorization"))
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("content-type = %q, want application/json", r.Header.Get("Content-Type"))
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(SyncResponse{
					ProjectID: 42,
					Accepted:  []SyncAcceptedEntry{{Path: "vision/test.md", Action: "created"}},
					Deleted:   []string{},
					Errors:    []SyncErrorEntry{},
				})
			},
			wantErr:       false,
			wantCreated:   false,
			wantProjectID: 42,
			wantAccepted:  1,
		},
		{
			name: "201 Created - auto-create project",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(SyncResponse{
					ProjectID: 99,
					Accepted:  []SyncAcceptedEntry{{Path: "vision/plan.md", Action: "created"}},
					Deleted:   []string{},
					Errors:    []SyncErrorEntry{},
				})
			},
			wantErr:       false,
			wantCreated:   true,
			wantProjectID: 99,
			wantAccepted:  1,
		},
		{
			name: "207 Multi-Status - partial success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusMultiStatus)
				json.NewEncoder(w).Encode(SyncResponse{
					ProjectID: 42,
					Accepted:  []SyncAcceptedEntry{{Path: "vision/ok.md", Action: "created"}},
					Deleted:   []string{},
					Errors:    []SyncErrorEntry{{Path: "vision/bad.md", Message: "invalid"}},
				})
			},
			wantErr:       false,
			wantCreated:   false,
			wantProjectID: 42,
			wantAccepted:  1,
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name: "unauthorized",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("invalid token"))
			},
			wantErr: true,
		},
		{
			name: "bad response JSON",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`not json`))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			c := NewAuthenticatedClient(srv.URL, "test-token")
			payload := &archsync.Payload{
				Created: []archsync.FileEntry{
					{Path: "vision/test.md", SHA256: "abc", Content: "# Test"},
				},
				Modified: []archsync.FileEntry{},
				Deleted:  []string{},
			}
			resp, created, err := c.Sync(context.Background(), payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Sync error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if created != tt.wantCreated {
				t.Errorf("created = %v, want %v", created, tt.wantCreated)
			}
			if resp.ProjectID != tt.wantProjectID {
				t.Errorf("ProjectID = %d, want %d", resp.ProjectID, tt.wantProjectID)
			}
			if len(resp.Accepted) != tt.wantAccepted {
				t.Errorf("Accepted count = %d, want %d", len(resp.Accepted), tt.wantAccepted)
			}
		})
	}
}

func TestSync_SendsPayloadFields(t *testing.T) {
	pid := 42
	projectName := "my-project"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload archsync.Payload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}

		if payload.ProjectID == nil || *payload.ProjectID != pid {
			t.Errorf("ProjectID = %v, want %d", payload.ProjectID, pid)
		}
		if payload.ProjectName == nil || *payload.ProjectName != projectName {
			t.Errorf("ProjectName = %v, want %q", payload.ProjectName, projectName)
		}
		if len(payload.Created) != 1 {
			t.Errorf("Created count = %d, want 1", len(payload.Created))
		}
		if len(payload.Deleted) != 1 {
			t.Errorf("Deleted count = %d, want 1", len(payload.Deleted))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SyncResponse{ProjectID: 42})
	}))
	defer srv.Close()

	c := NewAuthenticatedClient(srv.URL, "token")
	payload := &archsync.Payload{
		ProjectID:   &pid,
		ProjectName: &projectName,
		Created: []archsync.FileEntry{
			{Path: "vision/test.md", SHA256: "abc", Content: "# Test"},
		},
		Modified: []archsync.FileEntry{},
		Deleted:  []string{"vision/old.md"},
	}
	_, _, err := c.Sync(context.Background(), payload)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
}

func TestSync_ConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	c := NewAuthenticatedClient(srv.URL, "token")
	_, _, err := c.Sync(context.Background(), &archsync.Payload{
		Created:  []archsync.FileEntry{},
		Modified: []archsync.FileEntry{},
		Deleted:  []string{},
	})
	if err == nil {
		t.Fatal("expected error for closed server")
	}
}
