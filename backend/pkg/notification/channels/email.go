package channels

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	texttemplate "text/template"
	"time"

	"github.com/OpenNSW/nsw/pkg/notification"
	"github.com/go-mail/mail/v2"
)

// cachedTemplate stores both text and HTML versions of a template.
type cachedTemplate struct {
	text *texttemplate.Template
	html *template.Template
}

// EmailChannel implements notification.EmailChannel using SMTP.
type EmailChannel struct {
	config notification.EmailConfig
	cache  sync.Map // map[string]*cachedTemplate
}

// NewEmailChannel creates a new email channel with the given configuration.
func NewEmailChannel(config notification.EmailConfig) notification.EmailChannel {
	return &EmailChannel{
		config: config,
		cache:  sync.Map{},
	}
}

// Send sends emails to the specified recipients.
// It returns a map of recipient to error for any failures.
func (e *EmailChannel) Send(ctx context.Context, payload notification.EmailPayload) map[string]error {
	results := make(map[string]error)

	for _, recipient := range payload.Recipients {
		if err := e.sendToRecipient(ctx, payload, recipient); err != nil {
			results[recipient] = err
		}
	}

	return results
}

func (e *EmailChannel) sendToRecipient(ctx context.Context, payload notification.EmailPayload, recipient string) error {
	var subject, plainBody, htmlBody string
	var err error

	if payload.TemplateID != "" {
		subject, plainBody, htmlBody, err = e.renderTemplate(payload.TemplateID, payload.TemplateData)
		if err != nil {
			return fmt.Errorf("failed to render template %s: %w", payload.TemplateID, err)
		}
	} else {
		subject = payload.Subject
		plainBody = payload.Body
		// For plain text emails without template, don't add HTML alternative
		htmlBody = ""
	}

	msg := mail.NewMessage()
	msg.SetHeader("To", recipient)
	msg.SetHeader("From", e.config.SMTPSender)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/plain", plainBody)
	if htmlBody != "" {
		msg.AddAlternative("text/html", htmlBody)
	}

	// Create a new dialer for this send to avoid race conditions
	dialer := mail.NewDialer(e.config.SMTPHost, e.config.SMTPPort, e.config.SMTPUsername, e.config.SMTPPassword)
	dialer.Timeout = 5 * time.Second
	// Require TLS to prevent sending OTP codes and credentials in plaintext
	dialer.StartTLSPolicy = mail.MandatoryStartTLS

	// Retry logic
	const maxRetries = 3
	const retryDelay = 500 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		err = dialer.DialAndSend(msg)
		if err == nil {
			slog.InfoContext(ctx, "email sent successfully", "recipient", recipient)
			return nil
		}
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	return fmt.Errorf("failed to send email after %d retries: %w", maxRetries, err)
}

func (e *EmailChannel) renderTemplate(templateID string, data map[string]interface{}) (subject, plainBody, htmlBody string, err error) {
	var ct *cachedTemplate

	if val, ok := e.cache.Load(templateID); ok {
		ct = val.(*cachedTemplate)
	} else {
		templatePath := filepath.Join(e.config.TemplateRoot, templateID+".tmpl")
		templateContent, err := os.ReadFile(templatePath)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to read template file %s: %w", templatePath, err)
		}

		// Parse with text/template for subject and plain body (no HTML escaping)
		textTmpl, err := texttemplate.New("email").Parse(string(templateContent))
		if err != nil {
			return "", "", "", fmt.Errorf("failed to parse text template: %w", err)
		}

		// Parse with html/template for HTML body (with HTML escaping)
		htmlTmpl, err := template.New("email").Parse(string(templateContent))
		if err != nil {
			return "", "", "", fmt.Errorf("failed to parse HTML template: %w", err)
		}

		ct = &cachedTemplate{
			text: textTmpl,
			html: htmlTmpl,
		}
		e.cache.Store(templateID, ct)
	}

	// Render subject with text template
	subjectBuf := &bytes.Buffer{}
	if err := ct.text.ExecuteTemplate(subjectBuf, "subject", data); err != nil {
		return "", "", "", fmt.Errorf("failed to render subject: %w", err)
	}
	subject = subjectBuf.String()

	// Render plain body with text template
	plainBuf := &bytes.Buffer{}
	if err := ct.text.ExecuteTemplate(plainBuf, "plainBody", data); err != nil {
		return "", "", "", fmt.Errorf("failed to render plainBody: %w", err)
	}
	plainBody = plainBuf.String()

	// Render HTML body with HTML template
	htmlBuf := &bytes.Buffer{}
	if err := ct.html.ExecuteTemplate(htmlBuf, "htmlBody", data); err != nil {
		return "", "", "", fmt.Errorf("failed to render htmlBody: %w", err)
	}
	htmlBody = htmlBuf.String()

	return subject, plainBody, htmlBody, nil
}
