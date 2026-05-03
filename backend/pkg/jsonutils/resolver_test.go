package jsonutils

import (
	"reflect"
	"testing"
)

func TestResolveTemplate(t *testing.T) {
	// Mock lookup data
	store := map[string]any{
		"consignment.id": "C-123",
		"exporter.name":  "Organic Farms",
		"items.0.code":   "HS-001",
		"items.1.code":   "HS-002",
		"meta.priority":  10,
		"meta.active":    true,
	}

	lookup := func(key string) any {
		return store[key]
	}

	tests := []struct {
		name     string
		template any
		want     any
	}{
		{
			name:     "Simple string replacement",
			template: "consignment.id",
			want:     "C-123",
		},
		{
			name: "Nested map resolution",
			template: map[string]any{
				"header": map[string]any{
					"exporter": "exporter.name",
				},
				"id": "consignment.id",
			},
			want: map[string]any{
				"header": map[string]any{
					"exporter": "Organic Farms",
				},
				"id": "C-123",
			},
		},
		{
			name:     "Array of strings",
			template: []any{"consignment.id", "exporter.name"},
			want:     []any{"C-123", "Organic Farms"},
		},
		{
			name: "Complex nested structure (The OGA example)",
			template: map[string]any{
				"header": map[string]any{
					"priority": "meta.priority",
				},
				"body": map[string]any{
					"items": []any{
						map[string]any{"hs_code": "items.0.code"},
						map[string]any{"hs_code": "items.1.code"},
					},
				},
				"status": "meta.active",
			},
			want: map[string]any{
				"header": map[string]any{
					"priority": 10,
				},
				"body": map[string]any{
					"items": []any{
						map[string]any{"hs_code": "HS-001"},
						map[string]any{"hs_code": "HS-002"},
					},
				},
				"status": true,
			},
		},
		{
			name: "Preserve non-matching strings and types",
			template: map[string]any{
				"literal": "I am a literal string",
				"number":  42,
				"missing": "not.in.store",
			},
			want: map[string]any{
				"literal": "I am a literal string",
				"number":  42,
				"missing": "not.in.store",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveTemplate(tt.template, lookup)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveTemplateWithPlaceholders(t *testing.T) {
	store := map[string]any{
		"name":          "Alice",
		"student.grade": 7,
		"sections":      []any{"A", "B"},
		"dynamic.key":   "student",
		"plain.path":    "resolved",
	}

	lookup := func(key string) any {
		return store[key]
	}

	tests := []struct {
		name     string
		template any
		want     any
	}{
		{
			name:     "Direct lookup stays supported",
			template: "plain.path",
			want:     "resolved",
		},
		{
			name: "Placeholder interpolation in nested values",
			template: map[string]any{
				"name": "$name",
				"school": map[string]any{
					"class":    "Grade ${student.grade}",
					"sections": "$sections",
				},
			},
			want: map[string]any{
				"name": "Alice",
				"school": map[string]any{
					"class":    "Grade 7",
					"sections": []any{"A", "B"},
				},
			},
		},
		{
			name: "Bare dollar inside a larger string is left literal",
			template: map[string]any{
				"text": "Hello $name",
			},
			want: map[string]any{
				"text": "Hello $name",
			},
		},
		{
			name: "Placeholder interpolation in keys",
			template: map[string]any{
				"${dynamic.key}": map[string]any{
					"grade_${student.grade}": "$name",
				},
			},
			want: map[string]any{
				"student": map[string]any{
					"grade_7": "Alice",
				},
			},
		},
		{
			name:     "Unknown placeholders are preserved",
			template: "Grade ${unknown}",
			want:     "Grade ${unknown}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveTemplateWithPlaceholders(tt.template, lookup)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveTemplateWithPlaceholders() = %v, want %v", got, tt.want)
			}
		})
	}
}
