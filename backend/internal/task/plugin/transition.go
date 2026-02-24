package plugin

import (
	"fmt"

	"github.com/OpenNSW/nsw/pkg/jsonform"
)

// TransitionConfig drives dynamic FSM action resolution from a response field.
// It is the simpler sibling of EmitConfig: one field, one value → one action.
type TransitionConfig struct {
	Field   string            `json:"field"`             // dot-path into the request content
	Mapping map[string]string `json:"mapping"`           // field value → FSM action
	Default string            `json:"default,omitempty"` // fallback if no value matches
}

// Resolve extracts the configured field from data and returns the mapped FSM action.
func (t *TransitionConfig) Resolve(data map[string]any) (string, error) {
	val, exists := jsonform.GetValueByPath(data, t.Field)
	if !exists {
		if t.Default != "" {
			return t.Default, nil
		}
		return "", fmt.Errorf("transition field %q not found in data", t.Field)
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("transition field %q is not a string (got %T)", t.Field, val)
	}

	if action, ok := t.Mapping[str]; ok {
		return action, nil
	}

	if t.Default != "" {
		return t.Default, nil
	}

	return "", fmt.Errorf("no transition mapped for field %q value %q and no default set", t.Field, str)
}
