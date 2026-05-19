# templatesource

Resolves JSON template blobs by ID. The package exposes a single `Source`
interface with two implementations:

| Implementation | Constructor | When to use |
|---|---|---|
| Local filesystem | `NewLocal(dir)` | Development / offline testing |
| GitHub raw content | `NewGitHub(ctx, cfg)` | Staging and production |

The package returns opaque `json.RawMessage` bytes and never inspects the JSON
shape, so it can serve forms, workflow definitions, or any other
manifest-keyed artifact. It is intentionally free of OGA-specific config or
env coupling so it can be reused by other services.

## Import

```go
import "github.com/OpenNSW/nsw/pkg/templatesource"
```

## Usage

### Local source

`NewLocal` reads every `.json` file in `dir` into memory at startup. The file
basename without the `.json` extension becomes the template ID.

```go
src, err := templatesource.NewLocal("/etc/oga/templates")
if err != nil {
    log.Fatal(err)
}
defer src.Close()

raw, ok, err := src.GetTemplate(context.Background(), "build-licence")
```

- Returns an error and refuses to start if `dir` is missing or any file
  contains invalid JSON.
- Subdirectories and non-`.json` files are silently skipped.
- `Close` is a no-op but must still be called to satisfy the interface.

### GitHub source

`NewGitHub` fetches `manifest.json` from a GitHub repository at startup
(fail-fast), then lazily fetches and caches individual template files on
first access.

```go
src, err := templatesource.NewGitHub(context.Background(), templatesource.GitHubConfig{
    Repo:            "OpenNSW/one-trade-templates",
    Ref:             "abc1234",           // pin to a SHA in production
    RefreshInterval: 5 * time.Minute,    // 0 disables background refresh
})
if err != nil {
    log.Fatal(err)
}
defer src.Close()

raw, ok, err := src.GetTemplate(context.Background(), "build-licence")
```

#### `GitHubConfig` fields

| Field | Required | Default | Description |
|---|---|---|---|
| `Repo` | yes | — | `"owner/name"` e.g. `"OpenNSW/one-trade-templates"` |
| `Ref` | yes | — | Branch name or commit SHA. Pin to a SHA in production. |
| `RefreshInterval` | no | `0` (disabled) | How often to re-fetch `manifest.json` in the background. |
| `BaseURL` | no | `https://raw.githubusercontent.com` | Override for GitHub Enterprise, mirrors, or test servers. |
| `HTTPClient` | no | 10 s-timeout client | Override for custom TLS, proxies, or test transports. |

#### How it works

1. **Manifest** — `manifest.json` must contain a top-level `byId` object that
   maps template IDs to repo-relative file paths:
   ```json
   { "byId": { "build-licence": "forms/build-licence.json" } }
   ```
2. **Lazy fetch** — the first `GetTemplate` call for an ID fetches the file
   and caches it.
3. **Background refresh** — when `RefreshInterval > 0`, a goroutine
   periodically re-fetches the manifest. Cache entries whose paths no longer
   appear in the new manifest are evicted automatically.
4. **`Close`** — stops the background goroutine. Safe to call multiple times.

## `Source` contract

```
(bytes, true,  nil) — template found
(nil,  false,  nil) — ID unknown to this source; caller should skip
(nil,  false,  err) — fetch or parse failed; caller should warn-log and skip
```

## Running the tests

```bash
cd backend
go test ./pkg/templatesource/...
```

Tests use `net/http/httptest` — no network access required.
