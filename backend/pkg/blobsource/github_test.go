package blobsource

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// stubBlobServer serves a manifest + arbitrary blob files. Routes match the
// raw.githubusercontent.com layout: /<owner>/<repo>/<ref>/<path>.
type stubBlobServer struct {
	t        *testing.T
	mu       sync.Mutex
	manifest atomic.Value // []byte
	files    map[string][]byte
	fetches  atomic.Int64 // counts manifest GETs
}

func newStubBlobServer(t *testing.T) (*stubBlobServer, *httptest.Server) {
	t.Helper()
	s := &stubBlobServer{t: t, files: make(map[string][]byte)}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		// path looks like "<owner>/<repo>/<ref>/<rest>"
		// strip the first three segments to get the in-repo path.
		parts := strings.SplitN(path, "/", 4)
		if len(parts) < 4 {
			http.NotFound(w, r)
			return
		}
		repoRelative := parts[3]

		if repoRelative == "manifest.json" {
			s.fetches.Add(1)
			body, _ := s.manifest.Load().([]byte)
			if body == nil {
				http.NotFound(w, r)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
			return
		}

		s.mu.Lock()
		body, ok := s.files[repoRelative]
		s.mu.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return s, srv
}

// setManifest installs a manifest whose byId is the given map.
func (s *stubBlobServer) setManifest(byID map[string]string) {
	body, err := json.Marshal(map[string]any{"byId": byID})
	if err != nil {
		s.t.Fatalf("marshal manifest: %v", err)
	}
	s.manifest.Store(body)
}

// setRawManifest installs an arbitrary body (for parse-error tests).
func (s *stubBlobServer) setRawManifest(body []byte) {
	s.manifest.Store(body)
}

func (s *stubBlobServer) setFile(path, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[path] = []byte(body)
}

func (s *stubBlobServer) deleteFile(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.files, path)
}

func newTestGitHubSource(t *testing.T, srvURL string, interval time.Duration) Source {
	t.Helper()
	src, err := NewGitHub(context.Background(), GitHubConfig{
		Repo:            "owner/repo",
		Ref:             "main",
		BaseURL:         srvURL,
		RefreshInterval: interval,
	})
	if err != nil {
		t.Fatalf("NewGitHub: %v", err)
	}
	t.Cleanup(func() { _ = src.Close() })
	return src
}

