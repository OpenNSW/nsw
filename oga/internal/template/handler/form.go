package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/OpenNSW/nsw/oga/internal/template/model"
)

type FormHandler struct{}

func NewFormHandler() *FormHandler {
	return &FormHandler{}
}

func (h *FormHandler) Type() model.TemplateType {
	return model.TemplateTypeForm
}

func (h *FormHandler) Validate(content map[string]any) error {
	if content == nil {
		return fmt.Errorf("content is required")
	}

	schema, ok := content["schema"]
	if !ok || schema == nil {
		return fmt.Errorf("schema is required")
	}
	if _, ok := schema.(map[string]any); !ok {
		return fmt.Errorf("schema must be an object")
	}

	uiSchema, ok := content["uiSchema"]
	if !ok || uiSchema == nil {
		return fmt.Errorf("uiSchema is required")
	}
	if _, ok := uiSchema.(map[string]any); !ok {
		return fmt.Errorf("uiSchema must be an object")
	}

	return nil
}

func (h *FormHandler) Process(_ context.Context, content json.RawMessage, _ map[string]any) (any, error) {
	return content, nil
}
