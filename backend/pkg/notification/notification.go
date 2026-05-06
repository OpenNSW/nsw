package notification

import (
	"context"
	"fmt"
)

type Manager struct {
	providers map[ChannelType]Provider
}

func NewManager(configPath string, providers []Provider) (*Manager, error) {
	return newManager(configPath, providers)
}

func newManager(configPath string, providers []Provider) (*Manager, error) {
	cfgMap, err := loadConfigMap(configPath)
	if err != nil {
		return nil, err
	}
	m := &Manager{providers: make(map[ChannelType]Provider)}

	for _, p := range providers {
		ch := p.Type()
		raw, ok := cfgMap[string(ch)]
		if !ok {
			continue
		}
		if err := p.Configure(raw); err != nil {
			return nil, fmt.Errorf("configure %s provider: %w", ch, err)
		}
		m.providers[ch] = p
	}
	return m, nil
}

func (m *Manager) Send(ctx context.Context, req Request) error {
	if err := req.Validate(); err != nil {
		return err
	}

	// TODO: req can contain multiple channels, so we should send to all of them
	// TODO: we should also consider retrying failed sends, and maybe even queueing them for later processing
	// For now, assume req.Channel is a single channel type and send to it directly
	p, ok := m.providers[req.Channel]
	if !ok {
		return fmt.Errorf("unsupported channel: %s", req.Channel)
	}
	return p.Send(ctx, req)
}
