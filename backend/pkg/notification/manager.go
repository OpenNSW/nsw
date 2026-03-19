package notification

import (
	"context"
	"log/slog"
	"sync"
)

// Manager is responsible for handling notification channels and dispatching messages.
type Manager struct {
	mu          sync.RWMutex
	smsChannels []SMSChannel
}

// NewManager initializes a new notification manager.
func NewManager() *Manager {
	return &Manager{
		smsChannels: make([]SMSChannel, 0),
	}
}

// RegisterSMSChannel registers a new SMS/WhatsApp provider.
func (m *Manager) RegisterSMSChannel(channel SMSChannel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.smsChannels = append(m.smsChannels, channel)
}

// SendSMS dispatches SMS/WhatsApp notifications asynchronously to all registered providers.
func (m *Manager) SendSMS(ctx context.Context, payload SMSPayload) {
	m.mu.RLock()
	channels := m.smsChannels
	m.mu.RUnlock()

	go func() {
		for _, channel := range channels {
			results := channel.Send(ctx, payload)
			m.logErrors("SMS", results)
		}
	}()
}

func (m *Manager) logErrors(cType string, results map[string]error) {
	for recipient, err := range results {
		if err != nil {
			slog.Error("failed to send notification",
				"type", cType,
				"recipient", recipient,
				"error", err)
		}
	}
}
