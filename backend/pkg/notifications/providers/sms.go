package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/OpenNSW/nsw/pkg/notifications"
	"github.com/OpenNSW/nsw/pkg/remote"
)

type smsConfig struct {
	ServiceID string `json:"service_id"`
	SIDCode   string `json:"sid_code"`
	UserName  string `json:"username"`
	Password  string `json:"password"`
}

type govSMSRequest struct {
	Data        string `json:"data"`
	PhoneNumber string `json:"phoneNumber"`
	SIDCode     string `json:"sIDCode"`
	UserName    string `json:"userName"`
	Password    string `json:"password"`
}

// SMSProvider sends SMS via the GovSMS service.
type SMSProvider struct {
	cfg     smsConfig
	manager *remote.Manager
}

// NewSMSProvider returns an SMSProvider backed by the given remote manager.
func NewSMSProvider(m *remote.Manager) *SMSProvider {
	return &SMSProvider{manager: m}
}

func (s *SMSProvider) Type() notifications.ChannelType { return notifications.ChannelSMS }

func (s *SMSProvider) Configure(raw json.RawMessage) error {
	if err := json.Unmarshal(raw, &s.cfg); err != nil {
		return fmt.Errorf("unmarshal sms config: %w", err)
	}
	if s.cfg.ServiceID == "" {
		return errors.New("sms provider: service_id is required")
	}
	if s.cfg.SIDCode == "" {
		return errors.New("sms provider: sid_code is required")
	}
	if s.cfg.UserName == "" {
		return errors.New("sms provider: username is required")
	}
	if s.cfg.Password == "" {
		return errors.New("sms provider: password is required")
	}
	return nil
}

func (s *SMSProvider) Send(ctx context.Context, req notifications.Request) error {
	if err := s.manager.Call(ctx, s.cfg.ServiceID, remote.Request{
		Method: http.MethodPost,
		Path:   "/govsms/V1/prod/send",
		Body: govSMSRequest{
			Data:        req.Body,
			PhoneNumber: req.To,
			SIDCode:     s.cfg.SIDCode,
			UserName:    s.cfg.UserName,
			Password:    s.cfg.Password,
		},
	}, nil); err != nil {
		return fmt.Errorf("govsms send: %w", err)
	}
	return nil
}
