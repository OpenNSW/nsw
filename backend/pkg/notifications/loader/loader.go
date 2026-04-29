package loader

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/OpenNSW/nsw/pkg/notifications"
	"github.com/OpenNSW/nsw/pkg/notifications/providers/email"
	"github.com/OpenNSW/nsw/pkg/notifications/providers/sms"
)

var envVarRe = regexp.MustCompile(`\$\{([^}]+)\}`)

type ChannelConfig struct {
	Provider          string          `json:"provider"`
	EmailTemplateRoot string          `json:"emailTemplateRoot,omitempty"`
	SMSTemplateRoot   string          `json:"smsTemplateRoot,omitempty"`
	Options           json.RawMessage `json:"options"`
}

type FileConfig struct {
	Channels []ChannelConfig `json:"channels"`
}

// expandEnv replaces ${VAR} placeholders with JSON-string-safe values.
// Each value is JSON-encoded and the surrounding quotes stripped, so special
// characters (quotes, backslashes, newlines, PEM blocks) cannot break the
// surrounding JSON structure.
func expandEnv(data []byte) ([]byte, error) {
	var missing []string
	result := envVarRe.ReplaceAllFunc(data, func(match []byte) []byte {
		name := string(match[2 : len(match)-1])
		val, ok := os.LookupEnv(name)
		if !ok {
			missing = append(missing, name)
			return match
		}
		// json.Marshal on a string escapes ", \, and control chars including \n.
		// Strip the surrounding quotes — the placeholder is already inside a JSON string.
		encoded, _ := json.Marshal(val)
		return encoded[1 : len(encoded)-1]
	})
	if len(missing) > 0 {
		return nil, fmt.Errorf("unset environment variables in notifications config: %v", missing)
	}
	return result, nil
}

// LoadFromFile reads path, expands ${VAR} references, and builds a Manager.
func LoadFromFile(path string) (*notifications.Manager, error) {
	if path == "" {
		return nil, fmt.Errorf("NOTIFICATIONS_CONFIG_PATH is required")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read notifications config %q: %w", path, err)
	}

	expanded, err := expandEnv(raw)
	if err != nil {
		return nil, err
	}

	var fc FileConfig
	if err := json.Unmarshal(expanded, &fc); err != nil {
		return nil, fmt.Errorf("parse notifications config: %w", err)
	}

	var providers []notifications.Provider
	cfg := notifications.Config{}

	for i, ch := range fc.Channels {
		switch ch.Provider {
		case "email_service":
			var ecfg email.Config
			if err := json.Unmarshal(ch.Options, &ecfg); err != nil {
				return nil, fmt.Errorf("channel[%d] email_service options: %w", i, err)
			}
			if ecfg.BaseURL == "" {
				return nil, fmt.Errorf("channel[%d] email_service options: baseURL is required", i)
			}
			providers = append(providers, email.New(ecfg))
			if ch.EmailTemplateRoot != "" {
				cfg.EmailTemplateRoot = ch.EmailTemplateRoot
			}
		case "govsms":
			var scfg sms.GovSMSConfig
			if err := json.Unmarshal(ch.Options, &scfg); err != nil {
				return nil, fmt.Errorf("channel[%d] govsms options: %w", i, err)
			}
			if scfg.UserName == "" {
				return nil, fmt.Errorf("channel[%d] govsms options: userName is required", i)
			}
			providers = append(providers, sms.NewGovSMS(scfg))
			if ch.SMSTemplateRoot != "" {
				cfg.SMSTemplateRoot = ch.SMSTemplateRoot
			}
		default:
			return nil, fmt.Errorf("channel[%d]: unknown provider %q", i, ch.Provider)
		}
	}

	return notifications.New(cfg, providers...), nil
}