func TestGitHub_HappyPath(t *testing.T) {
	stub, srv := newStubBlobServer(t)
	stub.setManifest(map[string]string{"alpha": "blobs/a.json"})
	stub.setFile("blobs/a.json", `{"schema":{"type":"object"}}`)

	src := newTestGitHubSource(t, srv.URL, 0)

	body, ok, err := src.Get(context.Background(), "alpha")
	if err != nil || !ok {
		t.Fatalf("expected alpha (ok=%v, err=%v)", ok, err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("returned blob is not valid JSON: %v", err)
	}
	if _, ok := got["schema"]; !ok {
		t.Errorf("expected schema field, got %v", got)
	}

	// Second call should hit the in-memory cache (no new file fetch); easiest
	// check is that it succeeds even after we delete the upstream file.
	stub.deleteFile("blobs/a.json")
	if _, ok, err := src.Get(context.Background(), "alpha"); err != nil || !ok {
		t.Fatalf("expected cached alpha (ok=%v, err=%v)", ok, err)
	}
}

func TestGitHub_UnknownIDReturnsNotFound(t *testing.T) {
	stub, srv := newStubBlobServer(t)
	stub.setManifest(map[string]string{"alpha": "blobs/a.json"})

	src := newTestGitHubSource(t, srv.URL, 0)

	body, ok, err := src.Get(context.Background(), "missing")
	if err != nil {
		t.Fatalf("expected nil error for unknown ID, got %v", err)
	}
	if ok || body != nil {
		t.Fatalf("expected (nil, false, nil) for unknown ID, got (%v, %v, %v)", body, ok, err)
	}
}

func TestGitHub_ManifestMissingFailsFast(t *testing.T) {
	// Manifest not installed -> handler returns 404 on the first fetch.
	_, srv := newStubBlobServer(t)

	_, err := NewGitHub(context.Background(), GitHubConfig{
		Repo:    "owner/repo",
		Ref:     "main",
		BaseURL: srv.URL,
	})
	if err == nil {
		t.Fatalf("expected ctor error when manifest is missing, got nil")
	}
}

func TestGitHub_ManifestInvalidJSONFailsFast(t *testing.T) {
	stub, srv := newStubBlobServer(t)
	stub.setRawManifest([]byte(`{not json`))

	_, err := NewGitHub(context.Background(), GitHubConfig{
		Repo:    "owner/repo",
		Ref:     "main",
		BaseURL: srv.URL,
	})
	if err == nil {
		t.Fatalf("expected ctor error on invalid manifest JSON, got nil")
	}
}

func TestGitHub_BlobFetchErrorPropagates(t *testing.T) {
	stub, srv := newStubBlobServer(t)
	stub.setManifest(map[string]string{"alpha": "blobs/a.json"})
	// Intentionally do not install blobs/a.json.

	src := newTestGitHubSource(t, srv.URL, 0)

	_, _, err := src.Get(context.Background(), "alpha")
	if err == nil {
		t.Fatalf("expected fetch error for missing blob file, got nil")
	}
}

// TestGitHub_NonJSONBlobIsServedVerbatim documents that the package no longer
// validates blob payloads — any bytes the GitHub endpoint returns are passed
// through to the caller as-is.
func TestGitHub_NonJSONBlobIsServedVerbatim(t *testing.T) {
	stub, srv := newStubBlobServer(t)
	stub.setManifest(map[string]string{"alpha": "blobs/a.json"})
	stub.setFile("blobs/a.json", `not json`)

	src := newTestGitHubSource(t, srv.URL, 0)

	body, ok, err := src.Get(context.Background(), "alpha")
	if err != nil || !ok {
		t.Fatalf("expected blob to be served verbatim (ok=%v, err=%v)", ok, err)
	}
	if string(body) != "not json" {
		t.Errorf("expected raw bytes 'not json', got %q", body)
	}
}

func TestGitHub_ManifestRefreshInvalidatesCache(t *testing.T) {
	stub, srv := newStubBlobServer(t)
	stub.setManifest(map[string]string{"alpha": "blobs/a.json"})
	stub.setFile("blobs/a.json", `{"v":1}`)
	stub.setFile("blobs/a2.json", `{"v":2}`)

	src := newTestGitHubSource(t, srv.URL, 0)

	if _, ok, err := src.Get(context.Background(), "alpha"); err != nil || !ok {
		t.Fatalf("expected initial alpha (ok=%v, err=%v)", ok, err)
	}

	// Repoint alpha to a different path; manually reload manifest via type assertion.
	stub.setManifest(map[string]string{"alpha": "blobs/a2.json"})
	gs, ok := src.(*githubSource)
	if !ok {
		t.Fatalf("expected *githubSource, got %T", src)
	}
	if err := gs.loadManifest(context.Background()); err != nil {
		t.Fatalf("manifest reload: %v", err)
	}

	body, ok, err := src.Get(context.Background(), "alpha")
	if err != nil || !ok {
		t.Fatalf("expected refreshed alpha (ok=%v, err=%v)", ok, err)
	}
	if got := string(body); got != `{"v":2}` {
		t.Errorf("expected new bytes after manifest swap, got %s", got)
	}
}

// TestGitHub_ManifestRefreshClearsStaleContent verifies that updating a blob
// file in-place (same manifest path, new bytes) is reflected after a manifest
// refresh. The old selective-eviction logic would have served stale content
// in this scenario.
func TestGitHub_ManifestRefreshClearsStaleContent(t *testing.T) {
	stub, srv := newStubBlobServer(t)
	stub.setManifest(map[string]string{"alpha": "blobs/a.json"})
	stub.setFile("blobs/a.json", `{"v":1}`)

	src := newTestGitHubSource(t, srv.URL, 0)

	if _, ok, err := src.Get(context.Background(), "alpha"); err != nil || !ok {
		t.Fatalf("expected initial alpha (ok=%v, err=%v)", ok, err)
	}

	// Update file content in-place; manifest path is unchanged.
	stub.setFile("blobs/a.json", `{"v":2}`)
	gs, ok := src.(*githubSource)
	if !ok {
		t.Fatalf("expected *githubSource, got %T", src)
	}
	if err := gs.loadManifest(context.Background()); err != nil {
		t.Fatalf("manifest reload: %v", err)
	}

	body, ok, err := src.Get(context.Background(), "alpha")
	if err != nil || !ok {
		t.Fatalf("expected refreshed alpha (ok=%v, err=%v)", ok, err)
	}
	if got := string(body); got != `{"v":2}` {
		t.Errorf("expected updated content after manifest refresh, got %s", got)
	}
}

func TestGitHub_BackgroundRefreshTicks(t *testing.T) {
	stub, srv := newStubBlobServer(t)
	stub.setManifest(map[string]string{"alpha": "blobs/a.json"})

	src := newTestGitHubSource(t, srv.URL, 30*time.Millisecond)

	// First fetch happened in ctor (count == 1). Wait for at least one tick.
	deadline := time.Now().Add(2 * time.Second)
	for stub.fetches.Load() < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("manifest never re-fetched: fetches=%d", stub.fetches.Load())
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := src.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestGitHub_RequiresRepoAndRef(t *testing.T) {
	_, err := NewGitHub(context.Background(), GitHubConfig{Ref: "main"})
	if err == nil {
		t.Fatalf("expected error for missing Repo")
	}
	_, err = NewGitHub(context.Background(), GitHubConfig{Repo: "owner/repo"})
	if err == nil {
		t.Fatalf("expected error for missing Ref")
	}
}

// roundTripFunc lets a plain function satisfy http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGitHub_InvalidBaseURLReturnsError(t *testing.T) {
	_, err := NewGitHub(context.Background(), GitHubConfig{
		Repo:    "owner/repo",
		Ref:     "main",
		BaseURL: "://invalid", // empty scheme — url.Parse rejects this
	})
	if err == nil {
		t.Fatal("expected error for invalid BaseURL, got nil")
	}
}

func TestGitHub_DefaultBaseURLFallback(t *testing.T) {
	// Failing transport prevents real network calls while still exercising the
	// empty-BaseURL → DefaultGitHubBaseURL assignment and the transport-error path.
	failClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		}),
	}
	_, err := NewGitHub(context.Background(), GitHubConfig{
		Repo:       "owner/repo",
		Ref:        "main",
		HTTPClient: failClient,
		// BaseURL intentionally omitted to exercise the default-assignment branch.
	})
	if err == nil {
		t.Fatal("expected error when manifest fetch fails, got nil")
	}
}

