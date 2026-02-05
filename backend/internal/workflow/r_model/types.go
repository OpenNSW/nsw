package r_model

import "github.com/google/uuid"

// UUIDArray is a type alias for UUID slices that can be serialized across different databases
type UUIDArray []uuid.UUID

// MarshalJSON implements json.Marshaler for UUIDArray
func (a UUIDArray) MarshalJSON() ([]byte, error) {
	if a == nil {
		return []byte("[]"), nil
	}
	if len(a) == 0 {
		return []byte("[]"), nil
	}

	ids := make([]string, len(a))
	for i, id := range a {
		ids[i] = `"` + id.String() + `"`
	}

	result := "[" + ids[0]
	for i := 1; i < len(ids); i++ {
		result += "," + ids[i]
	}
	result += "]"

	return []byte(result), nil
}

// UnmarshalJSON implements json.Unmarshaler for UUIDArray
func (a *UUIDArray) UnmarshalJSON(data []byte) error {
	// Handle empty array
	if string(data) == "[]" || string(data) == "null" {
		*a = []uuid.UUID{}
		return nil
	}

	// Parse the JSON array manually
	str := string(data)
	if len(str) < 2 || str[0] != '[' || str[len(str)-1] != ']' {
		*a = []uuid.UUID{}
		return nil
	}

	// Remove brackets and parse UUIDs
	content := str[1 : len(str)-1]
	if len(content) == 0 {
		*a = []uuid.UUID{}
		return nil
	}

	// Simple split by comma (works for UUID strings)
	var ids []uuid.UUID
	current := ""
	inQuote := false

	for _, ch := range content {
		if ch == '"' {
			inQuote = !inQuote
		} else if ch == ',' && !inQuote {
			if len(current) > 0 {
				if id, err := uuid.Parse(current); err == nil {
					ids = append(ids, id)
				}
				current = ""
			}
		} else if inQuote {
			current += string(ch)
		}
	}

	// Add last UUID
	if len(current) > 0 {
		if id, err := uuid.Parse(current); err == nil {
			ids = append(ids, id)
		}
	}

	*a = ids
	return nil
}
