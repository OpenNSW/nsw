package notification

import (
	"context"
	"log/slog"
	"sync"
)

// Manager is responsible for handling notification channels and dispatching messages.
type Manager struct {
	mu           sync.RWMutex
	emailChannel EmailChannel
	smsChannel   SMSChannel
}

// NewManager initializes a new notification manager.
func NewManager() *Manager {
	return &Manager{}
}

// RegisterEmailChannel registers a new email provider.
func (m *Manager) RegisterEmailChannel(channel EmailChannel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emailChannel = channel
}

// RegisterSMSChannel registers a new SMS/WhatsApp provider.
func (m *Manager) RegisterSMSChannel(channel SMSChannel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.smsChannel = channel
}

// SendEmail dispatches email notifications asynchronously to all registered providers.
func (m *Manager) SendEmail(ctx context.Context, payload EmailPayload) {
	m.mu.RLock()
	channel := m.emailChannel
	m.mu.RUnlock()

	if channel == nil {
		slog.WarnContext(ctx, "email channel not registered, skipping send")
		return
	}

	go func() {
		results := channel.Send(ctx, payload)
		m.logErrors(ctx, "EMAIL", results)
	}()
}

// SendSMS dispatches SMS/WhatsApp notifications asynchronously to all registered providers.
func (m *Manager) SendSMS(ctx context.Context, payload SMSPayload) {
	m.mu.RLock()
	channel := m.smsChannel
	m.mu.RUnlock()

	if channel == nil {
		slog.WarnContext(ctx, "sms channel not registered, skipping send")
		return
	}

	go func() {
		results := channel.Send(ctx, payload)
		m.logErrors(ctx, "SMS", results)
	}()
}

func (m *Manager) logErrors(ctx context.Context, cType string, results map[string]error) {
	for recipient, err := range results {
		if err != nil {
			slog.ErrorContext(ctx, "failed to send notification",
				"type", cType,
				"recipient", recipient,
				"error", err)
		}
	}
}
