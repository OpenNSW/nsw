// Package templatesource resolves JSON template blobs by ID. It exposes a
// Source interface with two implementations: a local-folder reader and a
// GitHub-manifest-backed loader. The package returns opaque bytes and never
// inspects the JSON shape, so it can serve forms, workflow definitions, or
// any other manifest-keyed artifact. It is intentionally free of any
// OGA-specific config or env coupling so it can be reused by other services.
package templatesource

import (
	"context"
	"encoding/json"
)

// Source resolves templates by ID.
//
// Return contract:
//   - (bytes, true, nil)   — template found and returned.
//   - (nil,   false, nil)  — the ID is not known to this source. Callers should
//     treat this as "skip and continue without the template".
//   - (nil,   false, err)  — fetch or parse failed for an otherwise-known ID.
//     Callers should warn-log and continue without the template.
type Source interface {
	GetTemplate(ctx context.Context, id string) (json.RawMessage, bool, error)
	Close() error
}
