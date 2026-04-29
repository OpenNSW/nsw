package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type ChannelType string

const (
	ChannelSMS   ChannelType = "sms"
	ChannelEmail ChannelType = "email"
)

type Request struct {
	Channel  ChannelType
	To       string
	Subject  string // email only, ignored by SMS
	Body     string
	HTMLBody string // email only, optional
}

type EmailRequest struct {
	To           string
	Subject      string
	Body         string         // plain text
	HTMLBody     string         // optional
	TemplateID   string         // if set, renders subject/Body/HTMLBody from template
	TemplateData map[string]any // data passed to template
}

type SMSRequest struct {
	To           string
	Body         string         // used when TemplateID is empty
	TemplateID   string         // if set, renders Body from template
	TemplateData map[string]any // data passed to template
}

// Provider is the contract every notification channel must satisfy.
// Adding a new channel = implement this interface, then register it.
type Provider interface {
	Send(ctx context.Context, req Request) error
	Type() ChannelType
}

// Config holds optional Manager configuration.
type Config struct {
	EmailTemplateRoot string
	SMSTemplateRoot   string
}

type Manager struct {
	providers map[ChannelType]Provider
	templates *templateCache
	wg        sync.WaitGroup
}

func New(cfg Config, providers ...Provider) *Manager {
	m := &Manager{
		providers: make(map[ChannelType]Provider),
		templates: newTemplateCache(cfg.EmailTemplateRoot, cfg.SMSTemplateRoot),
	}
	for _, p := range providers {
		m.providers[p.Type()] = p
	}
	return m
}

func (m *Manager) Send(ctx context.Context, req Request) error {
	p, ok := m.providers[req.Channel]
	if !ok {
		return fmt.Errorf("no provider registered for channel: %s", req.Channel)
	}
	return p.Send(ctx, req)
}

// SendEmail renders the template (if set) then dispatches delivery in a
// goroutine so callers are not blocked on the external HTTP call.
// Template errors are returned immediately; provider errors are logged.
func (m *Manager) SendEmail(ctx context.Context, req EmailRequest) error {
	subject, body, htmlBody := req.Subject, req.Body, req.HTMLBody
	if req.TemplateID != "" {
		var err error
		subject, body, htmlBody, err = m.templates.renderEmail(req.TemplateID, req.TemplateData)
		if err != nil {
			return fmt.Errorf("render email template %q: %w", req.TemplateID, err)
		}
	}
	r := Request{
		Channel:  ChannelEmail,
		To:       req.To,
		Subject:  subject,
		Body:     body,
		HTMLBody: htmlBody,
	}
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		detachedCtx := context.WithoutCancel(ctx)
		if err := m.Send(detachedCtx, r); err != nil {
			slog.ErrorContext(detachedCtx, "email send failed", "recipient", req.To, "error", err)
		}
	}()
	return nil
}

// SendSMS renders the template (if set) then dispatches delivery in a
// goroutine so callers are not blocked on the external HTTP call.
// Template errors are returned immediately; provider errors are logged.
func (m *Manager) SendSMS(ctx context.Context, req SMSRequest) error {
	body := req.Body
	if req.TemplateID != "" {
		var err error
		body, err = m.templates.renderSMS(req.TemplateID, req.TemplateData)
		if err != nil {
			return fmt.Errorf("render SMS template %q: %w", req.TemplateID, err)
		}
	}
	r := Request{
		Channel: ChannelSMS,
		To:      req.To,
		Body:    body,
	}
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		detachedCtx := context.WithoutCancel(ctx)
		if err := m.Send(detachedCtx, r); err != nil {
			slog.ErrorContext(detachedCtx, "SMS send failed", "recipient", req.To, "error", err)
		}
	}()
	return nil
}

// Wait blocks until all background send goroutines have completed.
// Call during graceful shutdown before exiting.
func (m *Manager) Wait() {
	m.wg.Wait()
}
