package jsonform

import (
	"regexp"
	"strconv"
	"strings"
)

var arrayRegex = regexp.MustCompile(`^([a-zA-Z0-9_-]+)\[(\d+)]$`)

// GetValueByPath retrieves a value from formData using dot notation.
// Returns (value, true) if found
// Returns (nil, false) if path does not exist
func GetValueByPath(formData map[string]any, path string) (any, bool) {

	if path == "" {
		return formData, true
	}

	segments := strings.Split(path, ".")
	var current any = formData

	for _, segment := range segments {

		switch typed := current.(type) {

		case map[string]any:

			// Check for array syntax: items[0]
			if matches := arrayRegex.FindStringSubmatch(segment); len(matches) == 3 {

				field := matches[1]
				indexStr := matches[2]

				val, ok := typed[field]
				if !ok {
					return nil, false
				}

				arr, ok := val.([]any)
				if !ok {
					return nil, false
				}

				index, err := strconv.Atoi(indexStr)
				if err != nil || index < 0 || index >= len(arr) {
					return nil, false
				}

				current = arr[index]
				continue
			}

			val, ok := typed[segment]
			if !ok {
				return nil, false
			}

			current = val

		default:
			return nil, false
		}
	}

	return current, true
}

// SetValueByPath sets a value in formData using dot notation path.
// Creates nested maps as needed.
func SetValueByPath(formData map[string]any, path string, value any) {
	if path == "" {
		return
	}

	segments := strings.Split(path, ".")
	current := formData

	// Navigate to the parent of the target field, creating maps as needed
	for i := 0; i < len(segments)-1; i++ {
		segment := segments[i]

		// Check if the current segment exists
		if _, exists := current[segment]; !exists {
			// Create a new nested map
			current[segment] = make(map[string]any)
		}

		// Move to the next level
		if nestedMap, ok := current[segment].(map[string]any); ok {
			current = nestedMap
		} else {
			// If it's not a map, we can't traverse further
			return
		}
	}

	// Set the value at the final segment
	current[segments[len(segments)-1]] = value
}
