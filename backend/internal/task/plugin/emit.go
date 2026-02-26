package plugin

import "github.com/OpenNSW/nsw/pkg/jsonform"

// EmissionConfig holds the rules evaluated when a plugin action completes.
// Every rule whose conditions all match contributes its outcome to the result.
type EmissionConfig struct {
	Rules []EmissionRule `json:"rules"`
}

// EmissionRule emits Outcome when every condition in Conditions matches.
// Multiple conditions within one rule are AND-ed together.
// OR semantics are expressed by adding separate rules.
type EmissionRule struct {
	Outcome    string           `json:"outcome"`    // e.g. "npqs:phytosanitary:manual_review_required"
	Conditions []FieldCondition `json:"conditions"` // all must match
}

// FieldCondition checks that the value at Field (dot-path) equals Value.
type FieldCondition struct {
	Field string `json:"field"` // dot-path into the response data
	Value string `json:"value"` // exact string value to match
}

// Evaluate walks the rules in order and returns the outcome of the first rule
// whose conditions all pass against data. Returns nil if no rule matched.
// Rules are expected to be non-overlapping; the first match wins.
func (e *EmissionConfig) Evaluate(data map[string]any) *string {
	for _, rule := range e.Rules {
		if rule.matches(data) {
			return &rule.Outcome
		}
	}
	return nil
}

// matches returns true when every condition in the rule is satisfied.
func (r *EmissionRule) matches(data map[string]any) bool {
	for _, c := range r.Conditions {
		val, exists := jsonform.GetValueByPath(data, c.Field)
		if !exists {
			return false
		}
		str, ok := val.(string)
		if !ok || str != c.Value {
			return false
		}
	}
	return true
}
