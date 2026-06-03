package blobsource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"
)

// DefaultGitHubBaseURL is the raw-content host for github.com.
const DefaultGitHubBaseURL = "https://raw.githubusercontent.com"

// maxResponseBytes caps the size of manifest.json and blob payloads we will
// read into memory. 10 MiB is well above any realistic payload and prevents
// memory exhaustion if a misconfigured BaseURL points at something that
// streams unbounded bytes.
const maxResponseBytes int64 = 10 * 1024 * 1024

// GitHubConfig configures a Source backed by a GitHub repository's
// manifest.json (e.g. OpenNSW/one-trade-templates).
type GitHubConfig struct {
	// Repo is "owner/name", e.g. "OpenNSW/one-trade-templates".
	Repo string
	// Ref is a branch name or commit SHA. Pin to a SHA in production for
	// reproducibility.
	Ref string
	// ManifestPath is the repo-relative path to the manifest file. Defaults to
	// "manifest.json" (at repo root). Blob paths inside the manifest's byId
	// map are ALWAYS repo-root-relative, regardless of where the manifest lives.
	ManifestPath string
	// RefreshInterval is how often to re-fetch the manifest in the background.
	// 0 disables background refresh.
	RefreshInterval time.Duration
	// BaseURL overrides the raw-content host. Defaults to DefaultGitHubBaseURL.
	// Set this when pointing at an httptest server, a self-hosted mirror, or
	// GitHub Enterprise.
	BaseURL string
	// HTTPClient overrides the HTTP client. Defaults to a client with a 10s
	// timeout.
	HTTPClient *http.Client
}

// manifestData mirrors the subset of manifest.json that this backend relies on.
// The manifest is the backend's own index format and is always parsed as JSON,
// even though blob payloads are treated as opaque bytes.
type manifestData struct {
	ByID map[string]string `json:"byId"`
}

// githubSource loads blobs from a GitHub repo by reading its manifest.json
// and fetching individual files on demand. The manifest is refreshed on a
// background ticker; blob bytes are cached in memory keyed by their manifest
// path so pushes that move a blob to a different path invalidate the cache
// for free.
type githubSource struct {
	repo         string
	ref          string
	baseURL      string
	manifestPath string // repo-relative path to the manifest file (default: "manifest.json")
	interval     time.Duration
	client       *http.Client

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu        sync.RWMutex
	byID      map[string]string // blobID -> repo-relative path
	blobCache map[string][]byte // repo-relative path -> blob bytes
}

