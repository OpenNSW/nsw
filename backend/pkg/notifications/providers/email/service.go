package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/OpenNSW/nsw/pkg/notifications"
)

// ServiceConfig holds configuration for the external email HTTP service.
type ServiceConfig struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// ServiceProvider sends email via an external HTTP email service.
type ServiceProvider struct {
	config ServiceConfig
}

func NewService(config ServiceConfig) *ServiceProvider {
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &ServiceProvider{config: config}
}

func (p *ServiceProvider) Type() notifications.ChannelType {
	return notifications.ChannelEmail
}

type serviceEmailPayload struct {
	To       string `json:"to"`
	Subject  string `json:"subject"`
	TextBody string `json:"text_body"`
	HTMLBody string `json:"html_body,omitempty"`
}

func (p *ServiceProvider) Send(ctx context.Context, req notifications.Request) error {
	payload := serviceEmailPayload{
		To:       req.To,
		Subject:  req.Subject,
		TextBody: req.Body,
		HTMLBody: req.HTMLBody,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.BaseURL+"/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create email request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.config.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.config.Token)
	}

	resp, err := p.config.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("email service request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("email service returned status %d: %s", resp.StatusCode, string(body))
	}

	slog.InfoContext(ctx, "email sent", "recipient", req.To)
	return nil
}
