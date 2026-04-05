package handler

import (
	"context"
	"encoding/json"

	"github.com/OpenNSW/nsw/oga/internal/template/model"
)

type TemplateHandler interface {
	Type() model.TemplateType
	Validate(content map[string]any) error
	Process(ctx context.Context, content json.RawMessage, data map[string]any) (any, error)
}
