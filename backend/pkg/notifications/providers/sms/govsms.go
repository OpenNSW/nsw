package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/OpenNSW/nsw/pkg/notifications"
)

type GovSMSConfig struct {
	UserName   string
	Password   string
	SIDCode    string
	BaseURL    string
	HTTPClient *http.Client
}

type GovSMSProvider struct {
	config GovSMSConfig
}

func NewGovSMS(config GovSMSConfig) *GovSMSProvider {
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &GovSMSProvider{config: config}
}

func (p *GovSMSProvider) Type() notifications.ChannelType {
	return notifications.ChannelSMS
}

type govSMSRequestPayload struct {
	Data        string `json:"data"`
	PhoneNumber string `json:"phoneNumber"`
	SIDCode     string `json:"sIDCode"`
	UserName    string `json:"userName"`
	Password    string `json:"password"`
}

func (p *GovSMSProvider) Send(ctx context.Context, req notifications.Request) error {
	if !strings.HasPrefix(strings.ToLower(p.config.BaseURL), "https://") {
		return fmt.Errorf("insecure GovSMS BaseURL: HTTPS is required to protect credentials")
	}

	payload := govSMSRequestPayload{
		Data:        req.Body,
		PhoneNumber: req.To,
		SIDCode:     p.config.SIDCode,
		UserName:    p.config.UserName,
		Password:    p.config.Password,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal GovSMS payload: %w", err)
	}

	slog.InfoContext(ctx, "sending GovSMS", "recipient", req.To, "url", p.config.BaseURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create GovSMS request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.config.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("GovSMS request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GovSMS returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
