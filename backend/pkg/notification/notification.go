package notification

import (
	"context"
)

// BasePayload contains shared template and metadata information.
type BasePayload struct {
	TemplateID   string                 // ID of the template to use
	TemplateData map[string]interface{} // Data to inject into the template
	Metadata     map[string]string      // Additional context
}

// SMSPayload contains phone-based notification data (SMS, WhatsApp, etc.).
type SMSPayload struct {
	BasePayload
	Recipients []string
	Body       string // Used if TemplateID is empty
}

// SMSChannel defines the interface for a phone-based notification provider (SMS, WhatsApp).
type SMSChannel interface {
	Send(ctx context.Context, payload SMSPayload) map[string]error
}
