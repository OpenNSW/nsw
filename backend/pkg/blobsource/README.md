# blobsource

Resolves opaque byte blobs by ID. The package exposes a single `Source`
interface with two implementations:

| Implementation | Constructor | When to use |
|---|---|---|
| Local filesystem | `NewLocal(dir)` | Development / offline testing |
| GitHub raw content | `NewGitHub(ctx, cfg)` | Staging and production |

Payloads are treated as opaque bytes — the package never inspects their
structure. Interpretation (JSON parsing, schema validation, template rendering,
etc.) is the caller's job.

For env-driven setup, callers should construct a `Config` and use the
`NewFromConfig` factory; both backends share the same configuration entry
point.

## Import

```go
import "github.com/OpenNSW/nsw/backend/pkg/blobsource"
```

## Configuration

`Config` selects the backend via `Type` (`"local"` or `"github"`) and carries
the per-backend fields. It is composed into the central
`internal/config/config.go` and populated from `BLOBSOURCE_*` env vars.

```go
type Config struct {
    Type string // "local" or "github"

    // local backend
    LocalDir string

    // github backend
    GitHubRepo            string
    GitHubRef             string
    GitHubManifestPath    string        // optional, defaults to "manifest.json"
    GitHubBaseURL         string        // optional
    GitHubRefreshInterval time.Duration // optional, 0 disables
}
```

`Config.Validate()` enforces required fields per type and is called by the
central config's `Validate()`.

```go
src, err := blobsource.NewFromConfig(ctx, cfg.BlobSource)
if err != nil { /* ... */ }
defer src.Close()

raw, ok, err := src.Get(ctx, "build-licence")
```

### Env vars

| Env var | Default | Notes |
|---|---|---|
| `BLOBSOURCE_TYPE` | `local` | `"local"` or `"github"` |
| `BLOBSOURCE_LOCAL_DIR` | `./configs/blobs` | Required when `TYPE=local` |
| `BLOBSOURCE_GITHUB_REPO` | — | Required when `TYPE=github`, e.g. `OpenNSW/one-trade-templates` |
| `BLOBSOURCE_GITHUB_REF` | — | Required when `TYPE=github`. Pin to a SHA in production. |
| `BLOBSOURCE_GITHUB_MANIFEST_PATH` | `manifest.json` | Repo-relative manifest path; use a subdir path for per-group manifests |
| `BLOBSOURCE_GITHUB_BASE_URL` | `https://raw.githubusercontent.com` | Override for Enterprise / mirrors / tests |
| `BLOBSOURCE_GITHUB_REFRESH_INTERVAL` | `0` (disabled) | Background manifest refresh, e.g. `5m` |

## Direct constructors

### Local source

`NewLocal` serves blobs from a local directory. It has two modes, auto-selected
by whether the directory contains a `manifest.json`:

```go
src, err := blobsource.NewLocal("/etc/oga/blobs")
```

**Flat mode** (no `manifest.json`): reads every `.json` file directly in `dir`
into memory at startup. The file basename without the `.json` extension becomes
the blob ID.

- Returns an error if `dir` is missing or contains no `.json` files.
- Subdirectories and non-`.json` files are silently skipped.

**Manifest mode** (`dir/manifest.json` present): loads blobs from the paths
named in the manifest's `byId` map (the same format the GitHub backend uses),
resolved relative to `dir`. This lets a local clone of a manifest-based repo —
whose blobs are nested under subdirectories — be served directly. Paths that
escape `dir` via `..` or absolute paths are rejected.

In both modes:

- Payload bytes are not parsed or validated — files load even if their contents
  are not valid JSON.
- `Close` is a no-op but must still be called to satisfy the interface.

### GitHub source

`NewGitHub` fetches `manifest.json` from a GitHub repository at startup
(fail-fast), then lazily fetches and caches individual blob files on first
access.

```go
src, err := blobsource.NewGitHub(context.Background(), blobsource.GitHubConfig{
    Repo:            "OpenNSW/one-trade-templates",
    Ref:             "abc1234",          // pin to a SHA in production
    RefreshInterval: 5 * time.Minute,    // 0 disables background refresh
})
```

#### `GitHubConfig` fields

| Field | Required | Default | Description |
|---|---|---|---|
| `Repo` | yes | — | `"owner/name"` e.g. `"OpenNSW/one-trade-templates"` |
| `Ref` | yes | — | Branch name or commit SHA. Pin to a SHA in production. |
| `ManifestPath` | no | `manifest.json` | Repo-relative path to the manifest file. `byId` entries are always repo-root-relative. |
| `RefreshInterval` | no | `0` (disabled) | How often to re-fetch the manifest in the background. |
| `BaseURL` | no | `https://raw.githubusercontent.com` | Override for GitHub Enterprise, mirrors, or test servers. |
| `HTTPClient` | no | 10 s-timeout client | Override for custom TLS, proxies, or test transports. |

#### How it works

1. **Manifest** — the manifest file (`manifest.json` by default, or
   `ManifestPath`) must contain a top-level `byId` object that maps blob IDs to
   **repo-root-relative** file paths:
   ```json
   { "byId": { "build-licence": "forms/build-licence.json" } }
   ```
   `byId` paths are always relative to the repository root, regardless of where
   the manifest itself lives. With `ManifestPath: "agency-configs/fcau/manifest.json"`,
   a `byId` entry `"agency-configs/fcau/task-configs/x.json"` fetches that exact
   repo-root path. The manifest is always parsed as JSON, even though blob
   payloads themselves are treated as opaque bytes.
2. **Lazy fetch** — the first `Get` call for an ID fetches the file and caches
   it.
3. **Background refresh** — when `RefreshInterval > 0`, a goroutine
   periodically re-fetches the manifest. The blob cache is fully cleared on
   each refresh so in-place file edits (same path, new bytes) are picked up.
4. **`Close`** — stops the background goroutine. Safe to call multiple times.

## `Source` contract

```
(bytes, true,  nil) — blob found
(nil,  false,  nil) — ID unknown to this source; caller should skip
(nil,  false,  err) — fetch failed; caller should warn-log and skip
```

## Running the tests

```bash
cd backend
go test ./pkg/blobsource/...
```

Tests use `net/http/httptest` — no network access required.
