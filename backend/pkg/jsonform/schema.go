package jsonform

type GlobalContext struct {
	ReadFrom *string `json:"readFrom,omitempty"`
	WriteTo  *string `json:"writeTo,omitempty"`
}
type JSONSchema struct {
	Type       string                `json:"type,omitempty"`
	Properties map[string]JSONSchema `json:"properties,omitempty"`
	Items      *JSONSchema           `json:"items,omitempty"`
	Required   []string              `json:"required,omitempty"`

	Minimum        *float64       `json:"minimum,omitempty"`
	MinLength      *int           `json:"minLength,omitempty"`
	XGlobalContext *GlobalContext `json:"x-globalContext,omitempty"`
}
