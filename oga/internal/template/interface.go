package template

import (
	"context"
	"encoding/json"
)

type TemplateHandler interface {
	Type() TemplateType
	Validate(content map[string]any) error
	Process(ctx context.Context, content json.RawMessage, data map[string]any) (any, error)
}
