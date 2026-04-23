package jsonutils

import (
	"fmt"
	"regexp"
)

var fullPlaceholderPattern = regexp.MustCompile(`^\$\{([^}]+)\}$|^\$([A-Za-z0-9_.-]+)$`)
var interpolationPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// ResolveTemplate recursively traverses a template structure and replaces string values
// with results from the lookup function. It supports maps and slices.
func ResolveTemplate(template any, lookup func(string) any) any {
	switch v := template.(type) {
	case map[string]any:
		newMap := make(map[string]any)
		for k, val := range v {
			newMap[k] = ResolveTemplate(val, lookup)
		}
		return newMap
	case []any:
		newSlice := make([]any, len(v))
		for i, val := range v {
			newSlice[i] = ResolveTemplate(val, lookup)
		}
		return newSlice
	case string:
		if resolved := lookup(v); resolved != nil {
			return resolved
		}
		return v
	default:
		return v
	}
}

// ResolveTemplateWithPlaceholders recursively traverses a template structure and resolves:
// - direct whole-string lookups like "consignment.id"
// - placeholders in values like "$name" or "${student.grade}"
// - placeholders in map keys like "grade_${student.grade}"
func ResolveTemplateWithPlaceholders(template any, lookup func(string) any) any {
	switch v := template.(type) {
	case map[string]any:
		newMap := make(map[string]any)
		for k, val := range v {
			newMap[resolveKey(k, lookup)] = ResolveTemplateWithPlaceholders(val, lookup)
		}
		return newMap
	case []any:
		newSlice := make([]any, len(v))
		for i, val := range v {
			newSlice[i] = ResolveTemplateWithPlaceholders(val, lookup)
		}
		return newSlice
	case string:
		return resolveString(v, lookup)
	default:
		return v
	}
}

func resolveKey(key string, lookup func(string) any) string {
	resolved := interpolateString(key, lookup)
	if str, ok := resolved.(string); ok {
		return str
	}
	return fmt.Sprint(resolved)
}

func resolveString(s string, lookup func(string) any) any {
	if resolved, ok := interpolateStringIfPresent(s, lookup); ok {
		return resolved
	}

	// Backwards-compatible behavior: resolve plain strings as direct lookup paths.
	if resolved := lookup(s); resolved != nil {
		return resolved
	}
	return s
}

func interpolateString(s string, lookup func(string) any) any {
	if match := fullPlaceholderPattern.FindStringSubmatchIndex(s); match != nil {
		if resolved, ok := lookupPlaceholder(s, match, lookup); ok {
			return resolved
		}
		return s
	}

	return interpolationPattern.ReplaceAllStringFunc(s, func(token string) string {
		match := interpolationPattern.FindStringSubmatchIndex(token)
		if resolved, ok := lookupPlaceholder(token, match, lookup); ok {
			return fmt.Sprint(resolved)
		}
		return token
	})
}

func interpolateStringIfPresent(s string, lookup func(string) any) (any, bool) {
	if fullPlaceholderPattern.MatchString(s) || interpolationPattern.MatchString(s) {
		return interpolateString(s, lookup), true
	}
	return nil, false
}

func lookupPlaceholder(token string, match []int, lookup func(string) any) (any, bool) {
	var key string
	switch {
	case len(match) >= 4 && match[2] != -1:
		key = token[match[2]:match[3]]
	case len(match) >= 6 && match[4] != -1:
		key = token[match[4]:match[5]]
	default:
		return nil, false
	}

	resolved := lookup(key)
	if resolved == nil {
		return nil, false
	}
	return resolved, true
}
