package templatesource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// DefaultGitHubBaseURL is the raw-content host for github.com.
const DefaultGitHubBaseURL = "https://raw.githubusercontent.com"

// maxResponseBytes caps the size of manifest.json and template payloads we
// will read into memory. 10 MiB is well above any real JSON template and
// prevents memory exhaustion if a misconfigured BaseURL points at something
// that streams unbounded bytes.
const maxResponseBytes int64 = 10 * 1024 * 1024

// GitHubConfig configures a Source backed by a GitHub repository's
// manifest.json (e.g. OpenNSW/one-trade-templates).
type GitHubConfig struct {
	// Repo is "owner/name", e.g. "OpenNSW/one-trade-templates".
	Repo string
	// Ref is a branch name or commit SHA. Pin to a SHA in production for
	// reproducibility.
	Ref string
	// RefreshInterval is how often to re-fetch manifest.json in the background.
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

// manifestData mirrors the subset of one-trade-templates/manifest.json that we
// rely on. The full manifest also includes workflows/version/generated fields
// which we ignore.
type manifestData struct {
	ByID map[string]string `json:"byId"`
}

// githubSource loads templates from a GitHub repo by reading its manifest.json
// and fetching individual template files on demand. The manifest is refreshed
// on a background ticker; template bytes are cached in memory keyed by their
// manifest path so pushes that move a template to a different path invalidate
// the cache for free.
type githubSource struct {
	repo     string
	ref      string
	baseURL  string
	interval time.Duration
	client   *http.Client

	mu            sync.RWMutex
	byID          map[string]string          // templateID -> repo-relative path
	templateCache map[string]json.RawMessage // path -> template bytes

	done      chan struct{}
	closeOnce sync.Once
}

// NewGitHub builds a Source that loads its manifest from a GitHub repo at
// startup (fail-fast on error) and refreshes it on a background ticker if
// RefreshInterval > 0. Template files are fetched lazily on first GetTemplate
// and cached in memory.
func NewGitHub(ctx context.Context, cfg GitHubConfig) (Source, error) {
	if cfg.Repo == "" {
		return nil, fmt.Errorf("templatesource: GitHubConfig.Repo is required")
	}
	if cfg.Ref == "" {
		return nil, fmt.Errorf("templatesource: GitHubConfig.Ref is required")
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultGitHubBaseURL
	}
	if _, err := url.Parse(baseURL); err != nil {
		return nil, fmt.Errorf("templatesource: invalid BaseURL %q: %w", baseURL, err)
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	src := &githubSource{
		repo:          cfg.Repo,
		ref:           cfg.Ref,
		baseURL:       baseURL,
		interval:      cfg.RefreshInterval,
		client:        client,
		byID:          map[string]string{},
		templateCache: map[string]json.RawMessage{},
		done:          make(chan struct{}),
	}
	if err := src.loadManifest(ctx); err != nil {
		return nil, fmt.Errorf("templatesource: failed to load manifest from %s: %w", src.manifestURL(), err)
	}
	slog.Info("github template source initialized",
		"repo", src.repo, "ref", src.ref, "manifestEntries", len(src.byID))
	if src.interval > 0 {
		go src.refreshLoop()
	}
	return src, nil
}

func (s *githubSource) manifestURL() string {
	// BaseURL is validated at construction; JoinPath only errors on an
	// unparseable base, so the discarded error is unreachable here.
	u, _ := url.JoinPath(s.baseURL, s.repo, s.ref, "manifest.json")
	return u
}

func (s *githubSource) templateURL(path string) string {
	u, _ := url.JoinPath(s.baseURL, s.repo, s.ref, path)
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

	// Invalidate cache entries whose path is no longer referenced by the new
	// manifest. Path equality across refreshes is enough — if the same path
	// still appears in byId, its bytes are still correct.
	newPaths := make(map[string]struct{}, len(m.ByID))
	for _, p := range m.ByID {
		newPaths[p] = struct{}{}
	}
	for path := range s.templateCache {
		if _, ok := newPaths[path]; !ok {
			delete(s.templateCache, path)
		}
	}
	s.byID = m.ByID
	return nil
}

func (s *githubSource) refreshLoop() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := s.loadManifest(ctx); err != nil {
				slog.Warn("templatesource: github manifest refresh failed", "error", err)
			}
			cancel()
		}
	}
}

func (s *githubSource) GetTemplate(ctx context.Context, id string) (json.RawMessage, bool, error) {
	s.mu.RLock()
	path, known := s.byID[id]
	if !known {
		s.mu.RUnlock()
		return nil, false, nil
	}
	if cached, hit := s.templateCache[path]; hit {
		s.mu.RUnlock()
		return cached, true, nil
	}
	s.mu.RUnlock()

	body, err := s.fetch(ctx, s.templateURL(path))
	if err != nil {
		return nil, false, err
	}
	if !json.Valid(body) {
		return nil, false, fmt.Errorf("templatesource: template %q at %s is not valid JSON", id, path)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if cached, hit := s.templateCache[path]; hit {
		return cached, true, nil
	}
	// Only cache if the manifest still maps this id to this path. A concurrent
	// refresh could have moved the template elsewhere while we were fetching.
	if curPath, stillKnown := s.byID[id]; stillKnown && curPath == path {
		s.templateCache[path] = body
	}
	return body, true, nil
}

func (s *githubSource) Close() error {
	s.closeOnce.Do(func() { close(s.done) })
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
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s: unexpected status %d", requestURL, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
}
