package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"

	"github.com/OpenNSW/nsw/pkg/notification/internal/core"
	"github.com/OpenNSW/nsw/pkg/notification/providers/email"
	"github.com/OpenNSW/nsw/pkg/notification/providers/sms"
)

type Request = core.Request
type ChannelType = core.ChannelType

const (
	ChannelSMS   = core.ChannelSMS
	ChannelEmail = core.ChannelEmail
)

type Manager struct {
	providers map[core.ChannelType]core.Provider
}

func NewManager(configPath string) (*Manager, error) {
	return newManager(configPath, nil, nil)
}

func newManager(configPath string, providers []core.Provider, httpClient *http.Client) (*Manager, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if providers == nil {
		providers = defaultProviders(httpClient)
	}

	cfgMap, err := loadConfigMap(configPath)
	if err != nil {
		return nil, err
	}

	m := &Manager{providers: make(map[core.ChannelType]core.Provider, len(providers))}
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
	p, ok := m.providers[req.Channel]
	if !ok {
		return fmt.Errorf("unsupported channel: %s", req.Channel)
	}
	return p.Send(ctx, req)
}

func defaultProviders(client *http.Client) []core.Provider {
	return []core.Provider{
		email.NewProvider(client),
		sms.NewProvider(client),
	}
}

var envVarRe = regexp.MustCompile(`\$\{([^}]+)\}`)

func expandEnv(data []byte) ([]byte, error) {
	var missing []string
	result := envVarRe.ReplaceAllFunc(data, func(match []byte) []byte {
		name := string(match[2 : len(match)-1])
		val, ok := os.LookupEnv(name)
		if !ok {
			missing = append(missing, name)
			return match
		}
		encoded, _ := json.Marshal(val)
		return encoded[1 : len(encoded)-1]
	})
	if len(missing) > 0 {
		return nil, fmt.Errorf("unset environment variables in notification config: %v", missing)
	}
	return result, nil
}

func loadConfigMap(path string) (map[string]json.RawMessage, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read notification config %q: %w", path, err)
	}
	expanded, err := expandEnv(raw)
	if err != nil {
		return nil, err
	}
	var cfgMap map[string]json.RawMessage
	if err := json.Unmarshal(expanded, &cfgMap); err != nil {
		return nil, fmt.Errorf("parse notification config: %w", err)
	}
	return cfgMap, nil
}