// NewGitHub builds a Source that loads its manifest from a GitHub repo at
// startup (fail-fast on error) and refreshes it on a background ticker if
// RefreshInterval > 0. Blob files are fetched lazily on first Get and cached
// in memory.
func NewGitHub(ctx context.Context, cfg GitHubConfig) (Source, error) {
	if cfg.Repo == "" {
		return nil, fmt.Errorf("blobsource: GitHubConfig.Repo is required")
	}
	if cfg.Ref == "" {
		return nil, fmt.Errorf("blobsource: GitHubConfig.Ref is required")
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultGitHubBaseURL
	}
	if _, err := url.Parse(baseURL); err != nil {
		return nil, fmt.Errorf("blobsource: invalid BaseURL %q: %w", baseURL, err)
	}
	manifestPath := cfg.ManifestPath
	if manifestPath == "" {
		manifestPath = "manifest.json"
	}
	// Clean before validation so that bypass attempts like "fcau/.." are
	// normalised before the prefix check.
	manifestPath = path.Clean(manifestPath)
	if path.IsAbs(manifestPath) || manifestPath == ".." || strings.HasPrefix(manifestPath, "../") {
		return nil, fmt.Errorf("blobsource: ManifestPath %q must be a relative path that does not escape the repository root", cfg.ManifestPath)
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	srcCtx, cancel := context.WithCancel(context.Background())
	src := &githubSource{
		repo:         cfg.Repo,
		ref:          cfg.Ref,
		baseURL:      baseURL,
		manifestPath: manifestPath,
		interval:     cfg.RefreshInterval,
		client:       client,
		ctx:          srcCtx,
		cancel:       cancel,
		byID:         map[string]string{},
		blobCache:    map[string][]byte{},
	}
	if err := src.loadManifest(ctx); err != nil {
		return nil, fmt.Errorf("blobsource: failed to load manifest from %s: %w", src.manifestURL(), err)
	}
	slog.Info("github blob source initialized",
		"repo", src.repo, "ref", src.ref, "manifestEntries", len(src.byID))
	if src.interval > 0 {
		src.wg.Add(1)
		go src.refreshLoop()
	}
	return src, nil
}

func (s *githubSource) manifestURL() string {
	// BaseURL is validated at construction; JoinPath only errors on an
	// unparseable base, so the discarded error is unreachable here.
	u, _ := url.JoinPath(s.baseURL, s.repo, s.ref, s.manifestPath)
	return u
}

// blobURL resolves a manifest entry's byId value into a fully-qualified raw
// content URL. byId values are always repo-root-relative regardless of where
// the manifest file itself lives, so this is a direct join with no adjustment.
func (s *githubSource) blobURL(rel string) string {
	u, _ := url.JoinPath(s.baseURL, s.repo, s.ref, rel)
	return u
}

func (s *githubSource) loadManifest(ctx context.Context) error {
	url := s.manifestURL()
	body, err := s.fetch(ctx, url)
	if err != nil {
		return err
	}
	var m manifestData
	if err := json.Unmarshal(body, &m); err != nil {
		return fmt.Errorf("failed to parse manifest at %s: %w", url, err)
	}
	if m.ByID == nil {
		return fmt.Errorf("manifest at %s has no byId field", url)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear the entire blob cache on every refresh. When Ref is a branch name a
	// blob file can be updated in-place (same manifest path, new bytes), so
	// selective path-based eviction would keep serving stale content. A full
	// clear ensures the next Get always fetches the latest bytes. The trade-off
	// (one extra fetch per cached blob after each refresh) is acceptable given
	// typical refresh intervals.
	s.blobCache = make(map[string][]byte)
	s.byID = m.ByID
	return nil
}

func (s *githubSource) refreshLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.refresh()
		}
	}
}

func (s *githubSource) refresh() {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()
	if err := s.loadManifest(ctx); err != nil {
		slog.Warn("blobsource: github manifest refresh failed", "error", err)
	}
}

func (s *githubSource) Get(ctx context.Context, id string) ([]byte, bool, error) {
	s.mu.RLock()
	path, known := s.byID[id]
	if !known {
		s.mu.RUnlock()
		return nil, false, nil
	}
	if cached, hit := s.blobCache[path]; hit {
		s.mu.RUnlock()
		return cached, true, nil
	}
	s.mu.RUnlock()

	body, err := s.fetch(ctx, s.blobURL(path))
	if err != nil {
		return nil, false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if cached, hit := s.blobCache[path]; hit {
		return cached, true, nil
	}
	// Only cache if the manifest still maps this id to this path. A concurrent
	// refresh could have moved the blob elsewhere while we were fetching.
	if curPath, stillKnown := s.byID[id]; stillKnown && curPath == path {
		s.blobCache[path] = body
	}
	return body, true, nil
}

// Close stops the background refresh goroutine and cancels any in-flight
// manifest fetch. Blocks until the background goroutine has exited.
// Safe to call multiple times.
func (s *githubSource) Close() error {
	s.cancel()
	s.wg.Wait()
	return nil
}

func (s *githubSource) fetch(ctx context.Context, requestURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %s: %w", requestURL, err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", requestURL, err)
	}
	// Drain before closing so the underlying TCP/TLS connection can be reused.
	defer func() { _, _ = io.Copy(io.Discard, resp.Body); _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s: unexpected status %d", requestURL, resp.StatusCode)
	}
	// Read one byte past the limit so we can distinguish "exactly at limit" from
	// "over limit" — io.LimitReader alone would silently truncate.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("GET %s: read error: %w", requestURL, err)
	}
	if int64(len(body)) > maxResponseBytes {
		return nil, fmt.Errorf("GET %s: response exceeds %d bytes", requestURL, maxResponseBytes)
	}
	return body, nil
}
