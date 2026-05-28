package blobsource

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr string // substring; empty means expect nil
	}{
		{
			name: "local ok",
			cfg:  Config{Type: "local", LocalDir: "/some/dir"},
		},
		{
			name:    "local missing dir",
			cfg:     Config{Type: "local"},
			wantErr: "BLOBSOURCE_LOCAL_DIR",
		},
		{
			name: "github ok",
			cfg:  Config{Type: "github", GitHubRepo: "o/r", GitHubRef: "main"},
		},
		{
			name:    "github missing repo",
			cfg:     Config{Type: "github", GitHubRef: "main"},
			wantErr: "BLOBSOURCE_GITHUB_REPO",
		},
		{
			name:    "github missing ref",
			cfg:     Config{Type: "github", GitHubRepo: "o/r"},
			wantErr: "BLOBSOURCE_GITHUB_REF",
		},
		{
			name:    "github invalid base url",
			cfg:     Config{Type: "github", GitHubRepo: "o/r", GitHubRef: "main", GitHubBaseURL: "not-a-url"},
			wantErr: "BLOBSOURCE_GITHUB_BASE_URL",
		},
		{
			name: "github valid base url",
			cfg:  Config{Type: "github", GitHubRepo: "o/r", GitHubRef: "main", GitHubBaseURL: "https://example.com"},
		},
		{
			name:    "unsupported type",
			cfg:     Config{Type: "ftp"},
			wantErr: "unsupported BLOBSOURCE_TYPE",
		},
		{
			name:    "empty type",
			cfg:     Config{},
			wantErr: "unsupported BLOBSOURCE_TYPE",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestNewFromConfig_Local(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "alpha.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	src, err := NewFromConfig(context.Background(), Config{Type: "local", LocalDir: dir})
	if err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}
	defer func() { _ = src.Close() }()

	if _, ok, err := src.Get(context.Background(), "alpha"); err != nil || !ok {
		t.Fatalf("expected alpha to be served (ok=%v, err=%v)", ok, err)
	}
}

func TestNewFromConfig_GitHub(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/manifest.json") {
			_, _ = w.Write([]byte(`{"byId":{}}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	src, err := NewFromConfig(context.Background(), Config{
		Type:       "github",
		GitHubRepo: "owner/repo",
		GitHubRef:  "main",
		// Pass the test server URL as the BaseURL so no real network call is made.
		GitHubBaseURL: srv.URL,
	})
	if err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}
	defer func() { _ = src.Close() }()
}

func TestNewFromConfig_UnsupportedType(t *testing.T) {
	_, err := NewFromConfig(context.Background(), Config{Type: "ftp"})
	if err == nil {
		t.Fatal("expected error for unsupported type, got nil")
	}
}
