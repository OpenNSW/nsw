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

// EmailPayload contains email-specific notification data.
type EmailPayload struct {
	BasePayload
	Recipients []string
	Subject    string
	Body       string // Used if TemplateID is empty
}

// SMSPayload contains phone-based notification data (SMS, WhatsApp, etc.).
type SMSPayload struct {
	BasePayload
	Recipients []string
	Body       string // Used if TemplateID is empty
}

// EmailChannel defines the interface for an email notification provider.
type EmailChannel interface {
	Send(ctx context.Context, payload EmailPayload) map[string]error
}

// SMSChannel defines the interface for a phone-based notification provider (SMS, WhatsApp).
type SMSChannel interface {
	Send(ctx context.Context, payload SMSPayload) map[string]error
}

// EmailConfig holds configuration for email channel.
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPSender   string
	TemplateRoot string // Directory containing email templates
}