func TestGitHub_ManifestMissingByIDField(t *testing.T) {
	stub, srv := newStubBlobServer(t)
	stub.setRawManifest([]byte(`{"workflows":{}}`)) // valid JSON but no byId key

	_, err := NewGitHub(context.Background(), GitHubConfig{
		Repo:    "owner/repo",
		Ref:     "main",
		BaseURL: srv.URL,
	})
	if err == nil {
		t.Fatal("expected error when manifest lacks byId field, got nil")
	}
}

func TestGitHub_BackgroundRefreshLogsOnError(t *testing.T) {
	stub, srv := newStubBlobServer(t)
	stub.setManifest(map[string]string{})

	src := newTestGitHubSource(t, srv.URL, 30*time.Millisecond)

	// Replace the manifest with invalid JSON so the next background tick fails
	// and exercises the slog.Warn branch inside refresh().
	stub.setRawManifest([]byte("{not json"))

	deadline := time.Now().Add(2 * time.Second)
	for stub.fetches.Load() < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("background refresh never ticked: fetches=%d", stub.fetches.Load())
		}
		time.Sleep(10 * time.Millisecond)
	}
	_ = src
}

func TestGitHub_CustomManifestPathResolvesRelativeBlobs(t *testing.T) {
	stub, srv := newStubBlobServer(t)
	// Manifest lives at agency-configs/fcau/manifest.json. byId values are
	// repo-root-relative, so "agency-configs/fcau/task-configs/alpha.json"
	// fetches from that exact path at the repo root.
	stub.setFile("agency-configs/fcau/manifest.json",
		`{"byId":{"alpha":"agency-configs/fcau/task-configs/alpha.json"}}`)
	stub.setFile("agency-configs/fcau/task-configs/alpha.json", `{"v":1}`)

	src, err := NewGitHub(context.Background(), GitHubConfig{
		Repo:         "owner/repo",
		Ref:          "main",
		ManifestPath: "agency-configs/fcau/manifest.json",
		BaseURL:      srv.URL,
	})
	if err != nil {
		t.Fatalf("NewGitHub: %v", err)
	}
	t.Cleanup(func() { _ = src.Close() })

	body, ok, err := src.Get(context.Background(), "alpha")
	if err != nil || !ok {
		t.Fatalf("expected alpha (ok=%v, err=%v)", ok, err)
	}
	if string(body) != `{"v":1}` {
		t.Errorf("expected v:1 from nested path, got %s", body)
	}
}

func TestGitHub_RejectsManifestPathTraversal(t *testing.T) {
	_, srv := newStubBlobServer(t)
	for _, badPath := range []string{
		"../manifest.json",            // raw traversal
		"/etc/manifest.json",          // absolute path
		"foo/../../bar/manifest.json", // traversal via clean → ../bar/manifest.json
		"..",                          // exactly ".."
	} {
		_, err := NewGitHub(context.Background(), GitHubConfig{
			Repo:         "owner/repo",
			Ref:          "main",
			ManifestPath: badPath,
			BaseURL:      srv.URL,
		})
		if err == nil {
			t.Errorf("expected error for ManifestPath %q, got nil", badPath)
		}
	}
}

// Close is idempotent.
func TestGitHub_CloseIsIdempotent(t *testing.T) {
	stub, srv := newStubBlobServer(t)
	stub.setManifest(map[string]string{})

	src := newTestGitHubSource(t, srv.URL, 0)
	if err := src.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := src.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
